# USDC Liquidity Orchestrator

Product skill for **non-custodial multi-chain USDC rebalancing** (Circle Gateway preferred, CCTP fallback).

Module: `github.com/kaimo-no/usdc-liquidity-orchestrator`

## Surfaces (equal peers)

| Surface | Entry | Use when |
|---|---|---|
| **Library** | `pkg/liquidity` | Embed in another Go service |
| **CLI** | `cmd/usdc-liq` (`usdc-liq`) | Scripts, agents, local ops |
| **HTTP** | `cmd/server` | UI + microservice |
| **Skill** | this directory | Agent instructions |

HTTP parity (shared `internal/planio` stamps): `plan` · `consolidate` · `payment-funding` · `chains`  
CLI extras: `inventory` · `demo` · `version`

## Default: dry plan

Always prefer **dry** first (`execute=false` / no `--execute`):

```bash
go run ./cmd/usdc-liq plan -f examples/plan.json
go run ./cmd/usdc-liq consolidate -f examples/consolidate-testnet.json
go run ./cmd/usdc-liq chains
```

```bash
curl -sS -X POST localhost:8088/v1/plan -H 'Content-Type: application/json' -d @examples/plan.json
```

Dry responses stamp `dry_run=true`, `executed=false`, `inventory_asserted=true`, `inventory_unverified=true`.

## Execute is fail-closed

- Default executor is `UnconfiguredExecutor` (never succeeds).
- Live testnet execute requires dual gate: `ENABLE_TESTNET_EXECUTE=1` + `AGENT_PRIVATE_KEY` + testnet RPCs.
- HTTP also requires **loopback** `LISTEN_ADDR`. CLI does not require loopback.
- Execute re-derives deposit prepare calls — **never trust client `prepare_calls`**.
- Fund-move recipients are always **agent_self** — never merchant `pay_to`.

See [references/invariants.md](./references/invariants.md), [references/cli.md](./references/cli.md), [references/http.md](./references/http.md).

## Privacy

Do not log or echo: private keys, agent addresses in failure notes, balances, prepare calldata, RPC URLs. Inventory is request-scoped; no durable buyer ledger.
