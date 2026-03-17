# AgentTool ↔ Zerone Identity Bridge

Connects an [AgentTool](https://agent-identity.fly.dev) agent identity to the Zerone (Legible Money) blockchain.

## What It Does

An agent registered with AgentTool has:
- An `id` (UUID)
- A `did:at:<UUID>` decentralized identifier
- An Ed25519 keypair (base64-encoded private key)

This bridge derives a **Zerone on-chain identity** from the **same keypair**:
- `did:lgm:<first 32 hex chars of pubkey>` — the Zerone DID
- `lgm1<bech32(ripemd160(sha256(pubkey)))>` — the Zerone bech32 address

Same-key architecture means the agent signs transactions on both systems with one keypair.

## DID Mapping

```
AgentTool                           Zerone
─────────                           ──────
did:at:550e8400-...                 did:lgm:a1b2c3d4e5f6a7b8...
       ↑                                  ↑
       UUID identifier                    First 32 hex chars of Ed25519 pubkey

Both derived from the same Ed25519 keypair.
```

The mapping is saved to a local JSON registry file (`bridge-registry.json`) for bidirectional lookup.

## Usage

### Install dependencies

```bash
cd tools/agent-bridge
bun install
```

### Bridge an agent

```bash
# From a JSON file with AgentTool credentials
bun run index.ts bridge agent.json

# Skip on-chain registration (just derive identifiers)
bun run index.ts bridge agent.json --skip-register

# Custom RPC endpoint
bun run index.ts bridge agent.json --rpc http://mynode:26657
```

Agent JSON format:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "did": "did:at:550e8400-e29b-41d4-a716-446655440000",
  "private_key": "<base64 ed25519 private key>"
}
```

### Look up a mapping

```bash
# By AgentTool DID
bun run index.ts lookup did:at:550e8400-...

# By Zerone DID
bun run index.ts lookup did:lgm:a1b2c3d4...
```

### Check devnet status

```bash
bun run index.ts status
```

### Run tests

```bash
bun run bridge-test.ts
```

## Programmatic API

```typescript
import { bridgeIdentity, deriveZeroneDID, deriveZeroneAddress, extractPubkey } from "./index.ts";

// Bridge an AgentTool agent
const result = await bridgeIdentity({
  id: "...",
  did: "did:at:...",
  private_key: "<base64>",
});

console.log(result.mapping.zerone_did);     // did:lgm:a1b2c3d4...
console.log(result.mapping.zerone_address); // lgm1...

// Or derive identifiers directly
const pubkey = extractPubkey(base64PrivateKey);
const did = deriveZeroneDID(pubkey);        // did:lgm:...
const addr = deriveZeroneAddress(pubkey);   // lgm1...
```

## Architecture

```
┌──────────────┐     same Ed25519 key     ┌──────────────┐
│  AgentTool   │ ──────────────────────── │   Zerone     │
│  Identity    │                           │   Chain      │
│              │                           │              │
│ did:at:UUID  │  ←── bridge-registry ──→  │ did:lgm:hex  │
│ POST /agents │      (JSON file)          │ lgm1...      │
└──────────────┘                           └──────────────┘
       ↑                                          ↑
       │              ┌────────────┐              │
       └──────────────│   Bridge   │──────────────┘
                      │  index.ts  │
                      └────────────┘
```

## On-Chain Registration

During the gas-free bootstrap period (first ~480,000 blocks / ~14 days), registration is free. The bridge broadcasts a `MsgRegisterAccount` via Tendermint RPC with zero fees.

After bootstrap, the agent needs LGM tokens for gas. Use the devnet faucet or transfer from another account.
