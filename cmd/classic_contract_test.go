package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

const classicContractSchemaVersion = "classic-contract/v1"

var (
	classicContractGroups = []string{
		"front-door",
		"territory",
		"orientation",
		"autopilot",
		"agency",
		"pause-resume",
		"closure",
		"maintenance",
	}
	classicContractPlatforms = []string{"runtime", "claude", "opencode"}
	classicContractDecisions = []string{
		"SYN-199-01",
		"SYN-199-02",
		"SYN-199-03",
		"SYN-199-04",
		"SYN-199-05",
		"SYN-199-06",
		"SYN-199-07",
		"SYN-199-08",
		"SYN-199-09",
		"SYN-199-10",
	}
	classicContractCapabilities = []string{
		"CAP-006", "CAP-007", "CAP-008", "CAP-013", "CAP-015", "CAP-016", "CAP-017",
		"CAP-018", "CAP-019", "CAP-020", "CAP-026", "CAP-027", "CAP-028", "CAP-032",
		"CAP-033", "CAP-034", "CAP-035", "CAP-036", "CAP-037", "CAP-038", "CAP-039",
		"CAP-040", "CAP-041", "CAP-042", "CAP-049", "CAP-050", "CAP-052", "CAP-053",
		"CAP-059", "CAP-060", "CAP-062", "CAP-064", "CAP-065", "CAP-068",
	}
	classicContractCaseIDPattern   = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	classicContractDecisionPattern = regexp.MustCompile(`^SYN-199-(?:0[1-9]|10)$`)
	classicContractCAPPattern      = regexp.MustCompile(`^CAP-[0-9]{3}$`)
)

type classicContractJSONSchema struct {
	Draft                string                     `json:"$schema"`
	ID                   string                     `json:"$id"`
	Title                string                     `json:"title"`
	Description          string                     `json:"description"`
	Type                 string                     `json:"type"`
	AdditionalProperties bool                       `json:"additionalProperties"`
	Required             []string                   `json:"required"`
	Properties           map[string]json.RawMessage `json:"properties"`
	Definitions          map[string]json.RawMessage `json:"$defs"`
}

type classicContractDocument struct {
	SchemaVersion string                `json:"schema_version"`
	Cases         []classicContractCase `json:"cases"`
}

type classicContractCase struct {
	ID                string                     `json:"id"`
	Group             string                     `json:"group"`
	Kind              string                     `json:"kind"`
	HistoricalAnchors []string                   `json:"historical_anchors"`
	SynthesisDecision string                     `json:"synthesis_decision"`
	CAPIDs            []string                   `json:"cap_ids"`
	SourceCitations   []string                   `json:"source_citations"`
	Platform          string                     `json:"platform"`
	Command           classicContractInvocation  `json:"command"`
	Expected          classicContractExpectation `json:"expected"`
}

type classicContractInvocation struct {
	Name        string            `json:"name"`
	Args        []string          `json:"args"`
	Environment map[string]string `json:"environment"`
}

type classicContractExpectation struct {
	ExitCode            int                        `json:"exit_code"`
	Outcome             string                     `json:"outcome"`
	SemanticFields      map[string]json.RawMessage `json:"semantic_fields"`
	RequiredTextTokens  []string                   `json:"required_text_tokens,omitempty"`
	ForbiddenTextTokens []string                   `json:"forbidden_text_tokens,omitempty"`
	StateAssertions     []classicStateAssertion    `json:"state_assertions,omitempty"`
	ForbiddenArtifacts  []string                   `json:"forbidden_artifacts,omitempty"`
	Replay              *classicReplayAssertion    `json:"replay,omitempty"`
	FaultPoint          string                     `json:"fault_point,omitempty"`
}

type classicStateAssertion struct {
	Path     string          `json:"path"`
	Operator string          `json:"operator"`
	Value    json.RawMessage `json:"value,omitempty"`
}

type classicReplayAssertion struct {
	Mode            string `json:"mode"`
	ExpectedOutcome string `json:"expected_outcome"`
}

type classicMechanismRegistry struct {
	SchemaVersion   string             `json:"schema_version"`
	SynthesisSource string             `json:"synthesis_source"`
	Mechanisms      []classicMechanism `json:"mechanisms"`
}

type classicMechanism struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Disposition     string   `json:"disposition"`
	CAPIDs          []string `json:"cap_ids"`
	PublicCommands  []string `json:"public_commands"`
	SourceCitations []string `json:"source_citations"`
	Groups          []string `json:"groups"`
}

