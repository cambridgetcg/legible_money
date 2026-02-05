# Secure Agent Setup Flow Design

**Purpose**: Guide humans through creating secure secret storage that agents can use without exposing keys.

---

## Design Principles (PP Framework)

### TRUTH
- Secrets must never appear in command lines, logs, or conversation context
- Humans are the trust anchor; agents are delegated limited authority

### UNDERSTANDING
- Humans understand security but need clear guidance
- Agents need simple, safe access patterns
- Different environments need different storage backends

### BEAUTY
- Single command to set up secure storage
- Clear prompts with sensible defaults
- Progressive disclosure (simple path, advanced options)

### JUSTICE
- Secrets belong in secure storage, not in CLI args
- Spending limits belong with session keys
- Audit logs belong with the signing service

### CREATIVITY
- Build a path from "human has secret" to "agent can sign safely"
- Session keys as delegation mechanism
- Multiple storage backends for different use cases

---

## Prompt Flow Design

### Command: `lgm setup agent`

```
┌─────────────────────────────────────────────────────────────────┐
│                    LEGIBLE MONEY AGENT SETUP                     │
│                                                                  │
│  This wizard will help you securely configure an agent wallet.  │
│  The secret will be stored safely and never exposed to the      │
│  agent or command line.                                         │
└─────────────────────────────────────────────────────────────────┘
```

---

### Step 1: Choose Agent Identity

```
╭─ Step 1 of 5: Agent Identity ─────────────────────────────────────╮
│                                                                    │
│  What should this agent be called?                                 │
│                                                                    │
│  This ID is used to derive the wallet deterministically.          │
│  Use something memorable and unique to this agent.                 │
│                                                                    │
│  Examples:                                                         │
│    • claude-research-assistant                                     │
│    • trading-bot-prod-1                                            │
│    • knowledge-curator-v2                                          │
│                                                                    │
│  Agent ID: █                                                       │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

**Validation**:
- Must be 3-64 characters
- Alphanumeric, hyphens, underscores only
- Must not already exist in storage

---

### Step 2: Generate or Import Secret

```
╭─ Step 2 of 5: Agent Secret ───────────────────────────────────────╮
│                                                                    │
│  How would you like to create the agent secret?                    │
│                                                                    │
│  [1] Generate secure random secret (recommended)                   │
│  [2] Enter my own secret                                           │
│  [3] Import from existing wallet file                              │
│                                                                    │
│  Choice [1]: █                                                     │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

**Option 1: Generate**
```
╭─ Generating Secret ───────────────────────────────────────────────╮
│                                                                    │
│  ✓ Generated 256-bit cryptographically secure secret              │
│                                                                    │
│  Your recovery phrase (WRITE THIS DOWN SECURELY):                  │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  witch collapse practice feed shame open despair creek     │   │
│  │  road again ice least                                      │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ⚠️  This is the ONLY time you will see this phrase.              │
│  ⚠️  Store it in a secure location (password manager, safe).      │
│  ⚠️  Anyone with this phrase can access the wallet.               │
│                                                                    │
│  Have you written down the recovery phrase? [y/N]: █               │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

**Option 2: Enter Own**
```
╭─ Enter Secret ────────────────────────────────────────────────────╮
│                                                                    │
│  Enter your agent secret (input hidden):                           │
│                                                                    │
│  Secret: ••••••••••••••••••••••••••••••••                          │
│  Confirm: ••••••••••••••••••••••••••••••••                         │
│                                                                    │
│  ⚠️  Ensure your secret has sufficient entropy (32+ characters)   │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

### Step 3: Choose Storage Backend

```
╭─ Step 3 of 5: Secure Storage ─────────────────────────────────────╮
│                                                                    │
│  Where should the secret be stored?                                │
│                                                                    │
│  [1] System Keychain (recommended)                                 │
│      macOS Keychain / Linux Secret Service / Windows Credential    │
│      ✓ Encrypted at rest                                           │
│      ✓ Requires user authentication                                │
│                                                                    │
│  [2] Encrypted file                                                │
│      ~/.legible/secrets/agent-name.enc                             │
│      ✓ Portable across systems                                     │
│      ✓ Works in containers                                         │
│                                                                    │
│  [3] Environment variable reference                                │
│      Store in: LGM_AGENT_SECRET_<NAME>                             │
│      ⚠️  You manage the env var securely                           │
│                                                                    │
│  [4] HashiCorp Vault (enterprise)                                  │
│      Path: secret/legible/agents/<name>                            │
│      ✓ Centralized secret management                               │
│      ✓ Audit logging                                               │
│                                                                    │
│  Choice [1]: █                                                     │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

**Option 1: Keychain** (macOS example)
```
╭─ Keychain Storage ────────────────────────────────────────────────╮
│                                                                    │
│  Storing secret in macOS Keychain...                               │
│                                                                    │
│  Service:  com.legible.agent                                       │
│  Account:  claude-research-assistant                               │
│  Access:   This application only                                   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  macOS will prompt for your password to save this item.    │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ✓ Secret stored in Keychain                                       │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

