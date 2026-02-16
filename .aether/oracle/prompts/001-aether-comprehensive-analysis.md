<research_objective>
Conduct an exhaustive, comprehensive analysis of the entire Aether repository to produce a massive technical report (target: 200,000+ words) that serves as the definitive reference for understanding, improving, and perfecting the Aether multi-agent CLI framework.

This research must:
1. Read and analyze EVERY file in the repository systematically
2. Document exactly how every component works with file paths and line references
3. Identify ALL bugs, issues, and problems with exact locations
4. Catalog ALL areas for improvement with severity ratings
5. Perform gap analysis against industry best practices
6. Produce a detailed, phased implementation plan to bring Aether to production-ready status

The output will guide a development team to transform Aether into a professional, sophisticated, industry-leading tool.
</research_objective>

<scope>
**Repository Root:** /Users/callumcowie/repos/Aether

**Directories to Analyze (comprehensive):**
- `.aether/` - Core system files, utilities, schemas, agents, docs
- `.claude/commands/ant/` - All Claude Code slash commands
- `.opencode/` - OpenCode commands and agents
- `bin/` - CLI tools and libraries
- `runtime/` - Auto-generated staging files
- `tests/` - All test files (unit, bash, e2e, integration)
- `docs/` - All documentation
- Root level files (package.json, README.md, etc.)

