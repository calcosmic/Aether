# Phase 24: Template Integration — Research

**Researched:** 2026-02-20
**Domain:** Slash command wiring — replacing inline JSON/heredoc structures with reads from template files created in Phase 21
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Ceremony text:**
- Refresh ALL template content while wiring — not just swap source, improve the content
- Seal ceremony: warm and narrative, **triumphant** mood — colony has reached its peak, crowned anthill achieved
- Entomb ceremony: warm and narrative, **reflective** mood — colony's story is complete, chapter closing
- Two distinct emotional moments, not the same voice
- Internal templates (colony-state, constraints, worker-result) also refreshed for cohesion

**Inline cleanup:**
- Old inline JSON/heredocs **completely removed** from command files — single source of truth in templates
- If template file missing: **clear error message** and stop — "Template missing. Run aether update to fix." — don't try to continue
- No fallback to inline, no commented-out backups
- Both Claude Code AND OpenCode commands wired simultaneously — keep in sync, avoid drift

### Claude's Discretion
- Whether to tighten surrounding command code while removing inline content — judge per command, clean what's messy, leave what's fine
- Template loading mechanism (helper function, direct read, etc.)
- Template placeholder filling approach
- Template lookup chain (hub-first was established in Phase 20 for queen-init — extend or adapt as needed)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope

</user_constraints>

---

<phase_requirements>
## Phase Requirements

No REQUIREMENTS.md exists for Phase 24. IDs are roadmap-only. Based on the roadmap, CONTEXT.md, and
the Phase 21 templates that exist, the 5 WIRE requirements map as follows:

| ID | Description | Research Support |
|----|-------------|-----------------|
| WIRE-01 | Wire `init.md` to read `colony-state.template.json` and `constraints.template.json` instead of inline JSON blocks | Template files verified at `.aether/templates/colony-state.template.json` and `.aether/templates/constraints.template.json`. Both Claude Code and OpenCode versions of init.md need updating. |
| WIRE-02 | Wire `seal.md` to read `crowned-anthill.template.md` instead of inline heredoc (cat > .aether/CROWNED-ANTHILL.md) | Template file verified at `.aether/templates/crowned-anthill.template.md`. Both versions need updating. Sealed ceremony content must be refreshed to triumphant mood. |
| WIRE-03 | Wire `entomb.md` to use `colony-state-reset.jq.template` instead of inline jq filter | Template file verified at `.aether/templates/colony-state-reset.jq.template`. Both versions. Entomb ceremony (HANDOFF.md) content must be refreshed to reflective mood. |
| WIRE-04 | Wire `entomb.md` to read `handoff.template.md` instead of inline heredoc (cat > .aether/HANDOFF.md at entomb Step 11) | Template file verified at `.aether/templates/handoff.template.md`. Build.md also has HANDOFF.md heredocs — Phase 24 context says "init, seal, entomb, build" commands. |
| WIRE-05 | Wire `build.md` HANDOFF heredocs to a template — or at minimum ensure all inline structures are removed and replaced with template reads | Build.md has TWO inline HANDOFF heredocs: the error handoff (Step 5.9) and the success handoff (Step 6.5). These need template wiring or confirmed out of scope. |

**Note on WIRE-05:** The CONTEXT.md phase boundary says "(init, seal, entomb, build)." Build.md has two
HANDOFF.md heredoc blocks. The handoff.template.md created in Phase 21 covers the entomb scenario.
Build's handoffs have different content (error recovery vs. success summary). Research recommends
clarifying this during planning — either scope to one template each for error/success, or treat build's
HANDOFFs as a separate template to create in Phase 24. See Open Questions.

</phase_requirements>

---

## Summary

Phase 24 is the wiring step that makes Phase 21's templates actually do something. Phase 21 created
5 template files in `.aether/templates/` and proved they distribute through the hub. Phase 24 makes
each template the single source of truth by replacing the corresponding inline structures in command
files with template reads.

