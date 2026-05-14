# Aether Hybrid Runtime Strategy Research

Date saved: 2026-05-12

## Purpose

This note preserves the combined research direction from:

- Local downloaded research: `/Users/callumcowie/Downloads/aether_codinglanguageresearch.md`
- Oracle research artifacts:
  - `.aether/oracle/synthesis.md`
  - `.aether/oracle/gaps.md`
  - `.aether/oracle/research-plan.md`
  - `.aether/oracle/state.json`

The goal is to make the decision referable before starting the next GSD milestone.

## Plain-English Summary

Aether should not become "all Go" and should not roll back to "all Bash."

Go should be the safe engine: state, locks, validation, finalizers, install,
update, publish, recovery, and verification.

TypeScript should become the control plane: worker orchestration, provider
adapters, Oracle loops, ceremony rendering, platform-specific lifecycle hosting,
and tests around prompt contracts.

Markdown, YAML, and TOML should remain the editable colony brain: command
playbooks, agent instructions, prompts, ceremonies, skills, and platform surfaces.

Bash should only be small glue: smoke tests, setup checks, and release helpers.

## Core Decision

Adopt a hybrid architecture:

```text
Go                  = safety kernel and runtime authority
TypeScript/Node     = orchestration host and agent control plane
Markdown/YAML/TOML  = editable colony brain
Bash                = small glue and smoke tests only
```

This is not a two-product strategy. The older Bash/Node versions should be used
as behavior baselines, not as a permanent second product. The best Classic
behaviors should be restored into the hybrid architecture with tests that prove
they have not regressed.

## What The Downloaded Research Concluded

The downloaded research recommended:

- Keep some Go, but demote it to boring infrastructure.
- Do not fully rewrite Aether.
- Do not fully roll back to Bash/Node.
- Do not keep Go as the owner of Aether's living behavior.
- Restore ceremony, swarm visibility, Oracle confidence loops, and wrapper-driven
  orchestration into editable assets plus a TypeScript-hosted control layer.
- Use the last strong Classic line, likely `v5.4.0`, as a behavior baseline to
  compare against the current runtime.
- Treat the Go migration as a boundary failure, not a language failure.

The key sentence from the research direction:

```text
Go should own safety, not soul.
```

## What The Oracle Research Concluded

The Oracle run reached useful synthesis but exposed an Oracle bookkeeping bug:
after all research questions were answered, iteration synthesis was validated as
if it referenced an unknown question ID. The research still produced useful
artifacts under `.aether/oracle/`.

Its central recommendation matched the downloaded research:

- Keep Go as the kernel for manifests, finalizers, state, verification, and
  publish/update.
- Restore visible orchestration in a host layer:
  - worker waves
  - spawn ceremonies
  - platform worker dispatch
  - confidence loops
  - finalizer handoff
- Do not restore Classic wrapper-owned state mutation.
- Do not restore raw shell state writes.
- Do not let visual output parsing become an authority.
- Start with a minimal plan/build slice before expanding to Oracle and swarm.

The Oracle also noted gaps:

- It did not produce a full command-by-command restoration matrix.
- It did not rank every Classic behavior.
- No TypeScript control-plane source exists yet in the current repo.
- Live platform smoke tests are still required.

## Synthesis

The two research streams agree on the next architecture.

Aether's current Go runtime contains valuable engineering, especially around
state safety and install/update mechanics. The problem is that the Go layer grew
into areas that should stay editable and platform-aware: ceremony, Oracle
behavior, orchestration loops, command flow, and worker dispatch UX.

The next milestone should therefore be a boundary-setting milestone, not a broad
feature milestone.

The objective is to make Aether hard to corrupt but easy to evolve.

## Target Boundaries

### Go Owns

- State mutation and validation
- Atomic writes and locking
- Manifests and finalizers
- Install, update, publish, and release integrity
- Recovery, doctor, medic, and patrol checks
- Structured JSON event emission
- Verification commands and runtime contracts
- Safe subprocess supervision where needed

### TypeScript Owns

- Lifecycle orchestration host
- Platform adapters for Claude Code, OpenCode, and Codex
- Worker wave orchestration
- Oracle/RALF confidence iteration loop
- Ceremony rendering from templates
- Provider and agent SDK integration
- Prompt contract tests
- Golden workflow orchestration tests

