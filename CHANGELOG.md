# Changelog

All notable changes to the Aether Colony project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.85] - 2026-09-21

### Fixed

- **Planning no longer dead-ends when the map of the code is out of date.**
  The automatic map refresh told its survey helpers to write into a temporary
  folder that the write-protection rule did not allow, and the helpers' own
  instructions told them to refuse any folder but one. Both now agree with
  what the program orders, and a check feeds every path the refresh orders to
  the real rule so the two cannot drift apart again.
- **Archiving a finished project no longer fails when the project has both of
  its context notes.** Two different files were being saved into the archive
  under the same name. The project-level one now has its own name.
- **Running the test suite no longer writes a flag titled "test" into the real
  project data.** One smoke test was not pointed at its temporary folder.

## [1.0.84] - 2026-09-21

### Added

- **Marking a project finished now writes it into the project's changelog.**
  A dated entry with the project's goal and one line per phase is added to
  `CHANGELOG.md` in the project folder, newest first, and the file is created
  if there is none. A project closed before every phase was finished says so
  and marks the unfinished phases. Running the same finish twice adds nothing
  twice, and a changelog problem can never stop or undo the finish itself.

### Changed

- **The README was corrected against the current program.** Install
  instructions are honest about which routes are behind the source, the worked
  example shows one helper per build instead of four, the live view is
  described as one inline screen, commands and folders that no longer exist
  are gone, reliability figures with no evidence behind them were removed, and
  the history now reaches this release.

## [1.0.83] - 2026-09-21

All four fixes came from the owner's first morning of real use.

### Fixed

- **A finished-and-archived project no longer reads as damaged.** Filing a
  finished project in the archive left pieces of its old plan and spec in the
  saved record, so the update screen said "recover", resume printed a wall of
  "malformed" lines, and status said "no project". Archiving now clears them
  (the archive keeps the full verified copy), projects archived by 1.0.79 to
  1.0.82 are read correctly without being rewritten, and status, resume and
  update all say the same thing: your last project is finished and archived,
  start the next one.
- **Planning no longer refuses a project whose main page sits in a deep or
  long-named folder.** The internal name for such a page was built from its
  whole path against a 40-character limit, so any path deeper than about 22
  characters stopped the discussion step. Long paths now get a short stable
  name; short paths keep the names they already had.
- **Update is no longer locked out after Aether installs its own helper
  packages.** The program installs packages into the shared copy on your
  machine when it needs them, and those packages contain shortcut links the
  updater refused to pass. It now steps over package folders, which were never
  copied anyway, and still refuses a link anywhere else.
- **The closing advice on the status and resume screens comes from the one
  shared decision** every other screen uses, and that shared card no longer
  claims a saved goal or an unwritten handover note for an archived project.

### Changed

- The changelog now has an entry for every release from 1.0.64 to 1.0.82;
  eighteen were missing.

## [1.0.82] - 2026-09-20

### Added

- **Codex gets the nine `$ant-*` skills.** They were built in Phase 204.1 but had
  never been published; this is the first release that installs them.

### Changed

- **Codex builds use the proven direct route by default.** `$ant-build` runs
  `aether build <phase>` and the Go runtime starts the Codex workers itself. The
  unfinished native-helper bridge (Phase 204.2, parked 2026-09-20) stays in the
  tree and is reachable only with `AETHER_CODEX_NATIVE_BUILD=1`. A direct Codex
  build has no blocking team check-in; the runtime prints its own heads-up when a
  forced reviewer or an owner question is pending.

### Fixed

- Carries the 1.0.80/1.0.81 local repairs into the main line: reviewer result
  schema, dependency-gated waves, honest timeout figures, accurate rollback
  message, superseded blocked closeouts, and host-pinned worker platform.

## [1.0.80] and [1.0.81] - 2026-09-18

These were local hotfix builds made from a side branch and never formally
tagged on the main line before 1.0.82 absorbed them.

### Fixed

- **A helper now always runs on the platform it was actually given, with no
  quiet fallback.** Earlier, if the requested AI platform (Claude Code, Codex,
  OpenCode) could not be reached for a worker, the run could silently swap in
  a different one instead of stopping and saying so.
- **Two bookkeeping bugs that could leave a project's phase tracking wrong
  after a partial rebuild are fixed.** A finished check's verification result
  is now saved before a reviewer is sent out, and proof of work already done
  on earlier, unaffected tasks survives when only part of a phase is rebuilt.
  The most recent outcome recorded for a phase is now always the one that
  wins.
- **Five accuracy fixes reported from a real project in the field:** a
  reviewer's finished review is now accepted in the shape the program
  actually produces; work that depends on another task now waits for that
  task to be genuinely proven first rather than assuming it is done; a
  helper that times out without reporting back is now described honestly
  ("no result was reported") instead of showing a false zero; an automatic
  rollback message now names the actual files it kept; and a properly
  finished phase no longer hands the next phase a leftover "this was
  blocked" note that no longer applies.

## [1.0.79] - 2026-09-16

### Changed

- **Phase 203 — worker and runtime changes:** additional workers use shared
  admission limits and recorded handoffs. Worker requests, progress and results
  can be traced through the run, and guidance is adjusted using recorded outcomes.
- **Phase 204 — learning changes:** durable run records support checks before
  learned guidance is promoted, rollback when a change causes harm, and reports
  showing what helped and where the owner had to intervene.

### Fixed

- **Phase 205 — finishing and archiving:** archiving a finished colony handles
  missing handoff notes, and resuming refreshes those notes. Finish and archive
  messages reach the chat while machine-readable results remain valid. Completion
  checks reject claims that name tests which never ran.

## [1.0.78] - 2026-09-13

### Added

- **A helper that gets stuck can now ask the program for backup, and the
  program decides whether to grant it.** Every AI helper the colony (an
  Aether project) sends out is now told it can request another helper join
  it, but the request only succeeds if the program's own limits allow it —
  how many "ask for help" hops deep the chain has gone, how many helpers the
  run has already used, and whether the same job is already being worked.
  When you check a finished run, you now see a full family tree of who asked
  for backup, what it cost, and any request that was turned down.
- **The program's own steering notes now earn or lose trust based on whether
  they actually helped.** A short reminder the colony leaves for itself
  ("focus here", "never do this") gets strengthened the more it genuinely
  helps and weakened — or set aside — the more it doesn't, based on what
  really happened afterward, not just on whether a helper saw the note. A
  note you pinned yourself is never moved by this automatic tuning.

### Fixed

- **You can now retire a plan proposal that is sitting idle waiting for a
  decision**, instead of it being stuck with no way to withdraw it.
- **A screen that had quietly lost its plain-language status indicators is
  fixed** — the project status screen shows its per-line symbols again.

## [1.0.77] - 2026-09-13

### Fixed

- **A record of exactly which files a helper touched is now carried through
  correctly on every build path**, including the one used by the Claude Code
  menu commands, so finished work is never mistakenly treated as unproven.

## [1.0.76] - 2026-09-13

### Added

- **Backup-helper requests are now recorded durably before any decision is
  made about them**, and every detail of a request can be inspected with the
  `aether recruit` command.
- **Suggestions the program raises about itself, and files it wants to pull
  in from another project, now go through one shared approval line** instead
  of separate, inconsistent paths.

### Fixed

- **A security scan that used to stop at the first exposed secret in a file
  now reports every one it finds in that file**, not just the first.
- **A file a helper reported changing in its own notes is no longer silently
  dropped** if it did not also show up in a separate internal record — the
  program now trusts a helper's own claimed changes.
- **Writes to the approval queue (the shared list of pending suggestions and
  imports) are now all-or-nothing**, so a crash mid-write can no longer leave
  it in a half-written state.

## [1.0.75] - 2026-09-12

### Added

- **The first piece of the "ask for backup" system landed**: a helper's
  request for another helper to join it is now traced end to end and
  governed by one real gate in the program, rather than being an idea with
  no enforcement.
- **A steering note pulled in from another project is now quarantined and
  labelled with where it came from**, rather than being treated the same as
  a note this project generated itself.

### Fixed

- **Work already proven correct in an earlier attempt at a task is now
  properly credited**, instead of potentially being asked for again.

## [1.0.74] - 2026-09-12

### Fixed

- **If a project was left "paused" in a way that blocked you from resuming
  it, the program now tells you exactly how to get unstuck**, instead of
  leaving you at a dead end.

## [1.0.73] - 2026-09-12

### Added

- **Most of the remaining everyday screens got their plain-symbol treatment**
  (see 1.0.72 below for what this means): the project-status screen, the
  clarifying-questions screen, the specification screen, the finishing
  ("seal") screen, the phase-check screen, and the build screen all now show
  a small symbol at the start of each line explaining what kind of
  information it is, in plain English.
- **A safety rule about who is allowed to review higher-risk work (money,
  passwords, deleting data, etc.) is now checked at the right moment** — when
  the phase is actually being checked, not earlier when it is only being
  planned — closing a case where two internal rules disagreed about when
  that review happens.

### Fixed

- **A stray internal signal could pause a whole project from inside a
  helper's own task; that is fixed** so a helper working on its assignment
  can no longer accidentally halt the run it is part of.
- **A rare bug where machine-readable output leaked a plain-text symbol into
  JSON data is fixed.**

## [1.0.72] - 2026-09-12

### Added

- **The first screens got a return of the readable, symbol-per-line look**
  this project used earlier in its life: each line on the "what to do next"
  card now opens with a small icon showing what kind of information it is
  (a warning, a finished task, a suggestion, and so on), in plain English,
  building toward every everyday screen matching by 1.0.73.

### Fixed

- **You can now tell the program which of two waiting plans to review by
  name**, instead of it being ambiguous which one you meant.
- **The program no longer drafts a plan proposal that its own acceptance
  check is guaranteed to refuse** — that mismatch is caught earlier instead
  of producing wasted work.
- **Advice offered after a failed automatic bug repair now goes through the
  same shared decision path as everything else**, instead of a separate,
  inconsistent one.

## [1.0.71] - 2026-09-12

### Fixed

- **Planning helpers are now told to return what the program actually
  accepts.** A research or plan-writing helper that followed its own
  instructions had its result rejected, because the instructions described an
  older result shape and asked it to write files it was not allowed to write.

## [1.0.70] - 2026-09-12

### Fixed

- **An interrupted plan that had reached its second research round can be
  picked back up.** Rerunning planning used to start a brand-new run and
  abandon the finished first round; it now continues the run that was
  genuinely waiting.

## [1.0.69] - 2026-09-12

### Fixed

- **A planning run that stopped partway can be found again even when its
  bookmark file is missing.** The program now falls back to the run's own
  saved records, and refuses to guess when more than one run could match.

## [1.0.68] - 2026-09-12

### Fixed

- **A project marked finished by an older version no longer dead-ends when
  you try to archive it.** The refusal now names the way out: mark it finished
  again under the current version, then archive.

## [1.0.67] - 2026-09-12

This was the largest release in this run — 1,082 commits, covering several
phases of work.

### Added

- **A live, truthful view of what the colony is doing right now
  (`aether watch`).** It shows exactly one of three honest screens: a live
  dashboard while something is running, a replay of the most recent run once
  it finishes, or an honest "nothing has run yet" card — never an invented
  status.
- **Chasing down a bug now sends four different kinds of investigation at
  once** (`/ant-swarm`): one traces the bug through the project's history,
  one searches the current code for the same pattern elsewhere, one traces
  the actual error, and one researches outside sources. You get one card
  showing where they agree and disagree and which fix ranks highest. The
  winning fix is then applied automatically with a safety net — a save
  point beforehand, a check that it worked, and an automatic undo if it
  didn't — and if the same bug survives three attempts, the program writes
  up a plain-language case for you instead of trying again blindly.
- **Deep research now works in rounds and leads with the answer**
  (`/ant-oracle`): it asks one round of clarifying questions, then researches
  on its own, and its final answer leads with the actionable recommendation
  and an honest confidence level — sources are listed further down, not
  first.
- **Every "wrap up and close out" command now ends with the same clear
  summary card** — pausing, resuming, sealing (marking a project finished),
  and checking status all render the same, consistent closing message
  instead of each writing its own.
- **Opening a session now greets you with what the colony remembers about
  you** — your saved preferences, its strongest learned habits, and a note
  on what the last helper left for the next one.
- **A helper's own lesson learned on the job now flows automatically into a
  reusable project habit**, and once a lesson is strong enough it is shared
  with your other projects at the end of every check, not only when a
  project is formally closed.
- **Setup, resuming, and status tracking were rebuilt so project state can't
  be left half-written** if a command is interrupted partway through, and
  the guided setup command became the standard front door for starting a
  new project.
- **Overnight/unattended runs (`/ant-run`) got firmer guardrails** —
  clearer rules for when to pause versus continue automatically, and
  more reliable resuming of research and planning that was interrupted
  partway through.

## [1.0.66] - 2026-08-29

### Added

- **Build and check screens now stream a live line the moment each
  verification step starts and finishes**, naming exactly which check ran
  and what it found, instead of only showing a result once everything is
  done.
- **Finishing a project ("sealing" it) now requires an explicit yes from
  you before it's marked complete**, closing a gap where a seal could be
  recorded without a clear owner confirmation.
- **The `aether pause` command is now the standard name** for stopping work
  at a safe point (the older, longer name still works as an alias), and
  running `aether update` now restores that alias automatically if it ever
  goes missing.

### Fixed

- Corrected the wording of a finished check's plain-English result summary
  so it no longer showed raw internal terms.
- Fixed a case where the pause screen offered the same next-step command
  twice.

## [1.0.65] - 2026-08-28

### Added

- **Every build and check now ends with one honest line stating what it
  cost.** The team card also shows which AI model each helper used and why —
  routine, low-risk helpers are now automatically pinned to a cheaper model
  with the reason written down.
- **Cost tracking now uses the actual usage the AI platform reports**,
  replacing an old rough guess based on counting characters in the
  conversation.
- **A new read-only view lets you see spending broken down by helper**, not
  just as one combined total.

### Fixed

- Several accuracy bugs in the new cost-tracking system were caught and
  fixed in this same release: a retried task was overwriting a phase's
  recorded cost instead of adding to it, one measurement could get credited
  to two helpers at once, and a helper session the program couldn't read
  was being reported as costing nothing instead of being marked "unknown."

## [1.0.64] - 2026-08-27

### Added

- **A second AI reviewer is now only sent for one of five specific,
  named risky situations** — passwords/logins, payments, releasing a
  version, deleting data, or changing the database's structure — and only
  you, never the automatic Queen or autopilot, can decide to skip that
  reviewer. A simple one-task bug fix now gets just the one helper writing
  the code, instead of a full review team every time.
- **Several related tasks can now be handled by one helper as a single,
  coherent job** when they are genuinely connected (for example, several
  tasks that all touch the same file), rather than always being split one
  helper per task. The program checks first that grouping them this way
  cannot make anything run in the wrong order.
- **If a helper only finishes part of a grouped job, exactly the finished
  tasks are credited and only the unfinished ones are retried** — nothing
  already proven working gets redone, and a helper's own claim of what it
  finished is checked against the real files in the project before any
  credit is given.

### Fixed

- Closed a set of gaps in the new team-review and grouped-job systems found
  during their own follow-up review, including cases where a forced
  reviewer's decision could be lost on a retry and where a trimmed team's
  stated reasons could be dropped.

## [1.0.63] - 2026-08-21

### Added

