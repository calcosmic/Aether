---
phase: 160-fail-loudly
plan: 02
subsystem: dead-documentation-corpus
tags: [documentation-correctness, cli-contract-drift, invariant-test]
dependency-graph:
  requires: []
  provides:
    - "six corrected CLI call sites in .aether/docs/command-playbooks/*.md and colony/playbooks/build.md"
    - "TestSurveyLoadAbsentAndUncalled invariant test"
  affects:
    - "Plan 07's audit (unblocked — the six call sites it checks are no longer wrong)"
tech-stack:
  added: []
  patterns:
    - "Go static-assertion test idiom (read repo files, assert absent string, report file:line)"
    - "repoRootForCommandSourceTest() reused from cmd/command_source_hygiene_test.go"
key-files:
  created:
    - cmd/survey_load_absence_test.go
  modified:
    - colony/playbooks/build.md
    - .aether/docs/command-playbooks/build-prep.md
    - .aether/docs/command-playbooks/build-context.md
    - .aether/docs/command-playbooks/build-wave.md
    - .aether/docs/command-playbooks/build-full.md
    - .aether/docs/command-playbooks/build-complete.md
    - .aether/docs/command-playbooks/continue-verify.md
decisions:
  - "generate-progress-bar, skill-detect, state-checkpoint, print-next-up, verify-claims call sites rewritten to the flag/no-arg forms their cobra commands actually accept; the Go commands themselves were not touched (RESEARCH.md Pitfall 3: fix the call site, not the command)"
  - "survey-load was resolved by deletion, not correction — no substitute command was invented; the instruction was rewritten as prose plus a note that no such subcommand exists"
  - "print-next-up call sites in build-full.md and build-complete.md had their now-unused jq-derived state/current_phase/total_phases variable assignments removed rather than left as dead code, since print-next-up takes no arguments and reads colony state itself"
metrics:
  duration_minutes: 35
  tasks_completed: 2
  files_changed: 8
  completed: 2026-07-27
---

# Phase 160 Plan 02: Fix Six Broken CLI Call Sites Summary

Corrected six documented CLI invocations across the dead playbook corpus to match their commands' actual cobra contracts, and added an invariant test that fails if `survey-load` — a command deleted entirely from the Go runtime — ever reappears as an instruction or a registration.

## What This Actually Changes For A User

**Nothing, today.** Every file this plan edited lives in `.aether/docs/command-playbooks/` and `colony/playbooks/` — directories that nothing in the running system loads. The loader that used to read these files (`playbook-loader.ts`) was deleted in commit `b2b41486`. If you run `/ant-build` or `/ant-continue` right now, before and after this plan, you get byte-for-byte the same behavior, because neither command reads these files. This plan makes the *documentation correct* so a future contributor (or the audit in Plan 07) is not misled about what these commands accept, and so that if/when this corpus is ever reconnected to a live loader, it will not immediately fail six calls the way it would have yesterday.

## What Was Fixed

Six call sites, each rewritten to conform to the real cobra contract confirmed by reading the Go source directly (not by guessing):

1. **`colony/playbooks/build.md`** — deleted the `aether survey-load "{phase_name}"` instruction. This command does not exist in any form (positional or flag) in the Go runtime; RESEARCH.md confirms it was removed outright rather than corrected. Replaced with prose describing the same intent (load survey docs by phase type) plus a parenthetical noting no such subcommand exists.
2. **`build-prep.md`** — `aether generate-progress-bar "$current_phase" "$total_phases" 20` → `aether generate-progress-bar --current "$current_phase" --total "$total_phases" --width 20`.
3. **`build-context.md`** — `aether skill-detect "$(pwd)"` → `aether skill-detect` (the command resolves the hub path itself; it never accepted a directory argument).
4. **`build-wave.md`** — `aether state-checkpoint "pre-build-wave"` → `aether state-checkpoint --name "pre-build-wave"`.
5. **`build-full.md`** and **`build-complete.md`** — `aether print-next-up "$state" "$current_phase" "$total_phases"` → `aether print-next-up` (no arguments; reads colony state itself). Removed the now-unused `jq`-derived variable assignments that fed the old positional arguments.
6. **`continue-verify.md`** — `aether verify-claims ".aether/data/last-build-claims.json" "<watcher_json_or_path>" "<test_exit_code>"` → `aether verify-claims` (no arguments). Rewrote the accompanying sentence: the command reads `COLONY_STATE.json` directly and runs fixed internal consistency checks — it does not compare an external claims file against watcher output and a test exit code, a feature that was never built.

