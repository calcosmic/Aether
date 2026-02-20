# Pitfalls Research: Claude Code Subagent Creation

**Domain:** Adding Claude Code subagents to an existing multi-agent system (Aether)
**Researched:** 2026-02-20
**Confidence:** HIGH — primary source is official Claude Code docs (code.claude.com/docs/en/sub-agents), corroborated by community issue threads, practitioner blogs, and direct inspection of working vs. weak agents in `.claude/agents/` and `everything-claude-code/agents/`

---

## Critical Pitfalls

### Pitfall 1: Vague Description Kills Auto-Routing

**What goes wrong:**
Claude uses each agent's `description` field to decide when to delegate. Descriptions that describe *what the agent is* instead of *when to use it* are not picked up by auto-routing. The main conversation handles the task itself instead of delegating to the specialist. The agent exists but is never invoked.

**Why it happens:**
The natural instinct is to write a description like "A scout agent for research tasks." This describes identity, not routing signal. The auto-router is looking for trigger conditions and task types, not role labels.

**Evidence from agent comparison:**
Weak (community, `architect.md`): `"Software architecture specialist for system design, scalability, and technical decision-making."`

Strong (GSD, `gsd-planner.md`): `"Creates executable phase plans with task breakdown, dependency analysis, and goal-backward verification. Spawned by /gsd:plan-phase orchestrator."`

The strong description specifies what the agent *produces* and who *spawns it*. The weak description is a job title.

**How to avoid:**
- Lead with action: what does the agent *do*, not what it *is*
- Include trigger phrases: "Use proactively when...", "Use IMMEDIATELY after...", "MUST BE USED for..."
- Specify output type: "...and returns a structured JSON summary"
- Name the spawner if it has one: "Spawned by /ant:build orchestrator"
- Keep under 150 characters — the router reads this, not a human

**Warning signs:**
- Agent file exists in `.claude/agents/` but the Queen never delegates to it
- You have to explicitly say "use the aether-scout agent" for it to activate
- Description reads like a LinkedIn bio

**Phase to address:** Definition phase, before writing any agent body. Description is the most important line in the file.

---

### Pitfall 2: Subagents Cannot Spawn Other Subagents — Breaking the Colony Model

**What goes wrong:**
The existing Aether agents assume they can spawn child workers via `spawn-log`, `spawn-can-spawn`, and the task tool. In Claude Code's subagent model, subagents cannot spawn other subagents. A subagent that calls the Task tool receives nothing — no error, no nested agent, just silence. The spawn hierarchy that Aether's Queen coordinates does not translate directly.

**Why it happens:**
The official docs state explicitly: "Subagents cannot spawn other subagents. If your workflow requires nested delegation, use Skills or chain subagents from the main conversation." The Aether OpenCode agents were designed with spawn depth 0-3 in mind. Claude Code subagents are always depth 1.

**Consequences:**
An `aether-builder` converted naively from OpenCode format will contain instructions to call `spawn-can-spawn` and spawn sub-workers. In Claude Code, these calls produce nothing. The builder silently proceeds without the sub-workers it expected, producing incomplete output with no diagnostic.

**How to avoid:**
- Remove all spawning instructions from converted agents
- The Queen (main conversation or orchestrator) remains the only spawner
- Design agents to complete their work independently, not through sub-delegation
- If an agent genuinely needs parallelism, that parallelism must be orchestrated by the caller before invoking the agent

**Warning signs:**
- Agent body contains `spawn-can-spawn`, `spawn-log`, or Task tool calls
- Converted agent silently produces less output than expected
- `spawns: []` in agent output where you expected sub-workers

**Phase to address:** Conversion phase. Strip all spawn machinery from agent bodies before adding Claude Code-specific content.

---

### Pitfall 3: Name Collision Silently Overrides Agent Behavior

