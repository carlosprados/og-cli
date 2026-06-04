#!/usr/bin/env bash
# Lints OpenGate widget JavaScript (_widgetConfigCode.js, formatters, rule JS)
# against the EMPIRICAL constraints of the platform's render-time JSHint.
# Run before every `og dashboard deploy` / `og rules deploy`.
#
# Usage: scripts/lint-widget.sh <file.js> [more.js ...]
#        scripts/lint-widget.sh demo/workspaces/**/*.js
set -uo pipefail

if [ $# -eq 0 ]; then
  echo "usage: $0 <widget.js> [...]" >&2
  exit 2
fi

FAIL=0
for f in "$@"; do
  [ -f "$f" ] || { echo "SKIP (not a file): $f"; continue; }
  echo "== $f"

  # 1. Syntax check via node (wrapped in async fn like the platform does)
  if command -v node >/dev/null 2>&1; then
    if ! node -e "
      const fs = require('fs');
      const code = fs.readFileSync('$f', 'utf8');
      try { new Function('return async function(entityData, relatedEntities, timeserieData, alarmData, dashboardFilters, filters, pageElements, page, callback) {' + code + '}'); }
      catch (e) { console.error('  SYNTAX: ' + e.message); process.exit(1); }
    "; then
      FAIL=1
      continue
    fi
  fi

  # 2. Platform JSHint constraints (verified empirically 2026-06-04):
  #    const/let, arrow functions, for...of and nested async functions FAIL
  #    the render-time lint even though they are valid modern JS.
  ERRS=$(grep -nE '(^|[^a-zA-Z_])((const|let)[[:space:]]|=>|for[[:space:]]*\([[:space:]]*(const|let|var)?[[:space:]]*[a-zA-Z_$][a-zA-Z0-9_$]*[[:space:]]+of[[:space:]]|async[[:space:]]+function|`)' "$f" || true)
  if [ -n "$ERRS" ]; then
    echo "  PLATFORM-LINT (will fail at render with JshintError):"
    echo "$ERRS" | sed 's/^/    line /'
    echo "  → rewrite with var / function declarations / .forEach(function () {})"
    FAIL=1
  else
    echo "  OK"
  fi
done

exit $FAIL
