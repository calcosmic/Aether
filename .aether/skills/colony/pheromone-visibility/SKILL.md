---
name: pheromone-visibility
description: Use when performing any pheromone operation including creating, reinforcing, expiring, or auto-emitting signals during builds and continues
type: colony
domains: [pheromones, visibility, ux]
agent_roles: [builder, watcher, scout]
priority: normal
version: "1.0"
---

# Pheromone Visibility

## Purpose

Pheromone operations must never happen silently. Users need to see exactly what signals are being created, reinforced, expired, or auto-emitted. Every signal change is an opportunity to build user trust in the colony system.

## Mandatory Visibility Rules

Every pheromone operation must produce a user-visible output line. No exceptions.

### Signal Creation

When a new signal is created, display:

```
FOCUS emitted: "area description" [initial strength%]
REDIRECT emitted: "constraint description" [initial strength%]
FEEDBACK emitted: "observation" [initial strength%]
```

### Signal Reinforcement

When an existing signal is reinforced (duplicate content detected), display:

```
FOCUS reinforced: "area description" [new strength%] (x3 reinforcements)
```

### Signal Expiration

When a signal expires due to decay or phase change, display:

```
FOCUS expired: "area description" (was active for 3 phases)
```

### Auto-Emission

During builds and continues, the system auto-emits signals based on decisions and learnings. Each auto-emission must be visible:

```
Auto-emitted FEEDBACK: "prefer composition over inheritance" [70%]
```

## Signal Changes Summary

At the end of every `/ant-continue` or `aether continue` execution, display a "Signal Changes" summary table:

```
━━ S I G N A L   C H A N G E S ━━

| Signal  | Action     | Content              | Strength |
|---------|------------|----------------------|----------|
| FOCUS   | created    | "security"           | 85%      |
| REDIRECT| reinforced | "no inline styles"   | 95%      |
| FEEDBACK| expired    | "prefer dark theme"  | 0%       |

Active signals: 5 | Created: 1 | Reinforced: 1 | Expired: 1
```

If no signal changes occurred, still display the section with "No signal changes this cycle."

## Integration Points

- **During build** (build-wave): Show each auto-emitted signal as it happens, inline with worker output.
- **During continue** (continue-verify, continue-advance): Collect all changes and present the summary table at the end.
- **During seal**: Show which signals are being promoted to eternal memory or hive.

## Why This Matters

Without visibility, pheromone evolution feels like magic -- the user cannot learn what signals do, cannot predict behavior, and cannot trust the system to act on their guidance. Making every operation visible turns the pheromone system from an opaque mechanism into a transparent feedback loop.
