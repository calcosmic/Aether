package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestCompletionPacketSchemaMatchesStructs is the drift gate (D-04): the
// committed .aether/schemas/completion-packet.schema.json must be
// byte-identical to what generateCompletionPacketSchemaBytes() produces
// right now. Any struct field added, removed, or retagged on
// codexExternalBuildCompletion (or anything it pulls in transitively)
// without regenerating the committed file fails this test.
func TestCompletionPacketSchemaMatchesStructs(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	committedPath := filepath.Join(repoRoot, completionPacketSchemaRel)
	committed, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatalf("failed to read committed schema %s: %v", completionPacketSchemaRel, err)
	}

	generated, err := generateCompletionPacketSchemaBytes()
	if err != nil {
		t.Fatalf("failed to regenerate the completion-packet schema: %v", err)
	}

	if !bytes.Equal(generated, committed) {
		line := firstDifferingLine(generated, committed)
		t.Fatalf("completion-packet schema drift: %s no longer matches the Go structs (first differing line: %d); regenerate with `aether contract-schema --write`", completionPacketSchemaRel, line)
	}
}

// TestContractDocExampleValidatesAgainstSchema proves the shipped handoff
// contract doc's worked example is not aspirational prose — it validates
// against the generated schema's handoff definition. The fenced ```json
// block is located by scanning for fence markers, not a hard-coded line
// number, so the test survives the doc moving.
func TestContractDocExampleValidatesAgainstSchema(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	contractPath := filepath.Join(repoRoot, ".aether", "references", "contracts", "worker-handoff-contract.md")
	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", contractPath, err)
	}

	example := extractFencedTestBlock(t, string(content), "json", contractPath)

	var handoff any
	if err := json.Unmarshal([]byte(example), &handoff); err != nil {
		t.Fatalf("failed to parse worked example JSON from %s: %v\nexample:\n%s", contractPath, err, example)
	}

	// The schema exposes handoff only as a nested property of a worker
	// result, so validate by wrapping the example in a minimal packet and
	// asserting no violation lands under /dispatches/0/handoff. name and
	// status are the only required worker-result fields.
	packet := map[string]any{
		"dispatches": []any{
			map[string]any{
				"name":    "worked-example-worker",
				"status":  "completed",
				"handoff": handoff,
			},
		},
	}

	for _, v := range validateCompletionPacketStructure(packet) {
		if strings.HasPrefix(v.Field, "/dispatches/0/handoff") {
			t.Fatalf("worked example in %s failed schema validation at %s (%s): %s; the doc example and the schema have drifted apart", contractPath, v.Field, v.Rule, v.Message)
		}
	}
}

// TestWrapperFieldListMatchesSchema is the four-surface parity invariant
// (D-04): the handoff field set must be exactly equal across the schema's
// WorkerHandoff $defs properties, the brace-list in both wrapper build.md
// files, and codex.HandoffFieldsSummary in pkg/codex/handoff.go -- the one
// canonical constant renderResponseContract (native-Codex dispatch),
// composeBuildManifestBrief (wrapper-facing build brief), and continue's
// external reviewer briefs all reference by symbol (189-01/189-02 D-03), so
// checking the constant's own declared value is equivalent to checking every
// one of its readers at once. It also asserts every field build.md requires
// in a terminal worker result exists as a property of the schema's
// worker-result definition.
func TestWrapperFieldListMatchesSchema(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeBuildMD := filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md")
	opencodeBuildMD := filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md")
	handoffConstPath := filepath.Join(repoRoot, "pkg", "codex", "handoff.go")

	schemaFields := schemaDefinitionPropertyNames(t, "WorkerHandoff")
	claudeFields := wrapperHandoffBraceListFields(t, claudeBuildMD)
	opencodeFields := wrapperHandoffBraceListFields(t, opencodeBuildMD)
	handoffConstFields := handoffFieldsSummaryConstFields(t, handoffConstPath)

	assertFieldSetsEqual(t, "schema $defs.WorkerHandoff vs "+claudeBuildMD, schemaFields, claudeFields)
	assertFieldSetsEqual(t, "schema $defs.WorkerHandoff vs "+opencodeBuildMD, schemaFields, opencodeFields)
	assertFieldSetsEqual(t, "schema $defs.WorkerHandoff vs "+handoffConstPath, schemaFields, handoffConstFields)

	requiredResultFields := terminalResultRequiredFields(t, claudeBuildMD)
	resultProperties := schemaDefinitionPropertyNames(t, "codexExternalBuildWorkerResult")
	resultPropertySet := make(map[string]bool, len(resultProperties))
	for _, p := range resultProperties {
		resultPropertySet[p] = true
	}
	for _, field := range requiredResultFields {
		if !resultPropertySet[field] {
			t.Fatalf("%s requires terminal-result field %q, but it is not a property of the schema's codexExternalBuildWorkerResult definition", claudeBuildMD, field)
		}
	}
}

// schemaDefinitionPropertyNames returns the sorted set of property names
// under $defs.<name>.properties in the freshly-generated completion-packet
// schema. Fails loudly if the definition or its properties cannot be found.
func schemaDefinitionPropertyNames(t *testing.T, defName string) []string {
	t.Helper()

	raw, err := generateCompletionPacketSchemaBytes()
	if err != nil {
		t.Fatalf("failed to generate completion-packet schema: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("failed to parse generated completion-packet schema: %v", err)
	}

	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		t.Fatalf("generated completion-packet schema has no $defs map")
	}

	def, ok := defs[defName].(map[string]any)
	if !ok {
		t.Fatalf("generated completion-packet schema has no $defs.%s definition", defName)
	}

	props, ok := def["properties"].(map[string]any)
	if !ok {
		t.Fatalf("generated completion-packet schema's $defs.%s definition has no properties map", defName)
	}

	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	if len(names) == 0 {
		t.Fatalf("parsed zero property names from $defs.%s in the generated completion-packet schema", defName)
	}
	sort.Strings(names)
	return names
}

