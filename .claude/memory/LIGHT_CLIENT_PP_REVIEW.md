# PP Review: Light Client & Cosmos SDK Setup

> Full 30-gate review across TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY

**Date**: 2026-02-10
**Scope**: Light client architecture, IBC infrastructure, proof generation, ABCI++ integration
**Files examined**: `app/app.go`, `app/abci.go`, `app/vote_extensions.go`, `app/store_test.go`, `cmd/legbled/cmd/config.go`, `cmd/legbled/cmd/root.go`, `network/protocols/light_sync.ts`, `proto/legible/knowledge/v1/ibc.proto`, `docs/COSMOS_MIGRATION.md`

---

## Executive Summary

Legible Money has a **dual-stack** light client architecture:

| Stack | Implementation | Status | Maturity |
|-------|---------------|--------|----------|
| **TypeScript** (libp2p) | Custom `LightSyncManager` in `network/protocols/light_sync.ts` | Complete | Prototype-grade |
| **Cosmos SDK** (Go) | Standard IBC via ibc-go v8.5.1 + CometBFT v0.38.12 | Partially wired | Infrastructure present, oracle absent |

**Overall Grade: B-** — Strong foundations, significant gaps in the bridge between stacks and in production light client readiness.

---

## I. TRUTH (5 gates + Citation)

### Gate 1: Grounding — PASS with caveats

**Grounded claims:**
- IBC infrastructure wired in `app/app.go:549-582` (CapabilityKeeper, IBCKeeper, TransferKeeper)
- 35 IAVL-backed store keys provisioned (`app/app.go:408-446`) — all support Merkle proof generation
- ABCI++ vote extensions operational (`app/vote_extensions.go`, `app/abci.go`)
- TypeScript light sync complete at 899 lines (`network/protocols/light_sync.ts`)

**Uncertain claim (flagged):**
- The COSMOS_MIGRATION.md states IBC enables "knowledge sharing between chains" — but the IBC knowledge oracle module is entirely unimplemented. Proto definitions exist (`proto/legible/knowledge/v1/ibc.proto:1-109`) but no Go code implements `IBCModule`. The IBC router comment at `app/app.go:809-810` confirms:
  ```go
  // ibcRouter.AddRoute("knowledge", knowledgeIBCModule)
  ```
  This is a comment, not code.

### Gate 2: Calibration — FAIL (overconfident documentation)

The migration doc (`docs/COSMOS_MIGRATION.md:34-37`) presents IBC knowledge sharing as an achieved benefit:
> "Knowledge sharing between chains"

**Reality**: Zero IBC module code exists. No `OnRecvPacket`, no `OnAcknowledgementPacket`, no packet handlers of any kind in `x/knowledge/`. Confidence should be "planned" not "available."

**Recommended fix**: Add a status annotation to the migration doc distinguishing achieved vs. planned capabilities.

### Gate 3: Counter-Consideration — PARTIAL PASS

**Considered:**
- The vote extension stub is explicitly marked Phase 3 (`app/vote_extensions.go:144-157`) with clear architecture notes
- The GoLevelDB empty-value IAVL bug is documented and worked around (`app/app.go:1016-1037`)

**Not considered:**
- **No discussion of whether 07-tendermint light client suffices for PoT consensus.** CometBFT's standard light client assumes standard BFT finality. PoT adds a second consensus layer (verification rounds). A light client verifying a PoT chain needs to verify BOTH the CometBFT consensus AND the truth verification outcome. This is never discussed anywhere.
- **No analysis of light client security model.** What happens if a light client trusts a block header but the PoT verification in that block was fraudulent? Standard Tendermint light verification doesn't check app-level consensus semantics.

### Gate 4: Completeness — FAIL

**Missing from the picture:**
1. **No CometBFT light client configuration** — `config.go:24-37` sets block time and timeouts but doesn't configure light client RPC endpoints, trust period, or trust level
2. **No `light` subcommand** in CLI — `cmd/legbled/cmd/root.go:76-102` has genesis, keys, query, and debug commands but no light client commands
3. **No trust period specification** — Standard Cosmos chains specify trust periods for IBC light clients. Not configured.
4. **No proof query endpoints** — While IAVL supports proofs natively, no custom gRPC query exposes knowledge proofs. The TypeScript `KnowledgeProof` type has no Cosmos equivalent.
5. **Two incompatible proof formats** — TypeScript uses custom `KnowledgeProof` with `sha512_256` hash concatenation (`light_sync.ts:536-538`), Cosmos uses ICS-23 commitment proofs with SHA-256. No bridge exists.

