# Phase 191: Dead Wood - Patterns

Reference patterns for executors. This phase is pure deletion/correction work; the patterns below are
about doing that safely, not about building anything new.

<pattern name="the-ratchet-house-style">
## The Ratchet House Style

This repo's established convention for "prove a deleted/fixed thing stays deleted/fixed" — established
across Phases 187-190, referenced directly by this phase's own planning brief.

### Two tiers, pick the one that matches the risk

**Tier 1 — simple existence check (use for criterion 1 and criterion 2's file deletions).**
When the thing being guarded against reappearing is a single file's presence, a plain `os.Stat`-based
check is correct and sufficient. Do not build an AST scanner for "does this file exist." Example shape:

```go
func TestColonyAgentsYAMLDoesNotReappear(t *testing.T) {
    root := repoRootForTest(t) // however this repo's tests locate repo root
    for _, name := range []string{"ambassador", "archaeologist", /* ...all 27... */} {
        p := filepath.Join(root, "colony", "agents", name+".yaml")
        if _, err := os.Stat(p); err == nil {
            t.Errorf("colony/agents/%s.yaml has reappeared -- this file was ruled zero-reader and deleted in Phase 191 (ROSTER-01/02); if it is back, either it has a new reader (update this ratchet with the evidence) or it should be deleted again", name+".yaml")
        }
    }
}
```

The failure message names the phase, the ruling, and what a future author must do (either update the
ratchet with fresh evidence of a reader, or delete the file again) — never just "this shouldn't exist."

**Fail-then-pass proof, mandatory for every new ratchet:** temporarily recreate the deleted file (or
`git stash`/copy it back in a scratch location the test reads), confirm the ratchet test FAILS, then
remove it again and confirm the ratchet PASSES. Record which assertion failed in the plan's SUMMARY.
A ratchet that has never been observed to fail is unproven — this is the same discipline
`187-VERIFICATION.md` established for the crash-safe worktree guards and that this phase's planning
brief invokes by name.

**Tier 2 — structural AST-based scanning (use only for the skill-lifecycle CLI-surface removal).**
`cmd/subcommand_reachability_ratchet_test.go` and `cmd/colony_state_atomicity_ratchet_test.go` both
parse Go source with `go/ast` rather than grepping, specifically because grep-based detection has
concrete, documented failure modes in this codebase:
- **Substring collision** (T-172-07): `"skill-list"` is a literal substring of `"skill-list-lifecycle"`.
  A naive grep-based scanner credits a documented call to the longer name as evidence for the shorter
  one. The existing test proves this with a synthetic fixture (lines 1152-1170) — read it before
  touching anything in this file, and do not remove that proof.
- **Structural placement blindness**: a write/call can live inside an ordinary `*ast.FuncDecl` OR inside
  an inline closure attached to a `&cobra.Command{...}` composite literal, with no enclosing
  `*ast.FuncDecl` at all. A scanner that only walks `FuncDecl` bodies misses the second shape silently.

This phase does not need a NEW AST scanner — `cmd/subcommand_reachability_ratchet_test.go`'s existing
orphan-detection machinery already does this job. The skill-lifecycle task's job is to **run it, trust
its output, and keep `testdata/orphan_allowlist.json` + the D-08 block internally consistent** with
what the scanner reports once the 8 commands are gone — not to rebuild scanning logic.

### The shrink-only allowlist idiom

`testdata/orphan_allowlist.json` + a `-update-orphan-allowlist` regeneration flag is this repo's
established pattern for "track a list of known exceptions, generated from the scanner's own real
output, never hand-typed." When this phase's skill-lifecycle task removes 8 commands, it removes their
allowlist entries the same way any allowlist entry is removed here: by `name` match, verified against
the scanner's own fresh output (re-run with `-update-orphan-allowlist` in a scratch/dry pass to see what
the tool itself would write, then apply the equivalent hand-edit and confirm `go test` agrees) — not by
guessing which JSON keys to delete.

</pattern>

<pattern name="baseline-capture-before-any-change">
## Baseline Capture Before Any Change

Applies to criterion 2's four CWD-relative loaders (Phase 189/190 established the general principle;
this phase gives it a concrete two-layer shape).

