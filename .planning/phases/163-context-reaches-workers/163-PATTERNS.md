# Phase 163: Context Reaches Workers - Pattern Map

**Mapped:** 2026-07-29
**Files analyzed:** 16 (9 modification targets, 3 new files, 1 reference-only, 3 distributed-docs)
**Analogs found:** 14 / 16 (2 have no direct analog — called out explicitly, not force-matched)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/codex_build.go` (`codexBuildManifest`, `attachBuildDispatchContext`) | controller / manifest-composer | transform (compute-once, share) | `cmd/codex_build.go:1400` (`executeCodexBuildDispatches`, Path A) | exact — same file, proven twin path |
| `cmd/hook_cmds.go` (`protectedHookWriteReason`) | middleware / guard | request-response | itself — sibling `switch` cases in the same function | exact |
| `cmd/hook_cmds_test.go` | test | request-response (negative case) | `TestHookPreToolUseBlocksProtectedPath` (same file) | exact |
| `pkg/codex/permission_profile.go` (`repositoryReadOnlyCastes`) | config / model | CRUD (profile lookup) | `behavioralRestrictionsForCaste`'s surveyor case, same file lines 92-93 | exact — the internally-consistent pattern to converge on |
| `pkg/codex/worker.go` (artifacts schema ~921-926) | model / schema | request-response (validation) | `handoff` schema (933-960) and `scoutReportClaimSchema()` (963-994), same file | exact — sibling schema, same function family |
| `cmd/build_print_brief.go` | CLI command / visual-output | request-response (read-only inspector) | itself (`printWorkerBriefs`, `renderBriefComposition`) + `cmd/codex_visuals.go` conventions | exact — refine, don't rebuild |
| `cmd/build_print_brief_test.go` (new) | test | request-response, buffer capture | `TestHookPreToolUseBlocksProtectedPath` (`cmd/hook_cmds_test.go:34-64`) | role-match — nearest CLI-buffer-capture test |
| `cmd/colony_prime_context.go` (new charter section) | service / context-assembly | transform (rank + budget) | `review_depth` section (380-429) and `state` section (388-399), same file | exact — template already used 15+ times in this file |
| `cmd/codex_build_finalize.go` (suggest-analyze seam) | service / orchestration | event-driven (end-of-build collection) | itself's sequential-validation structure (201-284) + `cmd/suggest_analyze.go` RunE body (37-100+) as logic source | partial — seam is clear, source logic needs extraction (see note) |
| `cmd/gate.go` (`checkCharterComplianceGate`) | service / gate producer | request-response | `checkAntiPatternGate` (`cmd/gate.go:432-`) | exact — research names this the copy-model |
| `cmd/codex_continue.go` (gate wiring) | controller | request-response | anti-pattern gate wiring block (`cmd/codex_continue.go:2940-2956`) | exact |
| `pkg/colony/colony.go` (`Charter` struct) | model | CRUD (read) | itself — no modification needed, consumed as-is | reference only, no change |
| `.claude/rules/aether-colony.md` / `.aether/rules/aether-colony.md` / `.opencode/OPENCODE.md` (Protected Paths table) | config / docs | — | itself — existing "Protected Paths" table format | exact |
| Staleness warning (new, location at planner's discretion) | utility | transform | `cmd/session_cmds.go:488` (`getGitHEAD`) + `cmd/init_research.go:201` (`rev-list --count`) + `cmd/session_flow_cmds.go:36-73` (`sessionVerifyFresh` threshold pattern) | **no direct analog** — nearest 3 partial building blocks, see note |
| Budget-ceiling invariant test (CONTEXT-08, new) | test | invariant | `TestBuildWorkerBriefIsMostlyTask` (`cmd/codex_build_test.go:3186-3221`) | exact — research explicitly names this the model, floor→ceiling flip |
| `pkg/codex/permission_profile_test.go` (scout-write extension) | test | CRUD (table-driven) | `TestPermissionProfileForCasteUsesOnlyEnforceableReleaseProfiles` (same file, 12-37) | exact — the scout row (line 17) is the one that must flip |

---

## Pattern Assignments

### `cmd/codex_build.go` — manifest-level capsule carriage (CONTEXT-02/03)

**Analog:** the hosted/subprocess path already does exactly this at `cmd/codex_build.go:1400-1439`. Copy this shape to the manifest-build call site instead of inventing a new one.

**The proven "compute once, share" pattern to mirror** (`cmd/codex_build.go:1400-1439`):
```go
capsule := resolveCodexWorkerContext()          // computed ONCE
cleanupStaleWorkersBeforeDispatch(root)
pheromoneSection := resolvePheromoneSection()   // computed ONCE
for i, dispatch := range dispatches {
    workerDispatch := codex.WorkerDispatch{
        ID:                fmt.Sprintf("phase-%d-dispatch-%d", phase.ID, i+1),
        TaskBrief:         renderCodexBuildWorkerBrief(root, phase, dispatch, startedAt),
        ContextCapsule:    capsule,          // reused, not recomputed
        PheromoneSection:  pheromoneSection, // reused, not recomputed
        ...
    }
}
```

**The gap to close** — `attachBuildDispatchContext` (`cmd/codex_build.go:2702-2715`) currently loops per-dispatch and calls `composeBuildManifestBrief` per dispatch, which never touches `resolveCodexWorkerContext()`:
```go
func attachBuildDispatchContext(root string, phase colony.Phase, dispatches []codexBuildDispatch, startedAt time.Time) {
    for i := range dispatches {
        dispatches[i].PermissionProfile = codex.PermissionProfileForCaste(dispatches[i].Caste)
        assignment := resolveWorkerSkillAssignmentForWorkflow("build", dispatches[i].Caste, dispatches[i].Task)
        dispatches[i].SkillSection = assignment.Section
        ...
        dispatches[i].HandoffSection = renderWorkerHandoffSection("build", phase.ID, dispatches[i].Name)
        // Brief must be composed after HandoffSection is set — it embeds it.
        dispatches[i].Brief = composeBuildManifestBrief(root, phase, dispatches[i], startedAt)
    }
}
```

**Struct fields to add** — `codexBuildManifest` (`cmd/codex_build.go:62-105`) has no top-level context field today. Add alongside the existing `WorkerBriefs []string` field (line 87):
```go
type codexBuildManifest struct {
    ...
    WorkerBriefs []string              `json:"worker_briefs"`
    Dispatches   []codexBuildDispatch  `json:"dispatches"`
    // new:
    ContextCapsule  string `json:"context_capsule,omitempty"`
    CharterSection  string `json:"charter_section,omitempty"`
    ...
}
```

**Why NOT `codexBuildDispatch.Brief`:** `codexBuildDispatch` (`cmd/codex_build.go:22-52`) already documents its own convention at the `Brief` field (lines 38-44) — "Wrappers must inject this verbatim, never reconstruct it." `composeBuildManifestBrief`'s own doc comment (`cmd/codex_build.go:2717-2726`) explains it deliberately embeds only pheromones + handoffs (small, genuinely per-worker) and explains why the Go subprocess path does NOT reuse this composer (it delivers `PheromoneSection`/`HandoffSection` separately). Adding the capsule here would be the exact per-dispatch duplication CONTEXT-03 exists to prevent (Pitfall 1 in RESEARCH.md).

**Where to call it:** compute `resolveCodexWorkerContext()` once at the manifest-construction call site (`runCodexBuildPlanOnlyWithOptions`, starts `cmd/codex_build.go:161`, manifest built around line 1608 per RESEARCH.md), same as Path A does at line 1400 — not inside `attachBuildDispatchContext`'s per-dispatch loop.

---

### `cmd/hook_cmds.go` — sanctioned scratch-dir carve-out (D-04)

**Analog:** itself. `protectedHookWriteReason` (`cmd/hook_cmds.go:217-239`) is a flat `switch`/`strings.Contains` blocker with zero carve-outs today:
```go
func protectedHookWriteReason(target, cwd string) string {
    normalized := normalizeHookPath(target, cwd)
    if normalized == "" {
        return ""
    }
    slash := filepath.ToSlash(normalized)
    base := filepath.Base(slash)
    switch {
    case strings.Contains(slash, "/.aether/data/"):
        return "Protected colony state path. Update `.aether/data/*` through the `aether` CLI, not direct edits."
    case strings.Contains(slash, "/.aether/dreams/"):
        return "Protected dream journal path. Do not edit `.aether/dreams/` from a worker."
    case strings.HasPrefix(base, ".env"):
        return "Protected environment file. Do not edit `.env*` through a hook-triggered write."
    case strings.HasSuffix(slash, "/.codex/config.toml"):
        return "Protected Codex config path. Do not edit `.codex/config.toml` from a worker."
    case strings.Contains(slash, "/.github/workflows/"):
        return "Protected CI path. Workflow files require explicit user direction."
    default:
        return ""
    }
}
```

**Fix shape (per RESEARCH.md's Security Domain guidance):** add an explicit allowlist check for exact sanctioned subpaths BEFORE the `/.aether/data/` blanket-block case, not a broadened substring match:
```go
var sanctionedDataWritePrefixes = []string{
    "/.aether/data/planning/",
    "/.aether/data/worker-debug/",
    // + peers the plan identifies
}
...
case containsAny(slash, sanctionedDataWritePrefixes):
    return "" // allowed
