# Classic Baseline Document

**Phase:** 107 (Classic Baseline Identification)
**Date:** 2026-05-12
**Selected Tag:** `v5.4.0`
**Classification Source:** [Runtime Boundary Contract](./contracts/runtime-boundary-contract.md) (Phase 106)

---

## Executive Summary

v5.4.0 is selected as the Classic baseline because it is the only version containing the Node-to-Go delegation bridge (`version-gate.js` + `binary-downloader.js`), has the complete 16-module set, and was the last production Classic release before the full Go transition. This document provides the behavioral evidence, version comparison, and per-module checklist that Phase 108 (Golden Workflow Tests) and Phase 109 (TypeScript Orchestration Host) will use as their reference anchor.

---

## Version Comparison

| Module | v5.3.0 | v5.3.3 | v5.4.0 | Classification |
|--------|--------|--------|--------|----------------|
| spawn-logger.js | identical | identical | identical | Restore in TS |
| state-guard.js | identical | identical | identical | Keep in Go |
| caste-colors.js | identical | identical | identical | Keep in Go |
| event-types.js | identical | identical | identical | Keep in Go |
| file-lock.js | identical | identical | identical | Keep in Go |
| state-sync.js | identical | identical | identical | Obsolete |
| banner.js | identical | identical | identical | Keep in Go |
| colors.js | identical | identical | identical | Keep in Go |
| logger.js | identical | identical | identical | Restore in TS |
| init.js | identical | identical | identical | Keep in Go |
| interactive-setup.js | identical | identical | identical | Obsolete |
| nestmate-loader.js | identical | identical | identical | Obsolete |
| errors.js | identical | identical | identical | Restore in TS |
| update-transaction.js | present | 15 lines different | 15 lines different | Keep in Go |
| binary-downloader.js | absent | absent | present (267 lines) | Keep in Go |
| version-gate.js | absent | absent | present (179 lines) | Keep in Go |

### What Changed Across Versions

**v5.3.0 and v5.3.3 are functionally identical.** Only 15 lines differ in `update-transaction.js` (exchange directory handling: v5.3.0 excludes the exchange directory from hub sync entirely; v5.3.3 removes exchange from the exclusion list but filters exchange data files by extension, allowing only `.sh` scripts to distribute). All 13 other shared modules are byte-identical.

**v5.4.0 adds the Node-to-Go bridge.** The two new modules (`binary-downloader.js` at 267 lines and `version-gate.js` at 179 lines) implement Go binary download with SHA-256 verification and version-gated delegation. Additionally, 134 lines change in `bin/cli.js` to add a delegation shim that checks `shouldDelegate()` before every command and routes to the Go binary via `spawnSync` when the gate passes.

This makes v5.4.0 the bridge between the Node era and the Go era -- it is the only Classic version that knows how to hand off to Go, which is the same architectural pattern the hybrid runtime milestone restores.

---

## Selection Rationale

1. **Bridge architecture.** v5.4.0 is the only version that contains the Node-to-Go delegation bridge (`version-gate.js` + `binary-downloader.js`). The `shouldDelegate()` function in `version-gate.js` checks binary availability, version match, and command type before routing to Go -- the exact pattern the hybrid runtime milestone restores and improves. No other Classic version has this capability.

2. **Complete module set.** With 16 modules instead of 14, v5.4.0 covers every Classic behavior. Using v5.3.x would leave `binary-downloader.js` and `version-gate.js` without behavioral documentation, creating a gap in the baseline.

3. **Production endpoint.** v5.4.0 was the last Classic release before the full Go transition. It represents the mature state of the Node.js runtime with accumulated fixes across 47 commits from v5.3.0 to v5.4.0.

---

## Known Limitations

1. **package.json reports wrong version.** The `package.json` at tag `v5.4.0` reports version `"5.3.3"` instead of `"5.4.0"`. This is a metadata bug that does not affect functionality.
   - **Workaround:** Use the git tag (`v5.4.0`) as the authoritative version identifier, not `package.json`.

