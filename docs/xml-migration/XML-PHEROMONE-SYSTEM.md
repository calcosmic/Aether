# XML Pheromone System Implementation Guide

**Part of:** XML Migration Master Plan
**Priority:** HIGHEST
**Status:** Design Complete

---

## Overview

The XML Pheromone System enables safe cross-colony pheromone sharing through XML namespaces. This solves the collision problem when merging pheromones from multiple colonies or external sources.

---

## The Problem with JSON Pheromones

### Collision Scenario

```json
// Colony A pheromones.json
{
  "trails": [
    {"id": "phem_001", "type": "FOCUS", "substance": "use-typescript"}
  ]
}

// Colony B pheromones.json
{
  "trails": [
    {"id": "phem_001", "type": "REDIRECT", "substance": "avoid-typescript"}
  ]
}
```

**Result:** Same ID, conflicting advice. Which one wins?

### No Provenance

- Which colony created this signal?
- When was it created?
- How many colonies have validated it?
- What's the confidence based on cross-colony usage?

---

## XML Solution: Namespaced Pheromones

### Basic Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="http://aether.dev/pheromones/v1"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://aether.dev/pheromones/v1
                        http://aether.dev/schema/pheromone.xsd"
    generated="2026-02-16T15:30:00Z"
    colony-id="colony-abc123"
    source-repo="github.com/user/project">

    <!-- Colony's own pheromones (default namespace) -->
    <trail id="phem_001" type="PHILOSOPHY" decay="never">
        <substance>emergence-over-orchestration</substance>
        <strength>1.0</strength>
        <source>
            <command>/ant:init</command>
            <timestamp>2026-02-15T10:30:00Z</timestamp>
            <context>API design for microservices</context>
        </source>
        <validation-count>5</validation-count>
    </trail>

    <trail id="phem_002" type="FOCUS" decay="30d">
        <substance>error-handling-edge-cases</substance>
        <strength>0.85</strength>
        <source>
            <command>/ant:swarm</command>
            <timestamp>2026-02-16T14:22:00Z</timestamp>
        </source>
    </trail>

</pheromone-trails>
```

### Multi-Colony Merge

```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="http://aether.dev/pheromones/v1"
    xmlns:colony-a="http://aether.dev/colony/abc123"
    xmlns:colony-b="http://aether.dev/colony/def456"
    xmlns:template="http://aether.dev/template/nodejs-api"
    xmlns:shared="http://aether.dev/shared/best-practices">

    <!-- Colony A's signals -->
    <colony-a:trail id="phem_001" type="PHILOSOPHY">
        <colony-a:substance>emergence-over-orchestration</colony-a:substance>
        <colony-a:strength>1.0</colony-a:strength>
    </colony-a:trail>

    <!-- Colony B's signals -->
    <colony-b:trail id="phem_001" type="PHILOSOPHY">
        <colony-b:substance>minimal-surprise-principle</colony-b:substance>
        <colony-b:strength>0.95</colony-b:strength>
    </colony-b:trail>

    <!-- Template repo signals -->
    <template:trail id="pattern_001" type="PATTERN">
        <template:substance>express-route-handlers</template:substance>
        <template:applies-to>nodejs</template:applies-to>
    </template:trail>

    <!-- Shared community signals -->
    <shared:trail id="community_001" type="REDIRECT">
        <shared:substance>avoid-sync-fs</shared:substance>
        <shared:reason>Deprecated in Node.js</shared:reason>
    </shared:trail>

</pheromone-trails>
```

**Benefits:**
- No ID collisions - each colony has its own namespace
- Clear provenance - know which colony contributed each signal
- Selective inheritance - choose which namespaces to apply
- XPath filtering - `//colony-a:trail` vs `//shared:trail`

---

## XSD Schema

