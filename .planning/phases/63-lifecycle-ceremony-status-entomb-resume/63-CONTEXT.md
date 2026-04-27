# Phase 63: Lifecycle Ceremony -- Status, Entomb, Resume - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 63 enriches three lifecycle commands with ceremony-level behavior. Status shows runtime version truth and signal awareness. Entomb extracts near-miss wisdom before archiving and does a full temp sweep. Resume detects stale FOCUS pheromones and lets the user review them interactively.

**What this phase delivers:**
- Status dashboard shows runtime version line near the top (after goal), with mismatch warning when binary and hub disagree, plus a one-line signal summary with expiry notes
- Entomb extracts near-miss instincts (confidence 0.5-0.8), logs them to chamber manifest, and outputs promotion suggestions; performs full temp sweep (spawn trees, manifests, reviews, midden >30 days, expired pheromones, old session snapshots); updates registry with full final stats
- Resume detects stale FOCUS pheromones whose source_phase is less than current phase, outputs structured stale signal data for wrappers to present interactive keep/clean prompts (Codex gets warning-only runtime-native UX)

**What this phase does NOT deliver:**
- Discuss, chaos, oracle, patrol ceremony changes (Phase 64)
- Idea shelving system (Phase 65)
- Seal or init ceremony changes (Phase 62, already complete)

</domain>

<decisions>
## Implementation Decisions

### Status Version Display
- **D-01:** Runtime version line appears near the top of the status dashboard, after the colony goal line but before progress metrics
- **D-02:** Always shows both binary and hub versions. When they match: "Runtime: v1.0.24 | Hub: v1.0.24". When they mismatch: adds warning indicator like "MISMATCH" so user knows to run update
- **D-03:** One-line signal summary includes expiry awareness: "Signals: 3 active (2 FOCUS expire at seal, 1 REDIRECT persists)" — shows count by type and notes which signals are permanent vs phase-scoped

### Near-Miss Wisdom Extraction (Entomb)
- **D-04:** Entomb logs near-miss instincts (confidence 0.5-0.8) to chamber manifest AND outputs a suggestion line (e.g., "3 instincts eligible for hive promotion -- run aether hive-promote"). Does not auto-promote to hive
- **D-05:** Entomb performs a full temp sweep beyond current cleanup: spawn trees, build manifests, session-scoped review artifacts, midden entries older than 30 days, expired pheromones (strength 0), old session snapshots
- **D-06:** Registry entry gets full final stats when marked inactive: phase count, total plans, total learnings, total instincts, seal date, and colony duration

### Stale Pheromone Detection (Resume)
- **D-07:** Resume detects stale FOCUS pheromones by comparing signal's source_phase against current colony phase — if source_phase < current phase, the signal is stale
- **D-08:** Stale signal UX follows wrapper-runtime contract: Go runtime outputs structured stale signal data, platform wrappers (Claude/OpenCode) handle the interactive keep/clean prompt. Codex gets a warning line only (runtime-native, no interaction)
- **D-09:** Interactive prompt lets user keep or clean each stale FOCUS signal individually before resume proceeds

### Claude's Discretion
- Exact formatting of the version line and signal summary
- Which midden entries count as ">30 days old" (field name and comparison)
- How to structure the stale signal JSON output for wrapper consumption
- Chamber manifest format for near-miss instinct entries
- Exact wording of mismatch warning

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Status ceremony
- `cmd/status.go` -- renderDashboard(), generateProgressBar(), renderPheromoneSummary()
- `cmd/version.go` -- resolveVersion(), readInstalledHubVersion(), --check flag
- `.claude/commands/ant/status.md` -- Claude Code status wrapper

### Entomb ceremony
- `cmd/entomb_cmd.go` -- uniqueChamberName(), writeEntombManifest(), copyEntombArtifacts(), resetColonyStateForEntomb(), clearActiveColonyRuntimeFiles()
- `cmd/hive.go` -- hive-promote subcommand (for near-miss suggestion reference)
- `cmd/queen.go` -- queenPromoteInstinctCmd (for instinct extraction patterns)
- `.claude/commands/ant/entomb.md` -- Claude Code entomb wrapper

### Resume ceremony
- `cmd/session_flow_cmds.go` -- sessionVerifyFresh(), resumeColonyCmd
- `cmd/pheromone_write.go` -- pheromone CRUD, type filtering for FOCUS/REDIRECT
- `.claude/commands/ant/resume.md` -- Claude Code resume wrapper
- `.opencode/commands/ant/resume.md` -- OpenCode resume wrapper

### Cross-references
- `.planning/REQUIREMENTS.md` -- CERE-06, CERE-07, CERE-08 definitions
- `CLAUDE.md` -- UX Architecture section (wrapper-runtime contract)
- `.aether/docs/wrapper-runtime-ux-contract.md` -- full wrapper-runtime contract
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-CONTEXT.md` -- Phase 62 context (seal/init ceremony patterns to follow)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `renderDashboard()` in `cmd/status.go`: Main dashboard renderer. Version line and signal summary can be inserted after goal line display.
- `resolveVersion()` + `readInstalledHubVersion()` in `cmd/version.go`: Already compute both binary and hub versions. Reuse for status version line.
- `writeEntombManifest()` in `cmd/entomb_cmd.go`: Creates manifest.json. Extendable for near-miss instinct logging.
- `clearActiveColonyRuntimeFiles()` in `cmd/entomb_cmd.go`: Current cleanup logic. Extendable for full temp sweep.
- `sessionVerifyFresh()` in `cmd/session_flow_cmds.go`: Existing staleness detection pattern (age + git HEAD). Extend for pheromone staleness.
- `pheromoneWriteCmd` in `cmd/pheromone_write.go`: Has type filtering (FOCUS/REDIRECT). source_phase field exists for comparison.

### Established Patterns
- Wrapper-runtime contract: Go outputs JSON, wrappers handle interaction (from Phase 62 D-06)
- State mutations via `store.SaveJSON()` pattern throughout cmd/
- Ceremony emission via `emitLifecycleCeremony()` for lifecycle events
- Registry updates via `registry` subcommands in `cmd/`
- Version resolution already separates binary vs hub versions

### Integration Points
- Status flow: renderDashboard() → insert version line after goal → insert signal summary
- Entomb flow: entombCmd → extract near-miss instincts → full temp sweep → update registry → archive
- Resume flow: resumeColonyCmd → detect stale FOCUS signals → output structured data → wrapper prompts user
- Pheromone system: source_phase field on signals enables phase comparison

</code_context>

<specifics>
## Specific Ideas

- The signal summary should feel like it "knows" the colony lifecycle — showing which signals survive seal (REDIRECT) vs which get cleaned up (FOCUS)
- Near-miss instincts aren't lost — they're preserved in the chamber and the user gets a nudge to promote if they care
- The mismatch warning should make it obvious that something needs attention, not just a subtle difference

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 63-lifecycle-ceremony-status-entomb-resume*
*Context gathered: 2026-04-27*
