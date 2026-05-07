// Package registrar provides a generic Cross heartbeat registrar that any
// CodeVald service uses to announce itself to CodeValdCross and send periodic
// Register pings. All service-specific metadata (service name, topics, routes)
// are injected via constructor arguments — this package contains no hardcoded
// service identifiers.
package registrar

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"time"

	crossv1 "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldcross/v1"
	"github.com/aosanya/CodeValdSharedLib/types"
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

	// Publish forwards a service lifecycle event to CodeValdCross, which routes
	// it to CodeValdPubSub. Best-effort: errors are logged but do not propagate
	// — the originating operation is already persisted.
	Publish(ctx context.Context, agencyID, topic, source, payload string) error

	// SubscribeTopic asks Cross to register subscriberService as a subscriber
	// to topicPattern in PubSub for the given agency. Called by CodeValdAgency
	// on startup and after import/publish so subscriptions exist regardless of
	// whether the handler service is running.
	SubscribeTopic(ctx context.Context, agencyID, subscriberService, topicPattern string) error
}

// registrar is the unexported concrete implementation of Registrar.
type registrar struct {
	crossAddr    string
	listenAddr   string
	agencyID     string
	serviceName  string
	produces     []string
	producesHash string // SHA-256(sorted produces joined by "\n"), computed once at New()
	consumes     []string
	routes       []*crossv1.RouteDeclaration // converted from []types.RouteInfo at construction
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
//   - routes        — HTTP routes CodeValdCross should proxy to this service;
//     use [github.com/aosanya/CodeValdSharedLib/schemaroutes.RoutesFromSchema]
//     to derive these dynamically from a types.Schema
//   - pingInterval  — heartbeat cadence; if ≤ 0, only the initial ping is sent
//   - pingTimeout   — per-RPC timeout for each Register call
//
// Returns an error if the gRPC client address cannot be parsed.
func New(
	crossAddr, listenAddr, agencyID string,
	serviceName string,
	produces, consumes []string,
	routes []types.RouteInfo,
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
		producesHash: hashTopics(produces),
		consumes:     consumes,
		routes:       routesToProto(routes),
		pingInterval: pingInterval,
		pingTimeout:  pingTimeout,
		conn:         conn,
		client:       crossv1.NewOrchestratorServiceClient(conn),
	}, nil
}

// hashTopics returns SHA-256(sorted topics joined by "\n"), hex-encoded.
// Sorting ensures the hash is stable regardless of AllTopics() return order.
func hashTopics(topics []string) string {
	sorted := make([]string, len(topics))
	copy(sorted, topics)
	sort.Strings(sorted)
	h := sha256.New()
	for _, t := range sorted {
		fmt.Fprintln(h, t)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// routesToProto converts the public []types.RouteInfo slice into the proto
// representation sent in every Register heartbeat. This keeps proto types out
// of the registrar's public API.
func routesToProto(routes []types.RouteInfo) []*crossv1.RouteDeclaration {
	if len(routes) == 0 {
		return nil
	}
	decls := make([]*crossv1.RouteDeclaration, len(routes))
	for i, r := range routes {
		var bindings []*crossv1.PathBinding
		for _, pb := range r.PathBindings {
			bindings = append(bindings, &crossv1.PathBinding{
				UrlParam: pb.URLParam,
				Field:    pb.Field,
			})
		}
		var constants []*crossv1.ConstantBinding
		for _, cb := range r.ConstantBindings {
			constants = append(constants, &crossv1.ConstantBinding{
				Field: cb.Field,
				Value: cb.Value,
			})
		}
		decls[i] = &crossv1.RouteDeclaration{
			Method:           r.Method,
			Pattern:          r.Pattern,
			Capability:       r.Capability,
			GrpcMethod:       r.GrpcMethod,
			PathBindings:     bindings,
			ConstantBindings: constants,
			IsWrite:          r.IsWrite,
		}
	}
	return decls
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

// SubscribeTopic implements [Registrar]. It calls OrchestratorService.SubscribeTopic
// on CodeValdCross, which registers subscriberService as a subscriber to
// topicPattern in PubSub for the given agency.
func (r *registrar) SubscribeTopic(ctx context.Context, agencyID, subscriberService, topicPattern string) error {
	callCtx, cancel := context.WithTimeout(ctx, r.pingTimeout)
	defer cancel()
	_, err := r.client.SubscribeTopic(callCtx, &crossv1.SubscribeTopicRequest{
		AgencyId:          agencyID,
		SubscriberService: subscriberService,
		TopicPattern:      topicPattern,
	})
	return err
}

// Publish implements [Registrar]. It calls OrchestratorService.Publish on
// CodeValdCross, which forwards the event to CodeValdPubSub. Errors are
// returned to the caller; the caller decides whether to log or ignore them.
func (r *registrar) Publish(ctx context.Context, agencyID, topic, source, payload string) error {
	callCtx, cancel := context.WithTimeout(ctx, r.pingTimeout)
	defer cancel()
	_, err := r.client.Publish(callCtx, &crossv1.PublishEventRequest{
		AgencyId: agencyID,
		Topic:    topic,
		Source:   source,
		Payload:  payload,
	})
	return err
}

// ping sends a single Register RPC to CodeValdCross. Errors are logged; the
// caller is not blocked beyond the configured timeout.
func (r *registrar) ping(ctx context.Context) {
	callCtx, cancel := context.WithTimeout(ctx, r.pingTimeout)
	defer cancel()

	_, err := r.client.Register(callCtx, &crossv1.RegisterRequest{
		ServiceName:  r.serviceName,
		Addr:         r.listenAddr,
		AgencyId:     r.agencyID,
		Produces:     r.produces,
		ProducesHash: r.producesHash,
		Consumes:     r.consumes,
		Routes:       r.routes,
	})
	if err != nil {
		log.Printf("registrar[%s]: Register to CodeValdCross %s: %v", r.serviceName, r.crossAddr, err)
		return
	}
	log.Printf("registrar[%s]: registered with CodeValdCross at %s", r.serviceName, r.crossAddr)
}
