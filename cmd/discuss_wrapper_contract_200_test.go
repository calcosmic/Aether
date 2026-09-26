package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDiscussWrapperContract200(t *testing.T) {
	const (
		description = "💬 Resolve evidence-backed material decisions and hand settled intent to a draft specification"
		runtimeCall = "AETHER_OUTPUT_MODE=json aether discuss $ARGUMENTS"
		source      = ".aether/commands/discuss.yaml"
	)

	repoRoot := filepath.Clean("..")
	rawYAML := readDiscussContractFile200(t, filepath.Join(repoRoot, filepath.FromSlash(source)))
	var spec struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Runtime     struct {
			Command string `yaml:"command"`
		} `yaml:"runtime"`
		CeremonyContract map[string]any `yaml:"ceremony_contract"`
		Guardrails       []string       `yaml:"guardrails"`
	}
	if err := yaml.Unmarshal([]byte(rawYAML), &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Name != "ant-discuss" || spec.Description != description || spec.Runtime.Command != runtimeCall {
		t.Fatalf("canonical discuss wrapper contract drifted: %+v", spec)
	}

	canonical := strings.ToLower(rawYAML)
	assertDiscussSemantics200(t, source, canonical, []string{
		"material batch.cards", "decision", "why now", "evidence", "queen recommendation",
		"choices", "consequence", "prior answer", "revalidation", "planning resume",
		"exact answer syntax", "exact goal", "session", "specification revision",
		"base plan", "meaning", "behavior", "authority", "scope", "risk",
		"acceptance", "affected semantic", "draft spec", "/ant spec", "owner reviews and approves",
		"do not write", "structured runtime result", "never route settled discuss directly to /ant plan",
	})
	assertDiscussSemantics200(t, source, canonical, discussOwnerInterviewAnchors200)
	assertNoDiscussHostAuthority200(t, source, canonical)

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant-discuss.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "discuss.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "discuss.md"),
	}
	for _, path := range wrapperPaths {
		text := readDiscussContractFile200(t, path)
		wantHeader := "<!-- Aether-managed: runtime spec at " + source + ". Synced by aether update. -->"
		if !strings.HasPrefix(text, wantHeader+"\n") {
			t.Errorf("%s lacks canonical source linkage", path)
		}
		if !strings.Contains(text, `description: "`+description+`"`) {
			t.Errorf("%s description does not match canonical source", path)
		}
		if strings.Count(text, runtimeCall) != 1 {
			t.Errorf("%s contains runtime call %d times, want exactly one", path, strings.Count(text, runtimeCall))
		}
		lower := strings.ToLower(text)
		assertDiscussSemantics200(t, path, lower, []string{
			"material batch.cards", "decision", "why now", "evidence", "queen recommendation",
			"choices", "consequence", "prior answer", "revalidation", "planning resumes",
			"exact answer syntax", "exact goal", "session", "specification revision",
			"base plan", "meaning", "behavior", "authority", "scope", "risk",
			"acceptance", "affected semantic", "draft spec", "/ant spec", "owner reviews and approves",
			"structured result", "do not write", "never route settled discuss directly to /ant plan",
		})
		assertDiscussSemantics200(t, path, lower, discussOwnerInterviewAnchors200)
		assertNoDiscussHostAuthority200(t, path, lower)
	}

	contractPath := filepath.Join(repoRoot, "cmd", "contracts", "discuss.md")
	contract := strings.ToLower(readDiscussContractFile200(t, contractPath))
	assertDiscussSemantics200(t, contractPath, contract, []string{
		"evidence first", "material batch.cards", "decision id", "why now", "queen recommendation",
		"choices[].consequence", "affected semantic ids", "prior answer", "revalidation",
		"planning resumes", "exact answer syntax", "answer equivalence", "exact goal",
		"approved specification revision", "base plan revision", "decision meaning", "behavior",
		"authority", "scope", "risk", "acceptance meaning", "draft spec", "nine contract categories",
		"aether spec", "/ant spec", "separate", "wrappers never write",
	})
	assertDiscussSemantics200(t, contractPath, contract, discussOwnerInterviewAnchors200)

	discussGuide := commandGuideCatalog()["discuss"]
	guide := strings.ToLower(strings.Join(append(append([]string{}, discussGuide.PreSteps...), discussGuide.PostSteps...), "\n"))
	assertDiscussSemantics200(t, "command guide discuss (codex)", guide, discussOwnerInterviewAnchors200)
	if strings.Contains(guide, "$ant-plan") {
		t.Error("codex command guide still routes settled discuss to planning; the next boundary is the specification")
	}

	skillPath := filepath.Join(repoRoot, ".aether", "skills", "colony", "aether-colony-research", "SKILL.md")
	skill := strings.ToLower(readDiscussContractFile200(t, skillPath))
	assertDiscussSemantics200(t, skillPath, skill, []string{
		"aether command guide discuss --platform codex", "aether output mode=json aether discuss",
		"material batch.cards", "decision", "why now", "evidence", "queen recommendation",
		"consequence", "affected semantic ids", "prior answer", "revalidation",
		"planning resumes", "exact answer syntax", "exact goal", "session",
		"approved specification revision", "base plan revision", "decision meaning", "behavior",
		"authority", "scope", "risk", "acceptance meaning", "affected semantic ids",
		"draft spec", "aether spec", "separate owner action",
	})
	assertDiscussSemantics200(t, skillPath, skill, discussOwnerInterviewAnchors200)
	for _, oracleAnchor := range []string{
		"aether command guide oracle --platform codex", "compact batch of 3 6 questions",
		"tech eval", "architecture review", "bug investigation", "research brief",
		"--confidence target", "--background", "aether oracle status",
	} {
		if !strings.Contains(normalizeDiscussContractSemantics200(skill), normalizeDiscussContractSemantics200(oracleAnchor)) {
			t.Errorf("research skill regressed Oracle behavior %q", oracleAnchor)
		}
	}

	for flag, wantUsage := range map[string]string{
		"max-questions": "legacy compatibility flag; every material clarification is returned in one batch",
		"dry-run":       "analyze and preview questions without writing pending decisions",
		"resolve":       "clarification decision id to resolve",
		"answer":        "resolution text for --resolve",
		"add-question":  "materialize a composed clarification question",
		"options":       "pipe-separated answer options for --add-question",
		"category":      "category for --add-question",
		"grounding":     "required with --add-question",
		"hard":          "hard constraint",
		"source":        "stable dedup slug for --add-question",
	} {
		found := discussCmd.Flags().Lookup(flag)
		if found == nil {
			t.Errorf("runtime help lacks --%s", flag)
			continue
		}
		if !strings.Contains(strings.ToLower(found.Usage), wantUsage) {
			t.Errorf("runtime help for --%s = %q, want semantic anchor %q", flag, found.Usage, wantUsage)
		}
	}
}

