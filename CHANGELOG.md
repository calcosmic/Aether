# Changelog

All notable changes to the Aether Colony project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.11] - 2026-02-26

### Fixed
- `midden-recent-failures` now reads `.entries[]` from midden.json instead of querying non-existent `.signals[]` — builders can finally see past failures

### Added
- `instinct-create` subcommand with deduplication and 30-instinct cap — programmatic instinct management replaces manual JSON manipulation
- Midden context injected into builder prompts during build waves — workers avoid repeating past mistakes
- Decisions auto-emit FEEDBACK pheromones (strength 0.65, TTL 30d) so builders see architectural choices
- `context-update constraint` now handles `feedback` type alongside `redirect` and `focus`
- Context-update calls added to Claude Code pheromone commands (focus, redirect, feedback)
- OpenCode pheromone commands brought to parity with pheromone-write + context-update calls

### Changed
- `continue-advance.md` instinct extraction simplified to use `instinct-create` instead of inline JSON
- `learning-promote-auto` now also creates instincts from promoted learnings

## [1.1.10] - 2026-02-26

### Changed
- `lay-eggs` is now a pure bootstrap command — sets up `.aether/` in a repo from the global hub without starting a colony
- `init` now assumes Aether is already set up and focuses only on starting a colony with a goal
- All help files, rules, and workflow documentation updated to reflect the lay-eggs/init separation

## [1.1.5] - 2026-02-23

### Added
- `validate-worker-response` command with caste-aware schema checks for builder/watcher/probe/scout payloads, wired into build playbooks and OpenCode build flow.
- `queen-thresholds`, `spawn-efficiency`, `entropy-score`, `incident-rule-add`, and `eternal-store` commands.
- Incident/self-evolution artifacts: `.aether/docs/INCIDENT_TEMPLATE.md`, `.aether/scripts/weekly-audit.sh`, `.aether/scripts/incident-test-add.sh`.
- Additional regression coverage for spawn enforcement, state lock failure handling, threshold output, worker validation, spawn efficiency, eternal memory promotion, and entropy scoring.

### Changed
- Spawn guard now supports hard enforcement via `spawn-can-spawn [depth] --enforce`; worker and playbook guidance updated to use enforced mode.
- Build orchestration guidance now refreshes `colony-prime --compact` before each worker dispatch so new pheromones/memory are visible mid-build.
- Promotion thresholds consolidated behind a single source of truth and reused by learning promotion/proposal/memory metric paths.
- `midden.template.json` extended with `spawn_metrics`.

### Fixed
- Locked + atomic state writes for critical mutation paths in `COLONY_STATE.json` handling (`error-add`, failure event append in `spawn-complete`, `grave-add`, schema migration, and lock-aware state-loader validation handoff).
- Pheromone write path now sanitizes content, validates strength bounds, and uses lock + atomic write semantics.
- `pheromone-expire` now promotes high-strength expired signals (`>0.8`) into eternal memory instead of silently decaying.
- `detectDirtyRepo` now preserves porcelain status columns for the first line (prevents staged/unstaged misclassification when parsing git status output).

---

## v5.0.0 — Worker Emergence (2026-02-20)

**Major milestone:** Every ant caste is now a real Claude Code subagent. 22 agents ship through the hub, ready for resolution via the Task tool.

### Added
- **22 Claude Code subagents**: Every ant caste is now a first-class subagent resolvable via the Task tool
  - Core: Builder, Watcher
  - Orchestration: Queen, Scout, Route-Setter, 4 Surveyors (nest, disciplines, pathogens, provisions)
  - Specialists: Keeper, Tracker, Probe, Weaver, Auditor
  - Niche: Chaos, Archaeologist, Gatekeeper, Includer, Measurer, Sage, Ambassador, Chronicler
- **Agent distribution pipeline**: `npm install -g .` syncs agents to hub at `~/.aether/system/agents-claude/`, `aether update` delivers to target repos
- **6 AVA tests for agent quality**: Frontmatter validation, tool restrictions, naming conventions, content standards
- **repo-structure.md**: Quick re-orientation guide for the codebase

### Fixed
- **Bash line wrapping bug**: Fixed 58 instances across 7 command files where description text was inside code blocks causing "with: command not found" errors
- **Lint regression test**: CLEAN-03 test scans both Claude and OpenCode command directories

### Changed
- **.aether/docs/ curated**: 8 keepers at root, 6 archived for reference
- **README.md updated**: Action-oriented tone, v5.0 agent capabilities featured, caste table by tier
- **ROADMAP.md and STATE.md**: v5.0 marked shipped, all 31 phases complete at 100%

### Phases Shipped
- **Phase 27**: Distribution Infrastructure + First Core Agents (4 plans)
- **Phase 28**: Orchestration Layer + Surveyor Variants (3 plans)
- **Phase 29**: Specialist Agents + Agent Tests (3 plans)
- **Phase 30**: Niche Agents (3 plans)
- **Phase 31**: Integration Verification + Cleanup (3 plans)

---

