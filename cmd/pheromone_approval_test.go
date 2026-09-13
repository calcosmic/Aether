package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// clearSignalQuarantine is a deliberate, new addition to
// pheromoneJSONWriterAllowlist (cmd/pheromone_resolver_test.go,
// TestOnePheromoneWriterOnly) -- a ratchet explicitly designed to be
// extended "deliberately (with a stated reason)" for a new writer. Clearing
// a quarantine flag on an EXISTING signal is a distinct operation from
// writePheromoneSignal's create/reinforce chokepoint (there is nothing to
// dedup or reinforce -- the note already exists), so it is its own single,
// governed writer rather than a second create/reinforce path.
func init() {
	pheromoneJSONWriterAllowlist["clearSignalQuarantine"] = "the one function permitted to release a quarantined signal (approvePendingNote's import branch); a distinct operation from writePheromoneSignal's create/reinforce chokepoint"
}

// --- Task 1: one queue carries suggestions and quarantined imports alike ---

func TestPendingNoteOrigin(t *testing.T) {
	t.Run("legacy item without origin field reads as suggestion", func(t *testing.T) {
		legacy := colony.PendingSuggestion{ID: "sug_1", Type: "FOCUS", Content: "old suggestion", ContentHash: "sha256:abc"}
		if got := pendingNoteOrigin(legacy); got != colony.PendingOriginSuggestion {
			t.Errorf("pendingNoteOrigin(legacy) = %q, want %q", got, colony.PendingOriginSuggestion)
		}
	})

	t.Run("a fixture queue file written before this change loads and every item reads origin suggestion", func(t *testing.T) {
		_, s := setupExchangeTest(t)
		// Hand-written JSON matching the pre-Origin/SignalID shape exactly --
		// no origin/signal_id keys at all, proving a real on-disk legacy
		// fixture (not merely a Go zero-value) still reads correctly.
		raw := `{"version":"1.0","goal":"test","pending_suggestions":[{"id":"sug_legacy","type":"FEEDBACK","content":"legacy text","reason":"legacy reason","content_hash":"sha256:legacy","created_at":"2026-01-01T00:00:00Z","dismissed":false}]}`
		if err := s.AtomicWrite("COLONY_STATE.json", []byte(raw)); err != nil {
			t.Fatalf("seed legacy fixture: %v", err)
		}
		var cs colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &cs); err != nil {
			t.Fatalf("load legacy fixture: %v", err)
		}
		if cs.PendingSuggestions == nil || len(*cs.PendingSuggestions) != 1 {
			t.Fatalf("expected 1 pending suggestion, got %v", cs.PendingSuggestions)
		}
		item := (*cs.PendingSuggestions)[0]
		if item.Origin != nil {
			t.Errorf("legacy item.Origin = %v, want nil (unset on disk)", item.Origin)
		}
		if got := pendingNoteOrigin(item); got != colony.PendingOriginSuggestion {
			t.Errorf("pendingNoteOrigin(legacy fixture item) = %q, want %q", got, colony.PendingOriginSuggestion)
		}
	})

	t.Run("PendingSuggestion gains exactly the two new pointer fields, both omitempty", func(t *testing.T) {
		zero := colony.PendingSuggestion{ID: "x", Type: "FOCUS", Content: "c", ContentHash: "sha256:x", CreatedAt: "now"}
		data, err := json.Marshal(zero)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		for _, key := range []string{`"origin"`, `"signal_id"`} {
			if strings.Contains(string(data), key) {
				t.Errorf("zero-value PendingSuggestion JSON contains %s, want omitted (omitempty)", key)
			}
		}
		if zero.Origin != nil || zero.SignalID != nil {
			t.Errorf("zero-value Origin/SignalID must be nil pointers, got Origin=%v SignalID=%v", zero.Origin, zero.SignalID)
		}
	})
}

