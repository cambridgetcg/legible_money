# x/alignment — 5-Sensor Monitoring

## Purpose

Closed-loop alignment monitoring with 5 sensors detecting system misalignment. Runs SENSE, JUDGE, CORRECT, REFLECT cycle at epoch boundaries. Triggers multiplier corrections in autopoiesis based on severity thresholds. Meta-loop every 10 epochs with damping/smoothing.

## Key Concepts

- **5 Sensors**:
  - **TDI** (Truth Drift Index): Reversal rate, challenge asymmetry, unanimity excess
  - **DCI** (Depth Coherence Index): Citation depth, cross-domain, orphan rate
  - **HI** (Harmony Index): Variance, oscillation, bound saturation
  - **FI** (Fairness Index): Reward Gini, new entrant access, whale dominance
  - **NI** (Novelty Index): New domains, novel claims, cross-domain bridges
- **Severity Levels**: normal → elevated → severe → critical
- **Per-Dimension Enable**: Gradual rollout with individual sensor flags
- **Passive**: No CLI commands, no direct events (corrections logged by autopoiesis)

## CLI Commands

None (passive observational module).

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| enabled | true | Module activation |
| epoch_length_blocks | 1,111 | Should match autopoiesis |
| max_corrections | 3 | Max corrections per epoch |
| truth_enabled | true | Enable Truth sensor |
| understanding_enabled | true | Enable Understanding sensor |
| beauty_enabled | true | Enable Beauty sensor |
| justice_enabled | true | Enable Justice sensor |
| creativity_enabled | true | Enable Creativity sensor |

Plus per-dimension severity threshold overrides.

## Events

None (corrections applied through autopoiesis multipliers).

## Dependencies

KnowledgeKeeper, StakingKeeper, OntologyKeeper, AutopoiesisKeeper, EmergencyKeeper
