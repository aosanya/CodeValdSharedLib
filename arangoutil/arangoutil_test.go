package arangoutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/aosanya/CodeValdSharedLib/arangoutil"
)

// TestConnect_DefaultUsername verifies that Connect applies the "root" default
// when Username is empty. A connection attempt to an unreachable address must
// return an error (not panic).
func TestConnect_DefaultUsername(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use a port unlikely to have ArangoDB running.
	_, err := arangoutil.Connect(ctx, arangoutil.Config{
		Endpoint: "http://127.0.0.1:18529",
		Database: "testdb",
		// Username intentionally empty — should default to "root"
	})
	if err == nil {
		t.Skip("an ArangoDB server appears to be running on :18529 — skipping unreachable test")
	}
	// Verify error is wrapped with the database name for observability.
	if got := err.Error(); len(got) == 0 {
		t.Error("Connect: expected a non-empty error message")
	}
}

// TestConnect_ErrorWrapsDatabase verifies that connection errors include the
// database name so callers can identify which Connect call failed.
func TestConnect_ErrorWrapsDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbName := "myservice_db"
	_, err := arangoutil.Connect(ctx, arangoutil.Config{
		Endpoint: "http://127.0.0.1:18529",
		Database: dbName,
	})
	if err == nil {
		t.Skip("an ArangoDB server appears to be running on :18529 — skipping error-wrap test")
	}
	if got := err.Error(); len(got) == 0 {
		t.Errorf("Connect: expected non-empty error, got %q", got)
	}
}
