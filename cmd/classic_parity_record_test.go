package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// classicParitySlice identifies one signed restoration slice -- a routed
// capability ("CAP-###") row from a phase's own <phase>-CLASSIC-COVERAGE.json
// -- that PROOF-03's parity record must carry a row for.
type classicParitySlice struct {
	Phase string
	ID    string
}

// classicParityDimension carries one of PROOF-03's four verification-contract
// dimensions (Outcome, Behavior, Experience, Safety) for one parity row.
// Either Evidence+PublicPath is populated (the dimension is evidenced), or
// NotEvidencedReason is populated (the dimension is honestly recorded as not
// evidenced), never both, never neither.
type classicParityDimension struct {
	Evidence           string
	PublicPath         string
	NotEvidencedReason string
}

// classicParityRow is one row of the parity record: a phase, the one or more
// slice identifiers it credits (more than one when two slices genuinely
// share the same piece of evidence), and its four dimensions.
type classicParityRow struct {
	Phase      string
	SliceIDs   []string
	Dimensions map[string]classicParityDimension
}

// classicParityRecordPathInRoot returns the fixed path to the PROOF-03
// parity record this plan produces.
func classicParityRecordPathInRoot(root string) string {
	return filepath.Join(root, ".planning", "phases", "205-owner-acceptance-and-restoration-seal", "205-PARITY.md")
}

// classicParityDimensionNames derives the four dimension names by parsing
// the shared synthesis template's own "## 7. Verification contract" table --
// the "Dimension" column -- never a hand-typed list in this file. It reuses
// classicSynthesisTemplateHeadingPattern (cmd/classic_synthesis_audit_test.go)
// to locate the section and splitClassicCoverageTableRow /
// stripClassicCoverageTableCell (cmd/classic_coverage_ratchet_test.go) to
// parse its table rows.
func classicParityDimensionNames(root string) ([]string, error) {
	templatePath := filepath.Join(root, ".planning", "research", "v1.28-classic-synthesis-template.md")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read synthesis template %s: %w", templatePath, err)
	}
	lines := strings.Split(string(data), "\n")
	const wantSection = "Verification contract"
	startIdx := -1
	endIdx := len(lines)
	for i, line := range lines {
		m := classicSynthesisTemplateHeadingPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if startIdx == -1 {
			if m[2] == wantSection {
				startIdx = i
			}
			continue
		}
		endIdx = i
		break
	}
	if startIdx == -1 {
		return nil, fmt.Errorf("section %q not found in template %s", wantSection, templatePath)
	}

	var names []string
	for _, line := range lines[startIdx+1 : endIdx] {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := splitClassicCoverageTableRow(trimmed)
		if len(cells) == 0 {
			continue
		}
		first := stripClassicCoverageTableCell(cells[0])
		if first == "" || first == "Dimension" || strings.Trim(first, "-") == "" {
			continue // header row or separator row
		}
		names = append(names, first)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no dimension names parsed from the %q section of %s", wantSection, templatePath)
	}
	return names, nil
}

// classicParitySliceIDs returns the union of signed slice identifiers across
// every EXISTING <phase>-CLASSIC-COVERAGE.json file under .planning/phases/,
// discovered from phase directories actually present on disk (reusing
// classicSynthesisPhaseDirPattern, the same Phase 199-204 corpus boundary
// SYNTH-07's own audit uses) -- never a hand-typed phase list. A phase
// directory that has no coverage file yet is skipped, not an error: not
// every Phase 199-204 phase has signed its coverage rows at every point in
// time this command might run. Results are ordered ascending phase then
// ascending identifier.
func classicParitySliceIDs(root string) ([]classicParitySlice, error) {
	phasesDir := filepath.Join(root, ".planning", "phases")
	entries, err := os.ReadDir(phasesDir)
	if err != nil {
		return nil, fmt.Errorf("read phases directory %s: %w", phasesDir, err)
	}

	type candidate struct {
		phase   string
		num     float64
		dirName string
	}
	var candidates []candidate
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m := classicSynthesisPhaseDirPattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		phase := m[1] + m[2]
		num, err := strconv.ParseFloat(phase, 64)
		if err != nil {
			return nil, fmt.Errorf("parse phase number from directory %q: %w", e.Name(), err)
		}
		candidates = append(candidates, candidate{phase: phase, num: num, dirName: e.Name()})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].num < candidates[j].num })

	var result []classicParitySlice
	for _, c := range candidates {
		jsonPath := filepath.Join(phasesDir, c.dirName, c.phase+"-CLASSIC-COVERAGE.json")
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // this phase has not signed its coverage rows yet
			}
			return nil, fmt.Errorf("read coverage file %s: %w", jsonPath, err)
		}
		var doc classicCoverageDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("decode coverage file %s: %w", jsonPath, err)
		}
		var ids []string
		for _, row := range doc.Rows {
			if row.Type != "GOAL" {
				continue // parity is over restored capability slices, not REQ/RESEARCH/CONTEXT rows
			}
			ids = append(ids, row.ID)
		}
		sort.Strings(ids)
		for _, id := range ids {
			result = append(result, classicParitySlice{Phase: c.phase, ID: id})
		}
	}
	return result, nil
}

