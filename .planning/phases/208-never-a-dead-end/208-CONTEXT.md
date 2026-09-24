# Phase 208: context

**Requirements:** UED-10 to UED-15. **Decision:** `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`. **Research:** `.planning/research/2026-09-21-popular-frameworks.md`, `.planning/research/2026-09-21-reliability-and-delivery.md`.

Not yet planned. Keep it small; the proof is a real run in a real chat; one review round.

## Gap-closure decisions (owner, 2026-09-23)

Recorded after 208-VERIFICATION.md reported `gaps_found` on ROADMAP Success
Criterion 2 (second clause) and asked for a ruling on WINDOWS.md row 55.

**D-01 — Ask-vs-act on a stop-and-confirm refusal (WINDOWS row 55): "Act when alone, ask when you're there."**
When a `ProtectsWork: true` stop refusal fires and names its `NextCommand`
(e.g. `colonize-existing-survey-found` → `aether colonize --force-resurvey`):
- In an interactive chat with the owner present, the assistant asks first —
  exactly the behaviour observed in the 208-08 run. Unchanged.
- In an unattended chat with no owner to answer (an automated `claude -p`
  chain such as the journey harness), the assistant runs the named
  `NextCommand` itself and carries on.
- The rule must be stated in `.aether/commands/colonize.yaml` (and any
  sibling wrapper carrying the same "follow the runtime recovery guidance"
  instruction), and must be decided by an observable fact about the session
  (is there a person who can answer?), never inferred from wording.
- Rejected: "always ask + harness answers as the owner" (more to build, every
  new stop needs a scripted answer) and "always act" (could replace saved
  work in a real project without checking — contrary to the phase goal).

**D-02 — Paid proof: "One walk first, then decide."**
The plan may run exactly ONE real journey walk (`AETHER_JOURNEY_TRIALS=1
make eval-gate-journey`; the last one cost $0.93 and 90 s, a walk that gets
further may cost up to ~$25). If that walk clears the survey step, stop and
report to the owner before the three-walk release gate is run. The three-walk
gate is NOT authorised by this decision. A single-trial run is a measurement,
never a passed gate (`journeyGateVerdict` must keep refusing it).

## Second gap-closure round (owner, 2026-09-23, after the 208-10 walk)

Recorded after the re-verification (`208-VERIFICATION.md`, 2026-09-23T16:20)
kept `gaps_found` on the same single clause. Three real walks have now stopped
at the same step (`survey`, 2 of 14) for three different causes in turn: a
missing `generated_at` field (a code defect, fixed in 208-01/208-07), an
ambiguous wrapper instruction (fixed by D-01 in 208-09), and — in the 208-10
walk, with D-01's guidance sentence rendering verbatim and correctly in the
real transcript — a driving chat that read "do not ask first" and asked the
owner anyway.

**D-03 — Close it in the runtime, not in the instruction: "The program does it
itself when nobody is there."**

The owner's ruling, put to them with four options and chosen deliberately:
when Aether can observe that no person is present to answer
(`sessionHasNoOneToAsk()`, the one fact D-01 already established), the runtime
performs the recovery itself rather than printing an instruction and depending
on the chat to follow it. The out-of-date territory snapshot is refreshed by
the program on the path that detects it, so no chat cooperation is required
for the journey to get past that step.

What this does NOT change:
- Attended behaviour. With a person present the refusal still stops and asks,
  byte-for-byte as today. D-01's interactive half stands unchanged, and
  `TestAttendedRefusalTextIsUnchanged` must keep passing.
- The guidance sentence itself. D-01's unattended sentence stays where it is;
  it becomes a belt-and-braces explanation of what the program already did,
  not the mechanism.
- The refusal contract. No refusal may be deleted, downgraded from `stop` to
  `warn`, or stripped of its `NextCommand` to get past this.

Why the alternatives were rejected, in the owner's framing:
- *Make the instruction harder to ignore* (a Stop-hook-style check that sends
  the chat's reply back when it asks a question nobody can answer): still
  depends on the chat eventually cooperating, and costs a model turn each
  time. The repository already records this exact lesson for screens — "a
  wrapper can ask the assistant to relay the screen, but a request is not a
  mechanism" (CLAUDE.md, Phase 205 Part C) — and it was not carried across to
  refusals. This round carries it across.
- *Change what the rehearsal tests here* (drop the stale-snapshot trap so the
  walk reaches the later steps): abandons a real dead end a real user can hit.
- *Leave it open and move on*: rejected; the release check still could not run
  end to end.

**D-04 — The paid-walk rule is unchanged and still binding.**
D-02 continues to govern: this round may run exactly ONE real walk, and the
three-walk release gate remains unauthorised. A single-trial run stays a
measurement, never a passed gate. If the walk gets past `survey` and stops
somewhere new, that is reported to the owner before anything further is spent.

## Third gap-closure round (owner, 2026-09-24, after the 208-12 walk)

Recorded after the second re-verification (`208-VERIFICATION.md`,
2026-09-23T18:10) kept `gaps_found` for two reasons: the pre-existing live
proof of Success Criterion 2's second clause, and a NEW finding this pass made
on its own initiative — two of 208-11's own declared must_haves are enforced by
tests that cannot fail, independently reproduced by mutation in a disposable
worktree.

**D-05 — Fix the cause first, then spend exactly one more walk.**
The owner's ruling, chosen from three options: diagnose and fix the
`source_revision` mismatch locally, prove the fix on this machine, and only
then spend ONE real walk. The three prior walks cost $0.93, ~$1 and $1.84 and
took 90s / 86s / 171s — well inside the owner's approved ~£12–25 range. If the
walk stops again, stop there and report; no second paid walk this round, and
no permission to fix-and-retry a new stopping point without asking again.
Rejected: "fix only, spend nothing" (leaves the whole question unanswered
again) and "try once, and again if it moves forward, up to ~£5" (the owner
kept the one-walk-per-round discipline of D-02/D-04 rather than widening it).
D-02 and D-04 remain binding and are extended, not replaced: one walk, a
single-trial run is a measurement and never a passed gate, and
`journeyGateVerdict` must keep refusing it.

**D-06 — If it still does not run through, the phase closes with the gap named.**
The owner's ruling: once the two unfalsifiable guards are genuinely fixed,
Phase 208 is signed off even if no real walk has ever carried the practice
project past step 2 of 14. "Never proven end to end" is carried forward as a
named, visible entry on `.planning/WINDOWS.md` (row 53), honestly open with the
current evidence, for Phase 209/210 to pick up — never quietly dropped, never
re-described as satisfied. Rejected: keeping the phase open until a real walk
runs through, however many rounds and however much money that takes.

**What this round covers, in scope order:**
1. The two guards that cannot fail (the one-decision guard, and the guard that
   is meant to prove only a declared refusal is ever recovered from). Full
   rigour: each fix proved by re-running the exact planted mutation the
   verification reproduced, confirming it now fails, then restoring.
2. The two owner-facing wording defects in the same file: the notice claiming
   Aether "ran" a command it has only queued on the plan-only lane, and the
   notice's reason string printing planning-decision IDs and a `.planning/`
   filename to the owner's screen.
3. The `source_revision` mismatch: traced and fixed locally with a test that
   fails without the fix.
4. One authorised walk, after 1–3 land, measured and honestly recorded.
5. Close the phase per D-06, with row 53 left honestly open if the walk does
   not clear the survey step.
