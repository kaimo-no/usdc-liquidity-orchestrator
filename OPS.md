# OPS

Pre-production / hackathon ops checklist for **usdc-liquidity-orchestrator**.

## Plan-only mode (current default)

- No secrets required
- `cmd/server` is stateless; safe to run ephemerally
- Inventory is request-scoped — do not persist balances server-side

## Optional testnet deposit execute

**Dangerous:** signs and broadcasts real txs. Dual-gated and loopback-only.

| Requirement | Detail |
|---|---|
| `ENABLE_TESTNET_EXECUTE=1` | Explicit opt-in |
| `AGENT_PRIVATE_KEY` | Hex ECDSA matching inventory `agent_address` |
| `RPC_URLS_JSON` or `RPC_URL_eip155_*` | Testnet GatewayOK chains only (mainnet keys refused at startup) |
| `LISTEN_ADDR` | Must be loopback (`127.0.0.1:8088`, `[::1]:8088`, `localhost:8088`) — bare `:8088` refused |

Supported action: `circle_gateway_consolidate` deposit steps only. Signs **re-derived** prepare calls (not client calldata). Partial failures return hashes + `executed=false`.

```bash
# Example (local only — never commit key):
export ENABLE_TESTNET_EXECUTE=1
export LISTEN_ADDR=127.0.0.1:8088
export AGENT_PRIVATE_KEY=0x…   # testnet throwaway
export RPC_URL_eip155_84532=https://sepolia.base.org
go run ./cmd/server
```

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
