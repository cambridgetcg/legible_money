/**
 * Evidence Archiving System
 *
 * Automatic archiving of evidence to multiple providers:
 * - IPFS: Content-addressed distributed storage
 * - Arweave: Permanent blockchain storage
 * - Wayback Machine: Web archive snapshots
 *
 * CONFIDENCE IMPACT:
 *   - Verified archives: +3% per archive (max +10%)
 *   - Multiple archives: Redundancy bonus
 *   - No archives: Evidence marked pending_archive
 */

import type { DID } from '../../agents/identity/did.js';
import type {
  Evidence,
  ArchiveLocation,
  ArchiveProvider,
  ArchiveRequest,
  ArchiveRequestStatus,
  ArchivePriority,
  ArchiveConfig,
} from './types.js';
import { DEFAULT_ARCHIVE_CONFIG } from './types.js';

// ============ Constants ============

/**
 * Archive confidence bonuses
 */
export const ARCHIVE_CONFIDENCE = {
  /** Bonus per verified archive */
  PER_ARCHIVE_BONUS: 0.03,

  /** Maximum total archive bonus */
  MAX_ARCHIVE_BONUS: 0.10,

  /** Penalty for no archives */
  NO_ARCHIVE_PENALTY: 0.05,

  /** Minimum archives for full bonus */
  MIN_ARCHIVES_FOR_BONUS: 2,
} as const;

/**
 * Archive retry configuration
 */
export const ARCHIVE_RETRY = {
  /** Initial retry delay (blocks) */
  INITIAL_DELAY: 100,

  /** Maximum retry delay (blocks) */
  MAX_DELAY: 10000,

  /** Backoff multiplier */
  BACKOFF_MULTIPLIER: 2,

  /** Maximum retry attempts */
  MAX_RETRIES: 5,
} as const;

// ============ Archive Request Management ============

/**
 * Create an archive request
 */
export function createArchiveRequest(
  evidenceId: string,
  originalUrl: string,
  requester: DID,
  currentBlock: bigint,
  priority: ArchivePriority = 'normal',
  providers: ArchiveProvider[] = ['ipfs', 'arweave', 'wayback']
): ArchiveRequest {
  return {
    id: `archive_${evidenceId}_${currentBlock}`,
    evidenceId,
    originalUrl,
    requestedAt: Date.now(),
    requestedAtBlock: currentBlock,
    requester,
    status: 'pending',
    priority,
    retryCount: 0,
    maxRetries: ARCHIVE_RETRY.MAX_RETRIES,
    targetProviders: providers,
  };
}

/**
 * Calculate retry delay with exponential backoff
 */
export function calculateRetryDelay(retryCount: number): number {
  const delay = ARCHIVE_RETRY.INITIAL_DELAY *
    Math.pow(ARCHIVE_RETRY.BACKOFF_MULTIPLIER, retryCount);
  return Math.min(delay, ARCHIVE_RETRY.MAX_DELAY);
}

/**
 * Check if archive request should be retried
 */
export function shouldRetryArchive(request: ArchiveRequest): boolean {
  return request.status === 'failed' &&
         request.retryCount < request.maxRetries;
}

/**
 * Update archive request status
 */
export function updateArchiveRequestStatus(
  request: ArchiveRequest,
  status: ArchiveRequestStatus,
  error?: string
): ArchiveRequest {
  return {
    ...request,
    status,
    lastError: error,
    retryCount: status === 'failed' ? request.retryCount + 1 : request.retryCount,
  };
}

// ============ Archive Location Management ============

/**
 * Create an archive location record
 */
export function createArchiveLocation(
  provider: ArchiveProvider,
  identifier: string,
  contentHash: string,
  currentBlock: bigint,
  archiver?: DID,
  isPrimary: boolean = false
): ArchiveLocation {
  return {
    provider,
    identifier,
    archivedAt: Date.now(),
    archivedAtBlock: currentBlock,
    contentHash,
    isPrimary,
    verified: false,
    archiver,
  };
}

