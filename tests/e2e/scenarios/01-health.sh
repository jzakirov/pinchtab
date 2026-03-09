#!/bin/bash
# 01-health.sh — Basic connectivity tests

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab health"

pt_get /health
assert_json_eq "$RESULT" '.status' 'ok'

end_test

# ─────────────────────────────────────────────────────────────────
start_test "fixtures server"

assert_fixtures_accessible

end_test
