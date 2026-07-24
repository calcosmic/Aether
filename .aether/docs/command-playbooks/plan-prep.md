---
name: ant:plan-prep
description: "Plan preparation: status, depth selection, clarification gate"
---

### Step 1: Status Check

Run using the Bash tool with description "Loading colony status...":
```bash
AETHER_OUTPUT_MODE=visual aether status
```

Verify the colony is in a state that allows planning (not sealed, not already executing).

If the status indicates issues:
- Sealed colony: "This colony is sealed. Start a new colony with `/ant-init`." Stop.
- No colony: "No colony initialized. Run `/ant-init` first." Stop.

Display brief context:
```
📋 Planning — Phase {current_phase}/{total_phases}
Goal: "{colony_goal}"
```

### Step 2: Planning Depth Selection (Phase Granularity)

Prompt for planning depth:
1. **Fast** — sprint granularity, 1-3 phases. Target confidence: 80, max iterations: 4.
2. **Balanced** — milestone granularity, 4-7 phases. Target confidence: 90, max iterations: 6. **Recommended default.**
3. **Deep** — quarter granularity, 8-12 phases. Target confidence: 95, max iterations: 8.
4. **Exhaustive** — major granularity, 13-20 phases. Target confidence: 99, max iterations: 12.

If `$ARGUMENTS` already contains `--depth`, use it and state the selected depth. Otherwise, ask the user.

### Step 3: Planning Depth Selection (Task Decomposition)

After selecting phase granularity, choose task decomposition depth:
1. **Light** — coarse tasks, 1-3 per plan with objective-level descriptions.
2. **Standard** — normal task breakdown. **Recommended default.**
3. **Deep** — granular subtasks including edge cases, error handling, and test coverage as separate tasks.

This controls task detail within each plan, not how many phases are generated. If `$ARGUMENTS` already contains `--planning-depth`, use it.

### Step 4: Unresolved Clarifications Gate

Check for unresolved clarifications before planning:

```bash
aether pending-decision-list | jq '.unresolved' 2>/dev/null || echo "0"
```

If unresolved clarifications exist:
```
⚠ Unresolved clarifications detected ({count} pending).

Consider running `/ant-discuss` to resolve these before planning.
```

Route to `/ant-discuss` if the user wants to resolve them first. Proceed with implicit assumptions only if the user explicitly chooses to continue.

### Step 5: Request Plan Manifest

Immediately before requesting the manifest:
```
Asking the runtime for the plan manifest...
```

Run:
```bash
aether host plan --depth <choice> --planning-depth <choice2> --target <target> --max-iterations <iterations> $ARGUMENTS
```

Save the full JSON envelope to a temporary manifest file outside `.aether/data/`.

Parse `result.plan_manifest` or `result.planning_manifest`; do not parse visual output.

Before rendering spawn ceremonies, inspect `result.orchestrator_boundary_guidance`. If it is active or `next` is `aether discuss`, stop, show summary, route to `/ant-discuss`, and request a fresh manifest after resolution.
