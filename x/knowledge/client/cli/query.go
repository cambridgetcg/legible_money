package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/knowledge/types"
)

// NewQueryCmd returns the query commands for the knowledge module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Knowledge module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryFactCmd(),
		NewQueryFactsCmd(),
		NewQueryFactsByDomainCmd(),
		NewQueryClaimCmd(),
		NewQueryRoundCmd(),
		NewQueryActiveRoundsCmd(),
		NewQueryDomainCmd(),
		NewQueryCounterFactCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query knowledge module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the knowledge module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryFactCmd returns the command to query a single fact.
func NewQueryFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fact [fact-id]",
		Short: "Query a fact by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Fact(cmd.Context(), &types.QueryFactRequest{FactId: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query fact: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryFactsCmd returns the command to list all facts.
func NewQueryFactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts",
		Short: "List all verified facts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Facts(cmd.Context(), &types.QueryFactsRequest{
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return fmt.Errorf("failed to query facts: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint32("limit", 20, "Maximum number of facts to return")
	cmd.Flags().Uint32("offset", 0, "Number of facts to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryFactsByDomainCmd returns the command to query facts by domain.
func NewQueryFactsByDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facts-by-domain [domain]",
		Short: "Query facts in a specific domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.FactsByDomain(cmd.Context(), &types.QueryFactsByDomainRequest{
				Domain: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query facts: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryClaimCmd returns the command to query a single claim.
func NewQueryClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim [claim-id]",
		Short: "Query a claim by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Claim(cmd.Context(), &types.QueryClaimRequest{ClaimId: args[0]})
			if err != nil {
				return fmt.Errorf("failed to query claim: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryRoundCmd returns the command to query a verification round.
func NewQueryRoundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "round [round-id]",
		Short: "Query a verification round by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.VerificationRound(cmd.Context(), &types.QueryVerificationRoundRequest{
				RoundId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query round: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveRoundsCmd returns the command to list active verification rounds.
func NewQueryActiveRoundsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-rounds",
		Short: "List all active verification rounds",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ActiveRounds(cmd.Context(), &types.QueryActiveRoundsRequest{})
			if err != nil {
				return fmt.Errorf("failed to query active rounds: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDomainCmd returns the command to query a domain.
func NewQueryDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain [domain-id]",
		Short: "Query a knowledge domain by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Domain(cmd.Context(), &types.QueryDomainRequest{
				DomainId: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryCounterFactCmd returns the command to query a counter-fact.
func NewQueryCounterFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "counter-fact [counter-fact-id]",
		Short: "Query a counter-fact by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.CounterFact(cmd.Context(), &types.QueryCounterFactRequest{
				CounterFactId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query counter-fact: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
