# Phase 137: Hive Wisdom Injection - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-18
**Phase:** 137-hive-wisdom-injection
**Areas discussed:** Wisdom placement, Read timing, Presentation format

---

## Wisdom Placement

| Option | Description | Selected |
|--------|-------------|----------|
| Separate section | New "HIVE WISDOM (Cross-Colony Patterns)" section after skills | ✓ |
| Folded into colony-prime | Wisdom blended with local context in existing section | |
| Merged with skills | Combined "Skills & Wisdom" block | |

**User's choice:** Separate section
**Notes:** Workers see four distinct knowledge layers: colony context > pheromones > skills > hive wisdom. Clean separation makes each layer's purpose clear.

---

## Read Timing

| Option | Description | Selected |
|--------|-------------|----------|
| Once per build | One hive-read call, cached for all workers | ✓ |
| Once per wave | Per-wave calls, workers in later waves see fresher wisdom | |
| Per worker | Per-worker calls, maximum freshness | |

**User's choice:** Once per build
**Notes:** One subprocess call keeps latency low. Consistent wisdom across all workers in the build. Simpler implementation.

---

## Presentation Format

| Option | Description | Selected |
|--------|-------------|----------|
| Distilled text only | Bulleted list of text entries, no metadata | |
| Structured with metadata | Domain tag + confidence + text per entry | ✓ |
| Summarized per domain | Condensed paragraph per domain | |

**User's choice:** Structured with metadata
**Notes:** Workers see entries like "(go, 0.95) Prefer table-driven tests over testify assertions". They can judge which wisdom applies to their task. Higher token cost but smarter application.

---

## Claude's Discretion

- How hive-injector.ts is structured (standalone module vs. integrated into prompt-assembler)
- How domain tags are sourced at dispatch time
- Token budget for the wisdom section
- How to handle the HIVE-05 integration test
- Relevance discounting for partial domain matches (HIVE-07)
- Where in the dispatch pipeline the hive-read call is made

## Deferred Ideas

None -- discussion stayed within phase scope.
