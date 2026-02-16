# XML Infrastructure Analysis

## Executive Summary

The Aether colony system includes a sophisticated XML infrastructure designed for "eternal memory" - structured, validated, versioned storage of colony wisdom, pheromones, prompts, and registry data. This analysis documents the complete XML system including 5 XSD schemas, 30+ utility functions, XInclude composition, security measures, and current usage status.

**Key Finding**: The XML infrastructure is comprehensive and production-ready but currently **dormant** - only one command (`colonize`) has minimal XML integration, and the pheromone export function exists but is not actively used by any workflow.

---

## XSD Schema Catalog

### 1. prompt.xsd (417 lines)
- **Purpose**: Define structured prompts for colony workers and commands
- **Namespace**: `http://aether.colony/schemas/prompt/1.0`
- **Key Types**:
  - `casteType`: Enumeration of 22 castes (builder, watcher, scout, chaos, oracle, architect, prime, colonizer, route_setter, archaeologist, chronicler, guardian, gatekeeper, weaver, probe, sage, measurer, keeper, tracker, includer)
  - `requirementType`: Individual requirement with ID, priority (critical/high/normal/low)
  - `constraintType`: Hard/soft constraints with strength (must/should/may/must-not/should-not)
  - `thinkingType`: Step-by-step approach guidance with checkpoints
  - `successCriteriaType`: Measurable completion criteria
- **Root Element**: `<aether-prompt>` with metadata, objective, requirements, constraints, thinking, tools, output, verification
- **Usage Status**: **DORMANT** - Infrastructure exists but no commands generate/use XML prompts

### 2. pheromone.xsd (251 lines)
- **Purpose**: Define XML structure for pheromone signals used in colony communication
- **Namespace**: `http://aether.colony/schemas/pheromones`
- **Key Types**:
  - `SignalType`: Individual pheromone with id, type (FOCUS/REDIRECT/FEEDBACK), priority, source, timestamps
  - `ContentType`: Mixed content with optional text and structured data
  - `ScopeType`: Target castes, paths (glob patterns), phases with match mode (any/all/none)
  - `TagType`: Weighted categorization tags (0.0-1.0 weight)
  - `CasteEnum`: All 22 castes (matches prompt.xsd)
- **Signal Types**:
  - `FOCUS`: Direct attention (normal priority)
  - `REDIRECT`: Hard constraint (high priority)
  - `FEEDBACK`: Gentle adjustment (low priority)
- **Usage Status**: **PARTIALLY ACTIVE** - `pheromone-export` function exists in aether-utils.sh (line 3366) but not actively called by commands

### 3. colony-registry.xsd (310 lines)
- **Purpose**: Multi-colony registry with lineage tracking and pheromone inheritance
- **Namespace**: Default (qualified elements)
- **Key Types**:
  - `colonyType`: Complete colony definition with identity, location, status, lineage
  - `lineageType`: Ancestry chain with parent relationships and generation tracking
  - `pheromoneType`: Inherited pheromones with strength (0.0-1.0) and source tracking
  - `relationshipType`: Cross-colony relationships (parent/child/sibling/fork/merge/reference)
  - `registryInfoType`: Registry metadata with version and colony count
- **Key Constraints**:
  - `colonyIdKey`: Unique colony IDs
  - `parentColonyRef`, `forkedFromRef`, `ancestorRef`: Referential integrity for lineage
  - `relationshipTargetRef`: Valid relationship targets
- **Usage Status**: **DORMANT** - No active registry management

### 4. worker-priming.xsd (277 lines)
- **Purpose**: Modular configuration composition using XInclude for worker initialization
- **Namespace**: `http://aether.colony/schemas/worker-priming/1.0`
- **Key Types**:
  - `workerIdentityType`: Worker ID, name, caste, generation, parent colony
  - `configSourceType`: XInclude or inline configuration sources with priority
  - `queenWisdomSectionType`: Eternal wisdom inclusion
  - `activeTrailsSectionType`: Current pheromone signals
  - `stackProfilesSectionType`: Technology-specific configuration
  - `overrideRulesType`: Configuration merging rules (replace/merge/append/prepend/remove)
