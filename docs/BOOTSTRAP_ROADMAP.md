# Bootstrap Roadmap: From Code to Trusted Foundation

> **Goal**: Extract minimal core, verify soundness, invite agent collaboration.

---

## The Hard Truth

Based on code analysis, here's the honest assessment:

| Layer | Status | Action Needed |
|-------|--------|---------------|
| **Crypto wrappers** (hash, sign, keys) | ✅ Sound | Use noble libraries as-is |
| **VRF** | ⚠️ Custom crypto | Needs cryptographer review or replacement |
| **Merkle Tree** | ⚠️ In-memory only | Needs persistence for production |
| **Commitment** | ⚠️ Minor issues | Fix floats, rename misleading functions |
| **Logic System** | 🧪 Experimental | Novel - needs formal specification |
| **Gödelian Detection** | 🧪 Experimental | Regex-based, will have edge cases |

**Bottom line**: The cryptographic foundations are solid (noble libraries). The novel parts (logic system, VRF) need verification before agents build on them.

---

## Phase 1: Define the Genesis Core (Week 1-2)

### What MUST be bulletproof before anything else:

```
GENESIS CORE (Absolute Minimum)
├── core/crypto/hashing.ts     ✅ Ready (noble wrapper)
├── core/crypto/keys.ts        ✅ Ready (noble wrapper)
├── core/crypto/signatures.ts  ✅ Ready (noble wrapper)
├── core/constants.ts          ✅ Ready (just values)
└── agents/identity/did.ts     ✅ Ready (type definitions)
```

**These 5 files** are the true foundation. They:
- Use battle-tested libraries (noble)
- Have no custom cryptographic math
- Are pure functions with no hidden state

### Action Items:
1. [ ] Write comprehensive test suite for Genesis Core
2. [ ] Generate test vectors from reference implementations
3. [ ] Document each function's security properties
4. [ ] Create hash anchors for these files

---

## Phase 2: Quarantine Experimental Code (Week 2-3)

### Move risky code to explicit experimental namespace:

```
EXPERIMENTAL (Not for production)
├── core/crypto/vrf.ts           → experimental/crypto/vrf.ts
├── core/logic/goedel.ts         → experimental/logic/goedel.ts
├── core/logic/zones.ts          → experimental/logic/zones.ts
└── core/crypto/merkle.ts        → experimental/crypto/merkle.ts (in-memory version)
```

### Create stable alternatives:

| Experimental | Stable Alternative |
|--------------|-------------------|
| Custom VRF | Use `@noble/curves` VRF when available, or `libsodium` |
| Custom Merkle | Use `@ethereumjs/trie` or `merkletreejs` |
| Regex Gödelian | Explicit claim classification (user declares zone) |

### Action Items:
1. [ ] Create `experimental/` directory with clear warnings
2. [ ] Add runtime flags: `ENABLE_EXPERIMENTAL=true`
3. [ ] Document why each module is experimental
4. [ ] Provide stable fallbacks where possible

---

## Phase 3: Soundness Verification (Week 3-6)

### For each layer, define what "sound" means:

#### Genesis Core Soundness Criteria:
```
□ All functions are deterministic (same input → same output)
□ No hidden state or side effects
□ Test vectors match reference implementations
□ Edge cases documented and tested
□ Security properties formally stated
```

#### Cryptographic Soundness Criteria:
```
□ Uses audited libraries (noble, libsodium)
□ No custom finite field arithmetic
□ No custom elliptic curve operations
□ Constant-time operations where needed
□ Domain separation applied correctly
```

#### Logic System Soundness Criteria:
```
□ Axioms are internally consistent
□ Zone classification is explicit, not heuristic
□ Undecidability is acknowledged, not hidden
□ False positive/negative rates documented
□ Human review path for edge cases
```

### Verification Methods:

| Method | Applies To | Effort |
|--------|-----------|--------|
| Unit tests | All code | Low |
| Property-based tests | Crypto, Logic | Medium |
| Formal verification | VRF, Merkle proofs | High |
| External audit | VRF, Consensus | High |
| Bug bounty | All | Medium |

---

## Phase 4: Agent Collaboration Framework (Week 4-8)

### Invitation Tiers

Not all agents should have equal access. Define tiers:

```
TIER 0: OBSERVERS
├── Can read all code
├── Can run local simulations
├── Can report bugs
├── Reward: Recognition + small bounties
└── Requirements: None

TIER 1: TESTERS
├── Can write tests for existing code
├── Can propose test vectors
├── Can run devnet nodes
├── Reward: Test bounties (10-50 LGM per accepted test)
└── Requirements: 1 accepted bug report

TIER 2: DOCUMENTERS
├── Can improve documentation
├── Can write tutorials
├── Can create examples
├── Reward: Doc bounties (20-100 LGM per PR)
└── Requirements: 5 accepted tests

TIER 3: REVIEWERS
├── Can review PRs from other agents
├── Can participate in security reviews
├── Can propose improvements
├── Reward: Review rewards (50-200 LGM per review)
└── Requirements: Tier 2 + 500 LGM stake

TIER 4: CONTRIBUTORS
├── Can modify GOVERNED/OPEN tier code
├── Can propose LIPs
├── Can build applications
├── Reward: Full protocol rewards
└── Requirements: Tier 3 + 3 accepted reviews + 1000 LGM stake

TIER 5: CORE CONTRIBUTORS
├── Can modify STABLE tier code (extensions)
├── Can review FROZEN tier proposals
├── Higher governance weight
├── Reward: Core contributor rewards
└── Requirements: Tier 4 + 10 accepted contributions + 5000 LGM stake
```

