# Aether Hardening Plan

**Written:** 2026-08-14
**Status:** APPROVED by owner 2026-08-14 — "go ahead, cut them all"
**Applied to:** `.planning/ROADMAP.md` and `.planning/STATE.md` on 2026-08-14
**Supersedes:** phases 175–179 of milestone v1.26 "Intelligent Orchestration"

Hardening items H1–H5 in this document became roadmap phases **180–184**.
The rescued cost line became phase **185**. Order: 180 → 181 → 182 → 183 →
184 → 185 → 175 → 179.

---

## How to use this document

If you are an AI assistant picking this up in a fresh session with no memory of
the conversation that produced it: **this document is the current plan.** It
overrides `.planning/ROADMAP.md` for everything numbered 175 and above. Read
this file first, then `.planning/STATE.md` for position.

The owner is non-technical. Do not explain work in terms of file paths,
function names, or this repo's invented vocabulary. Say what a change does for
a person using Aether.

---

## The decision, in one paragraph

Aether's engine is sound. The problem is that it spends a large, fixed amount
of effort on every job regardless of how big the job is, and it hands every
worker a pile of instructions that have nothing to do with the task. The next
piece of work is not new capability. It is **removing things** until a small
job costs a small amount. After that, we stop building and start using it.

---

## What proved this

On 2026-08-14 Aether ran a real job in a real project (CalVault — an Obsidian
notes vault with no program code in it). The job: copy 110 markdown files from
known locations into a new folder tree. All the hard thinking had already been
done and verified in an earlier phase.

Three workers ran. They produced twelve empty folders, one placeholder note,
and two reports. **Zero of the 110 files were copied.** Cost: 256,292 tokens
and 12.5 minutes. Seven more workers were queued.

Creating twelve folders and one small file cost 51,914 tokens on its own.

Causes, all confirmed in Aether's own code:

1. **Every worker is handed ~28KB of instructions, most of it irrelevant.** A
   worker whose entire job was making folders received a design guide covering
   AI model selection, GDPR and HIPAA, plus a document describing Aether's own
   internal Go and TypeScript architecture. Neither has anything to do with an
   Obsidian vault.

2. **Helper documents are chosen by the worker's job title, not by the task.**
   In the scoring that picks them, "what the task actually is" is worth 2
   points; "what the worker's job title is" is worth 3, and "what kind of
   output is expected" is worth 4. Task relevance is the *lowest*-weighted
   signal in the system.

3. **Aether ships 73 of its own internal reference documents (380KB) onto the
   machine, and every project reads from that shared folder.** One of them —
   describing which parts of Aether are written in Go versus TypeScript — is
   tagged `priority: critical` and sent to every worker with the job title
   "builder", in every project.

4. **Specialists are summoned by keyword-matching the phase description.** If
   the text contains any of *security, auth, crypto, secret, token, permission,
   credential, password, compliance, release, sign-off*, a security specialist
   is dispatched. The CalVault phase said its source folders were "read-only"
   and warned workers about a plugin. That was enough to summon a 107,155-token
   security review of a file-copying job.

5. **The rule that should have skipped the test-coverage specialist cannot
   fire in practice.** To be classified "no code here", a phase must contain one
   of *documentation, readme, changelog, guide, manual, doc, docs, tutorial* AND
   contain none of *implement, build, refactor, migrate, endpoint, api, function,
   component, module, schema, test, fix, bug*. A phase about copying markdown
   notes contains no word from the first list, so it is classified as code work.
   Even if it did, the word "build" in the second list would disqualify it — and
   the command that runs a phase is called build.

6. **The worker limit does not limit.** The count carries over between
   commands instead of resetting, and it reports going over rather than
   stopping. The failure log shows 27 workers and 37 workers against a cap of
   20.

7. **One job gets split across many workers.** Six file-copy batches became six
   separate workers, each a fresh ~100,000-token agent re-reading the same list
   from scratch.

**In fairness:** the security specialist did find a real and expensive problem
— an Obsidian plugin that would have silently renamed all 110 recovered files.
But that exact warning was already written into the next worker's instructions
before the specialist ran. It spent 107,155 tokens rediscovering something the
planner had already worked out. Real finding, no new information.

---

## What we are stopping

| Item | Decision | Why |
|---|---|---|
| **Phase 174 Spend Ledger** — plans 3–9 | **Cut** | Plans 1 and 2 are done and already deliver the token measurement we wanted. The rest is bookkeeping machinery around a number we can already read. |
| **Phase 176 Roster Ruling** | **Cut** | Tidying files nothing reads. Invisible to a user. |
| **Phase 177 Worker Delegation Grant** | **Cut** | Lets workers spawn more workers. We are trying to reduce worker count, not add a mechanism for more. |
| **Phase 178 Skill Authoring Hardening** | **Cut** | Letting users author their own skills is a new feature, not hardening. |

## What we are keeping

| Item | Decision | Why |
|---|---|---|
| **Phase 174** — plans 1 and 2 | **Already done, keep** | This is the "what did that cost" number. It shipped. |
| **Phase 175 Orchestration Visibility** | **Keep, do after hardening** | "Here is who I sent and why" in plain English. Directly addresses the owner having stopped reading output. |
| **Phase 179 Proof** | **Keep, becomes the final gate** | Run real tasks and confirm it works. Was scheduled last; stays last, but is now the actual finish line rather than a formality. |

