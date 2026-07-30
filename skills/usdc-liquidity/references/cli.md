# CLI (`usdc-liq`)

```bash
go run ./cmd/usdc-liq <command> [flags]
# or: go build -o usdc-liq ./cmd/usdc-liq && ./usdc-liq …
```

Optional gitignored `.env` is loaded at startup (`internal/envfile`).

## Commands

| Command | Input | Output |
|---|---|---|
| `plan` | JSON body (`-f`, default stdin) | `PlanResponse` |
| `consolidate` | JSON body | `PlanResponse` |
| `payment-funding` | JSON body | `PlanResponse` |
| `chains` | none | `ChainsResponse` |
| `inventory` | `-agent` or `AGENT_ADDRESS` + RPC env | wire `Inventory` |
| `demo` | scenario env | multi-plan demo |
| `version` | none | version string |

Shared plan flags: `-f file` (default `-`), `--execute`.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success (dry or full execute) |
| 1 | plan/execute/configure failure (JSON still on stdout when plan-shaped) |
| 2 | usage / unknown command |

Body limit **1 MiB**. Stderr notes never include secrets, agent_address, balances, or RPC URLs.

## Dry vs execute

- Dry: always `UnconfiguredExecutor` — no live config required.
- `--execute`: `execenv.BuildExecutor` without loopback; incomplete dual-gate → exit 1 sanitized, no Execute call.
