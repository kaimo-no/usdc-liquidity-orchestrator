# cmd/server/

Thin HTTP microservice wrapping `pkg/liquidity` (+ optional `pkg/execonchain`).

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET | `/` | Plan-only web UI (embedded `internal/httpserver/static/`) |
| GET | `/healthz` | `ok` |
| GET | `/v1/chains` | Registered corridors (CAIP-2, USDC, Gateway domain, `testnet`, `gateway_wallet`) |
| POST | `/v1/plan` | Decode `PlanRequest` → plan; stamp dry/execute; optional Executor |
| POST | `/v1/consolidate` | Decode `ConsolidateRequest` → deposit plan; stamp dry/execute; optional Executor |

Handlers live in `internal/httpserver` (`NewMux` / `NewMuxWithOptions`) so `cmd/server/tests` can black-box the surface.

## Execute stamping (`stampPlanResponse`)

| Case | Plan stamps | Receipt | HTTP |
|---|---|---|---|
| `execute=false` | force dry | none | 200 |
| success (no err) | `dry_run=false` `executed=true` | `tx_hashes` | 200 |
| partial (hashes + err) | `dry_run=false` `executed=false` | `tx_hashes` + error | 400 |
| fail zero hashes | force dry | none + error | 400 |

API error messages use stable `pkg/errors` Message only — never raw RPC strings.

## Env (execute dual gate)

| Var | Notes |
|---|---|
| `LISTEN_ADDR` | default `:8088`; **must be loopback** when execute enabled |
| `ENABLE_TESTNET_EXECUTE` | `1` to enable |
| `AGENT_PRIVATE_KEY` | hex ECDSA (required when enabled) |
| `RPC_URLS_JSON` / `RPC_URL_eip155_*` | testnet RPC map |

Else: `UnconfiguredExecutor` (fail-closed). Never log key/agent/balances/calldata/RPC URL.

## Invariants

- Max body 1 MiB (`MaxBytesReader`)
- **Never log request bodies** (wallet inventory, calldata) — applies to `/v1/plan` **and** `/v1/consolidate`
- No auth required for local/hackathon; production may add bearer later
- Default plan responses force `dry_run=true`, `executed=false`, inventory stamps

## Tests

`cmd/server/tests/` — black-box `package server_test` against `httpserver.NewMux` / `NewMuxWithOptions` (stamp matrix).
