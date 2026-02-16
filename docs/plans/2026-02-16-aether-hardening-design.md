# Aether Hardening Design

> Comprehensive plan to implement deterministic enforcement, modular memory, and CI gates based on research document analysis.

**Date:** 2026-02-16
**Status:** Draft - Pending Approval
**Source Research:** Aether Research for Claude & Agent.md files

---

## Executive Summary

This plan addresses six key areas to harden the Aether codebase against common agentic failures:

| Area | Current State | Target State |
|------|---------------|--------------|
| Modular Memory | 203-line CLAUDE.md, no rules/ | Thin CLAUDE.md + focused rules files |
| Hooks Enforcement | npm preinstall only | Git hooks for deterministic checks |
| Permissions | Basic allowlist | Deny rules, protected paths, sandbox |
| CI Pipeline | None | 6-phase gates (Build→Types→Lint→Tests→Security→Diff) |
| OpenCode Alignment | 26 agents defined | Synchronized rules with Claude |
| Governance | Ad-hoc | Weekly/monthly audits, versioning |

---

## Phase 1: Modular Memory Structure

**Goal:** Reduce CLAUDE.md bloat and improve rule focus by splitting into modular files.

### 1.1 Create `.claude/rules/` Directory Structure

```
.claude/
├── CLAUDE.md              # Thin root (essential rules only, <100 lines)
├── rules/
│   ├── coding-standards.md    # Code style, patterns, conventions
│   ├── testing.md             # Test requirements, coverage targets
│   ├── security.md            # Security guidelines, secret handling
│   ├── git-workflow.md        # Branch naming, commit style, PR process
│   ├── spawn-discipline.md    # Agent spawn limits, depth rules
│   └── aether-specific.md     # .aether/ structure, sync rules
├── settings.json           # Hooks configuration (Phase 2)
└── hooks/                  # Hook scripts (Phase 2)
```

### 1.2 Thin CLAUDE.md Template

```markdown
# CLAUDE.md — Aether Repo Rules

> This file is kept minimal. Domain-specific rules are in `.claude/rules/`.

## 0) Priority / conflicts
1) The user's request in this session
2) This file (repo-wide rules)
3) Repo docs referenced by *path* (not links)

## 1) Essential Commands
- Build: `npm ci`
- Test: `npm test`
- Lint: `npm run lint`
- Sync: `npm install -g .` (syncs .aether/ → runtime/)

## 2) Critical Rules
- **Edit `.aether/` NOT `runtime/`** — runtime/ is auto-generated
- **Max spawn depth: 3** — No worker beyond depth 3
- **Max workers per phase: 10** — Global cap prevents runaway

## 3) Rule Modules
See `.claude/rules/` for domain-specific guidelines:
- @rules/coding-standards.md
- @rules/testing.md
- @rules/security.md
- @rules/spawn-discipline.md
```

### 1.3 Implementation Tasks

- [ ] Create `.claude/rules/` directory
- [ ] Extract coding standards from CLAUDE.md → `rules/coding-standards.md`
- [ ] Extract TDD/testing rules → `rules/testing.md`
- [ ] Extract spawn discipline → `rules/spawn-discipline.md`
- [ ] Create `rules/security.md` with secret handling guidelines
- [ ] Create `rules/aether-specific.md` with .aether/ structure docs
- [ ] Thin CLAUDE.md to <100 lines with rule imports
- [ ] Test: Verify rules load with `/memory` command

---

## Phase 2: Hooks Enforcement

**Goal:** Implement deterministic enforcement via git hooks that cannot be bypassed by agent behavior.

### 2.1 Hook Configuration (`.claude/settings.json`)

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/block-destructive.sh" }
        ]
      },
      {
        "matcher": "Edit|Write",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/protect-paths.sh" }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/auto-format.sh" }
        ]
      },
      {
        "matcher": "*",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/log-action.sh" }
        ]
      }
    ]
  },
  "permissions": {
    "allow": [ /* existing allows */ ],
    "deny": [
      "Bash(rm -rf /*)",
      "Bash(*curl*|*wget*|*nc*)",
      "Edit(.aether/data/*)",
      "Edit(.env*)",
      "Write(.aether/checkpoints/*)"
    ]
  }
}
```

### 2.2 Hook Scripts to Create

| Script | Purpose | Trigger |
|--------|---------|---------|
| `block-destructive.sh` | Block `rm -rf`, `drop database`, etc. | PreToolUse:Bash |
| `protect-paths.sh` | Block edits to `.aether/data/`, `.env` | PreToolUse:Edit |
| `auto-format.sh` | Run prettier/eslint --fix after edits | PostToolUse:Edit |
| `log-action.sh` | Append to `.aether/ledger.jsonl` | PostToolUse:* |
| `gate-continue.sh` | Block `/ant:continue` if tests not passed | PreUserPrompt |

### 2.3 Git Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

# 1. Check if runtime/ was edited directly
if git diff --cached --name-only | grep -q "^runtime/"; then
  echo "ERROR: Direct edits to runtime/ detected."
  echo "runtime/ is generated from .aether/. Edit source files instead."
  echo ""
  echo "Fix: git checkout -- runtime/ && npm install -g ."
  exit 1
fi

# 2. Run sync to ensure runtime/ is up to date
bash bin/sync-to-runtime.sh

# 3. Stage any synced changes
git add runtime/ 2>/dev/null || true

exit 0
```

