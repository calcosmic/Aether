# Aether Shell-to-Go Conversion: Master Execution Plan

**Purpose:** This prompt contains everything needed to verify the Go implementation is complete, ensure zero functionality loss from the shell scripts, clean up the development environment, and get Aether to a clean working state with Go as the primary runtime.

**How to use:** After `/clear`, paste this entire file as your prompt. It contains all context from the research — you don't need to read any other files first.

---

## CONTEXT: WHERE WE ARE

### The Project
Aether is a multi-agent development framework. Users install it via `npm install -g .`, then run slash commands like `/ant:init`, `/ant:build`, `/ant:continue` in Claude Code or OpenCode. Under the hood, those commands call `aether-utils.sh` (a 5,500-line bash dispatcher with 305 subcommands) and ~50 utility scripts in `.aether/utils/`.

### The Goal
Convert the entire shell backend to Go so that:
1. Aether works as a standalone Go binary (`aether init`, `aether build`, etc.)
2. The npm package ships the Go binary instead of (or alongside) the shell scripts
3. Multiple user types can use it (not just Claude Code/OpenCode — anyone with the binary)
4. Zero functionality is lost in the conversion

### Current Status (April 2026)
- **Go binary compiles clean** (`go build ./cmd/aether` succeeds)
- **All 1,028 Go tests pass** across 10 packages (NOT 524 — that was an outdated count)
- **153 Go commands implemented** out of 305 shell subcommands
- **High test coverage:** colony (100%), agent (90.8%), events (87.2%), exchange (93.4%), graph (90.4%), memory (86.1%), storage (64.9%), llm (68.4%)
- **v5.4 milestone (Shell-to-Go Rewrite):** Phases 45-56 complete, Phase 50 (CLI Commands) at 1/6 plans
- **59 planning phases total** (phases 1-44 were shell improvements, 45+ are Go conversion)
- **3 open worktrees** with unmerged branches
- **162 shell subcommands not yet in Go** — but only 28 are critical (called by slash commands)
- **goreleaser.yml IS configured** for cross-platform releases
- **Makefile exists** with build targets
- **Only 4 placeholder/TODO items** found in the entire Go codebase

---

## PART 1: WHAT'S ACTUALLY MISSING FROM GO

### The Real Gap: 28 shell functions that slash commands depend on

The slash commands (`.claude/commands/ant/*.md`) call 82 shell subcommands. Of those, **53 are already in Go**. These **28 are NOT in Go yet** and must be implemented:

```
colony-archive-xml      — XML export of colony data (called by seal)
context-capsule         — Build worker context capsule (called by build)
context-update          — Update colony-prime context (called by build)
data-safety-stats       — Data integrity stats (called by status)
eternal-init            — Initialize eternal memory (called by init)
generate-ant-name       — Random ant name generation (called by build)
generate-commit-message — Generate commit message (called by build/continue)
generate-progress-bar   — Terminal progress bar (called by status/build)
learning-approve-proposals — Approve learning proposals (called by continue)
midden-write            — Write failure record (called by build/continue)
milestone-detect        — Detect current milestone (called by build)
pheromone-display       — Display pheromone signals (called by status)
pheromone-export-xml    — Export pheromones to XML (called by export-signals)
pheromone-import-xml    — Import pheromones from XML (called by import-signals)
print-next-up           — Display next steps (called by continue/build)
registry-export-xml     — Export registry to XML (called by export)
registry-import-xml     — Import registry from XML (called by import)
resume-dashboard        — Session resume info (called by resume)
session-clear           — Clear session state (called by init/resume)
session-init            — Initialize session (called by init)
session-mark-resumed    — Mark session as resumed (called by resume)
session-read            — Read session data (called by resume)
session-update          — Update session data (called by various)
session-verify-fresh    — Verify session is fresh (called by various)
update-progress         — Update phase progress (called by build/continue)
version-check-cached    — Cached version check (called by status)
wisdom-export-xml       — Export wisdom to XML (called by export)
wisdom-import-xml       — Import wisdom from XML (called by import)
```

### The other 134 "missing" subcommands
The remaining 134 shell subcommands not in Go are:
- **Internal helpers** called by other shell functions (not directly by slash commands)
- **Display formatters** (generate-threshold-bar, rolling-summary, etc.)
- **Dead code** referenced by no active command
- **Curation ant internals** (curation-run, curation-archivist, etc.) — 9 commands for the structural learning stack

