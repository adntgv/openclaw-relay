# OpenClaw Relay

A lightweight, self-hosted relay that lets multiple OpenClaw clients connect to a single hub for command routing, telemetry, and remote orchestration.

> **Status:** scaffolding + docs. Implementation can start immediately.

## What this repo contains
- **Relay server** (planned): WebSocket hub with token auth, routing, and audit logs.
- **Relay client** (planned): tiny agent that connects to the hub, executes hooks, returns results.
- **OpenClaw skill** (included): CLI scaffolding to deploy, generate tokens, and manage clients.
- **Protocol spec**: message schema, auth handshake, and envelope format.

## Quickstart (once implemented)
```bash
# 1) Run relay (Docker)
docker compose up -d

# 2) Create a token
openclaw relay token create --claw-id my-laptop

# 3) Install client
openclaw relay client install --url wss://relay.example.com --token <TOKEN>

# 4) Check status
openclaw relay status
```

## Repository layout
```
openclaw-relay/
  docs/
    IMPLEMENTATION_PLAN.md
    ARCHITECTURE.md
  protocol/
    relay-protocol.md
  server/
    README.md
    .env.example
  client/
    README.md
    config.example.json
  skill/
    SKILL.md
    scripts/relay.sh
    templates/
  LICENSE
  CONTRIBUTING.md
  .gitignore
```

## MVP scope
- WebSocket relay hub
- Token auth + scope
- Client registration
- Command routing
- Result ACK + logs

## Next steps
1. Implement the server (Go or Node).
2. Implement the client (Go recommended for tiny binaries).
3. Wire OpenClaw skill commands to admin API.

## License
MIT
