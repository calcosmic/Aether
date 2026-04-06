package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Registry types.

type registryEntry struct {
	RepoPath     string   `json:"repo_path"`
	Domains      []string `json:"domains"`
	Active       bool     `json:"active"`
	RegisteredAt string   `json:"registered_at"`
	LastGoal     string   `json:"last_goal,omitempty"`
}

type registryData struct {
	Colonies []registryEntry `json:"colonies"`
}

// --- registry-add ---

var registryAddCmd = &cobra.Command{
	Use:   "registry-add",
	Short: "Register a colony repository",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := mustGetString(cmd, "repo")
		if repo == "" {
			return nil
		}
		domainsStr, _ := cmd.Flags().GetString("domain")
		active, _ := cmd.Flags().GetBool("active")

		hub := resolveHubPath()
		registryPath := filepath.Join(hub, "registry", "registry.json")

		var rd registryData
		if raw, err := os.ReadFile(registryPath); err == nil {
			json.Unmarshal(raw, &rd)
		}

		// Check if already registered
		for i, c := range rd.Colonies {
			if c.RepoPath == repo {
				// Update domains
				if domainsStr != "" {
					rd.Colonies[i].Domains = strings.Split(domainsStr, ",")
					for j := range rd.Colonies[i].Domains {
						rd.Colonies[i].Domains[j] = strings.TrimSpace(rd.Colonies[i].Domains[j])
					}
				}
				rd.Colonies[i].Active = active
				if err := writeRegistry(registryPath, rd); err != nil {
					outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
					return nil
				}
				outputOK(map[string]interface{}{"registered": true, "updated": true, "repo": repo})
				return nil
			}
		}

		domains := []string{}
		if domainsStr != "" {
			domains = strings.Split(domainsStr, ",")
			for i := range domains {
				domains[i] = strings.TrimSpace(domains[i])
			}
		}

		entry := registryEntry{
			RepoPath:     repo,
			Domains:      domains,
			Active:       active,
			RegisteredAt: time.Now().UTC().Format(time.RFC3339),
		}

		rd.Colonies = append(rd.Colonies, entry)
		if err := writeRegistry(registryPath, rd); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"registered": true, "updated": false, "repo": repo, "total": len(rd.Colonies)})
		return nil
	},
}

// --- registry-list ---

var registryListCmd = &cobra.Command{
	Use:   "registry-list",
	Short: "List all registered colonies",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		registryPath := filepath.Join(hub, "registry", "registry.json")

		var rd registryData
		if raw, err := os.ReadFile(registryPath); err != nil {
			outputOK(map[string]interface{}{"colonies": []registryEntry{}, "total": 0})
			return nil
		} else {
			json.Unmarshal(raw, &rd)
		}

		outputOK(map[string]interface{}{"colonies": rd.Colonies, "total": len(rd.Colonies)})
		return nil
	},
}

func writeRegistry(path string, rd registryData) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	encoded, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(path, append(encoded, '\n'), 0644)
}

func init() {
	registryAddCmd.Flags().String("repo", "", "Repository path (required)")
	registryAddCmd.Flags().String("domain", "", "Comma-separated domain tags")
	registryAddCmd.Flags().Bool("active", true, "Set colony as active (default: true)")

	rootCmd.AddCommand(registryAddCmd)
	rootCmd.AddCommand(registryListCmd)
}
