package eventbus

// Git-domain topic constants for cross-service consumers.
//
// These are the subset of CodeValdGit topics that other services need to
// reference without importing CodeValdGit directly. The values are identical
// to the constants in CodeValdGit/events.go — SharedLib is the single source
// of truth for any topic string that crosses a service boundary.
const (
	// TopicGitFileWrite is consumed by CodeValdGit to write a file on a branch.
	// Published by CodeValdAI when the LLM emits a git.file.write action.
	TopicGitFileWrite = DomainGit + "file.write"

	// TopicGitFileWritten fires after CodeValdGit successfully writes a file.
	// Consumed by CodeValdAI to update the run debrief.
	TopicGitFileWritten = DomainGit + "file.written"

	// TopicGitBranchCreate is consumed by CodeValdGit to create a branch on demand.
	// Published by CodeValdAI when the LLM emits a git.branch.create action.
	TopicGitBranchCreate = DomainGit + "branch.create"
)
