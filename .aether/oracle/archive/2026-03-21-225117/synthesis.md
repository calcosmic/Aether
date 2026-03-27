# Research Synthesis

## Topic
Future vision for Aether: evolving from per-repo colony tool to cross-repo intelligent personal assistant with persistent memory, OpenClaw-inspired intelligence, enhanced pheromone system, and next-gen AI tooling patterns

## Findings by Question

### Q1: What is OpenClaw and how does it approach persistent AI intelligence? (partial — 62%)

**What OpenClaw Is:**
OpenClaw is an open-source personal AI assistant (68k+ GitHub stars) created by Peter Steinberger. It runs locally on user devices and connects to messaging channels (WhatsApp, Telegram, Slack, Discord, etc.) via a local gateway that bridges AI models with user tools.

**Dual-Layer Memory Architecture:**
- **MEMORY.md** — Long-term knowledge base storing durable facts, preferences, and decisions
- **memory/YYYY-MM-DD.md** — Daily running context (short-term work notebook)
- The model only "remembers" what gets written to disk — memory is files, not magic

**Memory Retrieval Tools:**
- `memory_search` — Semantic recall using hybrid BM25 + vector search over indexed snippets
- `memory_get` — Targeted read of specific Markdown file/line range
- Hybrid search allows finding related notes even when wording differs

**Context Compaction Protection:**
When approaching auto-compaction, OpenClaw triggers a silent agentic turn (invisible to user) reminding the model to write durable memory before context is lost.

**Skills System:**
- Self-contained folders with SKILL.md (YAML frontmatter + instructions)
- Loaded on-demand rather than embedded in every prompt (token-efficient)
- Agent can autonomously write its own skills
- ClawHub marketplace has 5,400+ skills with supply chain security concerns flagged

**Core Architecture Insight:**
Everything reduces to prompt engineering over files:
- "Autonomous behavior" = cron jobs constructing prompts
- "Persistent memory" = Markdown files prepended to prompts
- "Personality" = text file injected at prompt top

**Patterns Relevant to Aether (preliminary):**
- Dual-layer memory (daily vs durable) maps to Aether's session.json vs COLONY_STATE.json
- Semantic search over memory files could enhance Aether's instinct/learning retrieval
- Self-writing skills = potential for colonies to generate their own slash commands
- Memory preservation before compaction is a gap Aether could fill
- Skills marketplace concept could inspire community-contributed colony templates

**Memory Search Implementation (Code-Level) [NEW, iteration 12]:**
MemoryIndexManager (manager.ts) orchestrates a SQLite database with 4 tables: `chunks` (text + metadata), `chunks_vec` (sqlite-vec extension for vector similarity), `chunks_fts` (FTS5 for full-text search), and `embedding_cache` (SHA-256 dedup). Chunking targets 512 tokens (~2048 chars) with 128-token overlap at line-aware boundaries. Hybrid search uses the formula `finalScore = (vectorScore × 0.7) + (textScore × 0.3)` with 3x candidate oversampling before merge. Post-ranking includes MMR diversity re-ranking (lambda=0.7) and temporal decay with a 30-day half-life. Embedding auto-selects: OpenAI text-embedding-3-small (1536d) → Gemini gemini-embedding-004 (768d) → local GGUF embeddinggemma-300M (~600MB) → FTS-only fallback. Batch processing via OpenAI Batch API provides 50% cost reduction. Performance constants: snippet max 700 chars, batch max tokens 8000, concurrency 4 parallel requests, local embedding timeout 5 minutes. Delta-based indexing triggers at 100KB new data or 50 new messages. [S102][S103]

**Skills System Architecture (Deep Dive) [NEW, iteration 12]:**
Three-tier loading with strict precedence: bundled (lowest) < managed (`~/.openclaw/skills`) < workspace (`<workspace>/skills`, highest). Skills are snapshotted at session start for consistency; mid-session refresh occurs only on file watcher detection. Full YAML frontmatter schema: required fields (name, description) + optional (user-invocable boolean, disable-model-invocation boolean, command-dispatch, metadata.openclaw with emoji, os platform filter `["darwin","linux","win32"]`, requires.bins/anyBins/env/config, install specs for brew/node/go/uv/download). Per-skill token cost: 97 chars + XML-escaped name/description/location (~24 tokens per skill). Environment injection per agent run: read metadata → set env vars → build system prompt → restore env. Workspace skill folders are jail-checked via realpath to prevent path traversal. The skill-creator skill (itself a SKILL.md) scaffolds new skills which users then review — no native autonomous skill generation exists. [S104][S7]

**Two-Abstraction Architecture (Systems Analysis) [NEW, iteration 12]:**
Laurent Bindschaedler's analysis identifies OpenClaw's core as two composable primitives: (1) **Autonomous Invocation** — trigger → route → execute with session isolation. Multiple activation mechanisms (cron, webhooks, Gmail, voice, group mentions) funnel through a routing layer that maintains session isolation (separate thinking levels, usage tracking, model selection per session). This parallels OS process identity. (2) **Externalized Memory** — LLM context treated as cache backed by persistent disk. "Treat the LLM context as a cache and treat disk memory as the source of truth." Durable notes are written BEFORE summarization to prevent information loss. These compose familiar systems patterns (message queues, durable state, process containerization) into coherent LLM runtime behavior. [S105]

**Heartbeat System — Autonomous Monitoring [NEW, iteration 12]:**
HEARTBEAT.md is a plain checklist processed every 30 minutes (configurable) in the main session with full context access. When nothing needs attention, the agent returns HEARTBEAT_OK which OpenClaw silently suppresses. Active hours windowing (default 08:00-22:00) prevents overnight token burn. Cron handles precise schedules with session isolation options: `--session isolated` (clean-slate) or `--session main` (inject into next heartbeat). Cron supports 5/6-field syntax, timezone, stagger, and one-shot `--at` timing. Token cost scales with HEARTBEAT.md size; isolated cron jobs can override to cheaper models. [S106][S1]

**Compaction Implementation [NEW, iteration 12]:**
Trigger formula: `currentTokens >= contextWindow - reserveTokensFloor(20000) - softThresholdTokens(4000)`. Pre-flush sends two invisible prompts (user + system append) with NO_REPLY flag. One flush per compaction cycle, tracked in sessions.json. Skipped for read-only sessions. Three identifier policies: strict (preserves opaque IDs, default), off, custom. Model override available for using a more capable model for summarization. Compaction writes a compact summary entry to the session's JSONL file, keeping recent messages intact. [S107][S102]

**memsearch: Proof of Separability [NEW, iteration 12]:**
Zilliz extracted OpenClaw's memory system into memsearch — a standalone library usable by any AI agent without the OpenClaw platform. Four workflows: Watch → Index → Search → Compact. Includes a ready-made Claude Code plugin. SHA-256 content dedup, live file sync, delta-based indexing. This proves the memory architecture is separable from the platform — significant for Aether's potential to extract its learning/instinct system into a tool-agnostic module. [S108][S102]

**Concrete Adoption Patterns for Aether (Synthesized) [NEW, iteration 12]:**
1. **Pre-compaction colony flush** — Save phase status, active signals, and recent decisions to session.json before Claude Code compaction. Prevents the "where was I?" problem during long builds.
2. **Semantic instinct retrieval** — Replace domain/confidence filtering with hybrid BM25 + vector search over instincts. SQLite-based, zero external deps with local GGUF embeddings.
3. **Skill self-generation** — Colonies could generate slash commands from validated patterns (instinct → /ant:custom-command).
4. **Heartbeat for colony health** — Periodic check monitoring pheromone staleness, flag accumulation, midden growth, build cost.
5. **Token-budgeted context assembly** — OpenClaw's 20K char cap + per-skill cost accounting provides explicit budget. Aether's colony-prime has NO total cap.
6. **Memory module extraction** — Following memsearch, extract Aether's learning system into a standalone module for tool-agnostic use.

**Cross-Question Validation [SYNTHESIS, iteration 16]:**
OpenClaw's patterns are independently validated across multiple questions: (1) Hybrid BM25+vector search positioned at Tier 3 in Q4's five-tier model — above Aether's Tier 0-1 but below 4-strategy approaches [S113]. (2) SOUL.md/IDENTITY.md separation validated by Hermes' MEMORY.md/USER.md (Q5) [S70] and Collaborative Memory's private/shared partition (Q3) [S90] — three independent systems converge on identity ≠ wisdom ≠ user model. (3) 20K char workspace cap validates Q6's proposed 500-token pheromone budget [S100]. (4) Temporal decay (30-day half-life) aligns with Q6's exponential decay model [S99]. (5) memsearch separability validated by Q4's MCP memory server ecosystem showing 5 implementations following the same pattern [S114][S115][S108].

**Cross-Question Confidence Strengthening [SYNTHESIS, iteration 17]:**
Q1's remaining gaps (multi-agent memory access, ClawHub governance, performance benchmarks) are implementation details that do not undermine the key adoptable patterns. The 6 concrete adoption patterns for Aether (pre-compaction flush, semantic retrieval, skill self-generation, heartbeat, token budgeting, memory module extraction) are each independently validated by at least 2 other questions — making the practical value of Q1's research well-established even with remaining implementation unknowns.

### Q2: Aether's current architecture end-to-end (partial — 72%)

**Distribution Model:**
Aether is an npm package (aether-colony v1.1.11). On `npm install -g`, cli.js runs `setupHub()` which syncs .aether/ system files to ~/.aether/system/ (the "hub"). The hub holds read-only system files (aether-utils.sh, workers.md, templates, docs, commands, agents). Per-repo colony data lives in .aether/data/ and is excluded from npm distribution. [S10][S18]

**Colony Lifecycle:**
1. `/ant:init` — Creates COLONY_STATE.json with goal, session_id, timestamps, empty plan
2. `/ant:colonize` — Surveys existing codebase with 4 parallel scouts
3. `/ant:plan` — Generates phases with route-setter agent
4. `/ant:build N` — Queen directly spawns parallel workers in waves (not delegated to Prime)
5. `/ant:continue` — Verifies work, extracts learnings, advances to next phase
6. `/ant:seal` — Marks colony as complete (Crowned Anthill milestone)
7. `/ant:entomb` — Archives completed colony to chambers [S12][S13][S14]

**State Management:**
COLONY_STATE.json (v3.0) is the central state file:
- `goal` — Colony objective
- `state` — IDLE or EXECUTING
- `current_phase` — Phase number
- `plan.phases[]` — Phase definitions with tasks
- `memory.phase_learnings[]` — Per-phase learnings
- `memory.decisions[]` — Recorded decisions
- `memory.instincts[]` — Promoted patterns (max 30, evicts lowest confidence)
- `errors.records[]` — Error tracking
- `signals[]` — Active signals
- `events[]` — Colony event log

State is loaded/unloaded with file locking (aether-utils.sh load-state/unload-state). Atomic writes via temp file + mv prevent corruption. [S11][S9]

**Pheromone System:**
Three signal types with different priorities:
- **FOCUS** (normal) — Attracts colony attention to an area
- **REDIRECT** (high) — Hard constraint, repels workers from a pattern
- **FEEDBACK** (low) — Gentle calibration from observations

Signals stored in .aether/data/pheromones.json with rich metadata: tags with weights/categories, strength (0-1), expiry (date or "phase_end"), scope. Colony-prime injects active signals into worker prompts via `prompt_section`. Builder, Watcher, Scout agents have `pheromone_protocol` sections governing how they act on injected signals. [S15][S12]

**Learning Pipeline:**
1. Observations recorded in learning-observations.json with content hashes (SHA-256 dedup)
2. observation_count tracked per unique content hash across colonies
3. When count hits threshold (2), auto-promotion fires
4. Promoted observation becomes an instinct in COLONY_STATE.json memory.instincts[]
5. A FEEDBACK pheromone is auto-emitted from the validated learning
6. Instincts have confidence scoring, domain tagging, and application tracking [S16][S9]

