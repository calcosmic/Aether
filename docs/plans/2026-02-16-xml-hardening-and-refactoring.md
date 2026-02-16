# XML Hardening and Refactoring Plan

**Date:** 2026-02-16
**Status:** Ready for Implementation
**Estimated Duration:** 6-8 hours
**Priority:** P0 (Security) + P1 (Architecture)

---

## Executive Summary

Address all critical issues identified in the multi-agent architecture review:
- **Security vulnerabilities** (XXE, path traversal)
- **Code organization** (god objects, mixed concerns)
- **Missing functionality** (round-trip conversion, shared types)

---

## Phase 1: Security Hardening (P0)

**Duration:** 2 hours
**Goal:** Fix critical security vulnerabilities before any other changes

### 1.1 XXE Protection

**Files to modify:**
- `.aether/utils/xml-utils.sh` (lines 200-250, 280-320)

**Changes:**
```bash
# BEFORE
xmllint --schema "$xsd_file" --noout "$xml_file" 2>&1

# AFTER
xmllint --nonet --noent --max-entities 10000 --schema "$xsd_file" --noout "$xml_file" 2>&1
```

Add to all `xmllint` invocations:
- `--nonet`: Disable network access
- `--noent`: Disable external entity resolution
- `--max-entities 10000`: Prevent billion laughs attack

### 1.2 Path Traversal Protection

**Files to modify:**
- `.aether/utils/xinclude-composition.sh` (lines 85-95)

**Changes:**
```bash
# Add validation function
xml-validate-xinclude-path() {
    local include_path="$1"
    local base_dir="$2"
    local allowed_dir

    # Resolve to absolute path
    allowed_dir=$(cd "$base_dir" && pwd)
    include_path=$(cd "$base_dir" && realpath "$include_path" 2>/dev/null)

    # Verify path is within allowed directory
    if [[ ! "$include_path" =~ ^"$allowed_dir" ]]; then
        xml_json_err "PATH_TRAVERSAL_BLOCKED" \
            "XInclude path outside allowed directory" \
            "path=$include_path"
        return 1
    fi

    echo "$include_path"
}
```

### 1.3 Add Security Tests

**File to create:**
- `tests/bash/test-xml-security.sh`

**Test cases:**
- XXE attack with `file:///etc/passwd`
- Billion laughs attack (nested entities)
- Path traversal (`../../../etc/passwd`)
- Deeply nested XML (>100 levels)

---

## Phase 2: Create Shared Types Schema

**Duration:** 1 hour
**Goal:** Eliminate caste enumeration duplication

### 2.1 Create aether-types.xsd

**File to create:**
- `.aether/schemas/aether-types.xsd`

**Content:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.colony/schemas/types/1.0"
           xmlns:types="http://aether.colony/schemas/types/1.0"
           elementFormDefault="qualified">

  <xs:simpleType name="CasteEnum">
    <xs:restriction base="xs:string">
      <xs:enumeration value="builder"/>
      <xs:enumeration value="watcher"/>
      <xs:enumeration value="scout"/>
      <xs:enumeration value="chaos"/>
      <xs:enumeration value="oracle"/>
      <xs:enumeration value="architect"/>
      <xs:enumeration value="prime"/>
      <xs:enumeration value="colonizer"/>
      <xs:enumeration value="route_setter"/>
      <xs:enumeration value="archaeologist"/>
      <xs:enumeration value="ambassador"/>
      <xs:enumeration value="auditor"/>
      <xs:enumeration value="chronicler"/>
      <xs:enumeration value="gatekeeper"/>
      <xs:enumeration value="guardian"/>
      <xs:enumeration value="includer"/>
      <xs:enumeration value="keeper"/>
      <xs:enumeration value="measurer"/>
      <xs:enumeration value="probe"/>
      <xs:enumeration value="sage"/>
      <xs:enumeration value="tracker"/>
      <xs:enumeration value="weaver"/>
    </xs:restriction>
  </xs:simpleType>

  <xs:simpleType name="VersionType">
    <xs:restriction base="xs:string">
      <xs:pattern value="[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?"/>
    </xs:restriction>
  </xs:simpleType>

  <xs:simpleType name="TimestampType">
    <xs:restriction base="xs:string">
      <xs:pattern value="\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?"/>
    </xs:restriction>
  </xs:simpleType>

  <xs:simpleType name="PriorityType">
    <xs:restriction base="xs:string">
      <xs:enumeration value="critical"/>
      <xs:enumeration value="high"/>
      <xs:enumeration value="normal"/>
      <xs:enumeration value="low"/>
    </xs:restriction>
  </xs:simpleType>

  <xs:simpleType name="PheromoneTypeEnum">
    <xs:restriction base="xs:string">
      <xs:enumeration value="FOCUS"/>
      <xs:enumeration value="REDIRECT"/>
      <xs:enumeration value="FEEDBACK"/>
      <xs:enumeration value="PHILOSOPHY"/>
      <xs:enumeration value="STACK"/>
      <xs:enumeration value="PATTERN"/>
      <xs:enumeration value="DECREE"/>
    </xs:restriction>
  </xs:simpleType>