func TestPendingNoteQueue(t *testing.T) {
	t.Run("importing one note enqueues one linked quarantined item", func(t *testing.T) {
		tmpDir, s := setupExchangeTest(t)
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<pheromones version="1.0" count="1">
  <signal id="sig_from_elsewhere" type="FOCUS" priority="normal" source="user" created_at="2026-04-01T10:00:00Z" active="true" strength="1.0">
    <content><text>A note from another project</text></content>
  </signal>
</pheromones>`
		xmlFile := filepath.Join(tmpDir, "one.xml")
		if err := os.WriteFile(xmlFile, []byte(xmlContent), 0644); err != nil {
			t.Fatalf("write xml: %v", err)
		}
		rootCmd.SetArgs([]string{"import", "pheromones", xmlFile})
		defer rootCmd.SetArgs([]string{})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("import pheromones failed: %v", err)
		}

		var pf colony.PheromoneFile
		if err := s.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		if len(pf.Signals) != 1 {
			t.Fatalf("expected 1 stored signal, got %d", len(pf.Signals))
		}
		sig := pf.Signals[0]
		if sig.Quarantined == nil || !*sig.Quarantined {
			t.Fatalf("stored signal Quarantined = %v, want true", sig.Quarantined)
		}

		var cs colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &cs); err != nil {
			t.Fatalf("load colony state: %v", err)
		}
		if cs.PendingSuggestions == nil || len(*cs.PendingSuggestions) != 1 {
			t.Fatalf("expected 1 queued item, got %v", cs.PendingSuggestions)
		}
		item := (*cs.PendingSuggestions)[0]
		if pendingNoteOrigin(item) != colony.PendingOriginImport {
			t.Errorf("queued item origin = %q, want %q", pendingNoteOrigin(item), colony.PendingOriginImport)
		}
		if item.SignalID == nil || *item.SignalID != sig.ID {
			t.Errorf("queued item SignalID = %v, want %q", item.SignalID, sig.ID)
		}
	})

	t.Run("importing identical content twice produces one queued item", func(t *testing.T) {
		tmpDir, s := setupExchangeTest(t)
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<pheromones version="1.0" count="1">
  <signal id="sig_dup" type="FOCUS" priority="normal" source="user" created_at="2026-04-01T10:00:00Z" active="true" strength="1.0">
    <content><text>Duplicate imported note</text></content>
  </signal>
</pheromones>`
		xmlFile := filepath.Join(tmpDir, "dup.xml")
		if err := os.WriteFile(xmlFile, []byte(xmlContent), 0644); err != nil {
			t.Fatalf("write xml: %v", err)
		}
		for i := 0; i < 2; i++ {
			rootCmd.SetArgs([]string{"import", "pheromones", xmlFile})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("import pheromones failed (pass %d): %v", i, err)
			}
		}
		rootCmd.SetArgs([]string{})

		var cs colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &cs); err != nil {
			t.Fatalf("load colony state: %v", err)
		}
		active := filterActiveSuggestions(cs.PendingSuggestions)
		if len(active) != 1 {
			t.Fatalf("expected 1 active queued item after importing identical content twice, got %d", len(active))
		}
	})

	t.Run("empty queue lists total zero without error", func(t *testing.T) {
		setupExchangeTest(t)
		rootCmd.SetArgs([]string{"suggest-approve"})
		defer rootCmd.SetArgs([]string{})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("suggest-approve on empty queue failed: %v", err)
		}
	})

	t.Run("listing text carries no untranslated repo words", func(t *testing.T) {
		item := colony.PendingSuggestion{ID: "x", Type: "FOCUS", Content: "c", ContentHash: "sha256:x"}
		imported := colony.PendingOriginImport
		item.Origin = &imported
		label := pendingNoteOriginLabel(item)
		for _, word := range []string{"pheromone", "colony", "quarantine", "quarantined"} {
			if strings.Contains(strings.ToLower(label), word) {
				t.Errorf("origin label %q contains untranslated repo word %q", label, word)
			}
		}
		suggestionItem := colony.PendingSuggestion{ID: "y", Type: "FOCUS", Content: "c", ContentHash: "sha256:y"}
		suggestionLabel := pendingNoteOriginLabel(suggestionItem)
		for _, word := range []string{"pheromone", "colony", "quarantine", "quarantined"} {
			if strings.Contains(strings.ToLower(suggestionLabel), word) {
				t.Errorf("origin label %q contains untranslated repo word %q", suggestionLabel, word)
			}
		}
	})
}