// --- parity record document format ---
//
// One "## Phase <phase>" heading per phase, ascending. Under it, one
// "### <id>[, <id>...]" heading per row -- more than one id when two slices
// genuinely share the same piece of evidence. Under each row heading, one
// "- **<Dimension>**: ..." bullet per dimension, either
// "Evidenced: <evidence> | Path: <public path>" or "Not evidenced: <reason>".

var classicParityPhaseHeadingPattern = regexp.MustCompile(`^## Phase (\S+)$`)
var classicParitySliceHeadingPattern = regexp.MustCompile(`^### (.+)$`)
var classicParityDimensionLinePattern = regexp.MustCompile(`^- \*\*(.+?)\*\*: (.+)$`)

const classicParityEvidencedPrefix = "Evidenced: "
const classicParityPathSeparator = " | Path: "
const classicParityNotEvidencedPrefix = "Not evidenced: "

// parseClassicParityDimensionCell parses one dimension bullet's text (the
// part after "- **Dimension**: ") into a classicParityDimension.
func parseClassicParityDimensionCell(text string) classicParityDimension {
	if strings.HasPrefix(text, classicParityEvidencedPrefix) {
		rest := strings.TrimPrefix(text, classicParityEvidencedPrefix)
		if idx := strings.Index(rest, classicParityPathSeparator); idx != -1 {
			return classicParityDimension{
				Evidence:   strings.TrimSpace(rest[:idx]),
				PublicPath: strings.TrimSpace(rest[idx+len(classicParityPathSeparator):]),
			}
		}
		return classicParityDimension{Evidence: strings.TrimSpace(rest)}
	}
	if strings.HasPrefix(text, classicParityNotEvidencedPrefix) {
		return classicParityDimension{NotEvidencedReason: strings.TrimSpace(strings.TrimPrefix(text, classicParityNotEvidencedPrefix))}
	}
	return classicParityDimension{}
}

// loadClassicParityRecord parses the parity record document at its fixed
// path into rows. Read-only: never writes to the document.
func loadClassicParityRecord(root string) ([]classicParityRow, error) {
	path := classicParityRecordPathInRoot(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read parity record %s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")

	var rows []classicParityRow
	currentPhase := ""
	var current *classicParityRow
	flush := func() {
		if current != nil {
			rows = append(rows, *current)
			current = nil
		}
	}
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		if m := classicParityPhaseHeadingPattern.FindStringSubmatch(line); m != nil {
			flush()
			currentPhase = m[1]
			continue
		}
		if m := classicParitySliceHeadingPattern.FindStringSubmatch(line); m != nil {
			flush()
			var ids []string
			for _, part := range strings.Split(m[1], ", ") {
				part = strings.TrimSpace(part)
				if part != "" {
					ids = append(ids, part)
				}
			}
			current = &classicParityRow{Phase: currentPhase, SliceIDs: ids, Dimensions: map[string]classicParityDimension{}}
			continue
		}
		if m := classicParityDimensionLinePattern.FindStringSubmatch(line); m != nil {
			if current == nil {
				return nil, fmt.Errorf("dimension line %q in %s appears before any slice heading", line, path)
			}
			current.Dimensions[m[1]] = parseClassicParityDimensionCell(m[2])
			continue
		}
	}
	flush()
	if len(rows) == 0 {
		return nil, fmt.Errorf("no parity rows parsed from %s", path)
	}
	return rows, nil
}

