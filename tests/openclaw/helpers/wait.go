//go:build openclaw

package helpers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	agentsdk "github.com/legible-money/legible/tools/agent-sdk"
)

const pollInterval = agentsdk.BlockTime // ~2521ms

// WaitForBlocks waits until n additional blocks have been produced.
func WaitForBlocks(ctx context.Context, client agentsdk.QueryClient, n int64) error {
	status, err := client.ChainStatus(ctx)
	if err != nil {
		return fmt.Errorf("getting initial height: %w", err)
	}
	target := status.LatestHeight + n
	return WaitForHeight(ctx, client, target)
}

// WaitForHeight waits until the chain reaches a specific block height.
func WaitForHeight(ctx context.Context, client agentsdk.QueryClient, height int64) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for height %d: %w", height, ctx.Err())
		case <-ticker.C:
			status, err := client.ChainStatus(ctx)
			if err != nil {
				continue
			}
			if status.LatestHeight >= height {
				return nil
			}
		}
	}
}

// WaitForRoundPhase polls until a verification round reaches the expected phase.
func WaitForRoundPhase(ctx context.Context, client agentsdk.QueryClient, roundID, phase string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for round %s phase %s: %w", roundID, phase, ctx.Err())
		case <-ticker.C:
			detail, err := client.RoundDetail(ctx, roundID)
			if err != nil {
				continue
			}
			if detail.Phase == phase {
				return nil
			}
		}
	}
}

// WaitForTx waits for a transaction to be included in a block.
// Since we use -y (broadcast immediately), we just wait a few blocks.
func WaitForTx(ctx context.Context, client agentsdk.QueryClient) error {
	return WaitForBlocks(ctx, client, 2)
}

// RESTGet performs a simple HTTP GET and returns the response body.
func RESTGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// WaitForFact polls until a fact appears in the given domain containing
// claimSubstring. Returns nil on success, context error on timeout.
func WaitForFact(ctx context.Context, client agentsdk.QueryClient, domain, claimSubstring string) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for fact containing %q in domain %q: %w",
				claimSubstring, domain, ctx.Err())
		case <-ticker.C:
			facts, err := client.FactsByDomain(ctx, domain)
			if err != nil {
				continue
			}
			for _, f := range facts {
				if strings.Contains(f.ClaimText, claimSubstring) {
					return nil
				}
			}
		}
	}
}
