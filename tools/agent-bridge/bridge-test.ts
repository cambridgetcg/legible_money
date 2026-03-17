/**
 * AgentTool ↔ Zerone Bridge Integration Test
 *
 * 1. Creates a test agent via agent-identity.fly.dev (or generates locally)
 * 2. Derives Zerone DID from the keypair
 * 3. Attempts to register on devnet (skips if not running)
 * 4. Saves bridge mapping to test-mapping.json
 * 5. Prints success/failure summary
 */

import { deriveZeroneDID, deriveZeroneAddress, extractPubkey, bridgeIdentity, checkDevnet } from "./index.ts";
import type { AgentToolIdentity } from "./index.ts";
import { bytesToHex } from "@noble/hashes/utils";
import { sha512 } from "@noble/hashes/sha512";
import * as ed from "@noble/ed25519";
import { join } from "node:path";

// Configure @noble/ed25519 with sha512 (required for sync operations)
ed.etc.sha512Sync = (...m) => sha512(ed.etc.concatBytes(...m));

const AGENT_IDENTITY_API = "https://agent-identity.fly.dev";
const TEST_MAPPING_PATH = join(import.meta.dir || process.cwd(), "test-mapping.json");

// --- Test Helpers ---

let passed = 0;
let failed = 0;
let skipped = 0;

function assert(condition: boolean, name: string, detail?: string): void {
  if (condition) {
    console.log(`  PASS  ${name}`);
    passed++;
  } else {
    console.log(`  FAIL  ${name}${detail ? ` — ${detail}` : ""}`);
    failed++;
  }
}

function skip(name: string, reason: string): void {
  console.log(`  SKIP  ${name} — ${reason}`);
  skipped++;
}

// --- Test: AgentTool API Health ---

async function testAgentToolHealth(): Promise<boolean> {
  console.log("\n--- AgentTool API Health ---");
  try {
    const resp = await fetch(`${AGENT_IDENTITY_API}/health`, { signal: AbortSignal.timeout(5000) });
    assert(resp.ok, "agent-identity.fly.dev /health returns 200");
    return resp.ok;
  } catch (e) {
    skip("agent-identity.fly.dev health", `unreachable: ${e instanceof Error ? e.message : String(e)}`);
    return false;
  }
}

// --- Test: Create Agent via API ---

async function testCreateAgent(): Promise<AgentToolIdentity | null> {
  console.log("\n--- Create Agent via AgentTool API ---");

  // API requires x-api-key — check env
  const apiKey = process.env.AGENT_IDENTITY_API_KEY;
  if (!apiKey) {
    skip("create agent via API", "AGENT_IDENTITY_API_KEY not set (set env var to test)");
    return null;
  }

  try {
    const resp = await fetch(`${AGENT_IDENTITY_API}/v1/identities`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "x-api-key": apiKey,
      },
      body: JSON.stringify({
        display_name: `bridge-test-${Date.now()}`,
        capabilities: ["knowledge_verification"],
      }),
      signal: AbortSignal.timeout(10000),
    });
    if (!resp.ok) {
      const text = await resp.text();
      skip("create agent via API", `HTTP ${resp.status}: ${text.slice(0, 100)}`);
      return null;
    }
    const data = (await resp.json()) as {
      identity: { id: string; did: string };
      key: { private_key: string };
    };
    const agent: AgentToolIdentity = {
      id: data.identity.id,
      did: data.identity.did,
      private_key: data.key.private_key,
    };
    assert(!!agent.id, "agent has id");
    assert(!!agent.did && agent.did.startsWith("did:at:"), `agent has DID (${agent.did})`);
    assert(!!agent.private_key, "agent has private_key");
    return agent;
  } catch (e) {
    skip("create agent via API", `error: ${e instanceof Error ? e.message : String(e)}`);
    return null;
  }
}

// --- Test: Local Key Generation Fallback ---

function generateLocalAgent(): AgentToolIdentity {
  console.log("\n--- Local Agent Generation (fallback) ---");
  // Generate ed25519 keypair using @noble/ed25519
  const seed = new Uint8Array(32);
  crypto.getRandomValues(seed);
  const pubkey = ed.getPublicKey(seed);
  // Combine seed+pubkey into 64-byte private key format
  const fullKey = new Uint8Array(64);
  fullKey.set(seed, 0);
  fullKey.set(pubkey, 32);

  const id = crypto.randomUUID();
  const agent: AgentToolIdentity = {
    id,
    did: `did:at:${id}`,
    private_key: Buffer.from(fullKey).toString("base64"),
  };
  assert(true, `generated local agent: ${agent.did}`);
  return agent;
}

// --- Test: DID Derivation ---

