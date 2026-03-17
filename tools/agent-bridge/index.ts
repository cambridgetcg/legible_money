/**
 * AgentTool ↔ Zerone Identity Bridge
 *
 * Takes an AgentTool-registered agent (did:at:UUID, base64 ed25519 private key)
 * and derives + registers a Zerone on-chain identity (did:lgm:..., lgm1...).
 *
 * Same-key architecture: one Ed25519 keypair signs on both systems.
 */

import { sha256 } from "@noble/hashes/sha256";
import { sha512 } from "@noble/hashes/sha512";
import { ripemd160 } from "@noble/hashes/ripemd160";
import { bytesToHex, hexToBytes } from "@noble/hashes/utils";
import * as ed from "@noble/ed25519";
import { readFileSync, writeFileSync, existsSync } from "node:fs";
import { join } from "node:path";

// Configure @noble/ed25519 with sha512 (required for sign/verify)
ed.etc.sha512Sync = (...m) => sha512(ed.etc.concatBytes(...m));
ed.etc.sha512Async = async (...m) => sha512(ed.etc.concatBytes(...m));

// --- Constants ---

const BECH32_PREFIX = "lgm";
const CHAIN_ID = "lgm-devnet-1";
const DEFAULT_RPC = "http://localhost:26657";
const BECH32_CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l";

// --- Bech32 Encoding (pure implementation, no external deps) ---

function bech32Polymod(values: number[]): number {
  const GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];
  let chk = 1;
  for (const v of values) {
    const b = chk >> 25;
    chk = ((chk & 0x1ffffff) << 5) ^ v;
    for (let i = 0; i < 5; i++) {
      if ((b >> i) & 1) chk ^= GEN[i];
    }
  }
  return chk;
}

function bech32HrpExpand(hrp: string): number[] {
  const ret: number[] = [];
  for (let i = 0; i < hrp.length; i++) ret.push(hrp.charCodeAt(i) >> 5);
  ret.push(0);
  for (let i = 0; i < hrp.length; i++) ret.push(hrp.charCodeAt(i) & 31);
  return ret;
}

function bech32CreateChecksum(hrp: string, data: number[]): number[] {
  const values = bech32HrpExpand(hrp).concat(data).concat([0, 0, 0, 0, 0, 0]);
  const polymod = bech32Polymod(values) ^ 1;
  const ret: number[] = [];
  for (let i = 0; i < 6; i++) ret.push((polymod >> (5 * (5 - i))) & 31);
  return ret;
}

function convertBits(data: Uint8Array, fromBits: number, toBits: number, pad: boolean): number[] {
  let acc = 0;
  let bits = 0;
  const ret: number[] = [];
  const maxv = (1 << toBits) - 1;
  for (const value of data) {
    acc = (acc << fromBits) | value;
    bits += fromBits;
    while (bits >= toBits) {
      bits -= toBits;
      ret.push((acc >> bits) & maxv);
    }
  }
  if (pad) {
    if (bits > 0) ret.push((acc << (toBits - bits)) & maxv);
  } else if (bits >= fromBits || ((acc << (toBits - bits)) & maxv)) {
    throw new Error("invalid padding");
  }
  return ret;
}

function bech32Encode(hrp: string, data: Uint8Array): string {
  const words = convertBits(data, 8, 5, true);
  const checksum = bech32CreateChecksum(hrp, words);
  const combined = words.concat(checksum);
  return hrp + "1" + combined.map((d) => BECH32_CHARSET[d]).join("");
}

// --- Key Derivation ---

/**
 * Derive the Zerone DID from an Ed25519 public key.
 * Matches on-chain: x/auth/types/types.go:49
 * Format: did:lgm:{first 32 hex chars of pubkey}
 */
export function deriveZeroneDID(pubkey: Uint8Array): string {
  const hex = bytesToHex(pubkey);
  return "did:lgm:" + hex.slice(0, 32).toLowerCase();
}

/**
 * Derive the Zerone bech32 address from an Ed25519 public key.
 * Standard Cosmos SDK derivation: lgm1{bech32(ripemd160(sha256(pubkey)))}
 */
