import sys
import json
import sqlite3
import subprocess
import os

# --- MCP JSON-RPC Server Implementation ---

def read_message():
    line = sys.stdin.readline()
    if not line:
        return None
    try:
        return json.loads(line)
    except json.JSONDecodeError:
        return None

def write_message(msg):
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()

def init_db():
    conn = sqlite3.connect("metadata.db")
    cursor = conn.cursor()
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS schema_cache (
            id INTEGER PRIMARY KEY,
            table_name TEXT,
            column_name TEXT,
            data_type TEXT,
            last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.commit()
    conn.close()

def refresh_schema(connection_string):
    try:
        # Call .NET MetadataFetcher
        # Adjust path to where the binary is published in Dockerfile
        result = subprocess.run(
            ["dotnet", "/app/bin/whodb/MetadataFetcher.dll", connection_string],
            capture_output=True,
            text=True
        )

        if result.returncode != 0:
            return f"Error fetching metadata: {result.stderr}"

        schema_json = result.stdout
        tables = json.loads(schema_json)

        conn = sqlite3.connect("metadata.db")
        cursor = conn.cursor()

        # Clear old cache (simple approach)
        cursor.execute("DELETE FROM schema_cache")

        for table in tables:
            t_name = table["Name"]
            for col in table["Columns"]:
                cursor.execute(
                    "INSERT INTO schema_cache (table_name, column_name, data_type) VALUES (?, ?, ?)",
                    (t_name, col["Name"], col["Type"])
                )

        conn.commit()
        conn.close()
        return "Schema refreshed successfully."
    except Exception as e:
        return f"Failed to refresh schema: {str(e)}"

def get_schema_summary():
    conn = sqlite3.connect("metadata.db")
    cursor = conn.cursor()
    cursor.execute("SELECT table_name, column_name, data_type FROM schema_cache ORDER BY table_name")
    rows = cursor.fetchall()
    conn.close()

    if not rows:
        return "No schema cached. Please run refresh_metadata first."

    summary = []
    current_table = None
    for r in rows:
        t_name, c_name, d_type = r
        if t_name != current_table:
            summary.append(f"Table: {t_name}")
            current_table = t_name
        summary.append(f"  - {c_name} ({d_type})")

    return "\n".join(summary)

def handle_request(req):
    method = req.get("method")
    params = req.get("params", {})
    req_id = req.get("id")

    result = None
    error = None

    if method == "initialize":
        result = {
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {
                    "refresh_metadata": {},
                    "get_cached_schema": {}
                }
            },
            "serverInfo": {"name": "sqlite-metadata-bridge", "version": "1.0.0"}
        }
    elif method == "tools/list":
        result = {
            "tools": [
                {
                    "name": "refresh_metadata",
                    "description": "Fetches database schema from the configured SQL Server and caches it locally in SQLite. Use this to update the agent's view of the database structure.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {
                            "connection_string": {"type": "string", "description": "ADO.NET connection string for the remote database."}
                        },
                        "required": ["connection_string"]
                    }
                },
                {
                    "name": "get_cached_schema",
                    "description": "Retrieves the currently cached database schema from the local SQLite database.",
                    "inputSchema": {
                        "type": "object",
                        "properties": {},
                    }
                }
            ]
        }
    elif method == "tools/call":
        name = params.get("name")
        args = params.get("arguments", {})
        if isinstance(args, str):
            try:
                args = json.loads(args)
            except:
                pass

        if name == "refresh_metadata":
            conn_str = args.get("connection_string")
            if not conn_str:
                error = {"code": -32602, "message": "Missing connection_string"}
            else:
                msg = refresh_schema(conn_str)
                result = {"content": [{"type": "text", "text": msg}]}
        elif name == "get_cached_schema":
            msg = get_schema_summary()
            result = {"content": [{"type": "text", "text": msg}]}
        else:
            error = {"code": -32601, "message": "Method not found"}

    elif method == "notifications/initialized":
        return # No response

    if req_id is not None:
        resp = {"jsonrpc": "2.0", "id": req_id}
        if error:
            resp["error"] = error
        else:
            resp["result"] = result
        write_message(resp)

if __name__ == "__main__":
    init_db()
    while True:
        req = read_message()
        if req is None:
            break
        handle_request(req)
