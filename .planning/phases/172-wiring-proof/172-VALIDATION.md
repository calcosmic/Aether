---
phase: 172
slug: wiring-proof
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-08
---

# Phase 172 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `172-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go built-in `testing` package (no third-party test framework in `cmd/`) |
| **Config file** | none — Go tests need no config; CI invocation lives in `.github/workflows/ci.yml` ("Run Go tests" step) |
| **Quick run command** | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced\|TestOrphanAllowlist\|TestCommandCallsMatchCobraContracts\|TestSpawnCanSpawn' -v` |
| **Full suite command** | `go test ./... -count=1 -timeout 900s` |
| **Estimated runtime** | ~15s quick · ~300–600s full suite |

---

## Sampling Rate

- **After every task commit:** Run the quick run command above
- **After every plan wave:** `go test ./cmd/... -count=1 -timeout 900s -v` — the whole `cmd`
  package, because these tests mutate/read the shared `rootCmd` global and ordering side
  effects must surface early
- **Before `/gsd-verify-work`:** Full suite green AND the new named CI step(s) from D-15 green
- **Max feedback latency:** 15 seconds for the quick command

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map is keyed by requirement until plans exist.
The planner MUST extend this table with concrete task IDs.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 1 | WIRE-01 | — | A registered subcommand with no caller outside its own definition file fails the ratchet, naming the command | unit (AST static analysis + fixture) | `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced -v` | ❌ W0 — new `cmd/subcommand_reachability_ratchet_test.go` | ⬜ pending |
| TBD | TBD | 1 | WIRE-01 | — | The orphan allowlist may only shrink against the committed baseline; any addition hard-fails (D-10, D-11) | unit (baseline diff) | `go test ./cmd -run TestOrphanAllowlistOnlyShrinks -v` | ❌ W0 — same new file | ⬜ pending |
| TBD | TBD | 1 | WIRE-01 | — | Fixture proof: a synthetic orphan command registered in-test makes the ratchet fail, naming it | unit (fixture self-test) | `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced -v` | ❌ W0 — same new file | ⬜ pending |
| TBD | TBD | 1 | WIRE-01 (D-12) | — | The flag audit's `skipSubcommands` map gets the same shrink-only treatment, so exemption pressure cannot relocate to an unguarded list | unit | `go test ./cmd -run TestFlagAuditSkipListOnlyShrinks -v` *(name at planner discretion)* | ❌ W0 — extends `cmd/cli_flag_audit_test.go` | ⬜ pending |
| TBD | TBD | 1 | WIRE-02 | — | `aether spawn-can-spawn 5 --enforce` exits 0 — the exact string `.aether/workers.md:292` instructs | integration (subprocess or in-package `RunE`) | `go test ./cmd -run TestSpawnCanSpawnAcceptsDocumentedInvocation -v` *(name at planner discretion)* | ❌ W0 — `cmd/spawn_test.go` or new file | ⬜ pending |
| TBD | TBD | 1 | WIRE-02 (D-13) | — | `--enforce` carries real semantics: a deny answer produces a non-zero exit. Unreachable today (`can_spawn` hardcoded true) but the wiring must exist and be asserted | unit | test drives the deny path directly / asserts the error-exit wiring | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | WIRE-03 | — | `.aether/workers.md:292`'s `--enforce` is caught as unregistered before the WIRE-02 fix, and passes after | unit (corpus extension of an existing test) | `go test ./cmd -run TestCommandCallsMatchCobraContracts -v` | ✅ exists — `cmd/command_call_audit_test.go` | ⬜ pending |
| TBD | TBD | 2 | WIRE-03 | — | A `.aether` markdown flag violation names **file, line, and flag** | unit | same command as above | ✅ exists — `validateCallAgainstCobra` already emits `unknown flag --%s` with file:line | ⬜ pending |
| TBD | TBD | 3 | WIRE-01, WIRE-03 (D-15) | — | Both checks run inside the CI command the release gate already runs, under their own named step, so a failure reads as a wiring problem | CI behavioural proof | delete a caller → observe the gate go red (NOT by reading the workflow file) | ❌ W0 — `.github/workflows/ci.yml` | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/subcommand_reachability_ratchet_test.go` — new file carrying
      `TestNoRegisteredSubcommandIsUnreferenced`, the shrink-only allowlist assertion, and a
      fixture self-test proving the ratchet detects a synthetic orphan. Mirror the existing
      function-local `AddCommand` / `defer RemoveCommand` pattern from
      `TestAuditDetectsPositionalDrift` so the fixture cannot permanently trip the ratchet it tests.
- [ ] `cmd/testdata/orphan_allowlist.json` — committed baseline, **seeded from the scanner's
      real output**, not from the assumed count (D-07). Research found the true orphan count is
      likely 6, not 8: `skill-parse-frontmatter` and `skill-cache-rebuild` are genuinely invoked
      from `.claude/commands/ant/skill-create.md:220-242`.
- [ ] Extend `cmd/cli_flag_audit_test.go` — shrink-only guard over the currently-unguarded
      2-entry `skipSubcommands` map (D-12).
- [ ] `cmd/spawn.go` — register `--enforce`, relax `Args` from `cobra.NoArgs` to accept the
      documented positional depth (D-14), wire deny → non-zero exit using the existing
      `outputError` deferred-exit-code convention already used by sibling `spawnLogCmd`.
- [ ] `cmd/command_call_audit_test.go` — extend `auditedCorpora` to bring `.aether` markdown
      into scope (see Open Scope Call below).
- [ ] `.github/workflows/ci.yml` — named step(s) per D-15, near the existing
      "Verify command catalog classification" step.
- [ ] Framework install: **none** — `go test` is already the toolchain.

---

## Open Scope Call (blocks the WIRE-03 corpus definition)

Success criterion 3 says `.aether/*.md`. D-06 additionally locks
`.aether/docs/command-playbooks/*.md` **into** flag-audit scope. So the corpus is at minimum
top-level `.aether/*.md` **plus** the playbooks directory. Whether it extends to all of
`.aether/**/*.md` (~250 tracked files) is the open call recorded in `172-RESEARCH.md`
Open Question 1 and resolved before planning.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The release gate actually goes red when a caller disappears | WIRE-01, criterion 4 | Criterion 4 explicitly forbids proving this by reading the workflow file. It is a claim about CI's live behaviour, which no in-repo Go test can assert about itself. | On a scratch branch: delete a caller of a currently-called command (e.g. remove the wrapper that names it), push, observe the named CI step fail and name the command. Revert. |

*Everything else in this phase has automated verification — which is the point of the phase.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s for the quick command
- [ ] The allowlist baseline was seeded from real scanner output, not the assumed count
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
