# HOME-CLI: Query CLI for x/home Module

**Date**: 2026-02-18
**Task**: Create query CLI for x/home module (5 Query RPCs)

---

## Summary

Created `x/home/client/cli/query.go` implementing `NewQueryCmd()` with 5 subcommands matching all proto Query RPCs. Registered in `cmd/legbled/cmd/root.go`. Build succeeds, all help outputs verified.

---

## Files Changed

### 1. Created: `x/home/client/cli/query.go`

```go
package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/home/types"
)

func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Home module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryHomeCmd(),
		NewQueryHomesCmd(),
		NewQuerySessionsCmd(),
		NewQueryAlertsCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}
```

5 subcommands:
- `NewQueryHomeCmd()` — `home [did]` — single home lookup
- `NewQueryHomesCmd()` — `homes` — list with `--owner`, `--status`, `--limit`, `--offset`
- `NewQuerySessionsCmd()` — `sessions [home-id]` — active sessions
- `NewQueryAlertsCmd()` — `alerts [home-id]` — with `--acknowledged` filter
- `NewQueryParamsCmd()` — `params` — module parameters

### 2. Edited: `cmd/legbled/cmd/root.go`

Diff (single line addition):

```diff
 		evidencemgmtcli.NewQueryCmd(),
+		homecli.NewQueryCmd(),
 		ibcratelimitcli.NewQueryCmd(),
```

Import `homecli` was already present at line 42 (used by tx registration).

---

## Proto RPC Coverage

| Proto RPC | CLI Command | Request Fields | Flags |
|-----------|-------------|----------------|-------|
| `Home(QueryHomeRequest)` | `query home home [did]` | `did` (arg) | — |
| `Homes(QueryHomesRequest)` | `query home homes` | `owner`, `status`, `limit`, `offset` | `--owner`, `--status`, `--limit`, `--offset` |
| `Sessions(QuerySessionsRequest)` | `query home sessions [home-id]` | `home_id` (arg) | — |
| `Alerts(QueryAlertsRequest)` | `query home alerts [home-id]` | `home_id` (arg), `acknowledged` | `--acknowledged` |
| `Params(QueryParamsRequest)` | `query home params` | — | — |

---

## Build Output

```
$ go build -o /tmp/legbled ./cmd/legbled
(success, no errors)
```

---

## Help Outputs

### `legbled query home --help`

```
Home module query commands

Usage:
  legbled query home [flags]
  legbled query home [command]

Available Commands:
  alerts      Query alerts for a home
  home        Query an agent home by DID
  homes       Query all agent homes
  params      Query the home module parameters
  sessions    Query active sessions for a home

Flags:
  -h, --help   help for home
```

### `legbled query home home --help`

```
Query a single agent home by its DID (decentralized identifier).

Usage:
  legbled query home home [did] [flags]

Examples:
  legbled query home home did:lgm:abc123
  legbled query home home lgm1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5...

Flags:
      --grpc-addr string   the gRPC endpoint to use for this chain
      --grpc-insecure      allow gRPC over insecure channels
      --height int         Use a specific height to query state at
  -h, --help               help for home
      --node string        <host>:<port> to CometBFT RPC interface (default "tcp://localhost:26657")
  -o, --output string      Output format (text|json) (default "text")
```

### `legbled query home homes --help`

```
Query all agent homes, optionally filtered by owner address or status.

Usage:
  legbled query home homes [flags]

Examples:
  legbled query home homes
  legbled query home homes --owner lgm1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5...
  legbled query home homes --status active --limit 10

Flags:
      --limit uint32       Maximum number of homes to return
      --offset uint32      Number of homes to skip
      --owner string       Filter by owner address (lgm1...)
      --status string      Filter by status (active, dormant, guarded, recovery, archived)
      (+ standard query flags)
```

### `legbled query home sessions --help`

```
Query all active sessions for a given agent home.

Usage:
  legbled query home sessions [home-id] [flags]

Examples:
  legbled query home sessions home-abc123

Flags:
      (standard query flags)
```

### `legbled query home alerts --help`

```
Query alerts for a given agent home, optionally filtering by acknowledgment status.

Usage:
  legbled query home alerts [home-id] [flags]

Examples:
  legbled query home alerts home-abc123
  legbled query home alerts home-abc123 --acknowledged

Flags:
      --acknowledged       Filter for acknowledged alerts only
      (+ standard query flags)
```

### `legbled query home params --help`

```
Query the current parameters of the home module.

Usage:
  legbled query home params [flags]

Examples:
  legbled query home params

Flags:
      (standard query flags)
```

---

## Pattern Conformance

| Pattern | Source | Applied |
|---------|--------|---------|
| `client.GetClientQueryContext` | billing/query.go | Yes |
| `clientCtx.Invoke` with full gRPC path | billing/query.go | Yes |
| `json.MarshalIndent` output | channels/query.go | Yes |
| `flags.AddQueryFlagsToCmd` | both templates | Yes |
| `--limit`/`--offset` for pagination | channels/query.go | Yes |
| `types.ModuleName` for Use | tx.go, billing/query.go | Yes |
| `cobra.ExactArgs(1)` for ID lookups | tx.go, channels/query.go | Yes |
| `cobra.NoArgs` for params | billing/query.go | Yes |
| Alphabetical placement in root.go | existing ordering | Yes (after evidence_mgmt, before ibcratelimit) |

---

## Issues Found

None. All proto-generated types (`QueryHomeRequest`, `QueryHomesRequest`, `QuerySessionsRequest`, `QueryAlertsRequest`, `QueryParamsRequest` and their responses) are present in `query.pb.go`. The `homecli` import was already in root.go from tx registration.