// classicParityRowFirstID returns a row's primary (first-listed) slice id,
// used as the secondary sort key -- ascending phase, then ascending
// identifier, with the identifier breaking ties within a phase.
func classicParityRowFirstID(row classicParityRow) string {
	if len(row.SliceIDs) == 0 {
		return ""
	}
	return row.SliceIDs[0]
}

// renderClassicParityRecord renders rows as plain, deterministic text:
// ascending phase order, then ascending identifier within a phase. It is a
// pure function of rows and dimensions (no map iteration order dependency,
// no wall-clock input), so repeated calls with the same input are
// byte-identical.
func renderClassicParityRecord(rows []classicParityRow, dimensions []string) (string, error) {
	sorted := append([]classicParityRow(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi, erri := strconv.ParseFloat(sorted[i].Phase, 64)
		pj, errj := strconv.ParseFloat(sorted[j].Phase, 64)
		if erri == nil && errj == nil && pi != pj {
			return pi < pj
		}
		if sorted[i].Phase != sorted[j].Phase {
			return sorted[i].Phase < sorted[j].Phase
		}
		return classicParityRowFirstID(sorted[i]) < classicParityRowFirstID(sorted[j])
	})

	var b strings.Builder
	b.WriteString("# Phase 205: Restoration Parity Record\n\n")
	b.WriteString("PROOF-03's four-dimension parity ledger: every restored capability row (\"slice\") " +
		"from Phases 199-204 gets one entry naming, for useful outcome, actual behaviour, understandable " +
		"experience, and fail-closed safety, either real evidence naming a public command a person can " +
		"run, or an honest statement that no evidence exists yet and why. No percentage, endpoint count, " +
		"or aggregate score appears anywhere in this document.\n\n")

	currentPhase := ""
	for _, row := range sorted {
		if row.Phase != currentPhase {
			fmt.Fprintf(&b, "## Phase %s\n\n", row.Phase)
			currentPhase = row.Phase
		}
		fmt.Fprintf(&b, "### %s\n\n", strings.Join(row.SliceIDs, ", "))
		for _, dim := range dimensions {
			d := row.Dimensions[dim]
			if strings.TrimSpace(d.Evidence) != "" && strings.TrimSpace(d.PublicPath) != "" {
				fmt.Fprintf(&b, "- **%s**: Evidenced: %s | Path: %s\n", dim, d.Evidence, d.PublicPath)
			} else {
				fmt.Fprintf(&b, "- **%s**: Not evidenced: %s\n", dim, d.NotEvidencedReason)
			}
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

// --- public path resolution ---
//
// A dimension's evidence must name a public entry point a person could run
// -- a registered, non-hidden Aether CLI command ("aether <name>") or a
// managed Claude/OpenCode slash command ("/ant-<name>") that actually exists
// on disk -- read from the project's own command inventory at test time,
// never a hand-written allowlist.

var classicParityCLIPathPattern = regexp.MustCompile(`^aether\s+([a-z][a-z0-9-]*)`)
var classicParitySlashPathPattern = regexp.MustCompile(`^/ant-([a-z][a-z0-9-]*)`)

// classicParityCobraCommandNames walks the real, registered Cobra command
// tree (rootCmd, the same variable cmd/root.go builds and
// cmd/classic_command_parity_test.go's own classicAssertCobraPublicCommand
// already reads) and returns every non-hidden command and alias name. A
// fresh call always sees the current registration state.
func classicParityCobraCommandNames() map[string]bool {
	rootCmd.InitDefaultHelpCmd() // idempotent: guarantees "help" is registered regardless of test order
	names := map[string]bool{}
	var walk func(cmds []*cobra.Command)
	walk = func(cmds []*cobra.Command) {
		for _, c := range cmds {
			if c.Hidden {
				continue
			}
			names[c.Name()] = true
			for _, alias := range c.Aliases {
				names[alias] = true
			}
			walk(c.Commands())
		}
	}
	walk(rootCmd.Commands())
	return names
}

// classicParityPathIsPublic reports whether path names a real, currently
// registered public entry point: an "aether <command>" whose first word is a
// non-hidden Cobra command name, or a "/ant-<name>" whose managed Claude
// slash-command file exists on disk. Anything else -- an internal Go
// file/function reference, a bare description, an unregistered word -- is
// not public.
func classicParityPathIsPublic(root, path string) bool {
	trimmed := stripClassicCoverageTableCell(path)
	if m := classicParityCLIPathPattern.FindStringSubmatch(trimmed); m != nil {
		return classicParityCobraCommandNames()[m[1]]
	}
	if m := classicParitySlashPathPattern.FindStringSubmatch(trimmed); m != nil {
		candidate := filepath.Join(root, ".claude", "commands", "ant", m[1]+".md")
		info, err := os.Stat(candidate)
		return err == nil && !info.IsDir()
	}
	return false
}

// validateClassicParityRecord is the PROOF-03 gate: every known slice has
// exactly one row; every row's declared dimensions each carry either an
// evidence entry with a public path, or a non-empty not-evidenced reason; a
// dimension with neither fails naming the row and the dimension; a slice
// with no row fails naming the slice; a row for an unknown slice fails
// naming it; and an evidenced dimension whose public path does not resolve
// against the real command inventory fails naming the row and the dimension.
func validateClassicParityRecord(rows []classicParityRow, slices []classicParitySlice, dimensions []string, root string) error {
	knownPhase := map[string]string{}
	for _, s := range slices {
		knownPhase[s.ID] = s.Phase
	}

	var errs []string
	coveredBy := map[string][]int{}
	for i, row := range rows {
		rowLabel := fmt.Sprintf("%s/%s", row.Phase, strings.Join(row.SliceIDs, ","))
		for _, id := range row.SliceIDs {
			coveredBy[id] = append(coveredBy[id], i)
			if _, ok := knownPhase[id]; !ok {
				errs = append(errs, fmt.Sprintf("row %s cites unknown slice %q", rowLabel, id))
			}
		}
		for _, dim := range dimensions {
			d, ok := row.Dimensions[dim]
			if !ok {
				errs = append(errs, fmt.Sprintf("row %s: dimension %q is missing entirely", rowLabel, dim))
				continue
			}
			hasEvidence := strings.TrimSpace(d.Evidence) != "" && strings.TrimSpace(d.PublicPath) != ""
			hasReason := strings.TrimSpace(d.NotEvidencedReason) != ""
			if !hasEvidence && !hasReason {
				errs = append(errs, fmt.Sprintf("row %s: dimension %q has neither evidence with a public path nor a not-evidenced reason", rowLabel, dim))
				continue
			}
			if hasEvidence && !classicParityPathIsPublic(root, d.PublicPath) {
				errs = append(errs, fmt.Sprintf("row %s: dimension %q evidence path %q does not resolve to a public command, menu command, or screen", rowLabel, dim, d.PublicPath))
			}
		}
	}

	for _, s := range slices {
		idxs := coveredBy[s.ID]
		switch len(idxs) {
		case 0:
			errs = append(errs, fmt.Sprintf("slice %s (phase %s) has no parity row", s.ID, s.Phase))
		case 1:
			// exactly one row -- correct, including a shared row crediting
			// this slice alongside another
		default:
			errs = append(errs, fmt.Sprintf("slice %s is claimed by %d parity rows, want exactly one", s.ID, len(idxs)))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return errors.New(strings.Join(errs, "; "))
}

// --- Task 1: one phase, four dimensions, held by a command that fails on a blank ---

func TestClassicParityRecordCoversPhase203(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()

	allSlices, err := classicParitySliceIDs(root)
	if err != nil {
		t.Fatalf("discover signed slices: %v", err)
	}
	var phase203Slices []classicParitySlice
	for _, s := range allSlices {
		if s.Phase == "203" {
			phase203Slices = append(phase203Slices, s)
		}
	}
	if len(phase203Slices) == 0 {
		t.Fatal("expected Phase 203 to have signed slices")
	}

	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	var phase203Rows []classicParityRow
	for _, r := range rows {
		if r.Phase == "203" {
			phase203Rows = append(phase203Rows, r)
		}
	}

	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}
	if len(dimensions) != 4 {
		t.Fatalf("expected 4 dimension names parsed from the template, got %d: %v", len(dimensions), dimensions)
	}

	if err := validateClassicParityRecord(phase203Rows, phase203Slices, dimensions, root); err != nil {
		t.Fatal(err)
	}
}

func TestClassicParityBlankDimensionFails(t *testing.T) {
	// Positive control: a legitimate "Not evidenced" cell parses a real,
	// non-blank reason -- proving parseClassicParityDimensionCell's other
	// branch works before proving the blank branch fails validation.
	notEvidenced := parseClassicParityDimensionCell("Not evidenced: owner walk-through not yet recorded")
	if strings.TrimSpace(notEvidenced.NotEvidencedReason) == "" {
		t.Fatal("expected a 'Not evidenced: ...' cell to parse a non-empty reason")
	}

	root := findTestModuleRootForClassicCoverage199()
	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}
	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no rows loaded")
	}

	clone := rows[0]
	cloneDims := map[string]classicParityDimension{}
	for k, v := range clone.Dimensions {
		cloneDims[k] = v
	}
	clone.Dimensions = cloneDims
	targetDim := dimensions[0]
	clone.Dimensions[targetDim] = classicParityDimension{} // blank: no evidence, no path, no reason

	allSlices, err := classicParitySliceIDs(root)
	if err != nil {
		t.Fatalf("discover signed slices: %v", err)
	}
	var scopedSlices []classicParitySlice
	for _, s := range allSlices {
		for _, id := range clone.SliceIDs {
			if s.Phase == clone.Phase && s.ID == id {
				scopedSlices = append(scopedSlices, s)
			}
		}
	}

	err = validateClassicParityRecord([]classicParityRow{clone}, scopedSlices, dimensions, root)
	if err == nil {
		t.Fatal("expected validation to fail on a blank dimension")
	}
	for _, id := range clone.SliceIDs {
		if !strings.Contains(err.Error(), id) {
			t.Fatalf("error does not name slice %s: %v", id, err)
		}
	}
	if !strings.Contains(err.Error(), targetDim) {
		t.Fatalf("error does not name dimension %q: %v", targetDim, err)
	}
}

func TestClassicParityEvidenceNamesAPublicPath(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()

	// Positive controls: real, currently registered public entry points.
	if !classicParityPathIsPublic(root, "aether watch") {
		t.Fatal("expected `aether watch` to resolve as a public path")
	}
	if !classicParityPathIsPublic(root, "/ant-watch") {
		t.Fatal("expected `/ant-watch` to resolve as a public path")
	}

	// Negative control: an internal Go file/function is never a public path.
	if classicParityPathIsPublic(root, "cmd/pheromone_resolver.go:resolveEffectivePheromone") {
		t.Fatal("expected an internal function reference to NOT resolve as a public path")
	}

	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}
	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("no rows loaded")
	}

	clone := rows[0]
	cloneDims := map[string]classicParityDimension{}
	for k, v := range clone.Dimensions {
		cloneDims[k] = v
	}
	clone.Dimensions = cloneDims
	targetDim := dimensions[0]
	original := clone.Dimensions[targetDim]
	if strings.TrimSpace(original.Evidence) == "" {
		t.Fatalf("row's dimension %q has no real evidence to corrupt the path of", targetDim)
	}
	clone.Dimensions[targetDim] = classicParityDimension{
		Evidence:   original.Evidence,
		PublicPath: "cmd/pheromone_resolver.go:resolveEffectivePheromone",
	}

	allSlices, err := classicParitySliceIDs(root)
	if err != nil {
		t.Fatalf("discover signed slices: %v", err)
	}
	var scopedSlices []classicParitySlice
	for _, s := range allSlices {
		for _, id := range clone.SliceIDs {
			if s.Phase == clone.Phase && s.ID == id {
				scopedSlices = append(scopedSlices, s)
			}
		}
	}

	err = validateClassicParityRecord([]classicParityRow{clone}, scopedSlices, dimensions, root)
	if err == nil {
		t.Fatal("expected validation to fail when a dimension's evidence names an internal function rather than a public path")
	}
	for _, id := range clone.SliceIDs {
		if !strings.Contains(err.Error(), id) {
			t.Fatalf("error does not name slice %s: %v", id, err)
		}
	}
	if !strings.Contains(err.Error(), targetDim) {
		t.Fatalf("error does not name dimension %q: %v", targetDim, err)
	}
}

