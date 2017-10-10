package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/masterminds/semver"

	"github.com/alecthomas/kingpin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	bind    = kingpin.Flag("bind", "addr to bind the server").Default(":9333").String()
	debug   = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token   = os.Getenv("GITHUB_TOKEN")

	updateGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "up_to_date",
		Help: "will be 0 if there is a new version available",
	}, []string{"current", "latest"})
	probeDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})
)

func main() {
	kingpin.Version("version_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *debug {
		log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	log.Info("starting version_exporter ", version)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", probeHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(
			w, `
			<html>
			<head><title>Version Exporter</title></head>
			<body>
				<h1>Version Exporter</h1>
				<p><a href="/metrics">Metrics</a></p>
				<p><a href="/probe?repo=prometheus/prometheus&tag=v1.7.2">probe prometheus/prometheus</a></p>
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

// Release from github api
type Release struct {
	TagName     string    `json:"tag_name,omitempty"`
	Draft       bool      `json:"draft,omitempty"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

func probeHandler(w http.ResponseWriter, r *http.Request) {
	var params = r.URL.Query()
	var repo = params.Get("repo")
	var tag = params.Get("tag")
	var registry = prometheus.NewRegistry()
	var start = time.Now()
	var log = log.With("repo", repo)
	registry.MustRegister(updateGauge)
	registry.MustRegister(probeDurationGauge)
	if repo == "" {
		http.Error(w, "repo parameter is missing", http.StatusBadRequest)
		return
	}
	if tag == "" {
		http.Error(w, "tag parameter is missing", http.StatusBadRequest)
		return
	}
	currentVersion, err := semver.NewVersion(tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	version, err := findLatest(repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if version == nil {
		// repo probably doesnt have any releases at all
		updateGauge.WithLabelValues(currentVersion.String(), "unknown").Set(1)
	} else {
		log.Infof(
			"checking current version (%s) against latest (%s)",
			currentVersion,
			version,
		)
		updateGauge.WithLabelValues(
			currentVersion.String(),
			version.String(),
		).Set(boolToFloat(!version.GreaterThan(currentVersion)))
	}
	probeDurationGauge.Set(time.Since(start).Seconds())
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
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
			log.With("error", err).Errorf("failed to parse %s", release.TagName)
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
