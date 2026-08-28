package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// The fixture's fixed timestamps (see cmd/testdata/spend/opencode/README.md):
// ses_child_a and ses_child_b sit inside this window, ses_stale sits well
// before it.
var (
	openCodeFixtureWindowStart = time.UnixMilli(1786708800000)
	openCodeFixtureWindowEnd   = time.UnixMilli(1786712400000)
	openCodeFixtureStaleTime   = time.UnixMilli(1786352400000)
)

// copyOpenCodeFixture copies the committed OpenCode storage fixture into
// destRoot, rewriting the __REPO_ROOT__ placeholder to repoRoot so worktree
// discovery is exercised for real rather than stubbed. No test in this file
// reads the developer's real OpenCode store -- every test sets HOME to a
// t.TempDir() first (grep -c 'os.UserHomeDir()' on this file returns 0: the
// production code resolves HOME, this file never does).
func copyOpenCodeFixture(t *testing.T, destRoot, repoRoot string) {
	t.Helper()
	srcRoot := filepath.Join("testdata", "spend", "opencode")
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(srcRoot, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." || rel == "README.md" {
			return nil
		}
		destPath := filepath.Join(destRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), "__REPO_ROOT__") {
			data = []byte(strings.ReplaceAll(string(data), "__REPO_ROOT__", repoRoot))
		}
		if mkErr := os.MkdirAll(filepath.Dir(destPath), 0o755); mkErr != nil {
			return mkErr
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy opencode fixture: %v", err)
	}
}

func setupOpenCodeFixtureHome(t *testing.T) (storageRoot, repoRoot string) {
	t.Helper()
	home := t.TempDir()
	repoRoot = t.TempDir()
	t.Setenv("HOME", home)
	storageRoot = filepath.Join(home, ".local", "share", "opencode", "storage")
	copyOpenCodeFixture(t, storageRoot, repoRoot)
	return storageRoot, repoRoot
}

func TestOpenCodeSessionUsageReadsDisjointTokenColumns(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67", "Vigil-12"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(entries), entries)
	}

	var mason, vigil *openCodeSessionUsage
	for i := range entries {
		switch entries[i].WorkerName {
		case "Mason-67":
			mason = &entries[i]
		case "Vigil-12":
			vigil = &entries[i]
		}
	}
	if mason == nil {
		t.Fatalf("Mason-67 entry not found: %+v", entries)
	}
	// Literal figures hand-summed from the committed fixture's two assistant
	// messages (msg_1.json + msg_2.json under message/ses_child_a/), never
	// recomputed -- the four disjoint columns must survive intact and apart.
	//   input  543 +  1201 =  1744
	//   output 123 +   806 =   929
	//   read 19770 + 30500 = 50270
	//   write    0 +  4096 =  4096
	//   total 20436 + 36603 = 57039
	if mason.Usage.InputTokens != 1744 {
		t.Errorf("mason InputTokens = %d, want 1744", mason.Usage.InputTokens)
	}
	if mason.Usage.OutputTokens != 929 {
		t.Errorf("mason OutputTokens = %d, want 929", mason.Usage.OutputTokens)
	}
	if mason.Usage.CachedInputTokens != 50270 {
		t.Errorf("mason CachedInputTokens = %d, want 50270", mason.Usage.CachedInputTokens)
	}
	if mason.Usage.CacheCreationTokens != 4096 {
		t.Errorf("mason CacheCreationTokens = %d, want 4096", mason.Usage.CacheCreationTokens)
	}
	if mason.Usage.TotalTokens != 57039 {
		t.Errorf("mason TotalTokens = %d, want 57039", mason.Usage.TotalTokens)
	}
	if mason.Usage.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("mason Source = %q, want %q", mason.Usage.Source, codex.UsageSourceSessionTranscript)
	}

	if vigil == nil {
		t.Fatalf("Vigil-12 entry not found: %+v", entries)
	}
	if vigil.Usage.TotalTokens != 8100 {
		t.Errorf("vigil TotalTokens = %d, want 8100", vigil.Usage.TotalTokens)
	}
	if vigil.Usage.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("vigil Source = %q, want %q", vigil.Usage.Source, codex.UsageSourceSessionTranscript)
	}

	for _, e := range entries {
		if e.Usage.Source == codex.UsageSourceProvider {
			t.Errorf("entry %q tagged provider-grade; must never be", e.WorkerName)
		}
	}
}

