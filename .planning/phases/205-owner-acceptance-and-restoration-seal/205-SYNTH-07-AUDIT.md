# SYNTH-07 Audit: Do the Seven Restoration Studies Actually Hold Up?

Before this project changed any code across Phases 199 through 204, each phase
first had to write a "synthesis" document — a study that says what the old
2026 ("Classic") version of this program did, what the current program does
now, what other options were considered, and which specific check proves the
final choice was the right one. This is a safeguard against "restoring" a
feature in name only: writing a heading that says a feature is back without
the feature actually being real, tested, and traceable to the work that
implemented it.

This document is the audit of those seven studies. It checks two different
things, on purpose, because they need different tools: a small computer
program (`cmd/classic_synthesis_audit_test.go`, runnable as `go test ./cmd/
-run TestClassicSynthesis`) mechanically checks that every study contains all
eight required sections with real content underneath each heading — a
check that can never be fooled by a heading with nothing useful beneath it,
because it also looks at what follows the heading, not just the heading
itself. This document then does the second thing the program cannot: it
reads each study's actual words and judges whether it truly cites real
evidence, records real alternatives, links its conclusions to real code and
tests, and refuses to copy old behavior just because it used to exist. The
short answer: six of the seven studies are strong on both counts. The
seventh — Phase 199's, the very first one written, before this project had
settled on the eight-section format — is substantively strong (real
evidence, real alternatives, real test links) but organizes its content
under different heading names than the other six. That is a real, named
finding below, not something this document glosses over.

## How to read each artifact's section below

Each of the seven sections names the eight sections the shared template
(`.planning/research/v1.28-classic-synthesis-template.md`) requires, and
states, for each one, whether the automated check found it **present** (the
heading exists with real content under it), **empty** (the heading exists
but nothing follows it), or **absent** (no heading matches). It then judges
the artifact against the four things SYNTH-07 (this project's own
requirement for these studies) demands:

1. **Evidence** — does it cite exact old and current evidence (a specific
   file, line, or command output), not a vague memory of how things used to
   work?
2. **Alternatives** — does it record what else was considered and why it was
   rejected, not just announce the chosen answer?
3. **Linkage** — does it connect its conclusions to the actual plan tasks,
   tests, capability rows ("CAP" rows — numbered entries in this project's
   master list of old features that need a home in the new program), and
   verification that followed?
4. **No label-only restoration** — does it refuse to bring back old behavior
   just because a heading claims to, and does it call out when copying old
   code would be a mistake?

## Phase 199 — Front Door and Classic Contract

**Mandatory sections (8 required):** 2 of 8 present with matching heading
text (*Classic mechanism reconstruction*, *Verification contract*); 6 of 8
absent under the template's exact wording (*Outcome under investigation*,
*Current Go mechanism audit*, *Comparative synthesis matrix*, *Selected
architecture*, *Research-to-plan linkage*, *Open decisions and confidence*).

This is a real, structural finding, not a formatting nitpick this document is
waving away: Phase 199's study was written and signed on 2026-09-03, before
this project's eight-section template existed in its current numbered form.
Its actual heading names are different words for largely the same ideas —
for example, it has "Signed outcome" instead of "Outcome under
investigation," and "Comparative synthesis decisions" instead of
"Comparative synthesis matrix." The automated check is deliberately strict
(an exact match, not "close enough"), so it correctly reports these as
absent under the template's own wording — weakening that rule to let this
one document pass would defeat the entire point of building the check.

**Judgement of the four clauses**, read directly from the document's actual
content rather than inferred from its headings:

1. **Evidence — satisfied.** Every one of its ten decision rows cites an
   exact old source (e.g. `git show 3a5b81c2:.claude/commands/ant/help.md`)
   next to an exact current source (e.g. `cmd/init_cmd.go`).
