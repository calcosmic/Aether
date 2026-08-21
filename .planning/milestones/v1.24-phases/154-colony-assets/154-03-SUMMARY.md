# 154-03 Summary — Colony Assets: Policy YAMLs + Schema Extension

## What Changed

- Extended `control-ts/src/schemas/policy.schema.ts` to support 8 policy types:
  - `model_routing`, `memory_rules`, `skill_creation`, `safety_gates` (existing)
  - `dispatch_contract`, `pheromone_lifecycle`, `signal_rules`, `autopilot` (new)
- All new keys are optional with sensible defaults so existing tests pass without modification.
- Created 8 policy YAML files under `colony/policies/`:
  - `model-routing.yaml` — 27 routing rules (one per caste), default provider anthropic, opus for queen/oracle/gatekeeper/auditor/architect/route_setter/tracker/archaeologist/measurer/sage, sonnet for rest
  - `memory-rules.yaml` — max_learnings 100, auto_promote_threshold 0.75, learning_retention_days 60, instinct_cap 30, event_cap 100
  - `skill-creation.yaml` — allowed true, auto_approve false, max_skills_per_colony 50, require_wisdom_threshold 0.8
  - `safety-gates.yaml` — security_scan true, quality_gate true, chaos_scan true, auditor_score_threshold 60
  - `dispatch-contract.yaml` — max_workers_per_phase 10, spawn_depth_limits {0:4,1:4,2:2,3:0}, timeout_defaults {build:600, continue:300, plan:300, verify:180}
  - `pheromone-lifecycle.yaml` — default_ttl_days 30, auto_expire_on_phase_end true, max_active_signals 100, strength_decay_per_day 0.05
  - `signal-rules.yaml` — hard_constraint_prefixes ["[error-pattern]", "[redirect]"], auto_emit_on_phase_complete true, max_feedback_per_phase 3
  - `autopilot.yaml` — pause_conditions [test_failure, critical_chaos, security_gate_failure, quality_gate_failure, runtime_verification_needed], replan_interval 3, max_phases_per_run omitted/unlimited
- Created matching fixtures under `control-ts/tests/fixtures/policies/` for all 8 policies.
- Expanded `control-ts/tests/schemas/policy.schema.test.ts` from 3 to 11 tests, covering all fixtures plus defaults and rejection cases.

## Test Results

```
 RUN  v4.1.7 /Users/callumcowie/repos/Aether/control-ts

 Test Files  4 passed (4)
      Tests  25 passed (25)
   Start at  17:48:23
   Duration  208ms
```

All schema tests pass with no regressions across agent, phase, event, and policy suites.
