# OpenClaw Relay — Implementation Plan

## 1) Requirements
- Self-hosted, minimal dependencies
- Multiple client connections
- Token-based auth and per-client scope
- Routing: server → client and client → server
- Audit log and health endpoints

## 2) Protocol (MVP)
- **Transport:** WebSocket
- **Auth:** token (HMAC/JWT) with TTL
- **Handshake:** client sends `hello` with `claw_id`, `capabilities`
- **Message envelope:** `{ id, type, ts, payload, signature }`
- **ACK:** `{ id, type: "ack", status, result }`

## 3) Server
- **Language:** Go (preferred) or Node
- **Endpoints:**
  - `GET /health`
  - `POST /token` (admin)
  - `GET /clients`
- **WS:** `/ws` for client connections
- **Storage:** in-memory + optional SQLite for logs

## 4) Client
- Persistent WS connection
- Auto-reconnect with backoff
- Hook executor (local commands or OpenClaw hooks)
- Sends ACK with output / status

## 5) OpenClaw Skill
- `relay init` — deploy relay
- `relay token create` — create client token
- `relay client install` — show OS-specific install steps
- `relay status` — list connected clients

## 6) Milestones
1. Server WS + auth + routing
2. Client connection + execute + ACK
3. OpenClaw skill wiring
4. Packaging + docs

## 7) Open questions
- Go vs Node implementation
- Public relay tier or self-host only
- Coolify template details
