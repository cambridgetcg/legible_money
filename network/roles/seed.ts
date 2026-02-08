/**
 * Seed Node Role
 *
 * Persistent bootstrap infrastructure for network discovery.
 * Seed nodes are stateless — they don't process blocks or transactions.
 * Their sole purpose is maintaining a peer address book and serving
 * peer exchange requests so new nodes can find the network.
 *
 * Capabilities:
 * - Topics: none
 * - Storage: peer address book only
 * - Sync: none
 * - DHT: server (high fan-out)
 * - Protocols: peer exchange, bootstrap
 */

import { EventEmitter } from "events";
import {
  validateMultiaddress,
  type BootstrapPeer,
} from "../discovery/bootstrap.js";
import { ROLE_PROTOCOL_IDS } from "./types.js";

// ============ Constants ============

/** Maximum peers to track in the address book */
const MAX_ADDRESS_BOOK_SIZE = 10_000;

/** How often to check peer liveness (ms) */
const LIVENESS_CHECK_INTERVAL = 60_000; // 1 minute

/** Peer considered stale after this many ms without contact */
const PEER_STALE_THRESHOLD = 300_000; // 5 minutes

/** Maximum peers to return in a single peer exchange response */
const MAX_PEX_RESPONSE_SIZE = 50;

/** Maximum age of a peer before eviction (ms) */
const PEER_EVICTION_AGE = 3_600_000; // 1 hour

// ============ Types ============

export interface SeedPeerEntry {
  /** Peer's multiaddress */
  address: string;

  /** Extracted peer ID */
  peerId: string;

  /** When this peer was first seen */
  firstSeen: number;

  /** Last time we confirmed this peer was alive */
  lastSeen: number;

  /** Last time this peer was served in a PEX response */
  lastServed: number;

  /** Number of successful liveness checks */
  successCount: number;

  /** Number of failed liveness checks */
  failureCount: number;

  /** Geographic region hint (if known) */
  region?: string;

  /** Whether this peer is a well-known static seed */
  isStatic: boolean;
}

export interface PeerExchangeRequest {
  /** Requesting peer's ID */
  requesterId: string;

  /** Maximum peers requested */
  maxPeers?: number;

  /** Preferred region (if any) */
  preferRegion?: string;
}

export interface PeerExchangeResponse {
  /** List of known peers */
  peers: Array<{
    address: string;
    peerId: string;
    lastSeen: number;
    region?: string;
  }>;

  /** Total peers known by this seed */
  totalKnown: number;
}

export interface SeedNodeConfig {
  /** Static seed peers from genesis (always kept) */
  staticPeers: BootstrapPeer[];

  /** Maximum address book size */
  maxPeers: number;

  /** Liveness check interval (ms) */
  livenessInterval: number;

  /** Peer stale threshold (ms) */
  staleThreshold: number;

  /** Peer eviction age (ms) */
  evictionAge: number;
}

export interface SeedNodeStats {
  /** Total peers in address book */
  totalPeers: number;

  /** Peers seen in the last stale threshold */
  alivePeers: number;

  /** Static peers count */
  staticPeers: number;

  /** Total PEX requests served */
  pexRequestsServed: number;

  /** Uptime in ms */
  uptime: number;
}

// ============ Seed Node Service ============

/**
 * Seed node service managing the peer address book and peer exchange protocol.
 *
 * Seed nodes are the entry point for new nodes joining the network.
 * They maintain a curated list of known peers and serve them
 * via the peer exchange (PEX) protocol.
 */
export class SeedNodeService extends EventEmitter {
  private addressBook: Map<string, SeedPeerEntry> = new Map();
  private config: SeedNodeConfig;
  private livenessTimer: ReturnType<typeof setInterval> | null = null;
  private startedAt: number = 0;
  private pexRequestsServed: number = 0;

  constructor(config?: Partial<SeedNodeConfig>) {
    super();
    this.config = {
      staticPeers: [],
      maxPeers: MAX_ADDRESS_BOOK_SIZE,
      livenessInterval: LIVENESS_CHECK_INTERVAL,
      staleThreshold: PEER_STALE_THRESHOLD,
      evictionAge: PEER_EVICTION_AGE,
      ...config,
    };
  }

  /**
   * Start the seed node service.
   * Loads static peers and begins liveness monitoring.
   */
  start(): void {
    this.startedAt = Date.now();

    // Load static peers from genesis
    for (const peer of this.config.staticPeers) {
      const validation = validateMultiaddress(peer.address);
      if (validation.valid && validation.peerId) {
        this.addressBook.set(validation.peerId, {
          address: peer.address,
          peerId: validation.peerId,
          firstSeen: Date.now(),
          lastSeen: Date.now(),
          lastServed: 0,
          successCount: 0,
          failureCount: 0,
          region: peer.region,
          isStatic: true,
        });
      }
    }

    // Start liveness monitoring
    this.livenessTimer = setInterval(
      () => this.evictStalePeers(),
      this.config.livenessInterval,
    );

    this.emit("started", { peers: this.addressBook.size });
  }

