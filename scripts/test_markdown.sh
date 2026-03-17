#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/gin_auth_cookie_md.txt}"

email="md_user_$(date +%s)_$RANDOM@example.com"
username="md_user_$RANDOM"
password="password123"

request() {
  local method="$1"
  local path="$2"
  local data="${3:-}"
  local tmp_body
  tmp_body="$(mktemp)"
  local status

  if [[ -n "$data" ]]; then
    status="$(curl -s -o "$tmp_body" -w "%{http_code}" \
      -X "$method" \
      -H "Content-Type: application/json" \
      -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
      "$BASE_URL$path" \
      -d "$data")"
  else
    status="$(curl -s -o "$tmp_body" -w "%{http_code}" \
      -X "$method" \
      -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
      "$BASE_URL$path")"
  fi

  echo "$status"
  cat "$tmp_body"
  rm -f "$tmp_body"
}

json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf "%s" "$s"
}

echo "== Register"
register_body="$(printf '{"username":"%s","email":"%s","password":"%s"}' "$username" "$email" "$password")"
register_status="$(request POST /api/register "$register_body" | head -n 1)"
echo "register status: $register_status"

echo "== Login"
login_body="$(printf '{"email":"%s","password":"%s"}' "$email" "$password")"
login_status="$(request POST /api/login "$login_body" | head -n 1)"
echo "login status: $login_status"

echo "== Submit Markdown"
title="Demo Note $RANDOM"
content="# 小标题\n\n这是一个测试提交。\n\n- item 1\n- item 2"
markdown_body="$(printf '{"title":"%s","content":"%s"}' "$(json_escape "$title")" "$(json_escape "$content")")"
submit_response="$(request POST /api/markdown "$markdown_body")"
submit_status="$(printf "%s\n" "$submit_response" | head -n 1)"
submit_body="$(printf "%s\n" "$submit_response" | sed '1d')"
echo "submit status: $submit_status"

if [[ "$submit_status" != "201" ]]; then
  echo "submit body: $submit_body"
  exit 1
fi

entry_id="$(printf "%s\n" "$submit_body" | sed -E -n 's/.*"id":[[:space:]]*([0-9]+).*/\1/p')"
if [[ -z "$entry_id" ]]; then
  echo "failed to parse entry id"
  echo "submit body: $submit_body"
  exit 1
fi

echo "== List Markdown"
list_status="$(request GET "/api/markdown?limit=5" | head -n 1)"
echo "list status: $list_status"

echo "== Read Markdown"
read_status="$(request GET "/api/markdown/$entry_id" | head -n 1)"
echo "read status: $read_status"
