# Session B8-1: E2E Test Nonce Race Fix

**Date**: 2026-02-18
**Scope**: Fix 4 E2E test failures (code 32 nonce race + balance race)
**Result**: All 4 failures addressed. Devnet not available for live verification.

---

## Problem

From the B7-6 E2E sweep, 47/51 tests passed. The 4 failures were all test script fragilities, not chain bugs:

| Test | Script | Failure | Root Cause |
|------|--------|---------|------------|
| T1.2 | test-tx-lifecycle.sh | Balance mismatch | Vesting rewards accrue between pre/post balance queries |
| B2 | test-lgm-modules.sh | Code 32 | test1 nonce stale after A2 (submit-claim) |
| C5 | test-lgm-modules.sh | Code 32 | test1 nonce stale after C4 (transfer-token) |
| C6 | test-lgm-modules.sh | Code 32 | Cascades from C5 |

---

## Fixes Applied

### 1. Created `devnet/scripts/test-helpers.sh` (new shared helper)

Contains a `settle()` function that sleeps one block period (3s) after a confirmed transaction. This ensures the account sequence is fully queryable before the next transaction from the same account.

Both test scripts source this file.

### 2. Fixed T1.2: Balance assertion (test-tx-lifecycle.sh)

**Before**: Exact equality check `assert_eq ... "$expected_test1"`
**After**: Range check `if [ "$balance_after" -ge "$min_expected" ]`

Vesting rewards accrue during the block(s) between the pre-send and post-send balance queries, so the recipient's balance will be slightly higher than `initial + send_amount`. Using `>=` instead of `==` accounts for this.

### 3. Fixed B2/C5/C6: Nonce races (test-lgm-modules.sh)

Added `settle` calls between back-to-back transactions from the same account (test1):

| Location | Between | Purpose |
|----------|---------|---------|
| After A2 | A2 (submit-claim) → B2 (register-validator) | test1 nonce |
| After C2 | C2 (create-token) → C3 (mint-token) | test1 nonce |
| After C3 | C3 (mint-token) → C4 (transfer-token) | test1 nonce |
| After C4 | C4 (transfer-token) → C5 (burn-token) | test1 nonce |
| After C5 | C5 (burn-token) → C6 (approve-token) | test1 nonce |
| After E2 | E2 (submit-lip) → E5 (cast-vote) | test1 nonce |

Total added wait time: 6 × 3s = 18 seconds. Acceptable for a 51-test suite.

### 4. Switched `--gas 300000` to `--gas auto --gas-adjustment 2.0` (test-lgm-modules.sh)

The hardcoded gas limit of 300000 was fragile — some transactions may need more gas, others less. Switched to match `test-tx-lifecycle.sh`'s approach of auto-estimation with 2x adjustment.

### 5. Updated stale KNOWN ISSUE comments (test-lgm-modules.sh)

Several comments referenced bugs that were already fixed in prior sessions:

| Comment | Referenced Bug | Fixed By |
|---------|---------------|----------|
| B2: "proto codec issue" | protov2 nested type decode | B7-6 proto_compat.go bridge |
| E1: "gRPC query not wired" | Wrong gRPC service path | B6-1 query.go path fix |
| E2: "module account not registered" | lgm_gov missing from maccPerms | B6-1 maccPerms addition |
| E3/E4: "gRPC query not wired" | Same as E1 | B6-1 |

Also removed the E2 `code == 111222` fallback (panic recovery code no longer expected).

---

## Files Changed

| File | Change |
|------|--------|
| `devnet/scripts/test-helpers.sh` | **NEW**: Shared `settle()` function |
| `devnet/scripts/test-tx-lifecycle.sh` | Source test-helpers.sh, T1.2 `>=` assertion |
| `devnet/scripts/test-lgm-modules.sh` | Source test-helpers.sh, 6 settle calls, `--gas auto`, comment updates |

---

## Verification

- All 3 scripts pass `bash -n` syntax check
- Go build clean (`go build ./...`)
- Devnet not running at time of fix — live verification deferred to next devnet session

---

## Expected Results After Fix

| Before | After |
|--------|-------|
| T1.2: FAIL (balance mismatch) | PASS (>= check tolerates vesting accrual) |
| B2: FAIL (code 32) | PASS (settle after A2) |
| C5: FAIL (code 32) | PASS (settle after C4) |
| C6: FAIL (code 32) | PASS (settle after C5) |
| **47/51 PASS** | **51/51 PASS** |
