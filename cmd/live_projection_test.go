package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/calcosmic/Aether/pkg/events"
)

// liveEmissionViolation names one function, and its position, that calls
// Publish with a live.* topic outside the one allowed emission boundary.
type liveEmissionViolation struct {
	Function string
	Position string
}

// callPublishesLiveTopic reports whether call is a Publish(...) invocation
// carrying a live.* topic argument, recognizing both a literal topic string
// present in the live vocabulary returned by events.ColonyLiveTopics() (the
// form a careless direct call would most plausibly use) and a selector
// referencing one of the LiveTopic* constants (the form real emission code
// uses).
func callPublishesLiveTopic(call *ast.CallExpr, liveTopics map[string]bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Publish" {
		return false
	}
	for _, arg := range call.Args {
		switch a := arg.(type) {
		case *ast.BasicLit:
			if a.Kind == token.STRING {
				if unquoted, err := strconv.Unquote(a.Value); err == nil && liveTopics[unquoted] {
					return true
				}
			}
		case *ast.SelectorExpr:
			if strings.HasPrefix(a.Sel.Name, "LiveTopic") {
				return true
			}
		}
	}
	return false
}

// findLiveEmissionBoundaryViolations walks every function declared across
// files and reports each Publish call site carrying a live.* topic whose
// enclosing function is not allowedCaller. The topic set it checks against
// is liveTopics -- derived from events.ColonyLiveTopics(), never a literal
// list maintained in this test.
func findLiveEmissionBoundaryViolations(fset *token.FileSet, files []*ast.File, allowedCaller string, liveTopics map[string]bool) []liveEmissionViolation {
	var violations []liveEmissionViolation
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			funcName := fn.Name.Name
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if !callPublishesLiveTopic(call, liveTopics) {
					return true
				}
				if funcName != allowedCaller {
					violations = append(violations, liveEmissionViolation{
						Function: funcName,
						Position: fset.Position(call.Pos()).String(),
					})
				}
				return true
			})
		}
	}
	return violations
}

// TestEveryLiveEventGoesThroughOneBoundary proves, from the parsed syntax
// tree rather than from review, that emitColonyLive (cmd/live_events.go) is
// the only function in the cmd package permitted to publish a live.* topic.
//
// The topic set the guard checks against is read from
// events.ColonyLiveTopics() -- a topic added later is covered
// automatically, with no update needed here.
func TestEveryLiveEventGoesThroughOneBoundary(t *testing.T) {
	const allowedCaller = "emitColonyLive"

	liveTopics := map[string]bool{}
	for _, topic := range events.ColonyLiveTopics() {
		liveTopics[topic] = true
	}

	fset := token.NewFileSet()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}

	var realFiles []*ast.File
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		realFiles = append(realFiles, file)
	}

	t.Run("real package has zero violations", func(t *testing.T) {
		violations := findLiveEmissionBoundaryViolations(fset, realFiles, allowedCaller, liveTopics)
		if len(violations) != 0 {
			t.Fatalf("found live-topic Publish call(s) outside %s: %+v", allowedCaller, violations)
		}
		// Ceremony emitters share the same Publish signature but publish to
		// ceremony.* topics, never live.* -- confirm they are genuinely
		// unflagged rather than absent from the scan by construction.
		foundCeremonyFile := false
		for _, name := range names {
			if name == "ceremony_emitter.go" {
				foundCeremonyFile = true
			}
		}
		if !foundCeremonyFile {
			t.Fatal("fixture is broken: cmd/ceremony_emitter.go was not part of the scanned file set")
		}
	})

	t.Run("a fixture function publishing a live topic directly is reported by name and position", func(t *testing.T) {
		const fixtureSrc = `package cmd

import "github.com/calcosmic/Aether/pkg/events"

func sneakyLivePublisher() {
	var bus interface {
		Publish(topic string, payload []byte) error
	}
	_ = bus.Publish("live.worker.started", nil)
	_ = events.ColonyLiveSchemaVersion
}
`
		fixtureFset := token.NewFileSet()
		fixtureFile, err := parser.ParseFile(fixtureFset, "fixture_violation.go", fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse synthetic fixture: %v", err)
		}

		violations := findLiveEmissionBoundaryViolations(fixtureFset, []*ast.File{fixtureFile}, allowedCaller, liveTopics)
		if len(violations) != 1 {
			t.Fatalf("expected exactly 1 violation from the fixture, got %d: %+v", len(violations), violations)
		}
		if violations[0].Function != "sneakyLivePublisher" {
			t.Fatalf("violation named function %q, want %q", violations[0].Function, "sneakyLivePublisher")
		}
		if !strings.Contains(violations[0].Position, "fixture_violation.go") {
			t.Fatalf("violation position %q does not name the fixture file", violations[0].Position)
		}
	})
}

// TestLiveSequenceNumbersAreMonotonicPerEpisode proves nextLiveSequence
// (cmd/live_events.go) assigns strictly increasing sequence numbers within
// one episode, and that two episodes emitting concurrently each keep their
// own independent increasing series -- asserted under -race so the shared
// counter map's own concurrency safety is exercised, not just its output.
func TestLiveSequenceNumbersAreMonotonicPerEpisode(t *testing.T) {
	const iterations = 200
	episodeA := "episode-A-" + t.Name()
	episodeB := "episode-B-" + t.Name()

	seenA := make([]int64, iterations)
	seenB := make([]int64, iterations)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			seenA[i] = nextLiveSequence(episodeA)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			seenB[i] = nextLiveSequence(episodeB)
		}
	}()
	wg.Wait()

	assertStrictlyIncreasingSequence(t, "episode A", seenA)
	assertStrictlyIncreasingSequence(t, "episode B", seenB)

	// The two series must be independent: episode A's final value must not
	// have been inflated by episode B's emissions, and vice versa.
	if seenA[len(seenA)-1] != int64(iterations) {
		t.Fatalf("episode A's final sequence number = %d, want exactly %d (its own emission count, uncontaminated by episode B)", seenA[len(seenA)-1], iterations)
	}
	if seenB[len(seenB)-1] != int64(iterations) {
		t.Fatalf("episode B's final sequence number = %d, want exactly %d (its own emission count, uncontaminated by episode A)", seenB[len(seenB)-1], iterations)
	}
}

func assertStrictlyIncreasingSequence(t *testing.T, label string, seq []int64) {
	t.Helper()
	if len(seq) == 0 {
		t.Fatalf("%s: fixture is broken, no sequence numbers captured", label)
	}
	if seq[0] < 1 {
		t.Fatalf("%s: sequence should start at >= 1, got %d", label, seq[0])
	}
	for i := 1; i < len(seq); i++ {
		if seq[i] <= seq[i-1] {
			t.Fatalf("%s: sequence not strictly increasing at index %d: %d <= %d", label, i, seq[i], seq[i-1])
		}
	}
}
