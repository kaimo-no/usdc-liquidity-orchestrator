# pkg/types/

Agent-facing wire shapes for plan I/O (JSON tags only).

## Types

- `Required` — merchant claim (`pay_to` untrusted; role merchant); `amount_atomic` is **real**
- Optional scenario stamps on `Required` / `PlanStep`: `amount_logical_atomic`, `scale_factor` (real = floor(logical / scale))
- `Orchestration` — optional target/source allowlist / prefer_rail
- `Fee` — optional kaimo orchestration fee (x402 settle; not fund-rail dest)
- `Plan` / `PlanStep` — dry/execute plan envelope
- `PrepareCall` — unsigned EVM call on deposit steps (`prepare_calls`)
- `Inventory` / `Balance` — client-asserted balances (also bare success body for `POST /v1/inventory`)
- `InventoryRequest` — `POST /v1/inventory` input (`agent_address` only)
- `PlanRequest` / `ConsolidateRequest` / `PaymentFundingRequest` / `FundingSource` — HTTP inputs
- `DepositRequest` / `MoveRequest` — CLI JSON only (no HTTP this cut); fixed-N deposit (single fields XOR `sources[]`) + self-land move
- `PlanResponse` / `APIError` / `ExecuteReceipt` — HTTP outputs (`APIError` is also bare error body for inventory)
- `ExecuteReceipt` — optional `tx_hashes` on successful/partial execute (no notes on wire)
- `ChainInfo` / `ChainsResponse` — `GET /v1/chains` discovery (`testnet`, `gateway_wallet`)

## Invariants

- No custom `MarshalJSON`
- `pay_to` / `pay_to_role` are `omitempty` (land Required for move has neither; merchant plans still require pay_to in validation)
- `pay_to_role` is `"merchant"` for merchant Required; fund steps use `agent_self` in PlanStep
- Consolidate / deposit responses omit `required` / `amount_source` (no merchant claim)
- Move self-land emits `required` with land N + dest and `amount_source=self` (no pay_to)
- `amount_atomic` is always real on-chain units; logical/scale are optional metadata only
