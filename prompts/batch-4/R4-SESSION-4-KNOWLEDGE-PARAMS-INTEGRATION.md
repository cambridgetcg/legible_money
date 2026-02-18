# R4 Session 4: Knowledge ExtendedParams Integration Verification

## Context
Session R3-4 moved 60+ constants into ExtendedParams. The session report shows 35 files modified and ~172 new param references. But there's a critical insight from R3-5's report:

> "go build ./... fails in x/knowledge (missing ExtendedParams arguments) — pre-existing, unrelated to staking"

This means R3-4's changes may have introduced build failures that weren't caught because R3-4 tested only `./x/knowledge/...` in isolation. Other modules that import x/knowledge types may be broken.

## Tasks

### 1. Full Build Verification
```bash
go build ./...
```
If this fails, fix all compilation errors. The most likely issue: functions whose signatures changed (now take `ExtendedParams`) are called from outside x/knowledge.

### 2. Cross-Module Call Sites
Search for all external callers of the 6 functions whose signatures changed:
- `CalculateDecayedConfidence`
- `CalculateDecayedConfidenceNovel`
- `calculateDecayedConfidenceWithMultiplier`
- `CalculateMaturity`
- `NeedsReverification`
- `ApplyReverificationBonus`

```bash
grep -rn "CalculateDecayedConfidence\|CalculateMaturity\|NeedsReverification\|ApplyReverificationBonus" --include="*.go" | grep -v "_test.go" | grep -v "x/knowledge/"
```

Fix any external callers to pass ExtendedParams.

### 3. ExtendedParams Fallback Safety
The helper methods (GetCategoryDecayProfile, GetEvidenceTypeStrength, etc.) fall back to hardcoded maps when the param map is empty. Verify this fallback works correctly:
- New chain with default genesis → ExtendedParams populated → direct lookup works
- Existing chain upgrading → ExtendedParams may be empty → fallback to hardcoded maps
- Manually test with empty ExtendedParams → all getters return valid defaults

### 4. Validation Completeness
R3-4 added validation for BPS bounds, monotonic ordering, etc. Check for gaps:
- Are all map values validated? (CategoryDecayProfiles, EvidenceTypeStrengths, StratumConfidenceCeilings, ChallengeWindowMultipliers)
- Can governance set a category decay profile with HalfLife=0 and MaxConfidence=0? (Should it be allowed?)
- Can governance add new categories that don't exist in the code?

### 5. Golden Vector Integrity
```bash
go test ./tests/cross_stack/... -v
```
Cross-stack golden vectors must still pass. If R3-4 changed any default values (even accidentally), vectors will break. Verify and document.

### 6. Event Emission
When ExtendedParams are updated via MsgUpdateExtendedParams, is an event emitted? Add one if missing — governance parameter changes should be observable.

## Verification
```bash
go build ./...
go test ./...
```
ALL packages must pass, not just x/knowledge.

## Report
Write to `/reports/batch-4/SESSION-R4-4-KNOWLEDGE-INTEGRATION.md` with:
- Build status (clean or fixed N errors)
- External call sites found and fixed
- Fallback behavior verified
- Validation gaps found
- Golden vector status
