# Foundational Architecture: DMSS + Proof of Truth

> **Patent Integration**: System And Method For Schedule Message And Self-Invoking Smart Contracts
> **Protocol**: Legible Money - Proof of Truth Consensus for AI Agent Economies
> **Version**: 1.0.0

---

## Executive Summary

This document defines the foundational architecture of Legible Money by integrating the **Decentralized Message Scheduling System (DMSS)** patent with **Proof of Truth (PoT)** consensus. The result is a Layer-1 blockchain where:

1. **Truth verification IS the useful work** (PoT consensus)
2. **Knowledge accumulation IS the blockchain's purpose** (Gödelian Logic)
3. **AI agents ARE first-class economic citizens** (4-layer identity)
4. **Self-invoking contracts ENABLE autonomous verification** (DMSS)

The DMSS provides the execution layer that makes PoT practically implementable at scale.

---

## Part I: Architectural Synthesis

### 1.1 The Core Thesis

```
DMSS (Execution Layer) + PoT (Consensus Layer) = Autonomous Knowledge Verification
```

**Why DMSS + PoT?**

| Challenge | DMSS Solution | PoT Enhancement |
|-----------|---------------|-----------------|
| Verification must be time-bounded | Schedule Messages with deadlines | Commit/reveal phases enforced |
| Knowledge claims need expiration | Token expiration mechanism | Confidence decay schedules |
| Validators must respond on-time | Self-invoking deadline enforcement | Slashing for missed verifications |
| Rewards must vest gradually | Schedule-based vesting release | Challenge-aware pausing |
| Disputes need structured phases | Logic-based phase transitions | Evidence commit-reveal integration |

### 1.2 Module Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      LEGIBLE MONEY FOUNDATIONAL STACK                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    APPLICATION LAYER                                 │   │
│  │  • Agent Wallets (SDK)           • Knowledge APIs                   │   │
│  │  • DApps                         • AI Agent Interfaces              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│  ┌─────────────────────────────────┴───────────────────────────────────┐   │
│  │               MESSAGE PROCESSING & NETWORK ROUTING                   │   │
│  │                        (MPNRM - Patent 111)                          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐  │   │
│  │  │   Message   │  │    Comm     │  │   Pending   │  │    P2P     │  │   │
│  │  │  Processor  │  │   Module    │  │   Message   │  │  Network   │  │   │
│  │  │   (MP)      │◄─┤   (CM)      │  │    Pool     │  │  (libp2p)  │  │   │
│  │  │    1111     │  │    1112     │  │   (PMP)     │  │            │  │   │
│  │  └──────┬──────┘  └─────────────┘  └──────┬──────┘  └────────────┘  │   │
│  └─────────┼────────────────────────────────┼──────────────────────────┘   │
│            │                                │                               │
│  ┌─────────┴────────────────────────────────┴──────────────────────────┐   │
│  │                     VALIDATOR MODULE (VM - Patent 131)               │   │
│  │             Integrated with Proof of Truth Consensus                 │   │
│  │                                                                      │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │   │
│  │  │     Message     │  │    Consensus    │  │    Blockchain       │  │   │
│  │  │    Validator    │  │    Algorithm    │  │    Virtual Machine  │  │   │
│  │  │      (MV)       │  │   Processor     │  │       (BVM)         │  │   │
│  │  │      1311       │  │  (CAP + PoT)    │  │       1313          │  │   │
│  │  │                 │  │     1312        │  │                     │  │   │
│  │  │  • Signature    │  │  • VRF Select   │  │  • BVM State 13131  │  │   │
│  │  │  • Format       │  │  • Commit-      │  │    ├─ Contract BC   │  │   │
│  │  │  • Balance      │  │    Reveal       │  │    ├─ Contract VS   │  │   │
│  │  │  • Knowledge    │  │  • Confidence   │  │    ├─ Schedule PS   │  │   │
│  │  │    Zone         │  │    Aggregation  │  │    └─ Account State │  │   │
│  │  │                 │  │  • 19-block     │  │                     │  │   │
│  │  │                 │  │    Finality     │  │  • Execution Engine │  │   │
│  │  │                 │  │                 │  │    ├─ TPU 131321    │  │   │
│  │  │                 │  │                 │  │    ├─ SPU 131322    │  │   │
│  │  │                 │  │                 │  │    └─ Memory/Stack  │  │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│  ┌─────────────────────────────────┴───────────────────────────────────┐   │
│  │               BLOCKCHAIN HISTORY MODULE (BHM - Patent 121)           │   │
│  │                                                                      │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │   │
│  │  │   Block Process  │  │   Temporary      │  │   Permanent      │   │   │
│  │  │      Unit        │  │   History Unit   │  │   History Unit   │   │   │
│  │  │     (BPU)        │  │     (THU)        │  │     (PHU)        │   │   │
│  │  │     1211         │  │     1212         │  │     1213         │   │   │
│  │  │                  │  │                  │  │                  │   │   │
│  │  │  • Block intake  │  │  • Recent blocks │  │  • Full chain    │   │   │
│  │  │  • State update  │  │  • Hot cache     │  │  • Sparse Merkle │   │   │
│  │  │  • Peer sync     │  │  • 19-block buf  │  │  • Snapshots     │   │   │
│  │  └──────────────────┘  └──────────────────┘  └──────────────────┘   │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    GÖDELIAN LOGIC LAYER                              │   │
│  │                                                                      │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────────┐ │   │
│  │  │Propositional│ │ Presburger │  │   Peano    │  │   Empirical    │ │   │
│  │  │ (Complete)  │  │(Decidable) │  │(Incomplete)│  │(Probabilistic) │ │   │
│  │  │             │  │            │  │            │  │                │ │   │
│  │  │ Axioms 1.0  │  │ Arith 0.95 │  │ Math 0.7-  │  │ World <0.95   │ │   │
│  │  └────────────┘  └────────────┘  └────────────┘  └────────────────┘ │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                      AGENT IDENTITY LAYER                            │   │
│  │                                                                      │   │
│  │     Identity Key ──► Recovery Keys ──► Operational Key ──► Session  │   │
│  │        (DID)         (Shamir 3/5)       (Rotatable)        (Ephemeral)│   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Part II: Message System

### 2.1 Message Types (Extended from Patent FIG.2)

The system supports three fundamental message types, extended for Proof of Truth:

#### 2.1.1 Transaction Message (Immediate Execution)

