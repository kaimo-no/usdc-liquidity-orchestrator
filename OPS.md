# OPS

Pre-production / hackathon ops checklist for **usdc-liquidity-orchestrator**.

## Plan-only mode (current default)

- No secrets required
- `cmd/server` is stateless; safe to run ephemerally
- Inventory is request-scoped — do not persist balances server-side
- Optional local `.env` (gitignored): copy from `.env.example`; `internal/envfile` never logs values
- `cmd/server` and `cmd/demo` both load `.env` at startup (does not override existing process env)
- Demo payment scenario (`PAYMENT_*`, `SOURCE_AMOUNT_*`, `USDC_SCALE_FACTOR`) is **CLI-only** — not applied to HTTP `/v1/plan`

## Optional testnet Gateway execute

**Dangerous:** signs and broadcasts real txs. Dual-gated and loopback-only.

| Requirement | Detail |
|---|---|
| `ENABLE_TESTNET_EXECUTE=1` | Explicit opt-in |
| `AGENT_PRIVATE_KEY` | Hex ECDSA matching inventory `agent_address` |
| Named RPCs | `RPC_URL_BASE_SEPOLIA`, `RPC_URL_ARBITRUM_SEPOLIA`, `RPC_URL_ARC_TESTNET` (EVM execute) |
| `RPC_URL_SOLANA_DEVNET` | Placeholder only — not used by EVM execute today |
| Alternates | `RPC_URLS_JSON` or `RPC_URL_eip155_*` |
| `LISTEN_ADDR` | Must be loopback (`127.0.0.1:8088`, `[::1]:8088`, `localhost:8088`) — bare `:8088` refused |
| `GATEWAY_API_BASE` | Optional; default `https://gateway-api-testnet.circle.com` |
| `GATEWAY_MAX_FEE_ATOMIC` | Optional burn-intent maxFee (default `2010000`) |

Supported actions:

| Action | Behaviour |
|---|---|
| `circle_gateway_consolidate` | Deposit steps only; signs **re-derived** prepare calls |
| `circle_gateway_deposit_withdraw` | Deposits, then EIP-712 burn intents + `POST /v1/transfer` + `gatewayMint` on dest |
| `circle_gateway_withdraw` | Burn intent + transfer + mint (no deposits) |

Burn/mint `destinationRecipient` is always the agent (never merchant `pay_to`). Partial failures return hashes + `executed=false`. After deposits, transfer API is retried (default 5×); Base/Arb need ~13–19m confirmations before Gateway balances/attestations (Circle docs). Withdraw-only with empty `from_chain` allocates burns from live `POST /v1/balances`, skipping domains at/below maxFee floor (~2.01 USDC default).

```bash
# Example (local only — never commit key):
export ENABLE_TESTNET_EXECUTE=1
export LISTEN_ADDR=127.0.0.1:8088
export AGENT_PRIVATE_KEY=0x…   # testnet throwaway
export RPC_URL_BASE_SEPOLIA=https://sepolia.base.org
# export RPC_URL_ARBITRUM_SEPOLIA=…
# export RPC_URL_ARC_TESTNET=…
# export RPC_URL_SOLANA_DEVNET=https://api.devnet.solana.com   # ignored by EVM execute
go run ./cmd/server
```

See [`.env.example`](./.env.example) for all placeholders.

## Live inventory (demo only)

`cmd/demo` scenario plan may call `internal/inventory.Load` when `AGENT_ADDRESS` / key and testnet RPCs are set: ERC-20 `balanceOf` USDC + optional Gateway `POST /v1/balances` (domains 3, 6, 26). Hard-coded `SOURCE_AMOUNT_*` reals still drive funding; each source real must not exceed live native on that chain (`insufficient_liquidity`). Dry plans still stamp `inventory_unverified=true`. Never log balances or RPC URLs.

Docker must **not** enable execute by default. Do not pass keys into images.

## Hard fail-closed rules

1. Never configure live spend keys in CI or public logs  
2. Keep keys out of git; document env vars in root `CLAUDE.md` + `README.md` / `OPS.md`  
3. Do not log full inventory, private keys, prepare calldata, or RPC URLs — covers `POST /v1/plan` and `POST /v1/consolidate`  
4. Merchant `pay_to` is untrusted discovery data — prepare never transfers to it  
5. Gateway Wallet addresses are non-secret constants (testnet/mainnet); not credentials  
6. Mainnet execute is out of scope — RPC map construction refuses non-testnet chains  

## HTTP

| Var | Default | Notes |
|---|---|---|
| `LISTEN_ADDR` | `:8088` | Bind address (loopback required for execute) |
| `MAX_AMOUNT_ATOMIC` | unset | Optional Guard max amount (atomic units) for execute/plan Guard |

Docker/compose publish the same port (`8088:8088`). Probe externally (`curl …/healthz`); the distroless image has no shell for in-container healthchecks.

## Security scans (local)

```bash
gitleaks detect --source . --verbose
bash scripts/check-no-live-secrets.sh
govulncheck ./...
```

## Relationship to kaimo-go

Private monorepo may embed or HTTP-call this service for World B native_x402 liquidity prep. This repo must remain import-clean of kaimo-go (public hackathon surface).
