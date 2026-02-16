# Go/Cosmos SDK Chain — Critical Component Analysis

> Generated: 2026-02-16 | Source: Deep exploration of `app/`, `x/*/`, `cmd/legbled/`
> Chain: SDK v0.50.10, CometBFT v0.38, IBC-Go v8.5.1

---

## 1. CONSENSUS-CRITICAL COMPONENTS

### 1.1 ABCI++ Vote Extension Pipeline
**Files**: `app/abci.go`, `app/vote_extensions.go`

The mechanism by which PoT verification rounds integrate into CometBFT consensus. Replaces standard BFT voting with custom commit-reveal.

**5-step pipeline**:
1. `ExtendVoteHandler` (vote_extensions.go:41): VRF generation → selection check → commitment hash → salt stored in `LocalCommitmentStore`
2. `PrepareProposalHandler` (abci.go:131): Proposer collects vote extensions → serializes `VoteExtInjection` pseudo-tx at position 0 (prefix `{0x00, 'V', 'E', 'X'}`)
3. `ProcessProposalHandler` (abci.go:210): All validators validate injection format + reject non-emergency txs during halt
4. `PreBlocker` (app.go:1333): Processes injection → `KnowledgeKeeper.StoreCommitmentInRound()` / `StoreRevealInRound()`
5. `BeginBlocker` (knowledge module): Phase transitions advance rounds

**Commitment hash**: `SHA-256("LGM.commitment.v1:" || len4(roundID) || roundID || len4(verdict) || verdict || confidence8 || len4(salt) || salt)` — domain-separated, length-prefixed

**Risks**:
- Vote extensions serialized as JSON — no proto encoding, no cross-version guarantee
- **CRITICAL**: `LocalCommitmentStore` is in-memory only (app.go:1280) — node restart between commit/reveal loses all pending commitments → missed reveal slashes
- Gas estimation rough: `estimatedGas = uint64(len(tx)) * 10` (abci.go:185)
- Race window between halt finalization and next block in emergency checks

**Dependencies**: `KnowledgeKeeper`, `EmergencyKeeper`, `VoteExtensionConfig`

---

### 1.2 VRF Validator Selection
**File**: `x/knowledge/crypto/vrf.go`

Determines which validators participate in each verification round. Broken VRF = no consensus or no randomness.

**Implementation**: ECVRF using Ed25519 (simplified RFC 9381):
- `GenerateVRF` (line 35): seed → domain hash → `hash_to_curve` → `Gamma = sk * H` → deterministic nonce → proof `(Gamma, c, s)` 96 bytes
- `VerifyVRF` (line 103): Recomputes U'/V' from proof, verifies challenge match
- `IsValidatorSelected` (line 171): `big.Int` comparison `outputNum * totalStake < stake * targetCount * 2^64` — handles uint64 overflow correctly

**Risks**:
- `hashToCurve` (line 273): Uses `append` for trial-and-error (up to 256 attempts) — potential slice aliasing
- `computeChallenge` (line 297): Takes only 24 bytes of SHA-512 — non-standard, not audited against RFC 9381
- **HIGH**: No dedicated test file for `x/knowledge/crypto/vrf.go`

**Dependencies**: `filippo.io/edwards25519`, standard crypto

---

### 1.3 Knowledge Keeper — PoT Core
**Files**: `x/knowledge/keeper/keeper.go`, `rounds.go`, `phases.go`, `confidence.go`

The knowledge module IS the PoT consensus engine: verification rounds, phase transitions, stake-weighted aggregation, rewards/slashes.

**Confidence aggregation** (confidence.go:21):
- Stake weights adjusted by `captureDefenseKeeper.GetReputationScore()` (domain reputation)
- Weighted verdicts: `weightedConf = reveal.Confidence * stake / 1_000_000`
- Thresholds: `>= AcceptanceThreshold` → accepted/rejected; `>= 55%` → contested; else inconclusive
- Stratum ceiling via `ontologyKeeper.GetConfidenceCeiling()` or hardcoded fallback
- Category ceiling from `CATEGORY_DECAY` table
- Evidence bonus from `evidenceMgmtKeeper` (additive, capped at 15%)
- Method diversity adjustment (AH-4): penalizes uniform method declarations