---

## The hardening work

Five pieces. Every one of them **removes** something. None of them add a
feature. Ordered by how much they fix the observed problem.

### H1 — Stop sending Aether's own internals into other people's projects

Nothing about Aether's architecture, its Go code, its TypeScript host, or its
internal contracts should ever reach a worker in an unrelated project.

**Done when:** a worker running in a project that is not Aether receives zero
Aether-internal reference documents, proven by a test that fails if one
appears.

### H2 — Choose a worker's reading material by the task, not the job title

Task relevance must outrank job title and expected-output-type in the scoring.
When nothing scores well, send nothing.

**Done when:** a task whose verbs are "copy files" and "make folders" receives
no design guide, no architecture contract, and no AI-integration material —
proven by a test using the real CalVault task text.

### H3 — Stop summoning specialists on word-mentions

A phase that *warns about* a risk currently reads identically to a phase that
*creates* one. Replace keyword-spotting on the phase description with a
judgement about what the phase actually changes.

**Done when:**
- A file-copying phase that mentions "read-only" summons no security
  specialist.
- A phase that genuinely changes login, passwords or tokens still does.
- A phase that writes no program code summons no test-coverage specialist,
  and this holds for a notes vault, not just for text containing the word
  "documentation".

### H4 — Make the worker limit actually stop things

The count must reset when a new command starts, and must refuse or trim before
dispatch rather than reporting an overrun afterwards.

**Done when:** a run that would exceed the limit is trimmed or refused up
front, and the failure log stops recording counts above the cap.

### H5 — Let one worker own a run of related file work

Sequential file operations on the same list should not become six isolated
workers each paying full startup cost.

**Done when:** a phase of six dependent file-copy steps dispatches one worker,
not six.

### H6 — Swap the reviewers (applies to how we work, not to Aether)

Two kinds of check run on our own work here. Their measured record:

- The "did we achieve the goal?" review: 168 documents produced, ~19,700 lines,
  **zero real problems found** in a sample of six, and two occasions where it
  certified code that had never been written.
- The "read the actual code" review: 233 real findings, and it caught two
  serious defects on the same phase, the same day, that the first one had
  passed 10/10. It currently runs on 16% of the work.

Turn the first off. Turn the second on for everything. No new code; a settings
change.

---

## The settings change

Our own working config was empty, which means every optional step was running
at its heaviest default — nobody chose it, it just was. Turned down to:

- goal-verification review: **off** (see H6 — zero yield, two false sign-offs)
- test-strategy template pass: **off** (half never leave draft)
- research-before-planning: **off by default**, switched on per phase when the
  work genuinely needs it
- post-planning gap scan: **off**
- security verification pass: **off by default**, on for phases that touch
  security
- AI-integration phase: **off** (we are not building AI features here)
- plan checking before execution: **stays on** — it is the one gate that
  blocks bad plans before they cost anything
- code review: **stays on**, and expands to every phase

---

## The finish line

**Aether is done when all three are true:**

1. A small job costs one or two workers and finishes in minutes.
2. You can read one screen that tells you who was sent, why, and what it cost.
3. It never sends Aether's own internals into someone else's project.

That is H1–H5, plus Phase 175, confirmed by Phase 179 running real tasks in a
real project.

**When those are true, we stop building.** Aether gets touched again only when
using it annoys someone. Everything currently on the roadmap beyond this list
is the system improving itself for its own benefit.

---

## Why we are keeping GSD

The planning system we use to build Aether has been expensive — one recent
phase produced 37 documents and 104,530 words to ship about 1,550 lines of
logic. But it solves a real problem: **work survives when a session runs out
of context.** That is why this document exists as a file rather than a chat
message.

We are not replacing it. We are turning its dials down (see settings above)
and giving it much smaller work to do.

---

## Progress log

Append one line per completed item. Keep it short.

| Date | Item | Result |
|---|---|---|
| 2026-08-14 | Plan written | Approved by owner; applied to ROADMAP.md and STATE.md |
| 2026-08-14 | **H2 / Phase 181 — reading material chosen by the task** | **Done.** Task relevance went from the lowest-weighted signal (2, below job title 3 and output type 4) to the highest (6) and is now *required*, not rewarded. Whole-word matching replaced a comparison that stripped every space and then asked whether either string contained the other. Skills got the same treatment. Measured on the real CalVault task text: references 5 → 0, skills → 0. Controls hold: a genuine AI-design task still matches `ai-design-contract`; a Go task still matches the Go skill; a README task still matches the documentation skill. Ties break on content so renaming cannot win a slot; over-budget skills are dropped whole rather than cut mid-sentence, and the slicing helper is deleted. Full suite green. Published. **Residue:** the `accessibility` skill matches most tasks via workspace detection, and `ai-design-contract` still matched "fix the failing test in the reference matcher". Both are keyword-quality problems in the skill definitions, not in the selection mechanism — worth a sweep, not a phase. |
| 2026-08-14 | **H1 / Phase 180 — Aether stays out of other projects** | **Done.** 22 of 73 reference documents marked `scope: aether-internal`; they now load only from Aether's own authored library. Verified by running the real command from a non-Aether directory: before, all five documents a builder received were about building and shipping Aether, including the release-publishing playbook; after, none are. Aether's own workers still receive them — asserted in both directions. Full test suite green. Published to the hub, so it reaches CalVault. |
