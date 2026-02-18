package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// NewQueryCmd returns the query commands for the capture defense module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Capture defense module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryQualificationCmd(),
		NewQueryValidatorQualificationsCmd(),
		NewQueryDomainReputationCmd(),
		NewQueryCaptureRiskCmd(),
	)

	return queryCmd
}

// NewQueryQualificationCmd returns the command to query a validator's qualification for a domain.
func NewQueryQualificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "qualification [validator-addr] [domain]",
		Short: "Query a validator's qualification for a domain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryQualificationRequest{
				ValidatorAddr: args[0],
				Domain:        args[1],
			}
			resp := &types.QueryQualificationResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_defense.v1.Query/Qualification", req, resp); err != nil {
				return fmt.Errorf("failed to query qualification: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryValidatorQualificationsCmd returns the command to list all qualifications for a validator.
func NewQueryValidatorQualificationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-qualifications [validator-addr]",
		Short: "List all domain qualifications for a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryValidatorQualificationsRequest{
				ValidatorAddr: args[0],
			}
			resp := &types.QueryValidatorQualificationsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_defense.v1.Query/ValidatorQualifications", req, resp); err != nil {
				return fmt.Errorf("failed to query validator qualifications: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDomainReputationCmd returns the command to query a validator's reputation in a domain.
func NewQueryDomainReputationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain-reputation [validator-addr] [domain]",
		Short: "Query a validator's effective reputation in a domain",
		Long: `Query the effective reputation for a validator in a domain.

The effective reputation is computed using a layered fallback:
domain-specific reputation (if available), then stratum-level,
then global reputation with appropriate discounts applied.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDomainReputationRequest{
				ValidatorAddr: args[0],
				Domain:        args[1],
			}
			resp := &types.QueryDomainReputationResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_defense.v1.Query/DomainReputation", req, resp); err != nil {
				return fmt.Errorf("failed to query domain reputation: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryCaptureRiskCmd returns the command to query capture risk metrics for a domain.
func NewQueryCaptureRiskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture-risk [domain]",
		Short: "Query capture risk metrics for a domain",
		Long: `Query the Herfindahl-Hirschman Index (HHI) and related capture risk
metrics for a knowledge domain.

Thresholds:
  - Trigger (0.4): Raises a capture warning
  - Pause  (0.6): Automatically pauses domain verification`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryCaptureRiskRequest{
				Domain: args[0],
			}
			resp := &types.QueryCaptureRiskResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/legible.capture_defense.v1.Query/CaptureRisk", req, resp); err != nil {
				return fmt.Errorf("failed to query capture risk: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
