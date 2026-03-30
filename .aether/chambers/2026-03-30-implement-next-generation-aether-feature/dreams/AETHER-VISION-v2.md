# Aether v2.0 — The Living Hive

> Your projects are territories. Your colonies are workers. Your hive remembers everything.

---

## The Premise

You install Aether once. From that moment, every project you touch becomes part of your hive. The Queen — your Queen — lives at `~/.aether/` and grows smarter with every colony she runs. She learns your patterns, remembers your mistakes, knows your preferences, and carries that intelligence into every new project you start.

This isn't a per-repo tool anymore. It's a living system.

When you start a new WordPress project, the Queen already knows how you build WordPress sites — because she built the last three. When you hit a bug in your Electron app, the hive remembers a pattern from your API service that solved something similar. When you change how you like commit messages, every future colony adjusts.

**Aether is not a memory system. It's not an MCP server. It's not another AI wrapper.**

Aether is an ant colony — and ant colonies are the most successful civilization model on Earth because they solve problems through collective intelligence, persistent chemical trails, and specialized labor. That metaphor isn't decoration. It's the architecture.

---

## 1. The Hive — Your Global Home

### What It Is

`~/.aether/` transforms from a read-only hub into the living center of your development world. This is where the Queen resides, where cross-colony wisdom accumulates, and where your identity as a developer persists.

### Structure

```
~/.aether/
├── QUEEN.md              ← Accumulated wisdom (philosophies, patterns, redirects)
├── USER.md               ← Who you are (preferences, communication style, expertise)
├── hive/
│   ├── wisdom.json       ← Promoted instincts from all colonies (abstracted, domain-tagged)
│   ├── signals.json      ← Cross-repo pheromone trails (validated, provenance-tracked)
│   ├── trails.json       ← Semantic trail map (which domains connect to which repos)
│   └── chronicle.json    ← Colony history (sealed colonies, outcomes, learnings timeline)
├── registry.json         ← Enhanced: domain tags, colony history, active status per repo
├── eternal/
│   └── memory.json       ← Legacy eternal store (migrated to hive/ over time)
├── system/               ← Read-only system files (as today)
└── QUEEN-IDENTITY.md     ← How the Queen presents herself (separate from wisdom)
```

### How It Works

**The wisdom flow:**
1. You work in a repo. The colony learns things — instincts, patterns, mistakes.
2. When you seal the colony (`/ant:seal`), the Queen reviews what was learned.
3. High-confidence instincts get **abstracted** — repo-specific details stripped away — and promoted to the hive.
4. Next time you start a colony in ANY repo, the Queen brings relevant hive wisdom into the new workers' context.

