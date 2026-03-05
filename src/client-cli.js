const fs = require('node:fs');

const { createRelayClient } = require('./client');

function normalizeConfig(input) {
  return {
    ...input,
    url: input.url || input.relay_url,
  };
}

function readConfig() {
  const configPath = process.env.RELAY_CONFIG;
  if (!configPath) {
    return normalizeConfig({
      url: process.env.RELAY_URL,
      token: process.env.RELAY_TOKEN,
      claw_id: process.env.CLAW_ID,
      capabilities: (process.env.CAPABILITIES || 'shell')
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean),
      version: process.env.CLIENT_VERSION || '0.1.0',
    });
  }

  const raw = fs.readFileSync(configPath, 'utf8');
  return normalizeConfig(JSON.parse(raw));
}

const config = readConfig();
if (!config.url || !config.token || !config.claw_id) {
  console.error('missing config values: url(or relay_url), token, claw_id');
  process.exit(1);
}

const client = createRelayClient(config);
client.start();
console.log(`relay client connecting as ${config.claw_id} to ${config.url}`);

process.on('SIGINT', () => {
  client.stop();
  process.exit(0);
});
process.on('SIGTERM', () => {
  client.stop();
  process.exit(0);
});
