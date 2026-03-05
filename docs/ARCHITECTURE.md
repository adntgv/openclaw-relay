# OpenClaw Relay — Architecture

## Components
- **Relay server**: WebSocket hub + admin API
- **Relay client**: lightweight daemon per machine
- **OpenClaw skill**: CLI wrapper and deployment tooling

## Data flow
1. Client connects with token
2. Client registers capabilities
3. Server pushes command
4. Client executes locally
5. Client returns ACK + result

## Security
- Token-bound to `claw_id`
- Scope-based permissions
- Nonce + timestamp for replay protection
- Rate limits and audit logs
