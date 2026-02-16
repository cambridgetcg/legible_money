# W1: Signal Resolution Sprint

> Clear the noise so deeper patterns become visible.

## Why This Is First
Signals are symptoms of misalignment. Active signals obscure the path forward — you can't detect emergent beauty (B4) while ugliness signals are firing. Resolving these is prerequisite to all deeper work.

## Success Criteria
1. [ ] Fix all doc/code discrepancies identified in reflections
2. [ ] Resolve ownership overlaps between docs and module implementations
3. [ ] Makefile targets match documentation (pr-check, go-vet alignment)
4. [ ] Zero injustice signals in next PP assessment
5. [ ] Zero ugliness signals in next PP assessment

## Concrete Work
- Audit: `docs/OPEN_SOURCE_DEV_FLOW.md` references vs actual Makefile targets
- Audit: `CONTRIBUTING.md` (if exists) vs `docs/AGENT_CI_CD_WORKFLOW.md` overlap
- Fix: Any file:line where documentation claims differ from implementation
- Fix: Branch naming pattern consistency (reflection noted T1 branch naming discrepancy)
- Verify: Run `/verify` on completed fixes

## Gates Targeted
| Gate | Current | Target | How |
|------|---------|--------|-----|
| J2 (Ownership) | FAIL | PASS | Each doc owns its scope, no overlaps |
| J6 (Integrity) | FAIL | PASS | Docs match code, code matches docs |
| B4 (Elegance) | FAIL | PARTIAL | Remove redundancy between doc files |
| T1 (Grounding) | FAIL | PASS | All claims cite verified file:line |

## Mastery Impact
- **High** for Justice (J): directly resolves 2 of 3 failing J gates
- **Medium** for Beauty (B): removes ugliness, creates space for emergence
- **Low effort, high signal**: estimated 1 session
