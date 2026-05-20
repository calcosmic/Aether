# Domain Pitfalls: Grounded Planning + Ceremony Restore

**Domain:** Aether colony framework -- adding survey noise filtering, plan grounding validation, decision binding, and ceremony restore to an existing hybrid Go/TypeScript/YAML system with 3 platform surfaces (Claude, OpenCode, Codex).
**Researched:** 2026-05-18

## Critical Pitfalls

### Pitfall 1: Noise Filter Divergence
**What goes wrong:** You fix the `.venv` bug in `shouldSkipSurveyDir` but forget to fix `dirsToSkip` in codegraph. Plans improve but codegraph still scans virtual environments for dependency graphs. Or vice versa.
**Why it happens:** Two separate skip lists in two different files (`cmd/codex_colonize.go:978` and `pkg/codegraph/codegraph.go:72`), maintained independently with no shared contract.
**Consequences:** Survey says "Python project with 847 source files" (polluted by site-packages) while codegraph says "Python project with 12 source files" (correct). Plans produced from survey data reference virtual environment files that do not exist in the user's mental model.
**Prevention:** Create `pkg/codegraph/scan_filter.go` as the single source of truth. Both `codegraph.Scan()` and `surveyWorkspace()` call `ShouldSkipDir` and `ShouldSkipFile`. Delete the inline lists from both call sites.
**Detection:** Add a test that lists all dirs skipped by codegraph and colonize; assert they are identical. Any divergence fails the test.

### Pitfall 2: Grounding Gate False Positives
**What goes wrong:** The grounding gate warns on every research phase and architecture decision phase because these legitimately have no file references. Users learn to ignore the warning, defeating its purpose.
**Why it happens:** The grounding check is purely structural ("does the plan text contain file path patterns?") without understanding phase semantics.
**Consequences:** Warning fatigue. Users either ignore grounding warnings or file issues asking to disable them.
**Prevention:** Make the gate phase-aware. If the phase name contains "research", "oracle", "architect", "survey", "infrastructure", or "deploy", suppress the grounding warning. Only warn for phases with names like "implement", "build", "add", "refactor", "fix" where file references are expected.
**Detection:** Test the gate with known phase types. Research phase -> no warning. Implementation phase with no files -> warning. Implementation phase with files -> no warning.

### Pitfall 3: Playbook Parse Brittleness
**What goes wrong:** The TS host tries to parse playbook step headers (`### Step 5.1: Spawn Wave 1 Workers`) with a regex, but playbooks use inconsistent formatting. Some steps have sub-steps (`### Step 5.0.1: Oracle Research Step`), some have note blocks, some have code blocks that look like headers.
**Why it happens:** Playbooks are free-form markdown written by multiple authors over time. They were designed for human reading, not programmatic execution.
**Consequences:** The TS host skips steps, executes steps out of order, or crashes on unexpected formatting. Ceremony becomes unreliable.
**Prevention:** Do NOT try to parse playbooks into a structured execution graph. Instead, treat each playbook as a single document to be injected into the orchestration context. The LLM (worker) already knows how to follow markdown instructions -- it has been doing so via `renderBuildPlaybookContext`. The TS host's job is to load the playbook and make it available, not to parse and execute it step by step.
**Detection:** Test with all 13 existing playbooks. Any parsing failure means the approach is wrong. If you cannot reliably parse all 13 files, switch to document-injection instead of step-execution.

### Pitfall 4: Triple Conductor Regression
**What goes wrong:** After ceremony restore, YAML command definitions, TS host lifecycle.ts, and Go runtime ceremony commands all try to own the orchestration flow. A change in one breaks the others.
**Why it happens:** The root cause of the current ceremony degradation. YAML defines ceremony in markdown wrappers, TS host has its own orchestration, Go has ceremony commands. No clear ownership boundary.
**Consequences:** Ceremony works on one platform but not another. A fix in Go ceremony breaks TS host orchestration. Users see different ceremony on Claude Code vs Codex.
**Prevention:** Establish clear ownership:
- **Playbooks** define WHAT ceremony steps happen (the script)
- **TS host** decides WHEN to follow the script (the reader)
- **Go ceremony adapter** decides HOW it looks (the renderer)
- **YAML** defines command metadata (name, flags, runtime command)

No two of these should duplicate ceremony content. YAML does not contain ceremony steps. Go does not decide ceremony flow. TS host does not render ceremony visuals.

**Detection:** Grep for ceremony step content (e.g., "spawn plan", "wave start") across YAML, TS, and Go. Each ceremony element should appear in exactly one place.

## Moderate Pitfalls

