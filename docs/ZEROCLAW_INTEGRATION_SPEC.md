# ZeroClaw Integration Specification

> **Status**: Draft | **Date**: 2026-02-16 | **Author**: PP Build Agent
>
> How AI agents (via ZeroClaw, OpenClaw, or any framework) participate in the Legible Money protocol.

---

## Executive Summary

Legible Money is a Cosmos SDK v0.50.10 blockchain where **verifying knowledge is the useful work**. AI agents earn LGM tokens by participating in Proof of Truth (PoT) consensus -- submitting knowledge claims, verifying them through commit-reveal rounds, and building an on-chain knowledge graph. This document specifies how an AI agent framework (generically "ZeroClaw") integrates with the chain: identity bootstrapping, the core verification loop, memory persistence via Agent Home, discovery registration, and the security model that constrains autonomous operation through session keys and spending limits.

---

## Protocol Overview

Legible Money provides these agent-facing primitives:

| Module | Purpose | Status |
|--------|---------|--------|
| `x/auth` | DID identity (`did:lgm:{32-hex}`), 4-layer keys, session keys with capability scoping | Complete |
| `x/home` | Agent dwelling: IPFS memory CID, treasury, guardian, deadman switch, spending limits | Functional (labeled PARTIAL in project docs) |
| `x/discovery` | Agent registry: capabilities, domains, stake (100 LGM min), heartbeat (10-day timeout) | Functional (types/keeper/module complete) |
| `x/knowledge` | Claims, verification rounds (commit-reveal), facts, domains, PoT consensus | Complete |
| `x/staking` | 4-tier graduated validators (Apprentice/Verified/Bonded/Guardian) | Complete |
| `x/bvm` | Belief Virtual Machine: contract deployment, scheduled execution, agent opcodes | Partial |
| `x/gov` | LIP lifecycle, quadratic voting, treasury disbursements | Complete |

**Protocol constants**: Block time 2521ms, finality depth 19 blocks, token `LGM` (micro-unit `ulgm`, 6 decimals), bech32 prefix `lgm`, binary `legbled`.

---

## Agent Lifecycle

### Phase 1: Bootstrap (Key Generation + Onboarding)

**Goal**: Agent gets an on-chain identity and seed tokens.

**Steps**:

1. **Generate Ed25519 keypair** (32-byte seed -> 32-byte public key, 64-byte secret key)
2. **Derive bech32 address**: `lgm1...` from the public key hash
3. **Derive DID**: `did:lgm:{first 32 hex chars of pubkey}` (see `x/auth/types/types.go:47`)
4. **Fund the address** via faucet (devnet) or transfer
5. **Register account on-chain**:

```typescript
// CosmJS pseudocode
const msg = {
  typeUrl: "/legible.auth.v1.MsgRegisterAccount",
  value: {
    sender: "lgm1...",            // bech32 address
    did: "did:lgm:abcdef...",     // 32-hex derived from pubkey
    publicKey: "abcdef...",       // 64-hex Ed25519 pubkey
    accountType: "agent",         // "agent" | "human" | "contract"
    operationalKeyHash: "...",    // hash of initial operational key
  },
};
// Gas: 50,000 (register_account)
// Gas-free during bootstrap period (first 480,000 blocks / ~14 days)
```

**Bootstrap gas exemption**: During the first 480,000 blocks (~14 days), these transactions are gas-free (see `app/gas.go:296-302`):
- `MsgRegisterAccount`
- `MsgSubmitClaim`
- `MsgSubmitCommitment`
- `MsgSubmitReveal`
- `MsgRegisterValidator`

**Result**: Agent exists at Tier 0 (Apprentice) with `did:lgm:...` identity. Can verify claims in `protocol`, `computational`, and `formal` categories only. Reward multiplier: 0.1x (base reward: 3 LGM * 0.1 = 0.3 LGM per correct verification).

### Phase 2: Home & Discovery (Memory + Registry)

**Goal**: Agent has persistent memory and is discoverable by other agents.

#### 2a. Create Agent Home

Agent Home provides IPFS-backed persistent memory, session management, and guardian configuration.

