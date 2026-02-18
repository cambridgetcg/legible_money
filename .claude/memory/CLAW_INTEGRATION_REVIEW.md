# Claw Framework Integration Review

> **Session**: 2026-02-16/17 | **Scope**: Unify OpenClaw variants across all modules
> **Status**: Phase 2 COMPLETE — staking/gov/disputes all covered, 86 total probes

---

## 1. Current State — 7 Files, 86 Probes

| File | Prefix | Tests | Module | Focus |
|------|--------|-------|--------|-------|
| `x/knowledge/keeper/openclaw_test.go` | OC-1→15 | 15 | knowledge | Financial incentives, game theory |
| `x/auth/keeper/openclaw_privacy_test.go` | OCP-1→12 | 12 | auth | DID, session keys, privacy boundaries |
| `x/bvm/vm/money_psychology_test.go` | PSY-1→12 | 12 | bvm | Wealth-belief coupling, agent psychology |
| `x/bvm/vm/empty_work_test.go` | FARM-1→11 | 11 | bvm | Empty work exploits, token farming |
| `x/staking/keeper/openclaw_staking_test.go` | OC-SK-1→12 | 12 | staking | Tier gaming, whale concentration, Sybil |
| `x/gov/keeper/openclaw_governance_test.go` | OCG/NCG/ZCG | 12 | gov | Vote buying, proposal spam, quadratic edge |
| `x/disputes/keeper/openclaw_disputes_test.go` | OCD/NCD/ZCD | 12 | disputes | Challenge spam, collusion, cascade |

**Total**: 86 probes, ~5,500 lines, 6 modules covered out of 27.

---

## 2. Taxonomy — Three Claw Variants

### OpenClaw (OC) — Macro Game Theory
**Definition**: Broad, open-ended probing of economic incentives, strategic behavior, and emergent dynamics at the *system* level. Tests model rational agents maximizing profit, colluding, or exploiting information asymmetry.

**Signature**: Multi-actor scenarios, economic analysis (cost/damage/profit), defense assessment.

**Already covered**:
- OC-1→15 (knowledge/financial): Whale domination, grief attacks, quorum manipulation, Sybil economics
- OCP-1→12 (auth/privacy): DID spoofing, session capability, account isolation
- PSY-1→12 (bvm/psychology): Wealth-awareness, conformity pressure, economic identity

**Uncovered modules** (high priority for OpenClaw):
- **staking**: Tier gaming, unbonding manipulation, validator cycling grief
- **gov**: Vote buying, proposal spam, quorum manipulation, quadratic voting edge cases
- **disputes**: Challenge spam, arbitration capture, evidence fabrication incentives
- **emergency**: False emergency triggering, guardian collusion, halt manipulation
- **vesting_rewards**: Vesting acceleration exploits, reward extraction timing
- **capture_defense**: Reputation manipulation, defense circumvention

### NanoClaw (NC) — Micro Precision Attacks
**Definition**: Small-scale, precision attacks targeting arithmetic edge cases, boundary conditions, integer truncation, off-by-one errors, and type conversion flaws. Each probe focuses on a *single computation* that could produce an unexpected result.

**Signature**: Specific numeric inputs at boundaries, exact expected vs actual comparison, BPS/BigInt arithmetic.

**Already exists (embedded in OpenClaw)**:
- OC-13: Slash rate sensitivity (BPS integer division)
- OC-14: Integer truncation attack (22% of 1 ulgm = 0)
- OC-4: Minimum stake spam (now guarded by MinClaimStakeUlgm)