### 2.4 Implementation Tasks

- [ ] Create `.claude/hooks/` directory
- [ ] Implement `block-destructive.sh`
- [ ] Implement `protect-paths.sh`
- [ ] Implement `auto-format.sh`
- [ ] Implement `log-action.sh`
- [ ] Update `.claude/settings.json` with hook config
- [ ] Create `.git/hooks/pre-commit` for runtime/ protection
- [ ] Test: Attempt `rm -rf` command, verify blocked
- [ ] Test: Edit `runtime/` directly, verify commit blocked

---

## Phase 3: Permissions Hardening

**Goal:** Implement least-privilege access with deny rules and protected paths.

### 3.1 Protected Paths

```
# Never allow agents to edit:
.aether/data/          # Colony state
.aether/checkpoints/   # Session checkpoints
.aether/locks/         # File locks
.env*                  # Environment secrets
.claude/settings.json  # Hook configuration
.github/workflows/     # CI configuration
```

### 3.2 Permission Rules Update

```json
{
  "permissions": {
    "allow": [
      "Bash(git *)",
      "Bash(npm *)",
      "Bash(node *)",
      "Bash(bash .aether/*)",
      "Edit(src/**/*)",
      "Edit(.aether/workers.md)",
      "Edit(.aether/docs/*)",
      "Read(**/*)"
    ],
    "deny": [
      "Bash(rm -rf /*)",
      "Bash(*sudo*)",
      "Bash(*curl*|*wget*)",
      "Edit(.aether/data/*)",
      "Edit(.env*)",
      "Edit(.claude/settings.json)",
      "Write(.aether/checkpoints/*)"
    ]
  }
}
```

### 3.3 Sandbox Mode (Optional)

For maximum security, enable OS-level sandboxing:

```bash
# Enable sandbox with auto-allow for safe commands
claude --sandbox --permission-mode auto-allow
```

### 3.4 Implementation Tasks

- [ ] Audit current permissions in `settings.local.json`
- [ ] Add deny rules for protected paths
- [ ] Add deny rules for dangerous bash patterns
- [ ] Document sandbox usage in `rules/security.md`
- [ ] Test: Attempt to edit `.aether/data/`, verify denied

---

## Phase 4: CI Pipeline (6-Phase Gates)

**Goal:** Implement GitHub Actions workflow with mandatory quality gates.

### 4.1 Workflow Structure

```yaml
# .github/workflows/ci.yml
name: Aether CI

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'

      - name: Install dependencies
        run: npm ci

      - name: Type Check
        run: npm run tsc || echo "No TypeScript"

      - name: Lint
        run: npm run lint

      - name: Test
        run: npm test

      - name: Security Scan
        run: npm audit --audit-level=high

      - name: Verify Sync
        run: |
          bash bin/sync-to-runtime.sh
          if git diff --exit-code runtime/; then
            echo "runtime/ in sync"
          else
            echo "ERROR: runtime/ out of sync with .aether/"
            exit 1
          fi
```

### 4.2 Branch Protection Rules

Configure in GitHub Settings → Branches:

- [ ] Require status checks to pass before merging
- [ ] Require branches to be up to date before merging
- [ ] Status checks required: `build`
- [ ] Require pull request reviews (1 approval)

### 4.3 Diff Verification (Phase 6)

Create a script to verify code changes match planned work:

```bash
# scripts/verify-plan-diff.sh
#!/bin/bash
# Compare .aether/plan.json to actual git diff
# Exit 1 if changes don't match plan
```

### 4.4 Implementation Tasks

- [ ] Create `.github/workflows/ci.yml`
- [ ] Configure branch protection rules in GitHub
- [ ] Create `scripts/verify-plan-diff.sh` (optional, advanced)
- [ ] Test: Open PR with failing test, verify blocked
- [ ] Test: Open PR with passing checks, verify merge allowed

---

## Phase 5: OpenCode Alignment

**Goal:** Ensure OpenCode agents follow same rules as Claude Code.

### 5.1 Current State

OpenCode has 26 agent definitions in `.opencode/agents/` but no shared rules with Claude.

