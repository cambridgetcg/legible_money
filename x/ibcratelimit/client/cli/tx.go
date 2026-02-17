package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/legible-money/legible/x/ibcratelimit/types"
)

// NewTxCmd returns the transaction commands for the ibcratelimit module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "IBC rate limit module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// All ibcratelimit messages are governance-only (MsgUpdateParams).
	// No user-facing transaction subcommands.

	return txCmd
}
