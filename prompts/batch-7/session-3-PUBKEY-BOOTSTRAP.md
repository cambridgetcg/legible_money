# Session: PUBKEY-BOOTSTRAP — Fix account pubkey bootstrap issue for devnet/testnet

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Handle the SDK v0.50 pubkey bootstrap requirement gracefully

### Context
B6-6 PoT devnet testing discovered: genesis-funded accounts have `null` pubkey on-chain. Any transaction attempt before a bank MsgSend bootstraps the pubkey fails with "signature verification failed." This is confusing for operators and breaks automated test scripts.

### Option A: Devnet script fix (minimal)
Add a pubkey bootstrap phase to all devnet/testnet init scripts:
```bash
# Bootstrap pubkeys for all test accounts
for ACCOUNT in test1 test2 test3; do
  legbled tx bank send validator $ACCOUNT_ADDR 1ulgm --yes --gas 200000 --fees 200000ulgm
  sleep 3
done
```

Update:
- `devnet/scripts/cosmos-devnet.sh`
- `scripts/testnet-genesis.sh`
- `deployment/docker/init-cosmos-docker.sh`
- `devnet/scripts/test-lgm-modules.sh` (add bootstrap phase before module tests)

### Option B: Ante handler fix (proper)
Add a decorator that auto-sets pubkey from tx signatures when the account exists but has no pubkey. The SDK's `SetPubKeyDecorator` already does this — verify it's in the ante chain:

```bash
grep -n "SetPubKeyDecorator\|SetPubKey" app/ante.go app/ante_legible.go
```

If it's present, the issue might be ordering — the decorator needs to run before signature verification.

If it's NOT present, add it to the ante chain before `SigVerificationDecorator`.

### Step 1: Diagnose the actual cause
Read `app/ante.go` to understand the decorator chain order. Is `SetPubKeyDecorator` in the chain? If so, why isn't it working?

### Step 2: Implement the fix
- If Option B works: fix the ante chain (preferred — transparent to users)
- If Option B doesn't apply: implement Option A (script fix)

### Verify
On a fresh devnet:
```bash
# Fund an account at genesis
# Immediately try a non-bank tx (e.g., knowledge submit-claim) without first doing a bank send
# Should work without the "signature verification failed" error
```

## Report format
Write to reports/batch-7/SESSION-B7-3-PUBKEY-BOOTSTRAP.md:
- Root cause analysis (is SetPubKeyDecorator present? What order?)
- Fix implemented (Option A or B)
- Verification on fresh devnet
