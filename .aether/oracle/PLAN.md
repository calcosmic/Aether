# Oracle Research Plan: Post-Worktree-Merge Recovery

**Generated:** 2026-04-02
**Confidence:** 95%+ across all 6 questions
**Sources:** 28 codebase files

---

## Section 1: Immediate Fixes — Compile Errors

### Problem
After merging 13 worktree branches into main, `go build ./cmd/aether` fails with exactly **one** compile error.

### Root Cause
`cmd/flags.go:5` imports `"strings"` but never uses it. The file uses `fmt`, `colony`, `table`, and `cobra` — but no strings functions anywhere in its 93 lines. The `filterFlags()` function compares with `!=` and boolean checks, not strings operations.

### Fix

**File: `cmd/flags.go` line 5**

Remove the unused import:

```go
// BEFORE (lines 3-9):
import (
    "fmt"
    "strings"

    "github.com/aether-colony/aether/pkg/colony"
    "github.com/jedib0t/go-pretty/v6/table"
    "github.com/spf13/cobra"
)

// AFTER:
import (
    "fmt"

    "github.com/aether-colony/aether/pkg/colony"
    "github.com/jedib0t/go-pretty/v6/table"
    "github.com/spf13/cobra"
)
```

### Verification Commands

```bash
# 1. Build the binary
go build ./cmd/aether

# 2. Run all tests
go test ./...

# 3. Run race tests on packages
go test -race ./pkg/...

# 4. Verify npm side is unaffected
npm test && npm run lint
```

### Other Files Confirmed Clean
- `cmd/history.go` — compiles clean, uses `strings.SplitN()` (line 70) and `strings.Contains()` (line 43)
- `cmd/phase.go` — compiles clean, uses `strings.Builder` (line 55)
- All 13 cmd/ files (1,713 lines) — only flags.go has the error
- All 8 pkg/ test suites pass with `-race` flag

---

## Section 2: Safety Fix — Worktree Merge-Back Gap

### Problem
The colony build system spawns agents into git worktrees but has **no step** to merge those branches back to the target branch after work completes. This caused 13 orphaned branches with valuable code — the exact scenario the memory system warns about.

### Current Infrastructure (Confirmed Dead Ends)

| Component | What It Does | What's Missing |
|-----------|-------------|----------------|
| `_worktree_create()` in worktree.sh | Creates git worktree, copies .aether/data/ and .aether/exchange/, injects pheromone signals | Nothing — works correctly |
| `_worktree_cleanup()` in worktree.sh | Removes worktree, deletes branch with `git branch -D` | **No merge step — deletes without merging** |
| continue-advance.md Step 2.0.5 | Checks for `pheromone-branch-export.json` | **Dead code — nothing creates this file** |
| continue-advance.md Step 2.0.6 | References `$last_merged_branch` and `$last_merge_sha` | **Dead code — no prior step sets these vars** |
| build-verify.md Step 5.9 | References `$last_merged_branch` | **Dead code — never populated** |

### Recommended Fix: Two-Part Implementation

#### Part A: Add `_worktree_merge()` to `worktree.sh`

Add a new function after `_worktree_cleanup()` at line 189 of `.aether/utils/worktree.sh`:

