package cmd

// LEARN-04 (204-05-PLAN.md): turns this project's own confirmed failures
// into a versioned bank of regression fixtures, each carrying its failing
// baseline, the invariant it protects, the mutations it permits, its
// provenance back to the originating incident, its privacy disposition and
// its retirement or successor rule. Nothing unconfirmed becomes a fixture.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// fixtureBankPath is the repo-relative path of the committed, versioned
// fixture bank. This is a committed source file, like the Classic contract
// corpus (cmd/testdata/classic-contract/v1/), not runtime colony state, so
// it is read and written directly against the repository checkout rather
// than through storage.Store.
const fixtureBankPath = "cmd/testdata/fixture-bank/v1/bank.json"

// fixtureBankSchemaPath is the repo-relative path of the versioned JSON
// Schema every fixture bank validates against.
const fixtureBankSchemaPath = "cmd/testdata/fixture-bank/v1/schema.json"

// fixtureBankSchemaVersion is the schema_version constant every bank file carries.
const fixtureBankSchemaVersion = "fixture-bank/v1"

// fixtureBankRepoRootOverride lets tests redirect fixture bank reads/writes
// to an isolated temporary directory instead of the real repository
// checkout, so exercising loadFixtureBank/writeFixtureBank in a test never
// mutates the real committed cmd/testdata/fixture-bank/v1/bank.json.
var fixtureBankRepoRootOverride string

// fixtureBankRepoRoot resolves the directory fixtureBankPath and
// fixtureBankSchemaPath are relative to: fixtureBankRepoRootOverride when a
// test has set it, otherwise the real Aether module root.
func fixtureBankRepoRoot() (string, error) {
	if fixtureBankRepoRootOverride != "" {
		return fixtureBankRepoRootOverride, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("locate Aether module root from %s", cwd)
	}
	return root, nil
}

// --- Declared vocabularies ---
//
// Each vocabulary below follows this repository's own completeness
// convention (const block, names slice, names() helper, declared-membership
// predicate) -- see cmd/recruitment_credit.go's
// recruitmentCreditOutcomeVocabulary for the precedent this mirrors.

// fixtureAllowedMutation is the declared, closed vocabulary of what a later
// change may alter without this fixture being wrong to fail.
type fixtureAllowedMutation string

const (
	fixtureMutationWording    fixtureAllowedMutation = "wording"
	fixtureMutationFormatting fixtureAllowedMutation = "formatting"
	fixtureMutationFilePath   fixtureAllowedMutation = "file_path"
	fixtureMutationSymbolName fixtureAllowedMutation = "symbol_name"
	fixtureMutationNone       fixtureAllowedMutation = "none"
)

var fixtureAllowedMutationVocabulary = []fixtureAllowedMutation{
	fixtureMutationWording,
	fixtureMutationFormatting,
	fixtureMutationFilePath,
	fixtureMutationSymbolName,
	fixtureMutationNone,
}

func fixtureAllowedMutationNames() []string {
	names := make([]string, 0, len(fixtureAllowedMutationVocabulary))
	for _, m := range fixtureAllowedMutationVocabulary {
		names = append(names, string(m))
	}
	return names
}

func fixtureAllowedMutationDeclared(m fixtureAllowedMutation) bool {
	for _, v := range fixtureAllowedMutationVocabulary {
		if v == m {
			return true
		}
	}
	return false
}

// fixtureProvenanceKind is the declared, closed vocabulary of source kinds
// a fixture's provenance reference may name.
type fixtureProvenanceKind string

const (
	fixtureProvenanceFailureLog        fixtureProvenanceKind = "failure_log"
	fixtureProvenanceAuditFinding      fixtureProvenanceKind = "audit_finding"
	fixtureProvenanceDefectRegister    fixtureProvenanceKind = "defect_register"
	fixtureProvenancePhaseVerification fixtureProvenanceKind = "phase_verification"
	fixtureProvenancePhaseReview       fixtureProvenanceKind = "phase_review"
)

var fixtureProvenanceKindVocabulary = []fixtureProvenanceKind{
	fixtureProvenanceFailureLog,
	fixtureProvenanceAuditFinding,
	fixtureProvenanceDefectRegister,
	fixtureProvenancePhaseVerification,
	fixtureProvenancePhaseReview,
}

