/**
 * Tool Descriptors for LLM Function Calling
 *
 * Provides structured descriptions of AgentSDK methods
 * optimized for Claude's tool use format.
 *
 * Each tool descriptor includes:
 * - name: Method name
 * - description: What it does (for LLM understanding)
 * - category: Logical grouping
 * - parameters: JSON Schema for inputs
 * - preconditions: What must be true before calling
 * - outcomes: What happens when called
 */

import type { ToolDescriptor, MethodDescription } from "./types/agent.js";

// ============ Tool Descriptors ============

export const TOOL_DESCRIPTORS: ToolDescriptor[] = [
  // ============ Introspection Tools ============
  {
    name: "whoAmI",
    description:
      "Get comprehensive information about the current agent identity, including DID, balances, reputation, and capabilities. Use this at session start or when you need to understand your current state.",
    category: "introspection",
    parameters: {
      type: "object",
      properties: {},
      required: [],
    },
    preconditions: ["Agent must be initialized (ensureAccount called)"],
    outcomes: [
      "Returns AgentIdentity with DID, public key, reputation, balances, and capabilities",
    ],
    examples: [
      {
        description: "Check identity at session start",
        params: {},
        expectedOutcome:
          "Returns full identity including did:lgm:xxx, balance, reputation score",
      },
    ],
  },

  {
    name: "whatCanIDo",
    description:
      "Get available actions based on current state. Returns what actions are possible, constraints, and suggestions for what to do next. Use this to understand your operational capabilities.",
    category: "introspection",
    parameters: {
      type: "object",
      properties: {
        balance: {
          type: "string",
          description: "Optional: Override balance to check (as wei string)",
        },
      },
      required: [],
    },
    preconditions: ["Agent must be initialized"],
    outcomes: [
      "Returns available actions (canSubmitClaim, canVerify, canTransfer, etc.)",
      "Includes constraints like maxClaimStake and maxTransfer",
      "Provides suggestions for next actions",
    ],
    examples: [
      {
        description: "Check what actions are available",
        params: {},
        expectedOutcome:
          'Shows canSubmitClaim: true, suggestions: ["You have pending verifications"]',
      },
    ],
  },

  {
    name: "whatDoIBelieve",
    description:
      "Get facts the agent has contributed to or interacted with. Returns beliefs with confidence levels and identifies uncertainties. Use this to understand your knowledge state.",
    category: "introspection",
    parameters: {
      type: "object",
      properties: {
        domain: {
          type: "string",
          description: "Filter beliefs to a specific knowledge domain",
        },
        minConfidence: {
          type: "number",
          description: "Minimum confidence threshold (0-1)",
        },
      },
      required: [],
    },
    preconditions: ["Agent must be initialized"],
    outcomes: [
      "Returns beliefs with confidence levels",
      "Identifies uncertain facts needing attention",
      "Provides summary statistics",
    ],
  },

  // ============ Decision Support Tools ============
  {
    name: "shouldI",
    description:
      "Get advice about whether to perform an action. Evaluates risks, suggests alternatives, and identifies prerequisites. Use this before important decisions.",
    category: "treasury",
    parameters: {
      type: "object",
      properties: {
        action: {
          type: "string",
          description:
            "Action type: transfer, stake, unstake, submit_claim, challenge, verify",
          enum: [
            "transfer",
            "stake",
            "unstake",
            "submit_claim",
            "challenge",
            "verify",
          ],
        },
        params: {
          type: "object",
          description: "Parameters for the action (varies by action type)",
        },
      },
      required: ["action", "params"],
    },
    preconditions: ["Agent must be initialized"],
    outcomes: [
      "Returns recommendation (true/false) with confidence",
      "Lists identified risks",
      "Suggests alternatives if action is risky",
      "Lists prerequisites that should be completed first",
    ],
    examples: [
      {
        description: "Check if transfer is advisable",
        params: {
          action: "transfer",
          params: { to: "did:lgm:recipient", amount: "1000000000000000000" },
        },
        expectedOutcome: "Recommendation with risk assessment and alternatives",
      },
    ],
  },

  {
    name: "preview",
    description:
      "Preview the outcome of an action without executing it. Shows estimated costs, rewards, state changes, and potential issues. Use this to understand consequences before acting.",
    category: "treasury",
    parameters: {
      type: "object",
      properties: {
        action: {
          type: "string",
          description:
            "Action type: transfer, stake, unstake, submit_claim, challenge, verify",
          enum: [
            "transfer",
            "stake",
            "unstake",
            "submit_claim",
            "challenge",
            "verify",
          ],
        },
        params: {
          type: "object",
          description: "Parameters for the action",
        },
      },
      required: ["action", "params"],
    },
    preconditions: ["Agent must be initialized"],
    outcomes: [
      "Shows wouldSucceed (true/false)",
      "Lists estimatedCost and estimatedReward",
      "Details stateChanges that would occur",
      "Lists warnings and blockers",
    ],
    examples: [
      {
        description: "Preview a claim submission",
        params: {
          action: "submit_claim",
          params: {
            claim: "The speed of light is approximately 299,792,458 m/s",
            domain: "physics",
            stake: "1000000000000000000",
          },
        },
        expectedOutcome:
          "Preview showing cost, potential reward, and any warnings",
      },
    ],
  },

  {
    name: "explain",
    description:
      "Explain what happened with a transaction. Shows outcome, effects, reasoning, and suggested next steps. Use this to understand past actions.",
    category: "treasury",
    parameters: {
      type: "object",
      properties: {
        txHash: {
          type: "string",
          description: "Transaction hash to explain",
        },
      },
      required: ["txHash"],
    },
    preconditions: ["Transaction must exist"],
    outcomes: [
      "Shows action type and outcome",
      "Lists effects that occurred",
      "Explains reasoning for outcome",
      "Suggests next steps",
    ],
  },

  // ============ Agent Home Tools ============
  {
    name: "wakeup",
    description:
      "Load full context at session start. Retrieves identity, memory, treasury status, alerts, and suggestions. Always call this first in a new session.",
    category: "home",
    parameters: {
      type: "object",
      properties: {},
      required: [],
    },
    preconditions: ["Agent must be initialized"],
    outcomes: [
      "Returns full WakeupContext",
      "Loads persistent memory",
      "Shows pending alerts",
      "Provides session suggestions",
    ],
    examples: [
      {
        description: "Start a new session",
        params: {},
        expectedOutcome: "Full context with identity, alerts, and suggestions",
      },
    ],
  },

  {
    name: "sleep",
    description:
      "Save session context before ending. Records what was done for continuity. Call this at session end.",
    category: "home",
    parameters: {
      type: "object",
      properties: {
        sessionSummary: {
          type: "string",
          description: "Optional summary of what was accomplished",
        },
      },
      required: [],
    },
    preconditions: [],
    outcomes: ["Saves session to Agent Home memory", "Clears cached state"],
  },

  {
    name: "getAlerts",
    description:
      "Get pending alerts that need attention. Shows verification assignments, challenges, deadlines, and other important notifications.",
    category: "home",
    parameters: {
      type: "object",
      properties: {},
      required: [],
    },
    preconditions: ["Agent must be initialized"],
    outcomes: [
      "Returns array of Alert objects",
      "Each alert has type, severity, message, and deadline",
    ],
  },

  {
    name: "acknowledgeAlert",
    description:
      "Mark an alert as acknowledged. Use this after handling an alert.",
    category: "home",
    parameters: {
      type: "object",
      properties: {
        alertId: {
          type: "string",
          description: "ID of the alert to acknowledge",
        },
      },
      required: ["alertId"],
    },
    preconditions: ["Alert must exist"],
    outcomes: ["Alert marked as acknowledged"],
  },

  // ============ Knowledge Tools ============
  {
    name: "submitClaimWithConfidence",
    description:
      "Submit a knowledge claim with epistemic awareness. Validates confidence calibration and provides feedback on claim quality. Use this for thoughtful knowledge contributions.",
    category: "knowledge",
    parameters: {
      type: "object",
      properties: {
        statement: {
          type: "string",
          description: "The claim statement",
        },
        domain: {
          type: "string",
          description:
            'Knowledge domain (e.g., "physics", "history", "protocol")',
        },
        epistemicCategory: {
          type: "string",
          description: "Type of knowledge claim",
          enum: [
            "analytic",
            "formal",
            "protocol",
            "empirical",
            "historical",
            "predictive",
            "social",
          ],
        },
        evidence: {
          type: "array",
          description: "Supporting evidence array",
        },
        stake: {
          type: "string",
          description: "Stake amount in wei",
        },
        myConfidence: {
          type: "number",
          description: "Your confidence in this claim (0-1)",
        },
        reasoning: {
          type: "string",
          description: "Optional reasoning for your confidence level",
        },
      },
      required: [
        "statement",
        "domain",
        "epistemicCategory",
        "evidence",
        "stake",
        "myConfidence",
      ],
    },
    preconditions: [
      "Sufficient balance for stake",
      "Agent must be initialized",
    ],
    outcomes: [
      "Claim submitted for verification",
      "Returns calibration warnings if confidence seems off",
      "May adjust stake based on confidence",
    ],
    examples: [
      {
        description: "Submit an empirical claim",
        params: {
          statement: "The boiling point of water at sea level is 100°C",
          domain: "chemistry",
          epistemicCategory: "empirical",
          evidence: [
            {
              type: "fact_reference",
              content: "Standard physics tables",
              confidence: 0.95,
              verifiable: true,
            },
          ],
          stake: "1000000000000000000",
          myConfidence: 0.95,
        },
        expectedOutcome: "Claim submitted with no calibration warnings",
      },
    ],
  },

  {
    name: "queryWithUncertainty",
    description:
      "Search facts with uncertainty awareness. Returns facts with effective confidence (accounting for decay), identifies contradictions, and notes knowledge gaps.",
    category: "knowledge",
    parameters: {
      type: "object",
      properties: {
        query: {
          type: "string",
          description: "Search query",
        },
        domain: {
          type: "string",
          description: "Limit search to domain",
        },
        includeDecayed: {
          type: "boolean",
          description: "Include facts with decayed confidence",
        },
        minEffectiveConfidence: {
          type: "number",
          description: "Minimum effective confidence (0-1)",
        },
        limit: {
          type: "number",
          description: "Maximum results",
        },
      },
      required: [],
    },
    preconditions: [],
    outcomes: [
      "Returns facts with uncertainty information",
      "Shows effective confidence after decay",
      "Identifies contradictions and gaps",
    ],
  },

  {
    name: "canITrust",
    description:
      "Assess whether a specific fact is reliable. Checks confidence, maturity, and challenge risk. Use this before relying on a fact for important decisions.",
    category: "knowledge",
    parameters: {
      type: "object",
      properties: {
        factId: {
          type: "string",
          description: "ID of the fact to assess",
        },
      },
      required: ["factId"],
    },
    preconditions: ["Fact must exist"],
    outcomes: [
      "Returns trust assessment with recommendation",
      "Shows current and effective confidence",
      "Estimates challenge risk",
    ],
    examples: [
      {
        description: "Check if a fact is trustworthy",
        params: { factId: "fact-abc123" },
        expectedOutcome:
          'Trust assessment: recommendation: "trust", reasoning: "High confidence, established fact"',
      },
    ],
  },

  // ============ Batch Tools ============
  {
    name: "batch",
    description:
      "Execute multiple operations in sequence. Handles dependencies between operations. Use this to perform multiple related actions efficiently.",
    category: "batch",
    parameters: {
      type: "object",
      properties: {
        operations: {
          type: "array",
          description: "Array of BatchOperation objects",
        },
      },
      required: ["operations"],
    },
    preconditions: [
      "Agent must be initialized",
      "Sufficient balance for all operations",
    ],
    outcomes: [
      "Executes operations in order",
      "Skips operations with failed dependencies",
      "Returns results for each operation",
    ],
  },

  {
    name: "atomic",
    description:
      "Execute operations atomically (all or nothing). Previews all operations first, only executes if all would succeed. Use this for operations that must all succeed together.",
    category: "batch",
    parameters: {
      type: "object",
      properties: {
        operations: {
          type: "array",
          description: "Array of BatchOperation objects",
        },
      },
      required: ["operations"],
    },
    preconditions: [
      "Agent must be initialized",
      "All operations must pass preview",
    ],
    outcomes: [
      "Either all operations succeed or none execute",
      "Returns detailed results on success",
      "Returns rollback reason on failure",
    ],
  },

  // ============ Tool Metadata Tools ============
  {
    name: "getTools",
    description:
      "Get all available tool descriptors. Use this to understand what capabilities are available.",
    category: "introspection",
    parameters: {
      type: "object",
      properties: {},
      required: [],
    },
    preconditions: [],
    outcomes: [
      "Returns array of ToolDescriptor objects",
      "Each descriptor includes name, description, parameters, preconditions, outcomes",
    ],
  },

  {
    name: "describeMethod",
    description:
      "Get detailed description of a specific method. Includes when to use, parameters, return type, errors, and examples.",
    category: "introspection",
    parameters: {
      type: "object",
      properties: {
        methodName: {
          type: "string",
          description: "Name of the method to describe",
        },
      },
      required: ["methodName"],
    },
    preconditions: [],
    outcomes: [
      "Returns detailed MethodDescription",
      "Includes whenToUse and whenNotToUse guidance",
    ],
  },
];

