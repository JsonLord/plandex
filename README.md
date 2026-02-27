---
title: Plandex WhoDB
emoji: 🧠
colorFrom: blue
colorTo: purple
sdk: docker
app_port: 8080
pinned: false
---

# Plandex WhoDB Integration

This Space runs Plandex with integrated WhoDB support via MCP.

## Configuration

Set the following secrets in the Space settings:

* `BLABLADOR_API_KEY`: API Key for Blablador LLM.
* `OPENAI_API_KEY` (Optional): Fallback.

## Architecture

* **Single Container**: Runs Plandex Server (Go), WhoDB MCP Server (.NET 8), and PostgreSQL.
* **Internal Connections**:
    * **WhoDB MCP**: Configured to `127.0.0.1:8080` (TCP/Socat).
    * **Database**: Configured to `postgres://plandex:plandex@127.0.0.1:5432/plandex`.
* **LLM**: Defaults to `blablador/alias-large` for reasoning and `blablador/alias-huge` for investigation.

## Usage

The agent can automatically query the configured Azure SQL database (if connection string is provided in the prompt or context) using the `execute_sql_query` tool.
