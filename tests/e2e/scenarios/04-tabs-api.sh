#!/bin/bash
# 04-tabs-api.sh — Tab-scoped API tests (regression test for #207)

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --tab <id> (regression #207)"

# Create a tab
pt_post /navigate -d "{\"url\":\"${FIXTURES_URL}/index.html\"}"
show_tab "created" "$(get_tab_id)"

# Get tab ID from /tabs endpoint
pt_get /tabs
TAB_ID=$(get_last_tab)
show_tab "listed" "$TAB_ID"

# Test: /tabs/{id}/snapshot should work (was broken in #207)
pt_get "/tabs/${TAB_ID}/snapshot"
assert_ok "tab snapshot"
assert_json_contains "$RESULT" '.title' 'E2E Test'

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab text/screenshot --tab <id>"

pt_post /navigate -d "{\"url\":\"${FIXTURES_URL}/form.html\"}"
TAB_ID=$(get_tab_id)

pt_get "/tabs/${TAB_ID}/text"
assert_ok "tab text"

pt_get "/tabs/${TAB_ID}/screenshot"
assert_ok "tab screenshot"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab tab close"

BEFORE=$(get_tab_count)

pt_post /navigate -d "{\"url\":\"${FIXTURES_URL}/buttons.html\"}"
TAB_ID=$(get_tab_id)

pt_post "/tabs/${TAB_ID}/close" -d '{}'
assert_ok "tab close"

sleep 1
assert_tab_closed "$BEFORE"

end_test
