---
name: session_cleanup
description: Remove local temp artifacts (pr.diff, coverage.out). Does not touch git or source.
---

# session_cleanup

```bash
rm -f kaimo-pr.diff /tmp/kaimo-pr.diff /tmp/kaimo-opencode-glm.txt /tmp/kaimo-opencode-qwen.txt
rm -f coverage.out coverage.txt
```

Never remove `.claude/agents`, `.claude/skills`, source under `pkg/`/`cmd/`.
