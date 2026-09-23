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
