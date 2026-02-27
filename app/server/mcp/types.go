package mcp

import (
	"encoding/json"
)

// MCP 2.0 JSON-RPC Types

const (
	JsonRpcVersion = "2.0"
)

type JsonRpcRequest struct {
	JsonRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      *interface{}    `json:"id,omitempty"` // Can be int, string, or nil for notifications
}

type JsonRpcResponse struct {
	JsonRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JsonRpcError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

type JsonRpcNotification struct {
	JsonRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JsonRpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// MCP Specific Payload Types

// InitializeRequest params
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation  `json:"clientInfo"`
}

type ClientCapabilities struct {
	Roots        *bool `json:"roots,omitempty"`
	Sampling     *bool `json:"sampling,omitempty"`
	Experimental *map[string]interface{} `json:"experimental,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult result
type InitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation  `json:"serverInfo"`
}

type ServerCapabilities struct {
	Logging      *map[string]interface{} `json:"logging,omitempty"`
	Prompts      *map[string]interface{} `json:"prompts,omitempty"`
	Resources    *map[string]interface{} `json:"resources,omitempty"`
	Tools        *map[string]interface{} `json:"tools,omitempty"`
	Experimental *map[string]interface{} `json:"experimental,omitempty"`
}

// Tool Types
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"` // JSON Schema
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type     string `json:"type"` // "text", "image", "resource"
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`     // Base64 for images
	MimeType string `json:"mimeType,omitempty"` // For images
	Resource interface{} `json:"resource,omitempty"` // For resources
}
