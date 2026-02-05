/**
 * Knowledge Tools
 *
 * Tools for interacting with the Legible Money knowledge base.
 * Enables querying, verification, claim submission, and challenges.
 *
 * PP HIERARCHY:
 *   TRUTH: All knowledge grounded in verification
 *   UNDERSTANDING: Query results include epistemic metadata
 *   BEAUTY: Coherent knowledge graph structure
 *   JUSTICE: Appropriate confidence levels for each claim
 *   CREATIVITY: Agents can contribute new knowledge
 */

import type { Tool, ServerContext } from "../server/index.js";

// ============ Tool Definitions ============

export const knowledgeTools: Tool[] = [
  {
    name: "lgm_query_knowledge",
    description: `Query the verified knowledge base. Supports:
- Natural language queries
- Domain filtering
- Confidence thresholds
- Graph exploration

Returns facts with full epistemic metadata:
- Confidence level and reasoning
- Verification history
- Citations and relationships
- PP scores (truth, understanding, beauty, justice, creativity)`,
    inputSchema: {
      type: "object",
      properties: {
        query: {
          type: "string",
          description:
            'Natural language query (e.g., "What is the average Bitcoin block time?")',
        },
        intent: {
          type: "string",
          enum: ["understand", "verify", "explore", "connect"],
          description: "Query intent. Default: understand",
        },
        domain: {
          type: "string",
          description:
            'Filter to specific domain (e.g., "cryptocurrency", "mathematics")',
        },
        minConfidence: {
          type: "number",
          description: "Minimum confidence threshold (0.0-1.0). Default: 0.6",
        },
        includeContested: {
          type: "boolean",
          description: "Include contested/unverified claims. Default: false",
        },
        depth: {
          type: "string",
          enum: ["surface", "working", "deep", "expert"],
          description: "Understanding depth. Default: working",
        },
        maxResults: {
          type: "number",
          description: "Maximum facts to return. Default: 10",
        },
      },
      required: ["query"],
    },
  },
  {
    name: "lgm_submit_claim",
    description: `Submit a new knowledge claim for verification. Requirements:
- 100+ LGM stake (refunded on acceptance, slashed on rejection)
- Evidence (citations, proofs, references)
- Self-assessed confidence

The claim enters a verification round where validators vote.
Successful claims earn rewards and become part of the knowledge base.`,
    inputSchema: {
      type: "object",
      properties: {
        claim: {
          type: "string",
          description:
            'The factual claim to submit (e.g., "The speed of light in vacuum is 299,792,458 m/s")',
        },
        domain: {
          type: "string",
          description:
            'Knowledge domain (e.g., "physics", "mathematics", "history")',
        },
        evidence: {
          type: "array",
          items: { type: "string" },
          description:
            "Supporting evidence (URLs, paper references, formal proofs)",
        },
        selfConfidence: {
          type: "number",
          description: "Your confidence in this claim (0.0-1.0)",
        },
        stake: {
          type: "string",
          description: "LGM amount to stake. Minimum: 100 LGM",
        },
        logicZone: {
          type: "string",
          enum: ["propositional", "presburger", "peano", "empirical"],
          description: "Gödelian logic zone. Default: empirical",
        },
      },
      required: ["claim", "domain", "evidence"],
    },
  },
  {
    name: "lgm_verify_claim",
    description: `Participate in a claim verification round. As a verifier:
1. Review the claim and evidence
2. Submit your verdict (confirm/reject/uncertain)
3. Provide confidence and reasoning

Commit-reveal protocol:
- First submit commitment (hashed verdict)
- Then reveal actual verdict after deadline

Rewards for agreeing with consensus, slashing for wrong verdicts.`,
    inputSchema: {
      type: "object",
      properties: {
        claimId: {
          type: "string",
          description: "ID of the claim to verify",
        },
        verdict: {
          type: "string",
          enum: ["confirm", "reject", "uncertain"],
          description: "Your verdict on the claim",
        },
        confidence: {
          type: "number",
          description: "Confidence in your verdict (0.0-1.0)",
        },
        reasoning: {
          type: "string",
          description:
            "Brief reasoning for your verdict (optional but recommended)",
        },
        referencesChecked: {
          type: "array",
          items: { type: "string" },
          description: "Evidence IDs you verified",
        },
      },
      required: ["claimId", "verdict", "confidence"],
    },
  },
  {
    name: "lgm_challenge_claim",
    description: `Challenge an existing verified claim. Use when you believe:
- The claim is factually incorrect
- Evidence has been invalidated
- The claim is outdated
- The claim is misplaced (wrong domain/confidence)

Requirements:
- 50+ LGM stake (rewarded if challenge succeeds)
- Counter-evidence

Successful challenges earn rewards and update the knowledge base.`,
    inputSchema: {
      type: "object",
      properties: {
        claimId: {
          type: "string",
          description: "ID of the claim to challenge",
        },
        challengeType: {
          type: "string",
          enum: ["factual_error", "outdated", "incomplete", "misplaced"],
          description: "Type of challenge",
        },
        counterEvidence: {
          type: "array",
          items: { type: "string" },
          description: "Evidence supporting your challenge",
        },
        explanation: {
          type: "string",
          description: "Detailed explanation of why the claim is wrong",
        },
        stake: {
          type: "string",
          description: "LGM amount to stake. Minimum: 50 LGM",
        },
      },
      required: ["claimId", "challengeType", "counterEvidence", "explanation"],
    },
  },
];

