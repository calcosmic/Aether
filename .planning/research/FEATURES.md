# Feature Landscape: v1.1 Bug Fixes & Reliability Improvements

**Domain:** CLI-based AI agent orchestration framework (Aether Colony System)
**Researched:** 2026-02-14
**Confidence:** HIGH (based on documented bugs in TO-DOs.md, CONCERNS.md, and codebase analysis)

## Executive Summary

This research focuses on the v1.1 milestone bug fixes for the Aether Colony System. v1.0 delivered hardened infrastructure; v1.1 addresses critical bugs discovered during real-world usage: phase advancement loops, update system reliability, data loss prevention, and misleading output timing. These are not feature additions but fixes to existing functionality that is broken or dangerous.

## Feature Categories

### Table Stakes (Must-Have Fixes)

Features that are broken and must be fixed for the system to be trustworthy.

| Feature | Why Broken | Complexity | Notes |
|---------|------------|------------|-------|
| **Targeted Git Checkpoints** | Current checkpoint stashes ALL dirty files including user work (1,145 lines nearly lost) | Low | Use explicit allowlist: only stash `.aether/*.md`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `runtime/`, `bin/cli.js`. Never touch user data like TO-DOs.md, `.aether/data/`, `.aether/dreams/`, `.aether/oracle/` |
| **Deterministic Dependency Builds** | No package-lock.json means `npm install` pulls different versions over time | Low | Run `npm install` to generate lockfile, commit it, update CI to use `npm ci` |
| **Unit Tests for Core Sync** | `syncDirWithCleanup`, `hashFileSync`, `generateManifest` in cli.js have no unit tests | Medium | Add AVA tests for hash comparison, dry-run mode, empty directory cleanup, collision handling |
| **Synchronous Worker Spawns** | `run_in_background: true` causes misleading output timing — summary appears before agent notifications | Low | Remove flag from build.md Steps 5.1, 5.4, 5.4.2. Multiple Task calls already run in parallel without it. Remove TaskOutput collection steps |

### Differentiators (Better Than Before)

Improvements that make the system more reliable than the baseline fix.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Phase Advancement Guards** | Prevent AI from repeating same phases by adding explicit state validation | Medium | Add gate in `/ant:continue` to detect if current phase was already completed; verify `current_phase` matches phase being built |
| **Cross-Repo Sync Reliability** | `aether update --all` needs better error handling for dirty repos, network failures, partial updates | Medium | Add retry logic, better dirty file detection, atomic per-repo updates (all-or-nothing per repo) |
| **Version-Aware Update Notifications** | Non-blocking version check at start of `/ant:status`, `/ant:build` to notify when update available | Low | Compare hub version to repo version, show one-line notice if behind |
| **Checkpoint Recovery Tracking** | Track stash operations in local log, verify stash pop after update | Low | Add `.aether/data/stash-log.json` to track created stashes with timestamps, auto-suggest recovery |

### Anti-Features (Things to Deliberately NOT Do When Fixing)

Common mistakes when fixing these bugs.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Broad git stash with `--include-untracked`** | Stashes user work, causes data loss | Use targeted stash with explicit file list, or skip stash and warn user |
| **Automatic stash pop after update** | Could overwrite user changes | Log stash creation, notify user, let them manually pop when ready |
| **Adding more background Task flags** | Exacerbates output timing issues | Use foreground Task calls; they run in parallel without `run_in_background` |
| **Complex checkpoint systems** | Over-engineering a simple problem | Simple allowlist of system files is sufficient; don't build a full backup system |
| **Global version enforcement** | Blocking commands on version mismatch is annoying | Non-blocking notification only; user decides when to update |
| **Removing checkpoint feature entirely** | Checkpoints are useful for system files | Keep checkpoints but scope them correctly to system files only |

## Feature Dependencies

