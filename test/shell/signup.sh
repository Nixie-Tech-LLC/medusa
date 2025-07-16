#!/usr/bin/env bash
set -euo pipefail

# Config
BASE_URL="http://localhost:8080/api/admin/auth"
DEFAULT_NAME="Adam Younes"

# Dependencies
if ! command -v jq &>/dev/null; then
  echo "Error: jq is required. Install it and re-run." >&2
  exit 1
fi

# Prompt
read -p "Enter email for signup: " EMAIL
read -s -p "Enter password: " PASSWORD
echo

# Build payload
PAYLOAD=$(jq -n \
  --arg e "$EMAIL" \
  --arg p "$PASSWORD" \
  --arg n "$DEFAULT_NAME" \
  '{email: $e, password: $p, name: $n}')

# Call signup
HTTP_RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST "$BASE_URL/signup" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")

BODY=$(printf "%s\n" "$HTTP_RESPONSE" | sed '$d')
CODE=$(printf "%s\n" "$HTTP_RESPONSE" | tail -n1)

echo "Status: $CODE"
echo "$BODY" | jq .

if [[ "$CODE" -ne 200 ]]; then
  echo "✖ Signup failed (HTTP $CODE)" >&2
  exit 1
fi

# Extract and export token
TOKEN=$(echo "$BODY" | jq -r .token)
echo "✅ Signup successful."
echo "To export your token for downstream calls, run:"
echo "  export TOKEN=$TOKEN"

