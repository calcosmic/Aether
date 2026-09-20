package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const classicPublicCommandParitySchema = "classic-command-parity/v2"

type classicPublicCommandParity struct {
	SchemaVersion string                          `json:"schema_version"`
	Purpose       string                          `json:"purpose"`
	Commands      []classicPublicCommandParityRow `json:"commands"`
}

type classicPublicCommandParityRow struct {
	Category    string `json:"category"`
	PublicName  string `json:"public_name"`
	CobraName   string `json:"cobra_name"`
	Description string `json:"description"`
}

func TestClassicCommandParity(t *testing.T) {
	manifest := loadClassicPublicCommandParity(t)
	assertClassicPublicCommandParity(t, manifest)
}

func loadClassicPublicCommandParity(t *testing.T) classicPublicCommandParity {
	t.Helper()
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find command source root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".aether", "commands", "classic-command-parity.json"))
	if err != nil {
		t.Fatalf("read command parity manifest: %v", err)
	}
	var manifest classicPublicCommandParity
	if err := decodeClassicStrictJSON(data, &manifest); err != nil {
		t.Fatalf("decode command parity manifest: %v", err)
	}
	return manifest
}

func assertClassicPublicCommandParity(t *testing.T, manifest classicPublicCommandParity) {
	t.Helper()
	if manifest.SchemaVersion != classicPublicCommandParitySchema || strings.TrimSpace(manifest.Purpose) == "" {
		t.Fatalf("invalid public parity manifest metadata: %+v", manifest)
	}
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find command source root: %v", err)
	}
	want := classicPublicCommandInventory()
	if len(manifest.Commands) != len(want) {
		t.Fatalf("public command count = %d, want %d", len(manifest.Commands), len(want))
	}
	seen := map[string]bool{}
	for index, row := range manifest.Commands {
		if seen[row.PublicName] {
			t.Fatalf("duplicate public command %q", row.PublicName)
		}
		seen[row.PublicName] = true
		if row != want[index] {
			t.Fatalf("public command %d = %+v, want %+v", index, row, want[index])
		}
		classicAssertCanonicalAndManagedCommand(t, root, row)
		classicAssertCobraPublicCommand(t, row)
	}
	for _, forbidden := range []string{"setup", "finalizer", "recover", "abandon", "pause-colony", "resume-colony", "skill-list", "skill-cache-rebuild"} {
		if seen[forbidden] {
			t.Fatalf("protocol or retired command %q leaked into public manifest", forbidden)
		}
	}
}

func TestDiscussAndSpecRegisteredAcrossPublicSurfaces(t *testing.T) {
	manifest := loadClassicPublicCommandParity(t)
	rows := make(map[string]classicPublicCommandParityRow, len(manifest.Commands))
	for _, row := range manifest.Commands {
		rows[row.PublicName] = row
	}

	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find command source root: %v", err)
	}
	for _, publicName := range []string{"discuss", "spec"} {
		row, ok := rows[publicName]
		if !ok {
			t.Fatalf("normal public inventory is missing %q", publicName)
		}
		if row.Category != "normal" || row.CobraName != publicName {
			t.Fatalf("public %s row = %+v, want normal Cobra command %s", publicName, row, publicName)
		}
		classicAssertCobraPublicCommand(t, row)
		classicAssertCanonicalAndManagedCommand(t, root, row)
	}

	for _, publicName := range []string{"discuss", "spec", "plan"} {
		if got, want := platformCommandName(publicName, "claude"), "/ant-"+publicName; got != want {
			t.Fatalf("Claude spelling for %s = %q, want %q", publicName, got, want)
		}
		if got, want := platformCommandName(publicName, "opencode"), "/ant-"+publicName; got != want {
			t.Fatalf("OpenCode spelling for %s = %q, want %q", publicName, got, want)
		}
		want := "$ant-" + publicName
		if publicName == "spec" {
			want = "aether spec" // Specification skills remain deferred.
		}
		if got := platformCommandName(publicName, "codex"); got != want {
			t.Fatalf("Codex spelling for %s = %q, want %q", publicName, got, want)
		}
	}
}

