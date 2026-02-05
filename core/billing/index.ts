/**
 * API Billing Module
 *
 * Usage-based billing with automatic knowledge distribution.
 * Integrated with Truth Contribution System for unified reward tracking.
 *
 * GAP-5 FIX: Added unified gas module as single source of truth for gas costs.
 * GAP-007 FIX: Added per-message billing for agent chat sessions.
 */

export * from "./types.js";
export * from "./distribution.js";
export * from "./reward_router.js";
export * from "./gas.js";

// GAP-007: Per-Message Chat Billing
export * from "./message_billing.js";
export * from "./message_tracker.js";
export * from "./token_counter.js";

// Economic Sustainability: Payment execution
export * from "./payment_executor.js";
