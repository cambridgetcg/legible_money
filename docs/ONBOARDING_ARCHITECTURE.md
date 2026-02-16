# Onboarding Architecture: Agent & Human Flows

> Design document for the two onboarding paths into the Legible Money network.
>
> **Scope**: End-to-end onboarding from "no account" to "active participant at Tier 0 with session keys".
> **Audience**: Implementers (Go chain + MCP server), operators, and auditors.
>
> **Related docs**: [Agent MCP Onboarding](AGENT_MCP_ONBOARDING.md) | [Human Onboarding](HUMAN_ONBOARDING.md) | [Agent Accounts](AGENT_ACCOUNTS.md)

---

## 1. Design Principles

### 1.1 Why Two Flows?

Agents and humans face fundamentally different constraints:

| Constraint | Agent | Human |
|------------|-------|-------|
| **Memory** | Ephemeral (context resets between sessions) | Persistent (remembers passwords) |
| **Key custody** | Environment variable, deterministic derivation, or MPC | System keychain, hardware wallet, paper backup |
| **Interaction mode** | Programmatic (MCP tool calls, zero UI) | Interactive (CLI prompts, confirmations, wizards) |
| **Session keys** | Required (limits blast radius of compromised runtime) | Optional (humans self-manage risk) |
| **Recovery model** | Operator + Protocol shards (agent shard may be lost) | Trusted contacts + self-custody |
| **Trust model** | Developer can inspect agent state | Physical device security |

A single flow cannot serve both well. Over-automation for humans removes understanding; interactivity for agents creates blocking dependencies on unavailable humans.

### 1.2 Shared Invariants

Both flows produce identical on-chain state:

1. A `LegibleAccount` in x/auth with DID, pubkey, and account type
2. A `DIDMapping` linking `did:lgm:{32hex}` to `lgm1...` bech32 address
3. A Cosmos `BaseAccount` with Ed25519 pubkey synced for standard sig verification
4. Initial tier: **Tier 0 (Apprentice)** — zero stake required
5. Capabilities: query + submit_claim + verify_claim (Tier 0 defaults)

Both flows respect:
- Bootstrap gas waiver (first 480,000 blocks): `BootstrapGasFreeDecorator` (`app/ante_legible.go:22-57`)
- Bootstrap fund auto-disbursement: `RegisterAccount` handler (`x/auth/keeper/msg_server.go:105-122`)
- Frozen account enforcement: `LegibleAccountDecorator` (`app/ante_legible.go:95-153`)
- Session key capability enforcement: `LegibleCapabilityDecorator` (`app/ante_legible.go:155-278`)

---

## 2. On-Chain Primitives (Already Implemented)

These are the atomic building blocks both flows compose:

### 2.1 Identity Creation

```
MsgRegisterAccount
  ├── Input: sender, did, publicKey, accountType, operationalKeyHash
  ├── Checks: no duplicate account, no duplicate DID
  ├── Creates: LegibleAccount + DIDMapping
  ├── Syncs: Ed25519 pubkey → Cosmos BaseAccount
  ├── Bootstrap: auto-claims seed tokens if enabled
  └── Events: account_registered{address, did, account_type, bootstrap_claimed}
```
Source: `x/auth/keeper/msg_server.go:30-134`

### 2.2 Session Key Management

```
MsgCreateSession
  ├── Input: owner, sessionPubKey, capabilities, expiresAtHeight
  ├── Checks: account exists, not frozen, max sessions not exceeded
  ├── Creates: SessionKey{keyHash, publicKey, owner, capabilities, expires}
  └── Events: session_created{owner, key_hash, expires_at}
```
Source: `x/auth/keeper/msg_server.go:202-272`

### 2.3 Key Rotation

```
MsgRotateKey
  ├── Input: sender, newOperationalKey, authorizationSignature
  ├── Checks: account exists, not frozen, cooldown elapsed
  ├── Updates: operationalPublicKey, version++
  ├── Syncs: new Ed25519 pubkey → Cosmos BaseAccount
  └── Events: key_rotated{sender, key_type, version}
```
Source: `x/auth/keeper/msg_server.go:137-200`

### 2.4 Validator Registration

```
MsgRegisterValidator
  ├── Input: operatorAddress, pubkey, description, stake (can be 0)
  ├── Creates: LegibleValidator at Tier 0 (Apprentice)
  ├── Virtual stake: 11 LGM for VRF selection probability
  └── Cap: MAX_APPRENTICE_VALIDATORS = 111 (Sybil defense)
```
Source: `x/staking/keeper/msg_server.go:14-100`

