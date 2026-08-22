# Phase 174: Spend Ledger - Research

**Researched:** 2026-08-13
**Domain:** Token measurement persistence and reporting across three worker-dispatch platforms (Claude Code, OpenCode, Codex CLI) inside a Go CLI runtime
**Confidence:** MEDIUM-HIGH (the parsing/arithmetic core is already solved and tested; the wrapper-path measurement source was the open research question and now has a directly-verified answer, but its exact undocumented semantics carry residual risk — see Assumptions Log)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Dollars vs tokens (owner-decided via reshape approval)**
- **D-01: Dollars never headline, anywhere.** The report and the end-of-build
  line lead with tokens. A dollars figure on a subscription plan invites the
  wrong conclusion ("this cost $3" when it cost nothing).
- **D-02: No model-price table is ever built.** Computing USD for rows the
  provider didn't price is API-billing machinery that rots and serves nobody
  here. USD is stored only when the provider itself reported it
  (`pkg/codex/usage.go` already parses `USDCost` in that case) and may appear
  only as a skippable footnote labelled as hypothetical API price.

**Where the operator meets the number (review finding, owner-approved)**
- **D-03: The primary surface is one plain-English line at the end of every
  build/continue** — "This phase used ~N tokens (measured)" or "(partly
  estimated)". A non-technical operator does not run inspection commands;
  without this line the ledger is a dashboard nobody reads (SPEND-08).
- **D-04: `aether spend` is the detail view**, per-worker tokens and tool
  calls, byte-identical on repeated runs (inspection mutates nothing — this
  repo shipped dry-run commands that mutated state for months; the test earns
  its keep).
- **D-05: Workers appear under their colony identity** (caste emoji + name,
  e.g. "🔨 Builder Mason-67"), consistent with Phase 173's non-technical
  readability requirement (its D-14).

**Measurement honesty on the wrapper path (review-flagged gap, locked)**
- **D-06: A token figure relayed by the orchestrating LLM through the results
  JSON is an assertion, not a measurement.** The wrapper-path figure must
  either come from a genuine artifact (e.g., a per-session transcript the
  runtime reads) or be stored tagged as non-provider-grade. It must never
  carry the `provider` source tag. Research should determine what genuine
  local sources exist; if none, the tagged-assertion route is acceptable and
  the tag must surface in the report.

**Roll-up scope (trimmed by reshape)**
- **D-07: Parent attribution per row + a no-double-count invariant, not the
  dual-column accountant view.** Every ledger row records its parent (linkage
  `spawn-log` already stores); the grand total equals the sum of worker rows,
  asserted as an invariant. The full self/subtree two-column table is deferred
  until worker delegation actually exists (Phase 177 grants it).

### Claude's Discretion
- Ledger file format, location, and retention (suggest: per-run files kept
  modestly, e.g. current colony's runs; prune with existing data-clean paths).
- How the end-of-build line handles a mixed measured/estimated run (the
  "(partly estimated)" wording is a suggestion, not a mandate).
- Whether tool-call counts appear in the end-of-build line or only in
  `aether spend` (suggest: only in the detail view — keep the line short).

### Deferred Ideas (OUT OF SCOPE)
- Self/subtree dual-column spend view — revisit after Phase 177 (Worker
  Delegation Grant) makes real delegation trees common.
- Spend thresholds/alerts ("warn me at N tokens") — new capability, not in
  any SPEND requirement; belongs in a future phase if ever.
- `2026-08-01-finalize-reconcile-task-evidence-gate.md` — evidence-gate
  bookkeeping, unrelated to spend measurement (reviewed todo, not folded in).
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — timeout
  configurability, unrelated to spend measurement (reviewed todo, not folded in).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SPEND-01 | Worker token usage persists beyond the process and is readable after a run | `pkg/codex/usage.go` already computes `WorkerUsage` correctly for the direct/Codex-CLI dispatch path, but it dies at the process boundary: `mapInternalWorkerResult` (`cmd/internal_worker_adapter.go:474`) does not copy `codex.WorkerResult.Usage` into the durable `internalWorkerResult`, and `codexExternalBuildWorkerResult` (`cmd/codex_build_finalize.go:61`) has no usage field either. A durable ledger file/store must be added; the existing per-attempt journal (`.aether/data/build/phase-N/attempts/attempt-*.completion.json`) and `SpawnTree`'s per-run boundary (`pkg/agent/spawn_tree.go`) are the two existing durable substrates to build on |
| SPEND-02 | The wrapper path reports token usage | Directly researched (D-06). Two genuine local artifacts confirmed by direct inspection in this environment: Claude Code's session JSONL transcript (`~/.claude/projects/<encoded-cwd>/<session>.jsonl`, path delivered via the already-registered `PreToolUse` hook's `transcript_path` field, confirmed in `173-HOOK-FINDINGS.md`) and OpenCode's local session storage (`~/.local/share/opencode/storage/{project,session,message}/`). See Architecture Patterns and Code Examples below |
| SPEND-03 | A total includes cache-read and cache-creation tokens | Already solved and regression-tested for the direct dispatch path: `pkg/codex/usage.go`'s `billedTotal()` and `pkg/codex/usage_test.go`'s `TestClaudeUsageTotalIncludesCacheReadAndCreation` assert the exact documented 50/100,000/2,000/500 → 102,550 figures. This phase extends the *plumbing*, not the arithmetic. The wrapper-path artifacts have different granularity per platform — OpenCode's session storage already reports disjoint columns (`input`/`output`/`reasoning`/`cache.read`/`cache.write`); Claude Code's transcript `<usage>` tag on `Agent`/`Task` tool results is a single aggregate `subagent_tokens` figure with no disjoint breakdown observed — see Assumptions Log |
| SPEND-04 | A run with no provider figure appears tagged as an estimate, never readable as measurement | `WorkerUsage.Source`/`Measured()` (`pkg/codex/usage.go:39`) already implements this two-way split for the direct path. This phase must decide where session-transcript-sourced rows sit relative to that split (see Architecture Patterns — likely a third source value, distinct from both `provider` and `estimate`, tagged as measured-but-non-provider per D-06) and build the ledger's measured/estimated subtotal split plus the `--include-estimates` gate from scratch — no existing code computes a derived per-phase metric today |
| SPEND-05 | Nested child spend rolls up to its parent; no double counting | `pkg/agent/spawn_tree.go`'s `SpawnEntry.ParentName` (Phase 173) is the exact linkage D-07 requires. A ledger row keyed by worker name can join to its `ParentName` for a display-only roll-up; the grand-total invariant is a straightforward sum-of-rows property test, the same style as `TestQueenOrchestratePreservesSafetyCastes`-class invariant tests already used in this repo |
| SPEND-06 | `aether spend` reports per-worker tokens and tool calls, mutating nothing; USD only as a labelled provider footnote | No `aether spend` (or similarly named) command exists anywhere in the repo today — confirmed by search. `cmd/consolidation_dryrun_test.go`'s `assertConsolidationDryRunIsPure` (hash a watched-file snapshot before/after, assert byte-identical) is the directly reusable idempotency-test pattern CLAUDE.md's Definition of Done requires for an inspection command. **A pre-existing, un-wired violation of the "no model-price table" principle already exists** in `pkg/trace/cost.go` (a hardcoded USD-per-1K-token rate table for stale model names, called from `pkg/agent/pool.go:192`) — flagged as a Pitfall/Open Question below, not silently folded into scope |
| SPEND-07 | No spend figure derives from a character budget; docs stop calling it a "Token Budget" | `colonyPrimeBudgetChars = 8000` (`cmd/colony_prime_context.go:21`) is already correctly named in code (chars, not tokens) — the mislabelling is in documentation. `grep` found 9 files calling it a "Token Budget": `CLAUDE.md`, `CHANGELOG.md`, `AGENTS.md`, `.aether/QUEEN.md`, `.aether/docs/context-continuity.md`, `.aether/docs/ci-context-assembly-design.md`, `.aether/docs/PARITY_CLASSIC_VS_GO.md`, and two skill files. Each needs individual review at plan time — not all necessarily describe this same budget |
| SPEND-08 | One plain-English end-of-build/continue line sourced from the ledger | `aether ceremony closeout` (`cmd/ceremony_cmd.go:214`, `renderCeremonyCloseoutVisual` at line 531) is a **single shared function** invoked by both `build.md` and `continue.md` (`--workflow build` / `--workflow continue`) — one code change surfaces the line on both paths at once |