// --- Task 2: accept, edit or reject -- and the only way a quarantine clears ---

func seedQueuedSuggestion(t *testing.T, s interface{ SaveJSON(string, interface{}) error }) colony.PendingSuggestion {
	t.Helper()
	item := colony.PendingSuggestion{
		ID:          "sug_test_1",
		Type:        "FOCUS",
		Content:     "focus on the thing",
		Reason:      "test reason",
		ContentHash: "sha256:" + sha256Sum("focus on the thing"),
		CreatedAt:   "2026-01-01T00:00:00Z",
	}
	cs := colony.ColonyState{PendingSuggestions: &[]colony.PendingSuggestion{item}}
	if err := s.SaveJSON("COLONY_STATE.json", cs); err != nil {
		t.Fatalf("seed colony state: %v", err)
	}
	return item
}

func TestPendingNoteApproval(t *testing.T) {
	t.Run("approving a runtime suggestion writes an active note through the single writer", func(t *testing.T) {
		_, s := setupExchangeTest(t)
		item := seedQueuedSuggestion(t, s)

		result, err := approvePendingNote(item.ID, false)
		if err != nil {
			t.Fatalf("approvePendingNote: %v", err)
		}
		if !result.Found || result.Signal == nil {
			t.Fatalf("expected approval to find the item and produce a signal, got %+v", result)
		}
		if extractText(result.Signal.Content) != item.Content {
			t.Errorf("signal content = %q, want %q", extractText(result.Signal.Content), item.Content)
		}
		if result.Item.Action == nil || *result.Item.Action != colony.PendingActionAccepted {
			t.Errorf("item.Action = %v, want %q", result.Item.Action, colony.PendingActionAccepted)
		}
		if result.Item.ActionAt == nil || *result.Item.ActionAt == "" {
			t.Errorf("item.ActionAt not stamped")
		}
		if !result.Item.Dismissed {
			t.Errorf("approved item.Dismissed = false, want true (no longer pending)")
		}

		var pf colony.PheromoneFile
		if err := s.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		found := false
		for _, sig := range pf.Signals {
			if sig.ID == result.Signal.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("approved signal %q was not persisted to pheromones.json", result.Signal.ID)
		}
	})

	t.Run("approving an import clears quarantine and the resolver then includes it", func(t *testing.T) {
		_, s := setupExchangeTest(t)
		importProvenance := colony.PheromoneProvenanceImport
		sig := colony.PheromoneSignal{
			ID:          "sig_import_1",
			Type:        "FOCUS",
			Content:     json.RawMessage(`{"text":"a note from elsewhere"}`),
			Priority:    "normal",
			Source:      "import",
			CreatedAt:   "2026-01-01T00:00:00Z",
			Active:      true,
			Strength:    floatPtr(1.0),
			ContentHash: strPtr("sha256:import1"),
			Provenance:  &importProvenance,
			Quarantined: boolPtr(true),
		}
		if err := s.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{sig}}); err != nil {
			t.Fatalf("seed pheromones: %v", err)
		}

		queuedID := "sug_import_1"
		signalID := sig.ID
		item := colony.PendingSuggestion{
			ID:          queuedID,
			Type:        "FOCUS",
			Content:     "a note from elsewhere",
			ContentHash: "sha256:import1",
			Origin:      strPtr(colony.PendingOriginImport),
			SignalID:    &signalID,
		}
		if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{PendingSuggestions: &[]colony.PendingSuggestion{item}}); err != nil {
			t.Fatalf("seed colony state: %v", err)
		}

		// Before approval: excluded, reason quarantined.
		var pfBefore colony.PheromoneFile
		if err := s.LoadJSON("pheromones.json", &pfBefore); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		resolvedBefore := resolveEffectivePheromones(&pfBefore, mustParseRFC3339(t, "2026-01-02T00:00:00Z"))
		if len(resolvedBefore) != 1 || resolvedBefore[0].InEffect || resolvedBefore[0].ExcludedReason != pheromoneExcludedQuarantine {
			t.Fatalf("expected the signal excluded with reason %q before approval, got %+v", pheromoneExcludedQuarantine, resolvedBefore)
		}

		result, err := approvePendingNote(queuedID, false)
		if err != nil {
			t.Fatalf("approvePendingNote: %v", err)
		}
		if !result.Found || result.Signal == nil {
			t.Fatalf("expected approval to find the item and its linked signal, got %+v", result)
		}
		if result.Signal.Quarantined == nil || *result.Signal.Quarantined {
			t.Errorf("Signal.Quarantined = %v, want false after approval", result.Signal.Quarantined)
		}

		var pfAfter colony.PheromoneFile
		if err := s.LoadJSON("pheromones.json", &pfAfter); err != nil {
			t.Fatalf("load pheromones after approval: %v", err)
		}
		if len(pfAfter.Signals) != 1 {
			t.Fatalf("approving an import must not write a NEW signal, got %d signals", len(pfAfter.Signals))
		}
		resolvedAfter := resolveEffectivePheromones(&pfAfter, mustParseRFC3339(t, "2026-01-02T00:00:00Z"))
		if len(resolvedAfter) != 1 || !resolvedAfter[0].InEffect {
			t.Fatalf("expected the signal in effect after approval, got %+v", resolvedAfter)
		}
	})

	t.Run("editing a queued item stores the owner's wording and recomputes the hash", func(t *testing.T) {
		_, s := setupExchangeTest(t)
		item := seedQueuedSuggestion(t, s)

		result, err := editPendingNote(item.ID, "the owner's replacement wording", false)
		if err != nil {
			t.Fatalf("editPendingNote: %v", err)
		}
		if !result.Found {
			t.Fatalf("expected editPendingNote to find the item")
		}
		if result.Item.Content != "the owner's replacement wording" {
			t.Errorf("item.Content = %q, want owner's wording", result.Item.Content)
		}
		wantHash := "sha256:" + sha256Sum("the owner's replacement wording")
		if result.Item.ContentHash != wantHash {
			t.Errorf("item.ContentHash = %q, want %q", result.Item.ContentHash, wantHash)
		}
		if result.Item.Action == nil || *result.Item.Action != colony.PendingActionEdited {
			t.Errorf("item.Action = %v, want %q", result.Item.Action, colony.PendingActionEdited)
		}

		var cs colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &cs); err != nil {
			t.Fatalf("load colony state: %v", err)
		}
		if cs.PendingSuggestions == nil || len(*cs.PendingSuggestions) != 1 || (*cs.PendingSuggestions)[0].Content != "the owner's replacement wording" {
			t.Fatalf("edited content was not persisted: %v", cs.PendingSuggestions)
		}
	})

	t.Run("rejecting a queued item marks it dismissed and leaves a linked signal quarantined", func(t *testing.T) {
		_, s := setupExchangeTest(t)
		importProvenance := colony.PheromoneProvenanceImport
		signalID := "sig_reject_1"
		sig := colony.PheromoneSignal{
			ID: signalID, Type: "FOCUS", Content: json.RawMessage(`{"text":"reject me"}`),
			Source: "import", Active: true, Strength: floatPtr(1.0),
			ContentHash: strPtr("sha256:rej1"), Provenance: &importProvenance, Quarantined: boolPtr(true),
		}
		if err := s.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{sig}}); err != nil {
			t.Fatalf("seed pheromones: %v", err)
		}
		item := colony.PendingSuggestion{
			ID: "sug_reject_1", Type: "FOCUS", Content: "reject me", ContentHash: "sha256:rej1",
			Origin: strPtr(colony.PendingOriginImport), SignalID: &signalID,
		}
		if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{PendingSuggestions: &[]colony.PendingSuggestion{item}}); err != nil {
			t.Fatalf("seed colony state: %v", err)
		}

		result, err := rejectPendingNote(item.ID, false)
		if err != nil {
			t.Fatalf("rejectPendingNote: %v", err)
		}
		if !result.Found || !result.Item.Dismissed {
			t.Fatalf("expected item found and dismissed, got %+v", result)
		}
		if result.Item.Action == nil || *result.Item.Action != colony.PendingActionRejected {
			t.Errorf("item.Action = %v, want %q", result.Item.Action, colony.PendingActionRejected)
		}

		var pf colony.PheromoneFile
		if err := s.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("load pheromones: %v", err)
		}
		if pf.Signals[0].Quarantined == nil || !*pf.Signals[0].Quarantined {
			t.Errorf("rejected item's linked signal Quarantined = %v, want still true", pf.Signals[0].Quarantined)
		}
	})
}

