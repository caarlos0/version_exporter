# version_exporter

Exports the expiration time of your domains as prometheus metrics.

## Running

```console
./version_exporter --bind ":9333"
```

Or with docker:

```console
docker run -p 127.0.0.1:9333:9333 -v config.yaml:/config.yaml:ro caarlos0/version_exporter
```

Or with docker-compose:

```yaml
version: '3'
services:
  releases:
    image: caarlos0/gversion_exporter:v1
    restart: always
    volumes:
    - /path/to/config.yml:/etc/config.yml:ro
    command:
    - '--config.file=/etc/config.yml'
    ports:
    - 127.0.0.1:9333:9333
    env_file:
    - .env
```

## Configuration

On the prometheus settings, add the domain_expoter prober:

```yaml
scrape_configs:
  - job_name: version
    static_configs:
      - targets: [ 'version_exporter:9222' ]
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
