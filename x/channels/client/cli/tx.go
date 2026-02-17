package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/channels/types"
)

// NewTxCmd returns the transaction commands for the channels module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Channels module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewOpenChannelCmd(),
		NewUpdateChannelCmd(),
		NewCloseChannelCmd(),
		NewSettleChannelCmd(),
		NewDisputeChannelCmd(),
		NewDepositChannelCmd(),
		NewUpdateStateCmd(),
		NewClaimExpiredCmd(),
	)

	return txCmd
}

// NewOpenChannelCmd returns the command for opening a payment channel.
func NewOpenChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-channel [receiver] [deposit] [timeout-height]",
		Short: "Open a new payment channel",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			timeoutHeight, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid timeout height: %w", err)
			}

			msg := &types.MsgOpenChannel{
				Sender:        clientCtx.GetFromAddress().String(),
				Receiver:      args[0],
				Deposit:       args[1],
				TimeoutHeight: timeoutHeight,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateChannelCmd returns the command for updating a payment channel balance.
func NewUpdateChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-channel [channel-id] [new-balance-sender] [new-balance-receiver] [sequence] [signature-hex]",
		Short: "Submit an on-chain channel balance update",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sequence, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sequence: %w", err)
			}

			signature, err := hex.DecodeString(args[4])
			if err != nil {
				return fmt.Errorf("invalid signature hex: %w", err)
			}

			msg := &types.MsgUpdateChannel{
				Sender:             clientCtx.GetFromAddress().String(),
				ChannelId:          args[0],
				NewBalanceSender:   args[1],
				NewBalanceReceiver: args[2],
				Sequence:           sequence,
				Signature:          signature,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCloseChannelCmd returns the command for closing a payment channel.
func NewCloseChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-channel [channel-id] [final-balance-sender] [final-balance-receiver] [sequence] [counterparty-signature-hex]",
		Short: "Close a payment channel with final balances",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sequence, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sequence: %w", err)
			}

			counterpartySig, err := hex.DecodeString(args[4])
			if err != nil {
				return fmt.Errorf("invalid counterparty signature hex: %w", err)
			}

			msg := &types.MsgCloseChannel{
				Closer:                clientCtx.GetFromAddress().String(),
				ChannelId:             args[0],
				FinalBalanceSender:    args[1],
				FinalBalanceReceiver:  args[2],
				Sequence:              sequence,
				CounterpartySignature: counterpartySig,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSettleChannelCmd returns the command for settling a disputed channel.
func NewSettleChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "settle-channel [channel-id]",
		Short: "Settle a channel after the dispute period expires",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSettleChannel{
				Authority: clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDisputeChannelCmd returns the command for disputing a channel close.
func NewDisputeChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute-channel [channel-id] [claimed-balance-sender] [claimed-balance-receiver] [sequence] [proof-signature-hex]",
		Short: "Dispute a channel close with a newer state",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sequence, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid sequence: %w", err)
			}

			proofSig, err := hex.DecodeString(args[4])
			if err != nil {
				return fmt.Errorf("invalid proof signature hex: %w", err)
			}

			msg := &types.MsgDisputeChannel{
				Disputer:               clientCtx.GetFromAddress().String(),
				ChannelId:              args[0],
				ClaimedBalanceSender:   args[1],
				ClaimedBalanceReceiver: args[2],
				Sequence:               sequence,
				ProofSignature:         proofSig,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDepositChannelCmd returns the command for depositing funds into an existing channel.
func NewDepositChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit-channel [channel-id] [amount]",
		Short: "Deposit additional funds into an existing channel",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDepositChannel{
				Depositor: clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
				Amount:    args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateStateCmd returns the command for submitting an off-chain state update.
func NewUpdateStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-state [channel-id] [nonce] [spent] [state-hash] [payer-signature] [provider-signature]",
		Short: "Submit an off-chain state update for a channel",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			nonce, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid nonce: %w", err)
			}

			msg := &types.MsgUpdateState{
				Sender:    clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
				Update: types.ChannelStateUpdate{
					ChannelId:         args[0],
					Nonce:             nonce,
					Spent:             args[2],
					StateHash:         args[3],
					PayerSignature:    args[4],
					ProviderSignature: args[5],
				},
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewClaimExpiredCmd returns the command for claiming a refund from an expired channel.
func NewClaimExpiredCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-expired [channel-id]",
		Short: "Claim a refund from an expired channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgClaimExpired{
				Claimer:   clientCtx.GetFromAddress().String(),
				ChannelId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