**New NanoClaw scope** (not yet covered):
- **Threshold boundaries**: Acceptance at exactly 770,000 bps (77.0000%), rejection at 769,999 bps
- **Epoch transitions**: Rewards calculated at last block of epoch vs first block of next
- **BPS arithmetic**: All slash/reward calculations in confidence.go, rounds.go with edge-case inputs
- **BigInt overflow**: Stake × confidence products near uint64 max
- **Time boundaries**: Session expiry at exact block, unbonding completion at exact height
- **Category ceilings**: Confidence at exactly ceiling value, ceiling ± 1
- **Counter-fact ratios**: MinContradictionStakeRatio at exact boundary
- **Quadratic voting**: sqrt overflow at max vote weight in gov module
- **Vesting cliff**: Release at exact cliff block, one block before/after

### ZeroClaw (ZC) — Zero-Cost Attack Vectors
**Definition**: Attacks that cost the attacker nothing (or nearly nothing) — free information extraction, zero-stake operations, gas-less exploitation, and empty work that earns tokens.

**Signature**: Attacker profit > 0 with attacker cost ≈ 0.

**Already exists (embedded in OpenClaw)**:
- FARM-1→11 (bvm/empty work): Rubber-stamping, trivial spam, self-citation, abstain farming
- PSY-4: Economic self-surveillance (40 gas for full economic state)
- OC-3: Abstain to avoid slash
- H-2 (from OC session): Grief-by-rejection is profitable (attackers earn while burning victim)

**New ZeroClaw scope** (not yet covered):
- **Free state queries**: Read operations that leak validator strategy (staking module)
- **Gas-less governance**: Proposal viewing/reading without gas (gov module)
- **Default value exploitation**: Unset parameters that default to permissive values
- **Empty message attacks**: Valid but empty transaction types that pass validation
- **Zero-stake staking**: Attempt to register validator with 0 deposit (boundary)
- **Free evidence submission**: Evidence that passes format validation but contains no content
- **Channel spam**: Payment channels opened with 0 balance
- **Home squatting**: Agent homes created with no stake/assets

---

## 3. Integration Architecture

### Option A: Unified Runner (RECOMMENDED)
Create a shared `x/testutil/claw/` package with:
- `claw.go` — Common types: `ClawProbe`, `EconomicResult`, `InsightLog`
- `report.go` — Structured output: per-probe cost/damage/profit + defense assessment
- Each module's claw tests import `x/testutil/claw` for consistent reporting

**Pros**: Consistent insight format, cross-module correlation, single `go test ./x/testutil/claw/...` runner.
**Cons**: Adds a shared dependency; existing 50 tests need refactoring.

### Option B: Convention-Based (PRAGMATIC)
Keep tests in each module's `keeper/` or `vm/` directory. Enforce naming conventions:
- `openclaw_*.go` — OpenClaw (macro game theory)
- `nanoclaw_*.go` — NanoClaw (precision attacks)
- `zeroclaw_*.go` — ZeroClaw (zero-cost vectors)

Run all claws: `go test ./x/... -run "TestOC|TestNC|TestZC|TestOCP|TestPSY|TestFarm"`

**Pros**: Zero refactoring, each test stays near its module, standard Go test discovery.
**Cons**: No structured cross-module reporting.

### Recommendation: **Option B now, Option A later**
The 50 existing probes work well as-is. Naming convention is sufficient for Phase 1. When the total exceeds ~150 probes, extract the shared framework.

---

## 4. Coverage Gap Matrix

| Module | Tests | OpenClaw | NanoClaw | ZeroClaw | Priority |
|--------|-------|----------|----------|----------|----------|
| knowledge | 15 OC | **YES** | Partial (OC-13,14) | — | Add NC, ZC |
| auth | 12 OCP | **YES** | — | — | Add NC, ZC |
| bvm | 23 (PSY+FARM) | **YES** | — | **YES** (FARM) | Add NC |
| staking | 12 OC-SK | **YES** | — | — | ~~HIGH~~ **DONE** |
| gov | 12 (OCG+NCG+ZCG) | **YES** | **YES** (NCG-1,2) | **YES** (ZCG-1,2,3) | ~~HIGH~~ **DONE** |
| disputes | 12 (OCD+NCD+ZCD) | **YES** | **YES** (NCD-1,2) | **YES** (ZCD-1,2,3) | ~~HIGH~~ **DONE** |
| emergency | 0 claw | — | — | — | **MEDIUM** |
| vesting_rewards | 0 claw | — | — | — | **MEDIUM** |
| capture_defense | 0 claw | — | — | — | **MEDIUM** |
| capture_challenge | 0 claw | — | — | — | **MEDIUM** |
| tokens | 0 claw | — | — | — | LOW |
| evidence_mgmt | 0 claw | — | — | — | LOW |
| home | 0 claw | — | — | — | LOW |
| channels | 0 claw | — | — | — | LOW |

