# Context-Aware Wisdom Sharing

**How Aether Ensures Colonies Get Relevant Guidance**

---

## The Problem: Conflicting Advice

Different colonies need different guidance:

| Colony Type | Auth Pattern | Why |
|-------------|--------------|-----|
| Web API | JWT tokens | Stateless, scalable |
| Mobile App | Session cookies | Native auth flows |
| CLI Tool | API keys | Simple, scriptable |
| Microservice | mTLS certificates | Service-to-service |

If every colony blindly inherits "use JWT tokens," you'd have the wrong solution half the time.

---

## The Solution: Context Metadata

Every pheromone carries context describing where it applies:

```xml
<trail id="auth-jwt" type="PATTERN">
  <substance>Use JWT tokens for authentication</substance>
  <context>
    <domain>web-api</domain>
    <architecture>stateless</architecture>
    <scale>distributed</scale>
    <stack>nodejs</stack>
    <stack>python</stack>
  </context>
  <confidence>0.92</confidence>
</trail>
```

---

## Context Dimensions

### 1. Domain
What kind of system is this?
- `web-api` — HTTP APIs
- `mobile-app` — iOS/Android applications
- `cli-tool` — Command-line utilities
- `microservice` — Service mesh components
- `frontend` — Browser-based UI
- `data-pipeline` — ETL/streaming

### 2. Architecture
How is it structured?
- `stateless` — No server-side session
- `stateful` — Session-based
- `event-driven` — Message queues
- `serverless` — Function-as-a-service
- `monolithic` — Single deployable

### 3. Scale
What's the operational scale?
- `single-instance` — One server
- `distributed` — Multiple instances
- `global` — Multi-region
- `high-traffic` — 10k+ RPS

### 4. Stack
What technologies?
- `nodejs`, `python`, `rust`, `go`
- `postgresql`, `mongodb`, `redis`
- `docker`, `kubernetes`

### 5. Constraints
Special requirements?
- `high-security` — Financial/health data
- `low-latency` — Real-time systems
- `offline-capable` — Spotty connectivity
- `compliance-gdpr` — Data residency

---

## The Matching Algorithm

When a colony spawns, it declares its context:

```xml
<colony id="payment-api" status="active">
  <context>
    <domain>web-api</domain>
    <architecture>stateless</architecture>
    <scale>distributed</scale>
    <stack>nodejs</stack>
    <constraints>
      <constraint>high-security</constraint>
      <constraint>compliance-pci</constraint>
    </constraints>
  </context>
</colony>
```

### Matching Rules

1. **Exact match** — All dimensions align → High confidence
2. **Partial match** — Some dimensions align → Medium confidence, flag for review
3. **Conflict** — Same topic, incompatible contexts → Surface both, let colony choose
4. **No match** — Wisdom doesn't apply → Filter out

### Example Query

```bash
# Find auth patterns for a stateless web API
xmllint --xpath "//trail[context/domain='web-api'
                         and context/architecture='stateless']
                         /substance/text()" pheromones.xml

# Result: "Use JWT tokens for authentication"
```

---

## Handling Conflicts

### Scenario: Two Valid Approaches

```xml
<!-- From Colony A (web API) -->
<colony-a:trail id="auth-001">
  <substance>Use JWT tokens</substance>
  <context><domain>web-api</domain></context>
  <validations>5</validations>
</colony-a:trail>

<!-- From Colony B (mobile app) -->
<colony-b:trail id="auth-001">
  <substance>Use session cookies</substance>
  <context><domain>mobile-app</domain></context>
  <validations>4</validations>
</colony-b:trail>
```

**New Colony: Payment API (web API domain)**

The system surfaces:
```
AUTH PATTERN: Multiple approaches detected

RECOMMENDED (context match: 95%):
  Colony A: "Use JWT tokens"
  Validations: 5 colonies

ALTERNATIVE (context match: 30%):
  Colony B: "Use session cookies"
  Validations: 4 colonies
  Note: Mobile context differs from your web-api domain
```

---

## Colony-Specific Overrides

Sometimes a colony needs to break the rules. That's fine — but it's recorded:

```xml
<colony id="payment-api">
  <context>...</context>
  <overrides>
    <override pattern="auth-jwt">
      <reason>PCI compliance requires session-based auth</reason>
      <replacement>session-cookies-with-encryption</replacement>
    </override>
  </overrides>
</colony>
```

This override:
- Is local to this colony (doesn't affect others)
- Is recorded for future reference
- Contributes to QUEEN.md evolution ("JWT doesn't apply when PCI compliance required")

---

## Promotion to QUEEN.md

Patterns become instincts when validated across contexts:

```xml
<pattern id="input-validation" status="candidate">
  <substance>Always validate user input at API boundary</substance>
  <validations>
    <colony-ref id="web-api-1" context-match="95%"/>
    <colony-ref id="mobile-app-1" context-match="80%"/>
    <colony-ref id="cli-tool-1" context-match="70%"/>
  </validations>
  <!-- 3 validations, but diverse contexts -->
</pattern>

<philosophy id="input-validation" status="instinct">
  <substance>Always validate user input at trust boundary</substance>
  <evidence>
    <!-- 5+ validations across diverse contexts -->
    <colony-ref id="web-api-1"/>
    <colony-ref id="mobile-app-1"/>
    <colony-ref id="cli-tool-1"/>
    <colony-ref id="microservice-1"/>
    <colony-ref id="frontend-1"/>
  </evidence>
  <!-- Promoted to philosophy: universally applicable -->
</philosophy>
```

---

## Confidence Scoring

### Calculation

```
confidence = base_confidence × context_similarity × validation_weight

Where:
- base_confidence: How sure was the originating colony? (0.0-1.0)
- context_similarity: How similar are the contexts? (0.0-1.0)
- validation_weight: How many colonies validated this? (1.0 + 0.1×count)
```

### Example

| Pheromone | Base | Context | Validations | Final Confidence |
|-----------|------|---------|-------------|------------------|
| JWT tokens | 0.9 | 95% | 5 | 0.9 × 0.95 × 1.5 = **1.28** (capped at 1.0) |
| Session cookies | 0.9 | 30% | 4 | 0.9 × 0.3 × 1.4 = **0.38** |

The JWT recommendation wins for a web API colony.

---

## Inheritance and Lineage

Child colonies inherit parent context plus modifications:

```xml
<colony id="payment-v2" parent="payment-v1">
  <context>
    <!-- Inherits from parent -->
    <domain>web-api</domain>
    <architecture>stateless</architecture>

    <!-- Adds new constraint -->
    <constraints>
      <constraint>compliance-pci-dss-v4</constraint>
    </constraints>
  </context>
</colony>
```

Inherited pheromones are filtered through the child's context.

---

## Best Practices

### For Colony Creators
1. **Be specific about context** — Generic context = generic advice
2. **Record overrides** — Don't silently ignore wisdom; document why
3. **Validate across contexts** — A pattern that works in 3 domains is stronger than one validated 10× in the same domain

### For System Designers
1. **Default to exclusion** — When in doubt, filter out
2. **Surface conflicts** — Don't silently choose; let the colony decide
3. **Learn from overrides** — Patterns that get overridden frequently need refinement

---

## Summary

Context-aware sharing means:
- **Relevance** — Colonies get advice that fits their situation
- **Safety** — Conflicting advice is surfaced, not hidden
- **Learning** — The system learns which patterns are universal vs. situational
- **Flexibility** — Colonies can override when needed, but it's recorded

---

*This document is part of the Aether XML documentation suite.*
