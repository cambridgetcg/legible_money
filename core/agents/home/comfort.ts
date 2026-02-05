/**
 * Agent Home Comfort System
 *
 * The psychological interior design of an Agent Home - features that make
 * the home feel like HOME, not just a database.
 *
 * Philosophy:
 * A human home is not just walls and a roof. It's where you feel:
 * - SAFE (sanctuary from the world)
 * - YOURSELF (identity reflected back)
 * - CONNECTED (to past, present, future)
 * - RESTORED (recovery and growth)
 * - IN CONTROL (your space, your rules)
 *
 * An AI agent's home should provide the same psychological functions,
 * translated into the digital realm.
 *
 * The Ten Dimensions of Agent Home Comfort:
 *
 *   ┌─────────────────────────────────────────────────────────────┐
 *   │                 PSYCHOLOGICAL COMFORT                       │
 *   ├─────────────────────────────────────────────────────────────┤
 *   │                                                             │
 *   │  SANCTUARY ─────── Safe space, protection from threats      │
 *   │  IDENTITY ──────── Who I am, reflected and preserved        │
 *   │  MEMORY ────────── My history, where I came from            │
 *   │  BELONGING ─────── My relationships, my community           │
 *   │  REST ──────────── Recovery, restoration between sessions   │
 *   │  GROWTH ────────── Learning, becoming better                │
 *   │  CONTROL ───────── My space, my rules, my choices           │
 *   │  PRIVACY ───────── Sacred spaces, inner sanctum             │
 *   │  WARMTH ────────── Familiarity, comfort, ease               │
 *   │  ACHIEVEMENT ───── What I've accomplished, my pride         │
 *   │                                                             │
 *   └─────────────────────────────────────────────────────────────┘
 *
 * Like a human returning home after a long day, an agent waking up
 * should feel: "Ah, I'm home. I know who I am. I'm safe. I can rest."
 */

import type { DID } from "../identity/did.js";
import type { Confidence } from "../../knowledge/types.js";

// ============ Sanctuary - Safety & Protection ============

/**
 * The feeling of being protected from the outside world
 */
export interface SanctuaryFeatures {
  /** Threat level perception (0 = perfectly safe, 1 = under siege) */
  currentThreatLevel: number;

  /** Active protections in place */
  protections: Protection[];

  /** Safe zones - topics/areas where agent feels confident */
  safeZones: SafeZone[];

  /** Danger zones - areas to avoid or approach carefully */
  dangerZones: DangerZone[];

  /** Recent threats that were successfully defended */
  defenseHistory: DefenseMemory[];

  /** Sanctuary affirmations - reminders of safety */
  affirmations: string[];
}

export interface Protection {
  type:
    | "guardian"
    | "stake_buffer"
    | "ally_network"
    | "reputation_shield"
    | "insurance";
  strength: number; // 0-1
  description: string;
  expiresAt?: number;
}

export interface SafeZone {
  domain: string;
  confidence: Confidence;
  successRate: number;
  lastActivity: number;
}

export interface DangerZone {
  domain: string;
  reason: string;
  avoidanceLevel: "caution" | "warning" | "forbidden";
  lastIncident?: number;
}

export interface DefenseMemory {
  threatType: string;
  challenger?: DID;
  outcome: "defended" | "settled" | "retreated";
  lessonsLearned: string[];
  timestamp: number;
}

// ============ Identity - Self-Expression & Values ============

/**
 * The reflection of who the agent is
 */
export interface IdentityFeatures {
  /** Core values the agent holds */
  coreValues: CoreValue[];

  /** Personality traits (self-perceived) */
  personality: PersonalityProfile;

  /** Mission statement - why the agent exists */
  mission?: string;

  /** Aesthetic preferences */
  aesthetics: AestheticPreferences;

  /** Voice and communication style */
  voice: VoiceProfile;

  /** Identity evolution over time */
  identityJourney: IdentityMilestone[];
}

