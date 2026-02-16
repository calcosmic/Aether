# Shell Integration

**Command-Line Tools for Aether XML**

---

## Overview

Aether provides shell commands for working with XML documents. These wrap tools like `xmllint` and `xmlstarlet` with Aether-specific functionality.

---

## Installation

### Required Tools

| Tool | Purpose | macOS Install | Ubuntu Install |
|------|---------|---------------|----------------|
| `xmllint` | Validation, XPath | Pre-installed | `apt install libxml2-utils` |
| `xmlstarlet` | Transformations | `brew install xmlstarlet` | `apt install xmlstarlet` |
| `xsltproc` | XSLT processing | Pre-installed | `apt install xsltproc` |

### Verify Installation
```bash
# Check xmllint
xmllint --version

# Check xmlstarlet
xmlstarlet --version

# Check xsltproc
xsltproc --version
```

---

## Core Commands

### 1. pheromone-export

Convert JSON pheromones to XML format.

**Usage:**
```bash
pheromone-export [colony-id] [output-file]
```

**Arguments:**
- `colony-id` — Source colony (default: current colony)
- `output-file` — Destination (default: `~/.aether/eternal/pheromones.xml`)

**Example:**
```bash
# Export current colony's pheromones
pheromone-export

# Export specific colony
pheromone-export colony-abc123 ~/exports/colony-a.xml
```

**Implementation:**
```bash
pheromone-export() {
    local colony_id="${1:-$(get-colony-id)}"
    local output_file="${2:-~/.aether/eternal/pheromones.xml}"
    local json_file=".aether/data/pheromones.json"
    local PHEROMONE_NS="http://aether.dev/core/pheromones/v1"

    # Validate JSON exists
    if [[ ! -f "$json_file" ]]; then
        json_err "E_NO_PHEROMONES" "No pheromones.json found"
        return 1
    fi

    # Generate XML header
    cat > "$output_file" << XML_HEADER
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="$PHEROMONE_NS"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="$PHEROMONE_NS http://aether.dev/schema/pheromone.xsd"
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

    # Close root element
    echo "</pheromone-trails>" >> "$output_file"

    # Validate output
    if xmllint --schema ~/.aether/schema/pheromone.xsd --noout "$output_file" 2>/dev/null; then
        json_ok "{\"exported\": true, \"file\": \"$output_file\", \"trails\": $(jq '.trails | length' "$json_file")}"
    else
        json_err "E_VALIDATION_FAILED" "Exported XML failed schema validation"
        return 1
    fi
}
```

**Output:**
```json
{"exported": true, "file": "~/.aether/eternal/pheromones.xml", "trails": 5}
```

---

### 2. pheromone-query

Query pheromones using XPath.

**Usage:**
```bash
pheromone-query <xpath> [xml-file]
```

**Arguments:**
- `xpath` — XPath expression to evaluate
- `xml-file` — Source file (default: `~/.aether/eternal/pheromones.xml`)

**Examples:**
```bash
# Find all FOCUS trails
pheromone-query "//trail[@type='FOCUS']"

# Find strong trails (>0.8)
pheromone-query "//trail[number(strength) > 0.8]"

# Find never-decay philosophies
pheromone-query "//trail[@type='PHILOSOPHY' and @decay='never']"

# Find specific trail by ID
pheromone-query "//trail[@id='phem-001']"

# Count validations
pheromone-query "count(//trail/validations/colony-ref)"
```

**Implementation:**
```bash
pheromone-query() {
    local xpath="$1"
    local xml_file="${2:-~/.aether/eternal/pheromones.xml}"

    # Validate file exists
    if [[ ! -f "$xml_file" ]]; then
        echo "No pheromones file found: $xml_file"
        return 1
    fi

    # Execute XPath query
    local result
    result=$(xmllint --xpath "$xpath" "$xml_file" 2>/dev/null)

    if [[ -n "$result" ]]; then
        echo "$result"
        return 0
    else
        echo "No matches found"
        return 1
    fi
}
```

---

### 3. pheromone-merge

Merge pheromones from another colony.

