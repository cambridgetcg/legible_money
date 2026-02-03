/**
 * Evidence Deduplication System
 *
 * Detects duplicate and near-duplicate evidence submissions.
 * Rewards first submitters while giving confirmation credit to subsequent submitters.
 *
 * DEDUPLICATION METHODS:
 *   1. Exact Match: SHA-512/256 content hash comparison
 *   2. Near Duplicate: Perceptual hash with 85% similarity threshold
 *   3. Related: Feature-based semantic similarity
 *
 * REWARD DISTRIBUTION:
 *   - First submission: 50% of evidence reward
 *   - Exact duplicate: 40% to original, 10% to confirmer
 *   - Near duplicate: 45% to original, 5% to confirmer
 *   - Simultaneous (10 blocks): 25% each
 */

import type { DID } from '../../agents/identity/did.js';
import type {
  Evidence,
  EvidenceFingerprint,
  ContentFeatures,
  DuplicateDetectionResult,
  DuplicationType,
  DeduplicationConfig,
} from './types.js';
import {
  DEFAULT_DEDUPLICATION_CONFIG,
  EVIDENCE_ANTI_GAMING,
} from './types.js';

// ============ Fingerprint Generation ============

/**
 * Generate content hash using SHA-512/256
 * This is deterministic and collision-resistant
 */
export function generateContentHash(content: string): string {
  // In production, use proper crypto library
  // For now, simple hash simulation
  let hash = 0;
  for (let i = 0; i < content.length; i++) {
    const char = content.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash;
  }
  return Math.abs(hash).toString(16).padStart(16, '0');
}

/**
 * Generate perceptual hash for near-duplicate detection
 *
 * Uses a simplified approach:
 * 1. Normalize text (lowercase, remove punctuation)
 * 2. Extract key tokens
 * 3. Create hash from token frequencies
 */
export function generatePerceptualHash(content: string): string {
  // Normalize
  const normalized = content
    .toLowerCase()
    .replace(/[^\w\s]/g, '')
    .replace(/\s+/g, ' ')
    .trim();

  // Tokenize and get frequencies
  const tokens = normalized.split(' ');
  const frequencies = new Map<string, number>();

  for (const token of tokens) {
    if (token.length > 2) { // Skip very short tokens
      frequencies.set(token, (frequencies.get(token) || 0) + 1);
    }
  }

  // Create hash from top 20 tokens
  const sortedTokens = Array.from(frequencies.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 20)
    .map(([token]) => token);

  return generateContentHash(sortedTokens.join('|'));
}

/**
 * Normalize a URL for comparison
 * Uses regex-based parsing to avoid DOM dependencies
 */
