FROM mcr.microsoft.com/devcontainers/go:1.22

# Explicitly disable BuildKit inline cache to avoid common Codespaces issues
ARG BUILDKIT_INLINE_CACHE=0
ENV BUILDKIT_INLINE_CACHE=0

# Enable verbose logging for better diagnostics
ARG BUILDKIT_PROGRESS=plain
ENV BUILDKIT_PROGRESS=${BUILDKIT_PROGRESS}
ENV DOCKER_BUILDKIT=0

# For diagnostics
RUN echo "=== Starting container build: $(date) ===" && \
    echo "Docker version: $(cat /etc/os-release | grep PRETTY_NAME)" && \
    echo "Available disk space: $(df -h / | awk 'NR==2 {print $4}')" 

# Install basic dependencies - log each step separately for better diagnostics
RUN apt-get update && \
    echo "APT update completed successfully"

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get -y install --no-install-recommends \
    ffmpeg \
    sqlite3 \
    libsqlite3-dev \
    python3-pip \
    python3-dev \
    curl \
    git \
    make && \
    echo "Core dependencies installed successfully"

# Log cleanup step
RUN apt-get clean -y && \
    rm -rf /var/lib/apt/lists/* && \
    echo "Cleanup completed successfully"

# Install Go tools - log each step separately
RUN echo "Installing Go tools..." && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    echo "staticcheck installed successfully"

RUN go install github.com/greptile/greptile/cmd/greptile@latest && \
    echo "greptile installed successfully"

# Set the working directory
WORKDIR /workspace

# Set environment variables
ENV CGO_ENABLED=1
ENV PATH="${PATH}:/go/bin"

# Install Whisper.cpp with error handling and timeout
WORKDIR /tmp
RUN echo "Starting Whisper.cpp installation..." && \
    timeout 300 git clone --depth=1 https://github.com/ggerganov/whisper.cpp.git || echo "Whisper clone timed out or failed - continuing" && \
    if [ -d "whisper.cpp" ]; then \
        cd whisper.cpp && \
        make -j4 || echo "Whisper make failed - continuing" && \
        if [ -f "./main" ]; then \
            cp ./main /usr/local/bin/whisper || echo "Whisper binary copy failed"; \
        fi && \
        cd .. && \
        rm -rf whisper.cpp; \
    fi && \
    echo "Whisper.cpp installation completed or skipped"

# Install Ollama with error handling
RUN echo "Starting Ollama installation..." && \
    (curl -m 120 -fsSL https://ollama.com/install.sh | sh) || echo "Ollama installation failed - continuing" && \
    echo "Ollama installation completed or skipped"

# Copy and run diagnostic script
COPY .devcontainer/debug-build.sh /tmp/debug-build.sh
RUN chmod +x /tmp/debug-build.sh && \
    /tmp/debug-build.sh > /tmp/build-diagnostics.log 2>&1 || true

# Final diagnostic message before completion
RUN echo "=== Container build process completed: $(date) ===" 