**Reward calculation** (confidence.go:207):
- `baseReward` from params (fallback 100,000 ulgm)
- Difficulty bonus: `reward + reward*(1_000_000 - confidence)/1_000_000`
- Wrong verdict slash: `stake * 60,000 / 1_000_000` (6%)
- Missed reveal slash: `stake * 100 / 1_000_000` (0.01%)
- Research tax: 22% deducted before validator payout

**Risks**:
- Integer overflow if `reveal.Confidence = 1_000_000` and `stake > uint64_max / 1_000_000` (~18.4M ulgm) — practically safe but undocumented
- **HIGH**: Research tax failure (confidence.go:318) silently bypasses tax — validator gets full reward
- `captureDefenseKeeper` nil guards mean capture defense is silently disabled without indication

**BeginBlocker ordering**: Knowledge runs LAST (app.go:1124) — depends on staking, tokens, qualification, capture_defense, emergency

**Test coverage**: 16+ tests in `x/knowledge/keeper/keeper_test.go`

---

### 1.4 BeginBlocker/EndBlocker Ordering
**File**: `app/app.go` lines 1042-1190

Wrong ordering → silent state corruption or consensus failures.

**Key dependency chain**:
```
upgrade → ALL
emergency → capture_defense (halt status)
vesting_rewards → staking (minted rewards available for unbonding credits)
tokens → staking (period sweep before OriginPeriod credits)
staking → knowledge (validator set must be current for round phases)
ontology → qualification → knowledge
knowledge → LAST
```

**InitGenesis ordering**: auth → staking → vesting_rewards → tokens → emergency → ... → knowledge (LAST)

**Risk**: No automated ordering enforcement — developer adding a module could break dependencies silently. Detailed inline comments help but are not machine-checked.

---

## 2. SECURITY-CRITICAL COMPONENTS

### 2.1 Ante Handler Chain (16 decorators)
**Files**: `app/ante.go`, `app/ante_legible.go`

**Full chain** (ante.go:25-56):
1. `ibcante.NewRedundantRelayDecorator` — IBC duplicate protection
2. `ante.NewSetUpContextDecorator` — gas meter init (MUST be first)
3. `NewEmergencyHaltDecorator` — blocks non-emergency txs when halted
4. `NewLGMGasDecorator` — gas limit validation, per-tx cap, min gas price
5. `NewFeeRouterDecorator` — fee split intent logging
6. `ante.NewExtensionOptionsDecorator`
7. `ante.NewValidateBasicDecorator`
8. `ante.NewTxTimeoutHeightDecorator`
9. `ante.NewValidateMemoDecorator`
10. `ante.NewConsumeGasForTxSizeDecorator`
11. `ante.NewDeductFeeDecorator`
12. `ante.NewSetPubKeyDecorator`
13. `ante.NewValidateSigCountDecorator`
14. `ante.NewSigGasConsumeDecorator`
15. `ante.NewSigVerificationDecorator` — **signature verification**
16. `ante.NewIncrementSequenceDecorator`
17. `NewLegibleDIDDecorator` — DID validation (post-auth)
18. `NewLegibleAccountDecorator` — frozen check + activity tracking (post-auth)
19. `NewLegibleCapabilityDecorator` — session key capability enforcement (post-auth)

**Session key capability mapping** (ante_legible.go:206-240):
- `isTransferMsg`: 3 types | `isStakeMsg`: 4 types | `isClaimMsg`: 4 types
- `isVoteMsg`: 5 types | `isICAMsg`: 2 types
- **NOT covered**: disputes, research, home, billing, etc. — session keys cannot perform these

**Risks**:
- `LGMGasDecorator` skips all checks in simulate mode
- `LegibleAccountDecorator` updates `LastActiveBlock` even on handler failure (documented as intentional)
- New message types without capability mapping silently deny session keys (default-deny)

---

### 2.2 Emergency Kill Switch
**Files**: `x/emergency/keeper/ceremony.go`, `msg_server.go`

