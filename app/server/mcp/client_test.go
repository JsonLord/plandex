package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MockTransport for testing
type MockTransport struct {
	sent     []interface{}
	received []interface{}
	mu       sync.Mutex
	respChan chan interface{}
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		respChan: make(chan interface{}, 10),
	}
}

func (t *MockTransport) Send(msg interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sent = append(t.sent, msg)

	// Simulate server response based on request
	go func() {
		// Mock logic to determine response
		// In a real mock, we'd need to inspect the message content more deeply
		// Here we assume happy path based on method name embedded in our request struct logic (not really visible here without type assertion, but simplified)

		// We need to re-marshal and unmarshal to access fields genericly or type assert if we know the type

		// Simplified: assumes msg is JsonRpcRequest struct
		if req, ok := msg.(JsonRpcRequest); ok {
			var resp map[string]interface{}

            // Use ID directly as it's passed (pointer to interface{})
            idVal := *req.ID

			switch req.Method {
			case "initialize":
				res := InitializeResult{
					ProtocolVersion: "2024-11-05",
					Capabilities:    ServerCapabilities{},
					ServerInfo: Implementation{
						Name:    "mock-server",
						Version: "1.0.0",
					},
				}
				resBytes, _ := json.Marshal(res)
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      idVal,
					"result":  json.RawMessage(resBytes),
				}
			case "tools/list":
				res := ListToolsResult{
					Tools: []Tool{
						{Name: "mock_tool", Description: "A mock tool", InputSchema: json.RawMessage(`{}`)},
					},
				}
				resBytes, _ := json.Marshal(res)
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      idVal,
					"result":  json.RawMessage(resBytes),
				}
			case "tools/call":
				// Mock execution
				res := CallToolResult{
					Content: []ContentItem{
						{Type: "text", Text: "Tool execution successful"},
					},
				}
				resBytes, _ := json.Marshal(res)
				resp = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      idVal,
					"result":  json.RawMessage(resBytes),
				}
			}

			if resp != nil {
				time.Sleep(10 * time.Millisecond)
				t.respChan <- resp
			}
		}
	}()
	return nil
}

func (t *MockTransport) Receive() (interface{}, error) {
	select {
	case msg := <-t.respChan:
		return msg, nil
	case <-time.After(2 * time.Second): // Longer timeout for test
		return nil, fmt.Errorf("timeout in mock transport receive")
	}
}

func (t *MockTransport) Close() error {
	return nil
}

func TestMcpClientFlow(t *testing.T) {
	transport := NewMockTransport()
	client := NewClient(transport)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start client loop
	go func() {
		if err := client.Start(ctx); err != nil && err != context.Canceled {
			// This might log an error when context is canceled, which is fine
			// t.Logf("Client loop ended: %v", err)
		}
	}()

	// 1. Initialize
	initRes, err := client.Initialize(ctx, "test-client", "0.0.1")
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if initRes.ServerInfo.Name != "mock-server" {
		t.Errorf("Expected server name 'mock-server', got %s", initRes.ServerInfo.Name)
	}

	// 2. List Tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "mock_tool" {
		t.Errorf("Expected 1 tool 'mock_tool', got %v", tools)
	}

	// 3. Call Tool
	callRes, err := client.CallTool(ctx, "mock_tool", map[string]interface{}{"arg": "val"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if len(callRes.Content) == 0 || callRes.Content[0].Text != "Tool execution successful" {
		t.Errorf("Unexpected tool result: %v", callRes)
	}
}
