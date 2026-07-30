# HTTP (`cmd/server`)

```bash
go run ./cmd/server   # LISTEN_ADDR=:8088 default
open http://127.0.0.1:8088/
```

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET | `/` | MVP web UI (Gateway hero, Live/Asserted/Hybrid inventory, activity timeline) |
| GET | `/healthz` | `ok` |
| GET | `/v1/chains` | corridor registry |
| POST | `/v1/plan` | shortfall-only `PlanOrchestration` |
| POST | `/v1/payment-funding` | scenario full-funding |
| POST | `/v1/consolidate` | Gateway deposit plan |
| POST | `/v1/inventory` | live balances for `{"agent_address"}` (bare `Inventory` / bare `APIError`; `Cache-Control: no-store`) |

Handlers use shared `internal/planio` (identical stamps to CLI). Max body 1 MiB. Never log request bodies.

### Inventory status matrix

| Case | HTTP | Code |
|---|---|---|
| invalid JSON / empty agent | 400 | `invalid_query` |
| loader unavailable / empty RPC / native RPC fail | 503 | `liquidity_rail_unavailable` |
| success (Gateway soft-skip OK) | 200 | bare `Inventory` |

Agent from body only (never env/key). Plan path does not call live load; plans still stamp `inventory_unverified`.

## Execute stamps

| Case | Stamps | Receipt | HTTP |
|---|---|---|---|
| `execute=false` | force dry | none | 200 |
| success | `dry_run=false` `executed=true` | `tx_hashes` | 200 |
| partial (hashes + err) | `executed=false` | hashes + error | 400 |
| fail zero hashes | force dry | error | 400 |

## Dual gate (testnet)

`ENABLE_TESTNET_EXECUTE=1` + key + RPCs + **loopback** `LISTEN_ADDR`. Incomplete config → process fatal at boot.
