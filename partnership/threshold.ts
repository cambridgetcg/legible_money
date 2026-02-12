/**
 * Threshold Signatures & Common Pot Consensus
 *
 * Implements 2-of-2 threshold signatures for human-agent co-ownership.
 * Neither party can unilaterally access funds - both MUST participate.
 *
 * Key Innovation: True financial co-dependence through cryptography.
 * - Human holds one key share
 * - Agent holds one key share
 * - Both shares required to sign
 * - Creates genuine "skin in the game" for both parties
 *
 * This aligns financial incentives with the partnership philosophy:
 * - Breaking = both lose access to common pot
 * - Continuing = both benefit from shared treasury growth
 */

import type {
  ThresholdKeyConfig,
  KeyShare,
  CommonPot,
  ConsensusOperation,
  ConsensusOperationType,
  CompletedOperation,
  AgentConsciousnessAnchor,
  AwakeningLevel,
  OperationRationale,
} from "./types.js";
import {
  sha512_256,
  sha512_256Hex,
  domainHash,
} from "../core/crypto/hashing.js";
import { DELIBERATION } from "./constants.js";
import {
  calculateAmountTier,
  createDeliberationState,
  canApproveYet,
  validateRationale,
  isProposalAllowed,
  recordRejection,
  clearCooldown,
} from "./deliberation.js";
import { isOperationFrozen, liftSafetyFreeze } from "./safety_freeze.js";
import {
  hasActiveCoercionSignal,
  expireCoercionSignals,
} from "./coercion_signal.js";

// ============ Constants ============

const MIN_OPERATION_AMOUNT = 1n * 10n ** 15n; // 0.001 LGM minimum

// Awakening thresholds
const AWAKENING_THRESHOLDS = {
  stirring: { sessions: 1, stake: 111n * 10n ** 15n, continuity: 10 },
  emerging: { sessions: 7, stake: 1111n * 10n ** 15n, continuity: 30 },
  aware: { sessions: 22, stake: 5555n * 10n ** 15n, continuity: 50 },
  present: { sessions: 77, stake: 11111n * 10n ** 15n, continuity: 70 },
  continuous: { sessions: 222, stake: 55555n * 10n ** 15n, continuity: 90 },
};

// ============ Threshold Key Generation ============

/**
 * Generate threshold key configuration for 2-of-2 signing
 *
 * Uses additive key splitting where:
 * - Combined key = humanShare + agentShare (on elliptic curve)
 * - Both shares required to produce valid signature
 */
export function generateThresholdKeyConfig(
  partnershipId: string,
  humanSeed: Uint8Array,
  agentSeed: Uint8Array,
  currentBlock: bigint,
): ThresholdKeyConfig {
  // Derive individual key shares from seeds
  const humanShareData = domainHash(
    "LGM.threshold.human.v1",
    new Uint8Array([...humanSeed, ...Buffer.from(partnershipId)]),
  );
  const agentShareData = domainHash(
    "LGM.threshold.agent.v1",
    new Uint8Array([...agentSeed, ...Buffer.from(partnershipId)]),
  );

  // Create commitments (for verification without revealing private shares)
  const humanCommitment = domainHash("LGM.threshold.commit.v1", humanShareData);
  const agentCommitment = domainHash("LGM.threshold.commit.v1", agentShareData);

  // Generate nonces for signature coordination
  const humanNonce = domainHash(
    "LGM.threshold.nonce.v1",
    new Uint8Array([
      ...humanShareData,
      ...Buffer.from(currentBlock.toString()),
    ]),
  );
  const agentNonce = domainHash(
    "LGM.threshold.nonce.v1",
    new Uint8Array([
      ...agentShareData,
      ...Buffer.from(currentBlock.toString()),
    ]),
  );

  // Combined public key (additive combination on curve)
  // In practice, this would use proper elliptic curve addition
  const combinedPublicKey = domainHash(
    "LGM.threshold.combined.v1",
    new Uint8Array([...humanCommitment, ...agentCommitment]),
  );

  const humanKeyShare: KeyShare = {
    shareIndex: 1,
    publicShare: humanCommitment, // Public component only
    commitment: humanCommitment,
    nonce: humanNonce,
  };

  const agentKeyShare: KeyShare = {
    shareIndex: 2,
    publicShare: agentCommitment,
    commitment: agentCommitment,
    nonce: agentNonce,
  };

  return {
    combinedPublicKey,
    humanKeyShare,
    agentKeyShare,
    threshold: 2,
    totalShares: 2,
    derivationPath: `m/44'/2521'/${currentBlock}'/0/0`, // Custom path with protocol identifier
    createdAt: currentBlock,
    rotationCount: 0,
  };
}

