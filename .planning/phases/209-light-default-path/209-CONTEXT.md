# Phase 209: context

**Requirements:** UED-16 to UED-18 (owner decisions needed before planning). **Decision:** `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`. **Research:** `.planning/research/2026-09-21-popular-frameworks.md`, `.planning/research/2026-09-21-reliability-and-delivery.md`.

Not yet planned. Keep it small; the proof is a real run in a real chat; one review round.

## Owner decisions (2026-09-24, before planning)

Recorded in answer to the ROADMAP's own precondition for this phase
("Depends on: Phase 208; the owner's decisions on the details before
planning"). Three questions were put to the owner; all three are answered
below. Nothing in this phase may be planned against a guess where a ruling
exists here.

<decisions>

- **D-01 — One entry command, and it is a new name: `/ant-go "<what you want>"`.**
  [user-facing]
  The owner chose a new single door over reusing either existing name.
  Reasoning recorded with the choice: `/ant-init` promises "start a project"
  and would read wrongly when the job is a one-line fix; `/ant-quick` promises
  "small" and would read wrongly when the job turns out to be a week of work.
  `/ant-go` means exactly what it does and carries no wrong promise either way.
  - `/ant-go "<what you want done>"` is the one command for all ordinary work,
    from a typo fix to a whole feature. The program decides small or big from
    the size of the change, not from how the sentence is worded, and says which
    it picked and why in plain English before it starts.
  - `/ant-init` and `/ant-quick` keep working exactly as they do today for
    anyone who types them. They are not deleted, not deprecated in behaviour,
    and not rewired. They come off the default menu (see D-02) and move to the
    advanced set.
  - Rejected: reusing `/ant-init` as the single door (name promises a project);
    reusing `/ant-quick` (name promises smallness).

- **D-02 — About six commands on the default menu; the owner reviews the real screen before the set is locked in.** [user-facing]
  The owner declined to pick from a list and asked to judge the rendered screen
  as a user. Planning therefore proceeds on a *proposal*, and the phase carries
  a real owner checkpoint on the drawn screen — not a paragraph describing it.
  - **Proposed default six** (the starting point, not the ruling):
    1. `/ant-go "<what you want>"` — do the work
    2. `/ant-status` — where am I
    3. `/ant-continue` — check what was built and move on
    4. `/ant-flags` — what is waiting on my decision
    5. `/ant-resume` — pick back up after a break
    6. `/ant-seal` — mark it finished
    `/ant-help` itself is always reachable; it is the screen being shown, not a
    seventh entry on it.
  - Deliberately NOT in the proposed six, and why: `/ant-pause` (closing the
    chat and running `/ant-resume` later already works — the deliberate stop is
    an advanced move); `/ant-init` and `/ant-quick` (superseded as the everyday
    door by D-01); `/ant-plan` and `/ant-build` (reached through `/ant-go`, not
    typed by hand on an ordinary job).
  - **The checkpoint is mandatory and is a real screen.** The plan must stop and
    show the owner the actual rendered default menu — the bytes the program
    draws — and take his ruling on the six before the set is treated as
    settled. A described or summarised menu does not discharge this decision.
    His answer may add, drop or reorder entries; the plan must be able to absorb
    that without replanning.
  - Everything not on the final six stays fully working and reachable; it is
    hidden from the default menu only, behind one setting the owner can switch
    on. Hiding a command must never disable it.

- **D-03 — When "small" turns out to be big, the program switches by itself and says so in one line.** [user-facing]
  - The quick attempt stops, the job moves up to the planning route on the
    program's own authority, and one plain-English line says it was bigger than
    it looked and what made that clear. No question is put to the owner, and
    nothing waits on an answer.
  - Rejected: stopping to ask first (puts a question in front of the owner
    mid-job); finishing the small attempt and reporting afterwards (risks
    leaving half-done work that looked finished).
  - This is a one-way move only. Nothing in this phase moves a job *down* from
    the planning route to the quick route automatically.

</decisions>

## Standing constraints on this phase (from the milestone ruling)

These are not new decisions; they are the milestone's own rules restated so a
planner cannot plan past them.

- **No new strict rules.** `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`
  forbids adding a new way for the program to refuse the owner. The size router
  (D-01) and the escalation (D-03) derive, say what they decided, and continue —
  they never refuse.
- **Nothing existing is taken away.** Every command that works today still works
  after this phase. D-02 hides; it does not remove, disable or deprecate.
- **The proof is a real run in a real chat, never paperwork.** One review round.
- **Success criterion 4 is a measurement, not a gate.** The same small and
  medium jobs are timed against plain Claude and the numbers are reported to
  the owner. A slower number is reported honestly; it does not block the phase.

## Open items carried in, not closed by this phase

- `.planning/WINDOWS.md` row 53 (the out-of-date map of the code is never
  refreshed during the practice walk) is still open at the time of planning and
  is Phase 208's unfinished business, not this phase's. Do not plan work against
  it here.

## Planning-time rulings (assistant, 2026-09-24 — NOT owner decisions)

Two scope questions surfaced by the pattern-mapping pass. Both are answered by
the milestone ruling and by a hard technical fact, so they were decided here
rather than put to the owner. Either can be overturned by the owner at the
D-02 checkpoint without replanning.

- **A-01 — The advanced-commands setting is machine-wide, not per-project.**
  It lives with the owner's other preferences in the installed copy on his
  machine, not inside one project's saved state. Decisive reason: the menu is
  shown in a folder with no project set up in it, and a per-project setting
  cannot be read there — the toggle would be unreadable exactly where a new
  user first meets the menu. Supporting reason: "show me everything" is a fact
  about the person, not about one project; the owner works across several real
  projects and should not have to switch it on in each.

- **A-02 — Success criterion 4 is a once-off measurement written up for the
  owner, not a new automated timing harness.**
  The milestone ruling forbids new features and says the proof is a real run in
  a real chat, never paperwork; the criterion's own wording is "the numbers
  reported to the owner". Building a harness that times Claude against Claude
  is the inward-turning ceremony that ruling exists to prevent. The phase
  therefore runs the same small job and the same medium job twice — once
  through Aether's new single door, once through plain Claude — records wall
  time and cost for each, and writes the four numbers plus an honest verdict
  into the phase report. A slower number is reported, not hidden, and does not
  block the phase.

## Probe results carried into planning (deterministic, 2026-09-24)

The spec-less edge probe was run against UED-16, UED-17 and UED-18 (this phase
has no SPEC document, so the fallback path applies). It returned **3 applicable,
0 resolved, 3 unresolved — all three classified `unclassified`**.

Under the probe's own rules an `unclassified` row is never auto-resolved and
never silently dropped: each of the three must appear in the plan as an
explicit, flagged planner assumption naming what was not settled, and the count
must balance (3 surfaced == 3 authored or flagged).

Three further gates were run and all three came back negative, so their
checkpoints do not apply to this phase and were not raised: the external-API
coverage checkpoint (no external API in scope), the assumption-delta identity
checkpoint (no singular-to-plural transition detected), and the database schema
push gate (this is a Go project with no ORM schema files).
