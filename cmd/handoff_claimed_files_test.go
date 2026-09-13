package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// boltStyleWorkerResult reproduces the exact shape of the builder report that
// lost seven verified files downstream (CosmicDashboard, Aether v1.0.75).
//
// The paths and the split are taken from the real runtime-persisted packet
// (.aether/data/build/phase-2/attempts/attempt-20260913T092913...completion.json,
// worker Bolt-68), not invented: six files named at top level AND in the
// worker's own handoff, seven more named only inside task receipts, every
// receipt status "completed". The builder had been asked to correct one task
// and resent its report; the resend narrowed both the top-level lists and the
// handoff to the follow-up fix alone.
//
// Building the fixture in the runtime's own shape is the point. A fixture in a
// shape the runtime cannot produce is a false certificate -- and the FIRST
// diagnosis of this bug (that the synthesised-fallback branch of
// buildWorkerHandoffRecord dropped the receipts) was disproved precisely by
// reading this packet: the real report supplied a non-empty handoff, so that
// branch never ran. Fixing it would have changed nothing.
func boltStyleWorkerResult() codex.WorkerResult {
	topLevel := []string{
		"dashboard/lib/connector-error.ts",
		"dashboard/__tests__/connector-error.test.ts",
		"dashboard/lib/integrations/readiness.ts",
		"dashboard/lib/integrations/__tests__/readiness.test.ts",
		"dashboard/lib/integrations/__tests__/business-connection-reads.test.ts",
		"dashboard/scripts/connector-smoke.ts",
	}
	receiptOnly := []string{
		"dashboard/lib/google-oauth.ts",
		"dashboard/lib/__tests__/google-oauth.test.ts",
		"dashboard/app/api/integrations/status/route.ts",
		"dashboard/app/api/integrations/status/__tests__/route.test.ts",
		"dashboard/components/business/IntegrationMappings.tsx",
		"dashboard/__tests__/components/IntegrationMappings.test.tsx",
		"dashboard/__tests__/business-cockpit-foundation.test.ts",
	}
	receipts := make([]codex.TaskReceipt, 0, len(receiptOnly))
	for i, path := range receiptOnly {
		receipts = append(receipts, codex.TaskReceipt{
			TaskID:        string(rune('a'+i)) + "-task",
			Status:        "completed",
			Summary:       "earlier completed work, named only here",
			FilesModified: []string{path},
		})
	}
	return codex.WorkerResult{
		WorkerName:    "Bolt-68",
		Caste:         "builder",
		Status:        "completed",
		Summary:       "resent after correction",
		FilesModified: topLevel,
		TaskReceipts:  receipts,
		// The resend's handoff carries the SAME narrowed list as the
		// top-level fields -- this is what made the fallback branch
		// irrelevant and the bug survive.
		Handoff: codex.WorkerHandoff{ChangedFiles: topLevel, VerificationStatus: "passed"},
	}
}

func boltStyleAllPaths() []string {
	r := boltStyleWorkerResult()
	all := append([]string{}, r.FilesModified...)
	for _, receipt := range r.TaskReceipts {
		all = append(all, receipt.FilesModified...)
	}
	return all
}

// TestAllClaimedFilesUnionsReceiptsAndHandoff proves the single definition of
// "every file this worker claimed" loses nothing from any of the three places
// a worker can name a path.
func TestAllClaimedFilesUnionsReceiptsAndHandoff(t *testing.T) {
	got := codex.AllClaimedFiles(boltStyleWorkerResult())
	have := make(map[string]bool, len(got))
	for _, p := range got {
		have[p] = true
	}
	for _, want := range boltStyleAllPaths() {
		if !have[want] {
			t.Fatalf("claimed file %q was dropped; AllClaimedFiles returned %d paths: %v", want, len(got), got)
		}
	}
	if len(got) != 13 {
		t.Fatalf("expected 13 distinct claimed paths (6 top-level + 7 receipt-only), got %d: %v", len(got), got)
	}
}

