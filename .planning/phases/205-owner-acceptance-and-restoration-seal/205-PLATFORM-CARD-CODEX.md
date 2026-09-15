# Codex — Platform Honesty Card

Codex is OpenAI's own command-line coding assistant, a third AI platform this program can run inside. Codex has no menu commands (no short `/ant-...` shortcuts) — every colony operation has to be typed out as the program's own `aether` command, or described in plain language for Codex to type out itself. Like OpenCode, the owner decided Codex does not have to work this milestone; this card is an honest record of what a real run actually did, with no extra engineering effort spent on it. This describes one real, non-interactive run of Codex's own command line, made in a disposable throwaway copy of this project so the real checkout was never touched. Every line below is backed by the exact captured output in `evidence/platform-runs/codex-run.txt`.

## What worked

- Given a plain-language instruction with no short command name, Codex chose on its own to run this program's real status command directly, exactly as CODEX.md says it should -- no invented shortcut, no substitute path. [transcript: codex-run.txt:27 | "AETHER_OUTPUT_MODE=visual aether status"]
- This program's own status screen then rendered correctly and told the truth: no project exists yet in this throwaway copy, with a plain next step. [transcript: codex-run.txt:31 | "No colony initialized in this repo."]

## What stopped early with a truthful message

- This particular run did not stop early -- the command Codex ran completed and reported success rather than a truthful failure message. [transcript: codex-run.txt:28 | "succeeded in 0ms:"]

## What is not supported

- Codex genuinely has no menu-command system in this milestone: the model had to be told the exact program command in plain language rather than typing a short command name. [transcript: codex-run.txt:34 | "to start a colony."]
- Not demonstrated by this run: this was a single, already-safe-to-repeat status read; it does not show whether Codex can carry out a longer, multi-step operation such as a full build or a project check from start to finish. [transcript: codex-run.txt:21 | "show me its full output verbatim"]