### pheromone.xsd

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.dev/pheromones/v1"
           xmlns:aether="http://aether.dev/pheromones/v1"
           elementFormDefault="qualified">

    <!-- Enumerations -->
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

    <!-- Complex Types -->
    <xs:complexType name="Source">
        <xs:sequence>
            <xs:element name="command" type="xs:string"/>
            <xs:element name="timestamp" type="xs:dateTime"/>
            <xs:element name="context" type="xs:string" minOccurs="0"/>
        </xs:sequence>
        <xs:attribute name="colony-id" type="xs:string"/>
        <xs:attribute name="repo-url" type="xs:anyURI"/>
    </xs:complexType>

    <xs:complexType name="Trail">
        <xs:sequence>
            <xs:element name="substance" type="xs:string"/>
            <xs:element name="strength" type="aether:Strength"/>
            <xs:element name="source" type="aether:Source"/>
            <xs:element name="context" type="xs:string" minOccurs="0"/>
            <xs:element name="validation-count" type="xs:nonNegativeInteger" minOccurs="0"/>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="required"/>
        <xs:attribute name="type" type="aether:PheromoneType" use="required"/>
        <xs:attribute name="decay" type="aether:DecayPeriod" default="30d"/>
    </xs:complexType>

    <!-- Root Element -->
    <xs:element name="pheromone-trails">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="trail" type="aether:Trail" maxOccurs="unbounded"/>
            </xs:sequence>
            <xs:attribute name="generated" type="xs:dateTime"/>
            <xs:attribute name="colony-id" type="xs:string"/>
            <xs:attribute name="source-repo" type="xs:anyURI"/>
        </xs:complexType>
    </xs:element>

</xs:schema>
```

---

## Shell Integration

### aether-utils.sh Commands

```bash
#!/bin/bash
# XML Pheromone commands for aether-utils.sh

PHEROMONE_NS="http://aether.dev/pheromones/v1"
PHEROMONE_XSD="~/.aether/schema/pheromone.xsd"

# Export pheromones to XML
pheromone-export() {
    local colony_id="${1:-$(get-colony-id)}"
    local output_file="${2:-~/.aether/eternal/pheromones.xml}"

    # Read JSON pheromones
    local json_file=".aether/data/pheromones.json"

    # Convert to XML using jq + printf
    cat > "$output_file" << XML_HEADER
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="$PHEROMONE_NS"
    generated="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    colony-id="$colony_id">
XML_HEADER

    # Convert each trail
    jq -r '.trails[] | "
    <trail id=\"\(.id)\" type=\"\(.type)\" decay=\"\(.decay // "30d")\">
        <substance>\(.substance)</substance>
        <strength>\(.strength)</strength>
        <source>
            <command>\(.source.command)</command>
            <timestamp>\(.source.timestamp)</timestamp>
        </source>
    </trail>"' "$json_file" >> "$output_file"

    echo "</pheromone-trails>" >> "$output_file"

    # Validate
    if xmllint --schema "$PHEROMONE_XSD" --noout "$output_file" 2>/dev/null; then
        json_ok "{\"exported\": true, \"file\": \"$output_file\"}"
    else
        json_err "E_VALIDATION_FAILED" "Exported XML failed schema validation"
    fi
}

# Query pheromones with XPath
pheromone-query() {
    local xpath="$1"
    local xml_file="${2:-~/.aether/eternal/pheromones.xml}"

    # Add namespace binding
    local ns_bind="ns=$PHEROMONE_NS"

    result=$(xmllint --xpath "//$xpath" "$xml_file" 2>/dev/null)

    if [[ -n "$result" ]]; then
        echo "$result"
        return 0
    else
        echo "No matches found"
        return 1
    fi
}

# Merge pheromones from another colony
pheromone-merge() {
    local source_xml="$1"
    local target_xml="${2:-~/.aether/eternal/pheromones.xml}"
    local colony_ns="$3"  # e.g., "colony-abc123"

    # Extract source colony ID
    local source_id=$(xmllint --xpath "string(/*/@colony-id)" "$source_xml")

    # Create namespace prefix
    local prefix="colony-${source_id}"

    # Transform to add namespace prefix
    xmlstarlet ed \
        -N "aether=$PHEROMONE_NS" \
        -r "//aether:trail" -v "${prefix}:trail" \
        "$source_xml" > "${target_xml}.tmp"

    # Append to target
    # (Implementation depends on merge strategy)

    rm "${target_xml}.tmp"
}

