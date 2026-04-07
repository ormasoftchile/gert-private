/**
 * all-examples.spec.ts
 *
 * Playwright spec that runs all 13 example runbooks through the gert web UI
 * and captures a screenshot for each, recording the run outcome.
 *
 * Run with:
 *   cd /Volumes/Projects/gert/web
 *   npx playwright test /Volumes/Projects/gert/.squad/screenshots/specs/all-examples.spec.ts \
 *     --reporter=list --project=chromium
 */

import { test, expect } from '@playwright/test';
import * as path from 'path';
import * as fs from 'fs';

// ── Config ────────────────────────────────────────────────────────────────────

const EXAMPLES_DIR = '/Volumes/Projects/gert/examples';
const SCREENSHOTS_DIR = '/Volumes/Projects/gert/.squad/screenshots/examples';
const MAX_RUN_MS = 90_000; // 90 seconds for a full run to complete

// Ensure screenshot directory exists
fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });

// ── Runbook list ──────────────────────────────────────────────────────────────

const RUNBOOKS = [
  'edge-case-branch-target-1.runbook.yaml',
  'edge-case-branch-target-2.runbook.yaml',
  'edge-case-branch.runbook.yaml',
  'edge-case-single-step-timeout.runbook.yaml',
  'edge-case-single-step.runbook.yaml',
  'incident-triage.app-crash.runbook.yaml',
  'incident-triage.connectivity-test.runbook.yaml',
  'incident-triage.network.runbook.yaml',
  'incident-triage.resource-exhaustion.runbook.yaml',
  'incident-triage.runbook.yaml',
  'multi-region-rollout.runbook.yaml',
  'simple-health-check.runbook.yaml',
  'service-health-branching.runbook.yaml',
];

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Derive screenshot filename from runbook name (strip .runbook.yaml suffix). */
function screenshotName(runbook: string): string {
  return runbook.replace('.runbook.yaml', '') + '.png';
}

/**
 * Navigate to the runner view and prepare it for a new run.
 * Returns once the path input is visible and ready.
 */
async function navigateToRunner(page: import('@playwright/test').Page): Promise<void> {
  await page.goto('/');
  await page.waitForLoadState('domcontentloaded');

  const tab = page.locator('[data-testid="tab-runner"]');
  await tab.waitFor({ state: 'visible', timeout: 15_000 });
  await tab.click();

  await page.locator('[data-testid="runbook-path-input"]').waitFor({ state: 'visible', timeout: 15_000 });
}

/**
 * Run a runbook end-to-end, handling choice modals along the way.
 *
 * Returns the recorded outcome string or "timeout" / "error".
 */
async function runRunbook(
  page: import('@playwright/test').Page,
  runbookPath: string,
): Promise<string> {
  // Fill path and start run
  const pathInput = page.locator('[data-testid="runbook-path-input"]');
  await pathInput.fill(runbookPath);
  await page.locator('[data-testid="run-button"]').click();

  // Wait for step list to appear (execution started)
  const graph = page.locator('[data-testid="workflow-map"] svg');
  await graph.waitFor({ state: 'visible', timeout: 15_000 });

  const deadline = Date.now() + MAX_RUN_MS;

  // Poll until run-outcome appears, handling choice modals along the way
  while (Date.now() < deadline) {
    // Check for error banner first
    const errorBanner = page.locator('[data-testid="run-error"]');
    const errorVisible = await errorBanner.isVisible().catch(() => false);
    if (errorVisible) {
      return 'error';
    }

    // Check for run-outcome
    const runOutcome = page.locator('[data-testid="run-outcome"]');
    const outcomeVisible = await runOutcome.isVisible().catch(() => false);
    if (outcomeVisible) {
      const outcome = await runOutcome.getAttribute('data-outcome').catch(() => null);
      return outcome || 'completed';
    }

    // Check for choice buttons and auto-select first option
    const firstOption = page.locator('[data-testid="choice-option-0"]');
    const optionVisible = await firstOption.isVisible().catch(() => false);
    if (optionVisible) {
      await firstOption.click();
      // Give the UI time to advance
      await page.waitForTimeout(500);
      continue;
    }

    // Short wait before next poll
    await page.waitForTimeout(500);
  }

  return 'timeout';
}

// ── Tests ─────────────────────────────────────────────────────────────────────

test.describe('All example runbooks', () => {
  // Allow 120s per test (90s run + startup overhead)
  test.setTimeout(120_000);

  for (const runbook of RUNBOOKS) {
    const fullPath = path.join(EXAMPLES_DIR, runbook);
    const screenshotPath = path.join(SCREENSHOTS_DIR, screenshotName(runbook));

    test(`runs ${runbook}`, async ({ page }) => {
      await navigateToRunner(page);

      const outcome = await runRunbook(page, fullPath);

      // Capture screenshot after run (or timeout)
      await page.screenshot({ path: screenshotPath, fullPage: true });

      console.log(`[${runbook}] outcome=${outcome}  screenshot=${screenshotPath}`);

      // All 13 outcomes are recorded; no hard assertion so every test completes
      // and captures a screenshot regardless of backend errors or timeouts.
      // Inspect the outcome table in CI output to identify regressions.
      expect(['success', 'failure', 'error', 'timeout', 'completed', 'needs_rca', 'resolved',
              'escalated', 'mitigated', 'skipped']).toContain(outcome);
    });
  }
});
