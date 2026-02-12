# PP Review: Vote Extensions (PoT Consensus)

> **Review Date**: 2026-02-10
> **Files Reviewed**: `app/vote_extensions.go`, `app/abci.go`, `app/local_state.go`, `x/knowledge/keeper/rounds.go`, `x/knowledge/keeper/validator_selection.go`, `x/knowledge/keeper/phases.go`, `x/knowledge/keeper/confidence.go`, `x/knowledge/crypto/vrf.go`
> **Scope**: Full ABCI++ vote extension pipeline for Proof of Truth consensus

---

## Architecture Summary

```
                         PoT Vote Extension Data Flow
                         ═══════════════════════════

  ┌─────────────┐       ┌──────────────────┐       ┌─────────────────┐
  │  ExtendVote │──────▶│  VerifyVoteExt   │──────▶│PrepareProposal  │
  │  (per-val)  │       │  (per-val)       │       │  (proposer)     │
  │             │       │                  │       │                 │
  │ VRF select  │       │ VRF proof check  │       │ Collect exts    │
  │ Commit/Rev  │       │ Commitment fmt   │       │ Sort determ.    │
  │ Local salt  │       │ Reveal→commit    │       │ Inject as tx[0] │
  └─────────────┘       └──────────────────┘       └────────┬────────┘
                                                             │
  ┌─────────────┐       ┌──────────────────┐                │
  │ BeginBlocker│◀──────│    PreBlocker    │◀───────────────┘
  │  (x/knowl)  │       │                  │       ┌─────────────────┐
  │             │       │ Decode injection │◀──────│ProcessProposal  │
  │ Phase trans │       │ Store commit/rev │       │  (all vals)     │
  │ Aggregation │       │ in keeper state  │       │ Format check    │
  │ Complete    │       └──────────────────┘       └─────────────────┘
  └─────────────┘
```

---

## TRUTH GATES (6 gates)

### T1: Grounding — PASS (with caveats)

Every claim in this review is grounded in specific file:line citations from the read files. Key facts verified:

| Claim | Evidence |
|-------|----------|
| VRF uses Ed25519 ECVRF | `x/knowledge/crypto/vrf.go:1-8` (RFC 9381 simplified) |
| Commit-reveal via vote extensions | `app/vote_extensions.go:59-80` (commit/reveal switch) |
| ABCI++ integration | `app/abci.go:1-19` (mapping comments), `app/app.go:992-993` (handler wiring) |
| Deterministic injection ordering | `app/abci.go:347-359` (sort.Slice by RoundID then Validator) |
| PreBlocker stores data before BeginBlocker | `app/app.go:975-976`, `app/abci.go:367` |

**Caveats**: The VRF implementation states "RFC 9381 simplified" (`vrf.go:8`) — the exact deviations from the standard are not documented.

### T2: Calibration — PASS

Confidence is appropriate to evidence:
- **High confidence**: Data flow architecture is well-structured and verified across 8 files
- **Medium confidence**: VRF cryptographic correctness (simplified, not fully RFC-compliant)
- **Low confidence**: Production security posture (stub verdict, no test coverage)

### T3: Counter-Consideration — PASS

Counterarguments and edge cases are identified below in the findings section. Key counter-considerations:
- What if a proposer injects fake commitments? (partially mitigated by CometBFT consensus)
- What if VRF selection drifts across blocks? (design concern identified)
- What if hash collisions in commitment scheme? (second-preimage risk identified)

### T4: Completeness — PASS

Full pipeline reviewed from ExtendVote through aggregation and round completion. No major components omitted.

### T5: Representation — PASS

Findings rated as CRITICAL/HIGH/MEDIUM/LOW with explicit confidence levels.

### T6 (Claude): Citation — PASS

Every claim cites specific `file:line`. All files were read via the Read tool before analysis.

---

## UNDERSTANDING GATES (6 gates)

