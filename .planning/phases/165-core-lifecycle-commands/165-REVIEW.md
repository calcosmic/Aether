---
phase: 165-core-lifecycle-commands
reviewed: 2026-08-03T00:00:00Z
depth: standard
files_reviewed: 22
files_reviewed_list:
  - .aether/commands/build.yaml
  - .aether/commands/continue.yaml
  - .aether/commands/init.yaml
  - .aether/commands/plan.yaml
  - .aether/docs/wrapper-host-contract.md
  - .claude/commands/ant-build.md
  - .claude/commands/ant-continue.md
  - .claude/commands/ant-init.md
  - .claude/commands/ant-plan.md
  - .claude/commands/ant/build.md
  - .claude/commands/ant/continue.md
  - .claude/commands/ant/init.md
  - .claude/commands/ant/plan.md
  - .opencode/commands/ant/build.md
  - .opencode/commands/ant/continue.md
  - .opencode/commands/ant/init.md
  - .opencode/commands/ant/plan.md
  - cmd/build_wrapper_ceremony_test.go
  - cmd/continue_wrapper_ceremony_test.go
  - cmd/init_wrapper_ceremony_test.go
  - cmd/lifecycle_wrapper_contract_test.go
  - cmd/plan_wrapper_ceremony_test.go
findings:
  critical: 1
  warning: 7
  info: 7
  total: 15
status: issues_found
---

# Phase 165: Code Review Report

**Reviewed:** 2026-08-03
**Depth:** standard
**Files Reviewed:** 22
**Status:** issues_found

## Summary

Reviewed the four rewritten lifecycle wrapper specs (init/plan/build/continue) across the YAML sources, both platform wrapper copies, the flat installed mirrors, the wrapper-host contract doc, and the five Go contract/ceremony test files. Verified cross-platform parity mechanically: build.md, continue.md, and plan.md are byte-identical across all copies; init.md differs by exactly the one sanctioned AskUserQuestion line, and the test locking that delta works. All Phase 165 wrapper tests pass (`go test ./cmd -run '...WrapperCeremony|Lifecycle...|SpecialistCommand...'` — PASS).

The headline finding is a Critical one: init.md's Shelf Backlog stage still instructs the wrapper to hand-append promoted shelf items to `active_todos` "in the session file or colony state" — a direct hand-mutation of protected state that contradicts the file's own new `<read_only>` block, contradicts the guardrails, and is exactly the D-2 Frankenstein-state regression this phase's own test fence (`TestInitWrapperCeremonyContract`) was written to prevent. I traced the runtime (`cmd/shelf_init.go`) to confirm no Go path performs this write: `promoteShelfEntry` only flips shelf status, and `shelfEntryToTodo` has no callers. The remaining findings are internal contradictions between `<read_only>` blocks and guardrails, guardrail drift between the YAML sources and generated wrappers, ordering hazards, and test-quality gaps.

## Critical Issues

### CR-01: init.md instructs a hand-write to session file / colony state (D-2 Frankenstein-state regression path)

**File:** `.claude/commands/ant/init.md:257` (identical in `.opencode/commands/ant/init.md:257` and flat mirror `.claude/commands/ant-init.md:257`)
**Issue:** The Shelf Backlog stage says:

> "Promoted items become todos: append `[shelf:{category}] {text}` to `active_todos` in the session file or colony state"

This is an instruction to the LLM wrapper to directly edit `.aether/data/session.json` or `COLONY_STATE.json`. It directly contradicts:
- The same file's `<read_only>` block (lines 51-58: "This wrapper never writes these files by hand... session.json... COLONY_STATE.json")
- The guardrail in `.aether/commands/init.yaml:42` ("Do not write colony state files, session files, or pheromone files by hand")
- The repo-wide Protected Paths policy (`.claude/rules/aether-colony.md`)

