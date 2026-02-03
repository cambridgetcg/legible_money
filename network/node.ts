/**
 * Network Node
 *
 * libp2p node setup and management for Legible Money P2P network.
 */

import { EventEmitter } from 'events';
import { createLibp2p, Libp2p } from 'libp2p';
import { tcp } from '@libp2p/tcp';
import { webSockets } from '@libp2p/websockets';
import { noise } from '@chainsafe/libp2p-noise';
import { yamux } from '@chainsafe/libp2p-yamux';
import { gossipsub, GossipSub } from '@chainsafe/libp2p-gossipsub';
import { kadDHT } from '@libp2p/kad-dht';
import { mdns } from '@libp2p/mdns';
import { bootstrap } from '@libp2p/bootstrap';
import { identify } from '@libp2p/identify';
import { multiaddr } from '@multiformats/multiaddr';
import { pipe } from 'it-pipe';
import * as lp from 'it-length-prefixed';
import type { Stream } from '@libp2p/interface';
import type { PeerId as Libp2pPeerId } from '@libp2p/interface';
import {
  NetworkConfig,
  DEFAULT_NETWORK_CONFIG,
  PUBSUB_TOPICS,
  NetworkMessage,
} from './config.js';
import type { TopicManager, ValidationResult } from './pubsub/topics.js';
import type { SyncManager } from './protocols/sync.js';

// ============ Types ============

export type PeerId = string;
export type Multiaddr = string;

export interface PeerInfo {
  peerId: PeerId;
  addresses: Multiaddr[];
  protocols: string[];
  latency: number;
  score: number;
  connectedAt: number;
}

export interface NetworkStats {
  peerId: PeerId;
  connectedPeers: number;
  knownPeers: number;
  bytesReceived: bigint;
  bytesSent: bigint;
  messagesReceived: number;
  messagesSent: number;
  uptime: number;
}

export interface ConnectionEvent {
  type: 'connect' | 'disconnect';
  peer: PeerInfo;
  timestamp: number;
}

export interface MessageEvent {
  type: string;
  topic?: string;
  from: PeerId;
  message: NetworkMessage;
  timestamp: number;
}

// ============ Network Node ============

/**
 * Legible Money network node
 *
 * Full libp2p implementation for P2P networking.
 */
export class LegibleNode extends EventEmitter {
  private config: NetworkConfig;
  private started: boolean = false;
  private peerId: PeerId | null = null;
  private peers: Map<PeerId, PeerInfo> = new Map();
  private stats: NetworkStats;

  // libp2p instance
  private libp2p: Libp2p | null = null;

  // Component references for message routing
  private topicManager: TopicManager | null = null;
  private syncManager: SyncManager | null = null;

  constructor(config: Partial<NetworkConfig> = {}) {
    super();
    this.config = { ...DEFAULT_NETWORK_CONFIG, ...config };
    this.stats = this.initStats();
  }

  // ============ Lifecycle ============

  /**
   * Start the network node
   */
  async start(): Promise<void> {
    if (this.started) {
      throw new Error('Node already started');
    }

    console.log('[Network] Starting Legible Money node...');

    // Initialize libp2p
    await this.initLibp2p();

    // Setup event handlers
    this.setupLibp2pEvents();

    // Subscribe to protocol topics
    await this.subscribeTopics();

    this.started = true;
    this.stats.uptime = Date.now();
    this.emit('started', { peerId: this.peerId });

    console.log(`[Network] Node started with peer ID: ${this.peerId}`);

    // Log listen addresses
    const addrs = this.getListenAddresses();
    for (const addr of addrs) {
      console.log(`[Network] Listening on: ${addr}`);
    }
  }

  /**
   * Stop the network node
   */
  async stop(): Promise<void> {
    if (!this.started) return;

    console.log('[Network] Stopping node...');

    // Stop libp2p
    if (this.libp2p) {
      await this.libp2p.stop();
      this.libp2p = null;
    }

    this.peers.clear();
    this.started = false;
    this.emit('stopped');

    console.log('[Network] Node stopped');
  }

  /**
   * Check if node is running
   */
  isStarted(): boolean {
    return this.started;
  }

  // ============ Component Integration ============

  /**
   * Set the topic manager for message routing
   */
  setTopicManager(topicManager: TopicManager): void {
    this.topicManager = topicManager;
  }

