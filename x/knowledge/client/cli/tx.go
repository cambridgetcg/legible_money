package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/knowledge/types"
)

// NewTxCmd returns the transaction commands for the knowledge module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Knowledge module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitClaimCmd(),
		NewSubmitCommitmentCmd(),
		NewSubmitRevealCmd(),
		NewChallengeFactCmd(),
		NewSubmitContradictionCmd(),
		NewPatronizeFactCmd(),
	)

	return txCmd
}

// NewSubmitClaimCmd creates a CLI command for MsgSubmitClaim.
func NewSubmitClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-claim [claim-text] [domain] [category] [stake-amount]",
		Short: "Submit a knowledge claim for verification",
		Long: `Submit a knowledge claim to the Proof of Truth verification pipeline.

The claim goes through structural evaluation, and if it scores above the
provisional threshold (50%), a provisional fact is created with a challenge
window. If it scores below the reject threshold (30%), it is rejected.

Epistemic categories: analytic, formal, protocol, empirical, historical,
                      predictive, social, replicated, computational

Logic zones: propositional, presburger, peano, empirical`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			claimText := args[0]
			domain := args[1]
			category := args[2]
			stakeAmount := args[3]

			logicZone, _ := cmd.Flags().GetString("logic-zone")
			refs, _ := cmd.Flags().GetStringSlice("references")

			msg := &types.MsgSubmitClaim{
				ClaimText:         claimText,
				Domain:            domain,
				EpistemicCategory: category,
				Submitter:         clientCtx.GetFromAddress().String(),
				StakeAmount:       stakeAmount,
				LogicZone:         logicZone,
				References:        refs,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("logic-zone", "empirical", "Logic zone (propositional, presburger, peano, empirical)")
	cmd.Flags().StringSlice("references", nil, "Referenced fact IDs (comma-separated)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewSubmitCommitmentCmd creates a CLI command for MsgSubmitCommitment.
func NewSubmitCommitmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-commitment [round-id] [commitment-hash]",
		Short: "Submit a verification commitment (commit phase)",
		Long: `Submit a commitment hash for a verification round.

The commitment hash is SHA-256("LGM.commitment.v1:" + verdict + "|" + confidence + "|" + salt).
Use 'compute-commitment' to generate the hash from your verdict and salt.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSubmitCommitment{
				RoundId:        args[0],
				Validator:      clientCtx.GetFromAddress().String(),
				CommitmentHash: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitRevealCmd creates a CLI command for MsgSubmitReveal.
func NewSubmitRevealCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-reveal [round-id] [verdict] [confidence] [salt]",
		Short: "Reveal a verification verdict (reveal phase)",
		Long: `Reveal your verdict for a verification round.

The verdict, confidence, and salt must match the commitment hash submitted
during the commit phase.

Verdicts: accept, reject, abstain
Confidence: 0-1000000 (basis points, where 1000000 = 100%)`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			confidence, err := parseUint64(args[2])
			if err != nil {
				return fmt.Errorf("invalid confidence: %w", err)
			}

			justification, _ := cmd.Flags().GetString("justification")
			vrfProof, _ := cmd.Flags().GetString("vrf-proof")

			msg := &types.MsgSubmitReveal{
				RoundId:       args[0],
				Validator:     clientCtx.GetFromAddress().String(),
				Verdict:       args[1],
				Confidence:    confidence,
				Salt:          args[3],
				Justification: justification,
				VrfProof:      vrfProof,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("justification", "", "Justification for the verdict")
	cmd.Flags().String("vrf-proof", "", "VRF proof for validator selection")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewChallengeFactCmd creates a CLI command for MsgChallengeFact.
func NewChallengeFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-fact [fact-id] [reason] [stake]",
		Short: "Challenge a verified fact",
		Long: `Challenge a verified or provisional fact, triggering a reverification round.

The challenger must stake tokens as collateral. If the challenge fails,
22% of the stake is slashed. If the challenge succeeds, the challenger
receives 30% of the submitter's slashed stake.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgChallengeFact{
				FactId:     args[0],
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

// NewSubmitContradictionCmd creates a CLI command for MsgSubmitContradiction.
func NewSubmitContradictionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-contradiction [target-fact-id] [counter-claim] [stake]",
		Short: "Submit a counter-fact contradicting an existing fact",
		Long: `Submit a counter-fact (negative knowledge) asserting that an existing
fact is false. The counter-claim must explain why the target fact is wrong.

Stake must be at least 50% of the target fact's original stake.
If the contradiction is accepted, the submitter receives a falsification reward.
If rejected, 33% of the stake is slashed.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			logicZone, _ := cmd.Flags().GetString("logic-zone")
			evidenceIDs, _ := cmd.Flags().GetStringSlice("evidence")
			ack, _ := cmd.Flags().GetBool("incompleteness-ack")

			msg := &types.MsgSubmitContradiction{
				TargetFactId:      args[0],
				CounterClaim:      args[1],
				Submitter:         clientCtx.GetFromAddress().String(),
				Stake:             args[2],
				LogicZone:         logicZone,
				EvidenceIds:       evidenceIDs,
				IncompletenessAck: ack,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("logic-zone", "empirical", "Logic zone for the contradiction")
	cmd.Flags().StringSlice("evidence", nil, "Evidence IDs supporting the contradiction")
	cmd.Flags().Bool("incompleteness-ack", false, "Acknowledge Godelian incompleteness for Peano zone claims")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewPatronizeFactCmd creates a CLI command for MsgPatronizeFact.
func NewPatronizeFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patronize-fact [fact-id] [amount]",
		Short: "Stake tokens to protect a fact from pruning",
		Long: `Patronize a fact by staking tokens to protect it from being pruned
due to low usage. This is the knowledge preservation mechanism.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPatronizeFact{
				Patron: clientCtx.GetFromAddress().String(),
				FactId: args[0],
				Amount: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseUint64 parses a string to uint64.
func parseUint64(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
