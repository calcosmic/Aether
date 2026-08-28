package cmd

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/codex"
)

// Every expected figure in this file is written out by hand from the fixture
// lines below. Nothing here is produced by calling the resolver, either reader,
// or BilledTotalTokens().
//
// Mason-67's subagent completion record:   input 100 + cacheCreate 200 + cacheRead 300 + output 400
// Vigil-12's subagent completion record:   input  11 + cacheCreate  22 + cacheRead  33 + output  44
// the orchestrating session's own turns:   input   1 + cacheCreate   2 + cacheRead   3 + output   4
const (
	resolverClaudeMasonTotal   = 1000
	resolverClaudeVigilTotal   = 110
	resolverClaudeSessionTotal = 10
)

// newResolverClaudeTranscript writes a Claude Code transcript inside a fresh
// temporary HOME and records it as this run's session, exactly as the PreToolUse
// hook would. It returns the temp home.
//
// THE SHAPE HERE IS THE SHAPE CLAUDE CODE ACTUALLY WRITES, and that is the whole
// point of this helper. Verified on 2026-08-28 against every transcript in
// ~/.claude/projects on the owner's machine: 250 subagent completion records,
// 250 of them carrying a tool_result whose tool_use_id joins back to the
// dispatching tool_use block, and `toolUseResult.agentType` equal to that
// block's `input.subagent_type` on all 250. Not one of them carried a
// deterministic worker name in `agentType`.
//
// An earlier version of this helper wrote "Mason-67" into `agentType`. That is a
// shape the platform never produces, and it made every Claude-path test pass
// while production attributed nothing at all -- the "fixture built in a shape
// production never emits" failure CLAUDE.md's Definition of Done names.
//
// The transcript deliberately repeats the orchestrating session's assistant line
// under the SAME message.id, so the deduplication the transcript reader performs
// is still observable through the resolver rather than only in the reader's own
// tests. It also carries one subagent completion record for a worker this run
// never dispatched, so an unattributed row is observable too.
func newResolverClaudeTranscript(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := filepath.Join(home, ".claude", "projects", "-resolver-fixture")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir transcript dir: %v", err)
	}
	transcriptPath := filepath.Join(projectDir, "sess-resolver.jsonl")

	lines := []string{
		resolverAssistantLine("u-1"),
		resolverAssistantLine("u-2"), // same message.id -- must be billed once, not twice
	}
	lines = append(lines, resolverSubagentDispatch(resolverSubagentFixture{
		ToolUseID:    "toolu_mason",
		Description:  "🔨🐜 Builder Mason-67: implement the parser",
		SubagentType: "aether-builder",
		AgentID:      "a1b2c3d4e5f60718",
		In:           100, CacheCreate: 200, CacheRead: 300, Out: 400,
	})...)
	lines = append(lines, resolverSubagentDispatch(resolverSubagentFixture{
		ToolUseID:    "toolu_vigil",
		Description:  "👁️🐜 Watcher Vigil-12: verify the parser",
		SubagentType: "aether-watcher",
		AgentID:      "b2c3d4e5f6071829",
		In:           11, CacheCreate: 22, CacheRead: 33, Out: 44,
	})...)
	// A worker this run never dispatched. Its agent DEFINITION is the same one
	// Roam-90 was dispatched as, so a resolver that joined on the definition
	// alone would credit Roam-90 with somebody else's 777,777 tokens.
	lines = append(lines, resolverSubagentDispatch(resolverSubagentFixture{
		ToolUseID:    "toolu_stray",
		Description:  "🔍🐜 Scout Stray-99: research the store",
		SubagentType: "aether-scout",
		AgentID:      "c3d4e5f607182930",
		In:           777777,
	})...)

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	if err := store.SaveJSON(spendSessionRel, spendSessionRecord{
		SchemaVersion:  spendSessionSchemaVersion,
		Platform:       "claude-code",
		SessionID:      "sess-resolver",
		TranscriptPath: transcriptPath,
		Cwd:            t.TempDir(),
		CapturedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save spend session record: %v", err)
	}
	return home
}

// resolverClaudeAgentNames is the map from the accounting key (the worker's own
// deterministic name) to the agent DEFINITION it was spawned as -- exactly what
// the build manifest carries and what the wrapper passes as `subagent_type`.
func resolverClaudeAgentNames() map[string]string {
	return map[string]string{
		"Mason-67": "aether-builder",
		"Vigil-12": "aether-watcher",
		// Roam-90 shares Stray-99's definition on purpose.
		"Roam-90": "aether-scout",
	}
}

func resolverAssistantLine(uuid string) string {
	return `{"type":"assistant","uuid":"` + uuid + `","requestId":"req_1","message":{"id":"msg_1","model":"claude-opus-5","content":[{"type":"text","text":"[redacted]"}],"usage":{"input_tokens":1,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}}}`
}