  /**
   * Set the sync manager for block synchronization
   */
  setSyncManager(syncManager: SyncManager): void {
    this.syncManager = syncManager;
  }

  /**
   * Register a custom protocol handler
   */
  async registerProtocol(
    protocol: string,
    handler: (stream: Stream, peerId: string) => Promise<void>
  ): Promise<void> {
    if (!this.libp2p) {
      throw new Error('Node not initialized');
    }

    await this.libp2p.handle(protocol, async ({ stream, connection }) => {
      const remotePeerId = connection.remotePeer.toString();
      try {
        await handler(stream, remotePeerId);
      } catch (error) {
        console.error(`[Network] Protocol handler error for ${protocol}:`, error);
      }
    });

    console.log(`[Network] Registered protocol handler: ${protocol}`);
  }

  // ============ Peer Management ============

  /**
   * Get own peer ID
   */
  getPeerId(): PeerId | null {
    return this.peerId;
  }

  /**
   * Get listen addresses
   */
  getListenAddresses(): string[] {
    if (!this.libp2p) return [];
    return this.libp2p.getMultiaddrs().map(ma => ma.toString());
  }

  /**
   * Get connected peers
   */
  getConnectedPeers(): PeerInfo[] {
    return Array.from(this.peers.values());
  }

  /**
   * Get peer count
   */
  getPeerCount(): number {
    return this.peers.size;
  }

  /**
   * Connect to a peer
   */
  async connect(address: Multiaddr): Promise<PeerInfo | null> {
    if (!this.libp2p) {
      throw new Error('Node not initialized');
    }

    console.log(`[Network] Connecting to: ${address}`);

    try {
      const ma = multiaddr(address);
      const connection = await this.libp2p.dial(ma);
      const remotePeerId = connection.remotePeer.toString();

      // Check if we already have peer info
      let peerInfo = this.peers.get(remotePeerId);
      if (!peerInfo) {
        peerInfo = {
          peerId: remotePeerId,
          addresses: [address],
          protocols: [],
          latency: 0,
          score: 0,
          connectedAt: Date.now(),
        };
        this.peers.set(remotePeerId, peerInfo);
        this.emit('peer:connect', { type: 'connect', peer: peerInfo, timestamp: Date.now() });
      }

      return peerInfo;
    } catch (error) {
      console.warn(`[Network] Failed to connect to ${address}:`, error);
      return null;
    }
  }

  /**
   * Disconnect from a peer
   */
  async disconnect(peerId: PeerId): Promise<void> {
    if (!this.libp2p) return;

    const peer = this.peers.get(peerId);
    if (!peer) return;

    try {
      // Get all connections to this peer and close them
      const connections = this.libp2p.getConnections(peerId as unknown as Libp2pPeerId);
      for (const conn of connections) {
        await conn.close();
      }
    } catch (error) {
      console.warn(`[Network] Error disconnecting from ${peerId}:`, error);
    }

    this.peers.delete(peerId);
    if (peer) {
      this.emit('peer:disconnect', { type: 'disconnect', peer, timestamp: Date.now() });
    }
  }

  // ============ Messaging ============

  /**
   * Broadcast a message to all peers on a topic
   */
  async broadcast(topic: string, message: NetworkMessage): Promise<void> {
    if (!this.started || !this.libp2p) {
      throw new Error('Node not started');
    }

    const pubsub = this.libp2p.services.pubsub as GossipSub;

    // Encode message as JSON
    const data = new TextEncoder().encode(JSON.stringify(message));

    // Publish to topic
    await pubsub.publish(topic, data);

    this.stats.messagesSent++;
    this.stats.bytesSent += BigInt(data.length);
    this.emit('message:sent', { topic, message, timestamp: Date.now() });
  }

  /**
   * Send a direct message to a specific peer via protocol stream
   */
  async send(peerId: PeerId, protocol: string, message: NetworkMessage): Promise<void> {
    if (!this.started || !this.libp2p) {
      throw new Error('Node not started');
    }

    const peer = this.peers.get(peerId);
    if (!peer) {
      throw new Error(`Not connected to peer: ${peerId}`);
    }

    // Find a connection to this peer
    const connections = this.libp2p.getConnections(peerId as unknown as Libp2pPeerId);
    if (connections.length === 0) {
      throw new Error(`No active connection to peer: ${peerId}`);
    }

    // Open a new stream on the protocol
    const stream = await connections[0].newStream(protocol);

    // Encode and send message
    const data = new TextEncoder().encode(JSON.stringify(message));

    await pipe(
      [data],
      lp.encode,
      stream.sink
    );

    this.stats.messagesSent++;
    this.stats.bytesSent += BigInt(data.length);
    this.emit('message:sent', { to: peerId, protocol, message, timestamp: Date.now() });
  }

