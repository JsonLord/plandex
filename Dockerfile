FROM ubuntu:22.04

# Avoid interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies
# - Go (for Plandex Server)
# - .NET 8 (for WhoDB MCP)
# - PostgreSQL (for local state)
# - Socat (for MCP TCP bridging)
# - Python3/Pip (for integration scripts if needed)
# - Supervisor (process management)
# - Git/Curl (utilities)
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

# Install Go 1.23
RUN wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz && \
    rm go1.23.0.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

# Install .NET 8 SDK
RUN wget https://packages.microsoft.com/config/ubuntu/22.04/packages-microsoft-prod.deb -O packages-microsoft-prod.deb && \
    dpkg -i packages-microsoft-prod.deb && \
    rm packages-microsoft-prod.deb && \
    apt-get update && \
    apt-get install -y dotnet-sdk-8.0

# Setup Directories
WORKDIR /app

# --- Build WhoDB MCP Server ---
COPY app/mcp-servers/whodb /app/mcp-servers/whodb
WORKDIR /app/mcp-servers/whodb
RUN dotnet publish -c Release -o /app/bin/whodb

# --- Build Plandex Server ---
WORKDIR /app/server
COPY app/server .
# Copy shared module (assuming it's a sibling in the repo structure, usually handled by go.mod replace)
# We need to copy the whole 'app' context or fix paths.
# Let's copy the entire app context to /app/src first to be safe with relative imports
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
# Set environment variables "already" via localhost as requested
ENV MCP_WHODB_HOST=127.0.0.1:8080
ENV DATABASE_URL="postgres://plandex:plandex@127.0.0.1:5432/plandex?sslmode=disable"
# Expose Plandex Server Port
EXPOSE 8080

# Hugging Face Spaces Entrypoint
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
