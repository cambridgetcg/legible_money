# R5 Session 3: Genesis Ceremony — Testnet Genesis Validation

## Context
The current `genesis.json` at repo root is a template. For testnet, we need a validated genesis that includes all 28 modules' genesis state. The devnet script creates one, but it may not exercise all modules.

## Tasks

### 1. Audit Current Genesis Script
Read `devnet/scripts/cosmos-devnet.sh` and `devnet/scripts/genesis-ceremony.sh`:
- Which modules get explicit genesis configuration?
- Which modules rely purely on DefaultGenesis?
- Are the R3/R4 changes (ExtendedParams, TierConfigs in Params, tokens Phase 2) reflected?

### 2. Validate Genesis Round-Trip
```bash
# Init fresh chain
build/legbled init test-node --chain-id lgm-test-1 --home /tmp/lgm-genesis-test

# Export default genesis
build/legbled export --home /tmp/lgm-genesis-test 2>/dev/null > /tmp/genesis-export-1.json

# Import and re-export
build/legbled init test-node-2 --chain-id lgm-test-1 --home /tmp/lgm-genesis-test-2
cp /tmp/genesis-export-1.json /tmp/lgm-genesis-test-2/config/genesis.json
build/legbled validate-genesis --home /tmp/lgm-genesis-test-2

# Diff — should be empty or only cosmetic
diff <(jq -S . /tmp/genesis-export-1.json) <(jq -S . /tmp/genesis-export-2.json)
```

### 3. Check Each Module's DefaultGenesis
For each of the 28 modules, verify `DefaultGenesis()` produces valid state:
- Params populated with sensible defaults
- No nil pointer dereferences during InitGenesis
- Module-specific invariants hold

Focus on modules changed in R3/R4:
- **knowledge**: ExtendedParams must be populated (not empty)
- **staking**: TierConfigs must be in Params (4 tiers)
- **tokens**: Delegation and wrap genesis entries (empty but structurally valid)
- **schedule**: CleanupIntervalBlocks in Params
- **gov**: QuorumThresholdBps set

### 4. Validate Genesis Accounts and Balances
The devnet script creates:
- Validator: 50,000 LGM
- Foundation: ~22.2M LGM
- Research: ~17.1M LGM
- Community: ~11.1M LGM
- Test accounts: 1,000 LGM each

Verify these match the tokenomics in `docs/TRUSTPAPER.md`. If they diverge, document.

### 5. Test Genesis with Vote Extensions
Vote extensions (ABCI++) require an enable height in genesis. Verify:
```bash
jq '.consensus.params.abci.vote_extensions_enable_height' genesis.json
```
Should be set to a low block height for devnet (e.g., 1 or 2).

### 6. Create Testnet Genesis Template
If the current genesis ceremony needs updates for R3/R4 changes:
- Update `devnet/scripts/genesis-ceremony.sh` or create `devnet/scripts/testnet-genesis.sh`
- Include explicit genesis for: knowledge ExtendedParams, staking TierConfigs, schedule CleanupInterval
- Include faucet account for testnet

## Verification
```bash
build/legbled validate-genesis --home $HOME/.legbled-devnet
```

## Report
Write to `/reports/batch-5/SESSION-R5-3-GENESIS-CEREMONY.md` with:
- Module-by-module genesis validation (28 modules)
- Round-trip test results
- Any DefaultGenesis issues found
- Genesis account balance summary
- Vote extensions configuration status