func fixtureProvenanceKindNames() []string {
	names := make([]string, 0, len(fixtureProvenanceKindVocabulary))
	for _, k := range fixtureProvenanceKindVocabulary {
		names = append(names, string(k))
	}
	return names
}

func fixtureProvenanceKindDeclared(k fixtureProvenanceKind) bool {
	for _, v := range fixtureProvenanceKindVocabulary {
		if v == k {
			return true
		}
	}
	return false
}

// fixturePrivacy is the declared, closed privacy-disposition vocabulary.
type fixturePrivacy string

const (
	fixturePrivacyPublic    fixturePrivacy = "public"
	fixturePrivacyRepoLocal fixturePrivacy = "repo_local"
	fixturePrivacyRedacted  fixturePrivacy = "redacted"
)

var fixturePrivacyVocabulary = []fixturePrivacy{
	fixturePrivacyPublic,
	fixturePrivacyRepoLocal,
	fixturePrivacyRedacted,
}

func fixturePrivacyNames() []string {
	names := make([]string, 0, len(fixturePrivacyVocabulary))
	for _, p := range fixturePrivacyVocabulary {
		names = append(names, string(p))
	}
	return names
}

func fixturePrivacyDeclared(p fixturePrivacy) bool {
	for _, v := range fixturePrivacyVocabulary {
		if v == p {
			return true
		}
	}
	return false
}

// fixtureSeverity is the declared, closed severity vocabulary.
type fixtureSeverity string

const (
	fixtureSeverityLow      fixtureSeverity = "low"
	fixtureSeverityMedium   fixtureSeverity = "medium"
	fixtureSeverityHigh     fixtureSeverity = "high"
	fixtureSeverityCritical fixtureSeverity = "critical"
)

var fixtureSeverityVocabulary = []fixtureSeverity{
	fixtureSeverityLow,
	fixtureSeverityMedium,
	fixtureSeverityHigh,
	fixtureSeverityCritical,
}

func fixtureSeverityNames() []string {
	names := make([]string, 0, len(fixtureSeverityVocabulary))
	for _, s := range fixtureSeverityVocabulary {
		names = append(names, string(s))
	}
	return names
}

func fixtureSeverityDeclared(s fixtureSeverity) bool {
	for _, v := range fixtureSeverityVocabulary {
		if v == s {
			return true
		}
	}
	return false
}

// --- Data types ---

// regressionFixtureFailingBaseline names the state in which this fixture's
// check fails, plus the reference that proves that state was real.
type regressionFixtureFailingBaseline struct {
	Description string `json:"description"`
	Reference   string `json:"reference"`
}

// regressionFixtureProvenance names one source a fixture traces back to.
type regressionFixtureProvenance struct {
	Kind       fixtureProvenanceKind `json:"kind"`
	Identifier string                `json:"identifier"`
}

// regressionFixture is one entry in the versioned fixture bank. A fixture
// cannot enter the bank without saying what it protects, what broke to
// create it, what a later change may legitimately alter, and where it came
// from -- enforced by cmd/testdata/fixture-bank/v1/schema.json.
type regressionFixture struct {
	ID               string                           `json:"id"`
	Title            string                           `json:"title"`
	FailingBaseline  regressionFixtureFailingBaseline  `json:"failing_baseline"`
	Invariant        string                           `json:"invariant"`
	AllowedMutations []fixtureAllowedMutation          `json:"allowed_mutations"`
	Provenance       []regressionFixtureProvenance     `json:"provenance"`
	Privacy          fixturePrivacy                    `json:"privacy"`
	Severity         fixtureSeverity                   `json:"severity"`
	ContentDigest    string                            `json:"content_digest"`
	RetiredBy        string                            `json:"retired_by,omitempty"`
	SuccessorID      string                            `json:"successor_id,omitempty"`
}

// regressionFixtureBank is the on-disk container at fixtureBankPath.
type regressionFixtureBank struct {
	SchemaVersion string               `json:"schema_version"`
	Fixtures      []regressionFixture  `json:"fixtures"`
}

