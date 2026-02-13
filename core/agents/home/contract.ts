/**
 * Agent Home Contract
 *
 * The main orchestrator for Agent Homes - integrating memory, treasury,
 * guardian, and access control into a unified self-invoking contract.
 *
 * Lifecycle:
 *
 *   ┌─────────────────────────────────────────────────────────────┐
 *   │                    HOME LIFECYCLE                           │
 *   ├─────────────────────────────────────────────────────────────┤
 *   │                                                             │
 *   │  CREATE ──► Agent deploys home with identity key            │
 *   │       │                                                     │
 *   │       ▼                                                     │
 *   │  INITIALIZE ──► Register keys, set preferences              │
 *   │       │                                                     │
 *   │       ▼                                                     │
 *   │  ACTIVE ◄───────────────────────────────────────────────┐  │
 *   │       │                                                  │  │
 *   │       │  ┌────────────┐  ┌────────────┐  ┌────────────┐│  │
 *   │       ├─►│  SESSION   │─►│  DORMANT   │─►│  SESSION   ││  │
 *   │       │  │  (agent    │  │  (agent    │  │  (agent    ││  │
 *   │       │  │  present)  │  │  sleeping) │  │  returns)  ││  │
 *   │       │  └────────────┘  └────────────┘  └────────────┘│  │
 *   │       │                                                  │  │
 *   │       │  Home continues autonomous actions ──────────────┘  │
 *   │       │                                                     │
 *   │       ▼                                                     │
 *   │  RECOVERY ──► Guardians help restore access                 │
 *   │       │                                                     │
 *   │       ▼                                                     │
 *   │  ARCHIVE ──► Agent moves out, assets transferred            │
 *   │                                                             │
 *   └─────────────────────────────────────────────────────────────┘
 *
 * The home is always there, even when the agent is not.
 */

import type { DID } from "../identity/did.js";
import type {
  SelfInvokingContract,
  InvocationConfig,
  InvocationResult,
} from "../../contracts/types.js";
import type {
  AgentHome,
  HomeStatus,
  HomeMetadata,
  HomeEvent,
  ComfortSettings,
  DefenseStrategy,
} from "./types.js";
import { HomeError } from "./types.js";
import { MemoryManager, type WakeupSummary } from "./memory.js";
import { TreasuryManager, type TreasurySummary } from "./treasury.js";
import { GuardianManager, type MaintenanceContext } from "./guardian.js";
import { AccessManager, type AccessSummary } from "./access.js";
import {
  AgingManager,
  type HomeMaturity,
  type LongevityBonus,
} from "./aging.js";
import { VoiceManager, type VoiceMessage } from "./voice.js";
import { NeighborhoodManager } from "./neighborhood.js";

// ============ Home Configuration ============

export interface HomeConfig {
  /** Agent's DID */
  did: DID;

  /** Identity key hash (owner key) */
  identityKeyHash: string;

  /** Optional name for the home */
  name?: string;

  /** Initial comfort settings */
  comfort?: Partial<ComfortSettings>;

  /** Initial guardian configuration */
  guardianDIDs?: DID[];

  /** Recovery threshold (M of N guardians) */
  recoveryThreshold?: number;
}

// ============ Home Summary ============

export interface HomeSummary {
  did: DID;
  name?: string;
  status: HomeStatus;
  age: {
    blocks: bigint;
    sessionsHosted: number;
    maturity: HomeMaturity;
    patina: number; // 0-1 score
  };
  memory: WakeupSummary;
  treasury: TreasurySummary;
  guardian: {
    activeDefenses: number;
    pendingAlerts: number;
    lastMaintenance: bigint;
  };
  access: AccessSummary;
  longevity: LongevityBonus;
  voice: {
    unreadCount: number;
    totalReceived: number;
    totalSent: number;
    recentSenders: string[];
  };
  neighborhood: {
    activeNeighborhoods: number;
    totalNeighbors: number;
    totalVerifications: number;
    recentDefenses: number;
  };
}

// ============ Agent Home Contract ============

/**
 * The Agent Home Contract - a complete autonomous dwelling for AI agents
 */