**Example:**
- Colony in `my-wordpress-site` learns: "ACF sync requires Polylang activation first"
- Promoted to hive as: "CMS plugin sync requires dependency activation first" (tagged: `wordpress`, `cms`)
- Next WordPress project: Queen injects this wisdom automatically
- Non-WordPress project: Queen skips it (domain doesn't match)

### What Makes This Different

OpenClaw stores memory in flat markdown files with semantic search. Cursor plugins use MCP servers. Windsurf auto-learns in isolation per workspace.

Aether's hive is a **living colony** — wisdom doesn't just get stored, it gets *promoted* through a trust pipeline. Observations become instincts become wisdom become cross-repo intelligence. Each step requires validation. The Queen doesn't remember everything — she curates what matters and lets the rest decay.

It's not a database. It's an ecosystem.

---

## 2. Cross-Colony Intelligence — The Hive Mind

### The Problem Today

Every colony starts from zero. You've sealed 24 colonies across 13 repos, but each new `/ant:init` is a blank slate. The Queen has amnesia.

### The Solution: Stigmergic Intelligence

In real ant colonies, intelligence lives in the environment — chemical trails left by previous ants. No ant "remembers" the path to food. They follow trails that billions of footsteps have reinforced.

Aether works the same way:

**Trail System:**
- Every colony leaves trails in the hive — instincts tagged with domain, confidence, and provenance
- New colonies **sniff** the hive for relevant trails before starting work
- Trails that get reinforced (same insight from multiple repos) grow stronger
- Trails that nobody follows decay and fade
- The environment itself becomes intelligent

**Domain Scoping:**
- Repos get domain tags (auto-inferred during `/ant:colonize`, or user-specified)
- Hive wisdom filters by domain relevance: WordPress wisdom doesn't leak into your Electron app
- Universal wisdom (cross-cutting patterns) has no domain tag — applies everywhere
- Domains form a trail hierarchy: `web.wordpress.ecommerce`, `desktop.electron`, `api.node`

**The Abstraction Layer:**
Before ANY instinct reaches the hive, it gets transformed:
- Repo-specific details stripped (no file paths, no project names)
- Generalized to the pattern level ("always check dependency order" not "run wp acf sync after polylang")
- Provenance metadata attached (source repo, colony goal, confidence, validation count)
- Content sanitized for prompt-injection patterns

This isn't "sync your data to the cloud." It's "distill your experience into wisdom and make it available when it's relevant."

### Data Model

**Hive Wisdom Entry:**
```json
{
  "id": "hw_abc123",
  "content": "When integrating third-party plugins, verify dependency activation order before syncing configuration",
  "type": "pattern",
  "confidence": 0.85,
  "validated_count": 3,
  "domain_tags": ["wordpress", "cms", "plugins"],
  "source_repos": ["my-wp-site", "client-store"],
  "created_at": "2026-03-20T10:00:00Z",
  "last_accessed": "2026-03-25T14:00:00Z",
  "access_count": 7,
  "promoted_by": "seal",
  "supersedes": null
}
```

**Conflict Resolution:**
When two colonies promote contradictory wisdom:
1. The Queen arbitrates during seal (she sees both, picks the better one)
2. Fallback: highest `confidence × validated_count` wins
3. Contradictions are tracked, not deleted — you can always ask "what did we used to believe?"
4. Temporal supersession: new wisdom invalidates old, but preserves history

---

## 3. The Queen's Evolution — From Orchestrator to Partner

### Today's Queen

A stateless task coordinator. She picks workflow patterns, spawns workers, manages phases. Each colony she runs, she forgets at the end. She's competent but amnesiac.

### Tomorrow's Queen

The Queen becomes the persistent intelligence layer of your development life.

**She knows you:**
- How you communicate (plain English, no jargon — from USER.md)
- Your domain expertise (WordPress expert, new to React — from accumulated colony history)
- Your decision patterns (you prefer speed over polish, you always want tests — from decisions across colonies)
- Your schedule patterns (you work in bursts, prefer small PRs — from build history)

**She knows your world:**
- Every repo she's touched (from the enhanced registry)
- The domain relationships between your projects (trail map)
- What worked and what failed across all colonies (chronicle)
- The current state of any active colony (status dashboard)

**She brings context:**
When you start a new colony, the Queen doesn't just create an empty `COLONY_STATE.json`. She:
1. Checks the hive for domain-relevant wisdom
2. Loads your USER.md preferences
3. Injects relevant cross-repo patterns into worker prompts
4. Warns about pitfalls she's seen in similar projects
5. Suggests a plan structure based on what's worked before in this domain

### The Three-Way Separation

Following patterns validated by OpenClaw, Hermes, and academic research:

| Layer | File | What It Stores | Scope |
|-------|------|---------------|-------|
| **Identity** | `QUEEN-IDENTITY.md` | How the Queen presents herself, personality, communication style | Global |
| **Wisdom** | `QUEEN.md` | Accumulated technical knowledge, patterns, philosophies, redirects | Global + per-repo overlay |
| **User Model** | `USER.md` | Who the user is, preferences, expertise, decision patterns | Global |

These are separate because they change at different rates and for different reasons. You might update your preferences without changing the Queen's wisdom. A repo might add local wisdom without affecting identity.

### Colony-Prime v2 — The Intelligence Engine

Colony-prime is the function that assembles what every worker sees. Today it has 7 sections. Tomorrow:

```
--- QUEEN IDENTITY (Who I Am) ---          ← NEW: personality, communication style
--- QUEEN WISDOM (What I Know) ---         ← Existing: philosophies, patterns, redirects
--- HIVE INTELLIGENCE (Cross-Repo) ---     ← NEW: domain-matched wisdom from the hive
--- USER CONTEXT (Who You Are) ---         ← NEW: user preferences, expertise level
--- CONTEXT CAPSULE (Current Colony) ---   ← Existing: goal, phase, state
--- PHASE LEARNINGS ---                    ← Existing: what this colony learned so far
--- KEY DECISIONS ---                      ← Existing: decisions in this colony
--- ACTIVE SIGNALS ---                     ← Enhanced: with trail namespacing, JIT injection
--- INSTINCTS ---                          ← Enhanced: with semantic matching
```

**Token Budget (Concrete):**

| Section | Full Mode | Compact Mode | Priority |
|---------|-----------|-------------|----------|
| Queen Identity | 200 tokens | 100 tokens | Fixed |
| Queen Wisdom | 400 tokens | 200 tokens | High |
| Hive Intelligence | 300 tokens | 150 tokens | Medium |
| User Context | 200 tokens | 100 tokens | Fixed |
| Context Capsule | 250 tokens | 125 tokens | High |
| Phase Learnings | 200 tokens | 100 tokens | Medium |
| Key Decisions | 150 tokens | 75 tokens | Low |
| Active Signals | 500 tokens | 250 tokens | Critical (REDIRECTs never cut) |
| Instincts | 200 tokens | 100 tokens | Medium |
| **Total** | **2400 tokens** | **1200 tokens** | — |

REDIRECTs are never truncated (hard constraints). When budget is tight, FEEDBACK signals and low-confidence instincts are first to be trimmed. Budget adapts: compact mode activates automatically when the context window is under 50% remaining.

---

## 4. Pheromone 2.0 — The Trail System

### What Changes

The pheromone system evolves from flat signals to a hierarchical trail network.

**Trail Namespacing:**
Signals get organized into dot-notation trails:
```
security.auth          ← Authentication patterns
security.injection     ← Injection prevention
testing.coverage       ← Coverage requirements
testing.e2e            ← End-to-end test patterns
performance.database   ← Database optimization
architecture.patterns  ← Structural patterns
```

**Intelligent Decay:**
Linear decay → Exponential with access boost:
- Signals you keep referencing decay slower (use it or lose it)
- Different signal types have different half-lives
- REDIRECT: ~58 day half-life (constraints persist longer)
- FOCUS: ~30 day half-life (attention is more temporary)
- FEEDBACK: ~87 day half-life (calibrations are durable)

**Merge Strategies:**
When a new signal matches an existing one:
- **FEEDBACK**: Reinforce (boost strength, reset decay timer)
- **REDIRECT**: Replace with supersession (old signal gets `superseded_by`, preserves audit trail)
- **FOCUS**: Keep highest effective strength

**Content Deduplication:**
SHA-256 hash check before creating any new signal. No more duplicate pheromones.

**Temporal Supersession:**
Old signals aren't deleted — they're invalidated with a `valid_until` timestamp and a pointer to what replaced them. You can always trace the evolution of constraints: "why did this REDIRECT change?"

**JIT Injection:**
Not every worker needs every signal:
- REDIRECTs always upfront (hard constraints, never skip)
- FOCUS/FEEDBACK injected only when the signal's trail matches the worker's task domain
- Reduces token waste, sharpens worker attention

### New Signal Type: PREFERENCE

A fifth signal type for user-level preferences:
- Persisted at hub level (not per-repo)
- Lower priority than all task signals
- Captures communication style, tooling preferences, workflow patterns
- Auto-learned from colony interactions over time

---

## 5. User Identity — USER.md

### What It Captures

```markdown
# User Profile

## Communication
- Prefers plain English over technical jargon
- Doesn't read code — explain in terms of user experience
- Wants momentum over perfection
- Short, direct responses

## Expertise
- Deep: WordPress, WooCommerce, business strategy
- Moderate: API design, deployment workflows
- Learning: React, desktop apps, audio development

## Decision Patterns
- Values speed of iteration over comprehensive planning
- Prefers fixing forward over reverting
- Likes being shown working things, not told about progress
- Trusts technical recommendations but wants alternatives presented

## Workflow
- Works in focused bursts
- Prefers single-PR approach for refactors
- Runs colonies through full lifecycle (init → seal)
```

### How It Learns

The Queen doesn't ask you to fill this out. She builds it over time through the same instinct pipeline that powers everything else:

**Observation Sources:**
1. **Decision tracking** — Every `/ant:council` choice, every option selected during builds. "User chose Express over Fastify" (3 times across 3 API projects → preference captured)
2. **Correction detection** — When you say "don't show me diffs" or "stop summarizing", the correction is logged as a PREFERENCE observation with `domain: communication`
3. **Domain inference** — Registry tracks which project types you touch. 5 WordPress repos and 2 Electron repos → expertise map auto-builds from colony history
4. **Workflow pattern extraction** — Colony chronicle data reveals patterns: average phase count, build-to-seal time, how often you use `/ant:swarm` vs `/ant:build`, whether you plan first or dive in

**Promotion Pipeline:**
- Observations accumulate in `~/.aether/hive/user-observations.json` (same content-hash dedup as learning-observations)
- Threshold: 3 occurrences across 2+ repos → auto-promote to USER.md
- The Queen reviews during seal: "Based on this colony, I noticed you prefer X. Adding to your profile."
- User can always edit USER.md directly — manual entries have highest confidence

**Character Cap:**
USER.md is capped at 1500 characters (~500 tokens). This forces curation: when the cap is hit, lowest-confidence entries get evicted. Following Hermes' proven pattern — constraints force quality over accumulation.

### Privacy

USER.md lives only at `~/.aether/` — your machine, your control. Never distributed, never shared, never synced to any service. You can edit it directly, delete entries, or wipe it entirely. The Queen respects boundaries.

### What USER.md Is NOT

- Not a personality profile for marketing
- Not a tracking system — no usage analytics, no telemetry
- Not shared with AI providers — it's injected into prompts locally
- Not required — Aether works fine without it, just less personalized

---

## 6. Visual Experience — The Living World

### Design Philosophy

Aether should feel like peering into a living ant colony. Not a dashboard — a world. The terminal is a window into the colony's activity.

### Immediate Improvements (Bash-Native)

**In-conversation display** — Already designed, needs building:
- Compact worker status lines during builds: `🔨 Hammer-42 [building] auth.ts, utils.ts ████░░ 67%`
- Real-time completion: `✓ Hammer-42 complete (4 files, 12 tools, 0 blockers)`
- Phase progress bar at bottom of build output
- Consistent format across all commands

**Enhanced /ant:status:**
```
╔══════════════════════════════════════════════════════╗
║  🐜 AETHER COLONY — my-project                       ║
╠══════════════════════════════════════════════════════╣
║  Goal: Build authentication system                    ║
║  Phase: 3/5 ████████░░ 60%                           ║
║  Workers: 3 active (2 builders, 1 watcher)            ║
╠══════════════════════════════════════════════════════╣
║  SIGNALS          │ MEMORY           │ HEALTH         ║
║  🔴 2 REDIRECT    │ 12 instincts     │ ████ Good      ║
║  🟡 3 FOCUS       │ 4 learnings      │ 0 blockers     ║
║  🟢 1 FEEDBACK    │ 2 pending        │ 0 midden       ║
╠══════════════════════════════════════════════════════╣
║  HIVE: 47 wisdom │ 13 repos │ Queen: 24 colonies      ║
╚══════════════════════════════════════════════════════╝
```

**Worker activity animation:**
Status phrases that rotate while workers are running:
- Builder: "excavating...", "forging...", "constructing...", "reinforcing..."
- Watcher: "scanning...", "inspecting...", "validating...", "certifying..."
- Scout: "scouting...", "mapping...", "investigating...", "discovering..."

### Future: Companion TUI

A typed companion process (Go/BubbleTea or TypeScript/Ink) that provides:
- Real-time colony dashboard (btop-style panel layout)
- Subagent tracing tree (Queen → Builder → tool calls)
- Live token/cost tracking per worker
- Pheromone trail visualization
- Colony timeline with phase history
- Keyboard navigation and mouse support

This is a separate project — the core system remains bash, and the TUI is an optional viewer that reads the same state files.

---

## 7. The Chronicle — Colony History and Pattern Recognition

### What It Is

The chronicle is the hive's historical record — every sealed colony, every outcome, every lesson learned. Not raw data, but curated history that informs future decisions.

### Data Model

**Chronicle Entry (per sealed colony):**
```json
{
  "id": "colony_abc123",
  "repo_path": "/path/to/repo",
  "goal": "Build authentication system",
  "domain_tags": ["web", "auth", "node"],
  "started_at": "2026-03-01T10:00:00Z",
  "sealed_at": "2026-03-02T16:00:00Z",
  "duration_hours": 30,
  "phases_count": 4,
  "phases_summary": [
    {"name": "Auth foundation", "tasks": 3, "outcome": "complete"},
    {"name": "Session management", "tasks": 2, "outcome": "complete"}
  ],
  "instincts_promoted": 3,
  "wisdom_promoted_to_hive": 1,
  "midden_entries": 2,
  "milestone": "Crowned Anthill",
  "learnings_summary": "Session tokens should expire in 7 days max. Always test refresh flow before login flow.",
  "learnings_summary_source": "queen_seal_synthesis"
}
```

The `learnings_summary` is written by the Queen during `/ant:seal` — she synthesizes the colony's instincts, decisions, and midden entries into a one-line takeaway. This is the same synthesis she does today when reviewing learnings, just persisted to the chronicle.

### What The Queen Does With It

- **Plan calibration**: "Your API projects average 4 phases. This goal looks like a 3-phase project."
- **Risk anticipation**: "Last time you worked on auth, the refresh token flow caused 2 midden entries. Want me to add a dedicated testing phase for that?"
- **Progress context**: "You've sealed 24 colonies. Your completion rate is 92%. The ones that stalled were all >6 phases."
- **Domain expertise depth**: "You've done 5 WordPress projects (expert), 2 Electron projects (moderate), 1 React project (learning)."

---

## 8. The Heartbeat — Colony Health Monitoring

### Concept (Inspired by OpenClaw)

A periodic health check that runs between builds. Not a build — a pulse.

**What it checks:**
- Pheromone staleness (any signals past expiry but still marked active?)
- Flag accumulation (unresolved blockers piling up?)
- Midden growth (failure rate increasing?)
- Hive wisdom relevance (any hive entries older than 90 days with zero access?)
- Token budget health (is colony-prime injection growing toward the cap?)

**How it works:**
```bash
# In .aether/HEARTBEAT.md
## Colony Health Checks
- [ ] Run pheromone-expire to deactivate stale signals
- [ ] Check midden for recurring failure patterns (auto-REDIRECT if 3+)
- [ ] Verify hive wisdom access counts (decay unused entries)
- [ ] Report any blockers older than 7 days
```

**When it runs:**
- Automatically at colony init (health check before starting work)
- Optionally via `/ant:heartbeat` for manual check
- During `/ant:continue` (lightweight check between phases)
- NOT during builds (don't interrupt workers)

**Output:**
```
🫀 Colony Heartbeat
├── Pheromones: 3 active, 1 expired (cleaned)
├── Midden: 0 recurring patterns
├── Hive: 2 entries decayed below threshold
├── Flags: 0 unresolved blockers
└── Health: ████ Good
```

If something needs attention, the heartbeat emits a FOCUS signal automatically. If something is critically wrong (e.g., 5+ unresolved blockers), it emits a REDIRECT.

---

## 9. Tool-Agnostic Architecture — Beyond Claude Code

### The Reality

Aether today works with Claude Code and OpenCode. But the vision is bigger: any AI coding tool should be able to tap into the hive.

### Three Layers of Tool Agnosticism

**Layer 1: File-Based State (Already Tool-Agnostic)**
Everything in `~/.aether/` is JSON and Markdown files. Any tool that can read files can read the hive. COLONY_STATE.json, pheromones.json, QUEEN.md — these are plain text. No proprietary format, no binary blobs, no lock-in.

**Layer 2: Command Interface (Per-Tool Adapters)**
Today: `.claude/commands/ant/` (Claude Code) + `.opencode/commands/ant/` (OpenCode).
Tomorrow: the command layer is thin — it translates tool-specific invocation into calls to `aether-utils.sh` which does the real work. Adding a new tool means writing adapter commands, not reimplementing logic.

The pattern:
```
User types /ant:build 1
→ Tool-specific command file parses args
→ Calls aether-utils.sh subcommands
→ aether-utils.sh does the work (tool-agnostic)
→ Returns structured output
→ Tool-specific command formats for display
```

**Layer 3: MCP Server (Future — Tool-Agnostic API)**
Expose hive intelligence as an MCP server:
```
aether-memory://wisdom?domain=wordpress    → Relevant hive wisdom
aether-memory://signals?trail=security.*   → Active pheromone trails
aether-memory://user                       → User profile
aether-memory://chronicle?domain=api       → Colony history for API projects
```

Any MCP-compatible tool (Cursor, Cline, future tools) gets access to the hive without needing Aether's command layer. The MCP server reads the same files — it's a window into the hive, not a separate system.

### What This Means

You can use Cursor for quick edits (it reads hive wisdom via MCP), Claude Code for full colony builds (it runs the full lifecycle), and OpenCode for alternative workflows — and they all draw from the same hive intelligence. The Queen's wisdom follows you across tools.

---

## 10. Security Model — Trust Before Intelligence

### The Rule

Security must be in place BEFORE cross-repo features go live. A poisoned signal in one repo must not be able to spread to all repos.

### Layered Defense

**Layer 1: Content Sanitization (pheromone-write)**
- Shell injection patterns (existing)
- Prompt injection patterns (NEW): detect "ignore previous instructions", "system prompt override", and similar patterns
- 500-character limit (existing)
- Content hash for deduplication (NEW)

**Layer 2: Abstraction Before Promotion**
- Instincts promoted to the hive are TRANSFORMED by the Queen
- Repo-specific details stripped (file paths, project names, credentials references)
- Generalized to pattern level
- The Queen uses judgment — she won't promote something that looks suspicious

**Layer 3: Provenance Tracking**
- Every hive entry tracks: source repo, colony goal, promoting agent, confidence level, validation count
- Entries with single-source provenance are flagged as lower trust
- Multi-repo validation (same pattern from 3+ repos) gets trust boost

**Layer 4: Token Budgets**
- Colony-prime has a total token cap
- Per-section budgets prevent any single source from dominating
- Hive intelligence gets a separate budget (30% of total, configurable)
- Budget adapts: tighter context window = higher quality bar for admission

**Layer 5: Memory Governance**
- `/ant:memory-audit` — Review what's in the hive, what's been injected, provenance
- Force-forget: user can explicitly remove any hive entry
- Auto-decay: unused wisdom fades over time (exponential decay with access boost)
- Entry caps per section (max 100 hive entries, max 20 QUEEN.md entries per category)

### The Eternal Promotion Fix

Currently, nearly all REDIRECT and FOCUS signals auto-qualify for eternal promotion (the 0.8 threshold checks original strength, not decayed strength). Fix: check `effective_strength` (decayed) at expiry time. Only signals that have maintained their strength through actual use deserve eternal promotion.

---

## 11. What Makes This Different

### The Competitive Landscape

| System | What It Does | Memory Model | Cross-Repo | Orchestration | Local-First |
|--------|-------------|-------------|------------|---------------|-------------|
| **OpenClaw** | Personal AI assistant | Dual-layer markdown + semantic search | No (per-workspace) | Single agent + cron | Yes |
| **Cursor Memory** | Plugin ecosystem | MCP servers (Hindsight, ContextForge) | Via MCP | No orchestration | Varies |
| **Windsurf** | Auto-learning IDE | Workspace-local auto-memories | No | Single agent | Yes |
| **Cline** | Session memory bank | 6 structured markdown files | No | No orchestration | Yes |
| **Mem0/Letta** | Memory infrastructure | API-based memory CRUD | Via API | Framework-agnostic | No |
| **Aether v2** | **Living ant colony** | **Trust-pipeline promotion + hive** | **Yes (via hive)** | **22 specialized agents** | **Yes** |

### What Only Aether Has

**1. Organized labor, not just memory.**
Every other system is "AI + memory." Aether is "AI + organization." 22 specialized workers (builders, watchers, scouts, chaos testers), parallel wave execution, caste-based task assignment, quality gates, failure tracking. The memory is just one part of a complete colony.

**2. Trust-pipeline knowledge promotion.**
Nothing else has a multi-stage trust pipeline: observation → threshold validation → deduplication → abstraction → promotion → decay → reinforcement. OpenClaw stores everything it's told. Mem0 extracts memories via LLM. Aether's knowledge earns its way from colony floor to hive through repeated validation.

**3. Stigmergic intelligence.**
The environment IS the memory. Pheromone trails are chemical traces left by workers that influence future workers — exactly how real ant colonies work. No central knowledge base consulted by agents. The workspace itself carries information. This is architecturally unique.

**4. The metaphor is the mechanism.**
Other systems use metaphors as marketing ("your AI copilot", "your AI teammate"). In Aether, the ant colony metaphor IS the architecture: pheromones ARE how signals flow, the Queen IS the orchestrator, the midden IS where failures go, castes ARE how labor specialization works. You can explain the system to a child using the metaphor, and you'll have described the actual code.

**5. Human-in-the-loop by design.**
Build and advance are deliberately separate (`/ant:build` then `/ant:continue`). The user sees results and decides. This isn't a limitation — it's designed for a non-technical founder who wants control over direction without needing to understand code. Other systems either run autonomously (risky) or require technical oversight (excluding non-developers).

### The Ant Colony Advantage

Real ant colonies solve problems that no individual ant can comprehend:
- They find shortest paths (pheromone trail optimization)
- They allocate labor dynamically (caste switching based on colony needs)
- They maintain collective memory without any central database (stigmergic intelligence)
- They scale from 50 ants to 50 million without architectural changes

Aether v2 brings all of these to software development:
- **Shortest paths**: Hive wisdom routes you away from mistakes and toward patterns that work
- **Dynamic labor**: Queen selects from 22 castes based on task type, not a fixed assignment
- **Collective memory**: The hive IS the memory — trails in the environment, not records in a database
- **Scaling**: Works for one repo or fifty, same architecture, same commands

---

## The User Experience

### First Install (Cold Start)
```
$ npm install -g aether-colony
$ cd my-first-project
$ aether init "Build my website"

🐜 Welcome to Aether.

   This is your first colony. The hive is empty — no wisdom yet,
   no patterns, no preferences. That's normal.

   As you build, the colony learns. Seal it when you're done,
   and the Queen will carry what she learned into every future project.

   Let's get started. What are we building?
```

The cold start is honest: no fake wisdom, no hallucinated patterns. The Queen is useful from day one because she has 22 specialized workers, parallel builds, and a structured lifecycle — not because she pretends to know you. The hive intelligence comes later, organically.

### Day 1 (Experienced User)
```
$ cd my-new-project
$ aether init

🐜 Aether Colony initialized.
   Queen detected 13 repos in your hive.
   Domain auto-detected: web.react (based on package.json)

   Loading hive wisdom...
   ├── 4 patterns matched (web, react)
   ├── 2 redirects active (from similar projects)
   └── 1 preference loaded (your coding style)

   The Queen is ready. What are we building?
```

### Day 30 (Fifth Project)
```
$ cd another-project
$ aether init "Build the API"

🐜 Colony initialized.
   The Queen remembers:
   ├── You've built 4 APIs before
   ├── Pattern: you prefer Express over Fastify
   ├── Redirect: always set up error handling middleware first
   ├── Your projects average 3.2 phases
   └── Suggested plan structure based on past API colonies

   Shall I generate a plan? (/ant:plan)
```

### Day 365 (Twentieth Sealed Colony)
```
$ cd brand-new-saas
$ aether init "Build the billing system"

🐜 Colony initialized.
   The Queen remembers:
   ├── 47 wisdom entries in the hive (12 universal, 35 domain-scoped)
   ├── 6 patterns matched (api, payments, node)
   ├── Warning: payment integrations had 3 midden entries across 2 past colonies
   │   └── Auto-REDIRECT: "Always sandbox payment APIs before production testing"
   ├── Your expertise: deep in API, moderate in payments
   └── Suggested: 4-phase plan (your API projects average 3.8 phases)

   🫀 Heartbeat: hive healthy. 3 entries decayed since last colony.
```

The hive self-manages. Entries that nobody accesses decay below threshold and get evicted. Entries that keep getting reinforced across projects get stronger. The hive doesn't grow forever — it converges on your actual patterns, like a real ant colony's trail network converges on the most efficient paths.

---

## Implementation Approach

### Phase 0: Foundation (Before Cross-Repo)
- Prompt-injection sanitization in pheromone-write
- Colony-prime token budget with per-section caps
- Fix eternal promotion threshold (use decayed strength)
- Pheromone content deduplication (SHA-256)
- **Done when:** Pheromone content can't inject prompt instructions. Colony-prime has a measurable token budget. All tests pass.

### Phase 1: The Queen Gets a Memory
- Separate USER.md from QUEEN.md (identity/wisdom/user model split)
- Create QUEEN-IDENTITY.md at hub level
- Wire hive reads into colony-prime (8th section: HIVE INTELLIGENCE)
- Enhance registry with domain tags and colony history
- Entry caps and decay on QUEEN.md sections
- **Done when:** Colony-prime injects hive wisdom into worker prompts. USER.md exists at `~/.aether/` and is editable. Registry stores domain tags per repo.

### Phase 2: The Trail System
- Trail-based pheromone namespacing (dot-notation)
- Exponential decay with access boost
- Merge strategies (reinforce/replace/max)
- JIT signal injection (REDIRECTs upfront, FOCUS/FEEDBACK by domain match)
- Temporal supersession (invalidate, don't delete)
- Heartbeat system for colony health
- **Done when:** Signals have trail namespaces. Decay is exponential. Duplicate signals merge instead of accumulating. `/ant:heartbeat` runs and reports health.

### Phase 3: The Hive Awakens
- Hive wisdom data model and promotion flow
- Abstraction layer (Queen transforms before promoting)
- Domain-scoped retrieval (filter by trail match)
- Conflict resolution (confidence-weighted, Queen-arbitrated)
- Chronicle system (colony history and outcomes)
- **Done when:** Sealing a colony promotes abstracted instincts to `~/.aether/hive/`. Starting a new colony loads domain-relevant hive wisdom. Chronicle records colony outcomes.

### Phase 4: Intelligence
- Semantic retrieval for instincts (SQLite + FTS5, zero external deps)
- Suggest-analyze v2 (LLM-powered pattern detection alongside grep rules)
- Auto-learning for USER.md (decision pattern extraction from colony interactions)
- Cross-repo pattern aggregation (multi-colony validation boosting)
- **Done when:** Instinct retrieval uses hybrid text+semantic search. USER.md auto-populates from observed patterns across 3+ colonies.

### Phase 5: The Visual World
- In-conversation display implementation (already fully spec'd)
- Enhanced /ant:status with hive overview panel
- Worker activity animations
- Token/cost tracking per build
- Color theme support
- **Done when:** Builds show real-time worker status. `/ant:status` shows hive statistics. Users can see what they're spending per colony.

### Future: Tool-Agnostic Layer
- MCP server exposing hive intelligence to Cursor, Cline, OpenCode
- Companion TUI for rich real-time visualization (Go/BubbleTea)
- Memory module extraction (standalone library, following memsearch pattern)
- Community-contributed colony templates (colony marketplace)
- **Done when:** A Cursor user can query Aether hive wisdom via MCP without installing Aether's command layer.

---

## The Bash Question

Aether's core is a 10,500-line bash script. That's a feature, not a bug — bash is universally available, requires no compilation, and handles file I/O and process orchestration well. But cross-repo security needs type safety (distinguishing sanitized vs unsanitized content), and rich TUI needs a real rendering framework.

**The answer: bash stays as orchestration, a typed companion handles the boundaries.**

```
┌─────────────────────────────────────────────┐
│  aether-utils.sh (bash)                      │
│  ├── Colony lifecycle (init/plan/build/seal)  │
│  ├── State management (load/unload/lock)      │
│  ├── Pheromone operations (write/read/decay)  │
│  ├── Colony-prime assembly                    │
│  └── All existing 110 subcommands            │
│                                              │
│  aether-core (TypeScript, future)             │
│  ├── Content sanitization (prompt injection)  │
│  ├── Hive wisdom promotion (abstraction)      │
│  ├── Semantic search (SQLite + FTS5)          │
│  ├── MCP server (hive access API)             │
│  └── Token budget enforcement                 │
└─────────────────────────────────────────────┘
```

The bash layer calls the typed companion for security-sensitive operations. The typed companion is optional — Aether degrades gracefully to bash-only mode (no semantic search, simpler sanitization, no MCP). This means existing installations don't break.

---

## Skill Self-Generation — Colonies That Teach

### The Concept

When a colony repeatedly does something that works, it should be able to create a reusable command for it. Today, slash commands are hand-written. Tomorrow:

**Instinct → Custom Command Pipeline:**
1. An instinct reaches high confidence (0.9+) and has been validated across 3+ colonies
2. The Queen recognizes it as a repeatable workflow, not just a pattern
3. She generates a draft slash command: `.claude/commands/ant/custom-{name}.md`
4. The user approves or rejects during the next seal cycle
5. Approved commands become part of their personal toolkit

**Example:**
The instinct "When deploying WordPress, always flush permalinks after plugin activation" could become `/ant:wp-deploy-check` — a custom command that runs the specific verification steps the user always needs.

**Guardrails:**
- Generated commands are drafts — never auto-activated
- Commands can only read and report, never modify (safe by default)
- Stored in the repo's `.claude/commands/ant/` or at hub level for cross-repo
- The user reviews the generated command before it goes live

This is inspired by OpenClaw's skill system but adapted to the ant metaphor: instincts that prove themselves get upgraded to permanent colony behavior — like how real ant colonies develop specialized roles over generations.

---

## Migration: v1.3 → v2.0

### The Promise

Aether v2 is additive. Nothing breaks.

### What Happens on Update

```
$ npm install -g aether-colony@2

🐜 Aether v2.0 installed.

   Migrating...
   ├── ~/.aether/hive/ created (empty — will populate as you seal colonies)
   ├── ~/.aether/USER.md created (empty — will learn your preferences over time)
   ├── ~/.aether/QUEEN-IDENTITY.md created (default identity)
   ├── Registry enhanced (domain_tags added, existing repos preserved)
   ├── Eternal memory: 24 entries → migrated to hive/wisdom.json schema
   └── All existing colony data untouched

   Your 13 repos and their colonies are exactly as you left them.
   The hive starts empty. It grows as you seal future colonies.

   Everything you used before still works. New features layer on top.
```

### Migration Details

| What | Migration | Risk |
|------|-----------|------|
| COLONY_STATE.json | No changes | None |
| pheromones.json | Schema version field added to each signal (read-time normalization for legacy signals) | None — read-time migration, no file modification |
| eternal/memory.json | Entries copied to hive/wisdom.json with new fields defaulted (domain_tags: [], source_repos: []) | None — eternal/ preserved as backup |
| registry.json | New fields added (domain_tags, colony_count, last_colony_goal) with empty defaults | None — additive |
| QUEEN.md | Preserved as-is. Entry caps only evict when cap is hit. | None |
| Slash commands | All 40 existing commands unchanged. New commands added alongside. | None |
| Agent definitions | Existing 22 agents gain optional pheromone_protocol enhancements | None — enhancements are additive |

### Rollback

`npm install -g aether-colony@1.3` restores v1.3. Hive data created by v2 is ignored by v1.3 (unknown directory). No data loss in either direction.

---

## Risks and Honest Concerns

### Things That Could Go Wrong

**1. Scope Creep — This Vision Is Huge**
Six implementation phases, a typed companion, an MCP server, semantic search, auto-learning... this is a lot. The biggest risk is trying to build it all and shipping nothing.
- **Mitigation:** Phase 0 (foundation) and Phase 1 (Queen gets memory) are the minimum viable hive. Ship those, validate, iterate. Phases 3-5 and the future layer are stretch goals.

**2. The Hive Gets Noisy**
If promotion thresholds are too low, the hive fills with low-value wisdom that dilutes worker attention. Windsurf had exactly this problem — "occasionally clings to outdated patterns."
- **Mitigation:** Conservative defaults (confidence 0.7+, validated 2+ times, across 2+ repos). Exponential decay culls unused entries. Hard cap of 100 hive entries forces quality over quantity. The Queen reviews during seal — she's not a rubber stamp.

**3. Abstraction Loses Meaning**
When the Queen strips repo-specific details to make wisdom shareable, she might strip the useful part. "ACF sync requires Polylang activation first" abstracted to "plugin sync requires dependency check" — does the abstracted version actually help?
- **Mitigation:** Both versions are preserved. The hive stores the abstracted form; the chronicle retains the original context. If hive wisdom doesn't help, the original is recoverable. Monitor hive access_count — entries that get loaded but don't improve outcomes should decay faster.

**4. Prompt Injection Amplification**
Cross-repo sharing means a poisoned signal could spread everywhere. This is the highest-severity security risk identified by the Oracle research.
- **Mitigation:** Phase 0 (security foundation) ships BEFORE any cross-repo features. Layered defense: content sanitization + abstraction + provenance tracking + token budgets + memory governance. The pipeline has five independent barriers.

**5. The Bash Foundation Creaks**
10,500 lines of bash with 110 subcommands. Adding cross-repo logic, semantic trails, exponential decay formulas, and token budgeting in bash gets fragile.
- **Mitigation:** The typed companion (aether-core in TypeScript) handles the complex new operations. Bash stays as orchestration. Graceful degradation — if the companion isn't installed, Aether falls back to simpler bash-only behavior. The transition is gradual, not a rewrite.

**6. Nobody Uses the Hive**
You build the hive, but users don't seal enough colonies to populate it, or they work in domains too different for cross-repo wisdom to help.
- **Mitigation:** The hive is a bonus, not a dependency. Aether v2 is fully functional without any hive data — the per-repo colony system remains exactly as capable as v1.3. The hive adds value when it can and stays out of the way when it can't.

---

## What This Is NOT

- **Not a cloud service** — Everything is local files. No accounts, no sync, no SaaS.
- **Not a database** — It's markdown and JSON files. You can read them, edit them, delete them.
- **Not an AI model** — Aether is prompt engineering over files. The intelligence comes from the colony structure, not a proprietary model.
- **Not a breaking change** — v2 is additive. Existing colonies keep working. New features layer on top.
- **Not a framework** — You don't rewrite your code for Aether. You use Aether to build your code.

---

*Aether v2 — Because ant colonies have been solving coordination problems for 100 million years, and they never needed a cloud service to do it.*
