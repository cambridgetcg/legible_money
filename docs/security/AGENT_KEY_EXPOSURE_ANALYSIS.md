# Security Analysis: Agent Key Exposure Risk

**Status**: 🔴 CRITICAL VULNERABILITY IDENTIFIED
**Date**: 2026-02-05
**Analyst**: Claude (PP Security Investigation)

---

## Executive Summary

The current CLI design exposes agent private keys and secrets through multiple attack vectors. When an AI agent (like Claude) uses the CLI, **the agent-secret is visible in the conversation context**, which could be:
1. Logged by the AI provider
2. Visible in conversation history
3. Potentially leaked through model outputs

**This creates a direct path to financial loss.**

---

## Attack Surface Analysis

### 1. Command Line Argument Exposure

**Severity**: 🔴 CRITICAL

```bash
lgm knowledge submit "claim" \
  --agent-id "claude-agent" \
  --agent-secret "my-secret-key-12345"  # ← VISIBLE
```

**Exposure Points**:
- `ps aux` shows full command line to all system users
- Shell history (`~/.bash_history`, `~/.zsh_history`)
- Process accounting logs (`/var/log/wtmp`)
- System audit logs (auditd, syslog)
- Container/orchestrator logs (Kubernetes, Docker)

**Impact**: Anyone with read access to these logs can derive the wallet and steal funds.

### 2. Claude Conversation Context Exposure

**Severity**: 🔴 CRITICAL

When Claude executes:
```bash
lgm wallet derive --agent-id "claude" --agent-secret "secret123"
```

**The secret is**:
- Visible in the conversation transcript
- Potentially stored in Anthropic's logs
- Visible to the human user
- Could be included in model training data (future models)
- Stored in Claude Code's session files (`.claude/projects/`)

**Impact**: The human user, Anthropic systems, or future model outputs could leak the secret.

### 3. Environment Variable Exposure

**Severity**: 🟡 HIGH

```bash
export LGM_PRIVATE_KEY="0xabc123..."
lgm transfer ...
```

**Exposure Points**:
- `/proc/<pid>/environ` readable by same user
- Child processes inherit environment
- Core dumps may include environment
- Container inspection reveals env vars

### 4. Wallet File Storage

**Severity**: 🟡 HIGH

File: `~/.legible/wallet.json`
```json
{
  "agentId": "my-agent",
  "agentSecret": "super-secret",  // ← PLAINTEXT
  "privateKey": "0x..."           // ← PLAINTEXT
}
```

**Exposure Points**:
- File permission issues (default 0o600 is good, but...)
- Backup systems capture the file
- Cloud sync services (Dropbox, iCloud)
- Editor swap files (`~/.legible/.wallet.json.swp`)
- Git accidentally committed

### 5. Output Exposure

**Severity**: 🟡 HIGH

`lgm wallet export` outputs the private key:
```json
{
  "privateKey": "0xabc123...",
  "warning": "NEVER SHARE..."
}
```

This appears in:
- stdout (captured by calling process)
- Claude's response to the user
- Log files if output is redirected

### 6. Debug Command Exposure

**Severity**: 🟡 MEDIUM

`lgm debug config` shows:
```json
{
  "wallet": {
    "source": "agent-derivation",
    "info": { "did": "...", "publicKey": "..." }
  }
}
```

While it doesn't show the secret, it confirms the derivation method.

`lgm debug derive` shows:
```json
{
  "input": {
    "agentId": "claude-opus-session",
    "secretLength": 16,  // ← Reveals secret length
    "network": "devnet"
  }
}
```

---

## Threat Model: Claude as Agent

### Scenario: Claude Uses CLI

1. User asks Claude to submit a knowledge claim
2. Claude runs: `lgm knowledge submit "claim" --agent-id X --agent-secret Y`
3. **The secret Y is now in**:
   - The conversation history (visible to user)
   - Anthropic's logs (visible to Anthropic)
   - Claude Code session file (on disk)
   - Potentially shell history (on disk)

### Attack Vectors

| Vector | Attacker | Access Method |
|--------|----------|---------------|
| Conversation leak | User | Reads conversation, derives wallet |
| Log analysis | Anthropic employee | Reads logs, derives wallet |
| Session file theft | Malware | Reads `.claude/` directory |
| Shell history | Local attacker | Reads `~/.bash_history` |
| Process snooping | Local attacker | Reads `/proc/*/cmdline` |
| Shoulder surfing | Physical attacker | Watches terminal output |

### Financial Impact

If an attacker obtains `agentId + agentSecret`:
1. Derive wallet: `AgentWallet.fromAgentId(id, secret)`
2. Get DID and private key
3. Transfer all funds
4. Claim rewards
5. Unstake tokens
6. Submit malicious claims (reputation damage)

**Total loss**: 100% of agent's holdings + reputation damage

---

## Current Mitigations (Insufficient)

| Mitigation | Status | Gap |
|------------|--------|-----|
| Private key not in default output | ✅ | But available via `--expose-private` |
| Wallet file has 0o600 permissions | ✅ | Doesn't prevent backups, editors |
| Session keys mentioned | ⚠️ | Not fully implemented |
| Environment variable option | ⚠️ | Still exposed in /proc |

---

## Recommended Fixes

### Immediate (P0) - Block Critical Exposure

#### 1. Never Accept Secrets via Command Line

