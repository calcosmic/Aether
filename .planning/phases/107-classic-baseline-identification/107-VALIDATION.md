# Phase 107: Validation Plan

**Phase:** 107 - Classic Baseline Identification
**Created:** 2026-05-12
**Status:** Ready for execution

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Bash (smoke test) + manual verification |
| Config file | none |
| Quick run command | `bash scripts/smoke-test-classic.sh` |
| Full suite command | `bash scripts/smoke-test-classic.sh && go test ./cmd/ -run TestBoundary -v` |

## Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BASE-01 | v5.4.0 selected with evidence | manual | `cat .aether/references/classic-baseline.md` | Wave 0 |
| BASE-02 | Smoke test passes for Classic v5.4.0 | automated | `bash scripts/smoke-test-classic.sh` | Wave 0 |
| BASE-03 | Baseline document complete with all sections | manual | `grep -c "Selection Rationale\|Known Limitations\|Behavioral Checklist" .aether/references/classic-baseline.md` | Wave 0 |

## Content Assertions (Per-Requirement)

### BASE-01: v5.4.0 selected with evidence
- Document contains "v5.4.0" as the selected version tag
- Selection Rationale section exists with at least 3 numbered reasons
- Version Comparison table covers all 16 modules

### BASE-02: Smoke test passes for Classic v5.4.0
- Script checks out v5.4.0 via `git worktree add --detach`
- Script isolates HOME to prevent delegation shim interference
- Script runs `npm install --production` before CLI commands
- Test 4: sync-state causes observable COLONY_STATE.json mutation (hash comparison)
- Test 5: wrapper files contain ceremony markers (grep for "Stage", "worker", "caste", "Builder")
- D-04 deviation documented in-script (plan/build/continue are slash commands, not CLI subcommands)

### BASE-03: Baseline document complete with all sections
- Classification counts correct: 3 "Restore in TS", 11 "Keep in Go", 3 "Obsolete"
- 5 known limitations present, each with "Workaround" sub-bullet
- Behavioral checklist has entries for all 16 modules
- Cross-references link to runtime-boundary-contract.md

## Sampling Rate

- **Per task commit:** `bash scripts/smoke-test-classic.sh`
- **Per wave merge:** `bash scripts/smoke-test-classic.sh && go test ./... -race`
- **Phase gate:** Smoke test green + baseline document complete + content assertions pass

## Automated Verification Commands

```bash
# Smoke test (BASE-02)
bash scripts/smoke-test-classic.sh

# Baseline document section presence (BASE-01, BASE-03)
grep -c "Selection Rationale\|Known Limitations\|Behavioral Checklist\|Version Comparison\|Cross-References" .aether/references/classic-baseline.md

# Classification count assertions (BASE-03)
grep -c "Restore in TS" .aether/references/classic-baseline.md  # >= 3
grep -c "Keep in Go" .aether/references/classic-baseline.md     # >= 11
grep -c "Obsolete" .aether/references/classic-baseline.md       # >= 3

# Known limitations with workarounds (BASE-03)
grep -c "Workaround" .aether/references/classic-baseline.md     # >= 5

# v5.4.0 as selected version (BASE-01)
grep -c "v5.4.0" .aether/references/classic-baseline.md         # >= 5
```

## Wave 0 Gaps

- [ ] `scripts/smoke-test-classic.sh` -- covers BASE-02
- [ ] `.aether/references/classic-baseline.md` -- covers BASE-01, BASE-03
