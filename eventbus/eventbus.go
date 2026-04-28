// Package eventbus is the CodeValdSharedLib event-publishing contract.
//
// CodeValdAgency, CodeValdComm, CodeValdDT, and CodeValdWork each fire
// lifecycle events (entity created, status changed, edge written, …) at
// CodeValdCross's event-routing layer. Before this package, every consumer
// duplicated the same `CrossPublisher` interface plus the same log-only
// stub. eventbus replaces those copies with a single contract.
//
// The interface carries a typed [Event.Payload] so subscribers receive
// structured data rather than a (topic, agencyID) tuple that forces a
// follow-up RPC. Service-specific payload structs live in the consuming
// service — eventbus is generic infrastructure and never imports a service
// package.
//
// # Cross dependency
//
// The eventual real-world implementation calls a CodeValdCross
// `OrchestratorService.Publish` RPC (planned but not yet shipped). Until
// that RPC lands, services should construct a [LogPublisher] as the
// default. Once the RPC ships, an eventbus-internal adapter will be added
// here without changing the [Publisher] surface — service code stays put.
package eventbus

import (
	"context"
	"log"
	"time"
)

// Event is the unit of publication. Topic, AgencyID, and Timestamp are
// always set; Payload is opaque to the eventbus layer.
type Event struct {
	// Topic is the dotted-namespace event name, e.g. "work.task.created"
	// or "agency.created". Required.
	Topic string

	// AgencyID is the owning agency. Required.
	AgencyID string

	// Timestamp is when the event was constructed (UTC). Producers that
	// leave this zero-valued have it filled in by [Publisher] adapters
	// that care; consumers should treat zero as "unknown when".
	Timestamp time.Time

	// Payload is service-specific event data — typically a small struct
	// the consumer side knows how to type-assert. Carrying a concrete
	// payload avoids the round-trip-to-DB pattern that bare topic events
	// force on subscribers. May be nil for events that need no detail.
	Payload any
}

// Publisher delivers [Event]s to CodeValdCross. Implementations must be safe
// for concurrent use.
//
// A nil Publisher is valid — service code that accepts a Publisher should
// treat nil as "drop the event silently" (events are best-effort by
// design; the persisted operation has already succeeded).
//
// Errors are non-fatal — implementations should log and return nil rather
// than propagate failures up the originating call path.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// PublisherFunc adapts a plain function to the [Publisher] interface —
// useful for tests that record events or for ad-hoc inline implementations.
type PublisherFunc func(ctx context.Context, event Event) error

// Publish invokes f.
func (f PublisherFunc) Publish(ctx context.Context, event Event) error {
	return f(ctx, event)
}

// LogPublisher returns a [Publisher] that writes each event to the standard
// `log` package, prefixed with serviceName. This matches the per-service
// "log-only" stub the migration replaces, and remains the recommended
// default until the CodeValdCross Publish RPC ships.
func LogPublisher(serviceName string) Publisher {
	return PublisherFunc(func(_ context.Context, e Event) error {
		log.Printf("eventbus[%s]: topic=%q agencyID=%q payload=%T",
			serviceName, e.Topic, e.AgencyID, e.Payload)
		return nil
	})
}

// SafePublish is a small helper that calls p.Publish only when p is non-nil
// and stamps Timestamp when the caller left it zero. Lets each service's
// publish hook stay a single line without repeating the nil check.
func SafePublish(ctx context.Context, p Publisher, event Event) {
	if p == nil {
		return
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_ = p.Publish(ctx, event)
}