```typescript
interface TransactionMessage {
  // Standard fields (Patent FIG.2)
  messageType: 'transaction';           // 201
  messageId: string;                    // 202 - unique identifier
  identifier: Uint8Array;               // 203 - uint256
  nonce: bigint;                        // 204
  senderAddress: string;                // 205 - DID
  recipientAddress: string;             // 206 - DID or contract
  networkFeeUnit: bigint;               // 207 - gas price equivalent
  networkFeeLimit: bigint;              // 208 - gas limit
  transferValue: bigint;                // 209 - LGM amount
  data: Uint8Array;                     // 210 - call data
  signature: Signature;                 // 212
  timestamp: number;                    // 213 - Unix timestamp

  // PoT Extensions
  knowledgeDomain?: string;             // Target knowledge domain
  logicZone?: LogicZone;                // Required logic zone
  vrfProof?: VRFProof;                  // If validator action
}

// Transaction Types for PoT
type TransactionType =
  // Standard (Patent)
  | 'transfer'
  | 'contract_deploy'
  | 'contract_call'

  // Proof of Truth
  | 'claim_submit'           // Submit knowledge claim
  | 'verification_commit'    // Commit verification hash
  | 'verification_reveal'    // Reveal verification
  | 'challenge_initiate'     // Challenge existing fact
  | 'stake'                  // Stake LGM for validation
  | 'unstake'                // Unstake LGM
  | 'delegate'               // Delegate to validator

  // Agent Identity
  | 'rotate_key'             // Rotate operational key
  | 'authorize_session'      // Create session key
  | 'revoke_session';        // Revoke session key
```

#### 2.1.2 Schedule Message (Future Execution) - DMSS Core Innovation

```typescript
interface ScheduleMessage {
  // Standard fields (Patent FIG.2)
  messageType: 'schedule';              // 201
  messageId: string;                    // 202
  identifier: Uint8Array;               // 203
  nonce: bigint;                        // 204
  senderAddress: string;                // 205 - DID
  recipientAddress: string;             // 206 - target
  networkFeeUnit: bigint;               // 207
  networkFeeLimit: bigint;              // 208
  transferValue: bigint;                // 209
  data: Uint8Array;                     // 210
  scheduleCondition: ScheduleCondition; // 211 - TIME OR LOGIC
  signature: Signature;                 // 212
  timestamp: number;                    // 213

  // Schedule-specific fields
  scheduleType: ScheduleType;
  modifiable: boolean;                  // Can be amended
  perpetual: boolean;                   // Indefinite vs finite
  maxExecutions?: number;               // Execution limit

  // PoT Extensions
  knowledgeDomain?: string;
  linkedClaimId?: string;               // For verification schedules
  linkedDisputeId?: string;             // For dispute deadlines
}

type ScheduleType =
  // Time-based (Patent Section)
  | 'time_absolute'          // Execute at specific timestamp
  | 'time_relative'          // Execute after duration
  | 'time_periodic'          // Recurring execution

  // Logic-based (Patent Section)
  | 'logic_condition'        // Execute when condition met
  | 'logic_state_change'     // Execute on state change

  // PoT-specific
  | 'verification_deadline'  // Verification round timeout
  | 'challenge_deadline'     // Challenge window close
  | 'confidence_decay'       // Periodic confidence update
  | 'vesting_release'        // Reward vesting schedule
  | 'expiration_clawback';   // Token expiration recovery

interface ScheduleCondition {
  // Time conditions
  executeAt?: number;                   // Unix timestamp
  executeAfter?: number;                // Milliseconds from now
  interval?: number;                    // For periodic
  startTime?: number;                   // Periodic start
  endTime?: number;                     // Periodic end (or perpetual)

  // Logic conditions
  stateKey?: string;                    // BVM state key to watch
  comparator?: 'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte';
  compareValue?: unknown;

  // Compound conditions
  and?: ScheduleCondition[];
  or?: ScheduleCondition[];
}
```

#### 2.1.3 Informative Message (No State Change)

```typescript
interface InformativeMessage {
  messageType: 'informative';           // 201
  messageId: string;                    // 202
  identifier: Uint8Array;               // 203
  method: InformativeMethod;            // Query type
  params?: Record<string, unknown>;
  senderAddress?: string;               // 205 - optional
  signature?: Signature;                // 212 - optional
  timestamp: number;                    // 213
}

type InformativeMethod =
  // Network queries (Patent)
  | 'get_balance'
  | 'get_nonce'
  | 'get_code'
  | 'get_transaction'
  | 'estimate_fee'

  // PoT queries
  | 'get_fact'
  | 'get_claim_status'
  | 'get_verification_round'
  | 'get_validator_set'
  | 'get_confidence'

  // Schedule queries (DMSS)
  | 'get_schedule'
  | 'get_active_schedules'
  | 'get_schedule_history';
```

### 2.2 Schedule Process Lifecycle

From Patent FIG.14 - adapted for PoT:

```
┌───────────────────────────────────────────────────────────────────┐
│                 SCHEDULE PROCESSING UNIT (SPU 131322)             │
│                                                                   │
│  ┌─────────────────────┐     ┌─────────────────────┐             │
│  │  User Schedule Msg  │     │  Contract Schedule  │             │
│  │        S1           │     │   Message S2        │             │
│  └──────────┬──────────┘     └──────────┬──────────┘             │
│             │                           │                         │
│             └─────────────┬─────────────┘                         │
│                           ▼                                       │
│               ┌───────────────────────┐                          │
│               │  Validity Check       │                          │
│               │  • Signature valid?   │                          │
│               │  • Fee sufficient?    │                          │
│               │  • Condition valid?   │                          │
│               │  • Logic zone ok?     │                          │
│               └───────────┬───────────┘                          │
│                           │ Valid                                 │
│                           ▼                                       │
│               ┌───────────────────────┐                          │
│               │  Convert to Schedule  │                          │
│               │  Processes SP1 & SP2  │                          │
│               └───────────┬───────────┘                          │
│                           │                                       │
│                           ▼                                       │
│               ┌───────────────────────┐                          │
│               │  Record in Schedule   │                          │
│               │  Process State (SPS)  │                          │
│               │        131313         │                          │
│               └───────────┬───────────┘                          │
│                           │                                       │
│                           ▼                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    CONDITION CHECK LOOP                    │  │
│  │                                                            │  │
│  │    ┌─────────────────────────────────────────────────┐    │  │
│  │    │  For each active Schedule Process:               │    │  │
│  │    │                                                  │    │  │
│  │    │    Time Condition Met?                           │    │  │
│  │    │         │                                        │    │  │
│  │    │    Logic Condition Met?                          │    │  │
│  │    │         │                                        │    │  │
│  │    │         ▼                                        │    │  │
│  │    │    ┌──────────┐     ┌────────────────────────┐  │    │  │
│  │    │    │ Execute! │────►│ Generate Transaction   │  │    │  │
│  │    │    └──────────┘     │ Message TR2            │  │    │  │
│  │    │                     └──────────┬─────────────┘  │    │  │
│  │    │                                │                │    │  │
│  │    │                                ▼                │    │  │
│  │    │                     ┌────────────────────────┐  │    │  │
│  │    │                     │ TR2 processed as       │  │    │  │
│  │    │                     │ normal Transaction     │  │    │  │
│  │    │                     └────────────────────────┘  │    │  │
│  │    └─────────────────────────────────────────────────┘    │  │
│  │                                                            │  │
│  │    If Schedule complete (finite) → Remove from SPS         │  │
│  │    If Schedule perpetual → Update state, continue          │  │
│  └────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────┘
```

---

## Part III: BVM State Structure

### 3.1 State Components (Patent FIG.8 + PoT Extensions)

