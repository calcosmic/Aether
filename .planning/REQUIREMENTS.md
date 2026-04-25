# Requirements: Aether v1.8

**Defined:** 2026-04-25
**Core Value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## v1.8 Requirements

### Detection (Scanner)

- [ ] **DETECT-01** (R073): `aether recover` detects missing build packet — build was started (phase marked EXECUTING) but no build packet file exists on disk
- [ ] **DETECT-02** (R074): `aether recover` detects stale spawned workers — spawned.json references workers that never completed and have exceeded the abandoned threshold
- [ ] **DETECT-03** (R075): `aether recover` detects partial phase — phase marked in_progress but no real build artifacts exist
- [ ] **DETECT-04** (R076): `aether recover` detects bad manifest — manifest JSON is unparseable or references files/directories that don't exist
- [ ] **DETECT-05** (R077): `aether recover` detects dirty worktree — uncommitted changes left by an interrupted worker in a worktree-parallel colony
- [ ] **DETECT-06** (R078): `aether recover` detects broken survey — survey data files under .aether/data/survey/ are incomplete or reference missing files
- [ ] **DETECT-07** (R079): `aether recover` detects missing agent files — expected agent definitions are absent from .claude/agents/ant/ or .opencode/agents/

### Output

- [ ] **OUTP-01** (R080): `aether recover` prints a single clean diagnosis listing all detected issues with severity and a one-line description (not debug output)
- [ ] **OUTP-02** (R081): `aether recover` exits 0 when no issues found, exits 1 when issues detected (usable in scripts)

### Repair

- [ ] **REPAIR-01** (R082): `aether recover --apply` auto-fixes the 5 safe classes: missing packet, stale workers, partial phase, broken survey, missing agents
- [ ] **REPAIR-02** (R083): `aether recover --apply` requires user confirmation for dirty worktree fixes (stash or discard)
- [ ] **REPAIR-03** (R084): `aether recover --apply` requires user confirmation for corrupted manifest repair (rebuild from disk state)
- [ ] **REPAIR-04** (R085): All repairs create backups before mutating state, following the existing medic backup-first pattern
- [ ] **REPAIR-05** (R086): Repairs that touch multiple state files do so atomically — all succeed or all roll back

### Verification

- [ ] **TEST-01** (R087): E2E test proving recovery from each of the 7 stuck states individually
- [ ] **TEST-02** (R088): E2E test proving recovery from a compound stuck state (multiple issues simultaneously)
- [ ] **TEST-03** (R089): Test proving recover does not false-positive on an active, healthy colony

## Deferred

- **PERF-01** (R016): Pheromone markets and reputation exchange
- **FED-01** (R017): Federation and inter-colony coordination
- **EVO-01** (R018): Evolution engine / self-modifying agents

## Out of Scope

| Feature | Reason |
|---------|--------|
| Plan --force integration | v1.7 already handles that path; recover handles runtime state |
| New agent castes | No new agents needed for recovery |
| Visual/UX changes | Recovery is CLI-native, presentation follows existing patterns |
| Medic replacement | Recover is specialized for stuck-state; medic remains general health |
| Fallback plan quality | Recovery is about state repair, not plan quality |
| Publish pipeline changes | v1.6 shipped that; not revisiting |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| DETECT-01 (R073) | Phase 49 | Pending |
| DETECT-02 (R074) | Phase 49 | Pending |
| DETECT-03 (R075) | Phase 49 | Pending |
| DETECT-04 (R076) | Phase 49 | Pending |
| DETECT-05 (R077) | Phase 49 | Pending |
| DETECT-06 (R078) | Phase 49 | Pending |
| DETECT-07 (R079) | Phase 49 | Pending |
| OUTP-01 (R080) | Phase 49 | Pending |
| OUTP-02 (R081) | Phase 49 | Pending |
| REPAIR-01 (R082) | Phase 50 | Pending |
| REPAIR-02 (R083) | Phase 50 | Pending |
| REPAIR-03 (R084) | Phase 50 | Pending |
| REPAIR-04 (R085) | Phase 50 | Pending |
| REPAIR-05 (R086) | Phase 50 | Pending |
| TEST-01 (R087) | Phase 51 | Pending |
| TEST-02 (R088) | Phase 51 | Pending |
| TEST-03 (R089) | Phase 51 | Pending |

**Coverage:**
- v1.8 requirements: 16 total
- Mapped to phases: 16
- Unmapped: 0

---
*Requirements defined: 2026-04-25*
