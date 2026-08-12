---
phase: 173
slug: delegation-guard
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-12
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
| SPAWN-04 | With `COLONY_STATE.json` and the spawn tree made unreadable, **every** delegation guard denies and exits non-zero — including `spawn-can-spawn-swarm`, which today returns `can_spawn: true` on exactly this path. Hook denies a spawn whose requester depth cannot be resolved | unit + fault injection |
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
      residue named explicitly rather than implied (D-21).
- [ ] **No new framework install required** — `go test` infrastructure exists.
- [ ] Confirm where the reaper's configurable threshold (D-17) lives. RESEARCH.md
      recommends a new optional field on `colony.ColonyState`.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The rendered delegation tree reads as English to a non-technical operator (D-14) | SPAWN-07 | Readability is a human judgement; a test can assert structure and absence of raw JSON, but not comprehensibility | Run a real multi-worker build, run `aether spawn-tree-active` mid-run, and confirm the owner can tell who called whom and what they are doing without reading code |

---

## Known Residue (name it, do not imply coverage)

Per D-21 and `172-STOP-RULE.md`, criteria must be bounded claims. Two items are
known NOT to be covered and must be stated rather than glossed:

1. **`spawn-can-spawn` alone is advisory.** Only `spawn-log` is authoritative for
   depth, because `spawn-can-spawn` accepts a caller-supplied depth. A caller can
   lie to the checker; it cannot lie to the recorder. RESEARCH.md open question 1
   asks whether `--name` should be added to verify claimed depth against the tree.
   Whichever way that lands, the residue must be written down.
2. **The hook's requester-identity bridge.** There is no existing mapping from
   Claude Code's `agent_id` to Aether's spawn-tree `AgentName`. RESEARCH.md proposes
   a bounded heuristic (`aether-*` caste ⇒ depth 1, `general-purpose` ⇒ depth 2) and
   rates it MEDIUM. A naive "deny inside any subagent" hook would over-deny the
   depth-1→depth-2 case D-01 explicitly permits.

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] Every new guard test added to `.github/workflows/ci.yml:100`'s `-run` filter
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
