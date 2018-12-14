package collector

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"

	"github.com/Masterminds/semver"
	"github.com/patrickmn/go-cache"
)

// Release from github api
type Release struct {
	TagName     string    `json:"tag_name,omitempty"`
	Draft       bool      `json:"draft,omitempty"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func findLatest(token, repo string, c *cache.Cache) (*semver.Version, error) {
	var log = log.With("repo", repo)
	version, found := c.Get(repo)
	if found {
		log.Debug("using result from cache")
		return version.(*semver.Version), nil
	}
	log.Info("refreshing")
	releases, err := findReleases(token, repo)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		version, err := semver.NewVersion(release.TagName)
		if err != nil {
			log.With("error", err).
				With("tag", release.TagName).
				Errorf("failed to parse %s", release.TagName)
			continue
		}
		if version.Prerelease() != "" {
			continue
		}
		c.Set(repo, version, cache.DefaultExpiration)
		return version, nil
	}
	return nil, nil
}

func findReleases(token, repo string) ([]Release, error) {
	var releases []Release
	req, _ := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/releases", repo),
		nil,
	)
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return releases, errors.Wrap(err, "failed to get repository releases")
	}
	if resp.StatusCode != http.StatusOK {
		return releases, errors.Wrap(err, "github responded a non-200 status code")
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return releases, errors.Wrap(err, "failed to parse the response body")
	}
	return releases, nil
}
