#!/usr/bin/env bash
# ============================================================================
# Plane API helpers — talk to a self-hosted Plane instance from the shell.
#   source scripts/plane-api.sh
#   plane_projects                                  # list id / identifier / name
#   plane_issues  "My Project"                      # list a project's work items
#   plane_issue   "My Project" "title" "desc"       # create a work item
#   plane_proj_desc "My Project" "new description"  # set project description
#   plane_pages_sync /path/to/pages.json            # create/update wiki pages (ORM)
#   plane_curl GET /api/v1/workspaces/<slug>/projects/   # raw call
#
# Config (env or a secrets file sourced below — never commit the token):
#   PLANE_API_TOKEN   workspace API token  ("plane_api_…", from Settings → API Tokens)
#   PLANE_WS_SLUG     workspace slug        (default: first arg / "default")
#   PLANE_BASE_URL    base URL              (default: http://127.0.0.1:8000 — loopback)
#   PLANE_ENV         path to a secrets file to source (default: ~/.plane-secrets.env)
#   PLANE_API_DIR     Plane apps/api dir on the host (for plane_pages_sync ORM tool)
#
# ===================== HOW PLANE'S API WORKS (so you don't read the source) =====
# TWO separate APIs:
#   1) PUBLIC REST   /api/v1/...     auth header:  X-Api-Key: $PLANE_API_TOKEN
#        covers: workspaces, projects, ISSUES (= "work items"), cycles, modules,
#                states, labels, members, issue links/comments/attachments.
#                NOT pages, NOT instance config.
#   2) APP API       /api/workspaces/...   session-cookie auth (what the UI uses).
#        covers pages + everything else. Needs a logged-in Plane session.
#
# PAGES are not in the public API → create them via Plane's Django ORM on the host
#   (Page + ProjectPage; set description_html; the `live`/hocuspocus service rebuilds
#    the collab binary from the HTML on first open). Tool: scripts/plane-orm-page.py
#
# GOTCHAS:
#   * On macOS, python/requests/urllib honour the *system* proxy (System Settings →
#     Network → Proxies). If that proxy is down, even loopback calls fail with
#     ProxyError. We export no_proxy=* below so the helpers bypass it. (curl ignores
#     the system proxy already.)
#   * Project NAME cannot contain special characters — Plane rejects '-'. Use spaces.
#   * Public list endpoints use CURSOR pagination: ?per_page=100&cursor=… ; loop while
#     the JSON has next_page_results, advancing cursor=next_cursor.
#   * On the Plane host use loopback http://127.0.0.1:8000 (no TLS, no proxy);
#     off-host use your public https URL (nginx routes /api/ → the api port).
#
# ENDPOINT CHEAT-SHEET (public, X-Api-Key):
#   GET   /api/v1/workspaces/<slug>/projects/
#   PATCH /api/v1/workspaces/<slug>/projects/<pid>/            {name?, description?}
#   GET   /api/v1/workspaces/<slug>/projects/<pid>/issues/?per_page=100
#   POST  /api/v1/workspaces/<slug>/projects/<pid>/issues/     {name, description_html, priority?}
#   Mint a token without the UI (Django shell):
#     APIToken.objects.create(user=ws.owner, workspace=ws, label="…").token  # "plane_api_…"
# ============================================================================

set -a; . "${PLANE_ENV:-$HOME/.plane-secrets.env}" 2>/dev/null; set +a
export no_proxy='*' NO_PROXY='*'
export PLANE_BASE="${PLANE_BASE_URL:-http://127.0.0.1:8000}"
export PLANE_WS_SLUG="${PLANE_WS_SLUG:-default}"
PLANE_API_DIR="${PLANE_API_DIR:-/opt/plane/apps/api}"

plane_curl() { # METHOD PATH [JSON]
  local m="$1" p="$2" b="$3"
  if [ -n "$b" ]; then
    curl -s -X "$m" "$PLANE_BASE$p" -H "X-Api-Key: $PLANE_API_TOKEN" \
         -H "Content-Type: application/json" -d "$b" --max-time 30
  else
    curl -s -X "$m" "$PLANE_BASE$p" -H "X-Api-Key: $PLANE_API_TOKEN" --max-time 30
  fi
}

plane_pid() { # NAME -> id
  plane_curl GET "/api/v1/workspaces/$PLANE_WS_SLUG/projects/?per_page=100" \
   | python3 -c "import sys,json;d=json.load(sys.stdin);r=d.get('results',d);print(next((p['id'] for p in r if p['name']==sys.argv[1]),''))" "$1"
}

plane_projects() {
  plane_curl GET "/api/v1/workspaces/$PLANE_WS_SLUG/projects/?per_page=100" \
   | python3 -c "import sys,json;d=json.load(sys.stdin);[print(p['id'],p['identifier'],'|',p['name']) for p in d.get('results',d)]"
}

plane_issues() { # NAME
  local id; id=$(plane_pid "$1"); [ -z "$id" ] && { echo "no project: $1"; return 1; }
  plane_curl GET "/api/v1/workspaces/$PLANE_WS_SLUG/projects/$id/issues/?per_page=100" \
   | python3 -c "import sys,json;d=json.load(sys.stdin);[print(i.get('sequence_id'),'|',i.get('name')) for i in d.get('results',d)]"
}

plane_issue() { # NAME TITLE DESC
  local id; id=$(plane_pid "$1"); [ -z "$id" ] && { echo "no project: $1"; return 1; }
  local body; body=$(python3 -c "import json,sys;print(json.dumps({'name':sys.argv[1],'description_html':'<p>'+sys.argv[2]+'</p>'}))" "$2" "${3:-}")
  plane_curl POST "/api/v1/workspaces/$PLANE_WS_SLUG/projects/$id/issues/" "$body" \
   | python3 -c "import sys,json;d=json.load(sys.stdin);print('created' if d.get('id') else d, d.get('sequence_id',''))"
}

plane_proj_desc() { # NAME DESC
  local id; id=$(plane_pid "$1"); [ -z "$id" ] && { echo "no project: $1"; return 1; }
  local body; body=$(python3 -c "import json,sys;print(json.dumps({'description':sys.argv[1]}))" "$2")
  plane_curl PATCH "/api/v1/workspaces/$PLANE_WS_SLUG/projects/$id/" "$body" >/dev/null && echo "ok: $1"
}

plane_pages_sync() { # JSON {"<project-name-hyphenated>":"<html>"}
  local json="${1:?usage: plane_pages_sync <pages.json>}"
  local here; here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  bash -lc "cd '$PLANE_API_DIR' && source .venv/bin/activate && set -a && . ./.env && set +a && \
    export no_proxy='*' NO_PROXY='*' PLANE_PAGES_JSON='$json' PLANE_WS_SLUG='$PLANE_WS_SLUG' && \
    python manage.py shell -c \"exec(open('$here/plane-orm-page.py').read())\""
}