type resolverSubagentFixture struct {
	ToolUseID    string
	Description  string
	SubagentType string
	AgentID      string
	In           int64
	CacheCreate  int64
	CacheRead    int64
	Out          int64

	// At is the moment the platform recorded this dispatch. Claude Code writes
	// a `timestamp` on every line: measured 2026-08-28 over every transcript in
	// ~/.claude/projects, all 252 subagent usage records carry one and all 252
	// are RFC3339. A fixture without one is a shape the platform never emits.
	// The zero value means "just now", which is inside any window a test opens
	// around the present.
	At time.Time

	// OmitDescription writes the dispatching tool_use with no description at
	// all, which is what a caller that passed none produces.
	OmitDescription bool
	// OmitDispatch writes the completion record with no dispatching tool_use
	// line before it, which is what a transcript truncated mid-session looks
	// like.
	OmitDispatch bool
}

// resolverSubagentDispatch writes the TWO lines one subagent dispatch really
// produces: the assistant line whose `tool_use` block carries the description
// and the subagent type, then the user line whose `tool_result` names that same
// tool_use id and whose `toolUseResult` carries the usage.
func resolverSubagentDispatch(f resolverSubagentFixture) []string {
	at := f.At
	if at.IsZero() {
		at = time.Now()
	}
	stamp := at.UTC().Format(time.RFC3339Nano)

	var lines []string
	if !f.OmitDispatch {
		description := fmt.Sprintf(`"description":%q,`, f.Description)
		if f.OmitDescription {
			description = ""
		}
		lines = append(lines, fmt.Sprintf(
			`{"type":"assistant","uuid":"u-dispatch-%s","requestId":"req-%s","timestamp":%q,"message":{"id":"msg-%s","model":"claude-opus-5","content":[{"type":"tool_use","id":%q,"name":"Agent","input":{%s"subagent_type":%q,"prompt":"[redacted]"}}]}}`,
			f.AgentID, f.AgentID, stamp, f.AgentID, f.ToolUseID, description, f.SubagentType))
	}
	lines = append(lines, fmt.Sprintf(
		`{"type":"user","uuid":"u-%s","timestamp":%q,"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":%q,"content":"[redacted]"}]},"toolUseResult":{"agentType":%q,"agentId":%q,"resolvedModel":"claude-sonnet-5","usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d}}}`,
		f.AgentID, stamp, f.ToolUseID, f.SubagentType, f.AgentID, f.In, f.CacheCreate, f.CacheRead, f.Out))
	return lines
}

// TestARetryIsNotCreditedWithTheFirstAttemptsTokens is NEW-02.
//
// wrapperUsageRequest carries the run's own time window and the OpenCode reader
// honours it (openCodeSessionInWindow). The Claude reader read the whole
// transcript file and ignored it. One chat session routinely holds a build, its
// check, and a re-run of the same phase -- and because worker names are
// deterministic per phase and caste, BOTH attempts' dispatch descriptions name
// the same worker. The resolver's duplicate guard then keeps the first row in
// file order, which is the OLDEST one, so the retry's row was filed under the
// retry's own attempt id carrying the first attempt's measurement and marked
// "measured".
//
// Reproduced at 25x: attempt one's Mason-67 spent 1,000,000, the retry's spent
// 40,000, and the retry reported 1,000,000.
func TestARetryIsNotCreditedWithTheFirstAttemptsTokens(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := filepath.Join(home, ".claude", "projects", "-retry")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(projectDir, "sess.jsonl")

	now := time.Now()
	// Both attempts describe the SAME worker, because a re-run of one phase
	// gives its workers the same deterministic names.
	description := "🔨🐜 Builder Mason-67: implement the parser"
	var lines []string
	lines = append(lines, resolverSubagentDispatch(resolverSubagentFixture{
		ToolUseID:    "toolu_first",
		Description:  description,
		SubagentType: "aether-builder",
		AgentID:      "1111111111111111",
		At:           now.Add(-2 * time.Hour),
		In:           1_000_000,
	})...)
	lines = append(lines, resolverSubagentDispatch(resolverSubagentFixture{
		ToolUseID:    "toolu_retry",
		Description:  description,
		SubagentType: "aether-builder",
		AgentID:      "2222222222222222",
		At:           now.Add(-10 * time.Minute),
		In:           40_000,
	})...)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := store.SaveJSON(spendSessionRel, spendSessionRecord{
		SchemaVersion:  spendSessionSchemaVersion,
		Platform:       "claude-code",
		SessionID:      "sess-retry",
		TranscriptPath: path,
		Cwd:            t.TempDir(),
		CapturedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save session record: %v", err)
	}

	// The RETRY's own window: it started half an hour ago, long after the first
	// attempt finished.
	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:          "claude-code",
		RepoRoot:          t.TempDir(),
		StartedAt:         now.Add(-30 * time.Minute),
		EndedAt:           now,
		WorkerNames:       []string{"Mason-67"},
		AgentNameByWorker: map[string]string{"Mason-67": "aether-builder"},
	})

	mason := resolverWorker(t, res, "Mason-67")
	if !mason.Reported {
		t.Fatalf("the retry's own worker got no figure at all: %+v, diagnostics %v", mason, res.Diagnostics)
	}
	if got := mason.Usage.BilledTotalTokens(); got != 40_000 {
		t.Errorf("the retry reports %d tokens for a worker that spent 40000; the earlier attempt's record is in the same transcript file and outside this run's window, and reading the whole file credits this run with it. Diagnostics: %v",
			got, res.Diagnostics)
	}

	t.Run("a first attempt still reads its own figure", func(t *testing.T) {
		res := resolveWrapperWorkerUsage(wrapperUsageRequest{
			Platform:          "claude-code",
			RepoRoot:          t.TempDir(),
			StartedAt:         now.Add(-3 * time.Hour),
			EndedAt:           now.Add(-90 * time.Minute),
			WorkerNames:       []string{"Mason-67"},
			AgentNameByWorker: map[string]string{"Mason-67": "aether-builder"},
		})
		mason := resolverWorker(t, res, "Mason-67")
		if !mason.Reported || mason.Usage.BilledTotalTokens() != 1_000_000 {
			t.Errorf("the first attempt = %+v, want 1000000: its own record is inside its own window. Diagnostics: %v", mason, res.Diagnostics)
		}
	})
}

