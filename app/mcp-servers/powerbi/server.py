import sys
import json

def read_message():
    line = sys.stdin.readline()
    if not line: return None
    try: return json.loads(line)
    except: return None

def write_message(msg):
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()

def handle_request(req):
    method = req.get("method")
    req_id = req.get("id")
    result = None

    if method == "initialize":
        result = {
            "protocolVersion": "2024-11-05",
            "capabilities": {"tools": {}},
            "serverInfo": {"name": "powerbi-mcp", "version": "1.0.0"}
        }
    elif method == "tools/list":
        result = {
            "tools": [
                {
                    "name": "list_dashboards",
                    "description": "List available Power BI dashboards.",
                    "inputSchema": {"type": "object", "properties": {}}
                }
            ]
        }
    elif method == "tools/call":
        # Mock implementation for now
        result = {
            "content": [{"type": "text", "text": "Power BI dashboard list: [Sales Overview, Marketing Report]"}]
        }

    if req_id is not None:
        write_message({"jsonrpc": "2.0", "id": req_id, "result": result})

if __name__ == "__main__":
    while True:
        req = read_message()
        if req is None: break
        handle_request(req)
