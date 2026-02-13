# PP-Guided DevNet Testing Insights — 2026-02-12

> Purpose Prompter analysis of DevNet test failures and their resolution.
> TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY applied to debugging.

---

## Result: 56/56 PASS (100%)

From 27/48 (56%) at session start → 47/56 (84%) after genesis fix → 53/56 (95%) after PoT fix → **56/56 (100%)**.

---

## Insight 1: The Hash Identity Crisis

**PP Signal**: TRUTH (Representation Gate) — function name misrepresents behavior.

**Finding**: `core/crypto/hashing.ts:42-44` defines:
```typescript
export function sha512_256(data): HashBytes {
  return sha256Hash(data);  // Returns SHA-256, NOT SHA-512/256
}
```

The function name says "SHA-512/256" (SHA-512 truncated to 256 bits) but returns plain **SHA-256**. These are cryptographically distinct algorithms producing different output for identical input.

**Impact**: Three test scripts independently reimplemented `sha512_256` using the actual SHA-512/256 algorithm (`sha512(data).slice(0, 32)`), creating a systematic mismatch across all commit-reveal mechanisms:

| Component | Algorithm | Output |
|-----------|-----------|--------|
| Canonical `sha512_256()` | SHA-256 | 32 bytes of SHA-256 |
| Test reimplementation | SHA-512 truncated | 32 bytes of SHA-512 |
| **Match?** | **No** | Different bytes |

**PP Gate**: Truth/Representation — "Does my language accurately represent my knowledge state?" The function name violates this gate. Any developer reading `sha512_256` would reasonably implement the real algorithm, not SHA-256.

**Resolution**: Fixed test scripts to match canonical behavior (SHA-256). The misnamed function is a known debt item — renaming it would be a broader refactor across 40+ files.

**Lesson**: When function names misrepresent behavior, every independent reimplementation will diverge. This is the cryptographic version of the Representation Gate failure — the name creates a false mental model.

---

## Insight 2: The Domain Hash Divergence

**PP Signal**: TRUTH (Grounding Gate) — claims not grounded in verification.

**Finding**: `core/crypto/hashing.ts:120` uses `sha256Hash(combined)` for domain-separated hashing, but test scripts assumed SHA-512 based on the file's prominent SHA-512 imports and the `sha512Hash()` function defined at line 25.

