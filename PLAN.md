# OpenClaw Relay Production Plan

## Goal
Harden the relay repository for real-world usage with production-ready packaging, deployment templates, and clear operator docs.

## Constraints
- Do all work in this repository (`/home/aid/workspace/openclaw-relay`).
- Do not use `~/.openclaw`.
- Use red/green TDD for new behavior covered by tests.
- Commit at every milestone and attempt to push after each milestone.

## Milestone 1: Refresh Plan + Repo Audit
### Deliverables
- Update this plan to the production checklist.
- Scan repo layout for missing ops artifacts.

### Commit
- `chore: refresh production plan`

## Milestone 2: Dockerfiles (Server + Client)
### Deliverables
- Dockerfile for relay server with minimal runtime image.
- Dockerfile for relay client daemon.
- Document build/run commands locally.

### Commit
- `feat(docker): add server and client Dockerfiles`

## Milestone 3: docker-compose Stack
### Deliverables
- Compose file for local/dev stack (server + client + env).
- Default env file template and notes for overriding secrets.
- Healthcheck wiring for the server service.

### Commit
- `feat(compose): add local stack with env templates`

## Milestone 4: Coolify Deployment Template
### Deliverables
- Coolify-ready template or docs (server + client options).
- Minimal instructions for configuring env vars + domains.

### Commit
- `docs(coolify): add deployment template and steps`

## Milestone 5: Docs + Env Examples
### Deliverables
- README quickstart for self-host/public relay modes.
- Example env files for server + client.
- Basic healthcheck instructions.

### Commit
- `docs: expand quickstart + env examples`

## Milestone 6: CI Workflow (Container Images)
### Deliverables
- GitHub Actions workflow to build/publish container images (if feasible).
- Validate YAML + notes for registry auth.

### Commit
- `ci: add container build workflow`

## Finalization
1. Verify any run/deploy commands referenced in docs.
2. Generate summary report in `/home/aid/.openclaw/workspace/relay-next-steps.html`.
3. Run completion command:
   - `openclaw system event --text "Done: relay setup continued with docker+coolify+docs" --mode now`