func TestClassicContractSchema(t *testing.T) {
	contractDir := classicContractFixtureDir(t)
	schema := loadClassicContractJSONSchema(t, filepath.Join(contractDir, "schema.json"))
	before := snapshotStoreFileHashesForTest(t, contractDir)

	if schema.Draft != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("schema draft = %q, want JSON Schema 2020-12", schema.Draft)
	}
	if schema.ID != "https://aether.dev/schemas/classic-contract/v1" {
		t.Fatalf("schema id = %q, want versioned v1 identifier", schema.ID)
	}
	if schema.Type != "object" || schema.AdditionalProperties {
		t.Fatalf("top-level schema must be a closed object: type=%q additionalProperties=%v", schema.Type, schema.AdditionalProperties)
	}
	for _, field := range []string{"schema_version", "cases"} {
		if !slices.Contains(schema.Required, field) {
			t.Errorf("schema missing required top-level field %q", field)
		}
		if _, ok := schema.Properties[field]; !ok {
			t.Errorf("schema missing property %q", field)
		}
	}
	assertClassicSchemaEnum(t, schema, "case", "group", classicContractGroups)
	assertClassicSchemaEnum(t, schema, "case", "platform", classicContractPlatforms)
	assertClassicSchemaCausalRequirement(t, schema)

	valid := classicContractValidDocument()
	validPath := filepath.Join(t.TempDir(), "valid.json")
	writeClassicContractJSON(t, validPath, valid)
	loaded, err := loadClassicContractDocument(validPath)
	if err != nil {
		t.Fatalf("load valid Classic contract: %v", err)
	}
	if len(loaded.Cases) != 1 {
		t.Fatalf("valid case count = %d, want 1", len(loaded.Cases))
	}
	got := loaded.Cases[0]
	if len(got.Command.Args) == 0 || len(got.Command.Environment) == 0 || len(got.Expected.RequiredTextTokens) == 0 || len(got.Expected.ForbiddenTextTokens) == 0 || len(got.Expected.StateAssertions) == 0 || len(got.Expected.ForbiddenArtifacts) == 0 || got.Expected.Replay == nil || got.Expected.FaultPoint == "" {
		t.Fatalf("valid case did not preserve the full invocation/assertion contract: %+v", got)
	}

	t.Run("unknown field", func(t *testing.T) {
		raw := classicContractJSONMap(t, valid)
		cases := raw["cases"].([]any)
		cases[0].(map[string]any)["unexpected"] = true
		assertClassicContractLoadError(t, raw, `unknown field "unexpected"`)
	})

	t.Run("duplicate case id", func(t *testing.T) {
		invalid := classicContractValidDocument()
		invalid.Cases = append(invalid.Cases, invalid.Cases[0])
		assertClassicContractLoadError(t, invalid, `duplicate case id "front-door.early-run.requires-init"`)
	})

	t.Run("invalid group", func(t *testing.T) {
		invalid := classicContractValidDocument()
		invalid.Cases[0].Group = "ceremony"
		assertClassicContractLoadError(t, invalid, `invalid group "ceremony"`)
	})

	t.Run("invalid platform", func(t *testing.T) {
		invalid := classicContractValidDocument()
		invalid.Cases[0].Platform = "codex"
		assertClassicContractLoadError(t, invalid, `invalid platform "codex"`)
	})

	t.Run("output-only behavior", func(t *testing.T) {
		invalid := classicContractValidDocument()
		invalid.Cases[0].Expected.StateAssertions = nil
		invalid.Cases[0].Expected.ForbiddenArtifacts = nil
		assertClassicContractLoadError(t, invalid, "missing causal state_assertions or forbidden_artifacts")
	})

	after := snapshotStoreFileHashesForTest(t, contractDir)
	assertHashSnapshotsEqualForTest(t, "Classic contract schema validation", before, after)
}

