//go:build dbus && dbus_test

package dbus

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

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
	// Creste simple update manager without functionality to get DBus server to startup
	manager := &raucgithub.UpdateManager{}
	dbusServer, err := New(manager)
	require.NoError(t, err)
	err = dbusServer.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		dbusServer.Close()
	})
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

	dbusServer, err := New(updater, useSessionBus())
	require.NoError(t, err)
	require.NotNil(t, dbusServer)
	err = dbusServer.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		dbusServer.Close()
	})

	cmd := exec.Command("./test_run_update.py")
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)
}

func TestNewUpdateSignal(t *testing.T) {
	t.Skip()
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := raucgithub.NewUpdateManager(repo, raucgithub.WithRaucClient(raucClient), raucgithub.CheckForUpdatesEvery(time.Millisecond*500))
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

	dbusServer, err := New(updater, useSessionBus())
	require.NoError(t, err)
	require.NotNil(t, dbusServer)
	err = dbusServer.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		dbusServer.Close()
	})

	cmd := exec.Command("./test_update_signal.py")
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	require.NoError(t, err)
}
