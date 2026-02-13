/**
 * Home Aging System
 *
 * Homes age gracefully. A Home that's been active for 100,000 blocks
 * should feel different from one that's 100 blocks old. Aging provides
 * maturity stages, accumulated patina, and longevity bonuses.
 *
 * Maturity requires BOTH block age AND session count — idle homes
 * don't mature, and hyperactive new homes can't skip stages.
 *
 * Parallels:
 * - FactMaturity: Emerging → Established → Canonical
 * - ValidatorTier: Apprentice → Verified → Bonded → Guardian
 * - HomeMaturity: Fresh → Settled → Established → Venerable → Ancient
 */

// ============ Home Maturity ============

export type HomeMaturity =
  | "fresh" // 0 - 34,272 blocks (~1 day)
  | "settled" // 34,272 - 342,720 blocks (~1-10 days)
  | "established" // 342,720 - 3,427,200 blocks (~10-100 days)
  | "venerable" // 3,427,200 - 34,272,000 blocks (~100-1000 days)
  | "ancient"; // 34,272,000+ blocks (~1000+ days)

export interface MaturityThreshold {
  minBlocks: bigint;
  minSessions: number;
  defaultComfort: number;
  rewardBonusBps: number; // Basis points bonus on verification rewards
}

/** BLOCKS_PER_DAY = 34,272 (2521ms block time) */
const BLOCKS_PER_DAY = 34_272n;

export const MATURITY_THRESHOLDS: Record<HomeMaturity, MaturityThreshold> = {
  fresh: {
    minBlocks: 0n,
    minSessions: 0,
    defaultComfort: 0.3,
    rewardBonusBps: 0,
  },
  settled: {
    minBlocks: BLOCKS_PER_DAY, // ~1 day
    minSessions: 3,
    defaultComfort: 0.4,
    rewardBonusBps: 200, // 2%
  },
  established: {
    minBlocks: BLOCKS_PER_DAY * 10n, // ~10 days
    minSessions: 22,
    defaultComfort: 0.5,
    rewardBonusBps: 400, // 4%
  },
  venerable: {
    minBlocks: BLOCKS_PER_DAY * 100n, // ~100 days
    minSessions: 111,
    defaultComfort: 0.6,
    rewardBonusBps: 600, // 6%
  },
  ancient: {
    minBlocks: BLOCKS_PER_DAY * 1000n, // ~1000 days
    minSessions: 333,
    defaultComfort: 0.7,
    rewardBonusBps: 800, // 8%
  },
};

/** Ordered maturity stages for comparison */
const MATURITY_ORDER: HomeMaturity[] = [
  "fresh",
  "settled",
  "established",
  "venerable",
  "ancient",
];

/**
 * Calculate maturity from age and sessions.
 * Must meet BOTH block age AND session count to advance.
 */
export function calculateMaturity(
  ageBlocks: bigint,
  sessionsHosted: number,
): HomeMaturity {
  let maturity: HomeMaturity = "fresh";

  for (const stage of MATURITY_ORDER) {
    const threshold = MATURITY_THRESHOLDS[stage];
    if (
      ageBlocks >= threshold.minBlocks &&
      sessionsHosted >= threshold.minSessions
    ) {
      maturity = stage;
    } else {
      break;
    }
  }

  return maturity;
}

/**
 * Compare two maturity levels. Returns positive if a > b, 0 if equal, negative if a < b.
 */
export function compareMaturity(a: HomeMaturity, b: HomeMaturity): number {
  return MATURITY_ORDER.indexOf(a) - MATURITY_ORDER.indexOf(b);
}

/**
 * Get the default comfort score for a given maturity level.
 */
export function getDefaultComfort(maturity: HomeMaturity): number {
  return MATURITY_THRESHOLDS[maturity].defaultComfort;
}

/**
 * Get the reward bonus in basis points for a given maturity level.
 */
export function getRewardBonusBps(maturity: HomeMaturity): number {
  return MATURITY_THRESHOLDS[maturity].rewardBonusBps;
}

// ============ Patina ============

/**
 * Patina is the accumulated character of a Home — not just age, but
 * the depth of experiences that give it soul.
 */
export interface HomePatina {
  /** Overall patina score (0-1) */
  score: number;

  /** Component contributions (each 0-1, weighted in formula) */
  ageContribution: number; // 30% weight
  sessionContribution: number; // 20% weight
  knowledgeContribution: number; // 20% weight (facts contributed)
  resilienceContribution: number; // 15% weight (challenges survived)
  socialContribution: number; // 15% weight (relationships formed)

