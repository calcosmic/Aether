# Aether XML Migration Master Plan

**Version:** 1.0
**Date:** 2026-02-16
**Status:** Research Complete - Ready for Implementation Planning

---

## Executive Summary

After extensive multi-agent research into the Aether system, JSON vs XML trade-offs, and multi-repository data sharing patterns, this document synthesizes findings into a comprehensive migration strategy.

**Key Finding:** Aether should adopt a **hybrid JSON/XML architecture** rather than a full migration:
- **Keep JSON** for operational, colony-local data (performance, jq tooling)
- **Adopt XML** for cross-colony eternal memory (namespaces, validation, document composition)

---

## Research Summary

### Agent 1: Aether System Architecture Analysis
**Focus:** Current JSON usage patterns and pain points

**Key Findings:**
- 15+ JSON file types across the system
- Heavy reliance on complex jq queries (3000+ lines in aether-utils.sh)
- 6 major pain points identified where XML would help:
  1. Complex jq queries for simple lookups
  2. No schema validation
  3. Cross-colony data merging complexity
  4. Namespace collision risks
  5. Schema evolution challenges
  6. QUEEN.md mixed format complexity

### Agent 2: JSON vs XML Trade-offs
**Focus:** Technical comparisons and industry patterns

**Key Findings:**
- JSON excels in web APIs, shell scripting (jq), JavaScript ecosystems
- XML dominates in document formats, enterprise integration, strict validation
- Modern systems increasingly use **hybrid approaches**
- Aether's JSON usage is well-justified for operational data
- XML makes sense for prompts (as noted in TO-DOS.md) and cross-colony sharing

### Agent 3: XML Technologies for Aether
**Focus:** Specific XML tech (XSD, Namespaces, XPath, XInclude, XSLT)

**Key Findings:**
- XSD provides superior type safety with native enumerations
- XML Namespaces enable safe pheromone merging between colonies
- XPath/XQuery simplify complex lineage queries
- XInclude enables modular worker priming
- XSLT can automate QUEEN.md generation

### Agent 4: Multi-Repository Data Sharing
**Focus:** Cross-repo patterns from npm, Cargo, Bazel, Maven

**Key Findings:**
- Cargo's workspace inheritance maps to colony parent-child relationships
- Bazel's repo_mapping provides pattern for pheromone overrides
- Maven's groupId:artifactId offers proven naming for instincts
- Content-addressable storage (Git-style) enables chamber deduplication
- 5 key recommendations for Aether's shared data layer

---

## Current Architecture (JSON-Only)

### Data Files

