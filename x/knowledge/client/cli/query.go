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
		NewQueryDomainsCmd(),
		NewQueryDomainsByStratumCmd(),
		NewQueryDomainChildrenCmd(),
		NewQueryDomainPathCmd(),
		NewQueryStratumCmd(),
		NewQueryStrataCmd(),
		NewQueryStratumRegistryCmd(),
		NewQueryValidateDependencyCmd(),
		NewQueryCalculateConfidenceCmd(),
		NewQueryDomainProposalCmd(),
		NewQueryDomainProposalsCmd(),
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

// ---------- Ontology & Domain Hierarchy Queries ----------

// NewQueryDomainsCmd returns the command to list domains with filters.
func NewQueryDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "List knowledge domains with optional filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			stratum, _ := cmd.Flags().GetString("stratum")
			parent, _ := cmd.Flags().GetString("parent")
			pattern, _ := cmd.Flags().GetString("pattern")
			limit, _ := cmd.Flags().GetUint64("limit")
			offset, _ := cmd.Flags().GetUint64("offset")

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Domains(cmd.Context(), &types.QueryDomainsRequest{
				Stratum:      stratum,
				ParentDomain: parent,
				Pattern:      pattern,
				Limit:        limit,
				Offset:       offset,
			})
			if err != nil {
				return fmt.Errorf("failed to query domains: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().String("stratum", "", "Filter by stratum")
	cmd.Flags().String("parent", "", "Filter by parent domain")
	cmd.Flags().String("pattern", "", "Pattern match (e.g., \"physics.*\")")
	cmd.Flags().Uint64("limit", 20, "Maximum number of domains to return")
	cmd.Flags().Uint64("offset", 0, "Number of domains to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryDomainsByStratumCmd returns the command to query domains in a stratum.
func NewQueryDomainsByStratumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains-by-stratum [stratum-id]",
		Short: "Query all domains in a specific stratum",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DomainsByStratum(cmd.Context(), &types.QueryDomainsByStratumRequest{
				StratumId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query domains: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDomainChildrenCmd returns the command to query child domains.
func NewQueryDomainChildrenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-children [domain-id]",
		Short: "Query child domains of a parent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			includeDescendants, _ := cmd.Flags().GetBool("include-descendants")

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DomainChildren(cmd.Context(), &types.QueryDomainChildrenRequest{
				DomainId:           args[0],
				IncludeDescendants: includeDescendants,
			})
			if err != nil {
				return fmt.Errorf("failed to query domain children: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Bool("include-descendants", false, "Include all descendants, not just direct children")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryDomainPathCmd returns the command to query the path from root to a domain.
func NewQueryDomainPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-path [domain-id]",
		Short: "Query the path from root to a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DomainPath(cmd.Context(), &types.QueryDomainPathRequest{
				DomainId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query domain path: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryStratumCmd returns the command to query a single stratum.
func NewQueryStratumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stratum [stratum-id]",
		Short: "Query a stratum by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Stratum(cmd.Context(), &types.QueryStratumRequest{
				StratumId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query stratum: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryStrataCmd returns the command to list all strata.
func NewQueryStrataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "strata",
		Short: "List all ontological strata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			minLevel, _ := cmd.Flags().GetUint32("min-level")
			maxLevel, _ := cmd.Flags().GetUint32("max-level")

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Strata(cmd.Context(), &types.QueryStrataRequest{
				MinLevel: minLevel,
				MaxLevel: maxLevel,
			})
			if err != nil {
				return fmt.Errorf("failed to query strata: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint32("min-level", 0, "Minimum stratum level filter")
	cmd.Flags().Uint32("max-level", 0, "Maximum stratum level filter (0 = no limit)")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryStratumRegistryCmd returns the command to query the full stratum registry.
func NewQueryStratumRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stratum-registry",
		Short: "Query the full stratum registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.StratumRegistry(cmd.Context(), &types.QueryStratumRegistryRequest{})
			if err != nil {
				return fmt.Errorf("failed to query stratum registry: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryValidateDependencyCmd returns the command to validate a stratum dependency.
func NewQueryValidateDependencyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate-dependency [from-stratum] [to-stratum]",
		Short: "Check if a stratum dependency is valid",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ValidateDependency(cmd.Context(), &types.QueryValidateDependencyRequest{
				FromStratum: args[0],
				ToStratum:   args[1],
			})
			if err != nil {
				return fmt.Errorf("failed to validate dependency: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryCalculateConfidenceCmd returns the command to calculate effective confidence.
func NewQueryCalculateConfidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calculate-confidence [domain-id] [base-confidence]",
		Short: "Calculate effective confidence for a domain",
		Long: `Calculate the ontologically-adjusted confidence for a domain,
factoring in stratum dependencies and epistemic category modifiers.

Base confidence: 0-1000000 (basis points)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			var baseConf uint32
			_, err = fmt.Sscanf(args[1], "%d", &baseConf)
			if err != nil {
				return fmt.Errorf("invalid base confidence: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.CalculateConfidence(cmd.Context(), &types.QueryCalculateConfidenceRequest{
				DomainId:       args[0],
				BaseConfidence: baseConf,
			})
			if err != nil {
				return fmt.Errorf("failed to calculate confidence: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDomainProposalCmd returns the command to query a domain proposal.
func NewQueryDomainProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-proposal [proposal-id]",
		Short: "Query a domain proposal by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DomainProposal(cmd.Context(), &types.QueryDomainProposalRequest{
				ProposalId: args[0],
			})
			if err != nil {
				return fmt.Errorf("failed to query domain proposal: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDomainProposalsCmd returns the command to list domain proposals.
func NewQueryDomainProposalsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-proposals",
		Short: "List domain proposals with optional status filter",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint64("limit")
			offset, _ := cmd.Flags().GetUint64("offset")

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DomainProposals(cmd.Context(), &types.QueryDomainProposalsRequest{
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return fmt.Errorf("failed to query domain proposals: %w", err)
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint64("limit", 20, "Maximum number of proposals to return")
	cmd.Flags().Uint64("offset", 0, "Number of proposals to skip")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// NewQueryProvisionalChallengeCmd returns the command to query a provisional challenge by ID.
