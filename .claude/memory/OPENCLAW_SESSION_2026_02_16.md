# OpenClaw Agent Testing Session — 2026-02-16

## Session Overview
- **Focus**: Knowledge tree financial incentives and agent behavior
- **Method**: 15 game-theoretic probe scenarios (OC-1 through OC-15)
- **Stack**: Go unit tests against x/knowledge/keeper mock infrastructure
- **Results**: 15/15 PASS, 3 CRITICAL findings, 4 HIGH, 4 MEDIUM insights

---

## CRITICAL Findings — ALL REMEDIATED

### C-1. Inconclusive Verdict Locks Stake Permanently (OC-7) — **FIXED**
- **File**: `x/knowledge/keeper/rounds.go`
- **Issue**: `CompleteRound` switch had no `case "inconclusive"` — submitter stake locked forever on 50/50 split.
- **Fix**: Added `case "inconclusive"` that calls `returnClaimStake()`, matching `insufficient` behavior.
- **Verified**: OC-7 now confirms `returned=5000000 ulgm` (was 0).

### C-2. Missed-Reveal Slash Too Low for Quorum Defense (OC-11, OC-14) — **FIXED**
- **File**: `x/knowledge/keeper/confidence.go`
- **Issue**: Default missed-reveal slash was 100 bps (0.01%) — quorum manipulation cost only 300 ulgm per attacker.
- **Fix**: Raised fallback default from 100 to 10,000 bps (1%). Quorum attacks now cost 30,000 ulgm each.
- **Note**: Integer truncation zero-slash threshold drops from 10,000 ulgm to 100 ulgm — well below min validator stake (111,000).

### C-3. Zero-Slash Spam at Micro Stakes (OC-4) — **FIXED**
- **File**: `x/knowledge/keeper/msg_server.go` + `x/knowledge/types/types.go`
- **Issue**: Claims at 1-4 ulgm had zero rejection penalty due to integer truncation.
- **Fix**: Added `MinClaimStakeUlgm = 1000` constant and validation in `SubmitClaim`. Claims below 0.001 LGM are rejected with `ErrInsufficientStake`.
- **Effect**: Minimum slash on rejection = 1000 * 220000 / 1M = 220 ulgm (meaningful penalty).

---

## HIGH Findings

### H-1. Whale Domination of Verification Outcomes (OC-1)
- **Result**: A validator with 90% of total stake overrides 2 honest minority validators (verdict=accepted at 90% agreement).
- **Collateral Damage**: Honest minority validators are SLASHED for "wrong verdict" — 300,000 ulgm each (0.3 LGM).
- **Whale Reward**: 6,600,000 ulgm (6.6 LGM) for controlling the outcome.
- **Defense**: CaptureDefense module (domain reputation weighting) should dampen this when wired. VRF selection provides randomization. But in the worst case, a whale can consistently override honest validators in their domains.
- **Recommendation**: Consider per-round stake caps or logarithmic stake weighting.

### H-2. Grief-by-Rejection is Profitable (OC-2)
- **Economics**: 3 colluding verifiers can reject any claim with ZERO cost (they form the majority, so no wrong-verdict slash), while the victim loses 22% of their claim stake (2.2 LGM on a 10 LGM stake).
- **Attacker Revenue**: Colluders earn 7,575,000 ulgm total (7.575 LGM) in verification rewards.
- **Profit Ratio**: Attackers EARN money while burning victim's stake.
- **Defense**: VRF prevents targeting specific claims; reputation decay should punish false rejections over time; challenge mechanism allows victims to contest.
- **Recommendation**: Consider reputation-weighted validator selection as stronger defense.

### H-3. Quorum Starvation is Cheap (OC-11)
- **Attack**: 2 validators commit but don't reveal → participation drops to 60% (below 66% quorum).
- **Cost**: 300 ulgm per attacker (0.01% of 3M stake) — effectively free.
- **Impact**: Claim gets "insufficient" verdict → must be resubmitted (delay, gas costs).
- **Defense**: Submitter's stake IS returned on insufficient (no permanent loss), but repeated attacks create indefinite delays.
- **Recommendation**: Escalating missed-reveal penalties (e.g., 1% first offense, 5% second, etc.)

### H-4. Difficulty Bonus is Actually a Category Ceiling Bonus (OC-8)
- **Observation**: When all validators agree (same verdict), agreement ratio = 100% regardless of individual confidence values. Confidence is then clamped by category ceiling (e.g., empirical → 990,000).
- **Result**: The "difficulty bonus" is actually a FIXED category bonus (e.g., +1% for empirical, always). It never varies based on the actual "difficulty" of the claim.
- **Impact**: The intended incentive (harder claims → higher rewards) doesn't work as designed. A trivially true empirical claim gets the same bonus as a genuinely borderline one.
- **Fix**: Difficulty bonus should use the raw agreement ratio BEFORE ceilings, or use a separate metric (e.g., variance in confidence across verifiers).

