# Phase 39: Quality Coverage - Research

**Researched:** 2026-02-22
**Domain:** Test coverage analysis, performance benchmarking, agent integration
**Confidence:** HIGH

## Summary

This phase integrates two specialist agents (Probe and Measurer) into the existing colony commands to improve test coverage and establish performance baselines. The integration follows the established pattern from Phase 38 (Gatekeeper and Auditor security gates).

**Probe** spawns conditionally in `/ant:continue` when test coverage falls below 80% after tests pass. It generates tests for uncovered code paths and discovers edge cases through mutation testing. Probe is strictly non-blocking — the colony continues even if coverage cannot be improved.

**Measurer** spawns conditionally in `/ant:build` for performance-sensitive phases (those containing keywords like "performance", "optimize", "latency", "throughput", "benchmark"). It establishes performance baselines for new code and identifies bottlenecks with recommendations.

Both agents are read-only (no code modification for Measurer, test-only modification for Probe) and log findings to the midden for future reference.

**Primary recommendation:** Follow the Phase 38 integration pattern — spawn agents conditionally after verification gates, parse their JSON output, and log warnings to midden without blocking phase advancement.

---

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| COV-01 | Probe spawns in `/ant:continue` Phase 4.5 when coverage < 80% after tests pass | Probe agent definition exists in `.opencode/agents/aether-probe.md`. Coverage check already exists in continue.md Step 1.5 (Phase 4). Insert Probe spawn after test verification, before secrets scan. |
| COV-02 | Probe generates tests for uncovered code paths | Probe agent has explicit mandate to generate test cases. Read-only boundaries restrict it to test files only (`.aether/docs/pheromones.md` line 125-132). |
| COV-03 | Probe discovers edge cases through mutation testing | Probe agent lists "mutation testing" as core responsibility (line 22). Output format includes `mutation_score` field (line 67). |
| COV-04 | Probe is non-blocking — continues even if can't improve coverage | Probe's `failure_modes` section specifies "Never fail silently" but also non-blocking behavior. Gate decision logic must NOT block on Probe results. |
| COV-05 | Measurer spawns in `/ant:build` Step 5.5 for performance-sensitive phases | build.md Step 5.5 currently processes Watcher results. Insert Measurer spawn after Watcher (Step 5.5), before Chaos (Step 5.6). |
| COV-06 | Measurer establishes performance baselines for new code | Measurer agent has "Establish performance baselines" as primary role (line 20). Output includes `baseline_vs_current` field (line 85). |
| COV-07 | Measurer identifies bottlenecks and provides recommendations | Measurer agent lists "Identify bottlenecks" and "Recommend optimizations" as responsibilities (lines 23-24). Output includes `recommendations` array with priority and estimated improvement (lines 93-95). |

---

## Standard Stack

### Core (Already in Project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| AVA | ^6.0.0 | Unit test runner | Already configured in package.json, used for all unit tests |
| Node.js built-in | >=16.0.0 | Performance timing | `process.hrtime()`, `performance.now()` available natively |
| aether-utils.sh | latest | Midden logging | `midden-write` function already exists for persistence |

### Supporting (No Additional Dependencies Required)
| Tool | Purpose | When to Use |
|------|---------|-------------|
| npm test | Test execution | Already used in verification loop |
| npm run test:coverage | Coverage reporting | If defined in package.json (currently not present) |
| git diff --stat | Change detection | For identifying new code to baseline |

### Coverage Tools (Optional Enhancement)
| Tool | Purpose | Notes |
|------|---------|-------|
| c8 | Native V8 coverage | Zero-config, works with AVA, no babel needed |
| nyc | Istanbul coverage | More features but requires configuration |

**Decision:** Do NOT add coverage tools as requirements. Probe should work with whatever coverage data is available (from `npm run test:coverage` if present, or static analysis if not).

---

## Architecture Patterns

### Pattern 1: Conditional Agent Spawn (from Phase 38)

**What:** Spawn specialist agent only when specific conditions are met, after primary verification passes.

