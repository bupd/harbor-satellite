#!/bin/bash
# Find Raspberry Pi on the local network
# Ping sweeps the subnet, then shows all live devices (excluding the gateway)

GATEWAY=$(ip -4 route show default | awk '{print $3}' | head -1)
SUBNET_PREFIX="${GATEWAY%.*}"
MY_IP=$(ip -4 addr show wlan0 2>/dev/null | grep -oP 'inet \K[0-9.]+' | head -1)

echo "Scanning ${SUBNET_PREFIX}.0/24 ..."

for i in $(seq 1 254); do
    ping -c 1 -W 1 "${SUBNET_PREFIX}.${i}" > /dev/null 2>&1 &
done
wait

echo ""
echo "Live devices (excluding gateway and self):"
arp -an | grep wlan0 | grep -v incomplete | while read -r line; do
    ip=$(echo "$line" | grep -oP '\(\K[0-9.]+')
    mac=$(echo "$line" | grep -oP '([0-9a-f]{2}:){5}[0-9a-f]{2}')
    [ "$ip" = "$GATEWAY" ] && continue
    [ "$ip" = "$MY_IP" ] && continue
    echo "  ${ip}  (${mac})"
done
