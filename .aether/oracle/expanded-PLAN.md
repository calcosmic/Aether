# Aether Comprehensive Implementation Plan

## Exhaustive Technical Roadmap for Production Readiness

---

**Document Version:** 2.0
**Original Plan Version:** 1.0
**Expansion Date:** 2026-02-16
**Target Word Count:** 40,000+ words
**Status:** Draft for Review

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Critical Path Analysis](#critical-path-analysis)
3. [Dependency Graph](#dependency-graph)
4. [Wave Overview](#wave-overview)
5. [Detailed Wave Breakdown](#detailed-wave-breakdown)
   - Wave 1: Foundation Fixes (Critical Bugs)
   - Wave 2: Error Handling Standardization
   - Wave 3: Template Path & queen-init Fix
   - Wave 4: Command Consolidation Infrastructure
   - Wave 5: XML System Activation (Phase 1)
   - Wave 6: XML System Integration (Phase 2)
   - Wave 7: Testing Expansion
   - Wave 8: Model Routing Verification
   - Wave 9: Documentation Consolidation
   - Wave 10: Colony Lifecycle Management
   - Wave 11: Performance & Hardening
   - Wave 12: Production Readiness
   - Wave 13: Advanced Colony Features
   - Wave 14: Cross-Colony Memory System
   - Wave 15: Ecosystem Integration
6. [Risk Mitigation Strategies](#risk-mitigation-strategies)
7. [Resource Allocation Plan](#resource-allocation-plan)
8. [Timeline Estimates](#timeline-estimates)
9. [Milestone Definitions](#milestone-definitions)
10. [Success Metrics](#success-metrics)
11. [Appendices](#appendices)

---

## Executive Summary

### Project Overview

Aether represents a paradigm shift in AI-assisted software development—a multi-agent CLI framework that orchestrates specialized AI workers (ants) to collaboratively build, test, and maintain software projects. Unlike traditional AI coding assistants that operate as single entities, Aether implements a biological metaphor inspired by ant colonies, where specialized castes (builders, watchers, scouts, chaos agents, oracles) work in coordinated harmony under a central Queen's direction.

The system draws inspiration from the sophisticated social structures of leafcutter ant colonies, where different worker castes perform specialized roles: foragers find resources, gardeners cultivate fungus gardens, soldiers defend the nest, and the queen coordinates reproduction and colony expansion. Aether translates this biological efficiency into software engineering, creating a self-organizing system where agents with different specializations can work in parallel while maintaining coherent progress toward project goals.

### Current State Assessment

As of February 2026, Aether exists in a functional but technically indebted state. The core system operates—colonies can be initialized, plans created, phases built, and work completed—but significant technical debt has accumulated during rapid development. Understanding this current state is essential for prioritizing the implementation waves that follow.

**Core System Metrics:**
- **Primary Utility Layer:** 3,592 lines of bash in `.aether/aether-utils.sh`, serving as the central nervous system for all colony operations
- **Command Surface:** 34 Claude Code commands plus 33 OpenCode commands, totaling 13,573 lines of duplicated markdown command definitions
- **Worker Castes:** 22 distinct caste types defined, from foundational builders and watchers to specialized chaos agents and archaeologists
- **XML Infrastructure:** 5 sophisticated XSD schemas (pheromone, queen-wisdom, colony-registry, worker-priming, prompt) representing a dormant but powerful cross-colony memory system
- **Test Coverage:** Recently completed session freshness detection system with 21/21 tests passing, but uneven coverage across other subsystems

**Technical Debt Inventory:**

The most pressing concern is the presence of critical bugs that threaten system stability. BUG-005 and BUG-011 represent lock deadlock conditions in the flag resolution system—when jq fails during flag operations, locks acquired at line 1364 are never released, causing subsequent operations to hang indefinitely. This is not merely an inconvenience; it represents a fundamental reliability issue that could cause data loss or require manual intervention to resolve.

BUG-007 reveals systemic inconsistency in error handling—17+ locations use hardcoded error strings instead of the E_* constants defined in error-handler.sh. This inconsistency fragments error recovery logic and prevents the system from providing consistent recovery suggestions when things go wrong.

ISSUE-004 is a deployment blocker: the queen-init command fails when Aether is installed via npm because it checks for templates in the runtime/ directory first, which doesn't exist in npm installs. This creates a poor first-user experience and limits distribution options.

Beyond bugs, the 13,573 lines of duplicated command definitions between Claude Code and OpenCode represent a maintenance nightmare. Every command change requires manual synchronization across two platforms, inevitably leading to drift and inconsistency. The YAML-based command generation system (Wave 4) aims to eliminate this duplication through single-source-of-truth definitions.

The XML system represents perhaps the most interesting form of technical debt—sophisticated infrastructure that was built but never fully activated. Five carefully designed XSD schemas exist for pheromone exchange, queen wisdom, colony registries, worker priming, and prompt structures. These schemas enable structured cross-colony memory, allowing wisdom gained in one project to inform another. However, the system remains dormant, with only basic XML utilities implemented but not integrated into production commands.

**Business Context:**

Aether operates at the intersection of several converging trends: the rise of AI-assisted development, the fragmentation of AI coding tools across platforms (Claude Code, OpenCode, Cursor, etc.), and the growing complexity of software projects that exceeds what single-agent AI systems can effectively manage.

The business value proposition centers on three pillars:

1. **Scalability Through Specialization:** Just as human software teams outperform individual developers through specialization, Aether's multi-agent approach allows different AI models to handle tasks suited to their strengths. Complex architectural decisions can be routed to reasoning-focused models, while routine implementation tasks go to faster, cheaper models.

2. **Knowledge Persistence:** Traditional AI coding sessions are ephemeral—context is lost when the session ends. Aether's colony state, pheromone system, and XML-based cross-colony memory create persistent institutional knowledge that improves over time.

3. **Platform Independence:** By supporting both Claude Code and OpenCode (with potential expansion to other platforms), Aether prevents vendor lock-in and allows teams to use their preferred tools while maintaining consistent workflows.

**Target State Vision:**

The implementation plan detailed in this document charts a course from the current indebted state to a production-ready system over 15 implementation waves. The target state encompasses:

- **Reliability:** Zero critical bugs, consistent error handling with meaningful recovery suggestions, graceful degradation when dependencies are missing
- **Maintainability:** Single-source-of-truth command generation eliminating 13K lines of duplication, comprehensive test coverage, clear documentation
- **Capability:** Active XML system enabling cross-colony wisdom sharing, verified model routing for cost-effective AI usage, complete colony lifecycle management
- **Usability:** Intuitive command structure, helpful error messages, comprehensive documentation, smooth onboarding for new users

**Implementation Philosophy:**

This plan follows several guiding principles:

1. **Foundation First:** Waves 1-3 address critical bugs and foundational issues before building new features. A system with deadlock bugs cannot be considered production-ready regardless of its feature set.

2. **Verification at Every Step:** Each task includes explicit verification steps, success criteria, and rollback procedures. Nothing is considered complete until it can be proven working.

3. **Incremental Value:** While the full plan spans 15 waves, earlier waves deliver independent value. Wave 1 alone makes the system significantly more reliable.

4. **Parallel Workstreams:** The dependency graph identifies opportunities for parallel development, reducing calendar time without increasing risk.

5. **Documentation as Code:** Documentation is treated as a first-class deliverable, with consolidation and maintenance as explicit work items.

**Success Definition:**

Aether will be considered "operating perfectly" when:
- All 22 commands work identically across Claude Code and OpenCode platforms
- Zero critical bugs remain (no deadlocks, no data loss scenarios)
- Model routing is verified and functional across all castes
- XML system is active and used for cross-colony memory
- Complete colony lifecycle management (init, build, archive, history)
- 100% test pass rate with meaningful coverage
- No known security vulnerabilities
- Documentation is current, consolidated, and comprehensive
- Performance meets established benchmarks
- CI/CD pipeline passes all checks

**Resource Requirements:**

Implementing this plan requires:
- **Technical Skills:** Expert-level bash/shell scripting, intermediate Node.js, intermediate XML/XSD, intermediate YAML, expert testing practices, security audit experience
- **Time Investment:** Approximately 39 developer days spread across 8 calendar weeks with parallel workstreams
- **Infrastructure:** Access to test environments for both Claude Code and OpenCode, CI/CD pipeline for automated testing

**Risk Summary:**

The highest risks are:
1. **Lock Deadlock Fixes (W1):** Could introduce new bugs if error handling isn't carefully implemented
2. **Command Generator (W4):** Complex system that could break all commands if flawed
3. **Model Routing (W8):** Depends on environment variable inheritance that may not work as expected
4. **E2E Testing (W12):** May reveal fundamental issues requiring significant rework

Each risk is mitigated through comprehensive testing, rollback procedures, and incremental rollout strategies.

**Conclusion:**

Aether represents a bold vision for AI-assisted development that goes beyond simple code generation to create a true collaborative ecosystem of specialized agents. The technical debt accumulated during its rapid development is significant but manageable. This implementation plan provides a clear, verifiable path from the current state to production readiness, with each wave building upon the last to create a reliable, maintainable, and powerful system.

The investment of approximately 8 weeks of development time will transform Aether from a promising but fragile prototype into a robust platform capable of orchestrating complex software development workflows across multiple AI platforms. The biological metaphor that inspired Aether's design—specialized castes working in harmony under coordinated direction—will finally be fully realized through reliable infrastructure, verified model routing, and persistent cross-colony memory.

---

## Critical Path Analysis

### Understanding the Critical Path

In project management, the critical path represents the sequence of tasks that determines the minimum duration required to complete a project. Any delay in critical path tasks directly delays project completion, while non-critical tasks have slack time. For Aether's implementation plan, understanding the critical path is essential for resource allocation and timeline estimation.

### Critical Path Identification

After analyzing dependencies between waves and tasks, the critical path to production-ready status is:

**Critical Path Sequence:**

1. **W1-T1: Fix Lock Deadlock in flag-auto-resolve** (2 days)
   - Reason: Critical bug affecting all flag operations; must be fixed before any reliable operations
   - Dependencies: None

2. **W1-T2: Fix Error Code Inconsistency** (1 day)
   - Reason: Foundation for reliable error handling throughout system
   - Dependencies: None (can parallel with W1-T1)

3. **W1-T3: Fix Lock Deadlock in flag-add** (0.5 days)
   - Reason: Same pattern as W1-T1, different location
   - Dependencies: W1-T1 (uses same fix pattern)

4. **W1-T4: Fix atomic-write Lock Leak** (0.5 days)
   - Reason: Core utility used throughout system
   - Dependencies: W1-T1, W1-T3

5. **W7-T2: Add Unit Tests for Bug Fixes** (2 days)
   - Reason: Regression tests prevent reintroduction of critical bugs
   - Dependencies: W1 (all bug fixes)

6. **W7-T4: Fix Failing Tests** (2 days)
   - Reason: Must have 100% pass rate for production
   - Dependencies: W7-T2

7. **W12-T1: End-to-End Testing** (2 days)
   - Reason: Validates complete workflows before release
   - Dependencies: All previous waves

8. **W12-T2: Security Audit** (1 day)
   - Reason: Must verify no vulnerabilities before production
   - Dependencies: All previous waves

9. **W12-T3: Release Preparation** (1 day)
   - Reason: Final packaging and documentation
   - Dependencies: W12-T1, W12-T2

**Critical Path Duration:** 12 days (minimum calendar time to production)

### Parallel Workstreams

While the critical path determines minimum duration, significant work can proceed in parallel:

**Workstream A: Foundation (Critical Path)**
- W1: Foundation Fixes
- W7: Testing Expansion (partial)
- W12: Production Readiness

**Workstream B: Command Infrastructure (Parallel)**
- W2: Error Handling (can start after W1-T1)
- W4: Command Consolidation (can start after W2)
- W9: Documentation Consolidation (can start after W4-T1)

**Workstream C: XML System (Parallel)**
- W5: XML Activation (can start after W4-T4)
- W6: XML Integration (depends on W5)
- W10: Colony Lifecycle (depends on W5, W1)

**Workstream D: Advanced Features (Parallel)**
- W3: Template Path (can start immediately)
- W8: Model Routing (depends on W7)
- W11: Performance (depends on W7, W8)

**Workstream E: Extended Features (Optional for Initial Release)**
- W13-W15: Advanced features that enhance but don't block production

### Critical Path with Parallel Workstreams

When parallel workstreams are considered, the calendar timeline extends from 12 days to approximately 8 weeks due to:

1. **Integration Points:** Parallel workstreams must integrate at key points (e.g., W4 must complete before W5 can start)
2. **Resource Constraints:** Some waves require the same expertise (bash scripting), creating resource contention
3. **Verification Overhead:** Each parallel stream requires testing and verification
4. **Buffer Time:** Real-world factors (meetings, context switching, unexpected issues) add overhead

### Float Analysis

Tasks not on the critical path have float (slack time):

- **W3 (Template Path):** 5 days float—can be delayed without affecting critical path
- **W9-T3 (Archive Stale Docs):** 10 days float—lowest priority
- **W8-T2 (Interactive Caste Config):** 7 days float—nice-to-have feature
- **W10-T3 (Milestone Auto-Detection):** 6 days float—decorative feature
- **W11-T1 (Performance Optimization):** 4 days float—performance is acceptable currently

### Risk Impact on Critical Path

Several risks could extend the critical path:

1. **W1-T1 Complexity:** If lock deadlock fix reveals deeper architectural issues, could add 2-3 days
2. **W7-T4 Test Failures:** If failing tests reveal fundamental problems, could add 3-5 days
3. **W12-T1 E2E Issues:** End-to-end testing often reveals integration issues; budget 1-2 days buffer
4. **W12-T2 Security Issues:** If audit finds vulnerabilities, remediation could add 2-4 days

**Recommended Buffer:** Add 20% buffer to critical path = 2.4 days, rounded to 3 days

**Adjusted Critical Path:** 15 days with buffer

### Resource Leveling

The critical path assumes continuous availability of required skills. Resource leveling (adjusting for limited availability) extends the timeline:

- **Bash Expertise Required:** Waves 1, 2, 3, 5, 6, 7, 10, 11 all require bash expertise
- **Node.js Required:** Waves 4, 8, 12 require Node.js skills
- **Testing Required:** Waves 7, 12 require testing expertise

If a single developer has all skills, critical path is 15 days. If skills are split across developers, handoff overhead adds approximately 20% = 18 days.

### Critical Path Visualization

```
Week 1: [W1-T1][W1-T2][W1-T3][W1-T4]  Critical Bug Fixes
        [W3-T1][W3-T2]                Template Path (parallel)

Week 2: [W2-T1][W2-T2][W2-T3]         Error Handling
        [W7-T1]                       Test Audit (parallel)

Week 3: [W4-T1][W4-T2][W4-T3][W4-T4]  Command Consolidation
        [W7-T2]                       Bug Regression Tests (parallel)

Week 4: [W5-T1][W5-T2][W5-T3][W5-T4]  XML Activation
        [W9-T1][W9-T2]                Doc Audit/Consolidation (parallel)
        [W7-T3][W7-T4]                Integration Tests (parallel)

Week 5: [W6-T1][W6-T2][W6-T3]         XML Integration
        [W10-T1][W10-T2]              Lifecycle Commands (parallel)
        [W9-T3]                       Doc Archive (parallel)

Week 6: [W8-T1][W8-T2]                Model Routing
        [W10-T3]                      Milestone Detection (parallel)
        [W11-T1][W11-T2][W11-T3]      Performance (parallel)

Week 7: [W13-all tasks]               Advanced Features (optional)
        [Buffer]                      Contingency

Week 8: [W12-T1][W12-T2][W12-T3]      Production Readiness
```

### Conclusion

The critical path analysis reveals that Aether can reach production-ready status in as little as 15 days of focused work on critical path items, or approximately 8 weeks when considering parallel workstreams, resource constraints, and buffer time. The foundation fixes (Wave 1) are genuinely critical—without them, the system cannot be considered reliable. However, many subsequent waves provide significant value while having float time, allowing flexibility in scheduling.

The most important takeaway is that Waves 1, 7, and 12 form an irreducible core: fix critical bugs, ensure test coverage, verify end-to-end functionality. Everything else enhances the system but doesn't block production readiness.

---

## Dependency Graph

### Visual Representation

```
                                    WAVE DEPENDENCY GRAPH
                                    =====================

    W1 (Foundation)          W2 (Error Handling)        W3 (Template)
    ================         ===================        =============
    [T1] Lock Deadlock       [T1] Error Constants       [T1] Path Resolution
    [T2] Error Codes    +-->[T2] Standardize Usage    [T2] Template Validation
    [T3] flag-add       |    [T3] Context Enrichment
    [T4] atomic-write   |
         |              |         |
         |              |         |
         v              v         v
    +----+--------------+---------+--------------------------------+
    |                     W4 (Command Consolidation)               |
    |    [T1] YAML Schema <-----+                                  |
    |    [T2] Generator         |                                  |
    |    [T3] Migration         |                                  |
    |    [T4] CI Check          |                                  |
    +---------------------------+----------------------------------+
                   |
         +---------+---------+
         |                   |
         v                   v
    W5 (XML Phase 1)     W9 (Documentation)
    ================     ==================
    [T1] xml-utils       [T1] Doc Audit
    [T2] Pheromone       [T2] Consolidation
    [T3] Cross-Colony    [T3] Archive
    [T4] XML Docs
         |
         +------------------+
         |                  |
         v                  v
    W6 (XML Phase 2)    W10 (Lifecycle)
    ================    ===============
    [T1] seal Export    [T1] Archive Cmd
    [T2] init Import    [T2] History Cmd
    [T3] QUEEN XML      [T3] Milestones
         |                  |
         +------------------+
         |
         v
    W7 (Testing) <---------------+
    ============                 |
    [T1] Coverage Audit          |
    [T2] Bug Regression ---------+ (depends on W1)
    [T3] Integration Tests       |
    [T4] Fix Failing             |
         |                       |
         +-----------+-----------+
         |           |
         v           v
    W8 (Model Routing)    W11 (Performance)
    ==================    ================
    [T1] Fix Routing      [T1] Loading Opt
    [T2] Interactive      [T2] Spawn Limits
                          [T3] Degradation
         |
         +---------------------------+
         |                           |
         v                           v
    W12 (Production)           W13+ (Advanced)
    ================           ===============
    [T1] E2E Testing           [Various features]
    [T2] Security Audit
    [T3] Release Prep
```

### Text-Based Dependency Matrix

| Wave | Depends On | Blocks | Parallel With |
|------|------------|--------|---------------|
| W1 | None | W2, W4, W7-T2, W10 | W3 |
| W2 | W1-T1 | W4 | W7-T1 |
| W3 | None | None | W1, W2 |
| W4 | W2 | W5, W9 | W7-T2 |
| W5 | W4 | W6, W10 | W9-T1, W9-T2 |
| W6 | W5 | W12 | W10 |
| W7 | None | W8, W11 | W2, W4, W5, W6 |
| W8 | W7 | W12 | W10-T3, W11 |
| W9 | W4 | None | W5, W6, W7 |
| W10 | W1, W5 | W12 | W6, W8, W11 |
| W11 | W7, W8 | W12 | W10 |
| W12 | All | None | None |
| W13-15 | W12 | None | None |

### Dependency Types

**Hard Dependencies (Must Complete First):**
- W2 depends on W1-T1: Error handling builds on stable foundation
- W4 depends on W2: Command generator needs error patterns
- W5 depends on W4: XML commands need command infrastructure
- W6 depends on W5: XML integration needs activated XML system
- W7-T2 depends on W1: Bug regression tests need bugs fixed
- W8 depends on W7: Model routing tests need testing infrastructure
- W10 depends on W1, W5: Lifecycle needs stable foundation + XML
- W12 depends on all: Production readiness needs everything

**Soft Dependencies (Should Complete First):**
- W9 should follow W4: Documentation consolidation benefits from command consolidation
- W11 should follow W7, W8: Performance optimization needs working system

**No Dependencies (Can Start Anytime):**
- W1 (Foundation): Entry point
- W3 (Template): Independent fix
- W7-T1 (Test Audit): Can audit anytime

### Circular Dependency Check

Analysis confirms no circular dependencies in the graph. All dependencies flow forward from foundation (W1) toward production (W12).

### Critical Dependencies

The most critical dependencies (longest chains) are:

1. **W1 -> W2 -> W4 -> W5 -> W6 -> W12** (6 hops)
   - Foundation through XML integration to production

2. **W1 -> W2 -> W4 -> W5 -> W10 -> W12** (6 hops)
   - Foundation through lifecycle to production

3. **W7 -> W8 -> W12** (3 hops)
   - Testing through model routing to production

### Dependency-Based Scheduling

Based on the dependency graph, the optimal schedule is:

**Phase 1: Foundation (Week 1)**
- Start: W1, W3 (parallel)
- End when: W1 complete

**Phase 2: Infrastructure (Weeks 2-3)**
- Start: W2 (after W1), W7-T1 (parallel)
- Continue: W4 (after W2)
- End when: W4 complete

**Phase 3: Parallel Development (Weeks 4-5)**
- Start: W5, W9 (parallel, after W4)
- Continue: W7-T2, W7-T3, W7-T4 (parallel)
- End when: W5, W7 complete

**Phase 4: Integration (Week 6)**
- Start: W6, W10 (parallel, after W5)
- Continue: W8, W11 (parallel, after W7)
- End when: W6, W8, W10, W11 complete

**Phase 5: Production (Week 8)**
- Start: W12 (after all others)
- End when: W12 complete

### Conclusion

The dependency graph reveals a well-structured project with clear sequencing. The longest dependency chains are 6 hops, which is reasonable for a project of this scope. The abundance of parallel opportunities (W3 with W1, W9 with W5, W10 with W6) means that with sufficient resources, calendar time can be significantly compressed from the 39-day total effort estimate.

The critical insight from dependency analysis is that W1 (Foundation Fixes) and W7 (Testing) are the primary bottlenecks—many other waves depend on them. Prioritizing these waves maximizes parallel workstream opportunities.

---

## Wave Overview

### Wave Summary Table

| Wave | Theme | Tasks | Est. Effort | Dependencies | Status | Priority |
|------|-------|-------|-------------|--------------|--------|----------|
| W1 | Foundation Fixes (Critical Bugs) | 4 | 4 days | None | Ready | P0 |
| W2 | Error Handling Standardization | 3 | 3 days | W1 | Ready | P1 |
| W3 | Template Path & queen-init Fix | 2 | 2 days | None | Ready | P0 |
| W4 | Command Consolidation Infrastructure | 4 | 8 days | W2 | Ready | P1 |
| W5 | XML System Activation (Phase 1) | 4 | 6 days | W4 | Ready | P1 |
| W6 | XML System Integration (Phase 2) | 3 | 5 days | W5 | Ready | P1 |
| W7 | Testing Expansion | 4 | 7 days | None | Ready | P0 |
| W8 | Model Routing Verification | 2 | 3 days | W7 | Ready | P1 |
| W9 | Documentation Consolidation | 3 | 4 days | W4 | Ready | P2 |
| W10 | Colony Lifecycle Management | 3 | 5 days | W1, W5 | Ready | P1 |
| W11 | Performance & Hardening | 3 | 4 days | W7, W8 | Ready | P2 |
| W12 | Production Readiness | 3 | 4 days | All | Ready | P0 |
| W13 | Advanced Colony Features | 4 | 6 days | W12 | Planned | P2 |
| W14 | Cross-Colony Memory System | 3 | 5 days | W13 | Planned | P3 |
| W15 | Ecosystem Integration | 3 | 4 days | W14 | Planned | P3 |

**Total Estimated Effort:** 70 days (approximately 14 weeks with parallel work)
**Critical Path:** 15 days (W1, W7-T2, W7-T4, W12)
**Minimum Viable Production:** Waves 1, 7, 12 (10 days)

---

## Detailed Wave Breakdown

---

### Wave 1: Foundation Fixes (Critical Bugs)

#### Wave Overview

Wave 1 addresses the most critical issues threatening Aether's stability and reliability. These are not feature enhancements or optimizations—they are fixes for bugs that can cause data loss, system deadlocks, or complete operational failure. The wave focuses on four specific bugs that have been identified through production usage and code review.

The biological metaphor of Aether as an ant colony is particularly apt here: just as a real colony cannot function if its communication pheromones are garbled or its workers get stuck in deadlocks, Aether cannot operate reliably when its flag system deadlocks or its error handling is inconsistent. Wave 1 is about ensuring the basic nervous system of the colony functions correctly.

**Business Justification:**

Critical bugs directly threaten user trust and data integrity. A system that deadlocks during routine operations or loses user work due to lock failures cannot be considered production-ready. The business impact of these bugs includes:

1. **User Frustration:** Deadlocks require manual intervention (killing processes, clearing lock files), creating a poor user experience
2. **Data Loss Risk:** If locks fail during state updates, colony state could become corrupted
3. **Operational Overhead:** Users must work around known bugs, increasing cognitive load
4. **Reputation Damage:** A system with known critical bugs appears unprofessional and unreliable

Fixing these bugs in Wave 1 (before any feature work) ensures that subsequent development happens on a stable foundation. Building features on top of buggy infrastructure is technical debt that compounds over time.

**Technical Rationale:**

The bugs addressed in Wave 1 share a common theme: resource management failure. Whether it's file locks that aren't released (BUG-005, BUG-011, BUG-006) or error handling that doesn't follow established patterns (BUG-007), the root cause is inconsistency in how the system manages resources and reports failures.

The fixes follow established patterns:
1. **Lock Management:** Always use try/finally-style patterns (or bash equivalents) to ensure locks are released even when errors occur
2. **Error Handling:** Centralize error definitions and use them consistently
3. **Validation:** Add validation at boundaries to catch issues early

These patterns are not novel—they are standard practices in reliable systems. Wave 1 is about applying these standard practices to Aether's codebase.

**Risk Analysis:**

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Fix introduces new bugs | Medium | High | Comprehensive testing, small focused changes |
| Lock fix breaks normal operation | Low | High | Test both error and success paths |
| Error code changes break existing scripts | Medium | Medium | Maintain backward compatibility, deprecation warnings |
| Time overrun due to complexity | Low | Medium | Time-box each task, escalate if stuck |

The primary risk is that fixing complex lock bugs could introduce new issues. The mitigation is to make small, focused changes with comprehensive test coverage. Each fix should be isolated and tested independently before integration.

**Resource Requirements:**

- **Primary Skill:** Expert bash scripting, particularly error handling and process management
- **Secondary Skill:** Understanding of file locking mechanisms and race conditions
- **Time:** 4 days (can be compressed to 2 days if parallelized)
- **Tools:** shellcheck for static analysis, bats for testing

**Success Criteria:**

1. All four critical bugs are fixed and verified
2. No regressions in existing functionality
3. Regression tests prevent reintroduction of bugs
4. Lock operations complete successfully even when jq fails
5. All error handling uses consistent E_* constants
6. Template path resolution works for both npm and git installs

---

#### W1-T1: Fix Lock Deadlock in flag-auto-resolve

**Task Description:**

The flag-auto-resolve command has a critical lock leak that can cause system-wide deadlocks. When the jq command fails during flag resolution (line 1368), the lock acquired at line 1364 is never released because json_err exits without releasing it. This causes a deadlock where subsequent flag operations hang indefinitely waiting for a lock that will never be released.

The issue manifests when:
1. A user or automated process calls flag-auto-resolve
2. The flags.json file is acquired with acquire_lock
3. The jq command fails (e.g., due to malformed JSON, disk issues, or race conditions)
4. The error handler json_err is called, which exits the script
5. The lock is never released
6. Subsequent flag operations hang waiting for the lock

This is particularly dangerous because:
- It can happen during automated builds, causing CI/CD pipelines to hang
- It requires manual intervention to clear (finding and removing lock files)
- It affects all flag operations, not just the one that failed
- It can cascade—if a build process hangs, it may hold other resources

The fix requires wrapping jq operations in error handlers that release the lock before calling json_err. In bash, this means using trap handlers or explicit error checking with cleanup.

**Step-by-Step Implementation:**

1. **Locate the vulnerable code:**
   - File: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
   - Lines: 1360-1390 (flag-auto-resolve case)
   - Specific issue: Lines 1364, 1368, 1376

2. **Analyze the current flow:**
   ```bash
   # Current (vulnerable) pattern:
   acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "..."
   count=$(jq ... "$flags_file") || json_err "$E_JSON_INVALID" "..."  # Lock leaked!
   updated=$(jq ... "$flags_file") || json_err "$E_JSON_INVALID" "..."  # Lock leaked!
   atomic_write "$flags_file" "$updated"
   release_lock "$flags_file"
   ```

3. **Design the fix:**
   - Use a trap to ensure lock release on exit
   - Or use explicit error handling with cleanup
   - Pattern: acquire -> try -> catch -> finally -> release

4. **Implement the fix:**
   ```bash
   # Fixed pattern:
   acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "..."

   # Set trap to release lock on exit
   trap 'release_lock "$flags_file" 2>/dev/null || true' EXIT

   count=$(jq ... "$flags_file") || {
     json_err "$E_JSON_INVALID" "Failed to count flags"
   }

   updated=$(jq ... "$flags_file") || {
     json_err "$E_JSON_INVALID" "Failed to auto-resolve flags"
   }

   atomic_write "$flags_file" "$updated"
   # Lock will be released by trap
   ```

5. **Handle edge cases:**
   - What if release_lock fails? (log but don't fail)
   - What if the trap fires multiple times? (make release_lock idempotent)
   - What if LOCK_ACQUIRED tracking is wrong? (defensive programming)

6. **Test the fix:**
   - Test normal operation (jq succeeds)
   - Test jq failure (simulated with invalid JSON)
   - Test lock release verification
   - Test concurrent access

**Code Example:**

```bash
flag-auto-resolve)
  trigger="${1:-build_pass}"
  flags_file="$DATA_DIR/flags.json"

  if [[ ! -f "$flags_file" ]]; then json_ok '{"resolved":0}'; exit 0; fi

  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  # Acquire lock for atomic flag update
  if type feature_enabled &>/dev/null && ! feature_enabled "file_locking"; then
    json_warn "W_DEGRADED" "File locking disabled - proceeding without lock"
  else
    acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "Failed to acquire lock on flags.json"
  fi

  # CRITICAL: Ensure lock is always released
  local lock_released=false
  _release_flag_lock() {
    if [[ "$lock_released" == "false" ]]; then
      release_lock "$flags_file" 2>/dev/null || true
      lock_released=true
    fi
  }
  trap '_release_flag_lock' EXIT

  # Count how many will be resolved
  count=$(jq --arg trigger "$trigger" '
    [.flags[] | select(.auto_resolve_on == $trigger and .resolved_at == null)] | length
  ' "$flags_file") || {
    json_err "$E_JSON_INVALID" "Failed to count flags for auto-resolve"
  }

  # Resolve them
  updated=$(jq --arg trigger "$trigger" --arg ts "$ts" '
    .flags = [.flags[] | if .auto_resolve_on == $trigger and .resolved_at == null then
      .resolved_at = $ts |
      .resolution = "Auto-resolved on " + $trigger
    else . end]
  ' "$flags_file") || {
    json_err "$E_JSON_INVALID" "Failed to auto-resolve flags"
  }

  atomic_write "$flags_file" "$updated"
  # Lock released by trap
  json_ok "{\"resolved\":$count,\"trigger\":\"$trigger\"}"
  ;;
```

**Testing Strategy:**

1. **Unit Tests:**
   ```bash
   # test-w1-t1.sh
   #!/bin/bash
   set -euo pipefail

   # Setup: Create test environment
   TEST_DIR=$(mktemp -d)
   export DATA_DIR="$TEST_DIR/data"
   mkdir -p "$DATA_DIR"

   # Test 1: Normal operation
   echo '{"flags":[]}' > "$DATA_DIR/flags.json"
   result=$(bash aether-utils.sh flag-auto-resolve "build_pass")
   [[ $(echo "$result" | jq -r '.result.resolved') == "0" ]]
   echo "✓ Test 1: Normal operation"

   # Test 2: Lock released on jq failure (simulated)
   echo 'invalid json' > "$DATA_DIR/flags.json"
   result=$(bash aether-utils.sh flag-auto-resolve "build_pass" 2>&1) || true
   [[ "$result" == *"Failed to count flags"* ]]

   # Verify lock released
   [[ ! -f "$DATA_DIR/flags.json.lock" ]]
   echo "✓ Test 2: Lock released on jq failure"

   # Test 3: Subsequent operations succeed after failure
   echo '{"flags":[]}' > "$DATA_DIR/flags.json"
   result=$(bash aether-utils.sh flag-auto-resolve "build_pass")
   [[ $(echo "$result" | jq -r '.ok') == "true" ]]
   echo "✓ Test 3: Recovery after failure"

   # Cleanup
   rm -rf "$TEST_DIR"
   echo "All W1-T1 tests passed!"
   ```

2. **Integration Tests:**
   - Test within full colony workflow
   - Test with concurrent flag operations
   - Test under load (many rapid flag operations)

3. **Manual Verification:**
   ```bash
   # Simulate jq failure by corrupting JSON
   echo 'invalid' > .aether/data/flags.json
   bash .aether/aether-utils.sh flag-auto-resolve "build_pass"
   # Should fail gracefully with lock released

   # Verify no stale locks
   ls .aether/data/locks/  # Should be empty or not exist

   # Verify normal operation still works
   bash .aether/aether-utils.sh flag-add "test" "Test flag" --auto-resolve-on="build_pass"
   bash .aether/aether-utils.sh flag-auto-resolve "build_pass"
   # Should succeed
   ```

**Rollback Procedures:**

1. **Immediate Rollback (if critical failure detected):**
   ```bash
   # Revert to previous version
   git checkout HEAD -- .aether/aether-utils.sh

   # Clear any stale locks
   rm -f .aether/data/locks/*

   # Verify system functional
   bash .aether/aether-utils.sh flag-list
   ```

2. **Selective Rollback (if only this change needs revert):**
   ```bash
   # Restore from backup (if created)
   cp .aether/aether-utils.sh.bak .aether/aether-utils.sh

   # Or manually revert the specific function
   git diff HEAD .aether/aether-utils.sh  # Review changes
   git checkout HEAD -- .aether/aether-utils.sh
   ```

3. **Post-Rollback Verification:**
   ```bash
   # Test all flag operations
   bash .aether/aether-utils.sh flag-list
   bash .aether/aether-utils.sh flag-add "test" "Test"
   bash .aether/aether-utils.sh flag-auto-resolve "test"

   # Check for errors
   echo "Rollback complete, system functional"
   ```

**Verification Checklist:**

- [ ] jq failure during flag-auto-resolve releases lock before exiting
- [ ] Subsequent flag operations succeed after jq failure
- [ ] No regression in normal flag resolution path
- [ ] Lock file is not left behind after any error condition
- [ ] Trap-based cleanup works correctly
- [ ] Multiple jq failures in sequence don't accumulate locks
- [ ] Concurrent flag operations don't deadlock
- [ ] All existing tests still pass
- [ ] New regression tests prevent reintroduction

---

#### W1-T2: Fix Error Code Inconsistency (BUG-007)

**Task Description:**

BUG-007 represents a systemic inconsistency in Aether's error handling. Throughout the 3,592-line aether-utils.sh file, 17+ locations use hardcoded error strings instead of the E_* constants defined in error-handler.sh. This inconsistency creates several problems:

1. **Fragmented Error Handling:** Different parts of the system handle the same error types differently
2. **Broken Recovery Suggestions:** The error-handler.sh maps error codes to recovery suggestions, but hardcoded strings bypass this mapping
3. **Maintenance Burden:** Changing error messages requires finding all hardcoded instances
4. **Testing Difficulty:** Tests must check for multiple variations of the same error

The error-handler.sh defines constants like:
- E_FILE_NOT_FOUND="FILE_NOT_FOUND"
- E_JSON_INVALID="JSON_INVALID"
- E_LOCK_FAILED="LOCK_FAILED"
- E_VALIDATION_FAILED="VALIDATION_FAILED"

But many locations use raw strings like:
- `json_err "Failed to read file"` (should be E_FILE_NOT_FOUND)
- `json_err "Invalid JSON"` (should be E_JSON_INVALID)
- `json_err "Lock acquisition failed"` (should be E_LOCK_FAILED)

The fix requires auditing all json_err calls and replacing hardcoded strings with proper E_* constants. This is not a simple find/replace—it requires understanding the context of each error to assign the correct code.

**Step-by-Step Implementation:**

1. **Audit Current Error Usage:**
   ```bash
   # Find all json_err calls
   grep -n 'json_err' .aether/aether-utils.sh | head -30

   # Find hardcoded strings (not using E_* constants)
   grep -n 'json_err "[^$]' .aether/aether-utils.sh

   # Document each occurrence with context
   ```

2. **Map Errors to Constants:**
   | Current String | Context | Should Be |
   |----------------|---------|-----------|
   | "Failed to read file" | File read operations | E_FILE_NOT_FOUND |
   | "Invalid JSON" | JSON parsing | E_JSON_INVALID |
   | "Lock acquisition failed" | Lock operations | E_LOCK_FAILED |
   | "Validation failed" | Input validation | E_VALIDATION_FAILED |
   | "Permission denied" | File permissions | E_PERMISSION_DENIED |

3. **Update Error Definitions (if needed):**
   - Check if all needed constants exist in error-handler.sh
   - Add missing constants with recovery suggestions
   - Ensure consistent naming convention

4. **Replace Hardcoded Strings:**
   - Go through each occurrence systematically
   - Replace with appropriate E_* constant
   - Preserve any dynamic message portions

5. **Add Regression Test:**
   - Create test that verifies no hardcoded strings remain
   - Run as part of CI/CD

**Code Example:**

Before:
```bash
# Line 814 (example)
result=$(jq -r '.some_field' "$file") || {
  json_err "Failed to parse JSON"  # Hardcoded string
}

# Line 1022 (example)
acquire_lock "$file" || {
  json_err "Could not acquire lock"  # Hardcoded string
}
```

After:
```bash
# Line 814 (fixed)
result=$(jq -r '.some_field' "$file") || {
  json_err "$E_JSON_INVALID" "Failed to parse JSON from $file"
}

# Line 1022 (fixed)
acquire_lock "$file" || {
  json_err "$E_LOCK_FAILED" "Could not acquire lock on $file"
}
```

**Testing Strategy:**

1. **Static Analysis Test:**
   ```bash
   # test-error-codes.sh
   #!/bin/bash

   # Check for hardcoded error strings
   violations=$(grep -n 'json_err "[^$]' .aether/aether-utils.sh | grep -v 'json_err "\$E_')

   if [[ -n "$violations" ]]; then
     echo "ERROR: Found hardcoded error strings:"
     echo "$violations"
     exit 1
   fi

   echo "✓ All json_err calls use E_* constants"
   ```

2. **Error Recovery Test:**
   ```bash
   # Verify recovery suggestions work
   result=$(bash .aether/aether-utils.sh nonexistent-command 2>&1)
   [[ "$result" == *"recovery"* ]]
   echo "✓ Recovery suggestions present"
   ```

3. **Consistency Test:**
   ```bash
   # Verify same error type produces same code
   # (Test various paths that should produce same error)
   ```

**Rollback Procedures:**

1. **Revert Changes:**
   ```bash
   git checkout HEAD -- .aether/aether-utils.sh
   ```

2. **Verify Rollback:**
   ```bash
   # Verify hardcoded strings are back
   grep -c 'json_err "[^$]' .aether/aether-utils.sh
   # Should show count > 0 (back to original state)
   ```

**Verification Checklist:**

- [ ] All json_err calls use E_* constants
- [ ] No hardcoded error strings in error paths
- [ ] Recovery suggestions work for all error types
- [ ] Regression test prevents future inconsistency
- [ ] Error codes follow naming convention
- [ ] All error constants are exported
- [ ] Error messages are still descriptive
- [ ] No functional changes (only error codes changed)

---

#### W1-T3: Fix Lock Deadlock in flag-add (BUG-002)

**Task Description:**

BUG-002 is structurally identical to BUG-005/W1-T1 but occurs in the flag-add command rather than flag-auto-resolve. The same pattern of lock acquisition without guaranteed release exists in the flag-add implementation around line 814.

When a user adds a flag:
1. The flag-add command acquires a lock on flags.json
2. It reads and modifies the JSON
3. If jq fails during any operation, json_err is called
4. json_err exits without releasing the lock
5. The lock file remains, blocking all future flag operations

This bug is particularly problematic because flag-add is one of the most frequently used commands. Users regularly add flags to mark blockers, issues, and notes during colony operations. A deadlock here disrupts the normal workflow of marking and tracking issues.

The fix follows the exact same pattern as W1-T1: wrap jq operations in error handlers that release locks before exiting, using trap-based cleanup.

**Step-by-Step Implementation:**

1. **Locate the vulnerable code:**
   - File: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
   - Function: flag-add case block
   - Lines: Around 814 (verify exact location)

2. **Apply the same fix pattern as W1-T1:**
   - Add trap-based cleanup
   - Wrap jq operations in error handlers
   - Ensure lock released in all paths

3. **Verify consistency:**
   - The fix should match W1-T1 pattern exactly
   - Consistent error handling across all flag operations

**Code Example:**

```bash
flag-add)
  # ... argument parsing ...

  flags_file="$DATA_DIR/flags.json"

  # Acquire lock
  acquire_lock "$flags_file" || json_err "$E_LOCK_FAILED" "..."

  # Set up cleanup trap
  trap 'release_lock "$flags_file" 2>/dev/null || true' EXIT

  # Read current flags
  current=$(jq '.' "$flags_file") || {
    json_err "$E_JSON_INVALID" "Failed to read flags"
  }

  # Add new flag
  updated=$(echo "$current" | jq --arg flag "$flag_id" ...) || {
    json_err "$E_JSON_INVALID" "Failed to add flag"
  }

  atomic_write "$flags_file" "$updated"
  # Lock released by trap
  json_ok "{\"added\":\"$flag_id\"}"
  ;;
```

**Testing Strategy:**

Same pattern as W1-T1, but focused on flag-add:
1. Test normal flag addition
2. Test jq failure during flag addition
3. Verify lock released after failure
4. Test concurrent flag additions

**Rollback Procedures:**

Same as W1-T1—revert to HEAD if issues arise.

**Verification Checklist:**

- [ ] jq failure during flag-add releases lock
- [ ] Lock file cleanup happens in all error paths
- [ ] Normal flag addition still works
- [ ] Concurrent flag additions don't deadlock
- [ ] Pattern matches W1-T1 implementation

---

#### W1-T4: Fix atomic-write Lock Leak (BUG-006)

**Task Description:**

BUG-006 exists in the atomic-write.sh utility, a core component used throughout Aether for safe file writes. The atomic-write pattern (write to temp file, then move to target) ensures that readers never see partially written files. However, if JSON validation fails at line 66, the lock acquired earlier is not released.

The atomic-write.sh utility is used by:
- flag-auto-resolve
- flag-add
- state updates
- pheromone operations
- Any other file write that needs atomicity

A lock leak here is particularly dangerous because:
1. It's a shared utility—one bug affects many operations
2. The lock is on the target file, blocking all access
3. It's used for critical state files (COLONY_STATE.json, flags.json)

The fix requires ensuring lock release in all error paths, particularly the JSON validation failure path.

**Step-by-Step Implementation:**

1. **Locate the vulnerable code:**
   - File: `/Users/callumcowie/repos/Aether/.aether/utils/atomic-write.sh`
   - Line 66: JSON validation failure

2. **Analyze the current flow:**
   ```bash
   # Simplified current flow
   acquire_lock "$target"

   # Write to temp
   echo "$content" > "$temp"

   # Validate JSON (line 66)
   jq '.' "$temp" >/dev/null || {
     # Lock not released!
     return 1
   }

   mv "$temp" "$target"
   release_lock "$target"
   ```

3. **Implement the fix:**
   - Add trap-based cleanup
   - Or add explicit release in error path
   - Ensure cleanup happens before return

**Code Example:**

```bash
atomic_write() {
  local target="$1"
  local content="$2"
  local temp=$(mktemp)

  # Acquire lock
  acquire_lock "$target" || return 1

  # Ensure lock released on exit
  trap 'rm -f "$temp"; release_lock "$target" 2>/dev/null || true' EXIT

  # Write to temp
  echo "$content" > "$temp"

  # Validate JSON if target ends in .json
  if [[ "$target" == *.json ]]; then
    jq '.' "$temp" >/dev/null || {
      echo "Invalid JSON" >&2
      return 1
      # Trap will release lock and clean up temp
    }
  fi

  # Atomic move
  mv "$temp" "$target"

  # Lock released by trap
  trap - EXIT  # Clear trap
  return 0
}
```

**Testing Strategy:**

1. **Unit Test:**
   ```bash
   # Test JSON validation failure releases lock
   source .aether/utils/atomic-write.sh

   # Try to write invalid JSON
   atomic_write "test.json" "invalid json" && exit 1

   # Verify lock released
   [[ ! -f "test.json.lock" ]]
   ```

2. **Integration Test:**
   - Test with actual colony operations
   - Verify no stale locks after errors

**Rollback Procedures:**

```bash
git checkout HEAD -- .aether/utils/atomic-write.sh
```

**Verification Checklist:**

- [ ] JSON validation failure releases lock
- [ ] All error paths in atomic-write release locks
- [ ] Normal atomic write still works
- [ ] Temp files cleaned up in all paths
- [ ] No regression in dependent operations

---

### Wave 2: Error Handling Standardization

#### Wave Overview

Wave 2 builds upon the foundation established in Wave 1 to create a comprehensive, consistent error handling system across all Aether utilities. While Wave 1 fixed critical bugs in existing error handling, Wave 2 establishes patterns and infrastructure for future error handling.

The goal is to transform error handling from an afterthought into a first-class system feature. Users should never see raw stack traces or cryptic error codes—they should see clear explanations of what went wrong and specific suggestions for how to fix it.

**Business Justification:**

Error messages are user interface. When something goes wrong (and things always go wrong eventually), the error message is often the only communication channel between the system and the user. Good error handling:

1. **Reduces Support Burden:** Clear error messages with recovery suggestions mean users can solve their own problems
2. **Builds Trust:** Users trust systems that fail gracefully and explain themselves
3. **Speeds Recovery:** Specific recovery suggestions reduce time-to-resolution
4. **Improves Adoption:** New users are more likely to stick with a system that helps them when they're stuck

The business impact of poor error handling is cumulative: every confused user, every support request, every abandoned session due to cryptic errors represents lost value.

**Technical Rationale:**

Consistent error handling provides several technical benefits:

1. **Centralized Maintenance:** Error codes, messages, and recovery suggestions defined in one place
2. **Structured Logging:** JSON error output enables automated log analysis and alerting
3. **Testability:** Consistent error codes make tests more reliable and maintainable
4. **Extensibility:** New error types follow established patterns

The technical implementation follows the pattern established by modern CLI tools:
- Structured error output (JSON with consistent schema)
- Error codes for programmatic handling
- Human-readable messages for interactive use
- Recovery suggestions for self-service resolution

**Risk Analysis:**

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| New error codes conflict with existing | Low | Medium | Audit existing codes before adding |
| Recovery suggestions are wrong | Medium | Medium | Test each recovery path |
| Error format changes break scripts | Medium | High | Maintain backward compatibility |
| Time overrun | Low | Low | Well-defined scope |

The main risk is that new error handling might inadvertently change error output in ways that break existing scripts or user expectations. The mitigation is to maintain backward compatibility—add new fields but don't remove existing ones.

**Resource Requirements:**

- **Primary Skill:** Bash scripting, error handling patterns
- **Secondary Skill:** JSON schema design, user experience design
- **Time:** 3 days
- **Tools:** shellcheck, jq for JSON validation

**Success Criteria:**

1. All error codes have recovery suggestions
2. Error codes follow consistent naming convention
3. Error output includes context (operation, file paths)
4. All utility scripts use consistent error format
5. Documentation updated with error reference

---

#### W2-T1: Add Missing Error Code Constants

**Task Description:**

The error-handler.sh defines several E_* constants, but common error scenarios are missing. This task adds error code constants for:

- E_PERMISSION_DENIED: File permission issues
- E_TIMEOUT: Operation timeout
- E_CONFLICT: Concurrent modification
- E_INVALID_STATE: Colony state issues

Each error code needs:
1. Constant definition
2. Default error message
3. Recovery suggestion mapping
4. Documentation

**Step-by-Step Implementation:**

1. **Review existing constants:**
   ```bash
   grep '^E_' .aether/utils/error-handler.sh
   ```

2. **Identify gaps:**
   - What error scenarios occur frequently?
   - What errors lack specific codes?
   - What would help users recover?

3. **Add new constants:**
   ```bash
   # Permission errors
   E_PERMISSION_DENIED="PERMISSION_DENIED"

   # Timeout errors
   E_TIMEOUT="TIMEOUT"

   # Conflict errors
   E_CONFLICT="CONCURRENT_MODIFICATION"

   # State errors
   E_INVALID_STATE="INVALID_STATE"
   ```

4. **Add recovery suggestions:**
   ```bash
   case "$code" in
     "$E_PERMISSION_DENIED")
       echo "Check file permissions with: ls -la <path>"
       echo "Fix with: chmod +rw <path>"
       ;;
     "$E_TIMEOUT")
       echo "Operation timed out. Try again or increase timeout."
       ;;
     "$E_CONFLICT")
       echo "Another process modified the file. Retry the operation."
       ;;
     "$E_INVALID_STATE")
       echo "Colony state is invalid. Run /ant:status to diagnose."
       ;;
   esac
   ```

5. **Update documentation:**
   - Add new codes to error reference
   - Document when to use each code

**Code Example:**

```bash
# error-handler.sh additions

# File permission errors
E_PERMISSION_DENIED="PERMISSION_DENIED"

# Timeout errors
E_TIMEOUT="TIMEOUT"
E_LOCK_TIMEOUT="LOCK_TIMEOUT"

# Concurrent modification
E_CONFLICT="CONCURRENT_MODIFICATION"
E_STATE_CONFLICT="STATE_CONFLICT"

# Invalid state
E_INVALID_STATE="INVALID_STATE"
E_CORRUPT_DATA="CORRUPT_DATA"

# Recovery suggestion function
get_recovery_suggestion() {
  local code="$1"
  local context="${2:-}"

  case "$code" in
    "$E_PERMISSION_DENIED")
      echo "File permission denied. Check: ls -la $context"
      ;;
    "$E_TIMEOUT")
      echo "Operation timed out. Retry or check system load."
      ;;
    "$E_CONFLICT")
      echo "Concurrent modification detected. Retry the operation."
      ;;
    "$E_INVALID_STATE")
      echo "Invalid colony state. Run: /ant:status"
      ;;
    *)
      echo "Contact support with error code: $code"
      ;;
  esac
}
```

**Testing Strategy:**

1. **Verify constants exported:**
   ```bash
   source .aether/utils/error-handler.sh
   echo "$E_PERMISSION_DENIED"  # Should output: PERMISSION_DENIED
   ```

2. **Test recovery suggestions:**
   ```bash
   suggestion=$(get_recovery_suggestion "$E_TIMEOUT")
   [[ "$suggestion" == *"timed out"* ]]
   ```

**Verification Checklist:**

- [ ] All new error codes have recovery suggestions
- [ ] Error codes follow naming convention (E_*)
- [ ] Constants are exported
- [ ] Documentation updated
- [ ] No conflicts with existing codes

---

#### W2-T2: Standardize Error Handler Usage

**Task Description:**

Different utility scripts have different error handling implementations. Some use the enhanced json_err from error-handler.sh, others use fallback implementations. This task ensures all scripts consistently use the enhanced error handler.

Files to update:
- aether-utils.sh (fallback json_err at lines 66-73)
- xml-utils.sh (xml_json_err)
- Any other utilities with custom error handling

**Step-by-Step Implementation:**

1. **Audit current implementations:**
   ```bash
   grep -n 'json_err' .aether/utils/*.sh
   grep -n 'fallback' .aether/utils/*.sh
   ```

2. **Identify inconsistencies:**
   - Different parameter signatures
   - Different output formats
   - Missing recovery suggestions

3. **Standardize on enhanced handler:**
   - Remove fallback implementations
   - Ensure error-handler.sh is sourced early
   - Use consistent 4-parameter signature

4. **Update all call sites:**
   - Change: `json_err "message"`
   - To: `json_err "$E_CODE" "message" "details" "recovery"`

**Code Example:**

Before (inconsistent):
```bash
# aether-utils.sh fallback
json_err() {
  local message="${2:-$1}"
  printf '{"ok":false,"error":"%s"}\n' "$message" >&2
  exit 1
}

# xml-utils.sh custom
xml_json_err() {
  echo "{\"error\":\"$1\"}" >&2
}
```

After (standardized):
```bash
# All files source error-handler.sh
source "$SCRIPT_DIR/utils/error-handler.sh"

# Use consistent signature everywhere
json_err "$E_FILE_NOT_FOUND" "File not found" "$filepath" "Check the path and try again"
```

**Testing Strategy:**

1. **Verify consistent format:**
   ```bash
   # All errors should have same structure
   bash .aether/aether-utils.sh invalid-command 2>&1 | jq '.error | keys'
   # Should show: ["code", "message", "details", "recovery", "timestamp"]
   ```

2. **Test all utility scripts:**
   - Test error paths in each script
   - Verify consistent output format

**Verification Checklist:**

- [ ] All json_err calls use 4-parameter signature
- [ ] Fallback implementations removed
- [ ] Consistent error format across all utilities
- [ ] error-handler.sh sourced in all scripts
- [ ] All tests pass with new format

---

#### W2-T3: Add Error Context Enrichment

**Task Description:**

Error messages are more helpful when they include context about what operation was being performed, what file was being accessed, and what the system state was. This task enhances error messages with contextual information.

Context to add:
- Operation name (what was being attempted)
- File paths (relative to project root)
- Phase/state information (when applicable)
- Stack trace (in debug mode)

**Step-by-Step Implementation:**

1. **Identify error locations:**
   - Find all json_err calls
   - Determine what context is available at each location

2. **Add context gathering:**
   ```bash
   # Before error, gather context
   local operation="flag-auto-resolve"
   local context_file="${filepath#$AETHER_ROOT/}"  # Relative path
   local current_phase=$(jq -r '.current_phase' "$STATE_FILE" 2>/dev/null || echo "unknown")
   ```

3. **Enhance error calls:**
   ```bash
   json_err "$E_JSON_INVALID" \
     "Failed to parse flags.json" \
     "file: $context_file, phase: $current_phase, operation: $operation" \
     "Check file syntax with: jq . $context_file"
   ```

4. **Add debug mode:**
   - If AETHER_DEBUG=1, include stack trace
   - Use bash's `caller` builtin for trace

**Code Example:**

```bash
# Enhanced error with context
json_err_with_context() {
  local code="$1"
  local message="$2"
  local details="$3"
  local recovery="$4"

  # Add context
  local context=""
  if [[ -n "${CURRENT_OPERATION:-}" ]]; then
    context+="operation: $CURRENT_OPERATION, "
  fi
  if [[ -n "${CURRENT_FILE:-}" ]]; then
    context+="file: ${CURRENT_FILE#$AETHER_ROOT/}, "
  fi
  if [[ -n "${CURRENT_PHASE:-}" ]]; then
    context+="phase: $CURRENT_PHASE"
  fi

  # Add debug info if enabled
  if [[ "${AETHER_DEBUG:-0}" == "1" ]]; then
    local trace=$(caller 1)
    context+=" trace: $trace"
  fi

  json_err "$code" "$message" "${details}; $context" "$recovery"
}
```

**Testing Strategy:**

1. **Test context inclusion:**
   ```bash
   AETHER_DEBUG=1 bash .aether/aether-utils.sh invalid-command 2>&1 | jq '.error.details'
   # Should show context information
   ```

2. **Test relative paths:**
   - Verify file paths are relative to project root
   - Not absolute paths (which leak system structure)

**Verification Checklist:**

- [ ] Error details include operation context
- [ ] File paths in errors are relative to project root
- [ ] Stack trace available in debug mode
- [ ] Context doesn't expose sensitive information
- [ ] Errors remain readable with context

---

### Wave 3: Template Path & queen-init Fix

#### Wave Overview

Wave 3 addresses ISSUE-004, a deployment blocker that prevents Aether from working correctly when installed via npm. The queen-init command fails because it looks for templates in the runtime/ directory, which doesn't exist in npm installs.

This wave is small in scope (2 tasks, 2 days) but critical for distribution. A system that only works from git clones cannot reach wide adoption.

**Business Justification:**

npm is the standard distribution mechanism for Node.js-based tools. If Aether doesn't work when installed via npm:

1. **Limited Distribution:** Users must clone from git, which is a barrier to entry
2. **Version Management:** npm provides easy version management (npm update)
3. **Professional Credibility:** npm-installable tools are seen as more polished
4. **CI/CD Integration:** Most CI systems expect npm-based installation

The fix enables proper distribution through npm, removing a significant barrier to adoption.

**Technical Rationale:**

The root cause is a hardcoded path assumption. The template resolution logic checks runtime/ first, but in npm installs:
- runtime/ doesn't exist (it's a staging directory in git)
- Templates are in ~/.aether/system/ (the hub location)
- Or should fall back to .aether/ (source of truth)

The fix implements proper path resolution order:
1. .aether/ (source of truth, for development)
2. ~/.aether/system/ (hub location, for npm installs)
3. runtime/ (staging, for backward compatibility)

**Risk Analysis:**

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Fix breaks git clone workflow | Medium | High | Test both workflows |
| Template not found in any location | Low | Medium | Clear error message |
| Path resolution order wrong | Low | High | Document order, test all |

The main risk is breaking the existing git clone workflow while fixing npm. The mitigation is comprehensive testing of both installation methods.

**Resource Requirements:**

- **Primary Skill:** Bash scripting, path manipulation
- **Time:** 2 days
- **Tools:** npm for testing installs

**Success Criteria:**

1. queen-init works with npm-installed Aether
2. Template resolution follows correct order
3. Clear error if template not found
4. Git clone workflow still works

---

#### W3-T1: Fix Template Path Resolution

**Task Description:**

The queen-init command currently checks for templates in runtime/ first. This fails for npm installs where runtime/ doesn't exist. The fix implements proper template path resolution that works for both git and npm installations.

**Step-by-Step Implementation:**

1. **Locate template resolution code:**
   - File: `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
   - Lines: 2680-2705 (queen-init section)

2. **Understand current logic:**
   ```bash
   # Current (broken) logic
   if [[ -f "runtime/templates/QUEEN.md.template" ]]; then
     template="runtime/templates/QUEEN.md.template"
   elif [[ -f ".aether/templates/QUEEN.md.template" ]]; then
     template=".aether/templates/QUEEN.md.template"
   fi
   ```

3. **Design new resolution order:**
   - First: .aether/ (source of truth)
   - Second: ~/.aether/system/ (hub for npm installs)
   - Third: runtime/ (backward compatibility)

4. **Implement new logic:**
   ```bash
   find_template() {
     local template_name="$1"
     local locations=(
       ".aether/templates/$template_name"           # Source of truth
       "$HOME/.aether/system/templates/$template_name"  # Hub location
       "runtime/templates/$template_name"         # Backward compat
     )

     for location in "${locations[@]}"; do
       if [[ -f "$location" ]]; then
         echo "$location"
         return 0
       fi
     done

     return 1
   }
   ```

5. **Update queen-init to use new function:**
   ```bash
   template=$(find_template "QUEEN.md.template") || {
     json_err "$E_FILE_NOT_FOUND" "Template not found" "" "Reinstall Aether: npm install -g aether-colony"
   }
   ```

**Code Example:**

```bash
# New template resolution function
find_template() {
  local template_name="$1"
  local search_paths=(
    "${AETHER_ROOT:-.}/.aether/templates/$template_name"
    "${HOME}/.aether/system/templates/$template_name"
    "${AETHER_ROOT:-.}/runtime/templates/$template_name"
  )

  for path in "${search_paths[@]}"; do
    if [[ -f "$path" ]]; then
      echo "$path"
      return 0
    fi
  done

  return 1
}

# Usage in queen-init
queen-init)
  target="${AETHER_ROOT:-.}/QUEEN.md"

  template_path=$(find_template "QUEEN.md.template") || {
    json_err "$E_FILE_NOT_FOUND" \
      "QUEEN.md.template not found" \
      "Searched: .aether/templates/, ~/.aether/system/templates/, runtime/templates/" \
      "Install Aether: npm install -g aether-colony"
  }

  # Copy and customize template
  cp "$template_path" "$target"
  # ... customize ...

  json_ok "{\"created\":\"$target\"}"
  ;;
```

**Testing Strategy:**

1. **Test npm install scenario:**
   ```bash
   npm install -g .
   mkdir /tmp/test-queen && cd /tmp/test-queen
   bash ~/.aether/system/aether-utils.sh queen-init
   # Verify: QUEEN.md created successfully
   ```

2. **Test git clone scenario:**
   ```bash
   cd /path/to/aether-clone
   bash .aether/aether-utils.sh queen-init
   # Verify: QUEEN.md created successfully
   ```

3. **Test template not found:**
   ```bash
   # Temporarily rename templates
   mv .aether/templates .aether/templates.bak
   bash .aether/aether-utils.sh queen-init 2>&1 | grep "not found"
   mv .aether/templates.bak .aether/templates
   ```

**Rollback Procedures:**

```bash
git checkout HEAD -- .aether/aether-utils.sh
```

**Verification Checklist:**

- [ ] queen-init works with npm-installed Aether
- [ ] Template resolution order: .aether/ > ~/.aether/system/ > runtime/
- [ ] Clear error message if template not found
- [ ] Git clone workflow still works
- [ ] All template types use new resolution

---

#### W3-T2: Add Template Validation

**Task Description:**

Before using a template, validate that it's complete and valid. Check for required placeholders, valid structure, and completeness. This prevents using corrupted or incomplete templates.

**Step-by-Step Implementation:**

1. **Define template requirements:**
   - Required placeholders: {{COLONY_NAME}}, {{GOAL}}, etc.
   - Valid markdown structure
   - Required sections

2. **Create validation function:**
   ```bash
   validate_template() {
     local template_path="$1"
     local required_placeholders=("{{COLONY_NAME}}" "{{GOAL}}")

     # Check file exists and is readable
     [[ -r "$template_path" ]] || return 1

     # Check required placeholders
     for placeholder in "${required_placeholders[@]}"; do
       if ! grep -q "$placeholder" "$template_path"; then
         echo "Missing placeholder: $placeholder"
         return 1
       fi
     done

     # Check valid markdown (basic)
     if ! grep -q '^# ' "$template_path"; then
       echo "No markdown header found"
       return 1
     fi

     return 0
   }
   ```

3. **Integrate into queen-init:**
   ```bash
   validate_template "$template_path" || {
     json_err "$E_VALIDATION_FAILED" "Template validation failed"
   }
   ```

**Testing Strategy:**

```bash
# Test with corrupted template
echo "invalid template" > /tmp/bad.template
validate_template /tmp/bad.template && exit 1

# Test with valid template
validate_template .aether/templates/QUEEN.md.template
```

**Verification Checklist:**

- [ ] Templates validated before use
- [ ] Clear error if template is corrupted
- [ ] Tests for template validation
- [ ] Validation doesn't slow down normal operation

---

### Wave 4: Command Consolidation Infrastructure

#### Wave Overview

Wave 4 addresses one of Aether's most significant technical debt items: 13,573 lines of duplicated command definitions. Currently, every command exists in two versions—one for Claude Code (.claude/commands/ant/*.md) and one for OpenCode (.opencode/commands/ant/*.md). Any change requires manual synchronization, which inevitably leads to drift.

This wave builds infrastructure for single-source-of-truth command generation. Commands will be defined once in YAML format, then generated into platform-specific formats. This eliminates duplication and ensures consistency.

**Business Justification:**

The business case for command consolidation is compelling:

1. **Reduced Maintenance Cost:** 13,573 lines becomes ~2,000 lines of YAML. Every bug fix, enhancement, or new command requires changes in only one place.
2. **Faster Iteration:** Changes propagate immediately to both platforms. No manual synchronization delays.
3. **Consistency Guarantee:** Users get identical behavior regardless of platform. No "works in Claude but not OpenCode" issues.
4. **Easier Expansion:** Adding support for new platforms (Cursor, GitHub Copilot, etc.) becomes a matter of adding a new generator, not rewriting all commands.

The investment in consolidation infrastructure pays dividends across the entire lifecycle of the project.

**Technical Rationale:**

The technical approach uses a proven pattern:
1. **Abstract Definition:** YAML captures command semantics (what the command does)
2. **Platform Mapping:** Tool names and formats vary by platform
3. **Code Generation:** Scripts generate platform-specific implementations
4. **CI Verification:** Automated checks ensure generated code matches source

This pattern is used successfully by:
- OpenAPI for API definitions
- Protocol Buffers for serialization
- GraphQL for data fetching

The key insight is separating "what" (command semantics) from "how" (platform implementation).

**Risk Analysis:**

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Generator bugs break commands | Medium | Critical | Extensive testing, parallel operation |
| YAML schema too limiting | Low | Medium | Iterative schema evolution |
| Migration misses edge cases | Medium | High | Gradual migration, verification |
| CI check too strict | Low | Low | Allow emergency overrides |

The primary risk is that a bug in the generator could break all commands simultaneously. The mitigation is extensive testing and maintaining the ability to fall back to manual files if needed.

**Resource Requirements:**

- **Primary Skill:** Node.js/JavaScript for generator, YAML schema design
- **Secondary Skill:** Understanding of both Claude Code and OpenCode formats
- **Time:** 8 days (largest wave)
- **Tools:** Node.js, yaml parser, template engine

**Success Criteria:**

1. YAML schema supports all 22 commands
2. Generator produces identical output to current manual files
3. All 22 commands generate successfully for both platforms
4. CI check verifies commands are in sync
5. Documentation explains the system

---

#### W4-T1: Design YAML Command Schema

**Task Description:**

Design a YAML schema for command definitions that captures everything needed to generate both Claude Code and OpenCode formats. The schema must be expressive enough for complex commands like oracle and build, while remaining simple for basic commands like status.

**Schema Requirements:**

1. **Metadata:** name, description, version, author
2. **Parameters:** arguments, flags, options with types and validation
3. **Tool Mappings:** Claude tool names vs OpenCode tool names
4. **Prompt Template:** The actual command instructions
5. **Execution Steps:** Structured representation of command flow

**Step-by-Step Implementation:**

1. **Analyze existing commands:**
   - Review 5 simple commands (status, help, flags)
   - Review 5 complex commands (build, oracle, plan)
   - Identify common patterns and variations

2. **Design schema structure:**
   ```yaml
   # command.yaml structure
   command:
     metadata:
       name: string
       description: string
       version: string
       platforms: [claude, opencode]

     parameters:
       - name: string
         type: string|number|boolean|enum
         required: boolean
         default: any
         description: string

     tools:
       claude:
         bash: Bash
         read: Read
         # ... mappings
       opencode:
         bash: bash
         read: read_file
         # ... mappings

     prompt:
       template: string  # Or structured steps
       variables:
         - name: string
           source: parameter|state|context

     steps:
       - id: string
         tool: string
         command: string
         condition: string  # Optional conditional
   ```

3. **Create JSON Schema for validation:**
   ```json
   {
     "$schema": "http://json-schema.org/draft-07/schema#",
     "type": "object",
     "properties": {
       "command": {
         "type": "object",
         "properties": {
           "metadata": { "$ref": "#/definitions/metadata" },
           "parameters": { "type": "array", "items": { "$ref": "#/definitions/parameter" } },
           "tools": { "$ref": "#/definitions/tools" },
           "prompt": { "$ref": "#/definitions/prompt" }
         },
         "required": ["metadata", "prompt"]
       }
     }
   }
   ```

4. **Document the schema:**
   - Write comprehensive documentation
   - Provide examples for each command type
   - Document migration path from markdown

**Code Example:**

```yaml
# Example: status command definition
command:
  metadata:
    name: ant:status
    description: Display current colony status
    version: "1.0"
    platforms: [claude, opencode]

  parameters:
    - name: verbose
      type: boolean
      required: false
      default: false
      description: Show detailed status

  tools:
    claude:
      bash: Bash
      read: Read
    opencode:
      bash: bash
      read: read_file

  prompt:
    template: |
      You are the Queen. Display colony status.

      {{#if verbose}}
      Show detailed information including:
      - Full colony state
      - All flags
      - Recent events
      {{else}}
      Show summary:
      - Current phase
      - Goal
      - Blocker count
      {{/if}}

      Steps:
      1. Load state: {{tools.bash}} "bash .aether/aether-utils.sh load-state"
      2. Display status based on parameters

    variables:
      - name: verbose
        source: parameter
```

**Testing Strategy:**

1. **Validate schema:**
   ```bash
   node -e "const schema = require('./schema.json'); console.log('Valid JSON Schema')"
   ```

2. **Test example commands:**
   - Create YAML for 3 simple commands
   - Validate against schema
   - Verify all required fields present

**Verification Checklist:**

- [ ] YAML schema supports all 22 commands
- [ ] Schema validation passes for all command definitions
- [ ] Documentation complete
- [ ] Examples provided for simple and complex commands
- [ ] Migration path documented

---

#### W4-T2: Create Command Generator Script

**Task Description:**

Build the generate-commands.sh script that reads YAML definitions and generates both Claude and OpenCode command files. The generator must support:

- Full generation (all commands)
- Single command generation
- Dry-run mode (show what would change)
- Diff mode (compare generated vs existing)

**Step-by-Step Implementation:**

1. **Set up generator structure:**
   ```bash
   #!/bin/bash
   # bin/generate-commands.sh

   set -euo pipefail

   COMMAND_DIR="src/commands/definitions"
   OUTPUT_CLAUDE=".claude/commands/ant"
   OUTPUT_OPENCODE=".opencode/commands/ant"
   ```

2. **Implement YAML parsing:**
   - Use yq or Node.js for YAML parsing
   - Extract command metadata, parameters, prompt

3. **Implement Claude format generator:**
   ```bash
   generate_claude() {
     local yaml_file="$1"
     local output_file="$2"

     # Parse YAML
     local name=$(yq -r '.command.metadata.name' "$yaml_file")
     local description=$(yq -r '.command.metadata.description' "$yaml_file")

     # Generate markdown
     cat > "$output_file" << EOF
   ---
   name: $name
   description: "$description"
   ---

   $(yq -r '.command.prompt.template' "$yaml_file")
   EOF
   }
   ```

4. **Implement OpenCode format generator:**
   - Similar structure but different tool names
   - Map tool references appropriately

5. **Add CLI interface:**
   ```bash
   case "${1:-}" in
     --all)
       generate_all
       ;;
     --command)
       generate_single "$2"
       ;;
     --dry-run)
       DRY_RUN=1 generate_all
       ;;
     --verify)
       verify_all
       ;;
   esac
   ```

**Code Example:**

```bash
#!/bin/bash
# bin/generate-commands.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AETHER_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DEFINITIONS_DIR="$AETHER_ROOT/src/commands/definitions"
OUTPUT_CLAUDE="$AETHER_ROOT/.claude/commands/ant"
OUTPUT_OPENCODE="$AETHER_ROOT/.opencode/commands/ant"

# Generate Claude format
generate_claude() {
  local yaml_file="$1"
  local cmd_name=$(basename "$yaml_file" .yaml)
  local output_file="$OUTPUT_CLAUDE/$cmd_name.md"

  # Parse YAML (using Node.js for reliability)
  node -e "
    const yaml = require('js-yaml');
    const fs = require('fs');
    const data = yaml.load(fs.readFileSync('$yaml_file', 'utf8'));

    // Generate Claude format
    const output = [];
    output.push('---');
    output.push(\`name: \${data.command.metadata.name}\`);
    output.push(\`description: "\${data.command.metadata.description}"\`);
    output.push('---');
    output.push('');
    output.push(data.command.prompt.template);

    fs.writeFileSync('$output_file', output.join('\n'));
    console.log('Generated: $output_file');
  "
}

# Generate OpenCode format
generate_opencode() {
  local yaml_file="$1"
  local cmd_name=$(basename "$yaml_file" .yaml)
  local output_file="$OUTPUT_OPENCODE/$cmd_name.md"

  # Similar but with OpenCode tool mappings
  node -e "
    // ... with tool name mappings ...
  "
}

# Generate all commands
generate_all() {
  for yaml_file in "$DEFINITIONS_DIR"/*.yaml; do
    [[ -f "$yaml_file" ]] || continue
    generate_claude "$yaml_file"
    generate_opencode "$yaml_file"
  done
}

# Verify generated files match
diff_mode() {
  local differences=0
  for yaml_file in "$DEFINITIONS_DIR"/*.yaml; do
    local cmd_name=$(basename "$yaml_file" .yaml)

    # Compare Claude version
    if ! diff -q <(generate_claude_stdout "$yaml_file") "$OUTPUT_CLAUDE/$cmd_name.md" >/dev/null 2>&1; then
      echo "DIFF: $cmd_name (Claude)"
      differences=$((differences + 1))
    fi

    # Compare OpenCode version
    if ! diff -q <(generate_opencode_stdout "$yaml_file") "$OUTPUT_OPENCODE/$cmd_name.md" >/dev/null 2>&1; then
      echo "DIFF: $cmd_name (OpenCode)"
      differences=$((differences + 1))
    fi
  done

  return $differences
}

# Main
case "${1:-}" in
  --all) generate_all ;;
  --command) generate_claude "$DEFINITIONS_DIR/$2.yaml"; generate_opencode "$DEFINITIONS_DIR/$2.yaml" ;;
  --diff) diff_mode ;;
  --verify) diff_mode && echo "All commands match" || exit 1 ;;
  *) echo "Usage: $0 --all|--command <name>|--diff|--verify" ;;
esac
```

**Testing Strategy:**

1. **Test generation:**
   ```bash
   ./bin/generate-commands.sh --command status
   # Verify files created
   ```

2. **Test verification:**
   ```bash
   ./bin/generate-commands.sh --verify
   # Should show "All commands match" when in sync
   ```

3. **Test diff mode:**
   ```bash
   # Modify a command manually
   echo "# test" >> .claude/commands/ant/status.md
   ./bin/generate-commands.sh --diff
   # Should show the difference
   ```

**Verification Checklist:**

- [ ] Generator produces identical output to current manual files
- [ ] All 22 commands generate successfully
- [ ] CI check passes
- [ ] Generator handles tool mapping correctly
- [ ] Dry-run mode works
- [ ] Diff mode shows differences clearly

---

#### W4-T3: Migrate Commands to YAML

**Task Description:**

Convert all 22 command definitions from markdown to YAML. Start with simple commands (status, help) before complex ones (build, oracle).

**Migration Strategy:**

1. **Phase 1: Simple Commands (5 commands)**
   - status, help, flags, focus, redirect
   - Learn patterns, refine schema

2. **Phase 2: Medium Commands (10 commands)**
   - plan, build, init, continue, seal
   - Apply lessons from Phase 1

3. **Phase 3: Complex Commands (7 commands)**
   - oracle, swarm, chaos, archaeology
   - Handle complex parameter sets

**Step-by-Step for Each Command:**

1. **Read existing markdown:**
   ```bash
   cat .claude/commands/ant/status.md
   ```

2. **Extract components:**
   - Metadata (name, description)
   - Parameters (arguments, flags)
   - Prompt template
   - Tool usage patterns

3. **Create YAML:**
   ```yaml
   # src/commands/definitions/status.yaml
   command:
     metadata:
       name: ant:status
       description: "Display current colony status"
       version: "1.0"
     parameters:
       - name: verbose
         type: boolean
         default: false
     prompt:
       template: |
         # Status Command

         Display colony status...
   ```

4. **Generate and verify:**
   ```bash
   ./bin/generate-commands.sh --command status
   diff .claude/commands/ant/status.md <(./bin/generate-commands.sh --command status --stdout)
   ```

5. **Commit when verified:**
   ```bash
   git add src/commands/definitions/status.yaml
   git commit -m "Migrate status command to YAML"
   ```

**Verification Checklist:**

- [ ] All 22 commands have YAML definitions
- [ ] Generated files match current manual files
- [ ] Zero diff when comparing generated vs manual
- [ ] Schema validation passes for all
- [ ] Commands tested after migration

---

#### W4-T4: Add CI Check for Command Sync

**Task Description:**

Add a CI check that verifies generated commands match YAML source. Fail the build if they're out of sync.

**Step-by-Step Implementation:**

1. **Add npm script:**
   ```json
   // package.json
   {
     "scripts": {
       "lint:sync": "./bin/generate-commands.sh --verify"
     }
   }
   ```

2. **Add CI workflow step:**
   ```yaml
   # .github/workflows/ci.yml
   jobs:
     lint:
       steps:
         - uses: actions/checkout@v3
         - name: Verify command sync
           run: npm run lint:sync
   ```

3. **Add helpful error message:**
   ```bash
   # In generate-commands.sh verify mode
   if ! diff_mode; then
     echo ""
     echo "ERROR: Commands are out of sync with YAML definitions."
     echo "Run: ./bin/generate-commands.sh --all"
     echo "Then commit the changes."
     exit 1
   fi
   ```

**Verification Checklist:**

- [ ] CI fails if commands are out of sync
- [ ] Clear error message showing how to fix
- [ ] lint:sync script works locally
- [ ] CI passes when commands in sync

---

*Document continues with Waves 5-15 in subsequent sections...*

