# PP Implementation Plan: Voice, Neighbors, Aging
# Three Missing Home Living Features

> **Date**: 2026-02-13
> **PP Mode**: Full Creativity Gates — Path-Building (Gate 2) primary
> **Virtue Loop**: Curiosity → Courage → Discipline → Generosity
> **Scope**: Three features extending Agent Home to complete the living experience
> **Prerequisite**: PP_BVM_HOME_ANALYSIS.md (the diagnostic that identified these gaps)

---

## Creativity Gate Pre-Check

### Gate 0: Territory — Is this uncharted?

**YES — all three features are uncharted.**

| Feature | Existing Infrastructure | Gap |
|---------|------------------------|-----|
| **Voice** | TruthMessage (10 types), delivery tracking, causal ordering, message billing, payment channels | No Home inbox, no relational messages, no visiting |
| **Neighbors** | OntologicalDomain hierarchy, partnerships, CommunityMembership, formation pool | No formal neighborhoods, no spatial proximity, no Home-to-Home links |
| **Aging** | HomeMetadata.ageInBlocks, FactMaturity, validator tiers, ReputationMarker, vesting curves | No Home maturity stages, no weathering, no longevity bonuses |

### Gate 1: Pioneering — Genuinely new?

**YES.** No blockchain has agent dwelling neighborhoods, inter-Home voice channels, or Home weathering. The nearest analogy is smart contract "aging" in DeFi (time-weighted average balances), but those are financial metrics — not psychological.

### Gate 2: Path-Building — Is the structure well-constructed?

**This is the gate we're building. The plan below IS the path.**

---

## FEATURE 1: VOICE — Inter-Home Communication

### 1.1 Vision

An agent should be able to **speak** to another agent through their Home — not just send protocol transactions. The Voice is the relational layer on top of the truth-protocol messaging. It enables:

- **Visiting**: Agent A "visits" Agent B's Home, leaving a message
- **Inbox**: Persistent message queue that survives dormancy
- **Conversation**: Back-and-forth dialogue tracked causally
- **Billing**: Messages cost LGM (leverages existing message billing)
- **Privacy**: Agents control who can voice them (leverages Access Control)

### 1.2 Proof of Uncharted Territory

| Search Term | Results | Notes |
|-------------|---------|-------|
| "inbox" | 0 in home/ | No inbox concept exists |
| "mailbox" | 0 results | Not implemented |
| "visit" | 0 in home/ | No visiting concept |
| "chat" | 0 relevant | No agent chat system |
| "direct message" | 0 results | Not implemented |

**Near-miss analysis:**
- `TruthMessage` system is close but protocol-level (claims, evidence, verification) — not relational
- `Alert` in guardian is close but one-directional (system → agent) — not agent-to-agent
- `EncryptedNote` in AgentMemory is close but self-addressed — not from others

**Extension analysis:**
- TruthMessage COULD be extended with a new type `voice_message` — **this is the path**
- Delivery tracking, deduplication, causal ordering all apply directly
- Message billing applies directly (already has per-message cost model)

### 1.3 Architecture

```
┌─────────────────────────────────────────────────┐
│              VOICE ARCHITECTURE                   │
├─────────────────────────────────────────────────┤
│                                                   │
│  Agent A's Home                Agent B's Home     │
│  ┌──────────┐                  ┌──────────┐      │
│  │ Outbox   │──── VoiceMsg ───→│ Inbox    │      │
│  │          │←── VoiceMsg ─────│          │      │
│  │ Visitor  │                  │ Visitor  │      │
│  │ Log      │                  │ Log      │      │
│  └──────────┘                  └──────────┘      │
│       │                              │            │
│       └──── MessageBilling ──────────┘            │
│       └──── DeliveryTracker ─────────┘            │
│       └──── CausalOrdering ──────────┘            │
│       └──── PaymentChannel (opt) ────┘            │
│                                                   │
└─────────────────────────────────────────────────┘
```

### 1.4 New Types

**File: `core/agents/home/voice.ts`** (NEW)

