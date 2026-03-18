# Bug Fixes: Add-Step Handler Critical Bugs

**Timestamp:** 2026-03-18T15:00:00Z  
**By:** Kurapika (Frontend Engineer)  
**Issue:** Acceptance test bugs #1-3 in runbookEditorPanel.ts  
**Status:** ✅ FIXED

## Summary

Fixed 3 critical bugs in the add-step handler that prevented step insertion into branch and iterate nodes.

## Changes Made

### 1. getSelectionType() - Iterate Node Detection (Line 760)
**Problem:** Root iterate nodes at path [0] were not detected because only `includes('iterate')` check was used.

**Fix:** Added node.iterate property check BEFORE the includes() check:
```typescript
} else if (node && node.iterate) {
  return 'iterate';
}
```

**Result:** Root iterate nodes [0] now correctly identified as 'iterate' type.

### 2. add-step Handler - Branch Path Construction (Line 1137)
**Problem:** Path was double-appended: `[...parentPath, 'steps']` when parentPath already contained 'steps'.  
Result: `[0, 'branches', 0, 'steps', 'steps']` ❌

**Fix:** Modified conditional logic to NOT append 'steps' for branch/iterate cases:
```typescript
} else if (selType === 'branch' || selType === 'iterate') {
  // Branch and iterate paths already include 'steps'
  this.addStepAtPath(parentPath as any, newStep);
} else {
  // For other cases, append 'steps'
  this.addStepAtPath([...parentPath, 'steps'] as any, newStep);
}
```

**Result:** Branch path = `[0, 'branches', 0, 'steps']` ✓

### 3. add-step Handler - Iterate Path Construction (Line 1139)
**Problem:** Iterate path was just `this.selectedNodePath` without 'iterate' and 'steps' appended.

**Fix:** Changed iterate case to:
```typescript
} else if (selType === 'iterate') {
  // Add inside iterate's steps
  parentPath = [...this.selectedNodePath, 'iterate', 'steps'];
}
```

**Result:** Iterate path = `[0, 'iterate', 'steps']` ✓

## Verification

**Phase 4 Trace (Branch):**
- selectedNodePath = [0, 'branches', 0]
- getSelectionType() → 'branch'
- parentPath = [0, 'branches', 0, 'steps']
- addStepAtPath([0, 'branches', 0, 'steps'], newStep) ✓

**Phase 5 Trace (Iterate):**
- selectedNodePath = [0]
- getNodeAtPath([0]).iterate exists
- getSelectionType() → 'iterate'
- parentPath = [0, 'iterate', 'steps']
- addStepAtPath([0, 'iterate', 'steps'], newStep) ✓

## Impact

- ✅ Steps can be added to branch nodes
- ✅ Steps can be added to iterate nodes
- ✅ No more double-'steps' path errors
- ✅ Hisoka's acceptance tests should now pass
