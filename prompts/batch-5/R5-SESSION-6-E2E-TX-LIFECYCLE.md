# R5 Session 6: Extended E2E TX Lifecycle Tests

## Context
The existing `test-tx-lifecycle.sh` has 23 tests covering bank, SDK staking, governance, auth, and economics. But it doesn't test ANY of the custom LGM modules. For testnet confidence, we need live-chain tests for the core protocol modules.

## Tasks

### 1. Start Devnet
```bash
cd /Users/yournameisai/Desktop/legible-money
make devnet-reset && make devnet
build/legbled start --home $HOME/.legbled-devnet &
sleep 15
```

### 2. Create Extended TX Test Script
Create `devnet/scripts/test-lgm-modules.sh` with tests for:

**Phase A: Knowledge Module (Core PoT)**
```bash
# A1: Submit a claim
legbled tx knowledge submit-claim "Earth orbits the Sun" empirical --stake 1000ulgm --from test1

# A2: Query the claim
legbled query knowledge claim <claim_id>

# A3: Submit verification evidence
legbled tx knowledge submit-evidence <claim_id> on_chain "Verified via orbital data" --from test2

# A4: Query verification round
legbled query knowledge round <round_id>

# A5: Query params (verify ExtendedParams accessible)
legbled query knowledge params
```

**Phase B: LGM Staking Module**
```bash
# B1: Register as validator
legbled tx legible-staking register-validator <pubkey> 10000ulgm --from test1

# B2: Query validator
legbled query legible-staking validator <addr>

# B3: Query tier configs (verify parameterized)
legbled query legible-staking params
```

**Phase C: Tokens Module (Phase 2)**
```bash
# C1: Create secondary token
legbled tx tokens create-token "TestToken" "TST" 6 --features wrappable --from test1

# C2: Mint tokens
legbled tx tokens mint-token <token_id> 1000000 <addr> --from test1

# C3: Delegate power
legbled tx tokens delegate-power <token_id> <delegate_addr> 500000 --from test1

# C4: Wrap token
legbled tx tokens wrap-token <token_id> 200000 --from test1

# C5: Unwrap token
legbled tx tokens unwrap-token lgm20/<token_id> 100000 --from test1

# C6: Verify balances
legbled query tokens balance <token_id> <addr>
legbled query bank balances <addr>
```

**Phase D: Home Module**
```bash
# D1: Create agent home
legbled tx home create-home "test-agent" --from test1

# D2: Query home
legbled query home home <home_id>
```

**Phase E: Governance (LGM Gov)**
```bash
# E1: Submit LIP proposal
legbled tx lgm-gov submit-lip "Test Improvement" "Description" --from test1

# E2: Vote on LIP
legbled tx lgm-gov vote-lip <lip_id> support 1000 --from test1

# E3: Query LIP
legbled query lgm-gov lip <lip_id>
```

### 3. Test Script Structure
Follow the same pattern as `test-tx-lifecycle.sh`:
- Colored pass/fail output
- Exit code reflects test results
- Each test is independent (own function)
- Wait for tx inclusion before querying

### 4. Run All Tests
```bash
# Existing tests
bash devnet/scripts/test-tx-lifecycle.sh

# New module tests
bash devnet/scripts/test-lgm-modules.sh
```

### 5. Clean Up
```bash
kill $LEGBLED_PID
make devnet-stop
```

## Verification
- All existing 23 tx lifecycle tests pass
- New module tests pass (knowledge, staking, tokens, home, gov)
- No panics in node logs during tests

## Report
Write to `/reports/batch-5/SESSION-R5-6-E2E-TX-LIFECYCLE.md` with:
- Pass/fail for all tests (existing + new)
- Any CLI commands that error (missing subcommand, wrong args, etc.)
- Any handlers that panic on live chain
- Node log excerpts for any errors
