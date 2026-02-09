# Cosmos Genesis Launch Audit

> **Created**: 2026-02-08
> **Updated**: 2026-02-09 (Rev 3 — vesting_rewards wired, P0-2 confirmed fixed, staking decimals fixed)
> **Scope**: All x/ modules, app.go wiring, genesis CLI, config defaults, cross-module flows
> **Status**: 0 P0 remaining, 4 P1 open, 6 P2 open

---

## Summary

The Go chain (`legbled`) has 13 custom modules. Seven are wired into `app.go` (auth, staking,
tokens, emergency, gov, research, knowledge). Six are defined but not wired (ontology, disputes,
evidence_mgmt, vesting_rewards, bvm, channels, claiming_pot, tree).

**Overall Genesis Readiness: 70% for DevNet, 40% for TestNet, 20% for MainNet.**
*(Up from 60/35/15 — decimal fixes and module additions have improved posture)*

**Progress since Rev 1**: 10 of 12 decimal mismatches fixed. x/gov and x/research fully
implemented with proper 6-dec ulgm. x/vesting_rewards types expanded with 7 msg handlers.
x/research wired into app.go ModuleManager. Emergency decimals corrected.

---

## Rev 2 Change Log (2026-02-09)

| Original Finding | Rev 1 Status | Rev 2 Status | Action |
|-----------------|--------------|--------------|--------|
| P0-1: Decimal mismatch (12 fields) | 12 WRONG | **2 WRONG** (10 fixed) | Downgraded to P1 |
| P0-2: ModuleBasics missing customs | OPEN | **FIXED (Rev 3)** | All 14 custom modules in ModuleBasics |
| P0-3: Unwired modules | 4 unwired | **FIXED (Rev 3)** | All modules now wired including vesting_rewards |
| P0-4: Domain conflict | OPEN | **DE-RISKED** | Ontology not wired = no runtime conflict |
| P1-1: migrate-genesis | Stub | **Stub** | Unchanged |
| P1-2: Reward/penalty gaps | OPEN | **OPEN** | Unchanged |
| P1-3: ResearchTreasuryAddress | Empty | **Empty** | Unchanged |
| P1-4: Vote extensions | Not wired | **Not wired** | Unchanged |
| P1-5: Supply cap | Missing | **Missing** | Unchanged |
| — | — | **P1-6: FIXED (Rev 3)** | vesting_rewards fully wired into app.go |
| — | — | **NEW P2-5** | Staking tier comment self-contradiction |
| — | — | **NEW P2-6** | x/vesting_rewards gRPC Handler: nil |

---

## P0 — Launch Blockers (0 remaining — ALL FIXED)

### P0-2: Custom Modules Missing from ModuleBasics — FIXED (Rev 3)

All 14 custom Legible modules are now registered in `ModuleBasics` (app/app.go:210-223),
including vesting_rewards added in Rev 3.

**Impact**: `app.DefaultGenesis()` calls `ModuleBasics.DefaultGenesis()` which will NOT include
any custom module defaults. Running `legbled init` produces a genesis file missing all
Legible-specific params, domains, tier configs, etc.

**Fix**: Add custom module `AppModuleBasic{}` implementations and register them in `ModuleBasics`.

**Note**: The custom modules ARE in ModuleManager (`app/app.go:582-588`) and `SetOrderInitGenesis()`
(lines 644-650), so if genesis JSON includes their state, initialization works. But
auto-generated genesis is incomplete.

---

### ~~P0-1: Decimal Mismatch~~ → Downgraded to P1-7 (2 remaining fields)

**10 of 12 original fields FIXED** since Rev 1. Only 2 remain wrong:

| Module | Field | Current Value (WRONG) | Correct (6-dec) | Line |
|--------|-------|----------------------|------------------|------|
| `x/staking` | `DefaultTierConfigs[TierBonded].MinStake` | `"1111000000000000000000"` | `"1111000000"` | :73 |
| `x/staking` | `DefaultTierConfigs[TierGuardian].MinStake` | `"11111000000000000000000"` | `"11111000000"` | :83 |

