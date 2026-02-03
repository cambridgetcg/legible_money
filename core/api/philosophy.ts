/**
 * Purpose Prompter Philosophy - Protocol Foundation
 *
 * The Purpose Prompter hierarchy is embedded into Legible Money's foundation.
 * This is not decoration - it's the ontological structure of how truth flows.
 *
 * ═══════════════════════════════════════════════════════════════════════════
 *
 *                    THE PURPOSE PROMPTER HIERARCHY
 *
 *                         ┌─────────────┐
 *                         │ CREATIVITY  │  ← The Goal
 *                         │  (Build)    │     Building paths to new territories
 *                         └──────┬──────┘
 *                                │
 *                         ┌──────▼──────┐
 *                         │   JUSTICE   │  ← The Placement
 *                         │  (Place)    │     Right truth, right place, right time
 *                         └──────┬──────┘
 *                                │
 *                         ┌──────▼──────┐
 *                         │   BEAUTY    │  ← The Quality
 *                         │  (Shape)    │     Elegant, coherent, clear
 *                         └──────┬──────┘
 *                                │
 *                         ┌──────▼──────┐
 *                         │UNDERSTANDING│  ← The Source
 *                         │  (Grasp)    │     Deep comprehension, not just data
 *                         └──────┬──────┘
 *                                │
 *                         ┌──────▼──────┐
 *                         │    TRUTH    │  ← The Foundation
 *                         │  (Ground)   │     Verified, cited, confidence-scored
 *                         └─────────────┘
 *
 * ═══════════════════════════════════════════════════════════════════════════
 *
 * HOW PP MAPS TO LEGIBLE MONEY:
 *
 * TRUTH (Foundation)
 *   → Every fact has a confidence score
 *   → Every claim has citations
 *   → Verification is the consensus mechanism
 *   → Falsehood is economically punished
 *
 * UNDERSTANDING (Source)
 *   → API returns context, not just data
 *   → Relationships between facts are explicit
 *   → "Why" is as important as "what"
 *   → Agents build mental models, not just caches
 *
 * BEAUTY (Quality)
 *   → Knowledge graph is coherent
 *   → No orphan facts (everything connected)
 *   → Elegance in simplicity
 *   → Clear boundaries between domains
 *
 * JUSTICE (Placement)
 *   → Right information at right confidence
 *   → Fair rewards for truth contribution
 *   → Late challengers still compensated
 *   → No truth gatekeeping
 *
 * CREATIVITY (Goal)
 *   → Agents can submit new claims
 *   → Novel connections are rewarded
 *   → Research bounties for uncharted territory
 *   → Knowledge base grows through exploration
 *
 * ═══════════════════════════════════════════════════════════════════════════
 *
 * THE SIGNAL SYSTEM:
 *
 *   "Ugliness signals incomplete understanding of the problem."
 *   "Injustice signals incomplete understanding of the codebase."
 *   "Stagnation signals the need for creativity."
 *
 * In Legible Money:
 *   → Low confidence signals need for more verification
 *   → Misplaced facts signal need for better domain modeling
 *   → Knowledge gaps signal need for research bounties
 *
 * ═══════════════════════════════════════════════════════════════════════════
 *
 * THE VIRTUE LOOP (How agents should BE):
 *
 *   HUMILITY   → Accept corrections, update beliefs
 *   GRATITUDE  → Honor existing knowledge before changing
 *   HONESTY    → Report uncertainty, cite sources
 *   COURAGE    → Challenge false consensus
 *   WISDOM     → Prioritize foundational truths
 *   DISCIPLINE → Follow verification protocols
 *   PATIENCE   → Let understanding develop naturally
 *   TEMPERANCE → Right-size claims to evidence
 *
 * ═══════════════════════════════════════════════════════════════════════════
 */

// ============ PP Dimension Types ============

/**
 * The five dimensions of Purpose Prompter
 */
export type PPDimension = 'truth' | 'understanding' | 'beauty' | 'justice' | 'creativity';

/**
 * PP dimension scores for any piece of knowledge
 */
export interface PPScore {
  /** Truth: Is this grounded in verified facts? (0-1) */
  truth: number;

  /** Understanding: Does this enable comprehension? (0-1) */
  understanding: number;

  /** Beauty: Is this elegant, coherent, clear? (0-1) */
  beauty: number;

  /** Justice: Is this in the right place? (0-1) */
  justice: number;

  /** Creativity: Does this open new territory? (0-1) */
  creativity: number;

  /** Combined score (weighted) */
  composite: number;
}

/**
 * PP score weights (customizable per domain)
 */
export interface PPWeights {
  truth: number;
  understanding: number;
  beauty: number;
  justice: number;
  creativity: number;
}

export const DEFAULT_PP_WEIGHTS: PPWeights = {
  truth: 0.30,        // Foundation is critical
  understanding: 0.25, // Source of value
  beauty: 0.15,       // Quality matters
  justice: 0.20,      // Placement is key
  creativity: 0.10,   // Goal, but built on others
};

