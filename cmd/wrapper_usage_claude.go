package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Phase 196 plan 03 — the Claude Code session-transcript reader.
//
// Claude Code writes ONE transcript line per content block of a reply: one for
// the thinking, one for the text, one for each tool call. Every one of those
// lines repeats the SAME cumulative usage object for the whole reply. Add the
// lines up and you bill one reply two or three times. On the fixture beside
// this file that is 2,764,260 tokens against a true 1,572,834 — 1.757x.
//
// So the rule (decision D-04 as corrected, 2026-08-27): deduplicate by
// `message.id`, last occurrence winning.
//
// D-04's FIRST recorded reading of this defect was wrong, and the way it was
// wrong is worth keeping in front of whoever edits this file. It said the
// repetition ran across the `user`, `queue-operation` and `attachment` line
// types and should be keyed on `tool_use_id`. Re-measured across the whole
// corpus on this machine — 1,726 transcripts, 1.0 GB — the usage-bearing lines
// are:
//
//	assistant   142581   usage at .message.usage
//	user           252   usage at .toolUseResult.usage
//
// and nothing else, anywhere. `queue-operation` and `attachment` carry usage
// ZERO times, and `tool_use_id` does not appear on a usage-bearing line at all,
// so a reader built to that reading would have deduplicated NOTHING while
// looking like it deduplicated something.
//
// The mistake was reading a NESTED `type` field on a content block and taking
// it for the line type — the "structure, not substrings" error inside the very
// decision that exists to forbid it. Hence the single most important rule in
// this file: THE LINE TYPE IS THE TOP-LEVEL `type` FIELD. Never a type on a
// content block, never a substring match anywhere.

// claudeTranscriptMaxLineBytes bounds any single transcript line the reader
// will hold in memory. A line larger than this is skipped and the rest of the
// file is still read — one pathological line must not cost the whole session.
// Mirrors internalWorkerRequestMaxBytes (cmd/internal_worker_adapter.go),
// which already bounds an untrusted input file the same way.
const claudeTranscriptMaxLineBytes = 2 << 20

// claudeTranscriptDefaultMaxBytes bounds how much of a transcript the
// unbounded entry point will read. Long sessions reach tens of megabytes; this
// is generous enough for a real session and finite enough that a corrupted or
// hostile file cannot turn a build's closeout into an unbounded read.
const claudeTranscriptDefaultMaxBytes int64 = 64 << 20

// claudeTranscriptMainSessionWorker is the worker name for the session's own
// turns — the `assistant` lines, which belong to the orchestrating session
// rather than to any dispatched subagent.
const claudeTranscriptMainSessionWorker = "main-session"

// claudeTranscriptUnnamedSubagentWorker is the fallback name for a subagent
// completion record that declares neither a type nor an id. Kept as its own
// row rather than folded into the session's: a worker whose identity was not
// recorded is still a worker, and merging its spend into the session's would
// silently misattribute it.
const claudeTranscriptUnnamedSubagentWorker = "subagent"

