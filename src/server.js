const http = require('node:http');

const { createRelayCore } = require('./relay-core');
const { createAcceptValue, WebSocketFrameParser, writeTextFrame } = require('./ws');

function json(res, statusCode, body) {
  res.statusCode = statusCode;
  res.setHeader('content-type', 'application/json');
  res.end(JSON.stringify(body));
}

function readJsonBody(req) {
  return new Promise((resolve, reject) => {
    let data = '';
    req.on('data', (chunk) => {
      data += chunk.toString('utf8');
    });
    req.on('end', () => {
      if (!data) {
        resolve({});
        return;
      }
      try {
        resolve(JSON.parse(data));
      } catch (error) {
        reject(error);
      }
    });
    req.on('error', reject);
  });
}

function adminAuthorized(req, adminToken) {
  return req.headers['x-admin-token'] === adminToken;
}

async function startRelayServer(options = {}) {
  const port = Number(options.port ?? 8080);
  const host = options.host ?? '127.0.0.1';
  const adminToken = options.adminToken ?? 'dev-admin-token';

  const core = createRelayCore();
  const sockets = new Set();
  const socketTokens = new Map();

  function closeSocket(socket) {
    try {
      socket.end();
    } catch {
      // ignore
    }
  }

  const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, `http://${req.headers.host || host}`);

    if (req.method === 'GET' && url.pathname === '/health') {
      json(res, 200, { status: 'ok' });
      return;
    }

    if (req.method === 'POST' && url.pathname === '/token') {
      if (!adminAuthorized(req, adminToken)) {
        json(res, 401, { error: 'unauthorized' });
        return;
      }

      try {
        const body = await readJsonBody(req);
        const issued = core.issueToken(body);
        json(res, 201, issued);
      } catch (error) {
        json(res, 400, { error: error.message || 'invalid request' });
      }
      return;
    }

    if (req.method === 'GET' && url.pathname === '/clients') {
      if (!adminAuthorized(req, adminToken)) {
        json(res, 401, { error: 'unauthorized' });
        return;
      }
      json(res, 200, { clients: core.listClients() });
      return;
    }

    if (req.method === 'GET' && url.pathname === '/audit') {
      if (!adminAuthorized(req, adminToken)) {
        json(res, 401, { error: 'unauthorized' });
        return;
      }
      json(res, 200, { events: core.getAuditEvents() });
      return;
    }

    if (req.method === 'POST' && url.pathname === '/command') {
      if (!adminAuthorized(req, adminToken)) {
        json(res, 401, { error: 'unauthorized' });
        return;
      }

      try {
        const body = await readJsonBody(req);
        const dispatched = core.dispatchCommand(body);
        writeTextFrame(dispatched.client, JSON.stringify(dispatched.envelope));
        json(res, 202, { accepted: true, command_id: dispatched.envelope.id });
      } catch (error) {
        const status = error.message === 'client not connected' ? 404 : 400;
        json(res, status, { error: error.message || 'invalid request' });
      }
      return;
    }

    json(res, 404, { error: 'not found' });
  });

  server.on('upgrade', (req, socket) => {
    const url = new URL(req.url, `http://${req.headers.host || host}`);
    if (url.pathname !== '/ws') {
      socket.destroy();
      return;
    }

    const token = url.searchParams.get('token');
    if (!token || !core.validateToken(token)) {
      socket.write('HTTP/1.1 401 Unauthorized\r\nConnection: close\r\n\r\n');
      socket.destroy();
      return;
    }

    const wsKey = req.headers['sec-websocket-key'];
    if (!wsKey || Array.isArray(wsKey)) {
      socket.destroy();
      return;
    }

    const acceptValue = createAcceptValue(wsKey);
    socket.write(
      [
        'HTTP/1.1 101 Switching Protocols',
        'Upgrade: websocket',
        'Connection: Upgrade',
        `Sec-WebSocket-Accept: ${acceptValue}`,
        '\r\n',
      ].join('\r\n')
    );

    sockets.add(socket);
    socketTokens.set(socket, token);

    const parser = new WebSocketFrameParser((text) => {
      try {
        const envelope = JSON.parse(text);
        core.handleEnvelope({
          client: socket,
          token,
          envelope,
        });
      } catch {
        // invalid payloads are dropped
      }
    });

    socket.on('data', (chunk) => parser.push(chunk));
    socket.on('error', () => {
      sockets.delete(socket);
      socketTokens.delete(socket);
      core.disconnectClient(socket);
    });
    socket.on('end', () => {
      sockets.delete(socket);
      socketTokens.delete(socket);
      core.disconnectClient(socket);
    });
    socket.on('close', () => {
      sockets.delete(socket);
      socketTokens.delete(socket);
      core.disconnectClient(socket);
    });
  });

  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, host, resolve);
  });

  const address = server.address();
  const actualPort = typeof address === 'object' && address ? address.port : port;

  return {
    port: actualPort,
    host,
    baseUrl: `http://${host}:${actualPort}`,
    wsUrl: `ws://${host}:${actualPort}/ws`,
    stop: () => new Promise((resolve, reject) => {
      for (const socket of sockets) {
        closeSocket(socket);
      }
      server.close((error) => {
        if (error) {
          reject(error);
          return;
        }
        resolve();
      });
    }),
  };
}

module.exports = {
  startRelayServer,
};
