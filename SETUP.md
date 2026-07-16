# SETUP

## Prerequisites

- Go (version from `go.mod`)
- Optional: `golangci-lint`, `gitleaks`, `govulncheck`, `actionlint` for full local CI parity

## Clone

```bash
git clone https://github.com/kaimo-no/usdc-liquidity-orchestrator.git
cd usdc-liquidity-orchestrator
go mod download
```

## Run tests

```bash
go test -v -race ./...
bash scripts/check-test-layout.sh
```

## Demo (worked example)

```bash
go run ./cmd/demo
```

Expect Arc Testnet `circle_gateway_withdraw` shortfall 22 USDC (need 42, native Arc 20, gateway covers rest) to `agent_self`, plus optional fee metadata (`settle_via=x402`).

Legacy Base/Arb fragmented example: `examples/plan-base-fragmented.json`.

## HTTP server

```bash
go run ./cmd/server
# another terminal:
bash examples/curl.sh              # Arc + gateway (default)
curl -s localhost:8088/v1/chains | jq .
```

## Agent / IDE

- Root `CLAUDE.md` + package `CLAUDE.md` files for coding agents
- Skills under `.claude/skills/` (`/ship_feature`, `/team_review`, `/ci_local`, …)
- Agents under `.claude/agents/`
