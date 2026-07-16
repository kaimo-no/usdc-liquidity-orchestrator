# pkg/types/

Agent-facing wire shapes for plan I/O (JSON tags only).

## Types

- `Required` — merchant claim (`pay_to` untrusted; role merchant)
- `Orchestration` — optional target/source allowlist / prefer_rail
- `Fee` — optional kaimo orchestration fee (x402 settle; not fund-rail dest)
- `Plan` / `PlanStep` — dry/execute plan envelope
- `Inventory` / `Balance` — client-asserted balances
- `PlanRequest` / `PlanResponse` / `APIError` — HTTP body shapes
- `ChainInfo` / `ChainsResponse` — `GET /v1/chains` discovery

## Invariants

- No custom `MarshalJSON`
- `pay_to_role` is `"merchant"` for Required; fund steps use `agent_self` in PlanStep
