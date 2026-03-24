// Package agencyimport provides an idempotent YAML-to-draft import utility for
// CodeValdAgency.
//
// It reads an agency.yaml file and, in order:
//
//  1. Sets agency details via POST /agency/{agencyId}.
//  2. Creates or reuses an existing open AgencyDraft for the agency.
//  3. Idempotently upserts DraftConfiguredRole, DraftGoal, DraftWorkflow,
//     DraftWorkItem, DraftInstruction, and DraftDeliverable entities.
//
// The draft is left open; this package never calls PromoteDraft.
//
// All POST requests go to the CodeValdCross HTTP proxy, which routes them to
// the CodeValdAgency gRPC service. Entity creation uses EntityService routes
// generated from the agency schema (schemaroutes), so the body format is
//
//	{"properties": { <entity fields> }}
//
// Sub-entity idempotency relies on the UniqueKey defined in each TypeDefinition
// (e.g. ["draft_ref_code", "code"]) — the server calls UpsertEntity, so a second
// import with the same code updates the existing record rather than inserting
// a duplicate.
package agencyimport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the runtime configuration for an Importer.
type Config struct {
	// BaseURL is the CodeValdCross HTTP base URL, e.g. "http://localhost:8080".
	// It must not have a trailing slash.
	BaseURL string
}

// ImportResult contains the identifiers of the resources that were created or
// reused during the import.
type ImportResult struct {
	// AgencyID is the agency identifier (equal to agency.code in the YAML).
	AgencyID string
	// DraftID is the server-assigned UUID of the open AgencyDraft entity.
	DraftID string
}

// Importer reads an agency.yaml file and idempotently populates a
// CodeValdAgency draft via CodeValdCross HTTP routes.
// Construct via [New].
type Importer struct {
	cfg    Config
	client *http.Client
}

// New constructs an Importer backed by the given Config.
// The HTTP client uses a 30-second timeout per request.
func New(cfg Config) *Importer {
	return &Importer{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Import reads the agency YAML at yamlPath and idempotently populates a draft.
//
// It returns an [ImportResult] containing the agencyID and draftID on success.
// On any HTTP or parse failure the operation is aborted and a wrapped error is
// returned — previously created entities are not rolled back.
func (imp *Importer) Import(ctx context.Context, yamlPath string) (*ImportResult, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("Import: read %q: %w", yamlPath, err)
	}

	var spec AgencyYAML
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("Import: unmarshal yaml: %w", err)
	}

	if spec.Agency.Code == "" {
		return nil, fmt.Errorf("Import: agency.code is required in %q", yamlPath)
	}

	agencyID := spec.Agency.Code

	if err := imp.setAgencyDetails(ctx, agencyID, spec.Agency); err != nil {
		return nil, fmt.Errorf("Import %s: set agency details: %w", agencyID, err)
	}

	draftID, err := imp.ensureDraft(ctx, agencyID, spec.Agency)
	if err != nil {
		return nil, fmt.Errorf("Import %s: ensure draft: %w", agencyID, err)
	}

	if err := imp.importConfiguredRoles(ctx, agencyID, draftID, spec.ConfiguredRoles); err != nil {
		return nil, fmt.Errorf("Import %s draft %s: configured roles: %w", agencyID, draftID, err)
	}

	if err := imp.importGoals(ctx, agencyID, draftID, spec.Goals); err != nil {
		return nil, fmt.Errorf("Import %s draft %s: goals: %w", agencyID, draftID, err)
	}

	if err := imp.importWorkflows(ctx, agencyID, draftID, spec.Workflows); err != nil {
		return nil, fmt.Errorf("Import %s draft %s: workflows: %w", agencyID, draftID, err)
	}

	return &ImportResult{AgencyID: agencyID, DraftID: draftID}, nil
}

// ── Step 1: agency details ────────────────────────────────────────────────────

// setAgencyDetails calls POST /agency/{agencyId} to set the root agency fields.
// The Cross proxy auto-wraps the flat JSON body into the SetAgencyDetailsRequest
// json string field, so properties are sent as a flat object.
func (imp *Importer) setAgencyDetails(ctx context.Context, agencyID string, a AgencySpec) error {
	url := imp.url("/agency/", agencyID)
	body := map[string]any{
		"name":    a.Name,
		"mission": strings.TrimSpace(a.Mission),
		"vision":  strings.TrimSpace(a.Vision),
		"code":    a.Code,
	}
	_, err := imp.post(ctx, url, body)
	return err
}

// ── Step 2: draft ─────────────────────────────────────────────────────────────

