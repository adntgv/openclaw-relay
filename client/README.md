# Relay Client

Node.js daemon that connects to relay `/ws`, sends `hello`, handles `command`, and responds with `ack`.

## Run with env vars
```bash
RELAY_URL=ws://127.0.0.1:8080/ws \
RELAY_TOKEN=<TOKEN> \
CLAW_ID=my-laptop \
CAPABILITIES=shell,fs \
npm run start:client
```

## Run with config file
```bash
RELAY_CONFIG=client/config.example.json npm run start:client
```

## Supported commands
- `hook.run` -> returns `hook:<name>`
- `shell.exec` -> executes `args.command` locally (use with caution)

## Config
See `config.example.json`.
