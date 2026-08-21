# XML exchange — disposition (RECLAIM-03)

**Decision date:** 2026-08-16
**Status:** ruled — the pheromone lane lives, the rest is marked for
retirement instead of staying half-alive.

The XML exchange surface splits into two very different halves:

**Alive and wired — keep:** the pheromone signal lane.
`/ant-export-signals` and `/ant-import-signals` are real wrapper commands with
runtime backing; cross-colony signal sharing is the one exchange flow an
operator actually uses.

**Half-alive — retire in a dedicated deletion change:** the archival lane.
`colony-archive-xml`, `registry-export-xml`, `registry-import-xml`,
`wisdom-export-xml`, `wisdom-import-xml`, `pheromone-validate-xml` have no
caller anywhere. The v5.4.0 behaviour they mirror (a colony archive at seal)
is served today by `CROWNED-ANTHILL.md` plus `aether entomb`'s chamber
archive, and cross-colony wisdom moved to the Hive Brain (`hive-promote` /
`hive-read`, JSON, capped and locked) — an XML sidecar of the same data is a
second source of truth that nothing reconciles. Their allowlist reasons carry
this disposition dated; the deletion commit follows the test-deletion ledger
pattern, and takes the stale `.aether/exchange/` fixtures and
`.aether/docs/xml-utilities.md` references with it.

Per RECLAIM-03's own wording this is the "explicitly retired" arm of the
either/or — chosen over wiring `colony-archive-xml` into seal because seal
already produces a durable archive, and duplicating it in a second format
would be new surface, not restoration.
