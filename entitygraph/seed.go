package entitygraph

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aosanya/CodeValdSharedLib/types"
)

// SeedSchema seeds a pre-delivered schema idempotently on startup.
// It is a no-op if an active schema version already exists for agencyID.
// On first run it calls SetSchema, Publish, then Activate(1) to make the
// default schema live.
//
// Callers should wrap this in a short timeout context, e.g.:
//
//	seedCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
//	defer cancel()
//	if err := entitygraph.SeedSchema(seedCtx, sm, agencyID, schema); err != nil {
//	    log.Printf("schema seed: %v", err)
//	}
func SeedSchema(ctx context.Context, sm SchemaManager, agencyID string, schema types.Schema) error {
	_, err := sm.GetActive(ctx, agencyID)
	if err == nil {
		return nil // already active — idempotent
	}
	if !errors.Is(err, ErrSchemaNotFound) {
		return fmt.Errorf("SeedSchema %s: check active: %w", agencyID, err)
	}
	schema.AgencyID = agencyID
	if err := sm.SetSchema(ctx, schema); err != nil {
		return fmt.Errorf("SeedSchema %s: set schema: %w", agencyID, err)
	}
	if err := sm.Publish(ctx, agencyID); err != nil {
		return fmt.Errorf("SeedSchema %s: publish: %w", agencyID, err)
	}
	if err := sm.Activate(ctx, agencyID, 1); err != nil {
		return fmt.Errorf("SeedSchema %s: activate: %w", agencyID, err)
	}
	log.Printf("entitygraph: schema seeded for agency %s", agencyID)
	return nil
}
