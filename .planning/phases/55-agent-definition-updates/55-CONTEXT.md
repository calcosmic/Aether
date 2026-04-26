# Phase 55: Agent Definition Updates - Context

**Gathered:** 2026-04-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Seven review agents (Gatekeeper, Auditor, Chaos, Watcher, Archaeologist, Measurer, Tracker) gain Write tool access and findings persistence instructions so they can write structured findings to their domain review ledgers. Write-scope guardrails restrict them to ONLY write to designated ledger files under `.aether/data/reviews/`. All 7 agents synced across 4 surfaces (28 files). Dispatch flows inject findings-path instructions at task assignment time.

</domain>

<decisions>
## Implementation Decisions

### Write-scope guardrail enforcement
- **D-01:** Two-layer guardrails: agent body text carries permanent boundary instructions ("only write to .aether/data/reviews/{domain}/"), dispatch injection adds concrete path at task time
- **D-02:** No runtime validation in Go code — relies on LLM following instructions, same trust model as existing "no Bash" rules for Gatekeeper
- **D-03:** Guardrails explicitly list what agents MUST NOT write to (source code, tests, colony state, dreams, .env) and the single exception (review ledger files)

### Dispatch prompt injection
- **D-04:** Findings-path instructions injected at dispatch time in Go code, NOT embedded in agent body
- **D-05:** Agent body has generic guardrails (what NOT to write), dispatch adds the specific path (e.g., "Write findings to `.aether/data/reviews/security/` using `review-ledger-write`")
- **D-06:** Go code changes required in build dispatch (`codexBuildSpecialistDispatch` for Watcher, Chaos, Measurer, Archaeologist) and continue dispatch (`codexContinueReviewSpecs` for Gatekeeper, Auditor, Probe)

### Tracker's role boundary
- **D-07:** Specific carve-out for Tracker: replace "no Write tool" with "Write tool restricted to `.aether/data/reviews/` only"
- **D-08:** Update Tracker boundary from "Do not write to `.aether/data/`" to "`.aether/data/` except `reviews/bugs/`"
- **D-09:** Preserve Tracker's diagnose-only identity — Write is for persisting findings only, never for applying fixes

### Mirror sync strategy
- **D-10:** Single plan handles all 7 agents across all 4 surfaces (28 files) — changes are mechanical, one coherent pass easier to verify parity
- **D-11:** Surfaces: `.claude/agents/ant/` (canonical), `.aether/agents-claude/` (mirror), `.opencode/agents/` (mirror), `.codex/agents/` (TOML mirror)

### Claude's Discretion
- Exact wording of findings write instructions in agent bodies
- Exact format of dispatch injection text in Go code
- Whether to add a `findings_section` to the agent return format or rely on dispatch instructions
- Error handling when agents write to wrong paths

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Agent definitions (canonical — edit these, mirrors follow)
- `.claude/agents/ant/aether-gatekeeper.md` — Gatekeeper: security domain, currently "no Bash no Write"
- `.claude/agents/ant/aether-auditor.md` — Auditor: quality/security/performance domains
- `.claude/agents/ant/aether-chaos.md` — Chaos: resilience domain, currently "no Write or Edit tools by design"
- `.claude/agents/ant/aether-watcher.md` — Watcher: testing/quality domains
- `.claude/agents/ant/aether-archaeologist.md` — Archaeologist: history domain
- `.claude/agents/ant/aether-measurer.md` — Measurer: performance domain
- `.claude/agents/ant/aether-tracker.md` — Tracker: bugs domain, has specific "no Write" and "do not write to .aether/data/" boundaries

### Dispatch flow (Go code — add findings-path injection)
- `cmd/codex_build.go` §`codexBuildSpecialistDispatch` (line ~660) — Build dispatch for specialist agents (Watcher, Chaos, Measurer, Archaeologist)
- `cmd/codex_continue.go` §`codexContinueReviewSpecs` (line ~793) — Continue review dispatch specs (Gatekeeper, Auditor, Probe)
- `cmd/codex_build.go` §`expectedDispatchOutcome` (line ~1485) — Expected outcome strings per caste

### Review ledger system (target for agent writes)
- `cmd/review_ledger.go` — `review-ledger-write`, `review-ledger-read`, `review-ledger-summary`, `review-ledger-resolve` subcommands
- `pkg/colony/review_ledger.go` — `ReviewLedgerEntry`, `ReviewLedgerFile`, `DomainOrder`, `ReviewSeverity` types

### Requirements
- `.planning/REQUIREMENTS.md` AGENT-01 through AGENT-10 — The 10 locked requirements for this phase

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Agent frontmatter pattern: `tools: Read, Bash, Grep, Glob` line in YAML frontmatter — add `Write` to this list for each agent
- Boundary sections in agent bodies: `<boundaries>` XML tags with explicit declarations — add write-scope restrictions here
- Dispatch task injection: `codexBuildSpecialistDispatch` and `codexContinueReviewSpecs` already construct task descriptions per agent — append findings-path text here
- Mirror sync is mechanical: `.aether/agents-claude/` must be byte-identical to `.claude/agents/ant/`, `.opencode/agents/` structural parity, `.codex/agents/` TOML translation

### Established Patterns
- Agent definitions use XML sections: `<role>`, `<execution_flow>`, `<critical_rules>`, `<boundaries>`, `<return_format>`
- Write-scope boundary is new — no existing agent has restricted Write access (Builder has unrestricted Write)
- Guardrail pattern: explicit "MUST NOT" and "NEVER" language in `<boundaries>` and `<critical_rules>` sections

### Integration Points
- Build dispatch creates specialist dispatches at lines 581-651 in `codex_build.go` — Watcher, Chaos, Measurer, Archaeologist, Probe
- Continue dispatch creates review specs at lines 793-796 in `codex_continue.go` — Gatekeeper, Auditor, Probe
- Tracker is NOT dispatched in build or continue currently — it's an on-demand agent invoked via `/ant-swarm` — its findings write happens during swarm investigations

</code_context>

<specifics>
## Specific Ideas

- The findings write instructions should reference `aether review-ledger-write` CLI command so agents use the existing CRUD infrastructure (not raw file writes)
- Each agent should include a JSON example of a findings write command in its body so the LLM knows the exact format
- Tracker's carve-out should be visible — "Write is available ONLY for persisting findings to the bugs domain ledger. You still do not fix, modify, or apply changes to source code."

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 55-agent-definition-updates*
*Context gathered: 2026-04-26*
