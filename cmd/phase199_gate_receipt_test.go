package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	phase199GateReceiptPath    = ".planning/phases/199-front-door-and-classic-contract/199-GATE-RECEIPT.json"
	phase199GateReceiptVersion = "phase199-gate-receipt/v1"
)

var phase199ProtectedReceiptPaths = []string{
	".planning/STATE.md",
	".gsd",
	".planning/phases/199-front-door-and-classic-contract/199-PATTERNS.md",
}

type phase199GateReceipt struct {
	SchemaVersion string                               `json:"schema_version"`
	Phase         string                               `json:"phase"`
	Plan          string                               `json:"plan"`
	Status        string                               `json:"status"`
	CreatedAt     string                               `json:"created_at"`
	Repository    phase199ReceiptRepository            `json:"repository"`
	Protected     []phase199ProtectedFingerprintRecord `json:"protected_fingerprints"`
	Gates         []phase199GateRun                    `json:"gates"`
}

type phase199ReceiptRepository struct {
	Revision string `json:"revision"`
	Tree     string `json:"tree"`
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
		t.Fatalf("validate checked-in incomplete receipt schema: %v", err)
	}

	complete := phase199ValidCompleteReceipt(t)
	for name, mutate := range map[string]func(*phase199GateReceipt){
		"missing gate": func(r *phase199GateReceipt) { r.Gates = r.Gates[:1] },
		"stale gate": func(r *phase199GateReceipt) {
			r.Gates[0].StartedAt = time.Now().UTC().Add(-25 * time.Hour).Format(time.RFC3339Nano)
		},
		"nonzero exit":       func(r *phase199GateReceipt) { r.Gates[0].ExitCode = 1 },
		"substitute command": func(r *phase199GateReceipt) { r.Gates[1].Command = "go test ./...  -race" },
		"bad digest":         func(r *phase199GateReceipt) { r.Gates[0].OutputSHA256 = "not-a-sha256" },
		"wrong tree":         func(r *phase199GateReceipt) { r.Gates[0].Tree = strings.Repeat("0", 40) },
		"changed protected state": func(r *phase199GateReceipt) {
			r.Protected[0].After.Digest = strings.Repeat("0", 64)
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
	for name, receipt := range map[string]phase199GateReceipt{
		"incomplete": receipt,
		"partial":    phase199PartialReceipt(t),
	} {
		t.Run("final mode rejects "+name, func(t *testing.T) {
			if err := validatePhase199GateReceiptForMode(receipt, time.Now().UTC(), true); err == nil {
				t.Fatalf("final-mode validator accepted %s receipt", name)
			}
		})
	}
}

func TestPhase199GateReceipt(t *testing.T) {
	root := findTestModuleRoot(t)
	receipt := loadPhase199GateReceipt(t)
	finalMode := phase199GateReceiptFinalMode()
	if err := validatePhase199GateReceiptForMode(receipt, time.Now().UTC(), finalMode); err != nil {
		t.Fatalf("gate receipt schema: %v", err)
	}
	if err := validatePhase199ReceiptRepository(root, receipt.Repository); err != nil {
		t.Fatalf("gate receipt repository identity: %v", err)
	}
	actual := collectPhase199ProtectedFingerprints(t, root)
	if !phase199FingerprintSetsEqual(receipt.Protected, actual) {
		t.Fatalf("protected ownership fingerprint changed or receipt is stale\nreceipt: %#v\nactual:  %#v", receipt.Protected, actual)
	}
	if err := validatePhase199ReceiptEvidenceOnlyChanges(root, receipt.Repository.Revision); err != nil {
		t.Fatalf("gate receipt source freshness: %v", err)
	}
}

func phase199GateReceiptFinalMode() bool {
	return flag.Lookup("test.run").Value.String() == "^TestPhase199GateReceipt$"
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
	var receipt phase199GateReceipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		t.Fatalf("decode Phase 199 gate receipt: %v", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		t.Fatal("Phase 199 gate receipt contains multiple JSON values")
	}
	return receipt
}

func validatePhase199GateReceiptSchema(receipt phase199GateReceipt, now time.Time) error {
	if receipt.SchemaVersion != phase199GateReceiptVersion || receipt.Phase != "199" || receipt.Plan != "29" {
		return fmt.Errorf("unexpected receipt identity schema=%q phase=%q plan=%q", receipt.SchemaVersion, receipt.Phase, receipt.Plan)
	}
	if receipt.Status != "incomplete" && receipt.Status != "partial" && receipt.Status != "complete" {
		return fmt.Errorf("unsupported receipt status %q", receipt.Status)
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.CreatedAt); err != nil {
		return fmt.Errorf("invalid created_at: %w", err)
	}
	if !phase199SHA1Like(receipt.Repository.Revision) || !phase199SHA1Like(receipt.Repository.Tree) {
		return fmt.Errorf("repository revision/tree must be full git object IDs")
	}
	if err := validatePhase199ProtectedFingerprints(receipt.Protected); err != nil {
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
		if err != nil || finished.Before(started) || finished.After(now.Add(time.Minute)) || now.Sub(started) > 24*time.Hour {
			return fmt.Errorf("gate %q has stale or invalid execution timestamps", gate.Command)
		}
		if gate.Revision != receipt.Repository.Revision || gate.Tree != receipt.Repository.Tree {
			return fmt.Errorf("gate %q revision/tree differs from receipt repository identity", gate.Command)
		}
		if !phase199SHA256(gate.OutputSHA256) {
			return fmt.Errorf("gate %q output digest is not a SHA-256", gate.Command)
		}
	}
	return nil
}

func validatePhase199ProtectedFingerprints(fingerprints []phase199ProtectedFingerprintRecord) error {
	if len(fingerprints) != len(phase199ProtectedReceiptPaths) {
		return fmt.Errorf("protected fingerprint count = %d, want %d", len(fingerprints), len(phase199ProtectedReceiptPaths))
	}
	for index, wantPath := range phase199ProtectedReceiptPaths {
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

func validatePhase199ReceiptRepository(root string, repository phase199ReceiptRepository) error {
	output, err := exec.Command("git", "rev-parse", repository.Revision+"^{tree}").CombinedOutput()
	if err != nil {
		return fmt.Errorf("resolve recorded revision: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if strings.TrimSpace(string(output)) != repository.Tree {
		return fmt.Errorf("recorded tree %s does not belong to recorded revision %s", repository.Tree, repository.Revision)
	}
	return nil
}

func validatePhase199ReceiptEvidenceOnlyChanges(root, revision string) error {
	command := exec.Command("git", "diff", "--name-only", revision)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("list changes since tested revision: %w", err)
	}
	allowed := map[string]bool{
		phase199GateReceiptPath: true,
		".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.md": true,
		".planning/phases/199-front-door-and-classic-contract/199-29-SUMMARY.md":       true,
	}
	for _, path := range strings.Fields(string(output)) {
		if !allowed[path] {
			return fmt.Errorf("post-tested change %q is not explicitly evidence-only", path)
		}
	}
	return nil
}

func collectPhase199ProtectedFingerprints(t *testing.T, root string) []phase199ProtectedFingerprintRecord {
	t.Helper()
	result := make([]phase199ProtectedFingerprintRecord, 0, len(phase199ProtectedReceiptPaths))
	for _, path := range phase199ProtectedReceiptPaths {
		fingerprint := phase199ProtectedFingerprintForPath(t, root, path)
		result = append(result, phase199ProtectedFingerprintRecord{Path: path, Before: fingerprint, After: fingerprint})
	}
	return result
}

func phase199ProtectedFingerprintForPath(t *testing.T, root, slashPath string) phase199ProtectedPathFingerprint {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(slashPath))
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat protected path %s: %v", slashPath, err)
	}
	command := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all", "--", slashPath)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("classify protected path %s: %v", slashPath, err)
	}
	status := "tracked-clean"
	if strings.TrimSpace(string(output)) != "" {
		status = "changed"
		if allPhase199StatusLinesUntracked(string(output)) {
			status = "untracked"
		}
	}
	return phase199ProtectedPathFingerprint{
		Status: status, Type: phase199FileType(info.Mode()),
		Mode: fmt.Sprintf("%04o", info.Mode().Perm()), Digest: phase199ContentDigest(t, path, info),
	}
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
	if !info.IsDir() {
		return phase199FileDigest(t, path, info.Mode())
	}
	var records []string
	err := filepath.WalkDir(path, func(entryPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(path, entryPath)
		if err != nil {
			return err
		}
		records = append(records, filepath.ToSlash(rel)+"\x00"+phase199FileType(entryInfo.Mode())+"\x00"+fmt.Sprintf("%04o", entryInfo.Mode().Perm())+"\x00"+phase199FileDigest(t, entryPath, entryInfo.Mode()))
		return nil
	})
	if err != nil {
		t.Fatalf("digest protected tree %s: %v", path, err)
	}
	sort.Strings(records)
	return phase199DigestBytes([]byte(strings.Join(records, "\n")))
}

func phase199FileDigest(t *testing.T, path string, mode os.FileMode) string {
	t.Helper()
	if mode.IsDir() {
		return phase199DigestBytes(nil)
	}
	if mode&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			t.Fatalf("read protected symlink %s: %v", path, err)
		}
		return phase199DigestBytes([]byte(target))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read protected file %s: %v", path, err)
	}
	return phase199DigestBytes(raw)
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