2. **Delegation shim conflicts with current Go binary.** If `~/.aether/bin/aether` exists, v5.4.0 delegates all commands (except `install`, `update`, `setup`) to the Go binary, bypassing Node behavior entirely.
   - **Workaround:** Isolate the test environment by setting `HOME` to a temporary directory so `version-gate.js` cannot find `~/.aether/bin/aether`.

3. **plan/build/continue are slash commands only, not CLI subcommands.** In Classic, `plan`, `build`, and `continue` are Markdown wrappers executed by the AI platform (Claude Code / OpenCode), not registered CLI subcommands. The Node CLI has 13 subcommands: `init`, `install`, `update`, `version`, `uninstall`, `setup`, `checkpoint`, `sync-state`, `spawn-log`, `spawn-tree`, `status`, `nestmates`, `context`.
   - **Workaround:** Verify wrapper Markdown file existence and ceremony markers instead of testing CLI subcommands. Full lifecycle testing against ceremony output belongs in Phase 108 (Golden Workflow Tests).

4. **Requires Node.js and npm install.** The Classic CLI depends on Node.js runtime and 3 npm packages (`commander ^12.1.0`, `js-yaml ^4.1.0`, `picocolors ^1.1.1`). The `node_modules/` directory is not committed in the git tag.
   - **Workaround:** Run `npm install --production` after checking out the tag and before any CLI commands.

5. **commander npm dependency required.** The CLI uses the `commander` package for subcommand routing. Without it, `node bin/cli.js` fails with `Cannot find module 'commander'`.
   - **Workaround:** Run `npm install --production` before testing. No other external dependencies exist.

---

## Behavioral Checklist

### Restore in TS

These modules contain orchestration behavior that the TypeScript host must reimplement. They manage worker visibility, structured logging, and error handling -- the "living" parts of Aether that degraded during the Bash/Node-to-Go migration.

---

#### spawn-logger.js

- **Purpose:** Track worker spawn events with model information for visible worker activity logging.
- **Expected Behavior:**
  - Appends pipe-delimited log lines to `.aether/data/spawn-tree.txt` with format: `timestamp|parent|caste|child|task|model|status`
  - Parses both new-format (7-part) and old-format (6-part) spawn lines, plus spawn-complete format (3-4 part)
  - Logs to activity log via `logActivity()` from logger module on each spawn
  - Provides `formatSpawnTree()` for display-formatted output with caste emojis and status indicators
  - Supports filtering spawns by parent, caste, or model via `getSpawnsByParent()`, `getSpawnsByCaste()`, `getSpawnsByModel()`
  - Fails silently on I/O errors (never throws from logging)
  - Caste emoji map: `prime`, `builder`, `watcher`, `oracle`, `scout`, `chaos`, `architect`, `archaeologist`, `colonizer`, `route_setter`
  - Status emoji map: `spawned`, `completed`, `failed`, `blocked`
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Restore in TS
- **Migration Notes:** The TS host must reimplement `logSpawn()` and `formatSpawnTree()` to provide visible worker activity during builds. The current Go runtime has `spawn-log` and `spawn-tree` subcommands but lacks the structured event tracking that made Classic worker waves visible. The pipe-delimited format in `spawn-tree.txt` is the canonical spawn record format.

---

#### logger.js

- **Purpose:** Structured logging to `~/.aether/data/activity.log` with consistent formatting and silent failure.
- **Expected Behavior:**
  - Writes timestamped lines to `~/.aether/data/activity.log` with format: `[HH:MM:SS] emoji action caste: description`
  - Provides `logActivity(action, caste, description)` for caste-attributed activity entries
  - Provides `logError(error)`, `logWarning(code, message)`, `logInfo(message)`, `logSuccess(caste, description)` for typed log entries
  - Sanitizes log input: removes newlines/control characters, trims whitespace, caps at 200 characters
  - Fails silently on all I/O errors (never cascades logging failures)
  - Provides `getRecentLogs(lines)` for reading the tail of the activity log
  - Log level emojis: `ERROR`, `WARN`, `INFO`, `SUCCESS`
  - Caste emojis: `queen`, `scout`, `builder`, `watcher`, `chaos`, `ant`
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Restore in TS
- **Migration Notes:** The TS host must provide structured activity logging that captures worker lifecycle events. The current Go runtime writes to `activity.log` but the orchestration-layer events (spawn, dispatch, worker start/complete) need TS-level instrumentation to be visible during builds. The log format and silent-failure pattern should be preserved.

