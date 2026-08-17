# Aether vs GSD benchmark harness

This harness measures Aether — in two different ways of running it — against
GSD (a separate development tool), on the same real coding tasks. Every
comparison runs in a "hermetic" environment: a brand-new, throwaway home
directory (`$HOME`) containing only the one system being tested, with no
memory of past runs and no visibility into the operator's own machine setup.
That word — hermetic — means sealed off and isolated; it is used throughout
this document with that meaning, never with any other project's vocabulary.

## Prerequisites

The following command-line tools must be installed and reachable on `PATH`
before running anything in this harness:

- `go`
- `node`
- `npm`
- `jq`
- `git`
- the `claude` CLI (Claude Code's command-line tool)
- `gsd-sdk` (GSD's command-line helper)

One prerequisite is not obvious: **GSD must already be installed in the
operator's real home directory** before running this harness. GSD ships no
installer of its own — there is no `gsd install` command. Instead, this
harness creates a hermetic (isolated, throwaway) copy of GSD by copying an
existing, already-working GSD installation into the fresh home directory.
Four things are copied or made reachable this way:

- `~/.claude/get-shit-done/` — the GSD toolchain itself
- `~/.claude/agents/gsd-*.md` — GSD's agent definitions
- `~/.claude/skills/gsd-*` — GSD's skill definitions
- the `gsd-sdk` executable — made reachable on `PATH` inside the isolated
  environment, not copied, since it lives outside `~/.claude`

If GSD is not already installed on the operator's real machine, the GSD lane
of this harness cannot run.

## The first gate: hermetic smoke

Before any benchmark task is run, one command must pass. This is a hard
prerequisite: nothing else in this harness is trusted until it does.

```
bench/smoke-hermetic.sh
```

It runs five checks (gates), each of which stops the whole script with a
named reason on the first failure:

1. Aether boots successfully inside a brand-new, isolated home directory.
2. Aether completes one trivial task inside that isolated home directory.
3. GSD boots successfully inside a second, separate isolated home directory.
4. The `claude` command-line tool can sign in and respond inside both
   isolated home directories.
5. The two isolated home directories never leak into each other — each one
   contains only its own system's files.

**Exit behaviour:** the script exits with a non-zero status the moment any
gate fails, and it always prints the specific reason — it never reports a
pass it did not earn.

## Why isolation matters

Each system under test gets its own throwaway home directory so that neither
system can read the other's learned memory, prior conversation history, or
the operator's own personal configuration. Without this, a system that had
"seen" a task before (or that quietly relied on files sitting in the
operator's real setup) could look artificially good — the comparison would
no longer be fair.

One thing is deliberately **not** isolated: the Go programming language's
build cache. That cache is shared with the real machine on purpose. It holds
downloaded library code, not anything specific to Aether or GSD, so sharing
it gives neither system an advantage — it only saves time, since isolating it
too would force every single run to re-download the same dependencies from
scratch.

## Known environment risks

The hermetic smoke exists specifically to test three risks that could break
the whole harness. Their status below reflects an actual run of
`bench/smoke-hermetic.sh` on this machine (macOS), not a guess about what
should happen.

**macOS Keychain authentication for the `claude` CLI under an overridden home
directory — observed-broken.** Signing in to Claude Code normally stores the
session in the macOS Keychain (the operating system's secure credential
store), which is tied to the logged-in Mac user, not to the `$HOME`
environment variable. Overriding `$HOME` to point at a throwaway directory
does not carry that stored sign-in with it, so the `claude` command reports
"Not logged in" even though the operator is signed in on the real machine.
Signing in again from inside the throwaway directory does not work either,
because that requires opening a web browser and pasting back a one-time
code — a manual, human step that cannot be scripted. The fix is to provide
credentials a different way before running the harness: Anthropic supports
an API key (via the `ANTHROPIC_API_KEY` environment variable) as a sign-in
method that does not depend on `$HOME` or the Keychain at all. An operator
running this harness for real benchmark work needs to provision that key
first; see the failure message from gate 4 for the exact detail captured on
a real run.

**`os.homedir()` (a Node.js function used to find the home directory)
resolving to the wrong place instead of respecting the overridden `$HOME` —
observed-working.** On this machine, the harness's `hermetic_home_probe`
check (in `bench/lib/hermetic-home.sh`) confirmed that `os.homedir()` agrees
with the overridden `$HOME` in every isolated home directory created during
the smoke run. This risk did not materialize here, but the probe remains in
place and will print a loud warning if it ever does on a different machine.

**GSD's install being a plain file copy with no installer — observed-working.**
Copying an existing `~/.claude/get-shit-done` installation, its `gsd-*`
agents, and its `gsd-*` skills into a fresh isolated home directory worked
correctly on this machine: the copy produced a non-empty `VERSION` file and
more than the minimum expected number of agents and skills, and the `gsd-sdk`
command responded correctly once made reachable on `PATH` inside the isolated
environment.
