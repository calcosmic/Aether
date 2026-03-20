# Phase 26: File Audit & Delete Dead Files - Research

**Researched:** 2026-02-20
**Domain:** Repository hygiene — audit, categorize, and delete dead files across a multi-directory repo
**Confidence:** HIGH (all findings based on direct file inspection, no external tooling needed)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Deletion aggressiveness**
- Aggressive approach — delete everything not actively used for the colony to run or for current development
- Old planning docs (design plans, handoff docs, implementation plans from v1.0-v1.3) should be deleted outright — git history preserves them
- `.aether/docs/` — Claude's discretion on which docs serve a clear purpose (user-facing or dev); delete the rest

**Archive vs delete policy**
- Colony-related artifacts go to `.aether/archive/` before deletion — safety net beyond git history
- Truly dead files (empty dirs, debugging artifacts, dated handoffs) are deleted outright — no archive
- `.aether/archive/` keeps its existing content (old model-routing research) and receives new archived items

**Borderline file handling**
- `.planning/phases/` — delete v1.0-v1.2 phase directories, keep v1.3 and v1.4 phase directories
- `TO-DOS.md` — clean it up, remove completed/obsolete items, keep only what's still relevant
- Audit covers EVERYTHING: `.aether/`, `.claude/`, `.opencode/`, `docs/`, repo root
- Repo root files audited too — if it's not serving a purpose, flag it

**Safety verification**
- Run full test suite (446 tests) AND `npm pack --dry-run` after deletions
- Small batches — one commit per logical category of deletions (easier to revert)
- Spot-check 3-5 key slash commands after cleanup to verify they still work
- If deletion breaks something: Claude decides per-case whether to fix the reference or revert

### Claude's Discretion