- **An honest "nothing needed changing" is now a first-class success.** A
  worker that proves the required behavior already exists reports
  `completed_no_change` (disposition `verified_existing`) instead of being
  forced to fabricate an edit or getting coerced to `failed`. Evidence is
  mandatory — a summary, a passing handoff verification, and the commands
  actually run — enforced at result merge
  (`TestNoChangeResultWithEvidenceIsAccepted` /
  `TestNoChangeResultWithoutEvidenceIsRejected`), build and continue
  provenance (`TestProvenanceAcceptsEvidencedNoChangeBuild`,
  `TestContinueProvenanceAcceptsNoChangeDispatch`), covered-task credit,
  task completion, and the in-process claims path
  (`TestClaimsNormalizationKeepsHonestStatuses`). Rate-limit/quota stops
  are now `interrupted` — terminal but resumable, never a code failure
  (`TestInterruptedIsTerminalButNotSuccess`). The worker contract teaches
  both outcomes on every lane, Go and TypeScript host alike
  (`TestResponseContractOffersNoChangeOutcome`); the completion-packet
  schema is regenerated in lockstep.
- **Team check-in before every build.** After the Queen shows her spawn plan,
  the build pauses and asks the owner to proceed, trim optional workers, or
  redirect — with a one-line reason per worker and safety workers marked as
  fixed (the runtime re-adds required castes regardless, so the card never
  offers a removal it would silently undo). Skippable with
  `aether build --no-checkin`; autopilot is unaffected. Rendered by the new
  `aether ceremony team-checkin` (locked read-only by
  `TestTeamCheckinDoesNotMutate`; card contents by
  `TestTeamCheckinCardShowsReasonAndRequiredMarking`).
- **Workers' open questions now reach the owner, not another agent.** Every
  worker handoff already carried `open_decisions`; they were delivered only
  into the next worker's prompt. New `aether handoff-decisions` lists the
  unanswered ones and `aether decision-answer` records the owner's ruling as
  a resolved clarification — which the context assembler already injects
  into every later worker prompt as CLARIFIED INTENT. Build wrappers ask at
  wave boundaries and the post-build checkpoint; continue asks before its
  steering checkpoint. An unanswered question never blocks a build. End to
  end relay locked by `TestResolvedOpenDecisionReachesNextWorkerPrompt`;
  workers are told to route judgement calls there instead of guessing
  (`TestResponseContractTellsWorkersToRouteJudgementCalls`).

### Fixed

- **The honest "nothing needed changing" result now actually works
  everywhere.** The `completed_no_change` and `interrupted` statuses were
  taught to the build path and left unknown to every other place that judges
  a worker's status, so a worker that followed the new contract truthfully
  was punished for it: the continue review gate blocked phase advance on it
  (`TestContinueWatcherPassesOnAnHonestNoChange`), the finalizer routed it
  into recovery and asked for the work to be redone, its wave counted it as
  a failure, the seal gate treated a required reviewer as absent, closeout
  dropped it from the tally, the status dashboard showed it as neither
  running nor finished, and finalization reported it as an unexpected
  status. Every one of those decision points now reads the vocabulary from
  one place, asserted as an invariant across all of them
  (`TestNoChangeSuccessIsHonouredWhereverStatusIsJudged`,
  `TestFinishedWorkerStatusesAreRecognisedNotDropped`,
  `TestInterruptedIsTerminalButNeverCountedAsSuccess`), with a ratchet that
  fails when a new hand-rolled success list appears
  (`TestNoHandRolledWorkerSuccessLists`). The same three fixes landed on the
  TypeScript host lane, which had its own copies. The 55 agent definitions
  across Claude, OpenCode and Codex that still listed only
  `completed | failed | blocked` now name the no-change outcome too.
- **The no-change evidence rule was enforced on one lane only.** A
  `completed_no_change` claim buys an exemption from the "show me the files
  you changed" requirement, and that exemption was granted on the in-process
  dispatch lane without checking anything — reopening the phantom-build
  loophole for any worker that simply said the words. Both lanes now apply
  one shared rule — a summary, a passing handoff verification, and the
  commands actually run — and the gate is asserted at its call site, not
  just as a function that exists
  (`TestRuntimeLaneDemandsNoChangeEvidence`,
  `TestRuntimeNoChangeEvidenceGateIsWiredIntoDispatch`).
- **The worker contract no longer promises a resume that does not exist.**
  Workers stopped by a quota limit were told "the colony will resume the
  unfinished slice"; nothing resumes a worker. The contract now describes
  what actually happens — the handoff is kept and passed to whoever picks
  the work up next, and the phase is restarted by the operator.
- Restored `gofmt` compliance to three Go sources committed unformatted.
- **A failed worker now has to say why.** A real build halted on
  "Keen-6=failed" with an empty reason, an empty summary, no blockers, and a
  worker report still saying "spawned" — nothing anywhere recorded what the
  worker had actually reported. Any status outside the recognised list was
  overwritten with a bare "failed" and the word the worker used was thrown
  away. The coercion stays (an unknown answer cannot be trusted as success),
  but what it displaced is now written down and shown
  (`TestUnrecognizedWorkerStatusSaysWhatItWas`, `TestMissingWorkerStatusSaysSo`).
- **Configuring one check no longer switches off the others.** Adding a
  "## Verification Commands" section naming a build command made the test
  command the same file had been supplying all along disappear, because the
  scan narrowed to the section and discarded everything outside it. A section
  may now add precision, never remove a command the file already provided
  (`TestAddingAVerificationSectionDoesNotUnresolveOtherCommands`).
- **A blocked build no longer points at a place you are not allowed to
  write.** The halt guidance offered `.aether/data/codebase.md` as somewhere
  to configure a command; that path is guarded and the write is refused,
  leaving no exit at all. It now names only the files anyone can edit
  (`TestBlockedVerificationGuidanceOnlyNamesWritablePlaces`).
- **Tasks folded into another worker are now named.** When dependent tasks are
  merged into one worker the plan said "(+2 more steps)" without saying which
  tasks those were, so a nine-task phase showing six workers looked like three
  tasks had been dropped — and a real colony hand-reconciled work that already
  had a claimant. The plan now names them and states the arithmetic
  (`TestSpawnPlanNamesTheTasksAMergedWorkerCovers`).
- **The build summary no longer claims a repair it did not make.** A column
  headed "Recovered" counted the recovery actions the system *decided on*, not
  workers that actually recovered — so a real build reported "1 recovered" for
  a worker that was still failed when it halted seconds later. The column now
  reads "Recovery Planned" and says what it counts
  (`TestWaveSummaryDoesNotClaimARepairItDidNotMake`). The underlying gap is
  unchanged and deliberate: deciding on a retry and carrying one out are still
  two different things, and nothing in this lane re-dispatches.
- **The four surveyors no longer look identical.** All four collapsed to one
  glyph and the word "Surveyor", so a codebase survey printed four
  indistinguishable lines; each now says what it surveys
  (`TestEachSurveyorSaysWhatItSurveys`).
- **A light codebase survey wrote nothing at all.** Choosing the cheaper
  survey correctly sent two surveyors instead of four, then the survey
  refused to save because four documents it had deliberately not asked for
  were missing — so `/ant-colonize` aborted and produced no artifacts. The
  requirement is now what this survey's own team promised, while a surveyor
  that was sent still owes every one of its files, and a survey with no
  surveyors is still rejected (`TestLightColonizeStillWritesItsSurvey`).
- **A worker crediting another worker's already-done task got the wrong
  stamp.** When one worker verified tasks belonging to several dispatches,
  the credited dispatches were recorded as ordinary completions with no
  files — and the next step then halted them as suspected phantom builds.
  They now inherit the claimant's actual outcome.
- **Manually reconciled work no longer reads as a phantom build.** Counting
  it as a success meant it was also asked for file outputs it cannot have by
  definition, which halted the manual recovery path.
- **The word "document" belonged to no worker.** A note said it had been
  handed from the knowledge-keeper to the documentation writer, but the
  documentation writer's keyword was "documentation", which never matches the
  bare word — so a phase asking to document something summoned neither
  (`TestChroniclerOwnsTheWordDocument`).
- **Five useless-spawn leaks closed.** Sealing a colony no longer requires a
  test-coverage Probe when the final phase produced no testable code
  (`TestSealProbeRequiresTestableCode`); a bug swarm no longer always
  summons a researcher and a git-history digger — they now ride on keyword
  relevance like every other specialist
  (`TestSwarmTrivialBugSkipsHistoryAndResearch`); a caste with no build
  dispatch path can no longer be selected onto a build team, where it
  consumed a budget slot and silently displaced a real specialist
  (`TestEveryBuildSelectableCasteCanDispatch`); the everyday continue path
  now honours the Queen's `--castes` proposal instead of only the heavy
  plan-only path (`TestContinueFastPathHonoursCasteProposal`), and the
  Keeper no longer arrives on incidental words like "standard" or
  "document" (`TestContinueDoesNotSummonKeeperOnIncidentalWords`); and
  `--verification-depth light` finally means something for colonize — two
  surveyors instead of a fixed four
  (`TestColonizeLightDepthTrimsSurveyors`).

## [1.0.58] - 2026-08-17

### Fixed

- **Publish no longer leaks private session files into the hub.** The Aether
  repo's own `.aether/` folder does double duty — shipped source plus the
  colony working data from developing Aether on itself. The publish
  exclusion list matched directory names only, so loose working files
  (session handoff snapshots, activity ledgers, failure logs, review
  archives, this repo's own colony memory) were swept into
  `~/.aether/system/` alongside the real product. Exclusion now covers
  exact file paths too, and is two-sided: the next publish also removes
  copies that leaked under earlier versions. Locked by
  `TestHubPublishExcludesPrivateColonyFiles`, which pins the exclusion
  floor so an entry cannot be silently dropped. The machine-local
  `registry.json` (personal repo paths) is also untracked from git and
  ignored going forward.

## [1.0.57] - 2026-08-17

Colony Conversation & Judgement: the colony becomes something you talk WITH,
not just watch — every proposal is asked, every block brings a way forward,
and three field-reported breakages are fixed.

### Added

- **`/ant-ask` — ask the colony anything.** "Where are we?", "why did phase 3
  block?", "what changed?" answered from the colony's own memory with zero
  setup. The briefing assembler gained its first parameterization: a question
  boosts the sections it is about and pulls in recent activity, read-only by
  locked test.
- **Queen-composed clarification questions.** The discuss flow composes 3–5
  questions from THIS goal and THIS codebase instead of the same canned trio;
  every composed question must cite what it is grounded in or the runtime
  refuses it. Answers land in the unchanged pipeline (hard constraints still
  become REDIRECT signals) and the canned generator remains the typed
  fallback.
- **A git save-point after every verified phase.** Exactly the files the
  phase's workers reported changing — the owner's dirty, untracked, and even
  pre-staged files can never be swept in — with a greppable
  `aether(phase-N):` subject and an `Aether-Phase` trailer for the
  Archaeologist. Never pushes (argv invariant test); a commit failure pauses
  autopilot via the previously-dead marker instead of blocking. Off switch:
  `aether phase-commits set off`.
- **Worker model tags on spawn lines.** `🔨🐜 Builder [sonnet] Mason-67`,
  resolved from agent frontmatter plus any `ANTHROPIC_DEFAULT_*_MODEL`
  redirect. Display only — routing stays with the platform, and a parity test
  pins the display table to the agent files so it cannot go stale.
- **Ranked post-init proposals.** Init proposes the sensible next moves
  computed from the actual repo (colonize first for existing code, discuss
  first for broad goals) with plain-English reasons; the wrapper asks, the
  recorded suggestion is the real top proposal instead of a hardcoded
  constant.
- **The classic end-of-phase footer.** Every phase end shows open flags 🚩
  with triage counts, active steering signals with content and strength,
  phase/task progress bars, an honestly-verified "safe to clear your context"
  line, and the next command as the user's choice via a real question — the
  build wrapper never rolls into verification on its own.
- **Deliberately-RED phases.** A typed `expect_failing_tests` field lets the
  route-setter plan TDD red-first phases whose deliverable IS a failing test
  run; verification inverts the tests check (green blocks, red advances), and
  timeouts are never credited as the expected failure.
- **Force seal (owner override).** `aether seal --force --reason "why"` files
  a project away past unverified phases, open blockers, and review blocks —
  for work finished outside the colony or a colony wedged on its own gates.
  Never silent: the reason is required, a `sealed_forced` event names every
  unverified phase, CROWNED-ANTHILL.md carries a permanent Owner Override
  section, and the wrapper asks before ever forcing.

### Fixed

- **Spawn budget counted all-time history** (field report): the append-only
  spawn ledger made any repo that ever finished more than 20 helpers
  permanently unable to spawn again, with no recovery command able to clear
  it. Fallback counting is live-helpers-only, and the deny message names the
  actual cause instead of claiming "no run is recorded" when one exists.
- **Review castes were briefed to run a CLI they cannot run** (field report):
  auditor and gatekeeper have no Bash by design, yet their briefs instructed
  `aether review-ledger-write` — they self-reported blocked and stalled the
  phase. The runtime now persists the findings workers return, in-process.
- **The flags gate silently lied**: it claimed to run "every time for safety"
  but never opened the flags file, so a blocker raised with `/ant-flag`
  blocked nothing. The classic Iron Law is restored, advancement-scoped:
  blockers stop `continue` (never `build`), cannot be acknowledged away, and
  machine-raised blockers clear on green verification evidence — while
  chaos-raised and owner-raised blockers never auto-clear. The age-based
  auto-resolve no longer defaults to resolving problems by growing old.
- **Critics must bring solutions**: structured review findings now actually
  feed the blocking decision (they were decorative); a blocking finding
  carries its fix in the same breath or names `/ant-unblock`, CRITICAL ledger
  writes without a suggestion are refused, Watcher issues carry
  suggestion+blocking, every Chaos finding carries a concrete
  `suggested_hardening`, and blocked output gains a Way Forward section.
- **The classic flag renderer was dead code**: written during the restoration
  round, never wired — `/ant-flags` now actually uses it.

## [1.0.56] - 2026-08-16

The v5.4.0 richness restoration: the colony you can watch work, back on the
modern runtime. Almost everything here is reconnection — machinery that
existed, computed, and rendered for nobody.

### Added

- **The classic caste identity is back.** Every worker renders glyph-plus-ant
  (`🔨🐜 Builder Mason-67`), the v5.4.0 house style, locked by test. Continue
  worker lines carry full identity instead of a bare `[caste]` tag, and the
  moment of dispatch announces itself again (`──── 🔨🐜 Spawning 3 Builders in
  parallel ────`).
- **The autopilot narrates the whole run.** `/ant-run` streams an AUTOPILOT
  ENGAGED banner, a header per phase, live worker lines, a PHASE ADVANCEMENT
  block with a momentum ticker between phases, a framed pause block with the
  reason and next step, and two celebrations at the end — including the classic
  `🎉 P R O J E C T   C O M P L E T E` with "The colony rests. Well done!".
  The classic pause engine is wired into the real loop (it had been connected
  to a code path the run never called), the replan checkpoint is back on its
  classic every-2-phases default, `--headless` queues the pause as a reviewable
  decision, and `--dry-run` previews every phase plus the full pause-trigger
  list. Proven end to end by a test that completes a multi-phase colony —
  the previous evidence completed zero phases.
- **The great ceremonies are back.** `aether init` shows the approved charter
  and closes with the colony-born banner (👑 intention, 🟢 READY, 🧠 hive
  wisdom seeded); the final `aether continue` celebrates project completion;
  `aether seal` draws the CROWNED ANTHILL and speaks the closing incantation.
- **Sectioned displays everywhere.** The pheromone view returns to its classic
  form — emoji headings that explain themselves (`🎯 FOCUS (Pay attention
  here)`), `[85%]` strengths, nested age/decay detail, a plain-English decay
  footer — and the same house style now covers flags, blockers, gate and
  failure classifications, memory health, review findings, and the loop-safety
  feed. An invariant test fails on the next machine-table anywhere.
- **Status explains its health score.** The five component signals (build
  velocity, error rate, signal health, memory, colony age) render beneath the
  health line, with real values where placeholders used to sit.
- **History reads like the classic activity feed** — `[time] ⚡ worker_spawned`
  with per-action icons, and continue shows what each worker actually found as
  nested detail with honest overflow counts.
- **Init does its cross-colony bookkeeping again.** A new colony registers
  itself in the machine-level registry with detected domain tags and seeds
  QUEEN.md from hive wisdom, both announced in the birth ceremony. Sealing
  marks the entry inactive. Medic's `--fix` creates a named rollback
  checkpoint and prints the undo command.

### Fixed

- `autofix-rollback` restored the checkpoint *envelope* over the colony state
  file — corrupting exactly what it promised to restore. Never caught because
  nothing called it. Fixed and wired into the medic flow.
- Autopilot's dry-run steps section silently vanished in visual mode (a type
  mismatch between the in-process and JSON-round-trip result shapes).
