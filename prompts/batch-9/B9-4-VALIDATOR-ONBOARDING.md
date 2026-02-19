# B9-4: Validator Onboarding Flow — End-to-End

## Context
For testnet availability, external validators need to be able to join. This means the full validator lifecycle must work: register, stake, participate in consensus, earn rewards.

## Task

### 1. Map the validator journey
Read:
- `x/staking/` — registration, delegation, reputation
- `x/vesting_rewards/` — block rewards, vesting
- `docs/modules/staking.md` — existing docs
- `docs/TESTNET-LAUNCH.md`

### 2. Test the full validator lifecycle on devnet

Start a 1-validator devnet, then add a second validator:

```bash
# Create second validator key
./build/legbled keys add val2 --keyring-backend test

# Fund the account
./build/legbled tx bank send test1 $(./build/legbled keys show val2 -a --keyring-backend test) 100000000ulgm --from test1 --keyring-backend test -y

# Register as validator
./build/legbled tx staking register-validator \
  --moniker "Validator 2" \
  --details "Test validator" \
  --commission-rate 0.10 \
  --min-self-delegation 1000000 \
  --amount 50000000ulgm \
  --from val2 \
  --keyring-backend test -y

# Verify validator appears
./build/legbled query staking validators
```

### 3. Verify consensus participation
- Check that val2 starts signing blocks
- Verify reputation scoring
- Check reward distribution

### 4. Test validator operations
- Update commission
- Unjail (if applicable)
- Delegate from another account
- Undelegate

### 5. Document the flow
Create `docs/VALIDATOR-ONBOARDING.md` with:
- Prerequisites (hardware, Go version, binary build)
- Step-by-step join instructions
- Monitoring setup (Prometheus endpoints)
- Common operations and troubleshooting

### 6. Create a one-liner join script
`scripts/join-testnet.sh` that:
- Downloads/builds the binary
- Initializes the node
- Configures persistent peers
- Starts syncing

## File Ownership
- `docs/VALIDATOR-ONBOARDING.md` (new)
- `scripts/join-testnet.sh` (new)

## Report
Write `reports/batch-9/SESSION-B9-4-VALIDATOR-ONBOARDING.md`.