### U1: Truth Foundation — PASS

Built on verified code readings, not assumptions.

### U2: Mental Model — PASS

The architecture diagram above shows the complete data flow with component relationships. The model captures:
- Per-validator operations (ExtendVote, VerifyVoteExtension)
- Proposer-only operations (PrepareProposal injection)
- All-validator operations (ProcessProposal, PreBlocker, BeginBlocker)
- The crucial ordering: PreBlocker → BeginBlocker (data stored before phase transitions)

### U3: Why Explanation — PASS

Key "why" questions answered:
- **Why vote extensions?** CometBFT's single-block cycle doesn't natively support multi-block commit-reveal rounds. Vote extensions bridge this by piggybacking PoT data on validator votes (`app/abci.go:12-15`).
- **Why injection pseudo-tx?** Vote extensions are per-validator; the injection tx creates a single deterministic representation that all validators process identically (`app/abci.go:73-76`).
- **Why PreBlocker not BeginBlocker?** Commitments/reveals must be in keeper state BEFORE the knowledge module's BeginBlocker advances phases (`app/app.go:1039-1042`).
- **Why local commitment store?** Salts from commit phase must survive to reveal phase but should never leave the validator node (`app/local_state.go:8-10`).

### U4: Knowledge Connection — PASS

Connected to: Cosmos SDK ABCI++ lifecycle, CometBFT consensus, commit-reveal cryptographic schemes, VRF-based leader selection (Algorand-like).

### U5: Practical Application — PASS

Findings include specific file:line locations and remediation paths.

### U6 (Claude): Exhaustion — PASS

Searched across `app/`, `x/knowledge/keeper/`, `x/knowledge/crypto/`, `x/knowledge/types/` for all related patterns. Confirmed no test files exist for vote extensions (`*_test.go` search for `VoteExtension|ExtendVote` returned 0 results).

---

## BEAUTY GATES (8 gates)

### B1: Foundation — PASS

Grounded in verified code.

### B2: Emergence — PASS

The architecture naturally follows from the problem constraint: CometBFT's block lifecycle doesn't support multi-round verification. Vote extensions emerge as the natural ABCI++ solution. The injection-tx pattern is the standard Cosmos SDK approach for this.

### B3: Fit — PASS (with concern)

Fits the Cosmos SDK ABCI++ model well. The one concern: using JSON for serialization (`app/abci.go:37`, `app/vote_extensions.go:87`) instead of protobuf. The code acknowledges this: "serialized as JSON for Phase 2; Phase 3 will use protobuf" (`app/abci.go:37`).

### B4: Elegance — MEDIUM CONCERN

**Extra elements**: The `VoteExtension` struct has both per-extension `VRFOutput/VRFProof` fields (`app/abci.go:47-48`, `[]byte`) AND per-commitment `VRFOutput/VRFProof` (`app/abci.go:57-58`, `string`). The top-level fields appear unused — the actual VRF data flows through commitments. This is mild clutter.

**Missing elements**: No `Justification` propagation. `VoteReveal.Justification` exists (`app/abci.go:68`) but is never populated in `handleRevealPhase` (`app/vote_extensions.go:203-208`).

### B5: Coherence — PASS

The pipeline hangs together as a unified whole. PreBlocker → BeginBlocker ordering is explicitly maintained. Type flow is consistent from `VoteExtension` → `VoteExtInjection` → keeper types.

### B6: Clarity — PASS

Excellent inline documentation. Every handler has doc comments explaining its role. The mapping comment in `app/abci.go:8-15` clearly bridges TS prototype concepts to ABCI++ equivalents. `ComputeCommitmentHash` is straightforward and verifiable.

### B7: Appropriateness — PASS

Right-sized for Phase 2/MVC. Uses JSON (acknowledged for Phase 3 upgrade), in-memory local store (acknowledged), stub verdict (acknowledged). Complexity matches the current maturity level.