  /**
   * Stop the seed node service.
   */
  stop(): void {
    if (this.livenessTimer) {
      clearInterval(this.livenessTimer);
      this.livenessTimer = null;
    }
    this.emit("stopped");
  }

  /**
   * Register a peer in the address book.
   * Called when a new peer connects or is reported via PEX.
   */
  registerPeer(address: string, region?: string): boolean {
    const validation = validateMultiaddress(address);
    if (!validation.valid || !validation.peerId) {
      return false;
    }

    const peerId = validation.peerId;
    const existing = this.addressBook.get(peerId);

    if (existing) {
      // Update last seen
      existing.lastSeen = Date.now();
      existing.successCount++;
      existing.failureCount = 0;
      if (region) existing.region = region;
      return true;
    }

    // Evict if at capacity
    if (this.addressBook.size >= this.config.maxPeers) {
      this.evictOldestPeer();
    }

    this.addressBook.set(peerId, {
      address,
      peerId,
      firstSeen: Date.now(),
      lastSeen: Date.now(),
      lastServed: 0,
      successCount: 1,
      failureCount: 0,
      region,
      isStatic: false,
    });

    this.emit("peer:registered", { peerId, address });
    return true;
  }

  /**
   * Mark a peer as failed (e.g., liveness check failed).
   */
  markPeerFailed(peerId: string): void {
    const entry = this.addressBook.get(peerId);
    if (entry) {
      entry.failureCount++;
    }
  }

  /**
   * Handle a peer exchange request.
   * Returns a curated list of known healthy peers.
   */
  handlePeerExchange(request: PeerExchangeRequest): PeerExchangeResponse {
    const maxPeers = Math.min(
      request.maxPeers ?? MAX_PEX_RESPONSE_SIZE,
      MAX_PEX_RESPONSE_SIZE,
    );

    // Get alive peers, excluding the requester
    const now = Date.now();
    let candidates = Array.from(this.addressBook.values()).filter(
      (p) =>
        p.peerId !== request.requesterId &&
        p.failureCount < 3 &&
        now - p.lastSeen < this.config.staleThreshold,
    );

    // Prefer requested region if specified
    if (request.preferRegion) {
      const regionPeers = candidates.filter(
        (p) => p.region === request.preferRegion,
      );
      const otherPeers = candidates.filter(
        (p) => p.region !== request.preferRegion,
      );
      candidates = [...regionPeers, ...otherPeers];
    }

    // Randomize to distribute load and prevent hotspots
    candidates = shuffleArray(candidates);

    const selected = candidates.slice(0, maxPeers);

    // Update lastServed
    for (const peer of selected) {
      peer.lastServed = now;
    }

    this.pexRequestsServed++;
    this.emit("pex:served", {
      requesterId: request.requesterId,
      peersServed: selected.length,
    });

    return {
      peers: selected.map((p) => ({
        address: p.address,
        peerId: p.peerId,
        lastSeen: p.lastSeen,
        region: p.region,
      })),
      totalKnown: this.addressBook.size,
    };
  }

  /**
   * Get seed node statistics.
   */
  getStats(): SeedNodeStats {
    const now = Date.now();
    const alive = Array.from(this.addressBook.values()).filter(
      (p) => now - p.lastSeen < this.config.staleThreshold,
    ).length;

    const staticCount = Array.from(this.addressBook.values()).filter(
      (p) => p.isStatic,
    ).length;

    return {
      totalPeers: this.addressBook.size,
      alivePeers: alive,
      staticPeers: staticCount,
      pexRequestsServed: this.pexRequestsServed,
      uptime: this.startedAt > 0 ? now - this.startedAt : 0,
    };
  }

  /**
   * Get the protocol ID for peer exchange.
   */
  static get protocolId(): string {
    return ROLE_PROTOCOL_IDS.PEER_EXCHANGE;
  }

  // ============ Internal ============

  private evictStalePeers(): void {
    const now = Date.now();
    const toEvict: string[] = [];

    for (const [peerId, entry] of this.addressBook) {
      // Never evict static peers
      if (entry.isStatic) continue;

      // Evict peers not seen within eviction age
      if (now - entry.lastSeen > this.config.evictionAge) {
        toEvict.push(peerId);
      }

      // Evict peers with too many failures
      if (entry.failureCount >= 5) {
        toEvict.push(peerId);
      }
    }

    for (const peerId of toEvict) {
      this.addressBook.delete(peerId);
      this.emit("peer:evicted", { peerId });
    }
  }

  private evictOldestPeer(): void {
    let oldest: SeedPeerEntry | null = null;
    let oldestId: string | null = null;

    for (const [peerId, entry] of this.addressBook) {
      if (entry.isStatic) continue;
      if (!oldest || entry.lastSeen < oldest.lastSeen) {
        oldest = entry;
        oldestId = peerId;
      }
    }

    if (oldestId) {
      this.addressBook.delete(oldestId);
    }
  }
}

// ============ Utilities ============

/**
 * Fisher-Yates shuffle for fair peer distribution.
 */
function shuffleArray<T>(array: T[]): T[] {
  const result = [...array];
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}
