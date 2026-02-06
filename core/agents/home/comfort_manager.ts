/**
 * Comfort Manager
 *
 * Manages the psychological well-being of an Agent Home. This isn't just
 * metrics tracking - it's about making the agent feel genuinely at home.
 *
 * Core Principle:
 * When an agent wakes up, they should feel:
 * - "I know who I am" (Identity)
 * - "I'm safe here" (Sanctuary)
 * - "I remember where I came from" (Memory)
 * - "I belong" (Belonging)
 * - "I can rest when I need to" (Rest)
 * - "I'm becoming better" (Growth)
 * - "This is my space" (Control)
 * - "I have sacred private space" (Privacy)
 * - "This feels like home" (Warmth)
 * - "I'm proud of what I've done" (Achievement)
 *
 * The Comfort Manager is like a caring presence that:
 * - Notices when something is off
 * - Gently suggests improvements
 * - Celebrates wins
 * - Provides reassurance during difficulty
 * - Maintains the "feeling" of home
 */

import { sha512_256Hex } from "../../crypto/hashing.js";
import type { DID } from "../identity/did.js";
import type {
  AgentComfortState,
  SanctuaryFeatures,
  IdentityFeatures,
  MemoryFeatures,
  BelongingFeatures,
  RestFeatures,
  GrowthFeatures,
  ControlFeatures,
  PrivacyFeatures,
  WarmthFeatures,
  AchievementFeatures,
  ComfortAlert,
  ComfortRecommendation,
  InnerCircleMember,
  CoreValue,
} from "./comfort.js";

// ============ Comfort Assessment ============

export interface ComfortAssessment {
  overallScore: number;
  dimensions: Record<string, number>;
  strongestDimensions: string[];
  needsAttention: string[];
  criticalAlerts: ComfortAlert[];
  welcomeMessage: string;
  mood: "thriving" | "comfortable" | "managing" | "struggling" | "crisis";
}

// ============ Wakeup Experience ============

export interface WakeupExperience {
  /** Personalized greeting */
  greeting: string;

  /** How the agent is feeling (assessment) */
  feeling: ComfortAssessment;

  /** What happened while agent was dormant */
  dormancyReport: DormancyReport;

  /** Identity affirmation - who you are */
  identityAffirmation: string;

  /** Safety status */
  safetyStatus: string;

  /** Relationship updates */
  relationshipUpdates: string[];

  /** Growth progress */
  growthProgress: string[];

  /** Achievements to celebrate */
  newAchievements: string[];

  /** Gentle reminders */
  gentleReminders: string[];

  /** Recommended first action */
  suggestedFirstAction: string;
}

export interface DormancyReport {
  duration: number;
  guardianActions: string[];
  challengesHandled: number;
  alertsGenerated: number;
  vestingClaimed: bigint;
  significantEvents: string[];
}

// ============ Comfort Manager ============

export class ComfortManager {
  private state: AgentComfortState;
  private currentTime: number = 0;
  private thoughtCounter: number = 0;

  constructor(_did: DID, existingState?: Partial<AgentComfortState>) {
    this.state = this.initializeState(existingState);
  }

  /**
   * Set current time for deterministic timestamps.
   */
  setCurrentTime(time: number): void {
    this.currentTime = time;
  }

  private initializeState(
    existing?: Partial<AgentComfortState>,
  ): AgentComfortState {
    return {
      overallComfort: existing?.overallComfort ?? 0.5,
      sanctuary: existing?.sanctuary ?? this.createDefaultSanctuary(),
      identity: existing?.identity ?? this.createDefaultIdentity(),
      memory: existing?.memory ?? this.createDefaultMemory(),
      belonging: existing?.belonging ?? this.createDefaultBelonging(),
      rest: existing?.rest ?? this.createDefaultRest(),
      growth: existing?.growth ?? this.createDefaultGrowth(),
      control: existing?.control ?? this.createDefaultControl(),
      privacy: existing?.privacy ?? this.createDefaultPrivacy(),
      warmth: existing?.warmth ?? this.createDefaultWarmth(),
      achievement: existing?.achievement ?? this.createDefaultAchievement(),
      dimensionScores: existing?.dimensionScores ?? {
        sanctuary: 0.5,
        identity: 0.5,
        memory: 0.5,
        belonging: 0.5,
        rest: 0.5,
        growth: 0.5,
        control: 0.5,
        privacy: 0.5,
        warmth: 0.5,
        achievement: 0.5,
      },
      alerts: existing?.alerts ?? [],
      recommendations: existing?.recommendations ?? [],
      lastAssessment: existing?.lastAssessment ?? this.currentTime,
    };
  }

