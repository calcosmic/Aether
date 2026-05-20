# Architecture Patterns: Grounded Planning + Ceremony Restore

**Project:** Aether v1.22 Grounded Planning + Ceremony Restore
**Domain:** CLI colony framework -- hybrid Go runtime + TypeScript orchestration host
**Researched:** 2026-05-18

## Recommended Architecture

The milestone restores two capabilities that were lost during the shell-to-Go migration:
1. **Evidence-based planning** (survey produces grounded plans, not generic ones)
2. **Playbook-driven ceremony** (ceremony comes from editable markdown, not hardcoded code)

Neither requires architectural changes -- both are restorations of existing patterns that lost their wiring.

### Data Flow: Survey -> Plan Grounding

```
/aether colonize
    |
    v
surveyWorkspace(root)  [cmd/codex_colonize.go]
    |
    +-- Walk repo root with shared ScanFilter [pkg/codegraph/scan_filter.go]
    |       |
    |       +-- Skip noise dirs (.venv, __pycache__, node_modules, etc.)
    |       +-- Skip noise file extensions (.pyc, .d.ts, .min.js, etc.)
    |       +-- Collect: TopLevelDirs, Languages, Frameworks, ConfigFiles
    |       +-- NEW: Extract SourceAnchors (non-dependency, non-generated source files)
    |
    v
codexWorkspaceFacts  [includes new SourceAnchors field]
    |
    v
.aether/data/survey/blueprint.json  [written by surveyor dispatches]
    |
    v
/aether plan
    |
    +-- Colony-prime reads survey context (including SourceAnchors)
    +-- Pending decisions injected as REDIRECT pheromones
    +-- Scout research with anchoring context
    +-- Route-Setter produces plan
    |
    v
codex_plan_finalize  [NEW: grounding gate]
    |
    +-- Scan plan tasks for concrete file references
    +-- If SourceAnchors non-empty AND plan has zero file refs:
    |       emit grounding_warning
    +-- Accept plan with warning or clean plan without
    |
    v
COLONY_STATE.json  [plan written]
```

### Data Flow: Playbook-Driven Ceremony

```
/aether build <phase>  [user command]
    |
    v
YAML command definition (.aether/commands/build.yaml)
    |  (defines command name, flags, runtime command mapping)
    v
TS host lifecycle.ts
    |
    +-- Read build-full.md from .aether/docs/command-playbooks/
    |   (or split playbooks: build-prep, build-context, build-wave, etc.)
    |
    +-- Parse step headers (### Step N: Title)
    +-- For each step, follow the instructions
    |
    +-- When visual output needed:
    |       Call GoCeremonyAdapter.renderSpawnPlan()
    |       Call GoCeremonyAdapter.renderWaveStart()
    |       Call GoCeremonyAdapter.renderWorkerComplete()
    |       Call GoCeremonyAdapter.renderCloseout()
    |       |
    |       v
    |   Go runtime (cmd/ceremony_cmd.go)
    |       |
    |       v
    |   ANSI-colored ceremony output (terminal)
    |
    +-- When Go state mutation needed:
    |       Call Go bridge (aether build 1 --plan-only, aether spawn-log, etc.)
    |
    v
Build complete
```

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `pkg/codegraph/scan_filter.go` (new) | Shared directory/file exclusion logic | `codegraph.Scan()`, `surveyWorkspace()` |
| `cmd/codex_colonize.go` | Survey workspace facts extraction | `scan_filter.go`, colony state store |
| `cmd/discuss.go` | Decision capture and binding | Pheromone system (REDIRECT), pending decisions store |
| `cmd/codex_plan_finalize.go` | Plan validation including grounding gate | Plan output, colony context |
| `cmd/build_playbook_context.go` | Playbook file resolution and snippet injection | Colony-prime (worker context assembly) |
| `.aether/ts-host/src/lifecycle.ts` | Read playbooks and orchestrate build/plan/continue | Go bridge, ceremony adapter, platform dispatcher |
| `.aether/ts-host/src/ceremony-adapter.ts` | Route visual ceremony to Go runtime | Go CLI (ceremony commands) |
| `cmd/ceremony_cmd.go` | Render ceremony visuals (ANSI) | Terminal stdout |

## Patterns to Follow

### Pattern 1: Shared Filter Module
**What:** A single Go package (`pkg/codegraph/scan_filter.go`) provides `ShouldSkipDir` and `ShouldSkipFile` functions used by both codegraph and colonize.
**When:** Any code that walks a repository directory tree.
**Why:** Prevents the current divergence where codegraph skips 13 dirs and colonize skips 10 dirs with no overlap enforcement.
**Example:**
```go
// pkg/codegraph/scan_filter.go
package codegraph

func ShouldSkipDir(name string) bool {
    return noiseDirs[name]
}

func ShouldSkipFile(name string) bool {
    ext := filepath.Ext(name)
    return noiseFileExts[ext]
}
```

