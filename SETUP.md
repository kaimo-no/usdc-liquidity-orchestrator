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

Expect `circle_gateway_deposit_withdraw` with shortfall steps (22 USDC atomic units when need=42, Base has 20, Arb has 30).

## HTTP server

```bash
go run ./cmd/server
# another terminal:
bash examples/curl.sh
```

## Agent / IDE

- Root `CLAUDE.md` + package `CLAUDE.md` files for coding agents
- Skills under `.claude/skills/` (`/ship_feature`, `/team_review`, `/ci_local`, …)
- Agents under `.claude/agents/`