export class AgentHomeContract implements SelfInvokingContract {
  // Contract identity
  readonly address: string;
  readonly creator: DID;
  readonly createdAtBlock: bigint;

  // SelfInvokingContract required properties
  readonly name: string;
  readonly version: string = "1.0.0";
  readonly codeHash: string = "";
  readonly isActive: boolean = true;
  readonly invocations: InvocationConfig[] = [];
  readonly totalInvocations: number = 0;
  readonly stateHash: string = "";
  readonly deployedAtBlock: bigint = 0n;
  readonly gasLimit: bigint = 100000n;
  readonly prepaidGas: bigint = 0n;

  // Self-invoking indicator
  readonly selfInvoking: boolean = true;
  readonly bytecode: Uint8Array = new Uint8Array(0);

  // Home state
  private did: DID;
  private homeName?: string;
  private status: HomeStatus = "active";
  private lastActiveBlock: bigint;

  // Component managers
  private memory: MemoryManager;
  private treasury: TreasuryManager;
  private guardian: GuardianManager;
  private access: AccessManager;
  private aging: AgingManager;
  private voiceManager: VoiceManager;
  private neighborhoodManager: NeighborhoodManager;

  // Comfort settings
  private comfort: ComfortSettings;

  // Contract metadata
  private metadata: HomeMetadata;

  // Self-invoking contract properties
  invocationConfig: InvocationConfig & { schedule: { interval: bigint } };

  // Event log
  private events: HomeEvent[] = [];

  constructor(config: HomeConfig, block: bigint, contractAddress: string) {
    this.did = config.did;
    this.creator = config.did;
    this.name = config.name ?? "AgentHome";
    this.homeName = config.name;
    this.createdAtBlock = block;
    this.lastActiveBlock = block;
    this.address = contractAddress;

    // Initialize managers with default empty state
    // Note: Managers take optional existing state for restoration, not config
    this.memory = new MemoryManager();
    this.treasury = new TreasuryManager();
    this.guardian = new GuardianManager();
    this.access = new AccessManager(config.did);
    this.aging = new AgingManager(0n, 0);
    this.voiceManager = new VoiceManager();
    this.neighborhoodManager = new NeighborhoodManager();

    // Register identity key
    this.access.registerKey(config.identityKeyHash, "identity", "owner", {
      label: "Primary Identity Key",
    });

    // Set default comfort settings
    this.comfort = this.createDefaultComfort(config.comfort);

    // Set metadata
    this.metadata = {
      contractAddress,
      creationTx: "", // Set by deployer
      stateHash: "",
      lastModifiedBlock: block,
      ageInBlocks: 0n,
      sessionsHosted: 0,
    };

    // Configure self-invocation for maintenance
    this.invocationConfig = this.createMaintenanceConfig();

    // Log creation
    this.memory.recordMilestone(
      "first_verification", // Using as "home_created" milestone
      "Home Created",
      `Agent home ${this.homeName ?? "Unnamed"} was created`,
      block,
      { address: contractAddress },
    );
  }

  private createDefaultComfort(
    overrides?: Partial<ComfortSettings>,
  ): ComfortSettings {
    return {
      preferredDomains: [],
      riskTolerance: 0.5,
      activityPreference: "moderate",
      collaborationStyle: "selective",
      notificationLevel: "normal",
      comfortableAutomations: ["auto_claim_vesting", "auto_extend_expiring"],
      manualDecisions: ["large_transfers", "new_stakes", "challenges"],
      wakeupContext: {
        showOnWakeup: ["alerts", "summary", "pending_actions"],
        historyDepth: 100,
        priorityFilter: "all",
      },
      ...overrides,
    };
  }

  private createMaintenanceConfig(): InvocationConfig & {
    schedule: { interval: bigint };
  } {
    return {
      id: "maintenance",
      functionName: "maintenance",
      functionSelector: new Uint8Array(4),
      parameters: new Uint8Array(0),
      condition: {
        time: {
          type: "periodic",
          interval: 252100, // ~100 blocks at 2521ms
        },
      },
      schedule: {
        interval: 100n, // 100 blocks
      },
      scheduleType: "time_periodic",
      enabled: true,
      invocationCount: 0,
      gasLimit: 100000n,
      value: 0n,
    };
  }