---

#### errors.js

- **Purpose:** Centralized error class hierarchy with structured JSON output, error codes, and recovery suggestions.
- **Expected Behavior:**
  - Base class `AetherError` extends `Error` with fields: `code`, `message`, `details`, `recovery`, `timestamp`
  - `toJSON()` returns structured `{ error: { code, message, details, recovery, timestamp } }` for programmatic consumption
  - `toString()` returns human-readable format with recovery suggestion
  - Specialized subclasses: `HubError`, `RepoError`, `GitError`, `ValidationError`, `FileSystemError`, `ConfigurationError`, `StateSchemaError`
  - Error codes enum (`ErrorCodes`): system (1-99), validation (100-199), runtime (200-299), unexpected (300-399), configuration (400-499)
  - `getExitCode(code)` maps error codes to sysexits.h exit codes
  - `wrapError(error)` converts plain `Error` to `AetherError`
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Restore in TS
- **Migration Notes:** The TS host must use a compatible error hierarchy for orchestration-layer errors. The Go runtime already has its own error types in `cmd/`; the TS errors cover the orchestration control plane (dispatch failures, worker communication errors, spawn tracking errors). The JSON output format enables `AETHER_OUTPUT_MODE=json` consumption.

---

### Keep in Go

These modules contain safety-critical behavior that the Go runtime already owns and must retain. They manage state integrity, file locking, visual rendering, installation, and updates.

---

#### state-guard.js

- **Purpose:** Enforce the Iron Law -- phase advancement requires fresh verification evidence.
- **Expected Behavior:**
  - `StateGuardError` class with structured error codes: `E_IRON_LAW_VIOLATION`, `E_IDEMPOTENCY_CHECK`, `E_LOCK_TIMEOUT`, `E_INVALID_TRANSITION`, `E_STATE_NOT_FOUND`, `E_STATE_INVALID`
  - Error includes `recovery` suggestion field for user guidance
  - `toJSON()` for structured programmatic output
  - Integrates with `EventTypes` from event-types module for audit trail
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`cmd/codex_continue_finalize.go`). The Go runtime owns all state mutation and verification gating. The TS host must never bypass state guard checks -- it calls Go finalizers which enforce the Iron Law.

---

#### caste-colors.js

- **Purpose:** Centralized caste styling with ANSI colors and emoji prefixes for consistent worker identity display.
- **Expected Behavior:**
  - `CASTE_STYLES` map: `builder` (blue, hammer), `watcher` (green, eye), `scout` (yellow, magnifier), `chaos` (red, dice), `prime` (magenta, crown)
  - `getCasteStyle(caste)` returns style object with `color`, `emoji`, `ansi`, `pc` (picocolors function)
  - `formatAnt(name, caste)` returns colored+emoji formatted string: `"hammer Builder"`
  - `formatAntAnsi(name, caste)` returns raw ANSI-formatted string for bash scripts
  - `getCastes()` returns all caste keys for iteration
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`cmd/codex_visuals.go`). The `casteColorMap`, `casteEmojiMap`, and `casteLabelMap` in Go provide the same ANSI-colored caste identity. The `casteIdentity()` and `casteLabel()` functions replace `formatAnt()` and `getCasteStyle()`.

---

#### event-types.js