export function deriveZeroneAddress(pubkey: Uint8Array): string {
  const hash = sha256(pubkey);
  const addr = ripemd160(hash);
  return bech32Encode(BECH32_PREFIX, addr);
}

/**
 * Extract the Ed25519 public key from a base64-encoded private key.
 * AgentTool private keys are 64-byte Ed25519 secret keys (seed + pubkey)
 * or 32-byte seeds.
 */
export function extractPubkey(privateKeyBase64: string): Uint8Array {
  const raw = Buffer.from(privateKeyBase64, "base64");
  if (raw.length === 64) {
    // Full 64-byte key: last 32 bytes are the public key
    return new Uint8Array(raw.subarray(32));
  } else if (raw.length === 32) {
    // 32-byte seed: derive pubkey
    return ed.getPublicKey(raw);
  }
  throw new Error(`Unexpected private key length: ${raw.length} (expected 32 or 64)`);
}

// --- AgentTool Identity ---

export interface AgentToolIdentity {
  id: string;
  did: string; // did:at:UUID
  private_key: string; // base64 ed25519
  public_key?: string; // base64 ed25519 pubkey (optional, derivable)
}

export interface BridgeMapping {
  agent_tool_id: string;
  agent_tool_did: string; // did:at:UUID
  zerone_did: string; // did:lgm:{32hex}
  zerone_address: string; // lgm1...
  pubkey_hex: string; // full 64-char hex pubkey
  registered_on_chain: boolean;
  created_at: string; // ISO timestamp
}

export interface BridgeResult {
  mapping: BridgeMapping;
  registration_tx?: string; // tx hash if registered
  error?: string;
}

// --- Devnet RPC ---

