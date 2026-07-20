# OPS

Pre-production / hackathon ops checklist for **usdc-liquidity-orchestrator**.

## Plan-only mode (current default)

- No secrets required
- `cmd/server` is stateless; safe to run ephemerally
- Inventory is request-scoped — do not persist balances server-side

## Hard fail-closed rules

1. Never configure live spend keys in CI or public logs  
2. If adding Circle Gateway / CCTP execute: keep keys out of git; document env vars in root `CLAUDE.md` + `README.md`  
3. Do not log full inventory, private keys, or prepare calldata — covers `POST /v1/plan` and `POST /v1/consolidate`  
4. Merchant `pay_to` is untrusted discovery data — prepare never transfers to it  
5. Gateway Wallet addresses are non-secret constants (testnet/mainnet); not credentials  

## HTTP

| Var | Default | Notes |
|---|---|---|
| `LISTEN_ADDR` | `:8088` | Bind address |

Docker/compose publish the same port (`8088:8088`). Probe externally (`curl …/healthz`); the distroless image has no shell for in-container healthchecks.

## Security scans (local)

```bash
gitleaks detect --source . --verbose
bash scripts/check-no-live-secrets.sh
govulncheck ./...
```

## Relationship to kaimo-go

Private monorepo may embed or HTTP-call this service for World B native_x402 liquidity prep. This repo must remain import-clean of kaimo-go (public hackathon surface).
