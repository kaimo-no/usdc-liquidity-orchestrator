---
name: ci_review_loop
description: Two ci-local passes each followed by backend-go fix-ci, then create_pr when clean.
---

# ci_review_loop

```
Cycle 1:  ci-local  →  backend-go (MODE: fix-ci)   [skip if ready-for-pr]
Cycle 2:  ci-local  →  backend-go (MODE: fix-ci)   [skip if ready-for-pr]
Final:    create_pr  when cycle-2 OVERALL: ready-for-pr
```

Never a third pass. If blocked (infra only), report NEXT clearly.
