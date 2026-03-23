package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/disputes/types"
)

// NewTxCmd returns the transaction commands for the disputes module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Disputes module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewInitiateDisputeCmd(),
		NewCommitEvidenceCmd(),
		NewRevealEvidenceCmd(),
		NewArbiterVoteCmd(),
		NewEscalateDisputeCmd(),
		NewSettleDisputeCmd(),
		NewTimeoutDisputeCmd(),
	)

	return txCmd
}

// NewInitiateDisputeCmd creates a CLI command for MsgInitiateDispute.
func NewInitiateDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate [target-id] [reason] [stake]",
		Short: "Initiate a dispute against a verified fact",
		Long: `Initiate a dispute against a verified or provisional fact.

The challenger must stake tokens as collateral. If the challenge fails,
a portion of the stake is slashed. If the challenge succeeds, the challenger
receives a reward from the defender's slashed stake.

Escalation tiers:
  Tier 1 (Basic):     3 arbiters,  111,000,000 ulgm (111 LGM) bond, ~2 days
  Tier 2 (Escalated): 7 arbiters,  555,000,000 ulgm (555 LGM) bond, ~4 days
  Tier 3 (Final):     15 arbiters, 1,111,000,000 ulgm (1111 LGM) bond, ~7 days

Arbiters are automatically assigned from active validators.
Dispute status starts at "evidence" (not "open").
Both challenger and defender can submit evidence during the evidence phase.

Example (Tier 1):
  legbled tx disputes initiate <fact-id> "Counter-argument reason" 111000000 --from mykey`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgInitiateDispute{
				TargetId:   args[0],
				Challenger: clientCtx.GetFromAddress().String(),
				Reason:     args[1],
				Stake:      args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCommitEvidenceCmd creates a CLI command for MsgCommitEvidence.
func NewCommitEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-evidence [dispute-id] [commitment-hash]",
		Short: "Commit evidence hash for a dispute (commit phase)",
		Long: `Submit a commitment hash for dispute evidence.

The commitment hash is SHA-256(evidence_content + nonce).
Use 'reveal-evidence' to reveal the actual evidence content.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCommitEvidence{
				DisputeId:      args[0],
				Submitter:      clientCtx.GetFromAddress().String(),
				CommitmentHash: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRevealEvidenceCmd creates a CLI command for MsgRevealEvidence.
func NewRevealEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reveal-evidence [dispute-id] [nonce] [evidence-content...]",
		Short: "Reveal evidence for a dispute (reveal phase)",
		Long: `Reveal evidence for a dispute, completing the commit-reveal process.

The evidence content and nonce must match the previously committed hash.`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Remaining args are evidence items
			var evidence []*types.DisputeEvidence
			for _, content := range args[2:] {
				evidence = append(evidence, &types.DisputeEvidence{Content: content})
			}

			msg := &types.MsgRevealEvidence{
				DisputeId: args[0],
				Submitter: clientCtx.GetFromAddress().String(),
				Nonce:     args[1],
				Evidence:  evidence,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewArbiterVoteCmd creates a CLI command for MsgArbiterVote.
func NewArbiterVoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote [dispute-id] [decision]",
		Short: "Cast an arbiter vote on a dispute",
		Long: `Cast a vote as an arbiter on an active dispute.

Decisions: challenger (overturn), defender (uphold), abstain`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var decision types.ArbiterDecision
			switch strings.ToLower(args[1]) {
			case "challenger":
				decision = types.ArbiterDecision_ARBITER_DECISION_CHALLENGER
			case "defender":
				decision = types.ArbiterDecision_ARBITER_DECISION_DEFENDER
			case "abstain":
				decision = types.ArbiterDecision_ARBITER_DECISION_ABSTAIN
			default:
				return fmt.Errorf("invalid decision: must be 'challenger', 'defender', or 'abstain'")
			}

			reasoning, _ := cmd.Flags().GetString("reasoning")

			msg := &types.MsgArbiterVote{
				DisputeId: args[0],
				Arbiter:   clientCtx.GetFromAddress().String(),
				Vote:      decision,
				Reasoning: reasoning,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("reasoning", "", "Reasoning for the vote")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEscalateDisputeCmd creates a CLI command for MsgEscalateDispute.
func NewEscalateDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "escalate [dispute-id] [additional-stake]",
		Short: "Escalate a dispute to a higher tier",
		Long: `Escalate a dispute to a higher tier, increasing the number of arbiters
and the bond requirements.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEscalateDispute{
				DisputeId:       args[0],
				Requester:       clientCtx.GetFromAddress().String(),
				AdditionalStake: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSettleDisputeCmd creates a CLI command for MsgSettleDispute.
func NewSettleDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settle [dispute-id]",
		Short: "Settle a dispute after the voting deadline",
		Long: `Settle a dispute after the voting deadline has passed.
This tallies votes and resolves the dispute.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSettleDispute{
				DisputeId: args[0],
				Authority: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTimeoutDisputeCmd creates a CLI command for MsgTimeoutDispute.
func NewTimeoutDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeout [dispute-id]",
		Short: "Transition or expire a dispute past its deadline",
		Long: `Trigger a timeout transition for a dispute.

If in evidence phase past deadline: transitions to voting phase.
If in open phase past deadline: expires the dispute.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgTimeoutDispute{
				DisputeId: args[0],
				Authority: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
