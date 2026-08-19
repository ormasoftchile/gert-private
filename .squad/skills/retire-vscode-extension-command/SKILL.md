# SKILL: Retiring a Throwaway VS Code Extension Command

**Domain:** VS Code Extension Maintenance  
**Author:** Ken  
**Date:** 2026-08-19  
**Trigger:** Removing a diagnostic/throwaway chat command (or any extension command) from a VS Code extension cleanly.

---

## The Pattern

When retiring a VS Code extension command (especially a chat participant command), touch **five surfaces** in order:

### 1. Manifest (`package.json`)
- Remove the entry from `contributes.chatParticipants[n].commands` (or `contributes.commands` for workbench commands).
- Remove any configuration properties (`contributes.configuration.properties`) that exist solely to support the retiring command.

### 2. Handler dispatch (`src/extension.ts` or equivalent entry point)
- Find the `if (request.command === '<name>') { ... }` branch and delete it entirely.
- Update the "unknown command" fallback help text to remove the mention.

### 3. Source module (`src/<commandName>.ts`)
- Delete the file outright if it is not imported by anything else.
- Confirm no cross-module imports using grep before deleting.

### 4. Test file (`test/<commandName>.test.js`)
- Delete the associated test file.

### 5. Build artifacts (`out/<commandName>.js`, `out/<commandName>.js.map`)
- Run `git ls-files out/<commandName>.js` to check if tracked.
- If **not tracked**: leave them — next `npm run compile` overwrites them.
- If **tracked**: `git rm` them.

---

## Checklist

```
[ ] package.json: command entry removed from chatParticipants.commands
[ ] package.json: any command-specific configuration.properties removed
[ ] extension.ts: dispatch branch deleted
[ ] extension.ts: help text updated
[ ] extension.ts: import for the handler module removed (grep first)
[ ] src/<module>.ts: deleted
[ ] test/<module>.test.js: deleted
[ ] out/<module>.js / .js.map: git-tracked check done; git-rm if needed
[ ] grep --entire repo for command name, slug, and camelCase: no residual refs
[ ] npm run compile: exit 0
[ ] npm test: all pass
[ ] git status: only intended files changed
```

---

## Gotchas

### Missing import is a latent compile error
If the command handler is called but its module was never imported in `extension.ts`, the branch compiles with `any` inference or fails silently. Removing the branch may **fix** a pre-existing compile error rather than introducing one. Always run `npm run compile` after removal — a clean exit confirms this.

### Pre-existing working-tree changes
`git status` before starting may reveal the manifest was already partially cleaned (e.g. by a prior session). Confirm alignment with the decision before assuming they are intentional. Do not revert aligned pre-existing changes.

### Config properties tied to the command
Check `diagnostics.*` or any namespace that was introduced alongside the command. If the property's only consumer is the retiring command branch, remove it from `package.json` too.

### `out/` build artifacts
VS Code extension `out/` directories are typically gitignored but sometimes tracked (e.g., for CI-free installs). Always verify with `git ls-files` before assuming.

---

## Example Application

Removed `@gert /probe-token` from `gert-vscode` (2026-08-19):
- Deleted `src/probeToken.ts` (419 lines) and `test/probeToken.test.js` (456 lines).
- Removed dispatch branch from `extension.ts` (24 lines).
- Package.json `probe-token` command + `gert.diagnostics.unsafeErrorText` config were already removed.
- `out/` artifacts not git-tracked; left for rebuild.
- Compile: ✅. Tests: 185/185 ✅.
