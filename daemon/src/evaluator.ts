/**
 * Claude Evaluator
 *
 * Wraps the Anthropic SDK to evaluate knowledge claims using Claude.
 * Falls back to deterministic evaluation (mirroring x/knowledge/evaluation/
 * category.go) when Claude is unavailable or times out.
 *
 * The system prompt explains PoT evaluation criteria, mirroring the 6-stage
 * pipeline from x/knowledge/evaluation/engine.go.
 */

import Anthropic from "@anthropic-ai/sdk";
import type { ClaimData, EvaluationResult, Verdict } from "./types.js";

/** Category base confidence values matching x/knowledge/evaluation/category.go:5-15 */
const CATEGORY_BASE_CONFIDENCE: Record<string, number> = {
  protocol: 800_000,
  analytic: 700_000,
  formal: 700_000,
  computational: 650_000,
  replicated: 550_000,
  empirical: 500_000,
  historical: 450_000,
  social: 400_000,
  predictive: 350_000,
};

/** Default for unknown categories (matches category.go:17) */
const DEFAULT_CATEGORY_BASE = 400_000;

/** Verdict thresholds (matches evaluation/engine.go) */
const ACCEPT_THRESHOLD = 600_000;
const REJECT_THRESHOLD = 300_000;

const SYSTEM_PROMPT = `You are a truth verification agent for the Legible Money protocol.
You evaluate knowledge claims submitted to the blockchain for accuracy and verifiability.

Evaluate using these criteria:

1. EPISTEMIC CATEGORY base confidence:
   protocol=800k, analytic/formal=700k, computational=650k,
   replicated=550k, empirical=500k, historical=450k,
   social=400k, predictive=350k

2. EVIDENCE quality (type strength × reliability):
   on_chain(1M), computation(950k), experiment(850k),
   oracle_feed(700k), document(600k), attestation(500k)
   Note: oracle_feed includes prediction market resolutions (Polymarket,
   Kalshi) with real economic payouts. High volume and many traders
   indicate stronger truth signals. Weight accordingly.

3. FACT GRAPH: verified references boost (+30k each, max +150k),
   disproven references penalize (-50k each)

4. LOGIC ZONE: Higher zones have confidence ceilings
   propositional=1M, presburger=900k, peano=750k, empirical=600k, goedelian=500k
   (Gödelian incompleteness → abstain if unverifiable)

5. VERDICT THRESHOLDS:
   >= 600,000 → "accept"
   < 300,000  → "reject"
   between    → "abstain"

6. COUNTER-FACTS (negative knowledge):
   If referenced facts have verified counter-facts, apply penalties:
   - High confidence counter-fact (>=700k): -80,000 per counter-fact
   - Medium confidence (>=400k): -40,000 per counter-fact
   - Low confidence (<400k): -10,000 per counter-fact
   - If any referenced fact has bidirectional status "disproven", force REJECT
   Total counter-fact penalty capped at -400,000.

Respond with ONLY valid JSON, no markdown fences or extra text:
{"verdict":"accept"|"reject"|"abstain","confidence":<integer 0-1000000>,"justification":"<one sentence>"}`;

export class ClaudeEvaluator {
  private readonly client: Anthropic;
  private readonly model: string;
  private readonly maxTokens: number;
  private readonly timeoutMs: number;
  private readonly temperature: number;

  constructor(
    config: {
      model?: string;
      maxTokens?: number;
      timeoutMs?: number;
      temperature?: number;
    } = {},
  ) {
    // Anthropic SDK reads ANTHROPIC_API_KEY from env by default
    this.client = new Anthropic();
    this.model = config.model ?? "claude-sonnet-4-20250514";
    this.maxTokens = config.maxTokens ?? 512;
    this.timeoutMs = config.timeoutMs ?? 6000;
    this.temperature = config.temperature ?? 0.0;
  }

  /**
   * Evaluate a claim using Claude, falling back to deterministic evaluation
   * on timeout or error.
   */
  async evaluate(claim: ClaimData): Promise<EvaluationResult> {
    const start = Date.now();

    try {
      const result = await this.evaluateWithClaude(claim);
      result.latencyMs = Date.now() - start;
      return result;
    } catch {
      // Timeout, API error, or parse failure → deterministic fallback
      return this.evaluateDeterministic(claim);
    }
  }

  /**
   * Call Claude API with timeout to evaluate a claim.
   */
  private async evaluateWithClaude(
    claim: ClaimData,
  ): Promise<EvaluationResult> {
    const userPrompt = this.buildUserPrompt(claim);

    // Race Claude against timeout
    const response = await Promise.race([
      this.client.messages.create({
        model: this.model,
        max_tokens: this.maxTokens,
        temperature: this.temperature,
        system: SYSTEM_PROMPT,
        messages: [{ role: "user", content: userPrompt }],
      }),
      rejectAfter(this.timeoutMs),
    ]);

    // Extract text content
    const textBlock = response.content.find((b) => b.type === "text");
    if (!textBlock || textBlock.type !== "text") {
      throw new Error("No text content in Claude response");
    }

    return this.parseResponse(textBlock.text);
  }