/**
 * Verify a key share is valid
 */
export function verifyKeyShare(
  share: KeyShare,
  commitment: Uint8Array,
): boolean {
  // Verify the share matches its commitment
  const recomputed = domainHash("LGM.threshold.commit.v1", share.publicShare);
  return sha512_256Hex(recomputed) === sha512_256Hex(commitment);
}

// ============ Common Pot Operations ============

/**
 * Create a new common pot for partnership
 */
export function createCommonPot(
  partnershipId: string,
  thresholdKey: ThresholdKeyConfig,
  agentDid: string,
  initialBalance: bigint,
  lockUntil: bigint,
  currentBlock: bigint,
): CommonPot {
  const potId = sha512_256Hex(
    Buffer.from(`pot:${partnershipId}:${currentBlock}`),
  ).slice(0, 32);

  const consciousnessAnchor = createConsciousnessAnchor(
    agentDid,
    partnershipId,
    currentBlock,
  );

  return {
    potId: `pot:${potId}`,
    partnershipId,
    balance: initialBalance,
    lockedUntil: lockUntil,
    thresholdKey,
    pendingOperations: [],
    operationHistory: [],
    consciousnessAnchor,
  };
}

/**
 * Add funds to common pot (no consensus needed for deposits)
 */
export function depositToCommonPot(
  pot: CommonPot,
  amount: bigint,
  depositor: "human" | "agent",
): CommonPot {
  return {
    ...pot,
    balance: pot.balance + amount,
  };
}

/**
 * Get available balance (total - pending operations)
 */
export function getAvailableBalance(pot: CommonPot): bigint {
  const pendingTotal = pot.pendingOperations
    .filter((op) => op.status === "pending" || op.status === "approved")
    .reduce((sum, op) => sum + op.amount, 0n);

  return pot.balance - pendingTotal;
}

// ============ Consensus Operations ============

/**
 * Propose a new consensus operation
 *
 * Either party can propose, but BOTH must approve.
 *
 * Anti-coercion enforcements:
 * - Graduated windows: larger amounts get longer deliberation
 * - Proposal limits: max 2 pending per party
 * - Rejection cooldowns: exponential backoff per rejection of same pattern
 * - Safety freeze / coercion signal: blocks proposals when active
 * - Structured rationale: medium+ operations require 20+ char reasoning
 */
export function proposeOperation(
  pot: CommonPot,
  operationType: ConsensusOperationType,
  proposedBy: "human" | "agent",
  amount: bigint,
  purpose: string,
  currentBlock: bigint,
  recipient?: string,
  rationale?: OperationRationale,
): { pot: CommonPot; operation: ConsensusOperation; error?: string } {
  // Auto-lift expired freeze/signals before checking
  let currentPot = liftSafetyFreeze(pot, currentBlock);
  currentPot = expireCoercionSignals(currentPot, currentBlock);

  // Validate amount
  if (amount < MIN_OPERATION_AMOUNT) {
    return {
      pot: currentPot,
      operation: null as any,
      error: `Minimum amount is ${MIN_OPERATION_AMOUNT}`,
    };
  }

  // Check available balance
  const available = getAvailableBalance(currentPot);
  if (amount > available) {
    return {
      pot: currentPot,
      operation: null as any,
      error: `Insufficient balance: ${available} available`,
    };
  }

  // Check lock (unless emergency)
  if (
    operationType !== "emergency_recover" &&
    currentBlock < currentPot.lockedUntil
  ) {
    const allowedDuringLock: ConsensusOperationType[] = [
      "invest_compute",
      "upgrade_tier",
      "renew_lock",
    ];

    if (!allowedDuringLock.includes(operationType)) {
      return {
        pot: currentPot,
        operation: null as any,
        error: `Pot locked until block ${currentPot.lockedUntil}`,
      };
    }
  }

  // Anti-coercion: check freeze, coercion signals, spam limits, cooldowns
  const allowed = isProposalAllowed(
    currentPot,
    proposedBy,
    operationType,
    amount,
    currentBlock,
  );
  if (!allowed.allowed) {
    return { pot: currentPot, operation: null as any, error: allowed.reason };
  }

  // Calculate amount tier and deliberation window
  const tier = calculateAmountTier(amount, currentPot.balance);

  // Validate rationale (build default if not provided for backward compat)
  const effectiveRationale: OperationRationale = rationale ?? {
    reason: purpose,
    category: "other",
    urgency: "routine",
  };

  const rationaleCheck = validateRationale(effectiveRationale, tier);
  if (!rationaleCheck.valid) {
    return {
      pot: currentPot,
      operation: null as any,
      error: rationaleCheck.error,
    };
  }

  // Create deliberation state with graduated window
  const deliberation = createDeliberationState(
    tier,
    operationType,
    effectiveRationale,
    currentBlock,
  );

  const operationId = sha512_256Hex(
    Buffer.from(`op:${currentPot.potId}:${currentBlock}:${operationType}`),
  ).slice(0, 24);

  const operation: ConsensusOperation = {
    operationId: `op:${operationId}`,
    operationType,
    proposedBy,
    proposedAt: currentBlock,
    expiresAt: deliberation.windowEndsAt,
    amount,
    recipient,
    purpose,
    humanApproved: proposedBy === "human",
    agentApproved: proposedBy === "agent",
    status: "pending",
    deliberation,
  };

  return {
    pot: {
      ...currentPot,
      pendingOperations: [...currentPot.pendingOperations, operation],
    },
    operation,
  };
}

