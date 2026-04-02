# Research Synthesis

## Topic
Post-worktree-merge recovery: fix Go compile errors, close worktree merge-back gap, verify npm publish flow, and chart Go transition roadmap.

## Findings by Question

### Q1: Compile Errors (ANSWERED, 95% confidence)

**Single compile error identified:** `cmd/flags.go:5:2` imports `"strings"` but never uses it. This is the only build failure in the entire Go codebase.

**Evidence:**
- `go build ./cmd/aether` fails with exactly one error: `cmd/flags.go:5:2: "strings" imported and not used` [S1]
- `cmd/history.go` compiles clean — uses `strings.SplitN()` in `parseEvent()` (line 70) and `strings.Contains()` in filter logic (line 43) [S2]
- `cmd/phase.go` compiles clean — uses `strings.Builder` in `renderPhaseDetails()` (line 55) [S3]
- All 8 pkg/ tests pass including with `-race` flag: agent, agent/curation, colony, events, graph, llm, memory, storage [S1]
- cmd/ has 13 Go files totaling 1,713 lines — only flags.go has the error [S1,S2,S3]
- The unused import likely came from the worktree merge — the `filterFlags()` function doesn't use strings functions

**Fix required:** Remove line 5 (`"strings"`) from `cmd/flags.go`. No other changes needed. After fix, `go build ./cmd/aether`, `go test ./...`, and `go test -race ./...` should all pass.

### Q2: Worktree Merge-Back Gap (PARTIAL, 35% confidence)

**Gap confirmed: worktree lifecycle has create and cleanup but no merge-back.**

**Current worktree infrastructure:**
- `_worktree_create()` in `worktree.sh` [S4]: Creates git worktree, copies `.aether/data/` and `.aether/exchange/`, injects main's pheromone signals via `pheromone-snapshot-inject`
- `_worktree_cleanup()` in `worktree.sh` [S4]: Removes worktree, deletes branch with `git branch -D` — NO merge step, changes are lost unless manually preserved
- No `_worktree_merge()` function exists anywhere

**Dead code paths in continue-advance.md:**
- Step 2.0.5 (Pheromone Merge-Back) checks for `pheromone-branch-export.json` [S9]
- Step 2.0.6 (Midden Collection) references `$last_merged_branch` and `$last_merge_sha` [S9]
- Neither variable is ever set by any prior step — both are dead code paths
- Step 2.0.7 (Cross-PR Midden Analysis) depends on Step 2.0.6 output [S9]

**Build pipeline gap:**
- build-wave.md spawns workers that may use worktrees [S5]
- build-verify.md Step 5.9 references `$last_merged_branch` but never populates it [S6]
- build-complete.md synthesis has no merge step [S7]
- The 5-stage build pipeline (prep→context→wave→verify→complete) has no merge-back anywhere [S11]

**Recommended placement: continue-advance.md Step 2.0.4**
- After "Update State" (Step 2) but before "Pheromone Merge-Back" (Step 2.0.5) [S9]
- Rationale: merge must happen after state update but before pheromone/midden collection that depends on merged code
- This would activate the existing Step 2.0.5 and 2.0.6 dead code paths

**Safety checks needed:**
1. Check for uncommitted changes in worktree (abort if dirty)
2. Verify build passes in worktree before merge (`go build ./cmd/aether`)
3. Check for merge conflicts before attempting merge
4. Validate tests pass after merge on main (`go test ./...`)
5. Create flag on unresolvable conflicts
6. For `.aether/data/` conflicts, always prefer main (colony state is authoritative on main) [S8]

**Orphaned branches evidence:**
- `feature/v2-living-hive`, `gsd/49-agent-system-llm`, `gsd/phase-47-memory-pipeline` exist but were never merged [S4]
- This is the exact gap the memory warns about — valuable work trapped in branches

**Still unknown:**
- Exact git merge commands for the step
- Detailed conflict resolution strategy per file type
- Rollback procedure on merge failure
- Whether build-wave.md or continue-advance.md should trigger merge
- Integration with `pheromone-export-branch` mechanism

### Q3: Publish Flow (PARTIAL, 40% confidence)

**npm package is Node.js + Bash only — Go is NOT part of npm distribution.**