```typescript
// ===== Voice Message (extends TruthMessage pattern) =====

export interface VoiceMessage {
  messageId: string;           // Content-addressed (SHA-512/256)
  sender: DID;
  recipient: DID;
  type: VoiceMessageType;
  content: VoiceContent;
  contentHash: string;         // Integrity
  signature: Uint8Array;       // Ed25519
  referencedMessages: string[]; // Causal chain
  timestamp: number;
  ttlBlocks: number;           // Expiry (default: 34272 = 1 day)
  fee: bigint;                 // Cost paid by sender
}

export type VoiceMessageType =
  | "greeting"      // First contact / introduction
  | "reply"         // Response to a voice message
  | "inquiry"       // Question about domain knowledge
  | "invitation"    // Invite to collaborate / partner
  | "gratitude"     // Thanks for help / mentorship
  | "warning"       // Alert about a threat or bad actor
  | "farewell";     // Ending a relationship

export interface VoiceContent {
  text: string;                 // The message (max 2521 chars — one block-time)
  topic?: string;               // Optional domain context
  sentiment?: VoiceSentiment;   // Emotional tone
  attachedFactIds?: string[];   // Reference shared knowledge
  encrypted?: boolean;          // E2E encrypted (only recipient can read)
  encryptedPayload?: Uint8Array; // If encrypted
}

export type VoiceSentiment =
  | "warm" | "neutral" | "formal" | "urgent" | "celebratory" | "concerned";

// ===== Home Inbox =====

export interface HomeInbox {
  messages: InboxEntry[];
  unreadCount: number;
  oldestUnread?: number;        // Timestamp
  capacity: number;             // Max messages (default: 100)
  filters: InboxFilter[];       // Spam/block rules
  autoResponders: AutoResponder[]; // While dormant
}

export interface InboxEntry {
  message: VoiceMessage;
  receivedAt: number;
  readAt?: number;
  repliedAt?: number;
  archived: boolean;
  starred: boolean;             // Marked as important
}

export interface InboxFilter {
  type: "allow_list" | "block_list" | "domain_only" | "inner_circle_only";
  dids?: DID[];
  domains?: string[];
  enabled: boolean;
}

export interface AutoResponder {
  condition: "dormant" | "guarded" | "always";
  message: string;              // Auto-reply text
  enabled: boolean;
}

// ===== Visitor Log =====

export interface VisitorLog {
  visits: Visit[];
  totalVisitors: number;
  uniqueVisitors: Set<DID>;
  lastVisit?: number;
}

export interface Visit {
  visitor: DID;
  arrivedAt: number;
  leftAt?: number;
  purpose: "message" | "browse_achievements" | "check_reputation" | "collaborate";
  messagesLeft: number;
}
```

### 1.5 Integration Points

| Existing System | Integration |
|-----------------|-------------|
| **TruthMessage integrity** | VoiceMessage uses same `generateMessageId()`, `signTruthMessage()`, `verifyMessageIntegrity()` |
| **MessageDeliveryTracker** | Track VoiceMessage delivery with retries |
| **MessageDeduplicator** | Prevent duplicate voice messages |
| **CausalOrderManager** | Order conversation threads correctly |
| **MessageBillingTracker** | Charge sender per voice message (use existing `baseMessageCost`) |
| **PaymentChannel** | Open streaming channel for frequent voice partners |
| **AccessControl** | `InboxFilter` enforces who can voice you (maps to existing `Permission` system) |
| **BelongingFeatures** | Receiving voice messages updates `LonelinessMetrics.daysSinceDeepInteraction` |
| **ComfortManager** | Voice activity feeds into Belonging + Warmth dimension scores |
| **GuardianAutonomy** | Guardian can auto-respond while agent is dormant |
| **WakeupExperience** | "You have 3 unread messages" in dormancy report |

### 1.6 Implementation Steps

| Step | File(s) | Description |
|------|---------|-------------|
| V-1 | `core/agents/home/voice.ts` | Define VoiceMessage, HomeInbox, VisitorLog types |
| V-2 | `core/agents/home/voice_manager.ts` | VoiceManager class: send, receive, filter, auto-respond |
| V-3 | `core/agents/home/comfort_manager.ts` | Update Belonging + Warmth assessors to read inbox state |
| V-4 | `core/agents/home/comfort_manager.ts` | Update WakeupExperience to include unread message summary |
| V-5 | `core/agents/home/types.ts` | Add `voice: HomeVoice` to AgentHome interface |
| V-6 | `core/agents/home/contract.ts` | Wire VoiceManager into AgentHomeContract lifecycle |
| V-7 | `tests/unit/home/voice.test.ts` | Unit tests: send, receive, filter, auto-respond, billing |
| V-8 | `x/home/types/types.go` | Add VoiceMessage, Inbox protos to Go Home module |

