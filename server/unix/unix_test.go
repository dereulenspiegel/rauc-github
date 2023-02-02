package unix

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/mocks"
	"github.com/dereulenspiegel/raucgithub/repository"
	"github.com/godbus/dbus/v5"
	"github.com/holoplot/go-rauc/rauc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func executePyHelper(t *testing.T, helperName string) {
	t.Helper()
	cmd := exec.Command(helperName)
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	require.NoError(t, err)
}

func TestStatus(t *testing.T) {
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := raucgithub.NewUpdateManager(repo, raucgithub.WithRaucClient(raucClient))
	require.NoError(t, err)

	repo.EXPECT().Updates(mock.Anything).Return([]repository.Update{
		{
			Version: semver.New("1.2.2"),
		},
		{
			Version: semver.New("1.6.6"),
		},
		{
			Name:    "Penguin",
			Version: semver.New("1.8.2"),
			Bundles: []*repository.BundleLink{
				{
					URL:       "https://example.com/update-1.8.2.bundle",
					AssetName: "cbpifw-raspberrypi3-64_v1.8.2_update.bin",
				},
			},
		},
	}, nil)

	raucClient.EXPECT().GetBootSlot().Return("slot0", nil)
	raucClient.EXPECT().GetCompatible().Return("cbpifw-raspberrypi3-64", nil)
	raucClient.EXPECT().GetSlotStatus().Return([]rauc.SlotStatus{
		{
			SlotName: "slot0",
			Status: map[string]dbus.Variant{
				"bundle.version": dbus.MakeVariant(currentVersion.String()),
				"bundle.foo":     dbus.MakeVariant("bar"),
			},
		},
		{
			SlotName: "slot1",
			Status: map[string]dbus.Variant{
				"bundle.version": dbus.MakeVariant("1.7.0"),
				"bundle.foo":     dbus.MakeVariant("bar"),
			},
		},
	}, nil)

	raucClient.EXPECT().GetOperation().Return("idle", nil)

	conf := viper.New()
	conf.Set("enabled", true)
	conf.Set("socketPath", "./update.socket")
	t.Cleanup(func() {
		os.Remove("./update.socket")
	})

	server, err := New(updater, conf)
	require.NoError(t, err)
	require.NotNil(t, server)
	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Close()
	})
	executePyHelper(t, "./test_check_status.py")
}

func TestRunUpdate(t *testing.T) {
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := raucgithub.NewUpdateManager(repo, raucgithub.WithRaucClient(raucClient))
	require.NoError(t, err)

	repo.EXPECT().Updates(mock.Anything).Return([]repository.Update{
		{
			Version: semver.New("1.2.2"),
		},
		{
			Version: semver.New("1.6.6"),
		},
		{
			Name:    "Penguin",
			Version: semver.New("1.8.2"),
			Bundles: []*repository.BundleLink{
				{
					URL:       "https://example.com/update-1.8.2.bundle",
					AssetName: "cbpifw-raspberrypi3-64_v1.8.2_update.bin",
				},
			},
		},
	}, nil)

	raucClient.EXPECT().GetBootSlot().Return("slot0", nil)
	raucClient.EXPECT().GetCompatible().Return("cbpifw-raspberrypi3-64", nil)
	raucClient.EXPECT().GetSlotStatus().Return([]rauc.SlotStatus{
		{
			SlotName: "slot0",
			Status: map[string]dbus.Variant{
				"bundle.version": dbus.MakeVariant(currentVersion.String()),
				"bundle.foo":     dbus.MakeVariant("bar"),
			},
		},
		{
			SlotName: "slot1",
			Status: map[string]dbus.Variant{
				"bundle.version": dbus.MakeVariant("1.7.0"),
				"bundle.foo":     dbus.MakeVariant("bar"),
			},
		},
	}, nil)

	raucClient.EXPECT().InstallBundle("https://example.com/update-1.8.2.bundle", mock.Anything).
		After(time.Millisecond * 500).Return(nil)

	raucClient.EXPECT().GetProgress().Return(75, "installing", 1, nil)
	raucClient.EXPECT().GetOperation().Return("installing", nil)

	conf := viper.New()
	conf.Set("enabled", true)
	conf.Set("socketPath", "./update.socket")
	t.Cleanup(func() {
		os.Remove("./update.socket")
	})

	server, err := New(updater, conf)
	require.NoError(t, err)
	require.NotNil(t, server)
	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Close()
	})
	executePyHelper(t, "./test_start_update.py")
}
