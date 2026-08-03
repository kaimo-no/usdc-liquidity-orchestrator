# Invariants

1. **agent_self only** — fund-move step recipients are always the agent wallet; never merchant `pay_to`, never platform MoR.
2. **Empty pay_to OK** on Phase B withdraw/cctp (agent_self land). Residual pay_to never fund dest.
3. **Phase split** — Phase A = deposit/consolidate only; Phase B = gateway withdraw or CCTP only. Never emit `circle_gateway_deposit_withdraw` (deposit+withdraw) in one plan.
4. **Shortfall-only** on `PlanOrchestration` / `plan` / `move` — amount = required − dest_native.
5. **Scenario Phase A** (`payment-funding` / demo) is multi-source deposits only; wait ~13–19m finality then Phase B withdraw separately.
6. **Fixed multi deposit** (`PlanGatewayDeposits` / CLI `--from` / JSON `sources[]`) — no payment-sum rule; hard underfund per source; duplicate chains merge (sum) into one step.
7. **Dry stamps** — `dry_run=true`, `executed=false`, inventory asserted+unverified until real execute.
8. **Execute fail-closed** — default `UnconfiguredExecutor`; live path testnet-only, re-derived prepare_calls, agent_self mint dest; no composite execute.
9. **Multi-deposit partial execute** — if some steps land and others fail: re-load inventory and re-plan **remaining** amounts only (do not re-submit the full original plan).
10. **`MAX_AMOUNT_ATOMIC` / Guard** — per-step (and plan `required` when set), **not** a multi-step plan-total sum.
11. **`decimal.Decimal`** atomic USDC — no `float64`, no GBP-style `Round(2)`.
12. **Rails** named `circle_gateway_*` / `cctp_fast` — never bare `gateway` as a rail.
13. **`agent_address == pay_to`** refused when residual pay_to is set (anti–confused-deputy).
14. **Privacy** — no durable buyer ledger; never log keys, balances, calldata, RPC URLs, or agent_address in failure notes.
15. **Re-plan only** for execute — never trust client-supplied `prepare_calls` for signing.
