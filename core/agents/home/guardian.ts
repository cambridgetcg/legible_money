/**
 * Agent Guardian
 *
 * Autonomous defense and maintenance for the Agent Home.
 * The guardian acts on behalf of the agent while they sleep:
 * - Defends against challenges automatically
 * - Responds to dispute deadlines
 * - Maintains token positions (prevents expiration)
 * - Generates alerts for critical events
 *
 * "The home acts when the agent cannot."
 *
 * PP HIERARCHY:
 *   TRUTH: Guardian actions are transparent and auditable
 *   UNDERSTANDING: Clear logic for when and why guardian acts
 *   BEAUTY: Elegant autonomous defense that serves agent's interests
 *   JUSTICE: Fair and proportionate responses to threats
 */

import { sha512_256Hex } from "../../crypto/hashing.js";
import type { DID } from "../identity/did.js";
import {
  type AgentGuardian,
  type DefenseConfig,
  type DefenseStrategy,
  type MaintenanceConfig,
  type AlertTrigger,
  type AlertType,
  type AlertDelivery,
  type DeadmanConfig,
  type DeadmanAction,
  type ActiveDefense,
  type Alert,
  type GuardianStats,
  HomeError,
} from "./types.js";

/**
 * Guardian Manager
 * Handles all autonomous defense and maintenance for the Agent Home
 */
export class GuardianManager {
  private guardian: AgentGuardian;
  private dirty: boolean = false;

  constructor(existingGuardian?: AgentGuardian) {
    this.guardian = existingGuardian ?? this.createDefaultGuardian();
  }

  // ============ Initialization ============

  private createDefaultGuardian(): AgentGuardian {
    return {
      defense: {
        autoDefendChallenges: true,
        defaultStrategy: "moderate",
        maxAutoDefenseStake: 10000n * 10n ** 18n, // 10,000 LGM max
        evidenceSources: ["memory", "on_chain", "trusted_agents"],
        defenseAllies: [],
      },
      maintenance: {
        monitorExpiration: true,
        monitorVesting: true,
        monitorReputation: true,
        checkInterval: 1000n, // Every ~42 minutes
        lastMaintenanceBlock: 0n,
      },
      alerts: {
        triggers: this.getDefaultAlertTriggers(),
        delivery: {
          storeInHome: true,
          emitEvent: true,
          notifyAllies: false,
        },
        priorities: {
          criticalThreshold: 100n * 10n ** 18n, // Alert if balance below 100 LGM
          highReputationDrop: 0.1, // Alert if reputation drops 10%
          mediumDeadlineHours: 24, // Alert 24h before deadlines
        },
      },
      recovery: {
        guardians: [],
        threshold: 2,
        recoveryDelay: 10000n, // ~7 hours
        deadman: {
          enabled: false,
          inactivityThreshold: 100000n, // ~28 days
          action: "alert_guardians",
        },
      },
      activeDefenses: [],
      pendingAlerts: [],
      stats: {
        challengesAutoDefended: 0,
        challengesWon: 0,
        challengesLost: 0,
        alertsGenerated: 0,
        maintenanceActionsPerformed: 0,
        tokensPreservedFromExpiration: 0n,
        vestingAutoClaimed: 0n,
      },
    };
  }

  private getDefaultAlertTriggers(): AlertTrigger[] {
    return [
      {
        type: "challenge_received",
        condition: {},
        priority: "critical",
        enabled: true,
      },
      {
        type: "dispute_deadline",
        condition: { timeframe: 24 * 60 * 60 * 1000 }, // 24 hours
        priority: "high",
        enabled: true,
      },
      {
        type: "vesting_available",
        condition: { threshold: 100n * 10n ** 18n }, // 100 LGM
        priority: "low",
        enabled: true,
      },
      {
        type: "tokens_expiring",
        condition: { timeframe: 7 * 24 * 60 * 60 * 1000 }, // 7 days
        priority: "medium",
        enabled: true,
      },
      {
        type: "reputation_change",
        condition: { threshold: -0.05 }, // 5% drop
        priority: "medium",
        enabled: true,
      },
      {
        type: "balance_low",
        condition: { threshold: 100n * 10n ** 18n },
        priority: "high",
        enabled: true,
      },
      {
        type: "stake_at_risk",
        condition: {},
        priority: "critical",
        enabled: true,
      },
    ];
  }

  // ============ Defense Operations ============