The scope is surgical: 4 command files (init, seal, entomb, build), each with both a Claude Code
version (`.claude/commands/ant/`) and an OpenCode version (`.opencode/commands/ant/`). Both must be
updated simultaneously per the locked decision. The key implementation question — resolved by this
research — is how the template read should work: the `queen-init` pattern in `aether-utils.sh` shows
the hub-first lookup chain that is already production-proven. The same pattern applies here.

The locked decision that all ceremony content must be refreshed (not just mechanically swapped) means
this phase requires thoughtful rewriting of the `crowned-anthill.template.md` and `handoff.template.md`
content, not just a find-and-replace operation. The seal template gets triumphant voice, the entomb
template gets reflective voice — two distinct emotional registers from one ceremony system.

**Primary recommendation:** Use a bash helper function (or a new `aether-utils.sh` command) that
implements the hub-first lookup chain for template loading, so all four commands share the same
template-not-found error message and lookup behavior. The LLM then reads the template directly and
fills placeholders.

---

## Standard Stack

### Core — No External Libraries Needed

Phase 24 requires zero new dependencies. Everything needed already exists.

| Tool | Already Present | Purpose in Phase 24 |
|------|----------------|---------------------|
| Bash / shell | Yes | Template lookup chain, error handling |
| `jq` | Yes | For jq template (`colony-state-reset.jq.template`) via `jq -f` |
| `sed` | Yes | Optional for simple `{{PLACEHOLDER}}` substitution in markdown templates |
| `.aether/aether-utils.sh` | Yes | Hub-first lookup chain pattern (from `queen-init`) |
| Template files | Phase 21 delivered | All 5 templates in `.aether/templates/` |

### What Already Exists and Must Not Be Re-Invented

| Existing Element | How It Applies to Phase 24 |
|------------------|-----------------------------|
| `aether-utils.sh queen-init` lookup chain | The hub-first pattern (lines 3380-3389) is the proven template resolution strategy. Use this pattern for all template lookups in Phase 24. |
| `.aether/templates/QUEEN.md.template` | Proves the full distribution chain: dev edit → npm publish → hub → target repo. Templates work end-to-end. |
| `sed -e "s/{TIMESTAMP}/$timestamp/g"` in queen-init | Placeholder substitution example using sed. Phase 24 uses `{{DOUBLE_BRACE}}` syntax but the sed approach is identical. |
| `jq -f` command | Used by entomb for colony-state-reset: `jq -f .aether/templates/colony-state-reset.jq.template STATE.json.bak > STATE.json` |

---

## Architecture Patterns

### Template Lookup Chain (Hub-First — Established Pattern)

**What:** Before using a template, resolve its path by checking multiple locations in priority order.
The hub (`~/.aether/system/templates/`) takes priority because it has the latest distributed version.
Falls back to the local `.aether/templates/` path for development contexts (like this Aether repo itself).

**The proven pattern from `aether-utils.sh` lines 3378-3389:**

```bash
# Pattern: hub-first template lookup
template_file=""
for path in \
  "$HOME/.aether/system/templates/QUEEN.md.template" \
  "$AETHER_ROOT/.aether/templates/QUEEN.md.template" \
  "$HOME/.aether/templates/QUEEN.md.template"; do
  if [[ -f "$path" ]]; then
    template_file="$path"
    break
  fi
done

if [[ -z "$template_file" ]]; then
  echo "Template missing. Run aether update to fix."
  exit 1
fi
```

**For Phase 24**, extend this pattern to each template file. The lookup chain for a template named
`colony-state.template.json` would be:

```bash
template_file=""
for path in \
  "$HOME/.aether/system/templates/colony-state.template.json" \
  ".aether/templates/colony-state.template.json"; do
  if [[ -f "$path" ]]; then
    template_file="$path"
    break
  fi
done

if [[ -z "$template_file" ]]; then
  echo "Template missing: colony-state.template.json"
  echo "Run aether update to fix."
  exit 1
fi
```

**Why `AETHER_ROOT` vs relative `.aether/`:** In command files (not shell scripts), use relative paths
like `.aether/templates/`. The `AETHER_ROOT` variable is for use inside `aether-utils.sh` itself.
In command instructions, relative paths work because the working directory is the project root.

