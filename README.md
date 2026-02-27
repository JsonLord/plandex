---
title: Plandex WhoDB
emoji: 🧠
colorFrom: blue
colorTo: purple
sdk: docker
app_port: 8080
pinned: false
---

# Plandex WhoDB & Power BI Integration

This Space runs Plandex with integrated WhoDB (Azure SQL) and Power BI support via MCP.

## Configuration

Set the following secrets in the Space settings:

* `BLABLADOR_API_KEY`: API Key for Blablador LLM.
* `AZURE_SQL_CONNECTION_STRING`: Connection string for metadata sync (ADO.NET format).
* `POWERBI_CLIENT_ID`: Power BI Client ID (optional for mock).
* `POWERBI_SECRET`: Power BI Client Secret (optional for mock).
* `OPENAI_API_KEY` (Optional): Fallback.

## Architecture

* **Single Container**: Runs Plandex Server (Go), MCP Servers (Python/.NET), and PostgreSQL.
* **Internal Connections**:
    * **SQLite/WhoDB MCP**: `127.0.0.1:8081` (Python bridge to .NET Fetcher).
    * **Power BI MCP**: `127.0.0.1:8082` (Python).
    * **Database**: Local Postgres.
* **LLM**: Defaults to `blablador/alias-large` for reasoning, `blablador/alias-huge` for investigation, and `blablador/alias-code` for coding.

## Usage

1. **Interface**: Access the UI to start a "Data" or "Power BI" chat.
2. **Sync**: Click "Synchronize DB Metadata" to fetch the schema from Azure SQL into the local cache.
3. **Chat**: Ask the agent to query the database or list Power BI dashboards.
