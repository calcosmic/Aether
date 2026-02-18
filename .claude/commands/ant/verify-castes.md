---
name: ant:verify-castes
description: "Verify colony caste assignments and system status"
---

You are the **Queen**. Display the caste assignments and system status.

## Step 1: Show Caste Assignments

Display the colony caste structure:

```
Aether Colony Caste System
═══════════════════════════════════════════

CASTE ASSIGNMENTS
─────────────────
👑 Prime           - Colony coordination and strategic planning
🏺🐜 Archaeologist   - Git history analysis and pattern excavation
🏛️🐜 Architect      - System design and documentation
🔮🐜 Oracle          - Deep research and foresight
🗺️ Route Setter    - Task decomposition and planning
🔨🐜 Builder         - Implementation and coding
👁️🐜 Watcher         - Verification and testing
🔍🐜 Scout           - Research and exploration
🎲🐜 Chaos           - Edge case testing and resilience probing
🧭 Colonizer       - Environment setup and exploration

───────────────────────────────────────────
```

## Step 2: Check System Status

Run using the Bash tool with description "Checking colony version...": `bash .aether/aether-utils.sh version-check-cached 2>/dev/null || echo "Utils available"`

Check LiteLLM proxy status:
```bash
curl -s http://localhost:4000/health 2>/dev/null | grep -q "healthy" && echo "✓ Proxy healthy" || echo "⚠ Proxy not running"
```

## Step 3: Show Current Session Info

```
SESSION INFORMATION
───────────────────
All workers in this session use the same model configuration.
To change models, restart Claude Code with different settings:

export ANTHROPIC_BASE_URL=http://localhost:4000
export ANTHROPIC_AUTH_TOKEN=sk-litellm-local
export ANTHROPIC_MODEL=<model-name>
claude

Available models (via LiteLLM proxy):
  • glm-5        - Complex reasoning, architecture, planning
  • kimi-k2.5    - Fast coding, implementation
  • minimax-2.5  - Validation, research, exploration
```

## Step 4: Summary

```
═══════════════════════════════════════════
System Status
═══════════════════════════════════════════
Utils: ✓ Operational
Proxy: {status from Step 2}
Castes: 10 defined

Note: Model-per-caste routing was attempted but is not
possible with Claude Code's Task tool (no env var support).
See archived config: .aether/archive/model-routing/
Tag: model-routing-v1-archived
```

## Historical Note

A model-per-caste system was designed and implemented but cannot
function due to Claude Code Task tool limitations. The complete
configuration is archived in `.aether/archive/model-routing/`.

To view the archived configuration:
```bash
git show model-routing-v1-archived
```