case strings.Contains(slash, "/.aether/data/"):
    return "Protected colony state path. ..." // unchanged — still blocks COLONY_STATE.json etc.
```
Exact subpaths, not a widened prefix — `TestHookPreToolUseBlocksProtectedPath` must keep passing unmodified as the negative case (RESEARCH.md Known Threat Patterns table, Tampering row).

---

### `cmd/hook_cmds_test.go` — companion carve-out test

**Analog:** `TestHookPreToolUseBlocksProtectedPath` (`cmd/hook_cmds_test.go:34-64`), full text — copy its buffer-capture scaffold for the new positive-carve-out test:
```go
func TestHookPreToolUseBlocksProtectedPath(t *testing.T) {
    saveGlobalsCmd(t)
    resetRootCmd(t)

    var buf bytes.Buffer
    stdout = &buf
    var errBuf bytes.Buffer
    stderr = &errBuf

    _, tmpDir := newTestStoreCmd(t)
    defer os.RemoveAll(tmpDir)

    protected := filepath.Join(tmpDir, ".aether", "data", "COLONY_STATE.json")
    setHookStdin(t, `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"`+protected+`"}}`)

    rootCmd.SetArgs([]string{"hook-pre-tool-use"})
    if err := rootCmd.Execute(); err != nil {
        t.Fatalf("hook-pre-tool-use returned error: %v", err)
    }

    var result map[string]interface{}
    if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
        t.Fatalf("unmarshal hook output: %v", err)
    }
    if result["decision"] != "block" {
        t.Fatalf("decision = %v, want block", result["decision"])
    }
    ...
}
```
New companion test: same scaffold, target `.aether/data/planning/phase-plan.json`, assert `result["decision"] != "block"` (i.e. absent/allow). `setHookStdin` helper (`cmd/hook_cmds_test.go:16-32`) is already shared infra — reuse it.

---

### `pkg/codex/permission_profile.go` — scout permission fix (D-05, Pitfall 4)

**Analog:** the internally-consistent surveyor pattern, same file:
```go
// pkg/codex/permission_profile.go:53-56 — the bug
var repositoryReadOnlyCastes = map[string]struct{}{
    "includer": {},
    "scout":    {},
}

