/**
 * Live Knowledge Service — bridges BVM to StateManager
 *
 * Implements BVMKnowledgeService by delegating read-only queries
 * to StateManager's knowledge layer. Write operations (submitClaim,
 * commitVote, revealVote, initiateChallenge) return null/false since
 * those flow through the PoT consensus engine via RPC, not from
 * inside BVM execution.
 */

import type { StateManager } from "../../state/state_manager.js";
import type { BVMKnowledgeService } from "./types.js";
import type { DID } from "../agents/identity/did.js";
import type { Confidence } from "../knowledge/types.js";

export class LiveKnowledgeService implements BVMKnowledgeService {
  constructor(private stateManager: StateManager) {}

  queryFact(factId: string): {
    exists: boolean;
    confidence: Confidence;
    domain: string;
  } | null {
    const fact = this.stateManager.getFact(factId);
    if (!fact) {
      return null;
    }
    return {
      exists: true,
      confidence: fact.confidence,
      domain: fact.domain,
    };
  }

  queryDomain(
    domain: string,
    limit: number,
  ): { count: number; factIds: string[] } {
    const facts = this.stateManager.getFactsByDomain(domain);
    const factIds = facts.slice(0, limit).map((f) => f.id);
    return {
      count: facts.length,
      factIds,
    };
  }

  getNetworkConfidence(factId: string): Confidence {
    const fact = this.stateManager.getFact(factId);
    return fact?.confidence ?? 0;
  }

  // Write operations are handled externally via PoT consensus RPC.
  // Contracts cannot submit claims from inside BVM execution.

  submitClaim(
    _callerDID: DID,
    _domain: string,
    _dataHash: string,
    _evidenceHash: string,
    _blockNumber: bigint,
  ): { claimId: string } | null {
    return null;
  }

  commitVote(
    _callerDID: DID,
    _claimId: string,
    _voteHash: string,
    _blockNumber: bigint,
  ): boolean {
    return false;
  }

  revealVote(
    _callerDID: DID,
    _claimId: string,
    _vote: bigint,
    _salt: bigint,
    _blockNumber: bigint,
  ): boolean {
    return false;
  }

  initiateChallenge(
    _callerDID: DID,
    _factId: string,
    _evidenceHash: string,
    _stake: bigint,
    _blockNumber: bigint,
  ): { challengeId: string } | null {
    return null;
  }
}
