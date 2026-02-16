#!/bin/bash
# Run Ground Control with SPIFFE (X.509 PoP)
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GC_DEPLOY_DIR="$SCRIPT_DIR/deploy/quickstart/spiffe/x509pop/external/gc"

# Configurable via env vars
export HARBOR_URL="${HARBOR_URL:-http://host.docker.internal:8080}"
export HARBOR_USERNAME="${HARBOR_USERNAME:-admin}"
export HARBOR_PASSWORD="${HARBOR_PASSWORD:-Harbor12345}"
export ADMIN_PASSWORD="${ADMIN_PASSWORD:-Harbor12345}"
export GC_HOST_PORT="${GC_HOST_PORT:-9080}"
export SPIRE_HOST_PORT="${SPIRE_HOST_PORT:-9081}"
export SKIP_HARBOR_HEALTH_CHECK="${SKIP_HARBOR_HEALTH_CHECK:-false}"

echo "=== Starting Ground Control ==="
echo "  Harbor URL:      $HARBOR_URL"
echo "  GC Port:         $GC_HOST_PORT"
echo "  SPIRE Port:      $SPIRE_HOST_PORT"
echo ""

cd "$GC_DEPLOY_DIR"
./setup.sh

echo ""
echo "Ground Control is running at https://localhost:$GC_HOST_PORT"
echo "Stop with: cd $GC_DEPLOY_DIR && docker compose down"
