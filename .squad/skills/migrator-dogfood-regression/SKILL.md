---
name: "migrator-dogfood-regression"
description: "Turn already-migrated fixtures into idempotency regression tests for migration tools"
domain: "migration-testing"
confidence: "high"
source: "Stream E Day 2 migrator dogfood"
---

## Pattern

When building a source-to-source migrator, run it against fixtures that are already in the target language and require byte-for-byte no-op output.

## Why

A clean first migration is not enough. A second pass catches unsafe rewrites that would corrupt canonical target syntax, especially where legacy syntax and target syntax overlap.

## How

1. Add a test that walks the migrated fixture corpus.
2. For each target YAML file, run the migrator.
3. Assert zero translations, zero deferrals, and byte-for-byte unchanged output.
4. Run the CLI dogfood command with dry-run and diff disabled.
5. Treat any diff as a migrator bug, not fixture churn.

## Example

For `gert migrate-expr`, the regression walks `design/gert/testdata/runbooks/` and ensures each migrated `schema.yaml` remains unchanged. This caught the `$${amount}` ambiguity: in target GIS it means literal dollar plus interpolation, but a second pass was incorrectly treating it as old E-009 escape syntax.
