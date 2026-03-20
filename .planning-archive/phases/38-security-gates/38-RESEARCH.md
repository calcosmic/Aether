# Phase 38: Security Gates - Research

**Researched:** 2026-02-22
**Domain:** Security/Quality Gates for Colony Workflow
**Confidence:** HIGH

## Summary

This phase integrates two existing agents (Gatekeeper and Auditor) into the `/ant:continue` verification workflow. The research focuses on understanding:

1. **Agent capabilities** - Both agents are already defined in `.opencode/agents/` with clear output schemas
2. **Integration patterns** - The `/ant:continue` command has a well-structured verification loop with multiple gates
3. **Spawn mechanics** - Workers are spawned via Task tool with `subagent_type` parameter
4. **Midden logging** - Existing infrastructure for logging warnings without blocking

**Primary recommendation:** Insert security gates as Step 1.8 in `/ant:continue`, between Watcher verification (Step 1.7) and TDD Evidence Gate (Step 1.8, renumber to 1.9). Gatekeeper runs first (supply chain), then Auditor (code quality). Both use existing JSON output schemas for gate decisions.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Spawn Triggers:** Gatekeeper spawns when `package.json` exists in project root; skips gracefully if absent. Auditor spawns on every `/ant:continue`.
- **Blocking Severity:** Critical CVEs = hard block (no override); High CVEs = warn and continue, log to midden; Auditor quality score < 60 = hard block; Auditor critical findings = hard block.
- **Integration Point:** Security gates run after Watcher verification (Step 1.8 area); sequential order: Gatekeeper first, then Auditor; both gates must pass before phase can advance; replaces existing basic grep security scan (Step 1.5, Phase 5).
- **Override Behavior:** Hard blocks have no user-facing override; non-code phases can skip security gates entirely; no `--skip-security` flag; manual override possible via editing COLONY_STATE.json.
- **Agent Constraints:** Both agents are strictly read-only; neither will modify code, create files, or update colony state; if asked to modify: refuse and suggest appropriate agent (Builder, Tracker).

### Claude's Discretion
- Auditor spawn frequency (decided: always spawn for consistent coverage)
- Exact threshold values (60 for quality, could adjust later)
- Error message wording for blocked continues

