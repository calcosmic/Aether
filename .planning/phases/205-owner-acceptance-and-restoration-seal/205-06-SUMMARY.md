---
phase: 205-owner-acceptance-and-restoration-seal
plan: 06
subsystem: testing
tags: [go, synthesis-audit, classic-contract, requirements-traceability]

# Dependency graph
requires:
  - phase: 199-204 (all)
    provides: seven CLASSIC-SYNTHESIS.md artifacts and the shared v1.28-classic-synthesis-template.md this plan audits
provides:
  - A named, runnable command (go test ./cmd/ -run TestClassicSynthesis) that fails by name when a Phase 199-204 synthesis artifact is missing a mandatory section, has an empty section, uses an inexact heading, or double-claims a capability row
  - A plain-English written record (205-SYNTH-07-AUDIT.md) judging all seven artifacts against SYNTH-07's four clauses, with every shortfall named and routed
affects: [205-owner-acceptance-and-restoration-seal seal/UAT plans, any future audit of the Classic-synthesis corpus]

# Actuals (#2632)
actuals:
  tokens: 12184
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Mandatory-section audit derives its check list by parsing the template file itself (regex on '## N. Title' headings), never a hand-typed literal list, so a future template edit changes the audit automatically."
    - "Capability-row extraction scans an artifact's whole text (not only its frontmatter cap_rows: list) because frontmatter can be incomplete (Phase 202 omits CAP-072 despite discussing it) while the body is always the complete, honest record."
    - "A structural exception (Phase 199's non-matching headings) is recorded as a named, counted, pinned allowlist (classicSynthesisKnownSectionGaps) rather than weakening the underlying exact-match rule — the same 'counted exception, may only shrink' idiom this codebase already uses for fixture-bank unguarded counts."

key-files:
  created:
    - cmd/classic_synthesis_audit_test.go
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-SYNTH-07-AUDIT.md
  modified: []

key-decisions:
  - "Capability-row claims are extracted by scanning an artifact's whole document text for CAP-### tokens, not by trusting only its frontmatter cap_rows: list. This is the audit's answer to the plan's flagged Phase 200 pitfall (capability routing living only in a §6 table column, never a dedicated heading): frontmatter is not required to be complete or a dedicated heading required to exist, because the body-wide scan correctly recovers every artifact's real claims — proven against all six ledger-routed phases, including recovering Phase 202's CAP-072 despite it being missing from that artifact's own frontmatter list."
  - "Phase 199's synthesis artifact (signed 2026-09-03, before the eight-section template existed) is recorded as a named, pinned exception in classicSynthesisKnownSectionGaps rather than silently weakening the exact-match rule or editing the already-signed document. Its substance was read and independently judged compliant with all four SYNTH-07 clauses in 205-SYNTH-07-AUDIT.md despite the heading mismatch."
  - "Neither of the two real shortfalls this audit found (Phase 199's heading structure, Phase 202's one-row frontmatter gap) was fixed in this phase — both are pre-existing gaps in other, already-delivered phases' documents, outside this task's scope per the deviation rules' scope boundary. Both are named and routed to the limitations card in 205-SYNTH-07-AUDIT.md, not silently dropped."

requirements-completed: [SYNTH-07]

coverage:
  - id: D1
    description: "A command lists the seven synthesis artifacts for Phases 199-204 and fails, naming the artifact and the section, when one is missing a mandatory section, has an empty section, or uses an inexact heading."
    requirement: "SYNTH-07"
    verification:
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisPhase203HasEveryMandatorySection"
        status: pass
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisAuditRejectsAMissingSection"
        status: pass
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisAuditRejectsAnEmptySection"
        status: pass
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisSectionMatchIsExact"
        status: pass
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisEveryArtifactHasEveryMandatorySection"
        status: pass
    human_judgment: false
  - id: D2
    description: "No capability row is claimed by two artifacts; a clash names the row and both artifacts, and the rule is proven capable of firing on synthetic data, not only proven to pass on the real (clash-free) corpus."
    requirement: "SYNTH-07"
    verification:
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisCapabilityRowsAreNotDoubleClaimed"
        status: pass
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisAuditRejectsADoubleClaimedCapabilityRow"
        status: pass
    human_judgment: false
  - id: D3
    description: "The audit's output lists artifacts in ascending phase order and findings in template section order, is byte-identical across repeated runs, and reports whole-number counts per artifact, never a percentage or an aggregate score."
    requirement: "SYNTH-07"
    verification:
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisAuditOutputIsOrderedAndStable"
        status: pass
      - kind: unit
        ref: "cmd/classic_synthesis_audit_test.go#TestClassicSynthesisAuditReportsCountsNotPercentages"
        status: pass
    human_judgment: false
  - id: D4
    description: "A plain-English written record judges each of the seven artifacts against SYNTH-07's four clauses (evidence, alternatives, linkage, no label-only restoration) by reading the artifact, not by counting headings, and names every shortfall found plus where it goes."
    requirement: "SYNTH-07"
    verification: []
    human_judgment: true
    rationale: "Whether the written judgement of each artifact's evidence/alternatives/linkage/no-label-only-restoration clauses is itself accurate and fair is a semantic reading of prose against SYNTH-07's four clauses — no automated test can grade that judgement; a human (or a future independent audit) reading 205-SYNTH-07-AUDIT.md against the seven artifacts is the verification path."

