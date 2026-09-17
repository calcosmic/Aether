# Task dependency references

A task's `depends_on` entries may use either another task's `id` or its
`semantic_id`. Both names identify the same task. References are checked across
the entire plan; missing targets, duplicate/ambiguous names and cycles are
refused. A task in another phase is a satisfied build prerequisite only when
that earlier task is completed.

Planning validation, accepted-plan validation and build scheduling share the
same reference resolver. Build uses a private scheduling copy: the saved plan,
accepted revision, candidate content hash and approval receipt are not rewritten
to replace semantic names with numeric IDs.

## Recovering an existing accepted plan

The original 1.0.79 build planner can report an existing semantic target as
missing. Use a runtime containing the shared dependency resolver before retrying:

```sh
AETHER_OUTPUT_MODE=json aether plan --repair-artifact
AETHER_OUTPUT_MODE=json aether build 1 --plan-only
```

For an accepted plan, the first command reports `repair_scope: accepted_revision`,
`status: accepted_plan_validated`, `validated: true`, `repaired: false` and
`state_effect: unchanged`. It validates dependency references and preserves the
existing approval. It does not rewrite the old staging artifact or launch
workers. The second command prepares build assignments under that approval.
Use the phase-specific next command returned by the runtime when the current
phase is not 1.

Do not use `state-mutate` to rewrite only `plan.phases`: that would make the
active projection disagree with the accepted revision. A genuinely invalid
approved dependency needs a corrected candidate through the existing review and
acceptance process, not a manufactured or rewritten approval.

## Legacy staging artifacts

When there is no accepted revision, `--repair-artifact` retains the older repair
of `.aether/data/planning/phase-plan.json`. It reports
`repair_scope: legacy_phase_plan_artifact` and whether that artifact was actually
changed. It can normalize supported positional aliases such as `P1-T1` to
`1.1`; it does not accept a new plan or change approval bindings.

Repair runs before preset selection and does not rerun planning workers.
Combining it with generation, preset, revision, research or brief-printing
options is an error, so another mode cannot silently take precedence.
