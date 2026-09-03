#!/usr/bin/env bash

set -euo pipefail

# check_dependencies: ensures curl and jq are installed.
check_dependencies() {
  local missing=()

  command -v curl > /dev/null 2>&1 || missing+=("curl")
  command -v jq   > /dev/null 2>&1 || missing+=("jq")

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Missing required dependencies: ${missing[*]}"
    echo "Please install them before proceeding."
    exit 1
  fi
}

# call <METHOD> <URL> [data] [header1] [header2] ...
#
# Performs an HTTP request and returns "STATUS|BODY".
#
# Parameters:
#   $1  - HTTP method (GET, POST, PUT, DELETE...)
#   $2  - full URL
#   $3  - (optional) JSON body; pass "" to omit
#   $4+ - (optional) extra headers in "Key: Value" format
call() {
  local method="$1"; shift
  local url="$1";    shift
  local data="${1:-}"; [[ $# -gt 0 ]] && shift || true

  local -a extra_headers=()
  for h in "$@"; do
    extra_headers+=(--header "$h")
  done

  local -a curl_args=(
    --silent --show-error
    --write-out "HTTPSTATUS:%{http_code}"
    --location
    -X "$method" "$url"
  )

  if [[ -n "$data" ]]; then
    curl_args+=(--header "Content-Type: application/json" --data "$data")
  fi

  local response
  response=$(curl "${curl_args[@]}" "${extra_headers[@]}")

  local body status
  body=$(echo "$response"   | sed -e 's/HTTPSTATUS\:.*//g')
  status=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

  echo "${status}|${body}"
}

# print_response <STATUS> <BODY>: prints colored status and pretty-printed body (JSON or plain text).
print_response() {
  local status="$1"
  local body="$2"

  if   [[ "$status" =~ ^2 ]]; then
    printf "Status: \033[32m%s\033[0m\n" "$status"
  elif [[ "$status" =~ ^4 ]]; then
    printf "Status: \033[34m%s\033[0m\n" "$status"
  elif [[ "$status" =~ ^5 ]]; then
    printf "Status: \033[31m%s\033[0m\n" "$status"
  else
    printf "Status: \033[33m%s\033[0m\n" "$status"
  fi

  if [[ -z "$body" ]]; then
    echo "No body"
  elif echo "$body" | jq . > /dev/null 2>&1; then
    echo "$body" | jq .
  else
    echo "$body"
  fi

  echo "---------------------------------------------"
}

# response_status <resp>: extracts the HTTP status code from a call() response.
response_status() { echo "$1" | cut -d'|' -f1; }

# response_body <resp>: extracts the body from a call() response.
response_body()   { echo "$1" | cut -d'|' -f2-; }

# random_string: returns 8 random hex characters.
random_string() {
  date +%s%N | sha256sum | head -c8
}
