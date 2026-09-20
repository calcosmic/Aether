// Package shadow is the sealed compartment LEARN-06 (204-08-PLAN.md) builds:
// a proposed change (a candidate) can be tried beside the current behaviour
// (the baseline) over the same visible work and the same held-back checks,
// graded by something the candidate structurally cannot reach or alter.
//
// The defence here is structural, never a rule written in prose. A
// candidate that could choose what it is measured against, or edit the
// measure, would produce a number that means nothing -- CLAUDE.md's own
// Definition of Done names exactly this failure mode as this project's
// repeated history. So: the grader (FrozenEvaluator) has no setter, no
// pointer-receiver method and no exported field; nothing in this package
// exports a mutable reference to it; and Candidate holds no field of the
// grader's type at all. Four separate structural tests in isolation_test.go
// read the source of this package to prove each of those facts, not merely
// assert them once and trust they stay true.
//
// This package imports nothing from cmd -- it is deliberately isolated in
// the shared library area alongside this project's other sealed-off
// components (Planner Assumption Q, 204-08-PLAN.md), rather than inside the
// cmd package where everything can see everything. The holdout file this
// package's own comparisons are checked against is resolved by the calling
// command layer, never read from inside this package (see comparison.go's
// own doc comment on Compare).
package shadow

import (
	"crypto/sha256"
)

// FrozenEvaluator is the grader nothing a candidate can reach may alter. It
// carries exactly two fields, both unexported: a digest over the definition
// it was built from, and the run function that actually grades a candidate
// against a task. There is deliberately no setter and no pointer-receiver
// method anywhere on this type -- a pointer receiver here would defeat the
// whole guarantee this package exists to make, and
// TestEvaluatorHasNoMutator fails the moment one appears.
type FrozenEvaluator struct {
	digest [32]byte
	run    func(Candidate, Task) Result
}

// NewFrozenEvaluator builds a FrozenEvaluator from def, the byte definition
// of what this grader measures, and run, the function that actually grades
// a candidate's attempt at a task. The digest is computed once, here, with
// crypto/sha256 over def -- constructing twice from the identical def always
// yields the identical digest (TestEvaluatorDigestIsDeterministic), and a
// different def always yields a different one
// (TestDifferentDefinitionsDigestDifferently).
//
// The grader's identity can be made to cover an AcceptanceCriteria's own
// identity by folding that criteria's Digest() bytes into def before
// calling this constructor -- altering the criteria then alters the
// grader's own digest, without this constructor needing a criteria
// parameter of its own.
func NewFrozenEvaluator(def []byte, run func(Candidate, Task) Result) FrozenEvaluator {
	return FrozenEvaluator{digest: sha256.Sum256(def), run: run}
}

// Digest returns a copy of the grader's identity. A copy, never a pointer or
// a slice sharing the evaluator's own backing array -- there is nothing here
// a caller could mutate to change what NewFrozenEvaluator computed.
func (e FrozenEvaluator) Digest() [32]byte {
	return e.digest
}

// Run grades candidate's attempt at task. Running the evaluator never
// changes its own digest -- TestCandidateCannotAlterItsEvaluatorDigest
// attempts this and every other reachable path and asserts the digest is
// byte-identical at every point.
func (e FrozenEvaluator) Run(candidate Candidate, task Task) Result {
	return e.run(candidate, task)
}

// AcceptanceCriteria follows FrozenEvaluator's exact shape: unexported
// fields, a constructor taking everything up front, a digest, accessor
// methods only -- no setter, no pointer receiver, no exported field.
type AcceptanceCriteria struct {
	digest [32]byte
}

// NewAcceptanceCriteria builds an AcceptanceCriteria from def, the byte
// definition of what counts as passing. Same determinism guarantee as
// NewFrozenEvaluator: identical def, identical digest; different def,
// different digest.
func NewAcceptanceCriteria(def []byte) AcceptanceCriteria {
	return AcceptanceCriteria{digest: sha256.Sum256(def)}
}

// Digest returns a copy of the criteria's identity.
func (a AcceptanceCriteria) Digest() [32]byte {
	return a.digest
}

// Task is one unit of work a candidate and a baseline are both run over --
// either drawn from the visible set or from the hidden holdout set. Task
// carries only an identifier: this package has no path to a holdout file
// (comparison.go's Compare takes the holdout task list as a parameter and
// never resolves it itself), so a Task value here never carries anything
// that could tell a candidate which set it belongs to.
type Task struct {
	id string
}

// NewTask builds a Task from its identifier.
func NewTask(id string) Task {
	return Task{id: id}
}

// ID returns the task's identifier.
func (t Task) ID() string {
	return t.id
}

// Result is what FrozenEvaluator.Run returns for one candidate-task pair:
// whether that attempt passed.
type Result struct {
	passed bool
}

// NewResult builds a Result from whether the attempt passed.
func NewResult(passed bool) Result {
	return Result{passed: passed}
}

// Passed reports whether the graded attempt passed.
func (r Result) Passed() bool {
	return r.passed
}