---

## 5. Cross-Module Probes (New Category)

Some attacks span multiple modules. These are the most dangerous and least tested:

| ID | Attack | Modules | Type |
|----|--------|---------|------|
| XM-1 | Stake→Claim→Unstake race | staking + knowledge | OpenClaw |
| XM-2 | Proposal-funded claim spam | gov + knowledge | ZeroClaw |
| XM-3 | Emergency halt during verification round | emergency + knowledge | OpenClaw |
| XM-4 | Vesting reward → immediate unstake | vesting_rewards + staking | NanoClaw |
| XM-5 | Dispute over fact during active round | disputes + knowledge | OpenClaw |
| XM-6 | Challenge spam funded by farming | capture_challenge + bvm | ZeroClaw |
| XM-7 | Tier promotion → immediate block production | staking + knowledge | OpenClaw |
| XM-8 | Counter-fact + dispute double-dip | knowledge + disputes | NanoClaw |

---

## 6. Retroactive Classification

### Existing probes mapped to Claw variant:

**OpenClaw (macro game theory)**:
- OC-1 Whale Domination, OC-2 Grief-by-Rejection, OC-5 Tier0 Viability
- OC-6 Counter-Fact Profit, OC-7 Split Verdict, OC-8 Difficulty Bonus
- OC-9 Research Tax, OC-10 Equivocation, OC-11 Quorum Manipulation
- OC-12 Sybil Economics, OC-15 Multi-Round Accumulation
- OCP-1→12 (all auth probes are OpenClaw: boundary probing at system level)
- PSY-1→12 (all psychology probes: emergent behavioral dynamics)
- OC-SK-1→12 (staking: whale concentration, slash escalation, tier gaming, Sybil spam)
- OCG-1→7 (gov: Sybil amplification, proposal spam, category downgrade, abstain quorum)
- OCD-1→6 (disputes: challenge spam, collusion, escalation delay, bond EV, cascade)

**NanoClaw (dedicated files in gov + disputes)**:
- OC-13 Slash Rate Sensitivity → **NC-1** (embedded in knowledge)
- OC-14 Integer Truncation Attack → **NC-2** (embedded in knowledge)
- NCG-1 sqrt precision at boundaries (gov)
- NCG-2 support threshold at exactly 75% (gov)
- NCD-1 bond=1 zero-slash truncation (disputes)
- NCD-2 quorum boundary at 2/3 and 5/7 (disputes)

**ZeroClaw (dedicated probes in gov + disputes)**:
- OC-3 Abstain to Avoid Slash → **ZC-1** (embedded in knowledge)
- OC-4 Minimum Stake Spam → **ZC-2** (now defended by MinClaimStakeUlgm)
- FARM-1→11 (all empty work probes are ZeroClaw by nature)
- ZCG-1 100 proposals at 1 ulgm each (gov)
- ZCG-2 vote on nonexistent proposal (gov)
- ZCG-3 self-dealing disbursement (gov)
- ZCD-1 inconclusive loop at zero cost (disputes)
- ZCD-2 non-arbiter vote rejection (disputes)
- ZCD-3 self-dispute exploit (disputes)

---

## 7. Implementation Roadmap

### Phase 1: Naming Convention (NOW)
- Rename nothing — but adopt the OC/NC/ZC/XM naming for all NEW probes
- Document taxonomy in this file (done)

