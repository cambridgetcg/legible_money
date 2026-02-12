/**
 * Claude Evaluator Tests
 *
 * Tests Claude evaluation with mocked API, response parsing,
 * timeout fallback, and deterministic evaluation.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ClaimData } from "../src/types.js";

// Shared mock function accessible to all tests
const mockCreate = vi.fn();

// Mock the Anthropic SDK
vi.mock("@anthropic-ai/sdk", () => {
  return {
    default: vi.fn().mockImplementation(() => ({
      messages: {
        create: mockCreate,
      },
    })),
  };
});

import { ClaudeEvaluator } from "../src/evaluator.js";

function makeClaim(overrides: Partial<ClaimData> = {}): ClaimData {
  return {
    id: "claim-1",
    claimText: "The speed of light in vacuum is approximately 299,792,458 m/s",
    domain: "physics",
    epistemicCategory: "empirical",
    logicZone: "empirical",
    references: [],
    evidence: [],
    submitter: "did:lgm:abc123",
    stakeAmount: "1000",
    ...overrides,
  };
}

// Sequential to avoid shared mock interference
describe.sequential("ClaudeEvaluator", () => {
  let evaluator: ClaudeEvaluator;

  beforeEach(() => {
    mockCreate.mockClear();
    evaluator = new ClaudeEvaluator({ timeoutMs: 5000 });
  });

  describe("evaluate with Claude", () => {
    it("parses a valid JSON response", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: '{"verdict":"accept","confidence":750000,"justification":"Well-established physical constant"}',
          },
        ],
      });

      const result = await evaluator.evaluate(makeClaim());

      expect(result.verdict).toBe("accept");
      expect(result.confidence).toBe(750_000);
      expect(result.justification).toBe("Well-established physical constant");
      expect(result.source).toBe("claude");
    });

    it("parses JSON wrapped in surrounding text", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: 'Here is my evaluation:\n{"verdict":"reject","confidence":200000,"justification":"Unverifiable claim"}\n',
          },
        ],
      });

      const result = await evaluator.evaluate(makeClaim());
      expect(result.verdict).toBe("reject");
      expect(result.confidence).toBe(200_000);
      expect(result.source).toBe("claude");
    });

    it("clamps confidence above max to 1,000,000", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: '{"verdict":"accept","confidence":1500000,"justification":"over max"}',
          },
        ],
      });

      const result = await evaluator.evaluate(makeClaim());
      expect(result.confidence).toBe(1_000_000);
      expect(result.source).toBe("claude");
    });

    it("clamps negative confidence to 0", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: '{"verdict":"reject","confidence":-100,"justification":"negative"}',
          },
        ],
      });

      const result = await evaluator.evaluate(makeClaim());
      expect(result.confidence).toBe(0);
      expect(result.source).toBe("claude");
    });

    it("includes claim data in user prompt", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: '{"verdict":"abstain","confidence":450000,"justification":"uncertain"}',
          },
        ],
      });

      const claim = makeClaim({
        claimText: "Water boils at 100C at standard pressure",
        domain: "chemistry",
        epistemicCategory: "replicated",
        evidence: [
          {
            id: "ev-1",
            evidenceType: "experiment",
            content: "Lab measurement",
            reliability: 900_000,
            status: "verified",
          },
        ],
        references: [
          {
            factId: "fact-1",
            status: "verified",
            confidence: 800_000,
          },
        ],
      });

      await evaluator.evaluate(claim);

      // Verify the call was made with system + user messages
      expect(mockCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          messages: expect.arrayContaining([
            expect.objectContaining({
              role: "user",
              content: expect.stringContaining("Water boils"),
            }),
          ]),
        }),
      );
    });
  });

  describe("timeout and fallback", () => {
    it("falls back to deterministic on timeout", async () => {
      // Simulate Claude taking too long
      mockCreate.mockImplementation(
        () => new Promise((resolve) => setTimeout(resolve, 60000)),
      );

      const evaluatorFast = new ClaudeEvaluator({ timeoutMs: 50 });
      const result = await evaluatorFast.evaluate(
        makeClaim({ epistemicCategory: "empirical" }),
      );

      expect(result.source).toBe("deterministic");
      expect(result.confidence).toBe(500_000); // empirical base
      expect(result.verdict).toBe("abstain"); // 500k is between thresholds
    });

    it("falls back to deterministic on API error", async () => {
      mockCreate.mockRejectedValue(new Error("API rate limit exceeded"));

      const result = await evaluator.evaluate(
        makeClaim({ epistemicCategory: "protocol" }),
      );

      expect(result.source).toBe("deterministic");
      expect(result.confidence).toBe(800_000); // protocol base
      expect(result.verdict).toBe("accept"); // 800k >= 600k threshold
    });

    it("falls back on malformed response (no text content)", async () => {
      mockCreate.mockResolvedValue({
        content: [{ type: "image", source: {} }],
      });

      const result = await evaluator.evaluate(makeClaim());
      expect(result.source).toBe("deterministic");
    });

    it("falls back on invalid JSON in response", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: "I cannot evaluate this claim because reasons",
          },
        ],
      });

      const result = await evaluator.evaluate(makeClaim());
      expect(result.source).toBe("deterministic");
    });

    it("falls back on JSON with missing verdict", async () => {
      mockCreate.mockResolvedValue({
        content: [
          {
            type: "text",
            text: '{"confidence":500000,"justification":"missing verdict"}',
          },
        ],
      });

      const result = await evaluator.evaluate(makeClaim());
      expect(result.source).toBe("deterministic");
    });
  });

  describe("deterministic evaluation", () => {
    it("returns correct base for each category", () => {
      const cases: [string, number, string][] = [
        ["protocol", 800_000, "accept"],
        ["analytic", 700_000, "accept"],
        ["formal", 700_000, "accept"],
        ["computational", 650_000, "accept"],
        ["replicated", 550_000, "abstain"],
        ["empirical", 500_000, "abstain"],
        ["historical", 450_000, "abstain"],
        ["social", 400_000, "abstain"],
        ["predictive", 350_000, "abstain"],
      ];

      for (const [category, expectedConfidence, expectedVerdict] of cases) {
        const result = evaluator.evaluateDeterministic(
          makeClaim({ epistemicCategory: category }),
        );
        expect(result.confidence).toBe(expectedConfidence);
        expect(result.verdict).toBe(expectedVerdict);
        expect(result.source).toBe("deterministic");
      }
    });

    it("returns default base (400k) for unknown category", () => {
      const result = evaluator.evaluateDeterministic(
        makeClaim({ epistemicCategory: "unknown_category" }),
      );
      expect(result.confidence).toBe(400_000);
      expect(result.verdict).toBe("abstain");
    });

    it("has zero latency", () => {
      const result = evaluator.evaluateDeterministic(makeClaim());
      expect(result.latencyMs).toBe(0);
    });
  });
});
