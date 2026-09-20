package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"gopkg.in/yaml.v3"
)

type entombTransactionFixture199 struct {
	root     string
	dataRoot string
	outcome  colony.SealOutcome
}

func TestEntombTransaction199RefusalZeroWrite(t *testing.T) {
	if got, want := entombConfirmationCopy(), "Archive and clear this sealed colony after archive verification? [y/N]"; got != want {
		t.Fatalf("confirmation = %q, want %q", got, want)
	}

	t.Run("confirmation", func(t *testing.T) {
		fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
		before := entombFixtureDigest199(t, fixture.root)
		result, err := runEntombTransaction(entombTransactionInput{
			Root: fixture.root, DataRoot: fixture.dataRoot, Now: entombTransactionTime199(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.AwaitingConfirmation || result.Confirmation != entombConfirmationCopy() {
			t.Fatalf("missing exact confirmation result: %+v", result)
		}
		if after := entombFixtureDigest199(t, fixture.root); after != before {
			t.Fatalf("confirmation preview mutated the repository\nbefore=%s\nafter=%s", before, after)
		}
	})

	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, fixture entombTransactionFixture199)
	}{
		{
			name: "unsealed",
			mutate: func(t *testing.T, fixture entombTransactionFixture199) {
				state := readEntombState199(t, fixture.dataRoot)
				state.Milestone = "First Mound"
				writeEntombJSON199(t, filepath.Join(fixture.dataRoot, "COLONY_STATE.json"), state)
			},
		},
		{
			name: "inconsistent",
			mutate: func(t *testing.T, fixture entombTransactionFixture199) {
				state := readEntombState199(t, fixture.dataRoot)
				state.SealOutcome.OutcomeID = "different-seal"
				writeEntombJSON199(t, filepath.Join(fixture.dataRoot, "COLONY_STATE.json"), state)
			},
		},
		{
			name: "unverifiable",
			mutate: func(t *testing.T, fixture entombTransactionFixture199) {
				path := filepath.Join(fixture.dataRoot, "seal", "findings.json")
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				content = []byte(strings.Replace(string(content), fixture.outcome.Transaction.ID, "wrong-transaction", 1))
				if err := os.WriteFile(path, content, 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
			test.mutate(t, fixture)
			before := entombFixtureDigest199(t, fixture.root)
			_, err := runEntombTransaction(entombTransactionInput{
				Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
			})
			if err == nil || !strings.Contains(err.Error(), "Active colony state was retained") {
				t.Fatalf("unsafe preflight was not refused truthfully: %v", err)
			}
			if after := entombFixtureDigest199(t, fixture.root); after != before {
				t.Fatalf("failed preflight mutated the repository\nbefore=%s\nafter=%s", before, after)
			}
		})
	}
}

func TestEntombTransaction199StageOrder(t *testing.T) {
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
	result, err := runEntombTransaction(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Stage archive",
		"Write digest manifest",
		"Verify bytes and cross-references",
		"Publish chamber and tombstone",
		"Clear active state",
	}
	if !reflect.DeepEqual(result.Stages, want) {
		t.Fatalf("stage order = %#v, want %#v", result.Stages, want)
	}
}

func TestEntombTransaction199FaultRetainsActive(t *testing.T) {
	for _, stage := range []string{
		"Stage archive",
		"Write digest manifest",
		"Verify bytes and cross-references",
		"Publish chamber and tombstone",
		"Clear active state",
	} {
		t.Run(stage, func(t *testing.T) {
			fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
			statePath := filepath.Join(fixture.dataRoot, "COLONY_STATE.json")
			before, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			_, err = runEntombTransaction(entombTransactionInput{
				Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
				Fault: func(point string) error {
					if point == stage {
						return errors.New("injected " + stage + " failure")
					}
					return nil
				},
			})
			if err == nil || !strings.Contains(err.Error(), "Active colony state was retained") || !strings.Contains(err.Error(), "retry") {
				t.Fatalf("stage failure lacked retained-state recovery guidance: %v", err)
			}
			after, readErr := os.ReadFile(statePath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(after) != string(before) {
				t.Fatalf("active state changed at failed stage %q", stage)
			}
		})
	}
}

func TestEntombTransaction199ReplayExactlyOnce(t *testing.T) {
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
	input := entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
		Fault: func(point string) error {
			if point == "Clear active state" {
				return errors.New("interrupt before clear")
			}
			return nil
		},
	}
	if _, err := runEntombTransaction(input); err == nil {
		t.Fatal("expected injected interruption")
	}
	input.Fault = nil
	first, err := runEntombTransaction(input)
	if err != nil {
		t.Fatalf("resume entomb: %v", err)
	}
	if !first.Replay {
		t.Fatal("resumed transaction was not identified as replay")
	}
	second, err := runEntombTransaction(input)
	if err != nil {
		t.Fatalf("replay completed entomb: %v", err)
	}
	if !second.Replay || second.Receipt.ReceiptID != first.Receipt.ReceiptID || second.ManifestDigest != first.ManifestDigest {
		t.Fatalf("replay produced a different result\nfirst=%+v\nsecond=%+v", first, second)
	}

	entries, err := os.ReadDir(filepath.Join(fixture.root, ".aether", "chambers"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("replay produced %d chambers, want exactly one", len(entries))
	}
	receiptPath := filepath.Join(fixture.dataRoot, "transactions", first.Receipt.Transaction.ID, "receipt.json")
	if _, err := os.Stat(receiptPath); err != nil {
		t.Fatalf("one durable receipt missing: %v", err)
	}
	if state := readEntombState199(t, fixture.dataRoot); state.State != colony.StateIDLE {
		t.Fatalf("replayed transaction did not clear active state: %+v", state)
	}
}

func TestEntombTransaction199ForcedMarker(t *testing.T) {
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionForcedIncomplete)
	result, err := runEntombTransaction(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != colony.SealDispositionForcedIncomplete || result.OwnerReason != fixture.outcome.OwnerReason {
		t.Fatalf("result dropped forced identity: %+v", result)
	}
	for _, path := range []string{
		filepath.Join(result.ChamberPath, "manifest.json"),
		filepath.Join(result.ChamberPath, "CROWNED-ANTHILL.md"),
		filepath.Join(result.ChamberPath, "colony-archive.xml"),
		filepath.Join(fixture.root, ".aether", "HANDOFF.md"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range []string{"forced_incomplete", fixture.outcome.OwnerReason} {
			if !strings.Contains(string(content), marker) {
				t.Fatalf("%s omitted forced marker %q\n%s", path, marker, content)
			}
		}
	}
}

func TestEntombTransaction199Success(t *testing.T) {
	fixture := newEntombTransactionFixture199(t, colony.SealDispositionVerified)
	result, err := runEntombTransaction(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataRoot, Confirmed: true, Now: entombTransactionTime199(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChamberPath == "" || result.ManifestDigest == "" || result.Receipt.ReceiptID == "" || result.Next != `/ant-init "next goal"` {
		t.Fatalf("success result omitted required proof: %+v", result)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(result.ChamberPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest colony.ArchiveManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ManifestDigest != result.ManifestDigest || manifest.Receipt == nil || manifest.Receipt.ID != result.Receipt.ReceiptID {
		t.Fatalf("manifest and transaction receipt disagree: manifest=%+v result=%+v", manifest, result)
	}
	for _, retained := range []string{"QUEEN.md", "HANDOFF.md", "seal/outcome.json", "seal/learnings.json"} {
		if _, err := os.Stat(filepath.Join(result.ChamberPath, retained)); err != nil {
			t.Fatalf("retained archive source %s missing: %v", retained, err)
		}
	}
	for _, cleared := range []string{
		filepath.Join(fixture.root, ".aether", "CROWNED-ANTHILL.md"),
		filepath.Join(fixture.dataRoot, "seal", "outcome.json"),
		filepath.Join(fixture.dataRoot, "session.json"),
	} {
		if _, err := os.Stat(cleared); !os.IsNotExist(err) {
			t.Fatalf("active source %s was not cleared after verification: %v", cleared, err)
		}
	}
	state := readEntombState199(t, fixture.dataRoot)
	if state.State != colony.StateIDLE || state.Goal != nil || state.ArchiveReference == nil || state.ArchiveReference.ManifestDigest != result.ManifestDigest {
		t.Fatalf("active state did not retain only the archive pointer: %+v", state)
	}
	tombstone, err := os.ReadFile(filepath.Join(fixture.root, ".aether", "HANDOFF.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{result.ManifestDigest, result.Receipt.ReceiptID, "verified", `/ant-init "next goal"`} {
		if !strings.Contains(string(tombstone), want) {
			t.Fatalf("tombstone omitted %q\n%s", want, tombstone)
		}
	}
}

func TestEntombWrapperContract199(t *testing.T) {
	const (
		description = "Archive and clear the sealed colony."
		runtimeCall = "AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS"
		source      = ".aether/commands/entomb.yaml"
	)
	repoRoot := filepath.Clean("..")
	yamlPath := filepath.Join(repoRoot, filepath.FromSlash(source))
	rawYAML, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Runtime     struct {
			Command string `yaml:"command"`
		} `yaml:"runtime"`
		CeremonyContract struct {
			Authority   string `yaml:"authority"`
			WrapperRole string `yaml:"wrapper_role"`
		} `yaml:"ceremony_contract"`
		Guardrails []string `yaml:"guardrails"`
	}
	if err := yaml.Unmarshal(rawYAML, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Name != "ant-entomb" || spec.Description != description || spec.Runtime.Command != runtimeCall {
		t.Fatalf("canonical entomb wrapper contract drifted: %+v", spec)
	}
	canonicalContract := strings.ToLower(strings.Join(append([]string{
		spec.CeremonyContract.Authority,
		spec.CeremonyContract.WrapperRole,
	}, spec.Guardrails...), "\n"))
	for _, required := range []string{
		"separate", "optional", "preview", "confirmation", "stage archive",
		"write digest manifest", "verify bytes and cross-references",
		"publish chamber and tombstone", "clear active state", "typed runtime result",
		"exactly once", "do not copy", "do not parse", "do not clear", "do not fabricate",
	} {
		if !strings.Contains(canonicalContract, required) {
			t.Errorf("canonical wrapper contract lacks %q:\n%s", required, string(rawYAML))
		}
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant-entomb.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "entomb.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "entomb.md"),
	}
	var canonicalBody string
	for _, path := range wrapperPaths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		text := string(raw)
		wantHeader := "<!-- Aether-managed: runtime spec at " + source + ". Synced by aether update. -->"
		if !strings.HasPrefix(text, wantHeader+"\n") {
			t.Errorf("%s lacks canonical source linkage", path)
		}
		if !strings.Contains(text, `description: "`+description+`"`) {
			t.Errorf("%s description does not match the canonical sentence", path)
		}
		if strings.Count(text, runtimeCall) != 1 {
			t.Errorf("%s has %d runtime calls, want exactly one", path, strings.Count(text, runtimeCall))
		}
		lower := strings.ToLower(text)
		for _, required := range []string{
			"separate", "optional", "preview", "confirmation", "stage archive",
			"write digest manifest", "verify bytes and cross-references",
			"publish chamber and tombstone", "clear active state", "typed runtime result",
			"exactly once", "do not copy", "do not parse", "do not clear", "do not fabricate",
		} {
			if !strings.Contains(lower, required) {
				t.Errorf("%s lacks %q", path, required)
			}
		}
		for _, forbidden := range []string{
			"cp -", "cp ", "rsync ", "jq ", "colony_state.json", "os.readfile", "writefile(",
			"removeall(", "rm -", "aether seal", "after seal automatically", "automatically invoke",
		} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("%s contains forbidden wrapper-side archive/parse/clear/auto-seal behavior %q", path, forbidden)
			}
		}
		body := strings.TrimSpace(strings.Join(strings.Split(text, "\n")[5:], "\n"))
		if canonicalBody == "" {
			canonicalBody = body
		} else if body != canonicalBody {
			t.Errorf("generated wrapper %s is not semantically identical to the other entomb wrappers", path)
		}
	}
}

func newEntombTransactionFixture199(t *testing.T, disposition colony.SealDisposition) entombTransactionFixture199 {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(filepath.Join(dataRoot, "seal"), 0o755); err != nil {
		t.Fatal(err)
	}
	outcome := entombManifestSealOutcome199(disposition)
	goal := "Archive the verified lifecycle"
	state := colony.ColonyState{
		Version: "3.0", Goal: &goal, ColonyVersion: 4, Scope: colony.ScopeProject,
		State: colony.StateCOMPLETED, CurrentPhase: 1, Milestone: "Crowned Anthill", SealOutcome: &outcome,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Closure", Status: colony.PhaseCompleted}}},
	}
	common := map[string]any{
		"schema_version": colony.LifecycleSchemaVersion,
		"transaction_id": outcome.Transaction.ID,
		"outcome_kind":   outcome.OutcomeKind,
		"disposition":    outcome.Disposition,
		"owner_reason":   outcome.OwnerReason,
	}
	rollback := make(map[string]any, len(common)+1)
	for key, value := range common {
		rollback[key] = value
	}
	rollback["rollback"] = outcome.Rollback
	report := "# CROWNED-ANTHILL\n\n- Goal: " + goal + "\n"
	if disposition == colony.SealDispositionForcedIncomplete {
		report = "# Forced seal record — completion not verified\n\nDisposition: forced_incomplete\nOwner reason: " + outcome.OwnerReason + "\nTransaction: " + outcome.Transaction.ID + "\n"
	}
	writeEntombJSON199(t, filepath.Join(dataRoot, "COLONY_STATE.json"), state)
	writeEntombJSON199(t, filepath.Join(dataRoot, "seal", "outcome.json"), outcome)
	writeEntombJSON199(t, filepath.Join(dataRoot, "seal", "findings.json"), common)
	writeEntombJSON199(t, filepath.Join(dataRoot, "seal", "learnings.json"), common)
	writeEntombJSON199(t, filepath.Join(dataRoot, "seal", "checkpoints.json"), common)
	writeEntombJSON199(t, filepath.Join(dataRoot, "seal", "rollback.json"), rollback)
	writeEntombJSON199(t, filepath.Join(dataRoot, "seal", "receipt.json"), map[string]any{"receipt": outcome.Receipt, "disposition": outcome.Disposition, "owner_reason": outcome.OwnerReason})
	writeEntombJSON199(t, filepath.Join(dataRoot, "pheromones.json"), map[string]any{"version": "2.0", "signals": []any{}})
	writeEntombJSON199(t, filepath.Join(dataRoot, "session.json"), map[string]any{"session_id": "session-entomb-199"})
	writeEntombJSON199(t, filepath.Join(dataRoot, "reviews", "quality", "ledger.json"), map[string]any{"entries": []any{}})
	writeEntombFile199(t, filepath.Join(root, ".aether", "CROWNED-ANTHILL.md"), []byte(report))
	writeEntombFile199(t, filepath.Join(root, ".aether", "QUEEN.md"), []byte("# Retained colony memory\n"))
	writeEntombFile199(t, filepath.Join(root, ".aether", "HANDOFF.md"), []byte("# Sealed colony handoff\n"))
	writeEntombFile199(t, filepath.Join(root, ".aether", "CONTEXT.md"), []byte("# Sealed colony context\n"))
	return entombTransactionFixture199{root: root, dataRoot: dataRoot, outcome: outcome}
}

func entombTransactionTime199() time.Time {
	return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
}

func writeEntombJSON199(t *testing.T, path string, value any) {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeEntombFile199(t, path, append(content, '\n'))
}

func writeEntombFile199(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readEntombState199(t *testing.T, dataRoot string) colony.ColonyState {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dataRoot, "COLONY_STATE.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(content, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func entombFixtureDigest199(t *testing.T, root string) string {
	t.Helper()
	var records []string
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		records = append(records, filepath.ToSlash(relative)+"="+lifecycleDigest(content))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(records)
	return lifecycleDigest([]byte(strings.Join(records, "\n")))
}
