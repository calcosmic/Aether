# Phase 165: Core Lifecycle Commands - Research

**Researched:** 2026-08-02
**Domain:** Slash-command wrapper markdown (Claude Code / OpenCode), Go ceremony CLI surface, wrapper-vs-runtime ownership boundary
**Confidence:** HIGH (current file contents, test contracts, and CLI surface verified directly by reading repo source and running the test suite; ceremony-restoration wording choices remain a DISCRETION/MEDIUM area by design)

## Summary

Phase 165 rewrites four wrapper markdown files so they read as engineering method (stage purpose, files to read, spawn choreography, stop conditions) with Classic Queen-voice ceremony restored, instead of protocol prose whose primary job is describing how to parse `result.manifest.dispatch_manifest`. The governing fact from archaeology still holds after reading the live files: today's wrappers (152 lines for build.md, 124 for continue.md, 142 for plan.md, 158 for init.md) are already far more than the 65/60-line v5.4.0 stubs — they carry real JSON-envelope-handling detail (manifest fetch, brief/brief_path/context_capsule verbatim rules, permission_profile inspection, handoff object requirements) but almost none of the Classic ceremony (no colony-framing beat per stage, no rule-with-reason 🐜 blocks, no `<success_criteria>/<failure_modes>/<read_only>` blocks, no named cross-stage state-carry contract, no uniform stage skeleton). The rewrite must braid these two things together per stage without weakening any of the 5 existing enforcement test files (`build_wrapper_ceremony_test.go`, `continue_wrapper_ceremony_test.go`, `plan_wrapper_ceremony_test.go`, `plan_wrapper_cards_test.go`, `platform_doc_hygiene_test.go`), all of which pass today and encode exact required/forbidden substrings and ordering.

The single most important new finding beyond the archaeology report: **a third, untracked-by-archaeology copy of these wrapper files exists at repo root** — `.claude/commands/ant-build.md`, `ant-continue.md`, `ant-init.md`, `ant-plan.md` (flat, no `ant/` subdirectory). These are the "installed consumer" shape that `aether update`/`aether install` write into a repo (`.claude/commands/ant/*.md` -> `~/.claude/commands/ant-*.md`, confirmed in `install_cmd.go` and `update_roundtrip_test.go`). This Aether repo dogfoods itself, so it carries both the canonical nested source AND a locally-installed flat copy. Diffing them today shows `ant-continue.md` and `ant-plan.md` are byte-identical to their canonical counterparts, but `ant-build.md` and `ant-init.md` have already drifted (missing the `handoff` object requirement, `brief_path`/`context_capsule` verbatim rules, the Colony Mode section, the Prior Context section, and the Cross-Platform Drift Guard section). Existing tests already encode this asymmetry: `plan_wrapper_ceremony_test.go` and `continue_wrapper_ceremony_test.go` explicitly assert content in `ant-plan.md`/`ant-continue.md`, but no test asserts content in `ant-build.md` or `ant-init.md`. The planner must make an explicit scope decision here (see Open Questions) rather than silently repeating the drift pattern that has already bitten build.md and init.md once.

**Primary recommendation:** Rewrite each of the four wrappers stage-by-stage using the D-05 skeleton (colony beat -> Purpose -> Reads -> Spawn choreography -> Stop conditions), mining exact text from the named playbook files and `v5.4.0` init.md/plan.md (archaeology report §A has verbatim exemplars with line numbers), inlining rather than loading them, and add the `<success_criteria>/<failure_modes>/<read_only>` + named state-carry + stop-condition prose that both the Classic era and today's wrappers are each individually missing. Extend the existing 5 enforcement tests with proportion/invariant assertions (not just new required substrings) and add a sixth, `init_wrapper_ceremony_test.go`, which does not exist today. Decide and act on the flat-mirror-file question before or during Wave 1, not after.

## User Constraints (from CONTEXT.md)

<user_constraints>

### Locked Decisions

