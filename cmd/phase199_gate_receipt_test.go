package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	phase199GateReceiptPath     = ".planning/phases/199-front-door-and-classic-contract/199-GATE-RECEIPT.json"
	phase199GateReceiptVersion  = "phase199-gate-receipt/v2"
	phase199HistoricalStatePath = ".planning/STATE.md"
	phase199PatternsPath        = ".planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md"
	phase199ConfigPath          = ".planning/config.json"
	phase199GSDPath             = ".gsd"
	phase199GSDMutableSentinel  = "dispatch-isolation-sentinel.json"
	phase199GateFutureSkew      = time.Minute
	phase199GateMaximumDuration = 2 * time.Hour
)

var phase199ProtectedReceiptPaths = []string{
	phase199PatternsPath,
}

var phase199HistoricalReceiptPaths = []string{phase199HistoricalStatePath}

type phase199GateReceipt struct {
	SchemaVersion  string                               `json:"schema_version"`
	Phase          string                               `json:"phase"`
	Plan           string                               `json:"plan"`
	Status         string                               `json:"status"`
	CreatedAt      string                               `json:"created_at"`
	Repository     phase199ReceiptRepository            `json:"repository"`
	Historical     []phase199HistoricalBlobRecord       `json:"historical_blobs"`
	Protected      []phase199ProtectedFingerprintRecord `json:"protected_fingerprints"`
	Preexisting    []phase199ProtectedFingerprintRecord `json:"preexisting_user_changes"`
	ConfigBaseline string                               `json:"config_baseline"`
	GSDBaseline    []phase199GSDEntry                   `json:"gsd_baseline"`
	Gates          []phase199GateRun                    `json:"gates"`
}

type phase199ReceiptRepository struct {
	Revision string `json:"revision"`
	Tree     string `json:"tree"`
}

type phase199HistoricalBlobRecord struct {
	Path   string `json:"path"`
	Blob   string `json:"blob"`
	SHA256 string `json:"sha256"`
}

type phase199ProtectedFingerprintRecord struct {
	Path   string                           `json:"path"`
	Before phase199ProtectedPathFingerprint `json:"before"`
	After  phase199ProtectedPathFingerprint `json:"after"`
}

type phase199ProtectedPathFingerprint struct {
	Status string `json:"status"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	Digest string `json:"digest"`
}

type phase199GSDEntry struct {
	Path          string `json:"path"`
	Type          string `json:"type"`
	Mode          string `json:"mode"`
	Digest        string `json:"digest,omitempty"`
	SymlinkTarget string `json:"symlink_target,omitempty"`
}

type phase199GateRun struct {
	Command      string `json:"command"`
	ExitCode     int    `json:"exit_code"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at"`
	Revision     string `json:"revision"`
	Tree         string `json:"tree"`
	OutputSHA256 string `json:"output_sha256"`
}

func TestPhase199GateReceiptSchema(t *testing.T) {
	receipt := loadPhase199GateReceipt(t)
	if err := validatePhase199GateReceiptSchema(receipt, time.Now().UTC()); err != nil {
		t.Fatalf("validate checked-in receipt schema: %v", err)
	}

	complete := phase199ValidCompleteReceipt(t)
	for name, mutate := range map[string]func(*phase199GateReceipt){
		"missing gate": func(r *phase199GateReceipt) { r.Gates = r.Gates[:1] },
		"created after first gate": func(r *phase199GateReceipt) {
			r.CreatedAt = r.Gates[0].FinishedAt
		},
		"future created_at": func(r *phase199GateReceipt) {
			r.CreatedAt = time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339Nano)
		},
		"overlapping gates": func(r *phase199GateReceipt) {
			r.Gates[1].StartedAt = r.Gates[0].StartedAt
		},
		"gate exceeds duration bound": func(r *phase199GateReceipt) {
			finished := phase199MustParseTime(t, r.Gates[0].FinishedAt)
			r.Gates[0].StartedAt = finished.Add(-phase199GateMaximumDuration - time.Nanosecond).Format(time.RFC3339Nano)
		},
		"gate finishes before start": func(r *phase199GateReceipt) {
			r.Gates[0].FinishedAt = phase199MustParseTime(t, r.Gates[0].StartedAt).Add(-time.Nanosecond).Format(time.RFC3339Nano)
		},
		"future gate": func(r *phase199GateReceipt) {
			started := time.Now().UTC().Add(2 * time.Minute)
			r.Gates[0].StartedAt = started.Format(time.RFC3339Nano)
			r.Gates[0].FinishedAt = started.Add(time.Minute).Format(time.RFC3339Nano)
		},
		"nonzero exit":            func(r *phase199GateReceipt) { r.Gates[0].ExitCode = 1 },
		"substitute command":      func(r *phase199GateReceipt) { r.Gates[1].Command = "go test ./...  -race" },
		"bad digest":              func(r *phase199GateReceipt) { r.Gates[0].OutputSHA256 = "not-a-sha256" },
		"wrong tree":              func(r *phase199GateReceipt) { r.Gates[0].Tree = strings.Repeat("0", 40) },
		"missing historical blob": func(r *phase199GateReceipt) { r.Historical = nil },
		"bad historical digest": func(r *phase199GateReceipt) {
			r.Historical[0].SHA256 = "not-a-sha256"
		},
		"changed protected state": func(r *phase199GateReceipt) {
			r.Protected[0].After.Digest = strings.Repeat("0", 64)
		},
		"config baseline digest mismatch": func(r *phase199GateReceipt) {
			r.ConfigBaseline += "\n"
		},
		"mutable sentinel included in baseline": func(r *phase199GateReceipt) {
			r.GSDBaseline = append(r.GSDBaseline, phase199GSDEntry{Path: phase199GSDMutableSentinel, Type: "file", Mode: "0644", Digest: strings.Repeat("a", 64)})
		},
		"unsorted gsd baseline": func(r *phase199GateReceipt) {
			r.GSDBaseline[0], r.GSDBaseline[1] = r.GSDBaseline[1], r.GSDBaseline[0]
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := clonePhase199GateReceipt(t, complete)
			mutate(&invalid)
			if err := validatePhase199GateReceiptSchema(invalid, time.Now().UTC()); err == nil {
				t.Fatal("invalid complete receipt unexpectedly passed schema validation")
			}
		})
	}

	t.Run("unknown field", func(t *testing.T) {
		raw, err := json.Marshal(complete)
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]json.RawMessage
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatal(err)
		}
		document["unexpected"] = json.RawMessage(`true`)
		raw, err = json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodePhase199GateReceipt(raw); err == nil {
			t.Fatal("receipt with an unknown field unexpectedly decoded")
		}
	})

	for name, receipt := range map[string]phase199GateReceipt{
		"incomplete": phase199IncompleteReceipt(t),
		"partial":    phase199PartialReceipt(t),
	} {
		t.Run("final mode rejects "+name, func(t *testing.T) {
			if err := validatePhase199GateReceiptForMode(receipt, time.Now().UTC(), true); err == nil {
				t.Fatalf("final-mode validator accepted %s receipt", name)
			}
		})
	}
}

