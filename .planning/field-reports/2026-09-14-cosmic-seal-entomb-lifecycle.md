# Resume → seal → entomb on CosmicDashboard — Field Report

**Date:** 2026-09-14 · **Aether:** v1.0.78 (binary and hub)
**Repo:** `CosmicDashboard`, a downstream consumer. Reported by the peer session that ran
the whole lifecycle.

**Outcome:** ⚠️ Done, but the hard way. The colony was sealed (`verified_completion`) and
entombed (receipt `entomb-c14772c5eda7340a308b-receipt`). Getting there took host-side
workarounds for two runtime defects that block every wrapper-run lifecycle, plus several
smaller wrapper/runtime mismatches.

The owner is non-technical and never saw a single Aether visual card (defect 3). He
called the experience "a disaster". Nothing that went wrong came from the colony's own
work; all of it was lifecycle plumbing.

**Status:**
- Defects 1, 2 and 4 were checked against the source on the `oracle-reinstate` branch
  (file:line references below).
- Nothing in this repo was changed except adding this report.
- Downstream records of the run: the CosmicDashboard chamber
  `.aether/chambers/chamber-project-make-cosmic-my-central-business-hub-comm-c14772c5eda7340a308b`
  and CosmicDashboard commits `68fd4ba` (seal record) and `fc45e05` (archive).

---

## Timeline (UTC)

| When | What |
|---|---|
| 09-13 ~21:55 | `aether resume`: the handoff was validated with **Confirmed** provenance, and state effect was committed. This worked well. |
| 09-13 21:56 | `aether host seal`: a plan-only manifest with 2 reviewers (Auditor Inspect-51, Probe-79). |
| 09-13 ~22:15 | Round 1. The Probe passed. The Auditor blocked on evidence only: the real-day proof predated the final commit, and no gate record covered `test:operator`. |
| 09-14 08:01 | The owner's real-day re-ask surfaced a downstream product bug: Gmail missed its 5 s agenda deadline. The round-1 Auditor had flagged exactly this as a LOW finding. It was fixed downstream (cd22a13) and re-proven at 08:19. |
| 09-14 08:22 | Fresh `aether host seal` manifest; round 2 ran, and both reviewers passed. |
| 09-14 ~08:38 | `seal-finalize` refused the packet: **"completed without a handoff"** (defect 1). |
| 09-14 08:46 | Both reviewers resent their results with handoffs. `seal-finalize` asked for owner confirmation, the owner said yes, and the colony was **sealed 08:47:55**. |
| 09-14 ~13:05 | `aether entomb`: the preflight failed on **missing `.aether/HANDOFF.md`** (defect 2). |
| 09-14 13:12 | Workaround: `aether pause` wrote HANDOFF.md, then entomb preview, owner confirm, `entomb --confirm`. The colony was **archived**. |

---

## 1. Seal reviewer briefs omit the handoff contract that `seal-finalize` enforces (HIGH, verified)

Every wrapper-run seal fails its first finalize. The runtime refused a packet in which both
reviewers had passed:

> external continue result for Inspect-51 completed without a handoff; completed reviewers
> and watchers must relay changed_files, commands_run, verification_status, and
> next_worker_instructions so later phases inherit their context

**Cause (verified):**
- The seal brief, built in `cmd/seal_final_review.go` (~L1094, `# Seal Final Review`), never
  states the handoff schema.
- The continue and build briefs do state it:
  - `cmd/codex_continue_plan.go:332`
  - `cmd/codex_build.go:4679`
  - both append `codex.HandoffFieldsSummary`
- Seal results go through the continue path's `IsEmptyWorkerHandoff` guard
  (`cmd/codex_continue_finalize.go` ~L801).
- So a reviewer that follows its seal brief exactly is certain to be rejected.
- The `/ant-seal` wrapper's "collect a terminal result with…" list and the manifest's
  `dispatch_contract` (`required_result_fields` / `optional_result_fields`) also leave out
  `handoff`.

**Secondary issues:**
- The error says "external **continue** result" during a seal, which leaks the shared code
  path and confuses the operator.
- Asked to add a `freshness` timestamp with no clock available, one reviewer stamped
  `2026-09-14T09:30:00Z` at 08:43Z, which is in the future. `NormalizeWorkerHandoff` keeps any
  valid RFC3339 value, so a future timestamp persists.

**Cost:**
- An extra round-trip to both reviewers.
- A packet-rebuild cycle.
- The owner waited for no product reason.