### Deferred Ideas (OUT OF SCOPE)
- Emoji consistency investigation
- Support for other package manifests (requirements.txt, Cargo.toml, go.mod, etc.)
- Configurable severity thresholds
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SEC-01 | Gatekeeper spawns in `/ant:continue` Phase 5.5 when package manifest exists | Gatekeeper agent defined in `.opencode/agents/aether-gatekeeper.md` with clear output schema; spawn via Task tool with `subagent_type="aether-gatekeeper"` |
| SEC-02 | Gatekeeper performs CVE scanning, license compliance, supply chain security audit | Agent definition includes security scanning (CVE database, malicious package detection, typo squatting), license compliance (MIT/Apache/BSD vs GPL/AGPL), dependency health checks |
| SEC-03 | Gatekeeper blocks on critical CVEs, warns on high CVEs | Output schema includes `security.critical`, `security.high` counts; blocking logic: if `critical > 0` = hard block; if `high > 0` = warn + log to midden |
| SEC-04 | Auditor spawns in `/ant:continue` Step 1.8.5 when UI or API changes detected | Auditor agent defined in `.opencode/agents/aether-auditor.md`; decision: spawn on every continue for consistent coverage (per Claude's discretion) |
| SEC-05 | Auditor performs multi-lens review (security, performance, quality, maintainability) | Agent definition includes 4 audit dimensions: Security Lens, Performance Lens, Quality Lens, Maintainability Lens |
| SEC-06 | Auditor gate fails if overall_score < 60 or critical findings > 0 | Output schema includes `overall_score` and `findings.critical`; blocking logic: if `overall_score < 60` or `findings.critical > 0` = hard block |
</phase_requirements>

## Standard Stack

### Core (Already Exists)
| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| Gatekeeper Agent | `.opencode/agents/aether-gatekeeper.md` | Supply chain security scanning | Already defined, read-only, JSON output |
| Auditor Agent | `.opencode/agents/aether-auditor.md` | Code quality multi-lens review | Already defined, read-only, JSON output |
| Midden System | `.aether/aether-utils.sh:6674-6787` | Archive expired signals, log warnings | Existing infrastructure for non-blocking warnings |
| Spawn Logging | `.aether/aether-utils.sh` spawn-log, spawn-complete | Track worker spawns for gates | Used by all existing worker spawns |
| Verification Loop | `.aether/docs/disciplines/verification-loop.md` | 6-phase quality check pattern | Security gates extend this pattern |

### Supporting
| Component | Location | Purpose | When to Use |
|-----------|----------|---------|-------------|
| npm audit | Built-in | CVE scanning for Node projects | When package.json exists |
| grep patterns | Existing in continue.md | Secret scanning (debug artifacts) | Keep as fallback |
| COLONY_STATE.json | `.aether/data/COLONY_STATE.json` | State persistence | Store gate results |

## Architecture Patterns

### Pattern 1: Agent Spawn Pattern
**What:** Spawn agents via Task tool with `subagent_type` parameter
**When to use:** All agent spawns in the colony
**Example (from build.md:506):**
```markdown
For each Wave 1 task, use Task tool with `subagent_type="aether-builder"`, include `description: "🔨 Builder {Ant-Name}: {task_description}"` (DO NOT use run_in_background - multiple Task calls in a single message run in parallel and block until complete):
```

### Pattern 2: Gate with Hard Block
**What:** Check condition, display blocking message, STOP progression
**When to use:** Critical CVEs, quality score < 60, critical findings
**Example (from continue.md:185-205):**
```markdown
**If NOT READY (any of: build fails, tests fail, critical security issues, success criteria unmet):**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔🐜 V E R I F I C A T I O N   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase {id} cannot advance until issues are resolved.

🚨 Issues Found:
{list each failure with specific evidence}

🔧 Required Actions:
  1. Fix the issues listed above
  2. Run /ant:continue again to re-verify

The phase will NOT advance until verification passes.
```

**CRITICAL:** Do NOT proceed to Step 2. Do NOT advance the phase.
```

### Pattern 3: Warn and Continue with Midden Log
**What:** Display warning, log to midden for later review, continue progression
**When to use:** High CVEs (non-critical)
**Example (from aether-utils.sh:6674-6787):**
```bash
# Midden directory for archived signals
phe_midden_dir="$DATA_DIR/midden"
phe_midden_file="$phe_midden_dir/midden.json"

# Append expired signals to midden.json
phe_midden_updated=$(jq --argjson new_signals "$phe_expired_objects" '
  .signals += $new_signals |
  .archived_at_count = (.signals | length)
' "$phe_midden_file" 2>/dev/null)
```

### Pattern 4: JSON Output Parsing for Gate Decisions
**What:** Agents return JSON, parse specific fields for gate logic
**When to use:** All agent spawns where decisions depend on output
**Gatekeeper Output Schema:**
```json
{
  "ant_name": "{your name}",
  "caste": "gatekeeper",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you accomplished",
  "security": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0
  },
  "licenses": {},
  "outdated_packages": [],
  "recommendations": [],
  "blockers": []
}
```

**Auditor Output Schema:**
```json
{
  "ant_name": "{your name}",
  "caste": "auditor",
  "status": "completed" | "failed" | "blocked",
  "summary": "What you accomplished",
  "dimensions_audited": [],
  "findings": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0,
    "info": 0
  },
  "issues": [
    {"severity": "HIGH", "location": "file:line", "issue": "", "fix": ""}
  ],
  "overall_score": 0,
  "recommendation": "",
  "blockers": []
}
```

### Pattern 5: Conditional Spawn Based on File Existence
**What:** Check if file exists before spawning agent
**When to use:** Gatekeeper (only spawn if package.json exists)
**Example:**
```bash
if [[ -f "package.json" ]]; then
  # Spawn Gatekeeper
else
  # Skip gracefully with note
