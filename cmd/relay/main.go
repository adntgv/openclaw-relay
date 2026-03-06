package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	relayclient "github.com/adntgv/openclaw-relay/internal/client"
	"github.com/adntgv/openclaw-relay/internal/config"
	"github.com/adntgv/openclaw-relay/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "server":
		runServer(os.Args[2:])
	case "client":
		runClient(os.Args[2:])
	case "token":
		fmt.Println("token subcommand not implemented yet (Phase 5)")
		os.Exit(1)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`OpenClaw Relay - WebSocket relay server and client

Usage:
  relay <command> [options]

Commands:
  server    Start the relay server
  client    Start the relay client (not yet implemented)
  token     Manage authentication tokens (not yet implemented)
  help      Show this help message

Examples:
  relay server --port 8080 --admin-token secret
  relay client --url ws://localhost:8080/ws --token <jwt>

For more information on a command:
  relay <command> --help`)
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	host := fs.String("host", "0.0.0.0", "Host to bind to")
	port := fs.Int("port", 8080, "Port to listen on")
	adminToken := fs.String("admin-token", "", "Admin API authentication token (required)")
	jwtSecret := fs.String("jwt-secret", "", "JWT signing secret (required)")

	fs.Parse(args)

	if *adminToken == "" {
		log.Fatal("--admin-token is required")
	}
	if *jwtSecret == "" {
		log.Fatal("--jwt-secret is required")
	}

	cfg := server.Config{
		Host:       *host,
		Port:       *port,
		AdminToken: *adminToken,
		JWTSecret:  *jwtSecret,
	}

	srv := server.New(cfg)

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Printf("Relay server started on %s:%d", *host, *port)

	// Wait for shutdown signal
	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func runClient(args []string) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	url := fs.String("url", "", "Relay server WebSocket URL (ws://host:port/ws)")
	token := fs.String("token", "", "JWT token for authentication")
	clawID := fs.String("claw-id", "", "Client identifier")
	capabilities := fs.String("capabilities", "shell", "Comma-separated capabilities")
	configFile := fs.String("config", "", "Path to YAML config file")

	fs.Parse(args)

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// CLI flags override config/env
	if *url != "" {
		cfg.URL = *url
	}
	if *token != "" {
		cfg.Token = *token
	}
	if *clawID != "" {
		cfg.ClawID = *clawID
	}
	if *capabilities != "shell" || len(cfg.Capabilities) == 0 {
		cfg.Capabilities = strings.Split(*capabilities, ",")
	}

	if cfg.URL == "" || cfg.Token == "" || cfg.ClawID == "" {
		log.Fatal("--url, --token, and --claw-id are required")
	}

	client := relayclient.New(cfg)
	if err := client.Run(); err != nil {
		log.Fatalf("client error: %v", err)
	}
}
