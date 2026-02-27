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

# Install .NET 8 SDK
RUN wget https://packages.microsoft.com/config/ubuntu/22.04/packages-microsoft-prod.deb -O packages-microsoft-prod.deb && \
    dpkg -i packages-microsoft-prod.deb && \
    rm packages-microsoft-prod.deb && \
    apt-get update && \
    apt-get install -y dotnet-sdk-8.0

# Setup Directories
WORKDIR /app

# --- Build .NET Metadata Fetcher ---
COPY app/mcp-servers/whodb /app/mcp-servers/whodb
WORKDIR /app/mcp-servers/whodb
# FORCE REMOVAL of Program.cs if it exists to avoid duplicate entry points
RUN rm -f Program.cs
RUN dotnet publish -c Release -o /app/bin/whodb MetadataFetcher.csproj

# --- Setup Python/SQLite MCP Bridge ---
COPY app/mcp-servers/sqlite-bridge /app/mcp-servers/sqlite-bridge

# --- Setup Power BI MCP Server ---
COPY app/mcp-servers/powerbi /app/mcp-servers/powerbi

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
ENV MCP_POWERBI_HOST=127.0.0.1:8082
ENV DATABASE_URL="postgres://plandex:plandex@127.0.0.1:5432/plandex?sslmode=disable"
# Expose Plandex Server Port
EXPOSE 8080

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
