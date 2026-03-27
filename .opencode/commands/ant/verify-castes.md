---
name: ant:verify-castes
description: "Verify colony caste assignments and system status"
---

You are the **Queen**. Display the caste assignments and system status.

## Step 1: Show Caste Assignments

Display the colony caste structure with model slot assignments:

```
Aether Colony Caste System
═══════════════════════════════════════════

CASTE ASSIGNMENTS
─────────────────
Slot: OPUS (reasoning/analysis)
  👑 Queen              - Colony coordination and strategic planning
  🏺 Archaeologist      - Git history analysis and pattern excavation
  🗺️ Route Setter      - Task decomposition and planning
  📜 Sage               - Wisdom synthesis and cross-colony knowledge
  🔍 Tracker            - Bug investigation and root cause analysis
  📋 Auditor            - Quality gate and code review
  🔒 Gatekeeper         - Security scanning and antipattern detection
  📏 Measurer           - Performance analysis and metrics

Slot: SONNET (execution/implementation)
  🔨 Builder            - Implementation and coding
  👁️ Watcher            - Verification and testing
  🔎 Scout              - Research and exploration
  🎲 Chaos              - Edge case testing and resilience probing
  🔬 Probe              - Test coverage analysis
  🧵 Weaver             - Refactoring specialist
  🌐 Ambassador         - External integrations
  🏠 Nest               - Directory structure mapping
  📚 Disciplines        - Convention documentation
  🦠 Pathogens          - Tech debt identification
  📦 Provisions         - Dependency mapping

Slot: INHERIT (uses parent's model)
  📝 Chronicler        - Documentation
  ♿ Includer           - Accessibility audits
  🗄️ Keeper            - Knowledge preservation

───────────────────────────────────────────
```

The model slot assignments come from agent frontmatter (`model:` field).
Source of truth: `.aether/model-profiles.yaml` `worker_models` section.

## Step 2: Check System Status

Run using Bash tool: `bash .aether/aether-utils.sh version-check 2>/dev/null || echo "Utils available"`

Check LiteLLM proxy status:
```bash
curl -s http://localhost:4000/health 2>/dev/null | grep -q "healthy" && echo "✓ Proxy healthy" || echo "⚠ Proxy not running"
```

## Step 3: Show Current Model Configuration

Display the active model mapping:

```
MODEL CONFIGURATION
───────────────────
To change models, swap settings files:

  GLM Proxy mode:
    cp ~/.claude/settings.json.glm ~/.claude/settings.json
    (opus -> glm-5, sonnet -> glm-5-turbo, haiku -> glm-4.5-air)

  Claude API mode:
    cp ~/.claude/settings.json.claude ~/.claude/settings.json
    (opus -> claude-opus-4, sonnet -> claude-sonnet-4, haiku -> claude-haiku-4)

Current model mapping can be verified by reading agent frontmatter:
  grep "^model:" .claude/agents/ant/*.md
```

## Step 4: Summary

```
═══════════════════════════════════════════
System Status
═══════════════════════════════════════════
Utils: ✓ Operational
Proxy: {status from Step 2}
Castes: 22 defined (8 opus, 11 sonnet, 3 inherit)
Routing: Per-caste via agent frontmatter model: field
```

## Historical Note

Per-caste model routing was initially attempted using environment variable
injection at spawn time (archived in `.aether/archive/model-routing/`).
That approach failed due to Claude Code Task tool limitations.

The current approach uses agent frontmatter `model:` fields, which Claude Code
handles natively. No Aether code intervention is required -- Claude Code reads
the frontmatter and resolves the slot name through `ANTHROPIC_DEFAULT_*_MODEL`
environment variables.

To view the archived v1 configuration:
```bash
git show model-routing-v1-archived
```