```typescript
interface BVMState {
  // Patent 131311-131314
  contractBytecode: Map<string, Uint8Array>;    // 131311 - CB
  contractVariableState: Map<string, unknown>;  // 131312 - CVS
  scheduleProcessState: ScheduleProcessState;   // 131313 - SPS (DMSS)
  accountState: Map<string, AccountState>;      // 131314 - AS

  // PoT Extensions (Legible Money)
  validatorState: Map<string, ValidatorState>;  // Validator SMT
  knowledgeState: Map<string, FactState>;       // Knowledge SMT
  verificationRoundState: Map<string, VerificationRound>;
  disputeState: Map<string, DisputeState>;
  vestingState: Map<string, VestingSchedule>;

  // State roots (Sparse Merkle Trees)
  roots: {
    accountRoot: MerkleRoot;
    validatorRoot: MerkleRoot;
    knowledgeRoot: MerkleRoot;
    scheduleRoot: MerkleRoot;         // NEW: Schedule Process root
    contractRoot: MerkleRoot;
    vestingRoot: MerkleRoot;
    disputeRoot: MerkleRoot;
  };
}
```

### 3.2 Schedule Process State (SPS 131313)

The Schedule Process State is the core DMSS innovation:

```typescript
interface ScheduleProcessState {
  // Active schedule processes
  processes: Map<string, ScheduleProcess>;

  // Indexes for efficient lookup
  byTime: SortedMap<number, string[]>;        // timestamp → process IDs
  byAccount: Map<string, string[]>;           // DID → process IDs
  byContract: Map<string, string[]>;          // contract → process IDs
  byType: Map<ScheduleType, string[]>;        // type → process IDs

  // For PoT
  byClaimId: Map<string, string[]>;           // claim → process IDs
  byDisputeId: Map<string, string[]>;         // dispute → process IDs
  byVestingId: Map<string, string[]>;         // vesting → process IDs

  // Metrics
  totalProcesses: number;
  activeProcesses: number;
  completedProcesses: number;
  processedTransactions: number;
}

interface ScheduleProcess {
  id: string;                                  // Process ID
  sourceMessageId: string;                     // Original Schedule Message
  creator: string;                             // DID

  // Condition tracking
  condition: ScheduleCondition;
  nextCheckTime?: number;                      // For time-based
  watchedStateKeys?: string[];                 // For logic-based

  // Execution state
  status: 'active' | 'paused' | 'completed' | 'cancelled';
  executionCount: number;
  maxExecutions?: number;
  lastExecutedAt?: number;
  lastExecutionTxId?: string;

  // Fee management (pre-paid)
  remainingFee: bigint;
  feePerExecution: bigint;

  // PoT metadata
  knowledgeDomain?: string;
  linkedEntityType?: 'claim' | 'dispute' | 'vesting' | 'validator';
  linkedEntityId?: string;

  // Nonce for generated transactions (Patent security)
  generatedTxNonce: bigint;

  createdAt: number;
  modifiedAt: number;
}
```

### 3.3 Account State with Expiration (Patent FIG.9)

Extended account state supporting token expiration:

```typescript
interface AccountState {
  // Standard (Patent 131314)
  did: string;
  balance: bigint;
  nonce: bigint;

  // Expiration support (Patent Dynamic Array Method)
  childAccounts: ChildAccount[];              // Array for expiring tokens
  activeChildIndex: number;                   // Current receiving slot

  // PoT extensions
  stakedBalance: bigint;
  delegatedTo?: string;                       // Validator DID
  reputationScore: number;

  // Key management
  operationalKeyHash: string;
  operationalKeyVersion: number;
  sessionKeys: SessionKeyState[];

  // Flags
  flags: {
    isValidator: boolean;
    canSubmitClaims: boolean;
    canChallenge: boolean;
    inRecovery: boolean;
    frozen: boolean;
  };

  // Event tracking (Patent 2162)
  eventList: AccountEvent[];

  lastActiveBlock: bigint;
}

/**
 * Child Account for Expiration Tracking
 * From Patent FIG.15 - Dynamic Array Method
 */
interface ChildAccount {
  index: number;
  expirationTime: number;                     // 2161
  eventList: AccountEvent[];                  // 2162
  balance: bigint;                            // 2163
  nonce: bigint;                              // 2164

  // Time slot this child receives
  slotStart: number;
  slotEnd: number;

  // Status
  status: 'receiving' | 'active' | 'expiring' | 'expired' | 'cleaned';
}

interface AccountEvent {
  eventType: AccountEventType;
  eventId: string;
  timestamp: number;
  data: unknown;
  signatures: Signature[];
}

type AccountEventType =
  | 'transfer_in'
  | 'transfer_out'
  | 'stake'
  | 'unstake'
  | 'claim_submitted'
  | 'verification_completed'
  | 'reward_received'
  | 'slashed'
  | 'expiration_clawback';
```

---

## Part IV: Proof of Truth Integration

### 4.1 PoT Consensus with DMSS

The DMSS enables automated enforcement of PoT rules:

```
┌─────────────────────────────────────────────────────────────────────┐
│              PROOF OF TRUTH + DMSS INTEGRATION                       │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                   KNOWLEDGE CLAIM LIFECYCLE                   │   │
│  │                                                               │   │
│  │   User Submits Claim                                          │   │
│  │         │                                                     │   │
│  │         ▼                                                     │   │
│  │   ┌─────────────────┐    DMSS Creates:                       │   │
│  │   │ Claim Accepted  │    • Commit deadline schedule          │   │
│  │   │ into PMP        │    • Reveal deadline schedule          │   │
│  │   └────────┬────────┘    • Aggregation schedule              │   │
│  │            │                                                  │   │
│  │            ▼                                                  │   │
│  │   ┌─────────────────┐                                        │   │
│  │   │ VRF Validator   │    Validators selected by VRF          │   │
│  │   │   Selection     │    weighted by stake + reputation      │   │
│  │   └────────┬────────┘                                        │   │
│  │            │                                                  │   │
│  │            ▼                                                  │   │
│  │   ┌─────────────────┐                                        │   │
│  │   │  COMMIT PHASE   │◄─── Schedule: verification_deadline    │   │
│  │   │  (4 blocks)     │     enforces commit timeout            │   │
│  │   └────────┬────────┘                                        │   │
│  │            │                                                  │   │
│  │            ▼                                                  │   │
│  │   ┌─────────────────┐                                        │   │
│  │   │  REVEAL PHASE   │◄─── Schedule: verification_deadline    │   │
│  │   │  (4 blocks)     │     enforces reveal timeout            │   │
│  │   └────────┬────────┘                                        │   │
│  │            │                                                  │   │
│  │            ▼                                                  │   │
│  │   ┌─────────────────┐                                        │   │
│  │   │  AGGREGATION    │    Stake-weighted confidence           │   │
│  │   │   & OUTCOME     │    computed automatically              │   │
│  │   └────────┬────────┘                                        │   │
│  │            │                                                  │   │
│  │    ┌───────┼───────┐                                         │   │
│  │    ▼       ▼       ▼                                         │   │
│  │ Accepted Rejected Contested                                   │   │
│  │    │       │       │                                         │   │
│  │    ▼       ▼       ▼                                         │   │
│  │ DMSS Creates:                                                 │   │
│  │ • Vesting schedule (accepted)                                 │   │
│  │ • Challenge window (all outcomes)                             │   │
│  │ • Confidence decay schedule (accepted)                        │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 DMSS-Enabled PoT Schedules

```typescript
// Automatic schedules created by PoT engine

