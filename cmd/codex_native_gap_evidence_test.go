package cmd

import (
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A separate manifest pins the whole frozen validator corpus and measured
// toolchain/host inputs. An unavailable schema export remains missing proof;
// observed tool names or successful process exits cannot fill that field.
type nativeGapFrozenProvenance struct {
	Source           string                             `json:"source"`
	Production       string                             `json:"production_digest"`
	Harness          string                             `json:"harness_sha256"`
	TestCorpus       map[string]string                  `json:"test_corpus"`
	TestCorpusDigest string                             `json:"test_corpus_digest"`
	GoVersion        string                             `json:"go_version"`
	GoExecutable     nativeEvidenceFile                 `json:"go_executable"`
	GoEnvironment    nativeEvidenceFile                 `json:"go_environment"`
	Hosts            map[string]nativeGapHostProvenance `json:"hosts"`
}

type nativeGapHostProvenance struct {
	Version       string             `json:"version"`
	Model         string             `json:"model"`
	Args          []string           `json:"args"`
	Executable    nativeEvidenceFile `json:"executable"`
	ToolSchema    nativeEvidenceFile `json:"tool_schema"`
	Configuration nativeEvidenceFile `json:"configuration"`
}

func nativeGapFrozenIdentity(p nativeGapFrozenProvenance, source, production, harness string, corpus string) error {
	if p.Source != source || p.Production != production || p.Harness != harness {
		return fmt.Errorf("frozen source/production/harness binding differs")
	}
	if corpus == "" || len(p.TestCorpus) == 0 || p.TestCorpusDigest != corpus || nativeEvidenceTestCorpusDigest(p.TestCorpus) != corpus {
		return fmt.Errorf("frozen test corpus changed")
	}
	if !strings.HasPrefix(p.GoVersion, "go") {
		return fmt.Errorf("actual Go version absent")
	}
	for _, ref := range []nativeEvidenceFile{p.GoExecutable, p.GoEnvironment} {
		if _, err := nativeEvidenceBytes(ref); err != nil {
			return err
		}
	}
	return nil
}

func nativeGapFrozenProvenanceCheck(root string, q nativeEvidenceQualification, production, harness string) error {
	raw, err := nativeEvidenceBytes(q.FrozenProvenance)
	if err != nil {
		return fmt.Errorf("frozen provenance: %w", err)
	}
	var p nativeGapFrozenProvenance
	if err = json.Unmarshal(raw, &p); err != nil {
		return err
	}
	corpus, err := nativeEvidenceCurrentTestCorpus(root)
	if err != nil {
		return err
	}
	if err = nativeGapFrozenIdentity(p, q.TestedSource, production, harness, corpus); err != nil {
		return err
	}
	var goEnv map[string]string
	raw, err = nativeEvidenceBytes(p.GoEnvironment)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &goEnv); err != nil {
		return err
	}
	if goEnv["GOVERSION"] != p.GoVersion || goEnv["GOOS"] == "" || goEnv["GOARCH"] == "" {
		return fmt.Errorf("actual Go environment differs or is absent")
	}
	if len(p.Hosts) != len(q.Cases) {
		return fmt.Errorf("frozen host inventory incomplete")
	}
	for name, c := range q.Cases {
		raw, err := nativeEvidenceBytes(nativeEvidenceFile{c.OriginalReceipt, c.OriginalReceiptSHA256})
		if err != nil {
			return err
		}
		var r codexNativeLiveReceipt
		if err = json.Unmarshal(raw, &r); err != nil {
			return err
		}
		info, err := buildinfo.ReadFile(r.CandidatePath)
		if err != nil {
			return err
		}
		if info.GoVersion != p.GoVersion {
			return fmt.Errorf("%s candidate Go version differs", name)
		}
		if err = nativeGapHostIdentity(p.Hosts[name], r); err != nil {
			return fmt.Errorf("%s host provenance: %w", name, err)
		}
	}
	return nil
}

