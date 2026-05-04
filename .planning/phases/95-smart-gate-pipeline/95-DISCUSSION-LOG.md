# Phase 95: Smart Gate Pipeline - Discussion Log

**Date:** 2026-05-03
**Mode:** Default (interactive)

## Areas Discussed

### 1. Auto-resolve Criteria
**Q1: How should the queen decide whether to auto-resolve?**
- Options: Threshold-based, History-based, Hybrid
- **Selected: Threshold-based** — Simple numeric check, no LLM judgment needed

**Q2: Per-gate thresholds or global?**
- Options: Per-gate thresholds, Single global threshold
- **Selected: Per-gate thresholds** — Each soft_block gate has its own scale

**Q3: Hardcoded constants or configurable?**
- Options: Hardcoded constants, Configurable per-colony
- **Selected: Hardcoded constants** — Deterministic, follows gateClassifications pattern

**Q4: Where does auto-resolve live?**
- Options: Inside continue command, Separate command
- **Selected: Inside continue** — No new commands for the resolve flow

### 2. Threshold Configuration
**Q1: CLI inspection command?**
- Options: Yes (gate-auto-resolve command), No (keep internal)
- **Selected: Yes** — Mirrors gate-classify pattern from Phase 93

**Q2: Tie to verification depth or separate flag?**
- Options: Tie to verification depth, Separate --auto-resolve flag
- **Selected: Tie to verification depth** — Leverages existing depth model, no new flags

### 3. Queen Annotation Flow
**Q1: Auto-resolve fails — dispatch Fixer?**
- Options: Auto-resolve OR Fixer dispatch, Auto-resolve only (no Fixer)
- **Selected: Auto-resolve OR Fixer dispatch** — Fixer attempts repair automatically

**Q2: Reuse QueenAnnotation or new struct?**
- Options: Reuse QueenAnnotation, New struct with more fields
- **Selected: Reuse QueenAnnotation** — Existing struct from Phase 93 is sufficient

**Q3: Same gate-results file or separate?**
- Options: Same gate-results file, Separate resolve log file
- **Selected: Same gate-results file** — In-place annotation via existing persistence

## Summary
- 10 questions asked across 3 areas
- All decisions captured in 95-CONTEXT.md
- No deferred ideas
- No scope creep

---

*Discussion completed: 2026-05-03*