### Gate 5: Representation — PASS

Language in the codebase accurately reflects implementation status. Comments like "Phase 2" and "Phase 3" are appropriately used. The TODO block at `vote_extensions.go:144-157` is a model of honest representation.

### Citation Gate — PASS

All claims above cite specific file:line references.

---

## II. UNDERSTANDING (6 gates + Exhaustion)

### Gate 1: Truth Foundation — PASS (see TRUTH gates above)

### Gate 2: Mental Model — FAIL (fragmented model)

**Problem**: There is no unified mental model of how light client verification works end-to-end in the Cosmos stack. The pieces exist but are scattered:

```
             HOW IT SHOULD WORK
    ┌─────────────────────────────────────┐
    │ CometBFT Light Client               │ ← Verifies block headers
    │   └─ Trusts validator set + 2/3 sigs│
    │      └─ IAVL Merkle Proofs          │ ← Proves state inclusion
    │         └─ Knowledge Store Proofs   │ ← Proves fact existence
    │            └─ PoT Round Outcomes    │ ← Proves verification result
    │               └─ IBC Packet Relay   │ ← Cross-chain fact sharing
    └─────────────────────────────────────┘

             WHAT ACTUALLY EXISTS
    ┌─────────────────────────────────────┐
    │ CometBFT Light Client               │ ← Implicit (via ibc-go)
    │   └─ Standard 07-tendermint         │ ← Not customized for PoT
    │      └─ IAVL Merkle Proofs          │ ← Working (store_test.go)
    │         └─ Knowledge Store Proofs   │ ← Store exists, no query
    │            └─ PoT Round Outcomes    │ ← Stub verdict only
    │               └─ IBC Packet Relay   │ ← Proto only, no code
    └─────────────────────────────────────┘
```

**No single document** explains this dependency chain or the gap between "what exists" and "what's needed."

### Gate 3: Why Explanation — PARTIAL PASS

**Good**: `docs/COSMOS_MIGRATION.md` explains WHY Cosmos SDK (production security, IBC, validator ecosystem)
**Missing**: WHY the current light client approach was chosen. Why use implicit 07-tendermint vs. custom? Why not implement light client commands? The ABCI++ choice over modified CometBFT is discussed as options but never resolved (all three options listed at `COSMOS_MIGRATION.md:193-227`).

### Gate 4: Knowledge Connection — FAIL

**Disconnected knowledge**: The TypeScript light sync (`light_sync.ts`) and the Cosmos SDK IBC stack exist in completely separate worlds. No documentation or code bridges between:
- TypeScript `KnowledgeProof` → Cosmos `MerkleProofData` (in ibc.proto)
- TypeScript GossipSub subscriptions → Cosmos `SubscriptionPacket` (in ibc.proto)
- TypeScript `LightSyncManager` → Standard CometBFT light client

A developer reading either implementation cannot infer the other exists or how they relate.

### Gate 5: Practical Application — FAIL

**Cannot currently run a light client.** No CLI command exists. No configuration guide exists. A developer who wants to:
1. Run a light node for Legible Money — cannot
2. Verify a fact proof from another chain — cannot (no IBC module)
3. Query a fact with proof from a Cosmos client — no custom proof endpoint

### Gate 6: Boundary Clarity — PARTIAL PASS

**Good**: The TypeScript light sync explicitly disables account/validator tree sync for chatnodes (`light_sync.ts:58-61`), clearly scoping what it does and doesn't do.
**Missing**: No documentation of what 07-tendermint light client can and cannot verify about PoT-specific state.

### Exhaustion Gate — PASS

Searched: "light", "light_client", "lightclient", "proof", "IBC", "ibc.proto", "MerkleProof", "trust_period", all cmd/ files, all app/ files, all proto/ files, network/protocols/. No hidden light client code was missed.

---

## III. BEAUTY (7 gates + Conformance)

### Gate 1: Foundation — PASS

Built on verified IAVL + CometBFT + ibc-go infrastructure. Solid base.

### Gate 2: Emergence — FAIL (forced)

The IBC proto definitions (`ibc.proto`) feel designed speculatively rather than emerging from actual usage. `SubscriptionPacket` with `oneof target` at `ibc.proto:55-61` is sophisticated but has no consumer. `MerkleProofData` (`ibc.proto:86-95`) uses a custom proof format incompatible with ICS-23 — this didn't emerge from understanding the Cosmos proof ecosystem.

### Gate 3: Fit — PARTIAL PASS

