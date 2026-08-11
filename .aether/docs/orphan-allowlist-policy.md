# The "Nothing Calls It" Rule

A command that nothing ever calls does not ship. If a command exists, something
real has to run it — a menu item you can click, or code that actually invokes
it. If nothing runs it, the choice is simple: give it something that runs it,
or remove the command. This document explains the automatic check that
enforces that rule, and the two lists of already-existing exceptions it
watches.

## What the two lists are

Two automatic checks scan this project for commands nobody runs. Both found
commands that already existed before this rule was written — commands that
predate this discipline and haven't been fixed yet. Rather than pretend those
commands don't exist, each one is written down in a list, with a note saying
why it's being tolerated for now and which future piece of work is
responsible for cleaning it up.

- **The unused-command list** — every command found with no menu item and no
  code calling it, at the moment this rule went into effect.
- **The flag-check exceptions list** — a much smaller, second list for two
  commands that are typed into instructions as if they were real commands to
  run, but are actually just shorthand — the real commands they stand for are
  named differently. The check that looks for broken command instructions is
  told to skip these two, so it doesn't flag them by mistake.

## The one thing that is not allowed

Both lists can only get **shorter**. A name can never be added to either list.
If someone tries to add one — silently marking a new broken command as "fine,
ignore it" — the automated check that runs on every change fails immediately,
by name.

There is no override switch, no setting to turn the check off, and no
approval process that lets a reviewer wave a new entry through. The only way
either list changes is by editing the saved copy of the list directly — which
shows up as a visible change in the same review anyone would see for any other
edit. Nothing about it happens invisibly.

## Why it's built this way

Two shortcuts were deliberately rejected:

- **Just counting how many exceptions exist**, and failing if the count goes
  up. That looks safe, but it lets someone remove one tolerated command and
  quietly add a different one in its place — the same number, a different,
  worse problem. The check instead compares the exact set of names, not the
  count, so a straight swap still fails.
- **Comparing against the project's edit history** to see if a name is new.
  This project's automated checks run against a shallow copy of the history
  that doesn't always have everything to compare against, so a history-based
  check could pass or fail for reasons that have nothing to do with the list
  itself. The check instead compares against one saved, checked-in copy of
  the list — nothing else.

## The real numbers, as recorded today

The unused-command list currently holds **278 commands**. That is the honest,
full count found by the scan — not a smaller, rounder number that was
expected going in. Of those 278, **6** are tagged as belonging to a specific
future cleanup effort (the skill-related commands), because that effort's
finish line is defined as this exact set of 6 reaching zero. The remaining
272 are tracked as a wider backlog of pre-existing unused commands, not yet
assigned to any specific piece of work.

The flag-check exceptions list holds exactly 2 entries.

## Detail

The four files the checks read, and the automated tests that enforce the
rules above:

- `cmd/testdata/orphan_allowlist.json` — the live unused-command list (278
  entries, `name`/`reason`/`owner_phase` each).
- `cmd/testdata/orphan_allowlist_baseline.json` — the committed baseline
  `TestOrphanAllowlistOnlyShrinks` diffs the live list against.
- `cmd/cli_flag_audit_test.go` — holds the live flag-check exceptions list
  (`skipSubcommands`) and both tests that enforce this document:
  `TestFlagAuditSkipListOnlyShrinks` and
  `TestAllowlistPolicyNamesEveryGuardedFile`.
- `cmd/testdata/flag_audit_skiplist_baseline.json` — the committed baseline
  `TestFlagAuditSkipListOnlyShrinks` diffs the live exceptions list against.

Tests enforcing the rules on this page:

- `TestNoRegisteredSubcommandIsUnreferenced` — the check itself: fails,
  naming the command, when something is registered but nothing calls it.
- `TestOrphanAllowlistOnlyShrinks` — the shrink-only guard on the
  unused-command list.
- `TestCLIFlagAudit` — checks that documented commands and flags are real.
- `TestFlagAuditSkipListOnlyShrinks` — the shrink-only guard on the
  flag-check exceptions list.
- `TestAllowlistPolicyNamesEveryGuardedFile` — fails if this document stops
  naming one of the four files above.
- `TestWiringGuardsHaveNoRuntimeEscapeHatch` — fails if any of the guard
  files above ever grow a setting, flag, or environment variable that could
  switch the check off.