  /**
   * Request-response pattern
   */
  async request<T>(
    peerId: PeerId,
    protocol: string,
    request: NetworkMessage,
    timeout: number = 10000
  ): Promise<T> {
    if (!this.started || !this.libp2p) {
      throw new Error('Node not started');
    }

    return new Promise(async (resolve, reject) => {
      const timer = setTimeout(() => {
        reject(new Error('Request timeout'));
      }, timeout);

      try {
        // Find connection to peer
        const connections = this.libp2p!.getConnections(peerId as unknown as Libp2pPeerId);
        if (connections.length === 0) {
          clearTimeout(timer);
          reject(new Error(`No active connection to peer: ${peerId}`));
          return;
        }

        // Open stream
        const stream = await connections[0].newStream(protocol);

        // Send request
        const requestData = new TextEncoder().encode(JSON.stringify(request));

        // Pipe request and read response
        let responseData: Uint8Array | undefined;

        await pipe(
          [requestData],
          lp.encode,
          stream,
          lp.decode,
          async function (source) {
            for await (const data of source) {
              responseData = data.subarray();
              break; // Only read first response
            }
          }
        );

        clearTimeout(timer);

        if (responseData) {
          const decoded = JSON.parse(new TextDecoder().decode(responseData));
          resolve(decoded as T);
        } else {
          reject(new Error('No response received'));
        }
      } catch (error) {
        clearTimeout(timer);
        reject(error);
      }
    });
  }

  // ============ Topics ============

  /**
   * Subscribe to a topic
   */
  async subscribe(topic: string): Promise<void> {
    if (!this.libp2p) {
      throw new Error('Node not initialized');
    }

    const pubsub = this.libp2p.services.pubsub as GossipSub;
    pubsub.subscribe(topic);

    console.log(`[Network] Subscribed to topic: ${topic}`);
  }

  /**
   * Unsubscribe from a topic
   */
  async unsubscribe(topic: string): Promise<void> {
    if (!this.libp2p) return;

    const pubsub = this.libp2p.services.pubsub as GossipSub;
    pubsub.unsubscribe(topic);

    console.log(`[Network] Unsubscribed from topic: ${topic}`);
  }

  // ============ Stats ============

  /**
   * Get network statistics
   */
  getStats(): NetworkStats {
    return {
      ...this.stats,
      peerId: this.peerId || '',
      connectedPeers: this.peers.size,
      uptime: this.stats.uptime > 0 ? Date.now() - this.stats.uptime : 0,
    };
  }

  // ============ Private Methods ============

  private async initLibp2p(): Promise<void> {
    // Build libp2p services configuration
    const services: Record<string, unknown> = {
      identify: identify(),
      pubsub: gossipsub({
        emitSelf: false,
        gossipIncoming: true,
        fallbackToFloodsub: false,
        allowPublishToZeroTopicPeers: true,
        scoreThresholds: this.config.pubsub.scoreThresholds,
      }),
    };

    // Add DHT if enabled
    if (this.config.dht.enabled) {
      services.dht = kadDHT({
        clientMode: this.config.dht.clientMode,
      });
    }

    // Build peer discovery configuration
    const peerDiscovery: unknown[] = [];

    if (this.config.enableMdns) {
      peerDiscovery.push(mdns());
    }

    if (this.config.bootstrapPeers.length > 0) {
      peerDiscovery.push(bootstrap({
        list: this.config.bootstrapPeers,
      }));
    }

    // Create libp2p node
    this.libp2p = await createLibp2p({
      addresses: {
        listen: this.config.listenAddresses,
        announce: this.config.announceAddresses || [],
      },
      transports: [
        tcp(),
        webSockets(),
      ],
      connectionEncrypters: [noise()],
      streamMuxers: [yamux()],
      services,
      peerDiscovery: peerDiscovery.length > 0 ? peerDiscovery : undefined,
      connectionManager: {
        maxConnections: this.config.maxConnections,
        minConnections: this.config.minConnections,
      },
    });

    // Store peer ID
    this.peerId = this.libp2p.peerId.toString();
    this.stats.peerId = this.peerId;
  }

