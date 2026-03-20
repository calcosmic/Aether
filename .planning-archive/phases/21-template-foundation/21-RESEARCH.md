# Phase 21: Template Foundation — Research

**Researched:** 2026-02-19
**Domain:** Template extraction and distribution — creating standalone template files from inline command structures
**Confidence:** HIGH

---

## Summary

Phase 21 extracts five critical embedded structures from Aether's command files into standalone template files, then adds those templates to the distribution pipeline. The problem is clearly defined: commands like `init.md` and `seal.md` embed 20-30 line JSON blocks and heredoc markdown structures directly inside 300-500 line instruction files. LLM agents read the whole file and try to reproduce the structure from memory. They fail — adding fields, dropping required ones, or mangling the format.

The fix is the "read and fill" pattern: command files say "read template at path X, fill in these placeholders, write the result." The agent reads the exact structure at the moment it needs it, not reconstructed from memory. This is already proven — the GSD system (which lives in this same repo) uses 22+ templates and operates more reliably than the Aether commands for exactly this reason. Aether has exactly one template today (`QUEEN.md.template`), and the rest are inline.

Phase 21 is CREATE ONLY — no commands are wired to use the templates yet. Phase 24 (Template Integration) does the wiring. This additive approach means zero risk of breaking live commands during Phase 21.

**Primary recommendation:** Create 5-6 critical templates in `.aether/templates/` using the annotated JSON format (`__DOUBLE_UNDERSCORE__` placeholders, `_instructions` field) for JSON templates and `{{DOUBLE_BRACE}}` placeholders for markdown templates, register them in `validate-package.sh`, and verify they distribute correctly through the hub pipeline.

---

<phase_requirements>

## Phase Requirements

Phase 21 covers TMPL-01 through TMPL-06. No standalone REQUIREMENTS.md exists — these IDs are roadmap-only. Based on the analysis documents and codebase inspection, the requirements map as follows:

| ID | Description | Research Support |
|----|-------------|-----------------|
| TMPL-01 | Extract `colony-state.json.template` from init.md (lines 184-213) | Highest priority: wrong fields = every command fails. Inline JSON block confirmed at those exact lines. |
| TMPL-02 | Extract `constraints.json.template` from init.md (lines 219-225) | Small but read on every build. Inline 6-line block confirmed. |
| TMPL-03 | Extract `crowned-anthill.md.template` from seal.md (lines 209-232) | Heredoc confirmed in seal.md at those lines. Missing sections = entomb can't parse. |
| TMPL-04 | Extract `handoff.md.template` from entomb.md (Step 11, lines 411-441) | Heredoc confirmed. Missing sections = resume fails. |
| TMPL-05 | Extract `colony-state-reset.jq.template` from entomb.md (Step 10) | jq pipeline confirmed at lines 358-377. Wrong reset = stale data persists. |
| TMPL-06 | Register templates in distribution pipeline and validate-package.sh | Pipeline already supports templates/ directory — just needs files added. |

Note: The synthesis document recommends ~18 templates and the architecture plan identifies 26. Phase 21 is scoped to the 5 CRITICAL (Priority 1) templates only. Priority 2 (worker prompt templates) and Priority 3 (display/state templates) are out of scope for this phase.

</phase_requirements>

---

## Standard Stack

### Core — No External Libraries Needed

This phase requires zero new dependencies. Everything needed is already present.

| Tool | Already Present | Purpose in Phase 21 |
|------|----------------|---------------------|
| Bash / shell | Yes | Reading/writing template files |
| `jq` | Yes (used throughout) | JSON template validation |
| `validate-package.sh` | Yes (Phase 20 created it) | Add new templates to required files list |
| `bin/cli.js` | Yes | Hub sync already includes `templates/` directory |
| `.aether/.npmignore` | Yes | Already excludes private dirs; templates are NOT excluded |

### What Already Exists and Must Not Be Re-Invented