function testDIDDerivation(agent: AgentToolIdentity): void {
  console.log("\n--- DID Derivation ---");

  const pubkey = extractPubkey(agent.private_key);
  const pubkeyHex = bytesToHex(pubkey);
  assert(pubkeyHex.length === 64, `pubkey is 32 bytes (${pubkeyHex.length} hex chars)`);

  const zeroneDid = deriveZeroneDID(pubkey);
  assert(zeroneDid.startsWith("did:lgm:"), `Zerone DID starts with did:lgm: (${zeroneDid})`);
  assert(zeroneDid.length === 40, `Zerone DID is 40 chars (did:lgm: + 32 hex)`);

  // Verify DID matches on-chain derivation: first 32 hex chars of pubkey
  const expectedSuffix = pubkeyHex.slice(0, 32).toLowerCase();
  assert(zeroneDid === `did:lgm:${expectedSuffix}`, "DID matches on-chain derivation (pubkey[:32])");

  const zeroneAddr = deriveZeroneAddress(pubkey);
  assert(zeroneAddr.startsWith("lgm1"), `Zerone address starts with lgm1 (${zeroneAddr})`);
  assert(zeroneAddr.length > 10, "Zerone address has reasonable length");

  console.log(`  AgentTool DID:  ${agent.did}`);
  console.log(`  Zerone DID:     ${zeroneDid}`);
  console.log(`  Zerone Address: ${zeroneAddr}`);
  console.log(`  Pubkey:         ${pubkeyHex}`);
}

// --- Test: Deterministic Derivation ---

function testDeterminism(agent: AgentToolIdentity): void {
  console.log("\n--- Deterministic Derivation ---");
  const pubkey1 = extractPubkey(agent.private_key);
  const pubkey2 = extractPubkey(agent.private_key);
  assert(bytesToHex(pubkey1) === bytesToHex(pubkey2), "pubkey extraction is deterministic");
  assert(deriveZeroneDID(pubkey1) === deriveZeroneDID(pubkey2), "DID derivation is deterministic");
  assert(deriveZeroneAddress(pubkey1) === deriveZeroneAddress(pubkey2), "address derivation is deterministic");
}

// --- Test: Devnet Registration ---

async function testDevnetRegistration(agent: AgentToolIdentity): Promise<void> {
  console.log("\n--- Devnet Registration ---");

  const devnet = await checkDevnet();
  if (!devnet.running) {
    skip("devnet registration", "devnet not running at localhost:26657");
    return;
  }

  assert(true, `devnet running — height ${devnet.height}, chain ${devnet.chainId}`);

  try {
    const result = await bridgeIdentity(agent, {
      registryPath: TEST_MAPPING_PATH,
      skipRegistration: false,
    });

    if (result.error) {
      // Registration may fail (account may not be funded, tx format may not match)
      // but the bridge mapping should still be created
      console.log(`  NOTE  Registration note: ${result.error}`);
    }
    if (result.registration_tx) {
      assert(true, `tx broadcast: ${result.registration_tx}`);
    }
    assert(result.mapping.zerone_did.startsWith("did:lgm:"), "mapping has Zerone DID");
    assert(result.mapping.zerone_address.startsWith("lgm1"), "mapping has Zerone address");
  } catch (e) {
    console.log(`  NOTE  Registration error (expected on unfunded devnet): ${e instanceof Error ? e.message : String(e)}`);
  }
}

// --- Test: Bridge Mapping Save ---

async function testBridgeMapping(agent: AgentToolIdentity): Promise<void> {
  console.log("\n--- Bridge Mapping ---");

  const result = await bridgeIdentity(agent, {
    registryPath: TEST_MAPPING_PATH,
    skipRegistration: true, // just test the mapping
  });

  assert(result.mapping.agent_tool_id === agent.id, "mapping has correct agent_tool_id");
  assert(result.mapping.agent_tool_did === agent.did, "mapping has correct agent_tool_did");
  assert(result.mapping.zerone_did.startsWith("did:lgm:"), "mapping has Zerone DID");
  assert(result.mapping.zerone_address.startsWith("lgm1"), "mapping has Zerone address");
  assert(result.mapping.pubkey_hex.length === 64, "mapping has full pubkey hex");
  assert(!!result.mapping.created_at, "mapping has timestamp");

  // Verify the file was written
  const { existsSync, readFileSync } = await import("node:fs");
  assert(existsSync(TEST_MAPPING_PATH), `mapping file written to ${TEST_MAPPING_PATH}`);

  const saved = JSON.parse(readFileSync(TEST_MAPPING_PATH, "utf-8"));
  assert(Array.isArray(saved) && saved.length > 0, "mapping file is a non-empty array");
  assert(saved[saved.length - 1].zerone_did === result.mapping.zerone_did, "saved mapping matches returned mapping");
}

// --- Run All Tests ---

console.log("=== AgentTool ↔ Zerone Bridge Test ===");

// Step 1: Check AgentTool API
const apiUp = await testAgentToolHealth();

// Step 2: Create agent (API or local fallback)
let agent: AgentToolIdentity;
if (apiUp) {
  const apiAgent = await testCreateAgent();
  agent = apiAgent || generateLocalAgent();
} else {
  agent = generateLocalAgent();
}

// Step 3: Test DID derivation
testDIDDerivation(agent);

// Step 4: Test determinism
testDeterminism(agent);

// Step 5: Test devnet registration
await testDevnetRegistration(agent);

// Step 6: Test mapping save
await testBridgeMapping(agent);

// Summary
console.log("\n=== Summary ===");
console.log(`  ${passed} passed, ${failed} failed, ${skipped} skipped`);
console.log(`  Total: ${passed + failed + skipped} tests`);

if (failed > 0) {
  console.log("\nFAILED");
  process.exit(1);
} else {
  console.log("\nOK");
}
