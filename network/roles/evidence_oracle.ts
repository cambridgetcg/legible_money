/**
 * Evidence Oracle Node Role
 *
 * External evidence pipeline for fetching, archiving, and verifying
 * evidence from outside the LGM network. Evidence Oracles bridge
 * the gap between external truth sources and on-chain evidence records.
 *
 * Capabilities:
 * - Topics: EVIDENCE, EVIDENCE_REQUESTS, ATTESTATIONS
 * - Storage: evidence namespace + archives
 * - Sync: state sync (evidence subtree only)
 * - DHT: client
 * - Protocols: evidence fetch, archive
 */

import { EventEmitter } from "events";
import { ROLE_PROTOCOL_IDS } from "./types.js";
import type { DID } from "../../core/agents/identity/did.js";

// ============ Constants ============

/** Maximum concurrent fetches */
const MAX_CONCURRENT_FETCHES = 10;

/** Fetch timeout per source (ms) */
const DEFAULT_FETCH_TIMEOUT = 30_000;

/** Maximum content size to archive (bytes) */
const MAX_CONTENT_SIZE = 10 * 1024 * 1024; // 10 MB

/** Rate limit per external source (requests/minute) */
const SOURCE_RATE_LIMIT = 10;

/** How often to check the fetch queue (ms) */
const QUEUE_POLL_INTERVAL = 1_000;

// ============ Types ============

export type FetchStatus =
  | "pending"
  | "fetching"
  | "verifying"
  | "archived"
  | "failed";

export interface EvidenceFetchRequest {
  /** Unique request ID */
  id: string;

  /** URL or identifier to fetch */
  source: string;

  /** Source type */
  sourceType: "url" | "ipfs" | "arxiv" | "doi" | "api";

  /** Expected content hash (for verification) */
  expectedHash?: string;

  /** Requesting agent's DID */
  requesterDid?: DID;

  /** Evidence ID this fetch relates to */
  evidenceId?: string;

  /** Priority (higher = more urgent) */
  priority: number;

  /** When the request was created */
  createdAt: number;
}

export interface EvidenceFetchResult {
  /** Request ID */
  requestId: string;

  /** Whether the fetch succeeded */
  success: boolean;

  /** Computed content hash */
  contentHash?: string;

  /** Content size in bytes */
  contentSize?: number;

  /** Whether the hash matches expected */
  hashVerified?: boolean;

  /** Archive location (if archived) */
  archiveLocation?: string;

  /** Fetch duration (ms) */
  fetchDurationMs: number;

  /** Error message if failed */
  error?: string;
}

export interface OracleAttestation {
  /** Evidence ID being attested */
  evidenceId: string;

  /** Oracle node's DID */
  oracleDid: DID;

  /** Content hash verified by oracle */
  contentHash: string;

  /** Source URL/identifier */
  source: string;

  /** Whether the source was accessible */
  sourceAccessible: boolean;

  /** Whether the content hash matches */
  hashVerified: boolean;

  /** When the attestation was created */
  attestedAt: number;

  /** Block number when attested */
  attestedAtBlock: bigint;
}

export interface ArchiveRecord {
  /** Unique archive ID */
  id: string;

  /** Content hash */
  contentHash: string;

  /** Original source */
  source: string;

  /** Archive format */
  format: "raw" | "warc" | "ipfs_pin";

  /** Archive size in bytes */
  sizeBytes: number;

  /** When archived */
  archivedAt: number;

  /** Archive storage location */
  location: string;
}

export interface SourceReliability {
  /** Source domain or identifier */
  source: string;

  /** Total fetches attempted */
  totalFetches: number;

  /** Successful fetches */
  successfulFetches: number;

  /** Average response time (ms) */
  avgResponseTimeMs: number;

  /** Hash mismatches detected */
  hashMismatches: number;

  /** Last successful fetch */
  lastSuccess: number;

  /** Computed reliability score (0-1) */
  reliabilityScore: number;
}

export interface EvidenceOracleConfig {
  /** Maximum concurrent fetches */
  maxConcurrentFetches: number;

  /** Fetch timeout per source (ms) */
  fetchTimeout: number;

  /** Maximum content size (bytes) */
  maxContentSize: number;

  /** Rate limit per source (requests/minute) */
  sourceRateLimit: number;

  /** Queue poll interval (ms) */
  queuePollInterval: number;
}

export interface EvidenceOracleStats {
  /** Total fetch requests processed */
  totalFetches: number;

  /** Successful fetches */
  successfulFetches: number;

  /** Failed fetches */
  failedFetches: number;

  /** Hash verification successes */
  hashVerifications: number;

  /** Hash mismatches found */
  hashMismatches: number;

  /** Total content archived (bytes) */
  totalArchivedBytes: number;

  /** Active fetches in progress */
  activeFetches: number;

  /** Queue size */
  queueSize: number;

