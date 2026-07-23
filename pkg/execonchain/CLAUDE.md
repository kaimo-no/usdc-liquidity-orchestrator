# pkg/execonchain/

Optional **testnet-only** Circle Gateway deposit executor (go-ethereum). Implements `liquidity.Executor`.

## Surface

| Symbol | Role |
|---|---|
| `Config` | PrivateKeyHex, RPCs (CAIP-2→URL), Guard, WaitTimeout, Dial |
| `NewDepositExecutor` | Parse key; refuse empty/mainnet RPC keys |
| `DepositExecutor` | `Execute` for `circle_gateway_consolidate` deposits only |
| `ChainClient` | Minimal RPC surface (mockable in tests) |
| `IsTestnetExecutableChain` | Re-export / alias via `liquidity.IsTestnetExecutableChain` |

## Invariants

- **Testnet only** — every step chain + every RPC map key must be GatewayOK testnet
- **Action allowlist** — `circle_gateway_consolidate` only; all steps `circle_gateway_deposit`
- **Sign re-derived calls only** — `BuildDepositPrepareCalls`; never client calldata
- Optional exact-match if client already supplied `prepare_calls`
- Key address must `EqualFold` plan `AgentAddress`
- `eth_chainId` must match `eip155:N` before first tx on a chain
- Sequential approve → deposit; wait receipt `status==1`
- Partial fail: return hashes so far + error (`executed` stamped false by HTTP)
- Fixed `pkg/errors` codes/messages — no raw RPC strings to wire callers (HTTP sanitizes)
- No mainnet execute; no withdraw/CCTP

## Tests

`pkg/execonchain/tests/` — black-box `package execonchain_test` with mock `ChainClient`. No network in default `go test`.