// ensureDraft returns the entity UUID of an open AgencyDraft for the agency.
// It first lists existing drafts; if an open one with a matching code is found
// it is reused. Otherwise a new draft is created.
func (imp *Importer) ensureDraft(ctx context.Context, agencyID string, a AgencySpec) (string, error) {
	listURL := imp.url("/agency/", agencyID, "/drafts")

	respData, err := imp.get(ctx, listURL)
	if err != nil {
		return "", fmt.Errorf("ensureDraft: list drafts: %w", err)
	}

	var listResult listEntitiesResp
	if jsonErr := json.Unmarshal(respData, &listResult); jsonErr == nil {
		for _, e := range listResult.Entities {
			code, _ := e.Properties["code"].(string)
			status, _ := e.Properties["status"].(string)
			if code == a.Code && strings.EqualFold(status, "open") {
				return e.ID, nil
			}
		}
		// Also accept the first open draft even if code differs (tolerant mode).
		for _, e := range listResult.Entities {
			status, _ := e.Properties["status"].(string)
			if strings.EqualFold(status, "open") {
				return e.ID, nil
			}
		}
	}

	// No open draft found — create one.
	createURL := imp.url("/agency/", agencyID, "/drafts")
	entity, err := imp.postEntity(ctx, createURL, map[string]any{
		"code":        a.Code,
		"description": a.Name,
		"status":      "open",
	})
	if err != nil {
		return "", fmt.Errorf("ensureDraft: create draft: %w", err)
	}
	return entity.ID, nil
}

// ── Step 3: configured roles ──────────────────────────────────────────────────

// importConfiguredRoles upserts each DraftConfiguredRole entity.
func (imp *Importer) importConfiguredRoles(ctx context.Context, agencyID, draftID string, roles []ConfiguredRoleSpec) error {
	base := imp.url("/agency/", agencyID, "/drafts/", draftID, "/configured-roles")
	for _, r := range roles {
		props := map[string]any{
			"draft_ref_code": draftID,
			"code":           r.Code,
			"name":           r.Name,
			"description":    strings.TrimSpace(r.Description),
			"actor_type":     r.ActorType,
			"ordinality":     r.Ordinality,
		}
		if _, err := imp.postEntity(ctx, base, props); err != nil {
			return fmt.Errorf("configured role %s: %w", r.Code, err)
		}
	}
	return nil
}

// ── Step 4: goals ─────────────────────────────────────────────────────────────

// importGoals upserts each DraftGoal entity.
func (imp *Importer) importGoals(ctx context.Context, agencyID, draftID string, goals []GoalSpec) error {
	base := imp.url("/agency/", agencyID, "/drafts/", draftID, "/goals")
	for _, g := range goals {
		props := map[string]any{
			"draft_ref_code": draftID,
			"code":           g.Code,
			"title":          g.Title,
			"description":    strings.TrimSpace(g.Description),
			"ordinality":     g.Ordinality,
		}
		if _, err := imp.postEntity(ctx, base, props); err != nil {
			return fmt.Errorf("goal %s: %w", g.Code, err)
		}
	}
	return nil
}

// ── Step 5: workflows (+ instructions + work items + deliverables) ────────────

// importWorkflows upserts each DraftWorkflow and all of its nested entities.
func (imp *Importer) importWorkflows(ctx context.Context, agencyID, draftID string, workflows []WorkflowSpec) error {
	wfBase := imp.url("/agency/", agencyID, "/drafts/", draftID, "/workflows")

	for _, wf := range workflows {
		wfProps := map[string]any{
			"draft_ref_code": draftID,
			"code":           wf.Code,
			"name":           wf.Name,
			"description":    strings.TrimSpace(wf.Description),
			"ordinality":     wf.Ordinality,
		}
		wfEntity, err := imp.postEntity(ctx, wfBase, wfProps)
		if err != nil {
			return fmt.Errorf("workflow %s: %w", wf.Code, err)
		}
		wfID := wfEntity.ID

		if err := imp.importWorkflowInstructions(ctx, agencyID, draftID, wfID, wf.Instructions); err != nil {
			return fmt.Errorf("workflow %s instructions: %w", wf.Code, err)
		}

		if err := imp.importWorkItems(ctx, agencyID, draftID, wfID, wf.WorkItems); err != nil {
			return fmt.Errorf("workflow %s work items: %w", wf.Code, err)
		}
	}
	return nil
}

// importWorkflowInstructions upserts DraftInstruction entities scoped to a
// workflow (draft_workflow_ref_code is set; draft_work_item_ref_code is omitted).
func (imp *Importer) importWorkflowInstructions(ctx context.Context, agencyID, draftID, wfID string, instructions []InstructionSpec) error {
	base := imp.url("/agency/", agencyID, "/drafts/", draftID, "/instructions")
	for _, inst := range instructions {
		props := map[string]any{
			"draft_ref_code":          draftID,
			"code":                    inst.Code,
			"draft_workflow_ref_code": wfID,
			"content":                 strings.TrimSpace(inst.Content),
			"ordinality":              inst.Ordinality,
		}
		if _, err := imp.postEntity(ctx, base, props); err != nil {
			return fmt.Errorf("instruction %s: %w", inst.Code, err)
		}
	}
	return nil
}