```
Targeted Git Checkpoints
    ├──requires──> System File Allowlist Definition
    │                   └──requires──> Audit of all .aether/ subdirectories
    │
    └──enhances──> Checkpoint Recovery Tracking

Synchronous Worker Spawns
    └──requires──> Remove TaskOutput collection steps
        └──requires──> Update build.md instructions

Cross-Repo Sync Reliability
    ├──requires──> Unit Tests for Core Sync
    │                   └──requires──> Test fixtures for hash comparison
    └──enhances──> Version-Aware Update Notifications

Deterministic Dependency Builds
    └──requires──> package-lock.json generation
        └──requires──> CI update to use npm ci
```

### Dependency Notes

- **Targeted checkpoints require allowlist:** Must define exactly which files are system vs user data before implementing checkpoint fix
- **Sync reliability requires tests:** Cannot safely improve sync without tests verifying behavior
- **Worker spawn fix is isolated:** Can be done independently of other fixes

## Bug Fix Priority Matrix

| Bug Fix | User Impact | Implementation Cost | Priority |
|---------|-------------|---------------------|----------|
| Targeted git checkpoints (data loss) | CRITICAL — could lose hours of work | Low | P0 |
| package-lock.json (determinism) | HIGH — build reproducibility | Low | P0 |
| Unit tests for sync functions | HIGH — prevents regression | Medium | P0 |
| Remove run_in_background (timing) | MEDIUM — UX confusion | Low | P0 |
| Phase advancement guards | MEDIUM — prevents wasted work | Medium | P1 |
| Cross-repo sync reliability | MEDIUM — multi-repo workflows | Medium | P1 |
| Version-aware notifications | LOW — nice to have | Low | P2 |
| Checkpoint recovery tracking | LOW — safety net | Low | P2 |

**Priority key:**
- P0: Must fix before v1.1 release
- P1: Should fix, add if time permits
- P2: Nice to have, future consideration

## v1.1 MVP Definition

### Launch With (v1.1.0)

Minimum fixes needed for trustworthy operation:

1. **Targeted git checkpoints** — Only stash system files, never user data
2. **package-lock.json** — Deterministic builds
3. **Unit tests for sync** — Prevent regression in core functions
4. **Remove run_in_background** — Fix misleading output timing

### Add After Validation (v1.1.x)

Once core fixes are stable:

1. **Phase advancement guards** — Prevent AI loops
2. **Cross-repo sync reliability** — Better error handling

### Future Consideration (v1.2+)

Defer until product is stable:

1. **Version-aware notifications** — Non-blocking update nudges
2. **Checkpoint recovery tracking** — Stash operation logging

## Verification Requirements

Each fix must be verifiable:

| Fix | Verification Method |
|-----|---------------------|
| Targeted checkpoints | Test: Create dirty TO-DOs.md, run build, verify TO-DOs.md NOT stashed |
| package-lock.json | Test: Fresh clone, `npm ci` installs exact same versions |
| Unit tests | Run `npm test`, all new tests pass |
| Remove run_in_background | Visual: Build summary appears after all worker outputs |
| Phase advancement guards | Test: Try to continue already-completed phase, verify blocked |
| Cross-repo sync | Test: Update with dirty repo, verify graceful handling |

## Sources

- TO-DOs.md — Documented bugs with full context (data loss from stash, output ordering)
- CONCERNS.md — Technical debt and security audit
- PITFALLS.md — Domain-specific pitfalls for multi-agent systems
- bin/cli.js — Source code analysis of sync functions
- .claude/commands/ant/build.md — Worker spawn patterns

**Confidence Assessment:**

| Area | Level | Reason |
|------|-------|--------|
| Data loss bug | HIGH | Documented in TO-DOs, nearly lost user work |
| Output timing | HIGH | Documented in TO-DOs and CONCERNS |
| Sync testing gap | HIGH | Explicitly noted in CONCERNS |
| Phase loops | MEDIUM | Inferred from v1.1 goals, less documentation |

---
*Feature research for: v1.1 Bug Fixes*
*Researched: 2026-02-14*