### Why two layers, not one

**Layer 1 — Go-level regression test, using the existing `*PathOverride` seam.** All four loaders
already expose a test-only override variable. Write (or extend) a test that:
1. Points the override at a temp copy of the CURRENT `colony/...` file content (captured before
   deletion).
2. Renders whatever output that loader feeds (a template section, a caste-identity string, a
   dispatch-contract field, a review-depth keyword list).
3. Points the override at a nonexistent path (simulating "file deleted").
4. Renders the SAME output using the (now-folded) Go compiled defaults.
5. Asserts both renders are identical.

This is fast, deterministic, and becomes a permanent regression lock — CI catches any future drift
between the compiled defaults and what a colony/ file might otherwise have said, forever, without
needing the real file to exist.

**Layer 2 — real dev-checkout CLI capture.** The ROADMAP criterion's literal wording is "dev-checkout
ceremony/visual output byte-identical before and after" — this specifically means running the compiled
(or `go run`) `aether` binary FROM THIS REPO'S ROOT, which is the only environment where the
CWD-relative default path (`colony/prompts/colony-prime.md`, etc.) ever resolves at all. Go's own test
runner sets the working directory to the package directory (`cmd/`) when running `cmd/`'s tests — the
bare relative path never resolves there, override or not. So Layer 1 alone does NOT prove the
ROADMAP's literal claim; Layer 2 is required in addition, not instead.

Concretely: before touching any file, run the real CLI surface each loader feeds (a build dispatch that
assembles a colony-prime-templated worker context; a command that renders caste-identity output; a
dispatch-contract-driven planning/survey description; a review-depth-driven verification-depth
selection) and save the output. Delete the file, run the identical command again, diff. Byte-for-byte,
not "looks the same." Record the diff (or its clean absence) in the plan's SUMMARY, matching the
established "record the real number, whether or not it's dramatic" discipline from Phase 189/190's own
measurement tasks.

### Order matters

Capture BEFORE editing anything — not as an afterthought, and not interleaved with the fold step. If a
mid-task mistake corrupts the fold, an already-saved baseline is what makes it possible to tell whether
the corruption changed real output or not. This is why criterion 2's plans structure baseline capture as
its own first step, distinct from the fold-and-delete step that follows it.

</pattern>

<pattern name="cli-surface-vs-logic-deletion">
## CLI-Surface Deletion Is Not Logic Deletion

The single most important discipline in this phase's dead-code task (criterion 3's skill-lifecycle
commands), stated as a reusable pattern because it will recur any time a "reachability ratchet" finds an
unreferenced *command*.

**The reachability ratchet measures one thing precisely: is this CLI subcommand ever invoked as a
subprocess** (typed by a human, shelled out by a wrapper's documented instructions, or scripted in a
test/CI file)? It says nothing about whether the *Go function* the command's `RunE` closure calls is
also called, separately, **in-process**, by other Go code that never goes through the CLI at all.

