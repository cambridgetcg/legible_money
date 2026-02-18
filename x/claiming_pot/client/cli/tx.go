package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// NewTxCmd returns the transaction commands for the claiming_pot module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Claiming pot module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreatePotCmd(),
		NewFundPotCmd(),
		NewClaimFromPotCmd(),
		NewClosePotCmd(),
	)

	return txCmd
}

// NewCreatePotCmd creates a CLI command for MsgCreatePot.
func NewCreatePotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pot [initial-funding]",
		Short: "Create a new claiming pot with initial funding",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			expiryHeight, _ := cmd.Flags().GetUint64("expiry-height")

			msg := &types.MsgCreatePot{
				Creator:        clientCtx.GetFromAddress().String(),
				InitialFunding: args[0],
				ExpiryHeight:   expiryHeight,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64("expiry-height", 0, "Block height at which the pot expires (0 for no expiry)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFundPotCmd creates a CLI command for MsgFundPot.
func NewFundPotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-pot [pot-id] [amount]",
		Short: "Add funds to an existing claiming pot",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgFundPot{
				Funder: clientCtx.GetFromAddress().String(),
				PotId:  args[0],
				Amount: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimFromPotCmd creates a CLI command for MsgClaimFromPot.
func NewClaimFromPotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-from-pot [pot-id]",
		Short: "Claim your share from a claiming pot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClaimFromPot{
				Claimer: clientCtx.GetFromAddress().String(),
				PotId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClosePotCmd creates a CLI command for MsgClosePot.
func NewClosePotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-pot [pot-id]",
		Short: "Close a claiming pot and refund remaining balance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClosePot{
				Authority: clientCtx.GetFromAddress().String(),
				PotId:     args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
