package eventbus

// Domain prefix constants for CodeVald event topics.
// Every service Topic* constant must be built from one of these so that a
// prefix rename is a single-file change with compile-time enforcement.
const (
	DomainWork      = "work."
	DomainGit       = "git."
	DomainAI        = "ai."
	DomainComm      = "comm."
	DomainFunctions = "functions."
	DomainAgency    = "agency."
	DomainOrg       = "org."
	DomainCross     = "cross."
	DomainPubSub    = "pubsub."
)