// importWorkItems upserts each DraftWorkItem and its nested instructions and
// deliverables within the given workflow.
func (imp *Importer) importWorkItems(ctx context.Context, agencyID, draftID, wfID string, items []WorkItemSpec) error {
	wiBase := imp.url("/agency/", agencyID, "/drafts/", draftID, "/work-items")

	for _, wi := range items {
		wiProps := map[string]any{
			"draft_ref_code":          draftID,
			"code":                    wi.Code,
			"draft_workflow_ref_code": wfID,
			"title":                   wi.Title,
			"description":             strings.TrimSpace(wi.Description),
			"ordinality":              wi.Ordinality,
			"prompt":                  strings.TrimSpace(wi.Prompt),
		}
		if wi.AssignedRole != "" {
			wiProps["assigned_role"] = wi.AssignedRole
		}
		wiEntity, err := imp.postEntity(ctx, wiBase, wiProps)
		if err != nil {
			return fmt.Errorf("work item %s: %w", wi.Code, err)
		}
		wiID := wiEntity.ID

		if err := imp.importWorkItemInstructions(ctx, agencyID, draftID, wiID, wi.Instructions); err != nil {
			return fmt.Errorf("work item %s instructions: %w", wi.Code, err)
		}

		if err := imp.importDeliverables(ctx, agencyID, draftID, wiID, wi.Deliverables); err != nil {
			return fmt.Errorf("work item %s deliverables: %w", wi.Code, err)
		}
	}
	return nil
}

// importWorkItemInstructions upserts DraftInstruction entities scoped to a
// work item (draft_work_item_ref_code is set; draft_workflow_ref_code is omitted).
func (imp *Importer) importWorkItemInstructions(ctx context.Context, agencyID, draftID, wiID string, instructions []InstructionSpec) error {
	if len(instructions) == 0 {
		return nil
	}
	base := imp.url("/agency/", agencyID, "/drafts/", draftID, "/instructions")
	for _, inst := range instructions {
		props := map[string]any{
			"draft_ref_code":           draftID,
			"code":                     inst.Code,
			"draft_work_item_ref_code": wiID,
			"content":                  strings.TrimSpace(inst.Content),
			"ordinality":               inst.Ordinality,
		}
		if _, err := imp.postEntity(ctx, base, props); err != nil {
			return fmt.Errorf("instruction %s: %w", inst.Code, err)
		}
	}
	return nil
}

// importDeliverables upserts DraftDeliverable entities scoped to a work item.
func (imp *Importer) importDeliverables(ctx context.Context, agencyID, draftID, wiID string, deliverables []DeliverableSpec) error {
	if len(deliverables) == 0 {
		return nil
	}
	base := imp.url("/agency/", agencyID, "/drafts/", draftID, "/deliverables")
	for _, del := range deliverables {
		props := map[string]any{
			"draft_ref_code":           draftID,
			"code":                     del.Code,
			"draft_work_item_ref_code": wiID,
			"title":                    del.Title,
			"description":              strings.TrimSpace(del.Description),
			"ordinality":               del.Ordinality,
			"blocking":                 del.Blocking,
		}
		if _, err := imp.postEntity(ctx, base, props); err != nil {
			return fmt.Errorf("deliverable %s: %w", del.Code, err)
		}
	}
	return nil
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

// postEntity is a convenience wrapper around post that wraps props in the
// EntityService request envelope {"properties": props} and decodes the
// response into an entityItem.
func (imp *Importer) postEntity(ctx context.Context, url string, props map[string]any) (entityItem, error) {
	respData, err := imp.post(ctx, url, map[string]any{"properties": props})
	if err != nil {
		return entityItem{}, err
	}
	var entity entityItem
	if err := json.Unmarshal(respData, &entity); err != nil {
		return entityItem{}, fmt.Errorf("postEntity: decode response: %w", err)
	}
	return entity, nil
}

// post sends a JSON POST request and returns the raw response body.
// Non-2xx responses are decoded as crossError and returned as errors.
func (imp *Importer) post(ctx context.Context, url string, body any) ([]byte, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("post %s: marshal body: %w", url, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("post %s: build request: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	return imp.do(req)
}

// get sends a GET request and returns the raw response body.
// Non-2xx responses are decoded as crossError and returned as errors.
func (imp *Importer) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("get %s: build request: %w", url, err)
	}
	return imp.do(req)
}

// do executes the request and returns the body bytes on success.
// HTTP status codes outside 200–299 are interpreted as errors.
func (imp *Importer) do(req *http.Request) ([]byte, error) {
	resp, err := imp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http %s %s: %w", req.Method, req.URL, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("http %s %s: read body: %w", req.Method, req.URL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var ce crossError
		if jsonErr := json.Unmarshal(body, &ce); jsonErr == nil && ce.Code != "" {
			return nil, fmt.Errorf("http %s %s: %w", req.Method, req.URL, ce)
		}
		return nil, fmt.Errorf("http %s %s: status %d: %s", req.Method, req.URL, resp.StatusCode, body)
	}

	return body, nil
}

// url joins the base URL with the given path segments.
func (imp *Importer) url(segments ...string) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(imp.cfg.BaseURL, "/"))
	for _, s := range segments {
		b.WriteString(s)
	}
	return b.String()
}