- **Pruning Modes**: full, minimal, inherit, override
- **Usage Status**: **DORMANT** - No workers are primed via XML

### 5. queen-wisdom.xsd (326 lines)
- **Purpose**: Eternal memory structure for learned patterns, principles, and evolution tracking
- **Namespace**: `http://aether.colony/schemas/queen-wisdom/1.0`
- **Key Types**:
  - `wisdomEntryType`: Base type with id, confidence (0.0-1.0), domain, source, timestamps
  - `philosophyType`: Core beliefs with principles list
  - `patternType`: Validated approaches with pattern_type (success/failure/anti-pattern/emerging)
  - `redirectType`: Hard constraints with constraint_type (must/must-not/avoid/prefer)
  - `stackWisdomType`: Technology-specific insights with version_range and workaround
  - `decreeType`: Authoritative directives with authority, expiration, scope
  - `evolutionType`: Version tracking with supersession and deprecation
- **Domains**: architecture, testing, security, performance, ux, process, communication, debugging, general
- **Usage Status**: **DORMANT** - No wisdom promotion workflow active

---

## XML Utility Functions

### Core Functions (xml-utils.sh)

#### xml-detect-tools
- **Purpose**: Detect available XML processing tools (xmllint, xmlstarlet, xsltproc, xml2json)
- **Returns**: JSON with availability flags for each tool
- **Dependencies**: None (detection only)

#### xml-well-formed <xml_file>
- **Purpose**: Check if XML document is well-formed
- **Security**: Uses xmllint --noout (no entity expansion)
- **Returns**: `{"ok":true,"result":{"well_formed":true}}` or `{"well_formed":false,"error":"..."}`

#### xml-validate <xml_file> [xsd_file]
- **Purpose**: Validate XML against XSD schema
- **Security**: XXE protection via --noent flag
- **Returns**: `{"ok":true,"result":{"valid":true}}` or validation errors
- **Dependencies**: xmllint

#### xml-format <xml_file>
- **Purpose**: Pretty-print XML document
- **Security**: In-place formatting with --format
- **Returns**: Success confirmation with formatted indicator

#### xml-query <xml_file> <xpath_expression>
- **Purpose**: Execute XPath query against XML document
- **Security**: Read-only query execution
- **Returns**: Matching nodes with count
- **Dependencies**: xmlstarlet (preferred) or xmllint fallback

#### xml-merge <output_file> <input_files...>
- **Purpose**: Merge multiple XML documents using XInclude
- **Security**: Uses xml-compose with path validation
- **Returns**: Composed document path

### Conversion Functions

#### json-to-xml <json_file> [root_element]
- **Purpose**: Convert JSON to XML representation
- **Algorithm**: Recursive jq-based transformation
- **Handles**: Objects, arrays, primitives, nested structures
- **Returns**: XML string with specified root element (default: "root")

#### pheromone-to-xml <json_file> [output_xml] [schema_file]
- **Purpose**: Convert pheromone JSON to schema-valid XML
- **Features**:
  - Case normalization (focus -> FOCUS)
  - Invalid value fallback (invalid type -> FOCUS, invalid priority -> normal)
  - XML escaping for special characters
  - Caste validation against 22 valid castes
  - Schema validation if xmllint available
- **Returns**: XML output or validation result

#### queen-wisdom-to-xml <json_file> [output_xml]
- **Purpose**: Convert queen wisdom JSON to XML
- **Handles**: Philosophies, patterns, redirects, stack-wisdom, decrees
- **Returns**: Structured queen-wisdom XML

#### registry-to-xml <json_file> [output_xml]
- **Purpose**: Convert colony registry JSON to XML
- **Handles**: Colony entries, lineage, relationships, inherited pheromones
- **Returns**: colony-registry XML document

### Prompt Functions

#### prompt-to-xml <markdown_file> [output_xml]
- **Purpose**: Convert markdown prompt to structured XML
- **Extracts**: Objectives, requirements, constraints, thinking steps
- **Returns**: aether-prompt XML document

#### prompt-from-xml <xml_file>
- **Purpose**: Convert XML prompt back to markdown
- **Returns**: Markdown representation

#### prompt-validate <xml_file>
- **Purpose**: Validate prompt XML against prompt.xsd
- **Returns**: Validation result

