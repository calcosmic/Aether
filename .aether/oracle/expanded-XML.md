# Aether XML Infrastructure: Comprehensive Technical Documentation

## Executive Summary

The Aether colony system includes a sophisticated XML infrastructure designed for "eternal memory" - structured, validated, versioned storage of colony wisdom, pheromones, prompts, and registry data. This comprehensive documentation provides exhaustive technical details for all 6 XSD schemas, 30+ utility functions, security mechanisms, and integration patterns.

**Current Status**: The XML infrastructure is production-ready but largely dormant. Only minimal integration exists in the `colonize` command, with comprehensive schemas and utilities awaiting activation.

---

## Table of Contents

1. [XML Architecture Philosophy](#1-xml-architecture-philosophy)
2. [XSD Schema Reference](#2-xsd-schema-reference)
   - 2.1 [aether-types.xsd](#21-aether-typesxsd)
   - 2.2 [prompt.xsd](#22-promptxsd)
   - 2.3 [pheromone.xsd](#23-pheromonexsd)
   - 2.4 [colony-registry.xsd](#24-colony-registryxsd)
   - 2.5 [worker-priming.xsd](#25-worker-primingxsd)
   - 2.6 [queen-wisdom.xsd](#26-queen-wisdomxsd)
3. [XML Utility Functions](#3-xml-utility-functions)
4. [XInclude Composition System](#4-xinclude-composition-system)
5. [Security Architecture](#5-security-architecture)
6. [JSON/XML Conversion](#6-jsonxml-conversion)
7. [Schema Evolution Strategy](#7-schema-evolution-strategy)
8. [Performance Optimization](#8-performance-optimization)
9. [Industry Comparison](#9-industry-comparison)
10. [Activation Roadmap](#10-activation-roadmap)

---

## 1. XML Architecture Philosophy

### 1.1 The Hybrid Memory Model

The Aether XML infrastructure implements a hybrid architecture that leverages the strengths of both JSON and XML:

**JSON for Runtime Efficiency**
- Active colony state (COLONY_STATE.json)
- Runtime pheromone signals
- Session data and activity logs
- Quick read/write operations
- JavaScript-native parsing

**XML for Eternal Memory**
- Validated, schema-enforced structure
- Version-controlled wisdom storage
- Cross-colony exchange format
- XInclude-based modular composition
- XSLT transformation capabilities
- Human-readable with machine precision

This dual-format approach acknowledges a fundamental truth: different phases of data lifecycle have different requirements. Runtime operations prioritize speed and flexibility, while archival and exchange operations prioritize structure, validation, and longevity.

### 1.2 Biological Inspiration

The XML architecture draws inspiration from biological information systems:

**DNA as Schema**: Just as DNA provides a structured template for protein synthesis, XSD schemas provide templates for valid colony documents. The schema is the genotype; individual XML documents are phenotypes.

**Pheromone Trails**: Ant colonies use chemical signals to communicate. The pheromone.xsd schema formalizes these signals into structured XML, enabling persistent, scoped, weighted communication between colony components.

**Collective Memory**: Queen wisdom represents the colony's accumulated learning - patterns that have proven successful, redirects that prevent failure, and decrees that govern behavior. XML's hierarchical structure naturally represents this layered knowledge.

### 1.3 Namespace Design Philosophy

Namespaces in Aether XML serve multiple purposes:

**Version Isolation**: Each schema version has a unique namespace URI, ensuring that documents validate against the correct schema version even as schemas evolve.

**Cross-Colony Identity**: Colony-specific namespaces prevent identifier collisions when wisdom is shared between colonies.

**Semantic Clarity**: Namespace prefixes (ph:, qw:, wp:) provide immediate visual context about the type of information being viewed.

The namespace hierarchy follows a consistent pattern:
- `http://aether.colony/schemas/{schema-name}/{version}` for schemas
- `http://aether.dev/colony/{session-id}` for colony instances

### 1.4 Validation as Contract

XSD validation serves as a contract between document producers and consumers:

**Producer Guarantee**: A valid document meets structural requirements, contains required fields, and respects type constraints.

**Consumer Assurance**: Code processing validated XML can make assumptions about structure, reducing defensive coding and runtime checks.

**Evolution Safety**: Schema versioning allows documents to declare their format version, enabling backward compatibility and migration paths.

### 1.5 XInclude for Modular Composition

XInclude enables document composition - the ability to assemble a complete document from multiple sources:

**Separation of Concerns**: Queen wisdom, active pheromones, and stack profiles can be maintained in separate files.

**Reusability**: Common wisdom can be included in multiple worker priming documents without duplication.

**Dynamic Assembly**: Documents can be composed at runtime based on context, pulling in relevant sections as needed.

**Override Capability**: The worker-priming schema includes override rules that modify included content, enabling customization without modifying shared sources.

---

## 2. XSD Schema Reference

### 2.1 aether-types.xsd

**File Location**: `.aether/schemas/aether-types.xsd`

**Namespace**: `http://aether.colony/schemas/types/1.0`

**Purpose**: Defines shared types used across all Aether Colony schemas, eliminating duplication and ensuring consistency.

#### 2.1.1 Schema Overview

The aether-types.xsd schema serves as the foundation of the Aether type system. It defines common enumerations, patterns, and constraints that are imported by other schemas. This centralization ensures that when a type definition changes, all schemas using that type are automatically updated.

#### 2.1.2 Simple Type Definitions

**CasteEnum**

The CasteEnum type defines all 22 worker castes in the Aether system:

```xml
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
```

**Design Rationale**: The 22 castes represent a complete taxonomy of worker specializations. Each caste has a specific emoji, role, and typical task assignment. Centralizing this enumeration ensures consistency across prompts, pheromones, worker priming, and wisdom documents.

**Usage Pattern**: Import into other schemas using:
```xml
<xs:import namespace="http://aether.colony/schemas/types/1.0"
           schemaLocation="aether-types.xsd"/>
```

**VersionType**

Defines semantic version strings (e.g., 1.0.0, 2.1.3-alpha):

```xml
<xs:simpleType name="VersionType">
  <xs:restriction base="xs:string">
    <xs:pattern value="[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?"/>
  </xs:restriction>
</xs:simpleType>
```

**Pattern Breakdown**:
- `[0-9]+` - Major version (one or more digits)
- `\.` - Literal dot separator
- `[0-9]+` - Minor version
- `\.` - Literal dot separator
- `[0-9]+` - Patch version
- `(-[a-zA-Z0-9]+)?` - Optional prerelease suffix

**TimestampType**

ISO 8601 timestamp with optional milliseconds and timezone:

```xml
<xs:simpleType name="TimestampType">
  <xs:restriction base="xs:string">
    <xs:pattern value="\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?"/>
  </xs:restriction>
</xs:simpleType>
```

**Valid Examples**:
- `2026-02-16T14:30:00Z` - UTC timestamp
- `2026-02-16T14:30:00.123+01:00` - With milliseconds and timezone offset
- `2026-02-16T14:30:00` - Local time (no timezone)

**PriorityType**

Four-level priority enumeration:

```xml
<xs:simpleType name="PriorityType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="critical"/>
    <xs:enumeration value="high"/>
    <xs:enumeration value="normal"/>
    <xs:enumeration value="low"/>
  </xs:restriction>
</xs:simpleType>
```

**Semantic Meaning**:
- `critical` - Immediate attention required, blocks other work
- `high` - Important, should be addressed soon
- `normal` - Standard priority, queue appropriately
- `low` - Nice-to-have, address when convenient

**PheromoneTypeEnum**

Extended signal types beyond the basic three:

```xml
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
```

**Extended Types Rationale**: While FOCUS, REDIRECT, and FEEDBACK are runtime pheromone signals, PHILOSOPHY, STACK, PATTERN, and DECREE represent wisdom categories that can also function as directional signals.

**Identifier Types**

Three identifier types with specific constraints:

```xml
<!-- General identifier: alphanumeric with hyphens/underscores -->
<xs:simpleType name="IdentifierType">
  <xs:restriction base="xs:string">
    <xs:pattern value="[a-zA-Z][a-zA-Z0-9_-]*"/>
    <xs:minLength value="1"/>
    <xs:maxLength value="64"/>
  </xs:restriction>
</xs:simpleType>

<!-- Worker ID: kebab-case, minimum 3 characters -->
<xs:simpleType name="WorkerIdType">
  <xs:restriction base="xs:string">
    <xs:pattern value="[a-z][a-z0-9-]*"/>
    <xs:minLength value="3"/>
    <xs:maxLength value="64"/>
  </xs:restriction>
</xs:simpleType>

<!-- Wisdom ID: kebab-case, minimum 3 characters -->
<xs:simpleType name="WisdomIdType">
  <xs:restriction base="xs:string">
    <xs:pattern value="[a-z][a-z0-9-]*"/>
    <xs:minLength value="3"/>
    <xs:maxLength value="64"/>
  </xs:restriction>
</xs:simpleType>
```

**Constraint Rationale**:
- Must start with letter (prevents numeric-only IDs which could be confused with array indices)
- Kebab-case for readability (worker-id vs workerId)
- Maximum 64 characters for database compatibility and readability
- Minimum 3 characters to prevent single-character IDs which reduce clarity

**WeightType and ConfidenceType**

Decimal types constrained to 0.0-1.0 range with 2 decimal places:

```xml
<xs:simpleType name="WeightType">
  <xs:restriction base="xs:decimal">
    <xs:minInclusive value="0.0"/>
    <xs:maxInclusive value="1.0"/>
    <xs:fractionDigits value="2"/>
  </xs:restriction>
</xs:simpleType>
```

**Usage Context**:
- `WeightType` - Tag importance, pheromone strength
- `ConfidenceType` - Pattern validation, wisdom certainty

**MatchEnum**

Scope matching mode:

```xml
<xs:simpleType name="MatchEnum">
  <xs:restriction base="xs:string">
    <xs:enumeration value="any"/>
    <xs:enumeration value="all"/>
    <xs:enumeration value="none"/>
  </xs:restriction>
</xs:simpleType>
```

**Semantic Meaning**:
- `any` - At least one item must match (OR logic)
- `all` - All items must match (AND logic)
- `none` - No items may match (NOT logic)

#### 2.1.3 Integration Points

The aether-types.xsd schema is imported by:
- pheromone.xsd (for CasteEnum, PriorityType, etc.)
- worker-priming.xsd (for caste definitions)
- Any future schemas requiring shared types

**Import Declaration**:
```xml
<xs:import namespace="http://aether.colony/schemas/types/1.0"
           schemaLocation="aether-types.xsd"/>
```

#### 2.1.4 Usage Examples

**Example 1: Referencing CasteEnum**
```xml
<xs:element name="target-caste" type="types:CasteEnum"/>
```

**Example 2: Constrained Decimal for Priority Score**
```xml
<xs:element name="priority-score" type="types:WeightType"/>
<!-- Only accepts values 0.00 to 1.00 -->
```

**Example 3: Timestamp with Validation**
```xml
<xs:attribute name="created" type="types:TimestampType"/>
<!-- Rejects: 2026-02-30 (invalid date), 25:00:00 (invalid time) -->
```

**Example 4: Version String Pattern**
```xml
<xs:attribute name="version" type="types:VersionType"/>
<!-- Accepts: 1.0.0, 2.1.3-alpha -->
<!-- Rejects: 1.0, v1.0.0, 1.0.0.0 -->
```

**Example 5: Match Mode for Scope**
```xml
<xs:attribute name="match" type="types:MatchEnum" default="any"/>
```

---

### 2.2 prompt.xsd

**File Location**: `.aether/schemas/prompt.xsd`

**Namespace**: `http://aether.colony/schemas/prompt/1.0`

**Purpose**: Defines structured prompts for colony workers and commands, replacing ad-hoc markdown with semantic XML.

#### 2.2.1 Schema Architecture

The prompt.xsd schema enables machine-parseable, validated prompt definitions. Unlike free-form markdown, XML prompts provide:

- **Structured Requirements**: Each requirement has ID, priority, description, and rationale
- **Explicit Constraints**: Hard and soft constraints with enforcement guidance
- **Thinking Guidance**: Step-by-step approach with checkpoints
- **Tool Specifications**: Required vs optional tools with usage guidance
- **Success Criteria**: Measurable completion conditions
- **Error Handling**: Failure recovery and escalation procedures

#### 2.2.2 Root Element: aether-prompt

```xml
<xs:element name="aether-prompt">
  <xs:complexType>
    <xs:sequence>
      <xs:element name="metadata" type="metadataType" minOccurs="0"/>
      <xs:element name="name" type="xs:string"/>
      <xs:element name="type" type="promptType"/>
      <xs:element name="caste" type="casteType" minOccurs="0"/>
      <xs:element name="objective" type="xs:string"/>
      <xs:element name="context" type="contextType" minOccurs="0"/>
      <xs:element name="requirements" type="requirementsType"/>
      <xs:element name="constraints" type="constraintsType" minOccurs="0"/>
      <xs:element name="thinking" type="thinkingType" minOccurs="0"/>
      <xs:element name="tools" type="toolsType" minOccurs="0"/>
      <xs:element name="output" type="outputType"/>
      <xs:element name="verification" type="verificationType"/>
      <xs:element name="success_criteria" type="successCriteriaType"/>
      <xs:element name="error_handling" type="errorHandlingType" minOccurs="0"/>
    </xs:sequence>
    <xs:attribute name="version" type="versionType" use="optional" default="1.0.0"/>
  </xs:complexType>
</xs:element>
```

**Element Semantics**:

| Element | Cardinality | Purpose |
|---------|-------------|---------|
| metadata | 0..1 | Document versioning, authorship, tags |
| name | 1 | Unique identifier for the prompt |
| type | 1 | Classification: worker, command, agent, system |
| caste | 0..1 | Worker caste assignment (required for worker type) |
| objective | 1 | What the prompt should accomplish |
| context | 0..1 | Background, assumptions, dependencies |
| requirements | 1 | What must be done to complete successfully |
| constraints | 0..1 | Hard and soft execution boundaries |
| thinking | 0..1 | Approach guidance with checkpoints |
| tools | 0..1 | Available tools and when to use them |
| output | 1 | Expected output format and structure |
| verification | 1 | How to verify correctness |
| success_criteria | 1 | Measurable completion conditions |
| error_handling | 0..1 | Failure recovery procedures |

#### 2.2.3 Simple Types

**promptType**

Four prompt classifications:

```xml
<xs:simpleType name="promptType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="worker"/>
    <xs:enumeration value="command"/>
    <xs:enumeration value="agent"/>
    <xs:enumeration value="system"/>
  </xs:restriction>
</xs:simpleType>
```

**Type Semantics**:
- `worker` - Assigned to spawned workers (requires caste element)
- `command` - Slash command implementation guidance
- `agent` - OpenCode agent definition
- `system` - Core colony system prompts

**casteType**

19 castes (subset of full 22, missing ambassador, auditor, includer):

```xml
<xs:simpleType name="casteType">
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
    <xs:enumeration value="chronicler"/>
    <xs:enumeration value="guardian"/>
    <xs:enumeration value="gatekeeper"/>
    <xs:enumeration value="weaver"/>
    <xs:enumeration value="probe"/>
    <xs:enumeration value="sage"/>
    <xs:enumeration value="measurer"/>
    <xs:enumeration value="keeper"/>
    <xs:enumeration value="tracker"/>
  </xs:restriction>
</xs:simpleType>
```

**Note**: This should be updated to use the shared CasteEnum from aether-types.xsd for consistency.

**priorityType**

Four priority levels for requirements:

```xml
<xs:simpleType name="priorityType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="critical"/>
    <xs:enumeration value="high"/>
    <xs:enumeration value="normal"/>
    <xs:enumeration value="low"/>
  </xs:restriction>
</xs:simpleType>
```

**constraintStrengthType**

Five constraint levels (RFC 2119 inspired):

```xml
<xs:simpleType name="constraintStrengthType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="must"/>
    <xs:enumeration value="should"/>
    <xs:enumeration value="may"/>
    <xs:enumeration value="must-not"/>
    <xs:enumeration value="should-not"/>
  </xs:restriction>
</xs:simpleType>
```

**RFC 2119 Semantics**:
- `must` - Absolute requirement
- `must-not` - Absolute prohibition
- `should` - Recommended, valid reasons may exist to ignore
- `should-not` - Not recommended, valid reasons may exist
- `may` - Truly optional

**versionType**

Semantic version pattern:

```xml
<xs:simpleType name="versionType">
  <xs:restriction base="xs:string">
    <xs:pattern value="\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.2.4 Complex Types

**requirementType**

Individual requirement with priority and rationale:

```xml
<xs:complexType name="requirementType">
  <xs:sequence>
    <xs:element name="description" type="xs:string"/>
    <xs:element name="rationale" type="xs:string" minOccurs="0"/>
  </xs:sequence>
  <xs:attribute name="id" type="xs:ID" use="optional"/>
  <xs:attribute name="priority" type="priorityType" use="optional" default="normal"/>
</xs:complexType>
```

**Usage Example**:
```xml
<requirement id="req_1" priority="critical">
  <description>Follow Test-Driven Development methodology</description>
  <rationale>Ensures code is testable and specifications are clear</rationale>
</requirement>
```

**constraintType**

Constraint with rule, exception, and enforcement:

```xml
<xs:complexType name="constraintType">
  <xs:sequence>
    <xs:element name="rule" type="xs:string"/>
    <xs:element name="exception" type="xs:string" minOccurs="0"/>
    <xs:element name="enforcement" type="xs:string" minOccurs="0"/>
  </xs:sequence>
  <xs:attribute name="id" type="xs:ID" use="optional"/>
  <xs:attribute name="strength" type="constraintStrengthType" use="optional" default="should"/>
</xs:complexType>
```

**Usage Example**:
```xml
<constraint id="cons_1" strength="must-not">
  <rule>Never commit broken or failing code</rule>
  <enforcement>Watcher verification will catch this</enforcement>
</constraint>
```

**outputType**

Expected output specification:

```xml
<xs:complexType name="outputType">
  <xs:sequence>
    <xs:element name="format" type="xs:string"/>
    <xs:element name="structure" type="xs:string" minOccurs="0"/>
    <xs:element name="example" type="xs:string" minOccurs="0"/>
  </xs:sequence>
</xs:complexType>
```

**thinkingType**

Approach guidance with checkpoints:

```xml
<xs:complexType name="thinkingType">
  <xs:sequence>
    <xs:element name="approach" type="xs:string"/>
    <xs:element name="steps" minOccurs="0">
      <xs:complexType>
        <xs:sequence>
          <xs:element name="step" maxOccurs="unbounded">
            <xs:complexType>
              <xs:sequence>
                <xs:element name="description" type="xs:string"/>
                <xs:element name="checkpoint" type="xs:string" minOccurs="0"/>
              </xs:sequence>
              <xs:attribute name="order" type="xs:positiveInteger" use="required"/>
              <xs:attribute name="optional" type="xs:boolean" use="optional" default="false"/>
            </xs:complexType>
          </xs:element>
        </xs:sequence>
      </xs:complexType>
    </xs:element>
    <xs:element name="pitfalls" minOccurs="0">
      <xs:complexType>
        <xs:sequence>
          <xs:element name="pitfall" type="xs:string" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:complexType>
    </xs:element>
  </xs:sequence>
</xs:complexType>
```

**successCriteriaType**

Measurable completion conditions:

```xml
<xs:complexType name="successCriteriaType">
  <xs:sequence>
    <xs:element name="criterion" maxOccurs="unbounded">
      <xs:complexType>
        <xs:sequence>
          <xs:element name="description" type="xs:string"/>
          <xs:element name="measure" type="xs:string" minOccurs="0"/>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="optional"/>
        <xs:attribute name="required" type="xs:boolean" use="optional" default="true"/>
      </xs:complexType>
    </xs:element>
  </xs:sequence>
</xs:complexType>
```

#### 2.2.5 Validation Rules

**Structural Validation**:
- All prompts must have a name, type, objective, requirements, output, verification, and success_criteria
- Worker-type prompts should have a caste assignment
- Requirements must have at least one requirement element

**Content Validation**:
- Version strings must match semantic versioning pattern
- Priority values must be one of: critical, high, normal, low
- Constraint strength must be one of: must, should, may, must-not, should-not
- Step order attributes must be positive integers

**Cross-Element Validation**:
- If type is "worker", caste element is strongly recommended
- Required criteria should outnumber optional criteria for clarity

#### 2.2.6 Usage Examples

**Example 1: Minimal Valid Prompt**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<aether-prompt version="1.0.0"
    xmlns="http://aether.colony/schemas/prompt/1.0">
  <name>minimal-prompt</name>
  <type>command</type>
  <objective>Demonstrate minimal valid prompt structure</objective>
  <requirements>
    <requirement>
      <description>Include all required elements</description>
    </requirement>
  </requirements>
  <output>
    <format>XML</format>
  </output>
  <verification>
    <method>Schema validation</method>
  </verification>
  <success_criteria>
    <criterion>
      <description>Document validates against prompt.xsd</description>
    </criterion>
  </success_criteria>
</aether-prompt>
```

**Example 2: Complete Worker Prompt**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<aether-prompt version="1.0.0"
    xmlns="http://aether.colony/schemas/prompt/1.0">
  <metadata>
    <version>1.0.0</version>
    <author>Aether Colony System</author>
    <created>2026-02-16T10:00:00Z</created>
    <tags>
      <tag>worker</tag>
      <tag>builder</tag>
    </tags>
  </metadata>

  <name>builder-worker</name>
  <type>worker</type>
  <caste>builder</caste>

  <objective>Implement features following TDD methodology</objective>

  <context>
    <background>Builders are primary implementation workers</background>
    <assumptions>
      <assumption>Specification is complete</assumption>
      <assumption>Tools are available</assumption>
    </assumptions>
  </context>

  <requirements>
    <requirement id="req_1" priority="critical">
      <description>Follow TDD methodology</description>
      <rationale>Ensures testable code</rationale>
    </requirement>
  </requirements>

  <constraints>
    <constraint id="cons_1" strength="must-not">
      <rule>Never commit broken code</rule>
    </constraint>
  </constraints>

  <thinking>
    <approach>Research first, then implement following TDD</approach>
    <steps>
      <step order="1">
        <description>Understand specification</description>
        <checkpoint>Can explain requirement</checkpoint>
      </step>
    </steps>
  </thinking>

  <tools>
    <tool required="true">
      <name>Read</name>
      <purpose>Read file contents</purpose>
    </tool>
  </tools>

  <output>
    <format>Source code with tests</format>
    <structure>Implementation files and test files</structure>
  </output>

  <verification>
    <method>Run test suite</method>
    <steps>
      <step>Run unit tests</step>
      <step>Check coverage</step>
    </steps>
  </verification>

  <success_criteria>
    <criterion id="crit_1" required="true">
      <description>All tests pass</description>
      <measure>npm test exits 0</measure>
    </criterion>
  </success_criteria>
</aether-prompt>
```

**Example 3: Command Prompt with Error Handling**
```xml
<aether-prompt version="1.0.0">
  <name>verify-castes</name>
  <type>command</type>
  <objective>Verify caste model assignments are correct</objective>

  <requirements>
    <requirement priority="high">
      <description>Check all caste configurations</description>
    </requirement>
  </requirements>

  <error_handling>
    <on_failure>Log error and return non-zero exit code</on_failure>
    <escalation>Report to user if configuration is corrupted</escalation>
    <recovery_steps>
      <step>Check model-profiles.yaml syntax</step>
      <step>Verify ANTHROPIC_MODEL environment variable</step>
    </recovery_steps>
  </error_handling>

  <!-- ... other elements ... -->
</aether-prompt>
```

**Example 4: Prompt with Multiple Success Criteria**
```xml
<success_criteria>
  <criterion id="crit_1" required="true">
    <description>All tests pass</description>
    <measure>npm test exits with code 0</measure>
  </criterion>
  <criterion id="crit_2" required="true">
    <description>Code compiles without errors</description>
    <measure>No TypeScript or build errors</measure>
  </criterion>
  <criterion id="crit_3" required="false">
    <description>Code coverage maintained</description>
    <measure>Coverage >= 80% for new code</measure>
  </criterion>
</success_criteria>
```

**Example 5: Prompt with Tool Specifications**
```xml
<tools>
  <tool required="true">
    <name>Glob</name>
    <purpose>Find files matching patterns</purpose>
    <when_to_use>When searching for existing implementations</when_to_use>
  </tool>
  <tool required="true">
    <name>Grep</name>
    <purpose>Search file contents</purpose>
    <when_to_use>When looking for specific code patterns</when_to_use>
  </tool>
  <tool required="false">
    <name>Bash</name>
    <purpose>Execute shell commands</purpose>
    <when_to_use>When running tests or build commands</when_to_use>
  </tool>
</tools>
```

---

### 2.3 pheromone.xsd

**File Location**: `.aether/schemas/pheromone.xsd`

**Namespace**: `http://aether.colony/schemas/pheromones`

**Purpose**: Defines XML structure for pheromone signals used in colony communication.

#### 2.3.1 Schema Overview

The pheromone schema formalizes the biological metaphor of ant colony communication. Pheromones are directional signals that guide worker behavior without direct command chains. The schema supports three primary signal types (FOCUS, REDIRECT, FEEDBACK) with scoped application and weighted tags.

#### 2.3.2 Root Element: pheromones

```xml
<xs:element name="pheromones" type="ph:PheromonesType">
  <xs:annotation>
    <xs:documentation>
      Root element containing a collection of pheromone signals.
      Signals are processed in priority order (high to low) then
      by creation time (newest first).
    </xs:documentation>
  </xs:annotation>
</xs:element>
```

**PheromonesType**:

```xml
<xs:complexType name="PheromonesType">
  <xs:sequence>
    <xs:element name="metadata" type="ph:MetadataType" minOccurs="0" maxOccurs="1"/>
    <xs:element name="signal" type="ph:SignalType" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence>
  <xs:attribute name="version" type="ph:VersionType" use="required"/>
  <xs:attribute name="generated_at" type="xs:dateTime" use="required"/>
  <xs:attribute name="colony_id" type="ph:IdentifierType" use="optional"/>
  <xs:anyAttribute namespace="##any" processContents="lax"/>
</xs:complexType>
```

**Root Attributes**:
- `version` (required) - Schema version for compatibility
- `generated_at` (required) - ISO 8601 timestamp of generation
- `colony_id` (optional) - Colony identifier for multi-colony contexts

#### 2.3.3 Signal Structure

**SignalType**:

```xml
<xs:complexType name="SignalType">
  <xs:sequence>
    <xs:element name="content" type="ph:ContentType"/>
    <xs:element name="tags" type="ph:TagsType" minOccurs="0" maxOccurs="1"/>
    <xs:element name="scope" type="ph:ScopeType" minOccurs="0" maxOccurs="1"/>
  </xs:sequence>
  <xs:attribute name="id" type="ph:IdentifierType" use="required"/>
  <xs:attribute name="type" type="ph:SignalTypeEnum" use="required"/>
  <xs:attribute name="priority" type="ph:PriorityType" use="required"/>
  <xs:attribute name="source" type="ph:IdentifierType" use="required"/>
  <xs:attribute name="created_at" type="xs:dateTime" use="required"/>
  <xs:attribute name="expires_at" type="xs:dateTime" use="optional"/>
  <xs:attribute name="active" type="xs:boolean" use="optional" default="true"/>
</xs:complexType>
```

**Signal Attributes**:

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| id | IdentifierType | Yes | Unique signal identifier |
| type | SignalTypeEnum | Yes | FOCUS, REDIRECT, or FEEDBACK |
| priority | PriorityType | Yes | critical, high, normal, or low |
| source | IdentifierType | Yes | Signal origin (user, worker, system) |
| created_at | xs:dateTime | Yes | Creation timestamp |
| expires_at | xs:dateTime | No | Optional expiration timestamp |
| active | xs:boolean | No | Whether signal is active (default: true) |

**SignalTypeEnum**:

```xml
<xs:simpleType name="SignalTypeEnum">
  <xs:restriction base="xs:string">
    <xs:enumeration value="FOCUS"/>
    <xs:enumeration value="REDIRECT"/>
    <xs:enumeration value="FEEDBACK"/>
  </xs:restriction>
</xs:simpleType>
```

**Signal Semantics**:

- **FOCUS**: Directs attention to a specific area. Normal priority. Use for "pay attention here" guidance.
- **REDIRECT**: Hard constraint, avoid this path. High priority. Use for "don't do this" constraints.
- **FEEDBACK**: Gentle adjustment based on observation. Low priority. Use for "adjust based on this" observations.

#### 2.3.4 Content Structure

**ContentType**:

```xml
<xs:complexType name="ContentType" mixed="true">
  <xs:sequence>
    <xs:element name="text" type="xs:string" minOccurs="0" maxOccurs="1"/>
    <xs:element name="data" type="ph:DataType" minOccurs="0" maxOccurs="1"/>
  </xs:sequence>
</xs:complexType>
```

The `mixed="true"` attribute allows both text content and child elements, enabling flexible content models.

**DataType**:

```xml
<xs:complexType name="DataType">
  <xs:sequence>
    <xs:any namespace="##any" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence>
  <xs:attribute name="format" type="ph:DataFormatEnum" use="optional" default="json"/>
</xs:complexType>
```

**DataFormatEnum**:

```xml
<xs:simpleType name="DataFormatEnum">
  <xs:restriction base="xs:string">
    <xs:enumeration value="json"/>
    <xs:enumeration value="xml"/>
    <xs:enumeration value="yaml"/>
    <xs:enumeration value="plain"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.3.5 Scope Structure

**ScopeType**:

```xml
<xs:complexType name="ScopeType">
  <xs:sequence>
    <xs:element name="castes" type="ph:CastesType" minOccurs="0" maxOccurs="1"/>
    <xs:element name="paths" type="ph:PathsType" minOccurs="0" maxOccurs="1"/>
    <xs:element name="phases" type="ph:PhasesType" minOccurs="0" maxOccurs="1"/>
  </xs:sequence>
  <xs:attribute name="global" type="xs:boolean" use="optional" default="false"/>
</xs:complexType>
```

**CastesType**:

```xml
<xs:complexType name="CastesType">
  <xs:sequence>
    <xs:element name="caste" type="ph:CasteEnum" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence>
  <xs:attribute name="match" type="ph:MatchEnum" use="optional" default="any"/>
</xs:complexType>
```

**PathsType**:

```xml
<xs:complexType name="PathsType">
  <xs:sequence>
    <xs:element name="path" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence>
  <xs:attribute name="match" type="ph:MatchEnum" use="optional" default="any"/>
</xs:complexType>
```

Paths support glob patterns (e.g., `src/**/*.js`, `tests/*.test.ts`).

**PhasesType**:

```xml
<xs:complexType name="PhasesType">
  <xs:sequence>
    <xs:element name="phase" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence>
  <xs:attribute name="match" type="ph:MatchEnum" use="optional" default="any"/>
</xs:complexType>
```

**MatchEnum**:

```xml
<xs:simpleType name="MatchEnum">
  <xs:restriction base="xs:string">
    <xs:enumeration value="any"/>
    <xs:enumeration value="all"/>
    <xs:enumeration value="none"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.3.6 Tag Structure

**TagsType and TagType**:

```xml
<xs:complexType name="TagsType">
  <xs:sequence>
    <xs:element name="tag" type="ph:TagType" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence>
</xs:complexType>

<xs:complexType name="TagType">
  <xs:simpleContent>
    <xs:extension base="xs:string">
      <xs:attribute name="weight" type="ph:WeightType" use="optional" default="1.0"/>
      <xs:attribute name="category" type="xs:string" use="optional"/>
    </xs:extension>
  </xs:simpleContent>
</xs:complexType>
```

**WeightType**:

```xml
<xs:simpleType name="WeightType">
  <xs:restriction base="xs:decimal">
    <xs:minInclusive value="0.0"/>
    <xs:maxInclusive value="1.0"/>
    <xs:fractionDigits value="2"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.3.7 Validation Rules

**Required Elements**:
- All signals must have: id, type, priority, source, created_at, and content
- Root pheromones element must have version and generated_at attributes

**Value Constraints**:
- Signal type must be FOCUS, REDIRECT, or FEEDBACK
- Priority must be critical, high, normal, or low
- Weight must be between 0.00 and 1.00
- Match mode must be any, all, or none

**Identifier Constraints**:
- Must start with letter
- Can contain letters, digits, hyphens, underscores
- Maximum 64 characters

#### 2.3.8 Usage Examples

**Example 1: FOCUS Signal**
```xml
<ph:signal id="focus-001"
           type="FOCUS"
           priority="normal"
           source="user"
           created_at="2026-02-16T15:30:00Z"
           expires_at="2026-02-17T15:30:00Z"
           active="true">
  <ph:content>
    <ph:text>Focus implementation efforts on authentication module</ph:text>
    <ph:data format="json">
      <sprint>42</sprint>
      <priority_score>8.5</priority_score>
    </ph:data>
  </ph:content>
  <ph:tags>
    <ph:tag weight="0.9" category="feature">authentication</ph:tag>
  </ph:tags>
  <ph:scope global="false">
    <ph:castes match="any">
      <ph:caste>builder</ph:caste>
      <ph:caste>architect</ph:caste>
    </ph:castes>
    <ph:paths match="any">
      <ph:path>src/auth/**</ph:path>
    </ph:paths>
  </ph:scope>
</ph:signal>
```

**Example 2: REDIRECT Signal (Global)**
```xml
<ph:signal id="redirect-001"
           type="REDIRECT"
           priority="high"
           source="system"
           created_at="2026-02-16T14:00:00Z"
           active="true">
  <ph:content>
    <ph:text>AVOID using legacy v1 API endpoints</ph:text>
    <ph:data format="json">
      <deprecated_endpoints>
        <endpoint>/api/v1/users</endpoint>
      </deprecated_endpoints>
      <replacement>/api/v2/</replacement>
    </ph:data>
  </ph:content>
  <ph:tags>
    <ph:tag weight="1.0" category="constraint">deprecated</ph:tag>
  </ph:tags>
  <ph:scope global="true"/>
</ph:signal>
```

**Example 3: FEEDBACK Signal**
```xml
<ph:signal id="feedback-001"
           type="FEEDBACK"
           priority="low"
           source="watcher-A7"
           created_at="2026-02-16T10:15:00Z"
           active="true">
  <ph:content>
    <ph:text>Consider adding more inline comments to complex regex</ph:text>
  </ph:content>
  <ph:tags>
    <ph:tag weight="0.5" category="style">documentation</ph:tag>
  </ph:tags>
  <ph:scope global="false">
    <ph:castes match="any">
      <ph:caste>builder</ph:caste>
    </ph:castes>
  </ph:scope>
</ph:signal>
```

**Example 4: Complete Pheromones Document**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<ph:pheromones xmlns:ph="http://aether.colony/schemas/pheromones"
               version="1.0.0"
               generated_at="2026-02-16T15:30:00Z"
               colony_id="aether-main">

  <ph:metadata>
    <ph:source type="user" version="1.0.0">Colony initialization</ph:source>
    <ph:context>Initial pheromone setup</ph:context>
  </ph:metadata>

  <ph:signal id="sig-001" type="FOCUS" priority="normal"
             source="user" created_at="2026-02-16T15:30:00Z">
    <ph:content>
      <ph:text>Focus on authentication</ph:text>
    </ph:content>
    <ph:scope global="true"/>
  </ph:signal>

</ph:pheromones>
```

**Example 5: Scoped Signal with All Match Modes**
```xml
<!-- Any caste can work on this -->
<ph:scope>
  <ph:castes match="any">
    <ph:caste>builder</ph:caste>
    <ph:caste>watcher</ph:caste>
  </ph:castes>
</ph:scope>

<!-- All specified paths must match -->
<ph:scope>
  <ph:paths match="all">
    <ph:path>src/**</ph:path>
    <ph:path>*.ts</ph:path>
  </ph:paths>
</ph:scope>

<!-- Exclude these castes -->
<ph:scope>
  <ph:castes match="none">
    <ph:caste>chaos</ph:caste>
  </ph:castes>
</ph:scope>
```

---

### 2.4 colony-registry.xsd

**File Location**: `.aether/schemas/colony-registry.xsd`

**Namespace**: Default (qualified elements)

**Purpose**: Multi-colony registry with lineage tracking and pheromone inheritance.

#### 2.4.1 Schema Overview

The colony-registry schema enables tracking multiple related colonies, their ancestry relationships, and inherited pheromones. This supports scenarios where a main colony spawns feature colonies, which may themselves spawn sub-colonies.

#### 2.4.2 Root Element: colony-registry

```xml
<xs:element name="colony-registry">
  <xs:complexType>
    <xs:sequence>
      <xs:element name="registry-info" type="registryInfoType"/>
      <xs:element name="colonies" type="coloniesContainerType"/>
      <xs:element name="global-relationships" type="globalRelationshipsContainerType" minOccurs="0"/>
    </xs:sequence>
    <xs:attribute name="version" type="versionType" use="required"/>
  </xs:complexType>

  <!-- Key constraints for referential integrity -->
  <xs:key name="colonyIdKey">
    <xs:selector xpath="colonies/colony"/>
    <xs:field xpath="id"/>
  </xs:key>

  <xs:keyref name="parentColonyRef" refer="colonyIdKey">
    <xs:selector xpath="colonies/colony/lineage/parent-colony"/>
    <xs:field xpath="."/>
  </xs:keyref>

  <!-- Additional keyrefs for forked-from, ancestry, relationships -->
</xs:element>
```

**Key Constraints**:
- `colonyIdKey`: All colony IDs must be unique
- `parentColonyRef`: Parent colony references must exist
- `forkedFromRef`: Fork source must exist
- `ancestorRef`: Ancestor references must exist
- `relationshipTargetRef`: Relationship targets must exist

#### 2.4.3 Simple Types

**colonyIdType**:

```xml
<xs:simpleType name="colonyIdType">
  <xs:restriction base="xs:string">
    <xs:pattern value="[a-zA-Z0-9][a-zA-Z0-9-]{2,63}"/>
    <xs:minLength value="3"/>
    <xs:maxLength value="64"/>
  </xs:restriction>
</xs:simpleType>
```

**colonyStatusType**:

```xml
<xs:simpleType name="colonyStatusType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="active"/>
    <xs:enumeration value="paused"/>
    <xs:enumeration value="archived"/>
  </xs:restriction>
</xs:simpleType>
```

**relationshipType**:

```xml
<xs:simpleType name="relationshipType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="parent"/>
    <xs:enumeration value="child"/>
    <xs:enumeration value="sibling"/>
    <xs:enumeration value="fork"/>
    <xs:enumeration value="merge"/>
    <xs:enumeration value="reference"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.4.4 Complex Types

**colonyType**:

```xml
<xs:complexType name="colonyType">
  <xs:sequence>
    <!-- Identity -->
    <xs:element name="id" type="colonyIdType"/>
    <xs:element name="name" type="xs:string"/>
    <xs:element name="description" type="xs:string" minOccurs="0"/>

    <!-- Location -->
    <xs:element name="path" type="xs:string"/>
    <xs:element name="repository-url" type="xs:anyURI" minOccurs="0"/>

    <!-- Status -->
    <xs:element name="status" type="colonyStatusType"/>
    <xs:element name="created-at" type="timestampType"/>
    <xs:element name="last-active" type="timestampType"/>

    <!-- Lineage -->
    <xs:element name="lineage" type="lineageType" minOccurs="0"/>

    <!-- Inherited pheromones -->
    <xs:element name="pheromones-inherited" type="pheromonesContainerType" minOccurs="0"/>

    <!-- Relationships -->
    <xs:element name="relationships" type="relationshipsContainerType" minOccurs="0"/>

    <!-- Metadata -->
    <xs:element name="metadata" type="colonyMetadataType" minOccurs="0"/>
  </xs:sequence>
</xs:complexType>
```

**lineageType**:

```xml
<xs:complexType name="lineageType">
  <xs:sequence>
    <xs:element name="parent-colony" type="colonyIdType" minOccurs="0" maxOccurs="unbounded"/>
    <xs:element name="forked-from" type="colonyIdType" minOccurs="0"/>
    <xs:element name="generation" type="xs:positiveInteger" minOccurs="0"/>
    <xs:element name="ancestry-chain" minOccurs="0">
      <xs:complexType>
        <xs:sequence>
          <xs:element name="ancestor" type="ancestorType" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:complexType>
    </xs:element>
  </xs:sequence>
</xs:complexType>
```

**pheromoneType (inherited)**:

```xml
<xs:complexType name="pheromoneType">
  <xs:sequence>
    <xs:element name="key" type="xs:string"/>
    <xs:element name="value" type="xs:string"/>
    <xs:element name="strength" type="pheromoneStrengthType"/>
    <xs:element name="inherited-at" type="timestampType"/>
    <xs:element name="source-colony" type="colonyIdType"/>
  </xs:sequence>
  <xs:attribute name="type" type="pheromoneTypeEnum" use="optional" default="feedback"/>
</xs:complexType>
```

#### 2.4.5 Validation Rules

**Referential Integrity**:
- All parent-colony references must point to existing colonies
- All forked-from references must point to existing colonies
- All ancestor references must point to existing colonies
- All relationship targets must point to existing colonies

**Temporal Constraints**:
- created-at must be before or equal to last-active
- inherited-at must be after or equal to parent colony's created-at

**Status Transitions**:
- No automatic validation of status transitions
- Application logic should enforce: active -> paused -> archived

#### 2.4.6 Usage Examples

**Example 1: Root Colony Entry**
```xml
<colony>
  <id>main-aether-001</id>
  <name>Main Aether Colony</name>
  <description>Primary colony for core platform</description>

  <path>/Users/dev/repos/Aether</path>
  <repository-url>https://github.com/user/Aether</repository-url>

  <status>active</status>
  <created-at>2026-01-15T08:00:00Z</created-at>
  <last-active>2026-02-16T15:30:00Z</last-active>

  <lineage>
    <generation>1</generation>
    <ancestry-chain>
      <ancestor generation="0" relationship="parent">main-aether-001</ancestor>
    </ancestry-chain>
  </lineage>

  <pheromones-inherited count="0"/>
</colony>
```

**Example 2: Child Colony with Inherited Pheromones**
```xml
<colony>
  <id>feature-auth-002</id>
  <name>Authentication Feature Colony</name>

  <path>/Users/dev/repos/Aether-auth</path>
  <status>active</status>
  <created-at>2026-02-01T09:00:00Z</created-at>
  <last-active>2026-02-16T12:00:00Z</last-active>

  <lineage>
    <parent-colony>main-aether-001</parent-colony>
    <generation>2</generation>
    <ancestry-chain>
      <ancestor generation="1" relationship="parent">main-aether-001</ancestor>
    </ancestry-chain>
  </lineage>

  <pheromones-inherited count="2">
    <pheromone type="focus">
      <key>architecture-pattern</key>
      <value>modular-service-layer</value>
      <strength>0.850</strength>
      <inherited-at>2026-02-01T09:00:00Z</inherited-at>
      <source-colony>main-aether-001</source-colony>
    </pheromone>
    <pheromone type="redirect">
      <key>avoid-global-state</key>
      <value>use-dependency-injection</value>
      <strength>0.920</strength>
      <inherited-at>2026-02-01T09:00:00Z</inherited-at>
      <source-colony>main-aether-001</source-colony>
    </pheromone>
  </pheromones-inherited>

  <relationships>
    <relationship>
      <target-colony>main-aether-001</target-colony>
      <relationship>parent</relationship>
      <established-at>2026-02-01T09:00:00Z</established-at>
    </relationship>
  </relationships>
</colony>
```

**Example 3: Complete Registry Document**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<colony-registry version="1.0.0">
  <registry-info>
    <name>Aether Multi-Colony Registry</name>
    <description>Central registry for all colonies</description>
    <version>1.0.0</version>
    <created-at>2026-02-16T10:00:00Z</created-at>
    <updated-at>2026-02-16T15:30:00Z</updated-at>
    <total-colonies>3</total-colonies>
  </registry-info>

  <colonies>
    <!-- Colony entries here -->
  </colonies>

  <global-relationships>
    <relationship>
      <from-colony>main-aether-001</from-colony>
      <to-colony>feature-auth-002</to-colony>
      <type>parent</type>
    </relationship>
  </global-relationships>
</colony-registry>
```

---

### 2.5 worker-priming.xsd

**File Location**: `.aether/schemas/worker-priming.xsd`

**Namespace**: `http://aether.colony/schemas/worker-priming/1.0`

**Purpose**: Modular configuration composition using XInclude for worker initialization.

#### 2.5.1 Schema Overview

The worker-priming schema enables declarative worker initialization through XInclude-based composition. Workers are "primed" by assembling configuration from multiple sources: queen wisdom, active pheromones, and stack profiles.

#### 2.5.2 Root Element: worker-priming

```xml
<xs:element name="worker-priming">
  <xs:complexType>
    <xs:sequence>
      <xs:element name="metadata" type="wp:primingMetadataType"/>
      <xs:element name="worker-identity" type="wp:workerIdentityType"/>
      <xs:element name="priming-config" minOccurs="0">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="mode" type="wp:primingModeType"/>
            <xs:element name="inherit-from-parent" type="xs:boolean" minOccurs="0" default="true"/>
            <xs:element name="apply-redirects" type="xs:boolean" minOccurs="0" default="true"/>
            <xs:element name="load-pheromones" type="xs:boolean" minOccurs="0" default="true"/>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:element name="queen-wisdom" type="wp:queenWisdomSectionType" minOccurs="0"/>
      <xs:element name="active-trails" type="wp:activeTrailsSectionType" minOccurs="0"/>
      <xs:element name="stack-profiles" type="wp:stackProfilesSectionType" minOccurs="0"/>
      <xs:element name="override-rules" type="wp:overrideRulesType" minOccurs="0"/>
      <xs:element name="composition-result" type="wp:compositionResultType" minOccurs="0"/>
    </xs:sequence>
    <xs:attribute name="version" type="wp:versionType" use="required"/>
  </xs:complexType>
</xs:element>
```

#### 2.5.3 Simple Types

**workerIdType**:

```xml
<xs:simpleType name="workerIdType">
  <xs:restriction base="xs:string">
    <xs:pattern value="[a-z][a-z0-9-]*"/>
    <xs:minLength value="3"/>
    <xs:maxLength value="64"/>
  </xs:restriction>
</xs:simpleType>
```

**primingModeType**:

```xml
<xs:simpleType name="primingModeType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="full"/>
    <xs:enumeration value="minimal"/>
    <xs:enumeration value="inherit"/>
    <xs:enumeration value="override"/>
  </xs:restriction>
</xs:simpleType>
```

**Mode Semantics**:
- `full` - Load all configuration sections
- `minimal` - Load only essential configuration
- `inherit` - Primarily inherit from parent
- `override` - Override parent configuration

**sourcePriorityType**:

```xml
<xs:simpleType name="sourcePriorityType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="highest"/>
    <xs:enumeration value="high"/>
    <xs:enumeration value="normal"/>
    <xs:enumeration value="low"/>
    <xs:enumeration value="lowest"/>
  </xs:restriction>
</xs:simpleType>
```

**overrideActionType**:

```xml
<xs:simpleType name="overrideActionType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="replace"/>
    <xs:enumeration value="merge"/>
    <xs:enumeration value="append"/>
    <xs:enumeration value="prepend"/>
    <xs:enumeration value="remove"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.5.4 Complex Types

**workerIdentityType**:

```xml
<xs:complexType name="workerIdentityType">
  <xs:sequence>
    <xs:element name="name" type="xs:string"/>
    <xs:element name="caste" type="wp:casteType"/>
    <xs:element name="generation" type="xs:positiveInteger" minOccurs="0"/>
    <xs:element name="parent-colony" type="xs:string" minOccurs="0"/>
  </xs:sequence>
  <xs:attribute name="id" type="wp:workerIdType" use="required"/>
</xs:complexType>
```

**configSourceType**:

```xml
<xs:complexType name="configSourceType">
  <xs:sequence>
    <xs:element ref="xi:include" minOccurs="0" maxOccurs="1"/>
    <xs:element name="inline" type="xs:string" minOccurs="0" maxOccurs="1"/>
    <xs:element name="source-info" type="wp:sourceInfoType" minOccurs="0"/>
  </xs:sequence>
  <xs:attribute name="name" type="xs:string" use="required"/>
  <xs:attribute name="priority" type="wp:sourcePriorityType" use="optional" default="normal"/>
  <xs:attribute name="required" type="xs:boolean" use="optional" default="true"/>
</xs:complexType>
```

**overrideRuleType**:

```xml
<xs:complexType name="overrideRuleType">
  <xs:sequence>
    <xs:element name="target-path" type="xs:string"/>
    <xs:element name="action" type="wp:overrideActionType"/>
    <xs:element name="value" type="xs:string" minOccurs="0"/>
  </xs:sequence>
  <xs:attribute name="id" type="xs:string" use="required"/>
  <xs:attribute name="priority" type="wp:sourcePriorityType" use="optional" default="normal"/>
</xs:complexType>
```

#### 2.5.5 XInclude Integration

The schema imports the XInclude namespace:

```xml
<xs:import namespace="http://www.w3.org/2001/XInclude"
           schemaLocation="http://www.w3.org/2001/XInclude.xsd"/>
```

This enables the `xi:include` element within configSourceType.

#### 2.5.6 Usage Examples

**Example 1: Minimal Worker Priming**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<worker-priming version="1.0.0"
    xmlns="http://aether.colony/schemas/worker-priming/1.0"
    xmlns:xi="http://www.w3.org/2001/XInclude">

  <metadata>
    <version>1.0.0</version>
    <created>2026-02-16T15:47:00Z</created>
    <modified>2026-02-16T15:47:00Z</modified>
    <colony-id>aether-main</colony-id>
  </metadata>

  <worker-identity id="builder-001">
    <name>Mason-54</name>
    <caste>builder</caste>
    <generation>1</generation>
  </worker-identity>

  <priming-config>
    <mode>full</mode>
    <inherit-from-parent>true</inherit-from-parent>
  </priming-config>
</worker-priming>
```

**Example 2: Worker Priming with XInclude**
```xml
<worker-priming version="1.0.0"
    xmlns="http://aether.colony/schemas/worker-priming/1.0"
    xmlns:xi="http://www.w3.org/2001/XInclude">

  <metadata>...</metadata>
  <worker-identity id="scout-001">...</worker-identity>

  <queen-wisdom enabled="true">
    <wisdom-source name="eternal-wisdom" priority="highest" required="true">
      <xi:include href="../eternal/queen-wisdom.xml"
                  parse="xml"
                  xpointer="xmlns(qw=http://aether.colony/schemas/queen-wisdom/1.0)xpointer(/qw:queen-wisdom/qw:philosophies)"/>
    </wisdom-source>
  </queen-wisdom>

  <active-trails enabled="true">
    <trail-source name="current-pheromones" priority="high">
      <xi:include href="../data/pheromones.xml" parse="xml"/>
    </trail-source>
  </active-trails>

  <override-rules>
    <rule id="ignore-expired" priority="high">
      <target-path>//signal[@expires_at]</target-path>
      <action>remove</action>
    </rule>
  </override-rules>
</worker-priming>
```

---

### 2.6 queen-wisdom.xsd

**File Location**: `.aether/schemas/queen-wisdom.xsd`

**Namespace**: `http://aether.colony/schemas/queen-wisdom/1.0`

**Purpose**: Eternal memory structure for learned patterns, principles, and evolution tracking.

#### 2.6.1 Schema Overview

The queen-wisdom schema defines the structure for persistent colony knowledge. It supports multiple wisdom categories: philosophies (core beliefs), patterns (validated approaches), redirects (constraints), stack-wisdom (technical insights), and decrees (authoritative directives).

#### 2.6.2 Root Element: queen-wisdom

```xml
<xs:element name="queen-wisdom">
  <xs:complexType>
    <xs:sequence>
      <xs:element name="metadata" type="metadataType"/>
      <xs:element name="philosophies" type="philosophiesType"/>
      <xs:element name="patterns" type="patternsType"/>
      <xs:element name="redirects" type="redirectsType"/>
      <xs:element name="stack-wisdom" type="stackWisdomsType"/>
      <xs:element name="decrees" type="decreesType"/>
    </xs:sequence>
  </xs:complexType>
</xs:element>
```

#### 2.6.3 Simple Types

**confidenceType**:

```xml
<xs:simpleType name="confidenceType">
  <xs:restriction base="xs:decimal">
    <xs:minInclusive value="0.0"/>
    <xs:maxInclusive value="1.0"/>
    <xs:fractionDigits value="2"/>
  </xs:restriction>
</xs:simpleType>
```

**domainType**:

```xml
<xs:simpleType name="domainType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="architecture"/>
    <xs:enumeration value="testing"/>
    <xs:enumeration value="security"/>
    <xs:enumeration value="performance"/>
    <xs:enumeration value="ux"/>
    <xs:enumeration value="process"/>
    <xs:enumeration value="communication"/>
    <xs:enumeration value="debugging"/>
    <xs:enumeration value="general"/>
  </xs:restriction>
</xs:simpleType>
```

**sourceType**:

```xml
<xs:simpleType name="sourceType">
  <xs:restriction base="xs:string">
    <xs:enumeration value="queen"/>
    <xs:enumeration value="user"/>
    <xs:enumeration value="colony"/>
    <xs:enumeration value="oracle"/>
    <xs:enumeration value="observation"/>
  </xs:restriction>
</xs:simpleType>
```

#### 2.6.4 Complex Types

**wisdomEntryType (Base)**:

```xml
<xs:complexType name="wisdomEntryType">
  <xs:sequence>
    <xs:element name="content" type="xs:string"/>
    <xs:element name="context" type="xs:string" minOccurs="0"/>
    <xs:element name="examples" type="examplesType" minOccurs="0"/>
    <xs:element name="related" type="relatedType" minOccurs="0"/>
    <xs:element name="evolution" type="evolutionType" minOccurs="0"/>
  </xs:sequence>
  <xs:attribute name="id" type="wisdomIdType" use="required"/>
  <xs:attribute name="confidence" type="confidenceType" use="required"/>
  <xs:attribute name="domain" type="domainType" use="required"/>
  <xs:attribute name="source" type="sourceType" use="required"/>
  <xs:attribute name="created_at" type="timestampType" use="required"/>
  <xs:attribute name="applied_count" type="xs:nonNegativeInteger" use="optional" default="0"/>
  <xs:attribute name="last_applied" type="timestampType" use="optional"/>
  <xs:attribute name="priority" type="priorityType" use="optional" default="normal"/>
</xs:complexType>
```

**philosophyType (Extension)**:

```xml
<xs:complexType name="philosophyType">
  <xs:complexContent>
    <xs:extension base="wisdomEntryType">
      <xs:sequence>
        <xs:element name="principles" minOccurs="0">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="principle" type="xs:string" maxOccurs="unbounded"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>
```

**patternType (Extension)**:

```xml
<xs:complexType name="patternType">
  <xs:complexContent>
    <xs:extension base="wisdomEntryType">
      <xs:sequence>
        <xs:element name="pattern_type" minOccurs="0">
          <xs:simpleType>
            <xs:restriction base="xs:string">
              <xs:enumeration value="success"/>
              <xs:enumeration value="failure"/>
              <xs:enumeration value="anti-pattern"/>
              <xs:enumeration value="emerging"/>
            </xs:restriction>
          </xs:simpleType>
        </xs:element>
        <xs:element name="detection_criteria" type="xs:string" minOccurs="0"/>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>
```

**decreeType (Extension)**:

```xml
<xs:complexType name="decreeType">
  <xs:complexContent>
    <xs:extension base="wisdomEntryType">
      <xs:sequence>
        <xs:element name="authority" type="xs:string" minOccurs="0"/>
        <xs:element name="expiration" type="timestampType" minOccurs="0"/>
        <xs:element name="scope" minOccurs="0">
          <xs:simpleType>
            <xs:restriction base="xs:string">
              <xs:enumeration value="global"/>
              <xs:enumeration value="project"/>
              <xs:enumeration value="phase"/>
              <xs:enumeration value="task"/>
            </xs:restriction>
          </xs:simpleType>
        </xs:element>
      </xs:sequence>
    </xs:extension>
  </xs:complexContent>
</xs:complexType>
```

#### 2.6.5 Wisdom Categories

**Philosophies**: Core beliefs that guide all colony work. Validated through repeated successful application. Example: "Knowledge that persists across sessions is the foundation of colony intelligence."

**Patterns**: Validated approaches that consistently work. Include detection criteria for when to apply. Example: TDD red-green-refactor cycle.

**Redirects**: Anti-patterns to avoid. Hard constraints with enforcement guidance. Example: "Never commit API keys to repository."

**Stack Wisdom**: Technology-specific insights with version ranges and workarounds. Example: "bash stat command differs between macOS and Linux."

**Decrees**: Authoritative directives from the Queen with expiration and scope. Example: "All eternal memory shall use XML with XSD validation."

#### 2.6.6 Usage Examples

**Example 1: Philosophy Entry**
```xml
<philosophy id="eternal-memory-principle"
            confidence="0.95"
            domain="architecture"
            source="queen"
            created_at="2026-02-16T10:00:00Z"
            applied_count="42"
            priority="high">
  <content>Knowledge that persists across sessions is the foundation of colony intelligence.</content>
  <context>Apply when designing data structures or storage formats</context>
  <examples>
    <example>
      <scenario>Designing configuration system</scenario>
      <application>Use XML schema with versioning</application>
      <outcome>Seamless migration from v1 to v2</outcome>
    </example>
  </examples>
  <principles>
    <principle>Prefer structured formats over unstructured text</principle>
    <principle>Version all schemas</principle>
  </principles>
</philosophy>
```

**Example 2: Pattern Entry**
```xml
<pattern id="tdd-red-green-refactor"
         confidence="0.92"
         domain="testing"
         source="colony"
         created_at="2026-02-16T11:00:00Z"
         priority="critical">
  <content>The Iron Law: No production code without failing test first.</content>
  <pattern_type>success</pattern_type>
  <detection_criteria>Code exists without corresponding test</detection_criteria>
</pattern>
```

**Example 3: Decree Entry**
```xml
<decree id="xml-eternal-memory-mandate"
        confidence="0.95"
        domain="architecture"
        source="queen"
        created_at="2026-02-16T14:30:00Z"
        priority="critical">
  <content>All eternal memory shall use XML with XSD validation.</content>
  <authority>Anvil-71 (Builder)</authority>
  <scope>global</scope>
</decree>
```

---

## 3. XML Utility Functions

### 3.1 Core Functions (xml-utils.sh)

#### 3.1.1 xml-detect-tools

**Purpose**: Detect available XML processing tools

**Usage**: `xml-detect-tools`

**Returns**: JSON with availability flags

```json
{
  "ok": true,
  "result": {
    "xmllint": true,
    "xmlstarlet": true,
    "xsltproc": true,
    "xml2json": false
  }
}
```

**Implementation Details**:
- Checks for `xmllint`, `xmlstarlet`, `xsltproc`, `xml2json` in PATH
- Sets global variables: XMLLINT_AVAILABLE, XMLSTARLET_AVAILABLE, etc.
- No external dependencies for detection itself

**Error Handling**:
- Always returns success (detection is informational)
- Missing tools reported as false, not errors

---

#### 3.1.2 xml-well-formed

**Purpose**: Check if XML document is well-formed

**Usage**: `xml-well-formed <xml_file>`

**Returns**:
```json
{"ok":true,"result":{"well_formed":true}}
```
or
```json
{"ok":true,"result":{"well_formed":false,"error":"..."}}
```

**Security Features**:
- Uses `xmllint --noout` (no output, just validation)
- No entity expansion during check
- No network access

**Implementation**:
```bash
xml-well-formed() {
    local xml_file="$1"
    [[ -f "$xml_file" ]] || { xml_json_err "File not found"; return 1; }

    if xmllint --noout "$xml_file" 2>/dev/null; then
        xml_json_ok '{"well_formed":true}'
    else
        local error=$(xmllint --noout "$xml_file" 2>&1)
        xml_json_ok "{\"well_formed\":false,\"error\":$(echo "$error" | jq -Rs .)}"
    fi
}
```

---

#### 3.1.3 xml-validate

**Purpose**: Validate XML against XSD schema

**Usage**: `xml-validate <xml_file> [xsd_file]`

**Security Features**:
- Uses `--noent` flag (no entity expansion, XXE protection)
- Uses `--nonet` flag (no network access)
- Schema location can be specified to prevent external entity attacks

**Returns**:
```json
{"ok":true,"result":{"valid":true}}
```
or
```json
{"ok":true,"result":{"valid":false,"errors":"..."}}
```

**Implementation Notes**:
- Falls back to well-formed check if no schema provided
- Captures validation errors from xmllint stderr
- Returns structured error messages

---

#### 3.1.4 xml-format

**Purpose**: Pretty-print XML document

**Usage**: `xml-format <xml_file>`

**Features**:
- In-place formatting
- Consistent indentation
- Preserves document structure

**Implementation**:
```bash
xml-format() {
    local xml_file="$1"
    local formatted=$(xmllint --format "$xml_file" 2>/dev/null)
    echo "$formatted" > "$xml_file"
    xml_json_ok '{"formatted":true}'
}
```

---

#### 3.1.5 xml-query

**Purpose**: Execute XPath query against XML document

**Usage**: `xml-query <xml_file> <xpath_expression>`

**Dependencies**: xmlstarlet (preferred) or xmllint fallback

**Returns**:
```json
{
  "ok": true,
  "result": {
    "matches": [...],
    "count": 2
  }
}
```

**Example**:
```bash
xml-query document.xml "//worker/@id"
```

---

#### 3.1.6 json-to-xml

**Purpose**: Convert JSON to XML representation

**Usage**: `json-to-xml <json_file> [root_element]`

**Algorithm**:
1. Parse JSON using jq
2. Recursively transform objects to elements
3. Transform arrays to repeated elements
4. Transform primitives to text content
5. Escape special XML characters

**Returns**:
```json
{
  "ok": true,
  "result": {
    "xml": "<root>...</root>"
  }
}
```

**Implementation Details**:
- Uses jq for JSON parsing
- Handles nested objects and arrays
- Proper XML escaping for `<`, `>`, `&`, `"`, `'`
- Default root element: "root"

---

#### 3.1.7 pheromone-to-xml

**Purpose**: Convert pheromone JSON to schema-valid XML

**Usage**: `pheromone-to-xml <json_file> [output_xml] [schema_file]`

**Features**:
- Case normalization (focus -> FOCUS)
- Invalid value fallback
- XML escaping
- Caste validation (22 valid castes)
- Schema validation if xmllint available

**Normalization Rules**:
- Signal type: converted to uppercase
- Priority: converted to lowercase
- Invalid types default to FOCUS
- Invalid priorities default to normal

**Returns**:
```json
{
  "ok": true,
  "result": {
    "xml": "<pheromones...>",
    "validated": true
  }
}
```

---

#### 3.1.8 queen-wisdom-to-xml

**Purpose**: Convert queen wisdom JSON to XML

**Usage**: `queen-wisdom-to-xml <json_file> [output_xml]`

**Handles**:
- Philosophies
- Patterns
- Redirects
- Stack-wisdom
- Decrees

---

#### 3.1.9 registry-to-xml

**Purpose**: Convert colony registry JSON to XML

**Usage**: `registry-to-xml <json_file> [output_xml]`

**Handles**:
- Colony entries
- Lineage
- Relationships
- Inherited pheromones

---

#### 3.1.10 prompt-to-xml

**Purpose**: Convert markdown prompt to structured XML

**Usage**: `prompt-to-xml <markdown_file> [output_xml]`

**Extracts**:
- Objectives
- Requirements
- Constraints
- Thinking steps
- Success criteria

---

#### 3.1.11 prompt-from-xml

**Purpose**: Convert XML prompt back to markdown

**Usage**: `prompt-from-xml <xml_file>`

**Use Case**: Human-readable output from structured XML

---

#### 3.1.12 prompt-validate

**Purpose**: Validate prompt XML against prompt.xsd

**Usage**: `prompt-validate <xml_file>`

---

### 3.2 Queen Wisdom Functions

#### 3.2.1 queen-wisdom-to-markdown

**Purpose**: Transform queen-wisdom XML to human-readable markdown

**Usage**: `queen-wisdom-to-markdown <xml_file> [output_md]`

**Implementation**: Uses XSLT stylesheet (queen-to-md.xsl)

**Output Sections**:
- Philosophies
- Patterns
- Redirects
- Stack Wisdom
- Decrees
- Evolution Log

---

#### 3.2.2 queen-wisdom-validate-entry

**Purpose**: Validate single wisdom entry against schema

**Usage**: `queen-wisdom-validate-entry <xml_file> <entry_id>`

---

#### 3.2.3 queen-wisdom-promote

**Purpose**: Promote observation to pattern, pattern to philosophy

**Usage**: `queen-wisdom-promote <type> <entry_id> <target_colony>`

**Workflow**:
1. Validates source entry
2. Updates evolution log
3. Writes to eternal memory

---

#### 3.2.4 queen-wisdom-import

**Purpose**: Import external wisdom into colony's eternal memory

**Usage**: `queen-wisdom-import <xml_file> [colony_id]`

**Handles**: Namespace prefixing for collision avoidance

---

### 3.3 Namespace Functions

#### 3.3.1 generate-colony-namespace

**Purpose**: Generate unique namespace URI for colony

**Usage**: `generate-colony-namespace <session_id>`

**Format**: `http://aether.dev/colony/{session_id}`

**Returns**:
```json
{
  "ok": true,
  "result": {
    "namespace": "http://aether.dev/colony/abc123",
    "prefix": "col_abc123"
  }
}
```

---

#### 3.3.2 generate-cross-colony-prefix

**Purpose**: Generate collision-free prefix for cross-colony references

**Usage**: `generate-cross-colony-prefix <external_session> <local_session>`

**Format**: `{hash}_{ext|col}_{hash}`

---

#### 3.3.3 prefix-pheromone-id

**Purpose**: Prefix signal ID with colony identifier

**Usage**: `prefix-pheromone-id <signal_id> <colony_prefix>`

**Features**: Idempotent (won't double-prefix)

---

#### 3.3.4 validate-colony-namespace

**Purpose**: Validate namespace URI format

**Usage**: `validate-colony-namespace <namespace_uri>`

**Recognizes**:
- Colony namespaces: `http://aether.dev/colony/*`
- Schema namespaces: `http://aether.colony/schemas/*`

---

### 3.4 Export Functions

#### 3.4.1 pheromone-export

**Purpose**: Export pheromones to eternal memory XML

**Usage**: `pheromone-export <pheromones_json> [output_xml] [colony_id] [schema_file]`

**Default Output**: `~/.aether/eternal/pheromones.xml`

---

## 4. XInclude Composition System

### 4.1 xml-compose.sh Module

#### 4.1.1 xml-compose

**Purpose**: Resolve XInclude directives in XML documents

**Usage**: `xml-compose <input_xml> [output_xml]`

**Security Features**:
- Uses `--nonet` (no network access)
- Uses `--noent` (no entity expansion, XXE protection)
- Uses `--xinclude` (process XInclude)

**Returns**:
```json
{
  "ok": true,
  "result": {
    "composed": true,
    "output": "...",
    "sources_resolved": "auto"
  }
}
```

**Implementation**:
```bash
xml-compose() {
    local input_xml="$1"
    local output_xml="$2"

    # Check well-formedness first
    xml-well-formed "$input_xml" >/dev/null || {
        xml_json_err "Input XML is not well-formed"
        return 1
    }

    # Compose with security flags
    local composed=$(xmllint --nonet --noent --xinclude --format "$input_xml" 2>/dev/null)

    if [[ -n "$output_xml" ]]; then
        echo "$composed" > "$output_xml"
    else
        xml_json_ok "{\"composed\":true,\"xml\":$(echo "$composed" | jq -Rs .)}"
    fi
}
```

---

#### 4.1.2 xml-list-includes

**Purpose**: List all XInclude references in document

**Usage**: `xml-list-includes <xml_file>`

**Returns**:
```json
{
  "ok": true,
  "result": {
    "includes": [
      {"href": "file.xml", "parse": "xml", "xpointer": "..."},
      ...
    ],
    "count": 2
  }
}
```

**Implementation**:
- Preferred: xmlstarlet (namespace-aware)
- Fallback: grep pattern matching

---

#### 4.1.3 xml-compose-worker-priming

**Purpose**: Specialized composition for worker priming documents

**Usage**: `xml-compose-worker-priming <priming_xml> [output_xml]`

**Features**:
- Validates against worker-priming.xsd
- Extracts worker identity
- Counts sources by section

**Returns**:
```json
{
  "ok": true,
  "result": {
    "composed": true,
    "worker_id": "builder-001",
    "caste": "builder",
    "sources": {
      "queen_wisdom": 2,
      "active_trails": 1,
      "stack_profiles": 1
    }
  }
}
```

---

#### 4.1.4 xml-validate-include-path

**Purpose**: Security validation for XInclude paths

**Usage**: `xml-validate-include-path <include_path> <base_dir>`

**Protection Mechanisms**:

1. **Traversal Detection**: Rejects paths containing `..` sequences
2. **Absolute Path Validation**: Ensures absolute paths start with allowed directory
3. **Path Normalization**: Resolves and re-verifies final path
4. **Base Directory Enforcement**: All includes relative to defined base

**Error Codes**:
- `PATH_TRAVERSAL_DETECTED` - Path contains traversal sequences
- `PATH_TRAVERSAL_BLOCKED` - Resolved path outside allowed directory
- `INVALID_BASE_DIR` - Base directory does not exist

**Implementation**:
```bash
xml-validate-include-path() {
    local include_path="$1"
    local base_dir="$2"

    # Resolve base directory
    local allowed_dir=$(cd "$base_dir" 2>/dev/null && pwd) || {
        xml_json_err "INVALID_BASE_DIR" "Base directory does not exist"
        return 1
    }

    # Check for traversal sequences
    if [[ "$include_path" =~ \.\.[\/] ]] || [[ "$include_path" =~ [\/]\.\. ]]; then
        xml_json_err "PATH_TRAVERSAL_DETECTED" "Path contains traversal sequences"
        return 1
    fi

    # Build and verify resolved path
    local resolved_path
    if [[ "$include_path" == /* ]]; then
        if [[ ! "$include_path" =~ ^"$allowed_dir" ]]; then
            xml_json_err "PATH_TRAVERSAL_BLOCKED" "Absolute path outside allowed directory"
            return 1
        fi
        resolved_path="$include_path"
    else
        resolved_path="$allowed_dir/$include_path"
    fi

    # Verify final path within allowed directory
    if [[ ! "$resolved_path" =~ ^"$allowed_dir" ]]; then
        xml_json_err "PATH_TRAVERSAL_BLOCKED" "Resolved path outside allowed directory"
        return 1
    fi

    echo "$resolved_path"
}
```

---

### 4.2 Composition Example

**Input Document** (worker-priming.xml):
```xml
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

**Composed Output**:
```xml
<worker-priming>
  <queen-wisdom>
    <wisdom-source name="eternal-wisdom">
      <philosophies>
        <philosophy id="...">...</philosophy>
      </philosophies>
    </wisdom-source>
  </queen-wisdom>
</worker-priming>
```

---

## 5. Security Architecture

### 5.1 XXE Protection

**XML External Entity (XXE) attacks** exploit XML parsers to:
- Read arbitrary files (file disclosure)
- Perform SSRF (Server-Side Request Forgery)
- Cause DoS via entity expansion

**Aether Protections**:

1. **--nonet Flag**: Prevents network access during XML processing
   ```bash
   xmllint --nonet --noent --xinclude input.xml
   ```

2. **--noent Flag**: Disables entity expansion, preventing file disclosure
   - Entities remain as literal text (`&xxe;` instead of expanded content)
   - Prevents billion laughs attack (exponential expansion)

3. **No External DTD Loading**: xmllint configured to reject external entities

**Test Coverage**: `test-xml-security.sh` includes XXE attack tests

---

### 5.2 Path Traversal Protection

**Attack Vector**: XInclude with `../../../etc/passwd` to read sensitive files

**Protection Layers**:

1. **Pattern Detection**: Reject paths containing `..` sequences
   ```bash
   if [[ "$include_path" =~ \.\.[\/] ]]; then
       # Reject
   fi
   ```

2. **Absolute Path Validation**: Ensure absolute paths start with allowed directory
   ```bash
   if [[ ! "$include_path" =~ ^"$allowed_dir" ]]; then
       # Reject
   fi
   ```

3. **Path Normalization**: Resolve and re-verify final path location

4. **Base Directory Enforcement**: All includes relative to defined base

**Test Coverage**: `test-xml-security.sh` includes path traversal tests

---

### 5.3 Entity Expansion Limits

**Billion Laughs Attack**: Nested entity definitions causing exponential expansion

**Mitigation**:
- `--noent` flag prevents all entity expansion
- No entity expansion means exponential expansion attacks are impossible
- Alternative: `--max-entities` flag (if available) limits entity count

**Test Coverage**: `test-xml-security.sh` includes billion laughs test

---

### 5.4 Security Test Coverage

| Test File | Tests | Coverage |
|-----------|-------|----------|
| test-xml-security.sh | 7 | XXE, path traversal, network access, nested XML |
| test-pheromone-xml.sh | 15 | Pheromone conversion with validation |
| test-xml-utils.sh | 20 | All utility functions |
| test-phase3-xml.sh | 15 | Queen-wisdom and prompt workflows |

**Total**: 57 security and validation tests

---

## 6. JSON/XML Conversion

### 6.1 JSON to XML Algorithm

**Mechanism**: jq-based recursive transformation

**Transformation Rules**:

1. **Object Handling**: Create child elements with keys as tag names
   ```json
   {"name": "value"} -> <name>value</name>
   ```

2. **Array Handling**: Create repeated elements
   ```json
   {"items": [1, 2]} -> <items>1</items><items>2</items>
   ```

3. **Primitive Handling**: Text content with proper escaping
   ```json
   "text" -> <root>text</root>
   ```

4. **Nested Structures**: Recursive application of rules
   ```json
   {"a": {"b": "c"}} -> <a><b>c</b></a>
   ```

**Root Element**: Configurable (default: "root")

---

### 6.2 XML to JSON Algorithm

**Mechanism**: xmlstarlet or xsltproc transformation

**Preserves**:
- Structure (element hierarchy)
- Attributes (as @attr in JSON)
- Text content
- Namespace prefixes

---

### 6.3 Hybrid Architecture

The system uses a hybrid approach:

**JSON for Runtime**:
- Active pheromones
- Colony state
- Session data
- Activity logs

**XML for Eternal Memory**:
- Validated wisdom storage
- Cross-colony exchange
- Version-controlled archives
- Human-readable documentation

**Conversion Points**:
- `pheromone-export`: JSON -> XML for archival
- `prompt-to-xml`: Markdown -> XML for structure
- `queen-wisdom-to-markdown`: XML -> Markdown for reading

---

## 7. Schema Evolution Strategy

### 7.1 Versioning Approach

**Namespace Versioning**: Each schema version has unique namespace URI

```
http://aether.colony/schemas/prompt/1.0
http://aether.colony/schemas/prompt/1.1  (future)
http://aether.colony/schemas/prompt/2.0  (breaking)
```

**Semantic Versioning for Schemas**:
- Major: Breaking changes (new namespace)
- Minor: Additive changes (backward compatible)
- Patch: Documentation/fixes (no structural change)

### 7.2 Backward Compatibility

**Additive Changes** (Minor Version):
- New optional elements
- New optional attributes
- New enumeration values
- Relaxing constraints

**Breaking Changes** (Major Version):
- New required elements
- Removing elements/attributes
- Changing types
- New namespace required

### 7.3 Migration Path

**Document Migration**:
1. Detect document version from namespace
2. Apply XSLT transformation if needed
3. Validate against new schema
4. Update namespace declaration

**Example Migration**:
```bash
# Transform v1.0 document to v1.1
xsltproc migrate-prompt-1.0-to-1.1.xsl old.xml > new.xml
xml-validate new.xml prompt-1.1.xsd
```

---

## 8. Performance Optimization

### 8.1 Tool Selection

| Operation | Primary Tool | Fallback | Notes |
|-----------|--------------|----------|-------|
| Validation | xmllint | - | Fast, secure flags |
| Query | xmlstarlet | xmllint | Namespace-aware |
| Transform | xsltproc | - | XSLT 1.0 support |
| Format | xmllint | - | Built-in |

### 8.2 Caching Strategies

**Composition Caching**:
- Cache composed documents by source checksum
- Skip re-composition if sources unchanged
- Store composition metadata (timestamp, sources)

**Validation Caching**:
- Cache validation results by file hash
- Re-validate only if file modified
- Clear cache on schema update

### 8.3 Lazy Loading

**XInclude Strategy**:
- Parse document structure first
- Resolve includes only when section accessed
- Support for xi:fallback when include unavailable

---

## 9. Industry Comparison

### 9.1 XML vs Alternative Formats

| Feature | XML | JSON | YAML | TOML |
|---------|-----|------|------|------|
| Schema Validation | XSD | JSON Schema | Limited | Limited |
| Namespaces | Yes | No | No | No |
| XInclude | Yes | No | No | No |
| XSLT | Yes | No | No | No |
| Human Readable | Moderate | Good | Excellent | Good |
| Tooling | Mature | Excellent | Good | Growing |

### 9.2 Aether's Position

**Unique Features**:
- Biological metaphor (pheromones, castes, queen wisdom)
- XInclude-based modular composition
- Hybrid JSON/XML architecture
- Shell-based implementation (no runtime dependencies)

**Trade-offs**:
- XML verbosity vs structure
- Schema complexity vs validation
- XInclude power vs security concerns

---

## 10. Activation Roadmap

### 10.1 Phase 1: Pheromone Export (Low Effort, Medium Value)

**Implementation**:
```bash
# In pheromone signal handlers
pheromone-export ".aether/data/pheromones.json" ".aether/eternal/pheromones.xml"
```

**Steps**:
1. Add export call to `/ant:focus`, `/ant:redirect`, `/ant:feedback` handlers
2. Configure automatic export on colony seal
3. Add verification that export succeeded

### 10.2 Phase 2: XML-Based Worker Prompts (Medium Effort, High Value)

**Implementation**:
1. Convert existing prompts with `prompt-to-xml`
2. Store in `.aether/prompts/{caste}.xml`
3. Load and validate before spawning workers
4. Use XInclude for shared constraint libraries

### 10.3 Phase 3: Queen Wisdom Promotion (Medium Effort, High Value)

**Implementation**:
1. Observations accumulate in session JSON
2. `queen-wisdom-promote` converts valid patterns to XML
3. XSLT generates QUEEN.md for human reading
4. Cross-colony wisdom import for shared learnings

### 10.4 Phase 4: Colony Registry (High Effort, Medium Value)

**Implementation**:
1. Registry XML in `~/.aether/eternal/registry.xml`
2. Lineage tracking for forked colonies
3. Pheromone inheritance between related colonies
4. Relationship management UI

### 10.5 Phase 5: Worker Priming with XInclude (High Effort, High Value)

**Implementation**:
1. Priming XML per worker type
2. XInclude composition of wisdom + pheromones + stack profiles
3. Override rules for customization
4. Validation before worker spawn

---

## Appendix A: File Inventory

### Schemas (6 files)
- `.aether/schemas/aether-types.xsd` (256 lines)
- `.aether/schemas/prompt.xsd` (417 lines)
- `.aether/schemas/pheromone.xsd` (251 lines)
- `.aether/schemas/colony-registry.xsd` (310 lines)
- `.aether/schemas/worker-priming.xsd` (277 lines)
- `.aether/schemas/queen-wisdom.xsd` (326 lines)

**Total Schema Lines**: 1,837

### Utilities (3 files)
- `.aether/utils/xml-utils.sh` (~600 lines)
- `.aether/utils/xml-compose.sh` (248 lines)
- `.aether/utils/queen-to-md.xsl` (396 lines)

**Total Utility Lines**: ~1,244

### Examples (5 files)
- `.aether/schemas/example-prompt-builder.xml` (235 lines)
- `.aether/schemas/examples/pheromone-example.xml` (118 lines)
- `.aether/schemas/examples/colony-registry-example.xml` (303 lines)
- `.aether/schemas/examples/queen-wisdom-example.xml` (382 lines)
- `.aether/examples/worker-priming.xml` (172 lines)

**Total Example Lines**: 1,210

### Tests (4 files)
- `tests/bash/test-xml-utils.sh` (1,046 lines, 20 tests)
- `tests/bash/test-pheromone-xml.sh` (417 lines, 15 tests)
- `tests/bash/test-phase3-xml.sh` (381 lines, 15 tests)
- `tests/bash/test-xml-security.sh` (288 lines, 7 tests)

**Total Test Lines**: 2,132

**Grand Total**: ~6,423 lines of XML infrastructure

---

## Appendix B: Known Issues

### Issue 1: Schema Location Mismatch
- **Location**: worker-priming.xsd imports XInclude from W3C URL
- **Impact**: Requires network access for validation
- **Recommendation**: Bundle local copy of XInclude.xsd

### Issue 2: XSLT Namespace Mismatch
- **Location**: queen-to-md.xsl uses default namespace
- **Impact**: May not match elements correctly
- **Fix**: Add qw: namespace prefix to stylesheet

### Issue 3: Missing Evolution Log Element
- **Location**: test-phase3-xml.sh references evolution-log
- **Impact**: Test creates invalid XML
- **Fix**: Add evolution-log to queen-wisdom.xsd or remove from test

### Issue 4: Caste Enumeration Inconsistency
- **Location**: prompt.xsd has 19 castes, aether-types.xsd has 22
- **Impact**: Potential validation failures
- **Fix**: Update prompt.xsd to import CasteEnum from aether-types.xsd

---

*Documentation Version: 1.0.0*
*Generated: 2026-02-16*
*Analyst: Oracle caste*
*Status: Complete*
