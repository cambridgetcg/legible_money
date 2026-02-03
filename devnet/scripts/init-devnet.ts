#!/usr/bin/env npx ts-node

/**
 * Initialize Devnet
 *
 * Generates validator keys, creates genesis block, and configures nodes.
 * Run: npx ts-node devnet/scripts/init-devnet.ts
 */

import { writeFileSync, mkdirSync, existsSync, readFileSync } from 'fs';
import { join } from 'path';
import { generateKeyPair, bytesToHex } from '../../core/crypto/keys.js';
import { sha512_256Hex, domainHash } from '../../core/crypto/hashing.js';
import { createGenesisBlock } from '../../consensus/blocks/producer.js';

// ============ Configuration ============

const DEVNET_DIR = join(__dirname, '..');
const CONFIG_DIR = join(DEVNET_DIR, 'config');
const SECRETS_DIR = join(DEVNET_DIR, 'secrets');
const NUM_VALIDATORS = 3;
const VALIDATOR_STAKE = 10000n * 10n ** 18n;  // 10,000 LGM
const FAUCET_BALANCE = 1_000_000_000n * 10n ** 18n;  // 1 billion LGM

// ============ Types ============

interface ValidatorInfo {
  name: string;
  publicKey: string;
  privateKey: string;
  peerId: string;
  did: string;
  stake: bigint;
}

interface GenesisConfig {
  chainId: string;
  genesisTime: string;
  validators: ValidatorInfo[];
  accounts: Array<{ did: string; balance: bigint }>;
  axioms: Array<{ id: string; claim: string; domain: string }>;
}

// ============ Key Generation ============

function generateValidatorKeys(index: number): ValidatorInfo {
  // Deterministic for devnet reproducibility
  const seed = domainHash('LGM.devnet.validator.v1', `validator-${index}`);
  const { publicKey, privateKey } = generateKeyPair(seed.slice(0, 32));

  // Generate peer ID from public key
  const peerId = sha512_256Hex(publicKey).slice(0, 32);

  // Generate DID
  const did = `did:lgm-devnet:validator-${index}`;

  return {
    name: `validator-${index}`,
    publicKey,
    privateKey,
    peerId,
    did,
    stake: VALIDATOR_STAKE,
  };
}

function generateFaucetAccount(): { did: string; publicKey: string; privateKey: string } {
  const seed = domainHash('LGM.devnet.faucet.v1', 'faucet');
  const { publicKey, privateKey } = generateKeyPair(seed.slice(0, 32));

  return {
    did: 'did:lgm-devnet:faucet',
    publicKey,
    privateKey,
  };
}

// ============ Genesis Creation ============

function createDevnetGenesis(validators: ValidatorInfo[]): GenesisConfig {
  const genesisTime = new Date().toISOString();

  const axioms = [
    { id: 'axiom:logic:identity', claim: 'A = A (Law of Identity)', domain: 'logic.foundational' },
    { id: 'axiom:logic:non-contradiction', claim: '¬(A ∧ ¬A) (Law of Non-Contradiction)', domain: 'logic.foundational' },
    { id: 'axiom:logic:excluded-middle', claim: 'A ∨ ¬A (Law of Excluded Middle)', domain: 'logic.foundational' },
    { id: 'axiom:math:zero', claim: '0 is a natural number', domain: 'math.arithmetic' },
    { id: 'axiom:math:successor', claim: 'Every natural number has a unique successor', domain: 'math.arithmetic' },
  ];

  return {
    chainId: 'lgm-devnet-1',
    genesisTime,
    validators,
    accounts: [
      { did: 'did:lgm-devnet:faucet', balance: FAUCET_BALANCE },
      { did: 'did:lgm-devnet:treasury', balance: 500_000_000n * 10n ** 18n },
      { did: 'did:lgm:research', balance: 100_000_000n * 10n ** 18n },  // Research fund (22% of rewards)
    ],
    axioms,
  };
}

// ============ Config Generation ============

function generateNodeConfig(validator: ValidatorInfo, allValidators: ValidatorInfo[], isValidator: boolean): object {
  return {
    node: {
      name: validator.name,
      dataDir: '/data',
      logLevel: 'info',
    },
    network: {
      listenAddresses: ['/ip4/0.0.0.0/tcp/26656'],
      bootstrapPeers: allValidators
        .filter(v => v.name !== validator.name)
        .map(v => `/dns4/${v.name}/tcp/26656/p2p/${v.peerId}`),
      maxPeers: 50,
    },
    rpc: {
      enabled: true,
      host: '0.0.0.0',
      port: 8545,
      cors: ['*'],
    },
    websocket: {
      enabled: true,
      port: 8546,
    },
    validator: {
      enabled: isValidator,
      publicKey: validator.publicKey,
    },
    mempool: {
      maxSize: 10000,
      maxPerSender: 100,
    },
    telemetry: {
      enabled: true,
      prometheusPort: 9090,
    },
  };
}