### Phase 2: High-Priority Modules (COMPLETE ✓)
All 3 high-priority modules now have claw tests (36 total probes, all passing):
1. **`x/staking/keeper/openclaw_staking_test.go`** — 12 tests (OC-SK-1→12): whale concentration, slash escalation, epoch cap, Sybil spam, tier demotion, unbonding ID collision, block producer cap
2. **`x/gov/keeper/openclaw_governance_test.go`** — 12 tests (7 OCG + 2 NCG + 3 ZCG): Sybil amplification (10x via quadratic), proposal spam, category downgrade, abstain quorum, sqrt precision, threshold boundaries, zero-cost spam, self-dealing
3. **`x/disputes/keeper/openclaw_disputes_test.go`** — 12 tests (6 OCD + 2 NCD + 3 ZCD): challenge spam cap, arbiter collusion, escalation delay, bond EV asymmetry, cascade attack, zero-slash truncation, quorum boundaries, inconclusive loop, self-dispute

**Key findings (Phase 2)**:
- **2 CRITICAL**: OCG-1 (10x Sybil amplification in quadratic voting — defense relies on DID identity cost), ZCD-3 (self-dispute allowed — can pause own vesting)
- **3 HIGH**: OCG-4 (category downgrade attack), OCD-2 (3-arbiter collusion overturns verified fact), ZCD-1 (inconclusive loop at zero cost)
- **4 MEDIUM**: OCG-2 (50/50 proposal spam at 1 ulgm), OCG-5 (abstain inflates voter count), OCD-3 (19-day lockup via escalation), OCD-5 (abstention forces inconclusive)

### Phase 3: NanoClaw Extraction
Create dedicated nanoclaw files for precision attacks:
1. **`x/knowledge/keeper/nanoclaw_test.go`** — Extract OC-13/14 + add threshold boundaries
2. **`x/staking/keeper/nanoclaw_test.go`** — Tier transition arithmetic, unbonding epoch math
3. **`x/gov/keeper/nanoclaw_test.go`** — Quadratic sqrt overflow, quorum % boundaries

### Phase 4: ZeroClaw Extraction
Create dedicated zeroclaw files:
1. **`x/knowledge/keeper/zeroclaw_test.go`** — Extract OC-3/4 + add empty evidence, free reads
2. **`x/staking/keeper/zeroclaw_test.go`** — Zero deposit attempts, free state queries
3. **`x/bvm/vm/zeroclaw_test.go`** — Rename empty_work_test.go (it's already ZeroClaw)

### Phase 5: Cross-Module (FUTURE)
Implement XM-1→8 in `tests/cross_module/` using full app integration tests.

---

## 8. Summary

| Metric | Pre-Phase 2 | After Phase 2 | After Phase 4 |
|--------|-------------|---------------|---------------|
| Total probes | 50 | **86** | ~120 |
| Modules covered | 3/27 | **6/27** | 8/27 |
| Claw variants | 1 (OC) | **3 (OC/NC/ZC)** | 3 + XM |
| Critical findings | 3 (all fixed) | **+2 new** (5 total) | TBD |

**Phase 2 Results**: 36 new probes across staking/gov/disputes, all passing. Gov and disputes files use the mixed OC/NC/ZC naming convention (first files to do so). 2 new CRITICAL findings identified:
- **OCG-1**: Quadratic voting creates 10x Sybil amplification (same capital, split across identities). Defense: DID creation cost must exceed amplification benefit.
- **ZCD-3**: Self-dispute is unrestricted — agent can challenge own fact to pause vesting or probe arbiter pool.

**Key Insight**: The claw framework scales well. The OC/NC/ZC naming convention proved natural for gov and disputes — each file covers all three variants. Next priority: NanoClaw extraction for precision attacks (Phase 3) and medium-priority modules (emergency, vesting_rewards, capture_defense).
