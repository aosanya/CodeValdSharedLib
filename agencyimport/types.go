// Package agencyimport provides an idempotent YAML-to-draft import utility for
// CodeValdAgency. It reads an agency.yaml file and populates a draft via the
// CodeValdCross HTTP proxy routes. The draft is left open; promotion is never
// triggered by this package.
package agencyimport

// ── YAML input types ─────────────────────────────────────────────────────────

// AgencyYAML is the top-level structure of an agency.yaml file.
type AgencyYAML struct {
	Agency          AgencySpec           `yaml:"agency"`
	ConfiguredRoles []ConfiguredRoleSpec `yaml:"configured_roles"`
	Goals           []GoalSpec           `yaml:"goals"`
	Workflows       []WorkflowSpec       `yaml:"workflows"`
}

// AgencySpec holds the root agency fields from the YAML.
type AgencySpec struct {
	// Code is the stable machine-readable identifier, used as the agencyId URL
	// parameter on every request and as the draft UniqueKey value.
	Code    string `yaml:"code"`
	Name    string `yaml:"name"`
	Mission string `yaml:"mission"`
	Vision  string `yaml:"vision"`
}

// ConfiguredRoleSpec describes a single configured role entry.
type ConfiguredRoleSpec struct {
	// Code is the UniqueKey component, e.g. "ROLE-001".
	Code        string `yaml:"code"`
	RefCode     string `yaml:"ref_code"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// ActorType is one of: "human", "ai_agent", "compute_agent".
	ActorType  string `yaml:"actor_type"`
	Ordinality int    `yaml:"ordinality"`
}

// GoalSpec describes a single strategic goal.
type GoalSpec struct {
	// Code is the UniqueKey component, e.g. "GOAL-001".
	Code        string `yaml:"code"`
	RefCode     string `yaml:"ref_code"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Ordinality  int    `yaml:"ordinality"`
}

// WorkflowSpec describes a workflow and its nested instructions and work items.
type WorkflowSpec struct {
	// Code is the UniqueKey component, e.g. "WF-001".
	Code         string            `yaml:"code"`
	RefCode      string            `yaml:"ref_code"`
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	Ordinality   int               `yaml:"ordinality"`
	Instructions []InstructionSpec `yaml:"instructions"`
	WorkItems    []WorkItemSpec    `yaml:"work_items"`
}

// InstructionSpec describes a single instruction attached to a workflow or
// work item.
type InstructionSpec struct {
	// Code is the UniqueKey component, e.g. "INST-WF001-001".
	Code       string `yaml:"code"`
	RefCode    string `yaml:"ref_code"`
	Content    string `yaml:"content"`
	Ordinality int    `yaml:"ordinality"`
}

// WorkItemSpec describes a unit of work within a workflow.
type WorkItemSpec struct {
	// Code is the UniqueKey component, e.g. "WI-001".
	Code        string `yaml:"code"`
	RefCode     string `yaml:"ref_code"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	// Ordinality controls sequencing; equal ordinality values run in parallel.
	Ordinality int `yaml:"ordinality"`
	// AssignedRole is the code of the ConfiguredRole that executes this item,
	// e.g. "ROLE-001". Stored as the assigned_role property on DraftWorkItem.
	AssignedRole string            `yaml:"assigned_role"`
	Prompt       string            `yaml:"prompt"`
	Instructions []InstructionSpec `yaml:"instructions"`
	Deliverables []DeliverableSpec `yaml:"deliverables"`
}

// DeliverableSpec describes an expected output a work item must produce.
type DeliverableSpec struct {
	// Code is the UniqueKey component, e.g. "DEL-WI001-001".
	Code        string `yaml:"code"`
	RefCode     string `yaml:"ref_code"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Ordinality  int    `yaml:"ordinality"`
	// Blocking=true halts the workflow if this deliverable is rejected.
	Blocking bool `yaml:"blocking"`
}

// ── HTTP response types ───────────────────────────────────────────────────────

// entityItem is the JSON shape returned by EntityService for a single entity.
// The id field is the server-assigned UUID used as draft_ref_code, draft_workflow_ref_code,
// etc. in subsequent calls.
type entityItem struct {
	ID         string         `json:"id"`
	Properties map[string]any `json:"properties"`
}

// listEntitiesResp is the JSON shape returned by EntityService.ListEntities.
type listEntitiesResp struct {
	Entities []entityItem `json:"entities"`
}

// crossError is the standard error envelope returned by CodeValdCross.
type crossError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e crossError) Error() string { return e.Code + ": " + e.Message }