**npm files whitelist excludes Go entirely:**
- package.json `files` array includes: `bin/`, `.claude/commands/ant/`, `.claude/agents/ant/`, `.opencode/commands/ant/`, `.opencode/agents/`, `.opencode/opencode.json`, `.aether/`, `README.md`, `LICENSE`, `DISCLAIMER.md`, `CHANGELOG.md` [S13]
- Go source directories (`cmd/`, `pkg/`) and `go.mod` are NOT in the whitelist — they will NOT be in the npm tarball [S13]

**No Go compilation in publish pipeline:**
- `prepublishOnly` runs `bash bin/validate-package.sh` [S13]
- validate-package.sh checks: required .aether/ files exist, private directories excluded, content-aware checks (no QUEEN.md, no temp files, no data/, no exchange XML data, exchange modules present) [S16]
- validate-package.sh has ZERO Go awareness — no `go build`, no `go test`, no `go.mod` check [S16]

**Go binary is local-only:**
- .gitignore excludes compiled binary (`/aether`) and test binaries (`*.test`) [S19]
- go.mod is tracked in git but won't appear in npm package [S17]
- npm bin entries are `bin/cli.js` (Node.js) and `bin/npx-entry.js` (Node.js) — no Go binary reference [S13]
- Postinstall is `node bin/cli.js install --quiet` — Node.js only [S13, S18]

