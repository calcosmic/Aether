# JSON vs XML Trade-offs for Aether

**Reference Document:** Comprehensive analysis of JSON vs XML for the Aether system
**Based on:** Multi-agent research across technical specs, real-world systems, and industry patterns

---

## Executive Summary

| Aspect | JSON | XML | Aether's Need |
|--------|------|-----|---------------|
| **Operational Data** | ✅ Excellent | ⚠️ Overkill | **JSON** - Colony state, flags, activity logs |
| **Cross-Colony Sharing** | ❌ Poor | ✅ Excellent | **XML** - Pheromones, wisdom, registry |
| **Schema Validation** | ⚠️ Optional | ✅ Strict | **XML** - Eternal memory needs validation |
| **Shell Scripting** | ✅ jq native | ⚠️ Extra tools | **JSON** - jq is ubiquitous |
| **Namespace Support** | ❌ None | ✅ Native | **XML** - Multi-colony collision prevention |
| **Document Composition** | ⚠️ Manual | ✅ XInclude | **XML** - Worker priming, modular config |
| **Mixed Content** | ❌ Awkward | ✅ Native | **XML** - QUEEN.md evolution |

**Recommendation:** Hybrid architecture - JSON for local operational data, XML for shared eternal memory.

---

## Detailed Comparison

### 1. Schema Validation

#### JSON Schema (Current)
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "state": {
      "type": "string",
      "enum": ["READY", "EXECUTING", "PLANNING"]
    },
    "confidence": {
      "type": "number",
      "minimum": 0.0,
      "maximum": 1.0
    }
  },
  "required": ["state"]
}
```

**Pros:**
- Wide tooling support
- JavaScript-native
- Can be embedded in JS apps

**Cons:**
- Validation is optional (easy to skip)
- Less formally rigorous than XSD
- No native support in most shell environments

#### XSD (XML Schema Definition)
```xml
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ColonyState">
    <xs:restriction base="xs:string">
      <xs:enumeration value="READY"/>
      <xs:enumeration value="EXECUTING"/>
      <xs:enumeration value="PLANNING"/>
    </xs:restriction>
  </xs:simpleType>

  <xs:simpleType name="Confidence">
    <xs:restriction base="xs:decimal">
      <xs:minInclusive value="0.0"/>
      <xs:maxInclusive value="1.0"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>
```

**Pros:**
- Strict validation (enforced at parse time)
- Superior type system (inheritance, substitution groups)
- Widely supported (`xmllint --schema`)
- Industry standard for enterprise contracts

**Cons:**
- Verbose syntax
- Learning curve
- Extra tooling dependency

**Verdict for Aether:** XML wins for eternal memory (needs strict validation), JSON sufficient for operational data.

---

### 2. Query Languages

#### jq (JSON)
```bash
# Find all critical pathogens
jq '.pathogens[] | select(.severity == "critical")' pathogens.json

# Count blockers by phase
jq '[.flags[] | select(.type == "blocker")] | group_by(.phase) | map({phase: .[0].phase, count: length})'

# Find high-confidence instincts
jq '.memory.instincts[] | select(.confidence >= 0.8 and .applications == 0)'
```

**Pros:**
- Excellent shell integration
- Intuitive syntax for simple queries
- Native to JSON workflow
- Fast for small documents

**Cons:**
- Complex queries become unreadable
- Limited aggregation capabilities
- No standardization (jq-specific)

#### XPath/XQuery (XML)
```bash
# Find all critical pathogens
xmllint --xpath "//pathogen[severity='critical']" pathogens.xml

# Count blockers by phase (simpler)
xmllint --xpath "//flag[@type='blocker']/@phase" flags.xml | sort | uniq -c

# Complex query with conditions
xmllint --xpath "//instinct[confidence >= 0.8 and applications = 0]" instincts.xml
```

**Pros:**
- W3C standardized
- Powerful for hierarchical data
- Excellent for document traversal
- XSLT for transformations

**Cons:**
- Verbose syntax
- Requires separate tooling (xmllint, xmlstarlet)
- Less ergonomic for shell scripting

**Verdict for Aether:** JSON/jq wins for operational queries (familiarity, shell integration). XML/XPath wins for cross-colony lineage queries (hierarchical data).

---

### 3. Namespace Support

#### JSON
```json
{
  "trails": [
    {"id": "phem_001", "source": "colony-a", "type": "FOCUS"},
    {"id": "phem_001", "source": "colony-b", "type": "REDIRECT"}
  ]
}
```

**Problem:** Same ID collision. Manual source tracking. No collision prevention.

**Workarounds:**
- Prefix IDs: `colony-a-phem_001`
- Add source field (enforced by convention)
- UUIDs (opaque, hard to debug)

#### XML
```xml
<pheromones xmlns:colony-a="http://aether.dev/colony/a"
            xmlns:colony-b="http://aether.dev/colony/b">
  <colony-a:trail id="phem_001" type="FOCUS"/>
  <colony-b:trail id="phem_001" type="REDIRECT"/>
