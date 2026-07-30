# pkg/execonchain/

Optional **testnet-only** Circle Gateway executor (go-ethereum). Implements `liquidity.Executor`.

## Surface

| Symbol | Role |
|---|---|
| `Config` | PrivateKeyHex, RPCs, Guard, WaitTimeout, Dial, GatewayAPI, HTTPDo, MaxFeeAtomic, transfer retry, SaltFn |
| `NewDepositExecutor` | Parse key; refuse empty/mainnet RPC keys |
| `DepositExecutor` | `Execute` for consolidate, deposit_withdraw, withdraw |
| `ChainClient` | Minimal RPC surface (mockable in tests) |

## Supported actions

| Action | Flow |
|---|---|
| `circle_gateway_consolidate` | Re-derived deposit prepare_calls only |
| `circle_gateway_deposit_withdraw` | Deposits → EIP-712 burn intent(s) → `POST /v1/transfer` → `gatewayMint` on dest |
| `circle_gateway_withdraw` | Burn + transfer + mint; empty `from_chain` → allocate from live Gateway `/v1/balances` (multi-domain) |

## Invariants

- **Testnet only** — every step chain + every RPC map key must be GatewayOK testnet
- **Sign re-derived deposit calls only** — `BuildDepositPrepareCalls`; never client calldata
- Optional exact-match if client already supplied `prepare_calls`
- **Burn/mint destinationRecipient = agent_self only** — never merchant `pay_to`
- Key address must `EqualFold` plan `AgentAddress`
- `eth_chainId` must match `eip155:N` before first tx on a chain
- Transfer API retries (default 5×) after deposits — Base/Arb need ~13–19m finality (Circle docs)
- Burn `maxFee` capped below burn `value` (default 2.01 USDC must not exceed 1 USDC source burns)
- Partial fail: return hashes so far + error (`executed` stamped false by HTTP)
- Fixed `pkg/errors` codes — no raw RPC / Gateway HTTP bodies to callers
- No mainnet execute; no CCTP

## Env

| Var | Role |
|---|---|
| `GATEWAY_API_BASE` | Override transfer/balances base (default testnet public API) |
| `GATEWAY_MAX_FEE_ATOMIC` | Burn-intent maxFee (default `2010000`) |

## Tests

`pkg/execonchain/tests/` — black-box `package execonchain_test` with mock `ChainClient` + mock HTTP. No network in default `go test`.