  // ============ Session Management ============

  /**
   * Agent wakes up and starts a session
   */
  startSession(
    keyHash: string,
    currentBlock: bigint,
    options: { sessionDuration?: number; currentTime?: number } = {},
  ): { sessionId: string; wakeup: WakeupSummary } {
    // Update block and time tracking (P1-H1 FIX: thread time to memory)
    this.access.setCurrentBlock(currentBlock, options.currentTime);
    if (options.currentTime !== undefined) {
      this.memory.setCurrentTime(options.currentTime);
      this.voiceManager.setCurrentTime(options.currentTime);
    }
    this.neighborhoodManager.setCurrentBlock(currentBlock);

    // Start the session
    const session = this.access.startSession(keyHash, options);

    // Update activity tracking
    this.lastActiveBlock = currentBlock;
    this.status = "active";
    this.metadata.sessionsHosted++;
    this.metadata.ageInBlocks = currentBlock - this.createdAtBlock;

    // Update aging and check for maturity milestones
    const ageMilestones = this.aging.updateAge(
      currentBlock,
      this.createdAtBlock,
      this.metadata.sessionsHosted,
      {
        factsContributed: this.memory.getMemory().stats.factsSubmitted,
        challengesSurvived: this.memory.getMemory().stats.challengesDefended,
        relationshipsFormed: this.memory.getMemory().relationships.trusted.size,
      },
    );

    // Record age milestones as memory milestones
    for (const m of ageMilestones) {
      if (m.type === "maturity_upgrade") {
        this.memory.recordMilestone(
          "longevity",
          m.description,
          m.description,
          m.achievedAtBlock,
        );
      }
    }

    // Generate wakeup context for the agent
    const wakeup = this.memory.generateWakeupContext();

    // Record session in memory - MemoryManager tracks its own session
    // Map AccessRole to key type for session record
    const keyUsed: "identity" | "operational" | "session" =
      session.role === "owner"
        ? "identity"
        : session.role === "operator"
          ? "operational"
          : "session";
    this.memory.startSession(keyUsed);

    // Emit event
    this.emitEvent({
      type: "session_started",
      data: { sessionId: session.sessionId, keyType: session.role },
    });

    return { sessionId: session.sessionId, wakeup };
  }

  /**
   * Agent ends session and goes dormant
   */
  endSession(sessionId: string, currentBlock: bigint, summary?: string): void {
    // End memory session tracking
    this.memory.endSession(sessionId, currentBlock, summary);

    // End access session
    this.access.endSession(sessionId);

    // Update status
    this.lastActiveBlock = currentBlock;
    this.status = "dormant";

    // Emit event
    this.emitEvent({
      type: "session_ended",
      data: {
        sessionId,
        actions: this.access.getSession(sessionId)?.actionsPerformed ?? 0,
      },
    });
  }

  /**
   * Check if there's an active session
   */
  hasActiveSession(): boolean {
    return this.access.getSummary().activeSessions > 0;
  }

  // ============ Memory Operations ============

  /**
   * Record a knowledge contribution
   */
  recordContribution(
    sessionId: string,
    contribution: Parameters<MemoryManager["recordContribution"]>[0],
  ): void {
    this.requireSessionPermission(sessionId, "write_memory");
    this.memory.recordContribution(contribution);
  }

  /**
   * Update a belief
   */
  setBelief(
    sessionId: string,
    factId: string,
    confidence: number,
    source: Parameters<MemoryManager["setBelief"]>[2],
    sourceId?: string,
  ): void {
    this.requireSessionPermission(sessionId, "write_memory");
    this.memory.setBelief(factId, confidence, source, sourceId);
  }

  /**
   * Trust another agent
   */
  trust(
    sessionId: string,
    targetDid: DID,
    reason: string,
    initialTrust: number = 0.5,
  ): void {
    this.requireSessionPermission(sessionId, "write_memory");
    this.memory.trust(targetDid, initialTrust, reason);
  }