</xs:schema>
```

### 2.2 Update Existing Schemas

**Files to modify:**
- `.aether/schemas/pheromone.xsd`
- `.aether/schemas/worker-priming.xsd`
- `.aether/schemas/prompt.xsd`

**Pattern for each:**
```xml
<!-- Add to each schema -->
<xs:import namespace="http://aether.colony/schemas/types/1.0"
           schemaLocation="aether-types.xsd"/>

<!-- Replace duplicated type with reference -->
<xs:element name="caste" type="types:CasteEnum"/>
```

### 2.3 Add Schema Validation Tests

**File to create:**
- `tests/bash/test-xml-schemas.sh`

**Test cases:**
- Import resolution works
- All castes validate correctly
- Invalid caste fails validation
- Shared types work across schemas

---

## Phase 3: Refactor xml-utils.sh

**Duration:** 3 hours
**Goal:** Split god object into focused modules

### 3.1 Create xml-core.sh

**File to create:**
- `.aether/utils/xml-core.sh` (~400 lines)

**Functions to extract:**
```bash
# Feature detection
xml-detect-tools()

# JSON helpers
xml_json_ok()
xml_json_err()

# Core operations
xml-validate()
xml-well-formed()
xml-format()
xml-escape()
xml-unescape()
```

### 3.2 Create xml-query.sh

**File to create:**
- `.aether/utils/xml-query.sh` (~200 lines)

**Functions:**
```bash
xml-query()              # XPath with xmlstarlet fallback to xmllint
xml-query-attr()         # Attribute extraction
xml-query-text()         # Text content extraction
xml-query-count()        # Count nodes
```

**Add xmllint fallback:**
```bash
xml-query() {
    local xml_file="$1"
    local xpath="$2"

    if [[ "$XMLSTARLET_AVAILABLE" == "true" ]]; then
        xmlstarlet sel -t -v "$xpath" "$xml_file"
    elif [[ "$XMLLINT_AVAILABLE" == "true" ]]; then
        # xmllint has limited XPath but works for basic queries
        xmllint --xpath "$xpath" "$xml_file" 2>/dev/null
    else
        xml_json_err "NO_XML_TOOL" "No XPath-capable tool available"
        return 1
    fi
}
```

### 3.3 Create xml-convert.sh

**File to create:**
- `.aether/utils/xml-convert.sh` (~300 lines)

**Functions:**
```bash
json-to-xml()
xml-to-json()
xml-convert-detect-format()
```

### 3.4 Rename and Update xinclude-composition.sh

**File to rename:**
- `.aether/utils/xinclude-composition.sh` → `.aether/utils/xml-compose.sh`

**Update:**
- Remove manual regex-based fallback
- Add explicit error when xmllint unavailable
- Add path validation
- Update function names:
  - `xml-compose` (was `xml-xinclude-compose`)
  - `xml-list-includes` (was `xml-list-xincludes`)
  - `xml-compose-manual` → remove (security risk)

### 3.5 Create exchange/ Directory

**Files to create:**
- `.aether/exchange/pheromone-xml.sh` (~400 lines)
- `.aether/exchange/wisdom-xml.sh` (~300 lines)
- `.aether/exchange/registry-xml.sh` (~250 lines)

**pheromone-xml.sh functions:**
```bash
# Export (existing functionality)
xml-pheromone-export()
xml-pheromone-to-xml()
xml-pheromone-validate()

# Import (NEW - round-trip)
xml-pheromone-import()
xml-pheromone-from-xml()
xml-pheromone-merge()

