---
phase: 148
name: Learning and Workflow Restoration
goal: The learning extraction lifecycle works end-to-end (hypothesis, validated, disproven), workers receive full context, and all 9 flagship workflows produce correct state and ceremony
milestone: v1.23
requirements: [WORKFLOW-01, WORKFLOW-02, WORKFLOW-03, WORKFLOW-04, WORKFLOW-05, WORKFLOW-06, WORKFLOW-07, WORKFLOW-08, WORKFLOW-09, WORKFLOW-10, WORKFLOW-11]
---

# Phase 148: Learning and Workflow Restoration

## Success Criteria
1. After running `/ant-continue`, learnings are extracted as hypotheses, tracked with evidence, and promoted to instincts with validated/disproven status
2. Workers spawned during build receive pheromone signals, matched skills, survey data, colony goal, and phase description in their context
3. Oracle findings flow end-to-end: research results become instincts, instincts become learnings, learnings promote to QUEEN.md, and high-confidence instincts reach the Hive Brain
4. All 9 flagship workflows (build, continue, plan, colonize, autopilot, seal, entomb, swarm, oracle) produce correct state mutations and ceremony output when run sequentially
5. Autopilot (`aether run`) respects all 10 Classic pause conditions

## Plans

| Plan | Wave | Depends On | Focus |
|------|------|------------|-------|
| 148-01 | 1 | - | Deep learning extraction in continue-finalize |
| 148-02 | 1 | - | Hypothesis lifecycle (hypothesis/validated/disproven) |
| 148-03 | 2 | 148-01, 148-02 | Worker context audit for continue/plan/colonize |
| 148-04 | 3 | 148-01, 148-02, 148-03 | Autopilot pause conditions + 9 workflow verification |

## Key Risks
- Changing `learn.Entry` struct may break existing JSON on disk
- Adding pause conditions to autopilot may break existing autopilot tests
- Continue/plan/colonize may use different dispatch struct types than build
- Ceremony emissions may conflict with visual output mode