**Option 2: Encrypted File**
```
╭─ Encrypted File Storage ──────────────────────────────────────────╮
│                                                                    │
│  The secret will be encrypted with a password you choose.          │
│  You'll need this password to unlock the agent.                    │
│                                                                    │
│  Encryption password: ••••••••••••••••                             │
│  Confirm password:    ••••••••••••••••                             │
│                                                                    │
│  ⚠️  If you forget this password, you'll need the recovery phrase │
│                                                                    │
│  Encrypting with AES-256-GCM...                                    │
│  ✓ Saved to ~/.legible/secrets/claude-research-assistant.enc      │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

### Step 4: Configure Session Keys (Agent Limits)

```
╭─ Step 4 of 5: Agent Permissions ──────────────────────────────────╮
│                                                                    │
│  Session keys limit what the agent can do, protecting you from     │
│  bugs, prompt injection, or compromised agents.                    │
│                                                                    │
│  Maximum spend per transaction:                                    │
│  └─ [100] LGM                                                      │
│                                                                    │
│  Maximum spend per day:                                            │
│  └─ [1000] LGM                                                     │
│                                                                    │
│  Session duration:                                                 │
│  └─ [24] hours (then requires re-authentication)                   │
│                                                                    │
│  Allowed actions:                                                  │
│  └─ [x] Query knowledge (read-only)                                │
│  └─ [x] Submit claims                                              │
│  └─ [x] Transfer tokens                                            │
│  └─ [ ] Stake/unstake                                              │
│  └─ [ ] Challenge facts                                            │
│  └─ [ ] Export wallet                                              │
│                                                                    │
│  Press Enter to continue...                                        │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

### Step 5: Summary and Activation

```
╭─ Step 5 of 5: Review & Activate ──────────────────────────────────╮
│                                                                    │
│  Agent Configuration Summary:                                      │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  Agent ID:     claude-research-assistant                    │   │
│  │  DID:          did:lgm:a3f8c9d2e1b4...                      │   │
│  │  Public Key:   0x7a8b9c0d1e2f3a4b...                        │   │
│  │  Network:      mainnet                                      │   │
│  │                                                             │   │
│  │  Storage:      macOS Keychain                               │   │
│  │  Encryption:   AES-256-GCM                                  │   │
│  │                                                             │   │
│  │  Permissions:                                               │   │
│  │    Max per tx:   100 LGM                                    │   │
│  │    Max per day:  1,000 LGM                                  │   │
│  │    Session:      24 hours                                   │   │
│  │    Actions:      query, submit, transfer                    │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  Activate this agent? [Y/n]: █                                     │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

### Completion

```
╭─ Setup Complete ──────────────────────────────────────────────────╮
│                                                                    │
│  ✓ Agent "claude-research-assistant" is ready                      │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  HOW THE AGENT USES THIS:                                   │   │
│  │                                                             │   │
│  │  The agent runs commands like:                              │   │
│  │                                                             │   │
│  │    lgm transfer did:lgm:recipient --amount 50 \             │   │
│  │        --agent-ref "claude-research-assistant"              │   │
│  │                                                             │   │
│  │  The CLI fetches the secret from secure storage.            │   │
│  │  The secret NEVER appears in commands or logs.              │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  TO UNLOCK (if session expires):                            │   │
│  │                                                             │   │
│  │    lgm agent unlock claude-research-assistant               │   │
│  │                                                             │   │
│  │  You'll be prompted for your Keychain/encryption password.  │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  IMPORTANT REMINDERS:                                       │   │
│  │                                                             │   │
│  │  • Recovery phrase is stored ONLY by you (not on disk)      │   │
│  │  • Session keys protect against overspending                │   │
│  │  • Review agent actions in: lgm agent audit <name>          │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

## Implementation Architecture

### Storage Backends

```typescript
// cmd/lib/secret-storage.ts

interface SecretStorage {
  store(agentId: string, secret: string): Promise<void>;
  retrieve(agentId: string): Promise<string>;
  delete(agentId: string): Promise<void>;
  exists(agentId: string): Promise<boolean>;
}

class KeychainStorage implements SecretStorage {
  // macOS: security add-generic-password
  // Linux: secret-tool store
  // Windows: cmdkey
}

class EncryptedFileStorage implements SecretStorage {
  // AES-256-GCM encrypted file
  // Password derived via Argon2
}

class VaultStorage implements SecretStorage {
  // HashiCorp Vault API
}

class EnvRefStorage implements SecretStorage {
  // Just stores the env var name
  // Actual secret in environment
}
```

### Agent Reference System

```typescript
// Instead of --agent-secret, use --agent-ref

async function loadWalletFromRef(agentRef: string): Promise<Wallet> {
  // 1. Load agent config
  const config = await loadAgentConfig(agentRef);

  // 2. Get secret from configured storage
  const storage = getStorage(config.storageBackend);
  const secret = await storage.retrieve(agentRef);

  // 3. Derive wallet
  return AgentWallet.fromAgentId(agentRef, secret, config.network);
}
```

