#!/usr/bin/env bash
set -euo pipefail

# Config
BASE_URL="http://localhost:8080/api/admin/auth"

# Dependencies
if ! command -v jq &>/dev/null; then
  echo "Error: jq is required. Install it and re-run." >&2
  exit 1
fi

# Prompt
read -p "Enter email for login: " EMAIL
read -s -p "Enter password: " PASSWORD
echo

# Build payload
PAYLOAD=$(jq -n --arg e "$EMAIL" --arg p "$PASSWORD" '{email: $e, password: $p}')

# Call login
HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

BODY=$(printf "%s\n" "$HTTP_RESPONSE" | sed '$d')
CODE=$(printf "%s\n" "$HTTP_RESPONSE" | tail -n1)

echo "Status: $CODE"
echo "$BODY" | jq .

if [[ "$CODE" -ne 200 ]]; then
  echo "✖ Login failed (HTTP $CODE)" >&2
  exit 1
fi

# Extract and export token
TOKEN=$(echo "$BODY" | jq -r .token)
echo "✅ Login successful."
echo "To export your token for downstream calls, run:"
echo "  export TOKEN=$TOKEN"

