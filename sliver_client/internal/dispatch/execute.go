package dispatch

import (
	"context"
	"fmt"
	"strings"

	"sliver-client/internal/state"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ExecuteCmd returns a DispatchFunc that runs an executable on each target.
func ExecuteCmd(path string, args []string) DispatchFunc {
	return func(ctx context.Context, rpc rpcpb.SliverRPCClient, target state.Target) (string, string, error) {
		async := target.Kind == state.KindBeacon

		req := &sliverpb.ExecuteReq{
			Path:   path,
			Args:   args,
			Output: true,
			Request: &commonpb.Request{
				Async:     async,
				SessionID: target.ID,
			},
		}
		if async {
			req.Request.BeaconID = target.ID
			req.Request.SessionID = ""
		}

		resp, err := rpc.Execute(ctx, req)
		if err != nil {
			return "", "", fmt.Errorf("execute RPC: %w", err)
		}
		if resp.Response != nil && resp.Response.Err != "" {
			return "", "", fmt.Errorf("implant error: %s", resp.Response.Err)
		}

		// For beacons, the response will have a task ID in the response
		if async && resp.Response != nil {
			return "", resp.Response.TaskID, nil
		}

		stdout := cleanOutput(string(resp.Stdout))
		stderr := cleanOutput(string(resp.Stderr))
		var parts []string
		if stdout != "" {
			parts = append(parts, stdout)
		}
		if stderr != "" {
			parts = append(parts, "[stderr] "+stderr)
		}
		output := strings.Join(parts, "\n")
		if output == "" {
			output = fmt.Sprintf("(exit %d, no output)", resp.Status)
		}
		return output, "", nil
	}
}

// cleanOutput strips \r, collapses runs of blank lines, and trims edges.
func cleanOutput(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	lines := strings.Split(s, "\n")
	var out []string
	blanks := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blanks++
			if blanks <= 1 {
				out = append(out, "")
			}
			continue
		}
		blanks = 0
		out = append(out, l)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