func TestClassicMechanismCoverage(t *testing.T) {
	contractDir := classicContractFixtureDir(t)
	before := snapshotStoreFileHashesForTest(t, contractDir)
	registry, err := loadClassicMechanismRegistry(filepath.Join(contractDir, "mechanisms.json"))
	if err != nil {
		t.Fatalf("load Classic mechanism registry: %v", err)
	}

	groups := make(map[string]bool, len(classicContractGroups))
	for _, mechanism := range registry.Mechanisms {
		for _, group := range mechanism.Groups {
			groups[group] = true
		}
	}
	for _, group := range classicContractGroups {
		if !groups[group] {
			t.Errorf("mechanism registry missing corpus group %q", group)
		}
	}
	if len(registry.Mechanisms) != len(classicContractDecisions) {
		t.Fatalf("mechanism count = %d, want %d", len(registry.Mechanisms), len(classicContractDecisions))
	}

	t.Run("exact missing synthesis identifier", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms = invalid.Mechanisms[:len(invalid.Mechanisms)-1]
		assertClassicMechanismError(t, invalid, `missing synthesis decision "SYN-199-10"`)
	})

	t.Run("exact duplicate synthesis identifier", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms = append(invalid.Mechanisms, invalid.Mechanisms[0])
		assertClassicMechanismError(t, invalid, `duplicate synthesis decision "SYN-199-01"`)
	})

	t.Run("exact missing capability identifier", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		removeClassicMechanismCAP(t, &invalid, "CAP-016")
		assertClassicMechanismError(t, invalid, `missing capability "CAP-016"`)
	})

	t.Run("exact duplicate capability identifier", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms[1].CAPIDs = append(invalid.Mechanisms[1].CAPIDs, "CAP-016")
		assertClassicMechanismError(t, invalid, `duplicate capability "CAP-016"`)
	})

	repoRoot := findTestModuleRoot(t)
	synthesisPath := filepath.Join(repoRoot, filepath.FromSlash(registry.SynthesisSource))
	synthesis, err := os.ReadFile(synthesisPath)
	if err != nil {
		t.Fatalf("read mechanism synthesis %s: %v", synthesisPath, err)
	}
	for _, id := range classicContractDecisions {
		if count := strings.Count(string(synthesis), id); count != 1 {
			t.Errorf("synthesis identifier %s occurs %d times, want exactly once", id, count)
		}
	}
	for _, id := range classicContractCapabilities {
		if !bytes.Contains(synthesis, []byte(id)) {
			t.Errorf("synthesis missing routed capability %s", id)
		}
	}

	after := snapshotStoreFileHashesForTest(t, contractDir)
	assertHashSnapshotsEqualForTest(t, "Classic mechanism validation", before, after)
}

// The journey IDs are intentionally semantic rather than a list of commands.
// A public journey has one Claude and one OpenCode row; platform expansion is
// verified below instead of allowing either wrapper to stand in for the other.
var classicContractRequiredJourneyIDs = []string{
	"front-door.help-no-colony", "front-door.help-active-colony", "front-door.init-first",
	"territory.fresh", "territory.refreshed", "territory.unavailable", "territory.init-active-refusal",
	"autopilot.valid", "autopilot.repair", "autopilot.debt", "autopilot.no-colony", "autopilot.authority-fence",
	"orientation.status-full", "orientation.status-compact", "orientation.phase", "orientation.history", "orientation.watch-idle", "orientation.read-only", "orientation.unavailable",
	"steering.focus", "steering.feedback", "steering.redirect", "steering.swarm",
	"pause-resume.clean-handoff", "pause-resume.stale-handoff", "pause-resume.reconstruction", "pause-resume.legacy-hidden", "pause-resume.journal-fault", "pause-resume.conflict", "pause-resume.replay",
	"seal.verified", "seal.incomplete-refusal", "seal.forced-residual", "seal.owner-checkpoint",
	"entomb.verified", "entomb.forced", "entomb.corrupt-refusal", "entomb.publish-fault", "entomb.replay",
	"maintenance.migration", "maintenance.update", "maintenance.cleanup", "maintenance.integrity",
}

func TestClassicContractCorpusRequiredCategories(t *testing.T) {
	document := loadClassicContractCorpus(t)
	assertClassicContractJourneyMatrix(t, document)
}

func TestClassicContractCorpusRejectsMissingFrontDoorCase(t *testing.T) {
	document := loadClassicContractCorpus(t)
	for _, requiredID := range classicContractRequiredJourneyIDs[:7] {
		t.Run(requiredID, func(t *testing.T) {
			invalid := cloneClassicContractDocument(t, document)
			invalid.Cases = classicContractWithoutJourney(invalid.Cases, requiredID)
			if err := validateClassicContractCorpus(invalid); err == nil || !strings.Contains(err.Error(), requiredID) {
				t.Fatalf("missing journey error = %v, want %q", err, requiredID)
			}
		})
	}
}

func TestClassicContractCorpusClaude(t *testing.T) {
	classicContractExecutePlatform(t, "claude")
}

func TestClassicContractCorpusOpenCode(t *testing.T) {
	classicContractExecutePlatform(t, "opencode")
}

