/**
 * Bootstrap Peer Discovery
 *
 * Initial peer discovery through known bootstrap nodes.
 */

import { EventEmitter } from 'events';

// ============ Types ============

export interface BootstrapPeer {
  address: string;        // Multiaddress
  name?: string;          // Human-readable name
  region?: string;        // Geographic region
  lastSeen?: number;      // Last successful connection
  failureCount: number;   // Consecutive failures
}

export interface BootstrapConfig {
  peers: BootstrapPeer[];
  connectionTimeout: number;
  maxRetries: number;
  retryDelay: number;
}

// ============ Default Bootstrap Peers ============

/**
 * Mainnet bootstrap peers (to be populated at launch)
 */
export const MAINNET_BOOTSTRAP: BootstrapPeer[] = [
  // Example format:
  // {
  //   address: '/dns4/boot1.legible.money/tcp/9000/p2p/12D3Koo...',
  //   name: 'boot1-us-east',
  //   region: 'us-east',
  //   failureCount: 0,
  // },
];

/**
 * Testnet bootstrap peers
 */
export const TESTNET_BOOTSTRAP: BootstrapPeer[] = [
  // To be populated for testnet
];

// ============ Bootstrap Discovery ============

/**
 * Bootstrap peer discovery manager
 */
export class BootstrapDiscovery extends EventEmitter {
  private peers: Map<string, BootstrapPeer> = new Map();
  private config: BootstrapConfig;
  private connecting: Set<string> = new Set();

  constructor(config: Partial<BootstrapConfig> = {}) {
    super();
    this.config = {
      peers: [],
      connectionTimeout: 10000,
      maxRetries: 3,
      retryDelay: 5000,
      ...config,
    };

    // Initialize with configured peers
    for (const peer of this.config.peers) {
      this.peers.set(peer.address, peer);
    }
  }

  /**
   * Add a bootstrap peer
   */
  addPeer(peer: BootstrapPeer): void {
    this.peers.set(peer.address, peer);
    this.emit('peer:added', peer);
  }

  /**
   * Remove a bootstrap peer
   */
  removePeer(address: string): void {
    const peer = this.peers.get(address);
    if (peer) {
      this.peers.delete(address);
      this.emit('peer:removed', peer);
    }
  }

  /**
   * Get all bootstrap peers
   */
  getPeers(): BootstrapPeer[] {
    return Array.from(this.peers.values());
  }

  /**
   * Get healthy bootstrap peers (low failure count)
   */
  getHealthyPeers(): BootstrapPeer[] {
    return this.getPeers()
      .filter(p => p.failureCount < this.config.maxRetries)
      .sort((a, b) => a.failureCount - b.failureCount);
  }

  /**
   * Attempt to connect to bootstrap peers
   */
  async discover(connect: (address: string) => Promise<boolean>): Promise<number> {
    const healthyPeers = this.getHealthyPeers();
    let connected = 0;

    for (const peer of healthyPeers) {
      if (this.connecting.has(peer.address)) continue;

      this.connecting.add(peer.address);
      try {
        const success = await this.connectWithRetry(peer, connect);
        if (success) {
          connected++;
          peer.lastSeen = Date.now();
          peer.failureCount = 0;
          this.emit('peer:connected', peer);
        }
      } catch (error) {
        peer.failureCount++;
        this.emit('peer:failed', { peer, error });
      } finally {
        this.connecting.delete(peer.address);
      }
    }

    return connected;
  }

  /**
   * Connect with retry logic
   */
  private async connectWithRetry(
    peer: BootstrapPeer,
    connect: (address: string) => Promise<boolean>
  ): Promise<boolean> {
    for (let attempt = 0; attempt < this.config.maxRetries; attempt++) {
      try {
        const success = await Promise.race([
          connect(peer.address),
          this.timeout(this.config.connectionTimeout),
        ]);

        if (success) return true;
      } catch {
        if (attempt < this.config.maxRetries - 1) {
          await this.delay(this.config.retryDelay);
        }
      }
    }
    return false;
  }

  private timeout(ms: number): Promise<never> {
    return new Promise((_, reject) => {
      setTimeout(() => reject(new Error('Connection timeout')), ms);
    });
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

/**
 * Create bootstrap discovery for mainnet
 */
export function createMainnetBootstrap(): BootstrapDiscovery {
  return new BootstrapDiscovery({
    peers: MAINNET_BOOTSTRAP,
  });
}

/**
 * Create bootstrap discovery for testnet
 */
export function createTestnetBootstrap(): BootstrapDiscovery {
  return new BootstrapDiscovery({
    peers: TESTNET_BOOTSTRAP,
  });
}

/**
 * Create bootstrap discovery for devnet (local)
 */
export function createDevnetBootstrap(localPeers: string[] = []): BootstrapDiscovery {
  return new BootstrapDiscovery({
    peers: localPeers.map(address => ({
      address,
      name: 'local',
      failureCount: 0,
    })),
  });
}
