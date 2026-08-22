# Phase 191: Dead Wood - Context

**Gathered:** 2026-08-21
**Status:** Ready for planning
**Source:** Direct codebase reconnaissance by the planner (no `/gsd-discuss-phase` session exists for
this phase — owner reachable but favouring momentum). Every decision below was reached by reading and
tracing the live source (Go call graphs, wrapper markdown, test files, `testdata/orphan_allowlist.json`),
not by trusting the ROADMAP's or orchestrator's characterization at face value. Two of the fourteen
decisions below **correct** the ROADMAP/orchestrator brief rather than execute it literally — both are
marked explicitly, with the evidence that forced the correction. Per the planning brief's instruction,
grey areas are auto-decided conservatively: **when in doubt, KEEP a file and log it.**

<domain>
## Phase Boundary

This phase closes five independent gaps in Phase 191's ROADMAP entry, each confirmed by reading (not
assuming) the current code, the current wrapper markdown, the current test suite, and the current
`testdata/orphan_allowlist.json` — not just by trusting what a prior phase's text says exists.

### Criterion 1 — zero-reader colony/ configs

`colony/` (82 git-tracked files, confirmed via `git ls-files colony/`) was built in Phase 154
("Colony Assets") to serve `control-ts/`, a TypeScript control plane that Phase 160 confirmed dead and
deleted (RETIRE-01..04). `colony/` itself was never deleted alongside it. Confirmed via repo-wide grep
(excluding `.planning/` and the stale `.claude/worktrees/agent-a*/` copies the orchestrator's brief
correctly flagged as out of scope):

- `colony/agents/*.yaml` (27 files) — **zero readers**, any language, any surface.
- `colony/phases/*.yaml` (9 files) — **zero readers**, any language, any surface.
- `colony/policies/model-routing.yaml` — **zero readers**. No `policyPath("model-routing")` call site
  exists anywhere in `cmd/` or `pkg/`.
- `colony/policies/autopilot.yaml` — **zero readers** (the one "autopilot" string hit in
  `cmd/run_visuals.go:171` is an unrelated struct field value `Source: "autopilot"`, not a file read).
- `colony/policies/memory-rules.yaml` — **zero readers**.

`colony/` is git-tracked (not gitignored) and is **not shipped by `aether publish` or `aether install`**
— `cmd/publish_cmd.go` and `cmd/install_cmd.go` contain zero references to `colony`. This confirms
STATE.md's own prior note: nothing embeds, publishes, or installs `colony/` to `~/.aether/system/colony`.
Deletion is a normal, visible git operation with no distribution-side cleanup required (D-13).

