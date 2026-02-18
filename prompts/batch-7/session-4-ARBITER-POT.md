# Session: ARBITER-POT — Test vote extensions via arbiter-path challenge

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Trigger and verify the arbiter-path PoT resolution using vote extensions

### Context
B6-6 verified the deterministic path works end-to-end. But vote extensions (the core ABCI++ integration) only activate for arbiter-path challenges. The arbiter path has never been tested on a live network.

From `app/vote_extensions.go:76`:
```go
if adversarialEnabled && round.Type != knowledgetypes.RoundTypeChallenge {
    continue
}
```

Vote extensions trigger when `round.Type == RoundTypeChallenge`. We need to understand what creates an arbiter-path challenge vs. a deterministic one.

### Step 1: Understand resolution path selection
Read these files to understand how the resolution path is chosen:
1. `x/knowledge/keeper/challenge.go` — where challenges are created
2. `x/knowledge/types/types.go` — RoundTypeChallenge and related types
3. `x/knowledge/keeper/phases.go` — how round type affects phase processing

Key questions:
- What makes a challenge use arbiter (vote extensions) vs deterministic resolution?
- Is it the epistemic category? The domain? A param?
- Can we configure a domain to use arbiter resolution for testing?

### Step 2: Configure for arbiter resolution
Based on Step 1, configure the devnet genesis or submit governance transactions to create conditions where a challenge uses arbiter resolution.

Possibilities:
- Create a domain with arbiter resolution enabled
- Submit an empirical claim (might default to arbiter)
- Configure params to force arbiter mode

### Step 3: Boot 3-validator Docker devnet
Use the existing Docker setup from B6-4/B6-6. Ensure:
- 3 validators
- Vote extensions enabled at height 1
- min_verifiers ≤ 3

### Step 4: Submit claim → challenge → trigger arbiter path
1. Submit a claim in the arbiter-configured domain
2. Wait for provisional fact
3. Challenge the provisional fact
4. Monitor for vote extension activity:
   ```bash
   docker logs lgm-devnet-validator-1 2>&1 | grep -i "extend\|commit\|reveal\|vrf\|selected"
   docker logs lgm-devnet-validator-2 2>&1 | grep -i "extend\|commit\|reveal\|vrf\|selected"
   ```

### Step 5: Verify the full arbiter cycle
- Block N: Challenge submitted, round created with type=Challenge
- Block N+1 to N+commit_duration: Validators extend votes with commitments
- VerifyVoteExtension validates each commitment
- PrepareProposal collects extensions into injection tx
- PreBlocker processes injection → stores commitments
- Block N+commit+1 to N+commit+reveal: Reveal phase via vote extensions
- Aggregation → verdict → resolution

### Step 6: Verify commit/reveal integrity
The batch-1 P0 fixes should be active:
- No false equivocation (multi-block commitment reuse)
- Reveal hash verified against commitment
- No proposer tampering possible

## Report format
Write to reports/batch-7/SESSION-B7-4-ARBITER-POT.md:
- How arbiter path is triggered (what conditions)
- Configuration applied
- Block-by-block lifecycle of arbiter resolution
- Vote extension logs from all validators
- Commit/reveal data from vote extensions
- Final resolution result
- If arbiter path doesn't trigger: exact reason and what code change is needed
