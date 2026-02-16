#!/bin/bash
# Bandwidth limiter for Raspberry Pi (uses tc/tbf, no kernel modules needed for egress)
# Usage:
#   ./bw-limit.sh on [down_mbit] [up_mbit]   - Enable limits (default: 2mbit down, 1mbit up)
#   ./bw-limit.sh off                          - Remove limits
#   ./bw-limit.sh status                       - Show current tc rules

IFACE="${IFACE:-wlan0}"
DOWN="${2:-2}"
UP="${3:-1}"

case "${1:-}" in
    on)
        echo "Limiting $IFACE: download=${DOWN}mbit upload=${UP}mbit"

        # Clear existing rules
        sudo tc qdisc del dev "$IFACE" root 2>/dev/null
        sudo tc qdisc del dev "$IFACE" ingress 2>/dev/null
        sudo tc qdisc del dev ifb0 root 2>/dev/null

        # Egress (upload) limit
        sudo tc qdisc add dev "$IFACE" root tbf rate "${UP}mbit" burst 32kbit latency 400ms
        echo "  Upload limited to ${UP}mbit"

        # Ingress (download) limit via ifb
        if sudo modprobe ifb 2>/dev/null; then
            sudo ip link add ifb0 type ifb 2>/dev/null || true
            sudo ip link set ifb0 up
            sudo tc qdisc add dev "$IFACE" handle ffff: ingress
            sudo tc filter add dev "$IFACE" parent ffff: protocol ip u32 match u32 0 0 action mirred egress redirect dev ifb0
            sudo tc qdisc add dev ifb0 root tbf rate "${DOWN}mbit" burst 32kbit latency 400ms
            echo "  Download limited to ${DOWN}mbit"
        else
            echo "  WARNING: ifb module not available, download shaping skipped (egress only)"
        fi

        echo "Done. Use '$0 off' to remove limits."
        ;;

    off)
        echo "Removing bandwidth limits on $IFACE"
        sudo tc qdisc del dev "$IFACE" root 2>/dev/null
        sudo tc qdisc del dev "$IFACE" ingress 2>/dev/null
        sudo tc qdisc del dev ifb0 root 2>/dev/null
        sudo ip link set ifb0 down 2>/dev/null
        echo "Done"
        ;;

    status)
        echo "=== $IFACE egress ==="
        tc qdisc show dev "$IFACE" 2>/dev/null
        echo ""
        echo "=== $IFACE ingress ==="
        tc filter show dev "$IFACE" parent ffff: 2>/dev/null
        echo ""
        echo "=== ifb0 ==="
        tc qdisc show dev ifb0 2>/dev/null || echo "(not active)"
        ;;

    *)
        echo "Usage: $0 {on|off|status} [download_mbit] [upload_mbit]"
        echo ""
        echo "Examples:"
        echo "  $0 on           # 2mbit down, 1mbit up"
        echo "  $0 on 5 2       # 5mbit down, 2mbit up"
        echo "  $0 on 0.5 0.25  # 512kbit down, 256kbit up"
        echo "  $0 off           # remove limits"
        echo "  $0 status        # show current rules"
        exit 1
        ;;
esac