func TestApprovalDryRunDoesNotMutate(t *testing.T) {
	_, s := setupExchangeTest(t)
	item := seedQueuedSuggestion(t, s)

	dataDir := s.BasePath()
	hashBefore := hashDirFilesForTest(t, dataDir)
	writesBefore := pendingNoteWriteCount

	result, err := approvePendingNote(item.ID, true)
	if err != nil {
		t.Fatalf("approvePendingNote (dry-run): %v", err)
	}
	if !result.Found || !result.WouldApply {
		t.Fatalf("expected a dry-run preview, got %+v", result)
	}

	if pendingNoteWriteCount != writesBefore {
		t.Errorf("pendingNoteWriteCount changed during dry-run: before=%d after=%d", writesBefore, pendingNoteWriteCount)
	}
	hashAfter := hashDirFilesForTest(t, dataDir)
	if hashBefore != hashAfter {
		t.Errorf("dry-run mutated the store: file set/contents changed under %s", dataDir)
	}
}

// hashDirFilesForTest returns a stable, order-independent digest of every
// regular file's contents under dir, so a dry-run's "writes nothing" claim
// is checked against the real filesystem, not just an in-memory counter.
func hashDirFilesForTest(t *testing.T, dir string) string {
	t.Helper()
	var parts []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		parts = append(parts, rel+":"+string(data))
		return nil
	})
	sort.Strings(parts)
	return strings.Join(parts, "\x00")
}

