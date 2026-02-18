# Phase 8: XML Integration - Context

**Gathered:** 2026-02-18
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire existing XML exchange utilities (pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh) into the Aether system so colony knowledge becomes portable. XML serves as the exchange/archival format — JSON remains primary storage. The exchange scripts already exist and are functional; this phase connects them to the live system at seal and entomb milestones.

</domain>

<decisions>
## Implementation Decisions

### When XML triggers
- XML export happens automatically on both `/ant:seal` AND `/ant:entomb` — no manual step needed
- No standalone export command — XML only fires at milestone moments (seal/entomb)
- No mid-session snapshot capability — keep it clean, XML is for archival

### Claude's Discretion: trigger feedback
- Whether to show a brief confirmation line or stay silent when XML export completes during seal/entomb

### Cross-colony import
- Merge is additive — imported signals get added alongside current ones, nothing is overwritten
- On conflict (same area, opposite advice), current colony's signals win
- Imported signals are tagged with their source colony name so you can see where they came from
- Import happens from `/ant:tunnels` — browse old colonies and import from there, no separate import command

### Archive contents
- All three data types included: pheromones, wisdom, and colony registry
- Single combined XML file (not separate files per type) — one colony-archive.xml with sections
- Only active signals included — expired/inactive signals are excluded for a cleaner archive
- XML archive file stored in the chamber alongside other archive files (.aether/chambers/{colony}/)

### Missing tools fallback
- Check for XML tools before every XML operation (not just once at init)
- If XML export fails during entomb, entomb STOPS — don't archive without the XML component
- If tools are missing, offer to install them ("Want me to install xmllint?") rather than just showing the command
- Discover XML readiness at entomb time — no proactive health check in /ant:status

### Claude's Discretion
- Whether to require xmllint only or also support xmlstarlet as a fallback
- Exact subcommand naming conventions for the aether-utils.sh integration
- Error message formatting and wording
- Whether registry needs a persistent JSON backing file or is generated on-demand from chamber data
- Combined XML file structure and section ordering

</decisions>

<specifics>
## Specific Ideas

- Exchange scripts already handle namespace prefixes for cross-colony merge (colony_prefix:signal_id pattern)
- XSD schemas exist at `.aether/schemas/` with examples — validation infrastructure is ready
- `xml-core.sh` provides the foundation (escape, validate, format, well-formed checks)
- The merge function (`xml-pheromone-merge`) is specifically designed for combining signals from multiple colonies — this is the real value of XML in Aether
- Import flow: `/ant:tunnels` → browse chamber → select colony → import wisdom/signals into current colony

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-xml-integration*
*Context gathered: 2026-02-18*
