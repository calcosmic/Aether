# Knowledge Gaps

## Evidence Ceiling Assessment

This audit has reached the achievable confidence ceiling for **static code review (~80% after cross-validation calibration, max ~84%)**. The remaining gaps fall into two categories that cannot be resolved through codebase analysis alone:

### Category 1: Runtime Testing Required
- Q4: Non-TTY visual rendering behavior (untested)
- Q6: Practical frequency of concurrent LLM+bash writes to COLONY_STATE.json
- Q6: Corrupted-JSON degradation path testing (unit-level)
- Q7: Claude behavior when HANDOFF.md is missing (requires runtime crash simulation)
- Q7: CONTEXT.md staleness impact during real non-build workflows

### Category 2: Design Intent Clarification Required
- Q1: Are dead-end commands (preferences, watch) intentionally terminal? (Likely yes — both are utility commands)
- Q3: Is the asymmetric touchpoint distribution deliberate "trust the system" design? (Evidence suggests yes — consistent pattern across 5 questions)
- Q4: Is per-command visual theming intentional, or accidental divergence? (Evidence suggests accidental — abandoned print-standard-banner)

## Open Questions (Confidence Calibrated — Iteration 10)

- **Q1 (82%):** Routing comprehensively mapped. Post-seal bug is the only issue. Cross-validated by Q2, Q3, Q7. **Ceiling: ~85%.**
- **Q2 (77%):** Full injection path traced. Agent protocol gap quantified. Cross-validated by Q6 (reliability) and Q3 (autonomous-first). **Ceiling: ~82%.**
- **Q3 (82%):** Complete touchpoint inventory. Autonomous-first pattern validated across 5 question tracks. **Ceiling: ~85%.**
- **Q4 (73%):** Visual inventory comprehensive for static analysis. Cross-validated by Q3 and Q6 (Pattern 3, 6 instances). **Ceiling: ~78%.**
- **Q5 (84%):** All 6 v1.1.11 features traced. 542 tests passing. Zero regressions. Cross-validated by Q1, Q2, Q6. **Ceiling: ~88%.**
- **Q6 (74%):** Risk areas identified. Pattern 3 validated across 6 independent instances in 4 questions. **Ceiling: ~78%.**
- **Q7 (75%):** Context management mapped. Dual-document disconnect identified as Pattern 4 instance. **Ceiling: ~80%.**
- **Q8 (90%):** 15 prioritized improvements, 4 cross-question patterns, 8 resolved contradictions. **Ceiling: ~95%.**

## Contradictions

All 8 contradictions identified during investigation have been resolved in synthesis.md. See "Resolved Contradictions" table. No new contradictions discovered during synthesis passes.

## Source Attribution

All findings have source_ids. Trust ratio: 91% (49 multi-source, 5 single-source, 0 unsourced). No findings missing source attribution.

Single-source findings (capped at 50% confidence contribution): Q1-F3 (S5 only), Q2-F3 (S16 only), Q5-F1 (S11 only), Q6-F3 (S28 only), Q6-F5 (S27 only). All are corroborated by multi-source findings on the same question, so the confidence impact is minimal.

## Last Updated
Iteration 10 — 2026-03-22T00:30:00Z (SYNTHESIS PASS — cross-validation confidence calibration)
