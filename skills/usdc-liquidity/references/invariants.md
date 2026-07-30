# Invariants

1. **agent_self only** — fund-move step recipients are always the agent wallet; never merchant `pay_to`, never platform MoR.
2. **Empty pay_to** fail-closed on withdraw/cctp; deposit-only consolidate may omit pay_to.
3. **Shortfall-only** on `PlanOrchestration` / `plan` — amount = required − dest_native.
4. **Scenario full-funding** (`payment-funding` / demo) is separate; hard-coded sources + full withdraw.
5. **Dry stamps** — `dry_run=true`, `executed=false`, inventory asserted+unverified until real execute.
6. **Execute fail-closed** — default `UnconfiguredExecutor`; live path testnet-only, re-derived prepare_calls, agent_self mint dest.
7. **`decimal.Decimal`** atomic USDC — no `float64`, no GBP-style `Round(2)`.
8. **Rails** named `circle_gateway_*` / `cctp_fast` — never bare `gateway` as a rail.
9. **`agent_address == pay_to`** refused on fund-moving plans (anti–confused-deputy).
10. **Privacy** — no durable buyer ledger; never log keys, balances, calldata, RPC URLs, or agent_address in failure notes.
11. **Re-plan only** for execute — never trust client-supplied `prepare_calls` for signing.
