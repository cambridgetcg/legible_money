package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/disputes/types"
)

// NewQueryCmd returns the query commands for the disputes module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Disputes module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryDisputeCmd(),
		NewQueryDisputesByFactCmd(),
		NewQueryActiveDisputesCmd(),
		NewQueryEvidenceCmd(),
		NewQueryVotesCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query disputes module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the disputes module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.disputes.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDisputeCmd returns the command to query a single dispute.
func NewQueryDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute [dispute-id]",
		Short: "Query a dispute by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDisputeRequest{DisputeId: args[0]}
			resp := &types.QueryDisputeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.disputes.v1.Query/Dispute", req, resp); err != nil {
				return fmt.Errorf("failed to query dispute: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDisputesByFactCmd returns the command to query disputes by fact.
func NewQueryDisputesByFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disputes-by-fact [fact-id]",
		Short: "Query all disputes for a fact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDisputesByFactRequest{FactId: args[0]}
			resp := &types.QueryDisputesByFactResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.disputes.v1.Query/DisputesByFact", req, resp); err != nil {
				return fmt.Errorf("failed to query disputes: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveDisputesCmd returns the command to list active disputes.
func NewQueryActiveDisputesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-disputes",
		Short: "List all active (non-resolved, non-expired) disputes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryActiveDisputesRequest{}
			resp := &types.QueryActiveDisputesResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.disputes.v1.Query/ActiveDisputes", req, resp); err != nil {
				return fmt.Errorf("failed to query active disputes: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEvidenceCmd returns the command to query evidence for a dispute.
func NewQueryEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evidence [dispute-id]",
		Short: "Query all evidence for a dispute",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryEvidenceRequest{DisputeId: args[0]}
			resp := &types.QueryEvidenceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.disputes.v1.Query/Evidence", req, resp); err != nil {
				return fmt.Errorf("failed to query evidence: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryVotesCmd returns the command to query votes for a dispute.
func NewQueryVotesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "votes [dispute-id]",
		Short: "Query all votes for a dispute",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryVotesRequest{DisputeId: args[0]}
			resp := &types.QueryVotesResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.disputes.v1.Query/Votes", req, resp); err != nil {
				return fmt.Errorf("failed to query votes: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
