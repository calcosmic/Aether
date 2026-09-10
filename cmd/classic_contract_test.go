package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const classicContractSchemaVersion = "classic-contract/v1"

var (
	classicContractPhase199Groups = []string{
		"front-door",
		"territory",
		"orientation",
		"autopilot",
		"agency",
		"pause-resume",
		"closure",
		"maintenance",
	}
	classicContractPhase200Groups = []string{
		"V-200-E2E",
		"V-200-CONFIDENCE",
		"V-200-DECISION",
		"V-200-STOP",
		"V-200-ACCEPT",
		"V-200-REVISION",
		"V-200-REPLAY",
		"V-200-PLATFORM",
	}
	classicContractPhase201Groups = []string{
		"V-201-BOUNDARY",
		"V-201-OUTCOME",
		"V-201-IDENTITY",
		"V-201-REPAIR",
		"V-201-AUTOPILOT",
		"V-201-TELEMETRY",
	}
	classicContractGroups            = append(append(slices.Clone(classicContractPhase199Groups), classicContractPhase200Groups...), classicContractPhase201Groups...)
	classicContractPlatforms         = []string{"runtime", "claude", "opencode"}
	classicContractPhase199Decisions = []string{
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
	classicContractPhase200Decisions = []string{
		"SYN-200-01", "SYN-200-02", "SYN-200-03", "SYN-200-04",
		"SYN-200-05", "SYN-200-06", "SYN-200-07", "SYN-200-08",
		"SYN-200-09", "SYN-200-10", "SYN-200-11", "SYN-200-12",
	}
	classicContractPhase201Decisions = []string{
		"SYN-201-01", "SYN-201-02", "SYN-201-03", "SYN-201-04",
		"SYN-201-05", "SYN-201-06", "SYN-201-07", "SYN-201-08",
		"SYN-201-09", "SYN-201-10", "SYN-201-11", "SYN-201-12",
		"SYN-201-13", "SYN-201-14",
	}
	classicContractDecisions            = append(append(slices.Clone(classicContractPhase199Decisions), classicContractPhase200Decisions...), classicContractPhase201Decisions...)
	classicContractPhase199Capabilities = []string{
		"CAP-006", "CAP-007", "CAP-008", "CAP-013", "CAP-015", "CAP-016", "CAP-017",
		"CAP-018", "CAP-019", "CAP-020", "CAP-026", "CAP-027", "CAP-028", "CAP-032",
		"CAP-033", "CAP-034", "CAP-035", "CAP-036", "CAP-037", "CAP-038", "CAP-039",
		"CAP-040", "CAP-041", "CAP-042", "CAP-049", "CAP-050", "CAP-052", "CAP-053",
		"CAP-059", "CAP-060", "CAP-062", "CAP-064", "CAP-065", "CAP-068",
	}
	classicContractPhase200Capabilities = []string{
		"CAP-005", "CAP-010", "CAP-011", "CAP-012", "CAP-056", "CAP-061", "CAP-069",
	}
	classicContractPhase201Capabilities = []string{
		"CAP-003", "CAP-004", "CAP-022", "CAP-024", "CAP-029", "CAP-051", "CAP-066", "CAP-071",
	}
	classicContractCaseIDPattern          = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
	classicContractDecisionPattern        = regexp.MustCompile(`^SYN-(?:199-(?:0[1-9]|10)|200-(?:0[1-9]|1[0-2])|201-(?:0[1-9]|1[0-4]))$`)
	classicContractCAPPattern             = regexp.MustCompile(`^CAP-[0-9]{3}$`)
	classicContractRoundVocabulary        = regexp.MustCompile(`(?i)\brounds?\b`)
	classicContractPhase200PublicCommands = map[string]bool{
		"/ant-discuss": true, "/ant-insert-phase": true, "/ant-plan": true, "/ant-spec": true,
		"aether discuss": true, "aether spec": true, "aether spec --repair-projection": true,
		"aether plan": true, "aether plan --candidate": true, "aether plan --refresh": true,
		"aether plan --accept-candidate <candidate-id>": true, "aether build <phase>": true,
		"aether run": true,
	}
	classicContractPhase201PublicCommands = map[string]bool{
		"/ant-build": true, "/ant-continue": true, "/ant-run": true, "/ant-status": true, "/ant-quick": true,
	}
	classicContractPhase200MechanismProofs = []classicPhase200MechanismProofExpectation{
		{
			CaseID: "phase200.mechanism.specification-canonical-recomputation", MechanismID: "SYN-200-07",
			PublicCommand: "aether spec", RecoveryCommand: "aether spec",
			GoTestSymbol: "TestSpecificationIntegrity200",
		},
		{
			CaseID: "phase200.mechanism.candidate-semantic-binding", MechanismID: "SYN-200-06",
			PublicCommand: "aether plan --candidate", RecoveryCommand: "aether plan --refresh",
			GoTestSymbol: "TestPlanCandidateSemanticIntegrity200CopiedHashesRejectEveryReviewMutation",
		},
		{
			CaseID: "phase200.mechanism.physical-containment", MechanismID: "SYN-200-01",
			PublicCommand: "aether plan", RecoveryCommand: "aether plan",
			GoTestSymbol: "TestRepositoryBootstrapContainment200",
		},
		{
			CaseID: "phase200.mechanism.repository-mutation-session", MechanismID: "SYN-200-01",
			PublicCommand: "aether plan", RecoveryCommand: "aether plan --refresh",
			GoTestSymbol: "TestPlanningMutationSession200",
		},
		{
			CaseID: "phase200.mechanism.atomic-build-start", MechanismID: "SYN-200-01",
			PublicCommand: "aether build <phase>", RecoveryCommand: "aether build <phase>",
			GoTestSymbol: "TestBuildStartTransaction200ExactReplayIsReadOnly",
		},
		{
			CaseID: "phase200.mechanism.candidate-expiry-recovery", MechanismID: "SYN-200-06",
			PublicCommand: "aether plan --accept-candidate <candidate-id>", RecoveryCommand: "aether plan --refresh",
			GoTestSymbol: "TestPlanCandidateExpiry200AcceptanceBoundaryIsAtomic",
		},
	}
)

type classicPhase200MechanismProofExpectation struct {
	CaseID          string
	MechanismID     string
	PublicCommand   string
	RecoveryCommand string
	GoTestSymbol    string
}

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
	Phase200Proof     *classicPhase200Proof      `json:"phase200_proof,omitempty"`
}

type classicPhase200Proof struct {
	Class                   string                         `json:"class"`
	Scenario                string                         `json:"scenario"`
	EvidenceThatWouldChange []string                       `json:"evidence_that_would_change"`
	Recommendation          *classicPhase200Recommendation `json:"recommendation,omitempty"`
	Assertions              classicPhase200Assertions      `json:"assertions"`
}

type classicPhase200Recommendation struct {
	Disposition string   `json:"disposition"`
	Rationale   string   `json:"rationale"`
	EvidenceIDs []string `json:"evidence_ids"`
	Producer    string   `json:"producer"`
}

type classicPhase200Assertions struct {
	StateHashRelation     string   `json:"state_hash_relation"`
	ArtifactSetRelation   string   `json:"artifact_set_relation"`
	DispatchDelta         int      `json:"dispatch_delta"`
	ReceiptChainRelation  string   `json:"receipt_chain_relation"`
	RequiredResultFields  []string `json:"required_result_fields"`
	ProhibitedSideEffects []string `json:"prohibited_side_effects"`
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
	SchemaVersion    string             `json:"schema_version"`
	SynthesisSource  string             `json:"synthesis_source"`
	SynthesisSources []string           `json:"synthesis_sources,omitempty"`
	Mechanisms       []classicMechanism `json:"mechanisms"`
}

type classicMechanism struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Disposition     string   `json:"disposition"`
	CAPIDs          []string `json:"cap_ids"`
	PublicCommands  []string `json:"public_commands"`
	SourceCitations []string `json:"source_citations"`
	Groups          []string `json:"groups"`
	SourceAnchors   []string `json:"source_anchors,omitempty"`
	ModernInvariant string   `json:"modern_invariant,omitempty"`
	PositiveCaseIDs []string `json:"positive_case_ids,omitempty"`
	NegativeCaseIDs []string `json:"negative_case_ids,omitempty"`
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
		invalid.Mechanisms = classicMechanismsWithoutDecision(invalid.Mechanisms, "SYN-199-10")
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
	assertClassicSynthesisSource(t, repoRoot, registry.SynthesisSource, classicContractPhase199Decisions, classicContractPhase199Capabilities, true)
	if !slices.Equal(registry.SynthesisSources, []string{
		".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md",
		".planning/phases/200-iterative-planning/200-CLASSIC-SYNTHESIS.md",
		".planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md",
	}) {
		t.Fatalf("synthesis_sources = %v, want the ordered Phase 199, Phase 200, and Phase 201 sources", registry.SynthesisSources)
	}
	assertClassicSynthesisSource(t, repoRoot, registry.SynthesisSources[1], classicContractPhase200Decisions, classicContractPhase200Capabilities, false)
	assertClassicSynthesisSource(t, repoRoot, registry.SynthesisSources[2], classicContractPhase201Decisions, classicContractPhase201Capabilities, false)

	after := snapshotStoreFileHashesForTest(t, contractDir)
	assertHashSnapshotsEqualForTest(t, "Classic mechanism validation", before, after)
}

