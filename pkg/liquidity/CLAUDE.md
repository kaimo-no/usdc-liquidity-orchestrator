# pkg/liquidity/

Pure multi-chain USDC liquidity planner + fail-closed execute stub.
General orchestrator: agent pins **target** + optional **source** allowlist; Gateway preferred.
Testnet-ready: multi-chain **consolidate** deposits + unsigned **prepare_calls**.

## Surface

| Symbol | Role |
|---|---|
| `RequiredFromWire` | Build `Required` from merchant-claim wire; amount override only when amount missing |
| `PlanLiquidity` | Dry plan (nil orchestration/fee) → delegates to `PlanOrchestration` |
| `PlanOrchestration` | Shortfall-only rebalance + `Orchestration` + optional `FeeConfig` |
| `PlanPaymentFunding` | Scenario Phase A multi-source deposits only (not shortfall; no withdraw) |
| `PlanConsolidate` | Full-balance Gateway deposits (no pay_to/fee); action `circle_gateway_consolidate` |
| `PlanGatewayDeposit` | Fixed-N single-source Gateway deposit (wraps `PlanGatewayDeposits`; no pay_to/fee); hard underfund error |
| `PlanGatewayDeposits` | Fixed multi-source Gateway deposits (no pay_to/fee; no payment-sum rule); merge duplicate chains; hard underfund per normalized source; MaxAmountAtomic is per-step |
| `PlanSelfRebalance` | Land N on dest agent_self shortfall-only (empty pay_to; `selfRebalance`) |
| `ListChains` / `LookupChain` / `LookupByGatewayDomain` / `ResolveChainRef` / `GatewayWalletAddress` | Registry + domain/name/CAIP-2 resolve + Gateway Wallet |
| `PlanToWire` | Agent-facing `types.Plan` stamps (+ fee + prepare_calls; optional logical/scale) |
| `InventoryToWire` / `InventoryFromWire` / `OrchestrationFromWire` / `FeeConfigFromWire` | Wire helpers |
| `Guard` | MaxAmountAtomic + AllowedAgentAddresses; dual predicates (merchant claim vs fund-moving) |
| `CheckAgent` | Agent allowlist without merchant Required |
| `UnconfiguredExecutor` | Always errors after `CheckPlan` |
| `Executor` | Interface; live adapter is `pkg/execonchain.DepositExecutor` (testnet deposits) |
| `BuildDepositPrepareCalls` | Pure re-derive of approve+deposit calls (execute must sign these only) |
| `IsTestnetExecutableChain` | Registered GatewayOK testnet corridor |

## Invariants

- Fund steps: `RecipientRole=agent_self`, `Recipient=AgentAddress`
- Never step.Recipient = Required.PayTo; never platform MoR as fund dest
- Shortfall = required − dest native; steps move shortfall only (`PlanOrchestration`)
- **`PlanPaymentFunding`**: Phase A hard-coded deposits only — no withdraw; wait finality then Phase B separately
- Bare location `"gateway"` → ignored (invalid); use `circle_gateway`
- Solana / unknown dest → `corridor_unsupported` (EVM registry-first)
- Fee (`orchestrator` / `settle_via=x402`) is **plan.fee only** — not a fund rail recipient; `orchestrator_fee` is **not** a valid step kind (injected fee steps → unknown kind refuse)
- `FeeConfig` is bps + recipient (+ optional settle_via); fee chain/asset always follow plan `Required`
- `agent_address == pay_to` refused on fund-moving plans (anti–confused-deputy)
- Inventory amounts must be positive (zero/negative → `invalid_query`)
- Dest-native shortfall uses same-chain USDC match (symbol `"USDC"` ↔ registry contract)
- `CheckPlan`: all fund steps agent_self; empty pay_to OK for withdraw/cctp/deposit; residual pay_to never fund dest
- Deposit steps attach advisory `prepare_calls` (approve+deposit); pure-Go ABI; not re-checked by Guard
- Atomic `decimal.Decimal` without `Round(2)`

## Known limitations

- Default execute is fail-closed; optional live path is testnet Gateway deposit + burn/mint (`pkg/execonchain`)
- No CCTP / mainnet execute in this package
- Solana corridors unsupported
- Inventory balances are client-asserted unless demo loads via `internal/inventory` (still stamps `inventory_unverified=true`)
- `prepare_calls` are server-generated advisory for agents (unsigned); live execute re-derives via `BuildDepositPrepareCalls`

## Tests

`pkg/liquidity/tests/` — black-box `package liquidity_test`.
