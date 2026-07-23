# CLAUDE.md

Guidance for agents working in **usdc-liquidity-orchestrator** (kaimo-no).

> **Module detail lives in per-directory `CLAUDE.md` files.** Auto-load any `CLAUDE.md` on the path from the working directory up to the root. Root file = repo-wide rules + index. Module files = invariants and behaviour for that package.

## Overview

**Non-custodial multi-chain USDC rebalancing for agentic commerce.**

Agents hold fragmented USDC across chains; merchants accept USDC on one chain (e.g. Arc Testnet or Base via x402). This component is a **general orchestrator**: agents declare **target** + optional **source** allowlist, then it **plans** how to fund the **agent’s own wallet** on the target using:

1. **Circle Gateway** (preferred) — unified balance, deposit/withdraw  
2. **CCTP Fast Transfer** (fallback) — burn source / mint dest  

It does **not** take product custody and does **not** route funds through a platform MoR wallet. Optional kaimo fee is plan metadata / post-prepare x402 settle only.

| Layer | Status |
|---|---|
| L0 wire types (`pkg/types`) | shipped |
| L1 pure planner (`pkg/liquidity`) | shipped — shortfall-only moves |
| L2 execute | fail-closed default; optional **testnet-only** deposit execute (`pkg/execonchain`) |
| HTTP microservice (`cmd/server`) | `GET /` plan UI, `POST /v1/plan`, `POST /v1/consolidate`, `GET /v1/chains`, `GET /healthz` |
| CLI demo (`cmd/demo`) | shortfall plan + consolidate (+ optional live testnet execute) |

Related private monorepo: `kaimo-no/kaimo-go` (World B commerce router can call this as a library or HTTP service). **This repo has zero import of kaimo-go.**

## Architecture — module index

| Package | One-liner | Detail |
|---|---|---|
| `pkg/liquidity/` | Pure planner, guard, corridor matrix, executor stub | [pkg/liquidity/CLAUDE.md](pkg/liquidity/CLAUDE.md) |
| `pkg/execonchain/` | Optional testnet-only Gateway deposit execute (go-ethereum) | [pkg/execonchain/CLAUDE.md](pkg/execonchain/CLAUDE.md) |
| `pkg/types/` | Agent-facing wire JSON shapes | [pkg/types/CLAUDE.md](pkg/types/CLAUDE.md) |
| `pkg/errors/` | Stable `.Code` strings (`insufficient_liquidity`, …) | [pkg/errors/CLAUDE.md](pkg/errors/CLAUDE.md) |
| `cmd/server/` | Thin HTTP microservice | [cmd/server/CLAUDE.md](cmd/server/CLAUDE.md) |
| `cmd/demo/` | CLI worked example | inline |

## Repo-wide invariants

- **`decimal.Decimal` for all atomic USDC amounts.** Never `float64`. Chain units are **whole atomic units** — do **not** apply GBP-style `Round(2)`.
- **Wire keys via struct tags** on `pkg/types`. No hand-rolled `MarshalJSON` for agent-facing types.
- **Typed errors** in `pkg/errors` (`*errors.Error` + stable `Code`). Classify with `errors.As` / `errors.CodeOf`.
- **Non-custodial prepare:** fund-move step recipients are always `agent_self` / `Inventory.AgentAddress`. Never merchant `pay_to`, never platform MoR wallets.
- **Empty `pay_to` fail-closed.** Never invent a merchant target.
- **Amount override** only when probe amount is missing; cannot change payTo / network / asset.
- **Dry plan stamps:** `dry_run=true`, `executed=false`, `inventory_asserted=true`, `inventory_unverified=true` until real execute lands.
- **`UnconfiguredExecutor` never succeeds.** Default `execute=true` returns `liquidity_rail_unavailable` (or `insufficient_liquidity` for shortfall actions).
- **Optional testnet deposit execute** (`pkg/execonchain.DepositExecutor`): dual-gated env; **consolidate deposits only**; signs **re-derived** `BuildDepositPrepareCalls` only; mainnet RPC keys refused; loopback `LISTEN_ADDR` required for HTTP.
- **Consolidate:** `PlanConsolidate` / `POST /v1/consolidate` — full-balance Gateway deposits, no pay_to/fee; deposit steps may include advisory unsigned `prepare_calls`.
- **Guard dual predicates:** withdraw/cctp need merchant `pay_to`; deposit-only may have empty pay_to.
- **Rail naming:** `circle_gateway_*` / `cctp_fast` — never bare `gateway` (avoids MoR / HTTP-gateway confusion). Bare inventory location `"gateway"` is ignored as invalid.
- **Shortfall-only rebalance:** plan amount = `required − dest_native`, not full required when dest already holds partial funds.
- **Orchestration options:** optional `target_chain_caip2` (must match required), `source_chain_caip2s` allowlist, `allow_circle_gateway`, `prefer_rail`.
- **Fee:** optional `fee_bps` + `fee_recipient` — plan.fee envelope only (never a step); settle via x402 after prepare; not a fund-rail destination.
- **`agent_address == pay_to` refused** on fund-moving plans (anti–confused-deputy).
- **No durable buyer ledger** in this repo. Inventory is request-scoped; do not log balances, wallet private keys, or prepare calldata (`/v1/plan` and `/v1/consolidate`).
- **Canonical layout:** library packages under `pkg/`; thin `cmd/*/main.go`; black-box tests under `tests/` as `package <foo>_test`.
- **Sources / adapters are small interfaces** (`Executor`). No type embedding for behaviour reuse.

