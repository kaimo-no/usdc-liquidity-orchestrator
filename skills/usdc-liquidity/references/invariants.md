# Invariants

1. **agent_self only** — fund-move step recipients are always the agent wallet; never merchant `pay_to`, never platform MoR.
2. **Empty pay_to OK** on Phase B withdraw/cctp (agent_self land). Residual pay_to never fund dest.
3. **Phase split** — Phase A = deposit/consolidate only; Phase B = gateway withdraw or CCTP only. Never emit `circle_gateway_deposit_withdraw` (deposit+withdraw) in one plan.
4. **Shortfall-only** on `PlanOrchestration` / `plan` / `move` — amount = required − dest_native.
5. **Scenario Phase A** (`payment-funding` / demo) is multi-source deposits only; wait ~13–19m finality then Phase B withdraw separately.
6. **Dry stamps** — `dry_run=true`, `executed=false`, inventory asserted+unverified until real execute.
7. **Execute fail-closed** — default `UnconfiguredExecutor`; live path testnet-only, re-derived prepare_calls, agent_self mint dest; no composite execute.
8. **`decimal.Decimal`** atomic USDC — no `float64`, no GBP-style `Round(2)`.
9. **Rails** named `circle_gateway_*` / `cctp_fast` — never bare `gateway` as a rail.
10. **`agent_address == pay_to`** refused when residual pay_to is set (anti–confused-deputy).
11. **Privacy** — no durable buyer ledger; never log keys, balances, calldata, RPC URLs, or agent_address in failure notes.
12. **Re-plan only** for execute — never trust client-supplied `prepare_calls` for signing.
