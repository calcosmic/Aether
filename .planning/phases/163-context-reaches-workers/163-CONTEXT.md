# Phase 163: Context Reaches Workers - Context

**Gathered:** 2026-07-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Everything the colony knows — the colonize/survey codebase map, phase research,
the approved charter, signals, memory — demonstrably arrives in worker prompts,
measured and inspectable by a person, without blowing the token budget. Plus the
worker write-contract fixes folded in from Phase 160's deferred list: the
permission contradiction, the unusable artifacts schema, and the `.aether/data/`
guardrail that workers currently evade.

**Executes ahead of roadmap order** (pulled forward 2026-07-28): Phase 161's
auto-routing was descoped by user decision, and this phase is what actually
serves the milestone's "usable daily on an inexpensive model" goal.

**Premise corrections this phase must plan against:** the requirement text
predates Phase 160. The playbook loader is deleted and pinned dead; `survey-load`
is deleted with a test asserting its absence (`TestSurveyLoadAbsentAndUncalled`);
`.aether/docs/command-playbooks/` is confirmed non-executing reference material.
Delivery happens on the modern manifest/brief path (D-04).

</domain>

<decisions>
## Implementation Decisions

### What "works well on cheap models" actually means (the phase's north star)
- **D-01:** Grounding, not brevity. In the user's words: cheap models worked when
  "agents are writing the right files and they're doing the right things" because
  of roadmaps and colonize — the codebase map and plan continuity. Restoring that
  grounding to worker prompts is the priority payload. It was never "about
  necessarily shortening anything."
