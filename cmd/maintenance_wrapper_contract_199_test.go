package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	maintenanceWrapperSource199      = ".aether/commands/maintenance.yaml"
	maintenanceWrapperDescription199 = "Inspect or repair Aether internals with preview and rollback."
	maintenanceWrapperInvocation199  = "AETHER_OUTPUT_MODE=visual aether maintenance $ARGUMENTS"
)

type maintenanceWrapperSpec199 struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	SourceOfTruth string `yaml:"source_of_truth"`
	Runtime       struct {
		Command string `yaml:"command"`
	} `yaml:"runtime"`
	Inventory struct {
		Inspection []string `yaml:"inspection"`
		Mutation   []string `yaml:"mutation"`
	} `yaml:"inventory"`
	Wrapper    string   `yaml:"wrapper"`
	Guardrails []string `yaml:"guardrails"`
}

func TestMaintenanceWrapperContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	canonical, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(maintenanceWrapperSource199)))
	if err != nil {
		t.Fatalf("read canonical maintenance wrapper source: %v", err)
	}
	var spec maintenanceWrapperSpec199
	if err := yaml.Unmarshal(canonical, &spec); err != nil {
		t.Fatalf("parse canonical maintenance wrapper source: %v", err)
	}

	wantInspection := maintenanceOperationIDs199(buildMaintenanceCatalog("claude").Inspection)
	wantMutation := maintenanceOperationIDs199(buildMaintenanceCatalog("claude").Mutation)
	if issues := maintenanceCanonicalWrapperIssues199(spec, wantInspection, wantMutation); len(issues) > 0 {
		t.Fatalf("canonical maintenance wrapper contract drifted:\n%s", strings.Join(issues, "\n"))
	}

	generated := []string{
		".claude/commands/ant-maintenance.md",
		".claude/commands/ant/maintenance.md",
		".opencode/commands/ant/maintenance.md",
	}
	wantHeader := "<!-- Aether-managed: runtime spec at " + maintenanceWrapperSource199 + ". Synced by aether update. -->"
	wantManagedBody := strings.TrimSpace(spec.Wrapper)
	var sharedBody string
	for _, rel := range generated {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("read generated maintenance wrapper: %v", err)
			}
			text := strings.ReplaceAll(string(data), "\r\n", "\n")
			firstLine, managedBody, ok := strings.Cut(text, "\n")
			if !ok {
				t.Fatal("generated maintenance wrapper has no body")
			}
			if firstLine != wantHeader {
				t.Fatalf("managed source header = %q, want %q", firstLine, wantHeader)
			}
			managedBody = strings.TrimSpace(managedBody)
			if managedBody != wantManagedBody {
				t.Error("generated body is not mechanically sourced from maintenance.yaml")
			}
			if sharedBody == "" {
				sharedBody = managedBody
			} else if managedBody != sharedBody {
				t.Error("Claude and OpenCode maintenance wrappers are not semantically identical")
			}

			frontmatter, _, err := parseSourceCheckWrapper(data)
			if err != nil {
				t.Fatalf("parse generated maintenance wrapper: %v", err)
			}
			if frontmatter.Name != "ant-maintenance" || frontmatter.Description != maintenanceWrapperDescription199 {
				t.Fatalf("frontmatter = %#v, want ant-maintenance and exact approved description", frontmatter)
			}
			if issues := maintenanceWrapperSemanticIssues199(managedBody, wantInspection, wantMutation); len(issues) > 0 {
				t.Errorf("wrapper violates the maintenance authority contract:\n%s", strings.Join(issues, "\n"))
			}
		})
	}

	if hits, err := codexMaintenanceSurfaces199(repoRoot); err != nil {
		t.Fatalf("inspect Codex surfaces: %v", err)
	} else if len(hits) > 0 {
		t.Fatalf("maintenance gained a deferred Codex-native $ant-* surface:\n%s", strings.Join(hits, "\n"))
	}

	fixtures := []struct {
		name string
		body string
		want string
	}{
		{
			name: "missing maintenance subcommand",
			body: strings.Replace(wantManagedBody, "`skills.diff`", "`skills.diff.removed`", 1),
			want: "missing operation",
		},
		{
			name: "direct filesystem mutation",
			body: wantManagedBody + "\n- Run `echo '{}' > .aether/data/COLONY_STATE.json` before returning.\n",
			want: "filesystem or state mutation",
		},
		{
			name: "non-runtime state authority",
			body: wantManagedBody + "\n- Read `COLONY_STATE.json` and decide whether the repair is complete.\n",
			want: "filesystem or state mutation",
		},
		{
			name: "visual output parsing",
			body: wantManagedBody + "\n- Parse visual stdout with `jq` to determine the result.\n",
			want: "visual output parsing",
		},
		{
			name: "invented repair evidence",
			body: wantManagedBody + "\n- Report repair succeeded and evidence verified when the process exits zero.\n",
			want: "invented repair or evidence",
		},
		{
			name: "root level live skill command",
			body: wantManagedBody + "\n- Run `/ant-skill-list` to inspect live skills.\n",
			want: "root-level live-skill command",
		},
		{
			name: "second runtime invocation",
			body: wantManagedBody + "\n- Retry `" + maintenanceWrapperInvocation199 + "` if output is unclear.\n",
			want: "exactly one runtime delegation",
		},
	}
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run("rejects_"+strings.ReplaceAll(fixture.name, " ", "_"), func(t *testing.T) {
			issues := maintenanceWrapperSemanticIssues199(fixture.body, wantInspection, wantMutation)
			if !containsMaintenanceContractIssue199(issues, fixture.want) {
				t.Fatalf("fixture was not rejected for %q; issues=%#v", fixture.want, issues)
			}
		})
	}

	t.Run("rejects_codex_native_ant_surface", func(t *testing.T) {
		issues := codexMaintenanceSurfaceIssues199(map[string]string{
			".codex/skills/ant-maintenance/SKILL.md": "Use `$ant-maintenance` from Codex.",
		})
		if !containsMaintenanceContractIssue199(issues, "Codex-native") {
			t.Fatalf("Codex fixture was not rejected: %#v", issues)
		}
	})
}

