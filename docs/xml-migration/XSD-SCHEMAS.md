# XSD Schemas

**Formal Definitions for Aether XML Documents**

---

## Overview

XSD (XML Schema Definition) provides strict validation for Aether's eternal memory. Unlike optional JSON Schema, XSD validation is enforced at parse time.

---

## Schema 1: Pheromone Trail

File: `http://aether.dev/schema/pheromone.xsd`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.dev/core/pheromones/v1"
           xmlns:aether="http://aether.dev/core/pheromones/v1"
           elementFormDefault="qualified">

    <!-- Enumerations -->
    <xs:simpleType name="PheromoneType">
        <xs:restriction base="xs:string">
            <xs:enumeration value="FOCUS"/>
            <xs:enumeration value="REDIRECT"/>
            <xs:enumeration value="PHILOSOPHY"/>
            <xs:enumeration value="PATTERN"/>
            <xs:enumeration value="STACK"/>
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

    <xs:complexType name="Context">
        <xs:sequence>
            <xs:element name="domain" type="xs:string" minOccurs="0"/>
            <xs:element name="architecture" type="xs:string" minOccurs="0"/>
            <xs:element name="scale" type="xs:string" minOccurs="0"/>
            <xs:element name="stack" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
            <xs:element name="constraints" minOccurs="0">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="constraint" type="xs:string" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
        </xs:sequence>
    </xs:complexType>

    <xs:complexType name="ColonyRef">
        <xs:attribute name="id" type="xs:string" use="required"/>
        <xs:attribute name="confidence" type="aether:Strength"/>
        <xs:attribute name="timestamp" type="xs:dateTime"/>
    </xs:complexType>

    <xs:complexType name="Trail">
        <xs:sequence>
            <xs:element name="substance" type="xs:string"/>
            <xs:element name="strength" type="aether:Strength"/>
            <xs:element name="source" type="aether:Source"/>
            <xs:element name="context" type="aether:Context" minOccurs="0"/>
            <xs:element name="validation-count" type="xs:nonNegativeInteger" minOccurs="0"/>
            <xs:element name="validations" minOccurs="0">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="colony-ref" type="aether:ColonyRef" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
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
            <xs:attribute name="merged" type="xs:boolean" default="false"/>
        </xs:complexType>
    </xs:element>

</xs:schema>
```

---

## Schema 2: Queen Wisdom

File: `http://aether.dev/schema/queen-wisdom.xsd`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.dev/core/queen/v1"
           xmlns:queen="http://aether.dev/core/queen/v1"
           elementFormDefault="qualified">

    <xs:simpleType name="WisdomType">
        <xs:restriction base="xs:string">
            <xs:enumeration value="PHILOSOPHY"/>
            <xs:enumeration value="PATTERN"/>
            <xs:enumeration value="REDIRECT"/>
            <xs:enumeration value="STACK"/>
            <xs:enumeration value="DECREE"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:simpleType name="WisdomStatus">
        <xs:restriction base="xs:string">
            <xs:enumeration value="candidate"/>
            <xs:enumeration value="validated"/>
            <xs:enumeration value="instinct"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:complexType name="ColonyRef">
        <xs:attribute name="id" type="xs:string" use="required"/>
        <xs:attribute name="phase" type="xs:string"/>
        <xs:attribute name="timestamp" type="xs:dateTime"/>
    </xs:complexType>

    <xs:complexType name="Philosophy">
        <xs:sequence>
            <xs:element name="belief" type="xs:string"/>
            <xs:element name="rationale" type="xs:string"/>
            <xs:element name="evidence">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="colony-ref" type="queen:ColonyRef" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="required"/>
        <xs:attribute name="status" type="queen:WisdomStatus" default="candidate"/>
        <xs:attribute name="threshold" type="xs:positiveInteger" default="5"/>
    </xs:complexType>

    <xs:complexType name="Pattern">
        <xs:sequence>
            <xs:element name="description" type="xs:string"/>
            <xs:element name="applies-to" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
            <xs:element name="example" type="xs:string" minOccurs="0"/>
            <xs:element name="evidence">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="colony-ref" type="queen:ColonyRef" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="required"/>
        <xs:attribute name="status" type="queen:WisdomStatus" default="candidate"/>
        <xs:attribute name="threshold" type="xs:positiveInteger" default="3"/>
    </xs:complexType>

    <xs:complexType name="Redirect">
        <xs:sequence>
            <xs:element name="pattern" type="xs:string"/>
            <xs:element name="reason" type="xs:string"/>
            <xs:element name="alternative" type="xs:string"/>
            <xs:element name="evidence">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="colony-ref" type="queen:ColonyRef" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="required"/>
        <xs:attribute name="status" type="queen:WisdomStatus" default="candidate"/>
        <xs:attribute name="threshold" type="xs:positiveInteger" default="2"/>
        <xs:attribute name="strength" type="xs:decimal"/>
    </xs:complexType>

    <xs:complexType name="EvolutionEntry">
        <xs:attribute name="timestamp" type="xs:dateTime" use="required"/>
        <xs:attribute name="colony" type="xs:string" use="required"/>
        <xs:attribute name="action" type="xs:string" use="required"/>
        <xs:attribute name="type" type="xs:string" use="required"/>
        <xs:attribute name="details" type="xs:string"/>
    </xs:complexType>

    <xs:element name="queen-wisdom">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="philosophies" minOccurs="0">
                    <xs:complexType>
                        <xs:sequence>
                            <xs:element name="philosophy" type="queen:Philosophy" maxOccurs="unbounded"/>
                        </xs:sequence>
                    </xs:complexType>
                </xs:element>
                <xs:element name="patterns" minOccurs="0">
                    <xs:complexType>
                        <xs:sequence>
                            <xs:element name="pattern" type="queen:Pattern" maxOccurs="unbounded"/>
                        </xs:sequence>
                    </xs:complexType>
                </xs:element>
                <xs:element name="redirects" minOccurs="0">
                    <xs:complexType>
                        <xs:sequence>
                            <xs:element name="redirect" type="queen:Redirect" maxOccurs="unbounded"/>
                        </xs:sequence>
                    </xs:complexType>
                </xs:element>
                <xs:element name="evolution-log" minOccurs="0">
                    <xs:complexType>
                        <xs:sequence>
                            <xs:element name="entry" type="queen:EvolutionEntry" maxOccurs="unbounded"/>
                        </xs:sequence>
                    </xs:complexType>
                </xs:element>
            </xs:sequence>
            <xs:attribute name="version" type="xs:string" use="required"/>
            <xs:attribute name="evolved-at" type="xs:dateTime"/>
        </xs:complexType>
    </xs:element>

