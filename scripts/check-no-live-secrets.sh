#!/usr/bin/env bash
# Fail-closed complement to gitleaks: live/production credential patterns
# must never appear in git-tracked files.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

failed=0

check_pattern() {
  local label="$1"
  local pattern="$2"
  if git grep -nE "$pattern" -- . \
    ':(exclude)scripts/check-no-live-secrets.sh' \
    >/dev/null 2>&1; then
    echo "FAIL: $label matched in tracked files:"
    git grep -nE "$pattern" -- . \
      ':(exclude)scripts/check-no-live-secrets.sh' || true
    failed=1
  fi
}

check_pattern "Stripe live secret key (sk_live_)" 'sk_live_[a-zA-Z0-9]{10,}'
check_pattern "GitHub personal access token (ghp_)" 'ghp_[a-zA-Z0-9]{20,}'
check_pattern "GitHub OAuth token (gho_)" 'gho_[a-zA-Z0-9]{20,}'
check_pattern "GitHub fine-grained PAT (github_pat_)" 'github_pat_[a-zA-Z0-9_]{20,}'
check_pattern "PEM private key block" '-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----'

if [ "$failed" -ne 0 ]; then
  exit 1
fi

echo "OK: no live-secret patterns in tracked files"
