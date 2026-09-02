# Phase 199: Front Door and Classic Contract - Context

**Gathered:** 2026-09-02
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore one understandable Claude Code and OpenCode `/ant-*` front door around the authoritative Go runtime, and turn the February-April Classic lifecycle into an executable modern contract. This phase owns the visible journey through help/setup, initialization, territory freshness, status/phase/history, guided or autonomous entry, pause/resume/recovery, honest seal, post-seal entomb, safe maintenance separation, and the corpus fixtures that prove the restored behavior and experience.

This phase defines cross-phase interaction contracts for Autopilot, live steering, Swarm, and typed activity, but does not absorb the substantive planning loop (Phase 200), work-cycle implementation (Phase 201), live Swarm/Oracle system (Phase 202), causal pheromone flow (Phase 203), Learning Governor (Phase 204), or final owner acceptance (Phase 205). Codex-native `$ant-*` lifecycle skills remain a later milestone.

</domain>

<decisions>
## Implementation Decisions

### Entry Commands

- **D-01:** Claude Code and OpenCode expose the colony through `/ant-*` commands. Raw `aether ...` commands are runtime plumbing, not the ordinary user vocabulary.
- **D-02:** `/ant-init` begins the guided colony journey. After initialization and an accepted plan, `/ant-build 1` and `/ant-run` are equal choices; neither is preselected or labelled recommended. Guided operation is an option, not a mandatory default.
- **D-03:** `/ant-run` is the optional Autopilot and may begin only after initialization and plan acceptance. If invoked too early, it performs no work, names the missing prerequisite, and routes to the exact `/ant-init` or `/ant-plan` command.
- **D-04:** A valid `/ant-run` first shows a short operating contract containing the goal, intended phase range, active pheromones, and pause conditions, then starts automatically. Invoking the command is already consent to run.

### Autopilot Boundaries

- **D-05:** `/ant-run` aims to finish every remaining accepted phase. It performs safely bounded repairs automatically, accumulates non-blocking warnings and debt for the final report, and pauses only when continuing would be unsafe or dishonest: safety failure, corrupt state, missing authority, a material owner decision, or a failed dependency that invalidates later work.
- **D-06:** Autopilot may revise tasks, dependencies, sequencing, and implementation details when evidence invalidates the accepted plan. It must pause before changing the goal, promised behavior, scope, risk authority, or owner acceptance criteria.
- **D-07:** FOCUS and FEEDBACK should steer active ants without stopping unrelated work where the host supports live delivery. The receiving ant acknowledges the signal, and runtime evidence records the decision it changed. REDIRECT stops only work that conflicts with it. Where live delivery is impossible, the product says so and applies the signal at the next safe boundary rather than claiming false influence. Phase 199 defines this contract; Phase 203 implements the causal pheromone mechanism.
- **D-08:** Invoking `/ant-swarm` during Autopilot is a localized intervention: checkpoint and pause only the affected job, let independent work continue, integrate the verified Swarm result, and resume the affected path. Phase 199 defines the front-door behavior; Phase 202 supplies the substantive Swarm and typed-event implementation.

### Orientation and Colony Voice

- **D-09:** `/ant-status` is the complete authoritative snapshot by default. It shows colony identity and goal, phase/task progress, active ants and lineage, pheromones, research and Dreams, memory, findings, verification, elapsed time and reported cost, recent history, blockers, and clear next choices without exposing raw protocol records. `/ant-status --compact` is the optional short view.
- **D-10:** Major commands end with focused rich closeouts: colony identity, participating ants, what happened, evidence and results, state and pheromone changes, unresolved issues, and next choices. They do not repeat the entire status dashboard.
- **D-11:** Queen-led voice and ceremony are strong at lifecycle moments—initialization, planning, build waves, Swarm, verification, recovery, seal, and entomb. Quick inspections remain shorter but retain colony identity and plain language. Presentation may never invent workers, activity, cost, evidence, or outcomes absent from runtime truth.
- **D-12:** `/ant-status` and `/ant-watch` remain complementary. Status is the authoritative snapshot; Watch is the live colony cockpit and falls back to the latest status plus activity history when idle. Phase 202 owns the real typed-event live implementation.

### Pause, Resume, Seal, and Entomb