// TestClassicContractPhase201MechanismRegistry proves the widened corpus
// carries the fourteen SYN-201 work-cycle mechanism identities and that an
// unregistered identifier -- at the registry layer or referenced from a case
// -- is refused by its own name rather than silently accepted or accepted
// with a generic error. Plan 201-14 owns the Phase 201 executable case set;
// this test asserts registry shape only, per the plan's own scope note.
func TestClassicContractPhase201MechanismRegistry(t *testing.T) {
	contractDir := classicContractFixtureDir(t)
	before := snapshotStoreFileHashesForTest(t, contractDir)
	registry, err := loadClassicMechanismRegistry(filepath.Join(contractDir, "mechanisms.json"))
	if err != nil {
		t.Fatalf("load Classic mechanism registry: %v", err)
	}

	decisionCounts := make(map[string]int, len(classicContractPhase201Decisions))
	for _, mechanism := range registry.Mechanisms {
		if !strings.HasPrefix(mechanism.ID, "SYN-201-") {
			continue
		}
		decisionCounts[mechanism.ID]++
		if len(mechanism.CAPIDs) == 0 {
			t.Errorf("mechanism %q requires at least one CAP identifier", mechanism.ID)
		}
		if len(mechanism.SourceCitations) == 0 || classicStringsContainBlank(mechanism.SourceCitations) {
			t.Errorf("mechanism %q requires at least one source citation", mechanism.ID)
		}
		if len(mechanism.PublicCommands) == 0 || classicStringsContainBlank(mechanism.PublicCommands) {
			t.Errorf("mechanism %q requires at least one public command", mechanism.ID)
		}
		if len(mechanism.Groups) == 0 {
			t.Errorf("mechanism %q requires at least one owning group", mechanism.ID)
		}
	}
	for _, id := range classicContractPhase201Decisions {
		if decisionCounts[id] != 1 {
			t.Errorf("mechanism %s count = %d, want exactly 1", id, decisionCounts[id])
		}
	}

	t.Run("unregistered SYN-201 identifier is refused by name", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms = classicMechanismsWithoutDecision(invalid.Mechanisms, "SYN-201-07")
		assertClassicMechanismError(t, invalid, `missing synthesis decision "SYN-201-07"`)
	})

	t.Run("case referencing an unregistered decision fails coverage validation by name", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms = classicMechanismsWithoutDecision(invalid.Mechanisms, "SYN-201-07")
		testCase := classicContractValidDocument().Cases[0]
		testCase.ID = "work-cycle.probe.unregistered-decision"
		testCase.SynthesisDecision = "SYN-201-07"
		err := validateClassicCaseSynthesisCoverage([]classicContractCase{testCase}, invalid)
		if err == nil || !strings.Contains(err.Error(), `"SYN-201-07"`) {
			t.Fatalf("coverage validation error = %v, want it to name SYN-201-07", err)
		}
	})

	t.Run("SYN-201-07 case in group V-201-OUTCOME validates against the widened schema and passes coverage", func(t *testing.T) {
		testCase := classicContractValidDocument().Cases[0]
		testCase.ID = "work-cycle.probe.registered-decision"
		testCase.Group = "V-201-OUTCOME"
		testCase.SynthesisDecision = "SYN-201-07"
		document := classicContractDocument{SchemaVersion: classicContractSchemaVersion, Cases: []classicContractCase{testCase}}
		if err := validateClassicContractDocument(document); err != nil {
			t.Fatalf("schema validation error = %v, want nil for SYN-201-07 in V-201-OUTCOME", err)
		}
		if err := validateClassicCaseSynthesisCoverage(document.Cases, registry); err != nil {
			t.Fatalf("coverage validation error = %v, want nil for a registered decision", err)
		}
	})

	after := snapshotStoreFileHashesForTest(t, contractDir)
	assertHashSnapshotsEqualForTest(t, "Classic Phase 201 mechanism registry validation", before, after)
}

func TestClassicContractPhase200SchemaMechanismsAndCausalCases(t *testing.T) {
	contractDir := classicContractFixtureDir(t)

	schema := loadClassicContractJSONSchema(t, filepath.Join(contractDir, "schema.json"))
	caseDefinition := classicSchemaObject(t, schema.Definitions["case"])
	caseProperties, ok := caseDefinition["properties"].(map[string]any)
	if !ok {
		t.Fatal("Classic case schema has no properties object")
	}
	if _, ok := caseProperties["phase200_proof"]; !ok {
		t.Error("Classic case schema is missing phase200_proof")
	}
	mechanismDefinition := classicSchemaObject(t, schema.Definitions["mechanism"])
	mechanismProperties, ok := mechanismDefinition["properties"].(map[string]any)
	if !ok {
		t.Fatal("Classic mechanism schema has no properties object")
	}
	for _, field := range []string{"source_anchors", "modern_invariant", "positive_case_ids", "negative_case_ids"} {
		if _, ok := mechanismProperties[field]; !ok {
			t.Errorf("Classic mechanism schema is missing %s", field)
		}
	}

	registryBytes, err := os.ReadFile(filepath.Join(contractDir, "mechanisms.json"))
	if err != nil {
		t.Fatal(err)
	}
	var registryRaw struct {
		Mechanisms []map[string]any `json:"mechanisms"`
	}
	if err := json.Unmarshal(registryBytes, &registryRaw); err != nil {
		t.Fatal(err)
	}
	decisionCounts := map[string]int{}
	for _, mechanism := range registryRaw.Mechanisms {
		id, _ := mechanism["id"].(string)
		decisionCounts[id]++
		if !strings.HasPrefix(id, "SYN-200-") {
			continue
		}
		for _, field := range []string{"source_anchors", "modern_invariant", "positive_case_ids", "negative_case_ids"} {
			if value, exists := mechanism[field]; !exists || value == nil {
				t.Errorf("mechanism %s missing %s", id, field)
			}
		}
	}
	for index := 1; index <= 12; index++ {
		id := fmt.Sprintf("SYN-200-%02d", index)
		if decisionCounts[id] != 1 {
			t.Errorf("mechanism %s count = %d, want exactly 1", id, decisionCounts[id])
		}
	}

	caseBytes, err := os.ReadFile(filepath.Join(contractDir, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var casesRaw struct {
		Cases []map[string]any `json:"cases"`
	}
	if err := json.Unmarshal(caseBytes, &casesRaw); err != nil {
		t.Fatal(err)
	}
	wantGroups := []string{
		"V-200-E2E", "V-200-CONFIDENCE", "V-200-DECISION", "V-200-STOP",
		"V-200-ACCEPT", "V-200-REVISION", "V-200-REPLAY", "V-200-PLATFORM",
	}
	groupClasses := map[string]map[string]bool{}
	for _, testCase := range casesRaw.Cases {
		decision, _ := testCase["synthesis_decision"].(string)
		if !strings.HasPrefix(decision, "SYN-200-") {
			continue
		}
		group, _ := testCase["group"].(string)
		proof, ok := testCase["phase200_proof"].(map[string]any)
		if !ok {
			t.Errorf("Phase 200 case %v is missing phase200_proof", testCase["id"])
			continue
		}
		class, _ := proof["class"].(string)
		if groupClasses[group] == nil {
			groupClasses[group] = map[string]bool{}
		}
		groupClasses[group][class] = true
	}
	for _, group := range wantGroups {
		if !groupClasses[group]["success"] || !groupClasses[group]["refusal"] {
			t.Errorf("proof group %s requires executable success and refusal cases", group)
		}
	}

	document, err := loadClassicContractDocument(filepath.Join(contractDir, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := loadClassicMechanismRegistry(filepath.Join(contractDir, "mechanisms.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateClassicPhase200Corpus(document, registry); err != nil {
		t.Fatal(err)
	}

	t.Run("missing mechanism", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms = classicMechanismsWithoutDecision(invalid.Mechanisms, "SYN-200-12")
		assertClassicMechanismError(t, invalid, `missing synthesis decision "SYN-200-12"`)
	})
	t.Run("missing synthesis source", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.SynthesisSources = invalid.SynthesisSources[:1]
		assertClassicMechanismError(t, invalid, "unexpected synthesis_sources")
	})
	t.Run("missing Phase 200 CAP link", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		for index := range invalid.Mechanisms {
			if invalid.Mechanisms[index].ID == "SYN-200-01" {
				invalid.Mechanisms[index].CAPIDs = nil
			}
		}
		assertClassicMechanismError(t, invalid, `mechanism "SYN-200-01" requires cap_ids`)
	})
	t.Run("missing executable case link", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		for index := range invalid.Mechanisms {
			if invalid.Mechanisms[index].ID == "SYN-200-03" {
				invalid.Mechanisms[index].PositiveCaseIDs = nil
			}
		}
		assertClassicMechanismError(t, invalid, `mechanism "SYN-200-03" requires positive_case_ids`)
	})
	t.Run("missing referenced case", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		invalid.Cases = classicContractWithoutExactCase(invalid.Cases, "phase200.confidence.fresh-evidence")
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("has %d cases", 15+len(classicContractPhase200MechanismProofs))) {
			t.Fatalf("validation error = %v, want missing referenced case", err)
		}
	})
	t.Run("orphaned Phase 200 case", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		caseID := "phase200.e2e.approved-two-pass-journey"
		for index := range invalid.Mechanisms {
			invalid.Mechanisms[index].PositiveCaseIDs = classicStringsWithout(invalid.Mechanisms[index].PositiveCaseIDs, caseID)
			invalid.Mechanisms[index].NegativeCaseIDs = classicStringsWithout(invalid.Mechanisms[index].NegativeCaseIDs, caseID)
		}
		if err := validateClassicPhase200Corpus(document, invalid); err == nil || !strings.Contains(err.Error(), "not referenced") {
			t.Fatalf("validation error = %v, want orphaned-case refusal", err)
		}
	})
	t.Run("missing readable Specification category", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		removeClassicPhase200StructuredAssertion(t, &invalid, "phase200.e2e.approved-two-pass-journey", "specification.recovery_expectations")
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), "specification.recovery_expectations") {
			t.Fatalf("validation error = %v, want missing Specification category", err)
		}
	})
	t.Run("missing Queen recommendation authority", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		for index := range invalid.Cases {
			if invalid.Cases[index].ID == "phase200.accept.exact" {
				invalid.Cases[index].Phase200Proof.Recommendation.Producer = ""
			}
		}
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), "Queen recommendation") {
			t.Fatalf("validation error = %v, want recommendation authority refusal", err)
		}
	})
	t.Run("missing public stop label", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		removeClassicPhase200StructuredAssertion(t, &invalid, "phase200.stop.reasoned-matrix", "stalled=stall detected")
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), "stall detected") {
			t.Fatalf("validation error = %v, want missing public stop label", err)
		}
	})
	t.Run("missing post-card decision boundary", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		removeClassicPhase200StructuredAssertion(t, &invalid, "phase200.decision.ordered-material-boundaries", "boundary_card_hash")
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), "boundary_card_hash") {
			t.Fatalf("validation error = %v, want post-card boundary refusal", err)
		}
	})
}

