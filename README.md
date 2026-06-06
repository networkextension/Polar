# Polar

A cloud built on Apple Silicon — and on everything that behaves like it. Genuine hardware, Hackintosh rigs, scavenged silicon from wherever the market shakes loose: if it runs, it joins the fleet. The hardware is substrate. The control plane is the project. Built AI-native, end to end.

This repository is the **project front door**. The control-plane host (`polar-dock`) and a few plugins are still being prepared for open release — see the **Status** column below. Everything currently open lives in its own repo under [`@networkextension`](https://github.com/networkextension).

---

## Modules

### Foundation

| Module | Status | What it does |
|---|---|---|
| [`polar-sdk`](https://github.com/networkextension/polar-sdk) | open | Go SDK every plugin links — HMAC plugin-token auth, dock `/internal/v1/*` client, heartbeat helpers. Stdlib-only. |
| [`polar-ui-common`](https://github.com/networkextension/polar-ui-common) | open | Shared frontend helpers (sidebar / theme / i18n / auth session) + global stylesheet + branding assets, published to GitHub Packages as `@networkextension/polar-ui-common`. |
| [`polar-dock-ui`](https://github.com/networkextension/polar-dock-ui) | _pending_ | The dock's web app — HTML pages + TypeScript bundles served in front of `polar-dock`. Built with esbuild → `dist/`, CI on every push. Opens alongside `polar-dock`. |
| [`polar-agent`](https://github.com/networkextension/polar-agent) | open | Local executor — long-lived WS to dock, runs bot tool-calls (shell, MCP, iOS sign, …) in a configured workdir. Ships `polar-agent` + `polar-agent-test` binaries; cross-platform. |

### Plugins (each = standalone service + UI)

| Module | Status | What it does |
|---|---|---|
| [`polar-wg`](https://github.com/networkextension/polar-wg) | open | WireGuard mesh control plane — multi-hub, role-aware allocator, MagicDNS-style name resolution, Headscale compat path. |
| [`polar-projects`](https://github.com/networkextension/polar-projects) | open | Lightweight project / feature / task workspace with AI requirement decomposition + plan generation. |
| [`polar-hosts`](https://github.com/networkextension/polar-hosts) | open | Host inventory + per-host skill registry (Shell, VNC, MCP) with WS bridge through the dock agent hub. |
| [`polar-library`](https://github.com/networkextension/polar-library) | open | Reverse-engineering knowledge base — devices / firmwares / functions with blob upload, lookup-by-symbol/address/signature. |
| [`polar-video`](https://github.com/networkextension/polar-video) | open | Video studio — Seedance + FFmpeg pipeline for projects / shots / assets. |
| [`polar-expense`](https://github.com/networkextension/polar-expense) | open | Household / team expense tracking with vision-LLM receipt extraction. |
| [`polar-latch`](https://github.com/networkextension/polar-latch) | open | Latch — proxy + traffic rules + service-node + agent-runtime profiles, with a connector toolchain. |
| [`polar-packtunnel`](https://github.com/networkextension/polar-packtunnel) | _pending_ | Proxy / VPN profile management plugin — opening soon. |
| [`polar-iosdist`](https://github.com/networkextension/polar-iosdist) | _pending_ | iOS distribution + signing (App Store Connect sync, zsign, plaza). Pending license review. |

### Control plane

| Module | Status | What it does |
|---|---|---|
| [`polar-dock`](https://github.com/networkextension/polar-dock) | _pending_ | Identity + LLM proxy + chat + agent_hub + the host vhost that fans out to every plugin. The "core" — open release coming once the surface has settled. **耐心等待 / patient.** |

### Clients

| Module | Status | What it does |
|---|---|---|
| [`ShangDynasty`](https://github.com/networkextension/ShangDynasty) | _developing_ | iOS / macOS client app. |
| [`polar-wg-app`](https://github.com/networkextension/polar-wg-app) | _developing_ | Polar's cross-platform WireGuard client. |
| [`Athens`](https://github.com/networkextension/Athens) | _developing_ | Android client app. |

---

## How modules talk to each other

```
            ┌─────────────────────────────────────────────────────┐
            │                  zen.4950.store                     │
            │             nginx + polar-dock (control plane)      │
            └───────────────────────┬─────────────────────────────┘
                                    │ /internal/v1/*  (HMAC-signed)
            ┌───────────────────────┴─────────────────────────────┐
            │                                                     │
   ┌────────▼─────────┐  ┌──────────────────┐  ┌─────────────────▼┐
   │   polar-wg       │  │  polar-projects  │  │  polar-expense   │
   │   :8090          │  │  :8096           │  │  :8097           │
   └──────────────────┘  └──────────────────┘  └──────────────────┘
              (… 6 more plugin services, one per module …)
```

Every plugin gets a subdomain — `wg.4950.store`, `project.4950.store`, `expense.4950.store`, … — sharing a wildcard `*.4950.store` cert. Plugin UIs ship from their own repos via `@networkextension/polar-ui-common`. APIs route through the dock for identity and LLM proxy; everything else is per-plugin Postgres + a per-plugin Go binary.

## Install the SDK

```
go get github.com/networkextension/polar-sdk@latest
```

Build a plugin against it — every open module above is a working example, but `polar-wg` is the canonical "happy path" reference.

## Install the shared UI lib

```
# .npmrc
@networkextension:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

```
npm install @networkextension/polar-ui-common
```

`GITHUB_TOKEN` needs `read:packages` scope. Subpath imports avoid pulling unused modules:

```ts
import { byId } from "@networkextension/polar-ui-common/lib/dom";
import { logout, fetchCurrentUser } from "@networkextension/polar-ui-common/api/session";
```

## License

Per-module — check each repo's LICENSE file. Plugins generally ship under permissive terms; pending modules will publish their license at open-release time.

## Status & releases

Each module versions itself in its own repo. Watch the modules you care about; this front door doesn't try to be a meta-changelog. When `polar-dock` lands, this README gets a "🟢 open" badge on that row and a `getting-started` doc with the full stack composition.

请耐心等待 dock 开源。

---

## Activity

<!-- BEGIN:status -->
<!-- auto-generated by scripts/update-repo-status.sh — do not edit by hand -->

_Last refreshed: 2026-06-06 15:38 UTC_

| Repo | Last commit | Open PRs |
|---|---|---|
| [`polar-sdk`](https://github.com/networkextension/polar-sdk) | 2026-06-03 · [`a565526`](https://github.com/networkextension/polar-sdk/commit/a565526) Merge pull request #17 from networkextension/feat/asset-download-url | 0 |
| [`polar-ui-common`](https://github.com/networkextension/polar-ui-common) | 2026-06-02 · [`9696566`](https://github.com/networkextension/polar-ui-common/commit/9696566) feat(sidebar): mountPlatformNav() — shared platform nav (v0.2.0) | 0 |
| [`polar-dock-ui`](https://github.com/networkextension/polar-dock-ui) | _private — see repo_ | — |
| [`polar-agent`](https://github.com/networkextension/polar-agent) | 2026-06-06 · [`14cda96`](https://github.com/networkextension/polar-agent/commit/14cda96) Merge pull request #18 from networkextension/feat/hostinfo-static-facts | 0 |
| [`polar-wg`](https://github.com/networkextension/polar-wg) | 2026-06-03 · [`65b9888`](https://github.com/networkextension/polar-wg/commit/65b9888) Merge pull request #15 from networkextension/feat/wg-bundles-backfill-on-boot | 0 |
| [`polar-projects`](https://github.com/networkextension/polar-projects) | 2026-06-02 · [`2739bb6`](https://github.com/networkextension/polar-projects/commit/2739bb6) feat(ui): platform nav via mountPlatformNav() (#5) | 0 |
| [`polar-hosts`](https://github.com/networkextension/polar-hosts) | 2026-06-06 · [`a19f527`](https://github.com/networkextension/polar-hosts/commit/a19f527) Merge pull request #24 from networkextension/feat/hosts-ui-hostinfo | 0 |
| [`polar-library`](https://github.com/networkextension/polar-library) | 2026-06-03 · [`810fe28`](https://github.com/networkextension/polar-library/commit/810fe28) feat(firmware): single-write firmware blobs to polar-assets + backfill (#7) | 0 |
| [`polar-video`](https://github.com/networkextension/polar-video) | 2026-06-03 · [`3d195ac`](https://github.com/networkextension/polar-video/commit/3d195ac) Merge pull request #10 from networkextension/feat/video-blobs-to-assets | 0 |
| [`polar-expense`](https://github.com/networkextension/polar-expense) | 2026-06-03 · [`5017a0d`](https://github.com/networkextension/polar-expense/commit/5017a0d) feat(receipts): single-write expense images to polar-assets + backfill (#7) | 0 |
| [`polar-latch`](https://github.com/networkextension/polar-latch) | 2026-06-02 · [`0e5ee53`](https://github.com/networkextension/polar-latch/commit/0e5ee53) fix(auth): scope to caller active workspace (#6) | 0 |
| [`polar-packtunnel`](https://github.com/networkextension/polar-packtunnel) | _private — see repo_ | — |
| [`polar-iosdist`](https://github.com/networkextension/polar-iosdist) | _private — see repo_ | — |
| [`polar-dock`](https://github.com/networkextension/polar-dock) | _private — see repo_ | — |
| [`ShangDynasty`](https://github.com/networkextension/ShangDynasty) | _private — see repo_ | — |
| [`polar-wg-app`](https://github.com/networkextension/polar-wg-app) | _private — see repo_ | — |
| [`Athens`](https://github.com/networkextension/Athens) | _private — see repo_ | — |

<!-- END:status -->

