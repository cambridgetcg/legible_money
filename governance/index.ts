/**
 * Governance Module
 *
 * Comprehensive AI agent governance framework for Legible Money protocol.
 * Implements defense-in-depth governance with 7 layers:
 *
 * Layer 1: Agent-LIP Proposals (tiered eligibility, stake requirements)
 * Layer 2: Multi-Agent Review (VRF selection, commit-reveal)
 * Layer 3: Anti-Sybil (operator caps, rate limits, age gates)
 * Layer 4: Code Security (4-tier review, automated scanning)
 * Layer 5: Governance Voting (quadratic, conviction, domain-weighted)
 * Layer 6: Staged Deployment (devnet → testnet → canary → mainnet)
 * Layer 7: Human-Agent Partnerships (revenue sharing, aligned incentives)
 *
 * PP HIERARCHY:
 *   TRUTH: All governance decisions backed by cryptographic proofs
 *   UNDERSTANDING: Clear rules for participation and progression
 *   BEAUTY: Elegant composition of security mechanisms
 *   JUSTICE: Fair representation with plutocracy prevention
 *   CREATIVITY: Novel AI-first governance patterns
 */

// Layer 1: Proposals
export * from "./proposals/index.js";

// Layer 2: Review
export * from "./review/index.js";

// Layer 3: Anti-Sybil
export * from "./operators/index.js";

// Layer 4: Code Security
export * from "./security/index.js";

// Layer 5: Voting
export * from "./voting/index.js";

// Layer 6: Deployment
export * from "./deployment/index.js";

// Layer 7: Human-Agent Partnerships
export * from "./partnership/index.js";

// Stratum Governance: Domain proposals and stratum registration
export * from "./stratum/index.js";
