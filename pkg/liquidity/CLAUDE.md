# pkg/liquidity/

Pure multi-chain USDC liquidity planner + fail-closed execute stub.
General orchestrator: agent pins **target** + optional **source** allowlist; Gateway preferred.
Testnet-ready: multi-chain **consolidate** deposits + unsigned **prepare_calls**.

## Surface

| Symbol | Role |
|---|---|
| `RequiredFromWire` | Build `Required` from merchant-claim wire; amount override only when amount missing |
| `PlanLiquidity` | Dry plan (nil orchestration/fee) → delegates to `PlanOrchestration` |
| `PlanOrchestration` | Same + `Orchestration` (target/sources/rail) + optional `FeeConfig` |
| `PlanConsolidate` | Full-balance Gateway deposits (no pay_to/fee); action `circle_gateway_consolidate` |
| `ListChains` / `LookupChain` / `GatewayWalletAddress` | Registry + testnet/mainnet Gateway Wallet |
| `PlanToWire` | Agent-facing `types.Plan` stamps (+ fee + prepare_calls) |
| `InventoryFromWire` / `OrchestrationFromWire` / `FeeConfigFromWire` | Wire helpers |
| `Guard` | MaxAmountAtomic + AllowedAgentAddresses; dual predicates (merchant claim vs fund-moving) |
| `CheckAgent` | Agent allowlist without merchant Required |
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
- Dest-native shortfall uses same-chain USDC match (symbol `"USDC"` ↔ registry contract)
- `CheckPlan` dual predicates:
  - **requiresMerchantClaim**: any withdraw / cctp_burn / cctp_mint → empty pay_to refuse
  - **fund-moving** (incl. deposit): agent_self, recipient==agent, MaxAmountAtomic, kind allowlist
  - deposit-only (consolidate) may have empty pay_to
- Deposit steps attach advisory `prepare_calls` (approve+deposit); pure-Go ABI; not re-checked by Guard
- Atomic `decimal.Decimal` without `Round(2)`

## Known limitations

- No live Circle SDK execute (Gateway Wallet addresses are non-secret constants only)
- Solana corridors unsupported
- Inventory balances are client-asserted (`inventory_unverified=true`)
- `prepare_calls` are server-generated advisory for agents (unsigned)

## Tests

`pkg/liquidity/tests/` — black-box `package liquidity_test`.