func resolverWorker(t *testing.T, res wrapperUsageResolution, name string) wrapperWorkerUsage {
	t.Helper()
	for _, w := range res.Workers {
		if w.WorkerName == name {
			return w
		}
	}
	t.Fatalf("worker %q is missing from the resolution entirely; a vanished worker makes a run look cheaper than it was: %+v", name, res.Workers)
	return wrapperWorkerUsage{}
}

func assertNoFigure(t *testing.T, w wrapperWorkerUsage) {
	t.Helper()
	if w.Reported {
		t.Errorf("worker %q is marked reported; it should be marked not reported", w.WorkerName)
	}
	if w.Usage.Source != "" {
		t.Errorf("worker %q carries source %q; an unreported worker carries no measurement at all", w.WorkerName, w.Usage.Source)
	}
	if w.Usage.BilledTotalTokens() != 0 || w.Usage.InputTokens != 0 || w.Usage.OutputTokens != 0 ||
		w.Usage.CachedInputTokens != 0 || w.Usage.CacheCreationTokens != 0 || w.Usage.TotalTokens != 0 {
		t.Errorf("worker %q carries a token figure %+v; D-01 as amended gives an unreported worker no number at all -- not zero, not a guess", w.WorkerName, w.Usage)
	}
}

func TestResolverReadsTranscriptOnTheClaudePath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	newResolverClaudeTranscript(t)

	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:          "claude-code",
		RepoRoot:          t.TempDir(),
		StartedAt:         time.Now().Add(-time.Hour),
		WorkerNames:       []string{"Mason-67", "Vigil-12", "Roam-90"},
		AgentNameByWorker: resolverClaudeAgentNames(),
	})

	mason := resolverWorker(t, res, "Mason-67")
	if !mason.Reported {
		t.Fatalf("Mason-67 came back not reported, but the transcript carries its completion record")
	}
	if got := mason.Usage.BilledTotalTokens(); got != resolverClaudeMasonTotal {
		t.Errorf("Mason-67 billed total = %d, want %d", got, resolverClaudeMasonTotal)
	}
	if mason.Usage.InputTokens != 100 || mason.Usage.CacheCreationTokens != 200 ||
		mason.Usage.CachedInputTokens != 300 || mason.Usage.OutputTokens != 400 {
		t.Errorf("Mason-67 columns = %+v, want input 100 / cache-create 200 / cache-read 300 / output 400", mason.Usage)
	}
	if mason.Usage.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("Mason-67 source = %q, want %q -- a transcript row is never provider-grade", mason.Usage.Source, codex.UsageSourceSessionTranscript)
	}

	vigil := resolverWorker(t, res, "Vigil-12")
	if !vigil.Reported || vigil.Usage.BilledTotalTokens() != resolverClaudeVigilTotal {
		t.Errorf("Vigil-12 = %+v, want a reported billed total of %d", vigil, resolverClaudeVigilTotal)
	}

	assertNoFigure(t, resolverWorker(t, res, "Roam-90"))

	// The orchestrating session's own turns are never a worker's row and are
	// never reported as a phase's cost either -- see wrapperUsageResolution.
	for _, w := range res.Workers {
		if w.WorkerName == claudeTranscriptMainSessionWorker {
			t.Errorf("the orchestrating session appears as a dispatched worker; it is not one")
		}
		if w.Usage.BilledTotalTokens() == resolverClaudeSessionTotal+resolverClaudeMasonTotal {
			t.Errorf("worker %q absorbed the session's own turns into its own figure", w.WorkerName)
		}
	}

	stray := false
	for _, d := range res.Diagnostics {
		if strings.Contains(d, "Stray-99") {
			stray = true
		}
	}
	if !stray {
		t.Errorf("a transcript row for a worker this run never dispatched was silently dropped; expected a plain-English note naming Stray-99, got %v", res.Diagnostics)
	}
}

