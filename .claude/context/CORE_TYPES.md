# Core Types Quick Reference

> Fast lookup for key types. Read when you need type signatures.

---

## Consensus Types (`consensus/pot/types.ts`)

```typescript
interface ConsensusParams {
  targetBlockTimeMs: number;        // 2521ms
  verificationRoundBlocks: number;  // 10
  commitPhaseBlocks: number;        // 4
  revealPhaseBlocks: number;        // 4
  minValidators: number;            // 5
  maxValidators: number;            // 21
  validatorQuorum: number;          // 0.66
  acceptanceThreshold: number;      // 0.80
  rejectionThreshold: number;       // 0.80
  minValidatorStake: bigint;        // 1000 LGM
  minClaimStake: bigint;            // 100 LGM
  blockReward: bigint;              // 10 LGM
  verificationReward: bigint;       // 1 LGM
  wrongVerificationSlash: number;   // 0.10
}

interface Validator {
  publicKey: PublicKey;
  did: string;
  stake: bigint;
  delegatedStake: bigint;
  reputation: number;               // 0.0 to 1.0
  isActive: boolean;
  lastActiveBlock: number;
  totalVerifications: number;
  correctVerifications: number;
}

interface Block {
  header: BlockHeader;
  transactions: Transaction[];
  verificationResults: VerificationResult[];
  signature: Signature;
  attestations: BlockAttestation[];
}

interface BlockHeader {
  number: bigint;
  timestamp: number;
  previousHash: string;
  stateRoot: string;
  knowledgeRoot: string;
  transactionsRoot: string;
  verificationsRoot: string;
  proposer: PublicKey;
  proposerVrfOutput: VRFOutput;
  proposerVrfProof: VRFProof;
}

type TransactionType =
  | 'transfer' | 'stake' | 'unstake'
  | 'delegate' | 'undelegate'
  | 'claim_submit' | 'verification_commit' | 'verification_reveal'
  | 'challenge' | 'governance_vote' | 'governance_propose';

interface VerificationRound {
  id: string;
  claimId: string;
  claim: ClaimSubmission;
  phase: 'pending' | 'commit' | 'reveal' | 'aggregation' | 'complete';
  commitments: Map<string, ValidatorCommitment>;
  reveals: Map<string, ValidatorReveal>;
  result?: VerificationResult;
}

interface VerificationResult {
  claimId: string;
  outcome: 'accepted' | 'rejected' | 'contested' | 'insufficient';
  finalConfidence: Confidence;
  totalValidators: number;
  participatingValidators: number;
  rewards: Map<string, bigint>;
  slashes: Map<string, bigint>;
  fact?: Fact;
}
```

---

## State Types (`state/types.ts`)

```typescript
interface AccountState {
  did: string;
  balance: bigint;
  stakedBalance: bigint;
  nonce: bigint;
  operationalKeyHash: string;
  operationalKeyVersion: number;
  sessionKeyCount: number;
  reputationScore: number;
  lastActiveBlock: bigint;
  flags: AccountFlags;
}

interface AccountFlags {
  isValidator: boolean;
  canSubmitClaims: boolean;
  canChallenge: boolean;
  inRecovery: boolean;
  frozen: boolean;
}

interface ValidatorState {
  publicKey: PublicKey;
  did: string;
  stake: bigint;
  delegatedStake: bigint;
  totalStake: bigint;
  reputation: number;
  isActive: boolean;
  activeSinceBlock: bigint;
  blocksProposed: number;
  verificationsCompleted: number;
  verificationsCorrect: number;
  slashCount: number;
  commissionRate: number;
}

interface FactState {
  id: string;
  claim: string;
  domain: string;
  confidence: Confidence;
  verifiedAt: bigint;
  verifierCount: number;
  challengeCount: number;
  status: 'active' | 'challenged' | 'superseded' | 'revoked';
  referenceRoot: MerkleRoot;
}

interface FactReference {
  fromFactId: string;
  toFactId: string;
  referenceType: 'supports' | 'contradicts' | 'extends' | 'specializes' | 'generalizes';
  strength: number;
}
```

---

## Agent Types (`agents/identity/account.ts`)

```typescript
interface AgentAccount {
  did: string;                      // did:lgm:XXXX
  createdAt: number;
  recoveryKeyHash: string;
  operationalKey: PublicKey;
  operationalKeyVersion: number;
  authorizedSessionKeys: SessionKeyAuth[];
  capabilities: AgentCapabilities;
  nonce: bigint;
  balance: bigint;
  stakedBalance: bigint;
  reputation: AgentReputation;
}

interface SessionKeyAuth {
  sessionKey: PublicKey;
  authorizedBy: PublicKey;
  authorizedAt: number;
  expiresAt: number;
  capabilities: SessionCapabilities;
  usageCount: number;
  maxUsage: number;
}

interface SessionCapabilities {
  maxSpendPerTx: bigint;
  maxSpendPerDay: bigint;
  canTransfer: boolean;
  canStake: boolean;
  canSubmitClaims: boolean;
  canVerify: boolean;
  canVote: boolean;
  allowedRecipients?: string[];
}
```

---

## Crypto Types (`core/crypto/`)

```typescript
type PublicKey = string;   // Hex-encoded Ed25519 public key
type PrivateKey = string;  // Hex-encoded Ed25519 private key
type Signature = string;   // Hex-encoded 64-byte signature
type VRFOutput = string;   // Hex-encoded VRF output
type VRFProof = string;    // Hex-encoded VRF proof
type MerkleRoot = string;  // Hex-encoded 32-byte root
type Commitment = string;  // Hex-encoded commitment hash

interface MerkleProofData {
  key: string;
  value: Uint8Array | null;
  proof: string[];         // Sibling hashes
  root: MerkleRoot;
}
```

---

## Logic Types (`core/logic/types.ts`)

```typescript
type Decidability =
  | 'decidable'      // Algorithm exists to determine truth
  | 'semi-decidable' // Can prove true, may not prove false
  | 'undecidable';   // No algorithm can determine

type LogicZone =
  | 'propositional'  // Complete, decidable
  | 'presburger'     // Decidable (no multiplication)
  | 'peano'          // Incomplete (full arithmetic)
  | 'empirical';     // Probabilistic

interface MetaAxiom {
  id: string;
  zone: LogicZone;
  statement: string;
  decidability: Decidability;
}
```