```bash
# _worktree_merge
# Merges a worktree branch back to the target branch with safety checks.
#
# Usage: _worktree_merge --branch <branch-name> [--target <target-branch>] [--force]
# Returns JSON: {ok:true, result:{merged, branch, target, sha, commits_merged}}
_worktree_merge() {
    local branch=""
    local target=""
    local force=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --branch) branch="${2:-}"; shift 2 ;;
            --target) target="${2:-}"; shift 2 ;;
            --force)  force=true; shift ;;
            *) shift ;;
        esac
    done

    if [[ -z "$branch" ]]; then
        json_err "$E_VALIDATION_FAILED" "Usage: worktree-merge --branch <branch-name> [--target <target-branch>]"
    fi

    # Sanitize branch name
    if [[ "$branch" == *..* ]] || [[ "$branch" == */* ]] || [[ "$branch" == *\\* ]]; then
        json_err "$E_VALIDATION_FAILED" "Branch name must not contain '..', '/', or backslashes"
    fi

    local worktree_dir="$WORKTREE_BASE_DIR/$branch"

    # Default target to current branch
    if [[ -z "$target" ]]; then
        target=$(git -C "$AETHER_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")
    fi

    # Safety check 1: worktree must exist
    if [[ ! -d "$worktree_dir" ]]; then
        json_err "$E_RESOURCE_NOT_FOUND" "No worktree found for branch '$branch'"
    fi

    # Safety check 2: worktree must have committed changes (unless --force)
    if [[ "$force" == "false" ]]; then
        local dirty_count
        dirty_count=$(git -C "$worktree_dir" status --porcelain 2>/dev/null \
            | grep -v '\.aether/' \
            | wc -l | tr -d ' ') || dirty_count=0
        if [[ "$dirty_count" -gt 0 ]]; then
            json_err "$E_VALIDATION_FAILED" "Worktree '$branch' has $dirty_count uncommitted changes. Commit or stash before merging."
        fi
    fi

    # Safety check 3: worktree must have commits ahead of target
    local ahead_count
    ahead_count=$(git -C "$AETHER_ROOT" rev-list --count "${target}..${branch}" 2>/dev/null || echo "0")
    if [[ "$ahead_count" -eq 0 ]]; then
        # Nothing to merge — just clean up
        _worktree_cleanup --branch "$branch" --force
        return $?
    fi

    # Safety check 4: check for merge conflicts (dry run)
    local conflict_check
    conflict_check=$(git -C "$AETHER_ROOT" merge-tree $(git -C "$AETHER_ROOT" merge-base "$target" "$branch") "$target" "$branch" 2>/dev/null | grep -c "changed in both" || echo "0")
    if [[ "$conflict_check" -gt 0 ]] && [[ "$force" == "false" ]]; then
        json_err "$E_VALIDATION_FAILED" "Merge would produce $conflict_check conflict(s). Resolve manually or use --force."
    fi

    # Perform the merge
    local merge_sha
    if ! git -C "$AETHER_ROOT" merge "$branch" --no-edit --no-ff -m "merge: worktree branch $branch into $target" >/dev/null 2>&1; then
        # Merge failed — abort
        git -C "$AETHER_ROOT" merge --abort 2>/dev/null || true
        json_err "$E_GIT_ERROR" "Merge of '$branch' into '$target' failed. Aborted."
    fi

    merge_sha=$(git -C "$AETHER_ROOT" rev-parse HEAD 2>/dev/null || echo "unknown")

    # Export pheromones from the worktree branch before cleanup
    if [[ -f "$worktree_dir/.aether/data/pheromones.json" ]]; then
        mkdir -p "$AETHER_ROOT/.aether/exchange"
        cp "$worktree_dir/.aether/data/pheromones.json" \
           "$AETHER_ROOT/.aether/exchange/pheromone-branch-export.json" 2>/dev/null || true
    fi

    # Cleanup the worktree
    _worktree_cleanup --branch "$branch" --force 2>/dev/null || true

    local result
    result=$(jq -n \
        --arg branch "$branch" \
        --arg target "$target" \
        --arg sha "$merge_sha" \
        --argjson ahead "$ahead_count" \
        '{merged: true, branch: $branch, target: $target, sha: $sha, commits_merged: $ahead}')
    json_ok "$result"
}
```

#### Part B: Wire Merge-Back into Build Pipeline

**Location:** `continue-advance.md`, insert as **new Step 2.0.4** between Step 2 (Update State) and Step 2.0.5 (Pheromone Merge-Back).

```markdown
### Step 2.0.4: Worktree Merge-Back (NON-BLOCKING)

After state update, check for any completed worktree branches from the build wave and merge them back to the target branch.

Run using the Bash tool with description "Checking for worktree branches to merge...":
\```bash
# List worktree branches created during this build
branches=$(git -C "$AETHER_ROOT" worktree list --porcelain 2>/dev/null \
    | grep "worktree-agent-\|worktree-" \
    | awk '{print $NF}' || echo "")

last_merged_branch=""
last_merge_sha=""
merged_count=0

for branch in $branches; do
    [[ -z "$branch" ]] && continue
    result=$(bash "$AETHER_UTILS" worktree-merge --branch "$branch" 2>/dev/null || echo '{"ok":false}')
    ok=$(echo "$result" | jq -r '.ok // false')
    if [[ "$ok" == "true" ]]; then
        last_merged_branch="$branch"
        last_merge_sha=$(echo "$result" | jq -r '.result.sha // ""')
        merged_count=$((merged_count + 1))
    fi
done

if [[ "$merged_count" -gt 0 ]]; then
    echo "Merged $merged_count worktree branch(es). Last: $last_merged_branch ($last_merge_sha)"
fi
\```

This step sets `$last_merged_branch` and `$last_merge_sha` which activates the existing dead code paths in Steps 2.0.5 and 2.0.6.
```

