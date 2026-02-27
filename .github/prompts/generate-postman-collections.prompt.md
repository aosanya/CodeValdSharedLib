---
agent: agent
---

# Regenerate Proto Stubs for CodeValdCross

This prompt guides regeneration of the Go code generated from `proto/codevaldcross/v1/` when the CodeValdCross proto contract changes.

## When to Use This Prompt

- A new RPC method is added to `proto/codevaldcross/v1/orchestrator.proto`
- A message type is added, renamed, or has fields changed
- Any consuming service reports compile errors against `gen/go/codevaldcross/v1/`

## Steps

### 1. Edit the Proto Source

All proto changes are made to:
```
proto/codevaldcross/v1/orchestrator.proto
```

Follow standard proto3 field numbering rules:
- **Never reuse** a field number that was previously used (even for removed fields)
- **Append** new fields at the end of a message; do not renumber existing fields
- Add a godoc-style comment above every new `rpc` and `message` entry

### 2. Regenerate the Go Stubs

```bash
cd /workspaces/CodeValdSharedLib

# Lint the proto first
buf lint

# Regenerate
buf generate

# Confirm gen/ directory was updated
git diff --stat gen/
```

Expected output files in `gen/go/codevaldcross/v1/`:
- `orchestrator.pb.go` — message types
- `orchestrator_grpc.pb.go` — server/client interfaces

**Never hand-edit any file in `gen/`.** Treat them as build artefacts.

### 3. Verify the Module Still Builds

```bash
go build ./...
go vet ./...
```

### 4. Verify Consuming Services Compile

For each consuming service, check that it compiles with the updated stubs:

```bash
cd /workspaces/CodeValdGit  && go build ./... && echo "git OK"
cd /workspaces/CodeValdWork && go build ./... && echo "work OK"
cd /workspaces/CodeValdCross && go build ./... && echo "cross OK"
```

If a consuming service uses a `replace` directive pointing to `../CodeValdSharedLib`, it will pick up the new gen code automatically.

### 5. Update Documentation

- If a new RPC was added, note it in `documentation/2-SoftwareDesignAndArchitecture/architecture.md` under the `gen/go/codevaldcross/v1` package section
- If a message type changed in a breaking way, note the migration path for consumers

## buf.yaml Configuration

`buf.yaml` at the repo root controls which proto files are included and which linting rules apply. Do not change lint rules without team review.

## Success Criteria

- ✅ `buf lint` passes with 0 warnings
- ✅ `buf generate` succeeds and `gen/` is updated
- ✅ `go build ./...` and `go vet ./...` pass in CodeValdSharedLib
- ✅ All consuming services compile against the new stubs
- ✅ No hand-edits to any file under `gen/`
