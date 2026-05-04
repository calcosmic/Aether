---
schema_version: "1.0"
id: queen-decision-example
kind: example
category: examples
title: Queen Decision Example
description: "Example Queen decision list showing classification tiers, auto-resolve, recovery preview, and circuit breaker."
output_types: [gate-decision-example, queen-decision-example, execution-policy]
agent_roles: [queen, watcher, architect, builder, fixer]
task_types: [gate, decision, example, queen, classify]
task_keywords: [queen, decision, tier, block, advisory, auto-resolve, recovery, example, circuit, breaker, escalation, budget]
workflow_triggers: [continue, seal]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---

# Queen Decision Example

This example shows a realistic queen-state file with multiple gates at different
classification tiers, demonstrating auto-resolve, escalation, and circuit
breaker behavior.

## For Beginners

This is a sample of what the Queen's decision record looks like after a phase
run. It shows how different types of problems get handled: some are fixed
automatically, some require a human to step in, and some are just warnings.
Use this example to understand the Queen's decision-making process and what
each field means.

## Example: Phase 3 Queen State

Below is a representative `queen-state-3.json` file produced after Phase 3
verification. This phase had three workers: a builder, a watcher, and a
scout.

```json
{
  "phase": 3,
  "budget": {
    "initial": 3,
    "remaining": 1,
    "consumed": 2
  },
  "gates": [
    {
      "gate_name": "build-compile",
      "worker_name": "Builder Mason-67",
      "status": "pass",
      "classification": "",
      "findings": [],
      "recovery_preview": {
        "classification": "",
        "first_action": "pass",
        "budget_remaining": 3,
        "would_auto_resolve": false,
        "would_escalate": false
      }
    },
    {
      "gate_name": "test-suite",
      "worker_name": "Builder Mason-67",
      "status": "fail",
      "classification": "soft_block",
      "findings": [
        "TestQueenDecide/soft_block_with_budget: expected escalate, got auto-resolve"
      ],
      "recovery_preview": {
        "classification": "soft_block",
        "first_action": "auto-resolve",
        "budget_remaining": 3,
        "would_auto_resolve": true,
        "would_escalate": false
      },
      "resolution": {
        "action": "auto-resolve",
        "fixer_dispatched": "Fixer Tinker-41",
        "fix_result": "pass",
        "budget_consumed": 1
      }
    },
    {
      "gate_name": "race-detection",
      "worker_name": "Watcher Sentinel-23",
      "status": "fail",
      "classification": "soft_block",
      "findings": [
        "DATA RACE in cmd/queen_decision.go:142"
      ],
      "recovery_preview": {
        "classification": "soft_block",
        "first_action": "auto-resolve",
        "budget_remaining": 2,
        "would_auto_resolve": true,
        "would_escalate": false
      },
      "resolution": {
        "action": "auto-resolve",
        "fixer_dispatched": "Fixer Tinker-41",
        "fix_result": "pass",
        "budget_consumed": 1
      }
    },
    {
      "gate_name": "source-parity",
      "worker_name": "Watcher Sentinel-23",
      "status": "pass",
      "classification": "",
      "findings": [],
      "recovery_preview": {
        "classification": "",
        "first_action": "pass",
        "budget_remaining": 1,
        "would_auto_resolve": false,
        "would_escalate": false
      }
    },
    {
      "gate_name": "security-scan",
      "worker_name": "Scout Pathfinder-91",
      "status": "fail",
      "classification": "hard_block",
      "findings": [
        "Exposed API key in cmd/config.go:23"
      ],
      "recovery_preview": {
        "classification": "hard_block",
        "first_action": "escalate",
        "budget_remaining": 1,
        "would_auto_resolve": false,
        "would_escalate": true
      }
    }
  ],
  "breaker_tripped_workers": [],
  "escalations": [
    {
      "timestamp": "2026-05-04T15:42:17Z",
      "gate_name": "security-scan",
      "worker_name": "Scout Pathfinder-91",
      "breaker_tripped_workers": [],
      "escalation_action": "escalate",
      "rationale": "Hard block: exposed secret detected. Auto-resolve not permitted for security findings."
    }
  ],
  "recommendation": "escalate",
  "queen_decision_timestamp": "2026-05-04T15:42:17Z"
}
```

## Walkthrough

### Gate 1: build-compile (pass)

The build compiled successfully. No classification was needed. The recovery
preview shows `first_action: "pass"` because there is nothing to recover from.
Budget is at the initial value of 3.

### Gate 2: test-suite (fail, soft_block, auto-resolved)

A test failed in the Queen decision logic. The gate classified this as a
`soft_block` because it is a test assertion error, not a critical failure.

- **Recovery preview** shows `would_auto_resolve: true` because the budget
  (3 at this point) is greater than zero.
- **Resolution** shows Fixer Tinker-41 was dispatched, fixed the issue, and
  the re-run passed. Budget consumed: 1 (remaining: 2).

### Gate 3: race-detection (fail, soft_block, auto-resolved)

A data race was found in the Queen decision code. This is a `soft_block` --
serious but fixable.

- **Recovery preview** shows `would_auto_resolve: true` with budget remaining
  at 2.
- **Resolution** shows the same Fixer handled it. Budget consumed: 1
  (remaining: 1).

### Gate 4: source-parity (pass)

Source-to-mirror parity check passed. No action needed.

### Gate 5: security-scan (fail, hard_block, escalated)

An exposed API key was found. This is a `hard_block` -- the highest severity.

- **Recovery preview** shows `would_auto_resolve: false` and
  `would_escalate: true` regardless of budget.
- **No resolution** field because auto-resolve is not permitted.
- **Escalation** is logged with rationale explaining why auto-resolve was
  bypassed.

### Final Recommendation

The Queen's final recommendation is `escalate` because of the hard_block.
Even though two soft blocks were successfully auto-resolved, the hard block
takes precedence. The phase cannot advance until the user addresses the
security finding.

## Circuit Breaker Scenario

If Builder Mason-67 had triggered a third soft block after the first two
were auto-resolved, the breaker would have tripped:

```json
{
  "breaker_tripped_workers": ["Builder Mason-67"],
  "escalations": [
    {
      "timestamp": "2026-05-04T15:45:02Z",
      "gate_name": "lint-check",
      "worker_name": "Builder Mason-67",
      "breaker_tripped_workers": ["Builder Mason-67"],
      "escalation_action": "escalate",
      "rationale": "Circuit breaker tripped: Builder Mason-67 has triggered repeated soft blocks. Escalating despite budget remaining."
    }
  ]
}
```

In this scenario, the worker is in the `breaker_tripped_workers` array, so
even if budget remained, the Queen would escalate rather than auto-resolve.

## Key Takeaways

- Recovery previews are generated for **all** gates, including passing ones
- `soft_block` gates auto-resolve when budget remains
- `hard_block` gates always escalate, regardless of budget
- Circuit breaker tracks workers by name and prevents infinite retry loops
- The final recommendation reflects the most severe unresolved gate
- Budget is consumed by auto-resolve attempts, not by passing gates
