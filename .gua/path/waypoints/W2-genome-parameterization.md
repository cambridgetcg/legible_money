# W2: Genome Parameterization

> Give the protocol a mutable genome — constants become parameters, rigidity becomes adaptability.

## Why This Path
89 frozen constants across Go modules are the structural root of J5 (Balance) failure. Constants hardcoded in code belong to no one — not governance, not validators, not the protocol itself. Moving them to `Params` transfers ownership to the governance system, which is where it belongs (J2: Ownership).

This is also prerequisite for W4 (First Living Hormone) — autopoiesis needs parameters to modulate.

## Success Criteria
1. [ ] x/knowledge: 25+ fields moved from constants to Params struct
2. [ ] x/billing: gas costs moved to Params, MsgUpdateParams handler live
3. [ ] x/staking: tier configs moved to Params (fix TierBonded/TierGuardian decimal bug)
4. [ ] x/gov: bounds validation added to param updates
5. [ ] x/emergency: MsgUpdateParams handler implemented
6. [ ] x/vesting_rewards: 22% research tax and reward multipliers in Params
7. [ ] All module Params have ValidateBasic() with bounds checking
8. [ ] Governance can update any Param via standard SDK proposal flow
9. [ ] All existing Go tests still passing after migration

## Concrete Work (from docs/MODULE_UPGRADE_PLANS.md)
- **P0** (Week 1): x/knowledge, x/billing, x/gov
- **P1** (Week 2): x/staking, x/vesting_rewards, x/emergency
- **P2** (Week 3): x/tokens, remaining modules
- Total: ~1,500 LOC across Params structs + migration + validation + tests

## The Beauty Argument (B4, B5)
When `effective_value = params × multiplier`, the system gains a natural separation:
- **Params** = what governance decided (Layer 2: Genome)
- **Multipliers** = what the protocol learned (Layer 3: Phenotype)
- **Constants** = what cannot change (Layer 1: DNA)

This 3-layer architecture is *elegant because it's true* — it maps to how living systems actually work. Beauty emerges from the understanding, not from decoration.

## Gates Targeted
| Gate | Current | Target | How |
|------|---------|--------|-----|
| J2 (Ownership) | FAIL | PASS | Params owned by governance, not hardcoded |
| J5 (Balance) | FAIL | PASS | Consistent Params pattern across all modules |
| B4 (Elegance) | FAIL | PARTIAL | 3-layer separation is structurally elegant |
| B5 (Coherence) | PARTIAL | PASS | All modules follow same Params pattern |

## Mastery Impact
- **Critical** for Justice: resolves the structural cause of J failures
- **Foundation** for Creativity: W4 (autopoiesis) cannot exist without this
- **Medium effort**: ~3 sessions, well-defined from MODULE_UPGRADE_PLANS.md
