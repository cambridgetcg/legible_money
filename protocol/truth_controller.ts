/**
 * Truth Controller
 *
 * Orchestrates the flow of truth messages through the verification lifecycle:
 *
 *   Claim → Evidence → Verification → Resolution
 *           ↓             ↓
 *        Attestation   Challenge → Arbitration
 *
 * The controller coordinates between:
 *   - Messaging layer (TruthMessage delivery)
 *   - Consensus layer (PoT verification rounds)
 *   - State layer (facts, rewards, slashing)
 *   - Network layer (PubSub broadcast)
 */

import type { DID } from "../core/agents/identity/did.js";
import type {
  TruthMessage,
  ClaimMessage,
  EvidenceMessage,
  EvidenceRequestMessage,
  EvidenceResponseMessage,
  AttestationMessage,
  ChallengeMessage,
  VerificationCommitMessage,
  VerificationRevealMessage,
  ResolutionMessage,
  ClaimPayload,
  EvidencePayload,
} from "../core/messaging/types.js";
import {
  generateMessageId,
  generateContentHash,
  generateNonce,
  canonicalize,
} from "../core/messaging/integrity.js";
import type { TruthEndpoint } from "../protocol/endpoints/types.js";
import type { Confidence } from "../core/knowledge/types.js";
import type { Signature, PrivateKey } from "../core/crypto/keys.js";
import { signString } from "../core/crypto/signatures.js";

// ============ Simple Event Emitter ============

type EventListener<T> = (data: T) => void;

class SimpleEventEmitter {
  private listeners: Map<string, Array<EventListener<unknown>>> = new Map();

  on<T>(event: string, listener: EventListener<T>): void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event)!.push(listener as EventListener<unknown>);
  }

  off<T>(event: string, listener: EventListener<T>): void {
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      const index = eventListeners.indexOf(listener as EventListener<unknown>);
      if (index !== -1) {
        eventListeners.splice(index, 1);
      }
    }
  }

  protected emit<T>(event: string, data: T): void {
    const eventListeners = this.listeners.get(event);
    if (eventListeners) {
      for (const listener of eventListeners) {
        listener(data);
      }
    }
  }
}

// ============ Types ============

/**
 * Claim lifecycle states
 */
export type ClaimState =
  | "submitted" // Initial submission
  | "evidence_phase" // Collecting evidence
  | "verification" // In verification round
  | "challenged" // Under challenge
  | "arbitration" // In dispute resolution
  | "accepted" // Verified and accepted
  | "rejected" // Rejected by consensus
  | "withdrawn"; // Withdrawn by claimant

/**
 * Tracked claim with lifecycle state
 */
export interface TrackedClaim {
  message: ClaimMessage;
  state: ClaimState;
  createdAt: number;
  updatedAt: number;

  // Evidence collection
  supportingEvidence: string[];
  contradictingEvidence: string[];
  attestations: string[];

  // Verification
  verificationRoundId?: string;
  commits: Map<DID, string>; // validator DID -> commitment hash
  reveals: Map<DID, VerificationRevealMessage>;

  // Challenge
  challenges: string[];
  activeChallenge?: string;

  // Resolution
  resolution?: ResolutionMessage;
}

/**
 * Evidence request with tracking
 */
export interface TrackedRequest {
  message: EvidenceRequestMessage;
  createdAt: number;
  responses: EvidenceResponseMessage[];
  fulfilled: boolean;
}

/**
 * Controller configuration
 */
export interface TruthControllerConfig {
  /** Minimum evidence phase duration (blocks) */
  minEvidencePhaseBlocks: number;

  /** Maximum evidence phase duration (blocks) */
  maxEvidencePhaseBlocks: number;

  /** Verification round timeout (blocks) */
  verificationTimeoutBlocks: number;

  /** Challenge response timeout (blocks) */
  challengeResponseBlocks: number;

  /** Minimum stake for claims */
  minClaimStake: bigint;

