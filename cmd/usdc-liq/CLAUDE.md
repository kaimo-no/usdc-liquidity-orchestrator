# cmd/usdc-liq/

Thin CLI entrypoint for **usdc-liq** — dual-surface peer of `cmd/server` (HTTP) and `pkg/liquidity` (library).

## Behaviour

Logic lives in `internal/liqcli.Main`. This package only:

1. Loads optional gitignored `.env` via `internal/envfile` (never logs values)
2. Calls `liqcli.Main(os.Args[1:], stdin, stdout, stderr)` and exits with its code

## Subcommands

| Command | HTTP parity | Notes |
|---|---|---|
| `plan` | `POST /v1/plan` | shortfall-only; `-f` body, `--execute` |
| `consolidate` | `POST /v1/consolidate` | deposit plan; same flags |
| `payment-funding` | `POST /v1/payment-funding` | scenario full-funding |
| `chains` | `GET /v1/chains` | corridor registry |
| `inventory` | CLI-only | live balances; needs agent + RPCs |
| `demo` | CLI-only | `internal/demorun` worked examples |
| `version` | CLI-only | print version |

## Dry vs execute

- **Dry (default):** always `UnconfiguredExecutor` — no live keys/RPCs required
- **`--execute`:** `execenv.BuildExecutor(Options{})` (no loopback); incomplete dual-gate → exit 1 sanitized, no Execute
- HTTP boot still requires loopback `LISTEN_ADDR` when execute is enabled

## Exit codes

| Code | Meaning |
|---|---|
| 0 | StampOK (dry or full execute success) |
| 1 | StampPartial / StampFail / configure error |
| 2 | usage / unknown command |

JSON on stdout; stderr notes only (never keys, agent_address, balances, calldata, RPC URLs). Body limit 1 MiB.

## Product skill

Agent-facing guide: [`skills/usdc-liquidity/`](../../skills/usdc-liquidity/).