func TestOpenCodeSessionUsageIgnoresSessionsOutsideTheRunWindow(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	// ses_stale carries the same worker name (Mason-67) but sits outside
	// the window -- without the time bound this would be ambiguous.
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want exactly 1 (stale decoy must be excluded by the window): %+v", len(entries), entries)
	}
	if entries[0].SessionID != "ses_child_a" {
		t.Fatalf("resolved session = %q, want ses_child_a", entries[0].SessionID)
	}
	if entries[0].Usage.TotalTokens == 999999 || entries[0].Usage.InputTokens == 999999 {
		t.Fatalf("stale session's impossible 999999 figure leaked through: %+v", entries[0].Usage)
	}
}

func TestOpenCodeSessionUsageRefusesAmbiguousWorkerMatch(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	// Widen the window to enclose both ses_child_a and ses_stale, so
	// Mason-67 now has two in-window candidates.
	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureStaleTime, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0 for an ambiguous match: %+v", len(entries), entries)
	}
	found := false
	for _, r := range reasons {
		if strings.Contains(r, "Mason-67") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a diagnostic naming Mason-67, got: %v", reasons)
	}
}

func TestOpenCodeSessionUsageIgnoresNonMatchingWorktree(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)
	otherRoot := t.TempDir()
	if otherRoot == repoRoot {
		t.Fatalf("test setup collision: otherRoot must differ from repoRoot")
	}

	entries, reasons := openCodeSessionUsageForRun(storageRoot, otherRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67", "Vigil-12"})
	if len(entries) != 0 {
		t.Fatalf("got %d entries for a non-matching worktree, want 0: %+v", len(entries), entries)
	}
	if len(reasons) == 0 {
		t.Fatalf("expected a diagnostic naming the non-matching repo root")
	}
}

func TestOpenCodeSessionUsageRefusesStorageRootOutsideHome(t *testing.T) {
	home := t.TempDir()
	repoRoot := t.TempDir()
	t.Setenv("HOME", home)

	outsideRoot := t.TempDir() // sibling temp dir, not under $HOME/.local/share/opencode/storage
	entries, reasons := openCodeSessionUsageForRun(outsideRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(entries) != 0 {
		t.Fatalf("got %d entries for an out-of-root storage path, want 0", len(entries))
	}
	if len(reasons) == 0 {
		t.Fatalf("expected a diagnostic rejecting the out-of-root storage path")
	}

	// An absent storage root (no OpenCode installed at all) must fail
	// soft, never panic -- this is the normal case on a Claude-Code-only
	// machine.
	freshHome := t.TempDir()
	t.Setenv("HOME", freshHome)
	if _, ok := openCodeSessionUsageForRunOrNone(repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"}); ok {
		t.Fatalf("expected ok=false when the opencode storage root is entirely absent")
	}
}

func TestOpenCodeSessionUsageToleratesMalformedRecords(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	malformedPath := filepath.Join(storageRoot, "session", "prj_fixture", "ses_malformed.json")
	if err := os.WriteFile(malformedPath, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write malformed session file: %v", err)
	}

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67", "Vigil-12"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 despite the malformed sibling session file: %+v", len(entries), entries)
	}
}

// --- Phase 196 plan 05, the salvage fixes ------------------------------------

