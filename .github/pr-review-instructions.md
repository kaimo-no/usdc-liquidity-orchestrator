# PR-review instructions

Read `/CLAUDE.md` first — source of truth for invariants and module map. Module detail in per-directory `CLAUDE.md`. Also `/README.md` for the public overview.

<!-- digest:start -->

## Do not flag documented known limitations as bugs

- Live Circle Gateway / CCTP **execute** is intentionally unconfigured (`UnconfiguredExecutor` always errors).
- Solana corridors are EVM-first `corridor_unsupported` in this cut.
- Inventory balances are client-asserted (`inventory_unverified=true`).

## Do not suggest changes that contradict repo-wide invariants

- **Money / amounts:** `decimal.Decimal` atomic units — never `float64`. Do **not** apply GBP `Round(2)` to chain atomics.
- **Wire keys** via struct tags on `pkg/types`. No hand-rolled `MarshalJSON`.
- **Typed errors** in `pkg/errors`; classify via `errors.As` / `CodeOf`.
- **Non-custodial prepare:** fund-move recipients always `agent_self`. Never merchant `pay_to` as bridge/mint dest.
- **Empty pay_to fail-closed.** Never invent merchant targets.
- **Amount override** only when probe amount missing; cannot change payTo/network/asset.
- **Dry stamps:** `executed=false`, `dry_run=true` on plan-only responses.
- **Rail names:** `circle_gateway_*` / `cctp_fast` — never bare `gateway`.
- **Shortfall-only** rebalance (required − dest native).
- **No durable inventory ledger** in this service.
- **Tests** under `tests/` as `package *_test`.

<!-- digest:end -->

## Schema-change consistency

Changes to `pkg/types` wire shapes must update tests + relevant `CLAUDE.md` / `README.md` examples.

## Dependencies

`go.mod` changes need `go.sum` tidy; call out runtime concerns in root `CLAUDE.md` if any.

## Test conventions

- Table-driven tests
- `testify/require` hard, `assert` soft
- No live network / Circle API in unit tests

## Smells to flag

- Fund step with recipient = pay_to
- Full-amount bridge ignoring partial dest balance
- `UnconfiguredExecutor` returning nil error
- Logging inventory / private keys
- `float64` money
- Tests co-located outside `tests/`