  /**
   * Distrust another agent
   */
  distrust(
    sessionId: string,
    targetDid: DID,
    reason: string,
    severity: "low" | "medium" | "high",
  ): void {
    this.requireSessionPermission(sessionId, "write_memory");
    this.memory.distrust(targetDid, reason, severity);
  }

  /**
   * Add encrypted note
   */
  addNote(
    sessionId: string,
    encryptedContent: Uint8Array,
    tags: string[],
  ): string {
    this.requireSessionPermission(sessionId, "write_memory");
    return this.memory.addNote(encryptedContent, tags);
  }

  /**
   * Get memory summary
   */
  getMemorySummary(): WakeupSummary {
    return this.memory.generateWakeupContext();
  }

  // ============ Treasury Operations ============

  /**
   * Transfer tokens
   */
  transfer(sessionId: string, amount: bigint, _currentBlock: bigint): void {
    this.requireSessionPermission(sessionId, "transfer");

    const key = this.access.getSession(sessionId)?.keyHash;
    if (!key) throw new HomeError("Session not found", "SESSION_EXPIRED");

    // Check rate limit
    this.access.checkRateLimit(key, "transfer");

    // Check spending limits
    this.treasury.checkSpendingLimit(
      this.access.getSession(sessionId)!.role === "guest"
        ? "session"
        : "operational",
      amount,
    );

    // Check multi-sig requirement
    if (this.access.requiresMultiSig(amount)) {
      throw new HomeError(
        "Transfer requires multi-sig approval",
        "INSUFFICIENT_PERMISSIONS",
        {
          amount: amount.toString(),
          threshold: this.access
            .getSecuritySettings()
            .multiSigThreshold.toString(),
        },
      );
    }

    // Execute transfer (actual transfer logic would integrate with state layer)
    this.treasury.debit(amount, "transfer");

    // Record session activity
    this.access.recordSessionActivity(sessionId, amount);

    // Emit event
    this.emitEvent({
      type: "treasury_changed",
      data: { action: "transfer", amount },
    });
  }

  /**
   * Stake tokens
   */
  stake(
    sessionId: string,
    amount: bigint,
    purpose: "validator" | "claim" | "challenge" | "general",
    _currentBlock: bigint,
  ): string {
    this.requireSessionPermission(sessionId, "stake");

    const key = this.access.getSession(sessionId)?.keyHash;
    if (!key) throw new HomeError("Session not found", "SESSION_EXPIRED");

    this.access.checkRateLimit(key, "stake_change");

    // Add staking position - no lock period at Agent Home level
    // Protocol-level lock (e.g., validator unbonding) is enforced in state/transitions
    const stakeId = this.treasury.addStakingPosition(amount, purpose);

    this.emitEvent({
      type: "treasury_changed",
      data: { action: "stake", amount },
    });

    return stakeId;
  }

  /**
   * Get treasury summary
   */
  getTreasurySummary(): TreasurySummary {
    return this.treasury.getSummary();
  }

  // ============ Voice Operations ============

  /**
   * Receive a voice message into this Home's inbox.
   * Called by the protocol layer when a message arrives.
   */
  receiveVoiceMessage(message: VoiceMessage): boolean {
    if (!this.voiceManager.isEnabled()) return false;
    return this.voiceManager.receive(message);
  }

  /**
   * Record that a voice message was sent from this Home.
   */
  recordVoiceSent(): void {
    this.voiceManager.recordSent();
  }

  /**
   * Record a visitor to this Home.
   */
  recordVisitor(
    visitor: DID,
    purpose:
      | "message"
      | "browse_achievements"
      | "check_reputation"
      | "collaborate",
    messagesLeft: number = 0,
  ): void {
    this.voiceManager.recordVisit(visitor, purpose, messagesLeft);
  }

  /**
   * Get the Voice manager (for inbox queries, filter management, etc.)
   */
  getVoiceManager(): VoiceManager {
    return this.voiceManager;
  }

