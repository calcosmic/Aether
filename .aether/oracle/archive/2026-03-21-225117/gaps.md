# Knowledge Gaps

## Open Questions
- **Q1 (65%)** — OpenClaw: Deep code-level understanding of memory (SQLite, hybrid search, embedding pipeline), skills (3-tier loading, frontmatter schema), heartbeat, and compaction. Cross-validated by Q4 landscape positioning (Tier 3), Q5 identity separation, Q6 decay model. Key adoptable patterns (6 identified) are each validated by 2+ other questions. **Remaining gaps:** (1) Multi-agent memory access — do concurrent agents share one SQLite index? (2) ClawHub marketplace governance for malicious skills (parallels Q7 security concerns). (3) Performance benchmarks (memory_search latency, indexing throughput). (4) Memory conflict handling (daily log vs MEMORY.md contradictions). (5) QMD sidecar backend details. *Note: remaining gaps are implementation specifics that don't undermine the practical adoption patterns.*
- **Q2 (75%)** — Aether architecture: Deep understanding of build wave mechanics, colony-prime 7-section assembly, agent dispatch mechanism (22 agents mapped to commands). Cross-question synthesis strengthened via Q6 token budget analysis and Q7 security surface mapping. **Remaining gaps:** (1) Context-capsule next_action routing edge cases. (2) Full traced memory-capture pipeline (single end-to-end flow). (3) Quality gate interaction sequence — does Gatekeeper blocking prevent Auditor? (4) Worker JSON response validation fallback behavior in practice. *Note: these are edge cases within a well-understood system.*
- **Q3 (75%)** — Cross-repo hive mind: Concrete hive data model specified with JSON schemas. Performance budget verified. Meta-repo pattern validates hub approach. Cross-validated with Q5 (QUEEN.md wiring), Q7 (security prerequisites), Q2 (colony-prime extensibility). **Remaining gaps:** (1) Domain tag assignment — manual vs auto-inferred during colonize. (2) Abstraction step reliability — can LLM reliably generalize repo-specific instincts? (3) Cold-start problem for untagged repos. (4) Migration from eternal-store to hive schema. *Note: gaps are design decisions and operational challenges, not conceptual unknowns.*
- **Q4 (78%)** — AI coding tools memory: Comprehensive landscape with deep technical architectures for Mem0, Letta/MemGPT, Zep/Graphiti, Windsurf Cascade. LoCoMo benchmarks quantified. Five-tier sophistication model validated across Q1/Q6/Q8. MCP memory server ecosystem documented. **Remaining gaps:** (1) Claude Code cron maintenance internals (lesson rotation, staleness detection). (2) MemOS architecture specifics. (3) Cost comparison across tools (accuracy-per-dollar). (4) Real-world adoption data — which memory approach do most teams actually use?
- **Q5 (76%)** — Queen evolution: Colony-prime wiring CONFIRMED (corrected iteration 4 error). Root cause of empty QUEEN.md identified (no seal cycle). Concrete 4-phase roadmap informed by OpenClaw (Q1) and Hermes patterns. Cross-validated three-way separation (identity/wisdom/user model) across Q1+Q3+Q5. Character/entry caps gap NARROWED: three precedents established (Claude Code 200-line [S20], OpenClaw 20K chars [S105], Hermes 2200/1375 chars [S70]). **Remaining gaps (narrowed):** (1) Character/entry caps — now a DESIGN DECISION, not a research gap (three precedents available). (2) USER.md data model specification — three templates available (OpenClaw, Hermes, Claude Code auto-memory) but Aether-specific fields TBD.
- **Q6 (77%)** — Pheromone evolution: Comprehensive 8-part architecture specification. SBP merge strategies mapped. Graphiti temporal supersession applied. ICLR 2026 adaptive admission integrated. Cross-validated by Q4 retrieval patterns, Q7 security analysis, Q2 colony-prime extensibility. **Remaining gaps:** (1) jq implementation of exponential decay (Taylor series feasibility verified but not tested). (2) Canonical trail taxonomy for Aether's known domains. (3) Lightweight fuzzy string matching for non-exact signal dedup in bash/jq. (4) Token estimation accuracy (content_length/4 is approximate).
- **Q7 (72%)** — Risks and dangers: Codebase-verified risk assessment with 6+ codebase sources. Prompt injection via pheromones identified as highest severity with 5+ sources. OWASP 5-control quantitative assessment (1.5/5). Cross-validated: Q6 eternal promotion bug amplifies concern; Q3 security model provides mitigation layers. Concrete 4-tier mitigation roadmap specified. **Remaining gaps:** (1) Empirical scan of existing pheromone content for actual injection risk. (2) Concrete regex patterns for prompt-injection detection in bash. (3) Token cost estimation methodology for per-worker budgets. (4) Whether 500-char limit meaningfully prevents prompt injection (analysis suggests NO — "Ignore all previous instructions" is only 35 chars). *Note: risk model is comprehensive; gaps are about specific mitigations, not risk identification.*
- **Q8 (60%)** — Visual/interactive patterns: Good framework landscape. Existing display infrastructure mapped. Cross-referencing with Q7 (scope creep) narrows recommendation to bash-native improvements. Cross-validated by Q2 (data model for display) and Q4 (display patterns). In-conversation display plan fully spec'd but unimplemented. **Remaining gaps (narrow):** (1) Accessibility of emoji-heavy display — no --plain flag exists (implementation item, not conceptual gap). (2) Implementation priority validated by consolidated 10-item list (Q8 visual work ranks #10).

