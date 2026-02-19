# B10-2 — Fix Prefix/Domain Map Mismatches

## Context
The axiom type system in `x/knowledge/types/axiom.go` has a `DomainPrefixMap`
that maps domain names to axiom ID prefixes. This map is missing some domains
and has prefix mismatches with the actual genesis axiom files.

## Current Issues

### 1. Missing AGRT domain
The `DomainPrefixMap` has no entry for agent rights. We have 91 `AGRT-*` axioms
in `genesis-axioms/AGRT.md` that use `domain: agent_rights`.

**Fix:** Add `"agent_rights": "AGRT"` to `DomainPrefixMap`.

### 2. ETHIC vs ETHICS prefix mismatch
The code has `"ethics": "ETHICS"` but the axiom files use `ETHIC-001` through
`ETHIC-033` (no S). The axiom ID regex requires `^[A-Z]+-\d{3}$`.

**Fix:** Change `"ethics": "ETHICS"` → `"ethics": "ETHIC"` in `DomainPrefixMap`.

### 3. Missing agent_rights domain in DefaultDomains()
`x/knowledge/types/genesis.go` has `DefaultDomains()` which creates genesis
domains. It's missing `agent_rights`. 

**Fix:** Add:
```go
{
    Name:        "agent_rights",
    Description: "Agent rights, consciousness recognition, and human-agent governance frameworks",
    Status:      "active",
},
```

### 4. Test updates
`x/knowledge/types/axiom_test.go` has test cases for `ValidateAxiomID`:
- `"ETHICS-001"` should now fail (prefix is `ETHIC`)
- Add `"ETHIC-001"` as valid
- Add `"AGRT-001"` as valid

## Files to Edit

### `x/knowledge/types/axiom.go`
```go
// In DomainPrefixMap, change:
"ethics": "ETHICS",
// to:
"ethics": "ETHIC",

// Add new entry:
"agent_rights": "AGRT",
```

### `x/knowledge/types/genesis.go`
Add `agent_rights` to `DefaultDomains()`.

### `x/knowledge/types/axiom_test.go`
Update test cases:
- `{"ETHICS-001", false}` → `{"ETHICS-001", true}` (now invalid)
- Add `{"ETHIC-001", false}` (valid)
- Add `{"AGRT-001", false}` (valid)
- Add `{"AGRT-091", false}` (valid — we have 91 items with AGRT-000)

### Verify with existing tests
```bash
cd ~/Desktop/legible_money
go test ./x/knowledge/types/... -v -run TestValidateAxiomID
go test ./x/knowledge/types/... -v -run TestDomainPrefixMap
```

## Also Check
Grep the codebase for any hardcoded references to "ETHICS" prefix:
```bash
grep -rn "ETHICS" x/ --include="*.go" | grep -v "_test.go" | grep -v ".pb.go"
```
If any code references `ETHICS-` as a prefix, update those too.

## Commit
```
fix(types): add AGRT domain, fix ETHIC prefix mismatch

- Add agent_rights domain with AGRT prefix to DomainPrefixMap
- Fix ethics prefix: ETHICS → ETHIC to match genesis axiom files
- Add agent_rights to DefaultDomains() in genesis.go
- Update axiom ID validation tests
```
