# How To Run (Manual Setup)

This guide gets the full ZTNA stack running locally without Docker.

## Prerequisites

- **Go 1.24+** — [Download](https://go.dev/dl/)
- **Rust 1.70+** — [Install](https://rustup.rs/)
- **Node.js 20+** and npm — [Download](https://nodejs.org/)
- **PostgreSQL 16+** — [Download](https://www.postgresql.org/download/) *(optional — SQLite used if `DATABASE_URL` is empty)*

---

## 1. PostgreSQL Setup (optional)

Skip this section if you want to use the SQLite fallback (set `DATABASE_URL=` empty in `.env`).

### Install PostgreSQL

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql && sudo systemctl enable postgresql
```

**macOS (Homebrew):**
```bash
brew install postgresql@16
brew services start postgresql@16
```

### Create Database and User

```bash
sudo -u postgres psql
```

```sql
CREATE DATABASE ztna;
CREATE USER ztnaadmin WITH PASSWORD 'inkztnapass';
GRANT ALL PRIVILEGES ON DATABASE ztna TO ztnaadmin;
\q
```

### Verify Connection

```bash
psql -h localhost -U ztnaadmin -d ztna
# \q to exit
```

> **Shortcut:** `bash scripts/setup-db.sh` does all of the above automatically.

---

## 2. Environment Configuration

All secrets live in `services/controller/.env` (already populated).  
The frontend reads `apps/frontend/.env`.

### Generate a JWT Secret (if `JWT_SECRET` is empty)

```bash
openssl rand -hex 32
# Paste the output into JWT_SECRET in services/controller/.env
```

### Key values already set in `services/controller/.env`

| Variable | Value |
|---|---|
| `DATABASE_URL` | `postgres://ztnaadmin:inkztnapass@localhost:5432/ztna?sslmode=disable` |
| `DB_USER` | `ztnaadmin` / password `inkztnapass` |
| `ADMIN_AUTH_TOKEN` | `7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4` |
| `GOOGLE_CLIENT_ID` | `60482071733-dfnm857cq…` |
| `GOOGLE_CLIENT_SECRET` | `GOCSPX-ywuKJnso…` |
| `ADMIN_LOGIN_EMAILS` | `gztnaadmin1906@gmail.com` |
| `SMTP_USER` | `gztnaadmin1906@gmail.com` |
| `SMTP_PASS` | *(Gmail App Password)* |
| `OAUTH_REDIRECT_URL` | `http://localhost:8081/oauth/callback` |
| `DASHBOARD_URL` | `http://localhost:5173` |

---

## 3. Controller (Go)

```bash
# Option A — standard run
make dev-controller

# Option B — live reload with Air (requires: go install github.com/air-verse/air@latest)
make dev-controller-air

# Option C — build then run
make build-controller
./dist/controller
```

**Verify:**
```bash
# Should return JSON
curl http://localhost:8081/api/admin/connectors \
  -H "Authorization: Bearer 7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4"
```

---

## 4. Frontend (React + Express BFF)

```bash
cd apps/frontend
npm install
```

The `.env` is already created at `apps/frontend/.env`.

```bash
make dev-frontend
# Or:
cd apps/frontend && npm run dev
```

This starts:
- **Vite dev server** → http://localhost:5173
- **Express BFF** → http://localhost:3001

---

## 5. Access the Application

| URL | Purpose |
|---|---|
| http://localhost:5173/login | Login page |
| http://localhost:5173/dashboard/groups | Dashboard |
| http://localhost:8081 | Admin API |
| http://localhost:8081/oauth/login | OAuth entry point |
| http://localhost:8081/oauth/callback | Google OAuth callback |

### Login Flow

1. Open http://localhost:5173/login
2. Click **Sign in with Google**
3. Authenticate with `gztnaadmin1906@gmail.com` (or any email in `ADMIN_LOGIN_EMAILS`)
4. Redirected to dashboard automatically

---

## 6. Connector (Rust — optional)

```bash
make dev-connector
# Or:
cd services/connector && cargo run
```

Requires enrollment token from controller:
```bash
curl -X POST http://localhost:8081/api/admin/tokens \
  -H "Authorization: Bearer 7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4"
```

---

## 7. Tunneler (Rust — optional)

```bash
make dev-tunneler
# Or:
cd services/tunneler && cargo run
```

---

## 8. Common Operations

### Send a User Invite

```bash
curl -X POST http://localhost:8081/api/admin/users/invite \
  -H "Authorization: Bearer 7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com"}'
```

When SMTP is not configured, the invite URL is printed in the controller log.

### View Admin Audit Logs

```bash
curl http://localhost:8081/api/admin/audit-logs \
  -H "Authorization: Bearer 7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4"
```

---

## 9. Recommended Terminal Layout

```
Terminal 1 — Controller:   make dev-controller-air
Terminal 2 — Frontend:     make dev-frontend
Terminal 3 — (optional):   psql -h localhost -U ztnaadmin -d ztna
```

---

## 10. Troubleshooting

### PostgreSQL: connection refused / auth failed

```bash
sudo systemctl status postgresql
psql -h localhost -U ztnaadmin -d ztna
# If pg_hba.conf rejects local connections:
sudo nano /etc/postgresql/16/main/pg_hba.conf
# Ensure: host all all 127.0.0.1/32 md5
sudo systemctl restart postgresql
```

### Port already in use

```bash
sudo lsof -i :8081   # find PID
kill -9 <PID>
```

### OAuth callback mismatch

In [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials → OAuth 2.0 Client:

Add authorized redirect URI: `http://localhost:8081/oauth/callback`

### Frontend can't reach backend

```bash
# Verify controller is up
curl http://localhost:8081/api/admin/connectors \
  -H "Authorization: Bearer 7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4"

# Ensure BACKEND_URL in apps/frontend/.env = http://localhost:8081
# Ensure ADMIN_AUTH_TOKEN matches in both .env files
```

### SMTP not sending

- Gmail requires 2FA + an [App Password](https://myaccount.google.com/apppasswords)
- `SMTP_PASS` in `services/controller/.env` must be the app password, not the account password

---

## 11. Build for Production

```bash
make build-all          # builds controller, connector, tunneler, frontend

# Run production frontend
cd apps/frontend
npm run build
npm start               # Express serves built dist/
```

---

## Quick Start (TL;DR)

```bash
# 1. Postgres (if using)
bash scripts/setup-db.sh

# 2. Start controller
make dev-controller

# 3. Start frontend (new terminal)
cd apps/frontend && npm install && npm run dev

# 4. Open browser
xdg-open http://localhost:5173/login   # Linux
open http://localhost:5173/login       # macOS
```
