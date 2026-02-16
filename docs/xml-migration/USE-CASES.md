# Use Cases

**Real-World Scenarios for Aether XML**

---

## Use Case 1: New Colony Inherits from Multiple Sources

### Scenario
You're starting a new payment service. You want to inherit wisdom from:
- The parent colony (auth-service v1)
- A Node.js API template
- Your personal instincts file

### Without XML
```bash
# Manually copy files
cp ~/chambers/auth-v1/pheromones.json ./temp/
cp ~/templates/nodejs/pheromones.json ./temp/
cp ~/.aether/instincts.json ./temp/

# Write a script to merge them
cat temp/*.json | jq -s '...' > merged.json
# Hope there are no ID collisions
# Hope the contexts are compatible
```

### With XML
```bash
# Create new colony with parent reference
aether init --parent auth-v1 --template nodejs-api

# Merge wisdom from sources
pheromone-merge ~/chambers/auth-v1/pheromones.xml
pheromone-merge ~/templates/nodejs-api/pheromones.xml --namespace template
pheromone-merge ~/.aether/eternal/instincts.xml --namespace me

# Result: Namespaced, no collisions
```

### Result
```xml
<pheromone-trails xmlns="http://aether.dev/core/pheromones/v1"
                  xmlns:auth-v1="http://aether.dev/colony/auth-v1/v1"
                  xmlns:template="http://aether.dev/template/nodejs-api/v1"
                  xmlns:me="http://aether.dev/user/johndoe/v1">

    <!-- My colony's new pheromones -->
    <trail id="payment-001" type="PATTERN">
        <substance>Validate credit card format before processing</substance>
        <context>
            <domain>payment-service</domain>
            <constraint>pci-compliance</constraint>
        </context>
    </trail>

    <!-- Inherited from auth-v1 -->
    <auth-v1:trail id="auth-001" type="PHILOSOPHY">
        <auth-v1:substance>Never trust user input</auth-v1:substance>
    </auth-v1:trail>

    <!-- From Node.js template -->
    <template:pattern id="express-error-handling">
        <template:description>Use centralized error middleware</template:description>
    </template:pattern>

    <!-- My personal instinct -->
    <me:trail id="instinct-001" type="FOCUS">
        <me:substance>Always write tests first</me:substance>
    </me:trail>
</pheromone-trails>
```

---

## Use Case 2: Resolving Conflicting Advice

### Scenario
Colony A (web API) says: "Use JWT tokens for auth"
Colony B (mobile app) says: "Use session cookies for auth"

Your new colony is a payment API. Which do you choose?

### Context Matching
```xml
<!-- Colony A's pheromone -->
<colony-a:trail id="auth-001" type="PATTERN">
    <colony-a:substance>Use JWT tokens</colony-a:substance>
    <colony-a:context>
        <colony-a:domain>web-api</colony-a:domain>
        <colony-a:architecture>stateless</colony-a:architecture>
    </colony-a:context>
    <colony-a:validations>5</colony-a:validations>
</colony-a:trail>

<!-- Colony B's pheromone -->
<colony-b:trail id="auth-001" type="PATTERN">
    <colony-b:substance>Use session cookies</colony-b:substance>
    <colony-b:context>
        <colony-b:domain>mobile-app</colony-b:domain>
        <colony-b:architecture>stateful</colony-b:architecture>
    </colony-b:context>
    <colony-b:validations>4</colony-b:validations>
</colony-b:trail>
```

### Your Colony's Context
```xml
<colony id="payment-api">
    <context>
        <domain>web-api</domain>
        <architecture>stateless</architecture>
        <constraints>
            <constraint>pci-compliance</constraint>
        </constraints>
    </context>
</colony>
```

### The Match
```bash
# Query auth patterns for web-api context
pheromone-query "//trail[context/domain='web-api' and contains(substance, 'auth')]"

# Result: Colony A's JWT advice (95% context match)
#         Colony B's session advice (30% context match)
```

### System Output
```
AUTH PATTERN: Multiple approaches detected

RECOMMENDED (context match: 95%):
  Colony A: "Use JWT tokens for stateless auth"
  Validations: 5 colonies
  Context: web-api, stateless
  Confidence: 0.95

ALTERNATIVE (context match: 30%):
  Colony B: "Use session cookies for native auth"
  Validations: 4 colonies
  Context: mobile-app, stateful
  Confidence: 0.38
  Note: Your web-api context differs from mobile-app

OVERRIDE DETECTED:
  PCI compliance constraint requires:
  "JWT tokens with encrypted claims" (not basic JWT)
```

---

## Use Case 3: Pattern Becomes an Instinct

### Scenario
Multiple colonies discover that "always validate user input" prevents bugs. Time to promote it to QUEEN.md.

### The Journey