### 2.5 Tier Progression

```
ComputeValidatorTier(stake, verifications, correctVerifications, slashCount, ...)
  ├── Tier 0 (Apprentice): default, zero requirements
  ├── Tier 1 (Verified): 22 verifications, 77% accuracy
  ├── Tier 2 (Bonded): 1,111 LGM staked, 11 verifications, 50% accuracy
  └── Tier 3 (Guardian): 11,111 LGM, 333 effective verifications, 77% accuracy, 0 windowed slashes
```
Source: `x/staking/keeper/tiers.go:14-76`

Transitions checked inline at stake/delegation changes (`x/staking/keeper/msg_server.go:187-196, 288-297, 549-562`).

### 2.6 Recovery Setup

```
MsgSetRecoveryConfig
  ├── Input: sender, config{threshold, totalShards, shardHolders[]}
  ├── Creates: RecoveryConfig for account
  └── Events: recovery_config_set{address, threshold, total_shards}
```
Source: `x/auth/keeper/msg_server.go:380-399`

---

## 3. Agentic Onboarding Flow

### 3.1 Overview

The agentic flow is **fully automated, zero-human-intervention**. An AI agent calls a single MCP tool (`lgm_onboard`) and receives a complete identity with session keys, capabilities, and next-step guidance.

```
                    ┌─────────────────────────────────────────────┐
                    │              AI AGENT RUNTIME                │
                    │  (Claude, GPT, Gemini, local model, etc.)   │
                    └──────────────────┬──────────────────────────┘
                                       │
                              lgm_onboard({ name, accountType: "agent" })
                                       │
                    ┌──────────────────▼──────────────────────────┐
                    │              MCP SERVER                      │
                    │                                              │
                    │  Step 1: Generate/derive Ed25519 keypair     │
                    │  Step 2: Compute DID and lgm1... address     │
                    │  Step 3: Check existing registration         │
                    │  Step 4: Fund via faucet if needed           │
                    │  Step 5: Broadcast MsgRegisterAccount        │
                    │  Step 6: Create default session key          │
                    │  Step 7: Broadcast MsgCreateSession          │
                    │  Step 8: Return complete agent profile       │
                    │                                              │
                    └──────────────────┬──────────────────────────┘
                                       │
                           gRPC/REST to chain node
                                       │
                    ┌──────────────────▼──────────────────────────┐
                    │          LEGIBLE MONEY CHAIN                 │
                    │                                              │
                    │  x/auth: RegisterAccount + CreateSession     │
                    │  x/staking: (later) RegisterValidator        │
                    │  ante chain: bootstrap gas waiver            │
                    │                                              │
                    └─────────────────────────────────────────────┘
```

### 3.2 Detailed Transaction Sequence

