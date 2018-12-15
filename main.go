package main

import (
	"fmt"
	"net/http"

	"github.com/alecthomas/kingpin"
	"github.com/caarlos0/version_exporter/client"
	"github.com/caarlos0/version_exporter/collector"
	"github.com/caarlos0/version_exporter/config"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

// nolint: gochecknoglobals
var (
	bind       = kingpin.Flag("bind", "addr to bind the server").Default(":9333").String()
	debug      = kingpin.Flag("debug", "show debug logs").Default("false").Bool()
	token      = kingpin.Flag("github.token", "github token").Envar("GITHUB_TOKEN").String()
	configFile = kingpin.Flag("config.file", "config file").Default("config.yaml").ExistingFile()
	interval   = kingpin.Flag("refresh.interval", "time between refreshes with github api").Default("15m").Duration()

	version = "dev"
)

func main() {
	kingpin.Version("version_exporter version " + version)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Info("starting version_exporter")

	if *debug {
		_ = log.Base().SetLevel("debug")
		log.Debug("enabled debug mode")
	}

	var cache = cache.New(*interval, *interval)

	var cfg config.Config
	config.Load(*configFile, &cfg, func() {
		log.Debug("flushing cache...")
		cache.Flush()
	})

	var client = client.NewCachedClient(client.NewClient(*token), cache)

	prometheus.MustRegister(collector.NewVersionCollector(&cfg, client))
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
	log.Info("listening on ", *bind)
	if err := http.ListenAndServe(*bind, nil); err != nil {
		log.Fatalf("error starting server: %s", err)
	}
}