// pkg/codex/permission_profile.go:86-100 — the pattern to converge scout onto
func behavioralRestrictionsForCaste(caste string) []string {
    switch normalizePermissionCaste(caste) {
    case "archaeologist", "auditor", "gatekeeper", "measurer", "tracker", "watcher":
        return []string{"do not modify project source or tests; persist only approved review evidence"}
    case "probe":
        return []string{"write test files only; do not modify project source"}
    case "surveyor_disciplines", "surveyor_nest", "surveyor_pathogens", "surveyor_provisions":
        return []string{"write survey artifacts under .aether/data/survey only"}
    ...
    }
}
```
**Fix:** remove `"scout"` from `repositoryReadOnlyCastes` and add a `"scout"` case to `behavioralRestrictionsForCaste` returning something like `[]string{"write phase research artifacts under .aether/data/phase-research only"}` — this makes scout fall through to the default `PermissionWorkspaceWrite` profile (line 74-84) exactly like surveyors do, instead of the hard `PermissionRepositoryReadOnly` profile that contradicts what `cmd/phase_research.go:124`'s brief instructs it to do.

**The exact contradiction being fixed** (already verified in RESEARCH.md, cited here for the planner's direct reference): `cmd/phase_research.go:70-77` dispatches a `"scout"` caste, and `cmd/phase_research.go:124`'s `renderPhaseResearchBrief` tells that same scout `Write your findings to .aether/data/phase-research/phase-%d-research.md`.

**Existing test that must be updated, not just extended** — `TestPermissionProfileForCasteUsesOnlyEnforceableReleaseProfiles` (`pkg/codex/permission_profile_test.go:12-37`) currently asserts scout stays read-only:
```go
tests := []struct {
    caste string
    want  PermissionProfileName
}{
    {"scout", PermissionRepositoryReadOnly},   // line 17 — must flip to PermissionWorkspaceWrite
    {"aether-includer", PermissionRepositoryReadOnly},
    {"builder", PermissionWorkspaceWrite},
    ...
}
```
This is a table-driven test — flip the `want` value for the `"scout"` row rather than writing a wholly new test.

---

### `pkg/codex/worker.go` — artifacts schema fix (D-05)

**Analog:** two sibling schemas in the same function family, both showing the named-typed-fields shape to copy:

**The bug** (`pkg/codex/worker.go:921-926`):
```go
"artifacts": map[string]interface{}{
    "type":                 "object",
    "additionalProperties": false,
    "properties":           map[string]interface{}{},
    "required":             []string{},
},
```

**Pattern to copy — `handoff` schema, immediately adjacent** (`pkg/codex/worker.go:933-958`):
```go
"handoff": map[string]interface{}{
    "type":                 "object",
    "additionalProperties": false,
    "required": []string{
        "changed_files", "commands_run", "verification_status",
        "known_failures", "open_decisions", "assumptions",
        "next_worker_instructions", "do_not_repeat", "freshness",
    },
    "properties": map[string]interface{}{
        "changed_files":       stringArray,
        "commands_run":        stringArray,
        "verification_status": map[string]interface{}{"type": "string", "enum": []string{"pass", "fail", "partial", "not_run", "unknown"}},
        ...
    },
},
```

**Pattern to copy — `scoutReportClaimSchema()`, full nested example** (`pkg/codex/worker.go:963-994`):
```go
func scoutReportClaimSchema() map[string]interface{} {
    stringArray := map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}
    findingSchema := map[string]interface{}{
        "type": "object", "additionalProperties": false,
        "required": []string{"area", "discovery", "source"},
        "properties": map[string]interface{}{
            "area": map[string]interface{}{"type": "string"},
            ...
        },
    }
    return map[string]interface{}{
        "type": "object", "additionalProperties": false,
        "required": []string{"findings", "gaps", "confidence", "study_files"},
        "properties": map[string]interface{}{
            "findings":    map[string]interface{}{"type": "array", "items": findingSchema},
            "gaps":        stringArray,
            "confidence":  map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
            "study_files": stringArray,
        },
    }
}
```
**Fix:** replace the empty `properties`/`required` on `artifacts` with named typed fields (e.g. `research_file: {"type": "string"}`), keeping `additionalProperties: false` — per RESEARCH.md's Security Domain table, do NOT flip `additionalProperties` to `true`.

---

### `cmd/build_print_brief.go` — checklist-default / `--full` refinement (D-06)

**Analog:** itself. `printWorkerBriefs` (`cmd/build_print_brief.go:27-96`) and `renderBriefComposition` (`cmd/build_print_brief.go:107-141`) already exist, already work, and already have the section-splitting machinery this needs (`splitBriefSections`, `briefOwnedSections`, lines 143-202). This is a UX refinement, not new plumbing — see RESEARCH.md Pitfall 2.

**Current behavior (unconditional full dump + composition table), to be gated behind `--full`:**
```go
for _, dispatch := range dispatches {
    ...
    single := []codexBuildDispatch{dispatch}
    attachBuildDispatchContext(root, phase, single, startedAt)
    brief := single[0].Brief

    out.WriteString(strings.Repeat("━", 72))
    out.WriteString(fmt.Sprintf("\n%s  %s  (%s)\n", casteEmoji(dispatch.Caste), dispatch.Name, dispatch.Caste))
    out.WriteString(strings.Repeat("━", 72))
    out.WriteString("\n\n")
    out.WriteString(brief)              // <- gate this block behind --full
    out.WriteString("\n")
    out.WriteString(renderBriefComposition(brief))  // <- this becomes the default-mode content
    out.WriteString("\n")
}
```

**What D-06 needs added:** a default checklist mode — section name, present/absent, size, total vs the real budget constants (colony-prime 8000/4000 from `cmd/colony_prime_context.go:335-338`, not the removed "playbook 7K" line RESEARCH.md flags as stale) — using `splitBriefSections` output, plus manifest-level `ContextCapsule`/`CharterSection` once those exist. `renderBriefComposition` already computes per-section `Chars` and `%` of the brief's own total (lines 124-136) — extend it (or add a sibling renderer) to also show `chars / budget` for the checklist view, and add a `--full` bool flag (mirror the existing flag registration idiom at `cmd/codex_workflow_cmds.go:1165-1167`):
```go
buildCmd.Flags().Bool("print-brief", false, "...")
buildCmd.Flags().String("worker", "", "With --print-brief, print only the named worker's prompt")
// add:
buildCmd.Flags().Bool("full", false, "With --print-brief, print the raw assembled prompt instead of the checklist")
```

**Visual conventions to reuse from `cmd/codex_visuals.go`:**
```go
// AETHER_OUTPUT_MODE gate (cmd/codex_visuals.go:269-283)
func shouldRenderVisualOutput(w io.Writer) bool {
    mode := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_OUTPUT_MODE")))
    switch mode {
    case "json":
        return false
    case "visual", "human", "pretty":
        return true
    }
    if os.Getenv("AETHER_FORCE_VISUAL") == "1" {
        return true
    }
    return isTerminalWriter(w)
}