</xs:schema>
```

---

## Schema 3: Colony Registry

File: `http://aether.dev/schema/registry.xsd`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.dev/core/registry/v1"
           xmlns:registry="http://aether.dev/core/registry/v1"
           elementFormDefault="qualified">

    <xs:simpleType name="ColonyStatus">
        <xs:restriction base="xs:string">
            <xs:enumeration value="active"/>
            <xs:enumeration value="sealed"/>
            <xs:enumeration value="archived"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:complexType name="Lineage">
        <xs:sequence>
            <xs:element name="parent" minOccurs="0">
                <xs:complexType>
                    <xs:attribute name="ref" type="xs:string" use="required"/>
                </xs:complexType>
            </xs:element>
            <xs:element name="children" minOccurs="0">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="child" maxOccurs="unbounded">
                            <xs:complexType>
                                <xs:attribute name="ref" type="xs:string" use="required"/>
                            </xs:complexType>
                        </xs:element>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
            <xs:element name="inheritance" minOccurs="0">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="pheromone" maxOccurs="unbounded">
                            <xs:complexType>
                                <xs:simpleContent>
                                    <xs:extension base="xs:string">
                                        <xs:attribute name="type" type="xs:string"/>
                                    </xs:extension>
                                </xs:simpleContent>
                            </xs:complexType>
                        </xs:element>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
        </xs:sequence>
    </xs:complexType>

    <xs:complexType name="Override">
        <xs:sequence>
            <xs:element name="reason" type="xs:string"/>
            <xs:element name="replacement" type="xs:string"/>
        </xs:sequence>
        <xs:attribute name="pattern" type="xs:string" use="required"/>
    </xs:complexType>

    <xs:complexType name="Colony">
        <xs:sequence>
            <xs:element name="goal" type="xs:string"/>
            <xs:element name="lineage" type="registry:Lineage" minOccurs="0"/>
            <xs:element name="overrides" minOccurs="0">
                <xs:complexType>
                    <xs:sequence>
                        <xs:element name="override" type="registry:Override" maxOccurs="unbounded"/>
                    </xs:sequence>
                </xs:complexType>
            </xs:element>
        </xs:sequence>
        <xs:attribute name="id" type="xs:ID" use="required"/>
        <xs:attribute name="status" type="registry:ColonyStatus" use="required"/>
        <xs:attribute name="created-at" type="xs:dateTime"/>
        <xs:attribute name="sealed-at" type="xs:dateTime"/>
    </xs:complexType>

    <xs:element name="colony-registry">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="colony" type="registry:Colony" maxOccurs="unbounded"/>
            </xs:sequence>
            <xs:attribute name="current" type="xs:string"/>
            <xs:attribute name="version" type="xs:string" default="1.0"/>
        </xs:complexType>
    </xs:element>