// --- Task 2: every slice, ordered, deduplicated, free of aggregate scoring ---

func TestClassicParityRecordCoversEverySignedSlice(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()

	slices, err := classicParitySliceIDs(root)
	if err != nil {
		t.Fatalf("discover signed slices: %v", err)
	}
	if len(slices) == 0 {
		t.Fatal("expected at least one signed slice")
	}

	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}

	if err := validateClassicParityRecord(rows, slices, dimensions, root); err != nil {
		t.Fatal(err)
	}
}

var classicParityRenderedPhaseHeaderPattern = regexp.MustCompile(`(?m)^## Phase (\S+)$`)
var classicParityRenderedSliceHeaderPattern = regexp.MustCompile(`(?m)^### (.+)$`)

func TestClassicParityRowsAreOrderedAndStable(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}

	first, err := renderClassicParityRecord(rows, dimensions)
	if err != nil {
		t.Fatalf("render (first pass): %v", err)
	}
	second, err := renderClassicParityRecord(rows, dimensions)
	if err != nil {
		t.Fatalf("render (second pass): %v", err)
	}
	if first != second {
		t.Fatal("parity record rendering is not byte-identical across repeated runs")
	}

	phaseMatches := classicParityRenderedPhaseHeaderPattern.FindAllStringSubmatch(first, -1)
	if len(phaseMatches) == 0 {
		t.Fatal("no phase headers found in rendered output")
	}
	lastPhase := -1.0
	for _, m := range phaseMatches {
		num, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("phase header %q does not parse as a number: %v", m[1], err)
		}
		if num < lastPhase {
			t.Fatalf("phase headers are not in ascending order: %v encountered after a higher phase", m[1])
		}
		lastPhase = num
	}

	blocks := classicParityRenderedPhaseHeaderPattern.Split(first, -1)
	if len(blocks) != len(phaseMatches)+1 {
		t.Fatalf("unexpected block/header count mismatch: %d blocks, %d headers", len(blocks), len(phaseMatches))
	}
	for i := range phaseMatches {
		block := blocks[i+1]
		sliceMatches := classicParityRenderedSliceHeaderPattern.FindAllStringSubmatch(block, -1)
		if len(sliceMatches) == 0 {
			t.Fatalf("phase %s: no slice headers found in its rendered block", phaseMatches[i][1])
		}
		lastID := ""
		for _, sm := range sliceMatches {
			firstID := strings.SplitN(sm[1], ", ", 2)[0]
			if firstID < lastID {
				t.Fatalf("phase %s: slice headers are not in ascending identifier order (%q after %q)", phaseMatches[i][1], firstID, lastID)
			}
			lastID = firstID
		}
	}
}