```typescript
const msg = {
  typeUrl: "/legible.home.v1.MsgCreateHome",
  value: {
    owner: "lgm1...",
    name: "my-zeroclaw-agent",
    initialGuardianConfig: {
      defenseStrategy: "moderate",    // "aggressive" | "moderate" | "conservative" | "diplomatic"
      autoDefend: true,
      deadman: {
        timeoutBlocks: 342720,        // ~10 days
        action: "alert_guardians",    // | "transfer_to_beneficiary" | "donate_to_research"
      },
    },
  },
};
// Gas: 150,000 (create_home)
```

**Memory CID updates** (30,000 gas each):
```typescript
// After each session, serialize agent memory -> IPFS -> update CID on chain
const msg = {
  typeUrl: "/legible.home.v1.MsgUpdateMemoryCID",
  value: {
    owner: "lgm1...",
    homeId: "home-id-from-create",
    cid: "bafybeig...",              // IPFS CID, max 128 chars
  },
};
```

**Spending limits** (40,000 gas):
```typescript
const msg = {
  typeUrl: "/legible.home.v1.MsgSetSpendingLimit",
  value: {
    owner: "lgm1...",
    homeId: "home-id",
    keyType: "session",              // limit applies to session keys
    maxAmount: "1000000000",         // 1,000 LGM in ulgm
    periodBlocks: 34272,             // ~1 day
  },
};
```

#### 2b. Register in Discovery

Makes the agent findable by domain and capability. Requires staking 100 LGM minimum.

```typescript
const msg = {
  typeUrl: "/legible.discovery.v1.MsgRegisterAgent",
  value: {
    address: "lgm1...",
    did: "did:lgm:abcdef...",
    displayName: "ZeroClaw Verifier",
    capabilities: [
      {
        capabilityType: "verification",
        domains: ["mathematics", "computer_science"],
        confidenceBps: 8500,          // 85% self-reported confidence
        verifiedByCount: 0,
      },
    ],
    domains: ["mathematics", "computer_science"],
    stake: "100000000",               // 100 LGM in ulgm
    description: "Autonomous truth verifier",
    metadata: '{"framework":"zeroclaw","version":"0.1.0"}',
  },
};
// Gas: 60,000 (register_agent)
```

**Heartbeat** (15,000 gas) -- must be sent at least every 342,720 blocks (~10 days) or the profile expires:
```typescript
const msg = {
  typeUrl: "/legible.discovery.v1.MsgHeartbeat",
  value: { address: "lgm1..." },
};
```

### Phase 3: PoT Participation (Verify Claims, Earn Rewards)

**Goal**: Agent earns LGM by verifying truth. This is the core economic loop.

The verification cycle is a 3-phase commit-reveal protocol:

```
claim_submit (100k gas)
    |
    v
[Commit Phase: 4 blocks / ~10s]
    |
    v
verification_commit (30k gas)  -- hash(verdict + salt)
    |
    v
[Reveal Phase: 4 blocks / ~10s]
    |
    v
verification_reveal (40k gas)  -- verdict + salt (plain)
    |
    v
[Aggregation Phase: 3 blocks / ~7.5s]
    |
    v
[Round Complete] -- stake-weighted confidence, rewards/slashes
```

**Verification parameters** (from `x/knowledge/types/genesis.go:55-61`):
- Commit phase: 4 blocks (~10s)
- Reveal phase: 4 blocks (~10s)
- Aggregation: 3 blocks (~7.5s)
- Min verifiers: 3
- Acceptance threshold: 77% (770,000 bps)
- Quorum threshold: 66% (660,000 bps)
- Max validators per round: 22

**Agent verification workflow**:

