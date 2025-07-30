#!/usr/bin/env bash
set -euo pipefail

# Config
BASE_URL="http://localhost:8080/api/admin/auth"

# Dependencies
if ! command -v jq &>/dev/null; then
  echo "Error: jq is required. Install it and re-run." >&2
  exit 1
fi

# Ensure TOKEN is set
if [[ -z "${TOKEN-:-}" ]]; then
  echo "Error: TOKEN environment variable not set." >&2
  echo "Run signup.sh or login.sh and export the TOKEN first." >&2
  exit 1
fi

# Call get current profile
HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X GET "$BASE_URL/current_profile" \
  -H "Authorization: Bearer $TOKEN")

BODY=$(printf "%s\n" "$HTTP_RESPONSE" | sed '$d')
CODE=$(printf "%s\n" "$HTTP_RESPONSE" | tail -n1)

echo "Status: $CODE"
echo "$BODY" | jq .

