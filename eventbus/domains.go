package eventbus

// Domain prefix constants for CodeVald event topics.
// Every service Topic* constant must be built from one of these so that a
// prefix rename is a single-file change with compile-time enforcement.
//
// As of BUG-20260609-001 the platform is dropping per-service domain prefixes
// in favor of intent-keyed topic names (e.g. `task.assigned` instead of
// `work.task.assigned`). Domains are blanked out service-by-service as their
// rename ships; the constants remain so existing concatenation expressions
// keep compiling without churning every call site.
const (
	DomainWork      = ""
	DomainGit       = "git."
	DomainAI        = "ai."
	DomainComm      = "comm."
	DomainFunctions = "functions."
	DomainAgency    = "agency."
	DomainOrg       = "org."
	DomainCross     = "cross."
	DomainPubSub    = "pubsub."
)