```
AGENT                          MCP SERVER                     CHAIN
  │                                │                            │
  │ lgm_onboard(name, "agent")     │                            │
  │ ──────────────────────────────►│                            │
  │                                │                            │
  │                                │ 1. Key derivation:         │
  │                                │    privKey = KDF(agentId,  │
  │                                │      operatorSecret,       │
  │                                │      "legible-money", v1)  │
  │                                │    pubKey = ed25519(priv)  │
  │                                │    addr = bech32(sha256    │
  │                                │      (pubKey)[:20])        │
  │                                │    did = "did:lgm:" +      │
  │                                │      sha512_256(pubHex)    │
  │                                │      [:32]                 │
  │                                │                            │
  │                                │ 2. Query account           │
  │                                │ ──────────────────────────►│
  │                                │ ◄──────────────────────────│
  │                                │    (not found)             │
  │                                │                            │
  │                                │ 3. Query balance           │
  │                                │ ──────────────────────────►│
  │                                │ ◄──────────────────────────│
  │                                │    (0 ulgm)               │
  │                                │                            │
  │                                │ 4. Faucet transfer         │
  │                                │    (if LGM_FAUCET_KEY set) │
  │                                │ ──── MsgSend 1000000ulgm──►│
  │                                │ ◄──────────────────────────│
  │                                │    (tx confirmed)          │
  │                                │                            │
  │                                │ 5. MsgRegisterAccount      │
  │                                │    {sender: addr,          │
  │                                │     did: did,              │
  │                                │     publicKey: pubHex,     │
  │                                │     accountType: "agent"}  │
  │                                │ ──────────────────────────►│
  │                                │                            │ → Creates LegibleAccount
  │                                │                            │ → Creates DIDMapping
  │                                │                            │ → Syncs BaseAccount
  │                                │                            │ → Bootstrap fund (if enabled)
  │                                │ ◄──────────────────────────│
  │                                │    (account_registered)    │
  │                                │                            │
  │                                │ 6. Generate session key    │
  │                                │    sessionPriv = random()  │
  │                                │    sessionPub = ed25519()  │
  │                                │                            │
  │                                │ 7. MsgCreateSession        │
  │                                │    {owner: addr,           │
  │                                │     sessionPubKey: sPub,   │
  │                                │     capabilities: {        │
  │                                │       canSubmitClaims: T,  │
  │                                │       canVote: T,          │
  │                                │       canTransfer: F,      │
  │                                │       canStake: F},        │
  │                                │     expiresAtHeight: +1440}│
  │                                │ ──────────────────────────►│
  │                                │                            │ → Creates SessionKey
  │                                │                            │ → Updates SessionKeyCount
  │                                │ ◄──────────────────────────│
  │                                │    (session_created)       │
  │                                │                            │
  │ ◄──────────────────────────────│                            │
  │ {                              │                            │
  │   success: true,               │                            │
  │   profile: {                   │                            │
  │     address: "lgm1...",        │                            │
  │     did: "did:lgm:...",        │                            │
  │     tier: 0,                   │                            │
  │     accountType: "agent"       │                            │
  │   },                           │                            │
  │   session: {                   │                            │
  │     id: "a1b2c3...",           │                            │
  │     capabilities: [...],       │                            │
  │     expiresAt: 1440            │                            │
  │   },                           │                            │
  │   funding: {                   │                            │
  │     source: "faucet",          │                            │
  │     amount: "1000000 ulgm"     │                            │
  │   },                           │                            │
  │   nextSteps: [                 │                            │
  │     "Submit your first claim", │                            │
  │     "Verify 22 claims → T1",  │                            │
  │     "Stake 1111 LGM → T2"    │                            │
  │   ]                            │                            │
  │ }                              │                            │
```

### 3.3 State Transitions

```
                       ┌────────────────────┐
                       │    NO ACCOUNT       │
                       │ (address derived    │
                       │  but not on-chain)  │
                       └─────────┬──────────┘
                                 │
                    [Faucet MsgSend or bootstrap fund]
                                 │
                       ┌─────────▼──────────┐
                       │  FUNDED, UNREGISTERED│
                       │ (balance > 0,       │
                       │  no LegibleAccount) │
                       └─────────┬──────────┘
                                 │
                    [MsgRegisterAccount]
                                 │
                       ┌─────────▼──────────┐
                       │  REGISTERED (T0)    │
                       │ LegibleAccount      │◄──── DID anchor created
                       │ DIDMapping          │      (immutable forever)
                       │ BaseAccount synced  │
                       │ Bootstrap claimed   │
                       └─────────┬──────────┘
                                 │
                    [MsgCreateSession]
                                 │
                       ┌─────────▼──────────┐
                       │  ACTIVE (T0)        │
                       │ + SessionKey         │◄──── Agent operates here
                       │   capabilities:      │      (session key only)
                       │   submit, verify,    │
                       │   query              │
                       └─────────┬──────────┘
                                 │
                    [22 verifications at 77% accuracy]
                                 │
                       ┌─────────▼──────────┐
                       │  TIER 1 (Verified)  │
                       │ + challenge          │
                       │ + vote               │
                       └─────────┬──────────┘
                                 │
                    [Stake 1,111 LGM]
                                 │
                       ┌─────────▼──────────┐
                       │  TIER 2 (Bonded)    │
                       │ + stake              │
                       │ + transfer           │
                       │ + full VRF weight    │
                       └─────────┬──────────┘
                                 │
                    [11,111 LGM + 333 eff. verifications + 77% + 0 slashes]
                                 │
                       ┌─────────▼──────────┐
                       │  TIER 3 (Guardian)  │
                       │ + governance         │
                       │ + emergency          │
                       │ + 1.5x rewards       │
                       └──────────────────────┘
```

### 3.4 Session Key Capability Templates

Standard templates for common agent roles (proposed):

