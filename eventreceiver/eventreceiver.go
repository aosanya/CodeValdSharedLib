// Package eventreceiver provides the platform-wide standard for services that
// receive pub/sub events pushed by CodeValdCross via EventReceiverService.NotifyEvent.
package eventreceiver

import "github.com/aosanya/CodeValdSharedLib/types"

// ReceivedEvent is written by a consumer service immediately upon receiving a
// NotifyEvent RPC call. It is a pure log: no status field, no mutation after
// creation.
type ReceivedEvent struct {
	ID         string
	EventID    string
	Topic      string
	AgencyID   string
	Source     string
	Payload    string
	ReceivedAt string // RFC3339 UTC
}

// ReceivedEventTypeDefinition returns the TypeDefinition for the ReceivedEvent
// entity, scoped to the given service prefix (e.g. "ai" → collection "ai_received_events").
// Register the returned definition in the service's Schema so the collection is
// seeded on startup.
func ReceivedEventTypeDefinition(prefix string) types.TypeDefinition {
	return types.TypeDefinition{
		Name:              "ReceivedEvent",
		DisplayName:       "Received Event",
		PathSegment:       "received-events",
		EntityIDParam:     "receivedEventId",
		StorageCollection: prefix + "_received_events",
		Immutable:         true,
		Properties: []types.PropertyDefinition{
			{Name: "event_id", Type: types.PropertyTypeString, Required: true},
			{Name: "topic", Type: types.PropertyTypeString, Required: true},
			{Name: "agency_id", Type: types.PropertyTypeString},
			{Name: "source", Type: types.PropertyTypeString},
			{Name: "payload", Type: types.PropertyTypeString},
			{Name: "received_at", Type: types.PropertyTypeString, Required: true},
		},
	}
}
