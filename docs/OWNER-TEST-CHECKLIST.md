# Testing Aether Yourself — v1.0.48

This is a hands-on check that Aether works the way it should. It takes about
15–20 minutes. You don't need to read any code.

**How to use this:** copy each grey block exactly, paste it into a terminal,
press Enter, and read what comes back. Each step tells you what "good" looks
like. If you see the word `FAIL`, or something clearly different from what's
described, stop and paste the output back to me.

---

## Step 0 — Get the new version

```
cd ~/repos/Aether
aether publish --channel stable --binary-dest "$HOME/.local/bin"
aether version
```

✅ **PASS** — the last line says `1.0.48`.
❌ **FAIL** — it still says `1.0.47` or older. Stop here; the update didn't take.

---

## Step 1 — The safety net still passes

```
cd ~/repos/Aether && make smoke
```

This installs Aether into a throwaway sandbox and checks four things: that
updating doesn't destroy your settings, that running it twice changes nothing,
that the planning helper starts, and that a build plan can be produced.

✅ **PASS** — the last line says `SMOKE PASS: all daily-driver gates green`.
❌ **FAIL** — anything containing `SMOKE FAIL`.

---

## Step 2 — Make a throwaway project

```
mkdir -p ~/aether-test && cd ~/aether-test && git init -q && aether update --force
```

✅ **PASS** — it finishes and mentions files copied.

> If this folder already has a `.claude/settings.json` with your own settings
> in it, even better — Step 3 checks those survived.

---

## Step 3 — Your own settings survived

```
cd ~/aether-test && cat .claude/settings.json
```

✅ **PASS** — Aether's hooks are there (lines mentioning `aether hook-...`),
**and** anything that was already in the file is still there too.
❌ **FAIL** — something you had before is gone.

*This is the bug that used to wipe your GSD setup. It should be impossible now.*

---

## Step 4 — Run the colony (in Claude Code, not the terminal)

Open Claude Code in `~/aether-test`, then type these one at a time, waiting for
each to finish:

```
/ant-init
```
Answer its questions about what you want to build. Pick something small and
real — "add a function that formats dates" is plenty.

```
/ant-plan
```

```
/ant-build 1
```
Workers will spawn and do actual work. Watch them go.

```
/ant-continue
```

```
/ant-seal
```

---

## What to look for while you do that

### A. The suggestions are commands you can actually type

After every command, Aether prints a **Next Up** box at the bottom.

✅ **PASS** — the suggestions start with a slash, like `/ant-continue` or
`/ant-build 2`.
❌ **FAIL** — a suggestion says ``Run `aether continue` `` — the raw form you'd
type in a terminal, not in chat.

> **Not a failure:** `aether publish`, `aether update`, `aether version`. Those
> genuinely are terminal commands and have no slash version, so they correctly
> appear without one.

### B. `/ant-continue` actually finishes — this is the big one

At v1.0.47 this step reliably got stuck. A reviewer worker would do its job
correctly but phrase its answer in a way Aether couldn't read, and the whole
phase would jam.

✅ **PASS** — it verifies the work and moves to the next phase (or clearly
explains what's blocking, in plain language).
❌ **FAIL** — you see `no worker claims found` or `parse worker output`.

**If this step works, the main thing this release fixed is working.**

### C. Any command it tells you to run, actually runs

If Aether ever says something like "Run `aether flag-resolve --id abc123` to
clear this", copy that exact command and run it in a terminal.

✅ **PASS** — it runs and does something.
❌ **FAIL** — `unknown command` or `unknown flag`.

*Aether used to suggest a command that didn't exist, at the exact moment you
were already stuck. There's now an automatic check that makes that impossible
to ship.*

### D. The workers make sense

Look at who spawns during `/ant-build`.

✅ **PASS** — the workers suit the job (builders, a probe, a watcher).
❌ **FAIL** — an **Ambassador** (external integrations) or **Gatekeeper**
(security) spawns on a phase with no external services or security work, then
reports it had nothing to do.

---

## Step 5 — The docs agree with reality

```
cd ~/repos/Aether && grep -c "1.0.4[1-7]" README.md CLAUDE.md AGENTS.md
```

✅ **PASS** — every line ends in `:0`.
❌ **FAIL** — any number above zero means a document still claims an old version.

---

## Step 6 (optional) — OpenCode

If you use OpenCode, repeat Step 4 in a fresh folder there. The same
slash-command suggestions apply (`/ant-continue`, not `aether continue`).

---

## Cleanup

```
rm -rf ~/aether-test
```

---

## If something fails

Tell me which step, and paste the output. Every check above maps to a specific
guard in the test suite, so a failure here points at exactly one place.
