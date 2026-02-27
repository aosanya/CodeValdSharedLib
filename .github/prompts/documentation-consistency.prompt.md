---
agent: agent
---

# Documentation Consistency & Organization Checker

## Purpose
Perform systematic documentation consistency checks through **one question at a time**, identifying outdated references, consolidating related files, and organizing documentation structure for maintainability.

---

## Instructions for AI Assistant

Conduct a comprehensive documentation consistency analysis through **iterative single-question exploration**. Ask ONE question at a time, wait for the response, then decide whether to:
- **🔍 DEEPER**: Go deeper into the same topic with follow-up questions
- **📝 NOTE**: Record an issue/gap for later action
- **➡️ NEXT**: Move to the next consistency check area
- **📊 REVIEW**: Summarize findings and determine next steps

The goal is to systematically check documentation consistency one area at a time rather than overwhelming with batch operations.

---

## Current Technology Stack (Reference)

**Update this section when stack changes:**

```yaml
Library:
  Language: Go 1.21+
  Module: github.com/aosanya/CodeValdSharedLib
  Purpose: Shared infrastructure library for all CodeVald microservices

Packages:
  - registrar: Generic Registrar (Run/Close/ping loop) — replaces per-service internal/registrar/
  - serverutil: NewGRPCServer, RunWithGracefulShutdown, EnvOrDefault, ParseDuration*
  - arangoutil: Connect(ctx, Config) driver.Database
  - types: PathBinding, RouteInfo, ServiceRegistration
  - gen/go/codevaldcross/v1: Generated Go stubs for CodeValdCross proto

Consumers:
  - CodeValdCross (types, gen)
  - CodeValdGit (registrar, serverutil, arangoutil, gen)
  - CodeValdWork (registrar, serverutil, arangoutil, gen)
  - Future services

Dependency rule:
  SharedLib MUST NOT import from any CodeVald service.
  Allowed imports: stdlib, google.golang.org/grpc, github.com/arangodb/go-driver, protobuf runtime

Documentation structure:
  2-SoftwareDesignAndArchitecture:
    architecture: documentation/2-SoftwareDesignAndArchitecture/architecture.md
  3-SofwareDevelopment:
    mvp: documentation/3-SofwareDevelopment/mvp.md
```

---

## Question-by-Question Consistency Check Process

## Question-by-Question Consistency Check Process

### Session Initiation

When starting a documentation consistency check:

1. **State the scope** - Which documentation area are we checking?
2. **Scan quickly** - Get overview of file structure and sizes
3. **Ask the first question** - Start with highest priority check
4. **Wait for user input** - Get confirmation or additional context before proceeding

### Question Flow

**After each answer, explicitly choose one of these paths:**

- 🔍 **DEEPER**: "Let me examine this area more closely..."
  - Investigate specific files flagged
  - Check related documents
  - Verify cross-references

- 📝 **NOTE**: "I'll note this inconsistency: [description]..."
  - Record issue for action list
  - Mark files needing updates
  - Continue to different check

- ➡️ **NEXT**: "Moving to [new consistency check area]..."
  - Current check complete
  - Proceed to next question category
  - Maintain systematic progress

- 📊 **REVIEW**: "Let me summarize what we've found..."
  - List inconsistencies discovered
  - Identify files to archive/update
  - Propose actions

---

## Question Categories (Execute in Order)

### Question 1: Technology Stack Consistency

**Objective**: Verify all documentation reflects current technology decisions.

**Question Format**:
```
🔍 [Technology Stack Check]

**Question**: Are there any references to [outdated technology] in the 
current documentation?

**Search Pattern**: [specific grep pattern or file locations]

**Context**: We migrated from [old tech] to [new tech], need to verify 
no outdated references remain.

**What I'm Looking For**: File paths with outdated references, or 
confirmation that area is clean.
```

**Common Searches**:
- `replace` directive references (ensure all consuming services point to correct local path)
- `github.com/aosanya/CodeValdSharedLib` module path (ensure consistent across all `go.mod` files)
- Any reference to service-specific types that belong in `types/` but aren't there yet

**For Each Match Found**:
- **🔍 DEEPER**: Read file to determine context
- **📝 NOTE**: Is it outdated? Archive or update?
- **➡️ NEXT**: No issues found, proceed to next check

---

### Question 2: Cross-Reference Validation

**Objective**: Ensure all internal document links point to current files.

