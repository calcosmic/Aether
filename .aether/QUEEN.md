# QUEEN.md -- Colony Wisdom

> Last evolved: 2026-03-24T23:40:00Z
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen). These shape how the colony communicates and what it prioritizes. User decisions are the most important wisdom.

*No user preferences recorded yet.*

---

## Codebase Patterns

Validated approaches that work in this codebase, and anti-patterns to avoid. Includes architecture conventions, naming patterns, error handling style, and technology-specific insights. Tagged [repo] for project-specific or [general] for cross-colony patterns.

- [general] **Use explicit jq if/elif chains instead of the // operator when checking fields that can legitimately be false** (source: colony 1771335865738, 2026-03-20)

---

## Build Learnings

What worked and what failed during builds. Captures the full picture of colony experience -- successes, failures, and adjustments. Each entry includes the phase where it was learned.



### Phase 0: migration-test
- [repo] QUEEN.md v2 migration validated -- *Phase 0 (migration-test)* (2026-03-24)
---

## Instincts

High-confidence behavioral patterns that have been validated through repeated colony work. Auto-promoted when confidence reaches 0.8 or higher. These represent the colony's deepest learned behaviors.

- [instinct] **testing** (0.85): When codebase changes, then always run full test suite after module extraction

---

## Evolution Log

| Date | Source | Type | Details |
|------|--------|------|---------|
| 2026-03-24T23:40:41Z | instinct | promoted_instinct | testing: always run full test suite after module extraction... |
| 2026-03-24T23:40:36Z | phase-0 | build_learnings | Added 1 learnings from Phase 0: migration-test |
| 2026-03-24T23:40:00Z | system | migrated | QUEEN.md migrated from v1 (6-section) to v2 (4-section) format |
| 2026-03-20T12:37:32Z | 1771335865738 | promoted_pattern | Added: Use explicit jq if/elif chains instead of the // o... |
| 2026-03-19T22:07:00Z | system | initialized | QUEEN.md created from template |

---

<!-- METADATA
{
  "version": "2.0.0",
  "wisdom_version": "2.0",
  "last_evolved": "2026-03-24T23:40:41Z",
  "colonies_contributed": [],
  "stats": {
    "total_user_prefs": 0,
    "total_codebase_patterns": 1,
    "total_build_learnings": 1,
    "total_instincts": 1
  },
  "evolution_log": [{"timestamp": "2026-03-24T23:40:00Z", "action": "migrate", "wisdom_type": "system", "content_hash": "v1-to-v2-migration", "colony": "system"}, {"timestamp": "2026-03-20T12:37:32Z", "action": "promote", "wisdom_type": "pattern", "content_hash": "sha256:f8aa50cfda0f37cac6cabba140bb99f1d75aa6d01a7100fe7a5ccddc2b3a017b", "colony": "1771335865738"}]
}
-->