func TestResolverReadsSessionStoreOnTheOpenCodePath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	_, repoRoot := setupOpenCodeFixtureHome(t)

	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:    "opencode",
		RepoRoot:    repoRoot,
		StartedAt:   openCodeFixtureWindowStart,
		EndedAt:     openCodeFixtureWindowEnd,
		WorkerNames: []string{"Mason-67", "Vigil-12", "Roam-90"},
	})

	mason := resolverWorker(t, res, "Mason-67")
	if !mason.Reported {
		t.Fatalf("Mason-67 came back not reported, but the session store holds its two assistant messages")
	}
	// The hand-summed two-message total from cmd/testdata/spend/opencode/README.md.
	if got := mason.Usage.BilledTotalTokens(); got != openCodeFixtureSumTotal {
		t.Errorf("Mason-67 billed total = %d, want %d", got, openCodeFixtureSumTotal)
	}
	if mason.Usage.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("Mason-67 source = %q, want %q", mason.Usage.Source, codex.UsageSourceSessionTranscript)
	}

	vigil := resolverWorker(t, res, "Vigil-12")
	if !vigil.Reported || vigil.Usage.BilledTotalTokens() != 8100 {
		t.Errorf("Vigil-12 = %+v, want a reported billed total of 8100", vigil)
	}

	assertNoFigure(t, resolverWorker(t, res, "Roam-90"))
}

func TestResolverUsesAttachedUsageOnTheDirectPath(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("HOME", t.TempDir())

	// The provider's own documented disjoint-column example.
	attached := codex.WorkerUsage{
		InputTokens:         50,
		CachedInputTokens:   100000,
		CacheCreationTokens: 2000,
		OutputTokens:        500,
		Source:              codex.UsageSourceProvider,
	}

	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:    "codex",
		RepoRoot:    t.TempDir(),
		StartedAt:   time.Now().Add(-time.Hour),
		WorkerNames: []string{"Mason-67", "Roam-90"},
		Attached:    map[string]codex.WorkerUsage{"Mason-67": attached},
	})

	mason := resolverWorker(t, res, "Mason-67")
	if !mason.Reported {
		t.Fatalf("Mason-67 came back not reported, but the provider parser had already attached a measurement")
	}
	// 50 + 100000 + 2000 + 500, added by hand.
	if got := mason.Usage.BilledTotalTokens(); got != 102550 {
		t.Errorf("Mason-67 billed total = %d, want 102550", got)
	}
	if mason.Usage.Source != codex.UsageSourceProvider {
		t.Errorf("Mason-67 source = %q, want the provider tag it arrived with", mason.Usage.Source)
	}

	assertNoFigure(t, resolverWorker(t, res, "Roam-90"))

	t.Run("a provider measurement outranks a transcript read for the same worker", func(t *testing.T) {
		newResolverClaudeTranscript(t)
		res := resolveWrapperWorkerUsage(wrapperUsageRequest{
			Platform:          "claude-code",
			RepoRoot:          t.TempDir(),
			StartedAt:         time.Now().Add(-time.Hour),
			WorkerNames:       []string{"Mason-67"},
			AgentNameByWorker: resolverClaudeAgentNames(),
			Attached:          map[string]codex.WorkerUsage{"Mason-67": attached},
		})
		mason := resolverWorker(t, res, "Mason-67")
		if mason.Usage.Source != codex.UsageSourceProvider || mason.Usage.BilledTotalTokens() != 102550 {
			t.Errorf("Mason-67 = %+v, want the provider's own measurement rather than the transcript read", mason.Usage)
		}
	})
}

func TestResolverMarksUnreportedWorkersWithNoFigure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("HOME", t.TempDir())

	cases := []struct {
		name string
		req  wrapperUsageRequest
	}{
		{
			name: "the claude path with no session recorded at all",
			req: wrapperUsageRequest{
				Platform:    "claude-code",
				WorkerNames: []string{"Mason-67", "Vigil-12"},
				StartedAt:   time.Now().Add(-time.Hour),
			},
		},
		{
			name: "the opencode path with no session store on the machine",
			req: wrapperUsageRequest{
				Platform:    "opencode",
				RepoRoot:    t.TempDir(),
				WorkerNames: []string{"Mason-67", "Vigil-12"},
				StartedAt:   time.Now().Add(-time.Hour),
			},
		},
		{
			name: "the direct path where the provider attached nothing",
			req: wrapperUsageRequest{
				Platform:    "codex",
				WorkerNames: []string{"Mason-67", "Vigil-12"},
				Attached:    map[string]codex.WorkerUsage{"Mason-67": {}},
				StartedAt:   time.Now().Add(-time.Hour),
			},
		},
		{
			name: "a platform nobody recognises",
			req: wrapperUsageRequest{
				Platform:    "some-future-harness",
				WorkerNames: []string{"Mason-67", "Vigil-12"},
				StartedAt:   time.Now().Add(-time.Hour),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := resolveWrapperWorkerUsage(tc.req)
			if len(res.Workers) != len(tc.req.WorkerNames) {
				t.Fatalf("got %d worker rows, want %d -- every expected worker must come back, reported or not: %+v",
					len(res.Workers), len(tc.req.WorkerNames), res.Workers)
			}
			for _, name := range tc.req.WorkerNames {
				assertNoFigure(t, resolverWorker(t, res, name))
			}
		})
	}
}

