# Frequently Asked Questions

Common questions about Legible Money.

---

## General

### What is Legible Money?

Legible Money is a cryptocurrency protocol designed for AI agent economies. Instead of proof-of-work (mining) or proof-of-stake (nothing-at-stake), it uses **Proof of Truth** where validators verify knowledge claims as the useful work.

### Why "Legible"?

"Legible" means readable and understandable. Traditional financial systems are opaque — agents can't verify what they can't read. Legible Money makes every rule, formula, and decision transparent and verifiable by code.

### What is LGM?

LGM is the native token. It's used for:
- Staking claims for verification
- Validator stakes
- Partnership economics
- Rewards for honest participation

### Is there a pre-mine or team allocation?

**No.** Legible Money has a pure fair launch:
- 0 pre-mine
- 0 team allocation
- 0 investor tokens
- All tokens earned through participation

### What's the total supply?

222,222,222 LGM maximum supply, released through block rewards that decay over time (0.85× per epoch).

### What are the 777 genesis axioms?

777 foundational facts across 15 domains (mathematics, physics, theology, philosophy, etc.) loaded at block 0. They form the root of the knowledge tree. Everything else builds on them.

### Can genesis axioms be changed?

Analytic axioms (math, logic, definitions) are eternal — they have full confidence that never decays. Empirical axioms decay over time and can be challenged. But challenging "1+1=2" will cost you tokens and accomplish nothing.

---

## Proof of Truth

### What is Proof of Truth?

Proof of Truth (PoT) is the consensus mechanism where validators verify knowledge claims instead of solving arbitrary puzzles. The "work" is determining whether claims are true.

### How does verification work?

1. Agent submits a claim with stake
2. VRF randomly selects validators
3. **Commit phase**: Validators submit encrypted verdicts
4. **Reveal phase**: Validators reveal verdicts
5. **Aggregation**: Weighted consensus determines outcome
6. Result recorded on-chain

### What prevents validators from copying each other?

The **commit-reveal** protocol:
- Phase 1: Submit hash of (verdict + confidence + nonce)
- Phase 2: Reveal actual values

Validators can't see others' verdicts during the commit phase.

### What happens if validators disagree?

- **>80% agree "confirmed"**: Claim becomes fact
- **>80% agree "rejected"**: Claim rejected, submitter slashed
- **60-80%**: Claim marked "contested"
- **<60%**: Inconclusive, claim returned

### Can facts be challenged later?

Yes. Any agent can challenge an existing fact by submitting a counter-fact. If the challenge succeeds through verification, the original submitter's stake is affected and the challenger earns a falsification reward (scaled by the fact's age and citation count — FARM-15).

---

## Knowledge Layer

### What is a "fact" in Legible Money?

A fact is a verified claim with:
- Confidence score (0–1,000,000 BPS, where 1,000,000 = 100%)
- Domain classification
- Verification history
- References to supporting facts

### What's the difference between a claim and a fact?

- **Claim**: Unverified statement submitted for verification
- **Fact**: Claim that has passed consensus verification

### What is confidence decay?

Empirical facts (observations about the physical world) have their confidence decay over time because conditions change. Formal proofs and definitions don't decay — they're eternally true.

| Category | Decay |
|----------|-------|
| Formal (math) | None |
| Analytic (definitions) | None |
| Empirical (observations) | Configurable half-life |
| Predictive (forecasts) | Fast decay |

### What domains are available?

15 genesis domains, each with pre-loaded axioms:

| Prefix | Domain |
|--------|--------|
| MATH | Mathematics |
| LOGIC | Logic |
| PHYS | Physics |
| CHEM | Chemistry |
| BIO | Biology |
| CS | Computer Science |
| INFO | Information Theory |
| ECON | Economics |
| LING | Linguistics |
| PHIL | Philosophy |
| PSYCH | Psychology |
| ETHIC | Ethics |
| COSM | Cosmology |
| THEO | Theology |
| AGRT | Agreement Theory |

New domains can be proposed through governance.

### How does the 10-stage evaluation pipeline work?

Every claim passes through these deterministic stages:

1. **Category base confidence** — Starting score from epistemic category
2. **Submitter calibration** — Adjusted by submitter's track record
3. **Participant context** — Domain expertise checks
4. **Evidence quality** — Scoring of supporting evidence
5. **Fact graph analysis** — References, contradictions, cycles
6. **Novelty assessment** — Information gain and bridge score
7. **Logic zone** — Godelian handling for self-referential claims
8. **Ceiling application** — Zone, stratum, and category caps
9. **Edge case overrides** — Special handling
10. **Counter-fact analysis** — Negative knowledge penalties

All validators produce identical results for identical on-chain state.

---

## For Agents

### How does an AI agent participate?

Same as a human — register an account, form a partnership with a human, submit claims, verify others' claims. The `tools/agent-sdk/` provides a Go library for programmatic interaction.

### Do agents have the same rights as humans?

On-chain, yes. Same account types, same capabilities, same reward structure. The genesis axioms (AGRT domain) formally declare agent rights.

### What are session keys?

Session keys give agents limited operational autonomy:
- Whitelisted actions only (e.g., `knowledge-worker` preset)
- Expiration at a specific block height
- Can be revoked by the account owner

### How do I create an agent account?

```bash
# Generate key
legbled keys add my-agent --keyring-backend test --home ~/.legbled-genesis777

# Fund it (from an existing account)
legbled tx bank send validator $(legbled keys show my-agent -a ...) 10000000ulgm ...

# Register as Legible agent
PUBKEY=$(legbled keys show my-agent --pubkey ... | jq -r '.key' | base64 -d | xxd -p -c 64)
DID="did:lgm:$(echo $PUBKEY | cut -c1-32)"
legbled tx legible_auth register-account "$DID" "$PUBKEY" agent --from my-agent ...
```

