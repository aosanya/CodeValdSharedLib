# CodeValdSharedLib — EventReceiverService & ReceivedEvent

## Overview

Every CodeVald service that consumes pub/sub events must:

1. Implement the shared `EventReceiverService` gRPC interface so Cross can push events to it
2. Write a `ReceivedEvent` record to its own ArangoDB collection on receipt

Both the proto and the Go type + schema helper live here in SharedLib so no
service duplicates the contract.

---

## 1. Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Proto location | SharedLib | Single fully-qualified method path for all consumers; Cross never needs per-service method names |
| `ReceivedEvent` definition | SharedLib | Same Go type and schema helper reused by every consumer service |
| Collection naming | Prefixed per service | `ai_received_events`, `work_received_events`, etc. — clear when debugging across services |
| `status` field | Absent (MVP) | Keep it clean; workflow tracking fields added in a follow-on when processing logic lands |
| Write order | DB write first, then return success | Guarantees the event is logged before Cross marks delivery as delivered |
| On DB failure | Return error to Cross | Delivery stays `pending`; retry logic deferred to post-MVP |

---

## 2. Proto — `EventReceiverService`

**File**: `proto/codevaldshared/v1/eventreceiver.proto`

```protobuf
syntax = "proto3";

package codevaldshared.v1;

option go_package = "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldshared/v1;codevaldsharedv1";

// EventReceiverService is implemented by every CodeVald consumer service.
// CodeValdCross calls NotifyEvent to push a subscribed event to the service.
service EventReceiverService {
  // NotifyEvent delivers a single event to the implementing service.
  // The service MUST write a ReceivedEvent record before returning success.
  // Return an error if the write fails — the delivery stays "pending" in PubSub.
  rpc NotifyEvent(NotifyEventRequest) returns (NotifyEventResponse);
}

// NotifyEventRequest carries the full event pushed by Cross.
message NotifyEventRequest {
  string event_id  = 1;
  string topic     = 2;
  string agency_id = 3;
  string source    = 4;
  string payload   = 5; // JSON-encoded event body
}

// NotifyEventResponse is empty on success.
message NotifyEventResponse {}
```

Generate: `buf generate` in `CodeValdSharedLib/`.

Fully-qualified gRPC method Cross calls on every consumer:

```
/codevaldshared.v1.EventReceiverService/NotifyEvent
```

---

## 3. Go Type — `ReceivedEvent`

**File**: `eventreceiver/eventreceiver.go`

```go
// Package eventreceiver provides the ReceivedEvent domain type and the
// schema helper that any consumer service includes in its schema.go.
package eventreceiver

import "github.com/aosanya/CodeValdSharedLib/types"

// ReceivedEvent is written by a consumer service on successful receipt of a
// pushed event from CodeValdCross. It is a raw log — no status, no processing
// state. Workflow tracking fields will be added in a future iteration.
type ReceivedEvent struct {
    ID         string // UUID v4 assigned by the entitygraph layer
    EventID    string // PubSub event ID
    Topic      string // e.g. "work.task.status.changed"
    AgencyID   string
    Source     string // originating service, e.g. "codevaldwork"
    Payload    string // raw JSON from the publisher
    ReceivedAt string // RFC3339 UTC timestamp
}

// ReceivedEventTypeDefinition returns the TypeDefinition for the ReceivedEvent
// entity. Pass the service-specific prefix (e.g. "ai", "work") to name the
// storage collection correctly: "{prefix}_received_events".
//
// Usage in a service's schema.go:
//
//	import "github.com/aosanya/CodeValdSharedLib/eventreceiver"
//
//	func DefaultMySchema() types.Schema {
//	    return types.Schema{
//	        Types: append(myTypes, eventreceiver.ReceivedEventTypeDefinition("ai")),
//	    }
//	}
func ReceivedEventTypeDefinition(servicePrefix string) types.TypeDefinition {
    return types.TypeDefinition{
        Name:              "ReceivedEvent",
        DisplayName:       "Received Event",
        PathSegment:       "received-events",
        EntityIDParam:     "receivedEventId",
        StorageCollection: servicePrefix + "_received_events",
        Immutable:         true,
        Properties: []types.PropertyDefinition{
            {Name: "event_id",    Type: types.PropertyTypeString, Required: true},
            {Name: "topic",       Type: types.PropertyTypeString, Required: true},
            {Name: "agency_id",   Type: types.PropertyTypeString},
            {Name: "source",      Type: types.PropertyTypeString},
            {Name: "payload",     Type: types.PropertyTypeString},
            {Name: "received_at", Type: types.PropertyTypeString, Required: true},
        },
    }
}
```

---

## 4. How Each Service Uses This

### Step 1 — Add to schema

```go
// schema.go
import "github.com/aosanya/CodeValdSharedLib/eventreceiver"

func DefaultAISchema() types.Schema {
    return types.Schema{
        Types: append(domainTypes(), eventreceiver.ReceivedEventTypeDefinition("ai")),
    }
}
```

### Step 2 — Register gRPC service

```go
// internal/app/app.go
import (
    sharedev1 "github.com/aosanya/CodeValdSharedLib/gen/go/codevaldshared/v1"
)

sharedev1.RegisterEventReceiverServiceServer(grpcServer, server.NewEventReceiver(backend, cfg.AgencyID))
```

### Step 3 — Implement the handler

```go
// internal/server/event_receiver.go
func (s *EventReceiverServer) NotifyEvent(ctx context.Context, req *sharedev1.NotifyEventRequest) (*sharedev1.NotifyEventResponse, error) {
    _, err := s.backend.CreateEntity(ctx, s.agencyID, entitygraph.CreateEntityRequest{
        TypeID: "ReceivedEvent",
        Properties: map[string]any{
            "event_id":    req.GetEventId(),
            "topic":       req.GetTopic(),
            "agency_id":   req.GetAgencyId(),
            "source":      req.GetSource(),
            "payload":     req.GetPayload(),
            "received_at": time.Now().UTC().Format(time.RFC3339),
        },
    })
    if err != nil {
        log.Printf("eventreceiver: NotifyEvent: write ReceivedEvent: %v", err)
        return nil, status.Errorf(codes.Internal, "log received event: %v", err)
    }
    log.Printf("eventreceiver: ACK event_id=%s topic=%s source=%s", req.GetEventId(), req.GetTopic(), req.GetSource())
    return &sharedev1.NotifyEventResponse{}, nil
}
```

### Step 4 — Declare `consumes` in registrar

```go
consumes: []string{"work.task.status.changed"},
```

Cross uses this list to:
- Call `PubSub.Subscribe` on the service's behalf on every heartbeat (PubSub is idempotent on `(subscriber_service, topic_pattern)`)
- Fan out matching events via `EventReceiverService.NotifyEvent`

---

## 5. Definition of Done

- [ ] `proto/codevaldshared/v1/eventreceiver.proto` created
- [ ] `buf generate` run; `gen/go/codevaldshared/v1/` committed
- [ ] `eventreceiver/eventreceiver.go` package created with `ReceivedEvent` type and `ReceivedEventTypeDefinition(prefix)` helper
- [ ] Unit tests for `ReceivedEventTypeDefinition` — correct collection name, all fields present, `Immutable: true`
- [ ] CodeValdAI updated: schema, gRPC registration, handler, consumes declaration
- [ ] All other consumer services follow the same four-step pattern above
