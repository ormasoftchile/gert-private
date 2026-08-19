# Session Log: Chat-Mediated MCP Execution

**Date:** 2026-08-19T02-05-00Z  
**Agent:** Ken (ken-13)  
**Commits:** gert-vscode `9dfd345` → `4560d82`

## What shipped

Replaced the cached-token bridge with a chat-mediated execution architecture. The `@gert /run` command holds the VS Code chat handler open via `Promise.race([terminal, cancel, deadline])`. `vscode.lm.invokeTool()` is called only from the live handler's execution context via an in-handler invocation pump (`RunPump`). Two-attempt fallback derived from Petals: retry with `token=undefined` on Canceled-class error.

## Key finding

The tool invocation token is not an authorization credential (Petals comments it as "to avoid confirmation dialogs"). The pre-invoke `invocation_token_unavailable` gate was a false premise; replaced by `no_active_run`.

## Empirically unproven

Whether VS Code enforces strict synchronous handler stack vs. awaited continuation for `invokeTool`. Only a live session settles this.

## Tests

164/164/0/0 at 4560d82. Coordinator independently verified. Round 1 had two rejections: PUMP-8 vacuous (third occurrence), deadlock not a test failure. Both fixed in Round 2.