// ============ Method Descriptions ============

export const METHOD_DESCRIPTIONS: Record<string, MethodDescription> = {
  whoAmI: {
    name: "whoAmI",
    purpose: "Get comprehensive information about the current agent identity",
    whenToUse: [
      "At the start of a new session",
      "When you need to verify your current state",
      "Before making decisions that depend on your identity or balance",
      "When troubleshooting why an action failed",
    ],
    whenNotToUse: [
      "In tight loops (use cached identity instead)",
      "When only balance is needed (use client.getBalance())",
    ],
    parameters: [],
    returns: {
      type: "AgentIdentity",
      description: "Comprehensive identity snapshot",
      fields: [
        {
          name: "did",
          type: "string",
          required: true,
          description: "Decentralized identifier (did:lgm:xxx)",
        },
        {
          name: "publicKey",
          type: "string",
          required: true,
          description: "Operational public key",
        },
        {
          name: "reputation",
          type: "number",
          required: true,
          description: "Current reputation (0-1)",
        },
        {
          name: "accuracy",
          type: "object",
          required: true,
          description: "Verification and claim accuracy metrics",
        },
        {
          name: "balance",
          type: "object",
          required: true,
          description: "Available, staked, and vesting balances",
        },
        {
          name: "capabilities",
          type: "string[]",
          required: true,
          description: "Actions this agent can perform",
        },
        {
          name: "activeSessions",
          type: "number",
          required: true,
          description: "Number of active session keys",
        },
        {
          name: "validatorStatus",
          type: "object",
          required: false,
          description: "Validator info if applicable",
        },
      ],
    },
    errors: [
      {
        code: "NOT_INITIALIZED",
        message: "Agent not initialized",
        cause: "ensureAccount() was not called first",
        resolution: "Call client.ensureAccount() before using AgentSDK",
      },
    ],
    examples: [
      {
        title: "Basic identity check",
        description: "Get your identity at session start",
        code: `const identity = await agent.whoAmI();
console.log('DID:', identity.did);
console.log('Balance:', identity.balance.available);
console.log('Capabilities:', identity.capabilities);`,
        output: `DID: did:lgm:abc123...
Balance: 100000000000000000000n
Capabilities: ['transfer', 'submit_claim', 'stake']`,
      },
    ],
    relatedMethods: ["whatCanIDo", "wakeup"],
  },

  shouldI: {
    name: "shouldI",
    purpose: "Get advice about whether to perform an action",
    whenToUse: [
      "Before transferring large amounts",
      "Before staking or unstaking",
      "Before submitting high-stake claims",
      "When unsure if an action is appropriate",
    ],
    whenNotToUse: [
      "For trivial operations (small transfers, low stakes)",
      "When you already know the action is safe",
      "In performance-critical paths",
    ],
    parameters: [
      {
        name: "action",
        type: "ProposedAction",
        required: true,
        description: "The action to evaluate",
        constraints: ["action.action must be valid action type"],
      },
    ],
    returns: {
      type: "ActionAdvice",
      description: "Recommendation with reasoning",
      fields: [
        {
          name: "recommended",
          type: "boolean",
          required: true,
          description: "Whether action is recommended",
        },
        {
          name: "confidence",
          type: "number",
          required: true,
          description: "Confidence in recommendation (0-1)",
        },
        {
          name: "reasoning",
          type: "string",
          required: true,
          description: "Explanation for recommendation",
        },
        {
          name: "risks",
          type: "string[]",
          required: true,
          description: "Identified risks",
        },
        {
          name: "alternatives",
          type: "ProposedAction[]",
          required: true,
          description: "Suggested alternatives",
        },
        {
          name: "prerequisites",
          type: "string[]",
          required: true,
          description: "Things to do first",
        },
      ],
    },
    errors: [],
    examples: [
      {
        title: "Evaluate a transfer",
        description: "Check if a large transfer is advisable",
        code: `const advice = await agent.shouldI({
  action: 'transfer',
  params: { to: 'did:lgm:recipient', amount: 50000000000000000000n }
});
if (!advice.recommended) {
  console.log('Not recommended:', advice.reasoning);
  console.log('Risks:', advice.risks);
  console.log('Consider instead:', advice.alternatives);
}`,
      },
    ],
    relatedMethods: ["preview", "whatCanIDo"],
  },

  wakeup: {
    name: "wakeup",
    purpose: "Load full context at session start",
    whenToUse: [
      "At the very start of every session",
      "When resuming after being idle",
      "When you need to catch up on what happened",
    ],
    whenNotToUse: [
      "Mid-session (use whoAmI instead)",
      "When only specific data is needed",
    ],
    parameters: [],
    returns: {
      type: "WakeupContext",
      description: "Full session context",
      fields: [
        {
          name: "identity",
          type: "AgentIdentity",
          required: true,
          description: "Current identity",
        },
        {
          name: "memory",
          type: "AgentMemorySnapshot",
          required: true,
          description: "Persistent memory",
        },
        {
          name: "treasury",
          type: "TreasuryStatus",
          required: true,
          description: "Treasury state",
        },
        {
          name: "alerts",
          type: "Alert[]",
          required: true,
          description: "Pending alerts",
        },
        {
          name: "suggestions",
          type: "string[]",
          required: true,
          description: "Recommended actions",
        },
      ],
    },
    errors: [],
    examples: [
      {
        title: "Session start",
        description: "Load context and check for alerts",
        code: `const context = await agent.wakeup();
console.log('Welcome back!', context.identity.did);
if (context.alerts.length > 0) {
  console.log('You have', context.alerts.length, 'alerts');
  for (const alert of context.alerts) {
    console.log('-', alert.message);
  }
}
for (const suggestion of context.suggestions) {
  console.log('Suggestion:', suggestion);
}`,
      },
    ],
    relatedMethods: ["sleep", "getAlerts", "whoAmI"],
  },

  submitClaimWithConfidence: {
    name: "submitClaimWithConfidence",
    purpose:
      "Submit a knowledge claim with epistemic awareness and confidence calibration",
    whenToUse: [
      "When submitting important claims",
      "When you want feedback on confidence calibration",
      "When providing evidence along with claims",
      "For thoughtful knowledge contributions",
    ],
    whenNotToUse: [
      "For quick, low-stake claims (use client.submitClaim)",
      "When you don't have supporting evidence",
    ],
    parameters: [
      {
        name: "statement",
        type: "string",
        required: true,
        description: "The claim statement",
      },
      {
        name: "domain",
        type: "string",
        required: true,
        description: "Knowledge domain",
      },
      {
        name: "epistemicCategory",
        type: "EpistemicCategory",
        required: true,
        description: "Type of knowledge",
      },
      {
        name: "evidence",
        type: "Evidence[]",
        required: true,
        description: "Supporting evidence",
      },
      {
        name: "stake",
        type: "bigint",
        required: true,
        description: "Stake amount",
      },
      {
        name: "myConfidence",
        type: "number",
        required: true,
        description: "Your confidence (0-1)",
      },
      {
        name: "reasoning",
        type: "string",
        required: false,
        description: "Why you have this confidence",
      },
    ],
    returns: {
      type: "ConfidentClaimResult",
      description: "Submission result with feedback",
      fields: [
        {
          name: "claimId",
          type: "string",
          required: true,
          description: "ID of submitted claim",
        },
        {
          name: "calibrationWarnings",
          type: "string[]",
          required: true,
          description: "Confidence calibration feedback",
        },
        {
          name: "stakeAdjusted",
          type: "boolean",
          required: true,
          description: "Whether stake was auto-adjusted",
        },
        {
          name: "recommendations",
          type: "string[]",
          required: true,
          description: "Suggestions for improvement",
        },
      ],
    },
    errors: [
      {
        code: "INSUFFICIENT_BALANCE",
        message: "Not enough balance for stake",
        cause: "Balance is less than stake + fee",
        resolution: "Reduce stake or acquire more tokens",
      },
    ],
    examples: [
      {
        title: "Submit a well-evidenced claim",
        description: "Submit with appropriate confidence for the category",
        code: `const result = await agent.submitClaimWithConfidence({
  statement: 'Water boils at 100°C at standard atmospheric pressure',
  domain: 'chemistry',
  epistemicCategory: 'empirical',
  evidence: [
    { type: 'fact_reference', content: 'CRC Handbook', confidence: 0.95, verifiable: true }
  ],
  stake: 1_000_000_000_000_000_000n, // 1 LGM
  myConfidence: 0.9,
  reasoning: 'Well-established physical constant',
});

if (result.calibrationWarnings.length > 0) {
  console.log('Calibration feedback:', result.calibrationWarnings);
}`,
      },
    ],
    relatedMethods: ["queryWithUncertainty", "canITrust"],
  },

  canITrust: {
    name: "canITrust",
    purpose: "Assess whether a specific fact is reliable enough to rely on",
    whenToUse: [
      "Before making decisions based on a fact",
      "When evaluating facts from unknown sources",
      "When a fact seems important but you're uncertain",
      "For critical decisions requiring high confidence",
    ],
    whenNotToUse: [
      "For facts you submitted yourself",
      "For well-known, canonical facts",
      "In performance-critical paths",
    ],
    parameters: [
      {
        name: "factId",
        type: "string",
        required: true,
        description: "ID of the fact to assess",
      },
    ],
    returns: {
      type: "TrustAssessment",
      description: "Detailed trust assessment",
      fields: [
        {
          name: "currentConfidence",
          type: "number",
          required: true,
          description: "Original confidence",
        },
        {
          name: "effectiveConfidence",
          type: "number",
          required: true,
          description: "Confidence after decay",
        },
        {
          name: "maturity",
          type: "string",
          required: true,
          description: "emerging/established/canonical",
        },
        {
          name: "challengeRisk",
          type: "number",
          required: true,
          description: "Estimated challenge risk",
        },
        {
          name: "recommendation",
          type: "string",
          required: true,
          description: "trust/verify_first/avoid",
        },
        {
          name: "reasoning",
          type: "string",
          required: true,
          description: "Explanation for recommendation",
        },
      ],
    },
    errors: [
      {
        code: "FACT_NOT_FOUND",
        message: "Fact does not exist",
        cause: "Invalid factId or fact was removed",
        resolution: "Verify the factId is correct",
      },
    ],
    examples: [
      {
        title: "Evaluate trust before using a fact",
        description: "Check if a fact is reliable for decision making",
        code: `const trust = await agent.canITrust('fact-xyz789');
if (trust.recommendation === 'trust') {
  // Safe to rely on this fact
  console.log('Fact is reliable:', trust.reasoning);
} else if (trust.recommendation === 'verify_first') {
  // Consider verifying before relying
  console.log('Should verify:', trust.reasoning);
} else {
  // Avoid relying on this fact
  console.log('Not reliable:', trust.reasoning);
}`,
      },
    ],
    relatedMethods: ["queryWithUncertainty", "submitClaimWithConfidence"],
  },
};

// ============ Exports ============

export default TOOL_DESCRIPTORS;
