# Relay Server

Node.js relay server that exposes HTTP admin APIs and a WebSocket hub.

## Run
```bash
ADMIN_TOKEN=dev-admin-token PORT=8080 HOST=0.0.0.0 npm run start:server
```

## Environment
See `.env.example`.

## HTTP Endpoints
- `GET /health` -> `{ "status": "ok" }`
- `POST /token` (admin)
- `GET /clients` (admin)
- `POST /command` (admin)
- `GET /audit` (admin)

Admin auth header:
- `x-admin-token: <ADMIN_TOKEN>`

## Token payload
```json
{
  "claw_id": "my-laptop",
  "scopes": ["command"],
  "ttl_seconds": 3600
}
```