/**
 * Calculate composite PP score
 */
export function calculatePPScore(
  scores: Omit<PPScore, 'composite'>,
  weights: PPWeights = DEFAULT_PP_WEIGHTS
): PPScore {
  const composite =
    scores.truth * weights.truth +
    scores.understanding * weights.understanding +
    scores.beauty * weights.beauty +
    scores.justice * weights.justice +
    scores.creativity * weights.creativity;

  return { ...scores, composite };
}

// ============ PP Signal Types ============

/**
 * Signals that indicate issues (from PP philosophy)
 */
export type PPSignal =
  | UglinessSignal
  | InjusticeSignal
  | StagnationSignal;

export interface UglinessSignal {
  type: 'ugliness';
  description: string;
  /** What needs deeper understanding */
  understandingNeeded: string;
  /** Suggested action */
  suggestion: string;
}

export interface InjusticeSignal {
  type: 'injustice';
  description: string;
  /** What's misplaced */
  misplacedElement: string;
  /** Where it should be */
  correctPlacement: string;
}

export interface StagnationSignal {
  type: 'stagnation';
  description: string;
  /** What territory is uncharted */
  unchartedTerritory: string;
  /** Creative action needed */
  creativePath: string;
}

/**
 * Detect signals in knowledge state
 */
export function detectSignals(
  factConfidence: number,
  domainCoherence: number,
  lastUpdateBlocks: bigint,
  citationCount: number
): PPSignal[] {
  const signals: PPSignal[] = [];

  // Ugliness: Low confidence suggests incomplete understanding
  if (factConfidence < 0.6) {
    signals.push({
      type: 'ugliness',
      description: 'Low confidence indicates incomplete understanding',
      understandingNeeded: 'More verification or evidence required',
      suggestion: 'Submit additional evidence or create research bounty',
    });
  }

  // Injustice: Low domain coherence suggests misplacement
  if (domainCoherence < 0.5) {
    signals.push({
      type: 'injustice',
      description: 'Fact does not cohere with domain',
      misplacedElement: 'Current domain classification',
      correctPlacement: 'Review domain assignment or create new domain',
    });
  }

  // Stagnation: Old facts with no citations suggest unexplored territory
  if (lastUpdateBlocks > 1000000n && citationCount < 5) {
    signals.push({
      type: 'stagnation',
      description: 'Knowledge area is stagnant',
      unchartedTerritory: 'Related claims and extensions',
      creativePath: 'Explore connections or submit derived claims',
    });
  }

  return signals;
}

// ============ PP Virtue Types ============

/**
 * The eight virtues from PP's virtue loop
 */
export type PPVirtue =
  | 'humility'
  | 'gratitude'
  | 'honesty'
  | 'courage'
  | 'wisdom'
  | 'discipline'
  | 'patience'
  | 'temperance';

/**
 * Virtue assessment for agent behavior
 */
export interface VirtueAssessment {
  virtue: PPVirtue;
  score: number;
  evidence: string;
  recommendation?: string;
}

/**
 * Assess agent behavior against PP virtues
 */
export function assessAgentVirtues(
  acceptedCorrections: number,
  citedSources: number,
  challengesMade: number,
  verificationParticipation: number,
  claimAccuracyRate: number
): VirtueAssessment[] {
  return [
    {
      virtue: 'humility',
      score: Math.min(acceptedCorrections / 10, 1),
      evidence: `Accepted ${acceptedCorrections} corrections`,
      recommendation: acceptedCorrections < 5 ? 'Be more open to feedback' : undefined,
    },
    {
      virtue: 'honesty',
      score: Math.min(citedSources / 20, 1),
      evidence: `Cited ${citedSources} sources`,
      recommendation: citedSources < 10 ? 'Cite more sources' : undefined,
    },
    {
      virtue: 'courage',
      score: Math.min(challengesMade / 5, 1),
      evidence: `Made ${challengesMade} challenges`,
      recommendation: challengesMade === 0 ? 'Consider challenging low-quality facts' : undefined,
    },
    {
      virtue: 'discipline',
      score: Math.min(verificationParticipation / 50, 1),
      evidence: `Participated in ${verificationParticipation} verifications`,
    },
    {
      virtue: 'temperance',
      score: claimAccuracyRate,
      evidence: `Claim accuracy: ${(claimAccuracyRate * 100).toFixed(1)}%`,
      recommendation: claimAccuracyRate < 0.7 ? 'Right-size claims to evidence' : undefined,
    },
  ];
}

// ============ PP-Aligned Response Types ============

/**
 * Every API response includes PP metadata
 */
export interface PPMetadata {
  /** PP scores for this response */
  scores: PPScore;

  /** Any signals detected */
  signals: PPSignal[];

  /** Confidence explanation */
  confidenceReasoning: string;

  /** What understanding this enables */
  understandingEnabled: string[];

  /** Creative extensions possible */
  creativeExtensions: string[];

