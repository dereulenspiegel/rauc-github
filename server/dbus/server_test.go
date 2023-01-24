//go:build dbus_test
// +build dbus_test

package dbus

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/coreos/go-semver/semver"
	"github.com/dereulenspiegel/raucgithub"
	"github.com/dereulenspiegel/raucgithub/mocks"
	"github.com/dereulenspiegel/raucgithub/repository"
	dbus "github.com/godbus/dbus/v5"
	"github.com/holoplot/go-rauc/rauc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func useSessionBus() Option {
	return func(s *Server) *Server {
		s.useSessionBus = true
		return s
	}
}

func TestCreateDBusServer(t *testing.T) {
	dbusServer, err := Start(context.Background(), nil)
	require.NoError(t, err)
	assert.NotNil(t, dbusServer)
}

func TestRunningDBusServerIntegration(t *testing.T) {
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
					AssetName: "cbpifw-raspberrypi3-64_v1.8.2.bundle",
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

	dbusServer, err := Start(context.Background(), updater, useSessionBus())
	require.NoError(t, err)
	assert.NotNil(t, dbusServer)

	cmd := exec.Command("./testclient.py", "a-z", "A-Z")
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	require.NoError(t, err)
}