- **D-13:** Expose exactly one intentional pause command and one return command: `/ant-pause` and `/ant-resume`. The useful historical `pause-colony` and `resume-colony` behavior moves into those canonical commands. The suffixed names disappear from help, documentation, and installed command files; any short-lived parser redirect is invisible migration plumbing, not product vocabulary.
- **D-14:** `/ant-pause` stops at a safe boundary and writes one structured, validated handoff that preserves partial work, attempt identity, scoped decisions, context, and worker lineage exactly once.
- **D-15:** `/ant-resume` is the only recovery front door. It validates and restores a clean handoff when present; after an unclean interruption, it inspects durable state, worker activity, repository changes, and other runtime evidence to reconstruct the safest honest recovery point. Confirmed and reconstructed facts remain visibly distinct. Do not add a separate public `/ant-recover` command.
- **D-16:** `/ant-seal` is normal honest completion. `/ant-seal --force --reason "..."` is an explicit owner-only escape hatch for an incomplete colony; it durably records unfinished phases, failed or skipped gates, evidence, rollback information, and the reason, and is never rendered as successful completion. No ant, wrapper, or Autopilot path may add or invoke `--force` autonomously. `/ant-abandon` is not part of the ordinary public workflow.
- **D-17:** Seal and entomb remain distinct because they guard different transitions. `/ant-seal` crowns the colony, writes the durable Crowned Anthill record, and retains active state for review; it does not perform the entomb/archive-and-clear transition or reset active state. `/ant-entomb` is an optional post-seal operation that copies the colony into chambers, verifies archive integrity, retains memory and tombstone evidence, and only then clears active state for a new colony. It refuses unsealed or inconsistent state. Help describes it plainly as “archive and clear the sealed colony.” A forced seal remains visibly forced after entombment.

### the agent's Discretion

No owner-visible behavior was delegated. Research and planning may choose internal schemas, rendering composition, compatibility-redirect duration, and code organization provided they preserve the locked command vocabulary, platform order, runtime authority, truthfulness, and lifecycle boundaries above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone Authority

- `.planning/PROJECT.md` — Defines the active v1.28 restoration goal, Claude/OpenCode-first platform order, modern Go authority, Phase 199-205 sequence, and routing of the three pending todos.
- `.planning/ROADMAP.md` — Defines Phase 199 scope, requirements, success criteria, and downstream phase boundaries.
- `.planning/REQUIREMENTS.md` — Defines `SYNTH-01`, `CEC-01`, `CEC-02`, `CEC-04`, `CEC-08`, `LIFE-01..06`, and `PROOF-01`; also preserves the later-Codex boundary in `PROOF-04`.
- `.planning/research/priority-spec-v3-backlog.md` — Ratified governing backlog and priority context carried into v1.28.

### Classic Evidence and Synthesis

- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — Authoritative 30,000-plus-word restoration brief and causal comparison of the February-April Classic experience with the current runtime.
- `.planning/research/v1.28-classic-capability-ledger.md` — Complete `CAP-001..CAP-072` traceability. Its pre-discussion dispositions are inputs, not permission to override the locked decisions here; in particular, the Phase 199 synthesis must preserve the newly confirmed two-stage seal/entomb distinction.
- `.planning/research/v1.28-classic-synthesis-template.md` — Required keep-current/restore-modern/replace-better/retire-with-proof mechanism-study structure before implementation planning.
- Historical source anchors: Git ref `3a5b81c2` for February visual/colony identity, `v5.0.0` for Worker Emergence, and `v5.4`/`v5.4.0` for the mature April boundary. At `3a5b81c2` and `v5.0.0`, inspect `.claude/commands/ant/`; at `v5.4`/`v5.4.0`, also inspect `.aether/commands/` and `.aether/docs/command-playbooks/`. Use the actual historical source rather than remembered names or screenshots.

### Current Architecture and Contracts

- `.planning/codebase/ARCHITECTURE.md` — Maps the presentation-wrapper/runtime/state boundary and current lifecycle request paths.
- `.planning/codebase/CONVENTIONS.md` — Defines Go command organization, co-located tests, cancellation/concurrency patterns, and runtime conventions.
- `.planning/codebase/STACK.md` — Records the current Go/Cobra/runtime stack and supported platform surfaces.
- `.aether/docs/wrapper-host-contract.md` — Defines how thin Claude/OpenCode wrappers call authoritative Go transactions and return host-spawned results.
- `RUNTIME UPDATE ARCHITECTURE.md` — Defines canonical publish/install/update distribution and generated artifact flow across platforms.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- `.aether/commands/*.yaml`: canonical wrapper sources already exist for init, run, status, watch, pause, resume, seal, and entomb; they are the synchronization point for Claude and OpenCode presentation.
- `cmd/session_flow_cmds.go`: already implements pause handoff creation, staleness checks, and resume mutation. Its current command grammar is partially inverted (`resume-colony` is canonical and `resume` is an alias), making it a direct migration seam for D-13 through D-15.
- `cmd/context.go`: `resume-dashboard` already supplies recovery information that can feed the single smart `/ant-resume` experience, but its compatibility path may backfill the legacy top-level `session.json` mirror; planning must not treat the present command as strictly read-only.
- `cmd/status.go` and `cmd/compatibility_cmds.go`: existing status, Watch, and Autopilot projections provide the base for the restored dashboard and entry contracts.
- `cmd/codex_workflow_cmds.go`, `cmd/seal_final_review.go`, and `cmd/seal_confirmation.go`: existing seal, final-review, owner-confirmation, and force-reason paths provide much of the modern closure safety kernel.
- `cmd/entomb_cmd.go`: already describes the desired distinct transition—archive a sealed colony into chambers and reset active state—and has extensive co-located archive/reset tests in `cmd/entomb_cmd_test.go`. Its current pre-reset check proves only that four required archive paths exist, not that their contents are intact; the stronger D-17 integrity contract remains implementation work.
- `cmd/force_seal_test.go`, `cmd/seal_ceremony_test.go`, and `cmd/seal_confirmation_test.go`: existing tests pin the owner-only force escape hatch, reason requirement, no-autonomous-force rule, confirmation, and ceremony behavior.

