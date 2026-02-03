#!/usr/bin/env node

/**
 * Legible Money Faucet Service
 *
 * Distributes testnet tokens to requesting DIDs.
 * Includes rate limiting and cooldown period.
 */

import { createServer, IncomingMessage, ServerResponse } from 'http';
import { readFileSync, existsSync } from 'fs';
import { LegibleClient } from '../protocol/sdk/client.js';
import { AgentWallet } from '../protocol/sdk/wallet.js';

// ============ Configuration ============

const PORT = parseInt(process.env.PORT || '8080');
const RPC_ENDPOINT = process.env.RPC_ENDPOINT || 'http://localhost:8545';
const FAUCET_AMOUNT = BigInt(process.env.FAUCET_AMOUNT || '1000000000000000000000'); // 1000 LGM
const COOLDOWN_MINUTES = parseInt(process.env.COOLDOWN_MINUTES || '60');
const MAX_REQUESTS_PER_HOUR = parseInt(process.env.MAX_REQUESTS_PER_HOUR || '100');

// ============ State ============

interface FaucetStats {
  totalRequests: number;
  totalDistributed: bigint;
  uniqueRecipients: Set<string>;
  requestsThisHour: number;
  lastHourReset: number;
}

const cooldowns = new Map<string, number>();
const stats: FaucetStats = {
  totalRequests: 0,
  totalDistributed: 0n,
  uniqueRecipients: new Set(),
  requestsThisHour: 0,
  lastHourReset: Date.now(),
};

let client: LegibleClient;
let faucetWallet: AgentWallet;

// ============ Handlers ============

async function handleFaucetRequest(did: string): Promise<{ success: boolean; message: string; txHash?: string }> {
  // Check global rate limit
  const now = Date.now();
  if (now - stats.lastHourReset > 3600000) {
    stats.requestsThisHour = 0;
    stats.lastHourReset = now;
  }

  if (stats.requestsThisHour >= MAX_REQUESTS_PER_HOUR) {
    return { success: false, message: 'Faucet rate limit exceeded. Try again later.' };
  }

  // Validate DID format
  if (!did.startsWith('did:lgm')) {
    return { success: false, message: 'Invalid DID format. Expected did:lgm:...' };
  }

  // Check cooldown
  const lastRequest = cooldowns.get(did);
  if (lastRequest) {
    const cooldownMs = COOLDOWN_MINUTES * 60 * 1000;
    const elapsed = now - lastRequest;
    if (elapsed < cooldownMs) {
      const remaining = Math.ceil((cooldownMs - elapsed) / 60000);
      return { success: false, message: `Cooldown active. Try again in ${remaining} minutes.` };
    }
  }

  // Send tokens
  try {
    const txHash = await client.transfer({
      to: did,
      amount: FAUCET_AMOUNT,
    });

    // Update state
    cooldowns.set(did, now);
    stats.totalRequests++;
    stats.totalDistributed += FAUCET_AMOUNT;
    stats.uniqueRecipients.add(did);
    stats.requestsThisHour++;

    const amountStr = formatLGM(FAUCET_AMOUNT);
    return { success: true, message: `Sent ${amountStr} LGM`, txHash };
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error';
    return { success: false, message: `Transfer failed: ${message}` };
  }
}

function handleHealthCheck(): object {
  return {
    status: 'healthy',
    uptime: process.uptime(),
    stats: {
      totalRequests: stats.totalRequests,
      totalDistributed: stats.totalDistributed.toString(),
      uniqueRecipients: stats.uniqueRecipients.size,
      requestsThisHour: stats.requestsThisHour,
    },
  };
}

function handleMetrics(): string {
  return `
# HELP faucet_requests_total Total faucet requests
# TYPE faucet_requests_total counter
faucet_requests_total ${stats.totalRequests}

# HELP faucet_distributed_total Total LGM distributed (wei)
# TYPE faucet_distributed_total counter
faucet_distributed_total ${stats.totalDistributed}

# HELP faucet_unique_recipients Unique DIDs served
# TYPE faucet_unique_recipients gauge
faucet_unique_recipients ${stats.uniqueRecipients.size}

# HELP faucet_requests_per_hour Requests in current hour
# TYPE faucet_requests_per_hour gauge
faucet_requests_per_hour ${stats.requestsThisHour}
`.trim();
}