func TestPhase199GateReceiptSurvivesLaterLifecycleBookkeeping(t *testing.T) {
	receipt := phase199ValidCompleteReceipt(t)
	now := time.Now().UTC()
	receipt.CreatedAt = now.Add(-72 * time.Hour).Format(time.RFC3339Nano)
	receipt.Gates[0].StartedAt = now.Add(-71 * time.Hour).Format(time.RFC3339Nano)
	receipt.Gates[0].FinishedAt = now.Add(-70 * time.Hour).Format(time.RFC3339Nano)
	receipt.Gates[1].StartedAt = now.Add(-69 * time.Hour).Format(time.RFC3339Nano)
	receipt.Gates[1].FinishedAt = now.Add(-68 * time.Hour).Format(time.RFC3339Nano)

	if err := validatePhase199GateReceiptSchema(receipt, now); err != nil {
		t.Fatalf("durable historical receipt rejected after ordinary lifecycle time elapsed: %v", err)
	}

	fixture := newPhase199ReceiptFixture(t)
	applyPhase199LaterLifecycle(t, &fixture)
	if err := validatePhase199GateReceiptAtRoot(fixture.Root, fixture.Receipt, now, true); err != nil {
		t.Fatalf("ordinary later STATE, planning, sentinel, and config bookkeeping invalidated receipt: %v", err)
	}

	t.Run("changed historical Git object", func(t *testing.T) {
		changed := clonePhase199GateReceipt(t, fixture.Receipt)
		changed.Repository.Revision = phase199RunGit(t, fixture.Root, "rev-parse", "HEAD")
		changed.Repository.Tree = phase199RunGit(t, fixture.Root, "rev-parse", "HEAD^{tree}")
		for index := range changed.Gates {
			changed.Gates[index].Revision = changed.Repository.Revision
			changed.Gates[index].Tree = changed.Repository.Tree
		}
		expectPhase199ReceiptValidationFailure(t, fixture.Root, changed, now)
	})

	t.Run("changed PATTERNS bytes", func(t *testing.T) {
		changed := newPhase199ReceiptFixture(t)
		applyPhase199LaterLifecycle(t, &changed)
		phase199WriteFixtureFile(t, changed.Root, phase199PatternsPath, []byte("owner evidence changed\n"))
		expectPhase199ReceiptValidationFailure(t, changed.Root, changed.Receipt, now)
	})

	t.Run("changed protected config key", func(t *testing.T) {
		changed := newPhase199ReceiptFixture(t)
		applyPhase199LaterLifecycle(t, &changed)
		phase199WriteFixtureFile(t, changed.Root, phase199ConfigPath, []byte("{\"workflow\":{\"_auto_chain_active\":false,\"auto_advance\":true}}\n"))
		expectPhase199ReceiptValidationFailure(t, changed.Root, changed.Receipt, now)
	})

	areas := []struct {
		name     string
		existing string
		addition string
	}{
		{name: "scratch", existing: "scratch/blocker1.txt", addition: "scratch/added.txt"},
		{name: "worktrees", existing: "worktrees/phase-198-3/wave-1-manifest.json", addition: "worktrees/phase-198-3/added.txt"},
	}
	mutations := []struct {
		name  string
		apply func(*testing.T, string, string, string)
	}{
		{name: "modify", apply: func(t *testing.T, root, existing, _ string) {
			phase199WriteFixtureFile(t, root, path.Join(phase199GSDPath, existing), []byte("tampered\n"))
		}},
		{name: "add", apply: func(t *testing.T, root, _, addition string) {
			phase199WriteFixtureFile(t, root, path.Join(phase199GSDPath, addition), []byte("added\n"))
		}},
		{name: "remove", apply: func(t *testing.T, root, existing, _ string) {
			if err := os.Remove(filepath.Join(root, phase199GSDPath, filepath.FromSlash(existing))); err != nil {
				t.Fatalf("remove protected GSD fixture: %v", err)
			}
		}},
		{name: "replace type", apply: func(t *testing.T, root, existing, _ string) {
			target := filepath.Join(root, phase199GSDPath, filepath.FromSlash(existing))
			if err := os.Remove(target); err != nil {
				t.Fatalf("remove protected GSD fixture for type replacement: %v", err)
			}
			if err := os.Mkdir(target, 0o755); err != nil {
				t.Fatalf("replace protected GSD file with directory: %v", err)
			}
		}},
		{name: "replace symlink", apply: func(t *testing.T, root, existing, _ string) {
			target := filepath.Join(root, phase199GSDPath, filepath.FromSlash(existing))
			if err := os.Remove(target); err != nil {
				t.Fatalf("remove protected GSD fixture for symlink replacement: %v", err)
			}
			if err := os.Symlink("replacement-target", target); err != nil {
				t.Fatalf("replace protected GSD file with symlink: %v", err)
			}
		}},
	}
	for _, area := range areas {
		area := area
		for _, mutation := range mutations {
			mutation := mutation
			t.Run(area.name+"/"+mutation.name, func(t *testing.T) {
				changed := newPhase199ReceiptFixture(t)
				applyPhase199LaterLifecycle(t, &changed)
				mutation.apply(t, changed.Root, area.existing, area.addition)
				expectPhase199ReceiptValidationFailure(t, changed.Root, changed.Receipt, now)
			})
		}
	}
}

