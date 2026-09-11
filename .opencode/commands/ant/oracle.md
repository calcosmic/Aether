<!-- Aether-managed: runtime spec at .aether/commands/oracle.yaml. Synced by aether update. -->
---
name: ant-oracle
description: "🔮 Run the autonomous Oracle loop through the Aether CLI runtime"
---

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether oracle "$ARGUMENTS"` directly only for `status`, `stop`, or when the user explicitly asks to skip refinement.
- For inspection or control, prefer `aether oracle status` and `aether oracle stop`.
- Do not describe legacy loop control files or shell-managed orchestration from this command spec.
- Report the CLI result directly.

## Intent Refinement

When the user provides a research topic, run a short scoping pass before
starting the loop. The runtime owns the proposals; you conduct the conversation.

For beginners: this is the part where you turn "look into this" into a useful
research brief, so Oracle does not spend iterations answering the wrong
question.

**Step 1 — ask the runtime what it suggests.** This writes nothing:

```bash
AETHER_OUTPUT_MODE=json aether oracle propose --topic "<the user's raw topic>"
```

It returns a suggested output shape, evidence sources, depth, accuracy target,
and how many clarifying questions the topic needs, alongside every option the
user can pick from with its round count and rough duration. Every suggestion is
derived from keywords — present them as pre-selected choices the user can
change, never as decisions already made.

**Step 2 — ask how much scoping the user wants** before asking anything else.
Offer these as selectable options, pre-selecting the `clarifying_questions`
count from the proposal:

- **Skip** — I know exactly what I want (0 questions)
- **Quick** — 3 questions
- **Standard** — 6 questions
- **Thorough** — 10 questions

**Step 3 — ask that many questions**, drawn from whichever of these are still
unclear, most decision-shaping first:
- the decision the user needs to make
- the desired output shape (PRD, tech evaluation, architecture review, bug
  investigation, research brief)
- target users or affected worker roles
- constraints, non-goals, deadlines, or risk tolerance
- evidence sources to prefer: repo, web/current docs, or both
- what would make the answer actionable

**Step 4 — confirm depth and target accuracy** as selectable options, seeded
from the proposal:

| Depth | Rounds |
|---|---|
| `quick` | up to 5 |
| `balanced` | up to 15 |
| `deep` | up to 30 |
| `exhaustive` | up to 50 |

Accuracy targets are 80% (first pass), 90% (solid), 95% (thorough,
recommended), 99% (near-exhaustive). If the user gives an exact round cap, pass
`--max-iterations <1-50>`.

The PRD template/reference is automatic. Do not ask the user to run
`aether reference-match`; that command is only diagnostic.

## Research Brief Gate

Before any tokens burn on the loop, record the synthesized brief. The runtime
renders the approval panel — do not hand-draw it:

```bash
AETHER_OUTPUT_MODE=visual aether oracle brief \
  --topic "<one-line topic>" \
  --core-question "<the single question the loop must answer>" \
  --context "<why now, what decision this feeds>" \
  --success-criteria "<what a done answer contains>" \
  --success-criteria "<another criterion>" \
  --depth <depth> --confidence-target <percent> --template <template> --dry-run
```

Show the rendered panel and ask: **approve**, **edit** (revise a field and
re-present), or **cancel**. Maximum 2 edit rounds — after the second, run with
the latest brief or cancel. On approval, run the same command **without**
`--dry-run` to record it.

The Core Question must be a genuine question, scoped to one decision — never a
directory listing, a task list, or "everything about X". The runtime rejects a
question with no question mark or one longer than 240 characters. If the user's
topic is that broad, split it into focused briefs and gate each one.

The approved question becomes the loop's first research question, and the brief
is recorded with the finished research so it says what it set out to answer.

**Focus signal:** if the approved brief names a specific area of the codebase
or a constraint the colony should honor during upcoming builds, offer once to
record it: `aether focus "<area or constraint from the brief>"`. Write it only
with the user's approval — research topics are not automatically colony
steering.

## Running The Loop

Start from the approved brief and stream its progress:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --from-brief --background --follow
```

`--from-brief` takes the topic, depth, accuracy target, sources and output shape
from the approved brief, and **fails if no brief has been approved** — that
refusal is the gate, so do not work around it by passing the topic directly
unless the user explicitly asks to skip scoping.

`--background` detaches the controller so a long run cannot time out this tool
call. `--follow` then streams one line per round — phase, round number, current
confidence against the target, and the question being investigated — until the
run ends. Both together are the normal path.

`aether oracle status` remains the inspection path and never modifies the run.
If it reports a stale controller, `aether oracle recover` clears it.

A running round is also visible in the live colony dashboard: `aether watch` in
another view shows the round number, its cap, the current question, confidence
against target, and any newly found contradiction or gap, live and in place —
not only through this command's own `--follow` polling.

When the runtime detects a hosted Claude/OpenCode agent session and the command
is not already backgrounded, it auto-detaches the Oracle controller. Treat that
as a normal background run.

**Before a long run, prove the machinery works:**

```bash
aether oracle selftest --dry-run   # setup checks only; spawns nothing
aether oracle selftest             # one real round end to end
```

