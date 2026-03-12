#!/bin/bash
# 06-text.sh — CLI text extraction command

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab text"

pt_ok nav "${FIXTURES_URL}/index.html"
pt_ok text
assert_output_json
assert_output_contains "text" "returns text field"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab text --raw"

pt_ok text --raw
# Raw output succeeds - content varies by page
# Just verify command succeeded (exit 0)

end_test