**When to use:** For Probe (coverage < 80%) and Measurer (performance-sensitive phases).

**Example from Phase 38 (Gatekeeper integration):**
```markdown
### Step 1.8.1: Gatekeeper Security Gate (Conditional)

**Supply chain security audit — runs only when package.json exists.**

1. Check for package.json existence
2. Generate Gatekeeper name and log spawn
3. Update swarm display
4. Display spawn message
5. Spawn Gatekeeper agent with Task tool
6. Parse JSON output
7. Gate decision logic (block on critical, warn on high)
```

### Pattern 2: Non-Blocking Agent Integration (Probe-specific)

**What:** Agent runs and reports findings, but its results never block phase advancement.

**When to use:** For Probe (COV-04 requirement).

**Implementation:**
```markdown
**Probe Gate Decision Logic:**

- Probe always runs to completion
- Findings are logged to midden
- Phase advancement continues regardless of Probe results
- User is informed of coverage improvements (if any)
```

### Pattern 3: Performance Baseline Storage

**What:** Store performance metrics in midden for trend analysis across phases.

**When to use:** For Measurer baselines (COV-06).

**Storage format in midden:**
```json
{
  "category": "performance",
  "source": "measurer",
  "message": "Baseline established: api-response-time=45ms (Phase 39)",
  "timestamp": "2026-02-22T10:30:00Z"
}
```

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Coverage calculation | Custom coverage parser | Existing `npm run test:coverage` output | Probe should consume existing coverage data, not generate it |
| Performance timing | Custom benchmark framework | Node.js `performance.now()` or `process.hrtime()` | Native, high-resolution, no dependencies |
| Baseline storage | Custom database | midden-write utility | Already exists, integrated with colony logging |
| Test generation | Full test suite automation | Probe agent with bounded scope | Agent can generate tests but within test files only |

---

## Common Pitfalls

### Pitfall 1: Blocking on Coverage Improvements
**What goes wrong:** Probe fails to improve coverage, phase advancement stops.
**Why it happens:** Implementer copies Gatekeeper/Auditor gate logic which blocks on critical findings.
**How to avoid:** Explicitly document that Probe is advisory only. Always continue after Probe completes.
**Warning signs:** "Probe gate failed" language in implementation.

### Pitfall 2: Probe Modifying Source Code
**What goes wrong:** Probe changes implementation files to make tests pass.
**Why it happens:** Agent misunderstands read-only boundaries.
**How to avoid:** Probe's read_only section explicitly restricts to test files. Verify this in agent definition.
**Warning signs:** Files outside `tests/`, `__tests__/`, `*.test.*` being modified.

### Pitfall 3: Measurer Running on Every Phase
**What goes wrong:** Measurer spawns for all phases, creating noise.
**Why it happens:** Missing phase name/content filtering.
**How to avoid:** Implement keyword detection ("performance", "optimize", "latency", "throughput") before spawning.
**Warning signs:** Measurer spawning in documentation-only phases.

### Pitfall 4: Duplicate Coverage Checks
**What goes wrong:** Coverage checked in both continue.md and by Probe, causing redundant work.
**Why it happens:** Not leveraging existing verification loop data.
**How to avoid:** Pass coverage percentage from Step 1.5 verification loop to Probe context.
**Warning signs:** Two separate coverage report displays in output.

---

## Code Examples

### Probe Spawn in continue.md (Step 1.5 extension)

