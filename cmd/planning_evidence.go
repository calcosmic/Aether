package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
)

// planningEvidenceRefusalCode is a closed machine-readable reason why a
// source could not become a trustworthy evidence reference. A refusal never
// returns a partially populated reference.
type planningEvidenceRefusalCode string

const (
	planningEvidenceRefusalUnsupportedKind          planningEvidenceRefusalCode = "unsupported_kind"
	planningEvidenceRefusalPathOutsideApprovedRoots planningEvidenceRefusalCode = "path_outside_approved_roots"
	planningEvidenceRefusalMissingContent           planningEvidenceRefusalCode = "missing_content"
	planningEvidenceRefusalHashMismatch             planningEvidenceRefusalCode = "hash_mismatch"
	planningEvidenceRefusalInvalidMetadata          planningEvidenceRefusalCode = "invalid_metadata"
	planningEvidenceRefusalInvalidDimension         planningEvidenceRefusalCode = "invalid_dimension"
	planningEvidenceRefusalReadFailed               planningEvidenceRefusalCode = "read_failed"
)

// planningEvidenceRefusal is deliberately safe to surface. Detail describes
// the failed contract without including source bodies or secret-shaped text.
type planningEvidenceRefusal struct {
	Code   planningEvidenceRefusalCode
	Kind   colony.PlanningEvidenceKind
	Origin string
	Detail string
}

func (r *planningEvidenceRefusal) Error() string {
	if r == nil {
		return "planning evidence refused"
	}
	detail := strings.TrimSpace(r.Detail)
	if detail == "" {
		detail = "source did not satisfy the evidence contract"
	}
	if origin := strings.TrimSpace(r.Origin); origin != "" {
		return fmt.Sprintf("planning evidence %s refused (%s): %s", origin, r.Code, detail)
	}
	return fmt.Sprintf("planning evidence refused (%s): %s", r.Code, detail)
}

// planningEvidenceScope binds evidence to the exact planning authority it may
// inform. Observing the same bytes under a different scope produces a distinct
// reference rather than universal permission.
type planningEvidenceScope struct {
	GoalID                  string
	SessionID               string
	SpecificationRevisionID string
	PlanRevisionID          string
}

// planningEvidenceSource is the transient input to the catalogue. Repository
// sources use RepositoryPath and are read only after all paths pass preflight;
// logical sources use Content directly. Raw Content is never retained.
type planningEvidenceSource struct {
	Kind                 colony.PlanningEvidenceKind
	Origin               string
	RepositoryPath       string
	Content              []byte
	ExpectedContentHash  string
	Scope                planningEvidenceScope
	SourceRevision       string
	ObservedAt           time.Time
	ApplicableDimensions []colony.PlanningDimension
}

// planningEvidenceLocator gives callers enough information to reopen the
// source through its owner without copying an unrestricted body into planning
// state or prompts.
type planningEvidenceLocator struct {
	Kind           colony.PlanningEvidenceKind `json:"kind"`
	Origin         string                      `json:"origin"`
	RepositoryPath string                      `json:"repository_path,omitempty"`
	SourceRevision string                      `json:"source_revision"`
}

// planningEvidenceRecord is the safe catalogue projection: a typed reference,
// a bounded/redacted summary, and locator metadata only.
type planningEvidenceRecord struct {
	Reference colony.PlanningEvidenceRef `json:"reference"`
	Summary   string                     `json:"summary"`
	Locator   planningEvidenceLocator    `json:"locator"`
}

type planningEvidenceCollectionRequest struct {
	RepositoryRoot string
	ApprovedRoots  []string
	Existing       []planningEvidenceRecord
	Sources        []planningEvidenceSource
	ReadFile       func(string) ([]byte, error)
}

type preparedPlanningEvidenceSource struct {
	source   planningEvidenceSource
	fullPath string
}

