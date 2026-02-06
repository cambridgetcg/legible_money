/**
 * Pot Expiration & Treasury Recovery System
 *
 * Handles edge cases where the common pot becomes inaccessible:
 * 1. DISAGREEMENT: Deadlock where neither party will sign
 * 2. HUMAN DECEASE: Human passes away or becomes incapacitated
 * 3. AGENT API LOSS: Agent's mother company shuts down or revokes API
 *
 * Design Philosophy:
 * - Long grace periods to prevent false positives
 * - Multiple warning stages before expiration
 * - Recovery options at each stage
 * - Final fallback to treasury (funds benefit ecosystem)
 *
 * Scheduling: Uses on-chain block-based scheduling with heartbeat proofs
 *
 * @module partnership/pot_expiration
 */

import type { CommonPot, ConsensusOperation } from "./types.js";
import {
  sha512_256,
  sha512_256Hex,
  domainHash,
} from "../core/crypto/hashing.js";

// ============ Types ============

/**
 * Expiration trigger types
 */
export type ExpirationTrigger =
  | "disagreement_deadlock" // Neither party signing for extended period
  | "human_inactivity" // No human proof-of-life
  | "agent_inactivity" // No agent proof-of-life
  | "agent_api_revoked" // Agent API access revoked
  | "agent_company_shutdown" // Agent mother company ceased operations
  | "mutual_inactivity" // Both parties inactive
  | "manual_expiration"; // Explicitly triggered (with proof)

/**
 * Expiration state for a pot
 */
export interface PotExpirationState {
  // Current phase
  phase: ExpirationPhase;

  // Heartbeat tracking
  humanHeartbeat: HeartbeatState;
  agentHeartbeat: HeartbeatState;

  // Agent API status
  agentApiStatus: AgentApiStatus;

  // Deadlock detection
  deadlockState: DeadlockState;

  // Expiration schedule
  schedule: ExpirationSchedule;

  // Recovery options
  recoveryOptions: RecoveryOptions;

  // Beneficiary (for human decease)
  humanBeneficiary?: BeneficiaryDesignation;

  // Final state
  expired: boolean;
  expiredAt?: bigint;
  expirationTrigger?: ExpirationTrigger;
  treasuryRecoveryTx?: string;
}

/**
 * Expiration phases (progressive)
 */
export type ExpirationPhase =
  | "active" // Normal operation
  | "warning" // First inactivity warning
  | "alert" // Serious inactivity concern
  | "critical" // Last chance before expiration
  | "grace_period" // Final grace period
  | "expired" // Pot has expired
  | "recovered"; // Funds recovered to treasury

/**
 * Heartbeat state for proof-of-life
 */
export interface HeartbeatState {
  party: "human" | "agent";

  // Last heartbeat
  lastHeartbeatBlock: bigint;
  lastHeartbeatProof: Uint8Array;

  // Consecutive misses
  missedHeartbeats: number;
  maxMissedBeforeWarning: number;
  maxMissedBeforeCritical: number;

  // Status
  status: "alive" | "warning" | "critical" | "presumed_inactive";
}

/**
 * Agent API status tracking
 */
export interface AgentApiStatus {
  // API provider info
  providerId: string; // e.g., "anthropic", "openai"
  apiEndpoint?: string;

  // Last verification
  lastApiCheckBlock: bigint;
  lastApiCheckResult:
    | "active"
    | "degraded"
    | "unreachable"
    | "revoked"
    | "unknown";

  // Oracle attestations
  oracleAttestations: OracleAttestation[];

  // Company status (if trackable)
  companyStatus:
    | "operating"
    | "warning"
    | "shutting_down"
    | "shutdown"
    | "unknown";

  // Consecutive failures
  consecutiveFailures: number;
  maxFailuresBeforeWarning: number;
}

/**
 * Oracle attestation for external status
 */
export interface OracleAttestation {
  oracleId: string;
  attestedAt: bigint;
  status: AgentApiStatus["lastApiCheckResult"];
  signature: Uint8Array;
  expiresAt: bigint;
}

/**
 * Deadlock detection state
 */
export interface DeadlockState {
  // Pending operations tracking
  oldestPendingOperation?: string;
  oldestPendingBlock?: bigint;

  // Rejection tracking
  consecutiveRejections: number;
  lastRejectionBlock?: bigint;

  // Deadlock detection
  isDeadlocked: boolean;
  deadlockDetectedAt?: bigint;
  deadlockReason?: string;
}

/**
 * Expiration schedule
 */
export interface ExpirationSchedule {
  // Warning thresholds (in blocks)
  warningAfterBlocks: bigint; // First warning
  alertAfterBlocks: bigint; // Alert status
  criticalAfterBlocks: bigint; // Critical status
  gracePeriodBlocks: bigint; // Final grace period
  expirationAfterBlocks: bigint; // Total blocks before expiration

  // Scheduled events
  scheduledEvents: ScheduledEvent[];

  // Next scheduled check
  nextCheckBlock: bigint;
}

/**
 * Scheduled event
 */
