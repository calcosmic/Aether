---
schema_version: "1.0"
id: queen-execution-policy-contract
kind: contract
category: contracts
title: Queen Execution Policy Contract
description: "Classification tiers, auto-resolve eligibility, circuit breaker, and recovery preview for Queen gate decisions."
output_types: [gate-decision, queen-decision, execution-policy, gate-decision-example]
agent_roles: [queen, watcher, architect, builder, fixer]
task_types: [gate, decision, execution, policy, advance, continue]
task_keywords: [queen, decision, gate, tier, block, advisory, resolve, escalate, circuit, breaker, recovery, budget, classification]
workflow_triggers: [continue, seal]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---

# Queen Execution Policy Contract

This contract defines how the Queen classifies gate results, decides whether to
auto-resolve or escalate, and tracks circuit breaker state across a phase.

## For Beginners

The Queen is the orchestrator that decides what happens after each worker
finishes. When a gate (verification check) fails, the Queen must decide:
automatically fix it, send a Fixer agent, or stop and ask the user. This
contract describes those rules so every agent understands the decision process.

## Classification Tiers

Every gate result is classified into one of four tiers. The tier determines
the response strategy.

| Tier | Meaning | Response Rule |
|------|---------|---------------|
| `hard_block` | Critical failure | Always escalate. Never auto-resolve. |
| `soft_block` | Recoverable failure | Auto-resolve if budget remaining > 0. Escalate if budget exhausted. |
| `advisory` | Warning, non-blocking | Log and continue. Escalate only on failure. |
| `""` (unclassified) | No classification | Treated as advisory. |

### Tier Assignment

Classification is assigned by the gate that produces the result. The Queen does
not reclassify; it respects the tier the gate assigned.

### Hard Block

Hard blocks represent safety-critical or data-loss scenarios. Examples:

- State file corruption detected
- Protected path overwrite attempted
- Security gate failed with exposed secrets
- Binary integrity mismatch

The Queen **never** attempts auto-resolve for hard blocks. It immediately
escalates with full context including the gate name, worker, and breaker state.

### Soft Block

Soft blocks represent recoverable issues where a Fixer agent can likely solve
the problem. Examples:

- Test failure in a single test file
- Lint error in modified code
- Minor schema migration needed

Auto-resolve is permitted only when the remaining budget (configured per phase)
is greater than zero. If the budget is exhausted, the soft block is treated as
an escalation.

### Advisory

Advisory findings are informational. They are logged, included in the phase
report, and injected into subsequent worker context. They only trigger
escalation if a subsequent action fails that the advisory warned about.

## Queen Recommendations

The `queenDecide` function returns a `QueenRecommendation` with one of these
values:

| Recommendation | When Used |
|---------------|-----------|
| `pass` | All gates passed or only advisory findings |
| `auto-resolve` | Soft block with budget remaining |
| `dispatch-fixer` | Soft block where targeted fix is possible |
| `escalate` | Hard block, exhausted budget, or circuit breaker tripped |

`queenDecide` is a **pure function**: no goroutines, no daemons, no side
effects. It takes gate results and breaker state as input, returns a
recommendation as output. State persistence happens outside the function.

## Recovery Preview

A recovery preview is generated for **every** gate, not just failed ones. This
allows the continue system to display what *would* happen if the gate had
failed, which aids in phase review and debugging.

### Recovery Preview Fields

| Field | Description |
|-------|-------------|
| `classification` | The tier assigned to this gate |
| `first_action` | What the Queen would do first (pass, auto-resolve, escalate) |
| `budget_remaining` | How many auto-resolve attempts remain |
| `would_auto_resolve` | Boolean: would the Queen attempt auto-resolve? |
| `would_escalate` | Boolean: would the Queen escalate? |

Recovery previews are stored in the queen-state file and displayed during
continue review.

## Circuit Breaker

The circuit breaker prevents infinite retry loops when a worker consistently
fails gates.

### How It Works

1. The breaker tracks a `breaker_tripped_workers` array in queen state.
2. When a worker triggers a soft block, its name is added to the array.
3. If the same worker triggers another soft block while already in the array,
   the Queen escalates instead of auto-resolving, regardless of budget.
4. The breaker resets at the start of each phase.

### Escalation Entry

Each escalation is recorded with:

| Field | Description |
|-------|-------------|
| `timestamp` | When the escalation occurred |
| `gate_name` | Which gate triggered escalation |
| `worker_name` | Which worker was running |
| `breaker_tripped_workers` | Current breaker state at escalation time |
| `escalation_action` | What was done (escalate, dispatch-fixer) |
| `rationale` | Human-readable explanation |

## State Persistence

Queen decisions are persisted to `.aether/data/queen-state-{phase}.json` at the
end of each gate evaluation cycle. The file contains:

- All gate results for the current phase
- Classification assignments
- Recovery previews
- Circuit breaker state
- Escalation log
- Budget tracking (initial, remaining)

This file is read by the continue system to display phase review and by the
resume system to restore Queen context after session breaks.

## Contract Obligations

**Agents MUST:**
- Respect the classification tier assigned by gates
- Never attempt to bypass hard_block escalation
- Report budget consumption accurately
- Include full context in escalation entries

**Agents MUST NOT:**
- Reclassify gate results (the gate owns classification)
- Auto-resolve hard blocks under any condition
- Reset the circuit breaker mid-phase
- Modify queen-state files directly (use the runtime API)

**Builders** should understand that soft blocks may trigger automatic Fixer
dispatch. **Watchers** should classify their findings accurately since
misclassification directly affects Queen behavior. **Architects** should
consider tier implications when designing new gates.
