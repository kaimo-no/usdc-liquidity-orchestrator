# pkg/types/

Agent-facing wire shapes for plan I/O (JSON tags only).

## Types

- `Required` — merchant claim (`pay_to` untrusted; role merchant); `amount_atomic` is **real**
- Optional scenario stamps on `Required` / `PlanStep`: `amount_logical_atomic`, `scale_factor` (real = floor(logical / scale))
- `Orchestration` — optional target/source allowlist / prefer_rail
- `Fee` — optional kaimo orchestration fee (x402 settle; not fund-rail dest)
- `Plan` / `PlanStep` — dry/execute plan envelope
- `PrepareCall` — unsigned EVM call on deposit steps (`prepare_calls`)
- `Inventory` / `Balance` — client-asserted balances
- `PlanRequest` / `ConsolidateRequest` / `PaymentFundingRequest` / `FundingSource` — HTTP inputs
- `PlanResponse` / `APIError` / `ExecuteReceipt` — HTTP outputs
- `ExecuteReceipt` — optional `tx_hashes` on successful/partial execute (no notes on wire)
- `ChainInfo` / `ChainsResponse` — `GET /v1/chains` discovery (`testnet`, `gateway_wallet`)

## Invariants

- No custom `MarshalJSON`
- `pay_to_role` is `"merchant"` for Required; fund steps use `agent_self` in PlanStep
- Consolidate responses omit `required` / `amount_source` (no merchant claim)
- `amount_atomic` is always real on-chain units; logical/scale are optional metadata only