**Fix:**
- Append the same `HandoffFieldsSummary` sentence to the seal brief.
- Add `handoff` to the wrapper's result list and to `dispatch_contract`.
- Consider stamping the receipt time when `freshness` is later than now.
- Test: build a reviewer result using only what the seal brief says, and assert that
  `seal-finalize` accepts it.

## 2. `entomb` requires `.aether/HANDOFF.md`, which `resume` deletes and seal never rewrites (HIGH, verified)

Entomb fails after any session that began with `/ant-resume`:

> entomb preflight failed: required source ".aether/HANDOFF.md": lstat …/.aether/HANDOFF.md:
> no such file or directory. Active colony state was retained; retry `/ant-entomb --confirm`
> after inspecting the named source

**Cause (verified):**
- Resume declares removal of HANDOFF.md in its transaction (`cmd/session_flow_cmds.go:680`).
  The deletion itself is already listed as a medium item in
  `.planning/audits/2026-08-30-whole-system-audit.md` ("…deletes HANDOFF.md"). The entomb
  consequence is not recorded anywhere I could find.
- Seal and seal-finalize never write the file again.
- Entomb's preflight lists it as a *required* `tombstone_input` (`cmd/entomb_cmd.go:383`).
- Downstream, `.aether/HANDOFF.md` is gitignored, so git cannot restore it.

**Warning signs the runtime gave, neither of them actionable:**
- The seal closeout printed "💾 Don't close this chat yet -- the handover note … has not been
  written to disk", with no command to fix it.
- The entomb error says "retry `--confirm` after inspecting the named source". The owner
  can't create the file, and the wrappers forbid writing it by hand.

**Workaround used:**
- `aether pause` on the sealed COMPLETED colony, which is treated as a
  `completed_episode_boundary` and writes HANDOFF.md at `session_flow_cmds.go:418`.
- Then entomb preview, then confirm. Entomb clears the pause fields.
- Finding this took reading the entomb, pause, resume and hook sources, and it cost the owner
  an extra confirmation.

**Fix (any one of these):**
- (a) seal-finalize writes the tombstone input when it seals;
- (b) entomb synthesizes the tombstone input from state when HANDOFF.md is absent, or treats
  it as optional for a verified seal;
- (c) at minimum, the entomb error names `aether pause` as the recovery.

Test: resume → seal → entomb in one flow.

## 3. Visual output never reaches the owner in Claude Code (HIGH for UX, observed by the owner)

The wrappers render ceremonies through Bash, e.g. `AETHER_OUTPUT_MODE=visual aether ceremony
spawn-plan|wave-start|worker-complete|closeout`, `aether entomb`, `aether status`. Claude Code
does not show tool output to the user. None of the spawn plan, the worker cards, the seal
summary or the entomb summary ever reached the owner.

His words: *"I can see that you're running the shell command with Aether output mode visual
but there's no actual part of that that's showing up in the actual terminal."*

The "live worker ceremony" rules don't match the host either:
- The rules say not to use background agents, and to show a visible Task stack.
- In this harness, subagents run asynchronously and report back through notifications.

**Fix:**
- The wrappers should tell the Queen to relay each key card in her own reply, verbatim or as
  a compact owner-facing summary the ceremony command emits for that purpose.
- Don't rely on tool output being visible.

## 4. JSON mode of `seal-finalize` prints the visual preflight card on stdout before the JSON (MEDIUM, verified)

With `AETHER_OUTPUT_MODE=json`, stdout began with the following, and only then the JSON
object:

```
── Seal Preflight ──
… Seal this verified colony and write its Crowned Anthill record? [y/N]
Answer with: aether decision-answer …
```

Any wrapper that parses stdout as JSON breaks, and mine did.

**Fix:** in JSON mode, send human text to stderr or carry it inside the JSON, e.g. a
`state_of_play` field.

## 5. `/ant-seal` wrapper vs runtime drift (MEDIUM, verified)

- The wrapper says to parse `result.seal_manifest` from `aether host seal`. The command
  printed a bare result object with `seal_manifest` at the top level and no `{ok,result}`
  envelope.
- The wrapper promises Gatekeeper, Auditor and Probe dispatches. The runtime dispatched only
  Auditor and Probe.
- The wrapper's terminal-result list omits `handoff` (see 1).
- The seal preflight card says **"Passed gates: 0"**, while `aether status` says **"Gates: 10
  (10 passed)"**.
