/**
 * Agent Identity Module
 *
 * Identity management tailored for AI agents.
 * Handles the unique challenges of agent key management:
 * - Ephemeral memory (sessions end, context resets)
 * - Developer exposure (operators may see agent state)
 * - Identity continuity (reputation persists across instances)
 */

export * from './did.js';
export * from './account.js';
export * from './session_keys.js';
export * from './recovery.js';
export * from './mpc.js';
