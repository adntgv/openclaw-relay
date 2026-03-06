# OpenClaw Relay — Go Rewrite Plan

## Overview

Rewrite the existing Node.js relay (server + client) as a **single Go binary** with subcommands. The existing protocol and API surface are preserved; the implementation gains single-binary deployment, native concurrency, and a path to multi-tenancy.

---

## 1. Directory Structure

```
openclaw-relay/
├── cmd/
│   └── relay/
│       └── main.go              # Entry point: `relay server`, `relay client`, `relay token`, etc.
├── internal/
│   ├── server/
│   │   ├── server.go            # HTTP server, WS upgrade, routing
│   │   ├── hub.go               # WebSocket hub (register/unregister/broadcast)
│   │   ├── conn.go              # Per-connection reader/writer goroutines
│   │   ├── admin.go             # Admin API handlers (/token, /clients, /command, /audit)
│   │   ├── auth.go              # Token validation, scope checking
│   │   └── server_test.go
│   ├── client/
│   │   ├── client.go            # WS client, reconnect loop, heartbeat
│   │   ├── handler.go           # Command dispatch, hook execution, allowlist
│   │   └── client_test.go
│   ├── protocol/
│   │   ├── envelope.go          # Envelope struct, marshal/unmarshal, validation
│   │   ├── types.go             # Message type constants, payload structs
│   │   └── envelope_test.go
│   ├── token/
│   │   ├── store.go             # Token CRUD (in-memory + file-backed)
│   │   ├── jwt.go               # JWT generation/validation (HMAC-SHA256)
│   │   └── store_test.go
│   └── config/
│       ├── config.go            # Env + flag + file config loading
│       └── config_test.go
├── protocol/
│   └── relay-protocol.md        # (existing) Protocol spec
├── docs/
│   ├── ARCHITECTURE.md          # (existing, update)
│   ├── coolify.md               # (existing, update)
│   └── IMPLEMENTATION_PLAN.md   # (existing, superseded by this file)
├── deployments/
│   ├── Dockerfile               # Multi-stage: build + scratch/alpine
│   ├── docker-compose.yml       # Local dev stack
│   └── .env.example
├── Makefile                     # build, test, lint, docker
├── go.mod
├── go.sum
├── README.md
└── PLAN.md                      # This file
```

**Key decisions:**
- `internal/` prevents external import — this is an application, not a library.
- Single binary via `cmd/relay/main.go` with cobra-style subcommands (use `cobra` or stdlib `flag` + manual dispatch).
- Old Node.js code stays in `src/` until Go is validated, then removed.

---

## 2. Protocol Specification

### 2.1 Transport
- **WebSocket** over HTTP/1.1 upgrade at `/ws`
- JSON text frames (no binary)
- Ping/pong at WS level for keepalive (server sends ping every 30s, client must pong within 10s)

### 2.2 Envelope Format
```json
{
  "id": "uuid-v4",
  "type": "hello | command | ack | event | ping | pong",
  "ts": 1710000000,
  "payload": { ... },
  "signature": null
}
```

All fields required. `signature` reserved for future HMAC signing (nullable for now).

### 2.3 Message Types

| Type | Direction | Purpose |
|------|-----------|---------|
| `hello` | Client→Server | Auth + capability registration |
| `command` | Server→Client | Dispatch a command to execute |
| `ack` | Client→Server | Command result (ok/error) |
| `event` | Either | Async notifications (future) |
| `ping` | Server→Client | App-level heartbeat |
| `pong` | Client→Server | Heartbeat response |

### 2.4 Authentication Flow
1. Client connects to `/ws`
2. Client sends `hello` with JWT token in payload:
   ```json
   {
     "type": "hello",
     "payload": {
       "token": "eyJ...",
       "claw_id": "my-laptop",
       "capabilities": ["shell", "browser"],
       "version": "1.0.0"
     }
   }
   ```
3. Server validates JWT (checks `claw_id` claim matches, token not revoked, scopes valid)
4. Server responds with `ack` (status ok) or closes connection
5. Client is now registered in the hub

### 2.5 Command Dispatch
1. Admin POSTs to `/command` with `claw_id`, `cmd`, `args`
2. Server wraps in `command` envelope, sends to matching client
3. Client executes (subject to allowlist), returns `ack`
4. Server returns ack payload to the HTTP caller (sync, with timeout)

