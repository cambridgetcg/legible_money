/**
 * Participation Bridge
 *
 * Bridges state layer data to the token module's ParticipationMetrics interface.
 * This is the dependency inversion point: state/ knows about core/tokens/ types,
 * but core/tokens/ does NOT know about state/.
 *
 * Usage:
 *   const resolver = createStateParticipationResolver(stateReader);
 *   childAccountManager.setParticipationResolver(resolver);
 */

import type { DID } from "../core/agents/identity/did.js";
import type {
  ParticipationMetrics,
  ParticipationConfig,
  ParticipationResolver,
} from "../core/tokens/types.js";
import { createParticipationResolver } from "../core/tokens/participation.js";
import { DEFAULT_PARTICIPATION_CONFIG } from "../core/tokens/types.js";
import type { AccountState, ValidatorState } from "./types.js";

/**
 * Interface for reading state needed for participation metrics.
 * Keeps this module testable without importing the full StateManager.
 */
export interface ParticipationStateReader {
  getAccountState(did: string): AccountState | undefined;
  getValidatorState(did: string): ValidatorState | undefined;
  getCurrentBlock(): bigint;
  getClaimCount?(did: string): { submitted: number; accepted: number };
  getQueryCount?(did: string): number;
  getGovernanceVoteCount?(did: string): number;
}

/**
 * Build ParticipationMetrics from the state layer for a given account.
 * Returns null if account is not found in state.
 */
export function buildParticipationMetrics(
  account: DID,
  reader: ParticipationStateReader,
): ParticipationMetrics | null {
  const accountState = reader.getAccountState(account);
  if (!accountState) return null;

  const currentBlock = reader.getCurrentBlock();
  const validatorState = reader.getValidatorState(account);

  const claims = reader.getClaimCount?.(account) ?? {
    submitted: 0,
    accepted: 0,
  };
  const knowledgeQueriesMade = reader.getQueryCount?.(account) ?? 0;
  const governanceVotes = reader.getGovernanceVoteCount?.(account) ?? 0;

  return {
    claimsSubmitted: claims.submitted,
    claimsAccepted: claims.accepted,
    verificationsCompleted: validatorState?.verificationsCompleted ?? 0,
    verificationsCorrect: validatorState?.verificationsCorrect ?? 0,
    knowledgeQueriesMade,
    governanceVotes,
    activeStake: accountState.stakedBalance,
    lastActivityBlock: accountState.lastActiveBlock,
    currentBlock,
    accountCreatedBlock: 0n, // TODO: Track account creation block in AccountState
  };
}

/**
 * Create a ParticipationResolver wired to the state layer.
 */
export function createStateParticipationResolver(
  reader: ParticipationStateReader,
  config: ParticipationConfig = DEFAULT_PARTICIPATION_CONFIG,
  cacheSize: number = 1000,
  cacheTtlMs: number = 60_000,
): ParticipationResolver {
  return createParticipationResolver(
    (account: DID) => buildParticipationMetrics(account, reader),
    config,
    cacheSize,
    cacheTtlMs,
  );
}