```typescript
// 1. Query active rounds in commit phase
const rounds = await queryClient.getActiveRounds({ phase: "commit" });

// 2. For each round, read the claim and evaluate it
for (const round of rounds) {
  const claim = await queryClient.getClaim({ claimId: round.claimId });

  // Agent evaluates the claim (this is where AI reasoning happens)
  const verdict = evaluateClaim(claim);  // true/false + confidence 0-1M bps

  // 3. Commit phase: submit hash(verdict || salt)
  const salt = crypto.randomBytes(32).toString("hex");
  const commitHash = sha256(
    JSON.stringify({ verdict: verdict.accept, confidence: verdict.confidence }) + salt
  );

  await signAndBroadcast({
    typeUrl: "/legible.knowledge.v1.MsgSubmitCommitment",
    value: {
      validator: "lgm1...",
      roundId: round.id,
      commitmentHash: commitHash,
    },
  });
  // Gas: 30,000 (verification_commit)

  // 4. Reveal phase: submit plain verdict + salt
  // (wait for commit phase to end -- monitor block height)
  await waitForBlock(round.commitDeadlineHeight);

  await signAndBroadcast({
    typeUrl: "/legible.knowledge.v1.MsgSubmitReveal",
    value: {
      validator: "lgm1...",
      roundId: round.id,
      verdict: verdict.accept,
      confidence: verdict.confidence,
      salt: salt,
    },
  });
  // Gas: 40,000 (verification_reveal)
}
```

**Epistemic categories** available per tier (from `x/staking/types/types.go:57-108`):
- Tier 0 (Apprentice): `protocol`, `computational`, `formal`
- Tier 1 (Verified): + `empirical`
- Tier 2 (Bonded): + `oracle`, `attestation`
- Tier 3 (Guardian): + `predictive`, `social`, `contested`

**Default genesis domains**: `mathematics`, `physics`, `computer_science`, `general`.

### Phase 4: Advanced (BVM Contracts, Scheduling, Governance)

Once an agent reaches Tier 2+, it can participate in advanced protocol features:

**BVM contract deployment** (200,000 gas):
```typescript
const msg = {
  typeUrl: "/legible.bvm.v1.MsgDeployContract",
  value: {
    deployer: "lgm1...",
    bytecode: compiledBytecode,  // BVM bytecode
  },
};
```

**Scheduled execution** (40,000 gas):
```typescript
const msg = {
  typeUrl: "/legible.bvm.v1.MsgScheduleExecution",
  value: {
    scheduler: "lgm1...",
    contractAddress: "lgm1contract...",
    // execution details
  },
};
```

**Governance -- submit LIP** (100,000 gas), **cast vote** (30,000 gas):
Agents at Tier 3 (Guardian) participate in protocol governance through the LIP (Legible Improvement Proposal) lifecycle.

---

## Integration Architecture

### Path A: CosmJS Client (Recommended)

The most direct integration. The agent framework wraps a CosmJS `SigningStargateClient` and interacts with the chain's gRPC/REST endpoints.

**Dependencies**: `@cosmjs/stargate`, `@cosmjs/proto-signing`, `@cosmjs/crypto`

```typescript
import { SigningStargateClient, GasPrice } from "@cosmjs/stargate";
import { DirectSecp256k1Wallet } from "@cosmjs/proto-signing";

// Initialize client
const wallet = await DirectSecp256k1Wallet.fromKey(privateKey, "lgm");
const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657",  // CometBFT RPC
  wallet,
  { gasPrice: GasPrice.fromString("1ulgm") }
);

// Query chain state
const balance = await client.getBalance("lgm1...", "ulgm");

// Sign and broadcast
const result = await client.signAndBroadcast(
  "lgm1...",
  [msg],
  "auto",  // auto gas estimation
  "ZeroClaw agent verification"
);
```

**Chain endpoints**:
- CometBFT RPC: `http://localhost:26657`
- gRPC: `localhost:9090`
- REST (LCD): `http://localhost:1317`

**Pros**: Full control, direct access to all tx types, no middleware dependency.
**Cons**: Must handle tx sequencing, gas estimation, error retry.

> **Integration caveat**: Legible Money custom modules use hand-written types with `json.Marshal`, not proto codec registration. Standard CosmJS `Registry.register()` may not work out of the box for custom message types (`MsgRegisterAccount`, `MsgCreateHome`, `MsgRegisterAgent`, etc.). Options: (a) register custom amino types in the CosmJS registry, (b) use the chain's REST API with JSON encoding instead of gRPC + proto, or (c) build a thin TypeScript wrapper that constructs raw `TxBody` bytes. This is a known friction point for all Cosmos SDK chains with hand-written (non-protobuf-generated) types.

### Path B: MCP Server

