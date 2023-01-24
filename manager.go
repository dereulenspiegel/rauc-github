package raucgithub

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/dereulenspiegel/raucgithub/repository"
	"github.com/holoplot/go-rauc/rauc"
	"github.com/sirupsen/logrus"
)

var (
	ErrNoSuitableUpdate = errors.New("no suitable update found")
)

var compatibilityRegex = regexp.MustCompile(`^([a-zA-Z0-9\-\.]+)_.*`)

func ExtractCompatibility(assetName string) string {
	if compatibilityRegex.MatchString(assetName) {
		submatches := compatibilityRegex.FindAllStringSubmatch(assetName, -1)
		if len(submatches) > 0 && len(submatches[0]) > 1 {
			return submatches[0][1]
		}
	}
	return ""
}

type InstallCallback func(bool, error)

func OSVersion() (string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed to read /etc/os-release: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VERSION_ID=") {
			versionString := strings.TrimPrefix(line, "VERSION_ID=")
			return versionString, nil
		}
	}
	return "", errors.New("no version information found")
}

type raucDBUSClient interface {
	GetBootSlot() (string, error)
	GetSlotStatus() (status []rauc.SlotStatus, err error)
	GetCompatible() (string, error)
	InstallBundle(filename string, options rauc.InstallBundleOptions) error
	GetProgress() (percentage int32, message string, nestingDepth int32, err error)
	GetOperation() (string, error)
}

type UpdateManagerOption func(*UpdateManager) *UpdateManager

func WithRaucClient(client raucDBUSClient) UpdateManagerOption {
	return func(u *UpdateManager) *UpdateManager {
		u.rauc = client
		return u
	}
}

type UpdateManager struct {
	rauc                 raucDBUSClient
	repo                 repository.Repository
	logger               logrus.FieldLogger
	extractCompatibility func(string) string

	nextUpdate *repository.Update
}

func NewUpdateManager(repo repository.Repository, options ...UpdateManagerOption) (*UpdateManager, error) {

	u := &UpdateManager{
		repo: repo,
	}

	for _, opt := range options {
		u = opt(u)
	}
	if u.rauc == nil {
		raucClient, err := rauc.InstallerNew()

		if err != nil {
			return nil, fmt.Errorf("failed to instantiate rauc client: %w", err)
		}
		u.rauc = raucClient
	}
	if u.logger == nil {
		u.logger = logrus.WithField("component", "UpdateManager")
	}
	if u.extractCompatibility == nil {
		u.extractCompatibility = ExtractCompatibility
	}

	return u, nil
}

func (u *UpdateManager) compatibleBundle(update *repository.Update) (compatBundle *repository.BundleLink, err error) {
	compatibleString, err := u.rauc.GetCompatible()
	if err != nil {
		return nil, fmt.Errorf("failed to determine compatible string from rauc: %w", err)
	}
	for _, bundle := range update.Bundles {
		if bundle.Compatibility == compatibleString {
			return bundle, nil
		}
	}
	return nil, ErrNoSuitableUpdate
}

func (u *UpdateManager) getOSVersionFromRauc() (string, error) {
	bootSlotName, err := u.rauc.GetBootSlot()
	if err != nil {
		return "", fmt.Errorf("failed to get current boot slot from rauc")
	}
	slots, err := u.rauc.GetSlotStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get slot status from rauc")
	}
	var bootSlot rauc.SlotStatus
	bootSlotFound := false
	for _, slot := range slots {
		if slot.SlotName == bootSlotName {
			bootSlotFound = true
			bootSlot = slot
			break
		}
	}
	if !bootSlotFound {
		return "", fmt.Errorf("failed to identify current bootslot")
	}
	if variant, exists := bootSlot.Status["bundle.version"]; exists {
		versionString := variant.String()
		versionString = strings.Trim(versionString, "\"")
		return versionString, nil
	}
	return "", errors.New("failed to determine current OS version from rauc")
}