**What goes wrong:**
Claude Code applies implicit behavior when an agent name matches a known pattern. An agent named `code-reviewer` may have generic code review rules silently merged with your custom instructions, overriding carefully written constraints. Additionally, the name priority hierarchy means a user-level agent (`~/.claude/agents/aether-scout.md`) can silently win over a project agent (`.claude/agents/aether-scout.md`) with the same name.

**Why it happens:**
Two separate issues:
1. Claude Code's internal heuristics infer function from agent names, potentially injecting default behavior
2. When identical names exist at multiple scopes (user > project), the higher priority location wins — silently

For Aether specifically: the GSD agents in `.claude/agents/` use names like `gsd-executor`, `gsd-planner`. Aether agents will use `aether-*` names. But if a user has `~/.claude/agents/aether-scout.md` from a previous install and the project has `.claude/agents/aether-scout.md` from a new version, the user-level agent wins — meaning the old version runs, not the new one.

**How to avoid:**
- Use the `aether-` prefix consistently — it is meaningfully distinct from GSD's `gsd-` prefix and avoids community agent collisions
- Run `/agents` to inspect which version of each agent is actually active before testing
- Avoid names that are generic job titles Claude might have priors on (`reviewer`, `planner`, `builder` alone)
- During distribution: ensure the npm package distributes agents to `.claude/agents/` (project scope, priority 2) and document that user-level agents at `~/.claude/agents/` will override if they share a name

**Warning signs:**
- Agent behaves differently than its system prompt specifies
- `/agents` shows the agent but its active version is from `~/.claude/agents/` not `.claude/agents/`
- Agent follows rules you never wrote for it

**Phase to address:** Naming convention phase before writing agent bodies. Verify with `/agents` command after each agent is added.

---

### Pitfall 4: Aether-Specific Machinery Doesn't Translate to Claude Code Context