## Project Constraints (from CLAUDE.md)

Directives extracted from this repo's `CLAUDE.md` that bear directly on how
Phase 174 must be planned and built:

- **Definition of Done:** "A requirement is satisfied only when a command
  exists that someone can run, and that command fails when the requirement is
  unmet." Not a commit, not "covered by existing tests," not a checked box in
  a summary. Every SPEND requirement's plan must name the exact failing
  command (see Validation Architecture below) — CLAUDE.md cites this exact
  ledger's own history (parsed-but-never-persisted usage, an undercount test
  that restated its own parser) as the reason this rule exists.
- **Dry-run corollary:** "An inspection or `--dry-run` command must not
  mutate state." Directly governs `aether spend` (D-04/SPEND-06) — reuse the
  `assertConsolidationDryRunIsPure` test pattern (`cmd/consolidation_dryrun_test.go`),
  the precedent CLAUDE.md names for exactly this rule.
- **Documentation corollary:** "A documentation claim about runtime behaviour
  must be testable or removed." Governs SPEND-07's CLAUDE.md doc rename and
  any new claim this phase's plan adds about what `aether spend` or the
  end-of-build line report.
- **Proportion/invariant tests preferred over "section exists" tests.**
  Directly shapes SPEND-05's grand-total invariant and SPEND-04's
  measured/estimated split — assert the property, not that a field is present.
- **Communication Style / plain-English requirement:** every user-facing
  surface this phase adds (`aether spend` output, the end-of-build line) must
  translate jargon and lead with what a figure means, not just the number —
  consistent with D-01/D-03's "tokens, not dollars, headline" framing already
  locked in CONTEXT.md.
- **UX Architecture ownership model:** "State mutations: Go runtime. Wrapper
  markdown: presentation only, must not mutate state or duplicate
  verification/gating logic." Directly bears on Pattern 1 (Architecture
  Patterns) — the wrapper (`build.md`/`continue.md`) must not be the one
  computing or asserting a token figure; only the Go runtime may read the
  artifact and own the number.
- **Protected paths:** `.aether/data/` may only be written outside its four
  sanctioned scratch subpaths (`planning/`, `phase-research/`, `survey/`,
  `worker-debug/`) by the Go runtime itself, never by a worker's ordinary
  Write tool. If the plan adds a new ledger directory under `.aether/data/`,
  it needs an explicit addition to `sanctionedDataWritePrefixes`
  (`cmd/hook_cmds.go:395`) if any wrapper-side write ever targets it — see
  Standard Stack's "Alternatives Considered" for why deriving from the
  existing attempt journal avoids this entirely.