**Question Format**:
```
🔍 [Link Validation Check]

**Question**: Do all links in [specific file or folder] point to existing, 
non-archived documents?

**Context**: Need to verify README and index files have current references.

**What I'm Looking For**: Broken links, links to archived files, or 
confirmation all links are valid.
```

**Actions Per Broken Link**:
- **🔍 DEEPER**: Find the correct replacement document
- **📝 NOTE**: Add to update list
- **➡️ NEXT**: All links valid, move to next file/folder

---

### Question 3: File Organization Analysis

**Objective**: Identify documentation that needs consolidation or subfolder organization.

**Question Format**:
```
🔍 [File Organization Check]

**Question**: Are there 3 or more files in [folder] that share a common 
topic prefix (e.g., "agency-*.md", "a2a-*.md")?

**Context**: Files with shared topics should be organized in subfolders 
for better discoverability.

**What I'm Looking For**: File groupings that exceed organization threshold.
```

**Organization Rules**:
- **≤ 2 files on topic**: Keep in current directory
- **3+ files on topic**: Create subfolder named after topic
- **500+ lines**: Consider splitting into smaller focused documents
- **Duplicate content**: Consolidate into single source of truth

**For Each Topic Group**:
- **🔍 DEEPER**: Should these be merged or kept in subfolder?
- **📝 NOTE**: Add to organization action list
- **➡️ NEXT**: No groupings found, continue

---

### Question 4: File Size Compliance

**Objective**: Ensure documents remain maintainable and scannable.

**Question Format**:
```
🔍 [File Size Check]

**Question**: Are there any .md files in [folder] exceeding 500 lines?

**Context**: Large files should be split for better maintainability.

**What I'm Looking For**: Files over 500 lines (warning) or 1000+ lines 
(action required).
```

**Size Guidelines**:
- **Ideal**: 200-500 lines per document
- **Warning**: 500-1000 lines (consider splitting)
- **Action Required**: 1000+ lines (must split or justify)

**For Each Large File**:
- **🔍 DEEPER**: Analyze structure - can it be split logically?
- **📝 NOTE**: Add to refactoring list with split strategy
- **➡️ NEXT**: All files within guidelines

---

### Question 5: Naming Convention Compliance

**Objective**: Ensure consistent, discoverable file naming.

**Question Format**:
```
🔍 [Naming Convention Check]

**Question**: Do all files in [folder] follow the naming convention 
pattern [expected-pattern.md]?

**Context**: Consistent naming improves discoverability and automation.

**What I'm Looking For**: Files violating naming conventions.
```

**Naming Standards**:
```yaml
Architecture Documents:
  Pattern: "kebab-case-descriptive-name.md"
  Examples: "backend-architecture.md", "a2a-protocol-integration.md"
  
MVP Details:
  Pattern: "MVP-XXX.md" or "MVP-XXX-descriptive-name.md"
  
Use Case Documentation:
  Pattern: "UC-ABBR-NNN-short-name/"
  
Coding Sessions:
  Pattern: "MVP-XXX_descriptive_name.md" or "TASK-NNN_description.md"
  
Archive Files:
  Pattern: "original-name-deprecated.md"
```

**For Each Violation**:
- **🔍 DEEPER**: What's the correct name per convention?
- **📝 NOTE**: Add to rename action list
- **➡️ NEXT**: All names compliant

---

### Question 6: Content Duplication Detection

**Objective**: Identify and consolidate duplicate or near-duplicate content.

**Question Format**:
```
🔍 [Duplication Check]

**Question**: Are there multiple files in [folder] covering the same topic 
or with similar content?

**Context**: Duplicate content creates maintenance burden and confusion.

**What I'm Looking For**: Files with overlapping purpose or >70% similar content.
```

**For Each Potential Duplicate**:
- **🔍 DEEPER**: Compare files to verify duplication level
- **📝 NOTE**: Decide merge strategy or keep with cross-references
- **➡️ NEXT**: No duplicates detected

---

### Question 7: Interface and API Consistency Check

**Objective**: Verify that `documentation/2-SoftwareDesignAndArchitecture/architecture.md` is consistent with the implemented packages and exported APIs.

**Question Format**:
```
🔍 [Package API Consistency Check]

**Question**: Does the exported API in `registrar/`, `serverutil/`, `arangoutil/`,
and `types/` match what is described in `architecture.md`?

**Context**: architecture.md is the source of truth for what SharedLib exports — 
code and docs must agree.

**What I'm Looking For**: Missing exports, renamed functions, or signature mismatches.
```