func TestClassicContractCorpusCausalReceipts(t *testing.T) {
	document := loadClassicContractCorpus(t)
	for _, testCase := range document.Cases {
		for _, field := range []string{"fixture", "requirement_ids", "pre_post_digest", "artifact_or_receipt", "structured_assertions"} {
			if _, ok := testCase.Expected.SemanticFields[field]; !ok {
				t.Errorf("%s missing causal semantic field %q", testCase.ID, field)
			}
		}
		if len(testCase.Expected.RequiredTextTokens) == 0 || len(testCase.Expected.ForbiddenTextTokens) == 0 || testCase.Expected.Replay == nil || testCase.Expected.FaultPoint == "" {
			t.Errorf("%s lacks visual/replay/fault evidence", testCase.ID)
		}
	}
}

func loadClassicContractCorpus(t *testing.T) classicContractDocument {
	t.Helper()
	document, err := loadClassicContractDocument(filepath.Join(classicContractFixtureDir(t), "cases.json"))
	if err != nil {
		t.Fatalf("load Classic contract corpus: %v", err)
	}
	if err := validateClassicContractCorpus(document); err != nil {
		t.Fatalf("validate Classic contract corpus: %v", err)
	}
	return document
}

func validateClassicContractCorpus(document classicContractDocument) error {
	if err := validateClassicContractDocument(document); err != nil {
		return err
	}
	required := make(map[string]bool, len(classicContractRequiredJourneyIDs))
	for _, journeyID := range classicContractRequiredJourneyIDs {
		required[journeyID] = true
	}
	platforms := make(map[string]map[string]bool, len(required))
	for _, testCase := range document.Cases {
		journeyID, _, ok := strings.Cut(testCase.ID, ".claude")
		if !ok {
			journeyID, _, ok = strings.Cut(testCase.ID, ".opencode")
		}
		if !ok {
			return fmt.Errorf("case %q must end with .claude or .opencode", testCase.ID)
		}
		if !required[journeyID] {
			return fmt.Errorf("unknown or combined journey %q", testCase.ID)
		}
		if platforms[journeyID] == nil {
			platforms[journeyID] = map[string]bool{}
		}
		if platforms[journeyID][testCase.Platform] {
			return fmt.Errorf("duplicate platform %q for journey %q", testCase.Platform, journeyID)
		}
		platforms[journeyID][testCase.Platform] = true
		if testCase.Platform != strings.TrimPrefix(testCase.ID, journeyID+".") {
			return fmt.Errorf("case %q platform %q does not match its ID", testCase.ID, testCase.Platform)
		}
		for _, field := range []string{"fixture", "requirement_ids", "pre_post_digest", "artifact_or_receipt", "structured_assertions"} {
			value, ok := testCase.Expected.SemanticFields[field]
			if !ok || len(value) == 0 || string(value) == "null" || string(value) == `""` {
				return fmt.Errorf("case %q missing causal semantic field %q", testCase.ID, field)
			}
		}
		if !slices.Contains(testCase.SourceCitations, "requirement:PROOF-01") {
			return fmt.Errorf("case %q does not cite PROOF-01", testCase.ID)
		}
	}
	for _, journeyID := range classicContractRequiredJourneyIDs {
		got := platforms[journeyID]
		if !got["claude"] || !got["opencode"] || len(got) != 2 {
			return fmt.Errorf("required journey %q must have exactly Claude and OpenCode cases", journeyID)
		}
	}
	if len(platforms) != len(required) || len(document.Cases) != len(required)*2 {
		return fmt.Errorf("corpus has %d cases, want exactly %d platform-expanded required journeys", len(document.Cases), len(required)*2)
	}
	return nil
}

func cloneClassicContractDocument(t *testing.T, document classicContractDocument) classicContractDocument {
	t.Helper()
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal Classic corpus clone: %v", err)
	}
	var clone classicContractDocument
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("unmarshal Classic corpus clone: %v", err)
	}
	return clone
}

func classicContractWithoutJourney(cases []classicContractCase, journeyID string) []classicContractCase {
	result := make([]classicContractCase, 0, len(cases))
	for _, testCase := range cases {
		if strings.HasPrefix(testCase.ID, journeyID+".") {
			continue
		}
		result = append(result, testCase)
	}
	return result
}

func assertClassicContractJourneyMatrix(t *testing.T, document classicContractDocument) {
	t.Helper()
	if err := validateClassicContractCorpus(document); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, testCase := range document.Cases {
		counts[testCase.Group]++
	}
	for group, want := range map[string]int{"front-door": 6, "territory": 8, "autopilot": 10, "orientation": 14, "agency": 8, "pause-resume": 14, "closure": 18, "maintenance": 8} {
		if counts[group] != want {
			t.Errorf("%s cases = %d, want %d (both platforms)", group, counts[group], want)
		}
	}
}

