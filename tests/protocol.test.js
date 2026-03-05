const test = require('node:test');
const assert = require('node:assert/strict');

const {
  createEnvelope,
  validateEnvelope,
  parseHelloPayload,
  parseAckPayload,
} = require('../src/protocol');

test('createEnvelope emits required fields', () => {
  const envelope = createEnvelope('hello', { claw_id: 'agent-a' });

  assert.equal(typeof envelope.id, 'string');
  assert.equal(envelope.type, 'hello');
  assert.equal(typeof envelope.ts, 'number');
  assert.deepEqual(envelope.payload, { claw_id: 'agent-a' });
  assert.equal(envelope.signature, null);
});

test('validateEnvelope accepts known message types', () => {
  const envelope = createEnvelope('command', { cmd: 'hook.run', args: { name: 'sync' } });

  assert.equal(validateEnvelope(envelope), true);
});

test('validateEnvelope rejects malformed envelope', () => {
  const bad = {
    id: '',
    type: 'weird',
    ts: 'nope',
    payload: null,
  };

  assert.equal(validateEnvelope(bad), false);
});

test('parseHelloPayload validates required fields', () => {
  const payload = {
    claw_id: 'my-laptop',
    capabilities: ['shell', 'fs'],
    version: '0.1.0',
  };

  const parsed = parseHelloPayload(payload);
  assert.deepEqual(parsed, payload);

  assert.throws(
    () => parseHelloPayload({ capabilities: [], version: '0.1.0' }),
    /claw_id/
  );
});

test('parseAckPayload validates ack status', () => {
  const ok = parseAckPayload({ status: 'ok', result: 'ran hook' });
  assert.deepEqual(ok, { status: 'ok', result: 'ran hook' });

  assert.throws(() => parseAckPayload({ status: 'bad', result: '' }), /status/);
});
