# Phase 113: Ceremony Narrator - Context

**Gathered:** 2026-05-13
**Status:** Ready for planning

## Phase Boundary

Render Go ceremony events into living terminal output: ASCII banners, caste identity frames, spawn notifications, stage separators, and seal rituals. The TS host consumes events from the event bridge (built in Phase 112) and writes formatted output directly to `process.stdout`.

## Implementation Decisions

### D-01: Editable Ceremony Templates
- **Decision:** Ceremony banners, seal art, and spawn frame templates live in editable files under `.aether/templates/ceremony/`, NOT hardcoded in TypeScript.
- **Why:** Matches v1.17 philosophy ("Go owns safety, not soul"). Non-technical users can tweak ceremony without code changes.
- **Format:** YAML frontmatter + markdown body. Frontmatter defines variables (figlet font, emoji, title). Body is the template with `{variable}` substitution.
- **Example template:** `.aether/templates/ceremony/banner-build-start.md`

### D-02: Non-TTY Fallback Behavior
- **Decision:** Plain text with structure for non-TTY environments. Colors stripped, emojis preserved, stage separators kept.
- **Why:** Readable in CI logs and pipes. JSON stream mode available via `AETHER_OUTPUT_MODE=json` (already handled by Go runtime).
- **Implementation:** Use `chalk` with `supports-color` detection. When `process.stdout.isTTY === false`, disable ANSI codes but keep structure.

### D-03: Direct Terminal Output
- **Decision:** TS host writes ceremony directly to `process.stdout` in real-time as events stream in. Wrapper handles orchestration and final summaries.
- **Why:** Restores the "alive" feeling of Classic v5.4. Real-time output is what made the old system feel dynamic.
- **Boundary:** Wrapper still receives final structured result via Go finalizers. Ceremony is presentation-only.

### D-04: Per-Command Custom Stage Separators
- **Decision:** Each lifecycle command (build, plan, seal, continue, colonize) can have its own stage separator style configured in `ceremony.yaml`.
- **Styles:**
  - `build`: boxed `┌─ Build ─┐`
  - `seal`: ornate `╔═ Crowned Anthill ═╗`
  - `plan`: simple `── Plan ──`
  - `continue`: simple `── Continue ──`
  - `colonize`: simple `── Colonize ──`
- **Config location:** `ceremony.yaml` under `stage_separator.<command>` keys.

### D-05: Three-Mode Output Support
- **Decision:** TS host supports three output modes: `json` (machine), `visual` (TTY ANSI), `markdown` (plain text).
- **Mode selection:** `AETHER_OUTPUT_MODE` env var, with `visual` as default when TTY detected, `markdown` when not.
- **Why:** Matches existing Go runtime behavior and ROADMAP success criteria.

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Architecture & Contracts
- `.planning/phases/112-foundation/112-CONTEXT.md` — Event bridge, caste config, boundary contract decisions
- `.planning/REQUIREMENTS.md` — CER-01 through CER-07
- `.planning/ROADMAP.md` — Phase 113 goal and success criteria
- `.aether/config/ceremony.yaml` — Shared caste emoji, color, label maps
- `.aether/references/contracts/runtime-boundary-contract.md` — Go/TS ownership boundaries

### Existing Code
- `.aether/ts-host/src/event-bridge.ts` — Event consumption API (startEventBridge, onEvent callback)
- `.aether/ts-host/src/caste-config.ts` — Caste config loader with typed accessors
- `.aether/ts-host/src/types.ts` — CeremonyEvent, CeremonyPayload, CeremonyTopic types
- `cmd/codex_visuals.go` — Current Go rendering logic (to be replaced by event-driven TS rendering)
- `cmd/ceremony_emitter.go` — Go ceremony emitter that publishes to event bus
- `cmd/ceremony_cmd.go` — Existing ceremony CLI commands

## Existing Code Insights

### Reusable Assets
- **Event bridge** — already consumes Go ceremony events. Narrator just needs to subscribe and render.
- **Caste config** — already loads emoji, color, label from YAML. Narrator uses `getCasteEmoji`, `getCasteColor`, `getCasteLabel`.
- **Go ceremony emitter** — already publishes `ceremony.build.spawn`, `ceremony.build.wave.start`, etc. Narrator reacts to these topics.

### Established Patterns
- **Go emits events, TS host renders** — proven pattern from Phase 112. Narrator extends it with visual output.
- **JSON-mediated contract** — event payloads carry all context needed for rendering (caste, name, task, wave, phase).
- **Three-mode output** — Go runtime already handles `AETHER_OUTPUT_MODE`. TS host should respect the same env var.

### Integration Points
- **TS host `lifecycle.ts`** — Will need to wire the narrator into the build/continue lifecycle.
- **Wrapper `build.md`** — Will call `aether ceremony spawn-plan` for display (Go renders complex moments), but lightweight real-time updates come from the TS host via events.
- **Go `ceremony_emitter.go`** — Already publishes events. May need minor changes to ensure all relevant moments emit events (currently some rendering happens directly in `codex_visuals.go` without event emission).

## Specific Ideas

- The template system should support `{variable}` substitution with fallback defaults. For example, `{emoji:🔨}` uses the provided emoji or falls back to 🔨.
- Spawn frames should include: caste emoji, ANSI-colored label, deterministic name, task description, and wave number. Format: `🔨 Builder Mason-67  Task description`
- The "Crowned Anthill" seal art should be a figlet-generated banner using the font specified in `ceremony.yaml`.
- Build summary should render as a framed box with `boxen`, showing completed workers, failed workers, and next phase suggestion.

## Deferred Ideas

- **Template hot-reload** — Watch `.aether/templates/ceremony/` with chokidar and reload templates without restarting the TS host. Nice to have, not critical for v1.17.
- **Custom user templates** — Allow users to override default templates by placing files in `~/.aether/templates/ceremony/`. Belongs in a polish phase.
- **Animated transitions** — Spinner or progress animation between ceremony stages. Out of scope; Phase 115 (Swarm Dashboard) covers animated elements.

---

*Phase: 113-Ceremony Narrator*
*Context gathered: 2026-05-13*