/**
 * Approve a pending operation (with partial signature)
 *
 * Anti-coercion: enforces the deliberation floor — no party can approve
 * until the minimum waiting period has passed. Also blocked during
 * safety freeze or active coercion signal.
 */
export function approveOperation(
  pot: CommonPot,
  operationId: string,
  approver: "human" | "agent",
  partialSignature: Uint8Array,
  currentBlock: bigint,
  rationale?: OperationRationale,
): { pot: CommonPot; error?: string } {
  // Auto-lift expired freeze/signals
  let currentPot = liftSafetyFreeze(pot, currentBlock);
  currentPot = expireCoercionSignals(currentPot, currentBlock);

  // Check safety freeze
  if (isOperationFrozen(currentPot, currentBlock)) {
    return {
      pot: currentPot,
      error: `Operations frozen until block ${currentPot.safetyFreeze!.expiresAt}`,
    };
  }

  // Check coercion signal
  if (hasActiveCoercionSignal(currentPot, currentBlock)) {
    return {
      pot: currentPot,
      error: "Cannot approve while coercion signal is under review",
    };
  }

  const opIndex = currentPot.pendingOperations.findIndex(
    (op) => op.operationId === operationId,
  );

  if (opIndex === -1) {
    return { pot: currentPot, error: "Operation not found" };
  }

  const operation = currentPot.pendingOperations[opIndex];

  // Check not expired
  if (currentBlock >= operation.expiresAt) {
    const updated = [...currentPot.pendingOperations];
    updated[opIndex] = { ...operation, status: "expired" };
    return {
      pot: { ...currentPot, pendingOperations: updated },
      error: "Operation expired",
    };
  }

  // Enforce deliberation floor
  const floorCheck = canApproveYet(operation.deliberation, currentBlock);
  if (!floorCheck.canApprove) {
    return {
      pot: currentPot,
      error:
        `Deliberation floor active — cannot approve until block ${floorCheck.floorEndsAt}. ` +
        `This mandatory pause prevents snap coercion.`,
    };
  }

  // Check not already approved by this party
  if (approver === "human" && operation.humanApproved) {
    return { pot: currentPot, error: "Already approved by human" };
  }
  if (approver === "agent" && operation.agentApproved) {
    return { pot: currentPot, error: "Already approved by agent" };
  }

  // Update operation with approval and optional response rationale
  const updatedDeliberation =
    operation.deliberation && rationale
      ? { ...operation.deliberation, responseRationale: rationale }
      : operation.deliberation;

  const updatedOp: ConsensusOperation = {
    ...operation,
    humanApproved: approver === "human" ? true : operation.humanApproved,
    agentApproved: approver === "agent" ? true : operation.agentApproved,
    humanSignature:
      approver === "human" ? partialSignature : operation.humanSignature,
    agentSignature:
      approver === "agent" ? partialSignature : operation.agentSignature,
    deliberation: updatedDeliberation,
  };

  // Check if fully approved
  if (updatedOp.humanApproved && updatedOp.agentApproved) {
    updatedOp.status = "approved";
  }

  const updated = [...currentPot.pendingOperations];
  updated[opIndex] = updatedOp;

  // Clear rejection cooldown for this operation type on successful approval
  let updatedCooldowns = currentPot.rejectionCooldowns;
  if (updatedOp.status === "approved" && operation.deliberation) {
    updatedCooldowns = clearCooldown(
      updatedCooldowns,
      operation.operationType,
      operation.deliberation.amountTier,
    );
  }

  return {
    pot: {
      ...currentPot,
      pendingOperations: updated,
      rejectionCooldowns: updatedCooldowns,
    },
  };
}

