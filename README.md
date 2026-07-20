# USDC Liquidity Orchestrator

**[kaimo-no](https://github.com/kaimo-no)** · public component for agentic commerce

**Non-custodial multi-chain USDC rebalancing.** Agents set a **target chain** and optional **source** allowlist; this library + microservice **plans** how to fund the agent’s own wallet on the target via **Circle Gateway** (preferred unified balance) and **CCTP Fast Transfer** (fallback) — without product custody and without platform MoR. Arc Testnet is first-class for Circle/Arc hackathon corridors.

| | |
|---|---|
| Org | [`kaimo-no`](https://github.com/kaimo-no) |
| Module | `github.com/kaimo-no/usdc-liquidity-orchestrator` |
| License | Apache-2.0 |
| Status | L0–L1 shipped · L2 execute fail-closed stub |

Docs for agents: **[`CLAUDE.md`](./CLAUDE.md)** · **[`AGENTS.md`](./AGENTS.md)** · **[`SETUP.md`](./SETUP.md)** · **[`OPS.md`](./OPS.md)**

---

## Problem

| Actor | Reality |
|---|---|
| Merchant | Pay **42 USDC** to `payTo` on **Base** |
| Agent | 30 USDC on Arbitrum + 20 USDC on Base |
| Bad path | Fail, or platform takes custody (MoR) |

**This component:** plan deposit/withdraw (shortfall **22** only) to **agent_self on Base**, then agent signs merchant Payment-Signature separately.

---

## Quick start

```bash
git clone https://github.com/kaimo-no/usdc-liquidity-orchestrator.git
cd usdc-liquidity-orchestrator
go test ./...
go run ./cmd/demo
go run ./cmd/server   # :8088
bash examples/curl.sh

# or containerized:
docker compose up --build
```

### Library

```go
import "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"

plan, err := liquidity.PlanLiquidity(req, inv, nil)
wire := liquidity.PlanToWire(plan)
```

### HTTP

`POST /v1/plan` — see [`examples/plan.json`](./examples/plan.json) (Arc Testnet + Gateway)  
`POST /v1/consolidate` — multi-chain full-balance Gateway deposits + unsigned `prepare_calls` ([`examples/consolidate-testnet.json`](./examples/consolidate-testnet.json))  
`GET /v1/chains` — registered corridors (CAIP-2, USDC, Gateway domain, testnet, gateway_wallet)  
`GET /healthz`

---

## Architecture

```text
pkg/liquidity  pure planner + Guard + UnconfiguredExecutor
pkg/types      wire JSON
pkg/errors     stable codes
cmd/server     thin HTTP microservice
cmd/demo       worked example
```

Plan preference: `noop` → `circle_gateway_withdraw` → `circle_gateway_deposit_withdraw` → `cctp_fast` → `insufficient` / `corridor_unsupported` (override with `orchestration.prefer_rail`).

`POST /v1/consolidate` plans action `circle_gateway_consolidate` (full native USDC → Gateway Wallet deposits; no merchant claim). Deposit steps may include advisory unsigned `prepare_calls` (approve + deposit).

Optional `fee_bps` + `fee_recipient` attach a kaimo orchestration fee settled **after** prepare (e.g. x402) — never a prepare fund-move destination.

### Hard invariants

1. Fund-move recipient = **agent_self** only  
2. Empty `pay_to` → fail closed on withdraw/cctp (deposit-only consolidate may omit pay_to)  
3. Amount override only when probe amount missing  
4. Dry plan: `executed=false`, `dry_run=true`  
5. Execute stub always errors  
6. Atomic `decimal.Decimal` (no float64, no GBP Round(2))  
7. **Shortfall-only** rebalance (plan path); consolidate deposits full native balances  
8. Rails named `circle_gateway_*` / `cctp_fast` (never bare `gateway`)  
9. `agent_address == pay_to` is **refused** on fund-moving plans (anti–confused-deputy: prepare must not look like a merchant payout)  
10. Optional fee is **plan.fee envelope only** — never an `orchestrator_fee` step (naive agents must not auto-execute fee as a transfer)

---

## Agent workflow (same as kaimo-go)

| Skill | Purpose |
|---|---|
| `/team_review` | Design gate |
| `/ship_feature` | Design → implement → CI loop → PR |
| `/ci_local` | One local CI + dual OpenCode |
| `/ci_review_loop` | 2× review + fix → PR |
| `/create_pr` | Commit + open PR |
| `/session_cleanup` | Temp file cleanup |

Agents live under [`.claude/agents/`](./.claude/agents/). Skills under [`.claude/skills/`](./.claude/skills/).

---

## CI

| Workflow | Role |
|---|---|
| `ci.yml` | gofmt, test layout, golangci-lint (gosec), race tests, 60% coverage, build |
| `govulncheck.yml` | Reachable vulns |
| `gitleaks.yml` | Secret scan + live-secret patterns |
| `actionlint.yml` | Workflow lint |
| Dependabot | gomod + github-actions weekly |

PR rules: [`.github/pr-review-instructions.md`](./.github/pr-review-instructions.md)

---

## Package map

| Path | Role | Detail |
|---|---|---|
| `pkg/liquidity/` | Planner | [CLAUDE.md](./pkg/liquidity/CLAUDE.md) |
| `pkg/types/` | Wire | [CLAUDE.md](./pkg/types/CLAUDE.md) |
| `pkg/errors/` | Codes | [CLAUDE.md](./pkg/errors/CLAUDE.md) |
| `cmd/server/` | HTTP | [CLAUDE.md](./cmd/server/CLAUDE.md) |
| `cmd/demo/` | CLI demo | — |

---

## Related

- Private monorepo: [kaimo-no/kaimo-go](https://github.com/kaimo-no/kaimo-go) (commerce router; optional consumer)  
- This repo has **zero** dependency on kaimo-go  

## License

Apache-2.0 — see [LICENSE](./LICENSE).

Circle, USDC, Gateway, CCTP, and Arc are trademarks of their respective owners. Not affiliated with Circle.