### Why continue-advance.md (not build-complete.md)

1. **Timing:** Build-complete runs before verification. Merging before verification would merge potentially broken code.
2. **Continue is the right gate:** Continue runs after verification passes — we know the code works before merging.
3. **Activates dead code:** Steps 2.0.5-2.0.7 already handle pheromone/midden merge-back — they just need `$last_merged_branch` to be set.
4. **Rollback safety:** If merge fails, `git merge --abort` is called. The worktree branch is preserved.

### Orphaned Feature Branches (Separate Issue)

Three non-worktree feature branches exist with valuable work:
- `feature/v2-living-hive`
- `gsd/49-agent-system-llm`
- `gsd/phase-47-memory-pipeline`

These should be evaluated separately — they may contain work already superseded by the main branch merge.

---

## Section 3: Publish Checklist

### Why npm Publish IS Required

The worktree merge-back fix changes files that **are** distributed via npm:

- `.aether/utils/worktree.sh` — the new `_worktree_merge()` function
- `.aether/docs/command-playbooks/continue-advance.md` — the new Step 2.0.4

Both propagate through the hub:
1. `npm install -g .` updates `~/.aether/system/` (the hub)
2. Repos run `aether update` pulls from hub into their local `.aether/`

**If we don't publish, every repo with an active colony keeps running the broken pipeline** — agents spawn worktrees that never merge back, creating orphaned branches everywhere.

The Go compile fix (`cmd/flags.go`) is NOT in the npm package (Go source excluded from `files` whitelist), but the shell/playbook fix IS. Publishing distributes the merge-back safety fix to all repos.

### Current State Assessment

| Item | Status | Action Needed |
|------|--------|---------------|
| Go binary compilation | Fails (1 unused import) | Fix cmd/flags.go |
| Go tests | All 8 pkg/ suites pass | None |
| npm tests | Should pass (Go not in npm) | Verify after fix |
| go mod tidy | Clean | None |
| npm version (5.3.2) | **Needs patch bump** | Required — shell/playbook changes distribute |
| CI pipeline | Node.js only, no Go steps | Add Go CI job (Phase 50/51) |

### Step-by-Step Checklist

```bash
# === 1. FIX COMPILE ERROR ===
# Edit cmd/flags.go: remove line 5 ("strings" import)

# === 2. ADD WORKTREE MERGE-BACK ===
# Add _worktree_merge() to .aether/utils/worktree.sh
# Add Step 2.0.4 to .aether/docs/command-playbooks/continue-advance.md

# === 3. VERIFY GO SIDE ===
go build ./cmd/aether          # Must succeed
go test ./...                   # All tests pass
go test -race ./pkg/...         # Race tests pass

# === 4. VERIFY NPM SIDE ===
npm test                        # Shell tests pass
npm run lint                    # Linting passes

# === 5. COMMIT ===
git add cmd/flags.go .aether/utils/worktree.sh .aether/docs/command-playbooks/continue-advance.md
git commit -m "fix: remove unused strings import + add worktree merge-back to prevent orphaned branches"

# === 6. PUSH ===
git push origin main

# === 7. PUBLISH (distributes shell fix to hub) ===
npm version patch               # 5.3.2 -> 5.3.3
npm install -g .                # Update hub at ~/.aether/system/
npm publish                     # Distribute to all repos

# === 8. UPDATE OTHER REPOS ===
# For each active repo: aether update
```

### What Gets Distributed vs What Stays Local

**Distributed via npm (reaches other repos via `aether update`):**
- `.aether/utils/worktree.sh` — the merge-back function
- `.aether/docs/command-playbooks/continue-advance.md` — the pipeline step
- All other `.aether/` shell scripts, templates, skills

