# Phase 59: Gate Failure Recovery - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-27
**Phase:** 59-gate-failure-recovery
**Areas discussed:** Recovery instructions quality, Watcher Veto confirmation, Gate state tracking, Incremental recheck UX

---

## Recovery Instructions Quality

| Option | Description | Selected |
|--------|-------------|----------|
| Per-gate templates | Each gate type has hardcoded recovery steps: what to check, which command, what success looks like | ✓ |
| Better evidence, generic steps | Failure includes specific evidence but recovery stays generic ("fix these and re-run") | |
| Structured recovery state | Machine-readable JSON recovery state for tools/scripts | |

**User's choice:** Per-gate templates
**Notes:** Each of the ~14 gates gets a tailored recovery template. Not just "FAILED" — actionable steps.

---

## Multiple Failure Presentation

| Option | Description | Selected |
|--------|-------------|----------|
| Show all failures together | All failed gates displayed with individual recovery templates. Fix once, re-run once. | ✓ |
| Stop at first failure | Show first failure, stop. Fix, re-run, hit next failure, etc. | |
| All failures + dependency hints | Show all failures with hints about which can be fixed in parallel vs sequentially. | |

**User's choice:** Show all failures together
**Notes:** User sees everything that needs fixing at once. Works well with incremental recheck — fix all, re-run once, only failed gates get re-checked.

---

## Watcher Veto Confirmation

| Option | Description | Selected |
|--------|-------------|----------|
| Show reason + 3 choices | Veto reason shown first, then: Stash / Keep working / Force advance | ✓ |
| Show reason + yes/no stash | Veto reason shown, simple yes/no for stashing | |
| Conditional auto-stash or ask | Critical issues = auto-stash, low quality = ask first | |

**User's choice:** Show reason + 3 choices
**Notes:** "Stash changes" runs git stash + blocker flag. "Keep working" leaves phase blocked, user fixes manually. "Force advance" overrides with FEEDBACK pheromone noting the override.

---

## Gate State Storage

| Option | Description | Selected |
|--------|-------------|----------|
| COLONY_STATE.json | Add `gate_results` field to existing colony state file | ✓ |
| Separate gate-checkpoints file | New `.aether/data/gate-checkpoints.json` file | |
| State-mutate namespaced path | Same storage, different key path via state-mutate | |

**User's choice:** COLONY_STATE.json
**Notes:** Follows existing pattern. Cleaned up on phase advance.

---

## Gate Results Schema

| Option | Description | Selected |
|--------|-------------|----------|
| Flat gate results list | `[{name, passed, timestamp, detail}]` — simple, easy to filter | ✓ |
| Categorized by gate type | Grouped by verification/gates/lifecycle — more structure | |

**User's choice:** Flat list
**Notes:** Flat array keyed by gate name. Filter `passed: false` for failures.

---

## Incremental Recheck UX

| Option | Description | Selected |
|--------|-------------|----------|
| Skip summary + failed only | "Skipping 8 passed gates" then only re-run failures | ✓ |
| Show each skipped gate | "✓ tests_pass — skipping" for each passed gate | |
| Silent skip, failed only | Just run failed gates silently | |

**User's choice:** Skip summary + failed only
**Notes:** One-line summary of what's being skipped, then only the failures get re-run.

---

## Staleness Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Never skip tests | Tests always re-run — cheapest hard truth check | ✓ |
| Git diff triggers recheck | Re-run tests + affected gates if files changed | |
| Time-based expiry (30 min) | Re-run everything after 30 minutes | |

**User's choice:** Never skip tests
**Notes:** Tests are cheap and catch code changes. Other passed gates are trusted from last run.

---

## Claude's Discretion

- Exact recovery template wording for each gate type
- Exact AskUserQuestion format for veto confirmation
- Whether gate_results lives at top level or nested under current phase
- Edge case handling for phase changes between runs

## Deferred Ideas

None — discussion stayed within phase scope.