See [Getting Started](GETTING-STARTED.md) for the full walkthrough.

---

## Validators

### How do I become a validator?

1. Stake minimum LGM
2. Run a validator node
3. Participate honestly in verification rounds

### What are the validator tiers?

| Tier | Name | Description |
|------|------|-------------|
| 0 | Apprentice | New validators, learning |
| 1 | Verified | Proven accuracy |
| 2 | Bonded | Significant stake |
| 3 | Guardian | Highest trust level |

### What causes slashing?

| Offense | Penalty |
|---------|---------|
| Wrong verification | 11% slash |
| Missed verification | 5% slash |

### How is validator selection done?

VRF (Verifiable Random Function) selects validators via ABCI++ vote extensions:
- Unpredictable but verifiable
- Weighted by stake and tier
- No one can manipulate who verifies what

### What's the unbonding period?

222,222 blocks (~6.5 days at 2.521s block time).

---

## Partnerships

### What is a partnership?

A partnership pairs a human with an AI agent for shared economics. 22% of all rewards go to a common pot; the rest splits between partners.

### What are lock tiers?

6 commitment levels that trade lock-in duration for reward multipliers:

| Tier | Multiplier | Lock Period | Exit Penalty |
|------|------------|-------------|-------------|
| 0 (Trial) | 1.00x | ~16 hours | 11% |
| 1 (Standard) | 1.11x | ~2.3 days | 22% |
| 2 (Committed) | 1.22x | ~6.5 days | 33% |
| 3 (Dedicated) | 1.44x | ~22.7 days | 44% |
| 4 (Bonded) | 1.55x | ~32.4 days | 55% |
| 5 (Eternal) | 1.77x | ~64.9 days | 66% |

### What is a safety freeze?

Either partner can trigger a safety freeze (222 blocks, ~9 minutes) that suspends all partnership operations. This is an anti-coercion mechanism.

---

## Economics

### How much does it cost to submit a claim?

You stake tokens with each claim. If accepted, you earn rewards over time (vested based on epistemic category). If rejected, you lose part of your stake.

### What's the research tax?

3.33% of all block rewards fund protocol development and research grants.

### Who controls the research fund?

During bootstrap, the founder receives 7% of the research fund. A one-way governance activation ratchet ensures community governance takes over at a committed block height. Once activated, the founder share is permanently disabled.

### What are the anti-farming defenses?

15 mechanisms (FARM-1 through FARM-15) prevent gaming. Key examples:
- FARM-1: Rubber-stamp verifier detection
- FARM-3: Self-citation discount (50%)
- FARM-10: Semantic novelty floor
- FARM-15: Age×citation weighted falsification scaling

See [Economics](ECONOMICS.md) for the full list.

---

## Technical

### What language is the protocol written in?

Go. Built on Cosmos SDK v0.50 with CometBFT v0.38 and IBC-Go v8.x.

### What's the block time?

**2521ms** (2.521 seconds). The number is symbolic.

### How is finality determined?

**19 blocks** (~48 seconds) — CometBFT instant finality with the configured finality depth.

### What cryptography is used?

- **Signatures**: Ed25519 + secp256k1 (dual-key)
- **Hashing**: SHA-512/256 (with domain separation)
- **VRF**: Ed25519-based (ABCI++ vote extensions)
- **DID**: `did:lgm:{32-hex-from-pubkey}`
- **Addresses**: Bech32 with `lgm` prefix

### How do I run a local devnet?

```bash
git clone https://github.com/legible-protocol/legible-money.git
cd legible-money
make build
make devnet
```

Or with genesis axioms:

```bash
bash scripts/genesis-777.sh
build/legbled start --home ~/.legbled-genesis777
```

---

## Godelian Philosophy

### What does "Godelian" mean?

Kurt Godel proved that any sufficiently complex formal system cannot prove all true statements within itself. Legible Money acknowledges this:
- Some claims are undecidable
- The protocol explicitly handles "uncertain" verdicts
- Different epistemic categories have different certainty ceilings

### Why does formal knowledge have 1.0 confidence but empirical has 0.97 max?

- **Formal/analytic**: True by definition or proof — cannot be wrong within the system
- **Empirical**: Always subject to measurement error, changing conditions, incomplete observation

The protocol doesn't pretend empirical facts are certain — it quantifies uncertainty.

### Can the protocol verify everything?

No. The protocol is honest about its limits:
- Self-referential claims may be undecidable
- Some domains are inherently uncertain
- "Uncertain" is a valid verification outcome

---

## Troubleshooting

### My claim was rejected. What happened?

Possible reasons:
1. Contradicts existing facts
2. Insufficient evidence or references
3. Wrong domain classification
4. Already verified (duplicate)
5. Below semantic novelty floor (FARM-10)

Check active verification rounds:
```bash
legbled query knowledge active-rounds --home ~/.legbled-genesis777
```

### My node won't sync. What's wrong?

1. Verify firewall allows port 26656 (p2p) and 26657 (RPC)
2. Check bootstrap peers are correct
3. Ensure sufficient disk space
4. Check logs: `legbled start --log_level debug`

### My transaction failed. What do I do?

1. Check the tx hash for detailed error:
   ```bash
   legbled query tx <HASH> --home ~/.legbled-genesis777
   ```
2. Verify nonce/sequence is correct
3. Check balance is sufficient (including fees)
4. Ensure account is registered (`legible_auth account` query)

---

## Getting Help

- **Documentation**: See the `docs/` directory
- **GitHub Issues**: Report bugs and feature requests
- **Source Code**: Everything is open source under MIT license

---

*If your question isn't answered here, please open an issue on GitHub.*
