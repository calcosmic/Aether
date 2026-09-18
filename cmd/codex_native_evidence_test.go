package cmd

// Final phase evidence delegates scenario proof to the reviewed raw replay engine.
// Deterministic controls establish validator behavior, never actual host success.

import (
	"bufio"
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

const nativeEvidenceDirectory = ".planning/phases/204.2-codex-native-worker-lifecycle/evidence"
const nativeGapEvidenceDirectory = nativeEvidenceDirectory + "/gap-closure"

func nativeEvidencePaths(root, explicit string) (string, string) {
	if explicit == "" {
		explicit = filepath.Join(root, nativeGapEvidenceDirectory, "native-qualification.json")
	}
	return explicit, filepath.Join(root, nativeGapEvidenceDirectory, "native-regression.json")
}

type nativeEvidenceFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type nativeEvidenceCase struct {
	ContextProtocol           string            `json:"context_protocol,omitempty"`
	Outcome                   string            `json:"outcome"`
	SourceRevision            string            `json:"source_revision"`
	SourceStatus              string            `json:"source_status"`
	SourceDigest              string            `json:"source_digest"`
	HarnessSHA256             string            `json:"harness_sha256"`
	OriginalReceipt           string            `json:"original_receipt"`
	OriginalReceiptSHA256     string            `json:"original_receipt_sha256"`
	DerivedReceipt            string            `json:"derived_receipt"`
	DerivedReceiptSHA256      string            `json:"derived_receipt_sha256"`
	ProductionInventory       string            `json:"production_inventory"`
	ProductionInventorySHA256 string            `json:"production_inventory_sha256"`
	ArtifactsVerified         bool              `json:"artifacts_verified"`
	ArtifactCount             int               `json:"artifact_count"`
	InstalledFiles            map[string]string `json:"installed_files"`
}

type nativeEvidenceQualification struct {
	ProofContract             string                        `json:"proof_contract,omitempty"`
	ProofAmendmentSHA256      string                        `json:"proof_amendment_sha256,omitempty"`
	FrozenProvenance          nativeEvidenceFile            `json:"frozen_provenance"`
	SchemaVersion             string                        `json:"schema_version"`
	TestedSource              string                        `json:"tested_source"`
	SourceFrozenDuringCapture bool                          `json:"source_frozen_during_capture"`
	SourceFrozenDuringReplay  bool                          `json:"source_frozen_during_replay"`
	ScenarioCount             int                           `json:"scenario_count"`
	ExpectedScenarios         []string                      `json:"expected_scenarios"`
	Cases                     map[string]nativeEvidenceCase `json:"cases"`
	QualificationComplete     bool                          `json:"qualification_complete"`
	MatrixLog                 string                        `json:"matrix_log"`
	MatrixLogSHA256           string                        `json:"matrix_log_sha256"`
	ReplayManifest            string                        `json:"replay_manifest"`
	ReplayManifestSHA256      string                        `json:"replay_manifest_sha256"`
	NegativeControls          struct {
		Manifest string `json:"manifest"`
		SHA256   string `json:"sha256"`
	} `json:"negative_controls"`
	ValidatorIdentity struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Source string `json:"source"`
	} `json:"validator_identity"`
	MatrixInvocation struct {
		ExitCode          *int   `json:"exit_code"`
		SourceAfter       string `json:"source_after"`
		SourceStatusAfter string `json:"source_status_after"`
	} `json:"matrix_invocation"`
}

func TestCodexNativePhaseEvidence(t *testing.T) {
	root := antSkillSourceRoot(t)
	path, regressionPath := nativeEvidencePaths(root, os.Getenv("AETHER_CODEX_NATIVE_RECEIPT_PATH"))
	// No opt-in skip, fixture fallback, EXPECT_INCOMPLETE escape, or ambient
	// environment rewriting. Missing default evidence is also a failure.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("required actual native qualification: %v", err)
	}
	var qualification nativeEvidenceQualification
	if err := json.Unmarshal(raw, &qualification); err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	production := liveSkillSourceIdentity(t, root, scratch)
	var currentInventory map[string]string
	if err := nativeEvidenceReadJSON(filepath.Join(scratch, "source-files.json"), &currentInventory); err != nil {
		t.Fatal(err)
	}
	harness := liveSkillFileDigest(t, filepath.Join(root, "cmd", "codex_native_worker_live_test.go"))
	if err := nativeEvidenceQualificationCheck(t, root, qualification, production, harness, currentInventory); err != nil {
		t.Errorf("required actual native qualification failed: %v", err)
	}
	if os.Getenv("AETHER_CODEX_NATIVE_REQUIRE_REGRESSION") == "1" {
		if err := nativeEvidenceRegressionCheck(t, root, regressionPath, lifecycleDigest(raw), production, harness); err != nil {
			t.Errorf("required completed normal/race regression accounting failed: %v", err)
		}
	}
}

