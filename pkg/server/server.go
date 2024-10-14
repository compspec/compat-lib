package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/compspec/compat-lib/pkg/version"
	pb "github.com/compspec/compat-lib/protos"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	protocol = "tcp"
)

// Server is used to implement your Service.
type Server struct {
	pb.UnimplementedCompatibilityServiceServer
	server   *grpc.Server
	listener net.Listener
	name     string
	version  string
}

// NewServer creates a new "scheduler" server
// The scheduler server registers clusters and then accepts jobs
func NewServer(serverName string) *Server {
	return &Server{name: serverName, version: version.Version}
}

func (s *Server) String() string {
	return fmt.Sprintf("%s v%s", s.name, s.version)
}
func (s *Server) GetName() string {
	return s.name
}

func (s *Server) GetVersion() string {
	return s.version
}

func (s *Server) Stop() {
	log.Printf("stopping server: %s", s.String())
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("error closing listener: %v", err)
		}
	}
	if s.server != nil {
		s.server.Stop()
	}
}

// Start the server
func (s *Server) Start(ctx context.Context, host string) error {

	// Create a listener on the specified address.
	lis, err := net.Listen(protocol, host)
	if err != nil {
		return errors.Wrapf(err, "failed to listen: %s", host)
	}
	return s.serve(ctx, lis)
}

// serve is the main function to ensure the server is listening, etc.
// If we have an additional database to add, ensure it is added
func (s *Server) serve(_ context.Context, lis net.Listener) error {
	if lis == nil {
		return errors.New("listener is required")
	}
	s.listener = lis

	// If we have a certificate, prepare to load and use it
	s.server = grpc.NewServer()

	// This is the main rainbow scheduler service
	pb.RegisterCompatibilityServiceServer(s.server, s)

	log.Printf("server listening: %v", s.listener.Addr())
	if err := s.server.Serve(s.listener); err != nil && err.Error() != "closed" {
		return errors.Wrap(err, "failed to serve")
	}
	return nil
}

// Register a new cluster with the server
func (s *Server) CheckCompatibility(_ context.Context, in *pb.CompatRequest) (*pb.Response, error) {
	if in == nil {
		return nil, errors.New("request is required")
	}

	fmt.Println(in)

	//	log.Printf("📝️ received register: %s", in.Name)
	//	response, err := s.db.RegisterCluster(in.Name, s.globalToken, nodes)
	//	if err != nil {
	//		return response, err
	//	}

	// // If we get here, now we can interact with the graph database to add the nodes
	//
	//	if response.Status == pb.RegisterResponse_REGISTER_SUCCESS {
	//		err = s.graph.AddCluster(in.Name, &nodes, in.Subsystem)
	//	}
	response := &pb.Response{}
	return response, nil
}
