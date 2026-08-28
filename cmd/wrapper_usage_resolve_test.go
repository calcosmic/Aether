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

	assistantLine := func(uuid string) string {
		return `{"type":"assistant","uuid":"` + uuid + `","requestId":"req_1","message":{"id":"msg_1","model":"claude-opus-5","usage":{"input_tokens":1,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}}}`
	}
	subagentLine := func(agentType, agentID string, in, cc, cr, out int64) string {
		return fmt.Sprintf(`{"type":"user","uuid":"u-%s","toolUseResult":{"agentType":%q,"agentId":%q,"resolvedModel":"claude-sonnet-5","usage":{"input_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d,"output_tokens":%d}}}`,
			agentID, agentType, agentID, in, cc, cr, out)
	}

	lines := []string{
		assistantLine("u-1"),
		assistantLine("u-2"), // same message.id -- must be billed once, not twice
		subagentLine("Mason-67", "agent-a", 100, 200, 300, 400),
		subagentLine("Vigil-12", "agent-b", 11, 22, 33, 44),
		subagentLine("Stray-99", "agent-c", 777777, 0, 0, 0), // never dispatched by this run
	}
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
		Platform:    "claude-code",
		RepoRoot:    t.TempDir(),
		StartedAt:   time.Now().Add(-time.Hour),
		WorkerNames: []string{"Mason-67", "Vigil-12", "Roam-90"},
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

	if !res.SessionReported {
		t.Errorf("the orchestrating session's own turns were lost entirely")
	}
	if got := res.SessionUsage.BilledTotalTokens(); got != resolverClaudeSessionTotal {
		t.Errorf("session billed total = %d, want %d (the repeated assistant line shares one message.id and must be billed once)", got, resolverClaudeSessionTotal)
	}
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
			Platform:    "claude-code",
			RepoRoot:    t.TempDir(),
			StartedAt:   time.Now().Add(-time.Hour),
			WorkerNames: []string{"Mason-67"},
			Attached:    map[string]codex.WorkerUsage{"Mason-67": attached},
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
		Platform:    "claude-code",
		RepoRoot:    t.TempDir(),
		StartedAt:   time.Now().Add(-time.Hour),
		WorkerNames: []string{"Mason-67", "Vigil-12", "Roam-90"},
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
	if first.SessionUsage != second.SessionUsage || first.SessionReported != second.SessionReported {
		t.Errorf("session usage changed between calls: %+v then %+v", first.SessionUsage, second.SessionUsage)
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