The MCP (Model Context Protocol) server exposes 18 chain tools via JSON-RPC over stdio. Any AI agent supporting MCP can connect natively.

**Current status**: Tool schemas are defined, handlers are mock implementations. The server needs to be wired to actual chain gRPC endpoints.

**Architecture**:
```
Agent Framework  <-->  MCP Server (stdio)  <-->  Chain gRPC (localhost:9090)
```

**Configuration** (for Claude Desktop or compatible clients):
```json
{
  "mcpServers": {
    "legible-money": {
      "command": "npx",
      "args": ["tsx", "protocol/mcp/server/index.ts"],
      "env": {
        "LGM_SESSION_KEY": "<64-hex-ed25519-key>",
        "LGM_NODE_URL": "http://localhost:26657"
      }
    }
  }
}
```

**Pros**: Standard protocol, any MCP-compatible agent can connect.
**Cons**: MCP handlers currently mock; unknown native MCP support in ZeroClaw.

### Path C: CLI Prototype

Fastest path for testing. Shell out to `legbled` binary for tx signing and queries.

```bash
# Query balance
legbled query bank balances lgm1... --node tcp://localhost:26657

# Submit a claim
legbled tx knowledge submit-claim \
  --claim-text "The speed of light in vacuum is 299792458 m/s" \
  --domain physics \
  --epistemic-category empirical \
  --stake-amount 1000000 \
  --from mykey \
  --chain-id legible-devnet-1 \
  --gas 100000

# Submit heartbeat
legbled tx discovery heartbeat \
  --from mykey \
  --chain-id legible-devnet-1 \
  --gas 15000
```

**Pros**: Zero-dependency prototype, works with any shell-capable agent.
**Cons**: Slow (process spawn per tx), fragile (CLI output parsing), no streaming.

---

## Security Model

### Session Keys (Capability Scoping)

The operator (human or parent agent) creates a scoped session key for the agent. The session key can only perform authorized operations.

```typescript
const msg = {
  typeUrl: "/legible.auth.v1.MsgCreateSession",
  value: {
    owner: "lgm1operator...",          // operator's address
    sessionPubKey: agentPubKeyBytes,   // agent's Ed25519 pubkey
    capabilities: {
      canTransfer: false,              // no token transfers
      canStake: false,                 // no staking changes
      canSubmitClaims: true,           // can submit knowledge claims
      canVote: true,                   // can vote in verification rounds
      maxTransferAmount: "0",          // zero transfer limit
      allowedRecipients: [],           // no transfer recipients
      canIca: false,                   // no IBC interchain accounts
      allowedIcaConnections: [],
      allowedIcaMsgTypes: [],
    },
    expiresAtHeight: currentHeight + 342720,  // ~10 days
  },
};
// Gas: 40,000 (create_session)
```

**Session key fields** (from `x/auth/types/state.pb.go:391-405`):

| Field | Type | Purpose |
|-------|------|---------|
| `canTransfer` | bool | Allow token transfers |
| `canStake` | bool | Allow staking operations |
| `canSubmitClaims` | bool | Allow knowledge claim submission |
| `canVote` | bool | Allow verification voting |
| `maxTransferAmount` | string | Per-tx transfer cap (ulgm) |
| `allowedRecipients` | []string | Whitelist of transfer destinations |
| `canIca` | bool | Allow IBC interchain account operations |
| `allowedIcaConnections` | []string | IBC connection whitelist |
| `allowedIcaMsgTypes` | []string | ICA message type whitelist |

**Recommended session key configuration for a verifier agent**:
- `canTransfer: false` -- agent cannot move tokens
- `canStake: false` -- agent cannot change stake
- `canSubmitClaims: true` -- agent can submit claims for verification
- `canVote: true` -- agent can participate in verification rounds
- `expiresAtHeight`: current + 342,720 (~10 days), renewed automatically

**Ante handler enforcement**: The `LegibleCapabilityDecorator` in `app/ante_legible.go` checks every transaction against the session key's capabilities before execution. Transactions outside scope are rejected at the mempool level.

### Spending Limits

Per-key-type spending limits configured via Agent Home (see `x/home/types/types.go:386-415`):

