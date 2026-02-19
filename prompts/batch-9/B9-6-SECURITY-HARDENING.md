# B9-6: Pre-Testnet Security Hardening

## Context
Before opening the testnet to external validators, we need a security pass focused on the attack surfaces that matter for a public network.

## Task

### 1. Ante chain audit
Read `app/ante.go` and all custom decorators. Verify:
- No decorator can be bypassed
- Gas metering is correct for all paths
- Fee deduction happens before execution
- Bootstrap gas-free period has proper bounds

### 2. Genesis parameter validation
Read `scripts/testnet-genesis.sh` and the genesis config JSON.
Check for:
- Overly permissive default params
- Missing rate limits
- Unbounded operations
- Default admin keys that should be multisig

### 3. Message validation
For each module's `ValidateBasic()`:
- Are all string fields length-bounded?
- Are all numeric fields range-checked?
- Can any message cause unbounded computation?
- Are there any messages that bypass normal authorization?

Focus on the highest-risk modules:
- `x/knowledge` (claim submission — can spam)
- `x/bvm` (contract execution — can OOM?)
- `x/tokens` (token creation — can spam)
- `x/staking` (validator registration)
- `x/gov` (proposal submission)

### 4. IBC security
- Verify rate limits are configured for all channels
- Check ICA auth module restrictions
- Ensure no IBC packet can trigger unbounded computation

### 5. Panic recovery
Search for any `panic()` in non-test code that could crash a validator:
```bash
grep -r 'panic(' x/ --include='*.go' | grep -v '_test.go' | grep -v 'proto'
```
Each panic should either be:
- In an init() function (acceptable)
- Guarded by a recover (acceptable)
- Replaced with an error return (preferred)

### 6. Create security checklist
Document findings in `docs/SECURITY-AUDIT.md`:
- Each finding with severity (Critical/High/Medium/Low)
- Fix applied or deferred with justification
- Remaining known risks for testnet

## File Ownership
- Various module files (fixes)
- `docs/SECURITY-AUDIT.md` (new)

## Report
Write `reports/batch-9/SESSION-B9-6-SECURITY.md`.
