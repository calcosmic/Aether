# Phase 43: Clash Detection Integration - Context

**Gathered:** 2026-03-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire existing clash detection code into the active workflow: install the PreToolUse hook so Edit/Write operations are checked against other worktrees, wire `_worktree_create` to copy colony context, configure the merge driver for package-lock.json, and ensure `.aether/data/` is on the allowlist. All code already exists (~562 lines implementation + ~867 lines tests); this phase is integration wiring.

</domain>

<decisions>
## Implementation Decisions

### Hook Installation
- **D-01:** `_clash_setup --install` is called during `/ant:init` (after colony initialization). The hook persists across sessions via `.claude/settings.json`. Uninstall is manual via `clash-setup --uninstall`.
- **D-02:** The PreToolUse hook in `clash-pre-tool-use.js` is already written and tested. It exits 0 (allow) on any error (fail-open design). No behavior changes needed.

### Worktree Context Copy
- **D-03:** `_worktree_create` already copies `.aether/data/` and `.aether/exchange/` to the new worktree (worktree.sh lines 76-85). The Phase 40 D-01 wiring of `pheromone-snapshot-inject` is already in place. No new context copy code needed — verify it works end-to-end.
- **D-04:** Worktree creation already injects main HEAD pheromones via pheromone-snapshot-inject (worktree.sh lines 89-100). Verify this integration is tested.

### Merge Driver for Lockfiles
- **D-05:** `merge-driver-lockfile.sh` is already written (35 lines, keeps "ours"). Registration requires: `git config merge.lockfile.driver` + `.gitattributes` entry for `package-lock.json merge=lockfile`.
- **D-06:** Merge driver registration happens during `/ant:init` (alongside hook installation). Both are "setup infrastructure" steps that belong in init.

### Allowlist Scope
- **D-07:** Current allowlist: `.aether/data/` (branch-local colony state). This is sufficient — `.planning/` is gitignored and won't appear in worktrees. No additional paths needed.

### Claude's Discretion
- Exact order of setup steps in init (hook install vs merge driver registration)
- Whether to add a `--status` flag to `_clash_setup` for checking installation state
- Test coverage for the end-to-end init → worktree create → clash detect flow

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Clash Detection Implementation
- `.aether/utils/clash-detect.sh` — Core clash detection logic (239 lines), allowlist, worktree enumeration
- `.aether/utils/hooks/clash-pre-tool-use.js` — PreToolUse hook for Claude Code (99 lines), fail-open design
- `.aether/utils/worktree.sh` — Worktree create/cleanup with context copy (189 lines)
- `.aether/utils/merge-driver-lockfile.sh` — Lockfile merge driver, keeps "ours" (35 lines)

### Existing Tests
- `test/test-clash-detect.sh` — Core detection tests (173 lines)
- `test/test-clash-pre-tool-use.sh` — Hook behavior tests (157 lines)
- `test/test-clash-subcommands.sh` — Setup/install/uninstall tests (151 lines)
- `tests/bash/test-worktree-module.sh` — Worktree create/cleanup tests (386 lines)

### Prior Phase Context
- `.planning/phases/40-pheromone-propagation/40-CONTEXT.md` — Pheromone propagation decisions (D-01 wires snapshot-inject into worktree create)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `_clash_setup()`: Full install/uninstall logic for settings.json hook management — ready to wire
- `_clash_detect()`: Core detection with JSON output, allowlist check, worktree enumeration — complete
- `_worktree_create()`: Creates worktrees with context copy — already includes pheromone injection
- `merge-driver-lockfile.sh`: Keeps "ours" on package-lock.json conflicts — needs .gitattributes registration

### Established Patterns
- Hook installation pattern: `_clash_setup --install` writes to `.claude/settings.json` PreToolUse array (already implemented in clash-detect.sh lines 164-234)
- JSON output pattern: All functions return `{ok:true, result:{...}}` or `{ok:false, error:...}` via `json_ok`/`json_err`
- Fail-open pattern: Hook exits 0 on any error, allowing work to continue (clash-pre-tool-use.js line 90-93)

### Integration Points
- `.claude/commands/ant/init.md` — Where `_clash_setup --install` gets called during colony initialization
- `.claude/settings.json` — Where the PreToolUse hook is registered
- `.gitattributes` — Where the lockfile merge driver mapping is registered
- `.aether/aether-utils.sh` — Case statement dispatch for `clash-setup` and `clash-detect` subcommands

</code_context>

<specifics>
## Specific Ideas

No specific requirements — integration is straightforward wiring of existing, tested code.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 43-clash-detection-integration*
*Context gathered: 2026-03-31*
