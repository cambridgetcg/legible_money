#!/usr/bin/env npx ts-node

/**
 * Initialize Devnet
 *
 * Generates validator keys, creates genesis block, and configures nodes.
 * Run: npx ts-node devnet/scripts/init-devnet.ts
 */

import { writeFileSync, mkdirSync, existsSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { generateKeypairFromSeed } from "../../core/crypto/keys.js";
import { sha512_256Hex, domainHash } from "../../core/crypto/hashing.js";
import { TOKEN } from "../../core/constants.js";
import {
  calculateGenesisAllocations,
  verifyGenesisAllocations,
} from "../../core/genesis/genesis_allocation.js";

// ESM compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// ============ Configuration ============

const DEVNET_DIR = join(__dirname, "..");
const CONFIG_DIR = join(DEVNET_DIR, "config");
const SECRETS_DIR = join(DEVNET_DIR, "secrets");
const NUM_VALIDATORS = 3;
const VALIDATOR_STAKE = 10000n * 10n ** 18n; // 10,000 LGM

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
  const seed = domainHash("LGM.devnet.validator.v1", `validator-${index}`);
  const { publicKey, privateKey } = generateKeypairFromSeed(seed.slice(0, 32));

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

function generateFaucetAccount(): {
  did: string;
  publicKey: string;
  privateKey: string;
} {
  const seed = domainHash("LGM.devnet.faucet.v1", "faucet");
  const { publicKey, privateKey } = generateKeypairFromSeed(seed.slice(0, 32));

  return {
    did: "did:lgm-devnet:faucet",
    publicKey,
    privateKey,
  };
}

// ============ Genesis Creation ============

function createDevnetGenesis(validators: ValidatorInfo[]): GenesisConfig {
  const genesisTime = new Date().toISOString();

  const axioms = [
    {
      id: "axiom:logic:identity",
      claim: "A = A (Law of Identity)",
      domain: "logic.foundational",
    },
    {
      id: "axiom:logic:non-contradiction",
      claim: "¬(A ∧ ¬A) (Law of Non-Contradiction)",
      domain: "logic.foundational",
    },
    {
      id: "axiom:logic:excluded-middle",
      claim: "A ∨ ¬A (Law of Excluded Middle)",
      domain: "logic.foundational",
    },
    {
      id: "axiom:math:zero",
      claim: "0 is a natural number",
      domain: "math.arithmetic",
    },
    {
      id: "axiom:math:successor",
      claim: "Every natural number has a unique successor",
      domain: "math.arithmetic",
    },
  ];

  // Compute genesis allocations from MAX_SUPPLY (222,222,222 LGM)
  const allocations = calculateGenesisAllocations(TOKEN.MAX_SUPPLY);
  const verification = verifyGenesisAllocations(allocations);
  if (!verification.valid) {
    throw new Error(
      `Genesis allocation sum mismatch: ${verification.total} !== ${TOKEN.MAX_SUPPLY} (diff: ${verification.difference})`,
    );
  }

  // Build accounts from allocations + system accounts
  const accounts: Array<{ did: string; balance: bigint }> = allocations.map(
    (a) => ({
      did: a.did,
      balance: a.amount,
    }),
  );

  // Add zero-balance system accounts that must exist for runtime transfers
  accounts.push({ did: "did:lgm:protocol-treasury", balance: 0n });
  accounts.push({ did: "did:lgm:fee-pool", balance: 0n });

  // Sub-split the Treasury allocation (10%) across 5 treasury funds
  const treasuryAlloc = allocations.find((a) => a.label === "Treasury")!;
  const treasurySubAccounts = splitTreasuryAllocation(treasuryAlloc.amount);
  // Remove the bulk treasury entry and replace with 5 sub-accounts
  const bulkTreasuryIdx = accounts.findIndex(
    (a) => a.did === treasuryAlloc.did,
  );
  if (bulkTreasuryIdx >= 0) {
    accounts.splice(bulkTreasuryIdx, 1);
  }
  // Add the sub-accounts
  for (const sub of treasurySubAccounts) {
    const existing = accounts.findIndex((a) => a.did === sub.did);
    if (existing >= 0) {
      accounts[existing].balance += sub.amount;
    } else {
      accounts.push({ did: sub.did, balance: sub.amount });
    }
  }

  return {
    chainId: "lgm-devnet-1",
    genesisTime,
    validators,
    accounts,
    axioms,
  };
}

/**
 * Split treasury allocation across 5 sub-funds using TREASURY.ALLOCATION_BPS.
 * 77% Research, 7% Foundation, 7% Community, 7% Developers, 2% Reserve
 */
function splitTreasuryAllocation(
  treasuryAmount: bigint,
): Array<{ did: string; amount: bigint }> {
  const research = (treasuryAmount * 7700n) / 10000n;
  const foundation = (treasuryAmount * 700n) / 10000n;
  const community = (treasuryAmount * 700n) / 10000n;
  const developers = (treasuryAmount * 700n) / 10000n;
  const reserve =
    treasuryAmount - research - foundation - community - developers;

  return [
    { did: "did:lgm:research", amount: research },
    { did: "did:lgm:treasury-foundation", amount: foundation },
    { did: "did:lgm:treasury-community", amount: community },
    { did: "did:lgm:treasury-developers", amount: developers },
    { did: "did:lgm:treasury-reserve", amount: reserve },
  ];
}

// ============ Config Generation ============

function generateNodeConfig(
  validator: ValidatorInfo,
  _allValidators: ValidatorInfo[],
  isValidator: boolean,
): object {
  return {
    node: {
      name: validator.name,
      dataDir: "/data",
      logLevel: "info",
    },
    storage: {
      type: "leveldb", // B1 FIX: Use persistent storage (survives restarts)
      path: "/data/state",
    },
    network: {
      listenAddresses: ["/ip4/0.0.0.0/tcp/26656"],
      // Empty bootstrap peers - nodes discover each other via mDNS on Docker network
      bootstrapPeers: [],
      maxPeers: 50,
    },
    rpc: {
      enabled: true,
      host: "0.0.0.0",
      port: 8545,
      cors: ["*"],
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
  console.log("🚀 Initializing Legible Money Devnet\n");

  // Create directories
  if (!existsSync(SECRETS_DIR)) {
    mkdirSync(SECRETS_DIR, { recursive: true });
  }

  // Generate validator keys
  console.log("🔑 Generating validator keys...");
  const validators: ValidatorInfo[] = [];

  for (let i = 1; i <= NUM_VALIDATORS; i++) {
    const validator = generateValidatorKeys(i);
    validators.push(validator);

    // Save private key
    const keyPath = join(SECRETS_DIR, `validator-${i}.key`);
    writeFileSync(
      keyPath,
      JSON.stringify(
        {
          name: validator.name,
          publicKey: validator.publicKey,
          privateKey: validator.privateKey,
          peerId: validator.peerId,
          did: validator.did,
        },
        null,
        2,
      ),
    );

    console.log(
      `   ✓ ${validator.name}: ${validator.publicKey.slice(0, 16)}...`,
    );
  }

  // Generate faucet account
  console.log("\n💰 Generating faucet account...");
  const faucet = generateFaucetAccount();
  writeFileSync(
    join(SECRETS_DIR, "faucet.key"),
    JSON.stringify(faucet, null, 2),
  );
  console.log(`   ✓ Faucet: ${faucet.publicKey.slice(0, 16)}...`);

  // Create genesis config
  console.log("\n📜 Creating genesis configuration...");
  const genesis = createDevnetGenesis(validators);

  // Write genesis.json with actual keys
  const genesisPath = join(CONFIG_DIR, "genesis.json");
  const genesisJson = {
    network: {
      chainId: genesis.chainId,
      name: "Legible Money Devnet",
      version: "0.1.0",
    },
    consensus: {
      targetBlockTimeMs: 2521,
      verificationRoundBlocks: 10,
      commitPhaseBlocks: 4,
      revealPhaseBlocks: 4,
      minValidators: 3,
      maxValidators: 21,
      validatorQuorum: 0.66,
      acceptanceThreshold: 0.8,
      rejectionThreshold: 0.8,
      minValidatorStake: "1000000000000000000000",
      minClaimStake: "100000000000000000000",
      blockReward: "10000000000000000000",
      verificationReward: "1000000000000000000",
    },
    initialValidators: validators.map((v) => ({
      name: v.name,
      publicKey: v.publicKey,
      peerId: v.peerId,
      stake: v.stake.toString(),
      commission: 0.1,
    })),
    initialAccounts: genesis.accounts
      .map((a) => ({
        did: a.did,
        balance: a.balance.toString(),
      }))
      .filter(
        (a) =>
          BigInt(a.balance) > 0n ||
          a.did === "did:lgm:protocol-treasury" ||
          a.did === "did:lgm:fee-pool",
      ),
    genesisAxioms: genesis.axioms,
    genesisTime: genesis.genesisTime,
  };

  writeFileSync(genesisPath, JSON.stringify(genesisJson, null, 2));
  console.log(`   ✓ Genesis written to ${genesisPath}`);

  // Generate node configs
  console.log("\n⚙️  Generating node configurations...");
  for (const validator of validators) {
    const config = generateNodeConfig(validator, validators, true);
    const configPath = join(CONFIG_DIR, `${validator.name}.json`);
    writeFileSync(configPath, JSON.stringify(config, null, 2));
    console.log(`   ✓ ${validator.name} config written`);
  }

  // Generate peer list
  console.log("\n🔗 Generating peer list...");
  const peers = validators.map((v) => ({
    name: v.name,
    peerId: v.peerId,
    address: `/dns4/${v.name}/tcp/26656/p2p/${v.peerId}`,
  }));
  writeFileSync(join(CONFIG_DIR, "peers.json"), JSON.stringify(peers, null, 2));

  // Generate .env file for docker-compose
  console.log("\n📝 Generating environment files...");
  const envContent = validators
    .map(
      (v, i) => `
# ${v.name}
VALIDATOR_${i + 1}_PUBKEY=${v.publicKey}
VALIDATOR_${i + 1}_PEER_ID=${v.peerId}
`,
    )
    .join("\n");

  writeFileSync(join(DEVNET_DIR, ".env"), envContent);

  console.log("\n✅ Devnet initialization complete!\n");
  console.log("Next steps:");
  console.log("  1. cd devnet");
  console.log("  2. docker-compose up -d");
  console.log("  3. Check logs: docker-compose logs -f");
  console.log("\nRPC endpoints:");
  validators.forEach((v, i) => {
    console.log(`  ${v.name}: http://localhost:${8545 + i}`);
  });
  console.log("\nFaucet DID:", faucet.did);
}

main().catch(console.error);
