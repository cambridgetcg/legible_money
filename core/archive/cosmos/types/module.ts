/**
 * Cosmos SDK Module Types
 *
 * Defines the module interface for Legible Money on Cosmos SDK.
 * This provides the architecture for migration from the custom
 * implementation to a Cosmos SDK app chain.
 *
 * PP HIERARCHY:
 *   TRUTH: Module boundaries enforce separation of concerns
 *   UNDERSTANDING: Clear interfaces for each component
 *   BEAUTY: Elegant mapping from existing to SDK patterns
 *   JUSTICE: Standard SDK patterns for interoperability
 *   CREATIVITY: Novel PoT consensus as SDK module
 */

// ============ Module Names ============

export const MODULE_NAMES = {
  /** Core LGM module (token, accounts) */
  LGM: "lgm",

  /** Knowledge management (facts, verification) */
  KNOWLEDGE: "knowledge",

  /** Research funding (bounties, treasury) */
  RESEARCH: "research",

  /** Evidence management */
  EVIDENCE: "evidence",

  /** Dispute resolution */
  DISPUTE: "dispute",

  /** Reward vesting */
  VESTING: "vesting",
} as const;

export type ModuleName = (typeof MODULE_NAMES)[keyof typeof MODULE_NAMES];

// ============ Store Keys ============

/**
 * Store key prefixes for IAVL tree
 */
export const STORE_KEYS = {
  // LGM module
  LGM_ACCOUNT: "lgm/account/",
  LGM_BALANCE: "lgm/balance/",
  LGM_STAKED: "lgm/staked/",
  LGM_VALIDATOR: "lgm/validator/",

  // Knowledge module
  KNOWLEDGE_FACT: "knowledge/fact/",
  KNOWLEDGE_DOMAIN: "knowledge/domain/",
  KNOWLEDGE_CLAIM: "knowledge/claim/",
  KNOWLEDGE_ROUND: "knowledge/round/",

  // Research module
  RESEARCH_BOUNTY: "research/bounty/",
  RESEARCH_TREASURY: "research/treasury/",
  RESEARCH_CONTRIBUTION: "research/contribution/",

  // Evidence module
  EVIDENCE_ITEM: "evidence/item/",
  EVIDENCE_ARCHIVE: "evidence/archive/",

  // Dispute module
  DISPUTE_CASE: "dispute/case/",
  DISPUTE_VOTE: "dispute/vote/",

  // Vesting module
  VESTING_SCHEDULE: "vesting/schedule/",
  VESTING_CLAIM: "vesting/claim/",
} as const;

// ============ Module Interface ============

/**
 * Base module interface (maps to Cosmos SDK AppModule)
 */
export interface LegibleModule {
  /** Module name */
  name: ModuleName;

  /** Route for message handling */
  route: string;

  /** Query routes */
  queryRoutes: string[];

  /** Begin block hook */
  beginBlock?(ctx: BlockContext): Promise<void>;

  /** End block hook */
  endBlock?(ctx: BlockContext): Promise<EndBlockResponse>;

  /** Init genesis */
  initGenesis(ctx: GenesisContext, state: unknown): void;

  /** Export genesis */
  exportGenesis(ctx: GenesisContext): unknown;
}

/**
 * Block context (simplified from Cosmos SDK Context)
 */
export interface BlockContext {
  /** Block height */
  blockHeight: bigint;

  /** Block time (Unix timestamp) */
  blockTime: number;

  /** Chain ID */
  chainId: string;

  /** Transaction hash (if in tx context) */
  txHash?: string;

  /** Event manager */
  eventManager: EventManager;

  /** Logger */
  logger: Logger;
}

/**
 * Genesis context
 */
export interface GenesisContext {
  /** Chain ID */
  chainId: string;

  /** Genesis time */
  genesisTime: number;

  /** Initial height */
  initialHeight: bigint;
}

/**
 * End block response
 */
export interface EndBlockResponse {
  /** Validator updates */
  validatorUpdates: ValidatorUpdate[];

  /** Consensus param updates */
  consensusParamUpdates?: ConsensusParamUpdates;
}

export interface ValidatorUpdate {
  publicKey: string;
  power: bigint;
}

export interface ConsensusParamUpdates {
  block?: {
    maxBytes?: bigint;
    maxGas?: bigint;
  };
}

// ============ Event System ============

/**
 * Event manager for emitting module events
 */
export interface EventManager {
  /** Emit an event */
  emit(event: ModuleEvent): void;

  /** Get all events */
  events(): ModuleEvent[];
}

/**
 * Module event (maps to Cosmos SDK Event)
 */
export interface ModuleEvent {
  /** Event type (e.g., "transfer", "fact_accepted") */
  type: string;

  /** Event attributes */
  attributes: EventAttribute[];
}

export interface EventAttribute {
  key: string;
  value: string;
  index?: boolean;
}

/**
 * Standard event types
 */
export const EVENT_TYPES = {
  // LGM events
  TRANSFER: "transfer",
  STAKE: "stake",
  UNSTAKE: "unstake",
  SLASH: "slash",

  // Knowledge events
  CLAIM_SUBMITTED: "claim_submitted",
  VERIFICATION_STARTED: "verification_started",
  FACT_ACCEPTED: "fact_accepted",
  FACT_REJECTED: "fact_rejected",

  // Research events
  BOUNTY_CREATED: "bounty_created",
  BOUNTY_CLAIMED: "bounty_claimed",
  RESEARCH_FUNDED: "research_funded",

  // Evidence events
  EVIDENCE_SUBMITTED: "evidence_submitted",
  EVIDENCE_ARCHIVED: "evidence_archived",

  // Dispute events
  DISPUTE_INITIATED: "dispute_initiated",
  DISPUTE_RESOLVED: "dispute_resolved",

  // Vesting events
  VESTING_CREATED: "vesting_created",
  REWARDS_CLAIMED: "rewards_claimed",
  VESTING_FALSIFIED: "vesting_falsified",
} as const;

