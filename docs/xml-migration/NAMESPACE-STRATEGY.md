# Namespace Strategy

**How Aether Prevents Collisions and Organizes Wisdom**

---

## Why Namespaces Matter

Without namespaces, merging pheromones from multiple colonies is dangerous:

```json
// Colony A: pheromones.json
{ "trails": [{ "id": "auth-001", "advice": "use-jwt" }] }

// Colony B: pheromones.json
{ "trails": [{ "id": "auth-001", "advice": "use-sessions" }] }

// Merged: Which auth-001 wins?
```

With XML namespaces, both coexist peacefully:
```xml
<colony-a:trail id="auth-001">use-jwt</colony-a:trail>
<colony-b:trail id="auth-001">use-sessions</colony-b:trail>
```

---

## Namespace Hierarchy

Aether uses a reverse-DNS naming convention with clear semantic levels:

```
http://aether.dev/{scope}/{identifier}/{version}
```

### Scope Levels

| Scope | Purpose | Example |
|-------|---------|---------|
| `core` | System-wide, built-in | `http://aether.dev/core/v1` |
| `colony` | Specific colony's wisdom | `http://aether.dev/colony/abc123/v1` |
| `template` | Reusable templates | `http://aether.dev/template/nodejs-api/v1` |
| `shared` | Community wisdom | `http://aether.dev/shared/best-practices/v1` |
| `user` | Personal instincts | `http://aether.dev/user/johndoe/v1` |

---

## The Five Namespace Types

### 1. Core Namespace (System)
```xml
<pheromone-trails
    xmlns="http://aether.dev/core/pheromones/v1"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
```

**Purpose:** System-defined pheromone types and structures
**Controlled by:** Aether system
**Usage:** Every document includes this as the default namespace

---

### 2. Colony Namespaces (Private)
```xml
<pheromone-trails
    xmlns="http://aether.dev/core/pheromones/v1"
    xmlns:colony-a="http://aether.dev/colony/abc123/v1"
    xmlns:colony-b="http://aether.dev/colony/def456/v1">
```

**Purpose:** Isolate each colony's pheromones
**Generated from:** Colony ID (from COLONY_STATE.json)
**Example IDs:**
- `colony-20260215-abc123`
- `session_1739623456`

**Usage in documents:**
```xml
<!-- Colony A's own pheromones (default namespace) -->
<trail id="phem-001" type="PHILOSOPHY">
  <substance>Test-driven development</substance>
</trail>

<!-- Imported from Colony B -->
<colony-b:trail id="phem-001" type="PATTERN">
  <colony-b:substance>Always validate inputs</colony-b:substance>
</colony-b:trail>
```

---

### 3. Template Namespaces (Reusable)
```xml
<pheromone-trails
    xmlns:template="http://aether.dev/template/nodejs-api/v1"
    xmlns:template-auth="http://aether.dev/template/auth-service/v1">
```

**Purpose:** Shareable pattern libraries
**Examples:**
- `template/nodejs-api` — Node.js web API patterns
- `template/react-frontend` — React component patterns
- `template/postgresql` — Database design patterns
- `template/auth-service` — Authentication patterns

**Usage:**
```xml
<!-- Include template patterns -->
<template:pattern id="express-routing">
  <template:applies-to>express.js</template:applies-to>
  <template:code-example>app.get('/api/users', handler)</template:code-example>
</template:pattern>
```

---

### 4. Shared Namespaces (Community)
```xml
<pheromone-trails
    xmlns:shared="http://aether.dev/shared/best-practices/v1"
    xmlns:security="http://aether.dev/shared/security/v1">
```

**Purpose:** Curated wisdom from multiple colonies
**Governance:** Community-validated (5+ colony validations)
**Examples:**
- `shared/best-practices` — Universal patterns
- `shared/security` — Security-focused patterns
- `shared/performance` — Optimization patterns

---

### 5. User Namespaces (Personal)
```xml
<pheromone-trails
    xmlns:me="http://aether.dev/user/johndoe/v1">
```

**Purpose:** Individual developer's instincts
**Location:** `~/.aether/eternal/instincts.xml`
**Usage:** Personal preferences that follow you across projects

---

## Namespace Declaration Patterns

### Single-Colony Document
```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="http://aether.dev/core/pheromones/v1"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://aether.dev/core/pheromones/v1
                        http://aether.dev/schema/pheromone.xsd"
    generated="2026-02-16T10:30:00Z"
    colony-id="colony-abc123">

    <trail id="phem-001" type="PHILOSOPHY">...</trail>
</pheromone-trails>
```

