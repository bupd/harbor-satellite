#!/bin/bash
# Run Satellite on bare-metal (e.g., Raspberry Pi)
# Requires SPIRE agent running at /run/spire/sockets/agent.sock
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Required
GC_URL="${GC_URL:?Set GC_URL to Ground Control address (e.g., https://10.229.209.55:9080)}"
HARBOR_REGISTRY_URL="${HARBOR_REGISTRY_URL:?Set HARBOR_REGISTRY_URL to reachable Harbor address (e.g., http://10.229.209.55:8080)}"

# Optional (with defaults)
SPIFFE_ENDPOINT_SOCKET="${SPIFFE_ENDPOINT_SOCKET:-unix:///run/spire/sockets/agent.sock}"
SPIFFE_EXPECTED_SERVER_ID="${SPIFFE_EXPECTED_SERVER_ID:-spiffe://harbor-satellite.local/ground-control}"
SATELLITE_BINARY="${SATELLITE_BINARY:-$SCRIPT_DIR/satellite-arm64}"
CONFIG_DIR="${CONFIG_DIR:-}"
LOG_JSON="${LOG_JSON:-true}"

# Build if binary doesn't exist
if [ ! -f "$SATELLITE_BINARY" ]; then
    echo "Binary not found at $SATELLITE_BINARY"
    echo "Building satellite..."
    GOARCH="${GOARCH:-arm64}" go build -o "$SATELLITE_BINARY" "$SCRIPT_DIR/cmd/main.go"
fi

echo "=== Starting Satellite ==="
echo "  GC URL:              $GC_URL"
echo "  Harbor Registry URL: $HARBOR_REGISTRY_URL"
echo "  SPIFFE Socket:       $SPIFFE_ENDPOINT_SOCKET"
echo "  Binary:              $SATELLITE_BINARY"
echo ""

exec "$SATELLITE_BINARY" \
    --ground-control-url "$GC_URL" \
    --harbor-registry-url "$HARBOR_REGISTRY_URL" \
    --spiffe-enabled \
    --spiffe-endpoint-socket "$SPIFFE_ENDPOINT_SOCKET" \
    --spiffe-expected-server-id "$SPIFFE_EXPECTED_SERVER_ID" \
    --use-unsecure \
    --json-logging="$LOG_JSON" \
    ${CONFIG_DIR:+--config-dir "$CONFIG_DIR"}
