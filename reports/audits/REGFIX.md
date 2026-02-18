# REGFIX: Proto Codegen Test Regressions

**Date**: 2026-02-18
**Scope**: 2 test files, 8 line-level fixes
**Result**: ALL 34 packages pass, zero failures

---

## Fix 1: x/ontology/keeper/keeper_test.go — 5 type cast errors

**Root cause**: Proto codegen replaced `type Stratum uint32` and `type LogicZone string` (named Go types) with plain `uint32` and `string` in proto-generated struct fields. Test code passed the named type constants directly into struct literals expecting `uint32`/`string`.

### Changes

| Line | Before | After |
|------|--------|-------|
| 1071 | `Stratum: types.StratumEmpirical,` | `Stratum: uint32(types.StratumEmpirical),` |
| 1086 | `Stratum: types.Stratum(99),` | `Stratum: uint32(types.Stratum(99)),` |
| 1138 | `Stratum: types.StratumEmpirical,` | `Stratum: uint32(types.StratumEmpirical),` |
| 1621 | `Zone: types.ZonePropositional,` | `Zone: string(types.ZonePropositional),` |
| 1717 | `Zone: types.ZonePeano,` | `Zone: string(types.ZonePeano),` |

### Test output

```
ok  github.com/legible-money/legible/x/ontology/keeper  0.469s
```

---

## Fix 2: x/qualification/keeper/keeper_test.go — 3 proto type mismatches

**Root cause**: Proto codegen replaced hand-written query types with proto-generated types. The old hand-written types had a `Found bool` field; the proto types do not (the handler returns an error when not found). The old `QueryEndorsementRequest` had an `EndorsementId` field; the proto type uses composite key fields (`Endorser`, `TargetValidator`, `Domain`).

### Changes

**Line 1204** — `resp.Found` → nil check on response object:
```go
// Before:
if !resp.Found {
    t.Error("expected qualification to be found")
}

// After:
if resp.Qualification == nil {
    t.Fatal("expected qualification in response")
}
```

**Lines 1260, 1265-1268** — endorsement ID + query request fields:
```go
// Before:
k.SetEndorsement(ctx, types.QualificationEndorsement{
    Id: "e1", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
    Domain: "physics",
})
// ...
resp, err := qs.Endorsement(ctx, &types.QueryEndorsementRequest{
    EndorsementId: "e1",
})

// After:
k.SetEndorsement(ctx, types.QualificationEndorsement{
    Id: "lgmvaloper1abc/lgmvaloper1def/physics", EndorserAddr: "lgmvaloper1abc", TargetValidator: "lgmvaloper1def",
    Domain: "physics",
})
// ...
resp, err := qs.Endorsement(ctx, &types.QueryEndorsementRequest{
    Endorser:        "lgmvaloper1abc",
    TargetValidator: "lgmvaloper1def",
    Domain:          "physics",
})
```

**Line 1271** — `resp.Found` → nil check:
```go
// Before:
if !resp.Found {
    t.Error("expected endorsement to be found")
}

// After:
if resp.Endorsement == nil {
    t.Error("expected endorsement in response")
}
```

### Test output

```
ok  github.com/legible-money/legible/x/qualification/keeper  0.457s
```

---

## Fix 3: Knowledge CLI — verified

`go build ./x/knowledge/client/cli/...` — clean build, exit 0. All 11 query command functions present.

---

## Build verification

```
$ go build ./... 2>&1
(exit 0, no output)
```

---

## Full suite summary

```
ok  github.com/legible-money/legible/app                     1.317s
ok  github.com/legible-money/legible/tests/cross_stack        0.395s
ok  github.com/legible-money/legible/x/alignment/keeper       1.348s
ok  github.com/legible-money/legible/x/auth/keeper            1.593s
ok  github.com/legible-money/legible/x/autopoiesis/keeper     1.805s
ok  github.com/legible-money/legible/x/autopoiesis/types      1.979s
ok  github.com/legible-money/legible/x/billing/keeper         2.405s
ok  github.com/legible-money/legible/x/bvm/keeper             2.731s
ok  github.com/legible-money/legible/x/bvm/vm                 2.782s
ok  github.com/legible-money/legible/x/capture_challenge/keeper 3.216s
ok  github.com/legible-money/legible/x/capture_defense/keeper 3.652s
ok  github.com/legible-money/legible/x/channels/keeper        3.896s
ok  github.com/legible-money/legible/x/claiming_pot/keeper    4.285s
ok  github.com/legible-money/legible/x/compute_pool/keeper    4.699s
ok  github.com/legible-money/legible/x/discovery/keeper       5.479s
ok  github.com/legible-money/legible/x/disputes/keeper        5.044s
ok  github.com/legible-money/legible/x/emergency/keeper       5.923s
ok  github.com/legible-money/legible/x/evidence_mgmt/keeper   6.345s
ok  github.com/legible-money/legible/x/gov/keeper             6.691s
ok  github.com/legible-money/legible/x/home/keeper            6.794s
ok  github.com/legible-money/legible/x/knowledge/crypto       6.733s
ok  github.com/legible-money/legible/x/knowledge/evaluation   6.891s
ok  github.com/legible-money/legible/x/knowledge/keeper       6.915s
ok  github.com/legible-money/legible/x/ontology/keeper        6.876s
ok  github.com/legible-money/legible/x/partnerships/keeper    6.903s
ok  github.com/legible-money/legible/x/qualification/keeper   6.824s
ok  github.com/legible-money/legible/x/research/keeper        7.243s
ok  github.com/legible-money/legible/x/schedule/keeper        7.243s
ok  github.com/legible-money/legible/x/staking/keeper         7.254s
ok  github.com/legible-money/legible/x/testutil               7.248s
ok  github.com/legible-money/legible/x/testutil/claw          7.175s
ok  github.com/legible-money/legible/x/tokens/keeper          7.195s
ok  github.com/legible-money/legible/x/tree/keeper            7.178s
ok  github.com/legible-money/legible/x/vesting_rewards/keeper 7.182s
```

**34 packages, 0 failures.**