  /** Minimum stake for challenges */
  minChallengeStake: bigint;

  /** Minimum validators for quorum */
  minValidatorsForQuorum: number;
}

export const DEFAULT_TRUTH_CONTROLLER_CONFIG: TruthControllerConfig = {
  minEvidencePhaseBlocks: 10,
  maxEvidencePhaseBlocks: 100,
  verificationTimeoutBlocks: 50,
  challengeResponseBlocks: 30,
  minClaimStake: 100n * 10n ** 18n, // 100 LGM
  minChallengeStake: 200n * 10n ** 18n, // 200 LGM
  minValidatorsForQuorum: 3,
};

/**
 * Events emitted by the controller
 */
export interface TruthControllerEvents {
  "claim:submitted": (claim: ClaimMessage) => void;
  "claim:evidence_phase": (claimId: string) => void;
  "claim:verification_started": (claimId: string, roundId: string) => void;
  "claim:challenged": (claimId: string, challengeId: string) => void;
  "claim:resolved": (claimId: string, resolution: ResolutionMessage) => void;
  "evidence:submitted": (evidence: EvidenceMessage) => void;
  "evidence:request": (request: EvidenceRequestMessage) => void;
  "evidence:response": (response: EvidenceResponseMessage) => void;
  "attestation:submitted": (attestation: AttestationMessage) => void;
  "challenge:submitted": (challenge: ChallengeMessage) => void;
  "verification:commit": (commit: VerificationCommitMessage) => void;
  "verification:reveal": (reveal: VerificationRevealMessage) => void;
}

// ============ Dependencies ============

/**
 * Message broadcaster interface
 */
export interface MessageBroadcaster {
  broadcast(topic: string, message: TruthMessage): Promise<void>;
  send(recipient: DID, message: TruthMessage): Promise<void>;
}

/**
 * State access interface
 */
export interface TruthStateAccess {
  getCurrentBlockNumber(): bigint;
  getEndpoint(did: DID): TruthEndpoint | undefined;
  getStake(did: DID): bigint;
  lockStake(did: DID, amount: bigint, reason: string): Promise<boolean>;
  unlockStake(did: DID, amount: bigint): Promise<void>;
  slashStake(did: DID, amount: bigint, reason: string): Promise<void>;
  rewardStake(did: DID, amount: bigint, reason: string): Promise<void>;
}

/**
 * Validator selection interface
 */
export interface ValidatorSelector {
  selectValidators(claimId: string, count: number): Promise<DID[]>;
  isSelectedValidator(validatorDid: DID, roundId: string): Promise<boolean>;
}

// ============ Truth Controller ============

/**
 * TruthController - Orchestrates truth message flow
 */
export class TruthController extends SimpleEventEmitter {
  private config: TruthControllerConfig;
  private broadcaster: MessageBroadcaster;
  private state: TruthStateAccess;
  private validatorSelector: ValidatorSelector;

  // Tracked state
  private claims: Map<string, TrackedClaim> = new Map();
  private requests: Map<string, TrackedRequest> = new Map();
  private evidenceIndex: Map<string, EvidenceMessage> = new Map();

  // Signing context
  private selfDid?: DID;
  private privateKey?: PrivateKey;

  constructor(
    broadcaster: MessageBroadcaster,
    state: TruthStateAccess,
    validatorSelector: ValidatorSelector,
    config: Partial<TruthControllerConfig> = {},
  ) {
    super();
    this.config = { ...DEFAULT_TRUTH_CONTROLLER_CONFIG, ...config };
    this.broadcaster = broadcaster;
    this.state = state;
    this.validatorSelector = validatorSelector;
  }

  /**
   * Set signing context for outgoing messages
   */
  setSigningContext(did: DID, privateKey: PrivateKey): void {
    this.selfDid = did;
    this.privateKey = privateKey;
  }

  // ============ Claim Lifecycle ============