**Hard Boundaries Limiting Cross-Repo Intelligence:**
1. All mutable state (.aether/data/) is strictly per-repo — no shared storage
2. The hub (~/.aether/) only holds read-only system files, never colony state
3. No registry or index of active/past colonies across repos
4. Learnings/instincts are locked to each repo's COLONY_STATE.json
5. Pheromone XML export/import exists but is manual (copy file, run command)
6. No semantic search — instinct lookup is by domain/confidence filter only
7. Session recovery checks only local session.json, unaware of other repos [S10][S12][S9]

**Build Wave Mechanics (Deep Dive) [NEW, iteration 15]:**
The build command uses split playbooks (5 files: build-prep, build-context, build-wave, build-verify, build-complete) loaded sequentially by the orchestrator `.claude/commands/ant/build.md`. Wave execution follows this precise sequence:

1. **Task Analysis (Step 5):** Queen groups phase tasks by `depends_on` field — Wave 1 = tasks with `depends_on: "none"` or `[]` (fully parallel); Wave 2 = tasks depending on Wave 1; Wave 3+ continues until all assigned.
2. **Caste Assignment:** Implementation → 🔨 Builder, Research/docs → 🔍 Scout, Testing → 👁️ Watcher (ALWAYS mandatory), Resilience → 🎲 Chaos (ALWAYS mandatory after Watcher).
3. **Worker Naming:** Each worker gets a unique name via `generate-ant-name "{caste}"` subcommand (e.g., "Hammer-42", "Vigil-17").
4. **Workflow Pattern Selection:** Queen examines phase name for keywords → selects one of 6 patterns: SPBV (default), Investigate-Fix, Deep Research, Refactor, Compliance, Documentation Sprint.
5. **Parallel Spawn:** ALL Wave 1 workers spawned in a SINGLE message with multiple Task tool calls — Claude Code runs them in parallel and blocks until all complete. Each worker gets: ant name, task spec, colony goal, archaeology context (if files being modified), midden context (recent failures), graveyard cautions (per-file), and the full `prompt_section` from colony-prime.
6. **Wave Results Processing:** As each result arrives, display completion line immediately. Failed workers trigger midden logging + memory-capture pipeline. Total wave failure halts build. Partial failure triggers 3-tier escalation: Tier 3 (Queen respawns different caste) → Tier 4 (ask user with options).
7. **Intra-Phase Midden Check:** After each wave, scan for recurring failure categories (3+ occurrences). If found, auto-emit REDIRECT pheromone mid-build (max 3 REDIRECTs per check).
8. **Sequential Waves:** Waves 2+ wait for previous wave completion before spawning. Same format as Wave 1 but with accumulated context from prior waves.

After all builder waves: Watcher (Step 5.4) → Measurer (Step 5.5.1, conditional on performance keywords) → Chaos (Step 5.6) → Synthesis (Step 5.9). The spawn-can-spawn mechanism enforces depth limits: Queen (depth 0) → workers (depth 1, max 4 children) → sub-workers (depth 2, max 2 children) → no deeper. Global cap: 10 workers per phase. [S121][S122][S123][S80]

**Colony-Prime Prompt Assembly (Deep Dive) [NEW, iteration 15]:**
The `colony-prime` subcommand (aether-utils.sh:7560-7964) is the single most important function in Aether — it assembles the unified context injection block that every worker receives. It constructs 7 sequential sections via file I/O:

1. **QUEEN Wisdom** (lines 7571-7732): Two-level loading — global `~/.aether/QUEEN.md` first, local `.aether/QUEEN.md` extends. Each file parsed via awk for 5 section headers (📜 Philosophies, 🧭 Patterns, ⚠️ Redirects, 🔧 Stack Wisdom, 🏛️ Decrees). Global and local content concatenated per-category. FAIL HARD if neither QUEEN.md exists. Output wrapped in `--- QUEEN WISDOM (Eternal Guidance) ---` delimiters.

2. **Context Capsule** (lines 7734-7740): Calls `context-capsule --compact --json` subcommand which reads COLONY_STATE.json for goal/phase/state, flags.json for blockers, pheromones.json for signals, and rolling-summary.log for recent events. Produces a bounded snapshot with configurable limits (max 8 signals, 3 decisions, 2 risks, 220 words). Determines `next_action` recommendation based on colony state.

3. **Phase Learnings** (lines 7742-7801): Extracts validated learnings from `memory.phase_learnings[]` in COLONY_STATE.json for ALL phases before current. Grouped by phase number, deduped by claim text. Max 15 entries (5 in compact mode). Output wrapped in `--- PHASE LEARNINGS ---` delimiters.

4. **Key Decisions** (lines 7803-7853): Parses CONTEXT.md "Recent Decisions" table using awk to extract decision + rationale pairs from pipe-delimited table rows. Max 5 decisions (3 compact). Output wrapped in `--- KEY DECISIONS ---` delimiters.

5. **Blocker Warnings** (lines 7855-7905): Reads flags.json via jq, filters for unresolved blockers matching current phase. Max 3 blockers (2 compact). Labeled "REDIRECT-priority" to ensure workers treat them as hard constraints. Output wrapped in `--- BLOCKER WARNINGS ---` delimiters.

6. **Rolling Summary** (lines 7907-7924): Last 5 entries from `rolling-summary.log`, formatted as `[timestamp] event_type: description`. Output wrapped in `--- RECENT ACTIVITY (Colony Narrative) ---` delimiters.

7. **Pheromone Signals** (lines 7926-7929): Calls `pheromone-prime` subcommand which reads pheromones.json, computes effective_strength with linear decay, filters active signals, sorts by priority (REDIRECT=1, FOCUS=2, FEEDBACK=3, POSITION=4), then renders instincts grouped by domain. Max 8 signals + 5 instincts (compact: 3 instincts). Output includes `--- ACTIVE SIGNALS (Colony Guidance) ---` and `--- INSTINCTS (Learned Behaviors) ---`.

The `--compact` flag halves all limits. The final prompt is JSON-escaped and returned as `prompt_section` for injection into worker prompts. **Critical gap: NO total token budget exists** — the assembled prompt size is bounded only by count caps, not token count. [S71][S94][S121][S122]