/**
 * Mark archive as verified
 */
export function verifyArchive(
  archive: ArchiveLocation
): ArchiveLocation {
  return {
    ...archive,
    verified: true,
    verifiedAt: Date.now(),
  };
}

/**
 * Add archive to evidence
 */
export function addArchiveToEvidence(
  evidence: Evidence,
  archive: ArchiveLocation
): Evidence {
  // Check if we already have this archive
  const existing = evidence.archives.find(
    a => a.provider === archive.provider && a.identifier === archive.identifier
  );

  if (existing) {
    return evidence;
  }

  // If this is the first archive, make it primary
  const isPrimary = evidence.archives.length === 0;

  return {
    ...evidence,
    archives: [
      ...evidence.archives,
      { ...archive, isPrimary: isPrimary || archive.isPrimary },
    ],
    status: evidence.status === 'pending_archive' ? 'archived' : evidence.status,
  };
}

// ============ Archive Confidence Calculation ============

/**
 * Calculate confidence bonus from archives
 */
export function calculateArchiveBonus(archives: ArchiveLocation[]): number {
  const verifiedArchives = archives.filter(a => a.verified);

  if (verifiedArchives.length === 0) {
    return -ARCHIVE_CONFIDENCE.NO_ARCHIVE_PENALTY;
  }

  const bonus = Math.min(
    verifiedArchives.length * ARCHIVE_CONFIDENCE.PER_ARCHIVE_BONUS,
    ARCHIVE_CONFIDENCE.MAX_ARCHIVE_BONUS
  );

  return bonus;
}

/**
 * Apply archive bonus to confidence
 */
export function applyArchiveBonus(
  baseConfidence: number,
  archives: ArchiveLocation[]
): number {
  const bonus = calculateArchiveBonus(archives);
  return Math.max(0, Math.min(1, baseConfidence + bonus));
}

/**
 * Check if evidence needs archiving
 */
export function needsArchiving(
  evidence: Evidence,
  config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG
): boolean {
  if (!config.autoArchiveEnabled) {
    return false;
  }

  // Check if we have minimum archives
  const verifiedArchives = evidence.archives.filter(a => a.verified);
  return verifiedArchives.length < config.minArchiveCopies;
}

/**
 * Get missing archive providers
 */
export function getMissingProviders(
  evidence: Evidence,
  config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG
): ArchiveProvider[] {
  const existingProviders = new Set(evidence.archives.map(a => a.provider));

  return config.providers
    .filter(p => p.enabled && !existingProviders.has(p.provider))
    .map(p => p.provider);
}

// ============ Archive Queue Management ============

/**
 * Priority queue for archive requests
 */
export class ArchiveQueue {
  private requests: Map<string, ArchiveRequest> = new Map();
  private byPriority: Map<ArchivePriority, string[]> = new Map([
    ['critical', []],
    ['high', []],
    ['normal', []],
    ['low', []],
  ]);
  private byEvidence: Map<string, string[]> = new Map();

  /**
   * Add request to queue
   */
  enqueue(request: ArchiveRequest): void {
    this.requests.set(request.id, request);

    // Add to priority queue
    const priorityQueue = this.byPriority.get(request.priority)!;
    priorityQueue.push(request.id);

    // Index by evidence
    const evidenceQueue = this.byEvidence.get(request.evidenceId) || [];
    evidenceQueue.push(request.id);
    this.byEvidence.set(request.evidenceId, evidenceQueue);
  }

  /**
   * Get next request to process (highest priority first)
   */
  dequeue(): ArchiveRequest | undefined {
    for (const priority of ['critical', 'high', 'normal', 'low'] as ArchivePriority[]) {
      const queue = this.byPriority.get(priority)!;

      while (queue.length > 0) {
        const requestId = queue.shift()!;
        const request = this.requests.get(requestId);

        if (request && request.status === 'pending') {
          return request;
        }
      }
    }

    return undefined;
  }