| File | Location | Purpose | Pain Level |
|------|----------|---------|------------|
| COLONY_STATE.json | .aether/data/ | Central colony state | HIGH |
| constraints.json | .aether/data/ | Focus/redirect signals | LOW |
| pheromones.json | .aether/data/ | Active trails | MEDIUM |
| registry.json | ~/.aether/ | Multi-repo tracking | LOW |
| QUEEN.md | .aether/docs/ | Eternal wisdom | HIGH |
| chambers/*/ | .aether/chambers/ | Archive integrity | MEDIUM |

### Tooling
- **jq** - Universal JSON processor (excellent shell integration)
- **No schema validation** - Ad-hoc type checking only
- **Manual merging** - Complex bash/jq for cross-file operations

---

## Proposed Hybrid Architecture

### JSON Zone (Keep As-Is)
**Rationale:** Performance, tooling, no mixed content needs

```
.aether/data/
├── COLONY_STATE.json          # Operational state
├── constraints.json           # Active signals
├── activity.log              # Event stream
└── flags.json                # Blockers/issues
```

### XML Zone (New)
**Rationale:** Cross-colony sharing, validation, document composition

```
~/.aether/eternal/            # Cross-colony eternal memory
├── pheromones.xml            # Namespaced trails
├── queen-wisdom.xml          # Validated wisdom
├── registry.xml              # Multi-colony lineage
├── stack-profile/
│   ├── nodejs.xml
│   ├── python.xml
│   └── rust.xml
└── schema/
    ├── pheromone.xsd
    ├── queen-wisdom.xsd
    └── registry.xsd
```

---

## High-Value XML Migration Targets

### 1. Pheromone Exchange Format (HIGHEST PRIORITY)

**Current Problem:**
- JSON pheromones from Colony A and Colony B merge unpredictably
- No way to track which colony created a signal
- Potential for ID collisions

**XML Solution:**
```xml
<!-- ~/.aether/eternal/pheromones.xml -->
<pheromone-trails
    xmlns="http://aether.dev/pheromones/v1"
    xmlns:colony-a="http://aether.dev/colony/session_123"
    xmlns:colony-b="http://aether.dev/colony/session_456"
    xmlns:repo="http://aether.dev/repo/external">

    <trail id="phem_001" type="PHILOSOPHY" decay="never">
        <substance>emergence-over-orchestration</substance>
        <strength>1.0</strength>
        <source colony:id="colony-001" repo:url="github.com/user/project">
            <command>/ant:init</command>
            <timestamp>2026-02-15T10:30:00Z</timestamp>
        </source>
    </trail>

    <!-- External colony pheromone with namespace -->
    <repo:external-trail repo:source="shared-library">
        <repo:pattern>error-handling</repo:pattern>
    </repo:external-trail>
</pheromone-trails>
```

**Benefits:**
- Namespaces prevent collisions when merging
- XPath queries: `//aether:trail[@type='FOCUS']`
- XSD validation ensures valid pheromone types

### 2. QUEEN.md Evolution (HIGH PRIORITY)

**Current Problem:**
- Markdown + embedded JSON metadata
- Complex parsing with sed/awk/jq
- Mixed content handling is fragile

**XML Solution:**
```xml
<!-- ~/.aether/eternal/queen-wisdom.xml -->
<queen-wisdom version="1.0" evolved-at="2026-02-15T13:08:40Z">
    <philosophies threshold="5">
        <philosophy validated="5" colony="auth-system" timestamp="2026-02-15T13:08:24Z">
            <belief>Test-driven development ensures quality</belief>
            <evidence>
                <colony-ref id="colony-001" phase="3"/>
                <colony-ref id="colony-002" phase="1"/>
            </evidence>
        </philosophy>
    </philosophies>

    <redirects threshold="2">
        <redirect strength="0.9" colony="db-migration">
            <pattern>sync-file-operations</pattern>
            <reason>Caused race conditions in 3 colonies</reason>
            <alternative>Use async fs/promises</alternative>
        </redirect>
    </redirects>

    <!-- Evolution tracking built-in -->
    <evolution-log>
        <entry timestamp="2026-02-15" action="promoted" type="philosophy" from="pattern"/>
    </evolution-log>
</queen-wisdom>
```

**Benefits:**
- XPath: `//philosophy[evidence/colony-ref]` finds cross-colony validated wisdom
- XSLT generates markdown: `xsltproc queen-to-md.xsl queen-wisdom.xml`
- Schema enforces promotion thresholds

### 3. Multi-Colony Registry (MEDIUM PRIORITY)

**Current Problem:**
- registry.json tracks repos but not lineage
- No parent-child relationships
- Chamber archives are flat

**XML Solution:**
```xml
<!-- ~/.aether/eternal/registry.xml -->
<colony-registry current="colony-002" xmlns="http://aether.dev/registry/v1">
    <colony id="colony-001" status="sealed" sealed-at="2026-02-14T02:39:00Z">
        <goal>v1.1 Bug Fixes</goal>
        <lineage>
            <parent>null</parent>
            <children>
                <child ref="colony-002"/>
            </children>
        </lineage>
        <pheromones-extracted count="12"/>
    </colony>

    <colony id="colony-002" status="active">
        <goal>Phase 1: Pheromone Foundation</goal>
        <lineage>
            <parent ref="colony-001"/>
            <inheritance>
                <pheromone type="PHILOSOPHY">emergence</pheromone>
                <pheromone type="PHILOSOPHY">minimal-change</pheromone>
            </inheritance>
        </lineage>
    </colony>
</colony-registry>
```

**Benefits:**
- Natural hierarchy expression
- XPath lineage: `//colony[@id='X']/lineage//child`
- Inheritance tracking

### 4. Worker Spawn-Time Priming (MEDIUM PRIORITY)

**Current Problem:**
- Workers need multiple config sources merged
- Manual bash concatenation of files

**XML Solution:**
```xml
<!-- worker-priming.xml -->
<worker-context caste="builder" depth="2" xmlns:xi="http://www.w3.org/2001/XInclude">
    <xi:include href="~/.aether/eternal/queen-will.xml"/>

    <active-trails>
        <xi:include href=".aether/data/pheromones.xml"/>
    </active-trails>

    <stack-profile>
        <xi:include href="~/.aether/eternal/stack-profile/nodejs.xml"/>
    </stack-profile>
</worker-context>
```

**Benefits:**
- Declarative composition
- Modular configuration
- Automatic merging at parse time

---

## XML Schema (XSD) Definitions

### Pheromone Schema
```xml
<!-- schema/pheromone.xsd -->
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.dev/pheromones/v1"
           xmlns:aether="http://aether.dev/pheromones/v1">

    <xs:simpleType name="PheromoneType">
        <xs:restriction base="xs:string">
            <xs:enumeration value="FOCUS"/>
            <xs:enumeration value="REDIRECT"/>
            <xs:enumeration value="PHILOSOPHY"/>
            <xs:enumeration value="STACK"/>
            <xs:enumeration value="PATTERN"/>
            <xs:enumeration value="DECREE"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:simpleType name="DecayPeriod">
        <xs:restriction base="xs:string">
            <xs:pattern value="\d+d|never"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:simpleType name="Strength">
        <xs:restriction base="xs:decimal">
            <xs:minInclusive value="0.0"/>
            <xs:maxInclusive value="1.0"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:complexType name="Trail">
        <xs:sequence>
            <xs:element name="substance" type="xs:string"/>
            <xs:element name="strength" type="aether:Strength"/>
            <xs:element name="source" type="aether:Source"/>
            <xs:element name="context" type="aether:Context" minOccurs="0"/>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="required"/>
        <xs:attribute name="type" type="aether:PheromoneType" use="required"/>
        <xs:attribute name="decay" type="aether:DecayPeriod" default="30d"/>
    </xs:complexType>
</xs:schema>
```

---

## Migration Strategy: 4-Phase Approach

### Phase 1: Foundation (Week 1-2)
**Goal:** XML support infrastructure

1. Add XML validation utilities to aether-utils.sh
   - `xml-validate` command (xmllint wrapper)
   - `xml-query` command (xpath wrapper)
   - `xml-convert` command (json2xml/xml2json)

2. Create XSD schemas
   - pheromone.xsd
   - queen-wisdom.xsd
   - registry.xsd

3. Install XML tooling
   - xmllint (usually pre-installed)
   - xmlstarlet (for complex transformations)

### Phase 2: Pheromone XML (Week 3-4)
**Goal:** Cross-colony pheromone sharing

1. Create XML export for pheromones
   - `aether pheromones export` → ~/.aether/eternal/pheromones.xml
   - Convert existing pheromones.json

2. Add namespace support
   - Generate colony-specific namespace URIs
   - Prefix external colony pheromones

3. Update worker priming
   - Support XInclude for config composition
   - Maintain backward compatibility

### Phase 3: QUEEN.md Evolution (Week 5-6)
**Goal:** Structured wisdom with validation

1. Create queen-wisdom.xml schema
2. Implement XSLT for markdown generation
3. Add promotion workflow with validation
4. Deprecate mixed-format QUEEN.md (keep for display)

### Phase 4: Registry & Lineage (Week 7-8)
**Goal:** Multi-colony tracking

1. Create registry.xml with lineage support
2. Implement chamber inheritance
3. Add colony ancestry queries
4. Cross-colony pheromone discovery

---

## Tooling Requirements

### New Dependencies

| Tool | Purpose | Install Command |
|------|---------|-----------------|
| xmllint | Validation, XPath | Usually pre-installed (libxml2) |
| xmlstarlet | Transformations | `brew install xmlstarlet` / `apt install xmlstarlet` |
| xsltproc | XSLT processing | Usually pre-installed |

### Shell Integration

```bash
# Add to aether-utils.sh

xml-validate() {
    # Validate XML against XSD
    xmllint --schema "$2" --noout "$1" 2>&1
}

xml-query() {
    # XPath query with namespace support
    xmllint --xpath "$2" "$1" 2>/dev/null
}

xml-convert() {
    # json2xml or xml2json
    # Implementation using xmlstarlet or custom
}
```

---

## XPath vs jq Comparison

### Operation: All FOCUS trails

**jq (JSON):**
```bash
jq '.trails[] | select(.type == "FOCUS")' pheromones.json
```

**XPath (XML):**
```bash
xmllint --xpath "//aether:trail[@type='FOCUS']" pheromones.xml
```

### Operation: Strong trails (>0.7)

**jq (JSON):**
```bash
jq '.trails[] | select(.strength > 0.7)' pheromones.json
```

**XPath (XML):**
```bash
xmllint --xpath "//aether:trail[number(strength) > 0.7]" pheromones.xml
```

### Operation: Colony lineage

**jq (JSON):**
```bash
# Complex reduce required
jq '[.. | objects | select(.parent?)] | group_by(.parent)' registry.json
```

**XPath (XML):**
```bash
xmllint --xpath "//colony[@id='X']/lineage//child" registry.xml
```

---

## Trade-offs Summary

| Aspect | JSON (Current) | XML (Proposed) | Winner |
|--------|---------------|----------------|--------|
| **Tooling** | jq ubiquitous | xmllint less common | JSON |
| **Shell integration** | Native jq support | Requires XMLStarlet | JSON |
| **Human readability** | Excellent | Poor for casual editing | JSON |
| **AI parsing** | Excellent (training-dense) | Good (explicit structure) | Tie |
| **Validation** | JSON Schema (optional) | XSD (strict, powerful) | XML |
| **Namespaces** | N/A | Critical for multi-colony | XML |
| **Lineage queries** | Complex | Natural hierarchy | XML |
| **Mixed content** | Awkward | Natural (QUEEN.md) | XML |
| **Performance** | Faster for small docs | Better for large docs | Tie |

---

## Recommendations

### R1: Hybrid Approach (ADOPTED)
Keep JSON for colony-local operational data, use XML for cross-colony eternal memory.

### R2: Incremental Migration
Don't convert existing files immediately. Add XML support alongside JSON, migrate gradually.

### R3: Schema-First Development
Define XSD schemas before implementing features. Validate all XML at boundaries.

### R4: Namespace Strategy
Use reverse-DNS naming: `http://aether.dev/{component}/{version}`

### R5: Tooling Abstraction
Wrap XML tools in aether-utils.sh commands. Don't expose raw xmllint/xmlstarlet to users.

---

## Next Steps

1. **Review this plan** - Get feedback on hybrid approach
2. **Implement Phase 1** - XML foundation utilities
3. **Prototype pheromone XML** - Test cross-colony sharing
4. **Create schemas** - Formalize structure with XSD
5. **Update documentation** - Reflect new hybrid architecture

---

## Sources

- Aether System Architecture Analysis (Agent: Scout-Aether)
- JSON vs XML Trade-offs Research (Agent: Scout-JSON-XML)
- XML Technologies for Aether (Agent: Scout-XML-Tech)
- Multi-Repository Data Sharing Patterns (Agent: Scout-Vesper-7)
- Aether codebase: aether-utils.sh, COLONY_STATE.json, pheromones.md
- Industry patterns: npm, Cargo, Bazel, Maven

---

*This document represents the synthesis of extensive multi-agent research into the feasibility and strategy for XML adoption in the Aether system.*