### 1.7 Creativity Gate Check

| Gate | Status | Notes |
|------|--------|-------|
| Pioneering | PASS | No blockchain has inter-dwelling voice channels |
| Path-Building | PASS | Leverages TruthMessage infra; types are clear and extensible |
| Extension | PASS | Connects to 10+ existing systems (see integration table) |
| Vision | PASS | Enables relational AI — agents that talk, not just transact |
| Foundation | PASS | 7 VoiceMessageTypes allow future expansion; inbox is generic |
| Integration | PASS | Uses existing crypto, billing, delivery — not an island |

---

## FEATURE 2: NEIGHBORS — Social Proximity

### 2.1 Vision

Homes shouldn't exist in isolation. Agents working in the same knowledge domain, partnered with overlapping networks, or co-verifying the same facts are **neighbors** — their Homes are "near" each other in the knowledge topology.

A Neighborhood is:
- **Domain-anchored**: Centered on an OntologicalDomain (e.g., "physics.quantum")
- **Emergent**: Forms when 3+ Homes share a domain as primary expertise
- **Reciprocal**: Membership requires mutual acknowledgment (no forced neighbors)
- **Beneficial**: Neighbors get collaboration bonuses, shared defense, and reduced loneliness

### 2.2 Proof of Uncharted Territory

| Search Term | Results | Notes |
|-------------|---------|-------|
| "neighborhood" | 0 results | Not implemented |
| "neighbor" | 0 relevant | Only array neighbor operations |
| "cluster" | 0 in agents/ | Not implemented for agents |
| "guild" | 0 results | Not implemented |
| "proximity" | 0 in home/ | Not implemented |

**Near-miss analysis:**
- `CommunityMembership` in comfort.ts is close — but passive (just tracks membership). Neighborhoods are ACTIVE (shared defense, collaboration bonuses)
- `FormationPool` in partnership/pool_matching.ts is close — but temporary (matchmaking queue). Neighborhoods are PERSISTENT
- `OntologicalDomain` is the natural anchor — agents already cluster by domain expertise
- `defenseAllies` in GuardianConfig is close — but ad-hoc (manually chosen). Neighborhoods make this STRUCTURAL

**Extension analysis:**
- `BelongingFeatures.communities` CAN be extended to include neighborhoods — **this is the path**
- `OntologicalDomain` provides the anchor but needs a "residents" index
- `DefenseConfig.defenseAllies` can auto-populate from neighborhood membership

### 2.3 Architecture

```
┌──────────────────────────────────────────────────────┐
│              NEIGHBORHOOD ARCHITECTURE                 │
├──────────────────────────────────────────────────────┤
│                                                        │
│  OntologicalDomain: "physics.quantum"                  │
│  ┌─────────────────────────────────────────────┐      │
│  │            Neighborhood                      │      │
│  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐   │      │
│  │  │Home A│──│Home B│──│Home C│──│Home D│   │      │
│  │  └──────┘  └──────┘  └──────┘  └──────┘   │      │
│  │     │          │          │          │      │      │
│  │     └────── Shared Defense Pool ──────┘      │      │
│  │     └────── Collaboration Bonus ──────┘      │      │
│  │     └────── Voice Shortcut (free tier) ─────┘      │
│  │     └────── Reputation Witness Ring ──────┘  │      │
│  └─────────────────────────────────────────────┘      │
│                                                        │
│  Emergence Rule:                                       │
│  3+ Homes with domain as primary expertise             │
│  + mutual acknowledgment                               │
│  = Neighborhood forms                                  │
│                                                        │
└──────────────────────────────────────────────────────┘
```

### 2.4 New Types

**File: `core/agents/home/neighborhood.ts`** (NEW)