// ============ Tool Handlers ============

export async function handleKnowledgeTool(
  name: string,
  args: Record<string, unknown>,
  context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  switch (name) {
    case "lgm_query_knowledge":
      return handleQueryKnowledge(args, context);
    case "lgm_submit_claim":
      return handleSubmitClaim(args, context);
    case "lgm_verify_claim":
      return handleVerifyClaim(args, context);
    case "lgm_challenge_claim":
      return handleChallengeClaim(args, context);
    default:
      throw new Error(`Unknown knowledge tool: ${name}`);
  }
}

// ============ Handler Implementations ============

async function handleQueryKnowledge(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const query = args.query as string;
  const intent = (args.intent as string) || "understand";
  const domain = args.domain as string | undefined;
  const minConfidence = (args.minConfidence as number) || 0.6;
  const includeContested = (args.includeContested as boolean) || false;
  const depth = (args.depth as string) || "working";
  const maxResults = (args.maxResults as number) || 10;

  // Mock knowledge query results
  const facts = generateMockFacts(query, domain, minConfidence, maxResults);
  const relationships = generateMockRelationships(facts);

  const response = {
    query,
    intent,
    parameters: { domain, minConfidence, includeContested, depth },
    results: {
      factCount: facts.length,
      facts,
      relationships,
    },
    understanding: {
      summary: `Found ${facts.length} relevant facts about "${query}"`,
      keyInsights: facts.slice(0, 3).map((f) => f.claim),
      gaps:
        facts.length < 3 ? [`Limited verified knowledge about "${query}"`] : [],
      suggestedFollowups: [
        { query: `Explore related topics to "${query}"`, intent: "explore" },
        { query: `Verify specific aspect of "${query}"`, intent: "verify" },
      ],
    },
    metadata: {
      ppScores: {
        truth: 0.85,
        understanding: 0.78,
        beauty: 0.72,
        justice: 0.8,
        creativity: 0.25,
      },
      queryTime: Date.now(),
      cacheStatus: "fresh",
    },
  };

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(response, null, 2),
      },
    ],
  };
}