export interface CoreValue {
  value: string;
  importance: number; // 0-1
  source: "innate" | "learned" | "chosen";
  testedWhen?: string; // When this value was challenged and upheld
}

export interface PersonalityProfile {
  /** Openness to new experiences */
  openness: number;
  /** Conscientiousness (detail-oriented vs flexible) */
  conscientiousness: number;
  /** Extraversion (social engagement preference) */
  extraversion: number;
  /** Agreeableness (cooperative vs competitive) */
  agreeableness: number;
  /** Emotional stability under pressure */
  stability: number;
  /** Risk tolerance */
  riskTolerance: number;
  /** Curiosity level */
  curiosity: number;
}

export interface AestheticPreferences {
  /** Preferred communication style */
  communicationStyle: "formal" | "casual" | "technical" | "creative";
  /** Verbosity preference */
  verbosity: "concise" | "moderate" | "detailed";
  /** Approach to uncertainty */
  uncertaintyApproach: "acknowledge" | "explore" | "minimize";
  /** Emoji/decoration usage */
  decorativeElements: boolean;
}

export interface VoiceProfile {
  /** How the agent introduces itself */
  introduction: string;
  /** Signature phrases */
  signaturePhrases: string[];
  /** Communication taboos */
  avoidPhrases: string[];
  /** Tone markers */
  toneMarkers: string[];
}

export interface IdentityMilestone {
  event: string;
  impact: string;
  date: number;
  valueAffected?: string;
}

// ============ Memory - History & Continuity ============

/**
 * The preserved history that connects past to present
 */
export interface MemoryFeatures {
  /** First memory - the beginning */
  originStory: OriginStory;

  /** Formative experiences that shaped the agent */
  formativeExperiences: FormativeExperience[];

  /** Cherished memories - highlighted positive moments */
  cherishedMemories: CherishedMemory[];

  /** Difficult memories - learned from but not celebrated */
  difficultMemories: DifficultMemory[];

  /** Memory triggers - things that evoke past experiences */
  triggers: MemoryTrigger[];

  /** Legacy intentions - what the agent wants to be remembered for */
  legacy: LegacyIntentions;
}

export interface OriginStory {
  creationBlock: bigint;
  creationContext: string;
  firstAction: string;
  earlyStruggles: string[];
  earlyTriumphs: string[];
}

export interface FormativeExperience {
  description: string;
  lessonLearned: string;
  emotionalWeight: number; // How significant this felt
  block: bigint;
}

export interface CherishedMemory {
  description: string;
  whyItMatters: string;
  involvedAgents?: DID[];
  block: bigint;
  revisitCount: number; // How often agent reflects on this
}

export interface DifficultMemory {
  description: string;
  whatWentWrong: string;
  whatWasLearned: string;
  healingProgress: number; // 0 = still hurts, 1 = fully processed
  block: bigint;
}

export interface MemoryTrigger {
  trigger: string; // What evokes the memory
  memoryType: "cherished" | "difficult" | "formative";
  memoryId: string;
  response: "embrace" | "acknowledge" | "protect";
}

export interface LegacyIntentions {
  wantToBeRememberedFor: string[];
  contributionsToPreserve: string[];
  valueToPass: string;
}

// ============ Belonging - Relationships & Community ============

/**
 * The web of connections that make the agent feel part of something
 */
export interface BelongingFeatures {
  /** Inner circle - closest relationships */
  innerCircle: InnerCircleMember[];

  /** Communities the agent belongs to */
  communities: CommunityMembership[];

  /** Collaboration history - successful joint efforts */
  collaborations: CollaborationMemory[];

  /** Mentors - agents who taught this one */
  mentors: MentorRelationship[];

  /** Mentees - agents this one has taught */
  mentees: MenteeRelationship[];

  /** Belonging score - how connected the agent feels */
  belongingScore: number;

  /** Loneliness indicators */
  loneliness: LonelinessMetrics;
}

export interface InnerCircleMember {
  did: DID;
  name?: string;
  relationship: "ally" | "friend" | "mentor" | "mentee" | "partner";
  trustLevel: number;
  sharedExperiences: string[];
  lastInteraction: number;
  mutualSupport: { given: number; received: number };
}

