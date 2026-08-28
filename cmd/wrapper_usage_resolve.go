package cmd

import (
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// wrapperUsageRequest is everything the resolver needs to answer "what did
// each of these workers cost on this run?" for whichever chat platform ran it.
type wrapperUsageRequest struct {
	// Platform is the run's host platform as the build manifest records it
	// ("claude", "opencode", "codex", ...). An unrecognised or empty value is
	// the directly-spawned path, where the provider parser has already
	// attached what it measured.
	Platform string

	// RepoRoot is this repository's root, used on the OpenCode path to bind
	// discovery to this repository's own sessions.
	RepoRoot string

	// StartedAt and EndedAt bound the run's own time window. A zero EndedAt
	// leaves the window open-ended forward from StartedAt.
	StartedAt time.Time
	EndedAt   time.Time

	// WorkerNames are the workers this run expects to account for. Every one
	// of them comes back in the result, reported or not.
	WorkerNames []string

	// Attached is the usage the provider parser already attached at the
	// dispatch boundary, keyed by worker name. It is a genuine provider
	// measurement read by the Go runtime, never a figure relayed through a
	// completion packet.
	Attached map[string]codex.WorkerUsage
}

// wrapperWorkerUsage is one worker's resolved usage for a run.
type wrapperWorkerUsage struct {
	WorkerName string

	// Usage carries the measurement when Reported is true. When Reported is
	// false it is the zero value and holds NO token figure at all -- D-01 as
	// amended: a worker whose tool reported nothing is shown with no number,
	// never a zero and never a guess.
	Usage codex.WorkerUsage

	// Reported distinguishes "measured, and this is the figure" from "the
	// platform reported nothing". The renderer must be able to tell the two
	// apart WITHOUT inspecting a number, because a real measurement and an
	// absent one can both present as zero.
	Reported bool
}

// wrapperUsageResolution is the resolver's whole answer for one run.
type wrapperUsageResolution struct {
	// Workers holds exactly one entry per requested worker name, in the order
	// requested. A worker is never dropped for being unreported: a vanished
	// worker makes a run look cheaper than it was.
	Workers []wrapperWorkerUsage

	// SessionUsage is the orchestrating session's OWN turns, which are not a
	// dispatched worker's spend. It is kept apart rather than folded into any
	// worker's row or silently discarded.
	SessionUsage    codex.WorkerUsage
	SessionReported bool

	// Diagnostics are plain-English notes about anything the resolver
	// declined to resolve. They are never errors: a partial, honestly
	// reported result must never fail a build.
	Diagnostics []string
}

// resolveWrapperWorkerUsage returns per-worker usage for one run, whichever
// chat platform ran it.
func resolveWrapperWorkerUsage(req wrapperUsageRequest) wrapperUsageResolution {
	return wrapperUsageResolution{}
}
