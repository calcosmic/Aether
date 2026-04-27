# Phase 60: Oracle Loop Fix - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 60-oracle-loop-fix
**Areas discussed:** Research brief scope, Depth selection UX, OpenCode callback fix, RALF loop formulation

---

## Research Brief Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Full colony context | Project profile, pheromones, learnings, codebase structure, colony goal | ✓ |
| Lean context | Project profile and pheromones only | |
| Project + goal only | Skip codebase structure scanning | |

**User's choice:** Full colony context
**Notes:** Brief should also drive question generation, replacing the current `buildOracleQuestions()` keyword-matching approach.

---

## Brief → Questions

| Option | Description | Selected |
|--------|-------------|----------|
| Brief drives questions | Brief content shapes what the Oracle asks about (replaces generic templates) | ✓ |
| Brief as context only | Keep current keyword-based templates, brief is supplementary worker context | |

**User's choice:** Brief drives questions

---

## Depth Selection UX

| Option | Description | Selected |
|--------|-------------|----------|
| Prompt + flags | Interactive prompt after brief + CLI flags for automation | ✓ |
| Flags only | CLI flags only, no interactive prompt | |
| Prompt only | Interactive prompt only, no CLI flags | |

**User's choice:** Prompt + flags (recommended)

---

## Depth Targets

| Option | Description | Selected |
|--------|-------------|----------|
| 2/5/8/12 + 50/80/90/99 | Aggressive spread with 4 tiers | |
| 1/4/7/10 + 40/75/90/99 | Wider spread starting from 1 iteration | |
| 2/4/6/10 + 60/85/95/99 | Tight increments, 4 tiers | ✓ |

**User's choice:** Quick=2/60%, Balanced=4/85%, Deep=6/95%, Exhaustive=10/99%
**Notes:** User specified 4 options with the last (exhaustive) at 99% confidence.

---

## OpenCode Callback Fix

| Option | Description | Selected |
|--------|-------------|----------|
| Env var override | Separate env var for agent messaging endpoint vs LiteLLM proxy | ✓ |
| Documentation only | Document that users must set OPENCODE_AGENT_URL separately | |
| Defer to later | Skip ORCL-01, may need upstream fix | |

**User's choice:** Env var override
**Notes:** Aether doesn't manage callback URLs directly — it shells out to opencode. The fix adds a config path the OpenCode dispatcher can pass through.

---

## RALF Loop Formulation

| Option | Description | Selected |
|--------|-------------|----------|
| Smart formulation | Controller uses gaps, contradictions, findings to pick most impactful next question | ✓ |
| State quality only | Keep current selection, improve state tracking completeness | |

**User's choice:** Smart formulation
**Notes:** Current `selectOracleQuestion()` just picks lowest-confidence open question. The fix adds a scoring function that considers accumulated knowledge.

---

## Claude's Discretion

- Exact env var name for callback override
- How to render the depth selection prompt
- How brief-informed question generation works in detail
- Exact formulation scoring algorithm

## Deferred Ideas

None — discussion stayed within phase scope.
