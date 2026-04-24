# Route Write Classification

## Status

Proposed — 2026-04-24.

## Context

CodeValdCross gates write traffic with HTTP Basic auth (interim, replaced by
OAuth2 later). The middleware today decides read-vs-write from the HTTP
method alone, with two hardcoded Git Smart HTTP exceptions:

- `POST /…/git-upload-pack` → read (clone/fetch data)
- `GET  /…/info/refs?service=git-receive-pack` → write (push discovery)

This is too blunt. Downstream services already expose non-mutating operations
over `POST` because the query payload does not fit a URL — the standard REST
pattern for search and graph queries. Concrete cases in CodeValdGit:

| Route | Semantics | Current classification |
|---|---|---|
| `POST /git/{agencyId}/graph/search` | read (`SearchByKeywords`) | incorrectly treated as write |
| `POST /git/{agencyId}/repositories/{repoName}/branches/{branchName}/graph/query` | read (`QueryGraph`) | incorrectly treated as write |

The classifier cannot recover the right answer from the URL alone without
carrying service-specific knowledge into Cross. That couples the auth gate to
every downstream service's URL shapes and requires edits to Cross every time a
new POST-read appears anywhere.

## Decision

Make each route carry its own write/read classification. Services declare it
at registration time; Cross's middleware consults it on every request.

Add a single boolean field, **`IsWrite`**, to:

1. `CodeValdSharedLib/types.RouteInfo` — the Go struct used inside each service
   and sent as JSON where applicable.
2. `CodeValdSharedLib/proto/codevaldcross/v1/orchestrator.proto` on the
   `RouteDeclaration` message — the wire format of the Register RPC.

### Default value

`IsWrite = false` (safe default for reads). Rationale:

- Go's zero value for `bool` is `false`; existing route declarations that are
  not updated stay readable until the service author annotates them. This
  keeps the change forward-compatible: old services register successfully,
  their traffic flows through, and nothing breaks — they just lose the auth
  gate until they opt in.
- The safer default for a security gate would be `IsWrite = true`
  (fail-closed), but that would 401 every route on any not-yet-migrated
  service. Since the gate itself is interim (replaced by OAuth2), the lower
  migration friction wins. When OAuth2 lands it will not lean on `IsWrite` —
  identity propagation is orthogonal.
- CodeValdGit is the only downstream service registering today, and every
  route in its registrar gets annotated as part of this change, so the field
  is never silently defaulted in the current deployment.

### Middleware behaviour

The auth middleware lives in front of the dynamic proxy, so it does not yet
know which route matched. Two options:

**A. Match first, then gate.** Move route matching into a shared helper that
runs before auth, attach the matched route to the request context, then let
auth read `IsWrite` from context.

**B. Duplicate matching.** Auth runs its own lookup, proxy runs the real one.
Simple, but the match work runs twice per request.

Pick **A**. The match result is then reused by the proxy — net perf improves.
The context key lives in the `server` package of CodeValdCross. When no route
matches, auth treats the request as a write (fail-closed), which preserves
the existing behaviour where unknown routes return 404 from the proxy anyway.

The two existing Git Smart HTTP exceptions (`git-upload-pack`, the
`service=git-receive-pack` query parameter on `info/refs`) are encoded as
`IsWrite` values on those specific routes in
`CodeValdGit/internal/registrar/smart_http_routes.go`, not as middleware
special cases. `info/refs` is a single `GET` route — if services need per-
query-string classification in future, the middleware can fall back to the
current hardcoded check, but today the whole route can be classified as
`IsWrite = false` (the push-discovery subcase is rare enough that a single
unauthenticated probe is acceptable, and the subsequent `git-receive-pack`
POST is still gated). **Alternative**: keep one hardcoded special case for
`info/refs` in the middleware. See "Open questions" below.

## Alternatives considered

1. **URL-suffix matching in middleware.** Match `/graph/query`, `/graph/search`
   and treat as reads. Rejected: bleeds service-specific URL knowledge into
   Cross and does not scale to new services.
2. **Change POST → GET in CodeValdGit.** Rejected: the request bodies are
   structured JSON that do not fit URL query strings.
3. **Use the `Capability` string.** Treat certain capability name prefixes
   (`list_`, `get_`, `search_`, `query_`) as reads. Rejected: relies on naming
   convention and leaves the door open to accidental misclassification — an
   explicit boolean is unambiguous.

## File-by-file change plan

