# Polar-

Polar- is an AI-assisted product prototyping project built to explore a practical question:

How far can a small but real full-stack application be pushed when design, implementation, iteration, and documentation are continuously developed with AI coding tools in the loop?

This repository is not intended to be a large-scale production system. It is a working experience project: a compact, end-to-end web application used to validate ideas, refine UX decisions, test authentication flows, and evolve engineering structure through fast iteration.

Today, the project includes account authentication, Markdown content management, Passkey support, login IP and geolocation tracking, a sidebar-based dashboard, and lightweight theming.

## Project Character

- AI-assisted by design, not as a one-off experiment but as an active development workflow
- Built as a usable prototype rather than a throwaway mock
- Focused on complete user journeys over excessive abstraction
- Optimized for iteration speed, clarity, and demonstrable product behavior

## Current Feature Set

### Authentication

- Email/password registration and login
- Cookie + server-side session management
- Logout and authenticated user lookup
- Redis-backed session persistence

### Markdown Workspace

- Create, read, update, and delete Markdown entries
- Store Markdown metadata in PostgreSQL
- Store Markdown source content on disk
- Render Markdown previews in the dashboard and editor

### Passkey / WebAuthn

- Bind Passkeys for authenticated users
- Authenticate users via Passkey
- Support a more modern, password-light login experience

### Dashboard Experience

- Sidebar-oriented dashboard layout
- Centralized access to Markdown records
- Inline content preview
- Quick actions for editing, deleting, and creating content
- Theme switcher directly available from the sidebar

### Login IP and Geolocation

- Record login IP addresses on successful sign-in
- Resolve country, region, and city via GeoLite2 City
- Track login method:
  - Password login
  - Register-and-login flow
  - Passkey login
- Display recent login activity in the sidebar

### Visual Themes

- Default visual mode
- Monochrome mode
- Theme preference persisted locally in the browser

## Why This Project Exists

Polar- exists as a hands-on exploration of AI-native software development.

Instead of treating AI as a code generator used occasionally, this project treats AI as an active collaborator across:

- feature implementation
- UI refinement
- documentation
- structural cleanup
- product iteration

The value of the repository is not only in the code itself, but in the workflow it represents: rapid iteration with enough engineering discipline to keep the project understandable, extensible, and demo-ready.

## Technology Stack

### Backend

- Go
- Gin
- PostgreSQL
- Redis
- WebAuthn
- GeoLite2

### Frontend

- HTML
- CSS
- Vanilla JavaScript
- `marked` for Markdown rendering
- `express` for local UI hosting and API proxying

## Architecture Overview

Polar- intentionally keeps the stack compact.

### 1. Complete flows first

The project prioritizes complete product flows over early abstraction. A feature is considered valuable when it works end to end:

- register
- sign in
- enter dashboard
- create Markdown
- edit or delete content
- bind Passkey
- review login history

This keeps development grounded in user-facing behavior instead of speculative system design.

### 2. Metadata in the database, content on disk

Markdown content is split into two layers:

- PostgreSQL stores ownership, title, file path, and timestamps
- the filesystem stores the raw Markdown body

This design keeps the implementation lightweight while preserving enough structure for pagination, authorization, and future migration.

### 3. Sessions instead of JWT

Authentication is intentionally session-based:

- more natural for browser-first flows
- simpler logout semantics
- straightforward server-side control
- well suited to a prototype with evolving auth requirements

Redis is used as the session store to keep authentication checks fast and lightweight on the API path. Passkey support is layered on top of this foundation rather than replacing it.

### 4. Lightweight frontend by choice

The frontend deliberately avoids a heavy SPA framework at this stage.

This makes iteration faster and keeps the UI:

- easy to inspect
- easy to modify
- easy to refactor with AI assistance

For a project centered on experimentation and velocity, this tradeoff is intentional.

### 5. GeoLite as an enhancement, not a hard dependency

Login geolocation is useful, but it should not be able to break sign-in.

If the GeoLite database is unavailable:

- authentication still works
- login events can still be recorded with raw IP data
- geolocation simply becomes unavailable

This keeps the core product path resilient.

## Repository Structure

```text
.
├── cmd/dock                  # Application entrypoint
├── internal/app/dock         # Core backend logic
├── ui/public                 # Static frontend pages
├── ui/server.js              # Local UI server and API proxy
├── doc                       # Project documentation and work logs
├── scripts                   # Setup and testing helpers
└── data                      # Runtime data, Markdown files, GeoLite database
```

## Local Development

### Start the backend

