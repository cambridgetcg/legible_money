/**
 * Semantic Similarity Index Tests
 *
 * GAP-004: Tests for SemanticSimilarityIndex and StubEmbedder.
 * Tests embedding generation, indexing, similarity search, and filtering.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  SemanticSimilarityIndex,
  StubEmbedder,
  type Embedder,
  type EmbeddingResult,
} from "../../../core/archive/indexes/semantic_index.js";
import type { FactState } from "../../../state/types.js";
import { DEFAULT_ARCHIVE_CONFIG } from "../../../core/archive/config.js";

// ============================================================================
// Test Fixtures
// ============================================================================

function createTestFact(id: string, confidence: number = 0.8): FactState {
  return {
    id,
    claimHash: `claim_hash_${id}`,
    submitter: `did:legible:submitter_${id}`,
    submittedAt: 1000n,
    status: "verified",
    confidence,
    verifications: [],
    challenges: [],
    references: [],
  };
}

function createFreshIndex(): SemanticSimilarityIndex {
  return new SemanticSimilarityIndex(
    DEFAULT_ARCHIVE_CONFIG,
    new StubEmbedder(768),
  );
}

// ============================================================================
// StubEmbedder Tests
// ============================================================================

describe("StubEmbedder", () => {
  let embedder: StubEmbedder;

  beforeEach(() => {
    embedder = new StubEmbedder(768);
  });

  describe("embed()", () => {
    it("should return embedding with correct dimensions", async () => {
      const result = await embedder.embed("test text");

      expect(result.embedding).toBeInstanceOf(Float32Array);
      expect(result.embedding.length).toBe(768);
      expect(result.dimensions).toBe(768);
      expect(result.model).toBe("stub");
    });

    it("should return normalized embedding (unit vector)", async () => {
      const result = await embedder.embed("test text");

      // Calculate magnitude
      let magnitude = 0;
      for (let i = 0; i < result.embedding.length; i++) {
        magnitude += result.embedding[i] * result.embedding[i];
      }
      magnitude = Math.sqrt(magnitude);

      // Should be approximately 1.0
      expect(magnitude).toBeCloseTo(1.0, 5);
    });

    it("should be deterministic - same text produces same embedding", async () => {
      const result1 = await embedder.embed("consistent text");
      const result2 = await embedder.embed("consistent text");

      for (let i = 0; i < result1.embedding.length; i++) {
        expect(result1.embedding[i]).toBe(result2.embedding[i]);
      }
    });

    it("should produce different embeddings for different text", async () => {
      const result1 = await embedder.embed("first text");
      const result2 = await embedder.embed("second text");

      let differences = 0;
      for (let i = 0; i < result1.embedding.length; i++) {
        if (result1.embedding[i] !== result2.embedding[i]) {
          differences++;
        }
      }

      expect(differences).toBeGreaterThan(0);
    });

    it("should handle empty string", async () => {
      const result = await embedder.embed("");

      expect(result.embedding.length).toBe(768);
    });
  });

  describe("embedBatch()", () => {
    it("should return embeddings for all texts", async () => {
      const texts = ["first", "second", "third"];
      const results = await embedder.embedBatch(texts);

      expect(results.length).toBe(3);
      for (const result of results) {
        expect(result.embedding.length).toBe(768);
        expect(result.dimensions).toBe(768);
      }
    });

    it("should be consistent with single embed()", async () => {
      const singleResult = await embedder.embed("test");
      const batchResults = await embedder.embedBatch(["test"]);

      for (let i = 0; i < singleResult.embedding.length; i++) {
        expect(singleResult.embedding[i]).toBe(batchResults[0].embedding[i]);
      }
    });

    it("should handle empty array", async () => {
      const results = await embedder.embedBatch([]);
      expect(results).toEqual([]);
    });
  });

  describe("getDimensions()", () => {
    it("should return configured dimensions", () => {
      expect(embedder.getDimensions()).toBe(768);
    });

    it("should respect custom dimensions", () => {
      const customEmbedder = new StubEmbedder(256);
      expect(customEmbedder.getDimensions()).toBe(256);
    });
  });
});

// ============================================================================
// SemanticSimilarityIndex Tests
// ============================================================================

describe("SemanticSimilarityIndex", () => {
  describe("indexFact()", () => {
    it("should index a fact with generated embedding", async () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");

      await index.indexFact(
        fact,
        "The sky is blue",
        "physics.optics",
        "empirical",
        100n,
      );

      const entry = index.getEntry("fact-1");
      expect(entry).toBeDefined();
      expect(entry!.factId).toBe("fact-1");
      expect(entry!.claim).toBe("The sky is blue");
      expect(entry!.domainId).toBe("physics.optics");
      expect(entry!.stratum).toBe("empirical");
      expect(entry!.embedding.length).toBe(768);
    });

    it("should add fact to domain index", async () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");

      await index.indexFact(fact, "claim", "physics", "empirical", 100n);

      const stats = index.getStatistics();
      expect(stats.domainsIndexed).toBe(1);
      expect(stats.totalEmbeddings).toBe(1);
    });

    it("should handle multiple facts in same domain", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "claim 1",
        "physics",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "claim 2",
        "physics",
        "empirical",
        101n,
      );

      const stats = index.getStatistics();
      expect(stats.domainsIndexed).toBe(1);
      expect(stats.totalEmbeddings).toBe(2);
    });
  });

  describe("indexFactWithEmbedding()", () => {
    it("should index a fact with pre-computed embedding", () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");
      const embedding = new Float32Array(768).fill(0.01);

      index.indexFactWithEmbedding(
        fact,
        "Pre-computed claim",
        embedding,
        "math",
        "formal",
        100n,
      );

      const entry = index.getEntry("fact-1");
      expect(entry).toBeDefined();
      expect(entry!.embedding).toBe(embedding);
    });
  });

  describe("removeFact()", () => {
    it("should remove a fact from the index", async () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");
      await index.indexFact(fact, "claim", "physics", "empirical", 100n);

      expect(index.getEntry("fact-1")).toBeDefined();

      index.removeFact("fact-1");

      expect(index.getEntry("fact-1")).toBeUndefined();
    });

    it("should remove fact from domain index", async () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");
      await index.indexFact(fact, "claim", "physics", "empirical", 100n);

      index.removeFact("fact-1");

      const stats = index.getStatistics();
      expect(stats.totalEmbeddings).toBe(0);
    });

    it("should handle removing non-existent fact", () => {
      const index = createFreshIndex();
      expect(() => index.removeFact("non-existent")).not.toThrow();
    });
  });

  describe("searchByText()", () => {
    it("should find similar facts by text query", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1", 0.9),
        "The sky is blue due to Rayleigh scattering",
        "physics.optics",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2", 0.8),
        "Water refracts light",
        "physics.optics",
        "empirical",
        101n,
      );

      // Use minSimilarity: -1 to include all results since StubEmbedder generates
      // pseudo-random embeddings where cosine similarity can be negative
      const results = await index.searchByText("blue sky light", {
        minSimilarity: -1,
      });

      expect(results.length).toBeGreaterThan(0);
      expect(results[0]).toHaveProperty("factId");
      expect(results[0]).toHaveProperty("similarityScore");
    });

    it("should return results sorted by similarity (descending)", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "claim one",
        "domain",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "claim two",
        "domain",
        "empirical",
        101n,
      );
      await index.indexFact(
        createTestFact("fact-3"),
        "claim three",
        "domain",
        "empirical",
        102n,
      );

      const results = await index.searchByText("claim", { minSimilarity: 0 });

      for (let i = 1; i < results.length; i++) {
        expect(results[i - 1].similarityScore).toBeGreaterThanOrEqual(
          results[i].similarityScore,
        );
      }
    });

    it("should respect minSimilarity threshold", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "sky",
        "domain",
        "empirical",
        100n,
      );

      const results = await index.searchByText("sky", { minSimilarity: 0.9 });

      for (const result of results) {
        expect(result.similarityScore).toBeGreaterThanOrEqual(0.9);
      }
    });

    it("should respect limit option", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "light one",
        "domain",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "light two",
        "domain",
        "empirical",
        101n,
      );
      await index.indexFact(
        createTestFact("fact-3"),
        "light three",
        "domain",
        "empirical",
        102n,
      );

      const results = await index.searchByText("light", {
        minSimilarity: 0,
        limit: 2,
      });

      expect(results.length).toBeLessThanOrEqual(2);
    });

    it("should filter by domain", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "physics claim",
        "physics.optics",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "math claim",
        "math.algebra",
        "formal",
        101n,
      );

      const results = await index.searchByText("claim", {
        minSimilarity: 0,
        domain: "physics.optics",
      });

      for (const result of results) {
        expect(result.domain).toBe("physics.optics");
      }
    });

    it("should filter by stratum", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "empirical claim",
        "domain",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "formal claim",
        "domain2",
        "formal",
        101n,
      );

      const results = await index.searchByText("claim", {
        minSimilarity: 0,
        stratum: "formal",
      });

      for (const result of results) {
        expect(result.stratum).toBe("formal");
      }
    });

    it("should exclude specified fact IDs", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "light one",
        "domain",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "light two",
        "domain",
        "empirical",
        101n,
      );

      const results = await index.searchByText("light", {
        minSimilarity: 0,
        excludeIds: ["fact-1"],
      });

      const factIds = results.map((r) => r.factId);
      expect(factIds).not.toContain("fact-1");
    });
  });

  describe("findSimilar()", () => {
    it("should find facts similar to an indexed fact", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1", 0.9),
        "Light travels at 299792458 m/s",
        "physics.optics",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2", 0.8),
        "The speed of light is constant",
        "physics.optics",
        "empirical",
        101n,
      );
      await index.indexFact(
        createTestFact("fact-3", 0.7),
        "Apples are fruits",
        "biology.botany",
        "empirical",
        102n,
      );

      const results = index.findSimilar("fact-1", { minSimilarity: 0 });

      expect(results.length).toBeGreaterThan(0);
      // Should not include the source fact itself
      const factIds = results.map((r) => r.factId);
      expect(factIds).not.toContain("fact-1");
    });

    it("should return empty array for non-existent fact", () => {
      const index = createFreshIndex();
      const results = index.findSimilar("non-existent");
      expect(results).toEqual([]);
    });
  });

  describe("findSimilarInDomain()", () => {
    it("should find similar facts only within specified domain", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "Claim in physics",
        "physics",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "Another physics claim",
        "physics",
        "empirical",
        101n,
      );
      await index.indexFact(
        createTestFact("fact-3"),
        "Math claim",
        "math",
        "formal",
        102n,
      );

      const results = index.findSimilarInDomain("fact-1", "physics", {
        minSimilarity: 0,
      });

      for (const result of results) {
        expect(result.domain).toBe("physics");
      }
    });

    it("should return empty for non-existent domain", async () => {
      const index = createFreshIndex();
      await index.indexFact(
        createTestFact("fact-1"),
        "claim",
        "physics",
        "empirical",
        100n,
      );

      const results = index.findSimilarInDomain("fact-1", "non-existent");
      expect(results).toEqual([]);
    });
  });

  describe("findSimilarByEmbedding()", () => {
    it("should find similar facts by direct embedding comparison", async () => {
      const index = createFreshIndex();
      const embedder = new StubEmbedder(768);

      await index.indexFact(
        createTestFact("fact-1"),
        "Test claim",
        "domain",
        "empirical",
        100n,
      );

      const queryEmbedding = (await embedder.embed("Test claim")).embedding;
      const results = index.findSimilarByEmbedding(queryEmbedding, {
        minSimilarity: 0.5,
      });

      expect(results.length).toBeGreaterThan(0);
      // Same text should have high similarity
      expect(results[0].similarityScore).toBeGreaterThan(0.9);
    });
  });

  describe("getEmbedding()", () => {
    it("should return embedding for indexed fact", async () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");
      await index.indexFact(fact, "claim", "domain", "empirical", 100n);

      const embedding = index.getEmbedding("fact-1");

      expect(embedding).toBeDefined();
      expect(embedding).toBeInstanceOf(Float32Array);
    });

    it("should return undefined for non-existent fact", () => {
      const index = createFreshIndex();
      const embedding = index.getEmbedding("non-existent");
      expect(embedding).toBeUndefined();
    });
  });

  describe("getStatistics()", () => {
    it("should return correct statistics for empty index", () => {
      const index = createFreshIndex();
      const stats = index.getStatistics();

      expect(stats.totalEmbeddings).toBe(0);
      expect(stats.domainsIndexed).toBe(0);
      expect(stats.avgEmbeddingsPerDomain).toBe(0);
    });

    it("should return correct statistics after indexing", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "claim 1",
        "domain-a",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "claim 2",
        "domain-a",
        "empirical",
        101n,
      );
      await index.indexFact(
        createTestFact("fact-3"),
        "claim 3",
        "domain-b",
        "formal",
        102n,
      );

      const stats = index.getStatistics();

      expect(stats.totalEmbeddings).toBe(3);
      expect(stats.domainsIndexed).toBe(2);
      expect(stats.embeddingDimensions).toBe(768);
      expect(stats.avgEmbeddingsPerDomain).toBe(1.5); // 3 facts / 2 domains
    });
  });

  describe("setEmbedder()", () => {
    it("should allow changing the embedder", async () => {
      const index = createFreshIndex();
      const customEmbedder = new StubEmbedder(256);
      index.setEmbedder(customEmbedder);

      expect(index.getEmbedder()).toBe(customEmbedder);
      expect(index.getEmbedder().getDimensions()).toBe(256);
    });
  });

  describe("clear()", () => {
    it("should remove all entries from the index", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "claim",
        "domain",
        "empirical",
        100n,
      );

      index.clear();

      const stats = index.getStatistics();
      expect(stats.totalEmbeddings).toBe(0);
      expect(stats.domainsIndexed).toBe(0);
    });
  });

  describe("serialization", () => {
    it("should serialize and deserialize embeddings", async () => {
      const index = createFreshIndex();
      const fact = createTestFact("fact-1");
      await index.indexFact(fact, "claim", "domain", "empirical", 100n);

      const embedding = index.getEmbedding("fact-1")!;
      const serialized = index.serializeEmbedding(embedding);
      const deserialized = index.deserializeEmbedding(serialized);

      expect(deserialized.length).toBe(embedding.length);
      for (let i = 0; i < embedding.length; i++) {
        expect(deserialized[i]).toBeCloseTo(embedding[i], 5);
      }
    });

    it("should generate correct storage keys", () => {
      const index = createFreshIndex();
      const key = index.makeStorageKey("fact-123");
      expect(key).toContain("fact-123");
    });
  });

  describe("edge cases", () => {
    it("should handle dimension mismatch gracefully", () => {
      const index = createFreshIndex();
      // Create embedding with wrong dimensions
      const wrongDimEmbedding = new Float32Array(128);
      const correctDimEmbedding = new Float32Array(768);

      // Cosine similarity should return 0 for dimension mismatch
      const similarity = (index as any).cosineSimilarity(
        wrongDimEmbedding,
        correctDimEmbedding,
      );
      expect(similarity).toBe(0);
    });

    it("should handle empty domain filter", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "claim",
        "domain",
        "empirical",
        100n,
      );

      const results = await index.searchByText("claim", {
        domain: "non-existent-domain",
      });
      expect(results).toEqual([]);
    });

    it("should handle identical embeddings (similarity = 1.0)", async () => {
      const index = createFreshIndex();

      await index.indexFact(
        createTestFact("fact-1"),
        "identical claim",
        "domain",
        "empirical",
        100n,
      );
      await index.indexFact(
        createTestFact("fact-2"),
        "identical claim",
        "domain",
        "empirical",
        101n,
      );

      const results = index.findSimilar("fact-1", { minSimilarity: 0.99 });

      // Should find fact-2 with very high similarity
      expect(results.some((r) => r.factId === "fact-2")).toBe(true);
      const fact2Result = results.find((r) => r.factId === "fact-2");
      expect(fact2Result?.similarityScore).toBeGreaterThan(0.99);
    });

    it("should handle zero vector (degenerate case)", () => {
      const index = createFreshIndex();
      const zeroVector = new Float32Array(768).fill(0);
      const normalVector = new Float32Array(768).fill(0.01);

      const similarity = (index as any).cosineSimilarity(
        zeroVector,
        normalVector,
      );
      expect(similarity).toBe(0);
    });
  });
});

// ============================================================================
// Custom Embedder Tests
// ============================================================================

describe("Custom Embedder Integration", () => {
  it("should work with custom embedder implementation", async () => {
    // Create a simple mock embedder
    const mockEmbedder: Embedder = {
      async embed(text: string): Promise<EmbeddingResult> {
        const embedding = new Float32Array(128);
        for (let i = 0; i < 128; i++) {
          embedding[i] = text.length > i ? 0.1 : -0.1;
        }
        return { embedding, model: "mock", dimensions: 128 };
      },
      async embedBatch(texts: string[]): Promise<EmbeddingResult[]> {
        return Promise.all(texts.map((t) => this.embed(t)));
      },
      getDimensions(): number {
        return 128;
      },
    };

    const config = {
      ...DEFAULT_ARCHIVE_CONFIG,
      embeddingDimensions: 128,
    };
    const index = new SemanticSimilarityIndex(config, mockEmbedder);

    await index.indexFact(
      createTestFact("fact-1"),
      "short",
      "domain",
      "empirical",
      100n,
    );

    const entry = index.getEntry("fact-1");
    expect(entry?.embedding.length).toBe(128);
  });
});