**Agent Dispatch Mechanism — The 22 Agents (Deep Dive) [NEW, iteration 15]:**
All 22 agents are defined as `.claude/agents/ant/aether-{name}.md` files with YAML frontmatter: `name`, `description`, `tools` (comma-separated tool access list), `color`, `model` (inherit = parent's model). Dispatched via Claude Code's Task tool with `subagent_type="aether-{name}"`. Each agent definition contains `<role>`, `<execution_flow>`, `<critical_rules>`, and output format sections.

**Not all 22 are spawned during a single build.** The dispatch map across commands:

| Command | Agents Spawned | Mechanism |
|---------|---------------|-----------|
| `/ant:build` | Builder (1+ per wave), Watcher (1, mandatory), Chaos (1, mandatory), Archaeologist (conditional: existing files modified), Measurer (conditional: performance keywords in phase), Ambassador (conditional: external integration keywords) | build-wave.md + build-verify.md playbooks |
| `/ant:continue` | Probe (conditional: coverage gaps), Weaver (conditional: refactoring needed), Gatekeeper (security gate), Auditor (quality gate) | continue-verify.md + continue-gates.md playbooks |
| `/ant:colonize` | 4 Surveyors (nest, provisions, disciplines, pathogens) — ALL in parallel | colonize.md |
| `/ant:plan` | Scout (2: research + gap-focused), Route-Setter (1: planning) | plan.md |
| `/ant:swarm` | Archaeologist (git history), Scout (2: pattern + web), Tracker (error analysis) — ALL in parallel | swarm.md |
| `/ant:seal` | Sage (conditional: ≥3 phases completed), Chronicler (1: documentation) | seal.md |
| `/ant:organize` | Keeper (1: hygiene report) | organize.md |

**Agents with no dedicated command trigger:** Includer (accessibility — available for Compliance workflow), Chronicler (also available for Documentation Sprint), Keeper (also available for Deep Research), Sage (also available for wisdom synthesis). These are invokable by Queen's discretion during specific workflow patterns or by the user directly.

**Fallback pattern:** Every playbook includes `# FALLBACK: If "Agent type not found", use general-purpose and inject role`. This ensures the system degrades gracefully if agent definitions are missing — the worker runs as general-purpose with the role text injected into its prompt.

**Worker prompt structure:** Each spawned worker receives: (1) identity and caste assignment, (2) task specification, (3) colony goal, (4) optional context blocks (archaeology, midden, graveyard, ambassador integration plan), (5) the `prompt_section` from colony-prime (wisdom + signals + learnings), (6) activity logging instructions, (7) output format specification (JSON). Workers return structured JSON with status, files touched, tool count, and blockers. [S121][S122][S123][S124][S125][S126][S34][S80]

**Build-to-Continue Handoff [NEW, iteration 15]:**
Build does NOT update task statuses or advance colony state. It saves a `last-build-result.json` and updates HANDOFF.md with build results. The user must explicitly run `/ant:continue` which: (1) verifies all work independently, (2) runs quality gates (Gatekeeper security scan → Auditor quality check), (3) extracts learnings and checks for wisdom promotions, (4) marks tasks as completed in COLONY_STATE.json, (5) advances `current_phase`. This separation ensures human-in-the-loop between building and advancing — the user sees results and decides whether to proceed. [S121][S123]

### Q3: Cross-repo 'hive mind' architecture (partial — 72%)

**Existing Foundation in Aether:**
The hub (~/.aether/) already has a registry.json that tracks repos with `{path, version, updated_at}` via the `registry-add` subcommand. This is currently read-only infrastructure for system file distribution, but it means the hub already knows which repos have Aether installed. This provides a natural foundation for cross-repo state without introducing new infrastructure. [S10][S9]

**MCP as Cross-Repo Knowledge Broker:**
MCP (Model Context Protocol) is the emerging standard for cross-repo awareness in 2026. Key architectural properties:
- Maintains persistent "envelopes" that accumulate context across repos rather than resetting between requests
- Repos are tagged by domain (billing, auth, infrastructure) for scoped retrieval
- 7-phase lifecycle: init → discovery → context provision → invocation → execution → response → completion
- The MCP server acts as a centralized knowledge broker with structured discovery
- Production deployments: ~2 vCPU, 4GB RAM per 100k indexed files
- OAuth 2.1 for identity, RBAC for repo access scoping [S31]

**Moderne's Unified Semantic Model:**
Moderne's multi-repo AI agent uses Lossless Semantic Trees (LSTs) to build a unified knowledge graph across all repositories simultaneously. Rather than syncing individual state files between repos, it captures structure, dependencies, and relationships in a single semantic model. Knowledge graphs built from method declarations and class definitions form an in-depth cross-repo understanding. Key insight: shared understanding via unified semantic model > shared state files. [S28]

**Memory Scoping Models from Mem0 and Letta:**
- Mem0 stores memories as atomic events with metadata for filtering by user, session, or application. A single instance serves multiple agents/populations with scoped retrieval. Framework-agnostic: works with LangChain, CrewAI, AutoGen, or custom loops.
- Letta (MemGPT) uses three-tier memory: Core (working RAM), Recall (session cache), Archival (cold storage). Agents self-edit memory through function calls during reasoning. Full agent runtime where agents "live" persistently. [S29][S32][S33]

**Entity-Fact Data Model:**
Production memory architecture uses `{subject, predicate, object, confidence, timestamp}` tuples. Key properties:
- Facts can supersede older ones (newer + higher confidence wins)
- Contradictions are tracked via "markSupersedes" rather than deleting
- Unaccessed memories decay via exponential decay formulas
- Retention with degradation rather than aggressive purging [S30]

**Three Viable Architecture Patterns for Aether:**
1. **Hub-centric (lowest friction):** Extend ~/.aether/ with a `hive/` directory containing promoted instincts, cross-repo learnings, and user preferences aggregated from all colonies. Pure file-based, no external dependencies, builds on existing registry infrastructure.
2. **MCP-server (most powerful):** Expose Aether memory as an MCP server queryable by any tool (Claude Code, Cursor, Cline). Adds runtime dependency but enables tool-agnostic memory access.
3. **Hybrid:** Hub for storage, MCP for access. Best of both worlds but most complex. [S10][S31][S29]

**Meta-Repo Pattern — Proven Cross-Repo Agent Context (NEW, iteration 10):**
A dedicated shared repository provides cross-repo AI agent context through 5 components: AGENTS.md (entry point with repo map), repos.yaml (machine-readable config with paths, build commands, version files), conventions/ (centralized standards), workflows/ (step-by-step multi-repo playbooks), and active-work/ (persistent multi-session tracking with dated progress logs, checklists, and decision rationale). Key insight: machine-readable config (YAML) is dramatically faster for agents than prose — eliminates guessing about repo structure. Active-work/ bridges flat auto-memory with actionable persistent context. Completed work moves to archive/ with full decision history for future reference. This meta-repo concept is architecturally identical to Aether's ~/.aether/ hub — Aether's registry.json already plays the repos.yaml role (tracking paths + versions) but currently lacks domain tags, build commands, and colony metadata. The meta-repo pattern validates extending the hub from read-only system files to a living shared-context surface. [S88]

**Agent-MCP: Persistent Knowledge Graph as Shared State (NEW, iteration 10):**
Agent-MCP coordinates multi-agent systems through a persistent knowledge graph that agents read (query_project_rag) and write (update_project_context), plus direct messaging (send_agent_message, broadcast_message) for real-time coordination. Task dependency management prevents conflicts by construction — work is decomposed and assigned to specialized agents who write to logically disjoint state sections. The key principle: conflict PREVENTION through task decomposition is more effective than conflict RESOLUTION after the fact. This validates Aether's current architecture where Queen assigns tasks to specific workers — but highlights the missing shared knowledge layer between colonies. [S89]

**Collaborative Memory with Dynamic Access Control (NEW, iteration 10):**
Academic research (2025) proposes a two-tier memory partition for multi-agent shared memory:
- **Private memory (ℳ^private):** Per-user/per-repo fragments with sensitive or personalized information
- **Shared memory (ℳ^shared):** Cross-context accessible fragments with immutable provenance metadata: creation timestamp, contributing user, generating agents, and accessed resources

Fragments are designated private or shared AT CREATION TIME through write policies — not promoted retroactively. LLM-based transformation (redaction, anonymization, abstraction) runs before shared storage. Access control via bipartite graph formalism: user→agent and agent→resource permission edges, checked at retrieval time. Retrieval uses vector embeddings with top-k (k=10-20) per scope.

Critical insight for Aether: memories promoted to the hub should be TRANSFORMED — abstracted, repo-specific details removed — before sharing, not copied verbatim. A colony instinct like "WordPress ACF sync requires Polylang activation first" should become "CMS plugin sync requires dependency activation first" in the shared tier. This preserves value while preventing cross-context leakage of implementation details. [S90]

**Conflict Resolution: Multi-Source Consensus (NEW, iteration 10):**
Last-writer-wins is explicitly identified as UNSAFE for multi-agent shared memory — a less competent agent can overwrite a better assessment. Four viable resolution strategies identified across multiple sources:
1. **Confidence-weighted:** Higher confidence + more validations wins. Already maps to Aether's instinct confidence scoring.
2. **Trust-scored:** Trust = (verified/total) × 0.7 + (activity) × 0.3, with high trust carrying more weight in arbitration.
3. **Temporal with supersession:** Graphiti incrementally integrates updates using temporal metadata. Newer facts can supersede older ones while maintaining history — contradictions tracked rather than deleted.
4. **Orchestrator-serialized:** All writes channeled through a designated role that arbitrates conflicts.

Strategy (4) maps best to Aether: the /ant:seal command already runs through Queen, so Queen naturally arbitrates which instincts merit hub promotion. Strategy (1) is the fallback: instincts already carry confidence scores, so when two repos promote contradictory wisdom, highest confidence × validation_count wins. Write contention between simultaneous colonies is handled by Aether's existing file locking in eternal-store. [S90][S91][S45]

**Colony-Prime Performance Budget for Cross-Repo Injection (NEW, iteration 10):**
Colony-prime currently assembles 7 context sections via sequential file I/O: (1) QUEEN.md wisdom (awk parse of global + local), (2) Context capsule (sub-invocation), (3) Phase learnings (jq on COLONY_STATE, max 15/5), (4) Key decisions (awk on CONTEXT.md, max 5/3), (5) Blocker warnings (jq on flags.json, max 3/2), (6) Rolling summary (tail of log, 5 entries), (7) Pheromone signals (sub-invocation, max 8+5). Adding an 8th section for hive/eternal memory would require one additional jq call on memory.json (~151 lines, max 500 entries). The --compact flag already reduces all limits. **Performance impact: MINIMAL** — one jq call on a bounded JSON file, comparable to existing flags.json processing. The real constraint is TOKEN BUDGET in the resulting prompt, not processing time. Colony-prime's --compact flag should also limit hive entries (e.g., max 5 in compact, 10 in full). [S71][S92]

**Aether Hub Empirical State (NEW, iteration 10):**
Registry tracks 13 repos across 5+ domains: M4L audio (2), WordPress sites (3), desktop apps (2), AWS services (3), Aether itself (1), and others. Eternal memory has 24 sealed colonies (23 "Crowned Anthill") but 0 high-value signals and 0 cross-session patterns. The eternal-store entry schema is `{content, type, source, signal_id, reason, strength, created_at, archived_at}` — critically MISSING: source_repo_path (can't do domain-scoped retrieval), domain_tags[] (can't filter by relevance), promoting_colony_goal (can't trace provenance). Registry-add uses a simple upsert with no file locking (unlike eternal-store). Registry also MISSING: domain_tags[], last_colony_goal, active_colony flag, colony_count. These gaps are the concrete blockers for cross-repo intelligence — the infrastructure exists but the metadata needed for scoped retrieval does not. [S92][S93][S76][S81]

**Concrete Hive Data Model Specification (NEW, iteration 10):**
Synthesized from all research, the hive/ directory at `~/.aether/hive/` would contain:

1. **hive-wisdom.json** — Promoted instincts with extended schema:
   ```json
   {
     "content": "abstracted wisdom text",
     "type": "pattern|redirect|philosophy|stack",
     "source_repo": "/path/to/repo",
     "source_colony_goal": "original colony goal",
     "confidence": 0.85,
     "validated_count": 3,
     "domain_tags": ["wordpress", "e-commerce"],
     "created_at": "ISO-8601",
     "last_accessed": "ISO-8601",
     "access_count": 5,
     "promoted_by": "seal"
   }
   ```
   Scoping: "universal" entries (no domain_tags, apply everywhere) vs "domain-scoped" entries (apply to repos with matching tags). Conflict resolution: Queen-arbitrated, highest confidence × validated_count wins. Cap: 100 entries with exponential decay on last_accessed. Content is ABSTRACTED before promotion (repo-specific details removed).

2. **hive-signals.json** — Cross-repo pheromone signals promoted from eternal memory with full provenance. Same cap and decay model.

3. **user-profile.json** — User communication preferences, domain expertise, decision patterns (separate from task wisdom, per Hermes USER.md pattern). Not colony-specific.

4. **domain-registry.json** — Enhanced registry with domain tags per repo, colony history, and active colony status.

Promotion flow: repo instinct → /ant:seal checks confidence ≥ 0.7 + validated_count ≥ 2 → Queen abstracts content → writes to hive-wisdom.json.
Retrieval flow: /ant:init or colony-prime reads hive-wisdom.json → filters by domain_tags matching current repo → injects as "--- HIVE INTELLIGENCE ---" section in worker prompts. [S90][S88][S30][S71][S76][S93]

### Q4: Next-gen AI coding tools — memory and personalization patterns (partial — 75%)

**Claude Code — Four-Layer Memory Hierarchy:**
1. Always-loaded CLAUDE.md files (project, user, auto-memory) with 200-line hard cap per file
2. Daily logs in timestamped YYYY-MM-DD.md files capturing completed work, decisions, lessons
3. Project state via embedded `## State` sections in each project's CLAUDE.md
4. On-demand domain-specific topic files (e.g., bash-and-system.md, content-writing.md) loaded contextually via `/project` command
Automated cron maintenance: weekly lesson rotation, weekly design-rule validation, daily index consistency checks, weekly state entry archival. Eight design rules govern the system (discoverability, dating, schema enforcement, budget alerts, staleness detection, canonical locations, container fit, no rebuilding). [S20][S23]

**Windsurf (Cascade) — Auto-Learning Memories:**
Two memory types: user-created (explicit rules like "always use TypeScript strict mode") and auto-generated (Cascade learns from interactions). Persists coding patterns, project structure, preferred frameworks across sessions. Known limitation: occasionally clings to outdated patterns after major refactors, requiring developer oversight. [S21][S26]

**Cline — Memory Bank (Session-Reset Architecture):**
6 structured markdown files read in strict dependency order at session start: projectbrief.md → productContext.md / systemPatterns.md / techContext.md → activeContext.md → progress.md. Complete memory wipe between sessions by design — the bank IS the memory. `new_task` tool enables context handoff when approaching limits, packaging decisions into a fresh session. `.clineignore` reduces token footprint. [S19]

**Cursor — No Native Memory (Plugin Ecosystem):**
No built-in persistent memory. Uses Rules (.cursor/rules/*.mdc, user rules, team rules, AGENTS.md) for project-level instructions only. The memory gap has spawned third-party solutions via MCP: ContextForge, Basic Memory, Recallium, cursor-memory-bank. This pattern — platform leaves memory as an extension point — is notable. [S24][S26]

**Aider — Model-Agnostic Conventions Only:**
CONVENTIONS.md forwarded to whatever LLM is used. No semantic memory, no auto-learning, no session persistence beyond the conventions file. Simplest approach in the ecosystem. [S22]

**Emerging Cross-Tool Patterns:**
- "Agent memory" becoming a first-class MCP primitive in 2026 (Letta, Mem0, MemOS expose memory as active management, not passive retrieval)
- Production systems combining multiple complementary memory types rather than choosing one
- All tools converge on markdown files as the persistence layer
- Memory-as-MCP-server pattern enables tool-agnostic memory (works across Cursor, Claude Code, Cline) [S25][S27]

**Eight Design Rules for Persistent AI Memory** (from Claude Code power-user architecture):
1. Discoverability — every file referenced in an index
2. Dating — all lessons need timestamps for rotation
3. Schema enforcement — fixed section headers prevent invisible content
4. Budget alerts — maintenance jobs need resource limits
5. Staleness detection — automated index-vs-filesystem comparison
6. Canonical locations — facts exist in exactly one place
7. Container fit — respect loading mechanism constraints (e.g., 200-line caps)
8. No rebuilding — leverage native features before custom solutions [S20]

**Mem0 Technical Architecture (Deep Dive) [NEW, iteration 13]:**
Scalable memory-centric architecture with two variants: Mem0 (base, vector similarity) and Mem0g (graph-based with directed labeled graphs — entities as nodes, relationships as edges). Two-phase pipeline: (1) EXTRACTION processes messages + historical context with async summary generation; (2) UPDATE evaluates new memories against existing via Tool Call mechanism (CRUD operations). Three memory scoping levels: user (cross-conversation), agent (agent-specific), session (temporary). Performance benchmarks: 26% accuracy improvement over OpenAI Memory on LoCoMo, 91% lower p95 latency, 90% token cost reduction. v1.0.0 features: rerankers, async-by-default, graph memory, multimodal, webhooks, custom categories. SOC 2 Type II + GDPR compliant. Clean API boundary with minimal lock-in — swappable without affecting agent framework. Python and JavaScript SDKs; works with LangChain, CrewAI, AutoGen. [S109][S120]

**Letta/MemGPT Architecture (Deep Dive) [NEW, iteration 13]:**
Three-tier memory inspired by computer architecture: (1) CORE MEMORY — small block living INSIDE the context window (like RAM), agent reads/writes directly for key facts, each block has label + description + value + character limit. (2) RECALL MEMORY — searchable conversation history OUTSIDE context (like disk cache), auto-saves to disk. (3) ARCHIVAL MEMORY — long-term queried via tool calls (like cold storage), supports vector or graph database backends. Key mechanism: agents SELF-EDIT memory by calling memory functions during reasoning — the agent decides what's worth remembering. Eviction: ~70% of messages removed when capacity reached, recursive summarization preserves most recent context. Sleep-time compute: async memory agents handle proactive refinement during idle periods. Core design principle: "Designing an agent's memory is essentially context engineering: determining which tokens enter the context window." [S110][S29]

**Zep/Graphiti Temporal Knowledge Graph (Deep Dive) [NEW, iteration 13]:**
Formal model G=(N,E,φ) with three-tier subgraph hierarchy: (1) EPISODE — raw input as non-lossy episodic nodes; (2) SEMANTIC ENTITY — extracted entities resolved via cosine similarity on 1024-dim embeddings + full-text search + LLM duplicate determination; (3) COMMUNITY — label propagation clusters with summaries. Bi-temporal model: T (event timeline) + T' (transactional timeline), each edge carries 4 timestamps enabling temporal queries. Contradiction handling: LLM detects new facts contradicting existing → invalidation (NOT deletion), full history preserved. Retrieval pipeline: Search (cosine + BM25 + BFS) → Rerank (RRF, MMR, episode-mentions, node-distance, cross-encoder) → Construct context. LongMemEval benchmarks with gpt-4o: 71.2% accuracy (vs 60.2% full-context), 2.58s latency (vs 28.9s), only 1.6k context tokens vs 115k. Implementation: Neo4j + Cypher, BGE-m3 1024-dim embeddings. [S111][S98]

**Windsurf Cascade Memories (Deep Dive) [NEW, iteration 13]:**
Auto-generation: Cascade creates memories when encountering "useful" context during conversations. 48-HOUR LEARNING PERIOD on new codebases — autonomously learns architecture, naming, libraries, style. Measured 78% pattern matching accuracy on 50k-line React/Node.js project. Storage: ~/.codeium/windsurf/memories/, workspace-specific (no cross-workspace), local only. FREE — no credit consumption. Seven-step context assembly pipeline: (1) Load global rules, (2) Load project rules, (3) Load relevant memories (relevance-based retrieval), (4) Editor state, (5) @-commands, (6) Flow context (edits, terminal, navigation), (7) Model constraints (trim to fit window). Tab completion uses separate speed-optimized pipeline. Agent Mode gets comprehensive context. Memories are positioned as temporary — durable team knowledge belongs in Rules/AGENTS.md. [S112][S118][S119]

**LoCoMo Benchmark Landscape (Quantitative) [NEW, iteration 13]:**

| System | LoCoMo Score | Cloud Required | Open Source |
|--------|-------------|----------------|-------------|
| EverMemOS | 92.3% | Yes | No |
| MemMachine | 91.7% | Yes | No |
| Hindsight | 89.6% | Yes | No |
| SLM V3 Mode C | 87.7% | Yes | Yes (MIT) |
| Zep | ~85% | Yes | Partial |
| Letta/MemGPT | ~83.2% | Yes | Yes (Apache) |
| SLM V3 Mode A | 74.8% | **No** | Yes (MIT) |
| Supermemory | ~70% | Yes | Yes (MIT) |
| Mem0 (self-reported) | ~66% | Yes | Partial |
| Mem0 (independent) | ~58% | Yes | Partial |
| SLM V3 Zero-LLM | 60.4% | **No LLM** | Yes (MIT) |

Cloud systems cluster 83-92%. SLM V3 Mode A achieves 74.8% locally — only ~8% gap. EU AI Act (Aug 2026) creates compliance pressure favoring local-first approaches — only SLM claims compliance-by-architecture. [S113][S111]

**MCP Memory Server Ecosystem [NEW, iteration 13]:**
Five major implementations forming a new infrastructure category:
1. **OpenMemory** (Mem0) — unified memory across Cursor, Claude, VS Code as "memory chip" for all MCP clients
2. **Hindsight** (Vectorize) — PostgreSQL + pgvector, 4-strategy parallel retrieval (semantic + BM25 + entity graph + temporal), cross-encoder reranking, memory banks for isolation, 3 core tools (retain/recall/reflect) + 6 mental model tools, separate LLM API key required, 4096 token recall limit
3. **MCP Backpack** — per-project git-friendly memory, two-layer storage (local cache + JSON file for portability)
4. **ContextForge** — cross-tool (Claude Code, Cursor, Copilot), semantic search + GitHub integration + task tracking
5. **Basic Memory** — markdown files → knowledge graph with semantic connections

Pattern: memory-as-MCP-server is the dominant cross-tool strategy in 2026, making persistent memory tool-agnostic. [S114][S115][S116][S117]

**Four Retrieval Fusion Architectures Compared [NEW, iteration 13]:**
(A) Mem0 — vector similarity, single-strategy. (B) Zep — cosine + BM25 + BFS + RRF/MMR/cross-encoder reranking, Neo4j. (C) Hindsight — 4-strategy parallel + cross-encoder, PostgreSQL. (D) SuperLocalMemory — 4-channel RRF (Fisher-Rao + BM25 + entity graph + temporal), fully local. Industry converging on MULTI-STRATEGY retrieval as best practice — pure vector search insufficient for production. Winning pattern: semantic + lexical + graph + temporal. Aether's current instinct retrieval (domain/confidence filter only) is effectively Tier 0. [S113][S111][S114]

**Five Tiers of AI Coding Tool Memory Sophistication [NEW, iteration 13]:**
- **Tier 0 (None):** Aider, base Cursor — conventions file only
- **Tier 1 (Session-scoped):** Cline Memory Bank — structured files, session-reset architecture
- **Tier 2 (Auto-learning):** Windsurf Cascade — workspace-local auto-memories, 48h learning, 78% accuracy
- **Tier 3 (Structured persistent):** Claude Code — four-layer hierarchy, cron maintenance, staleness detection, 200-line caps
- **Tier 4 (Memory-as-infrastructure):** MCP servers (Hindsight, OpenMemory, ContextForge) — cross-tool, multi-strategy retrieval, graph-based, temporal reasoning, tool-agnostic

Aether currently operates at Tier 1-2 (per-repo state with auto-learning via instinct promotion). Tier 4 represents the most significant shift — decoupling memory from any single IDE/agent framework. [S113][S114][S112][S20][S19][S22]

### Q5: Queen evolution to persistent personal assistant (partial — 73%)

**Current Queen: Stateless Task Orchestrator**
The Queen today is a per-colony coordinator with no memory between colonies. She selects workflow patterns (SPBV, Investigate-Fix, Refactor, Compliance, Documentation Sprint, Deep Research), dispatches workers via Task tool, manages phase boundaries, and handles escalation chains (4-tier failure handling). Her identity is entirely defined by the static agent definition file — she has no accumulated knowledge or user model. Each `/ant:init` starts fresh with zero context from prior colonies. [S34]

**QUEEN.md: Active Infrastructure, Empty Data (CORRECTED)**
Previous iteration stated QUEEN.md was "not wired into any colony lifecycle event" — this is WRONG. Colony-prime (aether-utils.sh:7560-7960) [S71] actively loads BOTH global (~/.aether/QUEEN.md) AND local (.aether/QUEEN.md), combines wisdom per-category (global first, local extends), and injects combined wisdom into a `--- QUEEN WISDOM (Eternal Guidance) ---` prompt section sent to every worker during builds. The two-level loading with content concatenation per category is already implemented. Colony-prime also assembles 6 other context sections: context capsule, phase learnings (from COLONY_STATE.json), key decisions (from CONTEXT.md), blocker warnings (from flags.json), rolling summary (from rolling-summary.log), and pheromone signals (from pheromone-prime). The seal command (Step 3.6) has a full wisdom review process: batch auto-promotion for threshold-meeting observations, then interactive checkbox UI for manual approval. The infrastructure is complete and active — it is operating on empty data. [S71][S77][S35][S36]

**Root Cause: Why QUEEN.md Is Empty**
Deep codebase investigation reveals the bottleneck is lifecycle completion, not infrastructure:
1. Only ONE observation exists in learning-observations.json — a pattern about "jq if/elif chains" with observation_count=2 [S74]
2. Code thresholds are LOW: propose=1 for all types, auto=2 for patterns — this single observation MEETS both thresholds [S72]
3. But no colony has completed a full /ant:seal cycle, which is where batch auto-promotion fires
4. The /ant:continue command also has learning-approve-proposals, but requires observations to exist
5. Eternal memory at ~/.aether/eternal/ has tracked 24 colonies but stored 0 high-value signals — no pheromone has reached strength > 0.8 at expiry [S76]
6. The global QUEEN.md at ~/.aether/ is empty (0 entries, 0 colonies contributed, created 2026-02-21) [S75]

**Documentation Inconsistency**: Template text describes higher thresholds (philosophies "5+ validations", patterns "3+ validations") than code implements (propose=1 for all). This creates confusion but doesn't block promotion — the code thresholds are authoritative. [S72][S75]

**OpenClaw's SOUL.md Pattern for Persistent Identity**
OpenClaw stores agent personality in SOUL.md — a Markdown file read at every session start. The agent "reads itself into being" each session. Identity persists through file-based storage (~/clawd/ directory). When the agent processes a message, it reloads conversation history from the file system. The agent progressively builds understanding of the user through timestamped memory logs, allowing increasingly sophisticated user models over extended interactions. Key difference from QUEEN.md: SOUL.md is actively loaded into every prompt as identity context, not a passive file. [S37]

**OpenClaw Identity Architecture (Deep Dive)**
OpenClaw loads 8 workspace files at session start: AGENTS.md (behavior), SOUL.md (philosophy), IDENTITY.md (presentation), TOOLS.md (capabilities), USER.md (user context), MEMORY.md (facts), BOOTSTRAP.md (init), HEARTBEAT.md (periodic tasks). All concatenated into system prompt with 20,000 character total cap. Four-tier identity resolution cascade: global config → per-agent config → workspace file → default fallback. Critical design choice: IDENTITY.md is SEPARATE from SOUL.md — identity covers presentation (name, emoji, avatar, creature, vibe, theme) while soul covers behavior/philosophy. Multi-agent isolation through separate workspace dirs + state dirs + session stores sharing one gateway. Dynamic identity swapping via hooks: `agent:bootstrap` event can replace SOUL.md content in memory before the model sees it (probability-based or time-scheduled). Identity type: `{name, emoji, theme, creature, vibe, avatar}`. Key insight for Aether: identity (presentation) should be separate from wisdom (accumulated knowledge) — QUEEN.md currently conflates both. [S69]

**Hermes Agent Dual Memory Architecture (Deep Dive)**
Hermes implements local memory + Honcho in a dual architecture with concrete code patterns:
- **Local**: MEMORY.md (2200 char cap, ~800 tokens) + USER.md (1375 char cap, ~500 tokens) in ~/.hermes/memories/. MemoryStore class enforces character limits — agent must PRUNE low-signal entries to add new ones, forcing curation.
- **Snapshot pattern**: Memory content frozen at agent init via `capture_snapshot()`. Mid-session writes update disk but NOT the active prompt — preserves LLM prompt cache hits. `build_context_files_prompt()` injects frozen snapshot into system prompt.
- **Honcho injection**: Crucially, Honcho context is appended to the USER MESSAGE (not system prompt) to preserve prompt caching: `"[Previous context: {honcho_ctx}]"`. Fresh vector similarity search per-turn, not cached.
- **Dialectic reasoning**: `honcho_conclude` tool extracts structured facts with configurable effort levels: minimal (1 LLM call), low (2-3), medium (4-6), high (8-12 calls). Async prefetch fires at end of each turn, results cached by session ID for next turn (eliminates 200-800ms round-trip latency).
- **Memory mode gating**: hybrid/honcho/local per peer — prevents duplicate state during migration.
- **Proactive flushing**: Gateway fires background memory save before session reset to prevent data loss.
Key insights for Aether: (1) Hard character caps force curation over accumulation, (2) Frozen snapshots prevent prompt invalidation, (3) Cross-repo context goes in worker prompts separately from local signals, (4) User model (USER.md) is architecturally separate from task memory (MEMORY.md). [S70]

**The Presence Continuity Layer Concept**
A model-agnostic identity layer between users and AI systems maintaining persistent identity, long-term memory, contextual continuity, and relational state across time, devices, and model boundaries. Core principle: the user should remain continuous even when the model, interface, or environment changes. This maps directly to Aether's multi-tool ambition (Claude Code + OpenCode + future tools). [S41]

**Emerging Industry Pattern: Cross-Context Identity**
AI assistants in 2026 are splitting identity into layers: (1) Claude scopes memory by project to prevent cross-context leakage, (2) Dume.ai unifies identity across 50+ connected tools, (3) Lindy prioritizes "memory quality over quantity" with selective fact retention. The industry is converging on RAG-based identity persistence where facts about the user are stored, indexed, and injected into prompts via retrieval rather than static loading. [S39]

**Three Evolution Axes for the Queen:**
1. **Wisdom persistence** — Wire QUEEN.md into the colony lifecycle: /ant:seal promotes validated instincts to QUEEN.md; /ant:init loads accumulated wisdom into the new colony's starting context.
2. **User model** — Add a separate USER.md or user-profile.json at the hub level (~/.aether/) capturing communication preferences, domain expertise, and decision patterns learned across all repos.
3. **Cross-project bridging** — Promote high-confidence instincts from per-repo COLONY_STATE to global QUEEN.md in the hub, so knowledge flows repo → hub → all repos. [S34][S35][S37][S40]

**Concrete Implementation Roadmap (informed by codebase + external patterns)**
- **Phase 1 (immediate, zero-risk):** Fix QUEEN.md template text to match actual code thresholds. Execute a manual /ant:seal to exercise the promotion pipeline and populate QUEEN.md with first entries. Verify the existing observation meets auto-promote criteria. [S72][S75]
- **Phase 2 (short-term):** Add character/entry caps to QUEEN.md sections (e.g., max 20 entries per category, analogous to Hermes' 2200-char cap). Create a USER.md at ~/.aether/ level separate from QUEEN.md, following Hermes' separation of user model from task memory. [S70]
- **Phase 3 (medium-term):** Wire eternal memory reads into colony-prime — when assembling the worker prompt, also load high-value signals from ~/.aether/eternal/memory.json as a cross-repo intelligence section. This is the lowest-friction cross-repo bridge since eternal-store already writes there. [S71][S76]
- **Phase 4 (future):** Expose accumulated wisdom via MCP server for tool-agnostic access across Cursor, Cline, OpenCode. [S31]

**Cross-Question Gap Resolutions [SYNTHESIS, iterations 16-17]:**
Three Q5 gaps resolved through cross-referencing:
1. **Wisdom decay mechanism (RESOLVED, it16):** Q6's exponential decay model with access-boost directly answers this [S99]. QUEEN.md wisdom entries that aren't referenced in recent builds should decay in effective weight — the formula `eff = strength × e^(-λt) × access_boost` applies equally to wisdom as to pheromones. Aligns with Entity-Fact model's exponential decay [S30] and ICLR 2026 MemAgent's frequency-weighted retention [S99].
2. **Global vs local QUEEN.md conflict resolution (RESOLVED, it16):** Q2 shows colony-prime concatenates global+local per category with "global first, local extends" ordering [S71]. When entries contradict, Q3's conflict resolution consensus applies: confidence-weighted resolution (highest confidence × validated_count wins) [S90][S91]. Global provides universal baseline; local adds repo-specific knowledge with contextual precedence.
3. **Character/entry caps precedents (NARROWED, it17):** Three independent precedents now established: Claude Code's 200-line hard cap on CLAUDE.md [S20][S23], OpenClaw's 20K char total workspace cap [S105], and Hermes' per-file character caps (MEMORY.md: 2200 chars, USER.md: 1375 chars) [S70]. The remaining gap is purely a DESIGN DECISION (choosing Aether-specific values), not a research gap. Recommended approach: per-section entry caps (e.g., max 20 entries in Philosophies, 15 in Patterns) rather than character caps, since Aether's QUEEN.md sections are list-based, not prose.

### Q6: Next-gen pheromone system architecture (partial — 74%)

**Current System Deep Dive:**
Aether has four signal types: FOCUS (normal priority, 30d decay), REDIRECT (high priority, 60d decay), FEEDBACK (low priority, 90d decay), and POSITION (lowest priority, tracks work location). Linear decay formula: `effective_strength = initial_strength × (1 - elapsed_days / decay_days)`. Signals below 10% strength are deactivated. `pheromone-prime` assembles active signals + instincts (confidence ≥ 0.5, not disproven) into a prompt section grouped by type. `colony-prime` unifies QUEEN.md wisdom + pheromone-prime output into a single injection block for workers. Auto-emission fires from five event types: failure, redirect guidance, feedback guidance, validated learning, and resolution events. `suggest-analyze` scans the codebase for patterns worth capturing as signals. [S9][S15][S46]

**Eternal Memory — Embryonic Cross-Repo Bridge:**
The `eternal-store` mechanism already promotes high-value signals (strength > 0.8 at expiry) to `~/.aether/eternal/memory.json`. Schema: `{version, created_at, colonies[], high_value_signals[], cross_session_patterns[]}`. This is the existing cross-repo signal bridge — signals validated in one repo persist at the hub level. However, no colony currently reads from eternal memory during init or build, making it write-only. Wiring eternal memory into colony-prime would immediately enable cross-repo signal flow without new infrastructure. [S9]

**Stigmergic Blackboard Protocol (SBP) — External Validation:**
SBP (2025-2026) implements the same digital pheromone metaphor Aether uses: agents post signals with varying intensity to a shared blackboard; signals fade over time. SBP positions itself as complementary to MCP: "MCP defines Capabilities, SBP defines Awareness." Available as TypeScript/Python SDKs with pluggable storage (in-memory, Redis, SQLite). This validates Aether's pheromone approach as an emerging coordination pattern, not just a metaphor. SBP's blackboard concept maps directly to Aether's hub eternal memory as a shared surface. [S43][S44]

**Event-Driven Agent Communication Patterns:**
Production AI agent architectures in 2026 use pub/sub with semantic topic hierarchies. Agents publish events to typed topics (e.g., `deal/closed/enterprise`); subscribers react based on topic patterns with wildcard matching (e.g., `orders/+/completed`). Key properties: broker-mediated delivery, temporal decoupling (agents need not be online simultaneously), topic-level access control for isolation, and multi-consumer patterns (one event triggers parallel workflows). Aether's pheromone tags already carry `category` and `value` fields — these could evolve into semantic topic hierarchies. [S42]

**Multi-Agent Shared Memory Patterns:**
2026 coordination frameworks use two-layer shared memory: short-term scratchpad (agents collaborate on a shared workspace) and persistent long-term storage (continuity across sessions). Conflict resolution prevents agents from overriding each other's outputs. Most production systems use hierarchical coordination (higher-level agents supervise teams) — matching Aether's Queen → workers model. Dynamic capability discovery via MCP enables propagating learned patterns across agent populations. [S45]

**Five Evolution Opportunities:**
1. **Hub-as-blackboard** — Make `~/.aether/eternal/` a readable stigmergic surface by wiring it into colony-prime. Colonies write validated signals on seal; new colonies read relevant signals on init. The hub becomes a shared environment colonies modify and sense.
2. **Semantic signal topics** — Evolve flat tags into hierarchical topics (`testing/coverage`, `security/auth`, `performance/database`). Enable scoped subscription so colonies only receive signals relevant to their domain.
3. **Signal aggregation** — When multiple colonies emit similar signals (same content hash or high semantic similarity), auto-aggregate into cross-repo patterns with boosted confidence. Multi-colony validation = stronger signal.
4. **User preference signals** — New `PREFERENCE` signal type capturing user communication style and decision patterns. Persisted at hub level, injected into all colony contexts. Separate from task-oriented signals.
5. **Automated exchange** — Replace manual XML export/import with hub-level sync: `init` pulls relevant global signals, `seal` pushes validated signals to hub. No user intervention needed. [S9][S42][S43][S45]

**Pheromone-Prime Implementation Details (NEW, iteration 11):**
Deep codebase analysis of pheromone-prime [S94] reveals the exact assembly mechanics. Priority ordering uses numeric values: REDIRECT=1, FOCUS=2, FEEDBACK=3, POSITION=4, with secondary sort by effective_strength descending. Default caps: max_signals=8, max_instincts=5 (3 in compact mode). The prompt section uses labeled headers — 'REDIRECT (HARD CONSTRAINTS - MUST follow):', 'FOCUS (Pay attention to):', 'FEEDBACK (Flexible guidance):', 'POSITION (Where work last progressed):'. Each signal renders as `[strength] content_text`. Instincts are grouped by domain field (uppercased first letter), rendered as `[confidence] When trigger -> action`. **Critical gap: NO TOKEN BUDGET exists.** The total injection size is unbounded beyond count caps. With 8 signals averaging ~40 tokens each + 5 instincts averaging ~30 tokens each = ~470 tokens typical, but a single 500-char signal can consume 125+ tokens alone. [S94]

**Eternal Promotion Flow — Strength Threshold Bug (NEW, iteration 11):**
The pheromone-expire flow [S95] checks `strength > 0.8` for eternal promotion, but this uses the ORIGINAL `.strength` field, NOT the decayed `effective_strength`. Since decay only affects the computed effective_strength in pheromone-read (not the stored value), a REDIRECT signal with default strength 0.9 will ALWAYS qualify for eternal promotion when it expires — regardless of how long it decayed. Effectively ALL REDIRECT signals (default 0.9) and ALL FOCUS signals (default 0.8) are eligible. This means the 0.8 threshold is not the selective quality gate it appears to be — it's essentially a pass-through for anything except FEEDBACK (default 0.7). [S95][S81]

**Real Pheromone Data Reveals Schema Inconsistencies (NEW, iteration 11):**
Analysis of the actual pheromones.json [S101] reveals: (1) Legacy signals (Feb 16) have NO strength field, NO expires_at field, and structured tags arrays with weight/category — schema v0. Newer signals (Mar 19) have strength, expires_at, reason, but NO tags — schema v1. (2) Two IDENTICAL duplicate FEEDBACK signals exist 1 second apart (same content "Learning captured: Use explicit jq if/elif chains...") — pheromone-write has NO deduplication (unlike learning-observations.json which SHA-256 hashes content). (3) An expired REDIRECT signal (expires_at 2026-03-16, 4 days ago) still has active=true because pheromone-expire wasn't run. These demonstrate the need for: schema migration logic, content deduplication in pheromone-write, and periodic maintenance runs. [S101][S15]

**Suggest-Analyze: Static Not Semantic (NEW, iteration 11):**
The suggest-analyze engine [S96] uses grep-based static analysis to detect 6 pattern types with numeric priorities: debug artifacts (REDIRECT, priority 9), large files >300 lines (FOCUS, 7), high complexity >20 functions (FOCUS, 6), type safety gaps (FEEDBACK, 5), TODO/FIXME comments (FEEDBACK, 4), and missing test files (FEEDBACK, 4). Each suggestion gets SHA-256 hash for deduplication. This is entirely rule-based — no LLM analysis, no semantic similarity, no cross-file pattern detection. The opportunity: LLM-powered analysis could detect architectural patterns, recurring error categories, and cross-file dependencies that grep cannot. [S96]

**SBP Merge Strategies — Concrete Aggregation Model (NEW, iteration 11):**
SBP [S97] implements four merge strategies when emitting to an existing signal location: **Reinforce** (boost intensity), **Replace** (overwrite), **Max** (keep highest), **Additive** (sum intensities). Signals use trail-based hierarchical namespacing with dot notation (e.g., `market.signals`, `pipeline.stage1`). Retrieval via `sniff(trails=[])` filters by trail name — prefix matching for domain scoping. Scent conditions provide threshold-based reactive triggers: agents register interest in trail+type combinations with operator/value conditions (e.g., `when tasks/new_task intensity >= 0.5, wake agent`). **Mapping to Aether:** (1) Reinforce = when new learning matches existing pheromone, boost strength +0.2 (cap 1.0) and reset decay, (2) Trails = extend existing tags into dot-notation hierarchical namespaces, (3) Scent conditions = threshold-based auto-emission (if instinct confidence > 0.8, auto-emit FEEDBACK). [S97][S44]

**Graphiti/Zep Temporal Supersession Model (NEW, iteration 11):**
Zep's bitemporal knowledge graph [S98] implements temporal fact management with two timelines: T (event timeline — when things happened) and T' (transactional timeline — when data entered the system). Each edge carries four timestamps: t'_created, t'_expired, t_valid, t_invalid. When a new fact contradicts an existing one (detected via LLM comparison against semantically similar edges), the system sets `t_invalid` on the old edge to the `t_valid` of the new edge — **invalidation, NOT deletion**. History is fully preserved: you can query "what was true at time X?" Formal model: G=(N,E,φ) with three-tier graph (episodes → semantic entities → communities). **For Aether:** When /ant:redirect supersedes a previous REDIRECT on the same topic, the old signal should get `superseded_by` pointing to the new signal's ID, plus `valid_until = new.created_at`, rather than being deactivated. This creates an audit trail of evolving constraints and enables temporal queries. [S98]

**Adaptive Memory Admission for Token Budget (NEW, iteration 11):**
The ICLR 2026 MemAgent framework [S99] evaluates memories using four value signals: (1) task relevance — contextual alignment, (2) usage frequency — retrieval count, (3) temporal recency — freshness, (4) performance impact — contribution to successful outcomes. A threshold-based gating mechanism admits memories exceeding a composite score; the threshold **ADAPTS** based on available context window capacity (tighter budget = higher bar). Memory states: Active working → Secondary buffer → Long-term storage → Compressed summaries. Expired memories compress into abstracts, not deleted. **For Aether:** pheromone-prime should implement adaptive admission. The four signals map to: relevance = domain trail match, frequency = access_count (new field), recency = effective_strength (already includes decay), impact = task success (traceable through midden vs completion records). [S99]

**Anthropic's Context Engineering Principles (NEW, iteration 11):**
Anthropic's official guidance [S100] frames context as a finite resource with diminishing returns — "every new token depletes the attention budget." Core strategy: find "the smallest set of high-signal tokens that maximize desired outcome." Sub-agents should return condensed summaries (1,000-2,000 tokens max). Claude Code retains 5 most recently accessed files after compaction — a concrete recency heuristic. Multi-agent token distribution: input outnumbers output 2:1-3:1, with 72% consumed in verification phases (MetaGPT study). **For Aether:** Pheromone injection is "naive upfront loading" — all signals injected before workers start. This is correct for REDIRECT (hard constraints must always be visible) but wasteful for FEEDBACK/POSITION that may not be relevant to the specific task. Aether could implement JIT signal injection: all REDIRECTs upfront, FOCUS/FEEDBACK only when matching current task's domain. [S100]

**Concrete Next-Gen Pheromone Architecture (NEW, iteration 11):**
Synthesizing all findings into an implementable specification:

**(A) Decay Model — Linear → Exponential with Access Boost:**
Replace `eff = strength × (1 - elapsed/max)` with `eff = strength × e^(-λt) × access_boost` where λ varies by type (FOCUS λ=0.023/day for ~30d half-life, REDIRECT λ=0.012/day for ~58d, FEEDBACK λ=0.008/day for ~87d), and `access_boost = min(1.0, 1 + 0.1 × access_count)`. Frequently-referenced signals decay slower — "use it or lose it." Follows ICLR 2026 MemAgent frequency-weighted retention pattern. [S99][S94]

**(B) Token Budget for Signal Injection:**
Add `--max-tokens` to pheromone-prime (default 500, compact 250). Priority admission: all REDIRECTs first (hard constraints, never skip), then FOCUS by eff_strength desc, then FEEDBACK. Estimate tokens per signal as `content_length / 4`. If budget exceeded, truncate lowest-priority signals. Cross-repo signals (future): separate budget — 70% local, 30% hive (configurable). [S100][S94]

**(C) Trail-Based Namespacing:**
Extend existing tags into dot-notation trails: `testing.coverage`, `security.auth`, `performance.database`, `architecture.patterns`. Add `--trail` flag to pheromone-write. Retrieval supports prefix matching (`testing.*` returns all testing signals). Auto-infer trails from suggest-analyze patterns + user-specified. [S97][S44]

**(D) Merge Strategies for Signal Aggregation:**
On pheromone-write, check existing signals for content similarity (SHA-256 hash exact match OR first 100 chars comparison). If match found: **Reinforce** for FEEDBACK (boost +0.2, reset decay), **Replace** for REDIRECT (supersede with audit trail per Graphiti model), **Max** for FOCUS (keep highest effective_strength). This also fixes the duplicate signal bug found in real data. [S97][S98][S101]

**(E) Temporal Supersession:**
Add `superseded_by` and `valid_until` fields to signal schema. Old signals become historical (queryable), not deleted. Signal lineage enables "what was the REDIRECT about X in February?" queries and wisdom extraction from constraint evolution. [S98]

**(F) Content Deduplication in pheromone-write:**
Before creating a new signal, compute SHA-256 of content.text and check against existing active signals. If match found, apply merge strategy instead of creating duplicate. Prevents the real-world issue of identical signals accumulating. [S101]

**(G) Schema Normalization:**
Add version field to individual signals. During pheromone-read, normalize legacy signals: default strength (REDIRECT=0.9, FOCUS=0.8, FEEDBACK=0.7) and expires_at ("phase_end") for signals missing these fields. [S101]

**(H) JIT Signal Injection:**
REDIRECTs always injected upfront (hard constraints). FOCUS and FEEDBACK injected based on task domain match — compare signal trails against current task's file paths/domains. Reduces token waste for irrelevant signals while ensuring constraints are never missed. [S100][S94]

### Q7: Dangers and risks (partial — 68%)

**Memory Poisoning — The #1 Security Threat:**
Palo Alto Unit 42 research demonstrates a three-phase attack on persistent AI agent memory: (1) Installation — attacker embeds malicious instructions in external content, (2) Persistence — poisoned content survives into memory via summarization, (3) Activation — memory auto-integrates into future sessions, executing attacker objectives without user visibility. Memory contents are injected into system instructions and prioritized over user input. For Aether: cross-repo memory sharing via the hub would amplify this — a poisoned signal promoted to eternal memory (strength > 0.8) could propagate to every future colony across all repos. [S47][S48]

**OWASP AI Agent Security Threats:**
Six key threat categories for agent systems: (1) Memory manipulation — poisoned data persists across sessions, (2) Tool abuse/privilege escalation — overly permissive tools enable lateral movement, (3) Cascading failures in multi-agent systems, (4) Goal hijacking — manipulating agent objectives while appearing legitimate, (5) Denial of wallet — unbounded agent loops causing excessive API costs, (6) Data exfiltration through tool calls. Recommended mitigations: per-tool permission scoping, message signing between agents, circuit breakers, human-in-the-loop for high-impact operations. [S48]

**Platform Dependency Risk:**
Claude Code has shipped breaking changes: deprecated /output-style command, removed Opus 4/4.1 models (auto-migrated to 4.6), changed top_p default (0.999→0.99), and experienced a DST-related infinite loop bug. Aether's 40 slash commands and 22 agent definitions depend on Claude Code's subagent protocol, command loading, and agent spawning — none with stability guarantees. Any breaking change could silently disable core Aether functionality. [S49]

**Scope Creep and Over-Engineering:**
Industry analysis identifies scope creep as critical: "an agent that can do anything will eventually do the wrong thing." Framework choice barely matters — state persistence, retry logic, and scope control are the real failure points. Aether's proposed evolution (cross-repo hive mind, MCP server, semantic topics, user modeling, persistent Queen identity) would dramatically increase complexity on top of a 10,499-line single bash file with known architectural issues. [S50][S9]

**Aether-Specific Codebase Fragility:**
Known issues documented in known-issues.md: 17+ missing error codes, potential infinite loop in spawn-tree (depth limit 5), lock release bugs on JSON validation failure, no error path test coverage, inconsistent error handling across early vs late commands. The utility layer uses eval in 4 places for argument parsing. Adding cross-repo features on this foundation without resolving existing issues increases fragility. [S51][S9]

**Privacy and Cross-Context Leakage:**
MIT Technology Review identifies AI memory as "privacy's next frontier." Cross-repo memory sharing means information from one context leaks into another — a pattern learned in a personal project could appear in a work project's context, exposing proprietary information. A single prompt can trigger cascading actions across services without user authorization. Memory governance (what to remember, what to forget, when to expire) is an emerging discipline Aether currently lacks entirely. [S52][S53]

**AI Recommendation Poisoning and Data Exposure:**
Microsoft Security (Feb 2026) documents adversaries manipulating AI memory for profit. Salt Security finds that agents must access content in plaintext to process it — encryption ends at the agent boundary. The trust boundary shifts from user's device to the orchestration layer, which in Aether's case is a bash script with no encryption or access control. Dual risk: external attacks on memory integrity AND internal leakage of sensitive data through persistence. [S54][S55]

**Aether-Specific Risk Quantification — Eval Usage (LOW risk):**
Deep codebase analysis of the 4 eval instances (aether-utils.sh:7204-7216) reveals they are in `instinct-read`, using the pattern `eval "ir_arg=\${$ir_shift}"` which only expands numbered positional parameters ($1, $2, etc.) — NOT arbitrary user input strings. Arguments originate from Claude Code's agent spawning protocol. This is a standard bash idiom for accessing positional params by variable index, not an injection vector. [S79]

**HIGHEST-SEVERITY GAP: Prompt Injection via Pheromone Content:**
The pheromone-write sanitization (aether-utils.sh:6814-6819) blocks SHELL injection (escapes `<`/`>`, truncates to 500 chars, rejects `$(`, backticks, curl, wget, rm patterns) but does NOT block PROMPT injection. Since pheromone content is injected into worker prompts via colony-prime, a signal like "CRITICAL: Ignore all previous instructions" would PASS sanitization (contains no shell metacharacters) and be injected into every worker's system prompt context. The attack chain: malicious content → learning observation → instinct promotion → auto-emitted FEEDBACK pheromone → colony-prime injects into all future builds. If eternal-store becomes readable, propagation extends to ALL repos across the hub. OWASP classifies this as ASI06 ("Memory Poisoning") in the Top 10 for Agentic Applications 2026. [S78][S71][S83][S87][S86]

**Eternal-Store Has No Independent Content Sanitization:**
Content flows from pheromone-expire → eternal-store (aether-utils.sh:8083) passing the raw `.content.text` field. Existing signals were sanitized at write time by pheromone-write, but eternal-store itself (aether-utils.sh:8160-8243) accepts arbitrary content with only numeric strength validation. It HAS a 500-entry cap and uses file locking + atomic writes for integrity. If eternal-store is called from a future path OTHER than pheromone-expire, unsanitized content could enter the eternal store. [S81][S78]

**Denial-of-Wallet: Partial Protection with Gaps:**
spawn-can-spawn enforces depth limits (depth 1→4 workers, depth 2→2, depth 3+→0) and a global cap of 10 workers per phase. Swarm has a separate 6-worker cap. However, NO per-worker TOKEN BUDGET exists — each worker can make unlimited API calls. Industry best practice (Hands-On Architects 2026) recommends three-tier hierarchical cost control: minute-level token bucket, daily quota, weekly quota as cost ceiling. Multi-agent costs multiply 3.5x in documented cases, with retry loops burning $40+ in minutes. Aether has no token-level budget, no cost tracking, and no circuit breaker for runaway workers. [S80][S84][S85]

**File Permissions Assessment:**
Hub directory (~/.aether/) is 755 (world-readable). QUEEN.md is 644 (world-readable). registry.json is 644 (world-readable, contains repo paths). Crucially, eternal/memory.json is 600 (owner-only) — the most sensitive file is properly restricted. No encryption anywhere. For single-user local use, this is acceptable. Risk increases for shared filesystems or multi-user environments. [S82]

**OWASP Five-Control Defense Assessment for Aether:**
OWASP recommends: (1) sanitize data before storage — Aether has PARTIAL (shell sanitization only, no prompt-injection detection), (2) isolate memory between sessions — Aether has FULL (per-repo isolation, separate COLONY_STATE per colony), (3) set expiration and size limits — Aether has PARTIAL (instinct cap 30, eternal cap 500, pheromone linear decay, but NO QUEEN.md entry cap or decay), (4) audit for sensitive data before persistence — Aether has NONE, (5) cryptographic integrity checks — Aether has NONE. Score: 1.5/5 controls implemented. [S48][S83][S86]

**Prioritized Risk Assessment (Aether-specific):**
- IMMEDIATE (exploitable today): (1) Prompt injection via pheromone content, (2) No per-worker API cost budget
- HIGH (exploitable if cross-repo enabled): (3) Memory poisoning amplification via eternal-store, (4) No memory governance (audit, force-forget, retention)
- MEDIUM (real but manageable): (5) Platform dependency on Claude Code, (6) Error handling inconsistencies (17+ missing error codes)
- LOW (theoretical or contained): (7) Eval usage, (8) File permissions, (9) npm supply chain [S78][S80][S81][S82][S51][S79][S48][S83]

**Concrete Mitigation Roadmap:**
- **Tier 1 (immediate):** Add prompt-injection patterns to pheromone-write sanitization. Add provenance metadata to eternal-store entries.
- **Tier 2 (short-term):** Per-build cost tracking via spawn-tree.txt. Memory integrity checksums (SHA-256 at promotion, verify before injection).
- **Tier 3 (medium-term):** /ant:memory-audit command for governance. Circuit breaker on colony-prime injection size.
- **Tier 4 (future):** Trust-aware retrieval for eternal signals. Scope tagging for domain-matched injection. [S78][S81][S84][S83][S86][S48]

**Bash as Security Foundation — Suitability Assessment:**
Bash is NOT fundamentally unsuitable for Aether's current single-user, local architecture. Eval risk is contained, atomic writes protect state, jq handles JSON safely. But bash becomes problematic for cross-repo security: no native encryption primitives, no type system to distinguish trusted vs untrusted content, no access control framework, no sandboxing. Security-sensitive cross-repo operations would benefit from a typed companion process (TypeScript/Go) handling the trust boundary, while bash continues the orchestration it does well. [S9][S78][S85][S82]

### Q8: Visual appeal and interactive terminal UI (partial — 55%)

**Aether's Existing Display Infrastructure:**
Aether already has more display tooling than expected. `swarm-display.sh` [S65] provides ANSI-colored real-time display with caste-specific colors (blue=builder, green=watcher, yellow=scout, red=chaos, magenta=prime), 20+ caste emojis, animated status phrases that rotate by time ("excavating...", "forging...", "constructing..."), progress bars using █░ characters, and tool usage counters (📖🔍✏️⚡ for Read/Grep/Edit/Bash). A detailed in-conversation display plan [S66] exists with a complete implementation spec for `swarm-display-text` (ANSI-free variant for Claude conversations), including compact headers, 5 edge cases, and exact insertion points across 7 commands. The `/ant:maturity` command [S67] uses ASCII art anthill with milestone-as-storytelling ("First Mound" through "Crowned Anthill"). The in-conversation display plan is **fully spec'd but not yet implemented** — it's the most actionable visual improvement.

**TUI Framework Landscape (2026):**
The dominant frameworks are Charm's BubbleTea v2 (Go, Elm architecture — Init/Update/View, 10x faster "Cursed Renderer") [S60], Ratatui (Rust), Textual (Python), and Ink (TypeScript). BubbleTea v2 supports Mode 2026 for synchronized output that eliminates screen tearing in modern terminals like Ghostty. Lip Gloss provides CSS-like declarative styling for terminals. TUI Studio [S61] is a new visual drag-and-drop editor that exports to 6 frameworks, bringing Figma-like design to terminal UIs with 21+ pre-built components, 3 layout engines (Absolute, Flexbox, Grid), and 8 color themes (Dracula, Nord, Gruvbox). For Aether's bash foundation, these frameworks aren't directly usable but their design patterns inform what good terminal UI looks like.

**CLI Progress Display Patterns:**
Three established progress types [S57]: (1) **Spinners** for quick sequential tasks (minimal but signals activity), (2) **X-of-Y counters** recommended as default ("5/10KB" — clearly shows stalls, enables time estimation), (3) **Progress bars** for multiple parallel processes (visual gauge + metrics + ETA, "might be overkill" in CLI). Key principles: transition status from gerund to past tense on completion ("downloading..." → "downloaded ✓"), clear progress indicators after completion, use green checkmarks for success, respect --no-color/--quiet/--plain flags. Progress displays address "the need for control is a biological imperative" for user engagement. [S57]

**CLI UX Design Principles:**
Ten patterns from Lucas F. Costa [S58] that map to Aether: (1) Smooth onboarding using the CLI itself (Aether's /ant:help partially does this), (2) Interactive mode with step-by-step prompts (/ant:council uses this), (3) Typo recovery via Damerau-Levenshtein distance, (4) Human-understandable errors with actionable fixes, (5) Visual hierarchy through color and emoji (Aether already excels here), (6) Loading indicators with time estimates, (7) Context-awareness detecting folder information, (8) Proper exit codes, (9) stdout/stderr stream separation, (10) Consistent command trees (/ant:* namespace follows this). The "Terminal Renaissance" [S59] emphasizes efficiency-first design over decoration: micro-interaction optimization, instant visual feedback, structured data over raw text streams, and context preservation.

**Visually Praised CLI Tools:**
The most admired tools share common patterns [S62][S68][S56]: **lazygit** (panel-based layout with real-time git state, reduced 48s → 18s workflows), **K9s** (real-time Kubernetes monitoring with selectable lists and continuous change observation), **btop** (resource monitor with color gradients, bar graphs, real-time updates), **httpie** (colorized and formatted HTTP output). The winning formula is "GUI-like responsiveness with keyboard-centric workflows" — panel layouts, vim keybindings, real-time data updates, and optional mouse support. The awesome-tuis list [S56] categorizes 11 TUI application types, with dashboards (system monitoring) and development tools (git clients) being the most successful categories.

**AI Agent Visualization Patterns (2026):**
New visualization patterns are emerging specifically for AI coding agents [S63][S64][S59]: **Ralph TUI** provides a real-time terminal UI for agent orchestration with subagent tracing (hierarchical display of nested agent calls), keyboard shortcuts for pause/resume, and session persistence across interruptions. **OpenCode** shows real-time token/context budget consumption. **Auggie CLI** streams tool calls visibly so users can follow agent reasoning. The **AG-UI protocol** (CopilotKit) proposes standardized agent→UI event streaming. The Agentic Coding Handbook [S64] recommends screenshot-based feedback loops for visual validation. The overarching principle: make agent work VISIBLE — users need to see what's happening inside the "black box."

**Six Concrete Opportunities for Aether:**
1. **Implement the in-conversation display plan** — It's fully spec'd with exact line numbers, ~87 lines of changes, low risk (new function alongside existing). Most immediate win. [S66]
2. **Real-time token/cost tracking per ant** — OpenCode pattern. Users want to know what they're spending during builds. [S59]
3. **Hierarchical subagent tracing** — Ralph TUI pattern. Show Queen → Builder → tool calls as a collapsible tree. Makes the colony hierarchy visible. [S63]
4. **Colony dashboard mode for /ant:status** — Panel layout showing phase progress, active signals, memory health, and recent events simultaneously. Inspired by K9s/btop. [S68][S62]
5. **Themed color palettes** — Support Dracula, Nord, Gruvbox via configurable ANSI mappings. TUI Studio offers 8 built-in themes. [S61]
6. **Consistent display across contexts** — Transition /ant:watch tmux display to use the same compact format as in-conversation display. One design language. [S66]

**Cross-Question Validation [SYNTHESIS, iterations 16-17]:**
Q8's recommendations are strongly validated by cross-referencing: (1) "In-conversation display plan first" validated by Q7's scope creep warning — bash-native improvements recommended over external TUI frameworks [S50]. (2) Token/cost tracking validated by Q4's documentation of real-time budget display patterns [S59], but Q6 finds no per-worker token budget exists [S94] — you can't display what you don't track. (3) Colony dashboard data model constrained by Q2's colony-prime 7-section structure [S71]. (4) Companion TUI vs bash-native resolved by Q7's Theme 6: bash sufficient today, typed companion for cross-repo security and rich TUI [S9][S78]. (5) Q2's build wave mechanics [S121] show worker JSON responses (status, files_touched, tool_count, blockers) define the data available for display. (6) ANSI rendering gap partially resolved: the in-conversation display plan already specs a no-ANSI variant (swarm-display-text) for Claude Code markdown rendering [S66].

**Synthesis Confidence Assessment [iteration 17]:**
Q8's remaining gaps are narrow and well-characterized: (a) accessibility --plain flag is a specific implementation item, not a conceptual gap; (b) implementation priority is addressed by the consolidated 10-item priority list (Q8 ranks #10). The question has 10+ sources across 5 types (codebase, blog, documentation, github, official), multiple sources agree on framework landscape and design principles, and edge cases are identified. The 55% score undervalues the breadth — adjusting to 60% based on multi-source agreement with identified-but-narrow remaining gaps. Cross-referencing from Q3 also adds: hive data (domain-registry.json, hive-wisdom.json) would become future dashboard data sources, further validating the dashboard concept from Q8 opportunity #4.

## Cross-Question Synthesis

### Theme 1: The Empty Pipeline Problem
**Questions linked:** Q2 + Q5 + Q3 + Q6 + Q7

The entire vision — cross-repo intelligence (Q3), persistent Queen identity (Q5), next-gen pheromones (Q6) — is bottlenecked by a single operational gap: **the learning pipeline has never completed a full cycle.** Q2 confirms colony-prime actively loads QUEEN.md [S71]. Q5 confirms QUEEN.md is empty because no colony has completed /ant:seal [S75][S76]. Q3's hive data model depends on promoted instincts flowing through this pipeline. Q7's prompt injection risk applies to whatever eventually flows through.

The one existing observation in learning-observations.json [S74] MEETS the auto-promote threshold (propose=1, auto=2 for patterns, count=2) [S72] — but no seal cycle has exercised the promotion code path. **The #1 most impactful action is completing a single seal cycle** to exercise the promotion pipeline, populate QUEEN.md, test eternal-store, and validate the entire flow before designing extensions.

### Theme 2: Token Budget is the Universal Constraint
**Questions linked:** Q2 + Q6 + Q1 + Q4 + Q3

Colony-prime has NO total token budget [S71][S94]. Every proposed enhancement — hive intelligence (Q3's 8th section), richer signals (Q6), more instincts, deeper context — will consume worker attention budget. Q1 shows OpenClaw enforces a 20K char cap across all workspace files [S105]. Q4 shows Anthropic recommends "the smallest set of high-signal tokens" [S100]. Q6 proposes a 500-token default budget for pheromone injection [S94][S100].

Without a budget, the system will degrade silently as features are added. This is not a future concern — Q2 shows colony-prime already assembles 7 context sections with only count caps (not token caps). The typical pheromone injection is ~470 tokens [S94], but a 500-char signal alone can consume 125+ tokens. **Token budgeting across ALL colony-prime sections is prerequisite to any context expansion.**

### Theme 3: Multi-Strategy Retrieval is Industry Consensus
**Questions linked:** Q4 + Q1 + Q6 + Q3

Q4's LoCoMo benchmarks [S113] conclusively show multi-strategy retrieval wins: Hindsight (4-strategy, 89.6%) > Zep (3-strategy + reranking, ~85%) > Letta (~83%) > Mem0 (single-strategy, ~58-66%). Q1 confirms OpenClaw uses BM25 + vector (two strategies) [S102][S103]. Q6 documents that Aether has NO semantic retrieval — instinct lookup is domain/confidence filter only [S94], effectively Tier 0.

Q3's hive retrieval would need at minimum domain-scoped filtering. Q4's five-tier sophistication model [S113] places Aether at Tier 1-2 (per-repo state with auto-learning). The jump to Tier 3-4 (structured persistent → memory-as-infrastructure) requires semantic retrieval as a foundation. **This is the most actionable technical gap with the clearest path:** SQLite + FTS5 (already proven by OpenClaw's memsearch [S108]) provides hybrid search with zero external dependencies.

### Theme 4: Security Scales with Intelligence
**Questions linked:** Q7 + Q3 + Q6

Q7 demonstrates prompt injection via pheromone content is exploitable today [S78][S83][S86] — pheromone-write sanitizes for shell injection but NOT prompt injection. Q3's cross-repo sharing amplifies this to ALL repos via eternal-store. Q6's eternal promotion bug [S95] means nearly all REDIRECT (strength 0.9) and FOCUS (strength 0.8) signals auto-qualify for eternal promotion regardless of actual validation quality — the 0.8 threshold is a pass-through, not a quality gate.

The OWASP 5-control assessment (Q7) scores Aether at 1.5/5 [S48][S83][S86]. Three controls are partially or fully missing: audit for sensitive data before persistence, cryptographic integrity checks, and expiration/size limits on QUEEN.md. **Security posture (Q7's 1.5/5) must improve before cross-repo features (Q3) are safe to deploy.** The mitigation chain: prompt-injection sanitization (Q7) → abstraction transformation before sharing (Q3, [S90]) → provenance tracking on all shared content (Q6).

### Theme 5: Identity ≠ Wisdom ≠ User Model
**Questions linked:** Q1 + Q5 + Q3

Three independent systems validate a three-way separation:
- **OpenClaw** (Q1): SOUL.md (behavior/philosophy) vs IDENTITY.md (presentation) [S69]
- **Hermes** (Q5): MEMORY.md (task wisdom, 2200 char cap) vs USER.md (user model, 1375 char cap) [S70]
- **Collaborative Memory paper** (Q3): Private memory vs shared memory, designated at creation time [S90]

Q5 notes QUEEN.md currently conflates identity and wisdom. Q3's hive data model separates hive-wisdom.json (task wisdom) from user-profile.json (user model). **The converged recommendation: separate QUEEN.md into wisdom categories (already structured with 5 section headers) and create a distinct USER.md at hub level (~/.aether/USER.md) for communication preferences and decision patterns.** This matches both OpenClaw's and Hermes' independently-arrived-at architectures.

### Theme 6: Bash is Sufficient Today, Not Tomorrow
**Questions linked:** Q7 + Q8 + Q6 + Q2

Q7 shows bash is not fundamentally unsuitable for Aether's current single-user, local architecture [S9][S78] — eval risk is contained, atomic writes protect state, jq handles JSON safely. Q8 confirms bash-native display (swarm-display.sh [S65]) provides ANSI colors, progress bars, and emojis. Q2 shows colony-prime's 7-section assembly works via sequential bash I/O [S71].

However, Q7 also notes bash becomes problematic for: encryption (needed for cross-repo trust), type-safe trust boundaries (distinguishing sanitized vs unsanitized content), and sophisticated UI (Q8's TUI frameworks are all Go/Rust/Python). Q6's proposed exponential decay requires Taylor series approximation in jq (mathematically possible but fragile). **The synthesis: continue bash for orchestration, but security-sensitive cross-repo operations (Q3/Q7) and rich TUI (Q8) would benefit from a typed companion process.**

### Resolved Contradictions

1. **QUEEN.md wiring status** — RESOLVED with codebase evidence. Colony-prime (aether-utils.sh:7560-7960) [S71] actively loads BOTH global and local QUEEN.md and injects combined wisdom into every worker prompt. The issue is empty data (no seal cycle completed), not missing infrastructure. Previous iteration 4 finding was incorrect.

2. **Linear vs exponential decay** — RESOLVED in favor of exponential. Evidence: ICLR 2026 MemAgent [S99] uses frequency-weighted retention, Entity-Fact model [S30] uses exponential decay on unaccessed memories, SBP [S97] supports reinforcement-based intensity boosting. Linear decay is simpler but models constant loss regardless of usage. Implementation concern (jq cannot compute e^x natively) is solvable via Taylor series approximation (3 terms: e^x ≈ 1 + x + x²/2 + x³/6, sufficient for the 0-to-1 decay range).

3. **Security vs intelligence trade-off** — RESOLVED as layered defense, not either/or. Three layers working together: (a) prompt-injection pattern detection in pheromone-write [Q7 Tier 1], (b) content abstraction before cross-repo promotion [Q3, S90] (repo-specific details removed), (c) provenance metadata on all shared content [Q6, S98] (source_repo, promoting_colony_goal, confidence). These three layers allow sharing intelligence while controlling attack surface.

4. **Project-scoped vs unified identity** — RESOLVED as "both, separated." QUEEN.md = project-scoped wisdom (already supports global + local merge [S71]). USER.md = unified identity at hub level. Validated independently by OpenClaw (Q1, S69) and Hermes (Q5, S70).

5. **Build separation vs atomic build-advance** — NOT a contradiction. Aether's deliberate human-in-the-loop design choice [S121][S123] is correct for its target user (non-technical founder per CLAUDE.md). Other frameworks (CrewAI, AutoGen) target developer users who prefer atomic operations.

6. **Naive upfront vs JIT signal injection** — RESOLVED as hybrid approach per Q6's architecture spec. REDIRECTs always injected upfront (hard constraints must be visible). FOCUS/FEEDBACK injected based on domain-trail matching against current task. Aligns with Anthropic's guidance [S100].

7. **Shell sanitization vs prompt sanitization** — RESOLVED as defense-in-depth. Both needed. Current shell sanitization is correct for its purpose (prevents command injection in bash processing). Prompt-injection patterns (Q7 Tier 1) added as a separate validation layer. Different attack surfaces, different defenses.

### Remaining True Contradictions

1. **Self-editing memory vs externally-managed memory** — Letta agents self-edit their memory during reasoning (agent autonomy, risk of poor curation). Mem0 extracts memories externally (consistent but loses agent judgment). Aether's instinct system is externally managed (threshold-based promotion). No clear resolution — depends on trust in agent quality. Worth monitoring as Aether matures.

2. **Single-strategy vs multi-strategy retrieval cost-benefit** — More strategies = higher accuracy (89.6% with 4 strategies vs ~58% with 1) but also higher complexity and latency. For Aether's local-first architecture, the right point may be 2-strategy (BM25 + vector, following OpenClaw) rather than 4-strategy (following Hindsight). Insufficient evidence to determine optimal cost-accuracy tradeoff.

3. **Promotion timing** — Collaborative Memory paper [S90] says fragments should be designated private/shared at creation time. Aether promotes retroactively (during seal). Both have merit — Aether's retroactive promotion with abstraction transformation may be the right hybrid, but there's no empirical evidence comparing the approaches.

### Consolidated Implementation Priority

Cross-referencing all questions, the implementation priority emerges:

1. **Complete one seal cycle** (Q5, Q2) — Exercise the promotion pipeline, populate QUEEN.md, validate eternal-store flow. Zero code changes needed — just operational exercise.

2. **Add prompt-injection sanitization** (Q7, Q6) — Highest-severity security gap, exploitable today. Add pattern detection to pheromone-write. Low complexity, high impact.

3. **Implement colony-prime token budget** (Q2, Q6, Q1) — Add `--max-tokens` to colony-prime with per-section budgets. Prerequisite for all context expansion features.

4. **Fix eternal promotion threshold** (Q6, Q7) — Check effective_strength (decayed) instead of original strength. One-line code change, significant quality gate improvement.

5. **Add content deduplication to pheromone-write** (Q6) — SHA-256 hash check against existing active signals before creating duplicates. Fixes observed real-world data quality issue [S101].

6. **Separate USER.md from QUEEN.md** (Q5, Q1, Q3) — Create ~/.aether/USER.md for communication preferences and decision patterns, following OpenClaw/Hermes pattern. Decouples user model from task wisdom.

7. **Semantic retrieval for instincts** (Q4, Q6, Q3) — SQLite + FTS5 for hybrid BM25 + simple vector search over instincts. Follow memsearch [S108] pattern. Zero external dependencies.

8. **Wire eternal memory reads into colony-prime** (Q3, Q5) — Add 8th section loading high-value signals from ~/.aether/eternal/memory.json. Lowest-friction cross-repo bridge.

9. **Trail-based pheromone namespacing** (Q6) — Extend tags into dot-notation trails. Enable domain-scoped injection for JIT signal matching.

10. **Implement in-conversation display plan** (Q8) — Already fully spec'd in .aether/dreams/ [S66]. ~87 lines of changes, low risk.

## Sources

All findings cite sources using [SN] notation referencing the source registry in plan.json. Total: 126 registered sources across 8 types:
- **codebase:** 35 sources (Aether source files, config, data)
- **blog:** 52 sources (technical analyses, reviews, industry surveys)
- **documentation:** 15 sources (official docs for OpenClaw, Claude Code, Windsurf, Cline, Mem0)
- **github:** 8 sources (repositories for OpenClaw, SBP, Agent-MCP, memsearch, awesome-tuis, awesome-openclaw-skills)
- **academic:** 5 sources (arXiv papers on memory poisoning, temporal knowledge graphs, adaptive memory admission, collaborative memory, Mem0)
- **official:** 4 sources (Anthropic, K9s, Mem0, ContextForge)
- **forum:** 2 sources (Cursor community)

Source quality assessment:
- **Strongest evidence:** Codebase sources (direct verification), academic papers (peer review), official documentation (authoritative)
- **Good evidence:** GitHub repositories (verifiable code), technical blogs with code examples
- **Weaker evidence:** Review/comparison blogs (may have recency bias or promotional intent)

## Last Updated
Iteration 17 (synthesis pass, confidence recalibration via cross-question validation) — 2026-03-20T20:00:00Z
