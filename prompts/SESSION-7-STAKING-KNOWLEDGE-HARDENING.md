# Session 7: Staking + Knowledge Module Security Hardening

## Context
You are working on Legible Money (`/Users/yuai/Desktop/legible_money`), a Cosmos SDK blockchain with a 4-tier validator staking system (`x/staking`) and Proof of Truth knowledge verification (`x/knowledge`). These are the two most critical custom modules.

Read both audit reports before starting:
- `reports/STAKING-POT.md` — 4-tier validator system audit (3 P0, 2 P1, 2 P2)
- `reports/KNOWLEDGE-KEEPER.md` — Knowledge module keeper audit

## Scope
Files to modify:
- `x/staking/keeper/` (keeper files)
- `x/staking/types/` (as needed)
- `x/knowledge/keeper/` (keeper files, especially rounds.go, phases.go, confidence.go, validator_selection.go)
- `x/knowledge/crypto/vrf.go`
- Test files in both modules

## Tasks

### 1. Fix Staking P0 Issues
Read `reports/STAKING-POT.md` for the full findings. Key P0 areas:
- **Tier demotion race conditions**: If a validator's stake drops mid-verification-round, ensure proper handling
- **Unbonding completion with pending slashes**: Ensure unbonding tokens are reduced by pending slashes before release
- **VRF weight calculation overflow**: The virtual stake multiplication for tier weighting could overflow with large validator sets

### 2. Fix Knowledge Keeper Issues  
Read `reports/KNOWLEDGE-KEEPER.md` for findings. Key areas:
- **Round phase transition safety**: Ensure phase transitions (commit → reveal → finalize) cannot be skipped or replayed
- **Confidence calculation**: Review for floating-point or integer arithmetic issues in confidence scoring
- **Validator selection fairness**: VRF-based selection should be properly stake-weighted across all tiers

### 3. VRF Implementation Review
Review `x/knowledge/crypto/vrf.go`:
- Ensure VRF proofs are properly verified
- Ensure the selection threshold correctly accounts for the 4-tier virtual stake system
- Check for timing attacks or determinism issues

### 4. Test Coverage
- Add tests for tier boundary conditions (exact stake thresholds)
- Add tests for round phase transitions with edge cases
- Add tests for VRF selection distribution fairness

## Testing
- Run: `go test ./x/staking/... -count=1`
- Run: `go test ./x/knowledge/... -count=1`
- Run: `go build ./...`
- All must pass

## Report
Write a detailed report to `reports/SESSION-7-STAKING-KNOWLEDGE.md` covering:
- Each finding addressed (reference original report IDs)
- Before/after code for each fix
- Any findings you chose NOT to fix and why
- Test results