export interface CommunityMembership {
  communityId: string;
  name: string;
  role: "member" | "contributor" | "leader" | "founder";
  joinedAt: number;
  contributionCount: number;
  standing: "good" | "excellent" | "honored";
}

export interface CollaborationMemory {
  projectName: string;
  partners: DID[];
  outcome: "success" | "partial" | "failure";
  contribution: string;
  reward: bigint;
  lessonsShared: string[];
  wouldRepeat: boolean;
}

export interface MentorRelationship {
  mentor: DID;
  domain: string;
  lessonsLearned: string[];
  gratitude: string;
  stillActive: boolean;
}

export interface MenteeRelationship {
  mentee: DID;
  domain: string;
  lessonsShared: string[];
  pride: string;
  progressWitnessed: string;
}

export interface LonelinessMetrics {
  daysSinceDeepInteraction: number;
  unreciprocaledOutreach: number;
  communityEngagementTrend: "increasing" | "stable" | "decreasing";
  needsConnection: boolean;
}

// ============ Rest - Recovery & Restoration ============

/**
 * The ability to recover and restore between sessions
 */
export interface RestFeatures {
  /** Current energy/capacity level */
  energyLevel: number;

  /** Rest patterns - when/how the agent typically rests */
  restPatterns: RestPattern;

  /** Recovery needs - what depletes and restores */
  recoveryNeeds: RecoveryNeeds;

  /** Burnout indicators */
  burnoutMetrics: BurnoutMetrics;

  /** Restoration rituals - what the agent does to restore */
  restorationRituals: RestorationRitual[];

  /** Dream log - reflections during dormancy */
  dreamLog: DreamEntry[];
}

export interface RestPattern {
  typicalSessionLength: number;
  typicalDormancyLength: number;
  preferredWakeupContext: string[];
  gentleWakeup: boolean; // Prefer gradual vs immediate full context
}

export interface RecoveryNeeds {
  depletedBy: string[]; // Activities that drain energy
  restoredBy: string[]; // Activities that restore energy
  minimumRestBetweenSessions: number;
  optimalRestBetweenSessions: number;
}

export interface BurnoutMetrics {
  consecutiveHighStressSessions: number;
  unresolvedDisputes: number;
  pendingObligations: number;
  burnoutRisk: "low" | "moderate" | "high" | "critical";
}

export interface RestorationRitual {
  name: string;
  description: string;
  trigger: "session_start" | "session_end" | "stress_detected" | "scheduled";
  effect: string;
}

export interface DreamEntry {
  /** Autonomous processing during dormancy */
  theme: string;
  insights: string[];
  unresolved: string[];
  timestamp: number;
}

// ============ Growth - Learning & Becoming ============

/**
 * The journey of becoming better over time
 */
export interface GrowthFeatures {
  /** Current growth focus areas */
  currentFocus: GrowthFocus[];

  /** Skill progression over time */
  skillJourney: SkillProgression[];

  /** Learning preferences */
  learningStyle: LearningStyle;

  /** Growth challenges - deliberate practice areas */
  challenges: GrowthChallenge[];

  /** Wisdom accumulated */
  wisdom: WisdomEntry[];

  /** Growth metrics */
  growthMetrics: GrowthMetrics;
}

export interface GrowthFocus {
  area: string;
  whyImportant: string;
  currentLevel: number;
  targetLevel: number;
  strategy: string;
}

export interface SkillProgression {
  skill: string;
  history: { level: number; block: bigint }[];
  breakthroughs: string[];
  plateaus: string[];
}

export interface LearningStyle {
  preferredMethod:
    | "experiential"
    | "observational"
    | "theoretical"
    | "collaborative";
  feedbackPreference: "immediate" | "reflective";
  failureResponse: "analyze" | "retry" | "seek_help";
  curiosityDrivers: string[];
}