func TestResolverNeverReadsUsageFromACompletionPacket(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	t.Setenv("HOME", t.TempDir())

	t.Run("a packet asserting a figure changes nothing", func(t *testing.T) {
		// A completion packet is the wrapper's own word for what happened. It
		// declares no place to put a token figure on purpose, and even one
		// that smuggles a key in must not reach a worker's row.
		packet := map[string]any{
			"dispatches": []any{
				map[string]any{
					"name":   "Mason-67",
					"status": "success",
					"usage":  map[string]any{"total_tokens": 999999, "source": "provider"},
				},
			},
		}
		raw, err := json.Marshal(packet)
		if err != nil {
			t.Fatalf("marshal packet: %v", err)
		}
		packetPath := filepath.Join(t.TempDir(), "completion.json")
		if err := os.WriteFile(packetPath, raw, 0o644); err != nil {
			t.Fatalf("write packet: %v", err)
		}

		res := resolveWrapperWorkerUsage(wrapperUsageRequest{
			Platform:    "claude-code",
			RepoRoot:    filepath.Dir(packetPath),
			StartedAt:   time.Now().Add(-time.Hour),
			WorkerNames: []string{"Mason-67"},
		})
		assertNoFigure(t, resolverWorker(t, res, "Mason-67"))
	})

	t.Run("the resolver names no completion-packet symbol at all", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "wrapper_usage_resolve.go", nil, 0)
		if err != nil {
			t.Fatalf("parse wrapper_usage_resolve.go: %v", err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			words := splitIdentifierWordsForResolverGuard(ident.Name)
			has := func(w string) bool {
				for _, got := range words {
					if got == w {
						return true
					}
				}
				return false
			}
			if (has("completion") && has("packet")) || (has("external") && has("build") && has("result")) {
				t.Errorf("%s at %s names the completion packet; a token figure relayed by the orchestrating model is an assertion, not a measurement",
					ident.Name, fset.Position(ident.Pos()))
			}
			return true
		})
	})
}

// splitIdentifierWordsForResolverGuard splits a camelCase or snake_case
// identifier into lowercase words, so the guard matches whole words rather
// than substrings.
func splitIdentifierWordsForResolverGuard(name string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = nil
		}
	}
	for _, r := range name {
		switch {
		case r == '_' || r == '-':
			flush()
		case unicode.IsUpper(r):
			flush()
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	flush()
	return words
}

func TestResolverIsIdempotent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	newResolverClaudeTranscript(t)

	req := wrapperUsageRequest{
		Platform:          "claude-code",
		RepoRoot:          t.TempDir(),
		StartedAt:         time.Now().Add(-time.Hour),
		WorkerNames:       []string{"Mason-67", "Vigil-12", "Roam-90"},
		AgentNameByWorker: resolverClaudeAgentNames(),
	}

	first := resolveWrapperWorkerUsage(req)
	second := resolveWrapperWorkerUsage(req)

	if len(first.Workers) != len(second.Workers) {
		t.Fatalf("row count changed between calls: %d then %d", len(first.Workers), len(second.Workers))
	}
	for i := range first.Workers {
		a, b := first.Workers[i], second.Workers[i]
		if a.WorkerName != b.WorkerName || a.Reported != b.Reported || a.Usage != b.Usage {
			t.Errorf("row %d changed between calls: %+v then %+v -- nothing may accumulate across calls", i, a, b)
		}
	}
	if len(second.Workers) == 0 {
		t.Fatalf("the second call returned no worker rows at all")
	}
	if got := second.Workers[0].Usage.BilledTotalTokens(); got != resolverClaudeMasonTotal {
		t.Errorf("Mason-67 billed total on the second call = %d, want %d -- a doubled figure means the resolver accumulated", got, resolverClaudeMasonTotal)
	}

	t.Run("the same worker named twice in one request is answered once", func(t *testing.T) {
		dup := req
		dup.WorkerNames = []string{"Mason-67", "Mason-67"}
		res := resolveWrapperWorkerUsage(dup)
		if len(res.Workers) != 1 {
			t.Fatalf("got %d rows for one worker named twice, want 1: %+v", len(res.Workers), res.Workers)
		}
		if got := res.Workers[0].Usage.BilledTotalTokens(); got != resolverClaudeMasonTotal {
			t.Errorf("billed total = %d, want %d", got, resolverClaudeMasonTotal)
		}
	})
}

