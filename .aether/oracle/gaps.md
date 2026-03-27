# Knowledge Gaps

## Open Questions
All 5 questions are now marked as **answered**. Remaining unknowns are operational metrics requiring runtime monitoring — static code analysis has reached its ceiling.

- **Q1 (85%)** — All 13 components mapped with deep responsibility boundaries. CLI vs. aether-utils.sh boundary confirmed clean. Colony-prime orchestration fully traced. Subcommand criticality mapped (43 critical path, 76 dead code). Agent interaction model documented. All contradictions resolved (dead-code dispatch, CLI/bash race).
- **Q2 (82%)** — Complete build/continue data flow traced through 9 playbooks. Concurrent write protection analysis across 5 shared files. Race conditions identified with severity. POSIX append safety confirmed for practical write sizes.
- **Q3 (82%)** — 11 risk areas documented with deep findings. Historical evidence from 17 midden entries validates 5+ risks. All risks characterized with severity. 9/10 have dedicated Q5 recommendations; 2 low-priority risks mitigated by existing design.
- **Q4 (82%)** — 14 findings covering complete memory system from input to output. Context capsule resilience confirmed as strongest defense pattern. Memory-capture sequential kill-switch fully characterized. All contradictions resolved.
- **Q5 (80%)** — 12 findings: 9 recommendations with cross-question evidence, priority matrix, feasibility validation, and coverage analysis. All critical/medium risks addressed. Implementation order and dependencies documented.

## Contradictions

### Resolved (9)
1. **Decision pheromone double-emission**: Both paths emit identical format; dedup catches it via `.contains()` check [S66].
2. **"Evidence before claims" vs schema-only validation**: Structural minimum vs aspirational behavior — both serve different purposes [S17, S18].
3. **Staleness detected but never acted upon**: Deliberate design — "Restore identically regardless of time elapsed" [S33].
4. **Memory-capture "fire-and-forget" vs sequential kill-switch**: Step 1 failure kills all 5 downstream steps [S68, S69].
5. **Dead-code indirect callers via eval/dynamic dispatch**: Case-statement dispatch, not eval [S1]. Grep covers all static callers [S73].
6. **CLI/bash hub file race**: Temporally disjoint — CLI during install, bash during operations [S71, S1].
7. **Dual file-lock implementations**: Temporal separation mitigates cross-runtime issue [S71, S1]. If code fix needed: mkdir-based locking.
8. **"Security gate" vs actual detection scope**: Documentation accuracy issue [S21]. Label oversells; fix is documentation, not code.
9. **State-safety skill prescribes backups that aren't implemented**: Confirmed gap [S39, S43]. Q5 Rec 1 provides the fix.

### Open (0)
All contradictions resolved.

## Discovered Unknowns (Remaining — Operational Metrics Only)
These unknowns require runtime monitoring or deployment testing to resolve. Static code analysis has reached its ceiling:
- What is the actual concurrent write collision rate during parallel builds?
- How often does colony-prime budget trimming actually trigger in practice?
- What is the frequency of memory-capture corruption (learning-observations.json)?
- What percentage of worker spawns fall back to general-purpose agents in production?
- What is the actual failure rate within the suggest-analyze ERR trap gap?
- How does run-state.json desync with COLONY_STATE.json manifest to the user?

## Cross-Question Patterns (Synthesis — 6 Patterns)
1. **Documentation accuracy problem**: 6 instances where labels don't match behavior
2. **COLONY_STATE.json vulnerability chain**: 3 questions converge on inconsistent protection
3. **Three-layer error silence**: Systematic error suppression creating invisible failures
4. **Healthy architecture strengths**: Agent isolation, context capsule resilience, pheromone dedup, CLI/bash boundary, tiered resume, REDIRECT preservation
5. **Operational ceiling**: Most remaining unknowns require runtime data, not further code analysis
6. **Recommendation coverage**: 9 recs cover all critical/medium risks; 2 low-priority risks mitigated by design

## Last Updated
Iteration 11 — 2026-03-23T17:30:00Z (final synthesis: resolved all 9 contradictions, all questions marked answered at analytical ceiling)
