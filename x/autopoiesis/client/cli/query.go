package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/legible-money/legible/x/autopoiesis/types"
)

// NewQueryCmd returns the query commands for the autopoiesis module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Autopoiesis module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// No query server — queries will be added when gRPC query service is implemented.

	return queryCmd
}
