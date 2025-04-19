# Devcontainer Debug Tools

This directory contains tools to help diagnose and fix container build failures in GitHub Codespaces.

## Tools Available

1. **debug-build.sh**: Runs during container build to collect diagnostic information
2. **entrypoint.sh**: Runs when container starts to collect startup logs
3. **collect-logs.sh**: Run this script after a build failure to gather comprehensive logs

## How to Use

### When a Container Build Fails:

1. Run the log collection script from your local environment:
   ```
   chmod +x .devcontainer/collect-logs.sh
   ./.devcontainer/collect-logs.sh
   ```

2. This will create a zip file with all diagnostic information that you can examine or share with support.

### Common Build Failure Fixes

1. **BuildKit Issues**: 
   - Add a GitHub Codespaces secret named `DOCKER_BUILDKIT` with value `0`
   - This disables BuildKit which has been reported to resolve the common error:
     `error: failed to receive status: rpc error: code = Unavailable desc = error reading from server: EOF`

2. **Environment File Issues**:
   - Make sure `.env` file exists and is valid or create an empty one
   - If you don't need environment variables, remove the `runArgs` that reference it

3. **Network/Resource Issues**:
   - Check if you're hitting resource limits (CPU/memory)
   - Simplify your Dockerfile to use fewer layers
   - Add timeouts or retries for network operations

## Log Files

After container startup, logs are available in:
- `/workspace/logs/container-startup.log` - Container startup diagnostics
- `/workspace/logs/devcontainer/` - Logs from the devcontainer system

## GitHub Codespaces Support

For persistent issues, you can contact GitHub Support with:
1. The detailed logs collected using `collect-logs.sh`
2. Your devcontainer configuration files
3. The specific error messages you're seeing 