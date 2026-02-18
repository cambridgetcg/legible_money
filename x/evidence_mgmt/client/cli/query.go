package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// NewQueryCmd returns the query commands for the evidence_mgmt module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Evidence management module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryEvidenceCmd(),
		NewQueryEvidenceByClaimIdCmd(),
		NewQueryAuditCmd(),
		NewQueryAuditsForEvidenceCmd(),
		NewQueryOracleCmd(),
		NewQueryActiveOraclesCmd(),
		NewQueryConsensusCmd(),
		NewQueryParamsCmd(),
	)

	return queryCmd
}

// NewQueryEvidenceCmd returns the command to query evidence by ID.
func NewQueryEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evidence [evidence-id]",
		Short: "Query evidence by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryEvidenceRequest{EvidenceId: args[0]}
			res := &types.QueryEvidenceResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/Evidence",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query evidence: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryEvidenceByClaimIdCmd returns the command to query evidence by claim ID.
func NewQueryEvidenceByClaimIdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evidence-by-claim [claim-id]",
		Short: "Query all evidence for a specific claim",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryEvidenceByClaimIdRequest{ClaimId: args[0]}
			res := &types.QueryEvidenceByClaimIdResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/EvidenceByClaimId",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query evidence by claim: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAuditCmd returns the command to query an audit by ID.
func NewQueryAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit [audit-id]",
		Short: "Query an evidence audit by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAuditRequest{AuditId: args[0]}
			res := &types.QueryAuditResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/Audit",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query audit: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryAuditsForEvidenceCmd returns the command to query all audits for a piece of evidence.
func NewQueryAuditsForEvidenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audits-for-evidence [evidence-id]",
		Short: "Query all audits for a specific evidence item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryAuditsForEvidenceRequest{EvidenceId: args[0]}
			res := &types.QueryAuditsForEvidenceResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/AuditsForEvidence",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query audits for evidence: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryOracleCmd returns the command to query an oracle by ID.
func NewQueryOracleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oracle [oracle-id]",
		Short: "Query an oracle by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryOracleRequest{OracleId: args[0]}
			res := &types.QueryOracleResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/Oracle",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query oracle: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveOraclesCmd returns the command to list all active oracles.
func NewQueryActiveOraclesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-oracles",
		Short: "List all active oracles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryActiveOraclesRequest{}
			res := &types.QueryActiveOraclesResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/ActiveOracles",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query active oracles: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryConsensusCmd returns the command to query audit consensus for evidence.
func NewQueryConsensusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consensus [evidence-id]",
		Short: "Query audit consensus for a piece of evidence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryConsensusRequest{EvidenceId: args[0]}
			res := &types.QueryConsensusResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/Consensus",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query consensus: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd returns the command to query evidence_mgmt module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the evidence management module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			res := &types.QueryParamsResponse{}

			err = clientCtx.Invoke(cmd.Context(),
				"/legible.evidence_mgmt.v1.Query/Params",
				req, res,
			)
			if err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			out, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