// TestOpenCodeWorkerNameMatchingIsConcurrencySafe is FIX 2-1.
//
// The salvaged file carried a package-level map that openCodeTitleMatchesWorker
// both read and wrote with no synchronisation. Go's runtime aborts the whole
// process on a concurrent map write -- it is a hard crash, not a silent race --
// and this plan makes usage discovery run per worker, so it would have been a
// crash at the end of every build.
//
// The first subtest is the direct proof and needs -race to see the read/write
// pair. The second subtest states the same rule structurally, so it fails under
// a plain `go test` too: a matcher that shares no package-level mutable state
// cannot have the bug at all.
func TestOpenCodeWorkerNameMatchingIsConcurrencySafe(t *testing.T) {
	t.Run("matching from many goroutines agrees with the single-threaded answer", func(t *testing.T) {
		titles := []string{
			"🔨 Builder Mason-67: implement the parser (@general subagent)",
			"👁️ Watcher Vigil-12: check the parser",
			"🔨 Builder Mason-6: a decoy whose name is a prefix of another",
			"no worker name at all",
		}
		names := []string{"Mason-67", "Mason-6", "Vigil-12", "Keen-90"}

		// The expected table is built by hand, not by calling the matcher.
		want := map[string]bool{
			titles[0] + "|Mason-67": true,
			titles[0] + "|Mason-6":  false,
			titles[0] + "|Vigil-12": false,
			titles[0] + "|Keen-90":  false,
			titles[1] + "|Mason-67": false,
			titles[1] + "|Mason-6":  false,
			titles[1] + "|Vigil-12": true,
			titles[1] + "|Keen-90":  false,
			titles[2] + "|Mason-67": false,
			titles[2] + "|Mason-6":  true,
			titles[2] + "|Vigil-12": false,
			titles[2] + "|Keen-90":  false,
			titles[3] + "|Mason-67": false,
			titles[3] + "|Mason-6":  false,
			titles[3] + "|Vigil-12": false,
			titles[3] + "|Keen-90":  false,
		}

		const goroutines = 32
		const rounds = 40
		var wg sync.WaitGroup
		errs := make(chan string, goroutines*rounds*len(titles)*len(names))
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for r := 0; r < rounds; r++ {
					for _, title := range titles {
						for _, name := range names {
							got := openCodeTitleMatchesWorker(title, name)
							if got != want[title+"|"+name] {
								errs <- fmt.Sprintf("match(%q, %q) = %v, want %v", title, name, got, want[title+"|"+name])
							}
						}
					}
				}
			}()
		}
		wg.Wait()
		close(errs)
		for msg := range errs {
			t.Error(msg)
			break
		}
	})

	t.Run("the opencode reader shares no package-level mutable state", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "wrapper_usage_opencode.go", nil, 0)
		if err != nil {
			t.Fatalf("parse wrapper_usage_opencode.go: %v", err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range value.Names {
					t.Errorf("package-level var %s at %s is shared mutable state in the cost path; "+
						"the unsynchronised worker-name pattern cache lived exactly here and Go aborts the "+
						"process on a concurrent map write (FIX 2-1)", name.Name, fset.Position(name.Pos()))
				}
			}
		}
	})
}

// The two-message fixture session, summed by hand from the committed files.
// ses_child_a/msg_1.json: input 543,  output 123, cache.read 19770, cache.write 0
// ses_child_a/msg_2.json: input 1201, output 806, cache.read 30500, cache.write 4096
//
// Every figure below is written out by hand. Nothing here is produced by
// calling the reader or by calling BilledTotalTokens().
const (
	openCodeFixtureMsg1Total = 20436 //   543 +   123 + 19770 +    0
	openCodeFixtureMsg2Total = 36603 //  1201 +   806 + 30500 + 4096
	openCodeFixtureSumInput  = 1744  //   543 +  1201
	openCodeFixtureSumOutput = 929   //   123 +   806
	openCodeFixtureSumRead   = 50270 // 19770 + 30500
	openCodeFixtureSumWrite  = 4096  //     0 +  4096
	openCodeFixtureSumTotal  = 57039 // 20436 + 36603
)