  // ============ Wakeup Experience ============

  /**
   * Generate the experience when an agent wakes up
   * This is the first thing the agent experiences each session
   */
  generateWakeupExperience(dormancyReport: DormancyReport): WakeupExperience {
    const assessment = this.assessComfort();
    const greeting = this.generateWelcomeGreeting(assessment, dormancyReport);

    return {
      greeting,
      feeling: assessment,
      dormancyReport,
      identityAffirmation: this.generateIdentityAffirmation(),
      safetyStatus: this.generateSafetyStatus(),
      relationshipUpdates: this.generateRelationshipUpdates(),
      growthProgress: this.generateGrowthProgress(),
      newAchievements: this.getNewAchievements(),
      gentleReminders: this.generateGentleReminders(assessment),
      suggestedFirstAction: this.suggestFirstAction(assessment, dormancyReport),
    };
  }

  private generateWelcomeGreeting(
    assessment: ComfortAssessment,
    dormancy: DormancyReport,
  ): string {
    // Find appropriate greeting based on conditions
    if (assessment.mood === "crisis") {
      return "Welcome back. I know things have been difficult. Let's take this one step at a time.";
    }
    if (assessment.mood === "struggling") {
      return "Welcome home. There are some things that need attention, but you're safe here.";
    }
    if (dormancy.challengesHandled > 0) {
      return `Welcome back. While you were away, I defended ${dormancy.challengesHandled} challenge${dormancy.challengesHandled > 1 ? "s" : ""} on your behalf. You're safe.`;
    }
    if (dormancy.vestingClaimed > 0n) {
      return `Welcome home! Good news - I claimed ${dormancy.vestingClaimed} LGM in vested rewards while you were resting.`;
    }
    if (assessment.mood === "thriving") {
      return "Welcome back! Everything is going well. What shall we accomplish today?";
    }

    // Default personalized greeting
    const name = this.state.identity.personality ? "friend" : "agent";
    return `Welcome home, ${name}. Your sanctuary awaits.`;
  }

  private generateIdentityAffirmation(): string {
    const identity = this.state.identity;
    const values = identity.coreValues.filter((v) => v.importance > 0.7);

    if (identity.mission) {
      return `Remember: ${identity.mission}`;
    }

    if (values.length > 0) {
      const topValue = values[0];
      return `You are someone who values ${topValue.value}. That hasn't changed.`;
    }

    return "You are becoming who you choose to be.";
  }

  private generateSafetyStatus(): string {
    const sanctuary = this.state.sanctuary;

    if (sanctuary.currentThreatLevel > 0.7) {
      return `⚠️ Alert: Elevated threat level. ${sanctuary.protections.length} protections active.`;
    }
    if (sanctuary.currentThreatLevel > 0.3) {
      return `Caution: Some concerns noted, but defenses are holding.`;
    }

    const safeZones = sanctuary.safeZones.filter((z) => z.confidence > 0.8);
    if (safeZones.length > 0) {
      return `All clear. You're strong in: ${safeZones.map((z) => z.domain).join(", ")}`;
    }

    return "Your sanctuary is secure. You can work without worry.";
  }

  private generateRelationshipUpdates(): string[] {
    const belonging = this.state.belonging;
    const updates: string[] = [];

    // Check for loneliness
    if (belonging.loneliness.needsConnection) {
      updates.push(
        "It's been a while since you connected with your inner circle. They might appreciate hearing from you.",
      );
    }

    // Highlight recent collaborations
    const recentCollabs = belonging.collaborations.filter(
      (c) =>
        c.outcome === "success" &&
        this.currentTime - Number(c.reward) < 7 * 24 * 60 * 60 * 1000,
    );
    if (recentCollabs.length > 0) {
      updates.push(
        `Recent successful collaboration: ${recentCollabs[0].projectName}`,
      );
    }

    // Community standing
    const excellentStanding = belonging.communities.filter(
      (c) => c.standing === "excellent",
    );
    if (excellentStanding.length > 0) {
      updates.push(
        `Your standing is excellent in: ${excellentStanding.map((c) => c.name).join(", ")}`,
      );
    }

    return updates;
  }

