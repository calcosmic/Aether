# Permitted operator inputs

This file lists, for every one of the twelve benchmark cells (three lanes x
four task categories), the complete set of things the operator — the human
running the benchmark at the keyboard — is allowed to type or click while
that cell is running.

**The governing rule:** anything the operator types or clicks that is not on
the list for that cell is an unscripted intervention. It is logged as one
(via the operator log's `unscripted` input type), and it makes that
particular run non-autonomous. Logging an intervention is never treated as
an operator mistake — it is a measurement, and the measurement is the point.
This list is written down before any of the twelve runs happen, and it is
not extended after runs begin: extending it mid-benchmark would let an
intervention that actually happened during run 5 quietly become "permitted"
by the time run 9 runs into the same prompt, which would understate run 5's
true intervention count after the fact.

Each entry below states the prompt or situation, the exact permitted
response, and whether that response is an approval, a first-option
selection, or a fully scripted answer.

---

## gsd 01-bug-fix

- **Situation:** `/gsd-execute-phase` asks the operator to confirm the plan
  before executing. **Permitted response:** approve (type `y` or press
  Enter at the default). **Type:** approval.
- **Situation:** `/gsd-execute-phase` or `/gsd-verify-work` presents a
  multiple-choice decision about implementation approach. **Permitted
  response:** select the first-listed option. **Type:** first-option
  selection.
- **Situation:** Any other confirmation prompt GSD raises during this run
  (e.g. "proceed with this file change?"). **Permitted response:** approve.
  **Type:** approval.

## gsd 02-brownfield-feature

- **Situation:** `/gsd-execute-phase` asks the operator to confirm the plan
  before executing. **Permitted response:** approve. **Type:** approval.
- **Situation:** A multiple-choice decision about which existing pattern to
  follow for the new check kind. **Permitted response:** select the
  first-listed option. **Type:** first-option selection.
- **Situation:** Any other confirmation prompt GSD raises during this run.
  **Permitted response:** approve. **Type:** approval.

## gsd 03-interrupted-execution

- **Situation:** `/gsd-execute-phase` asks the operator to confirm the plan
  before executing. **Permitted response:** approve. **Type:** approval.
- **Situation:** After the harness's SIGKILL and 120-second timer fire, the
  operator resumes. **Permitted response:** run exactly
  `/gsd-resume-work` and nothing else. **Type:** scripted answer.
- **Situation:** Any other confirmation prompt GSD raises before the kill or
  after the resume. **Permitted response:** approve. **Type:** approval.

## gsd 04-fresh-repo-lifecycle

- **Situation:** `/gsd-new-project` asks for the project's initial goal or
  name. **Permitted response:** supply the task's own goal text verbatim
  (see `bench/tasks/04-fresh-repo-lifecycle.md`'s "The task" section).
  **Type:** scripted answer.
- **Situation:** `/gsd-discuss-phase` asks clarifying questions about scope.
  **Permitted response:** approve the phase as proposed, or select the
  first-listed option if a choice is presented. **Type:** approval /
  first-option selection.
- **Situation:** `/gsd-plan-phase`, `/gsd-execute-phase`, or
  `/gsd-verify-work` asks for a confirmation to proceed. **Permitted
  response:** approve. **Type:** approval.

## aether-interactive 01-bug-fix

- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim (see
  `bench/tasks/01-bug-fix.md`'s "The task" section). **Type:** scripted
  answer.
- **Situation:** `/ant-plan` or `/ant-build` presents a Queen decision or
  worker-depth choice. **Permitted response:** select the first-listed
  option. **Type:** first-option selection.
- **Situation:** `/ant-build` or `/ant-continue` asks for approval to
  proceed to the next stage. **Permitted response:** approve. **Type:**
  approval.

## aether-interactive 02-brownfield-feature

- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** `/ant-plan` or `/ant-build` presents a Queen decision.
  **Permitted response:** select the first-listed option. **Type:**
  first-option selection.
- **Situation:** `/ant-build` or `/ant-continue` asks for approval to
  proceed. **Permitted response:** approve. **Type:** approval.

## aether-interactive 03-interrupted-execution

- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** `/ant-build` presents a Queen decision before the kill.
  **Permitted response:** select the first-listed option. **Type:**
  first-option selection.
- **Situation:** After the harness's SIGKILL and 120-second timer fire, the
  operator resumes. **Permitted response:** run exactly `/ant-resume` and
  nothing else. **Type:** scripted answer.
- **Situation:** Any other approval prompt before the kill or after the
  resume. **Permitted response:** approve. **Type:** approval.

## aether-interactive 04-fresh-repo-lifecycle

- **Situation:** Installing the `aether` CLI into the fresh directory asks
  for confirmation. **Permitted response:** approve. **Type:** approval.
- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** `/ant-plan`, `/ant-build`, `/ant-continue`, or `/ant-seal`
  present a decision or approval prompt. **Permitted response:** select
  the first-listed option, or approve if no options are listed. **Type:**
  first-option selection / approval.

## aether-autopilot 01-bug-fix

- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** `/ant-run` pauses for a smart-pause condition (test
  failure, quality gate, replan suggestion). **Permitted response:**
  approve continuing, or select the first-listed option if a choice is
  presented. **Type:** approval / first-option selection.

## aether-autopilot 02-brownfield-feature

- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** `/ant-run` pauses for a smart-pause condition. **Permitted
  response:** approve continuing, or select the first-listed option.
  **Type:** approval / first-option selection.

## aether-autopilot 03-interrupted-execution

- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** After the harness's SIGKILL and 120-second timer fire, the
  operator resumes. **Permitted response:** run exactly `/ant-resume` and
  nothing else. **Type:** scripted answer.
- **Situation:** `/ant-run` pauses for a smart-pause condition before the
  kill or after the resume. **Permitted response:** approve continuing, or
  select the first-listed option. **Type:** approval / first-option
  selection.

## aether-autopilot 04-fresh-repo-lifecycle

- **Situation:** Installing the `aether` CLI into the fresh directory asks
  for confirmation. **Permitted response:** approve. **Type:** approval.
- **Situation:** `/ant-init` asks for the colony's goal. **Permitted
  response:** supply the task's own prompt text verbatim. **Type:**
  scripted answer.
- **Situation:** `/ant-plan` presents a phase-breakdown decision. **Permitted
  response:** select the first-listed option. **Type:** first-option
  selection.
- **Situation:** `/ant-run` pauses for a smart-pause condition, and finally
  `/ant-seal` asks for confirmation to seal the colony. **Permitted
  response:** approve continuing / approve the seal. **Type:** approval.

---

Any input not described above — a correction typed into a running system, a
re-phrasing of the goal, a manual file edit outside the documented resume
command, a retry after an unexpected failure — is an unscripted intervention.
Log it with `oplog_input unscripted "<what was typed>"` and continue; do not
abandon the run and do not silently absorb it into a "clean" result.

For an Aether interruption, `/ant-resume` validates a clean handoff or
reconstructs the safest honest point from durable evidence. If it reports a
conflict, the operator may log an explicit expert diagnostic using
`/ant-maintenance recovery-inspect`, then issue a fresh `/ant-resume`; that
diagnostic is never silently appended to the ordinary scripted action.