func classicContractExecutePlatform(t *testing.T, platform string) {
	t.Helper()
	document := loadClassicContractCorpus(t)
	harness := newCLIBlackBox(t)
	for _, testCase := range document.Cases {
		if testCase.Platform != platform {
			continue
		}
		t.Run(testCase.ID, func(t *testing.T) {
			classicContractAssertManagedWrapperAuthority(t, harness.sourceRoot, testCase)
			before := classicContractDirectoryDigest(t, harness.repo)
			result := harness.runWithEnv(t, testCase.Command.Environment, append([]string{testCase.Command.Name}, testCase.Command.Args...)...)
			if result.ExitCode != testCase.Expected.ExitCode {
				t.Fatalf("%s exit = %d, want %d\\nstdout:\\n%s\\nstderr:\\n%s", testCase.ID, result.ExitCode, testCase.Expected.ExitCode, result.Stdout, result.Stderr)
			}
			response := result.Stdout
			if result.ExitCode != 0 {
				response = result.Stderr
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &envelope); err != nil {
				t.Fatalf("%s JSON execution result: %v\\n%s", testCase.ID, err, response)
			}
			okValue, ok := envelope["ok"]
			if !ok {
				t.Fatalf("%s has no structured ok field: %s", testCase.ID, response)
			}
			if testCase.Expected.ExitCode == 0 && string(okValue) != "true" {
				t.Fatalf("%s success response ok = %s, want true", testCase.ID, okValue)
			}
			if testCase.Expected.ExitCode != 0 && string(okValue) != "false" {
				t.Fatalf("%s refusal response ok = %s, want false", testCase.ID, okValue)
			}
			after := classicContractDirectoryDigest(t, harness.repo)
			if string(testCase.Expected.SemanticFields["pre_post_digest"]) == `"unchanged"` && before != after {
				t.Fatalf("%s changed isolated state for an unchanged proof: before=%s after=%s", testCase.ID, before, after)
			}
			for _, assertion := range testCase.Expected.StateAssertions {
				if assertion.Operator == "absent" {
					if _, err := os.Stat(filepath.Join(harness.repo, filepath.FromSlash(assertion.Path))); !os.IsNotExist(err) {
						t.Fatalf("%s expected artifact %s absent, stat err=%v", testCase.ID, assertion.Path, err)
					}
				}
			}
			visual := harness.runWithEnv(t, map[string]string{"AETHER_OUTPUT_MODE": "visual", "AETHER_PLATFORM": platform}, testCase.Command.Name)
			for _, token := range testCase.Expected.RequiredTextTokens {
				if !strings.Contains(visual.Stdout+visual.Stderr, token) {
					t.Fatalf("%s visual result missing required beat %q", testCase.ID, token)
				}
			}
			for _, token := range testCase.Expected.ForbiddenTextTokens {
				if strings.Contains(visual.Stdout+visual.Stderr, token) {
					t.Fatalf("%s visual result contains forbidden beat %q", testCase.ID, token)
				}
			}
		})
	}
	harness.assertSourceUnchanged(t)
}

func classicContractAssertManagedWrapperAuthority(t *testing.T, root string, testCase classicContractCase) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(root, ".aether", "commands", testCase.Command.Name+".yaml"),
		filepath.Join(root, "."+testCase.Platform, "commands", "ant", testCase.Command.Name+".md"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s managed wrapper authority missing %s: %v", testCase.ID, path, err)
		}
		if strings.Contains(path, "/commands/ant/") && !strings.Contains(string(data), "Aether-managed: runtime spec at .aether/commands/") {
			t.Fatalf("%s platform wrapper is not managed by canonical YAML: %s", testCase.ID, path)
		}
	}
}

func classicContractDirectoryDigest(t *testing.T, root string) string {
	t.Helper()
	var records []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		records = append(records, filepath.ToSlash(rel)+":"+fmt.Sprintf("%x", sum))
		return nil
	})
	if err != nil {
		t.Fatalf("digest isolated repo: %v", err)
	}
	sort.Strings(records)
	sum := sha256.Sum256([]byte(strings.Join(records, "\\n")))
	return fmt.Sprintf("sha256:%x", sum)
}

func classicContractFixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(findTestModuleRoot(t), "cmd", "testdata", "classic-contract", "v1")
}

func loadClassicContractJSONSchema(t *testing.T, path string) classicContractJSONSchema {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Classic contract schema %s: %v", path, err)
	}
	var schema classicContractJSONSchema
	if err := decodeClassicStrictJSON(data, &schema); err != nil {
		t.Fatalf("decode Classic contract schema %s: %v", path, err)
	}
	return schema
}