func TestClassicParitySharedEvidenceIsCreditedOnce(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}
	slices, err := classicParitySliceIDs(root)
	if err != nil {
		t.Fatalf("discover signed slices: %v", err)
	}

	var shared *classicParityRow
	for i := range rows {
		if len(rows[i].SliceIDs) > 1 {
			shared = &rows[i]
			break
		}
	}
	if shared == nil {
		t.Fatal("expected at least one parity row citing more than one slice id (genuinely shared evidence), found none")
	}

	// The real, unmodified corpus must credit each shared id exactly once.
	if err := validateClassicParityRecord(rows, slices, dimensions, root); err != nil {
		t.Fatalf("real corpus with a shared-evidence row unexpectedly failed validation: %v", err)
	}

	// Duplicating the shared row means both its ids now appear on two rows
	// -- the same evidence claimed twice -- and must be rejected by name.
	duplicated := append([]classicParityRow(nil), rows...)
	duplicated = append(duplicated, *shared)
	err = validateClassicParityRecord(duplicated, slices, dimensions, root)
	if err == nil {
		t.Fatal("expected validation to fail when a shared-evidence row is duplicated (double credit)")
	}
	for _, id := range shared.SliceIDs {
		if !strings.Contains(err.Error(), id) {
			t.Fatalf("duplicated-row error does not name %s: %v", id, err)
		}
	}
}