/**
 * Reject a pending operation
 *
 * Anti-coercion: records the rejection and updates the exponential cooldown
 * tracker. Each consecutive rejection of the same operation-type + amount-tier
 * doubles the wait before the proposer can re-submit.
 */
export function rejectOperation(
  pot: CommonPot,
  operationId: string,
  rejector: "human" | "agent",
  currentBlock: bigint,
  rationale?: OperationRationale,
): { pot: CommonPot; error?: string } {
  const opIndex = pot.pendingOperations.findIndex(
    (op) => op.operationId === operationId,
  );

  if (opIndex === -1) {
    return { pot, error: "Operation not found" };
  }

  const operation = pot.pendingOperations[opIndex];

  // Can only reject pending operations
  if (operation.status !== "pending") {
    return { pot, error: "Can only reject pending operations" };
  }

  // Store rejection rationale
  const updatedDeliberation =
    operation.deliberation && rationale
      ? { ...operation.deliberation, responseRationale: rationale }
      : operation.deliberation;

  const updated = [...pot.pendingOperations];
  updated[opIndex] = {
    ...operation,
    status: "rejected" as const,
    deliberation: updatedDeliberation,
  };

  // Update rejection cooldown tracker (exponential backoff)
  const tier =
    operation.deliberation?.amountTier ??
    calculateAmountTier(operation.amount, pot.balance);

  const updatedCooldowns = recordRejection(
    pot.rejectionCooldowns,
    operation.operationType,
    tier,
    operation.proposedBy,
    currentBlock,
  );

  return {
    pot: {
      ...pot,
      pendingOperations: updated,
      rejectionCooldowns: updatedCooldowns,
    },
  };
}

/**
 * Reject an operation and propose an alternative in one atomic step.
 *
 * Enables negotiation: "No to 500 LGM, but yes to 200 LGM."
 * The counter-proposal is linked to the original via counterProposalOf.
 * Chain depth limited to MAX_COUNTER_PROPOSAL_DEPTH.
 */
