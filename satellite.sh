#!/bin/bash
# Run satellite with SPIFFE/SPIRE on Raspberry Pi (no Docker, no token)
# Prerequisites:
#   1. SPIRE agent running on the Pi (see setup-sat-pi.sh)
#   2. Satellite binary cross-compiled for arm64
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SATELLITE_BIN="${SCRIPT_DIR}/satellite"

# Defaults pointing to laptop running GC
GC_URL="${GC_URL:-https://10.229.209.55:9080}"
HARBOR_REGISTRY_URL="${HARBOR_REGISTRY_URL:-http://10.229.209.55:8080}"
SPIFFE_SOCKET="${SPIFFE_SOCKET:-unix:///run/spire/sockets/agent.sock}"

if [ ! -f "$SATELLITE_BIN" ]; then
    echo "ERROR: satellite binary not found at $SATELLITE_BIN"
    echo "Build with: GOOS=linux GOARCH=arm64 go build -o satellite ./cmd/"
    exit 1
fi

# Check SPIRE agent is running
if ! /opt/spire/bin/spire-agent healthcheck -socketPath /run/spire/sockets/agent.sock > /dev/null 2>&1; then
    echo "ERROR: SPIRE agent is not running. Run setup-sat-pi.sh first."
    exit 1
fi

echo "Starting satellite (SPIFFE mode)"
echo "  GC URL:     $GC_URL"
echo "  Harbor URL: $HARBOR_REGISTRY_URL"
echo "  SPIFFE:     $SPIFFE_SOCKET"
echo ""

exec "$SATELLITE_BIN" \
    --ground-control-url "$GC_URL" \
    --harbor-registry-url "$HARBOR_REGISTRY_URL" \
    --use-unsecure \
    --spiffe-enabled \
    --spiffe-endpoint-socket "$SPIFFE_SOCKET" \
    --spiffe-expected-server-id "spiffe://harbor-satellite.local/ground-control"