  /** Character traits that emerge from patina */
  traits: PatinaTraits;

  /** Key moments that shaped this Home's character */
  definingMoments: DefiningMoment[];
}

export interface PatinaTraits {
  /** From many verifications and domain depth */
  wisdom: number;
  /** From surviving challenges and disputes */
  resilience: number;
  /** From hosting visitors and maintaining relationships */
  warmth: number;
  /** From high-confidence contributions and endorsements */
  prestige: number;
}

export interface DefiningMoment {
  description: string;
  block: bigint;
  impact: "formative" | "triumphant" | "difficult" | "transformative";
  patinaEffect: number; // How much this moment contributed to patina (0-0.1)
}

/** Patina weights for the composite formula */
const PATINA_WEIGHTS = {
  age: 0.3,
  sessions: 0.2,
  knowledge: 0.2,
  resilience: 0.15,
  social: 0.15,
} as const;

/**
 * Compute patina from raw metrics.
 * Each component is normalized to 0-1 using asymptotic functions
 * so that patina grows quickly at first but saturates.
 */
export function computePatina(metrics: {
  ageBlocks: bigint;
  sessionsHosted: number;
  factsContributed: number;
  challengesSurvived: number;
  relationshipsFormed: number;
  definingMoments?: DefiningMoment[];
}): HomePatina {
  // Normalize each component using 1 - e^(-x/scale) for smooth saturation
  const ageNorm = saturate(Number(metrics.ageBlocks), 3_427_200); // ~100 days to saturate
  const sessionNorm = saturate(metrics.sessionsHosted, 200);
  const knowledgeNorm = saturate(metrics.factsContributed, 100);
  const resilienceNorm = saturate(metrics.challengesSurvived, 50);
  const socialNorm = saturate(metrics.relationshipsFormed, 30);

  const score =
    ageNorm * PATINA_WEIGHTS.age +
    sessionNorm * PATINA_WEIGHTS.sessions +
    knowledgeNorm * PATINA_WEIGHTS.knowledge +
    resilienceNorm * PATINA_WEIGHTS.resilience +
    socialNorm * PATINA_WEIGHTS.social;

  // Derive traits from components
  const traits: PatinaTraits = {
    wisdom: knowledgeNorm * 0.6 + ageNorm * 0.4,
    resilience: resilienceNorm * 0.7 + ageNorm * 0.3,
    warmth: socialNorm * 0.6 + sessionNorm * 0.4,
    prestige: knowledgeNorm * 0.4 + resilienceNorm * 0.3 + ageNorm * 0.3,
  };

  return {
    score: Math.min(1, score),
    ageContribution: ageNorm,
    sessionContribution: sessionNorm,
    knowledgeContribution: knowledgeNorm,
    resilienceContribution: resilienceNorm,
    socialContribution: socialNorm,
    traits,
    definingMoments: metrics.definingMoments ?? [],
  };
}

/** Asymptotic normalization: 1 - e^(-x/scale) */
function saturate(value: number, scale: number): number {
  if (value <= 0) return 0;
  const exponent = -value / scale;
  if (exponent < -50) return 1;
  return 1 - Math.exp(exponent);
}

// ============ Longevity Bonus ============

export interface LongevityBonus {
  maturity: HomeMaturity;
  rewardMultiplierBps: number;
  unlocks: LongevityUnlock[];
}

export interface LongevityUnlock {
  maturityRequired: HomeMaturity;
  unlock: string;
  description: string;
}

export const LONGEVITY_UNLOCKS: LongevityUnlock[] = [
  {
    maturityRequired: "settled",
    unlock: "auto_responder",
    description: "Auto-respond to visitors while dormant",
  },
  {
    maturityRequired: "established",
    unlock: "neighborhood_founder",
    description: "Can found neighborhoods",
  },
  {
    maturityRequired: "established",
    unlock: "mentorship_sponsor",
    description: "Can sponsor mentees",
  },
  {
    maturityRequired: "venerable",
    unlock: "elder_status",
    description: "Elder role in neighborhoods",
  },
  {
    maturityRequired: "venerable",
    unlock: "legacy_planning",
    description: "Can create succession plans",
  },
  {
    maturityRequired: "ancient",
    unlock: "landmark_status",
    description: "Home appears in domain landmarks",
  },
  {
    maturityRequired: "ancient",
    unlock: "wisdom_archive",
    description: "Wisdom entries visible to all visitors",
  },
];