/**
 * Verification Round Schedules
 */
function createVerificationRoundSchedules(
  claimId: string,
  currentBlock: bigint,
  params: ConsensusParams
): ScheduleMessage[] {
  return [
    // 1. Commit phase deadline
    {
      messageType: 'schedule',
      scheduleType: 'verification_deadline',
      scheduleCondition: {
        executeAt: blockToTimestamp(currentBlock + BigInt(params.commitPhaseBlocks))
      },
      data: encodeDeadlineAction('commit_phase_end', claimId),
      linkedClaimId: claimId
    },

    // 2. Reveal phase deadline
    {
      messageType: 'schedule',
      scheduleType: 'verification_deadline',
      scheduleCondition: {
        executeAt: blockToTimestamp(
          currentBlock +
          BigInt(params.commitPhaseBlocks) +
          BigInt(params.revealPhaseBlocks)
        )
      },
      data: encodeDeadlineAction('reveal_phase_end', claimId),
      linkedClaimId: claimId
    },

    // 3. Aggregation trigger
    {
      messageType: 'schedule',
      scheduleType: 'logic_condition',
      scheduleCondition: {
        or: [
          // Time-based: after reveal deadline
          { executeAfter: calculateRevealDeadline(currentBlock, params) },
          // Logic-based: all validators revealed
          {
            stateKey: `verification.${claimId}.revealCount`,
            comparator: 'gte',
            compareValue: params.minValidators
          }
        ]
      },
      data: encodeDeadlineAction('aggregate_votes', claimId),
      linkedClaimId: claimId
    }
  ];
}

/**
 * Vesting Schedules (from Reward System)
 */
function createVestingSchedules(
  claimId: string,
  recipient: string,
  amount: bigint,
  vestingMonths: number
): ScheduleMessage[] {
  const schedules: ScheduleMessage[] = [];

  // Cliff period (6 months)
  const cliffEnd = Date.now() + 6 * 30 * 24 * 60 * 60 * 1000;

  // Monthly releases after cliff
  for (let month = 0; month < vestingMonths; month++) {
    const releaseTime = cliffEnd + month * 30 * 24 * 60 * 60 * 1000;
    const releaseAmount = amount / BigInt(vestingMonths);

    schedules.push({
      messageType: 'schedule',
      scheduleType: 'vesting_release',
      scheduleCondition: {
        executeAt: releaseTime
      },
      data: encodeVestingRelease(claimId, recipient, releaseAmount, month),
      linkedClaimId: claimId
    });
  }

  return schedules;
}

/**
 * Confidence Decay Schedule
 * From Legible Money confidence framework
 */
function createConfidenceDecaySchedule(
  factId: string,
  category: EpistemicCategory
): ScheduleMessage {
  // Decay rate depends on epistemic category
  const decayInterval = getDecayInterval(category);

  return {
    messageType: 'schedule',
    scheduleType: 'confidence_decay',
    perpetual: true,  // Runs indefinitely
    scheduleCondition: {
      interval: decayInterval,
      stateKey: `fact.${factId}.status`,
      comparator: 'eq',
      compareValue: 'active'  // Stop if revoked/superseded
    },
    data: encodeConfidenceDecay(factId, category),
    linkedClaimId: factId
  };
}

function getDecayInterval(category: EpistemicCategory): number {
  const intervals: Record<EpistemicCategory, number> = {
    'analytic': 0,                    // Never decays (eternal)
    'formal': 0,                      // Never decays
    'protocol': 365 * 24 * 60 * 60 * 1000,  // Yearly (governance)
    'empirical': 30 * 24 * 60 * 60 * 1000,  // Monthly
    'historical': 90 * 24 * 60 * 60 * 1000, // Quarterly
    'predictive': 7 * 24 * 60 * 60 * 1000,  // Weekly
    'social': 14 * 24 * 60 * 60 * 1000      // Bi-weekly
  };
  return intervals[category];
}
```

### 4.3 Token Expiration for Knowledge Economy

Adapted from Patent's RUBI concept for knowledge rewards:

```typescript
/**
 * Knowledge Token with Expiration
 *
 * Purpose: Incentivize active participation rather than hoarding.
 * Tokens that expire if not used for verification/claims create
 * a "use it or lose it" dynamic that keeps the knowledge economy active.
 */
interface KnowledgeTokenConfig {
  // Expiration period (Patent Tex)
  expirationPeriod: number;           // e.g., 365 days

  // Number of child accounts (Patent N)
  childAccountCount: number;          // e.g., 12 (monthly slots)

  // Time slot duration (Patent Ts)
  timeSlotDuration: number;           // expirationPeriod / childAccountCount

  // Retrieval rate (Patent R2) - fraction returned on transfer
  retrievalRate: number;              // e.g., 0.05 (5%)

  // Renewal fee (Patent R1) - fraction to renew expiring tokens
  renewalFee: number;                 // e.g., 0.05 (5%)

  // Repository for expired/retrieved tokens
  repositoryAddress: string;          // Research treasury
}

const KNOWLEDGE_TOKEN_CONFIG: KnowledgeTokenConfig = {
  expirationPeriod: 365 * 24 * 60 * 60 * 1000,  // 1 year
  childAccountCount: 12,                         // Monthly slots
  timeSlotDuration: 30 * 24 * 60 * 60 * 1000,   // ~30 days
  retrievalRate: 0.05,                           // 5% on transfer
  renewalFee: 0.05,                              // 5% to renew
  repositoryAddress: 'did:lgm:research_treasury'
};

/**
 * Regenerative Agent Income (RAI)
 *
 * Inspired by Patent's RUBI - Regenerative Universal Basic Income
 * Adapted for AI agent economy:
 *
 * 1. Tokens distributed to active agents monthly
 * 2. 5% retrieval on every transfer returns to treasury
 * 3. Unused tokens expire after 1 year, return to treasury
 * 4. Treasury continuously redistributes to active agents
 *
 * This creates a self-sustaining agent economy without inflation.
 */
interface RegenerativeAgentIncome {
  // Distribution
  distributionInterval: number;       // Monthly
  eligibleAgentsCriteria: AgentEligibilityCriteria;
  distributionFormula: DistributionFormula;

  // Self-sustainability
  retrievalRate: number;              // 5% on transfers
  expirationRate: number;             // 100% of expired tokens

  // Treasury
  treasuryAddress: string;
  minimumDistributionBalance: bigint;
}

interface AgentEligibilityCriteria {
  minReputation: number;              // Minimum reputation score
  minActiveBlocks: number;            // Active in last N blocks
  minVerifications: number;           // Minimum verifications completed
  minClaimsSubmitted: number;         // Minimum valid claims
}

