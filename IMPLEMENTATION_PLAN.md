# OpenClaw Relay — Implementation Plan

## Current State

**Repo:** `/home/aid/workspace/openclaw-relay` → `git@github.com:adntgv/openclaw-relay.git`

### What Exists (Working)
- **Server** (`src/server.js`, `src/relay-core.js`, `src/ws.js`): HTTP + raw WebSocket relay hub with token auth, admin APIs (`/token`, `/clients`, `/command`, `/audit`, `/health`), audit logging
- **Client** (`src/client.js`, `src/cli.js`): WS connection, auto-reconnect, hello handshake, command handlers (`hook.run`, `shell.exec`)
- **Protocol** (`src/protocol.js`): Envelope format (id, type, ts, payload, signature), validation, hello/ack parsing
- **Docker**: `Dockerfile.server`, `Dockerfile.client`, `docker-compose.yml`
- **CI**: GitHub Actions for container images + release binaries
- **Docs**: Architecture, implementation plan, Coolify guide
- **Tests**: 11 unit tests (protocol, client, server) — all pass individually

### What's Missing
1. **Command sandboxing** — `shell.exec` runs arbitrary commands with no allowlist
2. **Heartbeat/keepalive** — no ping/pong or heartbeat mechanism in the WS layer
3. **E2E encryption** — payloads are plaintext (TLS only at transport)
4. **Multi-tenancy** — no namespace isolation for public relay use
5. **System service files** — no systemd/launchd/NSSM configs
6. **Install script** — no one-liner installer for clients
7. **OpenClaw skill integration** — skill scaffold exists but no working CLI
8. **Integration tests** — server.test.js tests relay-core only, no HTTP/WS integration tests
9. **Test runner issue** — `node --test tests/` fails as a directory (works per-file)
10. **Reconnect backoff** — fixed delay, no exponential backoff
11. **Token revocation** — no revoke endpoint despite being mentioned in design docs

---

## Implementation Plan

### Phase 1: Harden Core (Priority: HIGH)

#### 1.1 Command Sandboxing
- **File:** `src/client.js`
- Add `allowedCommands` config (array of allowed shell commands/patterns)
- `shell.exec` checks command against allowlist before executing
- Default: deny all shell commands unless explicitly allowed
- New test: verify blocked commands return error ack

#### 1.2 WebSocket Heartbeat
- **Files:** `src/server.js`, `src/client.js`
- Server sends ping frames every 30s
- Client responds with pong
- Server disconnects clients that miss 3 consecutive pongs
- Client detects missed pings and triggers reconnect

#### 1.3 Exponential Backoff for Reconnect
- **File:** `src/client.js`
- Replace fixed `reconnect_delay_ms` with exponential backoff (1s → 2s → 4s → ... → 30s max)
- Reset on successful connection

#### 1.4 Token Revocation Endpoint
- **Files:** `src/server.js`, `src/relay-core.js`
- `DELETE /token/:token` or `POST /token/revoke` with admin auth
- Immediately disconnect any client using that token
- Test: revoke token → verify client disconnected

#### 1.5 Fix Test Runner
- **File:** `package.json`
- Change test script to glob individual files: `node --test tests/*.test.js`

### Phase 2: Production Packaging (Priority: HIGH)

#### 2.1 System Service Files
- **New files:**
  - `install/relay-server.service` (systemd)
  - `install/relay-client.service` (systemd)
  - `install/com.openclaw.relay-client.plist` (launchd)
- ExecStart, env file references, restart policies

#### 2.2 One-Liner Install Script
- **New file:** `install/install.sh`
- Detects OS/arch, downloads latest release binary from GitHub
- Copies to `/usr/local/bin/` or `~/.local/bin/`
- Optionally installs systemd service
- Usage: `curl -fsSL https://raw.githubusercontent.com/adntgv/openclaw-relay/main/install/install.sh | bash -s -- --url wss://relay.example.com --token TOKEN --claw-id MY_CLAW`

#### 2.3 OpenClaw Skill CLI
- **Files:** `skill/SKILL.md`, `skill/scripts/relay.sh`
- Implement working commands:
  - `relay status` → calls `/health` + `/clients`
  - `relay token create <claw_id>` → calls `/token`
  - `relay token revoke <token>` → calls revoke endpoint
  - `relay install <claw_id>` → outputs OS-specific install command with prefilled URL/token

### Phase 3: Testing & Quality (Priority: MEDIUM)

#### 3.1 Integration Tests
- **New file:** `tests/integration.test.js`
- Start server programmatically
- Connect real WebSocket client
- Test full flow: issue token → connect → hello → command → ack
- Test auth rejection, invalid envelopes, disconnection

#### 3.2 Load/Stress Test
- **New file:** `tests/stress.test.js`
- Connect 100 concurrent clients
- Dispatch commands to all, verify acks
- Measure latency

### Phase 4: Security Enhancements (Priority: MEDIUM)

#### 4.1 Payload Encryption (Optional)
- NaCl box encryption for command payloads
- Key exchange during hello handshake
- Opt-in via config flag

#### 4.2 Rate Limiting
- **File:** `src/server.js`
- Per-token rate limits on command dispatch
- Per-IP connection limits
- Configurable via env vars

### Phase 5: Deploy & Distribute (Priority: HIGH)

#### 5.1 Deploy Server to Coolify
- Use existing Dockerfile.server
- Domain: `relay.adntgv.com` (or similar)
- Set ADMIN_TOKEN, configure healthcheck

#### 5.2 GitHub Release
- Tag v0.1.0
- CI builds + publishes container images to GHCR
- Verify install script works against release

---

## Files to Create/Edit Summary

| Action | File | Phase |
|--------|------|-------|
| Edit | `src/client.js` — sandboxing, backoff, heartbeat | 1 |
| Edit | `src/server.js` — heartbeat, revoke endpoint, rate limiting | 1, 4 |
| Edit | `src/relay-core.js` — token revocation | 1 |
| Edit | `package.json` — fix test script | 1 |
| Create | `install/relay-server.service` | 2 |
| Create | `install/relay-client.service` | 2 |
| Create | `install/com.openclaw.relay-client.plist` | 2 |
| Create | `install/install.sh` | 2 |
| Edit | `skill/scripts/relay.sh` — working CLI | 2 |
| Edit | `skill/SKILL.md` — update commands | 2 |
| Create | `tests/integration.test.js` | 3 |
| Create | `tests/stress.test.js` | 3 |

## Implementation Order (for coding agent)

1. Fix test runner in package.json (quick win)
2. Command sandboxing in client.js + tests
3. Exponential backoff in client.js
4. Token revocation in relay-core.js + server.js + tests
5. WebSocket heartbeat in server.js + client.js
6. Integration tests
7. System service files
8. Install script
9. Skill CLI
10. Deploy to Coolify