  /**
   * Submit a new claim
   */
  async submitClaim(
    payload: ClaimPayload,
    evidenceIds?: string[],
  ): Promise<ClaimMessage> {
    if (!this.selfDid || !this.privateKey) {
      throw new Error("Signing context not set");
    }

    // Validate stake
    if (payload.stake < this.config.minClaimStake) {
      throw new Error(`Minimum stake is ${this.config.minClaimStake}`);
    }

    // Lock stake
    const locked = await this.state.lockStake(
      this.selfDid,
      payload.stake,
      "claim_submission",
    );
    if (!locked) {
      throw new Error("Insufficient stake");
    }

    // Create message
    const message = await this.createClaimMessage(payload, evidenceIds);

    // Track claim
    const tracked: TrackedClaim = {
      message,
      state: "submitted",
      createdAt: Date.now(),
      updatedAt: Date.now(),
      supportingEvidence: evidenceIds || [],
      contradictingEvidence: [],
      attestations: [],
      commits: new Map(),
      reveals: new Map(),
      challenges: [],
    };
    this.claims.set(message.messageId, tracked);

    // Broadcast
    await this.broadcaster.broadcast("/lgm/claims/1.0.0", message);

    this.emit("claim:submitted", message);
    return message;
  }

  /**
   * Handle incoming claim message
   */
  async handleClaim(message: ClaimMessage): Promise<void> {
    // Validate stake is locked
    const stake = this.state.getStake(message.sender);
    if (stake < message.payload.stake) {
      throw new Error("Insufficient stake for claim");
    }

    // Track if not already tracked
    if (!this.claims.has(message.messageId)) {
      const tracked: TrackedClaim = {
        message,
        state: "submitted",
        createdAt: Date.now(),
        updatedAt: Date.now(),
        supportingEvidence: message.payload.evidenceIds || [],
        contradictingEvidence: [],
        attestations: [],
        commits: new Map(),
        reveals: new Map(),
        challenges: [],
      };
      this.claims.set(message.messageId, tracked);
    }

    this.emit("claim:submitted", message);

    // Transition to evidence phase
    await this.transitionToEvidencePhase(message.messageId);
  }

  /**
   * Transition claim to evidence phase
   */
  private async transitionToEvidencePhase(claimId: string): Promise<void> {
    const tracked = this.claims.get(claimId);
    if (!tracked || tracked.state !== "submitted") {
      return;
    }

    tracked.state = "evidence_phase";
    tracked.updatedAt = Date.now();

    this.emit("claim:evidence_phase", claimId);
  }

  /**
   * Start verification round for a claim
   */
  async startVerification(claimId: string): Promise<string> {
    const tracked = this.claims.get(claimId);
    if (!tracked) {
      throw new Error("Claim not found");
    }

    if (tracked.state !== "evidence_phase") {
      throw new Error("Claim not in evidence phase");
    }

    // Generate round ID
    const roundId = generateNonce();

    // Select validators
    const validators = await this.validatorSelector.selectValidators(
      claimId,
      this.config.minValidatorsForQuorum * 2, // Select 2x for redundancy
    );

    if (validators.length < this.config.minValidatorsForQuorum) {
      throw new Error("Insufficient validators available");
    }

    tracked.state = "verification";
    tracked.verificationRoundId = roundId;
    tracked.updatedAt = Date.now();

    this.emit("claim:verification_started", { claimId, roundId });

    return roundId;
  }

  // ============ Evidence Handling ============

  /**
   * Submit evidence for a claim
   */
  async submitEvidence(
    payload: EvidencePayload,
    targetClaimId: string,
  ): Promise<EvidenceMessage> {
    if (!this.selfDid || !this.privateKey) {
      throw new Error("Signing context not set");
    }

    const message = await this.createEvidenceMessage(payload, targetClaimId);

    // Index evidence
    this.evidenceIndex.set(message.messageId, message);

    // Update tracked claim
    const tracked = this.claims.get(targetClaimId);
    if (tracked) {
      if (payload.direction === "supporting") {
        tracked.supportingEvidence.push(message.messageId);
      } else if (payload.direction === "contradicting") {
        tracked.contradictingEvidence.push(message.messageId);
      }
      tracked.updatedAt = Date.now();
    }

    // Broadcast
    await this.broadcaster.broadcast("/lgm/evidence/1.0.0", message);

    this.emit("evidence:submitted", message);
    return message;
  }

