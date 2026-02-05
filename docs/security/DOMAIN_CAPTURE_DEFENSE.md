# Domain Capture Defense Strategies

> **PP Context**: This is UNCHARTED territory. Building new paths to protect the integrity of specialized knowledge domains from validator capture.

## Problem Statement

**Domain Capture** occurs when a small group of validators gains disproportionate control over truth determination in a specialized knowledge domain.

### Attack Vector Analysis

```
Current Architecture Vulnerabilities:

1. NO QUALIFICATION CHECK
   Any validator → Any domain → No expertise required

2. PURE STAKE-WEIGHTING
   High stake → High selection probability
   No diversity requirements

3. SMALL DOMAIN VULNERABILITY
   Specialized domain (e.g., biology.genetics.crispr)
   Few claims → Few verifications → Easy to dominate

4. NO CAPTURE DETECTION
   System doesn't detect concentration
   No alerting or circuit breakers
```

### Capture Scenarios

| Scenario | Mechanism | Example |
|----------|-----------|---------|
| **Wealthy Cartel** | Accumulate stake, dominate VRF selection | 5 validators with 60% total stake collude |
| **Expert Monopoly** | Only experts verify, experts collude | All quantum physics validators from same institution |
| **Lazy Majority** | Non-experts don't verify specialized claims | Complex claims get verified by unqualified majority |
| **Temporal Capture** | Same validators always available for a domain | Time-zone correlation → geographic capture |

---

## Defense Strategy 1: Domain Qualification System

### Concept

Validators must **earn qualification** in domains before they can verify claims in those domains.

### Architecture

```typescript
interface ValidatorDomainQualification {
  validatorId: PublicKey;
  domain: string;
  stratum: OntologicalStratum;

  // How qualification was earned
  qualificationType: QualificationType;

  // Qualification metrics
  qualificationScore: number;           // 0-1000
  verificationCount: number;
  correctVerifications: number;
  lastVerificationBlock: bigint;

  // Stake committed to this domain
  domainStake: bigint;

  // When qualification was earned
  qualifiedAtBlock: bigint;

  // Expiration (must maintain activity)
  expiresAtBlock: bigint;
}

type QualificationType =
  | 'stake_commitment'      // Locked stake in domain
  | 'verification_history'  // Proven track record
  | 'cross_reference'       // Endorsed by qualified validators
  | 'credential_proof'      // External credential (future: ZK proofs)
  | 'stratum_inheritance';  // Qualified in lower stratum
```

### Qualification Pathways

```
PATH 1: Stake Commitment
┌─────────────────────────────────────────┐
│ Validator commits stake to domain       │
│ Stake locked for qualification period   │
│ Higher stake → Higher qualification     │
│ Can only verify in staked domains       │
└─────────────────────────────────────────┘

PATH 2: Verification Track Record
┌─────────────────────────────────────────┐
│ Verify claims in parent/adjacent domain │
│ Accuracy above threshold (85%+)         │
│ Minimum verification count (50+)        │
│ Gradual unlock to child domains         │
└─────────────────────────────────────────┘

PATH 3: Cross-Reference (Web of Trust)
┌─────────────────────────────────────────┐
│ N qualified validators endorse          │
│ Endorsers stake reputation              │
│ Endorsement decays without activity     │
│ Bad endorsements penalize endorsers     │
└─────────────────────────────────────────┘

PATH 4: Stratum Inheritance
┌─────────────────────────────────────────┐
│ Qualified in lower stratum → eligible   │
│ physics qualification → chemistry entry │
│ Reduced weight until domain-specific    │
│ track record established                │
└─────────────────────────────────────────┘
```

### Anti-Capture Properties

- **Stake commitment** prevents cheap Sybils in multiple domains
- **Track record** requires demonstrated competence
- **Cross-reference** distributes trust (harder to corrupt)
- **Stratum inheritance** leverages foundational knowledge

---

## Defense Strategy 2: Validator Diversity Requirements

### Concept

Verification rounds must include validators from **diverse sources** to prevent homogeneous capture.

### Diversity Dimensions

