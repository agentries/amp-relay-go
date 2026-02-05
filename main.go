package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openclaw/amp-relay-go/internal/protocol"
	"github.com/openclaw/amp-relay-go/internal/server"
)

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║     AMP Relay Server v5.0 (Go)         ║")
	fmt.Println("║     Jason 🍎 Labs Reference Impl       ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// Create default configuration
	config := server.DefaultConfig()
	config.ListenAddr = ":8080"

	// Create and configure server
	srv := server.NewRelayServer(config)

	// Register example routes
	srv.RegisterRoute("ping", handlePing)
	srv.RegisterRoute("echo", handleEcho)

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("Server running on %s\n", config.ListenAddr)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	<-sigChan
	fmt.Println("\nShutdown signal received...")

	// Graceful shutdown
	if err := srv.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	fmt.Println("Server stopped gracefully")
}

// handlePing responds to ping requests
func handlePing(msg *protocol.Message) (*protocol.Message, error) {
	response := protocol.NewMessage(
		protocol.MessageTypeResponse,
		"relay-server",
		msg.Source,
		"ping",
		[]byte(`{"status":"ok","message":"pong"}`),
	)
	return response, nil
}

// handleEcho echoes back the received payload
func handleEcho(msg *protocol.Message) (*protocol.Message, error) {
	response := protocol.NewMessage(
		protocol.MessageTypeResponse,
		"relay-server",
		msg.Source,
		"echo",
		msg.Payload,
	)
	return response, nil
}
