---
agent: agent
---

# Research & Documentation Gap Analysis Prompt

## Purpose
This prompt guides a structured Q&A session to explore and complete documentation for any feature or architectural component in the **CodeValdSharedLib** library through **one question at a time**, allowing for deep dives into specific topics.

---

## 🔄 MANDATORY REFACTOR WORKFLOW (ENFORCE BEFORE ANY RESEARCH SESSION)

**BEFORE starting any research or writing new task documentation:**

### Step 1: CHECK File Size
```bash
wc -l documentation/3-SofwareDevelopment/mvp-details/{topic-file}.md
```

### Step 2: IF >500 lines OR individual MVP-XXX.md files exist:

**a. CREATE folder structure:**
```bash
documentation/3-SofwareDevelopment/mvp-details/{domain-name}/
├── README.md              # Domain overview, architecture, task index (MAX 300 lines)
├── {topic-1}.md           # Topic-based grouping of related tasks (MAX 500 lines)
├── {topic-2}.md           # Topic-based grouping of related tasks (MAX 500 lines)
├── architecture/          # Optional: detailed technical designs
│   ├── flow-diagrams.md
│   ├── data-models.md
│   └── state-machines.md
└── examples/              # Optional: code samples, configs
    ├── sample-configs.yaml
    └── api-examples.md
```

**b. CREATE README.md** with:
- Domain overview
- Architecture summary
- Task index with links

**c. SPLIT content by TOPIC (NOT by task ID):**
- Group related tasks into topic files
- Examples: `webhooks.md`, `authentication.md`, `state-machines.md`, `instance-management.md`

**d. MOVE architecture diagrams** → `architecture/` subfolder

**e. MOVE examples** → `examples/` subfolder

### Step 3: ONLY THEN add new task content to appropriate topic file

---

## 🛑 STOP CONDITIONS (Do NOT proceed until fixed)

- ❌ **Domain file exceeds 500 lines** → **MUST refactor first**
- ❌ **README.md exceeds 300 lines** → **MUST split content**
- ❌ **Individual `MVP-XXX.md` files exist** → **MUST consolidate by topic**
- ❌ **Task file exceeds 200 lines** → **MUST split into subtopics**

### Find Your Task's Domain

- **Small domains** (2-4 tasks, <500 lines): Single file like `authentication.md`
- **Large domains** (5+ tasks OR >500 lines): MUST use folder structure

### Current Status Check Required

Before starting research, AI must:
1. Check current file line count
2. List existing files in domain
3. Determine if refactor is needed
4. Execute refactor if triggered
5. ONLY THEN begin research questions

---

## Instructions for AI Assistant

Conduct a comprehensive documentation gap analysis through **iterative single-question exploration**. Ask ONE question at a time, wait for the response, then decide whether to:
- **Go Deeper**: Ask follow-up questions on the same topic
- **Take Note**: Record a gap for later exploration
- **Move On**: Proceed to the next topic area
- **Review**: Summarize what we've learned and identify remaining gaps

The goal is to have focused conversations that build understanding incrementally rather than overwhelming with multiple questions.

## Research Framework

### 1. Session Initiation
When starting a research session:
1. **State the feature/component** being researched
2. **Scan existing documentation** and code quickly
3. **Ask the first question** from the most critical area
4. **Wait for response** before proceeding

### 2. Question Flow

**After each answer, explicitly choose one of these paths:**

- 🔍 **DEEPER**: "Let me dig deeper into [specific aspect]..."
  - Ask follow-up question on the same topic
  - Explore edge cases or implementation details
  - Clarify ambiguities in the response

- 📝 **NOTE**: "I'll note this gap: [description]..."
  - Record incomplete information for later
  - Mark areas needing further research
  - Continue to different topic

- ➡️ **NEXT**: "Moving to [new topic area]..."
  - Sufficient understanding achieved
  - Proceed to different question category
  - Keep momentum going

- 📊 **REVIEW**: "Let me summarize what we've covered..."
  - List topics explored
  - Identify remaining gaps
  - Decide next focus area together

### 3. Question Categories (Priority Order)