### 5.2 Alignment Strategy

1. **Reference shared rules** from `.claude/rules/` in OpenCode agent prompts
2. **Synchronize castes** between Claude workers.md and OpenCode agents
3. **Use same permissions** in `opencode.json`

### 5.3 OpenCode Agent Template

```markdown
---
name: builder
mode: primary
description: Code construction agent
prompt: |
  You are a Builder Ant in the Aether Colony.

  Follow rules from .claude/rules/coding-standards.md and .claude/rules/testing.md

  Key constraints:
  - Max spawn depth: 3
  - TDD: Write failing test first
  - Edit .aether/ not runtime/
tools: [Read, Write, Edit, Bash]
model: claude-sonnet-4-5
---
```

### 5.4 Implementation Tasks

- [ ] Audit OpenCode agents vs Claude workers for consistency
- [ ] Update agent prompts to reference `.claude/rules/`
- [ ] Verify `.opencode/opencode.json` permissions match
- [ ] Test: Run same task in both Claude and OpenCode, compare behavior

---

## Phase 6: Governance & Maintenance

**Goal:** Establish ongoing maintenance rituals.

### 6.1 Weekly Audit Checklist

```markdown
## Weekly Aether Audit

- [ ] Check `.aether/ledger.jsonl` for recent entries
- [ ] Verify hook scripts ran without errors
- [ ] Confirm CLAUDE.md and rules in git are current
- [ ] Run `claude --continue --debug` to check for silent failures
- [ ] Review any new deny rules needed
```

### 6.2 Monthly Audit Checklist

```markdown
## Monthly Aether Audit

- [ ] Re-run CI pipeline on main branch (no code changes)
- [ ] Run `npm audit` and apply fixes
- [ ] Prune old `.aether/checkpoints/`
- [ ] Review `.aether/` size and clean up
- [ ] Check LiteLLM call logs for anomalies
- [ ] Update dependencies if needed
```

### 6.3 Versioning Policy

- **CLI:** SemVer (e.g., 3.1.14)
- **State Schema:** Version field in `.aether/state-schema.json`
- **Rules:** Git-tracked, changes require PR

### 6.4 Implementation Tasks

- [ ] Create `.aether/templates/weekly-audit.md`
- [ ] Create `.aether/templates/monthly-audit.md`
- [ ] Add `version` field to colony state schema
- [ ] Document versioning in `rules/aether-specific.md`

---

## Implementation Sequence

Recommended order for implementation:

```
Week 1: Phase 1 (Modular Memory) + Phase 3 (Permissions)
        ↓
Week 2: Phase 2 (Hooks)
        ↓
Week 3: Phase 4 (CI Pipeline)
        ↓
Week 4: Phase 5 (OpenCode Alignment) + Phase 6 (Governance)
```

**Rationale:**
1. Modular memory first — establishes rule structure others reference
2. Permissions with memory — defines boundaries early
3. Hooks second — enforces the rules we just created
4. CI third — catches what hooks miss
5. OpenCode + Governance last — alignment and maintenance

---

## Success Criteria

| Metric | Target | How to Verify |
|--------|--------|---------------|
| CLAUDE.md size | <100 lines | `wc -l CLAUDE.md` |
| Hook coverage | 100% of dangerous ops | Test each deny rule |
| CI pass rate | 100% before merge | Check GitHub PRs |
| Rules loaded | All rule files visible | `/memory` command |
| Audit compliance | Weekly checklist done | Review audit logs |

---

## Rollback Plan

If any phase causes issues:

1. **Hooks:** Remove entries from `.claude/settings.json`
2. **CI:** Disable GitHub Actions workflow
3. **Rules:** Revert to previous CLAUDE.md (git restore)
4. **Permissions:** Remove deny rules from settings

---

## Appendix: File Changes Summary

| Action | File |
|--------|------|
| Create | `.claude/rules/coding-standards.md` |
| Create | `.claude/rules/testing.md` |
| Create | `.claude/rules/security.md` |
| Create | `.claude/rules/spawn-discipline.md` |
| Create | `.claude/rules/aether-specific.md` |
| Create | `.claude/rules/git-workflow.md` |
| Create | `.claude/hooks/block-destructive.sh` |
| Create | `.claude/hooks/protect-paths.sh` |
| Create | `.claude/hooks/auto-format.sh` |
| Create | `.claude/hooks/log-action.sh` |
| Create | `.github/workflows/ci.yml` |
| Create | `.aether/templates/weekly-audit.md` |
| Create | `.aether/templates/monthly-audit.md` |
| Modify | `CLAUDE.md` (thin to <100 lines) |
| Modify | `.claude/settings.json` (add hooks, permissions) |
| Modify | `.opencode/agents/*.md` (reference shared rules) |