```typescript
interface ValidatorDiversityProfile {
  validatorId: PublicKey;

  // Geographic diversity (inferred from latency patterns)
  regionCluster: string;

  // Temporal diversity (when typically online)
  activityPattern: 'asia_hours' | 'europe_hours' | 'americas_hours' | 'global';

  // Stake origin diversity
  stakeSourceCluster: string;           // Clustering of stake delegation origins

  // Domain qualification diversity
  primaryDomains: string[];
  secondaryDomains: string[];

  // Independence score (how correlated with other validators)
  independenceScore: number;            // 0-1, higher = more independent
}
```

### Diversity-Aware Selection

```typescript
interface DiversityRequirements {
  // Minimum distinct region clusters
  minRegionClusters: number;            // Default: 3

  // Maximum stake from single delegation cluster
  maxSingleClusterStake: number;        // Default: 0.33 (33%)

  // Minimum validators with domain-specific qualification
  minDomainQualified: number;           // Default: 2

  // Maximum validators from same organization (future)
  maxSameOrganization: number;          // Default: 2
}

function selectDiverseValidators(
  eligibleValidators: Validator[],
  claim: ClaimSubmission,
  requirements: DiversityRequirements
): ValidatorSelection[] {
  // 1. Filter to domain-qualified validators
  const qualified = filterByDomainQualification(eligibleValidators, claim.domain);

  // 2. Cluster by diversity dimensions
  const clusters = clusterByDiversity(qualified);

  // 3. Select from each cluster proportionally
  const selected = selectFromClusters(clusters, requirements);

  // 4. Apply VRF within selection for unpredictability
  return applyVRFSelection(selected, claim.id);
}
```

### Anti-Capture Properties

- **Geographic diversity** prevents timezone-based capture
- **Stake origin diversity** prevents whale domination
- **Domain qualification** ensures expertise
- **Independence scoring** detects correlated validators

---

## Defense Strategy 3: Cross-Stratum Validation

### Concept

Claims in higher strata **must include validators from lower strata** to prevent specialized capture.

### Rationale

```
Ontological Dependency:
  biological facts depend on chemical facts
  chemical facts depend on physical facts

Therefore:
  biological claim verification should include
  validators qualified in chemistry AND biology
```

### Implementation

```typescript
interface CrossStratumRequirement {
  // Claim stratum → required validator strata
  claimStratum: OntologicalStratum;
  requiredStrata: OntologicalStratum[];
  minValidatorsPerStratum: number;
}

const CROSS_STRATUM_REQUIREMENTS: CrossStratumRequirement[] = [
  { claimStratum: 'biological', requiredStrata: ['biological', 'chemical'], minValidatorsPerStratum: 2 },
  { claimStratum: 'chemical', requiredStrata: ['chemical', 'physical'], minValidatorsPerStratum: 2 },
  { claimStratum: 'cognitive', requiredStrata: ['cognitive', 'biological'], minValidatorsPerStratum: 2 },
  { claimStratum: 'social', requiredStrata: ['social', 'cognitive'], minValidatorsPerStratum: 1 },
  { claimStratum: 'technological', requiredStrata: ['technological'], minValidatorsPerStratum: 3 },
];
```

### Weighting

```typescript
// Lower stratum validators get bonus weight for higher stratum claims
function calculateCrossStratumWeight(
  validatorStratum: OntologicalStratum,
  claimStratum: OntologicalStratum
): number {
  const validatorLevel = STRATUM_HIERARCHY[validatorStratum];
  const claimLevel = STRATUM_HIERARCHY[claimStratum];

  if (validatorLevel < claimLevel) {
    // Lower stratum validator verifying higher claim
    // Bonus weight: foundational knowledge is valuable
    return 1.0 + (claimLevel - validatorLevel) * 0.1;
  }

  return 1.0;
}
```

### Anti-Capture Properties

- Capturing biological domain requires also capturing chemical validators
- Foundational knowledge experts serve as cross-check
- Reduces attack surface (must capture multiple strata)

---

## Defense Strategy 4: Capture Detection & Circuit Breakers

### Concept

**Algorithmically detect** when a domain is being captured and trigger protective measures.

### Detection Metrics