**3-phase BFT protocol**:
- **Halt**: 75% Guardian stake quorum (prevote + precommit)
- **Revert**: 80% quorum
- **Resume**: 80% quorum
- Anti-abuse: `MaxProposalsPerGuardianPerEpoch`, `MaxProposalsPerEpoch`, `CooldownBlocks`
- Auto-resume: `CheckHaltExpiry()` prevents permanent halt from guardian inaction

**CRITICAL design gap** (ceremony.go:376-390): Revert ceremony is a **governance signal only** — no automated state rollback. Validators must manually coordinate out-of-band.

**Test coverage**: 30 tests, all passing

---

### 2.3 Auth/Identity Keeper
**Files**: `x/auth/keeper/keeper.go`, `msg_server.go`

Manages DID resolution, key rotation, session keys, account freezing.

**Key operations**: `RegisterAccount`, `RotateKey`, `CreateSession`, `GetAddressForDID`

**Risks**:
- JSON serialization for all LegibleAccount data (not proto) — hand-written types panic with gogoproto codec
- Session key expiry is block-height based (safe against clock drift)

**Test coverage**: 38 tests, all passing

---

## 3. ECONOMIC-CRITICAL COMPONENTS

### 3.1 Vesting Rewards — Block Reward Engine
**Files**: `x/vesting_rewards/keeper/rewards.go`, `keeper.go`

**Finite fund model** (rewards.go:141):
- Fund pre-minted at genesis; no new minting
- `R = max(initialReward * decayFactor^epoch * fundHealth, floorReward)`, capped at available balance
- Decay via integer exponentiation-by-squaring (correct `big.Int` implementation)
- Validator participation scaling: `effectiveReward * min(active, minForFull) / minForFull`
- Empty block reward: 0% (PoT alignment)

**Fee routing** (rewards.go:69): `RouteFees()` sweeps 22% of `fee_collector` → `research_fund` in BeginBlocker, BEFORE `x/distribution`

**FalsifyClaim** (rewards.go:362): Claws back 33% of released + forfeits unvested + reserve → all to challenger

**Risks**:
- **HIGH**: `RouteFees()` failure logged as warn but not propagated — fee routing silently fails
- No mechanism to refill fund if exhausted prematurely

---

### 3.2 Staking — Validator Tiers & Slashing
**File**: `x/staking/keeper/keeper.go`

**4-tier system**:
| Tier | Name | Min Stake | Reward Mult | Categories |
|------|------|-----------|-------------|------------|
| 0 | Apprentice | 0 (virtual 11 LGM) | 0.1x | protocol/analytic/formal |
| 1 | Verified | 22 verifications @ 77% | 0.5x | +empirical/historical |
| 2 | Bonded | 1,111 LGM | 1.0x | all |
| 3 | Guardian | 11,111 LGM + 1,111 verifications + 77% + 0 slashes | 1.5x | all + governance |

**SlashValidator** (keeper.go:532): Per-epoch cap, progressive escalation, autopoiesis SSI multiplier (up to 3x under stress), slash floor, deactivation after `MaxSlashCountDeactivate`

**Risks**:
- **CRITICAL**: `BurnCoins` failure logged but not propagated — state says slashed, tokens not burned
- `CountBlockProducers` iterates ALL validators every call — O(n) per block
- `SlashesThisEpoch` reset only at epoch boundaries

---

### 3.3 Gas Costs
**File**: `app/gas.go`

| Constant | Value |
|----------|-------|
| BlockGasLimit | 33,333,333 |
| TxGasLimit | 11,111,111 |
| MinGasLimit | 22,222 |
| MinGasPrice | 1 ulgm/gas |
| ResearchTaxBPS | 220,000 (22%) |
| ValidatorFeeBPS | 780,000 (78%) |
| BurnRateBPS | 50,000 (5%) |

**Risk**: Many message types (Agent Home 10, Qualification 4, Auth recovery 8) fall back to `MinGasLimit` (22,222) — cheap gas for novel tx types

---

## 4. SAFETY-CRITICAL COMPONENTS

### 4.1 Emergency Halt Guard
**Files**: `app/ante_legible.go:28`, `app/abci.go:136`

