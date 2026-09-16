package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
)

const codexSkillPayloadSchema = "codex-skill-payload/v1"

// Keep availability separate from the guide's much larger runtime catalog.
func codexPublicSkillCommands() []string {
	return []string{"init", "discuss", "oracle", "colonize", "plan", "build", "continue", "swarm", "seal"}
}

type codexSkillPayloadFile struct {
	RelativePath string `json:"relative_path"`
	Mode         uint32 `json:"mode"`
	SHA256       string `json:"sha256"`
	Content      []byte `json:"-"`
}

type codexSkillPayload struct {
	SchemaVersion     string                  `json:"schema_version"`
	SourceVersion     string                  `json:"source_version"`
	MinRuntimeVersion string                  `json:"min_runtime_version"`
	GeneratorIdentity string                  `json:"generator_identity"`
	Commands          []string                `json:"commands"`
	Files             []codexSkillPayloadFile `json:"files"`
}

var codexSkillIDPattern = regexp.MustCompile(`^[a-z]+(?:-[a-z]+)*$`)
var codexSkillVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)

func validateCodexSkillInventory(commands []string, catalog map[string]commandGuideDefinition) error {
	wanted := map[string]bool{"init": true, "discuss": true, "oracle": true, "colonize": true, "plan": true, "build": true, "continue": true, "swarm": true, "seal": true}
	if len(commands) != len(wanted) {
		return fmt.Errorf("codex skills: source inventory must contain exactly nine commands")
	}
	seen := map[string]bool{}
	for _, command := range commands {
		if !codexSkillIDPattern.MatchString(command) || !wanted[command] || seen[command] {
			return fmt.Errorf("codex skills: invalid or duplicate source command %q", command)
		}
		seen[command] = true
		def, ok := catalog[command]
		if !ok || def.Literal || def.RunCommand == "" || !codexSkillSupportName(def.SkillReference) {
			return fmt.Errorf("codex skills: missing or invalid guide definition for %q", command)
		}
	}
	return nil
}

func codexSkillSupportName(name string) bool {
	return name == "aether-colony-creation" || name == "aether-colony-research" || name == "aether-colony-build-cycle"
}

func buildCodexSkillPayload(packageDir string) (codexSkillPayload, error) {
	return buildCodexSkillPayloadFromInventory(packageDir, codexPublicSkillCommands(), commandGuideCatalog())
}

func buildCodexSkillPayloadFromInventory(packageDir string, commands []string, catalog map[string]commandGuideDefinition) (codexSkillPayload, error) {
	var payload codexSkillPayload
	if err := validateCodexSkillInventory(commands, catalog); err != nil {
		return payload, err
	}
	version := readRepoVersion(packageDir)
	if version == "" {
		version = resolveVersion(packageDir)
	}
	payload = codexSkillPayload{
		SchemaVersion: codexSkillPayloadSchema, SourceVersion: version,
		MinRuntimeVersion: "1.0.79", GeneratorIdentity: "aether/" + executingRuntimeVersion(),
		Commands: append([]string(nil), commands...),
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" {
				payload.GeneratorIdentity += " " + setting.Key + "=" + setting.Value
			}
		}
	}
	add := func(relative string, content []byte) {
		payload.Files = append(payload.Files, codexSkillPayloadFile{RelativePath: relative, Mode: 0o644, SHA256: lifecycleDigest(content), Content: content})
	}
	for _, command := range commands {
		add("ant-"+command+"/SKILL.md", []byte(renderCodexSkillShim(codexPublicSkillShim(command, catalog[command]))))
	}
	for _, support := range []string{"aether-colony-creation", "aether-colony-research", "aether-colony-build-cycle"} {
		file := filepath.Join(packageDir, ".aether", "skills", "colony", support, "SKILL.md")
		content, err := os.ReadFile(file)
		if err != nil {
			return codexSkillPayload{}, fmt.Errorf("codex skills: required support %s: %w", support, err)
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			return codexSkillPayload{}, fmt.Errorf("codex skills: empty support %s", support)
		}
		add("support/"+support+".md", content)
	}
	return payload, validateCodexSkillPayload(payload)
}

func codexPublicSkillShim(command string, def commandGuideDefinition) codexSkillShim {
	return codexSkillShim{
		Dir: "ant-" + command, Name: "ant-" + command,
		Description:      fmt.Sprintf("Use $ant-%s to run the Aether %s lifecycle action with runtime-owned guidance.", command, command),
		Body:             renderCodexCommandSkillShimBody(command, def),
		WorkflowTriggers: []string{"ant-" + command, command},
		TaskKeywords:     []string{"ant-" + command, "aether " + command, "command-guide " + command},
	}
}

// Validate the desired envelope before inspecting any destinations. A compatible
// later producer may add commands; consumers must not prune them to an old list.
func validateCodexSkillPayload(payload codexSkillPayload) error {
	if payload.SchemaVersion != codexSkillPayloadSchema {
		return fmt.Errorf("codex skills: unsupported payload schema %q", payload.SchemaVersion)
	}
	if !codexSkillVersionPattern.MatchString(payload.SourceVersion) || !codexSkillVersionPattern.MatchString(payload.MinRuntimeVersion) || payload.GeneratorIdentity == "" {
		return fmt.Errorf("codex skills: missing or invalid payload identity/version")
	}
	if runtimeVersion := executingRuntimeVersion(); compareVersions(payload.MinRuntimeVersion, runtimeVersion) > 0 {
		return fmt.Errorf("codex skills: runtime %s is older than required %s", runtimeVersion, payload.MinRuntimeVersion)
	}
	expected := map[string]string{}
	seenCommands := map[string]bool{}
	for _, command := range payload.Commands {
		if !codexSkillIDPattern.MatchString(command) || seenCommands[command] {
			return fmt.Errorf("codex skills: invalid or duplicate command %q", command)
		}
		seenCommands[command] = true
		expected["ant-"+command+"/SKILL.md"] = "ant-" + command
	}
	for _, required := range codexPublicSkillCommands() {
		if !seenCommands[required] {
			return fmt.Errorf("codex skills: incomplete payload, missing %s", required)
		}
	}
	for _, support := range []string{"aether-colony-creation", "aether-colony-research", "aether-colony-build-cycle"} {
		expected["support/"+support+".md"] = ""
	}
	if len(payload.Files) != len(expected) {
		return fmt.Errorf("codex skills: incomplete or extra payload files")
	}
	seen := map[string]bool{}
	for _, file := range payload.Files {
		name, ok := expected[file.RelativePath]
		if !ok || seen[file.RelativePath] || path.Clean(file.RelativePath) != file.RelativePath || strings.Contains(file.RelativePath, "\\") {
			return fmt.Errorf("codex skills: invalid or duplicate path %q", file.RelativePath)
		}
		seen[file.RelativePath] = true
		if file.Mode != 0o644 || len(strings.TrimSpace(string(file.Content))) == 0 || lifecycleDigest(file.Content) != file.SHA256 {
			return fmt.Errorf("codex skills: content/mode/digest mismatch for %s", file.RelativePath)
		}
		if name != "" {
			fm := parseSkillFrontmatter(string(file.Content))
			if fm == nil || fm.Name != name {
				return fmt.Errorf("codex skills: invalid YAML name in %s", file.RelativePath)
			}
		}
	}
	return nil
}

func codexSkillPayloadIdentity(payload codexSkillPayload) string {
	raw, _ := json.Marshal(payload) // All fields have deterministic JSON encodings.
	return lifecycleDigest(raw)
}