### B8 (Claude): Conformance — PASS

Follows Cosmos SDK patterns: `sdk.ExtendVoteHandler` / `sdk.VerifyVoteExtensionHandler` function signatures, `sdk.Context` usage, `EventManager().EmitEvent()` pattern, keeper method delegation.

---

## JUSTICE GATES (7 gates)

### J1: Placement — PASS

| Component | Location | Justification |
|-----------|----------|---------------|
| Vote extension handlers | `app/vote_extensions.go` | App-level ABCI++ handlers, alongside `app/abci.go` |
| Vote types + injection | `app/abci.go` | App-level ABCI types, not module-specific |
| Local commitment store | `app/local_state.go` | Node-local state, not on-chain |
| Round management | `x/knowledge/keeper/rounds.go` | Knowledge module owns verification rounds |
| VRF selection | `x/knowledge/keeper/validator_selection.go` | Knowledge module owns validator selection |
| Phase transitions | `x/knowledge/keeper/phases.go` | Knowledge module owns round lifecycle |
| Confidence aggregation | `x/knowledge/keeper/confidence.go` | Knowledge module owns verification results |
| VRF crypto | `x/knowledge/crypto/vrf.go` | Crypto primitives under knowledge module |

All placements are natural and discoverable.

### J2: Ownership — PASS

- App layer owns ABCI++ handlers and injection mechanics
- Knowledge keeper owns round lifecycle, aggregation, rewards
- Staking keeper owns stake data (accessed via cross-module interface)
- No feature envy detected

### J3: Responsibility — PASS

Appropriate abstraction levels:
- ExtendVote: local validator decision (VRF check, verdict, salt generation)
- VerifyVoteExtension: cryptographic validation (VRF proof, commitment format)
- PrepareProposal: deterministic data aggregation
- ProcessProposal: format validation
- PreBlocker: state storage bridge
- BeginBlocker: phase transitions and aggregation

### J4: Proportion — PASS

Complexity is concentrated in the right places:
- VRF crypto in `crypto/vrf.go` (highest complexity, well-encapsulated)
- Aggregation logic in `confidence.go` (stake-weighted math, stratum ceilings)
- Simple pipeline code in handlers (low complexity, delegation to keeper)

### J5: Balance — PASS

Follows existing module patterns. Consistent with knowledge keeper's other CRUD operations.

### J6: Integrity — PASS

Layer boundaries respected:
- App layer → keeper via typed interfaces
- Knowledge keeper → staking keeper via `StakingKeeper` interface
- No direct state access from app layer (all via keeper methods)

### J7 (Claude): Evidence — PASS

All placement decisions backed by evidence from file reads and pattern analysis.

---

## CREATIVITY GATES (7 gates)

### C0: Territory — CHARTED (Justice mode)

Vote extensions via ABCI++ is an established Cosmos SDK pattern (Skip Protocol, Oracle modules, etc.). This is charted territory adapted for PoT-specific needs.

### C1: Pioneering — N/A (territory is charted)

The PoT-specific adaptation (commit-reveal for knowledge verification via vote extensions) is novel for this codebase, but the ABCI++ pattern itself is well-established.

### C2: Path-Building — PASS

The structure is well-constructed. Others can follow: each handler is documented, the data flow is linear and traceable, the injection format is explicit.

### C3: Extension — PASS

Coherently extends the knowledge module. Rounds created in `msg_server.go` (via `MsgSubmitClaim`) flow naturally to vote extensions for commit-reveal, then back to keeper for aggregation.

### C4: Vision — PASS

Phase roadmap is explicit: Phase 2 (current, JSON + in-memory + stub) → Phase 3 (protobuf + persistent store + real evaluation). Each TODO is marked with "Phase 3".

### C5: Foundation — PASS

The injection pattern, round lifecycle, and VRF selection are solid foundations for Phase 3 upgrades. The interfaces are clean enough to swap implementations.