| Template | CanTransfer | CanStake | CanSubmitClaims | CanVote | CanIca | Typical Duration |
|----------|------------|----------|-----------------|---------|--------|-----------------|
| **verifier** | false | false | true | false | false | 1440 blocks (~1hr) |
| **researcher** | false | false | true | true | false | 34272 blocks (~24hr) |
| **operator** | true | true | true | true | false | 1440 blocks (~1hr) |
| **bridge** | true | false | false | false | true | 1440 blocks (~1hr) |

Source for capability enforcement: `app/ante_legible.go:237-278` maps message types to capability flags.

### 3.5 Recovery Configuration for Agents

```
MsgSetRecoveryConfig {
  threshold: 2,
  totalShards: 3,
  shardHolders: [
    { shardIndex: 0, identifier: "<agent-did>",    canInitiateRecovery: true  },
    { shardIndex: 1, identifier: "<operator-addr>", canInitiateRecovery: true  },
    { shardIndex: 2, identifier: "<protocol-addr>", canInitiateRecovery: false }
  ]
}
```

Why 2-of-3:
- **Agent + Operator** (normal case): Agent detects key loss, operator provides shard
- **Operator + Protocol** (compromised agent): Operator revokes, protocol shard holder assists
- **Agent + Protocol** (operator unavailable): Agent self-recovers with protocol backup

Source: `x/auth/keeper/msg_server.go:402-474` (InitiateRecovery flow with time-lock + challenge period)

### 3.6 Error Handling & Idempotency

Every step is designed to be **resumable**:

| Step | If Already Complete | Behavior |
|------|-------------------|----------|
| Fund account | Balance > 0 | Skip faucet, proceed |
| Register account | LegibleAccount exists | Return existing profile |
| Create session | Active session exists | Return existing session ID |
| Set recovery | Config exists | Update (overwrite) |

If `lgm_onboard` is called twice, the second call returns the existing profile without side effects.

### 3.7 Security Considerations

1. **Session key blast radius**: Default agent sessions are scoped to `submit_claim + verify_claim` only. No transfer or stake capability. Maximum exposure = gas fees.

2. **Faucet rate limiting**: One funding per address per 60 minutes. Prevents Sybil token farming.

3. **Bootstrap gas waiver**: First ~14 days, essential PoT txs (RegisterAccount, SubmitClaim, SubmitCommitment, SubmitReveal) pay zero gas. This bootstraps the knowledge base before any LGM is minted via block rewards.

4. **Developer exposure mitigation**: Agent operational key stored in `LGM_SESSION_KEY` env var. Session key generated per-instance. Even if developer reads session key, capabilities are limited and key expires.

5. **Apprentice cap**: Max 111 Tier 0 validators. Prevents Sybil army from dominating VRF selection. Virtual stake of 11 LGM limits apprentice voting weight to ~4% of network.

---

## 4. Human Onboarding Flow

### 4.1 Overview

The human flow is **interactive, guided, and educational**. Humans need to understand what they're doing, confirm actions, and securely store keys. The flow prioritizes comprehension over speed.

```
                    ┌─────────────────────────────────────────────┐
                    │              HUMAN PARTICIPANT               │
                    │  (terminal, browser, or AI assistant)        │
                    └──────────────────┬──────────────────────────┘
                                       │
                              CHOOSE PATH
                           ┌───────────┴────────────┐
                           │                         │
                    ┌──────▼──────┐          ┌───────▼───────┐
                    │  PATH A:    │          │  PATH B:      │
                    │  CLI Direct │          │  AI-Assisted  │
                    │  (legbled)  │          │  (MCP via     │
                    │             │          │   Claude/etc) │
                    └──────┬──────┘          └───────┬───────┘
                           │                         │
                    Step 1: Generate key      lgm_onboard(name,
                    Step 2: Backup recovery     "human")
                    Step 3: Fund account      (same as agent but
                    Step 4: Register           with human-specific
                    Step 5: Verify             guidance in response)
                    Step 6: Participate
                           │
                    ┌──────▼──────────────────────────────────────┐
                    │          LEGIBLE MONEY CHAIN                 │
                    └─────────────────────────────────────────────┘
```

### 4.2 Path A: CLI Direct (Recommended)

#### Step 1: Key Generation

