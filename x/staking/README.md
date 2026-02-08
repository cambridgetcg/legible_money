# x/staking - Graduated Validator Tiers Module

Custom Cosmos SDK staking module for Legible Money with **Proof of Truth** consensus and graduated validator tiers.

## Overview

This module extends standard Cosmos SDK staking with:
- **4-tier validator system** (Apprentice → Verified → Bonded → Guardian)
- **Zero-stake onboarding** via `register_validator` transaction
- **Performance-based progression** through verification accuracy tracking
- **Virtual stake** for tier 0/1 validators (VRF selection fairness)
- **Epistemic category restrictions** by tier
- **Tier-adjusted rewards and slashes**

## Validator Tiers

| Tier | Name | Requirements | Allowed Categories | Reward | Selection Weight |
|------|------|--------------|-------------------|---------|------------------|
| 0 | Apprentice | None (zero stake OK) | protocol, analytic, formal | 0.1x | 0.1x (virtual) |
| 1 | Verified | 22 verifications @ 77% accuracy | +empirical, historical | 0.5x | 0.5x (virtual) |
| 2 | Bonded | 1,111 LGM staked | all 7 categories | 1.0x | 1.0x (real stake) |
| 3 | Guardian | 11,111 LGM + 1,111 verif @ 77% + 0 slashes | all 7 categories + governance | 1.5x | 1.5x (real stake) |

### Tier Progression

- **Automatic tier upgrades**: When validator meets next tier requirements
- **Automatic tier downgrades**: On slashing, stake reduction, or accuracy drop
- **Apprentice cap**: Max 111 tier 0 validators (Sybil defense)
- **Virtual stake**: Tier 0/1 validators use 111 LGM virtual stake for VRF selection probability

## Module Architecture

### Types (`types/`)
- `types.go`: Core types (LegibleValidator, Delegation, UnbondingEntry, Params, Messages)
- `keys.go`: Store key prefixes
- `errors.go`: Error definitions
- `expected_keepers.go`: Keeper interfaces (AccountKeeper, BankKeeper, LegibleAuthKeeper)
- `codec.go`: Codec registration

### Keeper (`keeper/`)
- `keeper.go`: Core keeper logic (validator/delegation/unbonding CRUD)
- `msg_server.go`: Message handlers (RegisterValidator, UpdateValidatorStake)
- `tiers.go`: Pure tier computation functions (ComputeValidatorTier, CalculateTierReward, etc.)

### Module (`module.go`)
- AppModule implementation with BeginBlock/EndBlock
- Genesis init/export
- Service registration

## State Layout

### Store Keys

```
0x01 | validatorAddr -> LegibleValidator
0x02 | delegatorAddr | 0x00 | validatorAddr -> Delegation
0x03 | unbondingId -> UnbondingEntry
0x04 | tier -> TierConfig
0x05 -> Params
0x06 | did -> validatorAddr (DID index)
```

### Indexes

- **Primary**: operator address → validator
- **DID index**: did:lgm:xxx → operator address

## Messages

### MsgRegisterValidator

Register a new validator with optional initial stake:

```go
type MsgRegisterValidator struct {
    Sender          string // operator address
    ConsensusPubkey string // Ed25519 hex
    Did             string // did:lgm:xxx
    InitialStake    string // can be "0" for tier 0
    Commission      uint64 // basis points (1000 = 10%)
}
```

**Validation**:
- Checks apprentice cap if zero stake
- Validates DID uniqueness
- Enforces max validators limit

### MsgUpdateValidatorStake

Increase or decrease self-delegation:

```go
type MsgUpdateValidatorStake struct {
    Validator  string // operator address
    Amount     string // bigint as string
    IsIncrease bool   // true = stake, false = unstake
}
```

**Unstaking**:
- Creates unbonding entry with 7-day period (268560 blocks @ 2.521s)
- Tokens locked until `CompletesAtHeight`
- Processed in `BeginBlocker`

## Tier Computation

From `keeper/tiers.go` (translated from TypeScript `consensus/pot/validator_tiers.ts`):

