# Aether XML Vision

**The Wisdom Sharing System for AI Agent Colonies**

---

## In Plain English

Imagine you're building a software project with a team of AI agents. Each group of agents (called a **colony**) works on a specific part — one colony builds the login system, another handles payments, another manages the database.

As each colony works, it learns things:
- "Always validate user input at the API boundary"
- "Never use sync file operations in async code"
- "This pattern caused bugs — avoid it"

**The Problem:** When a new colony starts, it should benefit from what previous colonies learned. But how do you share those lessons without causing confusion?

**The Answer:** Aether's XML-based wisdom sharing system.

---

## Why XML Matters for Multi-Colony Systems

### The Collision Problem

Without namespacing, you get conflicts:

```json
// Colony A's lesson
{ "id": "phem_001", "advice": "Use JWT tokens" }

// Colony B's lesson
{ "id": "phem_001", "advice": "Avoid JWT tokens" }

// Which one is right? Depends on context!
```

XML namespaces solve this elegantly:
```xml
<colony-a:trail id="phem_001">Use JWT tokens</colony-a:trail>
<colony-b:trail id="phem_001">Use session cookies</colony-b:trail>
```

Both can coexist. Both can be queried. Neither overwrites the other.

---

## The Hybrid Architecture

Aether uses **both JSON and XML** — each for what it does best:

### JSON Zone (Operational Data)
Fast, local, high-churn information:
- `COLONY_STATE.json` — what this colony is doing right now
- `activity.log` — events as they happen
- `flags.json` — blockers and issues

### XML Zone (Eternal Memory)
Shared, validated, cross-colony wisdom:
- `pheromones.xml` — lessons learned with provenance
- `queen-wisdom.xml` — validated instincts across colonies
- `registry.xml` — colony lineage and relationships

---

## Context-Awareness: The Key Innovation

Not all wisdom applies everywhere. A web API colony needs different guidance than a mobile app colony.

### Context Metadata
Every piece of wisdom carries context:
```xml
<trail id="auth-pattern" type="PATTERN">
  <substance>Use JWT tokens for stateless auth</substance>
  <context>
    <domain>web-api</domain>
    <constraint>stateless</constraint>
    <stack>nodejs</stack>
  </context>
  <validations>
    <colony-ref id="api-gateway" confidence="0.95"/>
    <colony-ref id="microservice-a" confidence="0.90"/>
  </validations>
</trail>
```

### Smart Matching
When a new colony spawns workers, the system:
1. Matches context ("Is this a web API?")
2. Filters relevant pheromones
3. Ignores wisdom from incompatible contexts
4. Surfaces conflicts: "Colony A says X, Colony B says Y — your context is closer to B"

---

## The Role of QUEEN.md

QUEEN.md is the **eternal wisdom** — patterns that have proven themselves across multiple colonies.

### Current State (Mixed Format)
```markdown
## Philosophies
- **colony-a**: Test-driven development ensures quality

<!-- METADATA
{
  "promotion_thresholds": {
    "philosophy": 5
  }
}
-->
```

### Future State (XML Source)
```xml
<queen-wisdom version="1.0">
  <philosophy id="tdd-quality" threshold="5">
    <belief>Test-driven development ensures quality</belief>
    <evidence>
      <colony-ref id="colony-a" validations="5"/>
      <colony-ref id="colony-b" validations="3"/>
    </evidence>
  </philosophy>
</queen-wisdom>
```

### How It Works
1. Colonies leave pheromones as they work
2. When a pattern validates across multiple colonies, it gets promoted
3. QUEEN.md tracks the promotion with evidence
4. Future colonies inherit proven wisdom

---

## Multi-Colony Sharing in Practice

### Scenario: Starting a New Payment Colony

**Without XML:**
- Manually read old colony notes
- Copy-paste relevant patterns
- Hope you don't miss conflicting advice
- No way to track where wisdom came from

**With XML:**
```bash
# Merge wisdom from parent colony and templates
aether pheromone-merge parent-colony/pheromones.xml
aether pheromone-merge templates/payment-service.xml

# Query: What auth patterns apply to payment services?
xmllint --xpath "//trail[context/domain='payment-service']" pheromones.xml

# Result: JWT tokens (from api-gateway colony),
#         Session cookies (from mobile-app colony) —
#         with context showing which fits your architecture
```

---

## Benefits Summary

| Challenge | JSON Approach | XML Approach |
|-----------|--------------|--------------|
| ID collisions | Manual prefixing (fragile) | **Namespaces (automatic)** |
| Provenance | Convention-based | **Built-in tracking** |
| Context filtering | Complex jq queries | **XPath selection** |
| Validation | Optional, ad-hoc | **XSD strict validation** |
| Cross-colony merge | Manual, error-prone | **Structured, safe** |

---

## The Vision

Aether becomes a **living knowledge base**:

- Every colony contributes what it learns
- Wisdom compounds across projects
- Context ensures relevance
- Namespaces prevent conflicts
- Lineage tracks evolution

New colonies start **smarter** because they inherit the validated wisdom of all previous colonies.

---

## Next Steps

1. **Review this vision** — Does it match your mental model?
2. **Read the technical docs** — See CONTEXT-AWARE-SHARING.md and XSD-SCHEMAS.md
3. **Explore use cases** — Real scenarios in USE-CASES.md
4. **Begin planning** — What would Phase 1 look like?

---

*This document is part of the Aether XML documentation suite.*
