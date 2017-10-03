package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/masterminds/semver"

	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

var (
	bind    = kingpin.Flag("bind", "addr to bind the server").Default(":9222").String()
	version = "master"

	majorGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "latest_version_major",
		Help: "latest major version of a given repository",
	})
	minorGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "latest_version_minor",
		Help: "latest minor version of a given repository",
	})
	patchGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "latest_version_patch",
		Help: "latest patch version of a given repository",
	})
	probeDurationGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "probe_duration_seconds",
		Help: "Returns how long the probe took to complete in seconds",
	})
)

func main() {
	kingpin.Version("version_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Info("starting version_exporter", version)

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
				<p><a href="/probe?target=prometheus/prometheus">probe prometheus/prometheus</a></p>
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

type Release struct {
	TagName     string    `json:"tag_name,omitempty"`
	Draft       bool      `json:"draft,omitempty"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

func probeHandler(w http.ResponseWriter, r *http.Request) {
	var params = r.URL.Query()
	var target = params.Get("target")
	var registry = prometheus.NewRegistry()
	var start = time.Now()
	var log = log.With("repo", target)
	registry.MustRegister(majorGauge)
	registry.MustRegister(minorGauge)
	registry.MustRegister(patchGauge)
	registry.MustRegister(probeDurationGauge)
	if target == "" {
		http.Error(w, "target parameter is missing", http.StatusBadRequest)
		return
	}
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases", target))
	if err != nil {
		http.Error(w, "failed to get repository releases", http.StatusBadRequest)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("github responded with %d", resp.StatusCode), http.StatusBadRequest)
		return
	}
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		http.Error(w, "failed to parse body: "+err.Error(), http.StatusBadRequest)
		return
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
		majorGauge.Set(float64(version.Major()))
		minorGauge.Set(float64(version.Minor()))
		patchGauge.Set(float64(version.Patch()))
		break
	}
	probeDurationGauge.Set(time.Since(start).Seconds())
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
}
