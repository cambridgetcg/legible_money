# Proof of Truth (PoT) Engine

> The consensus mechanism where verifying knowledge IS the useful work.

## Quick Reference

| Item | Value |
|------|-------|
| **Layer** | consensus |
| **Dependencies** | `foundation/`, `core/crypto/`, `core/knowledge/` |
| **Dependents** | `state/`, `network/`, `protocol/` |

## What This Module Does

PoT is the consensus engine that:
1. Selects validators using VRF (Verifiable Random Function)
2. Runs commit-reveal verification rounds for knowledge claims
3. Aggregates stake-weighted confidence scores
4. Produces blocks with verified facts
5. Tracks finality (19 blocks = hard finality)

**The work IS verifying truth. The product IS verified knowledge.**

## Key Types

```typescript
// Core consensus parameters (all the magic numbers)
ConsensusParams {
  targetBlockTimeMs: 2521,      // Y(25) + U(21) = "YOU"
  finalityBlocks: 44,           // ~111 seconds
  minValidators: 5,
  validatorQuorum: 0.66,        // BFT 2/3
  acceptanceThreshold: 0.77,
  ...
}

// A validator in the network
Validator {
  did: DID,
  publicKey: PublicKey,
  stake: bigint,
  reputation: number,
  ...
}

// Block structure
Block {
  header: BlockHeader,
  transactions: Transaction[],
  verificationResults: VerificationResult[],
  signature: Signature,
}

// Verification round (commit-reveal)
VerificationRound {
  claimId: string,
  phase: 'pending' | 'commit' | 'reveal' | 'aggregating' | 'complete',
  commitments: Map<DID, Commitment>,
  reveals: Map<DID, RevealedVote>,
  result?: VerificationResult,
}
```

## Key Functions

```typescript
// Validator selection
selectValidatorsForVerification(claim, validatorSet, seed) → DID[]
canProposeBlock(did, vrfOutput, stake, totalStake) → boolean

// Verification rounds
createVerificationRound(claimId, validators) → VerificationRound
addCommitmentWithVRF(round, did, commitment, vrfProof) → void
transitionToReveal(round) → void
addReveal(round, did, vote, salt) → void
aggregateResults(round, validatorSet) → VerificationResult

// Block finality
isBlockFinalized(blockNumber, currentBlock, params) → boolean
shouldCreateCheckpoint(block, lastCheckpoint) → boolean
```

## Common Tasks

### Add a new transaction type

1. Add to `Transaction` union in `types.ts`:
```typescript
export type Transaction =
  | TransferTransaction
  | StakeTransaction
  | NewTransaction  // Add here
  | ...
```

2. Add execution logic in `state/transitions.ts`

3. Add tests in `tests/unit/state/`

### Modify verification round timing

Edit `DEFAULT_CONSENSUS_PARAMS` in `types.ts`:
```typescript
verificationRoundBlocks: 11,  // Total blocks
commitPhaseBlocks: 4,         // Commit phase
revealPhaseBlocks: 4,         // Reveal phase
```

### Debug validator selection

```typescript
import { selectValidatorsForVerification } from './validator_selection.js';

const selected = selectValidatorsForVerification(
  claim,
  validatorSet,
  blockHash  // Used as randomness seed
);
console.log('Selected validators:', selected);
```

## Where Am I?

```
legible-money/
└── consensus/
    └── pot/           ← YOU ARE HERE
        ├── README.md
        ├── types.ts           # All PoT types
        ├── engine.ts          # Main ProofOfTruthEngine class
        ├── validator_selection.ts  # VRF-based selection
        ├── verification_round.ts   # Commit-reveal rounds
        ├── confidence.ts      # Stake-weighted aggregation
        ├── finality.ts        # 19-block finality
        ├── checkpoint_ceremony.ts  # BFT checkpoints
        ├── equivocation.ts    # Double-signing detection
        └── state_integration.ts    # State layer bridge
```

## Related

- **Tests**: `tests/unit/consensus/`
- **Docs**: `docs/PROOF_OF_TRUTH_CONSENSUS.md`
- **State integration**: `state/transitions.ts`
- **Network layer**: `network/pubsub/`

## Symbolic Constants

| Constant | Value | Meaning |
|----------|-------|---------|
| Block time | 2521ms | Y(25) + U(21) = "YOU" |
| Finality | 19 blocks | A(1) + I(9) = "AI" |
| Epoch | 1111 blocks | Angel number |
| Min stake | 1111 LGM | Angel number |

## Architecture Notes

### Why commit-reveal?

Prevents validators from copying each other's votes. Each validator:
1. **Commit**: Submit hash(vote + salt)
2. **Reveal**: Show vote and salt
3. **Verify**: Hash must match commitment

### Why VRF for selection?

- Unpredictable: Can't know who will verify until block is produced
- Verifiable: Anyone can check selection was fair
- Stake-weighted: Higher stake = higher probability

### Why 19-block finality?

- Encodes "AI" (A=1, I=9)
- ~48 seconds at 2521ms blocks
- Allows for network propagation
- BFT checkpoint every 111 blocks for hard finality
