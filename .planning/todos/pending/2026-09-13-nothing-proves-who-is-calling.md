---
created: 2026-09-13T22:20:00Z
title: Nothing proves who is calling — two authorization decisions taken on an unauthenticated self-declared string
area: authorization
source: Phase 203 closing code-review gate (203-REVIEW.md CR-02 and CR-04); owner decision 2026-09-13 to design this rather than patch it
severity: critical
windows_entries: [18, 30]
---

## Problem

Two separate authorization decisions in this runtime are taken on a string the
caller types in, with nothing anywhere proving the caller is who it says it is.
They are the same defect wearing two hats, which is why the owner declined to
patch them individually and asked for one piece of design work instead.

### 1. The delegation depth cap is bypassable by self-assertion

`spawnParentIsRoot` (`cmd/spawn.go:22`) matches an unauthenticated,
caller-supplied `--parent` string against the fixed sentinel list
`spawnRootParentNames = {Queen, Prime-1, Swarm}` and, on a match, grants depth 0
with `DepthIsAuthoritative=true` and no spawn-tree entry required.

Both `aether spawn-can-spawn --name Queen` and `aether recruit --parent Queen`
therefore pass the depth check regardless of the caller's real depth, defeating
`spawnMaxDelegationDepth` — the runaway-spawn and cost control.

Everything *else* on that path is correctly fail-closed, which is what makes the
hole so sharp: depth is read from the recorded spawn tree via
`latestSpawnEntryByName` and never from a flag, and `validateRecruitmentIntent`
(`cmd/recruitment_intent.go:153`) explicitly refuses a non-sentinel parent whose
depth is not authoritative, with the reason "a parent's depth is never trusted
from a self-declared claim". The sentinel exemption is the single exception to a
rule the rest of the code states out loud.

Inherited from Phase 173, not introduced by Phase 203; 203-03 mirrored the
existing behaviour deliberately and documented it. Phase 203 made it reachable
through one more command.

### 2. Owner-only steering-note actions are gated by a flag that defaults to "owner"

Revoke, appeal, pin and unpin are authorized purely by `--actor`
(`cmd/pheromone_mgmt.go:417`), which **defaults to `"owner"`** when omitted, with
no caller authentication whatsoever. Any process that can run the `aether` binary
— including an ordinary dispatched worker — can revoke a permanent REDIRECT
constraint, or arm and disarm outcome-weighted tuning, by simply leaving the flag
off.

CLAUDE.md states "A note the owner pinned in place is never moved by this
automatic tuning." That is true of the tuning pass. It is not true of the pin
itself, which anyone can set or clear.

## Why this is design work, not a fix

Both holes close the same way: there has to be something that distinguishes a
caller who genuinely *is* the coordinator (or the owner) from one who merely says
so. That mechanism does not exist anywhere in this runtime today. Inventing a
narrow one at each call site would produce two different answers to the same
question — precisely the "two surfaces disagreeing" failure this project keeps
rediscovering.

Open questions the design has to answer:

- What *is* the trustworthy signal? A recorded spawn-tree entry the caller cannot
  forge? A per-run secret issued at dispatch and carried in the worker's
  environment? Process ancestry? Something the owner alone holds for owner-only
  actions?
- Is the owner's identity the same kind of fact as a coordinator's, or a
  different one? An interactive terminal can be evidence of an owner in a way it
  never is for a worker.
- What is the fail-closed behaviour when the signal is unavailable — refuse, or
  degrade to the lowest privilege? Refusing must not strand an ordinary run.
- Does this need to survive `aether run` (autopilot), where there is no human at
  the keyboard at all?

## Acceptance

Whatever is built, the proof is the same in both places and must be a command
that fails when the requirement is unmet:

- A caller claiming `--parent Queen` without genuinely being the coordinator is
  refused **by name**, and the depth cap holds.
- A caller performing a revoke, appeal, pin or unpin without genuine owner
  authority is refused **by name**, and omitting `--actor` does not grant it.
- Both refusals proven able to fail: remove the check, watch the test go red
  naming the right thing, restore.
- Ordinary runs — including autopilot — are measurably unaffected when nobody is
  claiming anything they should not.
