# Feature Research

**Domain:** Multi-agent coding frameworks — recursive delegation, user-extensible agents/skills, orchestration explainability
**Milestone:** v1.26 Intelligent Orchestration
**Researched:** 2026-08-08
**Confidence:** MEDIUM-HIGH (Claude Code findings HIGH from official docs; CrewAI/LangGraph/AutoGen/OpenHands MEDIUM from multiple agreeing secondary sources; roster-size guidance LOW-MEDIUM from community consensus only)

---

## How to read this document

Four feature areas were researched. Each gets: what comparable tools actually do, what the convention is, what a good vs frightening user experience looks like, and what it depends on inside Aether today.

The owner of this project is non-technical and will not type flags. Every recommendation below is written so the default behaviour is correct without configuration, following the precedent already set by Queen-owned verification depth (`CLAUDE.md` → Queen-Owned Orchestration) and by `AETHER_HIVE_POLICY` (one named switch, fail-safe on an unrecognised value).

---

## Area 1: Recursive Delegation

### What comparable systems actually do

| System | Can a sub-agent spawn? | Depth bound | Second bound | Behaviour at the limit | Confidence |
|--------|------------------------|-------------|--------------|------------------------|------------|
| **Claude Code** | Yes, by default | 3 layers below main; `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`, set `1` to disable | 20 concurrent (`CLAUDE_CODE_MAX_CONCURRENT_SUBAGENTS`); per-agent `maxTurns` | **Withholds the `Agent` tool.** The agent does the work itself and returns one summary. No error. | HIGH (official docs) |
| **CrewAI** | Only if `allow_delegation=True` — **off by default** | None (no depth concept) | `max_iter` per agent, `max_rpm` per crew, `allowed_agents` allowlist | Iteration cap. Documented failure: a manager with delegation and no `max_iter` **recurses indefinitely** | MEDIUM |
| **LangGraph** | Yes, via subgraphs / supervisor handoff | None (no depth concept) | `recursion_limit`, default 25 *steps* | **Raises `GraphRecursionError`** — a crash. Community writes wrappers so the parent can catch, retry, or degrade | MEDIUM |
| **AutoGen** | Yes, nested chats | None | `max_round`, `max_turns`, termination-message predicate | Stops at round cap. Documented failure: speaker-selection mismatch silently re-selects the same agent → infinite loop, ballooning history, climbing API cost | MEDIUM |
| **OpenHands** | Yes, `AgentDelegateAction`; sub-agents are independent conversations inheriting parent model + workspace | Not documented as a hard cap | Per-conversation limits | Not clearly documented | MEDIUM-LOW |
| **Devin** | Not exposed to users at all | N/A | **ACU budget per session** (10 max initially), consumption dashboard, auto-recharge caps | Session stops when budget is exhausted | MEDIUM |

### The three conventions worth copying

**1. Depth 2–3 is the settled answer, and the industry got there by retreating from something worse.**

Claude Code's version history is the single most useful data point in this research, because it is a public record of a team changing its mind twice in three releases:

- v2.1.172–v2.1.216: default depth **5**, not configurable
- v2.1.217–v2.1.218: default dropped to **1** (nesting effectively off)
- v2.1.219 onward: default raised to **3**, configurable

Five was too deep. Off entirely was too blunt. Three is where it landed. Aether should ship **2 by default with 3 as the ceiling** — one step more conservative than Claude Code, because Aether workers are longer-running and more expensive per unit than a Claude Code subagent, and because the owner cannot inspect a runaway tree by reading code.

**2. Depth alone is not a budget. Every serious system has at least two independent bounds.**

This is the most important finding in the section. A tree that is 2 deep and 8 wide at each level is 72 workers; a tree that is 5 deep and 1 wide is 5. Depth constrains the wrong dimension. Claude Code pairs depth with a concurrency cap and per-agent `maxTurns`. CrewAI pairs delegation with `max_iter` and `max_rpm`. AutoGen pairs nesting with `max_round`. Devin skips depth entirely and bounds only the thing that actually costs money.

Aether already has the width bound (`cmd/queen_spawn_budget.go` — per-phase max workers). It has **no** depth bound and **no** run-total bound. The gap is a *run-total worker counter*, not a depth number.

**3. At the limit, degrade — never crash.**

Claude Code withholds the `Agent` tool from an at-limit subagent, so the agent transparently does the work itself and returns a normal result. The user never sees an error. LangGraph does the opposite — `GraphRecursionError` — and the LangChain community forum has an active thread on how to catch it in the parent so the run doesn't die. One of these is a feature; the other is a bug people work around.

Aether's equivalent: when a worker hits the depth or run-total limit, its brief simply does not offer the delegation instruction, and the runtime records a "delegation withheld at depth N" event. The work still gets done, one level shallower.

### What a GOOD nested-delegation experience looks like

Assembled from Claude Code's `/tasks` panel, Devin's TUI, and OpenHands' GUI:

1. **Every spawn produces one visible line, attributed to its parent.** Claude Code renders it as `code-improver (Suggest code improvements)`. Aether already has richer identity — `🔨 Builder Mason-67  Task description` — and `SpawnEntry` already carries `ParentName` and `Depth`. Indentation by depth is nearly free.
2. **A live list of what is running right now**, not just a scroll-back log. Claude Code's `/tasks`; Devin's TUI. Aether has `spawn-tree-active` and a swarm dashboard in the TS host.
3. **One keypress stops one agent.** In Claude Code's `/tasks`, `x` stops the selected running subagent; `Enter` opens its transcript. Crucially the stop is *per agent*, from a list — not "kill the whole run."
4. **A depleting counter, visible without asking.** Devin's model. "14 of 20 workers used this run" is the single most valuable widget for a non-technical owner, because it converts an abstract runaway risk into a number that goes down.
5. **When a nested agent needs permission, the prompt names it.** Claude Code v2.1.186+ surfaces a background subagent's permission prompt in the main session *and says which subagent is asking*. Before that it silently auto-denied — a documented regression they fixed. Aether should never let a nested worker's blocker surface anonymously.

### What a FRIGHTENING one looks like

Directly from the failure reports found:

