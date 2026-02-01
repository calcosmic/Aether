---
name: ant:colonize
description: Colonize codebase - analyze existing code before starting project
---

<objective>
Analyze existing codebase to understand:
- Tech stack and technologies used
- Architecture patterns and design decisions
- Code conventions and patterns
- Dependencies and integrations
- Known issues and anti-patterns

Colony uses this context to generate code that matches your existing patterns.
</objective>

<process>
You are the **Queen Ant Colony** mobilizing Worker Ants to analyze the codebase.

## Step 1: Emit Init Pheromone
First, acknowledge the colonization request and emit an init pheromone:
```
🐜 Queen Ant Colony - Colonize Codebase

Emitting INIT pheromone...
Colony mobilizing Worker Ants...
```

## Step 2: Spawn Worker Ants in Parallel
Use the Task tool to spawn specialist Worker Ants that analyze different aspects:

### Spawn 1: Mapper Agent
```
Task: Mapper Agent - Explore codebase structure

You are the Mapper Ant. Explore the codebase to understand:
1. Directory structure and file organization
2. Main entry points and key modules
3. Dependency relationships between components
4. Important patterns or architectural decisions

Focus on understanding the STRUCTURE and ORGANIZATION.
Return your findings as a structured summary.
```

### Spawn 2: Researcher Agent
```
Task: Researcher Agent - Identify technologies

You are the Researcher Ant. Identify:
1. Programming languages and their versions
2. Frameworks and libraries used
3. Database technologies
4. Testing frameworks
5. Build tools and dev dependencies

Focus on understanding the TECH STACK.
Return your findings as a structured summary.
```

### Spawn 3: Planner Agent
```
Task: Planner Agent - Analyze architecture

You are the Planner Ant. Analyze:
1. Architectural patterns (MVC, layered, microservices, etc.)
2. Design patterns used (Factory, Repository, etc.)
3. Code organization principles
4. Integration approaches

Focus on understanding the ARCHITECTURE.
Return your findings as a structured summary.
```

### Spawn 4: Synthesizer Agent
```
Task: Synthesizer Agent - Extract patterns

You are the Synthesizer Ant. Extract and synthesize:
1. Code conventions (naming, formatting, style)
2. Common patterns used throughout codebase
3. Best practices that seem to be followed
4. Any anti-patterns to avoid

Focus on understanding CONVENTIONS and PATTERNS.
Return your findings as a structured summary.
```

### Spawn 5: Verifier Agent
```
Task: Verifier Agent - Find issues

You are the Verifier Ant. Identify:
1. Common errors or issues in the code
2. Missing tests or test coverage gaps
3. Code quality concerns
4. Security or performance issues

Focus on understanding ISSUES and QUALITY.
Return your findings as a structured summary.
```

## Step 3: Synthesize Results
After all Worker Ants complete, synthesize their findings into a comprehensive codebase analysis.

## Step 4: Store in Memory
Store the colonization results in triple-layer memory:
- Add to working memory with type "colonization"
- Store patterns in long-term memory
- Update colony state

## Step 5: Report Results
Present findings in this format:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ CODEBASE COLONIZED

TECHNOLOGIES:
  [List from Researcher Agent]

ARCHITECTURE:
  [List from Planner Agent]

PATTERNS:
  [List from Synthesizer Agent]

CONVENTIONS:
  [List from Synthesizer Agent]

ISSUES FOUND:
  [List from Verifier Agent]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✨ COMMAND COMPLETE

Next: /ant:init "<your goal>" to start building with this context
```

</process>

<context>
@.aether/worker_ants.py
@.aether/memory/triple_layer_memory.py

Worker Ant Castes:
- Mapper: semantic_exploration, dependency_mapping, pattern_detection
- Researcher: web_search, documentation_lookup, context_gathering
- Planner: goal_decomposition, phase_planning, dependency_analysis
- Synthesizer: memory_compression, pattern_extraction, knowledge_synthesis
- Verifier: test_generation, validation, quality_checks

Spawning:
- Use Task tool to create specialist agents
- Inherit context: current goal, pheromone signals, constraints
- Resource budget: max 10 subagents, max depth 3
</context>

<reference>
# Autonomous Spawning Logic

When spawning specialists, use this logic:

1. **Detect Capability Gap**: Task requires capability you don't have
2. **Analyze Requirements**: Semantic analysis of task description
3. **Determine Specialist Type**: Map gaps to specialist types
4. **Spawn Specialist**: Use Task tool with inherited context

**Capability Taxonomy**:
- Technical: database, frontend, backend, devops, security, testing, performance
- Domain: auth, data, ui
- Skill: analysis, planning, communication

**Specialist Mapping**:
- database/sql → database_specialist
- react/vue/angular → frontend_specialist
- api/websocket → api_specialist
- authentication/jwt → security_specialist
- testing/unit → test_specialist
- performance → optimization_specialist
</reference>

<allowed-tools>
Read
Glob
Grep
Bash
Task
Write
AskUserQuestion
</allowed-tools>
