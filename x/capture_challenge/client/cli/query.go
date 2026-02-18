package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

// NewQueryCmd returns the query commands for the capture challenge module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capture challenge module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryChallengeCmd(),
		NewQueryActiveChallengesCmd(),
		NewQueryBountyPoolCmd(),
	)

	return queryCmd
}

// NewQueryChallengeCmd returns the command to query a single challenge.
func NewQueryChallengeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge [challenge-id]",
		Short: "Query a capture challenge by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryChallengeRequest{ChallengeId: args[0]}
			resp := &types.QueryChallengeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_challenge.v1.Query/Challenge", req, resp); err != nil {
				return fmt.Errorf("failed to query challenge: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveChallengesCmd returns the command to list active challenges.
func NewQueryActiveChallengesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-challenges [domain]",
		Short: "List active capture challenges, optionally filtered by domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryActiveChallengesRequest{Domain: args[0]}
			resp := &types.QueryActiveChallengesResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_challenge.v1.Query/ActiveChallenges", req, resp); err != nil {
				return fmt.Errorf("failed to query active challenges: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryBountyPoolCmd returns the command to query a domain's bounty pool.
func NewQueryBountyPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bounty-pool [domain]",
		Short: "Query the bounty pool for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryBountyPoolRequest{Domain: args[0]}
			resp := &types.QueryBountyPoolResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_challenge.v1.Query/BountyPool", req, resp); err != nil {
				return fmt.Errorf("failed to query bounty pool: %w", err)
			}

			out, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
