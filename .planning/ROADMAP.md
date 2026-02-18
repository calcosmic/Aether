# Roadmap: Aether

## Milestones

- ✅ **v1.0 Repair & Stabilization** — Phases 1-9 (shipped 2026-02-18)
- ✅ **v1.1 Colony Polish & Identity** — Phases 10-13 (shipped 2026-02-18)
- 🚧 **v1.2 Hardening & Reliability** — Phases 14-18 (in progress)

## Phases

<details>
<summary>✅ v1.0 Repair & Stabilization (Phases 1-9) — SHIPPED 2026-02-18</summary>

- [x] Phase 1: Diagnostic (3 plans) — 120 tests, 66% pass, 9 critical failures identified
- [x] Phase 2: Core Infrastructure (5 plans) — fixed command foundations
- [x] Phase 3: Visual Experience (2 plans) — swarm display, emoji castes, colors
- [x] Phase 4: Context Persistence (2 plans) — drift detection, rich resume dashboard
- [x] Phase 5: Pheromone System (3 plans) — FOCUS/REDIRECT/FEEDBACK, auto-injection, eternal memory
- [x] Phase 6: Colony Lifecycle (3 plans) — seal ceremony, entomb archival, tunnels browser
- [x] Phase 7: Advanced Workers (3 plans) — oracle, chaos, archaeology, dream, interpret synced
- [x] Phase 8: XML Integration (4 plans) — pheromone/wisdom/registry XML, seal export, entomb hard-stop
- [x] Phase 9: Polish & Verify (4 plans) — 46/46 requirements PASS, full e2e test suite

**46 requirements verified. Full details: `.planning/milestones/v1.0-ROADMAP.md`**

</details>

<details>
<summary>✅ v1.1 Colony Polish & Identity (Phases 10-13) — SHIPPED 2026-02-18</summary>

- [x] Phase 10: Noise Reduction (4 plans) — bash descriptions on 34 commands, ~40% header reduction, version cache
- [x] Phase 11: Visual Identity (6 plans) — ━━━━ banners, progress bars, Next Up blocks, canonical caste-system.md
- [x] Phase 12: Build Progress (2 plans) — spawn announcements, completion lines, BUILD SUMMARY, tmux gating
- [x] Phase 13: Distribution Reliability (1 plan) — .update-pending sentinel, atomic recovery, version detection fix

**14/15 requirements satisfied. Full details: `.planning/milestones/v1.1-ROADMAP.md`**

</details>

---

### 🚧 v1.2 Hardening & Reliability (In Progress)

**Milestone Goal:** Fix every documented bug, clean up the distribution chain, and leave a bulletproof foundation for new features. All five phases publish together in one `npm install -g .` cycle.

- [x] **Phase 14: Foundation Safety** - Fix fallback json_err signature and template path resolution to unblock all subsequent work (completed 2026-02-18)
- [x] **Phase 15: Distribution Chain** - Correct update-transaction.js source directory, update EXCLUDE_DIRS atomically, remove dead duplicates, sync allowlist (completed 2026-02-18)
- [ ] **Phase 16: Lock Lifecycle Hardening** - Audit all acquire/release pairs, eliminate deadlocks on jq failure, add trap-based cleanup on all exit paths
- [ ] **Phase 17: Error Code Standardization** - Replace all hardcoded strings with E_* constants in json_err calls, document error codes
- [ ] **Phase 18: Reliability & Architecture Gaps** - Wire temp file cleanup, rotate spawn-tree, add exec error handling, document queen commands, validate JSON output

## Phase Details

### Phase 14: Foundation Safety
**Goal**: Establish a safe base where error code work cannot silently break callers and npm-installed users are not blocked by a template path bug
**Depends on**: Phase 13 (v1.1 shipped)
**Requirements**: ERR-01, ARCH-01
**Success Criteria** (what must be TRUE):
  1. Running `json_err "$E_FILE_NOT_FOUND" "message"` from a bash session where error-handler.sh failed to load still produces output with both a code field and the human-readable message
  2. A user running `queen-init` from an npm-installed copy of Aether (not a git clone) reaches the template without hitting a missing-directory error
  3. Neither fix changes any success-path behavior — commands that work today still work identically
**Plans:** 1/1 plans complete
Plans:
- [ ] 14-01-PLAN.md — Fix fallback json_err signature (ERR-01) and template path resolution (ARCH-01)

### Phase 15: Distribution Chain
**Goal**: Every `aether update` call copies exactly the right files — system files land in `.aether/`, hub metadata never syncs to target repos, no dead duplicates pollute the source tree
**Depends on**: Phase 14
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-06
**Success Criteria** (what must be TRUE):
  1. After `aether update` on a clean test repo, `.aether/` contains system files only — no `version.json`, `registry.json`, `manifest.json`, or `chambers/` entries from the hub root
  2. After `aether update`, `commands/`, `agents/`, and `rules/` subdirectories from `~/.aether/system/` are not duplicated into `.aether/`
  3. `caste-system.md` is present in a target repo after `aether update` (was missing from allowlist)
  4. `planning.md` phantom file is absent from all sync allowlists and does not appear in target repos
  5. The `.aether/agents/` and `.aether/commands/` dead duplicate directories are gone from the source repo
  6. Old 2.x npm versions are deprecated on the registry — `npm install -g aether` installs the current version
