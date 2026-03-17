#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
COOKIE_JAR="${COOKIE_JAR:-/tmp/gin_auth_cookie.txt}"

email="user_$(date +%s)_$RANDOM@example.com"
username="user_$RANDOM"
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

echo "== Register"
register_body="$(printf '{"username":"%s","email":"%s","password":"%s"}' "$username" "$email" "$password")"
register_status="$(request POST /api/register "$register_body" | head -n 1)"
echo "register status: $register_status"

echo "== Login"
login_body="$(printf '{"email":"%s","password":"%s"}' "$email" "$password")"
login_status="$(request POST /api/login "$login_body" | head -n 1)"
echo "login status: $login_status"

echo "== Me (should be 200)"
me_status="$(request GET /api/me | head -n 1)"
echo "me status: $me_status"

echo "== Logout"
logout_status="$(request POST /api/logout | head -n 1)"
echo "logout status: $logout_status"

echo "== Me after logout (should be 302)"
me_after_logout_status="$(request GET /api/me | head -n 1)"
echo "me after logout status: $me_after_logout_status"

echo "== Invalid login (should be 401)"
bad_login_body="$(printf '{"email":"%s","password":"%s"}' "$email" "wrongpass")"
bad_login_status="$(request POST /api/login "$bad_login_body" | head -n 1)"
echo "bad login status: $bad_login_status"