### Established Patterns

- Go runtime commands own authoritative state mutation, verification, gating, and lifecycle truth. Wrappers may render, orchestrate host workers, and write explicitly non-authoritative advisory artifacts such as hygiene reports, but they may not directly mutate authoritative runtime state.
- `.aether/commands/*.yaml` generates or governs `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md`; source and both installed surfaces must change together.
- The storage layer provides atomic file replacement, but several current pause, resume, handoff-restoration, and entomb paths remain allow-listed non-atomic read/modify/write sequences. Phase 199 must close or safely govern those lifecycle races. The historical and locked archive contract also requires validating the durable copy before clearing the active source; the current entomb path's existence-only check must be strengthened to meet that contract.
- User-facing visual output and machine-readable JSON are separate renderings of the same transaction; ceremony must remain evidence-backed.
- Tests are co-located with Go command implementations and should prove outcome, behavior, and experience—not snapshot wording alone.

### Integration Points

- The current generated surfaces expose both `pause.md`/`pause-colony.md` and `resume.md`/`resume-colony.md` on Claude and OpenCode; Phase 199 must collapse these visible duplicates while retaining only bounded parser compatibility if justified.
- `cmd/session_flow_cmds.go`, the root Cobra registration/help tree, `.aether/commands/pause.yaml`, and `.aether/commands/resume*.yaml` must agree on `/ant-pause` and `/ant-resume` as the only documented names.
- The current `/ant-run` engagement card shows only goal/current phase/maximum; Phase 199 must add the locked phase range, active pheromones, and pause conditions without adding another confirmation.
- The current force-seal kernel persists the override and reason, but shared seal rendering and the Markdown summary can still report all planned phases as completed and use an ordinary success ceremony. Phase 199 must give forced closure a visibly incomplete/forced result everywhere it appears.
- Status, phase, history, resume, run, seal, and entomb need one shared lifecycle projection so Next Up and blocker truth cannot disagree across commands.
- Historical corpus fixtures must exercise public Claude/OpenCode paths while asserting the Go transaction beneath them; later Codex `$ant-*` skills should be able to reuse the same contract without redefining it.

</code_context>

<specifics>
## Specific Ideas

- The target is the composite February-April experience from before the command/wrapper collapse, not a nostalgic copy of one tag and not a return to shell or prompt authority.
- The colony should feel Queen-led and alive at consequential moments, with deterministic caste identity, visible ants and lineage, clear stage changes, and meaningful closeouts—all grounded in actual runtime events.
- Keep the vocabulary explicitly Ant-shaped for users: Claude/OpenCode `/ant-init`, `/ant-run`, `/ant-swarm`, and related `/ant-*` commands. Do not train users to address Codex or other hosts using “Aether commands.”
- Preserve the historically meaningful distinction: sealing closes and crowns the book; entombing files it away and clears the desk.

</specifics>

<deferred>
## Deferred Ideas

- Implement Codex-native `$ant-init`, `$ant-run`, `$ant-swarm`, and the complete matching `$ant-*` skill family only after the Claude/OpenCode contract is stable. This is a later milestone, not Phase 199.
- Substantive live Swarm/Watch event machinery belongs to Phase 202; causal pheromone delivery and measured influence belong to Phase 203. Phase 199 records only their public front-door contracts.

### Reviewed Todos (not folded)

- `2026-08-20-spec-builder-feature.md` — The readable specification capability belongs to Phase 200's owner-readable specification and accepted-plan work.
- `2026-08-27-worker-turnaround-is-too-slow.md` — Worker turnaround and one-job cost belong to Phase 201's Queen-led work cycle and attempt model.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — Platform preflight and timeout handling remain routed to Phase 203 by the approved milestone contract. The keyword-only Phase 199 todo match does not change ownership.

</deferred>

---

*Phase: 199-front-door-and-classic-contract*
*Context gathered: 2026-09-02*