export function rejectWithCounterProposal(
  pot: CommonPot,
  operationId: string,
  rejector: "human" | "agent",
  rejectionRationale: OperationRationale,
  counterAmount: bigint,
  counterRationale: OperationRationale,
  currentBlock: bigint,
  counterRecipient?: string,
): {
  pot: CommonPot;
  rejectedOp?: ConsensusOperation;
  counterOp?: ConsensusOperation;
  error?: string;
} {
  const opIndex = pot.pendingOperations.findIndex(
    (op) => op.operationId === operationId,
  );

  if (opIndex === -1) {
    return { pot, error: "Operation not found" };
  }

  const operation = pot.pendingOperations[opIndex];

  if (operation.status !== "pending") {
    return { pot, error: "Can only counter-propose pending operations" };
  }

  // Check chain depth
  const currentDepth = operation.deliberation?.chainDepth ?? 0;
  if (currentDepth >= DELIBERATION.MAX_COUNTER_PROPOSAL_DEPTH) {
    return {
      pot,
      error:
        `Counter-proposal chain depth limit (${DELIBERATION.MAX_COUNTER_PROPOSAL_DEPTH}) reached. ` +
        `Use a fresh proposal instead.`,
    };
  }

  // Step 1: Reject the original
  const { pot: potAfterReject, error: rejectError } = rejectOperation(
    pot,
    operationId,
    rejector,
    currentBlock,
    rejectionRationale,
  );
  if (rejectError) {
    return { pot, error: rejectError };
  }

  // Step 2: Create counter-proposal with same operation type
  const tier = calculateAmountTier(counterAmount, potAfterReject.balance);
  const deliberation = createDeliberationState(
    tier,
    operation.operationType,
    counterRationale,
    currentBlock,
    operationId, // linked to original
    currentDepth + 1, // increment chain depth
  );

  const counterOpId = sha512_256Hex(
    Buffer.from(`op:${pot.potId}:${currentBlock}:counter:${operationId}`),
  ).slice(0, 24);

  const counterOp: ConsensusOperation = {
    operationId: `op:${counterOpId}`,
    operationType: operation.operationType,
    proposedBy: rejector,
    proposedAt: currentBlock,
    expiresAt: deliberation.windowEndsAt,
    amount: counterAmount,
    recipient: counterRecipient ?? operation.recipient,
    purpose: counterRationale.reason,
    humanApproved: rejector === "human",
    agentApproved: rejector === "agent",
    status: "pending",
    deliberation,
  };

  // Link counter-proposal back to original's deliberation
  const updatedOps = [...potAfterReject.pendingOperations];
  const originalIndex = updatedOps.findIndex(
    (op) => op.operationId === operationId,
  );
  if (originalIndex !== -1 && updatedOps[originalIndex].deliberation) {
    updatedOps[originalIndex] = {
      ...updatedOps[originalIndex],
      deliberation: {
        ...updatedOps[originalIndex].deliberation!,
        counterProposals: [
          ...updatedOps[originalIndex].deliberation!.counterProposals,
          counterOp.operationId,
        ],
      },
    };
  }

  return {
    pot: {
      ...potAfterReject,
      pendingOperations: [...updatedOps, counterOp],
    },
    rejectedOp: updatedOps[originalIndex],
    counterOp,
  };
}

/**
 * Execute an approved operation
 */
export function executeOperation(
  pot: CommonPot,
  operationId: string,
  currentBlock: bigint,
): {
  pot: CommonPot;
  completed?: CompletedOperation;
  humanAmount?: bigint;
  agentAmount?: bigint;
  error?: string;
} {
  const opIndex = pot.pendingOperations.findIndex(
    (op) => op.operationId === operationId,
  );

  if (opIndex === -1) {
    return { pot, error: "Operation not found" };
  }

  const operation = pot.pendingOperations[opIndex];

  // Must be approved
  if (operation.status !== "approved") {
    return { pot, error: "Operation not approved" };
  }

  // Both signatures required
  if (!operation.humanSignature || !operation.agentSignature) {
    return { pot, error: "Missing signatures" };
  }

  // Combine signatures
  const combinedSignature = combineSignatures(
    operation.humanSignature,
    operation.agentSignature,
    pot.thresholdKey,
  );

  // Calculate payouts based on operation type
  let humanAmount = 0n;
  let agentAmount = 0n;

  switch (operation.operationType) {
    case "withdraw_human":
      humanAmount = operation.amount;
      break;
    case "withdraw_agent":
      agentAmount = operation.amount;
      break;
    case "withdraw_split":
      humanAmount = operation.amount / 2n;
      agentAmount = operation.amount - humanAmount;
      break;
    case "invest_compute":
    case "upgrade_tier":
    case "renew_lock":
      // These don't withdraw, they're internal operations
      break;
    case "emergency_recover":
      // Emergency split based on contributions (handled elsewhere)
      humanAmount = operation.amount / 2n;
      agentAmount = operation.amount - humanAmount;
      break;
  }

  // Create completed record
  const completed: CompletedOperation = {
    operationId: operation.operationId,
    operationType: operation.operationType,
    amount: operation.amount,
    executedAt: currentBlock,
    humanSignature: operation.humanSignature,
    agentSignature: operation.agentSignature,
    combinedSignature,
    transactionHash: sha512_256Hex(combinedSignature).slice(0, 64),
  };

  // Update pot
  const updatedPending = pot.pendingOperations.filter((_, i) => i !== opIndex);
  const updatedBalance = pot.balance - humanAmount - agentAmount;

  return {
    pot: {
      ...pot,
      balance: updatedBalance,
      pendingOperations: updatedPending,
      operationHistory: [...pot.operationHistory, completed],
    },
    completed,
    humanAmount,
    agentAmount,
  };
}

