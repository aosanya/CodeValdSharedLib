// Package health provides the default HealthService gRPC server implementation
// shared by all CodeVald microservices.
//
// Register it on any gRPC server so CodeValdCross can probe liveness:
//
//	healthSrv := health.New("codevaldwork")
//	pb.RegisterHealthServiceServer(grpcServer, healthSrv)
//
// Services may expose additional operational data via SetMetadata:
//
//	healthSrv.SetMetadata("version", "1.2.3")
//	healthSrv.SetMetadata("queue_depth", "42")
package health

import (
	"context"
	"sync"
	"time"

	pb "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldhealth/v1"
)

// Server implements pb.HealthServiceServer.
// Create with New; register with pb.RegisterHealthServiceServer.
// It is safe for concurrent use.
type Server struct {
	pb.UnimplementedHealthServiceServer
	serviceName string
	startTime   time.Time
	mu          sync.RWMutex
	metadata    map[string]string
}

// New constructs a Server for the given serviceName.
func New(serviceName string) *Server {
	return &Server{
		serviceName: serviceName,
		startTime:   time.Now(),
		metadata:    make(map[string]string),
	}
}

// SetMetadata sets an arbitrary key/value pair that will be returned in every
// CheckResponse. Safe for concurrent use; can be updated at any time.
func (s *Server) SetMetadata(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata[key] = value
}

// Check implements pb.HealthServiceServer.
// Always returns SERVING_STATUS_SERVING — a successful RPC is itself the
// liveness signal that CodeValdCross uses to decide whether the service is
// alive. Override in a subtype if sub-component readiness matters.
func (s *Server) Check(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	s.mu.RLock()
	meta := make(map[string]string, len(s.metadata))
	for k, v := range s.metadata {
		meta[k] = v
	}
	s.mu.RUnlock()

	return &pb.HealthCheckResponse{
		Status:        pb.ServingStatus_SERVING_STATUS_SERVING,
		ServiceName:   s.serviceName,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Metadata:      meta,
	}, nil
}