func nativeGapHostIdentity(p nativeGapHostProvenance, r codexNativeLiveReceipt) error {
	if p.Version == "" || p.Version != r.ClientVersion || p.Model == "" || p.Model != r.Model || len(p.Args) == 0 || !reflect.DeepEqual(p.Args, r.Args) || p.Executable.Path != r.ClientPath || p.Executable.SHA256 != r.ClientSHA256 {
		return fmt.Errorf("actual host version/model/configuration binding differs or is absent")
	}
	for _, ref := range []nativeEvidenceFile{p.Executable, p.Configuration, p.ToolSchema} {
		if _, err := nativeEvidenceBytes(ref); err != nil {
			return err
		}
	}
	// Configuration and schema must have been captured with this invocation,
	// not written into the original inventory after the outcome was known.
	for _, ref := range []nativeEvidenceFile{p.Configuration, p.ToolSchema} {
		if r.Artifacts[ref.Path] != ref.SHA256 {
			return fmt.Errorf("host schema/configuration absent from original capture")
		}
	}
	return nil
}

func nativeGapQualificationOutcomes(q nativeEvidenceQualification) error {
	if q.SchemaVersion != "aether-native-final-qualification/v1" || len(q.TestedSource) != 40 || !q.SourceFrozenDuringCapture || !q.SourceFrozenDuringReplay {
		return fmt.Errorf("qualification schema/source freeze identity missing")
	}
	expected := []string{}
	for _, s := range codexNativeLiveScenarios {
		expected = append(expected, s.Name)
	}
	if q.ScenarioCount != len(expected) || len(q.Cases) != len(expected) || !reflect.DeepEqual(q.ExpectedScenarios, expected) {
		return fmt.Errorf("complete required scenario inventory differs from reviewed host matrix")
	}
	if !q.QualificationComplete {
		return fmt.Errorf("qualification_complete is false; mandatory incomplete cases are current gaps, never historical waivers")
	}
	for _, name := range expected {
		want := "passed"
		if name == "missing-skill" || name == "controls" || name == "controls-read-only" || name == "cancellation" || name == "spawn-gap" {
			want = "observed"
		}
		if q.Cases[name].Outcome != want {
			return fmt.Errorf("%s required outcome %s absent", name, want)
		}
	}
	return nil
}