These can be handled after the critical 28 are done. Many may not need Go equivalents at all if they're refactored into the packages that use them.

---

## PART 2: GO IMPLEMENTATION QUALITY AUDIT

### What's verified (from milestone audit .planning/v5.4-MILESTONE-AUDIT.md)
- **37/37 requirements satisfied** across phases 45-56
- **9/12 phases have VERIFICATION.md** (3 gap closure phases verified by sibling phases)
- **All Go tests pass** including with `-race` flag
- **Go packages implemented:** agent, agent/curation, colony, events, exchange, graph, llm, memory, storage

### Known issues from Oracle research (.aether/oracle/):
1. ~~**`pkg/events/events.go` is a near-empty stub**~~ — **CORRECTED:** events.go has 87.2% test coverage with real event bus implementation. The milestone audit was wrong.
2. **CI has NO Go steps** — `.github/workflows/ci.yml` only runs Node.js lint/test. Go compile errors silently pass CI.
3. **Go binary is NOT in npm distribution** — `package.json` files whitelist excludes `cmd/` and `pkg/`. The Go binary is local-only.
4. ~~**No goreleaser or build pipeline**~~ — **CORRECTED:** `.goreleaser.yml` IS configured. Makefile exists.
5. **Worktree merge-back gap** — agents spawn worktrees but never merge back. 3 orphaned worktree branches exist right now.
6. **4 placeholder items** in Go code:
   - `cmd/pheromone_write.go:185` — XML validation placeholder
   - `cmd/state_cmds.go:108` — State lock release placeholder
   - `cmd/midden_cmds.go:239` — Branch failure collection placeholder
   - `pkg/agent/curation/scribe.go:38` — Scribe agent is a stub

### Action items:
- [ ] Audit `pkg/events/` — is it really just a stub or does it have real implementation elsewhere?
- [ ] Add Go build + test to CI pipeline
- [ ] Set up goreleaser or equivalent for binary distribution
- [ ] Decide: does Go replace shell entirely, or do they coexist during transition?

---

## PART 3: WORKTREE CLEANUP (DO THIS FIRST)

There are **3 open worktrees** that need to be dealt with before anything else:

```
worktree-agent-a17bd9aa  — Phase 40-01 (pheromone snapshot injection into worktree.sh)
                           1 commit ahead of main — shell plumbing, probably not needed for Go
                           Has a SUMMARY.md — plan is "complete"

worktree-agent-a987c4ee  — Just a merge commit + badge update
                           2 commits ahead — nothing useful, can be deleted

worktree-agent-ad3d1704  — Phase 58 planning docs (YAML command generator)
                           9 commits ahead — planning artifacts only, may have useful docs
```

**Steps:**
1. Review `worktree-agent-ad3d1704` — merge the planning docs if useful, then remove
2. Remove `worktree-agent-a987c4ee` — nothing to save
3. Decide on `worktree-agent-a17bd9aa` — Phase 40 is shell plumbing. If we're going all-in on Go, this work is irrelevant. But check if the test file has value.
4. Clean up any other stale branches: `worktree-agent-a75c2f60`, `worktree-agent-a7cdb58a` (already merged to main, can delete)
5. Run `git worktree prune` and `git branch -d` for cleaned-up branches

**Also check for orphaned feature branches:**
- `feature/v2-living-hive`
- `gsd/49-agent-system-llm`
- `gsd/phase-47-memory-pipeline`

---

## PART 4: THE CONVERSION VERIFICATION PLAN

### Phase A: Implement the 28 critical missing commands

These are the shell functions that slash commands actually call. Implement each one in Go, in the appropriate `pkg/` package:

**Group 1 — Session management (implement in `pkg/colony/` or new `pkg/session/`):**
- session-init, session-read, session-update, session-clear, session-mark-resumed, session-verify-fresh, resume-dashboard

**Group 2 — Pheromone display/export (implement in existing `pkg/colony/`):**
- pheromone-display, pheromone-export-xml, pheromone-import-xml

**Group 3 — Context assembly (implement in `pkg/agent/` or new `pkg/context/`):**
- context-capsule, context-update

**Group 4 — XML exchange (implement in existing `pkg/exchange/`):**
- wisdom-export-xml, wisdom-import-xml, registry-export-xml, registry-import-xml, colony-archive-xml