**NOT distributed (local git only):**
- `cmd/flags.go` fix — Go source, not in npm `files` whitelist
- `cmd/`, `pkg/`, `go.mod` — all excluded from npm package
- Go binary — not in the package at all

### CI Gap (Technical Debt)

`.github/workflows/ci.yml` has zero Go steps. Go compile errors will **silently pass CI**. Add a Go job before Phase 51 ships the binary:

```yaml
  go-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go build ./cmd/aether
      - run: go test ./...
      - run: go test -race ./pkg/...
```

**Timing:** Add during Phase 50 or 51.

---

## Section 4: Repo Update Plan

### Registry Overview

| Category | Count | Version Range | Risk |
|----------|-------|---------------|------|
| Latest (5.3.2) | 15 | Current | Low — update after publish |
| Recent (5.0-5.2.1) | 5 | Behind | Medium — missing bugfixes |
| Ancient (1.1-4.0) | ~28 | Very behind | Low — likely inactive |

### Impact of Publishing v5.3.3

The worktree merge-back fix will protect all repos from orphaned worktree branches:

1. After we publish 5.3.3 to npm
2. Each repo runs `aether update` (pulls from `~/.aether/system/`)
3. They get the updated `worktree.sh` and `continue-advance.md`
4. Future builds in those repos will automatically merge worktree branches back

### Repos That Need Updating After Publish

| Repo | Version | Active Colony | Action |
|------|---------|--------------|--------|
| openclaude | 5.3.2 | Yes | `aether update` — gets merge-back fix |
| Formica | 5.3.2 | — | `aether update` — gets merge-back fix |
| Capture Vault | 5.3.2 | Yes | `aether update` — gets merge-back fix |
| 01_Plugin Dev. | 5.1.0 | Yes | **High priority** — 2 versions behind |
| STS - Workspace | 5.2.1 | Yes | `aether update` — gets fix + prior fixes |
| 05_Prompt_App | 5.3.2 | Yes | `aether update` — gets merge-back fix |
| API Bus Compressor | 5.3.2 | Yes | `aether update` — gets merge-back fix |
| STS (sonotherapie) | 5.3.2 | Yes | `aether update` — gets merge-back fix |

### No Manual Intervention Needed

`aether update` uses a two-phase commit with automatic rollback. It syncs:
- `.aether/` system files (shell scripts, templates, skills, docs)
- `.claude/commands/ant/` commands
- `.claude/agents/ant/` agents
- `.opencode/` equivalents
- `.claude/rules/` rules

No data migration, no state schema changes. Clean update.

### Repos Safe to Skip

~28 repos at ancient versions (1.1.x through 4.0.0) — likely inactive or abandoned. Running `aether update` on them would be safe but low priority.

---

## Section 5: Go Transition Next Steps

### Current Position

**Milestone v5.4: Shell-to-Go Rewrite**
- **Phases complete:** 14 of 20 (Phases 45-49 + prior milestones)
- **Phase 50 (CLI Commands):** 1 of 6 plans complete (50-01 done, 50-02 through 50-06 pending)
- **Phase 51 (XML+Dist+Testing):** Not started

**Go Codebase Stats:**
- 80 .go files, 14,166 lines
- 7 packages: agent (6 files), colony (17), events (3), graph (2), llm (9), memory (13), storage (6)
- 13 cmd files: root, version, completion, flags, status, phase, history, memory-metrics, colony-vital-signs, pheromone-read, pheromone-count + 2 more
- Binary: 5.9MB compiled
- All 8 pkg/ test suites pass with race detection

### Phase 50 Remaining Plans

| Plan | Focus | Value | Priority |
|------|-------|-------|----------|
| 50-02 | Status dashboard + read-only display commands | HIGH — most visible, exercises existing pkg/ packages | **Next** |
| 50-03 | Write/mutation commands (pheromones, flags, spawn, state, learning) | HIGH — enables colony operations via Go | Second |
| 50-04 | Swarm, hive, skills, midden, registry commands | Medium — advanced features | Third |
| 50-05 | Queen, immune, council, clash, autopilot | Medium — orchestration layer | Fourth |
| 50-06 | Remaining utilities + command_count_test >= 145 | Final — completeness gate | Last |

