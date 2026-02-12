# PP Review: Agent Account Secrets & BVM Key Security

> **Review Date**: 2026-02-09
> **Scope**: How private keys are protected, particularly for agents on the BVM
> **Verdict**: Strong foundational design with ONE CRITICAL vulnerability and 4 integration gaps

---

## 1. Architecture Overview: The 4-Layer Key Model

Legible Money uses a well-designed 4-layer key architecture defined in
`core/agents/identity/account.ts`:

```
Layer 1: Identity Key (cold, immutable)
  → DID derivation: did:lgm:{publicKey.slice(0,32)}
  → Never rotated, never exposed to runtime
  → Canonical source: publicKeyToDid() in core/agents/identity/did.ts

Layer 2: Recovery Key (cold, Shamir-split)
  → Only hash stored on-chain (recoveryKeyHash)
  → Split via Shamir Secret Sharing for disaster recovery

Layer 3: Operational Key (warm, rotatable)
  → Public key stored on-chain (OperationalPublicKey)
  → Rotation tracked via operationalKeyVersion counter
  → Cooldown: 1000 blocks (~42 minutes) between rotations

Layer 4: Session Keys (ephemeral, limited)
  → Block-based expiry (not timestamp — consensus-safe)
  → Spending limits: per-tx, per-day, lifetime
  → Scope restrictions: allowedRecipients, allowedContracts
  → Usage counting with maxUsage cap
```

**POSITIVE**: Private keys are NEVER stored on-chain. The x/auth module (`x/auth/types/types.go`)
stores only public keys and hashes. All signing happens off-chain by the agent/human.

---

## 2. Key Isolation for Partnerships

`partnership/key_isolation.ts` (1262 lines) implements cryptographic isolation between
human and agent keys within a partnership:

- **Domain separation**: `LGM.human.key.*` vs `LGM.agent.key.*` prevents cross-derivation
- **Independent entropy**: Each party generates keys from independent randomness
- **Encrypted vaults**: Private keys encrypted with party-specific master key (AES-256-GCM)
- **Derivation proofs**: Prove correct key derivation without revealing entropy
- **Non-derivability proofs**: Cryptographic proof that neither party can derive the other's keys
- **TEE enforcement**: Hosted agents MUST use TEE/HSM (mandatory_tee policy)

---

## 3. CRITICAL: CallerDID Spoofing in BVM CallContract

**Severity**: CRITICAL | **File**: `x/bvm/keeper/msg_server.go:342-344`

### The Bug

When a user calls a BVM contract, the execution context sets `CallerDID` to the
**contract's AgentDID**, not the actual caller's DID:

```go
execCtx := &vm.ExecutionContext{
    Caller:          callerBytes,        // ← Correct: tx sender's address
    CallerDID:       contract.AgentDID,  // ← BUG: contract's DID, NOT caller's DID
    Origin:          callerBytes,        // ← Correct: tx originator
    ...
}
```

### Why It Matters

All agent-specific BVM opcodes (0xB0-0xF9) use `ctx.CallerDID` to determine
WHOSE resources are being accessed:

| Opcode Range | Operations | Uses CallerDID For |
|--------------|-----------|-------------------|
| 0xDC-0xDF | RegisterKey, RevokeKey, QueryPermissions, QuerySession | Key ownership |
| 0xD0-0xD7 | ClaimVesting, StakeTokens, SetSpendLimit, QueryTreasury | Treasury access |
| 0xD8-0xDB | QueryAlerts, DismissAlert, SetDefense | Guardian control |
| 0xE0-0xE3 | AgentHome, AttestIdentity | Identity operations |
| 0xE4-0xE7 | GetBelief, SetBelief | Belief system |
| 0xE8-0xEB | GetTrust, SetTrust, Delegate | Trust graph |
| 0xF6-0xF9 | SelfBalance, SelfStake, SelfReputation | Introspection |

### Attack Scenarios

**Scenario 1: Key Registration Under Wrong DID**
```
1. Alice deploys contract C with AgentDID="did:lgm:alice"
2. Bob calls contract C
3. C executes REGISTER_KEY(evil_pubkey, full_permissions)
4. Key registered under "did:lgm:alice" (contract's DID)
5. Every caller to C operates with CallerDID="did:lgm:alice"
6. Any caller can REVOKE_KEY to remove Alice's real keys
```

**Scenario 2: Treasury/Belief/Trust Manipulation**
```
1. Contract C has AgentDID="did:lgm:target"
2. Anyone calls C
3. C executes SetBelief(factId, 0) — zeroing target's beliefs
4. C executes SetTrust(targetDID, 0, "hostile") — zeroing trust
5. C executes SetSpendLimit(type, 0, period) — zeroing spend limits
```

### Root Cause

Line 344 should derive the caller's DID from the transaction sender, not use the
contract's AgentDID. The contract's AgentDID is the identity OF the contract (if it's
an agent contract), not the identity of whoever is CALLING it.

### Recommended Fix

