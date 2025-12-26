// Package grpc provides gRPC server and service implementations.
package grpc

import (
	"context"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server.
type Server struct {
	server *grpc.Server
	logger *slog.Logger
}

// ServerConfig contains gRPC server configuration.
type ServerConfig struct {
	Port int
}

// NewServer creates a new gRPC server.
func NewServer(logger *slog.Logger, opts ...grpc.ServerOption) *Server {
	// Add default interceptors
	defaultOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			LoggingInterceptor(logger),
			RecoveryInterceptor(logger),
		),
		grpc.ChainStreamInterceptor(
			StreamLoggingInterceptor(logger),
			StreamRecoveryInterceptor(logger),
		),
	}

	allOpts := append(defaultOpts, opts...)
	server := grpc.NewServer(allOpts...)

	// Enable reflection for grpcurl/grpcui
	reflection.Register(server)

	return &Server{
		server: server,
		logger: logger,
	}
}

// GRPCServer returns the underlying gRPC server for service registration.
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}

// Start starts the gRPC server.
func (s *Server) Start(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.logger.Info("starting gRPC server", "addr", addr)

	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.logger.Error("gRPC server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the gRPC server gracefully.
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.server.GracefulStop()
}

// LoggingInterceptor logs gRPC requests.
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		logger.Debug("gRPC request", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("gRPC error", "method", info.FullMethod, "error", err)
		}
		return resp, err
	}
}

// RecoveryInterceptor recovers from panics.
func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered", "method", info.FullMethod, "panic", r)
				err = grpc.ErrServerStopped
			}
		}()
		return handler(ctx, req)
	}
}

// StreamLoggingInterceptor logs streaming gRPC requests.
func StreamLoggingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		logger.Debug("gRPC stream started", "method", info.FullMethod)
		err := handler(srv, ss)
		if err != nil {
			logger.Error("gRPC stream error", "method", info.FullMethod, "error", err)
		}
		return err
	}
}

// StreamRecoveryInterceptor recovers from panics in streams.
func StreamRecoveryInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC stream panic recovered", "method", info.FullMethod, "panic", r)
				err = grpc.ErrServerStopped
			}
		}()
		return handler(srv, ss)
	}
}