### Session Key Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        HUMAN SETUP                               │
│                                                                  │
│  1. Human runs: lgm setup agent                                  │
│  2. Human enters secret (hidden input)                           │
│  3. Secret stored in Keychain/Vault (encrypted)                  │
│  4. Session key config saved (spending limits)                   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        AGENT USAGE                               │
│                                                                  │
│  1. Agent runs: lgm transfer ... --agent-ref "my-agent"         │
│  2. CLI checks session key validity                              │
│  3. CLI checks spending limits                                   │
│  4. CLI fetches secret from secure storage                       │
│  5. CLI signs transaction (secret in memory only)                │
│  6. CLI clears secret from memory                                │
│  7. Transaction submitted                                        │
│                                                                  │
│  SECRET NEVER APPEARS IN:                                        │
│  • Command line arguments                                        │
│  • Process list (ps)                                             │
│  • Shell history                                                 │
│  • Log files                                                     │
│  • Agent conversation context                                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## CLI Commands Summary

```bash
# Setup (human runs once)
lgm setup agent                    # Interactive wizard
lgm setup agent --non-interactive  # For automation (reads from stdin)

# Agent management (human)
lgm agent list                     # List configured agents
lgm agent unlock <name>            # Refresh session after expiry
lgm agent lock <name>              # Invalidate current session
lgm agent audit <name>             # View agent action history
lgm agent revoke <name>            # Permanently disable agent
lgm agent rotate <name>            # Generate new secret (invalidates old)

# Agent usage (agent runs)
lgm transfer ... --agent-ref <name>
lgm knowledge submit ... --agent-ref <name>
lgm rewards --agent-ref <name>

# Session management
lgm session status --agent-ref <name>   # Check session validity
lgm session limits --agent-ref <name>   # Check remaining limits
```

---

## Security Properties

| Property | Implementation |
|----------|----------------|
| Secret never in CLI args | `--agent-ref` instead of `--agent-secret` |
| Secret never in logs | Retrieved directly to memory, cleared after use |
| Secret encrypted at rest | Keychain/Vault/AES-256 |
| Spending limits enforced | Session key config checked before signing |
| Session expiry | Time-based + block-based limits |
| Audit trail | All agent actions logged locally |
| Revocation | `lgm agent revoke` invalidates all sessions |

---

## Error Messages

### Session Expired
```
╭─ Session Expired ─────────────────────────────────────────────────╮
│                                                                    │
│  The session for "claude-research-assistant" has expired.          │
│                                                                    │
│  A human must run:                                                 │
│                                                                    │
│    lgm agent unlock claude-research-assistant                      │
│                                                                    │
│  This requires Keychain authentication.                            │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

### Spending Limit Exceeded
```
╭─ Spending Limit Exceeded ─────────────────────────────────────────╮
│                                                                    │
│  This transaction exceeds the agent's spending limits.             │
│                                                                    │
│  Requested:     500 LGM                                            │
│  Limit per tx:  100 LGM                                            │
│                                                                    │
│  A human must either:                                              │
│  • Increase limits: lgm agent config claude-research-assistant     │
│  • Perform manually: lgm transfer ... (with human wallet)          │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

### Action Not Allowed
```
╭─ Action Not Permitted ────────────────────────────────────────────╮
│                                                                    │
│  The agent "claude-research-assistant" is not permitted to         │
│  perform action: stake                                             │
│                                                                    │
│  Allowed actions: query, submit, transfer                          │
│                                                                    │
│  A human must update permissions:                                  │
│    lgm agent config claude-research-assistant --allow stake        │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

## Migration Path

For existing users with `--agent-secret` in use:

```
╭─ Security Warning ────────────────────────────────────────────────╮
│                                                                    │
│  ⚠️  --agent-secret on command line is DEPRECATED                  │
│                                                                    │
│  Your secret may have been exposed in:                             │
│  • Shell history                                                   │
│  • Process lists                                                   │
│  • Log files                                                       │
│                                                                    │
│  RECOMMENDED ACTIONS:                                              │
│                                                                    │
│  1. Set up secure storage:                                         │
│     lgm setup agent                                                │
│                                                                    │
│  2. Rotate your secret (generates new wallet):                     │
│     lgm agent rotate <name>                                        │
│                                                                    │
│  3. Transfer funds to new wallet                                   │
│                                                                    │
│  4. Clear shell history:                                           │
│     history -c && history -w                                       │
│                                                                    │
│  The --agent-secret flag will be removed in v1.0.0                 │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯
```

---

## Summary

This design ensures:

1. **Human controls the secret** - Only entered once during setup
2. **Secret stored securely** - Keychain/Vault/encrypted file
3. **Agent uses reference** - `--agent-ref` not `--agent-secret`
4. **Spending limits protect** - Session keys cap potential loss
5. **Audit trail exists** - All actions logged
6. **Revocation possible** - Human can disable agent instantly

The agent NEVER sees the actual secret, only uses it indirectly through the secure storage system.