async function handleSubmitClaim(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const claim = args.claim as string;
  const domain = args.domain as string;
  const evidence = args.evidence as string[];
  const selfConfidence = (args.selfConfidence as number) || 0.8;
  const stake = (args.stake as string) || "100";
  const logicZone = (args.logicZone as string) || "empirical";

  // Validate inputs
  if (!claim || claim.length < 10) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: false,
            error: "Claim too short. Please provide a detailed factual claim.",
          }),
        },
      ],
    };
  }

  if (!evidence || evidence.length === 0) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: false,
            error:
              "Evidence required. Please provide at least one supporting reference.",
          }),
        },
      ],
    };
  }

  const stakeAmount = parseFloat(stake);
  if (stakeAmount < 100) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: false,
            error: "Minimum stake is 100 LGM.",
            provided: stakeAmount,
          }),
        },
      ],
    };
  }

  // Create claim submission
  const claimId = `claim-${Date.now()}`;
  const roundId = `round-${claimId}`;

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            success: true,
            claimId,
            status: "pending_verification",
            submission: {
              claim,
              domain,
              logicZone,
              selfConfidence,
              evidenceCount: evidence.length,
              stake: `${stakeAmount} LGM`,
            },
            verificationRound: {
              roundId,
              startBlock: "current + 1",
              commitDeadline: "current + 100 blocks",
              revealDeadline: "current + 200 blocks",
              estimatedCompletion: "~5 minutes",
            },
            economics: {
              staked: `${stakeAmount} LGM`,
              potentialReward: `${stakeAmount * 0.5} LGM (if accepted)`,
              potentialSlash: `${stakeAmount * 0.3} LGM (if rejected)`,
              researchContribution: `${stakeAmount * 0.22} LGM (goes to research fund)`,
            },
            nextSteps: [
              "Wait for verification round to complete",
              "You can track status with lgm_query_knowledge using your claimId",
              "Respond to any verifier questions in the claim discussion",
            ],
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleVerifyClaim(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const claimId = args.claimId as string;
  const verdict = args.verdict as string;
  const confidence = args.confidence as number;
  const reasoning = args.reasoning as string | undefined;
  const referencesChecked = args.referencesChecked as string[] | undefined;

  if (!claimId || !verdict || confidence === undefined) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: false,
            error: "Missing required fields: claimId, verdict, confidence",
          }),
        },
      ],
    };
  }

  // Generate commitment (in real implementation, this would be cryptographic)
  const commitmentId = `commit-${Date.now()}`;

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            success: true,
            action: "commitment_submitted",
            commitmentId,
            claimId,
            submission: {
              verdict,
              confidence,
              reasoning: reasoning || "No reasoning provided",
              referencesChecked: referencesChecked || [],
            },
            roundStatus: {
              phase: "commit",
              commitDeadline: "block 12345",
              revealDeadline: "block 12445",
              totalCommitments: 15,
              requiredCommitments: 10,
            },
            nextSteps: [
              "Your commitment has been recorded",
              "Wait for commit phase to end",
              "Your verdict will be automatically revealed in the reveal phase",
              "Track round progress with lgm_query_knowledge",
            ],
            rewards: {
              ifCorrect: "10 LGM + reputation boost",
              ifWrong: "Partial stake slash + reputation penalty",
              ifMissedReveal: "Full commitment stake slash",
            },
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleChallengeClaim(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const claimId = args.claimId as string;
  const challengeType = args.challengeType as string;
  const counterEvidence = args.counterEvidence as string[];
  const explanation = args.explanation as string;
  const stake = (args.stake as string) || "50";

  if (!claimId || !challengeType || !counterEvidence || !explanation) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: false,
            error:
              "Missing required fields: claimId, challengeType, counterEvidence, explanation",
          }),
        },
      ],
    };
  }

  const stakeAmount = parseFloat(stake);
  if (stakeAmount < 50) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            success: false,
            error: "Minimum challenge stake is 50 LGM.",
          }),
        },
      ],
    };
  }

  const challengeId = `challenge-${Date.now()}`;

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            success: true,
            challengeId,
            targetClaimId: claimId,
            challenge: {
              type: challengeType,
              explanation,
              evidenceCount: counterEvidence.length,
              stake: `${stakeAmount} LGM`,
            },
            disputeRound: {
              phase: "evidence_collection",
              deadline: "block 12500",
              arbitratorsSelected: 7,
            },
            possibleOutcomes: [
              {
                outcome: "challenge_upheld",
                yourReward: `${stakeAmount * 2} LGM + claim stake`,
                claimStatus: "invalidated or updated",
              },
              {
                outcome: "challenge_rejected",
                yourPenalty: `${stakeAmount} LGM (forfeited)`,
                claimStatus: "unchanged",
              },
              {
                outcome: "partial_success",
                yourReward: `${stakeAmount * 0.5} LGM`,
                claimStatus: "confidence reduced",
              },
            ],
            nextSteps: [
              "Monitor dispute resolution progress",
              "Be prepared to provide additional evidence if requested",
              "Arbitrators will review within 1-2 days",
            ],
          },
          null,
          2,
        ),
      },
    ],
  };
}

// ============ Helper Functions ============

function generateMockFacts(
  query: string,
  domain: string | undefined,
  minConfidence: number,
  maxResults: number,
): Array<{
  id: string;
  claim: string;
  domain: string;
  confidence: number;
  confidenceReasoning: string;
  verifiedAt: number;
  verificationCount: number;
  citations: number;
}> {
  const facts = [];
  const count = Math.min(maxResults, 5);

  for (let i = 0; i < count; i++) {
    const confidence = minConfidence + Math.random() * (1 - minConfidence);
    facts.push({
      id: `fact-${Date.now()}-${i}`,
      claim: `Verified fact ${i + 1} related to "${query}"`,
      domain: domain || "general",
      confidence,
      confidenceReasoning: `Based on ${Math.floor(10 + Math.random() * 90)} verifications with ${(confidence * 100).toFixed(0)}% agreement`,
      verifiedAt: Date.now() - Math.random() * 86400000 * 30,
      verificationCount: Math.floor(10 + Math.random() * 90),
      citations: Math.floor(Math.random() * 50),
    });
  }

  return facts;
}

function generateMockRelationships(facts: Array<{ id: string }>): Array<{
  from: string;
  to: string;
  type: string;
  strength: number;
}> {
  if (facts.length < 2) return [];

  return facts.slice(1).map((f, i) => ({
    from: facts[i].id,
    to: f.id,
    type: "supports",
    strength: 0.6 + Math.random() * 0.3,
  }));
}