// TestOpenCodeUsageSumsEveryAssistantMessage is FIX 2-2.
//
// Every fixture session on the salvaged branch held exactly one message, so the
// accumulation across messages -- the only case that occurs in reality, where a
// real session had twenty-three -- was proven by nothing. If OpenCode ever
// reported a running cumulative total per message instead of a per-message
// figure, the old tests would have stayed green while the reported cost
// multiplied by the message count.
func TestOpenCodeUsageSumsEveryAssistantMessage(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(entries), entries)
	}
	got := entries[0].Usage

	if got.TotalTokens == openCodeFixtureMsg1Total {
		t.Fatalf("TotalTokens = %d, which is the FIRST message alone -- the second assistant message in the same session was not accumulated", got.TotalTokens)
	}
	if got.TotalTokens == openCodeFixtureMsg2Total {
		t.Fatalf("TotalTokens = %d, which is the LAST message alone -- later messages are overwriting earlier ones instead of adding to them", got.TotalTokens)
	}
	if got.TotalTokens != openCodeFixtureSumTotal {
		t.Errorf("TotalTokens = %d, want %d (%d + %d)", got.TotalTokens, openCodeFixtureSumTotal, openCodeFixtureMsg1Total, openCodeFixtureMsg2Total)
	}
	if got.InputTokens != openCodeFixtureSumInput {
		t.Errorf("InputTokens = %d, want %d", got.InputTokens, openCodeFixtureSumInput)
	}
	if got.OutputTokens != openCodeFixtureSumOutput {
		t.Errorf("OutputTokens = %d, want %d", got.OutputTokens, openCodeFixtureSumOutput)
	}
	if got.CachedInputTokens != openCodeFixtureSumRead {
		t.Errorf("CachedInputTokens = %d, want %d", got.CachedInputTokens, openCodeFixtureSumRead)
	}
	if got.CacheCreationTokens != openCodeFixtureSumWrite {
		t.Errorf("CacheCreationTokens = %d, want %d", got.CacheCreationTokens, openCodeFixtureSumWrite)
	}
	if got.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("Source = %q, want %q -- a transcript row is never provider-grade", got.Source, codex.UsageSourceSessionTranscript)
	}

	t.Run("the four disjoint columns still add up to the reported total", func(t *testing.T) {
		// Hand-written, not BilledTotalTokens(): this is the same equality
		// that was confirmed against the owner's real 23-message session.
		sum := got.InputTokens + got.OutputTokens + got.CachedInputTokens + got.CacheCreationTokens
		if sum != openCodeFixtureSumTotal {
			t.Errorf("columns sum to %d, but the store's own totals sum to %d", sum, openCodeFixtureSumTotal)
		}
	})

	t.Run("a non-assistant message in the same session contributes nothing", func(t *testing.T) {
		userMsg := filepath.Join(storageRoot, "message", "ses_child_a", "msg_user.json")
		if err := os.WriteFile(userMsg, []byte(`{"role":"user","tokens":{"total":500000,"input":500000,"output":0,"cache":{"read":0,"write":0}}}`), 0o644); err != nil {
			t.Fatalf("write user-role message: %v", err)
		}
		entries, _ := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Mason-67"})
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Usage.TotalTokens != openCodeFixtureSumTotal {
			t.Errorf("TotalTokens = %d, want %d -- a user-role record was counted as spend", entries[0].Usage.TotalTokens, openCodeFixtureSumTotal)
		}
	})
}

