# QUEEN.md -- Colony Wisdom

> Last evolved: 2026-03-24T23:40:00Z
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen). These shape how the colony communicates and what it prioritizes. User decisions are the most important wisdom.


- [charter] **Intent**: AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA... (Colony: Aether Colony)
- [charter] **Vision**: Short (Colony: Aether Colony)
---

## Codebase Patterns

Validated approaches that work in this codebase, and anti-patterns to avoid. Includes architecture conventions, naming patterns, error handling style, and technology-specific insights. Tagged [repo] for project-specific or [general] for cross-colony patterns.

- [general] **Use explicit jq if/elif chains instead of the // operator when checking fields that can legitimately be false** (source: colony 1771335865738, 2026-03-20)

- [hive] When creating testimonials, press bars, or review content: Use clearly labeled placeholders instead of fabricating content — mark sections as 'Your real testimonial here' or similar (cross-colony, confidence: 0.95)
- [hive] When implementing pricing or booking flow: Never display session prices — route all booking interest through contact/enquiry forms. Contact-first model. (cross-colony, confidence: 0.95)
- [hive] When Deploying files via SFTP to Cloudways: Use -oPreferredAuthentications=password flags for Cloudways SFTP (cross-colony, confidence: 0.9)
- [hive] When Verifying deployed changes on Cloudways: Use ?nocache= to bypass Varnish when verifying deploys (cross-colony, confidence: 0.9)
- [hive] When building bash utilities with scoring/accumulation loops: use process substitution (&lt; &lt;(jq)) not pipes to while loops — pipes create subshells that lose variable modifications (cross-colony, confidence: 0.85)
- [charter] **Governance**: Also short (Colony: Aether Colony)
- [charter] **Goal**: Goals (Colony: Aether Colony)
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
| 2026-03-27T16:37:32Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T16:37:22Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T16:36:49Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T16:36:39Z | system | charter_initialized | Colony charter created for Aether Colony |
| 2026-03-25T02:01:24Z | hive | seed | Seeded 5 cross-colony patterns from hive |
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
  "last_evolved": "2026-03-27T16:37:32Z",
  "colonies_contributed": [],
  "stats": {
    "total_user_prefs": 2,
    "total_codebase_patterns": 8,
    "total_build_learnings": 1,
    "total_instincts": 1
  },
  "evolution_log": [{"timestamp": "2026-03-24T23:40:00Z", "action": "migrate", "wisdom_type": "system", "content_hash": "v1-to-v2-migration", "colony": "system"}, {"timestamp": "2026-03-20T12:37:32Z", "action": "promote", "wisdom_type": "pattern", "content_hash": "sha256:f8aa50cfda0f37cac6cabba140bb99f1d75aa6d01a7100fe7a5ccddc2b3a017b", "colony": "1771335865738"}]
}
-->
