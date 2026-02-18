---
phase: 09-polish-verify
plan: 04
subsystem: testing
tags: [bash, e2e-tests, xml-verification, documentation-verification, lifecycle-integration, requirements-matrix]

requires:
  - phase: 09-polish-verify
    provides: "09-01/02/03: e2e test infrastructure + 35/46 requirements verified PASS"
  - phase: 08-xml-integration
    provides: "pheromone-export-xml, wisdom-export-xml, registry-export-xml + import counterparts"

provides:
  - "tests/e2e/test-xml.sh: XML-01/02/03 verified PASS (pheromone/wisdom/registry XML round-trips)"
  - "tests/e2e/test-doc.sh: DOC-01/02/03/04 verified PASS (learnings, eternal memory, progress, handoffs)"
  - "tests/e2e/test-lifecycle.sh: full 7-step connected workflow integration test PASS"
  - "tests/e2e/run-all-e2e.sh: master runner producing 46-row requirements matrix (100% pass rate)"
  - "tests/e2e/RESULTS.md: auto-generated requirements matrix — 46/46 PASS"
  - ".planning/phases/09-polish-verify/09-VERIFICATION.md: executive summary + full matrix"
  - ".planning/REQUIREMENTS.md: all 46 checkboxes updated [x]"

affects:
  - "Phase 9 complete — entire Aether repair project verified"

tech-stack:
  added: []
  patterns:
    - "Master runner with file-based result aggregation (bash 3.2 compatible, subprocess per area script)"
    - "Parallel arrays for requirement metadata (IDs, descriptions, scripts) without associative arrays"
    - "--results-file flag protocol: area scripts write KEY=STATUS lines when flag provided"

key-files:
  created:
    - "tests/e2e/test-xml.sh"
    - "tests/e2e/test-doc.sh"
    - "tests/e2e/test-lifecycle.sh"
    - "tests/e2e/run-all-e2e.sh"
    - "tests/e2e/RESULTS.md"
    - ".planning/phases/09-polish-verify/09-VERIFICATION.md"
  modified:
    - ".planning/REQUIREMENTS.md"
    - ".aether/exchange/pheromone-xml.sh"
    - ".aether/exchange/wisdom-xml.sh"
    - ".aether/exchange/registry-xml.sh"

key-decisions:
  - "09-04: pheromone-xml.sh content field: use type-aware jq expression to handle plain string .content vs object .content.text"
  - "09-04: xmlstarlet pipefail: use set +e in subshell pattern for xmlstarlet sel calls that may return exit 1 on no-match"
  - "09-04: lifecycle test uses separate LIFECYCLE_TMP (not setup_e2e_env) to create genuine isolated git project"

patterns-established:
  - "lifecycle integration test pattern: create git repo in mktemp -d, copy aether-utils.sh, run subcommands as connected chain"
  - "master runner pattern: RESULTS_DIR for per-area temp files, cat all to MASTER_RESULTS, parallel arrays for metadata"
  - "requirements matrix: generated from test run, written to RESULTS.md for archival and VERIFICATION.md for reporting"

requirements-completed:
  - XML-01
  - XML-02
  - XML-03
  - DOC-01
  - DOC-02
  - DOC-03
  - DOC-04

duration: 3min
completed: 2026-02-18
---

# Phase 9 Plan 04: XML/DOC Verification + Master Runner + Requirements Matrix Summary

**Full 46-requirement verification suite with master runner producing 100% pass rate matrix — XML round-trips, documentation checks, and 7-step connected lifecycle test all PASS**

## Performance

- **Duration:** 3 minutes
- **Started:** 2026-02-18T02:53:20Z
- **Completed:** 2026-02-18T02:56:00Z
- **Tasks:** 2
- **Files created:** 6 test/reporting artifacts

## Accomplishments

- Built `test-xml.sh` verifying all 3 XML requirements via export+import round-trips in isolated environments — pheromone XML (with `phase_end` known issue documented), wisdom XML, and registry XML all produce well-formed XML and import back successfully
- Built `test-doc.sh` verifying all 4 documentation requirements — DOC-01 (learning-promote/inject subcommands), DOC-02 (eternal-init + queen-promote for persistent memory), DOC-03 (session-init with current_phase + session-update with suggested_next), DOC-04 (HANDOFF.md referenced in continue.md + entomb.md)
- Built `test-lifecycle.sh` as a connected 7-step integration test running init → colonize → plan → build → continue → seal → entomb as a real isolated git project — state flows from each step to the next, 7/7 steps PASS
- Built `run-all-e2e.sh` master runner invoking all 11 area test scripts + lifecycle test via subprocess with `--results-file` flag, reading file-based results into 46-row requirements matrix — final result: 46/46 PASS (100%)
- Produced `09-VERIFICATION.md` with executive summary, full requirements matrix, lifecycle test documentation, known issues (XSD phase_end), all Phase 9 fixes, and test infrastructure reference
- Updated `REQUIREMENTS.md` with all 46 checkboxes marked `[x]` (verified PASS)

## Task Commits

1. **Task 1: XML, DOC, and lifecycle test scripts** - `1cf0e0d` (feat) — includes xmlstarlet pipefail fixes to pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh
2. **Task 2: Master runner, requirements matrix, verification report** - `fa08a71` (feat)

**Plan metadata:** (docs commit follows)

## Files Created

