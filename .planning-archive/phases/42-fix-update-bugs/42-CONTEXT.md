# Phase 42: Fix Update Bugs - Context

**Gathered:** 2026-02-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix bugs in the `/ant:update` command to make file operations atomic, fix counter reporting, and clean stale directories from target repos. The sync should be authoritative about what belongs in the target `.aether/` directory.

</domain>

<decisions>
## Implementation Decisions

### Sync Model
- **Dry-run first, then tick-to-approve** — Show what would change, user approves via tick-to-approve UI
- **Hub required** — Update only syncs from `~/.aether/` (hub), must be installed
- **Full mirror sync** — Target `.aether/` becomes authoritative mirror of source (except protected)

### Protected Files
- **Always preserve:**
  - `data/` — Colony state
  - `dreams/` — Session notes
  - `oracle/` — Research progress
  - `midden/` — Failure tracking
  - `QUEEN.md` — User's wisdom file (CRITICAL — never touch)
- **Clean everything else** — System dirs, docs, utils, templates get synced

### Trash Safety
- **Move to trash, don't delete** — Removed files go to `.aether/.trash/`
- **Manual cleanup** — Never auto-purge trash, user cleans when ready
- **One session per trash** — Trash folder timestamped for easy identification

### Approval UI (Tick-to-Approve)
- **Group by action type** — Show additions first, then removals, then updates
- **All pre-selected** — All changes ticked by default, user unticks to skip
- **File-by-file paths** — List each file with full path

### Conflict Handling
- **Ask per conflict** — If local file was modified from source, show diff and let user choose
- **Preserve user intent** — Don't blindly overwrite customized files

### Cleanup Behavior
- **Remove empty directories** — After sync, clean up any empty directories left behind
- **Entire .aether/ scope** — Sync covers all of `.aether/` except protected dirs

### Reporting
- **File-by-file after sync** — Show each file added, removed, updated with paths
- **Clear summary** — "Added X, Updated Y, Removed Z, Skipped N"

</decisions>

<specifics>
## Specific Ideas

- User has repo with "double documentation" — old files not cleaned up by previous updates
- The sync should be authoritative: "This is what's meant to be here, this isn't"
- QUEEN.md is sacred — contains user's accumulated wisdom, must never be touched

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 42-fix-update-bugs*
*Context gathered: 2026-02-22*
