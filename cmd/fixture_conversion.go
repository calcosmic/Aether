package cmd

// LEARN-04 (204-05-PLAN.md): turns this project's own confirmed failures
// into a versioned bank of regression fixtures, each carrying its failing
// baseline, the invariant it protects, the mutations it permits, its
// provenance back to the originating incident, its privacy disposition and
// its retirement or successor rule. Nothing unconfirmed becomes a fixture.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/calcosmic/Aether/pkg/colony"
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

// regressionFixtureGuard names the Go test that would fail if this
// fixture's invariant broke, and the package it lives in. A fixture with no
// Guard is counted as unguarded (204-07-PLAN.md, LEARN-05) -- listed, never
// silently hidden, on a ratchet that may only decrease
// (seedBankUnguardedFloor, cmd/eval_gates.go).
type regressionFixtureGuard struct {
	Test    string `json:"test"`
	Package string `json:"package"`
}

// regressionFixture is one entry in the versioned fixture bank. A fixture
// cannot enter the bank without saying what it protects, what broke to
// create it, what a later change may legitimately alter, and where it came
// from -- enforced by cmd/testdata/fixture-bank/v1/schema.json.
type regressionFixture struct {
	ID               string                           `json:"id"`
	Title            string                           `json:"title"`
	FailingBaseline  regressionFixtureFailingBaseline `json:"failing_baseline"`
	Invariant        string                           `json:"invariant"`
	AllowedMutations []fixtureAllowedMutation         `json:"allowed_mutations"`
	Provenance       []regressionFixtureProvenance    `json:"provenance"`
	Privacy          fixturePrivacy                   `json:"privacy"`
	Severity         fixtureSeverity                  `json:"severity"`
	ContentDigest    string                           `json:"content_digest"`
	RetiredBy        string                           `json:"retired_by,omitempty"`
	SuccessorID      string                           `json:"successor_id,omitempty"`
	Guard            *regressionFixtureGuard          `json:"guard,omitempty"`
}