// --- Task 3: lock the single approval surface ---

// assignsFalseToQuarantinedField reports whether fn assigns the boolean
// literal false -- directly, or through a local variable initialized to
// false (this codebase's own idiom: `cleared := false; sig.Quarantined =
// &cleared`, matching writePheromoneSignal's `quarantined := true` pattern
// for the opposite direction) -- to a field literally named Quarantined on
// a receiver that looks like a pheromone signal
// (looksLikePheromoneSignalReceiver, shared from pheromone_resolver_test.go).
// This is the release-side counterpart of assignsQuarantinedField (which
// catches ANY assignment, true or false): narrowing to exactly the release
// direction lets a legitimate quarantine-setting writer (import) coexist
// without double-flagging here.
func assignsFalseToQuarantinedField(fn *ast.FuncDecl) bool {
	falseVars := map[string]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			if ident, ok := unwrapParens(rhs).(*ast.Ident); ok && ident.Name == "false" {
				if i < len(assign.Lhs) {
					if lhsIdent, ok := assign.Lhs[i].(*ast.Ident); ok {
						falseVars[lhsIdent.Name] = true
					}
				}
			}
		}
		return true
	})

	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, lhs := range assign.Lhs {
			sel, ok := lhs.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "Quarantined" {
				continue
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok || !looksLikePheromoneSignalReceiver(ident.Name) {
				continue
			}
			if i >= len(assign.Rhs) {
				continue
			}
			rhs := unwrapParens(assign.Rhs[i])
			if idExpr, ok := rhs.(*ast.Ident); ok && idExpr.Name == "false" {
				found = true
			}
			if unary, ok := rhs.(*ast.UnaryExpr); ok && unary.Op == token.AND {
				if inner, ok := unary.X.(*ast.Ident); ok && falseVars[inner.Name] {
					found = true
				}
			}
		}
		return true
	})
	return found
}

