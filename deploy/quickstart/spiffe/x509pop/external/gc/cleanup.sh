#!/bin/bash
# Cleanup Ground Control X.509 PoP SPIRE setup
# Safe to run even if containers are already stopped/removed.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Cleaning up Ground Control (X.509 PoP) ==="

# Try to delete SPIRE entry if the server is still running
if docker inspect spire-server --format '{{.State.Running}}' 2>/dev/null | grep -q true; then
    echo "> Deleting Ground Control SPIRE entry..."
    ENTRY_ID=$(docker exec spire-server /opt/spire/bin/spire-server entry show \
        -spiffeID spiffe://harbor-satellite.local/ground-control \
        -socketPath /tmp/spire-server/private/api.sock 2>/dev/null \
        | grep "Entry ID" | awk '{print $4}') || true
    if [ -n "$ENTRY_ID" ]; then
        docker exec spire-server /opt/spire/bin/spire-server entry delete \
            -entryID "$ENTRY_ID" \
            -socketPath /tmp/spire-server/private/api.sock 2>/dev/null || true
    fi
fi

echo "> Stopping and removing containers, volumes, and orphans..."
docker compose down -v --remove-orphans 2>/dev/null || true

echo "> Removing certificates..."
rm -rf ./certs

echo "> Removing docker network..."
docker network rm harbor-satellite 2>/dev/null || true

echo "Cleanup complete"