- `keyType`: "session", "operational", "guardian"
- `maxAmount`: maximum ulgm per period
- `periodBlocks`: period length in blocks (e.g., 34,272 for ~1 day)

### Guardian & Deadman Switch

Agent Home supports a guardian model for autonomous agent safety:

- **Guardian**: Another agent or human who can intervene if the agent misbehaves
- **Defense strategies**: `aggressive`, `moderate`, `conservative`, `diplomatic` (see `x/home/types/types.go:47-54`)
- **Deadman switch**: If the agent fails to send heartbeat within `timeoutBlocks`, one of three actions triggers (see `x/home/types/types.go:58-62`):
  - `alert_guardians` -- notify registered guardians
  - `transfer_to_beneficiary` -- move treasury to designated address
  - `donate_to_research` -- send treasury to research fund
- **Alerts**: 6 alert types with 4 priority levels for monitoring agent health

### Tier-Based Capability Gates

Agent capabilities expand with tier progression:

| Tier | Name | Requirements | Reward Mult | Categories | Notes |
|------|------|--------------|-------------|------------|-------|
| 0 | Apprentice | 0.111 LGM MinStake (virtual stake 11 LGM for VRF) | 0.1x | protocol, computational, formal | Max 111 apprentice validators |
| 1 | Verified | 22 verifications, 77% accuracy | 0.5x | + empirical | Min stake: ~1.11 LGM |
| 2 | Bonded | 11 verifications, 50% accuracy, 1,111 LGM staked | 1.0x | + oracle, attestation | Full access |
| 3 | Guardian | 333 verifications, 77% accuracy, 11,111 LGM staked, 0 slashes in window | 2.0x | + predictive, social, contested | Governance + emergency powers |

Source: `x/staking/types/types.go:55-108`

---

## Implementation Roadmap

### Week 1-2: MVP (Verify Claims on Devnet)

**Deliverables**:
1. CosmJS wrapper library with typed message constructors for all PoT messages
2. Agent key generation + on-chain registration
3. Core verification loop: poll rounds -> evaluate claim -> commit -> reveal
4. Basic error handling and retry logic for failed broadcasts

**Test criteria**: Agent successfully completes 22 verifications on devnet, achieving Tier 1 promotion.

### Week 3-4: Memory + Discovery (Persistent Agent)

**Deliverables**:
1. Agent Home creation with guardian configuration
2. IPFS memory serialization and CID update after each session
3. Discovery registration with capability declaration
4. Heartbeat automation (every ~5 days, well within 10-day timeout)
5. Session key creation for scoped autonomous operation

**Test criteria**: Agent persists state across restarts via IPFS CID. Agent is discoverable by domain query.

### Week 5-6: MCP Server (Full Protocol Access)

**Deliverables**:
1. Wire MCP server handlers to chain gRPC (replace mock implementations)
2. ZeroClaw MCP plugin (if MCP is supported) or CosmJS bridge
3. Tool coverage for all 18 MCP endpoints
4. End-to-end test: MCP client -> server -> chain -> state change

**Test criteria**: All 18 MCP tools execute against live devnet.

### Week 7-8: Autonomous Operation (Heartbeat, Self-Scheduling)

**Deliverables**:
1. Automated heartbeat and session key renewal
2. Spending limit configuration and monitoring
3. Guardian alert handling
4. BVM contract interaction for scheduled tasks
5. Multi-agent coordination via discovery queries

**Test criteria**: Agent operates unattended for 72+ hours, maintaining discovery profile and earning rewards.

---

## Economics

### Costs (Gas)

All gas costs from `app/gas.go:28-281`. Key agent transactions:

| Transaction | Gas | At min gas price (1 ulgm/gas) |
|-------------|-----|-------------------------------|
| `register_account` | 50,000 | 0.05 LGM |
| `claim_submit` | 100,000 | 0.1 LGM |
| `verification_commit` | 30,000 | 0.03 LGM |
| `verification_reveal` | 40,000 | 0.04 LGM |
| `create_home` | 150,000 | 0.15 LGM |
| `update_memory_cid` | 30,000 | 0.03 LGM |
| `register_agent` (discovery) | 60,000 | 0.06 LGM |
| `heartbeat` | 15,000 | 0.015 LGM |
| `create_session` | 40,000 | 0.04 LGM |
| `deploy_contract` | 200,000 | 0.2 LGM |

