package cmd

import (
	"fmt"
	"regexp"
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

	// AgentNameByWorker maps each accounting key in WorkerNames -- the
	// worker's own deterministic name, "Mason-67" -- to the agent DEFINITION
	// that worker was spawned as, "aether-builder". They are two different
	// identities and the platform records the second one, so a resolver
	// holding only the first can attribute nothing.
	AgentNameByWorker map[string]string

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

	// The orchestrating session's OWN turns are deliberately not collected
	// here (WR-07). They are not a dispatched worker's spend, and this block
	// reports what the helpers cost -- see spendCostLineHeading, which says so
	// on screen rather than leaving the reader to infer it.
	//
	// The run window added for NEW-02 bounds the DISPATCHED workers' records.
	// Turning it on the session's own turns as well would be a new capability
	// (what share of a turn spanning two phases belongs to each?) rather than a
	// correction, and a field gathered against the day it arrives is a field
	// nothing reads.

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
		// Workers grouped by the agent definition they ran as. A definition
		// several workers share cannot identify any one of them.
		byDefinition := map[string][]string{}
		for _, name := range names {
			if definition := strings.TrimSpace(req.AgentNameByWorker[name]); definition != "" {
				byDefinition[definition] = append(byDefinition[definition], name)
			}
		}
		outsideWindow := 0
		for _, row := range rows {
			if row.WorkerName == claudeTranscriptMainSessionWorker {
				// The orchestrating session's own turns are skipped, never
				// folded into somebody else's row. See the note on
				// wrapperUsageResolution for why they are not reported either.
				continue
			}
			// This run's own window, the same bound the OpenCode reader applies
			// (openCodeSessionInWindow). ONE chat session holds a build, its
			// check and every re-run of the same phase in ONE transcript file,
			// and a re-run's workers carry the same deterministic names as the
			// first attempt's -- so without this the retry was credited with the
			// earlier attempt's measurement and marked "measured" (NEW-02,
			// reproduced at 25x). It also stops one phase's records reaching
			// another phase's ledger.
			//
			// A record the platform did not date cannot be placed in this run,
			// so it is left out rather than assumed to belong here; the platform
			// dates all of them.
			if !timeInWindow(row.RecordedAt, req.StartedAt, req.EndedAt) {
				outsideWindow++
				continue
			}
			worker, why := claudeRowWorker(row, names, seen, byDefinition)
			if worker == "" {
				res.Diagnostics = append(res.Diagnostics, why)
				continue
			}
			if _, taken := measured[worker]; taken {
				// Two transcript rows both claiming one worker. Overwriting
				// would silently drop one worker's whole spend, which is the
				// same class of loss as never attributing it at all.
				res.Diagnostics = append(res.Diagnostics, fmt.Sprintf(
					"two subagent records in the session transcript both resolve to worker %s, so the second was left unattributed rather than replacing the first", worker))
				continue
			}
			measured[worker] = row.Usage
		}
		if outsideWindow > 0 {
			// The verb has to agree with the count: at one record the old
			// wording read "1 helper's token record ... were written ... they
			// were left out". The owner reads this line; it should not be
			// visibly ungrammatical.
			wasWere, itThey := "was", "it"
			if outsideWindow != 1 {
				wasWere, itThey = "were", "they"
			}
			res.Diagnostics = append(res.Diagnostics, fmt.Sprintf(
				"%s in the session transcript %s written outside this run's own start and finish times, so %s %s left out of it. The same chat session records every earlier attempt at this phase, and every other phase run in it.",
				spendWorkerRecordWord(outsideWindow), wasWere, itThey, wasWere))
		}

	case wrapperUsagePlatformOpenCode:
		entries, reasons, ok := openCodeSessionUsageForRunOrNone(req.RepoRoot, req.StartedAt, req.EndedAt, names)
		if !ok {
			res.Diagnostics = append(res.Diagnostics,
				"no OpenCode session store was found on this machine, so no worker's usage could be read from it")
			break
		}
		// Every reason the reader gave for declining to resolve something
		// reaches the owner, exactly as the Claude branch's own notes do.
		res.Diagnostics = append(res.Diagnostics, reasons...)
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

// spendWorkerRecordWord renders a count of transcript records with its noun, so
// the note above reads naturally at one and at many.
func spendWorkerRecordWord(count int) string {
	if count == 1 {
		return "1 helper's token record"
	}
	return fmt.Sprintf("%d helpers' token records", count)
}

// claudeRowWorker decides which of this run's workers a subagent transcript row
// belongs to, and returns a plain-English reason when it declines to decide.
//
// This is CR-01's fix and the order of the rules is the point of it. Claude Code
// records TWO identities for a dispatched subagent and NEITHER of them is the
// name the ledger accounts under:
//
//   - `agentType` / `subagent_type` -- the agent DEFINITION ("aether-builder").
//     Several workers in one run share it, so on its own it identifies nobody.
//   - `agentId` -- an opaque per-dispatch identifier the runtime never sees.
//
// The deterministic worker name ("Mason-67") appears only in the description the
// wrapper composed when it spawned the worker, which Claude Code records on the
// dispatching `tool_use` block. The reader carries that description here.
//
// So: the description is consulted first, because it names the accounting key
// directly. A row whose description names none of this run's workers is NOT
// then resolved by its definition -- the description is the stronger evidence
// and it says the row belongs to somebody else; falling through would credit
// one worker with another's tokens. Only a row with no description at all
// reaches the definition rule, and that rule refuses a definition more than one
// worker ran as.
func claudeRowWorker(row claudeTranscriptUsage, names []string, requested map[string]bool, byDefinition map[string][]string) (worker string, why string) {
	// The direct cases: the transcript already carries the accounting key.
	if requested[row.WorkerName] {
		return row.WorkerName, ""
	}
	if id := strings.TrimSpace(row.AgentID); requested[id] {
		return id, ""
	}

	description := strings.TrimSpace(row.DispatchDescription)
	if description != "" {
		var named []string
		for _, name := range names {
			if workerNameAppearsIn(description, name) {
				named = append(named, name)
			}
		}
		switch len(named) {
		case 1:
			return named[0], ""
		case 0:
			// It says the LABEL did not match, not that the worker was a
			// stranger, because the code cannot tell those apart and the
			// likelier of the two is the label (NEW-03). The description is
			// composed by a model following a markdown instruction
			// (.claude/commands/ant/build.md), not by the runtime, so a
			// paraphrase is the ordinary way this happens -- and telling the
			// owner a worker he did not run showed up sends him looking in the
			// wrong place while his own worker's spend goes unrecorded.
			return "", fmt.Sprintf(
				"a helper's token record could not be matched to any worker on this run — the dispatch it came from was labelled %q, which names none of them, so its use was left out rather than guessed onto somebody",
				shortenForDiagnostic(description))
		default:
			return "", fmt.Sprintf(
				"one dispatch in the session transcript names more than one of this run's workers (%s), so its usage was left unattributed rather than guessed onto one",
				strings.Join(named, ", "))
		}
	}

	definition := strings.TrimSpace(row.AgentDefinition)
	if definition == "" {
		definition = strings.TrimSpace(row.WorkerName)
	}
	candidates := byDefinition[definition]
	switch len(candidates) {
	case 1:
		return candidates[0], ""
	case 0:
		return "", fmt.Sprintf(
			"the session transcript holds usage for %q, which is not one of this run's workers, so it was left unattributed rather than guessed onto one",
			claudeRowIdentity(row))
	default:
		return "", fmt.Sprintf(
			"%d of this run's workers (%s) all ran as %q and the session transcript records nothing that tells them apart, so none of them was given that usage rather than guessing",
			len(candidates), strings.Join(candidates, ", "), definition)
	}
}

// claudeRowIdentity is the most useful identity a transcript row carries, for
// a message the owner reads.
func claudeRowIdentity(row claudeTranscriptUsage) string {
	if definition := strings.TrimSpace(row.AgentDefinition); definition != "" {
		return definition
	}
	if name := strings.TrimSpace(row.WorkerName); name != "" {
		return name
	}
	return strings.TrimSpace(row.AgentID)
}

// workerNameAppearsIn reports whether text names worker exactly, bounded by
// non-word characters so "Mason-6" never matches inside "Mason-67".
//
// It is the single implementation of that rule for the whole spend subsystem:
// openCodeTitleMatchesWorker matches an OpenCode session title with it and the
// Claude path matches a dispatch description with it. Two implementations of
// one matching rule is how the two platforms start disagreeing about which
// worker a figure belongs to.
func workerNameAppearsIn(text, worker string) bool {
	worker = strings.TrimSpace(worker)
	if worker == "" || strings.TrimSpace(text) == "" {
		return false
	}
	pattern, err := regexp.Compile(`\b` + regexp.QuoteMeta(worker) + `\b`)
	if err != nil {
		return false
	}
	return pattern.MatchString(text)
}

// shortenForDiagnostic keeps a quoted fragment short enough to read in a note
// the owner sees on screen. It measures TEXT for display and is never an input
// to any token count.
func shortenForDiagnostic(text string) string {
	const limit = 80
	runes := []rune(strings.Join(strings.Fields(text), " "))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "…"
}
