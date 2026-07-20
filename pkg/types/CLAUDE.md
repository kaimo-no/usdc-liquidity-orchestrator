# pkg/types/

Agent-facing wire shapes for plan I/O (JSON tags only).

## Types

- `Required` — merchant claim (`pay_to` untrusted; role merchant)
- `Orchestration` — optional target/source allowlist / prefer_rail
- `Fee` — optional kaimo orchestration fee (x402 settle; not fund-rail dest)
- `Plan` / `PlanStep` — dry/execute plan envelope
- `PrepareCall` — unsigned EVM call on deposit steps (`prepare_calls`)
- `Inventory` / `Balance` — client-asserted balances
- `PlanRequest` / `ConsolidateRequest` / `PlanResponse` / `APIError` — HTTP body shapes
- `ChainInfo` / `ChainsResponse` — `GET /v1/chains` discovery (`testnet`, `gateway_wallet`)

## Invariants

- No custom `MarshalJSON`
- `pay_to_role` is `"merchant"` for Required; fund steps use `agent_self` in PlanStep
- Consolidate responses omit `required` / `amount_source` (no merchant claim)
