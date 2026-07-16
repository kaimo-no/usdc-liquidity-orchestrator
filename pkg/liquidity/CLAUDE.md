# pkg/liquidity/

Pure multi-chain USDC liquidity planner + fail-closed execute stub.

## Surface

| Symbol | Role |
|---|---|
| `RequiredFromWire` | Build `Required` from merchant-claim wire; amount override only when amount missing |
| `PlanLiquidity` | Dry plan: noop → circle_gateway_withdraw → deposit_withdraw → cctp_fast → insufficient |
| `PlanToWire` | Agent-facing `types.Plan` stamps |
| `InventoryFromWire` | Parse inventory atomic amounts |
| `Guard` | MaxAmountAtomic + AllowedAgentAddresses; separate from any x402 MoR guard |
| `UnconfiguredExecutor` | Always errors after `CheckPlan` |
| `Executor` | Interface for future Circle Gateway / CCTP live SDK |

## Invariants

- Fund steps: `RecipientRole=agent_self`, `Recipient=AgentAddress`
- Never step.Recipient = Required.PayTo
- Shortfall = required − dest native; steps move shortfall only
- Bare location `"gateway"` → ignored (invalid)
- Solana dest → `corridor_unsupported` (EVM-first matrix)
- Atomic `decimal.Decimal` without `Round(2)`

## Known limitations

- No live Circle SDK execute
- Solana corridors unsupported
- Inventory balances are client-asserted (`inventory_unverified=true`)

## Tests

`pkg/liquidity/tests/` — black-box `package liquidity_test`.