Before deleting any cobra.Command found by the reachability scanner:
1. Read its `RunE` closure and identify every non-trivial function it calls.
2. For each, grep for callers OUTSIDE the file defining the command and outside that file's own tests.
3. If a caller exists elsewhere in production code (not a test, not the command's own file), that
   function is load-bearing via a DIFFERENT calling convention than the CLI. Preserve it, unconditionally
   — delete only the `cobra.Command` registration and its `RunE` wrapper, never the shared function.
4. Only after the CLI wrapper is gone, re-check whether the now-orphaned callee has become genuinely
   unreachable (zero callers anywhere, including from other closures that were ALSO deleted in the same
   change) — and only then consider deleting it too.

This phase found exactly this shape: `matchSkillsForWorkflow`'s CLI wrapper (`skill-match`) has no
subprocess caller, but the function itself is called directly, in-process, from
`cmd/codex_build.go:3268` — the live worker-brief skill-injection path. Deleting the function because its
CLI wrapper looked orphaned would have silently broken that feature, passing every reachability test
(which only checks the CLI surface) while breaking the thing that actually matters.

</pattern>

<pattern name="prior-rulings-by-deletion">
## Prior Rulings-by-Deletion in This Repo's History

This phase is not the first time this codebase has resolved a "defines nothing" contradiction by
removing the thing that defined nothing, rather than building a caller for it after the fact:

- **`control-ts/` (Phase 160, RETIRE-01..04).** Self-described in its own `package.json` as a retired
  prototype. Deleted as a standalone first commit; `.aether/ts-host/` and `.aether/ts/` were verified
  explicitly UNCHANGED in the same phase, because they are load-bearing (`go build`, `aether publish`,
  `aether integrity` all depend on `ts-host/`) while `control-ts/` was not. Every test removed alongside
  it was recorded in a ledger as dead-with-no-replacement, or its coverage named as continuing in a
  specific surviving test — never silently dropped.
- **`findDispatchTask` (Phase 189, D-02)** and **`joinInternalWorkerSections` (Phase 190, D-10).** Both
  deleted outright once their last call site migrated, rather than left as harmless dead code — the
  established precedent this phase's own D-06/D-08 (cost.go, newLearningValidator) follow directly.
- **The hive double-injection removal (Phase 190, D-08/D-09).** The newer, narrower, single-consumer
  mechanism (`hive-injector.ts`) was deleted in favor of the older, already-multiply-consumed one
  (`resolveCodexWorkerContext()`), end to end — including the compiled `dist/` artifact, because a
  source-only edit to a tracked, compiled TypeScript output changes nothing at runtime until rebuilt.
  This phase's own criterion 2 does not involve a compiled-artifact risk (colony/ files are read
  directly by Go at runtime, not compiled), but the general lesson — **verify the actual runtime
  entrypoint reads what you think it reads, and rebuild anything compiled** — is worth carrying forward
  if any surprises turn up during execution.

The common thread: every prior ruling-by-deletion in this repo (a) named the specific thing kept and the
specific thing removed, with reasoning, not just "cleaned up"; (b) proved removal via a test that
FAILED before the fix and PASSED after (or, for the control-ts case, via a positive-existence test for
what survived); and (c) never silently expanded scope to nearby-but-unnamed dead code without recording
the decision to leave it (see Phase 191-CONTEXT.md's D-02/D-14 for this phase's own instances).

</pattern>

<pattern name="multi-surface-zero-readership-proof">
## Multi-Surface Zero-Readership Proof

Before any file in this phase is deleted, its zero-readership must be shown across every surface a
reader could exist on — not just a Go grep. This phase's planning pass already did this for every named
target (see 191-CONTEXT.md's `<domain>` section); executors extending or re-verifying this proof should
check, in order:

1. **Go source** (`cmd/`, `pkg/`) — including string-BUILT paths, not just literal path strings. A
   `filepath.Join("colony", "policies", someVariable+".yaml")` call site will not show up in a grep for
   the literal filename; check every caller of shared path-builder helpers (`policyPath`, etc.) with
   every possible argument, not just the ones already suspected.
2. **The TS host** (`.aether/ts-host/src/`) — a second runtime that reads its own files independently of
   Go; confirmed to have zero `colony/` references in this phase's targets, but re-check if extending
   scope beyond what 191-CONTEXT.md already covered.
3. **Wrapper markdown, all three platforms** (`.claude/commands/ant/`, `.opencode/commands/ant/`,
   `.codex/agents/`) — a wrapper can instruct an AI agent to read or shell out to something with no Go
   or TS code path involved at all. This is how `session-verify-fresh` turned out to be alive despite
   looking dead by Go-only grep (D-07) — the single most important lesson this phase's own
   reconnaissance produced.
4. **Skills** (`.aether/skills/`) — a `SKILL.md` can instruct the same way a wrapper can.
5. **The publish manifest** (`cmd/publish_cmd.go`, `cmd/install_cmd.go`) — does `aether publish` ship
   the file? If yes, deleting it changes every downstream repo on next `aether update`, a materially
   larger blast radius than a repo-local cleanup.
6. **Execution, not just text** — where a ratchet or smoke test can cheaply prove a command/flow still
   works after the change (the medic wrapper, `/ant-oracle`, `/ant-dream`), run it. Grep is a hypothesis;
   execution is the proof.

</pattern>
