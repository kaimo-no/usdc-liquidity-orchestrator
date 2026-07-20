# cmd/server/

Thin HTTP microservice wrapping `pkg/liquidity`.

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET | `/healthz` | `ok` |
| GET | `/v1/chains` | Registered corridors (CAIP-2, USDC, Gateway domain, `testnet`, `gateway_wallet`) |
| POST | `/v1/plan` | Decode `PlanRequest` (optional orchestration + fee) → plan; force dry stamps; `execute=true` → UnconfiguredExecutor error + plan body |
| POST | `/v1/consolidate` | Decode `ConsolidateRequest` → full-balance Gateway deposit plan; force dry stamps; no inventory echo; `execute=true` fail-closed |

Handlers live in `internal/httpserver` (`NewMux`) so `cmd/server/tests` can black-box the surface.

## Invariants

- Max body 1 MiB (`MaxBytesReader`)
- **Never log request bodies** (wallet inventory, calldata) — applies to `/v1/plan` **and** `/v1/consolidate`
- No auth required for local/hackathon; production may add bearer later
- Env: `LISTEN_ADDR` (default `:8088`)
- Plan responses force `dry_run=true`, `executed=false`, inventory stamps

## Tests

`cmd/server/tests/` — black-box `package server_test` against `httpserver.NewMux`.