fi
```

### Anti-Patterns to Avoid
- **Don't spawn agents in parallel if sequential dependency exists:** Gatekeeper must complete before Auditor (supply chain before code quality)
- **Don't modify agent behavior:** Agents are read-only by design; don't ask them to write files
- **Don't create new output schemas:** Use existing JSON schemas for gate decisions
- **Don't skip midden logging for warnings:** High CVEs must be logged for later review

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CVE scanning | Custom vulnerability scanner | Gatekeeper agent with npm audit | Agent already exists, standardized output |
| Code quality review | Custom linter integration | Auditor agent multi-lens review | Agent already exists, covers 4 dimensions |
| Warning archival | Custom logging system | Midden system (aether-utils.sh) | Already exists, integrated with pheromone expiry |
| Worker spawn tracking | Custom spawn registry | spawn-log / spawn-complete utilities | Already used by all workers, consistent |
| Gate display formatting | Custom banner code | Existing gate banner patterns | Consistent UX with other gates |

**Key insight:** Both agents already exist with well-defined output schemas. The work is integration, not creation.

## Common Pitfalls

### Pitfall 1: Blocking on High CVEs Instead of Critical Only
**What goes wrong:** User specified critical = hard block, high = warn. If implementation blocks on high CVEs, it violates user constraints.
**Why it happens:** Security best practices often suggest blocking on high, but user explicitly wants warn-and-continue for highs.
**How to avoid:** Check severity mapping carefully. Only `security.critical > 0` blocks. `security.high > 0` warns + logs to midden.
**Warning signs:** Gate fails on moderate security issues that shouldn't block development flow.

### Pitfall 2: Spawning Agents in Parallel When Sequential Required
**What goes wrong:** Gatekeeper and Auditor spawned simultaneously. Auditor may check dependencies that Gatekeeper would flag as vulnerable.
**Why it happens:** Parallel spawning is faster, but loses dependency ordering.
**How to avoid:** Always spawn Gatekeeper first, wait for completion, check results, then spawn Auditor.
**Warning signs:** Auditor reports issues on packages that Gatekeeper would have flagged.

### Pitfall 3: Missing Midden Log for High CVEs
**What goes wrong:** High CVEs are warned but not logged to midden. User loses visibility into recurring issues.
**Why it happens:** Focus on blocking logic, forget warning persistence.
**How to avoid:** Explicit midden-write call for high CVEs before continuing.
**Warning signs:** Same high CVEs appear across multiple continues with no record.

### Pitfall 4: Modifying Agent Definitions
**What goes wrong:** Trying to "improve" Gatekeeper/Auditor behavior during integration.
**Why it happens:** Natural urge to refine while touching.
**How to avoid:** Agents are out of scope per CONTEXT.md. Only modify `/ant:continue` command.
**Warning signs:** Changes to `.opencode/agents/aether-*.md` files.

### Pitfall 5: Breaking Existing Security Scan
**What goes wrong:** Replacing grep-based secret scan entirely, losing debug artifact detection.
**Why it happens:** Focus on new CVE scanning, forget existing secret scanning.
**How to avoid:** Keep existing Phase 5 security scan (secrets, debug artifacts). Gatekeeper adds CVE scanning, doesn't replace secret scanning.
**Warning signs:** No grep for `sk-`, `api_key`, `console.log`, `debugger` in verification loop.

## Code Examples

### Gatekeeper Spawn (Step 1.8.1)

```markdown
### Step 1.8.1: Gatekeeper Security Gate (Conditional)

**Check for package.json:**
Run using the Bash tool with description "Checking for package manifest...": `test -f package.json && echo "exists" || echo "missing"`

**If package.json missing:**
```
📦🐜 Gatekeeper: No package.json found — skipping supply chain audit
```
Continue to Step 1.8.2 (Auditor gate).

**If package.json exists:**

Generate Gatekeeper name and dispatch:
Run using the Bash tool with description "Naming gatekeeper...": `bash .aether/aether-utils.sh generate-ant-name "gatekeeper"` (store as `{gatekeeper_name}`)
Run using the Bash tool with description "Dispatching gatekeeper...": `bash .aether/aether-utils.sh spawn-log "Queen" "gatekeeper" "{gatekeeper_name}" "Supply chain security audit" && bash .aether/aether-utils.sh swarm-display-update "{gatekeeper_name}" "gatekeeper" "scanning" "Supply chain security audit" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 20`

Display:
```
📦🐜 Gatekeeper {gatekeeper_name} spawning
    Scanning dependencies for CVEs and license compliance...
```

Spawn Gatekeeper using Task tool with `subagent_type="aether-gatekeeper"`, include `description: "📦 Gatekeeper {gatekeeper_name}: Supply chain security audit"`:
# FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Gatekeeper Ant - guards what enters the codebase through dependency and supply chain analysis."

