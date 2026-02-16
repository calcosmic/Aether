# Documentation Analysis Report

**Date:** 2026-02-16
**Analyst:** Oracle Agent
**Scope:** Complete Aether documentation audit

---

## Executive Summary

This analysis catalogs 1,153 markdown files (excluding node_modules) across the Aether codebase. The documentation is extensive but suffers from significant duplication, stale handoff documents, and organizational fragmentation. The runtime/ directory duplicates .aether/ content, and multiple handoff documents from completed work remain in the repository.

**Key Findings:**
- 1,153 total markdown files (excluding node_modules)
- 528 node_modules markdown files (dependency documentation)
- 29 runtime/ docs that duplicate .aether/ source files
- 8 stale handoff documents from completed work
- 66 command files duplicated between Claude and OpenCode
- 25 agent definitions duplicated between .aether/agents and .opencode/agents

---

## File Count by Category

| Category | Count | Notes |
|----------|-------|-------|
| **Core system (.aether/*.md)** | 17 | Source of truth for system docs |
| **Core docs (.aether/docs/)** | 32 | Implementation guides, specs, reference |
| **Commands (.aether/commands/)** | 66 | 33 Claude + 33 OpenCode command definitions |
| **Agents (.aether/agents/)** | 25 | Worker/agent role definitions |
| **Agent dupes (.opencode/agents/)** | 25 | Mirror of .aether/agents/ |
| **OpenCode commands** | 33 | Mirror of .claude/commands/ |
| **Runtime duplicates** | 29 | Auto-generated from .aether/ |
| **Developer docs (docs/)** | 21 | Implementation plans, handoffs, XML migration |
| **XML migration docs** | 9 | New XML architecture documentation |
| **Plans (docs/plans/)** | 6 | Design documents pending implementation |
| **Handoff docs** | 8 | Session handoffs (mostly stale) |
| **Session freshness docs** | 4 | Implementation plan + 3 handoffs |
| **Dream journal** | 4 | Session notes and reflections |
| **Oracle research** | 4 | Research progress and prompts |
| **Data/survey** | 12 | Colony state documentation |
| **Archive** | 2 | Old model routing documentation |
| **Rules (.claude/rules/)** | 7 | Development guidelines |
| **Root level** | 7 | README, CHANGELOG, TO-DOs, etc. |
| **Tests** | 1 | E2E test documentation |
| **node_modules** | 528 | Dependency READMEs and changelogs |
| **.worktrees/** | ~40 | Git worktree duplicates (excluded) |

**Total: 1,153 markdown files (excluding node_modules and .worktrees)**

---

## Core Documentation (Actively Maintained)

These files represent the current, authoritative documentation:

### System Documentation (.aether/)
| File | Purpose | Status |
|------|---------|--------|
| `/Users/callumcowie/repos/Aether/.aether/workers.md` | Worker/caste definitions | Current |
| `/Users/callumcowie/repos/Aeter/.aether/aether-utils.sh` | Utility library (3,000+ lines) | Current |
| `/Users/callumcowie/repos/Aether/.aether/CONTEXT.md` | Colony context template | Current |
| `/Users/callumcowie/repos/Aether/.aether/DISCIPLINES.md` | Colony discipline rules | Current |
| `/Users/callumcowie/repos/Aether/.aether/QUEEN_ANT_ARCHITECTURE.md` | Queen system architecture | Current |
| `/Users/callumcowie/repos/Aether/.aether/verification.md` | Verification procedures | Current |
| `/Users/callumcowie/repos/Aether/.aether/tdd.md` | Test-driven development guide | Current |

### Master Specifications (.aether/docs/)
| File | Size | Purpose | Status |
|------|------|---------|--------|
| `/Users/callumcowie/repos/Aether/.aether/docs/AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md` | 73KB | Complete pheromone & multi-colony spec | Current |
| `/Users/callumcowie/repos/Aether/.aether/docs/AETHER-2.0-IMPLEMENTATION-PLAN.md` | 36KB | 10-feature roadmap | Current |
| `/Users/callumcowie/repos/Aether/.aether/docs/VISUAL-OUTPUT-SPEC.md` | 6KB | UI/UX standards | Current |
| `/Users/callumcowie/repos/Aether/.aether/docs/QUEEN-SYSTEM.md` | - | Wisdom promotion system | Current |
| `/Users/callumcowie/repos/Aether/.aether/docs/biological-reference.md` | - | Caste taxonomy | Current |

### Command Documentation
| Location | Count | Purpose |
|----------|-------|---------|
| `/Users/callumcowie/repos/Aether/.claude/commands/ant/*.md` | 34 | Claude Code slash commands |
| `/Users/callumcowie/repos/Aether/.opencode/commands/ant/*.md` | 33 | OpenCode slash commands |
| `/Users/callumcowie/repos/Aether/.aether/commands/claude/*.md` | 33 | Source for Claude commands |
| `/Users/callumcowie/repos/Aether/.aether/commands/opencode/*.md` | 33 | Source for OpenCode commands |

### Rules and Guidelines
| File | Purpose |
|------|---------|
| `/Users/callumcowie/repos/Aether/.claude/rules/aether-development.md` | Meta-context for Aether development |
| `/Users/callumcowie/repos/Aether/.claude/rules/aether-specific.md` | Aether-specific rules |
| `/Users/callumcowie/repos/Aether/.claude/rules/coding-standards.md` | Code style guidelines |
| `/Users/callumcowie/repos/Aether/.claude/rules/git-workflow.md` | Git commit policies |
| `/Users/callumcowie/repos/Aether/.claude/rules/security.md` | Protected paths and operations |
| `/Users/callumcowie/repos/Aether/.claude/rules/spawn-discipline.md` | Worker spawn limits |
| `/Users/callumcowie/repos/Aether/.claude/rules/testing.md` | Test framework guidelines |

### XML Migration Documentation (New)
| File | Purpose |
|------|---------|
| `/Users/callumcowie/repos/Aether/docs/xml-migration/XML-MIGRATION-MASTER-PLAN.md` | Hybrid JSON/XML architecture |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/AETHER-XML-VISION.md` | XML adoption vision |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/JSON-XML-TRADE-OFFS.md` | Technical comparison |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/NAMESPACE-STRATEGY.md` | Colony namespace design |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/XSD-SCHEMAS.md` | Schema definitions |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/SHELL-INTEGRATION.md` | XML shell tooling |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/USE-CASES.md` | Usage patterns |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/XML-PHEROMONE-SYSTEM.md` | Pheromone XML format |
| `/Users/callumcowie/repos/Aether/docs/xml-migration/CONTEXT-AWARE-SHARING.md` | Cross-colony sharing |

---

## Stale/Outdated Documentation

These files should be archived or deleted:

### Completed Session Handoffs
| File | Date | Status | Action |
|------|------|--------|--------|
| `/Users/callumcowie/repos/Aether/.aether/HANDOFF.md` | 2026-02-16 | Phase 2 XML complete | Archive |
| `/Users/callumcowie/repos/Aether/.aether/HANDOFF_AETHER_DEV_2026-02-15.md` | 2026-02-15 | Fixes merged | Archive |
| `/Users/callumcowie/repos/Aether/docs/aether_dev_handoff.md` | 2026-02-16 | Phase 1 utilities complete | Archive |
| `/Users/callumcowie/repos/Aether/docs/session-freshness-handoff.md` | 2026-02-16 | All 9 phases complete | Archive |
| `/Users/callumcowie/repos/Aether/docs/session-freshness-handoff-v2.md` | 2026-02-16 | All 9 phases complete | Archive |
| `/Users/callumcowie/repos/Aether/docs/colonize-fix-handoff.md` | - | Fix deployed | Archive |

### Duplicate/Consolidated Documents
| File | Issue | Action |
|------|-------|--------|
| `/Users/callumcowie/repos/Aether/.aether/docs/PHEROMONE-INJECTION.md` | Consolidated into MASTER-SPEC | Delete |
| `/Users/callumcowie/repos/Aether/.aether/docs/PHEROMONE-INTEGRATION.md` | Consolidated into MASTER-SPEC | Delete |
| `/Users/callumcowie/repos/Aether/.aether/docs/PHEROMONE-SYSTEM-DESIGN.md` | Consolidated into MASTER-SPEC | Delete |
| `/Users/callumcowie/repos/Aether/.aether/docs/implementation/pheromones.md` | Duplicate of docs/pheromones.md | Consolidate |
| `/Users/callumcowie/repos/Aether/.aether/docs/implementation/known-issues.md` | Subset of docs/known-issues.md | Consolidate |
| `/Users/callumcowie/repos/Aether/.aether/docs/implementation/pathogen-schema.md` | Duplicate of docs/pathogen-schema.md | Consolidate |

### Old Archive Files
| File | Date | Status |
|------|------|--------|
| `/Users/callumcowie/repos/Aether/.aether/archive/model-routing/README.md` | Old | Keep for history |
| `/Users/callumcowie/repos/Aether/.aether/archive/model-routing/STACK-v3.1-model-routing.md` | Old | Keep for history |
| `/Users/callumcowie/repos/Aether/.aether/oracle/archive/2026-02-16-191250-progress.md` | 2026-02-16 | Archive old research |

### Runtime Directory (Auto-Generated)
**All files in `/Users/callumcowie/repos/Aether/runtime/` are auto-generated from `.aether/`**

These should never be edited directly. The entire directory is essentially a stale copy that gets refreshed on `npm install -g .`.

| File | Source | Notes |
|------|--------|-------|
| `/Users/callumcowie/repos/Aether/runtime/workers.md` | `.aether/workers.md` | Staging copy |
| `/Users/callumcowie/repos/Aether/runtime/docs/*.md` | `.aether/docs/*.md` | 18 files duplicated |
| `/Users/callumcowie/repos/Aether/runtime/*.md` | `.aether/*.md` | 11 files duplicated |

---

## Missing Documentation

These important topics lack documentation:

### Critical Gaps
| Topic | Why Needed | Priority |
|-------|------------|----------|
| **Error Code Standards** | 17+ locations use inconsistent error codes | High |
| **Model Routing Verification** | Unproven whether caste model assignments work | High |
| **QUEEN.md Pipeline** | Wisdom promotion system undocumented | Medium |
| **Session Freshness API** | Docs exist but need integration guide | Medium |
| **Checkpoint Allowlist** | Fixed but not documented for users | Medium |
| **Command Duplication Strategy** | 13,573 lines duplicated between Claude/OpenCode | Medium |
| **Dream Journal Consumption** | Dreams written but never read | Low |
| **Telemetry Analysis** | telemetry.json logged but not analyzed | Low |

### Missing API Documentation
| Component | Missing Docs |
|-----------|--------------|
| `queen-init` | No user-facing documentation |
| `queen-read` | No user-facing documentation |
| `queen-promote` | No user-facing documentation |
| `spawn-tree` tracking | Undocumented spawn tracking system |
| `checkpoint-check` | New utility, needs docs |
| `normalize-args` | New utility, needs docs |

### Missing Developer Guides
| Topic | Current State |
|-------|---------------|
| Contributing to Aether | No CONTRIBUTING.md |
| Architecture decision records | No ADR directory |
| Migration guides | No upgrade path docs |
| Troubleshooting guide | Scattered in known-issues.md |

---

## Organization Issues

### 1. Deep Directory Nesting
```
.aether/docs/implementation/pheromones.md
.aether/docs/implementation/known-issues.md
.aether/docs/reference/biological-reference.md
```

**Issue:** Overly deep hierarchy makes files hard to find.
**Recommendation:** Flatten to `.aether/docs/` with descriptive filenames.

### 2. Duplicate Directory Structures
```
.aether/agents/          (25 files)
.opencode/agents/        (25 files - identical)

.aether/commands/claude/ (33 files)
.aether/commands/opencode/ (33 files)
.claude/commands/ant/    (34 files)
.opencode/commands/ant/  (33 files)
```

**Issue:** 66 command files + 25 agent files = 91 files duplicated.
**Recommendation:** Generate OpenCode files from Claude sources or use shared templates.

### 3. Stale Handoff Accumulation
**Issue:** Handoff documents from completed work remain in active directories.
**Recommendation:** Move to `.aether/archive/handoffs/` or delete after work is merged.

### 4. Runtime/ Staging Confusion
**Issue:** `runtime/` appears to be source code but is auto-generated.
**Recommendation:** Add prominent header to all runtime files: "AUTO-GENERATED: DO NOT EDIT"

### 5. Documentation Fragmentation
Related docs are scattered:
- Pheromone docs: `.aether/docs/PHEROMONE-*.md` (4 files)
- Session freshness: `docs/session-freshness-*.md` (4 files)
- XML migration: `docs/xml-migration/*.md` (9 files)
- Plans: `docs/plans/*.md` (6 files)

**Recommendation:** Consolidate by topic, not by document type.

### 6. Inconsistent Naming
| Pattern | Examples |
|---------|----------|
| ALL_CAPS.md | `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md` |
| lowercase.md | `pheromones.md`, `workers.md` |
| CamelCase.md | None |
| kebab-case.md | `session-freshness-handoff.md` |

**Recommendation:** Standardize on kebab-case for all docs.

---

## Improvement Opportunities

### Immediate (Low Effort, High Impact)

1. **Archive stale handoffs**
   - Move 6 completed handoff documents to `.aether/archive/handoffs/`
   - Est. time: 5 minutes

2. **Delete consolidated pheromone docs**
   - Remove 3 files consolidated into MASTER-SPEC
   - Est. time: 2 minutes

3. **Add runtime headers**
   - Add "AUTO-GENERATED" header to all runtime/ files
   - Est. time: 10 minutes

4. **Consolidate duplicate known-issues.md**
   - Merge `.aether/docs/implementation/known-issues.md` into `.aether/docs/known-issues.md`
   - Est. time: 15 minutes

### Short-term (Medium Effort, High Impact)

5. **Document error code standards**
   - Create `.aether/docs/error-codes.md`
   - Document all `$E_*` constants and usage patterns
   - Est. time: 1 hour

6. **Create missing API docs**
   - Document `queen-*` commands
   - Document `checkpoint-check` and `normalize-args`
   - Est. time: 2 hours

7. **Verify model routing**
   - Test and document whether caste model assignments work
   - Create verification procedure
   - Est. time: 2 hours

### Long-term (High Effort, High Impact)

8. **Command deduplication system**
   - Generate OpenCode commands from Claude sources
   - Create shared template system
   - Eliminate 13,573 lines of duplication
   - Est. time: 1 day

9. **Documentation consolidation**
   - Flatten `.aether/docs/` structure
   - Consolidate by topic (pheromones, session, XML, etc.)
   - Create single source of truth
   - Est. time: 2 days

10. **Automated documentation testing**
    - Verify all links work
    - Verify code examples run
    - Detect stale documentation
    - Est. time: 1 day

---

## File Manifest

### All Documentation Files by Location

```
/Users/callumcowie/repos/Aether/
├── README.md                           # Project overview
├── CHANGELOG.md                        # Release history
├── TO-DOs.md                           # Pending work (67KB)
├── CLAUDE.md                           # Project-specific rules
├── DISCLAIMER.md                       # Legal disclaimer
├── HANDOFF.md                          # STALE: Session handoff
├── RUNTIME UPDATE ARCHITECTURE.md      # Distribution flow
│
├── .aether/                            # SOURCE OF TRUTH
│   ├── workers.md                      # Worker definitions
│   ├── aether-utils.sh                 # Utility library
│   ├── CONTEXT.md                      # Context template
│   ├── DISCIPLINES.md                  # Colony disciplines
│   ├── QUEEN_ANT_ARCHITECTURE.md       # Queen system
│   ├── verification.md                 # Verification procedures
│   ├── tdd.md                          # TDD guide
│   ├── learning.md                     # Learning journal
│   ├── debugging.md                    # Debugging guide
│   ├── planning.md                     # Planning discipline
│   ├── verification-loop.md            # Verification process
│   ├── coding-standards.md             # Code standards
│   ├── workers-new-castes.md           # New caste proposals
│   ├── HANDOFF.md                      # STALE: Build handoff
│   ├── HANDOFF_AETHER_DEV_2026-02-15.md # STALE: Dev handoff
│   ├── PHASE-0-ANALYSIS.md             # Initial analysis
│   ├── RESEARCH-SHARED-DATA.md         # Shared data research
│   ├── DIAGNOSIS_PROMPT.md             # Self-diagnosis
│   ├── diagnose-self-reference.md      # Self-reference guide
│   │
│   ├── docs/                           # Core documentation
│   │   ├── README.md                   # Docs index
│   │   ├── AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md (73KB)
│   │   ├── AETHER-2.0-IMPLEMENTATION-PLAN.md (36KB)
│   │   ├── VISUAL-OUTPUT-SPEC.md       # UI standards
│   │   ├── QUEEN-SYSTEM.md             # Wisdom system
│   │   ├── QUEEN.md                    # Queen wisdom
│   │   ├── biological-reference.md     # Caste taxonomy
│   │   ├── codebase-review.md          # Review checklist
│   │   ├── command-sync.md             # Sync procedures
│   │   ├── constraints.md              # Colony constraints
│   │   ├── implementation-learnings.md # Learnings
│   │   ├── known-issues.md             # Bug tracking
│   │   ├── namespace.md                # Namespace design
│   │   ├── pathogen-schema.md          # Pathogen format
│   │   ├── planning-discipline.md      # Planning guide
│   │   ├── progressive-disclosure.md   # UI patterns
│   │   ├── RECOVERY-PLAN.md            # Recovery procedures
│   │   ├── PHEROMONE-INJECTION.md      # CONSOLIDATED
│   │   ├── PHEROMONE-INTEGRATION.md    # CONSOLIDATED
│   │   ├── PHEROMONE-SYSTEM-DESIGN.md  # CONSOLIDATED
│   │   │
│   │   ├── implementation/             # DUPLICATE
│   │   │   ├── pheromones.md           # Dup of ../pheromones.md
│   │   │   ├── known-issues.md         # Dup of ../known-issues.md
│   │   │   └── pathogen-schema.md      # Dup of ../pathogen-schema.md
│   │   │
│   │   ├── reference/                  # Reference materials
│   │   │   ├── biological-reference.md # Dup of ../
│   │   │   ├── command-sync.md         # Dup of ../
│   │   │   ├── constraints.md          # Dup of ../
│   │   │   ├── namespace.md            # Dup of ../
│   │   │   └── progressive-disclosure.md # Dup of ../
│   │   │
│   │   └── architecture/
│   │       └── MULTI-COLONY-ARCHITECTURE.md
│   │
│   ├── commands/                       # Command definitions
│   │   ├── claude/                     # 33 command files
│   │   └── opencode/                   # 33 command files
│   │
│   ├── agents/                         # 25 agent definitions
│   ├── data/survey/                    # 12 survey docs
│   ├── dreams/                         # 4 dream journal entries
│   ├── oracle/                         # 4 research files
│   └── archive/                        # 2 archive files
│
├── .claude/
│   ├── commands/ant/                   # 34 command files
│   └── rules/                          # 7 rule files
│
├── .opencode/
│   ├── commands/ant/                   # 33 command files
│   ├── agents/                         # 25 agent files (dup)
│   └── OPENCODE.md                     # OpenCode guide
│
├── runtime/                            # AUTO-GENERATED (29 files)
│   ├── workers.md                      # Copy of .aether/
│   ├── docs/                           # 18 copied docs
│   └── *.md                            # 11 copied files
│
├── docs/                               # Developer documentation
│   ├── xml-migration/                  # 9 XML docs (NEW)
│   ├── plans/                          # 6 design plans
│   ├── aether_dev_handoff.md           # STALE
│   ├── colonize-fix-handoff.md         # STALE
│   ├── session-freshness-handoff.md    # STALE
│   ├── session-freshness-handoff-v2.md # STALE
│   ├── session-freshness-api.md        # API docs
│   └── session-freshness-implementation-plan.md
│
└── tests/
    └── e2e/README.md                   # Test docs
```

---

## Recommendations Summary

### Priority 0 (Do Now)
1. Archive 6 stale handoff documents
2. Delete 3 consolidated pheromone docs
3. Consolidate duplicate known-issues.md files

### Priority 1 (This Week)
4. Document error code standards
5. Document queen-* commands
6. Verify and document model routing

### Priority 2 (This Month)
7. Flatten .aether/docs/ directory structure
8. Create command deduplication system
9. Add automated doc validation

### Priority 3 (Future)
10. Implement documentation testing
11. Create CONTRIBUTING.md
12. Build documentation site

---

## Appendix: Count Verification

```bash
# Total markdown files (excluding node_modules)
find /Users/callumcowie/repos/Aether -type f -name "*.md" | grep -v node_modules | wc -l
# Result: 1153

# By category breakdown:
# .aether/*.md:                    17
# .aether/docs/*.md:               32
# .aether/commands/**/*.md:        66
# .aether/agents/*.md:             25
# .aether/data/**/*.md:            12
# .aether/dreams/*.md:              4
# .aether/oracle/*.md:              4
# .aether/archive/*.md:             2
# .claude/commands/**/*.md:        34
# .claude/rules/*.md:               7
# .opencode/commands/**/*.md:      33
# .opencode/agents/*.md:           25
# .opencode/*.md:                   1
# runtime/*.md:                    11
# runtime/docs/*.md:               18
# docs/**/*.md:                    21
# tests/**/*.md:                    1
# Root *.md:                        7
# -----------------------------------
# Total:                          1153 (excluding node_modules)
```

---

*Analysis completed: 2026-02-16*
*Next review: After documentation consolidation project*