- **Purpose:** Standardized event type definitions for the colony audit trail in COLONY_STATE.json.
- **Expected Behavior:**
  - `EventTypes` enum: `PHASE_TRANSITION`, `PHASE_BUILD_STARTED`, `PHASE_BUILD_COMPLETED`, `PHASE_ROLLED_BACK`, `CHECKPOINT_CREATED`, `CHECKPOINT_RESTORED`, `UPDATE_STARTED`, `UPDATE_COMPLETED`, `UPDATE_FAILED`, `IRON_LAW_VIOLATION`
  - `validateEvent(event)` checks required fields (`timestamp`, `type`, `worker`, `details`), ISO 8601 timestamp format, valid event type, non-empty worker string, and details-is-object constraint
  - `createEvent(type, worker, details)` generates a valid event with ISO timestamp
  - ISO 8601 regex validation: `YYYY-MM-DDTHH:MM:SS.sssZ` or `YYYY-MM-DDTHH:MM:SSZ`
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go. The Go runtime owns the event bus and audit trail. The TS host reads events via Go CLI JSON output but never writes events directly.

---

#### file-lock.js

- **Purpose:** PID-based file locking with stale lock detection for safe concurrent access to shared resources.
- **Expected Behavior:**
  - `FileLock` class with configurable: `lockDir` (default `.aether/locks`), `timeout` (5s), `retryInterval` (50ms), `maxRetries` (100), `maxLockAge` (5 min)
  - `acquire(filePath)` creates `.lock` and `.lock.pid` files, checks for stale locks by verifying PID is still running
  - `release()` removes lock and PID files
  - `_ensureLockDir()` creates lock directory recursively
  - `_registerCleanupHandlers()` prevents duplicate cleanup registration across instances
  - Constructor validates all options, throws `ConfigurationError` on invalid input
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`pkg/storage/`). Go owns all file locking for `.aether/data/` access. The TS host must never acquire locks directly -- it calls Go commands which handle locking internally.

---

#### banner.js

- **Purpose:** Shared ASCII art banner for Aether installer output.
- **Expected Behavior:**
  - Exports `BANNER` constant containing the "AETHER" ASCII art in block letters
  - Pure display module with no side effects
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`cmd/codex_visuals.go`). The Go runtime owns all visual rendering including banners and ceremony display.

---

#### colors.js

- **Purpose:** Centralized color palette with NO_COLOR support for consistent CLI theming.
- **Expected Behavior:**
  - Uses `picocolors` for lightweight terminal colors
  - `isColorEnabled()` checks `--no-color` flag, `NO_COLOR` env var, and TTY detection
  - Brand colors: `queen` (magenta), `colony` (cyan), `worker` (yellow)
  - Semantic colors: `success` (green), `warning` (yellow), `error` (red), `info` (blue)
  - Text styles: `bold`, `dim`, `italic`, `underline`, `strikethrough`
  - Header combinators: `header(text)` (bold+cyan), `subheader(text)` (bold)
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`cmd/codex_visuals.go`). Go owns all visual rendering. The TS host should not generate ANSI output for display -- it delegates rendering to Go.

---

#### init.js

- **Purpose:** Repository initialization with local state files, directory structure, and hub file syncing.
- **Expected Behavior:**
  - `initializeRepo(repoPath, { goal, setupOnly })` creates `.aether/` directory structure (data, checkpoints, locks)
  - Syncs system files from hub (`~/.aether/system/`) to local `.aether/`
  - Syncs Claude commands, OpenCode commands, agents, and Claude agents from hub
  - Creates `.aether/.gitignore` to exclude local state from version control
  - `createInitialState(goal)` generates COLONY_STATE.json with version, current_phase (0), events, goal, state (INITIALIZING), session
  - `generateSessionId()` produces `session_{timestamp}_{random}` format
  - `registerRepo(repoPath, version)` adds entry to `~/.aether/registry.json`
  - `isInitialized(repoPath)` checks for `.aether/data/COLONY_STATE.json` existence
  - `validateInitialization(repoPath)` verifies directory structure completeness
  - Atomic writes via temp-file-then-rename pattern
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`cmd/codex_init.go`). The Go runtime owns all state mutation including initialization. The TS host calls `aether init` (Go CLI) for initialization, never writes `.aether/data/` directly.

