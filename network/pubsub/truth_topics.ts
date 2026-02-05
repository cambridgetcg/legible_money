/**
 * Truth Message PubSub Topic Handlers
 *
 * Topic handlers for the Truth & Evidence Message System.
 * Integrates with the existing TopicManager infrastructure.
 *
 * Topics:
 *   - EVIDENCE: Evidence submissions (supporting/contradicting)
 *   - EVIDENCE_REQUESTS: Bounty-based evidence requests
 *   - ATTESTATIONS: Third-party validations
 *   - CHALLENGES: Dispute announcements
 *   - RESOLUTIONS: Final outcomes
 *   - ACKNOWLEDGMENTS: Delivery confirmations
 */

import { PUBSUB_TOPICS, NetworkMessage } from "../config.js";
import type { TopicHandler, ValidationResult } from "./topics.js";
import {
  TruthMessage,
  EvidenceMessage,
  EvidenceRequestMessage,
  EvidenceResponseMessage,
  AttestationMessage,
  ChallengeMessage,
  ResolutionMessage,
  AcknowledgmentMessage,
  isEvidenceMessage,
  isEvidenceRequestMessage,
  isAttestationMessage,
  isChallengeMessage,
  isResolutionMessage,
  isAcknowledgmentMessage,
  validateTruthMessage,
  verifyMessageIntegrity,
} from "../../core/messaging/index.js";
import { verifyString } from "../../core/crypto/signatures.js";

// ============ Types ============

/**
 * Context for truth message handling
 */
export interface TruthMessageContext {
  /** Verify message signature */
  verifySignature: (message: TruthMessage) => Promise<boolean>;

  /** Check if sender is authorized for this message type */
  checkAuthorization: (sender: string, messageType: string) => Promise<boolean>;

  /** Check if sender has sufficient stake */
  checkStake: (sender: string, requiredStake: bigint) => Promise<boolean>;

  /** Get public key for a DID (hex-encoded) */
  getPublicKey: (did: string) => Promise<string | null>;
}

/**
 * Handler callbacks for each truth message type
 */
export interface TruthMessageHandlers {
  onEvidence?: (message: EvidenceMessage) => Promise<void>;
  onEvidenceRequest?: (message: EvidenceRequestMessage) => Promise<void>;
  onEvidenceResponse?: (message: EvidenceResponseMessage) => Promise<void>;
  onAttestation?: (message: AttestationMessage) => Promise<void>;
  onChallenge?: (message: ChallengeMessage) => Promise<void>;
  onResolution?: (message: ResolutionMessage) => Promise<void>;
  onAcknowledgment?: (message: AcknowledgmentMessage) => Promise<void>;
}

// ============ Decoder ============

/**
 * Decode a NetworkMessage payload to TruthMessage
 */
function decodeTruthMessage(message: NetworkMessage): TruthMessage | null {
  try {
    const text = new TextDecoder().decode(message.payload);
    const parsed = JSON.parse(text, (_key, value) => {
      // Revive BigInt values
      if (typeof value === "string" && /^\d+n$/.test(value)) {
        return BigInt(value.slice(0, -1));
      }
      return value;
    });
    return parsed as TruthMessage;
  } catch {
    return null;
  }
}

/**
 * Encode a TruthMessage to NetworkMessage payload
 */
export function encodeTruthMessage(message: TruthMessage): Uint8Array {
  const text = JSON.stringify(message, (_, value) => {
    // Convert BigInt to string with 'n' suffix
    if (typeof value === "bigint") {
      return value.toString() + "n";
    }
    return value;
  });
  return new TextEncoder().encode(text);
}

// ============ Base Validator ============

/**
 * Base validation for all truth messages
 */
