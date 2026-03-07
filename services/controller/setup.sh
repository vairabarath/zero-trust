#!/bin/bash
set -e

echo "=== Controller Setup ==="

# 1. Build controller
echo "Building controller..."
go build -o controller .
echo "✓ Built"

# 2. Generate CA if not exists
if [ ! -f ca/ca.crt ]; then
    echo "Generating CA certificates..."
    mkdir -p ca

    # Generate CA private key
    openssl ecparam -genkey -name prime256v1 -out ca/ca.key

    # Convert to PKCS8 format
    openssl pkcs8 -topk8 -nocrypt -in ca/ca.key -out ca/ca.pkcs8.key

    # Generate CA certificate
    openssl req -new -x509 -key ca/ca.key -out ca/ca.crt -days 3650 \
      -subj "/CN=Internal CA/O=MyCorp/C=US"

    echo "✓ CA certificates generated"
else
    echo "✓ CA certificates exist"
fi

# 3. Create .env if not exists
if [ ! -f .env ]; then
    echo "Creating .env file..."
    echo ""

    # Prompt for secrets that can't be hardcoded
    read -rp "Google OAuth Client ID: " GOOGLE_CLIENT_ID
    read -rp "Google OAuth Client Secret: " GOOGLE_CLIENT_SECRET
    read -rp "Admin email (for login access): " ADMIN_EMAIL
    read -rp "SMTP password (Gmail App Password, leave empty to skip): " SMTP_PASS

    JWT_SECRET=$(openssl rand -hex 32)

    cat > .env << ENVEOF
# ── General ───────────────────────────────────────────────────────────────────
TRUST_DOMAIN=mycorp.internal

# ── Controller ────────────────────────────────────────────────────────────────
CONTROLLER_ADDR=localhost:8443
ADMIN_HTTP_ADDR=0.0.0.0:8081

# Tokens / secrets
ADMIN_AUTH_TOKEN=7f8e91a2b3c4d5e6f7a8b9c0d1e2f3a4
INTERNAL_API_TOKEN=e4b2f8d1c3a9e6f7b0d2a4c9e8f1a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3
POLICY_SIGNING_KEY=e4b2f8d1c3a9e6f7b0d2a4c9e8f1a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3
JWT_SECRET=${JWT_SECRET}

# Policy TTL (seconds, default 600)
POLICY_SNAPSHOT_TTL_SECONDS=600

# ── Database ──────────────────────────────────────────────────────────────────
# Leave DATABASE_URL empty to use SQLite via DB_PATH
DATABASE_URL=postgres://ztnaadmin:inkztnapass@localhost:5432/ztna?sslmode=disable
DB_PATH=controller.db

# ── OAuth / Google ────────────────────────────────────────────────────────────
GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
OAUTH_REDIRECT_URL=http://localhost:8080/oauth/google/callback
OAUTH_CALLBACK_ADDR=:8080

# ── URLs ──────────────────────────────────────────────────────────────────────
DASHBOARD_URL=http://localhost:3000
INVITE_BASE_URL=http://localhost:8081

# ── Access control ────────────────────────────────────────────────────────────
ADMIN_LOGIN_EMAILS=${ADMIN_EMAIL}

# ── SMTP ──────────────────────────────────────────────────────────────────────
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=${ADMIN_EMAIL}
SMTP_PASS=${SMTP_PASS}
SMTP_FROM=${ADMIN_EMAIL}
ENVEOF
    echo "✓ .env created (JWT_SECRET auto-generated)"
else
    echo "✓ .env exists"
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "To run the controller:"
echo "  make dev-controller"
echo ""
echo "Or manually:"
echo "  INTERNAL_CA_CERT=\"\$(cat ca/ca.crt)\" \\"
echo "  INTERNAL_CA_KEY=\"\$(cat ca/ca.pkcs8.key)\" \\"
echo "  ./controller"