### Removed
- Old planning phases 10-19 archived to docs/plans/ (completed phases)
- Orphaned worktree salvage files moved to docs/worktree-salvage/

---

## v4.0.0 -- Distribution Simplification

**Breaking change:** The `runtime/` staging directory has been removed. The npm package now reads directly from `.aether/`, with private directories (data/, dreams/, oracle/, etc.) excluded by `.aether/.npmignore`.

### Changed
- `.aether/` is now directly included in the npm package (private dirs excluded by `.aether/.npmignore`)
- `bin/validate-package.sh` replaces `bin/sync-to-runtime.sh` — validates required files, no copying
- Hub sync uses exclude-based approach instead of triplicate 59-62 file allowlists
- Pre-commit hook repurposed for validation (no more runtime/ sync)
- `aether update` uses `syncAetherToRepo` (exclude-based) for all system file distribution
- All three distribution paths (system files, commands, agents) unified in `setupHub()`

### Removed
- `runtime/` staging directory — eliminated entirely
- `bin/sync-to-runtime.sh` — replaced by validation-only script
- `SYSTEM_FILES` allowlist arrays in cli.js and update-transaction.js
- `copySystemFiles()` and `syncSystemFilesWithCleanup()` functions

### Added
- `bin/validate-package.sh` — pre-packaging validation with `--dry-run` mode
- Private data exposure guard — blocks packaging if .npmignore doesn't cover private dirs
- Migration message for users upgrading from v3.x
- `npm pack --dry-run` recommended for verifying package contents

### Fixed
- ISSUE-004: Template path hardcoded to runtime/ — resolved by eliminating runtime/ entirely

### Migration
- Run `npm install -g aether-colony` to get v4.0
- Your colony state and data are unaffected
- The only change is how the package is built — distributed content is identical

---

## [3.1.5] - 2026-02-15

### Fixed
- **Agent Type Correction** — Changed all occurrences of `subagent_type="general"` to `subagent_type="general-purpose"` across all command files. The error "Agent type 'general' not found" was occurring because the correct agent type name is `general-purpose`. Fixed in: build.md, plan.md, organize.md, and workers.md (both Claude and OpenCode versions, plus runtime copy). (`.claude/commands/ant/build.md`, `.claude/commands/ant/plan.md`, `.claude/commands/ant/organize.md`, `.opencode/commands/ant/build.md`, `.opencode/commands/ant/plan.md`, `.opencode/commands/ant/organize.md`, `.aether/workers.md`, `runtime/workers.md`)

## [3.1.4] - 2026-02-15

### Fixed
- **Archaeologist Visualization** — Added swarm display integration for the Archaeologist scout (Step 4.5). The archaeologist now appears in the visual display with proper emoji (🏺), progress tracking (15% → 100%), and tool usage stats when spawned during pre-build scans. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)

## [3.1.3] - 2026-02-15