**Plans:** 3/3 plans complete
Plans:
- [ ] 15-01-PLAN.md — Fix source directory, EXCLUDE_DIRS, and allowlists (DIST-01, DIST-02, DIST-04, DIST-05)
- [ ] 15-02-PLAN.md — Remove dead duplicate directories from source repo (DIST-03)
- [ ] 15-03-PLAN.md — Stale-dir cleanup, user feedback, tests, and npm deprecation (DIST-06)

### Phase 16: Lock Lifecycle Hardening
**Goal**: Lock deadlocks are impossible when jq fails — every lock acquired is released on every exit path, including error branches
**Depends on**: Phase 14
**Requirements**: LOCK-01, LOCK-02, LOCK-03, LOCK-04
**Success Criteria** (what must be TRUE):
  1. Feeding invalid JSON as `flags.json` to flag-add, flag-auto-resolve, or flag-acknowledge leaves `.aether/locks/` empty after the command exits — no stale lock files
  2. Sending SIGTERM or SIGINT to a command holding a lock releases the lock before the process exits
  3. A simulated race on atomic-write backup creation does not corrupt the target file
  4. Concurrent `context-update` calls from two processes produce a valid merged result, not a half-written file
**Plans:** 3 plans
Plans:
- [ ] 16-01-PLAN.md — Unify trap pattern in flag commands + stale lock user prompt (LOCK-01, LOCK-02)
- [ ] 16-02-PLAN.md — Add locking to context-update + force-unlock subcommand (LOCK-04)
- [ ] 16-03-PLAN.md — Lock lifecycle tests + known-issues.md updates (LOCK-01, LOCK-02, LOCK-03, LOCK-04)

### Phase 17: Error Code Standardization
**Goal**: Every json_err call in aether-utils.sh produces machine-readable output with a structured code field — zero hardcoded strings remaining
**Depends on**: Phase 14
**Requirements**: ERR-02, ERR-03, ERR-04
**Success Criteria** (what must be TRUE):
  1. Triggering any documented error condition (file not found, permission denied, tool not installed) produces JSON with a `"code":"E_..."` field — no bare string codes in output
  2. An automated grep of aether-utils.sh for `json_err "` (bare string as first arg, not a variable) returns zero matches
  3. A contributor can look up any error constant in `.aether/docs/error-codes.md` and find its meaning and when to use it
  4. Error path tests for lock and flag operations execute without false positives and catch a deliberately introduced hardcoded-string call
**Plans**: TBD

### Phase 18: Reliability & Architecture Gaps
**Goal**: Stale resources stop accumulating, exec errors are caught, queen commands are discoverable, and JSON output is validated before leaving the read layer
**Depends on**: Phase 16, Phase 17
**Requirements**: ARCH-02, ARCH-03, ARCH-04, ARCH-05, ARCH-06, ARCH-07, ARCH-08, ARCH-09, ARCH-10
**Success Criteria** (what must be TRUE):
  1. After a session ends, `.aether/temp/` contains no orphaned `.tmp` files and `spawn-tree.txt` does not grow unboundedly across sessions
  2. `model-get` and `model-list` return a clear error message (not a silent hang or exit 0 with no output) when the underlying exec call fails
  3. Running `aether help` (or the equivalent help command) lists queen-* commands alongside all other available commands
  4. `queen-read` returns an error rather than invalid JSON when the state file contains malformed content
  5. Feature detection in aether-utils.sh completes without a race against error-handler.sh loading — no "function not found" errors on startup
**Plans**: TBD

---

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Diagnostic | v1.0 | 3/3 | Complete | 2026-02-18 |
| 2. Core Infrastructure | v1.0 | 5/5 | Complete | 2026-02-18 |
| 3. Visual Experience | v1.0 | 2/2 | Complete | 2026-02-18 |
| 4. Context Persistence | v1.0 | 2/2 | Complete | 2026-02-18 |
| 5. Pheromone System | v1.0 | 3/3 | Complete | 2026-02-18 |
| 6. Colony Lifecycle | v1.0 | 3/3 | Complete | 2026-02-18 |
| 7. Advanced Workers | v1.0 | 3/3 | Complete | 2026-02-18 |
| 8. XML Integration | v1.0 | 4/4 | Complete | 2026-02-18 |
| 9. Polish & Verify | v1.0 | 4/4 | Complete | 2026-02-18 |
| 10. Noise Reduction | v1.1 | 4/4 | Complete | 2026-02-18 |
| 11. Visual Identity | v1.1 | 6/6 | Complete | 2026-02-18 |
| 12. Build Progress | v1.1 | 2/2 | Complete | 2026-02-18 |
| 13. Distribution Reliability | v1.1 | 1/1 | Complete | 2026-02-18 |
| 14. Foundation Safety | v1.2 | Complete    | 2026-02-18 | - |
| 15. Distribution Chain | v1.2 | Complete    | 2026-02-18 | - |
| 16. Lock Lifecycle Hardening | v1.2 | 0/3 | Planned | - |
| 17. Error Code Standardization | v1.2 | 0/TBD | Not started | - |
| 18. Reliability & Architecture Gaps | v1.2 | 0/TBD | Not started | - |

---

*Roadmap created: 2026-02-17*
*v1.0 shipped: 2026-02-18*
*v1.1 shipped: 2026-02-18*
*v1.2 roadmap added: 2026-02-18*
