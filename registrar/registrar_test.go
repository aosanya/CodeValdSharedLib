package registrar_test

import (
	"context"
	"testing"
	"time"

	"github.com/aosanya/CodeValdSharedLib/registrar"
)

// TestNew_ReturnsRegistrar verifies that New succeeds with a valid-looking
// address and returns a non-nil Registrar.
func TestNew_ReturnsRegistrar(t *testing.T) {
	r, err := registrar.New(
		"localhost:50000", "localhost:9001", "agency-1",
		"testservice",
		[]string{"test.event"},
		[]string{},
		nil,
		10*time.Second, 5*time.Second,
	)
	if err != nil {
		t.Fatalf("New: unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("New: expected non-nil Registrar")
	}
	r.Close()
}

// TestNew_CloseSafe verifies that Close is safe to call multiple times.
func TestNew_CloseSafe(t *testing.T) {
	r, err := registrar.New(
		"localhost:50001", ":9002", "",
		"svc",
		nil, nil, nil,
		5*time.Second, 2*time.Second,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// First Close should not panic.
	r.Close()
	// Second Close should also be safe.
	r.Close()
}

// TestRun_ExitsOnContextCancel verifies that Run returns after the context
// is cancelled, within a reasonable deadline.
func TestRun_ExitsOnContextCancel(t *testing.T) {
	r, err := registrar.New(
		"localhost:59999", ":9003", "",
		"testsvc",
		nil, nil, nil,
		100*time.Millisecond, 50*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx)
	}()

	cancel()
	select {
	case <-done:
		// expected — Run returned after ctx was cancelled
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit within 2 s after context cancellation")
	}
}

// TestRun_OneShotWhenIntervalZero verifies that Run returns after the initial
// ping when pingInterval is zero, without waiting for context cancellation.
func TestRun_OneShotWhenIntervalZero(t *testing.T) {
	r, err := registrar.New(
		"localhost:59998", ":9004", "",
		"testsvc2",
		nil, nil, nil,
		0, 50*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close()

	ctx := context.Background()
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Run(ctx)
	}()

	select {
	case <-done:
		// expected — Run returned after single ping with zero interval
	case <-time.After(2 * time.Second):
		t.Fatal("Run with zero interval did not return within 2 s")
	}
}
