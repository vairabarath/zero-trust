#!/usr/bin/env bash
set -euo pipefail

# Uninstall Zero-Trust Tunneler

if [[ "${EUID}" -ne 0 ]]; then
  echo "ERROR: uninstall must be run as root." >&2
  exit 1
fi

echo "Stopping and disabling tunneler service..."
systemctl stop tunneler.service 2>/dev/null || true
systemctl disable tunneler.service 2>/dev/null || true

echo "Removing systemd unit..."
rm -f /etc/systemd/system/tunneler.service
systemctl daemon-reload

echo "Removing binary..."
rm -f /usr/bin/tunneler

echo "Removing configuration..."
rm -rf /etc/tunneler

echo "Removing state directory..."
rm -rf /var/lib/tunneler

echo "Removing runtime directory..."
rm -rf /run/tunneler

echo "Tunneler uninstalled."
