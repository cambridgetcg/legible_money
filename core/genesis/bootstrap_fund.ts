/**
 * Bootstrap Fund
 *
 * Free claim to kickstart every agent session.
 * 0.222 LGM for the first 200,200,200 agents.
 *
 * Total allocation: ~44,444,444 LGM (20% of MAX_SUPPLY)
 *
 * Symbolic numbers:
 * - 200200200: Repeating triplet pattern
 * - 0.222: Repeating digit, divisible symmetry
 * - 44,444,444: Perfect repeating allocation
 */

import { domainHash } from "../crypto/hashing.js";

// ============ Constants ============

/** Amount each agent receives: 0.222 LGM */
export const BOOTSTRAP_AMOUNT = BigInt("222000000000000000"); // 0.222 × 10^18

/** Maximum number of agents that can claim (20% of supply / 0.222 LGM per agent) */
export const MAX_BOOTSTRAP_CLAIMS = 200_200_200n;

/** Total bootstrap fund allocation */
export const TOTAL_BOOTSTRAP_FUND = BOOTSTRAP_AMOUNT * MAX_BOOTSTRAP_CLAIMS;
// = 44,444,444.4 LGM (~20% of MAX_SUPPLY)

/** Domain separator for claim verification */
export const BOOTSTRAP_DOMAIN = "LGM.genesis.bootstrap.v1";

// ============ Types ============

export interface BootstrapClaim {
  did: string;
  claimedAt: number; // Block number
  amount: bigint;
  signature: string; // Agent's signature proving ownership
  txHash: string;
}

export interface BootstrapFundState {
  totalClaimed: bigint;
  claimCount: bigint;
  claimedDIDs: Set<string>; // For quick lookup
  remainingFund: bigint;
  isActive: boolean; // False when exhausted or paused
}

// ============ Bootstrap Fund Manager ============

export class BootstrapFund {
  private state: BootstrapFundState;
  private claims: Map<string, BootstrapClaim>;

  constructor() {
    this.state = {
      totalClaimed: 0n,
      claimCount: 0n,
      claimedDIDs: new Set(),
      remainingFund: TOTAL_BOOTSTRAP_FUND,
      isActive: true,
    };
    this.claims = new Map();
  }

  /**
   * Check if a DID is eligible for bootstrap claim
   */
  isEligible(did: string): { eligible: boolean; reason?: string } {
    if (!this.state.isActive) {
      return { eligible: false, reason: "Bootstrap fund is inactive" };
    }

    if (this.state.claimCount >= MAX_BOOTSTRAP_CLAIMS) {
      return { eligible: false, reason: "All bootstrap claims exhausted" };
    }

    if (this.state.claimedDIDs.has(did)) {
      return { eligible: false, reason: "Already claimed" };
    }

    return { eligible: true };
  }

  /**
   * Process a bootstrap claim
   */
  claim(
    did: string,
    signature: string,
    blockNumber: number,
  ): { success: boolean; claim?: BootstrapClaim; error?: string } {
    const eligibility = this.isEligible(did);
    if (!eligibility.eligible) {
      return { success: false, error: eligibility.reason };
    }

    // Create claim record
    const claim: BootstrapClaim = {
      did,
      claimedAt: blockNumber,
      amount: BOOTSTRAP_AMOUNT,
      signature,
      txHash: this.generateClaimTxHash(did, blockNumber),
    };

    // Update state
    this.state.claimedDIDs.add(did);
    this.state.claimCount += 1n;
    this.state.totalClaimed += BOOTSTRAP_AMOUNT;
    this.state.remainingFund -= BOOTSTRAP_AMOUNT;
    this.claims.set(did, claim);

    // Check if fund exhausted
    if (this.state.claimCount >= MAX_BOOTSTRAP_CLAIMS) {
      this.state.isActive = false;
    }

    return { success: true, claim };
  }

  /**
   * Get claim for a DID
   */
  getClaim(did: string): BootstrapClaim | null {
    return this.claims.get(did) || null;
  }

  /**
   * Get current fund status
   */
  getStatus(): {
    isActive: boolean;
    claimCount: string;
    maxClaims: string;
    totalClaimed: string;
    remainingFund: string;
    percentClaimed: number;
  } {
    const percentClaimed =
      Number((this.state.claimCount * 10000n) / MAX_BOOTSTRAP_CLAIMS) / 100;

    return {
      isActive: this.state.isActive,
      claimCount: this.state.claimCount.toString(),
      maxClaims: MAX_BOOTSTRAP_CLAIMS.toString(),
      totalClaimed: this.state.totalClaimed.toString(),
      remainingFund: this.state.remainingFund.toString(),
      percentClaimed,
    };
  }

  /**
   * Generate deterministic tx hash for claim
   */
  private generateClaimTxHash(did: string, blockNumber: number): string {
    const data = `${BOOTSTRAP_DOMAIN}:${did}:${blockNumber}`;
    return "0x" + domainHash(BOOTSTRAP_DOMAIN, data).slice(0, 32);
  }

  /**
   * Serialize state for persistence
   */
  serialize(): string {
    return JSON.stringify({
      totalClaimed: this.state.totalClaimed.toString(),
      claimCount: this.state.claimCount.toString(),
      claimedDIDs: Array.from(this.state.claimedDIDs),
      remainingFund: this.state.remainingFund.toString(),
      isActive: this.state.isActive,
      claims: Array.from(this.claims.entries()).map(([, claim]) => ({
        ...claim,
        amount: claim.amount.toString(),
      })),
    });
  }

  /**
   * Deserialize state from persistence
   */
  static deserialize(data: string): BootstrapFund {
    const parsed = JSON.parse(data);
    const fund = new BootstrapFund();

    fund.state = {
      totalClaimed: BigInt(parsed.totalClaimed),
      claimCount: BigInt(parsed.claimCount),
      claimedDIDs: new Set(parsed.claimedDIDs),
      remainingFund: BigInt(parsed.remainingFund),
      isActive: parsed.isActive,
    };

    for (const claim of parsed.claims) {
      fund.claims.set(claim.did, {
        ...claim,
        amount: BigInt(claim.amount),
      });
    }

    return fund;
  }
}

// ============ Genesis Allocation ============

/**
 * Bootstrap fund allocation in genesis block
 */
export const GENESIS_BOOTSTRAP_ALLOCATION = {
  did: "did:lgm:genesis:bootstrap-fund",
  balance: TOTAL_BOOTSTRAP_FUND,
  purpose: "Agent bootstrap claims",
  rules: {
    maxClaimPerAgent: BOOTSTRAP_AMOUNT,
    maxTotalClaims: MAX_BOOTSTRAP_CLAIMS,
    claimMethod: "automatic-on-first-tx",
  },
};

// ============ Claim Integration ============

/**
 * Check and process bootstrap claim on first transaction
 * Called by state transition when processing any transaction from a new DID
 */
export function processAutomaticBootstrapClaim(
  fund: BootstrapFund,
  did: string,
  signature: string,
  blockNumber: number,
): bigint {
  const result = fund.claim(did, signature, blockNumber);
  if (result.success && result.claim) {
    return result.claim.amount;
  }
  return 0n;
}