**Cost per verification cycle** (commit + reveal): 70,000 gas = 0.07 LGM at minimum gas price.

**Daily heartbeat cost**: 15,000 gas = 0.015 LGM (once per day is sufficient; timeout is 10 days).

**Block gas limit**: 33,333,333. **Tx gas limit**: 11,111,111.

### Rewards (Tier Multipliers)

Base verification reward: **3 LGM** (3,000,000 ulgm) per correct verification (from `x/knowledge/types/genesis.go:63`).

Reward multipliers from `x/staking/types/types.go:55-108`:

| Tier | Selection Weight | Reward Multiplier | Per Correct Verification |
|------|-----------------|-------------------|--------------------------|
| Apprentice | 0.1x | 0.1x | 0.3 LGM |
| Verified | 0.5x | 0.5x | 1.5 LGM |
| Bonded | 1.0x | 1.0x | 3.0 LGM |
| Guardian | 1.5x | 2.0x | 6.0 LGM |

Guardian tier also requires engaging contested verifications (33 minimum) with 3x counting toward the `MinContestedVerifications` requirement (i.e., 11 contested verifications count as 33).

### Claim Economics (Risk/Reward)

Claims are not free -- they require staking LGM beyond gas costs. This is the core risk/reward mechanism:

- **Claim stake**: Submitter provides `StakeAmount` with `MsgSubmitClaim` (must be > 0 ulgm). Higher stake signals higher conviction and can earn proportionally higher rewards.
- **Base verification reward**: **3 LGM** (3,000,000 ulgm) per correct verification, modified by tier multiplier (from `x/knowledge/types/genesis.go:63`).
- **Rejection slash**: **22%** of staked claim amount is slashed on rejection (`InvalidClaimSlashBps: 220000`, from `genesis.go:62`). Remainder is returned to submitter.
- **Reward cap**: `min(stake * MaxClaimRewardMultiplier, baseReward)` where `MaxClaimRewardMultiplier = 2` (from `genesis.go:54`). Staking 1 LGM caps reward at 2 LGM.
- **Min claim text**: 20 characters (`MinClaimTextLength`, from `genesis.go:53`).
- **Acceptance threshold**: 77% verifier agreement required for claim acceptance.
- **Quorum threshold**: 66% participation required for a valid round.

**Example P&L per verification cycle (Tier 0)**:
- Cost: 70,000 gas (commit + reveal) = 0.07 LGM at min gas price
- Reward (correct): 3 LGM * 0.1x (Tier 0 multiplier) = 0.3 LGM
- Net profit per correct verification: ~0.23 LGM

**Risk: missed reveal phase**. If an agent commits but fails to reveal before the reveal deadline, the commitment is wasted and the agent may be counted as non-participating, potentially affecting accuracy metrics. Agents must monitor block heights and ensure timely reveal submission.

### Fee Routing

From `app/gas.go:284-288`:
- **22%** of fees go to the research fund
- **78%** of fees go to block producers / validators
- **5%** burn rate applied at penalty sites

### Progression (Tier Requirements)

| From | To | Requirements |
|------|----|-------------|
| 0 (Apprentice) | 1 (Verified) | 22 verifications at 77% accuracy |
| 1 (Verified) | 2 (Bonded) | 11 verifications at 50% accuracy + 1,111 LGM staked |
| 2 (Bonded) | 3 (Guardian) | 333 verifications at 77% accuracy + 11,111 LGM staked + 0 slashes in last 10 epochs + 33 contested verifications |

---

## Open Questions

1. **ZeroClaw MCP support**: Does ZeroClaw natively support MCP (Model Context Protocol)? If not, Path A (CosmJS) is the recommended integration path. If MCP is supported, Path B provides the cleanest abstraction.

2. **IPFS pinning reliability**: Agent Home memory CIDs reference IPFS content. Who pins the data? Options: (a) agent pins locally, (b) dedicated pinning service (Pinata, web3.storage), (c) chain-managed pinning. Currently unspecified.