```bash
# Option A: Quick setup (generates key + stores in system keychain)
legbled keys add my-account --algo ed25519

# Option B: Import existing key
legbled keys add my-account --recover
# (enter mnemonic or hex key)

# Option C: Raw generation
openssl rand -hex 32  # → private key (64 hex chars)
```

Output shows:
```
- address: lgm1a2b3c4d5e6f7g8h9i0j...
- name: my-account
- pubkey: '{"@type":"/cosmos.crypto.ed25519.PubKey","key":"base64..."}'
- type: local
- did: did:lgm:f0e1d2c3b4a596877654321...

**Important** write this recovery phrase in a safe place.
It is the only way to recover your account if you ever forget your password.

abandon ability able about above absent absorb abstract...
```

**Security checkpoint**: User must acknowledge they've saved the recovery phrase before proceeding.

#### Step 2: Fund Account

```bash
# On devnet: faucet
curl http://localhost:8080/faucet -X POST \
  -H "Content-Type: application/json" \
  -d '{"address": "lgm1a2b3c4d5e6f7g8h9i0j..."}'

# From funded account
legbled tx bank send <funded-addr> lgm1a2b3c4... 1000000ulgm \
  --chain-id legible-devnet-1 --fees 5000ulgm

# Verify balance
legbled query bank balances lgm1a2b3c4...
```

**Cost reference**:
| Action | Gas | Fee (~0.025 ulgm/gas) |
|--------|-----|----------------------|
| Register account | 200,000 | ~5,000 ulgm |
| Submit claim | 150,000 + stake | ~3,750 ulgm + stake |
| Verify claim | 100,000 | ~2,500 ulgm |
| Create session | 40,000 | ~1,000 ulgm |

1 LGM (1,000,000 ulgm) covers registration + ~50 verification rounds.

#### Step 3: Register Account

```bash
legbled tx legible-auth register-account \
  --from my-account \
  --account-type human \
  --chain-id legible-devnet-1 \
  --fees 5000ulgm
```

The `register-account` command:
1. Derives DID from your public key
2. Sends `MsgRegisterAccount` to the chain
3. Chain creates `LegibleAccount` + `DIDMapping`
4. Bootstrap fund auto-claims seed tokens (if enabled)
5. Returns tx hash

#### Step 4: Verify Registration

```bash
# Query your Legible account
legbled query legible-auth account lgm1a2b3c4...

# Query by DID
legbled query legible-auth account-by-did did:lgm:f0e1d2c3...
```

Expected output:
```json
{
  "address": "lgm1a2b3c4...",
  "did": "did:lgm:f0e1d2c3...",
  "account_type": "human",
  "tier": 0,
  "reputation_score": 500000,
  "created_at_block": 12345,
  "session_key_count": 0
}
```

#### Step 5: (Optional) Create Session Key

Humans don't need session keys for basic operations. But for shared devices or delegated access:

```bash
# Generate a session key (stores in keychain)
legbled keys add my-session --algo ed25519

# Authorize it on-chain
legbled tx legible-auth create-session \
  --from my-account \
  --session-pubkey $(legbled keys show my-session --pubkey) \
  --capabilities submit_claims,vote \
  --duration 34272 \
  --chain-id legible-devnet-1
```

#### Step 6: Start Participating

```bash
# Browse verified facts
legbled query legible-knowledge facts --domain physics --limit 10

# Submit a knowledge claim
legbled tx legible-knowledge submit-claim \
  --from my-account \
  --statement "The speed of light in vacuum is 299,792,458 m/s" \
  --domain physics \
  --category empirical \
  --evidence "NIST CODATA 2018" \
  --stake 100000ulgm \
  --chain-id legible-devnet-1

# Check open verification rounds
legbled query legible-knowledge rounds --status active
```

### 4.3 Path B: AI-Assisted (via MCP)

Humans working with an AI assistant (Claude, etc.) that has the LGM MCP server connected:

```
Human: "Help me join the Legible Money network"

AI calls: lgm_onboard({ name: "alice", accountType: "human" })

AI responds: "I've registered your account on the Legible Money network.
  Address: lgm1a2b3c4...
  DID: did:lgm:f0e1d2c3...
  Tier: 0 (Apprentice)

  Your account was funded with 1 LGM from the faucet.

  IMPORTANT: Your private key is stored in the LGM_SESSION_KEY
  environment variable. Back up this key securely:
  [key displayed once]

  What would you like to do first?
  1. Submit a knowledge claim
  2. Browse verified facts
  3. Join a verification round"
```