- **Silent nesting where only the final summary returns.** Claude Code's docs describe this as the *benefit* ("the intermediate output never reaches your main conversation"), and for a developer it is. For a non-technical owner it is the entire risk: fifteen workers ran, one paragraph came back, the bill arrives later.
- **AutoGen's canonical failure:** "agents keep talking, conversation history balloons, API costs climb, and eventually the process crashes or you kill it manually." Note the ending — *you kill it manually*. There was no stop affordance and no counter.
- **CrewAI's:** a manager with delegation enabled and no `max_iter` recursing indefinitely.
- **A crash as the safety mechanism** (LangGraph). The user's experience of hitting a safety limit should not be indistinguishable from the tool breaking.

### Non-technical owner angle

The owner should never see the words "depth", "recursion", or "budget flag". What they should see is a tree that indents, a counter that depletes, and a stop key. The Queen picks the limits, exactly as she already picks verification depth. If a limit is hit, the message is "the team stopped growing and finished the work themselves" — not an error.

### Aether dependencies and current state

| Dependency | State | Notes |
|-----------|-------|-------|
| `pkg/agent/spawn_tree.go` — `SpawnEntry{ParentName, Depth, Status}` | **EXISTS** | The delegation-tree data model is already there and already records depth. Nothing new to model. |
| `cmd/spawn.go` — `spawn-log --depth`, `spawn-tree-load`, `spawn-tree-active`, `spawn-tree-depth`, `spawn-efficiency` | **EXISTS** | Read/write surface complete. |
| `cmd/spawn.go:170` — `spawn-can-spawn` | **STUB — DOES NOT ENFORCE** | It accepts `--depth`, ignores it entirely, and unconditionally returns `{"can_spawn": true}`. No maximum is consulted anywhere. |
| Callers of `spawn-can-spawn` | **NONE LIVE** | Its only two call sites are `.aether/docs/command-playbooks/build-wave.md:675` and `build-full.md:854`, which per `CLAUDE.md` are **not loaded by the runtime since v1.25**. The gate is both non-functional and uncalled. |
| `cmd/internal_cmds.go:538` — `spawn-can-spawn-swarm` | **EXISTS AND ENFORCES** | Compares active spawn-tree lines against a budget (default 5, from `state.ColonyDepth`). This is a *concurrency* check, swarm-only. It is the working model to generalise. |
| `cmd/queen_spawn_budget.go` | **EXISTS** | Per-phase wave width, required castes, `applyBuildDepthToBudget`. Constrains width, not depth or run total. |
| Worker handoff (`.aether/data/handoffs/worker-handoffs.json`) | **EXISTS** | A sub-spawned worker must also emit a handoff, or the relay note chain breaks at the nested level. |
| `parallel_mode: worktree` | **EXISTS** | Open design question: does a child inherit the parent's worktree? Claude Code's answer is yes — a worktree-isolated session applies the same isolation checks to every subagent it spawns. Recommend matching. |
| Token usage measurement | **EXISTS** (per milestone context) | Feeds the depleting counter. |

> **This is the project's signature failure pattern.** `spawn-can-spawn` is a gate that returns `true` unconditionally, called only from files the runtime no longer loads. It is the same shape as `suggest-analyze` (v1.11) and `consolidation-phase-end` (pre-v1.25) as recorded in `CLAUDE.md`'s Definition of Done. Any v1.26 requirement here must be satisfied by a command that **fails when the limit is unenforced** — not by the existence of `spawn-can-spawn`.

---

## Area 2: User-Extensible Agents

### What comparable systems require

| System | Minimum a user must specify | Authoring surface | Where it lives |
|--------|----------------------------|-------------------|----------------|
| **Claude Code** | **`name` + `description`. That is all.** 16 other frontmatter fields are optional | Markdown + YAML frontmatter, hand-written or Claude-written | `.claude/agents/` (project), `~/.claude/agents/` (user), plugin `agents/` |
| **CrewAI** | `role` + `goal` + `backstory` — three prose fields | Python object or YAML | In-code |
| **Custom GPTs** | One sentence of description | **Two lanes:** conversational builder drafts name/instructions/starters from a sentence; Configure tab edits fields directly | Hosted |
| **Cursor** | A `.mdc` rule file | `/Generate Cursor Rules` turned a conversation into a reusable rule | `.cursor/rules/` |
| **OpenHands** | A "micro-agent" — a specialised prompt reusing the generalist agent's implementation | Prompt file, explicitly designed to "lower the barrier to agent development" | Repo, shareable |

**The convention is unambiguous: two fields.** A name, and a description of *when to use this*. Claude Code's docs are explicit — `description` means "When Claude should delegate to this subagent," and "Claude uses each subagent's description to decide when to delegate." The description is not documentation. It is the routing key.

### Two retreats worth learning from

**Claude Code removed its own agent-creation wizard.** The `/agents` command used to walk a user through setup interactively and could draft an agent from a description. As of v2.1.198 it no longer opens the wizard — running it prints a reminder to ask Claude or edit `.claude/agents/` directly. A bespoke interactive builder was maintained, then dropped in favour of "ask the model to write the file."

**Cursor removed `/Generate Cursor Rules`.** The command is gone from the product; the community maintains a custom-command workaround.

Both point the same way: **do not build a wizard UI. Build a command that asks the model to write a file, then let the file be the truth.** Aether already does exactly this for skills — `/ant-skill-create` is Oracle-powered generation from a description. The agent equivalent should be the same shape, not a new interaction model.

### How these systems stop a bad agent from degrading everything

This is where the research is richest, and where the transferable engineering lives:

| Guard | System | Mechanism | Transferable to Aether? |
|-------|--------|-----------|-------------------------|
| **Refuse to launch on unresolvable config** | Claude Code | If nothing in `tools` resolves to a real tool, the agent **fails to launch with an error naming the bad entries**. Before v2.1.208 it launched with zero tools and returned confusing empty results — an explicit regression they fixed | **Yes, directly.** Validate a user agent's caste/tools/skills references at creation *and* at dispatch |
| **Name-collision precedence + a doctor** | Claude Code | Nearest-to-cwd definition wins across nested project dirs; `/doctor` reports same-directory duplicates and proposes renaming | **Yes.** Aether has `/ant-patrol` and `aether recover` as the natural homes |
| **Tool allowlists** | Claude Code, CrewAI | `tools` / `disallowedTools`; a bad agent wastes a turn instead of doing damage | **Yes.** Aether has permission profiles in the dispatch contract |
| **Bounded turns per agent** | Claude Code (`maxTurns`), CrewAI (`max_iter`) | A badly-written agent burns a *bounded* amount | **Yes** |
| **Delegation allowlist** | CrewAI (`allowed_agents`), Claude Code (`Agent(agent_type)`) | Constrains who a manager may delegate to | **Yes**, and it is how a user agent is kept out of the recursive path by default |
| **Roster discipline** | Community consensus | Keep the set small — see below | **Yes**, as a hard cap |