// handoffBraceListPattern matches a backtick-quoted brace list, e.g.
// `{changed_files, commands_run, verification_status}`.
var handoffBraceListPattern = regexp.MustCompile("`\\{([^}]+)\\}`")

// wrapperHandoffBraceListFields locates the line in a wrapper build.md that
// mentions "handoff" and contains a backtick-quoted brace list, and returns
// the sorted set of field names inside it.
func wrapperHandoffBraceListFields(t *testing.T, path string) []string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		if !strings.Contains(strings.ToLower(line), "handoff") {
			continue
		}
		match := handoffBraceListPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		fields := splitFieldTokens(match[1])
		if len(fields) == 0 {
			t.Fatalf("parsed zero fields from the handoff brace list in %s", path)
		}
		return fields
	}

	t.Fatalf("no handoff brace-list declaration (`{field, field, ...}`) found in %s", path)
	return nil
}

// handoffFieldsSummaryConstPattern matches the declaration of
// codex.HandoffFieldsSummary in pkg/codex/handoff.go, capturing the field
// list up to the parenthetical freshness explanation. This constant is the
// one canonical source every reader (renderResponseContract,
// composeBuildManifestBrief, continue's external reviewer briefs) references
// by symbol rather than hand-copying, per 189-01/189-02 D-03.
var handoffFieldsSummaryConstPattern = regexp.MustCompile(`HandoffFieldsSummary\s*=\s*"([^(\n]+)\(`)

// handoffFieldsSummaryConstFields extracts the handoff field list from
// codex.HandoffFieldsSummary's own declared value in pkg/codex/handoff.go.
func handoffFieldsSummaryConstFields(t *testing.T, path string) []string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	match := handoffFieldsSummaryConstPattern.FindSubmatch(content)
	if match == nil {
		t.Fatalf("no \"HandoffFieldsSummary = ...\" constant declaration found in %s", path)
	}

	fields := splitFieldTokens(string(match[1]))
	if len(fields) == 0 {
		t.Fatalf("parsed zero fields from the HandoffFieldsSummary constant in %s", path)
	}
	return fields
}

// splitFieldTokens splits a comma-separated field list (optionally ending
// in "and <last field>") into a sorted, de-duplicated set of bare field
// names, stripping backticks and surrounding prose punctuation.
func splitFieldTokens(s string) []string {
	parts := strings.Split(s, ",")
	seen := make(map[string]bool, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "and ")
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "`")
		p = strings.TrimSuffix(p, ".")
		if p == "" {
			continue
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// terminalResultRequirementPattern locates the "Require terminal structured
// result with:" bullet line.
var terminalResultRequirementPattern = regexp.MustCompile("Require terminal structured result with")

// backtickTokenPattern matches a single backtick-quoted snake_case token,
// e.g. `files_created`.
var backtickTokenPattern = regexp.MustCompile("`([a-z_]+)`")

// terminalResultRequiredFields extracts the backtick-quoted field names
// from build.md's "Require terminal structured result with:" bullet.
func terminalResultRequiredFields(t *testing.T, path string) []string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		if !terminalResultRequirementPattern.MatchString(line) {
			continue
		}
		matches := backtickTokenPattern.FindAllStringSubmatch(line, -1)
		fields := make([]string, 0, len(matches))
		for _, m := range matches {
			fields = append(fields, m[1])
		}
		if len(fields) == 0 {
			t.Fatalf("parsed zero fields from the terminal-result requirement line in %s", path)
		}
		sort.Strings(fields)
		return fields
	}

	t.Fatalf("no \"Require terminal structured result with:\" line found in %s", path)
	return nil
}

// extractFencedTestBlock scans content line-by-line for a fenced block opened
// by "```<lang>" and closed by "```", returning its contents. Scanning
// instead of hard-coded line numbers means the test survives the doc
// moving. Fails loudly (never returns an empty string silently) if no
// fenced block is found.
func extractFencedTestBlock(t *testing.T, content, lang, sourcePath string) string {
	t.Helper()

	fenceStart := "```" + lang
	lines := strings.Split(content, "\n")
	start := -1
	end := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start == -1 {
			if trimmed == fenceStart {
				start = i + 1
			}
			continue
		}
		if trimmed == "```" {
			end = i
			break
		}
	}

	if start == -1 || end == -1 {
		t.Fatalf("no fenced ```%s block found in %s", lang, sourcePath)
	}

	block := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if block == "" {
		t.Fatalf("fenced ```%s block in %s is empty", lang, sourcePath)
	}
	return block
}

// assertFieldSetsEqual compares two sorted field-name slices for exact
// set equality, reporting the symmetric difference (what's missing from
// b, what's extra in b) so the fix is obvious.
func assertFieldSetsEqual(t *testing.T, label string, a, b []string) {
	t.Helper()

	aSet := make(map[string]bool, len(a))
	for _, x := range a {
		aSet[x] = true
	}
	bSet := make(map[string]bool, len(b))
	for _, x := range b {
		bSet[x] = true
	}

	var missingFromB, extraInB []string
	for _, x := range a {
		if !bSet[x] {
			missingFromB = append(missingFromB, x)
		}
	}
	for _, x := range b {
		if !aSet[x] {
			extraInB = append(extraInB, x)
		}
	}
	sort.Strings(missingFromB)
	sort.Strings(extraInB)

	if len(missingFromB) > 0 || len(extraInB) > 0 {
		t.Fatalf("%s: field sets differ (missing: %v; extra: %v)", label, missingFromB, extraInB)
	}
}