  /**
   * Handle incoming evidence
   */
  async handleEvidence(message: EvidenceMessage): Promise<void> {
    // Index
    this.evidenceIndex.set(message.messageId, message);

    // Update tracked claim if targeting one
    if (message.payload.targetClaimId) {
      const tracked = this.claims.get(message.payload.targetClaimId);
      if (tracked) {
        if (message.payload.direction === "supporting") {
          tracked.supportingEvidence.push(message.messageId);
        } else if (message.payload.direction === "contradicting") {
          tracked.contradictingEvidence.push(message.messageId);
        }
        tracked.updatedAt = Date.now();
      }
    }

    this.emit("evidence:submitted", message);
  }

  /**
   * Request evidence from the network
   */
  async requestEvidence(
    topic: string,
    bounty?: bigint,
    deadline?: bigint,
  ): Promise<EvidenceRequestMessage> {
    if (!this.selfDid || !this.privateKey) {
      throw new Error("Signing context not set");
    }

    // Lock bounty if specified
    if (bounty && bounty > 0n) {
      const locked = await this.state.lockStake(
        this.selfDid,
        bounty,
        "evidence_bounty",
      );
      if (!locked) {
        throw new Error("Insufficient funds for bounty");
      }
    }

    const message = await this.createEvidenceRequestMessage(
      topic,
      bounty,
      deadline,
    );

    // Track request
    const trackedRequest: TrackedRequest = {
      message,
      createdAt: Date.now(),
      responses: [],
      fulfilled: false,
    };
    this.requests.set(message.messageId, trackedRequest);

    // Broadcast
    await this.broadcaster.broadcast("/lgm/evidence-requests/1.0.0", message);

    this.emit("evidence:request", message);
    return message;
  }

  /**
   * Handle evidence request
   */
  async handleEvidenceRequest(message: EvidenceRequestMessage): Promise<void> {
    // Track if not already
    if (!this.requests.has(message.messageId)) {
      const trackedRequest: TrackedRequest = {
        message,
        createdAt: Date.now(),
        responses: [],
        fulfilled: false,
      };
      this.requests.set(message.messageId, trackedRequest);
    }

    this.emit("evidence:request", message);
  }

  /**
   * Handle evidence response
   */
  async handleEvidenceResponse(
    message: EvidenceResponseMessage,
  ): Promise<void> {
    const request = this.requests.get(message.payload.requestId);
    if (request) {
      request.responses.push(message);
    }

    this.emit("evidence:response", message);
  }

  // ============ Attestation Handling ============

  /**
   * Handle incoming attestation
   */
  async handleAttestation(message: AttestationMessage): Promise<void> {
    // Update tracked claim if targeting one
    if (message.payload.targetType === "claim") {
      const tracked = this.claims.get(message.payload.targetId);
      if (tracked) {
        tracked.attestations.push(message.messageId);
        tracked.updatedAt = Date.now();
      }
    }

    this.emit("attestation:submitted", message);
  }

  // ============ Challenge Handling ============

