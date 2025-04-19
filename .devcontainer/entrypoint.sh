#!/bin/bash
# Don't exit on error so script can continue
set +e

echo "Starting entrypoint script..."

# Create logs directory - try with and without sudo
mkdir -p /workspace/logs 2>/dev/null || sudo mkdir -p /workspace/logs 2>/dev/null || {
  echo "Failed to create /workspace/logs, trying alternative path..."
  # Try to find the actual workspace directory
  for potential_dir in /workspaces/* /workspace /home/vscode; do
    if [ -d "$potential_dir" ]; then
      mkdir -p "$potential_dir/logs" 2>/dev/null && {
        echo "Created logs directory at $potential_dir/logs"
        export LOG_DIR="$potential_dir/logs"
        break
      }
    fi
  done
  # If still no log directory, use /tmp
  if [ -z "$LOG_DIR" ]; then
    export LOG_DIR="/tmp"
    echo "Using $LOG_DIR for logs"
  fi
} || {
  export LOG_DIR="/tmp"
  echo "Falling back to /tmp for logs"
}

# Collect system information on container start
{
  echo "=== CONTAINER STARTUP LOG ==="
  date
  echo "=== SYSTEM INFO ==="
  uname -a
  echo "=== DISK USAGE ==="
  df -h
  echo "=== MEMORY INFO ==="
  free -h 2>/dev/null || vm_stat 2>/dev/null || echo "Could not get memory info"
  echo "=== DOCKER INFO ==="
  if command -v docker &> /dev/null; then
    docker --version 2>/dev/null || echo "Docker version command failed"
    docker info 2>/dev/null || echo "Docker info command failed"
  else
    echo "Docker command not found"
  fi
  echo "=== BUILD DIAGNOSTICS ==="
  if [ -f /tmp/build-diagnostics.log ]; then
    cat /tmp/build-diagnostics.log
  else
    echo "No build diagnostics log found"
  fi
} > "$LOG_DIR/container-startup.log" 2>&1 || echo "Failed to write startup log"

# Check if there were errors in any previous container build attempts
if [ -d /var/log/devcontainer ]; then
  cp -r /var/log/devcontainer "$LOG_DIR/" 2>/dev/null || echo "Could not copy devcontainer logs"
fi

echo "Container diagnostic logs written to $LOG_DIR/"
echo "If your container fails to build in the future, these logs will be available"

# Execute whatever command was passed, or default to bash
if [ $# -eq 0 ]; then
  echo "No command specified, starting bash shell"
  exec /bin/bash
else
  echo "Executing command: $@"
  exec "$@"
fi 