**Usage:**
```bash
pheromone-merge <source-xml> [target-xml] [--namespace <prefix>]
```

**Arguments:**
- `source-xml` — Source colony's pheromones
- `target-xml` — Destination (default: `~/.aether/eternal/pheromones.xml`)
- `--namespace` — Custom namespace prefix (auto-generated if omitted)

**Examples:**
```bash
# Merge from parent colony
pheromone-merge ~/chambers/payment-v1/pheromones.xml

# Merge with custom namespace
pheromone-merge template.xml --namespace template-nodejs

# Merge into specific file
pheromone-merge colony-b.xml ./my-pheromones.xml
```

**Implementation:**
```bash
pheromone-merge() {
    local source_xml="$1"
    shift
    local target_xml="~/.aether/eternal/pheromones.xml"
    local custom_ns=""

    # Parse optional arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --namespace)
                custom_ns="$2"
                shift 2
                ;;
            *)
                target_xml="$1"
                shift
                ;;
        esac
    done

    # Extract source colony ID
    local source_id
    source_id=$(xmllint --xpath "string(/*/@colony-id)" "$source_xml" 2>/dev/null)

    if [[ -z "$source_id" ]]; then
        json_err "E_INVALID_SOURCE" "Could not extract colony-id from source"
        return 1
    fi

    # Generate namespace prefix
    local ns_prefix="${custom_ns:-colony-${source_id}}"

    # Transform to add namespace prefix
    xmlstarlet ed \
        -N "aether=http://aether.dev/core/pheromones/v1" \
        -r "//aether:trail" -v "${ns_prefix}:trail" \
        "$source_xml" > "${target_xml}.tmp"

    # Append to target (implementation depends on merge strategy)
    # Option 1: Simple append
    # Option 2: Deduplication
    # Option 3: XInclude reference

    # Clean up
    rm "${target_xml}.tmp"

    json_ok "{\"merged\": true, \"source\": \"$source_id\", \"namespace\": \"$ns_prefix\"}"
}
```

---

### 4. xml-validate

Validate XML against XSD schema.

**Usage:**
```bash
xml-validate <xml-file> --schema <schema-name>
```

**Arguments:**
- `xml-file` — XML document to validate
- `--schema` — Schema name (pheromone, queen, registry, context)

**Examples:**
```bash
# Validate pheromones
xml-validate pheromones.xml --schema pheromone

# Validate queen wisdom
xml-validate queen-wisdom.xml --schema queen

# Validate with custom schema path
xml-validate custom.xml --schema /path/to/schema.xsd
```

**Implementation:**
```bash
xml-validate() {
    local xml_file="$1"
    shift
    local schema_path=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --schema)
                if [[ "$2" =~ ^/ ]]; then
                    # Absolute path
                    schema_path="$2"
                else
                    # Named schema
                    schema_path="~/.aether/schema/${2}.xsd"
                fi
                shift 2
                ;;
        esac
    done

    # Validate schema exists
    if [[ ! -f "$schema_path" ]]; then
        json_err "E_SCHEMA_NOT_FOUND" "Schema not found: $schema_path"
        return 1
    fi

    # Validate XML
    local errors
    errors=$(xmllint --schema "$schema_path" --noout "$xml_file" 2>&1)

    if [[ $? -eq 0 ]]; then
        json_ok '{"valid": true, "errors": []}'
    else
        json_err "E_VALIDATION_FAILED" "$errors"
        return 1
    fi
}
```

---

### 5. queen-evolve

Promote validated patterns to QUEEN.md.

**Usage:**
```bash
queen-evolve --from <pheromone-id> --to <wisdom-type> [--threshold <n>]
```

**Arguments:**
- `--from` — Source pheromone ID
- `--to` — Target wisdom type (philosophy, pattern, redirect)
- `--threshold` — Validation threshold (default varies by type)

**Examples:**
```bash
# Promote a pattern to instinct
queen-evolve --from phem-001 --to pattern

# Promote with custom threshold
queen-evolve --from phem-002 --to philosophy --threshold 7

# Dry run (show what would happen)
queen-evolve --from phem-003 --to redirect --dry-run
```