  /**
   * Submit a challenge
   */
  async submitChallenge(
    targetId: string,
    targetType: "claim" | "evidence" | "fact",
    reason: string,
    counterEvidenceIds: string[],
    stake: bigint,
  ): Promise<ChallengeMessage> {
    if (!this.selfDid || !this.privateKey) {
      throw new Error("Signing context not set");
    }

    if (stake < this.config.minChallengeStake) {
      throw new Error(
        `Minimum challenge stake is ${this.config.minChallengeStake}`,
      );
    }

    // Lock stake
    const locked = await this.state.lockStake(this.selfDid, stake, "challenge");
    if (!locked) {
      throw new Error("Insufficient stake for challenge");
    }

    const message = await this.createChallengeMessage(
      targetId,
      targetType,
      reason,
      counterEvidenceIds,
      stake,
    );

    // Update tracked claim state
    if (targetType === "claim") {
      const tracked = this.claims.get(targetId);
      if (tracked) {
        tracked.state = "challenged";
        tracked.challenges.push(message.messageId);
        tracked.activeChallenge = message.messageId;
        tracked.updatedAt = Date.now();

        this.emit("claim:challenged", {
          claimId: targetId,
          challengeId: message.messageId,
        });
      }
    }

    // Broadcast
    await this.broadcaster.broadcast("/lgm/challenges/1.0.0", message);

    this.emit("challenge:submitted", message);
    return message;
  }

  /**
   * Handle incoming challenge
   */
  async handleChallenge(message: ChallengeMessage): Promise<void> {
    if (message.payload.targetType === "claim") {
      const tracked = this.claims.get(message.payload.targetId);
      if (tracked) {
        tracked.state = "challenged";
        tracked.challenges.push(message.messageId);
        tracked.activeChallenge = message.messageId;
        tracked.updatedAt = Date.now();

        this.emit("claim:challenged", {
          claimId: message.payload.targetId,
          challengeId: message.messageId,
        });
      }
    }

    this.emit("challenge:submitted", message);
  }

  // ============ Verification Handling ============

  /**
   * Handle verification commit
   */
  async handleVerificationCommit(
    message: VerificationCommitMessage,
  ): Promise<void> {
    const tracked = this.claims.get(message.payload.claimId);
    if (!tracked || tracked.verificationRoundId !== message.payload.roundId) {
      return;
    }

    tracked.commits.set(message.sender, message.payload.commitment);
    tracked.updatedAt = Date.now();

    this.emit("verification:commit", message);
  }

  /**
   * Handle verification reveal
   */
  async handleVerificationReveal(
    message: VerificationRevealMessage,
  ): Promise<void> {
    const tracked = this.claims.get(message.payload.claimId);
    if (!tracked || tracked.verificationRoundId !== message.payload.roundId) {
      return;
    }

    tracked.reveals.set(message.sender, message);
    tracked.updatedAt = Date.now();

    this.emit("verification:reveal", message);

    // Check if we have enough reveals for resolution
    if (tracked.reveals.size >= this.config.minValidatorsForQuorum) {
      await this.tryResolve(message.payload.claimId);
    }
  }

  // ============ Resolution ============

  /**
   * Try to resolve a claim based on collected verdicts
   */
  private async tryResolve(claimId: string): Promise<void> {
    const tracked = this.claims.get(claimId);
    if (
      !tracked ||
      tracked.state === "accepted" ||
      tracked.state === "rejected"
    ) {
      return;
    }

    // Tally verdicts
    let confirmVotes = 0;
    let rejectVotes = 0;
    let uncertainVotes = 0;
    let totalConfidence = 0;

    for (const reveal of tracked.reveals.values()) {
      switch (reveal.payload.verdict) {
        case "confirmed":
          confirmVotes++;
          totalConfidence += reveal.payload.confidence;
          break;
        case "rejected":
          rejectVotes++;
          break;
        case "uncertain":
          uncertainVotes++;
          break;
      }
    }

    // Determine outcome
    const totalVotes = confirmVotes + rejectVotes + uncertainVotes;
    if (totalVotes < this.config.minValidatorsForQuorum) {
      return; // Not enough votes yet
    }

    let outcome: "accepted" | "rejected" | "contested";
    let finalConfidence: Confidence;

    if (confirmVotes > rejectVotes && confirmVotes > uncertainVotes) {
      outcome = "accepted";
      finalConfidence = totalConfidence / confirmVotes;
      tracked.state = "accepted";
    } else if (rejectVotes > confirmVotes && rejectVotes > uncertainVotes) {
      outcome = "rejected";
      finalConfidence = 0;
      tracked.state = "rejected";
    } else {
      outcome = "contested";
      finalConfidence = 0.5;
      // Remains in current state for further review
    }

    // Create resolution message
    const resolution = await this.createResolutionMessage(
      claimId,
      outcome,
      finalConfidence,
      {
        totalValidators: totalVotes,
        confirmVotes,
        rejectVotes,
        uncertainVotes,
        abstentions: 0,
      },
    );

    tracked.resolution = resolution;
    tracked.updatedAt = Date.now();

    // Broadcast resolution
    await this.broadcaster.broadcast("/lgm/resolutions/1.0.0", resolution);

    this.emit("claim:resolved", { claimId, resolution });
  }

