package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/gov/types"
)

// NewTxCmd returns the transaction commands for the governance module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Legible governance module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		// LIP lifecycle
		NewSubmitLIPCmd(),
		NewStakeLIPCmd(),
		NewAdvanceLIPStageCmd(),
		NewWithdrawLIPCmd(),
		NewAttachUpgradePlanCmd(),
		// Voting
		NewCastVoteCmd(),
		// Disbursement
		NewSubmitDisbursementCmd(),
		NewExecuteDisbursementCmd(),
		NewCancelDisbursementCmd(),
	)

	return txCmd
}

// NewSubmitLIPCmd creates a CLI command for MsgSubmitLIP.
func NewSubmitLIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-lip [title] [description] [initial-stake]",
		Short: "Submit a new Legible Improvement Proposal",
		Long: `Submit a new LIP with an initial stake deposit (anti-spam).

LIP types: parameter_change (1), software_upgrade (2), knowledge_policy (3), economic_policy (4), community (5)

The initial stake is in ulgm (raw amount, no denom suffix).`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			lipTypeStr, _ := cmd.Flags().GetString("lip-type")
			lipType := parseLIPType(lipTypeStr)

			spec, _ := cmd.Flags().GetString("specification")

			msg := &types.MsgSubmitLIP{
				Proposer:      clientCtx.GetFromAddress().String(),
				Title:         args[0],
				Description:   args[1],
				LipType:       lipType,
				InitialStake:  args[2],
				Specification: spec,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("lip-type", "community", "LIP type (parameter_change, software_upgrade, knowledge_policy, economic_policy, community)")
	cmd.Flags().String("specification", "", "Detailed specification text")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewStakeLIPCmd creates a CLI command for MsgStakeLIP.
func NewStakeLIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-lip [lip-id] [amount]",
		Short: "Stake tokens on a LIP to advance it from draft to review",
		Long: `Stake tokens on an existing LIP. When the total staked amount
reaches the required threshold for the LIP's category, the LIP
automatically advances from draft to review stage.

Amount is in ulgm (raw amount, no denom suffix).`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgStakeLIP{
				Staker: clientCtx.GetFromAddress().String(),
				LipId:  args[0],
				Amount: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAdvanceLIPStageCmd creates a CLI command for MsgAdvanceLIPStage.
func NewAdvanceLIPStageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "advance-lip-stage [lip-id]",
		Short: "Advance a LIP to its next stage",
		Long: `Advance a LIP to the next stage in its lifecycle.

Only the LIP proposer can advance stages. Transitions:
  review -> last_call (requires review period elapsed)
  last_call -> voting (requires positive stake)

Voting results are tallied automatically when the voting period expires.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAdvanceLIPStage{
				Authority: clientCtx.GetFromAddress().String(),
				LipId:     args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewWithdrawLIPCmd creates a CLI command for MsgWithdrawLIP.
func NewWithdrawLIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-lip [lip-id]",
		Short: "Withdraw a LIP (proposer only)",
		Long:  `Withdraw a LIP. Only the original proposer can withdraw. Cannot withdraw terminal LIPs (accepted/rejected/withdrawn).`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgWithdrawLIP{
				Proposer: clientCtx.GetFromAddress().String(),
				LipId:    args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAttachUpgradePlanCmd creates a CLI command for MsgAttachUpgradePlan.
func NewAttachUpgradePlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach-upgrade-plan [lip-id] [upgrade-name] [height]",
		Short: "Attach a software upgrade plan to a consensus-category LIP",
		Long: `Attach a software upgrade plan to a consensus-category LIP.

When the LIP is accepted, the upgrade plan is automatically scheduled
via x/upgrade, halting the chain at the specified height for the upgrade.

Only the LIP proposer can attach an upgrade plan. The LIP must be in
the consensus category and not in a terminal stage.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			height, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid height: %w", err)
			}

			info, _ := cmd.Flags().GetString("info")

			msg := &types.MsgAttachUpgradePlan{
				Proposer:    clientCtx.GetFromAddress().String(),
				LipId:       args[0],
				UpgradeName: args[1],
				Height:      height,
				Info:        info,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("info", "", "Upgrade info (e.g. binary download URL)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCastVoteCmd creates a CLI command for MsgCastVote.
func NewCastVoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cast-vote [proposal-id] [option] [stake]",
		Short: "Cast a quadratic vote on a LIP or disbursement",
		Long: `Cast a quadratic vote on a proposal (LIP or disbursement).

Vote options: yes, no, abstain, no_with_veto
Stake is in ulgm. Voting power = sqrt(stake) (quadratic voting).

The stake is escrowed into the governance module account.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			option, err := parseVoteOption(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgCastVote{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Option:     option,
				Stake:      args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitDisbursementCmd creates a CLI command for MsgSubmitDisbursement.
func NewSubmitDisbursementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-disbursement [title] [description] [amount] [recipient]",
		Short: "Submit a treasury disbursement proposal",
		Long: `Submit a proposal to disburse funds from the research treasury.

Amount is in ulgm. Recipient must be a valid lgm address.
The proposal enters voting immediately.`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			justification, _ := cmd.Flags().GetString("justification")

			msg := &types.MsgSubmitDisbursement{
				Submitter:     clientCtx.GetFromAddress().String(),
				Title:         args[0],
				Description:   args[1],
				Amount:        args[2],
				Recipient:     args[3],
				Justification: justification,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("justification", "", "Justification for the disbursement")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewExecuteDisbursementCmd creates a CLI command for MsgExecuteDisbursement.
func NewExecuteDisbursementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-disbursement [disbursement-id]",
		Short: "Execute an approved disbursement after timelock",
		Long: `Execute a disbursement that has been approved and whose
timelock period has expired.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgExecuteDisbursement{
				Authority:      clientCtx.GetFromAddress().String(),
				DisbursementId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCancelDisbursementCmd creates a CLI command for MsgCancelDisbursement.
func NewCancelDisbursementCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-disbursement [disbursement-id]",
		Short: "Cancel a disbursement proposal",
		Long:  `Cancel a non-terminal disbursement. Can be cancelled by the proposer or governance authority.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			reason, _ := cmd.Flags().GetString("reason")

			msg := &types.MsgCancelDisbursement{
				Authority:      clientCtx.GetFromAddress().String(),
				DisbursementId: args[0],
				Reason:         reason,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("reason", "", "Reason for cancellation")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// --- Helpers ---

func parseLIPType(s string) types.LIPType {
	switch strings.ToLower(s) {
	case "parameter_change", "1":
		return types.LIPType_LIP_TYPE_PARAMETER_CHANGE
	case "software_upgrade", "2":
		return types.LIPType_LIP_TYPE_SOFTWARE_UPGRADE
	case "knowledge_policy", "3":
		return types.LIPType_LIP_TYPE_KNOWLEDGE_POLICY
	case "economic_policy", "4":
		return types.LIPType_LIP_TYPE_ECONOMIC_POLICY
	case "community", "5":
		return types.LIPType_LIP_TYPE_COMMUNITY
	default:
		return types.LIPType_LIP_TYPE_COMMUNITY
	}
}

func parseVoteOption(s string) (types.VoteOption, error) {
	switch strings.ToLower(s) {
	case "yes", "1":
		return types.VoteOption_VOTE_OPTION_YES, nil
	case "no", "2":
		return types.VoteOption_VOTE_OPTION_NO, nil
	case "abstain", "3":
		return types.VoteOption_VOTE_OPTION_ABSTAIN, nil
	case "no_with_veto", "4":
		return types.VoteOption_VOTE_OPTION_NO_WITH_VETO, nil
	default:
		return types.VoteOption_VOTE_OPTION_UNSPECIFIED, fmt.Errorf("invalid vote option %q: must be yes, no, abstain, or no_with_veto", s)
	}
}

// Unused import suppressor.
var _ = strconv.Atoi