**Group 5 — Build utilities (implement in `cmd/` or `pkg/colony/`):**
- generate-ant-name, generate-commit-message, generate-progress-bar, print-next-up, update-progress, data-safety-stats, milestone-detect

**Group 6 — Learning/memory (implement in `pkg/memory/`):**
- learning-approve-proposals, midden-write

**Group 7 — Init/setup:**
- eternal-init

**Group 8 — Version:**
- version-check-cached

For each command:
1. Read the shell implementation in `.aether/utils/*.sh` or `.aether/aether-utils.sh`
2. Implement equivalent Go function in the appropriate package
3. Wire it to a cobra command in `cmd/`
4. Write a test
5. Verify the output matches the shell version

### Phase B: Slash command migration

Update `.claude/commands/ant/*.md` to call the Go binary instead of shell scripts.

**Current pattern in slash commands:**
```bash
bash .aether/aether-utils.sh session-init --colony "my-colony"
```

**New pattern:**
```bash
aether session-init --colony "my-colony"
```

**Critical slash commands to update (ordered by usage frequency):**
1. `status.md` — calls version-check-cached, data-safety-stats, pheromone-display
2. `init.md` — calls session-init, eternal-init, context-capsule
3. `build.md` — calls generate-ant-name, context-update, milestone-detect, generate-progress-bar, update-progress, midden-write, print-next-up
4. `continue.md` — calls learning-approve-proposals, midden-write, update-progress, print-next-up
5. `seal.md` — calls colony-archive-xml, wisdom-export-xml, pheromone-export-xml
6. `resume.md` — calls session-read, session-mark-resumed, session-update, resume-dashboard
7. `export-signals.md` — calls pheromone-export-xml
8. `import-signals.md` — calls pheromone-import-xml
9. `export.md` — calls wisdom-export-xml, registry-export-xml
10. `import.md` — calls wisdom-import-xml, registry-import-xml

### Phase C: Integration verification

After implementing the 28 commands and updating slash commands:

1. **Smoke test every slash command:**
   - `/ant:init "test colony"` — should work via Go binary
   - `/ant:status` — should display colony state
   - `/ant:focus "test area"` — should create signal
   - `/ant:build 1` — should execute phase
   - `/ant:continue` — should verify and advance
   - `/ant:seal` — should archive colony
   - `/ant:pheromones` — should display signals
   - `/ant:export-signals` — should export XML
   - `/ant:import-signals` — should import XML

2. **Compare outputs:** Run each command with both shell and Go, diff the output

3. **Test the npm package:** After updating, run `npm pack --dry-run` and verify the Go binary (or build instructions) are included

4. **Run the full test suite:** Both `npm test` (shell tests) and `go test ./...` (Go tests)

### Phase D: Distribution setup

1. **Decide distribution model:**
   - Option A: Pre-compile Go binary for mac/linux/windows, include in npm package
   - Option B: Ship Go source, compile on install via postinstall
   - Option C: Use existing goreleaser.yml → GitHub Releases, Homebrew (ALREADY CONFIGURED)
   - Option D: `go install github.com/aether-colony/aether/cmd/aether@latest`

2. **Update `package.json`:**
   - Add `cmd/` and `pkg/` to files whitelist (if shipping source)
   - Or add compiled binary to files whitelist (if shipping binary)
   - Update `bin/` to point to Go binary
   - Add Go build step to `prepublishOnly`

