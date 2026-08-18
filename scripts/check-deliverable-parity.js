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

// ---------------------------------------------------------------------------
// testdata files that are intentional synthetic-only fixtures.
// These appear in gert's internal/tool/testdata/ for testing purposes but are
// NOT deliverables and have no corresponding file in deliverables/phase2/.
// Add an entry here ONLY when a file is deliberately testdata-only; do not
// use this list to suppress genuine parity gaps.
// ---------------------------------------------------------------------------
const TESTDATA_SYNTHETIC_ONLY = new Set([
  'vscode-mcp-ops-synthetic.tool.yaml',
]);

// ---------------------------------------------------------------------------
// Collect file sets from both sides (general .tool.yaml suffix).
// Matching on the general suffix rather than a specific infix (e.g.
// .vscode-mcp.) means a renamed file keeps appearing in the comparison
// instead of silently dropping out of coverage.
// ---------------------------------------------------------------------------
const delivSet = new Set(
  fs.readdirSync(deliverablesDir).filter(f => f.endsWith('.tool.yaml')).sort()
);
const testSet = new Set(
  fs.readdirSync(gertTestdata).filter(f => f.endsWith('.tool.yaml')).sort()
);

// Compute the union of both sides. Every file in the union must appear on
// both sides (unless it is in TESTDATA_SYNTHETIC_ONLY). Any asymmetry is a
// hard error — a rename on one side alone is caught here.
const allFiles = [...new Set([...delivSet, ...testSet])].sort();

const MIN_COMPARED = 2; // guard: the comparison is never vacuously empty

// ---------------------------------------------------------------------------
// Compare each deliverable against the gert fixture
// ---------------------------------------------------------------------------
function sha256(buf) {
  return crypto.createHash('sha256').update(buf).digest('hex');
}

let drifted = 0;
let matched = 0;

console.log(`Comparing tool contract files (bidirectional set check):`);
console.log(`  deliverables : ${deliverablesDir}`);
console.log(`  gert testdata: ${gertTestdata}`);
console.log(`  union size: ${allFiles.length} file(s) (${TESTDATA_SYNTHETIC_ONLY.size} synthetic-only excluded)`);
console.log('');

for (const filename of allFiles) {
  // Testdata-only synthetic fixtures are explicitly excluded from parity.
  // They live in testdata for internal testing and have no deliverable mirror.
  if (TESTDATA_SYNTHETIC_ONLY.has(filename)) {
    console.log(`  SYNTHETIC ${filename} (testdata-only fixture, excluded from parity)`);
    continue;
  }

  const inDeliv = delivSet.has(filename);
  const inTest  = testSet.has(filename);

  if (inDeliv && !inTest) {
    console.error(`  MISSING-TESTDATA  ${filename}`);
    console.error(`    deliverable exists but gert fixture not found — add it to:`);
    console.error(`    ${gertTestdata}`);
    drifted++;
    continue;
  }

  if (inTest && !inDeliv) {
    console.error(`  ORPHAN-TESTDATA   ${filename}`);
    console.error(`    gert fixture exists but no corresponding deliverable — rename or`);
    console.error(`    add to TESTDATA_SYNTHETIC_ONLY in check-deliverable-parity.js if intentional:`);
    console.error(`    ${path.join(gertTestdata, filename)}`);
    drifted++;
    continue;
  }

  // Both sides have the file — compare byte-for-byte.
  const deliverablePath = path.join(deliverablesDir, filename);
  const fixturePath = path.join(gertTestdata, filename);

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
console.log(`Result: ${matched} matched, ${drifted} drifted (${allFiles.length} files in union, ${TESTDATA_SYNTHETIC_ONLY.size} excluded)`);

if (matched < MIN_COMPARED) {
  console.error('');
  console.error(`FAIL: only ${matched} file(s) compared — expected at least ${MIN_COMPARED}.`);
  console.error('This guards against a vacuously-empty comparison (e.g. both directories emptied).');
  process.exit(1);
}

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