### Multi-Colony Merged Document
```xml
<?xml version="1.0" encoding="UTF-8"?>
<pheromone-trails
    xmlns="http://aether.dev/core/pheromones/v1"
    xmlns:colony-a="http://aether.dev/colony/abc123/v1"
    xmlns:colony-b="http://aether.dev/colony/def456/v1"
    xmlns:template="http://aether.dev/template/nodejs-api/v1"
    xmlns:shared="http://aether.dev/shared/best-practices/v1"
    xsi:schemaLocation="..."
    generated="2026-02-16T10:30:00Z"
    merged="true">

    <!-- Local colony (no prefix) -->
    <trail id="local-001">...</trail>

    <!-- Imported from Colony A -->
    <colony-a:trail id="phem-001">...</colony-a:trail>

    <!-- Imported from Colony B -->
    <colony-b:trail id="phem-002">...</colony-b:trail>

    <!-- From template -->
    <template:pattern id="routing">...</template:pattern>

    <!-- From shared community -->
    <shared:trail id="common-001">...</shared:trail>
</pheromone-trails>
```

---

## XPath with Namespaces

Query specific namespaces:

```bash
# All trails from Colony A
xmllint --xpath "//colony-a:trail" merged.xml

# All patterns from templates
xmllint --xpath "//template:pattern" merged.xml

# All shared wisdom
xmllint --xpath "//shared:trail" merged.xml

# Specific ID across all namespaces
xmllint --xpath "//*[@id='auth-001']" merged.xml
```

---

## Versioning Strategy

Namespaces include version numbers for schema evolution:

```
http://aether.dev/core/pheromones/v1
http://aether.dev/core/pheromones/v2  <-- Breaking changes
http://aether.dev/core/pheromones/v1.1  <-- Backward compatible
```

### Migration Rules
1. **Patch versions** (v1.0 → v1.1): Automatic, backward compatible
2. **Minor versions** (v1 → v1.5): Automatic if no breaking changes
3. **Major versions** (v1 → v2): Manual migration required

---

## Collision Prevention

### ID Uniqueness Within Namespace
Within a single namespace, IDs must be unique:
```xml
<!-- Valid: Same ID, different namespaces -->
<colony-a:trail id="auth-001"/>
<colony-b:trail id="auth-001"/>

<!-- Invalid: Duplicate ID in same namespace -->
<colony-a:trail id="auth-001"/>
<colony-a:trail id="auth-001"/>  <!-- ERROR -->
```

### ID Generation
Recommended format:
```
{category}-{timestamp}-{random}

Examples:
  auth-20260216-abc123
  pattern-1739623456-def456
  redirect-2026-xyz789
```

---

## Best Practices

### 1. Always Declare Default Namespace
```xml
<!-- Good -->
<pheromone-trails xmlns="http://aether.dev/core/pheromones/v1">

<!-- Bad -->
<pheromone-trails>  <!-- No default namespace -->
```

### 2. Use Short, Consistent Prefixes
```xml
<!-- Good -->
xmlns:colony-a="http://aether.dev/colony/abc123/v1"

<!-- Bad -->
xmlns:theColonyFromLastWeek="http://aether.dev/colony/abc123/v1"
```

### 3. Document Namespace Sources
```xml
<pheromone-trails>
    <metadata>
        <namespace-definitions>
            <ns prefix="colony-a" uri="http://aether.dev/colony/abc123/v1">
                Parent colony (payment-service v1)
            </ns>
            <ns prefix="template" uri="http://aether.dev/template/nodejs-api/v1">
                Node.js API template v2.1
            </ns>
        </namespace-definitions>
    </metadata>
</pheromone-trails>
```

---

## Summary

| Namespace Type | URI Pattern | Use For |
|----------------|-------------|---------|
| Core | `http://aether.dev/core/{component}/v1` | System structures |
| Colony | `http://aether.dev/colony/{id}/v1` | Private colony wisdom |
| Template | `http://aether.dev/template/{name}/v1` | Reusable patterns |
| Shared | `http://aether.dev/shared/{category}/v1` | Community wisdom |
| User | `http://aether.dev/user/{username}/v1` | Personal instincts |

---

*This document is part of the Aether XML documentation suite.*