3. **Set up goreleaser** (`.goreleaser.yml` exists — check if it's configured)

4. **Add Go to CI** (`.github/workflows/ci.yml`):
   ```yaml
   - uses: actions/setup-go@v5
     with:
       go-version: '1.26'
   - run: go build ./cmd/aether
   - run: go test ./...
   - run: go test -race ./pkg/...
   ```

---

## PART 5: CLEANUP PLAN

After Go is verified working:

### 5A. GSD Planning Cleanup
The `.planning/` directory has 59 phases of planning history. This is development tooling, not Aether itself.

**Keep:**
- `.planning/ROADMAP.md` — update to reflect Go conversion complete
- `.planning/STATE.md` — reset to clean state
- `.planning/REQUIREMENTS.md` — keep as reference
- `.planning/PROJECT.md` — keep as reference

**Archive or delete:**
- `.planning/phases/` — all 59 phase directories. Archive to a git branch or delete.
- `.planning/v5.4-MILESTONE-AUDIT.md` — archive after conversion verified
- `.planning/config.json` — reset

### 5B. Aether Data Cleanup
- `.aether/data/` — colony state files, local only, can be cleared
- `.aether/oracle/` — research files, archive or delete (most is now outdated)
- `.aether/dreams/` — session notes, archive or delete
- `.aether/chambers/` — archived colonies, keep if desired
- `.aether/survey/` — territory survey results, can delete

### 5C. Test Infrastructure
- `tests/` — shell tests. After Go takes over, these become legacy. Keep them running during transition, archive after.
- `tests/bash/` — structural learning stack tests. May have Go equivalents already.

### 5D. Documentation Updates
- `CLAUDE.md` — update to reflect Go as primary runtime
- `.aether/docs/` — review and update for Go
- `RUNTIME UPDATE ARCHITECTURE.md` — may need rewrite for Go
- `README.md` — add Go installation instructions

### 5E. Shell Script Retirement
- `.aether/aether-utils.sh` — keep during transition, eventually deprecate
- `.aether/utils/*.sh` — keep during transition, eventually remove
- Slash commands that call shell — update to call Go (Phase B above)

---

## PART 6: EXECUTION ORDER

### Step 1: Clean up worktrees (15 min)
Merge or discard the 3 open worktrees. Delete stale branches. This unblocks everything else.

### Step 2: Fix known Go issues (30 min)
- Verify `pkg/events/` has real implementation or note the gap
- Remove any dead code / unused imports

### Step 3: Implement the 28 critical commands (the main work)
Start with the groups that unblock the most slash commands:
1. Session management (7 commands) — unblocks init, resume, status
2. Context assembly (2 commands) — unblocks build
3. Build utilities (7 commands) — unblocks build, continue, status
4. XML exchange (5 commands) — unblocks seal, export, import
5. Pheromone display/export (3 commands) — unblocks status, pheromones
6. Learning/memory (2 commands) — unblocks continue
7. Init/setup (1 command) — unblocks init
8. Version (1 command) — unblocks status

### Step 4: Update slash commands to call Go
Change `bash .aether/aether-utils.sh <cmd>` to `aether <cmd>` in all affected command files.

### Step 5: Integration testing
Smoke test every slash command. Compare outputs. Run full test suites.

### Step 6: Distribution setup
Set up goreleaser, update CI, update npm package.

### Step 7: Cleanup
Archive planning phases, clean up data files, update documentation.

---

## KEY FILES TO READ

These files contain important context for the implementation:

- `.planning/v5.4-MILESTONE-AUDIT.md` — Full audit of Go conversion status
- `.aether/oracle/synthesis.md` — Oracle research findings
- `.aether/oracle/gaps.md` — Known gaps and open questions
- `.aether/oracle/PLAN.md` — Oracle's recovery plan (post-worktree-merge)
- `.planning/ROADMAP.md` — All phases and their status
- `.planning/REQUIREMENTS.md` — All 37 v5.4 requirements
- `go.mod` — Go dependencies
- `.goreleaser.yml` — Go release configuration (if configured)
- `.github/workflows/ci.yml` — CI pipeline (needs Go steps)
- `package.json` — npm package config (needs Go binary inclusion)

---

## RISKS AND OPEN QUESTIONS

1. **Should Go completely replace shell, or coexist?** If coexist, what's the boundary?
2. **pkg/events/ stub** — Phase 46 says it passed but the file is 3 lines. Is there real implementation?
3. **npm distribution model** — Pre-compiled binary vs source vs separate channel?
4. **Cross-platform testing** — Go binary needs testing on Linux (CI is macOS only currently?)
5. **The 134 "non-critical" missing commands** — some may be needed by internal shell functions that slash commands call indirectly. Need trace-through testing.
6. **GSD system itself** — the `.claude/get-shit-done/` directory is how we plan and execute. It calls `gsd-tools.cjs` (Node). This stays as-is — it's development tooling, not Aether itself.

---

## WHAT SUCCESS LOOKS LIKE

- `aether init "test"` creates a colony using the Go binary
- `aether status` shows colony state
- `aether pheromone-write --type focus --content "test"` creates a signal
- `aether build 1` (or equivalent) executes a phase
- All slash commands (`/ant:*`) work by calling the Go binary
- `npm install -g .` installs a working Go binary
- `go test ./...` passes
- CI runs Go build + test
- The shell scripts still work during transition but are marked deprecated
- `.planning/` is cleaned up
- Documentation reflects Go as the primary runtime
