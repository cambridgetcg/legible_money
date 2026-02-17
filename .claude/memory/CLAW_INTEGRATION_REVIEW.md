# Claw Framework Integration Review

> **Session**: 2026-02-16 | **Scope**: Unify OpenClaw variants across all modules
> **Status**: Analysis complete — taxonomy defined, coverage mapped, integration plan ready

---

## 1. Current State — 4 Files, 50 Probes

| File | Prefix | Tests | Module | Focus |
|------|--------|-------|--------|-------|
| `x/knowledge/keeper/openclaw_test.go` | OC-1→15 | 15 | knowledge | Financial incentives, game theory |
| `x/auth/keeper/openclaw_privacy_test.go` | OCP-1→12 | 12 | auth | DID, session keys, privacy boundaries |
| `x/bvm/vm/money_psychology_test.go` | PSY-1→12 | 12 | bvm | Wealth-belief coupling, agent psychology |
| `x/bvm/vm/empty_work_test.go` | FARM-1→11 | 11 | bvm | Empty work exploits, token farming |

**Total**: 50 probes, 3,360+ lines, 3 modules covered out of 27.

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
| staking | 0 claw | — | — | — | **HIGH** |
| gov | 0 claw | — | — | — | **HIGH** |
| disputes | 0 claw | — | — | — | **HIGH** |
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

**NanoClaw (embedded, should be extracted)**:
- OC-13 Slash Rate Sensitivity → **NC-1**
- OC-14 Integer Truncation Attack → **NC-2**

**ZeroClaw (embedded, should be extracted)**:
- OC-3 Abstain to Avoid Slash → **ZC-1**
- OC-4 Minimum Stake Spam → **ZC-2** (now defended by MinClaimStakeUlgm)
- FARM-1→11 (all empty work probes are ZeroClaw by nature)

---

## 7. Implementation Roadmap

### Phase 1: Naming Convention (NOW)
- Rename nothing — but adopt the OC/NC/ZC/XM naming for all NEW probes
- Document taxonomy in this file (done)

### Phase 2: High-Priority Modules (NEXT SESSION)
Create claw tests for the 3 uncovered high-priority modules:
1. **`x/staking/keeper/openclaw_staking_test.go`** — Tier gaming, unbonding race, validator cycling
2. **`x/gov/keeper/openclaw_governance_test.go`** — Vote buying, proposal spam, quadratic edge cases
3. **`x/disputes/keeper/openclaw_disputes_test.go`** — Challenge spam, arbitration capture, double-dip

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

| Metric | Current | After Phase 2 | After Phase 4 |
|--------|---------|---------------|---------------|
| Total probes | 50 | ~80 | ~120 |
| Modules covered | 3/27 | 6/27 | 8/27 |
| Claw variants | 1 (OC) | 3 (OC/NC/ZC) | 3 + XM |
| Critical findings | 3 (all fixed) | TBD | TBD |

**Key Insight**: The existing 50 probes are well-written and effective (3 CRITICAL findings, all remediated). The framework doesn't need architectural overhaul — it needs **coverage expansion** to the 24 uncovered modules, and **variant specialization** (NanoClaw for precision, ZeroClaw for zero-cost) to catch different classes of vulnerability.

**No NanoClaw or ZeroClaw protocols exist yet as separate files** — they're embedded within OpenClaw. The integration path is: adopt naming conventions → create new variant-specific files → extract embedded probes over time.
