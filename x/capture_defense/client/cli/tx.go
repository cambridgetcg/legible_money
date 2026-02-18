package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// NewTxCmd returns the transaction commands for the capture defense module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capture defense module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRequestQualificationCmd(),
		NewEndorseQualificationCmd(),
	)

	return txCmd
}

// NewRequestQualificationCmd creates a CLI command for MsgRequestQualification.
func NewRequestQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-qualification [domain] [pathway] [stake]",
		Short: "Request qualification for a knowledge domain",
		Long: `Request qualification to verify claims in a knowledge domain.

Pathways: stake_commitment, track_record, cross_reference, stratum_inheritance

Stake requirements vary by domain depth:
  - Leaf domains (depth > 2): 22 LGM
  - Mid-level (depth 2):      222 LGM
  - Top-level (depth 1):      2222 LGM

The stake is locked for 7,777 blocks. If the qualification is granted,
the validator enters a probationary period of 777 blocks.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			endorsers, _ := cmd.Flags().GetStringSlice("endorsers")
			inheritFrom, _ := cmd.Flags().GetString("inherit-from")

			msg := &types.MsgRequestQualification{
				ValidatorAddr: clientCtx.GetFromAddress().String(),
				Domain:        args[0],
				Pathway:       types.QualificationPathway(args[1]),
				Stake:         args[2],
				Endorsers:     endorsers,
				InheritFrom:   inheritFrom,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("endorsers", nil, "Endorser addresses (comma-separated)")
	cmd.Flags().String("inherit-from", "", "Parent domain to inherit qualification from (stratum_inheritance pathway)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewEndorseQualificationCmd creates a CLI command for MsgEndorseQualification.
func NewEndorseQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorse-qualification [target-validator] [domain] [stake]",
		Short: "Endorse another validator's domain qualification",
		Long: `Endorse another validator's qualification for a knowledge domain.

The endorser must already be qualified for the target domain.
A minimum of 3 endorsements is required for endorsement-based qualification.
Endorsements decay after 77,777 blocks.

Self-endorsement is not allowed.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEndorseQualification{
				Endorser:        clientCtx.GetFromAddress().String(),
				TargetValidator: args[0],
				Domain:          args[1],
				Stake:           args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
