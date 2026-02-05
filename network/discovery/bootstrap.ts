/**
 * Bootstrap Peer Discovery
 *
 * Initial peer discovery through known bootstrap nodes.
 *
 * SECURITY (VULN-018):
 * - Validates multiaddress format before connection attempts
 * - Extracts and validates peer IDs from multiaddresses
 * - Blocks connections to invalid or malformed addresses
 */

import { EventEmitter } from "events";

// ============ Security Constants ============

/**
 * VULN-018 FIX: Valid multiaddress protocols
 */
const VALID_PROTOCOLS = [
  "ip4",
  "ip6",
  "dns4",
  "dns6",
  "tcp",
  "udp",
  "p2p",
  "ws",
  "wss",
];

/**
 * VULN-018 FIX: Peer ID pattern (Base58 encoded, typically 52 chars)
 */
const PEER_ID_PATTERN = /^12D3Koo[a-zA-Z0-9]{44,50}$/;

/**
 * VULN-018 FIX: Maximum multiaddress length to prevent DoS
 */
const MAX_MULTIADDR_LENGTH = 500;

// ============ Types ============

export interface BootstrapPeer {
  address: string; // Multiaddress
  name?: string; // Human-readable name
  region?: string; // Geographic region
  lastSeen?: number; // Last successful connection
  failureCount: number; // Consecutive failures
  peerId?: string; // VULN-018 FIX: Extracted peer ID
}

export interface BootstrapConfig {
  peers: BootstrapPeer[];
  connectionTimeout: number;
  maxRetries: number;
  retryDelay: number;
  validatePeers: boolean; // VULN-018 FIX: Enable peer validation
}

// ============ Peer Validation (VULN-018 FIX) ============

export interface PeerValidationResult {
  valid: boolean;
  peerId?: string;
  errors: string[];
}

/**
 * VULN-018 FIX: Validate a multiaddress format
 * Returns extracted peer ID if valid
 */
export function validateMultiaddress(address: string): PeerValidationResult {
  const errors: string[] = [];

  // Check length
  if (address.length > MAX_MULTIADDR_LENGTH) {
    errors.push(
      `Address too long: ${address.length} > ${MAX_MULTIADDR_LENGTH}`,
    );
    return { valid: false, errors };
  }

  // Must start with /
  if (!address.startsWith("/")) {
    errors.push("Multiaddress must start with /");
    return { valid: false, errors };
  }

  // Parse components
  const parts = address.slice(1).split("/");
  if (parts.length < 2) {
    errors.push("Multiaddress must have at least 2 components");
    return { valid: false, errors };
  }

  // Validate protocol/value pairs
  let peerId: string | undefined;
  for (let i = 0; i < parts.length - 1; i += 2) {
    const protocol = parts[i];
    const value = parts[i + 1];

    if (!VALID_PROTOCOLS.includes(protocol)) {
      errors.push(`Unknown protocol: ${protocol}`);
    }

    // Validate specific protocols
    switch (protocol) {
      case "ip4":
        if (!isValidIPv4(value)) {
          errors.push(`Invalid IPv4 address: ${value}`);
        }
        break;
      case "ip6":
        if (!isValidIPv6(value)) {
          errors.push(`Invalid IPv6 address: ${value}`);
        }
        break;
      case "tcp":
      case "udp": {
        const port = parseInt(value, 10);
        if (isNaN(port) || port < 1 || port > 65535) {
          errors.push(`Invalid port: ${value}`);
        }
        break;
      }
      case "p2p":
        peerId = value;
        if (!PEER_ID_PATTERN.test(value)) {
          errors.push(`Invalid peer ID format: ${value}`);
        }
        break;
      case "dns4":
      case "dns6":
        if (!isValidHostname(value)) {
          errors.push(`Invalid hostname: ${value}`);
        }
        break;
    }
  }

  return {
    valid: errors.length === 0,
    peerId,
    errors,
  };
}

function isValidIPv4(ip: string): boolean {
  const parts = ip.split(".");
  if (parts.length !== 4) return false;
  return parts.every((p) => {
    const n = parseInt(p, 10);
    return !isNaN(n) && n >= 0 && n <= 255 && p === n.toString();
  });
}

function isValidIPv6(ip: string): boolean {
  // Simplified IPv6 validation
  const parts = ip.split(":");
  if (parts.length < 3 || parts.length > 8) return false;
  return parts.every((p) => p === "" || /^[0-9a-fA-F]{1,4}$/.test(p));
}

function isValidHostname(hostname: string): boolean {
  // RFC 1123 hostname validation
  if (hostname.length > 253) return false;
  const labels = hostname.split(".");
  return labels.every(
    (label) =>
      label.length > 0 &&
      label.length <= 63 &&
      /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$/.test(label),
  );
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
 *
 * VULN-018 FIX: Validates all peer addresses before adding or connecting
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
      validatePeers: true, // VULN-018 FIX: Enable validation by default
      ...config,
    };

    // Initialize with configured peers (with validation)
    for (const peer of this.config.peers) {
      this.addPeerInternal(peer, false); // Don't emit during construction
    }
  }

  /**
   * VULN-018 FIX: Internal method to add peer with validation
   */
  private addPeerInternal(peer: BootstrapPeer, emit: boolean = true): boolean {
    // Validate address if validation is enabled
    if (this.config.validatePeers) {
      const validation = validateMultiaddress(peer.address);
      if (!validation.valid) {
        console.warn(
          `[Bootstrap] Rejecting invalid peer address: ${peer.address}`,
        );
        console.warn(
          `[Bootstrap] Validation errors: ${validation.errors.join(", ")}`,
        );
        this.emit("peer:validation_failed", {
          peer,
          errors: validation.errors,
        });
        return false;
      }
      // Store extracted peer ID
      peer.peerId = validation.peerId;
    }

    this.peers.set(peer.address, peer);
    if (emit) {
      this.emit("peer:added", peer);
    }
    return true;
  }

  /**
   * Add a bootstrap peer
   * VULN-018 FIX: Validates peer address before adding
   */
  addPeer(peer: BootstrapPeer): boolean {
    return this.addPeerInternal(peer, true);
  }

  /**
   * Remove a bootstrap peer
   */
  removePeer(address: string): void {
    const peer = this.peers.get(address);
    if (peer) {
      this.peers.delete(address);
      this.emit("peer:removed", peer);
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
      .filter((p) => p.failureCount < this.config.maxRetries)
      .sort((a, b) => a.failureCount - b.failureCount);
  }

  /**
   * Attempt to connect to bootstrap peers
   */
  async discover(
    connect: (address: string) => Promise<boolean>,
  ): Promise<number> {
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
          this.emit("peer:connected", peer);
        }
      } catch (error) {
        peer.failureCount++;
        this.emit("peer:failed", { peer, error });
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
    connect: (address: string) => Promise<boolean>,
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
      setTimeout(() => reject(new Error("Connection timeout")), ms);
    });
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
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
export function createDevnetBootstrap(
  localPeers: string[] = [],
): BootstrapDiscovery {
  return new BootstrapDiscovery({
    peers: localPeers.map((address) => ({
      address,
      name: "local",
      failureCount: 0,
    })),
  });
}
