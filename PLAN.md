# OpenClaw Relay MVP Plan

## Goal
Implement an MVP relay server and client that follow the existing protocol docs, with tests, updated README docs, and GitHub Actions release build workflow.

## Constraints
- Do all work in this repository (`/home/aid/workspace/openclaw-relay`).
- Do not use `~/.openclaw`.
- Use red/green TDD for new behavior covered by tests.
- Commit at every milestone and attempt to push after each milestone.

## Milestone 1: Bootstrap + Shared Protocol Types
### Deliverables
- Node workspace bootstrap with server/client packages.
- Shared protocol envelope/message validation helpers.
- Initial tests for protocol parsing and envelope validation.

### TDD cycle
1. Write failing tests for envelope/message validation.
2. Run tests and confirm failures.
3. Implement minimal protocol module.
4. Re-run tests until green.

### Commit
- `chore: bootstrap relay project with shared protocol module`

## Milestone 2: MVP Relay Server
### Deliverables
- WebSocket server at `/ws`.
- HTTP endpoints: `GET /health`, `POST /token`, `GET /clients`, `POST /command`.
- In-memory token store and connected client registry.
- Routing server->client `command` and client->server `ack`.
- Basic audit log capture in memory.
- Server tests for auth, registration, routing, and admin APIs.

### TDD cycle
1. Write failing integration tests for auth/registration/routing.
2. Run tests to confirm red.
3. Implement minimal server behavior.
4. Re-run tests to green, then refactor.

### Commit
- `feat(server): implement websocket relay hub and admin APIs`

## Milestone 3: MVP Relay Client
### Deliverables
- Client daemon with persistent WS connection and reconnect backoff.
- Sends `hello` on connect with `claw_id`, `capabilities`, `version`.
- Receives `command` and executes local hook handlers.
- Sends `ack` with `ok|error` and result text.
- Client tests for handshake, command execution, and ack behavior.

### TDD cycle
1. Write failing client behavior tests.
2. Confirm tests fail.
3. Implement minimal client runtime and hook executor.
4. Re-run tests to green.

### Commit
- `feat(client): implement relay client handshake and command ack flow`

## Milestone 4: Docs + Release Workflow
### Deliverables
- Update root/server/client READMEs for setup and usage.
- Add GitHub Actions workflow to build release artifacts on tags and run CI on pushes.
- Verify end-to-end command routing with test coverage.

### Verification
- Run full test suite.
- Run build command.
- Confirm workflow YAML syntax and release artifact steps.

### Commit
- `docs(ci): update readmes and add release build workflow`

## Finalization
1. Run full verification commands.
2. Report milestone commits and push status.
3. Run completion command:
   - `openclaw system event --text "Done: relay repo MVP implemented + commits pushed" --mode now`