- **No automatic model selection** (2026-07-28 decision, reaffirmed in this
  phase's own CONTEXT.md specifics): the spend ledger informs the operator
  and Phase 179's before/after proof — it must never trigger or suggest a
  model switch.
- **Verification commands:** `go test ./...`, `go test ./... -race`,
  `go vet ./...`, `go build ./cmd/aether` are the standard gates this phase's
  plan should route through — no new toolchain or test framework needed (see
  Environment Availability / Validation Architecture).

</phase_requirements>

## Summary

The arithmetic problem this phase exists to prevent (the 186x cache-token undercount) is **already fixed and regression-tested** in `pkg/codex/usage.go`. What is missing is plumbing: that correct arithmetic runs today only on the path where Go itself spawns the provider CLI as a subprocess and can read its raw stdout (the Codex-CLI platform, and — via `RealInvoker`/`AttachWorkerUsage` — any dispatch that goes through `pkg/codex`'s internal invoker). It never runs on the path an operator actually uses day to day: Claude Code or OpenCode's own `Agent`/`Task` tool, which executes *inside the orchestrating LLM's own process* and never gives the Go binary a raw stdout stream to parse. That is the "wrapper path" `codexExternalBuildWorkerResult` (`cmd/codex_build_finalize.go:61`) represents, and it has no usage field at all.

The phase's central research question — whether a genuine, non-LLM-asserted measurement source exists for that wrapper path — has a **confirmed yes**, verified by directly reading real local files in this development environment rather than relying on documentation or training knowledge:

- **Claude Code** writes a per-session JSONL transcript to disk (`~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`). When the orchestrating LLM uses its `Agent` (Task) tool to dispatch a subagent, the *platform itself* — not the model — appends a `tool_result` (or, for background/async agents, a `<task-notification>`) to that transcript carrying a real `<usage>` block: `subagent_tokens`, `tool_uses`, `duration_ms`. This is populated by the Claude Code CLI's own harness, not generated by the model narrating its own behavior. The exact file path for the *current* session is delivered to the Go binary today, for free, via the `PreToolUse` hook already registered for the `Agent|Task` matcher — `173-HOOK-FINDINGS.md` (Phase 173's own research) captured the real payload and confirmed a `transcript_path` field is present on every fire.
- **OpenCode** writes an equivalent local session store (`~/.local/share/opencode/storage/{project,session,message}/`), and its per-message records carry a fully disjoint token breakdown (`total`/`input`/`output`/`reasoning`/`cache.read`/`cache.write`) — richer than Claude Code's aggregate figure. Subagent sessions are recorded as child sessions (`parentID` field) whose `title` already embeds the dispatching worker's colony identity in real observed data (`"Builder Brick-11: Login flow test (@general subagent)"`), because OpenCode's own `build.md` sets the Task-tool description to the same `{caste emoji} {Caste} {name}: {task}` convention Claude Code's wrapper uses. OpenCode has no `PreToolUse`-equivalent hook (confirmed in `.opencode/OPENCODE.md`), so its session must be discovered by matching the project's `worktree` path and a time window, not delivered directly.
- **Codex CLI** dispatches workers as a genuine Go-spawned subprocess (`codex exec`, per `pkg/codex/platform_contract.go`'s documented `worker_dispatch` mechanism), so it already goes through the solved `pkg/codex/usage.go` parser — its gap is purely the SPEND-01 persistence gap, not a measurement-source gap.

Because both wrapper-path artifacts are populated by the platform's own harness rather than composed by the model, they should not be dismissed as "LLM assertions" — but their exact accounting semantics (does Claude Code's `subagent_tokens` already include cache reads correctly summed, the way `billedTotal()` does? is the tag stable across CLI versions?) are **undocumented**. Anthropic's own GitHub tracker shows a *related* feature request (aggregate sub-agent view in `/context`) was explicitly closed as "not planned" — the raw per-dispatch numbers already exist in the transcript regardless, and this phase does not need Anthropic to ship anything; it needs a parser. Treat wrapper-path figures as measured-but-not-provider-grade (their own new `Source` tag, never `"provider"`), per D-06.

**Primary recommendation:** Extend `WorkerUsage` with a third source tier for platform-harness-recorded (non-API) measurements; have the Go runtime independently read and parse the session artifact itself (never let the orchestrating LLM transcribe a number into a CLI flag — that reintroduces exactly the assertion problem D-06 warns against); wire the existing `PreToolUse` hook to persist `transcript_path` once per run; correlate transcript entries to Aether workers via the `{caste emoji} {Caste} {name}: {task}` description text `build.md` already sets; and land the one-line summary in the single shared `renderCeremonyCloseoutVisual` function so both build and continue get it from one change.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Token arithmetic (disjoint cache columns, billed total) | API / Backend (`pkg/codex`) | — | Already solved, provider-shape-aware, regression-tested. Extend, never re-derive |
| Wrapper-path usage capture (Claude Code) | API / Backend (Go runtime, via hook-delivered `transcript_path`) | Browser/Platform-harness (Claude Code CLI itself writes the artifact) | The Go binary must read the artifact itself; the orchestrating LLM must not be trusted to relay the number |
| Wrapper-path usage capture (OpenCode) | API / Backend (Go runtime, via local storage discovery) | Platform-harness (OpenCode CLI writes the artifact) | No hook channel exists on this platform — discovery is a Go-side heuristic over `~/.local/share/opencode/storage/` |
| Ledger persistence | Database / Storage (new `.aether/data/` substrate, or reuse of the existing attempt journal) | — | Must survive process exit (SPEND-01); Claude's Discretion on exact file shape |
| Parent/roll-up linkage | Database / Storage (`pkg/agent/spawn_tree.go`, already built in Phase 173) | — | `ParentName` is already recorded at spawn time; this phase reads it, does not re-derive it |
| `aether spend` detail view | API / Backend (new Cobra subcommand) | — | Read-only inspection command; must mutate nothing (D-04) |
| End-of-build/continue plain-English line | API / Backend (`cmd/ceremony_cmd.go`, `renderCeremonyCloseoutVisual`) | Frontend Server-equivalent (wrapper markdown just renders what Go returns) | Single existing function shared by both workflows — do not duplicate in `build.md` and `continue.md` prose |
| USD footnote (provider-reported only) | API / Backend (`pkg/codex/usage.go`'s existing `USDCost` field) | — | Never compute; only relay what the provider itself reported (D-02) |

## Standard Stack

### Core

No new external dependencies are required or recommended. This phase is entirely Go standard library plus extension of existing internal packages.

| Package | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib (go 1.26.5) | Parse session transcript JSONL lines, OpenCode session/message JSON files, and the ledger's own persisted format | Already the parsing tool used throughout `pkg/codex/usage.go` and the rest of the codebase — no reason to diverge |
| `bufio` | stdlib | Line-scan potentially large `.jsonl` transcript files without loading the whole file into memory | Claude Code transcripts observed in this environment range from a few KB to 600+ KB; `bufio.Scanner`/line-by-line reading avoids loading megabytes-large sessions wholesale |
| `path/filepath` | stdlib | Resolve and validate `transcript_path` / OpenCode storage paths safely (see Security Domain — path validation matters here) | Consistent with existing path-handling in `cmd/codex_build_finalize.go`'s claim-path validation |

### Supporting (existing internal packages to extend, not replace)

| Package | Purpose | When to Use |
|---------|---------|-------------|
| `pkg/codex` (`usage.go`, `worker.go`, `platform_dispatch.go`) | `WorkerUsage` struct, `ParseUsage`, `billedTotal`, `EstimateUsage`, `AttachWorkerUsage`, `Source` tagging | Extend `Source` with a new non-provider-measured tier; do not touch `ParseUsage`'s arithmetic (it is correct and tested) |
| `pkg/agent` (`spawn_tree.go`) | `SpawnTree`, `SpawnEntry.ParentName`, `SpawnRun` (`BeginRun`/`EndRun`/`CurrentRun`) | Source of the parent linkage (D-07) and the natural "current run" boundary SPEND-06's "for the current run" and SPEND-08's per-phase line both need |
| `pkg/storage` (`Store`, file locking) | Durable JSON persistence with existing locking discipline | Reuse for whatever ledger file(s) this phase adds — do not hand-roll file locking |
| `cmd/hook_cmds.go` (`claudeHookInput`) | Already has `SessionID` (Phase 173); needs a `TranscriptPath string `json:"transcript_path"`` field added | The hook already fires on every `Agent|Task` dispatch (`.claude/settings.json` matcher `"Agent|Task"`) — this is the delivery channel for SPEND-02, already half-wired |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reading the real session transcript | A tokenizer to estimate tokens from prompt/response text | Explicitly out of scope per `REQUIREMENTS.md`'s "Out of Scope" table: "A tokenizer dependency — Both providers report exact counts including cache reads." Do not introduce one |
| A hand-rolled model-price table for the USD footnote | Only ever relay `WorkerUsage.USDCost` when the provider itself set it | D-02 is explicit and absolute; a price table already exists elsewhere in this codebase (`pkg/trace/cost.go`) and is stale — do not repeat that pattern here even by omission |
| New sanctioned `.aether/data/spend/` directory for the ledger | Deriving `aether spend` directly from the existing per-attempt completion journal (`.aether/data/build/phase-N/attempts/attempt-*.completion.json`) | The existing journal is already Go-owned, already durable, and already schema-versioned via the completion packet. A brand-new directory needs an addition to `sanctionedDataWritePrefixes` (`cmd/hook_cmds.go:395`) if any wrapper-side writes ever touch it; deriving from the existing journal avoids that entirely. This is flagged as Claude's Discretion, not mandated |

**Installation:** None — no `go.mod` changes expected.

**Version verification:** N/A (no new packages). Runtime environment verified directly in this session: `go1.26.5 darwin/arm64`, Claude Code CLI `2.1.231`, OpenCode CLI `1.1.63`, Codex CLI `codex-cli 0.146.0` (all three platform binaries are installed and available in this development environment — see Environment Availability).

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────┐
                         │   Wrapper markdown (build.md/continue.md) │
                         │   spawns workers via the platform's own   │
                         │   Agent/Task tool, description =          │
                         │   "{emoji} {Caste} {name}: {task}"        │
                         └───────────────┬───────────────────────────┘
                                         │
        ┌────────────────────────────────┼─────────────────────────────────┐
        │                                │                                 │
        ▼                                ▼                                 ▼
┌───────────────┐            ┌────────────────────┐             ┌──────────────────────┐
│  Codex CLI     │            │  Claude Code        │             │  OpenCode             │
│  platform:     │            │  wrapper path:       │             │  wrapper path:        │
│  Go spawns     │            │  Agent runs INSIDE   │             │  Task runs INSIDE     │
│  `codex exec`  │            │  Claude's own        │             │  OpenCode's own       │
│  as subprocess │            │  process — Go never   │             │  process — Go never   │
│  → raw stdout  │            │  sees raw stdout      │             │  sees raw stdout      │
│  captured      │            │                       │             │                       │
└───────┬────────┘            └──────────┬────────────┘             └──────────┬────────────┘
        │                                │                                    │
        ▼                                ▼                                    ▼
┌────────────────┐        ┌───────────────────────────────┐   ┌───────────────────────────────┐
│ pkg/codex       │        │ PLATFORM HARNESS writes usage  │   │ PLATFORM HARNESS writes usage  │
│ ParseUsage()     │        │ to disk itself (NOT the model):│   │ to disk itself (NOT the model):│
│ billedTotal()    │        │ ~/.claude/projects/<cwd>/      │   │ ~/.local/share/opencode/       │
│ Source=provider  │        │  <session>.jsonl                │   │  storage/{project,session,     │
│ (SOLVED, tested) │        │ tool_result <usage> block on    │   │  message}/                     │
└───────┬──────────┘        │ each Agent/Task dispatch;       │   │ per-message disjoint `tokens`  │
        │                    │ path delivered via the already- │   │ block; child session `title`   │
        │                    │ registered PreToolUse hook       │   │ embeds worker identity         │
        │                    │ (transcript_path field,          │   │ (no hook — discover via         │
        │                    │ confirmed 173-HOOK-FINDINGS.md)  │   │ project worktree + time window)│
        │                    └──────────────┬───────────────────┘   └──────────────┬────────────────┘
        │                                   │                                      │
        │                                   ▼                                      ▼
        │                    ┌───────────────────────────────────────────────────────────────┐
        │                    │ Go runtime independently reads and parses the artifact itself   │
        │                    │ (never trusts the orchestrating LLM to relay the number) —       │
        │                    │ correlates by tool_use_id / session title to worker identity     │
        │                    │ using the "{emoji} {Caste} {name}: {task}" convention             │
        │                    └───────────────────────────────┬───────────────────────────────────┘
        │                                                    │
        └────────────────────────────────┬───────────────────┘
                                          ▼
                         ┌─────────────────────────────────────┐
                         │  Ledger: one row per worker dispatch  │
                         │  { WorkerName, Caste, ParentName,     │
                         │    Usage{..., Source: provider |      │
                         │    session-transcript | estimate},    │
                         │    ToolCount, RunID }                 │
                         │  measured/estimated kept as SEPARATE  │
                         │  subtotals (SPEND-04); grand total =  │
                         │  sum of rows, invariant (SPEND-05)    │
                         └───────────────┬───────────────────────┘
                                         │
                ┌─────────────────────────┴─────────────────────────┐
                ▼                                                   ▼
   ┌─────────────────────────┐                     ┌─────────────────────────────────┐
   │ `aether spend` (SPEND-06)│                     │ ceremony closeout (SPEND-08)      │
   │ detail view, per-worker, │                     │ ONE shared function used by both  │
   │ mutates nothing (D-04)   │                     │ build.md and continue.md — one    │
   │                          │                     │ plain-English line, no dollars     │
   └──────────────────────────┘                     └─────────────────────────────────┘
```

### Recommended Project Structure

No new top-level directories are required. Suggested file additions follow existing naming conventions observed in `cmd/` and `pkg/codex/`:

```
pkg/codex/
├── usage.go                    # EXTEND: add a third Source tier (existing file)
└── usage_test.go               # EXTEND: existing regression tests stay; add new ones

cmd/
├── spend_ledger.go              # NEW: ledger read/write, measured/estimated split, roll-up invariant
├── spend_ledger_test.go         # NEW
├── spend_cmd.go                 # NEW: `aether spend` Cobra command (detail view)
├── spend_cmd_test.go            # NEW: idempotency test (assertConsolidationDryRunIsPure pattern)
├── wrapper_usage_claude.go      # NEW: parse Claude Code session JSONL, correlate by tool_use_id
├── wrapper_usage_claude_test.go # NEW
├── wrapper_usage_opencode.go    # NEW: discover + parse OpenCode session storage
├── wrapper_usage_opencode_test.go # NEW
├── hook_cmds.go                 # EXTEND: add TranscriptPath to claudeHookInput; persist once per run
├── internal_worker_adapter.go   # EXTEND: mapInternalWorkerResult must copy Usage through
├── codex_build_finalize.go      # EXTEND: codexExternalBuildWorkerResult gains an optional usage field
└── ceremony_cmd.go               # EXTEND: renderCeremonyCloseoutVisual gains the one-line summary
```

### Pattern 1: Independent server-side artifact reading (never trust the relay)

**What:** The Go runtime reads the session transcript / OpenCode storage files itself, using a path delivered by a platform-controlled channel (the hook payload), rather than accepting a token count typed into a CLI flag by the orchestrating LLM.

**When to use:** Any time the "genuine artifact" for D-06 is consulted. This is the load-bearing distinction between "measured" and "asserted."

**Example (hook payload, real captured data — `173-HOOK-FINDINGS.md`):**
```json
{"session_id":"55e92197-a80f-40f5-95c7-c59f8aa931b6","transcript_path":"/Users/callumcowie/.claude/projects/-Users-callumcowie-repos-Aether/55e92197-a80f-40f5-95c7-c59f8aa931b6.jsonl","cwd":"/Users/callumcowie/repos/Aether","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Nested dispatch experiment level 1","subagent_type":"general-purpose"},"tool_use_id":"toolu_018MGHrN1TKQUwkXaKYRPr95"}
```

### Pattern 2: Correlating a transcript usage block to an Aether worker

**What:** `build.md`/`continue.md`'s existing spawn instruction already fixes the Task/Agent tool's `description` field to `{caste emoji} {Caste} {name}: {task}` (`.claude/commands/ant/build.md:258,271`). The transcript's `tool_use` block for that dispatch carries that exact string in `tool_input.description`; its matching `tool_result` (same `tool_use_id`, or `<tool-use-id>` inside an async `<task-notification>`) carries the `<usage>` block.

**Example (real transcript content observed in this repository's own session history, foreground shape):**
```
tool_use.input.description: "🔨 Builder Hammer-23: implement the parser"
tool_use.id:                "toolu_018EMHv18BVjT83EF72QirQJ"
    ... later in the file ...
tool_result.tool_use_id:    "toolu_018EMHv18BVjT83EF72QirQJ"
tool_result.content[0].text: "agentId: adae132fff1a68069 ...\n<usage>subagent_tokens: 110790\ntool_uses: 19\nduration_ms: 308218</usage>"
```

**Example (async/background shape — a separate `queue-operation` line, matched by `<tool-use-id>` inside the text, not the JSON `tool_use_id` field):**
```
{"type":"queue-operation","operation":"enqueue","content":"<task-notification>\n<task-id>a18b2dafc5c06d355</task-id>\n<tool-use-id>toolu_01VzCu8Th48dSK3A8AL1wd9s</tool-use-id>\n<status>completed</status>\n...\n<usage><subagent_tokens>59243</subagent_tokens><tool_uses>6</tool_uses><duration_ms>189273</duration_ms></usage>\n</task-notification>"}
```
Note the two shapes differ (plain `key: value\n` lines vs XML-style `<key>value</key>`) — a parser must handle both. `build.md` instructs "Do NOT set `run_in_background`" for the normal build spawn path, so the foreground shape should be the common case for `/ant-build`; the async shape exists for completeness and for any future background-dispatch surface.

**Example (OpenCode, real local session file on this machine, `~/.local/share/opencode/storage/session/<project-hash>/<child-session>.json`):**
```json
{
  "id": "ses_2454cb741ffenUyTGZyyqkIkI0",
  "parentID": "ses_24567d0d5ffepVb1KmxClCf3A1",
  "title": "Builder Brick-11: Login flow test (@general subagent)"
}
```
and its message (`~/.local/share/opencode/storage/message/<child-session>/<msg>.json`):
```json
{
  "role": "assistant",
  "cost": 0,
  "tokens": { "total": 20436, "input": 543, "output": 123, "reasoning": 47, "cache": { "read": 19770, "write": 0 } }
}
```

### Pattern 3: Extending `WorkerUsage.Source` with a third tier

**What:** `pkg/codex/usage.go` currently defines two source values (`"provider"`, `"estimate"`). D-06 forbids tagging a wrapper-path figure as `"provider"`. The cleanest fit is a third constant, e.g. `UsageSourceSessionTranscript = "session-transcript"`, distinct from both — it is a real measurement (platform-harness-recorded, not model-asserted) but not the raw provider API event `ParseUsage` decodes, and its exact accounting is undocumented (see Assumptions Log).

**Consequence:** `Measured()` (currently `Source == UsageSourceProvider`) and any ledger-level "measured vs estimated" split (SPEND-04) must decide whether `session-transcript` counts toward the *measured* subtotal (recommended — it is not a character-count guess) while still never being reported as `provider`-grade. This is a concrete design decision for the plan, not something research can settle unilaterally; flagged here so the planner makes it explicitly rather than by accident.

### Anti-Patterns to Avoid

- **Letting the orchestrating LLM copy a number it read into a `--tokens` CLI flag.** This looks like a measurement but is exactly the assertion D-06 rejects — the Go binary has no way to confirm the LLM read the transcript correctly, or read it at all, versus guessing a plausible number. Read the artifact server-side.
- **Restating the parser's own arithmetic in its test**, the way the original `usage_test.go` bug did (`186x` undercount shipped green because the test asserted `input+output`, the same wrong formula the parser used). Every new test in this phase must assert **literal, externally-sourced figures** (the documented Anthropic example, or a fixture transcript checked into the test with hand-computed expected totals), never re-derive the expected value from the code under test.
- **Building a second model-price table.** One already exists (`pkg/trace/cost.go`) and is stale (hardcoded rates for `claude-sonnet-4-20250514`, `gpt-3.5-turbo`, etc., and — notably — it has the *same undercount shape* as the original 186x bug: `CalculateCost` multiplies only `inputTokens`/`outputTokens`, ignoring cache tokens entirely). Do not add a second one for the spend ledger; if anything, this finding should be surfaced to the owner as a separate cleanup decision (see Open Questions), not silently extended.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token counting from text | A tokenizer | `WorkerUsage`, `ParseUsage` (already exact provider counts) | Explicitly excluded by `REQUIREMENTS.md`; both providers report exact counts including cache reads |
| USD estimation | A per-model price table | `WorkerUsage.USDCost` (only set when the provider itself reported it) | D-02 is absolute; a stale, hand-maintained rate table (`pkg/trace/cost.go`) already demonstrates the failure mode this decision exists to prevent |
| Durable JSON persistence + locking | A bespoke file writer | `pkg/storage.Store` | Already handles the locking discipline every other durable colony file uses |
| Parent/child spawn linkage | A new tree structure | `pkg/agent/spawn_tree.go`'s `SpawnEntry.ParentName` and `SpawnRun` | Built in Phase 173 specifically so this phase would not have to retrofit it |
| Idempotency testing for an inspection command | A new hashing helper | `cmd/consolidation_dryrun_test.go`'s watched-file-snapshot pattern | Already proven, already the precedent CLAUDE.md cites for "an inspection command must not mutate state" |

**Key insight:** every piece of new machinery this phase needs already has a same-shaped precedent elsewhere in the repo (parser, persistence, linkage, idempotency test). The actual work is wiring three already-solved pieces together across three platform-specific artifact sources — not designing new abstractions.

## Common Pitfalls

### Pitfall 1: The 186x-undercount shape recurring
**What goes wrong:** A new total is computed as `input + output`, silently dropping cache tokens, and the accompanying test restates the same formula instead of asserting an externally-sourced figure.
**Why it happens:** It already happened once in this exact file (`pkg/codex/usage.go`'s git history, per its own comments) and has a live second instance today in `pkg/trace/cost.go`'s `CalculateCost`.
**How to avoid:** Every new total this phase computes must route through `billedTotal()`-equivalent logic (or literally call it), and every new test must assert a literal number from outside the code under test.
**Warning signs:** A test whose "expected" value is computed by calling the same function/formula under test, rather than a hardcoded literal.

### Pitfall 2: Treating a wrapper-path figure as provider-grade
**What goes wrong:** A `session-transcript`-sourced row gets tagged `Source: "provider"` because it "came from a real measurement," collapsing D-06's distinction.
**Why it happens:** The temptation is real — Claude Code's harness genuinely does write the number, unlike a pure LLM guess. But its accounting semantics are undocumented (see Assumptions Log), and conflating it with the fully-verified, provider-API-shape-matched `ParseUsage` output erodes the exact trust boundary this phase exists to build.
**How to avoid:** A distinct `Source` constant, never `"provider"`, for anything not parsed directly from a raw provider API event.
**Warning signs:** `usage.Source == UsageSourceProvider` being set anywhere outside `ParseUsage`.

### Pitfall 3: Correlation drift between transcript entries and Aether workers
**What goes wrong:** Two workers dispatched with very similar `description` text (e.g., two Builders on the same phase) get their usage swapped, or a worker's usage is silently dropped because its `tool_use_id` never appears in a completed `tool_result` (e.g., the worker was cancelled).
**Why it happens:** The correlation key is a `tool_use_id`/session-`title` text match, not a first-class identifier Aether controls end to end.
**How to avoid:** Match on `tool_use_id` (exact, not fuzzy) wherever present; fall back to the `{emoji} {Caste} {name}: {task}` description text only as a secondary signal; treat any dispatch whose transcript entry cannot be resolved as needing the `estimate` fallback (never silently drop the row — `EstimateUsage` already exists precisely so "no row" never happens).
**Warning signs:** A ledger with fewer rows than `dispatch_manifest.execution_plan` had dispatches.

### Pitfall 4: OpenCode has no hook — silent staleness in discovery
**What goes wrong:** The Go runtime picks the wrong session (e.g., a stale one from a previous unrelated run in the same repo) because there is no platform-delivered session identifier to anchor on.
**Why it happens:** `.opencode/OPENCODE.md` documents plainly that OpenCode has no `PreToolUse`-equivalent hook registration; the runtime must discover the session by matching `worktree` and a time window, which is inherently heuristic.
**How to avoid:** Bound the discovery to sessions `created`/`updated` within the current build/continue run's own time window (the `SpawnRun.StartedAt`/`EndedAt` boundary already exists for this); if no unambiguous match is found, fall back to the `estimate` tag rather than guessing.
**Warning signs:** A spend figure that doesn't move between two different builds — a sign the same stale session is being read repeatedly.

### Pitfall 5: Character budget still gets called a token budget somewhere downstream
**What goes wrong:** SPEND-07 renames the primary `CLAUDE.md` section but a docs file, skill file, or code comment elsewhere keeps calling `colonyPrimeBudgetChars` a "Token Budget," and a later phase or a user reads it as a real spend figure again.
**Why it happens:** 9 files reference "Token Budget" today; not all are the same concept, and a partial rename leaves the confusion half-fixed.
**How to avoid:** Audit all 9 hits individually at plan time (listed in Phase Requirements above) rather than a single global find-and-replace, since some may correctly describe an actual token concept.
**Warning signs:** `grep -ri "token budget"` still returning hits describing `colonyPrimeBudgetChars` after the phase ships.

## Code Examples

### `WorkerUsage` and the already-solved arithmetic (`pkg/codex/usage.go`)
```go
// Source: /Users/callumcowie/repos/Aether/pkg/codex/usage.go
type WorkerUsage struct {
	InputTokens         int64   `json:"input_tokens,omitempty"`
	CachedInputTokens   int64   `json:"cached_input_tokens,omitempty"`
	CacheCreationTokens int64   `json:"cache_creation_tokens,omitempty"`
	OutputTokens        int64   `json:"output_tokens,omitempty"`
	TotalTokens         int64   `json:"total_tokens,omitempty"`
	USDCost             float64 `json:"usd_cost,omitempty"`
	Model               string  `json:"model,omitempty"`
	Source              string  `json:"source,omitempty"` // "provider", "estimate" today
}

func (u WorkerUsage) billedTotal() int64 {
	return u.InputTokens + u.CachedInputTokens + u.CacheCreationTokens + u.OutputTokens
}
```

### The persistence gap (`cmd/internal_worker_adapter.go:474`)
```go
// mapInternalWorkerResult does NOT copy result.Usage through today — this is
// the concrete SPEND-01 gap for the direct/Codex-CLI dispatch path, where a
// genuine provider measurement already exists in memory and is discarded.
func mapInternalWorkerResult(result codex.WorkerResult, invokeErr error) *internalWorkerResult {
	return &internalWorkerResult{
		Name: result.WorkerName, Caste: result.Caste, /* ... */
		ToolCount: result.ToolCount, // present
		// Usage: result.Usage,     // ABSENT — SPEND-01 fix site
	}
}
```

### The wrapper-path gap (`cmd/codex_build_finalize.go:61`)
```go
type codexExternalBuildWorkerResult struct {
	Stage, Caste, Name, Status, Summary string
	ToolCount     int                 `json:"tool_count,omitempty"` // present
	// Usage field is absent entirely — SPEND-02's target
	Handoff codex.WorkerHandoff `json:"handoff,omitempty"`
}
```

### The hook channel already half-wired (`cmd/hook_cmds.go:16`)
```go
type claudeHookInput struct {
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	AgentID       string `json:"agent_id"`   // Phase 173 addition
	AgentType     string `json:"agent_type"` // Phase 173 addition
	SessionID     string `json:"session_id"` // Phase 173 addition
	// TranscriptPath string `json:"transcript_path"` // NOT YET ADDED — SPEND-02 needs this
}
```

### The idempotency test precedent (`cmd/consolidation_dryrun_test.go:81`)
```go
func assertConsolidationDryRunIsPure(t *testing.T, command string) {
	// ... snapshot watched files, hash before, run command, hash after,
	// assert byte-identical. Directly reusable pattern for aether spend's
	// D-04 "mutates nothing" requirement.
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| No usage measurement anywhere; every budget in the codebase counted characters of assembled context (~6k tokens vs ~117k a real worker spends) | `pkg/codex/usage.go` parses real provider usage events on the direct dispatch path, with correct disjoint-cache arithmetic | Existing prior work (predates this phase — comment in `usage.go` itself documents the fix) | The 95% of real spend the framework doesn't compose is now measurable on ONE of three platform paths; this phase closes the other two |
| `usage_test.go`'s original assertion restated the parser's own (wrong) formula | Regression tests assert the literal documented provider figures (50/100,000/2,000/500 → 102,550) | Same prior fix | Establishes the testing discipline this phase's new tests must follow |
| Claude Code sub-agent token visibility requested as a first-class `/context` feature | Anthropic closed that specific feature request as "not planned" — but the raw numbers are already written to the local transcript by the harness, independent of whether Anthropic ever builds a UI for them | GitHub issue anthropics/claude-code#10164, status checked 2026-08-13 | This phase does not depend on Anthropic shipping anything; it needs to read a file that already exists |

**Deprecated/outdated:**
- `pkg/trace/cost.go`'s hardcoded per-model USD rate table: rates are pinned to old model identifiers (`claude-sonnet-4-20250514`, `gpt-3.5-turbo`, etc.) and the arithmetic ignores cache tokens entirely — both the naming and the undercount pattern are stale relative to what this phase establishes as correct. Not part of this phase's scope to fix, but the contradiction with D-02 should be surfaced to the owner (see Open Questions).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Claude Code's transcript `<usage>` block (`subagent_tokens`) already correctly sums disjoint cache/input/output the way `billedTotal()` does, rather than under- or double-counting cache tokens | Architecture Patterns, Pattern 3; SPEND-03 requirement support | If wrong, the wrapper-path total could itself be undercounted (or overcounted) in a way indistinguishable from a correct measurement without cross-checking against a real provider event — the exact failure mode this phase exists to prevent, just one layer removed. Mitigation: tag it with its own `Source` (never `provider`), so a future audit can revisit it without having silently promoted it to full trust |
| A2 | The `<usage>` transcript tag's format and presence are stable across Claude Code CLI versions and are not an internal/private implementation detail that could disappear or change shape without notice | Architecture Patterns, Pattern 2 | If Anthropic changes or removes this format, wrapper-path Claude Code measurement silently falls back to nothing being found — mitigated only if the parser fails soft into the `estimate` tag rather than erroring. This is undocumented behavior (not found in any official Claude Code documentation search), observed only by directly reading real local session files in this environment |
| A3 | OpenCode's session storage layout (`~/.local/share/opencode/storage/{project,session,message}/`, `worktree` field, `parentID`, per-message `tokens` block) is stable across OpenCode versions and is the officially intended way to read this data (vs. an internal implementation detail) | Architecture Patterns; Standard Stack | If the on-disk format changes, OpenCode wrapper-path measurement breaks silently; same estimate-fallback mitigation applies |
| A4 | The `{caste emoji} {Caste} {name}: {task}` description-text convention `build.md`/`continue.md` already set (pre-existing, not part of this phase) is a reliable-enough correlation key when combined with `tool_use_id` exact-matching | Common Pitfalls, Pitfall 3 | If two simultaneous workers produce identical description text (unlikely given deterministic per-worker names, but not structurally impossible), correlation could misattribute usage between them |
| A5 | Adding a `TranscriptPath` field to `claudeHookInput` and persisting it once per run is sufficient, and no `PostToolUse` hook is needed, because by the time `build-finalize` runs the transcript file already contains all completed workers' `<usage>` blocks | Pattern 1; SPEND-02 requirement support | If a worker's tool_result hasn't flushed to disk by finalize time (a timing/buffering assumption), that worker's row would need the `estimate` fallback rather than silently vanishing — same general estimate-fallback mitigation covers this |

**If this table is empty:** N/A — see rows above. All five assumptions carry the same class of mitigation (fail soft to the existing `estimate` tag, never silently drop a row, never promote an unverified figure to the `provider` tag), which the ledger's SPEND-04 measured/estimated split already structurally requires regardless of whether these assumptions hold.

## Open Questions

> **All three resolved at planning time (2026-08-14). Each carries an inline resolution note
> naming the plan that settles it. Nothing below is still open for Phase 174.**

1. **What should the plan do about `pkg/trace/cost.go`'s pre-existing model-price table?**
   - What we know: It exists, is called from `pkg/agent/pool.go:192` (a direct-streaming LLM client path using `pkg/llm`), has stale model rates, and repeats the input+output-only undercount pattern. `pkg/llm` and `agent.NewPool`-style construction were not found to have any live caller under `cmd/` in this search, suggesting it is likely unreachable from any of the three dispatch platforms this phase covers (Claude Code, OpenCode, Codex CLI) — but this was not exhaustively proven.
   - What's unclear: Whether it is genuinely dead code (in which case it's out of this phase's scope and a candidate for a separate cleanup decision) or reachable via some command not surfaced by this search.
   - Recommendation: Flag to the owner as a residue item rather than silently including or silently ignoring it. It directly contradicts D-02's "no model-price table is ever built," even though it predates this phase.
   - **RESOLVED (planning, 2026-08-14): flagged to the owner as residue, OUT OF SCOPE for Phase 174.** No plan touches `pkg/trace/cost.go` or `pkg/agent/pool.go`. The finding is recorded durably in two places so it cannot evaporate with this document: `.planning/STATE.md` § "Blockers/Concerns", and plan 174-09's required SUMMARY content. Phase 174 instead prevents a *second* instance — plan 174-03 and plan 174-07 both carry acceptance-criteria greps that fail if a price table, a per-1K rate or any token-to-dollar arithmetic appears in the new spend surface (D-02). Deciding the fate of the existing one needs an owner ruling (delete as dead code, or fix its cache-ignoring arithmetic), which is a decision, not a build task.

2. **Does a `PreToolUse` hook fire reliably enough, on every build, to guarantee `transcript_path` is captured before `build-finalize` needs it?**
   - What we know: `.claude/settings.json` already registers the hook for matcher `"Agent|Task"`, and 173-HOOK-FINDINGS.md confirmed it fires on real nested dispatches, at both the coordinator's own dispatch and a subagent's own further dispatch.
   - What's unclear: Whether there's any build path where zero `Agent`/`Task` dispatches occur before finalize (e.g., a phase with zero dispatches, or a degenerate single-worker case) — if so, no hook fire means no captured `transcript_path`, and the ledger needs a graceful "wrapper session not identified" fallback rather than an error.
   - Recommendation: Design the capture-then-read path to degrade to the `estimate` tag (or an explicit "unknown session" note) rather than fail the finalize step.
   - **RESOLVED (planning, 2026-08-14): the recommendation is adopted in full, and the fail-soft behaviour is tested rather than assumed.** Plan 174-02 makes `recordSpendSessionFromHook` return no value and swallow every error, asserted by `TestSpendSessionCaptureFailureDoesNotAffectHookDecision`; `claudeTranscriptUsageForCurrentSession` returns `ok == false` when no session record exists. Plan 174-06's resolver treats every wrapper-path source as optional and falls through to `codex.EstimateUsage`, with `TestSpendResolverNeverDropsADispatch` asserting the row count still equals the dispatch count across six source-availability permutations including "none available". A build with zero `Agent`/`Task` dispatches therefore produces a full-length, honestly-labelled estimate ledger and no error.

3. **Should the OpenCode wrapper-path parser be built in this phase, or is the Claude Code path (the platform this repo's operator actually uses day to day, per CLAUDE.md's Platform Policy: "Primary platforms: Claude Code and OpenCode. These are the main maintained user surfaces.") sufficient for a first landing, with OpenCode following once the pattern is proven?**
   - What we know: Both are declared primary platforms; OpenCode's data is actually richer (disjoint columns) than Claude Code's aggregate figure, but its discovery mechanism is a heuristic with no hook to anchor on.
   - What's unclear: Relative operator usage split between the two platforms in practice, and whether the phase's time budget favors landing one well-tested path first.
   - Recommendation: Plan-time decision; both are within this phase's stated requirements (SPEND-02 doesn't name a platform), but the risk profile differs enough to be worth an explicit sequencing call rather than parallel half-done work on both.
   - **RESOLVED (planning, 2026-08-14): build both, in the same wave, as two independent plans — 174-04 (Claude Code) and 174-05 (OpenCode).** CLAUDE.md's Platform Policy names both as maintained user surfaces, so shipping measurement for one would make every later efficiency claim silently platform-dependent. The "half-done work" risk the question raises is answered structurally rather than by sequencing: the two live in separate files with separate committed fixtures and share nothing but the `codex.WorkerUsage` type, so neither can leave the other half-built, and plan 174-06's resolver consults each independently and falls through to an estimate when either is unavailable. OpenCode's weaker discovery story (no hook to anchor on) is handled inside 174-05 by bounding to the project worktree plus the run's time window and refusing an ambiguous match outright.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All of this phase's implementation | ✓ | go1.26.5 darwin/arm64 | — |
| Claude Code CLI | Wrapper-path research/testing (SPEND-02) | ✓ | 2.1.231 | — |
| OpenCode CLI | Wrapper-path research/testing (SPEND-02) | ✓ | 1.1.63 | — |
| Codex CLI | Direct-dispatch path (already solved; SPEND-01 persistence only) | ✓ | codex-cli 0.146.0 | — |
| `~/.claude/projects/` session transcripts | Reading real Claude Code session data to build/test the parser | ✓ | — (39+ real session files with `<usage>` blocks found in this repo's own project directory) | — |
| `~/.local/share/opencode/storage/` | Reading real OpenCode session data to build/test the parser | ✓ | — (real project/session/message directories present on this machine) | — |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None — all three platform CLIs and their local data stores are present and readable in this development environment, which is unusually favorable for writing tests against real (not synthetic) artifact shapes. Plan-time tests should still use committed fixture files (redacted/trimmed real samples) rather than depending on this machine's live home directory, since CI will not have it.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (`go test`) |
| Config file | none — standard `go.mod`-driven test discovery |
| Quick run command | `go test ./pkg/codex/... ./cmd/... -run Spend` |
| Full suite command | `go test ./... -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SPEND-01 | Usage survives past process exit, readable after a run | unit | `go test ./cmd/... -run TestSpendLedgerPersistsAcrossProcesses -v` | ❌ Wave 0 |
| SPEND-02 | Wrapper-path worker produces a non-empty, non-`provider`-tagged usage row | unit (fixture transcript) | `go test ./cmd/... -run TestWrapperPathUsageFromClaudeTranscript -v` | ❌ Wave 0 |
| SPEND-03 | Anthropic's documented 50/100000/2000/500 → 102,550 example, literal assertion | unit (already exists for direct path) | `go test ./pkg/codex/... -run TestClaudeUsageTotalIncludesCacheReadAndCreation -v` | ✅ (existing, direct path only) |
| SPEND-04 | Mixed ledger renders separate measured/estimated subtotals; derived metric refused without `--include-estimates` | unit | `go test ./cmd/... -run TestLedgerMeasuredEstimatedSubtotalsAreSeparate -v` | ❌ Wave 0 |
| SPEND-05 | Grand total equals sum of individual worker rows (invariant); parent roll-up derivable | unit (property-style) | `go test ./cmd/... -run TestLedgerGrandTotalEqualsSumOfRows -v` | ❌ Wave 0 |
| SPEND-06 | `aether spend` reports per-worker tokens/tool-calls; byte-identical on repeat run | integration (CLI, dry-run-style purity test) | `go test ./cmd/... -run TestSpendCommandIsIdempotent -v` | ❌ Wave 0 |
| SPEND-07 | No spend figure derives from `colonyPrimeBudgetChars`; docs no longer call it a token budget | unit + doc-lint | `go test ./cmd/... -run TestSpendFigureNeverDerivesFromCharBudget -v` then `grep -ri "token budget" CLAUDE.md` (manual doc audit, 9 files) | ❌ Wave 0 |
| SPEND-08 | Closeout renders one plain-English token line for both build and continue | unit | `go test ./cmd/... -run TestCeremonyCloseoutIncludesTokenLine -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./pkg/codex/... ./cmd/... -run Spend`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/spend_ledger_test.go` — covers SPEND-01, SPEND-04, SPEND-05
- [ ] `cmd/spend_cmd_test.go` — covers SPEND-06 (idempotency, reusing `assertConsolidationDryRunIsPure`'s pattern)
- [ ] `cmd/wrapper_usage_claude_test.go` — covers SPEND-02, needs a committed fixture `.jsonl` (trimmed/redacted real transcript shape, both foreground `tool_result` and async `<task-notification>` forms)
- [ ] `cmd/wrapper_usage_opencode_test.go` — covers SPEND-02 (OpenCode), needs committed fixture `project.json`/`session.json`/`message.json` files
- [ ] `cmd/ceremony_cmd_test.go` addition — covers SPEND-08
- [ ] No new test framework install needed — `go test` is already fully configured for this repo

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | This phase touches no auth surface |
| V3 Session Management | no (Aether "session.json" is a colony concept, not an auth session) | — |
| V4 Access Control | no | No new privilege boundary introduced |
| V5 Input Validation | **yes** | Every new parser (Claude transcript JSONL, OpenCode storage JSON) must treat its input as untrusted: reject malformed JSON lines without crashing (mirror `ParseUsage`'s existing `continue`-on-decode-error loop), and never construct a file path to read from unvalidated string concatenation |
| V6 Cryptography | no | No new cryptographic material |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via a spoofed `transcript_path` value (a hook payload is platform-generated and normally trustworthy, but should not be blindly trusted as a file-read target without bounds) | Tampering | Validate the resolved path lies under the expected `~/.claude/projects/` root (mirror the existing `validateAndNormalizeClaimPathToRoot`-style boundary checks already used elsewhere in `cmd/codex_build_finalize.go` for claim paths) before opening it |
| Resource exhaustion from an unbounded transcript file read | Denial of Service | Stream with `bufio.Scanner`/bounded line length rather than reading the whole file into memory; the existing `internalWorkerRequestMaxBytes = 2 << 20` pattern in `cmd/internal_worker_adapter.go` is a precedent for bounding untrusted input size |
| A worker's task text containing content that could be mis-parsed as a usage block if description-matching is naive (e.g., a task brief that happens to contain the literal string `<usage>`) | Tampering (data confusion, not injection — no code execution risk) | Match on `tool_use_id` first (structural, not text-based); only fall back to description-text matching as a secondary signal, and only within content Go itself parsed as a `tool_result`/`<task-notification>` block, never raw substring search across the whole file |

## Sources

### Primary (HIGH confidence — direct repository/filesystem inspection in this session)
- `/Users/callumcowie/repos/Aether/pkg/codex/usage.go`, `usage_test.go` — solved arithmetic and regression test
- `/Users/callumcowie/repos/Aether/cmd/codex_build_finalize.go`, `internal_worker_adapter.go`, `build_worker_run.go` — the two persistence gaps (SPEND-01, SPEND-02)
- `/Users/callumcowie/repos/Aether/cmd/hook_cmds.go` — existing hook infrastructure, `claudeHookInput`
- `/Users/callumcowie/repos/Aether/.planning/phases/173-delegation-guard/173-HOOK-FINDINGS.md` — Phase 173's own empirical hook payload capture, confirming `transcript_path`
- `/Users/callumcowie/repos/Aether/pkg/agent/spawn_tree.go` — parent linkage, run boundaries
- `/Users/callumcowie/repos/Aether/pkg/trace/cost.go`, `pkg/agent/pool.go` — the pre-existing model-price-table finding
- `/Users/callumcowie/repos/Aether/cmd/ceremony_cmd.go` — shared closeout function for SPEND-08
- `/Users/callumcowie/repos/Aether/cmd/consolidation_dryrun_test.go` — idempotency test precedent
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md` — worker description convention (`{caste emoji} {Caste} {name}: {task}`)
- `/Users/callumcowie/repos/Aether/.opencode/OPENCODE.md` — confirms no `PreToolUse`-equivalent hook on OpenCode
- Real local session transcript files directly read in this session: `~/.claude/projects/-Users-callumcowie-repos-Aether/*.jsonl` (39 files containing `<usage>` blocks) and `~/.local/share/opencode/storage/{project,session,message}/` (real project/session/message records with disjoint token fields)
- `~/.aether/CLAUDE.md` and `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `174-CONTEXT.md` — phase scope and locked decisions

### Secondary (MEDIUM confidence — WebSearch verified against official/authoritative sources)
- [Anthropic prompt caching documentation](https://platform.claude.com/docs/en/build-with-claude/prompt-caching) (via search summary) — confirms `total_input_tokens = cache_read_input_tokens + cache_creation_input_tokens + input_tokens`, cross-verifying `billedTotal()`'s arithmetic independent of this repo's own test
- [OpenCode session storage / token usage](https://ccusage.com/guide/opencode/), [OpenCode session management (DeepWiki)](https://deepwiki.com/sst/opencode/2.1-session-management) — corroborate the local storage location and subagent session nesting model observed directly on disk

### Tertiary (LOW confidence — single source, or explains absence rather than presence)
- [GitHub anthropics/claude-code#10164](https://github.com/anthropics/claude-code/issues/10164) — "Show sub-agent token usage in /context command and Task tool output," closed as not planned. Relevant only to explain that this phase should not wait on or depend on Anthropic building a UI for this; the raw data was found to already exist independent of that feature request. The exact `<usage>` block format and its accounting semantics were **not found documented anywhere** — status is empirical observation only (see Assumptions Log A1, A2)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, pure extension of existing, well-understood internal packages
- Architecture (direct/Codex-CLI path): HIGH — fully solved and tested already
- Architecture (Claude Code wrapper path): MEDIUM-HIGH — genuine artifact confirmed by direct, repeated observation of real local files in this environment, but its exact semantics are undocumented by the vendor
- Architecture (OpenCode wrapper path): MEDIUM — genuine artifact confirmed by direct observation, richer data than Claude Code's, but discovery has no hook-delivered anchor and relies on a heuristic
- Pitfalls: HIGH — sourced from this repo's own documented history (the 186x bug, the dry-run-mutation bug) plus a freshly-found live instance of the same undercount pattern in `pkg/trace/cost.go`
- Security: MEDIUM — no exotic threat surface, standard input-validation discipline already precedented elsewhere in this codebase

**Research date:** 2026-08-13
**Valid until:** 30 days for the internal-package findings (stable, slow-moving); 14 days for the undocumented Claude Code/OpenCode transcript format specifics (Assumptions A1-A3) — re-verify against the installed CLI versions if planning is delayed, since these are unversioned, unofficial surfaces that could change without a changelog entry