```
You are {gatekeeper_name}, a 📦🐜 Gatekeeper Ant.

Mission: Supply chain security audit

Context:
- Project has package.json (Node.js/npm project)
- This is a verification gate before phase advancement

Work:
1. Inventory all dependencies from package.json
2. Run: npm audit --json (or npm audit if JSON unavailable)
3. Check for known CVEs in dependencies
4. Audit licenses for compliance (check node_modules/*/package.json)
5. Assess dependency health (outdated, maintenance status)

Log activity: bash .aether/aether-utils.sh activity-log "SCANNING" "{gatekeeper_name}" "description"

Report (JSON only):
{
  "ant_name": "{gatekeeper_name}",
  "caste": "gatekeeper",
  "status": "completed" | "failed" | "blocked",
  "summary": "Dependency count and key findings",
  "security": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0
  },
  "licenses": {
    "permissive": ["mit", "apache-2.0"],
    "copyleft": [],
    "unknown": []
  },
  "outdated_packages": [],
  "recommendations": [],
  "blockers": []
}
```

**Wait for results** (blocking — use TaskOutput with `block: true`).

Log completion:
Run using the Bash tool with description "Recording gatekeeper findings...": `bash .aether/aether-utils.sh spawn-complete "{gatekeeper_name}" "completed" "Supply chain audit" && bash .aether/aether-utils.sh swarm-display-update "{gatekeeper_name}" "gatekeeper" "completed" "Supply chain security audit" "Queen" '{"read":3,"grep":5,"edit":0,"bash":2}' 100 "fungus_garden" 100`

**Parse Gatekeeper JSON output:**
- Extract: `security.critical`, `security.high`, `security.medium`, `security.low`
- Extract: `status`, `blockers`

**Gate Decision:**

If `security.critical > 0`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔📦🐜 G A T E K E E P E R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{security.critical} critical CVE(s) found in dependencies.

🚨 Critical Issues:
{list critical CVEs with package names}

🔧 Required Actions:
  1. Update affected packages: npm update {package}
  2. Or run: npm audit fix
  3. Re-run /ant:continue after fixing

The phase will NOT advance with critical CVEs.
```
Do NOT proceed to Step 1.8.2. Do NOT advance the phase.

If `security.high > 0`:
```
⚠️📦🐜 Gatekeeper: {security.high} high severity CVE(s) found

Logged to midden for later review. Continuing...
```
Log to midden:
Run using the Bash tool with description "Logging high CVEs to midden...": `bash .aether/aether-utils.sh midden-write "security" "High CVEs found: {count}" "gatekeeper"`
Continue to Step 1.8.2.

If `security.critical == 0`:
```
✅📦🐜 Gatekeeper: No critical CVEs found
```
Continue to Step 1.8.2.
```

### Auditor Spawn (Step 1.8.2)

```markdown
### Step 1.8.2: Auditor Quality Gate

Generate Auditor name and dispatch:
Run using the Bash tool with description "Naming auditor...": `bash .aether/aether-utils.sh generate-ant-name "auditor"` (store as `{auditor_name}`)
Run using the Bash tool with description "Dispatching auditor...": `bash .aether/aether-utils.sh spawn-log "Queen" "auditor" "{auditor_name}" "Code quality audit" && bash .aether/aether-utils.sh swarm-display-update "{auditor_name}" "auditor" "reviewing" "Code quality audit" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 30`

Display:
```
👥🐜 Auditor {auditor_name} spawning
    Reviewing code with multi-lens analysis...
```

Spawn Auditor using Task tool with `subagent_type="aether-auditor"`, include `description: "👥 Auditor {auditor_name}: Code quality audit"`:
# FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are an Auditor Ant - examines code with specialized lenses for security, performance, quality, and maintainability."

```
You are {auditor_name}, a 👥🐜 Auditor Ant.

Mission: Code quality audit

Context:
- Phase {current_phase} completed, awaiting advancement
- Files modified: {list from git diff --name-only}

Work:
1. Read modified files to understand changes
2. Apply Security Lens: input validation, auth, secrets
3. Apply Performance Lens: algorithms, queries, memory
4. Apply Quality Lens: readability, tests, error handling
5. Apply Maintainability Lens: coupling, debt, duplication

Log activity: bash .aether/aether-utils.sh activity-log "REVIEWING" "{auditor_name}" "description"