**Fixed fields** (verified via direct file read):
- `x/emergency/types/types.go:218` — MinGuardianStake: `"111111000000"` (6-dec) ✅
- `x/emergency/types/types.go:224` — CouncilVirtualStake: `"11111000000"` (6-dec) ✅
- `x/staking/types/types.go:167` — Params.VirtualStake: `"111000000"` (6-dec) ✅
- `x/gov/types/types.go:155` — BaseCostPerVote: `"1000000"` (6-dec) ✅
- `x/gov/types/types.go:159` — DisbursementStake: `"2000000000"` (6-dec) ✅
- `x/gov/types/types.go:163-166` — All 4 LIPStake* values: 6-dec ✅
- `x/research/types/types.go:134-140` — All 3 research values: 6-dec ✅
- `x/vesting_rewards/types/genesis.go:10` — BlockRewardAmount: `"11000000"` (6-dec) ✅

**Impact of remaining 2**: Tier progression from Verified(1) to Bonded(2) requires
"1111000000000000000000" ulgm instead of "1111000000" ulgm — that's 1.111 * 10^12 LGM instead
of 1,111 LGM. No validator can ever reach Bonded or Guardian tier. VRF selection works only for
Apprentice and Verified tiers.

---

### ~~P0-3: Unwired Modules~~ → Split: P2-4 (stubs) + NEW P1-6 (vesting_rewards)

4 modules with real genesis implementations remain unwired:

| Module | Has Genesis | Implementation Depth | In ModuleManager | Risk |
|--------|-------------|---------------------|------------------|------|
| `x/vesting_rewards` | Yes (10 cats, 7 msg types, tests) | **Full** | No | **P1** — No block rewards |
| `x/ontology` | Yes (7 strata, 7 domains) | Moderate | No | P2 — Deferred OK |
| `x/disputes` | Yes (3 tiers, params) | Moderate | No | P2 — Deferred OK |
| `x/evidence_mgmt` | Yes (params, oracles) | Moderate | No | P2 — Deferred OK |

**Key insight**: `x/vesting_rewards` has been fully implemented (types.go with 7 msg types,
codec.go with gRPC registration, msg_server.go with 7 handlers, keeper_test.go with tests) but
is NOT wired. This means **block reward distribution is completely dead** — no token emission,
no research fund routing, no truth-linked vesting. See P1-6.

---

### ~~P0-4: Domain List Conflict~~ → Downgraded to P2 design debt

**De-risked**: Since `x/ontology` is not wired, its InitGenesis never runs, so there's no
runtime domain conflict. Knowledge module's 4 domains (mathematics, physics, computer_science,
general) are the only active domains. If/when ontology is wired, the conflict must be resolved.

---

## P1 — Critical Issues (6 open)

### P1-1: `migrate-genesis` Not Implemented — UNCHANGED

`cmd/legbled/cmd/genesis.go:131` — Still returns `"migrate-genesis is not yet implemented (Phase 5)"`.
Blocks any mainnet launch that builds on TS devnet state.

### P1-2: Reward/Penalty Imbalances — UNCHANGED

**Disputes** (`x/disputes/types/genesis.go:64-66`):
- Loser slashed: 50% of bond
- Winner reward: 30% of slash = 15% of bond
- Arbiter reward: 20% of slash = 10% of bond
- **Remaining 50% of slash (25% of bond)**: No destination specified

**Evidence** (`x/evidence_mgmt/types/genesis.go:28-29`):
- Slash rate: 10%, Reward rate: 5%
- **5% gap**: No destination specified

**Note**: Both modules are unwired (P0-3), so this doesn't affect DevNet yet. But must be
resolved before wiring.

### P1-3: Empty ResearchTreasuryAddress — UNCHANGED

`x/vesting_rewards/types/genesis.go:9`:
```go
ResearchTreasuryAddress: "" // set in genesis
```
No validation in `Validate()` (genesis.go:45-61) checks this is non-empty. Combined with P1-6
(vesting_rewards unwired), research fund routing is completely non-functional.

### P1-4: Vote Extensions Not in Genesis Config — UNCHANGED

`cmd/legbled/cmd/config.go:32-34` — Comment-only documentation for
`vote_extensions_enable_height = "1"`. No code sets this in genesis params. ABCI++ vote extension
handlers exist (`app/vote_extensions.go`) but won't activate without this config.

