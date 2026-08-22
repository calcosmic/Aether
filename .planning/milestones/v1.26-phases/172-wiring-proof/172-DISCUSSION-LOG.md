# Phase 172: Wiring Proof - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-08
**Phase:** 172-Wiring Proof
**Areas discussed:** What counts as called, What if there are more than 8 orphans, Where the allowlist lives, What `--enforce` actually does

---

## Pre-discussion: todo cross-reference

| Option | Description | Selected |
|--------|-------------|----------|
| Neither | Both matched on generic keywords; runtime behaviour bugs, not wiring | ✓ |
| Fold the ts-host one | ts-host preflight hardcoded timeout | |
| Fold the continue-finalize one | `--reconcile-task` evidence gate | |
| Fold both | | |

**User's choice:** Neither
**Notes:** Both remain in the backlog. The ts-host one is already carried as inserted Phase 163.2 in the v1.25 roadmap.

---

## What counts as called

### Q1 — What should count as proof that a command is actually being used?

| Option | Description | Selected |
|--------|-------------|----------|
| Only things that run it | Go code outside the command's own file, a loaded platform wrapper, or a shipped hook. Documentation excluded. The only rule under which today's 8 known orphans are caught | ✓ |
| Any mention anywhere counts | Simplest, near-zero false alarms — but passes all 8 known orphans on day one | |
| Mentioned-only = warning, not failure | Two-tier; complains but doesn't block | |

**User's choice:** Only things that run it
**Notes:** The deciding evidence was that the 8 `skill-*` orphans appear in ~20 files today and reach nobody. The two-tier option was rejected on this repo's track record of warnings nobody acted on.

### Q2 — How do we avoid flagging the hundreds of human-invoked commands?

| Option | Description | Selected |
|--------|-------------|----------|
| A menu entry counts | A `/ant-…` wrapper or `.aether/commands/*.yaml` spec proves user reachability | ✓ |
| Trust the existing labels | Exempt by `classification` tier in the 400-command catalog | |
| No exceptions at all | Every command needs a programmatic caller | |

**User's choice:** A menu entry counts
**Notes:** Classification tiers were rejected because `scripts/classify_commands.py` assigns them by name heuristic — one wrong guess would grant a silent permanent exemption. "No exceptions" was rejected because a several-hundred-entry allowlist is a list nobody reads.

### Q3 — What should the checks do with the retired playbook documents?

| Option | Description | Selected |
|--------|-------------|----------|
| Check them, don't trust them | In scope for the flag audit, never counted as caller evidence | ✓ |
| Ignore them completely | Out of both checks | |
| Treat them as fully live | Count for both purposes | |

**User's choice:** Check them, don't trust them
**Notes:** Directly informed by the `suggest-analyze` precedent recorded in CLAUDE.md — its only call site sat in a playbook the runtime never loaded, behind `2>/dev/null`, and it had never executed once.

---

## What if there are more than 8 orphans

### Q1 — If the check finds far more dead commands than the 8 expected?

| Option | Description | Selected |
|--------|-------------|----------|
| List them all, tagged | Honest baseline, each entry tagged with reason and owning phase | ✓ |
| One flat honest list | Honest but untagged | |
| Only allow 8 — fix the rest now | Truest to intent, but turns the phase into open-ended repair work | |

**User's choice:** List them all, tagged
**Notes:** Tagging is required because Phase 178's success criterion counts the 8 skill entries reaching zero — untagged entries in a larger list make that unmeasurable.

### Q2 — Does having a test count as being used?

| Option | Description | Selected |
|--------|-------------|----------|
| No — tests don't count | A passing test proves it works, not that anything uses it | ✓ |
| Yes — a test counts | Less noise up front | |

**User's choice:** No — tests don't count
**Notes:** "It works and nothing calls it" is the exact failure this phase catches; a test vouching for a command would blunt the alarm.

### Q3 — Do we allow an escape hatch for adding to the shrink-only list?

| Option | Description | Selected |
|--------|-------------|----------|
| No escape hatch | Hard-blocked. Wire it or delete it | ✓ |
| Allow it with a written reason | Reason + removal date per entry | |
| Allow it if a human approves | Reviewer sign-off | |