**Check Areas**:
1. `registrar.New` constructor signature and `Run`/`Close` methods
2. `serverutil.NewGRPCServer`, `RunWithGracefulShutdown`, `EnvOrDefault`, `ParseDuration*`
3. `arangoutil.Connect` signature and `Config` struct fields
4. `types.PathBinding`, `types.RouteInfo`, `types.ServiceRegistration` field names

**For Each Inconsistency**:
- **🔍 DEEPER**: Which is more recent — code or architecture doc?
- **📝 NOTE**: Record the discrepancy and which file to update
- **➡️ NEXT**: API definitions are consistent

---

### Question 8: Production Readiness - Security & Authentication

**Objective**: Verify security documentation and implementation completeness for production deployment.

**Question Format**:
```
🔍 [Security Production Readiness Check]

**Question**: Is there comprehensive documentation covering authentication, 
authorization, secret management, and security hardening for production?

**Context**: Production systems require robust security measures to protect 
user data and prevent unauthorized access.

**What I'm Looking For**: Documentation gaps in:
- Authentication mechanisms (JWT, OAuth, API keys)
- Authorization/RBAC implementation
- Secrets management (environment variables, vaults)
- TLS/HTTPS configuration
- API rate limiting and throttling
- Input validation and sanitization
- Security headers and CORS policies
- Audit logging for security events
```

**Production Security Checklist**:
- ✅ Authentication flow documented
- ✅ Authorization/permissions model defined
- ✅ Secret rotation strategy documented
- ✅ Security testing procedures defined
- ✅ Incident response plan exists
- ✅ Data encryption at rest/in transit documented
- ✅ Vulnerability scanning process defined
- ✅ Security compliance requirements addressed

**For Each Gap**:
- **🔍 DEEPER**: Check implementation files for undocumented security features
- **📝 NOTE**: Add missing documentation to action list
- **➡️ NEXT**: Security documentation complete

---

### Question 9: Production Readiness - Monitoring & Observability

**Objective**: Ensure monitoring, logging, and alerting are production-ready.

**Question Format**:
```
🔍 [Monitoring Production Readiness Check]

**Question**: Is there documentation for production monitoring, logging 
infrastructure, metrics collection, and alerting strategies?

**Context**: Production systems require comprehensive observability to 
detect and resolve issues quickly.

**What I'm Looking For**: Documentation gaps in:
- Metrics collection (Prometheus, custom metrics)
- Logging infrastructure (structured logging, log aggregation)
- Distributed tracing (if microservices)
- Alerting rules and escalation policies
- Dashboard configurations
- SLI/SLO/SLA definitions
- Performance monitoring
- Error tracking and reporting
```

**Production Observability Checklist**:
- ✅ Metrics endpoints documented
- ✅ Log format and retention policies defined
- ✅ Critical alerts documented (SLIs)
- ✅ Dashboard designs specified
- ✅ On-call procedures documented
- ✅ Runbook for common issues exists
- ✅ Performance baselines established
- ✅ Error budget policy defined

**For Each Gap**:
- **🔍 DEEPER**: Check deployments/prometheus.yml and implementation
- **📝 NOTE**: Add missing observability documentation
- **➡️ NEXT**: Monitoring documentation complete

---

### Question 10: Production Readiness - Deployment & Infrastructure

**Objective**: Verify deployment procedures, infrastructure configuration, and disaster recovery plans.

**Question Format**:
```
🔍 [Deployment Production Readiness Check]

**Question**: Is there complete documentation for deployment processes, 
infrastructure as code, scaling strategies, and disaster recovery?

**Context**: Production deployments require reliable, repeatable processes 
and recovery mechanisms.

**What I'm Looking For**: Documentation gaps in:
- CI/CD pipeline configuration
- Infrastructure as Code (Terraform, k8s manifests)
- Environment configuration (dev/staging/prod)
- Database migration procedures
- Rollback procedures
- Scaling strategies (horizontal/vertical)
- Backup and restore procedures
- Disaster recovery plan (RTO/RPO)
- Blue-green or canary deployment strategy
```

**Production Deployment Checklist**:
- ✅ CI/CD pipeline documented
- ✅ Environment variables catalog exists
- ✅ Database migration runbook exists
- ✅ Rollback procedures documented
- ✅ Backup schedule and testing documented
- ✅ Infrastructure diagrams current
- ✅ Scaling thresholds defined
- ✅ DR plan tested and documented

