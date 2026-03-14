# Repository Guidelines

## Project Structure & Module Organization
- `services/controller/`: Go control plane (gRPC + admin HTTP API).
- `services/connector/`: Rust gateway service.
- `services/agent/`: Rust resource-side enforcement service (nftables firewall enforcer).
- `services/ztna-client/`: Rust user-side client application (device auth, local callback service, SOCKS5 client-side access path).
- `apps/frontend/`: Vite + React + TypeScript UI plus Express server.
- `shared/proto/`: Protobuf definitions shared across services.
- `shared/configs/`: Config and `.env` examples.
- `docs/`: Architecture and development notes.
- `scripts/`: Setup and deployment helpers.

## Build, Test, and Development Commands
Run from repo root unless noted.
- `make build-all`: Build all components.
- `make dev-controller|dev-connector|dev-agent|dev-frontend`: Run a single component in dev mode.
- `make test-all`: Run all component test suites.
- `make test-controller|test-connector|test-agent|test-frontend`: Run component tests.
- `make clean`: Clear build artifacts.

Component-local equivalents:
- Go: `cd services/controller && go build ./... && go test ./...`
- Rust: `cd services/connector && cargo build --release && cargo test`
- Rust client: `cd services/ztna-client && cargo build && cargo test`
- Frontend: `cd apps/frontend && npm run dev|build|lint`

## Coding Style & Naming Conventions
- Use language-standard formatting: `gofmt` for Go, `rustfmt` for Rust.
- Frontend linting via `npm run lint` (ESLint).
- Tests follow language conventions: Go `_test.go`, Rust `#[test]` modules, frontend tests live near `src/` where applicable.
- Branch names: `feature/<component>-short-description` (see `docs/development.md`).

## Testing Guidelines
- Prefer `make test-<component>` before pushing.
- When updating `shared/proto/`, regenerate and verify all services.
- Add unit, integration, or end-to-end tests based on impact (see `docs/development.md`).

## Commit & Pull Request Guidelines
- Commit messages generally follow Conventional Commits, often with scopes:
  - Examples: `feat(controller): ...`, `fix: ...`, `chore: ...`, `test(agent): ...`.
- Prefer small, focused commits that group one coherent change area at a time.
- When updating operational behavior, setup flows, or architecture assumptions, update the relevant docs or config examples in the same change.
- Avoid mixing maintenance-only edits with feature work unless the maintenance is required for that feature to work or be understood.
- PRs target the `develop` branch and should include:
  - Clear description, linked issue (if any), and tests run.
  - Updates to docs or configs when behavior or env vars change.
  - Small, focused changes that are easy to review.

## Security & Configuration Tips
- Never commit secrets. Use `shared/configs/.env.example` for new variables.
- Controller uses PostgreSQL when `DATABASE_URL` is set; otherwise SQLite.
- Connector policy signing keys are derived from the controller mTLS session; normal setup should not require a static `POLICY_SIGNING_KEY`.
- Connector and agent runtime certificate state may be configured as runtime-only; be explicit in docs/scripts when changing persistence expectations.

## Current Architecture Notes
- `services/agent` is resource-side only. Do not add user-side SOCKS, proxy, or general client traffic interception there.
- `services/ztna-client` is the user/client-side application. New end-user access work such as local SOCKS5 listeners, client auth UX, token refresh, and split-tunnel behavior should go there.
- Client-side ACL checks are allowed as a fast pre-check for split tunneling and UX, but connector-side enforcement remains authoritative.
- For diagnostics and user install flows, prefer extending existing pages and routes before adding parallel pages.

## Agent-Specific Notes
- If you are using an automated assistant, read `CLAUDE.md` for architecture details, command summaries, and environment variables.
