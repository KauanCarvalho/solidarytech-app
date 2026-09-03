#!/usr/bin/env bash

set -euo pipefail

# Setup directories and load common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

check_dependencies

# Use environment variable or default for port/base URL
DEFAULT_PORT="${PORT_DONATION_SERVICE:-8082}"
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

# 2. Create donation (hot path / caminho crítico)
DONOR_NAME="check-script-$(random_string)"

echo "Creating donation from '$DONOR_NAME'..."
resp=$(call POST "$BASE_URL/donations" \
  "{\"ngo_id\":1,\"amount\":50.00,\"donor_name\":\"$DONOR_NAME\"}")
print_response "$(response_status "$resp")" "$(response_body "$resp")"

if [[ "$(response_status "$resp")" -ne 201 ]]; then
  echo "Donation creation failed"
  exit 1
fi

# 3. List donations and confirm the new one is present
echo "Listing all donations..."
resp=$(call GET "$BASE_URL/donations")
body=$(response_body "$resp")
print_response "$(response_status "$resp")" "$body"

if [[ "$(response_status "$resp")" -ne 200 ]]; then
  echo "Donation listing failed"
  exit 1
fi

if ! echo "$body" | jq -e --arg donor "$DONOR_NAME" '[.[] | select(.donor_name == $donor)] | length > 0' > /dev/null; then
  echo "Created donation not found in listing"
  exit 1
fi

echo "Donation service checks completed successfully"