  /** Source reliability summaries */
  sourceCount: number;

  /** Uptime in ms */
  uptime: number;
}

// ============ Source Rate Limiter ============

class SourceRateLimiter {
  private windows: Map<string, number[]> = new Map();
  private limit: number;

  constructor(limit: number) {
    this.limit = limit;
  }

  allow(source: string): boolean {
    const domain = this.extractDomain(source);
    const now = Date.now();
    const window = this.windows.get(domain) ?? [];
    const recent = window.filter((t) => now - t < 60_000);
    recent.push(now);
    this.windows.set(domain, recent);
    return recent.length <= this.limit;
  }

  cleanup(): void {
    const now = Date.now();
    for (const [domain, window] of this.windows) {
      const recent = window.filter((t) => now - t < 60_000);
      if (recent.length === 0) {
        this.windows.delete(domain);
      } else {
        this.windows.set(domain, recent);
      }
    }
  }

  private extractDomain(source: string): string {
    try {
      const url = new URL(source);
      return url.hostname;
    } catch {
      return source;
    }
  }
}

// ============ Evidence Oracle Service ============

/**
 * Evidence Oracle service for external evidence fetching and verification.
 *
 * The oracle fetches evidence from external sources, verifies content
 * hashes, archives content, and produces attestations that can be
 * submitted on-chain.
 */
export class EvidenceOracleService extends EventEmitter {
  private config: EvidenceOracleConfig;
  private fetchQueue: EvidenceFetchRequest[] = [];
  private activeFetches: Set<string> = new Set();
  private sourceReliability: Map<string, SourceReliability> = new Map();
  private rateLimiter: SourceRateLimiter;
  private queueTimer: ReturnType<typeof setInterval> | null = null;
  private cleanupTimer: ReturnType<typeof setInterval> | null = null;
  private startedAt: number = 0;

  /**
   * Fetch handler — set by the node orchestrator to connect
   * the oracle to actual HTTP/IPFS fetching capability.
   */
  private fetchHandler:
    | ((
        source: string,
        sourceType: string,
        timeout: number,
      ) => Promise<{ data: Uint8Array; contentType?: string }>)
    | null = null;

  /**
   * Hash handler — computes content hash using the protocol's hash function.
   */
  private hashHandler: ((data: Uint8Array) => string) | null = null;

  // Stats
  private totalFetches: number = 0;
  private successfulFetches: number = 0;
  private failedFetches: number = 0;
  private hashVerifications: number = 0;
  private hashMismatches: number = 0;
  private totalArchivedBytes: number = 0;

  constructor(config?: Partial<EvidenceOracleConfig>) {
    super();
    this.config = {
      maxConcurrentFetches: MAX_CONCURRENT_FETCHES,
      fetchTimeout: DEFAULT_FETCH_TIMEOUT,
      maxContentSize: MAX_CONTENT_SIZE,
      sourceRateLimit: SOURCE_RATE_LIMIT,
      queuePollInterval: QUEUE_POLL_INTERVAL,
      ...config,
    };
    this.rateLimiter = new SourceRateLimiter(this.config.sourceRateLimit);
  }

  /**
   * Set the fetch handler for external source retrieval.
   */
  setFetchHandler(
    handler: (
      source: string,
      sourceType: string,
      timeout: number,
    ) => Promise<{ data: Uint8Array; contentType?: string }>,
  ): void {
    this.fetchHandler = handler;
  }

  /**
   * Set the hash handler for content verification.
   */
  setHashHandler(handler: (data: Uint8Array) => string): void {
    this.hashHandler = handler;
  }

  /**
   * Start the oracle service.
   */
  start(): void {
    this.startedAt = Date.now();

    // Start queue processor
    this.queueTimer = setInterval(
      () => this.processQueue(),
      this.config.queuePollInterval,
    );

    // Start rate limiter cleanup
    this.cleanupTimer = setInterval(() => this.rateLimiter.cleanup(), 30_000);

    this.emit("started");
  }

  /**
   * Stop the oracle service.
   */
  stop(): void {
    if (this.queueTimer) {
      clearInterval(this.queueTimer);
      this.queueTimer = null;
    }
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
    this.emit("stopped");
  }

  /**
   * Submit a fetch request to the queue.
   */
  submitFetchRequest(request: EvidenceFetchRequest): boolean {
    // Check rate limit for source
    if (!this.rateLimiter.allow(request.source)) {
      this.emit("rate_limited", { source: request.source });
      return false;
    }

    // Insert by priority (higher priority first)
    const insertIdx = this.fetchQueue.findIndex(
      (r) => r.priority < request.priority,
    );
    if (insertIdx === -1) {
      this.fetchQueue.push(request);
    } else {
      this.fetchQueue.splice(insertIdx, 0, request);
    }

    this.emit("request:queued", { id: request.id });
    return true;
  }