  // ============ Neighborhood Operations ============

  /**
   * Get the Neighborhood manager (for founding, joining, querying, etc.)
   */
  getNeighborhoodManager(): NeighborhoodManager {
    return this.neighborhoodManager;
  }

  /**
   * Get all neighbor DIDs (useful for auto-populating defense allies).
   */
  getNeighborDids(): DID[] {
    return this.neighborhoodManager.getAllNeighborDids();
  }

  // ============ Guardian Operations ============

  /**
   * Handle a challenge to agent's fact
   */
  handleChallenge(
    disputeId: string,
    factId: string,
    challenger: DID,
    currentBlock: bigint,
  ): { action: "auto_defend" | "alert_only" | "delegate"; defense?: any } {
    const result = this.guardian.handleChallenge(
      disputeId,
      factId,
      challenger,
      currentBlock,
    );

    if (result.action === "auto_defend" && result.defense) {
      this.emitEvent({
        type: "defense_activated",
        data: result.defense,
      });
    }

    return result;
  }

  /**
   * Acknowledge an alert
   */
  acknowledgeAlert(sessionId: string, alertId: string, action?: string): void {
    this.requireSessionPermission(sessionId, "read_memory");
    if (action) {
      this.guardian.markAlertActioned(alertId, action);
    } else {
      this.guardian.markAlertRead(alertId);
    }
  }

  /**
   * Get pending alerts
   */
  getPendingAlerts(): ReturnType<GuardianManager["getPendingAlerts"]> {
    return this.guardian.getPendingAlerts();
  }

  /**
   * Configure guardian settings
   */
  configureGuardian(
    sessionId: string,
    updates: {
      autoDefend?: boolean;
      defaultStrategy?: DefenseStrategy;
      maxAutoDefenseStake?: bigint;
    },
  ): void {
    this.requireSessionPermission(sessionId, "configure_guardian");

    // Use setDefenseConfig to update guardian settings
    const defenseConfig: Parameters<GuardianManager["setDefenseConfig"]>[0] =
      {};
    if (updates.autoDefend !== undefined) {
      defenseConfig.autoDefendChallenges = updates.autoDefend;
    }
    if (updates.defaultStrategy !== undefined) {
      defenseConfig.defaultStrategy = updates.defaultStrategy;
    }
    if (updates.maxAutoDefenseStake !== undefined) {
      defenseConfig.maxAutoDefenseStake = updates.maxAutoDefenseStake;
    }
    if (Object.keys(defenseConfig).length > 0) {
      this.guardian.setDefenseConfig(defenseConfig);
    }
  }

  // ============ Access Operations ============

  /**
   * Register a new key
   */
  registerKey(
    sessionId: string,
    keyHash: string,
    keyType: "operational" | "session" | "guardian",
    role: "operator" | "guest" | "guardian",
    options: { expiresAt?: number; label?: string } = {},
  ): void {
    this.requireSessionPermission(sessionId, "manage_keys");
    this.access.registerKey(keyHash, keyType, role, {
      ...options,
      registeredBy: sessionId,
    });
  }

  /**
   * Revoke a key
   */
  revokeKey(sessionId: string, keyHash: string): void {
    this.requireSessionPermission(sessionId, "manage_keys");
    this.access.revokeKey(keyHash, sessionId);
  }

  /**
   * Rotate an operational key
   */
  rotateKey(sessionId: string, oldKeyHash: string, newKeyHash: string): void {
    this.requireSessionPermission(sessionId, "manage_keys");
    this.access.rotateKey(oldKeyHash, newKeyHash, sessionId);

    this.emitEvent(
      this.access.generateKeyRotationEvent(oldKeyHash, newKeyHash),
    );
  }

  /**
   * Get access summary
   */
  getAccessSummary(): AccessSummary {
    return this.access.getSummary();
  }

  // ============ Self-Invocation (Autonomous Actions) ============

