# Claude Code — Platform Honesty Card

Claude Code is the one platform the owner decided has to actually work this milestone. This card is the primary proof for that: one real, non-interactive run of Claude Code's own command line, running the program's real "show me everything about this project" command (`/ant-status`), made in a disposable throwaway copy of this project so the real checkout was never touched. Every line below is backed by the exact captured output in `evidence/platform-runs/claude-run.txt`.

## What worked

- Claude Code's `/ant-status` short command ran this program's real status check underneath it and displayed the program's true, correct state for this throwaway copy of the project. [transcript: claude-run.txt:11 | "No colony initialized in this repo."]
- Claude Code then turned that into a short, plain-English summary for the owner instead of leaving the raw technical readout to speak for itself, exactly as this program's own rules require. [transcript: claude-run.txt:18 | "No project has been started in this folder yet."]
- The whole run finished cleanly and quickly with a real answer, not a hang or a crash. [transcript: claude-run.txt:5 | "Exit status: 0"]

## What stopped early with a truthful message

- This particular run did not stop early -- the command completed and reported the program's real state rather than a truthful failure message. [transcript: claude-run.txt:5 | "Exit status: 0"]

## What is not supported

- Not demonstrated by this run: this covered only one read-only status check; it does not by itself show a longer, multi-step operation such as building or checking a project from start to finish. [transcript: claude-run.txt:3 | "/ant-status"]
- Not demonstrated by this run: because no project existed yet in this throwaway copy, this run only shows the honest "nothing started yet" screen, not what an in-progress project's status screen looks like. [transcript: claude-run.txt:14 | "to start a colony."]

## 2026-09-21 note

The second bullet above ("turned that into a short, plain-English summary
... instead of leaving the raw technical readout to speak for itself") was
accepted at the time as the correct behaviour. The owner reversed that
acceptance on 2026-09-21: the program's own drawn screen (banners, cards,
dashboards) must now be shown to the owner unchanged, not paraphrased into a
short summary, per release 1.0.88 ("the owner sees Aether's screens, every
time"). A short, plain-English summary is still owed for raw, undrawn
readouts and for the assistant's own added remarks after a shown screen --
never as a replacement for the screen itself.