- The ceremony-level taxonomy (`worker_theatre`/`guided_ritual`/`dashboard`/
  `progress`/`quiet`), written and tested with zero callers, now gates
  streaming: plumbing commands stay quiet, lifecycle commands narrate.
- The clear-context advice only claims "Handoff saved" when the handoff file
  actually exists on disk.
- Three redundant swarm state mutators (`swarm-findings-init/add`,
  `swarm-solution-set`) retired — the swarm run has recorded its own findings
  in-process for some time; `swarm-findings-read` and `swarm-cleanup` are now
  documented as the inspection and housekeeping paths. Orphan allowlist
  291 → 279, with dated dispositions for the XML archival lane and the
  zero-reader policy files recorded in `.planning/decisions/`.

## [1.0.55] - 2026-08-16

### Added

- The Oracle setup ritual is now real runtime, not wrapper prose: `oracle propose` suggests scope/depth/accuracy and how much clarifying to do, `oracle brief` records the approved core question, and `oracle --from-brief` refuses to run without one. The approved question opens the research.
- Live research visibility: every Oracle round appends to a progress log regardless of output mode, and `oracle status --follow` (or `--background --follow`) streams one line per round — phase, round N of M, confidence against target, current question. The previous per-round display was a guaranteed no-op on every background run.
- `aether oracle selftest` proves the research machinery end to end with one real round in a throwaway workspace; non-zero exit when the dispatcher, agent definition, artifacts, or progress log are broken.
- Completed research is saved durably to `.aether/research/<date>-<slug>.md` with front matter (core question, confidence, rounds); `oracle save` keeps a stopped run, `aether research` lists what's saved. The workspace copy is swept by the next run; the saved copy is not.
- Research handoff: `aether init --research <path>` and `aether plan --research <path>` record a pointer on colony state, and the runtime carries the document's contents into planning, phase-research, and build worker briefs (budget-bounded, framed as evidence not instructions). `plan --print-brief` confirms delivery before a plan run is paid for.
- Orchestration visibility: the Dispatch stage renders the Queen's team choice from the manifest rationale — one clause per selected caste, a short clause for castes considered and not called, and a named line whenever the runtime keeps a safety caste against the depth flag. Completed workers say whether they flagged anything or came back clean. `aether status` shows the colony health line the vital-signs computation always produced.
- Knowledge-repo support: a directory of notes with zero code (Obsidian/Logseq vault, docs archive) now scans as a knowledge base — no CI/LICENSE/README housekeeping suggestions, risks about content loss and broken links instead of regression, and a charter describing note counts. `.aether/` gains a WHAT-IS-THIS.md marker so disk cleanups can tell durable colony state from a build cache.

### Fixed

- A new colony no longer inherits the previous colony's clarified decisions, assumptions, or worker handoff notes — re-init clears them, so workers stop being briefed with a finished project's context.
- Deep and exhaustive Oracle runs no longer spend their opening third at the lowest reasoning effort; the survey phase is capped at a quarter of the round budget and runs at medium effort on deep runs.
- Oracle research questions no longer splice in the goal of an inactive, unrelated colony, and the colony-goal question stays under 240 characters.
- `aether oracle status` is read-only; repairing a dead controller moved to `aether oracle recover`.
- "A unknown project" and the pile of "No X detected" filler lines are gone from generated charters; an empty scan now says plainly that it found nothing.
- The Claude and OpenCode Oracle agent definitions described a Stop-hook loop and `--legacy` flag that never existed; they now describe the controller-owned loop the runtime actually runs, fenced by tests that fail on invented paths.
- The session-freshness tools (`session-verify-fresh`, `session-clear`) — documented for years, called by nothing — are now reachable from the medic wrapper and off the orphan allowlist.

## [1.0.48] - 2026-08-04

### Fixed

- Review workers that emit their results before the final turn are no longer lost, so `/ant-continue` advances at standard depth instead of blocking.
- Next-step hints name the command you actually type (`/ant-continue` in Claude Code and OpenCode, raw CLI in Codex).
- Recovery hints name commands that exist; a new audit checks every runtime-emitted hint against the real command tree.
- Archive extraction is contained to its staging directory (path-traversal entries are rejected).
- Caste keywords match on word boundaries, so an Ambassador is no longer dispatched to local-only phases.
- Planners no longer bind "unchanged file" criteria to artifact claims, which used to block phases with no easy recovery.
- 41 stale flat command mirrors regenerated; the guard now covers all 61 wrappers.

## [1.0.40] - 2026-05-19

### Added
- Unified 5 divergent skip lists into canonical ScanFilter with 36-entry noise map (Python, Node, Go, Rust, build artifacts, caches)
- Source anchor extraction from clean survey output (top 50 repo-owned files) wired through to planner context
- Plan-grounding validation gate with soft warnings when generic plans meet available anchors
- Decision conflict detection with 9 contradiction pairs and FEEDBACK pheromone emission
- PlaybookLoader TS module restoring ceremony — loads 5 build + 2 plan playbooks via resolution chain
- YAML slimmed to packaging-only (orchestration removed from build.yaml and plan.yaml)
- M4L regression test proving end-to-end survey-to-grounding pipeline
- Execution path audit test verifying one conductor per workflow per platform
- Codex command-guide smoke tests (build, plan) verifying output without playbook loading
- Cross-platform parity tests (8/8 passing)
- 84 new tests across 4 phases

### Changed
- Bumped version to 1.0.40

## [1.0.39] - 2026-05-17

### Added
- Restored universal classic ceremony parity across Codex lifecycle flows, including spawn plans, wave starts, worker completions, and closeouts.
- Added final seal evidence for the Universal Classic Ceremony Parity and Command UX Completion colony.

### Changed
- Bumped the Aether source and npm package version to 1.0.39 for hub-backed testing in other repositories.
- Aligned Codex and Claude project guides plus README version references with the release version.

### Fixed
- Hardened Porter full-release readiness with persisted receipts, version agreement, binary smoke, Go/TypeScript/npm gates, and redacted failed-command diagnostics.
- Strengthened final seal review so blocker-level security and quality findings stop sealing while non-blocking release lessons are preserved.

## [1.0.38] - 2026-05-16

### Docs
- AGENTS.md: add Runtime Lifecycle mermaid diagram (under Architecture Overview) and Wisdom Pipeline mermaid diagram (under Wisdom Pipeline). Model-agnostic, shows both `in-repo` and `worktree` parallel modes, and contains the `skill-match` sub-graph.
- Added Phase 5 lifecycle integration guidance covering Orchestrator Mode routing, finalizer validation, wrapper parity, and changelog planning.
- Added final release-readiness handoff notes covering smoke commands, provider/auth redaction evidence, seal-time blockers, explicit residual risks for tag-pinned Actions and absent `govulncheck`, and the publish-readiness handoff.
- Documented runtime-owned `orchestrator_boundary_guidance` routing across command guides, Claude/OpenCode wrappers, Codex build-cycle skill, and wrapper-runtime contract docs.
- Clarified Phase 3 provider/auth hardening docs: provider availability preflight is separate from post-launch provider/API/auth worker failures, generated context must use sanitized provider/cause/next-action wording only, and ignored `.opencode/package*.json` files are local install artifacts rather than release-surface manifests.
- Updated the publish/update runbook with actionable publish-warning commands plus release metadata and auth-gate details for GitHub `GITHUB_TOKEN` and npm `NPM_TOKEN` paths.

### Changed
- Lifecycle wrapper guidance now stops before worker spawning when Orchestrator Mode routes to `aether discuss`, then requires rerunning `after_discuss_next` with a fresh plan-only manifest after answers are resolved.
- `aether publish` warnings now include exact recovery and verification commands for hub-version changes, skipped TS host publish steps, and stable/dev binary co-location.
- Release and CI workflows now run TS-host install/typecheck/test/build gates and avoid direct secret references in release job conditionals.
- Worker subprocess diagnostics now redact provider secrets before surfacing through errors, debug artifacts, RawOutput reports, or TypeScript host failure summaries.

## [1.0.30] - 2026-05-06

Restore the live host-worker ceremony that sits on top of runtime command visuals.

### Changed
- Build, plan, colonize, continue, seal, and swarm wrappers now require visible live Task/subagent panels with caste-labelled descriptions instead of background-only dispatch summaries.
- Build wrappers now render the forced-color runtime spawn ceremony before platform worker dispatch, while still using JSON plan/finalizer contracts for state.
- The build-wave playbook and Codex command guide now explicitly preserve platform-native agent color/icon metadata and live stacked worker panels.

### Added
- Added wrapper contract tests that fail if host-orchestrated lifecycle commands lose the live worker ceremony or fall back to markdown-only worker tables.

## [1.0.29] - 2026-05-06

Restore visual command ceremony while keeping runtime-owned behavior.

### Added
- Added runtime closeout rendering for lifecycle wrapper finalizers so `build`, `plan`, `colonize`, `continue`, `seal`, and `swarm` keep JSON completion contracts and still show visual completion ceremony.
- Added visual renderers for additional runtime-owned command families including reference, queen, shelf, signal exchange, tunnels, medic, and porter surfaces.

### Changed
- Simplified Claude and OpenCode wrappers to delegate duplicated behavior back to the Go runtime while forcing visual mode for user-facing command output.
- Updated wrapper/source contract tests to catch missing visual mode, missing lifecycle closeouts, and generated wrapper drift.

## [1.0.28] - 2026-05-05

Stabilize hosted-agent planning, Oracle runtime, and hub-backed platform updates.

### Fixed
- `aether plan --refresh` now returns an agent-delegate manifest inside hosted Claude Code and OpenCode agent sessions instead of spawning nested planning workers.
- `aether oracle` now auto-detaches from hosted Claude/OpenCode agent sessions and no longer falls back to fake Oracle worker completion when no real dispatcher is available.
- `aether update --force` now refreshes global Claude, OpenCode, and Codex platform assets from the hub while preserving custom user agents.
- Oracle process handling now works across Unix and Windows snapshot builds.
- Porter test verification now clears inherited `AETHER_OUTPUT_MODE` before running `go test`, avoiding false failures when Porter itself is rendered as JSON.

## [1.0.20] - 2026-04-23

Truth recovery and platform parity checkpoint release.

### Fixed
- FakeInvoker blocked from production paths; real invoker requires honest platform dispatch (R045)
- DispatchBatch error propagation ensures dispatch errors surface to callers (R046)
- Four continue bypass paths closed for verified_partial, watcher timeout, reconcile, and git claims (R047, R048)
- In-repo build claims are now git-verified for ALL completed workers (R049)
- Environmental dismissal removed from verification -- all failures produce honest summaries (R050)
- Colony state advancement is atomic via UpdateJSONAtomically; state saved before side effects (R051)
- Continue detects abandoned builds (all dispatches stuck at "spawned" >10 min) and returns blocked with recovery commands
- Stale report files cleared before continue verification runs
- 523 stale worktrees and 259 orphaned branches cleaned up
- All 18 unresolved blocker flags archived (issues fixed by phases 31-33)
- All 25 OpenCode agents synced from Claude masters (zero drift)
- All 25 Codex agents updated with Phase 31-33 runtime concepts

### Known Limitations
- Medic does not detect "functionally stuck" deadlocks where colony state is structurally valid but all forward paths are blocked. A dedicated fix is planned for a future release.

## [1.0.19] - 2026-04-22

Bootstrap asset-name compatibility for published npm installs.

### Fixed
- the npm bootstrap now requests the real GitHub release archive names (`Aether_<version>_<os>_<arch>.tar.gz`) instead of the old lowercase `aether_v...` pattern
- checksum downloads now follow the actual release asset name `checksums.txt`
- Windows bootstrap extraction now matches the shipped tar.gz release archives instead of assuming zip assets

## [1.0.18] - 2026-04-22

Unified release versioning, npm bootstrap publishing, and operator documentation.

### Changed
- the public Aether version is now treated as a single release number across `.aether/version.json`, `npm/package.json`, README badges, platform guides, and release runbooks
- the npm bootstrap package page now tracks the same product framing as the GitHub README, including who Aether is for and how the bootstrap hands off to the Go runtime
- OpenCode and architecture docs now describe the real source-checkout publish path (`aether install --package-dir "$PWD"`) instead of the old npm-global packaging flow

### Fixed
- the npm package page can now be updated intentionally through the documented release path instead of drifting behind the repo README with no operator guidance
- Medic and the platform guides now recognize release-integrity failures as a coordinated Go binary, hub publish, npm publish, and downstream update problem
- stale version references across README, AGENTS, CLAUDE, CODEX, and roadmap surfaces no longer disagree about the current Aether release

## [1.0.17] - 2026-04-21

Shared dispatch-truth completion and recovery continuity across runtime surfaces.

### Changed
- `aether status`, `aether watch`, `aether resume-dashboard`, and `print-next-up` now reuse the same blocked-recovery guidance instead of falling back to generic `aether continue`
- blocked continue flow now persists the targeted recovery command into session, context, and handoff artifacts so recovery survives context clears and resumed sessions
- Claude/OpenCode blocked continue guidance now follows runtime-surfaced recovery commands first, keeping the wrapper beautiful but honest

### Fixed
- blocked phases no longer strand the right redispatch or reconcile command inside the immediate `aether continue` output
- watch/status/resume surfaces now point at the same recovery doorway the runtime already computed, rather than drifting into stale or generic next steps
- the completed `v1.2` dispatch-truth and recovery work is now packaged as a clean patch release for cross-repo testing

## [1.0.16] - 2026-04-21

Legacy colony-state compatibility hardening for cross-repo updates.

### Fixed
- The shared Go colony-state loader now accepts legacy JSON-encoded array strings for `memory.phase_learnings`, `memory.decisions`, and `memory.instincts` instead of failing at unmarshal time
- Repos with older `COLONY_STATE.json` snapshots like `"phase_learnings":"[]"` can now load through the runtime normally across Codex, Claude Code, and OpenCode
- Cross-repo updates no longer require manual JSON surgery just to recover old colonies after moving to the latest Aether patch release

## [1.0.15] - 2026-04-21

Versioning cleanup, cross-platform parity hardening, and restored pheromone visibility.

### Changed
- Claude/OpenCode build and continue wrappers now expose pheromone strength and remaining-life context more clearly while keeping the Go runtime authoritative
- `aether pheromones` and `aether status` now share compact lifetime semantics so active steering signals are easier to interpret
- Maintainer-facing version surfaces are now aligned again across `.aether/version.json`, `README.md`, `AGENTS.md`, `CLAUDE.md`, and the roadmap snapshot doc

### Fixed
- `ant:pheromones` no longer teaches direct pheromone-file mutation and instead routes back through runtime-owned commands
- Colony scope separation, restored build/continue ceremony, and living watch/status surfaces are now all included in the shipped `v1.0` baseline together
- Other repos can consume the latest Aether bundle cleanly after a local install or release update without version-number drift between platform docs

## [1.0.14] - 2026-04-18

Codex wrapper rollback with visuals and worker spawning preserved.

### Changed
- Codex guidance now routes literal `aether ...` commands back through the direct CLI path instead of trying to recreate Claude-style pre-command wrapper flows in repo instructions and skill text
- Codex repo templates and project instructions keep `AETHER_OUTPUT_MODE=visual` guidance so the CLI still owns banners, caste visuals, and workflow ceremony

