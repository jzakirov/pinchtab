#!/bin/bash
# 06-screenshot-pdf.sh — Screenshot and PDF export

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab screenshot"

pt_post /navigate -d "{\"url\":\"${FIXTURES_URL}/table.html\"}"
sleep 1

pt_get /screenshot
assert_ok "screenshot"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab pdf (shorthand)"

skip "/pdf shorthand not in server mode"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab screenshot --tab <id>"

pt_get /tabs
TAB_ID=$(get_first_tab)

pt_get "/tabs/${TAB_ID}/screenshot"
assert_ok "tab screenshot"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab pdf --tab <id>"

pt_get /tabs
TAB_ID=$(get_first_tab)

pt_get "/tabs/${TAB_ID}/pdf"
assert_ok "tab pdf"

end_test
