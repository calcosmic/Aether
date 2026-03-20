# Phase 34: Add User Approval UX - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Interactive CLI approval flow for promoting observations to permanent wisdom in QUEEN.md. Builds on Phase 33's proposal display to add the tick-to-approve mechanism, threshold override handling, and batch promotion execution.

**In scope:**
- Tick-to-approve interaction with checkbox-style selection
- Threshold enforcement display with override option
- queen-promote execution on batch approval
- Deferred proposal storage for non-approved items

**Out of scope:**
- Observation tracking (Phase 33)
- Wisdom extraction at seal/entomb (Phase 35)

</domain>

<decisions>
## Implementation Decisions

### Approval Interaction
- **Checkbox style** — Display uses `[ ]` brackets for unchecked items
- **Selection method** — User types numbers to select (e.g., "1 3 5" to tick boxes)
- **Post-selection flow** — Summary count ("3 proposals selected") then promote — no per-item confirmation
- **Zero selection** — If no selections made, keep all proposals for next run (don't discard)
- **Threshold override** — Users CAN approve proposals below threshold; show warning but allow selection

### Proposal Display
- **Grouping** — Proposals grouped by wisdom type (Philosophies, Patterns, Redirects, Stack Wisdom, Decrees)
- **Detail level** — Minimal: one line per proposal
  - Format: `[ ] 1. Philosophy: "Keep functions small" ●●●●● (5/5)`
- **Checkbox visual** — Bracket style `[ ]` for unchecked, `[x]` for selected
- **Threshold indicator** — Progress bar with filled circles (e.g., `●●●○○ 3/5`)
- **Verbose flag** — `--verbose` shows full content for proposals that need more context

### Batch vs Individual
- **Approval mode** — Batch approve all selected in one action
- **Error handling** — Stop on first error; show which succeeded before failure
- **Success feedback** — List each promoted item: `"✓ Promoted Pattern: Use colony-prime() for context"`
- **Undo** — Immediate undo prompt after promotion: `"Undo these promotions? (y/n)"`

### Rejected Proposals
- **Default behavior** — Defer to later, not discard
- **Storage** — learning-deferred.json (separate file, same format as observations)
- **Auto-represent** — Never auto-show deferred items in regular continue.md
- **Manual review** — `/ant:continue --deferred` shows deferred proposals with same approval UX

### Claude's Discretion
- Exact wording of summary/feedback messages
- Progress bar character choices (filled/unfilled)
- Error message format when promotion fails
- Whether to show "no proposals to review" message vs silent exit

</decisions>

<specifics>
## Specific Ideas

- Checkbox-style selection feels familiar from Git interactive staging (`git add -p`)
- Progress bar for threshold makes it visually clear which items are "ready" vs "early"
- Immediate undo prompt prevents regret without requiring separate command

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 34-add-user-approval-ux*
*Context gathered: 2026-02-20*
