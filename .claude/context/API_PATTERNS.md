# API Patterns Quick Reference

> Common patterns used throughout the codebase.

---

## State Manager

```typescript
// Get state
const account = state.getAccount(did);
const validator = state.getValidator(publicKey);
const fact = state.getFact(factId);
const balance = state.getBalance(did);

// Modify state (always in transaction)
state.beginTransaction();
try {
  state.transfer(from, to, amount);
  state.stake(did, amount);
  state.addFact(factState);
  state.commitTransaction();
} catch (e) {
  state.rollbackTransaction();
}

// Apply block
const newRoot = state.applyBlock(blockNumber, blockHash);
```

---

## Account State Tree

```typescript
const tree = new AccountStateTree();

// Create account
tree.createAccount(did, operationalKeyHash, initialBalance);

// Read
const account = tree.getAccount(did);
const balance = tree.getBalance(did);
const nonce = tree.getNonce(did);

// Modify
tree.transfer(from, to, amount);
tree.stake(did, amount);
tree.unstake(did, amount);
tree.applyDelta({ did, balanceChange: 100n, nonceIncrement: true });

// Prove
const proof = tree.proveAccount(did);
const balanceProof = tree.proveBalance(did, blockNumber);
AccountStateTree.verifyBalanceProof(proof); // static method
```

---

## Validator State Tree

```typescript
const tree = new ValidatorStateTree();

// Register
tree.registerValidator(publicKey, did, stake, blockNumber);

// Query
const validator = tree.getValidator(publicKey);
const isActive = tree.isActiveValidator(publicKey);
const activeValidators = tree.getActiveValidators();
const totalStake = tree.getTotalStake();

// Modify
tree.addStake(publicKey, amount);
tree.removeStake(publicKey, amount);
tree.slashValidator(publicKey, amount, reason);
tree.recordBlockProposal(publicKey, blockNumber);
tree.recordVerification(publicKey, correct);

// Delegation
tree.delegate(delegator, validator, amount, blockNumber);
tree.undelegate(delegator, validator, amount, currentBlock, unbondingPeriod);
```

---

## Knowledge State Tree

```typescript
const tree = new KnowledgeStateTree();

// Add fact
tree.addFact({
  id: factId,
  claim: "The sky is blue",
  domain: "observations.weather",
  confidence: 0.95,
  verifiedAt: blockNumber,
  verifierCount: 7,
  challengeCount: 0,
  status: 'active',
});

// Query
const fact = tree.getFact(factId);
const isActive = tree.isActiveFact(factId);
const domainFacts = tree.getFactsByDomain("observations.weather");
const results = tree.searchFacts("sky");

// Challenge/revoke
tree.challengeFact(factId, blockNumber);
tree.resolveChallenge(factId, upheld, newConfidence);
tree.revokeFact(factId);

// References (knowledge graph)
tree.addReference({
  fromFactId: "fact1",
  toFactId: "fact2",
  referenceType: 'supports',
  strength: 0.8,
});
const refs = tree.getOutgoingReferences(factId);
const supporting = tree.getSupportingFacts(factId);
const contradicting = tree.getContradictingFacts(factId);

// Prove
const proof = tree.proveFact(factId, blockNumber);
KnowledgeStateTree.verifyFactProof(proof);
```

---

## Sparse Merkle Tree

```typescript
const smt = new InMemorySMT();

// Basic operations
smt.set(key, value);    // Returns new root
smt.get(key);           // Returns value or null
smt.delete(key);        // Returns new root

// Proofs
const proof = smt.prove(key);
InMemorySMT.verify(proof);  // Returns boolean

// Batch
smt.batchSet([{ key, value }, ...]);
```

---

## Transaction Execution

```typescript
import { executeTransaction, executeBlock } from './state/transitions';

// Single transaction
const result = executeTransaction(state, tx, blockNumber, params);
// result: { success, stateRoot?, error?, gasUsed }

// Full block
const blockResult = executeBlock(state, transactions, verificationResults, blockNumber, params);
// blockResult: { stateRoot, results[], totalGasUsed, successCount, failureCount }

// Apply verification outcome
applyVerificationResult(state, verificationResult, blockNumber, params);

// Epoch transition
applyEpochTransition(state, epochPayload, params);
```

---

## Cryptographic Functions

```typescript
// Hashing
sha512_256(data);           // Uint8Array -> Uint8Array
sha512_256Hex(data);        // string -> string
blake3Hash(data);           // Uint8Array -> Uint8Array
domainHash(domain, data);   // Domain-separated hash

// Signatures
const { publicKey, privateKey } = generateKeyPair();
const sig = sign(message, privateKey);
const valid = verify(message, sig, publicKey);

// VRF
const { output, proof } = vrfGenerate(privateKey, input);
const valid = vrfVerify(publicKey, input, output, proof);

// Commitments
const { commitment, opening } = createCommitment(message);
const valid = verifyCommitment(commitment, message, opening);
```

---

## Genesis Setup

```typescript
import { createStateFromGenesis, DEV_GENESIS } from './state/state_manager';

const state = createStateFromGenesis({
  accounts: [
    { did: 'did:lgm:alice', operationalKeyHash: '...', balance: 1000n * 10n**18n }
  ],
  validators: [
    { publicKey: '...', did: 'did:lgm:validator1', stake: 10000n * 10n**18n }
  ],
  axioms: [
    { id: 'axiom:logic:identity', claim: 'A = A', domain: 'logic.foundational' }
  ],
});
```