### Fixed
- `aether init`, `aether plan`, `aether build`, `aether continue`, `aether run`, and `aether seal` in Codex no longer trigger large repo archaeology passes or approval theater before the real CLI command
- Codex guidance tests now enforce the minimal-wrapper contract again, reducing the chance of another instruction-layer regression like `v1.0.13`

## [1.0.13] - 2026-04-18

Codex command-class parity restore for colony-shaping commands.

### Changed
- Codex guidance now splits literal `aether ...` commands into direct pass-through versus mediated workflow commands instead of flattening every command into the same fast path
- `aether init` guidance once again performs a short foundation pass and approval checkpoint before the real colony initialization, matching the older Claude-style experience more closely

### Fixed
- Codex no longer auto-runs colony-shaping commands like `init`, `plan`, `build`, `continue`, `run`, and `seal` as if they were read-only status commands
- Repo-level Codex instructions, generated templates, and platform command docs now consistently tell the model to say plainly when a requested command such as `aether dream` is not actually exposed by the binary
- `aether init-research` now returns basic repo shape data (`file_count`, `top_level_dirs`, `is_git_repo`) so approval prompts can stay grounded in the Aether runtime instead of ad hoc shell probing

## [1.0.12] - 2026-04-18

Codex command-wrapper simplification and visual no-colony status.

### Changed
- Codex-specific colony skills and generated repo templates now tell Codex to stop wrapping literal `aether ...` commands with repo archaeology, skill narration, and long post-command summaries

### Fixed
- `aether status` with no active colony now renders through the Aether visual layer instead of falling back to plain text
- Updated Codex repos now get stronger guidance to let Aether’s own banners, caste emoji output, and next-step blocks speak for themselves
- Literal `aether ...` commands in Codex are now documented to use a near-zero wrapper instead of the current play-by-play commentary

## [1.0.11] - 2026-04-18

Update error visibility fix.

### Fixed
- JSON error envelopes now preserve structured `details`, so commands like `aether update` expose the actual sync failures instead of only reporting a count
- Codex can now see file-level update errors directly instead of guessing at hidden causes from a generic `"update failed with N sync error(s)"` message

## [1.0.10] - 2026-04-18

Colony lifecycle restore and Codex runtime hardening.

### Added
- Real `parallel-mode worktree` execution for Codex builds, with isolated git worktrees, sync-back, and regression coverage
- A real `aether swarm` investigate/fix/verify flow with structured results and worker-wave coverage
- Legacy colony-state compatibility tests covering paused and broken idle colonies

### Changed
- Paused colonies now resume through a recoverable pause flag instead of leaving the colony trapped in an unusable lifecycle state
- Claude and OpenCode lifecycle wrappers now stay shorter and route users back into the runtime CLI without long conversational detours
- Build, plan, and colonize consistently inject dedicated pheromone sections into Codex worker prompts

### Fixed
- `aether resume`, `aether status`, and next-step routing now restore older paused colonies into a runnable state instead of dead-ending on `PAUSED`
- Paused colonies no longer show ghost active workers, and interrupted phases route back to the correct current build
- Nested Codex workers now receive access to the Codex home/session directory, so real worker dispatches can create session state reliably
- Poisoned `BUILT` phases can be retried instead of wedging behind "already built" errors
- Large repos no longer spend excessive time rescanning the workspace on every Codex skill match
- `continue` can verify a valid build packet from persisted manifest data even when `BuildStartedAt` is missing

## [1.0.9] - 2026-04-17

Codex orchestration hardening, autonomous Oracle execution, and parity-doc cleanup.

### Added
- Autonomous Oracle RALF loop execution through `aether oracle`, including watchdogs, heartbeat state, controller-led response merging, and stop/process-tree cleanup
- Regression coverage for platform doc hygiene across the main Claude/OpenCode lifecycle surfaces and all OpenCode agent docs

### Changed
- Codex workflow docs, Claude command wrappers, and OpenCode command wrappers now consistently treat the Go `aether` CLI as the runtime source of truth
- OpenCode agent guidance now reports progress and blockers through structured returns instead of legacy `.aether/aether-utils.sh` helper calls
- Release CI now enforces stronger validation for the Codex-oriented runtime path and packaging flow

### Fixed
- `aether oracle` no longer behaves like a workspace stub; it launches and manages a real iterative research loop
- Oracle runs no longer get stuck rereading their own workspace without producing findings; controller-led question packets now drive progress and persist concrete blocker reasons
- Remaining Claude/OpenCode lifecycle docs no longer teach direct colony-state mutation, stale shell-session behavior, or outdated council/update flows
- Codex parity guidance now aligns more closely with the actual worker-dispatch behavior for `colonize`, `plan`, `build`, `run`, `continue`, `seal`, `entomb`, and `update`

## [1.0.8] - 2026-04-17

Legacy colony-state compatibility and clearer update guidance.

### Changed
- `aether update` no longer implies that `aether status` is the required next step when nothing changed
- Colony load failures now report as state-load errors instead of always collapsing into "No colony initialized"

### Fixed
- Legacy colonies that store `plan.confidence` as a structured object now load correctly in the Go CLI again
- `status`, `continue`, `build`, `plan`, `seal`, `entomb`, `resume`, and compatibility flows no longer misreport a parse failure as a missing colony

## [1.0.7] - 2026-04-17

Codex session-reload guidance for repo setup and updates.

### Changed
- `aether setup` and `aether update` now report when refreshed Codex repo files require a new Codex chat before the changes take effect
- Codex-facing setup and update visuals now route the user to reopen Codex first whenever repo instructions, Codex agents, or Codex skills changed

### Fixed
- Updated repos no longer look "broken" to non-technical users after `aether update`; the CLI now explicitly explains that Codex must be reopened to load refreshed `AGENTS.md`, `.codex/CODEX.md`, agents, or skills

## [1.0.6] - 2026-04-17

Codex repo-update delivery fix for lifecycle orchestration and visual output.

### Added
- Managed `.codex/CODEX.md` template generation for Aether-enabled repos
- Regression coverage for Codex project-doc refresh during `setup` and `update`

### Changed
- `aether setup` and `aether update` now refresh Aether-managed `AGENTS.md` and `.codex/CODEX.md` in target repos
- Codex repo guidance and lifecycle skills now tell Codex to run lifecycle commands as `AETHER_OUTPUT_MODE=visual aether ...` unless JSON is explicitly requested

### Fixed
- Updated repos now receive the repo-level Codex instructions needed to execute `aether build`, `aether continue`, and related lifecycle commands directly instead of roleplaying them
- Codex lifecycle commands executed through the shell now default to Aether’s own visual renderer, restoring caste emojis and ANSI ceremony in non-TTY execution

## [1.0.5] - 2026-04-17

Codex CLI workflow hardening for real updated repos.

### Added
- `aether entomb` and `aether tunnels` for sealing follow-through and archived chamber browsing in the Go CLI
- Regression coverage for entomb, legacy session recovery, and completed-colony status rendering

### Changed
- Codex lifecycle skills and generated AGENTS guidance now treat literal `aether ...` input as an exact CLI command, not a vague workflow request
- Updated repos now restore the top-level `session.json` mirror from legacy colony-scoped sessions more reliably during `update` and `resume`

### Fixed
- Sealed colonies now route to `aether entomb` consistently across status, seal, resume, and next-step guidance
- Completed colonies no longer show stale incomplete task counts in the final phase display
- Entomb now archives legacy colony-scoped session state before clearing active runtime files

## [1.0.4] - 2026-04-17

Final release packaging pass for the Codex parity rollout.

### Fixed
- `go mod tidy` normalization is now committed so `goreleaser release --clean` no longer fails on a dirty `go.mod`
- Release metadata, docs, and hub versioning now point at the final `v1.0.4` patch tag for Codex CLI support

## [1.0.3] - 2026-04-17

Codex CLI release hardening and parity update. This release closes the remaining
runtime gaps between the Codex workflow and the established colony lifecycle,
and fixes clean-checkout packaging for the GoReleaser path.

### Added
- Native Codex compatibility commands: `aether run`, `aether watch`, and `aether oracle`
- Regression coverage for Codex compatibility flows, compact colony-prime worker context, mirror synchronization, and prompt budgeting

### Changed
- Codex workers now receive compact colony-prime context with hive wisdom, user preferences, blockers, decisions, and phase learnings
- README, AGENTS.md, and `.codex/CODEX.md` now describe Codex as a supported release workflow, including autopilot, live watch, and oracle workspace entrypoints
- Public command counts were corrected to 46 slash commands for Claude Code and OpenCode

### Fixed
- Expired and decayed pheromone signals no longer leak into normal worker prompt reads
- Final Codex worker prompts now enforce a global end-to-end budget instead of only budgeting individual sections
- Packaged agent mirrors for Claude and Codex were resynchronized with canonical agent definitions
- Codex worker schema serialization now returns errors instead of panicking on marshal failure
- Clean-source GoReleaser builds now succeed because `.aether/commands/` ships with a tracked placeholder file for `go:embed`
- Full validation stayed clean for release: `go test ./...`, `go vet ./...`, and `go test ./... -race`

## [5.3.0] - 2026-03-31

Aether v2.7 — PR Workflow + Stability. Six phases (39-44) adding multi-branch safety, clash detection, and release hardening.

### Added
- **Pheromone propagation** — Signals flow across git branches via `pheromone-snapshot-inject` and `pheromone-merge-back`; worktree creation auto-copies active pheromones
- **Midden collection** — Failure records from merged branches collected into main via `midden-collect` with idempotent dedup; cross-PR pattern detection via `midden-cross-pr-analysis`; revert-aware tagging via `midden-handle-revert`
- **Clash detection** — PreToolUse hook (`clash-pre-tool-use.js`) blocks edits to files modified in other active worktrees; `.aether/data/` files allowlisted (branch-local state)
- **Worktree utilities** — `_worktree_create` auto-copies colony context (COLONY_STATE.json, pheromones.json) and runs pheromone-snapshot-inject
- **Merge driver** — `.gitattributes` merge driver resolves package-lock.json conflicts by keeping "ours" via `merge-driver-lockfile.sh`
- **Midden wiring** — `midden-collect` and `midden-cross-pr-analysis` wired into `/ant-continue` playbooks (non-blocking, follows pheromone merge-back pattern)
- **Interactive installer** — `npx aether-colony` now shows a 3-option menu (Full setup / Global only / Repo only) with environment detection and context-sensitive defaults; supports `--global`, `--repo`, `--yes` flags for scripting
- **`aether setup` command** — CLI equivalent of `/ant-lay-eggs` for setting up Aether in a repo without Claude Code open

### Changed
- **Package validation** — `validate-package.sh` expanded from 15 to 38+ required file entries (100% coverage of packaged utils)
- **NPX installer** — Replaced `npx-install.js` with interactive `npx-entry.js`; old installer kept as deprecation redirect
- **Package cleanliness** — 8 dev-only files excluded from npm tarball (scripts/, design docs, example schemas)
- **CLAUDE.md** — Full accuracy audit: version bumped to v2.7.0, all counts verified (5,500 lines, 35 utils, 45 commands, 509 tests)
- **README.md** — Architecture counts updated (35 utils, 45 commands, ~5,500 lines)
- **YAML commands** — 6 stale command files regenerated from YAML sources (init, plan, seal for Claude and OpenCode)

### Fixed
- **Clash detection dispatcher** — `clash-detect.sh` and `worktree.sh` wired into `aether-utils.sh` dispatcher (source lines, dispatch cases, help JSON)
- **Init command** — Clash detection hook verification and read-only worktree list integrated into `/ant-init` Step 7.6

## [2.1.0] - 2026-03-24

Six phases of production hardening (Phases 9-14) targeting reliability, maintainability, and developer experience.

### Added
- **State API facade** (`state-api.sh`) -- centralized COLONY_STATE.json access with lock/validate/migrate pattern
- **Builder output verification** (`verify-claims`) -- cross-references builder file claims against filesystem to catch fabrication
- **Error handling infrastructure** (`_aether_log_error`) -- structured error logging across all modules with `[error]` prefix
- **SUPPRESS:OK convention** -- intentional error suppressions annotated for auditability (cleanup, read-default, existence-test, cross-platform, idempotent, validation)
- **Per-phase research** -- scouts investigate domain knowledge before task decomposition (`/ant-plan` Step 3.6)
- **Research context injection** -- builder and watcher prompts receive domain research during builds (16K character budget)
- **Deprecation warning system** (`_deprecation_warning`) -- 18 dead subcommands emit stderr warnings with `[deprecated]` prefix
- **Rolling state checkpoints** -- COLONY_STATE.json backed up before every build-wave (3 max retained)
- **Context trimming notification** -- colony-prime emits visible notice when context is trimmed to stay within budget

### Changed
- **Monolith modularization** -- aether-utils.sh reduced from ~11,600 to ~5,200 lines (55% reduction)
  - 9 domain modules extracted: flag, spawn, session, suggest, queen, swarm, learning, pheromone, state-api
  - ~72 subcommands moved to domain modules, sourced on demand
  - All 580+ tests passing with modularized structure
- **Error handling overhaul** -- ~110 lazy error suppressions replaced with proper fallbacks, ~48 dangerous suppressions on state-mutation paths fixed with explicit error handling
- **Continue-advance state writes** -- now go through locked subcommand (prevents concurrent corruption)
- **Help JSON** -- updated with deprecation markers and Deprecated section
- **Documentation accuracy** -- all docs/ files swept for stale counts, line numbers, and dates; comprehensive v2.1 changelog added

### Fixed
- **Hive wisdom type coercion** -- retrieval works regardless of string/number confidence format
- **Midden race condition** -- PID-scoped temp files prevent concurrent write data loss
- **Learning recovery** -- corrupted learning-observations.json auto-recovers from template
- **Date-sensitive test failures** -- dynamic dates (futureISO) prevent recurring expiration failures
- **Context-continuity test** -- pre-existing failure fixed (QUAL-09)

## [2.0.0] - 2026-03-21

### Added
- **Hive Brain** — cross-colony wisdom sharing with domain-scoped retrieval, 200-entry LRU, multi-repo confidence boosting
- **Autopilot (`/ant-run`)** — automated build-verify-advance loop with smart pausing
- **User Preferences** — colony adapts to user communication style via QUEEN.md
- **Quality gate agents** — Probe (coverage), Auditor (quality), Gatekeeper (security), Measurer (performance)
- **`instinct-apply` subcommand** — tracks instinct usage with confidence adjustment
- **`midden-review` subcommand** — lists unacknowledged failures grouped by category
- **`midden-acknowledge` subcommand** — marks midden entries as addressed
- **Pheromone content deduplication** — SHA-256 hashing prevents duplicate signals
- **Pheromone prompt injection sanitization** — blocks LLM instruction override attempts
- **Colony-prime token budget** — 8000/4000 char budget with priority-based truncation
- **`/ant-patrol`** — comprehensive pre-seal audit of work against plan
- **Colony pheromone exchange** — XML export/import for cross-colony signal sharing

### Changed
- **Monolith decomposition** — extracted hive-* (561 lines) and midden-* (260 lines) from aether-utils.sh to .aether/utils/
- **Instinct trigger format** — triggers stored without "When" prefix; display/promotion adds it
- **Learning-promote-auto** — recurrence-calibrated confidence formula

### Fixed
- **"When when" stutter** — fixed across 4 locations (seal.md, prompt-print, learning-promote-auto, oracle)
- **Seal hive promotion** — jq path `.instincts[]` corrected to `.memory.instincts[]`
- **instinct-create locking** — added trap-based lock to prevent concurrent write corruption
- **midden-recent-failures limit** — positional parameter fix after extraction

## [1.1.11] - 2026-02-26

### Fixed
- `midden-recent-failures` now reads `.entries[]` from midden.json instead of querying non-existent `.signals[]` — builders can finally see past failures