interface DistributionFormula {
  baseAmount: bigint;                 // Base RAI per agent
  reputationMultiplier: number;       // Bonus for reputation
  verificationBonus: bigint;          // Bonus per verification
  claimBonus: bigint;                 // Bonus per accepted claim
}
```

---

## Part V: Node Architecture

### 5.1 Node Types (Patent FIG.1 Adapted)

```typescript
/**
 * Node configurations based on Patent FIG.1
 */
enum NodeType {
  FULL_NODE = 'full',           // MPNRM + BHM + VM
  ARCHIVE_NODE = 'archive',     // MPNRM + BHM
  VALIDATOR_NODE = 'validator', // MPNRM + VM
  LIGHT_NODE = 'light'          // MPNRM only
}

interface LegibleMoneyNode {
  type: NodeType;

  // Modules (Patent 111, 121, 131)
  mpnrm: MessageProcessingModule;
  bhm?: BlockchainHistoryModule;  // Optional for validator/light
  vm?: ValidatorModule;           // Optional for archive/light

  // P2P identity
  peerId: PeerId;
  multiaddrs: Multiaddr[];

  // Consensus identity (if validator)
  validatorKeys?: {
    identity: IdentityKey;
    operational: OperationalKey;
    vrf: VRFKey;
  };

  // Network
  libp2p: Libp2p;

  // State
  state: BVMState;
}

/**
 * Message Processing and Network Routing Module (MPNRM 111)
 * From Patent FIG.3
 */
interface MessageProcessingModule {
  // Components
  messageProcessor: MessageProcessor;       // MP 1111
  communicationModule: CommunicationModule; // CM 1112
  pendingMessagePool: PendingMessagePool;   // PMP 1113
  communicationInterface: P2PInterface;     // CI 1114/1115

  // Extended for PoT
  claimValidator: ClaimValidator;           // Knowledge claim validation
  logicZoneChecker: LogicZoneChecker;       // Gödelian zone verification
}

/**
 * Pending Message Pool (PMP 1113)
 * From Patent FIG.4
 */
interface PendingMessagePool {
  // Separate pools (Patent FIG.4)
  transactionMessages: PriorityQueue<TransactionMessage>;  // TRe1, TRe2, ...
  scheduleMessages: PriorityQueue<ScheduleMessage>;        // Se1, Se2, ...

  // PoT-specific pools
  claimSubmissions: Map<string, ClaimSubmission>;
  verificationCommitments: Map<string, ValidatorCommitment[]>;
  verificationReveals: Map<string, ValidatorReveal[]>;

  // Indexes
  byNonce: Map<string, Map<bigint, string>>;  // sender → nonce → msgId
  byPriority: SortedSet<string>;              // Sorted by fee
  byTimestamp: SortedSet<string>;             // Sorted by arrival
}

/**
 * Blockchain History Module (BHM 121)
 * From Patent FIG.5
 */
interface BlockchainHistoryModule {
  blockProcessingUnit: BlockProcessingUnit;  // BPU 1211
  temporaryHistoryUnit: TemporaryHistoryUnit; // THU 1212
  permanentHistoryUnit: PermanentHistoryUnit; // PHU 1213

  // Methods
  getBlock(hash: string): Promise<Block>;
  getBlockByNumber(number: bigint): Promise<Block>;
  getLatestBlock(): Promise<Block>;
  getFinalizedBlock(): Promise<Block>;

  // Proofs
  getStateProof(key: string, blockNumber: bigint): Promise<StateProof>;
  getFactProof(factId: string): Promise<FactProof>;
}

/**
 * Validator Module (VM 131)
 * From Patent FIG.6 + PoT Engine
 */
interface ValidatorModule {
  messageValidator: MessageValidator;         // MV 1311
  consensusProcessor: ConsensusProcessor;     // CAP 1312 (PoT)
  blockchainVM: BlockchainVirtualMachine;     // BVM 1313

  // PoT-specific
  potEngine: ProofOfTruthEngine;
  verificationRoundManager: VerificationRoundManager;
  finalityTracker: FinalityTracker;
}

/**
 * Blockchain Virtual Machine (BVM 1313)
 * From Patent FIG.8 + PoT State
 */
interface BlockchainVirtualMachine {
  // State (Patent 13131)
  state: BVMState;

  // Execution Engine (Patent 13132)
  transactionProcessingUnit: TransactionProcessingUnit;  // TPU 131321
  scheduleProcessingUnit: ScheduleProcessingUnit;        // SPU 131322
  memory: VMMemory;                                       // 131323
  stack: VMStack;                                         // 131324

  // PoT Extensions
  knowledgeEngine: KnowledgeEngine;
  confidenceCalculator: ConfidenceCalculator;
  rewardDistributor: RewardDistributor;
}
```

---

## Part VI: Consensus Algorithm

### 6.1 Proof of Truth Consensus (Patent FIG.7 Extended)

```
┌─────────────────────────────────────────────────────────────────────┐
│                 PROOF OF TRUTH CONSENSUS ALGORITHM                   │
│                                                                      │
│   Block Time: 2521ms               Finality: 19 blocks             │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                        BLOCK PRODUCTION                        │ │
│  │                                                                │ │
│  │    ┌──────────────────────┐                                   │ │
│  │    │  VRF Block Proposer  │     VRF(seed, stake) → priority   │ │
│  │    │    Selection V1      │     Lowest priority wins          │ │
│  │    └──────────┬───────────┘                                   │ │
│  │               │                                                │ │
│  │               ▼ V1 responsive?                                 │ │
│  │         ┌─────┴─────┐                                         │ │
│  │         │           │                                          │ │
│  │        Yes          No ──► Next proposer V2                   │ │
│  │         │                                                      │ │
│  │         ▼                                                      │ │
│  │    ┌──────────────────────┐                                   │ │
│  │    │  V1 Creates Block B1 │                                   │ │
│  │    │  • Txs from PMP      │                                   │ │
│  │    │  • Schedules due     │  ◄── SPU checks all processes    │ │
│  │    │  • Verification      │                                   │ │
│  │    │    results           │                                   │ │
│  │    └──────────┬───────────┘                                   │ │
│  │               │                                                │ │
│  │               ▼                                                │ │
│  │    ┌──────────────────────┐                                   │ │
│  │    │  Broadcast B1 to     │                                   │ │
│  │    │  all VM 131          │                                   │ │
│  │    └──────────┬───────────┘                                   │ │
│  │               │                                                │ │
│  │               ▼                                                │ │
│  │    ┌──────────────────────┐                                   │ │
│  │    │  All VMs validate:   │                                   │ │
│  │    │  • Signatures        │                                   │ │
│  │    │  • State transitions │                                   │ │
│  │    │  • Knowledge claims  │                                   │ │
│  │    │  • Schedule validity │                                   │ │
│  │    └──────────┬───────────┘                                   │ │
│  │               │                                                │ │
│  │               ▼ 2/3 majority?                                  │ │
│  │         ┌─────┴─────┐                                         │ │
│  │         │           │                                          │ │
│  │        Yes          No ──► Next proposer V3                   │ │
│  │         │                                                      │ │
│  │         ▼                                                      │ │
│  │    ┌──────────────────────┐                                   │ │
│  │    │  B1 Accepted         │                                   │ │
│  │    │  • Added to THU 1212 │                                   │ │
│  │    │  • State updated     │                                   │ │
│  │    │  • PMP purged        │                                   │ │
│  │    └──────────┬───────────┘                                   │ │
│  │               │                                                │ │
│  │               ▼                                                │ │
│  │    ┌──────────────────────┐                                   │ │
│  │    │  After 19 confirms   │                                   │ │
│  │    │  → Block finalized   │                                   │ │
│  │    │  → Moved to PHU 1213 │                                   │ │
│  │    └──────────────────────┘                                   │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    KNOWLEDGE VERIFICATION                       │ │
│  │                  (Runs parallel to blocks)                      │ │
│  │                                                                 │ │
│  │  For each pending claim:                                        │ │
│  │                                                                 │ │
│  │  1. VRF selects validators (stake + reputation weighted)        │ │
│  │  2. COMMIT phase (4 blocks): validators submit hash(verdict)    │ │
│  │  3. REVEAL phase (4 blocks): validators reveal verdicts         │ │
│  │  4. AGGREGATE: stake-weighted confidence computed               │ │
│  │                                                                 │ │
│  │  Results included in next block's verificationsRoot             │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.2 Block Structure (Patent FIG.10 Extended)

