#!/usr/bin/env bash
set -euo pipefail

# ─── CONFIG ────────────────────────────────────────────────────────────────────
# default to /api/admin where RegisterPlaylistRoutes() is mounted
API_BASE="${API_BASE:-http://localhost:8080/api/admin}"
HAS_JQ=false
if command -v jq &>/dev/null; then HAS_JQ=true; fi

# ensure JWT is set
if [[ -z "${JWT:-}" ]]; then
  echo "⚠️  Please export your JWT as the environment variable \$JWT"
  exit 1
fi

# ─── UTILITY ───────────────────────────────────────────────────────────────────
# $1 = HTTP method, $2 = path, $3 = extra curl args (e.g. -d '…')
function call_api() {
  local method="$1"
  local path="$2"
  local extra="$3"
  local url="${API_BASE}${path}"
  local tmp status body

  tmp="$(mktemp)"
  status=$(
    curl -s -X "$method" \
      -H "Authorization: Bearer $JWT" \
      -H "Content-Type: application/json" \
      $extra \
      -o "$tmp" -w '%{http_code}' \
      "$url"
  )
  body="$(<"$tmp")"
  rm "$tmp"

  echo "HTTP $status"
  if [[ -n "$body" ]]; then
    if $HAS_JQ && [[ "$body" =~ ^\{.*\}$ || "$body" =~ ^\[.*\]$ ]]; then
      echo "$body" | jq .
    else
      echo "$body"
    fi
  fi
}

function pause() {
  read -rp $'\nPress [ENTER] to continue…'
}

# ─── ENDPOINT FUNCTIONS ────────────────────────────────────────────────────────
function list_playlists()     { call_api GET    "/playlists"              ""; }
function get_playlist()       { read -rp "Playlist ID: " id; call_api GET    "/playlists/$id"         ""; }
function delete_playlist()    { read -rp "Playlist ID: " id; call_api DELETE "/playlists/$id"         ""; }

function create_playlist() {
  read -rp "Name: " name
  read -rp "Description: " desc
  local payload
  payload=$(jq -nc --arg n "$name" --arg d "$desc" '{name:$n,description:$d}')
  call_api POST "/playlists" "-d '$payload'"
}

function update_playlist() {
  read -rp "Playlist ID: " id
  read -rp "New name: " name
  read -rp "New description: " desc
  local payload
  payload=$(jq -nc --arg n "$name" --arg d "$desc" '{name:$n,description:$d}')
  call_api PUT "/playlists/$id" "-d '$payload'"
}

function add_item() {
  read -rp "Playlist ID: " pid
  read -rp "Content ID: " cid
  read -rp "Duration (s): " dur
  local payload
  payload=$(jq -nc --argjson c "$cid" --argjson d "$dur" '{content_id:$c,duration:$d}')
  call_api POST "/playlists/$pid/items" "-d '$payload'"
}

function update_item() {
  read -rp "Playlist ID: " pid
  read -rp "Item ID: " iid
  read -rp "Position: " pos
  read -rp "Duration: " dur
  local payload
  payload=$(jq -nc --argjson p "$pos" --argjson d "$dur" '{position:$p,duration:$d}')
  call_api PUT "/playlists/$pid/items/$iid" "-d '$payload'"
}

function remove_item() {
  read -rp "Playlist ID: " pid
  read -rp "Item ID: " iid
  call_api DELETE "/playlists/$pid/items/$iid" ""
}

function list_items() {
  read -rp "Playlist ID: " pid
  call_api GET "/playlists/$pid/items" ""
}

function reorder_items() {
  read -rp "Playlist ID: " pid
  echo "Enter new item IDs order (space-separated):"
  read -ra ids
  local arr
  arr=$(printf '%s\n' "${ids[@]}" | jq -R . | jq -s .)
  call_api PUT "/playlists/$pid/items" "-d '{\"item_ids\":$arr}'"
}

function add_integration() {
  read -rp "Playlist ID: " pid
  read -rp "Integration name: " name
  read -rp "Duration (s, blank=default): " dur
  read -rp "Position (blank=append): " pos

  local obj='{"integration_name":"'"$name"'"}'
  [[ -n "$dur" ]] && obj=$(jq -nc --argjson d "$dur" '$ARGS.positional|. + {"duration":$d}' "$obj")
  [[ -n "$pos" ]] && obj=$(jq -nc --argjson p "$pos" '$ARGS.positional|. + {"position":$p}' "$obj")

  call_api POST "/playlists/$pid/integrations" "-d '$obj'"
}

# ─── MAIN MENU ─────────────────────────────────────────────────────────────────
while true; do
  echo
  echo "=== Playlist API Tester (@ $API_BASE) ==="
  PS3="Select: "
  options=(
    "List Playlists" "Create Playlist" "Get Playlist" "Update Playlist"
    "Delete Playlist" "Add Item" "Update Item" "Remove Item"
    "List Items" "Reorder Items" "Add Integration" "Quit"
  )
  select opt in "${options[@]}"; do
    case $opt in
      "List Playlists")   list_playlists;;  
      "Create Playlist")  create_playlist;; 
      "Get Playlist")     get_playlist;;    
      "Update Playlist")  update_playlist;;
      "Delete Playlist")  delete_playlist;;
      "Add Item")         add_item;;       
      "Update Item")      update_item;;    
      "Remove Item")      remove_item;;    
      "List Items")       list_items;;     
      "Reorder Items")    reorder_items;;  
      "Add Integration")  add_integration;;
      "Quit")             echo "Bye!"; exit 0;;
      *)                  echo "Invalid choice";;
    esac
    break
  done
  pause
done

