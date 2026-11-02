#!/usr/bin/env bash
# ==============================================================================
# task-api Smoke Test Script
# Memverifikasi endpoint penting pada server yang sedang berjalan:
#   1. GET  /health   -> 200 OK
#   2. POST /register -> 201 Created (user random)
#   3. POST /login    -> 200 OK (ambil bearer token)
#   4. POST /tasks    -> 201 Created (buat task dengan auth)
#   5. GET  /tasks    -> 200 OK (ambil daftar task dengan auth)
# ==============================================================================
set -euo pipefail

BASE_URL="${API_URL:-http://127.0.0.1:8081}"

# ANSI Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

log_pass() {
  echo -e "${GREEN}[PASS]${NC} $1"
}

log_fail() {
  echo -e "${RED}[FAIL]${NC} $1" >&2
}

assert_status() {
  local expected="$1"
  local actual="$2"
  local step_name="$3"
  local body="$4"

  if [ "$actual" -eq "$expected" ]; then
    log_pass "$step_name (HTTP $actual)"
  else
    log_fail "$step_name: expected HTTP $expected, got HTTP $actual"
    echo -e "${RED}Response Body:${NC}\n$body" >&2
    exit 1
  fi
}

extract_token() {
  local json="$1"
  if command -v jq >/dev/null 2>&1; then
    echo "$json" | jq -r '.token // empty'
  else
    # Fallback jika jq tidak terinstall
    echo "$json" | grep -o '"token":"[^"]*"' | cut -d'"' -f4
  fi
}

echo "=========================================================="
echo " Starting task-api Smoke Tests on $BASE_URL"
echo "=========================================================="

# 1. Healthcheck
log_info "Step 1: Checking health endpoint (/health)..."
HEALTH_RESP=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/health")
HEALTH_CODE=$(echo "$HEALTH_RESP" | tail -n1)
HEALTH_BODY=$(echo "$HEALTH_RESP" | sed '$d')
assert_status 200 "$HEALTH_CODE" "Health check" "$HEALTH_BODY"

# 2. Register random user
RANDOM_ID="${RANDOM}_$(date +%s%N 2>/dev/null || date +%s)"
SMOKE_EMAIL="smoke_${RANDOM_ID}@example.com"
SMOKE_PASSWORD="SmokePassword123!"

log_info "Step 2: Registering temporary user ($SMOKE_EMAIL)..."
REGISTER_PAYLOAD=$(cat <<EOF
{
  "email": "$SMOKE_EMAIL",
  "password": "$SMOKE_PASSWORD"
}
EOF
)

REG_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/register" \
  -H "Content-Type: application/json" \
  -d "$REGISTER_PAYLOAD")
REG_CODE=$(echo "$REG_RESP" | tail -n1)
REG_BODY=$(echo "$REG_RESP" | sed '$d')
assert_status 201 "$REG_CODE" "User registration" "$REG_BODY"

# 3. Login
log_info "Step 3: Logging in with registered user..."
LOGIN_PAYLOAD=$(cat <<EOF
{
  "email": "$SMOKE_EMAIL",
  "password": "$SMOKE_PASSWORD"
}
EOF
)

LOGIN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "$LOGIN_PAYLOAD")
LOGIN_CODE=$(echo "$LOGIN_RESP" | tail -n1)
LOGIN_BODY=$(echo "$LOGIN_RESP" | sed '$d')
assert_status 200 "$LOGIN_CODE" "User login" "$LOGIN_BODY"

TOKEN=$(extract_token "$LOGIN_BODY")
if [ -z "$TOKEN" ]; then
  log_fail "Failed to extract JWT token from login response"
  echo "$LOGIN_BODY" >&2
  exit 1
fi
log_pass "JWT token acquired successfully"

# 4. Create Task
log_info "Step 4: Creating a new task (/tasks)..."
TASK_PAYLOAD=$(cat <<EOF
{
  "title": "Smoke Test Task $RANDOM_ID",
  "description": "Automated verification task",
  "status": "todo",
  "priority": "medium",
  "deadline": "2026-12-31"
}
EOF
)

CREATE_TASK_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/tasks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$TASK_PAYLOAD")
CREATE_TASK_CODE=$(echo "$CREATE_TASK_RESP" | tail -n1)
CREATE_TASK_BODY=$(echo "$CREATE_TASK_RESP" | sed '$d')
assert_status 201 "$CREATE_TASK_CODE" "Create task" "$CREATE_TASK_BODY"

# 5. List Tasks
log_info "Step 5: Fetching tasks list (/tasks)..."
LIST_TASK_RESP=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/tasks" \
  -H "Authorization: Bearer $TOKEN")
LIST_TASK_CODE=$(echo "$LIST_TASK_RESP" | tail -n1)
LIST_TASK_BODY=$(echo "$LIST_TASK_RESP" | sed '$d')
assert_status 200 "$LIST_TASK_CODE" "List tasks" "$LIST_TASK_BODY"

echo "=========================================================="
log_pass "ALL SMOKE TESTS COMPLETED SUCCESSFULLY"
echo "=========================================================="
