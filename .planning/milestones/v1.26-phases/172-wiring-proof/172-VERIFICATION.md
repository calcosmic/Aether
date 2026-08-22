---
phase: 172-wiring-proof
verified: 2026-08-12T16:10:00Z
status: passed
score: 4/4 roadmap success criteria hold in substance (2 hold as written; 2 hold only under narrower wording, with the residue named explicitly per 172-STOP-RULE.md)
overrides_applied: 1
overrides:
  - must_have: "Success criteria 1 and 4 hold exactly as originally worded (absolute: 'the allowlist may only shrink', 'the ratchet ... runs ... and blocks')"
    reason: >
      172-STOP-RULE.md, agreed with the user on 2026-08-12, is itself the governing
      instruction for this verification: round 4's own code review found CR-06, a
      NEW class of bypass (not among the five defects round 4 was scoped to close).
      Per the stop rule's own binding text, this triggers narrowing of criteria 1
      and 4 rather than a round 5. The narrower wording proposed below is what this
      verification found to be actually, independently proven; the residue (CR-06)
      is named explicitly rather than left implied, and a follow-up phase is
      recommended to carry it forward as a tracked limit.
    accepted_by: "user (via 172-STOP-RULE.md, agreed 2026-08-12)"
    accepted_at: "2026-08-12T14:16:00Z"
re_verification:
  previous_status: gaps_found
  previous_score: "2/4 (criteria 2 and 3 verified; criterion 1 and criterion 4 each had one independently-confirmed defect: CR-04/CR-05 and CR-01/CR-02/CR-03)"
  gaps_closed:
    - "CR-01 (job-level continue-on-error invisible to the gate check) — closed by gateJobAllowedKeys whitelist + auditWorkflowShape"
    - "CR-02 (env: GOFLAGS at job/step/workflow scope) — closed at all three scopes by the same whitelist mechanism"
    - "CR-03 (paths-ignore under on: triggers) — closed by releaseTriggerAllowedKeys whitelist rejecting any trigger key by absence, not by name"
    - "CR-04 (shrink-only guard compared bare leaf names, letting a same-leaf newcomer through) — closed by TestPathMigrationDidNotWidenTolerance's rewrite to full-path comparison, proven by TestPathMigrationRejectsASameLeafNewcomer including a generality loop over all 278 frozen leaves"
    - "CR-05 (frozen pre-migration anchor file had no integrity pin) — closed by preMigrationSnapshotSHA256 + TestPreMigrationSnapshotIsFrozen"
  gaps_remaining:
    - "CR-06 (new, this round's own review): the workflow-shape whitelist, the exact-equality run-line pin, and the execution harness all inspect only the two named gate steps' own YAML/text. None of the other ~18 steps in the same `go:` job (Checkout, Setup Go, Setup Node, Install goreleaser, Validate goreleaser config, Build, Vet, and the rest) is inspected at all. Any of them can write GOFLAGS/GOTOOLCHAIN to $GITHUB_ENV or shadow the go binary on $PATH — both standard, documented GitHub Actions mechanisms available to any step sharing the job's runner and filesystem — reproducing CR-02's outcome by a route none of this phase's three mechanisms was built to see. Per 172-STOP-RULE.md this is a NEW class, not one of the five the round was scoped to close, so it is not fixed in this round; it is named as residue in the narrowed criterion 4 below and should be carried by a tracked follow-up phase."
  regressions: []
gaps: []
deferred:
  - truth: "The release gate cannot be silently switched off by any route (absolute form of success criterion 4)"
    addressed_in: "Recommended new follow-up phase (none currently scheduled in ROADMAP.md) — CR-06's fix options are architectural (move the gate step to its own job/workflow with nothing untrusted ahead of it) or an extension of the shape whitelist to scan every step's run: block for $GITHUB_ENV/$GITHUB_PATH writes, per 172-REVIEW.md's Fix section"
    evidence: "172-STOP-RULE.md step 3: 'Open a tracked follow-up phase carrying the residue, so later phases that depend on this ratchet inherit a documented limit rather than a false guarantee.' No phase in the current ROADMAP.md yet claims this residue — it is not deferred to an already-planned phase, it is flagged here for the user/orchestrator to schedule one."
---

# Phase 172: Wiring Proof Verification Report (Round 4 — FINAL, per agreed stop rule)

