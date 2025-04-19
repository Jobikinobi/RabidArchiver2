FROM mcr.microsoft.com/devcontainers/go:1.22

# Install basic dependencies
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    ffmpeg \
    python3-pip \
    python3-dev \
    curl \
    sqlite3 \
    libsqlite3-dev \
    git \
    make \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install honnef.co/go/tools/cmd/staticcheck@latest
RUN go install github.com/greptile/greptile/cmd/greptile@latest

# Set the working directory
WORKDIR /workspace

# Set environment variables
ENV CGO_ENABLED=1
ENV PATH="${PATH}:/go/bin"

# Install Whisper.cpp (note: Mac-specific optimizations won't work in this container)
WORKDIR /tmp
RUN git clone https://github.com/ggerganov/whisper.cpp.git && \
    cd whisper.cpp && \
    make && \
    cp ./main /usr/local/bin/whisper && \
    cd .. && \
    rm -rf whisper.cpp

# Install Ollama (though Apple Silicon specific optimizations won't apply)
RUN curl -fsSL https://ollama.com/install.sh | sh 