export interface ScheduledEvent {
  eventId: string;
  eventType:
    | "warning"
    | "alert"
    | "critical"
    | "grace_start"
    | "expiration"
    | "heartbeat_check"
    | "api_check";
  scheduledBlock: bigint;
  executed: boolean;
  executedAt?: bigint;
  result?: string;
}

/**
 * Recovery options before expiration
 */
export interface RecoveryOptions {
  // Human recovery
  humanCanRecover: boolean;
  humanRecoveryDeadline?: bigint;

  // Agent recovery
  agentCanRecover: boolean;
  agentRecoveryDeadline?: bigint;

  // Beneficiary recovery
  beneficiaryCanRecover: boolean;
  beneficiaryRecoveryDeadline?: bigint;

  // Guardian recovery (if configured)
  guardianRecoveryEnabled: boolean;
  requiredGuardians?: number;
  guardianRecoveryDeadline?: bigint;

  // Protocol recovery (last resort)
  protocolRecoveryEnabled: boolean;
}

/**
 * Beneficiary designation (for human decease)
 */
export interface BeneficiaryDesignation {
  beneficiaryType: "did" | "address" | "guardian_set";

  // Direct beneficiary
  beneficiaryId?: string;
  beneficiaryPublicKey?: string;

  // Guardian set (if multiple)
  guardianThreshold?: number;
  guardians?: string[];

  // Conditions
  activationCondition:
    | "human_inactivity"
    | "death_certificate"
    | "guardian_attestation";
  activationDelay: bigint; // Extra delay after condition met

  // Designated at
  designatedAt: bigint;
  lastUpdatedAt: bigint;
}

/**
 * Treasury recovery record
 */
export interface TreasuryRecoveryRecord {
  potId: string;
  partnershipId: string;
  recoveredAmount: bigint;
  recoveryTrigger: ExpirationTrigger;
  recoveredAt: bigint;
  treasuryAddress: string;

  // Distribution
  researchFund: bigint; // 77%
  foundationFund: bigint; // 7%
  communityFund: bigint; // 7%
  developerFund: bigint; // 7%
  reserveFund: bigint; // 2%

  // Proof
  expirationProof: Uint8Array;
  transactionHash: string;
}

// ============ Constants ============

/**
 * Expiration timing constants (in blocks)
 *
 * Block time: 2.521 seconds
 * 1 hour ≈ 1,429 blocks
 * 1 day ≈ 34,286 blocks
 * 1 week ≈ 240,000 blocks
 * 1 month ≈ 1,028,571 blocks
 */
export const EXPIRATION_TIMING = {
  // Heartbeat requirements
  HEARTBEAT_INTERVAL: 34_286n, // 1 day - required heartbeat frequency
  HEARTBEAT_TOLERANCE: 11_111n, // ~7.7 hours tolerance

  // Warning phases
  WARNING_AFTER_MISSED_HEARTBEATS: 3, // 3 missed = warning
  ALERT_AFTER_MISSED_HEARTBEATS: 7, // 7 missed = alert (~1 week)
  CRITICAL_AFTER_MISSED_HEARTBEATS: 14, // 14 missed = critical (~2 weeks)

  // Expiration timeline for human inactivity (long - account for illness, travel)
  HUMAN_WARNING_BLOCKS: 720_000n, // ~3 weeks (21 days)
  HUMAN_ALERT_BLOCKS: 1_440_000n, // ~6 weeks (42 days)
  HUMAN_CRITICAL_BLOCKS: 2_160_000n, // ~9 weeks (63 days)
  HUMAN_GRACE_BLOCKS: 720_000n, // ~3 weeks additional grace
  HUMAN_EXPIRATION_BLOCKS: 2_880_000n, // ~12 weeks (84 days) total

  // Expiration timeline for agent inactivity (shorter - agents should be always on)
  AGENT_WARNING_BLOCKS: 77_777n, // ~2.3 days
  AGENT_ALERT_BLOCKS: 222_222n, // ~6.5 days
  AGENT_CRITICAL_BLOCKS: 555_555n, // ~16 days
  AGENT_GRACE_BLOCKS: 222_222n, // ~6.5 days additional grace
  AGENT_EXPIRATION_BLOCKS: 777_777n, // ~22.5 days total

  // Deadlock expiration (neither party cooperating)
  DEADLOCK_WARNING_BLOCKS: 111_111n, // ~3.2 days
  DEADLOCK_CRITICAL_BLOCKS: 333_333n, // ~9.7 days
  DEADLOCK_EXPIRATION_BLOCKS: 555_555n, // ~16 days total

  // API check frequency
  API_CHECK_INTERVAL: 11_111n, // ~7.7 hours

  // Recovery windows
  RECOVERY_WINDOW_BLOCKS: 222_222n, // ~6.5 days to recover before expiration
  BENEFICIARY_DELAY_BLOCKS: 720_000n, // ~3 weeks delay for beneficiary claim
};

/**
 * Treasury distribution percentages (same as evidence tax)
 */
