# bosminer-exporter

A [Prometheus](https://prometheus.io) exporter for miners running [Braiins OS](https://braiins.com/os) (BOSminer). It queries the miner's CGMiner-style API (TCP port 4028) on each scrape and exposes hashrate, temperatures, fans and power metrics.

The exporter is **read-only**: it never calls pool-mutating or `pause`/`resume` commands. See [docs/bosminer-api-spec.md](docs/bosminer-api-spec.md) for the full API reference.

> The BOSminer API is slated for deprecation by Braiins in favour of its gRPC/REST API.

## Setup

### Prerequisites

- A miner running Braiins OS, reachable from the exporter on port 4028.
- Go 1.26+ (to build from source) or Docker.

### Build from source

```sh
make build
./bin/bosminer-exporter --miner.address 192.168.1.50:4028
```

### Docker

Images are published to GHCR on every release (the package is private, so run `docker login ghcr.io` with a token that has `read:packages` first):

```sh
docker run -d -p 9928:9928 ghcr.io/mqllr/bosminer-exporter:latest --miner.address 192.168.1.50:4028
```

Or build locally:

```sh
docker build -t bosminer-exporter .
docker run -d -p 9928:9928 bosminer-exporter --miner.address 192.168.1.50:4028
```

Metrics are then served on <http://localhost:9928/metrics>.

### Prometheus

```yaml
scrape_configs:
  - job_name: bosminer
    static_configs:
      - targets: ["exporter-host:9928"]
```

## Configuration

| Flag | Environment variable | Default | Description |
|------|----------------------|---------|-------------|
| `--miner.address` | `BOSMINER_EXPORTER_MINER_ADDRESS` | `localhost:4028` | `host:port` of the BOSminer API |
| `--miner.timeout` | `BOSMINER_EXPORTER_MINER_TIMEOUT` | `5s` | Timeout per API command |
| `--web.listen-address` | `BOSMINER_EXPORTER_LISTEN_ADDRESS` | `:9928` | Address to serve metrics on |

Endpoints: `/metrics`, `/healthz`, `/`.

## Metrics

| Metric | Labels | Source command |
|--------|--------|----------------|
| `bosminer_up` | | all (0 if any command fails) |
| `bosminer_scrape_duration_seconds` | | |
| `bosminer_info` | `version` | `version` |
| `bosminer_hashrate_ghs` | `window` (5s, 1m, 5m, 15m, 24h, av) | `summary` |
| `bosminer_chain_hashrate_ghs` | `chain`, `window` | `devs` |
| `bosminer_shares_accepted_total`, `bosminer_shares_rejected_total`, `bosminer_hardware_errors_total` | | `summary` |
| `bosminer_chain_chips`, `bosminer_chain_cores`, `bosminer_chain_frequency_mhz`, `bosminer_chain_voltage_volts` | `chain`, `model` | `devdetails` |
| `bosminer_pool_up`, `bosminer_pool_stratum_active`, `bosminer_pool_priority`, `bosminer_pool_stratum_difficulty`, `bosminer_pool_last_share_timestamp_seconds` | `pool`, `url` | `pools` |
| `bosminer_pool_shares_{accepted,rejected,stale}_total`, `bosminer_pool_difficulty_{accepted,rejected}_total` | `pool`, `url` | `pools` |
| `bosminer_fan_rpm`, `bosminer_fan_speed_percent` | `fan` | `fans` |
| `bosminer_board_temperature_celsius`, `bosminer_chip_temperature_celsius` | `chain` | `temps` |
| `bosminer_temp_target_celsius`, `bosminer_temp_hot_celsius`, `bosminer_temp_dangerous_celsius` | | `tempctrl` |
| `bosminer_power_watts` | | `tunerstatus` |
| `bosminer_chain_power_watts`, `bosminer_chain_power_limit_watts` | `chain` | `tunerstatus` |

Field names were verified against a real miner (BOSer boser-openwrt 0.1.0). Metrics with missing fields are omitted. Note `temps` may return only some chains.

## Grafana dashboard

Import [grafana/bosminer-dashboard.json](grafana/bosminer-dashboard.json) (Dashboards → New → Import). It has a datasource and an instance selector and covers hashrate, efficiency, temperatures, fans, power, pools/shares and per-chain frequency/voltage.

## Development

```sh
make test
make vet
```

## License

[MIT](LICENSE)

## Releasing

Versions are computed automatically by [semantic-release](https://semantic-release.gitbook.io) from [Conventional Commits](https://www.conventionalcommits.org) merged into `main`: `fix:` → patch, `feat:` → minor, `feat!:` / `BREAKING CHANGE:` → major. Each release creates a `vX.Y.Z` tag, a GitHub Release and the Docker image tags `X.Y.Z`, `X.Y`, `X` and `latest`.