</pheromones>
```

**Benefits:**
- No ID collisions
- Clear provenance
- Hierarchical naming
- XPath filtering by namespace

**Verdict for Aether:** XML wins decisively. Multi-colony pheromone sharing requires namespace support.

---

### 4. Document Composition

#### JSON
```json
{
  "config": {
    "eternal": { /* copied from ~/.aether/eternal/queen-will.md */ },
    "local": { /* from .aether/data/pheromones.json */ },
    "stack": { /* from ~/.aether/eternal/stack-profile/ */ }
  }
}
```

**Problem:** Manual merging. Copy-paste updates. No single source of truth.

#### XML with XInclude
```xml
<worker-context xmlns:xi="http://www.w3.org/2001/XInclude">
  <xi:include href="~/.aether/eternal/queen-will.xml"/>
  <xi:include href=".aether/data/pheromones.xml"/>
  <xi:include href="~/.aether/eternal/stack-profile/nodejs.xml"/>
</worker-context>
```

**Benefits:**
- Declarative composition
- Automatic updates when sources change
- Modular configuration
- No duplication

**Verdict for Aether:** XML/XInclude wins. Worker priming needs modular config composition.

---

### 5. Mixed Content

#### JSON
```json
{
  "title": "QUEEN.md",
  "content": "# Philosophies\n\n1. Test-driven development...",
  "metadata": {"version": "1.0"}
}
```

**Problem:** Markdown embedded in JSON strings. Requires escaping. Hard to edit.

#### XML
```xml
<queen-wisdom version="1.0">
  <philosophies>
    <philosophy>
      <title>Test-driven development</title>
      <description>Write tests before implementation.</description>
      <rationale>Ensure quality from the start.</rationale>
    </philosophy>
  </philosophies>
</queen-wisdom>
```

**Benefits:**
- Structured content
- No escaping needed
- XSLT for markdown generation
- Validated structure

**Verdict for Aether:** XML wins. QUEEN.md has complex structure that benefits from XML.

---

### 6. Shell Integration

#### JSON with jq
```bash
# Check if available
command -v jq >/dev/null 2>&1 || echo "jq not installed"

# Parse and extract
jq -r '.goal' .aether/data/COLONY_STATE.json

# Update in place
jq '.state = "EXECUTING"' .aether/data/COLONY_STATE.json > tmp.json && mv tmp.json .aether/data/COLONY_STATE.json
```

**Availability:**
- macOS: `brew install jq` (very common)
- Ubuntu: `apt install jq` (standard package)
- Usually pre-installed in dev environments

#### XML with xmllint/xmlstarlet
```bash
# Check if available
command -v xmllint >/dev/null 2>&1 || echo "xmllint not installed"

# Parse and extract
xmllint --xpath "//goal/text()" .aether/data/COLONY_STATE.xml