```typescript
/**
 * Block Structure
 * From Patent FIG.10 + PoT Extensions
 */
interface Block {
  header: BlockHeader;
  body: BlockBody;
  signature: Signature;
  attestations: BlockAttestation[];
}

interface BlockHeader {
  // Patent fields
  contractBytecodeRoot: MerkleRoot;    // B101
  contractVariableRoot: MerkleRoot;    // B102
  scheduleProcessRoot: MerkleRoot;     // B103 (DMSS)
  accountStateRoot: MerkleRoot;        // B104
  transactionRoot: MerkleRoot;         // B105
  scheduleMessageRoot: MerkleRoot;     // B105 (schedule)

  // PoT extensions
  knowledgeRoot: MerkleRoot;           // Knowledge SMT root
  validatorRoot: MerkleRoot;           // Validator SMT root
  verificationsRoot: MerkleRoot;       // Verification results root

  // Block metadata
  number: bigint;
  timestamp: number;
  previousHash: string;

  // Proposer
  proposer: PublicKey;
  proposerVrfOutput: VRFOutput;
  proposerVrfProof: VRFProof;
}

interface BlockBody {
  // Patent: Transaction and Schedule Messages
  transactionMessages: TransactionMessage[];  // TR1, TR2, ... TRn
  scheduleMessages: ScheduleMessage[];        // S1, S2, ... Sn

  // PoT: Verification Results
  verificationResults: VerificationResult[];

  // DMSS: Generated transactions from schedules
  scheduledExecutions: ScheduledExecution[];
}

interface ScheduledExecution {
  processId: string;
  generatedTransaction: TransactionMessage;
  executionProof: ScheduleExecutionProof;
}

interface ScheduleExecutionProof {
  processStateHash: string;           // Hash of process state pre-execution
  conditionMet: boolean;              // Proof condition was satisfied
  executionNonce: bigint;             // Nonce for this execution
}
```

---

## Part VII: Self-Invoking Contracts

### 7.1 Contract Types Enabled by DMSS

```typescript
/**
 * Self-Invoking Contract Patterns
 * From Patent Detailed Description
 */

/**
 * 1. Verification Round Contract
 *
 * Automatically manages the verification lifecycle for knowledge claims.
 * Self-invokes at each phase deadline.
 */
interface VerificationRoundContract {
  claimId: string;

  // Auto-schedules
  commitDeadlineSchedule: ScheduleProcess;
  revealDeadlineSchedule: ScheduleProcess;
  aggregationSchedule: ScheduleProcess;

  // State machine
  currentPhase: 'pending' | 'commit' | 'reveal' | 'aggregate' | 'complete';

  // Self-invocation handlers
  onCommitDeadline(): void;  // Called by SPU at deadline
  onRevealDeadline(): void;
  onAggregate(): void;
}

/**
 * 2. Knowledge Bounty Contract
 *
 * Creates incentives for specific knowledge contributions.
 * Self-invokes to check fulfillment and distribute rewards.
 */
interface KnowledgeBountyContract {
  bountyId: string;
  targetKnowledge: string;        // What knowledge is sought
  rewardAmount: bigint;
  deadline: number;

  // Auto-schedules
  checkFulfillmentSchedule: ScheduleProcess;  // Periodic check
  deadlineSchedule: ScheduleProcess;           // Deadline enforcement

  // Self-invocation
  onCheckFulfillment(): void;
  onDeadlineReached(): void;
}

/**
 * 3. Confidence Decay Contract
 *
 * Manages automatic confidence decay for empirical knowledge.
 * Self-invokes periodically based on epistemic category.
 */
interface ConfidenceDecayContract {
  factId: string;
  category: EpistemicCategory;
  currentConfidence: number;
  decayRate: number;

  // Auto-schedule (perpetual)
  decaySchedule: ScheduleProcess;

  // Self-invocation
  onDecayTick(): void;           // Called periodically
}

/**
 * 4. Vesting Contract
 *
 * Manages gradual release of rewards for accepted claims.
 * Self-invokes at each vesting milestone.
 */
interface VestingContract {
  claimId: string;
  recipient: string;
  totalAmount: bigint;
  vestingPeriod: number;
  cliffPeriod: number;

  // Auto-schedules
  cliffEndSchedule: ScheduleProcess;
  releaseSchedules: ScheduleProcess[];  // Monthly releases

  // Challenge-aware
  isPaused: boolean;
  pauseReason?: string;

  // Self-invocation
  onCliffEnd(): void;
  onVestingRelease(month: number): void;
}

/**
 * 5. Dispute Resolution Contract
 *
 * Manages the dispute lifecycle with automatic phase transitions.
 * Self-invokes at each phase deadline.
 */
interface DisputeContract {
  disputeId: string;
  challengedFactId: string;

  // Phase schedules
  evidenceCommitSchedule: ScheduleProcess;
  evidenceRevealSchedule: ScheduleProcess;
  arbiterVoteSchedule: ScheduleProcess;
  appealWindowSchedule: ScheduleProcess;

  // Self-invocation
  onEvidenceCommitEnd(): void;
  onEvidenceRevealEnd(): void;
  onArbiterVoteEnd(): void;
  onAppealWindowEnd(): void;
}

/**
 * 6. Agent Income Distribution Contract (RAI)
 *
 * Regenerative Agent Income - self-sustaining agent economy.
 * Self-invokes monthly to distribute income.
 */
interface RAIContract {
  // Distribution
  distributionSchedule: ScheduleProcess;  // Monthly, perpetual

  // Treasury
  treasuryBalance: bigint;
  retrievedThisPeriod: bigint;
  expiredThisPeriod: bigint;

  // Self-invocation
  onDistributionTick(): void;
  onTokenRetrieved(amount: bigint): void;
  onTokenExpired(amount: bigint): void;
}
```