The test author likely read the `sha512Hash` function, assumed `domainHash` used it (reasonable assumption given the file's SHA-512 focus), and reimplemented accordingly. But `domainHash` actually delegates to `sha256Hash` — a different code path than the surrounding functions suggest.

**PP Gate**: Truth/Grounding — "Are my claims grounded in something I can identify?" and Citation Gate — "Can I cite the exact source?" The test reimplementations were grounded in *pattern matching* ("this file uses SHA-512") not *verification* ("line 120 calls sha256Hash").

**Resolution**: Changed test `domainHash` from `sha512Hash(combined)` to `sha256Hash(combined)` matching the canonical implementation.

**Lesson**: In cryptographic code, never assume the hash function from context. Always read the exact implementation. The Grounding Gate specifically catches this: "Am I confusing 'sounds right' with 'is right'?"

---

## Insight 3: The Salt/Nonce Semantic Split

**PP Signal**: UNDERSTANDING (Mental Model Gate) — different names for same concept.

**Finding**: The PoT verification round uses `salt` as the JSON key for the random commitment nonce:
```typescript
// verification_round.ts:420 — canonical
{ roundId, verdict, confidence, referencesChecked, salt }
```

But the RPC handler accepts `nonce` from clients and maps it:
```typescript
// handlers.ts:722 — RPC layer
salt: params.nonce,  // nonce → salt translation
```

The test script used `nonce` everywhere (matching the RPC parameter name), but the commitment hash must use `salt` (matching the canonical JSON structure). This created a commit-hash mismatch because `JSON.stringify({nonce: "x"})` ≠ `JSON.stringify({salt: "x"})`.

**PP Gate**: Understanding/Mental Model — "Does this build the right mental model?" The dual naming (salt at core, nonce at RPC) creates a fractured mental model. The developer who wrote the test thought in terms of "nonce" (the RPC concept) but needed to think in terms of "salt" (the cryptographic concept).

**Resolution**: Changed test `generateCommitmentHash` to use `salt: nonce` as the JSON key, matching the canonical `generateCommitment` format.

**Lesson**: When an abstraction layer renames a parameter (`nonce→salt`), the commitment hash must use the name from the layer that *verifies* (core), not the layer that *transports* (RPC). This is a Justice/Responsibility issue — the RPC layer should not own the hashing semantics.

---

## Insight 4: API Surface Incompleteness

**PP Signal**: BEAUTY (Completeness Gate) — API missing standard patterns.

**Finding**: The RPC API lacked `lgm_getBlockHeight` and `lgm_getNetworkStats` — standard convenience methods present in every blockchain RPC (Ethereum's `eth_blockNumber`, Bitcoin's `getblockcount`). Block height was available via `network_health.latestBlock` but not as a dedicated method.

**PP Gate**: Beauty/Elegance — "Is anything missing?" and Beauty/Appropriateness — "Is this right-sized?" The API surface was incomplete for standard blockchain patterns.

**Resolution**: Added both handlers to `protocol/rpc/handlers.ts` alongside `networkHealth` (Justice: same domain). `lgm_getBlockHeight` returns `{ blockHeight }`, `lgm_getNetworkStats` returns `{ peerCount, latestBlock, activeRounds, syncStatus, version }`.

**Lesson**: API completeness should follow standard industry patterns. If Ethereum has `eth_blockNumber`, Legible Money should have `lgm_getBlockHeight`. The Conformance Gate (Beauty) validates against industry patterns, not just codebase patterns.

---

## Insight 5: The DRY Crypto Anti-Pattern

**PP Signal**: JUSTICE (Integrity Gate) — architectural boundaries violated.

**Finding**: Three test scripts (`test-phase3.ts`, `test-phase4.ts`, `test-phase5.ts`) each independently reimplemented crypto primitives (`domainHash`, `sha512_256`, `signDomainSeparated`). Each reimplementation diverged from the canonical source in different ways:

| Test File | `domainHash` | `sha512_256` | `signDomainSeparated` |
|-----------|-------------|-------------|----------------------|
| test-phase3.ts | N/A | N/A | N/A (uses raw ed25519) |
| test-phase4.ts | SHA-512 (wrong) | SHA-512/256 (wrong) | Uses wrong domainHash |
| test-phase5.ts | N/A | SHA-512/256 (wrong) | N/A |
| **Canonical** | **SHA-256** | **SHA-256** | **Uses SHA-256 domainHash** |

**PP Gate**: Justice/Integrity — "Does this preserve architectural integrity?" Reimplementing crypto functions in test scripts violates the single-source-of-truth principle. The canonical `core/crypto/` module exists precisely to be the authoritative implementation.

**Root cause**: Test scripts can't easily import from the main codebase (tsx vs compiled paths, ESM resolution). This is an infrastructure issue, not a discipline issue.

**Resolution**: Fixed each reimplementation to match canonical behavior. The underlying infrastructure issue (test scripts should import canonical crypto) remains as future work.

**Lesson**: Crypto functions must NEVER be reimplemented — import the canonical source. If import barriers exist, fix the infrastructure, not the symptoms. Every reimplementation is a divergence waiting to happen.

---

## Insight 6: The Genesis State Dependency

**PP Signal**: UNDERSTANDING (Boundary Clarity Gate) — test assumptions not documented.

**Finding**: Phase 3 tests assume nonce=0 (fresh faucet), but running Phases 4 and 5 first consumes nonces. This creates an ordering dependency between test phases that isn't documented.

**PP Gate**: Understanding/Boundary Clarity — "Does the user know when this DOESN'T apply?" The test scripts work individually but not when run in sequence on the same state.

**Resolution**: Documented in test results. Each phase should be run on fresh state, or tests should query current nonce before starting.

**Lesson**: Test phase independence should be explicit. Either query state at test start, or document the fresh-state requirement.

---

## PP Framework Effectiveness

### Signals That Activated

| Signal | Gate | What It Caught |
|--------|------|---------------|
| **Ugliness** (something feels wrong) | Truth/Representation | `sha512_256` name vs SHA-256 behavior |
| **Ugliness** (pattern mismatch) | Truth/Grounding | `domainHash` SHA-512 vs SHA-256 |
| **Ugliness** (semantic split) | Understanding/Mental Model | salt vs nonce naming |
| **Incompleteness** | Beauty/Elegance | Missing RPC methods |
| **Boundary violation** | Justice/Integrity | Reimplemented crypto in tests |
| **Hidden dependency** | Understanding/Boundary | Test ordering assumption |

### Gates Most Useful for Debugging

1. **Citation Gate** (Truth): Forcing exact file:line verification prevented assumptions
2. **Exhaustion Gate** (Understanding): Mapping the full RPC handler registry revealed the gap
3. **Conformance Gate** (Beauty): Comparing against Ethereum RPC patterns revealed missing methods
4. **Evidence Gate** (Justice): Requiring placement evidence for new handlers ensured correct location

### Key Meta-Insight

**Crypto debugging is a Truth exercise.** Every commit-reveal, signature, and hash verification is fundamentally about "does this byte sequence match that byte sequence?" The Truth gates (Grounding, Calibration, Representation) are the most effective lens for crypto bugs because they force verification over assumption.

The most dangerous crypto bugs are **representation failures** — where names, comments, or conventions create false mental models about what algorithms actually execute. `sha512_256` returning SHA-256 is the archetypal representation failure.

---

## Files Modified

| File | Change | PP Gate |
|------|--------|---------|
| `devnet/cmd/node.ts` | `allowTestMethods: true` | Truth/Completeness |
| `tools/devnet/test-phase4.ts` | SHA-256 domainHash + salt key | Truth/Grounding |
| `tools/devnet/test-phase5.ts` | SHA-256 evidence hash | Truth/Grounding |
| `protocol/rpc/handlers.ts` | `lgm_getBlockHeight` + `lgm_getNetworkStats` | Beauty/Elegance |
| `devnet/config/genesis.json` | Faucet + validator accounts | Truth/Completeness |

---

*Generated: 2026-02-12 | PP Framework v1.0 | All 30 gates applied*