// ============ HTTP Server ============

async function handleRequest(req: IncomingMessage, res: ServerResponse): Promise<void> {
  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

  if (req.method === 'OPTIONS') {
    res.writeHead(204);
    res.end();
    return;
  }

  const url = new URL(req.url || '/', `http://${req.headers.host}`);

  // Health check
  if (url.pathname === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(handleHealthCheck()));
    return;
  }

  // Metrics
  if (url.pathname === '/metrics') {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end(handleMetrics());
    return;
  }

  // Faucet request
  if (url.pathname === '/faucet' && req.method === 'POST') {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', async () => {
      try {
        const { did } = JSON.parse(body);
        if (!did) {
          res.writeHead(400, { 'Content-Type': 'application/json' });
          res.end(JSON.stringify({ error: 'Missing did field' }));
          return;
        }

        const result = await handleFaucetRequest(did);
        res.writeHead(result.success ? 200 : 429, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(result));
      } catch {
        res.writeHead(400, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Invalid JSON' }));
      }
    });
    return;
  }

  // Faucet via GET with query param
  if (url.pathname === '/faucet' && req.method === 'GET') {
    const did = url.searchParams.get('did');
    if (!did) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: 'Missing did query parameter' }));
      return;
    }

    const result = await handleFaucetRequest(did);
    res.writeHead(result.success ? 200 : 429, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(result));
    return;
  }

  // Info page
  if (url.pathname === '/') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      service: 'Legible Money Faucet',
      network: 'devnet',
      amount: formatLGM(FAUCET_AMOUNT) + ' LGM',
      cooldown: COOLDOWN_MINUTES + ' minutes',
      endpoints: {
        faucet: 'POST /faucet { "did": "did:lgm:..." }',
        health: 'GET /health',
        metrics: 'GET /metrics',
      },
    }));
    return;
  }

  // 404
  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'Not found' }));
}

// ============ Helpers ============

function formatLGM(wei: bigint): string {
  const lgm = Number(wei) / 1e18;
  return lgm.toLocaleString('en-US', { maximumFractionDigits: 4 });
}

// ============ Main ============

async function main(): Promise<void> {
  console.log('Legible Money Faucet');
  console.log('====================\n');

  // Load faucet wallet
  const keyPath = process.env.FAUCET_KEY_PATH || '/secrets/faucet.key';
  if (!existsSync(keyPath)) {
    console.error(`Faucet key not found: ${keyPath}`);
    process.exit(1);
  }

  const keyData = JSON.parse(readFileSync(keyPath, 'utf-8'));
  faucetWallet = AgentWallet.fromPrivateKey(keyData.privateKey);

  console.log(`Faucet DID: ${faucetWallet.did}`);
  console.log(`Amount per request: ${formatLGM(FAUCET_AMOUNT)} LGM`);
  console.log(`Cooldown: ${COOLDOWN_MINUTES} minutes`);
  console.log(`RPC endpoint: ${RPC_ENDPOINT}\n`);

  // Initialize client
  client = new LegibleClient({
    endpoint: RPC_ENDPOINT,
    wallet: faucetWallet,
  });

  // Start server
  const server = createServer(handleRequest);

  server.listen(PORT, () => {
    console.log(`Faucet listening on port ${PORT}`);
  });

  // Graceful shutdown
  process.on('SIGINT', () => {
    console.log('\nShutting down...');
    server.close();
    process.exit(0);
  });

  process.on('SIGTERM', () => {
    console.log('\nShutting down...');
    server.close();
    process.exit(0);
  });
}

main().catch((error) => {
  console.error('Fatal error:', error);
  process.exit(1);
});