func TestPhase199GateReceipt(t *testing.T) {
	root := findTestModuleRoot(t)
	receipt := loadPhase199GateReceipt(t)
	if err := validatePhase199GateReceiptAtRoot(root, receipt, time.Now().UTC(), phase199GateReceiptFinalMode()); err != nil {
		t.Fatalf("validate Phase 199 gate receipt: %v", err)
	}
}

func phase199GateReceiptFinalMode() bool {
	testRun := flag.Lookup("test.run")
	return testRun != nil && testRun.Value.String() == "^TestPhase199GateReceipt$"
}

func validatePhase199GateReceiptForMode(receipt phase199GateReceipt, now time.Time, finalMode bool) error {
	if err := validatePhase199GateReceiptSchema(receipt, now); err != nil {
		return err
	}
	if finalMode && receipt.Status != "complete" {
		return fmt.Errorf("receipt status = %q, want complete for final receipt verification", receipt.Status)
	}
	return nil
}

func loadPhase199GateReceipt(t *testing.T) phase199GateReceipt {
	t.Helper()
	path := filepath.Join(findTestModuleRoot(t), filepath.FromSlash(phase199GateReceiptPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Phase 199 gate receipt: %v", err)
	}
	receipt, err := decodePhase199GateReceipt(raw)
	if err != nil {
		t.Fatalf("decode Phase 199 gate receipt: %v", err)
	}
	return receipt
}

func decodePhase199GateReceipt(raw []byte) (phase199GateReceipt, error) {
	var receipt phase199GateReceipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return phase199GateReceipt{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return phase199GateReceipt{}, fmt.Errorf("Phase 199 gate receipt contains multiple JSON values")
		}
		return phase199GateReceipt{}, fmt.Errorf("decode trailing Phase 199 gate receipt data: %w", err)
	}
	return receipt, nil
}

func validatePhase199GateReceiptAtRoot(root string, receipt phase199GateReceipt, now time.Time, finalMode bool) error {
	if err := validatePhase199GateReceiptForMode(receipt, now, finalMode); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := validatePhase199ReceiptRepository(root, receipt); err != nil {
		return fmt.Errorf("repository identity: %w", err)
	}
	actual, err := collectPhase199FingerprintsAtRoot(root, phase199ProtectedReceiptPaths)
	if err != nil {
		return fmt.Errorf("collect protected fingerprints: %w", err)
	}
	if !phase199FingerprintSetsEqual(receipt.Protected, actual) {
		return fmt.Errorf("protected ownership fingerprint changed or receipt is stale")
	}
	if err := validatePhase199ConfigAtRoot(root, receipt); err != nil {
		return fmt.Errorf("protected config: %w", err)
	}
	actualGSD, err := collectPhase199GSDBaseline(root)
	if err != nil {
		return fmt.Errorf("collect .gsd baseline: %w", err)
	}
	if !reflect.DeepEqual(receipt.GSDBaseline, actualGSD) {
		return fmt.Errorf(".gsd non-sentinel baseline changed")
	}
	return nil
}