**For Each Gap**:
- **🔍 DEEPER**: Check docker-compose.yml, Dockerfile, deployments/
- **📝 NOTE**: Add missing deployment documentation
- **➡️ NEXT**: Deployment documentation complete

---

### Question 11: Production Readiness - Data Management & Compliance

**Objective**: Ensure data handling, privacy, and compliance requirements are documented.

**Question Format**:
```
🔍 [Data Management Production Readiness Check]

**Question**: Is there documentation covering data models, database schemas, 
data retention policies, and regulatory compliance requirements?

**Context**: Production systems must handle data responsibly and comply 
with regulations (GDPR, CCPA, etc.).

**What I'm Looking For**: Documentation gaps in:
- Database schema documentation (ArangoDB collections)
- Data retention and archival policies
- PII (Personally Identifiable Information) handling
- GDPR/CCPA compliance procedures
- Data backup and recovery testing
- Database performance optimization
- Migration and upgrade procedures
- Data validation rules
```

**Production Data Management Checklist**:
- ✅ Schema documentation current
- ✅ Data retention policies defined
- ✅ PII handling documented
- ✅ Compliance requirements addressed
- ✅ Backup verification procedures exist
- ✅ Performance tuning guidelines documented
- ✅ Data migration tested
- ✅ Data access controls documented

**For Each Gap**:
- **🔍 DEEPER**: Check internal/database/ and compliance docs
- **📝 NOTE**: Add missing data management documentation
- **➡️ NEXT**: Data documentation complete

---

### Question 12: Production Readiness - API Documentation & Versioning

**Objective**: Verify API documentation is complete and production-ready for external consumers.

**Question Format**:
```
🔍 [API Production Readiness Check]

**Question**: Is there comprehensive API documentation including endpoints, 
request/response schemas, error codes, rate limits, and versioning strategy?

**Context**: Production APIs must be well-documented for developers and 
support teams.

**What I'm Looking For**: Documentation gaps in:
- OpenAPI/Swagger specification
- Authentication requirements per endpoint
- Request/response examples
- Error code catalog with resolution steps
- Rate limiting and quota documentation
- API versioning strategy (v1, v2, etc.)
- Deprecation policy and timeline
- Breaking change communication plan
```

**Production API Documentation Checklist**:
- ✅ API specification (OpenAPI/Swagger) exists
- ✅ Authentication per endpoint documented
- ✅ All endpoints have examples
- ✅ Error codes documented with meanings
- ✅ Rate limits clearly specified
- ✅ Versioning strategy documented
- ✅ Deprecation policy defined
- ✅ API changelog maintained

**For Each Gap**:
- **🔍 DEEPER**: Check api/ and internal/api/ folders
- **📝 NOTE**: Add missing API documentation
- **➡️ NEXT**: API documentation complete

---

### Question 13: Production Readiness - Testing & Quality Assurance

**Objective**: Ensure testing coverage and quality gates are production-ready.

**Question Format**:
```
🔍 [Testing Production Readiness Check]

**Question**: Is there documentation for test coverage requirements, testing 
strategies, and quality gates for production releases?

**Context**: Production code requires comprehensive testing to ensure 
reliability and prevent regressions.

**What I'm Looking For**: Documentation gaps in:
- Unit test coverage requirements (minimum %)
- Integration test strategy
- End-to-end test scenarios
- Performance/load testing procedures
- Security testing (SAST/DAST)
- Regression test suite
- Test data management
- Quality gates for CI/CD
```

**Production Testing Checklist**:
- ✅ Test coverage targets defined (e.g., 80%+)
- ✅ Integration test strategy documented
- ✅ E2E test scenarios identified
- ✅ Performance benchmarks established
- ✅ Security testing integrated
- ✅ Test data management documented
- ✅ CI/CD quality gates configured
- ✅ Testing runbook exists

**For Each Gap**:
- **🔍 DEEPER**: Check test/ folder and CI configuration
- **📝 NOTE**: Add missing testing documentation
- **➡️ NEXT**: Testing documentation complete

---

### Question 14: Production Readiness - Operations & Support

**Objective**: Verify operational runbooks, support procedures, and maintenance documentation exist.