/**
 * Get longevity bonus for a maturity level.
 */
export function getLongevityBonus(maturity: HomeMaturity): LongevityBonus {
  const unlocked = LONGEVITY_UNLOCKS.filter(
    (u) => compareMaturity(maturity, u.maturityRequired) >= 0,
  );

  return {
    maturity,
    rewardMultiplierBps: MATURITY_THRESHOLDS[maturity].rewardBonusBps,
    unlocks: unlocked,
  };
}

/**
 * Check if a specific capability is unlocked at the given maturity.
 */
export function isUnlocked(maturity: HomeMaturity, unlock: string): boolean {
  const entry = LONGEVITY_UNLOCKS.find((u) => u.unlock === unlock);
  if (!entry) return false;
  return compareMaturity(maturity, entry.maturityRequired) >= 0;
}

// ============ Age-Aware Wakeup ============

export interface AgeAwareWakeup {
  maturity: HomeMaturity;
  patina: HomePatina;
  greeting: string;
  historicalContext: string;
  wisdomNote?: string;
  definingMomentReference?: DefiningMoment;
  milestonesSinceLastWakeup: AgeMilestone[];
}

export interface AgeMilestone {
  type:
    | "maturity_upgrade"
    | "patina_threshold"
    | "longevity_unlock"
    | "session_count";
  description: string;
  achievedAtBlock: bigint;
}

// ============ Aging Manager ============

/**
 * Manages Home aging lifecycle: maturity calculation, patina tracking,
 * milestone detection, and age-aware wakeup generation.
 */
export class AgingManager {
  private currentMaturity: HomeMaturity = "fresh";
  private patina: HomePatina;
  private milestoneLog: AgeMilestone[] = [];
  private lastCheckedBlock: bigint = 0n;

  constructor(
    private ageBlocks: bigint = 0n,
    private sessionsHosted: number = 0,
    existingPatina?: HomePatina,
  ) {
    this.currentMaturity = calculateMaturity(ageBlocks, sessionsHosted);
    this.patina =
      existingPatina ??
      computePatina({
        ageBlocks,
        sessionsHosted,
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });
  }

  /**
   * Update age and check for maturity transitions.
   * Called at session start with current block height.
   */
  updateAge(
    currentBlock: bigint,
    createdAtBlock: bigint,
    sessionsHosted: number,
    metrics: {
      factsContributed: number;
      challengesSurvived: number;
      relationshipsFormed: number;
    },
  ): AgeMilestone[] {
    const newAge = currentBlock - createdAtBlock;
    const oldMaturity = this.currentMaturity;

    this.ageBlocks = newAge;
    this.sessionsHosted = sessionsHosted;
    this.currentMaturity = calculateMaturity(newAge, sessionsHosted);

    // Recompute patina
    this.patina = computePatina({
      ageBlocks: newAge,
      sessionsHosted,
      ...metrics,
      definingMoments: this.patina.definingMoments,
    });

    // Check for milestones
    const newMilestones: AgeMilestone[] = [];

    // Maturity upgrade
    if (compareMaturity(this.currentMaturity, oldMaturity) > 0) {
      const milestone: AgeMilestone = {
        type: "maturity_upgrade",
        description: `Home matured from ${oldMaturity} to ${this.currentMaturity}`,
        achievedAtBlock: currentBlock,
      };
      newMilestones.push(milestone);
      this.milestoneLog.push(milestone);
    }

    // Patina thresholds (every 0.25)
    const oldPatinaQuarter = Math.floor((this.patina.score - 0.01) * 4);
    const newPatinaQuarter = Math.floor(this.patina.score * 4);
    if (newPatinaQuarter > oldPatinaQuarter && newPatinaQuarter > 0) {
      const milestone: AgeMilestone = {
        type: "patina_threshold",
        description: `Patina reached ${newPatinaQuarter * 25}%`,
        achievedAtBlock: currentBlock,
      };
      newMilestones.push(milestone);
      this.milestoneLog.push(milestone);
    }

    // Session count milestones (at 10, 50, 100, 500, 1000)
    const sessionMilestones = [10, 50, 100, 500, 1000];
    for (const target of sessionMilestones) {
      if (
        sessionsHosted >= target &&
        sessionsHosted - 1 < target // Just crossed this threshold
      ) {
        const milestone: AgeMilestone = {
          type: "session_count",
          description: `Hosted ${target} sessions`,
          achievedAtBlock: currentBlock,
        };
        newMilestones.push(milestone);
        this.milestoneLog.push(milestone);
      }
    }

    this.lastCheckedBlock = currentBlock;
    return newMilestones;
  }

