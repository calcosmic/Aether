package cmd

import (
	"github.com/calcosmic/Aether/pkg/codex"
)

// Phase 196 plan 03 — the Claude Code session-transcript reader.
//
// SKELETON ONLY at this commit. The signatures are real and the package
// compiles; the bodies return nothing so the tests written against them fail
// for the right reason. The deleted branch worktree-agent-a59fd3ee68644ea21
// committed a test file referencing four symbols that had never existed in any
// ref, which turned a green repository red rather than failing a check. This
// is that mistake avoided deliberately.

// claudeTranscriptMaxLineBytes bounds any single transcript line the reader
// will hold in memory. A line larger than this is skipped and the rest of the
// file is still read. Mirrors internalWorkerRequestMaxBytes
// (cmd/internal_worker_adapter.go), which already bounds an untrusted input
// file the same way.
const claudeTranscriptMaxLineBytes = 2 << 20

// claudeTranscriptDefaultMaxBytes bounds how much of a transcript the
// unbounded entry point will read.
const claudeTranscriptDefaultMaxBytes int64 = 64 << 20

// claudeTranscriptMainSessionWorker is the worker name for the session's own
// turns — the assistant lines, which belong to the orchestrating session
// rather than to any dispatched subagent.
const claudeTranscriptMainSessionWorker = "main-session"

// claudeTranscriptUsage is one worker's usage as read by the Go runtime from
// the platform's own session transcript.
type claudeTranscriptUsage struct {
	// WorkerName is the readable identity: the subagent's declared type
	// (for example "gsd-executor"), or claudeTranscriptMainSessionWorker for
	// the session's own turns.
	WorkerName string

	// AgentID distinguishes two dispatches of the same subagent type. Empty
	// for the session's own row.
	AgentID string

	// Usage is always tagged UsageSourceSessionTranscript and never
	// provider-grade.
	Usage codex.WorkerUsage
}

// parseClaudeTranscriptUsage reads a Claude Code transcript and returns one
// usage row per worker.
func parseClaudeTranscriptUsage(path string) ([]claudeTranscriptUsage, error) {
	return parseClaudeTranscriptUsageWithBounds(path, claudeTranscriptDefaultMaxBytes)
}

// parseClaudeTranscriptUsageWithBounds is parseClaudeTranscriptUsage with an
// explicit bound on how many bytes of the transcript are read.
func parseClaudeTranscriptUsageWithBounds(path string, maxBytes int64) ([]claudeTranscriptUsage, error) {
	_ = path
	_ = maxBytes
	return nil, nil
}
