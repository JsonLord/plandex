# Project Plan: Plandex "WhoDB" & MCP Integration

## 1. Project Description

### Vision and Goals
The goal is to extend Plandex's capabilities to interact with enterprise data sources, specifically an Azure SQL database ("WhoDB") and Power BI, by integrating the **Model Context Protocol (MCP)**. This enables the LLM agents to securely query database schemas, execute SQL queries via **ADO.NET** connections, and interact with Power BI datasets within the same "hacking phase space" (Docker environment).

### In-app Integrations
*   **Plandex Server (Go)**: Acts as the **MCP Client**. It will be responsible for discovering and connecting to MCP servers, and exposing their tools to the LLM during the planning and execution phases.
*   **WhoDB MCP Server (C#/.NET)**: A dedicated MCP server implementation running as a sidecar or subprocess. It handles the specific logic for connecting to Azure SQL using **ADO.NET**. It exposes tools for `GetSchema`, `ExecuteQuery`, etc.
*   **Power BI MCP Server**: A separate MCP server for Power BI interactions.
*   **Shared Docker Environment**: All components run within the `plandex-network` to ensure seamless communication and file access ("hugging face space file system").

### Proposed Architecture
*   **Backend (Go)**:
    *   New package `app/server/mcp` to handle MCP protocol (JSON-RPC 2.0 over Stdio or SSE).
    *   Integration with `app/server/model` to inject MCP tools into the `OpenAI` request structure.
*   **MCP Servers**:
    *   Located in `app/mcp-servers/`.
    *   **WhoDB**: .NET 8 Console Application using `Microsoft.Data.SqlClient`.
*   **Configuration**:
    *   Environment variables for Connection Strings (ADO.NET formats).
    *   `PLAN_SETTINGS` extended to enable/disable specific MCP servers.

## 2. Tasks and Tests

### Phase 1: Infrastructure & MCP Client (Go)
1.  **Implement MCP Client Interface in Go**
    *   *Task*: Create `app/server/mcp/client.go` implementing the MCP specification (tool discovery, call execution).
    *   *Test*: Unit test `TestMcpToolDiscovery` mocking an MCP server via stdio.
2.  **Integrate MCP Tools into LLM Context**
    *   *Task*: Modify `app/server/model` to append discovered MCP tools to the `tools` array in OpenAI/Anthropic requests.
    *   *Test*: Integration test verifying that a dummy tool is sent in the API request to the LLM.

### Phase 2: WhoDB MCP Server (.NET)
3.  **Create WhoDB MCP Server Scaffold**
    *   *Task*: Initialize a C# project in `app/mcp-servers/whodb`.
    *   *Test*: Build test in CI to ensure .NET environment is correctly set up.
4.  **Implement ADO.NET Connection Logic**
    *   *Task*: Implement connection factory supporting the 4 specified ADO.NET connection string formats (SQL Auth, Entra Password, Entra Passwordless, Entra Integrated).
    *   *Test*: Unit test parsing connection strings and instantiating `SqlConnection`.
5.  **Implement Schema & Query Tools**
    *   *Task*: Implement `get_db_schema` and `execute_sql_query` tools exposed via MCP.
    *   *Test*: Integration test against a LocalDB or Dockerized SQL Server instance.

### Phase 3: Power BI MCP Server
6.  **Create Power BI MCP Server**
    *   *Task*: Initialize project in `app/mcp-servers/powerbi`.
    *   *Test*: Mock test for Power BI API authentication.

### Phase 4: Docker & Deployment
7.  **Update Docker Compose**
    *   *Task*: Add services for `whodb-mcp` and `powerbi-mcp` or configure `plandex-server` to spawn them.
    *   *Test*: `docker-compose up` verifies all containers start and can ping each other.

## 3. Functionality Expectations

### User Perspective
*   The user can instruct the agent: "Check the sales table in WhoDB and tell me the top region."
*   The agent automatically recognizes the `whodb` toolset.
*   The agent queries the schema first, then writes a correct SQL query, executes it, and interprets the results.

### Technical Perspective
*   **Default Connection**: The system defaults to **ADO.NET** for SQL interactions.
*   **Security**: Connection strings are sensitive; they should be stored in `authVars` or environment variables, not hardcoded in the plan.
*   **Data Flow**: `LLM -> Plandex Server -> (MCP Protocol) -> WhoDB MCP -> Azure SQL -> Result -> Plandex Server -> LLM`.

### Constraints
*   Must run within the existing Docker composition.
*   .NET Runtime must be available for the WhoDB MCP server.

## 4. API Endpoints to be Exposed

Since MCP Integration is primarily backend logic, new HTTP APIs are minimal, mostly for configuration.

1.  **GET /api/mcp/status**
    *   *Description*: Check the health and connection status of registered MCP servers.
    *   *Response*: `{"whodb": "connected", "powerbi": "disconnected"}`
2.  **POST /api/projects/{projectId}/settings/mcp**
    *   *Description*: Configure MCP server settings (e.g., enable/disable, set connection string references).
    *   *Request*: `{"whodb_enabled": true, "connection_string_var": "AZURE_SQL_CONN_STRING"}`