func TestQuarantineClearsOnlyOnApproval(t *testing.T) {
	allowed := map[string]bool{
		"clearSignalQuarantine": true,
	}
	funcs := parseCmdPackageFuncs(t)
	var offenders []string
	for name, fn := range funcs {
		if allowed[name] {
			continue
		}
		if assignsFalseToQuarantinedField(fn) {
			offenders = append(offenders, name)
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("found a function clearing PheromoneSignal.Quarantined outside clearSignalQuarantine: %v -- every quarantine release must go through clearSignalQuarantine, called only from approvePendingNote's import branch", offenders)
	}
}

// funcLitAssignedToCobraRunField finds every *ast.FuncLit assigned to a
// field named Run or RunE inside a composite literal of type cobra.Command,
// returning a map from the enclosing command variable's name to that
// closure body. Cobra commands in this codebase are declared as
// `var xCmd = &cobra.Command{RunE: func(...) {...}, ...}` -- a struct
// literal, not a named func -- so this walks composite literals directly
// rather than reusing parseCmdPackageFuncs (which only sees named FuncDecls).
func cobraCommandRunClosures(t *testing.T) map[string]*ast.FuncLit {
	t.Helper()
	closures := map[string]*ast.FuncLit{}
	pkgDir := "."
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(pkgDir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) == 0 {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					unary, ok := vs.Values[i].(*ast.UnaryExpr)
					var lit *ast.CompositeLit
					if ok && unary.Op == token.AND {
						lit, _ = unary.X.(*ast.CompositeLit)
					} else {
						lit, _ = vs.Values[i].(*ast.CompositeLit)
					}
					if lit == nil {
						continue
					}
					sel, ok := lit.Type.(*ast.SelectorExpr)
					if !ok || sel.Sel == nil || sel.Sel.Name != "Command" {
						continue
					}
					for _, elt := range lit.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok || (key.Name != "Run" && key.Name != "RunE") {
							continue
						}
						if funcLit, ok := kv.Value.(*ast.FuncLit); ok {
							closures[name.Name] = funcLit
						}
					}
				}
			}
		}
	}
	return closures
}

// funcLitCallsByName reports whether the given function literal's body
// directly calls a function with the given name.
func funcLitCallsByName(lit *ast.FuncLit, name string) bool {
	found := false
	ast.Inspect(lit.Body, func(n ast.Node) bool {
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

func TestOneApprovalSurface(t *testing.T) {
	closures := cobraCommandRunClosures(t)
	var offenders []string
	for cmdName, closure := range closures {
		if cmdName == "suggestApproveCmd" {
			continue
		}
		if funcLitCallsByName(closure, "approvePendingNote") || funcLitCallsByName(closure, "clearSignalQuarantine") {
			offenders = append(offenders, cmdName)
		}
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("found a Cobra command other than suggestApproveCmd approving or releasing a queued note: %v", offenders)
	}
}

// pendingNoteQueueProducerAllowlist names every function outside
// enqueuePendingNote itself permitted to construct a colony.PendingSuggestion{}
// composite literal directly. runSuggestAnalyzeWithForce predates this
// plan's queue consolidation and has its own established, separately tested
// batch-merge-and-dedup semantics (mergePendingSuggestions) -- it still
// writes into the SAME PendingSuggestions slice via the same struct and the
// same content-hash dedup rule, so it does not violate the single-queue
// invariant even though it does not literally call enqueuePendingNote.
var pendingNoteQueueProducerAllowlist = map[string]string{
	"runSuggestAnalyzeWithForce": "pre-existing batch suggestion producer with its own tested merge/dedup/trim pipeline (mergePendingSuggestions); still writes the same queue, same struct, same dedup rule",
}

// funcDeclConstructsPendingSuggestion reports whether fn builds a NEW
// colony.PendingSuggestion carrying at least one field -- a genuine
// "propose a note" call site. An empty colony.PendingSuggestion{} literal
// (e.g. findPendingNote's own "not found" zero-value return) is not a
// producer and is deliberately excluded, or every zero-value return
// anywhere in the package would false-positive.
func funcDeclConstructsPendingSuggestion(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "PendingSuggestion" {
			return true
		}
		if len(lit.Elts) == 0 {
			return true
		}
		found = true
		return true
	})
	return found
}

func TestEveryProposalEntersTheOneQueue(t *testing.T) {
	funcs := parseCmdPackageFuncs(t)
	var offenders []string
	for name, fn := range funcs {
		if name == "enqueuePendingNote" {
			continue
		}
		if !funcDeclConstructsPendingSuggestion(fn) {
			continue
		}
		if _, ok := pendingNoteQueueProducerAllowlist[name]; ok {
			continue
		}
		offenders = append(offenders, name)
	}
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Fatalf("found a function constructing colony.PendingSuggestion outside enqueuePendingNote and outside pendingNoteQueueProducerAllowlist: %v -- route through enqueuePendingNote, or add a named, reasoned allowlist entry", offenders)
	}
}

