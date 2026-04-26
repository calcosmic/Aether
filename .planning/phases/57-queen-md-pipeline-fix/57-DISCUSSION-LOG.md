# Phase 57: QUEEN.md Pipeline Fix - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-26
**Phase:** 57-queen-md-pipeline-fix
**Areas discussed:** Dedup strategy, Global wisdom injection, Seal auto-promotion wiring

---

## Dedup Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Line-exact match | Check if exact line exists in section. Simple, catches exact duplicates only. | |
| Normalized match | Strip dates/timestamps, normalize whitespace before comparing. Catches semantic duplicates. | ✓ |
| Hash-based | SHA-256 hash comparison. Fast but same weakness as line-exact. | |

**User's choice:** Normalized match
**Notes:** The ~270 duplicates were caused by same wisdom promoted with different dates. Normalized matching catches this pattern.

---

## Global Wisdom Injection

| Option | Description | Selected |
|--------|-------------|----------|
| Separate GLOBAL QUEEN section | Add new colony-prime section for global wisdom alongside existing local+hive sections. | (initially selected, then changed) |
| Merge into LOCAL section | Append global wisdom to existing local section. Less prompt bloat but loses provenance. | |

**User's choice:** Changed to "read whole file as block like CLAUDE.md"
**Notes:** User explained global QUEEN.md should work like CLAUDE.md — persistent instructions shaping every worker conversation. Not a parsed section, just the whole file read as a block. This supersedes the initial "separate section" approach.

---

## Seal Auto-Promotion Wiring

| Option | Description | Selected |
|--------|-------------|----------|
| Go runtime native | Seal command in Go loads instincts, filters >= 0.8, calls queen-promote. Deterministic. | |
| Wrapper markdown instructions | Seal wrapper includes instructions to call queen-promote-instinct in a loop. Follows existing ceremony pattern. | ✓ |

**User's choice:** Wrapper markdown instructions
**Notes:** Keeps ceremony in the wrapper layer where other seal steps already live. Also, `queen-promote-instinct` must be changed to write to global `~/.aether/QUEEN.md` (hubStore) instead of local colony store.

---

## Deferred Ideas

- `/ant-lay-eggs` should ask users about their project and pre-fill QUEEN.md with relevant context (new capability, future phase)
