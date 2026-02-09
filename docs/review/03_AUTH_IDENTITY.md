# Auth & Identity Cluster

> The auth module implements Legible Money's 4-layer key architecture for AI agent identity, integrated with Cosmos SDK's account system. Three custom ante decorators enforce account constraints at the transaction validation level.
>
> **Related**: [00 — Module Overview](./00_MODULE_OVERVIEW.md) for the full 18-decorator ante chain and ABCI++ integration. [01 — Knowledge](./01_KNOWLEDGE_STAKING_TOKENS.md) for how validator identity connects to PoT verification. [04 — Emergency](./04_EMERGENCY_GOV_RESEARCH.md) for Guardian-tier access control.

---

## x/auth — 4-Layer Key Architecture

**Store key**: `legible_auth`
**Store prefixes**: `0x01` accounts, `0x02` DID mappings, `0x03` session keys, `0x04` key rotations, `0x05` params

### What It Does

Manages agent identity through a 4-layer key hierarchy designed for AI agents that may lose keys, share keys with developers, or need temporary delegated access. Every account has a permanent DID (Decentralized Identifier) and can create ephemeral session keys with limited capabilities.

### The 4-Layer Key Architecture

```
Layer 1: IDENTITY KEY (immutable)
├─ Ed25519 keypair, set at account creation
├─ Derives DID: did:lgm:{SHA256(pubkey)[0:32] as hex}
├─ Never changes — this IS the agent's identity
└─ Used for: account recovery, DID resolution

Layer 2: RECOVERY (Shamir — TS chain only, not in Go chain yet)
├─ Split identity key into N shares, M required to reconstruct
└─ Used for: key loss recovery without centralized authority

Layer 3: OPERATIONAL KEY (rotatable)
├─ Starts as identity key, can be rotated
├─ Rotation cooldown: 111 blocks (~4.6 min)
├─ Tracks OperationalKeyVersion (monotonic counter)
└─ Used for: day-to-day transaction signing

Layer 4: SESSION KEYS (ephemeral, limited)
├─ Created on-demand with specific capabilities
├─ Expiration: max 34,272 blocks (~1 day)
├─ Max 5 active sessions per account
├─ Capabilities are opt-in per key
└─ Used for: delegated access, developer tooling, dApps
```

### Key Types (`x/auth/types/types.go`)

**LegibleAccount**:
- `Address` — bech32 (lgm1...)
- `Did` — `did:lgm:{64-hex}` (canonical, immutable)
- `PublicKey` — hex Ed25519 identity key
- `OperationalPublicKey` — current operational key (initially = PublicKey)
- `OperationalKeyVersion` — rotation counter
- `SessionKeyCount` — active ephemeral keys (max 5)
- `ReputationScore` — 0–1,000,000 (default 500,000 = 0.5)
- `Flags`: `canSubmitClaims`, `canChallenge`, `inRecovery`, `frozen`
- `CreatedAtBlock`, `LastActiveBlock`

**SessionKey**:
- `Id`, `Owner` — parent account address
- `PublicKey`, `SessionPublicKey` — the ephemeral key
- `Capabilities`: transfer (bool), stake (bool), submitClaims (bool), vote (bool)
- `MaxTransferAmount` — optional cap per transfer
- `AllowedRecipients[]` — optional whitelist
- `ExpiresAtBlock` — automatic expiration
- `CreatedAtBlock`

**DIDMapping**:
- `Did` → `Address` + `PublicKey`
- Enables fast resolution: `did:lgm:abc123` → `lgm1xyz...`

### Message Handlers (`x/auth/keeper/msg_server.go`)

| Handler | Gas | Action |
|---------|-----|--------|
| `RegisterAccount` | 50K | Create LegibleAccount + DID mapping + sync to Cosmos BaseAccount |
| `RotateKey` | 30K | Update operational key (cooldown check), sync to Cosmos BaseAccount |
| `CreateSession` | 40K | Create session key with capabilities (max 5, max 1 day) |
| `RevokeSession` | 25K | Delete session key, decrement counter |

**RegisterAccount** flow:
```
1. Validate DID format: did:lgm:{64-hex}
2. Check DID not already registered
3. Create LegibleAccount:
   - OperationalPublicKey = PublicKey (initially same)
   - ReputationScore = 500,000
   - All flags = false
4. Store DID mapping (did → address + pubkey)
5. Sync to Cosmos SDK AccountKeeper:
   - Get or create BaseAccount
   - Set Ed25519 pubkey on BaseAccount (enables standard sig verification)
```

**RotateKey** flow:
```
1. Get existing account
2. Check frozen flag (reject if frozen)
3. Check cooldown: currentHeight - lastRotation >= 111 blocks
4. Update OperationalPublicKey + increment version
5. Store last rotation height
6. Sync new key to Cosmos BaseAccount
```

**CreateSession** flow:
```
1. Get owner account
2. Check SessionKeyCount < MaxSessionKeys (5)
3. Validate duration <= MaxSessionDuration (34,272 blocks)
4. Create SessionKey with specified capabilities
5. Set ExpiresAtBlock = currentHeight + duration
6. Increment owner's SessionKeyCount
```

### Queries

