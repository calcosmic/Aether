# Phase 25: Queen Coordination - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Define how the colony handles failure escalation, codify 6 named workflow patterns in the Queen definition, and merge two overlapping agent pairs (Architect→Keeper, Guardian→Auditor). No new capabilities — this is internal coordination and consolidation.

</domain>

<decisions>
## Implementation Decisions

### Escalation chain behavior
- Colony should be very patient — worker retries, parent tries a different approach, Queen tries to reassign, user only hears about it if everything fails
- When escalation reaches the user: present options with recommendation ("We hit X. Here are 3 options: A (recommended), B, C.")
- Never skip failed tasks silently — every failure must be acknowledged, even if other tasks continue
- Distinct visual banner for escalation (━━━ ESCALATION ━━━ style) so user can't miss it in output
- Escalation state visible in /ant:status (e.g., "⚠️ 1 task escalated to Queen")

### Workflow patterns
- 6 named patterns confirmed: SPBV (Scout-Plan-Build-Verify), Investigate-Fix, Deep Research, Refactor, Compliance, Documentation Sprint
- Each pattern must include a rollback/reversal step — always, not just for risky patterns
- Colony announces which pattern it picked at the start of a build ("Using pattern: Investigate-Fix")
- User's engineering procedures (Plan→Patch→Test→Verify→Rollback and Symptom→Isolate→Prove→Fix→Guard) inform the pattern structure — each pattern should have defined phases with verification built in

### Agent merges
- Architect merges into Keeper with subtitle: "Keeper (Architect)" when doing architecture work
- Guardian folds into Auditor with subtitle: "Auditor (Guardian)" when doing security work
- Ant emoji caste identities preserved for all agents including merged ones
- Colony updates from 25 to 23 agents — update everywhere (caste-system.md, workers.md, all output, summaries, help text)
- Old agent files: Claude's discretion on delete vs redirect approach

### Colony feel
- Pattern announcement at build start (visible, not hidden)
- Escalation state in /ant:status
- All references updated to reflect 23-agent team — clean transition, not gradual

### Claude's Discretion
- Debug pattern structure (inspired by user's Symptom→Isolate→Prove→Fix→Guard but adapted to colony)
- Whether "Add Tests" becomes a 7th pattern or stays part of existing patterns
- Old agent file handling (delete vs keep as redirects)
- Exact escalation banner design

</decisions>

<specifics>
## Specific Ideas

- User shared engineering procedure templates that map well to colony workflow patterns — patterns should follow structured flows with verification and rollback built into every one
- User explicitly confirmed: Archaeologist, Dreamer, Chaos, and all non-merged agents stay exactly as they are — only Architect→Keeper and Guardian→Auditor are consolidated
- "Keeper (Architect)" and "Auditor (Guardian)" subtitle pattern for merged agent identity

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 25-queen-coordination*
*Context gathered: 2026-02-20*