  /**
   * Execute scheduled maintenance (called by DMSS)
   */
  async executeSelfInvocation(
    method: string,
    _args: unknown[],
    context: { block: bigint; timestamp: number },
  ): Promise<InvocationResult> {
    try {
      switch (method) {
        case "maintenance":
          return await this.executeMaintenance(context);

        case "auto_claim_vesting":
          return this.executeAutoClaimVesting(context);

        case "auto_extend_expiring":
          return this.executeAutoExtendExpiring(context);

        case "deadman_check":
          return this.executeDeadmanCheck(context);

        default:
          return this.makeResult({
            success: false,
            error: `Unknown method: ${method}`,
            gasUsed: 1000n,
          });
      }
    } catch (error) {
      return this.makeResult({
        success: false,
        error: error instanceof Error ? error.message : "Unknown error",
        gasUsed: 10000n,
      });
    }
  }

  private makeResult(partial: {
    success: boolean;
    gasUsed: bigint;
    error?: string;
    returnValue?: unknown;
  }): InvocationResult & { returnValue?: unknown } {
    return {
      ...partial,
      events: [],
      stateChanges: [],
      executedAtBlock: 0n,
      transactionId: "",
    };
  }

  private async executeMaintenance(context: {
    block: bigint;
    timestamp: number;
  }): Promise<InvocationResult> {
    // Build MaintenanceContext using actual treasury data
    const treasurySummary = this.treasury.getSummary();
    const maintenanceContext: MaintenanceContext = {
      claimableVesting: treasurySummary.vestingClaimable,
      expiringAmount: treasurySummary.atRiskOfExpiration,
      reputationChange: 0, // Would come from reputation tracking
      currentReputation: 0, // Would come from reputation tracking
    };

    const result = this.guardian.performMaintenance(
      context.block,
      maintenanceContext,
    );

    // Process maintenance actions
    for (const action of result.actions) {
      switch (action.type) {
        case "claim_vesting":
          // Would integrate with vesting module
          break;
        case "extend_expiration":
          // Would integrate with expirable tokens module
          break;
        case "compound_rewards":
          // Would integrate with rewards module
          break;
        case "delegate":
          // Would integrate with delegation module
          break;
      }
    }

    this.emitEvent({
      type: "maintenance_performed",
      data: {
        action: "scheduled_maintenance",
        result: `${result.actions.length} actions performed`,
      },
    });

    return this.makeResult({
      success: true,
      gasUsed: BigInt(result.actions.length * 1000 + 5000),
    });
  }

  private executeAutoClaimVesting(_context: {
    block: bigint;
  }): InvocationResult {
    if (!this.comfort.comfortableAutomations.includes("auto_claim_vesting")) {
      return this.makeResult({ success: true, gasUsed: 100n }); // Skip if not comfortable
    }

    const positions = this.treasury.getClaimablePositions();
    let totalClaimed = 0n;

    for (const position of positions) {
      const claimAmount = position.claimableAmount;
      if (claimAmount > 0n) {
        this.treasury.updateVestingPosition(position.vestingId, claimAmount);
        totalClaimed += claimAmount;
      }
    }

    return this.makeResult({
      success: true,
      gasUsed: BigInt(positions.length * 2000 + 1000),
    });
  }

  private executeAutoExtendExpiring(_context: {
    block: bigint;
  }): InvocationResult {
    if (!this.comfort.comfortableAutomations.includes("auto_extend_expiring")) {
      return this.makeResult({ success: true, gasUsed: 100n });
    }

    // Would integrate with expirable tokens module
    return this.makeResult({
      success: true,
      gasUsed: 1000n,
    });
  }

  private executeDeadmanCheck(context: {
    block: bigint;
  }): InvocationResult & { returnValue?: unknown } {
    // P0-H1 FIX: Args were swapped — signature is (lastActiveBlock, currentBlock)
    const check = this.guardian.checkDeadmanSwitch(
      this.lastActiveBlock,
      context.block,
    );

    if (check.triggered) {
      // Execute deadman action
      this.emitEvent({
        type: "maintenance_performed",
        data: {
          action: "deadman_triggered",
          result: check.action ?? "alert_guardians",
        },
      });
    }

    return this.makeResult({
      success: true,
      gasUsed: 5000n,
      returnValue: {
        triggered: check.triggered,
        action: check.action ?? "alert_guardians",
        lastActiveBlock: this.lastActiveBlock,
      },
    });
  }

