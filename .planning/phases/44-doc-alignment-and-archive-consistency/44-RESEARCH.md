# Phase 44: Doc Alignment and Archive Consistency - Research

**Researched:** 2026-04-23
**Domain:** Documentation audit and alignment (markdown files vs Go runtime behavior)
**Confidence:** HIGH

## Summary

This phase is a documentation audit, not a code change phase. The primary work is reading runtime source (Go) and comparing it against markdown docs, then fixing discrepancies directly. The codebase is large (50 commands, 25 agents, 29 skills, 11 doc files, 3 platform guides) but the audit is methodical: compare each doc's claims against the Go source of truth and fix.

Key finding: **there are already concrete discrepancies** identified during research that the planner should task-fix:

1. AGENTS.md version is v1.0.19, should be v1.0.20 (matches CLAUDE.md and version.json)
2. AGENTS.md colony skills count says "10" but there are 11
3. CLAUDE.md and AGENTS.md "Publishing Changes" sections do not mention `aether publish` -- still document only `aether install --package-dir "$PWD"` as the primary path
4. `aether integrity` command (Phase 43) is not documented in ANY markdown file
5. `aether medic --deep` now includes `scanIntegrity` (Phase 43) but no doc mentions it
6. v1.5 roadmap still shows Phase 33 as "Planned" and Phase 35 as "In Progress (1/3 plans)" despite both being complete per the milestone audit
7. v1.5 roadmap shows "v1.5 Runtime Truth Recovery (In Progress)" -- should be "completed"

**Primary recommendation:** Organize the audit into 5 workstreams: (1) version/count metadata fixes, (2) publish/integrity command documentation, (3) medic deep scan documentation update, (4) roadmap milestone status corrections, (5) cross-file consistency sweep.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Comprehensive audit of ALL Aether documentation surfaces
- Full command reference for publish, update, integrity, medic --deep, install
- Narrative + versions check for archive consistency
- Auto-fix mode -- fix inaccurate docs directly

### Claude's Discretion
None -- scope is clear from audit decisions.

### Deferred Ideas (OUT OF SCOPE)
None -- scope is clear from audit decisions.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REL-03 (R064) | Operations guide, publish-update-runbook, and AGENTS.md match actual runtime behavior exactly | Discrepancies catalogued below -- 7 confirmed issues across these docs and others |
| EVD-01 (R066) | Archived release and milestone evidence is internally consistent -- no contradictions after ship | v1.5 roadmap has stale phase statuses contradicting milestone audit; 3 files need updating |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Doc audit (read-only) | Browser / Client (researcher reads files) | -- | No runtime needed, pure file comparison |
| Doc fixes (write) | API / Backend (file writes) | -- | Direct file edits in repo |
| Command behavior verification | API / Backend (Go source) | -- | Go CLI is source of truth |
| Archive consistency check | Browser / Client (researcher reads files) | -- | No runtime needed |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| N/A | N/A | This is a documentation-only phase | No code dependencies |

No external tools or libraries needed. This phase edits markdown files and compares them against Go source code. The entire toolset is: `Read`, `Write`, `Grep`, and `Bash` (for verification).

## Architecture Patterns

### System Architecture Diagram

```
Go Source of Truth (cmd/*.go)
  |
  | [read and compare]
  v
Documentation Surface
  |
  +-- AETHER-OPERATIONS-GUIDE.md  (root-level)
  +-- CLAUDE.md                   (root-level, project instructions)
  +-- AGENTS.md                   (root-level, Codex system prompt)
  +-- .codex/CODEX.md             (Codex developer guide)
  +-- .opencode/OPENCODE.md       (OpenCode rules)
  +-- .aether/docs/publish-update-runbook.md
  +-- .aether/docs/README.md
  +-- .aether/skills/colony/medic/SKILL.md
  +-- .aether/docs/wrapper-runtime-ux-contract.md
  +-- .planning/milestones/v1.5-ROADMAP.md
  +-- .planning/v1.5-MILESTONE-AUDIT.md
  +-- .claude/commands/ant/*.md    (50 files)
  +-- .opencode/commands/ant/*.md  (50 files)
  +-- .claude/agents/ant/*.md     (25 files)
  |
  | [fix discrepancies]
  v
Aligned Documentation
```

