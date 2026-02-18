# Known Issues

This document tracks known issues and their workarounds in the Aether system.

## teammateMode Must Be "off" with LiteLLM Proxy

**Severity:** Medium (causes false "failed" messages - work completes successfully)

**Symptoms:** When spawning agents via the Task tool, the agent completes successfully but throws:
```
Agent "Builder: Task X" failed: classifyHandoffIfNeeded is not defined
```

**Root Cause:** Claude Code's internal completion handler calls `classifyHandoffIfNeeded` to route completed subagent results. This function is not available when routing through LiteLLM proxy to non-Anthropic models (kimi-k2.5, glm-5, minimax-2.5).

**Status:** This is a Claude Code runtime bug - not fixable via configuration. The `teammateMode: "off"` setting is already correctly applied but does not fully prevent the error.

**Workaround:**
1. Verify the agent actually completed successfully by checking:
   - SUMMARY.md file exists
   - Git commits were created
   - Expected output files are present
2. If work completed, ignore the error message - it's a false failure

**GSD Integration:** GSD workflows (execute-phase.md, execute-plan.md, quick.md) already handle this by running spot-checks and treating it as successful if the work completed.

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