  /**
   * Deterministic fallback using category base confidence.
   * Mirrors x/knowledge/evaluation/category.go:19-32.
   */
  evaluateDeterministic(claim: ClaimData): EvaluationResult {
    const confidence =
      CATEGORY_BASE_CONFIDENCE[claim.epistemicCategory] ??
      DEFAULT_CATEGORY_BASE;
    const verdict = this.verdictFromConfidence(confidence);

    return {
      verdict,
      confidence,
      justification: `deterministic fallback: base confidence for ${claim.epistemicCategory}`,
      source: "deterministic",
      latencyMs: 0,
    };
  }

  /** Build user prompt from claim data */
  private buildUserPrompt(claim: ClaimData): string {
    const parts: string[] = [
      `CLAIM: "${claim.claimText}"`,
      `DOMAIN: ${claim.domain}`,
      `EPISTEMIC CATEGORY: ${claim.epistemicCategory}`,
      `LOGIC ZONE: ${claim.logicZone}`,
      `SUBMITTER STAKE: ${claim.stakeAmount} LGM`,
    ];

    if (claim.evidence.length > 0) {
      parts.push("EVIDENCE:");
      for (const ev of claim.evidence) {
        const status = ev.status !== "pending" ? ` [${ev.status}]` : "";
        parts.push(
          `  - [${ev.evidenceType}] ${ev.content}${status} (reliability: ${ev.reliability})`,
        );
      }
    } else {
      parts.push("EVIDENCE: none");
    }

    if (claim.references.length > 0) {
      parts.push("REFERENCED FACTS:");
      for (const ref of claim.references) {
        let refLine = `  - ${ref.factId}: ${ref.status} (confidence: ${ref.confidence})`;
        // Include bidirectional confidence if available
        if (ref.bidirectional && ref.bidirectional.counterFactCount > 0) {
          refLine += ` [COUNTER-FACTS: ${ref.bidirectional.counterFactCount}, net: ${ref.bidirectional.netConfidence}, status: ${ref.bidirectional.status}]`;
        }
        parts.push(refLine);
        // List individual counter-facts
        if (ref.counterFacts && ref.counterFacts.length > 0) {
          for (const cf of ref.counterFacts) {
            parts.push(
              `    ↳ counter-fact ${cf.id}: ${cf.negationType} (confidence: ${cf.confidence}, status: ${cf.status})`,
            );
          }
        }
      }
    } else {
      parts.push("REFERENCED FACTS: none");
    }

    return parts.join("\n");
  }

  /** Parse Claude's JSON response, with regex fallback for markdown-wrapped JSON */
  private parseResponse(text: string): EvaluationResult {
    // Try direct JSON parse
    let parsed = tryParseJSON(text);

    // Fallback: extract JSON block from markdown or surrounding text
    if (!parsed) {
      const match = text.match(/\{[^}]*"verdict"[^}]*\}/);
      if (match) {
        parsed = tryParseJSON(match[0]);
      }
    }

    if (!parsed || !isValidEvaluation(parsed)) {
      throw new Error(`Failed to parse Claude response: ${text.slice(0, 200)}`);
    }

    // Clamp confidence to valid range
    const confidence = Math.max(
      0,
      Math.min(1_000_000, Math.round(parsed.confidence as number)),
    );

    return {
      verdict: parsed.verdict as Verdict,
      confidence,
      justification: String(parsed.justification ?? ""),
      source: "claude",
      latencyMs: 0,
    };
  }

  /** Map confidence to verdict using standard thresholds */
  private verdictFromConfidence(confidence: number): Verdict {
    if (confidence >= ACCEPT_THRESHOLD) return "accept";
    if (confidence < REJECT_THRESHOLD) return "reject";
    return "abstain";
  }
}

// ============ Helpers ============

function tryParseJSON(text: string): Record<string, unknown> | null {
  try {
    const obj = JSON.parse(text.trim());
    if (typeof obj === "object" && obj !== null) return obj;
    return null;
  } catch {
    return null;
  }
}

function isValidEvaluation(obj: Record<string, unknown>): boolean {
  return (
    typeof obj.verdict === "string" &&
    ["accept", "reject", "abstain"].includes(obj.verdict) &&
    typeof obj.confidence === "number" &&
    isFinite(obj.confidence as number)
  );
}

/** Promise that rejects after the given timeout */
function rejectAfter(ms: number): Promise<never> {
  return new Promise<never>((_, reject) => {
    setTimeout(
      () => reject(new Error(`Claude evaluation timed out after ${ms}ms`)),
      ms,
    );
  });
}
