# version_exporter

Exports versions of GitHub projects as Prometheus metrics, with constraint
version, from a configuration file, and latest version, fetched from GitHub, as
labels.

## Running

```bash
version_exporter --bind ":9333"
```

Or with docker:

```bash
docker run -p 127.0.0.1:9333:9333 -v $PWD/config.yaml:/config.yaml caarlos0/version_exporter
```

Or with `docker-compose`:

```yaml
version: '3'
services:
  releases:
    image: caarlos0/gversion_exporter:v1
    restart: always
    volumes:
    - /path/to/config.yml:/etc/config.yml
    command:
    - '--config.file=/etc/config.yml'
    ports:
    - 127.0.0.1:9333:9333
    env_file:
    - .env
```

You can personalize the `config.yaml` file like following:

```yaml
repositories:
  # repository: semver constraint (check https://github.com/masterminds/semver#working-with-pre-release-versions)
  prometheus/alertmanager: ~v0.14.0
  prometheus/prometheus: ^2.1.0
  caarlos0/version_exporter: 0.0.5
```

> You can reload the configuration file by sending a `SIGHUP` to
> `version_exporter` process.

On the Prometheus settings, add the `version_exporter` job:

```yaml
scrape_configs:
  - job_name: version
    static_configs:
      - targets: [ 'version_exporter:9333' ]
```

Alerting rules example:

```yaml
groups:
- name: versions
  rules:
  - alert: SoftwareOutOfDate
    expr: version_up_to_date == 0
    for: 1s
    labels:
      severity: warning
    annotations:
      summary: "{{$labels.repository}}: out of date"
      description: "latest version {{ $labels.latest }} is not within constraint {{ $labels.constraint }}"
```

## Building locally

Install the needed tooling and libs:

```bash
go mod tidy
```

Run with:

```bash
go run main.go
```

