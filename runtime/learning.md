# Colony Learning Discipline

## Overview

The colony learns from every phase. Patterns observed during builds become **instincts** - atomic learned behaviors with confidence scoring that improve future work.

## The Instinct Model

An instinct is a small learned behavior:

```yaml
id: prefer-composition
trigger: "when designing component architecture"
confidence: 0.7
domain: "architecture"
source: "phase-3-observation"
evidence:
  - "Composition pattern succeeded in Phase 3"
  - "User approved component structure"
action: "Use composition over inheritance for component reuse"
```

**Properties:**
- **Atomic** — one trigger, one action
- **Confidence-weighted** — 0.3 = tentative, 0.9 = near certain
- **Domain-tagged** — architecture, testing, code-style, debugging, workflow
- **Evidence-backed** — tracks what observations created it

## Confidence Scoring

| Score | Meaning | Behavior |
|-------|---------|----------|
| 0.3 | Tentative | Suggested but not enforced |
| 0.5 | Moderate | Applied when relevant |
| 0.7 | Strong | Auto-applied in matching contexts |
| 0.9 | Near-certain | Core colony behavior |

**Confidence increases when:**
- Pattern is repeatedly observed across phases
- Pattern leads to successful outcomes
- No corrections or rework needed

**Confidence decreases when:**
- Pattern leads to errors or rework
- User provides negative feedback
- Contradicting evidence appears

## Pattern Detection

During each phase, observe for:

### 1. Success Patterns
What worked well:
- Approaches that completed without issues
- Code structures that passed all tests
- Workflows that were efficient

### 2. Error Resolutions
What was learned from debugging:
- Root causes discovered
- Fixes that worked
- Architectural insights

### 3. User Corrections
What the user redirected:
- Feedback via `/ant:feedback`
- Redirects via `/ant:redirect`
- Explicit corrections

### 4. Tool Preferences
What tools/patterns were effective:
- Testing approaches that caught bugs
- File organization that worked
- Command sequences that succeeded

## Instinct Structure

Store in `memory.instincts`:

```json
{
  "instincts": [
    {
      "id": "instinct_<timestamp>",
      "trigger": "when X",
      "action": "do Y",
      "confidence": 0.5,
      "domain": "testing",
      "source": "phase-2",
      "evidence": ["observation 1", "observation 2"],
      "created_at": "<ISO-8601>",
      "last_applied": null,
      "applications": 0,
      "successes": 0
    }
  ]
}
```

## Instinct Lifecycle

```
Observation
    │
    ▼
┌─────────────────────────────┐
│     Pattern Detected        │
│   (success/error/feedback)  │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│     Instinct Created        │
│   (confidence: 0.3-0.5)     │
└─────────────────────────────┘
    │
    │ Applied in future phases
    ▼
┌─────────────────────────────┐
│   Confidence Adjusted       │
│   (+0.1 success, -0.1 fail) │
└─────────────────────────────┘
    │
    │ Reaches 0.9
    ▼
┌─────────────────────────────┐
│     Core Behavior           │
│   (always applied)          │
└─────────────────────────────┘
```

## Extracting Instincts

After each phase, the Prime Worker should identify:

1. **What patterns led to success?**
   - Approach taken
   - Tools used effectively
   - Code structures that worked

2. **What was learned from errors?**
   - Root causes found
   - Fixes that worked
   - What to avoid

3. **What feedback was received?**
   - User corrections
   - Focus areas emphasized
   - Patterns to avoid

Format for extraction:

```json
"patterns_observed": [
  {
    "type": "success",
    "trigger": "when implementing API endpoints",
    "action": "use repository pattern with dependency injection",
    "evidence": "All endpoint tests passed first try"
  },
  {
    "type": "error_resolution",
    "trigger": "when debugging async code",
    "action": "check for missing await statements first",
    "evidence": "Root cause was missing await in 3 cases"
  },
  {
    "type": "user_feedback",
    "trigger": "when structuring components",
    "action": "prefer smaller, focused components",
    "evidence": "User feedback: 'keep components small'"
  }
]
```

## Applying Instincts

When starting a new phase, workers should:

1. **Check relevant instincts** for the task domain
2. **Apply high-confidence instincts** (≥0.7) automatically
3. **Consider moderate instincts** (0.5-0.7) as suggestions
4. **Log applications** for confidence tracking

Format in worker prompt:

```
--- COLONY INSTINCTS ---
Relevant learned patterns for this phase:

[0.9] testing: Always run tests before claiming completion
[0.7] architecture: Use composition over inheritance
[0.5] code-style: Prefer functional patterns (tentative)
```

## Reporting Learned Patterns

Prime Worker output should include:

```json
"learning": {
  "patterns_observed": [
    {
      "type": "success|error_resolution|user_feedback",
      "trigger": "when X",
      "action": "do Y",
      "evidence": "observation"
    }
  ],
  "instincts_applied": ["instinct_id_1", "instinct_id_2"],
  "instinct_outcomes": [
    {"id": "instinct_id_1", "success": true},
    {"id": "instinct_id_2", "success": false}
  ]
}
```

## Colony Memory Evolution

Over time, instincts with high confidence become colony-wide behaviors:

| Threshold | Evolution |
|-----------|-----------|
| 0.9+ with 5+ applications | Core behavior - always applied |
| 3+ related instincts | Cluster into skill document |
| Domain-specific cluster | Specialist worker enhancement |

## Integration Points

### /ant:build
- Workers receive relevant instincts in prompt
- Workers report patterns observed
- Workers log instinct applications

### /ant:continue
- Extract patterns from build results
- Create/update instincts
- Adjust confidence based on outcomes

### /ant:status
- Show learned instincts with confidence
- Show application statistics
- Show recent pattern discoveries

### /ant:feedback
- Creates user_feedback instinct
- High initial confidence (0.7)
- Immediate application to current work

## Privacy

- Instincts stay local in `.aether/data/COLONY_STATE.json`
- Only patterns extracted, not raw code
- Export capability for sharing (future)
