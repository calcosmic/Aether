# Phase 8: XML Integration - Context

**Gathered:** 2026-02-18
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire existing XML exchange utilities (pheromone-xml.sh, wisdom-xml.sh, registry-xml.sh) into the Aether system so they are actually used, not just written. The XML tools already exist and are functional -- this phase connects them to the live system.

**Key discovery from research:** The exchange scripts exist in `.aether/exchange/` with full import/export/validate/merge capabilities, but NOTHING in the system sources or calls them. The pheromone system runs entirely on JSON (pheromones.json + constraints.json). XML is currently dead code.

</domain>

<decisions>
## Implementation Decisions

### Integration scope (auto-decided)
- XML as **exchange format** only, not primary storage
- JSON remains the internal storage format (pheromones.json, constraints.json) -- this works and changing it adds risk
- XML used for: cross-colony export, archival, and data portability
- The exchange scripts become the bridge: JSON -> XML for export, XML -> JSON for import

### Pheromone XML (XML-01)
- Wire `pheromone-xml.sh` into `aether-utils.sh` as new subcommands: `pheromone-export-xml` and `pheromone-import-xml`
- Export: reads pheromones.json, calls xml-pheromone-export, writes to .aether/exchange/pheromones.xml
- Import: reads XML file, calls xml-pheromone-import, merges into pheromones.json
- Validate: calls xml-pheromone-validate against XSD schema
- Integration point: `/ant:entomb` can export pheromones as XML in the archive

### Wisdom XML (XML-02)
- Wire `wisdom-xml.sh` into `aether-utils.sh` as subcommands: `wisdom-export-xml` and `wisdom-import-xml`
- Queen wisdom (QUEEN.md or a JSON equivalent) can be exported/imported via XML
- Integration point: `/ant:seal` or `/ant:entomb` can include wisdom XML in the archive
- Promotion logic (pattern -> philosophy) already built, just needs a caller

### Registry XML (XML-03)
- Wire `registry-xml.sh` into `aether-utils.sh` as subcommands: `registry-export-xml` and `registry-import-xml`
- Colony registry tracks lineage across sealed/entombed colonies
- Integration point: `/ant:tunnels` can use registry XML for cross-chamber queries
- Lineage queries already implemented, just needs to be called from somewhere

### Verification approach
- Test each exchange script's round-trip: JSON -> XML -> JSON should produce equivalent data
- Verify xmllint is available on macOS (comes with Xcode command line tools)
- Graceful degradation: if xmllint/xmlstarlet not available, XML features disabled with warning

### Claude's Discretion
- Exact subcommand naming conventions
- Whether to add XML export to entomb/seal automatically or make it opt-in
- Error message formatting for missing XML tools
- Whether registry needs a persistent JSON backing file or is generated on-demand from chamber data

</decisions>

<specifics>
## Specific Ideas

- Exchange scripts already handle namespace prefixes for cross-colony merge (colony_prefix:signal_id pattern)
- XSD schemas exist at `.aether/schemas/` with examples -- validation infrastructure is ready
- `xml-core.sh` provides the foundation (escape, validate, format, well-formed checks)
- The merge function (`xml-pheromone-merge`) is specifically designed for combining signals from multiple colonies -- this is the real value of XML in Aether

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 08-xml-integration*
*Context gathered: 2026-02-18*
