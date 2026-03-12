#!/bin/bash
# 07-screenshot.sh — CLI screenshot command

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab ss -o <file>"

pt_ok nav "${FIXTURES_URL}/buttons.html"

TMPFILE="/tmp/test-screenshot-$$.jpg"
pt_ok ss -o "$TMPFILE"

if [ -f "$TMPFILE" ] && [ -s "$TMPFILE" ]; then
  echo -e "  ${GREEN}✓${NC} screenshot saved to file"
  ((ASSERTIONS_PASSED++)) || true
  rm -f "$TMPFILE"
else
  echo -e "  ${RED}✗${NC} screenshot file not created or empty"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# Note: ss without -o outputs binary to stdout which doesn't work
# well with our text-based pt() function, so we only test file output
