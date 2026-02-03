#!/usr/bin/env node

/**
 * Legible Money CLI
 *
 * Command-line interface for interacting with Legible Money nodes.
 */

import { LegibleClient, LegibleError } from '../protocol/sdk/client.js';
import { AgentWallet } from '../protocol/sdk/wallet.js';
import { SESSION_PRESETS } from '../protocol/sdk/session.js';

// ============ Configuration ============

const DEFAULT_ENDPOINT = process.env.LGM_ENDPOINT || 'http://localhost:8545';

// ============ Commands ============

interface Command {
  name: string;
  description: string;
  usage: string;
  action: (args: string[], client: LegibleClient) => Promise<void>;
}

const commands: Record<string, Command> = {
  // Account commands
  balance: {
    name: 'balance',
    description: 'Get account balance',
    usage: 'lgm balance [DID]',
    action: async (args, client) => {
      const did = args[0] || client.getDid();
      if (!did) {
        console.error('Error: No DID specified and no wallet configured');
        return;
      }

      const balance = await client.getBalance(did);
      const staked = (await client.getAccount(did)).account?.stakedBalance || 0n;

      console.log(`Account: ${did}`);
      console.log(`Balance: ${formatLGM(balance)} LGM`);
      console.log(`Staked:  ${formatLGM(BigInt(staked))} LGM`);
      console.log(`Total:   ${formatLGM(balance + BigInt(staked))} LGM`);
    },
  },

  status: {
    name: 'status',
    description: 'Get agent status',
    usage: 'lgm status [DID]',
    action: async (args, client) => {
      const did = args[0] || client.getDid();
      if (!did) {
        console.error('Error: No DID specified and no wallet configured');
        return;
      }

      const status = await client.getAgentStatus(did);

      console.log(`\nAgent Status: ${did}\n`);
      console.log('Account:');
      console.log(`  Balance:    ${formatLGM(BigInt(status.account.balance))} LGM`);
      console.log(`  Staked:     ${formatLGM(BigInt(status.account.stakedBalance))} LGM`);
      console.log(`  Reputation: ${(status.account.reputation * 100).toFixed(1)}%`);

      console.log('\nActivity:');
      console.log(`  Claims submitted: ${status.activity.claimsSubmitted}`);
      console.log(`  Claims accepted:  ${status.activity.claimsAccepted}`);
      console.log(`  Verifications:    ${status.activity.verificationsCompleted}`);
      console.log(`  Accuracy:         ${status.activity.verificationsCompleted > 0
        ? ((status.activity.verificationsCorrect / status.activity.verificationsCompleted) * 100).toFixed(1)
        : 'N/A'}%`);

      console.log('\nSessions:');
      console.log(`  Active: ${status.sessions.active}`);
    },
  },

  transfer: {
    name: 'transfer',
    description: 'Transfer LGM tokens',
    usage: 'lgm transfer <TO_DID> <AMOUNT>',
    action: async (args, client) => {
      if (args.length < 2) {
        console.error('Usage: lgm transfer <TO_DID> <AMOUNT>');
        return;
      }

      const to = args[0];
      const amount = parseLGM(args[1]);

      console.log(`Transferring ${formatLGM(amount)} LGM to ${to}...`);

      const txHash = await client.transfer({ to, amount });

      console.log(`Transaction submitted: ${txHash}`);
      console.log('Waiting for confirmation...');

      // Poll for confirmation (simplified)
      await sleep(3000);
      console.log('Transfer complete!');
    },
  },

  // Knowledge commands
  claim: {
    name: 'claim',
    description: 'Submit a claim for verification',
    usage: 'lgm claim <DOMAIN> "<CLAIM>" [STAKE]',
    action: async (args, client) => {
      if (args.length < 2) {
        console.error('Usage: lgm claim <DOMAIN> "<CLAIM>" [STAKE]');
        return;
      }

      const domain = args[0];
      const claim = args[1];
      const stake = args[2] ? parseLGM(args[2]) : 100n * 10n ** 18n;

      console.log(`\nSubmitting claim to domain: ${domain}`);
      console.log(`Claim: "${claim}"`);
      console.log(`Stake: ${formatLGM(stake)} LGM\n`);

      const result = await client.submitClaim({ claim, domain, stake });

      console.log(`Claim submitted!`);
      console.log(`  Claim ID: ${result.claimId}`);
      console.log(`  Round ID: ${result.verificationRoundId}`);
      console.log(`  Est. completion: block ${result.estimatedCompletionBlock}`);
      console.log('\nWaiting for verification...');

      const fact = await client.waitForVerification(result.claimId, 120000);

      if (fact) {
        console.log(`\n✅ Claim ACCEPTED!`);
        console.log(`  Confidence: ${(fact.confidence * 100).toFixed(1)}%`);
        console.log(`  Verifiers:  ${fact.verifierCount}`);
      } else {
        console.log(`\n❌ Claim REJECTED`);
      }
    },
  },

  facts: {
    name: 'facts',
    description: 'List facts in a domain',
    usage: 'lgm facts [DOMAIN] [--search QUERY]',
    action: async (args, client) => {
      const domain = args[0];
      const searchIdx = args.indexOf('--search');
      const search = searchIdx !== -1 ? args[searchIdx + 1] : undefined;

      const facts = await client.searchFacts({ domain, search, limit: 20 });

      console.log(`\nFacts${domain ? ` in ${domain}` : ''}${search ? ` matching "${search}"` : ''}:\n`);

      if (facts.length === 0) {
        console.log('  No facts found.');
        return;
      }

      for (const fact of facts) {
        const confidence = (fact.confidence * 100).toFixed(0);
        console.log(`[${confidence}%] ${fact.claim}`);
        console.log(`       Domain: ${fact.domain} | ID: ${fact.id.slice(0, 16)}...`);
        console.log('');
      }

      console.log(`Total: ${facts.length} facts`);
    },
  },

  fact: {
    name: 'fact',
    description: 'Get fact details',
    usage: 'lgm fact <FACT_ID>',
    action: async (args, client) => {
      if (args.length < 1) {
        console.error('Usage: lgm fact <FACT_ID>');
        return;
      }

      const factId = args[0];
      const fact = await client.getFact(factId, true);

      if (!fact) {
        console.log(`Fact not found: ${factId}`);
        return;
      }

      console.log(`\nFact Details\n`);
      console.log(`ID:         ${fact.id}`);
      console.log(`Claim:      ${fact.claim}`);
      console.log(`Domain:     ${fact.domain}`);
      console.log(`Status:     ${fact.status}`);
      console.log(`Confidence: ${(fact.confidence * 100).toFixed(1)}%`);
      console.log(`Verifiers:  ${fact.verifierCount}`);
      console.log(`Challenges: ${fact.challengeCount}`);
      console.log(`Verified:   Block ${fact.verifiedAt}`);
    },
  },

  // Validator commands
  stake: {
    name: 'stake',
    description: 'Stake LGM tokens',
    usage: 'lgm stake <AMOUNT>',
    action: async (args, client) => {
      if (args.length < 1) {
        console.error('Usage: lgm stake <AMOUNT>');
        return;
      }

      const amount = parseLGM(args[0]);

      console.log(`Staking ${formatLGM(amount)} LGM...`);
      const txHash = await client.stake(amount);
      console.log(`Transaction: ${txHash}`);
      console.log('Stake successful!');
    },
  },

  unstake: {
    name: 'unstake',
    description: 'Unstake LGM tokens',
    usage: 'lgm unstake <AMOUNT>',
    action: async (args, client) => {
      if (args.length < 1) {
        console.error('Usage: lgm unstake <AMOUNT>');
        return;
      }

      const amount = parseLGM(args[0]);

      console.log(`Unstaking ${formatLGM(amount)} LGM...`);
      const txHash = await client.unstake(amount);
      console.log(`Transaction: ${txHash}`);
      console.log('Unstake initiated (subject to unbonding period).');
    },
  },

  // Network commands
  health: {
    name: 'health',
    description: 'Check network health',
    usage: 'lgm health',
    action: async (_, client) => {
      const health = await client.getNetworkHealth();

      console.log(`\nNetwork Health\n`);
      console.log(`Status:       ${health.status}`);
      console.log(`Latest Block: ${health.latestBlock}`);
      console.log(`Sync Status:  ${health.syncStatus}`);
      console.log(`Peers:        ${health.peerCount}`);
      console.log(`Version:      ${health.version}`);
    },
  },

  block: {
    name: 'block',
    description: 'Get block info',
    usage: 'lgm block [NUMBER|HASH|latest]',
    action: async (args, client) => {
      const query = args[0] || 'latest';

      let block;
      if (query === 'latest') {
        block = await client.getBlock(await client.getBlockNumber());
      } else if (query.startsWith('0x') || query.length === 64) {
        block = await client.getBlock(query);
      } else {
        block = await client.getBlock(BigInt(query));
      }

      if (!block) {
        console.log('Block not found');
        return;
      }

      console.log(`\nBlock ${block.header.number}\n`);
      console.log(`Timestamp:    ${new Date(block.header.timestamp).toISOString()}`);
      console.log(`Proposer:     ${block.header.proposer.slice(0, 32)}...`);
      console.log(`Transactions: ${block.transactions.length}`);
      console.log(`Verifications: ${block.verificationResults.length}`);
      console.log(`State Root:   ${block.header.stateRoot.slice(0, 32)}...`);
      console.log(`Knowledge Root: ${block.header.knowledgeRoot.slice(0, 32)}...`);
    },
  },

  // Utility commands
  keygen: {
    name: 'keygen',
    description: 'Generate new wallet keys',
    usage: 'lgm keygen <AGENT_ID> <SECRET>',
    action: async (args) => {
      if (args.length < 2) {
        console.error('Usage: lgm keygen <AGENT_ID> <SECRET>');
        return;
      }

      const agentId = args[0];
      const secret = args[1];

      const wallet = AgentWallet.fromAgentId(agentId, secret, 'testnet');

      console.log(`\nWallet Generated\n`);
      console.log(`Agent ID:   ${agentId}`);
      console.log(`DID:        ${wallet.did}`);
      console.log(`Public Key: ${wallet.publicKey}`);
      console.log(`\n⚠️  Keep your secret safe! You can recreate this wallet anytime with:`);
      console.log(`    lgm keygen ${agentId} <your-secret>`);
    },
  },

  faucet: {
    name: 'faucet',
    description: 'Request testnet tokens',
    usage: 'lgm faucet [AMOUNT]',
    action: async (args, client) => {
      const amount = args[0] ? parseLGM(args[0]) : 1000n * 10n ** 18n;

      console.log(`Requesting ${formatLGM(amount)} LGM from faucet...`);

      try {
        const txHash = await client.requestFaucet(amount);
        console.log(`Transaction: ${txHash}`);
        console.log('Tokens received!');
      } catch (error) {
        if (error instanceof LegibleError) {
          console.error(`Faucet error: ${error.message}`);
        } else {
          throw error;
        }
      }
    },
  },
};

