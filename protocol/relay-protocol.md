# OpenClaw Relay Protocol (Draft)

## Envelope
```json
{
  "id": "uuid",
  "type": "hello|command|ack|event",
  "ts": 1710000000,
  "payload": {},
  "signature": "optional"
}
```

## Hello
```json
{
  "type": "hello",
  "payload": {
    "claw_id": "my-laptop",
    "capabilities": ["shell","browser","fs"],
    "version": "0.1.0"
  }
}
```

## Command
```json
{
  "type": "command",
  "payload": {
    "cmd": "hook.run",
    "args": {"name": "sync"}
  }
}
```

## ACK
```json
{
  "type": "ack",
  "payload": {
    "status": "ok|error",
    "result": "..."
  }
}
```
