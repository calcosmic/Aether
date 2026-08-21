# Aether `/ant-init` — Field Report

**Date:** 2026-08-16 · **Aether:** v1.0.54 (binary and hub in sync)
**Repo:** `CalVault` — an Obsidian vault (markdown notes, zero source code)
**Outcome:** ❌ **No colony was created.** The command was abandoned after stage 1 of 8 and the work was completed without Aether.

---

## 1. What the user asked for

A long, rich, spoken-style goal: back up an Obsidian vault, simplify it to a personal diary + to-do + music-production vault, move other content to separate vaults, research prior intent from git history and sibling folders, and reorganise music notes into a usable structure.

This is a legitimate multi-phase project. It is exactly the shape of work a colony should own: research → plan → execute → verify. **Aether never got to offer that.**

## 2. What actually happened

| Stage (per `init.yaml`) | Ran? | Notes |
|---|---|---|
| `init-research` scan | ✅ | Returned in ~2s, `ok: true` |
| Codebase Summary | ⚠️ | Ran, but produced nothing usable (see §3) |
| Prior Context | ✅ | Correctly skipped — 0 archived colonies |
| Intent Refinement | ❌ | Abandoned |
| Colony Charter | ❌ | Abandoned |
| Colony Mode | ❌ | Abandoned |
| Pheromone Suggestions | ❌ | Abandoned |
| Shelf Backlog | ❌ | Abandoned |
| Approval / `aether init` | ❌ | Never called |

The assistant ran `init-research`, read the output, judged it worthless for the task, and silently switched to plain subagents. Final state: `aether status` → `"No colony initialized."`

**The drop-out was a judgement call by the assistant, not a crash.** Nothing errored. That is the important part: the CLI reported success while providing no value, so the path of least resistance was to walk away. A hard failure would have been *more* visible.

## 3. Root cause — the scan is code-only, and says nothing when it finds no code

`init-research` on this vault returned:

```json
{
  "detected_type": "unknown",
  "languages": [],
  "frameworks": [],
  "dir_classification": { "signals": ["no strong structural signals detected"] },
  "governance": { "linters": null, "test_frameworks": null, "ci_configs": null },
  "file_count": 538
}
```

Every discriminating field is empty. The repo is 538 files of markdown with a nine-month git history, a `.obsidian/` config directory, and clear top-level folder semantics — all of it invisible to the scanner because it only looks for languages, frameworks, linters, CI, and build tools.

### The generated charter was actively misleading

```
vision:      "A unknown project"          ← grammatical bug ("A unknown")
tech_stack:  "No specific tech stack detected"
governance:  "No formal governance detected"
key_risks:   "No CI/CD pipeline detected -- manual deployment risk.
              No test framework detected -- regression risk.
              No linter configured -- code quality may drift"
```

"Regression risk" and "manual deployment risk" for a personal note archive. If this had been shown to the user unedited it would have damaged trust in the tool.

### The pheromone suggestions were worse

All five deterministic suggestions were code-repo housekeeping:

- FOCUS: *"consider adding CI/CD pipeline for automated testing"*
- FEEDBACK: *"consider adding a LICENSE file"*
- FEEDBACK: *"consider adding a README.md"*
- FEEDBACK: *"consider adding code formatting configuration"*
- FEEDBACK: *"no documentation detected"*

For a diary. The spec (`init.yaml`) *does* correctly instruct the wrapper to separate scan housekeeping from strategic pheromones and not surface these as steering signals — that guidance works. But the runtime still spends effort generating five suggestions with a 100% discard rate on this repo class.

## 4. Secondary finding — the wrapper deleted Aether's own state

During the vault cleanup the assistant removed `.aether/` (62 MB, mostly `ts-host/node_modules` with two 10 MB esbuild binaries and 14 MB of TypeScript) as part of reclaiming disk space, alongside `.smart-env`, `.makemd`, and `.trash`.

Two observations for the Aether team:

1. **`.aether/` is indistinguishable from a build cache to an outside observer.** It sits at repo root, is untracked, is dominated by `node_modules`, and carries no top-level marker saying "this is durable colony state, not a cache." A `README` or `.aether/DO-NOT-DELETE` at the root of that directory would have stopped this.
2. **62 MB of `node_modules` inside a user's personal notes vault is a real cost.** In a 703 MB vault where the actual notes were 4.4 MB, Aether was 14× the size of the content it was there to help organise.

No colony state was lost here (none existed). But had a colony been running, this cleanup would have destroyed it.

## 5. Version check

`aether update` reported binary 1.0.54 = hub 1.0.54, no skew. Companion sync worked correctly: 11 assets copied, 306 unchanged. **The update path is healthy — this report is not about update.**

## 6. Recommendations, in priority order

**P1 — Detect and support non-code repos.** Add a repo class alongside the language/framework detection. A directory with a high markdown ratio, no source files, and (optionally) `.obsidian/`, `.logseq/`, `docs/`, or `content/` is a *knowledge repo*. For that class:
- suppress the CI/LICENSE/formatter/test pheromones entirely
- swap the risk template (content loss, broken links, orphaned notes, duplication — not "regression risk")
- describe structure by folder semantics and note counts, not by language

**P2 — Never emit a charter field that reads as a bug.** `"A unknown project"` is a string-concatenation defect. When confidence is low, say so plainly (`"Project type not determined from automated scan"`) rather than generating grammatically broken filler. Consider suppressing `tech_stack` / `governance` / `key_risks` entirely when every input is null, instead of printing five "No X detected" lines.

**P3 — Give the wrapper an explicit low-signal branch.** The spec's stop conditions cover *failure* (`init-research` fails → relay guidance and stop) but not *success-with-no-signal*. Add: when `detected_type == "unknown"` and `languages`, `frameworks`, and `governance` are all empty, skip the generated charter, tell the user the scan found nothing to go on, and drive Intent Refinement purely from the user's stated goal. Right now the assistant has to invent that branch, and inventing it looks a lot like abandoning the command.

**P4 — Mark `.aether/` as durable state.** Add a root-level marker file in the directory. Consider whether `ts-host/node_modules` needs to live inside the target repo at all, or could be centralised in `~/.aether/`.

**P5 — Consider whether `/ant-init` should decline.** For a task that is genuinely "reorganise files with careful verification," a colony may be the wrong tool. An honest *"this repo/task doesn't need a colony — here's why"* would be more valuable than an eight-stage interview that ends in a charter about CI pipelines. A tool that knows when not to run earns trust.

## 7. What was used instead, and how it went

Four general-purpose subagents in parallel (structure inventory, prior-design archaeology, git/sibling-vault history, stray-note comparison), then direct execution with verification at each step.

That worked well — the archaeology agent surfaced the decisive insight (four prior restructures had all failed the same way, and no previous plan had ever allowed *deleting* content). But note what was lost by not having a colony: **no phase plan, no persistent state, no pheromones capturing the hard-won constraints.**

That last point cost the user directly. Mid-session the assistant deleted a 125-file template library that the user had deliberately recovered two days earlier — and the user had to catch it. A REDIRECT pheromone (*"never delete the template library — it was forensically recovered"*) is precisely the mechanism that would have prevented it. **The colony's value proposition was demonstrated by its absence.**

---

### One-line summary for the Aether team

> `/ant-init` fails soft on non-code repositories: the scan succeeds, returns all-empty signals, and generates a grammatically broken charter plus five irrelevant CI/LICENSE pheromones — so the assistant abandons the command rather than show it to the user. Add a knowledge-repo class, suppress low-confidence charter fields, and give the wrapper an explicit low-signal branch.
