# OpenClaw Relay

A WebSocket relay server and client for dispatching commands to remote nodes. Single Go binary with subcommands.

## Quick Start

```bash
# Build
make build

# Start server
./bin/relay server --port 8080 --admin-token secret --jwt-secret mysecret

# Issue a token
./bin/relay token issue --admin-token secret --claw-id mynode --scopes shell

# Start client
./bin/relay client --url ws://localhost:8080/ws --token <jwt> --claw-id mynode

# Send a command
./bin/relay send --admin-token secret --claw-id mynode --cmd shell.exec --args '{"command":"echo hello"}'

# List clients
./bin/relay clients --admin-token secret
```

## Architecture

- **Server**: HTTP server with WebSocket endpoint (`/ws`), admin API, JWT auth
- **Client**: WebSocket client with reconnect, command execution, allowlists
- **Protocol**: JSON envelopes over WebSocket (hello, command, ack, ping/pong)

## Admin API

All admin endpoints require `X-Admin-Token` header.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (no auth) |
| POST | `/token` | Issue JWT token |
| DELETE | `/token/{jti}` | Revoke token |
| GET | `/clients` | List connected clients |
| POST | `/command` | Dispatch command (sync, 30s timeout) |
| GET | `/audit` | Audit log |

## Client Config

Create `relay-client.yml`:

```yaml
url: ws://localhost:8080/ws
token: <jwt>
claw_id: mynode
capabilities:
  - shell
allowed_commands:
  - hook.run
  - shell.exec
shell:
  timeout: 30s
  allowed_binaries:
    - /usr/bin/git
    - /usr/bin/curl
hooks_dir: ./hooks
```

Environment variables (`RELAY_URL`, `RELAY_TOKEN`, `RELAY_CLAW_ID`, `RELAY_CAPABILITIES`) override config file.

## Docker

```bash
# Build and run
docker compose up -d

# Or build manually
docker build -f deployments/Dockerfile -t openclaw-relay .
docker run -p 8080:8080 openclaw-relay server --admin-token secret --jwt-secret mysecret
```

## Development

```bash
make test      # Run tests
make build     # Build binary
make check     # Format, vet, test
make coverage  # Test with coverage report
```

## License

MIT
