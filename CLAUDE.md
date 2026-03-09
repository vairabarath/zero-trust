# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Twingate-style zero-trust identity and access control management system** with:
- A **Go backend** (gRPC, mTLS, SPIFFE IDs) acting as the control plane
- A **Next.js frontend** (React 19, TypeScript) for administrative UI
- **Dual-database backend**: PostgreSQL (via `DATABASE_URL`) or SQLite (fallback)
- SQLite for frontend local state

## Commands

### Frontend (`cd frontend`)

```bash
npm run dev       # Start dev server (default: localhost:3000)
npm run build     # Production build
npm run start     # Run production build
npm run lint      # ESLint
```

### Backend (`cd backend`)

Each component is a separate Go module under `backend/`:

```bash
# Build
cd backend/controller && go build ./...
cd backend/connector && go build ./...
cd backend/tunneler && go build ./...

# Run controller (requires env vars)
sudo TRUST_DOMAIN="mycorp.internal" \
  INTERNAL_CA_CERT="$(cat ca/ca.crt)" \
  INTERNAL_CA_KEY="$(cat ca/ca.pkcs8.key)" \
  ADMIN_AUTH_TOKEN="7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4" \
  INTERNAL_API_TOKEN="e4b2f8d1c3a9e6f7b0d2a4c9e8f1a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3" \
  CONTROLLER_ADDR="<host>:8443" \
  ADMIN_HTTP_ADDR="0.0.0.0:8081" \
  ./controller
```

The root `backend/` directory has no Go packages — `go build ./...` must be run from within `controller/`, `connector/`, or `tunneler/`.

## Architecture

### Backend (Go)

Three services in `backend/`:

- **Controller** (`backend/controller/`): Internal CA + enrollment gRPC server on `:8443`, admin HTTP API on `:8081`. Manages PostgreSQL/SQLite DB, token store, ACLs, policy distribution, Google OAuth login, email invites, and admin audit logging.
- **Connector** (`backend/connector/`): Long-running service that connects outbound to the controller. Accepts inbound tunneler connections on `:9443`.
- **Tunneler** (`backend/tunneler/`): Client-side service that connects to a connector with mTLS.

All services use SPIFFE IDs under trust domain `spiffe://mycorp.internal`. Identity format:
- Connector: `spiffe://mycorp.internal/connector/<id>`
- Tunneler: `spiffe://mycorp.internal/tunneler/<id>`

Admin HTTP API routes live in `backend/controller/admin/` — `handlers.go` for core, `ui_handlers.go` for UI-specific endpoints, `ui_routes.go` for routing, `oauth_invite_handlers.go` for OAuth/invite/audit endpoints, and `session_helpers.go` for JWT session management. gRPC implementations are in `backend/controller/api/`. SMTP mailer is in `backend/controller/mailer/`.

### Frontend (Next.js App Router)

**Data flow:** Next.js API routes (`app/api/`) act as middleware — they either proxy to the Go backend via `lib/proxy.ts` (pointing to `NEXT_PUBLIC_API_BASE_URL`, default `:8081`) or read/write directly to a local SQLite database via `lib/db.ts`.

Key lib files:
- `lib/types.ts` — all shared TypeScript types (User, Group, Resource, Connector, etc.)
- `lib/db.ts` — SQLite schema, migrations, and seeding (via `better-sqlite3`)
- `lib/proxy.ts` — proxies requests to the Go backend with Bearer token auth
- `lib/mock-api.ts` — frontend API client that calls `/api/*` routes
- `lib/sign-in-policy.ts`, `lib/resource-policies.ts`, `lib/device-profiles.ts` — policy management (client-side, persisted to localStorage)

**Pages** are under `app/dashboard/` — groups, users, resources, connectors, tunnelers, remote-networks, and policy sub-routes.

**Components** under `components/dashboard/` mirror the page structure. Shared UI primitives are shadcn/ui components in `components/ui/`.

### Environment Variables

| Variable | Service | Description |
|---|---|---|
| `NEXT_PUBLIC_API_BASE_URL` | Frontend | Go backend URL (default: `http://localhost:8081`) |
| `ADMIN_AUTH_TOKEN` | Frontend + Controller | Bearer token for admin API |
| `INTERNAL_CA_CERT` | Controller/Connector/Tunneler | PEM CA certificate |
| `INTERNAL_CA_KEY` | Controller | PEM PKCS#8 CA private key |
| `CONTROLLER_ADDR` | Connector/Tunneler | `host:port` of controller gRPC |
| `ADMIN_HTTP_ADDR` | Controller | HTTP listen address (default `:8081`) |
| `DATABASE_URL` | Controller | PostgreSQL connection string (if set, uses Postgres; otherwise SQLite) |
| `DB_PATH` | Controller | SQLite database path (used when `DATABASE_URL` is empty) |
| `TRUST_DOMAIN` | All | SPIFFE trust domain (default: `mycorp.internal`) |
| `GOOGLE_CLIENT_ID` | Controller | Google OAuth client ID (empty = OAuth disabled) |
| `GOOGLE_CLIENT_SECRET` | Controller | Google OAuth client secret |
| `OAUTH_REDIRECT_URL` | Controller | OAuth callback URL (e.g. `http://localhost:8081/oauth/callback`) |
| `ADMIN_LOGIN_EMAILS` | Controller | Comma-separated allowed admin emails for OAuth login |
| `JWT_SECRET` | Controller | Secret for signing session JWTs |
| `DASHBOARD_URL` | Controller | Frontend URL for OAuth redirect (default: `http://localhost:5173`) |
| `INVITE_BASE_URL` | Controller | Base URL for invite links (default: `http://localhost:8081`) |
| `SMTP_HOST` | Controller | SMTP host for email invites (empty = disabled, URL logged) |
| `SMTP_PORT` | Controller | SMTP port (e.g. `587`) |
| `SMTP_USER` | Controller | SMTP username |
| `SMTP_PASS` | Controller | SMTP password |
| `SMTP_FROM` | Controller | SMTP from address |
| `OAUTH_CALLBACK_ADDR` | Controller | Optional separate listener for OAuth callbacks |

### Key Design Notes

- **TypeScript errors ignored at build time** — `next.config.mjs` sets `typescript.ignoreBuildErrors: true`
- **Schema migrations** in `lib/db.ts` handle live upgrades (e.g., adding columns to `access_rules`)
- **Policy state** (sign-in policy, resource policies, device profiles) is stored in localStorage on the client, not in the database
- The Go module name for the controller is `controller` (set in `services/controller/go.mod`)
- **Dual-database**: `state.Open()` in `state/db.go` picks PostgreSQL or SQLite; `state.Rebind()` converts `?` placeholders to `$N` for Postgres
- **OAuth + invite flow**: Google OAuth login → JWT session cookie; invite tokens stored in `invite_tokens` table; admin audit logs in `admin_audit_logs`