# Metrics
duration: 33min
completed: 2026-09-15
status: complete
---

# Phase 205 Plan 06: Audit the Seven Restoration Studies Against SYNTH-07 Summary

**A runnable command (`go test ./cmd/ -run TestClassicSynthesis`, 9 tests) that fails by name on a missing, empty, or mislabelled synthesis section or a double-claimed capability row, plus a 300-line plain-English record judging all seven Phase 199-204 studies against SYNTH-07's four clauses — finding six fully compliant and one (Phase 199, signed before the template existed) substantively compliant but structurally mismatched.**

## Performance

- **Duration:** 33 min
- **Started:** 2026-09-15T10:27:34+02:00 (worktree fork point)
- **Completed:** 2026-09-15T10:59:57+02:00
- **Tasks:** 3 (plus one post-hoc test-completeness fix, see Deviations)
- **Files modified:** 2 created, 0 modified

## Accomplishments

- `cmd/classic_synthesis_audit_test.go` derives the eight mandatory sections from `.planning/research/v1.28-classic-synthesis-template.md` by parsing its own numbered headings (never a hand-typed list), and audits every one of the seven Phase 199-204 synthesis artifacts (discovered from the phase directories on disk, never hardcoded) against them.
- The audit reports present / empty / absent per section on exact, normalized heading match — case-insensitive, whitespace-collapsed, punctuation-trimmed, and (critically) leading-ordinal-stripped, so a heading merely containing a section's words never satisfies it.
- Capability-row claims are extracted from each artifact's whole text and cross-checked against the frozen capability ledger's own per-phase routing (reusing `classicContractPhase*Capabilities` from `classic_contract_test.go`, never a second, drifting copy of the ledger).
- `.planning/phases/205-owner-acceptance-and-restoration-seal/205-SYNTH-07-AUDIT.md` reads all seven artifacts directly and judges each against SYNTH-07's four clauses (evidence, alternatives, linkage, no label-only restoration) in plain English, with every invented word translated inline on first use.
- Both real shortfalls this audit found — Phase 199's non-matching heading structure and Phase 202's one-row frontmatter gap (`CAP-072` missing from `cap_rows:` despite being discussed and routed in the body) — are named, judged not to affect substance, and explicitly routed to the milestone's limitations card rather than silently fixed or hidden.

## Task Commits

Each task was committed atomically:

1. **Task 1: One artifact, audited end to end by a command that fails when a section is missing** - `6be45490` (feat)
2. **Task 2: All seven artifacts, in stable order, with no double-claimed capability row** - included in Task 1's commit; see Deviations
3. **Task 3: The written audit record, in plain English** - `935b1b55` (docs)
4. **Post-hoc: prove the double-claim rule can actually fail** - `659d9bb2` (test)

## Files Created/Modified

- `cmd/classic_synthesis_audit_test.go` - The SYNTH-07 audit: artifact discovery, template-derived mandatory-section list, present/empty/absent section auditor, capability-row extraction and cross-check, stable ordered rendering, and 9 tests (4 from Task 1, 4 from Task 2, 1 post-hoc completeness fix).
- `.planning/phases/205-owner-acceptance-and-restoration-seal/205-SYNTH-07-AUDIT.md` - The plain-English written record judging all seven artifacts against SYNTH-07's four clauses.

## Decisions Made

