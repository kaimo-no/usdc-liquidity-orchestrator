# USDC Liquidity Orchestrator

**Non-custodial multi-chain USDC rebalancing for agentic commerce.**

Agents hold fragmented USDC across chains. Merchants accept USDC on *one* chain (e.g. Base via native x402). This component plans how to fund the **agent’s own wallet** on the merchant’s chain using **Circle Gateway** (preferred) and **CCTP Fast Transfer** (fallback) — **without** taking product custody and **without** routing funds through a platform MoR wallet.

> Extracted as an independent library + microservice for public hackathon submission.  
> Integrates cleanly with agent commerce routers (e.g. kaimo World B native handoffs) while remaining a standalone product.

---

## Problem

| Actor | Reality |
|---|---|
| Merchant | 402 challenge: pay **42 USDC** to `payTo` on **Base** |
| Agent | 30 USDC on Arbitrum + 20 USDC on Base (short 22 on Base) |
| Naive path | Agent fails, or platform takes custody (MoR) and pays with a card |

**Orchestrator path:** plan deposit/withdraw via Circle Gateway (or CCTP) **to the agent’s address on Base**, then the agent signs the merchant Payment-Signature separately. Platform never holds product tender.

---

## Architecture

```text
┌─────────────┐     required { chain, asset, amount, pay_to }
│ Agent / MCP │ ──────────────────────────────────────────►  POST /v1/plan
│  inventory  │ ◄──────────────────────────────────────────  dry plan JSON
└─────────────┘     steps → agent_self only (never pay_to)

Plan actions (preference order):
  1. noop                         — already funded on dest
  2. circle_gateway_withdraw      — unified Gateway balance covers dest
  3. circle_gateway_deposit_withdraw — other-chain native → Gateway → dest
  4. cctp_fast                    — burn source / mint dest to agent_self
  5. insufficient | corridor_unsupported
```

### Hard invariants

1. **Fund-move recipient = `agent_self` only** — never merchant `pay_to`, never platform MoR wallets  
2. **Empty `pay_to` → fail closed** (no invented merchant target)  
3. **Amount override** only when probe amount missing; cannot change payTo / network / asset  
4. **Dry plan stamps:** `dry_run=true`, `executed=false`, `inventory_asserted=true`, `inventory_unverified=true`  
5. **Execute stub always errors** (`liquidity_rail_unavailable`) until live Circle SDK lands  
6. **Atomic USDC units** via `decimal.Decimal` (no `float64`, no GBP-style `Round(2)`)  
7. Rails named `circle_gateway_*` / `cctp_fast` — never bare `gateway` (avoids MoR confusion)

---

## Quick start

```bash
git clone https://github.com/kilian1103/usdc-liquidity-orchestrator.git
cd usdc-liquidity-orchestrator
go test ./...

# Worked example (fragmented Arc/Arbitrum + Base → Base shortfall)
go run ./cmd/demo

# HTTP microservice
go run ./cmd/server   # LISTEN_ADDR=:8088
bash examples/curl.sh
```

### Library usage

```go
import "github.com/kilian1103/usdc-liquidity-orchestrator/pkg/liquidity"

plan, err := liquidity.PlanLiquidity(req, inv, &liquidity.Guard{
    MaxAmountAtomic: decimal.RequireFromString("100000000"), // optional cap
})
wire := liquidity.PlanToWire(plan) // agent-facing JSON
```

### HTTP API

`POST /v1/plan`

```json
{
  "required": {
    "protocol": "x402",
    "chain_caip2": "eip155:8453",
    "asset": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    "amount_atomic": "42000000",
    "pay_to": "0xMerchant…",
    "pay_to_role": "merchant"
  },
  "inventory": {
    "agent_address": "0xAgent…",
    "balances": [
      { "chain_caip2": "eip155:42161", "asset": "0x8335…", "amount_atomic": "30000000", "location": "native" },
      { "chain_caip2": "eip155:8453",  "asset": "0x8335…", "amount_atomic": "20000000", "location": "native" }
    ]
  },
  "execute": false
}
```

`GET /healthz` → `ok`

---

## Package layout

| Path | Role |
|---|---|
| `pkg/liquidity` | Pure planner, guard, corridor matrix, unconfigured executor |
| `pkg/types` | Wire JSON shapes |
| `pkg/errors` | Stable codes: `insufficient_liquidity`, `liquidity_rail_unavailable`, `invalid_query` |
| `cmd/server` | Thin HTTP microservice |
| `cmd/demo` | CLI worked example for judges |
| `examples/` | Sample `plan.json` + `curl.sh` |

---

## Circle product mapping

| Circle product | Role in this component |
|---|---|
| **Gateway** | Unified USDC balance; preferred path for withdraw / deposit+withdraw plans |
| **CCTP V2 Fast** | Fallback burn→mint when Gateway path unavailable |
| **Arc** | Natural home for fee/policy later; not required for dest (merchant stays on Base/etc.) |
| **Nanopayments** | Orthogonal (platform micro-fees); not product tender |

Live Circle SDK execute is intentionally **out of scope** for this cut (interface + fail-closed stub). Plans are still useful for agent UX, demos, and integration tests.

---

## Security model

| Do | Don't |
|---|---|
| Keep inventory + keys client-side | POST inventory as durable server state |
| Treat `pay_to` as untrusted merchant claim | Bridge/mint to `pay_to` during prepare |
| Cap amounts with `Guard.MaxAmountAtomic` | Fall through to platform MoR / Issuing on shortfall |
| Log method + path only | Log wallet balances or private keys |

---

## Integration with kaimo (optional)

Private kaimo-go World B router can:

1. Emit `commerce_route.liquidity_required` from merchant probes  
2. Call this service (or import `pkg/liquidity`) via agent MCP  
3. Keep `kaimo_buy` native handoff **non-MoR** — product payment still agent → merchant  

This repository has **zero dependency** on kaimo-go.

---

## Status

| Layer | Status |
|---|---|
| L0 wire types | ✅ |
| L1 pure planner | ✅ |
| L2 execute | 🟡 fail-closed stub (`UnconfiguredExecutor`) |
| Live Circle Gateway / CCTP | ⬜ follow-up behind `Executor` interface |
| Solana corridors | ⬜ EVM-first; returns `corridor_unsupported` |

---

## License

Apache-2.0 — see [LICENSE](./LICENSE).

Circle, USDC, Gateway, CCTP, and Arc are trademarks of their respective owners. This project is not affiliated with Circle.