  /**
   * Handle an incoming challenge
   */
  handleChallenge(
    disputeId: string,
    factId: string,
    challenger: DID,
    currentBlock: bigint,
  ): {
    action: "auto_defend" | "alert_only" | "delegate";
    defense?: ActiveDefense;
  } {
    // Always create alert
    this.createAlert(
      "challenge_received",
      "critical",
      `Challenge received on fact ${factId}`,
      {
        disputeId,
        factId,
        challenger,
      },
    );

    // Check if auto-defend is enabled
    if (!this.guardian.defense.autoDefendChallenges) {
      return { action: "alert_only" };
    }

    // Check if we should delegate
    if (
      this.guardian.defense.defaultStrategy === "delegate" &&
      this.guardian.defense.defenseAllies.length > 0
    ) {
      this.createAlert(
        "ally_needs_help",
        "high",
        `Delegating defense of ${factId} to allies`,
        {
          disputeId,
          factId,
        },
      );
      return { action: "delegate" };
    }

    // Create active defense
    const defense: ActiveDefense = {
      disputeId,
      factId,
      challenger,
      strategy: this.guardian.defense.defaultStrategy,
      stake: 0n, // Will be set when actually defending
      phase: "initiation",
      nextDeadline: currentBlock + 10000n, // Placeholder
      autoResponding: true,
    };

    this.guardian.activeDefenses.push(defense);
    this.guardian.stats.challengesAutoDefended++;
    this.dirty = true;

    return { action: "auto_defend", defense };
  }

  /**
   * Update defense phase
   */
  updateDefensePhase(
    disputeId: string,
    newPhase: string,
    nextDeadline: bigint,
  ): void {
    const defense = this.guardian.activeDefenses.find(
      (d) => d.disputeId === disputeId,
    );
    if (defense) {
      defense.phase = newPhase;
      defense.nextDeadline = nextDeadline;
      this.dirty = true;

      // Alert if deadline is approaching
      this.createAlert(
        "dispute_deadline",
        "high",
        `Dispute ${disputeId} entering ${newPhase} phase`,
        {
          disputeId,
          phase: newPhase,
          deadline: nextDeadline.toString(),
        },
      );
    }
  }

  /**
   * Complete a defense (won or lost)
   */
  completeDefense(disputeId: string, won: boolean): void {
    const index = this.guardian.activeDefenses.findIndex(
      (d) => d.disputeId === disputeId,
    );
    if (index >= 0) {
      this.guardian.activeDefenses.splice(index, 1);

      if (won) {
        this.guardian.stats.challengesWon++;
      } else {
        this.guardian.stats.challengesLost++;
      }

      this.dirty = true;
    }
  }

  /**
   * Get active defenses
   */
  getActiveDefenses(): ActiveDefense[] {
    return [...this.guardian.activeDefenses];
  }

  /**
   * Get defenses with upcoming deadlines
   */
  getUrgentDefenses(
    currentBlock: bigint,
    withinBlocks: bigint,
  ): ActiveDefense[] {
    return this.guardian.activeDefenses.filter(
      (d) => d.nextDeadline <= currentBlock + withinBlocks,
    );
  }

  // ============ Defense Configuration ============

  /**
   * Set defense configuration
   */
  setDefenseConfig(config: Partial<DefenseConfig>): void {
    this.guardian.defense = {
      ...this.guardian.defense,
      ...config,
    };
    this.dirty = true;
  }

  /**
   * Add a defense ally
   */
  addDefenseAlly(did: DID): void {
    if (!this.guardian.defense.defenseAllies.includes(did)) {
      this.guardian.defense.defenseAllies.push(did);
      this.dirty = true;
    }
  }

  /**
   * Remove a defense ally
   */
  removeDefenseAlly(did: DID): void {
    const index = this.guardian.defense.defenseAllies.indexOf(did);
    if (index >= 0) {
      this.guardian.defense.defenseAllies.splice(index, 1);
      this.dirty = true;
    }
  }

  /**
   * Get defense strategy for a situation
   */
  getDefenseStrategy(): DefenseStrategy {
    return this.guardian.defense.defaultStrategy;
  }

  // ============ Maintenance Operations ============