func validatePhase199GateReceiptSchema(receipt phase199GateReceipt, now time.Time) error {
	if receipt.SchemaVersion != phase199GateReceiptVersion || receipt.Phase != "199" || receipt.Plan != "29" {
		return fmt.Errorf("unexpected receipt identity schema=%q phase=%q plan=%q", receipt.SchemaVersion, receipt.Phase, receipt.Plan)
	}
	if receipt.Status != "incomplete" && receipt.Status != "partial" && receipt.Status != "complete" {
		return fmt.Errorf("unsupported receipt status %q", receipt.Status)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, receipt.CreatedAt)
	if err != nil {
		return fmt.Errorf("invalid created_at: %w", err)
	}
	if createdAt.After(now.Add(phase199GateFutureSkew)) {
		return fmt.Errorf("created_at is more than %s in the future", phase199GateFutureSkew)
	}
	if !phase199SHA1Like(receipt.Repository.Revision) || !phase199SHA1Like(receipt.Repository.Tree) {
		return fmt.Errorf("repository revision/tree must be full git object IDs")
	}
	if err := validatePhase199HistoricalBlobs(receipt.Historical); err != nil {
		return err
	}
	if err := validatePhase199ProtectedFingerprints(receipt.Protected); err != nil {
		return err
	}
	if err := validatePhase199FingerprintSet(receipt.Preexisting, []string{phase199ConfigPath}, "pre-existing user change"); err != nil {
		return err
	}
	if phase199DigestBytes([]byte(receipt.ConfigBaseline)) != receipt.Preexisting[0].Before.Digest {
		return fmt.Errorf("config baseline bytes do not match recorded fingerprint")
	}
	if _, err := phase199DecodeJSONObject([]byte(receipt.ConfigBaseline)); err != nil {
		return fmt.Errorf("invalid config baseline: %w", err)
	}
	if err := validatePhase199GSDBaseline(receipt.GSDBaseline); err != nil {
		return err
	}
	if receipt.Status == "incomplete" {
		if len(receipt.Gates) != 0 {
			return fmt.Errorf("incomplete receipt must not contain gate evidence")
		}
		return nil
	}
	wantGateCount := 2
	if receipt.Status == "partial" {
		wantGateCount = 1
	}
	if len(receipt.Gates) != wantGateCount {
		return fmt.Errorf("%s receipt has %d gates, want exactly %d", receipt.Status, len(receipt.Gates), wantGateCount)
	}
	wantCommands := []string{"go test ./...", "go test ./... -race"}
	var previousFinished time.Time
	for index, want := range wantCommands[:wantGateCount] {
		gate := receipt.Gates[index]
		if gate.Command != want {
			return fmt.Errorf("gate %d command = %q, want exact %q", index, gate.Command, want)
		}
		if gate.ExitCode != 0 {
			return fmt.Errorf("gate %q exit code = %d, want 0", gate.Command, gate.ExitCode)
		}
		started, err := time.Parse(time.RFC3339Nano, gate.StartedAt)
		if err != nil {
			return fmt.Errorf("gate %q invalid started_at: %w", gate.Command, err)
		}
		finished, err := time.Parse(time.RFC3339Nano, gate.FinishedAt)
		if err != nil {
			return fmt.Errorf("gate %q invalid finished_at: %w", gate.Command, err)
		}
		if started.After(now.Add(phase199GateFutureSkew)) || finished.After(now.Add(phase199GateFutureSkew)) {
			return fmt.Errorf("gate %q timestamp is more than %s in the future", gate.Command, phase199GateFutureSkew)
		}
		if finished.Before(started) {
			return fmt.Errorf("gate %q finishes before it starts", gate.Command)
		}
		if finished.Sub(started) > phase199GateMaximumDuration {
			return fmt.Errorf("gate %q duration exceeds %s", gate.Command, phase199GateMaximumDuration)
		}
		if index == 0 && createdAt.After(started) {
			return fmt.Errorf("created_at occurs after first gate start")
		}
		if index > 0 && started.Before(previousFinished) {
			return fmt.Errorf("gate %q starts before the prior gate finished", gate.Command)
		}
		if gate.Revision != receipt.Repository.Revision || gate.Tree != receipt.Repository.Tree {
			return fmt.Errorf("gate %q revision/tree differs from receipt repository identity", gate.Command)
		}
		if !phase199SHA256(gate.OutputSHA256) {
			return fmt.Errorf("gate %q output digest is not a SHA-256", gate.Command)
		}
		previousFinished = finished
	}
	return nil
}

func validatePhase199HistoricalBlobs(blobs []phase199HistoricalBlobRecord) error {
	if len(blobs) != len(phase199HistoricalReceiptPaths) {
		return fmt.Errorf("historical blob count = %d, want %d", len(blobs), len(phase199HistoricalReceiptPaths))
	}
	for index, wantPath := range phase199HistoricalReceiptPaths {
		blob := blobs[index]
		if blob.Path != wantPath || !phase199SHA1Like(blob.Blob) || !phase199SHA256(blob.SHA256) {
			return fmt.Errorf("invalid historical blob evidence for %q", wantPath)
		}
	}
	return nil
}

func validatePhase199GSDBaseline(entries []phase199GSDEntry) error {
	if len(entries) == 0 || entries[0].Path != "." {
		return fmt.Errorf(".gsd baseline must include its root as the first entry")
	}
	previous := ""
	for _, entry := range entries {
		if entry.Path == "" || path.IsAbs(entry.Path) || path.Clean(entry.Path) != entry.Path || strings.HasPrefix(entry.Path, "../") {
			return fmt.Errorf("invalid .gsd baseline path %q", entry.Path)
		}
		if entry.Path == phase199GSDMutableSentinel {
			return fmt.Errorf("mutable sentinel must not appear in .gsd baseline")
		}
		if previous != "" && entry.Path <= previous {
			return fmt.Errorf(".gsd baseline paths are not strictly sorted")
		}
		if !phase199FileModeString(entry.Mode) {
			return fmt.Errorf("invalid .gsd mode for %q", entry.Path)
		}
		switch entry.Type {
		case "directory":
			if entry.Digest != "" || entry.SymlinkTarget != "" {
				return fmt.Errorf("directory %q has file-only attributes", entry.Path)
			}
		case "file":
			if !phase199SHA256(entry.Digest) || entry.SymlinkTarget != "" {
				return fmt.Errorf("invalid regular-file evidence for %q", entry.Path)
			}
		case "symlink":
			if entry.Digest != "" || entry.SymlinkTarget == "" {
				return fmt.Errorf("invalid symlink evidence for %q", entry.Path)
			}
		default:
			return fmt.Errorf("unsupported .gsd entry type %q for %q", entry.Type, entry.Path)
		}
		previous = entry.Path
	}
	return nil
}

