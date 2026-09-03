#!/usr/bin/env bash

set -euo pipefail

# Setup directories and load common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

check_dependencies

# Use environment variable or default for port/base URL
DEFAULT_PORT="${PORT_NGO_SERVICE:-8081}"
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

# 2. Create NGO
NGO_EMAIL="check-script-$(random_string)@solidarytech.org"

echo "Creating NGO '$NGO_EMAIL'..."
resp=$(call POST "$BASE_URL/ngos" \
  "{\"name\":\"Check Script NGO\",\"email\":\"$NGO_EMAIL\",\"cause\":\"Educação\",\"city\":\"São Paulo\"}")
print_response "$(response_status "$resp")" "$(response_body "$resp")"

if [[ "$(response_status "$resp")" -ne 201 ]]; then
  echo "NGO creation failed"
  exit 1
fi

# 3. List NGOs and confirm the new one is present
echo "Listing all NGOs..."
resp=$(call GET "$BASE_URL/ngos")
body=$(response_body "$resp")
print_response "$(response_status "$resp")" "$body"

if [[ "$(response_status "$resp")" -ne 200 ]]; then
  echo "NGO listing failed"
  exit 1
fi

if ! echo "$body" | jq -e --arg email "$NGO_EMAIL" '[.[] | select(.email == $email)] | length > 0' > /dev/null; then
  echo "Created NGO not found in listing"
  exit 1
fi

echo "NGO service checks completed successfully"