**Phase Goal:** A capability added by this milestone cannot ship without a caller. The orphan ratchet exists, runs in CI, and blocks — before any of the capabilities it constrains are built, so it is shaped by the standard rather than by whatever shipped.
**Verified:** 2026-08-12T16:10:00Z
**Status:** passed (governed by 172-STOP-RULE.md — see rationale below)
**Re-verification:** Yes — fourth round, after gap-closure plans 172-12 and 172-13

## Why status is `passed` and not `gaps_found`

This phase has run three prior build-verify rounds, each closing real defects
and each review finding new ones over the same adversarial surface (GitHub
Actions workflow semantics). The user agreed a binding stop rule
(`172-STOP-RULE.md`, 2026-08-12) before this round started: round 4 is the
**last** build round under criteria 1 and 4 as originally (absolutely)
worded. If round 4's review found a NEW class of bypass — not among the five
defects (CR-01..CR-05) the round was scoped to close — the phase closes
against **narrowed** criteria instead of triggering a round 5.

Round 4's own code review (`172-REVIEW.md`) found exactly this: CR-01
through CR-05 are all independently confirmed CLOSED (I re-ran all 34 named
guard tests myself — all green — and independently re-hashed the pinned
anchor file, matching the constant in source). But the review also found
**CR-06**, a genuinely new class of bypass (steps other than the two named
gate steps can poison the job's environment or toolchain, unaudited by any
of this phase's three mechanisms). I independently confirmed CR-06's
structural premise by reading `auditWorkflowShape` end to end: the
step-scope whitelist loop (`cmd/ci_wiring_gate_test.go:1427-1454`) iterates
only over `[]string{blanketGateStepName, wiringGateStepName}` — the two
named gate steps — and never inspects any other step's `run:` content. I
also independently confirmed `.github/workflows/ci.yml` has 18 steps
(Checkout through "Verify command catalog classification") ahead of the
first gate step ("Run Go tests", line 41) and the wiring step (line 99),
none of which any guard in this phase scans.

Per the stop rule's own binding instruction, this is the STOP condition:
narrow criteria 1 and 4 to name the residue explicitly, mark the phase
complete against the narrowed criteria, and recommend a tracked follow-up
phase for the residue — not plan a round 5. That is what this report does.

## Goal Achievement

### Observable Truths — the four ROADMAP Success Criteria

| # | Criterion (paraphrased) | Judgment | Evidence |
|---|---|---|---|
| 1 | Orphan ratchet fails naming an unwired command; allowlist seeded with 293 real orphans; a companion assertion means the allowlist "may only shrink"; caller evidence cannot leak between same-leaf commands | **TRUE ONLY IF NARROWED** | See narrowed wording below. Mechanism proven correct in isolation; residue is CR-06 (shared with criterion 4) |
| 2 | `aether spawn-can-spawn 5 --enforce` exits 0 | **TRUE AS WRITTEN** | Ran live: `{"ok":true,"result":{"can_spawn":true,"depth":5}}`, exit 0 |
| 3 | A test enumerates every `aether …` invocation in `.aether/*.md` and fails naming file/line/flag for an unregistered flag; passes only once criterion 2 does | **TRUE AS WRITTEN** | `TestAetherCorpusCatchesAnUnregisteredFlag`, `TestCLIFlagAudit`, `TestCLIFlagAuditSubcommandsRegistered` all pass; no absolute-over-adversarial-surface framing, no CR-06 exposure |
| 4 | The ratchet and flag test run in the same CI command the release gate already runs, verified by deleting a caller and observing the gate go red | **TRUE ONLY IF NARROWED** | See narrowed wording below. The literal claim is proven; the implied absolute ("cannot be silently switched off") is defeated by CR-06 |

**Score:** 4/4 — substance delivered on all four; two require narrower wording to avoid overclaiming, per the stop rule's own design (an absolute over an unbounded adversarial surface has no defined edge, and a narrower true claim beats a broader unproven one — CLAUDE.md's Definition of Done).

---

### Criterion 1 — Proposed narrowed wording