**User's choice:** No escape hatch
**Notes:** Rejected the reviewer-approval option on the grounds that the user is the only reviewer, so it would mean "allowed whenever inconvenient."

---

## Where the allowlist lives

### Q1 — Where should the list live?

| Option | Description | Selected |
|--------|-------------|----------|
| Its own dedicated file | One file, one job; visible in review | ✓ |
| A new column in the existing 400-command file | Reuses existing gate machinery | |

**User's choice:** Its own dedicated file
**Notes:** Decided by a fact confirmed during discussion — `scripts/classify_commands.py` rewrites `cmd/testdata/command_catalog.json` in place, so entries stored there could be silently erased by a routine regeneration.

### Q2 — How is shrink-only enforced?

| Option | Description | Selected |
|--------|-------------|----------|
| Compare against a saved copy | Committed baseline; any new name fails | ✓ |
| Just count the entries | Fail if the count rises | |
| Compare against the previous saved version | Use git history | |

**User's choice:** Compare against a saved copy
**Notes:** Counting was rejected because a one-out-one-in swap passes silently. Git history was rejected because CI shallow clones and rewritable history make it fail or pass for unrelated reasons.

### Q3 — Does the flag audit's existing skip-list get the same rule?

| Option | Description | Selected |
|--------|-------------|----------|
| Same rule for both | Both shrink-only, enforced identically | ✓ |
| Leave it alone | Only 2 entries today | |

**User's choice:** Same rule for both
**Notes:** Framed as closing the side door — otherwise exemption pressure relocates to the unguarded list. The user noted "later" is how the first list reached 8.

---

## What `--enforce` actually does

The first attempt at this question was rejected by the user as unintelligible jargon, and was re-asked using a plain-language analogy (a worker phoning the office for permission before hiring an assistant). This prompted a mid-session update to both CLAUDE.md files — see commit `8963bcc4`.

### Q1 — Should `--enforce` be made real in this phase, or deferred?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, make it real now | Deny ⇒ non-zero exit. No visible change today; Phase 173's enforcement works free | ✓ |
| Fix the phone line only | Accept the flag, ignore it | |
| Cross those words out for now | Remove `--enforce` from `workers.md`, reintroduce in Phase 173 | |

**User's choice:** Yes, make it real now
**Notes:** The inert-flag option was rejected as the precise anti-pattern the milestone is named after. Removing it from the manual was rejected as fixing a broken promise by deleting the promise.

### Q2 — Fix the command or the manual, for the positional depth argument?

| Option | Description | Selected |
|--------|-------------|----------|
| Make the command accept it | Accept the positional depth as `workers.md` writes it | ✓ |
| Change the instructions to match the command | Rewrite `workers.md` to use the existing `--depth` flag | |

**User's choice:** Make the command accept it
**Notes:** Instruction docs in this repo have a demonstrated history of drifting from the runtime; the criterion is pinned to the string the manual actually contains.

### Q3 — How should a wiring failure surface in CI?

| Option | Description | Selected |
|--------|-------------|----------|
| Its own named step | A named CI step running just these checks, plus normal test coverage | ✓ |
| Just part of the normal test run | Zero setup; already covered by `go test ./...` | |

**User's choice:** Its own named step
**Notes:** A failure must read as "wiring problem", not one anonymous failure among hundreds.

---

## Claude's Discretion

- Allowlist file name, format and location, and the shape of the reason tag
- Test names beyond the roadmap-pinned `TestNoRegisteredSubcommandIsUnreferenced`
- Caller-scan implementation (AST walk vs. text scan) and how a command's own definition file is identified
- Whether WIRE-03 extends `TestCLIFlagAudit` or ships as a new test, provided `.aether/*.md` enters scope and failures name file, line and flag

## Deferred Ideas

- Making `spawn-can-spawn` return a real deny decision — Phase 173 (SPAWN-01)
- Reclaiming or removing the 8 `skill-*` lifecycle commands — Phase 178 (SKILL-01)
- The wider unreachable-command sweep — already parked as RECLAIM in Future Requirements
- Retiring the playbook directory entirely — surfaced while ruling on playbook handling; not this phase
- Reviewed but not folded: `2026-08-01-finalize-reconcile-task-evidence-gate.md`, `2026-08-01-ts-host-preflight-hardcoded-timeout.md`