  /**
   * Update request status
   */
  updateStatus(requestId: string, status: ArchiveRequestStatus, error?: string): void {
    const request = this.requests.get(requestId);
    if (!request) return;

    const updated = updateArchiveRequestStatus(request, status, error);
    this.requests.set(requestId, updated);

    // Re-queue if should retry
    if (shouldRetryArchive(updated)) {
      const priorityQueue = this.byPriority.get(updated.priority)!;
      priorityQueue.push(requestId);
      // Reset to pending for retry
      this.requests.set(requestId, { ...updated, status: 'pending' });
    }
  }

  /**
   * Get requests for evidence
   */
  getRequestsForEvidence(evidenceId: string): ArchiveRequest[] {
    const requestIds = this.byEvidence.get(evidenceId) || [];
    return requestIds.map(id => this.requests.get(id)!).filter(Boolean);
  }

  /**
   * Get pending request count
   */
  getPendingCount(): number {
    return Array.from(this.requests.values()).filter(r => r.status === 'pending').length;
  }

  /**
   * Get queue statistics
   */
  getStats(): {
    total: number;
    pending: number;
    inProgress: number;
    completed: number;
    failed: number;
    byPriority: Record<ArchivePriority, number>;
  } {
    const requests = Array.from(this.requests.values());

    return {
      total: requests.length,
      pending: requests.filter(r => r.status === 'pending').length,
      inProgress: requests.filter(r => r.status === 'in_progress').length,
      completed: requests.filter(r => r.status === 'completed').length,
      failed: requests.filter(r => r.status === 'failed' || r.status === 'permanent_failure').length,
      byPriority: {
        critical: requests.filter(r => r.priority === 'critical').length,
        high: requests.filter(r => r.priority === 'high').length,
        normal: requests.filter(r => r.priority === 'normal').length,
        low: requests.filter(r => r.priority === 'low').length,
      },
    };
  }
}

// ============ Archive Verification ============

/**
 * Archive verification result
 */
export interface ArchiveVerificationResult {
  evidenceId: string;
  archiveIndex: number;
  provider: ArchiveProvider;
  wasVerified: boolean;
  isAccessible: boolean;
  contentMatches: boolean;
  newlyVerified: boolean;
  error?: string;
}

/**
 * Verify archive is accessible and content matches
 * In production, this would actually fetch from the provider
 */
export function verifyArchiveAccessibility(
  archive: ArchiveLocation,
  expectedContentHash: string
): { isAccessible: boolean; contentMatches: boolean; error?: string } {
  // Simulate verification - in production, would fetch content
  // and compare hash

  // For now, assume accessible if we have an identifier
  const isAccessible = !!archive.identifier && archive.identifier.length > 0;

  // Content matches if we recorded the same hash
  const contentMatches = archive.contentHash === expectedContentHash;

  return {
    isAccessible,
    contentMatches,
    error: !isAccessible ? 'Archive not accessible' :
           !contentMatches ? 'Content hash mismatch' : undefined,
  };
}

/**
 * Verify all archives for evidence
 */
export function verifyEvidenceArchives(
  evidence: Evidence
): ArchiveVerificationResult[] {
  return evidence.archives.map((archive, index) => {
    const wasVerified = archive.verified;
    const { isAccessible, contentMatches, error } = verifyArchiveAccessibility(
      archive,
      evidence.contentHash
    );

    const shouldVerify = isAccessible && contentMatches;
    const newlyVerified = !wasVerified && shouldVerify;

    return {
      evidenceId: evidence.id,
      archiveIndex: index,
      provider: archive.provider,
      wasVerified,
      isAccessible,
      contentMatches,
      newlyVerified,
      error,
    };
  });
}

/**
 * Process verification results and update evidence
 */
