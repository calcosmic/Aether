# The "Nothing Calls It" Rule

A command that nothing ever calls does not ship. If a command exists, something
real has to run it — a menu item you can click, or code that actually invokes
it. If nothing runs it, the choice is simple: give it something that runs it,
or remove the command. This document explains the automatic check that
enforces that rule, and the two lists of already-existing exceptions it
watches.

## What "a command" means

An entry in either list names the WHOLE command, including which menu it sits
under — not just the last word. `aether host colonize` and `aether colonize`
look similar, but they are two different, separately-registered commands: one
lives under the `host` menu, the other sits at the top level. Before this
document was corrected (172-09), the check only looked at the last word, so a
call to one of these two was wrongly accepted as proof the other was in use —
which meant a command could sit completely unused and the check would still
call it "fine." Naming the whole command, not just the last word, is what
fixes that.

## What the two lists are

Two automatic checks scan this project for commands nobody runs. Both found
commands that already existed before this rule was written — commands that
predate this discipline and haven't been fixed yet. Rather than pretend those
commands don't exist, each one is written down in a list, with a note saying
why it's being tolerated for now and which future piece of work is
responsible for cleaning it up.

- **The unused-command list** — every command found with no menu item and no
  code calling it, at the moment this rule went into effect (and, after
  172-09, at the moment the list was corrected to name whole commands instead
  of last words).
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

There is no approval process that lets a reviewer wave a new entry through.
The only way either list changes is by editing the saved copy of the list
directly — which shows up as a visible change in the same review anyone would
see for any other edit. Nothing about it happens invisibly.

## The one reviewed regeneration switch

There IS a switch that can rewrite the unused-command list:
`-update-orphan-allowlist`. It can rewrite the LIVE list
(`cmd/testdata/orphan_allowlist.json`) from the checker's own, honest scan of
the codebase. It can NEVER write either saved comparison copy — not the
regular baseline, and not the frozen pre-migration snapshot described below.
An automatic check reads this file's own source and fails if that ever
becomes possible.

This is the only switch of its kind this project keeps, and it is reviewed:
an automatic check scans every guard file for anything shaped like a runtime
on/off switch (an environment variable, a skip call, or a newly declared
command-line flag) and fails unless it finds **exactly one** such switch —
this one. If a second switch is ever added, the check fails by naming the new
line. If this one is ever quietly removed while the check still expects it,
the check fails that way too.

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

## The 172-09 migration: naming whole commands instead of last words

Before 172-09, an entry in either list was just the last word of a command
(`colonize`, `closeout`). That is exactly the gap described above: two
different commands sharing a last word could vouch for each other. 172-09
changed every entry to name the whole command instead, and that key-format
change means the entire list had to be regenerated and re-saved — every
existing entry's key text changed shape, even though almost none of the
underlying commands themselves changed.

To keep that migration honest and auditable, the list as it stood
immediately before this change was frozen into a permanent, never-edited-
again copy: `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json`.
An automatic check compares every entry in the current, whole-command-named
list against this frozen copy forever — every entry in the current list must
be one of the exact whole commands the frozen copy's entries became, or one
of the handful of separately reviewed discoveries named below. Sharing a last
word with an older tolerated command no longer carries a new command
forward: an earlier version of this check only compared last words, and that
meant a brand-new command nobody had ever reviewed could slip through simply
because its last word — a common word like `get` or `setup` — matched an
older, already-tolerated command sitting under a completely different menu.
This is what makes "the list can only shrink" still true across a change to
how entries are written, not just across ordinary edits.

Naming commands by their whole path (rather than by last word) revealed 9
commands that were being wrongly vouched for by an unrelated same-named
sibling, and are recorded below rather than fixed as part of this migration —
fixing them is separate work:

- `aether colonize` — wrongly vouched for by a call to `aether host colonize`
- `aether closeout` — wrongly vouched for by a call to `aether ceremony closeout`
- `aether host` (the bare menu with no specific action chosen) — wrongly
  vouched for by ANY call naming an action under that menu
- `aether host build` — wrongly vouched for by a call to the unrelated
  top-level `aether build`
- `aether host oracle` — wrongly vouched for by a call to the unrelated
  top-level `aether oracle`
- `aether host swarm` — wrongly vouched for by a call to the unrelated
  top-level `aether swarm`
- `aether host watch` — wrongly vouched for by a call to the unrelated
  top-level `aether watch`
- `aether export pheromones` — wrongly vouched for by a call to
  `aether import pheromones` (the two share the last word "pheromones";
  neither is actually called directly today — the real callers use the
  separate flat commands `aether export-signals` / `aether import-signals`)
- `aether import pheromones` — the mirror image of the entry above

## The real numbers, as recorded today