func (u *UpdateManager) CheckForUpdate(ctx context.Context) (*repository.Update, error) {
	compatible, err := u.rauc.GetCompatible()
	if err != nil {
		return nil, fmt.Errorf("failed to query compatible string from rauc: %w", err)
	}
	logger := u.logger.WithField("compatible", compatible)
	logger.Info("Checking for update")

	versionString, err := u.getOSVersionFromRauc()
	if err != nil {
		logger.WithError(err).Debug("failed to determine OS version from rauc, maybe this is a fresh install")
		// Maybe /etc/os-release has this information
		versionString, err = OSVersion()
		if err != nil {
			logger.WithError(err).Error("failed to determine OS version from /etc/os-release")
			return nil, fmt.Errorf("failed to determine current os version: %w", err)
		}
	}
	logger = logger.WithField("currentOSVersion", versionString)

	version, err := semver.NewVersion(versionString)
	if err != nil {
		return nil,
			fmt.Errorf("current installed version (%s) is not a semver version and can't be compared to other semver versions: %w", versionString, err)
	}
	possibleUpdates, err := u.repo.Updates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load possible updates from repository: %w", err)
	}
	// Sort updates, so we don't always accidentally select the latest update, as this might lack necessary migrations
	sort.SliceStable(possibleUpdates, func(i, j int) bool {
		return possibleUpdates[i].Version.LessThan(*possibleUpdates[j].Version)
	})

	for _, update := range possibleUpdates {
		if version.LessThan(*update.Version) {
			logger = logger.WithFields(logrus.Fields{
				"updateVersion": update.Version.String(),
				"updateName":    update.Name,
			})
			var compatibleBundle *repository.BundleLink
			// Identified possible update candidate
			for _, bundle := range update.Bundles {
				if bundle.AssetName == "" {
					_, bundle.AssetName = path.Split(bundle.URL)
				}
				bundle.Compatibility = u.extractCompatibility(bundle.AssetName)
				if bundle.Compatibility == compatible {
					compatibleBundle = bundle
					logger.WithField("bundleURL", bundle.URL).Info("identified possible next update")
				}
			}
			if compatibleBundle != nil {
				u.nextUpdate = &update
				return &update, nil
			}
			logger.Info("possible update has no compatible update bundles")
		}
	}
	return nil, ErrNoSuitableUpdate

}

func (u *UpdateManager) InstallNextUpdate(ctx context.Context) (err error) {
	if u.nextUpdate == nil {
		u.nextUpdate, err = u.CheckForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("failed to determine next suitable update: %w", err)
		}
	}
	return u.InstallUpdate(ctx, u.nextUpdate)
}

func (u *UpdateManager) InstallUpdate(ctx context.Context, update *repository.Update) (err error) {
	bundle, err := u.compatibleBundle(update)
	if err != nil {
		return fmt.Errorf("failed to identify compatible update bundle: %w", err)
	}
	logger := u.logger.WithFields(logrus.Fields{
		"updateVersion": update.Version.String(),
		"updateName":    update.Name,
		"bundleURL":     bundle.URL,
	})
	logger.Info("Starting update")
	err = u.rauc.InstallBundle(bundle.URL, rauc.InstallBundleOptions{IgnoreIncompatible: false})
	if err != nil {
		logger.WithError(err).Error("failed to install bundle")
		return fmt.Errorf("failed to install bundle: %w", err)
	}
	return nil
}

func (u *UpdateManager) InstallNextUpdateAsync(ctx context.Context, callback InstallCallback) chan int32 {
	var err error
	if u.nextUpdate != nil {
		u.nextUpdate, err = u.CheckForUpdate(ctx)
		if err != nil {
			callback(false, fmt.Errorf("failed to determine stuitable next update: %w", err))
		}
	}
	return u.InstallUpdateAsync(ctx, u.nextUpdate, callback)
}

func (u *UpdateManager) InstallUpdateAsync(ctx context.Context, update *repository.Update, callback InstallCallback) chan int32 {
	outputChan := make(chan int32, 1000)
	doneChan := make(chan bool)
	go func(callback InstallCallback, outputChan chan int32, doneChan chan bool) {
		err := u.InstallUpdate(ctx, update)
		defer func() {
			doneChan <- true
		}()
		if err != nil {
			callback(false, err)
		} else {
			callback(true, nil)
		}
	}(callback, outputChan, doneChan)
	go func(outputChan chan int32, doneChan chan bool) {
		defer func() {
			close(doneChan)
			close(outputChan)
		}()
		lastPercentage := 0
		for {
			select {
			case _, done := <-doneChan:
				if done {
					return
				}
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Millisecond * 100)
				percentage, _, _, err := u.rauc.GetProgress()
				if err != nil {
					u.logger.WithError(err).Error("failed to get progress on installation: %w", err)
					continue
				}

				// Do a non blocking write as the client might not read from the output channel
				if percentage != int32(lastPercentage) {
					select {
					case outputChan <- percentage:
					default:
					}
				}

			}
		}
	}(outputChan, doneChan)

	return outputChan
}

func (u *UpdateManager) Progress(ctx context.Context) (int32, error) {
	operation, err := u.rauc.GetOperation()
	if err != nil {
		return -1, fmt.Errorf("failed to get current operation from rauc via D-Bus: %w", err)
	}
	if operation == "installing" {
		percentage, _, _, err := u.rauc.GetProgress()
		if err != nil {
			return -1, fmt.Errorf("failed to get current installation progress from rauc: %w", err)
		}
		return percentage, nil
	}
	return -1, errors.New("no operation in progress")
}
