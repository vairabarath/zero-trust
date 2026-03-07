#!/usr/bin/env bash
# run-air.sh — source .env then start air for live-reload controller development
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load env vars from .env (ignore comments and blank lines)
if [ -f .env ]; then
  set -a
  source <(grep -v '^\s*#' .env | grep -v '^\s*$')
  set +a
fi

# Inject CA certs from files if env vars are empty
export INTERNAL_CA_CERT="${INTERNAL_CA_CERT:-$(cat ca/ca.crt 2>/dev/null || true)}"
export INTERNAL_CA_KEY="${INTERNAL_CA_KEY:-$(cat ca/ca.pkcs8.key 2>/dev/null || true)}"

exec air