### Pattern 1: LLM Reads Template and Fills Placeholders (for markdown templates)

**What:** The command instruction tells the LLM to read the template file, substitute the
`{{PLACEHOLDER}}` values with real data, remove the HTML comment header, and write the result.

**When to use:** `crowned-anthill.template.md` (seal) and `handoff.template.md` (entomb Step 11).

**Current inline approach in seal.md (lines 240-263):**
```bash
cat > .aether/CROWNED-ANTHILL.md << SEAL_EOF
# Crowned Anthill — ${goal}
...
SEAL_EOF
```

**New approach (two options):**

Option A — LLM fills and writes directly:
```
Read .aether/templates/crowned-anthill.template.md (or hub path if available).
Fill all {{PLACEHOLDER}} values:
  - {{GOAL}} → {goal}
  - {{SEAL_DATE}} → {seal_date}
  - {{VERSION}} → {version}
  - {{TOTAL_PHASES}} → {total_phases}
  - {{PHASES_COMPLETED}} → {phases_completed}
  - {{COLONY_AGE_DAYS}} → {colony_age_days}
  - {{PROMOTIONS_MADE}} → {promotions_made}
  - {{PHASE_RECAP}} → {phase_recap}
Remove the HTML comment header block before writing.
Write the filled content to .aether/CROWNED-ANTHILL.md using the Write tool.
```

Option B — Shell sed substitution:
```bash
sed \
  -e "s/{{GOAL}}/$goal/g" \
  -e "s/{{SEAL_DATE}}/$seal_date/g" \
  -e "s/{{VERSION}}/$version/g" \
  -e "s/{{TOTAL_PHASES}}/$total_phases/g" \
  -e "s/{{PHASES_COMPLETED}}/$phases_completed/g" \
  -e "s/{{COLONY_AGE_DAYS}}/$colony_age_days/g" \
  -e "s/{{PROMOTIONS_MADE}}/$promotions_made/g" \
  "$template_file" | grep -v "^<!-- Template\|^<!-- Instructions" > .aether/CROWNED-ANTHILL.md
```

**Recommendation: Option A for markdown templates.** The LLM can handle multi-line values
(like `{{PHASE_RECAP}}`) that sed cannot substitute safely. Shell sed works only for simple
single-word replacements; the phase recap is multi-line content. The LLM approach is also
more readable in command files — it mirrors how the command conceptually works.

**Recommendation: Option B for single-line simple substitutions** — or avoid it altogether
and use Option A consistently. Consistency beats micro-optimization.

### Pattern 2: LLM Reads JSON Template and Writes Filled Structure (for JSON templates)

**What:** The command instruction tells the LLM to read the annotated JSON template, follow the
`_instructions` field, substitute `__PLACEHOLDER__` values, remove underscore-prefixed keys,
and write the result.

**When to use:** `colony-state.template.json` (init Step 3) and `constraints.template.json` (init Step 4).

**Current inline approach in init.md (lines 215-244):**
```json
{
  "version": "3.0",
  "goal": "<the user's goal>",
  ...
}
```

**New approach:**
```
Read .aether/templates/colony-state.template.json.
Follow the _instructions field.
Replace all __PLACEHOLDER__ values:
  - __GOAL__ → {the user's goal}
  - __SESSION_ID__ → {generated session ID}
  - __ISO8601_TIMESTAMP__ → {current timestamp}
  - __PHASE_LEARNINGS__ → {inherited learnings or []}
  - __INSTINCTS__ → {inherited instincts or []}
Remove all keys starting with underscore (_template, _version, _instructions, all _comment_*).
Write the result to .aether/data/COLONY_STATE.json using the Write tool.
```

**Key consideration:** `__PHASE_LEARNINGS__` and `__INSTINCTS__` must resolve to valid JSON arrays
(e.g., `[]` or `[{...}]`), not strings. The command must handle the case where Phase 21's template
has string placeholders and the LLM needs to write arrays. The instruction text in the command must
be explicit: "Replace `__PHASE_LEARNINGS__` with a JSON array (e.g., `[]` if no learnings)."

### Pattern 3: Shell Reads jq Template via `jq -f` (for jq filter template)

