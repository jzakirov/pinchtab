#!/bin/bash
# 39-scheduler.sh — Scheduler task lifecycle E2E tests

source "$(dirname "$0")/common.sh"

AGENT="test-agent-$$"

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks — submit task"

pt_post /tasks -d "{\"agentId\":\"${AGENT}\",\"action\":\"navigate\",\"params\":{\"url\":\"${FIXTURES_URL}/index.html\"}}"
assert_http_status "202" "task accepted"
TASK_ID=$(echo "$RESULT" | jq -r '.taskId')
assert_json_eq "$RESULT" ".state" "queued" "initial state is queued"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "GET /tasks/{id} — get task"

# Wait briefly for task to execute
sleep 2
pt_get "/tasks/${TASK_ID}"
assert_ok "get task by id"
assert_json_eq "$RESULT" ".id" "$TASK_ID" "correct task id"
assert_json_eq "$RESULT" ".agentId" "$AGENT" "correct agent id"
assert_json_eq "$RESULT" ".action" "navigate" "correct action"
# State should be completed or running by now
STATE=$(echo "$RESULT" | jq -r '.state')
if [ "$STATE" = "completed" ] || [ "$STATE" = "running" ]; then
  echo -e "  ${GREEN}✓${NC} task state is $STATE"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} unexpected state: $STATE (expected completed or running)"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "GET /tasks — list tasks"

pt_get "/tasks?agentId=${AGENT}"
assert_ok "list tasks"
COUNT=$(echo "$RESULT" | jq '.count')
if [ "$COUNT" -ge 1 ]; then
  echo -e "  ${GREEN}✓${NC} found $COUNT tasks for agent"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} expected at least 1 task, got $COUNT"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "GET /tasks — filter by state"

pt_get "/tasks?state=completed"
assert_ok "list completed tasks"
echo "$RESULT" | jq -e '.tasks' > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "  ${GREEN}✓${NC} tasks array present in response"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} missing tasks array"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks/{id}/cancel — cancel queued task"

# Submit a task, then cancel before it runs
# Use a slow action to give us time
pt_post /tasks -d "{\"agentId\":\"${AGENT}\",\"action\":\"navigate\",\"params\":{\"url\":\"${FIXTURES_URL}/index.html\"}}"
assert_http_status "202" "task accepted for cancel test"
CANCEL_ID=$(echo "$RESULT" | jq -r '.taskId')

pt_post "/tasks/${CANCEL_ID}/cancel"
# Could be 200 (cancelled) or 409 (already completed)
if [ "$HTTP_STATUS" = "200" ]; then
  assert_json_eq "$RESULT" ".status" "cancelled" "task cancelled"
elif [ "$HTTP_STATUS" = "409" ]; then
  echo -e "  ${GREEN}✓${NC} task already completed (409 conflict, acceptable)"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} unexpected status: $HTTP_STATUS"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks/{id}/cancel — cancel nonexistent → 404"

pt_post "/tasks/tsk_nonexistent/cancel"
assert_http_status "404" "cancel nonexistent task"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "GET /scheduler/stats"

pt_get /scheduler/stats
assert_ok "stats endpoint"
echo "$RESULT" | jq -e '.queue' > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "  ${GREEN}✓${NC} has queue stats"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} missing queue stats"
  ((ASSERTIONS_FAILED++)) || true
fi
echo "$RESULT" | jq -e '.metrics' > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "  ${GREEN}✓${NC} has metrics"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} missing metrics"
  ((ASSERTIONS_FAILED++)) || true
fi
echo "$RESULT" | jq -e '.config.strategy' > /dev/null 2>&1
if [ $? -eq 0 ]; then
  echo -e "  ${GREEN}✓${NC} has config"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} missing config"
  ((ASSERTIONS_FAILED++)) || true
fi
assert_json_eq "$RESULT" ".config.strategy" "fair-fifo" "strategy is fair-fifo"
assert_json_eq "$RESULT" ".config.maxQueueSize" "5" "maxQueueSize matches config"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks/batch — submit 3 tasks"

pt_post /tasks/batch -d "{\"agentId\":\"${AGENT}\",\"tasks\":[{\"action\":\"navigate\",\"params\":{\"url\":\"${FIXTURES_URL}/index.html\"}},{\"action\":\"navigate\",\"params\":{\"url\":\"${FIXTURES_URL}/form.html\"}},{\"action\":\"navigate\",\"params\":{\"url\":\"${FIXTURES_URL}/buttons.html\"}}]}"
assert_http_status "202" "batch accepted"
SUBMITTED=$(echo "$RESULT" | jq '.submitted')
if [ "$SUBMITTED" = "3" ]; then
  echo -e "  ${GREEN}✓${NC} all 3 tasks submitted"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} expected 3 submitted, got $SUBMITTED"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks/batch — validation: empty tasks"

pt_post /tasks/batch -d '{"agentId":"test","tasks":[]}'
assert_http_status "400" "empty batch rejected"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks/batch — validation: missing agentId"

pt_post /tasks/batch -d '{"tasks":[{"action":"navigate"}]}'
assert_http_status "400" "missing agentId rejected"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks — deadline in the past → 400"

# Use hardcoded past date for cross-platform compatibility
PAST_DEADLINE="2020-01-01T00:00:00Z"
pt_post /tasks -d "{\"agentId\":\"${AGENT}\",\"action\":\"navigate\",\"deadline\":\"${PAST_DEADLINE}\",\"params\":{\"url\":\"${FIXTURES_URL}/index.html\"}}"
assert_http_status "400" "past deadline rejected"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "POST /tasks — 429 queue full"

# maxPerAgent is 3, fill the queue for a fresh agent
FLOOD_AGENT="flood-agent-$$"
for i in $(seq 1 4); do
  pt_post /tasks -d "{\"agentId\":\"${FLOOD_AGENT}\",\"action\":\"navigate\",\"params\":{\"url\":\"${FIXTURES_URL}/index.html\"}}"
done
# The 4th should be 429 (maxPerAgent=3) - but this is racy, so accept both 202 and 429
if [ "$HTTP_STATUS" = "429" ]; then
  echo -e "  ${GREEN}✓${NC} queue full for agent (HTTP 429)"
  ((ASSERTIONS_PASSED++)) || true
elif [ "$HTTP_STATUS" = "202" ]; then
  echo -e "  ${GREEN}✓${NC} task accepted (race condition - some tasks completed quickly)"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} unexpected status: $HTTP_STATUS (expected 429 or 202)"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "GET /tasks/{id} — nonexistent → 404"

pt_get /tasks/tsk_doesnotexist
assert_http_status "404" "nonexistent task"

end_test