func maintenanceCanonicalWrapperIssues199(spec maintenanceWrapperSpec199, wantInspection, wantMutation []string) []string {
	var issues []string
	if spec.Name != "ant-maintenance" {
		issues = append(issues, fmt.Sprintf("name = %q, want ant-maintenance", spec.Name))
	}
	if spec.Description != maintenanceWrapperDescription199 {
		issues = append(issues, fmt.Sprintf("description = %q, want %q", spec.Description, maintenanceWrapperDescription199))
	}
	if spec.Runtime.Command != maintenanceWrapperInvocation199 {
		issues = append(issues, fmt.Sprintf("runtime command = %q, want %q", spec.Runtime.Command, maintenanceWrapperInvocation199))
	}
	if !strings.Contains(strings.ToLower(spec.SourceOfTruth), "go") || !strings.Contains(spec.SourceOfTruth, "`aether`") {
		issues = append(issues, "source_of_truth does not name the Go aether runtime")
	}
	if !slices.Equal(spec.Inventory.Inspection, wantInspection) {
		issues = append(issues, fmt.Sprintf("inspection inventory = %#v, want %#v", spec.Inventory.Inspection, wantInspection))
	}
	if !slices.Equal(spec.Inventory.Mutation, wantMutation) {
		issues = append(issues, fmt.Sprintf("mutation inventory = %#v, want %#v", spec.Inventory.Mutation, wantMutation))
	}
	if strings.TrimSpace(spec.Wrapper) == "" {
		issues = append(issues, "canonical wrapper body is empty")
	} else {
		issues = append(issues, maintenanceWrapperSemanticIssues199(spec.Wrapper, wantInspection, wantMutation)...)
	}
	guardrails := strings.ToLower(strings.Join(spec.Guardrails, "\n"))
	for _, want := range []string{"runtime", "state", "evidence", "visual", "codex"} {
		if !strings.Contains(guardrails, want) {
			issues = append(issues, fmt.Sprintf("canonical guardrails do not cover %s authority", want))
		}
	}
	return issues
}

