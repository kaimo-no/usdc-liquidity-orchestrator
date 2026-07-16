---
name: ci_local
description: Single-pass local CI + dual OpenCode for usdc-liquidity-orchestrator. No fixes, no PR.
argument-hint: "[optional base branch, default main]"
---

# ci_local

1. Spawn **`ci-local`** (`composer-2.5-fast`).
2. Execute `.claude/agents/ci-local.md` Phases 1–4.
3. Return **CI LOCAL REPORT**.

After: `/session_cleanup`.
