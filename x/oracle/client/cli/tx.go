// Package cli provides CLI commands for the oracle prediction module.
package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/oracle/types"
)

// NewTxCmd returns the oracle module transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Oracle prediction module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(NewSubmitPredictionCmd())
	return txCmd
}

// NewSubmitPredictionCmd creates submit-prediction CLI command.
func NewSubmitPredictionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-prediction [claim] [domain] [probability-bps] [stake-ulgm] [resolution-block]",
		Short: "Stake a probabilistic prediction about a future state",
		Long: `Submit a prediction to the Zerone oracle module.

Probability in basis points: 0-1,000,000 (0% to 100%).
0 and 1,000,000 are FORBIDDEN — epistemic humility enforced.

Scoring: HWBS = log2(1+horizon_days) * (predicted_prob - actual_outcome)^2
Lower = better calibration.

Minimum stake: 1,000,000 ulgm (1 LGM)
IMPORTANT: stake is plain integer (ulgm), NOT denom-suffixed.

Example:
  legbled tx oracle submit-prediction \
    "Godel fact reaches 850k confidence" logic 750000 1000000 250000 \
    --resolution-type on_chain --from mykey`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			probBps, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("probability must be 0-1000000: %w", err)
			}
			stakeUlgm, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid stake: %w", err)
			}
			resBlock, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid resolution block: %w", err)
			}
			resType, _ := cmd.Flags().GetString("resolution-type")
			resRef, _ := cmd.Flags().GetString("resolution-ref")
			resCriteria, _ := cmd.Flags().GetString("resolution-criteria")

			msg := &types.MsgSubmitPrediction{
				Submitter:          clientCtx.GetFromAddress().String(),
				Claim:              args[0],
				Domain:             args[1],
				Probability:        probBps,
				StakeAmount:        stakeUlgm,
				ResolutionType:     resType,
				ResolutionRef:      resRef,
				ResolutionCriteria: resCriteria,
				ResolutionBlock:    resBlock,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("resolution-type", "arbiter", "on_chain | oracle | arbiter")
	cmd.Flags().String("resolution-ref", "", "Fact ID or oracle feed ID for auto-resolution")
	cmd.Flags().String("resolution-criteria", "", "Human-readable resolution condition")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
