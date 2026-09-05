---
phase: 199-front-door-and-classic-contract
plan: "31"
subsystem: documentation-and-benchmarks
tags: [lifecycle, pause, resume, opencode, benchmarks, tdd]
requires:
  - phase: 199-30
    provides: canonical pause/resume wrapper and OpenCode template vocabulary
provides:
  - Active public and platform documentation with one pause/resume lifecycle contract
  - Generated OpenCode project guidance parity proof
  - Interruption benchmarks that exercise the sole resume recovery door
affects: [phase-199-verification, public-documentation, benchmark-harness]
tech-stack:
  added: []
  patterns: [command-shaped documentation token checks, production-generator parity proof]
key-files:
  created: [cmd/current_vocabulary_docs_199_test.go]
  modified: [README.md, AGENTS.md, cmd/.opencode/OPENCODE.md, docs/phase3-section-commands.md, bench/RUNBOOK.md, bench/harness/permitted-inputs.md, bench/tasks/03-interrupted-execution.md]
key-decisions:
  - "Active Claude/OpenCode guidance exposes exactly /ant-pause and /ant-resume; Codex guidance uses raw aether pause and aether resume."
  - "A conflicting interruption can use logged recovery-inspect diagnostics only before a fresh resume, never as a normal scripted recovery step."
patterns-established:
  - "Documentation parity tests distinguish command-shaped tokens from Oracle recover and ordinary recovery prose."
requirements-completed: [SYNTH-01, CEC-02, CEC-04, LIFE-01, LIFE-04, PROOF-01]
duration: 35min
completed: 2026-09-05
---

# Phase 199 Plan 31: Pause/Resume Documentation and Benchmark Summary

Active guides and interruption benchmarks now teach a single evidence-backed pause/resume recovery contract.

## Performance

- **Duration:** 35min
- **Completed:** 2026-09-05T00:04:35Z
- **Tasks:** 2/2
- **Files modified:** 8

## Accomplishments

- Replaced retired public lifecycle commands with `/ant-pause` and `/ant-resume`, while keeping raw Codex commands as `aether pause` and `aether resume`.
- Proved `cmd/.opencode/OPENCODE.md` matches production generation from `.aether/templates/opencode-md-template.md`.
- Made both Aether interruption lanes run exactly one `/ant-resume` after SIGKILL, with explicit logged maintenance diagnostics only for conflicts.

## Task Commits

1. **Task 1: Migrate active public and platform guides** — `f6b16fab` (RED test), `0e47439b` (GREEN implementation)
2. **Task 2: Make interruption benchmarks exercise the sole resume door** — `b6113e5f` (RED test), `55cb7442` (GREEN implementation)

## Verification

- `go test ./cmd -run '^TestCurrentVocabularyDocs199$' -count=1` — PASS
- Production `install` → `lay-eggs` generation in a temporary home/repository, byte-compared with `cmd/.opencode/OPENCODE.md` — PASS
- 17 focused command-source, platform-document, and parity tests — PASS
- `go test -race ./cmd -run '^TestCurrentVocabularyDocs199$' -count=1` — PASS
- `go build ./cmd/aether` — PASS
- Owned-source retired-token and stub scans — PASS

## Decisions Made

- Resume describes Confirmed, Reconstructed, Conflicting, and Unknown evidence outcomes in the public and standalone lifecycle references.
- Oracle's `recover` subcommand and ordinary recovery prose remain valid because checks reject only command-shaped retired lifecycle tokens.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test setup] Corrected the OpenCode generation proof to stage the canonical template in the production generator's hub layout.**
- **Found during:** Task 1
- **Fix:** The parity test now invokes `syncProjectDocs` with a real `templates/` hub input rather than incorrectly treating the source checkout root as a hub.

**2. [Rule 1 - Test parsing] Normalized Markdown table backticks and both lane label forms before comparing the scripted command.**
- **Found during:** Task 2
- **Fix:** The benchmark proof compares command values rather than Markdown presentation, while retaining strict command-shaped retired-token rejection.

**Total deviations:** 2 auto-fixed (Rule 1 test correctness).

## Known Stubs

None.

## Issues Encountered

The full-suite baseline is intentionally owned by the Wave 15 root executor; this plan ran its required focused verification and build gates only.

## Self-Check: PASSED

All owned artifacts exist and all four TDD task commits are present in git history.
