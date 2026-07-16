# pkg/liquidity/

Pure multi-chain USDC liquidity planner + fail-closed execute stub.
General orchestrator: agent pins **target** + optional **source** allowlist; Gateway preferred.

## Surface

| Symbol | Role |
|---|---|
| `RequiredFromWire` | Build `Required` from merchant-claim wire; amount override only when amount missing |
| `PlanLiquidity` | Dry plan (nil orchestration/fee) → delegates to `PlanOrchestration` |
| `PlanOrchestration` | Same + `Orchestration` (target/sources/rail) + optional `FeeConfig` |
| `ListChains` / `LookupChain` | Registry discovery (Arc Testnet, Base, Arb, sepolias…) |
| `PlanToWire` | Agent-facing `types.Plan` stamps (+ fee) |
| `InventoryFromWire` / `OrchestrationFromWire` / `FeeConfigFromWire` | Wire helpers |
| `Guard` | MaxAmountAtomic + AllowedAgentAddresses; fee recipient ≠ pay_to; separate from x402 MoR |
| `UnconfiguredExecutor` | Always errors after `CheckPlan` |
| `Executor` | Interface for future Circle Gateway / CCTP live SDK |

## Invariants

- Fund steps: `RecipientRole=agent_self`, `Recipient=AgentAddress`
- Never step.Recipient = Required.PayTo; never platform MoR as fund dest
- Shortfall = required − dest native; steps move shortfall only
- Bare location `"gateway"` → ignored (invalid); use `circle_gateway`
- Solana / unknown dest → `corridor_unsupported` (EVM registry-first)
- Fee (`orchestrator` / `settle_via=x402`) is **plan.fee only** — not a fund rail recipient and **not** a step in `steps[]`
- `agent_address == pay_to` refused on fund-moving plans (anti–confused-deputy)
- Inventory amounts must be positive (zero/negative → `invalid_query`)
- Atomic `decimal.Decimal` without `Round(2)`

## Known limitations

- No live Circle SDK execute (testnet Gateway addresses are non-secret constants only)
- Solana corridors unsupported
- Inventory balances are client-asserted (`inventory_unverified=true`)

## Tests

`pkg/liquidity/tests/` — black-box `package liquidity_test`.
