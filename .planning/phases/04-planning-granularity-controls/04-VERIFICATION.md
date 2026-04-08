---
phase: 04-planning-granularity-controls
verified: 2026-04-08T12:00:00Z
status: gaps_found
score: 13/17 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/17
  gaps_closed:
    - "plan-granularity get returns persisted value or 'none' with source indicator"
    - "plan-granularity set validates against enum before persisting"
    - "/ant:status displays current granularity setting with human-readable description"
    - "state-mutate validates plan_granularity values through enum Valid()"
    - "Route-setter agent receives dynamic min/max phase bounds instead of hardcoded 3-6"
    - "Plan command reads persisted granularity from COLONY_STATE.json before planning"
    - "Plan command asks user for granularity if none is persisted and no --granularity flag provided"
    - "Plan command injects dynamic 'Maximum N phases' into route-setter prompt"
    - "Out-of-range plans trigger a warning with accept/adjust/replan options"
    - "All 4 agent definition files have matching changes"
  gaps_remaining:
    - "Autopilot reads persisted plan_granularity from COLONY_STATE.json at startup"
    - "Autopilot warns if the plan has more phases than the selected granularity range allows"
    - "Warning does not block execution -- plan was already accepted by user during /ant:plan"
    - "Both Claude and OpenCode autopilot commands have identical granularity checks (parity)"
  regressions: []
gaps:
  - truth: "Autopilot reads persisted plan_granularity from COLONY_STATE.json at startup"
    status: failed
    reason: "Neither .claude/commands/ant/run.md nor .opencode/commands/ant/run.md contains any granularity reference -- Plan 03 was never executed"
    artifacts:
      - path: ".claude/commands/ant/run.md"
        issue: "Zero granularity references -- no Step 0 granularity check"
      - path: ".opencode/commands/ant/run.md"
        issue: "Zero granularity references -- no Step 0 granularity check"
    missing:
      - "Add granularity check sub-step to Step 0 in both autopilot run.md files"
      - "Update AUTOPILOT ENGAGED display line to include Granularity field"
  - truth: "Autopilot warns if the plan has more phases than the selected granularity range allows"
    status: failed
    reason: "No granularity awareness in autopilot at all -- no phase count comparison against granularity bounds"
    missing:
      - "Add phase count vs granularity bounds comparison in autopilot Step 0"
  - truth: "Warning does not block execution -- plan was already accepted by user during /ant:plan"
    status: failed
    reason: "No warning exists at all, blocking or otherwise"
    missing:
      - "Add non-blocking informational NOTE message for granularity mismatch"
  - truth: "Both Claude and OpenCode autopilot commands have identical granularity checks (parity)"
    status: failed
    reason: "Neither Claude nor OpenCode autopilot has any granularity check"
    artifacts:
      - path: ".claude/commands/ant/run.md"
        issue: "No granularity references"
      - path: ".opencode/commands/ant/run.md"
        issue: "No granularity references"
    missing:
      - "Add matching granularity checks to both autopilot command files"
---

# Phase 4: Planning Granularity Controls Verification Report

**Phase Goal:** Let users control how many phases the plan generates by selecting a granularity range.
**Verified:** 2026-04-08T12:00:00Z
**Status:** gaps_found
**Re-verification:** Yes -- after gap closure (Plans 01 and 02 executed since initial verification)

## Goal Achievement

