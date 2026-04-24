---
name: focused-technical-research
description: Use when external library, framework, API, or architecture research must be isolated from implementation context
type: colony
domains: [research, evaluation, comparison, technology-selection]
agent_roles: [oracle, scout, architect]
workflow_triggers: [plan, build]
task_keywords: [research, framework, library, api, sdk, compare, recommendation]
priority: normal
version: "1.0"
---

# Focused Technical Research

## Purpose

Focused research in a context-isolated sandbox. When a phase needs answers about external tools, frameworks, or patterns, this skill investigates without polluting the main planning conversation. It produces a self-contained RESEARCH.md with findings, comparisons, and recommendations that the planner can consume on demand -- like sending a scout ahead while the party camps.

## When to Use

- The phase needs a technology or framework choice and you don't know the options
- Integration with an external service requires understanding its API, SDK, or limitations
- Multiple implementation approaches exist and trade-offs aren't clear
- A spike revealed that more research is needed before deciding
- You need to understand a domain concept before designing a solution

## Instructions

### 1. Frame the Research Question

Define what you need to know, why, and what decision it informs:

```markdown
## Research Brief

**Question:** {what you need to learn}
**Why:** {what decision this research unblocks}
**Scope:** {what's in scope, what's explicitly out of scope}
**Time budget:** {1-3 hours}
**Decision criteria:** {what makes one option better than another for this project}
```

**Example briefs:**

| Question | Decision It Unblocks | Decision Criteria |
|----------|---------------------|-------------------|
| "Which real-time library fits our stack?" | WebSocket/SSE choice for presence feature | Bundle size, server requirements, browser support |
| "How does Stripe handle multi-currency?" | Payment architecture for international launch | Fee structure, API complexity, PCI scope |
| "What are the options for offline data sync?" | Mobile-first architecture decision | Conflict resolution, storage limits, sync reliability |

### 2. Conduct Focused Research

Investigate within the defined scope. For each option or approach:

**A. Gather evidence:**
- Official documentation (prefer docs over blog posts for accuracy)
- GitHub repo health (stars, recent commits, issue response time, release frequency)
- Bundle size or performance benchmarks if relevant
- Known limitations and common complaints (check GitHub issues)
- Community size and ecosystem (plugins, tutorials, Stack Overflow activity)

**B. Test mental model:**
- Can you explain how it works in 3 sentences?
- What's the "hello world" look like? (Complexity of basic usage)
- Where does it get complicated? (Every tool has its rough edge)

**C. Evaluate against criteria:**
- Score each option against the decision criteria
- Note where information is incomplete (honesty > confidence)

### 3. Structure the Comparison

Present findings in a decision-friendly format:

```markdown
## Options Comparison

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| {criterion 1} | {finding} | {finding} | {finding} |
| {criterion 2} | {finding} | {finding} | {finding} |
| {criterion 3} | {finding} | {finding} | {finding} |
| **Overall fit** | {score} | {score} | {score} |
```

### 4. Make a Recommendation

Pick a winner and explain why:

```markdown
## Recommendation: {option}

**Why:** {2-3 sentences connecting the choice to project-specific needs}
**Trade-offs:** {what you gain and what you give up}
**Conditions where this is wrong:** {when would you pick differently}
```

### 5. Produce RESEARCH.md

Assemble the full research document:

```markdown
# Research: {topic}

**Date:** {date}
**Question:** {original question}
**Recommendation:** {short answer}

## Context
{why this research was needed, what decision it informs}

## Options Investigated
{detailed findings for each option}

## Comparison Matrix
{structured comparison table}

## Recommendation
{chosen option with reasoning}

## Implementation Notes
{practical guidance for the builder -- gotchas, setup steps, common patterns}

## References
{links to docs, repos, articles that were consulted}

## Gaps
{what you couldn't find out, what remains uncertain -- be honest}
```

### 6. Return to Caller

Hand off the research cleanly:
- Summarize the recommendation in 2-3 sentences
- Point to the RESEARCH.md file for full details
- Note any gaps or uncertainties that might affect the decision
- The main planning context never needs to see the investigation process -- only the findings

## Key Patterns

### The Depth-First Pattern
When comparing options, go deep on the top 2-3 before going wide. Surface-level comparison of 10 options is less useful than thorough evaluation of the 3 that actually matter.

### The Bad News Pattern
Actively look for reasons NOT to choose each option. The thing that breaks at scale, the API that's deprecated, the dependency that hasn't been updated in 2 years. Bad news is more valuable than good news in research.

### The Hello World Complexity Pattern
If the basic usage example is hard to understand, the advanced usage will be impossible. First impressions of developer experience are remarkably predictive.

### The Ecosystem Check
A great library with no community is a risk. Check: when was the last commit? How many open issues? Are maintainers responsive? Is there a migration path if it gets abandoned?

### The Honest Gap Pattern
Never fill knowledge gaps with assumptions. If the docs don't say whether it handles a case, write "unclear" not "probably fine." Uncertainty documented is better than uncertainty hidden.

## Output Format

- Creates `research/{topic-slug}/RESEARCH.md` with full findings
- Returns recommendation summary to the calling context
- The research directory is isolated -- doesn't pollute planning artifacts

## Examples

### Example 1: Library comparison

```
Human: "Need to pick a state management library for the new frontend. Not sure what's current."
Isolator: Framing research: "Which state management approach best fits a React app with real-time data updates and offline support?"
Researching: Redux Toolkit, Zustand, Jotai, and TanStack Query for server-state...
[conducts research]
Recommendation: TanStack Query for server state + Zustand for client state. Reasoning: real-time data is mostly server state, and TQ handles caching/revalidation beautifully. Zustand covers the small amount of client-only state without boilerplate.
Full comparison in research/state-management/RESEARCH.md
```

### Example 2: API integration research

```
Human: "We need to integrate with Notion's API for the content import feature."
Isolator: Framing research: "What are the capabilities, rate limits, and gotchas of Notion's API for bulk content import?"
Investigating Notion API docs, rate limits, authentication, and known issues...
[conducts research]
Findings: Notion API has a 3 requests/second rate limit, no bulk export endpoint, and rich text blocks need recursive parsing. Content import for 1000+ pages will need a rate-limited queue with retry logic.
Full details in research/notion-api/RESEARCH.md
```

### Example 3: Architecture research

```
Human: "Should we use microservices or a monolith for the backend?"
Isolator: Framing research: "Which backend architecture fits a 2-person team building a SaaS with <10k users in year one?"
Researching trade-offs for team size, deployment complexity, and scaling timeline...
[conducts research]
Recommendation: Modular monolith. Reasoning: At current team size and user count, microservices overhead (deployment, observability, network complexity) outweighs benefits. Design modules with clean boundaries now, split later if needed.
Analysis in research/backend-architecture/RESEARCH.md
```
