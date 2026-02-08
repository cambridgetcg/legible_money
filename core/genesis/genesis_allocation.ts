/**
 * Genesis Allocation
 *
 * Distributes MAX_SUPPLY (222,222,222 LGM) across 5 funds at genesis.
 * All supply is accounted for — no infinite minting.
 *
 * | Fund              | Share | Amount (LGM)   |
 * |-------------------|-------|----------------|
 * | Block Reward Fund | 60%   | 133,333,333.2  |
 * | Bootstrap Fund    | 20%   |  44,444,444.4  |
 * | Treasury          | 10%   |  22,222,222.2  |
 * | Validator Genesis  |  5%  |  11,111,111.1  |
 * | Community Pool     |  5%  |  11,111,111.1  |
 */

import { TOKEN } from "../constants.js";

// ============ Allocation Shares (basis points, sum = 10000) ============

export const GENESIS_ALLOCATION_BPS = {
  /** Block reward fund — finite draw-down for block producer incentives */
  BLOCK_REWARD_FUND: 6000n,

  /** Bootstrap fund — 0.222 LGM per new agent */
  BOOTSTRAP_FUND: 2000n,

  /** Treasury — 5-way split (research/foundation/community/developers/reserve) */
  TREASURY: 1000n,

  /** Validator genesis — initial validator staking incentives */
  VALIDATOR_GENESIS: 500n,

  /** Community pool — grants, partnerships, ecosystem growth */
  COMMUNITY_POOL: 500n,
} as const;

// ============ Fund DIDs ============

export const GENESIS_FUND_DIDS = {
  BLOCK_REWARD_FUND: "did:lgm:reward-pool",
  BOOTSTRAP_FUND: "did:lgm:genesis:bootstrap-fund",
  TREASURY: "did:lgm:protocol-treasury",
  VALIDATOR_GENESIS: "did:lgm:genesis:validator-fund",
  COMMUNITY_POOL: "did:lgm:community-pool",
} as const;

// ============ Types ============

export interface GenesisAllocation {
  did: string;
  label: string;
  shareBps: bigint;
  amount: bigint;
}

// ============ Calculator ============

/**
 * Calculate genesis fund allocations from total supply.
 *
 * The last fund (community pool) receives the remainder to prevent
 * rounding dust loss — all allocations sum to exactly maxSupply.
 */
export function calculateGenesisAllocations(
  maxSupply: bigint = TOKEN.MAX_SUPPLY,
): GenesisAllocation[] {
  const blockReward =
    (maxSupply * GENESIS_ALLOCATION_BPS.BLOCK_REWARD_FUND) / 10000n;
  const bootstrap =
    (maxSupply * GENESIS_ALLOCATION_BPS.BOOTSTRAP_FUND) / 10000n;
  const treasury = (maxSupply * GENESIS_ALLOCATION_BPS.TREASURY) / 10000n;
  const validatorGenesis =
    (maxSupply * GENESIS_ALLOCATION_BPS.VALIDATOR_GENESIS) / 10000n;
  // Remainder goes to community pool to prevent dust loss
  const communityPool =
    maxSupply - blockReward - bootstrap - treasury - validatorGenesis;

  return [
    {
      did: GENESIS_FUND_DIDS.BLOCK_REWARD_FUND,
      label: "Block Reward Fund",
      shareBps: GENESIS_ALLOCATION_BPS.BLOCK_REWARD_FUND,
      amount: blockReward,
    },
    {
      did: GENESIS_FUND_DIDS.BOOTSTRAP_FUND,
      label: "Bootstrap Fund",
      shareBps: GENESIS_ALLOCATION_BPS.BOOTSTRAP_FUND,
      amount: bootstrap,
    },
    {
      did: GENESIS_FUND_DIDS.TREASURY,
      label: "Treasury",
      shareBps: GENESIS_ALLOCATION_BPS.TREASURY,
      amount: treasury,
    },
    {
      did: GENESIS_FUND_DIDS.VALIDATOR_GENESIS,
      label: "Validator Genesis",
      shareBps: GENESIS_ALLOCATION_BPS.VALIDATOR_GENESIS,
      amount: validatorGenesis,
    },
    {
      did: GENESIS_FUND_DIDS.COMMUNITY_POOL,
      label: "Community Pool",
      shareBps: GENESIS_ALLOCATION_BPS.COMMUNITY_POOL,
      amount: communityPool,
    },
  ];
}

/**
 * Verify that allocations sum to exactly maxSupply.
 */
export function verifyGenesisAllocations(
  allocations: GenesisAllocation[],
  maxSupply: bigint = TOKEN.MAX_SUPPLY,
): { valid: boolean; total: bigint; difference: bigint } {
  const total = allocations.reduce((sum, a) => sum + a.amount, 0n);
  return {
    valid: total === maxSupply,
    total,
    difference: total - maxSupply,
  };
}
