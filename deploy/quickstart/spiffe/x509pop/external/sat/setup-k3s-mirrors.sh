#!/bin/bash
# Configure K3s to use satellite's local registry as mirror
set -e

SATELLITE_REGISTRY="${SATELLITE_REGISTRY:-http://localhost:8585}"

echo "=== K3s Private Registry Mirror Setup ==="
echo "Satellite registry: ${SATELLITE_REGISTRY}"

sudo mkdir -p /etc/rancher/k3s

sudo tee /etc/rancher/k3s/registries.yaml > /dev/null << EOF
mirrors:
  docker.io:
    endpoint:
      - "${SATELLITE_REGISTRY}"
  registry.goharbor.io:
    endpoint:
      - "${SATELLITE_REGISTRY}"
EOF

echo "Wrote /etc/rancher/k3s/registries.yaml"

if command -v k3s &> /dev/null && systemctl is-active --quiet k3s; then
    sudo systemctl restart k3s
    echo "K3s restarted"
fi

echo ""
echo "=== Setup Complete ==="
echo "Mirrored registries:"
echo "  docker.io            -> ${SATELLITE_REGISTRY}"
echo "  registry.goharbor.io -> ${SATELLITE_REGISTRY}"