### Queen Wisdom Functions

#### queen-wisdom-to-markdown <xml_file> [output_md]
- **Purpose**: Transform queen-wisdom XML to human-readable markdown
- **Implementation**: Uses XSLT stylesheet (queen-to-md.xsl)
- **Output Sections**: Philosophies, Patterns, Redirects, Stack Wisdom, Decrees, Evolution Log
- **Dependencies**: xsltproc

#### queen-wisdom-validate-entry <xml_file> <entry_id>
- **Purpose**: Validate single wisdom entry against schema
- **Returns**: Validation result with specific error location

#### queen-wisdom-promote <type> <entry_id> <target_colony>
- **Purpose**: Promote observation to pattern, pattern to philosophy
- **Workflow**: Validates, updates evolution log, writes to eternal memory
- **Returns**: Promotion confirmation

#### queen-wisdom-import <xml_file> [colony_id]
- **Purpose**: Import external wisdom into colony's eternal memory
- **Handles**: Namespace prefixing for collision avoidance
- **Returns**: Import statistics

### Namespace Functions

#### generate-colony-namespace <session_id>
- **Purpose**: Generate unique namespace URI for colony
- **Format**: `http://aether.dev/colony/{session_id}`
- **Returns**: Namespace URI and prefix

#### generate-cross-colony-prefix <external_session> <local_session>
- **Purpose**: Generate collision-free prefix for cross-colony references
- **Format**: `{hash}_{ext|col}_{hash}`
- **Returns**: Prefix for external colony elements

