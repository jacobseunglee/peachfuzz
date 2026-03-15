package dispatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"sliver-client/internal/state"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// UploadCmd returns a DispatchFunc that uploads a local file to each target.
// The file is read once before dispatch; all targets receive the same data.
func UploadCmd(localPath, remotePath string) (DispatchFunc, error) {
	// Validate and read the file once upfront
	cleanPath := filepath.Clean(localPath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read local file %q: %w", cleanPath, err)
	}

	return func(ctx context.Context, rpc rpcpb.SliverRPCClient, target state.Target) (string, string, error) {
		async := target.Kind == state.KindBeacon

		req := &sliverpb.UploadReq{
			Path:    remotePath,
			Data:    data,
			IsIOC:   false,
			Encoder: "",
			Request: &commonpb.Request{
				Async:     async,
				SessionID: target.ID,
			},
		}
		if async {
			req.Request.BeaconID = target.ID
			req.Request.SessionID = ""
		}

		resp, err := rpc.Upload(ctx, req)
		if err != nil {
			return "", "", fmt.Errorf("upload RPC: %w", err)
		}
		if resp.Response != nil && resp.Response.Err != "" {
			return "", "", fmt.Errorf("implant error: %s", resp.Response.Err)
		}

		if async && resp.Response != nil {
			return "", resp.Response.TaskID, nil
		}

		return fmt.Sprintf("uploaded to %s (%d bytes)", resp.Path, len(data)), "", nil
	}, nil
}