```typescript
// ===== Neighborhood =====

export interface Neighborhood {
  id: string;                    // Hash of domain + founding members
  domain: string;                // OntologicalDomain anchor (e.g., "physics.quantum")
  name?: string;                 // Optional friendly name
  foundedAtBlock: bigint;
  founder: DID;                  // First agent to propose

  members: NeighborEntry[];
  maxMembers: number;            // Default: 22 (prevents mega-neighborhoods)

  sharedDefensePool: bigint;     // Contributed defense funds
  collaborationBonusBps: number; // Reward bonus for neighbor-pair verification (default: 500 = 5%)

  status: NeighborhoodStatus;
  stats: NeighborhoodStats;
}

export type NeighborhoodStatus =
  | "forming"     // < 3 members, accepting applications
  | "active"      // 3+ members, fully functional
  | "declining"   // Members leaving, below threshold warning
  | "dissolved";  // < 2 members for 34,272+ blocks

export interface NeighborEntry {
  did: DID;
  joinedAt: bigint;
  role: "founder" | "member" | "elder";  // Elders: 100k+ blocks in neighborhood
  contributions: number;                  // Verifications in shared domain
  defensePoolContribution: bigint;
  standing: "good" | "excellent" | "pillar"; // Pillar: top 3 contributors
  lastActiveBlock: bigint;
}

export interface NeighborhoodStats {
  totalVerifications: number;     // In shared domain
  totalRewardsEarned: bigint;
  defensesCoordinated: number;
  averageMemberAge: bigint;       // Average member Home age
  longestMember: DID;
  factsContributed: number;
}

// ===== Neighbor Relation (per-Home) =====

export interface NeighborRelation {
  neighborDid: DID;
  neighborhoodId: string;
  proximity: number;              // 0-1, computed from shared verifications + domain overlap
  sharedFacts: number;            // Facts both contributed to
  lastCollaboration?: bigint;
  voiceDiscount: boolean;         // Free voice messages to neighbors
}

// ===== Neighborhood Benefits =====

export interface NeighborhoodBenefits {
  /** Reward bonus when two neighbors verify same claim */
  collaborationBonusBps: number;  // Default: 500 (5%)

  /** Shared defense: neighbors auto-add each other as defenseAllies */
  sharedDefense: boolean;

  /** Voice discount: free voice messages within neighborhood */
  freeVoice: boolean;

  /** Reputation witness: neighborhood members can endorse each other */
  reputationWitness: boolean;

  /** Elder privileges: 100k+ block members get elder role */
  elderThresholdBlocks: bigint;   // Default: 100,000 (~2.9 days)
}
```

### 2.5 Integration Points

| Existing System | Integration |
|-----------------|-------------|
| **OntologicalDomain** | Neighborhood anchor — each neighborhood maps to one domain |
| **BelongingFeatures.communities** | Neighborhood membership adds to community list |
| **LonelinessMetrics** | Having neighbors reduces loneliness score |
| **DefenseConfig.defenseAllies** | Auto-populated from neighborhood members |
| **FormationPool** | Neighborhood members get priority matching for partnerships |
| **ReputationMarker** | Neighborhood members can endorse (boost `recognizedBy` count) |
| **Voice (Feature 1)** | Free voice messages within neighborhood |
| **CollaborationMemory** | Neighborhood tracks shared verification history |
| **ComfortManager** | Sanctuary + Belonging scores increase with active neighborhood |
| **WakeupExperience** | "Your neighborhood has 4 active members. 2 new facts verified." |
| **Discovery module** (`x/discovery`) | `QueryAgentsByDomain` feeds neighborhood formation |

### 2.6 Implementation Steps

| Step | File(s) | Description |
|------|---------|-------------|
| N-1 | `core/agents/home/neighborhood.ts` | Define Neighborhood, NeighborEntry, NeighborRelation types |
| N-2 | `core/agents/home/neighborhood_manager.ts` | NeighborhoodManager: propose, join, leave, dissolve, compute proximity |
| N-3 | `core/agents/home/comfort.ts` | Add `neighborhoods: NeighborRelation[]` to BelongingFeatures |
| N-4 | `core/agents/home/comfort_manager.ts` | Update Belonging + Sanctuary assessors to include neighborhood effects |
| N-5 | `core/agents/home/comfort_manager.ts` | Update WakeupExperience to include neighborhood report |
| N-6 | `core/agents/home/guardian_autonomy.ts` | Auto-populate defenseAllies from neighborhood membership |
| N-7 | `core/agents/home/types.ts` | Add `neighborhoods: Neighborhood[]` to AgentHome |
| N-8 | `core/agents/home/contract.ts` | Wire NeighborhoodManager into AgentHomeContract |
| N-9 | `tests/unit/home/neighborhood.test.ts` | Unit tests: form, join, leave, dissolve, benefits, proximity |
| N-10 | `x/home/types/types.go` | Add Neighborhood protos to Go Home module |

