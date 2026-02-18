package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/home/types"
)

// NewQueryCmd returns the query commands for the home module.
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

// NewQueryHomeCmd returns the command to query a single agent home by DID.
func NewQueryHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "home [did]",
		Short: "Query an agent home by DID",
		Long:  "Query a single agent home by its DID (decentralized identifier).",
		Example: `  legbled query home home did:lgm:abc123
  legbled query home home lgm1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryHomeRequest{Did: args[0]}
			res := &types.QueryHomeResponse{}

			if err := clientCtx.Invoke(cmd.Context(), "/legible.home.v1.Query/Home", req, res); err != nil {
				return fmt.Errorf("failed to query home: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryHomesCmd returns the command to query all homes with optional filters.
func NewQueryHomesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "homes",
		Short: "Query all agent homes",
		Long:  "Query all agent homes, optionally filtered by owner address or status.",
		Example: `  legbled query home homes
  legbled query home homes --owner lgm1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5...
  legbled query home homes --status active --limit 10`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			owner, _ := cmd.Flags().GetString("owner")
			status, _ := cmd.Flags().GetString("status")
			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryHomesRequest{
				Owner:  owner,
				Status: status,
				Limit:  limit,
				Offset: offset,
			}
			res := &types.QueryHomesResponse{}

			if err := clientCtx.Invoke(cmd.Context(), "/legible.home.v1.Query/Homes", req, res); err != nil {
				return fmt.Errorf("failed to query homes: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().String("owner", "", "Filter by owner address (lgm1...)")
	cmd.Flags().String("status", "", "Filter by status (active, dormant, guarded, recovery, archived)")
	cmd.Flags().Uint32("limit", 0, "Maximum number of homes to return")
	cmd.Flags().Uint32("offset", 0, "Number of homes to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySessionsCmd returns the command to query active sessions for a home.
func NewQuerySessionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sessions [home-id]",
		Short:   "Query active sessions for a home",
		Long:    "Query all active sessions for a given agent home.",
		Example: "  legbled query home sessions home-abc123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QuerySessionsRequest{HomeId: args[0]}
			res := &types.QuerySessionsResponse{}

			if err := clientCtx.Invoke(cmd.Context(), "/legible.home.v1.Query/Sessions", req, res); err != nil {
				return fmt.Errorf("failed to query sessions: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAlertsCmd returns the command to query alerts for a home.
func NewQueryAlertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts [home-id]",
		Short: "Query alerts for a home",
		Long:  "Query alerts for a given agent home, optionally filtering by acknowledgment status.",
		Example: `  legbled query home alerts home-abc123
  legbled query home alerts home-abc123 --acknowledged`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			acknowledged, _ := cmd.Flags().GetBool("acknowledged")

			req := &types.QueryAlertsRequest{
				HomeId:       args[0],
				Acknowledged: acknowledged,
			}
			res := &types.QueryAlertsResponse{}

			if err := clientCtx.Invoke(cmd.Context(), "/legible.home.v1.Query/Alerts", req, res); err != nil {
				return fmt.Errorf("failed to query alerts: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	cmd.Flags().Bool("acknowledged", false, "Filter for acknowledged alerts only")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query home module parameters.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "params",
		Short:   "Query the home module parameters",
		Long:    "Query the current parameters of the home module.",
		Example: "  legbled query home params",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}

			if err := clientCtx.Invoke(cmd.Context(), "/legible.home.v1.Query/Params", req, res); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