// TestSpendPathContainmentHasOneImplementation is FIX 2-5.
//
// Two copies of a security boundary is one copy too many: the salvaged branch
// reimplemented the containment rule already in the session record, and added a
// symlink-evaluation helper without refactoring the original to use it. The
// rule below is executable rather than advisory, per CLAUDE.md's Definition of
// Done -- it fails the moment a second copy appears.
func TestSpendPathContainmentHasOneImplementation(t *testing.T) {
	fset := token.NewFileSet()
	funcs := map[string]*ast.FuncDecl{}
	positions := map[string]string{}
	for _, name := range []string{"spend_session_capture.go", "wrapper_usage_opencode.go", "wrapper_usage_claude.go"} {
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Body == nil {
				continue
			}
			funcs[fn.Name.Name] = fn
			positions[fn.Name.Name] = fset.Position(fn.Pos()).String()
		}
	}

	callsSelector := func(fn *ast.FuncDecl, pkg, sel string) bool {
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			s, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || s.Sel == nil || s.Sel.Name != sel {
				return true
			}
			if ident, ok := s.X.(*ast.Ident); ok && ident.Name == pkg {
				found = true
			}
			return true
		})
		return found
	}
	callsFunc := func(fn *ast.FuncDecl, name string) bool {
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == name {
				found = true
			}
			return true
		})
		return found
	}
	mentionsParentDir := func(fn *ast.FuncDecl) bool {
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if ok && lit.Kind == token.STRING && lit.Value == `".."` {
				found = true
			}
			return true
		})
		return found
	}

	var containment []string
	var symlinkEval []string
	for name, fn := range funcs {
		if callsSelector(fn, "filepath", "Rel") && mentionsParentDir(fn) {
			containment = append(containment, name)
		}
		if callsSelector(fn, "filepath", "EvalSymlinks") {
			symlinkEval = append(symlinkEval, name)
		}
	}
	sort.Strings(containment)
	sort.Strings(symlinkEval)

	const sharedContainment = "validateSpendContainedPath"
	const sharedSymlinkEval = "evalSpendPathSymlinks"

	if len(containment) != 1 || containment[0] != sharedContainment {
		t.Errorf("the spend path containment rule is implemented in %d place(s) %v, want exactly one named %s; "+
			"two copies of a security boundary is one copy too many (FIX 2-5)", len(containment), containment, sharedContainment)
	}
	if len(symlinkEval) != 1 || symlinkEval[0] != sharedSymlinkEval {
		t.Errorf("symlink evaluation is implemented in %d place(s) %v, want exactly one named %s", len(symlinkEval), symlinkEval, sharedSymlinkEval)
	}

	for _, caller := range []string{"validateSpendTranscriptPath", "validateOpenCodeStoragePath"} {
		fn, ok := funcs[caller]
		if !ok {
			t.Fatalf("%s not found in the parsed files", caller)
		}
		if !callsFunc(fn, sharedContainment) {
			t.Errorf("%s (%s) does not call %s -- both callers must share the one containment helper",
				caller, positions[caller], sharedContainment)
		}
	}
}