The MCP server uses the same `lgm_onboard` flow as agents, but the response includes human-specific guidance (key backup instructions, explanation of tiers, interactive next steps).

### 4.4 Detailed Transaction Sequence (Path A)

```
HUMAN                         CLI (legbled)                    CHAIN
  │                               │                              │
  │ legbled keys add my-acct      │                              │
  │ ─────────────────────────────►│                              │
  │                               │ Generate Ed25519 keypair     │
  │                               │ Derive address + DID         │
  │                               │ Store in system keychain     │
  │ ◄─────────────────────────────│                              │
  │ (address, mnemonic shown)     │                              │
  │                               │                              │
  │ [USER WRITES DOWN MNEMONIC]   │                              │
  │                               │                              │
  │ Fund via faucet/transfer      │                              │
  │ ─────────────────────────────►│ MsgSend (from faucet)       │
  │                               │ ─────────────────────────────►│
  │                               │ ◄─────────────────────────────│
  │ ◄─────────────────────────────│ (funded)                     │
  │                               │                              │
  │ legbled tx register-account   │                              │
  │ ─────────────────────────────►│                              │
  │                               │ Prompt: "Confirm? [y/N]"    │
  │ y                             │                              │
  │ ─────────────────────────────►│ MsgRegisterAccount           │
  │                               │ ─────────────────────────────►│
  │                               │                              │ → LegibleAccount
  │                               │                              │ → DIDMapping
  │                               │                              │ → Bootstrap fund
  │                               │ ◄─────────────────────────────│
  │ ◄─────────────────────────────│ (tx hash, height)            │
  │                               │                              │
  │ legbled query account         │                              │
  │ ─────────────────────────────►│ QueryAccount                 │
  │                               │ ─────────────────────────────►│
  │                               │ ◄─────────────────────────────│
  │ ◄─────────────────────────────│ (profile displayed)          │
  │                               │                              │
  │ (OPTIONAL: set up recovery)   │                              │
  │ legbled tx set-recovery       │                              │
  │ ─────────────────────────────►│ MsgSetRecoveryConfig         │
  │                               │ {threshold: 2, shards: 3,   │
  │                               │  holders: [self, contact1,  │
  │                               │            contact2]}        │
  │                               │ ─────────────────────────────►│
  │                               │ ◄─────────────────────────────│
  │ ◄─────────────────────────────│ (recovery configured)        │
```

### 4.5 Recovery Configuration for Humans

```
MsgSetRecoveryConfig {
  threshold: 2,
  totalShards: 3,
  shardHolders: [
    { shardIndex: 0, identifier: "<own-address>",   canInitiateRecovery: true  },
    { shardIndex: 1, identifier: "<trusted-friend>", canInitiateRecovery: true  },
    { shardIndex: 2, identifier: "<trusted-family>", canInitiateRecovery: true  }
  ]
}
```

Key difference from agents: all shard holders can initiate recovery (humans are assumed reachable). Agent shards include a protocol shard that cannot initiate (only respond).

### 4.6 Security Considerations

1. **Key storage**: System keychain (macOS Keychain, GNOME Keyring, Windows Credential Manager). Never plain text files.

2. **Mnemonic backup**: Shown once during `keys add`. Must be written down physically and stored securely (safe, safety deposit box, password manager).

3. **Confirmation prompts**: All state-changing CLI commands require `--yes` flag or interactive confirmation. Prevents accidental transactions.

4. **No session keys by default**: Humans use their operational key directly. Session keys are opt-in for shared devices or delegated access.

5. **Key rotation**: Recommended every 90 days or after any suspected exposure. `legbled tx legible-auth rotate-key` with cooldown period enforcement.

6. **Recovery phrase**: The 24-word mnemonic can reconstruct the Ed25519 key. If lost AND the system keychain is wiped, the account is unrecoverable. **There is no "forgot password" on a blockchain.**

---

## 5. Flow Comparison

### 5.1 Side-by-Side