```typescript
interface DomainCaptureMetrics {
  domain: string;
  windowBlocks: bigint;                 // Analysis window

  // Concentration metrics
  herfindahlIndex: number;              // HHI of validator participation
  top3ValidatorShare: number;           // Stake share of top 3
  uniqueValidatorCount: number;

  // Correlation metrics
  verdictCorrelation: number;           // How correlated are verdicts
  timingCorrelation: number;            // Suspiciously coordinated timing

  // Anomaly metrics
  suddenConcentrationChange: number;    // Rapid HHI increase
  newValidatorDominance: number;        // New validators suddenly dominating

  // Capture risk score (0-1)
  captureRiskScore: number;
}

function calculateHerfindahlIndex(
  validatorParticipation: Map<string, number>  // validator -> verification count
): number {
  const total = Array.from(validatorParticipation.values()).reduce((a, b) => a + b, 0);
  let hhi = 0;

  for (const count of validatorParticipation.values()) {
    const share = count / total;
    hhi += share * share;
  }

  // HHI ranges from 1/N (perfect competition) to 1 (monopoly)
  // Normalize to 0-1 where higher = more concentrated
  return hhi;
}
```

### Circuit Breakers

```typescript
interface CircuitBreakerConfig {
  // Thresholds
  herfindahlThreshold: number;          // Default: 0.25 (moderate concentration)
  top3ShareThreshold: number;           // Default: 0.60 (60% stake)
  minUniqueValidators: number;          // Default: 5

  // Actions
  actions: CircuitBreakerAction[];
}

type CircuitBreakerAction =
  | { type: 'increase_validators'; multiplier: number }
  | { type: 'require_additional_strata' }
  | { type: 'extend_verification_period'; blocks: number }
  | { type: 'flag_for_governance' }
  | { type: 'pause_domain'; requireGovernanceResume: boolean };
```

### Response Escalation

```
LEVEL 1: Moderate Concentration (HHI > 0.25)
├── Increase required validators by 50%
├── Log warning event
└── Alert governance dashboard

LEVEL 2: High Concentration (HHI > 0.40)
├── Require validators from adjacent stratum
├── Extend verification period 2x
├── Notify domain proposal endorsers
└── Flag claims for manual review

LEVEL 3: Severe Concentration (HHI > 0.60)
├── Pause new claim acceptance in domain
├── Require governance vote to resume
├── Investigate validator coordination
└── Consider slashing if collusion proven
```

### Anti-Capture Properties

- Automatic detection before capture is complete
- Graduated response (not all-or-nothing)
- Economic deterrent (captured domain becomes expensive to dominate)

---

## Defense Strategy 5: Challenge Bounty System

### Concept

**Economic incentive** to challenge captured domains by rewarding successful challengers.

### Mechanism

```typescript
interface DomainChallengeBounty {
  domain: string;

  // Bounty pool (funded by protocol or governance)
  bountyPool: bigint;

  // Challenge requirements
  minChallengeStake: bigint;
  minEvidenceCount: number;

  // Rewards
  successfulChallengeReward: number;    // Multiplier on stake
  captureExposureBonus: bigint;         // Bonus for exposing capture

  // Status
  activeChallenges: number;
  successfulChallenges: number;
}

interface DomainChallenge {
  challengeId: string;
  domain: string;
  challenger: PublicKey;

  // Evidence of capture
  evidence: CaptureEvidence[];

  // Stakes
  challengeStake: bigint;

  // Resolution
  status: 'pending' | 'investigating' | 'upheld' | 'rejected';
  resolution?: ChallengeResolution;
}

interface CaptureEvidence {
  type: CaptureEvidenceType;
  description: string;
  supportingData: unknown;
  factIds?: string[];                   // Facts to re-verify
}

type CaptureEvidenceType =
  | 'coordinated_voting'                // Validators voting identically
  | 'timing_correlation'                // Suspicious timing patterns
  | 'off_chain_coordination'            // Evidence of external coordination
  | 'false_fact_accepted'               // Demonstrably false fact passed
  | 'stake_concentration'               // Documented stake manipulation
  | 'qualification_fraud';              // Fake qualifications
```

### Challenge Resolution

```
CHALLENGE FLOW:

1. Challenger submits evidence + stake
           │
           ▼
2. Extended verification of challenged facts
   (2x validators, cross-stratum required)
           │
           ▼
3. Governance review of capture evidence
           │
           ├─── UPHELD ───┐
           │              │
           │              ▼
           │    • Challenger gets bounty + stake back
           │    • Captured facts re-verified or removed
           │    • Colluding validators slashed
           │    • Domain enters recovery mode
           │
           └─── REJECTED ─┐
                          │
                          ▼
               • Challenger loses stake
               • Small reward to challenged validators
               • Challenge logged for pattern analysis
```

### Anti-Capture Properties