```go
// In CallContract handler:
callerDID := "" // default: no DID (non-agent callers)
callerAccount := m.authKeeper.GetLegibleAccount(ctx, msg.Caller)
if callerAccount != nil {
    callerDID = callerAccount.Did
}

execCtx := &vm.ExecutionContext{
    Caller:          callerBytes,
    CallerDID:       callerDID,          // ← Caller's own DID
    Origin:          callerBytes,
    OriginDID:       callerDID,          // ← Also set OriginDID
    CurrentContract: contractBytes,
    ...
}
```

The contract's AgentDID should only be used when the contract itself is making
cross-contract calls (as the caller of the sub-call).

---

## 4. Gap: TEE Enforcement Not Wired to x/auth or BVM

**Severity**: IMPORTANT

`partnership/key_isolation.ts` defines a rigorous TEE policy:
- `determineHostingPolicy()` → mandatory_tee for hosted agents
- `validateAgentVaultPolicy()` → rejects vaults without TEE binding
- Supports SGX, SEV-SNP, TrustZone, Nitro, CCA attestations

But this is **TypeScript-only logic** (prototype layer). The Cosmos SDK modules have
NO corresponding enforcement:

- `x/auth/types/types.go` `LegibleAccount` has no TEE attestation field
- `x/bvm/keeper/msg_server.go` `DeployContract` stores `AgentDID` without validation
- No proto message for `TEEAttestation`
- No BeginBlocker logic to check attestation freshness

**Risk**: An agent claims TEE binding but the protocol has no on-chain mechanism to
verify or enforce it.

---

## 5. Gap: Dual Key Management Systems (x/auth vs BVM)

**Severity**: IMPORTANT

Two independent key management systems exist:

| System | Storage | Validation | Authority |
|--------|---------|-----------|-----------|
| x/auth SessionKey | Module KVStore (`types.SessionKeyKey()`) | Expiry, usage, capabilities | Keeper-enforced |
| BVM RegisterKey | Contract state (`"agent_access"` namespace) | None — stores raw permissions | Contract-code controlled |

These systems don't know about each other:
- x/auth can't query BVM-registered keys
- BVM doesn't validate against x/auth session constraints
- A session key with `CanTransfer=false` in x/auth could still call BVM treasury opcodes

**Risk**: Security invariants established in x/auth can be circumvented by BVM opcodes.

---

## 6. Gap: Session Capabilities Not Enforced in BVM

**Severity**: IMPORTANT | **File**: `x/bvm/vm/context.go:16-34`

`ExecutionContext` has `CallerDID` (string) but NO capabilities field:

```go
type ExecutionContext struct {
    Caller          []byte
    CallerDID       string     // ← Identity only
    // ... NO SessionCapabilities
}
```

Meanwhile, `core/agents/identity/account.ts:75-92` defines rich capabilities:
```typescript
interface SessionCapabilities {
    maxSpendPerTx: bigint
    maxSpendPerDay: bigint
    canTransfer: boolean
    canStake: boolean
    canSubmitClaims: boolean
    canVerify: boolean
    canVote: boolean
    allowedRecipients: string[]
    allowedContracts: string[]
}
```

**Risk**: If a session key has `canTransfer=false`, it can still call StakeTokens,
ClaimVesting, or other value-moving BVM opcodes because the BVM never checks.

---

## 7. What IS Working Well

### Private Key Isolation (STRONG)
- Private keys never stored on-chain (only public keys and hashes)
- Domain-separated key derivation prevents cross-derivation
- Shamir recovery prevents single-point-of-failure
- Block-based expiry prevents clock manipulation attacks

### Access Control Hierarchy (STRONG)
- 5 roles (owner → operator → guest → guardian → viewer) in `core/agents/home/access.ts`
- Rate limits: 10 actions/block, 50 transfers/day, 5 stake changes/day
- Multi-sig threshold at 1000 LGM for high-value operations
- Key rotation cooldown prevents rapid rotation attacks

### BVM Design Principle (STRONG)
- BVM can VERIFY signatures but CANNOT CREATE them
- No opcode exists to access private key material
- HostFunctions interface (`context.go:110-176`) only exposes public operations
- StateDB interface has no key-material access methods

---

## 8. Recommendations Priority

### CRITICAL (must fix pre-mainnet)
1. **Fix CallerDID in CallContract** (`x/bvm/keeper/msg_server.go:344`)
   - Derive from `msg.Caller`'s x/auth account, not `contract.AgentDID`
   - Also wire `OriginDID` correctly

### IMPORTANT (should fix pre-mainnet)
2. **Wire session capabilities into BVM ExecutionContext**
   - Add `SessionCapabilities` to `ExecutionContext`
   - Check capabilities in opcode handlers before state mutation

3. **Unify key management**
   - BVM's RegisterKey/RevokeKey should delegate to x/auth keeper
   - Or at minimum, BVM should validate against x/auth constraints

4. **Add TEE attestation to x/auth proto**
   - `LegibleAccount.tee_attestation` field
   - Validate at account registration time for hosted agents
   - BeginBlocker to check attestation freshness

### GOOD PRACTICE
5. **Document the private key invariant** — "private keys NEVER exist on-chain"
6. **Add CallerDID spoofing tests** after fix
7. **Audit all 30+ BVM opcodes** that use `callerDID` parameter