func TestRuntimeCannotReleaseAQuarantine(t *testing.T) {
	_, s := setupExchangeTest(t)

	importProvenance := colony.PheromoneProvenanceImport
	guardSignal := colony.PheromoneSignal{
		ID: "sig_guard_1", Type: "FOCUS", Content: json.RawMessage(`{"text":"must stay quarantined"}`),
		Source: "import", Active: true, Strength: floatPtr(1.0),
		ContentHash: strPtr("sha256:guard1"), Provenance: &importProvenance, Quarantined: boolPtr(true),
	}
	if err := s.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{guardSignal}}); err != nil {
		t.Fatalf("seed guard signal: %v", err)
	}

	assertStillQuarantined := func(t *testing.T, label string) {
		t.Helper()
		var pf colony.PheromoneFile
		if err := s.LoadJSON("pheromones.json", &pf); err != nil {
			t.Fatalf("[%s] load pheromones: %v", label, err)
		}
		for _, sig := range pf.Signals {
			if sig.ID == guardSignal.ID {
				if sig.Quarantined == nil || !*sig.Quarantined {
					t.Errorf("[%s] guard signal %q Quarantined = %v, want still true", label, guardSignal.ID, sig.Quarantined)
				}
				return
			}
		}
		t.Errorf("[%s] guard signal %q vanished from pheromones.json", label, guardSignal.ID)
	}

	// The phase-end emitters and the midden threshold emitter each write
	// their own new signals; none may touch the pre-existing quarantined one.
	emitPhaseCompletionFeedback(colony.Phase{ID: 1, Name: "test phase"}, phaseEndConsolidationSummary{}, 0, 0)
	assertStillQuarantined(t, "emitPhaseCompletionFeedback")

	emitDecisionFeedback("a worker question?", "an owner answer", 1)
	assertStillQuarantined(t, "emitDecisionFeedback")

	if err := s.SaveJSON("midden.json", colony.MiddenFile{Entries: []colony.MiddenEntry{
		{ID: "m1", Timestamp: "2026-01-01T00:00:00Z", Category: "cat", Source: "test", Message: "failure 1"},
		{ID: "m2", Timestamp: "2026-01-02T00:00:00Z", Category: "cat", Source: "test", Message: "failure 2"},
		{ID: "m3", Timestamp: "2026-01-03T00:00:00Z", Category: "cat", Source: "test", Message: "failure 3"},
	}}); err != nil {
		t.Fatalf("seed midden: %v", err)
	}
	emitMiddenThresholdRedirect()
	assertStillQuarantined(t, "emitMiddenThresholdRedirect")

	// A second, unrelated import must quarantine its OWN new signal and
	// never touch the pre-existing one.
	tmpDir := t.TempDir()
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<pheromones version="1.0" count="1">
  <signal id="sig_other_import" type="FOCUS" priority="normal" source="user" created_at="2026-04-01T10:00:00Z" active="true" strength="1.0">
    <content><text>An unrelated import</text></content>
  </signal>
</pheromones>`
	xmlFile := filepath.Join(tmpDir, "other.xml")
	if err := os.WriteFile(xmlFile, []byte(xmlContent), 0644); err != nil {
		t.Fatalf("write xml: %v", err)
	}
	rootCmd.SetArgs([]string{"import", "pheromones", xmlFile})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("import pheromones failed: %v", err)
	}
	rootCmd.SetArgs([]string{})
	assertStillQuarantined(t, "importPheromonesData (unrelated import)")
}
