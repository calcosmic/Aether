<!-- Generated from .aether/commands/oracle.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-oracle
description: "🔮 Run the autonomous Oracle loop through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether oracle "$ARGUMENTS"` directly when a topic is provided.
- For inspection or control, prefer `aether oracle status` and `aether oracle stop`.
- Do not describe legacy loop control files or shell-managed orchestration from this command spec.
- Report the CLI result directly.

## Post-Completion Persistence Suggestions

After the oracle loop completes and you have reported the research results, check whether the findings are worth preserving for the colony:

**Only suggest persistence when:**
- The oracle completed successfully (status is "complete" or "max_iterations_reached")
- Do NOT suggest persistence if the oracle was blocked or stopped early

**How to identify high-value findings:**
- Findings with high confidence that apply to the colony's current goal
- Actionable recommendations that would benefit future workers
- Domain-specific insights that are not obvious from the codebase alone
- Architecture or design decisions documented in the research

**Suggestion format:**
Present findings worth preserving as a tick-to-approve list:

```
🔮 Research findings worth preserving:

1. [x] {finding title}
   → Suggest: aether pheromone-write --type "FOCUS" --content '{"text":"{finding summary}"}' --priority "normal" --source "oracle" --reason "High-value research finding"

2. [ ] {finding title}
   → Suggest: aether hive-store --text "{generalized finding}" --domain "{detected domain}" --source-repo "{repo name}"
```

For each approved finding:
- If it is colony-specific guidance: use `aether pheromone-write --type "FOCUS" --content '{"text":"{finding}"}' --priority "normal" --source "oracle" --reason "High-value research finding"`
- If it is generalizable cross-colony wisdom: use `aether hive-store --text "{generalized finding}" --domain "{domain}" --source-repo "{repo name}"`
- Let the user approve each suggestion before persisting

**Do NOT suggest persistence for:** low-confidence findings, obvious observations, or findings already captured as pheromones.
