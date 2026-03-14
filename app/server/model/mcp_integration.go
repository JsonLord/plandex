package model

import (
	"context"
	"fmt"
	"log"
	"plandex-server/mcp"
	"sync"
)

// McpManager handles the lifecycle of MCP clients for a given plan execution context
type McpManager struct {
	clients map[string]*mcp.Client
	mu      sync.Mutex
}

var (
	// Global manager for now, in a real app this might be scoped to org/project
	globalMcpManager *McpManager
	mcpOnce          sync.Once
)

func GetMcpManager() *McpManager {
	mcpOnce.Do(func() {
		globalMcpManager = &McpManager{
			clients: make(map[string]*mcp.Client),
		}
	})
	return globalMcpManager
}

// RegisterClient adds a new MCP client (e.g., initialized from config)
func (m *McpManager) RegisterClient(name string, client *mcp.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[name] = client
}

// GetClient retrieves a client by name
func (m *McpManager) GetClient(name string) *mcp.Client {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clients[name]
}

// GetAllTools aggregates tools from all registered MCP clients
func (m *McpManager) GetAllTools(ctx context.Context) ([]mcp.Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var allTools []mcp.Tool
	for _, client := range m.clients {
		tools, err := client.ListTools(ctx)
		if err != nil {
			log.Printf("Error listing tools from MCP client: %v", err)
			continue // Skip failing clients for now
		}
		allTools = append(allTools, tools...)
	}
	return allTools, nil
}

// ExecuteTool finds the client responsible for the tool and executes it
// Note: This assumes tool names are unique across clients or we need a way to namespace them
func (m *McpManager) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	m.mu.Lock()
	clientMap := make(map[string]*mcp.Client)
	for k, v := range m.clients {
		clientMap[k] = v
	}
	m.mu.Unlock()

	if len(clientMap) == 0 {
		return nil, fmt.Errorf("no MCP clients registered")
	}

	// Naive search: ask each client if they have the tool.
	// Optimization: Cache tool->client mapping during ListTools

	var lastErr error
	for name, client := range clientMap {
		tools, err := client.ListTools(ctx) // Inefficient, should cache
		if err != nil {
			log.Printf("Error listing tools for client %s: %v", name, err)
			lastErr = err
			continue
		}
		for _, t := range tools {
			if t.Name == toolName {
				log.Printf("Executing tool %s on client %s", toolName, name)
				return client.CallTool(ctx, toolName, args)
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("tool not found: %s (last error: %v)", toolName, lastErr)
	}

	return nil, fmt.Errorf("tool not found: %s", toolName)
}
