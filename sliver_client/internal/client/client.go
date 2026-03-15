package client

import (
	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

// Connect loads a Sliver operator config and establishes an mTLS gRPC connection.
// Returns the RPC client, the underlying connection (for deferred Close), and any error.
func Connect(configPath string) (rpcpb.SliverRPCClient, *grpc.ClientConn, error) {
	config, err := assets.ReadConfig(configPath)
	if err != nil {
		return nil, nil, err
	}

	rpc, conn, err := transport.MTLSConnect(config)
	if err != nil {
		return nil, nil, err
	}

	return rpc, conn, nil
}
