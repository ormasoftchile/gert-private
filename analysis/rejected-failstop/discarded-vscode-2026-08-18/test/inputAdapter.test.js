'use strict';

const assert = require('node:assert/strict');
const http = require('node:http');
const test = require('node:test');

const { McpBridge, adaptInput } = require('../out/mcpBridge');

const spec = {
  registeredName: 'icm-get-incident',
  inputFields: { incident_id: { type: 'string', required: true } },
  inputAdapter: { incidentId: { from: 'incident_id', coerce: 'integer' } },
  outputFields: {},
};

function call(bridge, args) {
  return new Promise((resolve, reject) => {
    const url = new URL(bridge.bridgeUrl);
    const body = JSON.stringify({
      version: 'vscode-mcp-bridge/v1',
      request_id: 'adapter-test',
      tool: 'icm',
      action: 'get-incident',
      args,
      capability_proof: bridge.bridgeToken,
    });
    const req = http.request({
      hostname: url.hostname, port: url.port, path: '/', method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(body) },
    }, (res) => {
      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => resolve(JSON.parse(Buffer.concat(chunks).toString())));
    });
    req.on('error', reject);
    req.end(body);
  });
}

test('adapter mapping is load-bearing at bridge invocation', async (t) => {
  let received;
  const bridge = await McpBridge.create({
    tools: [{ name: 'icm-get-incident' }],
    async invokeTool(_name, options) {
      received = options.input;
      return { content: [{ value: '{}' }] };
    },
  }, 0, undefined, { registry: { 'icm/get-incident': spec } });
  t.after(() => bridge.dispose());

  const body = await call(bridge, { incident_id: '123456' });
  assert.equal(body.error, undefined);
  assert.deepEqual(received, { incidentId: 123456 });
});

test('adapter rejects invalid source values and unmapped logical arguments without exposing values', () => {
  for (const args of [
    {},
    { incident_id: '1.5' },
    { incident_id: '9007199254740992' },
    { incident_id: '123', extra: 'provider-secret' },
  ]) {
    const result = adaptInput('icm/get-incident', args, spec);
    assert.equal(result.ok, false);
    assert.doesNotMatch(result.reason, /1\.5|9007199254740992|provider-secret/);
  }
});
