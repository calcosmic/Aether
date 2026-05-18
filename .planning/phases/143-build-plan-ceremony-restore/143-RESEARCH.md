# Phase 143: Build + Plan Ceremony Restore - Research

**Researched:** 2026-05-18
**Domain:** Ceremony rendering, playbook-driven orchestration, TS host document injection, Go ceremony adapter events
**Confidence:** HIGH

## Summary

Phase 143 restores playbook-driven ceremony for build and plan workflows. The core problem is that the TS host currently has hardcoded ceremony rendering logic in `host.ts` (fetch manifest, render spawn-plan, dispatch workers, render worker-complete, render closeout) that does NOT follow the step sequences defined in the existing `.aether/docs/command-playbooks/build-*.md` files. The playbooks define rich, multi-step procedures (prep, context loading, archaeology, suggestion analysis, wave spawning, verification, synthesis, display), but the TS host short-circuits all of that by directly calling Go ceremony commands with manifest JSON.

The Go side already has the document-injection pattern via `renderBuildPlaybookContext()` in `cmd/build_playbook_context.go` -- it reads playbook markdown files from disk, truncates to a per-file character budget, and injects them as a "Relevant Playbooks" section into worker briefs. This is the pattern CEREMONY-06 asks for.

The current architecture has three conductors:
1. **YAML `wrapper_additions.orchestration`** -- a 15-step procedure in `build.yaml` that tells Claude/OpenCode wrappers how to orchestrate builds
2. **TS host `host.ts`** -- `runDispatchedBuildCommand()` / `runDispatchedPlanCommand()` that do their own ceremony flow
3. **Go ceremony commands** -- `aether ceremony spawn-plan/wave-start/worker-complete/closeout` that render visual output

CEREMONY-04 demands one conductor per platform. The plan is: TS host reads playbooks and follows their step sequences (playbook-driven), YAML is slimmed to packaging only (CEREMONY-03), and Go emits events that the TS host renders following playbook timing (CEREMONY-05).

The 465 TS tests include ~27 files referencing ceremony/playbook/spawn-plan/wave-start/closeout, but most ceremony tests test the GoCeremonyAdapter routing, narrator event handling, and snapshot comparisons. The real coupling risk is in `host.test.ts`, `host-integration.test.ts`, `golden-workflow.test.ts`, and `lifecycle.test.ts` which test the full dispatched pipeline.

