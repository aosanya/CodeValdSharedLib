---
agent: agent
---

# Debug a CodeValdSharedLib Issue

## How to Use This Prompt

When you encounter a bug in CodeValdSharedLib, describe the failing behaviour and use the guidelines below to add targeted debug logging, isolate the cause, and clean up before merging.

## Common Failure Scenarios

### Scenario 1: `Registrar.Run` Does Not Reconnect After gRPC Error
**Symptom**: Registration call fails; subsequent heartbeats are not retried
**Cause**: Error not handled in ping loop; connection not re-dialled
**Check**: Inspect `registrar/registrar.go` ping loop — ensure errors trigger a re-dial with backoff

### Scenario 2: `serverutil.RunWithGracefulShutdown` Hangs
**Symptom**: Service does not exit after context cancellation
**Cause**: `grpc.Server.GracefulStop()` waiting for a stalled RPC; drain timeout not enforced
**Check**: Confirm `drainTimeout` is set and a `time.AfterFunc` or `context.WithTimeout` guard is used

### Scenario 3: `arangoutil.Connect` Returns Auth Error
**Symptom**: `401 Unauthorized` or `database not found` from ArangoDB driver
**Cause**: `Config.Username`/`Config.Password` wrong, or database name typo
**Check**: Log `cfg.Endpoint` and `cfg.Database` at entry; verify against `docker-compose.yml` or env vars

### Scenario 4: Context Cancellation Not Respected in `Registrar.Run`
**Symptom**: Registrar goroutine keeps running after caller cancels context
**Cause**: Missing `ctx.Done()` check in the ping ticker loop
**Check**: Add `ctx.Err()` check at the top of each loop iteration

### Scenario 5: `EnvOrDefault` Not Returning Expected Value
**Symptom**: Default value returned even though env var is set
**Cause**: Env var name mismatch or env var set after `os.Setenv` call timing issue in tests
**Check**: Log `os.Getenv(key)` before the fallback to confirm the value

## Debug Print Guidelines

### Prefix Format
All debug prints MUST be prefixed with: `[TASK-ID]`

### Go
```go
log.Printf("[SHAREDLIB-XXX] Function called: %s with args: %+v", functionName, args)
log.Printf("[SHAREDLIB-XXX] State before: %+v", state)
log.Printf("[SHAREDLIB-XXX] Error in operation: %v", err)
```

### Strategic Placement

Add debug prints at:

1. **Function Entry Points**
   - Log function name and key parameters
   - Example: `log.Printf("[SHAREDLIB-XXX] Connect called: endpoint=%s db=%s", cfg.Endpoint, cfg.Database)`

2. **State Changes**
   - Before and after critical state modifications
   - Example: `log.Printf("[SHAREDLIB-XXX] Registrar dialling: addr=%s", crossAddr)`

3. **Conditional Branches**
   - Log which branch is taken and why
   - Example: `log.Printf("[SHAREDLIB-XXX] Re-dialling after error: %v", err)`

4. **Loop Iterations** (for ping/retry loops)
   - Log iteration count and key variables
   - Example: `log.Printf("[SHAREDLIB-XXX] Ping attempt %d: service=%s", attempt, r.serviceName)`

5. **Error Handling**
   - Log errors with context before returning
   - Example: `log.Printf("[SHAREDLIB-XXX] Connect failed: endpoint=%s err=%v", cfg.Endpoint, err)`

6. **Return Statements** (for complex functions)
   - Log what is being returned
   - Example: `log.Printf("[SHAREDLIB-XXX] RunWithGracefulShutdown: drain complete")`

### What NOT to Debug

Avoid adding debug prints to:
- Simple getters/setters
- Trivial utility functions (`EnvOrDefault`, `ParseDurationSeconds`)
- Hot paths called on every RPC
- Already well-instrumented code

### Cleanup Instructions

Always add a comment above debug blocks:
```go
// TODO: Remove debug log for SHAREDLIB-XXX after issue is resolved
log.Printf("[SHAREDLIB-XXX] Debug info here")
```

## Execution Steps

1. **Identify Task ID** from branch or context
2. **Analyse the failing code path** in `registrar/`, `serverutil/`, or `arangoutil/`
3. **Select strategic points** where debug prints will be most valuable
4. **Add debug prints** with proper format and task ID prefix
5. **Run tests** and filter output: `go test ./... 2>&1 | grep SHAREDLIB-XXX`
6. **Explain placement** briefly — why each print was added

## Output Format

After adding debug prints, provide:

```markdown
### Debug Prints Added for [TASK-ID]

**File**: `path/to/file.go`

**Locations**:
1. Line XX: Function entry — logs parameters
2. Line YY: State change — logs before/after values
3. Line ZZ: Conditional check — logs decision logic

**Usage**: Run `go test -v ./... 2>&1 | grep SHAREDLIB-XXX` to see output.
```

## Remember

- **Always** use task ID prefix
- **Be strategic** - don't over-instrument
- **Be descriptive** - logs should tell a story
- **Be consistent** - use same format throughout
- **Be removable** - add TODO comments for cleanup