**Dual .npmignore system:**
- Root `.npmignore` excludes `.planning/`, `.ralph/`, `.cache/`, `.git/`, dev files [S14]
- `.aether/.npmignore` excludes private dirs (data/, dreams/, oracle/, checkpoints/, locks/, temp/, archive/, chambers/) and private files (QUEEN.md, CONTEXT.md, CROWNED-ANTHILL.md, exchange/*.xml) [S15]
- Neither has Go rules because Go files live outside `.aether/` subtree and aren't in `files` whitelist anyway [S14, S15]

**Still unknown:**
- Should Go binary be distributed? (intentional local-only vs oversight)
- Is separate Go distribution planned (e.g., goreleaser, Homebrew)?
- CI/CD configuration for Go builds (if any)
- go.sum integrity in published package (not applicable if not publishing Go)

### Q4: Push/Publish Sequence (PARTIAL, 72% confidence)

**No npm publish needed for current Go changes. Go is local-only and not distributed via npm.**

**Branch strategy:**
- All current Go work (Phases 45-50) committed directly to main — no PR workflow in practice [S24, S25]
- REDIRECT signal requires PR-based workflow, creating a process gap
- Three orphaned feature branches exist but no active worktrees [S21]
- Uncommitted changes are minimal: 3 unused import deletions (cmd/flags.go, cmd/history.go, cmd/phase.go) plus oracle state files

**go mod tidy: NOT needed** — `go mod tidy -diff` produces no output. go.mod/go.sum are clean. Dependencies: cobra v1.10.2, go-pretty v6.7.8, anthropic-sdk-go v1.29.0, testify v1.11.1, gjson v1.18.0 [S17].

**npm version: should NOT bump** — Current is 5.3.2. Phase 50 (CLI Commands) is 1/6 plans. Phase 51 (Distribution) hasn't started. Version bump should wait until Phase 51 ships the Go binary with `go install` distribution [S13, S24, S25].

**CI pipeline has NO Go steps:**
- ci.yml runs: checkout → setup Node 20 → npm ci → lint → test → runtime/ sync check → npm audit [S20]
- No `setup-go`, no `go build`, no `go test` — Go compile errors silently pass CI [S20]
- Must be fixed before Phase 51 ships Go binary distribution
- Current recommendation: add Go steps to CI as a separate job in Phase 50 or 51

**Registry analysis (46 repos):**
- 10 repos at 5.3.2 (latest): Aether, M4L-AnalogWave, Prompt App, CornettoDB, Aether Website, Capture Vault, Colony Creation Station, API Bus Compressor, AetherVault, Formica, openclaude, AetherFormica_Business [S21]
- 3 repos at 5.2.1: SonicDoc, STS Workspace, .claude config [S21]
- ~32 repos at older versions (1.1.x to 5.0.0) [S21]
- 14 repos have `active_colony: true` [S21]
- `aether update` in other repos pulls shell/command changes only — Go code not distributed via npm

**Other workflows (all Node.js only):**
- deploy-pages.yml: deploys site/ to GitHub Pages on push to main (path filter: site/**) [S22]
- correlation-pipeline.yml: weekly npm download spike detection, triggers on `release:published` or weekly cron [S23]
- badges.yml: badge updates

**Recommended push sequence for current state:**
1. Fix compile error (remove unused strings import from cmd/flags.go)
2. Verify locally: `go build ./cmd/aether && go test ./... && go test -race ./pkg/...`
3. Verify npm: `npm test && npm run lint`
4. Commit to main (or PR per REDIRECT signal preference)
5. NO npm publish needed — Go changes don't affect npm package
6. NO aether update needed in other repos — only shell/command files trigger updates

### Q5: Other Repos Affected (OPEN, 0%)
(Not yet researched)

### Q6: Go Transition Roadmap (OPEN, 0%)
(Not yet researched)

## Sources
- [S1] `cmd/flags.go` — flags command source (accessed 2026-04-02)
- [S2] `cmd/history.go` — history command source (accessed 2026-04-02)
- [S3] `cmd/phase.go` — phase command source (accessed 2026-04-02)
- [S4] `.aether/utils/worktree.sh` — worktree-create and worktree-cleanup functions (accessed 2026-04-02)
- [S5] `.aether/docs/command-playbooks/build-wave.md` — worker spawning playbook (accessed 2026-04-02)
- [S6] `.aether/docs/command-playbooks/build-verify.md` — verification playbook (accessed 2026-04-02)
- [S7] `.aether/docs/command-playbooks/build-complete.md` — synthesis and results (accessed 2026-04-02)
- [S8] `.aether/docs/command-playbooks/continue-verify.md` — verification loop and claims verification (accessed 2026-04-02)
- [S9] `.aether/docs/command-playbooks/continue-advance.md` — state update, pheromone/midden merge-back (accessed 2026-04-02)
- [S10] `.aether/docs/command-playbooks/continue-finalize.md` — wisdom summary, handoff, changelog (accessed 2026-04-02)
- [S11] `.claude/commands/ant/build.md` — build orchestrator (accessed 2026-04-02)
- [S12] `.claude/commands/ant/continue.md` — continue orchestrator (accessed 2026-04-02)
- [S13] `package.json` — npm package config with files whitelist, scripts, bin entries (accessed 2026-04-02)
- [S14] `.npmignore` — root-level npm publish exclusions (accessed 2026-04-02)
- [S15] `.aether/.npmignore` — subdirectory npm exclusions for private dirs and files (accessed 2026-04-02)
- [S16] `bin/validate-package.sh` — pre-publish validation checking required .aether/ files (accessed 2026-04-02)
- [S17] `go.mod` — Go module definition, requires cobra, pretty, yaml, anthropic-sdk (accessed 2026-04-02)
- [S18] `bin/cli.js` — Node.js CLI entry point with install, update, checkpoint commands (accessed 2026-04-02)
- [S19] `.gitignore` — excludes /aether binary, *.test, build/dist, .planning/ (accessed 2026-04-02)
- [S20] `.github/workflows/ci.yml` — Node.js-only CI pipeline (lint, test, runtime sync, npm audit) (accessed 2026-04-02)
- [S21] `~/.aether/registry.json` — 46 repos registered, versions 1.1.0 to 5.3.2, 14 active colonies (accessed 2026-04-02)
- [S22] `.github/workflows/deploy-pages.yml` — GitHub Pages deployment on push to main (site/ path filter) (accessed 2026-04-02)
- [S23] `.github/workflows/correlation-pipeline.yml` — weekly npm download spike detection and release correlation (accessed 2026-04-02)
- [S24] `.planning/STATE.md` — v5.4 Shell-to-Go Rewrite, Phase 50 executing (1/6 plans), 14/20 phases complete (accessed 2026-04-02)
- [S25] `.planning/ROADMAP.md` — 51 phases total, Phase 51 (XML+Dist+Testing) handles Go binary distribution (accessed 2026-04-02)

## Last Updated
Iteration 4 -- 2026-04-02T10:45:00Z