async function validateBaseTruthMessage(
  message: TruthMessage,
  ctx: TruthMessageContext,
): Promise<{ valid: boolean; reason?: string }> {
  // Structural validation
  const structureResult = validateTruthMessage(message);
  if (!structureResult.valid) {
    return { valid: false, reason: structureResult.errors.join(", ") };
  }

  // Get sender's public key
  const publicKey = await ctx.getPublicKey(message.sender);
  if (!publicKey) {
    return { valid: false, reason: "Unknown sender" };
  }

  // Verify message integrity (signature, content hash, message ID)
  const integrityResult = verifyMessageIntegrity(
    message,
    publicKey,
    verifyString,
  );
  if (!integrityResult.valid) {
    return {
      valid: false,
      reason: `Integrity check failed: ${integrityResult.errors.join(", ")}`,
    };
  }

  // Authorization check
  const authorized = await ctx.checkAuthorization(
    message.sender,
    message.messageType,
  );
  if (!authorized) {
    return {
      valid: false,
      reason: "Sender not authorized for this message type",
    };
  }

  return { valid: true };
}

// ============ Topic Handlers ============

/**
 * Create evidence topic handler
 */
export function createEvidenceTopicHandler(
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.EVIDENCE,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      const truthMsg = decodeTruthMessage(message);
      if (!truthMsg) {
        return "reject";
      }

      if (!isEvidenceMessage(truthMsg)) {
        return "reject";
      }

      const baseResult = await validateBaseTruthMessage(truthMsg, ctx);
      if (!baseResult.valid) {
        console.warn(`Evidence validation failed: ${baseResult.reason}`);
        return "reject";
      }

      // Evidence-specific validation
      const payload = truthMsg.payload;
      if (payload.intrinsicStrength < 0 || payload.intrinsicStrength > 1) {
        return "reject";
      }

      return "accept";
    },

    async handle(message: NetworkMessage): Promise<void> {
      const truthMsg = decodeTruthMessage(message) as EvidenceMessage;
      if (handlers.onEvidence) {
        await handlers.onEvidence(truthMsg);
      }
    },
  };
}

/**
 * Create evidence request topic handler
 */
export function createEvidenceRequestTopicHandler(
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.EVIDENCE_REQUESTS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      const truthMsg = decodeTruthMessage(message);
      if (!truthMsg) {
        return "reject";
      }

      if (!isEvidenceRequestMessage(truthMsg)) {
        return "reject";
      }

      const baseResult = await validateBaseTruthMessage(truthMsg, ctx);
      if (!baseResult.valid) {
        console.warn(
          `Evidence request validation failed: ${baseResult.reason}`,
        );
        return "reject";
      }

      // Check bounty stake if present
      const payload = truthMsg.payload;
      if (payload.bounty && payload.bounty > 0n) {
        const hasStake = await ctx.checkStake(truthMsg.sender, payload.bounty);
        if (!hasStake) {
          return "reject";
        }
      }

      return "accept";
    },

    async handle(message: NetworkMessage): Promise<void> {
      const truthMsg = decodeTruthMessage(message) as EvidenceRequestMessage;
      if (handlers.onEvidenceRequest) {
        await handlers.onEvidenceRequest(truthMsg);
      }
    },
  };
}

/**
 * Create attestation topic handler
 */
export function createAttestationTopicHandler(
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.ATTESTATIONS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      const truthMsg = decodeTruthMessage(message);
      if (!truthMsg) {
        return "reject";
      }

      if (!isAttestationMessage(truthMsg)) {
        return "reject";
      }

      const baseResult = await validateBaseTruthMessage(truthMsg, ctx);
      if (!baseResult.valid) {
        console.warn(`Attestation validation failed: ${baseResult.reason}`);
        return "reject";
      }

      // Attestation-specific validation
      const payload = truthMsg.payload;
      if (payload.confidence < 0 || payload.confidence > 1) {
        return "reject";
      }

      // Verify stake
      if (payload.stake > 0n) {
        const hasStake = await ctx.checkStake(truthMsg.sender, payload.stake);
        if (!hasStake) {
          return "reject";
        }
      }

      return "accept";
    },

    async handle(message: NetworkMessage): Promise<void> {
      const truthMsg = decodeTruthMessage(message) as AttestationMessage;
      if (handlers.onAttestation) {
        await handlers.onAttestation(truthMsg);
      }
    },
  };
}

/**
 * Create challenge topic handler
 */
