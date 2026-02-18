package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/staking/types"
)

// NewTxCmd returns the transaction commands for the legible_staking module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Legible staking module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterValidatorCmd(),
		NewUpdateValidatorStakeCmd(),
		NewDelegateCmd(),
		NewUndelegateCmd(),
		NewRedelegateCmd(),
	)

	return txCmd
}

// NewRegisterValidatorCmd creates a CLI command for MsgRegisterValidator.
func NewRegisterValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-validator [pubkey-hex] [self-delegation]",
		Short: "Register a new validator with PoT consensus",
		Long: `Register a new validator for Proof of Truth consensus.

Validators start at the Apprentice tier and can advance through
Verified, Bonded, and Guardian tiers based on stake and performance.

Minimum self-delegation: 0.111 LGM (111000 ulgm)
The pubkey is a hex-encoded consensus public key.
The DID is passed via the --identity flag in the description.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			commission, _ := cmd.Flags().GetUint32("commission")
			moniker, _ := cmd.Flags().GetString("moniker")
			identity, _ := cmd.Flags().GetString("identity")
			website, _ := cmd.Flags().GetString("website")
			details, _ := cmd.Flags().GetString("details")

			msg := &types.MsgRegisterValidator{
				ValidatorAddress: clientCtx.GetFromAddress().String(),
				PubKey:           []byte(args[0]),
				SelfDelegation:   args[1],
				CommissionRate:   commission,
				Description: &types.ValidatorDescription{
					Moniker:  moniker,
					Identity: identity,
					Website:  website,
					Details:  details,
				},
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint32("commission", 1000, "Commission rate in basis points (1000 = 10%)")
	cmd.Flags().String("moniker", "", "Validator moniker/display name")
	cmd.Flags().String("identity", "", "DID or identity string (did:lgm:{hex})")
	cmd.Flags().String("website", "", "Validator website URL")
	cmd.Flags().String("details", "", "Validator details/description")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateValidatorStakeCmd creates a CLI command for MsgUpdateValidatorStake.
func NewUpdateValidatorStakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-stake [amount]",
		Short: "Increase or decrease validator self-delegation",
		Long: `Update the self-delegation for a validator.

Use --increase to add stake, or --decrease to remove stake.
Tier transitions happen automatically based on stake thresholds.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			isIncrease, _ := cmd.Flags().GetBool("increase")

			msg := &types.MsgUpdateValidatorStake{
				Validator:  clientCtx.GetFromAddress().String(),
				Amount:     args[0],
				IsIncrease: isIncrease,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Bool("increase", true, "Increase stake (false = decrease)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDelegateCmd creates a CLI command for MsgDelegate.
func NewDelegateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate [validator-address] [amount]",
		Short: "Delegate tokens to a validator",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDelegate{
				Delegator: clientCtx.GetFromAddress().String(),
				Validator: args[0],
				Amount:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUndelegateCmd creates a CLI command for MsgUndelegate.
func NewUndelegateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undelegate [validator-address] [amount]",
		Short: "Undelegate tokens from a validator (initiates unbonding)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUndelegate{
				Delegator: clientCtx.GetFromAddress().String(),
				Validator: args[0],
				Amount:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRedelegateCmd creates a CLI command for MsgRedelegate.
func NewRedelegateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redelegate [src-validator] [dst-validator] [amount]",
		Short: "Redelegate tokens from one validator to another",
		Long: `Move delegation from one validator to another.

Subject to a redelegation cooldown of 1,111 blocks (~46 minutes)
to prevent rapid stake-shifting attacks.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRedelegate{
				Delegator:    clientCtx.GetFromAddress().String(),
				SrcValidator: args[0],
				DstValidator: args[1],
				Amount:       args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
