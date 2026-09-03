#!/usr/bin/env bash

set -euo pipefail

# Setup directories and load common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

check_dependencies

# Use environment variable or default for port/base URL
DEFAULT_PORT="${PORT_VOLUNTEER_SERVICE:-8083}"
DEFAULT_BASE_URL="http://localhost:${DEFAULT_PORT}"
BASE_URL="${1:-$DEFAULT_BASE_URL}"

echo "Using base URL: $BASE_URL"
echo "---------------------------------------------"

# 1. Health check
echo "Checking health endpoint..."

resp=$(call GET "$BASE_URL/health")
status=$(response_status "$resp")
body=$(response_body "$resp")

print_response "$status" "$body"

if [[ "$status" -ne 200 ]]; then
  echo "Health check failed"
  exit 1
fi

# 2. Register volunteer for NGO id 1
VOLUNTEER_EMAIL="check-script-$(random_string)@solidarytech.org"

echo "Registering volunteer '$VOLUNTEER_EMAIL'..."
resp=$(call POST "$BASE_URL/volunteers" \
  "{\"name\":\"Check Script Volunteer\",\"email\":\"$VOLUNTEER_EMAIL\",\"ngo_id\":1}")
print_response "$(response_status "$resp")" "$(response_body "$resp")"

if [[ "$(response_status "$resp")" -ne 201 ]]; then
  echo "Volunteer registration failed"
  exit 1
fi

# 3. List volunteers by ngo_id and confirm the new one is present
echo "Listing volunteers for ngo_id=1..."
resp=$(call GET "$BASE_URL/volunteers/1")
body=$(response_body "$resp")
print_response "$(response_status "$resp")" "$body"

if [[ "$(response_status "$resp")" -ne 200 ]]; then
  echo "Volunteer listing failed"
  exit 1
fi

if ! echo "$body" | jq -e --arg email "$VOLUNTEER_EMAIL" '[.[] | select(.email == $email)] | length > 0' > /dev/null; then
  echo "Registered volunteer not found in listing"
  exit 1
fi

echo "Volunteer service checks completed successfully"