// regressionFixtureBank is the on-disk container at fixtureBankPath.
type regressionFixtureBank struct {
	SchemaVersion string              `json:"schema_version"`
	Fixtures      []regressionFixture `json:"fixtures"`
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

// --- Confirmed-incident conversion ---
//
// "Confirmed" means a human or a gate already agreed it was real (Planner
// Assumption J, 204-05-PLAN.md): a failure-log entry marked both reviewed
// and acknowledged; an audit finding marked verified; a defect-register
// entry that reached a fixed state; or a phase closing report's own
// found-and-repaired-at-the-gate row naming a commit. Anything else --
// unreviewed, unverified, still open, no commit named -- is not confirmed
// and produces nothing.

// fixtureIncidentSanitizeBudget mirrors pkg/colony/sanitize.go's own
// maxSignalContentLength (500 chars). Incident text is truncated to this
// budget BEFORE it reaches colony.SanitizeSignalContent -- the sanitizer's
// own length check runs first, before its escaping step, so an untruncated
// confirmed incident (this project's own audit/defect-register prose
// regularly runs well past 500 characters) would otherwise be refused for
// length on nearly every incident, not for genuinely unsafe content. This
// is a deliberate normalisation step, not a bypass: the sanitizer still
// runs, and still refuses genuinely unsafe content, on every incident.
const fixtureIncidentSanitizeBudget = 500

// confirmedIncident is the one normalised shape every source is read into
// before conversion, so convertConfirmedIncidentsToFixtures has one input
// type rather than four.
type confirmedIncident struct {
	SourceKind       fixtureProvenanceKind
	SourceIdentifier string
	Title            string
	Detail           string
	Severity         string
	Timestamp        string
}

// fixtureIncidentRefusal names one incident a reader declined to convert,
// and why.
type fixtureIncidentRefusal struct {
	Identifier string
	Reason     string
}

// fixtureIncidentReadResult is what every confirmedIncidentsFrom* reader
// returns: the incidents it confirmed, and -- alongside them -- every
// incident it refused and the reason class for each refusal, so the
// conversion can report what it declined rather than silently narrowing.
type fixtureIncidentReadResult struct {
	Incidents []confirmedIncident
	Refused   []fixtureIncidentRefusal
}

// confirmedIncidentsFromFailureLog reads the existing failure log
// (loadMiddenFile) and keeps entries where the reviewed flag and the
// acknowledged marker are both set. Read only -- this never modifies the
// failure-log type or its writers.
func confirmedIncidentsFromFailureLog() (fixtureIncidentReadResult, error) {
	mf, err := loadMiddenFile(store)
	if err != nil {
		// Matches every other reader of midden.json in this package
		// (cmd/context.go, cmd/memory_health.go, ...): a missing or
		// unreadable failure log is treated as empty, never an error.
		return fixtureIncidentReadResult{}, nil
	}

	var result fixtureIncidentReadResult
	for _, entry := range mf.Entries {
		if !entry.Reviewed {
			result.Refused = append(result.Refused, fixtureIncidentRefusal{
				Identifier: entry.ID,
				Reason:     "not reviewed",
			})
			continue
		}
		if entry.Acknowledged == nil || !*entry.Acknowledged {
			result.Refused = append(result.Refused, fixtureIncidentRefusal{
				Identifier: entry.ID,
				Reason:     "reviewed but not acknowledged",
			})
			continue
		}
		title := entry.Category
		if entry.Source != "" {
			title = title + " (" + entry.Source + ")"
		}
		result.Incidents = append(result.Incidents, confirmedIncident{
			SourceKind:       fixtureProvenanceFailureLog,
			SourceIdentifier: entry.ID,
			Title:            title,
			Detail:           entry.Message,
			Timestamp:        entry.Timestamp,
		})
	}
	return result, nil
}

// fixtureAuditFindingRecord is the shape of one entry in a whole-system
// audit's "confirmed" array (.planning/audits/*.json), read only for the
// fields this conversion needs.
type fixtureAuditFindingRecord struct {
	Subject  string `json:"subject"`
	Status   string `json:"status"`
	Claim    string `json:"claim"`
	Severity string `json:"severity"`
	Verified bool   `json:"verified"`
}

type fixtureAuditFindingsFile struct {
	Confirmed []fixtureAuditFindingRecord `json:"confirmed"`
}

// confirmedIncidentsFromAuditFindings parses a whole-system audit's JSON at
// path and keeps findings whose verified flag is set. The identifier
// recorded for each is its own array index within the "confirmed" array
// (e.g. "confirmed[3]") -- stable across re-runs of the same file and
// always resolvable back to a real entry, which plain "subject" text is
// not (this file's own subjects are not all unique).
func confirmedIncidentsFromAuditFindings(path string) (fixtureIncidentReadResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fixtureIncidentReadResult{}, nil
		}
		return fixtureIncidentReadResult{}, fmt.Errorf("read audit findings %s: %w", path, err)
	}
	var file fixtureAuditFindingsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return fixtureIncidentReadResult{}, fmt.Errorf("unmarshal audit findings %s: %w", path, err)
	}

	var result fixtureIncidentReadResult
	for i, finding := range file.Confirmed {
		identifier := fmt.Sprintf("confirmed[%d]", i)
		if !finding.Verified {
			result.Refused = append(result.Refused, fixtureIncidentRefusal{
				Identifier: identifier,
				Reason:     "not verified",
			})
			continue
		}
		result.Incidents = append(result.Incidents, confirmedIncident{
			SourceKind:       fixtureProvenanceAuditFinding,
			SourceIdentifier: identifier,
			Title:            finding.Subject,
			Detail:           finding.Claim,
			Severity:         finding.Severity,
		})
	}
	return result, nil
}

// fixtureWindowsLedgerEntry is the shape of one entry in .planning/
// WINDOWS.md's own fenced JSON ledger dump -- the defect register.
type fixtureWindowsLedgerEntry struct {
	ID          int    `json:"id"`
	Kind        string `json:"kind"`
	Phase       string `json:"phase"`
	File        string `json:"file"`
	Description string `json:"description"`
	Status      string `json:"status"`
	RecordedAt  string `json:"recorded_at"`
}

var (
	fixtureWindowsFenceOpen  = regexp.MustCompile("^`{3,}json\\s*$")
	fixtureWindowsFenceClose = regexp.MustCompile("^`{3,}\\s*$")
)

// extractFencedJSONBlock returns the content of the first fenced ```json
// (or, as WINDOWS.md itself uses, ````json) code block in content.
func extractFencedJSONBlock(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if fixtureWindowsFenceOpen.MatchString(strings.TrimRight(line, "\r")) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return "", false
	}
	for j := start; j < len(lines); j++ {
		if fixtureWindowsFenceClose.MatchString(strings.TrimRight(lines[j], "\r")) {
			return strings.Join(lines[start:j], "\n"), true
		}
	}
	return "", false
}

