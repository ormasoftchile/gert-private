# Decision: Deep-merge YAML serialization for comment preservation

**Date:** 2026-03-18
**By:** Gon (Lead Architect)
**Status:** Implemented

## What

Replace the top-level `doc.set(key, doc.createNode(value))` save approach with a recursive `deepMergeNode()` that walks the YAML AST in parallel with the edited JS object, preserving comments and formatting for unchanged values.

## Why

The previous approach created fresh AST nodes from plain JS objects, discarding all inline comments, key ordering, and formatting. Users who hand-edit YAML with carefully placed comments expect them to survive visual editor saves.

## Tradeoffs

- **Positional sequence merge** is correct for step arrays (add/remove at end) but imperfect if steps are reordered mid-array. Acceptable for MVP — reordering is rare and the fallback (fresh node for changed positions) still produces valid YAML.
- **Fallback to full stringify** if the merge throws, with a user-visible warning. No silent data loss.
