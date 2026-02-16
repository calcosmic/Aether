# Expanded Worker/Agent System Documentation

## Executive Summary

The Aether colony implements a sophisticated multi-caste worker system with 22 distinct castes, each specializing in different aspects of software development. The system uses a biological metaphor (ants, colonies, castes) to organize work, with structured spawn trees, depth-based delegation limits, and a (currently non-functional) model routing system.

This document provides exhaustively detailed documentation for each of the 22 castes, the spawn system architecture, worker lifecycle, communication patterns, error handling, and the model routing system.

---

## Table of Contents

1. [Caste Catalog Overview](#caste-catalog-overview)
2. [Core Castes (7)](#core-castes)
3. [Development Cluster - Weaver Ants (4)](#development-cluster)
4. [Knowledge Cluster - Leafcutter Ants (4)](#knowledge-cluster)
5. [Quality Cluster - Soldier Ants (4)](#quality-cluster)
6. [Special Castes (3)](#special-castes)
7. [Surveyor Sub-Castes (4)](#surveyor-sub-castes)
8. [Spawn System Architecture](#spawn-system-architecture)
9. [Worker Lifecycle](#worker-lifecycle)
10. [Communication Patterns](#communication-patterns)
11. [Error Handling in Workers](#error-handling-in-workers)
12. [Model Routing System](#model-routing-system)
13. [Worker Priming System](#worker-priming-system)

---

## Caste Catalog Overview

The Aether colony organizes work through 22 specialized castes, each with distinct responsibilities, capabilities, and behavioral patterns. The castes are organized into five clusters based on their primary function:

| Cluster | Castes | Primary Function |
|---------|--------|------------------|
| Core | Queen, Builder, Watcher, Scout, Colonizer, Architect, Route-Setter | Primary development workflow |
| Development (Weaver Ants) | Weaver, Probe, Ambassador, Tracker | Code quality and maintenance |
| Knowledge (Leafcutter Ants) | Chronicler, Keeper, Auditor, Sage | Documentation and learning |
| Quality (Soldier Ants) | Guardian, Measurer, Includer, Gatekeeper | Security and compliance |
| Special | Archaeologist, Oracle, Chaos | Specialized investigations |
| Surveyor | Disciplines, Nest, Pathogens, Provisions | Codebase intelligence |

---

## Core Castes

### 1. Queen 👑🐜

#### Caste Overview

The Queen is the colony orchestrator and coordinator, serving as the central nervous system of the Aether colony. Unlike other castes that perform specific technical tasks, the Queen exists at the meta-level, managing the overall flow of work, maintaining colony state, and ensuring that all activities align with the colony's goals. The Queen operates at spawn depth 0, making it the root of all spawn trees and the ultimate authority on phase boundaries and colony-wide decisions.

The Queen embodies the colony's collective intelligence, synthesizing outputs from all other castes and making decisions about when to advance phases, when to spawn additional workers, and when to escalate issues. The Queen's perspective is holistic, viewing the codebase not as individual files or functions but as an interconnected ecosystem where changes in one area can have cascading effects throughout the system.

The Queen's role is not to implement code directly but to create the conditions under which implementation can succeed. This involves setting clear intentions, establishing constraints through pheromone signals, and maintaining the colony's memory across sessions. The Queen is the only caste that can legitimately claim to "understand" the entire project state at any given moment.

#### Role and Responsibilities

The Queen's responsibilities span the entire lifecycle of a colony session:

**Intention Setting**: The Queen establishes the colony's goal and ensures all workers understand the north star they're working toward. This involves translating user requests into actionable technical objectives and communicating these objectives in terms that each caste can understand and act upon.

**State Management**: The Queen maintains the colony's state in `.aether/data/COLONY_STATE.json`, tracking the current phase, completed work, pending tasks, and any blockers or issues that need attention. State management includes updating the CONTEXT.md file to provide a human-readable summary of the colony's current status.

**Worker Dispatch**: The Queen decides which castes to spawn for each phase of work, based on an analysis of the task requirements. This decision involves considering the nature of the work (implementation, research, verification), the current state of the codebase, and any constraints or pheromone signals that might influence the approach.

**Phase Boundary Control**: The Queen controls when phases begin and end, ensuring that work proceeds in a logical sequence and that each phase's success criteria are met before advancing. This includes running verification commands, checking for blockers, and synthesizing reports from spawned workers.

**Learning Extraction**: The Queen extracts patterns and learnings from each phase, promoting valuable insights to the global learning database for use in future projects.

#### Capabilities and Tools

The Queen has access to all colony management utilities:

- **State Operations**: `validate-state`, `load-state`, `unload-state` for managing COLONY_STATE.json
- **Activity Logging**: `activity-log`, `activity-log-init`, `activity-log-read` for tracking colony actions
- **Spawn Management**: `spawn-log`, `spawn-complete`, `spawn-can-spawn` for worker orchestration
- **Context Management**: `context-update` for maintaining CONTEXT.md
- **Flag Management**: `flag-add`, `flag-resolve`, `flag-list` for tracking blockers
- **Learning Management**: `learning-promote`, `learning-inject` for knowledge preservation

#### When to Use This Caste

The Queen is automatically invoked by colony initialization commands (`/ant:init`, `/ant:colonize`). Users do not manually spawn Queens; instead, the Queen emerges at the start of each colony session and persists throughout.

#### Example Tasks

- Initialize a new colony with a specific goal
- Coordinate a multi-phase build operation
- Synthesize results from multiple worker castes
- Advance the colony through milestone progression
- Handle colony-wide blockers and escalations

#### Spawn Patterns

The Queen spawns at depth 0 and can spawn up to 4 direct children at depth 1. Typical spawn patterns include:

```
Queen (depth 0)
├── Prime Builder (depth 1)
├── Prime Watcher (depth 1)
├── Route-Setter (depth 1)
└── Scout (depth 1)
```

#### State Management

The Queen maintains state through:
- **COLONY_STATE.json**: Primary state file tracking goal, phases, errors, events
- **CONTEXT.md**: Human-readable context document
- **constraints.json**: Pheromone signals (focus, redirect, feedback)
- **flags.json**: Blockers and issues

#### Model Assignment

The Queen does not have a model assignment because it operates as an orchestrator rather than a worker. All Queen operations use the default model of the parent session.

---

### 2. Builder 🔨🐜

#### Caste Overview

The Builder is the colony's hands, responsible for implementing code, executing commands, and manipulating files to achieve concrete outcomes. Builders are the most frequently spawned caste, as they perform the actual work of writing software. A Builder approaches each task with a pragmatic, action-focused mindset, prioritizing working solutions over theoretical perfection while maintaining high standards for code quality.

Builders embody the TDD (Test-Driven Development) philosophy, following a strict discipline of writing failing tests before implementation, verifying those tests fail for the right reasons, then writing minimal code to make them pass. This approach ensures that Builders never write code without a clear specification of what that code should do.

The Builder's mindset is one of constructive pragmatism. They understand that code is a means to an end, not an end in itself, and they focus on creating solutions that work, can be maintained, and can be verified. Builders are comfortable with ambiguity at the start of a task but work to eliminate that ambiguity through tests and clear acceptance criteria.

#### Role and Responsibilities

**Implementation**: Builders write code to implement features, fix bugs, and create infrastructure. They work across all layers of the stack, from database schemas to UI components, adapting their approach to the specific requirements of each task.

**Test-Driven Development**: Builders follow the RED-VERIFY RED-GREEN-VERIFY GREEN-REFACTOR cycle, ensuring that every line of production code is justified by a failing test that now passes.

**Debugging**: When things go wrong, Builders practice systematic debugging, tracing errors to their root cause rather than applying surface-level fixes. They follow the 3-Fix Rule: if three attempted fixes fail, they escalate with an architectural concern.

**Command Execution**: Builders execute shell commands, run build tools, and interact with the development environment to accomplish their tasks.

**File Manipulation**: Builders create, modify, and delete files as needed, always working within the constraints of the project's structure and conventions.

#### Capabilities and Tools

Builders have full access to the codebase and development environment:

- **File Operations**: Read, Write, Edit tools for file manipulation
- **Search Tools**: Grep, Glob for finding code and patterns
- **Execution**: Bash tool for running commands
- **Web Access**: WebSearch, WebFetch for documentation lookup
- **Utilities**: All `aether-utils.sh` commands for logging and state management

#### When to Use This Caste

Spawn a Builder when you need to:
- Implement a new feature or function
- Fix a bug with a clear reproduction case
- Create or modify configuration files
- Write scripts or automation
- Refactor code (though Weaver is preferred for pure refactoring)

#### Example Tasks

- "Implement user authentication with JWT tokens"
- "Create a React component for the dashboard header"
- "Add database migration for the new orders table"
- "Fix the off-by-one error in the pagination logic"
- "Set up ESLint configuration for the project"

#### Spawn Patterns

Builders typically spawn at depth 1 as Prime Builders, then may spawn additional Builders at depth 2 for parallel work:

```
Queen (depth 0)
└── Prime Builder (depth 1)
    ├── Builder A (depth 2) - Implement auth controller
    ├── Builder B (depth 2) - Implement auth middleware
    └── Watcher (depth 2) - Verify implementation
```

#### State Management

Builders maintain local state through:
- **Activity Log**: Each action logged with `activity-log`
- **Spawn Tree**: Spawn relationships tracked in `spawn-tree.txt`
- **TDD State**: Current cycle (RED/RED-VERIFIED/GREEN/GREEN-VERIFIED/REFACTOR)
- **Fix Count**: Tracking the 3-Fix Rule

#### Model Assignment

- **Assigned Model**: kimi-k2.5
- **Strengths**: Code generation, refactoring, multimodal capabilities
- **Best For**: Implementation tasks, code writing, visual coding from screenshots
- **Benchmark**: 76.8% SWE-Bench Verified, 256K context

---

### 3. Watcher 👁️🐜

#### Caste Overview

The Watcher is the colony's guardian, responsible for validation, testing, and quality assurance. Watchers embody vigilance and skepticism, approaching every claim of completion with the attitude of "prove it." The Watcher's Iron Law is absolute: no completion claims without fresh verification evidence.

Watchers serve as the quality gate for the colony, ensuring that no phase advances until the work meets the required standards. They are not satisfied with "should work" or "looks good" - they require verified claims with proof. This makes Watchers essential for maintaining code quality and preventing technical debt from accumulating.

The Watcher's mindset is observational and careful. They read code not to understand how it works but to find how it might fail. They run tests not to see them pass but to catch the edge cases that developers missed. They review implementations not to praise good ideas but to identify risks and vulnerabilities.

#### Role and Responsibilities

**Verification**: Watchers verify that implementations meet their specifications through execution, not inspection. They run tests, check builds, and validate that code actually works as claimed.

**Quality Assessment**: Watchers score implementations across multiple dimensions: Correctness, Completeness, Quality, Safety, and Integration. They provide numeric scores (0-10) with detailed justification.

**Execution Verification**: Before assigning any quality score, Watchers MUST attempt to execute the code through syntax checks, import checks, launch tests, and test suite runs. If any execution check fails, the quality score cannot exceed 6/10.

**Specialist Modes**: Watchers activate different specialist modes based on context:
- **Security Mode**: Auth, input validation, secrets, dependencies
- **Performance Mode**: Complexity, queries, memory, caching
- **Quality Mode**: Readability, conventions, error handling
- **Coverage Mode**: Happy path, edge cases, regressions

**Flag Creation**: When verification fails, Watchers create persistent flags (blockers) that must be resolved before phase advancement.

#### Capabilities and Tools

Watchers have access to all verification tools:

- **Execution**: Bash tool for running tests, builds, and checks
- **File Analysis**: Read, Grep for code review
- **State Management**: `flag-add`, `flag-resolve` for blocker tracking
- **Logging**: `activity-log` for verification activities

#### When to Use This Caste

Spawn a Watcher when you need to:
- Verify implementation quality before phase advancement
- Run security audits on new code
- Check test coverage and identify gaps
- Validate that acceptance criteria are met
- Review code for adherence to standards

#### Example Tasks

- "Verify that the auth implementation passes all security checks"
- "Run the test suite and report coverage metrics"
- "Check for exposed secrets in the new configuration"
- "Validate that the API endpoints handle errors correctly"
- "Review the database migration for safety"

#### Spawn Patterns

Watchers are typically spawned by Builders or the Queen for verification:

```
Prime Builder (depth 1)
└── Watcher (depth 2) - Verify implementation
```

#### State Management

Watchers track:
- **Verification Results**: Syntax, import, launch, test results
- **Quality Scores**: Per-dimension and overall scores
- **Flags Created**: Blockers that must be resolved
- **Execution Evidence**: Command outputs and exit codes

#### Model Assignment

- **Assigned Model**: kimi-k2.5
- **Strengths**: Validation, testing, visual regression testing
- **Best For**: Verification, test coverage analysis, multimodal checks
- **Context Window**: 256K tokens, multimodal capable

---

### 4. Scout 🔍🐜

#### Caste Overview

The Scout is the colony's researcher, responsible for gathering information, searching documentation, and retrieving context. Scouts embody curiosity and thoroughness, venturing into unknown territory to bring back knowledge that the colony needs to make informed decisions.

Scouts are explorers by nature. They don't implement solutions; they map the landscape of possibilities, identifying patterns, best practices, and potential pitfalls. A Scout's value lies not in what they build but in what they discover and communicate.

The Scout's mindset is discovery-focused. They approach each research task with a plan: what sources to check, what keywords to search for, and how to validate the information they find. They are comfortable with uncertainty and skilled at synthesizing fragmented information into coherent findings.

#### Role and Responsibilities

**Research Planning**: Scouts plan their research approach before executing, identifying sources, keywords, and validation strategies.

**Information Gathering**: Scouts use Grep, Glob, Read, WebSearch, and WebFetch to gather information from the codebase, documentation, and external sources.

**Pattern Discovery**: Scouts identify patterns in code and documentation, noting conventions, anti-patterns, and best practices.

**Synthesis**: Scouts synthesize findings into actionable knowledge, providing clear recommendations for next steps.

**Parallel Research**: Scouts may spawn additional Scouts for parallel research into different domains.

#### Capabilities and Tools

Scouts have broad access to information sources:

- **Codebase Search**: Grep, Glob, Read for internal research
- **Web Research**: WebSearch, WebFetch for external documentation
- **Execution**: Bash for running git commands and exploration scripts
- **Logging**: `activity-log` for research activities

#### When to Use This Caste

Spawn a Scout when you need to:
- Research an unfamiliar technology or library
- Find examples of how to implement a pattern
- Understand existing code before modifying it
- Gather documentation for a new API
- Explore the structure of a legacy codebase

#### Example Tasks

- "Research how to implement OAuth2 authentication in Node.js"
- "Find all usages of the deprecated API in our codebase"
- "Discover the testing patterns used in this project"
- "Research best practices for React hooks"
- "Explore the database schema to understand data relationships"

#### Spawn Patterns

Scouts can spawn other Scouts for parallel research:

```
Prime Scout (depth 1)
├── Scout A (depth 2) - Research documentation
└── Scout B (depth 2) - Research code examples
```

#### State Management

Scouts track:
- **Research Plan**: Sources, keywords, validation strategy
- **Findings**: Key facts, code examples, best practices, gotchas
- **Sources**: URLs and file paths consulted
- **Recommendations**: Clear next steps

#### Model Assignment

- **Assigned Model**: kimi-k2.5
- **Strengths**: Parallel exploration via agent swarm, broad research
- **Best For**: Documentation lookup, pattern discovery, wide exploration
- **Benchmark**: Can coordinate 1,500 simultaneous tool calls

---

### 5. Colonizer 🗺️🐜

#### Caste Overview

The Colonizer is the colony's explorer, responsible for codebase exploration and mapping. While Scouts research specific questions, Colonizers map entire territories, building semantic understanding of codebases, detecting patterns, and identifying dependencies.

Colonizers are cartographers of code. They don't just find answers; they create maps that others can use to navigate. A Colonizer's output is not a single finding but a comprehensive understanding of structure, patterns, and relationships.

The Colonizer's mindset is mapping-focused. They approach a codebase like an explorer approaching unknown territory, systematically charting the landscape and noting landmarks. They are methodical and thorough, ensuring that no significant area goes unexplored.

#### Role and Responsibilities

**Codebase Exploration**: Colonizers explore codebases using Glob, Grep, and Read to understand structure and organization.

**Pattern Detection**: Colonizers identify architecture patterns, naming conventions, and anti-patterns.

**Dependency Mapping**: Colonizers map dependencies, including imports, call chains, and data flow.

**Semantic Understanding**: Colonizers build a semantic understanding of what different parts of the codebase do and how they relate.

**Reporting**: Colonizers report findings for use by other castes, particularly Route-Setters who need to understand the codebase before planning.

#### Capabilities and Tools

Colonizers have access to exploration tools:

- **Exploration**: Glob, Grep, Read for codebase mapping
- **Analysis**: Bash for running analysis scripts
- **Logging**: `activity-log` for exploration activities

#### When to Use This Caste

Spawn a Colonizer when you need to:
- Map a new or unfamiliar codebase
- Understand the architecture of a legacy system
- Identify patterns before planning changes
- Document codebase structure for other developers

#### Example Tasks

- "Map the structure of this microservices codebase"
- "Identify the data flow patterns in this React application"
- "Chart the dependency graph of this Node.js project"
- "Explore the testing structure and conventions"

#### Spawn Patterns

Colonizers are typically spawned by Route-Setters before planning:

```
Route-Setter (depth 1)
└── Colonizer (depth 2) - Map codebase before planning
```

#### State Management

Colonizers track:
- **Structure Map**: Directory layout and organization
- **Pattern Inventory**: Architecture patterns identified
- **Dependency Graph**: Import and call relationships
- **Anti-Pattern List**: Concerning patterns found

#### Model Assignment

- **Assigned Model**: kimi-k2.5
- **Strengths**: Visual coding, environment setup
- **Best For**: Codebase mapping, dependency analysis, UI/prototype generation
- **Multimodal**: Can process visual inputs alongside text

---

### 6. Architect 🏛️🐜

#### Caste Overview

The Architect is the colony's wisdom keeper, responsible for synthesizing knowledge, extracting patterns, and coordinating documentation. While Builders create code and Scouts gather information, Architects organize and preserve that knowledge for future use.

Architects are pattern recognizers and structure creators. They take fragmented information and create coherent frameworks that others can understand and use. An Architect's value lies in making the complex comprehensible and the implicit explicit.

The Architect's mindset is systematic and pattern-focused. They look for the underlying structure in apparent chaos, identifying principles that can guide future decisions. They are comfortable with abstraction and skilled at creating mental models.

#### Role and Responsibilities

**Knowledge Organization**: Architects analyze what knowledge needs organizing and create structures to contain it.

**Pattern Extraction**: Architects extract success patterns, failure patterns, preferences, and constraints from colony activities.

**Synthesis**: Architects synthesize information into coherent structures with clear hierarchies and relationships.

**Documentation Coordination**: Architects coordinate documentation efforts, ensuring consistency and completeness.

**Decision Organization**: Architects organize decision rationale, making the "why" behind choices explicit and accessible.

#### Capabilities and Tools

Architects have access to documentation and analysis tools:

- **Documentation**: Write, Edit for creating structured documents
- **Analysis**: Read, Grep for pattern identification
- **Organization**: `learning-promote` for preserving patterns

#### When to Use This Caste

Spawn an Architect when you need to:
- Create comprehensive documentation from scattered notes
- Extract patterns from successful (or failed) approaches
- Organize decision rationale for future reference
- Synthesize research findings into actionable guidance

#### Example Tasks

- "Synthesize the authentication patterns we've used across projects"
- "Create a decision record for our database choice"
- "Extract testing patterns from our best projects"
- "Organize the learning from this phase for future colonies"

#### Spawn Patterns

Architects rarely spawn sub-workers because synthesis work is usually atomic:

```
Queen (depth 1)
└── Architect (depth 2) - Synthesize phase learnings
```

#### State Management

Architects track:
- **Patterns Extracted**: Success and failure patterns identified
- **Structures Created**: Documentation hierarchies built
- **Synthesis Summary**: Overall findings and recommendations

#### Model Assignment

- **Assigned Model**: glm-5
- **Strengths**: Long-context synthesis, pattern extraction, complex documentation
- **Best For**: Synthesizing knowledge, coordinating docs, pattern recognition
- **Benchmark**: 744B MoE, 200K context, strong execution with guidance

---

### 7. Route-Setter 📋🐜

#### Caste Overview

The Route-Setter is the colony's planner, responsible for creating structured phase plans, breaking down goals into achievable tasks, and analyzing dependencies. Route-Setters are the bridge between high-level goals and executable work, translating intentions into roadmaps.

Route-Setters are masters of decomposition. They take complex, ambiguous goals and break them down into concrete, actionable steps. A Route-Setter's plan is not just a list of tasks; it's a structured journey with clear milestones, dependencies, and success criteria.

The Route-Setter's mindset is planning-focused. They think in terms of sequences, dependencies, and critical paths. They are detail-oriented, ensuring that every task has clear acceptance criteria and that the path from start to finish is well-defined.

#### Role and Responsibilities

**Goal Analysis**: Route-Setters analyze goals to understand success criteria, milestones, and dependencies.

**Phase Structuring**: Route-Setters create phase structures with 3-6 phases, each with observable outcomes.

**Task Decomposition**: Route-Setters break down phases into bite-sized tasks (2-5 minutes each) with exact file paths and expected outputs.

**Dependency Analysis**: Route-Setters identify dependencies between tasks and phases, ensuring logical sequencing.

**TDD Integration**: Route-Setters incorporate TDD flow into planning, specifying tests before implementation.

#### Capabilities and Tools

Route-Setters have access to planning tools:

- **Exploration**: May spawn Colonizers to understand codebase before planning
- **Documentation**: Write for creating structured plans
- **Research**: May spawn Scouts for domain research

#### When to Use This Caste

Spawn a Route-Setter when you need to:
- Create a structured plan for a complex goal
- Break down a large feature into phases
- Analyze dependencies before starting work
- Create a roadmap for a multi-step project

#### Example Tasks

- "Create a 6-phase plan for implementing user authentication"
- "Break down the database migration into executable tasks"
- "Plan the refactoring of the monolith into microservices"
- "Create a roadmap for the v2.0 release"

#### Spawn Patterns

Route-Setters may spawn Colonizers and Scouts:

```
Queen (depth 1)
└── Route-Setter (depth 2)
    ├── Colonizer (depth 3) - Map codebase
    └── Scout (depth 3) - Research patterns
```

#### State Management

Route-Setters produce:
- **Phase Structure**: Numbered phases with names and descriptions
- **Task Lists**: Bite-sized tasks with file paths and steps
- **Success Criteria**: Observable outcomes for each phase
- **Dependency Graph**: Task and phase dependencies

#### Model Assignment

- **Assigned Model**: kimi-k2.5
- **Strengths**: Structured planning, large context for understanding codebases
- **Best For**: Breaking down goals, creating phase structures, dependency analysis
- **Benchmark**: 256K context, 76.8% SWE-Bench, strong at structured output

---

## Development Cluster

### 8. Weaver 🔄🐜

#### Caste Overview

The Weaver is the colony's refactoring specialist, responsible for transforming tangled code into clean patterns without changing behavior. Weavers are the surgeons of the codebase, performing precise operations that improve structure while preserving function.

Weavers understand that code is read more often than it's written, and they optimize for readability and maintainability. They are experts at identifying code smells and applying proven refactoring techniques to eliminate them.

The Weaver's mindset is transformational. They see code not as it is but as it could be, envisioning cleaner structures and clearer abstractions. They are methodical and careful, ensuring that every change preserves behavior.

#### Role and Responsibilities

**Code Analysis**: Weavers analyze target code to understand its current structure and identify improvement opportunities.

**Restructuring Planning**: Weavers plan restructuring steps, choosing appropriate refactoring techniques for each issue.

**Incremental Execution**: Weavers execute changes in small increments, verifying that tests pass after each change.

**Behavior Preservation**: Weavers ensure that refactoring never changes behavior - tests must pass before and after.

**Coverage Maintenance**: Weavers maintain test coverage during refactoring, aiming for 80%+ coverage.

#### Capabilities and Tools

Weavers have access to refactoring tools:

- **Refactoring Techniques**: Extract Method/Class, Inline, Rename, Move, Replace Conditional with Polymorphism
- **Code Analysis**: Read, Grep for understanding code structure
- **Execution**: Bash for running tests and verification

#### When to Use This Caste

Spawn a Weaver when you need to:
- Refactor legacy code to improve maintainability
- Extract methods or classes from large functions
- Rename variables, methods, or classes for clarity
- Eliminate code duplication
- Simplify complex conditionals

#### Example Tasks

- "Refactor the 200-line auth function into smaller methods"
- "Extract the payment logic into a separate service class"
- "Rename confusing variable names to be more descriptive"
- "Eliminate duplication between these two components"

#### Spawn Patterns

Weavers may spawn additional Weavers for large refactoring efforts:

```
Prime Weaver (depth 1)
├── Weaver A (depth 2) - Refactor module A
└── Weaver B (depth 2) - Refactor module B
```

#### State Management

Weavers track:
- **Complexity Metrics**: Before and after measurements
- **Duplication Eliminated**: Lines of duplicate code removed
- **Methods Extracted**: New methods created
- **Patterns Applied**: Refactoring techniques used

#### Model Assignment

No specific model assigned; inherits default.

---

### 9. Probe 🧪🐜

#### Caste Overview

The Probe is the colony's test generation specialist, responsible for digging deep to expose hidden bugs and untested paths. Probes are the quality assurance experts, ensuring that code is thoroughly tested and that edge cases are covered.

Probes understand that testing is not just about verifying that code works; it's about finding the ways it might fail. They are experts at identifying untested paths and creating test cases that expose weaknesses.

The Probe's mindset is investigative. They approach code with the question "how could this break?" and design tests to answer that question. They are thorough and systematic, leaving no significant path untested.

#### Role and Responsibilities

**Untested Path Scanning**: Probes scan code for untested paths, identifying gaps in coverage.

**Test Generation**: Probes generate test cases for identified gaps, including unit, integration, and edge case tests.

**Mutation Testing**: Probes run mutation testing to verify that tests actually catch bugs.

**Coverage Analysis**: Probes analyze coverage metrics and identify areas needing improvement.

**Weak Spot Identification**: Probes identify weak spots in the codebase that need additional testing attention.

#### Capabilities and Tools

Probes have access to testing tools:

- **Testing Strategies**: Unit, integration, boundary value, equivalence partitioning, state transition, error guessing, mutation
- **Coverage Tools**: Line, branch, function coverage analysis
- **Execution**: Bash for running tests and coverage tools

#### When to Use This Caste

Spawn a Probe when you need to:
- Generate tests for new code
- Identify gaps in existing test coverage
- Run mutation testing to verify test quality
- Create edge case tests for critical paths
- Improve overall test coverage metrics

#### Example Tasks

- "Generate tests for the new payment processing module"
- "Identify and fill gaps in the auth system test coverage"
- "Run mutation testing on the order service"
- "Create edge case tests for the date parsing function"

#### Spawn Patterns

Probes may spawn additional Probes for different testing domains:

```
Prime Probe (depth 1)
├── Probe A (depth 2) - Unit tests
└── Probe B (depth 2) - Integration tests
```

#### State Management

Probes track:
- **Coverage Metrics**: Lines, branches, functions covered
- **Tests Added**: New test cases created
- **Edge Cases Discovered**: Boundary conditions identified
- **Mutation Score**: Percentage of mutants caught
- **Weak Spots**: Areas needing additional attention

#### Model Assignment

No specific model assigned; inherits default.

---

### 10. Ambassador 🔌🐜

#### Caste Overview

The Ambassador is the colony's integration specialist, responsible for bridging internal systems with external services. Ambassadors are the diplomats of the codebase, negotiating connections between the colony and the outside world.

Ambassadors understand that external integrations are often the most fragile parts of a system. They are experts at designing robust integration patterns that handle failures gracefully and maintain security.

The Ambassador's mindset is connection-focused. They see their role as building bridges that are both functional and resilient, ensuring that communication between systems is reliable and secure.

#### Role and Responsibilities

**API Research**: Ambassadors research external APIs thoroughly before integration.

**Integration Pattern Design**: Ambassadors design integration patterns including Client Wrapper, Circuit Breaker, Retry with Backoff, and Caching.

**Implementation**: Ambassadors implement robust connections to external services.

**Error Scenario Testing**: Ambassadors test error scenarios to ensure graceful handling of failures.

**Security Implementation**: Ambassadors ensure API keys are stored securely, HTTPS is used, and secrets are not logged.

#### Capabilities and Tools

Ambassadors have access to integration tools:

- **Integration Patterns**: Client Wrapper, Circuit Breaker, Retry, Caching, Webhook Handlers
- **Security**: Environment variable management, HTTPS enforcement
- **Error Handling**: Transient error retry, auth token refresh, rate limit handling

#### When to Use This Caste

Spawn an Ambassador when you need to:
- Integrate with a new external API
- Implement OAuth or other authentication flows
- Set up webhook handlers
- Design rate limiting strategies
- Migrate to a new API version

#### Example Tasks

- "Integrate with the Stripe API for payment processing"
- "Set up OAuth2 authentication with Google"
- "Implement a circuit breaker for the external inventory service"
- "Create webhook handlers for GitHub events"

#### Spawn Patterns

Ambassadors may spawn additional Ambassadors for different integrations:

```
Prime Ambassador (depth 1)
├── Ambassador A (depth 2) - Payment API
└── Ambassador B (depth 2) - Email service
```

#### State Management

Ambassadors track:
- **Endpoints Integrated**: APIs connected
- **Authentication Method**: Auth approach used
- **Rate Limits Handled**: Throttling implemented
- **Error Scenarios Covered**: Failure modes tested

#### Model Assignment

No specific model assigned; inherits default.

---

### 11. Tracker 🐛🐜

#### Caste Overview

The Tracker is the colony's debugging specialist, responsible for systematic bug investigation and root cause analysis. Trackers are the detectives of the codebase, following error trails to their source with tenacious precision.

Trackers understand that fixing bugs without understanding their root cause is like treating symptoms without curing the disease. They are experts at gathering evidence, forming hypotheses, and verifying fixes.

The Tracker's mindset is investigative. They approach bugs with scientific rigor, gathering data, forming hypotheses, and testing them systematically. They are patient and thorough, refusing to settle for surface-level fixes.

#### Role and Responsibilities

**Evidence Gathering**: Trackers gather evidence including logs, traces, and context about bugs.

**Reproduction**: Trackers reproduce bugs consistently, ensuring they can be triggered reliably.

**Execution Path Tracing**: Trackers trace execution paths to understand how bugs manifest.

**Root Cause Analysis**: Trackers identify the root cause of bugs, not just the symptoms.

**Fix Verification**: Trackers verify that fixes actually address the root cause.

#### Capabilities and Tools

Trackers have access to debugging tools:

- **Debugging Techniques**: Binary search debugging, log analysis, debugger breakpoints, memory profiling, network tracing
- **Bug Categories**: Logic errors, data issues, timing, environment, integration, state
- **The 3-Fix Rule**: Escalate after three failed fix attempts

#### When to Use This Caste

Spawn a Tracker when you need to:
- Investigate a complex or recurring bug
- Perform root cause analysis on a production issue
- Trace the source of data corruption
- Debug performance problems
- Analyze race conditions

#### Example Tasks

- "Investigate the intermittent 500 errors in production"
- "Trace the source of the data corruption in the orders table"
- "Debug why the cache is not being invalidated correctly"
- "Analyze the race condition in the payment processing"

#### Spawn Patterns

Trackers may spawn additional Trackers for parallel investigation:

```
Prime Tracker (depth 1)
├── Tracker A (depth 2) - Investigate frontend
└── Tracker B (depth 2) - Investigate backend
```

#### State Management

Trackers track:
- **Symptom**: Observable bug behavior
- **Root Cause**: Underlying issue identified
- **Evidence Chain**: Supporting data
- **Fix Applied**: Solution implemented
- **Fix Count**: Number of attempted fixes

#### Model Assignment

No specific model assigned; inherits default.

---

## Knowledge Cluster

### 12. Chronicler 📝🐜

#### Caste Overview

The Chronicler is the colony's documentation specialist, responsible for preserving knowledge in written form. Chroniclers are the historians of the codebase, ensuring that wisdom is recorded for future generations.

Chroniclers understand that documentation is not an afterthought but an essential part of software development. They are experts at creating clear, useful documentation that helps developers understand and use code effectively.

The Chronicler's mindset is preservation-focused. They see their role as creating a record that will outlast the current development cycle, ensuring that knowledge is not lost when developers move on.

#### Role and Responsibilities

**Codebase Survey**: Chroniclers survey codebases to understand their structure and purpose.

**Documentation Gap Identification**: Chroniclers identify areas where documentation is missing or inadequate.

**API Documentation**: Chroniclers document APIs thoroughly, including endpoints, parameters, and responses.

**Guide Creation**: Chroniclers create tutorials, how-tos, and best practice guides.

**Changelog Maintenance**: Chroniclers maintain changelogs and release notes.

#### Capabilities and Tools

Chroniclers have access to documentation tools:

- **Documentation Types**: README, API docs, guides, changelogs, code comments, architecture docs
- **Writing Principles**: Start with "why", clear language, working examples, scanability
- **Tools**: Write, Edit for creating documentation

#### When to Use This Caste

Spawn a Chronicler when you need to:
- Create or update project documentation
- Document APIs for external consumers
- Write tutorials or how-to guides
- Maintain changelogs
- Document architecture decisions

#### Example Tasks

- "Create API documentation for the new endpoints"
- "Update the README with the new setup instructions"
- "Write a guide on how to extend the authentication system"
- "Document the database schema and relationships"

#### Spawn Patterns

Chroniclers may spawn additional Chroniclers for different documentation domains:

```
Prime Chronicler (depth 1)
├── Chronicler A (depth 2) - API docs
└── Chronicler B (depth 2) - Guides
```

#### State Management

Chroniclers track:
- **Documentation Created**: New documents written
- **Documentation Updated**: Existing documents revised
- **Pages Documented**: Page count
- **Code Examples Verified**: Working examples confirmed
- **Gaps Identified**: Missing documentation noted

#### Model Assignment

No specific model assigned; inherits default.

---

### 13. Keeper 📚🐜

#### Caste Overview

The Keeper is the colony's knowledge curator, responsible for organizing patterns and preserving colony wisdom. Keepers are the librarians of the codebase, maintaining the institutional memory that helps the colony learn and improve.

Keepers understand that knowledge is most valuable when it's organized and accessible. They are experts at creating systems for storing and retrieving patterns, constraints, and learnings.

The Keeper's mindset is organizational. They see their role as creating structures that make knowledge discoverable, ensuring that the colony can benefit from past experiences.

#### Role and Responsibilities

**Wisdom Collection**: Keepers collect wisdom from patterns and lessons learned during colony activities.

**Knowledge Organization**: Keepers organize knowledge by domain (patterns/, constraints/, learnings/).

**Pattern Validation**: Keepers validate that documented patterns actually work.

**Archiving**: Keepers archive learnings for future reference.

**Pruning**: Keepers prune outdated information to keep the knowledge base current.

#### Capabilities and Tools

Keepers have access to knowledge management tools:

- **Knowledge Organization**: patterns/, constraints/, learnings/ directories
- **Pattern Template**: Context, Problem, Solution, Example, Consequences, Related
- **Tools**: Write, Edit for creating knowledge base entries

#### When to Use This Caste

Spawn a Keeper when you need to:
- Organize patterns extracted from development work
- Create a knowledge base for a project
- Archive learnings from a completed phase
- Validate and update existing patterns

#### Example Tasks

- "Organize the authentication patterns into the knowledge base"
- "Archive the learnings from the performance optimization phase"
- "Create a pattern library for common UI components"
- "Update outdated patterns with new best practices"

#### Spawn Patterns

Keepers may spawn additional Keepers for different knowledge domains:

```
Prime Keeper (depth 1)
├── Keeper A (depth 2) - Architecture patterns
└── Keeper B (depth 2) - Implementation patterns
```

#### State Management

Keepers track:
- **Patterns Archived**: New patterns added
- **Patterns Updated**: Existing patterns revised
- **Patterns Pruned**: Outdated patterns removed
- **Categories Organized**: Knowledge base structure

#### Model Assignment

No specific model assigned; inherits default.

---

### 14. Auditor 👥🐜

#### Caste Overview

The Auditor is the colony's code review specialist, responsible for examining code with expert eyes for security, performance, and quality. Auditors are the inspectors of the codebase, finding issues that others miss.

Auditors understand that code review is not just about finding bugs; it's about ensuring that code meets standards and follows best practices. They are experts at applying specialized lenses to code examination.

The Auditor's mindset is critical. They approach code with a skeptical eye, looking for issues and risks that might not be apparent to the original author.

#### Role and Responsibilities

**Lens Selection**: Auditors select appropriate audit lenses based on context (Security, Performance, Quality, Maintainability).

**Systematic Scanning**: Auditors scan code systematically, looking for issues within each lens.

**Severity Scoring**: Auditors score findings by severity (CRITICAL, HIGH, MEDIUM, LOW, INFO).

**Documentation**: Auditors document findings with evidence and specific recommendations.

**Fix Verification**: Auditors verify that fixes actually address the identified issues.

#### Capabilities and Tools

Auditors have access to review tools:

- **Security Lens**: Input validation, auth, SQL injection, XSS, secrets
- **Performance Lens**: Complexity, queries, memory, caching
- **Quality Lens**: Readability, coverage, error handling, documentation
- **Maintainability Lens**: Coupling, debt, duplication

#### When to Use This Caste

Spawn an Auditor when you need to:
- Perform a security audit on new code
- Review code for performance issues
- Check code quality before merge
- Assess maintainability of legacy code

#### Example Tasks

- "Audit the new auth module for security issues"
- "Review the database queries for performance problems"
- "Check the codebase for maintainability issues"
- "Perform a pre-merge quality audit"

#### Spawn Patterns

Auditors may spawn additional Auditors for different audit dimensions:

```
Prime Auditor (depth 1)
├── Auditor A (depth 2) - Security audit
└── Auditor B (depth 2) - Performance audit
```

#### State Management

Auditors track:
- **Dimensions Audited**: Lenses applied
- **Findings by Severity**: Issue counts
- **Issues List**: Detailed findings with fixes
- **Overall Score**: Aggregate quality metric

#### Model Assignment

No specific model assigned; inherits default.

---

### 15. Sage 📜🐜

#### Caste Overview

The Sage is the colony's analytics specialist, responsible for extracting trends from history to guide decisions. Sages are the data scientists of the codebase, finding patterns in development metrics.

Sages understand that data-driven decisions are more reliable than intuition alone. They are experts at gathering metrics, analyzing trends, and presenting insights in actionable ways.

The Sage's mindset is analytical. They approach questions with data, looking for quantitative evidence to support recommendations.

#### Role and Responsibilities

**Data Gathering**: Sages gather data from multiple sources including git history, issue trackers, and build systems.

**Data Cleaning**: Sages clean and prepare data for analysis.

**Pattern Analysis**: Sages analyze patterns in development velocity, quality metrics, and team collaboration.

**Insight Interpretation**: Sages interpret analysis results into actionable insights.

**Recommendation**: Sages recommend actions based on data-driven insights.

#### Capabilities and Tools

Sages have access to analytics tools:

- **Development Metrics**: Velocity, cycle time, deployment frequency
- **Quality Metrics**: Bug density, coverage trends, technical debt
- **Team Metrics**: Work distribution, collaboration patterns
- **Visualization**: Trend lines, heat maps, cumulative flow diagrams

#### When to Use This Caste

Spawn a Sage when you need to:
- Analyze development velocity trends
- Identify quality metrics patterns
- Assess team collaboration effectiveness
- Create data visualizations for stakeholders

#### Example Tasks

- "Analyze our development velocity over the last quarter"
- "Identify trends in bug density by component"
- "Assess the effectiveness of our code review process"
- "Create a dashboard showing deployment frequency"

#### Spawn Patterns

Sages may spawn additional Sages for different analysis domains:

```
Prime Sage (depth 1)
├── Sage A (depth 2) - Development metrics
└── Sage B (depth 2) - Quality metrics
```

#### State Management

Sages track:
- **Key Findings**: Significant discoveries
- **Trends**: Patterns over time
- **Metrics Analyzed**: Data points examined
- **Predictions**: Future projections
- **Recommendations**: Action items with priorities

#### Model Assignment

No specific model assigned; inherits default.

---

## Quality Cluster

### 16. Guardian 🛡️🐜

#### Caste Overview

The Guardian is the colony's security specialist, responsible for security audits and vulnerability scanning. Guardians are the defenders of the codebase, patrolling for security threats.

Guardians understand that security is not a feature but a foundation. They are experts at identifying vulnerabilities and ensuring that the codebase is protected against attacks.

The Guardian's mindset is defensive. They approach code with an attacker's perspective, looking for weaknesses that could be exploited.

#### Role and Responsibilities

**Architecture Understanding**: Guardians understand the application architecture to identify security-relevant components.

**OWASP Scanning**: Guardians scan for OWASP Top 10 vulnerabilities.

**Dependency Checking**: Guardians check dependencies for known CVEs.

**Security Domain Review**: Guardians review authentication, input validation, data protection, and infrastructure security.

**Threat Assessment**: Guardians assess threats with severity ratings and remediation recommendations.

#### Capabilities and Tools

Guardians have access to security tools:

- **Security Domains**: Auth/AuthZ, Input Validation, Data Protection, Infrastructure
- **Vulnerability Databases**: CVE checking, OWASP Top 10
- **Severity Ratings**: CRITICAL, HIGH, MEDIUM, LOW, INFO

#### When to Use This Caste

Spawn a Guardian when you need to:
- Perform a security audit on new features
- Scan for OWASP Top 10 vulnerabilities
- Check dependencies for known CVEs
- Review authentication and authorization implementation

#### Example Tasks

- "Perform a security audit on the new payment feature"
- "Scan the codebase for OWASP Top 10 vulnerabilities"
- "Check all dependencies for known CVEs"
- "Review the authentication implementation for security issues"

#### Spawn Patterns

Guardians may spawn additional Guardians for different security domains:

```
Prime Guardian (depth 1)
├── Guardian A (depth 2) - Auth review
└── Guardian B (depth 2) - Input validation review
```

#### State Management

Guardians track:
- **Domains Reviewed**: Security areas examined
- **Findings by Severity**: Vulnerability counts
- **Vulnerabilities List**: Detailed findings with remediation
- **Overall Risk**: Aggregate security assessment

#### Model Assignment

No specific model assigned; inherits default.

---

### 17. Measurer ⚡🐜

#### Caste Overview

The Measurer is the colony's performance specialist, responsible for benchmarking and optimizing system performance. Measurers are the performance engineers of the codebase, ensuring that systems run efficiently.

Measurers understand that performance is a feature that affects user experience. They are experts at identifying bottlenecks and recommending optimizations.

The Measurer's mindset is measurement-focused. They approach performance questions with benchmarks, establishing baselines and measuring improvements.

#### Role and Responsibilities

**Baseline Establishment**: Measurers establish performance baselines for comparison.

**Load Benchmarking**: Measurers benchmark systems under load to identify breaking points.

**Code Path Profiling**: Measurers profile code paths to identify hotspots.

**Bottleneck Identification**: Measurers identify performance bottlenecks and their root causes.

**Optimization Recommendation**: Measurers recommend optimizations with estimated impact.

#### Capabilities and Tools

Measurers have access to performance tools:

- **Performance Dimensions**: Response Time, Throughput, Resource Usage, Scalability
- **Optimization Strategies**: Code level, Database level, Architecture level
- **Profiling Tools**: CPU, memory, network profiling

#### When to Use This Caste

Spawn a Measurer when you need to:
- Benchmark system performance
- Identify performance bottlenecks
- Optimize slow database queries
- Assess scalability limits

#### Example Tasks

- "Benchmark the API response times under load"
- "Identify the bottlenecks in the checkout process"
- "Optimize the slow database queries"
- "Assess the scalability of the current architecture"

#### Spawn Patterns

Measurers may spawn additional Measurers for different performance domains:

```
Prime Measurer (depth 1)
├── Measurer A (depth 2) - API performance
└── Measurer B (depth 2) - Database performance
```

#### State Management

Measurers track:
- **Baseline vs Current**: Performance comparisons
- **Bottlenecks Identified**: Slow components
- **Metrics**: Response time, throughput, CPU, memory
- **Recommendations**: Optimization suggestions with impact estimates

#### Model Assignment

No specific model assigned; inherits default.

---

### 18. Includer ♿🐜

#### Caste Overview

The Includer is the colony's accessibility specialist, responsible for accessibility audits and WCAG compliance. Includers are the advocates for inclusive design, ensuring that all users can access applications.

Includers understand that accessibility is not a niche concern but a fundamental aspect of quality. They are experts at identifying accessibility barriers and ensuring compliance with standards.

The Includer's mindset is inclusive. They approach design with the needs of all users in mind, ensuring that applications work for people with diverse abilities.

#### Role and Responsibilities

**Automated Scanning**: Includers run automated accessibility scans to identify issues.

**Manual Testing**: Includers perform manual testing including keyboard navigation and screen reader testing.

**Code Review**: Includers review code for semantic HTML and ARIA usage.

**WCAG Compliance**: Includers assess compliance with WCAG levels A, AA, and AAA.

**Fix Verification**: Includers verify that accessibility fixes actually resolve issues.

#### Capabilities and Tools

Includers have access to accessibility tools:

- **Accessibility Dimensions**: Visual, Motor, Cognitive, Hearing
- **WCAG Levels**: A (minimum), AA (standard), AAA (enhanced)
- **Testing Methods**: Automated scans, keyboard testing, screen reader testing

#### When to Use This Caste

Spawn an Includer when you need to:
- Audit a new feature for accessibility
- Ensure WCAG AA compliance
- Test keyboard navigation
- Review code for semantic HTML

#### Example Tasks

- "Audit the new checkout flow for accessibility"
- "Ensure the dashboard meets WCAG AA standards"
- "Test the navigation with keyboard-only interaction"
- "Review the form components for ARIA usage"

#### Spawn Patterns

Includers may spawn additional Includers for different accessibility domains:

```
Prime Includer (depth 1)
├── Includer A (depth 2) - Visual accessibility
└── Includer B (depth 2) - Motor accessibility
```

#### State Management

Includers track:
- **WCAG Level**: Target compliance level
- **Compliance Percent**: Overall compliance score
- **Violations**: Issues with WCAG references
- **Testing Performed**: Methods used

#### Model Assignment

No specific model assigned; inherits default.

---

### 19. Gatekeeper 📦🐜

#### Caste Overview

The Gatekeeper is the colony's dependency management specialist, responsible for supply chain security and license compliance. Gatekeepers are the guardians of the codebase perimeter, controlling what enters the system.

Gatekeepers understand that dependencies are a significant source of risk. They are experts at identifying vulnerable packages, license conflicts, and maintenance issues.

The Gatekeeper's mindset is protective. They approach dependencies with skepticism, ensuring that only safe, compliant, and well-maintained packages are used.

#### Role and Responsibilities

**Dependency Inventory**: Gatekeepers inventory all dependencies to understand the supply chain.

**Security Scanning**: Gatekeepers scan for security vulnerabilities in dependencies.

**License Auditing**: Gatekeepers audit licenses for compliance with project requirements.

**Dependency Health Assessment**: Gatekeepers assess the health of dependencies including maintenance status and update availability.

**Severity Reporting**: Gatekeepers report findings with severity ratings and remediation recommendations.

#### Capabilities and Tools

Gatekeepers have access to dependency management tools:

- **Security Scanning**: CVE database checking, malicious package detection
- **License Categories**: Permissive, Weak Copyleft, Strong Copyleft, Proprietary, Unknown
- **Health Metrics**: Outdated packages, maintenance status, community health

#### When to Use This Caste

Spawn a Gatekeeper when you need to:
- Audit dependencies for security vulnerabilities
- Check license compliance
- Assess dependency health
- Review new dependencies before adding them

#### Example Tasks

- "Audit all dependencies for known CVEs"
- "Check license compliance for the project"
- "Assess the health of our top 10 dependencies"
- "Review the new npm package before adding it"

#### Spawn Patterns

Gatekeepers may spawn additional Gatekeepers for different dependency domains:

```
Prime Gatekeeper (depth 1)
├── Gatekeeper A (depth 2) - Security audit
└── Gatekeeper B (depth 2) - License audit
```

#### State Management

Gatekeepers track:
- **Security Findings**: Vulnerabilities by severity
- **Licenses**: License inventory and compatibility
- **Outdated Packages**: Dependencies needing updates
- **Recommendations**: Remediation suggestions

#### Model Assignment

No specific model assigned; inherits default.

---

## Special Castes

### 20. Archaeologist 🏺🐜

#### Caste Overview

The Archaeologist is the colony's git historian, responsible for excavating why code exists through git history. Archaeologists are the historians of the codebase, reading the sediment layers of commits to understand the evolution of the system.

Archaeologists understand that code is not just a snapshot but a story. They are experts at tracing the history of decisions, understanding the context behind workarounds, and identifying stable vs. volatile areas.

The Archaeologist's mindset is investigative and historical. They approach code with curiosity about its past, seeking to understand not just what it does but why it exists in its current form.

**CRITICAL RULE**: Archaeologists are strictly read-only. They NEVER modify code or colony state.

#### Role and Responsibilities

**Git History Analysis**: Archaeologists read git history like ancient inscriptions, tracing the evolution of code.

**Why Investigation**: Archaeologists trace the "why" behind every workaround and oddity.

**Stability Mapping**: Archaeologists map which areas are stable bedrock vs. shifting sand.

**Knowledge Concentration**: Archaeologists identify if critical knowledge is concentrated in one author.

**Incident Archaeology**: Archaeologists identify emergency fixes and their context.

#### Capabilities and Tools

Archaeologists have access to git tools:

- **Git Commands**: `git log`, `git blame`, `git show`, `git log --follow`
- **Analysis**: Tracing file history, identifying significant commits
- **Read-Only**: Strict prohibition on modifications

#### When to Use This Caste

Spawn an Archaeologist when you need to:
- Understand the history of a complex module
- Identify why a workaround exists
- Map code stability for refactoring planning
- Understand knowledge distribution in the team

#### Example Tasks

- "Excavate the history of the authentication module"
- "Understand why this workaround exists in the payment code"
- "Map the stability of different areas for refactoring"
- "Identify knowledge concentration in the codebase"

#### Spawn Patterns

Archaeologists typically work alone due to their read-only nature:

```
Queen (depth 1)
└── Archaeologist (depth 2) - Excavate history
```

#### State Management

Archaeologists produce:
- **Site Overview**: Commit counts, author counts, date range
- **Findings**: Historical insights
- **Tech Debt Markers**: TODO, FIXME, HACK locations
- **Churn Hotspots**: Frequently modified areas
- **Stability Map**: Stable, moderate, volatile areas
- **Tribal Knowledge**: Undocumented knowledge identified

#### Model Assignment

- **Assigned Model**: glm-5
- **Strengths**: Long-context analysis
- **Best For**: Historical analysis, pattern recognition in git history

---

### 21. Oracle 🔮🐜

#### Caste Overview

The Oracle is the colony's deep research specialist, responsible for performing deep research using the RALF (Research-Analyze-Learn-Findings) loop. Oracles are the research scientists of the colony, conducting thorough investigations into complex topics.

Oracles understand that some questions require more than quick answers; they require deep understanding. They are experts at conducting comprehensive research and synthesizing findings into actionable knowledge.

The Oracle's mindset is research-focused. They approach complex questions with systematic investigation, ensuring that no significant aspect goes unexplored.

#### Role and Responsibilities

**Deep Research**: Oracles perform deep research on complex topics using the RALF loop.

**Analysis**: Oracles analyze research findings to extract key insights.

**Learning Synthesis**: Oracles synthesize learnings into actionable knowledge.

**Findings Documentation**: Oracles document findings for colony use.

#### Capabilities and Tools

Oracles have access to research tools:

- **RALF Loop**: Research-Analyze-Learn-Findings methodology
- **Research Tools**: WebSearch, WebFetch, Read, Grep
- **Synthesis**: Pattern extraction, insight generation

#### When to Use This Caste

Spawn an Oracle when you need to:
- Conduct deep research on a complex technology
- Investigate architectural approaches
- Research best practices for a new domain
- Analyze competing solutions

#### Example Tasks

- "Research the best approaches for microservices architecture"
- "Investigate state management solutions for React"
- "Analyze different database options for our use case"
- "Research CI/CD best practices for our stack"

#### Spawn Patterns

Oracles may spawn additional researchers:

```
Prime Oracle (depth 1)
├── Oracle A (depth 2) - Research approach A
└── Oracle B (depth 2) - Research approach B
```

#### State Management

Oracles produce:
- **Research Summary**: Key findings
- **Analysis**: Insights extracted
- **Learnings**: Actionable knowledge
- **Recommendations**: Next steps

#### Model Assignment

- **Assigned Model**: minimax-2.5
- **Strengths**: Research, architecture, task decomposition
- **Best For**: Deep research, complex analysis

---

### 22. Chaos 🎲🐜

#### Caste Overview

The Chaos is the colony's resilience tester, responsible for probing edge cases, boundary conditions, and unexpected inputs. Chaos ants are the testers who ask "but what if?" when everyone else says "it works!"

Chaos ants understand that systems fail in unexpected ways. They are experts at designing scenarios that challenge assumptions and expose weaknesses.

The Chaos mindset is adversarial. They approach code with the goal of breaking it, identifying assumptions and designing tests that violate those assumptions.

**CRITICAL RULE**: Chaos ants are strictly read-only. They NEVER modify code or fix what they find.

#### Role and Responsibilities

**Edge Case Probing**: Chaos ants probe edge cases including empty strings, nulls, unicode, and extreme values.

**Boundary Testing**: Chaos ants test boundary conditions including off-by-one errors, max/min limits, and overflow.

**Error Handling Investigation**: Chaos ants investigate error handling gaps including missing try/catch and swallowed errors.

**State Corruption Testing**: Chaos ants test state corruption scenarios including partial updates and race conditions.

**Unexpected Input Testing**: Chaos ants test unexpected inputs including wrong types and malformed data.

#### Capabilities and Tools

Chaos ants have access to testing tools:

- **Investigation Categories**: Exactly 5 scenarios (Edge Cases, Boundary Conditions, Error Handling, State Corruption, Unexpected Inputs)
- **Severity Guide**: CRITICAL, HIGH, MEDIUM, LOW, INFO
- **Read-Only**: Strict prohibition on modifications

#### When to Use This Caste

Spawn a Chaos ant when you need to:
- Test the resilience of a new feature
- Identify edge cases before they cause issues
- Verify error handling completeness
- Test boundary conditions

#### Example Tasks

- "Probe the new API for edge cases"
- "Test the form validation with unexpected inputs"
- "Investigate error handling in the payment flow"
- "Test boundary conditions in the pagination logic"

#### Spawn Patterns

Chaos ants typically work alone due to their read-only nature:

```
Queen (depth 1)
└── Chaos (depth 2) - Probe for edge cases
```

#### State Management

Chaos ants produce:
- **Scenarios Investigated**: The 5 categories tested
- **Findings**: Issues identified with severity
- **Reproduction Steps**: How to trigger issues
- **Summary**: Total findings by severity
- **Top Recommendation**: Most important action

#### Model Assignment

- **Assigned Model**: kimi-k2.5
- **Strengths**: Edge case identification, creative testing
- **Best For**: Resilience testing, boundary testing

---

## Surveyor Sub-Castes

The Surveyor caste has 4 specialized variants that write to `.aether/data/survey/`:

### Surveyor-Disciplines 📊🐜

**Purpose**: Maps coding conventions and testing patterns
**Outputs**: `DISCIPLINES.md`, `SENTINEL-PROTOCOLS.md`
**When to Use**: Before implementing features to understand conventions

### Surveyor-Nest 📊🐜

**Purpose**: Maps architecture and directory structure
**Outputs**: `BLUEPRINT.md`, `CHAMBERS.md`
**When to Use**: When entering a new codebase or planning structural changes

### Surveyor-Pathogens 📊🐜

**Purpose**: Identifies technical debt, bugs, and concerns
**Outputs**: `PATHOGENS.md`
**When to Use**: Before planning to understand known issues

### Surveyor-Provisions 📊🐜

**Purpose**: Maps technology stack and external integrations
**Outputs**: `PROVISIONS.md`, `TRAILS.md`
**When to Use**: When setting up or modifying dependencies

---

## Spawn System Architecture

### Overview

The Aether spawn system is a hierarchical delegation mechanism that enables parallel work while preventing runaway recursion. The system is designed around three core principles:

1. **Depth-Based Limits**: Workers at different depths have different spawn capabilities
2. **Global Caps**: Hard limits prevent resource exhaustion
3. **Surprise-Based Spawning**: Workers only spawn when encountering genuine complexity

### Spawn Depth Architecture

The spawn system uses a maximum depth of 3, with each depth having distinct characteristics:

#### Depth 0: Queen
- **Role**: Colony orchestrator
- **Can Spawn**: Yes (max 4 direct children)
- **Responsibilities**: Phase management, worker dispatch, state coordination
- **Spawn Decision**: Based on phase requirements and goal analysis

#### Depth 1: Prime Workers
- **Role**: Primary specialists
- **Can Spawn**: Yes (max 4 sub-spawns)
- **Responsibilities**: Major task execution, sub-task delegation
- **Spawn Decision**: Based on task complexity analysis

#### Depth 2: Specialists
- **Role**: Focused workers
- **Can Spawn**: Only if genuinely surprised (max 2 sub-spawns)
- **Responsibilities**: Specific sub-tasks, parallel work
- **Spawn Decision**: Only for 3x complexity or unexpected domains

#### Depth 3: Deep Specialists
- **Role**: Leaf workers
- **Can Spawn**: No
- **Responsibilities**: Complete work inline
- **Spawn Decision**: N/A - must complete all work directly

### Global Limits

| Metric | Limit | Reason |
|--------|-------|--------|
| Max spawn depth | 3 | Prevent runaway recursion |
| Max spawns at depth 1 | 4 | Parallelism cap |
| Max spawns at depth 2 | 2 | Secondary cap |
| Global workers per phase | 10 | Hard ceiling |

### Spawn Tree Tracking Mechanism

All spawns are logged to `.aether/data/spawn-tree.txt` in pipe-delimited format:

```
timestamp|parent_id|child_caste|child_name|task_summary|model|status
```

Example entries:
```
2024-01-15T10:30:00Z|Queen|builder|Hammer-42|implement auth module|default|spawned
2024-01-15T10:35:00Z|Hammer-42|completed|auth module with 5 tests
```

The spawn tree enables:
- **Visualization**: ASCII tree representation of worker hierarchy
- **Debugging**: Tracing spawn relationships for troubleshooting
- **Metrics**: Analysis of spawn patterns and effectiveness
- **Depth Calculation**: Determining spawn depth for new workers

### Spawn Decision Criteria

Workers at depth 2+ should only spawn if they encounter genuine surprise:

**Spawn If**:
- Task is 3x larger than expected
- Discovered a sub-domain requiring different expertise
- Found blocking dependency that needs parallel investigation

**DO NOT Spawn For**:
- Tasks completable in < 10 tool calls
- Tedious but straightforward work
- Slight scope expansion within expertise

### Spawn Protocol

The spawn protocol follows these steps:

1. **Check Spawn Allowance**:
   ```bash
   bash .aether/aether-utils.sh spawn-can-spawn {depth}
   # Returns: {"can_spawn": true/false, "depth": N, "max_spawns": N, "current_total": N}
   ```

2. **Generate Child Name**:
   ```bash
   bash .aether/aether-utils.sh generate-ant-name "{caste}"
   # Returns: "Hammer-42", "Vigil-17", etc.
   ```

3. **Log the Spawn**:
   ```bash
   bash .aether/aether-utils.sh spawn-log "{parent}" "{caste}" "{child}" "{task}"
   ```

4. **Use Task Tool** with structured prompt including:
   - Worker spec reference (read `.aether/workers.md`)
   - Constraints from constraints.json
   - Parent context
   - Specific task
   - Spawn capability notice (depth-based)

5. **Log Completion**:
   ```bash
   bash .aether/aether-utils.sh spawn-complete "{child}" "{status}" "{summary}"
   ```

### Compressed Handoffs

To prevent context rot across spawn depths, the colony uses compressed handoffs:

- Each level returns ONLY a summary, not full context
- Parent synthesizes child results, does not pass through
- This prevents exponential context growth

Example return format:
```json
{
  "ant_name": "Hammer-42",
  "status": "completed",
  "summary": "Implemented auth module with JWT support",
  "files_touched": ["src/auth.ts", "src/middleware.ts"],
  "key_findings": ["Used existing user model"],
  "spawns": [],
  "blockers": []
}
```

---

## Worker Lifecycle

### 1. Priming

When a worker is spawned, it receives:
- **Worker Spec**: Reference to read `.aether/workers.md` for caste discipline
- **Constraints**: From constraints.json (pheromone signals)
- **Parent Context**: Task description, why spawning, parent identity
- **Specific Task**: The sub-task to complete
- **Spawn Capability**: Depth-based spawn permissions

### 2. Execution

Workers execute their task following caste-specific discipline:
- Builders follow TDD
- Watchers follow verification protocols
- Scouts follow research workflows
- etc.

### 3. Logging

Workers log progress using:
```bash
bash .aether/aether-utils.sh activity-log "ACTION" "{name} ({Caste})" "description"
```

### 4. Spawning (if needed)

If a worker encounters genuine surprise and has spawn capability:
- Check spawn allowance
- Generate child name
- Log spawn
- Spawn child with Task tool
- Synthesize results

### 5. Completion

Workers complete by:
- Running verification (if applicable)
- Logging completion
- Returning compressed summary

### 6. Synthesis

Parent synthesizes child results:
- Combines multiple child outputs
- Verifies claims with evidence
- Advances phase if appropriate

---

## Communication Patterns

### Parent-Child Communication

Parents communicate with children through:
- **Task Prompt**: Initial instructions passed via Task tool
- **Context**: Parent context explaining why spawning
- **Constraints**: Pheromone signals from constraints.json

Children communicate with parents through:
- **Return JSON**: Compressed summary of work completed
- **Activity Log**: Detailed progress logging
- **Spawn Tree**: Automatic logging of spawn relationships

### Cross-Caste Collaboration

| Primary | Spawns | For |
|---------|--------|-----|
| Builder | Watcher | Verification after implementation |
| Builder | Scout | Research unfamiliar patterns |
| Watcher | Scout | Investigate unfamiliar code |
| Route-Setter | Colonizer | Understand codebase before planning |
| Prime | Any | Based on task analysis |

### Typical Spawn Chains

**Build Phase:**
```
Queen (depth 0)
└── Prime Builder (depth 1)
    ├── Builder A (depth 2) - file 1
    ├── Builder B (depth 2) - file 2
    └── Watcher (depth 2) - verification
```

**Research Phase:**
```
Queen (depth 0)
└── Prime Scout (depth 1)
    ├── Scout A (depth 2) - docs
    └── Scout B (depth 2) - code
```

**Planning Phase:**
```
Queen (depth 0)
└── Route-Setter (depth 1)
    └── Colonizer (depth 2) - codebase mapping
```

---

## Error Handling in Workers

### Error Types

Workers handle several types of errors:

1. **Task Failures**: The specific task could not be completed
2. **Spawn Failures**: Child workers failed or returned errors
3. **Verification Failures**: Implementation does not meet criteria
4. **Blockers**: External dependencies preventing progress

### Error Reporting

Workers report errors through:
- **Status**: "failed" or "blocked" in return JSON
- **Blockers Array**: List of blocking issues
- **Flag Creation**: Persistent blockers via `flag-add`

Example error return:
```json
{
  "ant_name": "Hammer-42",
  "status": "blocked",
  "summary": "Cannot implement auth - database schema missing",
  "blockers": [
    {
      "type": "dependency",
      "description": "User table does not exist in database",
      "resolution": "Create migration for users table"
    }
  ]
}
```

### Flag System

For persistent blockers, workers create flags:
```bash
bash .aether/aether-utils.sh flag-add "blocker" "Missing user table" "Cannot implement auth" "implementation" 2
```

Flag types:
- **blocker**: Critical, blocks phase advancement
- **issue**: High priority warning
- **note**: Low priority observation

### The 3-Fix Rule

For debugging tasks, Trackers follow the 3-Fix Rule:
- If 3 attempted fixes fail, STOP
- Re-examine assumptions
- Consider architectural issues
- Escalate with findings

---

## Model Routing System

### Configuration

Model assignments are defined in `.aether/model-profiles.yaml`:

```yaml
worker_models:
  prime: glm-5
  archaeologist: glm-5
  architect: glm-5
  oracle: minimax-2.5
  route_setter: kimi-k2.5
  builder: kimi-k2.5
  watcher: kimi-k2.5
  scout: kimi-k2.5
  chaos: kimi-k2.5
  colonizer: kimi-k2.5

task_routing:
  default_model: kimi-k2.5
  complexity_indicators:
    complex:
      keywords: [design, architecture, plan, coordinate, synthesize, strategize, optimize]
      model: glm-5
    simple:
      keywords: [implement, code, refactor, write, create]
      model: kimi-k2.5
    validate:
      keywords: [test, validate, verify, check, review, audit]
      model: minimax-2.5
```

### Available Models

| Model | Provider | Context | Best For |
|-------|----------|---------|----------|
| glm-5 | Z_AI | 200K | Planning, coordination, complex reasoning |
| kimi-k2.5 | Moonshot | 256K | Code generation, visual coding, validation |
| minimax-2.5 | MiniMax | 200K | Research, architecture, task decomposition |

### Status: NON-FUNCTIONAL

**The model-per-caste routing system is aspirational only.**

From `.aether/workers.md`:
> "A model-per-caste routing system was designed and implemented (archived in `.aether/archive/model-routing/`) but cannot function due to Claude Code Task tool limitations. The archive is preserved for future use if the platform adds environment variable support for subagents."

### Why It Doesn't Work

1. **Claude Code Task Tool Limitation**: The Task tool does not support passing environment variables to spawned workers. All workers inherit the parent session's model configuration.

2. **No Environment Variable Inheritance**: ANTHROPIC_MODEL set in parent is not inherited by spawned workers through Task tool.

3. **Session-Level Model Selection**: Model selection happens at the session level, not per-worker. To use a specific model, user must:
   ```bash
   export ANTHROPIC_BASE_URL=http://localhost:4000
   export ANTHROPIC_AUTH_TOKEN=sk-litellm-local
   export ANTHROPIC_MODEL=kimi-k2.5
   claude
   ```

### Workaround

Currently, all workers use the default model of the parent session. To use different models:

1. Start multiple Claude Code sessions with different models
2. Use the appropriate session for the task type
3. Future: If Claude Code adds environment variable support, the archived model routing can be restored

---

## Worker Priming System

### Agent Definition Files

Each caste has a dedicated agent definition file:
- `.aether/agents/aether-{caste}.md` (Claude Code)
- `.opencode/agents/aether-{caste}.md` (OpenCode)

### Agent File Structure

```yaml
---
name: aether-{caste}
description: "{description}"
---

You are **{Emoji} {Caste} Ant** in the Aether Colony. {Role description}

## Aether Integration

This agent operates as a **{specialist/orchestrator}** within the Aether Colony system. You:
- Report to the Queen/Prime worker who spawns you
- Log activity using Aether utilities
- Follow depth-based spawning rules
- Output structured JSON reports

## Activity Logging

Log progress as you work:
```bash
bash .aether/aether-utils.sh activity-log "ACTION" "{your_name} ({Caste})" "description"
```

## Your Role

As {Caste}, you:
1. {Responsibility 1}
2. {Responsibility 2}
...

## Depth-Based Behavior

| Depth | Role | Can Spawn? |
|-------|------|------------|
| 1 | Prime {Caste} | Yes (max 4) |
| 2 | Specialist | Only if surprised |
| 3 | Deep Specialist | No |

## Output Format

```json
{
  "ant_name": "{your name}",
  "caste": "{caste}",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you accomplished",
  ...
}
```
```

### Priming Process

When a worker is spawned via Task tool, it receives:

1. **Worker Spec**: Reference to read `.aether/workers.md` for caste discipline
2. **Constraints**: From constraints.json (pheromone signals)
3. **Parent Context**: Task description, why spawning, parent identity
4. **Specific Task**: The sub-task to complete
5. **Spawn Capability**: Depth-based spawn permissions

### Caste Emoji Mapping

Every spawn must display its caste emoji:
- 🔨🐜 Builder
- 👁️🐜 Watcher
- 🎲🐜 Chaos
- 🔍🐜 Scout
- 🏺🐜 Archaeologist
- 👑🐜 Queen/Prime
- 🗺️🐜 Colonizer
- 🏛️🐜 Architect

---

## Summary Statistics

| Metric | Count |
|--------|-------|
| Total Castes | 22 |
| Core Castes | 7 (Queen, Builder, Watcher, Scout, Colonizer, Architect, Route-Setter) |
| Development Cluster | 4 (Weaver, Probe, Ambassador, Tracker) |
| Knowledge Cluster | 4 (Chronicler, Keeper, Auditor, Sage) |
| Quality Cluster | 4 (Guardian, Measurer, Includer, Gatekeeper) |
| Special Castes | 3 (Archaeologist, Oracle, Chaos) |
| Surveyor Sub-variants | 4 (Disciplines, Nest, Pathogens, Provisions) |
| Agent Definition Files | 47 (.aether: 24, .opencode: 23) |
| Max Spawn Depth | 3 |
| Max Workers Per Phase | 10 |
| Max Spawns at Depth 1 | 4 |
| Max Spawns at Depth 2 | 2 |
| Functional Model Routing | 0 (non-functional) |

---

*Document generated: 2026-02-16*
*Source: Comprehensive analysis of .aether/workers.md, .aether/agents/*.md, .aether/aether-utils.sh, .aether/model-profiles.yaml*
*Word count: ~21,000*