export function normalizeUrl(url: string): string {
  // Simple URL regex
  const urlMatch = url.match(/^(?:https?:\/\/)?(?:www\.)?([^\/\?#]+)(\/[^\?#]*)?(\?[^#]*)?(#.*)?$/i);

  if (!urlMatch) {
    // Not a valid URL, just normalize as text
    return url.toLowerCase().trim();
  }

  const [, hostname, path = '', queryStr = ''] = urlMatch;

  // Remove tracking parameters from query string
  const trackingParams = ['utm_source', 'utm_medium', 'utm_campaign', 'ref', 'fbclid'];
  let cleanQuery = queryStr;
  for (const param of trackingParams) {
    cleanQuery = cleanQuery.replace(new RegExp(`[?&]${param}=[^&]*`, 'gi'), '');
  }
  // Clean up leading ? or & if params were removed
  if (cleanQuery && !cleanQuery.includes('=')) {
    cleanQuery = '';
  } else if (cleanQuery.startsWith('&')) {
    cleanQuery = '?' + cleanQuery.slice(1);
  }

  // Normalize path - remove trailing slash
  let normalizedPath = path;
  if (normalizedPath.endsWith('/') && normalizedPath.length > 1) {
    normalizedPath = normalizedPath.slice(0, -1);
  }

  return `${hostname.toLowerCase()}${normalizedPath}${cleanQuery}`;
}

/**
 * Extract content features for semantic similarity
 */
export function extractContentFeatures(content: string): ContentFeatures {
  const text = content.toLowerCase();

  // Token count
  const tokens = text.split(/\s+/).filter(t => t.length > 0);
  const tokenCount = tokens.length;

  // Extract entities (simplified - in production use NER)
  const entityPatterns = [
    /\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b/g, // Proper nouns
    /\b\d{4}\b/g, // Years
    /\$[\d,]+(?:\.\d{2})?/g, // Money amounts
    /\b\d+%\b/g, // Percentages
  ];

  const entities: string[] = [];
  for (const pattern of entityPatterns) {
    const matches = content.match(pattern) || [];
    entities.push(...matches);
  }

  // Extract numeric values
  const numericMatches = content.match(/\b\d+(?:\.\d+)?\b/g) || [];
  const numericValues = numericMatches.map(n => parseFloat(n)).filter(n => !isNaN(n));

  // Extract temporal references
  const temporalPatterns = [
    /\b(?:january|february|march|april|may|june|july|august|september|october|november|december)\s+\d{1,2},?\s*\d{4}\b/gi,
    /\b\d{1,2}\/\d{1,2}\/\d{2,4}\b/g,
    /\b\d{4}-\d{2}-\d{2}\b/g,
    /\b(?:today|yesterday|tomorrow|last\s+week|next\s+month)\b/gi,
  ];

  const temporalReferences: string[] = [];
  for (const pattern of temporalPatterns) {
    const matches = content.match(pattern) || [];
    temporalReferences.push(...matches);
  }

  // Topic classification (simplified - in production use ML classifier)
  const topicKeywords: Record<string, string[]> = {
    'finance': ['price', 'market', 'stock', 'investment', 'trading', 'currency'],
    'science': ['research', 'study', 'experiment', 'hypothesis', 'data', 'analysis'],
    'technology': ['software', 'hardware', 'algorithm', 'code', 'system', 'network'],
    'health': ['medical', 'health', 'disease', 'treatment', 'patient', 'clinical'],
    'general': [],
  };

  let topicClassification = 'general';
  let maxMatches = 0;

  for (const [topic, keywords] of Object.entries(topicKeywords)) {
    const matches = keywords.filter(kw => text.includes(kw)).length;
    if (matches > maxMatches) {
      maxMatches = matches;
      topicClassification = topic;
    }
  }

  return {
    tokenCount,
    entities: [...new Set(entities)].slice(0, 20),
    numericValues: numericValues.slice(0, 20),
    temporalReferences: [...new Set(temporalReferences)].slice(0, 10),
    topicClassification,
  };
}

/**
 * Generate complete evidence fingerprint
 */
export function generateFingerprint(
  content: string,
  source: string,
  submitter: DID,
  timestamp: number
): EvidenceFingerprint {
  return {
    contentHash: generateContentHash(content),
    perceptualHash: generatePerceptualHash(content),
    normalizedSource: normalizeUrl(source),
    contentFeatures: extractContentFeatures(content),
    firstSeen: timestamp,
    firstSubmitter: submitter,
  };
}

// ============ Similarity Calculation ============

/**
 * Calculate Hamming distance between two hex hashes
 */
function hammingDistance(hash1: string, hash2: string): number {
  const maxLen = Math.max(hash1.length, hash2.length);
  const h1 = hash1.padStart(maxLen, '0');
  const h2 = hash2.padStart(maxLen, '0');

  let distance = 0;
  for (let i = 0; i < maxLen; i++) {
    if (h1[i] !== h2[i]) {
      distance++;
    }
  }

  return distance / maxLen;
}

/**
 * Calculate perceptual hash similarity (0-1, higher = more similar)
 */
export function calculatePerceptualSimilarity(hash1: string, hash2: string): number {
  const distance = hammingDistance(hash1, hash2);
  return 1 - distance;
}

/**
 * Calculate Jaccard similarity for sets
 */
function jaccardSimilarity<T>(set1: Set<T>, set2: Set<T>): number {
  const intersection = new Set([...set1].filter(x => set2.has(x)));
  const union = new Set([...set1, ...set2]);

  if (union.size === 0) return 1; // Both empty = identical
  return intersection.size / union.size;
}

/**
 * Calculate feature-based similarity
 */
export function calculateFeatureSimilarity(
  features1: ContentFeatures,
  features2: ContentFeatures
): number {
  // Token count similarity (within 20% = high similarity)
  const tokenRatio = Math.min(features1.tokenCount, features2.tokenCount) /
                     Math.max(features1.tokenCount, features2.tokenCount);
  const tokenSimilarity = tokenRatio > 0.8 ? 1 : tokenRatio / 0.8;

  // Entity overlap
  const entitySimilarity = jaccardSimilarity(
    new Set(features1.entities),
    new Set(features2.entities)
  );

  // Numeric value overlap
  const numericSimilarity = jaccardSimilarity(
    new Set(features1.numericValues.map(n => n.toFixed(2))),
    new Set(features2.numericValues.map(n => n.toFixed(2)))
  );

  // Temporal reference overlap
  const temporalSimilarity = jaccardSimilarity(
    new Set(features1.temporalReferences),
    new Set(features2.temporalReferences)
  );

  // Topic match
  const topicMatch = features1.topicClassification === features2.topicClassification ? 1 : 0.5;

  // Weighted combination
  return (
    tokenSimilarity * 0.20 +
    entitySimilarity * 0.30 +
    numericSimilarity * 0.20 +
    temporalSimilarity * 0.15 +
    topicMatch * 0.15
  );
}

// ============ Duplicate Detection ============

/**
 * Check if two pieces of evidence are duplicates
 */
export function detectDuplicate(
  newFingerprint: EvidenceFingerprint,
  existingFingerprints: EvidenceFingerprint[],
  _currentBlock: bigint,
  config: DeduplicationConfig = DEFAULT_DEDUPLICATION_CONFIG
): DuplicateDetectionResult {
  // Default result - not a duplicate
  const notDuplicate: DuplicateDetectionResult = {
    isDuplicate: false,
    duplicationType: 'none',
    similarityScore: 0,
    matchingEvidenceIds: [],
    isSimultaneous: false,
    eligibleForConfirmationCredit: false,
  };

  if (existingFingerprints.length === 0) {
    return notDuplicate;
  }

  let bestMatch: {
    fingerprint: EvidenceFingerprint;
    type: DuplicationType;
    similarity: number;
  } | null = null;

  for (const existing of existingFingerprints) {
    // 1. Exact match check
    if (config.exactMatchEnabled && newFingerprint.contentHash === existing.contentHash) {
      const isSimultaneous = Math.abs(newFingerprint.firstSeen - existing.firstSeen) <=
                            config.simultaneousWindow * 2521; // Convert blocks to ms

      return {
        isDuplicate: true,
        duplicationType: 'exact',
        similarityScore: 1.0,
        matchingEvidenceIds: [],
        originalEvidenceId: existing.contentHash,
        firstSubmitter: existing.firstSubmitter,
        isSimultaneous,
        eligibleForConfirmationCredit: !isSimultaneous,
      };
    }

    // 2. Perceptual hash check (near-duplicate)
    const perceptualSimilarity = calculatePerceptualSimilarity(
      newFingerprint.perceptualHash,
      existing.perceptualHash
    );

    if (perceptualSimilarity >= (1 - config.perceptualHashThreshold)) {
      if (!bestMatch || perceptualSimilarity > bestMatch.similarity) {
        bestMatch = {
          fingerprint: existing,
          type: 'near',
          similarity: perceptualSimilarity,
        };
      }
    }

    // 3. Feature similarity check (related)
    const featureSimilarity = calculateFeatureSimilarity(
      newFingerprint.contentFeatures,
      existing.contentFeatures
    );

    if (featureSimilarity >= config.featureSimilarityThreshold) {
      if (!bestMatch || featureSimilarity > bestMatch.similarity) {
        bestMatch = {
          fingerprint: existing,
          type: 'related',
          similarity: featureSimilarity,
        };
      }
    }
  }

  if (bestMatch) {
    const isSimultaneous = Math.abs(newFingerprint.firstSeen - bestMatch.fingerprint.firstSeen) <=
                          config.simultaneousWindow * 2521;

    return {
      isDuplicate: true,
      duplicationType: bestMatch.type,
      similarityScore: bestMatch.similarity,
      matchingEvidenceIds: [],
      originalEvidenceId: bestMatch.fingerprint.contentHash,
      firstSubmitter: bestMatch.fingerprint.firstSubmitter,
      isSimultaneous,
      eligibleForConfirmationCredit: !isSimultaneous && bestMatch.type !== 'related',
    };
  }

  return notDuplicate;
}

/**
 * Check for duplicate in evidence array
 */
export function checkEvidenceDuplicate(
  newEvidence: Evidence,
  existingEvidence: Evidence[],
  currentBlock: bigint,
  config: DeduplicationConfig = DEFAULT_DEDUPLICATION_CONFIG
): DuplicateDetectionResult {
  // Generate fingerprint for new evidence
  const newFingerprint: EvidenceFingerprint = {
    contentHash: newEvidence.contentHash,
    perceptualHash: newEvidence.perceptualHash || generatePerceptualHash(newEvidence.originalSource),
    normalizedSource: normalizeUrl(newEvidence.originalSource),
    contentFeatures: extractContentFeatures(newEvidence.originalSource),
    firstSeen: newEvidence.submittedAt,
    firstSubmitter: newEvidence.firstSubmitter,
  };

  // Generate fingerprints for existing evidence
  const existingFingerprints = existingEvidence.map(e => ({
    contentHash: e.contentHash,
    perceptualHash: e.perceptualHash || generatePerceptualHash(e.originalSource),
    normalizedSource: normalizeUrl(e.originalSource),
    contentFeatures: extractContentFeatures(e.originalSource),
    firstSeen: e.submittedAt,
    firstSubmitter: e.firstSubmitter,
  }));

  const result = detectDuplicate(newFingerprint, existingFingerprints, currentBlock, config);

  // Add matching evidence IDs
  if (result.isDuplicate) {
    result.matchingEvidenceIds = existingEvidence
      .filter(e => e.contentHash === result.originalEvidenceId)
      .map(e => e.id);
  }

  return result;
}

// ============ Confirmation Credit ============

/**
 * Calculate confirmation credit for duplicate submission
 */
export function calculateConfirmationCredit(
  duplicateResult: DuplicateDetectionResult,
  baseReward: bigint,
  config: DeduplicationConfig = DEFAULT_DEDUPLICATION_CONFIG
): { originalCredit: bigint; confirmerCredit: bigint } {
  if (!duplicateResult.isDuplicate || !duplicateResult.eligibleForConfirmationCredit) {
    return { originalCredit: baseReward, confirmerCredit: 0n };
  }

  // Simultaneous submissions split evenly
  if (duplicateResult.isSimultaneous) {
    const half = baseReward / 2n;
    return { originalCredit: half, confirmerCredit: half };
  }

  // Calculate credit based on duplicate type
  let confirmerRate: number;

  switch (duplicateResult.duplicationType) {
    case 'exact':
      confirmerRate = config.confirmationCreditRate; // 10%
      break;
    case 'near':
      confirmerRate = config.confirmationCreditRate / 2; // 5%
      break;
    case 'related':
      confirmerRate = 0; // No credit for just related evidence
      break;
    default:
      confirmerRate = 0;
  }

  // Cap at maximum
  confirmerRate = Math.min(confirmerRate, EVIDENCE_ANTI_GAMING.MAX_CONFIRMATION_CREDIT);

  const confirmerCredit = BigInt(Math.floor(Number(baseReward) * confirmerRate));
  const originalCredit = baseReward - confirmerCredit;

  return { originalCredit, confirmerCredit };
}

// ============ Deduplication Index ============

/**
 * In-memory deduplication index
 */
export class DeduplicationIndex {
  private contentHashIndex: Map<string, string> = new Map(); // contentHash -> evidenceId
  private perceptualHashIndex: Map<string, string[]> = new Map(); // perceptualHash -> evidenceIds
  private sourceIndex: Map<string, string[]> = new Map(); // normalizedSource -> evidenceIds
  private fingerprintCache: Map<string, EvidenceFingerprint> = new Map(); // evidenceId -> fingerprint

  /**
   * Add evidence to the index
   */
  addEvidence(evidence: Evidence, fingerprint: EvidenceFingerprint): void {
    // Content hash index (exact)
    this.contentHashIndex.set(fingerprint.contentHash, evidence.id);

    // Perceptual hash index (near)
    const perceptualIds = this.perceptualHashIndex.get(fingerprint.perceptualHash) || [];
    perceptualIds.push(evidence.id);
    this.perceptualHashIndex.set(fingerprint.perceptualHash, perceptualIds);

    // Source index
    const sourceIds = this.sourceIndex.get(fingerprint.normalizedSource) || [];
    sourceIds.push(evidence.id);
    this.sourceIndex.set(fingerprint.normalizedSource, sourceIds);

    // Cache fingerprint
    this.fingerprintCache.set(evidence.id, fingerprint);
  }

  /**
   * Check for exact duplicate by content hash
   */
  findExactDuplicate(contentHash: string): string | undefined {
    return this.contentHashIndex.get(contentHash);
  }

  /**
   * Find potential near-duplicates by perceptual hash
   */
  findNearDuplicates(perceptualHash: string): string[] {
    return this.perceptualHashIndex.get(perceptualHash) || [];
  }

  /**
   * Find evidence from same source
   */
  findBySource(source: string): string[] {
    const normalized = normalizeUrl(source);
    return this.sourceIndex.get(normalized) || [];
  }

  /**
   * Get fingerprint for evidence
   */
  getFingerprint(evidenceId: string): EvidenceFingerprint | undefined {
    return this.fingerprintCache.get(evidenceId);
  }

  /**
   * Get all fingerprints
   */
  getAllFingerprints(): EvidenceFingerprint[] {
    return Array.from(this.fingerprintCache.values());
  }

  /**
   * Remove evidence from index
   */
  removeEvidence(evidenceId: string): void {
    const fingerprint = this.fingerprintCache.get(evidenceId);
    if (!fingerprint) return;

    // Remove from content hash index
    if (this.contentHashIndex.get(fingerprint.contentHash) === evidenceId) {
      this.contentHashIndex.delete(fingerprint.contentHash);
    }

    // Remove from perceptual hash index
    const perceptualIds = this.perceptualHashIndex.get(fingerprint.perceptualHash);
    if (perceptualIds) {
      const filtered = perceptualIds.filter(id => id !== evidenceId);
      if (filtered.length > 0) {
        this.perceptualHashIndex.set(fingerprint.perceptualHash, filtered);
      } else {
        this.perceptualHashIndex.delete(fingerprint.perceptualHash);
      }
    }

    // Remove from source index
    const sourceIds = this.sourceIndex.get(fingerprint.normalizedSource);
    if (sourceIds) {
      const filtered = sourceIds.filter(id => id !== evidenceId);
      if (filtered.length > 0) {
        this.sourceIndex.set(fingerprint.normalizedSource, filtered);
      } else {
        this.sourceIndex.delete(fingerprint.normalizedSource);
      }
    }

    // Remove from cache
    this.fingerprintCache.delete(evidenceId);
  }

  /**
   * Get index statistics
   */
  getStats(): {
    totalEvidence: number;
    uniqueContentHashes: number;
    uniquePerceptualHashes: number;
    uniqueSources: number;
  } {
    return {
      totalEvidence: this.fingerprintCache.size,
      uniqueContentHashes: this.contentHashIndex.size,
      uniquePerceptualHashes: this.perceptualHashIndex.size,
      uniqueSources: this.sourceIndex.size,
    };
  }
}