### Fixed
- **Nested Spawn Visualization** — When builders or watchers spawn sub-workers, the swarm display now updates to show those nested spawns with colors and emojis. Added `swarm-display-update` calls to workers.md spawn protocol (Step 3 and Step 5), builder prompts, and watcher prompts. (`.aether/workers.md`, `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)

## [3.1.2] - 2026-02-15

### Fixed
- **Swarm Display Integration in Build Command** — The visualization system was fully implemented but never integrated into `/ant:build`. Added `swarm-display-init` at build start, `swarm-display-update` calls when spawning builders/watchers/chaos ants, progress updates when workers complete (updating to 100% completion), and final `swarm-display-render` at build completion. The build now shows real-time ant-themed visualization with caste emojis, colors, tool usage stats, and chamber activity maps. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)
- **Missing swarm-display-render Command** — Added new `swarm-display-render` command to `aether-utils.sh` that executes the visualization script to render the current swarm state to terminal. (`.aether/aether-utils.sh`)

### Changed
- **OpenCode Build Command Parity** — Synchronized OpenCode build.md with Claude version: added `--model` flag support, proxy health check (Step 0.6), colony state loading (Step 0.5), and full swarm display integration. (`.opencode/commands/ant/build.md`)

## [3.1.1] - 2026-02-15

### Fixed
- **Missing Visualization Assets** — Added `.aether/visualizations/` directory to npm package files array. The ASCII art anthill files required by `/ant:maturity` command were not being published, causing the command to fail in repos that installed/updated via npm. (`package.json`)
- **Visualization Sync in Install** — Updated `setupHub()` function in CLI to sync visualization files from package to hub (`~/.aether/visualizations/`). (`bin/cli.js`)
- **Visualization Sync in Update** — Updated `UpdateTransaction` to sync visualization files from hub to repos during `aether update`. Added `HUB_VISUALIZATIONS` constant and visualization sync result tracking. (`bin/lib/update-transaction.js`)

### Changed
- Version bump to 3.1.1 to trigger fresh installs with visualization assets. (`package.json`)

## [Unreleased]

### Added
- **Session Freshness Detection System** — Global system to prevent stale session files from silently breaking Aether workflows. Implements `session-verify-fresh` and `session-clear` commands with support for 7 commands (survey, oracle, watch, swarm, init, seal, entomb). Features cross-platform timestamp detection (macOS/Linux), environment variable overrides for testing, and protected operations (init/seal/entomb never auto-clear). Backward compatibility maintained with `survey-verify-fresh` and `survey-clear` wrappers. Added comprehensive test suite (`tests/bash/test-session-freshness.sh`) and API documentation (`docs/session-freshness-api.md`). (`.aether/aether-utils.sh`, `tests/bash/test-session-freshness.sh`, `docs/session-freshness-api.md`)
  - `/ant:colonize` — Added `--force-resurvey` flag, stale survey detection, and verification
  - `/ant:oracle` — Added `--force` flag, stale session detection with user options
  - `/ant:watch` — Added session timestamp capture and stale file handling
  - `/ant:swarm` — Added auto-clear for stale findings with verification
  - `/ant:init` — Added freshness check with protected state (no auto-clear)
  - `/ant:seal` — Added incomplete archive detection and integrity verification
  - `/ant:entomb` — Added incomplete chamber detection and integrity verification

### Fixed
- **Architecture Cleanup: Source of Truth Flipped** — Complete review and cleanup of the flipped source-of-truth architecture. `.aether/` is now the source of truth for system files, with `runtime/` auto-populated by `bin/sync-to-runtime.sh` during npm install. Fixed 6 stale documentation files, updated 5 planning files, expanded allowlist from 20 to 36 files, handled 4 orphan files, and verified zero drift between directories. (`.aether/recover.sh`, `.aether/RECOVERY-PLAN.md`, `.planning/codebase/STRUCTURE.md`, `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONVENTIONS.md`, `TO-DOS.md`, `bin/sync-to-runtime.sh`, `bin/lib/update-transaction.js`)

### Changed
- **Phase 4: UX Improvements to Lay-Eggs** — Enhanced lay-eggs.md with visual lifecycle diagrams (🟢 ACTIVE COLONY → 🏺 SEAL/ENTOMB → 🥚 LAY EGGS) in error messages to clarify workflow progression. Updated success output to explicitly distinguish between sealing vs laying eggs for new projects, preserving wisdom across colony lifecycles. (`.claude/commands/ant/lay-eggs.md`)

### Fixed
- **Phase 2: Fix Blocker Severity and Auto-Resolve Logic** — Made auto_resolve_on conditional by flag source: chaos-sourced blockers require manual resolution (auto_resolve_on: null), verification blockers auto-resolve on build pass. Reordered continue.md Flags Gate to run auto-resolve before blocker count check. Added advisory blocker warning (Step 1.5) to build.md so builders see active blockers before execution. (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`, `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`, `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)
- **Phase 1: Fix Chaos Ant Duplicate Flagging** — Eliminated duplicate flag creation during build-rebuild cycles by removing redundant chaos flagging from build.md Step 5.5 (Step 5.4.2 already handles it), injected existing flag titles into Chaos Ant spawn prompt to prevent re-investigating known issues, and added flag persistence to standalone /ant:chaos for critical/high findings using source 'chaos-standalone'. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.claude/commands/ant/chaos.md`, `.opencode/commands/ant/chaos.md`)

### Added
- **Phase 6: Final Verification and Integration Testing** — Added milestone display to /ant:status command (now shows "Milestone: <name>" in output), expanded milestone progression in /ant:archive to handle all 6 stages (First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill), added unrecognized milestone error handling. Full lint suite passes (shell, JSON, sync - 28 commands verified). (`.claude/commands/ant/status.md`, `.claude/commands/ant/archive.md`, `.opencode/commands/ant/status.md`, `.opencode/commands/ant/archive.md`)

- **Phase 1: Create Oracle infrastructure and command** — Added Oracle Ant deep research agent with RALF-pattern bash loop, agent prompt, /ant:oracle command definition (mirrored), and oracle caste registration in aether-utils.sh. (`.aether/oracle/oracle.sh`, `.aether/oracle/oracle.md`, `.claude/commands/ant/oracle.md`, `.opencode/commands/ant/oracle.md`, `.aether/aether-utils.sh`, `runtime/aether-utils.sh`)

### Verified
- **Phase 2: Verification and smoke test** — All Oracle Ant files verified: generate-commands.sh check passes (26/26 in sync, SHA-1 checksums verified), oracle.sh error handling works (exits code 1 with descriptive error when no research.json), oracle caste generates themed names (Vision-NN, Delph-NN), file structure matches spec (oracle.sh executable, oracle.md exists, command mirrors byte-identical, no stray files). Full lint suite passes (shell, JSON, sync).

### Verified
- **Phase 5: Path Localization Complete** — Full-repo audit confirmed zero actionable `~/.aether/` or `~/.config/opencode/` references in commands, scripts, or CLI. Remaining 4 `$HOME/.aether` references in aether-utils.sh are intentional hub/registry functions for multi-repo management. `generate-commands.sh check` passes (25/25 in sync, SHA-1 checksums verified). `aether-utils.sh` smoke tests pass (help, version, generate-ant-name). All three original goals met: no root access prompts, no cross-repo contamination, no out-of-project file operations.