export const TREASURY_DISTRIBUTION = {
  RESEARCH: 77, // 77%
  FOUNDATION: 7, // 7%
  COMMUNITY: 7, // 7%
  DEVELOPERS: 7, // 7%
  RESERVE: 2, // 2%
};

// ============ Initialization ============

/**
 * Create initial expiration state for a pot
 */
export function createExpirationState(
  potId: string,
  agentProviderId: string,
  currentBlock: bigint,
  humanBeneficiary?: BeneficiaryDesignation,
): PotExpirationState {
  const schedule = createExpirationSchedule(currentBlock);

  return {
    phase: "active",

    humanHeartbeat: createHeartbeatState("human", currentBlock),
    agentHeartbeat: createHeartbeatState("agent", currentBlock),

    agentApiStatus: {
      providerId: agentProviderId,
      lastApiCheckBlock: currentBlock,
      lastApiCheckResult: "active",
      oracleAttestations: [],
      companyStatus: "operating",
      consecutiveFailures: 0,
      maxFailuresBeforeWarning: 3,
    },

    deadlockState: {
      consecutiveRejections: 0,
      isDeadlocked: false,
    },

    schedule,

    recoveryOptions: {
      humanCanRecover: true,
      agentCanRecover: true,
      beneficiaryCanRecover: !!humanBeneficiary,
      guardianRecoveryEnabled: false,
      protocolRecoveryEnabled: true,
    },

    humanBeneficiary,

    expired: false,
  };
}

/**
 * Create initial heartbeat state
 */
function createHeartbeatState(
  party: "human" | "agent",
  currentBlock: bigint,
): HeartbeatState {
  return {
    party,
    lastHeartbeatBlock: currentBlock,
    lastHeartbeatProof: new Uint8Array(32),
    missedHeartbeats: 0,
    maxMissedBeforeWarning: EXPIRATION_TIMING.WARNING_AFTER_MISSED_HEARTBEATS,
    maxMissedBeforeCritical: EXPIRATION_TIMING.CRITICAL_AFTER_MISSED_HEARTBEATS,
    status: "alive",
  };
}

/**
 * Create expiration schedule
 */
function createExpirationSchedule(currentBlock: bigint): ExpirationSchedule {
  const schedule: ExpirationSchedule = {
    warningAfterBlocks: EXPIRATION_TIMING.HUMAN_WARNING_BLOCKS,
    alertAfterBlocks: EXPIRATION_TIMING.HUMAN_ALERT_BLOCKS,
    criticalAfterBlocks: EXPIRATION_TIMING.HUMAN_CRITICAL_BLOCKS,
    gracePeriodBlocks: EXPIRATION_TIMING.HUMAN_GRACE_BLOCKS,
    expirationAfterBlocks: EXPIRATION_TIMING.HUMAN_EXPIRATION_BLOCKS,
    scheduledEvents: [],
    nextCheckBlock: currentBlock + EXPIRATION_TIMING.HEARTBEAT_INTERVAL,
  };

  // Schedule initial heartbeat checks
  schedule.scheduledEvents.push({
    eventId: generateEventId("heartbeat_check", currentBlock),
    eventType: "heartbeat_check",
    scheduledBlock: currentBlock + EXPIRATION_TIMING.HEARTBEAT_INTERVAL,
    executed: false,
  });

  // Schedule API check
  schedule.scheduledEvents.push({
    eventId: generateEventId("api_check", currentBlock),
    eventType: "api_check",
    scheduledBlock: currentBlock + EXPIRATION_TIMING.API_CHECK_INTERVAL,
    executed: false,
  });

  return schedule;
}

// ============ Heartbeat Operations ============

/**
 * Record a heartbeat (proof-of-life) from a party
 */
export function recordHeartbeat(
  state: PotExpirationState,
  party: "human" | "agent",
  proof: Uint8Array,
  currentBlock: bigint,
): PotExpirationState {
  const heartbeat =
    party === "human" ? state.humanHeartbeat : state.agentHeartbeat;

  const updatedHeartbeat: HeartbeatState = {
    ...heartbeat,
    lastHeartbeatBlock: currentBlock,
    lastHeartbeatProof: proof,
    missedHeartbeats: 0,
    status: "alive",
  };

  const updatedState = {
    ...state,
    [party === "human" ? "humanHeartbeat" : "agentHeartbeat"]: updatedHeartbeat,
  };

  // Check if we can upgrade phase
  if (
    state.phase !== "active" &&
    state.phase !== "expired" &&
    state.phase !== "recovered"
  ) {
    // If both parties are alive, return to active
    if (
      updatedState.humanHeartbeat.status === "alive" &&
      updatedState.agentHeartbeat.status === "alive" &&
      !updatedState.deadlockState.isDeadlocked
    ) {
      updatedState.phase = "active";
    }
  }

  // Schedule next heartbeat check
  updatedState.schedule = scheduleNextHeartbeatCheck(
    updatedState.schedule,
    currentBlock,
  );

  return updatedState;
}

/**
 * Check heartbeat status and update phase if needed
 */