**Question Format**:
```
🔍 [Operations Production Readiness Check]

**Question**: Is there documentation for operational procedures, troubleshooting 
guides, support escalation, and maintenance windows?

**Context**: Production systems require clear operational procedures for 
support teams.

**What I'm Looking For**: Documentation gaps in:
- Operational runbooks for common tasks
- Troubleshooting guides with decision trees
- Support escalation matrix
- Maintenance window procedures
- Health check endpoints documentation
- System dependencies map
- Contact information for critical services
- Change management process
```

**Production Operations Checklist**:
- ✅ Runbooks for common operations exist
- ✅ Troubleshooting guides documented
- ✅ Escalation procedures defined
- ✅ Maintenance procedures documented
- ✅ Health check endpoints listed
- ✅ System dependency map current
- ✅ Emergency contacts documented
- ✅ Change management process defined

**For Each Gap**:
- **🔍 DEEPER**: Check documents/4-QA/ and operations docs
- **📝 NOTE**: Add missing operational documentation
- **➡️ NEXT**: Operations documentation complete

---

## Response Processing Guidelines

**After receiving each answer:**

1. **Acknowledge**: "✅ Understood - [brief summary of finding]"
2. **Decide Path**: Choose DEEPER/NOTE/NEXT/REVIEW
3. **State Choice**: "🔍 Going deeper..." / "📝 Noting issue..." / "➡️ Moving to..." / "📊 Reviewing..."
4. **Execute**: Ask next question OR provide review summary

**Example Flow**:
```
AI: 🔍 [Technology Stack Check]
    
    Question: Are there references to "React" in 
    /documents/2-SoftwareDesignAndArchitecture/?
    
    Context: We migrated to Flutter, need to verify no outdated references.
    
    What I'm Looking For: Files containing "React" references.

User: Found 3 files: frontend-architecture.md, react-migration-plan.md, 
      README.md

AI: ✅ Found 3 files with React references.
    
    🔍 Going deeper - checking if these are already archived or current:
    
    Question: Have frontend-architecture.md and react-migration-plan.md 
    already been archived, or are they still active documents?

User: frontend-architecture.md is now just a deprecation notice. 
      react-migration-plan.md is still in main folder.

AI: ✅ Understood - one is deprecation notice (good), one needs archiving.
    
    📝 Noting issue: react-migration-plan.md needs to be archived to 
    archive/react-migration-plan-deprecated.md
    
    ➡️ Moving to README.md check:
    
    Question: What React references exist in README.md?
```

---

## Issue Tracking During Session

**Maintain a running list visible every 3-5 questions:**

### 🚨 Inconsistencies Found
- 📝 **[File]**: Outdated tech reference - [specific issue]
- 📝 **[File]**: Broken link - [link target]
- 📝 **[Folder]**: Needs subfolder organization - [topic group]

### ✅ Verified Clean
- ✅ **[Area]**: No issues found
- ✅ **[File]**: Already compliant

### 🔄 Actions Required
- 🔧 Archive: [list of files]
- 🔧 Update: [list of files needing edits]
- 🔧 Organize: [folders needing restructure]
- 🔧 Rename: [files needing rename]

---

## Periodic Review Format

**Every 5-7 questions, provide progress summary:**

```
📊 **CONSISTENCY CHECK - Progress Review**

**Areas Checked:**
✅ Technology Stack (2-SoftwareDesignAndArchitecture/) - 3 issues found
✅ Cross-References (README files) - 2 broken links
⏸️ File Organization - Not yet checked
⏸️ File Sizes - Not yet checked

**Issues Identified:**
📝 react-migration-plan.md needs archiving
📝 README.md has 2 React references to update
📝 introduction.md references "React Developer" role

**Files to Archive:**
- react-migration-plan.md → archive/react-migration-plan-deprecated.md

**Files to Update:**
- README.md (2 locations)
- introduction.md (1 location)

**Remaining Checks:**
- File organization analysis
- File size compliance
- Naming convention check
- Duplication detection
- Use case mvp.md updates

**Next Steps**: Continue with file organization check, or address found 
issues first?
```

---

## Completion Criteria

**The consistency check session is complete when:**

- ✅ All 14 question categories have been systematically checked
- ✅ All inconsistencies have been identified and noted
- ✅ Action list is complete (archive/update/organize/rename)
- ✅ Production readiness gaps documented
- ✅ **Gaps documented in structured format** (see below)
- ✅ User confirms readiness to execute actions

---

## 📁 REQUIRED: Structured Gap Documentation