### Recommended Approach

Organize into 5 audit workstreams, executed in order:

**Wave 1: Metadata and Count Fixes** (fast, high-confidence)
- Fix version references (AGENTS.md v1.0.19 -> v1.0.20)
- Fix skill counts (AGENTS.md "10 colony" -> "11 colony")
- Fix any other stale numbers

**Wave 2: Command Documentation** (medium effort, core value)
- Add `aether publish` documentation to CLAUDE.md, AGENTS.md, and CODEX.md
- Add `aether integrity` command documentation to operations guide and runbook
- Update runbook `aether medic --deep` section to mention scanIntegrity
- Update medic skill (SKILL.md) to document the deep scan's integrity check

**Wave 3: Platform Guide Updates** (medium effort)
- Update CLAUDE.md "Publishing Changes" section to lead with `aether publish`
- Update AGENTS.md "Publishing Changes" section to lead with `aether publish`
- Update CODEX.md and OPENCODE.md publish sections if they exist
- Ensure all 3 platform guides document the same command set for publish/update/integrity

**Wave 4: Milestone/Archive Consistency** (focused scope)
- Fix v1.5-ROADMAP.md stale phase statuses (33 "Planned" -> complete, 35 "In Progress" -> complete)
- Fix v1.5-ROADMAP.md milestone status "In Progress" -> "completed"
- Cross-check v1.5-MILESTONE-AUDIT.md against roadmap for internal contradictions
- Check verification files (.planning/phases/*-VERIFICATION.md) for consistency

**Wave 5: Cross-file Consistency Sweep** (broad but shallow)
- Grep all .md files for `install --package-dir` and add `aether publish` alongside where appropriate
- Grep all .md files for `medic --deep` and verify the listed capabilities match actual deep scan (wrapper parity, hub publish integrity, ceremony integrity, release integrity)
- Verify command flag references match actual Go cobra flag definitions

### Pattern 1: Source-of-Truth Comparison

**What:** For every doc claim about command behavior, verify against Go source.
**When to use:** Every time a doc states "command X does Y" or "flag --z enables Z".
**Example:**
```
Doc says:  aether publish --channel dev --binary-dest <path>
Go source: publishCmd.Flags().String("channel", "", ...)
           publishCmd.Flags().String("binary-dest", "", ...)
           publishCmd.Flags().Bool("skip-build-binary", false, ...)
           publishCmd.Flags().String("package-dir", "", ...)
           publishCmd.Flags().String("home-dir", "", ...)
Result:    Doc matches source. Also check: does doc mention --package-dir and --home-dir?
```

### Anti-Patterns to Avoid

- **Auditing without fixing:** CONTEXT.md says auto-fix mode. Do not produce a "findings report" -- fix the issues directly.
- **Adding features to docs that don't exist in code:** Every doc addition must have a corresponding Go source implementation. Do not document aspirational behavior.
- **Forgetting the operations guide checklist:** The guide has a verification checklist in Section 10. Each step must actually work when executed.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Doc comparison | Custom diff script | `Grep` with specific patterns | Simple pattern search is sufficient; no tooling needed |
| Version extraction | Parse go.mod | `Read` .aether/version.json | Single source of truth for version |

## Runtime State Inventory

> Not applicable -- this is a documentation-only phase with no rename/refactor/migration scope. No runtime state is affected.

## Common Pitfalls

### Pitfall 1: Cascade Stale References
**What goes wrong:** Fixing a command name in one file but missing it in 5 others that reference it.
**Why it happens:** The same command is documented in CLAUDE.md, AGENTS.md, CODEX.md, OPENCODE.md, operations guide, runbook, and medic skill.
**How to avoid:** After fixing any command reference, grep the entire repo for the old form to catch cascading references.
**Warning signs:** After a fix, `git diff` shows changes in only one file when the command appears in multiple.

### Pitfall 2: Version Drift on Fix Date
**What goes wrong:** Updating version in AGENTS.md to v1.0.20 today, but then a new version ships next week and the doc is stale again.
**Why it happens:** Version numbers appear in multiple files and are easy to forget.
**How to avoid:** All version references should match `.aether/version.json`. After fixing, verify: `grep -rn "v1\.0\.\d\d" --include="*.md" .` and cross-check against version.json.

### Pitfall 3: Over-Documenting Unreleased Behavior
**What goes wrong:** Documenting `aether integrity` as a full reference when the command was just added and may evolve.
**Why it happens:** Phase 43 just added the command; docs may be premature.
**How to avoid:** Document what the command currently does based on the Go source. Note that it was added in v1.0.20 (Phase 43). Keep descriptions factual about current behavior, not aspirational.

### Pitfall 4: Medic Skill Scope Creep
**What goes wrong:** The medic skill is a large file. Trying to rewrite it all introduces risk.
**Why it happens:** The skill documents health checks for many data files; the deep scan additions are small.
**How to avoid:** Only add/fix the sections related to Phases 39-43 changes (scanIntegrity, publish integrity). Do not rewrite existing sections that are already accurate.

## Code Examples

### Verified: `aether publish` Flags (Source of Truth)

```go
// Source: cmd/publish_cmd.go lines 25-31
publishCmd.Flags().String("package-dir", "", "Source directory (default: current directory)")
publishCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
publishCmd.Flags().String("channel", "", "Runtime channel (stable or dev; default: infer from binary/env)")
publishCmd.Flags().String("binary-dest", "", "Destination directory for the built binary")
publishCmd.Flags().Bool("skip-build-binary", false, "Skip go build and use existing binary")
```

The operations guide already documents these flags correctly in Section 5. Other docs need to catch up.

### Verified: `aether integrity` Flags (Source of Truth)

```go
// Source: cmd/integrity_cmd.go lines 37-40
integrityCmd.Flags().Bool("json", false, "Output JSON instead of visual report")
integrityCmd.Flags().String("channel", "", "Override channel (stable or dev)")
integrityCmd.Flags().Bool("source", false, "Force source-repo checks")
```

Checks performed (source: cmd/integrity_cmd.go lines 90-105):
- Source context: sourceVersion, binaryVersion, hubVersion, hubCompanionFiles, downstreamSimulation
- Consumer context: binaryVersion, hubVersion, hubCompanionFiles, downstreamSimulation

### Verified: `aether medic --deep` Deep Scan Components (Source of Truth)

```go
// Source: cmd/medic_scanner.go lines 166-171
if opts.Deep {
    allIssues = append(allIssues, scanWrapperParity(fc)...)
    allIssues = append(allIssues, scanHubPublishIntegrity()...)
    allIssues = append(allIssues, scanCeremonyIntegrity(fc)...)
    allIssues = append(allIssues, scanIntegrity()...)
}
```

Deep scan includes 4 checks: wrapper parity, hub publish integrity, ceremony integrity, and release integrity (scanIntegrity). The runbook only lists 3 (wrapper parity, hub publish completeness, ceremony integrity).

### Verified: `aether update` Flags (Source of Truth)

```go
// Source: cmd/update_cmd.go lines 40-45
updateCmd.Flags().String("channel", "", "Runtime channel to update from (stable or dev; default: infer from binary/env)")
updateCmd.Flags().Bool("download-binary", false, "Also download a binary from GitHub Releases")
updateCmd.Flags().String("binary-version", "", "Binary version to download (default: resolved installed version)")
updateCmd.Flags().Bool("dry-run", false, "Show what would be updated without making changes")
updateCmd.Flags().Bool("force", false, "Overwrite modified companion files and remove stale ones")
```

### Verified: `aether install` Flags (Source of Truth)

```go
// Source: cmd/install_cmd.go lines 57-64
installCmd.Flags().String("package-dir", "", "Override the embedded install assets with a local Aether checkout or package directory")
installCmd.Flags().String("home-dir", "", "User home directory (default: $HOME)")
installCmd.Flags().String("channel", "", "Runtime channel to install (stable or dev; default: infer from binary/env)")
installCmd.Flags().Bool("download-binary", false, "Also download the Go binary from GitHub Releases")
installCmd.Flags().String("binary-dest", "", "Destination directory for binary (default: channel-specific hub bin, or current/local bin when rebuilding from source)")
installCmd.Flags().String("binary-version", "", "Binary version to download (default: current version)")
installCmd.Flags().Bool("skip-build-binary", false, "Skip auto-building the Go binary when installing from an Aether source checkout")
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `aether install --package-dir "$PWD"` as primary publish path | `aether publish` as recommended path, `install --package-dir` preserved for backward compat | Phase 40 (2026-04-23) | Docs still reference old path as primary |
| `aether medic --deep` (3 checks) | `aether medic --deep` (4 checks, including scanIntegrity) | Phase 43 (2026-04-23) | Runbook and medic skill don't mention scanIntegrity |
| No release integrity command | `aether integrity` command | Phase 43 (2026-04-23) | Not documented anywhere in markdown |

**Deprecated/outdated:**
- `aether install --package-dir "$PWD"` as primary publish workflow -- still documented as primary in CLAUDE.md and AGENTS.md. Operations guide correctly shows `aether publish` as recommended.

## Confirmed Discrepancies

### HIGH Confidence (verified by reading both docs and source)

| # | File | Issue | Fix |
|---|------|-------|-----|
| D1 | AGENTS.md:3,23,792 | Version says v1.0.19, should be v1.0.20 | Update 3 locations |
| D2 | AGENTS.md:483 | Colony skills count says "10", actual is 11 | Change "10" to "11" |
| D3 | CLAUDE.md:200, AGENTS.md:216-217 | Publishing sections don't mention `aether publish` | Add `aether publish` as recommended path, keep `install --package-dir` as backward-compat |
| D4 | All .md files | `aether integrity` command not documented | Add to operations guide and runbook |
| D5 | publish-update-runbook.md:264-270 | `aether medic --deep` section lists 3 checks, actually 4 (missing scanIntegrity) | Add "release integrity" to the list |
| D6 | .planning/milestones/v1.5-ROADMAP.md:83,87,107,124 | Phase 33 shows "Planned", Phase 35 shows "In Progress (1/3)" -- both complete per milestone audit | Update checkboxes and status text |
| D7 | .planning/milestones/v1.5-ROADMAP.md:74 | Milestone status says "In Progress", should be "completed" | Update status |
| D8 | .aether/skills/colony/medic/SKILL.md | Does not document scanIntegrity or release integrity as part of `--deep` scan | Add deep scan section |
| D9 | .codex/CODEX.md | Publishing section still references only `aether install --package-dir "$PWD"` | Add `aether publish` |
| D10 | .opencode/OPENCODE.md | Publishing section still references only `aether install --package-dir "$PWD"` | Add `aether publish` |

### MEDIUM Confidence (likely but needs verification during execution)

| # | File | Issue | Fix |
|---|------|-------|-----|
| D11 | AETHER-OPERATIONS-GUIDE.md | Section 13 "two commands you will use most" shows `go run ./cmd/aether install --channel dev` instead of `go run ./cmd/aether publish --channel dev` | Update to use publish |
| D12 | Wrapper commands (.claude/commands/ant/*.md, .opencode/commands/ant/*.md) | May reference outdated install/publish patterns | Grep for `install --package-dir` across all 100 command files |
| D13 | .aether/docs/wrapper-runtime-ux-contract.md | Last updated 2026-04-19, does not mention publish command or integrity checks | Add publish and integrity to runtime surface section |
| D14 | .planning/STATE.md | Current focus says "Phase 43" -- will be stale after Phase 43 completes | Update to Phase 44 (but this is a planning artifact, not distributed docs) |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The operations guide verification checklist (Section 10) still passes as written | Success Criteria | Low -- counts match Go constants (50/50/25/25/29) |
| A2 | v1.5 milestone docs are the only archived evidence to check for EVD-01 | Archive Consistency | Medium -- there may be other verification/summary files from earlier phases that reference v1.5 |
| A3 | `aether publish` is the only new command from Phases 39-43 that needs documentation | Command Docs | Low -- `aether integrity` is the other new command (Phase 43) |
| A4 | Wrapper commands in .claude/commands/ant/ and .opencode/commands/ant/ do NOT reference `aether publish` | Cross-file Sweep | Needs verification during execution (grep during Wave 5) |

## Open Questions

1. **Should the operations guide section 13 be updated to use `aether publish` instead of `go run ./cmd/aether install --channel dev`?**
   - What we know: The operations guide Section 5 correctly documents `aether publish`. Section 13 "two commands you will use most" still uses the old `install` form.
   - What's unclear: Whether this is intentional (backward compat emphasis) or an oversight.
   - Recommendation: Update to `aether publish` for consistency with Section 5.

2. **Should the wrapper-runtime-ux-contract.md be updated?**
   - What we know: It was last updated 2026-04-19, before Phases 40-43.
   - What's unclear: Whether it needs to enumerate publish/integrity in the runtime surface section or if its current abstraction level is fine.
   - Recommendation: Add a brief note that publish and integrity are runtime-owned commands.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- this is a documentation-only phase with no tools, services, or runtimes required beyond the Go test suite for verification).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (built-in) |
| Config file | none |
| Quick run command | `go test ./cmd/ -run TestIntegrity -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REL-03 | AGENTS.md operator flows are accurate | manual-only | N/A -- doc review | N/A |
| REL-03 | Operations guide verification checklist passes | manual-only | Run the find commands from Section 10 | N/A |
| REL-03 | Runbook steps match actual command behavior | manual-only | N/A -- doc review | N/A |
| EVD-01 | Archived v1.5 docs have no internal contradictions | manual-only | N/A -- doc review | N/A |

### Sampling Rate
- **Per task commit:** `go test ./... -count=1` (ensure no code was accidentally broken by doc-only changes)
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- None -- this phase requires no new tests. Existing test infrastructure (2900+ tests) covers all runtime behavior. Documentation accuracy is verified by manual review against source.

## Security Domain

Step skipped: `security_enforcement` is not relevant to a documentation-only phase. No code changes that affect authentication, access control, cryptography, or input validation.

## Sources

### Primary (HIGH confidence)
- `cmd/publish_cmd.go` -- publish command flags and behavior (read in full)
- `cmd/update_cmd.go` -- update command flags and stale publish detection (read in full)
- `cmd/integrity_cmd.go` -- integrity command checks and output (read in full)
- `cmd/medic_scanner.go` -- performHealthScan deep scan composition (read in full)
- `cmd/install_cmd.go` -- install command flags and hub setup (read in full)
- `cmd/runtime_channel.go` -- channel resolution logic (read in full)
- `.aether/version.json` -- current version v1.0.20 (verified)
- `.planning/v1.5-MILESTONE-AUDIT.md` -- milestone audit results (verified)
- `.planning/milestones/v1.5-ROADMAP.md` -- roadmap with stale statuses (verified)
- `CLAUDE.md` -- project instructions (verified)
- `AGENTS.md` -- Codex system prompt (verified)
- `AETHER-OPERATIONS-GUIDE.md` -- operations guide (verified)
- `.aether/docs/publish-update-runbook.md` -- runbook (verified)

### Secondary (MEDIUM confidence)
- `.codex/CODEX.md` -- Codex developer guide (first 100 lines read)
- `.opencode/OPENCODE.md` -- OpenCode rules (first 100 lines read)
- `.aether/docs/wrapper-runtime-ux-contract.md` -- UX contract (read in full)
- `.aether/docs/README.md` -- docs index (verified)
- `.aether/skills/colony/medic/SKILL.md` -- medic skill (grep for relevant patterns)
- `.aether/skills/colony/colony-interaction/SKILL.md` -- colony interaction skill (grep)

### Tertiary (LOW confidence)
- None -- all claims verified against source files read in this session.

## Metadata

**Confidence breakdown:**
- Standard stack: N/A (documentation-only phase)
- Architecture: HIGH -- all discrepancies confirmed by reading both doc and Go source
- Pitfalls: HIGH -- based on concrete examples found during research

**Research date:** 2026-04-23
**Valid until:** 30 days (stable domain -- docs and Go source change slowly)