// ============ Logger ============

export interface Logger {
  debug(msg: string, ...args: unknown[]): void;
  info(msg: string, ...args: unknown[]): void;
  warn(msg: string, ...args: unknown[]): void;
  error(msg: string, ...args: unknown[]): void;
}

// ============ Keeper Interface ============

/**
 * Base keeper interface
 */
export interface Keeper {
  /** Get the store key for this keeper */
  storeKey(): string;
}

/**
 * Account keeper interface
 */
export interface AccountKeeper extends Keeper {
  getAccount(ctx: BlockContext, address: string): Promise<Account | null>;
  setAccount(ctx: BlockContext, account: Account): Promise<void>;
  getBalance(
    ctx: BlockContext,
    address: string,
    denom: string,
  ): Promise<bigint>;
  setBalance(
    ctx: BlockContext,
    address: string,
    denom: string,
    amount: bigint,
  ): Promise<void>;
}

/**
 * Bank keeper interface (for token transfers)
 */
export interface BankKeeper extends Keeper {
  sendCoins(
    ctx: BlockContext,
    from: string,
    to: string,
    amount: Coin[],
  ): Promise<void>;
  mintCoins(
    ctx: BlockContext,
    moduleName: string,
    amount: Coin[],
  ): Promise<void>;
  burnCoins(
    ctx: BlockContext,
    moduleName: string,
    amount: Coin[],
  ): Promise<void>;
}

/**
 * Staking keeper interface
 */
export interface StakingKeeper extends Keeper {
  getValidator(ctx: BlockContext, address: string): Promise<Validator | null>;
  delegate(
    ctx: BlockContext,
    delegator: string,
    validator: string,
    amount: bigint,
  ): Promise<void>;
  undelegate(
    ctx: BlockContext,
    delegator: string,
    validator: string,
    amount: bigint,
  ): Promise<void>;
  slash(ctx: BlockContext, validator: string, fraction: number): Promise<void>;
}

// ============ Common Types ============

export interface Account {
  address: string;
  pubKey?: string;
  sequence: bigint;
  accountNumber: bigint;
}

export interface Coin {
  denom: string;
  amount: bigint;
}

export interface Validator {
  operatorAddress: string;
  consensusPubKey: string;
  status: "bonded" | "unbonding" | "unbonded";
  tokens: bigint;
  delegatorShares: bigint;
  commission: number;
  jailed: boolean;
}

// ============ Module Account ============

/**
 * Module accounts for holding module funds
 */
export const MODULE_ACCOUNTS = {
  /** Fee collector */
  FEE_COLLECTOR: `${MODULE_NAMES.LGM}/fee_collector`,

  /** Research treasury */
  RESEARCH_TREASURY: `${MODULE_NAMES.RESEARCH}/treasury`,

  /** Vesting pool */
  VESTING_POOL: `${MODULE_NAMES.VESTING}/pool`,

  /** Dispute escrow */
  DISPUTE_ESCROW: `${MODULE_NAMES.DISPUTE}/escrow`,

  /** Burn address */
  BURN: `${MODULE_NAMES.LGM}/burn`,
} as const;

// ============ Params ============

/**
 * Module parameters (can be updated via governance)
 */
export interface LGMParams {
  /** Block reward */
  blockReward: bigint;

  /** Minimum stake for validators */
  minValidatorStake: bigint;

  /** Minimum stake for claims */
  minClaimStake: bigint;

  /** Research share rate (0-1) */
  researchShareRate: number;

  /** Burn rate (0-1) */
  burnRate: number;
}

export interface KnowledgeParams {
  /** Verification round duration (blocks) */
  verificationRoundBlocks: bigint;

  /** Acceptance threshold */
  acceptanceThreshold: number;

  /** Rejection threshold */
  rejectionThreshold: number;

  /** Maximum claim length */
  maxClaimLength: number;

  /** Maximum references per claim */
  maxReferences: number;
}

export interface ResearchParams {
  /** Minimum bounty amount */
  minBountyAmount: bigint;

  /** Maximum bounty duration (blocks) */
  maxBountyDuration: bigint;

  /** Research confidence boost */
  researchConfidenceBoost: number;
}

export interface VestingParams {
  /** Vesting half-lives by category */
  halfLives: Record<string, bigint>;

  /** Cliff periods by category */
  cliffPeriods: Record<string, bigint>;

  /** Max release rates by category */
  maxReleaseRates: Record<string, number>;
}

// ============ Genesis State ============

export interface LGMGenesisState {
  params: LGMParams;
  accounts: Account[];
  balances: Array<{ address: string; coins: Coin[] }>;
  validators: Validator[];
}

export interface KnowledgeGenesisState {
  params: KnowledgeParams;
  facts: Array<{
    id: string;
    claim: string;
    domain: string;
    confidence: number;
    submitter: string;
  }>;
  axioms: Array<{
    id: string;
    claim: string;
    domain: string;
  }>;
}

export interface ResearchGenesisState {
  params: ResearchParams;
  treasuryBalance: bigint;
  bounties: Array<{
    id: string;
    description: string;
    reward: bigint;
    deadline: bigint;
  }>;
}

// ============ Full Genesis ============

export interface LegibleGenesisState {
  lgm: LGMGenesisState;
  knowledge: KnowledgeGenesisState;
  research: ResearchGenesisState;
}
