const test = require('node:test');
const assert = require('node:assert/strict');

const { createRelayCore } = require('../src/relay-core');

test('relay core issues and validates tokens', () => {
  const core = createRelayCore();

  const issued = core.issueToken({ claw_id: 'agent-a', scopes: ['command'], ttl_seconds: 60 });
  assert.equal(typeof issued.token, 'string');

  const valid = core.validateToken(issued.token);
  assert.equal(valid.claw_id, 'agent-a');
  assert.deepEqual(valid.scopes, ['command']);

  assert.equal(core.validateToken('missing-token'), null);
});

test('relay core registers hello and lists clients', () => {
  const core = createRelayCore();
  const token = core.issueToken({ claw_id: 'agent-a' }).token;

  const client = { id: 'socket-1' };
  const helloEnvelope = {
    id: 'msg-1',
    type: 'hello',
    ts: 1710000000,
    payload: {
      claw_id: 'agent-a',
      capabilities: ['shell'],
      version: '0.1.0',
    },
  };

  core.handleEnvelope({ client, token, envelope: helloEnvelope });

  const clients = core.listClients();
  assert.equal(clients.length, 1);
  assert.equal(clients[0].claw_id, 'agent-a');
});

test('relay core creates command envelope and records ack', () => {
  const core = createRelayCore();
  const token = core.issueToken({ claw_id: 'agent-a' }).token;
  const client = { id: 'socket-1' };

  core.handleEnvelope({
    client,
    token,
    envelope: {
      id: 'h-1',
      type: 'hello',
      ts: 1710000000,
      payload: {
        claw_id: 'agent-a',
        capabilities: ['shell'],
        version: '0.1.0',
      },
    },
  });

  const command = core.dispatchCommand({
    claw_id: 'agent-a',
    cmd: 'hook.run',
    args: { name: 'sync' },
  });

  assert.equal(command.envelope.type, 'command');
  assert.equal(command.envelope.payload.cmd, 'hook.run');

  core.handleEnvelope({
    client,
    token,
    envelope: {
      id: 'a-1',
      type: 'ack',
      ts: 1710000001,
      payload: {
        command_id: command.envelope.id,
        status: 'ok',
        result: 'sync done',
      },
    },
  });

  const events = core.getAuditEvents();
  assert.equal(events.some((event) => event.type === 'ack' && event.status === 'ok'), true);
});
