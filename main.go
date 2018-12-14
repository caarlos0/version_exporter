package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/masterminds/semver"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	yaml "gopkg.in/yaml.v2"
)

const ns = "version"

var (
	bind       = kingpin.Flag("bind", "addr to bind the server").Default(":9333").String()
	debug      = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token      = kingpin.Flag("github.token", "github token").Envar("GITHUB_TOKEN").String()
	configFile = kingpin.Flag("config.file", "config file").Default("config.yaml").ExistingFile()
	interval   = kingpin.Flag("refresh.interval", "time between refreshes with github api").Default("15m").Duration()

	version = "dev"
)

// Config struct representing the config file.
type Config struct {
	Repositories map[string]string `yaml:"repositories"`
}

func main() {
	kingpin.Version("version_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Info("starting version_exporter", version)

	if *debug {
		_ = log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	var config Config
	loadConfig(&config)
	var configCh = make(chan os.Signal, 1)
	signal.Notify(configCh, syscall.SIGHUP)
	go func() {
		for range configCh {
			log.Info("reloading config...")
			loadConfig(&config)
			log.Info("config reloaded...")
		}
	}()

	prometheus.MustRegister(newReleasesCollector(&config))
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w, `
			<html>
			<head><title>Version Exporter</title></head>
			<body>
				<h1>Version Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>
			`,
		)
	})
	log.Info("listening on", *bind)
	if err := http.ListenAndServe(*bind, nil); err != nil {
		log.Fatalf("error starting server: %s", err)
	}
}

func loadConfig(config *Config) {
	bts, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	var newConfig Config
	if err := yaml.Unmarshal(bts, &newConfig); err != nil {
		log.Fatal(err)
	}
	*config = newConfig
}

// Release from github api
type Release struct {
	TagName     string    `json:"tag_name,omitempty"`
	Draft       bool      `json:"draft,omitempty"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

type releasesCollector struct {
	mutex  sync.Mutex
	config *Config

	up             *prometheus.Desc
	upToDate       *prometheus.Desc
	scrapeDuration *prometheus.Desc
}

func newReleasesCollector(config *Config) prometheus.Collector {
	const namespace = "github"
	const subsystem = "version"
	return &releasesCollector{
		config: config,
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
func (c *releasesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.upToDate
	ch <- c.scrapeDuration
}

// Collect all metrics
func (c *releasesCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var success = true
	var start = time.Now()
	for repo, constraint := range c.config.Repositories {
		var log = log.With("repo", repo)
		log.Info("collecting")
		sconstraint, err := semver.NewConstraint(constraint)
		if err != nil {
			log.Errorf("failed to collect for %s: %s", repo, err.Error())
			success = false
			return
		}
		version, err := findLatest(repo)
		if err != nil {
			log.Errorf("failed to collect for %s: %s", repo, err.Error())
			success = false
			return
		}
		if version != nil {
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

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func findLatest(repo string) (*semver.Version, error) {
	releases, err := findReleases(repo)
	if err != nil {
		return nil, err
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		version, err := semver.NewVersion(release.TagName)
		if err != nil {
			log.With("error", err).With("repo", repo).With("tag", release.TagName).
				Errorf("failed to parse %s", release.TagName)
			continue
		}
		if version.Prerelease() != "" {
			continue
		}
		return version, nil
	}
	return nil, nil
}

func findReleases(repo string) ([]Release, error) {
	var releases []Release
	req, _ := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/releases", repo),
		nil,
	)
	if *token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", *token))
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