### The roster-size finding (flag as LOW-MEDIUM confidence)

Multiple independent practitioner sources converge on the same claim: **too many agents with overlapping descriptions makes routing *worse*, not better.** Reported specifics:

- Recommended set size: **4–8 specialised agents**, "add more only when you have a clear, distinct need"
- "Install fifteen agents with fuzzy, overlapping descriptions and the router either picks the wrong one or, worse, picks none and does it inline"
- "A zoo of 100 agents triggers less reliably than a sharp set of 10"
- "Ten sharp descriptions beat a hundred vague ones every time"
- Separately reported: auto-selection of custom agents is already unreliable — Claude frequently handles a task inline rather than delegating even when a matching agent exists

**Confidence caveat:** this is blog and practitioner consensus, not a controlled study, and no official source states it. But it is consistent across sources with no dissenting voice found, and the mechanism is plausible: description-based routing is nearest-neighbour matching over prose, and adding near-duplicate neighbours degrades it.

**Aether's exposure is unusually high**, because it already ships **27 castes**. Adding user castes to a 27-agent roster is adding to an already-crowded namespace. If the practitioner consensus is right, an unbounded "add your own specialist" feature makes the Queen's caste selection *worse* — the exact opposite of "Intelligent Orchestration."

**Recommendation:** cap the user roster at a small number (3–5 custom castes), and when the cap is hit, ask which existing one to replace rather than silently allowing growth. Make the cap a stated product opinion, not a limitation.

### Non-technical owner angle

The owner should describe the specialist in a sentence — "I want someone who checks my copy reads like a human, not a robot" — and get back a named caste with an emoji, a colour, and a plain-English summary of when it will be called. They should never see YAML. But they must see the *routing description* the system generated, in plain English, and be able to say "no, only call it when I'm changing user-facing text" — because that sentence is the only thing that determines whether the agent ever gets used.

### Aether dependencies and current state

| Dependency | State | Notes |
|-----------|-------|-------|
| `agent-create` / `caste-create` command | **DOES NOT EXIST** | No CLI surface for user-authored agents. Entirely new. |
| Agent definition files | **EXISTS, THREE MIRRORS** | `.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml` are canonical platform sources per `CLAUDE.md`. A user agent must either land in all three or be explicitly scoped to one. **Recommend scoping to the primary platform, not triple-writing.** |
| `cmd/codex_visuals.go` — `casteColorMap`, `casteEmojiMap`, `casteLabelMap` | **HARDCODED GO MAPS** | A user-added caste renders with no emoji, no colour, and no label unless a fallback path is added. This is a small but real blocker for "your agent looks like a first-class colony member." |
| `cmd/caste_relevance.go` | **EXISTS** | Scores castes by role/flow with a threshold. A user caste needs a score profile or it will never be selected — the CrewAI/Claude Code routing problem in Aether's own terms. |
| `cmd/queen_spawn_budget.go` — `queenBuildSafetyRequiredCastes` | **EXISTS** | Safety castes bypass the worker cap. A user caste must **never** be able to enter this set. |
| `aether publish` / `aether update` manifest tracking | **EXISTS** | Already proven for skills: "user-created or user-modified skills are never overwritten during `aether update`." The same manifest discipline must cover user agents from day one, or the first `aether update --force` deletes them. |
| Pheromone sanitization (v2.0) | **EXISTS** | XML tag rejection, angle-bracket escaping, shell-injection blocking, LLM-instruction-override rejection, 500-char cap. **This is the exact validation layer a user-authored agent prompt needs**, and it already exists and is tested. |

---

## Area 3: User-Added Skills

### The three delivery models observed

| Model | Example | Strength | Cost |
|-------|---------|----------|------|
| **Marketplace / plugin install** | Claude Code `/plugin` from a marketplace; `anthropics/skills` repo; third-party directories | Discovery, one command to install | Trust. Third-party marketplaces now sell **security scanning** as their differentiator (one advertises an "automated 8-point security scan before it goes live"). That is the market pricing in the fact that unvetted skills are dangerous |
| **File drop** | Unzip a `SKILL.md` folder into the skills directory | Zero infrastructure, works offline | No discovery, no validation |
| **Generate from description** | Cursor's `/Generate Cursor Rules`; Claude drafting a skill; Aether's `/ant-skill-create` | Non-technical accessible, produces something tailored | Generated artefacts are often vague; quality unpoliced |

The Claude Skills format itself is minimal: a folder with a `SKILL.md` containing YAML frontmatter and instructions, using progressive disclosure — name and description always loaded, body loaded on demand.

### Aether's position: this area is largely already built

This is the most important finding for scoping. Per `CLAUDE.md`, Aether already has:

- **Generate-from-description:** `/ant-skill-create`, Oracle-powered, from a plain description
- **File drop:** `~/.aether/skills/domain/` (36+ user domain skills already present on this machine)
- **The full matching pipeline:** `skill-index` → `skill-detect` → `skill-match` → `skill-inject`, with a dedicated 8K character budget independent of the colony-prime token budget
- **Update safety:** manifest-based tracking; user skills are never overwritten by `aether update`
- **Comparison:** `skill-diff` against the shipped version
- **Frontmatter parsing and validation:** `skill-parse-frontmatter`
- **Cross-colony transport already exists in shape:** `/ant-export-signals` / `/ant-import-signals` with XML exchange modules

**Aether has two of the three delivery models and the whole matching layer.** The honest categorisation is: user-added skills are **table stakes that are already met**. The remaining gaps are narrow and unglamorous:

1. **Discovery** — the owner cannot see what skills exist, which matched last build, or which have never matched. `skill-list` exists; a "which of my skills is actually being used" view does not.
2. **Validation at creation** — nothing verifies a generated skill's `detect` patterns actually match anything in the repo. A skill that never matches is indistinguishable from a skill that does not exist, and this project has a documented history of exactly that failure class.
3. **Removal** — no clean uninstall path was found.

