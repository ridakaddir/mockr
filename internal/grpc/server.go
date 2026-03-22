package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/ridakaddir/mockr/internal/config"
	"github.com/ridakaddir/mockr/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ServerOptions holds all configuration for the gRPC server.
type ServerOptions struct {
	// ProtoFiles is the list of .proto file paths to load at startup.
	ProtoFiles []string
	// ImportPaths are extra directories to search when resolving proto imports.
	// When empty, the directory of each ProtoFile is used automatically.
	ImportPaths []string
	// Target is the upstream gRPC address for proxy mode (e.g. "localhost:9090").
	// Empty means mock-only mode.
	Target string
	// Port is the TCP port the gRPC server listens on (default 50051).
	Port int
	// Loader provides the live config and config directory path.
	Loader interface {
		Get() *config.Config
		ConfigDir() string
	}
}

// Server wraps a grpc.Server and its supporting components.
type Server struct {
	opts    ServerOptions
	srv     *grpc.Server
	handler *handler
}

// NewServer initialises the gRPC server: parses proto files, wires the handler,
// and creates the underlying grpc.Server. Call Start to begin accepting connections.
func NewServer(opts ServerOptions) (*Server, error) {
	registry, err := NewRegistry(opts.ProtoFiles, opts.ImportPaths)
	if err != nil {
		return nil, fmt.Errorf("building proto registry: %w", err)
	}

	h := newHandler(opts.Loader, registry, opts.Target)

	srv := grpc.NewServer(
		grpc.UnknownServiceHandler(h.serve),
	)

	// Enable gRPC server reflection so tools like grpcurl work out of the box.
	reflection.Register(srv)

	return &Server{
		opts:    opts,
		srv:     srv,
		handler: h,
	}, nil
}

// Start listens on the configured port and serves gRPC requests until ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.opts.Port))
	if err != nil {
		return fmt.Errorf("gRPC listen on port %d: %w", s.opts.Port, err)
	}

	errCh := make(chan error, 1)
	go func() {
		methods := s.handler.registry.Methods()
		logger.Info("gRPC server running",
			"port", s.opts.Port,
			"proto_methods", len(methods),
			"target", s.opts.Target,
		)
		if err := s.srv.Serve(lis); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		s.srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

// NotifyReload resets transition state after a config hot-reload.
// Called from the config loader's onChange callback.
func (s *Server) NotifyReload() {
	s.handler.resetTransitions()
}
