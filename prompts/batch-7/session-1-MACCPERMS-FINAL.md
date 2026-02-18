# Session: MACCPERMS-FINAL — Fix the sole testnet blocker + comprehensive maccPerms audit

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Add ALL missing module accounts to maccPerms — the sole testnet blocker

### Context
B6-7 TESTNET-READINESS found 6 modules that call `SendCoinsFromAccountToModule` / `SendCoinsFromModuleToAccount` / `BurnCoins` / `MintCoins` but are NOT in the `maccPerms` map in `app/app.go`. At runtime, the bank keeper panics for unregistered module names.

Previous batches already caught and fixed `lgm_gov` and `knowledge`. This pattern keeps recurring because tests use mock bank keepers.

### Step 1: Comprehensive audit — find ALL missing modules
Do NOT just add the 6 known missing modules. Do a full sweep to ensure nothing else is missed:

```bash
# Find ALL module names used in bank keeper calls across the codebase
grep -rn "SendCoinsFromAccountToModule\|SendCoinsFromModuleToAccount\|SendCoinsFromModuleToModule\|BurnCoins\|MintCoins" x/ app/ --include="*.go" | grep -v "_test.go" | grep -v ".pb.go"
```

Extract every module name string passed to these functions. Cross-reference against `maccPerms` in `app/app.go`.

### Step 2: Determine permissions
For each missing module, determine what permissions it needs:
- **nil** (empty): only receives/sends coins between module accounts
- **Burner**: calls `BurnCoins` (e.g., slashing deposits)
- **Minter**: calls `MintCoins` (e.g., creating new tokens)
- **Staking**: interacts with the staking pool

Read each module's keeper code to determine which bank operations it performs.

### Step 3: Add all missing entries
Add them to `maccPerms` in `app/app.go`. Add any needed imports.

Known missing (verify these + find any others):
```go
ontologytypes.ModuleName:     nil,      // verify if needs Burner
researchtypes.ModuleName:     nil,      // verify if needs Burner  
scheduletypes.ModuleName:     nil,      // verify if needs Burner
discoverytypes.ModuleName:    nil,      // verify if needs Burner
bvmtypes.ModuleName:          nil,      // verify if needs Burner
"treasury_protocol":          nil,      // verify actual module name source
```

For "treasury_protocol" — find where this string is defined. Is it a constant in a types package, or hardcoded? Use the proper constant if one exists.

### Step 4: Verify exhaustively
```bash
go build ./...
go test ./... -count=1 -timeout 600s
```

Then verify on devnet that previously-panicking operations work:
```bash
# Boot devnet
# Test each previously-broken module:
legbled tx ontology propose-domain ...
legbled tx research submit-research ...
legbled tx schedule create-schedule ...
legbled tx discovery register-agent ...
legbled tx bvm call-contract ... --amount 1000ulgm
```

### Step 5: Add a compile-time guard (optional but recommended)
Consider adding an init() function in app/app.go that validates all modules with bank keeper dependencies have corresponding maccPerms entries. This would catch future regressions at startup rather than at runtime.

## Report format
Write to reports/batch-7/SESSION-B7-1-MACCPERMS-FINAL.md:
- Complete audit table: module → bank operations → required permissions → was in maccPerms?
- Entries added
- Build + test output
- Devnet verification per module
- Whether compile-time guard was added