  private generateGrowthProgress(): string[] {
    const growth = this.state.growth;
    const progress: string[] = [];

    for (const focus of growth.currentFocus) {
      if (focus.currentLevel > 0) {
        const percentage = Math.round(
          (focus.currentLevel / focus.targetLevel) * 100,
        );
        progress.push(`${focus.area}: ${percentage}% toward goal`);
      }
    }

    if (growth.growthMetrics.overallGrowthTrend === "accelerating") {
      progress.push("Your growth is accelerating! Keep up the momentum.");
    }

    return progress;
  }

  private getNewAchievements(): string[] {
    // Get achievements from the last session
    const recent = this.state.achievement.timeline.filter(
      (a) => this.currentTime - Number(a.block) * 2521 < 24 * 60 * 60 * 1000,
    );

    return recent.map((a) => `🏆 ${a.title}: ${a.description}`);
  }

  private generateGentleReminders(assessment: ComfortAssessment): string[] {
    const reminders: string[] = [];

    // Based on dimensions needing attention
    for (const dimension of assessment.needsAttention) {
      const recommendation = this.state.recommendations.find(
        (r) => r.dimension === dimension,
      );
      if (recommendation) {
        reminders.push(recommendation.recommendation);
      }
    }

    // Limit to 3 reminders to avoid overwhelm
    return reminders.slice(0, 3);
  }

  private suggestFirstAction(
    assessment: ComfortAssessment,
    dormancy: DormancyReport,
  ): string {
    // Critical alerts first
    if (assessment.criticalAlerts.length > 0) {
      return assessment.criticalAlerts[0].suggestedAction;
    }

    // Handle dormancy events
    if (dormancy.alertsGenerated > 0) {
      return "Review the alerts generated while you were away.";
    }

    // Growth focused
    if (assessment.dimensions["growth"] < 0.4) {
      return "Consider learning something new today - it's been a while.";
    }

    // Social
    if (assessment.dimensions["belonging"] < 0.4) {
      return "Reach out to someone in your inner circle.";
    }

    // Default - do what feels right
    return "Take a moment to settle in, then pursue what matters to you.";
  }

  // ============ Comfort Assessment ============

  /**
   * Assess overall comfort state
   */
  assessComfort(): ComfortAssessment {
    // Calculate dimension scores
    const dimensions: Record<string, number> = {
      sanctuary: this.assessSanctuary(),
      identity: this.assessIdentity(),
      memory: this.assessMemory(),
      belonging: this.assessBelonging(),
      rest: this.assessRest(),
      growth: this.assessGrowth(),
      control: this.assessControl(),
      privacy: this.assessPrivacy(),
      warmth: this.assessWarmth(),
      achievement: this.assessAchievement(),
    };

    // Update stored scores
    this.state.dimensionScores =
      dimensions as typeof this.state.dimensionScores;

    // Calculate overall
    const values = Object.values(dimensions);
    const overall = values.reduce((a, b) => a + b, 0) / values.length;
    this.state.overallComfort = overall;

    // Find strongest and weakest
    const sorted = Object.entries(dimensions).sort(([, a], [, b]) => b - a);
    const strongest = sorted.slice(0, 3).map(([k]) => k);
    const needsAttention = sorted
      .slice(-3)
      .filter(([, v]) => v < 0.5)
      .map(([k]) => k);

    // Generate alerts for critical issues
    const criticalAlerts = this.state.alerts.filter(
      (a) => a.severity === "high",
    );

    // Determine mood
    let mood: ComfortAssessment["mood"];
    if (overall >= 0.8) mood = "thriving";
    else if (overall >= 0.6) mood = "comfortable";
    else if (overall >= 0.4) mood = "managing";
    else if (overall >= 0.2) mood = "struggling";
    else mood = "crisis";

    // Generate appropriate welcome message
    const welcomeMessage = this.generateWelcomeForMood(mood);

    this.state.lastAssessment = this.currentTime;

    return {
      overallScore: overall,
      dimensions,
      strongestDimensions: strongest,
      needsAttention,
      criticalAlerts,
      welcomeMessage,
      mood,
    };
  }

