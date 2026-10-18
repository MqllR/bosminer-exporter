# BOSminer API specification

Source: <https://academy.braiins.com/braiins-os/papi-bosminer>

The BOSminer API is a basic subset of the upstream CGMiner API plus Braiins-specific commands.

> **Deprecation notice:** Braiins plans to deprecate this API in favour of its gRPC/REST Public API.

## Connection

| Item | Value |
|------|-------|
| Port | `4028` (TCP) |
| Request format | JSON: `{"command":"<name>"}` |
| Command with parameter | `{"command":"asc","parameter":0}` |
| Multiple commands | join with `+`, e.g. `{"command":"version+config"}` |

Example:

```sh
echo '{"command":"summary"}' | nc <miner-ip> 4028 | jq .
```

The server closes the connection after replying. Upstream CGMiner terminates replies with a trailing NUL byte; clients should strip it before JSON decoding.

## Common response envelope

```json
{
  "id": 1,
  "STATUS": [
    { "STATUS": "S", "When": 1700000000, "Code": 202, "Msg": "4 Fan(s)", "Description": "BOSminer+ 0.2.0-ea64aec8e" }
  ],
  "<COMMAND-SPECIFIC-ARRAY>": [ ]
}
```

- `STATUS.STATUS`: `S` = success (CGMiner convention: also `W` warning, `E` error, `I` info).
- `STATUS.When`: Unix timestamp. `STATUS.Code`: message code (see below).
- With `+` chained commands, CGMiner returns an object keyed by lower-case command name, each holding an array of the envelope parts. Confirm on a real miner before relying on it.

## CGMiner-derived commands

The documentation gives only short descriptions and no response examples for these. Field names are expected to follow upstream CGMiner and **must be verified against a real miner**.

| Command | Parameters | Description | Read-only |
|---------|-----------|-------------|-----------|
| `asccount` | `N` | Details of a single ASC, same format as `devs` | yes |
| `asc` | `\|N` | Number of ASCs | yes |
| `config` | none | Miner configuration information | yes |
| `devdetails` | none | Per-chain configuration (see below) | yes |
| `devs` | none | Each available ASC with details | yes |
| `edevs` | `[\|old]` | Like `devs`, ignoring blacklisted and zombie devices unless `old` is given | yes |
| `pools` | none | Pool configuration and statistics | yes |
| `summary` | none | Status summary of the miner | yes |
| `stats` | none | Each device or pool with getwork timing stats | yes |
| `estats` | none | Like `stats`, ignoring blacklisted and zombie devices | yes |
| `version` | none | API and BOSminer version | yes |
| `check` | none | Checks whether a command exists and is accessible | yes |
| `coin` | none | Coin mining information | yes |
| `lcd` | none | Short status summary | yes |
| `switchpool` | `\|N` | Switch selected pool to highest priority | **no** |
| `enablepool` | `\|N` | Enable selected pool | **no** |
| `disablepool` | `\|N` | Disable selected pool | **no** |
| `addpool` | `\|URL,USR,PASS` | Add a pool | **no** |
| `removepool` | `\|N` | Remove a pool | **no** |

**Known limitation:** `switchpool`, `enablepool`, `disablepool`, `addpool` and `removepool` are not fully implemented in Braiins OS: results reset after restart and pools are not actually activated. Braiins states this is a known issue being fixed.

## Braiins-specific commands

### `fans`: fan statistics

Status code `202`, `Msg` e.g. `"4 Fan(s)"`. Array `FANS`:

| Field | Description |
|-------|-------------|
| `FAN` | Fan index |
| `ID` | Fan ID |
| `RPM` | Rotation speed |
| `Speed` | Speed (percentage) |

### `tempctrl`: temperature control configuration

Status code `200`, `Msg` `"Temperature control"`. Array `TEMPCTRL`:

| Field | Description |
|-------|-------------|
| `Mode` | e.g. `"Automatic"` |
| `Target` | Target temperature |
| `Hot` | Hot threshold |
| `Dangerous` | Dangerous threshold |

### `temps`: temperature data

Status code `201`, `Msg` e.g. `"3 Temp(s)"`. Array `TEMPS`:

| Field | Description |
|-------|-------------|
| `ID` | Hash chain ID |
| `Board` | Board temperature |
| `Chip` | Chip temperature |
| `TEMP` | Temperature index/entry |

### `tunerstatus`: tuning statistics

Status code `203`, `Msg` `"Tuner Status"`. Array `TUNERSTATUS`:

| Field | Description |
|-------|-------------|
| `ApproximateMinerPowerConsumption` | Estimated miner power (W) |
| `ApproximateChainPowerConsumption` | Estimated chain power |
| `DynamicPowerScaling` | e.g. `"Disabled"` |
| `PowerLimit` | Configured power limit |
| `TunerChainStatus` | Array, one entry per chain (below) |

`TunerChainStatus` item:

| Field | Description |
|-------|-------------|
| `HashchainIndex` | Chain index |
| `ApproximatePowerConsumptionWatt` | Chain power (W) |
| `PowerLimitWatt` | Chain power limit (W) |
| `Status` | e.g. `"Tuning individual chips"` |
| `TunerRunning` | Whether the tuner is running |
| `Iteration` | Tuning iteration |
| `StageElapsed` | Time elapsed in current stage |
| `TuningElapsed` | Total tuning time |
| `LoadedProfileCreatedOn` | Profile creation date |

### `devdetails`: device details

Status code `69`, `Msg` `"Device Details"`. Array `DEVDETAILS`:

| Field | Description |
|-------|-------------|
| `DEVDETAILS` | Device index |
| `ID` | Device ID |
| `Name` | e.g. `"Hash Chain 6"` |
| `Model` | e.g. `"Bitmain Antminer S9"` |
| `Driver` | Driver name |
| `Kernel` | Kernel |
| `Device Path` | Device path |
| `Chips` | Chip count |
| `Cores` | Core count |
| `Frequency` | Chip frequency |
| `Voltage` | Voltage |

### `pause` / `resume`: **mutating**

- `pause` stops mining and power consumption and prepares the miner to resume.
- `resume` resumes mining.

The documentation shows log output (e.g. "Interrupted by PAUSE command", "Waiting for RESUME command...", "RESUME command received"), not a JSON response.

## Exporter usage

The exporter is read-only. It only polls:

| Command | Used for |
|---------|----------|
| `version` | `bosminer_info` |
| `summary` | hashrate, shares, uptime (field names to verify) |
| `devs` / `devdetails` | per-chain hashrate, chips, frequency, voltage |
| `pools` | pool status, accepted/rejected shares |
| `fans` | fan RPM / speed |
| `temps` | board and chip temperatures |
| `tempctrl` | target / hot / dangerous thresholds |
| `tunerstatus` | power consumption and tuner state |

It must **never** call `switchpool`, `enablepool`, `disablepool`, `addpool`, `removepool`, `pause` or `resume`.
