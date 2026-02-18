package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/qualification/types"
)

// NewTxCmd returns the transaction commands for the qualification module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Qualification module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRequestQualificationCmd(),
		NewEndorseValidatorCmd(),
		NewRenewQualificationCmd(),
		NewWithdrawQualificationCmd(),
	)

	return txCmd
}

// NewRequestQualificationCmd creates a CLI command for MsgRequestQualification.
func NewRequestQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-qualification [domain] [pathway]",
		Short: "Request qualification in a knowledge domain",
		Long: `Request qualification via one of four pathways:

  stake_commitment    - Stake tokens to qualify (requires --stake)
  track_record        - Qualify based on verification history
  cross_reference     - Qualify via endorsements from qualified validators
  stratum_inheritance - Inherit qualification from a parent domain (requires --inherit-from)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			stake, _ := cmd.Flags().GetString("stake")
			inheritFrom, _ := cmd.Flags().GetString("inherit-from")

			msg := &types.MsgRequestQualification{
				ValidatorAddr: clientCtx.GetFromAddress().String(),
				Domain:        args[0],
				Pathway:       types.QualificationPathway(args[1]),
				Stake:         stake,
				InheritFrom:   inheritFrom,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("stake", "", "Stake amount for stake_commitment pathway")
	cmd.Flags().String("inherit-from", "", "Parent domain for stratum_inheritance pathway")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEndorseValidatorCmd creates a CLI command for MsgEndorseValidator.
func NewEndorseValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorse-validator [target-validator] [domain] [stake]",
		Short: "Endorse another validator for a domain qualification",
		Long: `Endorse a target validator for qualification in a domain.

The endorser must already be qualified in the specified domain.
Endorser stake is locked as collateral for the endorsement.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEndorseValidator{
				EndorserAddr:    clientCtx.GetFromAddress().String(),
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

// NewRenewQualificationCmd creates a CLI command for MsgRenewQualification.
func NewRenewQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "renew-qualification [domain]",
		Short: "Renew an expiring qualification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRenewQualification{
				ValidatorAddr: clientCtx.GetFromAddress().String(),
				Domain:        args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewWithdrawQualificationCmd creates a CLI command for MsgWithdrawQualification.
func NewWithdrawQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-qualification [domain]",
		Short: "Voluntarily withdraw qualification and unlock stake",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgWithdrawQualification{
				ValidatorAddr: clientCtx.GetFromAddress().String(),
				Domain:        args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
