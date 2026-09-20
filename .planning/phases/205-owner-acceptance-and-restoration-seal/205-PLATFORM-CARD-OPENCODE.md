# OpenCode — Platform Honesty Card

OpenCode is a second AI coding assistant (an alternative to Claude Code) that this program can also run inside. The owner decided that for this milestone, only Claude Code has to work — OpenCode and Codex (below) get an honest record of what a real run actually did, with no extra engineering effort spent making them work better. This card describes one real, non-interactive run of OpenCode's own command line, made in a disposable throwaway copy of this project so the real checkout was never touched. Every line below is backed by the exact captured output in `evidence/platform-runs/opencode-run.txt`.

## What worked

- OpenCode's command-line entry point (`opencode run "/ant-status"`, OpenCode CLI version 1.1.63) started, understood the request, and picked its configured "build" helper's default model before trying to reach it. [transcript: opencode-run.txt:8 | "> build · minimax-2.5"]

## What stopped early with a truthful message

- The run then stopped early with a plain, readable error instead of hanging past its time limit or silently pretending to have worked: it could not connect to the model it had just picked. [transcript: opencode-run.txt:11 | "Error: Error: Unable to connect. Is the computer able to access the url?"]

## What is not supported

- Not demonstrated by this run: because OpenCode routes even a status request through a model conversation turn first, and that turn could not connect, this capture never reaches the deeper step of actually calling this program's own status command underneath the OpenCode command. Whether that deeper step works is not shown one way or the other by this run. [transcript: opencode-run.txt:8 | "> build · minimax-2.5"]
- Not demonstrated by this run: no further OpenCode output — no status screen, no error banner from this program itself, no state change — was produced beyond the one connection error captured here. [transcript: opencode-run.txt:11 | "Error: Error: Unable to connect. Is the computer able to access the url?"]