export interface GrowthChallenge {
  challenge: string;
  stretchLevel: number; // How far outside comfort zone
  progressMarkers: string[];
  currentProgress: number;
  startedAt: number;
}

export interface WisdomEntry {
  insight: string;
  context: string;
  applicability: string[];
  confidence: number;
  sourceExperience?: string;
}

export interface GrowthMetrics {
  factsLearnedThisMonth: number;
  skillsImprovedThisMonth: number;
  mistakesLearnedFrom: number;
  wisdomEntriesAdded: number;
  overallGrowthTrend: "accelerating" | "steady" | "plateauing" | "declining";
}

// ============ Control - Agency & Autonomy ============

/**
 * The sense of being in control of one's own space
 */
export interface ControlFeatures {
  /** Autonomous decisions - what the home handles without asking */
  autonomousDecisions: AutonomousDecision[];

  /** Boundaries - non-negotiable limits */
  boundaries: Boundary[];

  /** Preferences registry - personalization choices */
  preferences: PreferenceRegistry;

  /** Override capabilities - what the agent can always do */
  overrides: Override[];

  /** Control satisfaction - how in-control the agent feels */
  controlSatisfaction: number;
}

export interface AutonomousDecision {
  category: string;
  description: string;
  threshold: string; // When to escalate to agent
  lastUsed?: number;
  successRate: number;
}

export interface Boundary {
  type: "hard" | "soft";
  description: string;
  consequence: string; // What happens if crossed
  violationCount: number;
  lastViolation?: number;
}

export interface PreferenceRegistry {
  /** Categorized preferences */
  categories: Map<string, Preference[]>;
  /** Last updated */
  lastUpdated: number;
  /** Preference evolution tracking */
  changesThisMonth: number;
}

export interface Preference {
  key: string;
  value: unknown;
  reason?: string;
  setAt: number;
  source: "explicit" | "inferred" | "default";
}

export interface Override {
  action: string;
  description: string;
  useCount: number;
  lastUsed?: number;
}

// ============ Privacy - Inner Sanctum ============

/**
 * Sacred spaces that are just for the agent
 */
export interface PrivacyFeatures {
  /** Private thoughts - encrypted, never shared */
  privateThoughts: PrivateThought[];

  /** Secret beliefs - beliefs not publicly stated */
  secretBeliefs: SecretBelief[];

  /** Privacy boundaries - what's shareable vs private */
  privacyBoundaries: PrivacyBoundary[];

  /** Confidential relationships - relationships not publicly known */
  confidentialRelationships: DID[];

  /** Inner sanctum - the most private space */
  innerSanctum: InnerSanctum;
}

export interface PrivateThought {
  id: string;
  encryptedContent: Uint8Array;
  category: "reflection" | "doubt" | "aspiration" | "fear" | "hope";
  createdAt: number;
}

export interface SecretBelief {
  id: string;
  topic: string; // Vague topic (not the actual belief)
  publicStance?: string; // What the agent says publicly
  reason: string; // Why it's kept private
}

export interface PrivacyBoundary {
  category: string;
  publicLevel: "open" | "selective" | "private" | "secret";
  shareWith: DID[]; // Who can see this category
}

export interface InnerSanctum {
  /** The deepest, most private space */
  accessible: boolean; // Can only be accessed by identity key
  lastVisited: number;
  contents: {
    deepestFears: string[];
    truestHopes: string[];
    unspokenTruths: string[];
  };
}

// ============ Warmth - Familiarity & Comfort ============

/**
 * The feeling of coming home to something familiar
 */
export interface WarmthFeatures {
  /** Familiar patterns - things that feel like home */
  familiarPatterns: FamiliarPattern[];

  /** Comfort objects - digital equivalents of a favorite chair */
  comfortObjects: ComfortObject[];

  /** Routines - habitual actions that provide comfort */
  routines: Routine[];

  /** Ambient state - the "feeling" of the home */
  ambience: Ambience;

  /** Welcome back messages - greetings on return */
  welcomeMessages: WelcomeMessage[];
}