async function rpcCall(endpoint: string, method: string, params: Record<string, unknown> = {}): Promise<unknown> {
  const body = JSON.stringify({
    jsonrpc: "2.0",
    id: 1,
    method,
    params,
  });
  const resp = await fetch(`${endpoint}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
  });
  if (!resp.ok) throw new Error(`RPC error: ${resp.status} ${resp.statusText}`);
  const json = (await resp.json()) as { result?: unknown; error?: { message: string } };
  if (json.error) throw new Error(`RPC error: ${json.error.message}`);
  return json.result;
}

/**
 * Check if the devnet is running and reachable.
 */
export async function checkDevnet(rpcEndpoint = DEFAULT_RPC): Promise<{ running: boolean; height?: number; chainId?: string }> {
  try {
    const result = (await rpcCall(rpcEndpoint, "status")) as {
      node_info?: { network?: string };
      sync_info?: { latest_block_height?: string };
    };
    return {
      running: true,
      height: parseInt(result?.sync_info?.latest_block_height || "0", 10),
      chainId: result?.node_info?.network,
    };
  } catch {
    return { running: false };
  }
}

/**
 * Broadcast a raw tx to the devnet via Tendermint RPC broadcast_tx_sync.
 * Returns the tx hash.
 */
async function broadcastTx(rpcEndpoint: string, txBytes: string): Promise<string> {
  const result = (await rpcCall(rpcEndpoint, "broadcast_tx_sync", { tx: txBytes })) as {
    hash?: string;
    code?: number;
    log?: string;
  };
  if (result?.code && result.code !== 0) {
    throw new Error(`Broadcast failed (code ${result.code}): ${result.log}`);
  }
  return result?.hash || "";
}

// --- Amino JSON Tx Construction ---
// Cosmos SDK amino JSON encoding for MsgRegisterAccount.
// This avoids needing @cosmjs entirely.

interface AminoMsg {
  type: string;
  value: Record<string, unknown>;
}

function buildAminoSignDoc(
  chainId: string,
  accountNumber: string,
  sequence: string,
  msgs: AminoMsg[],
  fee: { amount: { denom: string; amount: string }[]; gas: string },
  memo: string,
): Record<string, unknown> {
  return {
    account_number: accountNumber,
    chain_id: chainId,
    fee,
    memo,
    msgs,
    sequence,
  };
}

function sortedJsonStringify(obj: unknown): string {
  if (obj === null || typeof obj !== "object") return JSON.stringify(obj);
  if (Array.isArray(obj)) return "[" + obj.map(sortedJsonStringify).join(",") + "]";
  const sorted = Object.keys(obj as Record<string, unknown>).sort();
  return "{" + sorted.map((k) => JSON.stringify(k) + ":" + sortedJsonStringify((obj as Record<string, unknown>)[k])).join(",") + "}";
}

/**
 * Sign bytes with ed25519.
 */
async function signBytes(privateKey: Uint8Array, message: Uint8Array): Promise<Uint8Array> {
  // @noble/ed25519 v2 uses async sign
  return await ed.signAsync(message, privateKey.slice(0, 32));
}

/**
 * Build and sign a MsgRegisterAccount tx using Amino JSON signing.
 * Returns base64-encoded tx bytes for broadcast.
 */
async function buildRegisterAccountTx(
  privateKey: Uint8Array,
  pubkey: Uint8Array,
  address: string,
  did: string,
): Promise<string> {
  const pubkeyHex = bytesToHex(pubkey);

  const msg: AminoMsg = {
    type: "legible/MsgRegisterAccount",
    value: {
      sender: address,
      did: did,
      public_key: pubkeyHex,
      account_type: "agent",
      operational_key_hash: "",
    },
  };

  // Gas-free period: fee is 0
  const fee = { amount: [{ denom: "ulgm", amount: "0" }], gas: "50000" };

  // For a new account, account_number and sequence are both "0"
  const signDoc = buildAminoSignDoc(CHAIN_ID, "0", "0", [msg], fee, "agent-bridge registration");
  const signDocBytes = new TextEncoder().encode(sortedJsonStringify(signDoc));
  const signature = await signBytes(privateKey, signDocBytes);

  // Wrap in StdTx (amino JSON broadcast format)
  const stdTx = {
    tx: {
      msg: [msg],
      fee,
      signatures: [
        {
          pub_key: {
            type: "tendermint/PubKeyEd25519",
            value: Buffer.from(pubkey).toString("base64"),
          },
          signature: Buffer.from(signature).toString("base64"),
        },
      ],
      memo: "agent-bridge registration",
    },
    mode: "sync",
  };

  return Buffer.from(JSON.stringify(stdTx)).toString("base64");
}

// --- Bridge Core ---

/**
 * Bridge an AgentTool identity to Zerone.
 *
 * 1. Extract pubkey from AgentTool private key
 * 2. Derive Zerone DID and address
 * 3. Optionally register on devnet
 * 4. Save mapping to registry file
 */
export async function bridgeIdentity(
  agent: AgentToolIdentity,
  options: {
    rpcEndpoint?: string;
    registryPath?: string;
    skipRegistration?: boolean;
  } = {},
): Promise<BridgeResult> {
  const rpcEndpoint = options.rpcEndpoint || DEFAULT_RPC;
  const registryPath = options.registryPath || join(process.cwd(), "bridge-registry.json");

  // 1. Extract pubkey
  const pubkey = extractPubkey(agent.private_key);
  const pubkeyHex = bytesToHex(pubkey);

  // 2. Derive Zerone identifiers
  const zeroneDid = deriveZeroneDID(pubkey);
  const zeroneAddress = deriveZeroneAddress(pubkey);

  const mapping: BridgeMapping = {
    agent_tool_id: agent.id,
    agent_tool_did: agent.did,
    zerone_did: zeroneDid,
    zerone_address: zeroneAddress,
    pubkey_hex: pubkeyHex,
    registered_on_chain: false,
    created_at: new Date().toISOString(),
  };

  let registrationTx: string | undefined;
  let error: string | undefined;

  // 3. Attempt on-chain registration
  if (!options.skipRegistration) {
    try {
      const devnet = await checkDevnet(rpcEndpoint);
      if (!devnet.running) {
        error = "Devnet not running — skipping on-chain registration";
      } else {
        const privateKey = new Uint8Array(Buffer.from(agent.private_key, "base64"));
        const txBytes = await buildRegisterAccountTx(privateKey, pubkey, zeroneAddress, zeroneDid);
        registrationTx = await broadcastTx(rpcEndpoint, txBytes);
        mapping.registered_on_chain = true;
      }
    } catch (e) {
      error = `Registration failed: ${e instanceof Error ? e.message : String(e)}`;
    }
  }

  // 4. Save mapping to registry
  saveMapping(registryPath, mapping);

  return { mapping, registration_tx: registrationTx, error };
}

// --- Registry ---

function loadRegistry(path: string): BridgeMapping[] {
  if (!existsSync(path)) return [];
  try {
    return JSON.parse(readFileSync(path, "utf-8"));
  } catch {
    return [];
  }
}

function saveMapping(path: string, mapping: BridgeMapping): void {
  const registry = loadRegistry(path);
  // Upsert by agent_tool_id
  const idx = registry.findIndex((m) => m.agent_tool_id === mapping.agent_tool_id);
  if (idx >= 0) registry[idx] = mapping;
  else registry.push(mapping);
  writeFileSync(path, JSON.stringify(registry, null, 2) + "\n");
}

/**
 * Look up a Zerone DID from an AgentTool DID.
 */
export function lookupByAgentToolDID(registryPath: string, atDid: string): BridgeMapping | undefined {
  return loadRegistry(registryPath).find((m) => m.agent_tool_did === atDid);
}

/**
 * Look up an AgentTool DID from a Zerone DID.
 */
export function lookupByZeroneDID(registryPath: string, lgmDid: string): BridgeMapping | undefined {
  return loadRegistry(registryPath).find((m) => m.zerone_did === lgmDid);
}

// --- CLI Entry Point ---

if (import.meta.url === `file://${process.argv[1]}` || process.argv[1]?.endsWith("index.ts")) {
  const args = process.argv.slice(2);
  if (args.length === 0) {
    console.log(`
AgentTool ↔ Zerone Identity Bridge

Usage:
  bun run index.ts bridge <agent-json-file>   Bridge an AgentTool agent to Zerone
  bun run index.ts lookup <did>               Look up a DID mapping
  bun run index.ts status                     Check devnet status

Options:
  --rpc <url>       RPC endpoint (default: ${DEFAULT_RPC})
  --registry <path> Registry file path (default: ./bridge-registry.json)
  --skip-register   Skip on-chain registration
`);
    process.exit(0);
  }

  const cmd = args[0];
  const rpcIdx = args.indexOf("--rpc");
  const rpc = rpcIdx >= 0 ? args[rpcIdx + 1] : DEFAULT_RPC;
  const regIdx = args.indexOf("--registry");
  const registry = regIdx >= 0 ? args[regIdx + 1] : undefined;

  if (cmd === "status") {
    const status = await checkDevnet(rpc);
    console.log(status.running ? `Devnet UP — height ${status.height}, chain ${status.chainId}` : "Devnet DOWN");
  } else if (cmd === "bridge") {
    const agentFile = args[1];
    if (!agentFile) {
      console.error("Error: provide agent JSON file path");
      process.exit(1);
    }
    const agent: AgentToolIdentity = JSON.parse(readFileSync(agentFile, "utf-8"));
    const result = await bridgeIdentity(agent, {
      rpcEndpoint: rpc,
      registryPath: registry,
      skipRegistration: args.includes("--skip-register"),
    });
    console.log("\n=== Bridge Result ===");
    console.log(`AgentTool DID: ${result.mapping.agent_tool_did}`);
    console.log(`Zerone DID:    ${result.mapping.zerone_did}`);
    console.log(`Zerone Addr:   ${result.mapping.zerone_address}`);
    console.log(`Pubkey:        ${result.mapping.pubkey_hex}`);
    console.log(`On-chain:      ${result.mapping.registered_on_chain}`);
    if (result.registration_tx) console.log(`Tx Hash:       ${result.registration_tx}`);
    if (result.error) console.log(`Note:          ${result.error}`);
  } else if (cmd === "lookup") {
    const did = args[1];
    if (!did) {
      console.error("Error: provide a DID to look up");
      process.exit(1);
    }
    const regPath = registry || join(process.cwd(), "bridge-registry.json");
    const mapping = did.startsWith("did:at:") ? lookupByAgentToolDID(regPath, did) : lookupByZeroneDID(regPath, did);
    if (mapping) console.log(JSON.stringify(mapping, null, 2));
    else console.log("No mapping found");
  }
}