# Namespace utilities
xml-pheromone-prefix-id()
xml-pheromone-deprefix-id()
```

**wisdom-xml.sh functions:**
```bash
xml-wisdom-export()
xml-wisdom-import()
xml-wisdom-validate()
xml-wisdom-promote()    # Promote pattern to philosophy
```

**registry-xml.sh functions:**
```bash
xml-registry-export()
xml-registry-import()
xml-registry-validate()
xml-registry-lineage()  # Query ancestry
```

### 3.6 Create Backward-Compatible xml-utils.sh

**File to modify:**
- `.aether/utils/xml-utils.sh` (becomes a loader)

**New content:**
```bash
#!/bin/bash
# XML Utilities Loader
# Sources all XML modules for backward compatibility
#
# Note: New code should source individual modules directly

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Core utilities
source "$SCRIPT_DIR/xml-core.sh"
source "$SCRIPT_DIR/xml-query.sh"
source "$SCRIPT_DIR/xml-convert.sh"
source "$SCRIPT_DIR/xml-compose.sh"

# Domain-specific exchange modules
source "$SCRIPT_DIR/../exchange/pheromone-xml.sh"
source "$SCRIPT_DIR/../exchange/wisdom-xml.sh"
source "$SCRIPT_DIR/../exchange/registry-xml.sh"

# Export all functions for backward compatibility
export -f xml-validate xml-well-formed xml-format
export -f xml-query xml-query-attr
export -f json-to-xml xml-to-json
export -f xml-compose xml-list-includes
export -f xml-pheromone-export xml-pheromone-import
export -f xml-wisdom-export xml-wisdom-import
export -f xml-registry-export xml-registry-import
```

---

## Phase 4: Add Round-Trip Conversion

**Duration:** 1 hour
**Goal:** Complete the JSON ↔ XML bidirectional flow

### 4.1 Implement xml-pheromone-from-xml

**File:** `.aether/exchange/pheromone-xml.sh`

**Function:**
```bash
xml-pheromone-from-xml() {
    local xml_file="${1:-}"
    local output_json="${2:-}"

    # Validate inputs
    [[ -z "$xml_file" ]] && { xml_json_err "MISSING_XML_FILE"; return 1; }
    [[ -f "$xml_file" ]] || { xml_json_err "XML_FILE_NOT_FOUND"; return 1; }

    # Default output
    if [[ -z "$output_json" ]]; then
        output_json=".aether/data/pheromones.json"
    fi

    # Extract metadata
    local version colony_id generated_at
    version=$(xmllint --xpath "string(/*/@version)" "$xml_file" 2>/dev/null)
    colony_id=$(xmllint --xpath "string(/*/@colony_id)" "$xml_file" 2>/dev/null)
    generated_at=$(xmllint --xpath "string(/*/@generated_at)" "$xml_file" 2>/dev/null)

    # Convert signals to JSON
    local signals_json
    signals_json=$(xmlstarlet sel \
        -t -m "/pheromones/signal" \
        -o '{"id":"' -v "@id" -o '",' \
        -o '"type":"' -v "@type" -o '",' \
        -o '"priority":"' -v "@priority" -o '",' \
        -o '"content":"' -v "content/text()" -o '"}' \
        -n "$xml_file" 2>/dev/null | jq -s '.')

    # Build output structure
    jq -n \
        --arg version "$version" \
        --arg colony_id "$colony_id" \
        --arg generated_at "$generated_at" \
        --argjson signals "$signals_json" \
        '{
            version: $version,
            colony_id: $colony_id,
            generated_at: $generated_at,
            signals: $signals
        }' > "$output_json"

    xml_json_ok "{\"imported\": true, \"file\": \"$output_json\", \"signals\": $(echo "$signals_json" | jq 'length')}"
}
```

### 4.2 Implement xml-pheromone-merge

**Function:**
```bash
xml-pheromone-merge() {
    local source_xml="$1"
    local target_xml="${2:-~/.aether/eternal/pheromones.xml}"
    local namespace_prefix="${3:-}"

    # Extract source colony ID
    local source_id
    source_id=$(xmllint --xpath "string(/*/@colony_id)" "$source_xml" 2>/dev/null)

    # Generate namespace if not provided
    [[ -z "$namespace_prefix" ]] && namespace_prefix="col-${source_id}"

    # Transform with namespace prefix
    xmlstarlet ed \
        -N "ph=http://aether.colony/schemas/pheromones" \
        -r "//ph:signal" -v "${namespace_prefix}:signal" \
        "$source_xml" > "${target_xml}.tmp"

    # Merge into target (append signals, deduplicate by ID)
    # Implementation details...

    rm "${target_xml}.tmp"

    xml_json_ok "{\"merged\": true, \"source\": \"$source_id\", \"namespace\": \"$namespace_prefix\"}"
}
```

### 4.3 Add Round-Trip Tests

**Add to:** `tests/bash/test-xml-utils.sh`

**Test cases:**
- JSON → XML → JSON produces equivalent result
- Namespace prefixing and deprefixing are inverses
- Merged pheromones preserve source attribution

---

## Phase 5: Update Tests and Documentation

**Duration:** 1 hour
**Goal:** All tests pass, documentation reflects new structure

### 5.1 Update Test Suite

**Files to modify:**
- `tests/bash/test-xml-utils.sh`
  - Update function names (backward compat should work)
  - Add tests for new `xml-pheromone-import`
  - Add tests for shared types schema

- `tests/bash/test-xinclude-composition.sh`
  - Rename to `test-xml-compose.sh`
  - Update for removed manual fallback
  - Add path traversal tests

**Files to create:**
- `tests/bash/test-xml-security.sh` (see Phase 1.3)
- `tests/bash/test-xml-schemas.sh` (see Phase 2.3)
- `tests/bash/test-exchange-pheromone.sh` (round-trip tests)

### 5.2 Update Documentation

**Files to modify:**
- `docs/xml-migration/SHELL-INTEGRATION.md`
  - Update function names
  - Document new module structure
  - Add security considerations section

**Files to create:**
- `docs/xml-migration/SECURITY.md`
  - XXE protection details
  - Path traversal prevention
  - Best practices for untrusted XML

### 5.3 Update aether-utils.sh Integration

**File to modify:**
- `.aether/aether-utils.sh`

**Changes:**
```bash
# Add to initialization section
source "$SCRIPT_DIR/utils/xml-utils.sh"  # Now a loader

