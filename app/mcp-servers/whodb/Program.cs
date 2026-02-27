using System;
using System.Collections.Generic;
using System.IO;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading.Tasks;
using Microsoft.Data.SqlClient;
using System.Data;
using System.Text;

namespace WhoDbMcp
{
    // --- MCP Types ---

    public class JsonRpcRequest
    {
        [JsonPropertyName("jsonrpc")]
        public string JsonRpc { get; set; } = "2.0";
        [JsonPropertyName("method")]
        public string Method { get; set; }
        [JsonPropertyName("params")]
        public JsonElement Params { get; set; }
        [JsonPropertyName("id")]
        public object? Id { get; set; }
    }

    public class JsonRpcResponse
    {
        [JsonPropertyName("jsonrpc")]
        public string JsonRpc { get; set; } = "2.0";
        [JsonPropertyName("result")]
        public object? Result { get; set; }
        [JsonPropertyName("error")]
        public JsonRpcError? Error { get; set; }
        [JsonPropertyName("id")]
        public object? Id { get; set; }
    }

    public class JsonRpcError
    {
        [JsonPropertyName("code")]
        public int Code { get; set; }
        [JsonPropertyName("message")]
        public string Message { get; set; }
    }

    public class InitializeResult
    {
        [JsonPropertyName("protocolVersion")]
        public string ProtocolVersion { get; set; } = "2024-11-05";
        [JsonPropertyName("capabilities")]
        public ServerCapabilities Capabilities { get; set; } = new();
        [JsonPropertyName("serverInfo")]
        public Implementation ServerInfo { get; set; } = new() { Name = "whodb-mcp", Version = "1.0.0" };
    }

    public class ServerCapabilities
    {
        [JsonPropertyName("tools")]
        public Dictionary<string, object> Tools { get; set; } = new();
    }

    public class Implementation
    {
        [JsonPropertyName("name")]
        public string Name { get; set; } = "whodb-mcp";
        [JsonPropertyName("version")]
        public string Version { get; set; } = "1.0.0";
    }

    public class ListToolsResult
    {
        [JsonPropertyName("tools")]
        public List<Tool> Tools { get; set; } = new();
    }

    public class Tool
    {
        [JsonPropertyName("name")]
        public string Name { get; set; }
        [JsonPropertyName("description")]
        public string Description { get; set; }
        [JsonPropertyName("inputSchema")]
        public object InputSchema { get; set; }
    }

    public class CallToolResult
    {
        [JsonPropertyName("content")]
        public List<ContentItem> Content { get; set; } = new();
        [JsonPropertyName("isError")]
        public bool IsError { get; set; }
    }

    public class ContentItem
    {
        [JsonPropertyName("type")]
        public string Type { get; set; } = "text";
        [JsonPropertyName("text")]
        public string Text { get; set; }
    }

    // --- Main Program ---

    class Program
    {
        static async Task Main(string[] args)
        {
            // Simple stdio loop
            var input = Console.OpenStandardInput();
            var output = Console.OpenStandardOutput();
            using var reader = new StreamReader(input);
            using var writer = new StreamWriter(output) { AutoFlush = true };

            while (true)
            {
                var line = await reader.ReadLineAsync();
                if (line == null) break; // End of stream
                if (string.IsNullOrWhiteSpace(line)) continue;

                try
                {
                    var request = JsonSerializer.Deserialize<JsonRpcRequest>(line);
                    if (request == null) continue;

                    object responseResult = null;

                    switch (request.Method)
                    {
                        case "initialize":
                            responseResult = new InitializeResult();
                            break;
                        case "tools/list":
                            responseResult = GetTools();
                            break;
                        case "tools/call":
                            responseResult = await ExecuteTool(request.Params);
                            break;
                        case "notifications/initialized":
                            // No response needed for notifications
                            continue;
                        default:
                            // Method not found or unsupported
                            break;
                    }

                    if (request.Id != null)
                    {
                        var response = new JsonRpcResponse
                        {
                            Id = request.Id,
                            Result = responseResult
                        };
                        var json = JsonSerializer.Serialize(response);
                        await writer.WriteLineAsync(json);
                    }
                }
                catch (Exception ex)
                {
                    // Log error to stderr
                    Console.Error.WriteLine($"Error processing request: {ex.Message}");

                    // Try to send error response if possible and we have an ID context (skipped for simplicity here)
                }
            }
        }

        static ListToolsResult GetTools()
        {
            return new ListToolsResult
            {
                Tools = new List<Tool>
                {
                    new Tool
                    {
                        Name = "get_db_schema",
                        Description = "Retrieves the database schema (tables and columns) from the SQL database.",
                        InputSchema = new
                        {
                            type = "object",
                            properties = new
                            {
                                connection_string = new { type = "string", description = "The ADO.NET connection string." }
                            },
                            required = new[] { "connection_string" }
                        }
                    },
                    new Tool
                    {
                        Name = "execute_sql_query",
                        Description = "Executes a SQL query against the database.",
                        InputSchema = new
                        {
                            type = "object",
                            properties = new
                            {
                                connection_string = new { type = "string", description = "The ADO.NET connection string." },
                                query = new { type = "string", description = "The SQL query to execute." }
                            },
                            required = new[] { "connection_string", "query" }
                        }
                    }
                }
            };
        }