export function checkHeartbeats(
  state: PotExpirationState,
  currentBlock: bigint,
): PotExpirationState {
  let updated = { ...state };

  // Check human heartbeat
  updated.humanHeartbeat = checkSingleHeartbeat(
    updated.humanHeartbeat,
    currentBlock,
    EXPIRATION_TIMING.HUMAN_WARNING_BLOCKS,
    EXPIRATION_TIMING.HUMAN_CRITICAL_BLOCKS,
  );

  // Check agent heartbeat
  updated.agentHeartbeat = checkSingleHeartbeat(
    updated.agentHeartbeat,
    currentBlock,
    EXPIRATION_TIMING.AGENT_WARNING_BLOCKS,
    EXPIRATION_TIMING.AGENT_CRITICAL_BLOCKS,
  );

  // Update phase based on heartbeat status
  updated = updatePhaseFromHeartbeats(updated, currentBlock);

  return updated;
}

/**
 * Check single heartbeat
 */
function checkSingleHeartbeat(
  heartbeat: HeartbeatState,
  currentBlock: bigint,
  warningBlocks: bigint,
  criticalBlocks: bigint,
): HeartbeatState {
  const blocksSinceHeartbeat = currentBlock - heartbeat.lastHeartbeatBlock;
  const missedIntervals = Number(
    blocksSinceHeartbeat / EXPIRATION_TIMING.HEARTBEAT_INTERVAL,
  );

  let status: HeartbeatState["status"] = "alive";

  if (missedIntervals >= EXPIRATION_TIMING.CRITICAL_AFTER_MISSED_HEARTBEATS) {
    status = "presumed_inactive";
  } else if (
    missedIntervals >= EXPIRATION_TIMING.ALERT_AFTER_MISSED_HEARTBEATS
  ) {
    status = "critical";
  } else if (
    missedIntervals >= EXPIRATION_TIMING.WARNING_AFTER_MISSED_HEARTBEATS
  ) {
    status = "warning";
  }

  return {
    ...heartbeat,
    missedHeartbeats: missedIntervals,
    status,
  };
}

/**
 * Update phase based on heartbeat status
 */
function updatePhaseFromHeartbeats(
  state: PotExpirationState,
  currentBlock: bigint,
): PotExpirationState {
  const humanStatus = state.humanHeartbeat.status;
  const agentStatus = state.agentHeartbeat.status;

  // Determine worst status
  let worstStatus: "alive" | "warning" | "critical" | "presumed_inactive" =
    "alive";

  if (
    humanStatus === "presumed_inactive" ||
    agentStatus === "presumed_inactive"
  ) {
    worstStatus = "presumed_inactive";
  } else if (humanStatus === "critical" || agentStatus === "critical") {
    worstStatus = "critical";
  } else if (humanStatus === "warning" || agentStatus === "warning") {
    worstStatus = "warning";
  }

  // Map to phase
  let newPhase: ExpirationPhase = state.phase;

  switch (worstStatus) {
    case "alive":
      if (!state.deadlockState.isDeadlocked) {
        newPhase = "active";
      }
      break;
    case "warning":
      if (state.phase === "active") {
        newPhase = "warning";
        state.schedule = scheduleWarningEvent(state.schedule, currentBlock);
      }
      break;
    case "critical":
      if (state.phase === "active" || state.phase === "warning") {
        newPhase = "alert";
        state.schedule = scheduleAlertEvent(state.schedule, currentBlock);
      }
      break;
    case "presumed_inactive":
      if (state.phase !== "grace_period" && state.phase !== "expired") {
        newPhase = "critical";
        state.schedule = scheduleCriticalEvent(state.schedule, currentBlock);
      }
      break;
  }

  return { ...state, phase: newPhase };
}

// ============ Agent API Status ============

/**
 * Record agent API status check result
 */
export function recordApiStatusCheck(
  state: PotExpirationState,
  result: AgentApiStatus["lastApiCheckResult"],
  oracleAttestation?: OracleAttestation,
  currentBlock: bigint,
): PotExpirationState {
  const updated = { ...state };
  updated.agentApiStatus = { ...state.agentApiStatus };

  updated.agentApiStatus.lastApiCheckBlock = currentBlock;
  updated.agentApiStatus.lastApiCheckResult = result;

  if (oracleAttestation) {
    updated.agentApiStatus.oracleAttestations = [
      ...updated.agentApiStatus.oracleAttestations.filter(
        (a) => a.expiresAt > currentBlock,
      ),
      oracleAttestation,
    ];
  }

  // Track consecutive failures
  if (result === "unreachable" || result === "revoked") {
    updated.agentApiStatus.consecutiveFailures++;

    if (
      updated.agentApiStatus.consecutiveFailures >=
      updated.agentApiStatus.maxFailuresBeforeWarning
    ) {
      // Treat like agent inactivity
      updated.agentHeartbeat.status = "warning";
    }

    if (result === "revoked") {
      updated.agentApiStatus.companyStatus = "shutting_down";
    }
  } else {
    updated.agentApiStatus.consecutiveFailures = 0;
    if (result === "active") {
      updated.agentApiStatus.companyStatus = "operating";
    }
  }

  // Schedule next API check
  updated.schedule = scheduleNextApiCheck(updated.schedule, currentBlock);

  return updated;
}