---

#### binary-downloader.js

- **Purpose:** Download platform-specific Go binary from GitHub Releases with SHA-256 verification and atomic install.
- **Expected Behavior:**
  - `downloadBinary(version, { quiet, timeout })` downloads correct binary for current platform
  - Platform detection via `getPlatformArch()`: maps `process.platform`/`process.arch` to goreleaser naming (darwin/linux/windows + amd64/arm64)
  - Follows HTTP 302 redirects (GitHub Releases always redirect)
  - Downloads `checksums.txt`, parses SHA-256 for the specific archive filename
  - Verifies download hash against expected hash before install
  - Extracts binary from tar.gz (POSIX) or zip (Windows) with `--strip-components=1`
  - `atomicInstall()` uses `fs.rename()` (atomic on POSIX) to install to `~/.aether/bin/aether`
  - Sets executable permission (`0o755`) on Unix
  - NEVER throws -- returns `{ success, reason, path }` object
  - Default timeout: 30 seconds
- **Versions Present:** v5.4.0 only (not in v5.3.0 or v5.3.3)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`pkg/downloader/`). Go owns the entire download, verify, and install pipeline. The TS host must never handle binary downloads.

---

#### update-transaction.js

- **Purpose:** Two-phase commit for updates with checkpoint, backup, sync, verify, and automatic rollback.
- **Expected Behavior:**
  - `UpdateError` class with recovery commands array for user guidance
  - Error codes: `E_UPDATE_FAILED`, `E_CHECKPOINT_FAILED`, `E_SYNC_FAILED`, `E_VERIFY_FAILED`, `E_ROLLBACK_FAILED`, `E_REPO_DIRTY`, `E_HUB_INACCESSIBLE`, `E_PARTIAL_UPDATE`, `E_NETWORK_ERROR`
  - Two-phase commit: backup current files, sync from hub, verify integrity, update version
  - Automatic rollback on any failure phase
  - Recovery commands displayed prominently on failure
  - v5.3.3/v5.4.0: filters exchange directory files by extension (`.sh` distributes, `.xml`/`.json` excluded)
- **Versions Present:** v5.3.0 (present), v5.3.3 (15 lines different), v5.4.0 (15 lines different from v5.3.0)
- **Classification:** Keep in Go
- **Migration Notes:** Already reimplemented in Go (`cmd/` update commands). Go owns all install/update/publish flows. The TS host calls `aether update` (Go CLI) for updates, never modifies hub files directly.

---

#### version-gate.js

- **Purpose:** Check Go binary availability, version match, and provide delegation logic for Node-to-Go command routing.
- **Expected Behavior:**
  - `checkBinary(opts)` returns `{ available, path, version, reason }` -- checks binary exists, is executable, version matches npm package
  - `shouldDelegate(argv, opts)` returns boolean -- routes to Go when gate passes, except for `NODE_ONLY_COMMANDS` (`install`, `update`, `setup`, `setup-hub`)
  - `compareVersions(a, b)` performs semver comparison handling `v` prefix and pre-release tags
  - `getBinaryPath()` returns `~/.aether/bin/aether`
  - Version check calls `aether version` via `execSync` and strips `aether v` prefix
  - 5-second timeout on version check
- **Versions Present:** v5.4.0 only (not in v5.3.0 or v5.3.3)
- **Classification:** Keep in Go
- **Migration Notes:** The delegation pattern is the architectural bridge that v5.4.0 introduced. The hybrid runtime milestone restores a similar pattern: the TS host calls Go for state mutations while handling orchestration in TypeScript. The Go runtime now IS the binary -- no version gate needed. Instead, the TS host calls Go CLI commands for all state-mutating operations.

---

### Obsolete

These modules served a purpose in the Classic era that is no longer needed. They have been replaced by Go implementations or by workflow changes.

---

#### state-sync.js