**CRITICAL**: All findings MUST be documented in the structured gaps directory for tracking and accountability.

### Gap Documentation Location

```
/workspaces/CodeValdCortex/documents/2-SoftwareDesignAndArchitecture/gaps/
├── README.md                           # Gap tracking overview
├── production-readiness/               # Production-critical gaps
│   ├── security-authentication.md
│   ├── deployment-infrastructure.md
│   ├── monitoring-observability.md
│   ├── data-compliance.md
│   └── api-documentation.md
├── file-organization/                  # File structure issues
│   ├── oversized-files.md
│   ├── duplicate-files.md
│   ├── misplaced-files.md
│   └── naming-violations.md
├── cross-references/                   # Link and reference issues
│   ├── broken-links.md
│   └── outdated-references.md
└── reports/                           # Timestamped full reports
    └── YYYY-MM-DD_consistency-check.md
```

### Required Documentation Actions

After completing the consistency check, you MUST:

1. **Create/Update Category-Specific Gap Files**
   - Update existing gap files with new findings
   - Create new gap files for new categories
   - Use consistent format (see templates in gaps/README.md)

2. **Create Timestamped Report**
   - File: `gaps/reports/YYYY-MM-DD_consistency-check.md`
   - Include executive summary, metrics, findings, and action plan
   - Reference category-specific gap files

3. **Update gaps/README.md**
   - Update "Current Gaps Summary" section
   - Update metrics and status counts
   - Update "Next Steps"

### Gap File Format Template

Each gap file MUST include:

```markdown
# [Category] [Type] Gap/Issue

**Last Updated**: YYYY-MM-DD  
**Priority**: 🔴 BLOCKER | 🟡 HIGH | 🟢 MEDIUM | ⚪ LOW  
**Status**: Open | In Progress | Resolved

## Summary
Brief description of the gap and impact

## Missing Documentation / Issues
- [ ] Item 1
- [ ] Item 2

## Impact
What happens if we ship without this?

## Recommended Solution
Specific files to create, actions to take

## Action Items
- [ ] Action 1
- [ ] Action 2

## Resolution
(Fill when resolved)
- **Resolved By**: 
- **Date**: 
- **Notes**: 
```

### Production Readiness Gap Template

For production blockers, use enhanced template:

```markdown
# [Category] Production Readiness Gap

**Last Updated**: YYYY-MM-DD  
**Priority**: 🔴 BLOCKER  
**Status**: Open

## Summary
CRITICAL description of missing production capability

## Missing Documentation
### 1. [Specific Doc Category]
- [ ] File 1
- [ ] File 2

### 2. [Another Category]
- [ ] File 3

## Current State
**What Exists:**
- ✅ Item that exists
- ⚠️ Partial item

**What's Missing:**
- ❌ Critical missing item 1
- ❌ Critical missing item 2

## Impact of Shipping Without This
1. Risk 1
2. Risk 2
3. Legal/compliance issue

## Recommended Solution
### Phase 1: Critical (Week 1)
Detailed creation plan

### Phase 2: Important (Week 2)
Next priority items

## Action Items
- [ ] Immediate action
- [ ] Urgent action
- [ ] Validation

## Resolution
(To be filled)
```

---

## Final Deliverable Structure

**After each consistency check session, create:**

### 1. Timestamped Report (Required)

**File**: `gaps/reports/YYYY-MM-DD_consistency-check.md`

**Contents**:
```markdown
# Documentation Consistency Check Report

**Date**: YYYY-MM-DD
**Scope**: [folders checked]

## Executive Summary
Overall assessment, key metrics, critical findings

## Metrics Summary
- Files Scanned: XX
- Issues Found: XX
- Production Blockers: XX
- High Priority: XX
- Medium Priority: XX

## Findings by Category
### 1. Technology Stack Consistency
### 2. File Organization
### 3. File Size Compliance
### 4. Cross-Reference Validation
### 5. Production Readiness

## Recommended Actions (Prioritized)
### 🔴 CRITICAL - This Week
### 🟡 HIGH - Next 2 Weeks
### 🟢 MEDIUM - Next Month

## Timeline Estimate
## Next Steps
## Conclusion
```

### 2. Category-Specific Gap Files (Required)

Update or create gap files in appropriate subfolder:
- `production-readiness/[category].md` for production gaps
- `file-organization/[issue-type].md` for file issues
- `cross-references/[issue-type].md` for link issues