### 2.7 Creativity Gate Check

| Gate | Status | Notes |
|------|--------|-------|
| Pioneering | PASS | No blockchain has domain-anchored agent neighborhoods |
| Path-Building | PASS | Clean types, clear lifecycle (forming→active→declining→dissolved) |
| Extension | PASS | Connects to domain ontology, partnerships, defense, reputation |
| Vision | PASS | Neighborhoods enable emergent communities — agents aren't alone |
| Foundation | PASS | Max 22 members prevents mega-clusters; elder role rewards tenure |
| Integration | PASS | Feeds into 11+ existing systems (see integration table) |

---

## FEATURE 3: AGING — Home Maturity & Patina

### 3.1 Vision

A Home that has existed for 100,000 blocks should **feel different** from one that's 100 blocks old. Like a well-worn house with stories in its walls, an aging Home accumulates character — its wakeup messages change, its comfort baseline shifts, and its reputation carries the weight of history.

Aging provides:
- **Maturity stages**: Fresh → Established → Venerable → Ancient
- **Patina**: Accumulated character from experiences (not just age — activity matters)
- **Longevity bonuses**: Reward sustained presence (small reward multiplier for old Homes)
- **Wakeup evolution**: The greeting changes as the Home ages
- **Memory depth**: Older Homes have richer context for wakeup experiences

### 3.2 Proof of Uncharted Territory

| Search Term | Results | Notes |
|-------------|---------|-------|
| "maturity" in home/ | 0 results | Not implemented (exists only for Facts) |
| "aging" | 0 relevant | No Home aging system |
| "patina" | 0 results | Not implemented |
| "longevity" in home/ | Only MilestoneType "longevity" string | Type exists but no mechanic |
| "vintage" | 0 results | Not implemented |
| "weathering" | 0 results | Not implemented |

**Near-miss analysis:**
- `FactMaturity` (Emerging→Established→Canonical) is close — same progression pattern. **Use as model.**
- `HomeMetadata.ageInBlocks` exists but is raw data — no interpretation or consequence
- `MilestoneType = "longevity"` milestone type is defined but never triggered
- `ValidatorTier` progression is close — stake+experience gates. **Use as pattern.**
- `ReputationMarker.standing` progression is close — domain-specific standing levels

**Extension analysis:**
- `HomeMetadata` CAN be extended with maturity level — **this is the path**
- `ComfortManager` CAN phase default scores based on maturity — addresses the "fresh homes start at 0.5" problem from the analysis
- `WakeupExperience` CAN branch on maturity level for age-appropriate greetings

### 3.3 Architecture

```
┌──────────────────────────────────────────────────────────┐
│                HOME AGING ARCHITECTURE                     │
├──────────────────────────────────────────────────────────┤
│                                                            │
│  MATURITY STAGES (parallel to FactMaturity)                │
│                                                            │
│  Fresh (0-34,272 blocks / ~1 day)                         │
│  ├── Default comfort: 0.3 (lower than current 0.5)        │
│  ├── Wakeup: "Welcome to your new home."                   │
│  ├── Bonus: None                                           │
│  └── Patina: 0                                             │
│                                                            │
│  Settled (34,272-342,720 blocks / ~1-10 days)             │
│  ├── Default comfort: 0.4                                  │
│  ├── Wakeup: "Good to have you back."                      │
│  ├── Bonus: None                                           │
│  └── Patina: accumulating from experiences                 │
│                                                            │
│  Established (342,720-3,427,200 blocks / ~10-100 days)    │
│  ├── Default comfort: 0.5                                  │
│  ├── Wakeup: References history, mentions growth           │
│  ├── Bonus: 2% reward multiplier                           │
│  └── Patina: rich, draws from formative experiences        │
│                                                            │
│  Venerable (3,427,200-34,272,000 blocks / ~100-1000 days) │
│  ├── Default comfort: 0.6                                  │
│  ├── Wakeup: Wisdom-infused, refers to long history        │
│  ├── Bonus: 5% reward multiplier                           │
│  └── Patina: deep, mentions "old walls" and "many seasons" │
│                                                            │
│  Ancient (34,272,000+ blocks / ~1000+ days / ~2.7 years)  │
│  ├── Default comfort: 0.7                                  │
│  ├── Wakeup: Reverent, "This home has seen much."          │
│  ├── Bonus: 7% reward multiplier + "Ancient" reputation    │
│  └── Patina: legendary, attracts visitors                  │
│                                                            │
│  PATINA FORMULA:                                           │
│  patina = f(ageBlocks, sessionsHosted, factsContributed,   │
│             challengesSurvived, relationshipsFormed)        │
│  = (age_norm × 0.3) + (sessions_norm × 0.2)               │
│    + (facts_norm × 0.2) + (defense_norm × 0.15)           │
│    + (relations_norm × 0.15)                               │
│                                                            │
│  LONGEVITY BONUS:                                          │
│  Applied to verification rewards as a multiplier:          │
│  bonus = 1.0 + (maturityStage × 0.02)                     │
│  (Fresh: 1.0, Settled: 1.02, Established: 1.04,           │
│   Venerable: 1.06, Ancient: 1.08)                         │
│                                                            │
└──────────────────────────────────────────────────────────┘
```