- **Purpose:** Synchronize `.planning/STATE.md` with `.aether/data/COLONY_STATE.json` to prevent split-brain state.
- **Expected Behavior:**
  - `syncStateFromPlanning(repoPath)` reads STATE.md and ROADMAP.md, updates COLONY_STATE.json fields (goal, current_phase, state, plan.phases)
  - `parseStateMd(content)` extracts phase, milestone, status, lastAction from Markdown
  - `parseRoadmapMd(content)` extracts phase objects with number, name, status from Markdown headers
  - `reconcileStates(repoPath)` detects mismatches between STATE.md and COLONY_STATE.json
  - `validateStateSchema(state)` checks required fields and types against schema definition
  - `pruneEvents(events, maxEvents)` caps events array at 100 entries (most recent by timestamp)
  - Uses file locking via `FileLock` for safe concurrent access
  - Atomic writes via temp-file-then-rename
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Obsolete
- **Migration Notes:** No longer needed. The Go runtime handles all state atomically via `pkg/storage/`. The split-brain problem existed because both Bash/Node and Markdown files could modify state independently. With Go as the sole authority for state mutation, cross-format synchronization is unnecessary.

---

#### interactive-setup.js

- **Purpose:** Interactive menu for `npx aether-colony` with environment-aware options (full setup, global only, repo only).
- **Expected Behavior:**
  - `interactiveSetup()` displays 3-option menu: Full setup, Global only, Repo only
  - `detectEnvironment()` checks hub installation, existing `.aether/`, and project directory (git, package.json, Makefile, pyproject.toml, Cargo.toml)
  - `getDefaultOption(env)` context-sensitive default: full setup for new projects, global only for no-hub, repo only for hub-without-local
  - Supports `--global`, `--repo`, `--yes` flag shortcuts
  - Non-TTY auto-selects default without prompting
  - Already-set-up detection with refresh option
  - Uses Node `readline` for input (zero npm dependencies for the menu itself)
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Obsolete
- **Migration Notes:** Replaced by the `/ant-discuss` flow and Go-based `aether init`. Interactive setup is now handled by the discuss-then-init workflow, which is more intentional and less error-prone than a menu.

---

#### nestmate-loader.js

- **Purpose:** Discover sibling repositories (nestmates) with Aether colonies and load their state and TODOs.
- **Expected Behavior:**
  - `findNestmates(currentRepoPath)` scans parent directory for sibling directories containing `.aether/`
  - `loadNestmateTodos(nestmatePath)` reads TODO items from `.planning/` files (lines matching `- [ ]` or `TODO:`)
  - `getNestmateState(nestmatePath)` reads COLONY_STATE.json for goal, state, currentPhase, milestone
  - `formatNestmates(nestmates)` formats nestmate list for display with goal truncation at 40 characters
  - Skips hidden directories (starting with `.`) and the current repo
  - Fails silently on all I/O errors
- **Versions Present:** v5.3.0, v5.3.3, v5.4.0 (identical across all three)
- **Classification:** Obsolete
- **Migration Notes:** Replaced by Go skill system (`aether skill-detect`, `aether skill-match`) and the registry system (`~/.aether/registry/`). Nestmate discovery is now handled by the Go runtime, not a standalone loader module.

---

## Cross-References

- **Runtime Boundary Contract:** [runtime-boundary-contract.md](./contracts/runtime-boundary-contract.md) -- Phase 106 deliverable containing the 16-module classification table that this document expands with behavioral detail.
- **Phase 107 Research:** [107-RESEARCH.md](../../../.planning/phases/107-classic-baseline-identification/107-RESEARCH.md) -- Full research data, module inventory, selection rationale, and pitfalls.
- **Phase 108 (Golden Workflow Tests):** Will use the "Restore in TS" modules (`spawn-logger.js`, `logger.js`, `errors.js`) as the behavioral specification for what the TS host must produce during golden test capture.
- **Phase 109 (TypeScript Orchestration Host):** Will use the "Restore in TS" classification to determine which Classic behaviors the TS host must reimplement, and "Keep in Go" to know which behaviors remain Go-exclusive.