Report (JSON only):
{
  "ant_name": "{auditor_name}",
  "caste": "auditor",
  "status": "completed" | "failed" | "blocked",
  "summary": "Files reviewed and key findings",
  "dimensions_audited": ["security", "performance", "quality", "maintainability"],
  "findings": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0,
    "info": 0
  },
  "issues": [
    {"severity": "HIGH", "location": "file:line", "issue": "description", "fix": "suggestion"}
  ],
  "overall_score": 75,
  "recommendation": "Top priority fix",
  "blockers": []
}
```

**Wait for results** (blocking — use TaskOutput with `block: true`).

Log completion:
Run using the Bash tool with description "Recording auditor findings...": `bash .aether/aether-utils.sh spawn-complete "{auditor_name}" "completed" "Code quality audit" && bash .aether/aether-utils.sh swarm-display-update "{auditor_name}" "auditor" "completed" "Code quality audit" "Queen" '{"read":5,"grep":3,"edit":0,"bash":1}' 100 "fungus_garden" 100`

**Parse Auditor JSON output:**
- Extract: `findings.critical`, `overall_score`
- Extract: `status`, `blockers`

**Gate Decision:**

If `findings.critical > 0`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔👥🐜 A U D I T O R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

{findings.critical} critical finding(s) found.

🚨 Critical Issues:
{list critical findings with file:line}

🔧 Required Actions:
  1. Fix the critical issues listed above
  2. Re-run /ant:continue after fixing

The phase will NOT advance with critical findings.
```
Do NOT proceed to Step 1.9. Do NOT advance the phase.

If `overall_score < 60`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔👥🐜 A U D I T O R   G A T E   F A I L E D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Quality score: {overall_score}/100 (threshold: 60)

🚨 Quality Below Threshold

🔧 Required Actions:
  1. Address the issues identified in the audit
  2. Re-run /ant:continue after improvements

The phase will NOT advance with quality score below 60.
```
Do NOT proceed to Step 1.9. Do NOT advance the phase.

If `findings.critical == 0 && overall_score >= 60`:
```
✅👥🐜 Auditor: Quality score {overall_score}/100 — PASSED
```
Continue to Step 1.9.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Basic grep for secrets (Phase 5) | Gatekeeper CVE scanning + license audit | Phase 38 | Professional supply chain security |
| No code quality gate | Auditor multi-lens review | Phase 38 | Systematic quality review before advancement |
| Manual security review | Automated agent-based scanning | Phase 38 | Consistent, repeatable security gates |

**Deprecated/outdated:**
- None — existing grep secret scan stays as Phase 5 (complementary to new gates)

## Open Questions

1. **Midden logging API for high CVEs**
   - What we know: Midden system exists in aether-utils.sh for archiving expired pheromones
   - What's unclear: Is there a direct `midden-write` command or should we use a different approach?
   - Recommendation: Check if `midden-write` subcommand exists; if not, append directly to `.aether/data/midden/midden.json` with proper JSON structure

2. **Non-code phase detection**
   - What we know: Docs-only phases can skip security gates
   - What's unclear: How to reliably detect "docs-only" phase (check file extensions? task descriptions?)
   - Recommendation: Check if all modified files are `.md`, `.txt`, or docs-only; if so, skip gates with note

3. **Agent fallback behavior**
   - What we know: Fallback pattern exists for "Agent type not found" errors
   - What's unclear: Whether Gatekeeper/Auditor agents are registered in all environments
   - Recommendation: Include fallback instructions in spawn prompts (already in examples above)

## Sources

### Primary (HIGH confidence)
- `.opencode/agents/aether-gatekeeper.md` - Agent definition, output schema, security scanning capabilities
- `.opencode/agents/aether-auditor.md` - Agent definition, output schema, 4-lens audit dimensions
- `.claude/commands/ant/continue.md` - Verification loop structure, gate patterns, spawn mechanics
- `.claude/commands/ant/build.md` - Worker spawn examples with Task tool and subagent_type
- `.aether/aether-utils.sh:6674-6787` - Midden system implementation
- `.aether/docs/disciplines/verification-loop.md` - 6-phase verification pattern

### Secondary (MEDIUM confidence)
- `.planning/phases/38-security-gates/38-CONTEXT.md` - User decisions and constraints
- `.aether/aether-utils.sh` spawn-log, spawn-complete, generate-ant-name - Worker tracking utilities

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All components exist and are documented
- Architecture: HIGH - Clear integration point in continue.md Step 1.8 area
- Pitfalls: MEDIUM-HIGH - Based on user constraints and existing patterns

**Research date:** 2026-02-22
**Valid until:** 2026-03-22 (30 days for stable architecture)