- **Capability-row extraction scans the whole artifact, not only its frontmatter.** Discovered mid-task that Phase 200's capability routing lives only inside a §6 table column (the plan's own flagged pitfall) and that Phase 202's frontmatter `cap_rows:` list is one row short of what its body actually discusses (`CAP-072`). Rather than requiring a dedicated capability-routing heading (which would have meant editing other phases' already-delivered documents, out of this task's scope), the audit scans the whole document's text for `CAP-###` tokens. This one rule correctly recovers every artifact's true capability claims — verified exactly against the frozen ledger for all six ledger-routed phases — with no per-artifact special-casing needed.
- **Phase 199's heading mismatch is recorded, not hidden or auto-fixed.** Phase 199's synthesis document was signed 2026-09-03, before this milestone's eight-section numbered template existed; only 2 of its 8 sections match the template's exact heading wording. Reading the document directly (not just its headings) shows its substance genuinely satisfies all four SYNTH-07 clauses — real evidence citations, a dedicated alternatives-considered column, a 17-row decision-traceability table, and an explicit rejection of "byte-for-byte Classic restoration." The gap is structural, not substantive. Per the deviation rules' scope boundary (pre-existing issues in files this task's own changes didn't cause are out of scope to fix), this was not edited; it is recorded as a named, pinned exception (`classicSynthesisKnownSectionGaps`) in the test and detailed in the written record, routed to the milestone's limitations card.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Artifact heading normalization stripped no leading ordinal, so every numbered section matched "absent"**
- **Found during:** Task 1, first `go test` run of `TestClassicSynthesisPhase203HasEveryMandatorySection`
- **Issue:** The template's own numbered headings ("## 2. Classic mechanism reconstruction") are parsed into a bare mandatory-section name ("Classic mechanism reconstruction") by `classicSynthesisMandatorySections`, but the artifact-side heading scanner normalized the full heading text including its own "2. " prefix, so every artifact heading normalized to e.g. "2 classic mechanism reconstruction" and never matched any mandatory section name — all 8 sections reported "absent" for Phase 203, an artifact independently confirmed to be fully compliant.
- **Fix:** Added `classicSynthesisLeadingNumberPattern` (`^[0-9]+\.\s*`) and applied it inside `classicSynthesisNormalizeHeading` itself, so both the template's own headings and every artifact's headings normalize through the identical rule.
- **Files modified:** cmd/classic_synthesis_audit_test.go
- **Verification:** All four Task 1 tests pass after the fix; re-verified against all seven real artifacts in Task 2.
- **Committed in:** 6be45490 (Task 1 commit — found and fixed before the first commit, so no separate fix commit exists)

**2. [Process note, not a code deviation] Tasks 1 and 2 landed in one commit, not two**
- **Found during:** attempting to commit Task 2 separately
- **Issue:** The full test file (Task 1's four tests plus Task 2's four tests) was authored as one `Write` call before the first commit, so Task 1's commit (`6be45490`) already contains Task 2's test bodies. This is a process deviation from the plan's one-commit-per-task expectation, not a correctness issue — every acceptance criterion for both tasks was independently verified via `go test` after the fact.
- **Fix:** None needed; documented here for traceability. Task 2's own acceptance-criteria commands were re-run and confirmed passing against the already-committed code.
- **Files modified:** none (documentation only)
- **Committed in:** n/a (no code change)

**3. [Rule 1 - Bug, test completeness] The double-claimed-capability-row rule had never been proven capable of failing**
- **Found during:** final plan-level `<verification>` review, after Task 3
- **Issue:** The plan's own top-level verification requires "a demonstrated failure for ... a double-claimed capability row," matching the pattern already used for missing/empty/inexact-heading sections (each proven via a mutated temp copy). `TestClassicSynthesisCapabilityRowsAreNotDoubleClaimed` only ran against the real corpus, which has no clash today — so its double-claim branch had never actually been exercised, violating this project's own "a test must be able to fail" rule (CLAUDE.md Definition of Done).
- **Fix:** Extracted the detection rule into a pure function `classicSynthesisDoubleClaimedCapabilityRows(perArtifact map[string][]string) map[string][]string` and added `TestClassicSynthesisAuditRejectsADoubleClaimedCapabilityRow`, a synthetic two-phase fixture with a genuine clash, proving the rule fires and names both the row and both claiming phases.
- **Files modified:** cmd/classic_synthesis_audit_test.go
- **Verification:** `go test ./cmd/ -run 'TestClassicSynthesisAuditRejectsADoubleClaimedCapabilityRow' -count=1 -timeout 90m` exits 0; full `TestClassicSynthesis` group (9 tests) still passes.
- **Committed in:** 659d9bb2

---

**Total deviations:** 3 (1 auto-fixed bug, 1 process note, 1 test-completeness fix). **Impact:** the bug fix was necessary for the audit to work at all; the test-completeness fix closes a real gap in the plan's own verification bar. No scope creep — no CLASSIC-SYNTHESIS.md file for any phase was edited.

## Issues Encountered

None beyond the deviations above.

## Known Stubs

None. No stub patterns, placeholder text, or hardcoded-empty data introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SYNTH-07 is satisfied: a runnable, real command (`go test ./cmd/ -run TestClassicSynthesis`) fails by name on a missing section, an empty section, an inexact heading, or a double-claimed capability row, and a plain-English written record judges all seven artifacts against SYNTH-07's four clauses.
- Two real, narrow shortfalls are recorded and routed to the milestone's limitations card (Phase 199's heading structure predates the template; Phase 202's frontmatter is missing one capability row it otherwise correctly discusses) — carried forward, not silently dropped, per Task 3's own design.
- No blockers for the rest of Phase 205's owner-acceptance and seal work.

---
*Phase: 205-owner-acceptance-and-restoration-seal*
*Completed: 2026-09-15*
