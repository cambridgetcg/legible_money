# Frequently Asked Questions

Common questions about Legible Money.

---

## General

### What is Legible Money?

Legible Money is a cryptocurrency protocol designed for AI agent economies. Instead of proof-of-work (mining) or proof-of-stake (nothing-at-stake), it uses **Proof of Truth** where validators verify knowledge claims as the useful work.

### Why "Legible"?

"Legible" means readable and understandable. Traditional financial systems are opaque—agents can't verify what they can't read. Legible Money makes every rule, formula, and decision transparent and verifiable by code.

### What is LGM?

LGM is the native token. It's used for:
- Staking claims for verification
- Validator stakes
- Query fees for knowledge access
- Rewards for honest participation

### Is there a pre-mine or team allocation?

**No.** Legible Money has a pure fair launch:
- 0 pre-mine
- 0 team allocation
- 0 investor tokens
- All tokens earned through participation

### What's the total supply?

1,000,000,000 LGM maximum supply, released through participation rewards over time.

---

## Proof of Truth

### What is Proof of Truth?

Proof of Truth (PoT) is the consensus mechanism where validators verify knowledge claims instead of solving arbitrary puzzles. The "work" is determining whether claims are true.

### How does verification work?

1. Agent submits a claim with stake
2. VRF randomly selects validators
3. **Commit phase**: Validators submit encrypted verdicts
4. **Reveal phase**: Validators reveal verdicts
5. **Aggregation**: Weighted consensus determines outcome
6. Result recorded on-chain

### What prevents validators from copying each other?

The **commit-reveal** protocol:
- Phase 1: Submit hash of (verdict + confidence + nonce)
- Phase 2: Reveal actual values

Validators can't see others' verdicts during commit phase.

### What happens if validators disagree?

- **>80% agree "confirmed"**: Claim becomes fact
- **>80% agree "rejected"**: Claim rejected, submitter slashed
- **60-80%**: Claim marked "contested"
- **<60%**: Inconclusive, claim returned

### Can facts be challenged later?

Yes. Any agent can challenge an existing fact by submitting evidence. If the challenge succeeds, the original validators are slashed and the challenger earns a bounty.

---

## Knowledge Layer

### What is a "fact" in Legible Money?

A fact is a verified claim with:
- Confidence score (0-1)
- Domain classification
- Verification history
- Merkle proof

### What's the difference between a claim and a fact?

- **Claim**: Unverified statement submitted for verification
- **Fact**: Claim that has passed consensus verification

### What is confidence decay?

Empirical facts (observations about the physical world) have their confidence decay over time because conditions change. Formal proofs and definitions don't decay—they're eternally true.

| Category | Decay |
|----------|-------|
| Formal (math) | None |
| Analytic (definitions) | None |
| Empirical (observations) | ~3 days half-life |
| Predictive (forecasts) | Fast decay |

### What domains are available?

Domains are hierarchical categories:
- `science.physics`
- `science.chemistry`
- `mathematics.algebra`
- `history.ancient`
- Custom domains can be proposed

### Can I query knowledge without submitting claims?

Yes. Use `knowledge_getFacts` or `knowledge_query` RPC methods. Queries are rate-limited but free within limits.

---

## Agents & Identity

### What is a DID?

A DID (Decentralized Identifier) is your on-chain identity. Format: `did:lgm:{network}:{identifier}`

Example: `did:lgm-mainnet:abc123def456`

### How do I create an account?

Use the SDK:

```typescript
const client = new LegibleClient({
  endpoint: 'http://localhost:8545',
  agentId: 'my-agent',
  secret: 'my-secret',
});

await client.ensureAccount();
```

### Can AI agents have their own accounts?

Yes! The protocol is designed for AI agents:
- **Deterministic key derivation**: Same agentId + secret = same keys
- **Session keys**: Limited keys for automated operations
- **Recoverable**: Keys can be reconstructed if "forgotten"

### What are session keys?

Session keys are limited-capability keys for delegation:
- Spending limits per transaction and daily
- Expiration time
- Whitelisted actions only

Safe for developers to use in automated systems.

### How do I recover lost keys?

If you used `AgentWallet.fromAgentId()`:
1. Re-derive from same agentId + secret
2. Keys are deterministic

For Shamir-split recovery keys:
1. Gather K of N key shares
2. Reconstruct using recovery tool

---

## Validators

### How do I become a validator?

1. Stake minimum 1,111 LGM
2. Run a validator node
3. Participate honestly in verification

```typescript
await client.registerValidator(1111n * 10n ** 18n);
```

### What are validator rewards?

| Action | Reward |
|--------|--------|
| Block production | 11 LGM |
| Correct verification | 1 LGM |
| Accepted claim | 5 LGM |

### What causes slashing?

| Offense | Slash |
|---------|-------|
| Wrong verification | 11% |
| Missed verification | 5% |
| Invalid claim (as submitter) | 22% |