func nativeEvidenceQualificationCheck(t *testing.T, sourceRoot string, q nativeEvidenceQualification, production, harness string, currentInventory map[string]string) error {
	t.Helper()
	if q.SchemaVersion != "aether-native-final-qualification/v1" || len(q.TestedSource) != 40 ||
		!q.SourceFrozenDuringCapture || !q.SourceFrozenDuringReplay {
		return fmt.Errorf("qualification schema/source freeze identity missing")
	}
	expected := make([]string, 0, len(codexNativeLiveScenarios))
	for _, scenario := range codexNativeLiveScenarios {
		expected = append(expected, scenario.Name)
	}
	if q.ScenarioCount != len(expected) || len(q.Cases) != len(expected) || !reflect.DeepEqual(q.ExpectedScenarios, expected) {
		return fmt.Errorf("complete required scenario inventory differs from reviewed host matrix")
	}
	var failures []error
	if err := nativeGapQualificationOutcomes(q); err != nil {
		failures = append(failures, err)
	}
	if err := nativeGapFrozenProvenanceCheck(sourceRoot, q, production, harness); err != nil {
		failures = append(failures, err)
	}
	if !q.QualificationComplete {
		failures = append(failures, fmt.Errorf("qualification_complete is false; mandatory incomplete cases are current gaps, never historical waivers"))
	}
	if q.MatrixInvocation.ExitCode == nil || *q.MatrixInvocation.ExitCode != 0 ||
		q.MatrixInvocation.SourceAfter != q.TestedSource || q.MatrixInvocation.SourceStatusAfter != "" {
		failures = append(failures, fmt.Errorf("actual matrix did not complete successfully on frozen capture source"))
	}
	matrix, err := nativeEvidenceBytes(nativeEvidenceFile{q.MatrixLog, q.MatrixLogSHA256})
	if err != nil {
		failures = append(failures, err)
	} else if err := nativeEvidenceMatrixEvents(matrix, expected); err != nil {
		failures = append(failures, err)
	}
	if q.ValidatorIdentity.Source != q.TestedSource {
		failures = append(failures, fmt.Errorf("aggregate validator/capture source mismatch"))
	}
	if _, err := nativeEvidenceBytes(nativeEvidenceFile{q.ValidatorIdentity.Path, q.ValidatorIdentity.SHA256}); err != nil {
		failures = append(failures, err)
	}
	if err := nativeEvidenceReplayManifest(q); err != nil {
		failures = append(failures, err)
	}
	if err := nativeEvidenceMutationManifest(sourceRoot, nativeEvidenceFile{q.NegativeControls.Manifest, q.NegativeControls.SHA256}); err != nil {
		failures = append(failures, err)
	}
	for _, name := range expected {
		c, ok := q.Cases[name]
		if !ok {
			failures = append(failures, fmt.Errorf("%s: case absent", name))
			continue
		}
		if err := nativeEvidenceCaseCheck(t, name, c, q.TestedSource, production, harness, currentInventory); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(failures...)
}

func nativeEvidenceCaseCheck(t *testing.T, name string, c nativeEvidenceCase, source, production, harness string, currentInventory map[string]string) error {
	t.Helper()
	originalRaw, err := nativeEvidenceBytes(nativeEvidenceFile{c.OriginalReceipt, c.OriginalReceiptSHA256})
	if err != nil {
		return err
	}
	derivedRaw, err := nativeEvidenceBytes(nativeEvidenceFile{c.DerivedReceipt, c.DerivedReceiptSHA256})
	if err != nil {
		return err
	}
	var original, retained codexNativeLiveReceipt
	if err := json.Unmarshal(originalRaw, &original); err != nil {
		return err
	}
	if err := json.Unmarshal(derivedRaw, &retained); err != nil {
		return err
	}
	if !nativeGapReceiptSchemas(original.SchemaVersion, retained.SchemaVersion) || original.Scenario != name || retained.Scenario != name {
		return fmt.Errorf("wrong raw receipt schema/scenario")
	}
	if err := nativeEvidenceContextProtocolBinding(c.ContextProtocol, original.ContextProtocol, retained.ContextProtocol); err != nil {
		return err
	}
	if err := nativeGapProofBinding(retained.ProofContract, retained.ProofAmendmentSHA256, original.ProofContract, original.ProofAmendmentSHA256); err != nil {
		return fmt.Errorf("derived receipt changed the original proof contract: %w", err)
	}
	if err := nativeEvidenceReceiptIdentity(original, source, production, harness); err != nil {
		return err
	}
	if err := nativeEvidenceReceiptIdentity(retained, source, production, harness); err != nil {
		return err
	}
	if c.SourceRevision != source || c.SourceStatus != "" || c.SourceDigest != production || c.HarnessSHA256 != harness {
		return fmt.Errorf("aggregate case/source/production/harness binding differs")
	}
	if retained.ValidationRevision != source || retained.ValidationOriginalReceipt != c.OriginalReceipt ||
		retained.ValidationOriginalSHA256 != lifecycleDigest(originalRaw) {
		return fmt.Errorf("derived receipt no longer names the exact original and reviewed validator")
	}
	if !c.ArtifactsVerified || c.ArtifactCount != len(retained.Artifacts) {
		return fmt.Errorf("aggregate artifact accounting differs")
	}
	if err := nativeEvidenceArtifacts(retained); err != nil {
		return err
	}
	inventoryRaw, err := nativeEvidenceBytes(nativeEvidenceFile{c.ProductionInventory, c.ProductionInventorySHA256})
	if err != nil {
		return err
	}
	var inventory map[string]string
	if err := json.Unmarshal(inventoryRaw, &inventory); err != nil {
		return err
	}
	// Comparison uses relative production paths, not checkout location or HEAD.
	// Test/report-only commits remain allowed; test harness bytes are pinned separately.
	if !reflect.DeepEqual(inventory, currentInventory) {
		return fmt.Errorf("captured production inventory differs from current candidate inputs")
	}
	if original.Artifacts[c.ProductionInventory] != lifecycleDigest(inventoryRaw) {
		return fmt.Errorf("production inventory absent from original capture")
	}
	installed := make(map[string]string)
	for path, digest := range original.Artifacts {
		if strings.Contains(filepath.ToSlash(path), "/home/") &&
			(strings.Contains(path, "/skills/") || strings.Contains(path, "/agents/") ||
				strings.Contains(path, "/commands/") || strings.Contains(path, "/support/")) {
			installed[path] = digest
		}
	}
	if len(installed) == 0 || !reflect.DeepEqual(installed, c.InstalledFiles) {
		return fmt.Errorf("installed skill/agent/support inventory differs or is empty")
	}
	if err := nativeEvidenceCandidateProvenance(original, filepath.Dir(c.OriginalReceipt)); err != nil {
		return err
	}
	// This is the only per-scenario proof engine. It discards cached assertions,
	// derives facts from inventoried raw host/child evidence, and calls the
	// existing reviewed validators. Writes are confined to testing temp dirs.
	replayed := original
	if err := nativeEvidenceReplayRaw(t, &replayed, source, production, harness); err != nil {
		return fmt.Errorf("actual original raw replay: %w", err)
	}
	if replayed.Outcome != "passed" && replayed.Outcome != "observed" {
		return fmt.Errorf("actual raw replay remained unqualified: %s", replayed.Outcome)
	}
	if c.Outcome != replayed.Outcome || retained.Outcome != replayed.Outcome {
		return fmt.Errorf("aggregate/retained outcome differs from fresh raw replay")
	}
	// Prove retained derived identities are not disconnected from the new replay.
	for label, pair := range map[string][2]string{
		"parent":  {retained.SessionID, replayed.SessionID},
		"child":   {retained.ChildID, replayed.ChildID},
		"attempt": {retained.AttemptID, replayed.AttemptID},
		"launch":  {retained.LaunchID, replayed.LaunchID},
		"result":  {retained.ResultSHA256, replayed.ResultSHA256},
		"resume":  {retained.ResumeSessionID, replayed.ResumeSessionID},
	} {
		if pair[0] != pair[1] {
			return fmt.Errorf("retained %s identity differs from raw replay", label)
		}
	}
	return nil
}

// Context transport belongs to each case: Claude and historical native cases
// may be empty while newly captured native cases pin child-fetch/v1. Neither
// the aggregate nor a derived replay may change the original case's protocol.
func nativeEvidenceContextProtocolBinding(aggregate, original, derived string) error {
	for _, protocol := range []string{aggregate, original, derived} {
		if protocol != "" && protocol != codexNativeContextProtocolChildFetch {
			return fmt.Errorf("unknown context protocol in aggregate/original/derived evidence")
		}
	}
	if aggregate != original || derived != original {
		return fmt.Errorf("aggregate or derived context protocol differs from original evidence")
	}
	return nil
}

func nativeEvidenceReceiptIdentity(r codexNativeLiveReceipt, source, production, harness string) error {
	if r.ContextProtocol != "" && r.ContextProtocol != codexNativeContextProtocolChildFetch {
		return fmt.Errorf("unknown native context proof protocol")
	}
	if _, err := nativeGapProofIdentity(r.ProofContract, r.ProofAmendmentSHA256); err != nil {
		return err
	}
	if r.SourceRevision != source || r.SourceStatus != "" {
		return fmt.Errorf("capture source binding changed")
	}
	if r.SourceDigest != production || r.HarnessSHA256 != harness {
		return fmt.Errorf("capture production or harness binding changed")
	}
	return nil
}

func nativeEvidenceReplayRaw(t *testing.T, r *codexNativeLiveReceipt, source, production, harness string) error {
	t.Helper()
	if err := nativeEvidenceReceiptIdentity(*r, source, production, harness); err != nil {
		return err
	}
	if err := nativeEvidenceArtifacts(*r); err != nil {
		return err
	}
	return nativeReplayQualificationReceipt(t, r)
}

func nativeEvidenceArtifacts(r codexNativeLiveReceipt) error {
	if len(r.Artifacts) == 0 {
		return fmt.Errorf("actual raw artifact inventory absent")
	}
	paths := make([]string, 0, len(r.Artifacts))
	for path := range r.Artifacts {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if _, err := nativeEvidenceBytes(nativeEvidenceFile{path, r.Artifacts[path]}); err != nil {
			return err
		}
	}
	return nil
}

func nativeEvidenceBytes(ref nativeEvidenceFile) ([]byte, error) {
	if ref.Path == "" || ref.SHA256 == "" {
		return nil, fmt.Errorf("required evidence path/digest absent")
	}
	raw, err := os.ReadFile(ref.Path)
	if err != nil {
		return nil, fmt.Errorf("required evidence %s: %w", ref.Path, err)
	}
	want := "sha256:" + strings.TrimPrefix(ref.SHA256, "sha256:")
	if lifecycleDigest(raw) != want {
		return nil, fmt.Errorf("evidence digest changed: %s", ref.Path)
	}
	return raw, nil
}

func nativeEvidenceReadJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func nativeEvidenceCandidateProvenance(r codexNativeLiveReceipt, root string) error {
	path := filepath.Join(root, "candidate-provenance.json")
	raw, err := nativeEvidenceBytes(nativeEvidenceFile{path, r.Artifacts[path]})
	if err != nil {
		return err
	}
	var p struct {
		SourceRevision string   `json:"source_revision"`
		SourceDigest   string   `json:"source_digest"`
		BinarySHA256   string   `json:"binary_sha256"`
		BuildArgv      []string `json:"build_argv"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.SourceRevision != r.SourceRevision || p.SourceDigest != r.SourceDigest || p.BinarySHA256 != r.CandidateSHA256 || !reflect.DeepEqual(p.BuildArgv, r.BuildArgv) {
		return fmt.Errorf("candidate provenance disconnected from exact capture")
	}
	for path, hash := range map[string]string{r.CandidatePath: r.CandidateSHA256, r.ClientPath: r.ClientSHA256} {
		if _, err := nativeEvidenceBytes(nativeEvidenceFile{path, hash}); err != nil {
			return err
		}
	}
	if !nativeEvidenceContains(r.BuildArgv, "-buildvcs=false") {
		return fmt.Errorf("candidate build argv lacks explicit VCS metadata boundary")
	}
	info, err := buildinfo.ReadFile(r.CandidatePath)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(info.GoVersion, "go") || r.ClientVersion == "" || len(r.Args) == 0 {
		return fmt.Errorf("actual Go/host version or invocation absent")
	}
	for _, setting := range info.Settings {
		if strings.HasPrefix(setting.Key, "vcs.") {
			return fmt.Errorf("candidate contains misleading nested-worktree VCS setting %s", setting.Key)
		}
	}
	buildInfoPath := filepath.Join(root, "candidate-build-info.txt")
	if _, err := nativeEvidenceBytes(nativeEvidenceFile{buildInfoPath, r.Artifacts[buildInfoPath]}); err != nil {
		return err
	}
	return nil
}

func nativeEvidenceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func nativeEvidenceMatrixEvents(raw []byte, expected []string) error {
	const pkg = "github.com/calcosmic/Aether/cmd"
	const top = "TestCodexNativeWorkerFreshHost"
	const prefix = top + "/"
	started, finished := map[string]int{}, map[string]int{}
	packageStarted, packageFinished := false, false
	if err := nativeEvidenceGoEvents(raw, func(e nativeEvidenceGoEvent) error {
		if e.Package != pkg {
			return fmt.Errorf("matrix contains unexpected package %s", e.Package)
		}
		if e.Action == "output" && (strings.Contains(e.Output, "WARNING: DATA RACE") || nativeEvidenceFatalPackageOutput(e)) {
			return fmt.Errorf("matrix fatal/race diagnostics present")
		}
		if packageFinished {
			return fmt.Errorf("matrix event after outer package terminal")
		}
		if e.Action == "start" {
			if packageStarted {
				return fmt.Errorf("duplicate matrix package start")
			}
			packageStarted = true
			return nil
		}
		if !packageStarted {
			return fmt.Errorf("matrix event before package start")
		}
		if e.Action == "fail" || e.Action == "skip" {
			return fmt.Errorf("mandatory matrix ended %s: %s", e.Action, e.Test)
		}
		if e.Test == "" {
			if e.Action == "pass" {
				if finished[top] != 1 {
					return fmt.Errorf("matrix outer test not complete")
				}
				for _, name := range expected {
					if finished[prefix+name] != 1 {
						return fmt.Errorf("matrix case unfinished: %s", name)
					}
				}
				packageFinished = true
			}
			return nil
		}
		if e.Test != top && (!strings.HasPrefix(e.Test, prefix) || !nativeEvidenceContains(expected, strings.TrimPrefix(e.Test, prefix))) {
			return fmt.Errorf("unexpected matrix test %s", e.Test)
		}
		switch e.Action {
		case "run":
			if started[e.Test] != 0 || finished[e.Test] != 0 || finished[top] != 0 {
				return fmt.Errorf("duplicate/out-of-order matrix run")
			}
			if e.Test != top && started[top] != 1 {
				return fmt.Errorf("matrix child before parent run")
			}
			started[e.Test]++
		case "pass":
			if started[e.Test] != 1 || finished[e.Test] != 0 {
				return fmt.Errorf("matrix terminal without unique prior run")
			}
			if e.Test == top {
				for _, name := range expected {
					if finished[prefix+name] != 1 {
						return fmt.Errorf("matrix outer test precedes unfinished case")
					}
				}
			}
			finished[e.Test]++
		}
		return nil
	}); err != nil {
		return err
	}
	if !packageStarted || !packageFinished || started[top] != 1 || finished[top] != 1 {
		return fmt.Errorf("matrix outer package/test completion absent")
	}
	for _, name := range expected {
		if started[prefix+name] != 1 || finished[prefix+name] != 1 {
			return fmt.Errorf("matrix %s missing unique run/pass", name)
		}
	}
	return nil
}

func nativeEvidenceFatalPackageOutput(e nativeEvidenceGoEvent) bool {
	if e.Action != "output" || e.Test != "" {
		return false
	}
	for _, line := range strings.Split(e.Output, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"panic:", "fatal error:", "runtime: ", "signal: ", "test timed out after ", "go: error", "# "} {
			if strings.HasPrefix(line, prefix) {
				return true
			}
		}
		if strings.Contains(line, "[build failed]") {
			return true
		}
	}
	return false
}

func nativeEvidenceReplayManifest(q nativeEvidenceQualification) error {
	raw, err := nativeEvidenceBytes(nativeEvidenceFile{q.ReplayManifest, q.ReplayManifestSHA256})
	if err != nil {
		return err
	}
	var m struct {
		Validator struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Source string `json:"source"`
		} `json:"validator"`
		Runs []struct {
			Scenario  string `json:"scenario"`
			ExitCode  *int   `json:"exit_code"`
			Log       string `json:"log"`
			LogSHA256 string `json:"log_sha256"`
		} `json:"runs"`
		OriginalsUnchanged bool `json:"all_original_receipts_unchanged"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.Validator.Source != q.TestedSource || m.Validator.Path != q.ValidatorIdentity.Path ||
		m.Validator.SHA256 != q.ValidatorIdentity.SHA256 || !m.OriginalsUnchanged || len(m.Runs) != len(q.Cases) {
		return fmt.Errorf("raw replay manifest identity/completeness differs")
	}
	seen := map[string]bool{}
	for _, run := range m.Runs {
		if _, ok := q.Cases[run.Scenario]; !ok || seen[run.Scenario] || run.ExitCode == nil || *run.ExitCode != 0 {
			return fmt.Errorf("replay run absent/duplicate/failed: %s", run.Scenario)
		}
		seen[run.Scenario] = true
		if _, err := nativeEvidenceBytes(nativeEvidenceFile{run.Log, run.LogSHA256}); err != nil {
			return err
		}
	}
	return nil
}

func nativeEvidenceMutationManifest(root string, ref nativeEvidenceFile) error {
	raw, err := nativeEvidenceBytes(ref)
	if err != nil {
		return err
	}
	var controls []struct {
		Mutation       string `json:"mutation"`
		File           string `json:"file"`
		OriginalSHA256 string `json:"original_sha256"`
		MutatedSHA256  string `json:"mutated_sha256"`
		Exit           *int   `json:"exit"`
		Log            string `json:"log"`
		SHA256         string `json:"sha256"`
	}
	if err := json.Unmarshal(raw, &controls); err != nil {
		return err
	}
	expected := map[string]string{
		"platform-order": "TestCodexNativeBuildGuidePlatformIsolation",
		"patch-evidence": "TestCodexNativeLiteralPatchBoundary",
		"batch-evidence": "TestCodexNativeReadOnlyBatchEventLinkage",
	}
	seen := map[string]bool{}
	for _, c := range controls {
		testName, ok := expected[c.Mutation]
		if !ok || seen[c.Mutation] || c.Exit == nil || *c.Exit != 1 || c.OriginalSHA256 == c.MutatedSHA256 {
			return fmt.Errorf("invalid source disconnect control %s", c.Mutation)
		}
		seen[c.Mutation] = true
		if _, err := nativeEvidenceBytes(nativeEvidenceFile{filepath.Join(root, c.File), c.OriginalSHA256}); err != nil {
			return err
		}
		log, err := nativeEvidenceBytes(nativeEvidenceFile{c.Log, c.SHA256})
		if err != nil {
			return err
		}
		if !strings.Contains(string(log), "--- FAIL: "+testName) || strings.Contains(string(log), "[build failed]") {
			return fmt.Errorf("disconnect %s lacks runnable causal test failure", c.Mutation)
		}
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("required source disconnect controls absent")
	}
	return nil
}

// Completed regression accounting is required only in strict mode. No full-regression
// receipt is read by ordinary mode, avoiding a circular demand during full runs.
type nativeEvidenceRegression struct {
	TestCorpusDigest    string                                 `json:"test_corpus_digest"`
	SchemaVersion       string                                 `json:"schema_version"`
	QualificationSHA256 string                                 `json:"qualification_sha256"`
	ProductionDigest    string                                 `json:"production_digest"`
	HarnessSHA256       string                                 `json:"harness_sha256"`
	Runs                map[string]nativeEvidenceRegressionRun `json:"runs"`
}
type nativeEvidenceRegressionRun struct {
	Argv      []string           `json:"argv"`
	ExitCode  *int               `json:"exit_code"`
	Discovery nativeEvidenceFile `json:"discovery"`
	RawJSON   nativeEvidenceFile `json:"raw_json"`
	// Each exception requires a separately reviewed exact diagnostic comparison.
	// It cannot authorize a required native, schema, or catalog failure.
	InheritedFailures map[string]nativeEvidenceInheritedFailure `json:"inherited_failures"`
}
type nativeEvidenceInheritedFailure struct {
	DiagnosticSHA256 string             `json:"diagnostic_sha256"`
	Baseline         nativeEvidenceFile `json:"baseline"`
	Comparison       nativeEvidenceFile `json:"comparison"`
}
type nativeEvidenceGoEvent struct {
	Action  string
	Package string
	Test    string
	Output  string
}

func nativeEvidenceGoEvents(raw []byte, visit func(nativeEvidenceGoEvent) error) error {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 65536), 32<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var e nativeEvidenceGoEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return fmt.Errorf("malformed/truncated Go JSON: %w", err)
		}
		if err := visit(e); err != nil {
			return err
		}
	}
	return scanner.Err()
}
func nativeEvidenceRegressionCheck(t *testing.T, root, path, qualification, production, harness string) error {
	var r nativeEvidenceRegression
	if err := nativeEvidenceReadJSON(path, &r); err != nil {
		return err
	}
	corpus, err := nativeEvidenceCurrentTestCorpus(root)
	if err != nil {
		return err
	}
	if err := nativeEvidenceRegressionIdentity(r, qualification, production, harness, corpus); err != nil {
		return err
	}
	// Independent discovery is run once, never supplied by the receipt. -list
	// compiles/lists but does not execute any tests or full-suite controller.
	discovery, err := nativeEvidenceCurrentDiscovery(root)
	if err != nil {
		return err
	}
	for _, name := range []string{"focused_normal", "focused_race", "normal", "race"} {
		run, ok := r.Runs[name]
		if !ok {
			return fmt.Errorf("required regression run absent: %s", name)
		}
		if err := nativeEvidenceRegressionRunCheck(name, run, discovery); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func nativeEvidenceRegressionIdentity(r nativeEvidenceRegression, qualification, production, harness, corpus string) error {
	if r.SchemaVersion != "aether-native-regression/v1" || r.QualificationSHA256 != qualification ||
		r.ProductionDigest != production || r.HarnessSHA256 != harness {
		return fmt.Errorf("completed regression receipt identity differs")
	}
	if corpus == "" || r.TestCorpusDigest != corpus {
		return fmt.Errorf("regression test corpus changed")
	}
	return nil
}

// Source-owned affected boundaries: native lifecycle, public pause/resume, scoped
// decisions/flags/context, finalization, semantic repair, installed guides and schema.
const nativeEvidenceFocusedSelection = "^(TestCodexNative|TestPauseResume199|TestPauseWrapperContract199|TestResumeWrapperContract199|Example_pauseResume199Contract$|TestResumeShows|TestResumeView|TestResumeDetail|TestDriftNote|TestColonyPrime|TestClarifiedIntent|TestDiscuss|TestDetectDecisionConflicts|TestPendingDecision|TestDecisionAnswer|TestRecordDecisionAnswer|TestGenericPendingDecisionResolver|TestFlag|TestSemanticDependencies|TestCrossPhaseDependencies|TestRepairArtifact|TestPlanRepairArtifact|TestAcceptedSemanticDependencies|TestCompletionPacketSchemaMatchesStructs$|TestAuditCatalogGolden$|TestCommandGuide|TestCodexLifecycle|TestCodexAntSkill(Inventory|InstallTracer|GuidesPreserveOtherPlatforms|GuideSupport)|TestCodexGenerated|TestSyncCodex|TestBuildManifestCarriesContext|TestBuildWorkerBrief|TestComposeBuildManifestBrief|TestStageBuildAttemptCompletion|TestBuildAttempt|TestNativePartialCredit|TestPartialBuildDoesNotShow|TestBuildStartCallers200$|TestStartupLifecycleCardsComeFromTheResolver$|TestInternalBuildAdapterStagesCompletionWhenAllWorkersAreTerminal$|TestRepair(BadManifest|DirtyWorktree)_DestructiveNeedsConfirmation$|TestBriefPathReferencedAcrossAllFourSurfaces$|TestBuildCommandYAMLCoherentJobsParity$|TestLifecycleFlatMirrorsMatchCanonical$|TestLifecycleGuidesDocumentApprovedTempCompletionContract$|TestLifecycleOrchestratorsMentionReadCacheLoopHandling$|TestNextActionNeverHardcoded$|TestNoRegisteredSubcommandIsUnreferenced$|TestPlanAndColonizeWrappersAreByteIdentical$|TestPlanWrapperCardsParity$|TestPlanWrapperStageSkeleton$|TestTerritoryWrapperAuthority199$|TestNativeReachability|TestBuildFinalizeNextAdviceMatchesResolvedAnswer$|TestExternalGroupedPartialPersistsExactTaskState$)"

func nativeEvidenceRunContract(name string, argv []string) error {
	if len(argv) < 2 || argv[0] != "go" || argv[1] != "test" {
		return fmt.Errorf("exact Go test invocation required")
	}
	focused := strings.HasPrefix(name, "focused_")
	race := strings.HasSuffix(name, "race")
	seen := map[string]bool{}
	for i := 2; i < len(argv); i++ {
		arg := argv[i]
		switch arg {
		case "./...", "-json", "-count=1", "-race":
		case "./cmd":
			if !focused {
				return fmt.Errorf("full invocation cannot narrow package scope")
			}
		case "-timeout":
			if i+1 >= len(argv) {
				return fmt.Errorf("timeout value absent")
			}
			i++
			if argv[i] != "90m" {
				return fmt.Errorf("evidence invocation timeout must be 90m")
			}
			arg = "-timeout=90m"
		case "-timeout=90m":
		case "-run":
			if !focused || i+1 >= len(argv) || argv[i+1] != nativeEvidenceFocusedSelection {
				return fmt.Errorf("selector not allowed by source-owned run contract")
			}
			i++
		default:
			return fmt.Errorf("argument outside exact run contract: %s", arg)
		}
		if seen[arg] {
			return fmt.Errorf("duplicate invocation argument: %s", arg)
		}
		seen[arg] = true
	}
	if !seen["-json"] || !seen["-count=1"] || !seen["-timeout=90m"] || seen["-race"] != race {
		return fmt.Errorf("required exact invocation flags absent")
	}
	if focused {
		if !seen["./cmd"] || seen["./..."] || !seen["-run"] {
			return fmt.Errorf("focused package/selector contract differs")
		}
	} else if !seen["./..."] || seen["./cmd"] || seen["-run"] {
		return fmt.Errorf("full invocation must execute unselected ./...")
	}
	return nil
}

func nativeEvidenceExpectedDiscovery(name string, all map[string][]string) map[string][]string {
	selected := map[string][]string{}
	pattern := regexp.MustCompile(nativeEvidenceFocusedSelection)
	for pkg, names := range all {
		if strings.HasPrefix(name, "focused_") && pkg != "github.com/calcosmic/Aether/cmd" {
			continue
		}
		selected[pkg] = []string{}
		for _, test := range names {
			if strings.HasPrefix(name, "focused_") && !pattern.MatchString(test) {
				continue
			}
			selected[pkg] = append(selected[pkg], test)
		}
		sort.Strings(selected[pkg])
	}
	return selected
}

func nativeEvidenceCurrentDiscovery(root string) (map[string][]string, error) {
	if strings.TrimSpace(os.Getenv("GOFLAGS")) != "" {
		return nil, fmt.Errorf("independent discovery requires explicit empty GOFLAGS; no ambient selectors accepted")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	list := exec.CommandContext(ctx, "go", "list", "./...")
	list.Dir = root
	raw, err := list.Output()
	if err != nil {
		return nil, fmt.Errorf("independent Go package discovery: %w", err)
	}
	all := map[string][]string{}
	for _, pkg := range strings.Fields(string(raw)) {
		if _, exists := all[pkg]; exists {
			return nil, fmt.Errorf("duplicate discovered package %s", pkg)
		}
		all[pkg] = []string{}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("independent source package discovery empty")
	}
	cmd := exec.CommandContext(ctx, "go", "test", "./...", "-list", "^(Test|Example|Fuzz)", "-json", "-count=1", "-timeout", "90s")
	cmd.Dir = root
	raw, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("independent list-only test discovery: %w", err)
	}
	ended := map[string]bool{}
	namePattern := regexp.MustCompile("^(Test|Example|Fuzz)[^[:space:]]*$")
	if err := nativeEvidenceGoEvents(raw, func(e nativeEvidenceGoEvent) error {
		if _, ok := all[e.Package]; !ok {
			return fmt.Errorf("list-only stream names undiscovered package %s", e.Package)
		}
		if e.Test != "" || e.Action == "run" || e.Action == "fail" {
			return fmt.Errorf("list-only discovery executed or failed a test")
		}
		if e.Action == "pass" || e.Action == "skip" {
			if ended[e.Package] {
				return fmt.Errorf("duplicate list-only package terminal")
			}
			ended[e.Package] = true
		}
		if e.Action == "output" {
			for _, line := range strings.Split(e.Output, "\n") {
				if namePattern.MatchString(line) {
					all[e.Package] = append(all[e.Package], line)
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	for pkg := range all {
		if !ended[pkg] {
			return nil, fmt.Errorf("list-only package unfinished: %s", pkg)
		}
		sort.Strings(all[pkg])
		for i := 1; i < len(all[pkg]); i++ {
			if all[pkg][i] == all[pkg][i-1] {
				return nil, fmt.Errorf("duplicate list-only case")
			}
		}
	}
	return all, nil
}

func nativeEvidenceCurrentTestCorpus(root string) (string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", "*_test.go")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		return "", err
	}
	files := map[string]string{}
	for _, path := range strings.Split(string(raw), "\x00") {
		if path == "" || strings.HasPrefix(path, ".planning/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return "", err
		}
		files[path] = lifecycleDigest(data)
	}
	if len(files) == 0 {
		return "", fmt.Errorf("current test corpus absent")
	}
	return nativeEvidenceTestCorpusDigest(files), nil
}
func nativeEvidenceTestCorpusDigest(files map[string]string) string {
	names := make([]string, 0, len(files))
	for path := range files {
		names = append(names, path)
	}
	sort.Strings(names)
	var digest strings.Builder
	for _, path := range names {
		fmt.Fprintf(&digest, "%s\x00%s\n", path, files[path])
	}
	return lifecycleDigest([]byte(digest.String()))
}

func nativeEvidenceRegressionRunCheck(name string, run nativeEvidenceRegressionRun, independentDiscovery map[string][]string) error {
	if err := nativeEvidenceRunContract(name, run.Argv); err != nil {
		return err
	}
	race := strings.HasSuffix(name, "race")
	if run.ExitCode == nil || (*run.ExitCode != 0 && *run.ExitCode != 1) || nativeEvidenceContains(run.Argv, "-race") != race ||
		!nativeEvidenceContains(run.Argv, "-json") || !nativeEvidenceContains(run.Argv, "-count=1") {
		return fmt.Errorf("missing/invalid exact invocation or captured exit")
	}
	if (name == "normal" || name == "race") && (!nativeEvidenceContains(run.Argv, "./...") || !nativeEvidenceContains(run.Argv, "90m") && !nativeEvidenceContains(run.Argv, "-timeout=90m")) {
		return fmt.Errorf("full suite scope or timeout differs")
	}
	discoveryRaw, err := nativeEvidenceBytes(run.Discovery)
	if err != nil {
		return err
	}
	var discovery map[string][]string // import path -> exact selected top-level Test and Example names
	if err := json.Unmarshal(discoveryRaw, &discovery); err != nil {
		return err
	}
	if len(discovery) == 0 {
		return fmt.Errorf("test discovery absent")
	}
	for pkg := range discovery {
		sort.Strings(discovery[pkg])
	}
	if !reflect.DeepEqual(discovery, nativeEvidenceExpectedDiscovery(name, independentDiscovery)) {
		return fmt.Errorf("receipt discovery differs from independently derived source selection")
	}
	wanted := map[string]bool{}
	for pkg, tests := range discovery {
		for _, test := range tests {
			key := pkg + "/" + test
			if wanted[key] || strings.Contains(test, "/") || (!strings.HasPrefix(test, "Test") && !strings.HasPrefix(test, "Example") && !strings.HasPrefix(test, "Fuzz")) {
				return fmt.Errorf("invalid/duplicate top-level discovery: %s", key)
			}
			wanted[key] = true
		}
	}
	if strings.HasPrefix(name, "focused_") && len(wanted) == 0 {
		return fmt.Errorf("focused source selection contains no executable tests")
	}
	if strings.HasPrefix(name, "focused_") && len(run.InheritedFailures) != 0 {
		return fmt.Errorf("focused required checks cannot inherit broad failures")
	}
	raw, err := nativeEvidenceBytes(run.RawJSON)
	if err != nil {
		return err
	}
	started, ended, packages, output := map[string]int{}, map[string]string{}, map[string]string{}, map[string]string{}
	casePackages := map[string]string{}
	diagnosticEcho := map[string]bool{}
	packageStarts := map[string]int{}
	if err := nativeEvidenceGoEvents(raw, func(e nativeEvidenceGoEvent) error {
		key := e.Package + "/" + e.Test
		if nativeEvidenceFatalPackageOutput(e) {
			return fmt.Errorf("unexpected package fatal diagnostic")
		}
		if e.Test != "" && packages[e.Package] != "" {
			return fmt.Errorf("real test event after package terminal")
		}
		if e.Action == "start" {
			packageStarts[e.Package]++
			if packageStarts[e.Package] != 1 {
				return fmt.Errorf("duplicate package start")
			}
		}
		// cmd's TestMain writes each lane once, then reprints failed lane
		// output only after this exact package-level marker. test2json parses
		// that diagnostic echo as test events. Never deduplicate real events
		// before the marker, and never credit a case appearing only in echo.
		// Keep the raw JSON intact and retain the outer package terminal/exit.
		if e.Action == "output" && strings.Contains(e.Output, "WARNING: DATA RACE") {
			return fmt.Errorf("race finding present")
		}
		if e.Action == "output" && e.Test == "" && e.Package == "github.com/calcosmic/Aether/cmd" &&
			strings.HasPrefix(e.Output, "full-suite controller failed:") {
			diagnosticEcho[e.Package] = true
			return nil
		}
		if diagnosticEcho[e.Package] && e.Test != "" {
			return nil
		}
		if _, ok := discovery[e.Package]; !ok {
			return fmt.Errorf("executed package absent from discovery: %s", e.Package)
		}
		if e.Test != "" && !wanted[e.Package+"/"+strings.SplitN(e.Test, "/", 2)[0]] {
			return fmt.Errorf("executed test absent from discovery: %s", key)
		}
		if e.Action == "output" {
			output[key] += e.Output
			if strings.Contains(e.Output, "WARNING: DATA RACE") {
				return fmt.Errorf("race finding present")
			}
		}
		if e.Test == "" {
			if e.Action == "pass" || e.Action == "fail" || e.Action == "skip" {
				if packages[e.Package] != "" {
					return fmt.Errorf("duplicate outer package terminal: %s", e.Package)
				}
				if diagnosticEcho[e.Package] && e.Action != "fail" {
					return fmt.Errorf("controller failure echo lacks failing outer package: %s", e.Package)
				}
				packages[e.Package] = e.Action
			}
			return nil
		}
		switch e.Action {
		case "run":
			if packageStarts[e.Package] != 1 || started[key] != 0 || ended[key] != "" {
				return fmt.Errorf("duplicate real run or run before package start: %s", key)
			}
			started[key]++
			casePackages[key] = e.Package
		case "pass", "fail", "skip":
			if started[key] != 1 {
				return fmt.Errorf("terminal without one prior run: %s", key)
			}
			if ended[key] != "" {
				return fmt.Errorf("duplicate terminal event: %s", key)
			}
			ended[key] = e.Action
		}
		return nil
	}); err != nil {
		return err
	}
	for pkg, tests := range discovery {
		if packages[pkg] == "" || packageStarts[pkg] != 1 {
			return fmt.Errorf("unfinished/missing package %s", pkg)
		}
		for _, test := range tests {
			key := pkg + "/" + test
			if started[key] != 1 || ended[key] == "" {
				return fmt.Errorf("discovered test missing/unfinished: %s", key)
			}
		}
	}
	failures := 0
	used := map[string]bool{}
	for key := range ended {
		if started[key] != 1 {
			return fmt.Errorf("terminal missing matching execution: %s", key)
		}
	}
	for key, count := range started {
		if count != 1 || ended[key] == "" {
			return fmt.Errorf("unfinished/duplicate executed case %s", key)
		}
		if ended[key] != "fail" {
			continue
		} // Explicit skips remain visible in raw accounting, never native proof.
		failures++
		waiver, ok := run.InheritedFailures[key]
		if !ok || !nativeEvidenceHistoricalFailure(key) {
			return fmt.Errorf("new or required failure cannot be inherited: %s", key)
		}
		if waiver.DiagnosticSHA256 != lifecycleDigest([]byte(output[key])) {
			return fmt.Errorf("diagnostics changed for %s", key)
		}
		if _, err := nativeEvidenceBytes(waiver.Baseline); err != nil {
			return err
		}
		comparisonRaw, err := nativeEvidenceBytes(waiver.Comparison)
		if err != nil {
			return err
		}
		var comparison struct {
			Test                    string `json:"test"`
			Verdict                 string `json:"verdict"`
			Reviewer                string `json:"reviewer"`
			CurrentDiagnosticSHA256 string `json:"current_diagnostic_sha256"`
			BaselineSHA256          string `json:"baseline_sha256"`
		}
		if err := json.Unmarshal(comparisonRaw, &comparison); err != nil {
			return err
		}
		if comparison.Test != key || comparison.Verdict != "same_substantive_historical_failure" || comparison.Reviewer == "" ||
			comparison.CurrentDiagnosticSHA256 != waiver.DiagnosticSHA256 || comparison.BaselineSHA256 != waiver.Baseline.SHA256 {
			return fmt.Errorf("historical diagnostic comparison incomplete for %s", key)
		}
		used[key] = true
	}
	if len(used) != len(run.InheritedFailures) {
		return fmt.Errorf("stale/unused historical exceptions present")
	}
	packageFailed := false
	for pkg, result := range packages {
		for key, terminal := range ended {
			if casePackages[key] == pkg && terminal == "fail" && result != "fail" {
				return fmt.Errorf("package terminal contradicts failed case")
			}
		}
		if result != "fail" {
			continue
		}
		packageFailed = true
		found := false
		for key, result := range ended {
			if casePackages[key] == pkg && result == "fail" {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("package failure outside accounted test cases: %s", pkg)
		}
	}
	if (*run.ExitCode == 0) != (!packageFailed && failures == 0) {
		return fmt.Errorf("raw outcomes disagree with captured exit")
	}
	return nil
}

// Names bound the historical comparison; name equality alone never authorizes
// an exception. Each actual diagnostic additionally needs the pinned review above.
func nativeEvidenceHistoricalFailure(key string) bool {
	for _, name := range []string{
		"TestBuildStartLegacyHelpersRetired200",
		"TestCodexBuildPlanOnlySpawnBudgetSeparatesCasteBudgetFromWorkerCount",
		"TestCurrentVocabulary199", "TestFailedCheckSendsExactlyOneBuilderFixAttempt",
		"TestFixAttemptIsCountedSeparately", "TestFixAttemptNeverOverwritesTheFirstResult",
		"TestGoSourceHintsMatchCobraContracts", "TestGoldenBuildVisualOutput",
		"TestGoldenContinueVisualOutput", "TestHumanFacingOutputGoesThroughWriteVisualOutput",
		"TestNoSecondAutomaticFixAttempt", "TestPhase199GateReceipt",
		"TestPlanningAdversarial200", "TestPlanningPublicPaths200", "TestResolveTestCommand_GoProject",
	} {
		if strings.HasSuffix(key, "/"+name) || strings.Contains(key, "/"+name+"/") {
			return true
		}
	}
	return false
}

// Controls are deterministic fixtures, not actual host proof. Each starts by
// passing BOTH the reviewed replay and this wrapper before any mutation.
func TestCodexNativeEvidenceRejects(t *testing.T) {
	for _, mutation := range []string{"missing-child", "changed-digest", "wrong-source", "parent-only", "exit-zero-no-native-events"} {
		t.Run(mutation, func(t *testing.T) {
			r := nativeInventoryReplayFixture(t)
			const source = "1111111111111111111111111111111111111111" // explicitly synthetic identity binding
			production, harness := lifecycleDigest([]byte("deterministic-production")), lifecycleDigest([]byte("deterministic-harness"))
			r.SourceRevision, r.SourceDigest, r.HarnessSHA256 = source, production, harness
			baseline := r
			if err := nativeReplayQualificationReceipt(t, &baseline); err != nil || baseline.Outcome != "passed" {
				t.Fatalf("control fixture does not pass reviewed raw replay before mutation: %v", err)
			}
			baseline = r
			if err := nativeEvidenceReplayRaw(t, &baseline, source, production, harness); err != nil {
				t.Fatalf("control fixture does not pass boundary wrapper before mutation: %v", err)
			}
			root := filepath.Dir(r.FixtureRoot)
			child := filepath.Join(root, "home", ".codex", "sessions", "2026", "09", "17", "child.jsonl")
			parent := filepath.Join(filepath.Dir(child), "parent.jsonl")
			rewrite := func(path string, raw []byte) {
				t.Helper()
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				// Rehash purposeful fixture mutations so event controls reach the
				// causal boundary instead of stopping at an unrelated digest error.
				r.Artifacts[path] = lifecycleDigest(raw)
			}
			switch mutation {
			case "missing-child":
				path := filepath.Join(r.FixtureRoot, ".aether", "data", "build", "phase-1", "attempts", "attempt.json")
				var attempt buildAttemptRecord
				if err := nativeEvidenceReadJSON(path, &attempt); err != nil {
					t.Fatal(err)
				}
				attempt.WorkerRuns[0].Native.ChildID = ""
				raw, err := json.Marshal(attempt)
				if err != nil {
					t.Fatal(err)
				}
				rewrite(path, raw)
			case "changed-digest":
				r.Artifacts[child] = lifecycleDigest([]byte("different captured bytes"))
			case "wrong-source":
				r.SourceRevision = "2222222222222222222222222222222222222222"
			case "parent-only":
				raw, err := os.ReadFile(child)
				if err != nil {
					t.Fatal(err)
				}
				var kept []byte
				for _, line := range bytes.SplitAfter(raw, []byte("\n")) {
					if bytes.Contains(line, []byte("\"FileChange\"")) {
						continue
					}
					kept = append(kept, line...)
				}
				rewrite(child, kept)
				parentRaw, err := os.ReadFile(parent)
				if err != nil {
					t.Fatal(err)
				}
				parentRaw = append(parentRaw, nativeEvidenceEvent(t, "item_completed", "parent", map[string]any{
					"type": "FileChange", "status": "completed", "changes": map[string]any{
						filepath.Join(r.FixtureRoot, "clamp.go"): map[string]any{"type": "update", "unified_diff": "@@ -1,1 +1,1 @@\n-old\n+new\n"},
					},
				})...)
				rewrite(parent, parentRaw)
			case "exit-zero-no-native-events":
				raw, err := os.ReadFile(parent)
				if err != nil {
					t.Fatal(err)
				}
				var kept []byte
				for _, line := range bytes.SplitAfter(raw, []byte("\n")) {
					if bytes.Contains(line, []byte("spawn_agent")) {
						continue
					}
					kept = append(kept, line...)
				}
				rewrite(parent, kept)
				childRaw, err := os.ReadFile(child)
				if err != nil {
					t.Fatal(err)
				}
				rewrite(child, append(bytes.SplitN(childRaw, []byte("\n"), 2)[0], '\n'))
				if r.ExitStatus != 0 {
					t.Fatal("zero-exit negative fixture lost its causal premise")
				}
			}
			err := nativeEvidenceReplayRaw(t, &r, source, production, harness)
			if err == nil {
				t.Fatalf("%s mutation passed", mutation)
			}
			want := map[string]string{
				"missing-child":              "actual native parent/child identity",
				"changed-digest":             "evidence digest changed",
				"wrong-source":               "capture source binding changed",
				"parent-only":                "raw native child edit/check evidence incomplete",
				"exit-zero-no-native-events": "native spawn/session binding incomplete",
			}[mutation]
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("%s failed at an unrelated boundary: got %v; want %q", mutation, err, want)
			}
			if mutation == "parent-only" && (r.ChildEditObserved || !r.ParentSubstitution) {
				t.Fatalf("parent-only fixture did not reach attribution boundary: edit=%v substitution=%v", r.ChildEditObserved, r.ParentSubstitution)
			}
		})
	}
}

// Synthetic Go JSON is a parser control, not a new full-suite receipt. Every
// negative starts from the same passing Example + delimited controller-echo run.
func TestCodexNativeEvidenceParser(t *testing.T) {
	for _, mutation := range []string{"example-and-echo", "duplicate-before-marker", "missing-case-only-in-echo", "test-level-marker", "missing-outer-terminal", "race-in-echo", "orphan-failure", "terminal-before-run", "outer-panic", "after-package", "package-success-with-failure"} {
		t.Run(mutation, func(t *testing.T) {
			run, events := nativeEvidenceParserFixture(t)
			if err := nativeEvidenceRegressionRunCheck("normal", run, map[string][]string{"github.com/calcosmic/Aether/cmd": {"Example_pauseResume199Contract", "TestGoldenBuildVisualOutput"}}); err != nil {
				t.Fatalf("parser control fixture failed before mutation: %v", err)
			}
			marker := -1
			for i, e := range events {
				if e.Test == "" && strings.HasPrefix(e.Output, "full-suite controller failed:") {
					marker = i
					break
				}
			}
			if marker < 0 {
				t.Fatal("fixture marker missing")
			}
			switch mutation {
			case "example-and-echo":
				return // Example is counted once; the explicit diagnostic echo is not execution.
			case "duplicate-before-marker":
				repeated := []nativeEvidenceGoEvent{
					{Action: "run", Package: events[0].Package, Test: "Example_pauseResume199Contract"},
					{Action: "pass", Package: events[0].Package, Test: "Example_pauseResume199Contract"},
				}
				events = append(append(append([]nativeEvidenceGoEvent{}, events[:marker]...), repeated...), events[marker:]...)
			case "missing-case-only-in-echo":
				var kept []nativeEvidenceGoEvent
				for i, e := range events {
					if i < marker && e.Test == "Example_pauseResume199Contract" {
						continue
					}
					kept = append(kept, e)
				}
				events = kept
			case "test-level-marker":
				events[marker].Test = "TestGoldenBuildVisualOutput"
			case "missing-outer-terminal":
				events = events[:len(events)-1]
			case "orphan-failure", "terminal-before-run":
				orphan := []nativeEvidenceGoEvent{{Action: "fail", Package: events[0].Package, Test: "TestGoldenBuildVisualOutput/orphan"}}
				if mutation == "terminal-before-run" {
					orphan = append(orphan, nativeEvidenceGoEvent{Action: "run", Package: events[0].Package, Test: "TestGoldenBuildVisualOutput/orphan"})
				}
				events = append(append(append([]nativeEvidenceGoEvent{}, events[:marker]...), orphan...), events[marker:]...)
			case "outer-panic":
				fatal := nativeEvidenceGoEvent{Action: "output", Package: events[0].Package, Output: "panic: new unaccounted failure\n"}
				events = append(append(append([]nativeEvidenceGoEvent{}, events[:marker]...), fatal), events[marker:]...)
			case "after-package":
				events = append(events, nativeEvidenceGoEvent{Action: "run", Package: events[0].Package, Test: "Example_pauseResume199Contract"})
			case "package-success-with-failure":
				events = append(events[:marker], nativeEvidenceGoEvent{Action: "pass", Package: events[0].Package})
			case "race-in-echo":
				race := nativeEvidenceGoEvent{Action: "output", Package: events[0].Package, Test: "TestGoldenBuildVisualOutput", Output: "WARNING: DATA RACE\n"}
				events = append(append(append([]nativeEvidenceGoEvent{}, events[:marker+1]...), race), events[marker+1:]...)
			}
			run.RawJSON = nativeEvidenceParserWriteEvents(t, filepath.Join(t.TempDir(), "mutated.jsonl"), events)
			err := nativeEvidenceRegressionRunCheck("normal", run, map[string][]string{"github.com/calcosmic/Aether/cmd": {"Example_pauseResume199Contract", "TestGoldenBuildVisualOutput"}})
			want := map[string]string{
				"duplicate-before-marker":      "duplicate real run",
				"missing-case-only-in-echo":    "discovered test missing/unfinished:",
				"test-level-marker":            "duplicate real run",
				"missing-outer-terminal":       "unfinished/missing package",
				"race-in-echo":                 "race finding present",
				"orphan-failure":               "terminal without one prior run",
				"terminal-before-run":          "terminal without one prior run",
				"outer-panic":                  "unexpected package fatal diagnostic",
				"after-package":                "real test event after package terminal",
				"package-success-with-failure": "package terminal contradicts failed case",
			}[mutation]
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s did not fail at intended parser boundary: got %v; want %q", mutation, err, want)
			}
		})
	}
}

func TestCodexNativeEvidencePackageOwnership(t *testing.T) {
	const rootPackage = "github.com/calcosmic/Aether"
	const commandPackage = rootPackage + "/cmd"
	discovery := map[string][]string{
		rootPackage:    {},
		commandPackage: {"Example_pauseResume199Contract", "TestGoldenBuildVisualOutput"},
	}
	for _, mutation := range []string{"distinct-package-outcomes", "unexplained-root-failure", "command-success-with-failure"} {
		t.Run(mutation, func(t *testing.T) {
			run, events := nativeEvidenceParserFixture(t)
			raw, err := json.Marshal(discovery)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "discovery.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			run.Discovery = nativeEvidenceFile{path, lifecycleDigest(raw)}
			events = append([]nativeEvidenceGoEvent{{Action: "start", Package: rootPackage}, {Action: "skip", Package: rootPackage}}, events...)
			run.RawJSON = nativeEvidenceParserWriteEvents(t, filepath.Join(t.TempDir(), "baseline.jsonl"), events)
			if err := nativeEvidenceRegressionRunCheck("normal", run, discovery); err != nil {
				t.Fatalf("root skip and separate reviewed cmd failure must validate before mutation: %v", err)
			}
			switch mutation {
			case "distinct-package-outcomes":
				return
			case "unexplained-root-failure":
				events[1].Action = "fail"
			case "command-success-with-failure":
				// Remove diagnostic echo so the actual test/package mismatch
				// is the boundary under test, not the echo's failing-package rule.
				for i, event := range events {
					if strings.HasPrefix(event.Output, "full-suite controller failed:") {
						events = append(events[:i], nativeEvidenceGoEvent{Action: "pass", Package: commandPackage})
						break
					}
				}
			}
			run.RawJSON = nativeEvidenceParserWriteEvents(t, filepath.Join(t.TempDir(), "mutated.jsonl"), events)
			want := "package failure outside accounted test cases"
			if mutation == "command-success-with-failure" {
				want = "package terminal contradicts failed case"
			}
			if err := nativeEvidenceRegressionRunCheck("normal", run, discovery); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s did not reject at exact package boundary: %v; want %q", mutation, err, want)
			}
		})
	}
}

func nativeEvidenceParserFixture(t *testing.T) (nativeEvidenceRegressionRun, []nativeEvidenceGoEvent) {
	t.Helper()
	const pkg = "github.com/calcosmic/Aether/cmd"
	const example = "Example_pauseResume199Contract"
	const historical = "TestGoldenBuildVisualOutput"
	const diagnostic = "synthetic historical diagnostic; no actual baseline judgment\n"
	root := t.TempDir()
	write := func(name string, raw []byte) nativeEvidenceFile {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		return nativeEvidenceFile{path, lifecycleDigest(raw)}
	}
	discovery, err := json.Marshal(map[string][]string{pkg: {example, historical}})
	if err != nil {
		t.Fatal(err)
	}
	baseline := write("synthetic-baseline.txt", []byte(diagnostic))
	diagnosticHash := lifecycleDigest([]byte(diagnostic))
	comparison, err := json.Marshal(map[string]string{
		"test": pkg + "/" + historical, "verdict": "same_substantive_historical_failure",
		"reviewer": "deterministic-parser-fixture-only", "current_diagnostic_sha256": diagnosticHash,
		"baseline_sha256": baseline.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	exit := 1
	run := nativeEvidenceRegressionRun{
		Argv:     []string{"go", "test", "./...", "-count=1", "-timeout", "90m", "-json"},
		ExitCode: &exit, Discovery: write("discovery.json", discovery),
		InheritedFailures: map[string]nativeEvidenceInheritedFailure{
			pkg + "/" + historical: {DiagnosticSHA256: diagnosticHash, Baseline: baseline, Comparison: write("synthetic-comparison.json", comparison)},
		},
	}
	events := []nativeEvidenceGoEvent{
		{Action: "start", Package: pkg},
		{Action: "run", Package: pkg, Test: example},
		{Action: "pass", Package: pkg, Test: example},
		{Action: "run", Package: pkg, Test: historical},
		{Action: "output", Package: pkg, Test: historical, Output: diagnostic},
		{Action: "fail", Package: pkg, Test: historical},
		{Action: "output", Package: pkg, Output: "full-suite controller failed: lane serial: exit status 1\n"},
		{Action: "run", Package: pkg, Test: example},
		{Action: "pass", Package: pkg, Test: example},
		{Action: "run", Package: pkg, Test: historical},
		{Action: "output", Package: pkg, Test: historical, Output: "diagnostic echo; must not alter original diagnostic hash\n"},
		{Action: "fail", Package: pkg, Test: historical},
		{Action: "fail", Package: pkg},
	}
	run.RawJSON = nativeEvidenceParserWriteEvents(t, filepath.Join(root, "raw.jsonl"), events)
	return run, events
}

func nativeEvidenceParserWriteEvents(t *testing.T, path string, events []nativeEvidenceGoEvent) nativeEvidenceFile {
	t.Helper()
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return nativeEvidenceFile{path, lifecycleDigest(buf.Bytes())}
}

func TestCodexNativeEvidenceDiscovery(t *testing.T) {
	const pkg = "github.com/calcosmic/Aether/cmd"
	authority := map[string][]string{pkg: {"Example_pauseResume199Contract", "TestGoldenBuildVisualOutput"}}
	for _, mutation := range []string{"full-selector", "full-skip", "self-narrowed-discovery", "focused-empty"} {
		t.Run(mutation, func(t *testing.T) {
			run, events := nativeEvidenceParserFixture(t)
			name := "normal"
			if mutation == "focused-empty" {
				name = "focused_normal"
				run.Argv = []string{"go", "test", "./cmd", "-run", nativeEvidenceFocusedSelection, "-count=1", "-timeout=90m", "-json"}
				exit := 0
				run.ExitCode, run.InheritedFailures = &exit, nil
				var selected []nativeEvidenceGoEvent
				for _, e := range events[:6] {
					if e.Test == "" || e.Test == "Example_pauseResume199Contract" {
						selected = append(selected, e)
					}
				}
				events = append(selected, nativeEvidenceGoEvent{Action: "pass", Package: pkg})
				run.RawJSON = nativeEvidenceParserWriteEvents(t, filepath.Join(t.TempDir(), "focused-valid.jsonl"), events)
				discovery := map[string][]string{pkg: {"Example_pauseResume199Contract"}}
				raw, _ := json.Marshal(discovery)
				path := filepath.Join(t.TempDir(), "focused-discovery.json")
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				run.Discovery = nativeEvidenceFile{path, lifecycleDigest(raw)}
			}
			if err := nativeEvidenceRegressionRunCheck(name, run, authority); err != nil {
				t.Fatalf("valid scope fixture: %v", err)
			}
			want := "receipt discovery differs"
			switch mutation {
			case "full-selector":
				run.Argv = append(run.Argv, "-run", "^TestGoldenBuildVisualOutput$")
				want = "selector not allowed"
			case "full-skip":
				run.Argv = append(run.Argv, "-skip=Example")
				want = "argument outside exact run contract"
			case "self-narrowed-discovery", "focused-empty":
				names := []string{"TestGoldenBuildVisualOutput"}
				if mutation == "focused-empty" {
					names = []string{}
				}
				raw, _ := json.Marshal(map[string][]string{pkg: names})
				path := filepath.Join(t.TempDir(), "narrow-discovery.json")
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				run.Discovery = nativeEvidenceFile{path, lifecycleDigest(raw)}
				if mutation == "focused-empty" {
					run.RawJSON = nativeEvidenceParserWriteEvents(t, filepath.Join(t.TempDir(), "empty-focused.jsonl"), []nativeEvidenceGoEvent{{Action: "start", Package: pkg}, {Action: "pass", Package: pkg}})
				}
			}
			if err := nativeEvidenceRegressionRunCheck(name, run, authority); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s accepted or failed at unrelated boundary: %v", mutation, err)
			}
		})
	}
}

func TestCodexNativeEvidenceTestCorpus(t *testing.T) {
	files := map[string]string{"cmd/example_test.go": lifecycleDigest([]byte("same test name, original assertion"))}
	corpus := nativeEvidenceTestCorpusDigest(files)
	r := nativeEvidenceRegression{SchemaVersion: "aether-native-regression/v1", QualificationSHA256: "qualification", ProductionDigest: "production", HarnessSHA256: "live-harness", TestCorpusDigest: corpus}
	if err := nativeEvidenceRegressionIdentity(r, "qualification", "production", "live-harness", corpus); err != nil {
		t.Fatalf("valid corpus identity: %v", err)
	}
	files["cmd/example_test.go"] = lifecycleDigest([]byte("same test name, weakened assertion"))
	changed := nativeEvidenceTestCorpusDigest(files)
	if changed == corpus {
		t.Fatal("test body did not change independently derived corpus")
	}
	if err := nativeEvidenceRegressionIdentity(r, "qualification", "production", "live-harness", changed); err == nil || !strings.Contains(err.Error(), "test corpus changed") {
		t.Fatalf("stale broad test corpus accepted or wrong refusal: %v", err)
	}
}

func TestCodexNativeEvidenceMatrix(t *testing.T) {
	const pkg = "github.com/calcosmic/Aether/cmd"
	const top = "TestCodexNativeWorkerFreshHost"
	var expected []string
	for _, s := range codexNativeLiveScenarios {
		expected = append(expected, s.Name)
	}
	for _, mutation := range []string{"missing-package-terminal", "missing-outer-test-terminal", "failed-package", "race-output", "terminal-without-run"} {
		t.Run(mutation, func(t *testing.T) {
			events := []nativeEvidenceGoEvent{{Action: "start", Package: pkg}, {Action: "run", Package: pkg, Test: top}}
			for _, name := range expected {
				events = append(events, nativeEvidenceGoEvent{Action: "run", Package: pkg, Test: top + "/" + name}, nativeEvidenceGoEvent{Action: "pass", Package: pkg, Test: top + "/" + name})
			}
			events = append(events, nativeEvidenceGoEvent{Action: "pass", Package: pkg, Test: top}, nativeEvidenceGoEvent{Action: "pass", Package: pkg})
			encode := func() []byte {
				t.Helper()
				var out bytes.Buffer
				for _, e := range events {
					if err := json.NewEncoder(&out).Encode(e); err != nil {
						t.Fatal(err)
					}
				}
				return out.Bytes()
			}
			if err := nativeEvidenceMatrixEvents(encode(), expected); err != nil {
				t.Fatalf("valid complete matrix fixture: %v", err)
			}
			want := ""
			switch mutation {
			case "missing-package-terminal":
				events = events[:len(events)-1]
				want = "outer package/test completion absent"
			case "missing-outer-test-terminal":
				events = append(events[:len(events)-2], events[len(events)-1])
				want = "outer test not complete"
			case "failed-package":
				events[len(events)-1].Action = "fail"
				want = "mandatory matrix ended fail"
			case "race-output":
				events = append(events[:2], append([]nativeEvidenceGoEvent{{Action: "output", Package: pkg, Output: "WARNING: DATA RACE\n"}}, events[2:]...)...)
				want = "fatal/race diagnostics"
			case "terminal-without-run":
				events = append(events[:2], events[3:]...)
				want = "terminal without unique prior run"
			}
			if err := nativeEvidenceMatrixEvents(encode(), expected); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("%s accepted or wrong refusal: %v", mutation, err)
			}
		})
	}
}

// Registered legacy entrypoint must route the two explicit receipt schemas.
// Its positive baseline is an actually replayable deterministic raw receipt;
// the aggregate refusal below proves dispatch, not actual qualification.
func TestCodexNativeEvidenceReceiptSchemaDispatch(t *testing.T) {
	r := nativeInventoryReplayFixture(t)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	run := func(input []byte) (string, error) {
		path := filepath.Join(t.TempDir(), "receipt.json")
		if err := os.WriteFile(path, input, 0600); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(os.Args[0], "-test.run", "^TestCodexNativeWorkerReceiptValidation$", "-test.v")
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "AETHER_CODEX_NATIVE_") {
				command.Env = append(command.Env, entry)
			}
		}
		command.Env = append(command.Env, "AETHER_CODEX_NATIVE_RECEIPT_PATH="+path)
		output, err := command.CombinedOutput()
		return string(output), err
	}
	for _, mode := range []string{"unknown_schema", "aggregate_dispatch"} {
		t.Run(mode, func(t *testing.T) {
			if output, err := run(raw); err != nil || !strings.Contains(output, "raw native receipt replay passed") {
				t.Fatalf("positive raw receipt baseline failed: %v\n%s", err, output)
			}
			var object map[string]any
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			want := "unsupported native receipt schema"
			if mode == "unknown_schema" {
				object["schema_version"] = "unsupported/v1"
			} else {
				object = map[string]any{"schema_version": "aether-native-final-qualification/v1"}
				want = "required actual native qualification failed: qualification schema/source freeze identity missing"
			}
			mutated, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			output, err := run(mutated)
			if err == nil || !strings.Contains(output, want) {
				t.Fatalf("schema route did not reach expected boundary %q: %v\n%s", want, err, output)
			}
		})
	}
}