// TestClaudeTranscriptJoinsOnWhatThePlatformRecords is CR-01.
//
// The Claude reader names a subagent row by the agent DEFINITION the wrapper
// passed as `subagent_type` ("aether-builder"). The ledger accounts a worker
// under its own deterministic name ("Mason-67"). Those two strings can never be
// equal, so before this test existed NO worker's tokens were attributed on
// Claude Code -- the repository's own primary platform -- and the owner's cost
// block read "Cost: not known" on every build.
//
// The join this locks is the one the platform itself records: the tool_use block
// that dispatched the worker carries BOTH the description the wrapper composed
// (which contains the deterministic name, by the contract in
// .claude/commands/ant/build.md: "Use the exact visible description:
// {caste emoji} {Caste} {name}: {task}") AND the subagent type, and the
// completion record's tool_use_id points straight back at it.
func TestClaudeTranscriptJoinsOnWhatThePlatformRecords(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s
	newResolverClaudeTranscript(t)

	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:          "claude-code",
		RepoRoot:          t.TempDir(),
		StartedAt:         time.Now().Add(-time.Hour),
		WorkerNames:       []string{"Mason-67", "Vigil-12", "Roam-90"},
		AgentNameByWorker: resolverClaudeAgentNames(),
	})

	mason := resolverWorker(t, res, "Mason-67")
	if !mason.Reported {
		t.Fatalf("Mason-67 got no figure. The transcript records its dispatch as agentType %q with the deterministic name only in the dispatch description, which is what Claude Code really writes; a resolver that only compares the accounting key against agentType attributes nothing on this platform at all. Diagnostics: %v",
			"aether-builder", res.Diagnostics)
	}
	if got := mason.Usage.BilledTotalTokens(); got != resolverClaudeMasonTotal {
		t.Errorf("Mason-67 billed total = %d, want %d", got, resolverClaudeMasonTotal)
	}

	vigil := resolverWorker(t, res, "Vigil-12")
	if !vigil.Reported || vigil.Usage.BilledTotalTokens() != resolverClaudeVigilTotal {
		t.Errorf("Vigil-12 = %+v, want a reported billed total of %d", vigil, resolverClaudeVigilTotal)
	}

	// Roam-90 was dispatched as the same agent definition Stray-99 ran as. The
	// transcript's own description says the row belongs to Stray-99, so Roam-90
	// must get NOTHING -- a definition-only join would hand it 777,777 tokens it
	// never spent.
	roam := resolverWorker(t, res, "Roam-90")
	if roam.Reported {
		t.Errorf("Roam-90 was credited %d tokens from a row the transcript says belongs to Stray-99; two workers sharing one agent definition must never be resolved by that definition when the dispatch record names one of them",
			roam.Usage.BilledTotalTokens())
	}

	stray := false
	for _, d := range res.Diagnostics {
		if strings.Contains(d, "Stray-99") {
			stray = true
		}
	}
	if !stray {
		t.Errorf("a transcript row for a worker this run never dispatched was silently dropped; expected a plain-English note naming Stray-99, got %v", res.Diagnostics)
	}

	t.Run("a dispatch that carried no description still resolves by its definition when only one worker ran as it", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		projectDir := filepath.Join(home, ".claude", "projects", "-nodesc")
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		path := filepath.Join(projectDir, "sess.jsonl")
		lines := resolverSubagentDispatch(resolverSubagentFixture{
			ToolUseID:       "toolu_a",
			SubagentType:    "aether-builder",
			AgentID:         "d4e5f60718293041",
			OmitDescription: true,
			In:              500,
		})
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write transcript: %v", err)
		}
		if err := store.SaveJSON(spendSessionRel, spendSessionRecord{
			SchemaVersion:  spendSessionSchemaVersion,
			Platform:       "claude-code",
			SessionID:      "sess-nodesc",
			TranscriptPath: path,
			Cwd:            t.TempDir(),
			CapturedAt:     time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save session record: %v", err)
		}

		res := resolveWrapperWorkerUsage(wrapperUsageRequest{
			Platform:          "claude-code",
			RepoRoot:          t.TempDir(),
			StartedAt:         time.Now().Add(-time.Hour),
			WorkerNames:       []string{"Anvil-20"},
			AgentNameByWorker: map[string]string{"Anvil-20": "aether-builder"},
		})
		anvil := resolverWorker(t, res, "Anvil-20")
		if !anvil.Reported || anvil.Usage.BilledTotalTokens() != 500 {
			t.Errorf("Anvil-20 = %+v, want 500 tokens: it is the only worker this run dispatched as aether-builder, so the definition identifies it unambiguously", anvil)
		}
	})

	t.Run("two workers sharing one definition and no description resolve to neither", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		projectDir := filepath.Join(home, ".claude", "projects", "-ambiguous")
		if err := os.MkdirAll(projectDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		path := filepath.Join(projectDir, "sess.jsonl")
		lines := resolverSubagentDispatch(resolverSubagentFixture{
			ToolUseID:       "toolu_x",
			SubagentType:    "aether-builder",
			AgentID:         "e5f6071829304152",
			OmitDescription: true,
			In:              900,
		})
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write transcript: %v", err)
		}
		if err := store.SaveJSON(spendSessionRel, spendSessionRecord{
			SchemaVersion:  spendSessionSchemaVersion,
			Platform:       "claude-code",
			SessionID:      "sess-ambiguous",
			TranscriptPath: path,
			Cwd:            t.TempDir(),
			CapturedAt:     time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save session record: %v", err)
		}

		res := resolveWrapperWorkerUsage(wrapperUsageRequest{
			Platform:    "claude-code",
			RepoRoot:    t.TempDir(),
			StartedAt:   time.Now().Add(-time.Hour),
			WorkerNames: []string{"Anvil-20", "Mason-67"},
			AgentNameByWorker: map[string]string{
				"Anvil-20": "aether-builder",
				"Mason-67": "aether-builder",
			},
		})
		for _, name := range []string{"Anvil-20", "Mason-67"} {
			assertNoFigure(t, resolverWorker(t, res, name))
		}
		refused := false
		for _, d := range res.Diagnostics {
			if strings.Contains(d, "aether-builder") && strings.Contains(d, "Anvil-20") && strings.Contains(d, "Mason-67") {
				refused = true
			}
		}
		if !refused {
			t.Errorf("two workers ran as one agent definition with nothing to tell them apart and the resolver said nothing about it; the owner must be able to see why neither has a figure. Got: %v", res.Diagnostics)
		}
	})
}