// TestBoundedReadOpensThenLimits is FIX 2-4.
//
// The salvaged read stat-ed the path and then read it, leaving a gap between
// the size check and the read. Opening first and limiting the read is the same
// amount of code and has no gap.
func TestBoundedReadOpensThenLimits(t *testing.T) {
	dir := t.TempDir()

	exact := filepath.Join(dir, "exact.json")
	if err := os.WriteFile(exact, []byte("0123456789"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := readBoundedFile(exact, 10)
	if err != nil {
		t.Fatalf("a file of exactly the bound must be read: %v", err)
	}
	if string(data) != "0123456789" {
		t.Errorf("read %q, want the whole file", string(data))
	}

	if _, err := readBoundedFile(exact, 9); err == nil {
		t.Errorf("a file one byte over the bound must be refused")
	}
	if _, err := readBoundedFile(dir, 1024); err == nil {
		t.Errorf("a directory must be refused")
	}
	if _, err := readBoundedFile(filepath.Join(dir, "absent.json"), 1024); err == nil {
		t.Errorf("an absent file must be refused")
	}

	t.Run("the bound is applied by limiting the read, not by a prior stat of the path", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "wrapper_usage_opencode.go", nil, 0)
		if err != nil {
			t.Fatalf("parse wrapper_usage_opencode.go: %v", err)
		}
		var fn *ast.FuncDecl
		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.FuncDecl); ok && d.Name != nil && d.Name.Name == "readBoundedFile" {
				fn = d
			}
		}
		if fn == nil || fn.Body == nil {
			t.Fatalf("readBoundedFile not found")
		}
		seen := map[string]bool{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			s, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || s.Sel == nil {
				return true
			}
			if ident, ok := s.X.(*ast.Ident); ok {
				seen[ident.Name+"."+s.Sel.Name] = true
			}
			return true
		})
		if seen["os.Stat"] {
			t.Errorf("readBoundedFile calls os.Stat on the path before reading it -- that is the check-then-read gap FIX 2-4 removes")
		}
		if seen["os.ReadFile"] {
			t.Errorf("readBoundedFile calls os.ReadFile, which reads the whole file regardless of the bound")
		}
		if !seen["os.Open"] {
			t.Errorf("readBoundedFile does not call os.Open -- the bound must be applied to an already-open file")
		}
		if !seen["io.LimitReader"] {
			t.Errorf("readBoundedFile does not call io.LimitReader -- the bound must limit the read itself")
		}
	})
}

// TestOpenCodeSessionWithNoReadableMessagesIsNotReported is CR-03.
//
// readOpenCodeUsage set its source tag before reading anything, so a session
// matched by title whose message directory is missing or empty came back tagged
// "measured" with every column zero. Every downstream decision keys on that tag
// by design, so the worker was filed as a measurement of zero: its real spend
// silently excluded from the total, the count of workers whose tools reported a
// figure inflated, and the footnote that would have said something was missing
// suppressed.
//
// That is the exact outcome D-01 as amended exists to prevent — a worker whose
// tool reported nothing must show NO number, not a zero, which reads as "this
// worker was free".
func TestOpenCodeSessionWithNoReadableMessagesIsNotReported(t *testing.T) {
	t.Run("the message directory does not exist", func(t *testing.T) {
		storageRoot, _ := setupOpenCodeFixtureHome(t)
		usage := readOpenCodeUsage(storageRoot, "ses_does_not_exist")
		assertOpenCodeUsageIsAbsentNotZero(t, usage)
	})

	t.Run("the message directory exists but holds nothing readable", func(t *testing.T) {
		storageRoot, _ := setupOpenCodeFixtureHome(t)
		empty := filepath.Join(storageRoot, "message", "ses_empty")
		if err := os.MkdirAll(empty, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		// A user-role record only: real, readable, and not a worker's spend.
		if err := os.WriteFile(filepath.Join(empty, "msg_1.json"), []byte(`{"role":"user"}`), 0o644); err != nil {
			t.Fatalf("write message: %v", err)
		}
		assertOpenCodeUsageIsAbsentNotZero(t, readOpenCodeUsage(storageRoot, "ses_empty"))
	})

	t.Run("the worker renders with no figure rather than a zero", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, _ := newTestStore(t)
		store = s
		storageRoot, repoRoot := setupOpenCodeFixtureHome(t)
		// Mason-67's session is matched by title; its messages are then gone.
		if err := os.RemoveAll(filepath.Join(storageRoot, "message", "ses_child_a")); err != nil {
			t.Fatalf("remove message dir: %v", err)
		}

		res := resolveWrapperWorkerUsage(wrapperUsageRequest{
			Platform:    "opencode",
			RepoRoot:    repoRoot,
			StartedAt:   openCodeFixtureWindowStart,
			EndedAt:     openCodeFixtureWindowEnd,
			WorkerNames: []string{"Mason-67"},
		})
		mason := resolverWorker(t, res, "Mason-67")
		if mason.Reported {
			t.Fatalf("Mason-67 is marked as measured with %d tokens although nothing about it could be read; a zero here reads as \"this worker was free\"",
				mason.Usage.BilledTotalTokens())
		}

		ledger := spendLedger{Phase: 3, Workflow: spendWorkflowBuild, Rows: []spendRow{
			{AgentName: "Mason-67", Caste: "builder", Status: "completed", Usage: mason.Usage},
		}}
		totals := computeSpendTotals([]spendLedger{ledger})
		if totals.MeasuredRows != 0 {
			t.Errorf("the phase counts %d workers whose tools reported a figure; nothing was read for this one", totals.MeasuredRows)
		}
		block := stripANSI(renderSpendCostLineFromLedgers([]spendLedger{ledger}))
		if !strings.Contains(block, spendNotReportedFigure) {
			t.Errorf("the cost block shows a number for a worker nothing was read for:\n%s", block)
		}
	})
}