func classicPublicCommandInventory() []classicPublicCommandParityRow {
	return []classicPublicCommandParityRow{
		{Category: "normal", PublicName: "init", CobraName: "init", Description: "Start a guided colony for one goal."},
		{Category: "normal", PublicName: "discuss", CobraName: "discuss", Description: "💬 Resolve evidence-backed material decisions and hand settled intent to a draft specification"},
		{Category: "normal", PublicName: "spec", CobraName: "spec", Description: "📜 Review, revise, approve, or repair the owner-readable specification"},
		{Category: "normal", PublicName: "plan", CobraName: "plan", Description: "📋 Run an evidence-backed Scout to Route-Setter planning loop and review the exact candidate"},
		{Category: "normal", PublicName: "build", CobraName: "build", Description: "🔨 Build a phase — Queen dispatches workers, colony self-organizes"},
		{Category: "normal", PublicName: "run", CobraName: "run", Description: "Autopilot the remaining accepted phases within the displayed safety contract."},
		{Category: "normal", PublicName: "status", CobraName: "status", Description: "Show the complete authoritative colony snapshot."},
		{Category: "normal", PublicName: "pause", CobraName: "pause", Description: "Stop at a safe boundary and save one resumable handoff."},
		{Category: "normal", PublicName: "resume", CobraName: "resume", Description: "Validate and restore the safest honest recovery point."},
		{Category: "normal", PublicName: "seal", CobraName: "seal", Description: "Close a verified colony, or explicitly record an owner-forced incomplete closure."},
		{Category: "normal", PublicName: "entomb", CobraName: "entomb", Description: "Archive and clear the sealed colony."},
		{Category: "steer-inspect", PublicName: "focus", CobraName: "focus", Description: "🔦 Emit a FOCUS pheromone through the Aether CLI runtime"},
		{Category: "steer-inspect", PublicName: "feedback", CobraName: "feedback", Description: "💬 Emit FEEDBACK through the Aether CLI runtime"},
		{Category: "steer-inspect", PublicName: "redirect", CobraName: "redirect", Description: "🚫 Emit a REDIRECT pheromone through the Aether CLI runtime"},
		{Category: "steer-inspect", PublicName: "watch", CobraName: "watch", Description: "👁️ View the current colony watch surface through the Aether CLI runtime"},
		{Category: "steer-inspect", PublicName: "phase", CobraName: "phase", Description: "🧱 View phase details through the Aether CLI runtime"},
		{Category: "steer-inspect", PublicName: "history", CobraName: "history", Description: "📜 Show colony event history"},
		{Category: "steer-inspect", PublicName: "swarm", CobraName: "swarm", Description: "🔥 Real-time colony swarm display + visible bug-destroyer workers"},
		{Category: "steer-inspect", PublicName: "oracle", CobraName: "oracle", Description: "🔮 Run the autonomous Oracle loop through the Aether CLI runtime"},
		{Category: "steer-inspect", PublicName: "memory-details", CobraName: "memory-details", Description: "📜 Show what the colony has learned — wisdom, lessons waiting to be promoted, lessons put aside, and recent failures"},
		{Category: "steer-inspect", PublicName: "flags", CobraName: "flag-list", Description: "🚩 List project flags (blockers, issues, notes)"},
		{Category: "maintenance", PublicName: "maintenance", CobraName: "maintenance", Description: "Inspect or repair Aether internals with preview and rollback."},
	}
}

func classicAssertCanonicalAndManagedCommand(t *testing.T, root string, row classicPublicCommandParityRow) {
	t.Helper()
	canonical := filepath.Join(root, ".aether", "commands", row.PublicName+".yaml")
	canonicalData, err := os.ReadFile(canonical)
	if err != nil {
		t.Fatalf("read canonical command %s: %v", row.PublicName, err)
	}
	classicAssertCommandMetadata(t, canonical, canonicalData, row, true)
	managed := []struct {
		platform string
		path     string
	}{
		{platform: "claude-flat", path: filepath.Join(root, ".claude", "commands", "ant-"+row.PublicName+".md")},
		{platform: "claude", path: filepath.Join(root, ".claude", "commands", "ant", row.PublicName+".md")},
		{platform: "opencode", path: filepath.Join(root, ".opencode", "commands", "ant", row.PublicName+".md")},
	}
	for _, surface := range managed {
		path := surface.path
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s wrapper for %s: %v", surface.platform, row.PublicName, err)
		}
		if !bytes.HasPrefix(data, []byte("<!-- Aether-managed: runtime spec at .aether/commands/"+row.PublicName+".yaml.")) {
			t.Fatalf("%s wrapper for %s is not canonical-YAML managed", surface.platform, row.PublicName)
		}
		classicAssertCommandMetadata(t, path, data, row, false)
	}
}

var classicCommandDescription = regexp.MustCompile(`(?m)^description: "([^"]+)"$`)

func classicAssertCommandMetadata(t *testing.T, path string, data []byte, row classicPublicCommandParityRow, requireCanonicalDescription bool) {
	t.Helper()
	if !bytes.Contains(data, []byte("name: ant-"+row.PublicName)) {
		t.Fatalf("%s does not expose ant-%s", path, row.PublicName)
	}
	if requireCanonicalDescription {
		match := classicCommandDescription.FindSubmatch(data)
		if len(match) != 2 || string(match[1]) != row.Description {
			t.Fatalf("%s description = %q, want %q", path, string(match[1]), row.Description)
		}
	}
}

func classicAssertCobraPublicCommand(t *testing.T, row classicPublicCommandParityRow) {
	t.Helper()
	for _, command := range rootCmd.Commands() {
		if command.Name() != row.CobraName {
			continue
		}
		if command.Hidden {
			t.Fatalf("Cobra public command %s is hidden", row.CobraName)
		}
		if row.PublicName != row.CobraName && !slices.Contains(command.Aliases, row.PublicName) {
			t.Fatalf("Cobra command %s does not preserve public alias %s", row.CobraName, row.PublicName)
		}
		return
	}
	t.Fatalf("Cobra public command %s is missing", row.CobraName)
}