// TestTheUnmatchedRecordNoteDoesNotBlameAWorkerThatRan is NEW-03.
//
// When a dispatch description names none of the run's workers, the note said
// the transcript "holds usage for a worker this run did not dispatch". The code
// cannot know that. The far likelier cause is that the wrapper paraphrased the
// description instead of using the required "{caste emoji} {Caste} {name}:
// {task}" form -- the description is composed by a model following a markdown
// instruction, not by the runtime. The owner was told a stranger's spend was
// seen, when the truth is that his own worker's spend was dropped, and the
// message pointed away from the real cause.
func TestTheUnmatchedRecordNoteDoesNotBlameAWorkerThatRan(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	home := t.TempDir()
	t.Setenv("HOME", home)
	projectDir := filepath.Join(home, ".claude", "projects", "-paraphrased")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(projectDir, "sess.jsonl")
	// The worker WAS dispatched. Only the label the wrapper wrote is wrong.
	lines := resolverSubagentDispatch(resolverSubagentFixture{
		ToolUseID:    "toolu_paraphrased",
		Description:  "Implement the parser",
		SubagentType: "aether-builder",
		AgentID:      "3333333333333333",
		In:           64_000,
	})
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := store.SaveJSON(spendSessionRel, spendSessionRecord{
		SchemaVersion:  spendSessionSchemaVersion,
		Platform:       "claude-code",
		SessionID:      "sess-paraphrased",
		TranscriptPath: path,
		Cwd:            t.TempDir(),
		CapturedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save session record: %v", err)
	}

	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:  "claude-code",
		RepoRoot:  t.TempDir(),
		StartedAt: time.Now().Add(-time.Hour),
		EndedAt:   time.Now().Add(time.Minute),
		// Two workers share the definition, so the definition rule cannot
		// answer and the description is the only evidence there is.
		WorkerNames: []string{"Mason-67", "Anvil-20"},
		AgentNameByWorker: map[string]string{
			"Mason-67": "aether-builder",
			"Anvil-20": "aether-builder",
		},
	})

	note := strings.Join(res.Diagnostics, " | ")
	if strings.Contains(note, "did not dispatch") {
		t.Errorf("the owner is told his run did not dispatch a worker it DID dispatch; the label on the dispatch is what did not match, and a message that points at the wrong cause is worse than a vague one. Got: %s", note)
	}
	if !strings.Contains(note, "Implement the parser") {
		t.Errorf("the note does not quote the label that failed to match, so the owner cannot see what to correct. Got: %s", note)
	}
	if !strings.Contains(note, "could not be matched") {
		t.Errorf("the note does not say what actually happened -- a token record that could not be matched to any worker on this run. Got: %s", note)
	}
}

// dispatchDescriptionContractSurfaces are the wrapper files that tell the
// orchestrating model what to write in a dispatch's visible description. That
// string is the ONLY place a worker's own name reaches Claude Code's
// transcript, so it is the only thing the token ledger can join a transcript
// row to a worker on.
var dispatchDescriptionContractSurfaces = []string{
	".claude/commands/ant/build.md",
	".opencode/commands/ant/build.md",
	".claude/commands/ant/continue.md",
	".opencode/commands/ant/continue.md",
}

// dispatchDescriptionFormat is the contract itself, quoted exactly as the
// wrappers must carry it.
const dispatchDescriptionFormat = "{caste emoji} {Caste} {name}: {task}"

// dispatchDescriptionInstruction is the spawning step that carries it. The
// wrappers mention the format elsewhere too, so the assertion is made against
// THIS line rather than the file, which is what makes it bite.
const dispatchDescriptionInstruction = "Use the exact visible description:"