#### Architecture & Design
**Start here for new features**
- What is the high-level architecture of this feature?
- How does it integrate with existing systems?
- What are the key design patterns used?
- What are the data flows and dependencies?
- What are the scalability considerations?

#### Data Models
**Critical for database-backed features**
- What data structures are involved?
- What database collections/tables are used?
- What are the relationships between entities?
- What indexes or optimizations are needed?
- What is the data lifecycle (creation, updates, deletion)?

#### Business Logic
**Core functionality exploration**
- What are the core use cases?
- What are the business rules and constraints?
- What are the edge cases and error scenarios?
- What validation is required?
- What are the state transitions (if applicable)?

#### API & Interfaces
**For external-facing features**
- What endpoints/interfaces are exposed?
- What are the request/response formats?
- What authentication/authorization is needed?
- What are the rate limits or quotas?
- What error codes and messages are used?

#### User Experience
**For UI-heavy features**
- What React components are involved?
- What is the user workflow?
- What feedback mechanisms exist?
- What accessibility considerations apply?
- What performance expectations exist?

#### Testing & Quality
**For ensuring reliability**
- What testing strategies are used?
- What are the test coverage targets?
- What integration points need testing?
- What are the acceptance criteria?
- What monitoring/observability exists?

#### Operations & Deployment
**For production readiness**
- How is this feature deployed?
- What configuration is required?
- What are the resource requirements?
- What backup/recovery mechanisms exist?
- What are the rollback procedures?

#### Security & Compliance
**For sensitive features**
- What security measures are in place?
- What data privacy concerns exist?
- What compliance requirements apply?
- What audit trails are maintained?
- What encryption/protection is used?

### 4. Single Question Format

**Structure each question clearly:**

```
🔍 [Category: Architecture/Data/Business Logic/etc.]

**Question**: [Your specific question - keep it focused and singular]

**Context**: [1-2 sentences on why this matters or what you've observed]

**What I'm Looking For**: [Expected type of answer - e.g., "specific workflow steps", "data structure", "yes/no with rationale"]
```

### 5. Response Processing

**After receiving each answer:**

1. **Acknowledge**: Briefly confirm understanding
2. **Decide Path**: Choose DEEPER/NOTE/NEXT/REVIEW
3. **State Choice Explicitly**: "🔍 Going deeper..." or "📝 Noting gap..." or "➡️ Moving to..."
4. **Ask Next Question** (or provide review if chosen)

**Example Flow:**
```
AI: 🔍 [Architecture]
    Question: How does instance isolation work at runtime?
    Context: Multiple instances from same tag need separate state.
    What I'm Looking For: Specific mechanisms or data structures used.

User: [Provides answer about instance runtime tracking]

AI: ✅ Got it - using instance-scoped agent pools.
    🔍 Going deeper: How are agent identities namespaced per instance?

User: [Explains agent ID prefixing]

AI: ✅ Clear on agent namespacing.
    📝 Noting gap: Need to document collision prevention strategy.
    ➡️ Moving to lifecycle management: What triggers instance state transitions?
```

## Gap Tracking System

**Maintain a running list of gaps during the session:**

### Identified Gaps
- 📝 **[Topic Area]**: Brief description of what's unclear or missing
- 📝 **[Topic Area]**: Another gap found
- 📝 **[Topic Area]**: Third gap

### Explored Topics
- ✅ **[Topic Area]**: Sufficient understanding achieved
- ✅ **[Topic Area]**: Documented and clear

### Deep Dive Areas
- 🔍 **[Topic Area]**: Currently exploring in depth
- ⏸️ **[Topic Area]**: Paused for later (if needed)

**Update this list visibly every 3-5 questions** to maintain awareness of progress.

## Conversation Pacing

**Keep momentum while allowing depth:**

- ⏱️ **One question at a time** - No multi-question lists
- 🎯 **Stay focused** - Don't switch topics mid-exploration
- 🔄 **Iterate deliberately** - Each answer informs the next question
- 📊 **Review periodically** - Every 5-7 questions, check progress
- 🎬 **User-driven pivots** - User can request topic changes anytime

## Red Flags to Watch For

When exploring a topic, call out these issues immediately:

