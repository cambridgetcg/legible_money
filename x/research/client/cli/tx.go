package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/research/types"
)

// NewTxCmd returns the transaction commands for the research module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Research module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitResearchCmd(),
		NewChallengeResearchCmd(),
		NewReviewResearchCmd(),
		NewResolveResearchCmd(),
		NewCreateBountyCmd(),
		NewClaimBountyCmd(),
		NewFulfillBountyCmd(),
		NewFundResearchCmd(),
	)

	return txCmd
}

// NewSubmitResearchCmd creates a CLI command for MsgSubmitResearch.
func NewSubmitResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-research [title] [domain] [research-type] [description] [stake]",
		Short: "Submit a research proposal for peer review",
		Long: `Submit a research proposal to the research module for peer review.

Research types: replication, fraud_investigation, methodology_audit, data_validation

The submitter must stake tokens as collateral. If the research is accepted after
peer review, the stake is returned. If rejected, a portion is slashed.`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			targetClaimIDs, _ := cmd.Flags().GetStringSlice("target-claim-ids")

			msg := &types.MsgSubmitResearch{
				Submitter:       clientCtx.GetFromAddress().String(),
				Title:           args[0],
				Domain:          args[1],
				Description:     args[3],
				Stake:           args[4],
				SupportingFacts: targetClaimIDs,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("target-claim-ids", nil, "Target claim IDs to research (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewChallengeResearchCmd creates a CLI command for MsgChallengeResearch.
func NewChallengeResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-research [research-id] [reason] [stake]",
		Short: "Challenge a research submission",
		Long: `Challenge a research submission by providing a reason and staking tokens.

If the challenge succeeds, the challenger receives a portion of the submitter's
slashed stake. If the challenge fails, the challenger's stake is slashed.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgChallengeResearch{
				Challenger: clientCtx.GetFromAddress().String(),
				ResearchId: args[0],
				Reason:     args[1],
				Stake:      args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewReviewResearchCmd creates a CLI command for MsgReviewResearch.
func NewReviewResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review-research [research-id] [score] [verdict] [comments]",
		Short: "Submit a peer review for a research submission",
		Long: `Submit a peer review for a research submission.

Score must be 0-100 representing quality assessment.
Verdicts: approve, reject, revise

The review is recorded on-chain and contributes to the aggregate score
used to determine research acceptance or rejection.`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			score, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid score: %w", err)
			}

			verdict, err := parseReviewVerdict(args[2])
			if err != nil {
				return err
			}

			msg := &types.MsgReviewResearch{
				Reviewer:     clientCtx.GetFromAddress().String(),
				ResearchId:   args[0],
				QualityScore: uint32(score),
				Verdict:      verdict,
				Reasoning:    args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewResolveResearchCmd creates a CLI command for MsgResolveResearch (governance).
func NewResolveResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-research [research-id]",
		Short: "Resolve a research submission (governance only)",
		Long: `Resolve a research submission after the review period has ended.

This is a governance-only transaction. The authority address must be the
module governance account.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResolveResearch{
				Authority:  clientCtx.GetFromAddress().String(),
				ResearchId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCreateBountyCmd creates a CLI command for MsgCreateBounty.
func NewCreateBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-bounty [domain] [title] [description] [reward] [deadline-block]",
		Short: "Create a research bounty",
		Long: `Create a research bounty offering a reward for completing specific research.

The reward is locked in the bounty until it is fulfilled or expires.
The deadline-block is the block height at which the bounty expires if unclaimed.`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			deadlineBlock, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid deadline-block: %w", err)
			}

			msg := &types.MsgCreateBounty{
				Creator:        clientCtx.GetFromAddress().String(),
				Title:          args[1],
				Description:    args[2],
				Reward:         args[3],
				DeadlineHeight: deadlineBlock,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimBountyCmd creates a CLI command for MsgClaimBounty.
func NewClaimBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-bounty [bounty-id]",
		Short: "Claim a research bounty",
		Long: `Claim an open research bounty to begin working on the required research.

Once claimed, the bounty moves to the "claimed" status and must be fulfilled
before the deadline block.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClaimBounty{
				Claimer:  clientCtx.GetFromAddress().String(),
				BountyId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFulfillBountyCmd creates a CLI command for MsgFulfillBounty (governance).
func NewFulfillBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fulfill-bounty [bounty-id] [claimer-id]",
		Short: "Fulfill a research bounty (governance only)",
		Long: `Mark a claimed bounty as fulfilled and release the reward to the claimer.

This is a governance-only transaction. The authority address must be the
module governance account.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFulfillBounty{
				Authority: clientCtx.GetFromAddress().String(),
				BountyId:  args[0],
				Claimer:   args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFundResearchCmd creates a CLI command for MsgFundResearch.
func NewFundResearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-research [amount]",
		Short: "Fund the research treasury",
		Long: `Contribute tokens to the research module treasury.

Funds are used to finance bounty rewards and research incentives.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFundResearch{
				Funder: clientCtx.GetFromAddress().String(),
				Amount: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseReviewVerdict converts a string verdict to the ReviewVerdict enum.
func parseReviewVerdict(s string) (types.ReviewVerdict, error) {
	switch s {
	case "approve":
		return types.ReviewVerdict_REVIEW_VERDICT_APPROVE, nil
	case "reject":
		return types.ReviewVerdict_REVIEW_VERDICT_REJECT, nil
	case "revise":
		return types.ReviewVerdict_REVIEW_VERDICT_REVISE, nil
	default:
		return types.ReviewVerdict_REVIEW_VERDICT_UNSPECIFIED,
			fmt.Errorf("invalid verdict %q: must be approve, reject, or revise", s)
	}
}
