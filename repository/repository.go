package repository

import (
	"context"
	"time"

	"github.com/coreos/go-semver/semver"
)

type Update struct {
	Version     *semver.Version
	ReleaseDate time.Time
	Name        string
	Notes       string
	Bundles     []*BundleLink
	Prerelease  bool
}

type BundleLink struct {
	URL           string
	AssetName     string
	Compatibility string
	Size          int64
}

type Repository interface {
	Updates(ctx context.Context) ([]Update, error)
}
