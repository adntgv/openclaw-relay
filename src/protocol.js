const { randomUUID } = require('node:crypto');

const MESSAGE_TYPES = new Set(['hello', 'command', 'ack', 'event']);

function createEnvelope(type, payload, signature = null) {
  return {
    id: randomUUID(),
    type,
    ts: Math.floor(Date.now() / 1000),
    payload,
    signature,
  };
}

function validateEnvelope(envelope) {
  if (!envelope || typeof envelope !== 'object') return false;
  if (typeof envelope.id !== 'string' || envelope.id.length === 0) return false;
  if (!MESSAGE_TYPES.has(envelope.type)) return false;
  if (typeof envelope.ts !== 'number' || !Number.isFinite(envelope.ts)) return false;
  if (!Object.prototype.hasOwnProperty.call(envelope, 'payload')) return false;
  return true;
}

function parseHelloPayload(payload) {
  if (!payload || typeof payload !== 'object') throw new Error('hello payload must be object');
  if (typeof payload.claw_id !== 'string' || payload.claw_id.length === 0) {
    throw new Error('hello payload requires claw_id');
  }
  if (!Array.isArray(payload.capabilities)) {
    throw new Error('hello payload requires capabilities array');
  }
  if (typeof payload.version !== 'string' || payload.version.length === 0) {
    throw new Error('hello payload requires version');
  }
  return payload;
}

function parseAckPayload(payload) {
  if (!payload || typeof payload !== 'object') throw new Error('ack payload must be object');
  if (payload.status !== 'ok' && payload.status !== 'error') {
    throw new Error('ack payload requires status ok|error');
  }
  if (typeof payload.result !== 'string') {
    throw new Error('ack payload requires result string');
  }
  return payload;
}

module.exports = {
  MESSAGE_TYPES,
  createEnvelope,
  validateEnvelope,
  parseHelloPayload,
  parseAckPayload,
};