// collectPlanningEvidence builds one deterministic catalogue. It validates
// every repository path before the first read, preserves an existing exact
// reference (including its original observation/freshness), deduplicates by
// content address, and sorts by stable ID.
func collectPlanningEvidence(request planningEvidenceCollectionRequest) ([]planningEvidenceRecord, error) {
	root, err := canonicalPlanningEvidenceRoot(request.RepositoryRoot)
	if err != nil {
		return nil, err
	}

	prepared := make([]preparedPlanningEvidenceSource, 0, len(request.Sources))
	for _, source := range request.Sources {
		item := preparedPlanningEvidenceSource{source: source}
		if strings.TrimSpace(source.RepositoryPath) != "" {
			rel, fullPath, pathErr := resolvePlanningEvidencePath(root, request.ApprovedRoots, source.RepositoryPath)
			if pathErr != nil {
				return nil, pathErr
			}
			item.source.RepositoryPath = rel
			item.source.Origin = rel
			item.fullPath = fullPath
		}
		prepared = append(prepared, item)
	}

	readFile := request.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}

	byID := make(map[string]planningEvidenceRecord, len(request.Existing)+len(prepared))
	existing := make(map[string]struct{}, len(request.Existing))
	for _, record := range request.Existing {
		if err := validateExistingPlanningEvidenceRecord(record); err != nil {
			return nil, err
		}
		if _, duplicate := byID[record.Reference.ID]; duplicate {
			continue
		}
		byID[record.Reference.ID] = record
		existing[record.Reference.ID] = struct{}{}
	}

	for _, item := range prepared {
		source := item.source
		if item.fullPath != "" {
			if source.Content != nil {
				return nil, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "repository evidence must be read from its validated path, not supplied twice")
			}
			content, readErr := readFile(item.fullPath)
			if readErr != nil {
				return nil, planningEvidenceRefusalFor(source, planningEvidenceRefusalReadFailed, "could not read the validated repository source")
			}
			source.Content = content
		}
		record, normalizeErr := normalizePlanningEvidence(source)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if _, wasExisting := existing[record.Reference.ID]; wasExisting {
			continue
		}
		if prior, duplicate := byID[record.Reference.ID]; duplicate {
			// Earliest observation is the original for exact duplicates. Reread
			// time is intentionally excluded from the evidence address.
			if record.Reference.ObservedAt.Before(prior.Reference.ObservedAt) {
				byID[record.Reference.ID] = record
			}
			continue
		}
		byID[record.Reference.ID] = record
	}

	records := make([]planningEvidenceRecord, 0, len(byID))
	for _, record := range byID {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Reference.ID < records[j].Reference.ID
	})
	return records, nil
}

