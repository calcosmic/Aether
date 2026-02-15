# Known Issues

This document tracks known issues and their workarounds in the Aether system.

## teammateMode Must Be "off" with LiteLLM Proxy

**Severity:** High (causes all agent completions to crash)

**Symptoms:** When spawning agents via the Task tool, the agent completes successfully but throws:
```
Agent "Builder: Task X" failed: classifyHandoffIfNeeded is not defined
```

**Root Cause:** The `teammateMode` setting in Claude Code's settings.json attempts to use the `classifyHandoffIfNeeded` function to route completed subagent results to other "teammates." This function is not available when routing through LiteLLM proxy to non-Anthropic models (kimi-k2.5, glm-5, minimax-2.5).

**Fix:** Set `"teammateMode": "off"` in `~/.claude/settings.json`:
```json
"teammateMode": "off"
```

**Prevention:** Do not enable `teammateMode` when using LiteLLM proxy with non-Anthropic models.

---

## Agent Registration - 5 Aether Agents Not Recognized

**Severity:** Medium (causes "Agent type not found" errors)

**Affected Agents:**
- aether-route-setter
- aether-architect
- aether-colonizer
- aether-archaeologist
- aether-chaos

**Symptoms:** When commands try to spawn these agents, Claude Code returns "Agent type not found" even though all 21 agent files exist in `~/.claude/agents/` with valid frontmatter.

**Root Cause:** Likely D - Claude Code needs to be restarted to pick up newly added agents. The 5 failing agents were added at a later timestamp (12:18) than the working ones (12:05).

**Workaround:** Restart Claude Code to register the new agents. The agents should work after restart.

**Fallback Strategy (if restart doesn't work):** Use `general-purpose` agent type with the role description injected into the prompt. For example:
- Instead of `subagent_type="aether-route-setter"`, use `subagent_type="general-purpose"` and prepend the prompt with "You are a Route-Setter Ant - creates structured phase plans and analyzes dependencies."
