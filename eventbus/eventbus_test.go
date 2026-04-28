package eventbus_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aosanya/CodeValdSharedLib/eventbus"
)

func TestPublisherFunc_DelegatesToFunction(t *testing.T) {
	var got eventbus.Event
	p := eventbus.PublisherFunc(func(_ context.Context, e eventbus.Event) error {
		got = e
		return nil
	})
	in := eventbus.Event{Topic: "x", AgencyID: "ag", Payload: 42}
	if err := p.Publish(context.Background(), in); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got.Topic != "x" || got.AgencyID != "ag" || got.Payload != 42 {
		t.Errorf("got %+v, want %+v", got, in)
	}
}

func TestLogPublisher_DoesNotError(t *testing.T) {
	p := eventbus.LogPublisher("test")
	if err := p.Publish(context.Background(), eventbus.Event{Topic: "x", AgencyID: "ag"}); err != nil {
		t.Errorf("LogPublisher returned error: %v", err)
	}
}

func TestSafePublish_NilPublisher_IsNoop(t *testing.T) {
	// Should not panic; nothing else to assert.
	eventbus.SafePublish(context.Background(), nil, eventbus.Event{Topic: "x"})
}

func TestSafePublish_StampsZeroTimestamp(t *testing.T) {
	var got eventbus.Event
	p := eventbus.PublisherFunc(func(_ context.Context, e eventbus.Event) error {
		got = e
		return nil
	})
	before := time.Now().UTC()
	eventbus.SafePublish(context.Background(), p, eventbus.Event{Topic: "x", AgencyID: "ag"})
	after := time.Now().UTC()

	if got.Timestamp.Before(before) || got.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", got.Timestamp, before, after)
	}
}

func TestSafePublish_PreservesNonZeroTimestamp(t *testing.T) {
	var got eventbus.Event
	p := eventbus.PublisherFunc(func(_ context.Context, e eventbus.Event) error {
		got = e
		return nil
	})
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	eventbus.SafePublish(context.Background(), p, eventbus.Event{Topic: "x", AgencyID: "ag", Timestamp: want})
	if !got.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, want)
	}
}

func TestPublisher_ConcurrentSafe(t *testing.T) {
	var (
		mu     sync.Mutex
		events []eventbus.Event
	)
	p := eventbus.PublisherFunc(func(_ context.Context, e eventbus.Event) error {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, e)
		return nil
	})
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			eventbus.SafePublish(context.Background(), p, eventbus.Event{
				Topic:    "concurrent",
				AgencyID: "ag",
				Payload:  i,
			})
		}(i)
	}
	wg.Wait()
	if len(events) != n {
		t.Errorf("got %d events, want %d", len(events), n)
	}
}