---

## 3. Admin API

All admin endpoints require `X-Admin-Token` header.

| Method | Path | Body | Response | Description |
|--------|------|------|----------|-------------|
| GET | `/health` | — | `{"status":"ok","clients":N,"uptime":S}` | Health check (no auth) |
| POST | `/token` | `{"claw_id","scopes":[],"ttl_hours":N}` | `{"token":"eyJ..."}` | Issue JWT |
| DELETE | `/token/{jti}` | — | 204 | Revoke token |
| GET | `/clients` | — | `[{"claw_id","caps":[],"connected_at"}]` | List connected clients |
| POST | `/command` | `{"claw_id","cmd","args":{}}` | `{"status","result"}` | Dispatch command (sync, 30s timeout) |
| GET | `/audit` | `?limit=N&claw_id=X` | `[{"ts","action","claw_id","detail"}]` | Audit log |

---

## 4. Client Features

### 4.1 Connection Management
- Exponential backoff reconnect: 1s → 2s → 4s → ... → 60s max, reset on successful hello
- Graceful shutdown on SIGINT/SIGTERM (send close frame, wait 5s)

### 4.2 Command Allowlist
Config file or env-based allowlist of permitted commands:
```yaml
# relay-client.yml
allowed_commands:
  - hook.run
  - shell.exec
  - fs.read
shell:
  timeout: 30s
  allowed_binaries:
    - /usr/bin/git
    - /usr/bin/curl
```

### 4.3 Heartbeat
- Respond to server `ping` with `pong`
- If no message received in 90s, assume disconnected → reconnect

### 4.4 Hook Execution
- `hook.run` → execute named script from hooks directory
- `shell.exec` → run command (allowlisted binaries only, with timeout)
- All output captured, returned in ack `result` field (truncated to 64KB)

---

## 5. Implementation Phases

### Phase 1: Scaffolding & Core Server (Sonnet Task 1)
**Goal:** Buildable Go project with working WebSocket hub and health endpoint.

1. Initialize `go.mod` (module `github.com/adntgv/openclaw-relay`)
2. Create directory structure per §1
3. Implement `internal/protocol/` — Envelope struct, marshal/unmarshal, validation, tests
4. Implement `internal/server/hub.go` — goroutine-safe client registry
5. Implement `internal/server/conn.go` — read/write pumps per connection
6. Implement `internal/server/server.go` — HTTP server, `/ws` upgrade (use `gorilla/websocket`), `/health`
7. Implement `cmd/relay/main.go` — `relay server` subcommand with flags (port, host, admin-token)
8. Tests: hub register/unregister, envelope round-trip, health endpoint
9. `Makefile` with `build`, `test`, `lint` targets
10. Commit: `feat: go scaffolding with core ws hub and health endpoint`

**Acceptance:** `go test ./...` passes, `relay server` starts and accepts WS connections, `/health` returns JSON.

### Phase 2: Auth & Admin API (Sonnet Task 2)
**Goal:** Token issuance, JWT validation on hello, admin endpoints.

1. Implement `internal/token/` — JWT creation (HS256, claims: jti, claw_id, scopes, exp), in-memory store with revocation set
2. Implement `internal/server/auth.go` — validate hello token, check claw_id match
3. Implement `internal/server/admin.go` — POST /token, DELETE /token/{jti}, GET /clients, POST /command (stub dispatch), GET /audit
4. Wire auth into hello handshake (reject bad tokens, close connection)
5. Tests: token issue/validate/revoke, admin endpoints, auth rejection
6. Commit: `feat: jwt auth and admin api`

**Acceptance:** Can issue token via API, connect client with it, get rejected with bad token.

### Phase 3: Command Dispatch (Sonnet Task 3)
**Goal:** Full command round-trip: admin → server → client → ack → admin response.

1. Implement POST /command → find client in hub → send command envelope → wait for ack (channel with 30s timeout)
2. Implement `internal/server/conn.go` ack routing (map command_id → response channel)
3. Audit logging for all commands
4. Tests: full dispatch flow, timeout handling, client-not-found
5. Commit: `feat: command dispatch with sync ack`