</xs:schema>
```

---

## Schema 4: Context Metadata

File: `http://aether.dev/schema/context.xsd`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://aether.dev/core/context/v1"
           xmlns:context="http://aether.dev/core/context/v1"
           elementFormDefault="qualified">

    <xs:simpleType name="Domain">
        <xs:restriction base="xs:string">
            <xs:enumeration value="web-api"/>
            <xs:enumeration value="mobile-app"/>
            <xs:enumeration value="cli-tool"/>
            <xs:enumeration value="microservice"/>
            <xs:enumeration value="frontend"/>
            <xs:enumeration value="data-pipeline"/>
            <xs:enumeration value="library"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:simpleType name="Architecture">
        <xs:restriction base="xs:string">
            <xs:enumeration value="stateless"/>
            <xs:enumeration value="stateful"/>
            <xs:enumeration value="event-driven"/>
            <xs:enumeration value="serverless"/>
            <xs:enumeration value="monolithic"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:simpleType name="Scale">
        <xs:restriction base="xs:string">
            <xs:enumeration value="single-instance"/>
            <xs:enumeration value="distributed"/>
            <xs:enumeration value="global"/>
            <xs:enumeration value="high-traffic"/>
        </xs:restriction>
    </xs:simpleType>

    <xs:element name="colony-context">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="domain" type="context:Domain" minOccurs="0"/>
                <xs:element name="architecture" type="context:Architecture" minOccurs="0"/>
                <xs:element name="scale" type="context:Scale" minOccurs="0"/>
                <xs:element name="stack" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
                <xs:element name="constraints" minOccurs="0">
                    <xs:complexType>
                        <xs:sequence>
                            <xs:element name="constraint" type="xs:string" maxOccurs="unbounded"/>
                        </xs:sequence>
                    </xs:complexType>
                </xs:element>
            </xs:sequence>
            <xs:attribute name="colony-id" type="xs:string" use="required"/>
        </xs:complexType>
    </xs:element>

</xs:schema>
```

---

## Validation Examples

### Valid Pheromone Document
```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="http://aether.dev/core/pheromones/v1"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://aether.dev/core/pheromones/v1
                        http://aether.dev/schema/pheromone.xsd"
    generated="2026-02-16T10:30:00Z"
    colony-id="colony-abc123">

    <trail id="phem-001" type="PHILOSOPHY" decay="never">
        <substance>Test-driven development ensures quality</substance>
        <strength>0.95</strength>
        <source>
            <command>/ant:init</command>
            <timestamp>2026-02-16T10:30:00Z</timestamp>
        </source>
    </trail>

</pheromone-trails>
```

### Invalid Document (Will Fail Validation)
```xml
<!-- ERROR: type must be one of the enumerated values -->
<trail id="phem-001" type="OPINION">  <!-- Invalid type -->

<!-- ERROR: strength must be 0.0-1.0 -->
<strength>1.5</strength>  <!-- Out of range -->

<!-- ERROR: decay must match pattern -->
<trail id="phem-002" decay="someday">  <!-- Invalid format -->
```

---

## Validation Commands

### Using xmllint
```bash
# Validate against schema
xmllint --schema pheromone.xsd pheromones.xml --noout

# Result: No output = valid
#         Error message = invalid

# Validate with verbose output
xmllint --schema pheromone.xsd pheromones.xml
```

### Using aether-utils.sh
```bash
# Validate wrapper command
aether xml-validate pheromones.xml --schema pheromone

# Expected output:
# {"valid": true, "errors": []}
# or
# {"valid": false, "errors": [...]}
```

---

## Schema Evolution

### Versioning Strategy
1. **Backward compatible changes:** Add optional elements/attributes → minor version bump
2. **Breaking changes:** Remove elements, make optional required → major version bump

### Example Evolution
```xml
<!-- v1.0 -->
<trail id="..." type="...">
  <substance>...</substance>
</trail>

<!-- v1.1 (backward compatible) -->
<trail id="..." type="...">
  <substance>...</substance>
  <context><!-- NEW, optional --></context>
</trail>

<!-- v2.0 (breaking) -->
<trail id="..." type="..." priority="..."><!-- NEW required attribute -->
  <substance>...</substance>
</trail>
```

---

*This document is part of the Aether XML documentation suite.*