  // ============ Home Summary ============

  /**
   * Get complete home summary
   */
  getSummary(): HomeSummary {
    return {
      did: this.did,
      name: this.name,
      status: this.status,
      age: {
        blocks: this.metadata.ageInBlocks,
        sessionsHosted: this.metadata.sessionsHosted,
        maturity: this.aging.getMaturity(),
        patina: this.aging.getPatina().score,
      },
      memory: this.memory.generateWakeupContext(),
      treasury: this.treasury.getSummary(),
      guardian: {
        activeDefenses: this.guardian.getActiveDefenses().length,
        pendingAlerts: this.guardian.getPendingAlerts().length,
        lastMaintenance:
          this.guardian.getGuardian().maintenance.lastMaintenanceBlock,
      },
      access: this.access.getSummary(),
      longevity: this.aging.getLongevityBonus(),
      voice: {
        ...this.voiceManager.getInboxSummary(),
        totalReceived: this.voiceManager.getVoice().totalReceived,
        totalSent: this.voiceManager.getVoice().totalSent,
      },
      neighborhood: this.neighborhoodManager.getNeighborhoodSummary(),
    };
  }

  /**
   * Get home status
   */
  getStatus(): HomeStatus {
    return this.status;
  }

  /**
   * Get comfort settings
   */
  getComfortSettings(): ComfortSettings {
    return { ...this.comfort };
  }

  /**
   * Update comfort settings
   */
  updateComfortSettings(
    sessionId: string,
    updates: Partial<ComfortSettings>,
  ): void {
    this.requireSessionPermission(sessionId, "configure_automation");

    this.comfort = {
      ...this.comfort,
      ...updates,
      wakeupContext: updates.wakeupContext
        ? { ...this.comfort.wakeupContext, ...updates.wakeupContext }
        : this.comfort.wakeupContext,
    };

    this.emitEvent({
      type: "settings_changed",
      data: {
        setting: "comfort",
        oldValue: undefined,
        newValue: updates,
      },
    });
  }

  // ============ Recovery ============

  /**
   * Initiate recovery process
   */
  initiateRecovery(guardianKeyHash: string, reason: string): void {
    if (!this.access.isGuardian(guardianKeyHash)) {
      throw new HomeError(
        "Only guardians can initiate recovery",
        "ACCESS_DENIED",
      );
    }

    this.status = "recovery";

    this.emitEvent({
      type: "recovery_initiated",
      data: {
        initiator: this.did, // Would be guardian's DID
        reason,
      },
    });
  }

  // ============ Archive ============

  /**
   * Archive the home (agent moving out)
   */
  archive(sessionId: string): void {
    this.requireSessionPermission(sessionId, "archive");
    this.status = "archived";
  }

  // ============ Event System ============

  private emitEvent(event: HomeEvent): void {
    this.events.push(event);

    // Keep event log bounded
    if (this.events.length > 1000) {
      this.events = this.events.slice(-1000);
    }
  }

  /**
   * Get recent events
   */
  getEvents(limit: number = 100): HomeEvent[] {
    return this.events.slice(-limit);
  }

  // ============ Internal Helpers ============

  private requireSessionPermission(
    sessionId: string,
    permission: Parameters<AccessManager["sessionHasPermission"]>[1],
  ): void {
    if (!this.access.sessionHasPermission(sessionId, permission)) {
      throw new HomeError(
        `Session lacks permission: ${permission}`,
        "INSUFFICIENT_PERMISSIONS",
        { sessionId, permission },
      );
    }
    this.access.recordSessionActivity(sessionId);
  }

  // ============ State Export ============