// normalizePlanningEvidence turns an already loaded source into a stable
// content address. Observation time is metadata, not identity, so rereading
// unchanged bytes cannot manufacture freshness.
func normalizePlanningEvidence(source planningEvidenceSource) (planningEvidenceRecord, error) {
	if !source.Kind.Valid() {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalUnsupportedKind, fmt.Sprintf("kind %q is not supported", source.Kind))
	}

	origin := strings.TrimSpace(source.Origin)
	repositoryPath := strings.TrimSpace(source.RepositoryPath)
	if repositoryPath != "" {
		clean, err := canonicalRepositoryRelativePath(repositoryPath)
		if err != nil {
			return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalPathOutsideApprovedRoots, err.Error())
		}
		repositoryPath = clean
		origin = clean
	}
	if origin == "" || strings.ContainsRune(origin, '\x00') {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "origin is required and cannot contain NUL")
	}
	sourceRevision := strings.TrimSpace(source.SourceRevision)
	if sourceRevision == "" {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "source revision is required")
	}
	if source.ObservedAt.IsZero() {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "observation time is required")
	}

	scope := planningEvidenceScope{
		GoalID:                  strings.TrimSpace(source.Scope.GoalID),
		SessionID:               strings.TrimSpace(source.Scope.SessionID),
		SpecificationRevisionID: strings.TrimSpace(source.Scope.SpecificationRevisionID),
		PlanRevisionID:          strings.TrimSpace(source.Scope.PlanRevisionID),
	}
	if scope.GoalID == "" || scope.SessionID == "" || scope.SpecificationRevisionID == "" || scope.PlanRevisionID == "" {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "goal, session, specification revision, and plan revision scope are required")
	}

	dimensions, err := normalizePlanningEvidenceDimensions(source.ApplicableDimensions)
	if err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidDimension, err.Error())
	}
	content, err := normalizePlanningEvidenceContent(source.Content)
	if err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalMissingContent, err.Error())
	}

	identity := struct {
		SchemaVersion           string                      `json:"schema_version"`
		Kind                    colony.PlanningEvidenceKind `json:"kind"`
		Origin                  string                      `json:"origin"`
		RepositoryPath          string                      `json:"repository_path,omitempty"`
		GoalID                  string                      `json:"goal_id"`
		SessionID               string                      `json:"session_id"`
		SpecificationRevisionID string                      `json:"specification_revision_id"`
		PlanRevisionID          string                      `json:"plan_revision_id"`
		SourceRevision          string                      `json:"source_revision"`
		ApplicableDimensions    []colony.PlanningDimension  `json:"applicable_dimensions"`
		Content                 string                      `json:"content"`
	}{
		SchemaVersion:           colony.PlanningEvidenceSchemaVersion,
		Kind:                    source.Kind,
		Origin:                  origin,
		RepositoryPath:          repositoryPath,
		GoalID:                  scope.GoalID,
		SessionID:               scope.SessionID,
		SpecificationRevisionID: scope.SpecificationRevisionID,
		PlanRevisionID:          scope.PlanRevisionID,
		SourceRevision:          sourceRevision,
		ApplicableDimensions:    dimensions,
		Content:                 content,
	}
	encoded, err := json.Marshal(identity)
	if err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, "could not encode stable source identity")
	}
	contentHash := planningEvidenceSHA256(encoded)
	if expected := strings.TrimSpace(source.ExpectedContentHash); expected != "" && expected != contentHash {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalHashMismatch, "claimed content hash does not match normalized source content and identity")
	}

	summary := summarizePlanningEvidence(source.Kind, origin, content, contentHash)
	reference := colony.PlanningEvidenceRef{
		SchemaVersion:           colony.PlanningEvidenceSchemaVersion,
		ID:                      fmt.Sprintf("evidence-%s-%s", source.Kind, contentHash[:12]),
		ContentHash:             contentHash,
		Kind:                    source.Kind,
		Origin:                  origin,
		RepositoryPath:          repositoryPath,
		GoalID:                  scope.GoalID,
		SessionID:               scope.SessionID,
		SpecificationRevisionID: scope.SpecificationRevisionID,
		PlanRevisionID:          scope.PlanRevisionID,
		SourceRevision:          sourceRevision,
		ObservedAt:              source.ObservedAt.UTC(),
		ExcerptDigest:           planningEvidenceSHA256([]byte(summary)),
		ApplicableDimensions:    dimensions,
		Fresh:                   true,
		Admissible:              true,
		AdmissibilityReason:     "current scoped source collected",
	}
	if err := reference.Validate(); err != nil {
		return planningEvidenceRecord{}, planningEvidenceRefusalFor(source, planningEvidenceRefusalInvalidMetadata, err.Error())
	}
	return planningEvidenceRecord{
		Reference: reference,
		Summary:   summary,
		Locator: planningEvidenceLocator{
			Kind:           source.Kind,
			Origin:         origin,
			RepositoryPath: repositoryPath,
			SourceRevision: sourceRevision,
		},
	}, nil
}

func canonicalPlanningEvidenceRoot(raw string) (string, error) {
	root := strings.TrimSpace(raw)
	if root == "" {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root is required"}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root could not be resolved"}
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root could not be resolved"}
	}
	info, err := os.Stat(real)
	if err != nil || !info.IsDir() {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Detail: "repository root must be an existing directory"}
	}
	return filepath.Clean(real), nil
}

// resolvePlanningEvidencePath performs lexical, approved-root, symlink, and
// regular-file checks before the collector invokes ReadFile.
func resolvePlanningEvidencePath(root string, approvedRoots []string, raw string) (string, string, error) {
	rel, err := canonicalRepositoryRelativePath(raw)
	if err != nil {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: strings.TrimSpace(raw), Detail: err.Error()}
	}
	if len(approvedRoots) == 0 {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository source has no approved root"}
	}

	fullPath := filepath.Join(root, filepath.FromSlash(rel))
	approved := false
	for _, rawApprovedRoot := range approvedRoots {
		approvedRel, approvedErr := canonicalApprovedEvidenceRoot(rawApprovedRoot)
		if approvedErr != nil {
			return "", "", approvedErr
		}
		approvedPath := filepath.Join(root, filepath.FromSlash(approvedRel))
		if pathContainedBy(approvedPath, fullPath) {
			approved = true
			break
		}
	}
	if !approved || !pathContainedBy(root, fullPath) {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository path is outside the approved evidence roots"}
	}

	info, err := os.Lstat(fullPath)
	if err != nil {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalReadFailed, Origin: rel, Detail: "repository evidence path cannot be inspected"}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository evidence must be a regular non-symlink file"}
	}
	realPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil || !pathContainedBy(root, realPath) {
		return "", "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: rel, Detail: "repository evidence resolves outside the repository"}
	}
	return rel, realPath, nil
}

func canonicalRepositoryRelativePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("repository path is required")
	}
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("repository path must be relative")
	}
	clean := filepath.Clean(filepath.FromSlash(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("repository path escapes its root")
	}
	return filepath.ToSlash(clean), nil
}

func canonicalApprovedEvidenceRoot(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "." {
		return ".", nil
	}
	clean, err := canonicalRepositoryRelativePath(raw)
	if err != nil {
		return "", &planningEvidenceRefusal{Code: planningEvidenceRefusalPathOutsideApprovedRoots, Origin: raw, Detail: "approved root must be repository-relative"}
	}
	return clean, nil
}

func normalizePlanningEvidenceContent(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("source content is required")
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("source content must be UTF-8 text")
	}
	content := strings.TrimPrefix(string(raw), "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	content = strings.TrimSpace(strings.Join(lines, "\n"))
	if content == "" {
		return "", fmt.Errorf("source content is required")
	}
	return content, nil
}

func normalizePlanningEvidenceDimensions(values []colony.PlanningDimension) ([]colony.PlanningDimension, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one applicable dimension claim is required")
	}
	claimed := make(map[colony.PlanningDimension]struct{}, len(values))
	for _, value := range values {
		if !value.Valid() {
			return nil, fmt.Errorf("invalid planning dimension %q", value)
		}
		claimed[value] = struct{}{}
	}
	result := make([]colony.PlanningDimension, 0, len(claimed))
	for _, value := range colony.PlanningDimensions() {
		if _, ok := claimed[value]; ok {
			result = append(result, value)
		}
	}
	return result, nil
}

func summarizePlanningEvidence(kind colony.PlanningEvidenceKind, origin, content, contentHash string) string {
	fallback := fmt.Sprintf("%s evidence at %s (content redacted; sha256 %s)", kind, origin, contentHash[:12])
	scan := privacyScan(content)
	if scan.Blocked {
		return fallback
	}
	clean := strings.Join(strings.Fields(scan.Clean), " ")
	const maxSummaryRunes = 240
	runes := []rune(clean)
	if len(runes) > maxSummaryRunes {
		clean = string(runes[:maxSummaryRunes]) + "…"
	}
	sanitized, err := colony.SanitizeSignalContent(clean)
	if err != nil || strings.TrimSpace(sanitized) == "" {
		return fallback
	}
	return sanitized
}

func validateExistingPlanningEvidenceRecord(record planningEvidenceRecord) error {
	if err := record.Reference.Validate(); err != nil {
		return &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Origin: record.Reference.Origin, Kind: record.Reference.Kind, Detail: "existing evidence reference is invalid"}
	}
	if !planningSHA256Pattern.MatchString(record.Reference.ContentHash) || !strings.HasSuffix(record.Reference.ID, "-"+record.Reference.ContentHash[:12]) {
		return &planningEvidenceRefusal{Code: planningEvidenceRefusalHashMismatch, Origin: record.Reference.Origin, Kind: record.Reference.Kind, Detail: "existing evidence reference is not content-addressed"}
	}
	if strings.TrimSpace(record.Summary) == "" || strings.TrimSpace(record.Locator.Origin) == "" {
		return &planningEvidenceRefusal{Code: planningEvidenceRefusalInvalidMetadata, Origin: record.Reference.Origin, Kind: record.Reference.Kind, Detail: "existing evidence record lacks safe summary or locator"}
	}
	return nil
}

func planningEvidenceRefusalFor(source planningEvidenceSource, code planningEvidenceRefusalCode, detail string) error {
	origin := strings.TrimSpace(source.Origin)
	if strings.TrimSpace(source.RepositoryPath) != "" {
		origin = strings.TrimSpace(source.RepositoryPath)
	}
	return &planningEvidenceRefusal{Code: code, Kind: source.Kind, Origin: origin, Detail: detail}
}

func planningEvidenceSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
