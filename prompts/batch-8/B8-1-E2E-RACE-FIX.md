# B8-1: Fix E2E Test Nonce Race Conditions

## Context
The E2E test scripts (`devnet/scripts/test-tx-lifecycle.sh` and `devnet/scripts/test-lgm-modules.sh`) have 4 failures out of 51 tests. All 4 failures are code 32 (account sequence mismatch) — nonce races from rapid-fire tx submissions, NOT chain bugs.

## Problem
Tests submit transactions back-to-back without waiting for the previous tx to be included in a block. When two txs land in the same block, the second one has a stale sequence number.

## Task

### 1. Read and understand the failing tests
Read `reports/batch-7/SESSION-B7-6-E2E-FULL-SWEEP.md` for the specific failing test IDs and error patterns.

Read both test scripts:
- `devnet/scripts/test-tx-lifecycle.sh`
- `devnet/scripts/test-lgm-modules.sh`

### 2. Add a tx-wait helper
Create or enhance a helper function that:
- Submits a tx
- Extracts the txhash from output
- Polls `legbled query tx <hash>` until confirmed (or timeout)
- Returns the result

This should be a shared function both scripts can source.

### 3. Apply the helper to race-prone sequences
Identify all places where tests submit multiple txs in sequence and add waits between them. Focus on:
- Tests that use the same account for consecutive txs
- Tests where the second tx depends on state from the first

### 4. Don't over-engineer
- Simple polling with sleep + retry is fine
- 10-second timeout per tx is generous for single-validator devnet
- Don't add waits between independent tests (different accounts)

### 5. Verify
Run both scripts against a local devnet and confirm all 51 tests pass.

## File Ownership
- `devnet/scripts/test-tx-lifecycle.sh`
- `devnet/scripts/test-lgm-modules.sh`
- `devnet/scripts/test-helpers.sh` (new, if created)

## Report
Write `reports/batch-8/SESSION-B8-1-E2E-RACE-FIX.md` with before/after results.