# Update (requires xmlstarlet)
xmlstarlet ed -u "//state" -v "EXECUTING" .aether/data/COLONY_STATE.xml
```

**Availability:**
- macOS: `xmllint` pre-installed, `xmlstarlet` via brew
- Ubuntu: `xmllint` in libxml2-utils, `xmlstarlet` via apt
- Less commonly pre-installed than jq

**Verdict for Aether:** JSON/jq wins for shell ergonomics. XML tools are available but require installation.

---

### 7. Performance

| Scenario | JSON | XML | Notes |
|----------|------|-----|-------|
| Small documents (<1MB) | ✅ Fast | ✅ Fast | Negligible difference |
| Large documents (>10MB) | ⚠️ Slower | ✅ Streaming (SAX) | XML can stream without loading full doc |
| Parsing speed | ✅ Faster | ⚠️ Slower | JSON simpler grammar |
| Query speed | Similar | Similar | Depends on implementation |
| Memory usage | Similar | Similar | XML slightly more overhead |

**Verdict for Aether:** Tie. Aether's files are typically small (<100KB). Performance not a deciding factor.

---

### 8. Industry Adoption

#### JSON Dominates
- Web APIs (REST, GraphQL)
- JavaScript/Node.js ecosystems
- NoSQL databases (MongoDB, CouchDB)
- Configuration files (package.json, tsconfig.json)

#### XML Dominates
- Enterprise integration (SOAP, WSDL)
- Document formats (DOCX, ODF, EPUB)
- Publishing (DocBook, TEI)
- Configuration with complex validation (Maven, Android)
- Data interchange with strict contracts (HL7 FHIR, UBL)

#### Hybrid Examples
- **HL7 FHIR:** Same logical model, JSON for APIs, XML for documents
- **OpenAPI:** JSON/YAML for specs, XML for request/response bodies
- **Spring Boot:** YAML/JSON for config, XML for legacy integration

**Verdict for Aether:** Hybrid approach aligns with industry trends. JSON for APIs/ops, XML for documents/cross-org sharing.

---

## Aether-Specific Recommendations

### Keep JSON For

| Use Case | Reason |
|----------|--------|
| `COLONY_STATE.json` | Operational, high churn, jq tooling |
| `constraints.json` | Simple structure, low complexity |
| `activity.log` | Append-only, line-oriented |
| `flags.json` | Fast read/write, simple queries |
| Command inputs/outputs | jq ubiquity, shell scripting |

### Migrate to XML For

| Use Case | Reason |
|----------|--------|
| `~/.aether/eternal/pheromones.xml` | Cross-colony sharing, namespaces |
| `~/.aether/eternal/queen-wisdom.xml` | Mixed content, validation |
| `~/.aether/eternal/registry.xml` | Lineage tracking, hierarchy |
| Chamber manifests | Archive integrity, XInclude |
| Worker priming configs | Modular composition |

---

## Migration Complexity Assessment

| Component | Complexity | Effort | Priority |
|-----------|-----------|--------|----------|
| Pheromone export/import | Low | 2-3 days | HIGH |
| QUEEN.md → XML | Medium | 1 week | HIGH |
| Registry XML | Low | 1-2 days | MEDIUM |
| XSD schemas | Medium | 3-4 days | HIGH |
| Shell tool integration | Low | 2 days | MEDIUM |
| Documentation | Medium | 2-3 days | MEDIUM |
| Testing | Medium | 1 week | HIGH |

**Total estimated effort:** 4-6 weeks for full hybrid implementation

---

## Risk Assessment

### Low Risk
- Adding XML support alongside JSON
- Export/import functionality
- Documentation updates

### Medium Risk
- Changing QUEEN.md format (breaking change)
- Schema validation failures (need graceful degradation)

### High Risk
- Removing JSON support (don't do this immediately)
- Breaking existing colonies (maintain backward compat)

---

## Best Practices Summary

### For JSON (Operational Data)
1. Use JSON Schema for validation (optional but recommended)
2. Prefer jq for shell scripting
3. Keep structures simple and flat
4. Version your schemas
5. Document with examples

### For XML (Eternal Memory)
1. Always use XSD schemas (strict validation)
2. Define namespace conventions early
3. Use XInclude for modular documents
4. Create XSLT for human-readable views
5. Maintain XML → JSON export capability

### For Hybrid Systems
1. Clear boundary: JSON for local, XML for shared
2. Export/import tools for conversion
3. Dual-write during migration period
4. Schema versioning for compatibility
5. Document the rationale for each format

---

## Conclusion

**The hybrid approach is optimal for Aether:**

- **JSON** maintains developer productivity for day-to-day operations
- **XML** enables the cross-colony, cross-repository sharing vision
- Industry trends support this bifurcation
- Migration can be incremental and safe

**Next step:** Begin with XML pheromone export/import as the first proof-of-concept.

---

## Sources

- JSON Schema 2020-12 Core Specification
- XSD 1.1 Specification (W3C)
- XPath 3.1 and XQuery 3.1 Specifications
- jq Manual (jqlang.org)
- RFC 8259 - JSON Data Interchange Format
- XML 1.0 Fifth Edition (W3C)
- Aether codebase analysis
- Industry pattern research (npm, Cargo, Bazel, Maven)

---

*This document is part of the XML Migration Master Plan.*