func assertOpenCodeUsageIsAbsentNotZero(t *testing.T, usage codex.WorkerUsage) {
	t.Helper()
	if usage.Source != "" {
		t.Errorf("usage is tagged %q although nothing was read; the tag is what every downstream \"was this reported?\" decision keys on", usage.Source)
	}
	if !usage.Empty() {
		t.Errorf("usage = %+v, want the zero value carrying no figure at all", usage)
	}
}

// The three-thousand-three-hundred session, summed by hand from the committed
// files. Its SECOND message carries every column and no `total` at all — an
// aborted or errored assistant turn is the obvious real candidate.
//
// ses_child_c/msg_1.json: total 3300, input 200, output 100, cache.read 3000, cache.write 0
// ses_child_c/msg_2.json: total ABSENT, input 400, output 300, cache.read 5000, cache.write 1000
//
// Every figure below is written out by hand. Nothing here is produced by calling
// the reader or by calling BilledTotalTokens().
const (
	openCodeMissingTotalColumnsSum = 10000 // 200 + 100 + 3000 + 0 + 400 + 300 + 5000 + 1000
	openCodeMissingTotalStatedOnly = 3300  // what a reader that trusts `total` reports
)

// TestOpenCodeReaderPrefersTheDisjointColumns is WR-03.
//
// The reader accumulated `tokens.total` alongside the four disjoint columns, and
// BilledTotalTokens prefers a positive TotalTokens over the columns. So one
// message record missing `total` shrank that worker's whole reported spend —
// silently, still labelled measured, with nothing anywhere signalling the
// shortfall. That is the 186x undercount's own shape pointed the other way.
func TestOpenCodeReaderPrefersTheDisjointColumns(t *testing.T) {
	storageRoot, repoRoot := setupOpenCodeFixtureHome(t)

	entries, reasons := openCodeSessionUsageForRun(storageRoot, repoRoot, openCodeFixtureWindowStart, openCodeFixtureWindowEnd, []string{"Ledge-33"})
	if len(reasons) != 0 {
		t.Fatalf("unexpected diagnostics: %v", reasons)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(entries), entries)
	}
	got := entries[0].Usage

	if got.BilledTotalTokens() == openCodeMissingTotalStatedOnly {
		t.Fatalf("the worker's billed total = %d, which is the ONE message that stated a total; the other message's columns were read and then thrown away because the store did not restate them as a total",
			got.BilledTotalTokens())
	}
	if got.BilledTotalTokens() != openCodeMissingTotalColumnsSum {
		t.Errorf("billed total = %d, want %d (the four disjoint columns, hand-summed)", got.BilledTotalTokens(), openCodeMissingTotalColumnsSum)
	}
	if got.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0: the store's own per-message total must not be accumulated at all, because reading it alongside the columns is two answers to one question and the consumer prefers the one a single malformed record can shrink",
			got.TotalTokens)
	}
	if got.Source != codex.UsageSourceSessionTranscript {
		t.Errorf("Source = %q, want %q", got.Source, codex.UsageSourceSessionTranscript)
	}
}