**What goes wrong:**
The OpenCode agents contain Aether-specific shell calls that assume `.aether/aether-utils.sh` is accessible and the colony state files exist. When these agents run as Claude Code subagents, the working directory may differ, the `.aether/` directory structure may not exist (in target repos that haven't initialized), and `aether-utils.sh` functions like `activity-log`, `spawn-log`, and `state-get` may fail silently.

Specific patterns that break:
- `bash .aether/aether-utils.sh activity-log "ACTION" "name" "description"` — fails if `aether-utils.sh` is absent
- `bash .aether/aether-utils.sh spawn-can-spawn {depth}` — fails AND is meaningless (subagents can't spawn)
- Output format requires `ant_name`, `caste`, `spawns` — the GSD agents expect different handoff formats

**Why it happens:**
OpenCode agents were designed for the Aether system and assume Aether infrastructure is present. Claude Code subagents run in their own context window with only what their system prompt tells them. The ambient Aether context does not automatically load.

**How to avoid:**
- Remove all `aether-utils.sh` calls from the converted agent body, OR make them optional with graceful fallback: `bash .aether/aether-utils.sh activity-log ... 2>/dev/null || true`
- Replace the output format with whatever the calling slash command expects
- Add a guard at the top of any agent that uses Aether tools: check if `.aether/aether-utils.sh` exists before calling it
- For agents distributed to target repos: assume `.aether/` exists (initialized repos only). Document this precondition in the agent description.

**Warning signs:**
- Agent produces no output or exits early without explanation
- `bash: .aether/aether-utils.sh: No such file or directory` in tool output
- Slash command receives malformed handoff because agent output format doesn't match expected schema

**Phase to address:** Conversion phase. Each agent needs a porting checklist: strip spawn calls, make Aether tool calls optional, verify output format matches caller expectations.

---

### Pitfall 5: Tool Inheritance Makes Agents More Permissive Than Intended

**What goes wrong:**
If the `tools` field is omitted from a subagent's frontmatter, the agent inherits all tools from the main conversation, including MCP tools. An `aether-scout` intended as read-only silently has Write, Edit, and Bash access if the calling session has those tools. The agent may use them, overstepping its intended boundaries.

This is the opposite of the OpenCode model where agent capabilities come from what the agent is prompted to do. In Claude Code, tool access is mechanically granted unless explicitly restricted.

**Why it happens:**
Developers converting from OpenCode assume the agent's role description ("you are a read-only scout") is sufficient to prevent write actions. In Claude Code, the description shapes behavior but does not mechanically prevent tool use. The `disallowedTools` field or explicit `tools` allowlist is required.

**Concrete consequences:**
- A `aether-scout` that accidentally calls Write, corrupting files it should only read
- A `aether-watcher` with Bash access running build commands it shouldn't
- Security surface expanded unexpectedly when MCP tools are available to main conversation and thus inherited

**How to avoid:**
- **Every agent must have an explicit `tools` field** — never rely on inheritance
- Read-only agents: `tools: Read, Grep, Glob, Bash` (no Write, Edit)
- Research agents: add `WebSearch, WebFetch` — but consider whether WebFetch is needed
- Agents that should not modify files: add `disallowedTools: Write, Edit`
- Review each converted agent's intended capability against its `tools` list before merging

**Warning signs:**
- Agent file has no `tools` field
- Agent that should be read-only modifies or creates files
- Agent has access to tools that have nothing to do with its purpose (e.g., a scout with WebSearch but also Write)

**Phase to address:** Conversion phase, simultaneously with body conversion. Tools list is as important as description.

---

### Pitfall 6: Model Selection Is Not Free — Opus for Everything Breaks Cost and Speed

**What goes wrong:**
The community `architect.md`, `code-reviewer.md`, `planner.md`, and `tdd-guide.md` all specify `model: opus`. If all Aether agents follow this pattern, every subagent invocation uses Opus pricing. For a colony that spawns 4-8 agents per phase, this is a significant cost multiplier. Worse, Opus has higher latency — a colony where every agent uses Opus is noticeably slower than one that routes appropriately.

**Why it happens:**
"Use the best model" is the default instinct when quality matters. But the entire point of model routing is that different tasks have different requirements. Research and exploration agents don't need Opus. Verification agents running deterministic checks don't need Opus.

**Recommended routing for Aether agents:**
| Agent Type | Recommended Model | Rationale |
|------------|-----------------|-----------|
| Queen / Route-Setter | `inherit` or `opus` | Complex orchestration, reasoning about plans |
| Builder / Weaver | `sonnet` | Implementation requires capability, not Opus |
| Scout / Chronicler | `haiku` | Research and summarization at lower cost |
| Watcher / Probe | `sonnet` | Verification needs accuracy, not creativity |
| Keeper / Auditor | `sonnet` | Analysis tasks, not novel reasoning |

**Warning signs:**
- All agents specify `model: opus`
- Colony cost per phase is unexpectedly high
- No model field means `inherit` — which is acceptable but check what the parent model is

**Phase to address:** Conversion phase. Set model per agent class, not per individual agent.

---

## Moderate Pitfalls

### Pitfall 7: Description That Triggers for Everything (Over-Routing)

**What goes wrong:**
Overly broad descriptions cause Claude to route tasks to the wrong agent. An `aether-builder` described as "use for any coding task" will be invoked when you ask a question about architecture, when you want research, when you want verification — because all of those involve code somehow.

The GSD agents avoid this by being specific about the workflow context: `gsd-executor` is "spawned by execute-phase orchestrator" — not "use whenever you want to execute something."

**How to avoid:**
- Include exclusions in the description: "Use for implementation tasks. Do not use for research — use aether-scout instead."
- Pair trigger phrases with anti-trigger phrases
- After writing descriptions, do a routing test: read each description and ask "would the main LLM route a code review request to this agent?" If yes for the wrong agent, the description is too broad.

**Warning signs:**
- Multiple agents activated for a single request
- Agent invoked for tasks outside its intended scope
- Queen receives unexpected output format because a different agent was selected

**Phase to address:** Definition phase. Test routing after all agents are defined by running `/agents` and reading descriptions side by side.

---

### Pitfall 8: Context Budget Waste from Over-Referencing

**What goes wrong:**
The GSD planner agent's system prompt is 1,100+ lines. When spawned as a subagent, this entire prompt occupies context before any work begins. If the agent additionally tries to load multiple reference files (`@.planning/ROADMAP.md`, `@.planning/STATE.md`, `@.planning/PROJECT.md`) at startup, a substantial fraction of the context window is consumed before the task begins. Complex tasks then hit context limits before completion.

**Why it happens:**
Agents inherited from the main conversation pattern load everything relevant "just in case." Subagents have the same context limit as the main conversation but receive no pre-loaded context — they start from only their system prompt. Adding 15+ file reads at startup replicates the worst of both worlds.

**How to avoid:**
- Agent system prompts should be concise: under 500 lines for most agents, under 200 for specialists
- Load files on demand (when needed for a specific task), not at startup
- Use `skills` frontmatter field to preload domain knowledge instead of instructing the agent to load it via Read tool calls
- Pass needed context in the spawn prompt from the orchestrator, not by having the agent discover it

**Warning signs:**
- Agent system prompt is over 500 lines
- Agent starts every invocation by reading 5+ files before doing any work
- Agent hits context limit on complex tasks

**Phase to address:** Conversion phase. Apply a length budget to each agent body.

---

### Pitfall 9: Slash Command Integration — Output Format Mismatch

**What goes wrong:**
Existing Aether slash commands (`/ant:build`, `/ant:colonize`) spawn OpenCode agents and expect structured JSON output. If the same commands are updated to spawn Claude Code subagents, the expected output format must match. OpenCode agents return `{"ant_name": ..., "caste": ..., "spawns": [...]}`. If the converted Claude Code agents return plain markdown or a different schema, the slash command's parsing logic breaks silently — no error, just malformed colony state.

**Why it happens:**
Output format is never formally declared — it is an implicit contract between the agent and the command that spawns it. When converting agents, developers focus on the agent body and forget to verify that the output format the command parser expects matches what the agent now produces.

**How to avoid:**
- Before converting each agent, read the slash command that spawns it and identify every field it reads from agent output
- Explicitly document the expected output format in the agent's system prompt: "Return output as JSON with fields: summary, files_created, blockers"
- Add an output format section at the end of the converted agent body
- After each conversion, run the slash command end-to-end and verify the calling command receives parseable output

**Warning signs:**
- Colony state updates silently with empty fields after running a build
- Slash command shows "completed" but state files are not updated
- `jq` parse errors in any command that processes agent output

**Phase to address:** Integration testing phase, after agents are converted but before slash commands are updated.

---

### Pitfall 10: Distribution — Agents Must Reach Target Repos

**What goes wrong:**
Claude Code subagents in `.claude/agents/` are project-scoped. When Aether distributes via `npm install -g . → hub → aether update`, the distribution chain must explicitly include `.claude/agents/` files. If the hub sync excludes `.claude/agents/` (treating it as a local directory), target repos never receive the agents even though they exist in the Aether source repo.

Currently, the Aether distribution chain uses `HUB_EXCLUDE_DIRS` to exclude private directories. The `.claude/agents/` directory is not currently excluded — but it is also not explicitly included in the validated SYSTEM_FILES list. Whether it syncs to hub depends on how `setupHub()` in `bin/cli.js` handles the `.claude/` directory.

**Why it happens:**
The distribution chain was designed before Claude Code subagents existed. The `.claude/` directory was primarily a container for slash commands. Adding agents to `.claude/agents/` extends the distribution surface without updating the distribution chain's understanding of what `.claude/` contains.

**How to avoid:**
- Verify that `aether update` in a test repo places agents in `.claude/agents/` of the target repo
- Check `bin/cli.js` hub sync logic to confirm `.claude/agents/` is included (not excluded or missed)
- Add `.claude/agents/` to `validate-package.sh` SYSTEM_FILES if it is not already present
- Test the full chain: edit an agent in source, `npm install -g .`, `aether update` in target, verify agent appears

**Warning signs:**
- `npm pack --dry-run` does not include `.claude/agents/*.md`
- After `aether update` in a test repo, `.claude/agents/` is empty or missing agents
- Agents work in the Aether source repo but not in any other repo

**Phase to address:** First phase that adds agents. Verify distribution before declaring any agent "shipped."

---

### Pitfall 11: Skipping the /agents Command Means Missing Load Errors

**What goes wrong:**
Subagent files are loaded at session start. If an agent file has malformed YAML frontmatter (extra quotes, wrong indentation, invalid field values), it silently fails to load. The agent appears absent from `/agents` but produces no error message. Developers assume the agent "doesn't exist yet" and create it again, creating duplicate files with different content.

**Why it happens:**
YAML is whitespace-sensitive and the Claude Code frontmatter schema has specific valid values for fields like `model` (`sonnet`, `opus`, `haiku`, `inherit` — not `claude-sonnet-4-5`), `permissionMode`, and `tools`. A single formatting error silently drops the entire agent.

The official docs note: "Subagents are loaded at session start. If you create a subagent by manually adding a file, restart your session or use `/agents` to load it immediately."

**How to avoid:**
- After creating or editing any agent file, restart the session and run `/agents` to confirm it loaded
- Validate YAML frontmatter: `model` must be `sonnet`, `opus`, `haiku`, or `inherit` — not a full model ID
- `tools` must be a comma-separated list of tool names or an array — not arbitrary descriptions
- Name must use only lowercase letters and hyphens — no spaces, no underscores, no uppercase

**Warning signs:**
- Agent does not appear in `/agents` after creating the file
- Agent appears then disappears after a restart
- You can explicitly reference the agent by name but it is not auto-routed

**Phase to address:** Every phase that creates or modifies agent files. Make `/agents` verification a required step after each file change.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Copy OpenCode agent directly, change frontmatter only | Fast conversion | Agent contains spawn calls and Aether tool calls that break silently | Never — always audit and strip incompatible sections |
| Omit `tools` field to inherit everything | No decisions needed now | Agent is more permissive than intended; security surface grows | Never — every agent must have explicit tools |
| Use `model: opus` for all agents | Guaranteed quality | Cost multiplies with every spawn; colony becomes slow | Only for Queen and Route-Setter |
| Write description as role label | Quick to write | Never auto-routed; must be explicitly invoked | Never — descriptions must be routing signals |
| Skip `/agents` verification after creating file | Faster workflow | Silent YAML errors go undetected | Never — always verify load after file creation |

---

## Integration Gotchas

Common mistakes when connecting Claude Code agents to existing Aether infrastructure.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Slash command → agent handoff | Slash command passes minimal context, agent has to rediscover everything | Pass rich context in the spawn prompt: current state, goal, relevant file paths |
| Agent → colony state write | Agent writes COLONY_STATE.json directly | Agent returns structured output; slash command does the state write via `aether-utils.sh` |
| Activity logging | Agent calls `aether-utils.sh activity-log` which may not exist | Make `activity-log` calls optional with `|| true`; or move logging to the slash command level |
| Parallel agents editing same file | Two agents update `constraints.json` at the same time | Use file locks or ensure parallel agents have mutually exclusive file ownership |
| OpenCode vs Claude Code agent format | Assume formats are interchangeable | They are not: tools, model routing, spawn behavior, and output handling differ |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Agent is written:** Does not mean it is loaded — verify with `/agents` after every file change
- [ ] **Agent is invoked:** Does not mean it was routed correctly — verify it was the intended agent, not a fallback
- [ ] **Agent produces output:** Does not mean the output format matches what the calling slash command expects — verify the schema
- [ ] **Agent has tools listed:** Does not mean the tools are sufficient — verify the agent can complete its task with only the tools listed
- [ ] **Agent converts from OpenCode:** Does not mean spawn calls and Aether tool calls were removed — audit each converted file
- [ ] **Agent distributed in npm package:** Does not mean `aether update` delivers it to target repos — verify the full distribution chain
- [ ] **Tests pass:** Does not mean agent behavioral constraints work — test with actual invocations

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Wrong agent auto-routed | LOW | Fix description, restart session, test routing again |
| Spawn calls silently fail | LOW | Add spawn section audit to conversion checklist; strip calls from agent body |
| Name collision with user-level agent | LOW | Check `~/.claude/agents/` for duplicates; rename or remove conflicting file |
| Tool over-permission causes file corruption | HIGH | `git checkout` to restore files; add `disallowedTools` or explicit tools allowlist |
| Output format mismatch breaks colony state | MEDIUM | Fix agent output format; run `aether-utils.sh state-get` to verify state is valid; repair state file if corrupted |
| Distribution chain missing agents | MEDIUM | Fix `validate-package.sh` or `cli.js` to include `.claude/agents/`; re-publish; run `aether update` in target repos |
| YAML malformation silently drops agent | LOW | Fix YAML, restart session, verify with `/agents` |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Vague description kills auto-routing | Agent definition phase | Routing test: read all descriptions side-by-side, verify no ambiguity |
| Subagents cannot spawn | Conversion phase | Audit each converted agent for spawn calls before merge |
| Name collision | Naming convention decision, pre-conversion | `/agents` shows correct version from correct scope |
| Aether-specific machinery breaks | Conversion phase (per-agent checklist) | Each agent runs in isolation without Aether system available |
| Tool over-permission | Conversion phase | Read-only agents tested with Write tool denied; verify no unintended file mutations |
| Model selection cost | Conversion phase | Model assigned per agent class, not blindly set to opus |
| Over-routing from broad description | Definition phase | End-to-end routing test with realistic prompts |
| Context budget waste | Conversion phase | Agent system prompt under 500 lines; no bulk file reads at startup |
| Output format mismatch | Integration testing phase | Run each slash command end-to-end; verify state files updated correctly |
| Distribution chain gap | First distribution phase | `npm pack --dry-run` includes agents; `aether update` delivers them to test repo |
| YAML malformation | Every agent creation | `/agents` check after every file write |

---

## Sources

- [Create custom subagents — Claude Code official docs](https://code.claude.com/docs/en/sub-agents) — HIGH confidence, current (2026)
- [Claude Code Subagents: Common Mistakes (claudekit.cc)](https://claudekit.cc/blog/vc-04-subagents-from-basic-to-deep-dive-i-misunderstood) — MEDIUM confidence, practitioner experience
- [Sub-Agent Best Practices (claudefa.st)](https://claudefa.st/blog/guide/agents/sub-agent-best-practices) — MEDIUM confidence, community patterns
- [Best practices for Claude Code sub-agents (PubNub)](https://www.pubnub.com/blog/best-practices-for-claude-code-sub-agents/) — MEDIUM confidence, team implementation learnings
- [Custom agents (ClaudeLog)](https://claudelog.com/mechanics/custom-agents/) — MEDIUM confidence
- [GitHub issue: subagents not called after v2.0.1](https://github.com/anthropics/claude-code/issues/8558) — HIGH confidence, confirmed bug + fix
- [GitHub issue: custom agents not appearing](https://github.com/anthropics/claude-code/issues/5185) — HIGH confidence, YAML load failure patterns
- Direct inspection of `.claude/agents/gsd-executor.md`, `gsd-planner.md` — working reference implementations
- Direct inspection of `everything-claude-code/agents/` — community quality range
- Direct inspection of `.opencode/agents/aether-*.md` — source format for conversion

---

*Pitfalls research for: Claude Code subagent creation — adding agents to Aether colony system*
*Researched: 2026-02-20*