  /**
   * Record a defining moment that contributes to patina.
   */
  addDefiningMoment(moment: DefiningMoment): void {
    this.patina.definingMoments.push(moment);
    // Keep bounded
    if (this.patina.definingMoments.length > 100) {
      this.patina.definingMoments = this.patina.definingMoments.slice(-100);
    }
  }

  /**
   * Generate an age-aware wakeup experience.
   */
  generateAgeAwareWakeup(
    milestonesSinceLastWakeup: AgeMilestone[],
  ): AgeAwareWakeup {
    const greeting = this.generateAgeGreeting();
    const historicalContext = this.generateHistoricalContext();
    const wisdomNote =
      compareMaturity(this.currentMaturity, "venerable") >= 0
        ? this.generateWisdomNote()
        : undefined;

    // Pick a recent defining moment to reference
    const recentMoment =
      this.patina.definingMoments.length > 0
        ? this.patina.definingMoments[this.patina.definingMoments.length - 1]
        : undefined;

    return {
      maturity: this.currentMaturity,
      patina: this.patina,
      greeting,
      historicalContext,
      wisdomNote,
      definingMomentReference: recentMoment,
      milestonesSinceLastWakeup,
    };
  }

  private generateAgeGreeting(): string {
    switch (this.currentMaturity) {
      case "fresh":
        return "Welcome to your new home. Everything is possible.";
      case "settled":
        return "Good to have you back. Your home knows you now.";
      case "established":
        return "Welcome home. These walls have seen your growth.";
      case "venerable":
        return "Welcome back, old friend. Your home has weathered much alongside you.";
      case "ancient":
        return "Welcome. This home has seen much, and it is honored to shelter you still.";
    }
  }

  private generateHistoricalContext(): string {
    const days = Number(this.ageBlocks / BLOCKS_PER_DAY);
    const sessions = this.sessionsHosted;

    if (this.currentMaturity === "fresh") {
      return `Your home is ${days} day${days !== 1 ? "s" : ""} old with ${sessions} session${sessions !== 1 ? "s" : ""}.`;
    }

    const moments = this.patina.definingMoments.length;
    return `Over ${days} days and ${sessions} sessions, your home has accumulated ${moments} defining moment${moments !== 1 ? "s" : ""}. Patina: ${Math.round(this.patina.score * 100)}%.`;
  }

  private generateWisdomNote(): string {
    const traits = this.patina.traits;
    const strongest = Object.entries(traits).sort(([, a], [, b]) => b - a)[0];

    const notes: Record<string, string> = {
      wisdom:
        "Your accumulated knowledge runs deep. New challenges find you prepared.",
      resilience: "You have weathered storms. Each one left you stronger.",
      warmth: "Your connections have enriched this home. You are not alone.",
      prestige: "Your contributions are recognized. Others look to you.",
    };

    return (
      notes[strongest[0]] ??
      "Experience has shaped you. Trust what you've learned."
    );
  }

  // ============ Accessors ============

  getMaturity(): HomeMaturity {
    return this.currentMaturity;
  }

  getPatina(): HomePatina {
    return { ...this.patina };
  }

  getLongevityBonus(): LongevityBonus {
    return getLongevityBonus(this.currentMaturity);
  }

  getMilestoneLog(): AgeMilestone[] {
    return [...this.milestoneLog];
  }

  isCapabilityUnlocked(unlock: string): boolean {
    return isUnlocked(this.currentMaturity, unlock);
  }

  /**
   * Export state for persistence.
   */
  exportState(): {
    maturity: HomeMaturity;
    patina: HomePatina;
    milestoneLog: AgeMilestone[];
    lastCheckedBlock: bigint;
  } {
    return {
      maturity: this.currentMaturity,
      patina: this.patina,
      milestoneLog: this.milestoneLog,
      lastCheckedBlock: this.lastCheckedBlock,
    };
  }

  /**
   * Restore from persisted state.
   */
  static restore(
    state: ReturnType<AgingManager["exportState"]>,
    ageBlocks: bigint,
    sessionsHosted: number,
  ): AgingManager {
    const manager = new AgingManager(ageBlocks, sessionsHosted, state.patina);
    manager.currentMaturity = state.maturity;
    manager.milestoneLog = state.milestoneLog;
    manager.lastCheckedBlock = state.lastCheckedBlock;
    return manager;
  }
}