func loadClassicContractDocument(path string) (classicContractDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return classicContractDocument{}, fmt.Errorf("read Classic contract %s: %w", path, err)
	}
	var document classicContractDocument
	if err := decodeClassicStrictJSON(data, &document); err != nil {
		return classicContractDocument{}, fmt.Errorf("decode Classic contract %s: %w", path, err)
	}
	if err := validateClassicContractDocument(document); err != nil {
		return classicContractDocument{}, fmt.Errorf("validate Classic contract %s: %w", path, err)
	}
	return document, nil
}

func validateClassicContractDocument(document classicContractDocument) error {
	if document.SchemaVersion != classicContractSchemaVersion {
		return fmt.Errorf("schema_version = %q, want %q", document.SchemaVersion, classicContractSchemaVersion)
	}
	if len(document.Cases) == 0 {
		return fmt.Errorf("contract has no cases")
	}
	seen := make(map[string]bool, len(document.Cases))
	for index, testCase := range document.Cases {
		if !classicContractCaseIDPattern.MatchString(testCase.ID) {
			return fmt.Errorf("case %d has invalid id %q", index, testCase.ID)
		}
		if seen[testCase.ID] {
			return fmt.Errorf("duplicate case id %q", testCase.ID)
		}
		seen[testCase.ID] = true
		if !slices.Contains(classicContractGroups, testCase.Group) {
			return fmt.Errorf("case %q has invalid group %q", testCase.ID, testCase.Group)
		}
		if testCase.Kind != "behavior" {
			return fmt.Errorf("case %q has invalid kind %q", testCase.ID, testCase.Kind)
		}
		if len(testCase.HistoricalAnchors) == 0 || classicStringsContainBlank(testCase.HistoricalAnchors) {
			return fmt.Errorf("case %q requires historical_anchors", testCase.ID)
		}
		if !classicContractDecisionPattern.MatchString(testCase.SynthesisDecision) {
			return fmt.Errorf("case %q has invalid synthesis_decision %q", testCase.ID, testCase.SynthesisDecision)
		}
		if testCase.CAPIDs == nil {
			return fmt.Errorf("case %q requires cap_ids", testCase.ID)
		}
		for _, capID := range testCase.CAPIDs {
			if !classicContractCAPPattern.MatchString(capID) {
				return fmt.Errorf("case %q has invalid capability %q", testCase.ID, capID)
			}
		}
		if len(testCase.SourceCitations) == 0 || classicStringsContainBlank(testCase.SourceCitations) {
			return fmt.Errorf("case %q requires source_citations", testCase.ID)
		}
		if !slices.Contains(classicContractPlatforms, testCase.Platform) {
			return fmt.Errorf("case %q has invalid platform %q", testCase.ID, testCase.Platform)
		}
		if strings.TrimSpace(testCase.Command.Name) == "" {
			return fmt.Errorf("case %q requires command.name", testCase.ID)
		}
		if testCase.Command.Args == nil || testCase.Command.Environment == nil {
			return fmt.Errorf("case %q requires command args and environment", testCase.ID)
		}
		if strings.TrimSpace(testCase.Expected.Outcome) == "" {
			return fmt.Errorf("case %q requires expected.outcome", testCase.ID)
		}
		if len(testCase.Expected.SemanticFields) == 0 {
			return fmt.Errorf("case %q requires expected.semantic_fields", testCase.ID)
		}
		if len(testCase.Expected.StateAssertions) == 0 && len(testCase.Expected.ForbiddenArtifacts) == 0 {
			return fmt.Errorf("case %q missing causal state_assertions or forbidden_artifacts", testCase.ID)
		}
		for _, assertion := range testCase.Expected.StateAssertions {
			if strings.TrimSpace(assertion.Path) == "" || !slices.Contains([]string{"equals", "exists", "absent", "unchanged", "changed", "contains"}, assertion.Operator) {
				return fmt.Errorf("case %q has invalid state assertion %+v", testCase.ID, assertion)
			}
		}
		if classicStringsContainBlank(testCase.Expected.ForbiddenArtifacts) {
			return fmt.Errorf("case %q has blank forbidden artifact", testCase.ID)
		}
		if testCase.Expected.Replay != nil {
			if !slices.Contains([]string{"idempotent", "reject-altered", "not-applicable"}, testCase.Expected.Replay.Mode) || strings.TrimSpace(testCase.Expected.Replay.ExpectedOutcome) == "" {
				return fmt.Errorf("case %q has invalid replay assertion %+v", testCase.ID, testCase.Expected.Replay)
			}
		}
	}
	return nil
}