func maintenanceWrapperSemanticIssues199(body string, wantInspection, wantMutation []string) []string {
	normalized := strings.ReplaceAll(body, "\r\n", "\n")
	lower := strings.ToLower(normalized)
	var issues []string

	if count := strings.Count(normalized, maintenanceWrapperInvocation199); count != 1 {
		issues = append(issues, fmt.Sprintf("expected exactly one runtime delegation, found %d", count))
	}
	for _, want := range []string{
		"return its stdout unchanged",
		"runtime owns the operation inventory",
		"runtime owns every state and evidence read",
		"runtime owns every mutation",
		"transaction receipt",
		"automatic rollback",
		"do not parse visual output",
		"do not write colony state",
	} {
		if !strings.Contains(lower, want) {
			issues = append(issues, fmt.Sprintf("missing runtime-authority contract %q", want))
		}
	}

	inspectionHeading := strings.Index(normalized, "## Inspection")
	mutationHeading := strings.Index(normalized, "## Mutation")
	if inspectionHeading < 0 || mutationHeading <= inspectionHeading {
		issues = append(issues, "inspection and mutation inventory headings are absent or out of order")
	}
	last := inspectionHeading
	for _, operationID := range wantInspection {
		index := strings.Index(normalized, "`"+operationID+"`")
		if index < 0 {
			issues = append(issues, fmt.Sprintf("missing operation %s", operationID))
			continue
		}
		if index <= last || (mutationHeading >= 0 && index >= mutationHeading) {
			issues = append(issues, fmt.Sprintf("inspection operation %s is outside the ordered inspection inventory", operationID))
		}
		last = index
	}
	last = mutationHeading
	for _, operationID := range wantMutation {
		index := strings.Index(normalized, "`"+operationID+"`")
		if index < 0 {
			issues = append(issues, fmt.Sprintf("missing operation %s", operationID))
			continue
		}
		if index <= last {
			issues = append(issues, fmt.Sprintf("mutation operation %s is outside the ordered mutation inventory", operationID))
		}
		last = index
	}

	for _, forbidden := range []struct {
		label   string
		pattern *regexp.Regexp
	}{
		{
			label:   "filesystem or state mutation",
			pattern: regexp.MustCompile(`(?i)(?:\.aether/data|COLONY_STATE\.json|session\.json|os\.WriteFile|writeFile\s*\(|atomicWrite\s*\(|(?:cat|tee|jq|sed|awk|perl|python3?|rm|mv|cp|echo)\b[^\n]*(?:>|\.aether/|COLONY_STATE))`),
		},
		{
			label:   "visual output parsing",
			pattern: regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:parse|strip|decode)\b[^\n]*(?:visual|ansi|stdout)|\b(?:jq|sed|awk|perl|python3?)\b[^\n]*(?:visual|ansi|stdout)`),
		},
		{
			label:   "invented repair or evidence",
			pattern: regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:if\b[^\n]{0,80},?\s*)?(?:report|declare|assume|invent|fabricate|treat)\b[^\n]{0,100}(?:repair\s+(?:succeeded|success)|evidence\s+(?:verified|complete)|success)`),
		},
		{
			label:   "root-level live-skill command",
			pattern: regexp.MustCompile(`(?i)(?:/ant-(?:skill-list|skill-diff|skill-cache-rebuild)|\baether\s+(?:skill-list|skill-diff|skill-cache-rebuild)\b)`),
		},
		{
			label:   "Codex-native $ant-* surface",
			pattern: regexp.MustCompile(`(?i)\$ant-(?:maintenance|skill)`),
		},
	} {
		if match := forbidden.pattern.FindString(normalized); match != "" {
			issues = append(issues, fmt.Sprintf("contains forbidden %s: %q", forbidden.label, match))
		}
	}

	for _, phrase := range []string{"wrapper decides", "wrapper verifies", "wrapper repairs", "wrapper mutates", "wrapper writes"} {
		if strings.Contains(lower, phrase) {
			issues = append(issues, fmt.Sprintf("wrapper claims non-runtime state authority through %q", phrase))
		}
	}
	for _, command := range []string{"`/ant-maintenance skills inspect`", "`/ant-maintenance skills diff"} {
		if !strings.Contains(lower, strings.ToLower(command)) {
			issues = append(issues, fmt.Sprintf("live skills are not exposed through maintenance landing %q", command))
		}
	}
	return issues
}

func maintenanceOperationIDs199(operations []maintenanceOperation) []string {
	ids := make([]string, 0, len(operations))
	for _, operation := range operations {
		ids = append(ids, operation.OperationID)
	}
	return ids
}

func codexMaintenanceSurfaces199(repoRoot string) ([]string, error) {
	root := filepath.Join(repoRoot, ".codex")
	contents := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		contents[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return codexMaintenanceSurfaceIssues199(contents), nil
}

func codexMaintenanceSurfaceIssues199(contents map[string]string) []string {
	var issues []string
	for path, content := range contents {
		lowerPath := strings.ToLower(filepath.ToSlash(path))
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, "$ant-maintenance") ||
			(strings.Contains(lowerPath, "/skills/") && strings.Contains(filepath.Base(lowerPath), "maintenance")) {
			issues = append(issues, "Codex-native maintenance surface: "+filepath.ToSlash(path))
		}
	}
	slices.Sort(issues)
	return issues
}

func containsMaintenanceContractIssue199(issues []string, want string) bool {
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue), strings.ToLower(want)) {
			return true
		}
	}
	return false
}
