# Polar

A cloud built on Apple Silicon — and on everything that behaves like it. Genuine hardware, Hackintosh rigs, scavenged silicon from wherever the market shakes loose: if it runs, it joins the fleet. The hardware is substrate. The control plane is the project. Built AI-native, end to end.

This repository is the **project front door**. Polar is a control plane (`polar-dock`) plus a fleet of independent plugins — each its own repo, its own Postgres, its own Go binary, its own subdomain. Many are open; the core and the commercial verticals are still private. The **Status** column says which is which; everything open lives under [`@networkextension`](https://github.com/networkextension).

---

## Modules

Status: **open** = public repo you can clone today · **private** = source not yet public (binaries run in production).

### Foundation

| Module | Status | What it does |
|---|---|---|
| [`polar-sdk`](https://github.com/networkextension/polar-sdk) | open | Go SDK every plugin links — HMAC plugin-token auth, dock `/internal/v1/*` client, heartbeat + platform-nav + assets helpers. Stdlib-only. |
| [`polar-ui-common`](https://github.com/networkextension/polar-ui-common) | open | Shared frontend helpers (sidebar / platform-nav / theme / i18n / auth session) + global stylesheet + branding, published to GitHub Packages as `@networkextension/polar-ui-common`. |
| [`polar-agent`](https://github.com/networkextension/polar-agent) | open | Local executor — long-lived WS to dock, runs bot tool-calls (shell, MCP, iOS sign, WireGuard, VNC, …) in a configured workdir. Reports host-info on hello. Ships `polar-agent` + `polar-agent-test`; cross-platform. |
| [`polar-dock-ui`](https://github.com/networkextension/polar-dock-ui) | private | The dock's web app — HTML pages + TypeScript bundles served in front of `polar-dock`. esbuild → `dist/`, CI on every push. |
| [`polar-dock`](https://github.com/networkextension/polar-dock) | private · **core** | Identity + workspaces/RBAC + LLM proxy + chat/rooms + agent hub + assets router + the host vhost that fans every plugin out under one origin. Open release comes once the surface settles. **耐心等待 / patient.** |

### Data plane

| Module | Status | What it does |
|---|---|---|
| [`polar-assets`](https://github.com/networkextension/polar-assets) | open | Edge-cache asset store — signed PUT/GET blobs, warm-pull across provider nodes, LRU eviction. Every media-handling plugin (video, expense, library, music…) writes its bytes here instead of local disk. |
| [`polar-release`](https://github.com/networkextension/polar-release) | private | Signed, content-addressed module release registry on the assets data plane — ed25519-signed manifests powering OTA module self-update + the module marketplace. |

### Plugins — horizontal (each = standalone Go service + its own UI)

| Module | Status | What it does |
|---|---|---|
| [`polar-wg`](https://github.com/networkextension/polar-wg) | open | WireGuard mesh control plane — multi-hub, role-aware allocator, MagicDNS-style name resolution, Headscale compat path, hub-status pipeline. |
| [`polar-hosts`](https://github.com/networkextension/polar-hosts) | open | Host inventory + 3-tier host-info + per-host skill registry (Shell, VNC, MCP) bridged over WS through the dock agent hub. |
| [`polar-projects`](https://github.com/networkextension/polar-projects) | open | Lightweight project / feature / task workspace with AI requirement decomposition + plan generation. |
| [`polar-library`](https://github.com/networkextension/polar-library) | open | Reverse-engineering knowledge base — devices / firmwares / functions with blob upload, lookup-by-symbol/address/signature. |
| [`polar-video`](https://github.com/networkextension/polar-video) | open | Video studio — Seedance + FFmpeg pipeline for projects / shots / assets. |
| [`polar-expense`](https://github.com/networkextension/polar-expense) | open | Household / team expense tracking with vision-LLM receipt extraction. |
| [`polar-latch`](https://github.com/networkextension/polar-latch) | open | Latch — proxy + traffic rules + service-node + agent-runtime profiles, with a connector toolchain. |
| [`polar-music`](https://github.com/networkextension/polar-music) | open | Workspace-scoped private music library (audio bytes in `polar-assets`), Apple-Music-style UI. |
| [`polar-dns`](https://github.com/networkextension/polar-dns) | open | DNS records management plugin. |
| [`polar-packtunnel`](https://github.com/networkextension/polar-packtunnel) | private | Proxy / VPN profile management — opening later. |
| [`polar-iosdist`](https://github.com/networkextension/polar-iosdist) | private | iOS distribution + signing (App Store Connect sync, zsign, plaza). Pending license review. |
| [`polar-photo`](https://github.com/networkextension/polar-photo) | private | Private photo library on `polar-assets` — search · faces · collections · trips · Apple-Photos import, with an Apple-Photos-style UI. |
| [`polar-survey`](https://github.com/networkextension/polar-survey) | private | Surveys / forms (WindTunnel) — form builder + response collection. |

### Verticals (industry solutions built on the platform base)

The same identity + LLM-proxy + assets + dataflow base, specialized per industry. Mostly private while they're commercialized.

| Module | Status | What it does |
|---|---|---|
| [`polar-buildings`](https://github.com/networkextension/polar-buildings) | private | 楼宇 — facility / building management: sites · floors · zones · devices · work orders · inspections, AI alarm→work-order dispatch, BMS telemetry time-series. |
| [`polar-lawyer`](https://github.com/networkextension/polar-lawyer) | private | 律所 — legal / contract / compliance RAG: statute library + case library, document upload/OCR/statute-import, AI drafting. |
| [`polar-screen`](https://github.com/networkextension/polar-screen) | private | DOOH ad network + "screen mining" — device-first provisioning, owner-sovereign content timeline + hard veto, signed proof-of-play earnings ledger, AI order desk, CPA attribution. |
| [`polar-film`](https://github.com/networkextension/polar-film) | private | 影视知识库 — film knowledge base: on-device `filmscan` (Whisper + Vision + FluidAudio) turns video into speaker-attributed subtitles + keyframes, RAG over the catalog. |
| [`polar-stock`](https://github.com/networkextension/polar-stock) | private | Stock / market data plugin. |

### Platform libraries

| Module | Status | What it does |
|---|---|---|
| [`polar-dataflow`](https://github.com/networkextension/polar-dataflow) | private | Reusable 5-stage ingest pipeline + per-vertical pgvector RAG — the retrieval substrate the verticals (lawyer, …) build on. |
| [`polar-zone`](https://github.com/networkextension/polar-zone) | private | Zone federation — stand up a full zen replica from signed, versioned module binaries (bootstrap binary + a `zone` release bundle). |

### SDKs & clients

| Module | Status | What it does |
|---|---|---|
| [`polar-sdk`](https://github.com/networkextension/polar-sdk) | open | Go plugin SDK (see Foundation). |
| [`polar-sdk-swift`](https://github.com/networkextension/polar-sdk-swift) | private | Cross-platform Swift SDK (API/UI split, transport-injected, Apple + Linux/NIO) + a `polar` CLI. |
| [`polar-sdk-py`](https://github.com/networkextension/polar-sdk-py) | private | Python SDK — generated from the dock API catalog. |
| [`ShangDynasty`](https://github.com/networkextension/ShangDynasty) | private | iOS / macOS client app. |
| [`polar-wg-app`](https://github.com/networkextension/polar-wg-app) | private | Polar's cross-platform WireGuard client — C protocol core (ex-FreeBSD), macOS/iOS NetworkExtension, Android, + a CLI reference client. |
| [`Athens`](https://github.com/networkextension/Athens) | private | Android client app. |

---

## How modules talk to each other

```
                      ┌──────────────────────────────────────────────┐
   browser / app ───▶ │  nginx · *.4950.store wildcard TLS            │
                      └───────────────┬──────────────────────────────┘
                                      │
                      ┌───────────────▼──────────────────────────────┐
                      │  polar-dock  (control plane / core)          │
                      │  identity · workspaces/RBAC · LLM proxy ·    │
                      │  chat+rooms · agent hub · assets router      │
                      └───────────────┬──────────────────────────────┘
                                      │  /internal/v1/*  (HMAC-signed plugin tokens)
        ┌───────────────┬─────────────┼─────────────┬───────────────┐
        ▼               ▼             ▼             ▼               ▼
  ┌───────────┐  ┌───────────┐  ┌──────────┐  ┌──────────┐   ┌──────────────┐
  │ polar-wg  │  │ polar-hosts│  │  …video  │  │ verticals│   │  data plane  │
  │ wg.4950…  │  │ host.4950… │  │ expense… │  │ lawyer…  │   │ assets·release│
  └───────────┘  └───────────┘  └──────────┘  └──────────┘   └──────────────┘
        ▲                                                            ▲
        └──────────  polar-agent  (per host: shell/VNC/MCP/wg) ──────┘
```

Every plugin gets a subdomain — `wg.4950.store`, `host.4950.store`, `expense.4950.store`, `lawyer.4950.store`, … — sharing a wildcard `*.4950.store` cert. Plugin UIs ship from their own repos and mount the shared platform nav via `@networkextension/polar-ui-common`. APIs route through the dock for identity + LLM proxy; media bytes go to `polar-assets`; everything else is a per-plugin Postgres + a per-plugin Go binary. New hosts join the fleet by running `polar-agent`.

## Install the SDK

```
go get github.com/networkextension/polar-sdk@latest
```

Build a plugin against it — every open module above is a working example; `polar-wg` is the canonical "happy path" reference.

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
import { mountPlatformNav } from "@networkextension/polar-ui-common/lib/sidebar";
```

## License

Per-module — check each repo's LICENSE file. Open plugins generally ship under permissive terms; private modules will publish their license at open-release time.

## Status & releases

Each module versions itself in its own repo. Watch the ones you care about; this front door isn't a meta-changelog. When `polar-dock` opens, this README gets a 🟢 on that row and a `getting-started` doc with the full stack composition.

请耐心等待 dock 开源。

---

## Activity

<!-- BEGIN:status -->
<!-- auto-generated by scripts/update-repo-status.sh — do not edit by hand -->

_Last refreshed: 2026-08-15 22:35 UTC_

| Repo | Last commit | Open PRs |
|---|---|---|
| [`polar-sdk`](https://github.com/networkextension/polar-sdk) | 2026-06-22 · [`11d27c2`](https://github.com/networkextension/polar-sdk/commit/11d27c2) fix(assets): append ?ct=<mime> to AssetDownloadURLWS signed URLs (#25) | 0 |
| [`polar-ui-common`](https://github.com/networkextension/polar-ui-common) | 2026-06-20 · [`ab1ddd7`](https://github.com/networkextension/polar-ui-common/commit/ab1ddd7) Merge pull request #1 from networkextension/feat/neon-topo-toolkit | 0 |
| [`polar-agent`](https://github.com/networkextension/polar-agent) | 2026-06-24 · [`562c0d5`](https://github.com/networkextension/polar-agent/commit/562c0d5) feat(hostinfo): tag per-NIC kind + MAC + CIDR + bridge members for topology (#25) | 0 |
| [`polar-dock-ui`](https://github.com/networkextension/polar-dock-ui) | _private — see repo_ | — |
| [`polar-dock`](https://github.com/networkextension/polar-dock) | _private — see repo_ | — |
| [`polar-assets`](https://github.com/networkextension/polar-assets) | 2026-06-16 · [`bf6bc7c`](https://github.com/networkextension/polar-assets/commit/bf6bc7c) Merge pull request #8 from networkextension/feat/blob-content-type | 0 |
| [`polar-release`](https://github.com/networkextension/polar-release) | _private — see repo_ | — |
| [`polar-wg`](https://github.com/networkextension/polar-wg) | 2026-07-27 · [`d59a720`](https://github.com/networkextension/polar-wg/commit/d59a720) fix(ui): 走查修复 — 宽表格横向可滚 / 行内控件紧凑 / CJK nowrap / 发布link按钮等高 (#43) | 0 |
| [`polar-hosts`](https://github.com/networkextension/polar-hosts) | 2026-08-15 · [`4c32464`](https://github.com/networkextension/polar-hosts/commit/4c32464) Merge pull request #36 from networkextension/feat/ws-agent-phase4b | 0 |
| [`polar-projects`](https://github.com/networkextension/polar-projects) | 2026-07-27 · [`f59801b`](https://github.com/networkextension/polar-projects/commit/f59801b) fix(ui): 项目详情页窄视口横向裁切 + 行内控件高度错位 (#7) | 0 |
| [`polar-library`](https://github.com/networkextension/polar-library) | 2026-06-15 · [`6be1a51`](https://github.com/networkextension/polar-library/commit/6be1a51) refactor(firmware): cutover to assets-only (drop local-disk fallback + backfill) (#8) | 0 |
| [`polar-video`](https://github.com/networkextension/polar-video) | 2026-06-15 · [`495824c`](https://github.com/networkextension/polar-video/commit/495824c) Merge pull request #11 from networkextension/feat/video-assets-cutover | 0 |
| [`polar-expense`](https://github.com/networkextension/polar-expense) | 2026-06-15 · [`65d6f63`](https://github.com/networkextension/polar-expense/commit/65d6f63) refactor(receipts): cutover to assets-only (drop local-disk fallback + backfill) (#8) | 0 |
| [`polar-latch`](https://github.com/networkextension/polar-latch) | 2026-06-02 · [`0e5ee53`](https://github.com/networkextension/polar-latch/commit/0e5ee53) fix(auth): scope to caller active workspace (#6) | 0 |
| [`polar-music`](https://github.com/networkextension/polar-music) | 2026-06-22 · [`7262cb5`](https://github.com/networkextension/polar-music/commit/7262cb5) feat(music): top-right account chip — in-page login + 乐库 switcher (#14) | 0 |
| [`polar-dns`](https://github.com/networkextension/polar-dns) | 2026-06-21 · [`9c50c83`](https://github.com/networkextension/polar-dns/commit/9c50c83) Merge pull request #3 from networkextension/feat/ui-split-horizon | 0 |
| [`polar-packtunnel`](https://github.com/networkextension/polar-packtunnel) | _private — see repo_ | — |
| [`polar-iosdist`](https://github.com/networkextension/polar-iosdist) | _private — see repo_ | — |
| [`polar-photo`](https://github.com/networkextension/polar-photo) | _private — see repo_ | — |
| [`polar-survey`](https://github.com/networkextension/polar-survey) | _private — see repo_ | — |
| [`polar-buildings`](https://github.com/networkextension/polar-buildings) | _private — see repo_ | — |
| [`polar-lawyer`](https://github.com/networkextension/polar-lawyer) | _private — see repo_ | — |
| [`polar-screen`](https://github.com/networkextension/polar-screen) | _private — see repo_ | — |
| [`polar-film`](https://github.com/networkextension/polar-film) | 2026-06-25 · [`e4ee0f2`](https://github.com/networkextension/polar-film/commit/e4ee0f2) film → identity: feed voiceprints (声纹) to the modeling layer (#34) | 0 |
| [`polar-stock`](https://github.com/networkextension/polar-stock) | _private — see repo_ | — |
| [`polar-dataflow`](https://github.com/networkextension/polar-dataflow) | _private — see repo_ | — |
| [`polar-zone`](https://github.com/networkextension/polar-zone) | _private — see repo_ | — |
| [`polar-sdk-swift`](https://github.com/networkextension/polar-sdk-swift) | _private — see repo_ | — |
| [`polar-sdk-py`](https://github.com/networkextension/polar-sdk-py) | _private — see repo_ | — |
| [`ShangDynasty`](https://github.com/networkextension/ShangDynasty) | _private — see repo_ | — |
| [`polar-wg-app`](https://github.com/networkextension/polar-wg-app) | 2026-08-15 · [`d80e5d7`](https://github.com/networkextension/polar-wg-app/commit/d80e5d7) Merge pull request #37 from networkextension/perf/wg-core-dataplane | 0 |
| [`Athens`](https://github.com/networkextension/Athens) | _private — see repo_ | — |

<!-- END:status -->