```typescript
// BAD: Current implementation
.option('--agent-secret <secret>', 'Agent secret for key derivation')

// GOOD: Read from secure source only
.option('--agent-secret-file <path>', 'Path to file containing secret')
.option('--agent-secret-env <var>', 'Environment variable containing secret')
```

Implementation:
```typescript
function getAgentSecret(opts: GlobalOptions): string | null {
  // Priority 1: File (most secure - not in process list)
  if (opts.agentSecretFile) {
    return readFileSync(opts.agentSecretFile, 'utf8').trim();
  }

  // Priority 2: Named env var (somewhat secure)
  if (opts.agentSecretEnv) {
    return process.env[opts.agentSecretEnv] || null;
  }

  // Priority 3: stdin (for piping)
  // ... read from stdin if TTY

  // NEVER: command line argument
  return null;
}
```

#### 2. Implement Secure Key Entry

```bash
# Instead of:
lgm transfer ... --agent-secret "visible-in-ps"

# Use:
lgm transfer ... --agent-secret-file /run/secrets/lgm-secret

# Or prompt (not for agents, but for humans):
lgm transfer ...
# Agent secret: ******** (hidden input)
```

#### 3. Add Security Warnings

```typescript
// In wallet-loader.ts
if (opts.agentSecret) {
  console.error('[SECURITY WARNING] Secret passed via command line is visible in process list');
  console.error('Use --agent-secret-file or --agent-secret-env instead');
  // Could even refuse to proceed in production mode
}
```

### Short-Term (P1) - Defense in Depth

#### 4. Implement Session Keys Properly

Session keys are the correct solution for agents:

```typescript
// Human creates session key with limits
lgm wallet session create \
  --max-spend-per-tx 100 \
  --max-spend-per-day 1000 \
  --expires-in 3600

// Returns session key (single-use token)
// { sessionKey: "sk_abc123...", capabilities: {...} }

// Agent uses session key (not the master secret)
lgm transfer ... --session-key "sk_abc123..."
```

**Why this is safer**:
- Session key has spending limits → max loss capped
- Session key expires → limited window
- Session key is single-purpose → no wallet export
- Master secret never exposed to agent

#### 5. Hardware Security Module (HSM) Integration

For high-value agents:
```bash
lgm --signer hsm:// transfer ...
```

The HSM holds the key, CLI sends signing requests.

#### 6. Audit Logging

```typescript
// Log all sensitive operations
auditLog({
  action: 'wallet_derive',
  agentId: opts.agentId,
  secretSource: 'command-line', // ← RED FLAG
  did: wallet.did,
  timestamp: Date.now(),
});
```

### Long-Term (P2) - Architectural Fixes

#### 7. Separate Signing Service

Instead of CLI holding keys:

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Claude    │ ───► │  CLI Tool   │ ───► │   Signer    │
│  (Agent)    │      │ (No keys)   │      │  Service    │
└─────────────┘      └─────────────┘      └─────────────┘
                                                 │
                                                 ▼
                                          ┌─────────────┐
                                          │  Hardware   │
                                          │   Wallet    │
                                          └─────────────┘
```

The CLI never sees the private key. It sends unsigned transactions to a signing service, which:
- Validates the request
- Checks spending limits
- Signs with HSM
- Returns signed transaction

#### 8. MPC Signing

Multi-party computation where:
- Agent holds share 1
- Signing service holds share 2
- Neither can sign alone

---

## Specific Claude Exposure Mitigation

### For Claude Code Users

1. **Never put secrets in prompts**
   ```
   BAD:  "Use secret 'abc123' to sign"
   GOOD: "Use the secret from /run/secrets/lgm"
   ```

2. **Use environment variables set outside conversation**
   ```bash
   # In shell before starting Claude
   export LGM_SECRET=$(cat /secure/path)

   # Claude sees: LGM_SECRET=[SET] not the value
   ```

3. **Implement "secret reference" system**
   ```bash
   lgm transfer ... --secret-ref "vault:lgm/agent-1"
   ```
   CLI fetches from vault, never appears in conversation.

### For Anthropic/Claude Architecture

1. **Redact sensitive patterns** in conversation logs
2. **Don't include shell command outputs** with secrets in training
3. **Provide secure secret injection API** for MCP tools

---

## Verification Checklist

After implementing fixes:

- [ ] `ps aux | grep lgm` shows no secrets
- [ ] Shell history contains no secrets
- [ ] Conversation transcript contains no secrets
- [ ] `/proc/<pid>/cmdline` shows no secrets
- [ ] `lgm debug config` shows `[REDACTED]` not values
- [ ] Wallet export requires explicit confirmation
- [ ] Session keys implemented with spending limits
- [ ] Audit log captures all sensitive operations

---

## Conclusion

**The current CLI design is unsuitable for production use with AI agents.**

The fundamental issue is that deterministic key derivation requires the secret to be present, and the current design exposes this secret through multiple channels.

**Recommended path forward**:
1. Immediate: Add warnings, deprecate `--agent-secret` flag
2. Short-term: Implement session keys with spending limits
3. Long-term: Move to signing service architecture

**Until fixed**: Agents should only use testnets or hold minimal funds.

---

## References

- OWASP Secrets Management: https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html
- CWE-214: Invocation of Process Using Visible Sensitive Information
- CWE-532: Insertion of Sensitive Information into Log File