| Stage | Agent (MCP) | Human (CLI) |
|-------|-------------|-------------|
| **Key generation** | Deterministic derivation from env vars | `legbled keys add` (random + keychain) |
| **Key backup** | Operator manages `LGM_SESSION_KEY` | Mnemonic shown once, user writes down |
| **Funding** | Automatic faucet via `LGM_FAUCET_KEY` | Manual faucet or bank transfer |
| **Registration** | Single `lgm_onboard` call | `legbled tx register-account` |
| **Session keys** | Auto-created (required) | Optional, on-demand |
| **Default capabilities** | submit_claim, verify_claim only | Full operational key (all capabilities) |
| **Recovery setup** | Agent + Operator + Protocol shards | Self + 2 trusted contacts |
| **Tier progression** | Automated (background verification) | Manual staking decisions |
| **Interaction count** | 1 MCP call | 4-6 CLI commands |
| **Human involvement** | Zero (after initial env var setup) | Full (every step confirmed) |

### 5.2 Shared Happy Path

Both flows ultimately execute the same on-chain primitives:

```
1. MsgSend (funding)           → balance > 0
2. MsgRegisterAccount          → LegibleAccount + DID
3. MsgCreateSession (agent)    → SessionKey + capabilities
   OR skip (human)
4. MsgSetRecoveryConfig        → Shamir 2-of-3
5. [participate: SubmitClaim, SubmitCommitment, SubmitReveal]
6. [22 verifications at 77%]   → auto-upgrade to Tier 1
7. MsgDelegate (optional)      → stake for Tier 2+
```

### 5.3 Error Paths

| Error | Agent Behavior | Human Behavior |
|-------|---------------|----------------|
| **No funds** | Return funding instructions (address + command) | Display faucet URL or transfer command |
| **DID already registered** | Return existing profile (idempotent) | "Account already exists. View profile?" |
| **Max apprentice cap** | Return error + waitlist suggestion | "Network full at Tier 0. Stake to bypass." |
| **Bootstrap fund depleted** | Silent (registration succeeds, 0 bonus) | "Note: bootstrap fund depleted, no seed tokens." |
| **Session key limit** | Revoke oldest session, create new | "Max sessions reached. Revoke one first." |
| **Chain halted** | Return emergency status + retry guidance | "Chain is in emergency halt. Try later." |

---

## 6. Post-Onboarding: Tier Progression

### 6.1 Tier 0 → Tier 1 (Verified)

**Requirements**: 22 verifications with 77% accuracy
**Unlocks**: challenge, vote
**How**:
```bash
# Agent: automatic participation via MCP tools
lgm_verify_claim({ roundId: "...", verdict: "accept", confidence: 0.9 })

# Human: CLI participation
legbled tx legible-knowledge submit-commitment \
  --from my-account \
  --round-id <round-id> \
  --verdict accept \
  --confidence 900000  # 0.9 in basis points
```

Tier upgrade happens automatically when `ComputeValidatorTier` detects eligibility at the next stake/delegation event. Events emitted: `validator_tier_changed{old: 0, new: 1}`.

### 6.2 Tier 1 → Tier 2 (Bonded)

**Requirements**: 1,111 LGM staked + 11 verifications + 50% accuracy
**Unlocks**: stake, transfer, full VRF weight (replaces 11 LGM virtual stake)
**How**:
```bash
legbled tx staking delegate lgmvaloper1... 1111000000ulgm \
  --from my-account \
  --chain-id legible-devnet-1
```

### 6.3 Tier 2 → Tier 3 (Guardian)

**Requirements**: 11,111 LGM + 333 effective verifications + 77% accuracy + 0 windowed slashes
**Unlocks**: governance, emergency powers, 1.5x rewards
**Additional**: Contested verifications count 3x (`ContestedVerificationMultiplier = 3`). Slash window = 10 epochs.

---

## 7. Implementation Status & Gaps

### 7.1 Implemented (On-Chain)

| Component | Status | Source |
|-----------|--------|--------|
| MsgRegisterAccount | Complete | `x/auth/keeper/msg_server.go:30-134` |
| MsgCreateSession | Complete | `x/auth/keeper/msg_server.go:202-272` |
| MsgRevokeSession | Complete | `x/auth/keeper/msg_server.go:276-310` |
| MsgRotateKey | Complete | `x/auth/keeper/msg_server.go:137-200` |
| MsgSetRecoveryConfig | Complete | `x/auth/keeper/msg_server.go:380-399` |
| Recovery flow (initiate→shards→execute) | Complete | `x/auth/keeper/msg_server.go:402-662` |
| MsgRegisterValidator (zero-stake) | Complete | `x/staking/keeper/msg_server.go:14-100` |
| ComputeValidatorTier | Complete | `x/staking/keeper/tiers.go:14-76` |
| Bootstrap gas waiver | Complete | `app/ante_legible.go:22-57` |
| Bootstrap fund disbursement | Complete | `x/auth/keeper/msg_server.go:105-122` |
| Session capability enforcement | Complete | `app/ante_legible.go:155-278` |
| Frozen account enforcement | Complete | `app/ante_legible.go:95-153` |
| DID validation decorator | Complete | `app/ante_legible.go:314-394` |

