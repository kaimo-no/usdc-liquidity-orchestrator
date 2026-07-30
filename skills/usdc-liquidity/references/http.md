# HTTP (`cmd/server`)

```bash
go run ./cmd/server   # LISTEN_ADDR=:8088 default
open http://127.0.0.1:8088/
```

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET | `/` | MVP web UI |
| GET | `/healthz` | `ok` |
| GET | `/v1/chains` | corridor registry |
| POST | `/v1/plan` | shortfall-only `PlanOrchestration` |
| POST | `/v1/payment-funding` | scenario full-funding |
| POST | `/v1/consolidate` | Gateway deposit plan |

Handlers use shared `internal/planio` (identical stamps to CLI). Max body 1 MiB. Never log request bodies.

## Execute stamps

| Case | Stamps | Receipt | HTTP |
|---|---|---|---|
| `execute=false` | force dry | none | 200 |
| success | `dry_run=false` `executed=true` | `tx_hashes` | 200 |
| partial (hashes + err) | `executed=false` | hashes + error | 400 |
| fail zero hashes | force dry | error | 400 |

## Dual gate (testnet)

`ENABLE_TESTNET_EXECUTE=1` + key + RPCs + **loopback** `LISTEN_ADDR`. Incomplete config → process fatal at boot.
