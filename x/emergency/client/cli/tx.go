package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/emergency/types"
)

// NewTxCmd returns the transaction commands for the emergency module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Emergency module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewProposeHaltCmd(),
		NewVoteHaltCmd(),
		NewProposeRevertCmd(),
		NewVoteRevertCmd(),
		NewProposeResumeCmd(),
		NewVoteResumeCmd(),
	)

	return txCmd
}

// NewProposeHaltCmd creates a CLI command for MsgProposeHalt.
func NewProposeHaltCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-halt [reason]",
		Short: "Propose an emergency chain halt (Guardian only)",
		Long: `Propose an emergency halt of the chain. Only Guardians can propose halts.

The proposal creates a BFT ceremony requiring prevote and precommit phases.
Anti-abuse protections: per-guardian limits, global epoch limits, cooldown blocks.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgProposeHalt{
				Proposer: clientCtx.GetFromAddress().String(),
				Reason:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewVoteHaltCmd creates a CLI command for MsgVoteHalt.
func NewVoteHaltCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-halt [proposal-id] [approve]",
		Short: "Vote on a halt ceremony (Guardian only)",
		Long: `Vote on an active halt ceremony. The vote type (prevote vs precommit)
is determined by the ceremony's current phase.

approve: true or false`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			approve, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid approve value: must be 'true' or 'false'")
			}

			msg := &types.MsgVoteHalt{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Approve:    approve,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeRevertCmd creates a CLI command for MsgProposeRevert.
func NewProposeRevertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-revert [revert-to-height] [justification]",
		Short: "Propose a chain state revert (Guardian only, chain must be halted)",
		Long: `Propose reverting chain state to a specific block height.
The chain must already be halted. Revert depth is limited by MaxRevertDepth parameter.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			height, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid revert height: %w", err)
			}

			msg := &types.MsgProposeRevert{
				Proposer:       clientCtx.GetFromAddress().String(),
				RevertToHeight: height,
				Justification: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewVoteRevertCmd creates a CLI command for MsgVoteRevert.
func NewVoteRevertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-revert [proposal-id] [approve]",
		Short: "Vote on a revert ceremony (Guardian only)",
		Long: `Vote on an active revert ceremony.

approve: true or false`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			approve, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid approve value: must be 'true' or 'false'")
			}

			msg := &types.MsgVoteRevert{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Approve:    approve,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeResumeCmd creates a CLI command for MsgProposeResume.
func NewProposeResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-resume",
		Short: "Propose resuming chain operations (Guardian only, chain must be halted)",
		Long:  `Propose resuming chain operations after an emergency halt.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgProposeResume{
				Proposer: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewVoteResumeCmd creates a CLI command for MsgVoteResume.
func NewVoteResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-resume [proposal-id] [approve]",
		Short: "Vote on a resume ceremony (Guardian only)",
		Long: `Vote on an active resume ceremony.

approve: true or false`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			approve, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid approve value: must be 'true' or 'false'")
			}

			msg := &types.MsgVoteResume{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Approve:    approve,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