### C6: Integration — PASS

Well-connected: wired in `app/app.go:989-993`, PreBlocker registered at `app/app.go:975`, knowledge keeper BeginBlocker processes stored data.

### C7 (Claude): Proof — N/A (territory is charted)

---

## FINDINGS

### CRITICAL

#### F1: Stub Verdict — All Claims Rubber-Stamped

**Location**: `app/vote_extensions.go:146-161`
**Status**: Known (marked with TODO)

```go
// TODO(Phase 3): Replace stub with real claim evaluation.
verdict := "accept"
confidence := uint64(800000) // 0.8
```

Every claim submitted to the network gets "accept" at 0.8 confidence from every selected validator. The entire Proof of Truth mechanism is non-functional for actual truth verification. This is the single largest gap before any real deployment.

**Impact**: No claim can be rejected. The "Proof of Truth" consensus currently proves nothing.

**Remediation**: Phase 3 must implement real claim evaluation. The TODO outlines the architecture: claim content retrieval, ontology evaluation, external oracle query.

---

### HIGH

#### F2: Commitment Hash Lacks Second-Preimage Resistance

**Location**: `app/abci.go:441-450`

```go
h.Write([]byte(roundID))
h.Write([]byte(verdict))
confBytes := make([]byte, 8)
binary.BigEndian.PutUint64(confBytes, confidence)
h.Write(confBytes)
h.Write([]byte(salt))
```

Fields are concatenated without length prefixes or delimiters. This creates ambiguity:
- `roundID="abc"` + `verdict="def"` hashes identically to `roundID="abcde"` + `verdict="f"`

While the confidence field (fixed 8 bytes) partially breaks some collisions, a roundID that ends with a valid confidence encoding could still collide.

**Remediation**: Use length-prefixed encoding or explicit delimiters:
```go
h.Write([]byte(fmt.Sprintf("%d:", len(roundID))))
h.Write([]byte(roundID))
h.Write([]byte(fmt.Sprintf("%d:", len(verdict))))
h.Write([]byte(verdict))
// ... etc
```

Or use domain separation: `SHA-256("LGM.commitment.v1:" || roundID || ":" || verdict || ":" || confidence || ":" || salt)`

#### F3: ProcessProposal Does Not Verify Injection Authenticity

**Location**: `app/abci.go:243-252`

```go
if isVoteExtInjectionTx(txBytes) {
    if _, err := decodeVoteExtInjection(txBytes); err != nil {
        // REJECT
    }
    continue // ← only format check, no content verification
}
```

ProcessProposal only validates that the injection tx is well-formed JSON. It does NOT verify that the injected commitments/reveals actually came from authenticated vote extensions. A malicious proposer could:
1. Add fabricated commitments for rounds they weren't selected for
2. Modify commitment hashes from real extensions
3. Include commitments from extensions that were rejected by VerifyVoteExtension