### Assets Own

- Command playbooks
- Agent instructions
- Worker role definitions
- Prompts
- Ceremony copy and stage names
- Skills
- Platform command surfaces

### Bash Owns

- Small smoke tests
- Setup checks
- Release glue
- Local developer helper scripts

## First Milestone Shape

The next GSD milestone should not try to rebuild all of Aether.

Recommended first slice:

1. Document current Go/TypeScript/assets/Bash boundaries.
2. Identify the Classic behavior baseline, likely `v5.4.0`, and verify it rather
   than assuming it.
3. Build a golden workflow test for one lifecycle path:
   `plan -> build 1 -> continue`.
4. Prototype a minimal TypeScript orchestration host that:
   - calls Go `--plan-only` commands for manifests
   - dispatches visible platform workers
   - records `spawn-log` and `spawn-complete`
   - calls Go finalizers
   - never writes `.aether/data` directly
5. Compare ceremony, worker activity, and state side effects against the Classic
   baseline.
6. Use the result to decide the next slice: Oracle loop, swarm visibility, or
   broader build/continue parity.

## Acceptance Criteria

The milestone is successful when:

- A written runtime boundary contract exists.
- A candidate Classic baseline commit/tag is identified and smoke-tested.
- One golden lifecycle workflow captures expected user-visible behavior and state
  side effects.
- A minimal TypeScript host can drive `plan -> build 1 -> continue` through Go
  manifests/finalizers without direct state writes.
- The host produces visible worker activity and ceremony rather than silent
  synthetic progress.
- Go remains the only authority for state mutation.
- A clear follow-up plan exists for Oracle confidence loops and swarm visibility.

## GSD Milestone Prompt

```text
GSD milestone: Aether Hybrid Runtime Boundary and Orchestration Recovery

Context:
Aether has gone through a broad Bash/Node to Go migration. The Go runtime now
has valuable safety machinery, but the migration also caused regressions in the
living parts of Aether: Queen orchestration, visible worker waves, ceremony,
Oracle/RALF confidence iteration, swarm visibility, and platform-specific agent
dispatch behavior. Recent research in
.aether/docs/hybrid-runtime-strategy-research.md, the downloaded runtime
strategy note, and .aether/oracle/synthesis.md all converge on the same direction:
keep Go as the safety kernel, introduce a TypeScript orchestration control plane,
keep Markdown/YAML/TOML as the editable colony brain, and keep Bash only as small
glue.

Goal:
Create the first concrete hybrid-recovery slice. Do not rewrite Aether. Do not
roll back to Classic as the product. Use the strongest Classic version, likely
v5.4.0 but verify this, as a behavior baseline. Prove one lifecycle workflow can
be restored through the right architecture:

  plan -> build 1 -> continue

Required outcomes:
1. Write a runtime boundary contract that clearly says what Go owns, what
   TypeScript owns, what editable assets own, and what Bash may still do.
2. Identify and smoke-test the best Classic baseline commit/tag for behavior
   comparison.
3. Add golden tests or snapshot tests for the selected lifecycle workflow,
   covering both visible ceremony/worker activity and state side effects.
4. Prototype the smallest viable TypeScript orchestration host that:
   - calls Go plan-only/build-plan commands to get manifests
   - dispatches visible platform workers
   - records spawn-log and spawn-complete
   - calls Go finalizers
   - never writes .aether/data directly
5. Ensure Go remains the only authority for state mutation, finalizers, locking,
   install/update/publish, and verification contracts.
6. Produce a follow-up migration map for restoring Oracle confidence iteration,
   swarm visibility, and broader build/continue parity after this slice.

Non-goals:
- Do not rewrite the whole runtime in TypeScript.
- Do not restore raw Bash state mutation.
- Do not maintain Classic and Go as two long-term products.
- Do not move install/update/publish safety out of Go.
- Do not make visual output parsing authoritative.

Definition of done:
- The boundary contract is committed.
- The Classic baseline is documented with evidence.
- The golden lifecycle workflow test exists.
- The TypeScript host spike exists and can run the selected workflow or, if a
  complete run is blocked, documents the exact blocker with a reproducible test.
- The work leaves Aether safer and clearer even if the TypeScript host remains
  experimental after this milestone.
```