```markdown
### Step 1.5: Verification Loop Gate (MANDATORY)

... (existing verification loop) ...

**Coverage Check** (if coverage command exists):
Run using the Bash tool with description "Checking test coverage...": `{coverage_command}`
Record: coverage percentage (target: 80%+ for new code)

### Step 1.5.1: Probe Coverage Agent (Conditional)

**Test coverage improvement — runs only when coverage < 80% and tests pass.**

Check if coverage data exists and is below threshold:
- If coverage >= 80%: Skip silently, continue to Step 1.5.2
- If coverage < 80%: Proceed with Probe spawn

1. Generate Probe name and log spawn:
Run using the Bash tool with description "Generating Probe name...": `probe_name=$(bash .aether/aether-utils.sh generate-ant-name "probe") && bash .aether/aether-utils.sh spawn-log "Queen" "probe" "$probe_name" "Coverage analysis and test generation" && echo "{"name":"$probe_name"}"`

2. Update swarm display (if visual_mode is true):
Run using the Bash tool with description "Updating swarm display...": `bash .aether/aether-utils.sh swarm-display-update "$probe_name" "probe" "analyzing" "Coverage gap analysis" "Quality" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

3. Display: `🧪 Probe {name} spawning — Analyzing coverage gaps and generating tests...`

4. Spawn Probe agent with Task tool, include coverage data in context.

5. Parse Probe JSON output and log completion.

**Probe is NON-BLOCKING:** Continue to Step 1.5.2 regardless of Probe results.
```

### Measurer Spawn in build.md (Step 5.5 extension)

```markdown
### Step 5.5: Process Watcher Results

... (existing Watcher processing) ...

### Step 5.5.1: Measurer Performance Agent (Conditional)

**Performance analysis — runs only for performance-sensitive phases.**

Check if phase is performance-sensitive:
- Keywords: "performance", "optimize", "latency", "throughput", "benchmark", "speed", "memory"
- If no keywords match: Skip silently, continue to Step 5.6
- If keywords match: Proceed with Measurer spawn

1. Generate Measurer name and log spawn.
2. Update swarm display.
3. Display spawn message.
4. Spawn Measurer agent with Task tool, include modified files in context.
5. Parse Measurer JSON output.
6. Log findings to midden for future reference.

**Measurer is NON-BLOCKING:** Continue to Step 5.6 regardless of results.
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual coverage review | Probe agent auto-generation | Phase 39 (planned) | Coverage gaps addressed automatically |
| Ad-hoc performance checks | Measurer baseline establishment | Phase 39 (planned) | Performance regression detection enabled |
| Blocking quality gates | Non-blocking advisory agents | Phase 39 (planned) | Faster iteration, logged for review |

---

## Open Questions

1. **Coverage command detection**
   - What we know: Some projects have `npm run test:coverage`, others don't
   - What's unclear: Should Probe attempt to add coverage tooling if missing?
   - Recommendation: Probe works with available data; adding tooling is a separate concern

2. **Performance baseline comparison**
   - What we know: Measurer establishes baselines for new code
   - What's unclear: How to compare against previous baselines for existing code?
   - Recommendation: Store baselines in midden with file:line references for future comparison

3. **Probe test file location**
   - What we know: Probe writes to `tests/`, `__tests__/`, `*.test.*`
   - What's unclear: Should Probe create new test files or add to existing?
   - Recommendation: Probe should prefer adding to existing test files, create new only when no existing file covers the module

---

## Sources

### Primary (HIGH confidence)
- `.opencode/agents/aether-probe.md` — Agent definition, capabilities, output format, read-only boundaries
- `.opencode/agents/aether-measurer.md` — Agent definition, capabilities, output format, read-only boundaries
- `.claude/commands/ant/continue.md` — Integration point for Probe (Step 1.5 verification loop)
- `.claude/commands/ant/build.md` — Integration point for Measurer (Step 5.5 after Watcher)
- `.aether/aether-utils.sh:6816-6868` — midden-write utility function

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` — COV-01 through COV-07 requirements specification
- `.planning/STATE.md` — Phase 38 completion status, integration patterns established
- `package.json` — Test infrastructure (AVA), no coverage tool currently configured

### Tertiary (LOW confidence)
- None — all findings verified against source files

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all tools already in project
- Architecture: HIGH — follows established Phase 38 pattern
- Pitfalls: MEDIUM — derived from agent definitions, not yet tested in practice

**Research date:** 2026-02-22
**Valid until:** 2026-03-22 (30 days for stable integration pattern)