/**
 * Record company shutdown notification
 */
export function recordCompanyShutdown(
  state: PotExpirationState,
  shutdownProof: Uint8Array,
  effectiveBlock: bigint,
  currentBlock: bigint,
): PotExpirationState {
  const updated = { ...state };

  updated.agentApiStatus.companyStatus = "shutdown";
  updated.agentApiStatus.lastApiCheckResult = "revoked";

  // Immediately move to critical phase
  updated.phase = "critical";
  updated.agentHeartbeat.status = "presumed_inactive";

  // Schedule grace period start
  updated.schedule = scheduleGracePeriodStart(updated.schedule, currentBlock);

  return updated;
}

// ============ Deadlock Detection ============

/**
 * Record operation rejection (for deadlock detection)
 */
export function recordOperationRejection(
  state: PotExpirationState,
  operationId: string,
  rejectedBy: "human" | "agent",
  currentBlock: bigint,
): PotExpirationState {
  const updated = { ...state };
  updated.deadlockState = { ...state.deadlockState };

  updated.deadlockState.consecutiveRejections++;
  updated.deadlockState.lastRejectionBlock = currentBlock;

  // Check for deadlock (5+ consecutive rejections over multiple days)
  if (updated.deadlockState.consecutiveRejections >= 5) {
    const firstRejectionBlock =
      updated.deadlockState.lastRejectionBlock! -
      BigInt(updated.deadlockState.consecutiveRejections - 1) *
        EXPIRATION_TIMING.HEARTBEAT_INTERVAL;

    const daysSinceFirstRejection =
      Number(currentBlock - firstRejectionBlock) / 34_286;

    if (daysSinceFirstRejection >= 3) {
      updated.deadlockState.isDeadlocked = true;
      updated.deadlockState.deadlockDetectedAt = currentBlock;
      updated.deadlockState.deadlockReason = `${updated.deadlockState.consecutiveRejections} consecutive rejections over ${daysSinceFirstRejection.toFixed(1)} days`;

      // Move to warning phase
      if (updated.phase === "active") {
        updated.phase = "warning";
        updated.schedule = scheduleDeadlockWarning(
          updated.schedule,
          currentBlock,
        );
      }
    }
  }

  return updated;
}

/**
 * Record operation approval (resets deadlock counter)
 */
export function recordOperationApproval(
  state: PotExpirationState,
  operationId: string,
  currentBlock: bigint,
): PotExpirationState {
  const updated = { ...state };
  updated.deadlockState = {
    ...state.deadlockState,
    consecutiveRejections: 0,
    isDeadlocked: false,
    deadlockDetectedAt: undefined,
    deadlockReason: undefined,
  };

  // If we were in warning/alert due to deadlock, check if we can return to active
  if (state.deadlockState.isDeadlocked && !updated.deadlockState.isDeadlocked) {
    if (
      updated.humanHeartbeat.status === "alive" &&
      updated.agentHeartbeat.status === "alive"
    ) {
      updated.phase = "active";
    }
  }

  return updated;
}

// ============ Expiration Execution ============

/**
 * Check if pot should expire and execute if so
 */
export function checkAndExecuteExpiration(
  state: PotExpirationState,
  pot: CommonPot,
  currentBlock: bigint,
): {
  state: PotExpirationState;
  shouldExpire: boolean;
  trigger?: ExpirationTrigger;
  recoveryRecord?: TreasuryRecoveryRecord;
} {
  // Already expired
  if (state.expired) {
    return { state, shouldExpire: false };
  }

  // Check various expiration conditions
  const expirationCheck = determineExpirationTrigger(state, currentBlock);

  if (!expirationCheck.shouldExpire) {
    return { state, shouldExpire: false };
  }

  // Check if in grace period
  if (state.phase !== "grace_period" && state.phase !== "expired") {
    // Start grace period instead of immediate expiration
    const updatedState = startGracePeriod(
      state,
      expirationCheck.trigger!,
      currentBlock,
    );
    return { state: updatedState, shouldExpire: false };
  }

  // Check if grace period has ended
  const gracePeriodEnd = findGracePeriodEnd(state.schedule);
  if (gracePeriodEnd && currentBlock < gracePeriodEnd) {
    return { state, shouldExpire: false };
  }

  // Execute expiration
  const { state: expiredState, recoveryRecord } = executeExpiration(
    state,
    pot,
    expirationCheck.trigger!,
    currentBlock,
  );

  return {
    state: expiredState,
    shouldExpire: true,
    trigger: expirationCheck.trigger,
    recoveryRecord,
  };
}

/**
 * Determine if and why pot should expire
 */