  /**
   * Process a single fetch request (called by queue processor).
   */
  async processFetchRequest(
    request: EvidenceFetchRequest,
  ): Promise<EvidenceFetchResult> {
    const startTime = Date.now();
    this.activeFetches.add(request.id);
    this.totalFetches++;

    try {
      if (!this.fetchHandler || !this.hashHandler) {
        throw new Error("Fetch or hash handler not configured");
      }

      // Fetch the content
      const { data } = await this.fetchHandler(
        request.source,
        request.sourceType,
        this.config.fetchTimeout,
      );

      // Size check
      if (data.length > this.config.maxContentSize) {
        throw new Error(
          `Content too large: ${data.length} > ${this.config.maxContentSize}`,
        );
      }

      // Compute content hash
      const contentHash = this.hashHandler(data);

      // Verify against expected hash
      let hashVerified: boolean | undefined;
      if (request.expectedHash) {
        hashVerified = contentHash === request.expectedHash;
        if (hashVerified) {
          this.hashVerifications++;
        } else {
          this.hashMismatches++;
        }
      }

      // Update source reliability
      this.updateSourceReliability(
        request.source,
        true,
        Date.now() - startTime,
        !hashVerified && request.expectedHash !== undefined,
      );

      this.successfulFetches++;

      const result: EvidenceFetchResult = {
        requestId: request.id,
        success: true,
        contentHash,
        contentSize: data.length,
        hashVerified,
        fetchDurationMs: Date.now() - startTime,
      };

      this.emit("fetch:complete", result);
      return result;
    } catch (error) {
      this.failedFetches++;
      this.updateSourceReliability(
        request.source,
        false,
        Date.now() - startTime,
        false,
      );

      const result: EvidenceFetchResult = {
        requestId: request.id,
        success: false,
        fetchDurationMs: Date.now() - startTime,
        error: error instanceof Error ? error.message : "Fetch failed",
      };

      this.emit("fetch:failed", result);
      return result;
    } finally {
      this.activeFetches.delete(request.id);
    }
  }

  /**
   * Get source reliability data.
   */
  getSourceReliability(source: string): SourceReliability | undefined {
    const domain = this.extractDomain(source);
    return this.sourceReliability.get(domain);
  }

  /**
   * Get oracle statistics.
   */
  getStats(): EvidenceOracleStats {
    return {
      totalFetches: this.totalFetches,
      successfulFetches: this.successfulFetches,
      failedFetches: this.failedFetches,
      hashVerifications: this.hashVerifications,
      hashMismatches: this.hashMismatches,
      totalArchivedBytes: this.totalArchivedBytes,
      activeFetches: this.activeFetches.size,
      queueSize: this.fetchQueue.length,
      sourceCount: this.sourceReliability.size,
      uptime: this.startedAt > 0 ? Date.now() - this.startedAt : 0,
    };
  }

  /**
   * Get the protocol ID for evidence fetch.
   */
  static get protocolId(): string {
    return ROLE_PROTOCOL_IDS.EVIDENCE_FETCH;
  }

  // ============ Internal ============

  private async processQueue(): Promise<void> {
    while (
      this.fetchQueue.length > 0 &&
      this.activeFetches.size < this.config.maxConcurrentFetches
    ) {
      const request = this.fetchQueue.shift();
      if (request) {
        // Fire and forget — result emitted via events
        this.processFetchRequest(request).catch(() => {
          // Error already handled in processFetchRequest
        });
      }
    }
  }

  private updateSourceReliability(
    source: string,
    success: boolean,
    responseTimeMs: number,
    hashMismatch: boolean,
  ): void {
    const domain = this.extractDomain(source);
    const existing = this.sourceReliability.get(domain) ?? {
      source: domain,
      totalFetches: 0,
      successfulFetches: 0,
      avgResponseTimeMs: 0,
      hashMismatches: 0,
      lastSuccess: 0,
      reliabilityScore: 0.5,
    };

    existing.totalFetches++;
    if (success) {
      existing.successfulFetches++;
      existing.lastSuccess = Date.now();
    }
    if (hashMismatch) {
      existing.hashMismatches++;
    }

    // Running average for response time
    existing.avgResponseTimeMs =
      (existing.avgResponseTimeMs * (existing.totalFetches - 1) +
        responseTimeMs) /
      existing.totalFetches;

    // Compute reliability score
    const successRate =
      existing.totalFetches > 0
        ? existing.successfulFetches / existing.totalFetches
        : 0;
    const mismatchPenalty =
      existing.totalFetches > 0
        ? existing.hashMismatches / existing.totalFetches
        : 0;
    existing.reliabilityScore = Math.max(
      0,
      Math.min(1, successRate - mismatchPenalty * 2),
    );

    this.sourceReliability.set(domain, existing);
  }

  private extractDomain(source: string): string {
    try {
      const url = new URL(source);
      return url.hostname;
    } catch {
      return source;
    }
  }
}
