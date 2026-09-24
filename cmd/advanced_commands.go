package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// hubPreferencesFileName is the file, stored with the owner's own
// preferences on his machine rather than inside one project, that holds
// machine-wide choices such as whether the full command menu is shown by
// default. A-01: the default help screen renders in a folder with no
// project set up in it, so this setting must be readable there too -- a
// per-project file cannot be.
const hubPreferencesFileName = "preferences.json"

// hubPreferencesSchemaVersion stamps the shape of preferences.json on every
// write. An absent file, an unreadable file, a malformed file, or a file
// carrying any other version all read as "the owner has never set this" --
// never as an error.
const hubPreferencesSchemaVersion = "aether-preferences/v1"

// hubPreferences is the whole shape of preferences.json. ShowAllCommands is
// the only field this plan reads or writes; a future preference is added
// here the same way, never as a second file.
type hubPreferences struct {
	SchemaVersion   string `json:"schema_version,omitempty"`
	ShowAllCommands bool   `json:"show_all_commands,omitempty"`
}

// readAdvancedCommandsSetting reports whether the owner has switched the
// full command menu on for this machine, and whether that answer is the
// untouched default or something he actually stored.
//
// This is a plain os.ReadFile -- no storage.NewStore, no MkdirAll, no lock --
// so merely asking never creates a data or lock directory anywhere. An empty
// hub path, a missing file, an unreadable file, a malformed file, or a file
// carrying an unrecognised schema version all report (false, "default");
// this reader can never fail a caller.
func readAdvancedCommandsSetting() (bool, string) {
	hub := resolveHubPathQuiet()
	if strings.TrimSpace(hub) == "" {
		return false, "default"
	}
	data, err := os.ReadFile(filepath.Join(hub, hubPreferencesFileName))
	if err != nil {
		return false, "default"
	}
	var prefs hubPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return false, "default"
	}
	if prefs.SchemaVersion != hubPreferencesSchemaVersion {
		return false, "default"
	}
	return prefs.ShowAllCommands, "stored"
}

// writeAdvancedCommandsSetting stores the owner's choice with his other
// preferences on this machine. The hub directory is created only here, when
// a write is actually happening -- never on a read. Any other key already
// present in preferences.json (a field this command does not know about) is
// preserved rather than dropped.
func writeAdvancedCommandsSetting(on bool) error {
	hub := resolveHubPath()
	if strings.TrimSpace(hub) == "" {
		return fmt.Errorf("hub path unavailable")
	}
	path := filepath.Join(hub, hubPreferencesFileName)
	raw := map[string]interface{}{}
	if existing, err := os.ReadFile(path); err == nil {
		// Best-effort: a malformed existing file simply gets replaced by a
		// well-formed one rather than blocking the write.
		_ = json.Unmarshal(existing, &raw)
	}
	raw["schema_version"] = hubPreferencesSchemaVersion
	raw["show_all_commands"] = on
	if err := os.MkdirAll(hub, 0o755); err != nil {
		return fmt.Errorf("create hub directory: %w", err)
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write preferences: %w", err)
	}
	return nil
}

// --- advanced-commands get|set ---

var advancedCommandsCmd = &cobra.Command{
	Use:   "advanced-commands",
	Short: "Get or set whether the full command menu shows by default",
	Args:  cobra.NoArgs,
	// This setting is machine-wide, not project-scoped (A-01), and must be
	// readable and writable in a folder with no Aether project set up in
	// it. Skipping store init here is what keeps that true -- neither
	// subcommand ever opens or creates a project store.
	Annotations: map[string]string{"aether.io/store-free": "true"},
}

var advancedCommandsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show whether the full command menu shows by default on this machine",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		on, source := readAdvancedCommandsSetting()
		outputOK(map[string]interface{}{"value": on, "source": source})
		return nil
	},
}

var advancedCommandsSetCmd = &cobra.Command{
	Use:   "set <on|off>",
	Short: "Turn the full command menu on or off by default on this machine",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var on bool
		switch strings.ToLower(strings.TrimSpace(args[0])) {
		case "on":
			on = true
		case "off":
			on = false
		default:
			outputError(1, fmt.Sprintf("invalid value %q for advanced-commands set: must be \"on\" or \"off\"", args[0]), nil)
			return nil
		}
		if err := writeAdvancedCommandsSetting(on); err != nil {
			outputError(2, fmt.Sprintf("failed to save preference: %v", err), nil)
			return nil
		}
		mode := "off"
		if on {
			mode = "on"
		}
		outputOK(map[string]interface{}{"value": on, "source": "cli", "mode": mode})
		return nil
	},
}

func init() {
	advancedCommandsCmd.AddCommand(advancedCommandsGetCmd)
	advancedCommandsCmd.AddCommand(advancedCommandsSetCmd)
	rootCmd.AddCommand(advancedCommandsCmd)
}
