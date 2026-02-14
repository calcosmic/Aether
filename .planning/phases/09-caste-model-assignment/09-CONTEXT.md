# Phase 9: Caste Model Assignment - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Enable users to view, configure, and verify AI model assignments per worker caste. The system must surface current assignments, allow interactive and CLI-based model changes, verify proxy health, and log actual model usage per spawn.

</domain>

<decisions>
## Implementation Decisions

### CLI Command Design
- **Main command:** `aether caste-models list` — shows current assignments
- **Quick change:** `aether caste-models set <caste>=<model>` — one-line override (e.g., `aether caste-models set builder=glm-5`)
- **Reset override:** `aether caste-models reset <caste>` — removes user override, reverts to default
- **Interactive command:** `/ant:assign-models` (not verify-castes) — interactive experience showing assignments and guiding changes

### Configuration Persistence
- **Location:** Extend `.aether/model-profiles.yaml` with `user_overrides:` section
- **Scope:** Per-repo (each project has its own assignments)
- **Persistence:** Permanent — survives restarts, committed to repo
- **Rationale:** Single file for all model config, clear separation of defaults vs overrides

### Verification Behavior
- **Proxy health check:** Manual only — run via `/ant:assign-models` or `aether caste-models list`
- **On proxy failure:** Warn loudly but continue with default model (kimi-k2.5)
- **Missing model:** If configured model unavailable on proxy, warn loudly but use default
- **Test spawn verification:** Actually spawn a test worker and have it report which model it received
- **Verification depth:** Full end-to-end — check proxy healthy → verify model available → spawn test worker → confirm model responds correctly

### Output and Display
- **Format:** Clean table with columns, plus emojis and checkmarks
- **Columns:** Caste (with emoji), Model, Provider, Capabilities, Context Window, Status (✓/✗)
- **Full details:** Show provider, capabilities, context window, cost tier
- **Status indicators:** Green check for working, warning for issues

### Quick Wins Integration
- **Dreams in status:** Add to `/ant:status` — show recent dream count and last dream timestamp
- **Auto-load context:** Commands should recognize "nestmates" (read TO-DOs and colony state automatically)

### Claude's Discretion
- Exact table formatting and styling
- Specific verification test worker implementation
- How to handle edge cases (corrupt YAML, proxy timeout, etc.)
- Error message wording
- Cache strategy for proxy health checks

</decisions>

<specifics>
## Specific Ideas

- Interactive experience: Claude shows current assignments, helps user select/change through conversation
- CLI for power users who know what they want: `aether caste-models set builder=glm-5`
- Model display example:
  ```
  Caste      Model        Provider  Context  Status
  ───────────────────────────────────────────────
  🔨 Builder  kimi-k2.5    kimi      256K     ✓
  👁️ Watcher  kimi-k2.5    kimi      256K     ✓
  🔮 Oracle   minimax-2.5  minimax   200K     ✓
  🏛️ Prime    glm-5        z_ai      200K     ✓
  ```

</specifics>

<deferred>
## Deferred Ideas

- Global model overrides (across all projects) — could add later if needed
- Task-based routing (keyword detection) — Phase 11 feature
- Scheduled/automatic proxy health monitoring — not needed for MVP
- Model performance telemetry — Phase 11 feature
- Cost tracking per model — out of scope for v3.1

</deferred>

---

*Phase: 09-caste-model-assignment*
*Context gathered: 2026-02-14*
