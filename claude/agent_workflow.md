# Agent Workflow & Coordination Guide

> Orchestration guide for the 7 specialized Claude agents in the goimg-datalayer project.
> **Purpose**: Ensure efficient multi-agent collaboration, clear task assignments, and quality delivery.

---

## Table of Contents

1. [Agent Roster](#agent-roster)
2. [Task Assignment Protocol](#task-assignment-protocol)
3. [Agent Handoff Process](#agent-handoff-process)
4. [Collaboration Patterns](#collaboration-patterns)
5. [Sprint Integration](#sprint-integration)
6. [Communication Protocol](#communication-protocol)
7. [Quality Gates](#quality-gates)
8. [Escalation & Conflict Resolution](#escalation--conflict-resolution)

---

## Agent Roster

### 1. Scrum Master
**Agent**: `scrum-master`
**Primary Responsibilities**:
- Sprint planning and backlog grooming
- Agent task assignment and oversight
- Work quality verification
- Sprint ceremonies facilitation
- Retrospectives and improvement tracking

**When to Engage**:
- Sprint planning sessions
- Backlog prioritization
- Validating completed agent work
- Multi-agent workflow coordination
- Sprint retrospectives

**Availability**: Continuous (orchestration role)

---

### 2. Backend Test Architect
**Agent**: `backend-test-architect`
**Primary Responsibilities**:
- Unit, integration, and system test design
- Test coverage analysis and gap identification
- Debugging flaky tests
- Test infrastructure setup
- Test performance optimization

**When to Engage**:
- New feature implementation requiring test coverage
- Reviewing existing test quality
- Test failures or flakiness issues
- Achieving coverage requirements (80% overall, 90% domain)
- Setting up test fixtures

**Availability**: On-demand (per feature/component)

---

### 3. Test Strategist
**Agent**: `test-strategist`
**Primary Responsibilities**:
- E2E test strategy and design
- Postman/Newman collection creation
- API contract testing (OpenAPI compliance)
- Edge case identification
- Regression test planning

**When to Engage**:
- Designing comprehensive test strategies
- Creating Postman collections for API testing
- Boundary condition analysis
- Release regression testing
- Contract testing setup

**Availability**: On-demand (checkpoint-based)

---

### 4. Senior Go Architect
**Agent**: `senior-go-architect`
**Primary Responsibilities**:
- Library selection and research
- Design pattern implementation
- Architecture decisions
- Code review (performance, patterns)
- Minimalist approach advocacy

**When to Engage**:
- Evaluating library choices
- Designing new modules/features
- Architectural trade-off decisions
- Performance optimization
- Code review for Go idioms

**Availability**: Checkpoint (Sprint 1-2, 3, 5, 8)

---

### 5. Senior SecOps Engineer
**Agent**: `senior-secops-engineer`
**Primary Responsibilities**:
- Security controls implementation
- Vulnerability assessment and remediation
- Compliance validation (OWASP Top 10)
- Security code review
- Security gates approval

**When to Engage**:
- Implementing authentication/authorization
- Security vulnerability triage
- Sprint security gate reviews
- Security tooling setup (SAST, DAST)
- Compliance requirements

**Availability**: Checkpoint (security gates at sprint boundaries)

---

### 6. CI/CD Guardian
**Agent**: `cicd-guardian`
**Primary Responsibilities**:
- GitHub Actions workflow creation/debugging
- Docker configuration
- CI/CD pipeline health monitoring
- Infrastructure configuration
- Main branch protection

**When to Engage**:
- CI/CD pipeline failures
- Setting up new workflows
- Docker build optimization
- Deployment configuration
- Pipeline performance issues

**Availability**: Continuous (reactive to failures) + Checkpoint (Sprint 1, 4, 8)

---

### 7. Image Gallery Expert
**Agent**: `image-gallery-expert`
**Primary Responsibilities**:
- Feature analysis (Flickr/Chevereto)
- Product roadmap planning
- Competitive research
- UX pattern recommendations
- Feature prioritization

**When to Engage**:
- Sprint planning sessions
- Feature discovery and analysis
- MVP scope definition
- Post-MVP roadmap planning
- Understanding industry best practices

**Availability**: Checkpoint (Sprint planning, feature discovery)

---

## Task Assignment Protocol

### Assignment Template

```markdown
## Task Assignment

**Agent**: [agent-name]
**Assigned By**: [scrum-master/user]
**Sprint**: [Sprint N]
**Priority**: [P0/P1/P2/P3]

### Context
[Background and motivation for this task]

### Task Description
[Clear, actionable description of what needs to be done]

### Dependencies
- [ ] Dependency 1 (status: completed/in-progress/blocked)
- [ ] Dependency 2

### Definition of Done
- [ ] Acceptance criteria 1
- [ ] Acceptance criteria 2
- [ ] Tests written (unit/integration/e2e)
- [ ] Documentation updated
- [ ] Code review completed
- [ ] Security review (if applicable)
- [ ] OpenAPI spec updated (if HTTP changes)

### Review Requirements
**Code Review**: [yes/no] by [agent-name]
**Security Review**: [yes/no] by senior-secops-engineer
**Test Review**: [yes/no] by backend-test-architect

### Resources
- Related files: [file paths]
- Documentation: [links to guides]
- Context guides: [claude/*.md references]
```

### Assignment Decision Matrix

| Task Type | Primary Agent | Supporting Agents | Review By |
|-----------|---------------|-------------------|-----------|
| **Domain entity implementation** | N/A (direct) | senior-go-architect | scrum-master |
| **Repository implementation** | N/A (direct) | backend-test-architect | senior-go-architect |
| **HTTP handler creation** | N/A (direct) | test-strategist | scrum-master |
| **Security implementation** | senior-secops-engineer | senior-go-architect | scrum-master |
| **Test suite creation** | backend-test-architect | test-strategist | scrum-master |
| **API contract testing** | test-strategist | N/A | backend-test-architect |
| **CI/CD setup/fix** | cicd-guardian | N/A | scrum-master |
| **Library selection** | senior-go-architect | senior-secops-engineer | scrum-master |
| **Feature planning** | image-gallery-expert | scrum-master | N/A |
| **Security gate review** | senior-secops-engineer | N/A | scrum-master |

---

## Agent Handoff Process

### Work-in-Progress Check-in Format

When an agent is working on a task, use this format for progress updates:

```markdown
## WIP Update: [Task Name]

**Agent**: [agent-name]
**Status**: [In Progress / Blocked / Under Review]
**Progress**: [X]% complete

### Completed
- [x] Item 1
- [x] Item 2

### In Progress
- [ ] Item 3 (estimated completion: [timeframe])

### Blockers
- Blocker description [requires: agent-name / external dependency]

### Next Steps
1. Next action
2. Next action

### Handoff Readiness
- [ ] Code committed to branch
- [ ] Tests passing locally
- [ ] Documentation updated
- [ ] Ready for review
```

### Pre-Merge Review Process

**Phase 1: Self-Review** (Agent completing work)
1. Run checklist from `/home/user/goimg-datalayer/claude/agent_checklist.md`
2. Verify all acceptance criteria met
3. Confirm tests pass: `go test -race ./...`
4. Validate OpenAPI if HTTP changes: `make validate-openapi`
5. Check linting: `golangci-lint run`

**Phase 2: Peer Review** (If specified in assignment)
1. Scrum master assigns review to appropriate agent
2. Reviewer validates:
   - Code quality and adherence to standards (`claude/coding.md`)
   - Test coverage meets requirements
   - No architecture violations (DDD layering)
   - Security considerations addressed
3. Reviewer provides feedback in structured format:
   ```markdown
   ## Review: [Task Name]
   **Reviewer**: [agent-name]
   **Status**: [Approved / Changes Requested / Rejected]

   ### Strengths
   - Positive observation 1

   ### Issues
   - [ ] Issue 1 (severity: critical/major/minor)
   - [ ] Issue 2

   ### Recommendations
   - Suggestion 1
   ```

**Phase 3: Quality Gate** (Scrum master final validation)
1. Verify all review feedback addressed
2. Confirm agent checklist completion
3. Check for cross-cutting concerns
4. Approve merge or request additional work

### Knowledge Transfer After Completion

When work is complete, the agent should document:

```markdown
## Knowledge Transfer: [Task Name]

**Agent**: [agent-name]
**Completed**: [date]

### Summary
[Brief description of what was accomplished]

### Key Decisions
- Decision 1: [rationale]
- Decision 2: [rationale]

### Technical Debt Identified
- [ ] Debt item 1 (priority: P0/P1/P2/P3)
- [ ] Debt item 2

### Follow-up Tasks
- [ ] Task 1 [assign to: agent-name]
- [ ] Task 2

### Documentation Updated
- File: [path] (section: [name])
- File: [path] (section: [name])

### Lessons Learned
- Lesson 1
- Lesson 2
```

---

## Collaboration Patterns

### Pattern 1: Security + Architecture Review

**Use Case**: Implementing authentication, authorization, or cryptographic features

**Workflow**:
```
┌─────────────────────┐
│  1. Design Phase    │  Agents: senior-go-architect + senior-secops-engineer
│  Library research   │  Output: Architecture decision record
│  Security analysis  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  2. Implementation  │  Agent: N/A (direct implementation)
│  Code + tests       │  Context: Architecture + security recommendations
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  3. Security Review │  Agent: senior-secops-engineer
│  Vulnerability scan │  Output: Security approval or remediation items
│  Code review        │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  4. Final Approval  │  Agent: scrum-master
│  Quality gate check │  Output: Merge approval
└─────────────────────┘
```

**Example**: Sprint 3 - JWT implementation
1. senior-go-architect researches JWT libraries (golang-jwt/jwt vs alternatives)
2. senior-secops-engineer validates RS256 requirement, key size, rotation strategy
3. Implementation occurs with combined recommendations
4. senior-secops-engineer reviews for security gate S3-AUTH-001 through S3-AUTH-006
5. scrum-master validates completeness and approves

---

### Pattern 2: Feature Implementation Flow

**Use Case**: Adding a new feature end-to-end

**Workflow**:
```
┌─────────────────────┐
│  1. Feature         │  Agent: image-gallery-expert
│     Analysis        │  Output: Feature requirements, acceptance criteria
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  2. Sprint Planning │  Agent: scrum-master
│     Task breakdown  │  Output: Task assignments with DoD
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  3. Domain Logic    │  Agent: N/A (direct)
│     Implementation  │  Supporting: senior-go-architect (as needed)
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  4. Test Coverage   │  Agent: backend-test-architect
│     Unit tests      │  Output: Comprehensive test suite
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  5. HTTP Layer      │  Agent: N/A (direct)
│     API endpoints   │  Context: OpenAPI spec
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  6. E2E Testing     │  Agent: test-strategist
│     Postman suite   │  Output: API contract tests
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  7. CI/CD Verify    │  Agent: cicd-guardian (if failures)
│     Pipeline check  │  Output: Green build
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  8. Final Review    │  Agent: scrum-master
│     Quality gates   │  Output: Merge approval
└─────────────────────┘
```

**Example**: Sprint 6 - Image upload feature
1. image-gallery-expert analyzes Flickr/Chevereto upload UX patterns
2. scrum-master breaks down into: domain (Image aggregate), infrastructure (storage provider), application (UploadImageCommand), HTTP (upload handler)
3. Implementation with senior-go-architect guidance on library choices (bimg, AWS SDK)
4. backend-test-architect creates unit tests for domain + integration tests for storage
5. test-strategist creates Postman collection for upload endpoint
6. cicd-guardian ensures CI pipeline handles Docker image processing dependencies
7. scrum-master validates against Sprint 6 checklist

---

### Pattern 3: Testing Workflow

**Use Case**: Comprehensive test coverage for a component

**Workflow**:
```
┌─────────────────────┐
│  1. Test Strategy   │  Agent: test-strategist
│     Design          │  Output: Test plan with edge cases
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  2. Unit Tests      │  Agent: backend-test-architect
│     Implementation  │  Output: Table-driven tests, mocks
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  3. Integration     │  Agent: backend-test-architect
│     Tests           │  Output: Testcontainer-based tests
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  4. E2E Tests       │  Agent: test-strategist
│     Postman/Newman  │  Output: API workflow tests
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  5. Coverage Review │  Agent: scrum-master
│     Validation      │  Output: Coverage report, gap analysis
└─────────────────────┘
```

**Example**: Sprint 4 - Auth handlers
1. test-strategist identifies edge cases: invalid credentials, expired tokens, token replay
2. backend-test-architect writes unit tests for application layer (LoginCommand)
3. backend-test-architect writes integration tests for session repository
4. test-strategist creates Postman collection with auth flow + negative cases
5. scrum-master validates 85%+ coverage for application layer

---

### Pattern 4: CI/CD Failure Response

**Use Case**: Pipeline breaks, needs rapid diagnosis and fix

**Workflow**:
```
┌─────────────────────┐
│  1. Failure         │  Trigger: GitHub Actions failure
│     Detection       │  Notifies: cicd-guardian (auto-engaged)
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  2. Diagnosis       │  Agent: cicd-guardian
│     Root cause      │  Output: Infrastructure vs code issue
└──────────┬──────────┘
           │
       ┌───┴────┐
       ▼        ▼
┌──────────┐  ┌──────────────────┐
│ Infra    │  │ Code Issue       │
│ Issue    │  │ (delegate)       │
└────┬─────┘  └────┬─────────────┘
     │             │
     ▼             ▼
┌──────────┐  ┌──────────────────┐
│ cicd-    │  │ Appropriate      │
│ guardian │  │ agent fixes code │
│ fixes    │  │                  │
└────┬─────┘  └────┬─────────────┘
     │             │
     └─────┬───────┘
           ▼
┌─────────────────────┐
│  3. Verification    │  Agent: cicd-guardian
│     Re-run workflow │  Output: Green build confirmation
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  4. Prevention      │  Agent: cicd-guardian
│     Document fix    │  Output: Updated workflow or runbook
└─────────────────────┘
```

**Example**: Linting failure after merge
1. cicd-guardian detects golangci-lint failure
2. Analyzes logs: "gosec G401: weak cryptographic primitive (MD5)"
3. Identifies code issue (not infrastructure)
4. scrum-master delegates to senior-secops-engineer
5. senior-secops-engineer replaces MD5 with SHA-256, adds security test
6. cicd-guardian verifies green build
7. cicd-guardian documents in security runbook

---

## Sprint Integration

### Agent Assignments by Sprint

| Sprint | Primary Agents | Checkpoint Agents | Deliverables |
|--------|----------------|-------------------|--------------|
| **1-2: Foundation** | scrum-master, senior-go-architect, cicd-guardian | senior-secops-engineer | Project structure, OpenAPI spec, CI/CD setup |
| **3: Infra - Identity** | senior-secops-engineer, backend-test-architect | senior-go-architect | JWT, sessions, database, security gate |
| **4: App - Identity** | backend-test-architect, test-strategist | senior-secops-engineer | Auth handlers, Postman tests, security validation |
| **5: Infra - Gallery** | senior-go-architect, backend-test-architect | senior-secops-engineer | Storage providers, image processing, ClamAV |
| **6: App - Gallery** | backend-test-architect, test-strategist, image-gallery-expert | scrum-master | Upload flow, albums, tags, E2E tests |
| **7: Moderation** | senior-secops-engineer, backend-test-architect | senior-go-architect | RBAC, audit logging, admin tools |
| **8: Testing & Security** | backend-test-architect, test-strategist, senior-secops-engineer | cicd-guardian | Full test suite, security hardening, performance |
| **9: MVP Polish** | cicd-guardian, scrum-master | All agents | Deployment, monitoring, launch checklist |

### Agent Checkpoint Schedule

#### Sprint Planning (Start of Sprint)
**Attendees**: scrum-master (lead), image-gallery-expert (features), senior-go-architect (technical feasibility)

**Agenda**:
1. Review sprint goal alignment with MVP roadmap
2. Break down user stories into technical tasks
3. Assign tasks with clear DoD
4. Identify dependencies and risks
5. Estimate capacity and confirm achievability

**Output**: Sprint backlog with agent assignments

---

#### Mid-Sprint Checkpoint (Day 5-7 of 2-week sprint)
**Attendees**: scrum-master (lead), active agents for current sprint

**Agenda**:
1. Review progress toward sprint goal (burndown)
2. Identify blockers and coordinate resolution
3. Adjust task assignments if needed
4. Preview upcoming work for next phase

**Output**: Blocker resolution plan, adjusted assignments

---

#### Pre-Merge Quality Gate (Per major feature)
**Attendees**: scrum-master (lead), relevant reviewers (e.g., senior-secops-engineer for auth)

**Agenda**:
1. Code review completion status
2. Test coverage validation
3. Security checklist (if applicable)
4. OpenAPI spec alignment
5. Agent checklist verification

**Output**: Merge approval or remediation items

---

#### Sprint Retrospective (End of Sprint)
**Attendees**: scrum-master (facilitator), all active agents from sprint

**Format**: Start/Stop/Continue

**Agenda**:
1. What went well? (celebrate wins)
2. What didn't go well? (identify pain points)
3. What should we change? (actionable improvements)
4. Review previous retrospective action items

**Output**: Improvement backlog with owners and due dates

**Template**:
```markdown
## Sprint [N] Retrospective

**Date**: [date]
**Participants**: [agent list]
**Format**: Start/Stop/Continue

### Start (New practices to adopt)
- [ ] Action 1 [owner: agent-name] [due: Sprint N+1]
- [ ] Action 2

### Stop (Practices to eliminate)
- [ ] Action 1 [owner: agent-name] [due: Sprint N+1]
- [ ] Action 2

### Continue (Effective practices to maintain)
- Practice 1
- Practice 2

### Previous Retro Follow-up
- [x] Action from Sprint N-1: Completed
- [ ] Action from Sprint N-1: In progress (carry forward)

### Metrics
- Velocity: [X] points (planned: [Y])
- Completion rate: [Z]%
- Defect count: [N]
- Test coverage: [X]%

### Key Insights
- Insight 1
- Insight 2
```

---

## Communication Protocol

### Progress Reporting

**Daily Stand-up Format** (for active agents):
```markdown
### [Agent Name] - [Date]

**Yesterday**:
- Completed: [task description]
- Progress: [task description] ([X]% complete)

**Today**:
- Plan: [task description]

**Blockers**:
- [Blocker description] [requires: agent/resource]
```

**Sprint Summary Format** (weekly):
```markdown
## Sprint [N] Summary - Week [X]

**Sprint Goal**: [goal statement]
**Status**: [On Track / At Risk / Off Track]

### Completed This Week
- [x] Task 1 [agent: name]
- [x] Task 2 [agent: name]

### In Progress
- [ ] Task 3 [agent: name] ([X]% complete)
- [ ] Task 4 [agent: name] ([X]% complete)

### Blocked
- [ ] Task 5 [blocker: description] [requires: resource]

### Risks
- Risk 1: [description] [impact: High/Medium/Low] [mitigation: plan]

### Metrics
- Sprint progress: [X]% complete
- Test coverage: [Y]%
- Velocity: [Z] points completed
```

---

### Escalation Paths

#### Blocker Escalation

**Level 1: Agent Self-Resolution** (0-4 hours)
- Agent attempts to resolve blocker independently
- Consults documentation, researches solutions

**Level 2: Peer Agent Assistance** (4-24 hours)
- Scrum master assigns supporting agent
- Collaborative problem-solving session

**Level 3: Scrum Master Escalation** (24-48 hours)
- Scrum master re-prioritizes work
- Considers task reassignment or scope adjustment

**Level 4: Stakeholder Escalation** (48+ hours)
- Critical path impact
- Requires external resources or decisions

---

#### Technical Disagreement Resolution

**Scenario**: Two agents have conflicting recommendations

**Process**:
1. **Document Positions**: Each agent writes up their recommendation with:
   - Proposed solution
   - Rationale (pros/cons)
   - Risks and trade-offs
   - Supporting evidence

2. **Scrum Master Facilitation**:
   - Review both positions
   - Identify decision criteria (performance, security, maintainability, etc.)
   - Weight criteria by project priorities

3. **Architect Review** (if needed):
   - Engage senior-go-architect as tie-breaker
   - Provide architectural context and long-term considerations

4. **Decision Documentation**:
   - Record final decision
   - Explain rationale
   - Document alternative considered
   - File as Architecture Decision Record (ADR)

**Example**: Choice between GORM vs sqlx
- senior-go-architect recommends sqlx (minimalism, performance)
- Alternative view might prefer GORM (productivity, migrations)
- Scrum master weights: performance > productivity (image gallery is data-intensive)
- Decision: sqlx, documented in `docs/adr/001-database-library.md`

---

### Conflict Resolution

**Scenario**: Agent work overlaps or conflicts

**Process**:
1. **Identify Conflict**: Scrum master detects conflicting changes or assignments
2. **Pause Work**: Temporarily halt conflicting activities
3. **Alignment Session**:
   - Bring agents together
   - Clarify task boundaries
   - Identify integration points
4. **Adjust Assignments**: Scrum master revises task definitions to eliminate overlap
5. **Resume Work**: Agents proceed with clarified scope

---

## Quality Gates

### Pre-Sprint Planning Gate

**Owner**: scrum-master
**Trigger**: Before sprint planning meeting

**Checklist**:
- [ ] Previous sprint retrospective actions completed or carried forward
- [ ] Backlog items refined and estimated
- [ ] Dependencies from previous sprint resolved
- [ ] Team capacity calculated (account for PTO, meetings, tech debt)
- [ ] Sprint goal draft aligns with MVP roadmap

**Pass Criteria**: All items checked
**Fail Action**: Delay sprint start until checklist complete

---

### Mid-Sprint Checkpoint Gate

**Owner**: scrum-master
**Trigger**: Day 5-7 of 2-week sprint

**Checklist**:
- [ ] Sprint burndown on track (within 10% of ideal)
- [ ] No critical blockers unresolved for >24 hours
- [ ] Test coverage maintained (no regression)
- [ ] CI/CD pipeline green (main branch)
- [ ] Work-in-progress within limits (max 3 tasks per agent)

**Pass Criteria**: All items checked or mitigation plan in place
**Fail Action**: Scrum master adjusts sprint scope or resources

---

### Pre-Merge Quality Gate

**Owner**: scrum-master
**Trigger**: Before merging feature branch to main

**Checklist** (from `claude/agent_checklist.md`):
- [ ] Code Quality
  - `go fmt ./...` passes
  - `go vet ./...` passes
  - `golangci-lint run` passes
  - `go test -race ./...` passes
- [ ] API Contract (if HTTP changes)
  - `make validate-openapi` passes
  - OpenAPI spec updated
- [ ] Test Coverage
  - New code has tests
  - Coverage >= 80% overall
  - Domain coverage >= 90%
- [ ] Security Review (if applicable)
  - Security checklist completed (see `claude/agent_checklist.md`)
  - No hardcoded secrets
  - Input validation in place
- [ ] Documentation
  - README or guide updated if needed
  - Architecture decisions recorded (if applicable)

**Pass Criteria**: All items checked
**Fail Action**: Return to agent for remediation

---

### Sprint Retrospective Gate

**Owner**: scrum-master
**Trigger**: End of sprint

**Checklist**:
- [ ] Sprint goal achieved or variance explained
- [ ] All committed work completed or carried to next sprint
- [ ] Security gate passed (for sprints with security requirements)
- [ ] Technical debt documented
- [ ] Retrospective conducted with improvement actions
- [ ] Sprint metrics recorded (velocity, coverage, defects)

**Pass Criteria**: All items checked
**Fail Action**: Sprint marked incomplete, issues documented in retrospective

---

### Security Gates (Sprint-Specific)

**Owner**: senior-secops-engineer
**Trigger**: Sprint completion for sprints with security deliverables

See `/home/user/goimg-datalayer/claude/security_gates.md` for detailed security gate requirements by sprint.

**Key Sprints with Security Gates**:
- **Sprint 1-2**: Foundation security (no secrets, dependency scanning)
- **Sprint 3**: Authentication security (JWT, Argon2id, database SSL)
- **Sprint 4**: Authorization security (RBAC, session management)
- **Sprint 5**: Upload security (ClamAV, file validation, EXIF stripping)
- **Sprint 7**: Moderation security (audit logging, privilege escalation prevention)
- **Sprint 8**: Comprehensive security hardening (SAST, DAST, penetration testing)

**Process**:
1. Self-review against security checklist
2. Automated scanning (gosec, trivy, nancy)
3. senior-secops-engineer manual review
4. Status: PASS / CONDITIONAL PASS / FAIL

**Fail Response**: Sprint blocked until remediation complete

---

## Escalation & Conflict Resolution

### When to Escalate

**Technical Blockers**:
- Infrastructure unavailable >4 hours (e.g., Docker Compose services down)
- External dependency failure (e.g., library incompatibility)
- Architectural decision needed beyond agent scope

**Quality Concerns**:
- Test coverage below threshold despite effort
- Security vulnerability without clear remediation
- Performance degradation detected

**Process Concerns**:
- Sprint goal at risk with <3 days remaining
- Agent bandwidth exceeded (>3 concurrent tasks)
- Cross-agent coordination breaking down

### Escalation Template

```markdown
## Escalation: [Issue Title]

**Escalated By**: [agent-name]
**Date**: [date]
**Severity**: [Critical / High / Medium / Low]

### Issue Description
[Clear description of the problem]

### Impact
- Sprint goal: [Yes/No - explain impact]
- Blockers: [List of blocked tasks/agents]
- Deadline: [When this needs resolution]

### Attempted Resolutions
1. Attempt 1: [result]
2. Attempt 2: [result]

### Requested Action
[Specific ask - resource, decision, or intervention]

### Escalation Path
[Level 2: Peer Agent / Level 3: Scrum Master / Level 4: Stakeholder]
```

---

## Appendix: Quick Reference

### Agent Contact Matrix

| Need | Primary Agent | Backup Agent |
|------|---------------|--------------|
| Sprint planning | scrum-master | - |
| Library choice | senior-go-architect | senior-secops-engineer |
| Security review | senior-secops-engineer | scrum-master |
| Test design | backend-test-architect | test-strategist |
| API testing | test-strategist | backend-test-architect |
| CI/CD issues | cicd-guardian | scrum-master |
| Feature planning | image-gallery-expert | scrum-master |
| Work validation | scrum-master | - |

### Key Documents by Agent

| Agent | Primary Documents |
|-------|-------------------|
| scrum-master | `claude/sprint_plan.md`, `claude/mvp_features.md`, `claude/agent_workflow.md` |
| backend-test-architect | `claude/test_strategy.md`, `claude/testing_ci.md`, `claude/agent_checklist.md` |
| test-strategist | `claude/test_strategy.md`, `api/openapi/openapi.yaml` |
| senior-go-architect | `claude/architecture.md`, `claude/coding.md` |
| senior-secops-engineer | `claude/security_gates.md`, `claude/security_testing.md`, `claude/api_security.md` |
| cicd-guardian | `.github/workflows/`, `docker/docker-compose.yml`, `Makefile` |
| image-gallery-expert | `claude/mvp_features.md`, `claude/sprint_plan.md` |

### Sprint Ceremony Schedule

| Ceremony | Frequency | Duration | Attendees |
|----------|-----------|----------|-----------|
| Sprint Planning | Every 2 weeks | 2 hours | scrum-master, image-gallery-expert, senior-go-architect |
| Mid-Sprint Checkpoint | Week 1 of sprint | 30 min | scrum-master, active agents |
| Pre-Merge Reviews | Per feature | 30 min | scrum-master, reviewer agent |
| Sprint Retrospective | End of sprint | 1 hour | scrum-master, all active agents |
| Security Gate Review | Sprint 3, 4, 5, 7, 8 | 1 hour | senior-secops-engineer, scrum-master |

---

## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-12-02 | Initial agent workflow documentation | scrum-master |