> Registering a new cobra subcommand with no caller outside its own definition
> file makes `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced`
> fail, naming the command — proven by a fixture that registers exactly such
> a command. The allowlist ships seeded with the scan's real output: 293
> pre-existing orphans (274 carried forward one-to-one from the pre-migration
> baseline including all 6 entries tagged `owner_phase: "178"`, 10 from
> splitting 4 collapsed leaf entries into the real commands each was
> invisibly covering, and 9 newly revealed and tagged
> `reason: "path-collision-revealed"`). A companion assertion, comparing
> **full command paths** (not bare leaf names) against a SHA-256-pinned
> pre-migration snapshot (`preMigrationSnapshotSHA256`,
> `TestPreMigrationSnapshotIsFrozen`), fails when the allowlist gains an
> entry it did not have in the committed baseline — proven both by
> `TestPathMigrationRejectsASameLeafNewcomer` (a same-leaf-name newcomer,
> including the reviewer's own `aether colony-depth setup` counterexample)
> and by a generality loop rejecting newcomer paths built from all 278
> frozen tolerated leaves at once. `TestCallerEvidenceIsNotSharedBetweenSameLeafNames`
> proves caller evidence for one command path can never leak to a
> same-leaf-name command at a different path.
> **Residue, named explicitly and not closed by this phase:** this
> guarantees the allowlist can only shrink whenever the guard test actually
> executes inside CI. It does not by itself guarantee the guard always
> executes — CR-06 (below, shared with criterion 4) shows any of the ~18
> steps that run before the gate step in the same CI job can alter the
> environment or replace the `go` binary the whole test run depends on,
> which would silence this assertion along with every other guard test in
> the same job, undetected by any mechanism this phase built.

**Why the residue is shared with criterion 1, not just criterion 4:** the
"may only shrink" property is enforced by a Go test
(`TestPathMigrationDidNotWidenTolerance`) that only has teeth when it
actually runs as part of `go test ./cmd -run '...'` inside the CI job. CR-06
is a route to making that whole `go test` invocation report success without
genuinely executing — so the absolute form of criterion 1 rests on the same
unaudited surface as criterion 4's absolute form, even though CR-06 itself
was found while reviewing criterion 4's mechanisms.

### Criterion 4 — Proposed narrowed wording

> The ratchet and the flag test run in the same CI command the release gate
> already runs, under a named step
> (`Verify subcommand wiring and CLI flag contracts`, `.github/workflows/ci.yml:99-100`)
> — proven by deleting a caller and observing
> `TestDeletingACallerMakesTheRatchetNameIt` name it, not by reading the
> workflow file. A whitelist of the workflow's root keys
> (`workflowRootAllowedKeys`), the `go` job's keys (`gateJobAllowedKeys`),
> each trigger's keys (`releaseTriggerAllowedKeys`), and the two gate steps'
> own keys (`gateStepAllowedKeys`) rejects any key outside a reviewed set by
> default, closing three independently-reproduced job/trigger-level
> disabling routes: `continue-on-error` at job scope, `env: GOFLAGS` at
> workflow/job/step scope, and `paths-ignore` under `on:` triggers.
> **Residue, tracked and explicitly not closed by this phase (CR-06):** the
> other ~18 steps that run earlier in the same `go:` job — Checkout, Setup
> Go, Setup Node, Install goreleaser, Validate goreleaser config, Build,
> Vet, and the rest — are not inspected by any mechanism this phase built.
> Any of them can write `GOFLAGS`/`GOTOOLCHAIN` to `$GITHUB_ENV`, or write a
> fake `go` earlier on `$PATH` via `$GITHUB_PATH`, or overwrite the `go`
> binary directly — all standard, documented GitHub Actions mechanisms
> available to any step sharing the job's runner and filesystem — and
> reproduce CR-02's outcome (the gate running zero tests and exiting 0) by a
> route the exact-equality pin, the execution harness, and the shape
> whitelist were each built to inspect the wrong text for. The gate step's
> command text is provably tamper-evident; the gate step's **execution
> environment**, inherited from every step that ran before it in the same
> job, is not.

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `cmd/ci_wiring_gate_test.go` | Execution-based release-gate harness + workflow-shape whitelist | ✓ VERIFIED | 1946 lines; `auditWorkflowShape`, `gateJobAllowedKeys`, `gateStepAllowedKeys`, `releaseTriggerAllowedKeys`, `workflowRootAllowedKeys` present and wired into `TestReleaseGateWorkflowShapeIsWhitelisted` / `TestWorkflowShapeWhitelistRejectsUnenumeratedKeys` |
| `cmd/subcommand_reachability_ratchet_test.go` | Orphan ratchet, path-keyed shrink-only guard, pinned anchor | ✓ VERIFIED | 1956 lines; `preMigrationSnapshotSHA256` constant matches independently re-computed SHA-256 of the anchor file; `TestPathMigrationDidNotWidenTolerance` compares full paths |
| `cmd/cli_flag_audit_test.go` | Flag-audit corpus scan | ✓ VERIFIED | 374 lines; `TestAetherCorpusCatchesAnUnregisteredFlag`, `TestCLIFlagAudit` pass |
| `.github/workflows/ci.yml` | Ratchet + flag test wired into the release-gate CI job | ✓ VERIFIED | Line 99-100: `Verify subcommand wiring and CLI flag contracts` step runs the exact 34-test `-run` filter, in the same `go:` job as `Run Go tests` (line 41-42) |
| `.aether/docs/orphan-allowlist-policy.md` | Written policy, arithmetic matches code | ✓ VERIFIED | 274+10=284, 284+9=293 accounting matches both JSON files and the independently re-derived arithmetic |
| `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` | Frozen anchor, pinned | ✓ VERIFIED | `shasum -a 256` → `873cad5a20e472af7050e69042be9faf8527a2c3e1d34907762b37e6a5845e0e`, byte-identical to `preMigrationSnapshotSHA256` |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `.github/workflows/ci.yml:99-100` | `cmd/*_test.go` guard suite | `go test ./cmd -run '<34-name filter>'` | WIRED | Filter string in ci.yml matches the 34 test names verified to exist and pass; `TestDeletingACallerMakesTheRatchetNameIt` proves deletion of a caller is observable through this exact chain |
| `.aether/workers.md:292` | `cmd/spawn.go` `spawn-can-spawn --enforce` | direct CLI invocation | WIRED | Live run confirms exit 0, matches documented invocation exactly |
| `.aether/*.md` corpus | `cmd/cli_flag_audit_test.go` | static enumeration + cobra flag-set lookup | WIRED | `TestAetherCorpusCatchesAnUnregisteredFlag` passes against the live corpus |

### Data-Flow Trace (Level 4)

Not applicable in the usual sense — this phase produces CI guard tests, not
a UI/data-rendering artifact. The equivalent check is "does the guard
actually run inside the release gate CI job", which is verified under Key
Link Verification above (the `-run` filter is textually present in the same
job as the release gate, and `TestDeletingACallerMakesTheRatchetNameIt`
proves the ratchet's own logic fires). The residue named in criteria 1 and 4
above (CR-06) is precisely the gap in this trace: the guard's *presence* in
the job is proven, but the job's *environment integrity* up to that point
is not.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| `spawn-can-spawn --enforce` exits 0 (criterion 2) | `go run ./cmd/aether spawn-can-spawn 5 --enforce` | `{"ok":true,"result":{"can_spawn":true,"depth":5}}`, exit 0 | ✓ PASS |
| All 34 named guard tests pass (criteria 1, 3, 4) | `go test ./cmd -run '<34-name filter, verbatim from ci.yml:100>' -count=1 -v` | `ok github.com/calcosmic/Aether/cmd 7.075s`, zero FAILs | ✓ PASS |
| Frozen anchor's pin matches its actual contents (CR-05 closure) | `shasum -a 256 cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` | `873cad5a...45e0e` — matches `preMigrationSnapshotSHA256` in source | ✓ PASS |
| CR-06's structural premise (unaudited steps ahead of the gate) | Read `.github/workflows/ci.yml` step list + `cmd/ci_wiring_gate_test.go:1427-1454` | 18 steps precede the wiring-gate step; `gateStepAllowedKeys` loop iterates only the 2 named gate steps | ✓ CONFIRMED (residue, not a defect closed this round) |

No source files were mutated to produce this report; `git diff --stat`
before and after this verification session shows no changes to any
tracked file this phase touches (the only diffs present are pre-existing,
unrelated deletions under `.planning/phases/160-*` through `165-*` that
predate this verification session).

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|---|---|---|---|---|
| WIRE-01 | 172-00, 172-02, 172-04, 172-05, 172-07, 172-08, 172-09, 172-10, 172-11, 172-12, 172-13 | Orphan ratchet fails on an unwired command, shrink-only allowlist | ✓ SATISFIED (narrowed per criterion 1 above) | `TestNoRegisteredSubcommandIsUnreferenced` + full-path shrink-only chain, all green |
| WIRE-02 | 172-01, 172-05, 172-07, 172-10, 172-11, 172-12, 172-13 | `aether spawn-can-spawn 5 --enforce` succeeds | ✓ SATISFIED | Live run confirmed, exit 0 |
| WIRE-03 | 172-00, 172-03, 172-04, 172-05, 172-06, 172-07, 172-08, 172-10, 172-11, 172-12, 172-13 | Flag-audit test over `.aether/*.md` corpus | ✓ SATISFIED | `TestAetherCorpusCatchesAnUnregisteredFlag` et al. green |

No orphaned requirements — all three IDs declared in `REQUIREMENTS.md`'s
Wiring Proof section are claimed by at least one 172 plan's frontmatter, and
vice versa. (Note, informational only: `REQUIREMENTS.md`'s later
traceability table at lines 137-141 still shows WIRE-01/02/03 as `Pending`
even though the WIRE section's own checklist above it has them `[x]`
checked — a stale-status doc mismatch, not a coverage gap. Not fixed here;
out of scope for a verification pass, flagged for whoever next edits
`REQUIREMENTS.md`.)

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` markers in any file this round's plans (172-12,
172-13) touched (`cmd/ci_wiring_gate_test.go`, `cmd/subcommand_reachability_ratchet_test.go`,
`.aether/docs/orphan-allowlist-policy.md`, the two testdata JSON files) —
confirmed by direct grep.

The round-4 code review (`172-REVIEW.md`) lists 9 Warning-level and 8
Info-level findings carried forward unchanged from prior rounds (WR-01,
WR-02, WR-03, WR-05, WR-07, WR-08, WR-09, WR-10, IN-01..IN-08). None of
these were must-haves of any 172 plan — they are pre-existing, previously
documented, out-of-scope-for-this-round findings (e.g. WR-02:
`-update-orphan-allowlist` flag skips core assertions before returning;
WR-08: anti-vacuity floor allows up to 41% of guard tests to be deleted
without tripping). They do not block this verification's `passed`
determination because no plan's own must_haves claim to close them, but
they remain real and are appropriately documented for whoever plans the
CR-06 follow-up phase to fold in alongside it if in scope.

| File | Pattern | Severity | Impact |
|---|---|---|---|
| `cmd/ci_wiring_gate_test.go:612-636` | WR-01: wiring step's run-line check tolerates trailing `\|\| true`/`; true`/`\| cat`/`&&` | Warning | Pre-existing, not this round's scope |
| `cmd/subcommand_reachability_ratchet_test.go:1164` | WR-02: `-update-orphan-allowlist` returns before core assertions | Warning | Pre-existing, not this round's scope |
| `cmd/ci_wiring_gate_test.go:801-828` | WR-03: `sh -c` executes workflow-sourced string with only a substring sanity floor | Warning | Pre-existing, not this round's scope |
| `cmd/cli_flag_audit_test.go:356` | WR-05: `../`-relative path used despite a same-file ban 270 lines earlier | Warning | Pre-existing, not this round's scope |
| `cmd/ci_wiring_gate_test.go:171` | WR-08: anti-vacuity floor tolerates 14/34 (41%) guard tests deleted | Warning | Pre-existing, margin widened in absolute terms as suite grew, not tightened |

### Human Verification Required

None. Every truth in this report was verified by reading source, running
the actual guard-test suite, and live-executing the documented CLI
invocation — no visual, real-time, or subjective judgment call remains
open.

### Gaps Summary

No gaps in the "unmet must-have" sense: every plan's own must_haves are
verified true in the code, and all 34 named guard tests pass. The one
substantive finding — CR-06 — is not a gap against any plan's must-haves;
it is a newly-discovered residue against the *original, absolute* wording
of ROADMAP success criteria 1 and 4, which is precisely the situation
`172-STOP-RULE.md` was written in advance to govern. This report proposes
the narrowed wording the stop rule requires and recommends the user/
orchestrator schedule a tracked follow-up phase (not currently in
`ROADMAP.md`) to carry CR-06 forward — most directly by moving the release
gate to a job/workflow with no untrusted step ahead of it, per
`172-REVIEW.md`'s Fix section, option 1.

---

_Verified: 2026-08-12T16:10:00Z_
_Verifier: Claude (gsd-verifier)_