// claudeTranscriptUsage is one worker's usage as read by the Go runtime from
// the platform's own session transcript.
//
// It is a measurement, not a relay: the Go runtime opened the artifact and read
// it, rather than trusting a figure an orchestrating model quoted back.
type claudeTranscriptUsage struct {
	// WorkerName is the readable identity: the subagent's declared type (for
	// example "gsd-executor"), or claudeTranscriptMainSessionWorker for the
	// session's own turns.
	WorkerName string

	// AgentID distinguishes two dispatches of the same subagent type. Empty
	// for the session's own row.
	AgentID string

	// DispatchDescription is the description the caller passed to the Task
	// tool when it spawned this subagent, read from the dispatching `tool_use`
	// block that this completion record's `tool_use_id` points back at.
	//
	// It is the ONLY place the deterministic worker name ("Mason-67") appears
	// on this platform: `.claude/commands/ant/build.md` requires the wrapper to
	// use "{caste emoji} {Caste} {name}: {task}" as the visible description,
	// and Claude Code records that string verbatim. Empty when the dispatching
	// line carried no description or is not in this transcript.
	DispatchDescription string

	// AgentDefinition is the agent definition this subagent ran as -- the
	// `subagent_type` the caller asked for, which the completion record echoes
	// back as `agentType`. Several workers in one run can share one, so it
	// identifies a row only when exactly one worker ran as it.
	AgentDefinition string

	// RecordedAt is the moment the platform wrote this worker's completion
	// record, read from the line's own `timestamp` field. It is the zero time
	// when the line carried none.
	//
	// It exists so a run can tell its OWN records apart from an earlier
	// attempt's: one chat session routinely holds a build, its check and a
	// re-run of the same phase in one transcript file, and a re-run's workers
	// carry the same deterministic names as the first attempt's. Without a
	// date on the row, a retry was credited with the first attempt's spend
	// (NEW-02). Measured over every transcript in ~/.claude/projects on
	// 2026-08-28: all 252 subagent usage records carry an RFC3339 `timestamp`.
	RecordedAt time.Time

	// Usage is always tagged UsageSourceSessionTranscript and never
	// provider-grade. Only codex.ParseUsage may set the provider tag: a
	// transcript's accounting semantics are undocumented by the vendor, and
	// borrowing that tag would erode the trust boundary the type exists to
	// hold.
	Usage codex.WorkerUsage
}

// claudeTranscriptLine is the only shape this reader decodes. Everything it
// needs sits at a declared position, so a usage object quoted inside a tool
// call's description — or anywhere else in free text — is never matched.
type claudeTranscriptLine struct {
	// Type is the TOP-LEVEL line type, and the only line type this reader
	// will ever consult.
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	RequestID string `json:"requestId"`

	// Timestamp is when the platform wrote this line, RFC3339. Read at its
	// declared position like everything else here.
	Timestamp string `json:"timestamp"`

	Message *struct {
		ID    string                      `json:"id"`
		Model string                      `json:"model"`
		Usage *claudeTranscriptUsageBlock `json:"usage"`
		// Content is decoded lazily because it is a string on some lines and
		// an array of blocks on others.
		Content json.RawMessage `json:"content"`
	} `json:"message"`

	// ToolUseResult is decoded lazily because older transcripts sometimes
	// carry a plain string here instead of an object.
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

// claudeTranscriptToolUseResult is a dispatched subagent's completion record.
// It carries the worker's identity and no message id at all.
type claudeTranscriptToolUseResult struct {
	AgentType     string                      `json:"agentType"`
	AgentID       string                      `json:"agentId"`
	ResolvedModel string                      `json:"resolvedModel"`
	Usage         *claudeTranscriptUsageBlock `json:"usage"`
}

// claudeTranscriptContentBlock is one entry of `message.content`.
//
// Reading `type` HERE is not the mistake the file header forbids. That rule is
// about deciding what KIND OF LINE this is, which is always the top-level
// `type` and never a content block's. Once the line type is settled, the blocks
// inside it are structured data with their own declared kinds, and reading them
// at their declared positions is the opposite of substring matching.
type claudeTranscriptContentBlock struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	ToolUseID string `json:"tool_use_id"`
	Input     *struct {
		Description  string `json:"description"`
		SubagentType string `json:"subagent_type"`
	} `json:"input"`
}

// claudeTranscriptContentBlocks decodes `message.content` when it is an array
// of blocks, and returns nothing when it is a plain string or absent. A decode
// failure is never an error: the line's usage still counts.
func claudeTranscriptContentBlocks(raw json.RawMessage) []claudeTranscriptContentBlock {
	if len(raw) == 0 {
		return nil
	}
	var blocks []claudeTranscriptContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil
	}
	return blocks
}