### Added
- `instinct-create` subcommand with deduplication and 30-instinct cap — programmatic instinct management replaces manual JSON manipulation
- Midden context injected into builder prompts during build waves — workers avoid repeating past mistakes
- Decisions auto-emit FEEDBACK pheromones (strength 0.65, TTL 30d) so builders see architectural choices
- `context-update constraint` now handles `feedback` type alongside `redirect` and `focus`
- Context-update calls added to Claude Code pheromone commands (focus, redirect, feedback)
- OpenCode pheromone commands brought to parity with pheromone-write + context-update calls

### Changed
- `continue-advance.md` instinct extraction simplified to use `instinct-create` instead of inline JSON
- `learning-promote-auto` now also creates instincts from promoted learnings

## [1.1.10] - 2026-02-26

### Changed
- `lay-eggs` is now a pure bootstrap command — sets up `.aether/` in a repo from the global hub without starting a colony
- `init` now assumes Aether is already set up and focuses only on starting a colony with a goal
- All help files, rules, and workflow documentation updated to reflect the lay-eggs/init separation

## [1.1.5] - 2026-02-23

### Added
- `validate-worker-response` command with caste-aware schema checks for builder/watcher/probe/scout payloads, wired into build playbooks and OpenCode build flow.
- `queen-thresholds`, `spawn-efficiency`, `entropy-score`, `incident-rule-add`, and `eternal-store` commands.
- Incident/self-evolution artifacts: `.aether/docs/INCIDENT_TEMPLATE.md`, `.aether/scripts/weekly-audit.sh`, `.aether/scripts/incident-test-add.sh`.
- Additional regression coverage for spawn enforcement, state lock failure handling, threshold output, worker validation, spawn efficiency, eternal memory promotion, and entropy scoring.

### Changed
- Spawn guard now supports hard enforcement via `spawn-can-spawn [depth] --enforce`; worker and playbook guidance updated to use enforced mode.
- Build orchestration guidance now refreshes `colony-prime --compact` before each worker dispatch so new pheromones/memory are visible mid-build.
- Promotion thresholds consolidated behind a single source of truth and reused by learning promotion/proposal/memory metric paths.
- `midden.template.json` extended with `spawn_metrics`.

### Fixed
- Locked + atomic state writes for critical mutation paths in `COLONY_STATE.json` handling (`error-add`, failure event append in `spawn-complete`, `grave-add`, schema migration, and lock-aware state-loader validation handoff).
- Pheromone write path now sanitizes content, validates strength bounds, and uses lock + atomic write semantics.
- `pheromone-expire` now promotes high-strength expired signals (`>0.8`) into eternal memory instead of silently decaying.
- `detectDirtyRepo` now preserves porcelain status columns for the first line (prevents staged/unstaged misclassification when parsing git status output).

---

## v5.0.0 — Worker Emergence (2026-02-20)

**Major milestone:** Every ant caste is now a real Claude Code subagent. 22 agents ship through the hub, ready for resolution via the Task tool.

### Added
- **22 Claude Code subagents**: Every ant caste is now a first-class subagent resolvable via the Task tool
  - Core: Builder, Watcher
  - Orchestration: Queen, Scout, Route-Setter, 4 Surveyors (nest, disciplines, pathogens, provisions)
  - Specialists: Keeper, Tracker, Probe, Weaver, Auditor
  - Niche: Chaos, Archaeologist, Gatekeeper, Includer, Measurer, Sage, Ambassador, Chronicler
- **Agent distribution pipeline**: `npm install -g .` syncs agents to hub at `~/.aether/system/agents-claude/`, `aether update` delivers to target repos
- **6 AVA tests for agent quality**: Frontmatter validation, tool restrictions, naming conventions, content standards
- **repo-structure.md**: Quick re-orientation guide for the codebase

### Fixed
- **Bash line wrapping bug**: Fixed 58 instances across 7 command files where description text was inside code blocks causing "with: command not found" errors
- **Lint regression test**: CLEAN-03 test scans both Claude and OpenCode command directories

### Changed
- **.aether/docs/ curated**: 8 keepers at root, 6 archived for reference
- **README.md updated**: Action-oriented tone, v5.0 agent capabilities featured, caste table by tier
- **ROADMAP.md and STATE.md**: v5.0 marked shipped, all 31 phases complete at 100%

### Phases Shipped
- **Phase 27**: Distribution Infrastructure + First Core Agents (4 plans)
- **Phase 28**: Orchestration Layer + Surveyor Variants (3 plans)
- **Phase 29**: Specialist Agents + Agent Tests (3 plans)
- **Phase 30**: Niche Agents (3 plans)
- **Phase 31**: Integration Verification + Cleanup (3 plans)

---

### Removed
- Old planning phases 10-19 archived to docs/plans/ (completed phases)
- Orphaned worktree salvage files moved to docs/worktree-salvage/

---

## v4.0.0 -- Distribution Simplification

**Breaking change:** The `runtime/` staging directory has been removed. The npm package now reads directly from `.aether/`, with private directories (data/, dreams/, oracle/, etc.) excluded by `.aether/.npmignore`.

### Changed
- `.aether/` is now directly included in the npm package (private dirs excluded by `.aether/.npmignore`)
- `bin/validate-package.sh` replaces `bin/sync-to-runtime.sh` — validates required files, no copying
- Hub sync uses exclude-based approach instead of triplicate 59-62 file allowlists
- Pre-commit hook repurposed for validation (no more runtime/ sync)
- `aether update` uses `syncAetherToRepo` (exclude-based) for all system file distribution
- All three distribution paths (system files, commands, agents) unified in `setupHub()`

### Removed
- `runtime/` staging directory — eliminated entirely
- `bin/sync-to-runtime.sh` — replaced by validation-only script
- `SYSTEM_FILES` allowlist arrays in cli.js and update-transaction.js
- `copySystemFiles()` and `syncSystemFilesWithCleanup()` functions

### Added
- `bin/validate-package.sh` — pre-packaging validation with `--dry-run` mode
- Private data exposure guard — blocks packaging if .npmignore doesn't cover private dirs
- Migration message for users upgrading from v3.x
- `npm pack --dry-run` recommended for verifying package contents

### Fixed
- ISSUE-004: Template path hardcoded to runtime/ — resolved by eliminating runtime/ entirely

### Migration
- Run `npm install -g aether-colony` to get v4.0
- Your colony state and data are unaffected
- The only change is how the package is built — distributed content is identical

---

## [3.1.5] - 2026-02-15

### Fixed
- **Agent Type Correction** — Changed all occurrences of `subagent_type="general"` to `subagent_type="general-purpose"` across all command files. The error "Agent type 'general' not found" was occurring because the correct agent type name is `general-purpose`. Fixed in: build.md, plan.md, organize.md, and workers.md (both Claude and OpenCode versions, plus runtime copy). (`.claude/commands/ant/build.md`, `.claude/commands/ant/plan.md`, `.claude/commands/ant/organize.md`, `.opencode/commands/ant/build.md`, `.opencode/commands/ant/plan.md`, `.opencode/commands/ant/organize.md`, `.aether/workers.md`, `runtime/workers.md`)

## [3.1.4] - 2026-02-15

### Fixed
- **Archaeologist Visualization** — Added swarm display integration for the Archaeologist scout (Step 4.5). The archaeologist now appears in the visual display with proper emoji (🏺), progress tracking (15% → 100%), and tool usage stats when spawned during pre-build scans. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)

## [3.1.3] - 2026-02-15

### Fixed
- **Nested Spawn Visualization** — When builders or watchers spawn sub-workers, the swarm display now updates to show those nested spawns with colors and emojis. Added `swarm-display-update` calls to workers.md spawn protocol (Step 3 and Step 5), builder prompts, and watcher prompts. (`.aether/workers.md`, `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)

## [3.1.2] - 2026-02-15

### Fixed
- **Swarm Display Integration in Build Command** — The visualization system was fully implemented but never integrated into `/ant-build`. Added `swarm-display-init` at build start, `swarm-display-update` calls when spawning builders/watchers/chaos ants, progress updates when workers complete (updating to 100% completion), and final `swarm-display-render` at build completion. The build now shows real-time ant-themed visualization with caste emojis, colors, tool usage stats, and chamber activity maps. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)
- **Missing swarm-display-render Command** — Added new `swarm-display-render` command to `aether-utils.sh` that executes the visualization script to render the current swarm state to terminal. (`.aether/aether-utils.sh`)

### Changed
- **OpenCode Build Command Parity** — Synchronized OpenCode build.md with Claude version: added `--model` flag support, proxy health check (Step 0.6), colony state loading (Step 0.5), and full swarm display integration. (`.opencode/commands/ant/build.md`)

## [3.1.1] - 2026-02-15

### Fixed
- **Missing Visualization Assets** — Added `.aether/visualizations/` directory to npm package files array. The ASCII art anthill files required by `/ant-maturity` command were not being published, causing the command to fail in repos that installed/updated via npm. (`package.json`)
- **Visualization Sync in Install** — Updated `setupHub()` function in CLI to sync visualization files from package to hub (`~/.aether/visualizations/`). (`bin/cli.js`)
- **Visualization Sync in Update** — Updated `UpdateTransaction` to sync visualization files from hub to repos during `aether update`. Added `HUB_VISUALIZATIONS` constant and visualization sync result tracking. (`bin/lib/update-transaction.js`)

### Changed
- Version bump to 3.1.1 to trigger fresh installs with visualization assets. (`package.json`)

## [Unreleased]

### Added
- **Session Freshness Detection System** — Global system to prevent stale session files from silently breaking Aether workflows. Implements `session-verify-fresh` and `session-clear` commands with support for 7 commands (survey, oracle, watch, swarm, init, seal, entomb). Features cross-platform timestamp detection (macOS/Linux), environment variable overrides for testing, and protected operations (init/seal/entomb never auto-clear). Backward compatibility maintained with `survey-verify-fresh` and `survey-clear` wrappers. Added comprehensive test suite (`tests/bash/test-session-freshness.sh`) and API documentation (`docs/session-freshness-api.md`). (`.aether/aether-utils.sh`, `tests/bash/test-session-freshness.sh`, `docs/session-freshness-api.md`)
  - `/ant-colonize` — Added `--force-resurvey` flag, stale survey detection, and verification
  - `/ant-oracle` — Added `--force` flag, stale session detection with user options
  - `/ant-watch` — Added session timestamp capture and stale file handling
  - `/ant-swarm` — Added auto-clear for stale findings with verification
  - `/ant-init` — Added freshness check with protected state (no auto-clear)
  - `/ant-seal` — Added incomplete archive detection and integrity verification
  - `/ant-entomb` — Added incomplete chamber detection and integrity verification

### Fixed
- **Architecture Cleanup: Source of Truth Flipped** — Complete review and cleanup of the flipped source-of-truth architecture. `.aether/` is now the source of truth for system files, with `runtime/` auto-populated by `bin/sync-to-runtime.sh` during npm install. Fixed 6 stale documentation files, updated 5 planning files, expanded allowlist from 20 to 36 files, handled 4 orphan files, and verified zero drift between directories. (`.aether/recover.sh`, `.aether/RECOVERY-PLAN.md`, `.planning/codebase/STRUCTURE.md`, `.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/CONVENTIONS.md`, `TO-DOS.md`, `bin/sync-to-runtime.sh`, `bin/lib/update-transaction.js`)

### Changed
- **Phase 4: UX Improvements to Lay-Eggs** — Enhanced lay-eggs.md with visual lifecycle diagrams (🟢 ACTIVE COLONY → 🏺 SEAL/ENTOMB → 🥚 LAY EGGS) in error messages to clarify workflow progression. Updated success output to explicitly distinguish between sealing vs laying eggs for new projects, preserving wisdom across colony lifecycles. (`.claude/commands/ant/lay-eggs.md`)

### Fixed
- **Phase 2: Fix Blocker Severity and Auto-Resolve Logic** — Made auto_resolve_on conditional by flag source: chaos-sourced blockers require manual resolution (auto_resolve_on: null), verification blockers auto-resolve on build pass. Reordered continue.md Flags Gate to run auto-resolve before blocker count check. Added advisory blocker warning (Step 1.5) to build.md so builders see active blockers before execution. (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`, `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md`, `.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`)
- **Phase 1: Fix Chaos Ant Duplicate Flagging** — Eliminated duplicate flag creation during build-rebuild cycles by removing redundant chaos flagging from build.md Step 5.5 (Step 5.4.2 already handles it), injected existing flag titles into Chaos Ant spawn prompt to prevent re-investigating known issues, and added flag persistence to standalone /ant-chaos for critical/high findings using source 'chaos-standalone'. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/build.md`, `.claude/commands/ant/chaos.md`, `.opencode/commands/ant/chaos.md`)

### Added
- **Phase 6: Final Verification and Integration Testing** — Added milestone display to /ant-status command (now shows "Milestone: <name>" in output), expanded milestone progression in /ant-archive to handle all 6 stages (First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill), added unrecognized milestone error handling. Full lint suite passes (shell, JSON, sync - 28 commands verified). (`.claude/commands/ant/status.md`, `.claude/commands/ant/archive.md`, `.opencode/commands/ant/status.md`, `.opencode/commands/ant/archive.md`)

- **Phase 1: Create Oracle infrastructure and command** — Added Oracle Ant deep research agent with RALF-pattern bash loop, agent prompt, /ant-oracle command definition (mirrored), and oracle caste registration in aether-utils.sh. (`.aether/oracle/oracle.sh`, `.aether/oracle/oracle.md`, `.claude/commands/ant/oracle.md`, `.opencode/commands/ant/oracle.md`, `.aether/aether-utils.sh`, `runtime/aether-utils.sh`)

### Verified
- **Phase 2: Verification and smoke test** — All Oracle Ant files verified: generate-commands.sh check passes (26/26 in sync, SHA-1 checksums verified), oracle.sh error handling works (exits code 1 with descriptive error when no research.json), oracle caste generates themed names (Vision-NN, Delph-NN), file structure matches spec (oracle.sh executable, oracle.md exists, command mirrors byte-identical, no stray files). Full lint suite passes (shell, JSON, sync).

### Verified
- **Phase 5: Path Localization Complete** — Full-repo audit confirmed zero actionable `~/.aether/` or `~/.config/opencode/` references in commands, scripts, or CLI. Remaining 4 `$HOME/.aether` references in aether-utils.sh are intentional hub/registry functions for multi-repo management. `generate-commands.sh check` passes (25/25 in sync, SHA-1 checksums verified). `aether-utils.sh` smoke tests pass (help, version, generate-ant-name). All three original goals met: no root access prompts, no cross-repo contamination, no out-of-project file operations.

### Changed
- **Phase 4: Implement Hash-Based Idempotency** — Added SHA256 hash comparison to syncDirWithCleanup function - files are now only copied when content actually changes, reducing unnecessary I/O. Added error handling with try-catch to hashFileSync, copyFileSync, unlinkSync operations to prevent crashes on single file errors. Added validateManifest function to verify manifest.json structure before use. Added optional --backup flag to preserve user-modified files before overwriting. 13 tests added covering hash comparison, user modification detection, and backward compatibility. (`bin/cli.js`, `test/sync-dir-hash.test.js`, `test/user-modification-detection.test.js`)

- **Phase 4: Remove global path operations from cli.js** — Removed `~/.aether/` runtime copy logic (RUNTIME_DEST, RUNTIME_SRC, learnings.json creation, execSync import) and `~/.config/opencode/` global install logic (OPENCODE_GLOBAL_COMMANDS_DEST, OPENCODE_GLOBAL_AGENTS_DEST) from install/uninstall commands. Updated help text to reflect new architecture. cli.js now only installs Claude Code slash-commands to `~/.claude/commands/ant/`. Net reduction of 130 lines. (`bin/cli.js`)
- **Phase 3: Document repo-local path architecture** — Updated CHANGELOG.md and documentation references to reflect the completed path localization migration. All runtime paths now use repo-local `.aether/` instead of `~/.aether/`. Phases 1-2 localized command files, agent definitions, system docs, planning docs, shell utilities, and cross-project state functions. Phase 3 updates documentation to reflect the new architecture where running colonies only read/write repo-local `.aether/`, while global install remains functional for command distribution. (`README.md`, `CHANGELOG.md`, `TO-DOS.md`)
- **Phase 2: Localize cross-project state in aether-utils.sh** — Redirected 6 `$HOME/.aether` references to repo-local `$DATA_DIR` paths in learning-promote, learning-inject, error-flag-pattern, error-patterns-check, signature-scan, and signature-match functions. Fixed atomic-write.sh `$HOME/.aether/utils` fallback to use SCRIPT_DIR-based resolution. Updated usage comment headers in all .sh files. Applied to both `.aether/` and `runtime/` copies (verified identical). (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`, `.aether/utils/atomic-write.sh`, `runtime/utils/atomic-write.sh`, `.aether/utils/file-lock.sh`, `runtime/utils/file-lock.sh`)
- **Phase 1: Localize ~/.aether/ path references** — Replaced all `~/.aether/` paths with repo-relative `.aether/` across 50 files: command prompts, agent definitions, system docs, planning docs, and template.yaml. Fixed 3 pre-existing mirror drifts (migrate-state.md, organize.md, plan.md) discovered during verification. (`.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.opencode/agents/*.md`, `.aether/workers.md`, `runtime/workers.md`, `.aether/docs/*.md`, `runtime/docs/*.md`, `.planning/*.md`, `src/commands/_meta/template.yaml`)
- **Phase 2: Upgrade Sync Checking to Content-Aware** — `generate-commands.sh check` now performs SHA-1 checksum comparison (Pass 2) after filename matching (Pass 1), detecting content drift between `.claude/` and `.opencode/` mirrors. Revealed 3 pre-existing drifts previously invisible to filename-only checks. (`bin/generate-commands.sh`)