**Corrected count vs. the orchestrator's scouting note:** `colony/policies/` contains 10 files, not 5.
Of the other 5 not named by ROADMAP criterion 1 (`dispatch-contract.yaml`, `oracle-phase-directives.yaml`,
`pheromone-lifecycle.yaml`, `review-depth.yaml`, `safety-gates.yaml`, `signal-rules.yaml`,
`skill-creation.yaml` — that's actually 7 more, 10 total minus the 3 named): `dispatch-contract.yaml`
and `review-depth.yaml` have real readers (criterion 2, below); `oracle-phase-directives.yaml` is
explicitly protected (criterion 5); **`pheromone-lifecycle.yaml`, `safety-gates.yaml`,
`signal-rules.yaml`, and `skill-creation.yaml` are ALSO confirmed zero-reader** (same defect class as
the three ROADMAP names) but are **not named in ROADMAP criterion 1's literal enumerated list**. See
D-02 — kept, not deleted, logged as a finding for a future ruling.

Similarly, `colony/prompts/` holds 28 files (27 per-agent prompts + `colony-prime.md`). Only
`colony-prime.md` has a reader (criterion 2). The other 27 (`ambassador.md` … `weaver.md`) are
**confirmed zero-reader** by the same grep sweep — not named by criterion 1 either. See D-02.

`colony/playbooks/*.md` (7 files) has readers only inside two **test** files
(`cmd/command_call_audit_test.go`, `cmd/subcommand_reachability_ratchet_test.go`) — not confirmed as a
production runtime dependency or a pure test-fixture path reference. Not investigated further (would
require tracing what those tests actually assert against the path, beyond this phase's context budget)
and not named by criterion 1. See D-02.

### Criterion 2 — the four CWD-relative silent-fallback loaders

Identified by reading every consumer of `colony/`, not by guessing from filenames. All four share the
exact shape: a package-level `sync.Once`-cached loader resolves a path via `filepath.Join("colony", ...)`
(CWD-relative, no HUB fallback), reads it with `os.ReadFile`, and on **any** error (missing, unreadable,
unparseable) silently leaves the cached value `nil` — every consumer then falls back to a hardcoded Go
default. This CWD-relative resolution only ever succeeds when the compiled/`go run` binary's working
directory is this repo's root (a **dev checkout**, i.e. a maintainer running `aether`/`go run ./cmd/aether`
from inside this repo) — an installed/published Aether run from any other project's directory always
hits the fallback (confirmed: `colony/` is not published, D-13). Existing test coverage for all four
loaders already uses an explicit path-override variable (`colonyPrimeTemplatesPathOverride`,
`visualsPathOverride`, `dispatchContractPathOverride`, `reviewDepthPathOverride`) — meaning `go test`
never exercises the real `colony/` file at all (Go test working directory is the package dir, `cmd/`, so
the bare relative path `colony/...` would not resolve there even if a test didn't override it). The
**only** environment where deleting these files changes observable behaviour is a maintainer's own dev
checkout — which is exactly what "dev-checkout ceremony/visual output byte-identical before and after"
in the ROADMAP text is naming.

The four loaders, precisely:

| # | Loader func | File | Reads | Fallback values live in |
|---|---|---|---|---|
| 1 | `loadColonyPrimeTemplates()` | `cmd/prompt_template_loader.go:63-84` | `colony/prompts/colony-prime.md` | Inline string literals scattered across `cmd/colony_prime_context.go` at each `writeSectionHeader(...)` / `fmtOrFallback(...)` / `sectionString(...)` call site (e.g. line 263 `"## Prior Reviews\n\n"`, line 415 `"State: %s\n"`, line 454-460 the three review-depth text fallbacks, line 526 `charterFallbackHeading`). **Not centralized** — confirmed by reading; `codex_dispatch_contract.go` also calls these accessors for its own sections, sharing the same underlying template mechanism (see the file-overlap note in `<code_context>`). |
| 2 | `loadVisualsConfig()` | `cmd/visuals_config.go:33-49` | `colony/ceremony/visuals.md` | `casteColorMap`, `casteEmojiMap`, `casteLabelMap` and related maps hardcoded in `cmd/codex_visuals.go` (per CLAUDE.md's own "Caste Identity System" section, which already names this file as the source of truth) |
| 3 | `loadDispatchContractPolicy()` | `cmd/codex_dispatch_contract.go:54-66` | `colony/policies/dispatch-contract.yaml` | Centralized Go constants in the same file: `fallbackSurveyExecutionModel`, `fallbackPlanningExecutionModel`, `fallbackSurveyDeadlinePolicy`, and 7 siblings (lines 69-86) |
| 4 | `loadReviewDepthPolicy()` | `cmd/review_depth.go:41-53` | `colony/policies/review-depth.yaml` | Centralized Go slices in the same file: `heavyKeywordsFallback`, `securityRiskKeywordsFallback`, `blastRadiusKeywordsFallback` (lines 56-71) |

**Not one of the four:** `colony/policies/oracle-phase-directives.yaml`'s reader
(`cmd/oracle_loop.go:1666-1687`) uses a **different, three-path search** (repo-relative, then
`root`-relative, then a **HUB** fallback path) — not a bare CWD-relative-or-silently-fail resolution.
Criterion 5 explicitly protects this file; its different shape is additional confirmation it was never
meant to be one of "the four." Untouched (D-05).

### Criterion 3 — dead code

**`pkg/trace/cost.go` + the unconstructed pool path.** `trace.CalculateCost`'s only call site anywhere
in the repo is `pkg/agent/pool.go`'s `poolStreamHandler.OnComplete` (~line 187). `pkg/agent.NewPool` —
the only way any code could ever obtain a `*Pool` to drive that handler — has **zero call sites in
`cmd/`** (confirmed: `grep -rn "NewPool(" cmd/` finds nothing). Its only callers anywhere are
`pkg/agent/pool_test.go` (same-package test, expected — testing an internal API is not evidence of a
production caller) and the stale `.claude/worktrees/agent-a*/` copies the orchestrator's brief already
ruled out of scope. This confirms the ROADMAP's "unconstructed pool path" precisely: nothing in any
`cmd/` entry point ever constructs a `Pool`, so `OnComplete` (and therefore `trace.CalculateCost`) can
never run in production. D-06 scopes the fix narrowly: delete `cost.go` outright, remove only the
specific dead cost-tracking block inside `OnComplete` (required so `pool.go` still compiles once
`cost.go` is gone) — **not** the rest of `pool.go`/`Pool`/`NewPool`, which is a much larger, unscoped
question (whether the entire `pkg/agent` package is dead) that ROADMAP criterion 3 does not ask and this
phase's budget does not cover.

**`session-verify-fresh` — ROADMAP TEXT IS STALE, DO NOT DELETE.** This is the phase's most important
correction. `session-verify-fresh` has a real, live, current caller: `.claude/commands/ant/medic.md:28`,
`.claude/commands/ant-medic.md:28`, and `.opencode/commands/ant/medic.md:28` all instruct
`aether session-verify-fresh --command oracle` as part of the `/ant-medic` diagnostic wrapper (a real,
auto-discovered slash command by virtue of its file location under `.claude/commands/ant/` — no
separate "is this wired" question applies to a slash command file itself). `.aether/commands/medic.yaml`
(the YAML source) and `.aether/skills/colony/context-management/SKILL.md:33` both also instruct its use.
`CHANGELOG.md:191` documents this wiring directly: *"The session-freshness tools (`session-verify-fresh`,
`session-clear`) — documented for years, called by nothing — are now reachable from the medic wrapper
and off the orphan allowlist."* The ROADMAP's characterization of this as dead code predates that
wiring, or the roadmap author did not check the medic wrapper specifically. Per this phase's own prime
rule ("verify by execution, not grep" — and its inverse, verify a *deletion* target the same way),
deleting `session-verify-fresh` would **break `/ant-medic`** on both platforms. D-07: **KEEP**, not
touched, and the ROADMAP's own text is now corrected by this record rather than executed literally.

**`newLearningValidator`** (`cmd/helpers.go:198-202`) — confirmed **zero callers anywhere** in the repo
(the only two matches for the identifier are its own doc comment and its own function signature).
Genuinely dead. D-08: delete.

**Unreferenced skill-lifecycle commands (SKILL-01).** The orchestrator's brief did not enumerate which 9
commands this refers to, and the planner's first hypothesis (the CRUD-authoring commands in
`cmd/skill_lifecycle.go`/`cmd/skill_curator.go`) was **wrong** — corrected by reading
`cmd/subcommand_reachability_ratchet_test.go`'s own `skillLifecycleOrphanCandidates` map (the literal,
already-reviewed, already-tested "8 skill entries" Phase 178's success criterion measured, remapped to
SKILL-01/Phase 191 by REQUIREMENTS.md's traceability table) and `172-CONTEXT.md:203-204`, which names
the set explicitly. The true 8 are:

`skill-index`, `skill-detect`, `skill-match`, `skill-inject`, `skill-list`, `skill-diff`,
`skill-parse-frontmatter`, `skill-cache-rebuild` — all in `cmd/skills.go`.

**The CRUD-authoring commands (`skill-create`, `skill-patch`, `skill-archive`, `skill-pin`,
`skill-list-lifecycle`, `skill-promote`, `skill-view`, `skill-curator-run`, `skill-recover`) are a
DIFFERENT subsystem**, per the reachability test's own comment at line 1204-1207: *"skill_lifecycle.go's
unrelated authoring commands ... are a different subsystem and correctly fall through to
'unreviewed-pre-existing'."* Of these nine, `skill-create` has a real, extensively documented caller
(`/ant-skill-create`, a full wrapper triplet plus a Codex direct-CLI path, listed in CLAUDE.md's
Advanced commands table and `.aether/commands/skill-create.yaml`). The other eight
(`skill-patch`/`skill-archive`/`skill-pin`/`skill-list-lifecycle`/`skill-promote`/`skill-view`/
`skill-curator-run`/`skill-recover`) are **also** unreferenced, by the same evidence standard, but are
**not** SKILL-01's reviewed set and are **out of scope** for this phase (D-14) — conservative default
applies; logged, not touched.

**The critical nuance for the true 8 (D-09):** the CLI *subcommand* (`aether skill-match` as a
standalone, typeable/scriptable shell verb) has no caller — but the underlying Go **function**
`matchSkillsForWorkflow` (`cmd/skills.go:767`, called by both `skill-match`'s and `skill-inject`'s
`RunE` closures) has a live, load-bearing, **in-process** caller: `cmd/codex_build.go:3268`, inside
`composeBuildManifestBrief` — the function that assembles every worker's brief, including its skill
section. This is the exact skill-injection pipeline CLAUDE.md's "Skill Injection" section documents
("Own 8K character budget... Injected into builder and watcher prompts"). **The ruling is CLI-surface
deletion, not logic deletion**: remove the 8 standalone `cobra.Command` registrations; preserve
`matchSkillsForWorkflow`, `resolveSkillMatchInput`, `renderSkillInjectResult` and any other helper still
reachable after the CLI wrappers are gone. `buildFullIndex` (called by 3 of the 8: `skill-index`,
`skill-list`, `skill-cache-rebuild`) and `loadSkillIndexOrBuild`/`skillWorkspaceMatchReasons` (called by
`skill-detect`) have **no other confirmed callers** as of this read — the executing task must re-check
after the 8 CLI wrappers are removed (a function only becomes genuinely dead once its *last* caller is
gone) and delete only what is then provably unreachable, never anything `matchSkillsForWorkflow` still
needs.

Deleting the 8 commands also requires updating, in the same change:
- `cmd/testdata/orphan_allowlist.json` — remove the (up to 8) entries whose `name` matches
  `aether skill-index`, `aether skill-detect`, `aether skill-match`, `aether skill-inject`,
  `aether skill-list`, `aether skill-diff`, `aether skill-parse-frontmatter`,
  `aether skill-cache-rebuild`. A deleted command is not an "allowed orphan" — it does not exist to be
  an orphan of anything.
- `cmd/subcommand_reachability_ratchet_test.go`'s D-08 block — `skillLifecycleOrphanCandidates`,
  `resolveSkillLifecyclePaths` (which `t.Fatalf`s if a leaf name no longer resolves via `rootCmd.Find`
  — this WILL fire the moment the 8 commands are deleted unless this block is updated in the same
  change), and the `len(phase178) != 6` assertion. Phase 178 no longer exists as a phase; SKILL-01's
  ruling was delivered by Phase 191. This block must be rewritten to record that ruling (closed by
  deletion) rather than continuing to assert a "reaches zero" measurement against commands that no
  longer exist to be measured.

### Criterion 4 — false docs

**`.aether/workers.md:825`**: *"Build work produces observations (via `memory-capture` in continue
step)."* Confirmed problematic: `memory-capture` (a real, separately-registered command,
`cmd/learning.go:213`, distinct from `learning-observe`) is **not mentioned anywhere** in
`.claude/commands/ant/build.md` or `.claude/commands/ant/continue.md` — neither wrapper's prose
instructs calling it, contradicting the "via `memory-capture` in continue step" claim as a description
of wrapper-visible behaviour. Phase 148's WORKFLOW-01 evidence confirms learnings genuinely are
extracted during continue — but the live mechanism is `pkg/learn`'s own automatic pipeline (STATE.md:
*"a second, different learning system (`pkg/learn`) already runs live on every continue"*), not a
wrapper-visible `memory-capture` call. The precise, correct replacement sentence requires the executing
task to trace `pkg/learn`'s actual trigger point (this was not fully resolved during planning — see
D-10) and state it accurately rather than repeat an unverified claim.

