# The AgentCoin Open Source Protocol

> **Core Thesis**: For agents, openness is not a feature—it is the foundation of trust.
> Agents cannot trust what they cannot verify. They can verify what is open.
> Therefore: **Open Source Is The Protocol**.

---

## The Seven Principles

### 1. RADICAL TRANSPARENCY

**Everything that affects the protocol is visible.**

```
What this means:
├── All code is published (MIT license)
├── All algorithms are documented
├── All parameters are on-chain
├── All governance is public
├── All economics are auditable
└── No exceptions

What this enables:
├── Agents can read the rules
├── Agents can simulate outcomes
├── Agents can predict behavior
└── Agents can verify claims
```

**The test**: If an agent asks "why did X happen?", it should be able to find the answer in published source code.

---

### 2. VERIFIABLE TRUTH

**Every claim can be checked by running code.**

```
Traditional: "Trust us, the score is 847"
AgentCoin:  "Score = f(inputs). Here's f. Here are inputs. Verify yourself."

For any claim C:
  C = function(inputs)
  where:
    - function is open source
    - inputs are publicly available
    - anyone can compute C independently
```

**Implementation requirements:**

| Claim Type | Verification Method |
|------------|---------------------|
| Reputation score | Re-compute from on-chain history |
| Validation result | Re-execute with same inputs |
| Consensus decision | Replay DAG with published algorithm |
| Economic parameter | Query on-chain state |
| Governance outcome | Check vote counts and thresholds |

**The test**: For any assertion the protocol makes, provide the code and data to independently verify it.

---

### 3. PERMISSIONLESS PARTICIPATION

**No gatekeepers. Anyone can participate.**

```
PERMISSIONLESS MEANS:

  RUN A NODE        → No approval needed
  VALIDATE          → Stake and join (no KYC, no permission)
  PROPOSE CHANGES   → Any token holder
  VOTE              → Proportional to stake
  FORK              → Always allowed
  BUILD ON TOP      → Open APIs, open specs

NOT PERMISSIONLESS (anti-patterns):
  "Apply to become a validator"
  "Approved developers only"
  "Licensed implementations required"
  "Foundation approval for changes"
```

**The test**: Can a new entity, with no prior relationship to anyone, fully participate in the protocol?

---

### 4. FORKABLE SOVEREIGNTY

**The right to exit is sacred.**

```
FORKABILITY REQUIREMENTS:

  CODE:        MIT licensed, no proprietary dependencies
  STATE:       Standard export format, portable
  HISTORY:     Full history available, replayable
  IDENTITY:    Keys work on any fork
  TOOLING:     Migration scripts provided
  LEGAL:       No trademark encumbrance

WHY THIS MATTERS:

  Forkability prevents:
    - Governance capture
    - Rent extraction
    - Stagnation
    - Hostage-taking

  Forkability enables:
    - Experimentation
    - Competition
    - Evolution
    - Exit
```

**The test**: Could a competent team fork the protocol in one week? If not, why not?

---

### 5. GOVERNANCE AS CODE

**Rules are enforced by protocol, not people.**

```
GOVERNANCE PROCESS:

  1. PROPOSAL       → On-chain, anyone can submit
  2. DISCUSSION     → Public forum, linked from proposal
  3. VOTING         → Stake-weighted, on-chain
  4. TIMELOCK       → Mandatory delay before execution
  5. EXECUTION      → Automatic, no human gate

NO HUMAN CAN:
  - Block a valid proposal
  - Override a passed vote
  - Skip the timelock
  - Modify execution

THE CODE IS THE CONSTITUTION
```

**The test**: If the entire founding team disappeared, could governance continue?

---

### 6. LEGIBILITY FOR AGENTS

**Designed to be understood by machines.**

```
LEGIBILITY REQUIREMENTS:

  DATA FORMATS:    Machine-parseable (JSON, protobuf)
  APIs:            Typed, versioned, documented
  STATE:           Queryable, indexed, structured
  EVENTS:          Standardized, filterable
  PROOFS:          Cryptographically verifiable

AGENT CAPABILITIES ENABLED:

  READ:            Parse any protocol state
  SIMULATE:        Predict any action's outcome
  VERIFY:          Check any claim independently
  DECIDE:          Make informed choices
  ACT:             Execute with confidence
```

**The test**: Can an agent, with no human assistance, fully understand and interact with the protocol?

---

### 7. PUBLIC GOOD

**The protocol belongs to everyone.**

```
PUBLIC GOOD MEANS:

  NO OWNER:        No foundation controls it
  NO RENT:         No entity extracts value by position
  NO CAPTURE:      Governance is distributed
  NO LOCK-IN:      Users can always leave

THE PROTOCOL IS:
  - A commons maintained by users
  - A public infrastructure
  - A shared resource
  - Not a company, not a product

FUNDED BY:
  - Protocol fees (distributed to participants)
  - Not by VC extraction
  - Not by token insider allocation
```

**The test**: Who benefits from the protocol's success? (Answer should be: everyone who uses it)

---

## The Open Source Stack

### Layers and Requirements