### P1-5: No Total Supply Guard — UNCHANGED

`cmd/legbled/cmd/genesis.go:84-89` — `add-genesis-account` adds coins to bank genesis without
any supply cap validation.

### P1-6: vesting_rewards Not Wired — **NEW**

`x/vesting_rewards` has a complete implementation:
- `types/types.go`: 10 vesting categories, 7 msg types, Params with BlockRewardAmount
- `types/codec.go`: RegisterCodec, RegisterInterfaces, 7-method gRPC ServiceDesc
- `keeper/msg_server.go`: 7 handlers (ClaimVestedRewards, FalsifyClaim, CreateVesting, PauseVesting, ResumeVesting, AccelerateVesting, CompleteVesting)
- `keeper/keeper_test.go`: Tests passing
- `module.go`: RegisterServices, BeginBlocker, EndBlocker, genesis

**But NOT in `app/app.go` ModuleManager** (lines 563-589).

**Impact**: This is the **token emission engine**:
- Block rewards (11 LGM/block) are not being distributed
- 22% research fund share is not being routed
- Truth-linked vesting (the core PoT economic incentive) is inactive
- Falsification clawback mechanism is unreachable

Without this module, validators have **zero incentive** to participate at DevNet launch.

**Fix**: Wire into app.go. Init order should be after legible_staking, before knowledge:
```
legible_staking → vesting_rewards → tokens → emergency → lgm_gov → research → knowledge
```

### P1-7: Staking Tier MinStake Decimals (Residual from P0-1) — **RECLASSIFIED**

2 values in `x/staking/types/types.go` `DefaultTierConfigs()`:
- Line 73: TierBonded `"1111000000000000000000"` → should be `"1111000000"`
- Line 83: TierGuardian `"11111000000000000000000"` → should be `"11111000000"`

**Impact**: Tier advancement to Bonded/Guardian impossible. Governance (Guardian-only) disabled.
Emergency halt (Guardian-only) disabled. PoT effectively runs on Apprentice+Verified only.

---

## P2 — Medium Issues (6 open)

### P2-1: Weak Genesis Validation — UNCHANGED

| Module | Validation Quality | Gap |
|--------|--------------------|-----|
| `x/knowledge` | 1 param checked (CrossStratumDiscount) | All other params unvalidated |
| `x/research` | Returns nil (no validation) | No validation at all |
| `x/emergency` | Quorum + timeout only | Stake format unchecked |
| `x/staking` | 3 basic checks | VirtualStake/tier MinStake not validated |
| `x/tokens` | Good (dedup, type) | RAI bounds unchecked |
| `x/disputes` | Delegates to Params.Validate() | Need to verify |
| `x/vesting_rewards` | Good (bps, half-life, category count) | ResearchTreasuryAddress unchecked |

### P2-2: Vesting Category Config Mismatch — UNCHANGED

`x/vesting_rewards` defines 10 epistemic categories:
```
axiomatic, formal_proof, on_chain, cryptographic, computational,
peer_reviewed, replicated, oracle_feed, attestation, contested
```

`x/knowledge` validates 9 categories:
```
protocol, analytic, formal, empirical, historical,
testimonial, revelation, computational, replicated
```

Only 2 overlap (computational, replicated). Claims verified with knowledge categories would need
mapping to vesting categories for reward scheduling. No mapping logic exists.

### P2-3: `add-genesis-account` Missing Safety Checks — UNCHANGED

- No minimum balance check
- No address format validation beyond bech32
- No cap on number of genesis accounts

### P2-4: Stub Modules Return Empty JSON

`x/bvm`, `x/channels`, `x/claiming_pot`, `x/tree` all return `json.RawMessage("{}")` as
DefaultGenesis. Not harmful if they remain unwired, but should be documented as Phase 3+.

### P2-5: Staking Tier Comment Self-Contradiction — **NEW**

`x/staking/types/types.go:73` reads:
```go
MinStake: "1111000000000000000000", // 1111 LGM in ulgm (18 decimals)
```

