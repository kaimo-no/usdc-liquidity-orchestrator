---
name: team_review
description: Pre-implementation design gate — backend-go designs, then qa + privacy + security in parallel, rebuttal if needed.
argument-hint: "<feature description>"
---

# team_review

**Project context:** Non-custodial USDC multi-chain liquidity planner (Circle Gateway + CCTP). Public kaimo-no repo. No kaimo-go imports. agent_self only.

### Phase 1 — Design (`backend-go`, MODE: design)

APPROACH, FILES, SIGNATURES, TESTS, OPEN QUESTIONS.

### Phase 2 — Parallel review (readonly)

`qa-engineer` + `data-privacy` + `security-researcher`.

### Phase 3 — Rebuttal (`backend-go`)

Skip if all ready / no P0.

### Phase 4 — Summary

```
FEATURE: ...
VERDICT: approved | needs-work | blocked
IMPLEMENTATION PLAN: ...
NEXT: backend-go MODE: implement
```