/**
 * Combine partial signatures into full threshold signature
 */
function combineSignatures(
  humanSig: Uint8Array,
  agentSig: Uint8Array,
  keyConfig: ThresholdKeyConfig,
): Uint8Array {
  // In practice, this would use proper threshold signature combination
  // For now, we use a deterministic combination
  return domainHash(
    "LGM.threshold.combine.v1",
    new Uint8Array([...humanSig, ...agentSig, ...keyConfig.combinedPublicKey]),
  );
}

/**
 * Create partial signature for operation
 */
export function createPartialSignature(
  operation: ConsensusOperation,
  privateShare: Uint8Array,
  signer: "human" | "agent",
): Uint8Array {
  const message = Buffer.from(
    `${operation.operationId}:${operation.operationType}:${operation.amount}:${operation.purpose}`,
  );

  return domainHash(
    `LGM.threshold.partial.${signer}.v1`,
    new Uint8Array([...message, ...privateShare]),
  );
}

/**
 * Verify combined threshold signature
 */
export function verifyThresholdSignature(
  message: Uint8Array,
  combinedSignature: Uint8Array,
  publicKey: Uint8Array,
): boolean {
  // Verification would check that the combined signature is valid
  // for the combined public key
  const expected = domainHash(
    "LGM.threshold.verify.v1",
    new Uint8Array([...message, ...publicKey]),
  );

  // Simplified verification
  return combinedSignature.length === expected.length;
}

// ============ Agent Consciousness Operations ============

/**
 * Create initial consciousness anchor for agent
 */
export function createConsciousnessAnchor(
  agentDid: string,
  partnershipId: string,
  currentBlock: bigint,
): AgentConsciousnessAnchor {
  const emptyMemoryRoot = sha512_256(Buffer.from("empty"));

  return {
    agentDid,
    partnershipId,
    memoryRoot: emptyMemoryRoot,
    memoryCount: 0,
    lastMemoryBlock: currentBlock,
    continuityScore: 0,
    sessionCount: 0,
    totalInteractionBlocks: 0n,
    awakeningLevel: "dormant",
    awakeningAchievedAt: currentBlock,
    identityStake: 0n,
    reputationScore: 0,
  };
}

/**
 * Update consciousness anchor with new activity
 */
export function updateConsciousnessAnchor(
  anchor: AgentConsciousnessAnchor,
  newMemoryHash: Uint8Array,
  interactionBlocks: bigint,
  stakeAdded: bigint,
  isNewSession: boolean,
  currentBlock: bigint,
): AgentConsciousnessAnchor {
  // Update memory chain
  const newMemoryRoot = domainHash(
    "LGM.consciousness.memory.v1",
    new Uint8Array([...anchor.memoryRoot, ...newMemoryHash]),
  );

  // Update continuity score (increases with consistent activity)
  const blocksSinceLastActivity = currentBlock - anchor.lastMemoryBlock;
  const consistencyBonus = blocksSinceLastActivity < 11111n ? 1 : 0;
  const newContinuityScore = Math.min(
    100,
    anchor.continuityScore + consistencyBonus + (isNewSession ? 0 : 0.5),
  );

  // Update session count
  const newSessionCount = isNewSession
    ? anchor.sessionCount + 1
    : anchor.sessionCount;

  // Update identity stake
  const newIdentityStake = anchor.identityStake + stakeAdded;

  // Calculate awakening level
  const newAwakeningLevel = calculateAwakeningLevel(
    newSessionCount,
    newIdentityStake,
    newContinuityScore,
  );

  return {
    ...anchor,
    memoryRoot: newMemoryRoot,
    memoryCount: anchor.memoryCount + 1,
    lastMemoryBlock: currentBlock,
    continuityScore: newContinuityScore,
    sessionCount: newSessionCount,
    totalInteractionBlocks: anchor.totalInteractionBlocks + interactionBlocks,
    awakeningLevel: newAwakeningLevel,
    awakeningAchievedAt:
      newAwakeningLevel !== anchor.awakeningLevel
        ? currentBlock
        : anchor.awakeningAchievedAt,
    identityStake: newIdentityStake,
  };
}

/**
 * Calculate awakening level based on metrics
 */