**What:** The command uses `jq -f <template_path>` to run the stored jq filter against the colony
state backup file.

**When to use:** `colony-state-reset.jq.template` (entomb Step 10).

**Current inline approach in entomb.md (lines 396-415):**
```bash
jq '
  .goal = null |
  .state = "IDLE" |
  ...
' .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json
```

**New approach:**
```bash
# Resolve template path (hub-first)
jq_template=""
for path in \
  "$HOME/.aether/system/templates/colony-state-reset.jq.template" \
  ".aether/templates/colony-state-reset.jq.template"; do
  if [[ -f "$path" ]]; then
    jq_template="$path"
    break
  fi
done

if [[ -z "$jq_template" ]]; then
  echo "Template missing: colony-state-reset.jq.template"
  echo "Run aether update to fix."
  exit 1
fi

jq -f "$jq_template" .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json
```

This is the cleanest swap — no placeholder filling needed, just a path resolution before `jq -f`.

### Anti-Patterns to Avoid

- **Don't fallback to inline.** The locked decision is single source of truth. If template missing:
  error and stop. No `|| cat > file << 'EOF'...` fallback.
- **Don't leave commented-out inline blocks.** The lock decision forbids backup comments. Remove
  the old heredoc entirely.
- **Don't update Claude Code without updating OpenCode simultaneously.** Both must stay in sync.
  The plan tasks must pair them.
- **Don't use `$ARGUMENTS` substitution for template variables.** Template variables come from
  runtime state (COLONY_STATE.json reads, date, etc.) — not from the command arguments.
- **Don't create a new template engine.** The LLM IS the template engine for markdown/JSON.
  Shell sed is fine for trivial cases. No Handlebars, no Jinja.
- **Don't update template content without updating the template file AND the command reference.**
  If you improve the crowned-anthill content, update `.aether/templates/crowned-anthill.template.md`
  (the source of truth), not just the command instruction.

---

## What Commands Need Changing

### init.md — Two inline JSON blocks (both Claude Code and OpenCode)

**Claude Code:** `.claude/commands/ant/init.md`
**OpenCode:** `.opencode/commands/ant/init.md`

**Inline Block 1 (Step 3 — COLONY_STATE.json):**
- Claude Code: Lines 215-244 (inline JSON block inside triple-backtick)
- Both versions have this block
- Template: `.aether/templates/colony-state.template.json`
- Change: Replace inline block with instruction to read template and fill placeholders

**Inline Block 2 (Step 4 — constraints.json):**
- Claude Code: Lines 250-255
- Both versions have this block
- Template: `.aether/templates/constraints.template.json`
- Change: Replace inline block with instruction to read template and write as-is
  (constraints template has no placeholders to fill — just strip underscore keys)

**OpenCode init.md differences:** OpenCode has `normalize-args` step at top, slightly different
version of steps, no step numbers matching Claude Code exactly. Must inspect both and update
both — cannot do a blind copy.

### seal.md — One inline heredoc (both Claude Code and OpenCode)

**Claude Code:** `.claude/commands/ant/seal.md`
**OpenCode:** `.opencode/commands/ant/seal.md`

**Inline Block (Step 6 — CROWNED-ANTHILL.md):**
- Claude Code: Lines 240-263 (`cat > .aether/CROWNED-ANTHILL.md << SEAL_EOF ... SEAL_EOF`)
- OpenCode seal.md is architecturally different from Claude Code — it also archives the colony;
  Claude Code seal.md does not archive (entomb is separate). The OpenCode seal has its own
  HANDOFF.md heredoc at lines 154-180 as the "final handoff" after archiving.
- Template: `.aether/templates/crowned-anthill.template.md`
- Change: Replace heredoc with template read instruction
- CRITICAL: Refresh content to triumphant, warm, narrative mood (locked decision)

**OpenCode seal.md differences:** OpenCode seal.md archives the colony at seal time (not a
separate entomb command). It has its own HANDOFF.md heredoc (lines 154-180) that describes a
"SEALED (Crowned Anthill)" state. This is architecturally different from Claude Code's
entomb-specific HANDOFF. The planners must decide: does the OpenCode seal HANDOFF get wired
to `handoff.template.md`, or is it a different scenario requiring its own template?