### Recommended Sequence

1. **Fix compile error** (5 min — this plan, Section 1)
2. **Add worktree merge-back** (Section 2 — prevents future orphaned branches)
3. **Publish v5.3.3** (Section 3 — distributes fix to hub)
4. **Execute Phase 50-02** — Status dashboard with progress bars, pheromone tables, memory health
5. **Execute Phase 50-03** — Mutation commands (pheromone-write, flag-create, state-mutate, learning capture)
6. **Execute Phase 50-04 through 50-06** — Complete remaining CLI commands
7. **Add Go CI job** to ci.yml — Before Phase 51 ships the binary
8. **Phase 51** — XML exchange, Go binary distribution (goreleaser, `go install`, cross-compile), full test parity

### High-Frequency Shell Subcommands to Migrate Next

Based on the 125+ shell subcommands, these are highest priority for Go migration:

| Subcommand | Frequency | Package Dependency | Plan |
|------------|-----------|-------------------|------|
| `state-mutate` | Every continue | storage | 50-03 |
| `pheromone-write` | Every build/continue | colony | 50-03 |
| `colony-prime` | Every build | colony + memory + llm | 50-05 |
| `memory-capture` | Every build/continue | memory | 50-03 |
| `status-display` | Manual checks | colony | 50-02 |
| `activity-log` | Every operation | events | 50-02 |
| `flag-create` | Manual + auto | colony | 50-03 |
| `instinct-create` | Every continue | memory | 50-03 |

### Known Discrepancies

1. **Phase 48 (Graph Layer):** ROADMAP shows `[ ]` (incomplete) but STATE.md marks it complete. Documentation sync issue — graph code exists and passes tests.
2. **Phase 49 (Agent System):** All 4 plans have summaries, confirming completion.
3. **REDIRECT signal vs practice:** REDIRECT says "all changes go through PRs" but all Go work has been direct-to-main. Process gap to address.

---

## Appendix: Source Files Referenced

| ID | File | Purpose |
|----|------|---------|
| S1 | `cmd/flags.go` | Unused strings import |
| S2 | `cmd/history.go` | Uses strings.SplitN, strings.Contains |
| S3 | `cmd/phase.go` | Uses strings.Builder |
| S4 | `.aether/utils/worktree.sh` | Worktree create/cleanup, no merge |
| S5 | `.aether/docs/command-playbooks/build-wave.md` | Worker spawning |
| S6 | `.aether/docs/command-playbooks/build-verify.md` | Verification, dead $last_merged_branch ref |
| S7 | `.aether/docs/command-playbooks/build-complete.md` | Synthesis, no merge step |
| S8 | `.aether/docs/command-playbooks/continue-verify.md` | Verification loop |
| S9 | `.aether/docs/command-playbooks/continue-advance.md` | Dead code Steps 2.0.5-2.0.7 |
| S10 | `.aether/docs/command-playbooks/continue-finalize.md` | Wisdom summary |
| S11 | `.claude/commands/ant/build.md` | Build orchestrator |
| S12 | `.claude/commands/ant/continue.md` | Continue orchestrator |
| S13 | `package.json` | Files whitelist, scripts |
| S14 | `.npmignore` | Root exclusions |
| S15 | `.aether/.npmignore` | Subdirectory exclusions |
| S16 | `bin/validate-package.sh` | Pre-publish validation |
| S17 | `go.mod` | Module definition |
| S18 | `bin/cli.js` | Node.js CLI entry |
| S19 | `.gitignore` | Go binary exclusion |
| S20 | `.github/workflows/ci.yml` | CI pipeline (Node.js only) |
| S21 | `~/.aether/registry.json` | 48 repos registered |
| S22 | `.github/workflows/deploy-pages.yml` | GitHub Pages |
| S23 | `.github/workflows/correlation-pipeline.yml` | Release correlation |
| S24 | `.planning/STATE.md` | Phase 50 at 1/6 plans |
| S25 | `.planning/ROADMAP.md` | 51 phases, Go transition |
| S26 | `bin/cli.js (updateRepo)` | Two-phase update mechanism |
| S27 | `cmd/root.go` | Cobra root command |
| S28 | `pkg/` | All Go packages |
