const assert = require('node:assert/strict');
const path = require('node:path');
const test = require('node:test');

const { localServerAddress, packageMapServeArgs } = require('../out/serverLaunch');

test('uses an explicit loopback address for auto-started servers', () => {
  assert.equal(localServerAddress(61667), '127.0.0.1:61667');
});

test('resolves a workspace package map against the server root', () => {
  const root = path.join(path.parse(process.cwd()).root, 'work', 'sql-livesite');
  assert.deepEqual(
    packageMapServeArgs(root, 'packages/incident-routing.vscode-mcp.package-map.yaml'),
    ['--package-map', path.join(root, 'packages', 'incident-routing.vscode-mcp.package-map.yaml')],
  );
});

test('passes configured path text as a literal spawn argument', () => {
  const root = path.join(path.parse(process.cwd()).root, 'work', 'sql-livesite');
  const configured = 'packages/$(not-a-shell-command).package-map.yaml';
  assert.deepEqual(
    packageMapServeArgs(root, configured),
    ['--package-map', path.join(root, configured)],
  );
});

test('omits the package-map flag when unset', () => {
  assert.deepEqual(packageMapServeArgs('C:\\work\\sql-livesite', '  '), []);
});
