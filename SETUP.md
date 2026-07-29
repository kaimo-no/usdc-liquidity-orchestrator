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
cp .env.example .env   # optional; fill AGENT_ADDRESS + payment scenario
go run ./cmd/demo
```

Primary path (when `PAYMENT_CHAIN` / scenario env is set): **full-funding** dry plan via `PlanPaymentFunding` — hard-coded `SOURCE_AMOUNT_*` deposits (scaled by `USDC_SCALE_FACTOR`) + withdraw full payment real to `agent_self`. Wire stamps `amount_atomic` (real) and optional `amount_logical_atomic` / `scale_factor`.

Also prints shortfall smoke (Arc Testnet `circle_gateway_withdraw` need 42 / native 20 → shortfall 22) and consolidate with unsigned `prepare_calls`.

Legacy Base/Arb fragmented example: `examples/plan-base-fragmented.json`. Scenario path is **not** HTTP `/v1/plan`.
## HTTP server

```bash
go run ./cmd/server
# browser UI (plan / consolidate / chains):
open http://127.0.0.1:8088/
# another terminal:
bash examples/curl.sh              # Arc + gateway (default)
curl -s localhost:8088/v1/chains | jq .
```

The UI is a single embedded page (`internal/httpserver/static/index.html`): set your agent address and asserted balances, run dry plans. No private keys — inventory is client-asserted only.

## Optional testnet deposit execute (local only)

Dual-gated; **not** for Docker default. See [`OPS.md`](./OPS.md).

```bash
export ENABLE_TESTNET_EXECUTE=1
export LISTEN_ADDR=127.0.0.1:8088   # loopback required (bare :8088 refused)
export AGENT_PRIVATE_KEY=0x…        # throwaway testnet key; never commit
export RPC_URL_BASE_SEPOLIA=https://sepolia.base.org
# export RPC_URL_ARBITRUM_SEPOLIA=…
# export RPC_URL_ARC_TESTNET=…
# export RPC_URL_SOLANA_DEVNET=…    # ops placeholder; not used by EVM deposit execute
# or CAIP form: RPC_URL_eip155_84532=… / RPC_URLS_JSON='{"eip155:84532":"https://…"}'
go run ./cmd/server
# demo live path (stderr tx hashes only):
go run ./cmd/demo
```

Copy [`.env.example`](./.env.example) for named placeholders. Requirements: inventory `agent_address` must match the key; only `circle_gateway_consolidate` deposit steps; mainnet RPCs refused; Solana RPC is stored for ops but not used by deposit execute yet.

## VS Code / Cursor

Requires the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go) (Delve debugger).

1. Open this repo as the workspace folder
2. **Run and Debug** → choose a configuration:
   - **HTTP Server** — `cmd/server` on `LISTEN_ADDR=:8088`
   - **Demo (CLI)** — `cmd/demo` worked examples (exits when done)

Configs live in [`.vscode/launch.json`](./.vscode/launch.json).

## Docker

Multi-stage image builds a static `cmd/server` binary (distroless, non-root). No secrets required for plan-only mode.

```bash
docker compose up --build
# or:
docker build -t usdc-liquidity-orchestrator:local .
docker run --rm -p 8088:8088 usdc-liquidity-orchestrator:local
```

Smoke test (another terminal):

```bash
curl -sS localhost:8088/healthz
bash examples/curl.sh
```

Bind address inside the container is `LISTEN_ADDR` (default `:8088`); map host port `8088` as above.

## Agent / IDE

- Root `CLAUDE.md` + package `CLAUDE.md` files for coding agents
- Skills under `.claude/skills/` (`/ship_feature`, `/team_review`, `/ci_local`, …)
- Agents under `.claude/agents/`
