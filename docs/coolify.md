# Coolify Deployment (Relay Server + Client)

## Relay Server (Required)
1. Create a new **Application** in Coolify.
2. Source: GitHub repo `openclaw-relay`.
3. Build pack: **Dockerfile** → `Dockerfile.server`.
4. Expose port **8080**.
5. Environment variables:
   - `ADMIN_TOKEN` (required)
   - `HOST=0.0.0.0`
   - `PORT=8080`
6. Healthcheck path: `/health`.
7. Deploy.

### Issue a client token
```bash
curl -sS -X POST https://YOUR-RELAY-DOMAIN/token \
  -H "x-admin-token: <ADMIN_TOKEN>" \
  -H "content-type: application/json" \
  -d '{"claw_id":"my-client","scopes":["command"]}'
```

## Relay Client (Optional)
1. Create a new **Application** in Coolify.
2. Source: same repo.
3. Build pack: **Dockerfile** → `Dockerfile.client`.
4. No ports required.
5. Environment variables:
   - `RELAY_URL=ws://YOUR-RELAY-DOMAIN/ws` (use `wss://` for TLS)
   - `RELAY_TOKEN` (from `/token` endpoint)
   - `CLAW_ID` (unique client id)
   - `CAPABILITIES=shell`
   - `CLIENT_VERSION=0.1.0`
6. Deploy.

## Notes
- Run multiple clients by duplicating the client app with unique `CLAW_ID` and `RELAY_TOKEN` values.
- If you already host a relay server elsewhere, only deploy the client.