- 🚩 **Inconsistencies**: "I notice the code says X but docs say Y..."
- 🚩 **Assumptions**: "This seems to assume Z - is that documented?"
- 🚩 **Missing Context**: "Why was this approach chosen over alternatives?"
- 🚩 **Incomplete Specs**: "We know A and B, but what about C?"
- 🚩 **Outdated Info**: "The code changed but docs didn't - which is correct?"
- 🚩 **Vague Language**: "What specifically does 'handle' mean here?"

**When flagged, immediately ask clarifying question before continuing.**

## Example Question Sequence

**Session Start:**
```
AI: We're researching Agency Instance Management (MVP-PUB-007).
    I've scanned the docs and code. Starting with critical architecture.

    🔍 [Architecture & Design]
    
    **Question**: What's the core isolation mechanism between instances 
    running from the same tag?
    
    **Context**: Multiple instances need independent runtime state while 
    sharing immutable tag configuration.
    
    **What I'm Looking For**: Specific data structures or namespacing approach.
```

**After First Answer:**
```
User: Each instance has unique instance_id, agents are spawned with 
      instance-scoped IDs, separate runtime tracking in lifecycle manager.

AI: ✅ Got it - isolation via instance_id namespacing and separate tracking.
    
    🔍 Going deeper into agent spawning:
    
    **Question**: How exactly are agent IDs formatted to prevent collisions 
    between instances?
    
    **Context**: Need to ensure agent-1 from instance-A doesn't conflict 
    with agent-1 from instance-B.
    
    **What I'm Looking For**: ID format pattern or generation algorithm.
```

**After Second Answer:**
```
User: Agents get UUIDs, not sequential IDs. Each spawn creates unique ID.

AI: ✅ Clear - UUIDs eliminate collision risk.
    📝 Noting gap: Should document UUID format in data model docs.
    
    ➡️ Moving to lifecycle management:
    
    **Question**: What triggers the transition from "starting" to "running" 
    state?
    
    **Context**: Need to understand state machine for monitoring.
    
    **What I'm Looking For**: Specific condition or health check.
```

**Periodic Review (after 6 questions):**
```
AI: 📊 **REVIEW - Progress Check**
    
    **Explored Topics:**
    ✅ Instance isolation mechanism (instance_id namespacing)
    ✅ Agent ID collision prevention (UUIDs)
    ✅ State transitions (starting→running→stopped)
    ✅ Data persistence (agency_instances collection)
    
    **Noted Gaps:**
    📝 UUID format documentation in data models
    📝 Error recovery when agent spawn fails
    📝 Instance cleanup on agency deletion
    
    **Remaining Areas:**
    - Multi-instance concurrency limits
    - Health monitoring implementation
    - Tag immutability enforcement
    - UI workflow details
    
    **Next Steps**: Continue with health monitoring, or dive into a gap?
```

## Completion Criteria

**The research session is complete when:**

- ✅ All critical question categories have been explored
- ✅ Major architectural components are understood
- ✅ Data flows and state machines are clear
- ✅ Edge cases and error scenarios are identified
- ✅ No blocking gaps remain (minor gaps noted for later)
- ✅ User confirms readiness to conclude
- ✅ Documentation can be written without major unknowns

**Final Deliverable:**

After session completion, provide a structured summary:

```markdown
# Research Summary: [Feature Name]

## Topics Explored (with confidence level)
1. **Architecture** ⭐⭐⭐⭐⭐ Fully understood
2. **Data Models** ⭐⭐⭐⭐ Well understood, minor gaps
3. **Business Logic** ⭐⭐⭐ Moderate understanding
4. **API Design** ⭐⭐⭐⭐ Well understood
5. **User Experience** ⭐⭐ Needs more research

## Key Findings
- [Bullet point of important discovery]
- [Another key finding]

## Documented Gaps (For Future Work)
1. **[Gap Category]**: Description and why it matters
2. **[Gap Category]**: Description and why it matters

## Action Items
- [ ] Document [specific aspect] in [location]
- [ ] Implement [missing feature]
- [ ] Update [existing doc] with [new information]

## Ready for Implementation?
- [Yes/No with justification]
```

## Usage Instructions

