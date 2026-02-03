/**
 * Consensus-Network Bridge
 *
 * Bridges the consensus engine to the network layer,
 * handling message broadcasting and event routing.
 */

import { EventEmitter } from 'events';
import type { LegibleNode } from '../node.js';
import type { ProofOfTruthEngine } from '../../consensus/pot/engine.js';
import type { Mempool } from '../../consensus/mempool/pool.js';
import { PUBSUB_TOPICS, MESSAGE_TYPES, type NetworkMessage } from '../config.js';

// ============ Types ============

export interface ConsensusBridgeConfig {
  node: LegibleNode;
  consensus: ProofOfTruthEngine;
  mempool: Mempool;
}

// ============ Bridge ============

/**
 * Adapter that wires consensus events to network broadcasts
 * and provides helper methods for broadcasting protocol messages
 */
export class ConsensusNetworkBridge extends EventEmitter {
  private node: LegibleNode;
  private consensus: ProofOfTruthEngine;
  private mempool: Mempool;
  private connected = false;

  constructor(config: ConsensusBridgeConfig) {
    super();
    this.node = config.node;
    this.consensus = config.consensus;
    this.mempool = config.mempool;
  }

  /**
   * Connect the bridge - wire up all event handlers
   */
  connect(): void {
    if (this.connected) return;

    this.wireConsensusToNetwork();
    this.wireMempoolToNetwork();

    this.connected = true;
    this.emit('connected');
  }

  /**
   * Disconnect the bridge - remove event handlers
   */
  disconnect(): void {
    if (!this.connected) return;

    // Note: In a full implementation, we'd store handler references
    // and remove them here. For now, handlers remain attached.

    this.connected = false;
    this.emit('disconnected');
  }

  // ============ Broadcast Methods ============

  /**
   * Broadcast a block to the network
   */
  async broadcastBlock(block: unknown): Promise<void> {
    const message = this.createMessage(MESSAGE_TYPES.NEW_BLOCK, block);
    await this.node.broadcast(PUBSUB_TOPICS.BLOCKS, message);
  }

  /**
   * Broadcast a transaction to the network
   */
  async broadcastTransaction(tx: unknown): Promise<void> {
    const message = this.createMessage(MESSAGE_TYPES.NEW_TRANSACTION, tx);
    await this.node.broadcast(PUBSUB_TOPICS.TRANSACTIONS, message);
  }

  /**
   * Broadcast a claim to the network
   */
  async broadcastClaim(claim: unknown): Promise<void> {
    const message = this.createMessage(MESSAGE_TYPES.NEW_CLAIM, claim);
    await this.node.broadcast(PUBSUB_TOPICS.CLAIMS, message);
  }

  /**
   * Broadcast a verification commitment
   */
  async broadcastCommitment(roundId: string, commitment: unknown): Promise<void> {
    const message = this.createMessage(MESSAGE_TYPES.VERIFICATION_COMMIT, {
      roundId,
      ...commitment as object,
    });
    await this.node.broadcast(PUBSUB_TOPICS.VERIFICATIONS, message);
  }

  /**
   * Broadcast a verification reveal
   */
  async broadcastReveal(roundId: string, reveal: unknown): Promise<void> {
    const message = this.createMessage(MESSAGE_TYPES.VERIFICATION_REVEAL, {
      roundId,
      ...reveal as object,
    });
    await this.node.broadcast(PUBSUB_TOPICS.VERIFICATIONS, message);
  }

  /**
   * Broadcast a consensus message
   */
  async broadcastConsensus(messageType: string, payload: unknown): Promise<void> {
    const message = this.createMessage(messageType, payload);
    await this.node.broadcast(PUBSUB_TOPICS.CONSENSUS, message);
  }

  // ============ Private Methods ============

  private wireConsensusToNetwork(): void {
    // Broadcast new blocks
    this.consensus.on('block_proposed', async (event: any) => {
      try {
        await this.broadcastBlock(event.block);
        this.emit('block:broadcast', event.block);
      } catch (error) {
        this.emit('error', { type: 'block_broadcast', error });
      }
    });

    // Broadcast verification round starts
    this.consensus.on('round_started', async (event: any) => {
      try {
        await this.broadcastClaim(event.round);
        this.emit('round:broadcast', event.round);
      } catch (error) {
        this.emit('error', { type: 'round_broadcast', error });
      }
    });
  }

  private wireMempoolToNetwork(): void {
    // Broadcast new transactions
    this.mempool.on('tx:added', async (entry: any) => {
      try {
        await this.broadcastTransaction(entry.tx);
        this.emit('tx:broadcast', entry.tx);
      } catch (error) {
        this.emit('error', { type: 'tx_broadcast', error });
      }
    });
  }

  private createMessage(type: string, payload: unknown): NetworkMessage {
    return {
      type,
      version: '1.0.0',
      timestamp: Date.now(),
      sender: this.node.getPeerId() || '',
      payload: new TextEncoder().encode(JSON.stringify(payload)),
    };
  }
}

// ============ Factory ============

/**
 * Create and connect a consensus-network bridge
 */
export function createConsensusBridge(config: ConsensusBridgeConfig): ConsensusNetworkBridge {
  const bridge = new ConsensusNetworkBridge(config);
  bridge.connect();
  return bridge;
}
