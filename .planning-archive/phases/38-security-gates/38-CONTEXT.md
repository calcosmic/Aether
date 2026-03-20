# Phase 38: Security Gates - Context

**Gathered:** 2026-02-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Integrate Gatekeeper and Auditor agents into the `/ant:continue` verification workflow. Gatekeeper handles supply chain security (CVEs, licenses). Auditor handles code quality review. Both agents already exist in `.opencode/agents/` — this phase wires them into the colony workflow.

**Out of scope:** Creating new agents, modifying agent behavior, new commands.

</domain>

<decisions>
## Implementation Decisions

### Spawn Triggers
- Gatekeeper spawns when `package.json` exists in project root
- If no package.json, Gatekeeper skips gracefully (no error, just a note)
- Auditor spawns on every `/ant:continue` (Claude's discretion — simpler, consistent coverage)

### Blocking Severity
- **Critical CVEs:** Hard block — no override, must fix to continue
- **High CVEs:** Warn and continue, log to midden for later review
- **Auditor quality score < 60:** Hard block — no override
- **Auditor critical findings:** Hard block — no override

### Integration Point
- Security gates run **after Watcher verification** (Step 1.8 area)
- Sequential order: Gatekeeper first (supply chain), then Auditor (code quality)
- Both gates must pass before phase can advance
- Replaces existing basic grep security scan (Step 1.5, Phase 5)

### Override Behavior
- Hard blocks (critical CVEs, low quality score) have no user-facing override
- Non-code phases (docs-only) can skip security gates entirely
- No `--skip-security` flag — intentional security discipline
- Manual override possible via editing COLONY_STATE.json if truly needed

### Agent Constraints
- Both Gatekeeper and Auditor are strictly read-only agents
- Neither will modify code, create files, or update colony state
- If asked to modify: refuse and suggest appropriate agent (Builder, Tracker)

### Claude's Discretion
- Auditor spawn frequency (decided: always spawn for consistent coverage)
- Exact threshold values (60 for quality, could adjust later)
- Error message wording for blocked continues

</decisions>

<specifics>
## Specific Ideas

- Current security scan in `/ant:continue` is just a grep for secrets — this replaces it with professional CVE scanning
- Integration should feel natural, not bolted on — reuse existing verification loop patterns
- Both agents use JSON output format — can be parsed for gate decisions

</specifics>

<deferred>
## Deferred Ideas

- **Emoji consistency investigation** — User reported missing emojis in other repos using latest Aether. May need separate phase to investigate distribution/regression issue.
- **Support for other package manifests** (requirements.txt, Cargo.toml, go.mod, etc.) — Start with package.json only, extend later if needed
- **Configurable severity thresholds** — Hardcoded for now, could become user-configurable in future

</deferred>

---

*Phase: 38-security-gates*
*Context gathered: 2026-02-22*