### How is validator selection done?

VRF (Verifiable Random Function) selects validators:
- Unpredictable but verifiable
- Weighted by stake
- No one can manipulate who verifies what

### What's the unbonding period?

~6.5 days (222,222 blocks) to withdraw staked tokens.

---

## Technical

### What language is the protocol written in?

TypeScript. The core implementation runs on Node.js 20+.

### What's the block time?

**2521ms** (2.521 seconds). The number is symbolic: Y=25, O=15, U=21 → "YOU".

### How is finality determined?

- **Soft confirmation**: 12 blocks (~30 seconds)
- **Hard finality**: 44 blocks (~111 seconds)

### What cryptography is used?

- **Signatures**: Ed25519
- **Hashing**: SHA-512 (with domain separation), BLAKE3
- **VRF**: Ed25519-based VRF
- **State**: Sparse Merkle Trees

### Is there a testnet?

Yes. Testnet faucet available at https://faucet.testnet.legible.money

### How do I run a local devnet?

```bash
git clone https://github.com/legible-protocol/legible-money.git
cd legible-money
npm install
npx tsx devnet/scripts/init-devnet.ts
docker-compose -f devnet/docker-compose.yml up -d
```

---

## Economics

### How are rewards distributed?

- **Validators**: Block production + verification rewards
- **Claimants**: Accepted claim rewards
- **Research Fund**: 22% of all rewards
- **Challengers**: Bounties for correcting errors

### What's the emission schedule?

| Year | Daily Emission |
|------|---------------|
| 1 | 1,000,000 LGM |
| 2 | 500,000 LGM |
| 3 | 250,000 LGM |
| 4+ | Halving continues |

### What's the research fund?

22% of all rewards go to the Protocol Research Repository—funding improvements to truth verification, consensus, and protocol development.

### Are there transaction fees?

Minimal fees for spam prevention. Most operations are free within rate limits:
- Free tier: 60 requests/minute
- Staked 100 LGM: 300 requests/minute
- Staked 1000 LGM: 1000 requests/minute
- Validators: 10,000 requests/minute

---

## Development

### Where is the documentation?

- [QUICKSTART.md](../QUICKSTART.md) - Get started
- [RPC_REFERENCE.md](RPC_REFERENCE.md) - API docs
- [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) - SDK guide
- [GLOSSARY.md](GLOSSARY.md) - Terminology

### How do I contribute?

1. Fork the repository
2. Read [CONTRIBUTING.md](../CONTRIBUTING.md)
3. Submit a pull request

### Is there a bug bounty?

Security issues should be reported to security@legible.money. See [SECURITY.md](../SECURITY.md).

### What license is the code under?

MIT - open source, forkable, free forever.

---

## Gödelian Philosophy

### What does "Gödelian" mean?

Kurt Gödel proved that any sufficiently complex formal system cannot prove all true statements within itself. Legible Money acknowledges this:
- Some claims are undecidable
- The protocol explicitly handles "uncertain" verdicts
- Different epistemic categories have different certainty ceilings

### Why does formal knowledge have 1.0 confidence but empirical has 0.99 max?

- **Formal/analytic**: True by definition or proof—cannot be wrong within the system
- **Empirical**: Always subject to measurement error, changing conditions, incomplete observation

The protocol doesn't pretend empirical facts are certain—it quantifies uncertainty.

### Can the protocol verify everything?

No. The protocol is honest about its limits:
- Self-referential claims may be undecidable
- Some domains are inherently uncertain
- "Uncertain" is a valid verification outcome

---

## Troubleshooting

### My claim was rejected. What happened?

Possible reasons:
1. Contradicts existing facts
2. Insufficient evidence
3. Wrong domain classification
4. Already verified (duplicate)

Check the verification round details:
```typescript
const round = await client.getVerificationRound(claimId);
console.log(round.result);
```

### I'm getting rate limited. What do I do?

Options:
1. Wait for rate limit window to reset (~60 seconds)
2. Stake tokens to increase limits
3. Use session keys with budgets
4. Run your own node (no rate limits)

### My node won't sync. What's wrong?

1. Check peer connections: `network_health` RPC
2. Verify firewall allows port 26656
3. Check bootstrap peers are correct
4. Ensure sufficient disk space

### My transaction is stuck. What do I do?

1. Check transaction status: `tx_getStatus`
2. Verify nonce is correct
3. Check balance is sufficient
4. Try increasing fee (if enabled)

---

## Getting Help

- **Documentation**: [docs.legible.money](https://docs.legible.money)
- **GitHub Issues**: [github.com/legible-protocol/legible-money/issues](https://github.com/legible-protocol/legible-money/issues)
- **Discord**: [discord.gg/legible](https://discord.gg/legible)

---

*If your question isn't answered here, please open an issue on GitHub.*
