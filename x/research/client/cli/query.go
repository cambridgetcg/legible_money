package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/research/types"
)

// NewQueryCmd returns the query commands for the research module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Research module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryResearchCmd(),
		NewQuerySubmissionsCmd(),
		NewQueryBountyCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryResearchCmd returns the command to query a single research submission.
func NewQueryResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "research [research-id]",
		Short: "Query a research submission by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryResearchRequest{ResearchId: args[0]}
			resp := &types.QueryResearchResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.research.v1.Query/Research", req, resp); err != nil {
				return fmt.Errorf("failed to query research: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySubmissionsCmd returns the command to list research submissions.
func NewQuerySubmissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submissions",
		Short: "List research submissions, optionally filtered by status and domain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			status, _ := cmd.Flags().GetString("status")
			domain, _ := cmd.Flags().GetString("domain")

			req := &types.QuerySubmissionsRequest{
				Status: status,
				Domain: domain,
			}
			resp := &types.QuerySubmissionsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.research.v1.Query/Submissions", req, resp); err != nil {
				return fmt.Errorf("failed to query submissions: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}

			return clientCtx.PrintBytes(out)
		},
	}

	cmd.Flags().String("status", "", "Filter by status (submitted, under_review, accepted, rejected, challenged)")
	cmd.Flags().String("domain", "", "Filter by knowledge domain")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryBountyCmd returns the command to query a single bounty.
func NewQueryBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bounty [bounty-id]",
		Short: "Query a research bounty by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryBountyRequest{BountyId: args[0]}
			resp := &types.QueryBountyResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.research.v1.Query/Bounty", req, resp); err != nil {
				return fmt.Errorf("failed to query bounty: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query research module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the research module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.research.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}

			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
