# Park the Codex native worker lifecycle programme

**Decision date:** 2026-09-20 (owner)

**Active authority:** `codexNativeBuildOptInEnv` / `codexNativeBuildOptedIn()` in
`cmd/codex_native_context.go`, gating `cmd/command_guide.go`,
`cmd/codex_build.go`, and `cmd/source_check.go`.

## What changed

Phases 204.2 through 204.5 (Codex Native Worker Lifecycle, Codex Workflow
Parity, Governed Recruitment and Recovery, Codex Parity Proof and Acceptance
Handoff) are **PARKED, not cancelled**. The reason is cost and time, not a
defect: replacing "the Go program starts Codex helpers itself" with "the
Codex chat starts its own built-in helpers and the Go program audits them"
consumed a week and the weekly usage quota chasing paid live proof of every
edge case, and is not needed for daily use of Aether.

Nothing already built is deleted or reworded. All Phase 204.2 code, its guide
text (`codexNativeBuildCommandGuide`), its skill prose, and its manifest
protocol pin stay exactly where they are in the tree. They are simply made
reachable only through an explicit opt-in environment variable,
`AETHER_CODEX_NATIVE_BUILD=1`. Without that variable set to exactly `"1"`,
Codex builds now default to the already-proven route: the Go runtime
dispatches Codex workers directly (`aether build <phase>`, no `--plan-only`,
no wrapper-driven manifest hop).

## What is kept

- All code, at commit `96ff6f45` (HEAD at the time of this decision).
- The F19/L23 proofs and inventory captured at
  `~/.aether-backups/phase204-2-collector/claude-handoff-current-20260920.json`.
- ANT-04 through ANT-16 stay unchecked and honestly incomplete — this decision
  does not regrade or retroactively satisfy them.
- Q25 stays failed, not regraded.
- The four backup `.DS_Store` files under the phase204-2-collector backup
  directory still carry temporary immutable flags and must be restored per
  the handoff before those backups are ever deleted.

## Supersession

This supersedes the 2026-09-16 scope amendment that sequenced Codex parity
ahead of the remaining Phase 205 walkthrough. That amendment is no longer
active guidance; the next work is publishing 1.0.82 with the direct-route
default, followed by a slim Claude-side trial (one small real task through
`/ant-build` → `/ant-continue`, timed against the owner's own 10-minute bar),
not further 204.2–204.5 work.

## Resuming this programme later

Resuming Phase 204.2 requires exporting `AETHER_CODEX_NATIVE_BUILD=1` in the
live evidence harness environment (`cmd/codex_native_worker_live_test.go`,
left untouched by this decision so its evidence-chain digest stays stable).

For dummies: Codex is Aether's other supported chat tool, alongside Claude
Code. There were two ways being built for it to run a build: a proven simple
one (the Go program does the work directly) and a fancier experimental one
(the Codex chat manages its own helpers while the Go program checks on them).
The fancier one turned out to cost a lot of time and quota chasing proof for
edge cases nobody hits day to day, so it is being set aside — not thrown
away, just switched off by default. Anyone who wants to turn it back on can,
with one environment variable.
