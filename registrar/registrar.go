// Package registrar provides a generic Cross heartbeat registrar that any
// CodeVald service uses to announce itself to CodeValdCross and send periodic
// Register pings. All service-specific metadata (service name, topics, routes)
// are injected via constructor arguments — this package contains no hardcoded
// service identifiers.
package registrar

import (
	"context"
	"log"
	"time"

	crossv1 "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldcross/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Registrar announces a service to CodeValdCross and keeps the registration
// alive with periodic heartbeat pings. Create with New; start with Run in a
// goroutine; stop by cancelling the context passed to Run, then call Close.
type Registrar interface {
	// Run sends an immediate Register ping to CodeValdCross, then repeats at
	// the configured interval until ctx is cancelled. Must be called inside a
	// goroutine. Transient errors are logged and do not stop the loop.
	Run(ctx context.Context)

	// Close releases the underlying gRPC connection. Call after the context
	// passed to Run has been cancelled.
	Close()
}

// registrar is the unexported concrete implementation of Registrar.
type registrar struct {
	crossAddr    string
	listenAddr   string
	agencyID     string
	serviceName  string
	produces     []string
	consumes     []string
	routes       []*crossv1.RouteDeclaration
	pingInterval time.Duration
	pingTimeout  time.Duration
	conn         *grpc.ClientConn
	client       crossv1.OrchestratorServiceClient
}

// New constructs a Registrar that heartbeats to the CodeValdCross gRPC address
// at crossAddr. The caller provides all service-specific metadata:
//
//   - crossAddr     — host:port of the CodeValdCross gRPC server
//   - listenAddr    — host:port on which the calling service listens (sent in
//     each heartbeat so CodeValdCross can dial back)
//   - agencyID      — agency this instance serves; empty is valid for unscoped
//     instances
//   - serviceName   — unique identifier for the calling service (e.g. "codevaldgit")
//   - produces      — pub/sub topics this service emits
//   - consumes      — pub/sub topics this service subscribes to
//   - routes        — HTTP routes CodeValdCross should proxy to this service
//   - pingInterval  — heartbeat cadence; if ≤ 0, only the initial ping is sent
//   - pingTimeout   — per-RPC timeout for each Register call
//
// Returns an error if the gRPC client address cannot be parsed.
func New(
	crossAddr, listenAddr, agencyID string,
	serviceName string,
	produces, consumes []string,
	routes []*crossv1.RouteDeclaration,
	pingInterval, pingTimeout time.Duration,
) (Registrar, error) {
	conn, err := grpc.NewClient(crossAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &registrar{
		crossAddr:    crossAddr,
		listenAddr:   listenAddr,
		agencyID:     agencyID,
		serviceName:  serviceName,
		produces:     produces,
		consumes:     consumes,
		routes:       routes,
		pingInterval: pingInterval,
		pingTimeout:  pingTimeout,
		conn:         conn,
		client:       crossv1.NewOrchestratorServiceClient(conn),
	}, nil
}

// Run sends an immediate Register ping, then repeats at the configured interval
// until ctx is cancelled. If pingInterval is ≤ 0 only the initial ping fires.
// All errors are logged; the loop never panics.
func (r *registrar) Run(ctx context.Context) {
	log.Printf("registrar[%s]: starting heartbeat to CodeValdCross at %s (interval=%s timeout=%s)",
		r.serviceName, r.crossAddr, r.pingInterval, r.pingTimeout)
	r.ping(ctx)

	if r.pingInterval <= 0 {
		return
	}

	ticker := time.NewTicker(r.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("registrar[%s]: stopping heartbeat to CodeValdCross", r.serviceName)
			return
		case <-ticker.C:
			r.ping(ctx)
		}
	}
}

// Close releases the underlying gRPC connection. It is safe to call Close
// multiple times; a nil connection is a no-op.
func (r *registrar) Close() {
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.Printf("registrar[%s]: close connection: %v", r.serviceName, err)
		}
	}
}

// ping sends a single Register RPC to CodeValdCross. Errors are logged; the
// caller is not blocked beyond the configured timeout.
func (r *registrar) ping(ctx context.Context) {
	callCtx, cancel := context.WithTimeout(ctx, r.pingTimeout)
	defer cancel()

	_, err := r.client.Register(callCtx, &crossv1.RegisterRequest{
		ServiceName: r.serviceName,
		Addr:        r.listenAddr,
		AgencyId:    r.agencyID,
		Produces:    r.produces,
		Consumes:    r.consumes,
		Routes:      r.routes,
	})
	if err != nil {
		log.Printf("registrar[%s]: Register to CodeValdCross %s: %v", r.serviceName, r.crossAddr, err)
		return
	}
	log.Printf("registrar[%s]: registered with CodeValdCross at %s", r.serviceName, r.crossAddr)
}
