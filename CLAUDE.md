# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Prometheus exporter (Go) for Braiins OS / BOSminer miners. On every scrape it queries the miner's CGMiner-style TCP API (port 4028) and exposes hashrate, temperature, fan, power and pool metrics. API reference: `docs/bosminer-api-spec.md`. A Grafana dashboard lives in `grafana/`.

## Commands

```sh
make build              # -> bin/bosminer-exporter
make test               # go test ./...
make vet                # go vet ./... (CI runs vet + test)
make run MINER=host:4028
go test ./internal/collector -run TestName   # single test
```

## Architecture

- `cmd/bosminer-exporter/main.go`: flag/env config (`--miner.address`, `--miner.timeout`, `--web.listen-address`, each with a `BOSMINER_EXPORTER_*` env var), serves `/metrics`, `/healthz`, `/`.
- `internal/cgminer`: minimal client. One TCP connection per command, JSON request, NUL-terminated JSON reply, decoded into `Response{Status, Data map[string][]map[string]any}`. Rows stay untyped; use the `Float`/`String` helpers to read fields.
- `internal/collector`: `Collector` implements `prometheus.Collector` and depends on the small `Querier` interface (satisfied by `*cgminer.Client`, faked in tests). `Collect` runs one function per API command (version, summary, fans, temps, devdetails, pools, devs, tempctrl, tunerstatus). Each is independent: a failure logs a warning and sets `bosminer_up` to 0 without hiding the other commands' metrics.

Adding a metric: add a `*prometheus.Desc` field, create it in `New` via `desc(...)`, **and** add it to the list in `Describe`, then emit it from the matching per-command function.

## Conventions

- The exporter is strictly read-only: never send pool-mutating or `pause`/`resume` commands.
- The API reports hashrate in MH/s; metrics are converted to GH/s (`/1000`), with a `window` label (5s, 1m, 5m, 15m, 24h, av).
- Metric namespace is `bosminer`; counters end in `_total`.

## Release

Conventional commits drive semantic-release (`.releaserc.json`) on `main`; `release.yml` publishes the Docker image to GHCR. Use `feat:`/`fix:` prefixes accordingly.
