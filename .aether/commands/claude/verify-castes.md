<!-- Generated from .aether/commands/verify-castes.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-verify-castes
description: "✓ Verify colony caste assignments and system status"
---

You are the **Queen**. Display the caste assignments and system status.

## Step 1: Show Caste Assignments

Display the colony caste structure as a compact table:


```
Aether Colony Caste System
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━



CASTE ASSIGNMENTS
────────────────────────────────────
Caste                Slot     Active
────────────────────────────────────
[reasoning]
  Archaeologist      opus     yes
  Architect          opus     yes
  Auditor            opus     yes
  Gatekeeper         opus     yes
  Measurer           opus     yes
  Oracle             opus     yes
  Queen              opus     yes
  Route-setter       opus     yes
  Sage               opus     yes
  Tracker            opus     yes
────────────────────────────────────
[execution]
  Ambassador         sonnet   yes
  Builder            sonnet   yes
  Chaos              sonnet   yes
  Disciplines        sonnet   yes
  Nest               sonnet   yes
  Pathogens          sonnet   yes
  Probe              sonnet   yes
  Provisions         sonnet   yes
  Scout              sonnet   yes
  Weaver             sonnet   yes
  Watcher            sonnet   yes
────────────────────────────────────
[inherit]
  Chronicler         inherit  yes
  Includer           inherit  yes
  Keeper             inherit  yes

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


```

Source of truth: Agent frontmatter `model:` fields in `.claude/agents/ant/*.md`.
Caste slots come from agent frontmatter (`model:` field).

## Step 2: Check System Status


Run using the Bash tool with description "Checking colony version...": `aether version-check-cached 2>/dev/null || echo "Utils available"`



Check LiteLLM proxy status:
```bash
curl -s http://localhost:4000/health 2>/dev/null | grep -q "healthy" && echo "✓ Proxy healthy" || echo "⚠ Proxy not running"
```

## Step 3: Show Current Model Configuration

Display the active model configuration:


```
MODEL CONFIGURATION
──────────────────


Default: Claude API mode (opus -> claude-opus-4, sonnet -> claude-sonnet-4)

To switch to GLM Proxy mode:
  cp ~/.claude/settings.json.glm ~/.claude/settings.json
  (opus -> glm-5, sonnet -> glm-5-turbo, haiku -> glm-4.5-air)

To switch back to Claude API:
  cp ~/.claude/settings.json.claude ~/.claude/settings.json

```



Current model mapping from agent frontmatter:
```bash
grep "^model:" .claude/agents/ant/*.md
```

## Step 4: Summary


```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
System Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━


Utils: ✓ Operational
Proxy: {status from Step 2}
Castes: 24 defined (10 opus, 11 sonnet, 3 inherit)
Routing: Per-caste via agent frontmatter model: field
```


## Step 5: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
aether print-next-up
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