**Phase 1: Individual Colony Pheromones**
```xml
<!-- Colony 1 (web API) -->
<trail id="input-001" type="PATTERN">
    <substance>Validate API inputs at boundary</substance>
    <context><domain>web-api</domain></context>
</trail>

<!-- Colony 2 (CLI tool) -->
<trail id="input-002" type="PATTERN">
    <substance>Validate command arguments</substance>
    <context><domain>cli-tool</domain></context>
</trail>

<!-- Colony 3 (mobile app) -->
<trail id="input-003" type="PATTERN">
    <substance>Validate form inputs client-side</substance>
    <context><domain>mobile-app</domain></context>
</trail>
```

**Phase 2: Validation Across Contexts**
```bash
# Query: Which input validation patterns have multiple validations?
pheromone-query "//trail[contains(substance, 'validat') and validation-count >= 3]"

# Result: 3 patterns across different domains
```

**Phase 3: Promotion to QUEEN.md**
```bash
# Promote to philosophy (requires 5 validations from diverse contexts)
queen-evolve --from input-validation --to philosophy --threshold 5
```

**Result in queen-wisdom.xml:**
```xml
<philosophy id="input-validation" status="instinct" threshold="5">
    <belief>Always validate data at trust boundaries</belief>
    <rationale>Invalid data causes crashes, security issues, and data corruption</rationale>
    <evidence>
        <colony-ref id="web-api-1" context-match="90%"/>
        <colony-ref id="cli-tool-1" context-match="85%"/>
        <colony-ref id="mobile-app-1" context-match="80%"/>
        <colony-ref id="microservice-1" context-match="95%"/>
        <colony-ref id="frontend-1" context-match="75%"/>
    </evidence>
</philosophy>
```

**Generated QUEEN.md:**
```markdown
## 📜 Philosophies

### Always validate data at trust boundaries
*Promoted from pattern (5 diverse validations)*

Invalid data causes crashes, security issues, and data corruption.

**Evidence:**
- web-api-1 (90% context match)
- cli-tool-1 (85% context match)
- mobile-app-1 (80% context match)
- microservice-1 (95% context match)
- frontend-1 (75% context match)
```

---

## Use Case 4: Worker Priming with Filtered Pheromones

### Scenario
Spawn a builder ant for a payment service. It should only see relevant pheromones.

### Context Declaration
```xml
<worker-context caste="builder" depth="2">
    <colony-context>
        <domain>web-api</domain>
        <stack>nodejs</stack>
        <stack>postgresql</stack>
        <constraints>
            <constraint>pci-compliance</constraint>
            <constraint>high-security</constraint>
        </constraints>
    </colony-context>
</worker-context>
```

### XInclude-based Priming
```xml
<?xml version="1.0" encoding="UTF-8"?>
<worker-context xmlns:xi="http://www.w3.org/2001/XInclude">
    <!-- Import eternal pheromones filtered by context -->
    <configuration>
        <xi:include href="~/.aether/eternal/pheromones.xml"
                    xpointer="xpointer(//trail[context/domain='web-api'])"/>
    </configuration>

    <!-- Import queen wisdom (always relevant) -->
    <instincts>
        <xi:include href="~/.aether/eternal/queen-wisdom.xml"
                    xpointer="xpointer(//philosophy)"/>
    </instincts>

    <!-- Import stack-specific patterns -->
    <stack-wisdom>
        <xi:include href="~/.aether/eternal/stack-profile/nodejs.xml"/>
        <xi:include href="~/.aether/eternal/stack-profile/postgresql.xml"/>
    </stack-wisdom>
</worker-context>
```

### What the Worker Sees
```xml
<!-- Only web-api relevant pheromones -->
<trail id="api-001" type="PATTERN">
    <substance>Use centralized error handling middleware</substance>
    <context>
        <domain>web-api</domain>
        <stack>nodejs</stack>
    </context>
</trail>

<!-- PCI compliance redirect -->
<trail id="security-001" type="REDIRECT">
    <substance>Never log credit card numbers</substance>
    <context>
        <constraints>
            <constraint>pci-compliance</constraint>
        </constraints>
    </context>
</trail>

<!-- Not included: mobile-app patterns -->
<!-- Not included: cli-tool patterns -->
```

---

## Use Case 5: Colony Lineage and Ancestry Queries

### Scenario
You have a lineage of payment service colonies:
```
payment-v1 (sealed)
  └── payment-v2 (sealed)
        └── payment-v3 (active)
              └── payment-refactor (active)
```

You want to understand what wisdom was inherited vs. newly discovered.

