// Package arangoutil provides a single-call helper for connecting to ArangoDB.
// Each CodeVald service calls Connect to obtain a driver.Database handle, then
// wraps it in service-specific collection logic. The connection bootstrap is
// shared here; schemas (collections, indexes) remain in each service.
package arangoutil

import (
	"context"
	"fmt"

	driver "github.com/arangodb/go-driver"
	driverhttp "github.com/arangodb/go-driver/http"
)

// Config holds the ArangoDB connection parameters for Connect.
type Config struct {
	// Endpoint is the ArangoDB HTTP endpoint (e.g. "http://localhost:8529").
	Endpoint string

	// Username is the ArangoDB username. Defaults to "root" if empty.
	Username string

	// Password is the ArangoDB password.
	Password string

	// Database is the ArangoDB database name. Created if it does not exist.
	Database string
}

// Connect opens an ArangoDB connection using cfg, authenticates, and returns a
// handle to the named database. The database is created if it does not exist.
// Returns an error if the connection, authentication, or database open fails.
func Connect(ctx context.Context, cfg Config) (driver.Database, error) {
	if cfg.Username == "" {
		cfg.Username = "root"
	}

	conn, err := driverhttp.NewConnection(driverhttp.ConnectionConfig{
		Endpoints: []string{cfg.Endpoint},
	})
	if err != nil {
		return nil, fmt.Errorf("arangoutil.Connect %s: connection: %w", cfg.Database, err)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(cfg.Username, cfg.Password),
	})
	if err != nil {
		return nil, fmt.Errorf("arangoutil.Connect %s: client: %w", cfg.Database, err)
	}

	exists, err := client.DatabaseExists(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("arangoutil.Connect %s: check exists: %w", cfg.Database, err)
	}
	if exists {
		db, err := client.Database(ctx, cfg.Database)
		if err != nil {
			return nil, fmt.Errorf("arangoutil.Connect %s: open: %w", cfg.Database, err)
		}
		return db, nil
	}

	db, err := client.CreateDatabase(ctx, cfg.Database, nil)
	if err != nil {
		return nil, fmt.Errorf("arangoutil.Connect %s: create: %w", cfg.Database, err)
	}
	return db, nil
}