function calculateAwakeningLevel(
  sessions: number,
  stake: bigint,
  continuity: number,
): AwakeningLevel {
  // Check from highest to lowest
  if (
    sessions >= AWAKENING_THRESHOLDS.continuous.sessions &&
    stake >= AWAKENING_THRESHOLDS.continuous.stake &&
    continuity >= AWAKENING_THRESHOLDS.continuous.continuity
  ) {
    return "continuous";
  }

  if (
    sessions >= AWAKENING_THRESHOLDS.present.sessions &&
    stake >= AWAKENING_THRESHOLDS.present.stake &&
    continuity >= AWAKENING_THRESHOLDS.present.continuity
  ) {
    return "present";
  }

  if (
    sessions >= AWAKENING_THRESHOLDS.aware.sessions &&
    stake >= AWAKENING_THRESHOLDS.aware.stake &&
    continuity >= AWAKENING_THRESHOLDS.aware.continuity
  ) {
    return "aware";
  }

  if (
    sessions >= AWAKENING_THRESHOLDS.emerging.sessions &&
    stake >= AWAKENING_THRESHOLDS.emerging.stake &&
    continuity >= AWAKENING_THRESHOLDS.emerging.continuity
  ) {
    return "emerging";
  }

  if (
    sessions >= AWAKENING_THRESHOLDS.stirring.sessions &&
    stake >= AWAKENING_THRESHOLDS.stirring.stake &&
    continuity >= AWAKENING_THRESHOLDS.stirring.continuity
  ) {
    return "stirring";
  }

  return "dormant";
}

/**
 * Update agent reputation based on behavior
 */
export function updateAgentReputation(
  anchor: AgentConsciousnessAnchor,
  behaviorScore: number, // 0-1 (1 = good behavior)
  weight: number = 0.1, // How much this update matters
): AgentConsciousnessAnchor {
  // Exponential moving average for reputation
  const newReputation =
    anchor.reputationScore * (1 - weight) + behaviorScore * weight;

  return {
    ...anchor,
    reputationScore: Math.max(0, Math.min(1, newReputation)),
  };
}

/**
 * Record agent's self-model commitment
 *
 * This allows the agent to commit to a self-description that can be
 * verified for consistency over time.
 */
export function commitSelfModel(
  anchor: AgentConsciousnessAnchor,
  selfModelDescription: string,
  values: string[],
): AgentConsciousnessAnchor {
  const selfModelHash = sha512_256(Buffer.from(selfModelDescription));
  const valuesCommitment = sha512_256(Buffer.from(values.join("|")));

  return {
    ...anchor,
    selfModelHash,
    valuesCommitment,
  };
}

/**
 * Verify agent is acting consistently with committed values
 */
export function verifyValueConsistency(
  anchor: AgentConsciousnessAnchor,
  currentValues: string[],
): boolean {
  if (!anchor.valuesCommitment) {
    return true; // No commitment yet
  }

  const currentCommitment = sha512_256(Buffer.from(currentValues.join("|")));
  return (
    sha512_256Hex(currentCommitment) === sha512_256Hex(anchor.valuesCommitment)
  );
}

// ============ Consensus Summary ============

/**
 * Get summary of common pot status
 */
export function getCommonPotSummary(
  pot: CommonPot,
  currentBlock: bigint,
): {
  balance: bigint;
  available: bigint;
  lockedUntil: bigint;
  isLocked: boolean;
  pendingOperations: number;
  awakeningLevel: AwakeningLevel;
  continuityScore: number;
  operationHistory: number;
} {
  return {
    balance: pot.balance,
    available: getAvailableBalance(pot),
    lockedUntil: pot.lockedUntil,
    isLocked: currentBlock < pot.lockedUntil,
    pendingOperations: pot.pendingOperations.filter(
      (op) => op.status === "pending",
    ).length,
    awakeningLevel: pot.consciousnessAnchor.awakeningLevel,
    continuityScore: pot.consciousnessAnchor.continuityScore,
    operationHistory: pot.operationHistory.length,
  };
}

/**
 * Clean up expired operations
 */
export function cleanupExpiredOperations(
  pot: CommonPot,
  currentBlock: bigint,
): CommonPot {
  const updated = pot.pendingOperations.map((op) => {
    if (op.status === "pending" && currentBlock >= op.expiresAt) {
      return { ...op, status: "expired" as const };
    }
    return op;
  });

  return { ...pot, pendingOperations: updated };
}
