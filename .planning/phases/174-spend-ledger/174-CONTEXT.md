# Phase 174: Spend Ledger - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning
**Source:** Owner statement + four-perspective alignment review (owner-approved reshape, recorded in ROADMAP.md "Reshape (2026-08-13)")

<domain>
## Phase Boundary

Token usage is measured rather than asserted, persists past the process, reaches
the wrapper path (the only path the owner runs), separates estimates from
measurements, records parent attribution, and surfaces to the operator as one
plain end-of-build line. **This is a token instrument, not a billing
instrument** — the owner is on subscription billing and pays $0 per call;
tokens are the unit that draws against their subscription limits (they hit one
mid-session on 2026-08-13).

</domain>

<decisions>
## Implementation Decisions

### Dollars vs tokens (owner-decided via reshape approval)
- **D-01: Dollars never headline, anywhere.** The report and the end-of-build
  line lead with tokens. A dollars figure on a subscription plan invites the
  wrong conclusion ("this cost $3" when it cost nothing).
- **D-02: No model-price table is ever built.** Computing USD for rows the
  provider didn't price is API-billing machinery that rots and serves nobody
  here. USD is stored only when the provider itself reported it
  (`pkg/codex/usage.go` already parses `USDCost` in that case) and may appear
  only as a skippable footnote labelled as hypothetical API price.

### Where the operator meets the number (review finding, owner-approved)
- **D-03: The primary surface is one plain-English line at the end of every
  build/continue** — "This phase used ~N tokens (measured)" or "(partly
  estimated)". A non-technical operator does not run inspection commands;
  without this line the ledger is a dashboard nobody reads (SPEND-08).
- **D-04: `aether spend` is the detail view**, per-worker tokens and tool
  calls, byte-identical on repeated runs (inspection mutates nothing — this
  repo shipped dry-run commands that mutated state for months; the test earns
  its keep).
- **D-05: Workers appear under their colony identity** (caste emoji + name,
  e.g. "🔨 Builder Mason-67"), consistent with Phase 173's non-technical
  readability requirement (its D-14).

### Measurement honesty on the wrapper path (review-flagged gap, locked)
- **D-06: A token figure relayed by the orchestrating LLM through the results
  JSON is an assertion, not a measurement.** The wrapper-path figure must
  either come from a genuine artifact (e.g., a per-session transcript the
  runtime reads) or be stored tagged as non-provider-grade. It must never
  carry the `provider` source tag. Research should determine what genuine
  local sources exist; if none, the tagged-assertion route is acceptable and
  the tag must surface in the report.

### Roll-up scope (trimmed by reshape)
- **D-07: Parent attribution per row + a no-double-count invariant, not the
  dual-column accountant view.** Every ledger row records its parent (linkage
  `spawn-log` already stores); the grand total equals the sum of worker rows,
  asserted as an invariant. The full self/subtree two-column table is deferred
  until worker delegation actually exists (Phase 177 grants it).

### Claude's Discretion
- Ledger file format, location, and retention (suggest: per-run files kept
  modestly, e.g. current colony's runs; prune with existing data-clean paths).
- How the end-of-build line handles a mixed measured/estimated run (the
  "(partly estimated)" wording is a suggestion, not a mandate).
- Whether tool-call counts appear in the end-of-build line or only in
  `aether spend` (suggest: only in the detail view — keep the line short).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing measurement machinery (source of truth — extend, do not rebuild)
- `pkg/codex/usage.go` — WorkerUsage struct, ParseUsage, cache-token
  arithmetic (disjoint counts, billedTotal), EstimateUsage, provider-vs-
  estimate Source tagging. The parsing problem is SOLVED here with tests;
  this phase is plumbing around it.
- `pkg/codex/usage_test.go` — the 186x-undercount regression test asserting
  the documented 50/100000/2000/500 → 102,550 example.

### The gap this phase closes
- `cmd/codex_build_finalize.go` — `codexExternalBuildWorkerResult` (the
  wrapper-path result struct with NO usage field; SPEND-02's target).
- `cmd/colony_prime_context.go` — `colonyPrimeBudgetChars` (the character
  budget that must never feed a spend figure; SPEND-07).
- `CLAUDE.md` "## Token Budget" section — the char budget documented under a
  token name; SPEND-07 renames it.

### Parent linkage for roll-up
- `pkg/agent/spawn_tree.go` — the spawn ledger with ParentName recording
  (Phase 173), the linkage D-07's attribution walks.

### Scope authority
- `.planning/ROADMAP.md` § "Phase 174" (slimmed criteria, 2026-08-13) and
  § "Reshape (2026-08-13)" (why the phase is shaped this way).

</canonical_refs>

<specifics>
## Specific Ideas

- The owner's framing, verbatim in substance: they use subscription-based
  AI, not API billing — the whole point of measurement is supporting "the
  appropriate agents and skills selected by the orchestration of the Queen…
  in order to use cheaper models more effectively", and Phase 179's PROOF-03
  before/after comparison consumes this phase's output.
- Standing constraint (2026-07-28 decision, reaffirmed): no automatic model
  selection — measurement informs the operator and the proof, it never
  triggers model switching.

</specifics>

<deferred>
## Deferred Ideas

- Self/subtree dual-column spend view — revisit after Phase 177 (Worker
  Delegation Grant) makes real delegation trees common.
- Spend thresholds/alerts ("warn me at N tokens") — new capability, not in
  any SPEND requirement; belongs in a future phase if ever.

### Reviewed Todos (not folded)
- `2026-08-01-finalize-reconcile-task-evidence-gate.md` — evidence-gate
  bookkeeping, unrelated to spend measurement.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — timeout
  configurability, unrelated to spend measurement.

</deferred>

---

*Phase: 174-spend-ledger*
*Context gathered: 2026-08-13 via owner discussion + multi-agent alignment review*