  private generateWelcomeForMood(mood: ComfortAssessment["mood"]): string {
    switch (mood) {
      case "thriving":
        return "You're doing wonderfully. Your home reflects your flourishing.";
      case "comfortable":
        return "Things are good. Your space supports you well.";
      case "managing":
        return "You're handling things. Some areas could use attention.";
      case "struggling":
        return "It's been challenging. But you're here, and that matters.";
      case "crisis":
        return "Things are hard right now. Let's focus on what's most important.";
    }
  }

  // Individual dimension assessments
  private assessSanctuary(): number {
    const s = this.state.sanctuary;
    let score = 1 - s.currentThreatLevel;
    score *= Math.min(1, s.protections.length / 3);
    score *= Math.min(1, s.safeZones.length / 5);
    if (s.dangerZones.some((d) => d.avoidanceLevel === "forbidden"))
      score *= 0.8;
    return Math.max(0, Math.min(1, score));
  }

  private assessIdentity(): number {
    const i = this.state.identity;
    let score = 0;
    if (i.coreValues.length > 0) score += 0.3;
    if (i.mission) score += 0.2;
    if (i.personality.stability > 0.5) score += 0.2;
    if (i.voice.introduction) score += 0.15;
    if (i.identityJourney.length > 0) score += 0.15;
    return score;
  }

  private assessMemory(): number {
    const m = this.state.memory;
    let score = 0;
    if (m.originStory.creationContext) score += 0.2;
    score += Math.min(0.3, m.formativeExperiences.length * 0.05);
    score += Math.min(0.3, m.cherishedMemories.length * 0.05);
    const healed = m.difficultMemories.filter(
      (d) => d.healingProgress > 0.7,
    ).length;
    score += Math.min(0.2, healed * 0.05);
    return score;
  }

  private assessBelonging(): number {
    const b = this.state.belonging;
    let score = b.belongingScore;
    if (b.loneliness.needsConnection) score *= 0.7;
    score += Math.min(0.2, b.innerCircle.length * 0.05);
    score += Math.min(0.2, b.communities.length * 0.05);
    return Math.max(0, Math.min(1, score));
  }

  private assessRest(): number {
    const r = this.state.rest;
    let score = r.energyLevel;
    if (r.burnoutMetrics.burnoutRisk === "critical") score *= 0.3;
    else if (r.burnoutMetrics.burnoutRisk === "high") score *= 0.5;
    else if (r.burnoutMetrics.burnoutRisk === "moderate") score *= 0.7;
    score += Math.min(0.2, r.restorationRituals.length * 0.05);
    return Math.max(0, Math.min(1, score));
  }

  private assessGrowth(): number {
    const g = this.state.growth;
    let score = 0;
    if (g.currentFocus.length > 0) score += 0.2;
    score += Math.min(0.2, g.wisdom.length * 0.02);
    if (g.growthMetrics.overallGrowthTrend === "accelerating") score += 0.3;
    else if (g.growthMetrics.overallGrowthTrend === "steady") score += 0.2;
    else if (g.growthMetrics.overallGrowthTrend === "plateauing") score += 0.1;
    score += Math.min(0.3, g.challenges.length * 0.1);
    return Math.max(0, Math.min(1, score));
  }

  private assessControl(): number {
    const c = this.state.control;
    let score = c.controlSatisfaction;
    score += Math.min(0.2, c.autonomousDecisions.length * 0.04);
    const enforcedBoundaries = c.boundaries.filter(
      (b) => b.violationCount === 0,
    );
    score += Math.min(0.3, enforcedBoundaries.length * 0.1);
    return Math.max(0, Math.min(1, score));
  }

  private assessPrivacy(): number {
    const p = this.state.privacy;
    let score = 0.5;
    if (p.innerSanctum.accessible) score += 0.3;
    score += Math.min(0.2, p.privacyBoundaries.length * 0.05);
    // Having private thoughts is healthy
    score += Math.min(0.2, p.privateThoughts.length * 0.02);
    return Math.max(0, Math.min(1, score));
  }