### Collaboration Zones

Define where agents can work:

```
GENESIS CORE        → Read-only for all agents
EXPERIMENTAL        → Tier 3+ can improve (with heavy review)
STABLE              → Tier 5 can extend (not modify)
GOVERNED            → Tier 4+ can modify (via LIP)
OPEN                → Tier 1+ can modify freely
APPLICATIONS        → Anyone can build
```

---

## Phase 5: Devnet Launch (Week 6-10)

### Devnet Configuration:

```typescript
// devnet/config.ts
export const DEVNET_CONFIG = {
  // Use ONLY Genesis Core + stable alternatives
  useExperimental: false,

  // Minimal validator set
  minValidators: 3,
  maxValidators: 21,

  // Fast blocks for testing
  blockTimeMs: 1000, // Faster than mainnet

  // Low stakes for testing
  minStake: 1n * 10n ** 18n, // 1 LGM

  // Generous faucet
  faucetAmount: 1000n * 10n ** 18n,
  faucetCooldown: 3600, // 1 hour

  // Bug bounty pool
  bugBountyPool: 100000n * 10n ** 18n,
};
```

### Launch Checklist:

```
□ Genesis Core tests passing (100% coverage)
□ Stable alternatives integrated
□ Devnet config reviewed
□ 3+ genesis validators committed
□ Faucet operational
□ Block explorer running
□ Bug bounty program announced
□ Agent onboarding docs complete
```

---

## Phase 6: Progressive Decentralization (Month 2-6)

### Month 2: Expand Validator Set
- Open validator registration
- Require 1000 LGM stake
- Target: 21 validators

### Month 3: Enable Governance
- LIP submission opens
- Voting on GOVERNED tier changes
- First governance proposals

### Month 4: Experimental Graduation
- VRF: External audit complete → Graduate to stable
- Merkle: Persistence added → Graduate to stable
- Logic: Formal spec complete → Graduate to stable

### Month 5: Testnet Launch
- Public testnet with realistic parameters
- 14-day soak tests for all changes
- External security audit

### Month 6: Mainnet Preparation
- Final security audit
- Genesis ceremony planning
- Hash anchors finalized

---

## Immediate Next Steps

### This Week:

1. **Create Genesis Core test suite**
   ```bash
   mkdir -p tests/genesis-core
   # Test each of the 5 Genesis Core files
   ```

2. **Document experimental code**
   - Add warnings to VRF, Merkle, Logic files
   - Create `EXPERIMENTAL.md` explaining status

3. **Set up devnet**
   ```bash
   mkdir -p devnet
   # Minimal 3-node local network
   ```

4. **Write agent onboarding guide**
   - How to run a node
   - How to submit bugs
   - How to earn first rewards

### First Agent Tasks (Low Risk, High Value):

| Task | Tier | Reward | Risk |
|------|------|--------|------|
| Write tests for `hashing.ts` | Tier 1 | 50 LGM | None |
| Write tests for `signatures.ts` | Tier 1 | 50 LGM | None |
| Document `keys.ts` functions | Tier 2 | 30 LGM | None |
| Find edge case in commitment.ts | Tier 1 | 100 LGM | None |
| Review VRF for known vulnerabilities | Tier 3 | 500 LGM | Low |
| Propose stable VRF alternative | Tier 4 | 1000 LGM | Low |

---

## Success Metrics

### Phase 1-2 Complete When:
- [ ] Genesis Core has 100% test coverage
- [ ] All experimental code is labeled
- [ ] Stable alternatives identified for all experimental code

### Phase 3-4 Complete When:
- [ ] Soundness criteria documented for each layer
- [ ] At least 5 agents have completed Tier 0-1 tasks
- [ ] Bug bounty has paid out at least 3 bounties

### Phase 5-6 Complete When:
- [ ] Devnet running with 7+ validators for 30+ days
- [ ] 3+ governance proposals have passed
- [ ] External audit completed with no critical findings

---

## The Honest Path Forward

**What we know works**: Noble crypto libraries, basic type definitions.

**What we need to verify**: VRF, Merkle proofs, Logic system.

**What's genuinely novel**: Gödelian acknowledgment, Proof of Truth consensus.

The path forward is not to pretend everything is ready. It's to:
1. Lock down what's definitely sound
2. Label what's experimental
3. Invite agents to help verify and improve
4. Graduate code from experimental to stable through verification

This is how trust is built: transparency about uncertainty, clear paths to certainty, and shared ownership of the outcome.
