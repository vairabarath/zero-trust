# Integration TODO: PostgreSQL + Google OAuth + Email Invites

Source reference: `../group-management-ui-backend/`

---

## 1. PostgreSQL Support

The current controller uses SQLite only. The target project supports PostgreSQL as primary with SQLite as fallback.

### Controller (`services/controller/`)

- [ ] Add `DATABASE_URL` env var support — if set, connect to Postgres; otherwise fall back to SQLite at `DB_PATH`
- [ ] Add `github.com/lib/pq` or `pgx` driver to `go.mod`
- [ ] Abstract DB initialization in `state/` so the same schema runs on both SQLite and Postgres
  - SQLite uses `TEXT` for UUIDs and booleans; Postgres uses `UUID` and `BOOLEAN` — handle dialect differences
  - Audit the `state/` package: `registry.go`, `tunnelers.go`, `tokens.go`, `users.go`, `acls.go`, `remote_networks.go`, `persistence.go`
- [ ] Add `audit_logs` table (see below under Audit Logs)
- [ ] Update `services/controller/.env.example` with `DATABASE_URL` example DSN

### Environment Variables to Add

```
DATABASE_URL=postgres://user:pass@localhost:5432/ztna?sslmode=disable
# leave empty to use SQLite fallback (DB_PATH)
```

---

## 2. Google OAuth Authentication

The current project uses a static `ADMIN_AUTH_TOKEN` for all admin access. The target adds Google OAuth login for the admin UI.

### Controller (`services/controller/admin/`)

- [ ] Add `golang.org/x/oauth2` and `golang.org/x/oauth2/google` to `go.mod`
- [ ] Create `oauth_invite_handlers.go` — Google OAuth flow:
  - `GET /oauth/login` — generate random CSRF state, store in short-lived cookie or signed value, redirect to Google consent screen with state param
  - `GET /oauth/callback` — verify state param matches (CSRF check), exchange code for Google token, extract email, verify email is in `ADMIN_LOGIN_EMAILS`, issue JWT session cookie, redirect to `DASHBOARD_URL`
  - `POST /oauth/logout` — clear session cookie
- [ ] Create `session_helpers.go` — JWT helpers:
  - Sign/verify JWT with a secret (`JWT_SECRET` env var)
  - Session cookie name, expiry, http-only flag
  - Helper to extract authenticated email from request context
- [ ] Add `ADMIN_LOGIN_EMAILS` check — only emails in this comma-separated list are allowed to authenticate
- [ ] Add OAuth middleware to protect admin routes (sits alongside the existing Bearer token auth)
- [ ] Register new routes in `admin/ui_routes.go`

### Environment Variables to Add

```
GOOGLE_CLIENT_ID=<your-oauth-client-id>
GOOGLE_CLIENT_SECRET=<your-oauth-client-secret>
OAUTH_REDIRECT_URL=http://localhost:8081/oauth/callback   # must match Google Console setting
ADMIN_LOGIN_EMAILS=admin@example.com,other@example.com
JWT_SECRET=<random-secret>
DASHBOARD_URL=http://localhost:5173
```

### Frontend (`apps/frontend/`)

- [ ] Add `/login` page — renders Google Sign-In button, redirects to `GET /oauth/login` on controller
- [ ] Add auth guard in `App.tsx` — redirect unauthenticated users to `/login`
- [ ] Handle session expiry — catch 401 responses and redirect to `/login`
- [ ] Add logout button in nav/sidebar calling `POST /oauth/logout`

---

## 3. Email Invites via SMTP

The target project supports inviting users by email through the admin UI.

### Controller (`services/controller/admin/`)