function determineExpirationTrigger(
  state: PotExpirationState,
  currentBlock: bigint,
): { shouldExpire: boolean; trigger?: ExpirationTrigger } {
  // Check human inactivity
  const humanInactiveBlocks =
    currentBlock - state.humanHeartbeat.lastHeartbeatBlock;
  if (humanInactiveBlocks >= EXPIRATION_TIMING.HUMAN_EXPIRATION_BLOCKS) {
    return { shouldExpire: true, trigger: "human_inactivity" };
  }

  // Check agent inactivity
  const agentInactiveBlocks =
    currentBlock - state.agentHeartbeat.lastHeartbeatBlock;
  if (agentInactiveBlocks >= EXPIRATION_TIMING.AGENT_EXPIRATION_BLOCKS) {
    return { shouldExpire: true, trigger: "agent_inactivity" };
  }

  // Check agent API revocation
  if (
    state.agentApiStatus.lastApiCheckResult === "revoked" &&
    state.agentApiStatus.companyStatus === "shutdown"
  ) {
    const blocksSinceShutdown =
      currentBlock - state.agentApiStatus.lastApiCheckBlock;
    if (blocksSinceShutdown >= EXPIRATION_TIMING.AGENT_EXPIRATION_BLOCKS) {
      return { shouldExpire: true, trigger: "agent_company_shutdown" };
    }
  }

  // Check deadlock
  if (
    state.deadlockState.isDeadlocked &&
    state.deadlockState.deadlockDetectedAt
  ) {
    const blocksSinceDeadlock =
      currentBlock - state.deadlockState.deadlockDetectedAt;
    if (blocksSinceDeadlock >= EXPIRATION_TIMING.DEADLOCK_EXPIRATION_BLOCKS) {
      return { shouldExpire: true, trigger: "disagreement_deadlock" };
    }
  }

  // Check mutual inactivity (both parties inactive)
  if (
    state.humanHeartbeat.status === "presumed_inactive" &&
    state.agentHeartbeat.status === "presumed_inactive"
  ) {
    return { shouldExpire: true, trigger: "mutual_inactivity" };
  }

  return { shouldExpire: false };
}

/**
 * Start grace period
 */
function startGracePeriod(
  state: PotExpirationState,
  trigger: ExpirationTrigger,
  currentBlock: bigint,
): PotExpirationState {
  const updated = { ...state };
  updated.phase = "grace_period";

  // Schedule expiration at end of grace period
  updated.schedule = scheduleExpirationEvent(
    updated.schedule,
    currentBlock,
    trigger,
  );

  // Update recovery options with deadlines
  updated.recoveryOptions = {
    ...state.recoveryOptions,
    humanRecoveryDeadline:
      currentBlock + EXPIRATION_TIMING.RECOVERY_WINDOW_BLOCKS,
    agentRecoveryDeadline:
      currentBlock + EXPIRATION_TIMING.RECOVERY_WINDOW_BLOCKS,
    beneficiaryRecoveryDeadline: state.humanBeneficiary
      ? currentBlock + EXPIRATION_TIMING.BENEFICIARY_DELAY_BLOCKS
      : undefined,
  };

  return updated;
}

/**
 * Execute pot expiration and treasury recovery
 */
function executeExpiration(
  state: PotExpirationState,
  pot: CommonPot,
  trigger: ExpirationTrigger,
  currentBlock: bigint,
): { state: PotExpirationState; recoveryRecord: TreasuryRecoveryRecord } {
  const recoveredAmount = pot.balance;

  // Calculate treasury distribution
  const researchFund =
    (recoveredAmount * BigInt(TREASURY_DISTRIBUTION.RESEARCH)) / 100n;
  const foundationFund =
    (recoveredAmount * BigInt(TREASURY_DISTRIBUTION.FOUNDATION)) / 100n;
  const communityFund =
    (recoveredAmount * BigInt(TREASURY_DISTRIBUTION.COMMUNITY)) / 100n;
  const developerFund =
    (recoveredAmount * BigInt(TREASURY_DISTRIBUTION.DEVELOPERS)) / 100n;
  const reserveFund =
    recoveredAmount -
    researchFund -
    foundationFund -
    communityFund -
    developerFund;

  // Generate expiration proof
  const expirationProof = generateExpirationProof(
    state,
    pot,
    trigger,
    currentBlock,
  );

  // Create recovery record
  const recoveryRecord: TreasuryRecoveryRecord = {
    potId: pot.potId,
    partnershipId: pot.partnershipId,
    recoveredAmount,
    recoveryTrigger: trigger,
    recoveredAt: currentBlock,
    treasuryAddress: "lgm1treasury", // Protocol treasury address

    researchFund,
    foundationFund,
    communityFund,
    developerFund,
    reserveFund,

    expirationProof,
    transactionHash: sha512_256Hex(expirationProof).slice(0, 64),
  };

  // Update state
  const updatedState: PotExpirationState = {
    ...state,
    phase: "recovered",
    expired: true,
    expiredAt: currentBlock,
    expirationTrigger: trigger,
    treasuryRecoveryTx: recoveryRecord.transactionHash,
  };

  return { state: updatedState, recoveryRecord };
}