Three enforcement points:
1. **Ante handler**: Rejects non-emergency txs → `ErrChainHalted` (no gas consumed)
2. **PrepareProposal**: Filters mempool to emergency-only during halt
3. **ProcessProposal**: Rejects proposals with non-emergency txs during halt

All use `emergencytypes.IsEmergencyMsg(msg)` — correctness is critical.

### 4.2 Autopoiesis — Self-Regulating Parameters
**File**: `x/autopoiesis/keeper/keeper.go`

Dynamically adjusts system parameters (slash mult, block reward, claim stake, decay rates) based on chain health.

**OBSERVE → COMPUTE → ADAPT cycle** (every 1,111 blocks):
- 5 hormones: SSI (system stress), CRI (knowledge retention), DVI (domain velocity), VCF (verification cascade), GRM (growth rate)
- Priority conflict resolution: SSI(4) > CRI(3) > DVI/VCF(2, averaged) > GRM(1)
- Dampened deltas prevent oscillation
- Bounds enforcement per path

**Risks**:
- `collectMetrics` calls many knowledge keeper methods every epoch — performance at scale
- Epoch key format `%016d` overflows at 10^17 epochs (not near-term)

### 4.3 IBC Rate Limiting
**File**: `app/app.go:637-657`

Middleware stack: `TransferKeeper → IBCFeeKeeper → RateLimitICS4Wrapper → IBCKeeper.ChannelKeeper`

---

## 5. STATE-CRITICAL COMPONENTS

### 5.1 Genesis & Store Initialization
**File**: `app/app.go:1296-1327`

**Critical bug fixed**: GoLevelDB empty-value IAVL incompatibility
- IAVL `SaveEmptyRoot` writes `[]byte{}`, GoLevelDB `Get` returns `nil`
- Fix: `ensureNonEmptyStores()` writes sentinel `__store_initialized → {0x01}` to all KV stores
- Runs in `InitChainer` after `ModuleManager.InitGenesis()`

### 5.2 SafeServiceRegistrar
**File**: `app/app.go:1602-1619`

Wraps gRPC router with panic recovery for hand-written service descriptors.

**Risk**: Failed registrations silently skipped — custom module endpoints unreachable until proto codegen

### 5.3 State Migrations
Migrator files exist for every module (all new, `??` in git status). Critical for future chain upgrades.

---

## 6. CROSS-MODULE DEPENDENCY MAP

```
BankKeeper ←──────────── almost every module
LegibleAuthKeeper ←────── Staking, Knowledge (via adapter), ante chain
LegibleStakingKeeper ←─── Knowledge, Emergency, Gov, VestingRewards,
                           CaptureDefense, CaptureChallenge, Partnerships,
                           Disputes, Autopoiesis (bidirectional)
TokensKeeper ←─────────── Staking, Knowledge
KnowledgeKeeper ←──────── EvidenceMgmt (bidirectional), Billing, BVM (adapter),
                           Autopoiesis, Disputes
OntologyKeeper ←────────── Knowledge (via SetOntologyKeeper), Qualification
QualificationKeeper ←────── Knowledge (via hooks), CaptureDefense
EmergencyKeeper ←──────── CaptureDefense, Autopoiesis, ante chain, ABCI
VestingRewardsKeeper ←──── Disputes (via SetVestingKeeper)
AutopoiesisKeeper ←──────── Staking (via SetAutopoiesisKeeper)
```

**Circular dependency pattern**: Broken via `Set*Keeper()` post-construction wiring (7 instances in app.go:891-932). All cross-module keepers nil-guarded with `if k.someKeeper != nil`.

---

## 7. BVM (Bytecode Virtual Machine)
**File**: `x/bvm/vm/interpreter.go` (2,072 lines)

Executes arbitrary smart contracts on-chain.

**Key protections**:
- `MaxCallDepth` check (line 48): Prevents recursive call DoS
- `MaxBytecodeSize` (line 53): 24KB limit (EIP-170)
- Pre-scanned `JUMPDEST` table (line 58): Prevents arbitrary jumps
- `ModeStaticCall` inherited by sub-calls (line 70): DELEGATECALL can't escape static mode
- `GasMeter.Consume` per opcode (line 76): Gas before execution

