#!/bin/bash
# Cleanup satellite and SPIRE agent on Raspberry Pi
echo "=== Cleaning up Satellite on Pi ==="

echo "> Stopping SPIRE agent..."
sudo pkill -f "spire-agent" 2>/dev/null || true
sleep 1

echo "> Stopping satellite..."
pkill -f "./satellite" 2>/dev/null || true

echo "> Removing SPIRE agent data and sockets..."
sudo rm -rf /opt/spire/data/agent/*
sudo rm -rf /run/spire/sockets/*

echo "> Removing satellite config..."
rm -rf ~/.config/satellite

echo "Cleanup complete"