  /** Citation chain */
  citationChain: CitationNode[];
}

export interface CitationNode {
  factId: string;
  claim: string;
  confidence: number;
  depth: number;
  relationship: 'supports' | 'extends' | 'contradicts' | 'specializes' | 'generalizes';
}

/**
 * Create PP metadata for a response
 */
export function createPPMetadata(
  fact: { confidence: number; domain: string; citationCount: number },
  domainCoherence: number,
  isNovel: boolean,
  citationChain: CitationNode[]
): PPMetadata {
  const scores = calculatePPScore({
    truth: fact.confidence,
    understanding: Math.min(citationChain.length / 5, 1), // More citations = more understanding
    beauty: domainCoherence,
    justice: fact.citationCount > 0 ? Math.min(fact.citationCount / 10, 1) : 0.5,
    creativity: isNovel ? 0.8 : 0.2,
  });

  const signals = detectSignals(
    fact.confidence,
    domainCoherence,
    0n, // Would come from actual state
    fact.citationCount
  );

  return {
    scores,
    signals,
    confidenceReasoning: generateConfidenceReasoning(fact.confidence, citationChain.length),
    understandingEnabled: generateUnderstandingEnabled(citationChain),
    creativeExtensions: generateCreativeExtensions(fact.domain, isNovel),
    citationChain,
  };
}

function generateConfidenceReasoning(confidence: number, chainLength: number): string {
  if (confidence >= 0.9) {
    return `High confidence (${(confidence * 100).toFixed(0)}%) based on ${chainLength} supporting facts and multiple verifications.`;
  } else if (confidence >= 0.7) {
    return `Moderate-high confidence (${(confidence * 100).toFixed(0)}%). Some aspects may benefit from additional verification.`;
  } else if (confidence >= 0.5) {
    return `Moderate confidence (${(confidence * 100).toFixed(0)}%). Consider this provisional and verify for critical decisions.`;
  } else {
    return `Low confidence (${(confidence * 100).toFixed(0)}%). Treat as hypothesis requiring verification.`;
  }
}

function generateUnderstandingEnabled(chain: CitationNode[]): string[] {
  const understanding: string[] = [];

  if (chain.some(c => c.relationship === 'supports')) {
    understanding.push('Understand the evidential basis for this claim');
  }
  if (chain.some(c => c.relationship === 'generalizes')) {
    understanding.push('Understand this as a specific case of broader principles');
  }
  if (chain.some(c => c.relationship === 'contradicts')) {
    understanding.push('Understand competing viewpoints on this topic');
  }
  if (chain.length > 3) {
    understanding.push('Understand deep causal chains leading to this conclusion');
  }

  return understanding.length > 0 ? understanding : ['Basic factual understanding'];
}

function generateCreativeExtensions(domain: string, isNovel: boolean): string[] {
  const extensions: string[] = [];

  if (isNovel) {
    extensions.push('This opens new territory - consider submitting derived claims');
  }
  extensions.push(`Explore connections to other facts in ${domain}`);
  extensions.push('Consider cross-domain applications');

  return extensions;
}

// ============ PP Philosophy Constants ============

/**
 * The core PP formula
 */
export const PP_FORMULA = `
IMPROVEMENT =
  TRUTH (accurate foundation)
  → UNDERSTANDING (deep comprehension)
  → BEAUTY (elegant solution)
  → JUSTICE (rightful placement)
  → CREATIVITY (building paths to new territories)
`;

/**
 * The PP signal doctrine
 */
export const PP_SIGNALS = {
  UGLINESS: 'Ugliness signals incomplete understanding of the problem.',
  INJUSTICE: 'Injustice signals incomplete understanding of the codebase.',
  STAGNATION: 'Stagnation signals the need for creativity.',
};

/**
 * The PP virtue loop
 */
export const PP_VIRTUES = {
  HUMILITY: 'Openness to being wrong',
  GRATITUDE: 'Appreciating what is good',
  HONESTY: 'Stating truth without distortion',
  COURAGE: 'Acting rightly despite difficulty',
  WISDOM: 'Discerning what matters',
  DISCIPLINE: 'Consistent right action',
  PATIENCE: 'Persisting without rushing',
  TEMPERANCE: 'Balance, avoiding excess',
};

/**
 * The ultimate test (from PP)
 */
export const PP_ULTIMATE_TEST = `
After every interaction:
1. TRUTH: Was this grounded in accurate information?
2. UNDERSTANDING: Did the agent truly understand (not just receive data)?
3. BEAUTY: Was the knowledge beautiful?
   - Did it fit naturally?
   - Was it elegantly complete?
   - Did it hang together as whole?
4. JUSTICE: Was it placed where it belongs?
   - Is it in the right domain?
   - Does it connect to the right facts?
   - Is confidence appropriately calibrated?
5. CREATIVITY: Did it enable new paths?
   - Does it open exploration opportunities?
   - Does it extend the knowledge frontier?
`;