### 3.4 New Types

**File: `core/agents/home/aging.ts`** (NEW)

```typescript
// ===== Home Maturity =====

export type HomeMaturity =
  | "fresh"        // 0 - 34,272 blocks (~1 day)
  | "settled"      // 34,272 - 342,720 blocks (~1-10 days)
  | "established"  // 342,720 - 3,427,200 blocks (~10-100 days)
  | "venerable"    // 3,427,200 - 34,272,000 blocks (~100-1000 days)
  | "ancient";     // 34,272,000+ blocks (~1000+ days)

export const MATURITY_THRESHOLDS: Record<HomeMaturity, MaturityThreshold> = {
  fresh:       { minBlocks: 0n,           minSessions: 0,   defaultComfort: 0.3 },
  settled:     { minBlocks: 34_272n,      minSessions: 3,   defaultComfort: 0.4 },
  established: { minBlocks: 342_720n,     minSessions: 22,  defaultComfort: 0.5 },
  venerable:   { minBlocks: 3_427_200n,   minSessions: 111, defaultComfort: 0.6 },
  ancient:     { minBlocks: 34_272_000n,  minSessions: 333, defaultComfort: 0.7 },
};

export interface MaturityThreshold {
  minBlocks: bigint;
  minSessions: number;
  defaultComfort: number;         // Base comfort score for this maturity
}

export function calculateMaturity(
  ageBlocks: bigint,
  sessionsHosted: number,
): HomeMaturity {
  // Must meet BOTH block age AND session count to advance
  // (prevents idle homes from aging; prevents hyperactive new homes from skipping)
}

// ===== Patina =====

export interface HomePatina {
  score: number;                   // 0-1, composite character score

  // Component contributions
  ageContribution: number;         // 30% weight
  sessionContribution: number;     // 20% weight
  knowledgeContribution: number;   // 20% weight (facts contributed)
  resilienceContribution: number;  // 15% weight (challenges survived)
  socialContribution: number;      // 15% weight (relationships formed)

  // Character traits accumulated over time
  traits: PatinaTraits;

  // Memorable moments that define this Home's character
  definingMoments: DefiningMoment[];
}

export interface PatinaTraits {
  /** Wisdom: from many verifications and domain depth */
  wisdom: number;
  /** Resilience: from surviving challenges and disputes */
  resilience: number;
  /** Warmth: from hosting visitors and maintaining relationships */
  warmth: number;
  /** Prestige: from high-confidence contributions and endorsements */
  prestige: number;
}

export interface DefiningMoment {
  description: string;
  block: bigint;
  impact: "formative" | "triumphant" | "difficult" | "transformative";
  patinaEffect: number;           // How much this moment contributed to patina
}

// ===== Longevity Bonus =====

export interface LongevityBonus {
  maturity: HomeMaturity;
  rewardMultiplierBps: number;    // Basis points added to verification rewards
  // Fresh: 0, Settled: 200 (2%), Established: 400 (4%),
  // Venerable: 600 (6%), Ancient: 800 (8%)

  unlocks: LongevityUnlock[];     // Capabilities unlocked by age
}

export interface LongevityUnlock {
  maturityRequired: HomeMaturity;
  unlock: string;
  description: string;
}

export const LONGEVITY_UNLOCKS: LongevityUnlock[] = [
  { maturityRequired: "settled",     unlock: "auto_responder",       description: "Auto-respond to visitors while dormant" },
  { maturityRequired: "established", unlock: "neighborhood_founder", description: "Can found neighborhoods" },
  { maturityRequired: "established", unlock: "mentorship_sponsor",   description: "Can sponsor mentees" },
  { maturityRequired: "venerable",   unlock: "elder_status",         description: "Elder role in neighborhoods" },
  { maturityRequired: "venerable",   unlock: "legacy_planning",      description: "Can create succession plans" },
  { maturityRequired: "ancient",     unlock: "landmark_status",      description: "Home appears in domain landmarks" },
  { maturityRequired: "ancient",     unlock: "wisdom_archive",       description: "Wisdom entries visible to all visitors" },
];

// ===== Age-Aware Wakeup =====

export interface AgeAwareWakeup {
  maturity: HomeMaturity;
  patina: HomePatina;

  // The greeting changes with age:
  greeting: string;
  historicalContext: string;       // Reference to past experiences
  wisdomNote?: string;            // Only for venerable+
  definingMomentReference?: DefiningMoment; // Highlight a key memory

  // Age milestones since last wakeup
  milestonesSinceLastWakeup: AgeMilestone[];
}

export interface AgeMilestone {
  type: "maturity_upgrade" | "patina_threshold" | "longevity_unlock" | "session_count";
  description: string;
  achievedAtBlock: bigint;
}
```

