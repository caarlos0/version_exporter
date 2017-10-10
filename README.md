# version_exporter

Exports the expiration time of your domains as prometheus metrics.

## Running

```console
./version_exporter -b ":9333"
```

Or with docker:

```console
docker run -p 9333:9333 caarlos0/version_exporter
```

## Configuration

On the prometheus settings, add the domain_expoter prober:

```yaml
scrape_configs:
  - job_name: version
    scrape_interval: 1m
    metrics_path: /probe
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_repo
        regex: ^(.*)\?tag=.*$
      - source_labels: [__address__]
        target_label: __param_tag
        regex: ^.*?tag=(.*)$
      - source_labels: [__param_repo]
        target_label: repo
      - target_label: __address__
        replacement: localhost:9333 # version_exporter address
    static_configs:
      - targets:
        - prometheus/prometheus?tag=v1.7.1
        - goreleaser/goreleaser?tag=v0.34.0
        - caarlos0/version_exporter?tag=v0.0.1
```

It works more or less like prometheus's
[blackbox_exporter](https://github.com/prometheus/blackbox_exporter).

Alerting rules example:

```rules
ALERT OutdatedSoftware
  IF up_to_date == 0
  FOR 30m
  LABELS {
    severity = "WARNING",
  }
  ANNOTATIONS {
    description = "we are running the version {{ $labels.current }} of {{ $labels.repo }}, but version {{ $labels.latest }} is available",
    summary = "{{ $labels.repo }}: new version available",
  }

```

## Building locally

Install the needed tooling and libs:

```console
make setup
```

Run with:

```console
go run main.go
```

Run tests with:

```console
make test
```
