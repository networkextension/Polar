# Deploying Polar

End-to-end guide for standing up the Polar stack on a fresh box. Covers macOS (launchd), Linux (systemd), and FreeBSD (rc.d). The walk-through assumes you control DNS for one domain (we use `example.com` as a placeholder; the production reference deployment uses `4950.store`).

If you only need a single piece — say, adding one plugin to an existing dock — jump to the [Plugin install](#plugin-install) section.

## Topology in one picture

```
                          ┌──────────────────────────┐
   user / iOS / Android   │  *.example.com (wildcard)│
        browser ──────────│  → public IP : 2443      │
                          │     (router hairpin to   │
                          │      LAN host : 443)     │
                          └────────────┬─────────────┘
                                       │
                          ┌────────────▼─────────────┐
                          │   nginx (one vhost per   │
                          │   plugin subdomain)      │
                          └────────────┬─────────────┘
                                       │
       ┌───────────────────────────────┼───────────────────────────────┐
       │                               │                               │
       ▼                               ▼                               ▼
┌──────────────┐               ┌───────────────┐               ┌──────────────┐
│  polar-dock  │  HMAC-signed  │  polar-wg-svc │   …8 more     │ polar-*-svc  │
│  :8080       │◄──────────────│  :8090        │   plugins     │ :8091..8098  │
│  identity +  │  /internal/v1 │  WireGuard    │               │              │
│  LLM proxy + │               │  control plane│               │              │
│  agent_hub   │               │               │               │              │
└──────┬───────┘               └───────┬───────┘               └──────┬───────┘
       │                               │                               │
       ▼                               ▼                               ▼
   ideamesh                       polar_wg                       polar_<svc>
   (Postgres)                     (Postgres)                     (Postgres)
   + redis
```

Every plugin is a tiny Go binary that talks to dock via signed HTTP for auth and to its own per-plugin Postgres database for state. Browsers reach each plugin under its own subdomain (`wg.example.com`, `project.example.com`, …), all sharing a wildcard cert.

The control plane (`polar-dock`) is currently closed — see the [front-door README](../README.md) for status. The deployment guide below describes the stack the **plugins** need, plus the dock-side hooks each plugin assumes are present.

## Prerequisites

- **Go 1.25+** on the build machine (not needed on production hosts — binaries are statically linked, `CGO_ENABLED=0`).
- **Postgres 15+** reachable from every plugin host (one logical database per plugin: `polar_<svc>`).
- **Redis 7+** reachable from dock (used for session + rate-limit state).
- **nginx 1.27+** on the host that fronts every subdomain.
- **DNS control** for the wildcard. The reference deployment uses Let's Encrypt **DNS-01** for `*.example.com`; the umbrella ships `scripts/dns/name-com-records.sh` to manage records via the name.com v4 API. Adapt to your provider as needed.
- **A non-root user** to run each daemon (`polar` on Linux/FreeBSD; the operator's own account on macOS).

## DNS + cert

Add one wildcard A record pointing at the public IP (or :2443 if your router does port translation):

```
*.example.com.   300   IN   A   <public-ip>
```

Then mint a wildcard cert. With acme.sh + Cloudflare for example:

```bash
acme.sh --issue --dns dns_cf -d example.com -d '*.example.com' --keylength ec-256
acme.sh --install-cert -d example.com \
    --fullchain-file /etc/letsencrypt/live/example.com/fullchain.pem \
    --key-file       /etc/letsencrypt/live/example.com/privkey.pem
```

(certbot manual mode also works; the reference deployment uses `certbot --manual --preferred-challenges dns-01` because the operator's DNS provider doesn't have an acme.sh plugin.)

## Cross-platform binaries

The umbrella ships a generic release builder. From a checkout of this repo:

```bash
./scripts/build-release.sh /path/to/polar-<svc>
```

It produces six tarballs in `<svc>/dist/`:

```
<svc>-<version>-macos-apple-silicon.tar.gz
<svc>-<version>-macos-intel.tar.gz
<svc>-<version>-linux-amd64.tar.gz
<svc>-<version>-linux-arm64.tar.gz
<svc>-<version>-freebsd-amd64.tar.gz
<svc>-<version>-freebsd-arm64.tar.gz
```

Each tarball contains:

```
<svc>-<version>/
  bin/<svc>-svc                        statically-linked Go binary
  etc/launchd/polar.<svc>-svc.plist    macOS launchd plist + wrapper
  etc/launchd/polar-<svc>-svc-launch.sh
  etc/systemd/polar-<svc>-svc.service  Linux systemd unit
  etc/rc.d/polar_<svc>_svc             FreeBSD rc.d script
  etc/nginx/<svc>-svc-snippet.conf     nginx include for the dock vhost
  migrations/<svc>-schema.sql          one-shot DDL
  env/<svc>-svc.env.sample             env vars the binary reads
  install.sh                           OS-detecting copier
  README.md
```

The builder is idempotent and re-runnable; bumping a release means tagging the plugin repo and re-running the script.

## Plugin install

On the target host (Linux/FreeBSD/macOS), download or rsync the right tarball, extract, and run `install.sh` as root:

```bash
tar xzf polar-wg-v0.4.0-linux-amd64.tar.gz
cd polar-wg-v0.4.0
sudo ./install.sh
```

This copies the binary to `/usr/local/bin/`, the service file to the conventional location, the nginx snippet, and a starter env file (mode 0600) you must edit. It does NOT start the service, NOT create the database, and NOT mint the dock plugin token — those are deliberate manual steps.

Concrete next steps after `install.sh`:

### 1. Create the per-plugin Postgres database

```sql
-- on the dock-side Postgres (any superuser):
CREATE DATABASE polar_wg OWNER polar;
```

Then apply the schema migration shipped in the tarball:

```bash
psql -d polar_wg -f migrations/wg-schema.sql
```

### 2. Mint the dock plugin token

On the dock host:

```bash
TOKEN=$(openssl rand -hex 16)          # capture this; you'll need it for the env file
echo "plaintext token: polar_plugin_${TOKEN}"
HASH=$(printf 'polar_plugin_%s' "$TOKEN" | shasum -a 256 | awk '{print $1}')
psql -d ideamesh <<SQL
INSERT INTO plugin_modules (name, display_name, endpoint, plugin_key_hash)
VALUES ('wg', 'WireGuard', '127.0.0.1:8090', '${HASH}');
SQL
```

The plaintext `polar_plugin_${TOKEN}` only ever lives in the plugin host's env file (mode 0600). Only the SHA-256 hash is stored on dock. If the env file is lost, mint a new token and rotate.

### 3. Edit the env file

```bash
sudoedit /etc/polar/wg-svc.env       # linux
sudoedit /usr/local/etc/polar/wg-svc.env   # freebsd
$EDITOR ~/.config/polar/wg-svc.env   # macOS (or wherever the launchd wrapper sources)
```

Set at minimum:

```
POLAR_DOCK_URL=https://dock.example.com
POLAR_PLUGIN_TOKEN=polar_plugin_<TOKEN-from-step-2>
POLAR_WG_DB_DSN=postgres://polar:<password>@dockhost:5432/polar_wg?sslmode=disable
POLAR_WG_LISTEN=127.0.0.1:8090
```

(Each plugin's `env/<svc>-svc.env.sample` lists all knobs it understands.)

### 4. Wire nginx

The shipped snippet defines one `location` block per route the plugin owns. Include it from your dock vhost (or per-subdomain vhost):

```nginx
# inside the plugin's <svc>.example.com server { ... }
include /etc/nginx/snippets/wg-svc.conf;
```

Reload nginx (`nginx -t && nginx -s reload`).

### 5. Start the service

```bash
sudo systemctl enable --now polar-wg-svc           # linux
sudo service polar_wg_svc start                    # freebsd
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/polar.wg-svc.plist  # macOS
```

Verify:

```bash
curl http://127.0.0.1:8090/healthz                  # plugin alive
curl https://wg.example.com:2443/api/admin/wg-tokens  # nginx → plugin → 401 (unauth, but reachable)
```

In the dock UI under "Plugins", the row should show a fresh heartbeat within 60s.

## Per-OS service paths

Quick reference for where files land after `install.sh`:

| | macOS | Linux | FreeBSD |
|---|---|---|---|
| Binary | `/usr/local/bin/<svc>-svc` | `/usr/local/bin/<svc>-svc` | `/usr/local/bin/<svc>-svc` |
| Service | `~/Library/LaunchAgents/polar.<svc>-svc.plist` | `/etc/systemd/system/polar-<svc>-svc.service` | `/usr/local/etc/rc.d/polar_<svc>_svc` |
| Env | `/usr/local/etc/polar/<svc>-svc.env` | `/etc/polar/<svc>-svc.env` | `/usr/local/etc/polar/<svc>-svc.env` |
| Nginx snippet | `/usr/local/etc/nginx/snippets/<svc>-svc.conf` | `/etc/nginx/snippets/<svc>-svc.conf` | `/usr/local/etc/nginx/snippets/<svc>-svc.conf` |
| Logs | `/var/log/polar/<svc>-svc.log` | `/var/log/polar/<svc>-svc.log` | `/var/log/polar/<svc>-svc.log` |
| Start | `launchctl bootstrap gui/<uid> <plist>` | `systemctl enable --now polar-<svc>-svc` | `service polar_<svc>_svc start` |
| Tail logs | `tail -f /var/log/polar/<svc>-svc.log` | `journalctl -u polar-<svc>-svc -f` | `tail -f /var/log/polar/<svc>-svc.log` |
| Restart | `launchctl kickstart -k gui/<uid>/polar.<svc>-svc` | `systemctl restart polar-<svc>-svc` | `service polar_<svc>_svc restart` |

## Dock-side setup

`polar-dock` is the orchestration core. Until its repo opens, treat this section as the interface contract a plugin assumes.

What dock provides each plugin:
- `POST /internal/v1/auth/verify` — verify a session token.
- `POST /internal/v1/heartbeat` — plugin pings every 60 s so dock can show liveness.
- `GET /internal/v1/users/:id` — display-name / avatar lookup for cross-plugin UI.
- Plus a few more — see [polar-sdk](https://github.com/networkextension/polar-sdk) for the typed client every plugin links against.

What dock needs from operator:
- Postgres database `ideamesh` (role `ideamesh`) with the dock schema.
- Redis on `localhost:6379` (or `REDIS_ADDR=` env override).
- A wildcard cert (per [DNS + cert](#dns--cert)).
- nginx vhost with the per-plugin subdomain server blocks (see `scripts/nginx/polar-plugin-subdomains.conf` in the dock repo for the reference deployment).
- The `POLAR_COOKIE_DOMAIN=.example.com` env so session cookies SSO across subdomains.

## Subdomain map (reference deployment)

The operator-facing pages are split one-per-plugin:

| subdomain | served from | what's behind it |
|---|---|---|
| `zen.example.com` | dock UI bundle | dock pages: dashboard, chat, bot management, system info |
| `wg.example.com` | polar-wg/ui/dist | WireGuard tokens / hubs / devices admin |
| `project.example.com` | polar-projects/ui/dist | Projects, features, tasks, plan generation |
| `latch.example.com` | polar-latch/ui/dist | Latch profiles + traffic rules |
| `hosts.example.com` | polar-hosts/ui/dist | Host inventory, shell + VNC sessions |
| `iosdist.example.com` | polar-iosdist/ui/dist | iOS distribution, sign, plaza |
| `library.example.com` | polar-library/ui/dist | RE knowledge base |
| `video.example.com` | polar-video/ui/dist | Video studio (Seedance pipelines) |
| `expense.example.com` | polar-expense/ui/dist | Expense tracking + receipt OCR |
| `packtunnel.example.com` | (no UI; API only) | proxy / VPN profile API |

Each subdomain's nginx vhost has the same shape: `root /var/polar-ui/<svc>/`, `try_files $uri @dock_dist`, `include polar-plugins.conf`, `/api/*` + `/ws/*` proxy to dock. See `polar-plugin-subdomains.conf` in the dock repo for the canonical block.

## Plugin UI deploys

Each plugin repo carries a `scripts/deploy-ui.sh` that builds its UI (via the `@networkextension/polar-ui-common` npm package) and rsyncs the output to the dock host:

```bash
cd polar-wg
GITHUB_TOKEN=<read:packages-PAT> ./scripts/deploy-ui.sh
```

That puts the built bundle at `local@dockhost:/var/polar-ui/wg/`. nginx serves it from there for `wg.example.com`. Re-running the script is a hot deploy — no service restart needed because nginx serves static files.

## Smoke-test checklist

After everything is wired:

```bash
# dock alive
curl -fsS https://zen.example.com:2443/api/system/info > /dev/null && echo "dock OK"

# every plugin alive (returns 401 from the plugin svc, meaning the route reaches it)
for sub in wg project latch hosts iosdist library video expense packtunnel; do
    printf "%-12s " "$sub.example.com"
    curl -sk -o /dev/null -w "%{http_code}\n" "https://$sub.example.com:2443/api/$sub/"
done

# cookie SSO
# 1. open zen.example.com, log in
# 2. open project.example.com in the same browser → should land on the page, not the login screen
```

## Common ops

**Rotate a plugin token** — re-mint, update the env file, restart the svc, then `DELETE` the old `plugin_modules` row.

**Move a plugin to a new host** — fresh `install.sh` on the new host, copy the env file (or recreate it), point DNS / nginx upstream at the new host, then take down the old one. Both can run side-by-side until you flip nginx.

**Upgrade a plugin** — re-run `build-release.sh` against a new tag, ship the tarball, `install.sh` is idempotent and replaces the binary in place. systemd/launchd auto-restart picks it up; FreeBSD needs `service polar_<svc>_svc restart`.

**Roll back a plugin** — keep the prior tarball, re-run its `install.sh`. The schema migration is forward-only; if a release ships a destructive migration, ship its `-down` sibling first.

## Troubleshooting

- `nginx -t` passes but the subdomain still 404s → likely the snippet's `location` doesn't match. Try `curl -v https://...` and look at the redirect chain.
- Plugin returns 502 → dock isn't reachable from the plugin host. `POLAR_DOCK_URL` is probably wrong, or there's a firewall between them.
- Plugin returns 401 on `/api/<svc>/<endpoint>` but you sent a valid cookie → the cookie isn't being sent. Check `Domain=` on the Set-Cookie; for cross-subdomain SSO it must be `.example.com` (set via `POLAR_COOKIE_DOMAIN`).
- WireGuard / VNC / shell connections drop after 10 minutes → nginx `proxy_read_timeout` default is 60s. The reference vhosts bump WS to 3600s.

## See also

- [README](../README.md) — module map + open-source status
- [polar-sdk](https://github.com/networkextension/polar-sdk) — Go SDK every plugin links
- [polar-ui-common](https://github.com/networkextension/polar-ui-common) — shared frontend
- Per-plugin repo READMEs for plugin-specific env knobs