The comment acknowledges "18 decimals" while claiming the value is "in ulgm" (which is 6 decimals).
This suggests awareness of the discrepancy during development. The inconsistency between the
comment and intended convention makes it clear this is a copy-paste from the TypeScript codebase
where 18-decimal bigints are the norm.

### P2-6: vesting_rewards gRPC Handler: nil — **NEW**

`x/vesting_rewards/types/codec.go:48-54` — All 7 method entries in `_Msg_serviceDesc` have
`Handler: nil`. While this is the standard pattern (SDK fills handlers at registration time),
it means the service descriptor was hand-written rather than protobuf-generated. If
`RegisterMsgServer` doesn't properly wire the `msgServer` implementation, all vesting messages
will silently fail at the gRPC layer. This should be tested with an integration test after
wiring.

---

## Cross-Module Flow Analysis (NEW in Rev 2)

### Token Emission Flow (BROKEN)

```
Expected:  BlockProducer → vesting_rewards.BeginBlocker → mint 11 LGM →
           22% → ResearchTreasury, 78% → ProducerReward → validator bank

Actual:    vesting_rewards NOT WIRED → no minting occurs → validators unpaid
           ResearchTreasuryAddress = "" → even if wired, 22% route fails
```

**Blocks**: P1-6 (wiring) + P1-3 (treasury address)

### Staking → Knowledge Flow (PARTIALLY WORKING)

```
Expected:  Validator stakes → tier assigned → VRF selection → verification round →
           confidence aggregation → rewards/slashes

Actual:    Staking wired ✅ → tier config wrong (P1-7) → VRF selection limited to
           Apprentice+Verified → rounds work ✅ → aggregation works ✅ →
           rewards call TokensKeeper.DebitBalance (nil-safe, optional) ✅
```

**Blocks**: P1-7 (tier MinStake decimals)

### Emergency Flow (WORKING with Guardian limitation)

```
Expected:  Guardian proposes halt → BFT vote → 75% quorum → chain halts

Actual:    Emergency wired ✅ → decimals fixed ✅ → BUT no one can reach
           Guardian tier (P1-7) → emergency system unreachable
```

**Blocks**: P1-7 (tier MinStake decimals)

### Governance Flow (WORKING)

```
Expected:  Submit LIP → stake → review → voting → quadratic votes → tally

Actual:    Gov wired ✅ → decimals fixed ✅ → all 9 msg handlers implemented ✅ →
           37 tests passing ✅ → FULLY FUNCTIONAL at genesis
```

### Research Flow (WORKING)

```
Expected:  Submit research → stake → peer review → accept/reject → rewards

Actual:    Research wired ✅ → decimals fixed ✅ → 9 msg handlers implemented ✅ →
           bank transfers use sdkmath.NewIntFromBigInt ✅ → FULLY FUNCTIONAL
```

---

## Module Implementation Depth (Rev 2 Assessment)

| Module | Types | Keeper | MsgServer | gRPC Query | Tests | Wired | Genesis Valid |
|--------|-------|--------|-----------|------------|-------|-------|--------------|
| x/knowledge | Full | Full | 10 msgs | 8 queries | 16 | ✅ | Weak |
| x/auth | Full | Full | 4 msgs | 4 queries | 38 | ✅ | N/A |
| x/staking | Full | Full | msgs impl | — | Yes | ✅ | Weak |
| x/tokens | Full | Full | msgs impl | — | — | ✅ | Good |
| x/emergency | Full | Full | 6 msgs | 4 queries | 30 | ✅ | Moderate |
| x/gov | Full | Full | 9 msgs | 4 queries | 37 | ✅ | Moderate |
| x/research | Full | Full | 9 msgs | 4 queries | Yes | ✅ | **None** |
| x/vesting_rewards | Full | Full | 7 msgs | — | Yes | **NO** | Good |
| x/ontology | Full | Partial | — | — | — | **NO** | Moderate |
| x/disputes | Full | Partial | — | — | — | **NO** | Moderate |
| x/evidence_mgmt | Full | Partial | — | — | — | **NO** | Moderate |

---

## BeginBlocker / EndBlocker Ordering Analysis (NEW in Rev 2)

### BeginBlocker Order (`app/app.go:592-608`)