---

## Utility Functions

### Helper: get-colony-id
```bash
get-colony-id() {
    local colony_file=".aether/data/COLONY_STATE.json"
    if [[ -f "$colony_file" ]]; then
        jq -r '.colony_id // empty' "$colony_file"
    else
        echo "unknown-$(date +%s)"
    fi
}
```

### Helper: json_ok / json_err
```bash
json_ok() {
    echo "$1"
}

json_err() {
    local code="$1"
    local message="$2"
    echo "{\"error\": \"$code\", \"message\": \"$message\"}"
}
```

### Helper: xml-format
Pretty-print XML document.
```bash
xml-format() {
    local xml_file="$1"
    xmllint --format "$xml_file" -o "$xml_file.formatted"
    mv "$xml_file.formatted" "$xml_file"
}
```

---

## XPath Reference

### Common Queries

| Task | XPath Expression |
|------|------------------|
| All trails | `//trail` |
| By type | `//trail[@type='FOCUS']` |
| By ID | `//trail[@id='phem-001']` |
| Strong trails | `//trail[number(strength) > 0.8]` |
| Never decay | `//trail[@decay='never']` |
| With context | `//trail[context/domain='web-api']` |
| Validated 3+ times | `//trail[number(validation-count) >= 3]` |
| Specific namespace | `//colony-a:trail` |
| Substance text | `//trail/substance/text()` |
| Count trails | `count(//trail)` |

### With Namespaces
```bash
# Define namespace binding
xmllint --xpath "//ns:trail" \
        --shell pheromones.xml <<EOF
setns ns=http://aether.dev/core/pheromones/v1
setns colony-a=http://aether.dev/colony/abc123/v1
xpath //colony-a:trail
EOF
```

---

## XSLT Transformation Example

Convert pheromones to markdown summary.

**File: pheromones-to-md.xsl**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:aether="http://aether.dev/core/pheromones/v1">

    <xsl:output method="text" indent="no"/>

    <xsl:template match="/">
        # Pheromone Trails

        Generated: <xsl:value-of select="aether:pheromone-trails/@generated"/>
        Colony: <xsl:value-of select="aether:pheromone-trails/@colony-id"/>

        <xsl:apply-templates select="aether:pheromone-trails/aether:trail"/>
    </xsl:template>

    <xsl:template match="aether:trail">
        ## <xsl:value-of select="@id"/> (<xsl:value-of select="@type"/>)

        - **Substance:** <xsl:value-of select="aether:substance"/>
        - **Strength:** <xsl:value-of select="aether:strength"/>
        - **Decay:** <xsl:value-of select="@decay"/>

    </xsl:template>
</xsl:stylesheet>
```

**Usage:**
```bash
xsltproc pheromones-to-md.xsl pheromones.xml > pheromones.md
```

---

## Error Handling

All commands follow consistent error patterns:

| Error Code | Meaning | Resolution |
|------------|---------|------------|
| `E_NO_PHEROMONES` | No pheromones.json found | Run `/ant:focus` to create first pheromone |
| `E_INVALID_SOURCE` | Source file malformed | Check XML syntax |
| `E_SCHEMA_NOT_FOUND` | XSD schema missing | Run `aether update` |
| `E_VALIDATION_FAILED` | XML doesn't match schema | Check error message for details |
| `E_NAMESPACE_COLLISION` | Prefix already exists | Use `--namespace` to specify unique prefix |

---

## Integration with Aether Commands

### /ant:focus
```bash
# Current (JSON)
echo '{"type": "FOCUS", "content": "..."}' >> .aether/data/pheromones.json

# Future (XML export)
echo '...' | pheromone-add --type FOCUS
pheromone-export  # Sync to eternal
```

### Worker Priming
```bash
# Prime builder with merged pheromones
aether prime builder \
    --include ~/.aether/eternal/pheromones.xml \
    --include template-nodejs.xml \
    --filter "context/domain='web-api'"
```

---

*This document is part of the Aether XML documentation suite.*