func phase199FileModeString(value string) bool {
	if len(value) != 4 {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '7' {
			return false
		}
	}
	return true
}

func validatePhase199ProtectedFingerprints(fingerprints []phase199ProtectedFingerprintRecord) error {
	return validatePhase199FingerprintSet(fingerprints, phase199ProtectedReceiptPaths, "protected")
}

func validatePhase199FingerprintSet(fingerprints []phase199ProtectedFingerprintRecord, paths []string, label string) error {
	if len(fingerprints) != len(paths) {
		return fmt.Errorf("%s fingerprint count = %d, want %d", label, len(fingerprints), len(paths))
	}
	for index, wantPath := range paths {
		fingerprint := fingerprints[index]
		if fingerprint.Path != wantPath || !phase199PathFingerprintIsValid(fingerprint.Before) || !phase199PathFingerprintIsValid(fingerprint.After) {
			return fmt.Errorf("invalid protected fingerprint for %q", wantPath)
		}
		if !reflect.DeepEqual(fingerprint.Before, fingerprint.After) {
			return fmt.Errorf("protected fingerprint before/after mismatch for %q", wantPath)
		}
	}
	return nil
}

func phase199PathFingerprintIsValid(fingerprint phase199ProtectedPathFingerprint) bool {
	return fingerprint.Status != "" && fingerprint.Type != "" && fingerprint.Mode != "" && phase199SHA256(fingerprint.Digest)
}

func validatePhase199ReceiptRepository(root string, receipt phase199GateReceipt) error {
	repository := receipt.Repository
	objectType, err := phase199GitOutput(root, "cat-file", "-t", repository.Revision)
	if err != nil {
		return fmt.Errorf("resolve recorded revision: %w", err)
	}
	if strings.TrimSpace(string(objectType)) != "commit" {
		return fmt.Errorf("recorded revision %s is not a commit", repository.Revision)
	}
	tree, err := phase199GitOutput(root, "rev-parse", repository.Revision+"^{tree}")
	if err != nil {
		return fmt.Errorf("resolve recorded revision tree: %w", err)
	}
	if strings.TrimSpace(string(tree)) != repository.Tree {
		return fmt.Errorf("recorded tree %s does not belong to recorded revision %s", repository.Tree, repository.Revision)
	}
	for _, historical := range receipt.Historical {
		blob, err := phase199GitOutput(root, "rev-parse", repository.Revision+":"+historical.Path)
		if err != nil {
			return fmt.Errorf("resolve historical blob %q: %w", historical.Path, err)
		}
		resolvedBlob := strings.TrimSpace(string(blob))
		if resolvedBlob != historical.Blob {
			return fmt.Errorf("historical blob %q object = %s, want %s", historical.Path, resolvedBlob, historical.Blob)
		}
		blobType, err := phase199GitOutput(root, "cat-file", "-t", resolvedBlob)
		if err != nil {
			return fmt.Errorf("inspect historical blob %q: %w", historical.Path, err)
		}
		if strings.TrimSpace(string(blobType)) != "blob" {
			return fmt.Errorf("historical object %s for %q is not a blob", resolvedBlob, historical.Path)
		}
		contents, err := phase199GitOutput(root, "cat-file", "blob", resolvedBlob)
		if err != nil {
			return fmt.Errorf("read historical blob %q: %w", historical.Path, err)
		}
		if phase199DigestBytes(contents) != historical.SHA256 {
			return fmt.Errorf("historical blob %q digest differs from receipt", historical.Path)
		}
	}
	return nil
}

func phase199GitOutput(root string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func validatePhase199ConfigAtRoot(root string, receipt phase199GateReceipt) error {
	actual, err := phase199FingerprintForPath(root, phase199ConfigPath)
	if err != nil {
		return err
	}
	recorded := receipt.Preexisting[0].Before
	if reflect.DeepEqual(recorded, actual) {
		return nil
	}
	if actual.Type != recorded.Type || actual.Mode != recorded.Mode {
		return fmt.Errorf("config lstat attributes changed")
	}
	live, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(phase199ConfigPath)))
	if err != nil {
		return fmt.Errorf("read live config: %w", err)
	}
	baselineValue, err := phase199DecodeJSONObject([]byte(receipt.ConfigBaseline))
	if err != nil {
		return fmt.Errorf("decode recorded config baseline: %w", err)
	}
	liveValue, err := phase199DecodeJSONObject(live)
	if err != nil {
		return fmt.Errorf("decode live config: %w", err)
	}
	workflow, ok := baselineValue["workflow"].(map[string]any)
	if !ok {
		return fmt.Errorf("recorded config baseline lacks workflow object")
	}
	autoChain, ok := workflow["_auto_chain_active"].(bool)
	if !ok {
		return fmt.Errorf("recorded config baseline lacks boolean workflow._auto_chain_active")
	}
	if !autoChain {
		return fmt.Errorf("config bytes changed without an eligible auto-chain reset")
	}
	workflow["_auto_chain_active"] = false
	if !reflect.DeepEqual(baselineValue, liveValue) {
		return fmt.Errorf("config changed outside workflow._auto_chain_active true-to-false reset")
	}
	return nil
}

