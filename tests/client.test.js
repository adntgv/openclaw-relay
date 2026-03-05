const test = require('node:test');
const assert = require('node:assert/strict');

const {
  createHelloEnvelope,
  processCommandEnvelope,
} = require('../src/client');

test('createHelloEnvelope follows protocol fields', () => {
  const envelope = createHelloEnvelope({
    claw_id: 'agent-a',
    capabilities: ['shell', 'fs'],
    version: '0.1.0',
  });

  assert.equal(envelope.type, 'hello');
  assert.equal(envelope.payload.claw_id, 'agent-a');
  assert.deepEqual(envelope.payload.capabilities, ['shell', 'fs']);
  assert.equal(envelope.payload.version, '0.1.0');
});

test('processCommandEnvelope returns ok ack for successful handler', async () => {
  const envelope = {
    id: 'cmd-1',
    type: 'command',
    ts: 1710000000,
    payload: {
      cmd: 'hook.run',
      args: { name: 'sync' },
    },
  };

  const handlers = {
    'hook.run': async ({ name }) => `ran:${name}`,
  };

  const ack = await processCommandEnvelope(envelope, handlers);
  assert.equal(ack.type, 'ack');
  assert.equal(ack.payload.command_id, 'cmd-1');
  assert.equal(ack.payload.status, 'ok');
  assert.equal(ack.payload.result, 'ran:sync');
});

test('processCommandEnvelope returns error ack for missing handler', async () => {
  const envelope = {
    id: 'cmd-2',
    type: 'command',
    ts: 1710000000,
    payload: {
      cmd: 'hook.unknown',
      args: {},
    },
  };

  const ack = await processCommandEnvelope(envelope, {});
  assert.equal(ack.payload.command_id, 'cmd-2');
  assert.equal(ack.payload.status, 'error');
  assert.match(ack.payload.result, /no handler/i);
});
