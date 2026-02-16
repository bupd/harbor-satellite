#!/bin/bash
# Setup SPIRE agent + satellite on Raspberry Pi (bare metal, no Docker)
# Run this on the Pi. Requires: certs copied from GC, SPIRE binary.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPIRE_VERSION="1.12.3"
SPIRE_DIR="/opt/spire"
SPIRE_CONF_DIR="$SPIRE_DIR/conf/agent"
SPIRE_DATA_DIR="$SPIRE_DIR/data/agent"
SPIRE_SOCKET_DIR="/run/spire/sockets"
CERTS_SRC="${SCRIPT_DIR}/certs"

GC_IP="${GC_IP:-10.229.209.55}"
SPIRE_SERVER_PORT="${SPIRE_SERVER_PORT:-9081}"

echo "=== Satellite Pi Setup (SPIRE Agent + X.509 PoP) ==="
echo "  SPIRE Server: ${GC_IP}:${SPIRE_SERVER_PORT}"
echo ""

# Step 1: Check certs
echo "[1/4] Checking certificates..."
for f in ca.crt agent-satellite.crt agent-satellite.key; do
    if [ ! -f "$CERTS_SRC/$f" ]; then
        echo "ERROR: $CERTS_SRC/$f not found"
        echo "Copy certs from GC host: scp <gc-host>:.../gc/certs/{ca.crt,agent-satellite.crt,agent-satellite.key} $CERTS_SRC/"
        exit 1
    fi
done
echo "Certificates found"

# Step 2: Install SPIRE agent binary
echo "[2/4] Installing SPIRE agent..."
if [ -f "$SPIRE_DIR/bin/spire-agent" ]; then
    INSTALLED_VER=$("$SPIRE_DIR/bin/spire-agent" --version 2>&1 | grep -oP '\d+\.\d+\.\d+' || echo "unknown")
    echo "SPIRE agent already installed (${INSTALLED_VER})"
else
    ARCH=$(uname -m)
    case "$ARCH" in
        aarch64|arm64) SPIRE_ARCH="linux-arm64-musl" ;;
        x86_64)        SPIRE_ARCH="linux-amd64-musl" ;;
        armv7l)        SPIRE_ARCH="linux-arm-musl" ;;
        *)             echo "ERROR: unsupported architecture $ARCH"; exit 1 ;;
    esac

    TARBALL="spire-${SPIRE_VERSION}-${SPIRE_ARCH}.tar.gz"
    URL="https://github.com/spiffe/spire/releases/download/v${SPIRE_VERSION}/${TARBALL}"

    echo "Downloading SPIRE ${SPIRE_VERSION} (${SPIRE_ARCH})..."
    curl -fSLO "$URL"

    echo "Installing to ${SPIRE_DIR}..."
    sudo mkdir -p "$SPIRE_DIR/bin"
    tar xzf "$TARBALL"
    sudo cp "spire-${SPIRE_VERSION}/bin/spire-agent" "$SPIRE_DIR/bin/"
    rm -rf "$TARBALL" "spire-${SPIRE_VERSION}"
    echo "SPIRE agent installed"
fi

# Step 3: Setup directories and config
echo "[3/4] Setting up SPIRE agent config..."
sudo mkdir -p "$SPIRE_CONF_DIR" "$SPIRE_DATA_DIR" "$SPIRE_SOCKET_DIR"

# Copy certs
sudo cp "$CERTS_SRC/ca.crt" "$SPIRE_CONF_DIR/bootstrap.crt"
sudo cp "$CERTS_SRC/agent-satellite.crt" "$SPIRE_CONF_DIR/agent.crt"
sudo cp "$CERTS_SRC/agent-satellite.key" "$SPIRE_CONF_DIR/agent.key"
sudo chmod 644 "$SPIRE_CONF_DIR"/*.crt
sudo chmod 644 "$SPIRE_CONF_DIR"/agent.key

# Write agent config
sudo tee "$SPIRE_CONF_DIR/agent.conf" > /dev/null << EOF
agent {
    data_dir = "${SPIRE_DATA_DIR}"
    log_level = "INFO"
    server_address = "${GC_IP}"
    server_port = "${SPIRE_SERVER_PORT}"
    socket_path = "${SPIRE_SOCKET_DIR}/agent.sock"
    trust_bundle_path = "${SPIRE_CONF_DIR}/bootstrap.crt"
    trust_domain = "harbor-satellite.local"
}

plugins {
    NodeAttestor "x509pop" {
        plugin_data {
            private_key_path = "${SPIRE_CONF_DIR}/agent.key"
            certificate_path = "${SPIRE_CONF_DIR}/agent.crt"
        }
    }

    KeyManager "disk" {
        plugin_data {
            directory = "${SPIRE_DATA_DIR}"
        }
    }

    WorkloadAttestor "unix" {
        plugin_data {}
    }
}

health_checks {
    listener_enabled = true
    bind_address = "0.0.0.0"
    bind_port = "9999"
    live_path = "/live"
    ready_path = "/ready"
}
EOF

echo "Config written to $SPIRE_CONF_DIR/agent.conf"

# Step 4: Start SPIRE agent
echo "[4/4] Starting SPIRE agent..."

# Stop existing agent if running
if pgrep -f "spire-agent.*-config" > /dev/null 2>&1; then
    echo "Stopping existing SPIRE agent..."
    sudo pkill -f "spire-agent.*-config" || true
    sleep 2
fi

sudo "$SPIRE_DIR/bin/spire-agent" run \
    -config "$SPIRE_CONF_DIR/agent.conf" &
AGENT_PID=$!

echo "SPIRE agent started (PID: $AGENT_PID)"
echo "Waiting for agent to attest..."

for i in $(seq 1 30); do
    if "$SPIRE_DIR/bin/spire-agent" healthcheck -socketPath "$SPIRE_SOCKET_DIR/agent.sock" > /dev/null 2>&1; then
        echo "SPIRE agent is healthy"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: SPIRE agent failed to start/attest"
        echo "Check logs: sudo journalctl -u spire-agent or look at stderr above"
        exit 1
    fi
    echo "Waiting for SPIRE agent... ($i/30)"
    sleep 2
done

echo ""
echo "=== Setup Complete ==="
echo ""
echo "SPIRE agent running (PID: $AGENT_PID)"
echo "Socket: $SPIRE_SOCKET_DIR/agent.sock"
echo ""
echo "Next: ./satellite.sh"
