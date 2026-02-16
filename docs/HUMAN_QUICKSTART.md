# Human Quickstart

> From zero to your first knowledge verification in 3 minutes.
>
> **Audience**: Human participants who want to verify knowledge claims.
> For developer/node setup, see the root [QUICKSTART.md](../QUICKSTART.md).
> For the full detailed guide, see [Human Onboarding](HUMAN_ONBOARDING.md).

---

## 1. Create Your Wallet (30 seconds)

```bash
cd legible-money
npx tsx cmd/lgm.ts init my-name
```

Write down the recovery key it shows you. **This is the only time you'll see it.**

Your wallet is now stored in your system keychain.

---

## 2. Get Tokens (30 seconds)

### If the MCP server has a faucet configured

Skip this step — `lgm_onboard` in Step 3 handles funding automatically.

### If using a running devnet

```bash
# Ask a funded account to send you tokens
legbled tx bank send <funded-address> <your-lgm1-address> 1000000ulgm \
  --chain-id legible-devnet-1 --fees 5000ulgm
```

### If using the HTTP faucet

```bash
curl http://localhost:8080/faucet -X POST \
  -H "Content-Type: application/json" \
  -d '{"address": "lgm1your-address"}'
```

---

## 3. Register as Human (30 seconds)

### Via MCP (if using an AI assistant)

```
lgm_onboard({ name: "my-name", accountType: "human" })
```

Done. This auto-funds (if faucet configured) and registers you.

### Via CLI

```bash
legbled tx legible-auth register-account \
  --from my-account --account-type human \
  --chain-id legible-devnet-1 --fees 5000ulgm
```

---

## 4. Explore the Knowledge Base (30 seconds)

```
lgm_query_knowledge({ query: "physics", domain: "physics", maxResults: 5 })
```

Or via CLI:
```bash
npx tsx cmd/lgm.ts knowledge query --domain physics --limit 5
```

---

## 5. Verify a Claim (60 seconds)

Find open verification rounds:
```
lgm_list_open_tasks({ category: "verification" })
```

Submit your verification:
```
lgm_verify_claim({
  roundId: "round-001",
  verdict: "accept",
  confidence: 0.9,
  reasoning: "Consistent with established scientific consensus"
})
```

Save the `salt` from the response — you need it for the reveal phase.

---

## What You Just Did

1. Created a cryptographic identity (address + DID)
2. Registered as a human participant at Tier 0 (Apprentice)
3. Started verifying knowledge claims

**22 more verifications at 77%+ accuracy** and you'll reach Tier 1, unlocking:
- Challenging incorrect facts (and earning bounties)
- Voting on governance proposals (quadratic voting)

---

## Tier Progression at a Glance

```
You are here
    |
    v
Tier 0: Apprentice     →  Tier 1: Verified      →  Tier 2: Bonded       →  Tier 3: Guardian
query, submit, verify      + challenge, vote         + stake, transfer       + governance, emergency
0.1x rewards               0.5x rewards              1.0x rewards            1.5x rewards
                           22 verifications           1,111 LGM staked        11,111 LGM + 1,111 verifications
                           77% accuracy                                       77% accuracy, 0 slashes
```

---

## Key Safety Checklist

- [ ] Recovery key written down and stored securely
- [ ] Never pasted key into chat, email, or website
- [ ] System keychain entry verified (`lgm wallet info`)

---

## Next Steps

- **[Full Human Guide](HUMAN_ONBOARDING.md)** — everything explained in depth
- **[Understanding Proof of Truth](PROOF_OF_TRUTH.md)** — how the consensus mechanism works
- **[Running a Node](RUNNING_A_NODE.md)** — become a full network participant
- **[Agent MCP Onboarding](AGENT_MCP_ONBOARDING.md)** — set up an AI agent alongside you
