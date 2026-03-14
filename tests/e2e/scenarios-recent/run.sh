#!/bin/bash
# run-recent.sh - Run only recently added/changed E2E test scenarios
# This runs a fast subset for fail-fast CI before the full suites.

set -uo pipefail

SCRIPT_DIR="$(dirname "$0")"
COMMON_DIR="$(dirname "$SCRIPT_DIR")/scenarios"
source "${COMMON_DIR}/common.sh"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}PinchTab E2E Recent Tests${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PINCHTAB_URL: ${PINCHTAB_URL}"
echo "FIXTURES_URL: ${FIXTURES_URL}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "Waiting for instances to become ready..."
wait_for_instance_ready "${PINCHTAB_URL}"
wait_for_instance_ready "${PINCHTAB_SECURE_URL}"
if [ -n "${PINCHTAB_LITE_URL:-}" ]; then
  wait_for_instance_ready "${PINCHTAB_LITE_URL}"
fi
echo ""

# Recent test files — add new scenarios here for fast CI feedback.
# Move to the full suite (run-all.sh) once stable.
# All test files in this directory run as the recent suite.
for script in "${SCRIPT_DIR}"/[0-9][0-9]-*.sh; do
  if [ -f "$script" ]; then
    echo -e "${YELLOW}Running: $(basename "$script")${NC}"
    echo ""
    source "$script"
    echo ""
  fi
done

print_summary

if [ -d "${RESULTS_DIR:-}" ]; then
  echo "passed=$TESTS_PASSED" > "${RESULTS_DIR}/summary.txt"
  echo "failed=$TESTS_FAILED" >> "${RESULTS_DIR}/summary.txt"
  echo "timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "${RESULTS_DIR}/summary.txt"
fi