```bash
cd /path/to/Polar-
env GOCACHE=/tmp/polar-go-cache go run ./cmd/dock
```

Default backend address:

- `http://localhost:8080`

### Start the UI

```bash
cd /path/to/Polar-/ui
node server.js
```

Default UI address:

- `http://localhost:3000`

## Environment Variables

- `POSTGRES_DSN`
  - PostgreSQL connection string
- `REDIS_ADDR`
  - Redis server address
- `REDIS_PASSWORD`
  - Redis password, if required
- `REDIS_DB`
  - Redis logical database index
- `REDIS_PREFIX`
  - key prefix used for session records
- `MARKDOWN_DIR`
  - directory used to store Markdown source files
- `GEOLITE_DB_PATH`
  - path to the GeoLite2 City database
- `PASSKEY_ORIGIN`
  - origin used for Passkey / WebAuthn validation
- `PASSKEY_RP_ID`
  - relying party ID for Passkey
- `PASSKEY_RP_NAME`
  - relying party display name
- `PUBLIC_BASE_URL`
  - public base URL used to build email verification links, for example `https://polar.example.com`
- `APPLE_PUSH_TOPIC`
  - default APNs topic, usually the iOS app bundle ID
- `APPLE_PUSH_TOPIC_DEV`
  - sandbox APNs topic, overrides `APPLE_PUSH_TOPIC`
- `APPLE_PUSH_TOPIC_PROD`
  - production APNs topic, overrides `APPLE_PUSH_TOPIC`
- `APPLE_PUSH_KEY_ID`
  - default APNs key ID for `.p8` auth
- `APPLE_PUSH_KEY_ID_DEV`
  - sandbox APNs key ID, overrides `APPLE_PUSH_KEY_ID`
- `APPLE_PUSH_KEY_ID_PROD`
  - production APNs key ID, overrides `APPLE_PUSH_KEY_ID`
- `APPLE_PUSH_TEAM_ID`
  - default Apple Developer Team ID for `.p8` auth
- `APPLE_PUSH_TEAM_ID_DEV`
  - sandbox APNs team ID, overrides `APPLE_PUSH_TEAM_ID`
- `APPLE_PUSH_TEAM_ID_PROD`
  - production APNs team ID, overrides `APPLE_PUSH_TEAM_ID`
- `SMTP_HOST`
  - SMTP server hostname, for example `smtp.mail.me.com`
- `SMTP_PORT`
  - SMTP server port, iCloud app-specific password typically uses `587`
- `SMTP_USERNAME`
  - SMTP login username, for iCloud usually your full iCloud email address
- `SMTP_PASSWORD`
  - SMTP login password, supports iCloud App-Specific Password
- `SMTP_FROM_EMAIL`
  - sender email address shown in outgoing verification emails
- `SMTP_FROM_NAME`
  - optional sender display name

### iCloud SMTP Example

```bash
export PUBLIC_BASE_URL="https://polar.example.com"
export SMTP_HOST="smtp.mail.me.com"
export SMTP_PORT="587"
export SMTP_USERNAME="yourname@icloud.com"
export SMTP_PASSWORD="xxxx-xxxx-xxxx-xxxx"
export SMTP_FROM_EMAIL="yourname@icloud.com"
export SMTP_FROM_NAME="Polar-"
```

After configuration, authenticated users can call:

```text
POST /api/email-verification/send
```

The verification email link points to:

```text
GET /api/email-verification/verify?token=...
```

## GeoLite Setup

To enable login geolocation, provide a `GeoLite2-City.mmdb` database file.

Default lookup path:

```text
data/GeoLite2-City.mmdb
```

Or set it explicitly:

```bash
GEOLITE_DB_PATH=/your/path/GeoLite2-City.mmdb
```

If the file is missing, the application will continue to function normally, but location resolution will be unavailable.

## Existing Documentation

- [Authentication Notes](./doc/auth.md)
- [API Reference](./doc/api.md)
- [Passkey Notes](./doc/passkey.md)
- [Work Log: 2026-03-19](./doc/worklog-2026-03-19.md)

## Roadmap Directions

Potential next steps include:

- a dedicated login history page
- suspicious login or location change alerts
- search and tagging for Markdown entries
- a richer editor experience
- broader automated testing coverage
- stronger UI consistency and design system patterns

## Closing Note

Polar- is best understood as a living prototype.

It is a place where product thinking, engineering pragmatism, interface experimentation, and AI-assisted development meet in one repository. The codebase is intentionally small enough to move quickly, but complete enough to surface real architectural and UX decisions.