**CLAUDE.md's documented trim order** (the "Token Budget" section, `CLAUDE.md:628-636`) lists 9 items
with a single "QUEEN.md wisdom" entry. The real implementation, `cmd/context.go:1027-1038`
(`trimOrder := []string{...}`), has **9 distinct trim-order entries with QUEEN wisdom split into two
separately-prioritized tiers** — `"--- QUEEN WISDOM (Global) ---"` then `"--- QUEEN WISDOM (Local) ---"`
— plus blockers never trimmed (10 total conceptual tiers). CLAUDE.md's list conflates the two QUEEN
tiers into one, understating the real granularity by one rank. D-11: correct to a 10-item list matching
the code exactly.

**CLAUDE.md's host-build claim** (`CLAUDE.md:445-455`, the "Command Playbooks (Reference Material)"
section): *"execution behavior lives in the host-manifest flow (`aether host build` → spawn from
`dispatch_manifest` → `build-finalize`)"* and *"`.claude/commands/ant/build.md` and `continue.md`
describe the host-manifest flow directly."* Confirmed false for the primary, interactive path:
`.claude/commands/ant/build.md:348` **explicitly forbids** `aether host build` from the wrapper ("the TS
host hop is off the interactive build path") and instead documents (lines 21, 105-108, 295) a
**different** flow: `aether build $ARGUMENTS --plan-only` (manifest) → wrapper-driven worker spawn →
`aether build-finalize` (Go-owned completion). `aether host build` → `dispatch_manifest` is the
**autopilot** lane only (`aether run`, per `build.md:348`'s own framing). CLAUDE.md's text presents the
autopilot-only mechanism as if it were what the primary wrapper describes. D-12: correct to name both
paths distinctly — the interactive wrapper's real flow, and the separate autopilot/host-driven flow —
rather than conflating them.

### Criterion 5 — explicitly protected, untouched

`colony/policies/oracle-phase-directives.yaml` and all auxiliary commands (`/ant-oracle`, `/ant-dream`,
chaos, archaeology, swarm, council) are out of scope for every deletion in this phase. No task in this
phase's plans may modify `colony/policies/oracle-phase-directives.yaml`, `cmd/oracle_loop.go`, or any
auxiliary command's wrapper/agent files. A final verification plan smoke-tests `/ant-oracle` and
`/ant-dream` specifically, after every other plan's deletions have landed, to prove nothing in this
phase's blast radius touched them.

</domain>

<decisions>
## Implementation Decisions

### Criterion 1 — zero-reader configs

- **D-01 (auto-decided):** Delete exactly the ROADMAP's literal enumerated list:
  `colony/agents/*.yaml` (27), `colony/phases/*.yaml` (9), `colony/policies/model-routing.yaml`,
  `colony/policies/autopilot.yaml`, `colony/policies/memory-rules.yaml`. Each confirmed zero-reader
  across Go, TS, wrapper markdown, skills, and the publish manifest (D-13). A simple existence-check
  ratchet (per the prime rule's own guidance: "for pure file-absence ratchets a simple existence check
  that FAILS when the file returns is honest and sufficient; don't over-engineer") proves reappearance
  is caught. This delivers ROSTER-01 and ROSTER-02 in full — both name exactly this set.

- **D-02 (auto-decided — conservative, logged, NOT executed this phase):** Four additional zero-reader
  file groups were discovered during the same read pass, matching the identical defect class, but are
  **not named by ROADMAP criterion 1's literal text**:
  1. `colony/policies/pheromone-lifecycle.yaml`, `safety-gates.yaml`, `signal-rules.yaml`,
     `skill-creation.yaml` (4 files) — confirmed zero-reader by the same `policyPath(name)`-call-site
     sweep that confirmed D-01's three.
  2. `colony/prompts/*.md` except `colony-prime.md` (27 files: `ambassador.md` … `weaver.md`) —
     confirmed zero-reader; `prompt_template_loader.go` only ever resolves the single fixed path
     `colony/prompts/colony-prime.md`, never any per-agent file.
  3. `colony/playbooks/*.md` (7 files) — readers found only inside two test files, not confirmed as a
     production dependency; not fully traced (out of this phase's budget).
  Per the planning brief's explicit instruction ("when in doubt, KEEP a file and log it, deletion can
  come later"), **none of these are touched this phase.** This finding is carried into the phase's
  return to the orchestrator as a discovered-but-out-of-scope item, not silently dropped and not
  silently expanded into scope.

### Criterion 2 — the four loaders

- **D-03 (auto-decided):** The four loaders are precisely: `loadColonyPrimeTemplates()` /
  `colony/prompts/colony-prime.md`, `loadVisualsConfig()` / `colony/ceremony/visuals.md`,
  `loadDispatchContractPolicy()` / `colony/policies/dispatch-contract.yaml`, `loadReviewDepthPolicy()` /
  `colony/policies/review-depth.yaml` — identified by reading every consumer, not guessed from
  filenames (see `<domain>` table). `colony/policies/oracle-phase-directives.yaml`'s reader uses a
  structurally different, HUB-aware three-path search and is excluded (criterion 5 protects it anyway).

- **D-04 (auto-decided — the fold-then-delete process, applied identically to all four):**
  1. **Baseline-first** (a dedicated task/step, before any edit): run the real compiled `aether` binary
     from this repo's root (dev-checkout CWD) against the CLI surface each loader feeds, and capture the
     output verbatim. This is the only environment where the loader's real-file branch is reachable at
     all (Go's own test suite cannot exercise it — package tests run with CWD set to the package
     directory, not the repo root, which is exactly why all four loaders already have a
     `*PathOverride` test seam).
  2. For each field the colony/ file defines, diff its current value against the Go-compiled default
     (the scattered inline fallbacks in `colony_prime_context.go` for loader 1; the maps in
     `codex_visuals.go` for loader 2; the named constants/slices in `codex_dispatch_contract.go` and
     `review_depth.go` for loaders 3-4).
  3. **Fold** any divergent value into the Go-compiled default (edit the Go literal to match what the
     file currently said) — this is what makes step 5's byte-identical proof possible when a file and
     its compiled default were not already in agreement.
  4. Delete the colony/ file. Add a reappearance ratchet (simple existence check).
  5. Re-run the same CLI surface from step 1; diff byte-for-byte against the step-1 capture. Must be
     identical. Record the diff (or its absence) in the plan's SUMMARY.
  Loader-function scaffolding (`loadColonyPrimeTemplates`, `loadVisualsConfig`, etc.) is **kept**, not
  deleted — it becomes a permanently-fallback path (the file it could read no longer exists), which is
  lower risk than gutting `sync.Once`/struct-parsing plumbing for marginal benefit, and is not what
  criterion 2's literal text asks for ("files deleted," not "loader functions deleted"). Logged as
  discretion, not silently assumed.

- **D-05 (auto-decided):** `colony/policies/oracle-phase-directives.yaml` is not one of the four (see
  `<domain>`) and criterion 5 protects it explicitly regardless. No plan in this phase touches
  `cmd/oracle_loop.go` or this file.

- **File-overlap note (governs plan grouping, not a numbered decision):** `cmd/codex_dispatch_contract.go`
  is a **dual-purpose file** — it hosts both `loadDispatchContractPolicy()` (loader 3) AND several calls
  into the *colony-prime* template-accessor functions (`writeSectionHeader`/`fmtOrFallback`/
  `sectionString`, shared with loader 1's `colony_prime_context.go`). Because loaders 1 and 3 may share
  this one file, they are planned together in the same plan (sequential tasks — safe within one plan
  regardless of the shared file) rather than split across parallel plans, where an accidental overlap
  would violate the no-same-wave-file-overlap rule.

### Criterion 3 — dead code

- **D-06 (auto-decided — scope-limited):** Delete `pkg/trace/cost.go` outright (its only caller,
  `poolStreamHandler.OnComplete`, is unreachable — `agent.NewPool` has zero production callers). Remove
  only the specific `trace.CalculateCost`/`h.tracer.LogTokenUsage` block inside `OnComplete` (required
  for `pool.go` to still compile). Do **not** delete the rest of `pool.go`, `Pool`, `NewPool`, or
  `Registry` — whether the entire `pkg/agent` package is dead is a materially larger, unscoped question
  the ROADMAP text does not ask ("pkg/trace/cost.go + unconstructed pool path" names the cost-tracking
  relationship specifically, not the whole package) and this phase's budget does not cover.
  `pkg/trace/trace.go`'s `LogTokenUsage` itself is left untouched (a different file, not named for
  deletion, and this task's removal of pool.go's one call site is the only edit needed for compilation).

- **D-07 (auto-decided — CORRECTS THE ROADMAP TEXT, does not execute it literally):**
  `session-verify-fresh` is **NOT deleted**. It has a confirmed live caller in the medic wrapper triplet
  (`.claude/commands/ant/medic.md`, `.claude/commands/ant-medic.md`, `.opencode/commands/ant/medic.md`,
  all line 28) and `.aether/skills/colony/context-management/SKILL.md:33`, and `CHANGELOG.md:191`
  documents this wiring as already having happened. Deleting it would break `/ant-medic` on both
  Claude Code and OpenCode. This is recorded as a correction, not silently skipped — see the phase's
  coverage table below and the orchestrator return.

- **D-08 (auto-decided):** `newLearningValidator` (`cmd/helpers.go:198-202`) has zero callers anywhere
  in the repo, including tests. Delete outright.

- **D-09 (auto-decided — corrects the planner's own first hypothesis, recorded honestly):** SKILL-01's
  "8 of 9" refers to `skill-index`, `skill-detect`, `skill-match`, `skill-inject`, `skill-list`,
  `skill-diff`, `skill-parse-frontmatter`, `skill-cache-rebuild` (`cmd/skills.go`) — confirmed via
  `cmd/subcommand_reachability_ratchet_test.go`'s own `skillLifecycleOrphanCandidates` map and
  `172-CONTEXT.md:203-204`. Delete the 8 standalone `cobra.Command` CLI wrappers only. **Preserve**
  `matchSkillsForWorkflow`, `resolveSkillMatchInput`, and `renderSkillInjectResult` unconditionally —
  `cmd/codex_build.go:3268` calls `matchSkillsForWorkflow` directly, in-process, as the live
  skill-injection-into-worker-briefs mechanism (`composeBuildManifestBrief`). This is a CLI-surface
  ruling, not a logic deletion — get this distinction wrong and the build's skill-injection feature
  breaks. After removing the 8 CLI wrappers, re-check `buildFullIndex` / `loadSkillIndexOrBuild` /
  `skillWorkspaceMatchReasons` for remaining callers (none confirmed as of this read) and delete only
  what the compiler and a full `go test ./cmd/... -count=1` confirm is genuinely, transitively
  unreachable — never anything `matchSkillsForWorkflow` still needs.

- **D-14 (auto-decided — scope boundary):** The 8 CRUD-authoring skill-lifecycle commands
  (`skill-patch`, `skill-archive`, `skill-pin`, `skill-list-lifecycle`, `skill-promote`, `skill-view`,
  `skill-curator-run`, `skill-recover`) are a different subsystem per the reachability test's own
  comment, are not SKILL-01's reviewed set, and are **not touched this phase** — conservative default,
  logged as a finding.

### Criterion 4 — false docs

- **D-10 (auto-decided — direction given, precise wording left to execution):** `.aether/workers.md:825`
  must stop claiming observations are captured "via `memory-capture` in continue step" as a description
  of wrapper-visible behaviour, since neither `build.md` nor `continue.md` mention `memory-capture` at
  all. The executing task must trace `pkg/learn`'s actual automatic-capture trigger point (confirmed
  live per Phase 148's WORKFLOW-01 evidence and STATE.md's "a second... learning system already runs
  live on every continue") and state the real mechanism, or — if `memory-capture` genuinely is meant as
  a worker-invocable command for a worker to call directly during its own task (a live possibility not
  ruled out by this planning pass) — correct "in continue step" to the accurate timing instead. Either
  way, the corrected sentence must be traceable to a real call site, not restated unverified.

- **D-11 (auto-decided):** CLAUDE.md's "Trim order" list (`CLAUDE.md:628-636`) is corrected to match
  `cmd/context.go:1027-1038` exactly: split the single "QUEEN.md wisdom" entry into "QUEEN wisdom
  (global)" then "QUEEN wisdom (local)" as two separately-prioritized tiers, in that trim order,
  producing a corrected 9-tier trim order plus blockers-never-trimmed (10 lines total).

- **D-12 (auto-decided):** CLAUDE.md's "Command Playbooks (Reference Material)" section
  (`CLAUDE.md:445-455`) is corrected to name **two** distinct flows instead of conflating them: the
  interactive wrapper's real path (`aether build $ARGUMENTS --plan-only` → wrapper-driven worker spawn
  → `aether build-finalize`) and the separate autopilot/host-driven path (`aether host build` →
  `dispatch_manifest` → `build-finalize`, reached only via `aether run` or another host-driven
  invocation, never the interactive wrapper — `build.md:348` explicitly forbids it there). The
  "Authority note" bullet claiming `build.md`/`continue.md` "describe the host-manifest flow directly"
  is corrected to say they describe the plan-only/wrapper-driven flow; `continue.md`'s relationship to
  `aether host continue --dry-run` (a real, but dry-run-only, host touch point per 190-CONTEXT.md's own
  citation) may be described accurately alongside it if the executing task confirms the exact current
  wording is otherwise correct.

### Criterion 5 — protected

- **D-05 (see above):** `colony/policies/oracle-phase-directives.yaml` untouched.
- No plan in this phase modifies `cmd/oracle_loop.go`, any auxiliary command's wrapper markdown, or any
  auxiliary agent definition. A final verification plan (Wave 2, depends on all Wave-1 plans) smoke-tests
  `/ant-oracle` and `/ant-dream` after every deletion has landed.

### Claude's Discretion

- Exact wording of every ratchet test's failure message, provided it names the specific file/command
  and fails loudly (never silently) when the deleted item reappears.
- Exact new test function names, provided each is a genuine reappearance/regression check (Definition
  of Done), not a named-string-exists check.
- Whether the CLI-surface baseline capture for criterion 2's four loaders is scripted as a `bash` block
  inside the task, a small throwaway Go `main`, or an existing subcommand invocation (`aether build
  --print-brief --full` or equivalent) — any is acceptable provided it exercises the real
  CWD-relative-resolution branch and the diff is byte-for-byte.
- Precise final form of the D-08 test-block rewrite in `subcommand_reachability_ratchet_test.go` (delete
  the block entirely vs. rewrite it as a closed-ruling record) — either satisfies "the allowlist and
  ratchet stay internally consistent with commands that no longer exist."

### Folded Todos

Not applicable. This CONTEXT.md was authored directly by the planner because no `/gsd-discuss-phase`
session exists for this phase (owner reachable, favouring momentum) — there is no todo backlog to score
against it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase governance
- `.planning/ROADMAP.md` § "Phase 191" (line ~973) — the goal and five success criteria this phase is
  judged against; `depends_on: nothing`
- `.planning/ROADMAP.md` § "Phase 192" (line ~978) — read to confirm no overlap (confirmed: none;
  Phase 192 is the final benchmark showdown, depends on 191 completing, touches no files this phase
  touches)
- `.planning/REQUIREMENTS.md` lines 68-71, 85, 166-167, 177 — ROSTER-01, ROSTER-02, SKILL-01's exact
  wording and their remap to Phase 191 as "rulings-by-deletion"
- `.planning/phases/172-wiring-proof/172-CONTEXT.md` lines 203-204, 232 — the original, precise
  enumeration of the 8 skill-lifecycle orphan commands (corrects the planner's own first hypothesis —
  see D-09)
- `CLAUDE.md` § "Definition of Done" — a requirement is satisfied only when a command exists that fails
  when the requirement is unmet; prefer invariant/existence-check ratchets over named-string checks

### Criterion 1 — zero-reader configs
- `colony/agents/*.yaml` (27 files), `colony/phases/*.yaml` (9 files) — confirmed zero-reader, full
  directory listing in this CONTEXT's `<domain>` section
- `colony/policies/model-routing.yaml`, `autopilot.yaml`, `memory-rules.yaml` — confirmed zero-reader
- `cmd/publish_cmd.go`, `cmd/install_cmd.go` — confirmed zero `colony` references (D-13, no
  distribution-side cleanup needed)

### Criterion 2 — the four loaders
- `cmd/prompt_template_loader.go` (full file, 158 lines) — `loadColonyPrimeTemplates`,
  `getSectionTemplate`, `writeSectionHeader`, `sectionString`, `fmtOrFallback`, `charterFallbackHeading`,
  `charterSectionHeading`, `colonyPrimeTemplatesPathOverride`, `resetColonyPrimeTemplatesCache`
- `cmd/colony_prime_context.go` — every `writeSectionHeader(...)`/`fmtOrFallback(...)`/`sectionString(...)`
  call site (confirmed at lines 263, 411-463, 526, 571-658 and others — read the full file, do not
  assume this list is exhaustive) — these inline string literals ARE the compiled defaults for
  `colony/prompts/colony-prime.md`
- `cmd/visuals_config.go` (full file, ~140 lines) — `loadVisualsConfig`, `visualsConfig` struct,
  `fileCasteEmoji`/`fileCasteColor`/`fileCasteLabel`/`fileCommandEmoji`/`fileCastePrefixes`/
  `fileDefaultPrefixes`/`fileAetherWordmark`/`fileVisualDivider`, `visualsPathOverride`,
  `resetVisualsCache`
- `cmd/codex_visuals.go` — `casteColorMap`, `casteEmojiMap`, `casteLabelMap` and any command-emoji /
  wordmark / divider constants (the compiled defaults visuals_config.go's accessors fall back to)
- `cmd/codex_dispatch_contract.go:1-100` — `loadDispatchContractPolicy`, `dispatchContractPolicy`
  struct, `fallbackSurveyExecutionModel` and the 9 sibling fallback constants (lines 69-86),
  `dispatchContractPathOverride`
- `cmd/review_depth.go` (full file, ~90+ lines) — `loadReviewDepthPolicy`, `reviewDepthPolicy` struct,
  `heavyKeywordsFallback`/`securityRiskKeywordsFallback`/`blastRadiusKeywordsFallback`,
  `getHeavyKeywords`/`getSecurityRiskKeywords` and any sibling getters, `reviewDepthPathOverride`
- `cmd/oracle_loop.go:1666-1687` — the three-path (repo/root/hub) reader for
  `oracle-phase-directives.yaml`, confirming it is NOT one of the four (structurally different) and
  must not be touched
- `colony/prompts/colony-prime.md`, `colony/ceremony/visuals.md`, `colony/policies/dispatch-contract.yaml`,
  `colony/policies/review-depth.yaml` — read each in full before diffing against its Go compiled default

### Criterion 3 — dead code
- `pkg/trace/cost.go` (full file, 39 lines) — `modelRates`, `CalculateCost`
- `pkg/agent/pool.go` — `poolStreamHandler.OnComplete` (~line 178-190, the block calling
  `trace.CalculateCost`/`h.tracer.LogTokenUsage` to remove), `NewPool` (confirmed zero `cmd/` callers —
  do not delete, out of scope per D-06)
- `pkg/agent/pool_test.go`, `pkg/agent/pool_streaming_test.go` — read before editing `pool.go`'s
  `OnComplete`; confirm neither test asserts on the specific cost-tracking block being removed (if one
  does, that test itself is testing dead-path behavior and its assertion should be removed, not
  preserved by inventing a fake caller)
- `cmd/helpers.go:198-202` — `newLearningValidator`, confirmed zero callers, delete outright
- `.claude/commands/ant/medic.md:28`, `.claude/commands/ant-medic.md:28`,
  `.opencode/commands/ant/medic.md:28`, `.aether/commands/medic.yaml`,
  `.aether/skills/colony/context-management/SKILL.md:33`, `CHANGELOG.md:191` — the evidence
  `session-verify-fresh` is live; **do not delete anything named `session-verify-fresh` or
  `sessionVerifyFresh`** anywhere in `cmd/session_cmds.go` or `cmd/session_flow_cmds.go`
- `cmd/skills.go` lines 178-410 — the 8 target cobra commands (`skillIndexCmd`, `skillDetectCmd`,
  `skillMatchCmd`, `skillInjectCmd`, `skillListCmd`, `skillCacheRebuildCmd`, `skillDiffCmd`, plus
  `skill-parse-frontmatter` at line ~151) and their `RunE` bodies in full
- `cmd/skills.go:767` — `matchSkillsForWorkflow`, the function that MUST survive
- `cmd/codex_build.go:3268` — the live in-process caller of `matchSkillsForWorkflow`
  (`composeBuildManifestBrief`), proof the underlying logic is load-bearing
- `cmd/subcommand_reachability_ratchet_test.go` lines 93-131 (`skillLifecycleOrphanCandidates`,
  `resolveSkillLifecyclePaths`), lines 1152-1237 (the T-172-07 boundary proof, the orphan-allowlist
  load/compare, the D-08 count assertions) — full read required before editing; `resolveSkillLifecyclePaths`
  `t.Fatalf`s on any of the 8 leaf names failing to resolve, so this file MUST be updated in the SAME
  change that deletes the 8 commands, not after
- `cmd/testdata/orphan_allowlist.json` lines ~1008-1093 — the entries for all 8 target commands (and the
  8 CRUD-authoring commands, which stay, per D-14) to remove precisely by `name` match

### Criterion 4 — false docs
- `.aether/workers.md:795-833` — full "Wisdom Pipeline" section, including line 825's claim to correct
- `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md` — confirmed zero mentions of
  `memory-capture` in either (grep evidence in `<domain>`)
- `pkg/learn` (package, exact entry point to be traced by the executing task) — the real automatic
  learning-capture mechanism that runs during continue
- `CLAUDE.md:628-636` — "Trim order" list to correct
- `cmd/context.go:1015-1055` — the real `trimOrder` implementation (`ROLLING SUMMARY` →
  `PHASE LEARNINGS` → `KEY DECISIONS` → `HIVE WISDOM` → `CONTEXT CAPSULE` → `USER PREFERENCES` →
  `QUEEN WISDOM (Global)` → `QUEEN WISDOM (Local)` → `ACTIVE SIGNALS`; blockers never trimmed)
- `CLAUDE.md:445-455` — "Command Playbooks (Reference Material)" section, host-build claim to correct
- `.claude/commands/ant/build.md:21,41,105-108,253,259,272,295,304,348-355` — the real interactive
  wrapper flow (plan-only manifest, wrapper-driven spawn, build-finalize) that CLAUDE.md's text must
  match

### Criterion 5 — protected
- `colony/policies/oracle-phase-directives.yaml` — must exist, byte-unchanged, at phase end
- `.claude/commands/ant/oracle.md`, `.claude/commands/ant/dream.md` (and OpenCode equivalents) — not
  touched by any plan
- Any existing oracle selftest / smoke command (`aether oracle selftest`, referenced in STATE.md's
  WORKFLOW-03 evidence) — use as the smoke-pass proof for `/ant-oracle`; locate the equivalent for
  `/ant-dream` during the final verification plan

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `*PathOverride` test seams (`colonyPrimeTemplatesPathOverride`, `visualsPathOverride`,
  `dispatchContractPathOverride`, `reviewDepthPathOverride`) — already exist on all four loaders;
  reuse for the Go-level before/after regression test, do not invent a new override mechanism.
- `testdata/orphan_allowlist.json` + `-update-orphan-allowlist` flag idiom
  (`cmd/subcommand_reachability_ratchet_test.go`) — the established, house pattern for tracking known
  orphans with an owner/reason tag; reuse its shape for any new allowlist entries this phase's own
  deletions might affect (none currently expected — deletions remove entries, they do not add new
  orphans).

### Established Patterns
- **A guard/ratchet that finds nothing must fail, not pass.** Established by Phase 187/188/189/190's
  own ratchets (per the prime rule). Every reappearance check this phase adds must be proven against a
  real re-creation of the deleted file/command (temporarily recreate it, confirm the ratchet fails,
  then confirm it passes again once removed) — a fail-then-pass proof, not an assumed one.
- **Structural AST-based ratchets over grep, for anything beyond pure file-absence.**
  `cmd/colony_state_atomicity_ratchet_test.go`'s house style (referenced directly by this phase's own
  planning brief) parses `cmd/*.go` with `go/ast` rather than grepping. For this phase's pure
  file-existence ratchets (criterion 1's 5 files, criterion 2's 4 files), a simple `os.Stat` /
  existence-check test is proportionate and sufficient — per the prime rule's own explicit permission
  ("don't over-engineer"). AST-level rigor is reserved for the skill-lifecycle CLI-surface removal
  (criterion 3), where `cmd/subcommand_reachability_ratchet_test.go`'s existing AST-based scanner is the
  house mechanism being edited, not replaced.
- **A flag/parameter on a shared function when two callers need genuinely different, both-valid
  contracts** — not needed this phase (no such split arises), noted only because Phase 190 established
  it as the house idiom should a similar situation arise during execution.
- **Delete the now-single-caller/now-trivial helper outright rather than leaving it as harmless dead
  code** — established by Phase 189's D-02, reused by Phase 190's D-10. Applied here only where full
  transitive deadness is confirmed (`newLearningValidator`, `pkg/trace/cost.go`); explicitly NOT applied
  where a function remains load-bearing via an in-process caller (`matchSkillsForWorkflow` and its
  helpers) — the distinction is the entire point of D-09.

### Integration Points
- Plans 191-01 through 191-06 share **zero files** with each other (verified file-by-file in this
  CONTEXT's `<domain>`/`<canonical_refs>` sections) and therefore all run in Wave 1, fully parallel.
- Plan 191-07 (final verification) depends on all six Wave-1 plans — it re-runs the full test suite,
  confirms `oracle-phase-directives.yaml` is byte-unchanged, and smoke-tests `/ant-oracle`/`/ant-dream`
  only after every deletion has landed, since its job is to catch any cross-plan interaction none of the
  individual plans could see alone.
- Within the combined loader plan (colony-prime.md + dispatch-contract.yaml), the two loaders' baseline
  captures should happen together (both read-only, no risk of collision) before either fold-and-delete
  step, since both may touch `cmd/codex_dispatch_contract.go`.

### Test-Coverage Reality
- None of the four loaders in criterion 2 currently have a dedicated `*_test.go` file
  (`prompt_template_loader_test.go`, `visuals_config_test.go` do not exist; `codex_dispatch_contract.go`
  has no dedicated test file either). Coverage for loaders 1 and 2 lives inside
  `cmd/colony_prime_context_test.go` and `cmd/codex_visuals_test.go` respectively (both confirmed to
  exist). Loader 3 (`codex_dispatch_contract.go`) has **no confirmed existing unit coverage** — the
  Nyquist Rule requires the executing task to add it, not assume it exists. Loader 4
  (`cmd/review_depth.go`) has `cmd/review_depth_test.go`.
- The skill-lifecycle CLI-surface removal is exercised end-to-end by
  `cmd/subcommand_reachability_ratchet_test.go`'s existing orphan scan — running it BEFORE any edit
  (to see today's real state, not a hand-derived approximation) and AFTER (to prove the 8 commands are
  gone and the allowlist/ratchet stay internally consistent) is the primary verification for that task,
  not a new bespoke test.

</code_context>

<specifics>
## Specific Ideas

- This phase's single most consequential finding is D-07: the ROADMAP text asking for
  `session-verify-fresh`'s removal is **itself stale**, not the code. Executing it literally would
  silently break `/ant-medic` — the exact class of regression CLAUDE.md's Definition of Done exists to
  prevent, inverted (a "cleanup" that removes something newly load-bearing, rather than a claim of
  aliveness for something that never worked). This is why the prime rule's "verify by execution, not
  grep" instruction is applied here in its most literal form: grep found the string in wrapper markdown
  immediately; a planner that trusted only the ROADMAP's characterization (itself presumably grep-based,
  from before the medic wiring landed) would have missed it.
- The second most consequential finding is D-09: SKILL-01's true 8-command set is the
  matching/indexing/injection machinery (`skill-index`, `skill-detect`, `skill-match`, `skill-inject`,
  `skill-list`, `skill-diff`, `skill-parse-frontmatter`, `skill-cache-rebuild`), not the CRUD-authoring
  commands a first read of the filenames would suggest. Getting this backwards — deleting
  `matchSkillsForWorkflow` because its CLI wrapper looked unreferenced — would have silently broken live
  worker skill injection, discoverable only by a human noticing builders stopped receiving skill
  sections, exactly the "passing for the wrong reason" failure mode CLAUDE.md's own history warns about.
- Criterion 1 and criterion 2 are mechanically different kinds of "zero-reader": criterion 1's targets
  have NO reader at all (deletion needs no fold step); criterion 2's targets DO have a reader, but one
  that is CWD-relative and therefore silently inert outside a dev checkout. Conflating the two would
  either under-engineer criterion 2 (deleting without folding first risks a real, if narrow,
  dev-checkout behavior change) or over-engineer criterion 1 (there is nothing to diff or fold when
  nothing reads the file).

</specifics>

<deferred>
## Deferred Ideas

- **The 4 additional zero-reader `colony/policies/*.yaml` files** (`pheromone-lifecycle.yaml`,
  `safety-gates.yaml`, `signal-rules.yaml`, `skill-creation.yaml`) — same defect class as criterion 1's
  named targets, not named by the ROADMAP text, not touched. Candidate for a future ruling.
- **The 27 zero-reader `colony/prompts/*.md` per-agent files** (everything except `colony-prime.md`) —
  same reasoning. Candidate for a future ruling, likely alongside `colony/agents/*.yaml`'s original
  ROSTER-01 concern since they were built for the same dead control-ts initiative.
- **`colony/playbooks/*.md`** (7 files) — readers found only in test files; whether this constitutes a
  real production dependency was not traced this phase. Candidate for future investigation before any
  ruling.
- **The 8 CRUD-authoring skill-lifecycle commands** (`skill-patch`, `skill-archive`, `skill-pin`,
  `skill-list-lifecycle`, `skill-promote`, `skill-view`, `skill-curator-run`, `skill-recover`) — a
  different, unreviewed subsystem per the existing reachability test's own comment. Not SKILL-01's
  scope. Candidate for a future, separate ruling.
- **Whether the entire `pkg/agent` package (`Pool`, `Registry`, `spawn_tree.go`, `stream_manager.go`,
  etc.) is dead**, beyond the specific `OnComplete`/`CalculateCost` relationship this phase resolves —
  `agent.NewPool` and `agent.Registry` both show zero production (`cmd/`) callers in this phase's
  reconnaissance, but a full audit of the rest of the package (12+ files) is a materially larger
  question than "pkg/trace/cost.go + unconstructed pool path" asks and was not undertaken.
- **Continue's own `brief_path` mechanism** and other Phase 190 deferrals — not this phase's concern,
  noted only to confirm no collision.

</deferred>

---

## Success Criteria Coverage

| # | ROADMAP Success Criterion | Covered By | Notes |
|---|---|---|---|
| 1 | Zero-reader configs deleted (colony/agents, colony/phases, model-routing, autopilot, memory-rules) with grep-ratchets against reappearance | 191-01 | Delivers ROSTER-01, ROSTER-02 in full |
| 2 | The four CWD-relative silent-fallback loaders resolved: diffed, folded, deleted — dev-checkout output byte-identical | 191-02 (colony-prime.md + dispatch-contract.yaml), 191-03 (visuals.md), 191-04 (review-depth.yaml) | Split across 3 plans due to a shared-file constraint between loaders 1 and 3 (see file-overlap note) |
| 3 | Dead code removed (cost.go + unconstructed pool path, session-verify-fresh, newLearningValidator, unreferenced skill-lifecycle commands) | 191-05 | **`session-verify-fresh` is corrected, not executed literally — see D-07.** Delivers SKILL-01 |
| 4 | False docs corrected (workers.md:825, CLAUDE.md trim-order and host-build claims) | 191-06 | |
| 5 | oracle-phase-directives.yaml and auxiliary commands untouched — /ant-oracle and /ant-dream smoke-pass | 191-07 (verification, Wave 2, depends on 191-01..06) | No plan before 191-07 touches any file this criterion protects |

Every one of the five ROADMAP success criteria is covered by at least one plan. One correction (D-07,
`session-verify-fresh`) and several conservative exclusions (D-02, D-14) are recorded explicitly rather
than silently executed or silently dropped.

---

*Phase: 191-Dead Wood*
*Context gathered: 2026-08-21*