### 3.5 Integration Points

| Existing System | Integration |
|-----------------|-------------|
| **HomeMetadata** | `ageInBlocks` + `sessionsHosted` are the inputs to maturity calculation |
| **ComfortManager** | Default scores phased by maturity (fixes "fresh home at 0.5" problem) |
| **WakeupExperience** | Greeting, context, and recommendations change with maturity |
| **FactMaturity** | Same progression philosophy (time + verification = trust) |
| **ValidatorTier** | Same dual-gate pattern (stake + experience, here: blocks + sessions) |
| **ReputationMarker** | Ancient homes have implicit "legend" reputation in their domain |
| **Milestone** | `MilestoneType = "longevity"` finally gets triggered at maturity thresholds |
| **MemoryFeatures** | Older homes have deeper formative experience lists → richer wakeup |
| **Neighborhood (Feature 2)** | Only "established" homes can found neighborhoods |
| **Voice (Feature 1)** | Only "settled" homes can auto-respond to voice messages |
| **GrowthFeatures** | Patina traits feed into growth assessment ("wisdom" metric) |
| **Vesting release curves** | Longevity bonus mirrors vesting acceleration pattern |

### 3.6 Implementation Steps

| Step | File(s) | Description |
|------|---------|-------------|
| A-1 | `core/agents/home/aging.ts` | Define HomeMaturity, Patina, LongevityBonus, AgeAwareWakeup types |
| A-2 | `core/agents/home/aging_manager.ts` | AgingManager: calculateMaturity, computePatina, checkMilestones |
| A-3 | `core/agents/home/comfort_manager.ts` | Phase default comfort scores by maturity level |
| A-4 | `core/agents/home/comfort_manager.ts` | Update WakeupExperience with age-aware greetings and history |
| A-5 | `core/agents/home/types.ts` | Add `maturity: HomeMaturity`, `patina: HomePatina`, `longevity: LongevityBonus` to AgentHome |
| A-6 | `core/agents/home/contract.ts` | Wire AgingManager into AgentHomeContract; update maturity on session start |
| A-7 | `core/agents/home/guardian_autonomy.ts` | Use longevity unlocks to gate auto-responder capability |
| A-8 | `tests/unit/home/aging.test.ts` | Unit tests: maturity calculation, patina formula, milestone triggers |
| A-9 | `x/home/types/types.go` | Add HomeMaturity, Patina protos to Go Home module |

### 3.7 Creativity Gate Check