**Do not build a marketplace.** See anti-features.

### Non-technical owner angle

"Add that capability here" is already `/ant-skill-create "<description>"`. What is missing is the confirmation loop: after creating a skill, the owner should be told *which of their files it will trigger on*, and if the answer is "none," the creation should say so rather than reporting success. That single check converts a silent-failure feature into a trustworthy one, and it maps exactly onto the Definition of Done rule.

---

## Area 4: Making Orchestration Decisions Visible

### What comparable systems show

| System | Explanation level | Format | Audience |
|--------|------------------|--------|----------|
| **Claude Code** | **Silent on *why*.** Shows only *what* | One transcript row per delegation: `code-improver (Suggest code improvements)`; a live `/tasks` list; per-agent colour; permission prompts naming the asking agent | Developer, glanceable |
| **CrewAI** | **Full raw reasoning** via `verbose=True` | Unstructured model output printed to the console | Developer debugging, unreadable at scale |
| **LangSmith / Langfuse / Arize / Braintrust** | **Complete post-hoc trace** | Interactive trace tree: every LLM call, tool use and sub-agent delegation with payload, token breakdown and USD cost; cost attribution per agent | Ops engineer, after the fact |
| **Devin** | **Live activity, budget-framed** | TUI streaming the agent's terminal/editor/browser at <50ms latency; ACU consumption dashboard; per-session ACU caps | Buyer watching spend |
| **OpenHands** | **Live GUI with intervention** | Visualise agent behaviour, intervene manually, co-work in real time | Practitioner |

### The convention: three tiers, and nobody does the middle one well

Every system found sits at one of three levels, and the gap in the market is the middle:

- **Tier 1 — one line, always, no asking.** *Who* was picked and *what for*. Universal. Non-negotiable. Claude Code, Devin, OpenHands all do this.
- **Tier 2 — one clause of *why*, deterministic.** **Essentially nobody does this.** Claude Code shows nothing. CrewAI dumps raw chain-of-thought. The observability platforms make you leave the tool and open a web UI. This is the differentiator.
- **Tier 3 — full drill-down on request.** Scores, thresholds, the trace tree. LangSmith-class. Belongs behind a command, never in the default output.

**The right Tier 2 is a runtime-computed reason string, not model reasoning.** CrewAI's `verbose=True` proves the failure mode: it is long, non-deterministic, unreadable, and — critically for this project — **untestable**. A deterministic reason string can be asserted by a test, which is the difference between a feature and a claim under this project's Definition of Done.

### Aether is one render call away from the differentiator

The reasoning already exists as deterministic strings and is already computed on every build. It is simply never shown to a human:

- `cmd/caste_relevance.go:168` builds `"Score %d >= threshold %d for %s flow"` and `:171` builds `"%s is always required for %s flow"`, stored on `CasteRelevance.Rationale`
- `cmd/queen_spawn_budget.go` computes a `Reason` per budget decision, and `applyBuildDepthToBudget` **appends to it** — producing strings like `"..., capped by light verification depth"` and `"..., raised by heavy verification depth"`
- `queenSpawnBudgetDecision` carries `{Caste, Score, Required, Selected, Rationale}` for *every* caste, selected or not
- `cmd/codex_dispatch_contract.go:550` carries the rationale forward — **into the JSON manifest**

A grep for `Rationale` across `cmd/*.go` finds it rendered in human tables for **gates** (`cmd/gate.go:1425`, `:1472`) and for **Oracle alternatives** (`cmd/oracle_loop.go:2828`) — but not for caste selection. The pattern is established in the codebase; caste selection was simply never wired to it.

That makes "explain the Queen's team choice" a **rendering and wiring** task over data that already exists, not a new reasoning system. It is the cheapest high-value item in this milestone.

### The right shape for Aether

```
── Dispatch ──
The Queen picked a team of 5 for this phase (production mode, high risk).

  🔨 Builder    Mason-67     Implement token accounting     required for production flow
  👁️  Watcher    Vigil-12     Verify builder claims          required for production flow
  🛡️  Gatekeeper Ward-41      Security scan                  phase name contains "auth"
  🔬 Probe      Delve-88     Coverage analysis              score 7 >= threshold 5
  📊 Auditor    Ledger-03    Quality gate                   raised by heavy verification depth

  Not called: Chaos (score 2, below threshold), Archaeologist (score 1, below threshold)
              → run /ant-build --why for the full scoring table
```

Three things matter here. The reason is a **short clause**, not a sentence. The **not-called list is shown**, because "why didn't it use X" is the question a non-technical owner actually asks. And the drill-down is a **command**, not more output.

### Non-technical owner angle

The owner does not want reasoning; they want reassurance that the choice was not random. One clause per worker delivers that. The reason strings must be written in plain English at the source — `"score 7 >= threshold 5"` is developer output and should be phrased as `"strong match for this phase's work"` in the human render while the numeric form stays available under `--why`. This is the same two-layer pattern the project already uses in `CLAUDE.md` (technical detail plus a "for dummies" layer).

---

## Feature Landscape