```go
func ComputeValidatorTier(
    stake string,
    verifications uint64,
    correctVerifications uint64,
    slashCount uint64,
) ValidatorTier {
    accuracy := (correctVerifications * 1000000) / verifications

    // Check Guardian requirements
    if stake >= 11111 LGM &&
       verifications >= 1111 &&
       accuracy >= 770000 &&
       slashCount == 0 {
        return TierGuardian
    }

    // Check Bonded
    if stake >= 1111 LGM {
        return TierBonded
    }

    // Check Verified
    if verifications >= 22 && accuracy >= 770000 {
        return TierVerified
    }

    // Default Apprentice
    return TierApprentice
}
```

## Integration Points

### With x/auth
- Uses `LegibleAuthKeeper.GetAccountByDID()` for DID → account lookup
- Creates accounts for new validators via `AccountKeeper`

### With x/bank
- Stake transfers to/from module account
- Unbonding returns via `SendCoinsFromModuleToAccount`

### With Consensus (future)
- `GetActiveValidatorSet()` for block proposer selection
- `RecordVerification()` called after each verification round
- `SlashValidator()` on incorrect verifications or malicious behavior

## Block Processing

### BeginBlocker
- Processes mature unbonding entries
- Returns tokens to delegators
- Emits `unbonding_completed` events

### EndBlocker
- Checks for tier transitions (safety net)
- Most transitions happen immediately on `RecordVerification()`

## Parameters

```go
type Params struct {
    MaxApprenticeValidators uint32 // 111 (Sybil defense)
    UnbondingPeriod         uint64 // 268560 blocks (~7 days)
    VirtualStake            string // "111000000000000000000" (111 LGM)
    MaxValidators           uint32 // 100
    MinSelfDelegation       string // "0" (allows tier 0)
}
```

## Events

- `validator_registered`: New validator registered
- `validator_stake_updated`: Stake increased/decreased
- `validator_tier_changed`: Tier transition
- `unbonding_completed`: Unbonding period finished

## Migration Path

### Current State (Phase 1)
- Hand-written Go types (this implementation)
- No proto definitions yet
- Basic functionality for devnet testing

### Phase 2 (Before Testnet)
- Generate proto definitions from types
- Add gRPC query service
- Add CLI commands (tx/query)
- Add REST endpoints

### Phase 3 (Before Mainnet)
- Full integration with Cosmos SDK staking hooks
- Delegation from non-validators
- Redelegation support
- Governance integration for Guardian tier

## Usage Example

```go
// Register zero-stake validator (tier 0)
msg := &types.MsgRegisterValidator{
    Sender:          "legible1abc...",
    ConsensusPubkey: "ed25519_hex_pubkey",
    Did:             "did:lgm:1234567890abcdef...",
    InitialStake:    "0",
    Commission:      1000, // 10%
}

// After 22 verifications at 77% accuracy, auto-upgrade to tier 1

// Add stake to reach tier 2
msgStake := &types.MsgUpdateValidatorStake{
    Validator:  "legible1abc...",
    Amount:     "1111000000000000000000", // 1111 LGM
    IsIncrease: true,
}

// After 1111 verifications at 77% with zero slashes, auto-upgrade to tier 3
```

## Testing

### Unit Tests (TODO)
- `types_test.go`: Type validation, serialization
- `keeper_test.go`: Validator registration, delegation, unbonding
- `tiers_test.go`: Tier computation logic
- `msg_server_test.go`: Message handlers

### Integration Tests (TODO)
- Multi-validator scenarios
- Tier transitions under various conditions
- Slash and recovery flows
- Unbonding period edge cases

## Security Considerations

1. **Sybil Defense**: Apprentice cap (111) prevents unlimited zero-stake validators
2. **Tier Demotion**: Slashing automatically demotes validators (especially Guardian → any slash)
3. **Unbonding Period**: 7-day lockup prevents rapid stake withdrawal during attacks
4. **Category Restrictions**: Lower tiers can't verify empirical/historical claims
5. **Virtual Stake Fairness**: Tier 0/1 use fixed 111 LGM for VRF selection (not zero)

## Future Enhancements

- [ ] Validator commission changes (with delay)
- [ ] Delegation from non-validator accounts
- [ ] Redelegation (move stake between validators)
- [ ] Validator jailing (temporary removal)
- [ ] Tombstoning (permanent ban)
- [ ] Governance proposals for tier config changes
- [ ] Validator reputation decay over time
- [ ] Multi-signature validator keys
