# Classic Planning Loop Restoration

Updated: 2026-05-17

This note records the v5.4 planning behavior that has been restored in the
hybrid TypeScript-host plus Go-engine architecture.

For dummies: old Aether kept planning until it was confident enough, ran out of
tries, stopped improving, or the user accepted the best plan. The modern system
now records those same stop reasons, but Go still writes the official plan.

## v5.4 Baseline

- Presets mapped planning depth to confidence and iteration budgets:
  `fast` 80/4, `balanced` 90/6, `deep` 95/8, and `exhaustive` 99/12.
- `--target <N>` overrode the confidence target.
- `--max-iterations <N>` overrode the loop budget.
- `--accept` allowed finalizing the current best plan below target.
- The loop stopped on target confidence, max iterations, or two consecutive
  low-improvement iterations.
- Phase research artifacts were written after planning so builders had durable
  context.

## Restored Surface

- `aether plan`, `aether plan --plan-only`, and `aether host plan` now accept
  `--target`, `--max-iterations`, and `--accept`.
- Plan manifests include `planning_loop` so wrappers know the target, budget,
  stall threshold, and pending finalization status.
- `aether plan-finalize` recomputes the final stop reason from accepted worker
  evidence and writes that result into canonical planning artifacts.
- `.aether/data/planning/phase-plan.json` includes `planning_loop` after
  finalization.
- Visual plan output surfaces the planning loop target, iteration count, and
  stop reason.

## Safety Boundary

- Wrappers and the TypeScript host forward loop controls; they do not write
  `.aether/data` planning state directly.
- Plan-only manifests are still intent, not evidence. They can say what should
  be run, but only `plan-finalize` records accepted Scout/Route-Setter output.
- Stale manifests, stale route-setter artifacts, root mismatches, and invalid
  worker results still block before state mutation.
