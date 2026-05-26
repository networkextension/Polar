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