- Which `.aether/docs/` files serve a clear purpose vs should be deleted
- Per-file decisions in `docs/` directory (audit each, delete what's clearly dead)
- Which root-level files to flag for removal
- Grouping of deletions into logical commit categories
- Per-case judgment when deletions break references

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLEAN-01 | Audit every file in repo root and `.aether/` root — mark each as KEEP, ARCHIVE, or DELETE | Full file inventory captured in Findings section; per-file verdicts below |
| CLEAN-02 | Remove `.aether/agents/` (dead duplicate) | NOT PRESENT — already cleaned up; directory does not exist |
| CLEAN-03 | Remove `.aether/commands/` (dead duplicate) | NOT PRESENT — already cleaned up; directory does not exist |
| CLEAN-04 | Remove `.aether/docs/` subdirectory duplicates | Confirmed: `reference/` and `implementation/` subdirs contain identical or stale copies of root-level docs; both subdirs are DELETE |
| CLEAN-05 | Archive remaining `.planning/phases/` completed phases to docs/plans/ | Interpretation: delete v1.0-v1.2 from `.planning/milestones/`; v1.3 phases (20-25) stay in `.planning/phases/` as they are gitignored local-only files; decision is user-context confirmed |
| CLEAN-06 | Clean up `docs/plans/` — consolidate or remove redundant planning docs | 10 planning docs in `docs/plans/`; research-backed per-file verdicts below |
| CLEAN-07 | Audit `.opencode/` and `.claude/` — remove any dead duplicates | Confirmed: `.opencode/agents/workers.md` is a stale near-duplicate of `.aether/workers.md`; `.claude/commands/gsd/new-project.md.bak` is dead; per-directory verdicts below |
| CLEAN-08 | Verify all slash commands still work after cleanup | Test plan: run `npm run lint:sync` + spot-check 3-5 commands after each deletion batch |
| CLEAN-09 | Verify `npm install -g .` still packages correctly | Test plan: run `npm pack --dry-run` after each batch; `bin/validate-package.sh` enforces REQUIRED_FILES list |
| CLEAN-10 | Update README to reflect new simplified structure | README describes v4.0 architecture; after deletion, update any directory references that no longer exist |
</phase_requirements>

---

## Summary

The repo has accumulated significant dead weight across three categories: (1) old planning and spec docs distributed in the npm package but serving no user purpose, (2) duplicate files across `docs/` subdirectories where identical content lives in two or three places, and (3) dev artifacts in `.aether/` root that were either trial drafts or old-era handoff files.

The good news: `CLEAN-02` and `CLEAN-03` are already done — `.aether/agents/` and `.aether/commands/` do not exist. The bad news: `npm pack --dry-run` reveals 35+ files under `.aether/docs/` being published, including 73KB v2.0 master spec docs, duplicate reference subdirectories, and a stale `QUEEN.md` from the Aether dev repo itself being shipped to users.

The most impactful single action is purging the `.aether/docs/` directory aggressively — removing old Aether 2.0 planning specs (~145KB), stale duplicate subdirectories (`reference/` and `implementation/`), and docs that serve no user-facing purpose. The validation constraint is `bin/validate-package.sh`, which requires `docs/README.md` to exist — so the `docs/` directory itself must survive with a valid README.

**Primary recommendation:** Delete in five logical batches (one commit each): (1) `.aether/` root dead files, (2) `.aether/docs/` dead docs + duplicate subdirs, (3) repo root dead files, (4) `.planning/milestones/` v1.0-v1.2 phases, (5) `docs/plans/` cleanup. Verify with `npm pack --dry-run` and `npm test` between each batch.

---

## Standard Stack

No external tooling needed. This phase uses only:

| Tool | Version | Purpose |
|------|---------|---------|
| `rm -rf` | system | Delete files and directories |
| `git rm` | system | Stage deletions for commit |
| `npm pack --dry-run` | 4.0.0 | Verify package contents after each batch |
| `npm test` | project scripts | Run unit + bash tests (446 total) |
| `npm run lint:sync` | project scripts | Verify slash command sync integrity |
| `bin/validate-package.sh` | v4.0 | Validates REQUIRED_FILES exist before packaging |

---

## Architecture Patterns

### The Distribution Graph

Understanding what gets published is critical for safe deletion. Three separate include/exclude mechanisms apply:

```
.npmignore (root)         → Governs everything EXCEPT files in the `files` field of package.json
.aether/.npmignore        → Governs what inside .aether/ gets excluded (even though .aether/ is in `files`)
package.json "files"      → Explicit allowlist: bin/, .claude/commands/ant/, .opencode/commands/ant/,
                            .opencode/agents/, .opencode/opencode.json, .aether/, README.md, LICENSE,
                            DISCLAIMER.md, CHANGELOG.md
```

**Key insight:** `.aether/` is entirely in the npm package EXCEPT what `.aether/.npmignore` excludes. Currently `.aether/.npmignore` excludes: `data/`, `dreams/`, `oracle/`, `checkpoints/`, `locks/`, `temp/`, `archive/`, `chambers/`, `examples/`, `__pycache__/`, and named files (HANDOFF.md, PHASE-0-ANALYSIS.md, etc.). Everything else in `.aether/` ships to users.

### Validate-Package.sh REQUIRED_FILES

These files MUST EXIST or `npm install -g .` will fail:

```
.aether/aether-utils.sh
.aether/workers.md
.aether/CONTEXT.md
.aether/model-profiles.yaml
.aether/docs/README.md              ← docs/ directory required, but not specific docs
.aether/utils/atomic-write.sh
.aether/utils/error-handler.sh
.aether/utils/file-lock.sh
.aether/templates/QUEEN.md.template
.aether/templates/colony-state.template.json
.aether/templates/constraints.template.json
.aether/templates/colony-state-reset.jq.template
.aether/templates/crowned-anthill.template.md
.aether/templates/handoff.template.md
.aether/templates/handoff-build-error.template.md
.aether/templates/handoff-build-success.template.md
.aether/rules/aether-colony.md
```

**Implication:** `.aether/docs/README.md` must survive. Individual docs files under it can be deleted freely — only the README itself is required.

### aether-utils.sh Update Allowlist

When users run `aether update`, only files on this allowlist get synced to their repo:

```
coding-standards.md, debugging.md, DISCIPLINES.md, learning.md, planning.md,
QUEEN_ANT_ARCHITECTURE.md, tdd.md, verification-loop.md, verification.md, workers.md,
docs/constraints.md, docs/pathogen-schema-example.json, docs/pathogen-schema.md,
docs/pheromones.md, docs/progressive-disclosure.md,
utils/atomic-write.sh, utils/colorize-log.sh, utils/file-lock.sh, utils/watch-spawn-tree.sh
```

Files NOT on this list are packaged into npm but NOT pushed to target repos. This means many `.aether/docs/` files are shipped in the npm tarball but never used — pure dead weight.

---

## Complete File Audit by Directory

### Repo Root (KEEP/DELETE per file)

| File | Verdict | Reason |
|------|---------|--------|
| `README.md` | KEEP | User-facing docs, in npm package |
| `CLAUDE.md` | KEEP | Active project instructions |
| `CHANGELOG.md` | KEEP | In npm package, meaningful history |
| `DISCLAIMER.md` | KEEP | In npm package, legal |
| `LICENSE` | KEEP | In npm package, required |
| `package.json` | KEEP | Required |
| `package-lock.json` | KEEP | Required |
| `.gitignore` | KEEP | Required |
| `.npmignore` | KEEP | Required |
| `TO-DOS.md` | CLEAN | Remove completed/obsolete items (multiple completed bugs, old features) |
| `RUNTIME UPDATE ARCHITECTURE.md` | KEEP | Referenced in CLAUDE.md; documents v4.0 architecture accurately |
| `DISCLAIMER.md` | KEEP | In npm package |
| `"Aether Notes"` | DELETE | macOS alias/bookmark binary file; no code purpose |
| `aether-logo.png` | DELETE | 1.2MB image file; not in npm package (excluded by root .npmignore); no code purpose |
| `logo_block.txt` | DELETE | ASCII art; excluded by .npmignore; not referenced in active code |
| `logo_block_color.txt` | DELETE | ASCII art; excluded by .npmignore; not referenced in active code |
| `planning/` | DELETE | Empty directory (root-level `planning/`, not `.planning/`) — contains only `_meta/` with a README |
| `.cursor/` | DELETE | Cursor IDE config; contains only `worktrees.json`; not for git |
| `.worktrees/` | DELETE | Worktree tracking data; excluded by .gitignore; contains stale data |

### `.aether/` Root (KEEP/DELETE per file)

| File/Dir | Verdict | Reason |
|----------|---------|--------|
| `aether-utils.sh` | KEEP | Core system file; REQUIRED |
| `workers.md` | KEEP | Core system file; REQUIRED |
| `CONTEXT.md` | KEEP | REQUIRED by validate-package.sh; installed as user context |
| `model-profiles.yaml` | KEEP | REQUIRED; model routing config |
| `coding-standards.md` | KEEP | Distributed to target repos via update allowlist |
| `debugging.md` | KEEP | Distributed to target repos via update allowlist |
| `DISCIPLINES.md` | KEEP | Distributed to target repos via update allowlist |
| `learning.md` | KEEP | Distributed to target repos via update allowlist |
| `tdd.md` | KEEP | Distributed to target repos via update allowlist |
| `verification.md` | KEEP | Distributed to target repos via update allowlist |
| `verification-loop.md` | KEEP | Distributed to target repos via update allowlist |
| `QUEEN_ANT_ARCHITECTURE.md` | KEEP | Distributed to target repos via update allowlist |
| `workers-new-castes.md` | DELETE | Old draft of caste expansion; superseded by current `workers.md`; referenced only in cli.js migration path (one-time hub migration, safe to remove from file) |
| `recover.sh` | DELETE | Old recovery script; not in REQUIRED_FILES; not in update allowlist; likely from pre-v4.0 era |
| `HANDOFF.md` | KEEP (local) | In `.npmignore`, stays local; active session handoff |
| `HANDOFF_AETHER_DEV_2026-02-15.md` | DELETE | Dated handoff doc; in `.npmignore`; stale |
| `PHASE-0-ANALYSIS.md` | DELETE | Old analysis; in `.npmignore`; stale |
| `RESEARCH-SHARED-DATA.md` | DELETE | Old research artifact; in `.npmignore`; stale |
| `diagnose-self-reference.md` | DELETE | Debugging artifact; in `.npmignore`; stale |
| `DIAGNOSIS_PROMPT.md` | DELETE | Debugging artifact; in `.npmignore`; stale |
| `pheromone_system.py` | DELETE | Python prototype; superseded by shell implementation; in `.npmignore` |
| `semantic_layer.py` | DELETE | Python prototype; superseded by shell implementation; in `.npmignore` |
| `__pycache__/` | DELETE | Python bytecode; excluded by `.npmignore` and `.gitignore`; no purpose |
| `manifest.json` | KEEP (local) | In `.npmignore`; runtime-generated, system uses it |
| `ledger.jsonl` | KEEP (local) | In `.npmignore`; runtime activity ledger |
| `registry.json` | KEEP (local) | In `.npmignore`; runtime colony registry |
| `version.json` | KEEP (local) | In `.npmignore`; runtime version tracking |
| `data/` | KEEP (local) | Colony state; NEVER TOUCH |
| `dreams/` | KEEP (local) | Dream journal; NEVER TOUCH |
| `oracle/` | KEEP (local) | Oracle research; NEVER TOUCH |
| `archive/` | KEEP (local) | Excluded from npm; destination for colony artifacts |
| `chambers/` | KEEP (local) | Sealed colony chambers; NEVER TOUCH |
| `checkpoints/` | KEEP (local) | Build checkpoints; excluded from npm |
| `locks/` | KEEP (local) | Runtime locks; excluded from npm |
| `temp/` | KEEP (local) | Temp files; excluded from npm |
| `docs/` | AUDIT (see below) | Required directory; specific files audited below |
| `utils/` | KEEP | All shell utilities actively used |
| `templates/` | KEEP | All templates REQUIRED by validate-package.sh |
| `schemas/` | KEEP | XSD schemas for XML system; packaged in npm |
| `exchange/` | KEEP | XML exchange scripts; packaged in npm |
| `rules/` | KEEP | `aether-colony.md` REQUIRED by validate-package.sh |
| `examples/` | DELETE | Excluded by `.npmignore`; contains a single stale `worker-priming.xml` example |

### `.aether/docs/` — Full Per-File Audit

Currently 35+ files are published in the npm package from this directory. Most serve no user-facing purpose.

**Files to KEEP (user-facing or actively referenced):**

| File | Reason to Keep |
|------|----------------|
| `README.md` | REQUIRED by validate-package.sh |
| `caste-system.md` | Referenced from workers.md; actively updated (Phase 25) |
| `pheromones.md` | In update allowlist — distributed to target repos |
| `constraints.md` | In update allowlist — distributed to target repos |
| `pathogen-schema.md` | In update allowlist — distributed to target repos |
| `pathogen-schema-example.json` | In update allowlist — distributed to target repos |
| `progressive-disclosure.md` | In update allowlist — distributed to target repos |
| `known-issues.md` | Referenced in `.claude/rules/aether-development.md`; actively maintained |
| `implementation-learnings.md` | Referenced in `.claude/rules/aether-development.md`; extracted dev findings |
| `error-codes.md` | Referenced in known-issues.md; actively maintained (Phase 17) |
| `QUEEN-SYSTEM.md` | Describes wisdom promotion system; referenced in oracle analysis reports |
| `queen-commands.md` | Used by queen command documentation |
| `QUEEN.md` | Generated QUEEN wisdom file; referenced by aether-utils.sh (queen-read, queen-promote commands use `.aether/docs/QUEEN.md` path in target repos; this instance is the Aether dev repo's own QUEEN.md) |

**Files to DELETE (dead weight in npm package):**

| File | Reason to Delete |
|------|-----------------|
| `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md` | 73KB Aether v2.0 planning spec; never referenced by any command or script; ships to every user unnecessarily |
| `AETHER-2.0-IMPLEMENTATION-PLAN.md` | 36KB old roadmap; superseded; not referenced |
| `aether_2.0_complete_implementation_-_100_phase_master_plan_6d0247f5.plan.md` | 35KB old AI-generated plan; clearly stale filename; not referenced |
| `PHEROMONE-INJECTION.md` | Per README.md: "consolidated into master spec"; not referenced by any command |
| `PHEROMONE-INTEGRATION.md` | Per README.md: "consolidated into master spec"; not referenced |
| `PHEROMONE-SYSTEM-DESIGN.md` | Per README.md: "consolidated into master spec"; not referenced |
| `VISUAL-OUTPUT-SPEC.md` | Visual UX spec from early development; not referenced by active commands |
| `RECOVERY-PLAN.md` | 311-line recovery plan from v1.x debugging era; not referenced |
| `biological-reference.md` | Biological ant research reference; identical to `reference/biological-reference.md`; not in update allowlist |
| `codebase-review.md` | Oracle analysis from 2026-02-14; internal dev artifact; not referenced by any command |
| `planning-discipline.md` | Dev planning guide for Aether dev; not for users of target repos |
| `namespace.md` | Namespace isolation doc; identical to `reference/namespace.md`; not in update allowlist |
| `command-sync.md` | Command sync guide; slightly different from `reference/command-sync.md`; internal dev doc |
| `architecture/` | Contains only `MULTI-COLONY-ARCHITECTURE.md` (multi-colony design from v2.0 era; not implemented); DELETE entire subdir |
| `reference/` | Entire subdirectory: `biological-reference.md`, `command-sync.md`, `constraints.md`, `namespace.md`, `progressive-disclosure.md` — ALL are identical or near-identical to root-level files; pure duplication; DELETE entire subdir |
| `implementation/` | Entire subdirectory: `known-issues.md` (older subset of root-level known-issues.md), `pheromones.md` (identical to root-level), `pathogen-schema.md` (identical), `pathogen-schema-example.json` (identical); DELETE entire subdir |

**Update README.md** in `.aether/docs/` to reflect the simplified structure after deletion.

### `.claude/` Directory

| File/Dir | Verdict | Reason |
|----------|---------|--------|
| `agents/` | KEEP | GSD agent definitions (11 agents); actively used |
| `commands/ant/` | KEEP | 34 active slash commands |
| `commands/gsd/` | KEEP | 31 active GSD commands |
| `commands/gsd/new-project.md.bak` | DELETE | Backup file; the `.md` version is the live one |
| `get-shit-done/` | KEEP | GSD framework; actively used |
| `hooks/` | KEEP | Pre-commit hooks; referenced in `.claude/settings.json` |
| `rules/` | KEEP | All rule files actively loaded via CLAUDE.md |
| `settings.json` | KEEP | Hook configuration |
| `settings.local.json` | KEEP | Local model config |
| `package.json` | KEEP | GSD framework dependencies |
| `gsd-file-manifest.json` | KEEP | GSD framework manifest |

### `.opencode/` Directory

| File/Dir | Verdict | Reason |
|----------|---------|--------|
| `agents/` (all *.md except workers.md) | KEEP | 22 active agent definitions |
| `agents/workers.md` | DELETE | Stale near-duplicate of `.aether/workers.md`; missing Phase 25 Architect-to-Keeper merge note and caste-system.md reference; the npm package ships `.aether/workers.md` to target repos |
| `commands/ant/` | KEEP | 34 active OpenCode slash commands |
| `OPENCODE.md` | KEEP | OpenCode-specific configuration |
| `opencode.json` | KEEP | In npm package |
| `package.json` | KEEP | OpenCode package config |
| `bun.lock` | KEEP | Lock file |
| `node_modules/` | KEEP | Runtime dependency |

### `docs/` Directory (repo root)

| File/Dir | Verdict | Reason |
|----------|---------|--------|
| `docs/plans/` | AUDIT (see below) | Design plans from v1.1-v1.2 era |
| `docs/worktree-salvage/` | DELETE | Salvage artifacts from worktree experiment; contains old agent drafts and docs already in `.aether/docs/`; DELETE entire directory |

**`docs/plans/` per-file audit:**

| File | Verdict | Reason |
|------|---------|--------|
| `2026-02-17-hub-system-directory-migration.md` | DELETE | Hub migration plan from v4.0; completed and shipped; git preserves it |
| `2026-02-17-pheromone-consumption-design.md` | DELETE | Design spec; work never started (deferred in TO-DOS); git preserves it |
| `2026-02-17-pheromone-consumption-plan.md` | DELETE | Companion to above; DELETE |
| `2026-02-18-agent-definition-architecture-plan.md` | DELETE | 39KB plan for Phase 22; Phase 22 complete; git preserves it |
| `2026-02-18-agent-improvement-synthesis.md` | DELETE | Research synthesis for Phase 22; complete; git preserves it |
| `2026-02-18-colony-team-structure-analysis.md` | DELETE | 33KB analysis; same content exists in `.planning/colony-team-analysis.md`; complete |
| `2026-02-18-distribution-chain-audit.md` | DELETE | Distribution audit for v4.0; complete and shipped; git preserves it |
| `2026-02-18-template-architecture-plan.md` | DELETE | 26KB plan for Phase 21; Phase 21 complete; git preserves it |
| `2026-02-18-template-improvement-synthesis.md` | DELETE | Research synthesis for Phase 21; complete; git preserves it |
| `2026-02-18-template-schema-system-design.md` | DELETE | 26KB design for Phase 21; complete; git preserves it |

**Verdict: DELETE all 10 files from `docs/plans/`.** All are implementation plans for phases 20-22, all of which shipped. The `.planning/phases/` directories preserve the actual artifacts. Git history preserves everything else. With all 10 deleted, `docs/plans/` itself becomes empty and should also be deleted (or `docs/` becomes empty and can be deleted if worktree-salvage is also gone).

### `.planning/` Directory

`.planning/` is gitignored (local-only). Per user decision:

| Item | Verdict | Reason |
|------|---------|--------|
| `.planning/phases/20-25` | KEEP | v1.3 phases; local-only; still valuable for reference |
| `.planning/phases/26-file-audit` | KEEP | Current active phase |
| `.planning/milestones/v1.0-phases/` | DELETE | v1.0 phases (9 dirs); user decided to delete v1.0-v1.2 |
| `.planning/milestones/v1.1-phases/` | DELETE | v1.1 phases (4 dirs); DELETE |
| `.planning/milestones/v1.2-phases/` | DELETE | v1.2 phases (6 dirs); DELETE |
| `.planning/milestones/v1.0-ROADMAP.md` | DELETE | Archived v1.0 roadmap; DELETE |
| `.planning/milestones/v1.0-REQUIREMENTS.md` | DELETE | Archived v1.0 requirements; DELETE |
| `.planning/milestones/v1.1-MILESTONE-AUDIT.md` | DELETE | Archived v1.1 audit; DELETE |
| `.planning/milestones/v1.1-ROADMAP.md` | DELETE | Archived v1.1 roadmap; DELETE |
| `.planning/milestones/v1.1-REQUIREMENTS.md` | DELETE | Archived v1.1 requirements; DELETE |
| `.planning/milestones/v1.2-MILESTONE-AUDIT.md` | DELETE | Archived v1.2 audit; DELETE |
| `.planning/milestones/v1.2-ROADMAP.md` | DELETE | Archived v1.2 roadmap; DELETE |
| `.planning/milestones/v1.2-REQUIREMENTS.md` | DELETE | Archived v1.2 requirements; DELETE |
| `.planning/milestones/v1.4-phases/` | KEEP (empty) | v1.4 is next milestone; keep for future work |
| `.planning/milestones/v1.4-REQUIREMENTS.md` | KEEP | Active v1.4 requirements |
| `.planning/milestones/v1.4-ROADMAP.md` | KEEP | Active v1.4 roadmap |
| `.planning/milestones/MILESTONES.md` | KEEP | Active milestone tracking |
| `.planning/codebase/` | KEEP | Codebase analysis docs; useful reference |
| `.planning/research/` | KEEP | v1.3 research; recent and useful |
| `.planning/colony-team-analysis.md` | DELETE | Duplicate of `docs/plans/2026-02-18-colony-team-structure-analysis.md` |
| `.planning/.continue-here.md` | KEEP | Session continuation marker |
| `.planning/config.json` | KEEP | GSD config |
| `.planning/PROJECT.md` | KEEP | Active project definition |
| `.planning/ROADMAP.md` | KEEP | Active roadmap |
| `.planning/STATE.md` | KEEP | Active state tracking |
| `.planning/v1.3-MILESTONE-AUDIT.md` | KEEP | Recent audit; useful reference |

**Note:** Since `.planning/` is gitignored, deletions here are local-only cleanup — no git operations needed for this directory.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Safe deletion | Custom safety checks | Direct `rm -rf` + `git rm` — git history is the safety net; `.aether/archive/` for colony artifacts |
| Package verification | Manual file listing | `npm pack --dry-run` — shows exactly what ships |
| Command sync check | Manual comparison | `npm run lint:sync` — already built |
| Validation | Custom scripts | `bin/validate-package.sh` — already enforces REQUIRED_FILES |

**Key insight:** The safety infrastructure already exists. The only risk is deleting a REQUIRED_FILE — and that list is explicit and small (17 files).

---

## Common Pitfalls

### Pitfall 1: Deleting a File That's in REQUIRED_FILES
**What goes wrong:** `npm install -g .` fails with "Required file missing from .aether/"
**Why it happens:** validate-package.sh enforces 17 specific files must exist
**How to avoid:** Cross-reference every deletion against the REQUIRED_FILES list before executing
**Warning signs:** `npm pack --dry-run` completes but `npm install -g .` errors on preinstall hook

### Pitfall 2: Deleting .aether/docs/README.md
**What goes wrong:** validate-package.sh fails — `docs/README.md` is in REQUIRED_FILES
**Why it happens:** The README is the only specific doc file that's required
**How to avoid:** The README must survive; update it to reflect the new simplified structure

### Pitfall 3: Deleting .aether/docs/ Subdir Files That Are In Update Allowlist
**What goes wrong:** `aether update` in target repos stops syncing expected files
**Why it happens:** aether-utils.sh update command copies from an explicit allowlist
**Files in allowlist under docs/:** `docs/constraints.md`, `docs/pathogen-schema-example.json`, `docs/pathogen-schema.md`, `docs/pheromones.md`, `docs/progressive-disclosure.md`
**How to avoid:** These 5 files must survive in `.aether/docs/` root (not subdirectories)

### Pitfall 4: Workers.md Divergence After Deleting .opencode/agents/workers.md
**What goes wrong:** OpenCode target repos may expect a workers.md in agents/
**Why it happens:** `.opencode/agents/workers.md` is in the npm package `files` list via `.opencode/agents/`
**Resolution:** The authoritative workers.md is `.aether/workers.md` (distributed via hub). The `.opencode/agents/workers.md` is a stale duplicate that was 1,034 lines vs `.aether/workers.md` at 765 lines — it's an old version. Safe to delete; OpenCode agents are defined in the individual agent files, not workers.md.

### Pitfall 5: Test Failures That Pre-Exist Phase 26
**What goes wrong:** Seeing test failures and thinking cleanup broke something
**Why it happens:** Current baseline has 2 pre-existing unit test failures (validate-state tests) and 2-3 bash test failures (flag-add ERR-04 tests) — these existed before this phase
**How to avoid:** Document baseline before starting; only block on NEW failures

---

## Code Examples

### Deletion Pattern (per batch)
```bash
# Step 1: Delete files
rm -rf /path/to/dead/dir
git rm -r /path/to/dead/dir  # For tracked files

# Step 2: Verify package integrity
npm pack --dry-run 2>&1 | grep -E "aether/docs|ERROR|warning"

# Step 3: Run tests (don't block on pre-existing failures)
npm test

# Step 4: Commit with category label
git add -A
git commit -m "chore: delete dead [category] files"
```

### Verify Nothing Broke In Slash Commands
```bash
# Verify command sync still passes
npm run lint:sync

# Spot-check a key command still has its content
head -5 .claude/commands/ant/build.md
head -5 .claude/commands/ant/init.md
head -5 .claude/commands/ant/seal.md
```

---

## Recommended Deletion Batches

Based on the audit, six logical commit groups:

| Batch | Scope | Files Affected | Risk |
|-------|-------|---------------|------|
| 1 | Repo root dead files | `"Aether Notes"`, `aether-logo.png`, `logo_block.txt`, `logo_block_color.txt`, `planning/` (empty root dir), `.cursor/` | LOW — all excluded from npm |
| 2 | `.aether/` root dead files | `workers-new-castes.md`, `recover.sh`, `HANDOFF_AETHER_DEV_*.md`, `PHASE-0-ANALYSIS.md`, `RESEARCH-SHARED-DATA.md`, `diagnose-self-reference.md`, `DIAGNOSIS_PROMPT.md`, `pheromone_system.py`, `semantic_layer.py`, `__pycache__/`, `examples/` | LOW — all in `.npmignore` or non-required |
| 3 | `.aether/docs/` big dead docs | `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md`, `AETHER-2.0-IMPLEMENTATION-PLAN.md`, `aether_2.0_*plan*.md`, `PHEROMONE-*.md` (3 files), `VISUAL-OUTPUT-SPEC.md`, `RECOVERY-PLAN.md`, `biological-reference.md`, `codebase-review.md`, `planning-discipline.md`, `namespace.md`, `command-sync.md`; DELETE entire `reference/`, `implementation/`, `architecture/` subdirs | MEDIUM — verify npm pack + lint:sync after |
| 4 | `.claude/` and `.opencode/` dead files | `.claude/commands/gsd/new-project.md.bak`, `.opencode/agents/workers.md` | LOW — .bak is dead; workers.md is stale duplicate |
| 5 | `docs/` repo root cleanup | DELETE all 10 files from `docs/plans/`, DELETE `docs/worktree-salvage/`, DELETE `docs/plans/` dir, DELETE `docs/` if now empty | LOW — planning docs; git preserves all |
| 6 | `TO-DOS.md` cleanup | Remove completed items: build checkpoint bug (fixed Phase 14), session freshness (complete), distribute simplification (shipped as v4.0) | LOW — content edit, not deletion |

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `.aether/agents/` for agent defs | `.opencode/agents/` only | v4.0 | `.aether/agents/` already removed (CLEAN-02 already done) |
| `.aether/commands/` for commands | `.claude/commands/ant/` + `.opencode/commands/ant/` | v4.0 | `.aether/commands/` already removed (CLEAN-03 already done) |
| `runtime/` staging for npm | Direct `.aether/` packaging | v4.0 (Phase 20) | Removes a whole category of stale files; validated by `validate-package.sh` |

---

## Open Questions

1. **Should `.aether/docs/README.md` be updated as part of this phase?**
   - What we know: It currently references AETHER-2.0 docs, PHEROMONE-*.md files, and a full directory structure that will be deleted
   - What's unclear: Whether updating the README is in scope for this phase or should be a quick follow-up
   - Recommendation: Update it as the last step of batch 3 — required by validate-package.sh so it must stay accurate

2. **Does deleting `.opencode/agents/workers.md` break anything in OpenCode?**
   - What we know: OpenCode uses `.opencode/agents/` for agent discovery; `workers.md` is generic worker role docs, not an agent definition; individual agent files (`aether-queen.md`, `aether-builder.md` etc.) are what OpenCode actually loads
   - What's unclear: Whether any OpenCode command explicitly references `workers.md` by name
   - Recommendation: Grep for `workers.md` in `.opencode/` before deleting; observation is it's not referenced by any agent file

3. **`TO-DOS.md` scope — which items are truly complete?**
   - What we know: "BUG: Build checkpoint stashes user data" — FIXED (Phase 14, checkpoint-allowlist); Session freshness — COMPLETE (all 9 phases); Distribution simplification — SHIPPED (v4.0)
   - What's unclear: Many TO-DO items are deferred features (XML integration, model routing verification, pheromone evolution) — these should stay
   - Recommendation: Remove only items explicitly marked as fixed or shipped; leave future/deferred items

---

## Sources

### Primary (HIGH confidence)
- Direct file inspection — all file contents verified by reading; no inference
- `npm pack --dry-run` output — authoritative list of 206 files currently published
- `bin/validate-package.sh` — authoritative REQUIRED_FILES list
- `aether-utils.sh` lines 2219-2238 — authoritative update allowlist
- `package.json` `files` field — authoritative include list

### Secondary (MEDIUM confidence)
- grep searches across `.claude/`, `.opencode/`, `aether-utils.sh` for file references — some references may exist in files not searched (e.g., deep in e2e test scripts)

---

## Metadata

**Confidence breakdown:**
- File inventory: HIGH — all directories directly inspected
- Delete verdicts: HIGH — each verified against distribution graph (npm package + update allowlist)
- Pitfalls: HIGH — derived from actual validate-package.sh code and .npmignore logic
- Batch sequencing: MEDIUM — ordering is a judgment call; any order works, this order minimizes risk

**Research date:** 2026-02-20
**Valid until:** This phase completes — structure will change after deletions