### Pattern 2: Soft Validation Gate
**What:** A validation step that emits a warning but does not block progress.
**When:** When validation is helpful but not always applicable (some valid cases violate the rule).
**Why:** Research phases and architecture phases legitimately produce plans without file references. Hard rejection would break these.
**Example:**
```go
func validatePlanGrounding(plan Plan, anchors []string) []string {
    var warnings []string
    if len(anchors) > 0 && !planReferencesFiles(plan) {
        warnings = append(warnings, "Plan contains no concrete file references despite "+
            fmt.Sprintf("%d source anchors being available. Consider specifying target files.", len(anchors)))
    }
    return warnings
}
```

### Pattern 3: Playbook as Execution Script
**What:** The TS host reads markdown playbooks and follows their step sequence, rather than hardcoding orchestration steps.
**When:** Any workflow that has a playbook definition (build, continue, plan).
**Why:** Playbooks are user-editable markdown; hardcoded steps are only developer-editable.
**Example:**
```typescript
// lifecycle.ts (conceptual)
async function executePlaybook(playbookPath: string, context: BuildContext): Promise<void> {
    const content = readFileSync(playbookPath, "utf-8");
    const steps = parseSteps(content); // Extract ### Step N: sections
    for (const step of steps) {
        await executeStep(step, context);
    }
}
```

### Pattern 4: Pheromone-Based Constraint Injection
**What:** Hard constraints from discuss decisions are injected into the colony via REDIRECT pheromones.
**When:** When a discuss question is resolved with `hard_constraint: true`.
**Why:** The pheromone system already handles priority ordering, deduplication, TTL, and injection into worker context. No new mechanism needed.
**Example:**
```go
// In resolveDiscussQuestion, after resolution:
if question.HardConstraint {
    pheromoneWrite("REDIRECT", resolution, "auto:discuss:"+question.ID, 0.8)
}
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Divergent Skip Lists
**What:** Two subsystems (codegraph and colonize) maintain separate directory exclusion lists.
**Why bad:** When you fix a bug in one list (e.g., adding `.venv`), the other still has the bug. Users see inconsistent behavior depending on which code path runs.
**Instead:** Single shared filter module in `pkg/codegraph/scan_filter.go`.

### Anti-Pattern 2: Triple Conductor
**What:** YAML, TS host, and Go runtime all try to own ceremony orchestration.
**Why bad:** Changes in one conductor break the others. Users cannot predict which conductor is active.
**Instead:** Playbooks = ceremony script, TS host = playbook reader, Go = visual renderer. YAML = packaging metadata only.

### Anti-Pattern 3: Hard Validation Without Exception Paths
**What:** A grounding gate that rejects plans without file references.
**Why bad:** Research, architecture, and infrastructure phases legitimately produce file-reference-free plans. Hard rejection breaks the workflow for these phase types.
**Instead:** Soft warning; user decides.

### Anti-Pattern 4: Config Over Convention
**What:** Adding a configuration file for noise exclusion patterns when a hardcoded list suffices.
**Why bad:** Configuration files add maintenance burden, version drift, and edge cases (missing config, stale config). The set of noise directories is well-known and stable across language ecosystems.
**Instead:** Hardcoded 25-entry default + pheromone REDIRECT for unusual exclusions.

## Scalability Considerations

| Concern | At 100 files | At 10K files | At 1M files |
|---------|--------------|--------------|-------------|
| Noise filter performance | Negligible (map lookup O(1)) | Negligible | Negligible (still map lookup) |
| Source anchor extraction | Instant | 50ms (single walk) | 5s (but capped at 50 anchors) |
| Grounding gate validation | <1ms | <1ms (regex scan of plan text) | <1ms |
| Playbook file reading | <1ms | <1ms (read once, cache) | <1ms |
| Pheromone write for discuss binding | <1ms | <1ms | <1ms |

None of these changes introduce scaling concerns. The most expensive operation (source anchor extraction) is a single directory walk, which already happens during survey. The cap of 50 anchors prevents unbounded output.

## Sources

- Existing codegraph architecture: `pkg/codegraph/codegraph.go` design doc header ("file-level imports only, no AST, no external dependencies")
- Existing playbook system: `cmd/build_playbook_context.go` (playbook resolution and snippet injection)
- Existing pheromone system: `cmd/discuss.go` (HardConstraint field), pheromone-write command
- Existing ceremony adapter: `.aether/ts-host/src/ceremony-adapter.ts` (GoCeremonyAdapter)
- Boundary contract: `.aether/ts-host/src/boundary-reference.ts` (GO_OWNED_PATHS)