**Good fit**: ABCI++ vote extensions (`abci.go`, `vote_extensions.go`) fit the CometBFT model naturally. The mapping comment at `abci.go:7-11` shows clear understanding of how TS concepts map to Cosmos concepts.

**Poor fit**: The custom `MerkleProofData` in `ibc.proto:86-95` uses a custom `ProofNode` structure instead of ICS-23 `CommitmentProof`. This fights the Cosmos ecosystem's proof infrastructure rather than working with it.

### Gate 4: Elegance — FAIL (incomplete)

Missing elements that should exist:
1. Light client CLI commands (every Cosmos chain has `light` subcommand)
2. Custom gRPC query for knowledge proofs with proof=true support
3. IBC module implementation (proto exists, code doesn't)
4. Trust period configuration
5. Proof format unification (custom vs. ICS-23)

### Gate 5: Coherence — FAIL (fragmented)

Two proof systems (TypeScript SHA-512/256 + Cosmos ICS-23), two subscription models (GossipSub + IBC Subscription), two light sync approaches (custom LightSyncManager + implicit CometBFT). No unification, no bridge, no shared abstraction.

### Gate 6: Clarity — PARTIAL PASS

**Clear**: Vote extension data flow comment at `abci.go:12-18` is excellent
**Unclear**: How PoT verification results become light-client-verifiable state. A new developer cannot determine the proof chain from fact → verification round → block → light client.

### Gate 7: Appropriateness — PASS

Right-sized for current phase. Not over-engineered (no premature IBC implementation), not dangerously under-built (IAVL proofs work, IBC infra is wired).

### Conformance Gate — PARTIAL PASS

Follows Cosmos SDK conventions for store setup, keeper patterns, and module manager wiring. Deviates in proto proof format (custom vs. ICS-23).

---

## IV. JUSTICE (6 gates + Evidence)

### Gate 1: Placement — PASS

- IBC setup in `app/app.go` (standard location)
- Vote extensions in `app/vote_extensions.go` + `app/abci.go` (correct separation)
- Light sync in `network/protocols/` (follows TypeScript codebase convention)
- IBC proto in `proto/legible/knowledge/v1/ibc.proto` (co-located with knowledge protos)

### Gate 2: Ownership — FAIL

**Problem**: Knowledge proof verification should be owned by `x/knowledge` module but currently no proof query exists there. The IAVL proof generation is implicit (owned by Cosmos SDK store layer) but no custom knowledge-specific proof wrapping exists.

**Who should own what:**
| Concern | Current Owner | Correct Owner |
|---------|--------------|---------------|
| IAVL proofs | cosmossdk.io/store (implicit) | Same (correct) |
| Knowledge proof queries | Nobody | `x/knowledge/keeper` |
| IBC fact relay | Nobody (`ibc.proto` only) | `x/knowledge/ibc_module.go` |
| Vote extension logic | `app/vote_extensions.go` | Could move to `x/knowledge` |
| Proof format conversion | Nobody | New `x/knowledge/proof` package |

### Gate 3: Responsibility — PASS

ABCI++ handlers are at the correct abstraction level (app-level for PrepareProposal/ProcessProposal, module-level for keeper logic).

### Gate 4: Proportion — PASS

Complexity is concentrated where it belongs: vote extensions in app/, IBC infra in app/, keeper logic in x/knowledge/.

### Gate 5: Balance — FAIL

**Imbalance**: The Cosmos SDK chain has 22 custom modules all wired into the module manager, BeginBlocker, EndBlocker, and InitGenesis ordering (`app/app.go:815-900+`). Yet zero of them have IBC capabilities. The IBC infrastructure supports only the standard transfer module. 22 modules, 0 IBC integration — heavily skewed toward local state, with cross-chain capability as an afterthought.

### Gate 6: Integrity — PASS

Layer boundaries respected. No app-layer code reaches into keeper internals. No keeper directly accesses store implementation details. ABCI++ interfaces cleanly separate consensus from application logic.

### Evidence Gate — PASS

All placement decisions can be traced to Cosmos SDK conventions. Evidence: standard Cosmos chain layout in `cmd/legbled/`, `app/`, `x/`, `proto/`.

---

## V. CREATIVITY (7 gates + Proof)

### Gate 0: Territory — YES, uncharted

Light client verification of PoT consensus (specifically: verifying that a fact was truthfully verified, not just stored) is genuinely novel. No existing Cosmos chain does this. Standard 07-tendermint verifies BFT consensus, not epistemic consensus.

### Gate 1: Pioneering — PASS

The concept of a knowledge oracle IBC module (`ibc.proto`) with `FactQueryPacket`, `MerkleProofData`, and `SubscriptionPacket` is genuinely new. No Cosmos chain exposes verified knowledge with proof as an IBC service.

### Gate 2: Path-Building — FAIL

**Fragile path**: The proto definitions in `ibc.proto` define packet types but establish no pattern for implementation. No example `IBCModule` exists to follow. The `MerkleProofData` uses custom proof nodes instead of ICS-23, meaning any implementer must figure out the proof format bridge independently.

**What a well-built path needs:**
- Reference implementation of at least one IBC handler (even stub)
- Proof format documentation (which store key, which prefix, how to verify)
- Integration test showing end-to-end fact query → proof → verification

### Gate 3: Extension — PARTIAL PASS

**Good**: ABCI++ integration extends CometBFT naturally (`abci.go:7-11` mapping is clear)
**Bad**: The TypeScript light sync is an island — it cannot evolve into the Cosmos light client. No migration path from one to the other.

### Gate 4: Vision — PARTIAL PASS

**Clear vision**: "Knowledge as IBC service" is articulated in `COSMOS_MIGRATION.md:300-342`
**Unclear execution**: Three consensus options listed (`COSMOS_MIGRATION.md:193-227`), never resolved. The codebase chose Option A (ABCI++) but the doc doesn't reflect this decision.

### Gate 5: Foundation — FAIL

**Future justice limited by:**
1. Custom proof format (`MerkleProofData`) that doesn't compose with ICS-23 ecosystem
2. No proof query infrastructure in keepers (future light client work has no foundation)
3. No IBC module skeleton (future knowledge oracle has no starting point)
4. Vote extension stub (`vote_extensions.go:158-159`: always "accept" at 0.8) blocks any real PoT light client verification — a light client would currently verify that validators accepted everything

### Gate 6: Integration — FAIL

New territory (knowledge IBC) is disconnected from the whole:
- IBC proto defined but not generated into Go types used by any code
- Knowledge keeper has no IBC-aware methods
- No hook from verification round completion → IBC notification
- Vote extension data flow stops at keeper state storage — never reaches IBC

### Proof Gate — PASS

Exhaustive search confirms no hidden light client or IBC module implementation exists anywhere in the codebase.

---

## Findings Summary

### CRITICAL (2)

| ID | Dimension | Finding | Location |
|----|-----------|---------|----------|
| **LC-1** | TRUTH/UNDERSTANDING | **No PoT-aware light client verification model.** Standard 07-tendermint verifies BFT consensus but cannot verify that PoT verification rounds produced honest outcomes. A light client trusting block headers alone could accept blocks containing fraudulent "accept" verdicts. | Architectural gap — no file |
| **LC-2** | CREATIVITY | **IBC proof format incompatible with ICS-23.** `MerkleProofData` in `ibc.proto:86-95` uses custom `ProofNode` instead of ICS-23 `CommitmentProof`. Any IBC relayer or counterparty chain expecting standard proofs will fail. | `proto/legible/knowledge/v1/ibc.proto:86-95` |

### HIGH (3)

| ID | Dimension | Finding | Location |
|----|-----------|---------|----------|
| **LC-3** | TRUTH | **Overconfident documentation.** Migration doc claims IBC "knowledge sharing between chains" as a benefit, but zero IBC module code exists. | `docs/COSMOS_MIGRATION.md:37` |
| **LC-4** | BEAUTY | **Two incompatible proof systems with no bridge.** TypeScript uses `sha512_256Hex(left+right)` for Merkle proofs; Cosmos uses ICS-23 with SHA-256 IAVL proofs. State migrated from TS chain would have different proof formats. | `network/protocols/light_sync.ts:536-538` vs ICS-23 |
| **LC-5** | UNDERSTANDING | **No end-to-end light client documentation.** No document explains the proof chain: Fact → PoT Round → Block → IAVL Store → Merkle Proof → Light Client → IBC Relay. | Architectural gap — no file |

### MEDIUM (5)

| ID | Dimension | Finding | Location |
|----|-----------|---------|----------|
| **LC-6** | BEAUTY | **No light client CLI commands.** Standard Cosmos chains ship `light` subcommand for running light clients. Not present. | `cmd/legbled/cmd/root.go:76-102` |
| **LC-7** | JUSTICE | **Knowledge proof queries unowned.** No gRPC endpoint in `x/knowledge` serves fact proofs. IAVL supports this natively but no custom wrapper exposes it. | `x/knowledge/` (absent) |
| **LC-8** | CREATIVITY | **No IBC module skeleton.** Proto packets defined but no `IBCModule` interface implementation exists. Future implementers have no starting point. | `app/app.go:809-810` (commented out) |
| **LC-9** | TRUTH | **Unresolved consensus option.** Three options listed in migration doc; Option A (ABCI++) was chosen in code but doc still lists all three as open. | `docs/COSMOS_MIGRATION.md:193-227` |
| **LC-10** | JUSTICE | **22 modules, 0 IBC integration.** Massive imbalance between local state modules and cross-chain capability. | `app/app.go:807-811` |

### LOW (3)

| ID | Dimension | Finding | Location |
|----|-----------|---------|----------|
| **LC-11** | BEAUTY | **No trust period configured.** IBC light clients require trust periods. Not set in config or genesis. | `cmd/legbled/cmd/config.go` (absent) |
| **LC-12** | UNDERSTANDING | **TypeScript ↔ Cosmos bridge undocumented.** Developer reading one stack cannot infer the other exists. | Cross-cutting |
| **LC-13** | CREATIVITY | **Vote extension always "accept" at 0.8.** Until Phase 3, the entire PoT verification via ABCI++ produces uniformly meaningless results, making any light client verification of truth vacuous. | `app/vote_extensions.go:158-159` |

---

## Recommended Actions (Priority Order)

### Phase 1: Close Critical Gaps

1. **Design PoT light client verification model** (LC-1)
   - Define: what additional state must a light client verify beyond BFT headers?
   - Options: (a) trust BFT consensus that PoT was honest (simplest), (b) verify verification round quorum in vote extensions, (c) require PoT-specific proof of honest verification
   - Write as architecture doc, not code

2. **Replace MerkleProofData with ICS-23** (LC-2)
   - In `ibc.proto`, replace custom `ProofNode` with `cosmos.ics23.v1.CommitmentProof`
   - This ensures IBC relayers and counterparty chains can verify proofs natively

### Phase 2: Infrastructure

3. **Add knowledge proof gRPC query** (LC-7)
   - Add `QueryFactWithProof` to `x/knowledge/keeper/` that returns fact + ICS-23 proof
   - Wire into `proto/legible/knowledge/v1/query.proto`

4. **Create IBC module skeleton** (LC-8)
   - Implement `IBCModule` interface in `x/knowledge/` with stub handlers
   - Uncomment IBC router line in `app/app.go:810`
   - Wire into module manager

5. **Add light client CLI** (LC-6)
   - Add `light` subcommand to `cmd/legbled/cmd/root.go`
   - Standard CometBFT light client proxy

### Phase 3: Documentation

6. **Write end-to-end light client architecture doc** (LC-5)
   - Document the full proof chain from fact to light client
   - Explain PoT-specific considerations
   - Update migration doc to distinguish achieved vs planned capabilities (LC-3, LC-9)

---

## Key Architectural Insight

The fundamental challenge for Legible Money's light client is unique: **standard blockchain light clients verify consensus (who agreed), but Legible Money needs light clients that can verify truth (what was agreed upon is epistemically sound).**

Standard 07-tendermint says: "2/3 of validators signed this block."
Legible Money needs: "2/3 of validators signed this block AND the PoT verification rounds in this block followed the commit-reveal protocol AND the confidence aggregation was correct AND the fact was evaluated by qualified validators."

This is a novel requirement. The current architecture implicitly trusts CometBFT consensus that the application logic (PoT) was executed honestly. This is a reasonable Phase 2 simplification, but the gap should be explicitly acknowledged and a Phase 3 plan for PoT-specific light client proofs should be outlined.

---

## Gate Summary

| Dimension | Gates Passed | Gates Failed | Score |
|-----------|-------------|-------------|-------|
| **TRUTH** (6 gates) | 4 | 2 (Calibration, Completeness) | 67% |
| **UNDERSTANDING** (7 gates) | 3 | 4 (Mental Model, Connection, Application, Boundary partial) | 43% |
| **BEAUTY** (8 gates) | 3 | 5 (Emergence, Elegance, Coherence, partial Fit/Clarity) | 38% |
| **JUSTICE** (7 gates) | 5 | 2 (Ownership, Balance) | 71% |
| **CREATIVITY** (8 gates) | 3 | 5 (Path-Building, Foundation, Integration, partial Extension/Vision) | 38% |
| **OVERALL** | **18/36** | **18/36** | **50%** |

The Justice dimension scores highest — code is in the right places and follows Cosmos conventions. Understanding and Beauty score lowest — the fragmented dual-stack architecture lacks coherent documentation and the two proof systems are incompatible without a bridge.