// TestHandoffRecordNeverDropsAClaimedFile is the real guarantee, asserted on
// the record the phase commit actually reads (phaseChangedFilesFromHandoffs
// unions workerHandoffRecord.ChangedFiles). It runs BOTH paths into that
// record -- a worker that supplied its own handoff, and one that did not --
// because the reported failure took the supplied-handoff path while the first
// proposed fix only touched the other one. A guarantee that holds on the path
// nobody used is worth nothing.
func TestHandoffRecordNeverDropsAClaimedFile(t *testing.T) {
	for _, tc := range []struct {
		name          string
		supplyHandoff bool
	}{
		{"worker supplied its own handoff (the reported case)", true},
		{"worker supplied no handoff (the synthesised fallback)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := boltStyleWorkerResult()
			if !tc.supplyHandoff {
				result.Handoff = codex.WorkerHandoff{}
			}
			record := buildWorkerHandoffRecord(
				codex.WorkerDispatch{Phase: 2, Wave: 1, WorkerName: "Bolt-68", Caste: "builder"},
				codex.DispatchResult{WorkerName: "Bolt-68", Status: "completed", WorkerResult: &result},
			)
			have := make(map[string]bool, len(record.ChangedFiles))
			for _, p := range record.ChangedFiles {
				have[p] = true
			}
			for _, want := range boltStyleAllPaths() {
				if !have[want] {
					t.Fatalf("the phase commit reads this record: claimed file %q is missing from ChangedFiles (%d entries: %v)",
						want, len(record.ChangedFiles), record.ChangedFiles)
				}
			}
		})
	}
}

// TestPhaseCommitNamesAClaimedFileItCannotSave proves the phase commit stops
// dropping an unsaveable claimed path in silence.
//
// A worker claiming a file that is neither in the working tree nor known to
// git is a worker claiming credit for work it did not do. Before this, that
// path was filtered out of the commit set with no output at all, and the
// commit reported plain success.
//
// This is the reachable half of the v1.0.75 downstream report. The other half
// -- files missing from the claimed set entirely -- is closed at its cause by
// TestHandoffRecordNeverDropsAClaimedFile, not here: a post-commit check can
// only ever ask about paths already in the claimed set, so no warning at this
// layer could have detected it.
func TestPhaseCommitNamesAClaimedFileItCannotSave(t *testing.T) {
	saveGlobals(t)
	root := initPhaseCommitRepo(t)

	// One real file the worker genuinely wrote...
	if err := os.WriteFile(filepath.Join(root, "real.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatalf("write real.go: %v", err)
	}
	// ...and one it only claims: never written, never tracked.
	writePhaseHandoffs(t, 1, "real.go", "imaginary/never-written.go")

	state := phaseCommitTestState("goal", 1, "")
	result := commitPhaseAdvance(root, state, colony.Phase{ID: 1, Name: "First"})

	if !result.Committed {
		t.Fatalf("the real file should still commit; the phase must advance: %+v", result)
	}
	if len(result.Unaccounted) != 1 || result.Unaccounted[0] != "imaginary/never-written.go" {
		t.Fatalf("expected the unsaveable claimed path to be named, got Unaccounted=%v", result.Unaccounted)
	}

	// And the owner actually sees it.
	rendered := map[string]interface{}{}
	attachPhaseCommitResult(rendered, result)
	entry, _ := rendered["phase_commit"].(map[string]interface{})
	if entry == nil {
		t.Fatalf("expected a phase_commit entry, got %v", rendered)
	}
	warning, _ := entry["unaccounted_warning"].(string)
	if !strings.Contains(warning, "imaginary/never-written.go") {
		t.Fatalf("expected the rendered warning to name the file, got %q", warning)
	}

	// The real file still landed: naming a problem must not suppress the work.
	committed := gitOut(t, root, "show", "--name-only", "--format=", "HEAD")
	if !strings.Contains(committed, "real.go") {
		t.Fatalf("expected real.go in the commit, got %q", committed)
	}
}

// TestAntipatternGateReportsEverySecretInAFile proves the exposed-secret scan
// reports every hit in a file rather than stopping at the first.
//
// Reported downstream on v1.0.75: the continue gate re-runs this scan, so one
// hit per file meant fixing a line only surfaced the next one. A file with 22
// lookalike values needed 22 sequential `aether continue` runs to clear.
func TestAntipatternGateReportsEverySecretInAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ts")

	// Six distinct offenders on known lines. The words the scan treats as
	// benign ("test", "example", "mock", "fake") are deliberately absent --
	// including from the file name -- so every line is a real hit.
	var b strings.Builder
	b.WriteString("export const config = {\n")
	wantLines := []int{}
	for i := 0; i < 6; i++ {
		b.WriteString("  SERVICE_API_TOKEN: 'abcdef0123456789',\n")
		wantLines = append(wantLines, i+2)
	}
	b.WriteString("};\n")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	criticals, _, err := scanFileForAntipatterns(path)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	var got []int
	for _, f := range criticals {
		if f.Pattern == "exposed-secret" && f.Line > 0 {
			got = append(got, f.Line)
		}
	}
	if len(got) != len(wantLines) {
		t.Fatalf("expected every one of the %d secrets reported with its own line, got %d: %+v",
			len(wantLines), len(got), criticals)
	}
	for i, want := range wantLines {
		if got[i] != want {
			t.Fatalf("expected hit %d on line %d, got line %d (all: %v)", i+1, want, got[i], got)
		}
	}
}