**Test coverage**: `vm_test.go`, `interpreter_test.go`, `calls_test.go`, `knowledge_test.go`

---

## 8. GOVERNANCE — DUAL SYSTEM
**File**: `app/app.go:934-970`

| System | Store | Purpose |
|--------|-------|---------|
| SDK x/gov | `gov` | IBC params, Cosmos ecosystem compat, text/parameter/upgrade |
| LGM x/gov | `lgm_gov` | LIPs, quadratic voting, treasury disbursements, upgrade scheduling |

Bridge: `LGMGovUpgradeAdapter` (app.go:741)

**Risk**: No enforcement at message routing level for which system governs which parameters.

---

## 9. TEST COVERAGE

| Module | Tests | Status |
|--------|-------|--------|
| x/knowledge/keeper | 16+ | Passing |
| x/emergency/keeper | 30 | Passing |
| x/auth/keeper | 38 | Passing |
| x/gov/keeper | 37 | Passing |
| x/autopoiesis (types+keeper) | 73 | Passing |
| x/staking/keeper | Exists | Passing |
| x/vesting_rewards/keeper | Exists | Passing |
| x/evidence_mgmt/keeper | Exists | Passing |
| x/home/keeper | Exists | Passing |
| x/schedule/keeper | Exists | Passing |
| x/bvm/vm | 4 test files | Comprehensive |
| app/ | ante, vote_ext, store | Exists |
| **x/knowledge/crypto/vrf.go** | **None** | **GAP** |

---

## 10. PRIORITY RISK REGISTER

### Critical (Production-blocking) — ALL RESOLVED (2026-02-16)
| ID | Risk | Fix | Status |
|----|------|-----|--------|
| C1 | `LocalCommitmentStore` in-memory only | Write-through disk persistence via `{dataDir}/pot_commitments.json` with atomic rename | **FIXED** |
| C2 | Emergency revert is signal-only | `StatusReverting` state + queryable `RevertTarget` + ERROR-level monitoring every block + auto-resume safety valve | **FIXED** |
| C3 | `BurnCoins` failure not propagated | `SlashValidator` now returns error on burn failure, aborting the slash atomically | **FIXED** |

### High (Pre-mainnet required) — ALL RESOLVED (2026-02-16)
| ID | Risk | Fix | Status |
|----|------|-----|--------|
| H1 | VRF crypto has no tests | Expanded `vrf_test.go`: 20 tests covering determinism, proof tampering (Gamma/c/s), 32-byte key path, boundary values, max uint64 overflow, aliasing regression | **FIXED** |
| H2 | Research tax failure bypassed silently | Tax failure now withholds entire reward (`continue`) instead of bypassing tax — module retains funds | **FIXED** |
| H3 | `MinGasLimit` fallback for unknown msgs | Added 69+ missing proto URL mappings to `msgTypeURLToGas` + 54 new `TransactionGasCosts` entries for discovery, partnerships, ontology, schedule, billing, governance extras | **FIXED** |
| H4 | gRPC silently unregistered | `SafeServiceRegistrar` now logs at ERROR level, tracks `FailedServices` list for diagnostics | **FIXED** |
| H5 | GoLevelDB sentinel is workaround | Added root-cause documentation + removal conditions (upstream fix or backend switch) | **DOCUMENTED** |

### Medium (Hardening) — ALL RESOLVED (2026-02-16)
| ID | Risk | Fix | Status |
|----|------|-----|--------|
| M1 | `hashToCurve` append aliasing | Replaced `append(input, byte(attempt))` with explicit `make+copy` buffer — regression test in `TestHashToCurve_DoesNotAliasInput` | **FIXED** |
| M2 | Non-standard VRF challenge | Corrected misleading comment: 24 bytes (192 bits) < ℓ (≈2^252) so Mod() never wraps — safe but restricted to [0, 2^192) | **DOCUMENTED** |
| M3 | No machine-checked BeginBlocker ordering | Created `app/blocker_order_validator.go` with 15 dependency edges, called at app init — panics on violation | **FIXED** |