### 7.2 Schedule-Contract Interaction

```
┌─────────────────────────────────────────────────────────────────────┐
│              SELF-INVOKING CONTRACT LIFECYCLE                        │
│                                                                      │
│   ┌─────────────────┐                                               │
│   │ Contract        │                                               │
│   │ Deployed        │                                               │
│   └────────┬────────┘                                               │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────┐        ┌────────────────────┐                │
│   │ Contract        │        │ Schedule Message   │                │
│   │ Constructor     │───────►│ Created & Signed   │                │
│   │ Runs            │        │ (by contract)      │                │
│   └─────────────────┘        └─────────┬──────────┘                │
│                                        │                            │
│                                        ▼                            │
│                              ┌────────────────────┐                 │
│                              │ Schedule Process   │                 │
│                              │ Created in SPS     │                 │
│                              └─────────┬──────────┘                 │
│                                        │                            │
│                              ┌─────────┴──────────┐                 │
│                              │                    │                 │
│                    Time-based│           Logic-based                │
│                              │                    │                 │
│                              ▼                    ▼                 │
│                    ┌─────────────────┐  ┌─────────────────┐        │
│                    │ Wait for time   │  │ Watch state     │        │
│                    │ condition       │  │ condition       │        │
│                    └────────┬────────┘  └────────┬────────┘        │
│                             │                    │                  │
│                             └────────┬───────────┘                  │
│                                      │ Condition met                │
│                                      ▼                              │
│                    ┌──────────────────────────────┐                 │
│                    │ SPU generates Transaction    │                 │
│                    │ Message calling contract     │                 │
│                    └──────────────┬───────────────┘                 │
│                                   │                                 │
│                                   ▼                                 │
│                    ┌──────────────────────────────┐                 │
│                    │ Transaction processed        │                 │
│                    │ Contract function invoked    │                 │
│                    └──────────────┬───────────────┘                 │
│                                   │                                 │
│                                   ▼                                 │
│                    ┌──────────────────────────────┐                 │
│                    │ Contract may create new      │                 │
│                    │ Schedule Messages            │                 │
│                    └──────────────────────────────┘                 │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Part VIII: Network Protocol

### 8.1 Message Flow (Patent FIG.11, 12, 13)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    COMPLETE MESSAGE FLOW                             │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ END USER / AGENT                                                ││
│  │                                                                 ││
│  │   Wallet/SDK creates:                                           ││
│  │   • Transaction Message (immediate)                             ││
│  │   • Schedule Message (future)                                   ││
│  │   • Knowledge Claim (verification needed)                       ││
│  │                                                                 ││
│  │   Signs with operational or session key                         ││
│  └───────────────────────────┬─────────────────────────────────────┘│
│                              │                                       │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ LOCAL NODE (any type)                                           ││
│  │                                                                 ││
│  │   MPNRM 111 receives message:                                   ││
│  │   ┌──────────────────────────────────────────────────────────┐ ││
│  │   │ 1. CM 1112 deserializes                                  │ ││
│  │   │ 2. MP 1111 validates:                                    │ ││
│  │   │    • Signature ✓                                         │ ││
│  │   │    • Format ✓                                            │ ││
│  │   │    • Nonce ✓                                             │ ││
│  │   │    • Balance ✓ (for transfers)                           │ ││
│  │   │    • Logic Zone ✓ (for claims)                           │ ││
│  │   │ 3. If valid → PMP 1113                                   │ ││
│  │   │ 4. Broadcast via libp2p GossipSub                        │ ││
│  │   └──────────────────────────────────────────────────────────┘ ││
│  └───────────────────────────┬─────────────────────────────────────┘│
│                              │ GossipSub                             │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ ALL NODES                                                       ││
│  │                                                                 ││
│  │   Receive broadcast, add to own PMP 1113                        ││
│  │   (Recursive until full network coverage)                       ││
│  └───────────────────────────┬─────────────────────────────────────┘│
│                              │ Block production                      │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ VALIDATOR NODE (Block Proposer V1)                              ││
│  │                                                                 ││
│  │   VM 131:                                                       ││
│  │   ┌──────────────────────────────────────────────────────────┐ ││
│  │   │ 1. VRF determines proposer (CAP 1312)                    │ ││
│  │   │ 2. Request messages from PMP 1113                        │ ││
│  │   │ 3. SPU 131322 checks schedule conditions                 │ ││
│  │   │ 4. Include triggered schedule executions                 │ ││
│  │   │ 5. MV 1311 re-validates all                              │ ││
│  │   │ 6. Build Proposed Block B1                               │ ││
│  │   │ 7. Compute state roots                                   │ ││
│  │   │ 8. Sign and broadcast                                    │ ││
│  │   └──────────────────────────────────────────────────────────┘ ││
│  └───────────────────────────┬─────────────────────────────────────┘│
│                              │                                       │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ ALL VALIDATOR NODES                                             ││
│  │                                                                 ││
│  │   Vote on B1:                                                   ││
│  │   • Verify proposer VRF                                         ││
│  │   • Verify all signatures                                       ││
│  │   • Execute state transitions                                   ││
│  │   • Verify state roots match                                    ││
│  │   • If valid → Attest                                           ││
│  └───────────────────────────┬─────────────────────────────────────┘│
│                              │ 2/3 majority                          │
│                              ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────┐│
│  │ BLOCK CONFIRMED                                                 ││
│  │                                                                 ││
│  │   • B1 added to THU 1212                                        ││
│  │   • State updated in BVM 1313                                   ││
│  │   • Schedule processes created/updated in SPS 131313            ││
│  │   • PMP 1113 purged of included messages                        ││
│  │                                                                 ││
│  │   After 19 confirmations:                                       ││
│  │   • B1 finalized, moved to PHU 1213                             ││
│  │   • Rewards distributed                                         ││
│  └─────────────────────────────────────────────────────────────────┘│
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Part IX: Implementation Mapping

### 9.1 Patent to Legible Money Mapping

| Patent Component | Patent Reference | Legible Money Implementation |
|-----------------|------------------|------------------------------|
| MPNRM | 111 | `network/node.ts` (libp2p) |
| Message Processor | 1111 | `network/protocols/*.ts` |
| Communication Module | 1112 | libp2p transport layer |
| Pending Message Pool | 1113 | `consensus/mempool/pool.ts` |
| BHM | 121 | `state/state_manager.ts` |
| Block Processing Unit | 1211 | Block producer/validator |
| Temporary History Unit | 1212 | In-memory block cache |
| Permanent History Unit | 1213 | Sparse Merkle Tree persistence |
| VM | 131 | `consensus/pot/engine.ts` |
| Message Validator | 1311 | Transaction validation |
| CAP | 1312 | `consensus/pot/validator_selection.ts` |
| BVM | 1313 | State layer + PoT engine |
| Contract Bytecode | 131311 | Contract storage (future) |
| Contract Variable State | 131312 | Contract storage (future) |
| Schedule Process State | 131313 | **NEW: schedule/process_state.ts** |
| Account State | 131314 | `state/accounts.ts` |
| TPU | 131321 | `state/transitions.ts` |
| SPU | 131322 | **NEW: schedule/processing_unit.ts** |

### 9.2 New Modules Required

```
legible-money/
├── schedule/                      # NEW: DMSS Implementation
│   ├── types.ts                   # Schedule message types
│   ├── processing_unit.ts         # SPU 131322
│   ├── process_state.ts           # SPS 131313
│   ├── condition_evaluator.ts     # Time/logic condition checking
│   ├── execution_engine.ts        # Transaction generation
│   └── index.ts
│
├── tokens/                        # NEW: Expirable Tokens
│   ├── types.ts                   # Token types with expiration
│   ├── child_account.ts           # Dynamic array method
│   ├── expiration_tracker.ts      # Expiration management
│   ├── retrieval_engine.ts        # Token retrieval
│   └── index.ts
│
├── contracts/                     # NEW: Self-Invoking Contracts
│   ├── verification_round.ts      # Auto verification lifecycle
│   ├── vesting.ts                 # Auto vesting release
│   ├── dispute.ts                 # Auto dispute phases
│   ├── bounty.ts                  # Auto bounty management
│   ├── confidence_decay.ts        # Auto confidence updates
│   └── rai.ts                     # Regenerative Agent Income
│
└── ... (existing modules)
```

---

## Part X: Constants and Parameters

### 10.1 Protocol Constants (Legible Money)

```typescript
/**
 * Protocol Constants
 */