- [ ] Add invite endpoint to `oauth_invite_handlers.go` (or a new `invite_handlers.go`):
  - `POST /api/admin/users/invite` — accepts `{ email: string }`, generates a signed invite token, stores in `invite_tokens` table, sends invite email
  - `GET /invite?token=<token>` — validates token exists and is not expired/used, then starts Google OAuth flow with the invite token embedded as the OAuth `state` parameter (so it survives the redirect roundtrip)
  - In `GET /oauth/callback`: detect if `state` encodes an invite token (vs a regular login state):
    - Verify the invite token from state is still valid in DB
    - Verify the Google-authenticated email **matches** the invited email exactly
    - If match: create user record in `users` table, mark invite token as used, issue JWT session cookie, redirect to `DASHBOARD_URL`
    - If mismatch: return error (wrong Google account used)
- [ ] Implement SMTP sender:
  - Create `mailer/mailer.go` using `net/smtp` from stdlib
  - Support Gmail App Password auth: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`
  - Email template: include `INVITE_BASE_URL/invite?token=<token>` link
- [ ] Store invite tokens in DB (`invite_tokens` table: `token`, `email`, `expires_at`, `used`)
- [ ] Expire tokens after 48 hours; mark used only after user record is successfully created

### Environment Variables to Add

```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=youremail@gmail.com
SMTP_PASS=<gmail-app-password>
SMTP_FROM=ZTNA Admin <youremail@gmail.com>
INVITE_BASE_URL=http://localhost:8081
```

### Frontend (`apps/frontend/`)

- [ ] Add "Invite User" button on the Users page
- [ ] BFF route `POST /api/users/invite` — forwards to controller invite endpoint
- [ ] Show success/error toast after invite is sent

---

## 4. Audit Logs (dependency of above features)

Both OAuth login events and invite actions should be recorded.

### Controller

- [ ] Create `audit_logs` table: `id`, `timestamp`, `actor` (email/token), `action`, `target`, `result`
- [ ] Log events: admin login, admin logout, invite sent, invite redeemed, user created/deleted, connector enrolled/revoked
- [ ] Add `GET /api/admin/audit-logs` endpoint (paginated)

### Frontend

- [ ] Add Audit Logs page (or section in Settings) that fetches from the endpoint above

---

## 5. Supporting Infrastructure

### Air (Live Reload for Controller)

- [ ] Add `.air.toml` to `services/controller/`
- [ ] Add `run-air.sh` script that sources `.env` then runs `air`
- [ ] Add `make dev-controller-air` target in root `Makefile`

---

## Implementation Order

1. **PostgreSQL** — foundation everything else builds on
   - DB abstraction layer in `state/`
   - Migrate schema to support both dialects
2. **Google OAuth + Sessions** — required before invite flow
   - `oauth_invite_handlers.go`, `session_helpers.go`
   - Frontend `/login` page + auth guard
3. **Email Invites** — depends on OAuth (invitee redeems via Google login)
   - SMTP mailer, invite token table, invite endpoint
   - Frontend "Invite User" UI
4. **Audit Logs** — add alongside steps 2 and 3
5. **Air** — dev experience improvement

---

## Files to Create or Modify

| File | Action |
|---|---|
| `services/controller/state/db.go` | Create — DB init abstraction (Postgres vs SQLite) |
| `services/controller/admin/oauth_invite_handlers.go` | Create |
| `services/controller/admin/session_helpers.go` | Create |
| `services/controller/admin/invite_handlers.go` | Create (or fold into oauth file) |
| `services/controller/mailer/mailer.go` | Create |
| `services/controller/admin/ui_routes.go` | Modify — register OAuth + invite routes |
| `services/controller/admin/handlers_users.go` | Modify — add invite endpoint |
| `services/controller/.env` | Modify — add new env vars |
| `apps/frontend/src/pages/Login.tsx` | Create |
| `apps/frontend/src/App.tsx` | Modify — add auth guard + `/login` route |
| `apps/frontend/server/routes/users.ts` | Modify — add `/invite` BFF route |
| `services/controller/.air.toml` | Create |
| `services/controller/run-air.sh` | Create |
| `Makefile` | Modify — add `dev-controller-air` target |