// Stage marker convention (cmd/codex_visuals.go:365-371)
func renderStageMarker(title string) string {
    title = strings.TrimSpace(title)
    if title == "" {
        return ""
    }
    return "── " + title + " ──\n"
}
```
`printWorkerBriefs` currently writes unconditionally to `stdout` via `fmt.Fprint` (line 94) with no `AETHER_OUTPUT_MODE` branching — since this is an inspection command (not JSON API output), that's likely fine to keep as-is, but if the checklist view should respect `--json` for scripting, gate it the same way `outputWorkflow` does (`cmd/codex_visuals.go:305-314`).

---

### `cmd/build_print_brief_test.go` (new) — buffer-capture CLI test

**No existing test file for this command** — confirmed zero `TestPrintBrief*`/`TestPrintWorkerBriefs*` matches in `cmd/*_test.go` (RESEARCH.md Wave 0 Gaps). Nearest analog for the buffer-capture + `rootCmd.Execute()` scaffold is `TestHookPreToolUseBlocksProtectedPath` (`cmd/hook_cmds_test.go:34-64`, full text quoted above) — same `saveGlobalsCmd(t)` / `resetRootCmd(t)` / `stdout = &buf` / `rootCmd.SetArgs([]string{...})` / `rootCmd.Execute()` shape applies directly; swap the hook-stdin setup for a colony state fixture with a phase and dispatches (see `printWorkerBriefs`'s own preconditions: `state.Plan.Phases` must be non-empty, `plannedBuildDispatchesForSelectionWithState` must return dispatches).

---

### `cmd/colony_prime_context.go` — charter section (CONTEXT-06)

**Analog:** the `review_depth` section (`cmd/colony_prime_context.go:401-429`) is the simplest full example of the exact pattern to copy — free-text content built into a `strings.Builder`, wrapped in a `colonyPrimeSection{}`, appended to `sections`:
```go
var rdSB strings.Builder
writeSectionHeader(&rdSB, "review_depth", "## Review Depth\n\n")
rdSB.WriteString(depthText)
rdSB.WriteString("\n")
sections = append(sections, colonyPrimeSection{
    name:           "review_depth",
    title:          "Review Depth",
    source:         statePath,
    content:        rdSB.String(),
    priority:       6,
    freshnessScore: 1.0,
})
```
The `state` section (`cmd/colony_prime_context.go:388-399`) shows the fuller version with `protectedSectionPolicy`:
```go
stateProtected, statePreserveReason := protectedSectionPolicy("state")
sections = append(sections, colonyPrimeSection{
    name:              "state",
    title:             "Colony State",
    source:            statePath,
    content:           stateSection.String(),
    priority:          5,
    freshnessScore:    1.0,
    confirmationScore: 1.0,
    relevanceScore:    sectionRelevanceScore("state"),
    protected:         stateProtected,
    preserveReason:    statePreserveReason,
})
```

**The `colonyPrimeSection` struct itself** (`cmd/colony_prime_context.go:57-72`) — every field a new charter section can populate:
```go
type colonyPrimeSection struct {
    name              string
    title             string
    source            string
    content           string
    priority          int
    baseTrustClass    colony.PromptTrustClass
    trustClass        colony.PromptTrustClass
    action            colony.PromptIntegrityAction
    findings          []colony.PromptIntegrityFinding
    freshnessScore    float64
    confirmationScore float64
    relevanceScore    float64
    protected         bool
    preserveReason    string
}
```

**Why this is sufficient — no new pipeline needed:** every section, once appended to `sections`, automatically runs through `colony.AssessPromptSource(sec.source, sec.content)` (`cmd/colony_prime_context.go:851`) and `colony.RankContextCandidates(allowedCandidates, budget)` (`cmd/colony_prime_context.go:864`) — charter inherits prompt-integrity assessment and budget ranking for free by becoming one more `colonyPrimeSection`, per RESEARCH.md's Don't Hand-Roll table and Security Domain note. Source the content from `state.Charter.Governance` (`pkg/colony/colony.go:274`), and optionally `.Constraints` (line 278).

---

### `cmd/codex_build_finalize.go` — suggest-analyze collection seam (CONTEXT-05, D-11)

**Analog:** `runCodexBuildFinalize`'s own existing step structure (`cmd/codex_build_finalize.go:201-284`) — a long sequential chain of validate-then-proceed steps, each returning early on error:
```go
func runCodexBuildFinalize(root string, phaseNum int, completion codexExternalBuildCompletion, skipVerify bool) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
    if store == nil { return nil, ..., fmt.Errorf("no store initialized") }
    manifest := completion.activeManifest()
    ...
    if err := validateFinalizerManifestRoot(...); err != nil { return nil, ..., err }
    ...
    state, err := loadActiveColonyState()
    ...
    if err := runPreBuildGates(store.BasePath(), phaseNum); err != nil { return nil, ..., err }
    ...
    if err := validateExternalWorkerResultClaimPaths(root, completion.workerResults()); err != nil { return nil, ..., err }
    // <- natural seam for a non-blocking suggest-analyze collection call, after
    //    the build's own validation chain but before/alongside final result assembly
}
```

**Important gap the planner must close, not just wire around:** `suggest_analyze.go`'s entire logic lives inline inside `suggestAnalyzeCmd`'s `RunE` closure (`cmd/suggest_analyze.go:33-100+`) — it is **not** currently an extractable function. To call it from `runCodexBuildFinalize` per D-11 (collect during build, present once at end), the RunE body needs to be factored into a standalone function (e.g. `runSuggestAnalyze(target string, dryRun bool) (map[string]interface{}, error)`) that both the CLI command and the finalize seam call — mirroring how `printWorkerBriefs` is already a standalone function that `codexBuildCmd`'s RunE merely invokes (`cmd/codex_workflow_cmds.go:127-135`). This is the one place in this phase that is genuinely "extend a function," not "call an existing one" — flag it as such in planning, don't understate the diff size.

**End-of-build presentation:** `filterActiveSuggestions` (`cmd/suggest_approve.go:472-`) and `colony.PendingSuggestion` (`pkg/colony/colony.go:286-294`) are the existing tick-to-approve machinery — no new UI needed, just a call site that persists suggestions during finalize so `suggest-approve` has something to show "once at the end" per D-11.

---

### `cmd/gate.go` — charter compliance gate (CONTEXT-06, D-09)

**Analog:** `checkAntiPatternGate` (`cmd/gate.go:432-`) — research explicitly names this the copy-model, and it is the second consecutive phase needing this exact producer shape:
```go
func checkAntiPatternGate(files []string) (gateCheck, gateCheck) {
    findingsCheck := gateCheck{Name: "anti_pattern"}
    executedCheck := gateCheck{Name: "anti_pattern_executed"}
    if store == nil {
        executedCheck.Passed = false
        executedCheck.Detail = "antipattern scan could not execute: no store initialized, colony root unresolvable"
        executedCheck.FixHint = gateRecoveryTemplate("anti_pattern")
        executedCheck.RecoveryOptions = []string{
            "Fix manually and run /ant-continue",
            "Run /ant-unblock for guided recovery",
        }
        findingsCheck.Passed = true
        findingsCheck.Detail = "antipattern scan did not run: no store initialized"
        return findingsCheck, executedCheck
    }
    root := filepath.Join(filepath.Dir(store.BasePath()), "..")
    // ... scan loop, populate findingsCheck/executedCheck.Passed/Detail
}
```
Two-check return shape (`findingsCheck`, `executedCheck`) exists specifically so "the gate never ran" (hard block) is distinguishable from "the gate ran and found nothing" (soft pass) — copy this distinction for `checkCharterComplianceGate(rules []string, scannedFiles []string) (gateCheck, gateCheck)` or equivalent signature, checking only the mechanically-enforceable subset of `Charter.Governance` per D-09 (RESEARCH.md Open Question 2 — verify at planning time whether `Governance` is structured enough, via `governanceDetectors`, `cmd/init_research.go:117` and `generateCharter`, `cmd/codex_workflow_cmds.go:1620`, before deciding the mechanical-vs-prose boundary).

---

### `cmd/codex_continue.go` — charter gate wiring (CONTEXT-06, D-09)

**Analog:** the anti-pattern gate's exact wiring block (`cmd/codex_continue.go:2940-2956`), inside `runCodexContinueGates` (starts line 2841):
```go
// anti_pattern / anti_pattern_executed gates — the live caller for the
// security gate that RESEARCH.md found had no live caller (T-160-01).
antiPatternCheck, antiPatternExecutedCheck := checkAntiPatternGate(verification.Claims.ScannedFiles)
if shouldSkipGate(priorGateResults, "anti_pattern") {
    checks = append(checks, gateCheck{Name: "anti_pattern", Passed: true, Detail: "skipped: previously passed"})
} else {
    if !antiPatternCheck.Passed {
        blockers = append(blockers, antiPatternCheck.Detail)
    }
    checks = append(checks, antiPatternCheck)
}
if !antiPatternExecutedCheck.Passed {
    blockers = append(blockers, antiPatternExecutedCheck.Detail)
}
checks = append(checks, antiPatternExecutedCheck)
```
Copy verbatim shape for `checkCharterComplianceGate`, substituting the appropriate skip-key string and inputs. `shouldSkipGate(priorGateResults, "...")` and `blockers = append(...)` are the two integration points every gate in this function uses — this is the exact wiring surface, not a new mechanism.

---

## Shared Patterns

### Compute-once-share-across-dispatches
**Source:** `cmd/codex_build.go:1400-1439` (Path A, `executeCodexBuildDispatches`)
**Apply to:** the manifest-level `ContextCapsule`/`CharterSection` fix (CONTEXT-02/03) — this is the single most load-bearing pattern in this phase; every other dispatch-context field (`capsule`, `pheromoneSection`) in this codebase is already computed once and shared, never inside a per-dispatch loop.

### `colonyPrimeSection` template
**Source:** `cmd/colony_prime_context.go:57-72` (struct), `:388-429` (two full worked examples: `state`, `review_depth`)
**Apply to:** any new colony-prime content (charter). Guarantees automatic `colony.AssessPromptSource` integrity assessment (`:851`) and `colony.RankContextCandidates` budget ranking (`:864`) with zero additional plumbing.

### Two-check gate producer (findings vs. executed)
**Source:** `cmd/gate.go:432-` (`checkAntiPatternGate`), wired at `cmd/codex_continue.go:2940-2956`
**Apply to:** `checkCharterComplianceGate` (CONTEXT-06/D-09) — distinguishes "rule violated" from "gate could not run," per the Definition of Done's demand that an inspection/gate command must not silently report success when it never executed.

### Named-typed-fields schema (not `additionalProperties: true`)
**Source:** `pkg/codex/worker.go:933-960` (`handoff`), `:963-994` (`scoutReportClaimSchema`)
**Apply to:** the `artifacts` schema fix (D-05) — keep `additionalProperties: false`, add real named fields.

### AETHER_OUTPUT_MODE / stage-marker visual conventions
**Source:** `cmd/codex_visuals.go:269-283` (`shouldRenderVisualOutput`), `:365-371` (`renderStageMarker`)
**Apply to:** `--print-brief`'s checklist rendering (D-06), if the checklist view should participate in the same JSON/visual mode switching other commands honor.

---

## No Analog Found

| File / Change | Role | Data Flow | Reason |
|---|---|---|---|
| Staleness warning ("map is N commits stale") | utility | transform | No existing function computes "commits since X" for display. Closest building blocks: `getGitHEAD()` (`cmd/session_cmds.go:488-496`, returns current HEAD via `git rev-parse HEAD`), `analyzeGitHistory`'s commit-count call (`cmd/init_research.go:201`, `git rev-list --count HEAD` — counts ALL commits, not commits-since-a-baseline, so it is not directly reusable as-is), and `sessionVerifyFresh`'s age/mismatch-threshold pattern (`cmd/session_flow_cmds.go:36-73`, compares a stored baseline SHA to current HEAD and flags staleness on mismatch — the closest *shape* match, but binary fresh/stale, not a commit-count). The new staleness check needs `git rev-list --count <baseline-commit>..HEAD` (a pattern not found anywhere in this codebase today) plus a stored baseline commit from colonize's own manifest generation (`cmd/codex_colonize_finalize.go:226-237` already validates `generated_at` freshness by timestamp, not commit count — a second partial analog, different axis). Build this as new code combining these three fragments; do not force-fit `sessionVerifyFresh`'s binary model onto a "N commits" display requirement. |
| `suggest_analyze.go` logic extraction into a callable function | refactor target, not new file | event-driven | Not "no analog" in the sense of missing code — the logic exists and is correct — but it is currently unextractable (entirely inline in a cobra `RunE` closure, `cmd/suggest_analyze.go:33-100+`). The planner should budget this as a small refactor (factor into `runSuggestAnalyze(...)`, have the RunE call it), following the existing precedent of `printWorkerBriefs` being a standalone function `codexBuildCmd`'s RunE merely invokes (`cmd/codex_workflow_cmds.go:127-135`), not as a "call an existing function" task. |

---

## Metadata

**Analog search scope:** `cmd/` (build/continue/gate/hook/print-brief/colony-prime families), `pkg/codex/` (permission profile, worker schema), `pkg/colony/` (Charter, PendingSuggestion structs), `.claude/rules/`, `.aether/rules/`, `.opencode/OPENCODE.md`
**Files scanned:** ~20 targeted reads across `cmd/codex_build.go`, `cmd/hook_cmds.go` (+test), `pkg/codex/permission_profile.go` (+test), `pkg/codex/worker.go`, `cmd/build_print_brief.go`, `cmd/codex_visuals.go`, `cmd/colony_prime_context.go`, `cmd/codex_build_finalize.go`, `cmd/gate.go`, `cmd/codex_continue.go`, `pkg/colony/colony.go`, `cmd/codex_build_test.go`, `cmd/suggest_analyze.go`, `cmd/suggest_approve.go`, `cmd/session_cmds.go`, `cmd/session_flow_cmds.go`, `cmd/init_research.go`, `cmd/codex_workflow_cmds.go`
**Pattern extraction date:** 2026-07-29
**Constraints honored:** playbook loader / `survey-load` left dead (not resurrected as analogs); no `build.md` structural edits proposed (Phase 165 territory — only the manifest-field/wrapper-read-once change RESEARCH.md scopes to this phase); distributed rules files (`.claude/rules/aether-colony.md`, `.opencode/OPENCODE.md`) flagged for hub-publish parity, not edited here.