# List all FOCUS trails
pheromone-list-focus() {
    local xml_file="${1:-~/.aether/eternal/pheromones.xml}"

    xmllint --xpath "//trail[@type='FOCUS']/substance/text()" "$xml_file" 2>/dev/null | tr '\n' ' '
}

# Get pheromone by ID
pheromone-get() {
    local id="$1"
    local xml_file="${2:-~/.aether/eternal/pheromones.xml}"

    xmllint --xpath "//trail[@id='$id']" "$xml_file" 2>/dev/null
}
```

---

## XPath Query Examples

### All FOCUS trails
```bash
xmllint --xpath "//trail[@type='FOCUS']" pheromones.xml
```

### Strong trails (>0.8 strength)
```bash
xmllint --xpath "//trail[number(strength) > 0.8]" pheromones.xml
```

### Never-decay philosophies
```bash
xmllint --xpath "//trail[@type='PHILOSOPHY' and @decay='never']" pheromones.xml
```

### Cross-colony validation
```bash
# Trails validated by 3+ colonies
xmllint --xpath "//trail[number(validation-count) >= 3]" merged-pheromones.xml
```

### Namespace-filtered
```bash
# Only template repo patterns
xmllint --xpath "//template:trail" merged-pheromones.xml
```

---

## Integration with Commands

### /ant:focus

**Current (JSON):**
```bash
echo '{"type": "FOCUS", "content": "..."}' >> .aether/data/pheromones.json
```

**New (XML):**
```bash
# Add to local XML
xmlstarlet ed \
    -s "/pheromone-trails" -t elem -n "trail" \
    -i "//trail[last()]" -t attr -n "id" -v "phem_$(date +%s)" \
    -i "//trail[last()]" -t attr -n "type" -v "FOCUS" \
    -s "//trail[last()]" -t elem -n "substance" -v "$content" \
    pheromones.xml > pheromones.xml.tmp
mv pheromones.xml.tmp pheromones.xml

# Export to eternal for sharing
aether-utils.sh pheromone-export
```

### Worker Priming

**XML with XInclude:**
```xml
<!-- worker-context.xml -->
<worker-context xmlns:xi="http://www.w3.org/2001/XInclude">
    <configuration>
        <xi:include href="~/.aether/eternal/pheromones.xml"
                    xpointer="xpointer(//trail[@type='PHILOSOPHY'])"/>
    </configuration>
</worker-context>
```

---

## Migration Path

### Phase 1: Export
- Convert existing `pheromones.json` to `pheromones.xml`
- Maintain both formats during transition

### Phase 2: Dual Write
- All new pheromones written to both JSON and XML
- Queries can use either format

### Phase 3: XML Primary
- Switch to XML as primary format
- Keep JSON for backward compatibility (read-only)

### Phase 4: JSON Deprecation
- Remove JSON support
- Full XML with namespaces

---

## Benefits Summary

| Feature | JSON | XML | Impact |
|---------|------|-----|--------|
| Cross-colony merging | ❌ Collisions | ✅ Namespaces | Critical |
| Provenance tracking | ❌ Manual | ✅ Built-in | High |
| Schema validation | ⚠️ Optional | ✅ Strict | High |
| XPath queries | ❌ jq only | ✅ Native | Medium |
| Namespace inheritance | ❌ None | ✅ Hierarchical | High |

---

## Next Steps

1. Implement `pheromone.xsd` schema
2. Add `pheromone-export` and `pheromone-query` commands
3. Create migration script from JSON
4. Update `/ant:focus` and `/ant:redirect` to support XML
5. Document namespace convention

---

*This document is part of the XML Migration Master Plan.*
