# CLI (`usdc-liq`)

```bash
go run ./cmd/usdc-liq <command> [flags]
# or: go build -o usdc-liq ./cmd/usdc-liq && ./usdc-liq …
```

Optional gitignored `.env` is loaded at startup (`internal/envfile`).

## Commands

| Command | Input | Output |
|---|---|---|
| `plan` | JSON (`-f`) **or** easy flags | `PlanResponse` (Phase B shortfall land; no pay_to) |
| `consolidate` | JSON **or** easy flags | `PlanResponse` (Phase A full native → gateway) |
| `deposit` | JSON **or** easy flags | `PlanResponse` (Phase A fixed-N single or multi-source deposit) |
| `move` | JSON **or** easy flags | `PlanResponse` (Phase B land N on dest agent_self) |
| `payment-funding` | JSON body | `PlanResponse` (Phase A multi-source deposits only) |
| `chains` | none | `ChainsResponse` |
| `inventory` | `-agent` or `AGENT_ADDRESS` + RPC env | wire `Inventory` |
| `demo` | scenario env | multi-plan demo |
| `version` | none | version string |

## Easy mode (plan / consolidate / deposit / move)

Flag-first path — no JSON file required. **Mutually exclusive with `-f`**.

Chain refs accept **Gateway domain id**, **registry name**, or **CAIP-2** (`usdc-liq chains` lists them). Domain ids disambiguate via default **testnet** or `--mainnet`.

```bash
# Phase B dry land (asserted inventory; no network)
usdc-liq plan \
  --agent 0xAgent… \
  --dest 26 \
  --amount 42 \
  --sources 6,3 \
  --balance 6=20 \
  --balance 3=10 \
  --gateway-balance 100

# Human USDC → atomic ×10^6 (refuse >6 fractional digits)
# --amount 42 → amount_atomic 42000000  (or use --amount-atomic)

# Live inventory (testnet RPCs; not with --balance)
usdc-liq plan \
  --private-key "$AGENT_PRIVATE_KEY" \
  --dest arc-testnet \
  --amount 42 \
  --sources base-sepolia,arbitrum-sepolia \
  --rpc base-sepolia=$RPC_URL_BASE_SEPOLIA \
  --rpc arc-testnet=$RPC_URL_ARC_TESTNET \
  --live

# Execute still dual-gated (ENABLE_TESTNET_EXECUTE=1 + key + RPCs)
ENABLE_TESTNET_EXECUTE=1 usdc-liq plan … --live --execute

# Phase A consolidate: full balances → gateway (not fixed amount; use deposit)
usdc-liq consolidate --agent 0x… --balance base-sepolia=10

# Phase A deposit: fixed N on one source
usdc-liq deposit --agent 0x… --source base-sepolia --amount 10 --balance base-sepolia=20
# Phase A multi fixed amounts (XOR --source/--amount):
usdc-liq deposit --agent 0x… \
  --from base-sepolia=3 --from arbitrum-sepolia=2 \
  --balance base-sepolia=3 --balance arbitrum-sepolia=2
# JSON multi: sources[] on DepositRequest (examples/deposit-multi.json)
# After deposit execute, wait ~13–19m finality before Phase B withdraw.

# Phase B move: land N on dest agent_self (shortfall-only)
usdc-liq move --agent 0x… --dest arc-testnet --amount 42 --gateway-balance 100
```

### Deposit amounts: `--from` vs atomic JSON

| Input | Units | Notes |
|---|---|---|
| Easy `--from REF=USDC` | **Human USDC only** (× 10^6 → atomic) | Same rules as `--amount` / `--balance`: refuse >6 fractional digits. Duplicate chain refs merge (sum). |
| Easy single `--amount` | Human USDC (× 10^6) | XOR `--amount-atomic` |
| Easy `--amount-atomic` | Raw whole atomic units | Single-source only |
| JSON `sources[].amount_atomic` | **Raw whole atomic units** | Multi fixed-N; mutually exclusive with single `source_chain_caip2` / `amount_atomic` |
| JSON single `amount_atomic` | Raw whole atomic units | With `source_chain_caip2` |

There is no `--from-atomic` flag. For raw atomic multi-source, use `-f` JSON (`examples/deposit-multi.json`).

### Multi-deposit execute and partial failure

Execute is **fail-closed** by default. When live multi-step deposit execute is enabled:

- Steps run as planned; if a later step fails after earlier deposits landed, **do not** re-submit the full original plan.
- **Re-load inventory**, then **re-plan only remaining** fixed amounts for chains still needing deposit.
- Duplicate `--from` / `sources[]` rows for the same chain are merged before plan — residual re-plans should state the remaining amount only.

### `MAX_AMOUNT_ATOMIC` (Guard)

Env `MAX_AMOUNT_ATOMIC` (via exec Guard) is a **per-step** cap on fund-moving step amounts (and plan `required` when set) — **not** a plan-total sum across multi-deposit steps. Two steps of 2 USDC each can pass a 2.5 USDC cap even though the plan total is 4 USDC.

| Rule | Behaviour |
|---|---|
| Incomplete easy | exit **2**, fixed usage; **never** hangs on stdin |
| Easy + `-f` | exit **2** exclusive |
| `--live` + balances | refuse |
| `--mainnet` + `--live` | refuse (MVP testnet inventory) |
| `--live` alone | does **not** set execute |
| Amount | `--amount` XOR `--amount-atomic`; no `USDC_SCALE_FACTOR` |
| Multi deposit | `--from` XOR `--source`/`--amount`/`--amount-atomic` |

### Private key / argv risk

`--private-key` is supported for agents/scripts (flag wins over `AGENT_PRIVATE_KEY`). **CLI argv may be visible** to local process listings (`ps`, audit tools). Prefer `AGENT_PRIVATE_KEY` in the environment for long-lived shells. The CLI never logs or echoes the key; failure notes use fixed messages only.

## JSON mode

Shared: `-f file` (default `-` = stdin), `--execute`. Runtime `--private-key` / `--rpc` still overlay env for execute when set.

```bash
go run ./cmd/usdc-liq plan -f examples/plan.json
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success (dry or full execute) |
| 1 | plan/execute/configure/live-inventory failure (JSON still on stdout when plan-shaped) |
| 2 | usage / exclusive modes / incomplete easy |

Body limit **1 MiB**. Stderr notes never include secrets, agent_address, balances, or RPC URLs.

## Dry vs execute

- Dry: always `UnconfiguredExecutor` — no live config required.
- `--execute`: `execenv.BuildExecutor` without loopback; incomplete dual-gate → exit 1 sanitized, no Execute call.
