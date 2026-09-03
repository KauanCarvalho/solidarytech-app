#!/usr/bin/env bash

set -euo pipefail

# Setup directories and load common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/_common.sh"

echo "================================================================"
echo "   SolidaryTech - Global Ecosystem Validation"
echo "================================================================"

# List of services to validate in logical order
SERVICES=(
  "ngo-service"
  "donation-service"
  "volunteer-service"
)

TOTAL_SERVICES=${#SERVICES[@]}
SUCCESS_COUNT=0

# Iterate through services and run individual check scripts
for i in "${!SERVICES[@]}"; do
  SVC="${SERVICES[$i]}"
  echo ""
  echo "[$((i+1))/$TOTAL_SERVICES] Checking: $SVC..."
  echo "----------------------------------------------------------------"

  # Format variable name (e.g. NGO_SERVICE_URL from ngo-service)
  VAR_NAME="$(echo "${SVC}" | tr '[:lower:]' '[:upper:]' | tr '-' '_')_URL"
  SVC_URL="${!VAR_NAME:-}"

  if "$SCRIPT_DIR/$SVC.sh" "$SVC_URL"; then
    echo "SUCCESS: $SVC is operational"
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  else
    echo "FAILURE: $SVC check failed"
    echo ""
    echo "Verification interrupted due to failure in $SVC."
    exit 1
  fi
done

echo ""
echo "================================================================"
echo "   All $SUCCESS_COUNT services are operational"
echo "================================================================"