CometBFT guarantees >2/3 validators agree on block contents (so a rejected proposal doesn't finalize), but a proposer controls what goes INTO the proposal.

**Mitigation note**: In practice, CometBFT requires >2/3 prevotes on the proposal, so fabricated data would need to pass ProcessProposal on >2/3 of validators. Since ProcessProposal only checks format, the fake data WOULD pass. The defense is that `StoreCommitmentInRound` (`rounds.go:100-134`) has its own guards (phase check, duplicate check), and VRF proofs in the commitment are verified during `VerifyVoteExtension` but NOT re-verified in ProcessVoteExtInjection.

**Remediation**: Either:
1. Re-verify VRF proofs in `ProcessVoteExtInjection` (expensive but thorough)
2. Include a Merkle root of the injected data in the block header
3. In newer Cosmos SDK versions, use `req.ProposedLastCommit` in ProcessProposal to cross-reference

#### F4: Zero Test Coverage for Vote Extensions

**Evidence**: `grep -r "VoteExtension|ExtendVote" *_test.go` → 0 results

The entire vote extension pipeline — ExtendVote, VerifyVoteExtension, PrepareProposal injection, ProcessProposal validation, PreBlocker storage, commitment hashing — has **no test coverage**.

**Remediation**: Priority test cases:
1. `TestExtendVote_CommitPhase` — VRF selection, commitment generation, local salt storage
2. `TestExtendVote_RevealPhase` — local commitment retrieval, reveal format
3. `TestExtendVote_NotSelected` — VRF rejection path
4. `TestVerifyVoteExtension_ValidCommitment` — VRF proof verification
5. `TestVerifyVoteExtension_InvalidVRF` — reject bad VRF proof
6. `TestVerifyVoteExtension_RevealMismatch` — reject reveal that doesn't match commitment
7. `TestPrepareProposal_InjectionOrder` — deterministic sort verification
8. `TestProcessVoteExtInjection_StoresCommitments` — round state updated
9. `TestComputeCommitmentHash_Deterministic` — same inputs → same hash
10. `TestLocalCommitmentStore_ThreadSafety` — concurrent read/write

---

### MEDIUM

#### F5: VRF Seed Varies Per Block (Design Concern)

**Location**: `app/vote_extensions.go:112` + `x/knowledge/keeper/validator_selection.go:86-87`

The VRF seed includes `prevBlockHash` from the current context:
```go
seed = SHA-256("LGM.vrf.seed.v1:" || claimID || ":" || startHeight || ":" || prevBlockHash)
```

Since `prevBlockHash` changes every block, a validator generates a DIFFERENT VRF output each block during the commit phase window. This means:
- A validator unselected at block H may be selected at H+1
- Selection is not fixed at round creation
- The set of "selected validators" is not predetermined

**Is this intentional?** It provides more validators the chance to participate across the commit window, which increases participation. But it also means:
- An attacker observing VRF outputs could strategically choose WHICH block to submit
- The `SelectedValidators` list on the round grows dynamically as commitments arrive

**Remediation (if unintentional)**: Fix the prevBlockHash to the block at round creation height, stored in the round itself:
```go
round.VRFSeedBlockHash = prevBlockHash  // set at CreateVerificationRound
```

#### F6: LocalCommitmentStore In-Memory Only

**Location**: `app/local_state.go:8-10`

```go
// This is in-memory only for Phase 2. If the validator restarts between commit
// and reveal phases, they forfeit pending rewards.
```

Known limitation, documented. A validator restart during the commit window loses all pending reveals. On a 2521ms block time with typical commit phase of ~20 blocks (~50 seconds), this risk is modest. But for longer commit phases or unreliable validators, it becomes significant.

**Impact**: Missed reveals trigger a 2% stake slash (`confidence.go:230`).

#### F7: Unused Top-Level VRF Fields on VoteExtension

**Location**: `app/abci.go:46-48`

```go
type VoteExtension struct {
    // ...
    VRFOutput []byte `json:"vrf_output,omitempty"`   // ← unused
    VRFProof  []byte `json:"vrf_proof,omitempty"`    // ← unused
    // ...
}
```

These `[]byte` fields exist at the VoteExtension level, but VRF data actually flows through per-commitment `VRFOutput`/`VRFProof` (`string` type, hex-encoded). The top-level fields are dead code that adds confusion.

#### F8: Stake Lookup Uses Different Functions

**Location**: `app/vote_extensions.go:126-128` vs `x/knowledge/keeper/validator_selection.go:101`

In `handleCommitPhase`:
```go
if vi, found := keeper.GetStakingKeeper().GetValidatorInfo(ctx, config.ValidatorAddress); found {
    effectiveStake = vi.Stake
}
```

In `VerifyValidatorVRFSelection`:
```go
effectiveStake := k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)
```

The commit phase uses `vi.Stake` (raw stake) while verification uses `GetEffectiveStake` (which may include tier adjustments, virtual stake, etc.). If these return different values, a validator could generate a VRF that passes local selection but fails remote verification (or vice versa).

**Remediation**: Use `GetEffectiveStake` consistently in both paths.

#### F9: No Rate Limiting on Vote Extension Size

**Location**: `app/vote_extensions.go:40-100`

A validator can include unlimited commitments and reveals in a single vote extension. There's no cap on `len(ext.Commitments)` or `len(ext.Reveals)`. A malicious validator could bloat their vote extension with thousands of entries.

`VerifyVoteExtensionHandler` (`app/vote_extensions.go:222-309`) iterates all commitments and reveals without bounds. For each commitment with VRF proof, it performs VRF verification (`validator_selection.go:51-120`) which involves curve operations.

**Remediation**: Add bounds in VerifyVoteExtension:
```go
const MaxCommitmentsPerExtension = 50
const MaxRevealsPerExtension = 50
if len(ext.Commitments) > MaxCommitmentsPerExtension { return rejectExtension() }
```

---

### LOW

#### F10: JSON Serialization for Consensus-Critical Data

**Location**: `app/abci.go:37`, `app/vote_extensions.go:87`

JSON encoding is non-deterministic by default in Go (map iteration order). The vote extension itself uses `json.Marshal` which IS deterministic for structs (field order is fixed). However, `VoteExtension` contains no maps, so this is safe. The `VerificationRound` type DOES have map fields (`Commitments map[string]...`), but these are serialized at the keeper level, not in vote extensions.

**Note**: Explicitly marked for Phase 3 upgrade to protobuf.

#### F11: CleanupExpired Called Every Block

**Location**: `app/vote_extensions.go:84`

```go
config.LocalStore.CleanupExpired(currentHeight, 100)
```

`CleanupExpired` iterates all stored commitments every block. With `maxAge=100`, commitments older than 100 blocks (~4 minutes) are purged. This is O(n) per block where n is stored commitments. Fine for small n, but if many rounds are active simultaneously, this adds latency to ExtendVote.

**Remediation**: Only clean up every N blocks, or use an ordered data structure.

#### F12: `handleRevealPhase` Doesn't Verify Reveal Matches Commitment

**Location**: `app/vote_extensions.go:192-209`

The local `handleRevealPhase` creates a reveal from the stored local commitment but doesn't verify that the reveal would actually match the commitment hash. If the local store was corrupted, the reveal would be sent but rejected by other validators.

This is LOW risk because: (1) the local store is simple in-memory map with no corruption vector, (2) `VerifyVoteExtensionHandler` catches mismatches.

---

## AGGREGATE ASSESSMENT

| Dimension | Status | Score |
|-----------|--------|-------|
| **Truth** | PASS | 6/6 gates |
| **Understanding** | PASS | 6/6 gates |
| **Beauty** | PASS (with concerns) | 7/8 gates (Elegance: medium concern) |
| **Justice** | PASS | 7/7 gates |
| **Creativity** | PASS (charted territory) | 6/7 gates (Territory: justice mode) |

**Overall**: 32/34 gates PASS. Architecture is sound, well-placed, well-documented, and follows established Cosmos SDK patterns. The primary concerns are:
1. **Stub verdict** (Critical, known, Phase 3) — the core mechanism doesn't actually verify truth
2. **Commitment hash second-preimage** (High) — fixable with domain separation
3. **Injection authenticity** (High) — ProcessProposal trusts proposer
4. **Zero test coverage** (High) — entire pipeline untested
5. **Stake function mismatch** (Medium) — `Stake` vs `EffectiveStake`

---

## RECOMMENDED ACTIONS (Priority Order)

1. **Add test suite** (F4) — highest ROI, catches regressions, enables safe refactoring
2. **Fix commitment hash** (F2) — simple change, prevents subtle cryptographic bugs
3. **Unify stake lookup** (F8) — use `GetEffectiveStake` in ExtendVote
4. **Add extension size limits** (F9) — simple bounds check
5. **Remove unused VRF fields** (F7) — code hygiene
6. **Plan Phase 3 verdict system** (F1) — design the oracle/evaluation interface
7. **Evaluate injection verification** (F3) — assess ProcessProposal trust model
8. **Decide VRF seed policy** (F5) — fixed vs per-block selection

---

## FIXES APPLIED (2026-02-10)

### F2: FIXED — Commitment Hash Second-Preimage Resistance
**File**: `app/abci.go:438-460`
- Added `"LGM.commitment.v1:"` domain separation prefix
- All variable-length fields (roundID, verdict, salt) now use 4-byte big-endian length prefix before content
- Confidence uses fixed 8-byte big-endian encoding (unchanged)
- Prevents field-boundary ambiguity (e.g. `roundID="abc"+verdict="def"` vs `roundID="abcde"+verdict="f"`)

### F4: FIXED — Test Suite Added
**File**: `app/vote_extensions_test.go` (new, 753 lines)
- 40+ test functions covering:
  - `ComputeCommitmentHash`: determinism, length, per-field variation, second-preimage resistance, edge cases (empty, max uint64, special chars, long salt, all verdicts)
  - `VerifyCommitmentHash`: valid + wrong field per dimension
  - `VoteExtInjection`: encode/decode roundtrip, empty, invalid format, prefix validation
  - `VoteExtension JSON`: roundtrip, empty
  - `LocalCommitmentStore`: store/get, not found, delete, overwrite, count, cleanup expired (3 scenarios), thread safety
  - `Injection sorting`: deterministic (RoundID, Validator) order
  - `VoteReveal/VoteCommitment`: valid verdicts, confidence bounds, hash length
- **Note**: Tests parse cleanly (`gofmt -e` reports 0 errors) but cannot be executed via `go test` due to pre-existing proto redeclaration conflicts in upstream modules (x/auth/types, x/bvm/types, x/channels/types)

### F7: FIXED — Removed Unused VRF Fields
**File**: `app/abci.go:38-47`
- Removed `VRFOutput []byte` and `VRFProof []byte` from `VoteExtension` struct
- VRF data flows through per-commitment fields (`VoteCommitment.VRFOutput/VRFProof` as hex strings)
- Verified 0 references to top-level `ext.VRFOutput` or `ext.VRFProof` in codebase

### F8: FIXED — Unified Stake Lookup
**File**: `app/vote_extensions.go:123-128`
- Changed from `vi.Stake` (raw) to `keeper.GetStakingKeeper().GetEffectiveStake(ctx, config.ValidatorAddress)`
- Now matches `VerifyValidatorVRFSelection` which uses `k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)`
- Also uses `keeper.GetTotalStake(ctx)` for total (consistent with verification path)

### F9: FIXED — Extension Size Limits
**File**: `app/vote_extensions.go:236-247`
- Added `MaxCommitmentsPerExtension = 50` and `MaxRevealsPerExtension = 50`
- Checked in `VerifyVoteExtensionHandler` before VRF proof iteration
- Oversized extensions rejected with warning log

### Remaining (Not Fixed)
- **F1** (CRITICAL): Stub verdict — Phase 3 scope, requires oracle/evaluation architecture
- **F3** (HIGH): Injection authenticity — requires design decision (re-verify VRF vs Merkle root vs ProposedLastCommit)
- **F5** (MEDIUM): VRF seed policy — design decision (fixed vs per-block selection)
- **F6** (MEDIUM): In-memory local store — Phase 3 scope, known limitation
- **F10** (LOW): JSON serialization — Phase 3 protobuf migration
- **F11** (LOW): CleanupExpired every block — acceptable for Phase 2
- **F12** (LOW): No local reveal verification — low risk, caught by remote validators