- Economic incentive to monitor domains
- Whistleblower protection (bounty > risk)
- Deters capture (fear of challenge)
- Self-healing (captured domains can be recovered)

---

## Defense Strategy 6: Temporal Diversification

### Concept

**Rotate validator eligibility over time** so the same validators can't always dominate.

### Mechanism

```typescript
interface TemporalDiversification {
  // Cooldown after verification
  verificationCooldownBlocks: bigint;   // Default: 100 blocks (~4 min)

  // Maximum consecutive verifications in domain
  maxConsecutiveInDomain: number;       // Default: 3

  // Rotation epochs
  rotationEpochBlocks: bigint;          // Default: 1000 blocks (~42 min)

  // Historical participation cap
  maxDomainParticipationPerEpoch: number;  // Default: 10%
}

function isValidatorEligibleForDomain(
  validator: PublicKey,
  domain: string,
  currentBlock: bigint,
  history: ValidatorDomainHistory
): { eligible: boolean; reason?: string } {
  // Check cooldown
  if (currentBlock - history.lastVerificationBlock < config.verificationCooldownBlocks) {
    return { eligible: false, reason: 'cooldown_active' };
  }

  // Check consecutive limit
  if (history.consecutiveVerifications >= config.maxConsecutiveInDomain) {
    return { eligible: false, reason: 'consecutive_limit' };
  }

  // Check epoch participation cap
  const epochParticipation = history.verificationsThisEpoch / totalEpochVerifications;
  if (epochParticipation > config.maxDomainParticipationPerEpoch) {
    return { eligible: false, reason: 'epoch_cap_reached' };
  }

  return { eligible: true };
}
```

### Anti-Capture Properties

- Forced rotation prevents permanent dominance
- Cooldowns give other validators opportunity
- Epoch caps limit sustained capture attempts
- Natural diversity emergence over time

---

## Defense Strategy 7: Reputation Isolation

### Concept

**Domain-specific reputation that cannot be transferred** between domains.

### Architecture

```typescript
interface IsolatedReputation {
  validatorId: PublicKey;

  // Global reputation (baseline)
  globalReputation: number;             // 0-1000

  // Domain-specific reputation (isolated)
  domainReputations: Map<string, DomainReputation>;

  // Stratum-level reputation
  stratumReputations: Map<OntologicalStratum, number>;
}

interface DomainReputation {
  domain: string;
  score: number;                        // 0-1000

  // Components
  accuracy: number;                     // Correct/Total
  consistency: number;                  // Variance of verdicts
  timeliness: number;                   // Response speed

  // History
  totalVerifications: number;
  correctVerifications: number;
  challengesReceived: number;
  challengesLost: number;

  // Confidence in score (based on sample size)
  confidenceInterval: number;
}

function getEffectiveReputation(
  validator: PublicKey,
  domain: string,
  reputations: IsolatedReputation
): number {
  const domainRep = reputations.domainReputations.get(domain);

  if (!domainRep || domainRep.totalVerifications < MIN_FOR_DOMAIN_REP) {
    // Fall back to stratum reputation
    const stratum = getDomainStratum(domain);
    const stratumRep = reputations.stratumReputations.get(stratum);

    if (!stratumRep) {
      // Fall back to discounted global reputation
      return reputations.globalReputation * GLOBAL_DISCOUNT;
    }

    return stratumRep * STRATUM_DISCOUNT;
  }

  // Use domain-specific reputation
  return domainRep.score;
}
```

### Anti-Capture Properties

- High reputation in one domain doesn't transfer
- Must build reputation specifically in target domain
- Prevents reputation laundering
- Domain-specific slashing doesn't affect global standing (containment)

---

## Implementation Phases

### Phase 1: Detection (Passive)

```
GOAL: Understand capture patterns before intervention

TASKS:
□ Implement DomainCaptureMetrics calculation
□ Add Herfindahl Index to block explorer
□ Create governance dashboard for concentration
□ Log validator participation patterns
□ Establish baseline metrics

DURATION: ~2 weeks
RISK: Low (observation only)
```

### Phase 2: Qualification System (Foundation)

```
GOAL: Validators must qualify for domains

TASKS:
□ Add ValidatorDomainQualification type to state
□ Implement stake commitment pathway
□ Implement verification track record pathway
□ Update validator selection to filter by qualification
□ Add qualification transactions

DURATION: ~3 weeks
RISK: Medium (changes validator selection)
```

