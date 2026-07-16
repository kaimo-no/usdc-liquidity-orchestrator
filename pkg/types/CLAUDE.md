# pkg/types/

Agent-facing wire shapes for plan I/O (JSON tags only).

## Types

- `Required` — merchant claim (`pay_to` untrusted; role merchant)
- `Plan` / `PlanStep` — dry/execute plan envelope
- `Inventory` / `Balance` — client-asserted balances
- `PlanRequest` / `PlanResponse` / `APIError` — HTTP body shapes

## Invariants

- No custom `MarshalJSON`
- `pay_to_role` is `"merchant"` for Required; fund steps use `agent_self` in PlanStep
