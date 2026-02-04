# v5 Field Notes

**Source:** First real-world test of Aether on an external repo (2026-02-04)
**Status:** Raw notes from live testing — to be consumed by `/cds:new-milestone`

---

## Note 1: Multi-Ant Colonization
Colonize command should use multiple ants — each colonizes/reviews the codebase independently, then they compare notes. Not just a single colonizer.

## Note 2: Colonizer Missing Visual Output
Colonizer ant output has no emojis/visual indicators despite this being addressed in previous versions. Visual formatting is missing in practice for colonize specifically.

## Note 3 (Uncertain — Needs Thinking): Pheromone-First Flow
After colonizing, the system should prompt the user to emit pheromones (FOCUS/INIT) for what they want to work on, *before* planning. Current flow is colonize → plan, but maybe should be colonize → pheromone injection → plan. The plan should be driven by pheromone signals rather than being the immediate next step. Needs evaluation.

## Note 4: Visual Indicators Isolated to Colonize
Visual indicators (emojis, progress markers) are working well in build phases but missing specifically in the colonize command. Not system-wide — colonize was missed or lost its visual treatment.

## Note 5 (CRITICAL): Context Clear Prompting
Every command that completes a meaningful unit of work MUST prompt the user to clear context when it's safe. This is non-negotiable. The system must:
- (a) Verify all state is persisted to files before suggesting clear
- (b) Explicitly tell the user "it's safe to `/clear` now"
- (c) Guarantee it can pick up exactly where it left off with zero information loss
- (d) Make the user feel confident doing it, not anxious

This is how CDS works and Aether must match it. Context death is the core problem this system solves — if users are afraid to clear because they're not sure if state is saved, the whole value proposition falls apart. Proactive clearing before context auto-compacts is essential (see Note 13 where this actually happened).

## Note 6: Deployment/Distribution Model
The `.aether/` directory (utils, worker specs, data) doesn't exist in target repos, so commands break immediately. The commands reference `.aether/aether-utils.sh` and `.aether/workers/` which are only in the Aether source repo. Need to establish a clear deployment/distribution model:
- Option A: `.aether/` lives at a global location and commands reference it there
- Option B: `/ant:init` bootstraps the `.aether/` infrastructure into whatever repo you're in
- Option C: Template repo approach
Currently there's no documented or working way to use Aether on any other repo. Fundamental usability gap.

## Note 7 (Positive): System Works
This is the best the system has been so far. It works — ants are spawning, visuals are good during build phases, and it feels effective. Real progress. The issues in these notes are refinements, not fundamental problems.

## Note 8 (Uncertain — Needs Thinking): Auto-Spawned Reviewer/Debugger Ants
Should there be code reviewer ants and debugger ants that auto-spawn at appropriate stages? Like a watcher that reviews code after a builder finishes, or a debugger that kicks in when tests fail. These should happen automatically at the right points in the development lifecycle, not require the user to trigger them. At times that are appropriate and typical for effective coding and development practices.

## Note 9: Pheromone User Documentation
The pheromone system needs clear user-facing documentation — when do you use FOCUS vs REDIRECT vs FEEDBACK? What are the practical scenarios? How does this actually make the system better than just telling Claude what to do? The value proposition needs to be obvious: context survives across clears, ants build on each other's work through signals, the colony adapts. Right now a new user wouldn't know how or why to use pheromones. This is both a docs problem and potentially a UX problem — the system should guide users toward pheromone usage at the right moments.

## Note 10: Observations from Real Build Output
- Build output format is genuinely good — step checklist, colony activity with caste emojis, task results, pheromone sensitivity display, watcher report with severity levels, learnings, auto-emitted pheromones
- The system assigned two builders and a scout to one phase — real caste coordination
- **Merge conflict risk:** "grouping all package.json changes into a single worker avoids merge conflicts when 6 tasks target the same file" — task-to-worker assignment matters. Multiple builders editing the same file in parallel would conflict. Planner/Queen needs to group file-overlapping tasks to the same worker
- **Watcher correctly distinguished** pre-existing test hash drift from actual Phase 1 problems — verification working
- **Learnings are specific and useful** — "ESM package with conditional exports requires both types and default" is actionable, not boilerplate
- **Low-severity issues surfaced but didn't block** — good signal-to-noise ratio
- **Pheromone sensitivity display is informative** — architect ignoring everything at current decay levels is interesting, may need thought

## Note 11: Pheromone Recommendations to User
The system should recommend specific pheromone commands to the user at appropriate moments. E.g. after a build: "Recommended: `/ant:focus "error handling in auth module"`". This teaches new users how pheromones work by showing real examples in context, and makes the system more useful by surfacing suggestions derived from the colony's own work. The ants should guide the Queen, not just the other way around.

## Note 12 (Architecture Question — Needs Thinking): Learning Scope
Where does colony learning live? Currently `.aether/data/memory.json` is per-repo, which is correct for project-specific patterns. But there's probably a two-tier model needed:
- **Per-project learnings** (in `.aether/data/` within each repo) — project-specific patterns, codebase conventions, tech stack quirks. Default.
- **Global learnings** (in `~/.aether/` or similar) — cross-project meta-learnings. Things like "grouping same-file tasks to one worker avoids conflicts" are universally useful.

The blend is probably: ants learn per-project by default, occasionally a learning gets promoted to global when clearly project-agnostic. Promotion mechanism (automatic? user approval? architect ant synthesis?) is an open design question.

## Note 13: Context Collapse Incident (Phase 2)
- Context compaction happened mid-display during Phase 2 — phase completed but user didn't see full results before collapse. Directly reinforces Note 5.
- The learning "parallel workers writing to same file can cause last-write-wins conflicts" appeared again (also in Phase 1). Colony rediscovering same lessons — learnings not influencing future planning.
- Two builders assigned, one had to "reapply Phase 1 changes" — one builder overwrote the other's work. Same-file conflict problem is real and recurring. Phase Lead or Queen needs a rule: tasks touching the same file go to the same worker.
- Watcher flagged same pre-existing test hash drift as medium (was low in Phase 1). Severity creep on known issues — should the system track "known issues" separately?

## Note 14 (Needs Careful Design): Organizer/Archivist Ant
There should be an organizer/archivist ant that reviews the project for outdated files, stale documentation, dead code, orphaned configs — things that cause confusion. Hard problem: needs to understand full project scope so it doesn't delete important things. Needs:
- Conservative by default — archive rather than delete, flag rather than act
- Tiered approach: (a) confidently stale → clean up, (b) probably stale → recommend to user, (c) unclear → just flag
- Could trigger at phase boundaries, milestone completions, or during colonize flow
- Design question: new caste, specialist mode of existing caste, or multi-caste behavior?

---

*Notes collected: 2026-02-04 during first live test*
*To be consumed by: /cds:new-milestone for v5 planning*
