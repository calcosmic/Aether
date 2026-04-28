---
name: technical-feasibility-spike
description: Use when a technical unknown needs a small throwaway experiment before committing to a plan
type: colony
domains: [feasibility, experimentation, validation, prototyping]
agent_roles: [scout, oracle, architect]
workflow_triggers: [discuss, plan]
task_keywords: [spike, feasible, experiment, prototype, validate, unknown]
priority: normal
version: "1.0"
---

# Technical Feasibility Spike

## Purpose

Rapid experimentation that answers one focused question with observable evidence. Spikes are throwaway by design -- the code doesn't ship, the knowledge does. Each spike produces a clear verdict (yes/no/maybe) backed by evidence, so the colony can plan with confidence instead of guesswork.

## When to Use

- A technical choice has multiple options and no clear winner
- Someone says "I'm not sure if X can handle Y" or "has anyone tried..."
- A plan has a dependency marked "needs validation"
- Performance, compatibility, or integration questions block estimation
- You need proof that an approach works before investing in a full build

## Instructions

### 1. Frame the Spike

Every spike answers exactly one question. If you have three questions, run three spikes.

**Spike Card format:**

```markdown
## Spike: {short-name}

**Question:** {one sentence, answerable with yes/no/how-much}
**Why it matters:** {what decision this unblocks}
**Time budget:** {1-4 hours suggested, hard cap}
**Success criteria:** {what evidence would settle the question}
```

Example spike cards:

| Spike | Question | Success Criteria |
|-------|----------|-----------------|
| ws-throughput | Can a single WebSocket server handle 10k concurrent presence streams under 50ms latency? | Load test showing p99 latency at target concurrency |
| sqlite-concurrent | Does SQLite handle 100 concurrent writes without locking timeouts in our schema? | Benchmark script with timing output |
| wasm-compat | Does our image processing WASM module run in Safari 16+ without polyfills? | Screenshot of working module in Safari 16.0 and 17.0 |
| import-existing | Can we parse and import existing Markdown docs without losing frontmatter? | Test script importing 50 sample docs with zero data loss |

### 2. Build the Throwaway Experiment

Create experiment code in a dedicated directory: `spikes/{spike-name}/`

Rules for spike code:
- **Minimal** -- only what's needed to answer the question
- **Isolated** -- no dependencies on production code unless testing integration
- **Observable** -- produces output you can point to as evidence
- **Documented** -- comments explain what's being tested and why
- **Throwaway** -- never import spike code into production

Typical spike structures:

```
spikes/ws-throughput/
  README.md           question, method, verdict
  server.js           minimal WebSocket server
  client.js           load test client
  results.json        raw output
```

```
spikes/sqlite-concurrent/
  README.md           question, method, verdict
  bench.py            benchmark script
  schema.sql          test schema
  output.log          timing results
```

### 3. Run and Record

Execute the experiment. Capture:

- **Setup:** What you did and what environment you used
- **Observation:** What happened -- raw output, measurements, screenshots
- **Verdict:** The answer to the spike question

**Verdict options:**

| Verdict | Meaning | Next Step |
|---------|---------|-----------|
| **YES** | Approach works, proceed with confidence | Route to phase planner |
| **NO** | Approach doesn't work, evidence shows why | Try alternative or revise scope |
| **MAYBE** | Works under conditions, or partial success | Refine constraints or run follow-up spike |
| **SURPRISE** | Discovered something unexpected (positive or negative) | Share finding, may change direction |

### 4. Track in MANIFEST

Maintain a spike inventory so nothing gets lost:

```markdown
# Spike MANIFEST

| # | Name | Question | Status | Verdict | Date |
|---|------|----------|--------|---------|------|
| 1 | ws-throughput | Can WS handle 10k streams? | done | YES | 2026-04-22 |
| 2 | sqlite-concurrent | Concurrent write limits? | done | MAYBE | 2026-04-22 |
| 3 | wasm-safari | Safari compatibility? | active | -- | -- |
```

### 5. Wrap Up Findings

Package everything into a persistent document the colony can reference:

```markdown
# Spike Report: {name}

## Question
{original question}

## Verdict: {YES | NO | MAYBE | SURPRISE}

## Evidence
{key findings with measurements, screenshots, or code output}

## Method
{how the experiment was conducted, so it's reproducible}

## Implications
{what this means for the project -- which decisions it unblocks or changes}

## Artifacts
{list of files in the spike directory}

## Recommendations
{concrete next steps based on findings}
```

## Key Patterns

### The Minimal Question
Strip scope creep ruthlessly. "Can we add real-time collaboration?" is not a spike question. "Can WebSocket handle 10k connections at 50ms latency on our infra?" is.

### The Evidence Standard
Opinions don't count. "I think it would work" is not a verdict. "Load test showed p99 latency of 38ms at 10k connections" is a verdict.

### The Time Box
If the spike isn't answering after the time budget, stop and report what you learned. "We couldn't answer the question in 4 hours because X" is valuable information -- it means the question was bigger than expected.

### The Throwaway Discipline
Never feel invested in spike code. If the answer is NO, delete it and move on. The knowledge is the deliverable, not the code.

### The Surprise Bonus
Sometimes spikes discover things you didn't ask about. Always record surprises -- they're often more valuable than the original answer.

## Output Format

- Writes spike report to `spikes/{name}/README.md`
- Updates `spikes/MANIFEST.md` inventory
- Returns verdict and recommendations to the conversation

## Examples

### Example 1: Performance spike

```
Human: "Can we use SQLite for the main database? Worried about concurrent writes."
Spiker: Framing spike: "Does SQLite handle 100 concurrent writes/second with our schema without locking timeouts?"
Creating experiment at spikes/sqlite-concurrent/ with a benchmark script...
[runs experiment]
Verdict: MAYBE -- handles 80 writes/sec cleanly, starts locking at 100. WAL mode pushes it to 120.
Recommendation: Use SQLite with WAL mode if write rate stays under 100/sec. If we expect spikes above that, add a write queue.
```

### Example 2: Compatibility spike

```
Human: "Will our WASM module work in all target browsers?"
Spiker: Framing spike: "Does the image-processing WASM module execute correctly in Safari 16+, Firefox 115+, Chrome 115+?"
Creating test page at spikes/wasm-compat/ with automated checks...
[runs tests]
Verdict: YES -- works in all three targets without polyfills. Safari 16.0 had a 200ms cold start vs 50ms elsewhere, documented in results.
```

### Example 3: Integration spike

```
Human: "Can we import 50,000 existing Markdown files without losing frontmatter?"
Spiker: Framing spike: "Does our parser extract and preserve all frontmatter fields from 50 sample docs with zero data loss?"
Creating test at spikes/import-existing/ with sample docs...
[runs test]
Verdict: NO -- 3 of 50 docs have multi-line YAML values that get truncated. Identified the parser edge case.
Recommendation: Fix the YAML parser before planning the import pipeline. Filed as a prerequisite bug.
```