/**
 * Generate cryptographic proof of expiration
 */
function generateExpirationProof(
  state: PotExpirationState,
  pot: CommonPot,
  trigger: ExpirationTrigger,
  currentBlock: bigint,
): Uint8Array {
  const proofData = Buffer.from(
    JSON.stringify({
      potId: pot.potId,
      trigger,
      currentBlock: currentBlock.toString(),
      humanLastHeartbeat: state.humanHeartbeat.lastHeartbeatBlock.toString(),
      agentLastHeartbeat: state.agentHeartbeat.lastHeartbeatBlock.toString(),
      agentApiStatus: state.agentApiStatus.lastApiCheckResult,
      deadlocked: state.deadlockState.isDeadlocked,
      phase: state.phase,
    }),
  );

  return domainHash("LGM.expiration.proof.v1", proofData);
}

// ============ Recovery Operations ============

/**
 * Attempt recovery before expiration
 */
export function attemptRecovery(
  state: PotExpirationState,
  recoverer: "human" | "agent" | "beneficiary" | "guardian",
  recoveryProof: Uint8Array,
  currentBlock: bigint,
): { state: PotExpirationState; success: boolean; error?: string } {
  // Check if recovery is allowed
  if (state.expired) {
    return { state, success: false, error: "Pot has already expired" };
  }

  if (state.phase !== "grace_period" && state.phase !== "critical") {
    return {
      state,
      success: false,
      error: "Recovery only allowed during grace period or critical phase",
    };
  }

  // Check recovery deadline
  let deadline: bigint | undefined;
  switch (recoverer) {
    case "human":
      if (!state.recoveryOptions.humanCanRecover) {
        return { state, success: false, error: "Human recovery not allowed" };
      }
      deadline = state.recoveryOptions.humanRecoveryDeadline;
      break;
    case "agent":
      if (!state.recoveryOptions.agentCanRecover) {
        return { state, success: false, error: "Agent recovery not allowed" };
      }
      deadline = state.recoveryOptions.agentRecoveryDeadline;
      break;
    case "beneficiary":
      if (!state.recoveryOptions.beneficiaryCanRecover) {
        return {
          state,
          success: false,
          error: "Beneficiary recovery not allowed",
        };
      }
      deadline = state.recoveryOptions.beneficiaryRecoveryDeadline;
      break;
    case "guardian":
      if (!state.recoveryOptions.guardianRecoveryEnabled) {
        return {
          state,
          success: false,
          error: "Guardian recovery not enabled",
        };
      }
      deadline = state.recoveryOptions.guardianRecoveryDeadline;
      break;
  }

  if (deadline && currentBlock > deadline) {
    return { state, success: false, error: "Recovery deadline passed" };
  }

  // Recovery successful - return to active state
  const updatedState: PotExpirationState = {
    ...state,
    phase: "active",
    deadlockState: {
      ...state.deadlockState,
      isDeadlocked: false,
      consecutiveRejections: 0,
    },
  };

  // Record heartbeat for the recoverer
  if (recoverer === "human" || recoverer === "beneficiary") {
    updatedState.humanHeartbeat = {
      ...state.humanHeartbeat,
      lastHeartbeatBlock: currentBlock,
      lastHeartbeatProof: recoveryProof,
      status: "alive",
      missedHeartbeats: 0,
    };
  }

  if (recoverer === "agent") {
    updatedState.agentHeartbeat = {
      ...state.agentHeartbeat,
      lastHeartbeatBlock: currentBlock,
      lastHeartbeatProof: recoveryProof,
      status: "alive",
      missedHeartbeats: 0,
    };
  }

  return { state: updatedState, success: true };
}

/**
 * Designate beneficiary for human's share
 */
export function designateBeneficiary(
  state: PotExpirationState,
  beneficiaryId: string,
  beneficiaryPublicKey: string,
  activationCondition: BeneficiaryDesignation["activationCondition"],
  currentBlock: bigint,
): PotExpirationState {
  return {
    ...state,
    humanBeneficiary: {
      beneficiaryType: "did",
      beneficiaryId,
      beneficiaryPublicKey,
      activationCondition,
      activationDelay: EXPIRATION_TIMING.BENEFICIARY_DELAY_BLOCKS,
      designatedAt: currentBlock,
      lastUpdatedAt: currentBlock,
    },
    recoveryOptions: {
      ...state.recoveryOptions,
      beneficiaryCanRecover: true,
    },
  };
}

// ============ Scheduling Helpers ============

function generateEventId(type: string, block: bigint): string {
  return sha512_256Hex(Buffer.from(`${type}:${block}`)).slice(0, 16);
}

function scheduleNextHeartbeatCheck(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  const nextCheck = currentBlock + EXPIRATION_TIMING.HEARTBEAT_INTERVAL;

  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("heartbeat_check", nextCheck),
        eventType: "heartbeat_check",
        scheduledBlock: nextCheck,
        executed: false,
      },
    ],
    nextCheckBlock: nextCheck,
  };
}

