# Caste System Reference

This is the **canonical source** for Aether caste emoji definitions.

- All commands and documentation should reference this file
- The `get_caste_emoji()` function in `aether-utils.sh` implements these mappings
- To add a new caste: update this file AND the function

## Display Format

Workers are displayed as: `{caste_emoji} {worker_name}`
Example: `🔨🐜 Hammer-42` (not "Hammer-42 (Builder)")

## Caste Table

| Caste | Emoji | Role | Name Patterns |
|-------|-------|------|---------------|
| queen | 👑🐜 | Colony coordinator | Queen, QUEEN, queen |
| builder | 🔨🐜 | Implementation work | Builder, Bolt, Hammer, Forge, Mason, Brick, Anvil, Weld |
| watcher | 👁️🐜 | Monitoring, observation | Watcher, Vigil, Sentinel, Guard, Keen, Sharp, Hawk, Alert |
| scout | 🔍🐜 | Research, discovery | Scout, Swift, Dash, Ranger, Track, Seek, Path, Roam, Quest |
| colonizer | 🗺️🐜 | New project setup | Colonizer, Pioneer, Map, Chart, Venture, Explore, Compass, Atlas, Trek |
| surveyor | 📊🐜 | Measurement, assessment | Surveyor, Chart, Plot, Survey, Measure, Assess, Gauge, Sound, Fathom |
| architect | 🏛️🐜 | Planning, design (merged into Keeper — no dedicated agent file) | Architect, Blueprint, Draft, Design, Plan, Schema, Frame, Sketch, Model |
| chaos | 🎲🐜 | Edge case testing | Chaos, Probe, Stress, Shake, Twist, Snap, Breach, Surge, Jolt |
| archaeologist | 🏺🐜 | Git history excavation | Archaeologist, Relic, Fossil, Dig, Shard, Epoch, Strata, Lore, Glyph |
| oracle | 🔮🐜 | Deep research (RALF loop) | Oracle, Sage, Seer, Vision, Augur, Mystic, Sibyl, Delph, Pythia |
| route_setter | 📋🐜 | Direction setting | Route, route |
| ambassador | 🔌🐜 | Third-party API integration | Ambassador, Bridge, Connect, Link, Diplomat, Network, Protocol |
| auditor | 👥🐜 | Code review, quality audits | Auditor, Review, Inspect, Examine, Scrutin, Critical, Verify |
| chronicler | 📝🐜 | Documentation generation | Chronicler, Document, Record, Write, Chronicle, Archive, Scribe |
| gatekeeper | 📦🐜 | Dependency management | Gatekeeper, Guard, Protect, Secure, Shield, Depend, Supply |
| guardian | 🛡️🐜 | Security audits (merged into Auditor — no dedicated agent file) | Guardian, Defend, Patrol, Secure, Vigil, Watch, Safety, Security |
| includer | ♿🐜 | Accessibility audits | Includer, Access, Inclusive, A11y, WCAG, Barrier, Universal |
| keeper | 📚🐜 | Knowledge curation | Keeper, Archive, Store, Curate, Preserve, Knowledge, Wisdom, Pattern |
| measurer | ⚡🐜 | Performance profiling | Measurer, Metric, Benchmark, Profile, Optimize, Performance, Speed |
| probe | 🧪🐜 | Test generation | Probe, Test, Excavat, Uncover, Edge, Case, Mutant |
| tracker | 🐛🐜 | Bug investigation | Tracker, Debug, Trace, Follow, Bug, Hunt, Root |
| weaver | 🔄🐜 | Code refactoring | Weaver, Refactor, Restruct, Transform, Clean, Pattern, Weave |
| dreamer | 💭🐜 | Creative ideation | Dreamer, Muse, Imagine, Wonder, Ponder, Reverie |

## Notes

- The global `get_caste_emoji()` function matches by **name pattern** (e.g., a worker named "Hammer-42" matches the builder caste)
- Castes without dedicated patterns fall back to the generic ant emoji `🐜`
- The `colonizer` canonical emoji is `🗺️🐜` — older references using `🌱🐜` should be updated
- The `route_setter` canonical emoji is `📋🐜` — older references using `🧭🐜` should be updated
- The `architect` and `guardian` castes are **merged**: their capabilities were absorbed by Keeper and Auditor respectively (Phase 25). The caste emoji rows remain because workers named after those patterns (e.g., "Blueprint-3", "Patrol-7") still resolve to the correct emojis via `get_caste_emoji()`. There are no longer dedicated agent files for these castes.
