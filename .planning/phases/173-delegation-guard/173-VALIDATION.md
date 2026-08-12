---
phase: 173
slug: delegation-guard
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-08-12
approved: 2026-08-12
---

# Phase 173 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `173-RESEARCH.md` § Validation Architecture. Governed by `CLAUDE.md`'s
> Definition of Done — *a requirement is satisfied only when a command exists that
> someone can run, and that command fails when the requirement is unmet.*

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib testing, Go toolchain already present) |
| **Config file** | none — `go.mod` at repo root; no new dependencies required |
| **Quick run command** | `go test ./cmd -run 'TestSpawn\|TestDelegation' -count=1` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | quick ~5s · full ~90s (18 packages) |

**CI registration is mandatory, not optional.** Any new guard test MUST be added to
the `-run` filter at `.github/workflows/ci.yml:100` or it does not run in CI.
Phase 172 built the ratchet that catches exactly this omission —
`TestWiringGateStepRunsEveryWiringTest` and `TestNoRegisteredSubcommandIsUnreferenced`.
`cmd/spawn_enforce_test.go` is already tracked by that ratchet, so new guards placed
alongside it inherit CI enforcement for free.

Registration is deliberately deferred to plan 10 rather than done per-plan. A test name
added to the CI filter before its file joins `wiringGateGuardFiles` makes
`TestWiringGateStepRunsEveryWiringTest` fail on a stale alternative, so plans 03, 05, 06,
08 and 09 are each explicitly forbidden from touching the filter. Plan 10 registers all
five files and every test name in one consistent change, and its `depends_on` names all
six producing plans — so no test can be orphaned by the deferral. Until plan 10 lands the
new tests still run under the blanket `go test ./...` release gate.

---

## Sampling Rate

- **After every task commit:** `go test ./cmd -run 'TestSpawn\|TestDelegation' -count=1`
- **After every plan wave:** `go test ./... -race`
- **Before `/gsd-verify-work`:** full suite green, and `go vet ./...` clean
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

Populated by the planner. Every task must map to a runnable command that fails when
its requirement is unmet. Requirement → proof shape, derived from RESEARCH.md:

| Requirement | Proof shape (must FAIL when unmet) | Test Type |
|---|---|---|
| SPAWN-01 | `aether spawn-can-spawn --depth 99` returns `can_spawn: false` with a named reason; a scripted spawn one level past the cap exits non-zero **and writes no spawn-tree entry** (assert entry count unchanged) | unit + CLI |
| SPAWN-02 | Build a 3-deep tree where every caller passes `--depth 0`; `aether spawn-tree-depth` reports 2 (root=0 per D-05). Caller-supplied value must be provably ignored | unit |
| SPAWN-03 | Wave cap 8, tree budget 20: a run never exceeding 8 per wave is refused at worker 21; budget consumed in wave 1 is NOT restored in wave 2 (assert both numbers move independently) | unit |
| SPAWN-04 | **Every** delegation guard denies and exits non-zero when the inputs **it** reads cannot be resolved: `spawn-can-spawn-swarm` on unreadable `COLONY_STATE.json` (which today returns `can_spawn: true` on exactly this path), `spawn-can-spawn` and `spawn-log` on unreadable run state or an unrecorded requester/parent, and the hook on a requester depth it cannot resolve. Per-guard, not a cross product — see the third Known Residue below | unit + fault injection |
| SPAWN-05 | An A→B→A chain within the depth cap is refused with the ancestor named | unit |
| SPAWN-06 | A test asserts the manifest worker's recorded depth equals 1 (the D-05 convention), and that `.aether/workers.md` no longer instructs `--depth 0` for children | unit + corpus assertion |
| SPAWN-07 | `aether spawn-tree-active` renders the tree indented by depth with parent attribution **while a run is in progress** — asserted against a live, non-finalised tree, not a completed one | unit |
| SPAWN-08 | An abandoned child is reaped, its budget released (assert remaining budget increases), and a command lists and clears orphans | unit |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] **Empirical `PreToolUse` check (BLOCKING for SPAWN-04).** RESEARCH.md rates the
      hook mechanism MEDIUM confidence and names a direct conflict between current
      official Anthropic documentation and closed GitHub issue #34692 on whether
      hooks fire inside subagent contexts at all, and whether `agent_type` carries
      `subagent_type`. Verify live before locking the hook design. If hooks do not
      fire in subagent context, SPAWN-04's hook half must be re-scoped and the
      residue named explicitly rather than implied (D-21). Owned by plan 01;
      plan 07 task 1 opens with a gate that stops on an `INCONCLUSIVE` verdict.
