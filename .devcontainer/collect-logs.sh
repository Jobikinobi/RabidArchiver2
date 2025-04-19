#!/bin/bash
# Script to collect logs after a failed container build

LOGDIR="container-build-logs-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$LOGDIR"

echo "Collecting logs to $LOGDIR..."

# Collect system information
{
  echo "=== SYSTEM INFORMATION ==="
  date
  uname -a
  echo "=== DISK SPACE ==="
  df -h
  echo "=== MEMORY ==="
  free -h 2>/dev/null || vm_stat 2>/dev/null || echo "Could not get memory info"
} > "$LOGDIR/system-info.log" 2>&1

# Collect Docker information
{
  echo "=== DOCKER INFORMATION ==="
  docker --version 2>/dev/null || echo "Docker not available"
  docker info 2>/dev/null || echo "Cannot get docker info"
  
  echo "=== DOCKER IMAGES ==="
  docker images 2>/dev/null || echo "Cannot list docker images"
  
  echo "=== DOCKER CONTAINERS ==="
  docker ps -a 2>/dev/null || echo "Cannot list docker containers"
  
  echo "=== DOCKER BUILDX ==="
  docker buildx version 2>/dev/null || echo "Buildx not available"
} > "$LOGDIR/docker-info.log" 2>&1

# Collect GitHub CLI information
{
  echo "=== GITHUB CLI INFORMATION ==="
  gh --version 2>/dev/null || echo "GitHub CLI not available"
  
  echo "=== CODESPACES INFORMATION ==="
  gh codespace list 2>/dev/null || echo "Cannot list codespaces"
} > "$LOGDIR/github-info.log" 2>&1

# Collect environment variables
{
  echo "=== ENVIRONMENT VARIABLES ==="
  env | grep -E 'BUILDKIT|DOCKER|CODESPACE|GITHUB|DEV|CONTAINER' | sort
} > "$LOGDIR/env-vars.log" 2>&1

# Check for devcontainer logs
{
  echo "=== DEVCONTAINER LOGS ==="
  for log in ~/.vscode-remote/extensions/ms-vscode-remote.remote-containers-*/scripts/logs/*.log; do
    if [ -f "$log" ]; then
      echo "=== $log ==="
      cat "$log"
      echo ""
    fi
  done
  
  # Check for specific GitHub Codespaces logs
  if [ -d ~/.codespaces ]; then
    echo "=== CODESPACES LOGS ==="
    find ~/.codespaces -name "*.log" -type f -exec echo "=== {} ===" \; -exec cat {} \; -exec echo "" \;
  fi
} > "$LOGDIR/devcontainer-logs.log" 2>&1

# Copy devcontainer files for reference
{
  echo "=== DEVCONTAINER FILES ==="
  mkdir -p "$LOGDIR/devcontainer-files"
  cp -r .devcontainer/* "$LOGDIR/devcontainer-files/" 2>/dev/null || echo "Could not copy devcontainer files"
  
  echo "=== DEVCONTAINER FILES CONTENT ==="
  for file in "$LOGDIR/devcontainer-files"/*; do
    if [ -f "$file" ]; then
      echo "=== $file ==="
      cat "$file"
      echo ""
    fi
  done
} > "$LOGDIR/devcontainer-files.log" 2>&1

# Zip everything up
if command -v zip >/dev/null 2>&1; then
  zip -r "$LOGDIR.zip" "$LOGDIR" >/dev/null 2>&1
  echo "Logs collected and compressed to $LOGDIR.zip"
  echo "You can attach this file when reporting the issue to GitHub Support"
else
  echo "Logs collected to $LOGDIR directory"
fi 