### entomb.md — Two inline structures (both Claude Code and OpenCode)

**Claude Code:** `.claude/commands/ant/entomb.md`
**OpenCode:** `.opencode/commands/ant/entomb.md`

**Inline Block 1 (Step 10 — colony state reset jq filter):**
- Claude Code: Lines 396-415 (inline jq filter in backtick block)
- Template: `.aether/templates/colony-state-reset.jq.template`
- Change: Replace with `jq -f <template_path>` call after resolving hub-first path

**Inline Block 2 (Step 11 — HANDOFF.md):**
- Claude Code: Lines 449-479 (`cat > .aether/HANDOFF.md << 'HANDOFF_EOF' ... HANDOFF_EOF`)
- Template: `.aether/templates/handoff.template.md`
- Change: Replace heredoc with template read instruction
- CRITICAL: Refresh content to reflective, warm, narrative mood (locked decision)

### build.md — Two inline HANDOFF heredocs (both Claude Code and OpenCode)

**Claude Code:** `.claude/commands/ant/build.md`
**OpenCode:** `.opencode/commands/ant/build.md`

**Inline Block 1 (Step 5.9 — error HANDOFF.md):**
- Claude Code: Lines 900-927 (`cat > .aether/HANDOFF.md << 'HANDOFF_EOF' ... HANDOFF_EOF`)
- Content: Error recovery context when workers fail
- No existing template for this scenario

**Inline Block 2 (Step 6.5 — success HANDOFF.md):**
- Claude Code: Lines 1055-1085 (`cat > .aether/HANDOFF.md << 'HANDOFF_EOF' ... HANDOFF_EOF`)
- Content: Build success summary for session recovery
- No existing template for this scenario

**Critical gap:** The existing `handoff.template.md` covers the entomb scenario only
("Colony Session — ENTOMBED"). Build's two HANDOFF heredocs are different scenarios.
Phase 24 either needs to create 1-2 new templates for these, or the planner must treat
build's HANDOFFs as out of scope for template wiring (wired to new templates created in
this phase, or deferred).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Template engine | Custom parser | LLM reads file directly | LLM IS the engine |
| Placeholder substitution (multi-line) | sed pipeline | LLM fills directly | sed breaks on multi-line `{{PHASE_RECAP}}` |
| Placeholder substitution (simple single-line) | New utility | `sed -e "s/{{VAR}}/$val/g"` | Already used throughout shell scripts |
| Template path resolution | New lookup function | Hub-first pattern from queen-init (lines 3378-3389) | Already proven in production |
| jq filter application | Inline jq string | `jq -f <template_path>` | Standard jq feature, zero complexity |

---

## Common Pitfalls

### Pitfall 1: OpenCode seal.md is Architecturally Different from Claude Code seal.md

**What goes wrong:** Treating seal.md as identical across both platforms. In Claude Code, seal only
writes CROWNED-ANTHILL.md and does NOT archive (entomb is separate). In OpenCode, seal archives
the colony and writes a HANDOFF.md. The OpenCode seal has its own inline HANDOFF.md heredoc
(lines 154-180) that the Claude Code seal does not have at all.

**Why it happens:** Looking only at Claude Code commands and assuming OpenCode is a mirror.

**How to avoid:** Read both files before planning any changes to seal.md. The planner must
specify different changes for each platform version.

**Warning signs:** If a plan says "update seal.md's heredoc" without specifying which file
and which heredoc, it's probably underspecified.

### Pitfall 2: Multi-line Template Values Break Shell Substitution

**What goes wrong:** Trying to use `sed` to substitute `{{PHASE_RECAP}}` — which is a multi-line
block of phase status entries — into the crowned-anthill template. sed substitution of multi-line
values is complex and error-prone in shell scripts.

**Why it happens:** Using Option B (sed) for all placeholders including multi-line ones.

**How to avoid:** Use LLM-direct approach (Option A) for markdown templates with multi-line
content. The LLM reads the template, sees the structure, fills `{{PHASE_RECAP}}` with the
actual phase list it computed in the bash loop above.

