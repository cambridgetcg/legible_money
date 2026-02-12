/**
 * Proof of Truth Consensus
 *
 * The work is verifying truth. The product is verified knowledge.
 *
 * @module consensus/pot
 */

// ============ Types ============
export * from "./types.js";

// ============ Core Engine ============
export * from "./engine.js";
export * from "./validator_selection.js";
export * from "./verification_round.js";
export * from "./confidence.js";
export * from "./finality.js";

// ============ State Integration ============
export * from "./state_integration.js";
export * from "./checkpoint_ceremony.js";

// ============ Domain Capture Defense ============
// Four interconnected defense systems:
// 1. Domain Qualification - Validators must qualify per domain
// 2. Cross-Stratum Validation - Higher strata need lower stratum validators
// 3. Challenge Bounties - Economic incentive to expose capture
// 4. Reputation Isolation - Domain-specific reputation (no transfer)
export * from "./domain_capture_integration.js";

// ============ Challenge/Dispute Integration ============
export * from "./challenge_integration.js";

// ============ Adversarial Verification (Survive-to-Accept) ============
export * from "./adversarial.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