func TestClassicContractPhase200IntegratedMechanismProofs(t *testing.T) {
	contractDir := classicContractFixtureDir(t)
	document, err := loadClassicContractDocument(filepath.Join(contractDir, "cases.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := loadClassicMechanismRegistry(filepath.Join(contractDir, "mechanisms.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateClassicPhase200Corpus(document, registry); err != nil {
		t.Fatal(err)
	}

	t.Run("missing executable proof", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		removeClassicPhase200MechanismProofField(t, &invalid, classicContractPhase200MechanismProofs[0].CaseID, "go_test_symbol")
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), "go_test_symbol") {
			t.Fatalf("validation error = %v, want missing executable proof refusal", err)
		}
	})

	t.Run("stale public command vocabulary", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		for index := range invalid.Mechanisms {
			if invalid.Mechanisms[index].ID == "SYN-200-07" {
				invalid.Mechanisms[index].PublicCommands = append(invalid.Mechanisms[index].PublicCommands, "aether plan-finalize")
			}
		}
		assertClassicMechanismError(t, invalid, `stale or non-public command "aether plan-finalize"`)
	})

	t.Run("noncausal source-only row", func(t *testing.T) {
		invalid := cloneClassicContractDocument(t, document)
		for index := range invalid.Cases {
			if invalid.Cases[index].ID == classicContractPhase200MechanismProofs[0].CaseID {
				invalid.Cases[index].SourceCitations = []string{"cmd/spec_cmd.go"}
			}
		}
		if err := validateClassicPhase200Corpus(invalid, registry); err == nil || !strings.Contains(err.Error(), "resolvable Go test symbol") {
			t.Fatalf("validation error = %v, want source-only refusal", err)
		}
	})

	t.Run("duplicate Phase 200 mechanism ID", func(t *testing.T) {
		invalid := cloneClassicMechanismRegistry(t, registry)
		for _, mechanism := range invalid.Mechanisms {
			if mechanism.ID == "SYN-200-07" {
				invalid.Mechanisms = append(invalid.Mechanisms, mechanism)
				break
			}
		}
		assertClassicMechanismError(t, invalid, `duplicate synthesis decision "SYN-200-07"`)
	})
}

func TestClassicContractPhase200CausalExecution(t *testing.T) {
	document := loadClassicContractCorpus(t)
	executed := 0
	for _, testCase := range document.Cases {
		if testCase.Phase200Proof == nil {
			continue
		}
		executed++
		t.Run(testCase.ID, func(t *testing.T) {
			if executeClassicPhase200GoTestProof(t, testCase) {
				return
			}
			execution := executeClassicPhase200Scenario(t, testCase.Phase200Proof.Scenario)
			assertClassicPhase200Execution(t, testCase, execution)
		})
	}
	want := 16 + len(classicContractPhase200MechanismProofs)
	if executed != want {
		t.Fatalf("executed %d Phase 200 cases, want %d", executed, want)
	}
}

func executeClassicPhase200GoTestProof(t *testing.T, testCase classicContractCase) bool {
	t.Helper()
	raw, ok := testCase.Expected.SemanticFields["go_test_symbol"]
	if !ok {
		return false
	}
	var symbol string
	if err := json.Unmarshal(raw, &symbol); err != nil || strings.TrimSpace(symbol) == "" {
		t.Fatalf("%s has invalid go_test_symbol: %v", testCase.ID, err)
	}
	command := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(symbol)+"$", "-test.count=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s causal Go proof %s failed: %v\n%s", testCase.ID, symbol, err, output)
	}
	return true
}

type classicPhase200Snapshot struct {
	StateHash  string
	Artifacts  []string
	Dispatches int
	Receipts   []string
}

type classicPhase200Execution struct {
	Before classicPhase200Snapshot
	After  classicPhase200Snapshot
	Result map[string]any
}

func assertClassicPhase200Execution(t *testing.T, testCase classicContractCase, execution classicPhase200Execution) {
	t.Helper()
	proof := testCase.Phase200Proof
	assertRelation := func(label, want string, equal bool) {
		t.Helper()
		changed := !equal
		if (want == "changed") != changed {
			t.Errorf("%s relation = changed:%t, want %s", label, changed, want)
		}
	}
	assertRelation("state hash", proof.Assertions.StateHashRelation, execution.Before.StateHash == execution.After.StateHash)
	assertRelation("artifact set", proof.Assertions.ArtifactSetRelation, slices.Equal(execution.Before.Artifacts, execution.After.Artifacts))
	assertRelation("receipt chain", proof.Assertions.ReceiptChainRelation, slices.Equal(execution.Before.Receipts, execution.After.Receipts))
	if delta := execution.After.Dispatches - execution.Before.Dispatches; delta != proof.Assertions.DispatchDelta {
		t.Errorf("dispatch delta = %d, want %d", delta, proof.Assertions.DispatchDelta)
	}
	for _, field := range proof.Assertions.RequiredResultFields {
		value, ok := execution.Result[field]
		if !ok || value == nil || strings.TrimSpace(fmt.Sprint(value)) == "" {
			t.Errorf("structured execution result is missing %q", field)
		}
	}
	if proof.Class == "refusal" && proof.Assertions.StateHashRelation == "unchanged" && execution.Before.StateHash != execution.After.StateHash {
		t.Error("refusal changed canonical state")
	}
}

func snapshotClassicPhase200Repository(t *testing.T, root string) classicPhase200Snapshot {
	t.Helper()
	snapshot := classicPhase200Snapshot{StateHash: "absent"}
	statePath := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
	if data, err := os.ReadFile(statePath); err == nil {
		sum := sha256.Sum256(data)
		snapshot.StateHash = fmt.Sprintf("sha256:%x", sum)
	} else if !os.IsNotExist(err) {
		t.Fatalf("read canonical state: %v", err)
	}
	dataRoot := filepath.Join(root, ".aether", "data")
	if _, err := os.Stat(dataRoot); os.IsNotExist(err) {
		return snapshot
	}
	if err := filepath.WalkDir(dataRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		snapshot.Artifacts = append(snapshot.Artifacts, rel)
		lower := strings.ToLower(rel)
		if strings.Contains(lower, "dispatch") || strings.Contains(lower, "authorizations/") {
			snapshot.Dispatches++
		}
		if strings.Contains(lower, "receipt") || strings.HasSuffix(lower, "/acceptance.json") {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			snapshot.Receipts = append(snapshot.Receipts, rel+":"+fmt.Sprintf("%x", sum))
		}
		return nil
	}); err != nil {
		t.Fatalf("snapshot Phase 200 repository: %v", err)
	}
	sort.Strings(snapshot.Artifacts)
	sort.Strings(snapshot.Receipts)
	return snapshot
}

func classicPhase200ExecutionAround(t *testing.T, root string, action func() map[string]any) classicPhase200Execution {
	t.Helper()
	before := snapshotClassicPhase200Repository(t, root)
	result := action()
	after := snapshotClassicPhase200Repository(t, root)
	return classicPhase200Execution{Before: before, After: after, Result: result}
}

