FROM ubuntu:22.04

# Avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    git \
    gnupg \
    ca-certificates \
    lsb-release \
    socat \
    supervisor \
    postgresql \
    postgresql-contrib \
    python3 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

# Install LiteLLM Proxy dependencies
RUN pip3 install --no-cache-dir litellm[proxy] uvicorn fastapi

# Install Go 1.23
RUN wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz && \
    rm go1.23.0.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# Setup Directories
WORKDIR /app

# --- Setup Python/SQLite MCP Bridge ---
COPY app/mcp-servers/sqlite-bridge /app/mcp-servers/sqlite-bridge

# --- Build Plandex Server ---
WORKDIR /app/server
COPY app/server .
WORKDIR /app/src
COPY app .
WORKDIR /app/src/server
RUN go build -o /app/bin/plandex-server

# --- Setup Postgres ---
# Initialize DB
RUN service postgresql start && \
    su - postgres -c "psql -c \"CREATE USER plandex WITH PASSWORD 'plandex';\"" && \
    su - postgres -c "psql -c \"CREATE DATABASE plandex OWNER plandex;\"" && \
    service postgresql stop

# --- Supervisor Configuration ---
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# --- Final Setup ---
WORKDIR /app
ENV MCP_WHODB_HOST=127.0.0.1:8081
ENV DATABASE_URL="postgres://plandex:plandex@127.0.0.1:5432/plandex?sslmode=disable"
# Expose Plandex Server Port
EXPOSE 8080

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