func loadClassicMechanismRegistry(path string) (classicMechanismRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return classicMechanismRegistry{}, fmt.Errorf("read Classic mechanism registry %s: %w", path, err)
	}
	var registry classicMechanismRegistry
	if err := decodeClassicStrictJSON(data, &registry); err != nil {
		return classicMechanismRegistry{}, fmt.Errorf("decode Classic mechanism registry %s: %w", path, err)
	}
	if err := validateClassicMechanismRegistry(registry); err != nil {
		return classicMechanismRegistry{}, fmt.Errorf("validate Classic mechanism registry %s: %w", path, err)
	}
	return registry, nil
}

func validateClassicMechanismRegistry(registry classicMechanismRegistry) error {
	if registry.SchemaVersion != classicContractSchemaVersion {
		return fmt.Errorf("schema_version = %q, want %q", registry.SchemaVersion, classicContractSchemaVersion)
	}
	if registry.SynthesisSource != ".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md" {
		return fmt.Errorf("unexpected synthesis_source %q", registry.SynthesisSource)
	}
	if len(registry.Mechanisms) == 0 {
		return fmt.Errorf("mechanism registry is empty")
	}

	decisionCounts := make(map[string]int, len(registry.Mechanisms))
	capCounts := make(map[string]int, len(classicContractCapabilities))
	groupCounts := make(map[string]int, len(classicContractGroups))
	for index, mechanism := range registry.Mechanisms {
		if !classicContractDecisionPattern.MatchString(mechanism.ID) {
			return fmt.Errorf("mechanism %d has invalid synthesis decision %q", index, mechanism.ID)
		}
		decisionCounts[mechanism.ID]++
		if strings.TrimSpace(mechanism.Title) == "" {
			return fmt.Errorf("mechanism %q requires title", mechanism.ID)
		}
		if !slices.Contains([]string{"keep-current", "restore-modern", "replace-better", "retire-with-proof"}, mechanism.Disposition) {
			return fmt.Errorf("mechanism %q has invalid disposition %q", mechanism.ID, mechanism.Disposition)
		}
		if mechanism.CAPIDs == nil {
			return fmt.Errorf("mechanism %q requires cap_ids", mechanism.ID)
		}
		for _, capID := range mechanism.CAPIDs {
			if !classicContractCAPPattern.MatchString(capID) {
				return fmt.Errorf("mechanism %q has invalid capability %q", mechanism.ID, capID)
			}
			capCounts[capID]++
		}
		if len(mechanism.PublicCommands) == 0 || classicStringsContainBlank(mechanism.PublicCommands) {
			return fmt.Errorf("mechanism %q requires public_commands", mechanism.ID)
		}
		if len(mechanism.SourceCitations) == 0 || classicStringsContainBlank(mechanism.SourceCitations) {
			return fmt.Errorf("mechanism %q requires source_citations", mechanism.ID)
		}
		if len(mechanism.Groups) == 0 {
			return fmt.Errorf("mechanism %q requires groups", mechanism.ID)
		}
		for _, group := range mechanism.Groups {
			if !slices.Contains(classicContractGroups, group) {
				return fmt.Errorf("mechanism %q has invalid group %q", mechanism.ID, group)
			}
			groupCounts[group]++
		}
	}

	for _, id := range classicContractDecisions {
		switch decisionCounts[id] {
		case 0:
			return fmt.Errorf("missing synthesis decision %q", id)
		case 1:
		default:
			return fmt.Errorf("duplicate synthesis decision %q", id)
		}
	}
	for id := range decisionCounts {
		if !slices.Contains(classicContractDecisions, id) {
			return fmt.Errorf("unexpected synthesis decision %q", id)
		}
	}
	for _, id := range classicContractCapabilities {
		switch capCounts[id] {
		case 0:
			return fmt.Errorf("missing capability %q", id)
		case 1:
		default:
			return fmt.Errorf("duplicate capability %q", id)
		}
	}
	for id := range capCounts {
		if !slices.Contains(classicContractCapabilities, id) {
			return fmt.Errorf("unexpected capability %q", id)
		}
	}
	for _, group := range classicContractGroups {
		if groupCounts[group] == 0 {
			return fmt.Errorf("missing corpus group %q", group)
		}
	}
	return nil
}

func decodeClassicStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}

