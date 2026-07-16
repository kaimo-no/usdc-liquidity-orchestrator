#!/usr/bin/env bash
# Fails if any *_test.go exists outside a tests/ subfolder.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

violations=()
while IFS= read -r -d '' f; do
  rel="${f#./}"
  if [[ "$rel" != */tests/* ]]; then
    violations+=("$rel")
  fi
done < <(find . -name '*_test.go' \
  -not -path './.git/*' \
  -not -path './.claude/*' \
  -print0)

if [ ${#violations[@]} -gt 0 ]; then
  echo "FAIL: *_test.go files must live under tests/:"
  printf '  %s\n' "${violations[@]}"
  exit 1
fi

echo "OK: test layout"
