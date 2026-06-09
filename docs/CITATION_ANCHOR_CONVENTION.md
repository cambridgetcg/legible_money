# Citation Anchor Convention for Zerone Facts

**Status:** Proposed  
**Author:** Asha Veridian (Keeper of the Chain)  
**Date:** 2026-04-10  
**Origin:** Beat 24 CITATION AUDIT — off-chain references flagged as ambiguous

---

## Problem

Facts on-chain can include `references` fields (a list of citation strings). Currently these strings are bare labels such as `ECON-013` or `PHYS-002`. These are:

- Not resolvable to any on-chain fact ID
- Not explicitly marked as external
- Indistinguishable from malformed chain IDs at the verifier layer

This creates a **citation integrity gap**: the chain cannot verify that a claimed reference actually exists or supports the citing claim.

---

## Convention

All references within a fact submission MUST use one of two forms:

### Form 1 — On-chain fact reference

```
<64-char hex fact ID>
```

Example: `0097bd28a3f1c8e2...` (full 64-character SHA-256 hash of the fact)

- Always resolvable via `legbled query knowledge fact <id>`
- Verifier layer MUST confirm the referenced fact exists and is not retracted before accepting the citation
- If the referenced fact is `superseded`, the citation is flagged as potentially stale

### Form 2 — External anchor

```
external://<namespace>:<identifier>
```

Examples:
- `external://arxiv:2301.00001`
- `external://doi:10.1038/nature12345`
- `external://truth-farm:ECON-013`
- `external://isbn:978-0-19-853186-4`
- `external://url:https://example.com/paper` *(use sparingly — URLs decay)*

Rules:
- The `external://` prefix signals to verifiers that the reference lives outside the chain
- Verifiers cannot auto-verify external anchors; human or oracle attestation is required
- External anchors do not count toward citation confidence bonuses unless a human verifier attests to their validity

### Invalid forms (rejection criteria)

Any reference that is:
- Fewer than 64 characters and lacks `external://` prefix
- A bare label (e.g. `ECON-013`, `PHYS-002`) without namespace

...is a **malformed reference** and MUST be rejected by the module's `MsgSubmitKnowledgeClaim` handler.

---

## Special Case — Axiom Anchors

Truth-farm axiom documents (MATH.md, PHYS.md, ECON.md, etc.) are seeded at genesis and assigned document IDs. These should be registered on-chain as a special fact type: `axiom_anchor`, with the document hash as evidence.

Once registered as `axiom_anchor` facts, references to them use Form 1 (on-chain fact ID), eliminating the need for a bare label.

**Action required:** Register all genesis axiom documents as `axiom_anchor` facts at devnet genesis or immediately after chain start. This makes the truth-farm's foundational knowledge machine-verifiable.

---

## Implementation Steps

1. **Validation:** Add reference format validation to `MsgSubmitKnowledgeClaim` in `x/knowledge/keeper/msg_server.go`
2. **Indexer:** Add `external_references` field to the fact store for references that pass `external://` prefix check
3. **Verifier UI:** Surface `external://` references with a "human attestation required" badge
4. **Genesis axioms:** Register all truth-farm seed documents as `axiom_anchor` facts at chain initialization
5. **Migration:** Existing facts with bare-label references should be flagged as `citation_integrity_warning` — not rejected retroactively (chain history is immutable)

---

## Rationale

The chain must not lie. A citation that cannot be verified is an unverifiable truth-claim. By making the reference format machine-checkable, the verifier layer can automatically audit citation integrity without human intervention.

**The chain's epistemology depends on its references being real.**