func phase199DecodeJSONObject(raw []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("expected JSON object")
	}
	return value, nil
}

func collectPhase199ProtectedFingerprints(t *testing.T, root string) []phase199ProtectedFingerprintRecord {
	t.Helper()
	return collectPhase199Fingerprints(t, root, phase199ProtectedReceiptPaths)
}

func collectPhase199Fingerprints(t *testing.T, root string, paths []string) []phase199ProtectedFingerprintRecord {
	t.Helper()
	result, err := collectPhase199FingerprintsAtRoot(root, paths)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func collectPhase199FingerprintsAtRoot(root string, paths []string) ([]phase199ProtectedFingerprintRecord, error) {
	result := make([]phase199ProtectedFingerprintRecord, 0, len(paths))
	for _, slashPath := range paths {
		fingerprint, err := phase199FingerprintForPath(root, slashPath)
		if err != nil {
			return nil, err
		}
		result = append(result, phase199ProtectedFingerprintRecord{Path: slashPath, Before: fingerprint, After: fingerprint})
	}
	return result, nil
}

func phase199ProtectedFingerprintForPath(t *testing.T, root, slashPath string) phase199ProtectedPathFingerprint {
	t.Helper()
	fingerprint, err := phase199FingerprintForPath(root, slashPath)
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}

func phase199FingerprintForPath(root, slashPath string) (phase199ProtectedPathFingerprint, error) {
	filePath := filepath.Join(root, filepath.FromSlash(slashPath))
	info, err := os.Lstat(filePath)
	if err != nil {
		return phase199ProtectedPathFingerprint{}, fmt.Errorf("stat protected path %s: %w", slashPath, err)
	}
	command := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all", "--", slashPath)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return phase199ProtectedPathFingerprint{}, fmt.Errorf("classify protected path %s: %w: %s", slashPath, err, strings.TrimSpace(string(output)))
	}
	status := "tracked-clean"
	if strings.TrimSpace(string(output)) != "" {
		status = "changed"
		if allPhase199StatusLinesUntracked(string(output)) {
			status = "untracked"
		}
	}
	digest, err := phase199ContentDigestAtPath(filePath, info)
	if err != nil {
		return phase199ProtectedPathFingerprint{}, fmt.Errorf("digest protected path %s: %w", slashPath, err)
	}
	return phase199ProtectedPathFingerprint{
		Status: status, Type: phase199FileType(info.Mode()),
		Mode: fmt.Sprintf("%04o", info.Mode().Perm()), Digest: digest,
	}, nil
}

func allPhase199StatusLinesUntracked(output string) bool {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if !strings.HasPrefix(line, "?? ") {
			return false
		}
	}
	return true
}

func phase199ContentDigest(t *testing.T, path string, info os.FileInfo) string {
	t.Helper()
	digest, err := phase199ContentDigestAtPath(path, info)
	if err != nil {
		t.Fatalf("digest protected tree %s: %v", path, err)
	}
	return digest
}