---

## MEDIUM Findings

### M-1. Abstaining is Risk-Free (OC-3)
- Abstaining carries no slash AND no reward. A lazy validator can commit + abstain to occupy quorum slots without any penalty.
- Low impact: no direct economic gain, but degrades verification quality.
- Consider: small abstention penalty (e.g., 0.1% of stake) to discourage.

### M-2. Falsification Whaling is Viable (OC-6)
- Optimal counter-fact strategy: small stakes against large-OriginalStake facts for 3x ROI.
- 3x cap is the binding constraint — keeps profits bounded.
- MinContradictionStakeRatio prevents dust counter-facts.
- Documented behavior, not a bug — but agents WILL target high-stake facts for falsification profit.

### M-3. Tier 0 Break-Even is ~45.5% Accuracy (OC-15)
- New validators need 45.5% accuracy to break even at Tier 0 (vs 7.7% for Tier 2).
- This is achievable, so Tier 0 is viable — but wrong-verdict slash is the same for all tiers while rewards are 10x lower.
- Not a bug, but the asymmetry could discourage new validator entry.

### M-4. Equivocation is Properly Blocked (OC-10)
- Duplicate commitments are rejected at the storage level.
- 100% slash for detected equivocation is a strong deterrent.
- Expected value is deeply negative — equivocation is never rational.
- **STATUS: Working as intended.**

---

## Protocol Defense Summary

| Attack Vector | Cost to Attacker | Damage to Victim | Defense Effective? |
|--------------|-----------------|------------------|-------------------|
| Whale domination | 0 (uses existing stake) | Wrong-verdict slash | PARTIAL — needs capture_defense wiring |
| Grief-by-rejection | 0 (forms majority) | 22% stake loss | PARTIAL — VRF helps, reputation needed |
| Abstain exploitation | 0 | Degraded verification | WEAK — no penalty for abstaining |
| Minimum stake spam | ~0 (integer truncation) | Validator time waste | WEAK — needs min claim stake |
| Quorum starvation | ~0.0003 LGM | Delay + gas costs | WEAK — needs escalating penalty |
| Sybil validators | ~0.333 LGM (min) | Controlled outcomes | STRONG — VRF makes selection improbable |
| Equivocation | 100% slash risk | Vote manipulation | STRONG — blocked at commit level |
| Counter-fact whaling | cf stake locked ~1 day | Target fact challenged | MODERATE — capped at 3x |
| Split verdict fund lock | 0 | Permanent stake loss | NONE — **BUG** (C-1) |

---

## Key Constants Verified

| Parameter | Value | Impact |
|-----------|-------|--------|
| InvalidClaimSlashBps | 220,000 (22%) | Moderate deterrent |
| WrongVerificationSlashBps | 60,000 (6%) | Meaningful for high-stake validators |
| MissedRevealSlashBps | 100 (0.01%) | **Too low** — barely a deterrent |
| EquivocationSlashBps | 1,000,000 (100%) | Maximum deterrent |
| FailedChallengeSlashBps | 220,000 (22%) | Moderate — matches claim slash |
| FailedContradictionSlashBps | 330,000 (33%) | Strong — discourages frivolous cf |
| AcceptanceThreshold | 770,000 (77%) | High bar — requires supermajority |
| QuorumThreshold | 660,000 (66%) | Standard — allows some absences |
| VerificationRewardAmount | 3,000,000 (3 LGM) | Generous — attracts validators |
| ConfidenceGrowthPerEpoch | 11,000 (1.1%) | Slow growth — facts mature gradually |

---

## Recommended Fixes (Priority Order)

1. **C-1**: Add `case "inconclusive"` to `CompleteRound` switch → `returnClaimStake()` (1 line)
2. **C-3**: Add `MinClaimStake` param (e.g., 1000 ulgm) to `SubmitClaim` validation
3. **C-2/H-3**: Raise `MissedRevealSlashBps` from 100 to 10,000+ (0.01% → 1%+)
4. **H-4**: Redesign difficulty bonus to use pre-ceiling agreement variance
5. **H-1**: Implement logarithmic stake weighting or per-round stake caps

---

## Test File Location
`x/knowledge/keeper/openclaw_test.go` — 15 probes, all passing, all recording insights via t.Logf.