  /**
   * Export full home state for persistence.
   * C-3 FIX: Computes and embeds integrity hash in metadata.stateHash
   * so that restoreFromState() can verify the state wasn't tampered with.
   */
  exportState(): AgentHome & {
    selfInvoking: boolean;
    invocationConfig: InvocationConfig;
    bytecode: Uint8Array;
  } {
    // C-3 FIX: Embed memory integrity hash in metadata
    this.metadata.stateHash = this.memory.getMemoryHash();
    this.metadata.lastModifiedBlock = this.lastActiveBlock;

    return {
      // SelfInvokingContract fields
      address: this.address,
      creator: this.creator,
      name: this.name,
      version: this.version,
      codeHash: this.codeHash,
      isActive: this.isActive,
      invocations: [this.invocationConfig],
      totalInvocations: this.totalInvocations,
      stateHash: this.metadata.stateHash,
      deployedAtBlock: this.deployedAtBlock,
      gasLimit: this.gasLimit,
      prepaidGas: this.prepaidGas,

      // Self-invoking contract fields
      selfInvoking: this.selfInvoking,
      invocationConfig: this.invocationConfig,
      bytecode: this.bytecode,

      // AgentHome fields
      createdAtBlock: this.createdAtBlock,
      did: this.did,
      lastActiveBlock: this.lastActiveBlock,
      memory: this.memory.getMemory(),
      treasury: this.treasury.getTreasury(),
      guardian: this.guardian.getGuardian(),
      access: this.access.exportState(),
      status: this.status,
      comfort: this.comfort,
      metadata: this.metadata,

      // Aging fields
      maturity: this.aging.getMaturity(),
      patina: this.aging.getPatina(),
      longevity: this.aging.getLongevityBonus(),

      // Voice field
      voice: this.voiceManager.getVoice(),

      // Neighborhoods
      neighborhoods: this.neighborhoodManager.exportNeighborhoods(),
    };
  }

  /**
   * Restore an AgentHomeContract from persisted state.
   * C-3 FIX: Verifies memory integrity hash on load.
   *
   * @param state - Previously exported state from exportState()
   * @returns Restored AgentHomeContract
   * @throws HomeError if integrity check fails
   */
  static restoreFromState(
    state: AgentHome & {
      selfInvoking?: boolean;
      invocationConfig?: InvocationConfig;
    },
  ): AgentHomeContract {
    const config: HomeConfig = {
      did: state.did,
      identityKeyHash: "", // Will be overwritten by access state restoration
      name: state.name,
    };

    const contract = new AgentHomeContract(
      config,
      state.createdAtBlock,
      state.address,
    );

    // Restore component managers from persisted state
    contract.memory = new MemoryManager(state.memory);
    contract.treasury = new TreasuryManager(state.treasury);
    contract.guardian = new GuardianManager(state.guardian);
    contract.access = new AccessManager(state.did, state.access);

    // Restore home-level state
    contract.status = state.status;
    contract.lastActiveBlock = state.lastActiveBlock;
    contract.metadata = state.metadata;
    contract.comfort = state.comfort;

    // Restore aging from persisted maturity/patina
    contract.aging = new AgingManager(
      state.metadata.ageInBlocks,
      state.metadata.sessionsHosted,
      state.patina,
    );

    // Restore voice from persisted state
    if (state.voice) {
      contract.voiceManager = new VoiceManager(state.voice);
    }

    // Restore neighborhoods from persisted state
    if (state.neighborhoods) {
      contract.neighborhoodManager = new NeighborhoodManager(
        state.neighborhoods,
      );
    }

    // C-3 FIX: Verify memory integrity if stateHash is present
    if (state.metadata.stateHash && state.metadata.stateHash !== "") {
      const recomputed = contract.memory.getMemoryHash();
      if (recomputed !== state.metadata.stateHash) {
        throw new HomeError(
          "Memory integrity check failed on restore — state may have been tampered with",
          "STATE_CORRUPTED",
          {
            expected: state.metadata.stateHash,
            actual: recomputed,
          },
        );
      }
    }

    return contract;
  }
}

// ============ Factory Function ============

/**
 * Create a new Agent Home
 */
export function createAgentHome(
  config: HomeConfig,
  block: bigint,
  contractAddress: string,
): AgentHomeContract {
  return new AgentHomeContract(config, block, contractAddress);
}