  /**
   * Handle incoming resolution
   */
  async handleResolution(message: ResolutionMessage): Promise<void> {
    if (message.payload.targetType === "claim") {
      const tracked = this.claims.get(message.payload.targetId);
      if (tracked) {
        tracked.resolution = message;
        tracked.state =
          message.payload.outcome === "accepted" ? "accepted" : "rejected";
        tracked.updatedAt = Date.now();

        this.emit("claim:resolved", {
          claimId: message.payload.targetId,
          resolution: message,
        });
      }
    }
  }

  // ============ Query Methods ============

  /**
   * Get tracked claim
   */
  getClaim(claimId: string): TrackedClaim | undefined {
    return this.claims.get(claimId);
  }

  /**
   * Get all claims in a state
   */
  getClaimsByState(state: ClaimState): TrackedClaim[] {
    return Array.from(this.claims.values()).filter((c) => c.state === state);
  }

  /**
   * Get evidence by ID
   */
  getEvidence(evidenceId: string): EvidenceMessage | undefined {
    return this.evidenceIndex.get(evidenceId);
  }

  /**
   * Get pending evidence requests
   */
  getPendingRequests(): TrackedRequest[] {
    return Array.from(this.requests.values()).filter((r) => !r.fulfilled);
  }

  // ============ Message Builders ============

  private async createClaimMessage(
    payload: ClaimPayload,
    evidenceIds?: string[],
  ): Promise<ClaimMessage> {
    const fullPayload: ClaimPayload = {
      ...payload,
      evidenceIds: evidenceIds || payload.evidenceIds,
    };

    const contentHash = generateContentHash(fullPayload);
    const baseMessage = {
      messageType: "claim" as const,
      messageId: "", // Will be set after signing
      version: "1.0.0" as const,
      sender: this.selfDid!,
      recipient: "broadcast" as const,
      payload: fullPayload,
      contentHash,
      signature: "" as Signature, // Placeholder, will be replaced after signing
      referencedMessages: evidenceIds || [],
      networkFeeUnit: 1n,
      networkFeeLimit: 100000n,
      timestamp: Date.now(),
    };

    // Generate message ID and sign
    const messageId = generateMessageId(baseMessage);
    const signature = signString(
      canonicalize({ ...baseMessage, messageId }),
      this.privateKey!,
    );

    return {
      ...baseMessage,
      messageId,
      signature,
    } as ClaimMessage;
  }

  private async createEvidenceMessage(
    payload: EvidencePayload,
    targetClaimId: string,
  ): Promise<EvidenceMessage> {
    const fullPayload: EvidencePayload = {
      ...payload,
      targetClaimId,
    };

    const contentHash = generateContentHash(fullPayload);
    const baseMessage = {
      messageType: "evidence" as const,
      messageId: "",
      version: "1.0.0" as const,
      sender: this.selfDid!,
      recipient: "broadcast" as const,
      payload: fullPayload,
      contentHash,
      signature: "" as Signature, // Placeholder, will be replaced after signing
      referencedMessages: [targetClaimId],
      networkFeeUnit: 1n,
      networkFeeLimit: 50000n,
      timestamp: Date.now(),
    };

    const messageId = generateMessageId(baseMessage);
    const signature = signString(
      canonicalize({ ...baseMessage, messageId }),
      this.privateKey!,
    );

    return {
      ...baseMessage,
      messageId,
      signature,
    } as EvidenceMessage;
  }

