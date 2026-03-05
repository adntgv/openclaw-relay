const { startRelayServer } = require('./server');

const port = Number(process.env.PORT || 8080);
const host = process.env.HOST || '0.0.0.0';
const adminToken = process.env.ADMIN_TOKEN || 'dev-admin-token';

startRelayServer({ port, host, adminToken })
  .then((relay) => {
    console.log(`relay server listening on ${relay.baseUrl}`);
  })
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  });