## Environment variables

| Var | Default | Purpose |
|---|---|---|
| `LISTEN_ADDR` | `:8088` | HTTP bind for `cmd/server` |
| `ENABLE_TESTNET_EXECUTE` | unset | Set `1` to enable optional testnet deposit execute (dual gate) |
| `AGENT_PRIVATE_KEY` | unset | Hex ECDSA key for deposit txs (never log; never commit) |
| `RPC_URLS_JSON` | unset | JSON object map CAIP-2 → RPC URL (testnet chains only) |
| `RPC_URL_eip155_<id>` | unset | Alternate per-chain RPC env (e.g. `RPC_URL_eip155_84532`) |
| `MAX_AMOUNT_ATOMIC` | unset | Optional Guard max step/required amount (atomic units) |

Plan-only mode needs no secrets. Testnet execute also requires **loopback** `LISTEN_ADDR` (`127.0.0.1`, `::1`, or `localhost` — bare `:8088` is refused). Never commit keys; never log keys, balances, prepare calldata, or RPC URLs.

## Common commands

```bash
go mod download
go build ./...
go test -v -race -coverprofile=coverage.out -coverpkg=./... ./...

gofmt -l .
bash scripts/check-test-layout.sh
bash scripts/check-no-live-secrets.sh
go vet ./...
golangci-lint run

go run ./cmd/demo
go run ./cmd/server   # LISTEN_ADDR=:8088 — UI at http://127.0.0.1:8088/
bash examples/curl.sh

docker compose up --build   # distroless cmd/server on :8088; see SETUP.md for IDE launch
```

## CI security & analysis

- **Dependabot**: `.github/dependabot.yml` (gomod + github-actions)
- **govulncheck**: `.github/workflows/govulncheck.yml`
- **gosec**: via `.golangci.yml` on every `ci` job
- **actionlint**: `.github/workflows/actionlint.yml`
- **gitleaks**: `.github/workflows/gitleaks.yml` (CLI; full history)

No GHAS-only tooling (CodeQL / SARIF upload) required for this public lightweight repo.

Local pre-PR:

```bash
golangci-lint run
go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
gitleaks detect --source . --verbose
bash scripts/check-test-layout.sh
go test -v -race ./...
```

## Testing notes

- One `_test.go` per source file area; table-driven where appropriate.
- **All unit tests under `tests/`** subfolders, `package <foo>_test`, exercise exported API only.
- `github.com/stretchr/testify/require` for hard assertions, `assert` for soft.
- No network in unit tests; no live Circle calls.

## Pre-PR workflow

Use `/create_pr` (5-phase). Design first: `/team_review`. Full path: `/ship_feature`. Post-impl: `/ci_review_loop` or `/ci_local`.

| Agent | Model | Role |
|---|---|---|
| `backend-go` | composer-2.5-fast | Design, code, fix CI |
| `ci-local` | composer-2.5-fast | CI + dual OpenCode |
| `qa-engineer` | grok-build readonly | Pre-impl QA |
| `data-privacy` | grok-build readonly | Pre-impl privacy |
| `security-researcher` | grok-build readonly | Pre-impl security |

Skills: `.claude/skills/` — `/ship_feature`, `/team_review`, `/ci_review_loop`, `/ci_local`, `/create_pr`, `/session_cleanup`.

## Pointers

- Public overview: `README.md`
- Ops checklist: `OPS.md`
- IDE / local run: `SETUP.md`
- PR rules: `.github/pr-review-instructions.md`
- CI: `.github/workflows/ci.yml` + security family