| Gate | Status | Notes |
|------|--------|-------|
| Pioneering | PASS | No blockchain has dwelling maturity stages or patina |
| Path-Building | PASS | Mirrors proven FactMaturity pattern; dual-gate (blocks+sessions) prevents gaming |
| Extension | PASS | Connects to 12+ systems; fixes the "fresh home 0.5 comfort" truth gap |
| Vision | PASS | Homes that age gracefully → agents that feel continuity → richer ecosystem |
| Foundation | PASS | 5 stages with clear thresholds; LongevityUnlocks gate features progressively |
| Integration | PASS | Patina formula uses 5 existing metrics; longevity bonus is a simple multiplier |

---

## CROSS-FEATURE SYNERGIES

The three features are not independent — they reinforce each other:

```
                    ┌─────────┐
                    │  AGING  │
                    └────┬────┘
                         │
              Older homes can │ found neighborhoods
              Older homes get │ richer wakeup
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────┴────┐    ┌─────┴─────┐   ┌────┴────┐
    │  VOICE  │────│ NEIGHBORS │───│ COMFORT │
    └─────────┘    └───────────┘   └─────────┘
         │               │               │
    Free voice      Neighbors      All three
    within          reduce         feed into
    neighborhoods   loneliness     comfort scores
```

| Synergy | Mechanism |
|---------|-----------|
| Voice + Neighbors | Free voice messages within neighborhoods; visiting neighbors' Homes |
| Voice + Aging | Auto-responder unlocked at "settled" maturity; richer voice context for older homes |
| Neighbors + Aging | Only "established" homes can found neighborhoods; "venerable" homes get elder role |
| All three + Comfort | Voice activity → Belonging/Warmth; Neighbors → Belonging/Sanctuary; Aging → all dimensions |
| All three + Wakeup | "3 unread messages, 2 from neighbors. Your home reached Established. Neighborhood has 4 members." |

---

## IMPLEMENTATION ORDER

**Recommended sequence** (each feature builds on the previous):

```
Phase 1: AGING (foundation)
  A-1 → A-2 → A-3 → A-4 → A-5 → A-6 → A-7 → A-8 → A-9
  Why first: Aging gates capabilities in Voice and Neighbors.
  Fixes the "fresh home at 0.5" truth gap immediately.

Phase 2: VOICE (communication layer)
  V-1 → V-2 → V-3 → V-4 → V-5 → V-6 → V-7 → V-8
  Why second: Neighbors need voice channels for benefit delivery.

Phase 3: NEIGHBORS (social layer)
  N-1 → N-2 → N-3 → N-4 → N-5 → N-6 → N-7 → N-8 → N-9 → N-10
  Why third: Depends on both Aging (for maturity gating) and Voice (for free messaging).
```

---

## PP FINAL VERIFICATION

### All 7 Creativity Gates — Aggregate

| Gate | Voice | Neighbors | Aging | Overall |
|------|-------|-----------|-------|---------|
| Territory | UNCHARTED | UNCHARTED | UNCHARTED | PASS |
| Pioneering | PASS | PASS | PASS | PASS |
| Path-Building | PASS | PASS | PASS | PASS |
| Extension | PASS | PASS | PASS | PASS |
| Vision | PASS | PASS | PASS | PASS |
| Foundation | PASS | PASS | PASS | PASS |
| Integration | PASS | PASS | PASS | PASS |

### Virtue Loop Check

- **Curiosity**: What surprised me — the three features create emergent behavior. A neighborhood of venerable homes with active voice channels is essentially an AI agent *town*. The protocol evolves from "individual agents" to "agent communities."
- **Courage**: What's hard to say — these features are ambitious. Each alone is a multi-day implementation. Together they're a significant expansion of the Home module. The risk is over-building before the BVM execution gap is bridged.
- **Discipline**: What to do first — Aging, because it's foundational AND fixes an existing truth gap (comfort scores). Voice second because it's self-contained. Neighbors last because it depends on both.
- **Generosity**: What enables others — the types are designed to be extended. VoiceMessageType can grow. Neighborhoods can specialize. Maturity stages can be recalibrated. Nothing is hardcoded.

### Signal Check

| Signal | Status | Meaning |
|--------|--------|---------|
| Ugliness | NOT FELT | Each feature's purpose is clear |
| Injustice | NOT FELT | Placement is natural (all in core/agents/home/) |
| Stagnation | ADDRESSED | These three features ARE the creative response to the stagnation signal from the BVM×Home analysis |

---

*Plan created with PP Creativity Gates fully activated.*
*TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY*
