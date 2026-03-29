# QUEEN.md -- Colony Wisdom

> Last evolved: 2026-03-24T23:40:00Z
> Wisdom version: 2.0.0

---

## User Preferences

Communication style, expertise level, and decision-making patterns observed from the user (the Queen). These shape how the colony communicates and what it prioritizes. User decisions are the most important wisdom.


- [charter] **Intent**: Comprehensive audit of today's session work — verify QUEEN.md, pheromone system, wisdom pipeline, charter-write, and all colony lifecycle changes are functioning correctly (Colony: Aether Colony)
- [charter] **Vision**: Confirm that everything built across three colonies today (versioning, seal audit fixes, visual consistency) works end-to-end with no regressions, silent failures, or data loss (Colony: Aether Colony)
---

## Codebase Patterns

Validated approaches that work in this codebase, and anti-patterns to avoid. Includes architecture conventions, naming patterns, error handling style, and technology-specific insights. Tagged [repo] for project-specific or [general] for cross-colony patterns.

- **1774650429** (2026-03-29T02:06:34Z): awk NF==7 for spawn detection and $3 match for completion detection handles pipes-in-summary edge cases
- **1774650429** (2026-03-29T02:06:30Z): Replacing bash while-read+sed loops with single-pass awk eliminates O(n^2) subprocess forking
- **1774650429** (2026-03-29T01:57:37Z): Chaos resilience strong: 4/5 scenarios resilient for spawn-tree awk rewrite, 1 medium finding on newline edge case
- **1774650429** (2026-03-29T01:11:51Z): Inserted phase 6 (Stabilize spawn-tree parsing and JSON output): Fix all 3 spawn-tree blockers: (1) Fix O(n^2) parse_spawn_tree scaling that causes test timeouts, (2) JSON-escape ant names before interpolation into JSON output strings, (3) Add structural JSON validation in spawn-tree-load before passing through json_ok
- **1774650429** (2026-03-28T23:56:11Z): JSON injection in spawn.sh persists despite being identified as an instinct — pattern recurs because the fix was applied only to queen.sh, not spawn.sh
- **1774650429** (2026-03-28T23:56:09Z): Lifecycle commands (init, seal, entomb) properly handle colony_version through template system — init uses colony-state.template.json, seal has 12 references, entomb reads and displays it
- **1774650429** (2026-03-28T03:45:43Z): Every atomic mv that overwrites a critical file must have a non-empty size guard to prevent data destruction from upstream pipeline failures
- **1774650429** (2026-03-28T03:45:41Z): Shell functions using sed c-command for line replacement must use head/tail instead to handle multi-line content safely on macOS BSD sed
- **1774650429** (2026-03-28T00:14:55Z): Shell functions that embed user-derived values in JSON output strings must use jq for safe construction to prevent JSON injection from special characters
- **1774650429** (2026-03-28T00:14:51Z): Shell functions that set traps must compose with _aether_exit_cleanup to avoid orphaning file locks and temp files when the function exits abnormally
- **1774650429** (2026-03-27T23:20:16Z): Test fixtures with hardcoded dates will break as calendar time advances — use dynamic date computation with cross-platform fallbacks instead
- **1774650429** (2026-03-27T23:14:32Z): Chaos resilience moderate: 5 scenarios tested on pheromone-expire date fix, 3 resilient, 2 findings (static fallback staleness, double date failure)
- **1774645519** (2026-03-27T21:50:42Z): Inserted phase 5 (Stabilize caste emojis in spawn and phase displays): Add caste emojis to all ant spawn announcements and phase header displays across all commands — every spawn shows its caste emoji and phase headers include visual emoji markers
- [general] **Use explicit jq if/elif chains instead of the // operator when checking fields that can legitimately be false** (source: colony 1771335865738, 2026-03-20)

- [hive] When creating testimonials, press bars, or review content: Use clearly labeled placeholders instead of fabricating content — mark sections as 'Your real testimonial here' or similar (cross-colony, confidence: 0.95)
- [hive] When implementing pricing or booking flow: Never display session prices — route all booking interest through contact/enquiry forms. Contact-first model. (cross-colony, confidence: 0.95)
- [hive] When Deploying files via SFTP to Cloudways: Use -oPreferredAuthentications=password flags for Cloudways SFTP (cross-colony, confidence: 0.9)
- [hive] When Verifying deployed changes on Cloudways: Use ?nocache= to bypass Varnish when verifying deploys (cross-colony, confidence: 0.9)
- [hive] When building bash utilities with scoring/accumulation loops: use process substitution (&lt; &lt;(jq)) not pipes to while loops — pipes create subshells that lose variable modifications (cross-colony, confidence: 0.85)
- [charter] **Governance**: CI/CD pipeline active — ensure all checks pass before merging (Colony: Aether Colony)
---

## Build Learnings

What worked and what failed during builds. Captures the full picture of colony experience -- successes, failures, and adjustments. Each entry includes the phase where it was learned.



### Phase 0: migration-test
- [repo] QUEEN.md v2 migration validated -- *Phase 0 (migration-test)* (2026-03-24)

