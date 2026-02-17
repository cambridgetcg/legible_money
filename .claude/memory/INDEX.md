# Memory File Index

> Master index of all `.claude/memory/` files. Read this to find what you need.
> Files grouped by function. Sizes approximate.

---

## Quick Start (read these first)

| File | Size | Purpose |
|------|------|---------|
| **CURRENT_STATE.md** | 43K | Living project status — what's built, what's next |
| **DECISIONS.md** | 7K | Immutable technical decision log |
| **BLOCKED.md** | 2K | Active blockers (currently: none) |
| **EUREKA.md** | 8K | 23 ahh-hah insights — deep patterns across the project |
| **THINKING.md** | 9K | Mental models and reasoning frameworks |

---

## Insight & Wisdom (accumulated understanding)

| File | Size | Purpose |
|------|------|---------|
| **INSIGHTS.md** | 7K | 13 core insights + 5 meta-insights from implementation |
| **INSIGHT_TREE.md** | 20K | Hierarchical synthesis: 10 domains, open critical items |
| **EUREKA.md** | 8K | 23 eureka moments from 90+ file review |
| **MOTIVATIONS.md** | 10K | The "why" behind 11 major design decisions |
| **EVOLUTION.md** | 6K | Project history: 5 phases of development |
| **PROMPTS.md** | 6K | 30+ effective prompt templates for common tasks |

---

## Strategic Planning

| File | Size | Purpose |
|------|------|---------|
| **LAUNCH_PLAN.md** | 11K | Genesis philosophy — "perpetual genesis" |
| **LAUNCH_ROADMAP.md** | 11K | Detailed roadmap with 6 phases |
| **LAUNCH_SIMULATION_AUDIT.md** | 30K | 5-phase simulation validation |
| **INFRA_PROVIDER_INSIGHTS.md** | 11K | Infrastructure provider evaluation (10 providers) |
| **TEST_PROGRESS.md** | 25K | Test coverage tracker by module |
| **GAPS.md** | 27K | Implementation gaps — all 10 complete |

---

## Security & Adversarial Analysis

| File | Size | Purpose |
|------|------|---------|
| **ZERO_DAY_VULNERABILITIES.md** | 22K | 38 findings (8 CRITICAL) — all critical remediated |
| **SECURITY_DIMENSIONS_PP_REVIEW.md** | 43K | 8 security dimensions, 418 tests |
| **PP_SECURITY_TESTING_INSIGHTS.md** | 6K | Security testing framework |
| **BVM_KEY_SECURITY_REVIEW.md** | 9K | BVM security considerations |
| **LOOPHOLE_REMEDIATIONS.md** | 21K | Financial loophole remediation strategy |
| **OPENCLAW_SESSION_2026_02_16.md** | 8K | Game-theoretic probe results (15 scenarios) |
| **OPENCLAW_FINANCIAL_INCENTIVES.md** | 18K | Deep game-theory analysis (OC-1 to OC-15) |
| **OPENCLAW_MONEY_PSYCHOLOGY.md** | 16K | Behavioral economics of the protocol |

---

## Deep Technical Reviews (PP 30-gate)

### Core Modules
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **ANTE_HANDLER_REVIEW.md** | 17K | app/ante | 16-decorator chain analysis |
| **AUTH_UPGRADEABILITY_REVIEW.md** | 8K | x/auth | Upgrade path assessment |
| **STORE_ARCHITECTURE_REVIEW.md** | 24K | state/storage | Store key design review |
| **PARAMS_MODULE_REVIEW.md** | 9K | x/params | Parameter module review |
| **LEGACY_REMOVAL_PP_REVIEW.md** | 10K | dead code | Code cleanup review |

### Consensus & Validation
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **VOTE_EXTENSIONS_PP_REVIEW.md** | 26K | ABCI++ | Vote extension design |
| **VALIDATOR_CYCLING_GRIEF_REVIEW.md** | 16K | x/staking | Rotation attack vectors |
| **SLASHING_REVIEW.md** | 23K | x/staking | Slashing mechanics (14 contexts) |
| **LIGHT_CLIENT_PP_REVIEW.md** | 22K | IBC | Light client design |
| **SCHEDULE_PP_REVIEW.md** | 17K | DMSS | Schedule module review |

