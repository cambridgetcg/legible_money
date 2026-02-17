package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/partnerships/types"
)

// NewTxCmd returns the transaction commands for the partnerships module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Partnerships module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewInitiatePartnershipCmd(),
		NewAcceptPartnershipCmd(),
		NewDepositToPotCmd(),
		NewDistributeRewardCmd(),
		NewProposeOperationCmd(),
		NewApproveOperationCmd(),
		NewRejectOperationCmd(),
		NewSafetyFreezeCmd(),
		NewRaiseCoercionSignalCmd(),
		NewInitiateExitCmd(),
	)

	return txCmd
}

// NewInitiatePartnershipCmd creates a CLI command for MsgInitiatePartnership.
func NewInitiatePartnershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate [partner-address] [lock-tier]",
		Short: "Initiate a partnership formation",
		Long: `Initiate a partnership with another address.

Lock tiers (0-5) determine commitment level:
  0 - Trial:     22,222 blocks, 1.00x multiplier, 11% exit penalty
  1 - Standard:  77,777 blocks, 1.11x multiplier, 22% exit penalty
  2 - Trusted:   222,222 blocks, 1.22x multiplier, 33% exit penalty
  3 - Elite:     777,777 blocks, 1.44x multiplier, 44% exit penalty
  4 - Sovereign: 1,111,111 blocks, 1.55x multiplier, 55% exit penalty
  5 - Eternal:   2,222,222 blocks, 1.77x multiplier, 66% exit penalty`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			lockTier, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return fmt.Errorf("invalid lock tier: %w", err)
			}

			msg := &types.MsgInitiatePartnership{
				Initiator: clientCtx.GetFromAddress().String(),
				Partner:   args[0],
				LockTier:  uint32(lockTier),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAcceptPartnershipCmd creates a CLI command for MsgAcceptPartnership.
func NewAcceptPartnershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accept [partnership-id]",
		Short: "Accept a pending partnership formation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAcceptPartnership{
				Acceptor:      clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDepositToPotCmd creates a CLI command for MsgDepositToPot.
func NewDepositToPotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [partnership-id] [amount]",
		Short: "Deposit funds into a partnership common pot",
		Long:  `Deposit ulgm into the partnership's common pot. Must be a participant.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDepositToPot{
				Depositor:     clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
				Amount:        args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDistributeRewardCmd creates a CLI command for MsgDistributeReward.
func NewDistributeRewardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "distribute-reward [partnership-id] [gross-amount] [source-type]",
		Short: "Distribute a reward through a partnership",
		Long: `Distribute a reward through the partnership's split configuration.

Source types: verification, citation, service`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDistributeReward{
				Authority:     clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
				GrossAmount:   args[1],
				SourceType:    args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeOperationCmd creates a CLI command for MsgProposeOperation.
func NewProposeOperationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-operation [partnership-id] [operation-type] [amount]",
		Short: "Propose a consensus operation",
		Long: `Propose a consensus operation requiring partner approval.

Operation types: withdrawal, split_change, tier_upgrade`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			rationale, _ := cmd.Flags().GetString("rationale")

			msg := &types.MsgProposeOperation{
				Proposer:      clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
				OperationType: args[1],
				Amount:        args[2],
				Rationale:     rationale,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("rationale", "", "Rationale for the operation")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewApproveOperationCmd creates a CLI command for MsgApproveOperation.
func NewApproveOperationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve-operation [operation-id]",
		Short: "Approve a pending consensus operation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgApproveOperation{
				Approver:    clientCtx.GetFromAddress().String(),
				OperationId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRejectOperationCmd creates a CLI command for MsgRejectOperation.
func NewRejectOperationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject-operation [operation-id]",
		Short: "Reject a pending operation with optional counter-proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			counterAmount, _ := cmd.Flags().GetString("counter-amount")
			counterRationale, _ := cmd.Flags().GetString("counter-rationale")

			msg := &types.MsgRejectOperation{
				Rejecter:         clientCtx.GetFromAddress().String(),
				OperationId:      args[0],
				CounterAmount:    counterAmount,
				CounterRationale: counterRationale,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("counter-amount", "", "Counter-proposal amount (optional)")
	cmd.Flags().String("counter-rationale", "", "Counter-proposal rationale (optional)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSafetyFreezeCmd creates a CLI command for MsgSafetyFreeze.
func NewSafetyFreezeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "safety-freeze [partnership-id]",
		Short: "Trigger a unilateral safety freeze",
		Long: `Trigger a safety freeze on a partnership. This is a unilateral action
that any participant can take to temporarily suspend operations.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSafetyFreeze{
				Freezer:       clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRaiseCoercionSignalCmd creates a CLI command for MsgRaiseCoercionSignal.
func NewRaiseCoercionSignalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raise-coercion [partnership-id]",
		Short: "Raise a coercion/duress flag",
		Long:  `Raise a coercion signal indicating potential duress within a partnership.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRaiseCoercionSignal{
				Raiser:        clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewInitiateExitCmd creates a CLI command for MsgInitiateExit.
func NewInitiateExitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initiate-exit [partnership-id]",
		Short: "Initiate the exit/dissolution process",
		Long:  `Start the exit process for a partnership. Subject to cooling period and exit penalties.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgInitiateExit{
				Initiator:     clientCtx.GetFromAddress().String(),
				PartnershipId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
