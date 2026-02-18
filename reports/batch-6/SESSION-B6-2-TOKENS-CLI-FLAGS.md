# SESSION-B6-2: TOKENS-CLI-FLAGS

## Fix: F-3 — `create-token` CLI missing feature flags

### Problem

The `create-token` CLI command had no flags for `--mintable`, `--burnable`, `--pausable`, or `--wrappable`. All tokens created via CLI got `Features: nil` (defaulted to `&TokenFeatures{}` with all `false`), making mint/burn/pause/wrap operations impossible from the CLI.

### Changes

**File: `x/tokens/client/cli/tx.go`** (`NewCreateTokenCmd`)

Added 4 boolean flags:

```go
cmd.Flags().Bool("mintable", false, "Enable minting for this token")
cmd.Flags().Bool("burnable", false, "Enable burning for this token")
cmd.Flags().Bool("pausable", false, "Enable pause/unpause for this token")
cmd.Flags().Bool("wrappable", false, "Enable wrapping into IBC-compatible coins")
```

In RunE, flags are read and populate `msg.Features`:

```go
mintable, _ := cmd.Flags().GetBool("mintable")
burnable, _ := cmd.Flags().GetBool("burnable")
pausable, _ := cmd.Flags().GetBool("pausable")
wrappable, _ := cmd.Flags().GetBool("wrappable")

msg := &types.MsgCreateToken{
    // ...
    Features: &types.TokenFeatures{
        Mintable:  mintable,
        Burnable:  burnable,
        Pausable:  pausable,
        Wrappable: wrappable,
    },
}
```

### Related Command Audit

| Command | Feature check | Location | Status |
|---------|--------------|----------|--------|
| `mint-token` | `Features.Mintable` | msg_server.go:125 | Correct |
| `burn-token` | `Features.Burnable` | msg_server.go:182 | Correct |
| `pause-token` | `Features.Pausable` | msg_server.go:367 | Correct |
| `unpause-token` | `Features.Pausable` | msg_server.go:401 | Correct |
| `wrap-token` (WrapToken) | `Features.Wrappable` | msg_server.go:883 | Correct |

All downstream handlers correctly gate on the corresponding feature flag with nil-safe checks (`token.Features == nil || !token.Features.Xxx`).

### Verification

- `go build ./...` — clean
- `go test ./x/tokens/... -count=1` — keeper tests pass (0.693s)
- `legbled tx tokens create-token --help` — shows all 4 new flags

### Usage Example

```bash
# Create a mintable+burnable token
legbled tx tokens create-token "Test Token" TST 6 1000000 \
  --mintable --burnable \
  --from validator --yes --gas 300000 --fees 300000ulgm

# Mint additional tokens (now succeeds)
legbled tx tokens mint-token <token-id> <to-addr> 500000 \
  --from validator --yes --gas 300000 --fees 300000ulgm
```
