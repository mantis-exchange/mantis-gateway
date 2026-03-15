package grpcclient

import (
	"context"
	"fmt"
	"time"

	pb "github.com/mantis-exchange/mantis-gateway/pkg/proto/mantis"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps the gRPC connection to the matching engine.
type Client struct {
	conn   *grpc.ClientConn
	engine pb.MatchingEngineClient
}

// New creates a new gRPC client connected to the matching engine at the given address.
// It uses a JSON codec as a bridge until proper protobuf-generated code is available.
// Run `make proto` to generate .pb.go files, then remove the ForceCodec option to
// switch to the standard proto codec.
func New(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("grpcclient: failed to connect to matching engine at %s: %w", addr, err)
	}

	return &Client{
		conn:   conn,
		engine: pb.NewMatchingEngineClient(conn),
	}, nil
}

// Close shuts down the gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// SubmitOrder sends a new order to the matching engine.
func (c *Client) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.SubmitOrderResponse, error) {
	return c.engine.SubmitOrder(ctx, req)
}

// CancelOrder requests cancellation of an existing order.
func (c *Client) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	return c.engine.CancelOrder(ctx, req)
}

// GetDepth retrieves the current order book depth from the matching engine.
func (c *Client) GetDepth(ctx context.Context, req *pb.GetDepthRequest) (*pb.GetDepthResponse, error) {
	return c.engine.GetDepth(ctx, req)
}
