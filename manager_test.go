package raucgithub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/dereulenspiegel/raucgithub/mocks"
	"github.com/dereulenspiegel/raucgithub/repository"
	dbus "github.com/godbus/dbus/v5"
	"github.com/holoplot/go-rauc/rauc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDefaultExtractCompatibility(t *testing.T) {
	assetNames := map[string]string{
		"cbpfw-rpi3_v0.1.0.img": "cbpfw-rpi3",
		"cbpifw-someboard_.img": "cbpifw-someboard",
		"invalid-assetname.img": "",
	}

	for assetName, compat := range assetNames {
		assert.Equal(t, compat, ExtractCompatibility(assetName))
	}
}

func TestNoUpdatesAvailable(t *testing.T) {
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)
	t.Cleanup(func() {
		repo.AssertExpectations(t)
		raucClient.AssertExpectations(t)
	})

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := NewUpdateManager(repo, WithRaucClient(raucClient))
	require.NoError(t, err)

	repo.EXPECT().Updates(mock.Anything).Return([]repository.Update{
		{
			Version: semver.New("1.2.2"),
		},
		{
			Version: semver.New("1.6.6"),
		},
	}, nil)
	raucClient.EXPECT().GetBootSlot().Return("slot0", nil)
	raucClient.EXPECT().GetCompatible().Return("cbpifw-rasperrypi3-64", nil)
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

	update, err := updater.CheckForUpdate(context.Background())
	assert.ErrorIs(t, err, ErrNoSuitableUpdate)
	assert.Nil(t, update)
}

func TestSuitableUpdateFound(t *testing.T) {
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)
	t.Cleanup(func() {
		repo.AssertExpectations(t)
		raucClient.AssertExpectations(t)
	})

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := NewUpdateManager(repo, WithRaucClient(raucClient))
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

	update, err := updater.CheckForUpdate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, update)
}

func TestCheckUpdateWithoutAssetName(t *testing.T) {
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)
	t.Cleanup(func() {
		repo.AssertExpectations(t)
		raucClient.AssertExpectations(t)
	})

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := NewUpdateManager(repo, WithRaucClient(raucClient))
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
					URL: "https://example.com/cbpifw-raspberrypi3-64_v1.8.2_update.bin",
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

	update, err := updater.CheckForUpdate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, update)

	assert.Equal(t, "Penguin", update.Name)
}

func TestRunUpdate(t *testing.T) {
	repo := mocks.NewRepository(t)
	raucClient := mocks.NewRaucDBUSClient(t)
	t.Cleanup(func() {
		repo.AssertExpectations(t)
		raucClient.AssertExpectations(t)
	})

	currentVersion, err := semver.NewVersion("1.8.1")
	require.NoError(t, err)

	updater, err := NewUpdateManager(repo, WithRaucClient(raucClient))
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
					URL: "https://example.com/cbpifw-raspberrypi3-64_v1.8.2_update.bin",
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

	update, err := updater.CheckForUpdate(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, update)

	assert.Equal(t, "Penguin", update.Name)

	raucClient.EXPECT().InstallBundle("https://example.com/cbpifw-raspberrypi3-64_v1.8.2_update.bin", mock.Anything).After(time.Millisecond * 200).Return(nil)

	raucClient.EXPECT().GetProgress().Return(75, "Installing", 1, nil)
	raucClient.EXPECT().GetOperation().Maybe().Return("installing")

	wg := &sync.WaitGroup{}
	wg.Add(1)
	updateChan := updater.InstallNextUpdateAsync(context.Background(), func(success bool, err error) {
		assert.True(t, success)
		assert.NoError(t, err)
		wg.Done()
	})
	wg.Wait()
	assert.GreaterOrEqual(t, len(updateChan), 1)
}