The unused-command list currently holds **293 commands** (whole-command
names, measured after the 172-09 migration; the previous count of 278 was
measured under the old last-word-only naming). The real arithmetic behind
that 293, exactly as the automatic check enforces it:

- Of the 278 frozen, pre-migration entries: **274** each became exactly one
  whole-command name (272 of them trivially — the same word with "aether "
  in front — and 2 relocated under a different menu than that trivial form
  would predict). The other **4** were last words that had quietly been
  standing in for more than one real command (`get`, `set`, `registry`,
  `wisdom`), and became **10** whole-command names between them. So the 278
  frozen entries legitimately expand to 274 + 10 = **284** whole-command
  names — never fewer, never more, without a reviewed edit to both the
  frozen copy and the code that describes the expansion.
- **9** are the newly revealed commands listed above, tagged as a reviewed
  finding of this migration rather than a pre-existing, unreviewed one.
- 284 + 9 = **293**. Exactly.
- **6** of the 284 are tagged as belonging to a specific future cleanup
  effort (the skill-related commands), because that effort's finish line is
  defined as this exact set of 6 reaching zero — they are already counted
  inside the 284 above, not an addition to it.

The flag-check exceptions list holds exactly 2 entries.

The files themselves are the authoritative count. The numbers above are
recorded as of the 172-09 migration and will drift as the lists shrink — read
the files directly rather than trusting this page for anything more current.

## Detail

The five files the checks read, and the automated tests that enforce the
rules above:

- `cmd/testdata/orphan_allowlist.json` — the live unused-command list (293
  entries, `name`/`reason`/`owner_phase` each; `name` is the whole command).
- `cmd/testdata/orphan_allowlist_baseline.json` — the committed baseline
  `TestOrphanAllowlistOnlyShrinks` diffs the live list against.
- `cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json` — the
  frozen, never-edited-again copy of the baseline exactly as it stood before
  the 172-09 whole-command migration (278 entries, last-word-only names).
  `TestPathMigrationDidNotWidenTolerance` diffs the current baseline against
  this file forever, so the migration itself stays auditable. Its exact
  contents are also written down inside the checking code as a SHA-256
  hash — `TestPreMigrationSnapshotIsFrozen` fails the moment this file
  differs by a single character, so a hand-added or hand-removed line can
  no longer pass unnoticed.
- `cmd/cli_flag_audit_test.go` — holds the live flag-check exceptions list
  (`skipSubcommands`) and both tests that enforce this document:
  `TestFlagAuditSkipListOnlyShrinks` and
  `TestAllowlistPolicyNamesEveryGuardedFile`.
- `cmd/testdata/flag_audit_skiplist_baseline.json` — the committed baseline
  `TestFlagAuditSkipListOnlyShrinks` diffs the live exceptions list against.

Tests enforcing the rules on this page:

- `TestNoRegisteredSubcommandIsUnreferenced` — the check itself: fails,
  naming the command, when something is registered but nothing calls it.
- `TestCallerEvidenceIsNotSharedBetweenSameLeafNames` — the permanent,
  hermetic proof that a call to one command can never vouch for a different,
  same-last-word command at a different menu.
- `TestOrphanAllowlistOnlyShrinks` — the shrink-only guard on the
  unused-command list.
- `TestOrphanAllowlistIsPathKeyed` — fails if any entry in either list stops
  being a real, whole-command name.
- `TestPathMigrationDidNotWidenTolerance` — the shrink-only guard that
  survives the whole-command migration: every current entry must be one of
  the 284 whole-command names the frozen snapshot's 278 entries legitimately
  expand to, or one of the 9 separately reviewed discoveries, compared by
  full command path rather than by last word alone. This is the check that
  makes it true that the headline claim above — "both lists can only get
  shorter" — actually holds, even across a change to how entries are named.
- `TestPathMigrationRejectsASameLeafNewcomer` — the by-construction proof
  that a brand-new, never-before-tolerated command sharing a last word with
  an older tolerated one (for example a made-up `aether newthing get`, or
  the real `aether colony-depth setup`) is rejected by its full path, for
  every one of the 278 frozen last words at once.
- `TestPreMigrationSnapshotIsFrozen` — fails if the frozen pre-migration
  snapshot's contents no longer match the SHA-256 hash pinned in Go source,
  catching a hand-edit to the one file every shrink-only guarantee above
  rests on.
- `TestCLIFlagAudit` — checks that documented commands and flags are real.
- `TestFlagAuditSkipListOnlyShrinks` — the shrink-only guard on the
  flag-check exceptions list.
- `TestAllowlistPolicyNamesEveryGuardedFile` — fails if this document stops
  naming one of the five files above.
- `TestWiringGuardsHaveNoRuntimeEscapeHatch` — fails if any of the guard
  files above ever grow a setting, flag, or environment variable that could
  switch the check off, other than the one reviewed regeneration switch
  described above.