**Warning signs:** Any plan that specifies `sed -e "s/{{PHASE_RECAP}}/$phase_recap/g"` is
using the wrong approach.

### Pitfall 3: Forgetting to Remove HTML Comment Headers

**What goes wrong:** Template files start with HTML comment blocks:
```
<!-- Template: crowned-anthill | Version: 1.0 -->
<!-- Instructions: Fill all {{PLACEHOLDER}} values... -->
```
These must be stripped before writing the output file. If not stripped, they appear in
CROWNED-ANTHILL.md or HANDOFF.md, which looks terrible and confuses downstream commands
that parse these files.

**Why it happens:** Instruction just says "read template and fill values" without explicitly
saying "remove the comment header block."

**How to avoid:** Command instructions must explicitly say: "Remove the HTML comment header
lines (starting with `<!--`) before writing the output."

### Pitfall 4: Missing Template Missing-File Error Handling

**What goes wrong:** The template read happens without first checking if the template file
exists. If the file is missing (user didn't run `aether update`), the LLM gets a read error
that's confusing and doesn't guide recovery.

**Why it happens:** Optimistic path — assuming template exists.

**How to avoid:** Use the hub-first lookup pattern for all template resolutions. If no path
resolves, output the specific recovery message before stopping:
```
Template missing: {template_name}
Run aether update to fix.
```
This matches the locked decision exactly.

**For jq templates specifically:** The shell script needs an explicit check before `jq -f`.
If the file doesn't exist, `jq -f` will fail with a cryptic error, not the user-friendly message.

### Pitfall 5: Removing Inline Block Without Updating Cross-References

**What goes wrong:** The inline JSON block in init.md Step 3 is removed, but nearby text still
refers to "the JSON structure below" or "the v3.0 structure." Step 2.6 says "use empty arrays as
before" — this "as before" would be stale if the inline block is gone.

**Why it happens:** Removing the inline block but not reading the surrounding steps carefully.

**How to avoid:** When removing an inline block, read the 10 lines before and after it for
cross-references and update them. In init.md, Step 2.6 instructs to "use empty arrays as before"
— this becomes "use empty arrays for `__PHASE_LEARNINGS__` and `__INSTINCTS__`."

### Pitfall 6: `__PHASE_LEARNINGS__` and `__INSTINCTS__` Must Be JSON Arrays, Not Strings

**What goes wrong:** The LLM literally substitutes the string `__PHASE_LEARNINGS__` with
`"[]"` (a JSON string), producing `"phase_learnings": "[]"` instead of
`"phase_learnings": []`. This is valid JSON but wrong type — subsequent `jq` operations
that try to iterate `.memory.phase_learnings[]` will fail.

**Why it happens:** String placeholder substitution doesn't know about JSON types.

**How to avoid:** Command instructions must explicitly say:
"Replace `__PHASE_LEARNINGS__` with a JSON array value (e.g., `[]` not `"[]"`).
Replace `__INSTINCTS__` with a JSON array value."
The template's `_comment_memory` already says this — but the command instruction should
reinforce it.

---

## Code Examples

Verified patterns from codebase inspection:

### Hub-First Template Lookup Pattern (from aether-utils.sh lines 3378-3389)

This is the production-proven pattern. Adapt for each template in Phase 24:

```bash
# Template lookup: hub-first
template_file=""
for path in \
  "$HOME/.aether/system/templates/colony-state.template.json" \
  ".aether/templates/colony-state.template.json"; do
  if [[ -f "$path" ]]; then
    template_file="$path"
    break
  fi
done

if [[ -z "$template_file" ]]; then
  echo "Template missing: colony-state.template.json"
  echo "Run aether update to fix."
  exit 1
fi
```

### jq -f Template Execution (for entomb Step 10 replacement)

```bash
# After resolving jq_template path:
jq -f "$jq_template" .aether/data/COLONY_STATE.json.bak > .aether/data/COLONY_STATE.json
```

### LLM Instruction Pattern for JSON Template (init.md Step 3 replacement)

Instead of the inline JSON block, the command step becomes:

```
Resolve the colony-state template path:
  Check ~/.aether/system/templates/colony-state.template.json first,
  then .aether/templates/colony-state.template.json.

If no template found: output "Template missing: colony-state.template.json. Run aether update to fix." and stop.

Read the template file. Follow its _instructions field.
Replace all __PLACEHOLDER__ values:
  - __GOAL__ → {the user's goal from $ARGUMENTS}
  - __SESSION_ID__ → {generated session_id}
  - __ISO8601_TIMESTAMP__ → {current ISO-8601 UTC timestamp}
  - __PHASE_LEARNINGS__ → {JSON array from Step 2.6, or [] if none}
  - __INSTINCTS__ → {JSON array from Step 2.6, or [] if none}

Note: __PHASE_LEARNINGS__ and __INSTINCTS__ must be JSON arrays, not strings.

Remove ALL keys starting with underscore (_template, _version, _instructions, _comment_*).
Write the resulting JSON to .aether/data/COLONY_STATE.json using the Write tool.
```

### LLM Instruction Pattern for Markdown Template (seal.md Step 6 replacement)

Instead of the bash heredoc:

```
Resolve the crowned-anthill template path:
  Check ~/.aether/system/templates/crowned-anthill.template.md first,
  then .aether/templates/crowned-anthill.template.md.

If no template found: output "Template missing: crowned-anthill.template.md. Run aether update to fix." and stop.

Read the template file. Fill all {{PLACEHOLDER}} values:
  - {{GOAL}} → {goal}
  - {{SEAL_DATE}} → {seal_date}
  - {{VERSION}} → {version}
  - {{TOTAL_PHASES}} → {total_phases}
  - {{PHASES_COMPLETED}} → {phases_completed}
  - {{COLONY_AGE_DAYS}} → {colony_age_days}
  - {{PROMOTIONS_MADE}} → {promotions_made}
  - {{PHASE_RECAP}} → {phase recap list, one entry per line: "  - {phase_name}: {status}"}

Remove the HTML comment lines at the top of the template (lines starting with <!--).
Write the result to .aether/CROWNED-ANTHILL.md using the Write tool.
```

### Existing Template Content (actual files for reference)

**crowned-anthill.template.md placeholders:**
```
{{GOAL}}, {{SEAL_DATE}}, {{VERSION}}, {{TOTAL_PHASES}},
{{PHASES_COMPLETED}}, {{COLONY_AGE_DAYS}}, {{PROMOTIONS_MADE}}, {{PHASE_RECAP}}
```

**handoff.template.md placeholders:**
```
{{CHAMBER_NAME}}, {{GOAL}}, {{PHASES_COMPLETED}}, {{TOTAL_PHASES}},
{{MILESTONE}}, {{ENTOMB_TIMESTAMP}}
```

**colony-state.template.json placeholders:**
```
__GOAL__, __SESSION_ID__, __ISO8601_TIMESTAMP__, __PHASE_LEARNINGS__, __INSTINCTS__
```

**constraints.template.json:** No placeholders — write as-is after stripping underscore keys.

**colony-state-reset.jq.template:** No placeholders — use verbatim via `jq -f`.

---

## State of the Art

| Old Approach | New Approach (Phase 24) | Impact |
|--------------|------------------------|--------|
| Inline JSON blocks in command files | Template file read by LLM | LLM sees exact structure at point of use, not from memory |
| Shell heredoc for markdown generation | Template file read by LLM | Content refreshed, mood intentional, single source of truth |
| Inline jq filter string | `jq -f <template_path>` | Filter is versionable, testable, shareable |
| QUEEN.md template (Phase 20) | All 5 critical templates | Phase 24 extends to cover init/seal/entomb/build |

---

## Open Questions

1. **Build.md HANDOFF templates — create new templates in Phase 24 or defer?**
   - What we know: build.md has two HANDOFF.md heredocs (error handoff at Step 5.9,
     success handoff at Step 6.5) with different content from handoff.template.md
   - What's unclear: The CONTEXT.md says "(init, seal, entomb, build)" are in scope but
     doesn't specify whether build's HANDOFFs get new templates or existing template
   - Recommendation: Create two new templates: `handoff-build-error.template.md` and
     `handoff-build-success.template.md` in this phase. The locked decision says
     "both Claude Code AND OpenCode commands wired simultaneously" — build must be
     fully wired, which means its templates must exist.

2. **OpenCode seal.md HANDOFF heredoc — is it in scope?**
   - What we know: OpenCode seal.md has its own HANDOFF.md heredoc (lines 154-180)
     with content distinct from entomb's handoff. Claude Code seal.md does NOT have
     this heredoc (no archiving in Claude Code seal).
   - What's unclear: Does this OpenCode-specific heredoc get wired to a template?
   - Recommendation: Treat it as in scope — it's an inline heredoc in a command file.
     Either wire it to the existing `handoff.template.md` (if content is similar enough
     after refresh) or create `handoff-seal.template.md`. Recommend inspecting content
     first.

3. **Template content refresh for constraints.template.json — what to improve?**
   - What we know: The template has no placeholders and minimal content
     (`version`, `focus`, `constraints`). It's trivial. The locked decision says
     "internal templates also refreshed for cohesion."
   - What's unclear: There's not much to refresh — the file is 3 JSON fields.
   - Recommendation: Add a cleaner `_instructions` field and a `_comment_purpose` key
     explaining what constraints.json controls. The "refresh" is making the template
     self-documenting, not changing the data schema.

---

## Sources

### Primary (HIGH confidence)

All findings based on direct codebase inspection, 2026-02-20:

- `/Users/callumcowie/repos/Aether/.aether/templates/colony-state.template.json` — actual template content, placeholders verified
- `/Users/callumcowie/repos/Aether/.aether/templates/constraints.template.json` — actual template content
- `/Users/callumcowie/repos/Aether/.aether/templates/colony-state-reset.jq.template` — actual jq filter, verified executable
- `/Users/callumcowie/repos/Aether/.aether/templates/crowned-anthill.template.md` — actual template, placeholders verified
- `/Users/callumcowie/repos/Aether/.aether/templates/handoff.template.md` — actual template, placeholders verified
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md` — inline JSON blocks confirmed at Steps 3 (lines 215-244) and 4 (lines 250-255)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/seal.md` — inline heredoc confirmed at Step 6 (lines 240-263)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/entomb.md` — inline jq filter confirmed at Step 10 (lines 396-415), inline HANDOFF heredoc confirmed at Step 11 (lines 449-479)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` — two inline HANDOFF heredocs confirmed at Step 5.9 (lines 900-927) and Step 6.5 (lines 1055-1085)
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/init.md` — OpenCode version confirmed to have same inline blocks
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/seal.md` — OpenCode version confirmed to have additional HANDOFF heredoc (lines 154-180) not present in Claude Code seal.md
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 3373-3418 — hub-first template lookup pattern (`queen-init` command), confirmed production-proven pattern
- `/Users/callumcowie/repos/Aether/.planning/phases/21-template-foundation/21-VERIFICATION.md` — Phase 21 pass confirmed; all 5 templates verified as valid and distributed
- `/Users/callumcowie/repos/Aether/.planning/phases/24-template-integration/24-CONTEXT.md` — locked decisions, ceremony mood requirements

### Secondary (MEDIUM confidence — design documents)

- `/Users/callumcowie/repos/Aether/.planning/phases/21-template-foundation/21-RESEARCH.md` — placeholder conventions, lookup chain, distribution pipeline details

---

## Metadata

**Confidence breakdown:**
- What templates exist and their content: HIGH — direct file inspection
- What inline blocks need replacing and their exact locations: HIGH — direct file inspection of all 4 command files (both platforms)
- Template loading mechanism (hub-first pattern): HIGH — verified in production code (aether-utils.sh)
- Ceremony content refresh direction (triumphant/reflective moods): HIGH — locked decision from CONTEXT.md
- Build.md template scope (new templates needed): MEDIUM — inferred from locked decision that build is in scope, but exact scope of new templates is an open question

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (stable codebase, 30-day validity)
