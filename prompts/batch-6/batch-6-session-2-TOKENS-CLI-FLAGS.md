# Session: TOKENS-CLI-FLAGS — Add missing feature flags to tokens CLI

You are working in the legible_money Cosmos SDK chain codebase at ~/Desktop/legible_money.

## Task: Fix F-3 — tokens CLI missing --mintable/--burnable/--pausable/--wrappable flags

### Context
E2E tests (R5-6) found that `create-token` CLI has no feature flags. All tokens created via CLI get `Features: &TokenFeatures{}` (all false), making mint/burn/pause/wrap operations impossible.

### Step 1: Read current state
1. `x/tokens/client/cli/tx.go` — find the `create-token` command, see what flags exist
2. `x/tokens/types/state.pb.go` or `types.go` — find `TokenFeatures` struct fields
3. `x/tokens/keeper/msg_server.go` — verify how `MsgCreateToken.Features` is used

### Step 2: Add flags
In the `create-token` command function:

```go
cmd.Flags().Bool("mintable", false, "Enable minting for this token")
cmd.Flags().Bool("burnable", false, "Enable burning for this token")
cmd.Flags().Bool("pausable", false, "Enable pause/unpause for this token")
cmd.Flags().Bool("wrappable", false, "Enable wrapping for this token")
```

In the command's RunE function, read the flags and populate the message:

```go
mintable, _ := cmd.Flags().GetBool("mintable")
burnable, _ := cmd.Flags().GetBool("burnable")
pausable, _ := cmd.Flags().GetBool("pausable")
wrappable, _ := cmd.Flags().GetBool("wrappable")

msg.Features = &types.TokenFeatures{
    Mintable:  mintable,
    Burnable:  burnable,
    Pausable:  pausable,
    Wrappable: wrappable,
}
```

### Step 3: Also check these related commands
- `mint-token` — does it check Features.Mintable? (Should already from msg_server)
- `burn-token` — does it check Features.Burnable?
- Is there a `pause-token` command? Does it check Features.Pausable?

### Step 4: Verify
- `go build ./...` clean
- `go test ./x/tokens/... -count=1` passes
- `legbled tx tokens create-token --help` shows the new flags
- On devnet:
  ```bash
  legbled tx tokens create-token TST2 "Test Token 2" --mintable --burnable --supply 1000000 --from validator --yes --gas 300000 --fees 300000ulgm
  legbled tx tokens mint-token TST2 500000 --from validator --yes --gas 300000 --fees 300000ulgm
  ```
  Mint should succeed (code 0).

## Report format
Write to reports/batch-6/SESSION-B6-2-TOKENS-CLI-FLAGS.md:
- Flags added with exact code
- Related command audit (mint/burn/pause/wrap CLI)
- Build + test output
- Devnet verification if possible