### Pitfall 5: Source Anchor Over-Selection
**What goes wrong:** Source anchors include too many files (e.g., all 847 Python files in a Django project), making the plan noisy and hard to read.
**Why it happens:** The anchor extraction algorithm does not have a meaningful cap or prioritization.
**Consequences:** Plans become unreadable with 50+ file references. The grounding gate passes (files are referenced) but the plan is worse than a generic one.
**Prevention:** Cap source anchors at 50, prioritized by path depth (shallowest first = most important). This gives the planner the top-level source structure without drowning in leaf files. The planner can discover additional files via codegraph during task execution.
**Detection:** Test anchor extraction on repos of varying sizes. Assert cap of 50. Assert shallow files appear before deep files.

### Pitfall 6: Discuss Decision Pheromone Spam
**What goes wrong:** Every resolved discuss question emits a REDIRECT pheromone, even for soft preferences ("I prefer tabs over spaces"). Workers see 15 REDIRECT signals and cannot distinguish critical constraints from style preferences.
**Why it happens:** Auto-emitting pheromones without filtering for `HardConstraint: true`.
**Consequences:** Workers ignore REDIRECT signals (signal fatigue), or over-constrain their work.
**Prevention:** Only auto-emit REDIRECT for `HardConstraint: true` decisions. Soft preferences should use FEEDBACK pheromones (lower priority) or not emit at all.
**Detection:** Test discuss resolution. Hard constraint -> REDIRECT emitted. Soft preference -> no pheromone.

### Pitfall 7: Playbook Path Resolution Failure
**What goes wrong:** The TS host cannot find playbooks because it looks in the wrong directory. Playbooks live in `.aether/docs/command-playbooks/` within the repo, but the TS host might look in the hub or the Aether repo itself.
**Why it happens:** `buildPlaybookCandidates()` in Go resolves paths from repo root, hub path, and absolute path. The TS host has its own path resolution logic that may not match.
**Consequences:** Playbook not found, ceremony falls back to inline defaults, or worse, crashes.
**Prevention:** Reuse the exact same resolution logic from Go. The TS host should call `aether playbook-resolve --name build-full` (new Go command) to get the resolved path, rather than implementing its own resolution.
**Detection:** Test playbook resolution with repos in different states (fresh, hub-installed, custom playbook paths).

## Minor Pitfalls

### Pitfall 8: Noise Filter Over-Exclusion
**What goes wrong:** Adding `env` to the noise filter breaks projects that have a source directory called `env/` (common in some monorepo layouts).
**Why it happens:** `.env` directory name is ambiguous -- could be a Python virtual environment or a source directory for environment configuration.
**Prevention:** Only exclude `.env` (with leading dot), not `env` (without dot). Most virtual environments use `.venv` or `.env` with a leading dot. Source directories typically use `env` or `environments` without a dot.
**Detection:** Test with a repo that has both `env/` (source) and `.venv/` (virtual environment).

### Pitfall 9: Grounding Regex False Negatives
**What goes wrong:** The grounding gate regex fails to detect file references in some plan formats. For example, a plan that says "Modify the user middleware" (without a path) has a real file target but no regex match.
**Why it happens:** File reference detection relies on path patterns, not semantic understanding of which files the task targets.
**Consequences:** Grounding gate warns even though the plan is specific. False warning reduces trust in the gate.
**Prevention:** Accept this limitation. The grounding gate is a heuristic, not a perfect validator. It catches the most common failure mode (completely generic plans) and occasionally false-positives. Document that it is a best-effort check.
**Detection:** Test with various plan formats. Generic plan -> warns correctly. Specific plan with paths -> no warning. Specific plan without paths -> may warn (acceptable false positive).

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Shared noise filter | Divergence between codegraph and colonize | Single `scan_filter.go` module; test both call sites |
| Source anchor extraction | Over-selection; `.env` ambiguity | Cap at 50; only exclude dotfiles |
| Plan grounding gate | False positives on research phases; false negatives on semantic specificity | Phase-aware suppression; document heuristic nature |
| Discuss decision binding | Pheromone spam from soft preferences | Only auto-emit REDIRECT for HardConstraint: true |
| Playbook-driven TS orchestration | Parse brittleness; path resolution failure | Document-injection model (not step-execution); reuse Go path resolution |
| Ceremony ownership clarification | Triple conductor regression | Clear ownership: playbooks=script, TS=reader, Go=renderer, YAML=metadata |

## Sources

- Existing bug evidence: `.planning/PROJECT.md` line 41 ("Known bug: survey trusts dependency/cache paths (.venv, __pycache__); plans come out generic")
- Existing gap evidence: `.planning/PROJECT.md` line 42 ("Known gap: ceremony playbooks exist but commands no longer load them")
- Divergent skip lists: `pkg/codegraph/codegraph.go:72` (13 dirs) vs `cmd/codex_colonize.go:978` (10 dirs, different set)
- Playbook format analysis: `.aether/docs/command-playbooks/` (inconsistent step numbering, sub-steps, code blocks)
- Pheromone system: `cmd/discuss.go` (HardConstraint field), existing pheromone-write with dedup and priority