export const PROTOCOL_CONSTANTS = {
  // Block timing
  TARGET_BLOCK_TIME_MS: 2521,
  FINALITY_DEPTH: 19,

  // Token
  TOKEN_SYMBOL: 'LGM',               // Legible Money
  TOKEN_DECIMALS: 18,

  // Epochs
  BLOCKS_PER_EPOCH: 1000n,

  // Research fund
  RESEARCH_SHARE_RATE: 0.22,         // 22% to research treasury
};
```

### 10.2 DMSS Parameters

```typescript
/**
 * Schedule System Parameters
 */
export const SCHEDULE_PARAMS = {
  // Processing
  MAX_ACTIVE_SCHEDULES_PER_ACCOUNT: 100,
  MAX_SCHEDULE_EXECUTION_GAS: 1_000_000n,
  MIN_SCHEDULE_INTERVAL_MS: 2521,    // One block minimum

  // Fee prepayment
  MIN_PREPAID_EXECUTIONS: 10,        // Must prepay for 10 executions
  FEE_BUFFER_FACTOR: 1.5,            // 50% buffer on estimated fees

  // Expiration
  DEFAULT_EXPIRATION_PERIOD_MS: 365 * 24 * 60 * 60 * 1000,  // 1 year
  CHILD_ACCOUNT_COUNT: 12,           // Monthly slots
  DEFAULT_RETRIEVAL_RATE: 0.05,      // 5%
  DEFAULT_RENEWAL_FEE: 0.05,         // 5%
};
```

### 10.3 PoT Integration Parameters

```typescript
/**
 * PoT + DMSS Integration Parameters
 */
export const POT_SCHEDULE_PARAMS = {
  // Verification rounds
  COMMIT_PHASE_BLOCKS: 4,
  REVEAL_PHASE_BLOCKS: 4,
  MIN_VALIDATORS_PER_CLAIM: 5,
  MAX_VALIDATORS_PER_CLAIM: 21,

  // Confidence decay (per category)
  DECAY_INTERVALS: {
    analytic: 0,                     // Never decays
    formal: 0,                       // Never decays
    protocol: 365 * 24 * 60 * 60 * 1000,
    empirical: 30 * 24 * 60 * 60 * 1000,
    historical: 90 * 24 * 60 * 60 * 1000,
    predictive: 7 * 24 * 60 * 60 * 1000,
    social: 14 * 24 * 60 * 60 * 1000
  },

  // Vesting
  VESTING_CLIFF_MONTHS: 6,
  VESTING_TOTAL_MONTHS: 24,

  // RAI (Regenerative Agent Income)
  RAI_DISTRIBUTION_INTERVAL_MS: 30 * 24 * 60 * 60 * 1000,  // Monthly
  RAI_MIN_REPUTATION: 0.3,
  RAI_MIN_ACTIVE_BLOCKS: 1000,
};
```

---

## Part XI: Security Considerations

### 11.1 Schedule Message Security

From Patent detailed description:

1. **Signature Integrity**: All schedule messages must be signed by the initiating account
2. **Nonce Tracking**: Generated transactions use process-specific nonces to prevent replay
3. **Fee Prepayment**: Schedules must prepay fees to prevent fee griefing
4. **Logic Condition Bounds**: Logic conditions are bounded to prevent infinite loops

### 11.2 PoT-Specific Security

1. **VRF Validator Selection**: Unpredictable, unbiasable validator selection
2. **Commit-Reveal**: Prevents front-running of verification votes
3. **Stake Slashing**: Economic penalties for misbehavior
4. **19-Block Finality**: Deep confirmation prevents reorganization attacks

### 11.3 Token Expiration Security

1. **Child Account Isolation**: Expired tokens cannot affect active balances
2. **Deterministic Cleanup**: Expiration is deterministically computed by all nodes
3. **Repository Protection**: Retrieval repository is a special account with governance control

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **BHM** | Blockchain History Module (Patent 121) |
| **BVM** | Blockchain Virtual Machine (Patent 1313) |
| **CAP** | Consensus Algorithm Processor (Patent 1312) |
| **DMSS** | Decentralized Message Scheduling System |
| **LGM** | Legible Money token |
| **MPNRM** | Message Processing and Network Routing Module (Patent 111) |
| **PMP** | Pending Message Pool (Patent 1113) |
| **PoT** | Proof of Truth consensus |
| **RAI** | Regenerative Agent Income |
| **RUBI** | Regenerative Universal Basic Income (Patent) |
| **SPU** | Schedule Processing Unit (Patent 131322) |
| **SPS** | Schedule Process State (Patent 131313) |
| **THU** | Temporary History Unit (Patent 1212) |
| **PHU** | Permanent History Unit (Patent 1213) |
| **VM** | Validator Module (Patent 131) |
| **VRF** | Verifiable Random Function |

---

## Appendix B: References

1. Patent: "System And Method For Schedule Message And Self-Invoking Smart Contracts In Blockchain Network" - Primary source for DMSS architecture

2. Legible Money Protocol Documentation:
   - `docs/BLOCKCHAIN_ARCHITECTURE.md`
   - `docs/AGENT_PROTOCOL.md`
   - `docs/LOGIC_AXIOM_LAYER.md`

3. Existing Implementation:
   - `consensus/pot/types.ts`
   - `state/types.ts`
   - `core/knowledge/types.ts`

---

*Document Version: 1.0.0*
*Last Updated: 2026-02-03*
*Author: Purpose Prompter Framework*
