package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// NewTxCmd returns the transaction commands for the evidence_mgmt module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Evidence management module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitEvidenceCmd(),
		NewAuditEvidenceCmd(),
		NewRegisterOracleCmd(),
		NewSubmitOracleDataCmd(),
	)

	return txCmd
}

// NewSubmitEvidenceCmd creates a CLI command for MsgSubmitEvidence.
func NewSubmitEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-evidence [claim-id] [evidence-type] [content-hash] [description]",
		Short: "Submit evidence for a knowledge claim",
		Long: `Submit evidence supporting a knowledge claim in the evidence management system.

Evidence types: on_chain, oracle_feed, attestation, computation, document, experiment

An optional --source-uri flag can specify where the evidence originated.`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sourceURI, _ := cmd.Flags().GetString("source-uri")

			msg := &types.MsgSubmitEvidence{
				Submitter:    clientCtx.GetFromAddress().String(),
				ClaimId:      args[0],
				EvidenceType: args[1],
				ContentHash:  args[2],
				Description:  args[3],
				SourceUri:    sourceURI,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("source-uri", "", "URI of the evidence source")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewAuditEvidenceCmd creates a CLI command for MsgAuditEvidence.
func NewAuditEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit-evidence [evidence-id] [score] [rationale] [stake]",
		Short: "Audit a piece of submitted evidence",
		Long: `Audit evidence by providing a score (0-1000000), a rationale, and a stake amount.

Score is in basis points where 1000000 = 100% accuracy.
Stake is collateral backing the audit assessment.`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			score, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid score: %w", err)
			}

			msg := &types.MsgAuditEvidence{
				Auditor:    clientCtx.GetFromAddress().String(),
				EvidenceId: args[0],
				Score:      score,
				Rationale:  args[2],
				Stake:      args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewRegisterOracleCmd creates a CLI command for MsgRegisterOracle.
func NewRegisterOracleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-oracle [name] [description] [stake]",
		Short: "Register a new evidence oracle",
		Long: `Register a new oracle for submitting evidence data.

Use --data-types to specify which evidence types the oracle supports (comma-separated).
Valid types: on_chain, oracle_feed, attestation, computation, document, experiment`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			dataTypes, _ := cmd.Flags().GetStringSlice("data-types")
			if len(dataTypes) == 0 {
				return fmt.Errorf("at least one --data-types value is required")
			}

			msg := &types.MsgRegisterOracle{
				Operator:    clientCtx.GetFromAddress().String(),
				Name:        args[0],
				Description: args[1],
				DataTypes:   dataTypes,
				Stake:       args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice("data-types", nil, "Evidence types the oracle supports (comma-separated: on_chain,oracle_feed,attestation,computation,document,experiment)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewSubmitOracleDataCmd creates a CLI command for MsgSubmitOracleData.
func NewSubmitOracleDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-oracle-data [oracle-id] [claim-id] [content-hash] [data-type]",
		Short: "Submit oracle data as evidence for a claim",
		Long: `Submit data from a registered oracle for a specific claim.

Data types: on_chain, oracle_feed, attestation, computation, document, experiment`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgSubmitOracleData{
				Operator:    clientCtx.GetFromAddress().String(),
				OracleId:    args[0],
				ClaimId:     args[1],
				ContentHash: args[2],
				DataType:    args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
