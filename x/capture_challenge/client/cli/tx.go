package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

// NewTxCmd returns the transaction commands for the capture challenge module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capture challenge module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitChallengeCmd(),
		NewAddEvidenceCmd(),
		NewResolveChallengeCmd(),
	)

	return txCmd
}

// NewSubmitChallengeCmd creates a CLI command for MsgSubmitChallenge.
func NewSubmitChallengeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-challenge [domain] [stake]",
		Short: "Submit a capture challenge against validators",
		Long: `Submit a capture challenge against validators suspected of capture behavior.

The challenger must stake tokens as collateral. Provide accused validator
addresses with --accused-validators and evidence as JSON with --evidence-json.

Evidence JSON format: [{"type":"coordinated_voting","description":"...","fact_ids":["f1"],"validator_ids":["v1"],"block_start":100,"block_end":200}]

Evidence types: coordinated_voting, timing_correlation, off_chain_coordination,
                false_fact_accepted, stake_manipulation, qualification_fraud,
                systematic_bias`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			accusedValidators, _ := cmd.Flags().GetStringSlice("accused-validators")
			evidenceJSON, _ := cmd.Flags().GetString("evidence-json")

			var evidence []*types.CaptureEvidence
			if evidenceJSON != "" {
				if err := json.Unmarshal([]byte(evidenceJSON), &evidence); err != nil {
					return fmt.Errorf("invalid evidence JSON: %w", err)
				}
			}

			msg := &types.MsgSubmitChallenge{
				Challenger:        clientCtx.GetFromAddress().String(),
				Domain:            args[0],
				Stake:             args[1],
				AccusedValidators: accusedValidators,
				Evidence:          evidence,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("accused-validators", nil, "Accused validator addresses (comma-separated)")
	cmd.Flags().String("evidence-json", "", "Evidence items as JSON array")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewAddEvidenceCmd creates a CLI command for MsgAddEvidence.
func NewAddEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-evidence [challenge-id] [evidence-json]",
		Short: "Add evidence to an active capture challenge",
		Long: `Add additional evidence to an existing capture challenge during
the evidence submission period. Only the original challenger may add evidence.

Evidence JSON format: {"type":"coordinated_voting","description":"...","fact_ids":["f1"],"validator_ids":["v1"]}`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var evidence types.CaptureEvidence
			if err := json.Unmarshal([]byte(args[1]), &evidence); err != nil {
				return fmt.Errorf("invalid evidence JSON: %w", err)
			}

			msg := &types.MsgAddEvidence{
				Challenger:  clientCtx.GetFromAddress().String(),
				ChallengeId: args[0],
				Evidence:    &evidence,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewResolveChallengeCmd creates a CLI command for MsgResolveChallenge.
func NewResolveChallengeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-challenge [challenge-id] [outcome] [justification]",
		Short: "Resolve a capture challenge (governance)",
		Long: `Resolve a capture challenge after the review period.

Outcomes: upheld, rejected, partial`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgResolveChallenge{
				Resolver:      clientCtx.GetFromAddress().String(),
				ChallengeId:   args[0],
				Outcome:       args[1],
				Justification: args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
