/**
 * State Layer
 *
 * Persistent state management using Sparse Merkle Trees.
 * Tracks accounts, validators, and knowledge (verified facts).
 *
 * @module state
 */

// ============ Types ============
export * from "./types.js";

// ============ State Trees ============
export * from "./accounts.js";
export * from "./validators.js";
export * from "./knowledge.js";

// ============ State Management ============
export * from "./state_manager.js";
export * from "./transitions.js";

// ============ Governance State ============
export * from "./governance.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