### Phase 6: Stabilize spawn-tree parsing and JSON output
- [general] Replacing bash while-read+sed loops with single-pass awk eliminates O(n^2) subprocess forking — 4000+ forks reduced to 1 awk process, runtime from 23s to 1.7s -- *Phase 6 (Stabilize spawn-tree parsing and JSON output)* (2026-03-29)
- [general] awk NF==7 for spawn detection and $3 match for completion detection handles pipes-in-summary edge cases that pipe-counting cannot -- *Phase 6 (Stabilize spawn-tree parsing and JSON output)* (2026-03-29)
---

## Instincts

High-confidence behavioral patterns that have been validated through repeated colony work. Auto-promoted when confidence reaches 0.8 or higher. These represent the colony's deepest learned behaviors.

- [instinct] **testing** (0.85): When codebase changes, then always run full test suite after module extraction

- [instinct] **testing** (0.8): When test fixtures use hardcoded dates, then replace with dynamic cross-platform date computation to prevent time-based test degradation
- [instinct] **code-style** (0.8): When shell functions set EXIT/TERM traps, then compose trap with _aether_exit_cleanup to preserve lock and temp file cleanup on abnormal exit
- [instinct] **code-style** (0.85): When shell scripts use sed c-command for line replacement, then replace with head/tail pattern for cross-platform newline safety — sed c breaks on macOS BSD with multi-line content
- [instinct] **code-style** (0.85): When atomic mv overwrites a critical data file, then add non-empty size guard (if [[ ! -s file ]]) before mv to prevent data destruction from upstream pipeline failures
- [instinct] **code-style** (0.85): When bash scripts use while-read+sed/cut loops for file parsing, then replace with single-pass awk to eliminate O(n^2) subprocess forking
---

## Evolution Log

| Date | Source | Type | Details |
|------|--------|------|---------|
| 2026-03-29T02:07:09Z | phase-6 | build_learnings | Added 2 learnings from Phase 6: Stabilize spawn-tree parsing and JSON output |
| 2026-03-29T02:06:48Z | instinct | promoted_instinct | code-style: replace with single-pass awk to eliminate O(n^2) s... |
| 2026-03-29T02:06:34Z | 1774650429 | promoted_pattern | Added: awk NF==7 for spawn detection and $3 match for com... |
| 2026-03-29T02:06:30Z | 1774650429 | promoted_pattern | Added: Replacing bash while-read+sed loops with single-pa... |
| 2026-03-29T01:57:37Z | 1774650429 | promoted_pattern | Added: Chaos resilience strong: 4/5 scenarios resilient f... |
| 2026-03-29T01:11:51Z | 1774650429 | promoted_pattern | Added: Inserted phase 6 (Stabilize spawn-tree parsing and... |
| 2026-03-28T23:56:11Z | 1774650429 | promoted_pattern | Added: JSON injection in spawn.sh persists despite being ... |
| 2026-03-28T23:56:09Z | 1774650429 | promoted_pattern | Added: Lifecycle commands (init, seal, entomb) properly h... |
| 2026-03-28T03:46:05Z | instinct | promoted_instinct | code-style: add non-empty size guard (if [[ ! -s file ]]) befo... |
| 2026-03-28T03:46:05Z | instinct | promoted_instinct | code-style: replace with head/tail pattern for cross-platform ... |
| 2026-03-28T03:45:43Z | 1774650429 | promoted_pattern | Added: Every atomic mv that overwrites a critical file mu... |
| 2026-03-28T03:45:41Z | 1774650429 | promoted_pattern | Added: Shell functions using sed c-command for line repla... |
| 2026-03-28T00:15:09Z | instinct | promoted_instinct | code-style: compose trap with _aether_exit_cleanup to preserve... |
| 2026-03-28T00:14:55Z | 1774650429 | promoted_pattern | Added: Shell functions that embed user-derived values in ... |
| 2026-03-28T00:14:51Z | 1774650429 | promoted_pattern | Added: Shell functions that set traps must compose with _... |
| 2026-03-27T23:20:27Z | instinct | promoted_instinct | testing: replace with dynamic cross-platform date computati... |
| 2026-03-27T23:20:16Z | 1774650429 | promoted_pattern | Added: Test fixtures with hardcoded dates will break as c... |
| 2026-03-27T23:14:32Z | 1774650429 | promoted_pattern | Added: Chaos resilience moderate: 5 scenarios tested on p... |
| 2026-03-27T22:26:57Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T21:50:42Z | 1774645519 | promoted_pattern | Added: Inserted phase 5 (Stabilize caste emojis in spawn ... |
| 2026-03-27T21:05:08Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T19:58:37Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T19:16:20Z | system | charter_updated | Colony charter updated for Aether Colony |
| 2026-03-27T18:52:28Z | system | charter_updated | Colony charter updated for Aether Colony |
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
  "last_evolved": "2026-03-29T02:07:09Z",
  "colonies_contributed": ["1774645519"],
  "stats": {
    "total_user_prefs": 2,
    "total_codebase_patterns": 20,
    "total_build_learnings": 3,
    "total_instincts": 6
  },
  "evolution_log": [{"timestamp": "2026-03-24T23:40:00Z", "action": "migrate", "wisdom_type": "system", "content_hash": "v1-to-v2-migration", "colony": "system"}, {"timestamp": "2026-03-20T12:37:32Z", "action": "promote", "wisdom_type": "pattern", "content_hash": "sha256:f8aa50cfda0f37cac6cabba140bb99f1d75aa6d01a7100fe7a5ccddc2b3a017b", "colony": "1771335865738"}]
}
-->