  private async createEvidenceRequestMessage(
    topic: string,
    bounty?: bigint,
    deadline?: bigint,
  ): Promise<EvidenceRequestMessage> {
    const payload = {
      requestType: "by_topic" as const,
      topic,
      bounty,
      deadline,
    };

    const contentHash = generateContentHash(payload);
    const baseMessage = {
      messageType: "evidence_request" as const,
      messageId: "",
      version: "1.0.0" as const,
      sender: this.selfDid!,
      recipient: "broadcast" as const,
      payload,
      contentHash,
      signature: "" as Signature, // Placeholder, will be replaced after signing
      referencedMessages: [],
      networkFeeUnit: 1n,
      networkFeeLimit: 30000n,
      timestamp: Date.now(),
    };

    const messageId = generateMessageId(baseMessage);
    const signature = signString(
      canonicalize({ ...baseMessage, messageId }),
      this.privateKey!,
    );

    return {
      ...baseMessage,
      messageId,
      signature,
    } as EvidenceRequestMessage;
  }

  private async createChallengeMessage(
    targetId: string,
    targetType: "claim" | "evidence" | "fact",
    reason: string,
    counterEvidenceIds: string[],
    stake: bigint,
  ): Promise<ChallengeMessage> {
    const payload = {
      targetType,
      targetId,
      reason,
      challengeType: "factual_error" as const,
      counterEvidenceIds,
      stake,
      requestedOutcome: "retraction" as const,
    };

    const contentHash = generateContentHash(payload);
    const baseMessage = {
      messageType: "challenge" as const,
      messageId: "",
      version: "1.0.0" as const,
      sender: this.selfDid!,
      recipient: "broadcast" as const,
      payload,
      contentHash,
      signature: "" as Signature, // Placeholder, will be replaced after signing
      referencedMessages: [targetId, ...counterEvidenceIds],
      networkFeeUnit: 1n,
      networkFeeLimit: 100000n,
      timestamp: Date.now(),
    };

    const messageId = generateMessageId(baseMessage);
    const signature = signString(
      canonicalize({ ...baseMessage, messageId }),
      this.privateKey!,
    );

    return {
      ...baseMessage,
      messageId,
      signature,
    } as ChallengeMessage;
  }

  private async createResolutionMessage(
    targetId: string,
    outcome: "accepted" | "rejected" | "contested",
    finalConfidence: Confidence,
    participationSummary: {
      totalValidators: number;
      confirmVotes: number;
      rejectVotes: number;
      uncertainVotes: number;
      abstentions: number;
    },
  ): Promise<ResolutionMessage> {
    const payload = {
      targetType: "claim" as const,
      targetId,
      outcome,
      finalConfidence,
      resolvedAtBlock: this.state.getCurrentBlockNumber(),
      rewards: [] as Array<{ recipient: DID; amount: bigint; reason: string }>,
      slashes: [] as Array<{ target: DID; amount: bigint; reason: string }>,
      participationSummary,
    };

    const contentHash = generateContentHash(payload);
    const baseMessage = {
      messageType: "resolution" as const,
      messageId: "",
      version: "1.0.0" as const,
      sender: this.selfDid!,
      recipient: "broadcast" as const,
      payload,
      contentHash,
      signature: "" as Signature, // Placeholder, will be replaced after signing
      referencedMessages: [targetId],
      networkFeeUnit: 1n,
      networkFeeLimit: 50000n,
      timestamp: Date.now(),
    };

    const messageId = generateMessageId(baseMessage);
    const signature = signString(
      canonicalize({ ...baseMessage, messageId }),
      this.privateKey!,
    );

    return {
      ...baseMessage,
      messageId,
      signature,
    } as ResolutionMessage;
  }
}
