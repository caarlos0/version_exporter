package collector

import (
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/caarlos0/version_exporter/client"
	"github.com/caarlos0/version_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type versionCollector struct {
	mutex  sync.Mutex
	config *config.Config
	client client.Client

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

// NewVersionCollector returns a versions collector
func NewVersionCollector(config *config.Config, client client.Client) prometheus.Collector {
	const namespace = "version"
	const subsystem = ""
	return &versionCollector{
		config: config,
		client: client,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up"),
			"Exporter is being able to talk with GitHub API",
			nil,
			nil,
		),
		upToDate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "up_to_date"),
			"Wether the repository latest version is in the specified semantic versioning range",
			[]string{"repository", "constraint", "latest"},
			nil,
		),
		scrapeDuration: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "scrape_duration_seconds"),
			"Returns how long the probe took to complete in seconds",
			nil,
			nil,
		),
	}
}

// Describe all metrics
func (c *versionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.upToDate
	ch <- c.scrapeDuration
}

// Collect all metrics
func (c *versionCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var success = true
	var start = time.Now()
	for repo, constraint := range c.config.Repositories {
		var log = log.With("repo", repo)
		log.Debug("collecting")
		sconstraint, err := semver.NewConstraint(constraint)
		if err != nil {
			log.Errorf("failed to collect for %s: %s", repo, err.Error())
			success = false
			return
		}
		version, err := getLatest(c.client, repo)
		if err != nil {
			log.Errorf("failed to collect for %s: %s", repo, err.Error())
			success = false
			return
		}
		if version == nil {
			continue
		}
		var up = sconstraint.Check(version)
		log.With("constraint", constraint).
			With("latest", version).
			With("up_to_date", up).
			Debug("checked")
		ch <- prometheus.MustNewConstMetric(
			c.upToDate,
			prometheus.GaugeValue,
			boolToFloat(up),
			repo,
			constraint,
			version.String(),
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.up,
		prometheus.GaugeValue,
		boolToFloat(success),
	)
	ch <- prometheus.MustNewConstMetric(
		c.scrapeDuration,
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)
}

func getLatest(client client.Client, repo string) (*semver.Version, error) {
	var log = log.With("repo", repo)
	releases, err := client.Releases(repo)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			log.With("tag", release.TagName).Debug("ignored draft/prerelease")
			continue
		}
		version, err := semver.NewVersion(release.TagName)
		if err != nil {
			log.With("error", err).
				With("tag", release.TagName).
				Errorf("failed to parse tag %s", release.TagName)
			continue
		}
		if version.Prerelease() != "" {
			log.With("tag", release.TagName).Debug("ignored prerelease")
			continue
		}
		return version, nil
	}
	return nil, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
