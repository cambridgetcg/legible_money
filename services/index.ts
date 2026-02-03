/**
 * Legible Money Services
 *
 * Auxiliary services for the Legible Money network.
 */

// Services are standalone executables, not library exports
// Run directly: node dist/services/faucet.js
// Run directly: node dist/services/explorer.js

export const SERVICES = {
  faucet: 'dist/services/faucet.js',
  explorer: 'dist/services/explorer.js',
} as const;
