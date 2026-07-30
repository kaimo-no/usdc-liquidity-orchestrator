# CLI (`usdc-liq`)

```bash
go run ./cmd/usdc-liq <command> [flags]
# or: go build -o usdc-liq ./cmd/usdc-liq && ./usdc-liq …
```

Optional gitignored `.env` is loaded at startup (`internal/envfile`).

## Commands

| Command | Input | Output |
|---|---|---|
| `plan` | JSON (`-f`) **or** easy flags | `PlanResponse` (shortfall-only) |
| `consolidate` | JSON **or** easy flags | `PlanResponse` |
| `payment-funding` | JSON body | `PlanResponse` |
| `chains` | none | `ChainsResponse` |
| `inventory` | `-agent` or `AGENT_ADDRESS` + RPC env | wire `Inventory` |
| `demo` | scenario env | multi-plan demo |
| `version` | none | version string |

## Easy mode (plan / consolidate)

Flag-first path — no JSON file required. **Mutually exclusive with `-f`**.

Chain refs accept **Gateway domain id**, **registry name**, or **CAIP-2** (`usdc-liq chains` lists them). Domain ids disambiguate via default **testnet** or `--mainnet`.

```bash
# Dry asserted inventory (no network)
usdc-liq plan \
  --agent 0xAgent… \
  --pay-to 0xMerchant… \
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
  --pay-to 0xMerchant… \
  --dest arc-testnet \
  --amount 42 \
  --sources base-sepolia,arbitrum-sepolia \
  --rpc base-sepolia=$RPC_URL_BASE_SEPOLIA \
  --rpc arc-testnet=$RPC_URL_ARC_TESTNET \
  --live

# Execute still dual-gated (ENABLE_TESTNET_EXECUTE=1 + key + RPCs)
ENABLE_TESTNET_EXECUTE=1 usdc-liq plan … --live --execute

# Consolidate: no pay_to / amount
usdc-liq consolidate --agent 0x… --balance base-sepolia=10
```

| Rule | Behaviour |
|---|---|
| Incomplete easy | exit **2**, fixed usage; **never** hangs on stdin |
| Easy + `-f` | exit **2** exclusive |
| `--live` + balances | refuse |
| `--mainnet` + `--live` | refuse (MVP testnet inventory) |
| `--live` alone | does **not** set execute |
| Amount | `--amount` XOR `--amount-atomic`; no `USDC_SCALE_FACTOR` |

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