export function applyVerificationResults(
  evidence: Evidence,
  results: ArchiveVerificationResult[]
): Evidence {
  const updatedArchives = evidence.archives.map((archive, index) => {
    const result = results.find(r => r.archiveIndex === index);
    if (!result) return archive;

    if (result.newlyVerified) {
      return verifyArchive(archive);
    }

    return archive;
  });

  return {
    ...evidence,
    archives: updatedArchives,
  };
}

// ============ Archive Provider Interfaces ============

/**
 * Abstract archive provider interface
 */
export interface ArchiveProviderInterface {
  readonly name: ArchiveProvider;

  /**
   * Archive content and return identifier
   */
  archive(url: string, content: string): Promise<{ identifier: string; contentHash: string }>;

  /**
   * Verify archive is accessible
   */
  verify(identifier: string): Promise<{ accessible: boolean; contentHash?: string }>;

  /**
   * Get archive URL for display
   */
  getDisplayUrl(identifier: string): string;
}

/**
 * IPFS archive provider (stub - would use actual IPFS client)
 */
export class IPFSArchiveProvider implements ArchiveProviderInterface {
  readonly name: ArchiveProvider = 'ipfs';

  async archive(_url: string, content: string): Promise<{ identifier: string; contentHash: string }> {
    // In production: upload to IPFS, get CID
    const hash = this.simpleHash(content);
    const cid = `Qm${hash.substring(0, 44)}`; // Fake CID format

    return {
      identifier: cid,
      contentHash: hash,
    };
  }

  async verify(identifier: string): Promise<{ accessible: boolean; contentHash?: string }> {
    // In production: check IPFS gateway
    return {
      accessible: identifier.startsWith('Qm'),
      contentHash: undefined,
    };
  }

  getDisplayUrl(identifier: string): string {
    return `https://ipfs.io/ipfs/${identifier}`;
  }

  private simpleHash(content: string): string {
    let hash = 0;
    for (let i = 0; i < content.length; i++) {
      const char = content.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}

/**
 * Arweave archive provider (stub)
 */
export class ArweaveArchiveProvider implements ArchiveProviderInterface {
  readonly name: ArchiveProvider = 'arweave';

  async archive(_url: string, content: string): Promise<{ identifier: string; contentHash: string }> {
    // In production: upload to Arweave, get transaction ID
    const hash = this.simpleHash(content);
    const txId = hash.substring(0, 43);

    return {
      identifier: txId,
      contentHash: hash,
    };
  }

  async verify(identifier: string): Promise<{ accessible: boolean; contentHash?: string }> {
    // In production: check Arweave gateway
    return {
      accessible: identifier.length === 43,
      contentHash: undefined,
    };
  }

  getDisplayUrl(identifier: string): string {
    return `https://arweave.net/${identifier}`;
  }

  private simpleHash(content: string): string {
    let hash = 0;
    for (let i = 0; i < content.length; i++) {
      const char = content.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    return Math.abs(hash).toString(16).padStart(64, '0');
  }
}

/**
 * Wayback Machine archive provider (stub)
 */
export class WaybackArchiveProvider implements ArchiveProviderInterface {
  readonly name: ArchiveProvider = 'wayback';

  async archive(url: string, _content: string): Promise<{ identifier: string; contentHash: string }> {
    // In production: submit to Wayback Machine Save Page Now API
    const timestamp = new Date().toISOString().replace(/[-:T.Z]/g, '').substring(0, 14);
    const archiveUrl = `https://web.archive.org/web/${timestamp}/${url}`;

    return {
      identifier: archiveUrl,
      contentHash: '', // Wayback doesn't provide content hash
    };
  }

  async verify(identifier: string): Promise<{ accessible: boolean; contentHash?: string }> {
    // In production: check if archived page is accessible
    return {
      accessible: identifier.includes('web.archive.org'),
      contentHash: undefined,
    };
  }

  getDisplayUrl(identifier: string): string {
    return identifier; // Already a full URL
  }
}
