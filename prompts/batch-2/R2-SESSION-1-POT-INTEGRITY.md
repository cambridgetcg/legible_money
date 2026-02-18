# R2 Session 1: Proof of Truth Integrity — End-to-End Hardening

## Context
Round 1 revealed the most dangerous finding in the codebase: a block proposer could tamper with reveal data (S6), and the commitment hash verification was missing. S6 fixed `StoreRevealInRound`, but this exposed a deeper question: **is the entire commit-reveal-finalize pipeline actually tamper-proof end-to-end?**

Session 7 confirmed VRF implementation is sound. Session 6 added commitment hash verification. But we need to verify the FULL data flow is airtight.

## Scope
- `x/knowledge/keeper/rounds.go` — commit/reveal/finalize logic
- `x/knowledge/keeper/phases.go` — phase transition + aggregation
- `x/knowledge/keeper/confidence.go` — reward/slash calculation
- `x/knowledge/keeper/validator_selection.go` — VRF selection
- `x/knowledge/types/commitment.go` — canonical hashing (from S6)
- `app/vote_extensions.go` — ExtendVote/VerifyVoteExtension
- `app/abci.go` — PrepareProposal VEX injection

## Tasks

### 1. Trace the Full Data Flow
Map the complete lifecycle of a single verification vote from ExtendVote → PrepareProposal → ProcessProposal → FinalizeBlock → StoreCommitment → StoreReveal → Aggregation → Rewards/Slashing. Document every trust boundary and what guarantees integrity at each step.

### 2. Verify Anti-Tampering at Each Boundary
For each handoff point, verify:
- Can a malicious PROPOSER alter the data? (They construct the block)
- Can a malicious VALIDATOR inject fake votes?
- Can a malicious RELAYER replay old data?
- Is there domain separation preventing cross-round/cross-claim confusion?

### 3. Fix Any Remaining Gaps
Round 1 found that proposers could tamper reveals. Check for similar issues:
- Can proposers OMIT specific votes (censorship)?
- Can proposers DUPLICATE votes?
- Can proposers reorder votes to change aggregation outcomes?
- Are there any remaining places where data is trusted without verification?

### 4. Add Integration Tests
Write tests that simulate the full lifecycle:
- Happy path: honest commit → reveal → finalize → rewards
- Adversarial: proposer omits a vote → what happens?
- Adversarial: proposer includes a vote from a non-selected validator
- Adversarial: commitment from round A replayed in round B

## Report
Write to `reports/R2-SESSION-1-POT-INTEGRITY.md`
