# Phase 74: Suggest-Analyze - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-29
**Phase:** 74-suggest-analyze
**Areas discussed:** Suggestion trigger and scope, Approval UX, Deduplication behavior, Platform differences

---

## Suggestion Trigger and Scope

### When should suggest-analyze run during builds?

| Option | Description | Selected |
|--------|-------------|----------|
| Every build | Runs on every /ant-build invocation. Simple, consistent, but may re-analyze unchanged codebases. | |
| First build only | Only on first build per colony. Saves time but misses new patterns that emerge during development. | |
| First build + change detection | First build always, then only if files changed significantly since last analysis (tracked via git diff or timestamp). Best balance but adds state tracking. | ✓ |

**User's choice:** First build + change detection
**Notes:** Balance between freshness and efficiency.

### Should suggest-analyze use the same 25 patterns as init-research, or a different set?

| Option | Description | Selected |
|--------|-------------|----------|
| Same 25 patterns | Reuses the same 25 patterns from init-research (generatePheromoneSuggestions). Zero new code for detection logic, consistent with Phase 73. | |
| 25 base + build-specific extras | Uses the 25 base patterns plus adds build-specific patterns (e.g., test coverage gaps, TODO/FIXME density, large file warnings). More useful but more code. | ✓ |
| Separate build-time patterns | Completely separate pattern set for build-time analysis. Maximum flexibility but doubles the maintenance burden. | |

**User's choice:** 25 base + build-specific extras
**Notes:** Wants the foundation from init-research plus patterns that are specifically useful during active development.

---

## Approval UX

### How should users review pheromone suggestions?

| Option | Description | Selected |
|--------|-------------|----------|
| Inline during build | Suggestions display inline during the build ceremony (after Step 4.2), user approves/dismisses each one before workers spawn. Pauses the build flow briefly. | ✓ |
| Separate command after build | Suggestions are stored silently during build. User runs /ant-suggest-review (or aether suggest-approve) separately after build completes. Doesn't pause the build. | |
| Inline with auto-approve timeout | Display inline during build but don't block — auto-approve after a timeout. Best of both worlds but more complex. | |

**User's choice:** Inline during build
**Notes:** Wants to see and act on suggestions as part of the build flow.

### What happens to suggestions the user doesn't review?

| Option | Description | Selected |
|--------|-------------|----------|
| Persist until resolved | Unreviewed suggestions persist in colony state until explicitly approved or dismissed. They survive /clear and show up on every build until resolved. | ✓ |
| Expire at end of build | Suggestions expire at the end of the current build. If user doesn't review them, they're gone. Simpler but may miss useful suggestions. | |
| Auto-dismiss after N builds | Suggestions persist for N builds (e.g., 3), then auto-dismiss if not reviewed. Balances persistence with noise reduction. | |

**User's choice:** Persist until resolved
**Notes:** Doesn't want useful suggestions to disappear because of a missed review.

---

## Deduplication Behavior

### How strictly should suggestions deduplicate against existing active pheromones?

| Option | Description | Selected |
|--------|-------------|----------|
| Exact match only | Only suppress suggestions that exactly match an active pheromone (same type + same content hash). Simple, predictable, already implemented in pheromone-write. | ✓ |
| Exact + semantic similarity | Exact match plus suppress suggestions whose content is very similar (e.g., 'never commit secrets' vs 'don't commit .env files'). More useful but requires similarity logic. | |
| No deduplication | Don't deduplicate at all — show all suggestions even if they duplicate existing pheromones. User decides what's redundant. | |

**User's choice:** Exact match only
**Notes:** Keeps it simple and predictable. The existing content_hash infrastructure already handles this.

### Should suggest-analyze re-suggest content that previously existed as an expired pheromone?

| Option | Description | Selected |
|--------|-------------|----------|
| Re-suggest expired | Re-suggest even if the same content existed as an expired pheromone. The user might want to re-activate it. | ✓ |
| Skip if ever existed | Skip suggestions that match any pheromone (active OR expired). Prevents re-suggesting things the user has already seen and let expire. | |

**User's choice:** Re-suggest expired
**Notes:** Expired pheromones may have been let go for timing reasons, not because the user didn't want them.

---

## Platform Differences

### How should suggest-analyze work across Claude Code, OpenCode, and Codex?

| Option | Description | Selected |
|--------|-------------|----------|
| Single CLI, wrappers display | All three platforms call the same Go CLI command (aether suggest-analyze). The runtime owns the logic, wrappers just display output. Consistent with CLAUDE.md platform policy. | ✓ |
| Native Codex + wrapper others | Codex gets a richer native experience (Go-only), Claude/OpenCode get markdown-rendered output. More work but better Codex UX. | |

**User's choice:** Single CLI, wrappers display
**Notes:** Consistent with the established platform policy — Go runtime owns logic, wrappers own presentation.

---

## Claude's Discretion

- Exact change detection threshold for re-analysis
- Number and content of build-specific extra patterns
- Visual rendering of the approval UI per platform
- Storage format for pending suggestions in colony state
- Summary vs full detail display for re-shown persisted suggestions

## Deferred Ideas

None.