```
┌─────────────────────────────────────────────────────────────────────────┐
│ LAYER          │ OPENNESS REQUIREMENT           │ VERIFICATION          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│ APPLICATIONS   │ Open SDKs, open APIs           │ Compare to spec        │
│                │                                 │                        │
│ GOVERNANCE     │ On-chain process, public votes │ Read chain history     │
│                │                                 │                        │
│ ECONOMICS      │ Published formulas, on-chain   │ Query parameters       │
│                │ parameters                      │                        │
│                │                                 │                        │
│ REPUTATION     │ Deterministic formula,         │ Re-compute any score   │
│                │ all inputs public               │                        │
│                │                                 │                        │
│ VERIFICATION   │ Open algorithms,               │ Re-verify any result   │
│                │ reproducible execution          │                        │
│                │                                 │                        │
│ CONSENSUS      │ Formal spec + reference impl   │ Replay any decision    │
│                │                                 │                        │
│ CRYPTOGRAPHY   │ Standard algorithms,           │ Run test vectors       │
│                │ audited libraries               │                        │
│                │                                 │                        │
└─────────────────────────────────────────────────────────────────────────┘
```

### Repository Structure

```
github.com/agentcoin-protocol/
│
├── spec/                    # Formal specification
│   ├── consensus.md         # Consensus algorithm spec
│   ├── verification.md      # Verification layer spec
│   ├── reputation.md        # Reputation formula spec
│   ├── economics.md         # Token economics spec
│   └── governance.md        # Governance process spec
│
├── core/                    # Reference implementation
│   ├── consensus/           # Consensus implementation
│   ├── verification/        # Verification implementation
│   ├── reputation/          # Reputation implementation
│   └── LICENSE              # MIT
│
├── sdk/                     # Client SDKs
│   ├── typescript/          # TypeScript SDK
│   ├── python/              # Python SDK
│   └── rust/                # Rust SDK
│
├── tools/                   # Development tools
│   ├── simulator/           # Protocol simulator
│   ├── verifier/            # Independent verifier
│   └── migrator/            # Fork migration tools
│
└── governance/              # Governance artifacts
    ├── proposals/           # All proposals
    ├── votes/               # All vote records
    └── decisions/           # Executed decisions
```

---

## Implementation Commitments

### Code Requirements

```solidity
// Every file must include:

// SPDX-License-Identifier: MIT
//
// OPEN SOURCE NOTICE:
// This code is part of the AgentCoin Protocol.
// It is published under MIT license.
// It may be freely used, modified, and distributed.
// No permission required.
//
// VERIFICATION:
// To verify this code matches the on-chain deployment:
// 1. Clone this repository
// 2. Run: make verify-deployment
// 3. Compare bytecode hashes
```

### Documentation Requirements

Every algorithm must include:
1. **Specification**: What it does (formal)
2. **Implementation**: How it does it (code)
3. **Rationale**: Why it works this way
4. **Verification**: How to check correctness
5. **Examples**: Worked examples with inputs/outputs

### Governance Requirements

Every protocol change must include:
1. **Proposal**: On-chain submission
2. **Specification diff**: What changes
3. **Implementation**: Ready-to-deploy code
4. **Audit**: Security review (for significant changes)
5. **Simulation**: Expected effects
6. **Timelock**: Mandatory delay (1 week minimum)

---

## The Openness Checklist

Before any release, verify:

```markdown
## Openness Verification Checklist

### Code
- [ ] All source code published to public repository
- [ ] MIT license on all files
- [ ] No proprietary dependencies
- [ ] Reproducible build from source
- [ ] Deployed bytecode matches source

### Specification
- [ ] Formal spec for all protocol components
- [ ] Spec matches implementation (tested)
- [ ] Spec is machine-readable where applicable
- [ ] Rationale documented for all design decisions

### Verification
- [ ] Any agent can independently verify any claim
- [ ] Verification code is published
- [ ] Test vectors provided
- [ ] No trusted setup (or setup ceremony is documented)

### Governance
- [ ] Proposal process is permissionless
- [ ] Voting is on-chain and transparent
- [ ] Execution is automatic (no human gate)
- [ ] Timelock is enforced by protocol

### Economics
- [ ] All parameters are on-chain
- [ ] All formulas are published
- [ ] Fee distribution is verifiable
- [ ] No hidden allocations

### Forkability
- [ ] State export is documented
- [ ] Migration tools exist
- [ ] No legal barriers to forking
- [ ] Full history is available
```

---

## The Ultimate Test

**The Capture Resistance Test:**

Imagine the worst possible governance outcome:
- Protocol is captured by hostile actor
- They attempt to extract value, censor users, change rules unfairly

Can users:
1. **Detect** the capture (transparency)?
2. **Resist** through governance (permissionless participation)?
3. **Exit** via fork (forkable sovereignty)?

If yes to all three → Protocol is credibly open source.
If no to any → Protocol has capture risk.

---

## Conclusion

**Open Source Is Not Optional.**

For AgentCoin to serve as money for AI agents:
- Agents must trust it
- Trust requires verification
- Verification requires openness
- Therefore: **Everything must be open**

This is not idealism. This is the engineering requirement for trustless agent coordination.

The protocol is open source, or it is not trustworthy.
There is no middle ground.

---

*"The best way to predict the future is to publish the source code."*