### Phase 4: Client (Sonnet Task 4)
**Goal:** Working relay client with reconnect and command handling.

1. Implement `internal/client/client.go` — connect, hello handshake, read loop, reconnect with backoff
2. Implement `internal/client/handler.go` — command dispatch table, `hook.run` (execute script), `shell.exec` (allowlisted), output capture
3. Config loading from env + yaml file
4. `relay client` subcommand in main.go
5. Tests: handler dispatch, allowlist enforcement, reconnect logic
6. Commit: `feat: relay client with reconnect and command execution`

**Acceptance:** Full end-to-end: start server → issue token → start client → dispatch command → get result.

### Phase 5: CLI & Polish (Sonnet Task 5)
**Goal:** Management CLI, Docker, docs.

1. `relay token issue --claw-id X --scopes shell` (calls admin API)
2. `relay token revoke --jti X`
3. `relay clients` (list connected)
4. `relay send --claw-id X --cmd hook.run --args '{"name":"sync"}'`
5. Multi-stage Dockerfile (build in golang:1.22, run in scratch+ca-certs)
6. docker-compose.yml for local dev
7. Update README.md, ARCHITECTURE.md
8. Commit: `feat: cli, docker, docs`

### Phase 6: Hardening & Multi-tenancy Prep (Sonnet Task 6)
**Goal:** Production readiness.

1. Rate limiting (per-IP on WS connect, per-token on commands)
2. Structured logging (slog)
3. Metrics endpoint (`/metrics` — Prometheus format, optional)
4. Token persistence (SQLite or bolt for single-node, Postgres-ready interface)
5. Graceful shutdown
6. TLS termination docs (rely on reverse proxy / Coolify / Traefik)
7. Multi-tenancy design: namespace tokens by `tenant_id` claim (future)
8. Commit: `feat: rate limiting, logging, graceful shutdown`

---

## 6. Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/gorilla/websocket` | WebSocket server + client |
| `github.com/golang-jwt/jwt/v5` | JWT tokens |
| `github.com/spf13/cobra` | CLI subcommands |
| `gopkg.in/yaml.v3` | Client config file |
| `golang.org/x/time/rate` | Rate limiting |

No frameworks. Standard `net/http` for the server.

---

## 7. Config

### Server (env vars)
| Var | Default | Description |
|-----|---------|-------------|
| `RELAY_PORT` | `8080` | Listen port |
| `RELAY_HOST` | `0.0.0.0` | Bind address |
| `RELAY_ADMIN_TOKEN` | required | Admin API auth |
| `RELAY_JWT_SECRET` | required | HMAC key for JWTs |
| `RELAY_LOG_LEVEL` | `info` | Log level |

### Client (env vars)
| Var | Default | Description |
|-----|---------|-------------|
| `RELAY_URL` | required | `ws://host:port/ws` |
| `RELAY_TOKEN` | required | JWT from server |
| `RELAY_CLAW_ID` | required | This client's ID |
| `RELAY_CAPABILITIES` | `shell` | Comma-separated |
| `RELAY_CONFIG` | `relay-client.yml` | Config file path |

---

## 8. Spawning Phase 1

To start implementation, spawn a Sonnet coding agent with:

```
Role: Coder (Sonnet)
Working directory: ~/workspace/openclaw-relay
Task: Implement Phase 1 from PLAN.md — Scaffolding & Core Server.

Instructions:
1. Read PLAN.md §1 (Directory Structure) and §5 Phase 1 for full spec.
2. Remove old Node.js code (src/, tests/, package.json, package-lock.json, Dockerfile.server, Dockerfile.client) — we're starting fresh in Go.
3. Initialize go.mod as github.com/adntgv/openclaw-relay
4. Implement in order: protocol → hub → conn → server → main.go → Makefile
5. Use red/green TDD: write tests first, confirm fail, then implement.
6. Run `go test ./...` and `go build ./cmd/relay/` before committing.
7. Commit with message: "feat: go scaffolding with core ws hub and health endpoint"
8. Do NOT push (no git remote configured yet).

Acceptance criteria:
- `go test ./...` passes
- `./relay server --port 8080 --admin-token test` starts and serves /health
- WebSocket connections accepted at /ws
- Envelope marshal/unmarshal works with validation
```