- **D-02:** Compact delivery forms preferred. The user pointed at modern
  code-graph approaches ("people have graphs and stuff to map code bases to save
  context") — research should evaluate graph/summary representations for the
  survey map rather than full-text dumps. Note Aether already has idle graph
  machinery (`pkg/codegraph`, `pkg/graph`) and colonize already writes
  `.aether/data/survey/`.
- **D-03:** Measurement (CONTEXT-08) is a guard, not a gate. Instrument
  per-section prompt sizes so every addition is honest against a budget, but the
  phase is not a diet program and grounding restoration does not wait on a
  trimming pass.

### Worker write rules (fixes the guardrail evasion seen live 2026-07-28)
- **D-04:** Sanctioned scratch areas. Specific workflow dirs
  (`.aether/data/planning/`, `.aether/data/worker-debug/`, and peers the plan
  identifies) become declared-writable in the distributed rules. Colony state,
  session files, and the rest of `.aether/data/` stay protected. A rule the
  system's own workers must evade protects nothing.
- **D-05:** The two folded contract bugs ship with this: the permission profile
  that says "without repository writes" while the brief orders writing
  phase-plan.json (`pkg/codex/permission_profile.go:96`, same class
  `cmd/phase_research.go:124`), and the `artifacts` sub-schema that accepts only
  `{}` (`pkg/codex/worker.go:921-926`). Instructions, permissions, and schemas
  must agree.

### The brief inspector (CONTEXT-07 — the user's own verification window)
- **D-06:** Default output is a sectioned checklist: each context section (task,
  survey map, charter, research, signals, memory) with present/absent and size,
  plus total against budget. A `--full` flag prints the raw assembled prompt.
  Ten-second check normally; full text when chasing a problem.

### Requirement reconciliation
- **D-07:** Intent over letter. CONTEXT-01 and CONTEXT-04 are reworded to name
  outcomes (survey and build context demonstrably present in worker prompts via
  the manifest/brief path) rather than the deleted mechanisms
  (`codexBuildPlaybooks()` playbook loading, `survey-load`). The mechanism change
  and its reason are recorded in REQUIREMENTS.md. Definition of Done still
  binds: a command must fail when context stops arriving.

### Validation (reshapes CONTEXT-09)
- **D-08:** No staged benchmark. The user: "Testing should come later when I use
  aether to develop real repos I am using." The formal before/after experiment is
  descoped; phase verification relies on the brief inspector plus automated
  presence/invariant tests. Real-world validation happens through the user's own
  projects after the phase ships.

### Charter enforcement (CONTEXT-06)
- **D-09:** Rules plus gate check. Approved charter rules ("TDD required, ESLint
  enforced") arrive in worker briefs as hard rules, AND the continue
  verification checks the mechanically-enforceable ones — an ignored rule
  surfaces as a gate finding, not silence.

### Codebase map freshness
- **D-10:** Manual refresh with a loud staleness warning. Colonize runs when the
  user chooses; worker briefs and the brief inspector display "map is N
  commits stale" so nobody grounds on fiction unknowingly. No auto-refresh —
  consistent with the user's no-silent-automation stance (see the Phase 161
  descope).

### Suggestion surfacing (CONTEXT-05)
- **D-11:** End-of-build summary. `suggest-analyze` results collect during the
  build and present once at the end as a tick-to-approve list; approved signals
  take effect from the next build. No mid-build interruptions.

### Claude's Discretion
- Graph/summary format mechanics, budget numbers and thresholds, brief section
  ordering, staleness-warning wording, the exact scratch-dir list, where the
  writable-dirs declaration lives, gate-check mechanics for charter rules.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` §CONTEXT-01..09 — requirement text (CONTEXT-01/04/09 to be reworded per D-07/D-08)
- `.planning/ROADMAP.md` — Phase 163 section AND the Phase 161 descope note (explains why this phase executes next and what "cheap models" means)

### Live evidence (why the folded decisions exist)
- `.aether/docs/known-issues.md` — "Workers route around the .aether/data/ write guardrail (2026-07-28)": the field observation driving D-04
- `.planning/phases/160-fail-loudly/160-CONTEXT.md` §deferred — the two contract bugs with file:line cites (D-05)

### Code the phase lands in
- `cmd/colony_prime_context.go` — context assembly; zero charter references today (CONTEXT-06 target)
- `pkg/codex/worker.go:921-926` — the `{}`-only artifacts schema (D-05)
- `pkg/codex/permission_profile.go:96` and `cmd/phase_research.go:124` — permission/brief contradictions (D-05)
- `cmd/codex_build.go:894` — `codexBuildPlaybooks()`; STALE mechanism cited by old CONTEXT-01, do not resurrect (D-07)
- `pkg/codegraph/`, `pkg/graph/` — existing graph machinery relevant to D-02
- `.aether/data/survey/` — colonize output that must reach workers (CONTEXT-04 intent)
- `CLAUDE.md` §Token Budget — the stacking budgets CONTEXT-08 measures

### Constraints
- Phase 160's pinned deletions: `TestSurveyLoadAbsentAndUncalled` (survey-load stays dead), dead playbook corpus — deliver intent via manifest/brief path
- `.aether/ts-host/` edits require `npm --prefix .aether/ts-host run build` (dist is embedded)
- Distributed rules files are hub-published: `aether publish` after edits

</canonical_refs>

<specifics>
## Specific Ideas

- The user's grounding description, verbatim anchor for D-01: "the way that we're
  able to maintain context throughout things... agents are writing the right
  files and they're doing the right things... we had everything with roadmaps
  and... colonize."
- The worker's own evasion report, anchor for D-04: "The Write tool is
  guardrail-blocked on .aether/data/ paths. These planning artifacts were staged
  in the repository and relocated with a scripted move."

</specifics>

<deferred>
## Deferred Ideas

- **Staged cheap-model benchmark fixture** — rejected for this phase (D-08);
  could become a regression guard later if real-world usage exposes context
  regressions
- **Auto-refresh of the codebase map at phase start** — rejected (D-10); revisit
  only if staleness warnings prove insufficient in practice
- **Mid-build suggestion prompts** — rejected (D-11)
- **Phase 161 residue** (`colony/` distribution, zero-reader policy file
  reckoning) — remains parked per the roadmap descope note, not this phase

</deferred>

---

*Phase: 163-context-reaches-workers*
*Context gathered: 2026-07-29*
