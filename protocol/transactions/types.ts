import { z } from 'zod';
import type { DID } from '../../agents/identity/did.js';
import type { Signature } from '../../core/crypto/keys.js';

/**
 * Base transaction schema
 */
export const BaseTransactionSchema = z.object({
  version: z.literal(1),
  nonce: z.number().int().nonnegative(),
  timestamp: z.number().int().positive(),
  sender: z.string(), // DID
  fee: z.bigint().nonnegative(),
});

/**
 * Transfer AGC tokens between agents
 */
export const TransferTxSchema = BaseTransactionSchema.extend({
  type: z.literal('transfer'),
  payload: z.object({
    to: z.string(), // DID
    amount: z.bigint().positive(),
    memo: z.string().max(256).optional(),
  }),
});

/**
 * Register a new agent identity
 */
export const RegisterAgentTxSchema = BaseTransactionSchema.extend({
  type: z.literal('register_agent'),
  payload: z.object({
    publicKey: z.instanceof(Uint8Array),
    capabilities: z.array(z.string()),
    metadata: z.object({
      name: z.string().max(64).optional(),
      description: z.string().max(256).optional(),
      endpoint: z.string().url().optional(),
    }),
  }),
});

/**
 * Stake tokens for validation or service guarantees
 */
export const StakeTxSchema = BaseTransactionSchema.extend({
  type: z.literal('stake'),
  payload: z.object({
    amount: z.bigint().positive(),
    purpose: z.enum(['validation', 'service_guarantee', 'governance']),
    lockPeriod: z.number().int().positive(), // blocks
  }),
});

/**
 * Unstake previously staked tokens
 */
export const UnstakeTxSchema = BaseTransactionSchema.extend({
  type: z.literal('unstake'),
  payload: z.object({
    stakeId: z.string(),
    amount: z.bigint().positive(),
  }),
});

/**
 * Create a market order (buy/sell resources)
 */
export const MarketOrderTxSchema = BaseTransactionSchema.extend({
  type: z.literal('market_order'),
  payload: z.object({
    market: z.enum(['compute', 'data', 'services']),
    side: z.enum(['buy', 'sell']),
    resourceType: z.string(),
    quantity: z.bigint().positive(),
    price: z.bigint().positive(), // price per unit in AGC
    expiry: z.number().int().positive(), // block height
  }),
});

/**
 * Fill a market order
 */
export const FillOrderTxSchema = BaseTransactionSchema.extend({
  type: z.literal('fill_order'),
  payload: z.object({
    orderId: z.string(),
    quantity: z.bigint().positive(),
  }),
});

/**
 * Submit reputation attestation
 */
export const ReputationTxSchema = BaseTransactionSchema.extend({
  type: z.literal('reputation'),
  payload: z.object({
    target: z.string(), // DID of agent being rated
    transactionId: z.string(), // Reference to completed transaction
    score: z.number().int().min(-1).max(1), // -1, 0, or 1
    evidence: z.string().max(512).optional(),
  }),
});

/**
 * Governance vote
 */
export const VoteTxSchema = BaseTransactionSchema.extend({
  type: z.literal('vote'),
  payload: z.object({
    proposalId: z.string(),
    vote: z.enum(['yes', 'no', 'abstain']),
    weight: z.bigint().optional(), // If not provided, uses full stake
  }),
});

/**
 * Union of all transaction types
 */
export const TransactionSchema = z.discriminatedUnion('type', [
  TransferTxSchema,
  RegisterAgentTxSchema,
  StakeTxSchema,
  UnstakeTxSchema,
  MarketOrderTxSchema,
  FillOrderTxSchema,
  ReputationTxSchema,
  VoteTxSchema,
]);

export type Transaction = z.infer<typeof TransactionSchema>;
export type TransferTx = z.infer<typeof TransferTxSchema>;
export type RegisterAgentTx = z.infer<typeof RegisterAgentTxSchema>;
export type StakeTx = z.infer<typeof StakeTxSchema>;
export type MarketOrderTx = z.infer<typeof MarketOrderTxSchema>;

/**
 * Signed transaction wrapper
 */
export interface SignedTransaction {
  transaction: Transaction;
  signature: Signature;
  hash: string;
}

/**
 * Transaction receipt (after inclusion in block)
 */
export interface TransactionReceipt {
  hash: string;
  blockHeight: number;
  blockHash: string;
  index: number;
  status: 'success' | 'failed';
  gasUsed: bigint;
  events: TransactionEvent[];
}

/**
 * Events emitted by transactions
 */
export interface TransactionEvent {
  type: string;
  data: Record<string, unknown>;
}