- **D-00 (governing insight):** Classic ceremony did not live in the old `build.md`/`continue.md` — it lived in `.aether/docs/command-playbooks/*.md` (~4,400 lines). The rewrite MINES that text and INLINES what it restores. The playbook Read-loader is forbidden by `cmd/build_wrapper_ceremony_test.go` and `cmd/platform_doc_hygiene_test.go` and must never return.
- **D-01 (voice/audience):** Queen voice in the Classic register, anchored to method. Each stage opens with one colony-framing beat, then engineering substance (Purpose / Reads / Spawns / Stop conditions). Signature idiom: rule-with-reason-in-colony-terms at the moment the rule fires. Persona is presentation; method is content. Success criterion 3 is the acid test.
- **D-02 (restore these specific beats):** the "You are the Queen. You DIRECTLY spawn..." opener with no-Prime-Worker rule as colony law; 🐜 "Why this matters" reason blocks at spawn/verification stages; 👑 intention-setting beat in init; the Next Up closing idiom and `Phase N of M` framing (narrated by wrapper, rendered by runtime commands where they exist); Classic `━━━ ... ━━━` separator conventions in wrapper-authored narration — but NEVER hand-render what a runtime ceremony command already renders.
- **D-03 (envelope mechanics leave wrapper prose):** the spawn stage says, as method: fetch the manifest (`aether host build --dry-run N`), spawn each worker with the runtime-composed brief verbatim, respect `execution_plan` waves, print runtime ceremony output as results arrive. Field-by-field parsing/envelope-shape docs/temp-file bookkeeping move to `.aether/docs/wrapper-host-contract.md`, referenced once per wrapper. Success criterion 2 (grep returns zero) is the test.
- **D-04 (handshake wording):** one line per mechanical step, subordinated under the stage's method. Nothing that reads as "parse JSON field X into Y".
- **D-05 (uniform stage skeleton):** stage name (aligned with runtime's `── Stage Name ──` markers: Context, Tasks, Dispatch, Verification, Housekeeping, Next Phase) -> colony beat -> Purpose -> Reads -> Spawn choreography (if any) -> Stop conditions. init.md keeps its own stage names but the same skeleton.
- **D-06 (restore three Classic method assets):** `<success_criteria>/<failure_modes>/<read_only>` blocks per wrapper (all four); the named cross-stage state-carry contract (v5.4.0's phase_id, depth, prompt_section, wave_results, verification_status, next_action — updated to current names); termination-condition prose (confidence target, stall detection, iteration caps, escape hatches — matching what the runtime actually enforces today; wrappers describe, never re-implement).
- **D-07 (spawn choreography stays wrapper-owned):** all wave-1 workers in a single message via multiple Task calls, wait for wave completion before next wave, never `run_in_background`, never invent castes or names — use the manifest.
- **D-08 (build.md ownership handshake):** an HTML comment header in build.md records what Phase 160 fixed (merged first) and reserves a named trailer marker (e.g. `<!-- PHASE-168: visual-guidance trailer appends below this line -->`) for Phase 168. The rewrite commit message states the ownership chain. A test greps for both the header record and the reserved marker.
- **D-09 (ceremony Definition of Done):** ship tests that fail when the ceremony or method is absent. Extend the existing wrapper test pattern with assertions that (a) each wrapper carries the required stage skeleton sections, (b) required narration beats are present (prefer proportion/invariant assertions over named-section greps), (c) every already-forbidden regression stays forbidden, (d) `.claude`/`.opencode` heading parity holds (`TestPlanWrapperCardsParity` style).
- **D-10 (regression fence — hard constraints, do NOT bring back):** playbook Read-loader (CRITICAL); wrapper state mutation of COLONY_STATE.json/pheromones/constraints (CRITICAL); plan.md watch-file/tmux writes; caste legend or speculative caste naming in prose; context-clear ceremony in continue.md; gate threshold arithmetic in markdown; synthetic build/plan forcing; verbatim user transcripts in prompt files; wrapper-driven git stash/commit; `.aether/aether-utils.sh` or LiteLLM-proxy assumptions; the retired four-value `colony_depth` vocabulary.
- **D-11 (platform parity):** `.claude` and `.opencode` rewritten together in the same plan(s), parity pinned by ordered-heading-set tests. Codex is runtime-native — no wrapper work. `.aether/commands/{init,plan,build,continue}.yaml` guardrail entries update in lockstep with wrapper guardrails.

### Claude's Discretion

- Exact wording of colony beats and narration (mine the playbooks freely).
- Which runtime ceremony commands each wrapper invokes at which stage, provided no hand-rendering of runtime-owned visuals.
- How to split the work into plans/waves.
- Whether `oracle.md`-style short wrappers need touch-ups is OUT of scope — only the four named files (both platforms) plus their YAML sources.

### Deferred Ideas (OUT OF SCOPE)

- Nine renderer-owned ceremony gaps (wave-failure banner, escalation banner, verification report grid, workflow pattern announce, graveyard caution, survey-loaded banner, visual checkpoint, project-complete block, resumption line) plus richer build-summary lines — belong to Phase 168 (Go renderer work).
- Milestone/maturity framing in build/continue — net-new, Phase 168 decides.
- `continue-finalize` not counting `--reconcile-task` as recorded reconciliation — Go runtime behavior fix, out of scope for a wrapper-text phase.
- ts-host preflight timeout hardcoded — largely addressed by Phase 163.2, weak match, stays in backlog.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CMD-01 | `init`, `plan`, `build`, `continue` wrappers carry engineering method — stage purpose, files to read, spawn choreography, synthesis instructions, stop conditions | D-05 skeleton + playbook-mined method text (Architecture Patterns, Code Examples). Current wrapper audit below shows exactly which sections are missing per file. |
| CMD-02 | No lifecycle wrapper instructs the model to parse an internal JSON envelope or write to a temporary manifest file as its primary job | D-03/D-04; exact phrases to relocate to `wrapper-host-contract.md` are listed in Common Pitfalls; grep-based test recipe in Validation Architecture. |
| CMD-03 | A user reading `build.md` can describe what each stage does without opening Go source | D-01/D-02 colony beats + Purpose/Reads prose; success criterion is a readability/prose test, not a mechanical one — flagged as manual-verification-supported in Validation Architecture. |
| CMD-04 | Specialist/delight commands (`chaos`, `archaeology`, `dream`, `oracle`, `swarm`, `sage`, `colonize`, `council`) keep working unchanged | Phase touches only 4 files (+ mirrors + YAML); regression test recipe: full `platform_doc_hygiene_test.go` + targeted greps on the 8 untouched command files to prove zero diff. |
| CMD-05 | `build.md` edited by at most one phase this milestone, or merge order stated explicitly | D-08 header/trailer-marker mechanism; exact grep test recipe in Validation Architecture. |

</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Manifest generation (dispatch plan, briefs, castes, waves) | API/Backend (TS host + Go CLI, `aether host {build,plan,continue}`) | — | Already Go/TS-owned; wrapper only fetches and reads it |
| Worker spawning (Task/subagent calls) | Browser/Client (platform Agent tool inside Claude Code / OpenCode) | Wrapper markdown (choreography rules) | Only the platform tool layer can actually spawn a Task; wrapper markdown is the instruction set consumed by the calling model, not a separate tier |
| Ceremony visual rendering (banners, spawn plans, wave starts, worker-complete lines, closeout) | API/Backend (Go `aether ceremony *` subcommands, `cmd/ceremony_cmd.go`, `cmd/codex_visuals.go`) | Wrapper markdown (narration around the rendered output) | Runtime owns all ANSI/text formatting; wrapper's job is deciding *when* to call the renderer and what plain-English framing surrounds it |
| State mutation (COLONY_STATE.json, session.json, pheromones, constraints) | API/Backend (Go finalizers: `build-finalize`, `continue-finalize`, `plan-finalize`) | — | Hard-forbidden in wrapper prose by D-10/enforcement tests; no tier ambiguity here |
| Stage purpose / files-to-read / stop-condition prose (this phase's actual deliverable) | Wrapper markdown | — | This is process documentation for the calling model, not a runtime capability — it has no other owner and cannot be pushed to Go without losing the "readable without opening Go source" property required by CMD-03 |
| Envelope field-parsing mechanics (temp manifest file handling, field names) | Docs (`.aether/docs/wrapper-host-contract.md`) | Wrapper markdown (one-line pointer) | D-03 explicitly relocates this off the wrapper's primary-job surface into the named contract doc |
| Ownership/merge-order bookkeeping for build.md | Wrapper markdown (HTML comment header + reserved trailer marker) | Git commit message | D-08; this is a documentation-plus-test concern, not a runtime one |

## Current State of the Four Wrappers (as of this research)

All four live wrapper files were read in full. None of them are the thin v5.4.0-style stubs the archaeology's headline framing might suggest — they have grown substantially since (152/124/142/158 lines for build/continue/plan/init respectively) and already carry real JSON-envelope/manifest-handling detail. What they are missing, uniformly, is ceremony and stage-skeleton structure:

| File | Lines today | Has: Ownership/contract prose | Has: colony beat per stage | Has: `<success_criteria>/<failure_modes>/<read_only>` | Has: named state-carry contract | Has: stop-condition prose | Has: D-05 stage skeleton headings |
|------|---|---|---|---|---|---|---|
| `build.md` | 152 | Yes (`## Ownership Split` table) | No | No | No | Partial (`## Verification Depth` only) | No — headings are `Ownership Split / Colony Context / Active Signals / Phase Framing / Dispatch Manifest / Guided Boundary Gate / Runtime Spawn Ceremony / Worker Spawning / Finalize / After the Build / Verification Depth / Cross-Platform Drift Guard / Guardrails` |
| `continue.md` | 124 | Yes | No | No | No | No | No — `Ownership Split / Default Continue / Heavy External Review / After Continue / Cross-Platform Drift Guard / Guardrails` |
| `plan.md` | 142 | Partial | No | No | No | Partial (`requires_next_iteration` handling only) | Partially — already has `## Decision Moment 1/2`, `## Clarification Gate` as quasi-stage headings |
| `init.md` | 158 | No | Weak (👑 intention beat is gone; only a plain Colony Charter echo remains) | No | No | No | No — `Codebase Summary / Prior Context / Intent Refinement / Colony Charter / Colony Mode / Pheromone Suggestions / Shelf Backlog / Cross-Platform Drift Guard / Approval` |

**Verified [VERIFIED: repo read + `go test`]:** all 5 existing enforcement tests pass against the current file contents (`go test ./cmd/... -run 'TestBuildWrapperCeremonyContract|TestContinueWrapperCeremonyContract|TestPlanWrapperCeremonyContract|TestPlanWrapperCardsParity|TestLifecycleCommandDocsPreferRuntimeCLI'` — `ok  github.com/calcosmic/Aether/cmd  1.192s`). Any rewrite must keep all `required`/`inOrder`/`forbidden` assertions in these five files satisfied while adding the new stage-skeleton/ceremony content.

## Enforcement Test Contracts (exact strings — extend, never weaken)

### `cmd/build_wrapper_ceremony_test.go` (`TestBuildWrapperCeremonyContract`)
Applies to `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` only (NOT the flat `ant-build.md` mirror — see Common Pitfalls).

- **Required substrings (23):** `Use the Go \`aether\` CLI as the source of truth.`; `AETHER_OUTPUT_MODE=visual aether status`; `## Active Signals`; `REDIRECT`; `FOCUS`; `FEEDBACK`; `strength or remaining-life context`; `## Phase Framing`; `Phase N of M -- Name`; `## Dispatch Manifest`; `aether host build --dry-run`; `` temporary manifest file outside `.aether/data/` ``; `result.manifest.dispatch_manifest`; `## Runtime Spawn Ceremony`; `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest_file>`; `` Do not set `run_in_background` ``; `` Do NOT run `aether host build` without `--dry-run` from this wrapper ``; `` Do NOT run `aether build --synthetic` after real ``; `AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file`; `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file`; `/ant-continue`; `dispatch_manifest.context_capsule`; `brief_path`.
- **Ordered markers (10):** `## Ownership Split` -> `## Colony Context` -> `## Active Signals` -> `## Phase Framing` -> `## Dispatch Manifest` -> `aether host build --dry-run` -> `## Runtime Spawn Ceremony` -> `AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file` -> `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file` -> `## After the Build`.
- **Forbidden (5):** `Do NOT load playbooks`; `\nAETHER_OUTPUT_MODE=visual aether build $ARGUMENTS\n`; `\nAETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only\n`; `` Do NOT run `aether build` without `--plan-only` from this wrapper. ``; `Do NOT run direct \`aether build\` from this wrapper for manifest generation`.
- **`TestBriefPathReferencedAcrossAllFourSurfaces`** requires the literal string `brief_path` and a case-insensitive `verbatim` in exactly these four files (asserted `len == 4`): `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`, `cmd/command_guide.go`.
- **`TestBuildWrapperVerbatimBriefBulletsStayMirrored`** requires the exact line containing `` Each dispatch carries `brief` `` and the exact line containing `The worker's prompt =` to be byte-identical between `.claude` and `.opencode` build.md. Rewriting these bullets requires editing both files' lines identically, not just semantically.

### `cmd/continue_wrapper_ceremony_test.go` (`TestContinueWrapperCeremonyContract`)
Applies to `.claude/commands/ant/continue.md` and `.opencode/commands/ant/continue.md`.

- 14 required substrings including `` AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS `` (the fast default path) and `result.manifest.continue_manifest`.
- Ordered markers: `## Default Continue` -> the fast command -> `## Heavy External Review` -> `aether host continue --dry-run --classic-ceremony $ARGUMENTS` -> `continue-finalize` -> closeout -> `## After Continue`.
- **Forbidden, permanently (D-5 in archaeology, the "most explicitly-flipped test"):** `It's safe to clear your context now.` and `/ant-resume` must NOT appear in the wrapper prose — this is a **runtime-owned** ceremony line, asserted separately by calling `renderContinueVisual`/`renderContinueBlockedVisual` directly in the same test (not by reading the markdown file). Any rewrite that reintroduces context-clear ceremony text in continue.md will fail this test.
- **`TestContinueWrapperSourcesUseFastDevContinue`** requires the exact fast command string in FOUR files: `.aether/commands/continue.yaml`, `.claude/commands/ant-continue.md` (flat mirror!), `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`. It also asserts two now-retired mirror paths (`.aether/commands/claude/continue.md`, `.aether/commands/opencode/continue.md`) do NOT exist.

### `cmd/plan_wrapper_ceremony_test.go` (`TestPlanWrapperCeremonyContract`)
Applies to `.claude/commands/ant/plan.md`, `.claude/commands/ant-plan.md` (flat mirror!), and `.opencode/commands/ant/plan.md` — three paths, unlike build/continue's two.

- 26 required substrings covering both Decision Moment 1 (depth proposal) and Decision Moment 2 (research batch) card contracts, the Clarification Gate, and the two-decision-moment invariant (`Do NOT add a third decision moment`).
- 9 ordered markers ending `## After Planning` -> `## Guardrails`.
- 8 forbidden strings including retired headings `## Depth Ceremony` / `## Planning Depth`.

### `cmd/plan_wrapper_cards_test.go` (`TestPlanWrapperCardsParity`)
This is the strictest parity test in the change zone and the template D-09 explicitly asks to extend: it uses `reflect.DeepEqual` on the **ordered list of every `## `-prefixed level-2 heading**, after stripping `<!-- -->` comment lines, between `.claude/commands/ant/plan.md` and `.opencode/commands/ant/plan.md`. A one-sided heading rename, addition, or reorder fails it (there is even a sub-test, `a_one_sided_heading_addition_would_fail`, that proves the assertion is load-bearing). It also checks `.aether/commands/plan.yaml` carries both `depth_proposal_card:` and `research_batch_card:` YAML keys, and that each wrapper contains `plan-research-approve --approve-all` and `--auto` exactly once each.

**This is the exact pattern D-09 asks the plan to replicate for build.md, continue.md, and init.md** — none of which currently have an ordered-heading-parity test. `plan_wrapper_cards_test.go` is the concrete template to copy.

### `cmd/platform_doc_hygiene_test.go` (`TestLifecycleCommandDocsPreferRuntimeCLI`)
Per-file required/forbidden lists for `.claude/commands/ant/{init,plan,build,continue}.md` and their exact `.opencode/commands/ant/{init,plan,build,continue}.md` mirrors (lines 72-235 for `.claude`, 386-540ish for `.opencode` — both sets present and currently pass). Forbidden across all four, universally: `.aether/aether-utils.sh`, `Write COLONY_STATE.json`, `Read \`.aether/data/COLONY_STATE.json\`.`. `init.md`-specific forbidden: `queen-init`. `build.md`-specific forbidden: `Briefly name the castes the colony is likely to send`, `such as Builder, Watcher, Scout, Architect, Oracle, or Chaos`, `Read the file with the Read tool.`. `continue.md`-specific forbidden: `continue-verify.md`, `continue-gates.md`, `Read build packet files:`.

## Runtime Ceremony CLI Surface (invoke, never re-render)

**[VERIFIED: `cmd/ceremony_cmd.go` read directly]** — the actual registered `aether ceremony` Cobra subcommands are only **four**, not six:

```
aether ceremony spawn-plan       --workflow <build|plan|continue> --manifest-file <path>
aether ceremony wave-start       --workflow <...> --manifest-file <path> --execution-wave <N>
aether ceremony worker-complete  --workflow <...> --worker-file <path>
aether ceremony closeout         --workflow <...> --completion-file <path>
```

`skill-assignments` and `queen-frame` (`renderCeremonySkillAssignments`, `renderCeremonyQueenFrame`) exist as internal Go functions called *inside* `spawn-plan`'s rendering path, not as separate CLI subcommands — **do not write wrapper text that invokes `aether ceremony skill-assignments` or `aether ceremony queen-frame` directly; those commands do not exist.** This corrects a phrasing ambiguity in the archaeology's canonical-refs list ("`spawn-plan`, `wave-start`, `worker-complete`, `skill-assignments`, `closeout`, `queen-frame` renderers" — accurate as a list of renderer *functions*, not of CLI *subcommands*).

Other runtime-owned surfaces confirmed present and callable exactly as the wrappers already use them: `AETHER_OUTPUT_MODE=visual aether status`, `aether pheromone-display` (referenced in playbook exemplar but not yet wired into any current wrapper — mining opportunity per D-02's "run it, then narrate the legend" pattern), `AETHER_FORCE_COLOR=1` env prefix for forcing color in non-TTY tool contexts.

`cmd/codex_visuals.go` confirms `casteIdentity()`, `casteLabel()`, `casteEmoji()` (lines 3555-3591+), `spacedTitle()` (333), `renderBanner()` (347), `renderNextUp()` (395), `renderStageMarker()` (365, produces the `── Stage Name ──` marker the runtime's own stage separators use — this is the exact marker convention D-05 asks wrapper headings to align with).

## Wrapper-Host Contract and Wrapper-Runtime UX Contract (D-03 destination)

`.aether/docs/wrapper-host-contract.md` (confirmed content, 80 lines) already states the "host-assisted orchestrator" doctrine and a boundary table (Wrappers / TS Host / Go CLI: May / Must Not) but does **not yet contain** field-by-field envelope shape documentation (e.g., what keys exist in `dispatch_manifest`, `continue_manifest`, `plan_manifest`) — that granular parsing detail currently lives only inline in the wrapper prose (e.g., build.md's "Require terminal structured result with: `name`, `caste`, `stage`, ..."). D-03 says this class of detail should move here; this phase should either (a) add a "Manifest and Completion Packet Shapes" section to `wrapper-host-contract.md` enumerating the fields wrappers must not invent, or (b) confirm in the plan that the existing per-wrapper "Required terminal structured result" bullet is method (what a worker owes back), not envelope-parsing mechanics, and is therefore fine to keep — this is a real judgment call, not a mechanical move, flagged as an Open Question.

`.aether/docs/wrapper-runtime-ux-contract.md` (218 lines, confirmed) is the broader ownership contract already covering anti-patterns 1-8 and the Codex exception. No changes needed here per CONTEXT.md scope, but its "Wrapper Additions" section's example commands (e.g. `Continue (default): AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard`) use a `--skip-watchers` flag the live continue.md wrapper does **not** use (`--verification-depth standard` only) — a pre-existing minor doc/wrapper drift, not part of this phase's scope but worth a one-line note if the plan touches this file at all.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Spawn plan / caste identity visuals | Wrapper-authored caste emoji legend or hand-drawn spawn table | `aether ceremony spawn-plan --workflow <x> --manifest-file <path>` (already wired in all three wrappers) | `casteEmojiMap`/`casteColorMap`/`casteLabelMap` in `codex_visuals.go` are the single source of truth; hand-rolled caste tables are explicitly forbidden by `platform_doc_hygiene_test.go`'s build.md forbidden list (`Briefly name the castes...`) |
| Wave/worker-completion streaming lines | Wrapper prose describing a bespoke line format | `aether ceremony wave-start` / `aether ceremony worker-complete` | Same renderer functions already implement the exact Classic format (`🔨 {Ant-Name}: ...`) — re-describing the format in prose risks drift from the actual renderer |
| Context-clear "safe to /clear" guidance | Re-adding this line to continue.md prose | Leave it to `renderContinueVisual`/`renderContinueBlockedVisual` (Go-owned) | This is the single most explicitly test-guarded regression in the whole change zone (D-5); a dedicated Go-level assertion (not just a markdown grep) checks the runtime function's output directly |
| Envelope field enumeration | Repeating full JSON schema tables in wrapper prose | One-line pointer to `.aether/docs/wrapper-host-contract.md` (D-03) | Keeps wrapper's primary job as method, not parsing-instruction authorship; also avoids the schema drifting out of sync in two places |
| Cross-file heading parity checking | Manual side-by-side diff during review | Copy the `TestPlanWrapperCardsParity` pattern (`level2Headings` + `reflect.DeepEqual`) for build/continue/init | Already proven to catch one-sided edits; D-09 explicitly asks for this pattern to extend to the other three wrappers |

**Key insight:** every ceremony element this phase might be tempted to hand-render in wrapper prose almost certainly already has a Go renderer function (delta table item categories `RT (done)` in the archaeology report) — the actual gap this phase fills is textual/structural (stage skeleton, colony beats, method prose), not visual.

## Common Pitfalls

### Pitfall 1: The flat legacy mirror files (`ant-build.md`, `ant-init.md`) are already stale and untested
**What goes wrong:** `.claude/commands/ant-build.md` and `.claude/commands/ant-init.md` (repo root, no `ant/` subdirectory) are separate, git-tracked, independently-editable copies of the canonical wrapper content — the "installed consumer" shape that `aether install`/`aether update` write (`install_cmd.go`: `.claude/commands/ant/ -> ~/.claude/commands/ant-*.md`). This Aether repo dogfoods its own commands, so it carries both shapes locally. **[VERIFIED: `diff` of the four pairs]** — `ant-continue.md` and `ant-plan.md` are currently byte-identical to their canonical `ant/` counterparts (and are actively asserted by `TestContinueWrapperSourcesUseFastDevContinue` and `TestPlanWrapperCeremonyContract` respectively), but `ant-build.md` is missing the `handoff` object requirement, the `brief_path`/`context_capsule` verbatim rules, and reads an older, thinner version of the Worker Spawning section; `ant-init.md` is missing the entire Prior Context section, the Colony Mode section, and the Cross-Platform Drift Guard section, and calls `aether init` without the `--colony-mode` flag the canonical version now requires. No test currently asserts content in `ant-build.md` or `ant-init.md` at all.
**Why it happens:** Nothing keeps the flat mirror in sync automatically inside this repo (that only happens via `aether update`/`aether publish` round-trips against the hub, which is a separate manual step from editing `.claude/commands/ant/*.md` directly). Two of four commands happened to get re-synced after their last edit; two did not.
**How to avoid:** The plan must explicitly decide, and act on, one of: (a) edit the flat mirrors in lockstep with the canonical files as part of this phase's tasks (mirroring what `plan.md`/`continue.md` already do, extending the same discipline to `build.md`/`init.md`), or (b) run the actual publish/update round-trip against this repo after the canonical rewrite so the flat mirrors regenerate correctly, or (c) explicitly document in the plan why the flat mirrors are out of scope (e.g., if they are dead weight in this repo and not what Claude Code actually loads here) — but do not simply ignore them, since two of the four already prove the drift is real, not theoretical.
**Warning signs:** `diff .claude/commands/ant-build.md .claude/commands/ant/build.md` produces non-empty output after this phase's rewrite lands.

### Pitfall 2: Restoring ceremony prose that duplicates a runtime renderer
**What goes wrong:** Mining the playbooks (e.g., the Caste Emoji Legend at `build-wave.md:64-77`, the six gate-failure banner templates, the verification report grid) and inlining the literal ASCII art into wrapper prose reintroduces content the Go runtime already renders and that `platform_doc_hygiene_test.go` / `build_wrapper_ceremony_test.go` explicitly forbid (`Briefly name the castes...`, playbook-loader phrasing).
**Why it happens:** The archaeology report's exemplars are presented as "restore these," but the delta table (§C) marks 20 of 32 elements `RT (done)` or `RT` (renderer-owned) — only ~9 are genuinely `WR` (wrapper-owned safe content): `<success_criteria>/<failure_modes>/<read_only>` blocks, the named state-carry contract, stop-condition prose, files-to-read routing description, and spawn choreography rules.
**How to avoid:** Cross-check every mined beat against the delta table's ownership column before inlining it. If ownership is `RT` or `RT (done)`, the wrapper's job is only to *narrate around* the already-wired `aether ceremony ...` call, never to re-describe its visual output.
**Warning signs:** New wrapper prose contains a literal `━━━`/`──── ` ASCII block that duplicates what a `renderCeremony*` function already outputs, rather than a one-line instruction to invoke that function.

### Pitfall 3: Reintroducing a test-forbidden phrase while restoring "safe" Classic prose
**What goes wrong:** Some Classic exemplars sit right next to forbidden mechanisms in the same playbook file (e.g., `build-wave.md:251`'s spawn-choreography rule, which is safe and D-07-sanctioned, sits near the Caste Emoji Legend, which is forbidden). Copy-pasting a whole playbook block risks carrying a forbidden phrase along with a wanted one.
**How to avoid:** Mine sentence-by-sentence against both D-06 (safe assets to restore) and D-10 (regression fence) simultaneously; don't copy whole playbook sections wholesale.
**Warning signs:** `go test ./cmd/... -run 'TestBuildWrapperCeremonyContract|TestContinueWrapperCeremonyContract|TestPlanWrapperCeremonyContract|TestLifecycleCommandDocsPreferRuntimeCLI'` fails after a prose restoration commit.

### Pitfall 4: `TestBuildWrapperVerbatimBriefBulletsStayMirrored` breaks on a "just reword slightly" edit
**What goes wrong:** This test does a byte-exact line comparison (not substring) of two specific lines between `.claude` and `.opencode` build.md. A ceremony rewrite that touches the Worker Spawning section's wording in one platform file but not the other — even a whitespace or word-order difference — fails it.
**How to avoid:** Any edit to the "Each dispatch carries `brief`" line or "The worker's prompt =" line must be applied identically, character-for-character, to both `.claude/commands/ant/build.md` and `.opencode/commands/ant/build.md` in the same commit.

### Pitfall 5: `renderContinueVisual`/`renderContinueBlockedVisual` assertions are Go-level, not markdown-level
**What goes wrong:** A plan that only reviews `continue_wrapper_ceremony_test.go`'s markdown-string assertions might miss that the same test file also directly calls Go rendering functions (`renderContinueVisual`, `renderContinueBlockedVisual`) and asserts their *output strings* contain/exclude the context-clear line. This part of the test has nothing to do with wrapper markdown content and cannot be satisfied by editing `continue.md` — it is already correct and must simply not be touched by an unrelated Go change during this markdown-only phase.
**How to avoid:** Treat this test file's two halves separately when scoping tasks: the markdown-content assertions are this phase's concern; the Go-render assertions are pre-existing and out of scope (no runtime behavior changes per CONTEXT.md phase boundary).

## Code Examples

### D-05 stage skeleton, applied to a representative build.md stage (illustrative — exact wording is Claude's discretion per CONTEXT.md)

```markdown
## Dispatch Manifest

🐜 The manifest is the colony's marching order — fetch it, then spawn exactly
what it names. Nothing invented, nothing skipped.

**Purpose:** Get the runtime's authoritative worker/wave plan for this phase
before any spawning happens.

**Reads:** none (this stage only calls the TS host).

**Spawns:** none yet — this stage only fetches the plan.

**Do:**
```
aether host build --dry-run $ARGUMENTS
```
The manifest is the sole source of worker names, castes, waves, and briefs —
see `.aether/docs/wrapper-host-contract.md` for the full envelope shape.

**Stop conditions:** if provider dispatch is unavailable, surface only the
Go-owned structured availability message (provider, sanitized cause, next
action) and stop here — do not fabricate a manifest.
```

### Named cross-stage state-carry contract (D-06), source text to mine and modernize

```
// Source: v5.4.0 build.md:35-60 (via 165-ARCHAEOLOGY.md §B)
## Required Cross-Stage State
Carry these values forward when produced:
- `phase_id` / `visual_mode` / `verbose_mode` / `suggest_enabled`
- `colony_depth` / `prompt_section` / `wave_results`
- `verification_status` / `synthesis_status` / `next_action`
```
Modernize the vocabulary per D-11 (`colony_depth` -> `--verification-depth <light|standard|heavy>`, per D-10's regression fence item) before inlining; do not restore the four-value `colony_depth` enum.

### Rule-with-reason 🐜 idiom (D-01/D-02), source text to mine

```
// Source: continue-gates.md:49-52 (via 165-ARCHAEOLOGY.md §A-1)
🐜 Why this matters:
  - Builders verify their own work = confirmation bias
  - Independent Watchers catch bugs builders miss
  - "Build passing" ≠ "App working"
```

### Ordered-heading-parity test pattern to extend (D-09), source to copy

```go
// Source: cmd/plan_wrapper_cards_test.go:37-47 (verified passing today)
func level2Headings(text string) []string {
	var headings []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line)
		}
	}
	return headings
}
// then: reflect.DeepEqual(claudeHeadings, opencodeHeadings)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `build.md`/`continue.md` Read-load playbook files as their execution contract | Playbooks mined once, ceremony text inlined; loader forbidden | CLAUDE.md notes "not loaded since v1.25" | This phase's rewrite must never reintroduce `Read the file with the Read tool.`-style loader phrasing |
| `colony_depth` four-value vocabulary (`light/standard/deep/full`) | `--verification-depth <light|standard|heavy>` + Queen smart defaults | Post-v5.4.0 | Any restored depth-label prose must use the current three-value vocabulary, not the old four-value one (D-11 regression fence item) |
| Wrapper hand-formats pheromone signals as free prose | `aether pheromone-display` renders the table; wrapper narrates only | Already true for build.md's Active Signals section, not yet wired into continue.md or plan.md | Mining opportunity: playbook's "run it, then narrate the legend" pattern (`build-context.md:23-31`) is a safe D-06 asset not yet fully adopted everywhere |
| Continue wrapper printed "safe to /clear" guidance | Runtime (`renderContinueVisual`) owns this line exclusively | Commit `ecf68ba4` (2026-04-21) | Hard-forbidden regression (D-5); do not restore in prose |

**Deprecated/outdated:** the nine-playbook-file loading mechanism (`build-prep.md` through `continue-finalize.md`) as an *execution* mechanism — the files themselves remain valid *source text* for mining, just never re-wired as a Read-loop.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The flat `.claude/commands/ant-*.md` files are what a Claude Code session actually loads when a user runs `/ant-build` etc. *inside this repo* (versus the nested `.claude/commands/ant/*.md` producing a differently-named/namespaced command) | Common Pitfalls Pitfall 1, Open Questions | If wrong (i.e., Claude Code only ever loads the nested `ant/` shape and the flat files are pure install-destination dead weight never read by this repo's own sessions), the urgency of fixing the flat-mirror drift is lower — but the existing tests (`plan_wrapper_ceremony_test.go`, `continue_wrapper_ceremony_test.go`) already treat the flat files as load-bearing for 2 of 4 commands, so treating them as truly dead for the other 2 would be an inconsistent, undocumented scope decision either way |
| A2 | `.aether/docs/wrapper-host-contract.md` is the right destination for the "Required terminal structured result" field list currently inline in build.md/continue.md (versus that list being acceptable "method" prose that can stay inline because it describes what a *worker* owes back, not how the *wrapper* parses an envelope) | D-03 destination discussion, Open Questions | If the field list is method (not envelope-parsing), moving it out could make CMD-01 (files to read / spawn choreography) less self-contained without a clear CMD-02 benefit; if it is envelope-parsing, leaving it inline risks tripping CMD-02's "grep for envelope-parsing-as-primary-job" intent even though no current test forbids these exact bullets |

**If this table is empty:** N/A — two assumptions are logged above, both flagged for the planner/discuss-phase to confirm before locking task-level scope.

## Open Questions (RESOLVED)

**All three were resolved during planning on 2026-08-02. The rulings are recorded
in `165-01-PLAN.md`'s objective and are implemented by Plan 01's tasks. They are
locked — downstream agents treat them as chosen, not open.**


1. **Do the flat legacy mirror files (`ant-build.md`, `ant-continue.md`, `ant-init.md`, `ant-plan.md`) get rewritten in this phase's tasks, or handled by a publish/update round-trip, or explicitly descoped?**
   - What we know: two of four (`ant-continue.md`, `ant-plan.md`) are currently in sync with canonical and are already test-asserted; two (`ant-build.md`, `ant-init.md`) are already stale and untested.
   - What's unclear: whether Claude Code, when invoked inside this repo, resolves `/ant-build` to the flat file, the nested file, both (as separate commands), or whichever the platform's directory-scan order happens to prefer first.
   - Recommendation: treat this as a discuss-phase-worthy scope question if not already implicitly decided; at minimum, the plan should state explicitly which of (a)/(b)/(c) from Pitfall 1 it chose, and why, rather than silently rewriting only the nested files and leaving the flat ones exactly as stale as `ant-build.md`/`ant-init.md` are today.
   - **RESOLVED (option a):** the flat mirrors are rewritten in lockstep with their canonical counterparts, and byte-equality is pinned permanently by `TestLifecycleFlatMirrorsMatchCanonical` (`cmd/lifecycle_wrapper_contract_test.go`, created in `165-01-PLAN.md` Task 2; the two drifted files are resynced in Task 3, and each Wave 2 plan re-syncs its own mirror as its Task 3). Rationale: the mirrors are git-tracked, two of four were already test-asserted, and the two that no test covered are the two that drifted — so the drift is caused by missing coverage, and coverage is the fix.

2. **Does the "Required terminal structured result" field-by-field list in build.md/continue.md count as D-03 envelope-parsing mechanics (move to `wrapper-host-contract.md`) or D-06 method (what a worker owes back, stays inline)?**
   - What we know: D-03's own example phrasing ("fetch the manifest... spawn each worker with the runtime-composed brief verbatim... respect execution_plan waves") describes *manifest consumption*, while the terminal-result list describes *worker output shape* — arguably a different direction of data flow than what D-03 names.
   - What's unclear: whether CMD-02's grep-based test (forbidding "primary job is parsing an internal JSON envelope") is meant to catch this list at all; no current test forbids these specific bullets.
   - Recommendation: keep the terminal-result field list inline (it reads as method: "what must a worker report back," not "how does the wrapper parse JSON") but move genuinely mechanical envelope-shape documentation (temp file handling, `result.manifest.dispatch_manifest` vs `result.plan_manifest` field-name differences across the three commands) to `wrapper-host-contract.md`.
   - **RESOLVED (method — stays inline):** the terminal structured result describes what a *worker owes back*, which is the opposite direction of data flow from manifest consumption, so it is D-06 method and remains in the wrappers. The ruling is written into `.aether/docs/wrapper-host-contract.md` as a named section `## Terminal Worker Result Belongs to the Wrapper` (`165-01-PLAN.md` Task 1) so it cannot be re-litigated silently, and the CMD-02 proportion test in `165-06-PLAN.md` Task 1 explicitly does not count it as envelope-parsing prose.

3. **Should `.aether/docs/wrapper-host-contract.md` gain a new "Manifest and Completion Packet Shapes" section as part of this phase, or is a one-line pointer per wrapper sufficient without expanding the contract doc's content?**
   - What we know: D-03 names this doc as "already the named contract doc" and says envelope mechanics "move to" it.
   - What's unclear: whether "move to" requires this phase to actually author new content there, or whether the existing boundary-table content already satisfies the requirement and only the wrapper-side pointer needs adding.
   - Recommendation: default to adding a brief new section there enumerating the 2-3 field names each wrapper currently explains inline (`dispatch_manifest.execution_plan`, `dispatch.brief_path`/`dispatch.brief`, `dispatch.permission_profile`), since CMD-02's grep test is easiest to satisfy cleanly if the detail genuinely lives elsewhere rather than merely being asserted-as-fine-to-keep.
   - **RESOLVED (yes — new content is authored):** `165-01-PLAN.md` Task 1 adds a `## Manifest and Completion Packet Shapes` section enumerating 15 manifest and completion field keys with a producer and a wrapper obligation for each, asserted by `TestWrapperHostContractDocumentsManifestShapes`. Rationale: a pointer to a document that does not contain the detail is a paper claim, which CLAUDE.md's Definition of Done rejects — the detail must genuinely live elsewhere before a wrapper points at it.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib), no external test framework |
| Config file | none — plain `go test` |
| Quick run command | `go test ./cmd/... -run 'TestBuildWrapperCeremonyContract|TestContinueWrapperCeremonyContract|TestPlanWrapperCeremonyContract|TestPlanWrapperCardsParity|TestLifecycleCommandDocsPreferRuntimeCLI|TestBriefPathReferencedAcrossAllFourSurfaces|TestBuildWrapperVerbatimBriefBulletsStayMirrored|TestContinueWrapperSourcesUseFastDevContinue'` (confirmed runs in ~1.2s) |
| Full suite command | `go test ./cmd/...` (whole package; also run `go vet ./...` and `go build ./cmd/aether` per CLAUDE.md verification commands) |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CMD-01 | Each wrapper carries the D-05 stage skeleton (colony beat/Purpose/Reads/Spawns/Stop conditions) per stage | unit (string/structure assertions on markdown) | `go test ./cmd/... -run TestBuildWrapperCeremonyContract` (extend with new required-substring + proportion checks) | Partial — ✅ files exist for build/continue/plan; ❌ new assertions needed for stage-skeleton proportion, and ❌ no equivalent test exists for init.md at all (Wave 0 gap) |
| CMD-01 | `<success_criteria>/<failure_modes>/<read_only>` blocks present in all four | unit | new assertion group, e.g. `TestLifecycleWrappersCarryStructuredBlocks` | ❌ Wave 0 — does not exist in any current test file |
| CMD-01 | Named cross-stage state-carry contract present, using current (not retired) vocabulary | unit (required substring + forbidden retired vocabulary) | extend `platform_doc_hygiene_test.go` forbidden lists with retired `colony_depth` four-value terms | Partial — file exists, new forbidden entries needed |
| CMD-02 | Zero matches for envelope-parsing-as-primary-job phrasing across all four wrappers | unit (grep-style forbidden-string / regex count) | new test, e.g. `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob`, asserting near-zero count of phrases like `Parse \`result.` outside a single pointer-sentence per wrapper | ❌ Wave 0 — no test currently measures this as a proportion; today's tests only forbid specific *phrases*, not the *general pattern* CMD-02 describes |
| CMD-03 | Readability — a user can describe each stage's purpose without opening Go source | manual (human review) | Not mechanically testable; verify via `/gsd-verify-work`-style human read-through per the project's own admission that this criterion is prose-quality, not grep-quality | N/A — manual-only, justified: this is an inherently qualitative criterion about plain-English clarity, not a parseable invariant |
| CMD-04 | 8 named specialist commands unchanged | unit (full untouched-command hygiene suite + explicit diff-based regression check) | `go test ./cmd/... -run TestLifecycleCommandDocsPreferRuntimeCLI` (already covers `oracle.md`, `colonize.md`, others in its table) plus `git diff --stat -- .claude/commands/ant/chaos.md .claude/commands/ant/archaeology.md .claude/commands/ant/dream.md .claude/commands/ant/oracle.md .claude/commands/ant/swarm.md .claude/commands/ant/sage.md .claude/commands/ant/colonize.md .claude/commands/ant/council.md` returning empty before the phase's final commit | ✅ existing test covers `oracle.md`/`colonize.md`; ❌ no test covers `chaos.md`/`archaeology.md`/`dream.md`/`swarm.md`/`sage.md`/`council.md` directly — Wave 0 gap if the plan wants this mechanically enforced rather than just diff-checked at commit time |
| CMD-05 | build.md ownership header + reserved trailer marker present, merge order traceable | unit (required substring pair) | new test, e.g. `TestBuildMdOwnershipHandshake`, asserting both the Phase-160-fixed header comment and the `<!-- PHASE-168: ... -->` reserved marker exist in `.claude/commands/ant/build.md` and its `.opencode` mirror | ❌ Wave 0 — does not exist today |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run '<TestNameForThisTask>'` (targeted to whichever wrapper/test file the task touches)
- **Per wave merge:** `go test ./cmd/...` (full package) plus `go vet ./...`
- **Phase gate:** full suite green, plus `go build ./cmd/aether` succeeds, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/init_wrapper_ceremony_test.go` — new file; init.md has zero dedicated ceremony-contract test today (only the generic hygiene-list entry in `platform_doc_hygiene_test.go`). Should mirror `build_wrapper_ceremony_test.go`'s shape: required/inOrder/forbidden lists plus a heading-parity sub-test against `.opencode/commands/ant/init.md`.
- [ ] Extend `cmd/build_wrapper_ceremony_test.go` with: (a) a `level2Headings`-style parity test against `.opencode/commands/ant/build.md` (copy `TestPlanWrapperCardsParity`'s pattern), (b) the D-08 ownership-header/trailer-marker assertion, (c) a proportion assertion for stage-skeleton density (D-09's "prefer proportion/invariant assertions over named-section greps" — e.g., assert every `## ` stage heading is followed within N lines by a `**Purpose:**` or equivalent marker, so a future edit that drops the skeleton from one new stage fails without needing a new named-string check every time).
- [ ] Extend `cmd/continue_wrapper_ceremony_test.go` similarly (heading parity, stage-skeleton proportion).
- [ ] Extend `cmd/plan_wrapper_ceremony_test.go` / reuse `plan_wrapper_cards_test.go` machinery for stage-skeleton proportion (parity already covered).
- [ ] Decide and encode the flat-mirror-file test gap from Pitfall 1: either extend `TestBuildWrapperCeremonyContract`'s `wrapperPaths` to include `.claude/commands/ant-build.md` (matching what `plan`/`continue` already do) and add an equivalent init check, or add a new explicit test documenting why they're excluded.

*(No framework install needed — Go stdlib testing is already fully wired for this package.)*

## Security Domain

Not a primary concern for this phase (`security_enforcement` absent from `.planning/config.json`, treated as enabled, but this phase edits markdown instruction text with no new endpoints, auth, or data-handling surface). The one relevant control already in force and unchanged by this phase:

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation (prompt-injection style) | Indirectly | Pheromone content sanitization (SHA-256 dedup, angle-bracket escaping, instruction-override phrase rejection, 500-char cap) already exists per CLAUDE.md's Pheromone System section and is unrelated to wrapper markdown structure — no new validation surface is introduced by rewriting stage prose |
| V6 Cryptography | No | N/A — no crypto surface touched |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Prompt-injection-shaped example text inside a wrapper (D-8/D-10 regression) | Tampering/Spoofing (model instruction confusion) | Never quote verbatim user transcripts as worked examples in wrapper prose (v5.4.0's `build-prep.md:29-36` scar, explicitly forbidden by D-8) — restore the Context Confirmation Rule as neutral instruction only |
| Raw provider stdout/stderr/token leakage in wrapper-composed summaries | Information Disclosure | Already enforced by existing forbidden-string patterns and D-10; this phase must not weaken the "sanitized cause/next-action only" wording in any wrapper |

## Sources

### Primary (HIGH confidence)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/{build,continue,plan,init}.md` — read in full (current live content)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant-{build,continue,plan,init}.md` — read/diffed against canonical (flat mirror discovery)
- `/Users/callumcowie/repos/Aether/cmd/{build_wrapper_ceremony_test.go,continue_wrapper_ceremony_test.go,plan_wrapper_ceremony_test.go,plan_wrapper_cards_test.go,platform_doc_hygiene_test.go}` — read in full
- `go test ./cmd/... -run '...'` — executed directly, confirmed passing (1.192s)
- `/Users/callumcowie/repos/Aether/cmd/ceremony_cmd.go`, `cmd/codex_visuals.go` — grepped for actual Cobra `Use:` registrations and renderer function names
- `/Users/callumcowie/repos/Aether/.aether/docs/wrapper-host-contract.md`, `.aether/docs/wrapper-runtime-ux-contract.md` — read in full
- `/Users/callumcowie/repos/Aether/cmd/install_cmd.go`, `cmd/update_roundtrip_test.go`, `cmd/publish_cmd_test.go`, `cmd/update_cmd_test.go` — grepped/read to confirm flat-mirror install mechanics
- `.planning/phases/165-core-lifecycle-commands/165-CONTEXT.md`, `165-ARCHAEOLOGY.md` — read in full (upstream inputs, reproduced verbatim in User Constraints)
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md` — grepped for CMD-01..05 and Phase 165 accumulated context
- `git log`/`git show` — confirmed `ant-build.md`'s independent commit history at repo root, distinct from `ant/build.md`

### Secondary (MEDIUM confidence)
- None — all findings in this report were directly verified against repo source or test execution rather than inferred from web search (this is an internal-codebase research task with no external library/framework surface).

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack / current wrapper state: HIGH — every file and test cited was read directly in this session
- Architecture (ceremony ownership, CLI surface): HIGH — `ceremony_cmd.go`/`codex_visuals.go` grepped directly; corrects one archaeology imprecision (6 vs 4 actual ceremony subcommands)
- Pitfalls (flat mirror drift): HIGH — confirmed via direct `diff` and `git log`, not inferred
- Wording/voice choices for ceremony restoration: MEDIUM by design — CONTEXT.md explicitly delegates exact wording to Claude's discretion; this is not a gap, it's the intended freedom area

**Research date:** 2026-08-02
**Valid until:** ~14 days (fast-moving change zone — 52 commits on these four files since v5.4.0, "restored 7 times" per archaeology; re-verify current file state and test-suite pass/fail immediately before planning tasks if more than a few days have elapsed)
