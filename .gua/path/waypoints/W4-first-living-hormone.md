# W4: First Living Hormone (SSI)

> The protocol learns to feel stress. The first step from mechanism to organism.

## Why This Path
This is the **creative frontier** — where Legible Money stops being a well-engineered blockchain and starts being something new. No Cosmos SDK chain has autopoietic self-adaptation. The System Stress Index (SSI) is the simplest hormone: observe system health, modulate parameters. If this works, the protocol becomes alive.

Blocked by W2 (Genome Parameterization) — autopoiesis needs parameters to modulate.

## Success Criteria
1. [ ] `x/autopoiesis/keeper/observer.go`: reads chain metrics per block
2. [ ] `x/autopoiesis/keeper/hormones.go`: SSI computation (mempool pressure, validator participation, slash rate)
3. [ ] `x/autopoiesis/keeper/adaptation.go`: multiplier output with safety bounds [0.25, 4.0]
4. [ ] `x/autopoiesis/module.go`: BeginBlocker observes, EndBlocker adapts (one-block delay)
5. [ ] SSI multiplier applied to at least 1 consumer (e.g., slash severity in x/staking)
6. [ ] Rate limiting: ±5% per epoch maximum change
7. [ ] Emergency freeze: autopoiesis disables when x/emergency is halted
8. [ ] 20+ tests covering: normal adaptation, rate limiting, bounds, emergency freeze
9. [ ] Devnet demonstrates measurable parameter change over 100 blocks

## The Architecture (from RECURSIVE_GROWTH_FRAMEWORK.md)
```
BeginBlocker:
  observer.Collect(ctx)     // Read chain state → raw metrics

EndBlocker:
  hormones.Compute(ctx)     // Raw metrics → hormone values
  adaptation.Apply(ctx)     // Hormone values → multiplier adjustments (bounded)

Consumer (any module):
  effective = params.SlashFraction * autopoiesis.GetMultiplier("slash_severity")
```

## The Creativity Argument (X3, X5)
- **X3 (Path-Building)**: This is a new path — autopoietic blockchain consensus
- **X5 (Vision)**: The vision is a protocol that adapts to truth, not just processes it
- No other chain does this. PoW chains burn energy. PoS chains redistribute stake. PoT chains should *learn*.

## The Beauty Argument (B4, B6)
- **B4 (Elegance)**: `effective = base × multiplier` is the simplest possible adaptation interface
- **B6 (Clarity)**: The hormone metaphor makes the system legible — SSI *means* stress, CRI *means* confidence retention

## SSI Formula (from TS spec)
```
SSI = w1 * mempool_pressure + w2 * (1 - validator_participation) + w3 * slash_rate
    where w1=0.3, w2=0.4, w3=0.3

If SSI > 0.7:  increase slash severity multiplier (punish more under stress)
If SSI < 0.3:  decrease slash severity multiplier (relax under health)
```

## Gates Targeted
| Gate | Current | Target | How |
|------|---------|--------|-----|
| X3 (Path-Building) | PARTIAL | PASS | New module, novel concept, followable path |
| X5 (Vision) | FAIL | PARTIAL | Autopoietic vision partially realized |
| B4 (Elegance) | FAIL | PASS | params × multiplier is naturally elegant |
| B6 (Clarity) | PARTIAL | PASS | Hormone metaphor makes adaptation legible |

## Mastery Impact
- **Critical** for Creativity: this is the uncharted territory
- **High** for Beauty: the architecture is naturally elegant if done right
- **Foundation** for L3: L3 requires "original paths built, well-integrated"
- **Blocked by W2**: parameters must exist before they can be modulated