#### prefix-pheromone-id <signal_id> <colony_prefix>
- **Purpose**: Prefix signal ID with colony identifier
- **Features**: Idempotent (won't double-prefix)
- **Returns**: Prefixed ID

#### validate-colony-namespace <namespace_uri>
- **Purpose**: Validate namespace URI format
- **Recognizes**: Colony namespaces, schema namespaces
- **Returns**: Validity flag and type

### Export Functions

#### pheromone-export <pheromones_json> [output_xml] [colony_id] [schema_file]
- **Purpose**: Export pheromones to eternal memory XML
- **Location**: Default `~/.aether/eternal/pheromones.xml`
- **Called by**: `pheromone-to-xml` with validation
- **Returns**: Export statistics

---

## XInclude Composition System

### xml-compose.sh Module

#### xml-compose <input_xml> [output_xml]
- **Purpose**: Resolve XInclude directives in XML documents
- **Security Features**:
  - Uses xmllint with --nonet (no network access)
  - Uses --noent (no entity expansion, XXE protection)
  - Uses --xinclude (process XInclude)
- **Returns**: Composed XML with all includes resolved
- **Dependencies**: xmllint (required, no fallback for security)

#### xml-list-includes <xml_file>
- **Purpose**: List all XInclude references in document
- **Implementation**: xmlstarlet (preferred) or grep fallback
- **Returns**: Array of include objects with href, parse, xpointer, resolved path

#### xml-compose-worker-priming <priming_xml> [output_xml]
- **Purpose**: Specialized composition for worker priming documents
- **Validates**: Against worker-priming.xsd
- **Extracts**: Worker identity, counts sources by section
- **Returns**: Composition result with worker metadata

#### xml-validate-include-path <include_path> <base_dir>
- **Purpose**: Security validation for XInclude paths
- **Protection**:
  - Rejects paths with `..` sequences (traversal detection)
  - Validates absolute paths start with allowed directory
  - Normalizes and re-verifies resolved path
- **Returns**: Normalized absolute path or error
- **Error Codes**: `PATH_TRAVERSAL_DETECTED`, `PATH_TRAVERSAL_BLOCKED`, `INVALID_BASE_DIR`

### Composition Example
```xml
<!-- worker-priming.xml -->
<worker-priming xmlns:xi="http://www.w3.org/2001/XInclude">
  <queen-wisdom>
    <wisdom-source name="eternal-wisdom">
      <xi:include href="../eternal/queen-wisdom.xml"
                  parse="xml"
                  xpointer="xmlns(qw=...)xpointer(/qw:queen-wisdom/qw:philosophies)"/>
    </wisdom-source>
  </queen-wisdom>
</worker-priming>
```

---

## Security Measures

### XXE Protection
1. **--nonet flag**: Prevents network access during XML processing
2. **--noent flag**: Disables entity expansion, preventing file disclosure
3. **No external DTD loading**: xmllint configured to reject external entities

### Path Traversal Protection
1. **Pattern detection**: Rejects paths containing `..` sequences
2. **Absolute path validation**: Ensures absolute paths start with allowed directory
3. **Path normalization**: Resolves and re-verifies final path location
4. **Base directory enforcement**: All includes relative to defined base

### Entity Expansion Limits
- Billion laughs attack mitigated by --noent flag
- No entity expansion means exponential expansion attacks are impossible

### Test Coverage
- `test-xml-security.sh`: 7 security tests covering XXE, path traversal, network access
- `test-pheromone-xml.sh`: 15 tests for pheromone conversion with validation
- `test-xml-utils.sh`: 20 tests for all utility functions
- `test-phase3-xml.sh`: 15 tests for queen-wisdom and prompt workflows

---

## JSON/XML Bidirectional Conversion

### JSON to XML
- **Mechanism**: jq-based recursive transformation
- **Object handling**: Creates child elements with keys as tag names
- **Array handling**: Creates repeated elements
- **Primitive handling**: Text content with proper escaping
- **Root element**: Configurable (default: "root")

### XML to JSON
- **Mechanism**: xmlstarlet or xsltproc transformation
- **Preserves**: Structure, attributes (as @attr), text content
- **Namespace handling**: Preserves namespace prefixes

### Hybrid Architecture
The system uses a hybrid approach:
- **JSON**: Runtime efficiency, active pheromones, colony state
- **XML**: Eternal memory, validation, versioning, cross-colony exchange

---

## Current Usage Analysis

### Active Usage (Minimal)

| Component | Usage | Location |
|-----------|-------|----------|
| xml-utils.sh | Sourced | `.aether/aether-utils.sh:30` |
| pheromone-export | Function exists, not called | `.aether/aether-utils.sh:3366-3381` |

### Dormant Infrastructure

| Schema | Status | Ready For |
|--------|--------|-----------|
| prompt.xsd | Dormant | XML-based worker prompts |
| pheromone.xsd | Dormant | Structured pheromone exchange |
| colony-registry.xsd | Dormant | Multi-colony management |
| worker-priming.xsd | Dormant | Declarative worker initialization |
| queen-wisdom.xsd | Dormant | Eternal wisdom storage |

### Commands with XML Potential

| Command | Current | XML Opportunity |
|---------|---------|-----------------|
| `/ant:colonize` | Minimal XML reference | Could generate survey XML |
| `/ant:focus` | JSON pheromones | Could export to XML |
| `/ant:redirect` | JSON pheromones | Could export to XML |
| `/ant:feedback` | JSON pheromones | Could export to XML |
| `/ant:oracle` | Research JSON | Could store findings as wisdom XML |
| `/ant:init` | JSON state | Could validate against schemas |
| `/ant:seal` | JSON archive | Could use registry format |

---

## Issues Found

### 1. Dormant Infrastructure (Not a Bug, But a Gap)
- **Issue**: Comprehensive XML system exists but is not used
- **Impact**: Development effort invested but not yielding value
- **Location**: All 5 schemas, xml-utils.sh, xml-compose.sh

### 2. Schema Location Mismatch
- **Issue**: worker-priming.xsd imports XInclude schema from W3C URL
- **Impact**: Requires network access for validation
- **Location**: `.aether/schemas/worker-priming.xsd:22-23`
- **Recommendation**: Bundle local copy of XInclude.xsd

### 3. XSLT Stylesheet Namespace Mismatch
- **Issue**: queen-to-md.xsl uses default namespace but schema defines qw: namespace
- **Impact**: XSLT may not match elements correctly
- **Location**: `.aether/utils/queen-to-md.xsl:22` vs `queen-wisdom.xsd:16`
- **Fix**: Add `xmlns:qw="http://aether.colony/schemas/queen-wisdom/1.0"` to stylesheet and update match patterns

### 4. Missing Evolution Log in queen-wisdom.xsd
- **Issue**: test-phase3-xml.sh references `<evolution-log>` element but schema doesn't define it
- **Impact**: Test creates invalid XML
- **Location**: `test-phase3-xml.sh:188-192` vs `queen-wisdom.xsd`
- **Fix**: Add evolution-log element to schema or remove from test

### 5. No Active pheromone-export Calls
- **Issue**: Function exists but never invoked
- **Impact**: Pheromone XML infrastructure unused
- **Location**: `.aether/aether-utils.sh:3366-3381`
- **Recommendation**: Integrate into pheromone signal workflow

---

## Improvement Opportunities

### Phase 1: Activate Pheromone Export
**Effort**: Low | **Value**: Medium

Add pheromone-to-XML export to the pheromone signal workflow:
```bash
# In pheromone signal handlers
pheromone-export ".aether/data/pheromones.json" ".aether/eternal/pheromones.xml"
```

### Phase 2: XML-Based Worker Prompts
**Effort**: Medium | **Value**: High

Convert worker prompts from markdown to XML:
1. Convert existing prompts with `prompt-to-xml`
2. Store in `.aether/prompts/{caste}.xml`
3. Load and validate before spawning workers
4. Use XInclude for shared constraint libraries

### Phase 3: Queen Wisdom Promotion Workflow
**Effort**: Medium | **Value**: High

Implement the wisdom promotion pipeline:
1. Observations accumulate in session JSON
2. `queen-wisdom-promote` converts valid patterns to XML
3. XSLT generates QUEEN.md for human reading
4. Cross-colony wisdom import for shared learnings

### Phase 4: Colony Registry for Multi-Repo
**Effort**: High | **Value**: Medium

Activate colony registry for multi-repository tracking:
1. Registry XML in `~/.aether/eternal/registry.xml`
2. Lineage tracking for forked colonies
3. Pheromone inheritance between related colonies
4. Relationship management (parent/child/sibling)

### Phase 5: Worker Priming with XInclude
**Effort**: High | **Value**: High

Implement declarative worker initialization:
1. Priming XML per worker type
2. XInclude composition of wisdom + pheromones + stack profiles
3. Override rules for customization
4. Validation before worker spawn

---

## File Inventory

### Schemas (5 files)
- `.aether/schemas/prompt.xsd` (417 lines)
- `.aether/schemas/pheromone.xsd` (251 lines)
- `.aether/schemas/colony-registry.xsd` (310 lines)
- `.aether/schemas/worker-priming.xsd` (277 lines)
- `.aether/schemas/queen-wisdom.xsd` (326 lines)

### Utilities (3 files)
- `.aether/utils/xml-utils.sh` (~600 lines)
- `.aether/utils/xml-compose.sh` (248 lines)
- `.aether/utils/queen-to-md.xsl` (396 lines)

### Examples (5 files)
- `.aether/schemas/example-prompt-builder.xml` (235 lines)
- `.aether/schemas/examples/pheromone-example.xml` (118 lines)
- `.aether/schemas/examples/colony-registry-example.xml` (303 lines)
- `.aether/schemas/examples/queen-wisdom-example.xml` (382 lines)
- `.aether/examples/worker-priming.xml` (172 lines)

### Tests (4 files)
- `tests/bash/test-xml-utils.sh` (1046 lines, 20 tests)
- `tests/bash/test-pheromone-xml.sh` (417 lines, 15 tests)
- `tests/bash/test-phase3-xml.sh` (381 lines, 15 tests)
- `tests/bash/test-xml-security.sh` (288 lines, 7 tests)

---

## Conclusion

The Aether XML infrastructure represents a sophisticated, well-designed system for structured colony memory. The schemas are comprehensive, the utility functions are robust with proper security measures, and the test coverage is thorough. However, the system is currently dormant - a significant investment waiting to be activated.

**Recommendation**: Begin with Phase 1 (pheromone export) to establish the XML workflow, then proceed to Phase 2 (XML prompts) for immediate value in worker initialization. The infrastructure is ready; it needs integration into active command workflows.

---

*Analysis generated: 2026-02-16*
*Analyst: Oracle caste*
*Status: Complete*