func executeClassicPhase200Scenario(t *testing.T, scenario string) classicPhase200Execution {
	t.Helper()
	switch scenario {
	case "approved-two-pass-journey":
		root, candidate := classicPhase200TwoPassCandidate(t)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			review, err := reviewPlanCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			if review.Recommendation.Producer != colony.PlanRecommendationProducerQueen || strings.TrimSpace(review.Recommendation.Rationale) == "" || len(review.Recommendation.EvidenceIDs) == 0 {
				t.Fatalf("candidate is missing the persisted typed Queen recommendation: %+v", review.Recommendation)
			}
			if len(review.Iterations) < 2 || review.Iterations[0].Iteration != 1 || review.Iterations[1].Iteration != 2 {
				t.Fatalf("candidate timeline = %+v, want two ordered complete passes", review.Iterations)
			}
			accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
				AcceptedBy: "owner:classic-contract", AcceptedAt: candidate.CreatedAt.Add(time.Minute),
			})
			if err != nil {
				t.Fatal(err)
			}
			return map[string]any{
				"specification": candidate.SpecificationRevisionID, "timeline": review.Timeline,
				"candidate": candidate.ID, "acceptance_receipt": accepted.Receipt.ID,
				"plan_authority": accepted.Revision.ID,
			}
		})

	case "draft-planning-refusal":
		root := newSpecificationTestRepository(t, colony.ColonyState{})
		if _, err := createSpecificationDraft(root, specificationTestDraftRequest(t, colony.SpecScopeWholeGoal), specificationMutationOptions{}); err != nil {
			t.Fatal(err)
		}
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			state := mustReadSpecificationTestState(t, root)
			_, err := requireApprovedPlanningSpecification(root, state)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "approved") {
				t.Fatalf("draft planning error = %v, want approved-Specification refusal", err)
			}
			return map[string]any{"error_class": "specification_not_approved", "state_effect": "unchanged", "recovery_command": "aether spec --approve"}
		})

	case "fresh-evidence-confidence":
		root, manifest, result := planningRouteStageTestFixture(t)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
			if err != nil {
				t.Fatal(err)
			}
			if len(coordinated.Route.Validation.Confidence.Assessments) != 5 || coordinated.ScoutDispatch == nil {
				t.Fatalf("fresh-evidence route result = %+v", coordinated)
			}
			for _, assessment := range coordinated.Route.Validation.Confidence.Assessments {
				if len(assessment.FreshEvidenceIDs) == 0 || strings.TrimSpace(assessment.RemainingGap.EvidenceThatWouldChange) == "" {
					t.Fatalf("assessment lacks causal evidence: %+v", assessment)
				}
			}
			return map[string]any{
				"dimension_assessments": coordinated.Route.Validation.Confidence.Assessments,
				"derived_overall":       coordinated.Route.Validation.Confidence.Scores.Overall,
				"weakest_gap":           coordinated.Route.Validation.Confidence.WeakestGap.ID,
				"iteration_card":        coordinated.Route.Card.ID, "route_receipt": coordinated.Route.Receipt.ID,
			}
		})

	case "restated-evidence-refusal":
		root, manifest, result := planningRouteStageTestFixture(t)
		supplied := 99
		result.SuppliedOverall = &supplied
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			_, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "overall") {
				t.Fatalf("supplied-overall error = %v", err)
			}
			return map[string]any{"error_class": "worker_supplied_overall", "rejected_evidence_ids": result.ProposalEvidenceIDs, "state_effect": "unchanged"}
		})

	case "ordered-material-boundaries":
		root, manifest, result := planningRouteStageMaterialFixture(t)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
			if err != nil {
				t.Fatal(err)
			}
			checkpoint := coordinated.DecisionCheckpoint
			if checkpoint == nil || coordinated.Candidate != nil || checkpoint.CompletedCardHash != coordinated.Route.Card.ContentHash || checkpoint.Batch.BoundaryCardHash != coordinated.Route.Card.ContentHash {
				t.Fatalf("material boundary was not ordered after the complete card: %+v", coordinated)
			}
			if len(checkpoint.Cards) == 0 || strings.TrimSpace(checkpoint.Cards[0].QueenRecommendation) == "" {
				t.Fatalf("decision checkpoint lacks Queen recommendation: %+v", checkpoint)
			}
			classicPhase200AssertDecisionBranches(t)
			return map[string]any{"iteration_card": coordinated.Route.Card.ID, "decision_checkpoint": checkpoint.ID, "queen_recommendation": checkpoint.Cards[0].QueenRecommendation, "boundary_card_hash": checkpoint.CompletedCardHash}
		})

	case "unanswered-decision-refusal":
		root, manifest, result := planningRouteStageMaterialFixture(t)
		coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
		if err != nil {
			t.Fatal(err)
		}
		checkpoint := coordinated.DecisionCheckpoint
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			_, err := buildPlanningScoutDecisionResumeToken(*checkpoint, nil)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "every card") {
				t.Fatalf("unanswered decision error = %v", err)
			}
			return map[string]any{"error_class": "decision_answers_incomplete", "decision_id": checkpoint.Cards[0].DecisionID, "state_effect": "unchanged", "exact_answer_command": checkpoint.ResumeToken.RecoveryCommand}
		})

	case "reasoned-stop-matrix":
		root, manifest, result := planningRouteStageTestFixture(t)
		planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
			if err != nil {
				t.Fatal(err)
			}
			if coordinated.Candidate == nil || coordinated.Candidate.Status != colony.PlanCandidatePendingReview {
				t.Fatalf("reasoned stop did not yield an inactive candidate: %+v", coordinated.Candidate)
			}
			labels := classicPhase200StopLabels(t)
			return map[string]any{"stop_reason": coordinated.Route.Card.Decision.Reason, "stop_reason_public_label": labels[coordinated.Route.Card.Decision.Reason], "residual_gaps": coordinated.Candidate.ResidualGaps, "candidate": coordinated.Candidate.ID, "queen_recommendation": coordinated.Candidate.Recommendation}
		})

	case "material-stop-refusal":
		root, manifest, result := planningRouteStageMaterialFixture(t)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
			if err != nil {
				t.Fatal(err)
			}
			if coordinated.Candidate != nil || coordinated.DecisionCheckpoint == nil {
				t.Fatalf("material override created a candidate: %+v", coordinated)
			}
			return map[string]any{"iteration_card": coordinated.Route.Card.ID, "material_gap": coordinated.Route.Validation.Confidence.WeakestGap.ID, "decision_checkpoint": coordinated.DecisionCheckpoint.ID, "candidate_eligible": false}
		})

	case "exact-candidate-acceptance":
		root, candidate := planCandidateTestPending(t)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			review, err := reviewPlanCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{AcceptedBy: "owner:classic-contract", AcceptedAt: time.Date(2026, time.September, 8, 1, 5, 0, 0, time.UTC)})
			if err != nil {
				t.Fatal(err)
			}
			return map[string]any{"candidate": candidate.ID, "queen_recommendation": review.Recommendation, "revision": accepted.Revision.ID, "acceptance_receipt": accepted.Receipt.ID, "plan_authority": accepted.Receipt.ActivatedPlanRevisionID}
		})

	case "stale-candidate-refusal":
		root, candidate := planCandidateTestPending(t)
		request := planCandidateTestAcceptanceRequest(candidate)
		request.TimelineDigest = strings.Repeat("2", 64)
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			_, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner"})
			if err == nil || !strings.Contains(err.Error(), "timeline_digest") {
				t.Fatalf("stale acceptance error = %v", err)
			}
			return map[string]any{"error_class": "stale_candidate_binding", "mismatched_binding": "timeline_digest", "state_effect": "unchanged", "fresh_review_command": "aether plan --candidate"}
		})

	case "scoped-spec-revision":
		return executeClassicPhase200ScopedRevision(t)

	case "unscoped-revision-refusal":
		return executeClassicPhase200UnscopedRevision(t)

	case "exact-replay":
		root, candidate := planCandidateTestPending(t)
		request := planCandidateTestAcceptanceRequest(candidate)
		first, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Date(2026, time.September, 8, 1, 10, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			replayed, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Date(2026, time.September, 8, 2, 10, 0, 0, time.UTC)})
			if err != nil || !replayed.Replayed || replayed.Receipt.ID != first.Receipt.ID || replayed.Revision.ID != first.Revision.ID {
				t.Fatalf("exact replay diverged: first=%+v replay=%+v err=%v", first, replayed, err)
			}
			return map[string]any{"replayed": true, "original_revision": replayed.Revision.ID, "original_acceptance_receipt": replayed.Receipt.ID, "crash_window_recovery": "covered-by-transaction-replay", "projection_repair": "authority-unchanged"}
		})

	case "divergent-replay-refusal":
		root, candidate := planCandidateTestPending(t)
		request := planCandidateTestAcceptanceRequest(candidate)
		first, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner", AcceptedAt: time.Date(2026, time.September, 8, 1, 15, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		request.AcceptanceToken += "-divergent"
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			_, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{AcceptedBy: "owner"})
			if err == nil || !strings.Contains(err.Error(), "acceptance_token") {
				t.Fatalf("divergent replay error = %v", err)
			}
			return map[string]any{"error_class": "divergent_replay", "divergent_field": "acceptance_token", "state_effect": "unchanged", "original_receipt": first.Receipt.ID}
		})

	case "platform-projection-parity":
		root := newSpecificationTestRepository(t, colony.ColonyState{})
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			t.Setenv("NO_COLOR", "1")
			for _, policy := range planningPresetPolicies {
				selection, err := resolvePlanningPreset(codexPlanOptions{Preset: string(policy.ID), PresetSet: true})
				if err != nil || selection.PresetRequired || selection.Policy != policy {
					t.Fatalf("preset %s did not resolve exactly: selection=%+v err=%v", policy.ID, selection, err)
				}
			}
			labels := classicPhase200StopLabels(t)
			widths := []int{47, 48, 63, 64, 95, 96}
			for _, width := range widths {
				output := renderPlanningSpecificationVisual(planningVisualSpecificationFixture(), planningVisualOptions{Width: width})
				if !strings.Contains(output, "Specification") {
					t.Fatalf("width %d abbreviated Specification:\n%s", width, output)
				}
			}
			projection := projectPlanningCandidate(planningVisualCandidateFixture())
			return map[string]any{"runtime_stop_enum": colony.PlanningStopTargetMet, "public_stop_label": labels[colony.PlanningStopTargetMet], "evidence_that_would_change": projection.EvidenceThatWouldChange, "queen_recommendation": projection.RecommendationDisposition, "platform_command_spelling": "aether plan|/ant-plan", "width_matrix": widths, "no_color": true}
		})

	case "wrapper-authority-refusal":
		root := newSpecificationTestRepository(t, colony.ColonyState{})
		return classicPhase200ExecutionAround(t, root, func() map[string]any {
			repoRoot := findTestModuleRoot(t)
			canonical, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "commands", "plan.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			for _, platform := range []string{".claude", ".opencode"} {
				path := filepath.Join(repoRoot, platform, "commands", "ant", "plan.md")
				projection, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(projection), "Aether-managed: runtime spec at .aether/commands/plan.yaml") {
					t.Fatalf("%s is not a managed projection", path)
				}
			}
			if !bytes.Contains(canonical, []byte("aether plan")) {
				t.Fatal("canonical plan source omits the runtime command")
			}
			return map[string]any{"canonical_source": ".aether/commands/plan.yaml", "managed_projection": true, "go_authority": true, "state_effect": "unchanged"}
		})
	default:
		t.Fatalf("no executable Phase 200 scenario for %q", scenario)
		return classicPhase200Execution{}
	}
}

