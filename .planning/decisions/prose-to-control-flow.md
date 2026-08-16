# Prose-to-control-flow sites — per-site dispositions

**Started:** 2026-08-16 (reconciled the same day: most TYPED work had already shipped in the pre-restoration hardening with tests in `cmd/typed_phase_mode_test.go`, unrecorded in the planning ledger). TYPED-05 requires every site where words in prose
drive runtime control flow to carry a written decision: **fix** (convert to a
typed signal), **accept** (keyword use is legitimate, reason recorded), or
**defer** (named owner).

| Site | What the prose decides | Disposition |
|---|---|---|
| `cmd/oracle_loop.go:88` (`inferOracleTemplate`) | research template from topic wording | **Fixed 2026-08-16 (oracle reinstatement):** surfaced as a labelled suggestion via `aether oracle propose`; never silently applied — the operator approves the brief. Matches Phase 167 criterion 4's prescription. |
| `cmd/oracle_loop.go:154` (`inferOracleAutoScope`) | research scope from topic wording | **Fixed 2026-08-16:** same propose-and-approve path as above. |
| `cmd/plan_grounding.go:40` | phase named "research" skips the grounding gate | **Fixed (pre-restoration hardening):** gate reads the typed `mode`; keyword check survives only as a legacy fallback for pre-typed phases. Locked by `TestGroundingGateUsesTypedModeNotKeywords`. |
| `cmd/review_depth.go:153` | review depth from phase-name keywords | **Fixed (pre-restoration hardening):** keywords may only ESCALATE depth (never lower it), always announce themselves with a printed note, and `--light` overrides — the labelled-suggestion contract, satisfied. |
| `cmd/queen_spawn_budget.go` (`effectiveQueenPhaseMode`) | falls back to inferring mode from wording | **Fixed (pre-restoration hardening, TYPED-04):** runtime inference removed — neutral default for unmoded phases; modes written at authoring (`resolveAuthoredPhaseMode`, with a labelled note when defaulting) and backfilled by `aether migrate-state`. Locked by `TestEffectiveQueenPhaseModeNeverInfersFromProse`, `TestMigrateStateBackfillsPhaseModes`, `TestResearchWordedProductionPhaseDispatchesBuilder`, and the invariant `TestInferPhaseModeCallSitesAreBounded`. |
| `cmd/recovery_engine.go:46` | recovery keywords classify failures | **Accept:** classifying *error text* by keywords is the only signal available for a failure message; the classification table is code-level, deterministic, and tested (`failureClassifications`). This is prose *about* failures, not prose steering healthy control flow. |
| `cmd/codex_continue.go:3499` | learning-stage keyword handling | **Accept:** operates on worker-reported learning text where wording is the data itself, not a switch on phase intent. |
| `cmd/hive.go:209` | domain keywords scope hive wisdom | **Accept:** domain *tags* are the intended matching mechanism for cross-colony wisdom; they are explicit metadata, not incidental prose. |
| `cmd/codex_visuals.go:222` (command emoji/keyword hints) | display flavor from command names | **Accept:** presentation-only; no behaviour rides on it. |
| Specialist-summoning keywords (`queenPhaseHasSecuritySignal` etc.) | security caste from phase wording | **Fixed pre-restoration (Phase 182, 2026-08-15):** narrowed to genuine security surfaces, both directions asserted. |

Rule going forward: a new site that switches runtime behaviour on free-text
wording needs an entry here before it merges, or a typed field instead.