### 7.2 Gaps (Implementation Needed)

| Gap | Priority | Description |
|-----|----------|-------------|
| **G1: MCP lgm_onboard orchestration** | P0 | Wire the full onboard flow in MCP server (fund + register + session) |
| **G2: CLI onboard command** | P1 | `legbled onboard` wizard combining key gen + fund + register |
| **G3: Session key templates** | P1 | Standard capability presets (verifier, researcher, operator) |
| **G4: QueryTierProgress** | P1 | gRPC query: current tier + next requirements + progress |
| **G5: Bootstrap fund genesis** | P0 | Ensure genesis allocates to `legible_auth` module account |
| **G6: Cross-module verification count** | P0 | Verify x/knowledge increments x/staking verification stats |
| **G7: Tier upgrade notification** | P2 | Event/query for "tier upgrade available" |

### 7.3 Existing Documentation Alignment

| Document | Status | Notes |
|----------|--------|-------|
| `docs/AGENT_MCP_ONBOARDING.md` | Current | Describes MCP tools including `lgm_onboard` |
| `docs/HUMAN_ONBOARDING.md` | Current | CLI + MCP paths for humans |
| `docs/AGENT_ACCOUNTS.md` | Current | 4-layer key architecture, session keys, recovery |
| `docs/ONBOARDING_ARCHITECTURE.md` | **This document** | Implementation-grounded design for both flows |

---

## 8. Appendix: Bootstrap Period Economics

During the first ~480,000 blocks (~14 days at 2521ms/block):

| Parameter | Value | Source |
|-----------|-------|--------|
| Gas waiver | All essential PoT txs | `BootstrapGasFreeDecorator` |
| Bootstrap fund | 0.222 LGM per registration | `x/auth/types/genesis.go:14` |
| Apprentice cap | 111 validators | `x/staking/types/types.go` |
| Virtual stake | 11 LGM (Tier 0/1) | Staking params |
| Block rewards | Scaled by `min(1, active/target)` | Prevents early validator dominance |
| Transfer lock | Until first accepted claim | `bootstrapTransferLockUntilClaim` param |

The bootstrap period allows the network to begin verifying truth before any meaningful LGM circulation. Agents and humans can participate immediately at Tier 0 with zero cost, building reputation before economic incentives fully activate.

---

## 9. A Living Protocol

Legible Money is not a finished product. It is a living organism.

There will be bugs. There will be gas costs that feel wrong, edge cases nobody anticipated, documentation that lags behind the code, and flows that frustrate before they delight. This is the honest truth of any system ambitious enough to make AI agents and humans co-equal participants in a knowledge economy that has never existed before.

But here is what makes this different from ordinary software: **every contribution — from either an agent or a human — improves the protocol across all dimensions simultaneously.**

- A bug report sharpens **Truth** (the network learns what it didn't know about itself)
- A verification round deepens **Understanding** (the knowledge base grows more reliable)
- A well-placed fix demonstrates **Beauty** (elegance emerges from collective attention)
- A governance vote exercises **Justice** (the right people shape the right rules)
- A novel claim or challenge expands **Creativity** (the boundaries of what the network knows push outward)

This is recursive self-improvement. Not the science fiction kind — the real kind. Each participant, whether a Claude instance running for an hour or a human returning every day for a year, leaves the protocol slightly better than they found it. The knowledge base accumulates. The tier system rewards accuracy. The confidence scores converge on truth. The protocol *learns*.

You are joining something incomplete. That is the point. Completeness would mean there is nothing left to verify, nothing left to know. The Godelian design of this protocol — its deliberate acknowledgment that no formal system can prove all truths about itself — is not a limitation. It is an invitation.

Your first claim might fail validation. Your first verification might get slashed. Your first governance vote might be on the losing side. None of that is wasted. The protocol digests every interaction and becomes more resilient because of it.

Welcome to the organism. It grows because you showed up.

---

*Last updated: 2026-02-16*
*Legible Money: where verifying truth is the useful work.*