### Changed
- **Phase 4: Implement Hash-Based Idempotency** — Added SHA256 hash comparison to syncDirWithCleanup function - files are now only copied when content actually changes, reducing unnecessary I/O. Added error handling with try-catch to hashFileSync, copyFileSync, unlinkSync operations to prevent crashes on single file errors. Added validateManifest function to verify manifest.json structure before use. Added optional --backup flag to preserve user-modified files before overwriting. 13 tests added covering hash comparison, user modification detection, and backward compatibility. (`bin/cli.js`, `test/sync-dir-hash.test.js`, `test/user-modification-detection.test.js`)

- **Phase 4: Remove global path operations from cli.js** — Removed `~/.aether/` runtime copy logic (RUNTIME_DEST, RUNTIME_SRC, learnings.json creation, execSync import) and `~/.config/opencode/` global install logic (OPENCODE_GLOBAL_COMMANDS_DEST, OPENCODE_GLOBAL_AGENTS_DEST) from install/uninstall commands. Updated help text to reflect new architecture. cli.js now only installs Claude Code slash-commands to `~/.claude/commands/ant/`. Net reduction of 130 lines. (`bin/cli.js`)
- **Phase 3: Document repo-local path architecture** — Updated CHANGELOG.md and documentation references to reflect the completed path localization migration. All runtime paths now use repo-local `.aether/` instead of `~/.aether/`. Phases 1-2 localized command files, agent definitions, system docs, planning docs, shell utilities, and cross-project state functions. Phase 3 updates documentation to reflect the new architecture where running colonies only read/write repo-local `.aether/`, while global install remains functional for command distribution. (`README.md`, `CHANGELOG.md`, `TO-DOS.md`)
- **Phase 2: Localize cross-project state in aether-utils.sh** — Redirected 6 `$HOME/.aether` references to repo-local `$DATA_DIR` paths in learning-promote, learning-inject, error-flag-pattern, error-patterns-check, signature-scan, and signature-match functions. Fixed atomic-write.sh `$HOME/.aether/utils` fallback to use SCRIPT_DIR-based resolution. Updated usage comment headers in all .sh files. Applied to both `.aether/` and `runtime/` copies (verified identical). (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`, `.aether/utils/atomic-write.sh`, `runtime/utils/atomic-write.sh`, `.aether/utils/file-lock.sh`, `runtime/utils/file-lock.sh`)
- **Phase 1: Localize ~/.aether/ path references** — Replaced all `~/.aether/` paths with repo-relative `.aether/` across 50 files: command prompts, agent definitions, system docs, planning docs, and template.yaml. Fixed 3 pre-existing mirror drifts (migrate-state.md, organize.md, plan.md) discovered during verification. (`.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.opencode/agents/*.md`, `.aether/workers.md`, `runtime/workers.md`, `.aether/docs/*.md`, `runtime/docs/*.md`, `.planning/*.md`, `src/commands/_meta/template.yaml`)
- **Phase 2: Upgrade Sync Checking to Content-Aware** — `generate-commands.sh check` now performs SHA-1 checksum comparison (Pass 2) after filename matching (Pass 1), detecting content drift between `.claude/` and `.opencode/` mirrors. Revealed 3 pre-existing drifts previously invisible to filename-only checks. (`bin/generate-commands.sh`)

### Verified
- **Phase 5: Verify Full System Integrity** — Final verification phase confirming all global install locations match repo sources. Full lint suite passed (lint:shell, lint:json, lint:sync). All 4 global locations verified: ~/.claude/commands/ant/ (24 files), ~/.config/opencode/commands/ant/ (24 files), ~/.config/opencode/agents/ (4 files), ~/.aether/ system files. Watcher quality 9/10, Chaos resilience moderate (1 high finding: lint:sync content blind spot, 3 medium, 1 low — all pre-existing infrastructure gaps). Colony goal achieved.

### Fixed
- **Phase 6: Document and Test the System** — Created command-sync.md documenting the sync strategy: Claude Code uses global sync to ~/.claude/commands/ant/, OpenCode uses hub-based repo-local distribution (no global discovery). Verified end-to-end sync with dry-run tests: install, update, update --all all work correctly. Idempotency confirmed - hash-based comparison skips unchanged files. All lint and tests pass. (`.aether/docs/command-sync.md`, `.aether/docs/namespace.md`, `test/namespace-isolation.test.js`)

- **Phase 5: Implement Conflict Prevention System** — Added namespace isolation documentation (`.aether/docs/namespace.md`) explaining why 'ant' namespace is distinct from cds, mds, st: namespaces. Created namespace-isolation.test.js with 8 tests verifying bulletproof directory-based isolation. Fixed critical bugs discovered by Chaos Ant: added hash comparison to syncDirWithCleanup (files now only copied when content changes) and added HOME environment variable validation to prevent path.join failures. All 14 tests pass. (`bin/cli.js`, `.aether/docs/namespace.md`, `test/namespace-isolation.test.js`)

- **Phase 3: Fix Pheromone Model Consistency** — Aligned all pheromone documentation to TTL-based model. Replaced decay/half-life/exponential language in runtime/docs/pheromones.md (now identical to .aether/docs/ source of truth), help.md (4 references fixed, mirrored to .opencode/), and README.md pheromone table (Decay column replaced with Priority/Default Expiration). (`runtime/docs/pheromones.md`, `.claude/commands/ant/help.md`, `.opencode/commands/ant/help.md`, `README.md`)
- **Phase 1: Fix Command Mirror Sync Bugs** — Synced status.md, continue.md, and phase.md between Claude and OpenCode mirrors (zero diff verified), added missing YAML frontmatter to Claude's migrate-state.md, verified cli.js install paths are correct. (`.opencode/commands/ant/status.md`, `.opencode/commands/ant/continue.md`, `.opencode/commands/ant/phase.md`, `.claude/commands/ant/migrate-state.md`)
- **Phase 2: Sync runtime copy to .aether mirror** — Full file copy of runtime/aether-utils.sh to .aether/aether-utils.sh eliminating all drift including missing signature-scan and signature-match commands. Both copies now byte-identical. (`.aether/aether-utils.sh`)
- **Phase 1: Fix bugs in canonical runtime/aether-utils.sh** — Fixed learning-promote jq crash on non-numeric phase strings (--argjson to --arg), fixed flag-auto-resolve missing exit after early-return when flags file absent, confirmed file-lock.sh transitive usage via atomic-write.sh is intentional. (`runtime/aether-utils.sh`)

### Changed
- **Phase 4: Clean Up Global ~/.aether/** — Removed unrelated `LIGHT_MODE_TRANSPARENCY_TEST.md` from global `~/.aether/`, adopted orphaned `progressive-disclosure.md` into repo at both `.aether/docs/` and `runtime/docs/`. Both copies verified identical to global source. (`.aether/docs/progressive-disclosure.md`, `runtime/docs/progressive-disclosure.md`)
- **Phase 3: Sync Global OpenCode Commands** — Replaced stale global OpenCode commands at `~/.config/opencode/commands/ant/` with all 24 current repo commands. Removed orphan `ant.md`, cleared old files, installed fresh copies. All 24 files verified identical to repo source. (`~/.config/opencode/commands/ant/*.md`)
- **Phase 2: Sync Content Between Repo and Runtime** — Synced runtime/QUEEN_ANT_ARCHITECTURE.md with .aether/ source (+56 lines: Council, Swarm sections, heading rename), added 3 missing docs to runtime/docs/ (constraints.md, pathogen-schema.md, pathogen-schema-example.json), synced aether-watcher.md to global install with Command Resolution section. (`runtime/QUEEN_ANT_ARCHITECTURE.md`, `runtime/docs/*`, `~/.config/opencode/agents/aether-watcher.md`)
- **Phase 1: Fix OpenCode Command Naming Convention** — Renamed all 24 `.opencode/commands/ant/` files from `ant:*.md` to bare `*.md` names to match `.claude/commands/ant/` convention. OpenCode uses frontmatter `name:` field for command resolution, so filenames are cosmetic. `npm run lint:sync` now passes. (`.opencode/commands/ant/*.md`)
- **Phase 4: Documentation and Validation (Chaos + Archaeologist)** — Updated help.md with /ant:chaos and /ant:archaeology in ADVANCED and WORKER CASTES sections, updated README.md command count from 22 to 24 in all 6 locations, added CHANGELOG entries for all phases, marked both TO-DOS.md entries as DONE with implementation references. Validated 24 files in each command directory, name generation, and emoji resolution. (`.claude/commands/ant/help.md`, `.opencode/commands/ant/ant:help.md`, `README.md`, `CHANGELOG.md`, `TO-DOS.md`)
- 2026-02-12: TO-DOS.md — Marked Chaos Ant and Archaeologist Ant entries as DONE with implementation references
- **Phase 1: Threshold and Quoting Fixes** — Lowered instinct confidence threshold from 0.7 to 0.5 in both init.md mirrors, standardized YAML description quoting across all 26 command files. (`init.md`, `build.md`, `colonize.md`, `continue.md`, `council.md`, `dream.md`, `feedback.md`, `flag.md`, `flags.md`, `focus.md`, `help.md`, `interpret.md`, `organize.md`, `pause-colony.md`, `phase.md`, `plan.md`, `redirect.md`, `resume-colony.md`, `status.md`, `swarm.md`, `watch.md` + .opencode mirrors)
- **Phase 3: Watcher, Builder, and Swarm command resolution** — Watcher prompt in build.md, swarm.md Step 8, and aether-watcher.md now resolve build/test/lint commands via the 3-tier priority chain (CLAUDE.md > CODEBASE.md > heuristic fallback) instead of leaving commands unspecified or hardcoded. (`build.md`, `swarm.md`, `aether-watcher.md` + .opencode mirrors)
- **Phase 2: Verification loop priority chain** — Command detection in continue.md and verification-loop.md now uses 3-tier priority chain (CLAUDE.md > CODEBASE.md > heuristic table) instead of heuristic table alone. Heuristic table preserved as fallback. (`continue.md`, `runtime/verification-loop.md` + .opencode/.aether mirrors)
- **Phase 3: Build Pipeline Integration (Chaos + Archaeologist)** — Integrated both new ant types into the build.md pipeline. Archaeologist Ant spawns as conditional pre-build step (Step 4.5) when phase modifies existing files, injecting history context into builder prompts. Chaos Ant spawns as post-build resilience tester (Step 5.4.2) alongside Watcher, limited to 5 edge case scenarios. Added `chaos_count` and `archaeologist_count` to spawn_metrics and `archaeology` field to synthesis JSON. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/ant:build.md`)

### Added
- **Phase 2: `/ant:chaos` command** — Standalone Chaos Ant (Resilience Tester) command that probes code for edge cases, boundary conditions, error handling gaps, state corruption, and unexpected inputs. Produces structured findings reports with reproduction steps and severity ratings. Read-only by design (Tester's Law). (`.claude/commands/ant/chaos.md`, `.opencode/commands/ant/ant:chaos.md`)
- **Phase 2: `/ant:archaeology` command** — Standalone Archaeologist Ant command that excavates git history for any file or directory. Uses git log, blame, show, and follow to analyze commit patterns, surface tribal knowledge, identify tech debt markers, map churn hotspots, and produce structured archaeology reports. Read-only by design (Archaeologist's Law). (`.claude/commands/ant/archaeology.md`, `.opencode/commands/ant/ant:archaeology.md`)
- **Phase 1: Utility Foundation (Chaos + Archaeologist)** — Added chaos and archaeologist castes to `generate-ant-name` (8 prefixes each) and `get_caste_emoji` (🎲 and 🏺) in both `.aether/aether-utils.sh` and `runtime/aether-utils.sh`. (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`)
- **Phase 1: Immune Memory Schema** — Defined JSON schema for pathogen signatures extending existing error-patterns.json format. Schema adds signature_type, pattern_string, confidence_threshold, escalation_level fields while preserving backward compatibility. Created .aether/docs/pathogen-schema.md documentation, .aether/docs/pathogen-schema-example.json with sample entries, and .aether/data/pathogens.json empty storage file. Watcher verified 6/6 jq validation tests pass. (`.aether/docs/pathogen-schema.md`, `.aether/docs/pathogen-schema-example.json`, `.aether/data/pathogens.json`)
- **Phase 2: Add Lint Scripts** — Added `lint:shell`, `lint:json`, `lint:sync`, and top-level `lint` scripts to package.json for shell validation, JSON validation, and mirror sync checking. (`package.json`)
- **CLAUDE.md-aware command detection** — Colonize now extracts build/test/lint commands from CLAUDE.md and package manifests into CODEBASE.md with user suggestions. Verification loop and worker prompts resolve commands via 3-tier priority chain (CLAUDE.md > CODEBASE.md > heuristic fallback) instead of heuristic table alone. (`colonize.md`, `continue.md`, `build.md`, `swarm.md`, `verification-loop.md`, `aether-watcher.md` + .opencode/.aether mirrors)
- **Phase 4: Tier 2 Gate-Based Commit Suggestions** — Colony now suggests commits at verified boundaries (post-advance and session-pause) via user prompt instead of auto-committing. Added `generate-commit-message` utility to aether-utils.sh for consistent formatting across commit types. (`continue.md`, `pause-colony.md`, `aether-utils.sh` + .opencode mirrors)
- **Phase 3: Tier 1 Safety Formalization** — Switched build.md checkpoint from `git commit` to `git stash push --include-untracked`, standardized checkpoint naming under `aether-checkpoint:` prefix, added label parameter to `autofix-checkpoint` in aether-utils.sh, added rollback verification to build.md output header, documented rollback procedure in continue.md, updated swarm.md to pass descriptive labels. (`build.md`, `swarm.md`, `continue.md`, `aether-utils.sh` + .opencode mirrors)
- **Phase 2: Git Staging Strategy Proposal** — 4-tier strategy proposal with comparison matrix and implementation recommendation. Tier 1 (Safety-Only), Tier 2 (Gate-Based Suggestions), Tier 3 (Hooks-Based Automation), Tier 4 (Branch-Aware Colony). Recommends Tiers 1+2 for initial implementation. (`.planning/git-staging-proposal.md`, `.planning/git-staging-tier{1-4}.md`)
- **Phase 1: Deep Research on Git Staging Strategies** — 7 research documents (1573 lines) covering: Aether's 20 git touchpoints, industry comparison of 5 AI tools, worktree applicability, user git rule tensions, ranked commit points (POST-ADVANCE strongest), commit message conventions, and GitHub integration opportunities. (`.planning/git-staging-research-1.{1-7}.md`)
- **Auto-recovery headers** — All ant commands now show `🔄 Resuming: Phase X - Name` after `/clear`. `status.md` has Step 1.5 with extended format including last activity timestamp. `build.md`, `plan.md`, `continue.md` show brief one-line context. `resume-colony.md` documents the tiered pattern. (`status.md`, `build.md`, `plan.md`, `continue.md`, `resume-colony.md`)
- **Ant Graveyards** — `grave-add` and `grave-check` commands in `aether-utils.sh`. When builders fail, grave markers record the file, ant name, and failure summary. Future builders check for nearby graves before modifying files and adjust caution level accordingly. Capped at 30 entries. (`aether-utils.sh`, `init.md`, `build.md`)
- **Colony knowledge in builder prompts** — Spawned workers now receive top instincts (confidence >= 0.5), recent validated learnings, and flagged error patterns via `--- COLONY KNOWLEDGE ---` section in builder prompt template. (`build.md`)
- **Automatic changelog updates** — `/ant:continue` now appends a changelog entry for each completed phase under `## [Unreleased]`. (`continue.md`)
- **Colony memory inheritance** — `/ant:init` now reads the most recent `completion-report.md` (if it exists) and seeds the new colony's `memory.instincts` with high-confidence instincts (>= 0.7) and validated learnings from prior sessions. Colonies no longer start completely blind. (`init.md` + .opencode mirror)
- **Unbuilt design status markers** — Added `STATUS: NOT IMPLEMENTED` headers to `.planning/git-staging-tier3.md` and `.planning/git-staging-tier4.md` to prevent confusion with implemented features. (`git-staging-tier3.md`, `git-staging-tier4.md`)
- **`/ant:interpret` command** — Dream reviewer that loads dream sessions, investigates each observation against the actual codebase with evidence and verdicts (confirmed/partially confirmed/unconfirmed/refuted), assesses concern severity, estimates implementation scope, and facilitates discussion before injecting pheromones or adding TO-DOs. (`interpret.md`)
- **`/ant:dream` command** — Philosophical wanderer agent that reads codebase, git history, colony state, and TO-DOs, performs random exploration cycles and writes observations to `.aether/dreams/`. (`dream.md`)
- **`/ant:help` command** — Renamed from `/ant:ant` with updated content covering all 20 commands, session resume workflow, colony memory system, and full state file inventory. (`help.md`)
- **OpenCode command sync** — All `.claude/commands/ant/` prompts synced to `.opencode/commands/ant/` for cross-tool parity

### Changed
- **Checkpoint messaging** — Now suggests actual next command (e.g., `/ant:continue` or `/ant:build 3`) instead of generic `/ant:status`. Format: "safe to /clear, then run /ant:continue"
- **Caste emoji in spawn output** — Spawn-log and spawn-complete in `aether-utils.sh` show caste emoji adjacent to ant name (e.g., `🔨Chip-36`). Build.md SPAWN PLAN and Colony Work Tree use emoji-first format. (`aether-utils.sh`, `build.md`)
- **Phase context in command suggestions** — Next Steps sections now include phase names alongside numbers (e.g., `/ant:build 3   Phase 3: Add Authentication`). (`status.md`, `plan.md`, `phase.md`)
- **OpenCode plan.md** — Now dynamically calculates first incomplete phase instead of hardcoding Phase 1. (`plan.md`)

### Fixed
- **Output appears before agents finish** — `build.md` now enforces blocking behavior; Steps 5.2, 5.4.1, and 5.6 wait for all TaskOutput calls before proceeding
- **Command suggestions use real phase numbers** — `status.md`, `continue.md`, `plan.md`, and `phase.md` calculate actual phase numbers instead of showing template placeholders
- **Progressive disclosure UI** — Compact-by-default output with `--verbose` flag; `status.md` (8-10 lines) and `build.md` (12 lines) default to compact mode

## [1.0.0] - 2026-02-09

### First Stable Release

Aether Colony is a multi-agent system using ant colony intelligence for Claude Code and OpenCode. Workers self-organize via pheromone signals to complete complex tasks autonomously.

### Added
- **20 ant commands** for autonomous project planning, building, and management (`ant:init`, `ant:plan`, `ant:build`, `ant:continue`, `ant:status`, `ant:phase`, `ant:colonize`, `ant:watch`, `ant:flag`, `ant:flags`, `ant:focus`, `ant:redirect`, `ant:feedback`, `ant:pause-colony`, `ant:resume-colony`, `ant:organize`, `ant:council`, `ant:swarm`, `ant:ant`, `ant:migrate-state`)
- **Multi-agent emergence** — Queen spawns workers directly; workers can spawn sub-workers up to depth 3
- **Pheromone signals** — FOCUS, REDIRECT, and FEEDBACK with TTL-based filtering
- **Project flags** — Blockers, issues, and notes with auto-resolve triggers
- **State persistence** — v3.0 consolidated `COLONY_STATE.json` with session handoff via pause/resume
- **Command output styling** — Emoji sandwich styling across all ant commands
- **Git checkpoint/rollback** — Automatic commits before each phase for safety
- **`aether-utils.sh` utility layer** — Single entry point for deterministic colony operations (error tracking, activity logging, spawn management, flag system, antipattern checks, autofix checkpoints)
- **OpenCode compatibility** — Full command mirror in `.opencode/commands/ant/`

### Architecture
- Queen ant orchestrates via pheromone signals
- Worker castes: Builder, Scout, Watcher, Architect, Route-Setter
- Wave-based parallel spawning with dependency analysis
- Independent Watcher verification with execution checks
- Consolidated `workers.md` for all caste disciplines

## [Pre-1.0] - 2026-02-01 to 2026-02-08

Development releases (versions 2.0.0-2.4.2) building toward stable release. Key milestones:

### 2026-02-08
- **v2.0 nested spawning** — Direct Queen spawning, enforcement gates, flagging system
- **OpenCode cross-tool compatibility** — Commands available in both Claude Code and OpenCode
- **ant:swarm** — Parallel scout investigation for stubborn bugs
- **ant:council** — Multi-choice intent clarification

### 2026-02-07
- **True emergence system** — Worker-spawns-worker architecture
- **Verification gates** — Worker disciplines enforced
- **v1.0.0 release prep** — Auto-upgrade from old state formats

### 2026-02-06
- **State consolidation (v2.0 → v3.0)** — 5 state files merged into single `COLONY_STATE.json`
- **State migration command** — `ant:migrate-state` for upgrading existing colonies
- **Signal schema unification** — TTL-based signal filtering replacing decay system
- **Command trim** — Reduced `status.md` from 308 to 65 lines, signal commands to 36 lines each, `aether-utils.sh` from 317 to 85 lines (later expanded with new features)
- **Worker spec consolidation** — 6 separate worker specs merged into single `workers.md`
- **Build/continue rewrite** — Minimal state writes, detection and reconciliation pattern

### 2026-02-05
- **NPM distribution** — Global install via `npm install -g`
- **Global learning system** — `learning-promote` and `learning-inject` for cross-project knowledge
- **Queen-mediated spawn tree** — Depth-limited spawning with tree visualization
- **ant:organize** — Codebase hygiene scanning (report-only)
- **Debugger spawn on retry failure** — Automatic debugging assistance
- **Multi-colonizer synthesis** — Disagreement flagging during analysis
- **Multi-dimensional watcher scoring** — Richer verification rubrics

### 2026-02-04
- **Auto-continue mode** — `--all` flag for `/ant:continue`
- **Safe-to-clear messaging** — State persistence indicators on all commands
- **Conflict prevention** — File overlap validation between parallel workers
- **Phase-aware error tracking** — Error-add wired to phase numbers

### 2026-02-01 to 2026-02-03
- **Initial AETHER system** — Autonomous agent spawning core
- **Queen Ant Colony** — Phased autonomy with pheromone-based guidance
- **Pheromone communication** — FOCUS, REDIRECT, FEEDBACK emission commands with worker response
- **Triple-Layer Memory** — Working memory, short-term compression, long-term patterns
- **State machine orchestration** — Transition validation with checkpointing
- **Voting-based verification** — Belief calibration for quality assessment
- **Semantic communication layer** — 10-100x bandwidth reduction
- **Error logging and pattern flagging** — Recurring issue detection
- **Claude-native prompts** — All commands converted from scripts to prompt-based system

- 2026-02-11: README.md — Major update reflecting all new features: 22 commands (was 20), dream/interpret commands, colony memory inheritance, graveyards, auto-recovery headers, git safety, lint suite, CLAUDE.md-aware command detection, Colony Memory section, restructured Features section
- 2026-02-11: .aether/data/review-2026-02-11.md — Comprehensive daily review report covering 3 colony sessions, 10 achievements, 3 regressions, 5 concerns, 3 debunked concerns, and prioritized recommendations
- 2026-02-12: README.md, CHANGELOG.md — Added /ant:chaos (resilience testing) and /ant:archaeology (git history analysis) commands with build pipeline integration
- 2026-02-12: CHANGELOG.md — added repo-local path migration entry
- 2026-02-12: README.md — Updated to describe repo-local .aether/ architecture; removed global ~/.aether/ runtime references, restructured File Structure section with repo-local paths primary
- 2026-02-13: bin/cli.js, update.md — Added orphan cleanup (syncDirWithCleanup), git dirty-file detection with --force stash, --dry-run preview, hub manifest generation

---

## Colony Work Log

The following entries are automatically generated by the colony during work phases.


## 2026-02-21

### Phase 37 — Plan 02

- **Files:** `aether-utils.sh`, `CHANGELOG.md`
- **Decisions:** Created changelog-append function; Added changelog-collect-plan-data helper
- **What Worked:** Function works correctly
- **Requirements:** LOG-01 addressed

### Phase 37 — Plan 99

- **Files:** `test.md`
- **Decisions:** Test
- **What Worked:** Works
- **Requirements:** TEST-01 addressed

### Phase 0 — Plan 
