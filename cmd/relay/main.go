package main

import (
	"context"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
		runToken(os.Args[2:])
	case "clients":
		runClients(os.Args[2:])
	case "send":
		runSend(os.Args[2:])
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
  client    Start the relay client
  token     Manage authentication tokens (issue/revoke)
  clients   List connected clients
  send      Send a command to a client
  help      Show this help message

Examples:
  relay server --port 8080 --admin-token secret --jwt-secret mysecret
  relay client --url ws://localhost:8080/ws --token <jwt> --claw-id mynode
  relay token issue --admin-token secret --claw-id mynode --scopes shell
  relay token revoke --admin-token secret --jti <token-jti>
  relay clients --admin-token secret
  relay send --admin-token secret --claw-id mynode --cmd shell.exec --args '{"command":"echo hi"}'`)
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	host := fs.String("host", "0.0.0.0", "Host to bind to")
	port := fs.Int("port", 8080, "Port to listen on")
	adminToken := fs.String("admin-token", "", "Admin API authentication token (required)")
	jwtSecret := fs.String("jwt-secret", "", "JWT signing secret (required)")
	tokenStore := fs.String("token-store", "", "Path to token store JSON file (optional, for persistence)")

	fs.Parse(args)

	// Env var fallbacks
	if *adminToken == "" {
		*adminToken = os.Getenv("RELAY_ADMIN_TOKEN")
	}
	if *jwtSecret == "" {
		*jwtSecret = os.Getenv("RELAY_JWT_SECRET")
	}
	if envHost := os.Getenv("RELAY_HOST"); envHost != "" && *host == "0.0.0.0" {
		*host = envHost
	}
	if envPort := os.Getenv("RELAY_PORT"); envPort != "" && *port == 8080 {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}
	if envStore := os.Getenv("RELAY_TOKEN_STORE"); envStore != "" && *tokenStore == "" {
		*tokenStore = envStore
	}

	if *adminToken == "" {
		slog.Error("--admin-token or RELAY_ADMIN_TOKEN is required")
		os.Exit(1)
	}
	if *jwtSecret == "" {
		slog.Error("--jwt-secret or RELAY_JWT_SECRET is required")
		os.Exit(1)
	}

	cfg := server.Config{
		Host:           *host,
		Port:           *port,
		AdminToken:     *adminToken,
		JWTSecret:      *jwtSecret,
		TokenStorePath: *tokenStore,
	}

	srv := server.New(cfg)

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("relay server started", "host", *host, "port", *port)

	// Wait for shutdown signal
	<-stop
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
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
		slog.Error("config error", "error", err)
		os.Exit(1)
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
		slog.Error("--url, --token, and --claw-id are required")
		os.Exit(1)
	}

	client := relayclient.New(cfg)
	if err := client.Run(); err != nil {
		slog.Error("client error", "error", err)
		os.Exit(1)
	}
}

func runToken(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: relay token <issue|revoke> [options]")
		os.Exit(1)
	}

	switch args[0] {
	case "issue":
		fs := flag.NewFlagSet("token issue", flag.ExitOnError)
		serverURL := fs.String("server", "http://localhost:8080", "Server URL")
		adminToken := fs.String("admin-token", "", "Admin token")
		clawID := fs.String("claw-id", "", "Client ID for token")
		scopes := fs.String("scopes", "shell", "Comma-separated scopes")
		ttl := fs.Int("ttl", 24, "Token TTL in hours")
		fs.Parse(args[1:])

		body, _ := json.Marshal(map[string]interface{}{
			"claw_id":   *clawID,
			"scopes":    strings.Split(*scopes, ","),
			"ttl_hours": *ttl,
		})
		resp := adminRequest("POST", *serverURL+"/token", *adminToken, body)
		fmt.Println(resp)

	case "revoke":
		fs := flag.NewFlagSet("token revoke", flag.ExitOnError)
		serverURL := fs.String("server", "http://localhost:8080", "Server URL")
		adminToken := fs.String("admin-token", "", "Admin token")
		jti := fs.String("jti", "", "Token JTI to revoke")
		fs.Parse(args[1:])

		adminRequest("DELETE", *serverURL+"/token/"+*jti, *adminToken, nil)
		fmt.Println("Token revoked")

	default:
		fmt.Fprintf(os.Stderr, "Unknown token command: %s\n", args[0])
		os.Exit(1)
	}
}

func runClients(args []string) {
	fs := flag.NewFlagSet("clients", flag.ExitOnError)
	serverURL := fs.String("server", "http://localhost:8080", "Server URL")
	adminToken := fs.String("admin-token", "", "Admin token")
	fs.Parse(args)

	resp := adminRequest("GET", *serverURL+"/clients", *adminToken, nil)
	fmt.Println(resp)
}

func runSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	serverURL := fs.String("server", "http://localhost:8080", "Server URL")
	adminToken := fs.String("admin-token", "", "Admin token")
	clawID := fs.String("claw-id", "", "Target client")
	cmd := fs.String("cmd", "", "Command to send")
	argsJSON := fs.String("args", "{}", "JSON args")
	fs.Parse(args)

	var cmdArgs map[string]interface{}
	json.Unmarshal([]byte(*argsJSON), &cmdArgs)

	body, _ := json.Marshal(map[string]interface{}{
		"claw_id": *clawID,
		"cmd":     *cmd,
		"args":    cmdArgs,
	})
	resp := adminRequest("POST", *serverURL+"/command", *adminToken, body)
	fmt.Println(resp)
}

func adminRequest(method, url, adminToken string, body []byte) string {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		slog.Error("request error", "error", err)
		os.Exit(1)
	}
	req.Header.Set("X-Admin-Token", adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("request failed", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.Error("server error", "status", resp.StatusCode, "body", string(data))
		os.Exit(1)
	}
	return string(data)
}
