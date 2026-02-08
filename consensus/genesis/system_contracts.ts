/**
 * Genesis System Contracts
 *
 * Registers protocol-intrinsic self-invoking contracts with the SPU at genesis.
 * These contracts are part of the protocol itself — not user-deployable.
 *
 * Currently registers:
 * 1. RAI Retrieval Schedule — sweeps expired token balances to RAI holding
 * 2. RAI Distribution Schedule — distributes RAI to eligible agents
 *
 * Both use periodic time conditions matching the token expiration period (7 days).
 * Distribution is offset 1 hour after retrieval to ensure funds are available.
 */

import { sha512_256Hex } from "../../core/crypto/hashing.js";
import { SYSTEM_CONTRACTS } from "../../core/constants.js";
import { DEFAULT_EXPIRATION_CONFIG } from "../../core/tokens/types.js";
import type { AutonomyIntegration } from "../pot/autonomy_integration.js";
import type { ScheduleMessage } from "../../core/schedule/types.js";
import type { Signature } from "../../core/crypto/keys.js";

/**
 * Distribution schedule offset from retrieval (1 hour in ms).
 * Ensures retrieval completes before distribution attempts to spend RAI funds.
 */
const DISTRIBUTION_OFFSET_MS = 3_600_000;

/**
 * Register all genesis system contracts with the SPU.
 *
 * Must be called once after AutonomyIntegration is configured on the PoT engine.
 * Idempotent registration — SPU deduplicates by messageId.
 */
export async function registerGenesisContracts(
  autonomy: AutonomyIntegration,
  genesisTimestamp: number,
): Promise<{ retrievalProcessId: string; distributionProcessId: string }> {
  const spu = autonomy.getSPU();
  const periodMs = DEFAULT_EXPIRATION_CONFIG.periodDurationMs;

  // Schedule 1: Sweep expired token balances → RAI holding account
  const retrievalMessage = createSystemScheduleMessage({
    messageId: sha512_256Hex(
      `LGM.genesis.schedule.v1:${SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID}:sweep_expired`,
    ),
    target: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
    data: SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
    scheduleType: "expiration_clawback",
    interval: periodMs,
    startTime: genesisTimestamp + periodMs,
    timestamp: genesisTimestamp,
  });

  const retrievalProcess = await spu.processScheduleMessage(retrievalMessage);

  // Schedule 2: Distribute RAI to eligible agents
  const distributionMessage = createSystemScheduleMessage({
    messageId: sha512_256Hex(
      `LGM.genesis.schedule.v1:${SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID}:distribute_rai`,
    ),
    target: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
    data: SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
    scheduleType: "rai_distribution",
    interval: periodMs,
    startTime: genesisTimestamp + periodMs + DISTRIBUTION_OFFSET_MS,
    timestamp: genesisTimestamp,
  });

  const distributionProcess =
    await spu.processScheduleMessage(distributionMessage);

  return {
    retrievalProcessId: retrievalProcess.id,
    distributionProcessId: distributionProcess.id,
  };
}

// ============ Internal ============

/**
 * Build a ScheduleMessage for a system contract.
 *
 * System messages use "system" as sender — SPU bypasses signature verification
 * for this sender address (processing_unit.ts:215).
 */
function createSystemScheduleMessage(opts: {
  messageId: string;
  target: string;
  data: Uint8Array;
  scheduleType: ScheduleMessage["scheduleType"];
  interval: number;
  startTime: number;
  timestamp: number;
}): ScheduleMessage {
  const identifier = new Uint8Array(32);
  const idBytes = new TextEncoder().encode(opts.messageId);
  identifier.set(idBytes.slice(0, 32));

  const emptySignature: Signature = "0".repeat(128) as Signature;

  return {
    messageType: "schedule",
    messageId: opts.messageId,
    identifier,
    nonce: 0n,
    senderAddress: "system",
    recipientAddress: opts.target,
    networkFeeUnit: 1n,
    networkFeeLimit: 100_000n,
    transferValue: 0n,
    data: opts.data,
    scheduleCondition: {
      time: {
        type: "periodic",
        interval: opts.interval,
        startTime: opts.startTime,
      },
    },
    signature: emptySignature,
    timestamp: opts.timestamp,
    scheduleType: opts.scheduleType,
    modifiable: false,
    perpetual: true,
    maxExecutions: undefined,
    prepaidFee: 2n ** 63n, // Effectively unlimited for system contracts
  };
}