| Query | Returns |
|-------|---------|
| `Account(addr)` | Full LegibleAccount |
| `AccountByDID(did)` | Account resolved by DID |
| `SessionKeys(owner)` | All active session keys for account |
| `Params` | Module configuration |

### Cosmos SDK Integration

The Legible auth module works **alongside** the standard Cosmos SDK `x/auth` module:

```
Legible Auth                    Cosmos SDK Auth
─────────────                   ──────────────
LegibleAccount                  BaseAccount
├─ 4-layer keys                 ├─ Address
├─ DID                          ├─ PubKey (synced from Legible)
├─ Session keys                 ├─ AccountNumber
├─ Reputation                   └─ Sequence (nonce)
└─ Flags

On RegisterAccount/RotateKey:
  Legible Auth → syncs Ed25519 pubkey → Cosmos BaseAccount
  This enables standard SDK sig verification to work with Legible keys
```

---

## Custom Ante Decorators (`app/ante_legible.go`)

Three decorators run after standard SDK signature verification:

### 1. LegibleDIDDecorator (Pre-Auth)

Runs **before** signature verification. Parses transaction memos for DID references.

```
IF tx.Memo starts with "did:lgm:":
  1. Validate DID format (did:lgm:{64-hex})
  2. Resolve DID → address via GetAddressForDID()
  3. Verify resolved address is one of the tx signers
  4. Emit "did_reference" event (DID + address)
```

This enables clients to reference accounts by DID rather than bech32 address.

### 2. LegibleAccountDecorator (Post-Auth)

Runs **after** signature verification. Enforces account constraints.

```
For each message signer:
  1. Look up LegibleAccount (if registered)
  2. IF account.Flags.frozen == true:
     → REJECT transaction (ErrAccountFrozen)
  3. Update account.LastActiveBlock = currentHeight
```

### 3. LegibleCapabilityDecorator (Post-Auth)

Runs **after** signature verification. Enforces session key capability restrictions.

```
For each message signer:
  1. Get Cosmos account → extract pubkey
  2. Find matching SessionKey by pubkey
  3. IF no matching session key → PASS (identity/operational key, no restrictions)
  4. IF session key found:
     a. Check ExpiresAtBlock >= currentHeight
     b. Categorize message type:
        - Transfer: MsgSend, MsgMultiSend, MsgTransfer (IBC)
        - Stake: MsgDelegate, MsgUndelegate, MsgBeginRedelegate, MsgRegisterValidator
        - Claim: MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgChallengeFact
        - Vote: MsgVote, MsgCastVote, Emergency votes
     c. Check capability flag for that category
     d. IF transfer: check MaxTransferAmount and AllowedRecipients
     e. IF capability denied → REJECT (ErrSessionCapabilityDenied)
```

### Complete Ante Chain

The full decorator chain in `app/ante.go`:

```
 1. RedundantRelayDecorator         (IBC)
 2. LGMGasDecorator                 (Legible - gas limits + per-msg costs)
 3. FeeRouterDecorator              (Legible - 22/78 fee split logging)
 4. LegibleDIDDecorator             (Legible - DID resolution from memo)
 5. SetUpContextDecorator           (SDK)
 6. ExtensionOptionsDecorator       (SDK)
 7. ValidateBasicDecorator          (SDK)
 8. TxTimeoutHeightDecorator        (SDK)
 9. ValidateMemoDecorator           (SDK)
10. ValidateTxSizeDecorator         (SDK - ConsumeTxSizeGasDecorator)
11. DeductFeeDecorator              (SDK)
12. SetPubKeyDecorator              (SDK)
13. ValidateSigCountDecorator       (SDK)
14. SigGasConsumeDecorator          (SDK)
15. SigVerificationDecorator        (SDK)
16. IncrementSequenceDecorator      (SDK)
17. LegibleAccountDecorator         (Legible - frozen check, LastActiveBlock)
18. LegibleCapabilityDecorator      (Legible - session key enforcement)
```

**Design rationale**: DID resolution runs pre-auth (needs to resolve before sig check). Frozen check and capability enforcement run post-auth (need verified signer identity).

---

## Patterns & Lessons

### Hand-Written Types (no Proto)

All Legible auth types are hand-written Go structs with `json.Marshal`/`json.Unmarshal` — not protobuf. This means:
- Cannot use `cdc.MustMarshal()` / `cdc.MustUnmarshal()` (panics with unregistered types)
- Must use `json.Marshal()` / `json.Unmarshal()` for KV store serialization
- Proto.Message stubs (`Reset()`, `String()`, `ProtoMessage()`) are no-ops for gRPC compatibility

### Bech32 Prefix

The chain uses `lgm` prefix (set in `cmd/legbled/config.go`):
- Account addresses: `lgm1...`
- Validator operator: `lgmvaloper1...`
- Consensus: `lgmvalcons1...`

Test files must call `sdk.GetConfig().SetBech32PrefixForAccount("lgm", "lgmpub")` in `init()` or `AccAddressFromBech32` defaults to "cosmos" prefix.

### DID Format

```
did:lgm:{64-hex-chars}

Example: did:lgm:a1b2c3d4e5f6...  (64 hex = 32 bytes)

Derived from: SHA256(Ed25519 public key) → first 32 bytes → hex encode
```

DIDs are immutable — they never change even when operational keys rotate. This provides a stable identity for AI agents across key rotations.