export function createChallengeTopicHandler(
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.CHALLENGES,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      const truthMsg = decodeTruthMessage(message);
      if (!truthMsg) {
        return "reject";
      }

      if (!isChallengeMessage(truthMsg)) {
        return "reject";
      }

      const baseResult = await validateBaseTruthMessage(truthMsg, ctx);
      if (!baseResult.valid) {
        console.warn(`Challenge validation failed: ${baseResult.reason}`);
        return "reject";
      }

      // Challenge requires stake
      const payload = truthMsg.payload;
      if (payload.stake <= 0n) {
        return "reject";
      }

      const hasStake = await ctx.checkStake(truthMsg.sender, payload.stake);
      if (!hasStake) {
        return "reject";
      }

      // Must have counter-evidence
      if (
        !payload.counterEvidenceIds ||
        payload.counterEvidenceIds.length === 0
      ) {
        return "reject";
      }

      return "accept";
    },

    async handle(message: NetworkMessage): Promise<void> {
      const truthMsg = decodeTruthMessage(message) as ChallengeMessage;
      if (handlers.onChallenge) {
        await handlers.onChallenge(truthMsg);
      }
    },
  };
}

/**
 * Create resolution topic handler
 */
export function createResolutionTopicHandler(
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.RESOLUTIONS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      const truthMsg = decodeTruthMessage(message);
      if (!truthMsg) {
        return "reject";
      }

      if (!isResolutionMessage(truthMsg)) {
        return "reject";
      }

      const baseResult = await validateBaseTruthMessage(truthMsg, ctx);
      if (!baseResult.valid) {
        console.warn(`Resolution validation failed: ${baseResult.reason}`);
        return "reject";
      }

      // Resolutions should only come from protocol/validators
      // (authorization check handles this)

      return "accept";
    },

    async handle(message: NetworkMessage): Promise<void> {
      const truthMsg = decodeTruthMessage(message) as ResolutionMessage;
      if (handlers.onResolution) {
        await handlers.onResolution(truthMsg);
      }
    },
  };
}

/**
 * Create acknowledgment topic handler
 */
export function createAcknowledgmentTopicHandler(
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): TopicHandler {
  return {
    topic: PUBSUB_TOPICS.ACKNOWLEDGMENTS,

    async validate(message: NetworkMessage): Promise<ValidationResult> {
      const truthMsg = decodeTruthMessage(message);
      if (!truthMsg) {
        return "reject";
      }

      if (!isAcknowledgmentMessage(truthMsg)) {
        return "reject";
      }

      const baseResult = await validateBaseTruthMessage(truthMsg, ctx);
      if (!baseResult.valid) {
        // Don't warn for acks - they're frequent and low-priority
        return "ignore";
      }

      return "accept";
    },

    async handle(message: NetworkMessage): Promise<void> {
      const truthMsg = decodeTruthMessage(message) as AcknowledgmentMessage;
      if (handlers.onAcknowledgment) {
        await handlers.onAcknowledgment(truthMsg);
      }
    },
  };
}

// ============ Registration Helper ============

/**
 * Register all truth topic handlers with a TopicManager
 */
export function registerTruthTopicHandlers(
  topicManager: { registerHandler: (handler: TopicHandler) => void },
  ctx: TruthMessageContext,
  handlers: TruthMessageHandlers,
): void {
  topicManager.registerHandler(createEvidenceTopicHandler(ctx, handlers));
  topicManager.registerHandler(
    createEvidenceRequestTopicHandler(ctx, handlers),
  );
  topicManager.registerHandler(createAttestationTopicHandler(ctx, handlers));
  topicManager.registerHandler(createChallengeTopicHandler(ctx, handlers));
  topicManager.registerHandler(createResolutionTopicHandler(ctx, handlers));
  topicManager.registerHandler(createAcknowledgmentTopicHandler(ctx, handlers));
}

/**
 * Get all truth topic names
 */
export function getTruthTopics(): string[] {
  return [
    PUBSUB_TOPICS.EVIDENCE,
    PUBSUB_TOPICS.EVIDENCE_REQUESTS,
    PUBSUB_TOPICS.ATTESTATIONS,
    PUBSUB_TOPICS.CHALLENGES,
    PUBSUB_TOPICS.RESOLUTIONS,
    PUBSUB_TOPICS.ACKNOWLEDGMENTS,
  ];
}
