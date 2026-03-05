const { randomUUID } = require('node:crypto');

const {
  createEnvelope,
  validateEnvelope,
  parseHelloPayload,
  parseAckPayload,
} = require('./protocol');

function createRelayCore() {
  const tokens = new Map();
  const clients = new Map();
  const clientToClawId = new Map();
  const auditEvents = [];

  function addAudit(event) {
    auditEvents.push({ ts: Date.now(), ...event });
    if (auditEvents.length > 1000) {
      auditEvents.shift();
    }
  }

  function issueToken({ claw_id, scopes = [], ttl_seconds = 3600 }) {
    if (typeof claw_id !== 'string' || claw_id.length === 0) {
      throw new Error('claw_id required');
    }

    const token = randomUUID();
    const expires_at = Date.now() + Number(ttl_seconds) * 1000;
    const entry = {
      token,
      claw_id,
      scopes: Array.isArray(scopes) ? scopes : [],
      expires_at,
    };

    tokens.set(token, entry);
    addAudit({ type: 'token_issued', claw_id });
    return entry;
  }

  function validateToken(token) {
    const entry = tokens.get(token);
    if (!entry) return null;
    if (entry.expires_at <= Date.now()) {
      tokens.delete(token);
      return null;
    }
    return entry;
  }

  function listClients() {
    return [...clients.entries()].map(([claw_id, entry]) => ({
      claw_id,
      capabilities: entry.capabilities,
      version: entry.version,
      connected_at: entry.connectedAt,
    }));
  }

  function disconnectClient(client) {
    const clawId = clientToClawId.get(client);
    if (!clawId) return;

    clientToClawId.delete(client);
    const existing = clients.get(clawId);
    if (existing && existing.client === client) {
      clients.delete(clawId);
      addAudit({ type: 'disconnect', claw_id: clawId });
    }
  }

  function handleEnvelope({ client, token, envelope }) {
    const tokenEntry = validateToken(token);
    if (!tokenEntry) {
      throw new Error('invalid token');
    }

    if (!validateEnvelope(envelope)) {
      throw new Error('invalid envelope');
    }

    if (envelope.type === 'hello') {
      const hello = parseHelloPayload(envelope.payload);
      if (hello.claw_id !== tokenEntry.claw_id) {
        throw new Error('token claw_id mismatch');
      }

      const existing = clients.get(hello.claw_id);
      if (existing && existing.client !== client) {
        disconnectClient(existing.client);
      }

      clients.set(hello.claw_id, {
        client,
        capabilities: hello.capabilities,
        version: hello.version,
        connectedAt: Date.now(),
      });
      clientToClawId.set(client, hello.claw_id);
      addAudit({ type: 'hello', claw_id: hello.claw_id });
      return { action: 'registered' };
    }

    if (envelope.type === 'ack') {
      const ack = parseAckPayload(envelope.payload);
      addAudit({
        type: 'ack',
        claw_id: clientToClawId.get(client) || null,
        command_id: envelope.payload.command_id || null,
        status: ack.status,
        result: ack.result,
      });
      return { action: 'ack' };
    }

    return { action: 'ignored' };
  }

  function dispatchCommand({ claw_id, cmd, args = {} }) {
    if (typeof claw_id !== 'string' || claw_id.length === 0) {
      throw new Error('claw_id required');
    }
    if (typeof cmd !== 'string' || cmd.length === 0) {
      throw new Error('cmd required');
    }

    const target = clients.get(claw_id);
    if (!target) {
      throw new Error('client not connected');
    }

    const envelope = createEnvelope('command', { cmd, args });
    addAudit({
      type: 'command',
      claw_id,
      command_id: envelope.id,
      cmd,
    });

    return {
      client: target.client,
      envelope,
    };
  }

  function getAuditEvents() {
    return auditEvents.slice();
  }

  return {
    issueToken,
    validateToken,
    listClients,
    handleEnvelope,
    dispatchCommand,
    disconnectClient,
    getAuditEvents,
  };
}

module.exports = {
  createRelayCore,
};
