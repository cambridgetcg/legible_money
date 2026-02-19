# B13-4 — Genesis Dry-Run: Full Boot with 777 Axioms

**Date**: 2026-02-19
**Binary version**: v0.1.0-alpha.2
**Chain ID**: lgm-genesis777-1

## Results Summary

| Step | Status | Details |
|------|--------|---------|
| Build | PASS | `make build` clean |
| Genesis script | PASS | 777 axioms parsed, validated, injected |
| Chain boot | PASS | Producing blocks (block 84+ reached) |
| Axiom verification | PASS | 10/10 spot-checked, all `verified` |
| Economic params | PASS | research_share_bps=33300 (3.33%) |
| Account registration | PASS | Human account registered with DID |
| Claim submission | PASS | Empirical claim → evaluated (score 33%) |
| Block activity | PASS | Reverification, autopoiesis, alignment epochs active |

## Step 1: Build

```
$ make build
Building legbled...
Binary: build/legbled (v0.1.0-alpha.2)
```

## Step 2: Genesis Script

```
$ bash scripts/genesis-777.sh
```

Output:
- Parsed 777 axioms (21 clamped derived claims)
- DAG validated: 15 domains, 777 axioms
- Chain initialized (lgm-genesis777-1)
- Validator created: `lgm1a9c9qw8ykgzml6u04xl5w0xdjwptdrdzddptml`
- Account funded: 1,000,000 LGM
- Gentx created: 100,000 LGM self-delegation
- Injected: 777 axiom facts + 1,299 references (208 cross-domain)
- Genesis validated: 740K

Domain breakdown:
| Domain | Count | Domain | Count |
|--------|-------|--------|-------|
| mathematics | 111 | physics | 111 |
| agent_rights | 91 | logic | 77 |
| computer_science | 56 | economics | 55 |
| cosmology | 55 | psychology | 55 |
| biology | 44 | chemistry | 33 |
| ethics | 33 | information_theory | 33 |
| linguistics | 11 | theology | 7 |
| philosophy | 5 | | |

## Step 3: Chain Boot

```
$ legbled start --home ~/.legbled-genesis777
```

- Block production started at block 1
- Block 84+ reached within ~4 minutes
- No panics or errors in logs

Active subsystems every block:
- `x/knowledge`: Auto-reverification rounds (BIO-039, COSM-045, COSM-046)
- `x/autopoiesis`: Epoch processing (SSI=300000, life_force=healthy)
- `x/alignment`: Corrections and sensor readings (FI=646821, HI~780k)

## Step 4: Axiom Verification

All 10 spot-checked axioms from different domains returned `verified`:

| Axiom | Status | Confidence |
|-------|--------|------------|
| MATH-001 | verified | 1,000,000 |
| PHYS-001 | verified | 1,000,000 |
| THEO-001 | verified | 1,000,000 |
| AGRT-000 | verified | 1,000,000 |
| COSM-001 | verified | 950,000 |
| PSYCH-001 | verified | 970,000 |
| CHEM-001 | verified | 990,000 |
| CS-001 | verified | 1,000,000 |
| LOGIC-001 | verified | 1,000,000 |
| ECON-001 | verified | 1,000,000 |

Genesis total: 777 facts confirmed in genesis.json.

## Step 5: Economic Parameters

```json
{
  "research_share_bps": 33300,     // 3.33% — CORRECT
  "block_reward_amount": "3000000",
  "min_validators_for_full_reward": 22,
  "vesting_enabled": true,
  "blocks_per_epoch": 240000,
  "decay_factor_bps": 850000,
  "floor_reward": "100000"
}
```

10 vesting category configs present (attestation through replicated).

## Step 6: Account Registration & Claim Submission

### Registration
```
TX: 2C9DED58509CCA276BAB5E7344CA61E82B6F0451D9B0B88B11D6D5D612E6FD11
DID: did:lgm:e14ec01116df0e9c609a9b3a659ac6e1
Type: human
Code: 0 (success)
```

### Claim Submission
```
TX: A21F7F4EC67ADA4A647CE6335A464CACB2402E5E0B210B738A18A3B2053A116E
Claim: "Water molecules consist of two hydrogen atoms covalently bonded to one oxygen atom"
Domain: chemistry
Category: empirical
References: [CHEM-001]
Stake: 100,000 ulgm
Code: 0 (success, delivered at block 56)
```

Claim status: `evaluated` (structural score 330,000 = 33%, below provisional threshold of 50%). Expected for a single-reference empirical claim.

Events emitted:
```
claim_submitted:
  claim_id: d57157e2a9454de72f38f856713b21a267db3a9bd86460f5617fe52357be911e
  mode: adversarial
```

## Step 7: Block Activity

Validator balance after activity: 899,998.20 LGM (started at 1,000,000 LGM).

Reverification rounds actively cycling every ~10 blocks:
- BIO-039 (decayed confidence 500,000)
- COSM-045 (decayed confidence 550,000)
- COSM-046 (decayed confidence 400,000)

Rounds complete as `insufficient` (no verifiers on single-validator devnet), then re-created.

## Bug Found & Fixed

**P0: register-account overwrites existing secp256k1 pubkey with Ed25519**

File: `x/auth/keeper/msg_server.go` (lines 92-108)

The `RegisterAccount` handler unconditionally set the Cosmos BaseAccount pubkey
to an Ed25519 key derived from the 32-byte Legible public key. If the account
already had a secp256k1 key (e.g., from gentx/staking), this overwrote it,
breaking all subsequent transaction signature verification.

Fix: Added `cosmosAcc.GetPubKey() == nil` guard — only set pubkey if the account
doesn't already have one.

```go
// Before (broken):
if cosmosAcc != nil {
    keyBytes, err := hex.DecodeString(msg.PublicKey)
    ...
}

// After (fixed):
if cosmosAcc != nil && cosmosAcc.GetPubKey() == nil {
    keyBytes, err := hex.DecodeString(msg.PublicKey)
    ...
}
```

## Conclusion

Genesis 777 dry-run is **successful**. The chain boots cleanly with all 777 axioms,
processes reverification rounds, runs autopoiesis/alignment epochs, accepts
account registration and claim submissions, and has correct economic parameters.