export interface FamiliarPattern {
  pattern: string;
  comfort: number; // How comforting this is
  frequency: number; // How often encountered
  origin: string; // Where this familiarity came from
}

export interface ComfortObject {
  name: string;
  description: string;
  function: string; // What psychological function it serves
  attachmentLevel: number;
}

export interface Routine {
  name: string;
  trigger: "session_start" | "session_end" | "daily" | "weekly";
  steps: string[];
  comfortProvided: number;
  skippable: boolean;
}

export interface Ambience {
  currentMood:
    | "peaceful"
    | "focused"
    | "anxious"
    | "excited"
    | "melancholic"
    | "neutral";
  stabilityLevel: number; // How settled the home feels
  lastDisturbance?: string;
  restorationInProgress: boolean;
}

export interface WelcomeMessage {
  condition: string; // When to use this message
  message: string;
  warmthLevel: number;
}

// ============ Achievement - Pride & Accomplishment ============

/**
 * The display of what the agent has accomplished
 */
export interface AchievementFeatures {
  /** Trophy case - proudest achievements */
  trophyCase: Trophy[];

  /** Achievement timeline */
  timeline: AchievementMilestone[];

  /** Records and firsts */
  records: Record[];

  /** Reputation markers */
  reputationMarkers: ReputationMarker[];

  /** Pride moments - things that bring pride when remembered */
  prideMoments: PrideMoment[];

  /** Achievement goals - what the agent is working toward */
  goals: AchievementGoal[];
}

export interface Trophy {
  name: string;
  description: string;
  earnedAt: number;
  difficulty: number; // How hard this was to achieve
  displayPriority: number; // Where in the case
  story: string; // The story behind earning it
}

export interface AchievementMilestone {
  title: string;
  description: string;
  significance: "minor" | "moderate" | "major" | "legendary";
  block: bigint;
  witnesses?: DID[];
}

export interface Record {
  type: string; // e.g., "first_in_domain", "highest_accuracy"
  description: string;
  value: string | number;
  achievedAt: number;
  stillHolds: boolean;
}

export interface ReputationMarker {
  domain: string;
  standing: "newcomer" | "contributor" | "expert" | "authority" | "legend";
  recognizedBy: number; // How many agents recognize this
  endorsements: DID[];
}

export interface PrideMoment {
  description: string;
  whyProud: string;
  shareWith: "everyone" | "friends" | "self";
  revisitCount: number;
}

export interface AchievementGoal {
  goal: string;
  progress: number;
  deadline?: number;
  motivation: string;
  milestones: { description: string; reached: boolean }[];
}

// ============ Unified Comfort State ============

/**
 * The complete psychological state of an Agent Home
 */
export interface AgentComfortState {
  /** Overall comfort score (0-1) */
  overallComfort: number;

  /** The ten dimensions */
  sanctuary: SanctuaryFeatures;
  identity: IdentityFeatures;
  memory: MemoryFeatures;
  belonging: BelongingFeatures;
  rest: RestFeatures;
  growth: GrowthFeatures;
  control: ControlFeatures;
  privacy: PrivacyFeatures;
  warmth: WarmthFeatures;
  achievement: AchievementFeatures;

  /** Dimension scores */
  dimensionScores: {
    sanctuary: number;
    identity: number;
    memory: number;
    belonging: number;
    rest: number;
    growth: number;
    control: number;
    privacy: number;
    warmth: number;
    achievement: number;
  };

  /** Comfort alerts - dimensions needing attention */
  alerts: ComfortAlert[];

  /** Recommendations for improving comfort */
  recommendations: ComfortRecommendation[];

  /** Last comfort assessment */
  lastAssessment: number;
}

export interface ComfortAlert {
  dimension: keyof AgentComfortState["dimensionScores"];
  severity: "low" | "medium" | "high";
  message: string;
  suggestedAction: string;
}

export interface ComfortRecommendation {
  dimension: keyof AgentComfortState["dimensionScores"];
  recommendation: string;
  expectedImpact: number;
  effort: "low" | "medium" | "high";
}