| # | Repo | File | Change |
|---|---|---|---|
| 1 | SharedLib | `types/types.go` | Add `IsWrite bool json:"is_write"` to `RouteInfo`. Update doc comment. |
| 2 | SharedLib | `types/types_test.go` | Extend `TestRouteInfo_JSONRoundTrip` and `TestRouteInfo_OmitEmpty` for the new field. |
| 3 | SharedLib | `proto/codevaldcross/v1/orchestrator.proto` | Add `bool is_write = <next>;` to `RouteDeclaration`. |
| 4 | SharedLib | `gen/go/codevaldcross/v1/*` | Regenerate from proto. |
| 5 | SharedLib | `registrar/registrar.go` | Carry `IsWrite` from `types.RouteInfo` → `crossv1.RouteDeclaration` in the conversion step inside `New`. |
| 6 | SharedLib | `schemaroutes/schemaroutes.go` | The generated routes (CRUD from schema) set `IsWrite` per verb: GET/HEAD → false, POST/PUT/PATCH/DELETE → true. |
| 7 | Cross | `internal/pubsub/registry.go` | Carry `IsWrite` back from `crossv1.RouteDeclaration` → `types.RouteInfo` when storing registrations. |
| 8 | Cross | `internal/server/proxy.go` | Expose a `matchRoute` helper that attaches the match to `r.Context()` via a private key. Reuse the matched route in the proxy body. |
| 9 | Cross | `internal/server/http.go` | `authMiddleware` reads the matched route from context: `IsWrite == true` → challenge, otherwise passthrough. Remove the HTTP-method-based `requiresAuth` fallback except for `OPTIONS` and `/health`. |
| 10 | Cross | `internal/server/http_auth_test.go` | Rewrite subtests around the context-attached match rather than URL shapes. |
| 11 | Git | `internal/registrar/*.go` | Annotate every route with `IsWrite: true/false`. Defaults are listed below. |

### Classification in CodeValdGit

| File | Route | `IsWrite` |
|---|---|---|
| branch_routes.go | POST create_branch / POST merge | true |
| branch_routes.go | GET list / GET byName / GET byId | false |
| branch_routes.go | DELETE delete_branch | true |
| repo_routes.go | POST create_repository, POST archive/unarchive | true |
| repo_routes.go | GET list / GET get | false |
| tag_routes.go | POST create_tag, DELETE delete_tag | true |
| tag_routes.go | GET list / GET get | false |
| file_routes.go | POST/PUT/PATCH/DELETE | true |
| file_routes.go | GET read / GET list | false |
| history_routes.go | GET log / GET diff | false |
| import_routes.go | POST import_repository | true |
| fetch_branch_routes.go | POST fetch_branch | true |
| docs_routes.go | POST create_keyword, POST create_edge, PUT, DELETE | true |
| docs_routes.go | GET list/get keywords, GET tree, GET neighbourhood | false |
| docs_routes.go | **POST graph/search** (`SearchByKeywords`) | **false** |
| docs_routes.go | **POST graph/query** (`QueryGraph`) | **false** |
| smart_http_routes.go | GET info/refs | false (see open question) |
| smart_http_routes.go | POST git-upload-pack | false |
| smart_http_routes.go | POST git-receive-pack | true |

## Migration

- Services not yet migrated register without `IsWrite` on any route. In the
  default `false` mode, their traffic is not gated — which is the current
  interim posture anyway for read-path traffic.
- CodeValdGit is annotated in the same PR as the SharedLib change, so the
  deployment goes out consistent. No flag, no staged rollout needed.
- When OAuth2 lands, the auth middleware is replaced entirely; `IsWrite`
  remains useful for downstream decisions (audit, per-verb rate limits,
  idempotency) and is not discarded.

## Open questions

1. **`info/refs` classification.** Today it is a single route serving both
   clone discovery (read) and push discovery (write) — distinguished by the
   `service` query parameter. Options:
   - (a) Declare the route `IsWrite = false` and accept that an unauthenticated
     caller can enumerate refs before push. The subsequent `git-receive-pack`
     POST is still challenged.
   - (b) Keep a single hardcoded exception in the middleware for
     `/info/refs` + `service=git-receive-pack`.
   Recommendation: **(b)** — one hardcoded case in Cross is acceptable for
   the Git wire protocol's unusual shape. Everything else uses `IsWrite`.
2. **Non-registered (built-in) Cross routes.** `/health` and
   `/services/registry` have no `RouteInfo`. Leave the current allowlist in
   the middleware (`/health` bypass) and explicitly gate `/services/registry`
   as a write — it leaks service topology.

## Testing

- SharedLib: JSON round-trip tests cover the new field.
- Cross:
  - auth middleware tests: add subtests that build a request context with a
    matched route carrying `IsWrite = true/false` and assert the expected
    pass/challenge.
  - integration: table-driven test against the full stack asserting that the
    two `/graph/*` POSTs pass without credentials and that write routes (e.g.
    `POST /branches`) still 401.
- Git: each registrar file already tests its route shapes; extend to assert
  the new `IsWrite` value per route.
