package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/openclaw/amp-relay-go/internal/auth"
	"github.com/openclaw/amp-relay-go/internal/config"
	"github.com/openclaw/amp-relay-go/internal/protocol"
	"github.com/openclaw/amp-relay-go/internal/server"
	"github.com/openclaw/amp-relay-go/internal/storage"
)

func main() {
	log.Println("AMP Relay Server v5.0 (Go) — Jason Labs Reference Impl")

	// Load configuration (file path from AMP_CONFIG_PATH env, or defaults)
	configPath := os.Getenv("AMP_CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create authenticator based on config
	authIntegration := auth.NewIntegrationPoint(cfg.Security.EnableAuth)

	// Build server config from loaded config
	srvConfig := &server.Config{
		ListenAddr:         cfg.Server.Address,
		AllowedOrigins:     cfg.Security.AllowedOrigins,
		Authenticator:      authIntegration.Authenticator,
		Storage:            storage.NewMemoryStore(),
		DefaultTTL:         cfg.Storage.DefaultTTL,
		MaxPayloadSize:     cfg.Server.MaxPayloadSize,
		RateLimitPerMinute: cfg.Security.RateLimitPerMinute,
	}

	// Create and configure server
	srv := server.NewRelayServer(srvConfig)

	// Register example routes
	srv.RegisterRoute("ping", handlePing)
	srv.RegisterRoute("echo", handleEcho)

	// Start server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Server running on %s", srvConfig.ListenAddr)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received...")

	if err := srv.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// handlePing responds to ping requests
func handlePing(msg *protocol.Message) (*protocol.Message, error) {
	response := protocol.NewMessage(
		protocol.MessageTypeResponse,
		"relay-server",
		msg.From,
		map[string]string{"status": "ok", "message": "pong"},
	)
	return response, nil
}

// handleEcho echoes back the received payload
func handleEcho(msg *protocol.Message) (*protocol.Message, error) {
	response := protocol.NewMessage(
		protocol.MessageTypeResponse,
		"relay-server",
		msg.From,
		msg.Body,
	)
	return response, nil
}