### Registry XML
```xml
<colony-registry current="payment-refactor">
    <colony id="payment-v1" status="sealed" sealed-at="2026-01-15T10:00:00Z">
        <goal>Initial payment API</goal>
        <lineage>
            <parent>null</parent>
            <children>
                <child ref="payment-v2"/>
            </children>
        </lineage>
    </colony>

    <colony id="payment-v2" status="sealed" sealed-at="2026-02-01T14:30:00Z">
        <goal>Add subscription billing</goal>
        <lineage>
            <parent ref="payment-v1"/>
            <children>
                <child ref="payment-v3"/>
            </children>
            <inheritance>
                <pheromone type="PATTERN">idempotency-keys</pheromone>
                <pheromone type="PHILOSOPHY">test-driven-development</pheromone>
            </inheritance>
        </lineage>
    </colony>

    <colony id="payment-v3" status="active">
        <goal>Support international payments</goal>
        <lineage>
            <parent ref="payment-v2"/>
            <children>
                <child ref="payment-refactor"/>
            </children>
        </lineage>
    </colony>

    <colony id="payment-refactor" status="active">
        <goal>Clean architecture refactor</goal>
        <lineage>
            <parent ref="payment-v3"/>
            <inheritance>
                <pheromone type="PATTERN">idempotency-keys</pheromone>
                <pheromone type="PATTERN">webhook-retries</pheromone>
            </inheritance>
        </lineage>
        <overrides>
            <override pattern="monolithic-architecture">
                <reason>Moving to microservices for scalability</reason>
                <replacement>service-oriented-architecture</replacement>
            </override>
        </overrides>
    </colony>
</colony-registry>
```

### Query Examples
```bash
# Get all children of payment-v1
xmllint --xpath "//colony[@id='payment-v1']/lineage//child/@ref" registry.xml
# Result: payment-v2 payment-v3 payment-refactor

# Get inheritance chain for current colony
xmllint --xpath "//colony[@id='payment-refactor']/ancestor::colony/@id" registry.xml
# Result: payment-v3 payment-v2 payment-v1

# Find all colonies that inherited a specific pattern
xmllint --xpath "//colony/lineage/inheritance/pheromone[text()='idempotency-keys']/../../.." registry.xml
```

---

## Use Case 6: Template Repository Sharing

### Scenario
You've created a Node.js API template that multiple projects should use.

### Template Pheromones
```xml
<pheromone-trails xmlns="http://aether.dev/core/pheromones/v1"
                  xmlns:template="http://aether.dev/template/nodejs-api/v1">

    <template:pattern id="project-structure">
        <template:description>Organize by domain, not by layer</template:description>
        <template:example>
src/
  users/
    controller.js
    service.js
    model.js
  orders/
    controller.js
    service.js
    model.js
        </template:example>
        <template:applies-to>nodejs</template:applies-to>
    </template:pattern>

    <template:pattern id="error-handling">
        <template:description>Use custom error classes</template:description>
        <template:code-example>
class ValidationError extends Error {
    constructor(message, fields) {
        super(message);
        this.fields = fields;
    }
}
        </template:code-example>
    </template:pattern>
</pheromone-trails>
```

### Distribution
```bash
# Publish template to shared location
cp template-pheromones.xml ~/.aether/templates/nodejs-api/v1.xml

# Other colonies import it
pheromone-merge ~/.aether/templates/nodejs-api/v1.xml \
                --namespace template-nodejs
```

---

## Use Case 7: Cross-Repository Wisdom

### Scenario
You have multiple repos, each with colonies. You want to share wisdom across repos.

### Repository A: auth-service
```xml
<auth:trail id="jwt-pattern" type="PATTERN">
    <auth:substance>Use short-lived access tokens with refresh tokens</auth:substance>
    <auth:context>
        <auth:domain>web-api</auth:domain>
        <auth:scale>distributed</auth:scale>
    </auth:context>
</auth:trail>
```

### Repository B: payment-service
```bash
# Import from auth-service
pheromone-merge /path/to/auth-service/.aether/eternal/pheromones.xml \
                --namespace auth-service
```

### Merged Result
```xml
<pheromone-trails xmlns="http://aether.dev/core/pheromones/v1"
                  xmlns:auth-service="http://aether.dev/repo/auth-service/v1"
                  xmlns:payment="http://aether.dev/colony/payment-001/v1">

    <!-- Local payment patterns -->
    <payment:trail id="pci-001" type="REDIRECT">
        <payment:substance>Never store raw credit card numbers</payment:substance>
    </payment:trail>

    <!-- Imported from auth-service -->
    <auth-service:trail id="jwt-pattern" type="PATTERN">
        <auth-service:substance>Use short-lived access tokens...</auth-service:substance>
    </auth-service:trail>
</pheromone-trails>
```

---

## Summary

| Use Case | XML Benefit |
|----------|-------------|
| Multi-source inheritance | Namespaces prevent collisions |
| Conflict resolution | Context metadata enables smart matching |
| Pattern promotion | Validation tracking with provenance |
| Worker priming | XInclude for modular composition |
| Lineage queries | Hierarchical structure for ancestry |
| Template sharing | Reusable, namespaced pattern libraries |
| Cross-repo sharing | Repository namespaces isolate sources |

---

*This document is part of the Aether XML documentation suite.*
