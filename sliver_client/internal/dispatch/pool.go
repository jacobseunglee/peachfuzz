package dispatch

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"sliver-client/internal/state"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// DispatchResult captures the outcome of a single dispatch job.
type DispatchResult struct {
	Target       state.Target
	Output       string
	Error        error
	BeaconTaskID string // non-empty for beacon targets
}

// DispatchFunc is the signature for module dispatch functions.
// It receives the RPC client, target, and returns output + optional beacon task ID.
type DispatchFunc func(ctx context.Context, rpc rpcpb.SliverRPCClient, target state.Target) (output string, beaconTaskID string, err error)

// Pool executes dispatch jobs across targets with bounded concurrency.
type Pool struct {
	MaxWorkers int
	Rpc        rpcpb.SliverRPCClient
}

// Run dispatches fn to all targets with at most p.MaxWorkers concurrent goroutines.
// Returns results for every target (never partial).
func (p *Pool) Run(ctx context.Context, targets []state.Target, fn DispatchFunc) []DispatchResult {
	if len(targets) == 0 {
		return nil
	}

	results := make([]DispatchResult, len(targets))
	sem := make(chan struct{}, p.MaxWorkers)
	var wg sync.WaitGroup

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target state.Target) {
			defer wg.Done()

			// Acquire semaphore slot
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[idx] = DispatchResult{
					Target: target,
					Error:  ctx.Err(),
				}
				return
			}

			output, taskID, err := fn(ctx, p.Rpc, target)
			results[idx] = DispatchResult{
				Target:       target,
				Output:       output,
				BeaconTaskID: taskID,
				Error:        err,
			}
		}(i, t)
	}

	wg.Wait()
	return results
}

// FormatResult returns a human-readable line for a single dispatch result.
func FormatResult(r DispatchResult) string {
	kind := "S"
	if r.Target.Kind == state.KindBeacon {
		kind = "B"
	}
	header := fmt.Sprintf("[%s] %s | %s | %s@%s",
		kind, r.Target.ShortID, r.Target.Address, r.Target.Username, r.Target.Hostname)

	if r.Error != nil {
		return fmt.Sprintf("%s  ERROR: %v", header, r.Error)
	}
	if r.BeaconTaskID != "" {
		return fmt.Sprintf("%s  Task queued: %s", header, r.BeaconTaskID)
	}
	if r.Output != "" {
		// Indent each line of output so it's visually nested under the header
		indented := indentLines(r.Output, "    ")
		return fmt.Sprintf("%s\n%s", header, indented)
	}
	return fmt.Sprintf("%s  OK (no output)", header)
}

func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
