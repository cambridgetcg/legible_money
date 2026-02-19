package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
)

// NewExploreCmd creates the knowledge explorer query command.
func NewExploreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explore [fact-id]",
		Short: "Explore a fact and its citation tree",
		Long: `Show detailed information about a fact including its references
and which other facts cite it.

Example:
  legbled query knowledge explore MATH-001`,
		Args: cobra.ExactArgs(1),
		RunE: runExplore,
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func runExplore(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	factID := args[0]
	out := cmd.OutOrStdout()
	ctx := cmd.Context()

	// Query the target fact
	queryClient := knowledgetypes.NewQueryClient(clientCtx)
	factResp, err := queryClient.Fact(ctx, &knowledgetypes.QueryFactRequest{FactId: factID})
	if err != nil {
		return fmt.Errorf("failed to query fact %s: %w", factID, err)
	}

	f := factResp.Fact
	if f == nil {
		return fmt.Errorf("fact %s not found", factID)
	}

	// Display the fact
	confPct := float64(f.Confidence) / 10000.0
	fmt.Fprintf(out, "%s -- %s (confidence: %.1f%%)\n", f.Id, f.Status, confPct)
	fmt.Fprintf(out, "  Statement: %s\n", f.ClaimText)
	fmt.Fprintf(out, "  Domain: %s | Category: %s\n", f.Domain, f.EpistemicCategory)
	if f.LogicZone != "" {
		fmt.Fprintf(out, "  Logic Zone: %s\n", f.LogicZone)
	}
	fmt.Fprintf(out, "  Submitter: %s\n", f.Submitter)
	fmt.Fprintf(out, "  Citations: %d | Verifiers: %d\n", f.CitationCount, f.VerifierCount)
	if f.CreatedAtHeight > 0 {
		fmt.Fprintf(out, "  Created: block %d", f.CreatedAtHeight)
		if f.VerifiedAtHeight > 0 {
			fmt.Fprintf(out, " | Verified: block %d", f.VerifiedAtHeight)
		}
		fmt.Fprintln(out)
	}

	// Show references (what this fact depends on)
	if len(f.References) > 0 {
		fmt.Fprintf(out, "  References:\n")
		for _, ref := range f.References {
			refResp, refErr := queryClient.Fact(ctx, &knowledgetypes.QueryFactRequest{FactId: ref})
			if refErr == nil && refResp.Fact != nil {
				rf := refResp.Fact
				fmt.Fprintf(out, "    -> %s: %s (%.1f%%)\n",
					rf.Id, truncate(rf.ClaimText, 60), float64(rf.Confidence)/10000.0)
			} else {
				fmt.Fprintf(out, "    -> %s (not found)\n", ref)
			}
		}
	}

	// Show citing facts (what depends on this fact)
	// Query all facts and find those referencing this one
	factsResp, factsErr := queryClient.Facts(ctx, &knowledgetypes.QueryFactsRequest{Limit: 200})
	if factsErr == nil && len(factsResp.Facts) > 0 {
		var citers []*knowledgetypes.Fact
		for _, cf := range factsResp.Facts {
			for _, ref := range cf.References {
				if ref == factID {
					citers = append(citers, cf)
					break
				}
			}
		}

		if len(citers) > 0 {
			fmt.Fprintf(out, "  Referenced by:\n")
			shown := 0
			for _, cf := range citers {
				if shown >= 10 {
					fmt.Fprintf(out, "    ... %d more\n", len(citers)-shown)
					break
				}
				fmt.Fprintf(out, "    <- %s: %s\n",
					cf.Id, truncate(cf.ClaimText, 60))
				shown++
			}
		}
	}

	return nil
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