// claudeTranscriptUsageBlock is the provider's four disjoint billed columns as
// the transcript spells them.
type claudeTranscriptUsageBlock struct {
	InputTokens              int64 `json:"input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
}

// claudeWorkerAccumulator collects one worker's distinct usage blocks.
//
// blocks is keyed by the deduplication key and holds the LAST occurrence seen,
// which is the whole point: the same reply is republished as the turn
// progresses, and the later line restates the same cumulative figure rather
// than reporting new spend.
type claudeWorkerAccumulator struct {
	workerName      string
	agentID         string
	model           string
	description     string
	agentDefinition string
	recordedAt      time.Time
	keyOrder        []string
	blocks          map[string]claudeTranscriptUsageBlock
}

// claudeSubagentDispatch is what the dispatching `tool_use` block recorded
// about one subagent, keyed by that block's own id.
//
// Claude Code writes the dispatch and its completion as two separate lines: the
// assistant line carrying the `tool_use` block comes first and holds the
// description and the requested subagent type, and the later `user` line
// carrying `toolUseResult` holds the usage and names the same tool_use id in
// its `tool_result` block. Measured across every transcript in
// ~/.claude/projects on this machine (2026-08-28): 250 subagent completion
// records, 250 of which carry that id and join back to a `tool_use` block, and
// `toolUseResult.agentType` equalled `input.subagent_type` on all 250.
type claudeSubagentDispatch struct {
	description  string
	subagentType string
}

// parseClaudeTranscriptUsage reads a Claude Code transcript and returns one
// usage row per worker, deduplicated.
//
// An absent file is not an error: a session that never wrote a transcript
// simply has nothing to report. A path outside the platform's own projects
// directory IS an error, refused before anything is opened.
func parseClaudeTranscriptUsage(path string) ([]claudeTranscriptUsage, error) {
	return parseClaudeTranscriptUsageWithBounds(path, claudeTranscriptDefaultMaxBytes)
}

// parseClaudeTranscriptUsageWithBounds is parseClaudeTranscriptUsage with an
// explicit bound on how many bytes of the transcript are read. The read stops
// at the bound rather than reading the whole file.
func parseClaudeTranscriptUsageWithBounds(path string, maxBytes int64) ([]claudeTranscriptUsage, error) {
	// Containment first, before anything is opened. This is the same rule the
	// session record already enforces — relative-path rejection, symlink
	// evaluation on both sides, and containment decided by filepath.Rel rather
	// than a lexical prefix match a crafted sibling directory could defeat.
	// Plan 196-05 lifts the shared helper out; the requirement here is that
	// the same rule is enforced and the same kind of error returned.
	validated, err := validateSpendTranscriptPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(validated)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open transcript %q: %w", validated, err)
	}
	defer func() { _ = file.Close() }()

	if maxBytes <= 0 {
		maxBytes = claudeTranscriptDefaultMaxBytes
	}
	reader := bufio.NewReaderSize(io.LimitReader(file, maxBytes), 64*1024)

	var order []string
	accumulators := map[string]*claudeWorkerAccumulator{}
	// dispatches is what each `tool_use` block recorded about the subagent it
	// spawned, keyed by that block's id. It is filled as the file is read and
	// consulted when the matching completion record arrives later in the same
	// file -- the dispatch line always precedes its own result.
	dispatches := map[string]claudeSubagentDispatch{}
	var ordinal int

	for {
		line, oversized, readErr := readBoundedTranscriptLine(reader, claudeTranscriptMaxLineBytes)
		ordinal++

		if len(line) > 0 && !oversized {
			absorbClaudeTranscriptLine(line, ordinal, &order, accumulators, dispatches)
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read transcript %q: %w", validated, readErr)
		}
	}

	if len(order) == 0 {
		return nil, nil
	}

	rows := make([]claudeTranscriptUsage, 0, len(order))
	for _, key := range order {
		accumulator := accumulators[key]
		usage := codex.WorkerUsage{
			Model:  accumulator.model,
			Source: codex.UsageSourceSessionTranscript,
		}
		for _, blockKey := range accumulator.keyOrder {
			block := accumulator.blocks[blockKey]
			usage.InputTokens += block.InputTokens
			usage.CacheCreationTokens += block.CacheCreationInputTokens
			usage.CachedInputTokens += block.CacheReadInputTokens
			usage.OutputTokens += block.OutputTokens
		}
		// TotalTokens is deliberately left unset so every consumer reaches the
		// authoritative sum through codex.WorkerUsage.BilledTotalTokens().
		// A second total-computing implementation in a second place is
		// Pitfall 1, the 186x undercount.
		rows = append(rows, claudeTranscriptUsage{
			WorkerName:          accumulator.workerName,
			AgentID:             accumulator.agentID,
			DispatchDescription: accumulator.description,
			AgentDefinition:     accumulator.agentDefinition,
			RecordedAt:          accumulator.recordedAt,
			Usage:               usage,
		})
	}
	return rows, nil
}

// absorbClaudeTranscriptLine decodes one line and, if it is usage-bearing,
// files its usage block under the right worker and the right deduplication key.
func absorbClaudeTranscriptLine(raw []byte, ordinal int, order *[]string, accumulators map[string]*claudeWorkerAccumulator, dispatches map[string]claudeSubagentDispatch) {
	var line claudeTranscriptLine
	if err := json.Unmarshal(raw, &line); err != nil {
		// A malformed or truncated line is skipped; the rest of the file is
		// still read.
		return
	}

	// THE LINE TYPE IS THE TOP-LEVEL FIELD. Reading a nested type off a
	// content block is what produced the superseded reading of D-04, and it
	// would silently widen the accepted set to line types that carry no usage
	// at all.
	switch line.Type {
	case "assistant":
		if line.Message == nil {
			return
		}
		// A subagent dispatch is recorded HERE, on the line that spawned it,
		// and nowhere else. Its description is the only place the deterministic
		// worker name appears on this platform, so it is captured whether or not
		// this line also carries usage.
		for _, block := range claudeTranscriptContentBlocks(line.Message.Content) {
			if block.Type != "tool_use" || block.Input == nil || block.ID == "" {
				continue
			}
			if strings.TrimSpace(block.Input.SubagentType) == "" {
				continue
			}
			dispatches[block.ID] = claudeSubagentDispatch{
				description:  strings.TrimSpace(block.Input.Description),
				subagentType: strings.TrimSpace(block.Input.SubagentType),
			}
		}
		if line.Message.Usage == nil {
			return
		}
		accumulator := claudeAccumulatorFor(claudeTranscriptMainSessionWorker, order, accumulators)
		accumulator.workerName = claudeTranscriptMainSessionWorker
		if at := parseClaudeTranscriptTime(line.Timestamp); !at.IsZero() {
			accumulator.recordedAt = at
		}
		if line.Message.Model != "" {
			accumulator.model = line.Message.Model
		}
		accumulator.record(claudeDeduplicationKey(line.Message.ID, line.RequestID, line.UUID, ordinal), *line.Message.Usage)

	case "user":
		if len(line.ToolUseResult) == 0 {
			return
		}
		var result claudeTranscriptToolUseResult
		if err := json.Unmarshal(line.ToolUseResult, &result); err != nil {
			// Older transcripts carry a plain string here, including the
			// legacy "<usage>subagent_tokens: N</usage>" text block. It is
			// tolerated — it does not break the read — and it is NOT counted:
			// scraping a number out of a text blob is exactly the substring
			// matching this file refuses to do.
			return
		}
		if result.Usage == nil {
			return
		}
		workerName := result.AgentType
		if workerName == "" {
			workerName = result.AgentID
		}
		if workerName == "" {
			workerName = claudeTranscriptUnnamedSubagentWorker
		}
		// Two dispatches of the same subagent type are two workers, so the
		// accumulator is keyed by the agent id where there is one.
		accumulatorKey := "agent:" + result.AgentID
		if result.AgentID == "" {
			accumulatorKey = fmt.Sprintf("agent-unnamed:%s:%d", workerName, ordinal)
		}
		accumulator := claudeAccumulatorFor(accumulatorKey, order, accumulators)
		accumulator.workerName = workerName
		accumulator.agentID = result.AgentID
		accumulator.agentDefinition = strings.TrimSpace(result.AgentType)
		if at := parseClaudeTranscriptTime(line.Timestamp); !at.IsZero() {
			accumulator.recordedAt = at
		}
		if result.ResolvedModel != "" {
			accumulator.model = result.ResolvedModel
		}
		// Join back to the line that dispatched this worker. The completion
		// record's own `tool_result` block names the `tool_use` id, which is the
		// platform's own link between the two -- not a guess and not a
		// heuristic.
		if line.Message != nil {
			for _, block := range claudeTranscriptContentBlocks(line.Message.Content) {
				if block.Type != "tool_result" || block.ToolUseID == "" {
					continue
				}
				dispatch, ok := dispatches[block.ToolUseID]
				if !ok {
					continue
				}
				accumulator.description = dispatch.description
				if dispatch.subagentType != "" {
					accumulator.agentDefinition = dispatch.subagentType
				}
			}
		}
		// A subagent completion record carries NO message id. It is counted
		// once on its own — never dropped for lack of an identifier, and never
		// folded into another worker's row. Its own line identity is the key,
		// so an exact repeat of the same line is still counted once.
		accumulator.record(claudeDeduplicationKey("", "", line.UUID, ordinal), *result.Usage)
	}
}

// parseClaudeTranscriptTime reads a transcript line's own `timestamp`. An
// absent or unparseable value is the zero time, which the window check treats
// as "cannot be placed in this run" rather than guessing it belongs here.
func parseClaudeTranscriptTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

// claudeDeduplicationKey chooses what a usage block is counted under.
//
// `message.id` is the key: the same reply is republished line by line as its
// content blocks are written out, every line restating the same cumulative
// usage. `requestId` corroborates it and stands in when a line carries no
// message id. Failing both, the line's own uuid — and failing that, its
// position — keeps the block as its own row rather than merging it with
// somebody else's spend.
func claudeDeduplicationKey(messageID, requestID, uuid string, ordinal int) string {
	switch {
	case messageID != "":
		return "message:" + messageID
	case requestID != "":
		return "request:" + requestID
	case uuid != "":
		return "uuid:" + uuid
	default:
		return fmt.Sprintf("line:%d", ordinal)
	}
}

func claudeAccumulatorFor(key string, order *[]string, accumulators map[string]*claudeWorkerAccumulator) *claudeWorkerAccumulator {
	if existing, ok := accumulators[key]; ok {
		return existing
	}
	created := &claudeWorkerAccumulator{blocks: map[string]claudeTranscriptUsageBlock{}}
	accumulators[key] = created
	*order = append(*order, key)
	return created
}

// record files a usage block under key, last occurrence winning.
func (a *claudeWorkerAccumulator) record(key string, block claudeTranscriptUsageBlock) {
	if _, seen := a.blocks[key]; !seen {
		a.keyOrder = append(a.keyOrder, key)
	}
	a.blocks[key] = block
}

// readBoundedTranscriptLine reads one newline-terminated line, refusing to hold
// more than limit bytes of it in memory.
//
// bufio.Scanner is deliberately not used: it abandons the whole file on a line
// longer than its buffer, and the requirement here is the opposite — skip the
// oversized line and keep reading. oversized reports that the line was consumed
// and discarded; the returned error is io.EOF at the end of input.
func readBoundedTranscriptLine(reader *bufio.Reader, limit int) ([]byte, bool, error) {
	var assembled []byte
	oversized := false

	for {
		chunk, err := reader.ReadSlice('\n')
		if !oversized {
			if len(assembled)+len(chunk) > limit {
				oversized = true
				assembled = nil
			} else {
				assembled = append(assembled, chunk...)
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		return assembled, oversized, err
	}
}