```
Standard SDK:
  upgrade → mint → distr → slashing → evidence → staking → ibc → ibctransfer

Custom Legible:
  emergency     — ceremony progress (MUST run before knowledge)
  tokens        — sweep expired tokens before unbonding returns
  legible_staking — process mature unbondings
  lgm_gov       — check voting deadlines
  research      — expire bounties past deadline
  knowledge     — round phase transitions (LAST — correct)
```

**Verdict**: Ordering correct. Emergency before knowledge ensures halt blocks verification.
Tokens before staking ensures expired token sweep before unbonding credit.

### EndBlocker Order (`app/app.go:610-624`)

```
Standard SDK:
  crisis → gov → staking → ibc → ibctransfer → feegrant

Custom Legible:
  legible_staking — tier transition safety checks
  knowledge       — round cleanup
  tokens          — post-block period accounting
  emergency       — reset epoch counters
  lgm_gov         — finalize governance
  research        — (no documented end-block logic)
```

**Note**: `vesting_rewards` is missing from both Begin and EndBlocker because it's unwired (P1-6).
When wired, it should run in BeginBlocker **before** `legible_staking` so block rewards are
minted before staking reward distribution.

---

## Recommended Fix Priority (Updated)

### Before DevNet (Immediate — 2 items)

1. **Fix 2 remaining tier MinStake decimals** in `x/staking/types/types.go` (P1-7)
   - Line 73: `"1111000000000000000000"` → `"1111000000"` (1,111 LGM)
   - Line 83: `"11111000000000000000000"` → `"11111000000"` (11,111 LGM)
2. **Add custom modules to ModuleBasics** (P0-2)

### Before DevNet with Incentives (High — 3 items)

3. **Wire x/vesting_rewards** into app.go (P1-6)
   - Add keeper field, store key, NewKeeper, module manager entry
   - Add to Begin/EndBlocker ordering
   - Add to InitGenesis between tokens and emergency
4. **Set ResearchTreasuryAddress** to a real module account or multisig (P1-3)
5. **Wire vote_extensions_enable_height** in genesis template or config (P1-4)

### Before TestNet (5 items)

6. **Resolve domain design** — ontology vs knowledge ownership (P0-4, now P2)
7. **Document reward/penalty fund flows** (P1-2)
8. **Strengthen genesis validation** across all modules (P2-1)
9. **Map vesting categories** to knowledge categories (P2-2)
10. **Add total supply guard** to genesis account CLI (P1-5)

### Before MainNet (4 items)

11. **Implement `migrate-genesis`** for TS → Cosmos state migration (P1-1)
12. **Wire remaining modules** (ontology, disputes, evidence_mgmt) or defer
13. **Full genesis audit** by external security team
14. **Genesis generation script** with production-safe defaults

---

## Module Genesis Parameter Quick Reference (Updated)

### Wired Modules (in app.go)

| Module | Key Params | Decimal Status |
|--------|-----------|----------------|
| `x/knowledge` | MinDomainProposalStake=1 LGM, MinVerifiers=3, AcceptThreshold=77% | 6-dec ulgm ✅ |
| `x/auth` | MaxSessionKeys=5, KeyRotationCooldown=111 blocks | N/A |
| `x/staking` | VirtualStake=111 LGM ✅, TierBonded=1111 LGM ❌, TierGuardian=11111 LGM ❌ | **Mixed** |
| `x/tokens` | PeriodBlocks=1 day, PeriodsUntilExpiry=365, NoFreshOnTransfer | 6-dec ulgm ✅ |
| `x/emergency` | HaltQuorum=75%, RevertQuorum=80%, MinGuardianStake=111,111 LGM | 6-dec ulgm ✅ |
| `x/gov` | VotingPeriod=100K blocks, BaseCostPerVote=1 LGM, SupportThreshold=75% | 6-dec ulgm ✅ |
| `x/research` | MinResearchStake=100 LGM, ReviewPeriod=2 days, MaxBounty=10,000 LGM | 6-dec ulgm ✅ |

### Unwired Modules (defined but not in app.go)

