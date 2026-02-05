/**
 * Infrastructure Layer
 *
 * ARCH-1: Layer Isolation
 * Infrastructure implements contracts defined in foundation.
 * Domain modules depend on foundation contracts, not infrastructure.
 *
 * This layer provides:
 * - Production implementations of foundation contracts
 * - External system integrations
 * - Persistence mechanisms
 * - Network communication
 */

// Event System
export * from "./events/index.js";
