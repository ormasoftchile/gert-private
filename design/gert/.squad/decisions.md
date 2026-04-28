# Squad Decisions Log

## 2026-04-28: Delegation Spec — Final Approval

**Session**: gert-domain-home delegation spec — Rounds 4-6 fix/review cycle  
**Decision**: ✅ APPROVED FOR MERGE  
**Consensus**: 5/5 reviewers (Ken, Ada, James, Barbara, John)

### Resolution

After 6 review rounds and 2 fix cycles:
- API schema now canonical (snake_case, field names aligned across schema/API/docs)
- Kotlin SDK corrected: STEP_COMPLETED constant, payload direct cast
- Required fields properly marked (delegateContact in POST/PUT)
- assigns transformation documented (flat API arrays ↔ YAML discriminated union)
- casa-santiago unit/type fields aligned
- All blocking issues resolved; no remaining concerns

### Commits

- f83cff2: ken-apply-fixes-4 (10 initial fixes)
- 777d19f, 80b6344: james-sdk-fix (SDK runtime fixes)
- d36106b: ken-apply-fixes-5 (schema alignment, Kotlin corrections)
- 7e1373c: review6 round (inline S09 label fix)

### Action Items

- ✅ Merge into main
- ✅ Deploy updated schema and API docs
- ✅ Release Kotlin SDK with corrected constants/payload handling
- ✅ Update integration guides
