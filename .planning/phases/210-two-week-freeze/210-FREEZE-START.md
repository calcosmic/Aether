# Phase 210: the two-week freeze starts

Written 2026-09-25, before the count began. Everything the fortnight is judged by is fixed
here, so nothing can be re-cut once the numbers exist.

## The dates

The fortnight runs from **Thursday 25 September 2026** to **Thursday 9 October 2026**, the
dates the owner confirmed on 2026-09-25. It runs on the Aether program already installed on
his machine. No publish, release or version bump is involved or needed (D-01).

## The projects

**French Fluency** (`/Users/callumcowie/WORKSPACE/French Fluency`), confirmed by the owner on
2026-09-25 when he chose to start the trial that day with French Fluency on the list.

He has not named any other project yet, and the list is **open to additions**: a project he
starts using Aether in during the fortnight counts from the day he names it. The list was
therefore not fixed in advance, and the final report must say so.

The end-of-fortnight question is judged against this list. Using Aether on Aether itself
(this repository) does not count towards it.

## What counts as a blocker

A blocker is something that made the owner **stop his real work and come to Aether's chat to
report it or get it fixed** (D-03, his own choice).

Two other definitions were offered and turned down:

- *Anything that made him stop and work around it* was rejected. It counts irritations he
  solved himself, which gives a higher count and a harsher verdict.
- *Only things that made the work impossible* was rejected. It counts only total dead ends,
  which gives a flattering count and defeats the point of asking.

This definition is fixed now and will not be re-cut once the numbers exist.

## The stopping rule

Quoted exactly from `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`:

> If, after the freeze, he still hits a blocker more than about once a week, investment in the full
> framework stops. The parts that already earn their place (quick jobs, flags, the memory of his
> preferences and lessons) are kept on top of plain Claude.

Over a fortnight, "about once a week" is roughly two.

The rule will be applied to the real count, including when the honest answer is that the full
framework does not survive.

## The build this runs on

Four facts, read from the machine on 2026-09-25 rather than remembered:

1. **The installed program** reports version `1.0.88`. The file was last replaced at
   2026-09-25 17:15, built from this repository's commit `bcf60ed4`.
2. **This repository** is at commit `bcf60ed4`, dated 2026-09-25 15:31.
3. **The menu commands** (the `/ant-…` commands typed in a chat): 67 checked against the copies
   installed on this machine, 0 missing, 0 different.
4. **Phase 209's last fix is in the installed program.** That fix stops a quick job wrongly
   being reported as "bigger than it looked" (commit `9b2135c5`). It is part of commit
   `bcf60ed4`, which the installed program was built from. The previous install
   (2026-09-24 23:29) also had it.

The program was replaced once more on the first day, at 17:15, with the fixes for blocker 1
(see the list of blockers). The version before that was kept at
`~/.local/bin/aether.backup-1.0.88-pre-french-fix`.

Two heads-ups for the owner:

- **Setting a project up counts as real use.** If getting Aether going in a project is itself a
  fight, that is a blocker like any other and it gets written down.
- **The update command is safe today but can go stale.** The shared installed copy that the
  update command pulls from was re-published from this repository on the morning of
  2026-09-25 and matched it then (fact 3). If a fix lands in this repository during the
  fortnight, running the update command in a project before that copy is refreshed would
  bring back the older commands. If that bites, it is a blocker too.

## What this fortnight cannot tell you

The count works because a blocker is defined as something that made the owner come to Aether's
own chat, and the instruction telling that chat to write a row down lives in this repository.
A short matching note was also added to his machine-wide instructions, so a chat in any of his
projects is asked to write the row too.

The count can still come out too low. If he hits a problem in one of his own projects and deals
with it there, and that project's chat does not write the row, nothing detects the miss. A low
count makes the verdict on the framework too kind. The honest fix is not more machinery. He
knows this now, and a row can be added later from memory if he realises one was missed.

This fortnight deliberately does no bug hunting, no audits and no fixes he did not ask for.
Nothing gets fixed unless he actually hit it. Defect register entry 60 is left alone on purpose:
the hand-off from `/ant-go` into planning has never been proved by a real unattended session,
and real use is what should find out whether it works. If he hits it, it counts as a blocker.

## Rule changes during the fortnight

- **2026-09-26, the "nothing new" rule was broken on purpose, by the owner's choice.** After
  blocker 4 he asked for Aether's clarifying step (`/ant-discuss`) to interview him with about
  20 multiple-choice questions, the way GSD's discussion step does, before writing the project
  description, and for that description to be shown as a readable plain-English page. Offered
  "wait until after the fortnight", he chose "now, break the rule". The final report must say
  that from this date the trial measured a changed program, and which change.