func TestClassicParityRecordCarriesNoAggregateScore(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()
	path := classicParityRecordPathInRoot(root)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read parity record %s: %v", path, err)
	}
	text := string(data)

	if strings.Contains(text, "%") {
		t.Error("parity record contains a percent sign; it must report evidence, never a percentage")
	}
	for _, forbidden := range []string{"average", "Average", "aggregate", "Aggregate", "overall score", "mean score", "endpoint count", "passing count"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("parity record contains %q -- an averaged, aggregated, or count-based completion claim is prohibited (PROOF-03)", forbidden)
		}
	}

	rows, err := loadClassicParityRecord(root)
	if err != nil {
		t.Fatalf("load parity record: %v", err)
	}
	dimensions, err := classicParityDimensionNames(root)
	if err != nil {
		t.Fatalf("load dimension names: %v", err)
	}
	rendered, err := renderClassicParityRecord(rows, dimensions)
	if err != nil {
		t.Fatalf("render parity record: %v", err)
	}
	if strings.Contains(rendered, "%") {
		t.Error("rendered parity record contains a percent sign")
	}
}

// --- Task 3: prove the parity audit writes nothing ---

func TestClassicParityAuditDoesNotMutate(t *testing.T) {
	root := findTestModuleRootForClassicCoverage199()

	var filesToSnapshot []string
	filesToSnapshot = append(filesToSnapshot, classicParityRecordPathInRoot(root))
	filesToSnapshot = append(filesToSnapshot, filepath.Join(root, ".planning", "research", "v1.28-classic-synthesis-template.md"))

	slices, err := classicParitySliceIDs(root)
	if err != nil {
		t.Fatalf("discover signed slices: %v", err)
	}
	phasesSeen := map[string]bool{}
	for _, s := range slices {
		phasesSeen[s.Phase] = true
	}
	var phaseList []string
	for phase := range phasesSeen {
		phaseList = append(phaseList, phase)
	}
	sort.Strings(phaseList)
	for _, phase := range phaseList {
		dir, err := classicCoveragePhaseDir(root, phase)
		if err != nil {
			t.Fatalf("resolve phase directory for %s: %v", phase, err)
		}
		filesToSnapshot = append(filesToSnapshot,
			filepath.Join(dir, phase+"-CLASSIC-COVERAGE.json"),
			filepath.Join(dir, phase+"-CLASSIC-COVERAGE.md"),
		)
	}

	type snapshotEntry struct {
		modTime int64
		digest  [32]byte
	}
	snapshot := func() map[string]snapshotEntry {
		result := map[string]snapshotEntry{}
		for _, p := range filesToSnapshot {
			info, err := os.Stat(p)
			if err != nil {
				t.Fatalf("stat %s: %v", p, err)
			}
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read %s: %v", p, err)
			}
			result[p] = snapshotEntry{modTime: info.ModTime().UnixNano(), digest: sha256.Sum256(data)}
		}
		return result
	}

	runFullParityAudit := func() {
		dimensions, err := classicParityDimensionNames(root)
		if err != nil {
			t.Fatalf("load dimension names: %v", err)
		}
		allSlices, err := classicParitySliceIDs(root)
		if err != nil {
			t.Fatalf("discover signed slices: %v", err)
		}
		rows, err := loadClassicParityRecord(root)
		if err != nil {
			t.Fatalf("load parity record: %v", err)
		}
		if err := validateClassicParityRecord(rows, allSlices, dimensions, root); err != nil {
			t.Fatalf("full parity audit unexpectedly failed: %v", err)
		}
		if _, err := renderClassicParityRecord(rows, dimensions); err != nil {
			t.Fatalf("render parity record: %v", err)
		}
	}

	before := snapshot()
	runFullParityAudit()
	after1 := snapshot()
	runFullParityAudit()
	after2 := snapshot()

	if !reflect.DeepEqual(before, after1) {
		t.Fatalf("parity audit mutated a file on its first run: before=%+v after=%+v", before, after1)
	}
	if !reflect.DeepEqual(after1, after2) {
		t.Fatalf("parity audit mutated a file on its second run: after1=%+v after2=%+v", after1, after2)
	}
	t.Logf("snapshotted %d files across two full parity audit runs; all unchanged", len(filesToSnapshot))
}
