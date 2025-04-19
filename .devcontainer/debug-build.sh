#!/bin/bash
# Diagnostic script for container build failures

echo "=== STARTING BUILD DIAGNOSTICS ===" 
date

echo "=== SYSTEM INFORMATION ==="
uname -a
cat /etc/os-release
df -h
free -h

echo "=== DOCKER INFORMATION ==="
if command -v docker &> /dev/null; then
    docker --version
    docker info
else
    echo "Docker not installed or not in PATH"
fi

echo "=== ENVIRONMENT VARIABLES ==="
env | grep -E 'BUILDKIT|DOCKER|CODESPACE|GITHUB|DEV|CONTAINER' | sort

echo "=== NETWORK CONNECTIVITY TESTS ==="
ping -c 3 github.com || echo "Cannot ping github.com"
ping -c 3 8.8.8.8 || echo "Cannot ping 8.8.8.8"
curl -s -o /dev/null -w "%{http_code}" https://github.com || echo "Cannot reach github.com via HTTPS"

echo "=== END BUILD DIAGNOSTICS ===" 