### 3. Update gaps/README.md (Required)

Update the summary section:
```markdown
## Current Gaps Summary

**Last Check**: YYYY-MM-DD

### Production Readiness
- 🔴 X BLOCKER gaps (list categories)
- 🟡 X HIGH gaps

### File Organization
- 🟡 X files exceeding 500 lines
- 🔴 X files exceeding 1,000 lines

### Total Issues
- **Blockers**: X
- **High Priority**: X
```

---

## Quality Checklist for Gap Documentation

Before completing a consistency check session, verify:

- [ ] Timestamped report created in `gaps/reports/`
- [ ] All production blockers documented in `production-readiness/`
- [ ] All file issues documented in `file-organization/`
- [ ] All broken links documented in `cross-references/`
- [ ] Each gap file follows template format
- [ ] Each gap has priority level (🔴/🟡/🟢/⚪)
- [ ] Each gap has actionable "Action Items"
- [ ] gaps/README.md updated with latest metrics
- [ ] Cross-references between gap files added where relevant
- [ ] Timeline estimates provided for resolution

---

## Production Launch Blockers Format

In the timestamped report, include:

```markdown
## Production Launch Readiness: [STATUS]

**Status Options**: ✅ READY | ⚠️ NEEDS WORK | ⛔ BLOCKED

### Critical Gaps (Must resolve before production)
- [ ] Security & Authentication - [Link to gap doc]
- [ ] Deployment & Infrastructure - [Link to gap doc]
- [ ] Disaster Recovery - [Link to gap doc]

### High Priority (Should resolve before production)
- [ ] Monitoring & Observability - [Link to gap doc]
- [ ] API Documentation - [Link to gap doc]

### Medium Priority (Can defer with mitigation)
- [ ] File organization improvements
- [ ] Link cleanup

**Timeline to Production Ready**: X weeks
```

---

## Example Question Sequence

**Session Start:**
```
AI: We're performing a documentation consistency check for 
    /workspaces/CodeValdCortex/documents/

    I've scanned the folder structure. Starting with Technology Stack Check.

    🔍 [Technology Stack Check - Question 1]
    
    **Question**: Are there any files in 
    documents/2-SoftwareDesignAndArchitecture/ that reference "React" 
    (excluding archive/ folder)?
    
    **Context**: We migrated from React to Flutter for the frontend. Need 
    to verify no outdated references remain in active documentation.
    
    **What I'm Looking For**: File paths containing "React" references, 
    or confirmation that the area is clean.
```

**After First Answer:**
```
User: Yes, found in frontend-architecture.md and README.md

AI: ✅ Found 2 files with React references.
    
    🔍 Going deeper into frontend-architecture.md:
    
    **Question**: Is frontend-architecture.md a current document describing 
    React architecture, or a deprecation notice pointing to the Flutter plan?
    
    **Context**: Need to determine if this is already handled or needs archiving.
    
    **What I'm Looking For**: File content/purpose - is it outdated content 
    or a redirect notice?
```

**After Progress (Question 5):**
```
AI: 📊 **PROGRESS CHECK**
    
    **Completed Checks:**
    ✅ Technology Stack - 3 issues identified
    ✅ Cross-References - 1 broken link found
    
    **Issues Found:**
    📝 react-migration-plan.md - needs archiving
    📝 README.md - 2 React references to update
    📝 docs link broken - points to archived file
    
    **Next Area**: File Organization Analysis
    
    Continue with organization check, or would you like to review/address 
    issues first?
```

---

## Success Criteria

**Documentation Consistency:**
- ✅ Zero references to outdated technologies in active docs
- ✅ All archived files have clear deprecation notices
- ✅ No broken internal links
- ✅ Topics with 3+ files organized in subfolders
- ✅ No files exceed 1500 lines without justification
- ✅ All use case mvp.md files reflect current architecture
- ✅ Comprehensive consistency report generated

**Production Readiness:**
- ✅ Security documentation complete (auth, secrets, hardening)
- ✅ Monitoring & alerting documented with SLIs/SLOs
- ✅ Deployment procedures and DR plans documented
- ✅ Data management and compliance requirements addressed
- ✅ API documentation production-ready (OpenAPI/Swagger)
- ✅ Testing coverage and quality gates defined
- ✅ Operational runbooks and troubleshooting guides exist
- ✅ No critical blockers for production deployment
- ✅ Production readiness checklist 100% complete
