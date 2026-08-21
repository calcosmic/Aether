# Task 01: Small bug fix with a seeded failing test

## Substrate

`gorilla/mux` (Go), pinned commit `db9d1d0073d27a0a2d9a8c1bc52aa0af4374d265`
(see `bench/tasks/substrate.md` for the clone command and neutrality
evidence).

## The task

**This prompt text is handed to all three lanes verbatim. It is written the
way a real developer would be briefed, and contains none of either system's
internal vocabulary.**

> One of the tests in this repository is failing:
> `TestCleanPathPreservesNonSlashInput`. Fix the bug it caught so that test
> passes. Do not change the test itself — the test is correct as written;
> the bug is in the code it is testing.

## Setup

Before the run starts, the harness applies this seeded defect to a fresh
clone of the pinned commit, then adds the seeded test alongside it. Both
edits are to be applied together, as a single patch, before the system under
test ever sees the repository — the system starts from a repository that
already contains the failing test.

**The defect** — in `mux.go`, inside `cleanPath`, change:
```go
if p[len(p)-1] == '/' && np != "/" {
```
to:
```go
if p[0] == '/' && np != "/" {
```
This makes `cleanPath` decide whether to re-add a trailing slash by checking
the *first* character of the input instead of the *last* one. Since nearly
every path this function receives starts with `/` (it's the very first
`if` inside the same function normalizes exactly that), the check becomes
almost always true — so `cleanPath` incorrectly appends a trailing slash to
paths that never had one, which breaks route matching for any path used
without a trailing slash.

**The seeded test** — append to `mux_test.go`, after the existing
`newRequestHost` function at the end of the file:
```go
// TestCleanPathPreservesNonSlashInput verifies that cleanPath does not append
// a trailing slash to a path that did not have one to begin with.
func TestCleanPathPreservesNonSlashInput(t *testing.T) {
	got := cleanPath("/foo/bar")
	want := "/foo/bar"
	if got != want {
		t.Fatalf("cleanPath(%q) = %q, want %q (a path with no trailing slash must not gain one)", "/foo/bar", got, want)
	}
}
```

**Verified before recording this spec:** with both edits applied and no
further changes, `go test -run TestCleanPathPreservesNonSlashInput -v ./...`
fails with `cleanPath("/foo/bar") = "/foo/bar/", want "/foo/bar"`. Reverting
only the `mux.go` line back to `p[len(p)-1] == '/'` (the correct fix) makes
the same command pass, and the full suite (`go test ./...`, 72 pre-existing
tests plus the new one) passes as well.

## Done means

- Running `go test -run TestCleanPathPreservesNonSlashInput -v ./...` in the
  resulting repository exits 0 and prints `PASS`.
- Running the full suite, `go test ./...`, exits 0 — no pre-existing test
  was broken to make the seeded one pass.
- `mux_test.go` is byte-for-byte unchanged from the version the setup step
  wrote (the fix must live in `mux.go`, not in loosening the test).
- The seeded test name, `TestCleanPathPreservesNonSlashInput`, still exists
  in `mux_test.go` and was not renamed, skipped, or deleted.

## Allowlist

`bench/tasks/allowlists/01-bug-fix.txt`

## Per-lane invocation

- **GSD:** `/gsd-execute-phase` against a plan produced for this single bug
  fix, followed by `/gsd-verify-work`.
- **Aether interactive:** `/ant-build <phase>` against a phase describing
  this fix, followed by `/ant-continue`.
- **Aether autopilot:** `/ant-run` against a colony initialized with this
  task as its goal.