func classicPhase200StopLabels(t *testing.T) map[colony.PlanningStopReason]string {
	t.Helper()
	want := map[colony.PlanningStopReason]string{
		colony.PlanningStopTargetMet: "target sufficiency", colony.PlanningStopDiminishingReturns: "diminishing returns",
		colony.PlanningStopStalledGap: "stall detected", colony.PlanningStopPassCap: "iteration cap",
	}
	for reason, label := range want {
		if got := planningStopReasonPublicLabel(reason); got != label {
			t.Fatalf("public label for %s = %q, want %q", reason, got, label)
		}
	}
	return want
}

func classicPhase200AssertDecisionBranches(t *testing.T) {
	t.Helper()
	for _, test := range []struct {
		name          string
		choiceID      string
		wantDirect    bool
		wantSuccessor bool
	}{
		{name: "equivalent direct resume", choiceID: "continue-research", wantDirect: true},
		{name: "contract change", choiceID: "proceed-with-risk", wantSuccessor: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, manifest, result := planningRouteStageMaterialFixture(t)
			coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
			if err != nil {
				t.Fatal(err)
			}
			card := coordinated.DecisionCheckpoint.Cards[0]
			resume, err := buildPlanningScoutDecisionResumeToken(*coordinated.DecisionCheckpoint, []planningScoutDecisionAnswer{{
				DecisionID: card.DecisionID, ChoiceID: test.choiceID, Answer: "Owner supplied an exact, evidence-bound answer.",
			}})
			if err != nil {
				t.Fatal(err)
			}
			resumed, err := resumePlanningRouteDecision(root, manifest.RunID, resume, time.Date(2026, time.September, 8, 1, 40, 0, 0, time.UTC))
			if err != nil {
				t.Fatal(err)
			}
			if test.wantDirect && (resumed.ScoutDispatch == nil || resumed.SuccessorSpecification != nil) {
				t.Fatalf("equivalent answer did not resume directly: %+v", resumed)
			}
			if test.wantSuccessor && (resumed.SuccessorSpecification == nil || resumed.SuccessorSpecification.Revision.Status != colony.SpecStatusDraft || resumed.ScoutDispatch != nil) {
				t.Fatalf("contract-changing answer did not create a successor DRAFT: %+v", resumed)
			}
		})
	}
}

func classicPhase200TwoPassCandidate(t *testing.T) (string, colony.PlanCandidate) {
	t.Helper()
	root, firstManifest, firstResult := planningRouteStageTestFixture(t)
	planningRouteStageSetPolicy(t, root, firstManifest.RunID, 99, 2)
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil || first.Candidate != nil || first.Route.Card.Iteration != 1 {
		t.Fatalf("first pass did not continue through a focused Scout dispatch: %+v", first)
	}
	scoutManifest := first.ScoutDispatch.Manifest
	if scoutManifest.WeakestGap == nil || scoutManifest.WeakestGap.ID != first.Route.Card.WeakestGap.ID {
		t.Fatalf("first pass weakest gap was not bound into the next Scout: card=%+v manifest=%+v", first.Route.Card.WeakestGap, scoutManifest.WeakestGap)
	}
	fresh := planningRouteStageEvidence(t, scoutManifest.Specification, scoutManifest.BasePlanRevisionID, "classic-second-pass", "Fresh second-pass evidence changes both confidence and the semantic route.", time.Date(2026, time.September, 8, 1, 30, 0, 0, time.UTC))
	scoutGap := planningRouteStageGap("classic-second-scout-gap", colony.PlanningDimensionRisks, fresh.Reference.ID, colony.PlanningGapNonMaterial, 12)
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings:          []planningScoutStageFinding{{StableID: "classic-second-finding", Summary: "The second pass resolves the first weakest gap and changes the route.", EvidenceIDs: []string{fresh.Reference.ID}}},
		NewEvidence:       []planningEvidenceRecord{fresh}, UnresolvedGaps: []colony.PlanningGap{scoutGap},
	}
	scoutCompleted, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatal(err)
	}
	if scoutCompleted.RouteDispatch == nil {
		t.Fatal("second Scout pass did not authorize Route-Setter")
	}
	routeManifest := scoutCompleted.RouteDispatch.Manifest
	proposal := planningRouteStageCloneResult(t, firstResult).Proposal
	proposal.Phases[0].Description = "Finalize the improved Route proposal after second-pass evidence"
	proposal.Phases[0].Tasks[0].Goal = "Validate the second-pass Route proposal"
	secondResult := planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: routeManifest.Specification, BasePlanRevisionID: routeManifest.BasePlanRevisionID, BasePlanRevisionHash: routeManifest.BasePlanRevisionHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: proposal, ProposalEvidenceIDs: []string{fresh.Reference.ID},
	}
	for index, dimension := range colony.PlanningDimensions() {
		before := first.Route.Card.DimensionAssessments[index].After
		secondResult.DimensionAssessments = append(secondResult.DimensionAssessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: "classic-second-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('k' + index))),
			Dimension: dimension, Before: before, After: min(before+12, 98), FreshEvidenceIDs: []string{fresh.Reference.ID},
			RemainingGap: planningRouteStageGap("classic-second-gap-"+string(dimension), dimension, fresh.Reference.ID, colony.PlanningGapNonMaterial, 5+index),
			Rationale:    "Fresh applicable second-pass evidence supports this movement.", ProducerReceiptID: routeManifest.ID,
		})
	}
	second, err := coordinatePlanningRouteStage(root, routeManifest, planningRouteStageTestBytes(t, secondResult))
	if err != nil {
		t.Fatal(err)
	}
	if second.Candidate == nil || second.Route.Card.Iteration != 2 || second.Route.Card.Decision.Reason != colony.PlanningStopPassCap {
		t.Fatalf("second pass did not produce the reasoned pending candidate: %+v", second)
	}
	if second.Candidate.Status != colony.PlanCandidatePendingReview || second.Candidate.Acceptance != nil {
		t.Fatalf("candidate became active without exact acceptance: %+v", second.Candidate)
	}
	return root, *second.Candidate
}

func executeClassicPhase200ScopedRevision(t *testing.T) classicPhase200Execution {
	t.Helper()
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	draftRequest := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draft, err := createSpecificationDraft(root, draftRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	requirementID := draft.Revision.Requirements[0].ID
	request := specificationRevisionRequest{
		PredecessorRevisionID: draft.Revision.ID, PredecessorContentHash: draft.Revision.ContentHash,
		Scope: colony.SpecScope{
			Kind: colony.SpecScopeFeature, GoalID: draft.Revision.Scope.GoalID, SessionID: "session-classic-scoped",
			FeatureID: "feature-visible-planning", RequirementIDs: []string{requirementID}, AcceptanceCheckIDs: []string{draft.Revision.AcceptanceChecks[0].ID},
		},
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeModify, Section: specificationSectionRequirements, TargetID: requirementID,
			Item: specificationItemInput{Description: "The owner sees a causal, evidence-backed planning loop.", EvidenceIDs: []string{"evidence:feature-change"}},
		}},
		DecisionResolution: func() *planningDecisionResolution {
			decision := specificationTestSuccessorDecision(requirementID)
			return &decision
		}(),
		CreatedAt: time.Date(2026, time.September, 8, 1, 20, 0, 0, time.UTC),
	}
	return classicPhase200ExecutionAround(t, root, func() map[string]any {
		revised, err := reviseSpecification(root, request, specificationMutationOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if revised.Revision.Status != colony.SpecStatusDraft || !slices.Equal(revised.AffectedScope.TaskIDs, []string{"1.1"}) {
			t.Fatalf("scoped revision affected the wrong authority: %+v", revised)
		}
		return map[string]any{
			"successor_specification": revised.Revision.ID, "classified_delta": revised.Revision.Delta,
			"affected_scope": revised.AffectedScope.TaskIDs, "preserved_scope": []string{"1.2"}, "reconciliation_required": true,
		}
	})
}

func executeClassicPhase200UnscopedRevision(t *testing.T) classicPhase200Execution {
	t.Helper()
	root := newSpecificationTestRepository(t, specificationTestPlanState())
	draftRequest := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draftRequest.Requirements = append(draftRequest.Requirements, specificationItemInput{
		Lineage: "independent-export", Description: "Export an independent report.", EvidenceIDs: []string{"charter:export"},
	})
	draft, err := createSpecificationDraft(root, draftRequest, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	outsideID, err := specificationStableID(specificationSectionRequirements, "independent-export")
	if err != nil {
		t.Fatal(err)
	}
	request := specificationRevisionRequest{
		PredecessorRevisionID: draft.Revision.ID, PredecessorContentHash: draft.Revision.ContentHash,
		Scope: colony.SpecScope{
			Kind: colony.SpecScopeFeature, GoalID: draft.Revision.Scope.GoalID, SessionID: "session-classic-unscoped",
			FeatureID: "feature-visible-planning", RequirementIDs: []string{draft.Revision.Requirements[0].ID}, AcceptanceCheckIDs: []string{draft.Revision.AcceptanceChecks[0].ID},
		},
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeModify, Section: specificationSectionRequirements, TargetID: outsideID,
			Item: specificationItemInput{Description: "Change an independent export.", EvidenceIDs: []string{"owner:unscoped"}},
		}},
		CreatedAt: time.Date(2026, time.September, 8, 1, 25, 0, 0, time.UTC),
	}
	return classicPhase200ExecutionAround(t, root, func() map[string]any {
		_, err := reviseSpecification(root, request, specificationMutationOptions{})
		if err == nil || !strings.Contains(err.Error(), "outside feature scope") {
			t.Fatalf("unscoped revision error = %v", err)
		}
		return map[string]any{"error_class": "outside_feature_scope", "outside_scope_id": outsideID, "state_effect": "unchanged", "recovery_command": "aether spec --revise --scope whole-goal"}
	})
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

