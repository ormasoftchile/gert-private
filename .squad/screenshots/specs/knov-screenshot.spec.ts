/**
 * knov-screenshot.spec.ts
 *
 * One-off visual capture spec written by Knov (Playwright & E2E Testing Specialist).
 * Demonstrates that screenshots can be taken autonomously — no manual pasting needed.
 *
 * Run with:
 *   cd /Volumes/Projects/gert/web
 *   npx playwright test tests/specs/knov-screenshot.spec.ts --reporter=list --project=chromium
 */

import { test, expect } from '@playwright/test';
import * as path from 'path';

const RUNBOOK_PATH =
  '/Volumes/Projects/gert/examples/windows-diagnostic/runbooks/network-health-check.runbook.yaml';

const SCREENSHOT_DIR = '/Volumes/Projects/gert/.squad/screenshots';

test.describe('Knov Visual Capture', () => {
  test.setTimeout(120_000);
  test('captures before and after screenshots of runbook execution', async ({ page }) => {
    // ── Navigate & capture initial (before) state ──────────────────────────
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Click the Runbook Runner tab
    const runnerTab = page.locator('[data-testid="tab-runner"]');
    await runnerTab.waitFor({ state: 'visible', timeout: 15000 });
    await runnerTab.click();

    // Wait for the path input to be ready
    const pathInput = page.locator('[data-testid="runbook-path-input"]');
    await pathInput.waitFor({ state: 'visible', timeout: 15000 });

    // Take BEFORE screenshot
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, 'knov-before.png'),
      fullPage: true,
    });

    // ── Fill in runbook path and start run ─────────────────────────────────
    await pathInput.fill(RUNBOOK_PATH);

    const runButton = page.locator('[data-testid="run-button"]');
    await runButton.waitFor({ state: 'visible', timeout: 10000 });
    await runButton.click();

    // Wait for workflow map to appear (run has started)
    const graph = page.locator('[data-testid="workflow-map"] svg');
    await graph.waitFor({ state: 'visible', timeout: 20000 });

    // Wait for run-outcome if it appears (run has completed)
    const runOutcome = page.locator('[data-testid="run-outcome"]');
    await runOutcome.waitFor({ state: 'visible', timeout: 90000 }).catch(() => {});

    // Take AFTER screenshot
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, 'knov-after.png'),
      fullPage: true,
    });

    // No strict assertion on run outcome to keep visual capture resilient
  });
});