### Verified
- **Phase 5: Verify Full System Integrity** — Final verification phase confirming all global install locations match repo sources. Full lint suite passed (lint:shell, lint:json, lint:sync). All 4 global locations verified: ~/.claude/commands/ant/ (24 files), ~/.config/opencode/commands/ant/ (24 files), ~/.config/opencode/agents/ (4 files), ~/.aether/ system files. Watcher quality 9/10, Chaos resilience moderate (1 high finding: lint:sync content blind spot, 3 medium, 1 low — all pre-existing infrastructure gaps). Colony goal achieved.

### Fixed
- **Phase 6: Document and Test the System** — Created command-sync.md documenting the sync strategy: Claude Code uses global sync to ~/.claude/commands/ant/, OpenCode uses hub-based repo-local distribution (no global discovery). Verified end-to-end sync with dry-run tests: install, update, update --all all work correctly. Idempotency confirmed - hash-based comparison skips unchanged files. All lint and tests pass. (`.aether/docs/command-sync.md`, `.aether/docs/namespace.md`, `test/namespace-isolation.test.js`)

- **Phase 5: Implement Conflict Prevention System** — Added namespace isolation documentation (`.aether/docs/namespace.md`) explaining why 'ant' namespace is distinct from cds, mds, st: namespaces. Created namespace-isolation.test.js with 8 tests verifying bulletproof directory-based isolation. Fixed critical bugs discovered by Chaos Ant: added hash comparison to syncDirWithCleanup (files now only copied when content changes) and added HOME environment variable validation to prevent path.join failures. All 14 tests pass. (`bin/cli.js`, `.aether/docs/namespace.md`, `test/namespace-isolation.test.js`)

- **Phase 3: Fix Pheromone Model Consistency** — Aligned all pheromone documentation to TTL-based model. Replaced decay/half-life/exponential language in runtime/docs/pheromones.md (now identical to .aether/docs/ source of truth), help.md (4 references fixed, mirrored to .opencode/), and README.md pheromone table (Decay column replaced with Priority/Default Expiration). (`runtime/docs/pheromones.md`, `.claude/commands/ant/help.md`, `.opencode/commands/ant/help.md`, `README.md`)
- **Phase 1: Fix Command Mirror Sync Bugs** — Synced status.md, continue.md, and phase.md between Claude and OpenCode mirrors (zero diff verified), added missing YAML frontmatter to Claude's migrate-state.md, verified cli.js install paths are correct. (`.opencode/commands/ant/status.md`, `.opencode/commands/ant/continue.md`, `.opencode/commands/ant/phase.md`, `.claude/commands/ant/migrate-state.md`)
- **Phase 2: Sync runtime copy to .aether mirror** — Full file copy of runtime/aether-utils.sh to .aether/aether-utils.sh eliminating all drift including missing signature-scan and signature-match commands. Both copies now byte-identical. (`.aether/aether-utils.sh`)
- **Phase 1: Fix bugs in canonical runtime/aether-utils.sh** — Fixed learning-promote jq crash on non-numeric phase strings (--argjson to --arg), fixed flag-auto-resolve missing exit after early-return when flags file absent, confirmed file-lock.sh transitive usage via atomic-write.sh is intentional. (`runtime/aether-utils.sh`)

