# Phase 190: Lean, Non-Duplicated Delivery - Pattern Map

**Mapped:** 2026-08-20
**Files analyzed:** 4 criteria, ~20 primary files touched or read, 1 net-new function
(`writeBuildWorkerBriefFiles`), 1 net-new function (`duplicatedBriefSections`), 1 file deleted
(`hive-injector.ts`) + its dedicated test file, 0 net-new source files otherwise.
**Analogs found:** 4 / 4 (every change has a same-repo precedent to copy; none is invented from
nothing).

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/codex_build.go` (modified — `writeBuildWorkerBriefFiles` extraction, plan-only wiring) | controller (worker artifact composition) | request-response | its own pre-existing `writeCodexBuildArtifacts` brief-writing loop (lines 2057-2080) | exact (same file, same shape, extracted not invented) |
| `cmd/codex_build_test.go` (modified — invert one assertion, add invariant tests) | test (proportion/invariant + regression) | batch | `TestDispatchEntryCarriesBriefPath`'s own existing shape (build a colony, run the CLI, parse JSON, walk dispatches) | exact |
| `cmd/build_print_brief.go` (modified — `duplicatedBriefSections`, wired into `printWorkerBriefs`) | controller (inspection/assertion) | batch | its own `briefOwnedSections`/`splitBriefSections` scaffolding, extended from attribution to counting | exact (same file, same registry, new pass) |
| `cmd/build_print_brief_test.go` (modified — new duplication tests) | test (invariant) | batch | its own existing `printBriefFixture`/`runPrintBriefCmd` fixture helpers | exact |
| `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, `.opencode/commands/ant/build.md` (modified, byte-identical) | presentation (wrapper instruction prose) | — | 189's own D-09/D-10 wrapper-triplet precedent (write once, copy verbatim to all three) | exact |
| `.aether/commands/build.yaml`, `cmd/command_guide.go`, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` (modified — companion prose) | presentation (source-of-truth YAML, Codex-lane guide, Codex skill) | — | `build.md`'s OWN "Cross-Platform Drift Guard" section, naming these three as required companions to any manifest-handling prose change | exact (the file's own stated rule, followed not invented) |
| `cmd/internal_worker_adapter.go` (modified — remove `HiveSection` field, delete `joinInternalWorkerSections`) | controller (Go/TS wire boundary) | request-response | its own pre-existing single-argument-after-removal shape, matching 189's D-02 "delete the now-trivial helper" precedent | exact |
| `cmd/internal_worker_adapter_test.go` (modified — fixture update) | test (fixture) | batch | its own existing fixture literal | exact |
| `.aether/ts-host/src/host.ts` (modified — remove 4 call sites, `prepareHiveSection`, `emitHiveSummary`, interface fields) | controller (TS orchestration, dispatch prep) | request-response | its own repeated two-line attachment pattern, removed uniformly at all four sites | exact |
| `.aether/ts-host/src/worker-dispatch.ts` (modified — remove request field + assignment) | controller (Go/TS wire construction) | request-response | its own existing conditional-field-copy pattern for the OTHER fields on the same request (`context_capsule`, `pheromone_section`) | exact |
| `.aether/ts-host/src/types.ts` (modified — remove interface field) | data (shared dispatch shape) | — | its own existing interface, one field removed | exact |
| `.aether/ts-host/src/hive-injector.ts` (deleted) | data/controller (redundant hive computation) | — | n/a — deletion, not replacement; `resolveCodexWorkerContext()` (Go) is the surviving channel | exact (removal of the duplicate, not creation of a new single source — the single source already existed) |
| `.aether/ts-host/test/host-integration.test.ts`, `.aether/ts-host/test/worker-dispatch.test.ts` (modified — rewrite hive-specific assertions) | test (regression lock) | batch | their own existing `it(...)` block shapes, content inverted from "attached" to "absent" | exact |
| `.aether/ts-host/test/hive-injector.test.ts` (deleted) | test | — | n/a — deleted alongside its subject module | exact |
| `.aether/ts-host/dist/**` (rebuilt, tracked in git) | compiled output | — | the project's own established `dist/`-is-tracked convention (`!.aether/ts-host/dist/**` in `.gitignore`) | exact |

## Pattern Assignments

### The direct-path-only artifact writer (the gap, and the extraction that closes it)

**Source of the gap** (`cmd/codex_build.go:2052-2100`, `writeCodexBuildArtifacts` — called only from
the two non-`--plan-only` dispatch functions, `cmd/codex_build.go:606,684`):
```go
func writeCodexBuildArtifacts(root string, state colony.ColonyState, phase colony.Phase, buildDirRel, checkpointRel, claimsRel string, dispatches []codexBuildDispatch, startedAt time.Time, dispatchMode string, selectedTaskIDs []string, reviewDepth colony.VerificationDepth, policy codexQueenExecutionPolicy) ([]string, []codexBuildDispatch, error) {
	briefPaths := make([]string, 0, len(dispatches))
	briefOutputs := map[string]string{}
	finalOutputs := map[string][]string{}

	for i := range dispatches {
		briefRel := filepath.ToSlash(filepath.Join(buildDirRel, "worker-briefs", fmt.Sprintf("%s.md", dispatches[i].Name)))
		content := dispatches[i].Brief
		if strings.TrimSpace(content) == "" {
			content = composeBuildManifestBrief(root, phase, dispatches[i], startedAt)
			dispatches[i].Brief = content
		}
		if err := store.AtomicWrite(briefRel, []byte(content)); err != nil {
			return nil, nil, fmt.Errorf("failed to write worker brief for %s: %w", dispatches[i].Name, err)
		}
		displayPath := displayDataPath(briefRel)
		briefPaths = append(briefPaths, displayPath)
		briefOutputs[dispatches[i].Name] = displayPath
		dispatches[i].BriefPath = displayPath
	}
	sort.Strings(briefPaths)
	...
```

**The function that never calls it** (`cmd/codex_build.go:207-381`, `runCodexBuildPlanOnlyWithOptions`
— the function behind the ONLY command the wrapper is allowed to run, `build.md:108,348`):
```go
attachBuildDispatchContext(root, phase, dispatches, generatedAt)   // sets .Brief only, line 281
...
manifest := buildCodexBuildManifest(root, state, phase, "", "", dispatches, generatedAt, "plan-only", selectedTaskIDs, nil, true, reviewDepth)   // nil workerBriefs, line 293
```

**The fix:** extract the loop into `writeBuildWorkerBriefFiles(root, phase, buildDirRel, dispatches,
startedAt, clearInlineBrief bool)`. `writeCodexBuildArtifacts` calls it with `false` (byte-identical
behavior, proven by the two existing tests that already assert `dispatch.Brief` stays populated for
the direct path). `runCodexBuildPlanOnlyWithOptions` calls it with `true`, right after
`attachBuildDispatchContext`, and threads the returned `briefPaths` into `buildCodexBuildManifest`'s
`workerBriefs` parameter instead of `nil`.

---

### The registry that attributes but does not count (the gap in criterion 2's machinery)

**Source** (`cmd/build_print_brief.go:214-246`, `splitBriefSections`):
```go
func splitBriefSections(brief string) []briefSection {
	...
	current := "(preamble)"
	size := 0
	flush := func() {
		if size > 0 {
			sections = append(sections, briefSection{Name: current, Chars: size})
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if !briefOwnedSections[heading] {
				size += len(line) + 1
				continue
			}
			flush()
			current = heading   // <-- a SECOND "## Pheromone Signals" simply restarts here; nothing notices it already flushed once
			size = len(line) + 1
			continue
		}
		size += len(line) + 1
	}
	flush()
	return sections
}
```
If a heading occurs twice, `flush()` is called (and a section recorded) for the FIRST occurrence
before `current` resets for the SECOND — the result is two separate `briefSection` entries with the
SAME `Name`. `renderBriefComposition`/`renderBriefChecklist` neither merge nor flag same-named
entries; `checklistRowFor`'s `strings.Index`-based lookup only ever finds the FIRST occurrence and
would not even notice a second one exists.

**The fix:** a new function, `duplicatedBriefSections(assembled string) []string`, that:
1. Counts occurrences per owned heading name (a `map[string]int`, not "first wins").
2. Separately counts occurrences of a stable anchor substring drawn from the handoff-schema sentence
   (`composeBuildManifestBrief`'s appended text, `cmd/codex_build.go:3233` — plain text, no heading of
   its own, per 189's own D-04 design, so heading-counting alone cannot catch it).
Returns the names of anything found more than once. `printWorkerBriefs` calls this per dispatch and
turns a non-empty result into a returned error — a `--print-brief` run that finds duplication FAILS,
per the ROADMAP's own word "asserts."

---

### The two live channels for hive wisdom (the double-injection, precisely located)

**Channel 1 — Go colony-prime, the survivor** (`cmd/colony_prime_context.go:708-741`):
```go
hubDir := resolveHubPath()
...
hiveEntries := readHiveWisdomEntriesForDomains(hubDir, 5, readRegistryDomainsForRepo(hubDir, repoRoot), &fallbacks)
...
hiveLines := buildHiveWisdomLines(hiveEntries)
if len(hiveLines) > 0 {
	var hiveSB strings.Builder
	writeSectionHeader(&hiveSB, "hive_wisdom", "## HIVE WISDOM (Cross-Colony Patterns)\n\n")
	...
	sections = append(sections, colonyPrimeSection{name: "hive_wisdom", ...})
}
```
Reached via `resolveCodexWorkerContext()` — confirmed called from NINE sites across build, continue,
plan, colonize, swarm, and seal. This is "canonical" by evidence of breadth, not by assertion.

**Channel 2 — TS-host, the duplicate** (`.aether/ts-host/src/hive-injector.ts:139-150`,
`formatHiveWisdomSection` — note the IDENTICAL header string):
```typescript
export function formatHiveWisdomSection(entries: HiveWisdomEntry[]): string {
  if (entries.length === 0) return "";
  const lines: string[] = ["## HIVE WISDOM (Cross-Colony Patterns)"];
  for (const entry of entries) {
    lines.push(`(${entry.domain}, ${entry.confidence.toFixed(2)}) ${entry.text}`);
  }
  return lines.join("\n\n");
}
```
Attached at FOUR call sites in `.aether/ts-host/src/host.ts`, all sharing this exact shape:
```typescript
const hiveSection = await prepareHiveSection(bridge);
for (const d of dispatches) {
  d.hive_section = hiveSection;
}
```
(`runDryRunDispatchedCommand:631`, `runDispatchedBuildCommand:1124`, `runDispatchedPlanCommand:1313`,
`runDispatchedContinueCommand:1427`.)

**Where the two channels collide** (`cmd/internal_worker_adapter.go:404`):
```go
skillSection := joinInternalWorkerSections(request.SkillSection, request.HiveSection)
config := codex.WorkerConfig{
	...
	ContextCapsule: strings.TrimSpace(request.ContextCapsule),   // ALREADY has "## HIVE WISDOM" baked in, from channel 1
	SkillSection:   skillSection,                                 // ALSO has "## HIVE WISDOM" baked in, from channel 2
	...
}
```
Both `ContextCapsule` and `SkillSection` reach `AssemblePrompt`/`AssembleHostedPrompt`
(`pkg/codex/worker.go:381`, `pkg/codex/platform_dispatch.go:892,947`) as separate sections of the same
assembled prompt.

**The fix:** remove channel 2 entirely — the four `host.ts` call sites, `prepareHiveSection`,
`emitHiveSummary` and its four call sites, `toWorkerDispatches`'s `hive_section` mapping, the
`hive_section` field on every TS interface that carries it, `hive-injector.ts` itself, the `HiveSection`
field on `internalWorkerDispatchRequest`, and the now-single-argument, now-trivial
`joinInternalWorkerSections` (deleted per D-10, mirroring 189's own "delete the now-orphaned helper"
precedent rather than leaving harmless-looking dead code for a future ratchet to catch).

---

### The wrapper-triplet + companion-doc rule (already established, reused verbatim)

**Source** (CLAUDE.md's UX Architecture / YAML Source Chain section; confirmed empirically by `diff`
this phase): `.claude/commands/ant/build.md`, `.claude/commands/ant-build.md`, and
`.opencode/commands/ant/build.md` are byte-identical today. `build.md`'s OWN "Cross-Platform Drift
Guard" section additionally names `.aether/commands/build.yaml`, `cmd/command_guide.go`, and the Codex
skill `aether-colony-build-cycle` as required companions to any manifest-handling prose change:
```
If you change build context framing, signal presentation, manifest handling,
worker spawning, finalization, or closeout behavior here, update
`.aether/commands/build.yaml`, `cmd/command_guide.go`, and the Codex skill
`aether-colony-build-cycle` in the same change.
```
**Apply to:** all six prose surfaces (3-file wrapper triplet + YAML + Go guide + skill), edited
identically for the "brief_path is now the routine case" fact, in ONE task (per the deliverables'
"keep wrapper-triplet edits within one plan" instruction, extended here to the full six-surface set
this file's own rule names).

## Shared Patterns

### One canonical helper, multiple readers converge on it
**Source:** Phase 188's D-02/D-04 (`loadMiddenFile`/`continueSupersededResult`); reused by Phase 189
for `codex.HandoffFieldsSummary`.
**Apply to:** `writeBuildWorkerBriefFiles` (both `writeCodexBuildArtifacts` and
`runCodexBuildPlanOnlyWithOptions` converge on it, distinguished only by a boolean flag, never a
hand-copied second loop); `resolveCodexWorkerContext()` remaining hive wisdom's sole channel once the
TS-side duplicate is removed.

### Delete the now-trivial/now-orphaned helper outright
**Source:** Phase 189's D-02 (`findDispatchTask` deleted once both call sites migrated to
`findDispatchTasks`).
**Apply to:** `joinInternalWorkerSections` (D-10) — once `HiveSection` is gone, its one remaining
call site has one remaining argument, and a single-argument "join" is a no-op wrapper worth deleting,
not preserving.

### A flag distinguishes two still-valid, genuinely different caller contracts
**New to this phase**, but structurally the same shape as Phase 189's D-04/D-06 (placement depends on
which composition layer a function feeds) — here, `clearInlineBrief` distinguishes
`writeCodexBuildArtifacts`'s callers (need `Brief` to survive, proven by existing tests) from
`runCodexBuildPlanOnlyWithOptions` (needs it cleared, the entire point of this phase's criterion 1).

### A guard/test that finds nothing must fail, not pass
**Source:** Phase 187's and 188's and 189's own ratchets.
**Apply to:** `duplicatedBriefSections`'s tests must be proven against a REAL duplicate (e.g.
temporarily double-appending a section in a test-local copy, not production code, and confirming the
detector catches it) before trusting it catches nothing on healthy input; the criterion-4 invariant
test must use a fixture with genuinely non-trivial, unique brief content, not an empty-dispatch edge
case that would pass trivially.

### Compiled output is tracked and load-bearing
**Source:** this project's own CLAUDE.md warning, restated by the orchestrator for this exact phase;
confirmed empirically via `.gitignore`'s explicit `!.aether/ts-host/dist/**` un-ignore and
`cmd/ts_host_artifacts.go:13`'s hardcoded `dist/host.js` entrypoint.
**Apply to:** every TS-side edit in Plan 190-02 must end with a clean `dist/` rebuild
(`rm -rf dist && npm run build`, not an incremental build — stale per-file output is not
auto-deleted) and a `git status`/`git ls-files` check proving both that something changed and that
`hive-injector.js`/`.d.ts` are gone.

## No Analog Found

| File/Test | Role | Data Flow | Reason |
|---|---|---|---|
| `duplicatedBriefSections`'s counting pass | test-support function (invariant detector) | batch | No existing function in this codebase counts `## ` heading occurrences (all existing brief-inspection code is presence/absence via `strings.Contains` or first-match via `strings.Index`) — this is new because the underlying requirement (assert ZERO duplicates, not just attribute content) is itself new to this phase. |
| The TS-side "hive_section is now absent" regression tests | test (negative invariant) | batch | No existing TS test in `.aether/ts-host/test/` asserts a field's ABSENCE as a design decision (existing hive tests all assert presence, since they were written to prove the ORIGINAL feature worked) — this is new because the design decision (remove, don't just stop reading) is itself new to this phase. The closest structural relative is `pkg/codex`'s `TestBuildWorkerBriefOmitsPlaybooks`-style forbidden-substring pattern (Go side, Phase 160-era), reused here as the shape but applied to a TS interface/field-shape assertion instead of a rendered string. |

## Metadata

**Analog search scope:** `cmd/codex_build*.go`, `cmd/build_print_brief*.go`, `cmd/internal_worker_adapter*.go`,
`.aether/ts-host/src/{host,worker-dispatch,types,hive-injector}.ts`, `.aether/ts-host/test/*.ts`,
`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.aether/commands/build.yaml`,
`cmd/command_guide.go`, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`.
**Files scanned directly (Read or targeted grep):** `cmd/codex_build.go` (full-file structural read,
~600 lines across multiple passes), `cmd/codex_build_test.go` (targeted, ~400 lines),
`cmd/build_print_brief.go` (full file, 429 lines), `cmd/internal_worker_adapter.go` (full file, 515
lines), `cmd/colony_prime_context.go` (targeted, ~740-line region), `.aether/ts-host/src/host.ts`
(full-file structural pass via function index + 5 targeted reads, 1695 lines total),
`.aether/ts-host/src/hive-injector.ts` (full file, 150 lines), `.aether/ts-host/src/worker-dispatch.ts`
(targeted), `.aether/ts-host/src/types.ts` (targeted), `.claude/commands/ant/build.md` (full file, 358
lines), `.opencode/commands/ant/{build,continue}.md` (targeted diffs and greps), `.codex/CODEX.md`
(targeted), `.gitignore` (targeted), `.aether/ts-host/package.json`/`tsconfig*.json` (full).
**Pattern extraction date:** 2026-08-20
