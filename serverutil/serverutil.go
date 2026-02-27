// Package serverutil provides common helpers for starting a gRPC server inside
// a CodeVald microservice. It wires health checking and reflection, handles
// graceful shutdown on context cancellation, and provides small utilities for
// reading duration and string configuration from environment variables.
package serverutil

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// NewGRPCServer creates a *grpc.Server pre-wired with:
//   - gRPC health service (grpc_health_v1) set to SERVING
//   - gRPC server reflection (for grpcurl / dynamic proxy)
//
// Register service-specific handlers on the returned *grpc.Server before
// calling Serve or RunWithGracefulShutdown.
func NewGRPCServer() (*grpc.Server, *health.Server) {
	srv := grpc.NewServer()
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(srv)
	return srv, healthSrv
}

// RunWithGracefulShutdown starts srv on lis in a goroutine, then blocks until
// ctx is cancelled. On cancellation it attempts a graceful drain for up to
// drainTimeout, then forces a stop if the drain has not completed.
// It returns after the server has fully stopped.
func RunWithGracefulShutdown(ctx context.Context, srv *grpc.Server, lis net.Listener, drainTimeout time.Duration) {
	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Printf("serverutil: gRPC server stopped: %v", err)
		}
	}()

	<-ctx.Done()

	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("serverutil: gRPC server stopped cleanly")
	case <-time.After(drainTimeout):
		log.Println("serverutil: drain timeout exceeded — forcing stop")
		srv.Stop()
	}
}

// EnvOrDefault returns os.Getenv(key), falling back to def when the variable
// is unset or empty.
func EnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ParseDurationSeconds reads key from the environment as a positive integer
// number of seconds (e.g. "30" → 30s). Falls back to def when the variable is
// unset, empty, zero, or not a valid positive integer.
func ParseDurationSeconds(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	secs, err := strconv.Atoi(v)
	if err != nil || secs <= 0 {
		log.Printf("serverutil: %s=%q is not a positive integer — using default %s", key, v, def)
		return def
	}
	return time.Duration(secs) * time.Second
}

// ParseDurationString reads key from the environment as a Go duration string
// (e.g. "10s", "1m30s"). Falls back to def when the variable is unset, empty,
// or not parseable by time.ParseDuration.
func ParseDurationString(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		log.Printf("serverutil: %s=%q is not a valid duration — using default %s", key, v, def)
		return def
	}
	return d
}
