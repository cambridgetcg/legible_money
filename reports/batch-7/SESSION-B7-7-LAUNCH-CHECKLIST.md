# Session B7-7: Final Launch Checklist

**Date**: 2026-02-18
**Branch**: main (c2835a1)
**Scope**: READ-ONLY verification + documentation

---

## Step 1: Verify Blocker Resolved

**RESULT: FAIL**

The B6-7 blocker (6 missing module accounts in `maccPerms`) is **NOT resolved**.

`app/app.go` maccPerms map still has **27 entries**. The following 6 modules are missing:

| Missing Module | ModuleName constant | Uses SendCoins* |
|---|---|---|
| ontology | `"ontology"` | `SendCoinsFromModuleToAccount` (domain proposal stake return) |
| research | `"research"` | `SendCoinsFromModuleToAccount` (bounty payouts) |
| schedule | `"schedule"` | `SendCoinsFromModuleToModule` (fee distribution) |
| discovery | `"discovery"` | `SendCoinsFromModuleToModule` (reward distribution) |
| bvm | `"bvm"` | `SendCoinsFromModuleToAccount` (contract payouts) |
| treasury_protocol | (literal string in billing) | `SendCoinsFromAccountToModule` (protocol treasury) |

**Impact**: Any transaction triggering these `SendCoins*` calls will panic at runtime. Tests pass because they use mock bank keepers that don't enforce maccPerms.

**Fix required**: Add 6 entries to `maccPerms` in `app/app.go:303-332` (~15 min).

---

## Step 2: Build + Test

### Build
```
$ make build
Building legbled... ✓
```
**RESULT: PASS**

### Go Vet
```
$ go vet ./...
(no output — clean)
```
**RESULT: PASS**

### Tests
```
$ go test ./... -count=1 -timeout 600s
37 packages OK, 0 failures
2818 test functions across 129 test files
Total test time: ~267s
```
**RESULT: PASS** — All 37 test packages pass, 0 failures.

---

## Step 3: Version Check

```
$ ./build/legbled version --long
name: legible
server_name: legbled
version: v0.1.0-347-gc2835a1-dirty
commit: c2835a1
cosmos_sdk_version: v0.50.15
go: go version go1.26.0 darwin/arm64
build_tags: netgo,ledger
```

**RESULT: PASS** — All fields populated via Makefile ldflags.

Note: `version` shows `-dirty` because `app/app.go` and `app/gas.go` have uncommitted changes. Clean build after committing will show clean tag.

---

## Step 4: Genesis Ceremony

```
$ bash scripts/testnet-genesis.sh init
  Genesis INIT complete.
  File at ~/.lgm/testnet/config/genesis.json is a valid genesis file ✓

$ bash scripts/testnet-genesis.sh add-validator val1
  Validator val1 added.
  Address: lgm1q9x9fvj780au8kl3950n0vqhh28avw2fykvw9c
  Stake: 10000000000 ulgm ✓

$ bash scripts/testnet-genesis.sh finalize
  Genesis FINALIZED.
  Validators: 1
  File at ~/.lgm/testnet/config/genesis.json is a valid genesis file ✓
```

**RESULT: PASS** — Full ceremony (init → add-validator → finalize) completes without errors. Genesis validates.

---

## Step 5: CHANGELOG.md

Created at `/CHANGELOG.md` in project root. See file for complete listing.

---

## Step 6: docs/TESTNET-LAUNCH.md Updates

The existing `docs/TESTNET-LAUNCH.md` (222 lines) is comprehensive and accurate. Corrections added from batch 6-7 findings:

1. **Account pubkey bootstrap**: First transaction from a new account must be sent to bootstrap the on-chain pubkey. Until then, queries like `legbled query auth account` return `null` pubkey.
2. **gRPC disabled note**: Already documented (item 2 in Known Limitations).
3. **Bootstrap gas-free period**: Already documented (item 6 in Known Limitations) — first 480,000 blocks for PoT transactions.
4. **Build command**: Should use `make build` (not bare `go build`) to inject version ldflags.
5. **Go version**: Updated from 1.22+ to 1.23+ (matches go.mod requirement).
6. **maccPerms blocker**: Added note that 6 module accounts must be registered before launch.

---

## Step 7: Final Git Status

```
$ git status
On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  modified:   app/app.go
  modified:   app/gas.go
```

```
$ git log --oneline -10
c2835a1 chore: add session prompts, batch-7 report, gitignore devnet-data
ce76b2a chore(build): set binary version via ldflags
4668649 chore(reports): batch-6 session reports + ping-pub explorer config
d569f5f feat(genesis): add production testnet genesis ceremony
7b56665 fix(e2e): fix event type filters in test-lgm-modules.sh
1383b12 feat(knowledge): proto-migrate ChallengeProvisionalFact + UpdateExtendedParams
72d8cce feat(tokens): add --mintable/--burnable/--pausable/--wrappable CLI flags
73dda01 fix(staking,gov): wire CLI commands and fix gRPC query paths
d6f20d4 fix(app): add missing module account permissions
613e569 fix: gov keeper test pointer fix + R5-4 vet cleanup report
```

Note: Commit `d6f20d4` says "add missing module account permissions" but the 6 modules identified in B6-7 are still not in maccPerms. That commit may have addressed a different set of missing permissions.

---

## Summary Table

| Check | Result | Notes |
|-------|--------|-------|
| Blocker resolved | **FAIL** | 6 modules still missing from maccPerms |
| Build | PASS | `make build` succeeds |
| Go vet | PASS | 0 warnings |
| Tests | PASS | 37 packages, 2818 tests, 0 failures |
| Version metadata | PASS | name/server_name/version/commit all set |
| Genesis ceremony | PASS | init + add-validator + finalize all succeed |
| CHANGELOG.md | CREATED | At project root |
| TESTNET-LAUNCH.md | UPDATED | Corrections from B6-7 findings |
| Git status | CLEAN | On main, 2 unstaged files |

---

## VERDICT: NOT READY FOR TESTNET

**Reason**: The maccPerms blocker from B6-7 is not resolved. Six module accounts (ontology, research, schedule, discovery, bvm, treasury_protocol) are not registered in `maccPerms`. Any runtime transaction that triggers `SendCoinsFromModuleToAccount` or `SendCoinsFromAccountToModule` for these modules will **panic** the chain.

**To become ready**: Add the 6 missing entries to `maccPerms` in `app/app.go`, rebuild, re-test, and verify. Estimated effort: ~15 minutes.

**Everything else passes**: build, vet, all 2818 tests, version metadata, genesis ceremony. The chain is functionally complete pending this one configuration fix.