**Explicitly Exclude:**
- `node_modules/` directories
- `.git/` directory
- `.worktrees/` (note their existence but don't analyze contents)

**Depth:** Complete - every meaningful file must be read and analyzed.
</scope>

<methodology>
## Phase 1: Systematic File Discovery and Reading

### Step 1: Discover All Files
Use Glob to find all files by type:
```
Glob("**/*.md") - All markdown files
Glob("**/*.sh") - All shell scripts
Glob("**/*.js") - All JavaScript files
Glob("**/*.json") - All JSON files
Glob("**/*.yaml") - All YAML files
Glob("**/*.xsd") - All XSD schema files
Glob("**/*.xml") - All XML files
Glob("**/*.xsl") - All XSLT files
```

### Step 2: Read Every File
For each file discovered:
- Read the complete content using the Read tool
- Document the file's purpose and role in the system
- Note line counts and key structures
- Cross-reference with other files

**CRITICAL:** Do not skip files. The comprehensiveness of this analysis depends on reading everything.

### Step 3: Categorize Files
As you read, categorize each file:
- Core System (utilities, state management)
- Command Interface (slash commands)
- Agent Definitions (worker definitions)
- Configuration (schemas, settings)
- Documentation (guides, references)
- Tests (unit, integration, e2e)
- Build/Deploy (npm scripts, sync tools)
</methodology>

<analysis_requirements>
## Section 1: Complete Architecture Analysis

For each major component, document:

### 1.1 Core Utility Layer (.aether/aether-utils.sh)
- Every function (document purpose, inputs, outputs, line numbers)
- Error handling patterns
- State management functions
- File locking mechanisms
- Pheromone system implementation
- XML utility integration points
- Dependencies between functions

### 1.2 Command System
**Claude Code Commands (.claude/commands/ant/):**
- List all 34 commands
- For each command: purpose, usage, implementation approach
- Common patterns across commands
- Dependencies on aether-utils.sh

**OpenCode Commands (.opencode/commands/ant/):**
- List all commands
- Compare with Claude Code equivalents
- Identify duplication

### 1.3 Worker/Agent System
- All 22 castes and their definitions
- How workers are spawned and managed
- Spawn tree tracking
- Caste-to-model routing (current status)
- Worker priming system

### 1.4 State Management
- COLONY_STATE.json structure and lifecycle
- Pheromone signal system (FOCUS, REDIRECT, FEEDBACK)
- Checkpoint system
- Session freshness detection
- Activity logging

### 1.5 XML Infrastructure
- All 5 XSD schemas (detailed breakdown)
- XML utility functions
- XInclude composition system
- Security measures (XXE protection)
- Integration points with JSON
- Current usage status (dormant vs active)

### 1.6 Distribution/Hub System
- npm package structure
- Sync mechanism (.aether/ → runtime/ → ~/.aether/)
- Update system
- Version management

## Section 2: Bug and Issue Identification

For EVERY bug or issue found, document:
- **Exact file path**
- **Line number(s)**
- **Severity** (Critical, High, Medium, Low)
- **Description** of the problem
- **Impact** on functionality
- **Root cause** analysis
- **Suggested fix** with code example

**Known Issues to Verify:**
- BUG-005/BUG-011: Lock deadlock in flag-auto-resolve
- ISSUE-004: Template path hardcoded to runtime/
- BUG-007: Error code inconsistency
- Model routing unverified
- Any new issues discovered

## Section 3: Improvement Opportunities

Catalog ALL areas for improvement:

### Code Quality
- Duplicate code (13K lines between Claude/OpenCode)
- Function length and complexity
- Error handling consistency
- Naming conventions
- Comment coverage

### Architecture
- Design patterns used/missing
- Coupling and cohesion issues
- Abstraction levels
- Extension points

### Performance
- Potential bottlenecks
- Inefficient algorithms
- Resource usage patterns
- Caching opportunities

### Security
- Input validation gaps
- Injection vulnerabilities
- Permission issues
- Secret handling

### Testing
- Coverage gaps
- Test quality issues
- Missing test scenarios
- Flaky tests

### Documentation
- Missing documentation
- Outdated docs
- Unclear explanations
- Formatting inconsistencies

## Section 4: Industry Best Practice Gap Analysis

Compare Aether against:
- CLI tool best practices (12-factor CLI, etc.)
- Multi-agent system patterns
- Node.js/npm package standards
- Shell scripting standards
- Documentation standards
- Testing best practices
- Security hardening guides

For each gap:
- Current state
- Industry standard
- Impact of not addressing
- Effort to fix
</analysis_requirements>

<implementation_plan_requirements>
Create a comprehensive implementation plan with the following structure:

## Wave Structure

Organize work into logical WAVES (groups of related tasks). Each wave should:
- Have a clear theme/purpose
- Contain 3-10 specific tasks
- Include dependencies on previous waves
- Have clear success criteria

**Minimum 10 waves expected**, potentially 20+ for comprehensive coverage.

### For Each Task, Document:

```
Task ID: W1-T1 (Wave 1, Task 1)
Title: Clear, actionable title
Description: Detailed explanation of what needs to be done
Files to Modify: Exact file paths
Files to Create: Exact file paths (if any)
Dependencies: List of task IDs that must complete first
Estimated Effort: Small/Medium/Large
Priority: P0 (Critical) / P1 (High) / P2 (Medium) / P3 (Low)
Success Criteria: Measurable conditions for completion
Verification Steps: How to confirm the fix works
Risk Assessment: What could go wrong
Rollback Plan: How to undo if needed
```

### Wave Categories (suggested):

1. **Foundation Fixes** - Critical bugs, security issues
2. **Code Consolidation** - Remove duplication, unify patterns
3. **XML System Activation** - Integrate XML into production flows
4. **Testing Expansion** - Fill coverage gaps, add integration tests
5. **Documentation Overhaul** - Consolidate, update, professionalize
6. **Performance Optimization** - Bottleneck removal, caching
7. **Error Handling Standardization** - Consistent patterns, better messages
8. **State Management Improvements** - Reliability, debugging
9. **Developer Experience** - Tooling, debugging, onboarding
10. **Release Hardening** - CI/CD, versioning, distribution
11. **Feature Completion** - Model routing (if unblocked), unverified features
12. **Polish and Refinement** - Code style, consistency, final touches

## Detailed Plan Requirements

For the implementation plan section:

1. **Executive Summary** - One-page overview of the entire plan
2. **Wave Dependencies Graph** - Visual/text representation of what depends on what
3. **Critical Path Analysis** - Minimum sequence to reach production-ready
4. **Risk Analysis** - High-risk tasks and mitigation strategies
5. **Resource Requirements** - Skills needed, estimated total effort
6. **Milestone Definitions** - Clear checkpoints for progress tracking
7. **Definition of Done** - What "operating perfectly" means for each component
</implementation_plan_requirements>

<output_structure>
Produce a single massive technical report saved to:
**`.aether/oracle/AETHER-COMPREHENSIVE-ANALYSIS-REPORT.md`**

## Report Structure:

```markdown
# Aether Comprehensive Analysis Report
**Generated:** [Timestamp]
**Confidence Level:** [X]%
**Files Analyzed:** [N]
**Total Lines of Code:** [N]

---

## Executive Summary
- One-page overview of findings
- Key statistics
- Critical issues summary
- Top recommendations

---

## Table of Contents
[Detailed TOC with links to sections]

---

## 1. Repository Overview
### 1.1 Statistics
### 1.2 Directory Structure
### 1.3 Technology Stack
### 1.4 Architecture Overview

---

## 2. Component Analysis
### 2.1 Core Utility Layer
[Deep dive into aether-utils.sh - every function documented]

### 2.2 Command System
[All commands analyzed]

### 2.3 Worker/Agent System
[22 castes documented]

### 2.4 State Management
[All state mechanisms]

### 2.5 XML Infrastructure
[Complete XSD and utility analysis]

### 2.6 Distribution System
[npm, hub, sync mechanisms]

---

## 3. Bug and Issue Catalog
### 3.1 Critical Issues (P0)
### 3.2 High Priority Issues (P1)
### 3.3 Medium Priority Issues (P2)
### 3.4 Low Priority Issues (P3)

---

## 4. Improvement Opportunities
### 4.1 Code Quality
### 4.2 Architecture
### 4.3 Performance
### 4.4 Security
### 4.5 Testing
### 4.6 Documentation

---

## 5. Industry Best Practice Gap Analysis
### 5.1 CLI Tool Standards
### 5.2 Multi-Agent Patterns
### 5.3 Node.js/npm Standards
### 5.4 Shell Scripting Standards
### 5.5 Documentation Standards
### 5.6 Testing Standards
### 5.7 Security Standards

---

## 6. Implementation Plan
### 6.1 Executive Summary
### 6.2 Wave Overview
### 6.3 Detailed Wave Breakdown
[W1 through WN - each with all tasks fully detailed]
### 6.4 Dependency Graph
### 6.5 Critical Path
### 6.6 Risk Analysis
### 6.7 Resource Requirements
### 6.8 Definition of Done

---

## 7. Appendices
### A. File Inventory
[Complete list of all files analyzed]

### B. Function Reference
[All functions with signatures]

### C. Schema Reference
[All XSD schemas documented]

### D. Test Inventory
[All tests catalogued]

### E. Glossary
[Terms and definitions]

### F. References
[External resources consulted]
```
</output_structure>

<research_execution>
## Iteration Strategy

This research will require multiple iterations. For each iteration:

1. **Select a focus area** (e.g., "Core utilities", "Command system", etc.)
2. **Read all relevant files** for that area
3. **Document findings** in detail
4. **Append to the report** (never overwrite previous work)
5. **Rate confidence** for that section
6. **Continue to next area** until all sections are complete

**Target:** 50 iterations minimum for comprehensive coverage

## Tool Usage Strategy

For maximum efficiency, invoke all relevant tools simultaneously:
- Use Glob to find files in parallel
- Read multiple independent files in parallel
- Search across files with Grep when looking for patterns
- Run verification commands with Bash

## Progress Tracking

After each iteration, append a progress summary:
```markdown
## Iteration N: [Focus Area]
- Files analyzed: [N]
- Functions documented: [N]
- Issues found: [N critical, N high, N medium, N low]
- Confidence in this section: [X]%
- Notes for next iteration: [what to focus on next]
```
</research_execution>

<verification>
Before declaring the research complete, verify:

1. **Coverage Check:**
   - [ ] All directories in scope have been analyzed
   - [ ] No major file types were skipped
   - [ ] Both code and documentation reviewed

2. **Quality Check:**
   - [ ] Every bug/issue has exact file path and line number
   - [ ] Every recommendation is actionable
   - [ ] Implementation plan has sufficient detail to execute

3. **Completeness Check:**
   - [ ] Report exceeds 100,000 words (aim for 200,000)
   - [ ] All 5 research questions from research.json are answered
   - [ ] Implementation plan has at least 10 waves

4. **Accuracy Check:**
   - [ ] Cross-reference findings with actual files
   - [ ] Verify file paths are correct
   - [ ] Confirm line numbers match reality
</verification>

<success_criteria>
The research is complete when:

1. **Report Size:** Comprehensive report saved to `.aether/oracle/AETHER-COMPREHENSIVE-ANALYSIS-REPORT.md` with 200,000+ words

2. **Coverage:** Every meaningful file in the repository has been read and analyzed

3. **Bug Catalog:** Complete inventory of all bugs/issues with exact locations and fixes

4. **Improvement Catalog:** Exhaustive list of all improvement opportunities with severity ratings

5. **Implementation Plan:** Detailed wave-based plan with at least 10 waves, specific tasks, dependencies, and success criteria

6. **Confidence Level:** 99%+ confidence that the analysis is thorough and accurate

7. **Actionability:** The report could be handed to a development team and used to bring Aether to production-ready status

8. **Completion Signal:** Output `<oracle>COMPLETE</oracle>` when done
</success_criteria>

<thinking_instructions>
Thoroughly analyze each component by:
- Reading code line-by-line for critical files
- Tracing execution paths through function calls
- Identifying patterns and anti-patterns
- Comparing against industry best practices
- Considering edge cases and failure modes
- Documenting not just what IS, but what SHOULD BE

Deeply consider:
- The user's goal of "operating perfectly"
- What "production-ready" means for a CLI framework
- The balance between ideal architecture and practical constraints
- Dependencies between different parts of the system

Explore multiple solutions for each problem:
- Quick fixes vs. proper refactors
- Local changes vs. architectural changes
- Different implementation approaches with trade-offs
</thinking_instructions>
