package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/dereulenspiegel/raucgithub/repository"
	"github.com/google/go-github/v49/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type GithubRepo struct {
	client *github.Client
	owner  string
	repo   string
	logger logrus.FieldLogger
}

func New(conf *viper.Viper) (repository.Repository, error) {
	owner := conf.GetString("owner")
	repo := conf.GetString("repo")
	return NewRepo(owner, repo)
}

func NewRepo(owner, repo string) (*GithubRepo, error) {
	githubClient := github.NewClient(nil)
	return &GithubRepo{
		client: githubClient,
		owner:  owner,
		repo:   repo,
		logger: logrus.WithFields(logrus.Fields{"repotype": "github", "owner": owner, "repo": repo}),
	}, nil
}

func (g *GithubRepo) Updates(ctx context.Context) (updates []repository.Update, err error) {
	logger := g.logger
	releases, _, err := g.client.Repositories.ListReleases(ctx, g.owner, g.repo, &github.ListOptions{
		PerPage: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query github repo %s/%s: %w", g.owner, g.repo, err)
	}

	for _, release := range releases {
		if release.GetDraft() {

			continue
		}
		tagName := strings.TrimPrefix(release.GetTagName(), "v")
		version, err := semver.NewVersion(tagName)
		if err != nil {
			logger.WithError(err).WithField("tagName", *release.TagName).
				Error("release can't be used because the tag name is not a semver version")
			continue
		}
		update := repository.Update{
			Version:     version,
			ReleaseDate: release.PublishedAt.Time,
			Name:        release.GetName(),
			Notes:       release.GetBody(),
			Prerelease:  release.GetPrerelease(),
		}

		for _, asset := range release.Assets {
			bundle := repository.BundleLink{
				URL:       *asset.BrowserDownloadURL,
				Size:      int64(asset.GetSize()),
				AssetName: asset.GetName(),
			}
			update.Bundles = append(update.Bundles, &bundle)
		}

		updates = append(updates, update)
	}
	return
}