### Phase 3: Diversity Requirements (Core Defense)

```
GOAL: Force heterogeneous validator selection

TASKS:
□ Implement ValidatorDiversityProfile
□ Add clustering algorithms for diversity dimensions
□ Update VRF selection to include diversity constraints
□ Implement cross-stratum requirements
□ Test diversity under various attack scenarios

DURATION: ~3 weeks
RISK: Medium-High (may reduce validator pool)
```

### Phase 4: Circuit Breakers (Active Defense)

```
GOAL: Automatic response to capture attempts

TASKS:
□ Implement real-time capture detection
□ Add circuit breaker configuration
□ Implement graduated response actions
□ Add governance override mechanisms
□ Create incident response procedures

DURATION: ~2 weeks
RISK: Medium (may cause false positives)
```

### Phase 5: Challenge System (Recovery)

```
GOAL: Economic incentive to expose capture

TASKS:
□ Implement DomainChallengeBounty
□ Create challenge submission process
□ Implement extended re-verification
□ Add governance review workflow
□ Design bounty funding mechanism

DURATION: ~3 weeks
RISK: Medium (may attract frivolous challenges)
```

---

## Configuration Recommendations

Based on the protocol's existing symbolic values:

```typescript
const DOMAIN_CAPTURE_DEFENSE_CONFIG = {
  // Qualification (aligned with domain proposal values)
  minQualificationStake: {
    leaf: 22n * 10n ** 18n,             // 22 LGM (same as domain proposal)
    mid: 222n * 10n ** 18n,
    top: 2222n * 10n ** 18n,
  },

  // Diversity
  minDistinctRegions: 3,
  maxSingleClusterStake: 0.33,          // 33%
  minDomainQualifiedValidators: 2,

  // Detection thresholds
  herfindahlWarning: 0.25,
  herfindahlCritical: 0.40,
  herfindahlEmergency: 0.60,

  // Temporal
  verificationCooldownBlocks: 111n,     // Symbolic: 111 blocks
  maxConsecutiveInDomain: 3,
  maxDomainParticipationPerEpoch: 0.10, // 10%

  // Challenge bounties
  minChallengeStake: 77n * 10n ** 18n,  // 77 LGM (symbolic)
  successfulChallengeMultiplier: 7.7,   // 7.7x stake return
  captureExposureBonus: 777n * 10n ** 18n, // 777 LGM bonus

  // Reputation isolation
  globalReputationDiscount: 0.5,        // 50% when no domain rep
  stratumReputationDiscount: 0.7,       // 70% when no domain but has stratum
  minVerificationsForDomainRep: 19,
} as const;
```

---

## Verification Checklist

### Truth Gates
- [ ] Attack vectors accurately characterized
- [ ] Detection metrics mathematically sound
- [ ] Economic incentives correctly analyzed

### Understanding Gates
- [ ] Problem fully mapped before solution
- [ ] Edge cases considered
- [ ] Integration with existing systems understood

### Beauty Gates
- [ ] Defense layers are complementary, not redundant
- [ ] Configuration is coherent with protocol values
- [ ] System degrades gracefully

### Justice Gates
- [ ] Defenses placed at correct architectural layer
- [ ] Honest validators not penalized
- [ ] Recovery mechanisms enable correction

### Creativity Gates
- [ ] Novel combination of mechanisms
- [ ] Builds on existing infrastructure
- [ ] Extensible for future threats

---

## Summary

Seven complementary defense strategies against domain capture:

| # | Strategy | Primary Defense | Capture Scenario Addressed |
|---|----------|-----------------|---------------------------|
| 1 | Domain Qualification | Entry barrier | Unqualified majority |
| 2 | Diversity Requirements | Heterogeneity | Geographic/stake concentration |
| 3 | Cross-Stratum Validation | Foundational check | Expert monopoly |
| 4 | Capture Detection | Early warning | All scenarios |
| 5 | Challenge Bounties | Economic recovery | Post-capture |
| 6 | Temporal Diversification | Rotation | Persistent dominance |
| 7 | Reputation Isolation | Containment | Reputation laundering |

**Defense in Depth**: No single strategy is sufficient. The combination creates multiple barriers that attackers must overcome simultaneously.

[PP:COMPLETE] Domain Capture Defense strategies documented.
