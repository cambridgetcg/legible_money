package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/tokens/types"
)

// NewTxCmd returns the transaction commands for the tokens module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tokens module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateTokenCmd(),
		NewMintTokenCmd(),
		NewBurnTokenCmd(),
		NewTransferTokenCmd(),
		NewApproveTokenCmd(),
		NewTransferFromCmd(),
		NewPauseTokenCmd(),
		NewUnpauseTokenCmd(),
		NewAddLiquidityCmd(),
		NewRemoveLiquidityCmd(),
		NewSwapCmd(),
	)

	return txCmd
}

// NewCreateTokenCmd creates a CLI command for MsgCreateToken.
func NewCreateTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-token [name] [symbol] [decimals] [initial-supply]",
		Short: "Create a new token definition",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			decimals, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid decimals: %w", err)
			}

			maxSupply, _ := cmd.Flags().GetString("max-supply")

			msg := &types.MsgCreateToken{
				Creator:       clientCtx.GetFromAddress().String(),
				Name:          args[0],
				Symbol:        args[1],
				Decimals:      uint32(decimals),
				InitialSupply: args[3],
				MaxSupply:     maxSupply,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("max-supply", "0", "Maximum token supply (0 for unlimited)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewMintTokenCmd creates a CLI command for MsgMintToken.
func NewMintTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mint-token [token-id] [to-address] [amount]",
		Short: "Mint new tokens to an address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgMintToken{
				Authority: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
				To:        args[1],
				Amount:    args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewBurnTokenCmd creates a CLI command for MsgBurnToken.
func NewBurnTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "burn-token [token-id] [amount]",
		Short: "Burn tokens from your balance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgBurnToken{
				Burner:  clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				Amount:  args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTransferTokenCmd creates a CLI command for MsgTransferToken.
func NewTransferTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-token [token-id] [to-address] [amount]",
		Short: "Transfer tokens to another address",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgTransferToken{
				Sender:  clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				To:      args[1],
				Amount:  args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewApproveTokenCmd creates a CLI command for MsgApproveToken.
func NewApproveTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve-token [token-id] [spender] [amount]",
		Short: "Approve a spender to transfer tokens on your behalf",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgApproveToken{
				Owner:   clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				Spender: args[1],
				Amount:  args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewTransferFromCmd creates a CLI command for MsgTransferFrom.
func NewTransferFromCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer-from [token-id] [from] [to] [amount]",
		Short: "Transfer tokens using an allowance",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgTransferFrom{
				Spender: clientCtx.GetFromAddress().String(),
				TokenId: args[0],
				From:    args[1],
				To:      args[2],
				Amount:  args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPauseTokenCmd creates a CLI command for MsgPauseToken.
func NewPauseTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-token [token-id]",
		Short: "Pause a token (creator only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgPauseToken{
				Authority: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUnpauseTokenCmd creates a CLI command for MsgUnpauseToken.
func NewUnpauseTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpause-token [token-id]",
		Short: "Unpause a token (creator only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnpauseToken{
				Authority: clientCtx.GetFromAddress().String(),
				TokenId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddLiquidityCmd creates a CLI command for MsgAddLiquidity.
func NewAddLiquidityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity [token-a-id] [token-b-id] [amount-a] [amount-b]",
		Short: "Add liquidity to a token pair pool",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			minLP, _ := cmd.Flags().GetString("min-lp-tokens")

			msg := &types.MsgAddLiquidity{
				Provider:  clientCtx.GetFromAddress().String(),
				TokenAId:  args[0],
				TokenBId:  args[1],
				AmountA:   args[2],
				AmountB:   args[3],
				MinLpTokens: minLP,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("min-lp-tokens", "0", "Minimum LP tokens to receive (slippage protection)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRemoveLiquidityCmd creates a CLI command for MsgRemoveLiquidity.
func NewRemoveLiquidityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-liquidity [pool-id] [lp-tokens]",
		Short: "Remove liquidity from a pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			minA, _ := cmd.Flags().GetString("min-amount-a")
			minB, _ := cmd.Flags().GetString("min-amount-b")

			msg := &types.MsgRemoveLiquidity{
				Provider:   clientCtx.GetFromAddress().String(),
				PoolId:     args[0],
				LpTokens:   args[1],
				MinAmountA: minA,
				MinAmountB: minB,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("min-amount-a", "0", "Minimum token A to receive")
	cmd.Flags().String("min-amount-b", "0", "Minimum token B to receive")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSwapCmd creates a CLI command for MsgSwap.
func NewSwapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap [pool-id] [token-in-id] [amount-in]",
		Short: "Swap tokens through a liquidity pool",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			minOut, _ := cmd.Flags().GetString("min-amount-out")

			msg := &types.MsgSwap{
				Sender:       clientCtx.GetFromAddress().String(),
				PoolId:       args[0],
				TokenInId:    args[1],
				AmountIn:     args[2],
				MinAmountOut: minOut,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("min-amount-out", "0", "Minimum output amount (slippage protection)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