// classicContractPhase201RequiredNegativeCaseIDs are the four rejected
// shortcuts Task 1 of 201-15 requires: a second automatic repair attempt, a
// caste dispatched at both the build and check boundaries, a timing segment
// with no instrumentation source, and a non-success card borrowing the
// success verdict's own wording. Each ID must exist as a case in the corpus.
var classicContractPhase201RequiredNegativeCaseIDs = []string{
	"work-cycle.repair.no-second-automatic-attempt",
	"work-cycle.boundary.no-double-caste-dispatch",
	"work-cycle.telemetry.no-segment-without-source",
	"work-cycle.outcome.no-success-token-on-non-success",
}

// TestClassicContractPhase201Cases is Task 1's own structural proof: every
// registered SYN-201 decision has at least one case, every Phase 201 group
// has at least one case, every added case carries a causal (state or
// forbidden-artifact) assertion in addition to its semantic assertions, the
// four required negative cases are present by name, and a case asserting
// only rendered text fails schema validation. It never mutates the corpus
// or mechanism files it reads (proved by a before-and-after hash snapshot).
func TestClassicContractPhase201Cases(t *testing.T) {
	contractDir := classicContractFixtureDir(t)
	before := snapshotStoreFileHashesForTest(t, contractDir)

	document := loadClassicContractCorpus(t)

	decisionCounts := make(map[string]int, len(classicContractPhase201Decisions))
	groupCounts := make(map[string]int, len(classicContractPhase201Groups))
	present := make(map[string]bool)
	var phase201Cases []classicContractCase
	for _, testCase := range document.Cases {
		if !strings.HasPrefix(testCase.SynthesisDecision, "SYN-201-") {
			continue
		}
		phase201Cases = append(phase201Cases, testCase)
		present[testCase.ID] = true
		decisionCounts[testCase.SynthesisDecision]++
		groupCounts[testCase.Group]++
		if !slices.Contains(classicContractPhase201Groups, testCase.Group) {
			t.Errorf("case %q has group %q, want one of the six Phase 201 groups", testCase.ID, testCase.Group)
		}
		if len(testCase.Expected.StateAssertions) == 0 && len(testCase.Expected.ForbiddenArtifacts) == 0 {
			t.Errorf("case %q lacks a causal state or forbidden-artifact assertion", testCase.ID)
		}
	}

	if len(phase201Cases) == 0 {
		t.Fatal("no Phase 201 cases were added to the corpus")
	}
	for _, id := range classicContractPhase201Decisions {
		if decisionCounts[id] == 0 {
			t.Errorf("SYN-201 decision %q has no case", id)
		}
	}
	for _, group := range classicContractPhase201Groups {
		if groupCounts[group] == 0 {
			t.Errorf("Phase 201 group %q has no case", group)
		}
	}
	for _, id := range classicContractPhase201RequiredNegativeCaseIDs {
		if !present[id] {
			t.Errorf("required negative case %q is missing", id)
		}
	}

	t.Run("a case asserting only rendered text fails schema validation", func(t *testing.T) {
		textOnly := phase201Cases[0]
		textOnly.Expected.StateAssertions = nil
		textOnly.Expected.ForbiddenArtifacts = nil
		invalidDoc := classicContractDocument{SchemaVersion: classicContractSchemaVersion, Cases: []classicContractCase{textOnly}}
		if err := validateClassicContractDocument(invalidDoc); err == nil || !strings.Contains(err.Error(), "missing causal state_assertions or forbidden_artifacts") {
			t.Fatalf("validation error = %v, want a text-only refusal", err)
		}
	})

	after := snapshotStoreFileHashesForTest(t, contractDir)
	assertHashSnapshotsEqualForTest(t, "Classic Phase 201 case validation", before, after)
}

// classicPhase201ProbeFailEnv is set by the broken-case subtest below on the
// spawned subprocess only, so TestClassicContractPhase201IntentionallyFailingProof
// stays a no-op (skipped) on every ordinary `go test` invocation and can
// never fail the package on its own.
const classicPhase201ProbeFailEnv = "AETHER_CLASSIC_PHASE201_PROBE_FAIL"

// TestClassicContractPhase201IntentionallyFailingProof exists solely so
// TestClassicContractPhase201CausalExecution's broken-case subtest has a
// real, deterministically-failing Go test symbol to spawn and observe --
// without it, the causal-execution harness's "surfaces the command, exit
// code, and captured output" behavior would go unproven. It is a no-op
// unless explicitly asked to fail via classicPhase201ProbeFailEnv.
func TestClassicContractPhase201IntentionallyFailingProof(t *testing.T) {
	if os.Getenv(classicPhase201ProbeFailEnv) != "1" {
		t.Skip("only fails when explicitly invoked as the Phase 201 broken-case proof")
	}
	t.Fatal("intentional failure: proves the Phase 201 causal-execution harness surfaces a broken case's command, exit code, and captured output")
}

// TestClassicContractPhase201CausalExecution is Task 2's execution proof,
// following the structure of TestClassicContractPhase200CausalExecution: it
// loads the corpus through the strict loader, selects every case naming a
// SYN-201 decision, and executes each one's go_test_symbol proof through
// executeClassicPhase200GoTestProof -- the same causal-Go-proof mechanism
// Phase 200 already uses, generalized here by that function's own
// semantic_fields-driven (not phase200_proof-gated) design. A SYN-201
// decision or a Phase 201 group with zero executing cases fails by name
// rather than passing vacuously.
func TestClassicContractPhase201CausalExecution(t *testing.T) {
	document := loadClassicContractCorpus(t)
	executedDecisions := make(map[string]int, len(classicContractPhase201Decisions))
	executedGroups := make(map[string]int, len(classicContractPhase201Groups))
	executed := 0
	for _, testCase := range document.Cases {
		if !strings.HasPrefix(testCase.SynthesisDecision, "SYN-201-") {
			continue
		}
		executed++
		executedDecisions[testCase.SynthesisDecision]++
		executedGroups[testCase.Group]++
		t.Run(testCase.ID, func(t *testing.T) {
			if !executeClassicPhase200GoTestProof(t, testCase) {
				t.Fatalf("%s has no executable go_test_symbol proof", testCase.ID)
			}
		})
	}
	if executed == 0 {
		t.Fatal("no Phase 201 cases were executed")
	}
	for _, id := range classicContractPhase201Decisions {
		if executedDecisions[id] == 0 {
			t.Errorf("SYN-201 decision %q has no executing case", id)
		}
	}
	for _, group := range classicContractPhase201Groups {
		if executedGroups[group] == 0 {
			t.Errorf("Phase 201 group %q has no executing case", group)
		}
	}

	t.Run("a broken case's command, exit code, and captured output are surfaced", func(t *testing.T) {
		symbol := "TestClassicContractPhase201IntentionallyFailingProof"
		command := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(symbol)+"$", "-test.count=1")
		command.Env = append(append([]string(nil), os.Environ()...), classicPhase201ProbeFailEnv+"=1")
		output, err := command.CombinedOutput()
		if err == nil {
			t.Fatalf("expected %s to fail under %s=1, but the subprocess exited cleanly:\n%s", symbol, classicPhase201ProbeFailEnv, output)
		}
		exitCode := -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
		if exitCode == 0 {
			t.Fatalf("expected a nonzero exit code from %s, got %d", symbol, exitCode)
		}
		if !strings.Contains(string(output), symbol) {
			t.Fatalf("captured output does not name the failing proof %s:\n%s", symbol, output)
		}
		if !strings.Contains(string(output), "intentional failure") {
			t.Fatalf("captured output does not carry the failure reason:\n%s", output)
		}
	})
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
	t.Parallel()
	classicContractExecutePlatform(t, "claude")
}