### Observable Truths

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | PlanGranularity enum has exactly 4 valid values: sprint, milestone, quarter, major | VERIFIED | `pkg/colony/colony.go`: type + 4 const values, Valid() method |
| 2  | GranularityRange returns correct min/max for each preset | VERIFIED | `pkg/colony/granularity.go`: sprint=1,3 milestone=4,7 quarter=8,12 major=13,20; 7 test cases pass |
| 3  | ColonyState serializes plan_granularity field to JSON with snake_case key | VERIFIED | `pkg/colony/colony.go` line 107: `PlanGranularity PlanGranularity \`json:"plan_granularity,omitempty"\`` |
| 4  | plan-granularity get returns persisted value or 'none' with source indicator | VERIFIED | `cmd/colony_cmds.go` lines 153-194: get command returns granularity, source, min, max |
| 5  | plan-granularity set validates against enum before persisting | VERIFIED | `cmd/colony_cmds.go` lines 196-285: set command calls g.Valid() before persisting |
| 6  | /ant:status displays current granularity setting with human-readable description | VERIFIED | `cmd/status.go` lines 123-127: reads PlanGranularity; lines 211-223: granularityLabel function |
| 7  | state-mutate validates plan_granularity values through enum Valid() | VERIFIED | `cmd/state_cmds.go` lines 119-125: case "plan_granularity" with Valid() check |
| 8  | Route-setter agent receives dynamic min/max phase bounds instead of hardcoded 3-6 | VERIFIED | All 3 route-setter files use {granularity_min}/{granularity_max}; zero hardcoded "3-6" remaining |
| 9  | Plan command reads persisted granularity from COLONY_STATE.json before planning | VERIFIED | `.claude/commands/ant/plan.md` lines 74-86: runs `aether plan-granularity get` and extracts bounds |
| 10 | Plan command asks user for granularity if none is persisted and no --granularity flag provided | VERIFIED | `.claude/commands/ant/plan.md` line 79: "ask user" with 4-option menu, defaults to milestone |
| 11 | Plan command injects dynamic 'Maximum N phases' into route-setter prompt | VERIFIED | `.claude/commands/ant/plan.md` line 501: `Maximum {granularity_max} phases` |
| 12 | Out-of-range plans trigger a warning with accept/adjust/replan options | VERIFIED | `.claude/commands/ant/plan.md` Step 4.5 (line 572): 3 options -- Accept, Adjust, Replan |
| 13 | All 4 agent definition files have matching changes | VERIFIED | `.aether/agents-claude/`, `.claude/agents/ant/`, `.opencode/agents/` all updated; `.aether/agents/` path was a wrong assumption in plan (agents-claude IS the mirror per CLAUDE.md) |
| 14 | Autopilot reads persisted plan_granularity from COLONY_STATE.json at startup | FAILED | Neither `.claude/commands/ant/run.md` nor `.opencode/commands/ant/run.md` contains any granularity reference |
| 15 | Autopilot warns if plan exceeds selected granularity range | FAILED | No granularity awareness in autopilot at all |
| 16 | Warning does not block execution | FAILED | No warning exists at all |
| 17 | Both Claude and OpenCode autopilot commands have identical granularity checks | FAILED | Neither file has any granularity check |

**Score:** 13/17 truths verified

### Deferred Items

