---
agent: agent
---

# Start New Task

Follow the **mandatory task startup process** for CodeValdSharedLib tasks:

## Task Startup Process (MANDATORY)

1. **Select the next task**
   - Check `documentation/3-SofwareDevelopment/mvp.md` for the task list and current status
   - Prefer foundational tasks (SHAREDLIB-001 module init, then SHAREDLIB-002 types) before dependent ones

2. **Read the specification**
   - Re-read the corresponding section in `documentation/2-SoftwareDesignAndArchitecture/architecture.md`
   - Understand what the package exports, what its allowed imports are, and which services consume it
   - Note the dependency rule: SharedLib must **never** import from a CodeVald service

3. **Create feature branch from `main`**
   ```bash
   cd /workspaces/CodeValdSharedLib
   git checkout main
   git pull origin main
   git checkout -b feature/SHAREDLIB-XXX_description
   ```
   Branch naming: `feature/SHAREDLIB-XXX_description` (lowercase with underscores)

4. **Read project guidelines**
   - Review `.github/instructions/rules.instructions.md`
   - Key rules: interface-first, no CodeVald service imports, context propagation, godoc on all exports, no domain logic

5. **Create a todo list**
   - Break the task into actionable steps
   - Use the manage_todo_list tool to track progress
   - Mark items in-progress and completed as you go

## Pre-Implementation Checklist

Before starting:
- [ ] Architecture section for this package re-read
- [ ] Feature branch created: `feature/SHAREDLIB-XXX_description`
- [ ] Confirmed no duplicate types or functions in existing SharedLib packages
- [ ] Understood which package to create or modify (`registrar/`, `serverutil/`, `arangoutil/`, `types/`, `gen/`)
- [ ] Confirmed this code is genuinely shared (used by ≥2 services or expected to be)
- [ ] Todo list created for this task

## Development Standards

- **No imports from CodeVald services** — only stdlib, gRPC, ArangoDB driver, protobuf runtime
- **No domain logic** — infrastructure and plumbing only
- **Every exported symbol** must have a godoc comment
- **Every exported method** takes `context.Context` as the first argument
- **Service-specific values** (service name, topics, routes) are always constructor arguments, never hardcoded

## Git Workflow

```bash
# Create feature branch
git checkout -b feature/SHAREDLIB-XXX_description

# Regular commits during development
git add .
git commit -m "SHAREDLIB-XXX: Descriptive message"

# Build validation before merge
go build ./...           # must succeed
go test -v -race ./...   # must pass
go vet ./...             # must show 0 issues
golangci-lint run ./...  # must pass

# Merge when complete
git checkout main
git merge feature/SHAREDLIB-XXX_description --no-ff
git branch -d feature/SHAREDLIB-XXX_description
```

## Success Criteria

- ✅ Architecture doc reviewed
- ✅ Feature branch created from `main`
- ✅ Todo list created with implementation steps
- ✅ Ready to implement following library design rules
