package eventreceiver_test

import (
	"testing"

	"github.com/aosanya/CodeValdSharedLib/eventreceiver"
	"github.com/aosanya/CodeValdSharedLib/types"
)

func TestReceivedEventTypeDefinition_CollectionName(t *testing.T) {
	cases := []struct {
		prefix string
		want   string
	}{
		{"ai", "ai_received_events"},
		{"work", "work_received_events"},
		{"comm", "comm_received_events"},
	}
	for _, c := range cases {
		td := eventreceiver.ReceivedEventTypeDefinition(c.prefix)
		if td.StorageCollection != c.want {
			t.Errorf("prefix %q: StorageCollection = %q, want %q", c.prefix, td.StorageCollection, c.want)
		}
	}
}

func TestReceivedEventTypeDefinition_Immutable(t *testing.T) {
	td := eventreceiver.ReceivedEventTypeDefinition("ai")
	if !td.Immutable {
		t.Error("TypeDefinition.Immutable = false, want true")
	}
}

func TestReceivedEventTypeDefinition_PathSegmentAndIDParam(t *testing.T) {
	td := eventreceiver.ReceivedEventTypeDefinition("ai")
	if td.PathSegment != "received-events" {
		t.Errorf("PathSegment = %q, want %q", td.PathSegment, "received-events")
	}
	if td.EntityIDParam != "receivedEventId" {
		t.Errorf("EntityIDParam = %q, want %q", td.EntityIDParam, "receivedEventId")
	}
}

func TestReceivedEventTypeDefinition_RequiredFields(t *testing.T) {
	td := eventreceiver.ReceivedEventTypeDefinition("ai")

	required := map[string]bool{}
	optional := map[string]bool{}
	for _, p := range td.Properties {
		if p.Required {
			required[p.Name] = true
		} else {
			optional[p.Name] = true
		}
	}

	mustBeRequired := []string{"event_id", "topic", "received_at"}
	for _, name := range mustBeRequired {
		if !required[name] {
			t.Errorf("property %q should be Required=true", name)
		}
	}

	mustBeOptional := []string{"agency_id", "source", "payload"}
	for _, name := range mustBeOptional {
		if !optional[name] {
			t.Errorf("property %q should be Required=false", name)
		}
	}
}

func TestReceivedEventTypeDefinition_AllProperties(t *testing.T) {
	td := eventreceiver.ReceivedEventTypeDefinition("ai")

	wantNames := []string{"event_id", "topic", "agency_id", "source", "payload", "received_at"}
	if len(td.Properties) != len(wantNames) {
		t.Fatalf("Properties count = %d, want %d", len(td.Properties), len(wantNames))
	}
	for i, name := range wantNames {
		if td.Properties[i].Name != name {
			t.Errorf("Properties[%d].Name = %q, want %q", i, td.Properties[i].Name, name)
		}
		if td.Properties[i].Type != types.PropertyTypeString {
			t.Errorf("Properties[%d].Type = %q, want PropertyTypeString", i, td.Properties[i].Type)
		}
	}
}