None. No later milestone phase addresses autopilot granularity awareness. Phase 5 (Orchestration) focuses on task decomposition and agent assignment, not planning granularity.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `pkg/colony/colony.go` | PlanGranularity type, 4 constants, Valid(), ErrInvalidGranularity, ColonyState field | VERIFIED | All present and correct |
| `pkg/colony/granularity.go` | GranularityRange function | VERIFIED | Correct ranges for all 4 presets plus default |
| `pkg/colony/granularity_test.go` | Tests for Valid() and GranularityRange() | VERIFIED | 16 test cases across 2 test functions, all pass |
| `pkg/colony/testdata/COLONY_STATE.golden.json` | plan_granularity field in golden file | VERIFIED | Contains `"plan_granularity": "milestone"` at line 522 |
| `cmd/colony_cmds.go` | plan-granularity get/set command pair | VERIFIED | Lines 147-294: full get/set with Valid() check, bounds display, init() wiring |
| `cmd/status.go` | Granularity display in dashboard | VERIFIED | Lines 123-127 display; granularityLabel function at lines 211-223 |
| `cmd/state_cmds.go` | plan_granularity case in state-mutate | VERIFIED | Field mutation at lines 119-125; field read at lines 732-733 |
| `cmd/testdata/colony_state.json` | plan_granularity field in test fixture | VERIFIED | Contains `"plan_granularity": "milestone"` at line 12 |
| `.aether/agents-claude/aether-route-setter.md` | Dynamic phase bounds | VERIFIED | 5 occurrences of granularity_min/max; zero "3-6" remaining |
| `.claude/agents/ant/aether-route-setter.md` | Dynamic phase bounds | VERIFIED | 5 occurrences of granularity_min/max; zero "3-6" remaining |
| `.opencode/agents/aether-route-setter.md` | Dynamic phase bounds | VERIFIED | 3 occurrences of granularity_min/max; zero "3-6" remaining |
| `.claude/commands/ant/plan.md` | --granularity flag, dynamic bounds, out-of-range validation | VERIFIED | Flag parsing at line 14; selection flow at lines 74-86; dynamic injection at lines 493,501; Step 4.5 at line 572 |
| `.claude/commands/ant/run.md` | Autopilot granularity check | MISSING | Zero granularity references |
| `.opencode/commands/ant/run.md` | Autopilot granularity check | MISSING | Zero granularity references |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `pkg/colony/colony.go` | ColonyState struct | PlanGranularity field with json tag | WIRED | Line 107: field exists with correct json tag |
| `cmd/colony_cmds.go` | `pkg/colony` | import colony.PlanGranularity and Valid() | WIRED | Lines 147-294: full command pair with colony.PlanGranularity usage |
| `cmd/state_cmds.go` | `pkg/colony` | plan_granularity case in switch | WIRED | Lines 119-125: case with Valid() check |
| `.claude/commands/ant/plan.md` | COLONY_STATE.json | reads plan_granularity field | WIRED | Lines 74-86: runs `aether plan-granularity get` |
| `.claude/commands/ant/plan.md` | route-setter | injects min/max bounds | WIRED | Lines 493,501: {granularity_min}-{granularity_max} in prompt |
| `.claude/commands/ant/run.md` | COLONY_STATE.json | reads plan_granularity in Step 0 | NOT_WIRED | No granularity references in file |
| `.opencode/commands/ant/run.md` | COLONY_STATE.json | reads plan_granularity in Step 0 | NOT_WIRED | No granularity references in file |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `cmd/colony_cmds.go` (get) | plan_granularity output | ColonyState + GranularityRange | Yes | FLOWING -- reads persisted state, computes range |
| `cmd/colony_cmds.go` (set) | plan_granularity persist | CLI input -> Valid() -> SaveJSON | Yes | FLOWING -- validates enum then persists |
| `cmd/status.go` | granularity display | ColonyState.PlanGranularity | Yes | FLOWING -- reads from state, maps to label |
| `.claude/commands/ant/plan.md` | granularity_min/max | `aether plan-granularity get` | Yes | FLOWING -- CLI command produces real bounds |
| `.claude/commands/ant/run.md` | N/A | N/A | N/A | DISCONNECTED -- no granularity data flow |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| PlanGranularity.Valid() accepts sprint/milestone/quarter/major | `go test ./pkg/colony/... -run TestPlanGranularityValid -v` | All 10 sub-tests pass | PASS |
| GranularityRange returns correct bounds | `go test ./pkg/colony/... -run TestGranularityRange -v` | All 7 sub-tests pass | PASS |
| `aether plan-granularity` command exists | `go run ./cmd/aether plan-granularity` | Shows get/set subcommands | PASS |
| All cmd tests pass | `go test ./cmd/... -v` | All tests pass | PASS |
| Full test suite passes with race detection | `go test ./... -race -count=1` | All 13 packages pass | PASS |
| Go binary builds | `go build ./cmd/...` | Success (exit 0) | PASS |
| Route-setter files have zero hardcoded "3-6" | `grep -rn "3-6" .aether/agents-claude/aether-route-setter.md .claude/agents/ant/aether-route-setter.md .opencode/agents/aether-route-setter.md` | No matches (exit 1) | PASS |
| Plan.md has zero hardcoded "Maximum 6" | `grep -rn "Maximum 6" .claude/commands/ant/plan.md` | No matches (exit 1) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| PLAN-01 | 04-01, 04-02 | User can select planning granularity with four ranges | SATISFIED | Enum + GranularityRange + CLI commands + plan command --granularity flag + "always ask" behavior all implemented |
| PLAN-02 | 04-02 | Route-setter receives min/max constraints from selected granularity | SATISFIED | All 3 route-setter agent files use {granularity_min}/{granularity_max} placeholders; plan.md injects bounds |
| PLAN-03 | 04-02 | User warned if plan exceeds selected range | SATISFIED | plan.md Step 4.5 with 3 options: Accept, Adjust granularity, Replan |
| PLAN-04 | 04-01 | Granularity persists in COLONY_STATE.json and visible in /ant:status | SATISFIED | ColonyState field + plan-granularity get/set CLI + status.go display with granularityLabel |
| PLAN-05 | 04-03 | Autopilot respects planning granularity | BLOCKED | Neither .claude/commands/ant/run.md nor .opencode/commands/ant/run.md has any granularity awareness |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/state_cmds.go` | 645 | Pre-existing "placeholder" in unrelated command | Info | Not related to this phase's work |

### Human Verification Required

None. All failures and successes are clearly detectable programmatically.

### Gaps Summary

Significant progress since initial verification: 10 of 14 gaps closed (Plans 01 and 02 fully executed). The remaining 4 gaps all share a single root cause: **Plan 03 was never executed**.

The autopilot (both Claude and OpenCode run.md files) has zero granularity awareness. The plan command, route-setter agents, CLI commands, status display, and state persistence are all fully wired and working. The only missing piece is the autopilot Step 0 check that reads persisted granularity and warns on mismatch.

**Roadmap Success Criteria status:** 4 of 5 criteria are met. Criterion 5 ("Running /ant:run after setting granularity respects the phase count across the entire autopilot loop") is the only unmet criterion, and it maps directly to the Plan 03 gap.

---

_Verified: 2026-04-08T12:00:00Z_
_Verifier: Claude (gsd-verifier)_