2. **Alternatives — satisfied.** Its decision table has a dedicated
   "Considered alternatives" column for every row (e.g. for the front door:
   "Keep flat help; copy Classic wrappers; compose modern runtime
   primitives"), and a separate "Rejected alternatives and non-goals"
   section spells out why full old-style copying was rejected.
3. **Linkage — satisfied.** A 17-row locked-decision table connects each
   owner decision to the synthesis row that decided it and the verification
   consequence that proves it; a further "Signed implemented capability
   evidence" table names, for all 34 routed capability rows, the exact
   current file and the exact passing test.
4. **No label-only restoration — satisfied.** It explicitly states:
   "Byte-for-byte Classic restoration is rejected because it would restore
   prompt-owned mutation, duplicated truth, and best-effort shell
   semantics" — a direct, named refusal of copying old code just because it
   is old.

**Overall:** the study's substance meets SYNTH-07's four clauses. Its
shortfall is purely structural (heading text, not content) and is carried to
the limitations list below rather than fixed in this phase — see "Shortfalls
and where they go."

## Phase 200 — Iterative Planning

**Mandatory sections:** 8 of 8 present with content, exact match.

**Judgement:**

1. **Evidence — satisfied.** Eleven historical rows cite exact paths and
   line ranges across three Classic reference points (`3a5b81c2`, `v5.0.0`,
   `v5.4`); eleven current-mechanism rows cite exact current file/line
   evidence.
2. **Alternatives — satisfied.** Every comparative-matrix row records the
   options considered before choosing a disposition (e.g. "delete
   insert-phase / keep in-place mutation / route through revision"), and a
   dedicated "Rejected alternatives" table records why restoring
   `semantic-cli.sh` (an old shell script) was rejected in favor of the
   services that already provide the same value today.
3. **Linkage — satisfied.** Section 6 ties every decision to specific plan
   numbers, capability rows, and named verification IDs (e.g. `V-200-LOOP-01`).
4. **No label-only restoration — satisfied.** The document is explicit that
   Classic is "evidence about a useful experience, not code to restore," and
   names Classic's own unsafe parts (prompt code owning files, self-reported
   confidence, research replacement that could destroy history) as defects
   the current design must not reintroduce.

**Overall:** fully satisfies all four clauses; confidence stated as `high`
with a clear, honest reason.

## Phase 201 — Queen-Led Work Cycle

("Queen" is this project's name for the part of the program that decides
which helper does a task and explains why — it is the phase's own title, not
a person.)

**Mandatory sections:** 8 of 8 present with content, exact match.

**Judgement:** satisfies all four clauses. Evidence citations pair old
`git show` references against exact current `path:line` citations for every
row; alternatives are recorded per decision with a rationale; linkage
connects all fourteen decisions to named plans and tests, including a
specific named regression this study found in the current code
(`cmd/codex_continue.go:3061`, a real bug where two code paths could
disagree about the same verification, which the study requires a negative
test to close); the "no label-only restoration" clause is satisfied by the
document's explicit rule that Classic's double-review pattern must not
survive "in a new form," and by naming precisely which behaviors are kept
because they are already better (the deterministic floor, coherent jobs)
rather than assumed to need replacement.

## Phase 202 — Swarm, Oracle, and Live Colony

**Mandatory sections:** 8 of 8 present with content, exact match.

**Judgement:** satisfies all four clauses. Evidence citations span five
historical reference points; twelve comparative-matrix rows each record
alternatives considered (for example, whether Swarm's fourth investigation
lens should share Oracle's existing research type directly or use a sibling
type — decided in favor of a sibling, to avoid a cross-package dependency);
linkage connects every decision to plan numbers 202-02 through 202-15 and
named tests; the document explicitly rejects building a second live-activity
system or a second Oracle confidence table where the current one already
works.

**One narrow, harmless finding:** this document's own frontmatter list of
capability rows (`cap_rows:`) is missing one row, `CAP-072`, even though the
document's own body discusses and routes `CAP-072` in three separate places
(its "why this phase owns it" sentence, its comparative-matrix row
`SYN-202-12`, and its research-to-plan linkage table). The audit tool
therefore does not rely on the frontmatter list alone — it scans the whole
document's text for capability-row mentions, which correctly recovers all
eight rows Phase 202 actually claims, matching the master capability list
exactly (confirmed by `TestClassicSynthesisCapabilityRowsAreNotDoubleClaimed`,
which passes). The frontmatter list itself staying one row short of complete
is a pre-existing, cosmetic gap in an already-delivered phase's document,
out of this phase's scope to edit — see "Shortfalls and where they go."

## Phase 202.1 — Classic Visual Voice

**Mandatory sections:** 8 of 8 present with content, exact match — but with
an unusual, explicitly-declared shape worth naming plainly: four of the
eight sections (2 through 5) do not repeat their content in this file at
all. Instead, each one says, in a full sentence, "see this phase's own
research document, in the named section" and briefly summarizes what that
other document concludes. The document states its own reason for this up
front: writing the same mechanism study twice in two files is "a drift risk,
not added rigor" — one file could be updated and the other forgotten,
silently going stale. The automated check counts this as present, because a
genuine sentence of real content (a citation to exactly where the fuller
study lives) sits under each heading — it is not a blank heading. Whether a
pointer-sentence should count as fully "present" the way a paragraph of
direct evidence does is a real judgment call, and it is recorded honestly
here rather than hidden: the check can prove a heading is not empty, but it
cannot on its own prove that a one-sentence pointer carries the same weight
as a full paragraph. This document's own reading of the referenced sections
(verified directly, not merely trusted) confirms they do contain real
evidence and real comparisons, so the pointer is honest, not a shortcut
around the substance.

**Judgement:** satisfies all four clauses. Sections 1 and 6 through 8 are
original to this file and cite real evidence (exact Classic commit
references, exact current file/line citations for every screen's renderer);
the "Comparative synthesis matrix" table records five decisions with
selected disposition, what delivered each one, and what test proves it;
linkage is unusually precise — every roadmap criterion is tied to a named
plan and a named test in one table; the "no label-only restoration" clause
is satisfied by the document naming exactly one prior mistake this phase
fixed at its root cause (an internal handoff code leaking onto the owner's
screen) rather than patching every place it showed up.

## Phase 203 — Biological Runtime

**Mandatory sections:** 8 of 8 present with content, exact match.

**Judgement:** satisfies all four clauses, and does so unusually thoroughly.
Its evidence base is exhaustive: it runs literal `git grep` searches across
both historical reference commits to prove a named old mechanism
("trophallaxis," a term for a worker asking another worker for help and
receiving a scoped answer back) never actually existed under any name in
2026 — and records that honestly as "no historical evidence recoverable"
rather than inventing a plausible-sounding old mechanism to compare against.
Alternatives are recorded for all twelve comparative-matrix rows, each with
a rejected-alternatives table entry explaining specifically why the
simpler-looking option was wrong (for example, why "just fix the bug in the
second, already-running helper scheduler" was rejected in favor of deleting
it outright — because even a fixed version would still keep two separate
counters that could silently drift apart). Linkage names the exact plan
numbers (203-02 through 203-15) for every decision, confirmed by reading
each plan's own frontmatter this session rather than assuming the numbers
were still accurate. The "no label-only restoration" clause is satisfied
by six separate, explicitly-decided rulings (labelled (a) through (f) in the
document) that each name a real risk of restoring something that only looks
right and say plainly why the chosen design avoids it — including a ruling
that a certain historical assumption should NOT be restored at all (native
nested helper spawning) because the underlying platform capability is
disputed and unstable upstream.

## Phase 204 — Learning Governor

**Mandatory sections:** 8 of 8 present with content, exact match.

**Judgement:** satisfies all four clauses. Evidence citations are re-read
against this session's own code revision, not carried over stale from an
earlier read; alternatives and dispositions are recorded per decision;
linkage connects every decision to its own cited public-path test. The "no
label-only restoration" clause is where this document is most direct: its
own verification contract states plainly that a lesson the program has not
actually checked must never be shown to a worker "under a heading that
calls it proven" — precisely the class of problem SYNTH-07 exists to catch
— and the document itself records, rather than hides, one confirmed case
where a labelled capability could not yet be wired all the way through
(the "hypothesis promoter" gap, already tracked in this project's own open
issue log, `.planning/WINDOWS.md`, entry 44).

## Shortfalls and where they go

Two shortfalls were found. Both are pre-existing gaps in already-delivered
phases' documents, discovered by this new audit rather than caused by it —
this phase's own scope is the audit command and this written record, not
retroactively rewriting other phases' signed studies, so neither is edited
here. Both are carried forward, not silently dropped:

1. **Phase 199's study uses different heading names than the shared
   template.** Its content substantively satisfies all four SYNTH-07
   clauses (see judgement above), so this is a structural gap, not a
   substance gap. **Routed to:** the limitations card for this milestone —
   whether to retitle a already-signed, already-implemented study's headings
   is an editorial decision for a human to make deliberately, not an
   automatic side-effect of building an audit tool.
2. **Phase 202's frontmatter capability-row list omits `CAP-072`**, even
   though the document's own body correctly discusses and routes it. The
   audit tool works around this by reading the whole document's text, not
   only its frontmatter list, so no capability row is silently lost from
   this audit's own findings. **Routed to:** the limitations card — a
   one-line frontmatter correction to an already-delivered phase's document,
   left for the next time that document is deliberately touched.

**One further, structural limitation of the automated command itself** is
recorded here rather than left implicit: the command can prove a section
heading exists and has real (non-blank) content beneath it, but it cannot,
by itself, judge whether that content is honest, well-evidenced prose or a
plausible-sounding paragraph with nothing real behind it. That deeper judgment —
reading what each study actually says and checking it against the real
code — is what this written document does, by hand, for all seven studies
above. This split is intentional, not an oversight: a fully automatic
semantic judge of "is this evidence real" does not exist and should not be
faked. **Routed to:** carried forward as a standing limitation, not a fixable
defect of this phase.

No percentage, average, or combined score appears anywhere in this document,
or in the automated command's own output — each artifact's result is reported
as its own whole-number count out of eight sections, exactly as SYNTH-07
and this project's own reporting rule (PROOF-03) require.

---

*Audit performed 2026-09-15 against `cmd/classic_synthesis_audit_test.go`
(`go test ./cmd/ -run TestClassicSynthesis`, all 8 tests passing) and a
direct reading of all seven `*-CLASSIC-SYNTHESIS.md` files under
`.planning/phases/`. The audit command is read-only: it never writes to a
synthesis artifact, confirmed by `git status --porcelain
.planning/phases/` showing no changes to any `CLASSIC-SYNTHESIS.md` file
after every test run in this phase.*
