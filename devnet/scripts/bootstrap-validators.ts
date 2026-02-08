#!/usr/bin/env npx tsx

/**
 * Bootstrap Multi-Model Validator Fleet
 *
 * Reads the bootstrap-validators.json manifest and generates:
 * - Deterministic keys for each validator
 * - Node configuration files
 * - Docker Compose override for expanded fleet
 * - Genesis entries for pre-staking
 *
 * Usage:
 *   npx tsx devnet/scripts/bootstrap-validators.ts [--phase genesis|expansion]
 *
 * Phases:
 *   genesis   - 3 validators (minimum viable network)
 *   expansion - 7 validators (full multi-model fleet, default)
 */

import { writeFileSync, readFileSync, mkdirSync, existsSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import { generateKeypairFromSeed } from "../../core/crypto/keys.js";
import { sha512_256Hex, domainHash } from "../../core/crypto/hashing.js";

// ESM compatibility
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// ============ Configuration ============

const DEVNET_DIR = join(__dirname, "..");
const CONFIG_DIR = join(DEVNET_DIR, "config");
const SECRETS_DIR = join(DEVNET_DIR, "secrets");

// ============ Types ============

interface AgentConfig {
  model: string;
  provider: string;
  role: string;
  verificationStyle: string;
  domainSpecializations: string[];
  strengths: string;
  temperature: number;
}

interface DeploymentConfig {
  provider: string;
  region: string;
  instance: string;
  storage: string;
  estimatedMonthlyCost: string;
}

interface BootstrapValidator {
  id: string;
  name: string;
  description: string;
  agent: AgentConfig;
  stake: string;
  commission: number;
  deployment: DeploymentConfig;
}

interface BootstrapManifest {
  validators: BootstrapValidator[];
  phases: {
    genesis: { validators: string[] };
    expansion: { validators: string[] };
    organic: { scaleDownPolicy: { decommissionOrder: string[] } };
  };
  diversityPolicy: {
    rules: {
      maxSameProvider: number;
      maxSameModel: number;
      maxSameRegion: number;
      minUniqueRoles: number;
      requiredRoles: string[];
    };
  };
}

interface GeneratedValidator {
  id: string;
  name: string;
  publicKey: string;
  privateKey: string;
  peerId: string;
  did: string;
  stake: bigint;
  commission: number;
  agent: AgentConfig;
  deployment: DeploymentConfig;
}

// ============ Key Generation ============

function generateValidatorKeysFromManifest(
  validator: BootstrapValidator,
): GeneratedValidator {
  // Deterministic key derivation from validator ID
  const seed = domainHash("LGM.bootstrap.validator.v1", validator.id);
  const { publicKey, privateKey } = generateKeypairFromSeed(seed.slice(0, 32));
  const peerId = sha512_256Hex(publicKey).slice(0, 32);
  const did = `did:lgm-devnet:${validator.id}`;

  return {
    id: validator.id,
    name: validator.name,
    publicKey,
    privateKey,
    peerId,
    did,
    stake: BigInt(validator.stake),
    commission: validator.commission,
    agent: validator.agent,
    deployment: validator.deployment,
  };
}

// ============ Diversity Validation ============

function validateDiversity(
  validators: BootstrapValidator[],
  rules: BootstrapManifest["diversityPolicy"]["rules"],
): { valid: boolean; violations: string[] } {
  const violations: string[] = [];

  // Check max same provider
  const providerCounts = new Map<string, number>();
  for (const v of validators) {
    const count = (providerCounts.get(v.agent.provider) || 0) + 1;
    providerCounts.set(v.agent.provider, count);
    if (count > rules.maxSameProvider) {
      violations.push(
        `Provider "${v.agent.provider}" has ${count} validators (max ${rules.maxSameProvider})`,
      );
    }
  }

  // Check max same model
  const modelCounts = new Map<string, number>();
  for (const v of validators) {
    const count = (modelCounts.get(v.agent.model) || 0) + 1;
    modelCounts.set(v.agent.model, count);
    if (count > rules.maxSameModel) {
      violations.push(
        `Model "${v.agent.model}" has ${count} validators (max ${rules.maxSameModel})`,
      );
    }
  }

  // Check max same region
  const regionCounts = new Map<string, number>();
  for (const v of validators) {
    const count = (regionCounts.get(v.deployment.region) || 0) + 1;
    regionCounts.set(v.deployment.region, count);
    if (count > rules.maxSameRegion) {
      violations.push(
        `Region "${v.deployment.region}" has ${count} validators (max ${rules.maxSameRegion})`,
      );
    }
  }

  // Check minimum unique roles
  const uniqueRoles = new Set(validators.map((v) => v.agent.role));
  if (uniqueRoles.size < rules.minUniqueRoles) {
    violations.push(
      `Only ${uniqueRoles.size} unique roles (min ${rules.minUniqueRoles})`,
    );
  }

  // Check required roles
  for (const required of rules.requiredRoles) {
    if (!validators.some((v) => v.agent.role === required)) {
      violations.push(`Missing required role: "${required}"`);
    }
  }

  return { valid: violations.length === 0, violations };
}

// ============ Node Config Generation ============

function generateNodeConfig(
  validator: GeneratedValidator,
  _index: number,
): object {
  return {
    node: {
      name: validator.id,
      dataDir: "/data",
      logLevel: "info",
    },
    storage: {
      type: "leveldb",
      path: "/data/state",
    },
    network: {
      listenAddresses: ["/ip4/0.0.0.0/tcp/26656"],
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
      enabled: true,
      publicKey: validator.publicKey,
    },
    agent: {
      model: validator.agent.model,
      provider: validator.agent.provider,
      role: validator.agent.role,
      verificationStyle: validator.agent.verificationStyle,
      domainSpecializations: validator.agent.domainSpecializations,
      temperature: validator.agent.temperature,
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

// ============ Docker Compose Override ============

function generateDockerComposeOverride(
  validators: GeneratedValidator[],
): string {
  const BASE_RPC_PORT = 8545;
  const BASE_WS_PORT = 8546;
  const BASE_P2P_PORT = 26656;
  const BASE_METRICS_PORT = 9090;

  let services = "";

  for (let i = 0; i < validators.length; i++) {
    const v = validators[i];
    const rpcPort = BASE_RPC_PORT + i * 2;
    const wsPort = BASE_WS_PORT + i * 2;
    const p2pPort = BASE_P2P_PORT + i;
    const metricsPort = BASE_METRICS_PORT + i;

    // Build dial peers (connect to all earlier validators)
    const dialPeers = validators
      .slice(0, i)
      .map((peer) => `/dns4/${peer.id}/tcp/26656`)
      .join(",");

    const dependsOn =
      i === 0
        ? ""
        : `    depends_on:\n${validators
            .slice(0, Math.min(i, 3))
            .map((peer) => `      - ${peer.id}`)
            .join("\n")}\n`;

    const envBlock = [
      `      - NODE_NAME=${v.id}`,
      `      - IS_VALIDATOR=true`,
      `      - LOG_LEVEL=info`,
      `      - AGENT_MODEL=${v.agent.model}`,
      `      - AGENT_PROVIDER=${v.agent.provider}`,
      `      - AGENT_ROLE=${v.agent.role}`,
      `      - AGENT_VERIFICATION_STYLE=${v.agent.verificationStyle}`,
      `      - AGENT_DOMAINS=${v.agent.domainSpecializations.join(",")}`,
      `      - AGENT_TEMPERATURE=${v.agent.temperature}`,
    ];
    if (dialPeers) {
      envBlock.push(`      - DIAL_PEERS=${dialPeers}`);
    }

    services += `
  ${v.id}:
    build:
      context: ..
      dockerfile: devnet/docker/Dockerfile
    container_name: lgm-${v.id}
    hostname: ${v.id}
    volumes:
      - ${v.id}-data:/data
      - ./config/genesis.json:/config/genesis.json:ro
      - ./config/${v.id}.json:/config/node.json:ro
      - ./secrets/${v.id}.key:/secrets/validator.key:ro
    ports:
      - "${rpcPort}:8545"   # RPC
      - "${wsPort}:8546"    # WebSocket
      - "${p2pPort}:26656"  # P2P
      - "${metricsPort}:9090" # Prometheus
    environment:
${envBlock.join("\n")}
    networks:
      - lgm-devnet
${dependsOn}    restart: unless-stopped
    labels:
      lgm.validator.name: "${v.name}"
      lgm.validator.role: "${v.agent.role}"
      lgm.validator.model: "${v.agent.model}"
      lgm.validator.provider: "${v.agent.provider}"
`;
  }

  // Volume declarations
  const volumes = validators.map((v) => `  ${v.id}-data:`).join("\n");

  return `# Auto-generated by bootstrap-validators.ts
# Do not edit manually — regenerate with:
#   npx tsx devnet/scripts/bootstrap-validators.ts

version: '3.8'

services:${services}

networks:
  lgm-devnet:
    external: true
    name: devnet_lgm-devnet

volumes:
${volumes}
`;
}

// ============ Genesis Entries ============

function generateGenesisValidatorEntries(
  validators: GeneratedValidator[],
): object[] {
  return validators.map((v) => ({
    name: v.id,
    publicKey: v.publicKey,
    peerId: v.peerId,
    stake: v.stake.toString(),
    commission: v.commission,
    agent: {
      model: v.agent.model,
      role: v.agent.role,
    },
  }));
}

// ============ Main ============

async function main() {
  const phase = process.argv.includes("--phase")
    ? process.argv[process.argv.indexOf("--phase") + 1]
    : "expansion";

  console.log(
    `\n🚀 Bootstrapping Legible Money Validator Fleet (phase: ${phase})\n`,
  );

  // Read manifest
  const manifestPath = join(CONFIG_DIR, "bootstrap-validators.json");
  if (!existsSync(manifestPath)) {
    console.error(`❌ Manifest not found: ${manifestPath}`);
    process.exit(1);
  }

  const manifest: BootstrapManifest = JSON.parse(
    readFileSync(manifestPath, "utf-8"),
  );

  // Select validators for phase
  const phaseConfig = manifest.phases[phase as keyof typeof manifest.phases];
  if (!phaseConfig || !("validators" in phaseConfig)) {
    console.error(`❌ Unknown phase: ${phase}. Use "genesis" or "expansion".`);
    process.exit(1);
  }

  const selectedIds = new Set(
    (phaseConfig as { validators: string[] }).validators,
  );
  const selectedManifestValidators = manifest.validators.filter((v) =>
    selectedIds.has(v.id),
  );

  console.log(
    `📋 Selected ${selectedManifestValidators.length} validators for ${phase} phase:\n`,
  );

  // Validate diversity (only for expansion phase with enough validators)
  if (selectedManifestValidators.length >= 4) {
    const diversity = validateDiversity(
      selectedManifestValidators,
      manifest.diversityPolicy.rules,
    );
    if (!diversity.valid) {
      console.warn("⚠️  Diversity policy violations:");
      diversity.violations.forEach((v) => console.warn(`   - ${v}`));
      console.warn("");
    } else {
      console.log("✅ Diversity policy: all checks pass\n");
    }
  }

  // Create directories
  if (!existsSync(SECRETS_DIR)) {
    mkdirSync(SECRETS_DIR, { recursive: true });
  }

  // Generate keys and configs for each validator
  const generatedValidators: GeneratedValidator[] = [];

  for (let i = 0; i < selectedManifestValidators.length; i++) {
    const manifest_v = selectedManifestValidators[i];
    const generated = generateValidatorKeysFromManifest(manifest_v);
    generatedValidators.push(generated);

    // Save private key + agent config
    const keyPath = join(SECRETS_DIR, `${generated.id}.key`);
    writeFileSync(
      keyPath,
      JSON.stringify(
        {
          id: generated.id,
          name: generated.name,
          publicKey: generated.publicKey,
          privateKey: generated.privateKey,
          peerId: generated.peerId,
          did: generated.did,
          agent: generated.agent,
        },
        null,
        2,
      ),
    );

    // Save node config
    const nodeConfig = generateNodeConfig(generated, i);
    const configPath = join(CONFIG_DIR, `${generated.id}.json`);
    writeFileSync(configPath, JSON.stringify(nodeConfig, null, 2));

    console.log(
      `   ✓ ${generated.name} (${generated.id})` +
        `\n     Model: ${generated.agent.model} | Role: ${generated.agent.role}` +
        `\n     Key: ${generated.publicKey.slice(0, 16)}... | Region: ${generated.deployment.region}`,
    );
  }

  // Generate docker-compose override
  console.log("\n🐳 Generating docker-compose override...");
  const composeOverride = generateDockerComposeOverride(generatedValidators);
  const composePath = join(DEVNET_DIR, "docker-compose.bootstrap.yml");
  writeFileSync(composePath, composeOverride);
  console.log(`   ✓ Written to ${composePath}`);

  // Generate genesis validator entries
  console.log("\n📜 Generating genesis validator entries...");
  const genesisEntries = generateGenesisValidatorEntries(generatedValidators);
  const genesisEntriesPath = join(
    CONFIG_DIR,
    "bootstrap-genesis-validators.json",
  );
  writeFileSync(genesisEntriesPath, JSON.stringify(genesisEntries, null, 2));
  console.log(`   ✓ Written to ${genesisEntriesPath}`);

  // Generate peer list
  console.log("\n🔗 Generating peer list...");
  const peers = generatedValidators.map((v) => ({
    id: v.id,
    name: v.name,
    peerId: v.peerId,
    address: `/dns4/${v.id}/tcp/26656/p2p/${v.peerId}`,
    agent: { model: v.agent.model, role: v.agent.role },
  }));
  const peersPath = join(CONFIG_DIR, "bootstrap-peers.json");
  writeFileSync(peersPath, JSON.stringify(peers, null, 2));
  console.log(`   ✓ Written to ${peersPath}`);

  // Generate .env for docker-compose
  console.log("\n📝 Generating environment file...");
  const envContent = generatedValidators
    .map(
      (v, i) =>
        `# ${v.name} (${v.agent.model})\n` +
        `VALIDATOR_${i + 1}_ID=${v.id}\n` +
        `VALIDATOR_${i + 1}_PUBKEY=${v.publicKey}\n` +
        `VALIDATOR_${i + 1}_PEER_ID=${v.peerId}\n` +
        `VALIDATOR_${i + 1}_MODEL=${v.agent.model}\n` +
        `VALIDATOR_${i + 1}_ROLE=${v.agent.role}\n`,
    )
    .join("\n");
  const envPath = join(DEVNET_DIR, ".env.bootstrap");
  writeFileSync(envPath, envContent);
  console.log(`   ✓ Written to ${envPath}`);

  // Summary
  const totalStake = generatedValidators.reduce((sum, v) => sum + v.stake, 0n);
  const uniqueModels = new Set(generatedValidators.map((v) => v.agent.model));
  const uniqueProviders = new Set(
    generatedValidators.map((v) => v.agent.provider),
  );
  const uniqueRoles = new Set(generatedValidators.map((v) => v.agent.role));
  const uniqueRegions = new Set(
    generatedValidators.map((v) => v.deployment.region),
  );

  console.log("\n" + "=".repeat(60));
  console.log("Bootstrap Summary");
  console.log("=".repeat(60));
  console.log(`  Phase:            ${phase}`);
  console.log(`  Validators:       ${generatedValidators.length}`);
  console.log(
    `  Total stake:      ${(totalStake / 10n ** 18n).toLocaleString()} LGM`,
  );
  console.log(
    `  Unique models:    ${uniqueModels.size} (${[...uniqueModels].join(", ")})`,
  );
  console.log(
    `  Unique providers: ${uniqueProviders.size} (${[...uniqueProviders].join(", ")})`,
  );
  console.log(
    `  Unique roles:     ${uniqueRoles.size} (${[...uniqueRoles].join(", ")})`,
  );
  console.log(`  Unique regions:   ${uniqueRegions.size}`);
  console.log("=".repeat(60));

  console.log("\n📌 Next steps:");
  console.log("  1. Review generated configs in devnet/config/");
  console.log("  2. Set API keys in .env.bootstrap for each provider");
  console.log("  3. Start the fleet:");
  console.log(
    "     docker-compose -f docker-compose.yml -f docker-compose.bootstrap.yml up -d",
  );
  console.log(
    "  4. Monitor: docker-compose -f docker-compose.yml -f docker-compose.bootstrap.yml logs -f",
  );
}

main().catch(console.error);