| Existing Element | How It Applies to Phase 21 |
|------------------|-----------------------------|
| `.aether/templates/QUEEN.md.template` | Proves the template directory works end-to-end. Use `{SINGLE_BRACE}` → but new templates should use `{{DOUBLE_BRACE}}` for markdown (the synthesis recommends deprecating QUEEN's old syntax) |
| `bin/cli.js` `HUB_EXCLUDE_DIRS` | `templates` is NOT in the exclusion list — it syncs to hub automatically |
| `bin/cli.js` line 886 `systemDirs` includes `'templates'` | Templates in `.aether/templates/` automatically migrate to `~/.aether/system/templates/` during hub migration |
| `bin/validate-package.sh` `REQUIRED_FILES` array | Add each new template file here to enforce it must exist |
| `.aether/.npmignore` | Does NOT exclude templates — they are public system files |

---

## Architecture Patterns

### Recommended Directory Structure for New Templates

```
.aether/templates/
  QUEEN.md.template              # Already exists (uses {SINGLE_BRACE} — don't modify)
  json/
    colony-state.template.json   # TMPL-01
    constraints.template.json    # TMPL-02
    colony-state-reset.jq.template  # TMPL-05 (jq script, not JSON)
  md/
    crowned-anthill.template.md  # TMPL-03
    handoff.template.md          # TMPL-04
  REGISTRY.json                  # Version registry (optional but recommended)
```

Note on naming: The synthesis document uses `colony-state.template.json` (type before extension) while the architecture plan uses `colony-state.json.template` (extension last). The architecture plan's convention is clearer for LLM agents reading the file ("this is a JSON template"). Either works — pick one and be consistent. Recommendation: `{name}.template.{ext}` putting type before extension, matching the existing `QUEEN.md.template` convention.

### Pattern 1: Annotated JSON Template (for LLM-written files)

**What:** JSON template files contain both the target structure AND in-file instructions for LLM agents. The agent reads the file, sees the structure AND the fill instructions at the same moment.

**When to use:** Any JSON structure an LLM agent writes (not a shell script).

**The confirmed working format (from schema design doc):**

```json
{
  "_template": "colony-state",
  "_version": "3.0",
  "_instructions": "Write this file to .aether/data/COLONY_STATE.json. Replace all __PLACEHOLDER__ values with real data. Remove all _template, _version, _instructions, and _comment_* keys before writing.",
  "version": "3.0",
  "goal": "__GOAL__",
  "state": "READY",
  "current_phase": 0,
  "session_id": "__SESSION_ID__",
  "initialized_at": "__ISO8601_TIMESTAMP__",
  ...
}
```

**Why `__DOUBLE_UNDERSCORE__` for JSON:** If the agent forgets to substitute a placeholder, `jq` will fail on `"__GOAL__"` as a value in a subsequent read — catching the error. `null` placeholders would produce silently-wrong valid JSON. This is the synthesis document's recommended convention.

**Why NOT `${VARIABLE}` style (architecture plan's suggestion):** Conflicts visually with shell variable syntax, no failure-detection property. The synthesis recommends `__DOUBLE_UNDERSCORE__` and this is the right call.

### Pattern 2: Markdown Template (for heredoc replacement)

**What:** Markdown files with `{{DOUBLE_BRACE}}` placeholders replacing literal text values.

**When to use:** Documents written during colony lifecycle (CROWNED-ANTHILL.md, HANDOFF.md).

**Example (confirmed structure from codebase inspection of seal.md lines 209-232):**

```markdown
# Crowned Anthill — {{GOAL}}

**Sealed:** {{SEAL_DATE}}
**Milestone:** Crowned Anthill
**Version:** {{VERSION}}

## Colony Stats
- Total Phases: {{TOTAL_PHASES}}
- Phases Completed: {{PHASES_COMPLETED}} of {{TOTAL_PHASES}}
- Colony Age: {{COLONY_AGE_DAYS}} days
- Wisdom Promoted: {{PROMOTIONS_MADE}} entries

## Phase Recap
{{PHASE_RECAP}}

## Pheromone Legacy
- Instincts and validated learnings promoted to QUEEN.md
- {{PROMOTIONS_MADE}} total entries promoted

## The Work
{{GOAL}}
```

**Two ways the LLM can use this:**

Option A (LLM reads and fills): Command says "Read template, fill `{{GOAL}}` with the colony goal, write result to `.aether/CROWNED-ANTHILL.md`." The LLM does it directly.

Option B (shell script substitution): `sed -e "s/{{GOAL}}/$goal/g" ... template.md > .aether/CROWNED-ANTHILL.md`

Phase 24 decides which approach. Phase 21 just creates the template. Both options work with the `{{DOUBLE_BRACE}}` convention.

### Pattern 3: jq Script Template (for colony-state-reset)

**What:** A jq filter script stored as a template file, used verbatim by entomb.md.

**When to use:** When the "template" is actually a script rather than a data structure. The colony-state-reset is a jq pipeline (entomb.md lines 358-377), not a JSON file.

**Confirmed current inline structure (from codebase):**

```jq
.goal = null |
.state = "IDLE" |
.current_phase = 0 |
.plan.phases = [] |
.plan.generated_at = null |
.plan.confidence = null |
.build_started_at = null |
.session_id = null |
.initialized_at = null |
.milestone = null |
.events = [] |
.errors.records = [] |
.errors.flagged_patterns = [] |
.signals = [] |
.graveyards = [] |
.memory.instincts = [] |
.memory.phase_learnings = [] |
.memory.decisions = []
```

This is a pure jq filter with no placeholders — it can be stored as-is in the template file and loaded with `jq -f .aether/templates/colony-state-reset.jq.template`. This is simpler than JSON or markdown templates.

### Anti-Patterns to Avoid

- **Don't create a template engine.** Templates are read by LLM agents directly. No Handlebars, no Jinja, no preprocessing step. The "template engine" is the LLM itself.
- **Don't modify the existing `QUEEN.md.template`.** It uses `{SINGLE_BRACE}` syntax — leave it alone. New templates use `{{DOUBLE_BRACE}}` for markdown, `__DOUBLE_UNDERSCORE__` for JSON.
- **Don't put templates in `.aether/data/`.** Data is private and never distributed. Templates must be in `.aether/templates/` which is distributed.
- **Don't wire commands in Phase 21.** The phase is CREATE ONLY. Commands still use their inline structures. Phase 24 does the wiring.
- **Don't add templates to `.aether/.npmignore`.** Templates are system files — they SHOULD be distributed. The npmignore correctly excludes only `data/`, `dreams/`, etc.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Template variable substitution | Custom sed/awk script | LLM reads template directly | The LLM IS the template engine |
| JSON schema validation | New validator | `jq` + `bash .aether/aether-utils.sh validate-state colony` | Already wired in init.md Step 6 |
| Template registry | New tooling | Simple `REGISTRY.json` file | Just a JSON map of name→version |
| Distribution integration | New sync mechanism | `validate-package.sh` REQUIRED_FILES array | Already the pattern for QUEEN.md.template |

**Key insight:** The entire template system for an LLM-driven project is just well-structured files with clear placeholder conventions. No template engine needed. The LLM reads the file and fills the blanks.

---

## Common Pitfalls

### Pitfall 1: Template Content Drift

**What goes wrong:** Template is created from the inline structure at extraction time, but someone later updates the inline structure without updating the template. Both exist, they diverge, and Phase 24's wiring connects to the stale template.

**Why it happens:** Phase 21 creates templates but Phase 21 does NOT remove the inline structures (that's Phase 24's job). During the gap between Phase 21 and Phase 24, both exist.

**How to avoid:** After Phase 21, add a note to each command file: "TEMPLATE EXISTS: see `.aether/templates/...` — if you update this inline structure, update the template too." A CI check would be ideal but is out of scope.

**Warning signs:** Template version field doesn't match the version string scattered through commands (currently "3.0" is hardcoded in 4 commands).

### Pitfall 2: Wrong Exclusion Logic for Templates

**What goes wrong:** Someone adds `templates/` to `HUB_EXCLUDE_DIRS` in `cli.js` or to `.aether/.npmignore`, thinking templates are private. Templates stop distributing.

**Why it happens:** Confusion between the templates directory (public system files) and the data directory (private user data).

**How to avoid:** Templates are system files — they distribute to `~/.aether/system/templates/` and then to target repos. Confirm the `.aether/.npmignore` does NOT contain `templates/` before considering Phase 21 done.

**Current state (verified):** `.aether/.npmignore` does NOT exclude `templates/`. `HUB_EXCLUDE_DIRS` in `cli.js` does NOT include `templates`. Both are correct — no action needed.

### Pitfall 3: jq Script Template Not Working as `jq -f` Input

**What goes wrong:** The colony-state-reset template is a raw jq filter. If the filter has issues (trailing pipe, missing field references), `jq -f` fails silently or with a cryptic error.

**Why it happens:** jq filters extracted from heredoc bash scripts may have trailing characters or variable interpolation issues when moved to a standalone file.

**How to avoid:** After creating the template, run: `jq -f .aether/templates/colony-state-reset.jq.template <<< '{"goal":"test","state":"IDLE","current_phase":0}'` and verify it produces valid JSON output.

### Pitfall 4: Underscore-Prefixed Keys Not Stripped

**What goes wrong:** The `_template`, `_version`, `_instructions`, and `_comment_*` keys from JSON templates are written to the actual data files. `jq` subsequently returns `null` for `._template` when colony commands check state fields, causing cryptic failures.

**Why it happens:** Phase 21 creates the templates with metadata keys. The instructions for stripping them live in the template itself (`_instructions` field). If Phase 24 wires a command incorrectly, the stripping step may be missed.

**How to avoid:** The template's `_instructions` field must explicitly say "Remove all keys beginning with underscore before writing." The template itself is the documentation. Phase 21 makes the instructions clear; Phase 24 must follow them.

### Pitfall 5: `validate-package.sh` Not Updated

**What goes wrong:** Templates are created in `.aether/templates/` but `validate-package.sh`'s `REQUIRED_FILES` array is not updated. Package validation still passes, but if a template is accidentally deleted, no check catches it.

**Why it happens:** Forgetting the two-step: create the file AND register it.

**How to avoid:** Updating `validate-package.sh` is part of TMPL-06. Add each new template to the `REQUIRED_FILES` array in `bin/validate-package.sh`. This is the same pattern used for `templates/QUEEN.md.template` (already in that array).

---

## Code Examples

Verified patterns from codebase inspection:

### Exact Inline Structure to Extract: colony-state.json (init.md lines 184-213)

```json
{
  "version": "3.0",
  "goal": "<the user's goal>",
  "state": "READY",
  "current_phase": 0,
  "session_id": "<generated session_id>",
  "initialized_at": "<ISO-8601 timestamp>",
  "build_started_at": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "phase_learnings": "<inherited learnings or []>",
    "decisions": [],
    "instincts": "<inherited instincts or []>"
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": [
    "<ISO-8601 timestamp>|colony_initialized|init|Colony initialized with goal: <the user's goal>"
  ]
}
```

**What becomes the template (annotated format):**

```json
{
  "_template": "colony-state",
  "_version": "3.0",
  "_instructions": "Write this file to .aether/data/COLONY_STATE.json. Replace all __PLACEHOLDER__ values with real data. Remove all keys starting with underscore (_template, _version, _instructions, and any _comment_*) before writing the file.",
  "_comment_goal": "The user's goal as provided to /ant:init or /ant:lay-eggs",
  "_comment_memory": "Seed from completion-report.md if prior colony found. Otherwise empty arrays.",
  "version": "3.0",
  "goal": "__GOAL__",
  "state": "READY",
  "current_phase": 0,
  "session_id": "__SESSION_ID__",
  "initialized_at": "__ISO8601_TIMESTAMP__",
  "build_started_at": null,
  "plan": {
    "generated_at": null,
    "confidence": null,
    "phases": []
  },
  "memory": {
    "phase_learnings": "__PHASE_LEARNINGS__",
    "decisions": [],
    "instincts": "__INSTINCTS__"
  },
  "errors": {
    "records": [],
    "flagged_patterns": []
  },
  "signals": [],
  "graveyards": [],
  "events": [
    "__ISO8601_TIMESTAMP__|colony_initialized|init|Colony initialized with goal: __GOAL__"
  ]
}
```

Note: `__PHASE_LEARNINGS__` and `__INSTINCTS__` must resolve to JSON arrays (e.g., `[]` or `[{...}]`), not strings. The agent must understand this from the `_comment_memory` annotation.

### Exact Inline Structure to Extract: constraints.json (init.md lines 219-225)

```json
{
  "version": "1.0",
  "focus": [],
  "constraints": []
}
```

This is so small it needs minimal annotation. Template form:

```json
{
  "_template": "constraints",
  "_version": "1.0",
  "_instructions": "Write this file to .aether/data/constraints.json with no placeholder substitutions. Remove all underscore-prefixed keys before writing.",
  "version": "1.0",
  "focus": [],
  "constraints": []
}
```

### Exact jq Filter to Extract: colony-state-reset (entomb.md lines 358-377)

```jq
.goal = null |
.state = "IDLE" |
.current_phase = 0 |
.plan.phases = [] |
.plan.generated_at = null |
.plan.confidence = null |
.build_started_at = null |
.session_id = null |
.initialized_at = null |
.milestone = null |
.events = [] |
.errors.records = [] |
.errors.flagged_patterns = [] |
.signals = [] |
.graveyards = [] |
.memory.instincts = [] |
.memory.phase_learnings = [] |
.memory.decisions = []
```

Store this verbatim in `.aether/templates/colony-state-reset.jq.template`. Usage: `jq -f .aether/templates/colony-state-reset.jq.template .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json`

### How validate-package.sh Registers Templates (current pattern)

```bash
# Current REQUIRED_FILES in bin/validate-package.sh:
REQUIRED_FILES=(
  "aether-utils.sh"
  "workers.md"
  "CONTEXT.md"
  "model-profiles.yaml"
  "docs/README.md"
  "utils/atomic-write.sh"
  "utils/error-handler.sh"
  "utils/file-lock.sh"
  "templates/QUEEN.md.template"   # ← This is the existing pattern
  "rules/aether-colony.md"
)

# After Phase 21, add:
  "templates/json/colony-state.template.json"
  "templates/json/constraints.template.json"
  "templates/colony-state-reset.jq.template"
  "templates/md/crowned-anthill.template.md"
  "templates/md/handoff.template.md"
```

---

## State of the Art

| Old Approach | Current Approach | Status in Aether |
|--------------|------------------|-----------------|
| Inline JSON blocks in command files | Standalone template files with annotated format | Aether still uses old approach for 5 critical structures |
| Heredoc markdown in bash scripts | Template file read by LLM or `sed` | seal.md and entomb.md still use heredoc |
| LLM reconstructs structure from memory | LLM reads template at point of use | Only QUEEN.md.template uses new approach |
| `{SINGLE_BRACE}` placeholder | `{{DOUBLE_BRACE}}` for markdown, `__DOUBLE_UNDERSCORE__` for JSON | QUEEN.md.template uses old style; new templates should use new style |

**The GSD system in this same repo is the current state of the art.** It uses 22+ template files and operates more reliably than Aether commands. Phase 21 applies this proven pattern to Aether.

---

## Open Questions

1. **Subdirectory structure: `templates/json/` and `templates/md/` vs flat `templates/`**
   - What we know: The architecture plan recommends `json/`, `md/`, `prompts/`, `results/`, `display/` subdirectories
   - What's unclear: Whether this complexity is needed now vs when Phases 22-24 add many more templates
   - Recommendation: Start flat for Phase 21's 5 templates (they fit without confusion), add subdirectories when Phase 24 adds prompt and result templates. The QUEEN.md.template already proves flat works.

2. **REGISTRY.json: include in Phase 21 or defer?**
   - What we know: The synthesis document recommends a REGISTRY.json version registry. It would track each template's version for `aether update` drift detection.
   - What's unclear: Whether this is actually needed now vs Phase 24 when commands depend on specific template versions
   - Recommendation: Create a simple REGISTRY.json in Phase 21 as a lightweight manifest. It costs nothing and proves the registry concept before Phase 24 depends on it.

3. **What exactly are "5 critical templates" for TMPL-01 through TMPL-05?**
   - Based on codebase inspection: colony-state.json, constraints.json, crowned-anthill.md, handoff.md, colony-state-reset.jq = 5 templates matching the 5 IDs
   - TMPL-06 is distribution registration (validate-package.sh update)
   - This interpretation is HIGH confidence from the architecture plan and the roadmap description

4. **Does `handoff.md` have ONE template or THREE variants?**
   - What we know: entomb.md has one HANDOFF.md heredoc (Step 11). The architecture plan lists separate handoff-entomb, handoff-pause, handoff-build, handoff-continue variants.
   - What's unclear: Whether Phase 21 creates one unified template or three variants
   - Recommendation: Start with one `handoff.md.template` covering the entomb case (confirmed inline in codebase). Other variants are Phase 3 (deferred per the architecture plan's own phasing).

---

## Planner Notes: Task Breakdown Guidance

Phase 21 tasks should follow this order (dependencies matter):

1. **Inspect source** — Read actual inline structures from init.md, seal.md, entomb.md to confirm exact content before extracting. The line numbers in the architecture plan match the codebase (verified during research).

2. **Create template files** (TMPL-01 through TMPL-05) — These are independent and can be done in any order, but colony-state first since it's the most critical.

3. **Validate jq template** — After creating colony-state-reset.jq.template, test it runs: `jq -f .aether/templates/colony-state-reset.jq.template <<< '{"version":"3.0",...}'`

4. **Register in validate-package.sh** (TMPL-06) — Add all new template paths to REQUIRED_FILES array.

5. **Run `npm install -g .`** — Validates package and syncs to hub. Confirms templates distribute correctly.

6. **Verify hub received templates** — Check `ls ~/.aether/system/templates/` to confirm distribution worked.

No command files are modified in Phase 21. Zero risk of breaking existing functionality.

---

## Sources

### Primary (HIGH confidence)

- Codebase inspection: `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md` — confirmed inline JSON block at lines 184-213, constraints block at lines 219-225
- Codebase inspection: `/Users/callumcowie/repos/Aether/.claude/commands/ant/seal.md` — confirmed heredoc at lines 209-232
- Codebase inspection: `/Users/callumcowie/repos/Aether/.claude/commands/ant/entomb.md` — confirmed jq filter at lines 358-377, HANDOFF.md heredoc at lines 411-441
- Codebase inspection: `/Users/callumcowie/repos/Aether/bin/validate-package.sh` — confirmed REQUIRED_FILES pattern, `templates/QUEEN.md.template` already registered
- Codebase inspection: `/Users/callumcowie/repos/Aether/bin/cli.js` — confirmed `HUB_EXCLUDE_DIRS` does NOT include `templates`, `systemDirs` line 886 includes `'templates'`
- Codebase inspection: `/Users/callumcowie/repos/Aether/.aether/.npmignore` — confirmed `templates/` is NOT excluded
- Codebase inspection: `/Users/callumcowie/repos/Aether/.aether/templates/QUEEN.md.template` — confirmed existing template proves distribution pipeline works end-to-end

### Secondary (MEDIUM confidence — from design documents in this repo)

- `docs/plans/2026-02-18-template-architecture-plan.md` — Full template inventory, format standard, before/after examples, priority system. Created 2026-02-18 by dedicated planning analysis.
- `docs/plans/2026-02-18-template-schema-system-design.md` — Annotated JSON format, loading patterns, variable substitution conventions, versioning.
- `docs/plans/2026-02-18-template-improvement-synthesis.md` — Synthesis of two parallel analyses; resolved conflicts between `${VAR}` and `__VAR__` conventions (recommends `__DOUBLE_UNDERSCORE__`).

### Tertiary (LOW confidence — not validated against official external sources)

- Agent architecture plan recommendations for template format — based on internal analysis, not external benchmarks. The claim that XML-structured prompts show "15-25% higher instruction following" comes from an internal analysis document, not a verified published study.

---

## Metadata

**Confidence breakdown:**
- Template content (what to extract): HIGH — directly verified from codebase inspection
- Distribution pipeline (how templates flow): HIGH — verified from validate-package.sh and cli.js
- Placeholder conventions (`__VAR__` vs `${VAR}`): HIGH — synthesis document explicitly resolves this conflict
- Phase scope (5 templates, no command wiring): HIGH — clearly stated in roadmap and synthesis

**Research date:** 2026-02-19
**Valid until:** 2026-03-19 (stable codebase, 30-day validity)