func readDiscussContractFile200(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func assertDiscussSemantics200(t *testing.T, path, content string, required []string) {
	t.Helper()
	content = normalizeDiscussContractSemantics200(content)
	for _, semantic := range required {
		semantic = normalizeDiscussContractSemantics200(semantic)
		if !strings.Contains(content, semantic) {
			t.Errorf("%s lacks discuss semantic %q", path, semantic)
		}
	}
}

func normalizeDiscussContractSemantics200(value string) string {
	value = strings.NewReplacer("_", " ", "-", " ", "`", "").Replace(strings.ToLower(value))
	return strings.Join(strings.Fields(value), " ")
}

func assertNoDiscussHostAuthority200(t *testing.T, path, content string) {
	t.Helper()
	for _, forbidden := range []string{
		"compose 3-5", "formulate codebase-aware questions", "run the canned fallback",
		"route settled discuss to /ant-plan", "route back to aether plan", "savejson(", "writefile(",
		"jq ", "cat >", "> .aether/data", "discuss-analyze --target",
	} {
		if strings.Contains(content, forbidden) {
			t.Errorf("%s contains forbidden wrapper-side authority or settled-to-plan shortcut %q", path, forbidden)
		}
	}
}

// discussOwnerInterviewAnchors200 is the owner's ruling of 2026-09-26 (Phase
// 210, after blocker 4): /ant-discuss interviews him with about 20
// multiple-choice questions composed for this goal, asked a few at a time,
// every pick recorded through the runtime so it reaches the draft
// specification, and the resulting page shown before /ant-spec. It
// reverses the earlier "wrapper must not compose questions" contract; every
// discuss surface must carry the same rule.
var discussOwnerInterviewAnchors200 = []string{
	"owner interview",
	"about 20 multiple-choice questions",
	"up to 4 at a time",
	"aether discuss --add-question",
	"--options",
	"aether discuss --resolve",
	"nothing is recorded without the owner's explicit pick",
	"what you decided",
}
