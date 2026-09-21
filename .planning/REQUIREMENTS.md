# Requirements: Aether v1.29 Use It Every Day

**Defined:** 2026-09-21

**Status:** Approved by the owner 2026-09-21 — [decision](decisions/2026-09-21-v1.29-use-it-every-day.md)

**One rule:** no new features and no new strict rules.

**Previous milestone requirements:** [v1.28](milestones/v1.28-REQUIREMENTS.md) (204.2-204.5 parked, 205 superseded)

**Core value:** Aether should feel alive and truthful at runtime, not only look clever in wrappers or tests.

## Milestone outcome

The owner can take a real piece of work from start to finish in each of his real projects without
hitting a dead end and without pasting a bug into the Aether chat.

For dummies: stop polishing the engine on the bench. Drive the car on real roads, fix only what
breaks, and make sure that when something does break it says how to get going again.

Every requirement below is satisfied only by a command someone can run that fails when the
requirement is unmet (CLAUDE.md, Definition of Done). "Proof" names that command or test.

## Requirements

### Screens reach the owner (Phase 206)

- [ ] **UED-01** — Every menu command that draws a screen tells the chat to show it unchanged. Proof: `TestEveryWrapperThatDrawsAScreenRelaysIt`.
- [ ] **UED-02** — In a chat where a menu command was used, a reply that hides the screen is sent back once, never twice, and the check changes nothing on disk. Proof: `TestStopHookSendsBackAReplyThatHidTheScreen`, `TestStopHookNeverBlocksTwice`, `TestStopHookScreenCheckDoesNotMutate`.
- [ ] **UED-03** — A screen printed for a chat carries no colour escape codes. Proof: `TestPipedVisualOutputCarriesNoEscapeCodes`.
- [ ] **UED-04** — Start-up, code survey and helper cards are in the guarded look. Proof: `TestEveryOrdinaryScreenIsMeasuredForVoice` with eleven families.
- [ ] **UED-05** — The program hands its screen to Claude Code to show directly (hook `systemMessage`), and the owner has judged it on his own screen. Proof: a real-chat run asserting the informational message carries the banner lines; owner sign-off recorded.
- [ ] **UED-06** — A permanent status line shows phase, task and next command. Proof: a test that the shipped settings register it and that its output comes from the shared what-next decision.

### A messy practice project gates releases (Phase 207)

- [ ] **UED-07** — A committed script builds the practice project with every trap listed in the roadmap. Proof: the script runs clean twice and a test asserts each trap exists.
- [ ] **UED-08** — The whole journey runs through a real chat with hooks and menu commands loaded, asserting on files and recorded commands, never wording; three trials; money and turn caps. Proof: the journey command itself.
- [ ] **UED-09** — The journey catches each of 2026-09-21's six blockers. Proof: with each fix reverted in turn, the journey fails at that step.

### Never a dead end (Phase 208)

- [ ] **UED-10** — Every refusal carries the one command that gets past it and says whether it protects against losing work. Proof: a test that fails on any refusal with an empty next command; the journey runs each printed command.
- [ ] **UED-11** — A refusal that does not protect against losing work is a warning that carries on. Proof: the sorted refusal list is a checked-in table and a test asserts the program's behaviour matches it.
- [ ] **UED-12** — A failed check adds tasks and carries on. Proof: a real-flow test.
- [ ] **UED-13** — No screen advises a command that has no menu version, and no invented word is unexplained; the failure log has a menu command. Proof: `TestSlashCommandGuidancePointsAtRealCommands` extended to program-command advice; voice plain-English test on the status warnings.
- [ ] **UED-14** — A failure whose text the safety filter rejects is still recorded in readable words. Proof: a test feeding a backtick-bearing failure and asserting the log names what failed.
- [ ] **UED-15** — One command writes a report bundle, and every refusal tells a chat to report it rather than patch Aether. Proof: a test on the bundle contents and on refusal text.

### A light default path (Phase 209)

- [ ] **UED-16** — One entry command; the program picks small or big from the size of the change, says why, and moves a job back up when "small" was wrong. Proof: real-flow tests for both routes and the escalation. Details need the owner's decisions first.
- [ ] **UED-17** — One planning pass by default; discuss, specification approval and deeper rounds are opt-in. Proof: a real-flow test from goal to built work with no extra steps.
- [ ] **UED-18** — About six menu commands visible by default, the rest behind an advanced setting. Proof: a test on the default help screen.

### Two-week freeze (Phase 210)

- [ ] **UED-19** — The owner completes one real piece of work start to finish in each real project without pasting a bug. Proof: the owner's own statement, recorded with dates and projects.
- [ ] **UED-20** — The stopping rule is applied honestly: blockers hit during the freeze are counted and the count is reported. Proof: the refusal/bug log for the fortnight.

## Out of scope

New features. New strict rules. Codex native work (parked). Republishing to npm or a GitHub
Release before Phase 210 ends.