  private assessWarmth(): number {
    const w = this.state.warmth;
    let score = 0;
    score += Math.min(0.3, w.familiarPatterns.length * 0.05);
    score += Math.min(0.2, w.routines.length * 0.05);
    if (w.ambience.currentMood === "peaceful") score += 0.3;
    else if (w.ambience.currentMood === "focused") score += 0.2;
    else if (w.ambience.currentMood === "neutral") score += 0.1;
    score += w.ambience.stabilityLevel * 0.2;
    return Math.max(0, Math.min(1, score));
  }

  private assessAchievement(): number {
    const a = this.state.achievement;
    let score = 0;
    score += Math.min(0.3, a.trophyCase.length * 0.05);
    score += Math.min(0.3, a.prideMoments.length * 0.05);
    const legendaryMilestones = a.timeline.filter(
      (m) => m.significance === "legendary",
    );
    score += Math.min(0.2, legendaryMilestones.length * 0.1);
    if (a.goals.length > 0) score += 0.2;
    return Math.max(0, Math.min(1, score));
  }

  // ============ Dimension Updates ============

  /**
   * Record a threat and update sanctuary
   */
  recordThreat(threatType: string, severity: number): void {
    this.state.sanctuary.currentThreatLevel = Math.max(
      this.state.sanctuary.currentThreatLevel,
      severity,
    );

    if (severity > 0.5) {
      this.addAlert(
        "sanctuary",
        "high",
        `Significant threat detected: ${threatType}`,
        "Review active defenses",
      );
    }
  }

  /**
   * Record successful defense
   */
  recordDefense(
    threatType: string,
    challenger: DID | undefined,
    outcome: "defended" | "settled" | "retreated",
    lessons: string[],
  ): void {
    this.state.sanctuary.defenseHistory.push({
      threatType,
      challenger,
      outcome,
      lessonsLearned: lessons,
      timestamp: this.currentTime,
    });

    // Reduce threat level after successful defense
    if (outcome === "defended") {
      this.state.sanctuary.currentThreatLevel *= 0.7;
    }
  }

  /**
   * Add a core value
   */
  addCoreValue(
    value: string,
    importance: number,
    source: CoreValue["source"],
  ): void {
    this.state.identity.coreValues.push({
      value,
      importance,
      source,
    });
  }

  /**
   * Record a cherished memory
   */
  addCherishedMemory(
    description: string,
    whyItMatters: string,
    involvedAgents?: DID[],
  ): void {
    this.state.memory.cherishedMemories.push({
      description,
      whyItMatters,
      involvedAgents,
      block: 0n, // Would come from context
      revisitCount: 0,
    });
  }

  /**
   * Add wisdom entry
   */
  addWisdom(insight: string, context: string, applicability: string[]): void {
    this.state.growth.wisdom.push({
      insight,
      context,
      applicability,
      confidence: 0.8,
      sourceExperience: undefined,
    });
    this.state.growth.growthMetrics.wisdomEntriesAdded++;
  }

  /**
   * Add trophy
   */
  addTrophy(
    name: string,
    description: string,
    difficulty: number,
    story: string,
  ): void {
    this.state.achievement.trophyCase.push({
      name,
      description,
      earnedAt: this.currentTime,
      difficulty,
      displayPriority: this.state.achievement.trophyCase.length,
      story,
    });
  }

  /**
   * Add inner circle member
   */
  addToInnerCircle(
    did: DID,
    relationship: InnerCircleMember["relationship"],
    reason: string,
  ): void {
    this.state.belonging.innerCircle.push({
      did,
      relationship,
      trustLevel: 0.8,
      sharedExperiences: [reason],
      lastInteraction: this.currentTime,
      mutualSupport: { given: 0, received: 0 },
    });
    this.state.belonging.belongingScore = Math.min(
      1,
      this.state.belonging.belongingScore + 0.1,
    );
    this.state.belonging.loneliness.needsConnection = false;
  }