// confirmedIncidentsFromDefectRegister parses the defect register's own
// fenced JSON ledger dump at path (.planning/WINDOWS.md) and keeps rows
// whose status is "fixed". The JSON dump is used rather than the markdown
// table beside it, because the JSON dump is the well-formed, machine-
// generated source of truth the table mirrors.
func confirmedIncidentsFromDefectRegister(path string) (fixtureIncidentReadResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fixtureIncidentReadResult{}, nil
		}
		return fixtureIncidentReadResult{}, fmt.Errorf("read defect register %s: %w", path, err)
	}
	block, ok := extractFencedJSONBlock(string(raw))
	if !ok {
		return fixtureIncidentReadResult{}, fmt.Errorf("defect register %s carries no fenced JSON ledger block", path)
	}
	var entries []fixtureWindowsLedgerEntry
	if err := json.Unmarshal([]byte(block), &entries); err != nil {
		return fixtureIncidentReadResult{}, fmt.Errorf("unmarshal defect register ledger %s: %w", path, err)
	}

	var result fixtureIncidentReadResult
	for _, entry := range entries {
		identifier := strconv.Itoa(entry.ID)
		if entry.Status != "fixed" {
			result.Refused = append(result.Refused, fixtureIncidentRefusal{
				Identifier: identifier,
				Reason:     fmt.Sprintf("status is %q, not fixed", entry.Status),
			})
			continue
		}
		title := entry.Kind
		if entry.File != "" {
			title = title + ": " + entry.File
		}
		result.Incidents = append(result.Incidents, confirmedIncident{
			SourceKind:       fixtureProvenanceDefectRegister,
			SourceIdentifier: identifier,
			Title:            title,
			Detail:           entry.Description,
			Timestamp:        entry.RecordedAt,
		})
	}
	return result, nil
}

// fixtureCommitRefPattern matches a backtick-quoted commit-like hex string,
// e.g. "`91dcb741`".
var fixtureCommitRefPattern = regexp.MustCompile("`([0-9a-fA-F]{6,40})`")

// confirmedIncidentsFromPhaseReports scans dir (a directory of per-phase
// subdirectories, e.g. .planning/phases) for *-VERIFICATION.md files and
// parses each one's own "Fixed | Cause | Commit" table -- the rows a phase
// closing report itself recorded as found-and-repaired at its own gate --
// keeping only rows naming a commit.
func confirmedIncidentsFromPhaseReports(dir string) (fixtureIncidentReadResult, error) {
	phaseDirs, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fixtureIncidentReadResult{}, nil
		}
		return fixtureIncidentReadResult{}, fmt.Errorf("read phase directory %s: %w", dir, err)
	}

	var result fixtureIncidentReadResult
	for _, phaseEntry := range phaseDirs {
		if !phaseEntry.IsDir() {
			continue
		}
		phaseDir := filepath.Join(dir, phaseEntry.Name())
		files, err := os.ReadDir(phaseDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), "-VERIFICATION.md") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(phaseDir, f.Name()))
			if err != nil {
				continue
			}
			incidents, refusals := parseFixedCauseCommitTable(string(raw), phaseEntry.Name())
			result.Incidents = append(result.Incidents, incidents...)
			result.Refused = append(result.Refused, refusals...)
		}
	}
	return result, nil
}

// parseFixedCauseCommitTable finds every markdown table in content whose
// header row is exactly "Fixed | Cause | Commit" (case-insensitive) and
// parses its data rows. A row naming no commit is refused, never silently
// dropped.
func parseFixedCauseCommitTable(content, phaseName string) ([]confirmedIncident, []fixtureIncidentRefusal) {
	lines := strings.Split(content, "\n")
	var incidents []confirmedIncident
	var refusals []fixtureIncidentRefusal

	for i := 0; i < len(lines); i++ {
		header := splitMarkdownTableRow(lines[i])
		if !isFixedCauseCommitHeader(header) {
			continue
		}
		for row := i + 2; row < len(lines); row++ {
			cells := splitMarkdownTableRow(lines[row])
			if cells == nil {
				break
			}
			if len(cells) < 3 {
				continue
			}
			fixedCell, causeCell, commitCell := cells[0], cells[1], cells[2]
			identifier := phaseName + ":" + stripMarkdownEmphasis(fixedCell)
			match := fixtureCommitRefPattern.FindStringSubmatch(commitCell)
			if match == nil {
				refusals = append(refusals, fixtureIncidentRefusal{
					Identifier: identifier,
					Reason:     "no commit named",
				})
				continue
			}
			incidents = append(incidents, confirmedIncident{
				SourceKind:       fixtureProvenancePhaseVerification,
				SourceIdentifier: identifier,
				Title:            stripMarkdownEmphasis(fixedCell),
				Detail:           stripMarkdownEmphasis(causeCell),
			})
		}
	}
	return incidents, refusals
}