// ============ Help ============

function printHelp(): void {
  console.log(`
Legible Money CLI

Usage: lgm <command> [options]

Commands:
  Account:
    balance [DID]              Get account balance
    status [DID]               Get agent status
    transfer <TO> <AMOUNT>     Transfer LGM tokens

  Knowledge:
    claim <DOMAIN> "<CLAIM>"   Submit a claim for verification
    facts [DOMAIN]             List facts in a domain
    fact <FACT_ID>             Get fact details

  Staking:
    stake <AMOUNT>             Stake LGM tokens
    unstake <AMOUNT>           Unstake LGM tokens

  Network:
    health                     Check network health
    block [NUMBER|HASH]        Get block info

  Utility:
    keygen <ID> <SECRET>       Generate wallet keys
    faucet [AMOUNT]            Request testnet tokens

Environment:
  LGM_ENDPOINT     RPC endpoint (default: http://localhost:8545)
  LGM_AGENT_ID     Agent ID for wallet
  LGM_SECRET       Secret for wallet derivation

Examples:
  lgm balance
  lgm transfer did:lgm:alice 100
  lgm claim science.physics "Water boils at 100°C"
  lgm facts science.physics
  lgm health
`);
}

// ============ Helpers ============

function formatLGM(wei: bigint): string {
  const lgm = Number(wei) / 1e18;
  return lgm.toLocaleString('en-US', { maximumFractionDigits: 4 });
}

function parseLGM(input: string): bigint {
  const num = parseFloat(input);
  return BigInt(Math.floor(num * 1e18));
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// ============ Main ============

async function main(): Promise<void> {
  const args = process.argv.slice(2);

  if (args.length === 0 || args[0] === 'help' || args[0] === '--help' || args[0] === '-h') {
    printHelp();
    return;
  }

  const commandName = args[0];
  const commandArgs = args.slice(1);

  const command = commands[commandName];
  if (!command) {
    console.error(`Unknown command: ${commandName}`);
    console.log('Run "lgm help" for available commands.');
    process.exit(1);
  }

  // Initialize client
  const client = new LegibleClient({
    endpoint: DEFAULT_ENDPOINT,
    agentId: process.env.LGM_AGENT_ID,
    secret: process.env.LGM_SECRET,
  });

  try {
    await command.action(commandArgs, client);
  } catch (error) {
    if (error instanceof LegibleError) {
      console.error(`\nError: ${error.message}`);
      if (error.retryable) {
        console.log(`This error is retryable. Try again in ${error.retryAfter || 1000}ms.`);
      }
    } else {
      console.error('\nUnexpected error:', error);
    }
    process.exit(1);
  }
}

main();