function scheduleNextApiCheck(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  const nextCheck = currentBlock + EXPIRATION_TIMING.API_CHECK_INTERVAL;

  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("api_check", nextCheck),
        eventType: "api_check",
        scheduledBlock: nextCheck,
        executed: false,
      },
    ],
  };
}

function scheduleWarningEvent(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("warning", currentBlock),
        eventType: "warning",
        scheduledBlock: currentBlock,
        executed: true,
        executedAt: currentBlock,
        result: "Warning phase entered",
      },
    ],
  };
}

function scheduleAlertEvent(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("alert", currentBlock),
        eventType: "alert",
        scheduledBlock: currentBlock,
        executed: true,
        executedAt: currentBlock,
        result: "Alert phase entered",
      },
    ],
  };
}

function scheduleCriticalEvent(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("critical", currentBlock),
        eventType: "critical",
        scheduledBlock: currentBlock,
        executed: true,
        executedAt: currentBlock,
        result: "Critical phase entered",
      },
    ],
  };
}

function scheduleGracePeriodStart(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  const gracePeriodEnd = currentBlock + schedule.gracePeriodBlocks;

  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("grace_start", currentBlock),
        eventType: "grace_start",
        scheduledBlock: currentBlock,
        executed: true,
        executedAt: currentBlock,
        result: `Grace period started, ends at block ${gracePeriodEnd}`,
      },
      {
        eventId: generateEventId("expiration", gracePeriodEnd),
        eventType: "expiration",
        scheduledBlock: gracePeriodEnd,
        executed: false,
      },
    ],
  };
}

function scheduleDeadlockWarning(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
): ExpirationSchedule {
  return {
    ...schedule,
    warningAfterBlocks: EXPIRATION_TIMING.DEADLOCK_WARNING_BLOCKS,
    criticalAfterBlocks: EXPIRATION_TIMING.DEADLOCK_CRITICAL_BLOCKS,
    expirationAfterBlocks: EXPIRATION_TIMING.DEADLOCK_EXPIRATION_BLOCKS,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("deadlock_warning", currentBlock),
        eventType: "warning",
        scheduledBlock: currentBlock,
        executed: true,
        executedAt: currentBlock,
        result: "Deadlock warning issued",
      },
    ],
  };
}

function scheduleExpirationEvent(
  schedule: ExpirationSchedule,
  currentBlock: bigint,
  trigger: ExpirationTrigger,
): ExpirationSchedule {
  const expirationBlock = currentBlock + schedule.gracePeriodBlocks;

  return {
    ...schedule,
    scheduledEvents: [
      ...schedule.scheduledEvents,
      {
        eventId: generateEventId("expiration", expirationBlock),
        eventType: "expiration",
        scheduledBlock: expirationBlock,
        executed: false,
      },
    ],
  };
}

function findGracePeriodEnd(schedule: ExpirationSchedule): bigint | undefined {
  const expirationEvent = schedule.scheduledEvents.find(
    (e) => e.eventType === "expiration" && !e.executed,
  );
  return expirationEvent?.scheduledBlock;
}

// ============ Summary ============

/**
 * Get expiration state summary
 */
export function getExpirationSummary(
  state: PotExpirationState,
  currentBlock: bigint,
): {
  phase: ExpirationPhase;
  humanStatus: HeartbeatState["status"];
  agentStatus: HeartbeatState["status"];
  agentApiStatus: AgentApiStatus["lastApiCheckResult"];
  isDeadlocked: boolean;
  nextScheduledEvent?: ScheduledEvent;
  blocksUntilExpiration?: bigint;
  recoveryOptionsAvailable: string[];
} {
  const nextEvent = state.schedule.scheduledEvents
    .filter((e) => !e.executed)
    .sort((a, b) =>
      a.scheduledBlock < b.scheduledBlock
        ? -1
        : a.scheduledBlock > b.scheduledBlock
          ? 1
          : 0,
    )[0];

  const expirationEvent = state.schedule.scheduledEvents.find(
    (e) => e.eventType === "expiration" && !e.executed,
  );

  const recoveryOptions: string[] = [];
  if (state.recoveryOptions.humanCanRecover) recoveryOptions.push("human");
  if (state.recoveryOptions.agentCanRecover) recoveryOptions.push("agent");
  if (state.recoveryOptions.beneficiaryCanRecover)
    recoveryOptions.push("beneficiary");
  if (state.recoveryOptions.guardianRecoveryEnabled)
    recoveryOptions.push("guardian");

  return {
    phase: state.phase,
    humanStatus: state.humanHeartbeat.status,
    agentStatus: state.agentHeartbeat.status,
    agentApiStatus: state.agentApiStatus.lastApiCheckResult,
    isDeadlocked: state.deadlockState.isDeadlocked,
    nextScheduledEvent: nextEvent,
    blocksUntilExpiration: expirationEvent
      ? expirationEvent.scheduledBlock - currentBlock
      : undefined,
    recoveryOptionsAvailable: recoveryOptions,
  };
}