## Contradictions

### Resolved (7)
1. **QUEEN.md wiring status** — Colony-prime actively loads QUEEN.md [S71]. Issue is empty data, not missing infrastructure. (Q5, corrected iteration 4 error)
2. **Linear vs exponential decay** — Exponential is better-supported by ICLR 2026 [S99], Entity-Fact [S30], SBP [S97]. Implementable in jq via Taylor series.
3. **Security vs intelligence** — Resolved as layered defense: prompt-injection sanitization + content abstraction before sharing + provenance tracking. (Q7 + Q3 + Q6)
4. **Project-scoped vs unified identity** — Both needed, separated: QUEEN.md = project-scoped wisdom, USER.md = unified identity at hub level. Validated by OpenClaw [S69] and Hermes [S70] independently.
5. **Build separation vs atomic advance** — Deliberate human-in-the-loop design, correct for non-technical user [S121][S123].
6. **Naive upfront vs JIT injection** — Hybrid: REDIRECTs always upfront, FOCUS/FEEDBACK via domain matching [S100].
7. **Shell vs prompt sanitization** — Defense-in-depth: both needed for different attack surfaces [S78].

### Unresolved (3)
1. **Self-editing memory vs externally-managed memory** — Letta (agent self-edits) vs Mem0 (external extraction) vs Aether (threshold-based promotion). No clear winner — depends on trust in agent quality.
2. **Single-strategy vs multi-strategy retrieval cost-benefit** — More strategies = higher accuracy but higher complexity. Optimal point for Aether's local-first architecture unclear (2-strategy like OpenClaw? 4-strategy like Hindsight?).
3. **Promotion timing — at creation vs retroactive** — Collaborative Memory paper [S90] says designate at creation; Aether promotes retroactively (during seal). Both have merit. No empirical comparison available.

## Cross-Question Themes (Synthesis)
1. **Empty Pipeline Problem** (Q2+Q5+Q3+Q6+Q7) — All future features bottlenecked by never-completed seal cycle
2. **Token Budget as Universal Constraint** (Q2+Q6+Q1+Q4+Q3) — No total cap on colony-prime injection
3. **Multi-Strategy Retrieval Consensus** (Q4+Q1+Q6+Q3) — Industry converged; Aether at Tier 0
4. **Security Scales with Intelligence** (Q7+Q3+Q6) — 1.5/5 OWASP controls; must improve before cross-repo
5. **Identity/Wisdom/User Model Separation** (Q1+Q5+Q3) — Three independent systems validate three-way split
6. **Bash Sufficient Today, Not Tomorrow** (Q7+Q8+Q6+Q2) — Current architecture works; cross-repo security and rich TUI need typed companion

## Gaps Resolved This Iteration (17)
*Previous iterations resolved 16 gaps. This iteration adds:*
- **Q5 gap: Character/entry caps** — NARROWED to design decision. Three independent precedents now available: Claude Code 200-line cap [S20][S23], OpenClaw 20K char workspace cap [S105], Hermes per-file caps (2200/1375 chars) [S70]. No further research needed — choose values.
- **Q8 gap: Implementation priority** — RESOLVED by consolidated 10-item cross-question priority list. Q8 visual work ranks #10, confirming it's important but not urgent relative to pipeline completion, security, and token budgeting.

## Discovered Unknowns
- OpenClaw's supply chain security for skills marketplace — relevant to any Aether template sharing
- learning-observations.json tracks colonies[] per observation — existing cross-colony mechanism within same repo
- Cron-based memory maintenance (Claude Code pattern) — Aether has no equivalent for data hygiene
- EU AI Act (Aug 2026) compliance pressure favoring local-first memory approaches [S113]
- Windsurf's 48-hour learning period and 78% accuracy provide baseline for auto-learning quality measurement
- OpenClaw's memsearch extraction proves memory architecture separability [S108]
- Colony-prime total injection size uncapped — can grow to degrade worker context
- Build-to-continue handoff files not verified for freshness at continue-time
- Worker response validation fallback is binary (pass/fail with no partial result extraction)
- Eternal promotion strength threshold uses original strength, not decayed — effectively a pass-through for REDIRECT and FOCUS signals [S95]

## Last Updated
Iteration 17 (synthesis pass, confidence recalibration) — 2026-03-20T20:00:00Z