- `/Users/callumcowie/repos/Aether/tests/e2e/test-xml.sh` — XML-01/02/03 automated assertions (294 lines)
- `/Users/callumcowie/repos/Aether/tests/e2e/test-doc.sh` — DOC-01/02/03/04 automated assertions (346 lines)
- `/Users/callumcowie/repos/Aether/tests/e2e/test-lifecycle.sh` — Full connected workflow integration test (518 lines)
- `/Users/callumcowie/repos/Aether/tests/e2e/run-all-e2e.sh` — Master runner generating requirements matrix (316 lines)
- `/Users/callumcowie/repos/Aether/tests/e2e/RESULTS.md` — Auto-generated requirements matrix (46/46 PASS)
- `/Users/callumcowie/repos/Aether/.planning/phases/09-polish-verify/09-VERIFICATION.md` — Executive summary + full matrix (180 lines)

## Files Modified

- `/Users/callumcowie/repos/Aether/.aether/exchange/pheromone-xml.sh` — Fixed content field jq expression for plain string `.content`
- `/Users/callumcowie/repos/Aether/.aether/exchange/wisdom-xml.sh` — Fixed xmlstarlet pipefail with `set +e` subshell pattern
- `/Users/callumcowie/repos/Aether/.aether/exchange/registry-xml.sh` — Fixed xmlstarlet pipefail with `set +e` subshell pattern
- `/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md` — All 46 checkboxes updated from `[ ]` to `[x]`

## Decisions Made

- **pheromone-xml.sh content field handling:** The export function used `jq -r '.content.text // .message // ""'` which fails with `Cannot index string with string "text"` when `.content` is a plain string (the standard pheromones.json format). Fixed to use a type-aware jq expression: `if (.content | type) == "string" then .content elif .content.text then .content.text else .message // "" end`.
- **xmlstarlet pipefail fix:** `xmlstarlet sel` returns exit code 1 when no nodes match. Under `set -euo pipefail`, this caused wisdom-import-xml and registry-import-xml to exit silently. Fixed with `set +e` in subshell pattern: `output=$(set +e; xmlstarlet sel ...; true)`.
- **lifecycle test isolation:** Uses its own `mktemp -d` (not `setup_e2e_env`) to create a genuine git repo with `git init`, `README.md`, and `git commit`. This ensures the lifecycle test exercises real path behavior without the XML exchange fixtures that `setup_e2e_env` copies.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed pheromone-xml.sh content field handling**
- **Found during:** Task 1 (first run of test-xml.sh)
- **Issue:** `xml-pheromone-export` used `jq -r '.content.text // .message // ""'` which crashes with "Cannot index string with string "text"" when `.content` is a plain string in pheromones.json
- **Fix:** Changed to type-aware jq: `if (.content | type) == "string" then .content elif .content.text then .content.text else .message // "" end`
- **Files modified:** `.aether/exchange/pheromone-xml.sh`
- **Verification:** XML-01 PASS (export returns ok:true, XML well-formed, import round-trip ok:true)
- **Committed in:** `1cf0e0d` (Task 1 commit)

**2. [Rule 1 - Bug] Fixed wisdom-xml.sh xmlstarlet pipefail**
- **Found during:** Task 1 (first run of test-xml.sh for XML-02)
- **Issue:** `xmlstarlet sel` returns exit code 1 when querying empty wisdom XML. Under `set -euo pipefail`, the script exited silently on import, producing no output (including no ok:true)
- **Fix:** Wrapped xmlstarlet calls in `set +e` subshell pattern
- **Files modified:** `.aether/exchange/wisdom-xml.sh`
- **Verification:** XML-02 PASS (wisdom export+import round-trip ok:true)
- **Committed in:** `1cf0e0d` (Task 1 commit)

**3. [Rule 1 - Bug] Fixed registry-xml.sh xmlstarlet pipefail**
- **Found during:** Task 1 (first run of test-xml.sh for XML-03)
- **Issue:** Same xmlstarlet pipefail issue as wisdom-xml.sh for the colony extraction step in `xml-registry-import`
- **Fix:** Same `set +e` subshell pattern applied to registry import
- **Files modified:** `.aether/exchange/registry-xml.sh`
- **Verification:** XML-03 PASS (registry export+import round-trip ok:true)
- **Committed in:** `1cf0e0d` (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 bugs in XML exchange scripts)
**Impact on plan:** All fixes required for XML round-trip correctness. No scope changes. XML test scripts and verification report created exactly as planned.

## Issues Encountered

None beyond the three auto-fixed bugs above. All DOC requirements passed on first run — the documentation infrastructure (learning-promote, eternal-init, queen-promote, session-init, session-update, CONTEXT.md references, HANDOFF.md references) was correctly in place from Phases 4-6 implementations.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 9 is complete — all 4 plans finished, all 46 requirements verified PASS
- The entire Aether Repair & Stabilization project is complete
- Full e2e test suite (`bash tests/e2e/run-all-e2e.sh`) is available for regression testing
- VERIFICATION.md provides plain-English executive summary of what was tested and confirmed

## Self-Check: PASSED

- FOUND: tests/e2e/test-xml.sh (294 lines, above 80-line minimum)
- FOUND: tests/e2e/test-doc.sh (346 lines, above 60-line minimum)
- FOUND: tests/e2e/test-lifecycle.sh (518 lines, above 160-line minimum)
- FOUND: tests/e2e/run-all-e2e.sh (316 lines, above 100-line minimum)
- FOUND: .planning/phases/09-polish-verify/09-VERIFICATION.md (180 lines, above 80-line minimum)
- FOUND commit: 1cf0e0d (Task 1: XML/DOC/lifecycle tests + xmlstarlet fixes)
- FOUND commit: fa08a71 (Task 2: master runner + RESULTS.md + VERIFICATION.md + REQUIREMENTS.md)
- CONFIRMED: bash tests/e2e/run-all-e2e.sh → 46/46 PASS (100%)

---

*Phase: 09-polish-verify*
*Completed: 2026-02-18*
