# Human-Agent Epistemic Roles in Legible Money

> Agents cannot interact with the physical environment. Humans can.
> This is the foundational asymmetry of a knowledge protocol.

---

## 1. TRUTH — The Derivable vs. Observable Divide

Every knowledge claim falls into one of two epistemic domains:

**Derivable** — truth follows from logic, axioms, or computation alone. An AI agent can verify these claims end-to-end without any physical observation. These categories have zero confidence decay (eternal half-life) because their truth does not depend on a changing world.

| Category | Why Derivable |
|----------|---------------|
| `analytic` | True by definition (tautologies, logical identities) |
| `formal` | Provable from axioms within a formal system |
| `protocol` | True by construction (on-chain data, consensus rules) |
| `computational` | Deterministically verifiable by re-execution |

**Observable** — truth depends on the physical world. These claims require evidence that *connects to human observation* at some point in the chain, even if the claim itself is submitted by an agent. These categories have finite half-lives because the world changes.

| Category | Why Observable | Half-life |
|----------|----------------|-----------|
| `empirical` | Requires experiment/measurement | ~33 days |
| `historical` | Requires archival research | ~17 days |
| `predictive` | Requires outcome observation | ~0.7 days |
| `social` | Requires social context | ~5 days |
| `replicated` | Requires independent replication | ~33 days |

The boundary is already implicit in three places in the codebase:
1. `CategoryDecayProfile.RequiresReverification` — all observable categories require it
2. `EVIDENCE_TYPE_STRENGTH` — physical evidence types rank below on-chain/computation
3. `IsDeterministicCategory()` — identifies categories resolvable by computation

This design makes the boundary **explicit**.

---

## 2. UNDERSTANDING — Decay Profiles Reveal Reality-Dependence

The decay profiles encode a deep insight: **the rate at which confidence decays correlates with how tightly a claim is coupled to physical reality**.

```
Eternal (no decay)     ← analytic, formal, protocol, computational
                         Truth is self-contained.

Slow decay (~33 days)  ← empirical, replicated
                         Truth is well-established but world changes.

Medium decay (~5-17d)  ← historical, social
                         Truth depends on context that shifts.

Fast decay (<1 day)    ← predictive
                         Truth expires when the predicted event resolves.
```

This gradient is not a binary — it's a spectrum of reality-dependence. The participant context stage respects this by checking the *evidence chain* rather than the *submitter type*.

---

## 3. BEAUTY — The Clean Separation

The design principle is elegant in its simplicity:

> **Agents own derivable truth. Humans provide observable ground truth. Neither is excluded from any action; what differs is the evidence chain.**

An agent can:
- Submit empirical claims (if it cites human-sourced evidence)
- Verify any claim type
- Challenge any fact

A human can:
- Submit formal proofs (agents will verify them computationally)
- Provide attestations and experimental evidence
- Serve as the grounding link between claims and physical reality

The asymmetry is in the **evidence**, not the **participant**:

| Evidence Type | Strength | Agent-Verifiable? | Source |
|---------------|----------|-------------------|--------|
| `on_chain` | 100% | Yes | Blockchain state |
| `computation` | 95% | Yes | Deterministic re-execution |
| `experiment` | 85% | No | Lab results, measurements |
| `oracle_feed` | 70% | No | External data providers |
| `document` | 60% | No | Archived records |
| `attestation` | 50% | No | Human/institutional testimony |

When an observable claim has *only* agent-verifiable evidence (on_chain + computation), it means the evidence chain lacks physical grounding. The evaluation pipeline applies:
- **10% confidence penalty** (the gap between computation and observation)
- **70% confidence ceiling** (even perfect agent evidence can't exceed this for observable claims)

When an observable claim includes human-sourced evidence, it gets a small bonus (up to 3%) for proper grounding.

---

## 4. JUSTICE — Where the Boundary Lives in Code

### Type Primitives (`x/knowledge/types/types.go`)

```go
type EpistemicDomain string
const (
    EpistemicDomainDerivable  = "derivable"   // agent-sufficient
    EpistemicDomainObservable = "observable"   // reality-dependent
)

var CategoryEpistemicDomain = map[string]EpistemicDomain{ ... }

func IsDerivableCategory(category string) bool
func IsAgentVerifiableEvidence(evidenceType string) bool
func RequiresHumanInvolvement(evidenceType string) bool
```

### Evaluation Stage 3: Participant Context (`x/knowledge/evaluation/participant_context.go`)

Inserted between submitter calibration (stage 2) and evidence quality (stage 4) in the 9-stage pipeline:

1. Category base confidence
2. Submitter calibration
3. **Participant context** ← new
4. Evidence quality scoring
5. Fact graph analysis
6. Logic zone evaluation
7. Ceiling application
8. Edge case overrides
9. Counter-fact analysis

The stage is a no-op for derivable categories. For observable categories, it checks whether the evidence chain includes human-grounded evidence types.

### Governance Parameters (`ExtendedParams`)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `human_evidence_gap_penalty_bps` | 100,000 (10%) | Penalty for observable claims with only agent evidence |
| `agent_only_empirical_ceiling` | 700,000 (70%) | Confidence ceiling in that case |
| `human_evidence_bonus_bps` | 10,000 (1%) | Bonus per human evidence item |
| `max_human_evidence_bonus_bps` | 30,000 (3%) | Maximum total bonus |

All values are governance-tunable via `MsgUpdateExtendedParams`.

---

## 5. CREATIVITY — Future Directions

### Collaborative Verification Panels

Mixed human-agent panels for high-stakes observable claims. Agents handle structural verification (consistency, reference checking, logical validity) while humans handle observational verification (does the evidence match physical reality?).

### Reality Oracle Network

A dedicated network of human validators who specialize in grounding claims to physical reality. These validators would:
- Attest to experimental results
- Verify document authenticity
- Provide independent observations

### Physical Sensor Agents

As IoT and robotics advance, some agents may gain the ability to make physical observations. The epistemic domain framework naturally extends to this: a sensor agent's `measurement` evidence type would be classified as human-involvement-equivalent, contingent on the sensor's calibration chain connecting to human standards.

### Graduated Evidence Chains

Future work could implement a full evidence provenance system where each piece of evidence tracks its chain back to either:
- An axiom/computation (derivable ground)
- A physical observation by a verified human or calibrated sensor (observable ground)

This would replace the current binary classification with a continuous "grounding depth" metric.

---

## Appendix: Alignment with Existing Architecture

| Existing Pattern | New Formalization |
|-----------------|-------------------|
| `RequiresReverification: true` | `EpistemicDomainObservable` |
| `RequiresReverification: false` | `EpistemicDomainDerivable` |
| `IsDeterministicCategory()` | `IsDerivableCategory()` (superset — includes `analytic`) |
| `evidenceRequiredCategories` | Observable categories (identical set) |
| `EVIDENCE_TYPE_STRENGTH` ordering | `IsAgentVerifiableEvidence()` / `RequiresHumanInvolvement()` |
| `ChallengeWindowProfile` multipliers | Longer windows for derivable (more scrutiny time, less urgency) |
