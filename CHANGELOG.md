# Changelog

All notable changes to the Aether Colony project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Verified
- **Phase 5: Path Localization Complete** — Full-repo audit confirmed zero actionable `~/.aether/` or `~/.config/opencode/` references in commands, scripts, or CLI. Remaining 4 `$HOME/.aether` references in aether-utils.sh are intentional hub/registry functions for multi-repo management. `generate-commands.sh check` passes (25/25 in sync, SHA-1 checksums verified). `aether-utils.sh` smoke tests pass (help, version, generate-ant-name). All three original goals met: no root access prompts, no cross-repo contamination, no out-of-project file operations.

### Changed
- **Phase 4: Remove global path operations from cli.js** — Removed `~/.aether/` runtime copy logic (RUNTIME_DEST, RUNTIME_SRC, learnings.json creation, execSync import) and `~/.config/opencode/` global install logic (OPENCODE_GLOBAL_COMMANDS_DEST, OPENCODE_GLOBAL_AGENTS_DEST) from install/uninstall commands. Updated help text to reflect new architecture. cli.js now only installs Claude Code slash-commands to `~/.claude/commands/ant/`. Net reduction of 130 lines. (`bin/cli.js`)
- **Phase 3: Document repo-local path architecture** — Updated CHANGELOG.md and documentation references to reflect the completed path localization migration. All runtime paths now use repo-local `.aether/` instead of `~/.aether/`. Phases 1-2 localized command files, agent definitions, system docs, planning docs, shell utilities, and cross-project state functions. Phase 3 updates documentation to reflect the new architecture where running colonies only read/write repo-local `.aether/`, while global install remains functional for command distribution. (`README.md`, `CHANGELOG.md`, `TO-DOS.md`)
- **Phase 2: Localize cross-project state in aether-utils.sh** — Redirected 6 `$HOME/.aether` references to repo-local `$DATA_DIR` paths in learning-promote, learning-inject, error-flag-pattern, error-patterns-check, signature-scan, and signature-match functions. Fixed atomic-write.sh `$HOME/.aether/utils` fallback to use SCRIPT_DIR-based resolution. Updated usage comment headers in all .sh files. Applied to both `.aether/` and `runtime/` copies (verified identical). (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`, `.aether/utils/atomic-write.sh`, `runtime/utils/atomic-write.sh`, `.aether/utils/file-lock.sh`, `runtime/utils/file-lock.sh`)
- **Phase 1: Localize ~/.aether/ path references** — Replaced all `~/.aether/` paths with repo-relative `.aether/` across 50 files: command prompts, agent definitions, system docs, planning docs, and template.yaml. Fixed 3 pre-existing mirror drifts (migrate-state.md, organize.md, plan.md) discovered during verification. (`.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.opencode/agents/*.md`, `.aether/workers.md`, `runtime/workers.md`, `.aether/docs/*.md`, `runtime/docs/*.md`, `.planning/*.md`, `src/commands/_meta/template.yaml`)
- **Phase 2: Upgrade Sync Checking to Content-Aware** — `generate-commands.sh check` now performs SHA-1 checksum comparison (Pass 2) after filename matching (Pass 1), detecting content drift between `.claude/` and `.opencode/` mirrors. Revealed 3 pre-existing drifts previously invisible to filename-only checks. (`bin/generate-commands.sh`)

### Verified
- **Phase 5: Verify Full System Integrity** — Final verification phase confirming all global install locations match repo sources. Full lint suite passed (lint:shell, lint:json, lint:sync). All 4 global locations verified: ~/.claude/commands/ant/ (24 files), ~/.config/opencode/commands/ant/ (24 files), ~/.config/opencode/agents/ (4 files), ~/.aether/ system files. Watcher quality 9/10, Chaos resilience moderate (1 high finding: lint:sync content blind spot, 3 medium, 1 low — all pre-existing infrastructure gaps). Colony goal achieved.

### Fixed
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
