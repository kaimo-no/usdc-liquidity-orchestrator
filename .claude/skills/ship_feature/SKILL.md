---
name: ship_feature
description: End-to-end feature pipeline for usdc-liquidity-orchestrator — team_review → backend-go implement → ci_review_loop → PR.
when-to-use: User wants a feature shipped from description to GitHub PR.
argument-hint: "<feature description>"
---

# ship_feature

Four phases. Agents: `backend-go`, `qa-engineer`, `data-privacy`, `security-researcher`, `ci-local`.

## Step 0 — Arm goal

Ship to PR: `$ARGUMENTS`

## Phase 1 — `/team_review`

Execute `.claude/skills/team_review/SKILL.md` in full. Stop on blocked.

## Phase 2 — implement

Spawn `backend-go` MODE: implement with approved plan.

## Phase 3 — `/ci_review_loop`

Execute `.claude/skills/ci_review_loop/SKILL.md` (2× ci-local + fix-ci → create_pr).

## Phase 4 — Done

Report PR URL + branch.

## Rules

- `update_goal` each phase
- Never skip team_review or ci_review_loop
- No Postgres required for this repo
- Post-impl OpenCode only locally