func isFixedCauseCommitHeader(cells []string) bool {
	if len(cells) < 3 {
		return false
	}
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	return norm(cells[0]) == "fixed" && norm(cells[1]) == "cause" && norm(cells[2]) == "commit"
}

// splitMarkdownTableRow splits a "| a | b | c |" line into trimmed cells.
// Returns nil when line is not a table row (does not start with "|") or is
// a separator row (only -, :, | and space).
func splitMarkdownTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return nil
	}
	if isMarkdownTableSeparator(trimmed) {
		return nil
	}
	parts := strings.Split(trimmed, "|")
	var cells []string
	for i, p := range parts {
		if (i == 0 || i == len(parts)-1) && strings.TrimSpace(p) == "" {
			continue
		}
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

func isMarkdownTableSeparator(line string) bool {
	for _, r := range line {
		switch r {
		case '|', '-', ':', ' ':
			continue
		default:
			return false
		}
	}
	return true
}

func stripMarkdownEmphasis(s string) string {
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	return strings.TrimSpace(s)
}

// fixtureConversionSummary reports what convertConfirmedIncidentsToFixtures
// did: every incident it dropped (and why), how many incidents
// deduplicated into an already-existing fixture, and the final fixture
// count.
type fixtureConversionSummary struct {
	Dropped           []fixtureIncidentRefusal
	Deduplicated      int
	FinalFixtureCount int
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func fixtureHasProvenance(existing []regressionFixtureProvenance, ref regressionFixtureProvenance) bool {
	for _, p := range existing {
		if p.Kind == ref.Kind && p.Identifier == ref.Identifier {
			return true
		}
	}
	return false
}

// convertConfirmedIncidentsToFixtures is the single conversion from
// confirmed incidents to fixture-bank entries. For each incident it runs
// the title and detail through colony.SanitizeSignalContent -- the same
// sanitiser every failure-log writer already uses, never a new scrubber --
// and drops an incident whose text the sanitiser refuses, recording it in
// the summary by identifier. It computes content_digest with crypto/sha256
// over the normalised sanitised text and uses that digest as the dedup
// key: an incident whose digest already exists appends its provenance
// reference to the existing fixture rather than creating a second one. It
// derives the fixture identifier deterministically from the digest, so a
// re-run produces the same identifier. It sets privacy to the
// repository-local disposition by default and to the redacted disposition
// when the sanitiser altered the text, never to the public disposition
// automatically.
func convertConfirmedIncidentsToFixtures(incidents []confirmedIncident) (regressionFixtureBank, fixtureConversionSummary, error) {
	var summary fixtureConversionSummary
	byDigest := map[string]*regressionFixture{}
	var digestOrder []string

	for _, incident := range incidents {
		combined := strings.TrimSpace(incident.Title)
		detail := strings.TrimSpace(incident.Detail)
		if detail != "" {
			if combined != "" {
				combined = combined + ". " + detail
			} else {
				combined = detail
			}
		}
		truncated := truncateRunes(combined, fixtureIncidentSanitizeBudget)

		sanitized, err := colony.SanitizeSignalContent(truncated)
		if err != nil {
			summary.Dropped = append(summary.Dropped, fixtureIncidentRefusal{
				Identifier: incident.SourceIdentifier,
				Reason:     fmt.Sprintf("sanitiser refused: %v", err),
			})
			continue
		}

		digestBytes := sha256.Sum256([]byte(sanitized))
		digest := hex.EncodeToString(digestBytes[:])

		provenanceRef := regressionFixtureProvenance{
			Kind:       incident.SourceKind,
			Identifier: incident.SourceIdentifier,
		}

		if existing, ok := byDigest[digest]; ok {
			if !fixtureHasProvenance(existing.Provenance, provenanceRef) {
				existing.Provenance = append(existing.Provenance, provenanceRef)
			}
			summary.Deduplicated++
			continue
		}

		privacy := fixturePrivacyRepoLocal
		if sanitized != truncated {
			privacy = fixturePrivacyRedacted
		}

		severity := fixtureSeverity(strings.ToLower(strings.TrimSpace(incident.Severity)))
		if !fixtureSeverityDeclared(severity) {
			severity = fixtureSeverityMedium
		}

		// This conversion is mechanical text-transcription, not deep
		// semantic analysis of the underlying code -- it cannot itself
		// determine a precise code invariant, only restate the
		// incident's own confirmed description. Per this plan's own
		// escape valve ("where the incident's description is too thin
		// to yield an invariant, set allowed_mutations to the
		// permissive members and say so in the fixture's own title"),
		// every generated fixture is honestly marked broad rather than
		// asserting a precision this program did not actually derive.
		fixture := &regressionFixture{
			ID:    "fixture-" + digest[:12],
			Title: "[broad-invariant] " + truncateRunes(sanitized, 96),
			FailingBaseline: regressionFixtureFailingBaseline{
				Description: sanitized,
				Reference:   incident.SourceIdentifier,
			},
			Invariant: "The confirmed defect must not recur: " + sanitized,
			AllowedMutations: []fixtureAllowedMutation{
				fixtureMutationWording,
				fixtureMutationFormatting,
				fixtureMutationFilePath,
				fixtureMutationSymbolName,
			},
			Provenance:    []regressionFixtureProvenance{provenanceRef},
			Privacy:       privacy,
			Severity:      severity,
			ContentDigest: digest,
		}
		byDigest[digest] = fixture
		digestOrder = append(digestOrder, digest)
	}

	fixtures := make([]regressionFixture, 0, len(digestOrder))
	for _, digest := range digestOrder {
		fixtures = append(fixtures, *byDigest[digest])
	}
	sort.SliceStable(fixtures, func(i, j int) bool { return fixtures[i].ID < fixtures[j].ID })

	summary.FinalFixtureCount = len(fixtures)
	bank := regressionFixtureBank{SchemaVersion: fixtureBankSchemaVersion, Fixtures: fixtures}
	return bank, summary, nil
}

// writeFixtureBank is the only writer of fixtureBankPath
// (TestFixtureBankHasOneWriter enforces this by name). It refuses to write
// a bank that fails its own schema, and aborts the write when the bank is
// byte-identical to what is already on disk, so a re-run touches no bytes.
func writeFixtureBank(bank regressionFixtureBank) error {
	if bank.SchemaVersion == "" {
		bank.SchemaVersion = fixtureBankSchemaVersion
	}
	bank.Fixtures = fixtureBankSortedByID(bank)
	if bank.Fixtures == nil {
		bank.Fixtures = []regressionFixture{}
	}

	preValidation, err := json.Marshal(bank)
	if err != nil {
		return fmt.Errorf("marshal fixture bank for validation: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(preValidation, &raw); err != nil {
		return fmt.Errorf("re-decode fixture bank for validation: %w", err)
	}
	violations, err := validateFixtureBankDocument(raw)
	if err != nil {
		return fmt.Errorf("validate fixture bank before write: %w", err)
	}
	if len(violations) > 0 {
		return fmt.Errorf("refusing to write an invalid fixture bank: %s", strings.Join(violations, "; "))
	}

	root, err := fixtureBankRepoRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(root, fixtureBankPath)

	encoded, err := json.MarshalIndent(bank, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal fixture bank: %w", err)
	}
	encoded = append(encoded, '\n')

	if existing, readErr := os.ReadFile(path); readErr == nil && bytes.Equal(existing, encoded) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir for fixture bank: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil { // nolint:gosec -- committed testdata, not secret
		return fmt.Errorf("write fixture bank %s: %w", path, err)
	}
	return nil
}

// retireFixture sets the retirement fields on the fixture named id and
// never removes it from the bank. A retirement naming a successor that
// does not exist in the bank is refused by name.
func retireFixture(id, successorID, reason string) error {
	id = strings.TrimSpace(id)
	successorID = strings.TrimSpace(successorID)
	reason = strings.TrimSpace(reason)
	if id == "" {
		return fmt.Errorf("retire fixture requires a non-empty id")
	}
	if successorID == "" {
		return fmt.Errorf("retire fixture %q requires a non-empty successor id -- a retirement never deletes, it names what replaces it", id)
	}
	if reason == "" {
		return fmt.Errorf("retire fixture %q requires a non-empty reason", id)
	}

	bank, err := loadFixtureBank()
	if err != nil {
		return err
	}

	var target *regressionFixture
	successorExists := false
	for i := range bank.Fixtures {
		if bank.Fixtures[i].ID == id {
			target = &bank.Fixtures[i]
		}
		if bank.Fixtures[i].ID == successorID {
			successorExists = true
		}
	}
	if target == nil {
		return fmt.Errorf("retire fixture: %q does not exist in the bank", id)
	}
	if !successorExists {
		return fmt.Errorf("retire fixture %q: successor %q does not exist in the bank", id, successorID)
	}

	target.RetiredBy = reason
	target.SuccessorID = successorID

	return writeFixtureBank(bank)
}