- [x] **No new framework install required** — `go test` infrastructure exists.
- [x] Confirm where the reaper's configurable threshold (D-17) lives. Settled:
      `SpawnReapThresholdMinutes` on `colony.ColonyState`, per plan 09 — the
      option RESEARCH.md open question 3 recommended.

*(Wave 0 is plan 01 itself and has not yet run, so `wave_0_complete` stays false.
The two settled items above are recorded as settled; the blocking item is not.)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The rendered delegation tree reads as English to a non-technical operator (D-14) | SPAWN-07 | Readability is a human judgement; a test can assert structure and absence of raw JSON, but not comprehensibility | Run a real multi-worker build, run `aether spawn-tree-active` mid-run, and confirm the owner can tell who called whom and what they are doing without reading code |
| The refusal reason reaches the operator's ceremony narration (D-10) | SPAWN-01, SPAWN-03 | The runtime tier is covered by an automated test; whether the reason renders inside `build.md` / `continue.md` narration is a wrapper-tier question no plan in this phase touches | Run a build that hits the depth cap or the budget ceiling and confirm the reason is visible without reading raw JSON. Recorded as residue 8 in `173-RESIDUE.md`; if invisible, it is a wrapper-tier follow-up, not a reopening of this phase |

---

## Known Residue (name it, do not imply coverage)

Per D-21 and `172-STOP-RULE.md`, criteria must be bounded claims. Three items are
known NOT to be covered and must be stated rather than glossed:

1. **`spawn-can-spawn` alone is advisory.** Only `spawn-log` is authoritative for
   depth, because `spawn-can-spawn` accepts a caller-supplied depth. A caller can
   lie to the checker; it cannot lie to the recorder. RESEARCH.md open question 1
   asks whether `--name` should be added to verify claimed depth against the tree.
   Settled by plan 04: `--name` is added as an optional flag, and when it resolves,
   the recorded depth overrides the caller's. Omitting it stays advisory, and that
   remains the residue.
2. **The hook's requester-identity bridge.** There is no existing mapping from
   Claude Code's `agent_id` to Aether's spawn-tree `AgentName`. RESEARCH.md proposes
   a bounded heuristic (`aether-*` caste ⇒ depth 1, `general-purpose` ⇒ depth 2) and
   rates it MEDIUM. A naive "deny inside any subagent" hook would over-deny the
   depth-1→depth-2 case D-01 explicitly permits.
3. **Fail-closed is per-guard, not universal across both state files.** Verified
   against source 2026-08-12: `cmd/spawn.go` contains no `COLONY_STATE.json`
   reference, `cmd/root.go`'s `PersistentPreRunE` only builds the storage handle,
   and `hookSpawnDenyReason` decides from its own stdin payload. Only
   `spawn-can-spawn-swarm` reads colony state. The provable claim is therefore
   *each guard denies when the inputs it actually reads cannot be resolved*, not
   *every guard denies whenever colony state is unreadable* — three of the four
   will correctly allow in that scenario. Plan 10 task 2 encodes this as a per-guard
   fault-axis table and asserts, via `go/parser`, that no guard has since started
   reading colony state; if one does, the test fails and forces both the table and
   the residue to widen. ROADMAP criterion 4 is narrowed to match, following the
   precedent already recorded for Phase 172's criteria 1 and 4.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] Every new guard test added to `.github/workflows/ci.yml:100`'s `-run` filter
      *(owned by plan 10, whose `depends_on` names all six producing plans so none can be orphaned)*
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-08-12 — every task across plans 01-10 carries a runnable
`<automated>` verify, no watch-mode flags are used, and Wave 0 is plan 01 itself.
`wave_0_complete` stays false until plan 01 actually runs.