### Economics
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **FINANCIAL_INCENTIVES_PP_REVIEW.md** | 32K | economics | 56 loopholes, 6 CRITICAL |
| **STAKING_REWARD_PP_REVIEW.md** | 17K | x/staking | Staking + rewards |
| **GAS_CALIBRATION_PP_REVIEW.md** | 22K | app/gas | Gas cost calibration |
| **GAS_FEES_REVIEW.md** | 23K | app/gas | Fee structure analysis |
| **REWARD_PATHWAY_ANALYSIS.md** | 28K | economics | 19 reward pathways mapped |
| **REWARD_PENALTY_AUDIT.md** | 20K | economics | 8 reward + 7 penalty types |
| **RETRIEVAL_MECHANISM_INSIGHTS.md** | 29K | tokens | 6 value recapture mechanisms |
| **TOKEN_EXPIRATION_SCENARIOS.md** | 26K | x/tokens | Complete reference: lifecycle, configs, anti-exploits, tiers |
| **TOKEN_EXPIRATION_EDGE_SCENARIOS.md** | 25K | x/tokens | Edge cases (C1-C3, H1-H5, M1-M8, L1-L6) |
| **EXPIRATION_REVIEW.md** | 14K | x/tokens | Token expiration mechanics |

### Knowledge & Claims
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **CLAIM_EDGE_SCENARIOS.md** | 25K | x/knowledge | Claim lifecycle edge cases |
| **CAPTURE_CHALLENGE_PP_REVIEW.md** | 20K | x/knowledge | Capture defense system |
| **EVIDENCE_MGMT_PP_REVIEW.md** | 29K | x/evidence | Evidence management deep dive |
| **CLAW_INTEGRATION_REVIEW.md** | 11K | counter-CLAW | Counter-CLAW mechanism |

### BVM & Agent Home
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **BVM_CONTRACT_INSIGHTS.md** | 19K | x/bvm | Smart contract design |
| **BVM_AGENT_HOME_PROGRESS.md** | 15K | x/bvm+home | BVM + Agent Home status |
| **PP_BVM_HOME_ANALYSIS.md** | 25K | x/bvm+home | 30-gate PP review |
| **PP_AGENT_PERSISTENCE_REVIEW.md** | 21K | x/home | Agent persistence (37→45/50) |
| **PP_HOME_LIVING_FEATURES_PLAN.md** | 36K | x/home | Living features plan |

### Cosmos SDK Integration
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **COSMOS_BRIDGE_ANALYSIS.md** | 15K | migration | SDK migration path |
| **COSMOS_GENESIS_AUDIT.md** | 21K | genesis | Genesis configuration |
| **IBC_TRANSFER_REVIEW.md** | 19K | IBC | IBC integration |
| **INTERCHAIN_ACCOUNT_REVIEW.md** | 20K | ICA | Inter-Chain Accounts |
| **MODULE_UPGRADEABILITY_REVIEW.md** | 34K | modules | Module upgrade paths |
| **MODULE_SERVICE_REGISTRATION_REVIEW.md** | 14K | gRPC | Service registration |
| **UPGRADEABILITY_PP_REVIEW.md** | 28K | upgrades | Full upgrade strategy |
| **CRISIS_MODULE_REVIEW.md** | 19K | x/emergency | Emergency response |

### Collaboration & Game Theory
| File | Size | Module | Key Findings |
|------|------|--------|-------------|
| **PP_COLLABORATION_GAME_THEORY_REVIEW.md** | 36K | cross-module | 67+ mechanisms, 46/50 score |

---

## DevNet Testing

| File | Size | Purpose |
|------|------|---------|
| **DEVNET_TESTING.md** | 23K | Master test plan (54 tests, 6 phases) |
| **DEVNET_TESTING_STRATEGY.md** | 20K | Strategy (6 scenarios A-F) |
| **DEVNET_POT_TESTING_STRATEGY.md** | 27K | PoT-specific testing |
| **DEVNET_TESTING_DIMENSIONS.md** | 24K | 6-dimensional test matrix |
| **DEVNET_DIMENSIONAL_TESTING.md** | 33K | Detailed dimensional tests |
| **DEVNET_PROGRESS_CHECK.md** | 10K | Historical snapshot (2026-02-12) |
| **DEVNET_TEST_RESULTS.md** | 9K | Test results (64/64 pass) |
| **PP_DEVNET_INSIGHTS.md** | 10K | Insights from DevNet testing |

---

## File Count Summary

| Category | Files | Total Size |
|----------|-------|------------|
| Quick Start | 5 | ~69K |
| Insight & Wisdom | 6 | ~57K |
| Strategic Planning | 6 | ~107K |
| Security & Adversarial | 8 | ~144K |
| Deep Technical Reviews | 27 | ~617K |
| DevNet Testing | 8 | ~156K |
| **Total** | **60** | **~1.15MB** |

> Note: 17 files from the original 77 were consolidated (3 TOKEN_EXPIRATION files → 1) or are superseded but retained as historical record.

---

_Last updated: 2026-02-17_
