---
phase: 199-front-door-and-classic-contract
plan: "33"
subsystem: documentation
tags: [lifecycle, seal, entomb, generated-docs, d17]
requires:
  - phase: 199-15
    provides: retained sealed-state lifecycle truth
  - phase: 199-16
    provides: forced-incomplete seal outcome truth
  - phase: 199-22
    provides: canonical command-source hygiene checks
  - phase: 199-23
    provides: project-document generation path
provides:
  - status-first retained-seal guidance across current public, rule, template, and generated guides
  - production-generation parity tests for Claude, OpenCode, and Codex documentation
affects: [phase-199-closeout, lifecycle-guidance, generated-project-docs]
tech-stack:
  added: []
  patterns: [canonical-source-first documentation, isolated production-generation parity]
key-files:
  created: [.planning/phases/199-front-door-and-classic-contract/199-33-SUMMARY.md]
  modified: [README.md, AGENTS.md, .aether/rules/aether-colony.md, .claude/rules/aether-colony.md, .aether/templates/opencode-md-template.md, cmd/.opencode/OPENCODE.md, .aether/templates/agents-md-template.md, cmd/AGENTS.md, cmd/current_vocabulary_docs_199_test.go]
key-decisions:
  - "A sealed colony remains active and reviewable; status is always the first post-seal action."
  - "Entomb is an optional explicit owner action, and only its successful archive-and-clear receipt enables a new init."
patterns-established:
  - "Generated project documentation is compared with isolated production generator output rather than hand-maintained independently."
requirements-completed: [CEC-02, CEC-08, LIFE-01, LIFE-03, LIFE-05, PROOF-01]
duration: 31min
completed: 2026-09-05
---

# Phase 199 Plan 33: Retained-Seal Documentation Summary

**Status-first lifecycle guidance now keeps sealed colonies reviewable, makes entomb an explicit optional archive-and-clear action, and unlocks a new goal only after a verified receipt.**

## Performance

- **Duration:** 31 min
- **Tasks:** 2/2
- **Files modified:** 9

## Accomplishments

- Documented the identical D-17 sequence across public, Codex, Claude, OpenCode, template, and generated guide surfaces.
- Preserved forced-incomplete visibility through status review and optional entomb, without describing it as verified completion.
- Added exact ordering, negative-case, canonical-rule, and isolated production-generation parity coverage.

## Task Commits

1. **Task 1: Make status review primary in public and canonical colony guides**
   - `9b7f84cb` `test(199-33): add failing D-17 documentation contract`
   - `23caf204` `docs(199-33): make retained-seal review primary`
2. **Task 2: Regenerate OpenCode and Codex project guidance with the same sequence**
   - `ba4d7598` `test(199-33): require generated D-17 project guides`
   - `b968d789` `docs(199-33): retain sealed colonies through review`

## Files Created/Modified

- `README.md` and `AGENTS.md` — public and Codex lifecycle closure guidance.
- `.aether/rules/aether-colony.md` and `.claude/rules/aether-colony.md` — canonical and production-generated Claude rule parity.
- `.aether/templates/{opencode-md-template,agents-md-template}.md` — canonical OpenCode and Codex project-document sources.
- `cmd/.opencode/OPENCODE.md` and `cmd/AGENTS.md` — current generated project guides.
- `cmd/current_vocabulary_docs_199_test.go` — D-17 order, negative cases, and isolated generator parity proof.

## Decisions Made

- Status/review is the primary post-seal route for verified and forced-incomplete closures.
- Entomb is separate, optional, and owner-invoked; its verified archive-and-clear receipt is the only new-init precondition.
- Claude/OpenCode use `/ant-*`; Codex guidance retains raw `aether ...` vocabulary. Codex-native ant skills remain outside the Phase 200–205 boundary.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Verification

- `go test ./cmd -run '^(TestCurrentVocabularyDocs199|TestCommandSourceHygiene|TestSyncProjectDocsGeneratesOpenCodeMD|TestSyncProjectDocsPreservesCustomOpenCodeMD|TestSyncProjectDocsRefreshesManagedOpenCodeMD|TestColonyRulesCopiesStayIdentical)$' -count=1`
- `go build ./cmd/aether`
- `go test -race ./cmd -run '^TestCurrentVocabularyDocs199$' -count=1`

## Next Phase Readiness

All current lifecycle guides share the retained-seal contract and generated-document parity guard. No manual setup or external service configuration is required.

## Self-Check: PASSED

- Summary exists at the planned path.
- All four RED/GREEN task commits exist in repository history.