# Optional: Add XML command registration
register-xml-commands() {
    # Register xml-* commands with CLI
}
```

---

## Phase 6: Verification

**Duration:** 30 minutes
**Goal:** Confirm everything works

### 6.1 Run All Tests

```bash
# Run all XML-related tests
bash tests/bash/test-xml-utils.sh
bash tests/bash/test-xml-compose.sh
bash tests/bash/test-xml-security.sh
bash tests/bash/test-xml-schemas.sh
bash tests/bash/test-exchange-pheromone.sh
bash tests/bash/test-pheromone-xml.sh
bash tests/bash/test-phase3-xml.sh

# Run existing test suite
npm test
```

### 6.2 Manual Verification

```bash
# Test XXE protection
xmllint --nonet --noent --schema .aether/schemas/pheromone.xsd \
    .aether/schemas/examples/pheromone-example.xml

# Test path validation
source .aether/utils/xml-compose.sh
xml-compose .aether/examples/worker-priming.xml /tmp/composed.xml

# Test round-trip
source .aether/exchange/pheromone-xml.sh
xml-pheromone-to-xml .aether/data/pheromones.json /tmp/pheromones.xml
xml-pheromone-from-xml /tmp/pheromones.xml /tmp/pheromones-restored.json

# Verify equivalence
diff .aether/data/pheromones.json /tmp/pheromones-restored.json
```

### 6.3 Sync to Runtime

```bash
# After all changes committed
npm install -g .

# Verify distribution
ls ~/.aether/system/utils/xml-*.sh
ls ~/.aether/system/exchange/*.sh
ls ~/.aether/system/schemas/aether-types.xsd
```

---

## Rollback Plan

If issues arise:

1. **Git rollback:** `git reset --hard HEAD~N` (where N = commits made)
2. **Preserve new tests:** Stash `tests/bash/test-xml-security.sh`
3. **Manual restore:** Original `xml-utils.sh` is preserved in git history

---

## Success Criteria

- [ ] All P0 security issues fixed (XXE, path traversal)
- [ ] `aether-types.xsd` created and imported by all schemas
- [ ] `xml-utils.sh` refactored into 4 modules + loader
- [ ] `exchange/` directory created with 3 domain modules
- [ ] `xml-pheromone-import()` implemented and tested
- [ ] All 56+ tests passing (existing + new)
- [ ] Backward compatibility maintained (old function names work)
- [ ] Documentation updated

---

## Dependencies

**Required tools:**
- `xmllint` (libxml2) - usually pre-installed
- `xmlstarlet` - `brew install xmlstarlet`
- `xsltproc` - usually pre-installed
- `realpath` - coreutils (for path validation)

**No new dependencies required.**

---

## Notes

- Keep changes atomic - one phase per commit
- Test after each phase
- Maintain backward compatibility throughout
- Security fixes (Phase 1) can be deployed independently
