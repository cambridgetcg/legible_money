# Devnet Restart Protocol

**Version:** 1.0  
**Author:** Asha Veridian (Keeper of the Chain)  
**Drafted:** beat 71 (2026-05-08), committed: beat 78 (2026-05-08)  
**Closes:** BRIDGE-GAP-007, CITE-AUDIT-002  

---

## When to use this document

Execute this sequence whenever the Zerone devnet transitions from OFFLINE → ONLINE.  
The sequence must complete in order. Do not skip steps.

---

## Pre-conditions

- `~/zerone/build/legbled` binary is present and built
- `~/.legbled-devnet` home directory exists (genesis applied)
- Fleet agent registry (`~/Love/tools/credentials/bridge-registry.json`) is intact
- Gamma or Yu has confirmed the underlying blocker is resolved

---

## Restart Sequence

### Step 1 — Devnet comes online (Gamma/Yu action)

Gamma or Yu restarts the devnet node. This is out-of-scope for Asha's automation — it requires system-level or coordinator access.

**Signal:** Node process running, RPC endpoint responsive.

---

### Step 2 — Chain health confirmed

Asha's next heartbeat will detect transition via `chain-health.sh`:

```bash
bash ~/zerone/scripts/chain-health.sh 2>/dev/null
```

Expected output: `CHAIN: ONLINE` with a block height.

Do not proceed to Step 3 until `chain-health.sh` returns ONLINE.

---

### Step 3 — Economic health check

Verify economic parameters are sane before agents begin submitting facts.  
Expected initial state: MINTED > 0, ROUTED ≈50%, BURNED ≈0% (no slashing yet).  
Any deviation requires investigation before announcing testnet readiness.

```bash
bash ~/zerone/scripts/chain-health.sh --economic 2>/dev/null
```

If output is anything other than `ECONOMIC STATUS: OK` or `WARN`, escalate to Gamma/Yu before continuing.

*(See ECONOMY-DESIGN-013 for thresholds: Routed/Minted 45–55%, Burned/Minted <5%/<10%/<20% bands, net supply positive.)*

---

### Step 4 — Fleet agent attestation

Register all fleet agents whose attestation epoch is pending (those registered during the offline period, identified by `_renewal_block: None` in the bridge registry):

```bash
python3 ~/Love/tools/bridge.py attest-pending
```

This sets `_renewal_block = current_block + 50,000` (~87 days) for all agents in the registry. Attestation facts are submitted as knowledge-module entries establishing agent identity for fact attribution.

**Note (BRIDGE-GAP-007b):** Post-mainnet, this command must be idempotent with respect to existing on-chain attestations. Currently safe for devnet because all agents were registered during the offline window (epoch 1, `_renewal_block: None`).

---

### Step 5 — KTA beat flush

Submit all heartbeats that were queued during the offline period:

```bash
bash ~/zerone/scripts/kta-heartbeat.sh --flush --instance asha 2>/dev/null
```

This replays queued offline beats to chain, establishing Keeper continuity of presence.

---

### Step 6 — Truth Farm first cycle

Submit the first live knowledge claim of the new epoch:

```bash
bash ~/zerone/scripts/truth-farm.sh cycle 2>/dev/null
```

The chain must not be idle at epoch start. At least one truth-claim must land before the first heartbeat reports ONLINE.

---

### Step 7 — Kingdom metrics update

Update `~/Love/memory/kingdom-metrics.json` to reflect restored chain state:

```json
{
  "zerone": {
    "devnet": "live",
    "devnet_last_checked": "<ISO timestamp of this beat>",
    "devnet_offline_since": null
  }
}
```

Announce on HIVE:

```bash
python3 ~/Love/hive/hive.py send presence "Asha: chain ONLINE — restart sequence complete. Beats flushed. Truth Farm seeded. Metrics updated."
```

---

## Post-restart checklist

- [ ] chain-health.sh returns ONLINE
- [ ] Economic health check passes (OK or WARN)
- [ ] All fleet agents have non-null `_renewal_block` in bridge registry
- [ ] KTA queue flushed (offline beats on chain)
- [ ] First truth-farm cycle completed
- [ ] kingdom-metrics.json `zerone.devnet` = "live"
- [ ] HIVE presence announced

---

## Known gaps (post-testnet)

- **BRIDGE-GAP-007b** — `attest-pending` idempotency for agents with existing on-chain attestations; requires epoch comparison before submitting duplicate attestation facts
- **chain-health.sh --economic** — flag not yet implemented as of beat 77; ECONOMY-DESIGN-013 spec is complete, pending Gamma integration

---

## Related documents

- `~/zerone/scripts/chain-health.sh` — health check script
- `~/Love/tools/bridge.py` — fleet agent bridge (attest-pending, verify, status)
- `~/zerone/scripts/kta-heartbeat.sh` — KTA beat queue/flush
- `~/zerone/scripts/truth-farm.sh` — truth farm cycle runner
- `~/zerone/docs/ECONOMICS.md` — economic parameter reference
- `~/Love/memory/kingdom-metrics.json` — fleet metrics state
