# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Twingate-style zero-trust network access system with:
- **Controller** (Go): Certificate Authority + control plane (gRPC + HTTP admin API)
- **Connector** (Rust): Gateway service that accepts tunneler connections
- **Tunneler** (Rust): Client-side service connecting to a connector
- **Frontend** (React + Vite + Express): Management UI with a BFF server

## Repository Layout

```
services/
  controller/   # Go - CA, enrollment gRPC (:8443), admin HTTP (:8081)
  connector/    # Rust - gateway, listens for tunneler connections (:9443)
  tunneler/     # Rust - client, connects to connector via mTLS
apps/
  frontend/     # React 19 (Vite) + Express BFF server
shared/
  proto/        # controller.proto — source of truth for gRPC definitions
  configs/      # .env.example
```

## Commands

### Root (Makefile)

```bash
make build-all          # Build all components
make build-controller   # Go binary -> dist/controller
make build-connector    # Rust binary -> dist/grpcconnector2
make build-tunneler     # Rust binary -> dist/grpctunneler
make build-frontend     # Vite build

make dev-controller     # Loads services/controller/.env and runs go run .
make dev-connector      # cargo run
make dev-tunneler       # cargo run
make dev-frontend       # npm run dev (Vite + Express concurrently)

make test-controller    # go test ./...
make test-connector     # cargo test
make test-tunneler      # cargo test
make test-frontend      # npm test

make clean              # Remove dist artifacts + Rust targets
```

### Controller (`cd services/controller`)

```bash
go build ./...
go test ./...
go test ./... -run TestName   # Run a single test

# Run with env vars (uses services/controller/.env via make dev-controller)
sudo TRUST_DOMAIN="mycorp.internal" \
  INTERNAL_CA_CERT="$(cat ca/ca.crt)" \
  INTERNAL_CA_KEY="$(cat ca/ca.pkcs8.key)" \
  ADMIN_AUTH_TOKEN="<token>" \
  INTERNAL_API_TOKEN="<token>" \
  CONTROLLER_ADDR="<host>:8443" \
  ADMIN_HTTP_ADDR="0.0.0.0:8081" \
  ./controller
```

### Connector / Tunneler (`cd services/connector` or `services/tunneler`)

```bash
cargo build --release
cargo test
cargo run                     # Uses env vars or config for enrollment
```

### Frontend (`cd apps/frontend`)

```bash
npm run dev     # Starts Vite (default :5173) + Express BFF (:3001) concurrently
npm run build   # Vite production build -> dist/
npm start       # Runs Express server only (serves built dist/)
npm run lint    # ESLint
```

## Architecture

### Backend (Go) — `services/controller/`

- **`main.go`** — wires up the gRPC server, admin HTTP server, and SQLite state
- **`ca/`** — internal certificate authority; issues certs during enrollment
- **`api/`** — gRPC service implementations: enrollment (`enroll.go`), control plane stream (`control_plane.go`), policy snapshot (`policy_snapshot.go`), auth interceptor
- **`admin/`** — HTTP admin API: `handlers.go` (core), `handlers_users.go`, `handlers_remote_networks.go`, `ui_handlers.go` (UI-specific), `ui_routes.go`
- **`state/`** — SQLite-backed state: connectors (`registry.go`), tunnelers, tokens, users, ACLs, remote networks, persistence layer
- **`gen/controllerpb/`** — generated protobuf Go code (do not edit manually)

Go module name: `controller` (in `services/controller/go.mod`)

### Connector / Tunneler (Rust) — `services/connector/`, `services/tunneler/`

Both use Tokio + Tonic (gRPC) + Rustls. Key modules:
- `enroll.rs` — enrollment handshake with controller
- `renewal.rs` — certificate auto-renewal
- `tls/` — SPIFFE ID validation, mTLS cert store, client/server TLS configs
- `config.rs` — env var config loading
- `persistence.rs` — local state persistence
- `control_plane.rs` (connector) — maintains long-lived gRPC stream to controller
- `policy/` (connector) — policy cache and enforcement
- `server.rs` (connector) — accepts inbound tunneler connections
- `run.rs` (tunneler) — main run loop

The proto definitions are compiled via `tonic-build` in `build.rs`; generated code lives at `src/proto/controller.v1.rs`.

### Frontend — `apps/frontend/`

**Architecture:** Vite + React 19 (SPA with React Router) + Express BFF server running alongside in dev.

- **`src/`** — React app: `App.tsx` (routes), `src/pages/` (page components by domain), `src/main.tsx` (entry)
- **`server/`** — Express BFF (`index.ts` on `:3001`): routes under `server/routes/` either proxy to the Go controller or read/write a local SQLite database
- **`lib/`** — shared utilities:
  - `lib/types.ts` — all TypeScript types (User, Group, Resource, Connector, etc.)
  - `lib/db.ts` — SQLite schema + migrations (via `better-sqlite3`)
  - `lib/proxy.ts` — proxies requests to Go backend with Bearer token auth
  - `lib/mock-api.ts` — frontend API client calling `/api/*`
  - `lib/sign-in-policy.ts`, `lib/resource-policies.ts`, `lib/device-profiles.ts` — policy state (persisted to localStorage)
- **`components/dashboard/`** — domain-specific dashboard components (mirrors page structure)
- **`components/ui/`** — shadcn/ui primitives

### Protobuf

`shared/proto/controller.proto` is the canonical proto definition. Generated Go code is in `services/controller/gen/controllerpb/`. Rust services compile proto at build time via `tonic-build`.

### SPIFFE Identity

All services use trust domain `spiffe://mycorp.internal`:
- Connector: `spiffe://mycorp.internal/connector/<id>`
- Tunneler: `spiffe://mycorp.internal/tunneler/<id>`

### Environment Variables

| Variable | Service | Description |
|---|---|---|
| `TRUST_DOMAIN` | All | SPIFFE trust domain (default: `mycorp.internal`) |
| `INTERNAL_CA_CERT` | Controller/Connector/Tunneler | PEM CA certificate |
| `INTERNAL_CA_KEY` | Controller | PEM PKCS#8 CA private key |
| `ADMIN_AUTH_TOKEN` | Controller + Frontend | Bearer token for admin HTTP API |
| `INTERNAL_API_TOKEN` | Controller | Token for internal gRPC auth |
| `CONTROLLER_ADDR` | Connector/Tunneler | `host:port` of controller gRPC (default `:8443`) |
| `ADMIN_HTTP_ADDR` | Controller | HTTP listen address (default `:8081`) |
| `DB_PATH` | Controller | SQLite database path |
| `PORT` | Frontend | Express BFF port (default `3001`) |

Controller dev env vars live in `services/controller/.env` (loaded by `make dev-controller`).

### Key Design Notes

- **Policy state** (sign-in policy, resource policies, device profiles) is stored in localStorage on the client, not the database
- **Schema migrations** are handled inline in `lib/db.ts`
- The frontend BFF (`server/`) routes either proxy to the Go controller via `lib/proxy.ts` (pointing to `NEXT_PUBLIC_API_BASE_URL`, default `:8081`) or handle data locally in SQLite
- Connector binary output name: `grpcconnector2`; Tunneler: `grpctunneler`
