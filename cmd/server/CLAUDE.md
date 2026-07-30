# cmd/server/

Thin HTTP microservice wrapping `pkg/liquidity` (+ optional `pkg/execonchain`).

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET | `/` | MVP web UI (Gateway hero, Live/Asserted/Hybrid inventory, scenario / shortfall plan / consolidate / chains) |
| GET | `/healthz` | `ok` |
| GET | `/v1/chains` | Registered corridors (CAIP-2, USDC, Gateway domain, `testnet`, `gateway_wallet`) |
| POST | `/v1/plan` | Phase B shortfall land `PlanOrchestration` (no pay_to required); stamp dry/execute |
| POST | `/v1/payment-funding` | Scenario Phase A deposits `PlanPaymentFunding` (hard-coded sources; no withdraw) |
| POST | `/v1/consolidate` | Decode `ConsolidateRequest` → deposit plan; stamp dry/execute; optional Executor |
| POST | `/v1/inventory` | Request-scoped live load (`{"agent_address"}`); bare `Inventory` / bare `APIError`; `Cache-Control: no-store` |

Handlers live in `internal/httpserver` (`NewMux` / `NewMuxWithOptions`) so `cmd/server/tests` can black-box the surface. Plan/stamp logic is shared with the CLI via `internal/planio`. Executor dual-gate is `internal/execenv.BuildExecutor` (loopback required for HTTP).

### POST /v1/inventory

| Case | HTTP | Body |
|---|---|---|
| invalid JSON / empty `agent_address` | 400 | bare `APIError` `invalid_query` |
| `LoadInventory` nil / empty RPC map / native RPC fail | 503 | bare `APIError` `liquidity_rail_unavailable` |
| success | 200 | bare `types.Inventory` (Gateway soft-skip OK; may omit `circle_gateway` rows) |

All inventory responses set `Cache-Control: no-store`. Agent is taken from the JSON body only — never invented from env/key. Plan path never calls `LoadInventory`; dry stamps stay `inventory_asserted` + `inventory_unverified`.

## Execute stamping (`internal/planio.StampPlan`)

| Case | Plan stamps | Receipt | HTTP / CLI exit |
|---|---|---|---|
| `execute=false` | force dry | none | 200 / 0 |
| success (no err) | `dry_run=false` `executed=true` | `tx_hashes` | 200 / 0 |
| partial (hashes + err) | `dry_run=false` `executed=false` | `tx_hashes` + error | 400 / 1 |
| fail zero hashes | force dry | none + error | 400 / 1 |

API error messages use stable `pkg/errors` Message only — never raw RPC strings. Peer CLI: `cmd/usdc-liq`.

## Env (execute dual gate)

Optional gitignored `.env` is loaded at startup via `internal/envfile` (does not override process env). VS Code: use **HTTP Server (testnet execute)** launch config for live rails.

| Var | Notes |
|---|---|
| `LISTEN_ADDR` | default `:8088`; **must be loopback** when execute enabled |
| `ENABLE_TESTNET_EXECUTE` | `1` to enable |
| `AGENT_PRIVATE_KEY` | hex ECDSA (required when enabled) |
| `RPC_URL_BASE_SEPOLIA` / `ARBITRUM_SEPOLIA` / `ARC_TESTNET` | named EVM testnet RPCs |
| `RPC_URL_SOLANA_DEVNET` | Solana placeholder (not used by EVM execute) |
| `RPC_URLS_JSON` / `RPC_URL_eip155_*` | alternate CAIP-2 RPC map |
| `MAX_AMOUNT_ATOMIC` | optional Guard cap (positive atomic units) |
| `GATEWAY_API_BASE` | optional Gateway API base for burn/mint transfer |
| `GATEWAY_MAX_FEE_ATOMIC` | optional burn-intent maxFee |

Else: `UnconfiguredExecutor` (fail-closed). Live actions: consolidate, deposit, withdraw. Never log key/agent/balances/calldata/RPC URL.

## Invariants

- Max body 1 MiB (`MaxBytesReader`)
- **Never log request bodies** (wallet inventory, calldata) — applies to `/v1/plan`, `/v1/payment-funding`, `/v1/consolidate`, `/v1/inventory`
- Path-only request logs (`LogRequests`); never agent, balances, RPC URLs
- No auth required for local/hackathon; production may add bearer later
- Default plan responses force `dry_run=true`, `executed=false`, inventory stamps

## Tests

`cmd/server/tests/` — black-box `package server_test` against `httpserver.NewMux` / `NewMuxWithOptions` (stamp matrix).
