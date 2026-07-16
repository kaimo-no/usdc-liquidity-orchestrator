# cmd/server/

Thin HTTP microservice wrapping `pkg/liquidity`.

## Routes

| Method | Path | Behaviour |
|---|---|---|
| GET | `/healthz` | `ok` |
| GET | `/v1/chains` | Registered corridors (CAIP-2, USDC, Gateway domain, flags) |
| POST | `/v1/plan` | Decode `PlanRequest` (optional orchestration + fee) → plan; force dry stamps; `execute=true` → UnconfiguredExecutor error + plan body |

Handlers live in `internal/httpserver` (`NewMux`) so `cmd/server/tests` can black-box the surface.

## Invariants

- Max body 1 MiB
- Never log request bodies (wallet inventory)
- No auth required for local/hackathon; production may add bearer later
- Env: `LISTEN_ADDR` (default `:8088`)

## Tests

`cmd/server/tests/` — black-box `package server_test` against `httpserver.NewMux`.
