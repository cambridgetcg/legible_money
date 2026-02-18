# Session: EVENT-EXTRACTION — Fix event emission for claim IDs and LIP IDs

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Ensure all transaction handlers emit proper events with extractable IDs

### Context
E2E test A3 was skipped because the claim ID couldn't be extracted from tx events after `MsgSubmitClaim`. This is a pattern problem — if handlers don't emit events with resource IDs, clients can't track what they created without querying separately.

### Step 1: Audit event emission across key modules
For each module, check whether the handlers emit events with the created resource's ID:

**Knowledge module** (`x/knowledge/keeper/msg_server.go`):
- `MsgSubmitClaim` → should emit `claim_id` in events
- `MsgSubmitCommitment` → should emit `round_id`
- `MsgSubmitReveal` → should emit `round_id`

**Gov module** (`x/gov/keeper/msg_server.go`):
- `MsgSubmitLIP` → should emit `lip_id` or `proposal_id`
- `MsgCastVote` → should emit `lip_id` + `voter`

**Home module** (`x/home/keeper/msg_server.go`):
- `MsgCreateHome` → should emit `home_id`
- `MsgCreateSession` → should emit `session_id`

**Tokens module** (`x/tokens/keeper/msg_server.go`):
- `MsgCreateToken` → should emit `token_symbol` or `token_id`

**BVM module** (`x/bvm/keeper/msg_server.go`):
- `MsgDeployContract` → should emit `contract_address`
- `MsgCallContract` → should emit `contract_address` + `result`

### Step 2: Add missing events
For each handler that doesn't emit a proper event with the resource ID:

```go
ctx.EventManager().EmitEvent(sdk.NewEvent(
    "submit_claim",
    sdk.NewAttribute("claim_id", claim.Id),
    sdk.NewAttribute("domain", claim.Domain),
    sdk.NewAttribute("submitter", msg.Submitter),
))
```

Follow the Cosmos SDK event convention:
- Event type = snake_case action name
- Attributes = key-value pairs with string values
- Include the resource ID, the actor, and key metadata

### Step 3: Verify events are queryable
After adding events, verify they appear in tx results:
```bash
# On devnet
legbled tx knowledge submit-claim --domain general --content "test claim" --from validator --yes --gas 300000 --fees 300000ulgm
# Get tx hash from output, then:
legbled query tx <hash> --output json | jq '.events[] | select(.type == "submit_claim")'
```

### Step 4: Update E2E test scripts
Update `devnet/scripts/test-lgm-modules.sh` to extract IDs from events:
```bash
# Extract claim_id from tx events
CLAIM_ID=$(legbled query tx $TX_HASH --output json | jq -r '.events[] | select(.type=="submit_claim") | .attributes[] | select(.key=="claim_id") | .value')
```

Fix the A3 skip (claim ID query) and any other event-dependent tests.

### Verify
- `go build ./...` clean
- `go test ./... -count=1` passes
- On devnet: previously skipped A3 test now passes

## Report format
Write to reports/batch-6/SESSION-B6-5-EVENT-EXTRACTION.md:
- Event audit table: module → handler → events emitted (before/after)
- Events added with exact code
- E2E test script updates
- Devnet verification