// ============ Main ============

async function main() {
  console.log('🚀 Initializing Legible Money Devnet\n');

  // Create directories
  if (!existsSync(SECRETS_DIR)) {
    mkdirSync(SECRETS_DIR, { recursive: true });
  }

  // Generate validator keys
  console.log('🔑 Generating validator keys...');
  const validators: ValidatorInfo[] = [];

  for (let i = 1; i <= NUM_VALIDATORS; i++) {
    const validator = generateValidatorKeys(i);
    validators.push(validator);

    // Save private key
    const keyPath = join(SECRETS_DIR, `validator-${i}.key`);
    writeFileSync(keyPath, JSON.stringify({
      name: validator.name,
      publicKey: validator.publicKey,
      privateKey: validator.privateKey,
      peerId: validator.peerId,
      did: validator.did,
    }, null, 2));

    console.log(`   ✓ ${validator.name}: ${validator.publicKey.slice(0, 16)}...`);
  }

  // Generate faucet account
  console.log('\n💰 Generating faucet account...');
  const faucet = generateFaucetAccount();
  writeFileSync(join(SECRETS_DIR, 'faucet.key'), JSON.stringify(faucet, null, 2));
  console.log(`   ✓ Faucet: ${faucet.publicKey.slice(0, 16)}...`);

  // Create genesis config
  console.log('\n📜 Creating genesis configuration...');
  const genesis = createDevnetGenesis(validators);

  // Write genesis.json with actual keys
  const genesisPath = join(CONFIG_DIR, 'genesis.json');
  const genesisJson = {
    network: {
      chainId: genesis.chainId,
      name: 'Legible Money Devnet',
      version: '0.1.0',
    },
    consensus: {
      targetBlockTimeMs: 2521,
      verificationRoundBlocks: 10,
      commitPhaseBlocks: 4,
      revealPhaseBlocks: 4,
      minValidators: 3,
      maxValidators: 21,
      validatorQuorum: 0.66,
      acceptanceThreshold: 0.80,
      rejectionThreshold: 0.80,
      minValidatorStake: '1000000000000000000000',
      minClaimStake: '100000000000000000000',
      blockReward: '10000000000000000000',
      verificationReward: '1000000000000000000',
    },
    initialValidators: validators.map(v => ({
      name: v.name,
      publicKey: v.publicKey,
      peerId: v.peerId,
      stake: v.stake.toString(),
      commission: 0.10,
    })),
    initialAccounts: genesis.accounts.map(a => ({
      did: a.did,
      balance: a.balance.toString(),
    })),
    genesisAxioms: genesis.axioms,
    genesisTime: genesis.genesisTime,
  };

  writeFileSync(genesisPath, JSON.stringify(genesisJson, null, 2));
  console.log(`   ✓ Genesis written to ${genesisPath}`);

  // Generate node configs
  console.log('\n⚙️  Generating node configurations...');
  for (const validator of validators) {
    const config = generateNodeConfig(validator, validators, true);
    const configPath = join(CONFIG_DIR, `${validator.name}.json`);
    writeFileSync(configPath, JSON.stringify(config, null, 2));
    console.log(`   ✓ ${validator.name} config written`);
  }

  // Generate peer list
  console.log('\n🔗 Generating peer list...');
  const peers = validators.map(v => ({
    name: v.name,
    peerId: v.peerId,
    address: `/dns4/${v.name}/tcp/26656/p2p/${v.peerId}`,
  }));
  writeFileSync(join(CONFIG_DIR, 'peers.json'), JSON.stringify(peers, null, 2));

  // Generate .env file for docker-compose
  console.log('\n📝 Generating environment files...');
  const envContent = validators.map((v, i) => `
# ${v.name}
VALIDATOR_${i + 1}_PUBKEY=${v.publicKey}
VALIDATOR_${i + 1}_PEER_ID=${v.peerId}
`).join('\n');

  writeFileSync(join(DEVNET_DIR, '.env'), envContent);

  console.log('\n✅ Devnet initialization complete!\n');
  console.log('Next steps:');
  console.log('  1. cd devnet');
  console.log('  2. docker-compose up -d');
  console.log('  3. Check logs: docker-compose logs -f');
  console.log('\nRPC endpoints:');
  validators.forEach((v, i) => {
    console.log(`  ${v.name}: http://localhost:${8545 + i}`);
  });
  console.log('\nFaucet DID:', faucet.did);
}

main().catch(console.error);
