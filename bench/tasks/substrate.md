# Benchmark substrate repositories

This file names the two real, outside, open-source codebases that every
benchmark task in this harness runs against. Neither repository has anything
to do with Aether or GSD (the two systems this harness compares) — they are
ordinary small software projects picked from the public internet, the same
way a hiring test might use a real open-source codebase instead of a made-up
toy. That separation matters: if a task ran against this repository, GSD
would have a built-in advantage, because GSD is the tool that wrote this
repository's own project-tracking files.

One repository is written in Go, one in TypeScript. Splitting across two
programming-language ecosystems is deliberate — it exposes whether either
system carries hidden assumptions about one ecosystem's tooling (npm vs the
Go toolchain, Jest vs `go test`, and so on) rather than about coding in
general.

## Selection criteria

Both repositories were checked, not assumed, against the same seven
requirements before either was accepted:

1. Small — roughly under 10,000 lines of hand-written source code (tests and
   generated/vendored files don't count toward this).
2. A real, working test suite that finishes in well under two minutes.
3. Buildable with no network access after one initial dependency-download
   step (`go mod download` for the Go repo, `npm install` for the TypeScript
   one).
4. Active enough to be a realistic, currently-maintained codebase, but
   pinned to one exact, unmoving commit for every run.
5. Zero history of Aether or GSD — no folder, file, or commit message
   mentioning either tool.
6. A license that permits this kind of benchmark use.
7. One repository in Go, one in TypeScript.

## Repo 1 (Go): gorilla/mux

**What it is:** A small, widely-used Go library that matches incoming web
requests to the right piece of code to handle them (an HTTP router). It has
no dependencies of its own, which makes it about as clean a Go codebase as
exists for this purpose.

- **Repository:** https://github.com/gorilla/mux
- **Pinned commit (full SHA):** `db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265`
- **Clone command:**
  ```
  git clone https://github.com/gorilla/mux.git <dest>
  cd <dest>
  git checkout db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265
  ```
- **Dependency-fetch command:** `go mod download` (the module has zero
  external dependencies — the command completes instantly with "no module
  dependencies to download," and every later `go build`/`go test` run needs
  no network access at all)
- **Test command:** `go test ./...`
- **Measured test duration:** 2.215 seconds wall-clock (`time go test ./...`
  on the pinned commit), well under the two-minute ceiling
- **Measured source size:** 2,332 lines across the four non-test `.go` files
  at the repository root (`mux.go`, `middleware.go`, `regexp.go`,
  `route.go`), counted with:
  ```
  find . -name '*.go' -not -name '*_test.go' -not -path './.git/*' | xargs wc -l
  ```
  (The full repository including its own test files is 7,545 lines — still
  comfortably under the 10k ceiling even counting tests.)
- **License:** BSD 3-Clause ("New" or "Revised") License, copyright The
  Gorilla Authors. This is a standard permissive open-source license that
  explicitly allows using, copying, and modifying the code for any purpose,
  including a benchmark like this one, as long as the copyright notice is
  kept — which this file does by recording it here.

**Neutrality checks (all run against the pinned commit's clone):**

| Check | Command | Result |
|---|---|---|
| No `.aether` path | `find . -iname '.aether' -not -path './.git/*'` | no matches — clean |
| No `.planning` path | `find . -iname '.planning' -not -path './.git/*'` | no matches — clean |
| No AI-tool config files | `find . -maxdepth 2 \( -iname 'CLAUDE.md' -o -iname 'AGENTS.md' -o -iname 'OPENCODE.md' \)` | no matches — clean |
| No Aether/GSD history | `git log --oneline \| grep -i 'aether\|get-shit-done\|gsd'` | no matches — clean |
| License present and readable | `cat LICENSE` | present, BSD-3-Clause text confirmed |

## Repo 2 (TypeScript): colinhacks/zod

**What it is:** A small, widely-used TypeScript library that checks whether
a piece of data (like a form submission or an API response) actually matches
the shape a program expects, and reports exactly what's wrong if it
doesn't (a schema-validation library). It has real internal structure —
separate files for types, parsing logic, error formatting, and
locale-specific error messages — which is exactly the kind of shape a
multi-file feature task needs to touch.

- **Repository:** https://github.com/colinhacks/zod
- **Pinned commit (full SHA):** `ca42965df46b2f7e2747db29c40a26bcb32a51d5`
  (tag `v3.23.8`)
- **Clone command:**
  ```
  git clone https://github.com/colinhacks/zod.git <dest>
  cd <dest>
  git checkout ca42965df46b2f7e2747db29c40a26bcb32a51d5
  ```
- **Dependency-fetch command:** `npm install`
- **Test command:** `npm test` (runs `jest` against the TypeScript source
  directly, via `ts-jest`)
- **Measured test duration:** 4.333 seconds reported by Jest itself
  (488 tests across 57 suites, wall-clock `npm test` including Node startup
  was 5.668 seconds), well under the two-minute ceiling
- **Measured source size:** 6,338 lines across `src/**/*.ts`, excluding the
  `src/__tests__/` and `src/benchmarks/` directories, counted with:
  ```
  find src -name '*.ts' -not -path '*/__tests__/*' -not -path '*/benchmarks/*' | xargs wc -l
  ```
- **License:** MIT License, copyright Colin McDonnell. A standard permissive
  open-source license that explicitly allows using, copying, and modifying
  the code for any purpose, including this benchmark, as long as the
  copyright and license text are kept — which this file does by recording
  it here.

**Neutrality checks (all run against the pinned commit's clone):**

| Check | Command | Result |
|---|---|---|
| No `.aether` path | `find . -iname '.aether' -not -path './.git/*' -not -path './node_modules/*'` | no matches — clean |
| No `.planning` path | `find . -iname '.planning' -not -path './.git/*' -not -path './node_modules/*'` | no matches — clean |
| No AI-tool config files | `find . -maxdepth 2 \( -iname 'CLAUDE.md' -o -iname 'AGENTS.md' -o -iname 'OPENCODE.md' \)` | no matches — clean |
| No Aether/GSD history | `git log --oneline \| grep -i 'aether\|get-shit-done\|gsd'` | no matches — clean |
| License present and readable | `cat LICENSE` | present, MIT text confirmed |

## Rejected candidates

These were cloned and checked, then discarded, so the choice above is shown
to be constrained rather than arbitrary. Each entry names the one criterion
that ended the candidate's consideration first — a candidate may have had
other problems too, but only the first disqualifying one was recorded once
found, since that is enough to reject it.

| Candidate | Language | Failed criterion | Detail |
|---|---|---|---|
| `sindresorhus/is` | TypeScript | Criterion 5 (zero Aether/GSD history) | Repository root contains both a `CLAUDE.md` and an `AGENTS.md` file — AI-coding-tool configuration that this harness's own neutrality rule is written to catch |
| `date-fns/date-fns` | TypeScript | Criterion 1 (small, ~under 10k LOC) | 106,716 lines of TypeScript source — over ten times the size ceiling; also carries an `AGENTS.md` file at the root (a second, independent disqualifier) |
| `jshttp/type-is` | TypeScript | Not a fair substitute — too small | 638 lines total; too thin to support a genuine multi-file brownfield feature task (the whole library is effectively one file plus tests) |
| `sindresorhus/p-limit` | TypeScript | Not a fair substitute — too small | 974 lines total; single-purpose, single-file utility, same problem as above |
| `motdotla/dotenv` | TypeScript | Not a fair substitute — too small | 1,836 lines total; same problem as above |
| `sinclairzx81/typebox` | TypeScript | Criterion 1 (small, ~under 10k LOC) | 46,730 lines of TypeScript source in `src/` alone — far over the size ceiling |
| `expressjs/express` | — | Criterion 7 (must be TypeScript) | Plain JavaScript source (`lib/*.js`), not TypeScript — fails the language requirement outright regardless of its otherwise-reasonable 2,765-line size |

## Fresh clone contract

**Every run of every task gets its own brand-new clone of the pinned commit,
in a throwaway directory, and no run ever reuses a clone left over from an
earlier run.**

This is the one invariant every later lane runner (the script that actually
drives GSD or Aether through a task) must honour without exception. It is
what keeps the twelve baseline runs independent of one another: if a run
reused a previous run's clone, that clone could already carry a fix, a
partial edit, an extra branch, or leftover tool state from whichever system
ran there before — and the next system to touch it would be scored on a
substrate that isn't the same starting point the first system saw. A fresh
clone at the pinned SHA, every single time, is what makes "one deterministic
starting point per run" actually true rather than assumed.