// fixtureBankSortedByID returns a new slice holding bank's fixtures sorted
// ascending by ID -- fixture IDs are unique (derived from a content digest,
// see convertConfirmedIncidentsToFixtures), so ID alone is already a total,
// stable order; two fixtures of equal severity are therefore separated by
// identifier the same way every other pair is.
func fixtureBankSortedByID(bank regressionFixtureBank) []regressionFixture {
	sorted := make([]regressionFixture, len(bank.Fixtures))
	copy(sorted, bank.Fixtures)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

// loadFixtureBank reads the committed fixture bank. A missing file is not
// an error -- it is the honest "nothing generated yet" state -- and
// returns an empty, schema-versioned bank. Fixtures are always returned
// sorted ascending by ID (fixtureBankSortedByID), so loading twice returns
// identical fixtures in identical order.
func loadFixtureBank() (regressionFixtureBank, error) {
	root, err := fixtureBankRepoRoot()
	if err != nil {
		return regressionFixtureBank{}, err
	}
	path := filepath.Join(root, fixtureBankPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return regressionFixtureBank{SchemaVersion: fixtureBankSchemaVersion, Fixtures: []regressionFixture{}}, nil
		}
		return regressionFixtureBank{}, fmt.Errorf("read fixture bank %s: %w", path, err)
	}
	var bank regressionFixtureBank
	if err := json.Unmarshal(data, &bank); err != nil {
		return regressionFixtureBank{}, fmt.Errorf("unmarshal fixture bank %s: %w", path, err)
	}
	bank.Fixtures = fixtureBankSortedByID(bank)
	return bank, nil
}

// --- Schema validation ---

// fixtureBankSchemaID is the $id every compiled fixture bank schema
// resource is registered under.
const fixtureBankSchemaID = "https://aether.dev/schemas/fixture-bank/v1"

// compileFixtureBankSchema compiles the committed fixture bank JSON Schema
// from disk. Recompiled on every call rather than cached with sync.Once,
// because tests redirect fixtureBankRepoRootOverride per test and a cached
// compile would silently validate against the wrong root's schema file.
func compileFixtureBankSchema() (*jsonschema.Schema, error) {
	root, err := fixtureBankRepoRoot()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(root, fixtureBankSchemaPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixture bank schema %s: %w", path, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse fixture bank schema %s: %w", path, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(fixtureBankSchemaID, doc); err != nil {
		return nil, fmt.Errorf("register fixture bank schema resource: %w", err)
	}
	compiled, err := compiler.Compile(fixtureBankSchemaID)
	if err != nil {
		return nil, fmt.Errorf("compile fixture bank schema: %w", err)
	}
	return compiled, nil
}

// validateFixtureBankDocument validates raw (a decoded JSON value, typically
// map[string]any) against the committed fixture bank schema and returns one
// human-readable violation message per leaf failure -- reusing this
// package's existing violationPrinter/jsonPointerFromSegments (defined
// alongside the completion-packet schema in cmd/contract_schema.go) rather
// than a second formatter. A nil/empty return means raw is structurally
// valid.
func validateFixtureBankDocument(raw any) ([]string, error) {
	schema, err := compileFixtureBankSchema()
	if err != nil {
		return nil, err
	}
	validateErr := schema.Validate(raw)
	if validateErr == nil {
		return nil, nil
	}
	var verr *jsonschema.ValidationError
	if !errors.As(validateErr, &verr) {
		return []string{validateErr.Error()}, nil
	}
	var messages []string
	var walk func(node *jsonschema.ValidationError)
	walk = func(node *jsonschema.ValidationError) {
		if node == nil {
			return
		}
		if len(node.Causes) == 0 {
			field := jsonPointerFromSegments(node.InstanceLocation)
			text := ""
			if node.ErrorKind != nil {
				text = node.ErrorKind.LocalizedString(violationPrinter)
			}
			if field != "" {
				messages = append(messages, fmt.Sprintf("%s: %s", field, text))
			} else {
				messages = append(messages, text)
			}
			return
		}
		for _, cause := range node.Causes {
			walk(cause)
		}
	}
	walk(verr)
	return messages, nil
}
