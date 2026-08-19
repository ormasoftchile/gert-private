'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const { pickServerRoot } = require('../out/serverRoot');
const { resolvePackageMapPath } = require('../out/serverLaunch');
const { buildRegistryFromPackageMap } = require('../out/toolDefinitionRegistry');

test('multi-root active project package map supplies both VS Code bridge actions', (t) => {
  const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'gert-bridge-multiroot-'));
  t.after(() => fs.rmSync(workspace, { recursive: true, force: true }));

  const folder0 = path.join(workspace, 'unrelated-project');
  const folder1 = path.join(workspace, 'sql-livesite');
  fs.mkdirSync(path.join(folder0, 'runbooks'), { recursive: true });
  fs.mkdirSync(path.join(folder0, 'packages'), { recursive: true });
  fs.mkdirSync(path.join(folder1, 'runbooks'), { recursive: true });
  const runbook = path.join(folder1, 'runbooks', 'icm-tsg-router.runbook.yaml');
  fs.writeFileSync(runbook, 'apiVersion: runbook/v1\n');

  const bindingRoot = path.join(folder1, 'packages', 'incident-routing-vscode-mcp');
  fs.mkdirSync(path.join(bindingRoot, 'tools'), { recursive: true });
  fs.writeFileSync(path.join(folder1, 'packages', 'incident-routing.vscode-mcp.package-map.yaml'), `apiVersion: config/v1
requires:
  - package: sql-livesite.incident-routing
    version: "^1.0.0"
    path: ./packages/incident-routing-vscode-mcp
`);
  fs.writeFileSync(path.join(bindingRoot, 'tools', 'icm.tool.yaml'), `apiVersion: tool/v1
meta: { name: icm, version: "1.0.0" }
transport: { mode: vscode-mcp, vscode_tool: icm-get-incident }
actions:
  - name: get-incident
    outputs: { title: { type: string, required: true } }
`);
  fs.writeFileSync(path.join(bindingRoot, 'tools', 'tsg-recommendation.tool.yaml'), `apiVersion: tool/v1
meta: { name: tsg-recommendation, version: "1.0.0" }
transport: { mode: vscode-mcp, vscode_tool: tsg-recommendation-recommend }
actions:
  - name: recommend
    outputs: { recommendation_status: { type: string, required: true } }
`);

  const serverRoot = pickServerRoot(runbook, [folder0, folder1], folder0);
  assert.equal(serverRoot, folder1, 'active runbook project must win over folder 0');
  const packageMap = resolvePackageMapPath(serverRoot, 'packages/incident-routing.vscode-mcp.package-map.yaml');
  const registry = buildRegistryFromPackageMap(serverRoot, packageMap);

  assert.equal(registry['icm/get-incident'].registeredName, 'icm-get-incident');
  assert.equal(registry['tsg-recommendation/recommend'].registeredName, 'tsg-recommendation-recommend');
});
