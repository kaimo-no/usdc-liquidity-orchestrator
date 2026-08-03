# cmd/usdc-liq/

Thin CLI entrypoint for **usdc-liq** — dual-surface peer of `cmd/server` (HTTP) and `pkg/liquidity` (library).

## Behaviour

Logic lives in `internal/liqcli.Main`. This package only:

1. Loads optional gitignored `.env` via `internal/envfile` (never logs values)
2. Calls `liqcli.Main(os.Args[1:], stdin, stdout, stderr)` and exits with its code

## Subcommands

| Command | HTTP parity | Notes |
|---|---|---|
| `plan` | `POST /v1/plan` | shortfall-only; JSON `-f` **or** easy flags |
| `consolidate` | `POST /v1/consolidate` | full native → gateway; JSON `-f` **or** easy flags |
| `deposit` | CLI-only | fixed-N Gateway deposit; single or multi `--from`; no HTTP this cut |
| `move` | CLI-only | self-land shortfall; no HTTP this cut |
| `payment-funding` | `POST /v1/payment-funding` | scenario full-funding (JSON only) |
| `chains` | `GET /v1/chains` | corridor registry |
| `inventory` | CLI-only | live balances; needs agent + RPCs |
| `demo` | CLI-only | `internal/demorun` worked examples |
| `version` | CLI-only | print version |

## JSON vs easy mode (`plan` / `consolidate` / `deposit` / `move`)

| Mode | When | Body |
|---|---|---|
| **Easy** | Any body easy flag: `--dest`, `--sources`, `--source`, `--from`, `--amount`, `--amount-atomic`, `--pay-to`, `--balance`, `--gateway-balance`, `--live` | Built in-process; **no stdin/`-f` read** |
| **JSON** | No body easy flags | `-f` file (default `-` = stdin) |
| **Exclusive** | Body easy flags **and** `-f` Visited | exit **2** |

Incomplete easy (missing dest / amount / agent / source as required) → exit **2**, never hangs on stdin.

Non-body flags alone do **not** force easy: `--agent`, `--private-key`, `--rpc`, `--mainnet`, `--execute`.

### Easy flags (summary)

| Flag | Meaning |
|---|---|
| `--dest` / `--sources` | chain ref: Gateway domain id, registry name, or CAIP-2 |
| `--source` | single deposit source chain (XOR `--from`) |
| `--from REF=USDC` | multi fixed deposit; **human USDC only** (×10^6); repeatable; duplicate chains merge |
| `--amount` / `--amount-atomic` | human USDC (×10^6) XOR atomic; refuse >6 frac digits |
| `--pay-to` | merchant claim (plan only; never fund dest) |
| `--balance REF=USDC` | asserted native (repeatable); XOR `--live` |
| `--gateway-balance` | asserted `circle_gateway` human USDC |
| `--live` | `inventory.Load` (testnet RPCs); refuse with balances; refuse with `--mainnet` |
| `--agent` / `--private-key` | identity; key prefers flag then `AGENT_PRIVATE_KEY`; match when both set |
| `--rpc REF=URL` | overlay env RPCs (repeatable) |
| `--mainnet` | domain resolution mainnet (default testnet) |
| `--execute` | dual-gate live execute (`ENABLE_TESTNET_EXECUTE=1`) |

`planio.RunPlan` / `RunConsolidate` / `RunDeposit` / `RunMove`. Deposit multi uses `sources[]` XOR single fields. `--from` is human USDC only — raw atomic multi via JSON `sources[].amount_atomic`. `--live` does not imply execute.

## Dry vs execute

- **Dry (default):** always `UnconfiguredExecutor` — no live keys/RPCs required
- **`--execute`:** `execenv.BuildExecutor` with optional `--private-key` / `--rpc` overlays (no loopback); incomplete dual-gate → exit 1 sanitized, no Execute
- HTTP boot still requires loopback `LISTEN_ADDR` when execute is enabled

## Exit codes

| Code | Meaning |
|---|---|
| 0 | StampOK (dry or full execute success) |
| 1 | StampPartial / StampFail / configure / live inventory error |
| 2 | usage / exclusive modes / incomplete easy |

JSON on stdout; stderr notes only (never keys, agent_address, balances, calldata, RPC URLs). Body limit 1 MiB.

## Product skill

Agent-facing guide: [`skills/usdc-liquidity/`](../../skills/usdc-liquidity/).