`selftest` reports which dispatcher it picked, resolves the agent definition,
runs a single real research round in a throwaway workspace, and exits non-zero
if any part of that fails. Use it whenever the user doubts Oracle is working.

## Reporting The Answer

When the run finishes, report the answer recommendation-first: lead with the
actionable recommendation in plain language, state confidence and any
unresolved questions honestly right beneath it, and keep the full source and
evidence trail further down for anyone who wants to read it. Never lead with
sources or the evidence trail — the runtime's own saved write-up is already
ordered this way; report it in that order rather than reshuffling it.

## Promoting Findings Into Colony Memory

When a research loop completes (or the user asks to capture what Oracle
learned), offer promotion:

```bash
AETHER_OUTPUT_MODE=json aether oracle promote --dry-run   # preview
AETHER_OUTPUT_MODE=json aether oracle promote             # write
```

Promote takes every finding from questions at or above 80% confidence
(`--min-confidence` to change) and stores the admissible ones as learnings and
instincts. Findings that name no file, command, or error are rejected with the
reason shown — that is the admissibility gate working, not a failure. Show the
user the outcome table: what was promoted, what was skipped, and why. Colony
steering (FOCUS/REDIRECT) stays a separate, user-approved step.

## Broad Scope And Timeout Handling

If the user asks for "everything", "all of the above", a full-system audit, or a
large uncommitted diff review, do not collapse every concern into one blocking
balanced-depth Oracle prompt. Either split the work into focused Oracle prompts
or run an explicit quick triage first:

```bash
AETHER_OUTPUT_MODE=visual aether oracle --depth quick --confidence-target <percent> --template <template> "<focused triage prompt>"
```

If the shell/tool call times out, immediately inspect the runtime state:

```bash
aether oracle status
```

Report that status. Do not assume Oracle failed, and do not bypass it with ad
hoc agents until the runtime status says it is blocked, stopped, or complete.
If OpenCode subprocess dispatch is unavailable, let Oracle use its automatic
Codex/Claude fallback unless the user explicitly set `AETHER_WORKER_PLATFORM=opencode`.
Do not fake Oracle worker completion; if the runtime reports no dispatcher,
preserve the Oracle workspace and surface the blocker plainly.

## Cross-Platform Drift Guard

If you change Oracle interview, template selection, persistence suggestions, or
closeout behavior here, update `.aether/commands/oracle.yaml`,
`cmd/command_guide.go`, and the Codex skill `aether-colony-research` in the same
change. Verify `aether command-guide oracle --platform codex` still describes
the matching Codex flow.

## Post-Completion Persistence Suggestions

After the oracle loop completes and you have reported the research results, check whether the findings are worth preserving for the colony:

**Only suggest persistence when:**
- The oracle completed successfully (status is "complete" or "max_iterations_reached")
- Do NOT suggest persistence if the oracle was blocked or stopped early

**How to identify high-value findings:**
- Findings with high confidence that apply to the colony's current goal
- Actionable recommendations that would benefit future workers
- Domain-specific insights that are not obvious from the codebase alone
- Architecture or design decisions documented in the research

**Suggestion format:**
Present findings worth preserving as a tick-to-approve list:

```
🔮 Research findings worth preserving:

1. [x] {finding title}
   → Suggest: aether pheromone-write --type "FOCUS" --content '{"text":"{finding summary}"}' --priority "normal" --source "oracle" --reason "High-value research finding"

2. [ ] {finding title}
   → Suggest: aether hive-store --text "{generalized finding}" --domain "{detected domain}" --source-repo "{repo name}"
```

For each approved finding:
- If it is colony-specific guidance: use `aether pheromone-write --type "FOCUS" --content '{"text":"{finding}"}' --priority "normal" --source "oracle" --reason "High-value research finding"`
- If it is generalizable cross-colony wisdom: use `aether hive-store --text "{generalized finding}" --domain "{domain}" --source-repo "{repo name}"`
- Let the user approve each suggestion before persisting

**Do NOT suggest persistence for:** low-confidence findings, obvious observations, or findings already captured as pheromones.

## Keeping The Research

A completed run saves its write-up to `.aether/research/<date>-<slug>.md`, with
front matter recording the core question, the confidence reached, and how many
rounds it took. That directory is durable: the next research run archives the
working files but never touches it.

Use `aether research list` to browse saved research, and `aether oracle save` to
keep the current run's write-up even when it stopped early.

Point a later colony at one of those documents so its planners read the research
instead of rediscovering it:

```bash
aether init --research .aether/research/<file>.md "<goal>"
aether plan --research .aether/research/<file>.md
aether plan --print-brief
```

`plan --research` attaches a document to a colony that already exists, and
`plan --print-brief` confirms the research reaches the planners before a plan
run is paid for.

**Next steps:**
- `aether research list` — browse research saved from completed runs
- `aether oracle status` — check a running loop without touching it
- `aether oracle status --follow` — reattach to a run and watch its rounds
- `aether oracle recover` — clear a run whose controller died
- `aether oracle selftest` — prove the research machinery works
- `aether oracle promote --dry-run` — preview capturing findings into colony memory
- `/ant-plan` (refresh with `--revision-evidence`) — fold findings into the plan
