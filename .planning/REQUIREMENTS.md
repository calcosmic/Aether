# Requirements: Aether v5.4

**Defined:** 2026-04-01
**Core Value:** The system must reliably interpret a user request, decompose it into executable work, verify outputs, and ship correct work with minimal user back-and-forth.

## v5.4 Requirements

Requirements for the Shell-to-Go rewrite. Each maps to roadmap phases.

### Storage

- [ ] **STOR-01**: Go reads and writes all existing JSON files (COLONY_STATE, pheromones, learnings, instincts, flags, constraints, midden) with exact field compatibility -- round-trip tests prove parity
- [ ] **STOR-02**: Atomic writes via temp+rename match shell behavior -- no partial writes on crash
- [ ] **STOR-03**: JSONL append/read for event queue matches existing format -- blank lines skipped

### Events

- [ ] **EVT-01**: Channel-based event bus publishes and subscribes to typed events -- handlers receive events in <1us
- [ ] **EVT-02**: JSONL persistence alongside channels for crash recovery -- events survive process restart
- [ ] **EVT-03**: TTL-based event pruning removes expired events -- matches shell prune behavior

### Memory

- [ ] **MEM-01**: Trust scoring calculates identical results to shell (ADR-002 algorithm) -- 40/35/25 weighted, 60-day half-life, 7 tiers
- [ ] **MEM-02**: Observations are captured and stored in learnings -- auto-promotion triggers at trust >=0.50 or 3+ similar patterns
- [ ] **MEM-03**: Instincts are promoted from learnings and stored with provenance -- graph edges link source to instinct
- [ ] **MEM-04**: QUEEN.md promotion bridges high-confidence instincts (>=0.75, 3+ applications) -- format matches existing QUEEN.md sections
- [ ] **MEM-05**: Phase-end consolidation runs trust decay, archives low-trust, and checks promotions -- matches shell behavior

### Graph

- [ ] **GRAPH-01**: In-memory directed graph supports nodes (learning, instinct, queen, phase, colony) and 16 edge types
- [ ] **GRAPH-02**: 1-hop and 2-hop neighbor queries return connected nodes -- matches jq graph behavior
- [ ] **GRAPH-03**: Shortest path (BFS) and cycle detection work on the instinct graph
- [ ] **GRAPH-04**: Graph persists to JSON and loads from JSON -- round-trip compatible

### Agents

- [x] **AGENT-01**: Agent interface defines Name, Caste, Triggers, Execute -- all agents implement this interface
- [ ] **AGENT-02**: Worker pool manages concurrent agent execution with bounded goroutines (errgroup with SetLimit)
- [ ] **AGENT-03**: Spawn tracking records running agents in spawn tree -- matches spawn-tree.txt format
- [ ] **AGENT-04**: Curation ants (8) implement event subscriptions and handle memory events -- matches shell ant behavior

### LLM

- [ ] **LLM-01**: Anthropic Go SDK integration sends messages and receives responses -- supports Claude calls from Go agents
- [ ] **LLM-02**: Streaming responses accumulate SSE events into complete messages -- matches Claude streaming behavior
- [ ] **LLM-03**: Tool use loop detects ToolUseBlock, executes tools, returns ToolResultBlock -- agentic pattern
- [x] **LLM-04**: Agent spec YAML frontmatter is parsed into Go AgentConfig structs -- model, tools, triggers resolved

### CLI

- [ ] **CLI-01**: Cobra root command with all 37 subcommands registered -- `aether init`, `aether build`, etc.
- [ ] **CLI-02**: Shell completion generated for bash/zsh/fish -- matches existing command UX
- [ ] **CLI-03**: Status command displays colony dashboard -- output matches shell status output
- [ ] **CLI-04**: All read-only commands (status, phase, flags, history, pheromones, memory-details) produce identical output to shell

### XML Exchange

- [ ] **XML-01**: Pheromone export/import produces valid XML matching existing XSD schemas
- [ ] **XML-02**: Wisdom export/import with confidence threshold filtering -- matches shell behavior
- [ ] **XML-03**: Registry export/import with lineage tracking -- max depth 10
- [ ] **XML-04**: JSON<->XML conversion round-trips without data loss

### Distribution

- [ ] **DIST-01**: Single Go binary replaces npm package -- `go install` works
- [ ] **DIST-02**: Cross-platform builds for linux/darwin/windows (amd64/arm64)
- [ ] **DIST-03**: `aether --version` reports version matching npm package version

### Testing

- [ ] **TEST-01**: All existing shell test cases ported to Go -- parity verified against shell output
- [ ] **TEST-02**: Race detector passes on all concurrent code -- `go test -race` clean
- [ ] **TEST-03**: Golden file tests compare Go output against shell-produced JSON fixtures

## Future Requirements

Deferred to future milestones.

### Polish (v5.5+)

- **PLSH-01**: Chamber specialization (code zones: fungus garden, nursery, refuse pile)
- **PLSH-02**: Colony Constitution (self-critique principles)
- **PLSH-03**: Worker Quality Scores (reputation system)
- **PLSH-04**: Colony Sleep (memory consolidation during pause)
- **PLSH-05**: SQLite migration when JSON files become unwieldy

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| gonum/graph dependency | Custom implementation sufficient for Aether's needs (BFS + cycles) |
| SQLite in v5.4 | JSON files work fine; migrate when data volume demands it |
| Multi-provider LLM abstraction | Only Anthropic is used -- provider interface is sufficient |
| npm backward compatibility | Go binary replaces npm package entirely |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| STOR-01 | Phase 45 | Pending |
| STOR-02 | Phase 45 | Pending |
| STOR-03 | Phase 45 | Pending |
| EVT-01 | Phase 46 | Pending |
| EVT-02 | Phase 46 | Pending |
| EVT-03 | Phase 46 | Pending |
| MEM-01 | Phase 47 | Pending |
| MEM-02 | Phase 47 | Pending |
| MEM-03 | Phase 47 | Pending |
| MEM-04 | Phase 47 | Pending |
| MEM-05 | Phase 47 | Pending |
| GRAPH-01 | Phase 48 | Pending |
| GRAPH-02 | Phase 48 | Pending |
| GRAPH-03 | Phase 48 | Pending |
| GRAPH-04 | Phase 48 | Pending |
| AGENT-01 | Phase 49 | Complete |
| AGENT-02 | Phase 49 | Pending |
| AGENT-03 | Phase 49 | Pending |
| AGENT-04 | Phase 49 | Pending |
| LLM-01 | Phase 49 | Pending |
| LLM-02 | Phase 49 | Pending |
| LLM-03 | Phase 49 | Pending |
| LLM-04 | Phase 49 | Complete |
| CLI-01 | Phase 50 | Pending |
| CLI-02 | Phase 50 | Pending |
| CLI-03 | Phase 50 | Pending |
| CLI-04 | Phase 50 | Pending |
| XML-01 | Phase 51 | Pending |
| XML-02 | Phase 51 | Pending |
| XML-03 | Phase 51 | Pending |
| XML-04 | Phase 51 | Pending |
| DIST-01 | Phase 51 | Pending |
| DIST-02 | Phase 51 | Pending |
| DIST-03 | Phase 51 | Pending |
| TEST-01 | Phase 51 | Pending |
| TEST-02 | Phase 51 | Pending |
| TEST-03 | Phase 51 | Pending |

**Coverage:**
- v5.4 requirements: 37 total
- Mapped to phases: 37
- Unmapped: 0

---
*Requirements defined: 2026-04-01*
*Last updated: 2026-04-01 after roadmap creation*