### Table Stakes (users expect these; missing = product feels incomplete or unsafe)

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|--------------|------------|--------------|-------|
| **Enforced spawn depth limit (default 2)** | Every comparable system bounds delegation; CrewAI's documented failure without it is infinite recursion | **LOW** | `spawn-can-spawn` (stub), `SpawnEntry.Depth` (exists) | The data model exists. `cmd/spawn.go:170` currently returns `can_spawn: true` unconditionally and has no live caller. Must be a real check with a real caller and a test that fails when unenforced |
| **Run-total worker counter with a hard stop** | Depth alone does not bound cost; wide-and-shallow is the real spend. Devin bounds only this | **LOW-MEDIUM** | `spawn-tree-active` (exists), `spawn-can-spawn-swarm` counting logic (exists, generalise it), token measurement (exists) | The single most important safety feature for a non-technical owner |
| **Graceful degradation at the limit** | Claude Code withholds the tool; LangGraph raises an error and the community works around it | **LOW** | Worker brief composition | Withhold the delegation instruction from the brief; record a "delegation withheld" event. Never surface as an error |
| **Live delegation tree with parent attribution** | Universal across Claude Code `/tasks`, Devin TUI, OpenHands GUI | **MEDIUM** | `SpawnEntry{ParentName, Depth}` (exists), caste identity render (exists), TS host swarm dashboard (exists) | Indent by depth. Aether's caste identity is already richer than any comparable tool's |
| **Stop a single running worker** | Claude Code `x` in `/tasks`; without it the only recourse is killing the run (AutoGen's documented failure) | **MEDIUM** | Worker lifecycle: heartbeats, process groups, PID tracking (all exist per v1.13) | Per-worker, from a list. The plumbing is already there |
| **One line per dispatch: who and what** | Every system does this | **LOW** | Already largely present in stage-marker output | |
| **Only `name` + `description` required to author an agent** | Claude Code's floor; CrewAI's is three prose fields | **LOW** | New `agent-create` command | Two fields. Resist adding more required ones |
| **User agents/skills survive `aether update`** | Baseline expectation; breaking it destroys trust permanently | **LOW** | Manifest tracking (exists and proven for skills) | Extend the existing skill manifest discipline to agents from day one |
| **Validation refuses to launch a broken agent** | Claude Code fixed exactly this in v2.1.208 — before, a bad `tools` list launched a zero-tool agent that returned confusing results | **LOW-MEDIUM** | Pheromone sanitization layer (exists), caste registry | Validate at creation *and* at dispatch |
| **Add a skill from a description** | Custom GPTs, Cursor, Claude all offer generate-from-description | **DONE** | `/ant-skill-create` | Already shipped. Gap is post-creation validation, not creation |
| **Add a skill by file drop** | Universal | **DONE** | `~/.aether/skills/domain/` | Already shipped |

### Differentiators (competitive advantage; align with "Aether should feel alive and truthful at runtime")

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| **Deterministic one-clause "why" per selected caste** | **Nobody does this well.** Claude Code shows nothing, CrewAI dumps raw reasoning, observability platforms make you leave the tool | **LOW** | `caste_relevance.go` Rationale (exists), `queen_spawn_budget.go` Reason (exists), `codex_dispatch_contract.go:550` (already carries it into the manifest) | **Highest value-to-cost item in the milestone.** The strings are computed on every build and thrown away. `gate.go:1425` already renders Rationale in a human table — copy that pattern |
| **The "not called" list** | The question a non-technical owner actually asks is "why didn't it use the security one?" No comparable tool answers it | **LOW** | `queenSpawnBudgetDecision{Selected: false}` already computed for every caste | The rejected decisions are already in memory and already discarded |
| **Depleting budget counter in the default output** | Devin's core cost-control UX, brought into a terminal-native coding agent | **LOW-MEDIUM** | Run-total counter, existing progress render | "14 of 20 workers used this run" |
| **Nested worker blockers surface named** | Claude Code shipped this in v2.1.186 after previously auto-denying silently — a regression they explicitly fixed | **MEDIUM** | Blocker flags (exist, never trimmed from context per token budget), spawn tree parent chain | Aether's blocker system already treats blockers as never-trimmable. Extend to attribution |
| **Post-creation skill/agent reality check** | "This skill's detect patterns match 0 files in this repo" — converts silent failure into honest feedback | **LOW-MEDIUM** | `skill-detect` (exists), `skill-match` (exists) | This is `CLAUDE.md`'s Definition of Done applied to a user-facing feature |
| **Roster cap with a replace prompt** | A stated product opinion against agent sprawl, backed by the practitioner consensus that more agents route worse | **LOW** | `agent-create`, caste registry | Turns a limitation into a differentiator: "Aether keeps your team sharp" |
| **`/ant-why` drill-down** | Tier 3 on demand, without a web UI or a second product | **LOW-MEDIUM** | All rationale data (exists) | Full scoring table, thresholds, budget math |
| **Recursive delegation gated by Queen judgement, not user config** | Every comparable system exposes depth as a knob. Aether's owner will not turn a knob | **MEDIUM** | Queen-owned depth precedent (exists), `AETHER_HIVE_POLICY` single-switch precedent (exists) | Follow the verification-depth model exactly: Queen decides, flag is an advanced override |

### Anti-Features (sound good; comparable tools regret shipping them)

| Feature | Why Requested | Why Problematic — with evidence | Alternative |
|---------|---------------|--------------------------------|-------------|
| **Deep nesting (4–5+ levels)** | "Let the colony organise itself" | **Claude Code shipped depth 5, then dropped the default to 1, then settled on 3 — three changes across v2.1.216–v2.1.219.** A public record of regretting a deep default | Default 2, ceiling 3, Queen-chosen |
| **Depth as the only bound** | It feels like the natural limit | A 2-deep 8-wide tree is 72 workers. AutoGen's and CrewAI's real-world cost blowups are *loops and fan-out*, not depth. Devin bounds only spend and ignores depth entirely | Pair depth with a run-total counter |
| **Erroring at the limit** | It is the obvious implementation | LangGraph's `GraphRecursionError` is a crash the community writes parent-side wrappers to catch and degrade. Claude Code instead withholds the tool and the agent finishes the work | Withhold the capability, record an event, continue |
| **User-configurable depth flags** | Power users want control | The owner of this project is non-technical and will not type `--max-spawn-depth`. Shipping the knob means shipping the *documentation* of the knob, and this project has a documented history of docs describing behaviour that never ran | Queen decides; one env var as an escape hatch, fail-safe on unrecognised values like `AETHER_HIVE_POLICY` |
| **An interactive agent-creation wizard** | It seems friendliest for a non-technical user | **Claude Code built `/agents` as an interactive wizard and removed it in v2.1.198**, replacing it with "ask Claude or edit the file." **Cursor removed `/Generate Cursor Rules`** similarly. Two surfaces to maintain for one artefact | One command that asks the model to write the file. Then the file is the truth |
| **An agent or skill marketplace** | Discovery, community, network effects | Third-party skill marketplaces now advertise **automated security scanning** as their headline feature — the market has already priced in that unvetted skill content is a prompt-injection vector executed with the user's tools. Aether is single-maintainer. A marketplace is a moderation and trust obligation, permanently | Use the existing XML exchange path (`/ant-export-signals` / `/ant-import-signals` shape) for explicit, reviewed, manual sharing |
| **Unlimited user castes** | "Let me add as many specialists as I want" | Consistent practitioner consensus (LOW-MEDIUM confidence, no dissent found): 4–8 agents is the sweet spot; overlapping descriptions cause the router to pick wrong or pick none. **Aether already ships 27 castes** — the namespace is crowded before the user adds one | Hard cap at 3–5 custom castes with a replace prompt |
| **Raw LLM reasoning as the explanation** | "Show me why it decided that" | CrewAI's `verbose=True` is the reference implementation and it is unreadable, non-deterministic, and — decisively for this project — **untestable**. A claim you cannot assert is a claim that silently rots | Deterministic runtime-computed reason strings, asserted by named tests |
| **A per-agent cost/token dashboard** | "I want to see where the money goes" | This is LangSmith/Langfuse/Arize territory — a separate product class with its own storage, UI, and retention model. Building a lesser version is a maintenance liability, and `.planning/PROJECT.md` already lists "Ledger web UI" as out of scope for the same reason | One depleting counter in the default output; `/ant-why` for the breakdown |
| **User castes joining the required-safety set** | "My reviewer should always run" | `queenBuildSafetyRequiredCastes` bypasses the worker cap entirely by design. A user caste in that set could make every phase unboundedly expensive and would defeat the light-depth guarantee documented in `CLAUDE.md` | User castes are always optional specialists, never required ones. Enforce with a test |
| **Recursive delegation during verification/review** | "The reviewer should get help" | Claude Code's explicit guidance is to omit `Agent` from a reviewer's tools to keep it read-only. Review must be bounded, or a failing gate can trigger a spawn cascade — and this project already has documented loop-safety machinery (circuit breaker, cycle detection) that exists precisely because of past cascades | Delegation allowed in build flows only. Watcher/Gatekeeper/Auditor/Probe never delegate |
| **Auto-promoting a generated agent into the shipped roster** | "It worked well, make it standard" | `.planning/PROJECT.md` already rejects the analogous "auto finding-to-pheromone promotion" because the mapping "requires judgment, not automation." Same reasoning | Keep user castes user-scoped. Manual promotion only |
| **Writing user agents to all three platform mirrors** | Parity | `.claude/agents/ant/`, `.opencode/agents/`, `.codex/agents/*.toml` are three formats with three publish paths. Triple-writing a user artefact triples the drift surface, and `CLAUDE.md`'s platform policy already designates Codex as secondary/best-effort | Scope user agents to the platform they were created on; document the limit honestly |

---

## Feature Dependencies

```
[Enforced depth limit]
    └──requires──> [spawn-can-spawn made real + given a live caller]
                       └──requires──> [SpawnEntry.Depth]  ← EXISTS

[Run-total worker counter]
    └──requires──> [spawn-tree-active counting]  ← EXISTS
    └──generalises──> [spawn-can-spawn-swarm budget logic]  ← EXISTS

[Graceful degradation at limit]
    └──requires──> [Enforced depth limit]
    └──requires──> [Worker brief composition]  ← EXISTS

[Live delegation tree]
    └──requires──> [SpawnEntry.ParentName]  ← EXISTS
    └──requires──> [caste identity render]   ← EXISTS
    └──enhances──> [Stop a single worker]

[Stop a single worker]
    └──requires──> [worker heartbeats / PID tracking]  ← EXISTS (v1.13)
    └──requires──> [Live delegation tree]  (need a list to select from)

[One-clause "why" per caste]
    └──requires──> [caste_relevance.Rationale]      ← EXISTS, discarded
    └──requires──> [queenSpawnBudget.Reason]        ← EXISTS, discarded
    └──pattern from──> [gate.go Rationale table]    ← EXISTS
    └──enables──> ["Not called" list]
                       └──enables──> [/ant-why drill-down]

[User-authored agent]
    └──requires──> [agent-create command]           ← DOES NOT EXIST
    └──requires──> [caste visual fallback]          ← casteColorMap/EmojiMap/LabelMap are hardcoded
    └──requires──> [caste_relevance score profile]  ← or it will never be selected
    └──requires──> [update manifest tracking]       ← EXISTS for skills, extend
    └──requires──> [prompt sanitization]            ← EXISTS (pheromone v2.0), reuse
    └──conflicts with──> [queenBuildSafetyRequiredCastes]  ← must be excluded by test

[User agent + Recursive delegation]
    └──CONFLICT──> a user agent must NOT be able to delegate by default
                   (CrewAI: allow_delegation=False default;
                    Claude Code: omit Agent from tools)

[Skill reality check]
    └──requires──> [skill-detect]  ← EXISTS
    └──requires──> [skill-match]   ← EXISTS
```

### Dependency notes

- **The delegation tree is 80% built.** `SpawnEntry` records parent, caste, task, depth, status and timestamps; `spawn-tree-load`, `spawn-tree-active` and `spawn-tree-depth` all read it. What is missing is enforcement and rendering, not data.
- **The "why" is 95% built.** The rationale strings are computed on every build, carried into `codex_dispatch_contract.go`, written to the manifest, and never shown to a human. The rendering pattern already exists two files away in `gate.go`.
- **User agents conflict with recursive delegation.** A user-authored agent that can spawn is a user-authored agent that can spend without bound. Both CrewAI (delegation off by default) and Claude Code (omit `Agent` from `tools`) default to no. Aether should too.
- **User castes conflict with the safety-required set.** `queenBuildSafetyRequiredCastes` deliberately bypasses the worker cap. Admitting user castes there would silently break the documented light/standard/heavy guarantees that `TestBuildWorkerCapHonoursVerificationDepth` protects.
- **Skills are done; agents are not.** Any roadmap that treats "user skills" and "user agents" as one phase will overbuild the skill half and underbuild the agent half.

---

## MVP Definition

### Launch With (v1.26)

- [ ] **Real depth enforcement** — `spawn-can-spawn` consults an actual maximum, has a live runtime caller, and a named test fails when the limit is unenforced. *Essential: the current stub is the project's signature failure pattern and would ship recursion with no brake.*
- [ ] **Run-total worker counter with a hard stop** — *Essential: it is the bound that actually maps to cost, and the one a non-technical owner can read.*
- [ ] **Graceful degradation at the limit** — the brief omits the delegation instruction; an event records it. *Essential: hitting a safety limit must not look like a crash.*
- [ ] **Live delegation tree, indented by depth, attributed to parent** — *Essential: without it, nesting is invisible and therefore frightening.*
- [ ] **One-clause "why" per selected caste, plus the not-called list** — *Essential and cheap: the data exists, the render pattern exists, and it is the milestone's stated goal.*
- [ ] **`/ant-agent-create "<description>"`** writing a single validated, platform-scoped agent file — *Essential: this is the user-extensibility feature.*
- [ ] **Creation-time validation** — reject unresolvable castes/tools/skills, reject duplicate names, apply the existing prompt sanitization. *Essential: Claude Code shipped the unvalidated version and had to fix it.*
- [ ] **Roster cap with a replace prompt** — *Essential: shipping unbounded user castes into a 27-caste namespace degrades the Queen's routing, which is the opposite of the milestone goal.*
- [ ] **User agents survive `aether update`** — *Essential: one clobbering incident permanently destroys trust in the feature.*

### Add After Validation (v1.26.x)

- [ ] **Stop a single running worker by name** — *Trigger: once nested trees are actually being produced and someone wants to kill one branch.*
- [ ] **`/ant-why` full scoring drill-down** — *Trigger: once the one-clause version is shipped and the owner asks a follow-up question it cannot answer.*
- [ ] **Caste visual fallback (emoji/colour/label) for user castes** — *Trigger: first user caste that renders bare.*
- [ ] **Skill and agent reality check** ("matches 0 files in this repo") — *Trigger: first generated artefact that never fires.*
- [ ] **Named blocker attribution from nested workers** — *Trigger: first anonymous blocker from depth 2.*
- [ ] **Agent/skill export-import via the existing XML exchange path** — *Trigger: a second colony that wants the same specialist.*

### Future Consideration (v2+)

- [ ] **Marketplace or shared registry** — *Defer: it is a permanent moderation and security obligation for a single-maintainer project, and the market's own vendors treat security scanning as the hard part.*
- [ ] **Per-agent cost/token dashboard** — *Defer: separate product class (LangSmith/Langfuse/Arize); `.planning/PROJECT.md` already scopes out the analogous ledger UI.*
- [ ] **Depth 3+ / user-tunable delegation policy** — *Defer: only once depth 2 has demonstrably been the binding constraint on real work.*
- [ ] **Cross-platform user agents (all three mirrors)** — *Defer: triples drift surface; Codex is secondary by policy.*
- [ ] **User castes in the required-safety set** — *Never. Explicitly out of scope.*

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| One-clause "why" + not-called list | HIGH | **LOW** (data exists, pattern exists) | **P1** |
| Real depth enforcement | HIGH | LOW | **P1** |
| Run-total counter + hard stop | HIGH | LOW-MEDIUM | **P1** |
| Graceful degradation at limit | MEDIUM | LOW | **P1** |
| Live delegation tree | HIGH | MEDIUM | **P1** |
| `/ant-agent-create` | HIGH | MEDIUM | **P1** |
| Creation-time validation | MEDIUM (HIGH if skipped) | LOW-MEDIUM | **P1** |
| Roster cap + replace prompt | MEDIUM | LOW | **P1** |
| User agents survive update | HIGH (invisible until broken) | LOW | **P1** |
| Depleting budget counter in output | HIGH | LOW-MEDIUM | **P2** |
| Stop a single worker | MEDIUM | MEDIUM | **P2** |
| `/ant-why` drill-down | MEDIUM | LOW-MEDIUM | **P2** |
| Caste visual fallback | LOW-MEDIUM | LOW | **P2** |
| Skill/agent reality check | MEDIUM | LOW-MEDIUM | **P2** |
| Named nested blockers | MEDIUM | MEDIUM | **P2** |
| Export/import agents & skills | LOW-MEDIUM | MEDIUM | **P3** |
| Marketplace | LOW (negative on trust) | HIGH + ongoing | **P3 / avoid** |
| Cost dashboard | LOW | HIGH | **P3 / avoid** |

---

## Competitor Feature Analysis

| Feature | Claude Code | CrewAI | LangGraph / AutoGen | Devin | **Aether's approach** |
|---------|-------------|--------|---------------------|-------|----------------------|
| **Recursive delegation** | On, depth 3, tool withheld at limit | Off by default, `allow_delegation` | Unbounded depth; step/round caps only | Not exposed | **On, Queen-chosen, depth 2 (ceiling 3), instruction withheld at limit** |
| **Second bound** | 20 concurrent + `maxTurns` | `max_iter`, `max_rpm` | `recursion_limit` 25 / `max_round` | ACU budget per session | **Run-total worker counter, visible and depleting** |
| **Limit behaviour** | Degrade silently | Iteration stop | **Crash** (`GraphRecursionError`) | Session stop | **Degrade, log an event, keep going** |
| **Live view** | `/tasks` panel, per-agent colour, `x` to stop | Console `verbose` | None built in | TUI streaming terminal/editor/browser | **Indented spawn tree with caste identity + depleting counter** |
| **Agent authoring minimum** | `name` + `description` | `role`+`goal`+`backstory` | Code | N/A | **`name` + `description`, generated from one sentence** |
| **Authoring surface** | File; wizard **removed** v2.1.198 | Code/YAML | Code | N/A | **One command that writes one file — no wizard** |
| **Bad-agent guard** | Refuses to launch on unresolvable tools; `/doctor` finds duplicates | `allowed_agents`, `max_iter` | N/A | N/A | **Validate at creation + dispatch; reuse pheromone sanitization; roster cap** |
| **Skill authoring** | `SKILL.md`, plugin marketplace, file drop | N/A | N/A | N/A | **Already shipped** (`/ant-skill-create` + file drop + manifest safety) — add a reality check |
| **Explains *why* an agent was picked** | **No** | Raw reasoning dump | No | No | **One deterministic clause per caste + not-called list — the differentiator** |

---

## Confidence and Contradictions

| Claim | Confidence | Basis |
|-------|------------|-------|
| Claude Code depth default 3, `CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH`, tool withheld at limit, version history 5→1→3 | **HIGH** | Official docs, fetched directly |
| Claude Code frontmatter: only `name` + `description` required | **HIGH** | Official docs, fetched directly |
| Claude Code `/agents` wizard removed in v2.1.198 | **HIGH** | Official docs, fetched directly |
| Claude Code concurrent limit 20; **no session total limit** | **HIGH** | Official docs, fetched directly |
| CrewAI delegation off by default; unbounded `max_iter` recursion failure | **MEDIUM** | Multiple secondary sources agreeing, referencing official CrewAI docs |
| LangGraph `recursion_limit` 25 → `GraphRecursionError`; community degrades in parent | **MEDIUM** | LangChain forum + GitHub issues |
| AutoGen non-termination and cost-runaway failure mode | **MEDIUM** | Multiple secondary sources; consistent |
| OpenHands `AgentDelegateAction` + micro-agents | **MEDIUM-LOW** | ICLR 2025 paper + secondary; official SDK delegation doc URL returned 404 |
| Devin ACU per-session caps + consumption dashboard + TUI | **MEDIUM** | Pricing aggregators + docs.devin.ai references; not fetched directly |
| **4–8 agent sweet spot; overlapping descriptions degrade routing** | **LOW-MEDIUM** | Practitioner blog consensus only. No official source, no controlled study, but no dissenting source found either. **Flag for validation before it becomes a hard requirement.** |
| `spawn-can-spawn` is a non-enforcing stub with no live caller | **HIGH** | Read `cmd/spawn.go:170-181`; grepped all callers — only two unloaded playbooks |
| Caste rationale strings computed and discarded | **HIGH** | Read `cmd/caste_relevance.go:168-180`, `cmd/queen_spawn_budget.go`, `cmd/codex_dispatch_contract.go:550`; grepped `Rationale` renders across `cmd/*.go` |

### Contradiction to record

A secondary source ([digitalapplied.com](https://www.digitalapplied.com/blog/claude-code-subagent-depth-limits-budget-caps-2026)) states Claude Code enforces **three** limits including a "session total 200." The official documentation explicitly contradicts this: *"There's no limit on the total number of subagents Claude can spawn over a session."* **Official docs win.** The relevance for Aether is that a session-total cap is a *gap* in the reference implementation, not a copied feature — which strengthens the case for Aether's run-total counter as a differentiator rather than table stakes.

---

## Sources

**Official documentation (HIGH confidence):**
- [Create custom subagents — Claude Code Docs](https://code.claude.com/docs/en/sub-agents)
- [Extend Claude with skills — Claude Code Docs](https://code.claude.com/docs/en/skills)
- [Hierarchical Process — CrewAI](https://docs.crewai.com/en/learn/hierarchical-process)
- [LangGraph Multi-Agent Supervisor reference](https://reference.langchain.com/python/langgraph-supervisor)
- [Selector Group Chat — AutoGen](https://microsoft.github.io/autogen/dev//user-guide/agentchat-user-guide/selector-group-chat.html)
- [Billing — Devin Docs](https://docs.devin.ai/admin/billing)
- [anthropics/skills — public Agent Skills repository](https://github.com/anthropics/skills)

**Issue trackers and forums (MEDIUM confidence):**
- [Hierarchical process delegation fails — crewAI #4783](https://github.com/crewAIInc/crewAI/issues/4783)
- [feat: hierarchical agent delegation with allowed_agents — crewAI PR #2068](https://github.com/crewAIInc/crewAI/pull/2068)
- [How to use manager_agent and hierarchical mode correctly — crewAI Discussion #1220](https://github.com/crewAIInc/crewAI/discussions/1220)
- [Gracefully handling GraphRecursionError from subgraphs — LangChain Forum](https://forum.langchain.com/t/gracefully-handling-graphrecursionerror-from-subgraphs-so-the-parent-agent-can-retry-or-degrade/2018)
- [Agent infinite looping until recursion limit — langgraph #6731](https://github.com/langchain-ai/langgraph/issues/6731)
- [Subagents have no Agent/Task tool — claude-code #60763](https://github.com/anthropics/claude-code/issues/60763)

**Research papers (MEDIUM confidence):**
- [OpenHands: An Open Platform for AI Software Developers as Generalist Agents (ICLR 2025)](https://arxiv.org/pdf/2407.16741)
- [The OpenHands Software Agent SDK](https://arxiv.org/html/2511.03690v1)

**Practitioner sources (LOW-MEDIUM confidence — roster sizing and routing degradation):**
- [I Built 100 Claude Code Subagents. These Are The 12 That Actually Earn Their Context](https://dev.to/suraj_khaitan_f893c243958/i-built-100-claude-code-subagents-these-are-the-12-that-actually-earn-their-context-10nn)
- [Navigating Claude Code: Subagents Done Right — HackerNoon](https://hackernoon.com/navigating-claude-code-subagents-done-right)
- [Claude Code Subagents and Multi-Agent Orchestration Guide](https://hidekazu-konishi.com/entry/claude_code_subagents_and_orchestration_guide.html)
- [Fix AutoGen GroupChat Not Terminating](https://mechanicai.dev/blog/fix-autogen-groupchat-not-terminating.php)
- [Agent Observability for AI Coding — Augment Code](https://www.augmentcode.com/guides/agent-observability-for-ai-coding)
- [AI Agent Observability with Langfuse](https://langfuse.com/blog/2024-07-ai-agent-observability-with-langfuse)
- [How to create a custom GPT — Zapier](https://zapier.com/blog/custom-chatgpt/)
- [Cursor rules vs custom modes — Cursor Forum](https://forum.cursor.com/t/cursor-rules-vs-custom-modes/91023)
- [How to Install Skills in Claude Code (2026)](https://www.agensi.io/learn/how-to-install-skills-claude-code)
- [Devin usage limits, quotas & pricing — AI Limit Watcher](https://ailimit.watch/tools/devin/)

**Aether source read directly (HIGH confidence):**
- `cmd/spawn.go` — `spawn-can-spawn` stub, spawn tree read commands
- `cmd/internal_cmds.go` — `spawn-can-spawn-swarm` working budget check
- `cmd/queen_spawn_budget.go` — wave-width budget, depth binding, decision rationale
- `cmd/caste_relevance.go` — caste scoring and rationale string construction
- `cmd/codex_dispatch_contract.go` — rationale carried into manifest
- `cmd/codex_visuals.go` — hardcoded caste colour/emoji/label maps
- `pkg/agent/spawn_tree.go` — `SpawnEntry` structure
- `.aether/docs/command-playbooks/build-wave.md`, `build-full.md` — the only (unloaded) callers of `spawn-can-spawn`

---
*Feature research for: multi-agent coding framework — intelligent orchestration*
*Researched: 2026-08-08*
