# OpenClaw Relay

Self-hosted relay for OpenClaw clients using a WebSocket hub and JSON envelope protocol.

## Status
MVP implemented in Node.js:
- Relay server with `/ws` WebSocket endpoint
- Admin APIs: `/health`, `/token`, `/clients`, `/command`, `/audit`
- Relay client with hello handshake, command execution, and ack responses
- CI + tag-based release artifact workflow

## Quickstart (Local dev)

### 1) Install deps
```bash
npm install
```

### 2) Start server
```bash
ADMIN_TOKEN=dev-admin-token npm run start:server
```

### 3) Issue a client token
```bash
curl -sS -X POST http://127.0.0.1:8080/token \
  -H 'x-admin-token: dev-admin-token' \
  -H 'content-type: application/json' \
  -d '{"claw_id":"my-laptop","scopes":["command"]}'
```

### 4) Start client
```bash
RELAY_URL=ws://127.0.0.1:8080/ws \
RELAY_TOKEN=<TOKEN_FROM_STEP_3> \
CLAW_ID=my-laptop \
npm run start:client
```

### 5) Dispatch a command
```bash
curl -sS -X POST http://127.0.0.1:8080/command \
  -H 'x-admin-token: dev-admin-token' \
  -H 'content-type: application/json' \
  -d '{"claw_id":"my-laptop","cmd":"hook.run","args":{"name":"sync"}}'
```

## Quickstart (Docker Compose)
```bash
cp .env.example .env
# edit .env with ADMIN_TOKEN + RELAY_TOKEN

docker compose up -d --build
```

Generate a token and update `.env`:
```bash
curl -sS -X POST http://127.0.0.1:8080/token \
  -H "x-admin-token: <ADMIN_TOKEN>" \
  -H "content-type: application/json" \
  -d '{"claw_id":"local-client","scopes":["command"]}'
```

## Self-hosted vs Public Relay
- **Self-hosted (LAN/dev):** use `ws://127.0.0.1:8080/ws` or `ws://<server-ip>:8080/ws`.
- **Public relay (TLS):** use `wss://your-domain/ws` and protect `ADMIN_TOKEN`.

## Healthcheck
```bash
curl -sS http://127.0.0.1:8080/health
```

## Config + Env Examples
- Server env template: `server/.env.example`
- Client env template: `client/.env.example`
- Compose env template: `.env.example`
- Client JSON config: `client/config.example.json`

## Coolify
See `docs/coolify.md` for a step-by-step deployment template.

## Development
- Run tests: `npm test`
- Build release bundle: `npm run build`

## Repository layout
```
openclaw-relay/
  docs/
  protocol/
  server/
  client/
  src/
  tests/
```

## License
MIT