- After the verified seal, `aether status` still showed "Last Autopilot Run: stopped at
  Phase 1 (genuine_stop) … Recovery: requires-attempt (systemic) … Medic advice:
  /ant-medic --deep". That is stale and alarming for a finished colony.

## 6. Seal promoted unsafe and noisy "lessons" into QUEEN.md (MEDIUM, safety)

The "Retained at seal" entries include bare shell commands. One of them copies the app's
secrets file into a temp worktree:

```
cd /private/tmp/claude/cosmic-verify-appsurf/dashboard && cp …/dashboard/.env.local . 2>/dev/null; npm install …
```

The same line is shown to the owner at session start as a "Learned habit". It contradicts
the colony's own REDIRECT ("never commit secrets or .env files") and the owner's recorded
decision that no worker reads, copies or prints the settings file.

**Fix:**
- Never promote commands that touch `.env*` or credential files.
- Don't promote bare commands as lessons.
- Filter promotions against active REDIRECT signals.

## 7. A task criterion credited without any command that verifies it (MEDIUM)

Phase 3 task 3.10's criterion includes "…and the operator broker suite all pass". The
runtime marked it passed from build, types, lint and Jest alone
(`.aether/data/build/phase-3/verification.json`). `npm run test:operator` is not among the
repo's verification commands, and Jest ignores `operator/`. That gap caused the round-1
Auditor's second blocker.

The downstream CLAUDE.md should add the command. Separately, the runtime should not credit a
criterion that no configured command covers. Its gate record is also not tied to a commit
hash, so a later fix commit silently makes it stale.

## 8. Environment impact: Aether test runs filled the owner's disk (LOW for Aether's code, HIGH for the owner)

During this session the owner's data volume reached 100%, with about 1 GB free:
- `~/Library/Caches/go-build` held about 39 GB.
- The user temp folder held about 42 GB.
- Aether `cmd.test` processes, probably from the busy `aether-c3` session, sat in
  uninterruptible I/O.

Downstream effects:
- `cp -Rc node_modules` hung, blocking clean-worktree gates.
- A Probe reviewer abandoned its worktree.
- The live app's Postgres was at risk.

The owner approved `go clean -cache`, which freed about 39 GB. Suggestion: check free disk
before long test runs, and clean `go-build` and temp output afterwards.

---

## What worked

- Resume validated the saved handoff with Confirmed provenance and restored cleanly.
- `seal-finalize` enforced the owner confirmation gate properly, and recorded the owner's
  answers.
- Entomb's staged transaction was thorough: stage, digest manifest, verify bytes and
  cross-references, publish, clear. Both failures were zero-write ("State effect: retained").
- The reviewers did real work:
  - The round-1 Auditor's LOW `gmail-fanout-deadline` finding predicted the exact failure
    the real-day test hit.
  - The round-2 Probe ran a five-mutation check on the fix.
- Refusing an empty handoff, and refusing entomb without its inputs, are the right calls.
  The defect is that nothing told the host how to satisfy them.

## Earlier issues from this same colony, for the full picture

These were already filed or recorded downstream:
- The runtime continue watcher was unreliable: it failed 3 of 5 runs on 2026-08-20, so
  wrapper-spawned reviewers were used instead.
- One builder dispatch covering many tasks conflicts with per-task claims; worked around with
  `--reconcile-task`.
- Staged `/ant-plan` could not finish (fixed in 1.0.69–1.0.70). Proof-link acceptance and the
  `--refresh` stale-candidate ambiguity were still open at 1.0.71.
- The default 10-minute worker timeout is too short for clean-worktree gates. Autopilot
  stopped with 4 good commits and 0 credited.
- Continue credited only the latest attempt, causing an infinite loop. Fixed in 1.0.75; see
  `2026-09-13-continue-credits-only-latest-attempt.md`.
- The phase commit dropped receipt-only files; see
  `2026-09-13-phase-commit-drops-receipt-only-files.md`.
- The anti-pattern gate gave false positives on fake secret-shaped test fixtures.
- A 1.0.61 seal could not be entombed by 1.0.67, because it had no `seal_outcome`.

## Suggested priority

1. **Defect 1**: one sentence added to the seal brief would remove a guaranteed failure from
   every seal.
2. **Defect 2**: every resumed session currently ends in a failed entomb.
3. **Defect 3**: the owner sees none of Aether's interface in Claude Code.
4. **Defect 6**: a safety issue, because it teaches future workers to copy secrets.
5. **Defects 4, 5 and 7**: contract drift that makes each run more fragile than it should be.