func TestCodexNativeGapEvidencePaths(t *testing.T) {
	t.Run("paths", func(t *testing.T) {
		root := t.TempDir()
		q, r := nativeEvidencePaths(root, "")
		if q != filepath.Join(root, nativeGapEvidenceDirectory, "native-qualification.json") || r != filepath.Join(root, nativeGapEvidenceDirectory, "native-regression.json") {
			t.Fatalf("wrong defaults: %s %s", q, r)
		}
		if _, err := os.ReadFile(q); !os.IsNotExist(err) {
			t.Fatal("missing default evidence must remain missing")
		}
		old := filepath.Join(root, nativeEvidenceDirectory, "native-qualification.json")
		explicit, strict := nativeEvidencePaths(root, old)
		if explicit != old || strict != r {
			t.Fatal("explicit history or strict regression dispatch changed")
		}
	})
	t.Run("qualification", func(t *testing.T) {
		for _, mutation := range []string{"missing-schema", "unknown-schema", "incomplete", "missing-case", "unknown-case", "claude-auth", "refusal-not-observed"} {
			t.Run(mutation, func(t *testing.T) {
				q := nativeEvidenceQualification{SchemaVersion: "aether-native-final-qualification/v1", TestedSource: strings.Repeat("a", 40), SourceFrozenDuringCapture: true, SourceFrozenDuringReplay: true, QualificationComplete: true, Cases: map[string]nativeEvidenceCase{}}
				for _, s := range codexNativeLiveScenarios {
					outcome := "passed"
					if s.Name == "missing-skill" || s.Name == "controls" || s.Name == "controls-read-only" || s.Name == "cancellation" || s.Name == "spawn-gap" {
						outcome = "observed"
					}
					q.Cases[s.Name] = nativeEvidenceCase{Outcome: outcome}
					q.ExpectedScenarios = append(q.ExpectedScenarios, s.Name)
				}
				q.ScenarioCount = len(q.Cases)
				if err := nativeGapQualificationOutcomes(q); err != nil {
					t.Fatalf("valid eleven-case control: %v", err)
				}
				switch mutation {
				case "missing-schema":
					q.SchemaVersion = ""
				case "unknown-schema":
					q.SchemaVersion = "unknown/v1"
				case "incomplete":
					q.QualificationComplete = false
				case "missing-case":
					delete(q.Cases, "question")
				case "unknown-case":
					q.ExpectedScenarios[0] = "unknown"
				case "claude-auth":
					q.Cases["claude"] = nativeEvidenceCase{Outcome: "incomplete"}
				case "refusal-not-observed":
					q.Cases["missing-skill"] = nativeEvidenceCase{Outcome: "passed"}
				}
				if err := nativeGapQualificationOutcomes(q); err == nil {
					t.Fatal("invalid matrix qualified")
				}
			})
		}
	})
	t.Run("frozen-identity", func(t *testing.T) {
		for _, mutation := range []string{"165db28c", "f9e32ca0", "cf51e9ac", "production", "harness", "corpus", "go-version", "go-executable", "go-environment"} {
			t.Run(mutation, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, "measured-input")
				raw := []byte("synthetic measured input")
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				ref := nativeEvidenceFile{path, lifecycleDigest(raw)}
				corpus := map[string]string{"cmd/test_test.go": "digest"}
				source := strings.Repeat("a", 40)
				p := nativeGapFrozenProvenance{Source: source, Production: "production", Harness: "harness", TestCorpus: corpus, TestCorpusDigest: nativeEvidenceTestCorpusDigest(corpus), GoVersion: "go1.26.5", GoExecutable: ref, GoEnvironment: ref}
				if err := nativeGapFrozenIdentity(p, source, "production", "harness", nativeEvidenceTestCorpusDigest(corpus)); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "165db28c", "f9e32ca0", "cf51e9ac":
					p.Source = mutation + strings.Repeat("0", 32)
				case "production":
					p.Production = "old"
				case "harness":
					p.Harness = "old"
				case "corpus":
					p.TestCorpusDigest = "old"
				case "go-version":
					p.GoVersion = ""
				case "go-executable":
					p.GoExecutable.SHA256 = "changed"
				case "go-environment":
					p.GoEnvironment.Path = ""
				}
				if err := nativeGapFrozenIdentity(p, source, "production", "harness", nativeEvidenceTestCorpusDigest(corpus)); err == nil {
					t.Fatal("stale frozen identity accepted")
				}
			})
		}
	})
	t.Run("host-provenance", func(t *testing.T) {
		for _, mutation := range []string{"version", "model", "args", "binary", "schema", "configuration", "original-inventory"} {
			t.Run(mutation, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "synthetic-captured-input")
				raw := []byte("synthetic host evidence")
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
				ref := nativeEvidenceFile{path, lifecycleDigest(raw)}
				r := codexNativeLiveReceipt{ClientVersion: "measured", Model: "measured-model", ClientPath: path, ClientSHA256: ref.SHA256, Args: []string{"actual-argv"}, Artifacts: map[string]string{path: ref.SHA256}}
				p := nativeGapHostProvenance{Version: r.ClientVersion, Model: r.Model, Args: r.Args, Executable: ref, ToolSchema: ref, Configuration: ref}
				if err := nativeGapHostIdentity(p, r); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "version":
					p.Version = ""
				case "model":
					p.Model = ""
				case "args":
					p.Args = nil
				case "binary":
					p.Executable.SHA256 = "changed"
				case "schema":
					p.ToolSchema.Path = ""
				case "configuration":
					p.Configuration.SHA256 = "changed"
				case "original-inventory":
					r.Artifacts = nil
				}
				if err := nativeGapHostIdentity(p, r); err == nil {
					t.Fatal("missing host link accepted")
				}
			})
		}
	})
}

func nativeGapReceiptSchemas(original, derived string) bool {
	return (original == "codex-native-tracer/v1" || original == "codex-native-tracer/v2") && derived == original
}

func TestCodexNativeGapEvidenceReceiptVersions(t *testing.T) {
	for _, version := range []string{"codex-native-tracer/v1", "codex-native-tracer/v2"} {
		if !nativeGapReceiptSchemas(version, version) {
			t.Fatalf("known schema rejected: %s", version)
		}
		for _, bad := range []string{"", "unknown/v1"} {
			if nativeGapReceiptSchemas(bad, version) || nativeGapReceiptSchemas(version, bad) {
				t.Fatal("missing/unknown schema accepted")
			}
		}
	}
	if nativeGapReceiptSchemas("codex-native-tracer/v1", "codex-native-tracer/v2") {
		t.Fatal("receipt schema silently relabeled")
	}
}