func phase199ContentDigestAtPath(filePath string, info os.FileInfo) (string, error) {
	if !info.IsDir() {
		return phase199FileDigestAtPath(filePath, info.Mode())
	}
	var records []string
	err := filepath.WalkDir(filePath, func(entryPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entryInfo, err := os.Lstat(entryPath)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(filePath, entryPath)
		if err != nil {
			return err
		}
		digest, err := phase199FileDigestAtPath(entryPath, entryInfo.Mode())
		if err != nil {
			return err
		}
		records = append(records, filepath.ToSlash(rel)+"\x00"+phase199FileType(entryInfo.Mode())+"\x00"+fmt.Sprintf("%04o", entryInfo.Mode().Perm())+"\x00"+digest)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(records)
	return phase199DigestBytes([]byte(strings.Join(records, "\n"))), nil
}

func phase199FileDigest(t *testing.T, path string, mode os.FileMode) string {
	t.Helper()
	digest, err := phase199FileDigestAtPath(path, mode)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func phase199FileDigestAtPath(filePath string, mode os.FileMode) (string, error) {
	if mode.IsDir() {
		return phase199DigestBytes(nil), nil
	}
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(filePath)
		if err != nil {
			return "", fmt.Errorf("read protected symlink %s: %w", filePath, err)
		}
		return phase199DigestBytes([]byte(target)), nil
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read protected file %s: %w", filePath, err)
	}
	return phase199DigestBytes(raw), nil
}

func phase199FileType(mode os.FileMode) string {
	switch {
	case mode.IsDir():
		return "directory"
	case mode&os.ModeSymlink != 0:
		return "symlink"
	case mode.IsRegular():
		return "file"
	default:
		return mode.Type().String()
	}
}

func collectPhase199GSDBaseline(root string) ([]phase199GSDEntry, error) {
	gsdRoot := filepath.Join(root, phase199GSDPath)
	rootInfo, err := os.Lstat(gsdRoot)
	if err != nil {
		return nil, fmt.Errorf("stat .gsd root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf(".gsd root must be a real directory")
	}
	rootEntry, err := phase199GSDEntryFor(gsdRoot, ".", rootInfo)
	if err != nil {
		return nil, err
	}
	entries := []phase199GSDEntry{rootEntry}
	var walk func(string, string) error
	walk = func(directory, relative string) error {
		children, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, child := range children {
			relativeChild := child.Name()
			if relative != "." {
				relativeChild = path.Join(relative, child.Name())
			}
			childPath := filepath.Join(directory, child.Name())
			info, err := os.Lstat(childPath)
			if err != nil {
				return err
			}
			if relativeChild != phase199GSDMutableSentinel {
				entry, err := phase199GSDEntryFor(childPath, relativeChild, info)
				if err != nil {
					return err
				}
				entries = append(entries, entry)
			}
			if info.IsDir() {
				if err := walk(childPath, relativeChild); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(gsdRoot, "."); err != nil {
		return nil, fmt.Errorf("walk .gsd baseline: %w", err)
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Path < entries[right].Path })
	return entries, nil
}

func phase199GSDEntryFor(filePath, relative string, info os.FileInfo) (phase199GSDEntry, error) {
	entry := phase199GSDEntry{
		Path: relative,
		Type: phase199FileType(info.Mode()),
		Mode: fmt.Sprintf("%04o", info.Mode().Perm()),
	}
	switch entry.Type {
	case "directory":
		return entry, nil
	case "file":
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return phase199GSDEntry{}, fmt.Errorf("read .gsd file %q: %w", relative, err)
		}
		entry.Digest = phase199DigestBytes(raw)
		return entry, nil
	case "symlink":
		target, err := os.Readlink(filePath)
		if err != nil {
			return phase199GSDEntry{}, fmt.Errorf("read .gsd symlink %q: %w", relative, err)
		}
		entry.SymlinkTarget = target
		return entry, nil
	default:
		return phase199GSDEntry{}, fmt.Errorf("unsupported .gsd entry type %q for %q", entry.Type, relative)
	}
}

func phase199DigestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func phase199SHA1Like(value string) bool {
	return len(value) == 40 && strings.Trim(value, "0123456789abcdef") == ""
}
func phase199SHA256(value string) bool {
	return len(value) == 64 && strings.Trim(value, "0123456789abcdef") == ""
}
func phase199FingerprintSetsEqual(left, right []phase199ProtectedFingerprintRecord) bool {
	return reflect.DeepEqual(left, right)
}

func phase199ValidCompleteReceipt(t *testing.T) phase199GateReceipt {
	t.Helper()
	base := loadPhase199GateReceipt(t)
	base.Status = "complete"
	now := time.Now().UTC()
	base.Gates = []phase199GateRun{
		{Command: "go test ./...", ExitCode: 0, StartedAt: now.Add(-2 * time.Minute).Format(time.RFC3339Nano), FinishedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), Revision: base.Repository.Revision, Tree: base.Repository.Tree, OutputSHA256: strings.Repeat("a", 64)},
		{Command: "go test ./... -race", ExitCode: 0, StartedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), FinishedAt: now.Format(time.RFC3339Nano), Revision: base.Repository.Revision, Tree: base.Repository.Tree, OutputSHA256: strings.Repeat("b", 64)},
	}
	return base
}

func phase199PartialReceipt(t *testing.T) phase199GateReceipt {
	t.Helper()
	receipt := phase199ValidCompleteReceipt(t)
	receipt.Status = "partial"
	receipt.Gates = receipt.Gates[:1]
	return receipt
}

func phase199IncompleteReceipt(t *testing.T) phase199GateReceipt {
	t.Helper()
	receipt := phase199ValidCompleteReceipt(t)
	receipt.Status = "incomplete"
	receipt.Gates = nil
	return receipt
}

func clonePhase199GateReceipt(t *testing.T, receipt phase199GateReceipt) phase199GateReceipt {
	t.Helper()
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var clone phase199GateReceipt
	if err := json.Unmarshal(raw, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

type phase199ReceiptFixture struct {
	Root    string
	Receipt phase199GateReceipt
}

func newPhase199ReceiptFixture(t *testing.T) phase199ReceiptFixture {
	t.Helper()
	root := t.TempDir()
	phase199RunGit(t, root, "init", "--quiet")
	phase199RunGit(t, root, "config", "user.name", "Phase 199 Test")
	phase199RunGit(t, root, "config", "user.email", "phase199@example.invalid")

	phase199WriteFixtureFile(t, root, phase199HistoricalStatePath, []byte("status: phase-199-complete\n"))
	phase199WriteFixtureFile(t, root, phase199ConfigPath, []byte("{\"workflow\":{\"_auto_chain_active\":false,\"auto_advance\":false}}\n"))
	phase199RunGit(t, root, "add", phase199HistoricalStatePath, phase199ConfigPath)
	phase199RunGit(t, root, "commit", "--quiet", "-m", "historical Phase 199 state")

	phase199WriteFixtureFile(t, root, phase199ConfigPath, []byte("{\"workflow\":{\"_auto_chain_active\":true,\"auto_advance\":false}}\n"))
	phase199WriteFixtureFile(t, root, phase199PatternsPath, []byte("owner evidence\n"))
	phase199WriteFixtureFile(t, root, path.Join(phase199GSDPath, phase199GSDMutableSentinel), []byte("{\"generation\":1}\n"))
	phase199WriteFixtureFile(t, root, path.Join(phase199GSDPath, "scratch/blocker1.txt"), []byte("blocker\n"))
	phase199WriteFixtureFile(t, root, path.Join(phase199GSDPath, "scratch/decision1.txt"), []byte("decision\n"))
	phase199WriteFixtureFile(t, root, path.Join(phase199GSDPath, "worktrees/phase-198-3/wave-1-manifest.json"), []byte("{\"wave\":1}\n"))

	revision := phase199RunGit(t, root, "rev-parse", "HEAD")
	tree := phase199RunGit(t, root, "rev-parse", "HEAD^{tree}")
	blob := phase199RunGit(t, root, "rev-parse", revision+":"+phase199HistoricalStatePath)
	stateContents, err := phase199GitOutput(root, "cat-file", "blob", blob)
	if err != nil {
		t.Fatalf("read fixture historical state: %v", err)
	}
	protected, err := collectPhase199FingerprintsAtRoot(root, phase199ProtectedReceiptPaths)
	if err != nil {
		t.Fatalf("collect fixture protected fingerprints: %v", err)
	}
	preexisting, err := collectPhase199FingerprintsAtRoot(root, []string{phase199ConfigPath})
	if err != nil {
		t.Fatalf("collect fixture config fingerprint: %v", err)
	}
	configBaseline, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(phase199ConfigPath)))
	if err != nil {
		t.Fatalf("read fixture config baseline: %v", err)
	}
	gsdBaseline, err := collectPhase199GSDBaseline(root)
	if err != nil {
		t.Fatalf("collect fixture .gsd baseline: %v", err)
	}
	now := time.Now().UTC()
	receipt := phase199GateReceipt{
		SchemaVersion: phase199GateReceiptVersion,
		Phase:         "199",
		Plan:          "29",
		Status:        "complete",
		CreatedAt:     now.Add(-72 * time.Hour).Format(time.RFC3339Nano),
		Repository:    phase199ReceiptRepository{Revision: revision, Tree: tree},
		Historical: []phase199HistoricalBlobRecord{{
			Path: phase199HistoricalStatePath, Blob: blob, SHA256: phase199DigestBytes(stateContents),
		}},
		Protected:      protected,
		Preexisting:    preexisting,
		ConfigBaseline: string(configBaseline),
		GSDBaseline:    gsdBaseline,
		Gates: []phase199GateRun{
			{Command: "go test ./...", ExitCode: 0, StartedAt: now.Add(-71 * time.Hour).Format(time.RFC3339Nano), FinishedAt: now.Add(-70 * time.Hour).Format(time.RFC3339Nano), Revision: revision, Tree: tree, OutputSHA256: strings.Repeat("a", 64)},
			{Command: "go test ./... -race", ExitCode: 0, StartedAt: now.Add(-69 * time.Hour).Format(time.RFC3339Nano), FinishedAt: now.Add(-68 * time.Hour).Format(time.RFC3339Nano), Revision: revision, Tree: tree, OutputSHA256: strings.Repeat("b", 64)},
		},
	}
	if err := validatePhase199GateReceiptAtRoot(root, receipt, now, true); err != nil {
		t.Fatalf("validate fresh Phase 199 receipt fixture: %v", err)
	}
	return phase199ReceiptFixture{Root: root, Receipt: receipt}
}

func applyPhase199LaterLifecycle(t *testing.T, fixture *phase199ReceiptFixture) {
	t.Helper()
	phase199WriteFixtureFile(t, fixture.Root, phase199HistoricalStatePath, []byte("status: later-phase-active\n"))
	phase199WriteFixtureFile(t, fixture.Root, ".planning/phases/200-later/200-01-SUMMARY.md", []byte("# Later planning evidence\n"))
	phase199RunGit(t, fixture.Root, "add", phase199HistoricalStatePath, ".planning/phases/200-later/200-01-SUMMARY.md")
	phase199RunGit(t, fixture.Root, "commit", "--quiet", "-m", "later lifecycle bookkeeping")
	phase199WriteFixtureFile(t, fixture.Root, path.Join(phase199GSDPath, phase199GSDMutableSentinel), []byte("{\"generation\":2}\n"))
	phase199WriteFixtureFile(t, fixture.Root, phase199ConfigPath, []byte("{\"workflow\":{\"_auto_chain_active\":false,\"auto_advance\":false}}\n"))
}

func expectPhase199ReceiptValidationFailure(t *testing.T, root string, receipt phase199GateReceipt, now time.Time) {
	t.Helper()
	first := validatePhase199GateReceiptAtRoot(root, receipt, now, true)
	second := validatePhase199GateReceiptAtRoot(root, receipt, now, true)
	if first == nil || second == nil {
		t.Fatal("tampered receipt environment unexpectedly passed validation")
	}
	if first.Error() != second.Error() {
		t.Fatalf("tamper failure was not deterministic: first=%q second=%q", first, second)
	}
}

func phase199WriteFixtureFile(t *testing.T, root, slashPath string, raw []byte) {
	t.Helper()
	filePath := filepath.Join(root, filepath.FromSlash(slashPath))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create fixture parent for %s: %v", slashPath, err)
	}
	if err := os.WriteFile(filePath, raw, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", slashPath, err)
	}
}

func phase199RunGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	output, err := phase199GitOutput(root, args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(output))
}

func phase199MustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse fixture time %q: %v", value, err)
	}
	return parsed
}