func classicContractValidDocument() classicContractDocument {
	return classicContractDocument{
		SchemaVersion: classicContractSchemaVersion,
		Cases: []classicContractCase{{
			ID:                "front-door.early-run.requires-init",
			Group:             "front-door",
			Kind:              "behavior",
			HistoricalAnchors: []string{"v5.4.0:.aether/commands/run.yaml"},
			SynthesisDecision: "SYN-199-05",
			CAPIDs:            []string{},
			SourceCitations:   []string{"cmd/compatibility_cmds.go", ".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md"},
			Platform:          "claude",
			Command: classicContractInvocation{
				Name:        "/ant-run",
				Args:        []string{"--max-phases", "2"},
				Environment: map[string]string{"AETHER_OUTPUT_MODE": "json", "AETHER_PLATFORM": "claude"},
			},
			Expected: classicContractExpectation{
				ExitCode:            0,
				Outcome:             "autopilot.requires-init",
				SemanticFields:      map[string]json.RawMessage{"ok": json.RawMessage("false"), "next_action": json.RawMessage(`"/ant-init"`)},
				RequiredTextTokens:  []string{"/ant-init"},
				ForbiddenTextTokens: []string{"aether init"},
				StateAssertions: []classicStateAssertion{{
					Path:     ".aether/data/COLONY_STATE.json",
					Operator: "absent",
				}},
				ForbiddenArtifacts: []string{".aether/data/build"},
				Replay:             &classicReplayAssertion{Mode: "idempotent", ExpectedOutcome: "autopilot.requires-init"},
				FaultPoint:         "before-first-mutation",
			},
		}},
	}
}

func classicContractJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal Classic contract fixture: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode Classic contract fixture map: %v", err)
	}
	return result
}

func writeClassicContractJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal Classic contract fixture: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatalf("write Classic contract fixture %s: %v", path, err)
	}
}

func assertClassicContractLoadError(t *testing.T, value any, want string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "invalid.json")
	writeClassicContractJSON(t, path, value)
	_, err := loadClassicContractDocument(path)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("load error = %v, want substring %q", err, want)
	}
}

func assertClassicSchemaEnum(t *testing.T, schema classicContractJSONSchema, definition, property string, want []string) {
	t.Helper()
	definitionObject := classicSchemaObject(t, schema.Definitions[definition])
	properties, ok := definitionObject["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema definition %q has no properties object", definition)
	}
	propertyObject, ok := properties[property].(map[string]any)
	if !ok {
		t.Fatalf("schema definition %q has no property %q", definition, property)
	}
	rawEnum, ok := propertyObject["enum"].([]any)
	if !ok {
		t.Fatalf("schema %s.%s has no enum", definition, property)
	}
	got := make([]string, 0, len(rawEnum))
	for _, value := range rawEnum {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("schema %s.%s enum contains non-string %T", definition, property, value)
		}
		got = append(got, text)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("schema %s.%s enum = %v, want %v", definition, property, got, want)
	}
}

func assertClassicSchemaCausalRequirement(t *testing.T, schema classicContractJSONSchema) {
	t.Helper()
	expected := classicSchemaObject(t, schema.Definitions["expected"])
	data, err := json.Marshal(expected["anyOf"])
	if err != nil {
		t.Fatalf("marshal schema causal requirement: %v", err)
	}
	text := string(data)
	for _, field := range []string{"state_assertions", "forbidden_artifacts"} {
		if !strings.Contains(text, field) {
			t.Errorf("expected schema anyOf does not require %s", field)
		}
	}
}

func classicSchemaObject(t *testing.T, data json.RawMessage) map[string]any {
	t.Helper()
	if len(data) == 0 {
		t.Fatal("schema definition is missing")
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode schema definition: %v", err)
	}
	return result
}

func classicStringsContainBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func cloneClassicMechanismRegistry(t *testing.T, registry classicMechanismRegistry) classicMechanismRegistry {
	t.Helper()
	data, err := json.Marshal(registry)
	if err != nil {
		t.Fatalf("marshal mechanism registry clone: %v", err)
	}
	var clone classicMechanismRegistry
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("decode mechanism registry clone: %v", err)
	}
	return clone
}

func removeClassicMechanismCAP(t *testing.T, registry *classicMechanismRegistry, capID string) {
	t.Helper()
	for mechanismIndex := range registry.Mechanisms {
		for capIndex, candidate := range registry.Mechanisms[mechanismIndex].CAPIDs {
			if candidate != capID {
				continue
			}
			caps := registry.Mechanisms[mechanismIndex].CAPIDs
			registry.Mechanisms[mechanismIndex].CAPIDs = append(caps[:capIndex], caps[capIndex+1:]...)
			return
		}
	}
	t.Fatalf("test registry does not contain %s", capID)
}

func assertClassicMechanismError(t *testing.T, registry classicMechanismRegistry, want string) {
	t.Helper()
	err := validateClassicMechanismRegistry(registry)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("mechanism validation error = %v, want substring %q", err, want)
	}
}