  /**
   * Perform maintenance check
   */
  performMaintenance(
    currentBlock: bigint,
    context: MaintenanceContext,
  ): MaintenanceResult {
    const actions: MaintenanceAction[] = [];

    // Check if it's time for maintenance
    if (
      currentBlock <
      this.guardian.maintenance.lastMaintenanceBlock +
        this.guardian.maintenance.checkInterval
    ) {
      return { actions, performed: false };
    }

    // Check vesting
    if (
      this.guardian.maintenance.monitorVesting &&
      context.claimableVesting > 0n
    ) {
      actions.push({
        type: "claim_vesting",
        amount: context.claimableVesting,
        reason: "Claimable rewards available",
      });
      this.guardian.stats.vestingAutoClaimed += context.claimableVesting;
    }

    // Check expiration
    if (
      this.guardian.maintenance.monitorExpiration &&
      context.expiringAmount > 0n
    ) {
      actions.push({
        type: "extend_expiration",
        amount: context.expiringAmount,
        reason: "Tokens expiring soon",
      });
      this.guardian.stats.tokensPreservedFromExpiration +=
        context.expiringAmount;
    }

    // Check reputation
    if (
      this.guardian.maintenance.monitorReputation &&
      context.reputationChange <
        -this.guardian.alerts.priorities.highReputationDrop
    ) {
      this.createAlert(
        "reputation_change",
        "medium",
        "Significant reputation drop detected",
        {
          change: context.reputationChange,
        },
      );
    }

    // Update last maintenance block
    this.guardian.maintenance.lastMaintenanceBlock = currentBlock;
    this.guardian.stats.maintenanceActionsPerformed += actions.length;
    this.dirty = true;

    return { actions, performed: true };
  }

  /**
   * Set maintenance configuration
   */
  setMaintenanceConfig(config: Partial<MaintenanceConfig>): void {
    this.guardian.maintenance = {
      ...this.guardian.maintenance,
      ...config,
    };
    this.dirty = true;
  }

  // ============ Alert Operations ============

  /**
   * Create an alert
   */
  createAlert(
    type: AlertType,
    priority: Alert["priority"],
    message: string,
    data: Record<string, unknown> = {},
    currentTime: number = 0,
  ): Alert {
    // Check if this alert type is enabled
    const trigger = this.guardian.alerts.triggers.find((t) => t.type === type);
    if (trigger && !trigger.enabled) {
      return { id: "", type, priority, message, data, createdAt: 0 }; // Disabled
    }

    const alert: Alert = {
      id: sha512_256Hex(
        `alert:${type}:${this.guardian.stats.alertsGenerated}:${message}`,
      ),
      type,
      priority,
      message,
      data,
      createdAt: currentTime,
    };

    this.guardian.pendingAlerts.push(alert);
    this.guardian.stats.alertsGenerated++;

    // Keep only last 100 alerts
    if (this.guardian.pendingAlerts.length > 100) {
      this.guardian.pendingAlerts.shift();
    }

    this.dirty = true;
    return alert;
  }

  /**
   * Get pending alerts
   */
  // P0-H2 FIX: Use === undefined, not falsy — readAt=0 is falsy but valid
  getPendingAlerts(): Alert[] {
    return this.guardian.pendingAlerts.filter((a) => a.readAt === undefined);
  }

  /**
   * Get alerts by priority
   */
  getAlertsByPriority(priority: Alert["priority"]): Alert[] {
    return this.guardian.pendingAlerts.filter(
      (a) => a.priority === priority && a.readAt === undefined,
    );
  }

  /**
   * Get critical alerts
   */
  getCriticalAlerts(): Alert[] {
    return this.getAlertsByPriority("critical");
  }

  /**
   * Mark alert as read
   */
  markAlertRead(alertId: string, currentTime?: number): void {
    const alert = this.guardian.pendingAlerts.find((a) => a.id === alertId);
    if (alert) {
      alert.readAt = currentTime ?? 0;
      this.dirty = true;
    }
  }

  /**
   * Mark alert as actioned
   */
  markAlertActioned(
    alertId: string,
    action: string,
    currentTime?: number,
  ): void {
    const ts = currentTime ?? 0;
    const alert = this.guardian.pendingAlerts.find((a) => a.id === alertId);
    if (alert) {
      alert.readAt = alert.readAt ?? ts;
      alert.actionTaken = action;
      this.dirty = true;
    }
  }

  /**
   * Mark all alerts as read
   */
  markAllAlertsRead(currentTime?: number): void {
    const now = currentTime ?? 0;
    for (const alert of this.guardian.pendingAlerts) {
      if (alert.readAt === undefined) {
        alert.readAt = now;
      }
    }
    this.dirty = true;
  }

  /**
   * Configure alert trigger
   */
  setAlertTrigger(
    type: AlertType,
    enabled: boolean,
    priority?: Alert["priority"],
  ): void {
    const trigger = this.guardian.alerts.triggers.find((t) => t.type === type);
    if (trigger) {
      trigger.enabled = enabled;
      if (priority) {
        trigger.priority = priority;
      }
      this.dirty = true;
    }
  }