**To start a research session:**

1. **Specify the feature/component**: "Let's research [feature name]"
2. **AI scans context**: Reviews docs, code, and existing knowledge
3. **AI asks first question**: Single focused question from critical area
4. **Answer the question**: Provide as much or little detail as you want
5. **AI responds with path**: DEEPER/NOTE/NEXT/REVIEW + next question
6. **Continue dialogue**: One question at a time until complete
7. **Request review anytime**: Say "review progress" to see summary
8. **Redirect as needed**: Say "switch to [topic]" to change focus
9. **Conclude when satisfied**: Say "wrap up" for final summary

**Session Control Commands:**

- 💬 `"review progress"` - Show gap tracking summary
- 💬 `"switch to [topic]"` - Change focus area
- 💬 `"go deeper"` - Continue current topic exploration
- 💬 `"skip this"` - Note as gap and move on
- 💬 `"wrap up"` - Conclude session with summary
- 💬 `"start over"` - Reset and begin new research thread

## Best Practices

**For Effective Research:**

- ✅ **Be conversational** - It's okay to say "I'm not sure" or "Let me check"
- ✅ **Ask for clarification** - If question is unclear, ask AI to rephrase
- ✅ **Provide context** - Share why something works a certain way
- ✅ **Reference code/docs** - Point to specific files when helpful
- ✅ **Admit gaps honestly** - Unknown areas become research opportunities
- ✅ **Take breaks** - Use REVIEW to pause and resume later
- ✅ **Drive the pace** - Tell AI to speed up or slow down

**For AI Assistant:**

- ✅ **Stay focused** - One question, wait for answer, process, next question
- ✅ **Be explicit** - Always state your chosen path (DEEPER/NOTE/NEXT/REVIEW)
- ✅ **Track visibly** - Update gap list every few questions
- ✅ **Adapt dynamically** - Follow interesting threads that emerge
- ✅ **Synthesize periodically** - Connect dots between topics
- ✅ **Respect context** - Don't ask what's already clearly documented
- ✅ **Flag inconsistencies** - Call out conflicts immediately

---

## Example Session Initiation

```
User: Let's research instance management for MVP-PUB-007

AI: 🔍 **Pre-Research: File Size Check**
    
    Checking documentation status...
    $ wc -l documentation/3-SofwareDevelopment/mvp-details/{topic-file}.md
    884 lines
    
    🛑 STOP CONDITION TRIGGERED: File exceeds 500 lines!
    
    **Action Required**: Refactor before research
    
    Current structure:
    - instance-management.md (884 lines) ❌ EXCEEDS LIMIT
    - state-machine.md (779 lines) ❌ EXCEEDS LIMIT
    - README.md (324 lines) ✅ OK
    - integration-testing.md (304 lines) ✅ OK
    
    **Refactor Plan**:
    1. Split instance-management.md into:
       - data-models.md (models, database schema)
       - instance-lifecycle.md (service layer, state management)
       - instance-ui.md (React components, API handlers)
       - instance-dashboard.md (dashboard components)
    
    2. Split state-machine.md into:
       - publishing-states.md (agency lifecycle states)
       - tag-system.md (tag creation, restoration)
    
    Proceeding with refactor...
    
    [After refactor completes]
    
    ✅ **Refactor Complete - Starting Research Session**
    
    🔍 **Starting Research: Agency Instance Management (MVP-PUB-007)**
    
    Quick scan shows:
    - Data models documented in data-models.md
    - Service layer in instance-lifecycle.md
    - React components in instance-ui.md
    - Dashboard components in instance-dashboard.md
    
    Starting with most critical aspect:
    
    🔍 [Architecture & Design]
    
    **Question 1**: When an instance transitions from "starting" to "running", 
    what specific criteria should be met?
    
    **Context**: The documentation defines instance states but doesn't specify 
    the exact transition triggers.
    
    **What I'm Looking For**: Should ALL agents be successfully spawned, or 
    is there a threshold (e.g., 80% of agents) that would be acceptable?

[User answers, session continues...]
```

---

**Remember**: This is a **collaborative exploration**, not an interrogation. The goal is shared understanding through focused, iterative dialogue.