I verified the runtime provides no alternative: `cmd/shelf_init.go`'s `promoteShelfEntry` only updates the shelf file's entry status/`PromotedTo`, and `shelfEntryToTodo` (which produces exactly this `[shelf:{category}] {text}` string) has zero call sites. So either the LLM follows this line and hand-mutates protected state — the documented state-corruption bug class — or promoted items silently never become todos. Note also that this line evades the phase's own regression fence: `TestInitWrapperCeremonyContract`'s forbidden list (`Write COLONY_STATE.json`, `aether state-write`, etc.) does not match this phrasing.

**Fix:** Remove the hand-append instruction and route the todo creation through the runtime. Either make `aether shelf-promote-batch` (or `aether init`) append the todos itself using the already-existing `shelfEntryToTodo`, or drop the "become todos" claim until a runtime command exists. Then add the phrase to the forbidden fence, e.g.:

```go
// cmd/init_wrapper_ceremony_test.go — forbidden slice
forbidden := []string{
    ...,
    "append `[shelf:",           // no wrapper-side todo writes
    "to `active_todos`",
}
```

## Warnings

### WR-01: `<read_only>` blocks say "may read" while guardrails say "Do NOT read" — internal contradiction in build.md and continue.md

**File:** `.claude/commands/ant/build.md:50-52` vs `:255`; `.claude/commands/ant/continue.md:190-192` vs `:205` (and byte-identical .opencode copies + flat mirrors)
**Issue:** build.md's `<read_only>` block: "This wrapper may read but never write: colony state, session files, and pheromone files." Its Guardrails section: "Do NOT read or write colony state files by hand." Same conflict in continue.md. The YAML sources (`build.yaml:63`, `continue.yaml:77`) both say "Do NOT read or write... by hand," so the `<read_only>` blocks also drift from their sources. plan.md got this right ("This wrapper never reads or writes, by hand: ..."). An LLM given both instructions has no deterministic resolution; one path licenses direct reads of `COLONY_STATE.json`, which the guardrail forbids.
**Fix:** Align the `<read_only>` wording with plan.md's shape: "This wrapper never reads or writes these files by hand; state is observed only through `aether status` / runtime commands." Update both platform copies and mirrors together (tests enforce byte-parity).

### WR-02: continue.md dropped several guardrails present in its YAML source

**File:** `.claude/commands/ant/continue.md:200-209` vs `.aether/commands/continue.yaml:72-86`
**Issue:** The YAML source lists guardrails absent from the generated wrapper: `Do NOT run aether continue --synthetic...` (yaml:75), `Do NOT parse visual output as authoritative state` (yaml:79), the reviewer read-loop fence (yaml:82), the extra-option-menus fence (yaml:83), and the hand-render-ceremony fence (yaml:84). Notably "Do NOT parse visual output as authoritative state" is present in build.md and plan.md guardrails but missing from continue.md entirely (grep confirms zero occurrences). The YAML's own `drift_guard` requires updating "this YAML, Claude/OpenCode wrappers, the Codex skill, and cmd/command_guide.go together" — this is drift within a single phase's deliverable.
**Fix:** Add the missing guardrails (at minimum the `--synthetic` and parse-visual fences) to continue.md's Guardrails section on both platforms, or prune them from continue.yaml with a stated reason if intentionally retired.

### WR-03: Build ceremony test never requires `build-completion-stage` — the staging step is unfenced

**File:** `cmd/build_wrapper_ceremony_test.go:21-54`
**Issue:** `TestBuildWrapperCeremonyContract` requires `build-finalize` and the closeout, but nothing in the required list mentions `aether build-completion-stage` or `result.completion_path`. The staging step is the load-bearing safety mechanism (`build.yaml:62`: "Before build-finalize, call build-completion-stage exactly once... pass only its Go-owned completion_path"). A future edit could delete the entire staging block from build.md — reverting to finalizing the raw wrapper temp file — and every test would stay green. Per the project's Definition of Done, a requirement needs a command that fails when it is unmet.
**Fix:** Add to the `required` slice:

```go
"AETHER_OUTPUT_MODE=json aether build-completion-stage $ARGUMENTS --completion-file",
"result.completion_path",
```

and add `"aether build-completion-stage"` between the manifest fetch and `build-finalize` in the `inOrder` slice.

### WR-04: plan.md checks boundary guidance only after Decision Moment 2 has mutated research-approval state

**File:** `.claude/commands/ant/plan.md:50-78`
**Issue:** The stage order is Decision Moment 1 → Planning Manifest → Decision Moment 2 (runs `aether plan-research-approve --approve-all/--flip/--auto`, a state-mutating command) → Clarification Gate (`orchestrator_boundary_guidance` / `unresolved_clarifications`). If boundary guidance is active in the first manifest, the wrapper still walks the user through both decision moments and applies research approvals before discovering it must route to `aether discuss` and discard the manifest. build.md gates the boundary immediately after manifest fetch and before anything else acts on the manifest (build.md:119-131). At best wasted user interaction; at worst research approvals recorded against a plan the discuss redirect is about to invalidate.
**Fix:** Move the Clarification Gate to immediately after the first manifest fetch (before Decision Moment 1's card is presented), or add an explicit line in both decision moments: "If the manifest carries active `orchestrator_boundary_guidance`, skip both decision moments and go straight to the Clarification Gate."

### WR-05: init Approval writes pheromones before the colony exists

**File:** `.claude/commands/ant/init.md:283-284` (same ordering in `.aether/commands/init.yaml:23-24`)
**Issue:** Approval says: run `aether pheromone-write` for each approved pheromone, *then* run `aether init` to create colony state. If `aether init` fails (setup missing, sealed-colony conflict, bad charter JSON), approved pheromones are already persisted against the previous or nonexistent colony — a partial-persistence state the failure modes section claims cannot happen ("Write nothing — no charter call, no pheromone writes"). If `aether init` resets or replaces `pheromones.json` on colony creation, the approved signals are silently destroyed instead.
**Fix:** Reverse the order: run `aether init` first, and write approved pheromones only after it reports success. Update init.yaml, both wrapper copies, and the flat mirror together.

### WR-06: Fragile shell-quoting templates in init command invocations

**File:** `.claude/commands/ant/init.md:16` and `:284`
**Issue:** Line 16: `aether init-research --goal "$ARGUMENTS" --target .` — a goal containing a double quote breaks out of the argument. Line 284: `--charter-json '<synthesized charter JSON>'` — the charter is AI-synthesized prose (intent, vision, constraints) and will routinely contain apostrophes ("don't break the API"), which terminate the single-quoted argument and at minimum fail the command, at worst execute trailing text as shell. The LLM composes this command from user-influenced text, so the template it copies matters.
**Fix:** Instruct writing the charter JSON to a temp file and passing a path (`--charter-file <path>`, adding runtime support if needed), or explicitly instruct the wrapper to shell-escape embedded quotes before substitution.

### WR-07: build.md partial-wave-failure policy is ambiguous

**File:** `.claude/commands/ant/build.md:44-45` vs `:184`
**Issue:** `<failure_modes>` says "Wave failure mid-build: do not continue to the next wave" (reads as: any failure in a wave halts). The Worker Spawning stop condition says "All workers in a wave fail — do not continue to the next wave" (reads as: only a total wave wipeout halts). Whether a wave with 1 of 3 workers failed proceeds to the next wave is exactly the "failed dependencies cascade" scenario both passages warn about, and the two passages answer it differently.
**Fix:** Pick one policy and state it identically in both places, e.g. "If any worker in a wave ends `failed` or `blocked`, do not start the next wave; surface the failure and let `build-finalize` judge the partial packet."

## Info

### IN-01: Dead helper `sliceBetweenMarkers` in continue ceremony test

**File:** `cmd/continue_wrapper_ceremony_test.go:270-289`
**Issue:** `sliceBetweenMarkers` has no call sites anywhere in `cmd/` (definition only). Dead test code.
**Fix:** Delete it, or use it where section-scoped assertions were intended.

### IN-02: Stale "currently RED" comment on a now-green test

**File:** `cmd/lifecycle_wrapper_contract_test.go:129-134`
**Issue:** The `TestLifecycleFlatMirrorsMatchCanonical` doc comment says "This is currently RED for build and init... Task 3 of this plan resyncs build and init and turns this fully green." The mirrors now match (test passes); the comment describes a completed TDD state as present-tense.
**Fix:** Reword to past tense: "Was RED for build and init when written; Task 3 resynced them."

### IN-03: Ratio constant duplicated as magic number `3` in three test files

**File:** `cmd/lifecycle_wrapper_contract_test.go:267` vs `cmd/build_wrapper_ceremony_test.go:324`, `cmd/continue_wrapper_ceremony_test.go:236`, `cmd/plan_wrapper_ceremony_test.go:198`
**Issue:** `wrapperMinMethodToEnvelopeRatio = 3` exists with a `wrapperRatioOK` helper, but the three per-verb `method_outweighs_envelope_mechanics` subtests hardcode `methodCount < 3*envelopeCount` instead. Changing the constant would silently leave four divergent thresholds. These subtests also fully duplicate `TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob` and the structured-blocks assertions in `TestLifecycleWrappersCarryStructuredBlocks`.
**Fix:** Have the per-verb subtests call `wrapperRatioOK`, or delete the duplicated subtests and rely on the shared lifecycle test.

### IN-04: Default continue command duplicates `--verification-depth` when the user supplies one

**File:** `.aether/commands/continue.yaml:5` and `.claude/commands/ant/continue.md:59`
**Issue:** `aether continue --verification-depth standard $ARGUMENTS` — if the user's arguments include `--verification-depth light`, the runtime receives the flag twice. This currently works only because Cobra/pflag string flags are last-wins (`cmd/codex_workflow_cmds.go:1187` registers a plain `String` flag). A stricter parser or a switch to a slice flag would break the override silently.
**Fix:** Instruct the wrapper to omit the hardcoded `--verification-depth standard` when `$ARGUMENTS` already names a depth.

### IN-05: Vacuous negative control in flat-mirror test

**File:** `cmd/lifecycle_wrapper_contract_test.go:164-169`
**Issue:** `a_mirror_content_change_would_fail` appends a sentinel to the canonical bytes and asserts the mutated copy differs from the mirror. Since mutated = canonical + sentinel, this is true by construction whenever the outer assertion passed; it proves nothing beyond `bytes.Equal` working. Contrast with the ratio test's synthetic negative control, which genuinely exercises the assertion logic.
**Fix:** Drop the subtest, or make it compare `mutated` against `canonicalBytes` through the same code path a real drift would take.

### IN-06: continue.yaml step 9 path wording is self-contradictory and absent from the wrapper

**File:** `.aether/commands/continue.yaml:50`
**Issue:** "Write per-reviewer JSON and the final completion JSON under `${TMPDIR:-/tmp}/aether-continue-<run>/continue-completion.json`" — per-reviewer files cannot live "under" a `.json` file path; the directory is presumably meant. The generated continue.md drops the concrete path entirely (only "temporary worker JSON file"), so the two describe different levels of specificity.
**Fix:** In the YAML, name the directory (`${TMPDIR:-/tmp}/aether-continue-<run>/`) for per-reviewer files and the `continue-completion.json` file within it for the packet.

### IN-07: build.md describes `--light` as "skip review agents," contradicting the depth table

**File:** `.claude/commands/ant/build.md:237`
**Issue:** "Use `--heavy` for full gates or `--light` to skip review agents." Per CLAUDE.md's Queen-owned orchestration table, build light is "Builder + Watcher + Probe" — review agents still run, just fewer of them. Runtime wins over docs, but this wrapper line teaches users the wrong mental model.
**Fix:** Reword: "`--light` runs the minimal review set (Watcher + Probe only)."

---

_Reviewed: 2026-08-03_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