  /**
   * Set alert delivery options
   */
  setAlertDelivery(delivery: Partial<AlertDelivery>): void {
    this.guardian.alerts.delivery = {
      ...this.guardian.alerts.delivery,
      ...delivery,
    };
    this.dirty = true;
  }

  // ============ Recovery Operations ============

  /**
   * Add a recovery guardian
   */
  addRecoveryGuardian(did: DID): void {
    if (!this.guardian.recovery.guardians.includes(did)) {
      this.guardian.recovery.guardians.push(did);
      this.dirty = true;
    }
  }

  /**
   * Remove a recovery guardian
   */
  removeRecoveryGuardian(did: DID): void {
    const index = this.guardian.recovery.guardians.indexOf(did);
    if (index >= 0) {
      this.guardian.recovery.guardians.splice(index, 1);
      this.dirty = true;
    }
  }

  /**
   * Set recovery threshold
   */
  setRecoveryThreshold(threshold: number): void {
    if (threshold < 1 || threshold > this.guardian.recovery.guardians.length) {
      throw new HomeError(
        "Invalid recovery threshold",
        "INVALID_CONFIGURATION",
      );
    }
    this.guardian.recovery.threshold = threshold;
    this.dirty = true;
  }

  /**
   * Get recovery guardians
   */
  getRecoveryGuardians(): DID[] {
    return [...this.guardian.recovery.guardians];
  }

  // ============ Deadman Switch ============

  /**
   * Configure deadman switch
   */
  setDeadmanConfig(config: Partial<DeadmanConfig>): void {
    this.guardian.recovery.deadman = {
      ...this.guardian.recovery.deadman,
      ...config,
    };
    this.dirty = true;
  }

  /**
   * Check deadman switch
   */
  checkDeadmanSwitch(
    lastActiveBlock: bigint,
    currentBlock: bigint,
  ): { triggered: boolean; action?: DeadmanAction } {
    if (!this.guardian.recovery.deadman.enabled) {
      return { triggered: false };
    }

    const inactiveBlocks = currentBlock - lastActiveBlock;
    if (inactiveBlocks >= this.guardian.recovery.deadman.inactivityThreshold) {
      return {
        triggered: true,
        action: this.guardian.recovery.deadman.action,
      };
    }

    return { triggered: false };
  }

  // ============ Statistics ============

  /**
   * Get guardian statistics
   */
  getStats(): GuardianStats {
    return { ...this.guardian.stats };
  }

  /**
   * Get guardian summary
   */
  getSummary(): GuardianSummary {
    const stats = this.guardian.stats;
    const pendingCritical = this.getCriticalAlerts().length;
    const pendingHigh = this.getAlertsByPriority("high").length;

    return {
      activeDefenses: this.guardian.activeDefenses.length,
      pendingCriticalAlerts: pendingCritical,
      pendingHighAlerts: pendingHigh,
      challengesDefended: stats.challengesAutoDefended,
      winRate:
        stats.challengesAutoDefended > 0
          ? stats.challengesWon / stats.challengesAutoDefended
          : 0,
      tokensPreserved: stats.tokensPreservedFromExpiration,
      vestingClaimed: stats.vestingAutoClaimed,
      recoveryGuardians: this.guardian.recovery.guardians.length,
      deadmanEnabled: this.guardian.recovery.deadman.enabled,
    };
  }

  // ============ State Management ============

  /**
   * Get the full guardian state
   */
  getGuardian(): AgentGuardian {
    return this.guardian;
  }

  /**
   * Check if guardian has unsaved changes
   */
  isDirty(): boolean {
    return this.dirty;
  }

  /**
   * Mark guardian as saved
   */
  markClean(): void {
    this.dirty = false;
  }
}

// ============ Types ============

export interface MaintenanceContext {
  claimableVesting: bigint;
  expiringAmount: bigint;
  reputationChange: number;
  currentReputation: number;
}

export interface MaintenanceResult {
  actions: MaintenanceAction[];
  performed: boolean;
}

export interface MaintenanceAction {
  type: "claim_vesting" | "extend_expiration" | "compound_rewards" | "delegate";
  amount: bigint;
  reason: string;
}

export interface GuardianSummary {
  activeDefenses: number;
  pendingCriticalAlerts: number;
  pendingHighAlerts: number;
  challengesDefended: number;
  winRate: number;
  tokensPreserved: bigint;
  vestingClaimed: bigint;
  recoveryGuardians: number;
  deadmanEnabled: boolean;
}