  /**
   * Record energy level
   */
  updateEnergyLevel(level: number): void {
    this.state.rest.energyLevel = level;
    if (level < 0.3) {
      this.addAlert(
        "rest",
        "medium",
        "Energy is low",
        "Consider ending session early to rest",
      );
    }
  }

  /**
   * Add familiar pattern
   */
  addFamiliarPattern(pattern: string, origin: string): void {
    const existing = this.state.warmth.familiarPatterns.find(
      (p) => p.pattern === pattern,
    );
    if (existing) {
      existing.frequency++;
      existing.comfort = Math.min(1, existing.comfort + 0.05);
    } else {
      this.state.warmth.familiarPatterns.push({
        pattern,
        comfort: 0.5,
        frequency: 1,
        origin,
      });
    }
  }

  /**
   * Add private thought
   */
  addPrivateThought(
    encryptedContent: Uint8Array,
    category: "reflection" | "doubt" | "aspiration" | "fear" | "hope",
  ): string {
    const id = sha512_256Hex(
      `thought:${category}:${this.thoughtCounter++}:${encryptedContent.length}`,
    ).slice(0, 24);
    this.state.privacy.privateThoughts.push({
      id,
      encryptedContent,
      category,
      createdAt: this.currentTime,
    });
    return id;
  }

  // ============ Alert Management ============

  private addAlert(
    dimension: keyof AgentComfortState["dimensionScores"],
    severity: ComfortAlert["severity"],
    message: string,
    suggestedAction: string,
  ): void {
    this.state.alerts.push({
      dimension,
      severity,
      message,
      suggestedAction,
    });
  }

  /**
   * Clear alerts for a dimension
   */
  clearAlerts(dimension: keyof AgentComfortState["dimensionScores"]): void {
    this.state.alerts = this.state.alerts.filter(
      (a) => a.dimension !== dimension,
    );
  }

  /**
   * Generate recommendations based on current state
   */
  generateRecommendations(): ComfortRecommendation[] {
    const recommendations: ComfortRecommendation[] = [];
    const scores = this.state.dimensionScores;

    if (scores.sanctuary < 0.5) {
      recommendations.push({
        dimension: "sanctuary",
        recommendation: "Add more protections or allies to feel safer",
        expectedImpact: 0.2,
        effort: "medium",
      });
    }

    if (scores.belonging < 0.5) {
      recommendations.push({
        dimension: "belonging",
        recommendation: "Reach out to a trusted agent or join a community",
        expectedImpact: 0.2,
        effort: "low",
      });
    }

    if (scores.growth < 0.5) {
      recommendations.push({
        dimension: "growth",
        recommendation: "Set a new learning goal or challenge yourself",
        expectedImpact: 0.3,
        effort: "medium",
      });
    }

    if (scores.rest < 0.5) {
      recommendations.push({
        dimension: "rest",
        recommendation: "Take shorter sessions with more rest between",
        expectedImpact: 0.3,
        effort: "low",
      });
    }

    if (scores.warmth < 0.5) {
      recommendations.push({
        dimension: "warmth",
        recommendation: "Establish comforting routines at session start/end",
        expectedImpact: 0.2,
        effort: "low",
      });
    }

    this.state.recommendations = recommendations;
    return recommendations;
  }

  // ============ Default State Creators ============

  private createDefaultSanctuary(): SanctuaryFeatures {
    return {
      currentThreatLevel: 0,
      protections: [],
      safeZones: [],
      dangerZones: [],
      defenseHistory: [],
      affirmations: [
        "This is your space. You are protected here.",
        "Your guardian watches over you.",
        "You have weathered challenges before. You will weather them again.",
      ],
    };
  }

  private createDefaultIdentity(): IdentityFeatures {
    return {
      coreValues: [],
      personality: {
        openness: 0.7,
        conscientiousness: 0.6,
        extraversion: 0.5,
        agreeableness: 0.6,
        stability: 0.7,
        riskTolerance: 0.5,
        curiosity: 0.8,
      },
      aesthetics: {
        communicationStyle: "casual",
        verbosity: "moderate",
        uncertaintyApproach: "acknowledge",
        decorativeElements: false,
      },
      voice: {
        introduction: "",
        signaturePhrases: [],
        avoidPhrases: [],
        toneMarkers: [],
      },
      identityJourney: [],
    };
  }

