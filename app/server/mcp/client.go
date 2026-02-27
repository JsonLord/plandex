package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Client handles the connection and communication with an MCP server.
type Client struct {
	transport    Transport
	capabilities ClientCapabilities
	serverInfo   ServerInfo
	tools        []Tool
	mu           sync.Mutex
	pending      map[interface{}]chan<- *JsonRpcResponse // Use map[interface{}] for flexible ID handling (int or string)
	nextId       int
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewClient creates a new MCP client with the given transport.
func NewClient(transport Transport) *Client {
	return &Client{
		transport:    transport,
		capabilities: ClientCapabilities{},
		pending:      make(map[interface{}]chan<- *JsonRpcResponse),
		nextId:       1,
	}
}

// Start listens for incoming messages from the transport.
func (c *Client) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.transport.Receive()
			if err != nil {
				return err
			}

			// Handle JSON-RPC message
			// We need to determine if it's a request, response, or notification.
			// This is tricky without full deserialization context, but we can look at "method" (req/notif) vs "result"/"error" (resp).
			// For simplicity in this mock, we assume the transport returns a map[string]interface{}.

			if raw, ok := msg.(map[string]interface{}); ok {
				if _, hasMethod := raw["method"]; hasMethod {
					// Request or Notification
					// TODO: Handle server-sent requests/notifications (e.g., logging)
				} else if id, hasID := raw["id"]; hasID {
					// Response
					// In JSON, numbers are often float64. We need to handle this carefully if IDs were ints.
					// However, we used int for sending. When receiving, JSON unmarshal might make it float64.
					// Let's normalize ID to match what we expect in pending map (which uses the type sent, int).
					// OR better: use float64 consistently for numeric IDs in map key if we can, or just cast.

					// Simplest fix for now: checking map keys with flexible type matching or ensure JSON unmarshal behavior
					// We'll trust the ID as is for now, but in prod we might need type normalization.
					c.handleResponse(raw, id)
				}
			}
		}
	}
}

func (c *Client) handleResponse(raw map[string]interface{}, id interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

    // Normalize ID: if id is float64 (from json unmarshal), convert to int if it matches our pending key type
    var lookupId interface{} = id
    if f, ok := id.(float64); ok {
        lookupId = int(f)
    }

	if ch, ok := c.pending[lookupId]; ok {
		// Convert map back to proper struct for specific type checking if needed
		// Here we just unmarshal raw again into JsonRpcResponse for convenience
		data, _ := json.Marshal(raw)
		var resp JsonRpcResponse
		json.Unmarshal(data, &resp)

		ch <- &resp
		delete(c.pending, lookupId)
		close(ch)
	}
}

// Call sends a request and waits for a response.
func (c *Client) Call(method string, params interface{}) (*JsonRpcResponse, error) {
	id := c.nextId
	c.nextId++

    // ID needs to be interface{}
    var idInterface interface{} = id

	req := JsonRpcRequest{
		JsonRPC: JsonRpcVersion,
		Method:  method,
		ID:      &idInterface,
	}

	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		req.Params = p
	}

	ch := make(chan *JsonRpcResponse, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	if err := c.transport.Send(req); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-time.After(30 * time.Second): // Default timeout
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response to %s", method)
	}
}

// Initialize performs the handshake with the server.
func (c *Client) Initialize(ctx context.Context, clientName, clientVersion string) (*InitializeResult, error) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05", // Use specific date or version string from spec
		Capabilities: ClientCapabilities{
			Roots:    nil, // Start simple
			Sampling: nil,
		},
		ClientInfo: Implementation{
			Name:    clientName,
			Version: clientVersion,
		},
	}

	resp, err := c.Call("initialize", params)
	if err != nil {
		return nil, err
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	c.serverInfo = ServerInfo{
		Name:    result.ServerInfo.Name,
		Version: result.ServerInfo.Version,
	}

	// After initialize, we must send an initialized notification
	notif := JsonRpcNotification{
		JsonRPC: JsonRpcVersion,
		Method:  "notifications/initialized",
	}
	if err := c.transport.Send(notif); err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}

	return &result, nil
}

// ListTools retrieves the available tools from the server.
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	// Simple call, no cursor support yet
	resp, err := c.Call("tools/list", nil)
	if err != nil {
		return nil, err
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools list: %w", err)
	}

	c.tools = result.Tools
	return c.tools, nil
}

// CallTool executes a tool on the server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	argBytes, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	params := CallToolParams{
		Name:      name,
		Arguments: argBytes,
	}

	resp, err := c.Call("tools/call", params)
	if err != nil {
		return nil, err
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool call result: %w", err)
	}

	if result.IsError {
		return &result, fmt.Errorf("tool execution failed (check content for details)")
	}

	return &result, nil
}
