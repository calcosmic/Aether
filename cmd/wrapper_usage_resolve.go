package cmd

import (
	"fmt"
	"strings"
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

// wrapper usage platform words, normalized. The build manifest records
// codex.Platform values ("claude", "opencode", ...) while the session record
// written by the PreToolUse hook says "claude-code"; both mean the same lane.
const (
	wrapperUsagePlatformClaude   = "claude"
	wrapperUsagePlatformOpenCode = "opencode"
)

// normalizeWrapperUsagePlatform folds the platform spellings in circulation
// onto one word. Anything unrecognised -- including the empty string -- is the
// directly-spawned path, where there is no session artifact to read and the
// provider parser has already attached whatever it measured.
func normalizeWrapperUsagePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "claude", "claude-code", "claudecode":
		return wrapperUsagePlatformClaude
	case "opencode", "open-code":
		return wrapperUsagePlatformOpenCode
	default:
		return ""
	}
}

// wrapperUsageWasReported is the ONE rule for "did the platform tell us
// anything about this worker?".
//
// It keys on the source tag, never on a number. A worker that genuinely billed
// zero and a worker whose tool reported nothing both present as zero tokens,
// and D-01 as amended renders those two differently -- the first as a figure,
// the second as no figure at all. Only the tag can tell them apart.
func wrapperUsageWasReported(usage codex.WorkerUsage) bool {
	return strings.TrimSpace(usage.Source) != ""
}

// resolveWrapperWorkerUsage returns per-worker usage for one run, whichever
// chat platform ran it.
//
// It reads the platform's own on-disk artifact -- Claude Code's session
// transcript, OpenCode's session store -- or falls through to the measurement
// the provider parser already attached at the dispatch boundary. It NEVER
// reads a token figure out of a completion packet and never computes one: a
// number relayed by the orchestrating model is an assertion, not a
// measurement, and a number derived from a character count is a guess.
//
// It never returns an error. A missing, unreadable or ambiguous source
// degrades to the not-reported outcome with a plain-English note; a build must
// not fail because its accounting could not be read.
func resolveWrapperWorkerUsage(req wrapperUsageRequest) wrapperUsageResolution {
	res := wrapperUsageResolution{}

	// Requested names, deduplicated but kept in the order asked for.
	var names []string
	seen := map[string]bool{}
	for _, raw := range req.WorkerNames {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}

	measured := map[string]codex.WorkerUsage{}

	switch normalizeWrapperUsagePlatform(req.Platform) {
	case wrapperUsagePlatformClaude:
		record, ok := loadSpendSessionRecord()
		if !ok {
			res.Diagnostics = append(res.Diagnostics,
				"no Claude Code session transcript was recorded for this run, so no worker's usage could be read from it")
			break
		}
		rows, err := parseClaudeTranscriptUsage(record.TranscriptPath)
		if err != nil {
			res.Diagnostics = append(res.Diagnostics,
				fmt.Sprintf("the Claude Code session transcript could not be read (%v), so no worker's usage could be read from it", err))
			break
		}
		for _, row := range rows {
			if row.WorkerName == claudeTranscriptMainSessionWorker {
				// The orchestrating session's own turns are not a dispatched
				// worker's spend. Kept apart rather than folded into somebody
				// else's row or thrown away.
				res.SessionUsage = row.Usage
				res.SessionReported = true
				continue
			}
			switch {
			case seen[row.WorkerName]:
				measured[row.WorkerName] = row.Usage
			case seen[strings.TrimSpace(row.AgentID)]:
				measured[strings.TrimSpace(row.AgentID)] = row.Usage
			default:
				res.Diagnostics = append(res.Diagnostics,
					fmt.Sprintf("the session transcript holds usage for %q, which is not one of this run's workers, so it was left unattributed rather than guessed onto one", row.WorkerName))
			}
		}

	case wrapperUsagePlatformOpenCode:
		entries, ok := openCodeSessionUsageForRunOrNone(req.RepoRoot, req.StartedAt, req.EndedAt, names)
		if !ok {
			res.Diagnostics = append(res.Diagnostics,
				"no OpenCode session store was found on this machine, so no worker's usage could be read from it")
			break
		}
		for _, entry := range entries {
			measured[entry.WorkerName] = entry.Usage
		}

	default:
		// The directly-spawned path. Nothing platform-specific to read; the
		// provider parser's own attached measurement is the only source, and
		// it is applied to every path below.
	}

	for _, name := range names {
		// The provider's own report outranks a session artifact read: it is
		// the measurement, where the artifact is the platform's record of it.
		if usage, ok := req.Attached[name]; ok && wrapperUsageWasReported(usage) {
			res.Workers = append(res.Workers, wrapperWorkerUsage{WorkerName: name, Usage: usage, Reported: true})
			continue
		}
		if usage, ok := measured[name]; ok && wrapperUsageWasReported(usage) {
			res.Workers = append(res.Workers, wrapperWorkerUsage{WorkerName: name, Usage: usage, Reported: true})
			continue
		}
		// No figure is invented, and the worker is not dropped either: a
		// vanished worker makes a run look cheaper than it was.
		res.Workers = append(res.Workers, wrapperWorkerUsage{WorkerName: name, Reported: false})
	}

	return res
}