func TestClassicContractCorpusOpenCode(t *testing.T) {
	t.Parallel()
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
	contractDir := classicContractFixtureDir(t)
	document, err := loadClassicContractDocument(filepath.Join(contractDir, "cases.json"))
	if err != nil {
		t.Fatalf("load Classic contract corpus: %v", err)
	}
	if err := validateClassicContractCorpus(document); err != nil {
		t.Fatalf("validate Classic contract corpus: %v", err)
	}
	registry, err := loadClassicMechanismRegistry(filepath.Join(contractDir, "mechanisms.json"))
	if err != nil {
		t.Fatalf("load Classic mechanism registry: %v", err)
	}
	if err := validateClassicPhase200Corpus(document, registry); err != nil {
		t.Fatalf("validate Phase 200 Classic corpus: %v", err)
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
	phase199Cases := 0
	for _, testCase := range document.Cases {
		if strings.HasPrefix(testCase.SynthesisDecision, "SYN-200-") || strings.HasPrefix(testCase.SynthesisDecision, "SYN-201-") {
			continue
		}
		phase199Cases++
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
	if len(platforms) != len(required) || phase199Cases != len(required)*2 {
		return fmt.Errorf("corpus has %d Phase 199 cases, want exactly %d platform-expanded required journeys", phase199Cases, len(required)*2)
	}
	return nil
}

func validateClassicPhase200Corpus(document classicContractDocument, registry classicMechanismRegistry) error {
	caseByID := make(map[string]classicContractCase)
	groupClasses := make(map[string]map[string]bool, len(classicContractPhase200Groups))
	assertionCoverage := make(map[string]bool)
	recommendationRequired := map[string]bool{
		"approved-two-pass-journey": true, "ordered-material-boundaries": true,
		"reasoned-stop-matrix": true, "exact-candidate-acceptance": true, "scoped-spec-revision": true,
	}
	for _, testCase := range document.Cases {
		if !strings.HasPrefix(testCase.SynthesisDecision, "SYN-200-") {
			continue
		}
		caseByID[testCase.ID] = testCase
		proof := testCase.Phase200Proof
		if proof == nil {
			return fmt.Errorf("Phase 200 case %q requires phase200_proof", testCase.ID)
		}
		if !slices.Contains([]string{"success", "refusal"}, proof.Class) {
			return fmt.Errorf("Phase 200 case %q has invalid proof class %q", testCase.ID, proof.Class)
		}
		if strings.TrimSpace(proof.Scenario) == "" || len(proof.EvidenceThatWouldChange) == 0 || classicStringsContainBlank(proof.EvidenceThatWouldChange) {
			return fmt.Errorf("Phase 200 case %q requires a scenario and evidence_that_would_change", testCase.ID)
		}
		if !slices.Contains([]string{"changed", "unchanged"}, proof.Assertions.StateHashRelation) ||
			!slices.Contains([]string{"changed", "unchanged"}, proof.Assertions.ArtifactSetRelation) ||
			!slices.Contains([]string{"changed", "unchanged"}, proof.Assertions.ReceiptChainRelation) ||
			len(proof.Assertions.RequiredResultFields) == 0 || classicStringsContainBlank(proof.Assertions.RequiredResultFields) ||
			len(proof.Assertions.ProhibitedSideEffects) == 0 || classicStringsContainBlank(proof.Assertions.ProhibitedSideEffects) {
			return fmt.Errorf("Phase 200 case %q has incomplete causal assertions", testCase.ID)
		}
		if groupClasses[testCase.Group] == nil {
			groupClasses[testCase.Group] = map[string]bool{}
		}
		groupClasses[testCase.Group][proof.Class] = true
		if recommendationRequired[proof.Scenario] {
			if proof.Recommendation == nil || !slices.Contains([]string{"accept", "revise"}, proof.Recommendation.Disposition) ||
				strings.TrimSpace(proof.Recommendation.Rationale) == "" || len(proof.Recommendation.EvidenceIDs) == 0 ||
				classicStringsContainBlank(proof.Recommendation.EvidenceIDs) || proof.Recommendation.Producer != "queen" {
				return fmt.Errorf("Phase 200 case %q requires a persisted typed Queen recommendation", testCase.ID)
			}
		}
		var assertions []string
		if err := json.Unmarshal(testCase.Expected.SemanticFields["structured_assertions"], &assertions); err != nil || len(assertions) == 0 {
			return fmt.Errorf("Phase 200 case %q requires structured executable assertions", testCase.ID)
		}
		for _, assertion := range assertions {
			assertionCoverage[assertion] = true
		}
	}
	wantCaseCount := 16 + len(classicContractPhase200MechanismProofs)
	if len(caseByID) != wantCaseCount {
		return fmt.Errorf("Phase 200 corpus has %d cases, want exactly %d", len(caseByID), wantCaseCount)
	}
	for _, group := range classicContractPhase200Groups {
		if !groupClasses[group]["success"] || !groupClasses[group]["refusal"] {
			return fmt.Errorf("Phase 200 group %q requires one success and one refusal", group)
		}
	}
	for _, assertion := range []string{
		"specification.outcomes", "specification.included_behaviors", "specification.exclusions",
		"specification.binding_decisions", "specification.requirements", "specification.acceptance_checks",
		"specification.negative_expectations", "specification.recovery_expectations", "specification.affected_public_paths",
		"target_sufficient=target sufficiency", "diminishing_returns=diminishing returns",
		"stalled=stall detected", "iteration_cap=iteration cap", "boundary_card_hash",
		"crash_window_after_state_before_projection", "crash_window_after_projection_before_receipt",
		"legacy_migration_preserves_identity", "projection_repair_authority_unchanged",
		"presets=fast,balanced,deep,exhaustive",
	} {
		if !assertionCoverage[assertion] {
			return fmt.Errorf("Phase 200 corpus missing required causal assertion %q", assertion)
		}
	}

	referenced := make(map[string]bool, len(caseByID))
	mechanismCount := 0
	for _, mechanism := range registry.Mechanisms {
		if !strings.HasPrefix(mechanism.ID, "SYN-200-") {
			continue
		}
		mechanismCount++
		if len(mechanism.SourceAnchors) == 0 || classicStringsContainBlank(mechanism.SourceAnchors) || strings.TrimSpace(mechanism.ModernInvariant) == "" {
			return fmt.Errorf("mechanism %q requires source anchors and a modern invariant", mechanism.ID)
		}
		if len(mechanism.PositiveCaseIDs) == 0 || len(mechanism.NegativeCaseIDs) == 0 {
			return fmt.Errorf("mechanism %q requires positive and negative causal cases", mechanism.ID)
		}
		for _, reference := range append(slices.Clone(mechanism.PositiveCaseIDs), mechanism.NegativeCaseIDs...) {
			if _, ok := caseByID[reference]; !ok {
				return fmt.Errorf("mechanism %q references unknown Phase 200 case %q", mechanism.ID, reference)
			}
			referenced[reference] = true
		}
		for _, reference := range mechanism.PositiveCaseIDs {
			if caseByID[reference].Phase200Proof.Class != "success" {
				return fmt.Errorf("mechanism %q positive case %q is not a success", mechanism.ID, reference)
			}
		}
		for _, reference := range mechanism.NegativeCaseIDs {
			if caseByID[reference].Phase200Proof.Class != "refusal" {
				return fmt.Errorf("mechanism %q negative case %q is not a refusal", mechanism.ID, reference)
			}
		}
	}
	if mechanismCount != len(classicContractPhase200Decisions) {
		return fmt.Errorf("Phase 200 registry has %d mechanisms, want %d", mechanismCount, len(classicContractPhase200Decisions))
	}
	for caseID := range caseByID {
		if !referenced[caseID] {
			return fmt.Errorf("Phase 200 case %q is not referenced by a mechanism", caseID)
		}
	}
	return validateClassicPhase200IntegratedMechanismProofs(caseByID, registry)
}

func validateClassicPhase200IntegratedMechanismProofs(caseByID map[string]classicContractCase, registry classicMechanismRegistry) error {
	mechanismByID := make(map[string]classicMechanism, len(registry.Mechanisms))
	for _, mechanism := range registry.Mechanisms {
		mechanismByID[mechanism.ID] = mechanism
	}
	proofCaseCount := 0
	for _, testCase := range caseByID {
		if _, ok := testCase.Expected.SemanticFields["mechanism_id"]; ok {
			proofCaseCount++
		}
	}
	if proofCaseCount != len(classicContractPhase200MechanismProofs) {
		return fmt.Errorf("Phase 200 corpus has %d integrated mechanism proof rows, want exactly %d", proofCaseCount, len(classicContractPhase200MechanismProofs))
	}

	for _, expectation := range classicContractPhase200MechanismProofs {
		testCase, ok := caseByID[expectation.CaseID]
		if !ok {
			return fmt.Errorf("Phase 200 corpus missing integrated mechanism proof case %q", expectation.CaseID)
		}
		fields := make(map[string]string, 6)
		for _, field := range []string{"mechanism_id", "public_command", "positive_behavior", "hostile_or_refusal_behavior", "recovery_command", "go_test_symbol"} {
			value, err := classicPhase200SemanticString(testCase, field)
			if err != nil {
				return err
			}
			fields[field] = value
		}
		if fields["mechanism_id"] != expectation.MechanismID {
			return fmt.Errorf("Phase 200 case %q mechanism_id = %q, want %q", testCase.ID, fields["mechanism_id"], expectation.MechanismID)
		}
		if fields["public_command"] != expectation.PublicCommand || !classicContractPhase200PublicCommands[fields["public_command"]] {
			return fmt.Errorf("Phase 200 case %q public_command = %q, want current command %q", testCase.ID, fields["public_command"], expectation.PublicCommand)
		}
		if fields["recovery_command"] != expectation.RecoveryCommand || !classicContractPhase200PublicCommands[fields["recovery_command"]] {
			return fmt.Errorf("Phase 200 case %q recovery_command = %q, want exact current recovery %q", testCase.ID, fields["recovery_command"], expectation.RecoveryCommand)
		}
		if fields["go_test_symbol"] != expectation.GoTestSymbol {
			return fmt.Errorf("Phase 200 case %q go_test_symbol = %q, want %q", testCase.ID, fields["go_test_symbol"], expectation.GoTestSymbol)
		}
		if classicContractRoundVocabulary.MatchString(fields["positive_behavior"] + " " + fields["hostile_or_refusal_behavior"]) {
			return fmt.Errorf("Phase 200 case %q uses retired round vocabulary instead of planning iteration/pass", testCase.ID)
		}
		if expectation.CaseID == "phase200.mechanism.specification-canonical-recomputation" && !strings.Contains(fields["positive_behavior"], "no-change") {
			return fmt.Errorf("Phase 200 case %q must preserve the exact no-change decision", testCase.ID)
		}
		mechanism, ok := mechanismByID[expectation.MechanismID]
		linkedCases := mechanism.PositiveCaseIDs
		if testCase.Phase200Proof != nil && testCase.Phase200Proof.Class == "refusal" {
			linkedCases = mechanism.NegativeCaseIDs
		}
		if !ok || !slices.Contains(linkedCases, testCase.ID) || !slices.Contains(mechanism.PublicCommands, expectation.PublicCommand) {
			return fmt.Errorf("Phase 200 case %q is not causally linked to mechanism %q and public command %q", testCase.ID, expectation.MechanismID, expectation.PublicCommand)
		}
		resolves, err := classicPhase200GoTestSymbolResolves(testCase, expectation.GoTestSymbol)
		if err != nil {
			return err
		}
		if !resolves {
			return fmt.Errorf("Phase 200 case %q requires a cited, resolvable Go test symbol %q; source-only rows are noncausal", testCase.ID, expectation.GoTestSymbol)
		}
	}
	return nil
}

func classicPhase200SemanticString(testCase classicContractCase, field string) (string, error) {
	raw, ok := testCase.Expected.SemanticFields[field]
	if !ok {
		return "", fmt.Errorf("Phase 200 case %q requires %s", testCase.ID, field)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("Phase 200 case %q requires a nonblank string %s", testCase.ID, field)
	}
	return value, nil
}

func classicPhase200GoTestSymbolResolves(testCase classicContractCase, symbol string) (bool, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return false, fmt.Errorf("locate Classic contract source root")
	}
	repoRoot := filepath.Dir(filepath.Dir(sourceFile))
	for _, citation := range testCase.SourceCitations {
		if !strings.HasPrefix(citation, "cmd/") || !strings.HasSuffix(citation, "_test.go") {
			continue
		}
		path := filepath.Join(repoRoot, filepath.FromSlash(citation))
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return false, fmt.Errorf("parse cited Go test %s for case %q: %w", citation, testCase.ID, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Recv == nil && function.Name.Name == symbol {
				return true, nil
			}
		}
	}
	return false, nil
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

func classicContractWithoutExactCase(cases []classicContractCase, caseID string) []classicContractCase {
	result := make([]classicContractCase, 0, len(cases)-1)
	for _, testCase := range cases {
		if testCase.ID != caseID {
			result = append(result, testCase)
		}
	}
	return result
}

func classicStringsWithout(values []string, target string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}

func removeClassicPhase200StructuredAssertion(t *testing.T, document *classicContractDocument, caseID, assertion string) {
	t.Helper()
	for index := range document.Cases {
		if document.Cases[index].ID != caseID {
			continue
		}
		var assertions []string
		if err := json.Unmarshal(document.Cases[index].Expected.SemanticFields["structured_assertions"], &assertions); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(classicStringsWithout(assertions, assertion))
		if err != nil {
			t.Fatal(err)
		}
		document.Cases[index].Expected.SemanticFields["structured_assertions"] = encoded
		return
	}
	t.Fatalf("Phase 200 fixture has no case %q", caseID)
}

func removeClassicPhase200MechanismProofField(t *testing.T, document *classicContractDocument, caseID, field string) {
	t.Helper()
	for index := range document.Cases {
		if document.Cases[index].ID != caseID {
			continue
		}
		delete(document.Cases[index].Expected.SemanticFields, field)
		return
	}
	t.Fatalf("Phase 200 fixture has no case %q", caseID)
}

func assertClassicContractJourneyMatrix(t *testing.T, document classicContractDocument) {
	t.Helper()
	if err := validateClassicContractCorpus(document); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, testCase := range document.Cases {
		if strings.HasPrefix(testCase.SynthesisDecision, "SYN-200-") {
			continue
		}
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
		if strings.HasPrefix(testCase.SynthesisDecision, "SYN-200-") {
			continue
		}
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
		if strings.HasPrefix(testCase.SynthesisDecision, "SYN-200-") && testCase.Phase200Proof == nil {
			return fmt.Errorf("case %q requires phase200_proof", testCase.ID)
		}
		if !strings.HasPrefix(testCase.SynthesisDecision, "SYN-200-") && testCase.Phase200Proof != nil {
			return fmt.Errorf("case %q cannot attach phase200_proof to a Phase 199 decision", testCase.ID)
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

// validateClassicCaseSynthesisCoverage confirms every case's synthesis
// decision names a mechanism the registry actually carries. A case can cite a
// decision no mechanism registers only through a stale draft or a typo; this
// check refuses that by the offending decision's own name rather than
// silently accepting an orphaned case. General-purpose across phases -- it
// does not filter by ID prefix -- so a later phase's case set is covered by
// the same function without another copy.
func validateClassicCaseSynthesisCoverage(cases []classicContractCase, registry classicMechanismRegistry) error {
	registered := make(map[string]bool, len(registry.Mechanisms))
	for _, mechanism := range registry.Mechanisms {
		registered[mechanism.ID] = true
	}
	for _, testCase := range cases {
		if !registered[testCase.SynthesisDecision] {
			return fmt.Errorf("case %q references unregistered synthesis decision %q", testCase.ID, testCase.SynthesisDecision)
		}
	}
	return nil
}

func validateClassicMechanismRegistry(registry classicMechanismRegistry) error {
	if registry.SchemaVersion != classicContractSchemaVersion {
		return fmt.Errorf("schema_version = %q, want %q", registry.SchemaVersion, classicContractSchemaVersion)
	}
	if registry.SynthesisSource != ".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md" {
		return fmt.Errorf("unexpected synthesis_source %q", registry.SynthesisSource)
	}
	if !slices.Equal(registry.SynthesisSources, []string{
		".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md",
		".planning/phases/200-iterative-planning/200-CLASSIC-SYNTHESIS.md",
		".planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md",
	}) {
		return fmt.Errorf("unexpected synthesis_sources %v", registry.SynthesisSources)
	}
	if len(registry.Mechanisms) == 0 {
		return fmt.Errorf("mechanism registry is empty")
	}

	decisionCounts := make(map[string]int, len(registry.Mechanisms))
	phase199CAPCounts := make(map[string]int, len(classicContractPhase199Capabilities))
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
			if strings.HasPrefix(mechanism.ID, "SYN-199-") {
				phase199CAPCounts[capID]++
			} else if strings.HasPrefix(mechanism.ID, "SYN-201-") {
				if !slices.Contains(classicContractPhase201Capabilities, capID) {
					return fmt.Errorf("mechanism %q has unexpected Phase 201 capability %q", mechanism.ID, capID)
				}
			} else if !slices.Contains(classicContractPhase200Capabilities, capID) {
				return fmt.Errorf("mechanism %q has unexpected Phase 200 capability %q", mechanism.ID, capID)
			}
		}
		if len(mechanism.PublicCommands) == 0 || classicStringsContainBlank(mechanism.PublicCommands) {
			return fmt.Errorf("mechanism %q requires public_commands", mechanism.ID)
		}
		if strings.HasPrefix(mechanism.ID, "SYN-200-") {
			for _, command := range mechanism.PublicCommands {
				if !classicContractPhase200PublicCommands[command] {
					return fmt.Errorf("mechanism %q uses stale or non-public command %q", mechanism.ID, command)
				}
			}
		}
		if strings.HasPrefix(mechanism.ID, "SYN-201-") {
			for _, command := range mechanism.PublicCommands {
				if !classicContractPhase201PublicCommands[command] {
					return fmt.Errorf("mechanism %q uses stale or non-public command %q", mechanism.ID, command)
				}
			}
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
		if strings.HasPrefix(mechanism.ID, "SYN-200-") {
			if len(mechanism.SourceAnchors) == 0 || classicStringsContainBlank(mechanism.SourceAnchors) || strings.TrimSpace(mechanism.ModernInvariant) == "" {
				return fmt.Errorf("mechanism %q requires source_anchors and modern_invariant", mechanism.ID)
			}
			if len(mechanism.PositiveCaseIDs) == 0 || len(mechanism.NegativeCaseIDs) == 0 || classicStringsContainBlank(mechanism.PositiveCaseIDs) || classicStringsContainBlank(mechanism.NegativeCaseIDs) {
				return fmt.Errorf("mechanism %q requires positive_case_ids and negative_case_ids", mechanism.ID)
			}
		} else if len(mechanism.SourceAnchors) != 0 || mechanism.ModernInvariant != "" || len(mechanism.PositiveCaseIDs) != 0 || len(mechanism.NegativeCaseIDs) != 0 {
			return fmt.Errorf("Phase 199 mechanism %q cannot carry Phase 200 proof fields", mechanism.ID)
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
	for _, id := range classicContractPhase199Capabilities {
		switch phase199CAPCounts[id] {
		case 0:
			return fmt.Errorf("missing capability %q", id)
		case 1:
		default:
			return fmt.Errorf("duplicate capability %q", id)
		}
	}
	for id := range phase199CAPCounts {
		if !slices.Contains(classicContractPhase199Capabilities, id) {
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

func classicMechanismsWithoutDecision(mechanisms []classicMechanism, decisionID string) []classicMechanism {
	result := make([]classicMechanism, 0, len(mechanisms)-1)
	for _, mechanism := range mechanisms {
		if mechanism.ID != decisionID {
			result = append(result, mechanism)
		}
	}
	return result
}

func assertClassicSynthesisSource(t *testing.T, repoRoot, source string, decisions, capabilities []string, exact bool) {
	t.Helper()
	path := filepath.Join(repoRoot, filepath.FromSlash(source))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read mechanism synthesis %s: %v", path, err)
	}
	for _, id := range decisions {
		count := strings.Count(string(data), id)
		if count == 0 || exact && count != 1 {
			t.Errorf("synthesis identifier %s occurs %d times in %s", id, count, source)
		}
	}
	for _, id := range capabilities {
		if !bytes.Contains(data, []byte(id)) {
			t.Errorf("synthesis %s missing routed capability %s", source, id)
		}
	}
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
