package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	authtypes "github.com/legible-money/legible/x/auth/types"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	partnershiptypes "github.com/legible-money/legible/x/partnerships/types"
)

// NewDashboardCmd creates the dashboard query command.
func NewDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard [address]",
		Short: "Show a human-readable account dashboard",
		Long: `Query account, partnership, and knowledge data for an address
and display a unified, human-readable dashboard.

Example:
  legbled query dashboard lgm1abc...def`,
		Args: cobra.ExactArgs(1),
		RunE: runDashboard,
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func runDashboard(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	addr := args[0]
	out := cmd.OutOrStdout()
	ctx := cmd.Context()

	divider := "==========================================="

	// ═══ Account Section ═══
	fmt.Fprintln(out, divider)

	acctResp := &authtypes.QueryAccountResponse{}
	acctErr := clientCtx.Invoke(ctx, "/legible.auth.v1.Query/Account",
		&authtypes.QueryAccountRequest{Address: addr}, acctResp)

	if acctErr == nil && acctResp.Account != nil {
		acct := acctResp.Account
		fmt.Fprintf(out, "  Account: %s\n", addr)
		repStr := ""
		if acct.ReputationScore > 0 {
			repStr = fmt.Sprintf(" | Reputation: %.1f%%", float64(acct.ReputationScore)/10000.0)
		}
		fmt.Fprintf(out, "  Type: %s%s\n", acct.AccountType, repStr)
		fmt.Fprintf(out, "  DID: %s\n", acct.Did)
	} else {
		fmt.Fprintf(out, "  Account: %s\n", addr)
		fmt.Fprintf(out, "  (not registered as Legible account)\n")
	}

	// Query bank balance
	balResp := &banktypes.QueryBalanceResponse{}
	balErr := clientCtx.Invoke(ctx, "/cosmos.bank.v1beta1.Query/Balance",
		&banktypes.QueryBalanceRequest{Address: addr, Denom: "ulgm"}, balResp)
	if balErr == nil && balResp.Balance != nil {
		fmt.Fprintf(out, "  Balance: %s\n", FormatUlgmToLGM(balResp.Balance.Amount.String()))
	}

	fmt.Fprintln(out, divider)

	// ═══ Partnership Section ═══
	partResp := &partnershiptypes.QueryPartnershipsByParticipantResponse{}
	partErr := clientCtx.Invoke(ctx, "/legible.partnerships.v1.Query/PartnershipsByParticipant",
		&partnershiptypes.QueryPartnershipsByParticipantRequest{Participant: addr}, partResp)

	if partErr == nil && len(partResp.Partnerships) > 0 {
		tierNames := []string{"Trial", "Standard", "Trusted", "Elite", "Sovereign", "Eternal"}
		for _, p := range partResp.Partnerships {
			fmt.Fprintf(out, "  Partnership: %s (%s)\n", p.Id, p.Status)
			if p.HumanAddr == addr {
				fmt.Fprintf(out, "    Partner: %s (agent)\n", p.AgentAddr)
			} else {
				fmt.Fprintf(out, "    Partner: %s (human)\n", p.HumanAddr)
			}
			tierName := "unknown"
			if int(p.LockTier) < len(tierNames) {
				tierName = tierNames[p.LockTier]
			}
			fmt.Fprintf(out, "    Lock Tier: %d (%s)\n", p.LockTier, tierName)
			fmt.Fprintf(out, "    Common Pot: %s\n", FormatUlgmToLGM(p.CommonPotBalance))
			if p.CooperationScore > 0 {
				fmt.Fprintf(out, "    Cooperation Score: %.1f%%\n", float64(p.CooperationScore)/100.0)
			}
			if p.SplitHumanBps > 0 || p.SplitAgentBps > 0 {
				fmt.Fprintf(out, "    Split: %.0f/%.0f (human/agent)\n",
					float64(p.SplitHumanBps)/10000.0*100, float64(p.SplitAgentBps)/10000.0*100)
			}
		}
	} else {
		fmt.Fprintln(out, "  No active partnerships")
	}

	fmt.Fprintln(out, divider)

	// ═══ Knowledge Section ═══
	factsResp := &knowledgetypes.QueryFactsResponse{}
	factsErr := clientCtx.Invoke(ctx, "/legible.knowledge.v1.Query/Facts",
		&knowledgetypes.QueryFactsRequest{Limit: 100}, factsResp)

	if factsErr == nil && len(factsResp.Facts) > 0 {
		submitted := 0
		verified := 0
		pending := 0
		domainCounts := make(map[string]int)
		var totalCitations uint64

		for _, f := range factsResp.Facts {
			if f.Submitter == addr {
				submitted++
				if f.Status == "verified" || f.Status == "active" {
					verified++
				} else {
					pending++
				}
				domainCounts[f.Domain]++
				totalCitations += f.CitationCount
			}
		}

		if submitted > 0 {
			fmt.Fprintf(out, "  Knowledge Profile:\n")
			fmt.Fprintf(out, "    Facts Submitted: %d (%d verified, %d pending)\n",
				submitted, verified, pending)
			fmt.Fprintf(out, "    Total Citations: %d\n", totalCitations)

			if len(domainCounts) > 0 {
				// Sort domains by count descending
				type domainEntry struct {
					name  string
					count int
				}
				var entries []domainEntry
				for d, c := range domainCounts {
					entries = append(entries, domainEntry{d, c})
				}
				sort.Slice(entries, func(i, j int) bool {
					if entries[i].count != entries[j].count {
						return entries[i].count > entries[j].count
					}
					return entries[i].name < entries[j].name
				})
				domains := ""
				for _, e := range entries {
					if domains != "" {
						domains += ", "
					}
					domains += fmt.Sprintf("%s (%d)", e.name, e.count)
				}
				fmt.Fprintf(out, "    Domains: %s\n", domains)
			}
		} else {
			fmt.Fprintln(out, "  No knowledge submissions")
		}
	} else {
		fmt.Fprintln(out, "  Knowledge: (query unavailable or no facts)")
	}

	fmt.Fprintln(out, divider)

	return nil
}