// TestDispatchDescriptionCarriesTheAccountingKey is NEW-04.
//
// After CR-01's fix, whether a worker's tokens are attributed on this
// repository's primary platform turns entirely on the dispatch description
// containing the worker's own name. That contract lived in one line of wrapper
// markdown that nothing read and no test asserted. Claude Code documents the
// Task tool's description as a short label, so the pressure to shorten it is
// real -- and shortening it to "{Caste}: {task}" would silently return the whole
// subsystem to reporting "Cost: not known" on every build, with the full suite
// green. That is the Definition of Done's "a documentation claim about runtime
// behaviour must be testable or removed", applied to a claim the runtime now
// depends on.
//
// The last assertion is the one that makes this more than a string check: it
// builds a description in the documented shape from a name the RUNTIME
// generates and puts it through the matcher the resolver really uses.
func TestDispatchDescriptionCarriesTheAccountingKey(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	if len(dispatchDescriptionContractSurfaces) != 4 {
		t.Fatalf("the contract lives on 4 wrapper surfaces, this list has %d", len(dispatchDescriptionContractSurfaces))
	}

	for _, rel := range dispatchDescriptionContractSurfaces {
		body, readErr := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if readErr != nil {
			t.Fatalf("read %s: %v", rel, readErr)
		}
		text := string(body)
		if strings.TrimSpace(text) == "" {
			t.Fatalf("%s is empty, so every assertion below would pass over nothing", rel)
		}
		// The assertion is made against the DISPATCH INSTRUCTION LINE, not the
		// file as a whole. These wrappers mention the format more than once,
		// so a whole-file substring check stays green while the one line the
		// spawning step actually follows is shortened -- measured: shortening
		// step 4 of build.md left a file-wide check passing.
		instructions := 0
		for _, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, dispatchDescriptionInstruction) {
				continue
			}
			instructions++
			if !strings.Contains(line, dispatchDescriptionFormat) {
				t.Errorf("%s tells the spawning step to use a visible description that no longer carries the worker's own name:\n  %s\nThat name is the only thing the token record can join a Claude Code transcript row to a worker on, so dropping it reports every build as costing nothing while every test stays green.", rel, strings.TrimSpace(line))
			}
		}
		if instructions == 0 {
			t.Errorf("%s no longer tells the spawning step what visible description to use (%q), so nothing requires the worker's own name to reach the transcript at all", rel, dispatchDescriptionInstruction)
		}
		// The consequence is written beside the instruction, so the next
		// person to shorten it can see what it costs before they do.
		if !strings.Contains(text, "joins a transcript row to a worker") {
			t.Errorf("%s carries the description format but does not say what depends on it; a future editor shortening that line has nothing telling them it turns off every build's cost figure.", rel)
		}
	}

	t.Run("a description in the documented shape resolves to the worker the runtime named", func(t *testing.T) {
		// Derived the way production derives it, not typed as a literal: this
		// is the same generator and the same seed the build planner uses.
		name := deterministicAntName("builder", "phase:9:builder")
		description := strings.NewReplacer(
			"{caste emoji}", "🔨🐜",
			"{Caste}", "Builder",
			"{name}", name,
			"{task}", "implement the parser",
		).Replace(dispatchDescriptionFormat)
		if !workerNameAppearsIn(description, name) {
			t.Errorf("a dispatch description written exactly as the wrappers require (%q) does not name the worker the runtime generated (%q), so nothing on this platform can be attributed", description, name)
		}
	})
}

// TestOpenCodeRefusalsReachTheOwner is WR-01.
//
// openCodeSessionUsageForRun returns diagnostics as its second value and
// documents them as the point of the design — "a partial, honestly-reported
// result is the correct outcome here". Its only caller discarded them, so all
// three of its carefully written notes were unreachable in production. The one
// that matters most: when two sessions share a worker's name the reader resolves
// nothing for that worker and says "refusing to guess", and the owner was told
// nothing at all — the row simply read "not reported", indistinguishable from a
// tool that genuinely said nothing.
//
// TestOpenCodeSessionUsageRefusesAmbiguousWorkerMatch calls the inner function
// directly, so it never saw the drop.
func TestOpenCodeRefusalsReachTheOwner(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, _ := newTestStore(t)
	store = s

	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)
	// A second in-window session carrying the same worker's name.
	duplicate := filepath.Join(storageRoot, "session", "prj_fixture", "ses_child_a_dup.json")
	if err := os.WriteFile(duplicate, []byte(`{"id":"ses_child_a_dup","parentID":"ses_parent","title":"🔨 Builder Mason-67: implement the parser (@general subagent)","time":{"created":1786710600000,"updated":1786710600000}}`), 0o644); err != nil {
		t.Fatalf("write duplicate session: %v", err)
	}

	res := resolveWrapperWorkerUsage(wrapperUsageRequest{
		Platform:    "opencode",
		RepoRoot:    repoRoot,
		StartedAt:   openCodeFixtureWindowStart,
		EndedAt:     openCodeFixtureWindowEnd,
		WorkerNames: []string{"Mason-67"},
	})

	assertNoFigure(t, resolverWorker(t, res, "Mason-67"))

	said := false
	for _, d := range res.Diagnostics {
		if strings.Contains(d, "Mason-67") && strings.Contains(d, "refusing to guess") {
			said = true
		}
	}
	if !said {
		t.Errorf("Mason-67 has no figure because two sessions carry its name, and nothing said so; an owner must be able to see why a worker has no figure. Got: %v", res.Diagnostics)
	}
}
