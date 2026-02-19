# B13-6: Testnet Documentation — Session Report

## Summary

Comprehensive documentation rewrite and creation covering README, Getting Started, Economics, FAQ, and verification of Genesis Axioms docs.

---

## Files Modified

### 1. `README.md` — Full Rewrite

**Before**: 135 lines, outdated SDK version (v0.50.10), incomplete module table (many marked "Stub" that are now Substantial), no genesis axiom mention, TypeScript prototype note prominent.

**After**: Complete rewrite with:
- Quick start with genesis-777 chain and first claim walkthrough
- Knowledge tree architecture (777 axioms, 10-stage pipeline, verification)
- Human-agent partnership summary (6-tier lock, common pot, safety)
- Economic model overview (3 LGM base, 0.85× decay, 222M cap, 3.33% research tax)
- Updated module table (25+ modules with accurate status)
- Full directory structure including tools/ and genesis-axioms/
- Development commands section (all Makefile targets)
- Documentation links section
- Accurate token properties table (bech32 prefix, DID format, finality)

### 2. `docs/GETTING-STARTED.md` — Created

Step-by-step guide covering:
1. Building from source (Go 1.24+, make build)
2. Running local testnet (genesis-777 or standard devnet)
3. Registering accounts (key gen, funding, DID derivation, register-account)
4. Forming partnerships (initiate, accept, lock tier table)
5. Submitting first claim (references, stake, domain)
6. Understanding evaluation (10-stage pipeline) and verification (commit-reveal)
7. Checking rewards and dashboard
8. Session keys for agent autonomy

Uses correct CLI commands throughout (`legible_auth register-account`, positional args, `--preset`).

### 3. `docs/ECONOMICS.md` — Created

Comprehensive economics documentation:
- Token supply: 222,222,222 LGM, pure fair launch
- Block reward emission: 3 LGM base, 240,000 blocks/epoch, 0.85× decay, 0.1 LGM floor
- Research tax: 3.33% with 7% founder share + governance ratchet
- Vesting curves: 10 categories with half-lives, cliffs, max release rates
- Clawback: 33% on falsification
- Verification rewards and slashing
- Citation rewards with FARM-3 self-citation discount
- Falsification rewards: FARM-15 dynamic scaling formula
- Partnership economics: 22% common pot, 6-tier lock multipliers, deliberation, safety
- All 15 anti-farming defenses (table format)
- Confidence system: epistemic categories, decay, propagation
- Genesis economics

### 4. `docs/FAQ.md` — Rewritten

**Key fixes from old version**:
| Old (wrong) | New (correct) |
|-------------|---------------|
| "TypeScript... Node.js 20+" | "Go... Cosmos SDK v0.50" |
| `npm install` / `docker-compose` setup | `make build` / `make devnet` |
| `client.ensureAccount()` TypeScript SDK | `legbled tx legible_auth register-account` CLI |
| `client.registerValidator(1111n...)` | CLI-based instructions |
| Research fund: 22% | Research fund: 3.33% |
| Emission: 1M/500k/250k halving | 3 LGM base, 0.85× decay per epoch |
| Finality: 12/44 blocks | 19 blocks |
| DID: `did:lgm:{network}:{id}` | DID: `did:lgm:{32-hex}` |
| Confidence: 0-1 float | Confidence: 0–1,000,000 BPS |
| Empirical max: 0.99 | Empirical max: 0.97 (970,000 BPS) |
| Domain format: `science.physics` | Domain prefix: `PHYS` |

Added:
- 10-stage evaluation pipeline explanation
- Partnership section (lock tiers, safety freeze, pot deliberation)
- All 15 domains with prefix table
- Correct CLI examples throughout
- Updated troubleshooting with `legbled` commands

### 5. `docs/GENESIS-AXIOMS.md` — No Changes

Already solid and accurate (103 lines covering 15 domains, axiom types, loading pipeline, validation rules). No updates needed.

---

## Data Sources

All numbers verified against source code:
- `go.mod`: Go 1.24.0
- `x/vesting_rewards/types/genesis.go`: 3 LGM base, 0.85 decay, 240k blocks/epoch, 3.33% research, 7% founder
- `x/vesting_rewards/types/keys.go`: 222,222,222 LGM max supply
- `x/partnerships/types/types.go`: 6 lock tiers with exact multipliers/penalties
- `x/partnerships/types/genesis.go`: 22% common pot default
- `x/knowledge/evaluation/engine.go`: 10-stage pipeline
- `x/knowledge/types/types.go`: FARM-1 through FARM-15, confidence values
- `Makefile`: All build/test/devnet targets

---

## Link Verification

All internal links from README.md verified:
- `docs/GETTING-STARTED.md` ✓ (created)
- `docs/ECONOMICS.md` ✓ (created)
- `docs/GENESIS-AXIOMS.md` ✓ (exists)
- `docs/FAQ.md` ✓ (updated)
- `docs/BLOCKCHAIN_ARCHITECTURE.md` ✓ (exists)
- `docs/AGENT_ACCOUNTS.md` ✓ (exists)

Cross-links within docs/ verified:
- FAQ → GETTING-STARTED.md ✓
- FAQ → ECONOMICS.md ✓
- GETTING-STARTED → ECONOMICS.md ✓
- GETTING-STARTED → GENESIS-AXIOMS.md ✓
- GETTING-STARTED → FAQ.md ✓