### Changed
- **Phase 4: Clean Up Global ~/.aether/** — Removed unrelated `LIGHT_MODE_TRANSPARENCY_TEST.md` from global `~/.aether/`, adopted orphaned `progressive-disclosure.md` into repo at both `.aether/docs/` and `runtime/docs/`. Both copies verified identical to global source. (`.aether/docs/progressive-disclosure.md`, `runtime/docs/progressive-disclosure.md`)
- **Phase 3: Sync Global OpenCode Commands** — Replaced stale global OpenCode commands at `~/.config/opencode/commands/ant/` with all 24 current repo commands. Removed orphan `ant.md`, cleared old files, installed fresh copies. All 24 files verified identical to repo source. (`~/.config/opencode/commands/ant/*.md`)
- **Phase 2: Sync Content Between Repo and Runtime** — Synced runtime/QUEEN_ANT_ARCHITECTURE.md with .aether/ source (+56 lines: Council, Swarm sections, heading rename), added 3 missing docs to runtime/docs/ (constraints.md, pathogen-schema.md, pathogen-schema-example.json), synced aether-watcher.md to global install with Command Resolution section. (`runtime/QUEEN_ANT_ARCHITECTURE.md`, `runtime/docs/*`, `~/.config/opencode/agents/aether-watcher.md`)
- **Phase 1: Fix OpenCode Command Naming Convention** — Renamed all 24 `.opencode/commands/ant/` files from `ant:*.md` to bare `*.md` names to match `.claude/commands/ant/` convention. OpenCode uses frontmatter `name:` field for command resolution, so filenames are cosmetic. `npm run lint:sync` now passes. (`.opencode/commands/ant/*.md`)
- **Phase 4: Documentation and Validation (Chaos + Archaeologist)** — Updated help.md with /ant-chaos and /ant-archaeology in ADVANCED and WORKER CASTES sections, updated README.md command count from 22 to 24 in all 6 locations, added CHANGELOG entries for all phases, marked both TO-DOS.md entries as DONE with implementation references. Validated 24 files in each command directory, name generation, and emoji resolution. (`.claude/commands/ant/help.md`, `.opencode/commands/ant/ant-help.md`, `README.md`, `CHANGELOG.md`, `TO-DOS.md`)
- 2026-02-12: TO-DOS.md — Marked Chaos Ant and Archaeologist Ant entries as DONE with implementation references
- **Phase 1: Threshold and Quoting Fixes** — Lowered instinct confidence threshold from 0.7 to 0.5 in both init.md mirrors, standardized YAML description quoting across all 26 command files. (`init.md`, `build.md`, `colonize.md`, `continue.md`, `council.md`, `dream.md`, `feedback.md`, `flag.md`, `flags.md`, `focus.md`, `help.md`, `interpret.md`, `organize.md`, `pause-colony.md`, `phase.md`, `plan.md`, `redirect.md`, `resume-colony.md`, `status.md`, `swarm.md`, `watch.md` + .opencode mirrors)
- **Phase 3: Watcher, Builder, and Swarm command resolution** — Watcher prompt in build.md, swarm.md Step 8, and aether-watcher.md now resolve build/test/lint commands via the 3-tier priority chain (CLAUDE.md > CODEBASE.md > heuristic fallback) instead of leaving commands unspecified or hardcoded. (`build.md`, `swarm.md`, `aether-watcher.md` + .opencode mirrors)
- **Phase 2: Verification loop priority chain** — Command detection in continue.md and verification-loop.md now uses 3-tier priority chain (CLAUDE.md > CODEBASE.md > heuristic table) instead of heuristic table alone. Heuristic table preserved as fallback. (`continue.md`, `runtime/verification-loop.md` + .opencode/.aether mirrors)
- **Phase 3: Build Pipeline Integration (Chaos + Archaeologist)** — Integrated both new ant types into the build.md pipeline. Archaeologist Ant spawns as conditional pre-build step (Step 4.5) when phase modifies existing files, injecting history context into builder prompts. Chaos Ant spawns as post-build resilience tester (Step 5.4.2) alongside Watcher, limited to 5 edge case scenarios. Added `chaos_count` and `archaeologist_count` to spawn_metrics and `archaeology` field to synthesis JSON. (`.claude/commands/ant/build.md`, `.opencode/commands/ant/ant-build.md`)

### Added
- **Phase 2: `/ant-chaos` command** — Standalone Chaos Ant (Resilience Tester) command that probes code for edge cases, boundary conditions, error handling gaps, state corruption, and unexpected inputs. Produces structured findings reports with reproduction steps and severity ratings. Read-only by design (Tester's Law). (`.claude/commands/ant/chaos.md`, `.opencode/commands/ant/ant-chaos.md`)
- **Phase 2: `/ant-archaeology` command** — Standalone Archaeologist Ant command that excavates git history for any file or directory. Uses git log, blame, show, and follow to analyze commit patterns, surface tribal knowledge, identify tech debt markers, map churn hotspots, and produce structured archaeology reports. Read-only by design (Archaeologist's Law). (`.claude/commands/ant/archaeology.md`, `.opencode/commands/ant/ant-archaeology.md`)
- **Phase 1: Utility Foundation (Chaos + Archaeologist)** — Added chaos and archaeologist castes to `generate-ant-name` (8 prefixes each) and `get_caste_emoji` (🎲 and 🏺) in both `.aether/aether-utils.sh` and `runtime/aether-utils.sh`. (`.aether/aether-utils.sh`, `runtime/aether-utils.sh`)
- **Phase 1: Immune Memory Schema** — Defined JSON schema for pathogen signatures extending existing error-patterns.json format. Schema adds signature_type, pattern_string, confidence_threshold, escalation_level fields while preserving backward compatibility. Created .aether/docs/pathogen-schema.md documentation, .aether/docs/pathogen-schema-example.json with sample entries, and .aether/data/pathogens.json empty storage file. Watcher verified 6/6 jq validation tests pass. (`.aether/docs/pathogen-schema.md`, `.aether/docs/pathogen-schema-example.json`, `.aether/data/pathogens.json`)
- **Phase 2: Add Lint Scripts** — Added `lint:shell`, `lint:json`, `lint:sync`, and top-level `lint` scripts to package.json for shell validation, JSON validation, and mirror sync checking. (`package.json`)
- **CLAUDE.md-aware command detection** — Colonize now extracts build/test/lint commands from CLAUDE.md and package manifests into CODEBASE.md with user suggestions. Verification loop and worker prompts resolve commands via 3-tier priority chain (CLAUDE.md > CODEBASE.md > heuristic fallback) instead of heuristic table alone. (`colonize.md`, `continue.md`, `build.md`, `swarm.md`, `verification-loop.md`, `aether-watcher.md` + .opencode/.aether mirrors)
- **Phase 4: Tier 2 Gate-Based Commit Suggestions** — Colony now suggests commits at verified boundaries (post-advance and session-pause) via user prompt instead of auto-committing. Added `generate-commit-message` utility to aether-utils.sh for consistent formatting across commit types. (`continue.md`, `pause-colony.md`, `aether-utils.sh` + .opencode mirrors)
- **Phase 3: Tier 1 Safety Formalization** — Switched build.md checkpoint from `git commit` to `git stash push --include-untracked`, standardized checkpoint naming under `aether-checkpoint:` prefix, added label parameter to `autofix-checkpoint` in aether-utils.sh, added rollback verification to build.md output header, documented rollback procedure in continue.md, updated swarm.md to pass descriptive labels. (`build.md`, `swarm.md`, `continue.md`, `aether-utils.sh` + .opencode mirrors)
- **Phase 2: Git Staging Strategy Proposal** — 4-tier strategy proposal with comparison matrix and implementation recommendation. Tier 1 (Safety-Only), Tier 2 (Gate-Based Suggestions), Tier 3 (Hooks-Based Automation), Tier 4 (Branch-Aware Colony). Recommends Tiers 1+2 for initial implementation. (`.planning/git-staging-proposal.md`, `.planning/git-staging-tier{1-4}.md`)
- **Phase 1: Deep Research on Git Staging Strategies** — 7 research documents (1573 lines) covering: Aether's 20 git touchpoints, industry comparison of 5 AI tools, worktree applicability, user git rule tensions, ranked commit points (POST-ADVANCE strongest), commit message conventions, and GitHub integration opportunities. (`.planning/git-staging-research-1.{1-7}.md`)
- **Auto-recovery headers** — All ant commands now show `🔄 Resuming: Phase X - Name` after `/clear`. `status.md` has Step 1.5 with extended format including last activity timestamp. `build.md`, `plan.md`, `continue.md` show brief one-line context. `resume-colony.md` documents the tiered pattern. (`status.md`, `build.md`, `plan.md`, `continue.md`, `resume-colony.md`)
- **Ant Graveyards** — `grave-add` and `grave-check` commands in `aether-utils.sh`. When builders fail, grave markers record the file, ant name, and failure summary. Future builders check for nearby graves before modifying files and adjust caution level accordingly. Capped at 30 entries. (`aether-utils.sh`, `init.md`, `build.md`)
- **Colony knowledge in builder prompts** — Spawned workers now receive top instincts (confidence >= 0.5), recent validated learnings, and flagged error patterns via `--- COLONY KNOWLEDGE ---` section in builder prompt template. (`build.md`)
- **Automatic changelog updates** — `/ant-continue` now appends a changelog entry for each completed phase under `## [Unreleased]`. (`continue.md`)
- **Colony memory inheritance** — `/ant-init` now reads the most recent `completion-report.md` (if it exists) and seeds the new colony's `memory.instincts` with high-confidence instincts (>= 0.7) and validated learnings from prior sessions. Colonies no longer start completely blind. (`init.md` + .opencode mirror)
- **Unbuilt design status markers** — Added `STATUS: NOT IMPLEMENTED` headers to `.planning/git-staging-tier3.md` and `.planning/git-staging-tier4.md` to prevent confusion with implemented features. (`git-staging-tier3.md`, `git-staging-tier4.md`)
- **`/ant-interpret` command** — Dream reviewer that loads dream sessions, investigates each observation against the actual codebase with evidence and verdicts (confirmed/partially confirmed/unconfirmed/refuted), assesses concern severity, estimates implementation scope, and facilitates discussion before injecting pheromones or adding TO-DOs. (`interpret.md`)
- **`/ant-dream` command** — Philosophical wanderer agent that reads codebase, git history, colony state, and TO-DOs, performs random exploration cycles and writes observations to `.aether/dreams/`. (`dream.md`)
- **`/ant-help` command** — Renamed from `/ant-ant` with updated content covering all 20 commands, session resume workflow, colony memory system, and full state file inventory. (`help.md`)
- **OpenCode command sync** — All `.claude/commands/ant/` prompts synced to `.opencode/commands/ant/` for cross-tool parity

### Changed
- **Checkpoint messaging** — Now suggests actual next command (e.g., `/ant-continue` or `/ant-build 3`) instead of generic `/ant-status`. Format: "safe to /clear, then run /ant-continue"
- **Caste emoji in spawn output** — Spawn-log and spawn-complete in `aether-utils.sh` show caste emoji adjacent to ant name (e.g., `🔨Chip-36`). Build.md SPAWN PLAN and Colony Work Tree use emoji-first format. (`aether-utils.sh`, `build.md`)
- **Phase context in command suggestions** — Next Steps sections now include phase names alongside numbers (e.g., `/ant-build 3   Phase 3: Add Authentication`). (`status.md`, `plan.md`, `phase.md`)
- **OpenCode plan.md** — Now dynamically calculates first incomplete phase instead of hardcoding Phase 1. (`plan.md`)

### Fixed
- **Output appears before agents finish** — `build.md` now enforces blocking behavior; Steps 5.2, 5.4.1, and 5.6 wait for all TaskOutput calls before proceeding
- **Command suggestions use real phase numbers** — `status.md`, `continue.md`, `plan.md`, and `phase.md` calculate actual phase numbers instead of showing template placeholders
- **Progressive disclosure UI** — Compact-by-default output with `--verbose` flag; `status.md` (8-10 lines) and `build.md` (12 lines) default to compact mode

## [1.0.0] - 2026-02-09

### First Stable Release

Aether Colony is a multi-agent system using ant colony intelligence for Claude Code and OpenCode. Workers self-organize via pheromone signals to complete complex tasks autonomously.

### Added
- **20 ant commands** for autonomous project planning, building, and management (`ant:init`, `ant:plan`, `ant:build`, `ant:continue`, `ant:status`, `ant:phase`, `ant:colonize`, `ant:watch`, `ant:flag`, `ant:flags`, `ant:focus`, `ant:redirect`, `ant:feedback`, `ant:pause-colony`, `ant:resume-colony`, `ant:organize`, `ant:council`, `ant:swarm`, `ant:ant`, `ant:migrate-state`)
- **Multi-agent emergence** — Queen spawns workers directly; workers can spawn sub-workers up to depth 3
- **Pheromone signals** — FOCUS, REDIRECT, and FEEDBACK with TTL-based filtering
- **Project flags** — Blockers, issues, and notes with auto-resolve triggers
- **State persistence** — v3.0 consolidated `COLONY_STATE.json` with session handoff via pause/resume
- **Command output styling** — Emoji sandwich styling across all ant commands
- **Git checkpoint/rollback** — Automatic commits before each phase for safety
- **`aether-utils.sh` utility layer** — Single entry point for deterministic colony operations (error tracking, activity logging, spawn management, flag system, antipattern checks, autofix checkpoints)
- **OpenCode compatibility** — Full command mirror in `.opencode/commands/ant/`

### Architecture
- Queen ant orchestrates via pheromone signals
- Worker castes: Builder, Scout, Watcher, Architect, Route-Setter
- Wave-based parallel spawning with dependency analysis
- Independent Watcher verification with execution checks
- Consolidated `workers.md` for all caste disciplines

## [Pre-1.0] - 2026-02-01 to 2026-02-08

Development releases (versions 2.0.0-2.4.2) building toward stable release. Key milestones:

### 2026-02-08
- **v2.0 nested spawning** — Direct Queen spawning, enforcement gates, flagging system
- **OpenCode cross-tool compatibility** — Commands available in both Claude Code and OpenCode
- **ant:swarm** — Parallel scout investigation for stubborn bugs
- **ant:council** — Multi-choice intent clarification

### 2026-02-07
- **True emergence system** — Worker-spawns-worker architecture
- **Verification gates** — Worker disciplines enforced
- **v1.0.0 release prep** — Auto-upgrade from old state formats

### 2026-02-06
- **State consolidation (v2.0 → v3.0)** — 5 state files merged into single `COLONY_STATE.json`
- **State migration command** — `ant:migrate-state` for upgrading existing colonies
- **Signal schema unification** — TTL-based signal filtering replacing decay system
- **Command trim** — Reduced `status.md` from 308 to 65 lines, signal commands to 36 lines each, `aether-utils.sh` from 317 to 85 lines (later expanded with new features)
- **Worker spec consolidation** — 6 separate worker specs merged into single `workers.md`
- **Build/continue rewrite** — Minimal state writes, detection and reconciliation pattern

### 2026-02-05
- **NPM distribution** — Global install via `npm install -g`
- **Global learning system** — `learning-promote` and `learning-inject` for cross-project knowledge
- **Queen-mediated spawn tree** — Depth-limited spawning with tree visualization
- **ant:organize** — Codebase hygiene scanning (report-only)
- **Debugger spawn on retry failure** — Automatic debugging assistance
- **Multi-colonizer synthesis** — Disagreement flagging during analysis
- **Multi-dimensional watcher scoring** — Richer verification rubrics

### 2026-02-04
- **Auto-continue mode** — `--all` flag for `/ant-continue`
- **Safe-to-clear messaging** — State persistence indicators on all commands
- **Conflict prevention** — File overlap validation between parallel workers
- **Phase-aware error tracking** — Error-add wired to phase numbers

### 2026-02-01 to 2026-02-03
- **Initial AETHER system** — Autonomous agent spawning core
- **Queen Ant Colony** — Phased autonomy with pheromone-based guidance
- **Pheromone communication** — FOCUS, REDIRECT, FEEDBACK emission commands with worker response
- **Triple-Layer Memory** — Working memory, short-term compression, long-term patterns
- **State machine orchestration** — Transition validation with checkpointing
- **Voting-based verification** — Belief calibration for quality assessment
- **Semantic communication layer** — 10-100x bandwidth reduction
- **Error logging and pattern flagging** — Recurring issue detection
- **Claude-native prompts** — All commands converted from scripts to prompt-based system

- 2026-02-11: README.md — Major update reflecting all new features: 22 commands (was 20), dream/interpret commands, colony memory inheritance, graveyards, auto-recovery headers, git safety, lint suite, CLAUDE.md-aware command detection, Colony Memory section, restructured Features section
- 2026-02-11: .aether/data/review-2026-02-11.md — Comprehensive daily review report covering 3 colony sessions, 10 achievements, 3 regressions, 5 concerns, 3 debunked concerns, and prioritized recommendations
- 2026-02-12: README.md, CHANGELOG.md — Added /ant-chaos (resilience testing) and /ant-archaeology (git history analysis) commands with build pipeline integration
- 2026-02-12: CHANGELOG.md — added repo-local path migration entry
- 2026-02-12: README.md — Updated to describe repo-local .aether/ architecture; removed global ~/.aether/ runtime references, restructured File Structure section with repo-local paths primary
- 2026-02-13: bin/cli.js, update.md — Added orphan cleanup (syncDirWithCleanup), git dirty-file detection with --force stash, --dry-run preview, hub manifest generation

---

## Colony Work Log

The following entries are automatically generated by the colony during work phases.


## 2026-02-21

### Phase 37 — Plan 02

- **Files:** `aether-utils.sh`, `CHANGELOG.md`
- **Decisions:** Created changelog-append function; Added changelog-collect-plan-data helper
- **What Worked:** Function works correctly
- **Requirements:** LOG-01 addressed

### Phase 37 — Plan 99

- **Files:** `test.md`
- **Decisions:** Test
- **What Worked:** Works
- **Requirements:** TEST-01 addressed

### Phase 0 — Plan 

## 2026-03-20

### Phase 0 — Plan 01

- **Files:** `.aether/aether-utils.sh`, `tests/bash/test-hive-init.sh`, `tests/bash/test-hive-read.sh`, `tests/bash/test-hive-integration.sh`, `tests/integration/hive-store.test.js`, `tests/unit/colony-state.test.js`
- **Decisions:** Hub-level lock isolation for shared files; Reuse pheromone-write sanitization pattern for user input
- **What Worked:** hive-init creates wisdom.json; hive-store adds/merges with dedup; hive-read filters by domain; Cross-repo lock fix

### Phase 0 — Plan 01

- **Files:** `.aether/aether-utils.sh`, `tests/bash/test-hive-abstract.sh`, `tests/bash/test-hive-promote.sh`, `tests/bash/test-hive-abstraction-pipeline.sh`
- **Decisions:** Orchestrator delegation via bash $0; Path stripping regex for abstraction
- **What Worked:** hive-abstract generalizes text; hive-promote orchestrates pipeline; Multi-repo merge works

### Phase 04 — Plan 01

- **Files:** `.claude/commands/ant/seal.md`, `.opencode/commands/ant/seal.md`, `tests/bash/test-seal-hive-promotion.sh`
- **Decisions:** Use --text and --source-repo for hive-promote API (not --instinct)
- **What Worked:** Archaeology pre-build scan catches API mismatches; Hive promotion non-blocking in seal ceremony

### Phase 05 — Plan 01

- **Files:** `.aether/aether-utils.sh`, `tests/bash/test-hive-confidence-boost.sh`
- **Decisions:** Confidence boosts at 2/3/4+ repo thresholds using max() for never-downgrade
- **What Worked:** Atomic jq pipelines preserve lock safety; awk for float math follows codebase precedent

### Phase 06 — Plan 01

- **Files:** `CLAUDE.md`
- **Decisions:** Document all Hive Brain features in CLAUDE.md
- **What Worked:** Chaos testing docs against code reveals accuracy gaps

### Phase 0 — Plan 00

- **Files:** `.aether/aether-utils.sh`, `CLAUDE.md`, `.claude/commands/ant/seal.md`, `.opencode/commands/ant/seal.md`
- **Decisions:** Colony sealed at Crowned Anthill; Build the Hive Brain — cross-colony wisdom intelligence layer
- **What Worked:** 6 phases completed; 16 wisdom proposals promoted to QUEEN.md; Hive Brain fully operational with domain-scoped retrieval and multi-repo confidence boosting

## 2026-03-21

### Phase 0 — Plan 01

- **Files:** `.aether/aether-utils.sh`, `.claude/commands/ant/seal.md`, `.opencode/commands/ant/seal.md`
- **Decisions:** Fix when-when stutter across 4 locations; Fix seal.md jq path .instincts to .memory.instincts; Add instinct-apply subcommand; Fix instinct-create locking bug
- **What Worked:** Archaeologist pre-scan catches latent bugs; Trap-based locking prevents deadlocks; 549 tests pass

### Phase 0 — Plan 01

- **Files:** `.aether/aether-utils.sh`
- **Decisions:** Add midden-review subcommand; Add midden-acknowledge subcommand
- **What Worked:** Midden feedback loop closed; 6 tests pass; Trap-based locking for acknowledge

### Phase 0 — Plan 01

- **Files:** `.aether/aether-utils.sh`, `.aether/utils/hive.sh`, `.aether/utils/midden.sh`
- **Decisions:** Extract hive-* subcommands to hive.sh; Extract midden-* subcommands to midden.sh; Fix limit arg bug in midden-recent-failures
- **What Worked:** 12004 to 11221 lines (-783); Source-and-delegate pattern proven; 576 tests pass

### Phase 0 — Plan 01

- **Files:** `TO-DOS.md`, `README.md`, `CHANGELOG.md`, `CLAUDE.md`
- **Decisions:** Audit TO-DOS (11 completed, 10 pending); Update README to v2.0.0; Write CHANGELOG [2.0.0]; Update CLAUDE.md counts
- **What Worked:** All docs reflect v2.0.0 reality; No stale version references

### Phase 0 — Plan 01

- **Files:** `package.json`, `.aether/version.json`
- **Decisions:** Bump version to 2.0.0; Push to hub
- **What Worked:** 542 tests pass; Package validated; Hub install verified

### Phase 0 — Plan 00

- **Files:** `.aether/aether-utils.sh`, `.aether/utils/hive.sh`, `.aether/utils/midden.sh`, `CLAUDE.md`, `README.md`, `package.json`
- **Decisions:** Colony sealed at Crowned Anthill; Ship-ready Aether v2
- **What Worked:** 5 phases completed; 22 wisdom proposals promoted; v2.0.0 on hub

## 2026-03-22

### Phase 0 — Plan 00

- **Files:** `.aether/utils/skills.sh`, `.aether/skills/`, `bin/cli.js`, `.claude/commands/ant/skill-create.md`
- **Decisions:** Colony sealed at Crowned Anthill; Implement Aether Skills Layer — smart-matched colony and domain skills for workers
- **What Worked:** 5 phases completed; 28 skills created (10 colony + 18 domain); skills.sh engine (502 lines); Build pipeline skill injection; Hub distribution with manifest protection; Colony wisdom promoted to QUEEN.md

## 2026-03-23

### Phase 0 — Plan 00

- **Files:** `.aether/utils/oracle/oracle.sh`, `.aether/.npmignore`, `bin/cli.js`, `.claude/commands/ant/oracle.md`
- **Decisions:** Colony sealed at Crowned Anthill; Fix oracle.sh distribution to installed repos
- **What Worked:** 4 phases completed; Oracle tmux launch path fixed; Hub state leak prevented

## 2026-03-27

### Phase 0 — Plan 00

- **Files:** `colony-state.template.json`, `colony-state-reset.jq.template`, `seal.md`, `entomb.md`, `emoji-audit.sh`, `colony-visuals/SKILL.md`, `aether-utils.sh`, `CLAUDE.md`
- **Decisions:** Colony sealed at Crowned Anthill v1; Add versioning to seal/entomb lifecycle and enforce consistent emoji usage
- **What Worked:** 5 phases completed; Colony wisdom promoted to QUEEN.md; 24 new tests added

### Phase 0 — Plan 00

- **Files:** `queen.sh`, `aether-utils.sh`, `midden.sh`, `aether-sage.md`, `test-queen-charter.test.sh`, `test-midden-bridge.sh`, `test-aether-utils.sh`
- **Decisions:** Colony sealed at Crowned Anthill v2; Fix seal ceremony audit issues
- **What Worked:** 5 phases completed; 4 instincts promoted to hive; Colony wisdom promoted to QUEEN.md

### Phase 0 — Plan 00

- **Files:** `init.md`, `seal.md`, `build-prep.md`, `build-verify.md`, `build-wave.md`, `build-context.md`, `continue-gates.md`, `continue-verify.md`, `colony-visuals/SKILL.md`
- **Decisions:** Colony sealed at Crowned Anthill; Enforce consistent visual styling across all Aether commands
- **What Worked:** 5 phases completed; Banners standardized; Spawn announcements unified; Emoji map expanded; Progress bars added

## 2026-03-28

### Phase 0 — Plan 01

- **Files:** `tests/bash/test-aether-utils.sh`
- **Decisions:** Replace hardcoded dates with dynamic cross-platform computation
- **What Worked:** Dynamic dates prevent time-based test degradation

### Phase 2 — Plan 01

- **Files:** `queen.sh`
- **Decisions:** Fix trap composition, JSON escaping, local declarations in queen.sh
- **What Worked:** Verification passed with 616 tests; Auditor found 3 HIGH pre-existing issues fixed

### Phase 0 — Plan 01

- **Files:** `queen.sh`
- **Decisions:** Replace sed c-command with head/tail; Add empty-file safety guards; Fix sources and priming flags
- **What Worked:** head/tail proven safer than sed c on macOS; empty-file guard prevents data destruction

## 2026-03-29

### Phase 4 — Plan 01

- **Files:** `init.md`, `seal.md`, `entomb.md`
- **Decisions:** Lifecycle commands handle colony_version via template system; Command parity maintained across Claude and OpenCode
- **What Worked:** Verification of init/seal/entomb colony_version handling; Cross-platform seal commit synthesis and push prompts confirmed

### Phase 0 — Plan 06

- **Files:** `spawn-tree.sh`, `spawn.sh`, `spawn-tree.test.js`
- **Decisions:** Replace O(n^2) bash loops with single-pass awk; Add jq validation to wrapper functions; Update test fixtures to 7-field format
- **What Worked:** awk single-pass parsing eliminates 4000+ subprocess forks; Test fixtures now match production format
- **Requirements:** spawn-tree.sh, spawn.sh, spawn-tree.test.js addressed

### Phase 7 — Plan 01

- **Files:** `.aether/data/AUDIT-REPORT.md`
- **Decisions:** audit-report-corrected

### Phase 0 — Plan 00

- **Files:** `COLONY_STATE.json`, `QUEEN.md`, `learning.sh`
- **Decisions:** Colony sealed at Crowned Anthill; Comprehensive audit colony
- **What Worked:** 6 phases completed; Colony wisdom promoted to QUEEN.md

## 2026-03-30

### Phase 0 — Plan 00

- **Files:** `aether-utils.sh`, `utils/immune.sh`, `utils/council.sh`, `utils/midden.sh`, `utils/session.sh`, `utils/state-api.sh`
- **Decisions:** Colony sealed at Crowned Anthill; Implement next-gen Aether features: immune response, headless autopilot, vital signs, quick scout, council expansion, midden library
- **What Worked:** 6 phases completed; Colony wisdom promoted to QUEEN.md

### Phase 1 — Plan 01

- **Files:** `.aether/utils/spawn-tree.sh`, `tests/unit/spawn-tree.test.js`
- **Decisions:** gsub order is load-bearing for JSON escaping
- **What Worked:** awk gsub escaping with correct order; TDD with 4 new tests
- **Requirements:** spawn-tree.sh, spawn-tree.test.js addressed

### Phase 2 — Plan 01

- **Files:** `.aether/utils/queen.sh`, `tests/bash/test-queen-module.sh`
- **Decisions:** use ENVIRON[] not awk -v for user content
- **What Worked:** ENVIRON-based awk approach; head/tail for multi-line replacement; orphan cleanup
- **Requirements:** .aether/utils/queen.sh addressed

### Phase 3 — Plan 01

- **Files:** `.aether/utils/error-handler.sh`, `.aether/utils/spawn.sh`, `.aether/aether-utils.sh`
- **Decisions:** guard central subcommand plus individual sites
- **What Worked:** AETHER_TESTING env guard
- **Requirements:** error-handler.sh, spawn.sh, aether-utils.sh addressed

### Phase 4 — Plan 01

- **Files:** `package.json`, `package-lock.json`
- **Decisions:** npm overrides for transitive deps
- **What Worked:** minimatch; path-to-regexp; picomatch; tar; brace-expansion; diff
- **Requirements:** package.json addressed

### Phase 5 — Plan 01

- **Decisions:** final verification sweep confirms all fixes
- **What Worked:** midden acknowledgment; full test suite verification

### Phase 0 — Plan 00

- **Files:** `.aether/utils/spawn-tree.sh`, `.aether/utils/queen.sh`, `.aether/utils/error-handler.sh`, `.aether/utils/spawn.sh`, `.aether/aether-utils.sh`, `package.json`
- **Decisions:** Colony sealed at Crowned Anthill; Fix critical midden entries and harden infrastructure
- **What Worked:** 5 phases completed; Colony wisdom promoted to QUEEN.md

### Phase 1 — Plan 01

- **Files:** `.aether/aether-utils.sh`
- **Decisions:** context-update now fully jq-safe
- **What Worked:** 1 remaining raw json_ok fixed
- **Requirements:** .aether/aether-utils.sh addressed

### Phase 03 — Plan 01

- **Files:** `.aether/aether-utils.sh`
- **Decisions:** Use jq -nc --arg for all json_ok calls; parallel builder verification catches fabricated completions

### Phase 2 — Plan 01

- **Files:** `aether-utils.sh`, `package.json`
- **Decisions:** Use jq --arg for all json_ok sites with user strings
- **What Worked:** jq --arg escaping; empty-file guard in validate-state
- **Requirements:** json_ok safe escaping addressed

### Phase 3 — Plan 01

- **Files:** `.aether/aether-utils.sh`, `.aether/utils/flag.sh`, `tests/bash/test-flag-module.sh`, `tests/bash/test-state-checkpoint.sh`
- **Decisions:** Fixed view-state jq filter injection; Fixed fallback json_err escaping; Converted 14+ json_ok sites to jq --arg
- **What Worked:** All 509 tests pass; Auditor score 73/100; No critical security issues

### Phase 05 — Plan 01

- **Files:** `.aether/utils/hive.sh`, `tests/bash/test-hive-read.sh`, `tests/bash/test-learning-recovery.sh`
- **Decisions:** Compose null fallback with tonumber to preserve prior type coercion fix
- **What Worked:** Archaeology pre-build scan prevented regression; Stale grep targets identified by root cause analysis

### Phase 0 — Plan 00

- **Files:** `aether-utils.sh`, `utils/hive.sh`, `utils/learning.sh`, `tests/`
- **Decisions:** Colony sealed at Crowned Anthill; hardened ~40 json_ok sites + checkpointing + hive null safety
- **What Worked:** 5 phases completed; 9 instincts created; 4 hive-eligible

## 2026-03-31

### Phase 0 — Plan 01

- **Files:** `build-complete.md`, `build.yaml`, `build.md`
- **Decisions:** Add Stage Audit Gate to build orchestrators
- **What Worked:** Pre-synthesis verification gate ensures all stages complete

### Phase 3 — Plan 01

- **What Worked:** lint:sync clean; lint clean; 524 tests pass

### Phase 0 — Plan 00

- **Files:** `build-complete.md`, `build.yaml`, `update-transaction.test.js`
- **Decisions:** Colony sealed at Crowned Anthill; Enforce non-skippable build playbook execution and verify exchange fix
- **What Worked:** 3 phases completed; Colony wisdom promoted to QUEEN.md

### Phase 1 — Plan 01

- **Files:** `build-complete.md`, `state-contract-design.md`
- **Decisions:** Deleted wrong test file; Fixed step numbering; Added DATA_DIR exception clause
- **What Worked:** Swarm parallel audit; Pre-existing quality issues logged

## 2026-04-01

### Phase 2 — Plan 01

- **Files:** `trust-scoring.sh`, `event-bus.sh`, `aether-utils.sh`
- **Decisions:** Stateless calculation module; JSONL event bus with file locking
- **What Worked:** Parallel builders; Self-registration pattern

### Phase 3 — Plan 01

- **Files:** `instinct-store.sh`, `graph.sh`, `learning.sh`, `aether-utils.sh`
- **Decisions:** Standalone instinct storage; jq graph traversal; Trust-scored observations
- **What Worked:** Additive parallel modification

### Phase 4 — Plan 01

- **Files:** `nurse.sh`, `herald.sh`, `librarian.sh`, `critic.sh`, `sentinel.sh`, `janitor.sh`, `archivist.sh`, `scribe.sh`, `orchestrator.sh`
- **Decisions:** 8 curation ants; curation-run orchestrator; Sentinel-first execution order
- **What Worked:** Parallel core/ops builders; Orchestrator integration

### Phase 5 — Plan 01

- **Files:** `consolidation.sh`, `consolidation-seal.sh`, `test-e2e-pipeline.sh`
- **Decisions:** Lightweight phase-end; Full seal consolidation; E2E integration test

### Phase 6 — Plan 01

- **Files:** `structural-learning-stack.md`, `CLAUDE.md`
- **Decisions:** Architecture documentation; Updated component counts; Full test sweep

### Phase 0 — Plan 00

- **Files:** `trust-scoring.sh`, `event-bus.sh`, `instinct-store.sh`, `graph.sh`, `consolidation.sh`, `curation-ants`
- **Decisions:** Colony sealed at Crowned Anthill; Structural Learning Stack complete
- **What Worked:** 6 phases completed; Colony wisdom promoted to QUEEN.md

### Phase 0 — Plan 01

- **Files:** `go.mod`, `pkg/storage/storage.go`, `pkg/storage/storage_test.go`
- **Decisions:** Go module at github.com/aether-colony/aether; atomic writes via temp+rename; per-path RWMutex for concurrent safety
- **What Worked:** Parallel builders for independent packages; TDD with race detector
- **Requirements:** go build, test, vet pass;91.2% coverage;524 npm tests unaffected addressed

### Phase 0 — Plan 02

- **Files:** `.aether/utils/trust-scoring.sh`, `.aether/utils/event-bus.sh`

### Phase 3 — Plan 03

- **Files:** `learning.sh`, `instinct-store.sh`, `graph.sh`, `test-instinct-store.sh`
- **Decisions:** Standalone instinct storage; Backward-compatible trust score migration; jq graph layer for instinct relationships
- **What Worked:** Trust score integration with learning-observe; Full instinct schema with provenance; Graph link/neighbors/reach/cluster

### Phase 0 — Plan 00

- **Files:** `learning.sh`, `instinct-store.sh`, `graph.sh`, `test-instinct-store.sh`, `trust-scoring.sh`, `event-bus.sh`
- **Decisions:** Colony sealed at Crowned Anthill; Structural Learning Stack verified
- **What Worked:** 3 phases completed; Colony wisdom promoted to QUEEN.md

- [2026-04-05] Phase 01, Plan 01: Implement critical Go commands: init, install, setup. All tests passing (11/11 packages).

- [2026-04-05] Phase 02, Plan 01: Port remaining shell commands to Go: all 141 subcommands now have Go equivalents, build lifecycle restructured as context-update subcommands, binary download added to install
## [2026-04-05] - Phase 3: Update Commands and Remove GSD

### Changed
- Updated 4 slash commands (init, lay-eggs, oracle, resume) to use Go binary
- Updated CLAUDE.md for Go-only architecture

### Removed
- Deleted 168 GSD system files (commands, agents, hooks, manifests, workflows, templates)
- Removed .claude/get-shit-done/ directory
- Removed .planning/ directory with all milestone/phase files

### Files
- 444 files changed, ~85,000 lines removed


- [2026-04-06] Phase 1-exec-timeouts, Plan 01: Phase 1 (Exec Command Timeouts): Converted all 23 bare exec.Command calls to exec.CommandContext with timeout constants (GeneralTimeout 30s, GitTimeout 60s, BuildTimeout 120s). Added timeout tests proving context cancellation kills subprocesses. Files: cmd/timeouts.go, cmd/clash.go, cmd/worktree_merge.go, cmd/context.go, cmd/session_cmds.go, pkg/storage/paths.go

- [2026-04-06] Phase 2-file-locking, Plan 01: feat(storage): add cross-process file locking with syscall.Flock (pkg/storage/lock.go, lock_test.go, concurrent_test.go, store_locking_test.go; integrated into Store; removed dead sync.Map mutexes)

- [2026-04-06] Phase 4, Plan 01: feat(cmd): add configurable --timeout flag to eventbus commands, replacing hardcoded 5s values

- [2026-04-06] Phase 5, Plan 01: Integration Verification: all 5 audit findings verified resolved — race detector clean, go vet clean, binary builds

- [2026-04-06] Phase seal, Plan 00: Colony sealed at Crowned Anthill — all 5 audit findings resolved, 5 phases completed

- [2026-04-06] Phase 2, Plan 01: Fix XML archive export: load actual colony data from store/hub for archive exports, add positional arg backward compat to colony-archive-xml. Files: cmd/exchange.go, cmd/alias_cmds.go, cmd/alias_cmds_test.go

- [2026-04-06] Phase 3, Plan 01: Add seal/milestone/pause/contextual commit types to generate-commit-message

- [2026-04-06] Phase 4, Plan 01: Fix learning-check-promotion --all flag and learning-promote-auto stub: connected stub to PromoteService.Promote(), added batch mode for check-promotion

- [2026-04-06] Phase 4, Plan 01: fix(learning): Add --all flag to check-promotion and wire promote-auto to PromoteService

- [2026-04-06] Phase 4, Plan 01: Fix learning-check-promotion --all and learning-promote-auto stub: Added --all flag for batch eligibility checks, wired promote-auto to PromoteService.Promote() for actual instinct creation

- [2026-04-07] Phase 1, Plan 01: Fix state-write positional arg rejection: Changed cobra.NoArgs to cobra.MaximumNArgs(1), added JSON validation and flag mutual exclusion, added TestStateWritePositionalArg

- [2026-04-07] Phase 03, Plan 01: Fix state-mutate bracket notation in expression parser — widened reFieldSet regex, added normalizeBracketPath

- [2026-04-07] Phase 04, Plan 01: Deprecate dead code (worktree_merge.go, suggest.go) and add silent error logging across 6 active files. -1049 net lines removed, all tests pass.

- [2026-04-07] Phase seal, Plan 00: Colony sealed at Crowned Anthill v2 — 5 phases completed: state-write positional arg fix, pheromone dedup, state-mutate bracket notation, dead code deprecation + error logging, integration verification

- [2026-04-07] Phase 5, Plan 05: contextual(phase-05): Replace aether-utils.sh references in 38 documentation, exchange XML, Go source comments, and testdata files with contextual Go binary references

- [2026-04-07] Phase 06-shell-deprecation, Plan 01: Phase 6: Updated weekly-audit.sh to use Go binary, added DEPRECATED headers to 41 dead shell scripts, ran final grep verification and Go test suite. All phases complete — Crowned Anthill milestone reached.

- [2026-04-08] Phase 01, Plan 01: Performance Baseline Benchmarks: PRContext ~20.8ms/1989 allocs, ColonyPrime ~266us/921 allocs

- [2026-04-09] Phase seal, Plan 00: Colony sealed at Crowned Anthill;Tackle the three speed fixes for pheromone injection: eliminate redundant process spawns, centralize validation, and add session-level caching;9 phases completed;Colony wisdom promoted to QUEEN.md

- [2026-04-12] Phase 10-comprehensive-verification, Plan 01: verification-only: 89 broken CLI calls found across .claude/.opencode dirs, 1207/1207 Go tests pass

- [2026-04-12] Phase final-cleanup, Plan 11: known-issues.md, pheromones.json, COLONY_STATE.json: Updated documentation, cleaned stale data, verified integrity

- [2026-04-12] Phase seal, Plan 00: Colony sealed at Crowned Anthill; Fix all ~120 broken CLI calls in markdown; 11 phases completed; Colony wisdom promoted to QUEEN.md

- [2026-04-14] Phase 1-codex-format-validation, Plan 1: .aether/docs/codex-format-spec.md: Validated AGENTS.md schema, tool names, skill discovery paths from Codex source

- [2026-04-14] Phase 3-agents-md-project-file, Plan 1: AGENTS.md, .codex/CODEX.md: Created Codex CLI project file and developer guide

- [2026-04-14] Phase create-hub-packaging-mirror, Plan 5: .aether/skills-codex/ (28 files), .aether/agents-codex/ (24 files), .aether/agents-claude/aether-route-setter.md: Created Codex packaging mirrors

- [2026-04-14] Phase 6-update-go-install, Plan 01: install_cmd.go, install_cmd_test.go: Added Codex sync pair to install command with hub mirror support

- [2026-04-14] Phase 7-update-setup-update-commands, Plan 01: cmd/setup_cmd.go, cmd/update_cmd.go, cmd/setup_cmd_test.go, cmd/update_cmd_test.go, .aether/templates/agents-md-template.md: Added Codex sync pairs to setup and update commands, AGENTS.md template generation

- [2026-04-14] Phase codex-integration, Plan 08: codex_e2e_test.go,package.json: Added 8 E2E tests for Codex sync lifecycle, added .codex/ to npm package files

- [2026-04-15] Phase 09-documentation-updates, Plan 01: CLAUDE.md, aether-colony.md, source-of-truth-map.md updated with Codex CLI platform support

- [2026-04-15] Phase codex-integration, Plan 10: Full Verification: all 2712 tests pass, file parity across Claude/OpenCode/Codex, install-setup-update cycle verified, npm pack correct

- [2026-04-16] Phase seal, Plan 00: Colony sealed at Crowned Anthill;Make Aether work for Codex CLI — research Codex's extension format, create command and agent files, update setup flow, and test end-to-end;10 phases completed;Colony wisdom promoted to QUEEN.md
