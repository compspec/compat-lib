package client

import (
	"context"
	"fmt"
	"log"

	pb "github.com/compspec/compat-lib/protos"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// CompatClient interacts with a compatibility server
type CompatClient struct {
	host       string
	connection *grpc.ClientConn
	service    pb.CompatibilityServiceClient
}

var _ Client = (*CompatClient)(nil)

// Client interface defines functions required for a valid client
type Client interface {
	CheckCompatibility(ctx context.Context, tocheck string) (*pb.Response, error)
}

// NewClient creates a new RainbowClient
func NewClient(host string) (Client, error) {
	if host == "" {
		return nil, errors.New("host is required")
	}

	log.Printf("🧩 starting client (%s)...", host)
	c := CompatClient{host: host}

	// prepare options (note we can add tls here)
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(host, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to connect to %s", host)
	}
	defer conn.Close()

	c.connection = conn
	c.service = pb.NewCompatibilityServiceClient(conn)
	return c, nil
}

// Close closes the created resources (e.g. connection).
func (c *CompatClient) Close() error {
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}

// Connected returns  true if we are connected and the connection is ready
func (c *CompatClient) Connected() bool {
	return c.service != nil && c.connection != nil && c.connection.GetState() == connectivity.Ready
}

// GetHost returns the private hostn name
func (c *CompatClient) GetHost() string {
	return c.host
}

// Check compatibility of an artifact against the known service database
// toCheck can be either a URI (to download from a registry) or the path to
// a json file. Right now I'm assuming the json path.
func (c CompatClient) CheckCompatibility(ctx context.Context, tocheck string) (*pb.Response, error) {
	fmt.Println(tocheck)
	response := &pb.Response{}
	return response, nil
}