`2>/dev/null` was removed from the three lines being edited that carried it (generate-progress-bar, skill-detect, state-checkpoint), since the line was being rewritten anyway and leaving the suppression would re-hide the call being fixed. The `|| echo "..."` fallback behavior was preserved on each. The remaining ~176 `2>/dev/null` occurrences elsewhere in the corpus were left untouched — out of scope per RESEARCH.md Pitfall 2 and reserved for Plan 03.

Each of the six edited `.aether/docs/command-playbooks/*.md` files (build-prep, build-context, build-wave, build-full, build-complete, continue-verify) gained a one-sentence banner near the top: "Reference documentation only. As of Phase 160, this file is not loaded or executed by the runtime — see CLAUDE.md's 'Command Playbooks (Reference Material)' section." This mitigates threat T-160-05 — a reader mistaking corrected call syntax for restored runtime behavior.

`colony/playbooks/build.md` did not get this banner (it is a different, separately dead third-copy file per the plan's scope note, and the plan only specified banners for the `.aether/docs/command-playbooks/` files).

## Invariant Test

`cmd/survey_load_absence_test.go` adds `TestSurveyLoadAbsentAndUncalled` with two subtests:

- **`NotRegistered`** — walks `rootCmd.Commands()` and fails if any command's `Name()` or `Aliases` list contains `survey-load`.
- **`NotReferenced`** — reads every `.md`/`.yaml` file in `.claude/commands/ant/`, `.opencode/commands/ant/`, `.aether/docs/command-playbooks/`, `.aether/commands/`, and `colony/playbooks/` (paths resolved via the existing `repoRootForCommandSourceTest()` helper) and fails if any line contains the literal text `aether survey-load`, reporting every offending file and line number.

Manual verification performed as required by the plan's acceptance criteria: temporarily appended `` Run `aether survey-load "x"` `` to `colony/playbooks/build.md`, re-ran the test, and confirmed it failed with:

```
survey_load_absence_test.go:105: found 1 reference(s) to "aether survey-load", a command with no runtime implementation:
  colony/playbooks/build.md:180: Run `aether survey-load "x"`
```

The line was then reverted via `git checkout -- colony/playbooks/build.md`, and the test was re-run to confirm it passes again (it does — see verification output below). No trace of the temporary line remains in any commit.

The test file contains no `rootCmd.Execute()` call and no `newTestStore` usage — it is a pure static/registry assertion, as required. It also avoids containing the literal contiguous string `aether survey-load` in its own source (built via string concatenation instead), so a future whole-repo grep for that phrase cannot false-positive on the test file itself.

## Verification

```
go test ./cmd -run TestSurveyLoadAbsentAndUncalled -count=1   → ok
go test ./cmd -run TestCLIFlagAudit -count=1                  → ok (125 subcommands found in markdown, 371 registered)
go vet ./cmd                                                  → clean
grep -rn 'aether survey-load' colony/ .aether/ .claude/ .opencode/  → no matches
git diff --stat -- .claude/commands/ant/build.md              → empty (Phase 165 ownership boundary honoured)
git diff --name-only -- cmd/                                  → only cmd/survey_load_absence_test.go
go build ./cmd/aether                                          → succeeds
```

## Deviations from Plan

None — plan executed exactly as written. The one judgment call made within the plan's own instructions: for the two `print-next-up` sites, the plan said to change only the invocation and "the surrounding sentence where the old wording described arguments." There was no explanatory sentence there, only three `jq`-derived shell variable assignments that fed the old positional arguments. Leaving those assignments in place would have produced dead, misleading code (three variables computed and then never used). They were removed as part of the same call-site fix rather than left as clutter — this is a minimal-diff judgment call within the task's stated scope, not a structural change to the surrounding section.

## Known Stubs

None. No UI, no data rendering — this plan only edits documentation text and adds a Go test.

## Threat Flags

None. The one new surface introduced — a Go test that reads markdown/YAML files under specific repo-relative directories — reads only repository-controlled files and performs no writes, matching the plan's own threat model (`markdown corpus → test process` boundary).

## Self-Check: PASSED

All 8 created/modified files confirmed present on disk. Both task commits (`f60a4c31`, `095de72b`) confirmed present in `git log --oneline --all`.