  private setupLibp2pEvents(): void {
    if (!this.libp2p) return;

    // Peer connection events
    this.libp2p.addEventListener('peer:connect', (event) => {
      const remotePeer = event.detail;
      const peerId = remotePeer.toString();

      if (!this.peers.has(peerId)) {
        const peerInfo: PeerInfo = {
          peerId,
          addresses: [],
          protocols: [],
          latency: 0,
          score: 0,
          connectedAt: Date.now(),
        };
        this.peers.set(peerId, peerInfo);
        this.emit('peer:connect', { type: 'connect', peer: peerInfo, timestamp: Date.now() });
        console.log(`[Network] Peer connected: ${peerId.slice(0, 16)}...`);
      }
    });

    this.libp2p.addEventListener('peer:disconnect', (event) => {
      const remotePeer = event.detail;
      const peerId = remotePeer.toString();
      const peer = this.peers.get(peerId);

      if (peer) {
        this.peers.delete(peerId);
        this.emit('peer:disconnect', { type: 'disconnect', peer, timestamp: Date.now() });
        console.log(`[Network] Peer disconnected: ${peerId.slice(0, 16)}...`);
      }
    });

    // Peer discovery events
    this.libp2p.addEventListener('peer:discovery', (event) => {
      const peerInfo = event.detail;
      this.stats.knownPeers++;
      this.emit('peer:discovered', {
        peerId: peerInfo.id.toString(),
        multiaddrs: peerInfo.multiaddrs.map(ma => ma.toString()),
        timestamp: Date.now()
      });
    });

    // PubSub message events
    const pubsub = this.libp2p.services.pubsub as GossipSub;
    pubsub.addEventListener('message', async (event) => {
      const topic = event.detail.topic;
      const data = event.detail.data;
      const from = event.detail.from.toString();

      this.stats.messagesReceived++;
      this.stats.bytesReceived += BigInt(data.length);

      try {
        const message: NetworkMessage = JSON.parse(new TextDecoder().decode(data));
        message.sender = from;

        // Route to topic manager if available
        if (this.topicManager) {
          const handler = this.topicManager.getHandler(topic);
          if (handler) {
            const result = await handler.validate(message);
            this.updatePeerScore(from, result);

            if (result === 'accept') {
              await handler.handle(message);
            }
          }
        }

        this.emit('message:received', {
          topic,
          message,
          from,
          timestamp: Date.now()
        });
      } catch (error) {
        console.warn(`[Network] Failed to process message on ${topic}:`, error);
        this.updatePeerScore(from, 'reject');
      }
    });
  }

  private updatePeerScore(peerId: string, validationResult: ValidationResult): void {
    const peer = this.peers.get(peerId);
    if (!peer) return;

    switch (validationResult) {
      case 'accept':
        peer.score += 1;
        break;
      case 'reject':
        peer.score -= 10;
        break;
      case 'ignore':
        // No score change
        break;
    }
  }

  private async subscribeTopics(): Promise<void> {
    // Subscribe to all protocol topics
    for (const topic of Object.values(PUBSUB_TOPICS)) {
      await this.subscribe(topic);
    }
  }

  private initStats(): NetworkStats {
    return {
      peerId: '',
      connectedPeers: 0,
      knownPeers: 0,
      bytesReceived: 0n,
      bytesSent: 0n,
      messagesReceived: 0,
      messagesSent: 0,
      uptime: 0,
    };
  }
}

// ============ Factory Functions ============

/**
 * Create a devnet node
 */
export function createDevnetNode(port: number = 9000): LegibleNode {
  return new LegibleNode({
    listenAddresses: [
      `/ip4/127.0.0.1/tcp/${port}`,
    ],
    enableMdns: true,
    bootstrapPeers: [],
  });
}

/**
 * Create a testnet node
 */
export function createTestnetNode(bootstrapPeers: string[]): LegibleNode {
  return new LegibleNode({
    bootstrapPeers,
    enableMdns: false,
  });
}