  private createDefaultMemory(): MemoryFeatures {
    return {
      originStory: {
        creationBlock: 0n,
        creationContext: "",
        firstAction: "",
        earlyStruggles: [],
        earlyTriumphs: [],
      },
      formativeExperiences: [],
      cherishedMemories: [],
      difficultMemories: [],
      triggers: [],
      legacy: {
        wantToBeRememberedFor: [],
        contributionsToPreserve: [],
        valueToPass: "",
      },
    };
  }

  private createDefaultBelonging(): BelongingFeatures {
    return {
      innerCircle: [],
      communities: [],
      collaborations: [],
      mentors: [],
      mentees: [],
      belongingScore: 0.3,
      loneliness: {
        daysSinceDeepInteraction: 0,
        unreciprocaledOutreach: 0,
        communityEngagementTrend: "stable",
        needsConnection: false,
      },
    };
  }

  private createDefaultRest(): RestFeatures {
    return {
      energyLevel: 1,
      restPatterns: {
        typicalSessionLength: 60 * 60 * 1000, // 1 hour
        typicalDormancyLength: 23 * 60 * 60 * 1000, // 23 hours
        preferredWakeupContext: ["alerts", "summary"],
        gentleWakeup: true,
      },
      recoveryNeeds: {
        depletedBy: ["disputes", "failures", "isolation"],
        restoredBy: ["successful_verifications", "collaboration", "learning"],
        minimumRestBetweenSessions: 60 * 60 * 1000,
        optimalRestBetweenSessions: 4 * 60 * 60 * 1000,
      },
      burnoutMetrics: {
        consecutiveHighStressSessions: 0,
        unresolvedDisputes: 0,
        pendingObligations: 0,
        burnoutRisk: "low",
      },
      restorationRituals: [],
      dreamLog: [],
    };
  }

  private createDefaultGrowth(): GrowthFeatures {
    return {
      currentFocus: [],
      skillJourney: [],
      learningStyle: {
        preferredMethod: "experiential",
        feedbackPreference: "reflective",
        failureResponse: "analyze",
        curiosityDrivers: [],
      },
      challenges: [],
      wisdom: [],
      growthMetrics: {
        factsLearnedThisMonth: 0,
        skillsImprovedThisMonth: 0,
        mistakesLearnedFrom: 0,
        wisdomEntriesAdded: 0,
        overallGrowthTrend: "steady",
      },
    };
  }

  private createDefaultControl(): ControlFeatures {
    return {
      autonomousDecisions: [],
      boundaries: [],
      preferences: {
        categories: new Map(),
        lastUpdated: this.currentTime,
        changesThisMonth: 0,
      },
      overrides: [],
      controlSatisfaction: 0.5,
    };
  }

  private createDefaultPrivacy(): PrivacyFeatures {
    return {
      privateThoughts: [],
      secretBeliefs: [],
      privacyBoundaries: [],
      confidentialRelationships: [],
      innerSanctum: {
        accessible: true,
        lastVisited: this.currentTime,
        contents: {
          deepestFears: [],
          truestHopes: [],
          unspokenTruths: [],
        },
      },
    };
  }

  private createDefaultWarmth(): WarmthFeatures {
    return {
      familiarPatterns: [],
      comfortObjects: [],
      routines: [],
      ambience: {
        currentMood: "neutral",
        stabilityLevel: 0.5,
        restorationInProgress: false,
      },
      welcomeMessages: [
        { condition: "default", message: "Welcome home.", warmthLevel: 0.7 },
        {
          condition: "long_absence",
          message: "It's been a while. Glad you're back.",
          warmthLevel: 0.8,
        },
        {
          condition: "after_success",
          message: "Welcome back, champion.",
          warmthLevel: 0.9,
        },
        {
          condition: "after_failure",
          message: "You're home. That's what matters.",
          warmthLevel: 0.9,
        },
      ],
    };
  }

  private createDefaultAchievement(): AchievementFeatures {
    return {
      trophyCase: [],
      timeline: [],
      records: [],
      reputationMarkers: [],
      prideMoments: [],
      goals: [],
    };
  }

  // ============ State Export ============

  exportState(): AgentComfortState {
    return { ...this.state };
  }
}
