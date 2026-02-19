# x/capture_defense — Capture Detection & Reputation

## Purpose

Detects validator capture via Herfindahl-Hirschman Index (HHI) concentration analysis. Applies reputation decay, manages domain qualifications with capture-specific suspension, and coordinates with emergency module on critical HHI thresholds.

## Key Concepts

- **HHI Detection**: Herfindahl index triggers at 0.4 (400,000 BPS), pauses at 0.6
- **Reputation Decay**: Isolated reputation decays over 77,777 blocks
- **Amortized Detection**: Runs every 777 blocks to avoid per-block overhead
- **Critical HHI**: Triggers emergency coordination (domain pause)

## CLI Commands

### Transactions
```bash
legbled tx capture_defense request-qualification [domain] --from alice
legbled tx capture_defense endorse-qualification [validator] [domain] --from qualified-validator
```

No query commands in current revision (read-only metrics).

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| StakeLockPeriod | 7,777 blocks | Qualification stake lock |
| HerfindahlTriggerThreshold | 400,000 BPS | HHI alert threshold |
| DecayIntervalBlocks | 7,777 | Reputation decay interval |
| DetectionIntervalBlocks | 777 | Capture detection frequency |
| MaxExpirationsPerBlock | 100 | Amortized expiry limit |

## Events

`qualification_expired`, `capture_detected`, `domain_paused_critical_hhi`, `reputation_decayed`

## Dependencies

StakingKeeper, OntologyKeeper, BankKeeper, EmergencyKeeper

## BeginBlock

Expires qualifications (max 100 per block for amortization).
