package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"plandex-server/mcp"
	"plandex-server/model"
	"plandex-server/routes"
	"plandex-server/setup"
	"time"

	"github.com/gorilla/mux"
)

func logEnvVarPresence(key string) {
	val := os.Getenv(key)
	if val != "" {
		// Log masked value for security (first 4 chars)
		masked := val
		if len(val) > 4 {
			masked = val[:4] + "..."
		}
		log.Printf("Environment variable %s is SET (starts with: %s)", key, masked)
	} else {
		log.Printf("Environment variable %s is NOT SET", key)
	}
}

func main() {
	// Configure the default logger to include milliseconds in timestamps
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	// Verify critical environment variables
	log.Println("--- Verifying Environment Variables ---")
	logEnvVarPresence("BLABLADOR_API_KEY")
	logEnvVarPresence("MCP_WHODB_HOST")
	logEnvVarPresence("DATABASE_URL")
	logEnvVarPresence("OLLAMA_BASE_URL")
	log.Println("---------------------------------------")

	routes.RegisterHandlePlandex(func(router *mux.Router, path string, isStreaming bool, handler routes.PlandexHandler) *mux.Route {
		return router.HandleFunc(path, handler)
	})

	err := model.EnsureLiteLLM(2)
	if err != nil {
		panic(fmt.Sprintf("Failed to start LiteLLM proxy: %v", err))
	}
	setup.RegisterShutdownHook(func() {
		model.ShutdownLiteLLMServer()
	})

	// Initialize MCP Manager
	mcpManager := model.GetMcpManager()

	// Check if WhoDB MCP is configured
	whoDbHost := os.Getenv("MCP_WHODB_HOST")
	if whoDbHost != "" {
		log.Printf("Connecting to WhoDB MCP at %s...", whoDbHost)
		go func() {
			// Connect to WhoDB MCP with retry logic
			for {
				transport, err := mcp.NewTcpTransport(whoDbHost)
				if err != nil {
					log.Printf("Failed to connect to WhoDB MCP: %v. Retrying in 5s...", err)
					time.Sleep(5 * time.Second)
					continue
				}

				client := mcp.NewClient(transport)
				// Use a long-lived context for the client connection
				// In a real scenario we'd want graceful shutdown handling here too
				// We don't cancel this context unless we want to stop the client
				clientCtx := context.Background()

				// Start the client message loop in a goroutine
				go func() {
					if err := client.Start(clientCtx); err != nil {
						log.Printf("WhoDB MCP client stopped: %v", err)
					}
				}()

				// Initialize handshake
				// We need a short timeout for initialization to not hang forever if protocol is mismatched
				initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				initRes, err := client.Initialize(initCtx, "plandex-server", "1.0.0")
				cancel()

				if err != nil {
					log.Printf("Failed to initialize WhoDB MCP: %v", err)
					transport.Close()
					time.Sleep(5 * time.Second)
					continue
				}

				log.Printf("Connected to WhoDB MCP: %s %s", initRes.ServerInfo.Name, initRes.ServerInfo.Version)
				mcpManager.RegisterClient("whodb", client)

				// Break the retry loop once successfully connected
				return
			}
		}()
	}

	r := mux.NewRouter()
	routes.AddHealthRoutes(r)
	routes.AddApiRoutes(r)
	routes.AddProxyableApiRoutes(r)
	setup.MustLoadIp()
	setup.MustInitDb()
	setup.StartServer(r, nil, nil)
	os.Exit(0)
}
