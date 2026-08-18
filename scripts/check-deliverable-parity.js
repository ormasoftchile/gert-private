#!/usr/bin/env node
//
// check-deliverable-parity.js
//
// Compares each consumer-facing deliverable in deliverables/phase2/*.vscode-mcp.tool.yaml
// against the corresponding test fixture in the gert core repo at
// internal/tool/testdata/*.vscode-mcp.tool.yaml.
//
// The gert testdata file is authoritative for gert's own tests.
// The deliverable is the consumer-facing artifact.
// This script proves they agree, byte-for-byte.
//
// Usage:
//   node scripts/check-deliverable-parity.js <gert-checkout-path>
//   GERT_CHECKOUT=<path> node scripts/check-deliverable-parity.js
//
// The gert checkout path MUST be supplied. If it is absent or the directory
// is not found the script exits non-zero with an actionable message.
// It will never silently skip the comparison.
//
// Exit codes:
//   0 — all files match
//   1 — one or more files drifted, or configuration error

'use strict';

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

// ---------------------------------------------------------------------------
// Resolve the gert checkout path
// ---------------------------------------------------------------------------
const gertCheckout = process.argv[2] || process.env.GERT_CHECKOUT || '';

if (!gertCheckout) {
  console.error('ERROR: gert checkout path not supplied.');
  console.error('');
  console.error('Supply it as the first argument or set GERT_CHECKOUT:');
  console.error('  node scripts/check-deliverable-parity.js <path-to-gert-checkout>');
  console.error('  GERT_CHECKOUT=<path> node scripts/check-deliverable-parity.js');
  console.error('');
  console.error('The gert checkout is required. This script never skips the comparison.');
  process.exit(1);
}

const gertTestdata = path.join(gertCheckout, 'internal', 'tool', 'testdata');

if (!fs.existsSync(gertTestdata)) {
  console.error(`ERROR: gert testdata directory not found at: ${gertTestdata}`);
  console.error('');
  console.error('Verify the supplied path is the root of a gert checkout that contains');
  console.error('  internal/tool/testdata/');
  console.error('');
  console.error(`Supplied path: ${gertCheckout}`);
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Collect deliverables
// ---------------------------------------------------------------------------
const scriptDir = path.dirname(path.resolve(process.argv[1]));
const repoRoot = path.dirname(scriptDir);
const deliverablesDir = path.join(repoRoot, 'deliverables', 'phase2');

if (!fs.existsSync(deliverablesDir)) {
  console.error(`ERROR: deliverables directory not found at: ${deliverablesDir}`);
  process.exit(1);
}

const deliverables = fs.readdirSync(deliverablesDir)
  .filter(f => f.endsWith('.vscode-mcp.tool.yaml'))
  .sort();

if (deliverables.length === 0) {
  console.error('ERROR: no *.vscode-mcp.tool.yaml files found under deliverables/phase2/');
  console.error('This is unexpected — the check cannot be vacuously empty.');
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Compare each deliverable against the gert fixture
// ---------------------------------------------------------------------------
function sha256(buf) {
  return crypto.createHash('sha256').update(buf).digest('hex');
}

let drifted = 0;
let matched = 0;

console.log(`Comparing ${deliverables.length} deliverable(s):`);
console.log(`  deliverables : ${deliverablesDir}`);
console.log(`  gert testdata: ${gertTestdata}`);
console.log('');

for (const filename of deliverables) {
  const deliverablePath = path.join(deliverablesDir, filename);
  const fixturePath = path.join(gertTestdata, filename);

  if (!fs.existsSync(fixturePath)) {
    console.error(`  MISSING  ${filename}`);
    console.error(`           deliverable exists but gert fixture not found at:`);
    console.error(`           ${fixturePath}`);
    drifted++;
    continue;
  }

  const deliverableBuf = fs.readFileSync(deliverablePath);
  const fixtureBuf = fs.readFileSync(fixturePath);

  const deliverableHash = sha256(deliverableBuf);
  const fixtureHash = sha256(fixtureBuf);

  if (deliverableHash === fixtureHash) {
    console.log(`  OK       ${filename}`);
    matched++;
  } else {
    console.error(`  DRIFTED  ${filename}`);
    console.error(`           deliverable: ${deliverableHash}`);
    console.error(`           gert fixture: ${fixtureHash}`);
    // Show first differing line for diagnosability
    const deliverableLines = deliverableBuf.toString('utf8').split('\n');
    const fixtureLines = fixtureBuf.toString('utf8').split('\n');
    const maxLines = Math.max(deliverableLines.length, fixtureLines.length);
    for (let i = 0; i < maxLines; i++) {
      const dl = deliverableLines[i] ?? '(absent)';
      const fl = fixtureLines[i] ?? '(absent)';
      if (dl !== fl) {
        console.error(`           first diff at line ${i + 1}:`);
        console.error(`             deliverable : ${dl}`);
        console.error(`             gert fixture: ${fl}`);
        break;
      }
    }
    drifted++;
  }
}

console.log('');
console.log(`Result: ${matched} matched, ${drifted} drifted`);

if (drifted > 0) {
  console.error('');
  console.error('FAIL: deliverables and gert testdata fixtures have drifted.');
  console.error('Update the gert fixture or the deliverable so they agree.');
  console.error('The gert testdata file is authoritative for gert\'s own tests;');
  console.error('the deliverable is the consumer-facing artifact — they must be identical.');
  process.exit(1);
}

console.log('PASS: all deliverables match their gert testdata fixtures.');
process.exit(0);
