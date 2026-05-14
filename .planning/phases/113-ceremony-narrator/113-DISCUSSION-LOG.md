# Phase 113: Ceremony Narrator - Discussion Log

**Date:** 2026-05-13
**Phase:** 113 — Ceremony Narrator
**Goal:** Render Go events into banners, caste identity, spawn frames, and stage separators

---

## Areas Discussed

### 1. Editable Ceremony Templates

**Options:**
- A: Templates in YAML/markdown under `.aether/templates/ceremony/`
- B: Hardcoded in TypeScript

**User chose:** A

**Rationale:** Non-technical users can tweak ceremony without code changes. Matches v1.17 philosophy.

**Decisions:**
- Templates use YAML frontmatter + markdown body with `{variable}` substitution
- Banners, seal art, and spawn frames are all template-driven
- Default templates ship with the repo, user overrides go in `.aether/templates/ceremony/`

---

### 2. Non-TTY Fallback Behavior

**Options:**
- A: Plain text with structure (no colors, keep emojis and separators)
- B: Minimal output (facts only)
- C: JSON stream

**User chose:** Let Claude decide

**Claude picked:** A — plain text with structure for non-TTY, JSON via `AETHER_OUTPUT_MODE=json`

**Rationale:** Readable in CI logs. Emojis render in most modern terminals. JSON mode already exists in Go runtime.

---

### 3. Direct Terminal vs. Wrapper Strings

**Options:**
- A: Direct to terminal (TS host writes to stdout in real-time)
- B: Hand back to wrapper (accumulated string pasted into wrapper response)

**User chose:** A

**Rationale:** Restores the "alive" feeling of Classic v5.4. Real-time output is what made the old system feel dynamic.

**Decisions:**
- TS host writes ceremony to `process.stdout` as events arrive
- Wrapper still handles high-level orchestration and final summaries
- Ceremony is presentation-only, not part of the structured result

---

### 4. Stage Separator Customization

**Options:**
- A: One universal separator (`── Stage Name ──`)
- B: Per-command custom separators (boxed for build, ornate for seal, etc.)
- C: Fully configurable per-command in YAML

**User chose:** B

**Rationale:** More visual variety while keeping ceremony distinctive per lifecycle stage.

**Decisions:**
- `build`: boxed `┌─ Build ─┐`
- `seal`: ornate `╔═ Crowned Anthill ═╗`
- `plan`: simple `── Plan ──`
- `continue`: simple `── Continue ──`
- `colonize`: simple `── Colonize ──`
- Config lives in `ceremony.yaml` under `stage_separator.<command>` keys

---

## Deferred Ideas

- Template hot-reload with chokidar
- Custom user templates in `~/.aether/templates/ceremony/`
- Animated transitions between ceremony stages (covered by Phase 115 Swarm Dashboard)

---

## Next Steps

`/gsd-plan-phase 113` — create detailed execution plans