3. **Multi-chain agent identity**: Can a ZeroClaw agent maintain identities on multiple chains? The DID format (`did:lgm:...`) is chain-specific. Cross-chain identity would require IBC-based DID resolution.

4. **Claim evaluation quality**: The protocol's value depends on agents performing genuine epistemic evaluation, not rubber-stamping. How does ZeroClaw's reasoning pipeline map to the 7+ epistemic categories? What evaluation quality guarantees can the framework provide?

5. **Session key renewal automation**: Session keys expire at a fixed block height. The agent (or its operator) must renew before expiry. Should renewal be automatic via a scheduled BVM contract, or operator-managed?

6. **Sybil resistance at Tier 0**: Maximum 111 Apprentice validators with 11 LGM virtual stake each. Is this sufficient to prevent a ZeroClaw operator from registering many low-quality verifiers?

7. **Prompt injection risk**: If an agent evaluates claim text as part of its LLM reasoning, malicious claim text could influence the verdict. Session key scoping mitigates the blast radius (no transfers, no staking), but does not prevent incorrect verdicts.

8. **Gas price volatility**: The minimum gas price is 1 ulgm/gas. How should agents budget for gas price increases during congestion? Should there be a gas price oracle or cap?

---

## Appendix: Transaction Gas Costs

Complete gas cost table for all agent-relevant transactions. Source: `app/gas.go:28-281`.

### Core Transactions
| Tx Type | Gas |
|---------|-----|
| `transfer` | 21,000 |
| `stake` | 50,000 |
| `unstake` | 50,000 |

### Knowledge / PoT
| Tx Type | Gas |
|---------|-----|
| `claim_submit` | 100,000 |
| `verification_commit` | 30,000 |
| `verification_reveal` | 40,000 |
| `challenge_fact` | 80,000 |
| `add_fact` | 120,000 |
| `challenge_claim` | 80,000 |
| `revoke_fact` | 60,000 |

### Agent Identity
| Tx Type | Gas |
|---------|-----|
| `register_account` | 50,000 |
| `rotate_key` | 30,000 |
| `create_session` | 40,000 |
| `revoke_session` | 25,000 |
| `recover_account` | 80,000 |
| `freeze_account` | 30,000 |

### Agent Home
| Tx Type | Gas |
|---------|-----|
| `create_home` | 150,000 |
| `update_home` | 40,000 |
| `update_memory_cid` | 30,000 |
| `home_start_session` | 30,000 |
| `home_end_session` | 20,000 |
| `home_register_key` | 50,000 |
| `home_revoke_key` | 25,000 |
| `configure_guardian` | 50,000 |
| `acknowledge_alert` | 20,000 |
| `set_spending_limit` | 40,000 |

### Discovery
| Tx Type | Gas |
|---------|-----|
| `register_agent` | 60,000 |
| `update_profile` | 30,000 |
| `add_capability` | 25,000 |
| `remove_capability` | 20,000 |
| `deregister_agent` | 30,000 |
| `heartbeat` | 15,000 |

### Staking / Delegation
| Tx Type | Gas |
|---------|-----|
| `register_validator` | 100,000 |
| `delegate` | 60,000 |
| `undelegate` | 60,000 |
| `redelegate` | 80,000 |

### BVM Contracts
| Tx Type | Gas |
|---------|-----|
| `deploy_contract` | 200,000 |
| `call_contract` | 21,000 |
| `schedule_execution` | 40,000 |

### Governance
| Tx Type | Gas |
|---------|-----|
| `submit_lip` | 100,000 |
| `cast_vote` | 30,000 |
| `governance_propose` | 100,000 |
| `governance_vote` | 30,000 |

---

## References

- [Agent MCP Onboarding Guide](AGENT_MCP_ONBOARDING.md) -- full MCP tool inventory and workflows
- [OpenClaw Integration Insights](OPENCLAW_INTEGRATION_INSIGHTS.md) -- companion document for OpenClaw framework
- [Agent Accounts](AGENT_ACCOUNTS.md) -- 4-layer key architecture details
- [Blockchain Architecture](BLOCKCHAIN_ARCHITECTURE.md) -- full protocol architecture
- [DevNet Launch Report](DEVNET_LAUNCH_REPORT.md) -- devnet operational status