| Module | Key Params | Decimal Status |
|--------|-----------|----------------|
| `x/vesting_rewards` | ResearchShare=22%, BlockReward=11 LGM, 10 categories | 6-dec ulgm ✅ |
| `x/ontology` | MinProposalStake=1 LGM, AllowNewStrata=false, 7 strata | 6-dec ulgm ✅ |
| `x/disputes` | SlashRate=50%, RewardRate=30%+20%, 3 tiers | 6-dec ulgm ✅ |
| `x/evidence_mgmt` | MinAuditStake=22 LGM, SlashRate=10%, RewardRate=5% | 6-dec ulgm ✅ |

---

## File Reference (Updated)

| File | What | Status |
|------|------|--------|
| `app/app.go:155-177` | ModuleBasics (still missing custom modules) | P0-2 |
| `app/app.go:179-193` | Module account permissions | OK |
| `app/app.go:563-589` | ModuleManager (7 customs wired) | OK |
| `app/app.go:592-608` | BeginBlocker ordering | OK (needs vesting_rewards) |
| `app/app.go:626-651` | InitGenesis ordering | OK (needs vesting_rewards) |
| `cmd/legbled/cmd/config.go:32-34` | Vote extensions (comment-only) | P1-4 |
| `cmd/legbled/cmd/genesis.go:131` | migrate-genesis (stub) | P1-1 |
| `x/staking/types/types.go:73,83` | TierBonded/Guardian MinStake (18-dec) | P1-7 |
| `x/vesting_rewards/types/genesis.go:9` | ResearchTreasuryAddress empty | P1-3 |
| `x/vesting_rewards/types/codec.go:48-54` | Handler: nil in ServiceDesc | P2-6 |
| `x/gov/types/types.go:149-168` | DefaultParams (all 6-dec ✅) | FIXED |
| `x/research/types/types.go:132-143` | DefaultParams (all 6-dec ✅) | FIXED |
| `x/emergency/types/types.go:218,224` | DefaultParams (all 6-dec ✅) | FIXED |

---

## Test Coverage (Go chain)

| Module | Test File | Tests | Status |
|--------|-----------|-------|--------|
| x/knowledge | keeper/keeper_test.go | 16 | Passing |
| x/auth | keeper/keeper_test.go | 38 | Passing |
| x/emergency | keeper/keeper_test.go | 30 | Passing |
| x/gov | keeper/keeper_test.go | 37 | Passing |
| x/research | keeper/keeper_test.go | Yes | Passing |
| x/staking | keeper/keeper_test.go | Yes | Passing |
| x/vesting_rewards | keeper/keeper_test.go | Yes | Passing |
| **Total** | 7 test files | **~160+** | **All passing** |

---

## Insights & Lessons

### Pattern: TS→Go Decimal Copying
The 18-decimal values consistently came from copying TypeScript bigint constants (where
1 LGM = 10^18) into Go without converting to ulgm (1 LGM = 10^6). The modules that were
implemented later (gov, research) got this right, suggesting the team learned the pattern.
The staking tier configs are a holdover from early implementation.

### Pattern: Implementation-before-Wiring
Several modules (vesting_rewards especially) have complete implementations but aren't wired.
This "build then integrate" approach creates hidden readiness — the code is there but inactive.
A `go build` passes but critical chain functionality is missing. Recommendation: add a genesis
smoke test that verifies all implemented modules are actually wired.

### Pattern: Self-Contradicting Comments
Comments like "1111 LGM in ulgm (18 decimals)" reveal the developer recognized the decimal
system but didn't fix the value. This suggests time pressure during implementation. Static
analysis could catch `big.Int` values > 10^12 in genesis params as likely 18-decimal errors.

### Cross-Module Dependency Safety
The current InitGenesis ordering is well-designed. The key insight is that `vesting_rewards`
should slot between `tokens` and `emergency` when wired — it depends on staking (for validator
info) and tokens (for period tracking), and emergency depends on staking tier information which
vesting rewards might influence through block reward distribution.

### Genesis Validation Gap
Only 2 of 7 wired modules have meaningful genesis validation (tokens, emergency). The rest
either return nil (research) or check 1-3 params. A genesis file with absurd parameters
(e.g., 0 block reward, 100% slash rate, negative quorum) would pass validation. This is
acceptable for DevNet but must be hardened for TestNet.