**Primary recommendation:** Add a `PlaybookLoader` to the TS host that reads playbook markdown files (same as Go's `buildPlaybookCandidates()` pattern), create a `PlaybookConductor` that sequences ceremony events following playbook steps, and slim the YAML `wrapper_additions.orchestration` sections to packaging-only. The Go ceremony adapter continues emitting events; the TS host renders them at the timing dictated by playbook steps rather than hardcoded flow.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CEREMONY-01 | TS host reads build playbooks (build-prep, build-context, build-wave, build-verify, build-complete) and follows their step sequences | 5 build playbooks exist at `.aether/docs/command-playbooks/build-*.md`; Go already has `renderBuildPlaybookContext()` at `cmd/build_playbook_context.go:13` that reads playbooks from repo/hub candidates; TS host `template-loader.ts` already loads `.aether/templates/ceremony/*.md` files |
| CEREMONY-02 | TS host reads plan playbooks and follows Scout -> Route-Setter flow with grounding gate integration | No plan playbooks currently exist; plan flow is in YAML `wrapper_additions.orchestration` (20 steps); Go `codexBuildPlaybooks()` at `cmd/codex_build.go:729` has no plan counterpart |
| CEREMONY-03 | YAML slimmed to packaging only -- remove orchestration procedure from `build.yaml` wrapper_additions.orchestration | build.yaml orchestration section has 15 steps (lines 21-37); plan.yaml has 20 steps (lines 32-53); continue.yaml has 14 steps in orchestration (lines 22-41) |
| CEREMONY-04 | One conductor per platform: Claude/OpenCode wrappers + TS host (playbook-driven), Codex (runtime-native command-guide + skills) | Three current conductors identified: YAML orchestration, TS host `host.ts` dispatched runners, Go ceremony commands; Codex already uses `command-guide` + skills (verified in YAML `codex_orchestration` sections) |
| CEREMONY-05 | Go ceremony adapter emits events; TS host renders them following playbook timing | Go `ceremony_emitter.go` already emits events via event bus; TS host `event-bridge.ts` already consumes `ceremony.*` events; TS host `narrator.ts` already handles `ceremony.build.spawn/wave.start/wave.end/closeout` events |
| CEREMONY-06 | Document-injection pattern for playbook consumption (not step-parsing) -- same pattern as existing Go `renderBuildPlaybookContext` | Go pattern at `cmd/build_playbook_context.go:13-41`: reads markdown files, truncates to budget, injects as "Relevant Playbooks" section in worker brief; candidate resolution at lines 57-77 checks repo, hub, and absolute paths |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Playbook file resolution | Go runtime (`cmd/build_playbook_context.go`) | TS host (reads from repo/hub) | Go already resolves playbook candidates across repo, hub, and absolute paths; TS host reuses same resolution logic |
| Ceremony event emission | Go runtime (`cmd/ceremony_emitter.go`) | -- | Go owns the event bus and ceremony lifecycle; TS host consumes, never emits ceremony events |
| Ceremony rendering (visual) | Go runtime (`cmd/ceremony_cmd.go`) | -- | Go owns all visual rendering via `aether ceremony` subcommands; TS host calls them, never reimplements |
| Playbook step sequencing | TS host (NEW) | -- | TS host is the conductor for Claude/OpenCode; it reads playbooks and sequences ceremony calls at playbook-dictated timing |
| Worker dispatch orchestration | TS host (`host.ts`) | -- | TS host already owns dispatch pipeline for Claude/OpenCode platform |
| YAML packaging metadata | YAML sources (`.aether/commands/*.yaml`) | -- | YAML defines command metadata, guardrails, codex orchestration; orchestration procedure removed per CEREMONY-03 |
| Codex ceremony | Go runtime (command-guide + skills) | -- | Codex is runtime-native; no wrapper layer; already handled via `codex_orchestration` sections |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Node.js `fs` | Built-in | Read playbook markdown files | Already used by `template-loader.ts` and Go's `build_playbook_context.go` |
| Node.js `path` | Built-in | Resolve playbook paths across repo/hub | Already used by `template-loader.ts:134` |
| Node.js `child_process` (execFileSync) | Built-in | Call Go ceremony subcommands | Already used by `ceremony-adapter.ts:49` |
| Go stdlib `os` | Go 1.26+ | Read playbook files | Already used by `build_playbook_context.go:43` |
| Go stdlib `path/filepath` | Go 1.26+ | Resolve playbook candidates | Already used by `build_playbook_context.go:57-77` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `js-yaml` | TS dependency (existing) | Parse playbook frontmatter | Already used by `template-loader.ts:90` for ceremony templates |
| Go `events` package | Existing | Ceremony event emission | Already used by `ceremony_emitter.go:12` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| TS host reads playbooks from disk | Go serves playbook content via JSON API | Adds a new Go endpoint; disk reads are simpler and match the existing Go `renderBuildPlaybookContext` pattern |
| Step-parsing playbook markdown | Document-injection (whole-playbook-as-context) | Step-parsing is fragile (markdown formatting changes break it); document-injection is what Go already does and is CEREMONY-06's explicit requirement |
| New plan playbooks | Reuse build playbooks for plan flow | Plan has fundamentally different steps (Scout then Route-Setter, grounding gate); a dedicated plan playbook is clearer |

**Installation:**
```bash
# Zero new dependencies -- all changes use existing packages
```

**Version verification:** Go 1.26.1 darwin/arm64 installed and current. Node.js present. [VERIFIED: `go version`, `node --version`]

## Architecture Patterns

### System Architecture Diagram

```
YAML Source (.aether/commands/build.yaml)
    |
    +-- name, description, guardrails, codex_orchestration
    +-- wrapper_additions.orchestration REMOVED (CEREMONY-03)
    +-- wrapper_additions.pre_build RETAINED (packaging only)
    +-- wrapper_additions.post_build RETAINED (packaging only)
    |
    v
Claude/OpenCode Wrapper (.claude/commands/ant/build.md)
    |
    +-- Reads YAML for metadata
    +-- Calls: aether host build $ARGUMENTS
    |
    v
TS Host (host.ts) -- THE CONDUCTOR
    |
    +-- Step 1: loadPlaybooks("build") -> [build-prep.md, build-context.md, ...]
    |   |
    |   +-- resolvePlaybookCandidates(root, playbook) -- same pattern as Go
    |   +-- readFileSync(candidate) -- first match wins
    |
    +-- Step 2: fetchManifest() -> callGoJSON("host build --plan-only")
    |   |
    |   +-- manifest.playbooks: ["build-prep.md", "build-wave.md", ...]
    |   +-- manifest.dispatches: [...]
    |   +-- manifest.execution_plan: [...]
    |
    +-- Step 3: for each playbook step in sequence:
    |   |
    |   +-- playbook says "render spawn-plan"
    |   |       -> ceremony.renderSpawnPlan("build", manifest)
    |   |       -> Go renders visual output
    |   |       -> TS host writes to stderr
    |   |
    |   +-- playbook says "dispatch wave 1"
    |   |       -> ceremony.renderWaveStart("build", manifest, wave=1)
    |   |       -> dispatchWorkers(dispatches for wave 1)
    |   |       -> ceremony.renderWorkerComplete("build", result)
    |   |
    |   +-- playbook says "finalize"
    |           -> callGoJSON("build-finalize", completionFile)
    |           -> ceremony.renderCloseout("build", completionFile)
    |
    v
Go Runtime (sole authority)
    |
    +-- cmd/codex_build.go -- manifest generation, state management
    +-- cmd/codex_build_finalize.go -- finalizer, state writes
    +-- cmd/ceremony_cmd.go -- visual rendering
    +-- cmd/ceremony_emitter.go -- event emission to bus
    +-- cmd/build_playbook_context.go -- playbook resolution for worker briefs
    |
    v
Event Bus (pkg/events/)
    |
    +-- ceremony.build.spawn
    +-- ceremony.build.wave.start
    +-- ceremony.build.wave.end
    +-- ceremony.build.closeout
    |
    v
TS Narrator (narrator.ts) -- EVENT CONSUMER (read-only)
    |
    +-- Subscribes to ceremony.* events
    +-- Renders via visual/markdown/json renderer
    +-- Writes to stdout (when dashboard not active)
```

### Recommended Project Structure
```
.aether/ts-host/src/
    playbook-loader.ts       # NEW: resolvePlaybookCandidates, loadPlaybook, loadPlaybooksForWorkflow
    playbook-conductor.ts    # NEW: PlaybookConductor class that sequences ceremony calls
    ceremony-adapter.ts      # EXISTING: GoCeremonyAdapter (unchanged)
    narrator.ts              # EXISTING: event consumer (unchanged)
    host.ts                  # MODIFY: use PlaybookConductor instead of hardcoded ceremony flow

.aether/ts-host/test/
    playbook-loader.test.ts  # NEW: unit tests for playbook resolution and loading
    playbook-conductor.test.ts # NEW: unit tests for ceremony sequencing

.aether/docs/command-playbooks/
    plan-prep.md             # NEW: plan preparation steps (optional, CEREMONY-02)
    plan-dispatch.md         # NEW: Scout -> Route-Setter dispatch flow (CEREMONY-02)
    build-prep.md            # EXISTING: already exists
    build-context.md         # EXISTING: already exists
    build-wave.md            # EXISTING: already exists
    build-verify.md          # EXISTING: already exists
    build-complete.md        # EXISTING: already exists
```

### Pattern 1: Playbook Loader (Document-Injection)

**What:** Resolve and load playbook markdown files from repo, hub, and absolute paths -- identical to Go's `buildPlaybookCandidates()`.
**When to use:** Whenever the TS host needs playbook content for a workflow.
**Example:**
```typescript
// Source: pattern from cmd/build_playbook_context.go:57-77
// NEW: .aether/ts-host/src/playbook-loader.ts

import { readFileSync, existsSync } from "node:fs";
import { join, isAbsolute } from "node:path";
import { homedir } from "node:os";

export interface Playbook {
  name: string;
  path: string;
  content: string;
}

const PLAYBOOK_DIR = ".aether/docs/command-playbooks";
const HUB_PLAYBOOK_DIR = "system/docs/command-playbooks";

/** Resolve playbook file candidates (same order as Go's buildPlaybookCandidates). */
function resolvePlaybookCandidates(root: string, playbook: string): string[] {
  const trimmed = playbook.trim();
  if (!trimmed) return [];
  const candidates: string[] = [];

  if (isAbsolute(trimmed)) {
    candidates.push(trimmed);
  }

  // Repo-local
  candidates.push(join(root, PLAYBOOK_DIR, trimmed));
  candidates.push(join(root, trimmed));

  // Hub
  const hubDir = join(homedir(), ".aether");
  candidates.push(join(hubDir, HUB_PLAYBOOK_DIR, trimmed));
  candidates.push(join(hubDir, trimmed));

  return candidates;
}

/** Load a single playbook by name, trying candidates in order. */
export function loadPlaybook(root: string, name: string): Playbook | null {
  const candidates = resolvePlaybookCandidates(root, name);
  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      return {
        name,
        path: candidate,
        content: readFileSync(candidate, "utf-8"),
      };
    }
  }
  return null;
}

/** Load all playbooks for a workflow. */
export function loadPlaybooksForWorkflow(root: string, workflow: "build" | "plan" | "continue"): Playbook[] {
  const playbookLists: Record<string, string[]> = {
    build: [
      "build-prep.md",
      "build-context.md",
      "build-wave.md",
      "build-verify.md",
      "build-complete.md",
    ],
    plan: [
      "plan-prep.md",    // NEW file
      "plan-dispatch.md", // NEW file
    ],
    continue: [
      "continue-verify.md",
      "continue-gates.md",
      "continue-advance.md",
      "continue-finalize.md",
    ],
  };

  const names = playbookLists[workflow] ?? [];
  const playbooks: Playbook[] = [];
  for (const name of names) {
    const pb = loadPlaybook(root, name);
    if (pb) playbooks.push(pb);
  }
  return playbooks;
}
```

### Pattern 2: Playbook Conductor (Sequencing)

**What:** A conductor class that takes playbook content and the ceremony adapter, and sequences ceremony calls following playbook step markers.
**When to use:** In `runDispatchedBuildCommand` and `runDispatchedPlanCommand` instead of hardcoded ceremony flow.
**Example:**
```typescript
// Source: NEW: .aether/ts-host/src/playbook-conductor.ts

import type { CeremonyAdapter, CeremonyWorkflow } from "./ceremony-adapter.js";
import type { BuildManifest, BuildDispatch } from "./types.js";

/**
 * Playbook step markers that the conductor recognizes.
 * These correspond to Go ceremony adapter methods.
 */
export type PlaybookStepType =
  | "render-spawn-plan"
  | "render-wave-start"
  | "dispatch-workers"
  | "render-worker-complete"
  | "finalize"
  | "render-closeout"
  | "load-context"       // colony-prime, pheromones, survey
  | "update-state"        // state-mutate
  | "git-checkpoint"      // stash/commit
  | "display"             // free-form display text
  | "custom";             // Go command call

export interface PlaybookStep {
  type: PlaybookStepType;
  /** Human-readable description from playbook. */
  description: string;
  /** For wave-start: which execution wave. */
  wave?: number;
  /** For custom: the Go command to call. */
  command?: string[];
  /** For display: the text to show. */
  text?: string;
}

/**
 * Parse a playbook into a sequence of steps.
 *
 * This is NOT step-parsing of arbitrary markdown. It recognizes a defined
 * set of ceremony hooks (CEREMONY-06: document-injection, not step-parsing).
 * The playbook content is injected as context; only the hooks trigger actions.
 */
export function parsePlaybookSteps(content: string): PlaybookStep[] {
  // The playbook content is used as document-injection for worker context.
  // The conductor recognizes known ceremony markers:
  // - "## Step N: ..." headings define ordering
  // - Known step names map to ceremony adapter calls
  //
  // For build playbooks, the step mapping is:
  const stepPatterns: [RegExp, PlaybookStepType][] = [
    [/## Step 1[:.].*Validate.*Read State/i, "load-context"],
    [/## Step 2[:.].*Update State/i, "update-state"],
    [/## Step 3[:.].*Git Checkpoint/i, "git-checkpoint"],
    [/## Step 4[:.].*Load Colony Context/i, "load-context"],
    [/## Step 5[:.].*Spawn.*Wave 1/i, "render-spawn-plan"],
    [/## Step 5\.1[:.].*Spawn Wave 1/i, "dispatch-workers"],
    [/## Step 5\.2[:.].*Process Wave 1/i, "render-worker-complete"],
    [/## Step 7[:.].*Display Results/i, "display"],
    [/## Step 8[:.].*Update Session/i, "finalize"],
  ];

  const steps: PlaybookStep[] = [];
  for (const [pattern, type] of stepPatterns) {
    if (pattern.test(content)) {
      steps.push({ type, description: `Matched: ${pattern.source}` });
    }
  }
  return steps;
}
```

**IMPORTANT DESIGN NOTE:** The above step-parsing is illustrative. Per CEREMONY-06, the primary consumption pattern is **document-injection** (inject the full playbook as context into worker prompts), NOT step-parsing. The Go `renderBuildPlaybookContext()` function injects playbook content truncated to a character budget. The conductor uses the playbooks for timing awareness (knowing what ceremony events to emit and when), but does not parse arbitrary markdown instructions.

The actual conductor implementation should:
1. Load playbooks for the workflow
2. Follow a predefined ceremony sequence for the workflow (not parsed from playbooks)
3. Inject playbook content into worker briefs as document-injection (same as Go pattern)
4. Call the ceremony adapter at the right points in the sequence

### Pattern 3: Playbook Document Injection (CEREMONY-06)

**What:** Inject playbook markdown content into worker prompts as a "Relevant Playbooks" section, truncated to a character budget.
**When to use:** When assembling worker briefs, after skill injection but before dispatch.
**Example:**
```typescript
// Source: NEW: .aether/ts-host/src/playbook-loader.ts
// Pattern from cmd/build_playbook_context.go:13-41

const PLAYBOOK_BUDGET_CHARS = 7000;
const PLAYBOOK_PER_FILE_CHARS = 2800;

export function renderPlaybookContext(
  playbooks: Playbook[],
  maxBudget = PLAYBOOK_BUDGET_CHARS,
  maxPerFile = PLAYBOOK_PER_FILE_CHARS
): string {
  if (playbooks.length === 0) return "";

  const parts: string[] = ["## Relevant Playbooks\n"];
  let remaining = maxBudget;

  for (const pb of playbooks) {
    if (remaining <= 0) break;
    const header = `### ${pb.name}\n\n`;
    const maxContent = Math.min(maxPerFile, remaining - header.length - 2);
    if (maxContent <= 0) break;

    let snippet = pb.content;
    if (snippet.length > maxContent) {
      snippet = snippet.slice(0, maxContent) + "\n\n[playbook truncated]";
    }

    parts.push(header + snippet + "\n\n");
    remaining = maxBudget - parts.join("").length;
  }

  return parts.join("").trimEnd();
}
```

### Anti-Patterns to Avoid

- **Step-parsing arbitrary markdown:** Playbook markdown is written for human readers and LLM consumption, not as a structured step protocol. Do not try to parse "Step N:" headings as executable instructions. Use document-injection instead (CEREMONY-06).
- **Duplicating Go ceremony rendering in TypeScript:** The Go runtime already renders all ceremony visuals via `aether ceremony` subcommands. The TS host must call these, not reimplement them. The narrator is for event-driven rendering only.
- **Adding orchestration steps back into YAML:** CEREMONY-03 explicitly removes the 15-step orchestration from YAML. Keep YAML to packaging metadata only.
- **Making the conductor block on playbook file I/O:** Playbook loading is synchronous file I/O. It should happen once at the start of the workflow, not per-step.
- **Treating plan playbooks as optional:** CEREMONY-02 requires plan playbooks. If they don't exist, the conductor should error gracefully, not fall back to hardcoded flow.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Playbook file resolution | Custom path joining | Reuse Go's `buildPlaybookCandidates` pattern | Already handles repo, hub, and absolute paths; well-tested |
| Visual ceremony rendering | TS ANSI rendering | `aether ceremony` Go subcommands | Go owns all visual rendering; duplicating in TS creates drift |
| Playbook truncation logic | Custom string slicing | Go's `truncateTextWithMarker` pattern (same constants) | Same 7000/2800 character budgets; keeps injection consistent |
| Worker brief assembly | New assembly in TS host | Go `renderCodexBuildWorkerBrief` already includes playbook context | Playbook context is already in the manifest's `worker_briefs` field |
| Event bus protocol | New event types | Existing `ceremony.*` event topics | Event bridge already consumes these; narrator already handles them |

**Key insight:** The Go runtime already does most of what CEREMONY-01 through CEREMONY-06 require. The TS host's job is to follow the playbook-defined ceremony sequence when calling Go, not to reimplement ceremony logic. The main new code is the `PlaybookLoader` and integrating it into the existing `runDispatchedBuildCommand` / `runDispatchedPlanCommand` flows.

## Runtime State Inventory

> Not applicable -- this phase is a code change only, not a rename/refactor/migration.

N/A -- Phase 143 does not rename or migrate runtime state. It modifies how ceremony is rendered.

## Common Pitfalls

### Pitfall 1: Confusing Document-Injection with Step-Parsing
**What goes wrong:** Implementing a markdown parser that tries to extract executable steps from playbook headings, then executing them in sequence.
**Why it happens:** The playbooks have "Step N:" headings that look like executable instructions.
**How to avoid:** CEREMONY-06 explicitly says "document-injection pattern for playbook consumption (not step-parsing)." The playbooks are injected as context into worker prompts, not parsed as a protocol. The conductor uses a predefined sequence; the playbooks provide context.
**Warning signs:** A markdown parser in `playbook-conductor.ts` that extracts step numbers and descriptions.

### Pitfall 2: Breaking the 465 TS Tests
**What goes wrong:** Changing `host.ts` ceremony flow breaks 27 test files that test dispatched build/plan/continue pipelines.
**Why it happens:** Tests mock `_callGoJSONRef` and `_dispatchWorkersRef` with specific call sequences. Changing the sequence breaks mock expectations.
**How to avoid:** Changes to `host.ts` should preserve the existing call sequence (fetch manifest, render ceremony, dispatch workers, finalize). The PlaybookLoader is an additive layer that provides context, not a replacement for the existing flow. Update test expectations incrementally per plan.
**Warning signs:** `npm test` shows more than the current 2 pre-existing failures.

### Pitfall 3: YAML Orchestration Drift
**What goes wrong:** Removing orchestration from YAML but forgetting to update the generated Claude/OpenCode wrapper commands.
**Why it happens:** Wrappers are generated from YAML via `aether publish`. If YAML is slimmed but wrappers aren't regenerated, they'll still reference the old 15-step procedure.
**How to avoid:** After slimming YAML, run `aether publish` and verify generated wrappers at `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` reflect the change.
**Warning signs:** Generated wrapper still has "## Worker Spawning" section with step-by-step instructions that duplicate TS host behavior.

### Pitfall 4: Playbook Files Not Published to Hub
**What goes wrong:** Playbooks exist in repo at `.aether/docs/command-playbooks/` but are not published to `~/.aether/system/docs/command-playbooks/`. TS host can't find them when running from other repos.
**Why it happens:** The publish pipeline may not sync playbook files.
**How to avoid:** Verify `aether publish` copies `.aether/docs/command-playbooks/` to the hub. If not, add the path to the publish manifest.
**Warning signs:** `loadPlaybook()` returns null for a playbook that exists in the repo.

### Pitfall 5: Plan Playbooks Don't Exist Yet
**What goes wrong:** CEREMONY-02 requires the TS host to read plan playbooks, but no plan playbooks exist (only build and continue playbooks exist).
**Why it happens:** The current plan flow is defined entirely in YAML `wrapper_additions.orchestration` (20 steps) and the TS host `runDispatchedPlanCommand`.
**How to avoid:** Create minimal plan playbooks (`plan-prep.md`, `plan-dispatch.md`) that capture the Scout -> Route-Setter flow. Or, if the plan ceremony is simple enough, use document-injection of the existing plan YAML guardrails as the "playbook."
**Warning signs:** `loadPlaybooksForWorkflow("plan")` returns empty array.

### Pitfall 6: Forgetting Codex Platform
**What goes wrong:** Changes to build/plan ceremony break Codex, which uses runtime-native command-guide + skills instead of wrappers.
**Why it happens:** CEREMONY-04 says Codex is "runtime-native command-guide + skills" and should NOT use playbook-driven ceremony.
**How to avoid:** Codex ceremony is owned by `cmd/command_guide.go` and `aether-colony-build-cycle` skill. TS host changes must NOT affect the Codex path. Verify `aether command-guide build --platform codex` still works after changes.
**Warning signs:** Codex build output format changes after TS host modifications.

## Code Examples

### Existing Go Playbook Resolution Pattern
```go
// Source: cmd/build_playbook_context.go:13-41 (VERIFIED)
func renderBuildPlaybookContext(root string, dispatch codexBuildDispatch, playbooks []string) string {
    selected := buildPlaybooksForDispatch(dispatch, playbooks)
    if len(selected) == 0 {
        return ""
    }
    var b strings.Builder
    b.WriteString("## Relevant Playbooks\n\n")
    remaining := buildWorkerBriefPlaybookBudgetChars
    for _, playbook := range selected {
        if remaining <= 0 {
            break
        }
        snippet := readBuildPlaybookSnippet(root, playbook, minInt(buildWorkerBriefPlaybookPerFileChars, remaining))
        if strings.TrimSpace(snippet) == "" {
            fmt.Fprintf(&b, "- `%s` (not found in repo or hub; use the manifest path as a reference)\n", playbook)
            continue
        }
        // ... truncate and append
    }
    return strings.TrimSpace(b.String())
}
```

### Existing Go Playbook Candidate Resolution
```go
// Source: cmd/build_playbook_context.go:57-77 (VERIFIED)
func buildPlaybookCandidates(root, playbook string) []string {
    playbook = filepath.ToSlash(strings.TrimSpace(playbook))
    if playbook == "" {
        return nil
    }
    var candidates []string
    if filepath.IsAbs(playbook) {
        candidates = append(candidates, filepath.FromSlash(playbook))
    }
    if strings.TrimSpace(root) != "" {
        candidates = append(candidates, filepath.Join(root, filepath.FromSlash(playbook)))
    }
    hubDir := resolveHubPath()
    if strings.TrimSpace(hubDir) != "" {
        trimmed := strings.TrimPrefix(playbook, ".aether/")
        candidates = append(candidates, filepath.Join(hubDir, "system", filepath.FromSlash(trimmed)))
        candidates = append(candidates, filepath.Join(hubDir, filepath.FromSlash(trimmed)))
    }
    candidates = append(candidates, filepath.FromSlash(playbook))
    return uniqueStringSlice(candidates)
}
```

### Existing Go Build Playbook List
```go
// Source: cmd/codex_build.go:729-736 (VERIFIED)
func codexBuildPlaybooks() []string {
    return []string{
        ".aether/docs/command-playbooks/build-prep.md",
        ".aether/docs/command-playbooks/build-wave.md",
        ".aether/docs/command-playbooks/build-verify.md",
        ".aether/docs/command-playbooks/build-complete.md",
    }
}
```

### Existing TS Host Ceremony Flow (to be playbook-driven)
```typescript
// Source: .aether/ts-host/src/host.ts:672-853 (VERIFIED)
// Current hardcoded flow in runDispatchedBuildCommand:
// Step 1: Fetch manifest
const buildResult = _callGoJSONRef<BuildManifestResult>(bridge, goArgs);
// Step 2: Render spawn-plan and wave-start ceremony
renderManifestCeremony(ceremony, "build", ceremonyEnvelope, dispatches);
// Step 3: Check available platforms
// Step 5: Initialize spawn orchestrator
// Step 6: Initialize confidence loop
// Step 7: Iteration loop (dispatch waves, evaluate confidence)
lastWaveResult = await dispatchBuildWave(bridge, parsed, ceremony, buildManifest, dispatches, spawnOrchestrator, iterationFeedback);
// Step 8: Render closeout
emitCeremonyOutput(ceremony.renderCloseout("build", finalCompletionPath));
```

### Existing TS Template Loader (pattern to reuse for playbook loading)
```typescript
// Source: .aether/ts-host/src/template-loader.ts:133-148 (VERIFIED)
export function loadTemplate(cwd: string, name: string): ParsedTemplate {
    const filePath = join(cwd, ".aether", "templates", "ceremony", `${name}.md`);
    let raw: string;
    try {
        raw = readFileSync(filePath, "utf-8");
    } catch {
        const fallback = DEFAULT_TEMPLATES[name];
        if (fallback) return fallback;
        throw new Error(`Template not found: ${name}`);
    }
    return parseTemplate(raw);
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Playbook steps hardcoded in wrapper markdown | Go ceremony commands called by TS host | v1.16 (Phase 106-111) | Wrappers delegate to TS host; Go owns rendering |
| Three conductors (YAML + TS host + Go ceremony) | One conductor per platform | This phase (143) | YAML slimmed, TS host is conductor for Claude/OpenCode |
| No plan playbooks | Plan playbooks to be created | This phase (143) | Enables CEREMONY-02 playbook-driven plan flow |
| Go worker briefs include playbook context via renderBuildPlaybookContext | Same (working well) | Existing | TS host should use same document-injection for consistency |

**Deprecated/outdated:**
- YAML `wrapper_additions.orchestration` -- being removed per CEREMONY-03
- Playbook step-parsing approaches -- CEREMONY-06 explicitly says document-injection

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 465 TS tests include ~27 ceremony-coupled test files that may break | Common Pitfalls #2 | Medium -- exact count verified (465 total, 463 pass, 2 pre-existing fail); ceremony coupling estimated from grep |
| A2 | No plan playbooks currently exist | Common Pitfalls #5 | Verified -- only build and continue playbooks exist at `.aether/docs/command-playbooks/` |
| A3 | Go `renderBuildPlaybookContext` is the document-injection pattern CEREMONY-06 refers to | Code Examples | High -- verified by reading `cmd/build_playbook_context.go` and the requirement text |
| A4 | The TS host already loads templates via `template-loader.ts` using the same file resolution pattern | Pattern 1 | Verified -- `loadTemplate` at `template-loader.ts:133-148` reads from `.aether/templates/ceremony/` |
| A5 | Codex does NOT use the TS host for ceremony | CEREMONY-04, Pitfall #6 | Medium -- Codex uses `command-guide` and skills per YAML `codex_orchestration` sections; no TS host dependency |

## Open Questions

1. **Should plan playbooks be new files or can we use the existing YAML guardrails as the "plan playbook"?**
   - What we know: CEREMONY-02 says "TS host reads plan playbooks." No plan playbooks exist.
   - What's unclear: Whether we need full markdown playbooks or can inject YAML guardrails as the plan context.
   - Recommendation: Create minimal `plan-prep.md` and `plan-dispatch.md` that capture the Scout -> Route-Setter flow. The plan ceremony is simpler than build (no archaeology, no suggestion analysis, no verification workers) so the playbooks can be short.

2. **Should the PlaybookConductor be a class or a set of functions?**
   - What we know: The TS host uses functional patterns (exported functions, not classes) for ceremony adapter, template loader, and narrator.
   - What's unclear: Whether a conductor class provides better testability.
   - Recommendation: Follow existing patterns -- use exported functions. The conductor is stateless (it receives ceremony adapter and manifest, returns rendered output).

3. **How does the grounding gate (Phase 142) integrate into the plan playbook flow?**
   - What we know: GROUND-06/07 add a plan-grounding validation gate in `plan-finalize`. CEREMONY-02 says "with grounding gate integration."
   - What's unclear: Whether the TS host needs to check grounding before dispatching workers, or if the Go finalizer handles it.
   - Recommendation: Grounding is a Go finalizer concern. The TS host doesn't need to check grounding -- it just follows the playbook flow. The Go `plan-finalize` already has the grounding check from Phase 142.

4. **Should the continue flow also become playbook-driven?**
   - What we know: Continue has 4 playbooks (continue-verify, continue-gates, continue-advance, continue-finalize). The continue YAML has 14 orchestration steps.
   - What's unclear: Whether CEREMONY-03 applies to continue as well, or only build and plan.
   - Recommendation: CEREMONY-03 says "remove orchestration procedure from build.yaml wrapper_additions.orchestration (the 15-step duplicate)." This implies the scope is build and plan. Continue YAML can be slimmed in a follow-up phase if needed.

## Environment Availability

Step 2.6: SKIPPED (no new external dependencies -- all changes use existing Go stdlib, Node.js built-ins, and existing project packages)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Node.js built-in test runner (`node:test`) |
| Config file | None (Node.js convention) |
| Quick run command | `cd .aether/ts-host && npm test 2>&1 \| grep -E "ℹ (tests|pass|fail)"` |
| Full suite command | `cd .aether/ts-host && npm test` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CEREMONY-01 | PlaybookLoader resolves build playbooks from repo/hub | unit | `node --test ts-host/test/playbook-loader.test.ts` | No -- Wave 0 |
| CEREMONY-01 | loadPlaybooksForWorkflow returns 5 build playbooks | unit | `node --test ts-host/test/playbook-loader.test.ts` | No -- Wave 0 |
| CEREMONY-01 | renderPlaybookContext truncates to budget | unit | `node --test ts-host/test/playbook-loader.test.ts` | No -- Wave 0 |
| CEREMONY-02 | loadPlaybooksForWorkflow returns plan playbooks | unit | `node --test ts-host/test/playbook-loader.test.ts` | No -- Wave 0 |
| CEREMONY-03 | build.yaml has no orchestration section | integration | `grep -c "orchestration:" .aether/commands/build.yaml` returns 0 | No -- Wave 0 |
| CEREMONY-04 | TS host conductor is sole conductor for Claude/OpenCode build | integration | `node --test ts-host/test/playbook-conductor.test.ts` | No -- Wave 0 |
| CEREMONY-05 | Go ceremony events render at playbook-timed intervals | integration | `node --test ts-host/test/playbook-conductor.test.ts` | No -- Wave 0 |
| CEREMONY-06 | Playbook content injected as document, not parsed as steps | unit | `node --test ts-host/test/playbook-loader.test.ts` | No -- Wave 0 |
| Regression | All 465 TS tests pass (no more than 2 pre-existing failures) | full | `cd .aether/ts-host && npm test` | YES -- existing |
| Regression | All Go tests pass | full | `go test ./... -count=1` | YES -- existing |

### Sampling Rate
- **Per task commit:** `cd .aether/ts-host && npm test 2>&1 | grep -E "ℹ (tests|pass|fail)"`
- **Per wave merge:** `cd .aether/ts-host && npm test`
- **Phase gate:** `cd .aether/ts-host && npm test` AND `go test ./... -count=1`

### Wave 0 Gaps
- [ ] `.aether/ts-host/src/playbook-loader.ts` -- covers CEREMONY-01, CEREMONY-02, CEREMONY-06
- [ ] `.aether/ts-host/test/playbook-loader.test.ts` -- unit tests for playbook resolution and document injection
- [ ] `.aether/ts-host/src/playbook-conductor.ts` -- covers CEREMONY-04, CEREMONY-05 (optional, may be simple functions)
- [ ] `.aether/ts-host/test/playbook-conductor.test.ts` -- unit tests for ceremony sequencing
- [ ] `.aether/docs/command-playbooks/plan-prep.md` -- CEREMONY-02 plan playbook (if needed)
- [ ] `.aether/docs/command-playbooks/plan-dispatch.md` -- CEREMONY-02 plan playbook (if needed)

## Security Domain

> Not applicable to this phase. Ceremony rendering is a presentation layer concern with no authentication, session management, access control, or cryptography requirements. No ASVS categories apply.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis of `.aether/ts-host/src/host.ts` (full file, 1225 lines) -- `runDispatchedBuildCommand`, `runDispatchedPlanCommand`, `runDispatchedContinueCommand`, ceremony helpers verified
- Direct codebase analysis of `.aether/ts-host/src/ceremony-adapter.ts` (full file, 193 lines) -- `GoCeremonyAdapter`, ceremony command routing verified
- Direct codebase analysis of `.aether/ts-host/src/narrator.ts` (full file, 180 lines) -- event handler registration, renderer selection verified
- Direct codebase analysis of `.aether/ts-host/src/template-loader.ts` (full file, 149 lines) -- template loading pattern to reuse for playbooks
- Direct codebase analysis of `cmd/build_playbook_context.go` (full file, 85 lines) -- `renderBuildPlaybookContext`, `buildPlaybookCandidates`, budget constants verified
- Direct codebase analysis of `cmd/ceremony_cmd.go` (full file, 1020 lines) -- ceremony subcommands, rendering functions verified
- Direct codebase analysis of `cmd/codex_build.go` -- `codexBuildPlaybooks()` at line 729, `renderCodexBuildWorkerBrief` at line 2041, playbook integration in briefs verified
- Direct codebase analysis of `cmd/ceremony_emitter.go` (first 60 lines) -- event emission pattern verified
- Direct codebase analysis of `.aether/commands/build.yaml` (full file, 57 lines) -- orchestration section at lines 21-37 (15 steps) verified
- Direct codebase analysis of `.aether/commands/plan.yaml` (full file, 68 lines) -- orchestration section at lines 32-53 (20 steps) verified
- Direct codebase analysis of `.aether/docs/command-playbooks/README.md` -- playbook structure verified
- Direct codebase analysis of `.aether/docs/command-playbooks/build-prep.md` (full file, 380 lines) -- step structure verified
- Direct codebase analysis of `.aether/ts-host/test/` -- 42 test files, 465 tests (463 pass, 2 pre-existing fail) verified via npm test
- Phase 142 RESEARCH.md -- grounding gate integration points verified

### Secondary (MEDIUM confidence)
- `.aether/docs/wrapper-runtime-ux-contract.md` -- wrapper/runtime ownership boundaries
- `.aether/docs/ceremony-revival-v1.6-handoff.md` -- historical ceremony architecture decisions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- zero new dependencies, all existing packages
- Architecture: HIGH -- all integration points verified by reading source code; Go patterns documented and reusable
- Pitfalls: HIGH -- test breakage risk well-understood (465 tests, 27 ceremony-coupled files); step-parsing vs document-injection explicitly addressed by CEREMONY-06

**Research date:** 2026-05-18
**Valid until:** 2026-06-18 (stable -- no external dependencies or fast-moving libraries)
