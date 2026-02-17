package orchestrator

import (
	"context"
	"sync"
	"time"
)

// ChainPoller periodically queries the chain for the latest block height
// and provides a consistent view to the conductor and movements.
type ChainPoller struct {
	pollInterval time.Duration
	queryFn      func(ctx context.Context) (int64, error) // Returns latest block height

	height int64
	mu     sync.RWMutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewChainPoller creates a poller that calls queryFn every pollInterval.
func NewChainPoller(pollInterval time.Duration, queryFn func(ctx context.Context) (int64, error)) *ChainPoller {
	return &ChainPoller{
		pollInterval: pollInterval,
		queryFn:      queryFn,
		done:         make(chan struct{}),
	}
}

// Start begins polling in a background goroutine.
func (p *ChainPoller) Start(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	go func() {
		defer close(p.done)
		ticker := time.NewTicker(p.pollInterval)
		defer ticker.Stop()

		// Initial poll.
		if h, err := p.queryFn(pollCtx); err == nil {
			p.mu.Lock()
			p.height = h
			p.mu.Unlock()
		}

		for {
			select {
			case <-pollCtx.Done():
				return
			case <-ticker.C:
				if h, err := p.queryFn(pollCtx); err == nil {
					p.mu.Lock()
					p.height = h
					p.mu.Unlock()
				}
			}
		}
	}()
}

// Stop halts the polling goroutine and waits for it to finish.
func (p *ChainPoller) Stop() {
	if p.cancel != nil {
		p.cancel()
		<-p.done
	}
}

// Height returns the most recently observed block height.
func (p *ChainPoller) Height() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.height
}

// SetHeight sets the block height directly (used in rehearsal/test mode).
func (p *ChainPoller) SetHeight(h int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.height = h
}

// WaitForBlock blocks until the chain reaches the target height or ctx is cancelled.
func (p *ChainPoller) WaitForBlock(ctx context.Context, target int64) error {
	if p.Height() >= target {
		return nil
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if p.Height() >= target {
				return nil
			}
		}
	}
}

// StaggerDelay sleeps for the configured stagger duration between agent actions
// to avoid thundering-herd tx submission.
func StaggerDelay(ctx context.Context, stagger time.Duration) error {
	if stagger <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(stagger):
		return nil
	}
}

// BlocksToWallTime converts a block count to approximate wall-clock duration
// using the protocol's 2521ms block time.
func BlocksToWallTime(blocks int64) time.Duration {
	return time.Duration(blocks) * 2521 * time.Millisecond
}

// WallTimeToBlocks converts a wall-clock duration to approximate block count.
func WallTimeToBlocks(d time.Duration) int64 {
	return int64(d / (2521 * time.Millisecond))
}