        static async Task<CallToolResult> ExecuteTool(JsonElement paramsElement)
        {
            try
            {
                var name = paramsElement.GetProperty("name").GetString();
                var args = paramsElement.GetProperty("arguments");

                string connString = "";
                string query = "";

                // Handle arguments potentially being stringified JSON or an object
                if (args.ValueKind == JsonValueKind.String)
                {
                    var argsObj = JsonSerializer.Deserialize<JsonElement>(args.GetString());
                    if (argsObj.TryGetProperty("connection_string", out var cs)) connString = cs.GetString();
                    if (argsObj.TryGetProperty("query", out var q)) query = q.GetString();
                }
                else
                {
                     if (args.TryGetProperty("connection_string", out var cs)) connString = cs.GetString();
                     if (args.TryGetProperty("query", out var q)) query = q.GetString();
                }

                if (string.IsNullOrEmpty(connString))
                {
                    return new CallToolResult { IsError = true, Content = new List<ContentItem> { new ContentItem { Text = "connection_string is required" } } };
                }

                if (name == "get_db_schema")
                {
                    var schema = await GetSchema(connString);
                    return new CallToolResult
                    {
                        Content = new List<ContentItem> { new ContentItem { Text = schema } }
                    };
                }
                else if (name == "execute_sql_query")
                {
                    if (string.IsNullOrEmpty(query))
                    {
                         return new CallToolResult { IsError = true, Content = new List<ContentItem> { new ContentItem { Text = "query is required" } } };
                    }
                    var result = await ExecuteQuery(connString, query);
                    return new CallToolResult
                    {
                        Content = new List<ContentItem> { new ContentItem { Text = result } }
                    };
                }

                return new CallToolResult { IsError = true, Content = new List<ContentItem> { new ContentItem { Text = "Tool not found" } } };
            }
            catch (Exception ex)
            {
                return new CallToolResult { IsError = true, Content = new List<ContentItem> { new ContentItem { Text = $"Error: {ex.Message}" } } };
            }
        }

        static async Task<string> GetSchema(string connectionString)
        {
            try
            {
                using var conn = new SqlConnection(connectionString);
                await conn.OpenAsync();

                // Fallback to manual query if GetSchema doesn't return what we expect (standard for SQL Server)
                var sb = new StringBuilder();
                var schemaQuery = @"
                    SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE
                    FROM INFORMATION_SCHEMA.COLUMNS
                    ORDER BY TABLE_NAME, ORDINAL_POSITION";

                using var cmd = new SqlCommand(schemaQuery, conn);
                using var reader = await cmd.ExecuteReaderAsync();

                string currentTable = "";
                while (await reader.ReadAsync())
                {
                    var tableName = reader["TABLE_NAME"].ToString();
                    var colName = reader["COLUMN_NAME"].ToString();
                    var dataType = reader["DATA_TYPE"].ToString();

                    if (tableName != currentTable)
                    {
                        if (currentTable != "") sb.AppendLine();
                        sb.AppendLine($"Table: {tableName}");
                        currentTable = tableName;
                    }
                    sb.AppendLine($"  - {colName} ({dataType})");
                }

                return sb.ToString();
            }
            catch (Exception ex)
            {
                return $"Failed to get schema: {ex.Message}";
            }
        }

        static async Task<string> ExecuteQuery(string connectionString, string query)
        {
            try
            {
                using var conn = new SqlConnection(connectionString);
                await conn.OpenAsync();

                using var cmd = new SqlCommand(query, conn);
                using var reader = await cmd.ExecuteReaderAsync();

                var sb = new StringBuilder();

                // Get column names
                for (int i = 0; i < reader.FieldCount; i++)
                {
                    if (i > 0) sb.Append("\t");
                    sb.Append(reader.GetName(i));
                }
                sb.AppendLine();

                // Get rows (limit to 100 for safety)
                int rowCount = 0;
                while (await reader.ReadAsync() && rowCount < 100)
                {
                    for (int i = 0; i < reader.FieldCount; i++)
                    {
                        if (i > 0) sb.Append("\t");
                        sb.Append(reader[i].ToString());
                    }
                    sb.AppendLine();
                    rowCount++;
                }

                if (rowCount >= 100)
                {
                    sb.AppendLine("... (truncated at 100 rows)");
                }

                return sb.ToString();
            }
            catch (Exception ex)
            {
                return $"Failed to execute query: {ex.Message}";
            }
        }
    }
}
