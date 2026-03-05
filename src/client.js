const { exec } = require('node:child_process');
const { promisify } = require('node:util');

const execAsync = promisify(exec);

const { createEnvelope, validateEnvelope } = require('./protocol');

function createHelloEnvelope({ claw_id, capabilities = [], version = '0.1.0' }) {
  return createEnvelope('hello', {
    claw_id,
    capabilities,
    version,
  });
}

async function processCommandEnvelope(envelope, handlers) {
  if (!validateEnvelope(envelope) || envelope.type !== 'command') {
    throw new Error('invalid command envelope');
  }

  const cmdName = envelope.payload?.cmd;
  const args = envelope.payload?.args ?? {};

  try {
    const handler = handlers[cmdName];
    if (typeof handler !== 'function') {
      throw new Error(`no handler for command: ${cmdName}`);
    }

    const output = await handler(args);
    return createEnvelope('ack', {
      command_id: envelope.id,
      status: 'ok',
      result: String(output ?? ''),
    });
  } catch (error) {
    return createEnvelope('ack', {
      command_id: envelope.id,
      status: 'error',
      result: error.message || 'command failed',
    });
  }
}

function defaultHandlers() {
  return {
    'hook.run': async (args) => {
      const name = args?.name || 'unknown';
      return `hook:${name}`;
    },
    'shell.exec': async (args) => {
      if (!args || typeof args.command !== 'string' || args.command.length === 0) {
        throw new Error('shell.exec requires args.command');
      }
      const { stdout, stderr } = await execAsync(args.command, {
        timeout: Number(args.timeout_ms ?? 10000),
        maxBuffer: 1024 * 1024,
      });
      return [stdout, stderr].filter(Boolean).join('\n').trim();
    },
  };
}

function createRelayClient(config) {
  const {
    url,
    token,
    claw_id,
    capabilities = ['shell'],
    version = '0.1.0',
    reconnect_delay_ms = 1000,
    handlers = defaultHandlers(),
  } = config;

  let ws = null;
  let reconnectTimer = null;
  let stopped = false;

  function connect() {
    if (stopped) return;

    ws = new WebSocket(`${url}?token=${encodeURIComponent(token)}`);

    ws.addEventListener('open', () => {
      const hello = createHelloEnvelope({ claw_id, capabilities, version });
      ws.send(JSON.stringify(hello));
    });

    ws.addEventListener('message', async (event) => {
      try {
        const envelope = JSON.parse(String(event.data));
        if (envelope.type !== 'command') return;
        const ack = await processCommandEnvelope(envelope, handlers);
        ws.send(JSON.stringify(ack));
      } catch {
        // drop malformed messages
      }
    });

    ws.addEventListener('close', () => {
      if (stopped) return;
      reconnectTimer = setTimeout(connect, reconnect_delay_ms);
    });

    ws.addEventListener('error', () => {
      if (!ws || ws.readyState === WebSocket.CLOSED) return;
      ws.close();
    });
  }

  return {
    start() {
      stopped = false;
      connect();
    },
    stop() {
      stopped = true;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    },
  };
}

module.exports = {
  createHelloEnvelope,
  processCommandEnvelope,
  createRelayClient,
  defaultHandlers,
};
