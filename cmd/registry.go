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

// registryFinalStats holds colony lifecycle statistics recorded at entomb time.
type registryFinalStats struct {
	PhaseCount    int    `json:"phase_count,omitempty"`
	PlanCount     int    `json:"plan_count,omitempty"`
	LearningCount int    `json:"learning_count,omitempty"`
	InstinctCount int    `json:"instinct_count,omitempty"`
	SealDate      string `json:"seal_date,omitempty"`
	Duration      string `json:"duration,omitempty"`
}

type registryEntry struct {
	RepoPath     string              `json:"repo_path"`
	Domains      []string            `json:"domains"`
	Active       bool                `json:"active"`
	RegisteredAt string              `json:"registered_at"`
	LastGoal     string              `json:"last_goal,omitempty"`
	FinalStats   *registryFinalStats `json:"final_stats,omitempty"`
}

type registryData struct {
	Colonies []registryEntry `json:"colonies"`
}

// --- registry-add ---

var registryAddCmd = &cobra.Command{
	Use:   "registry-add [version]",
	Short: "Register a colony repository",
	Args:  cobra.MaximumNArgs(1), // accept optional positional version arg (ignored)
	RunE: func(cmd *cobra.Command, args []string) error {
		// --path takes priority over --repo
		repo, _ := cmd.Flags().GetString("repo")
		if path, _ := cmd.Flags().GetString("path"); path != "" {
			repo = path
		}
		if repo == "" {
			return nil
		}

		// --tags takes priority over --domain
		domainsStr, _ := cmd.Flags().GetString("domain")
		if tags, _ := cmd.Flags().GetString("tags"); tags != "" {
			domainsStr = tags
		}

		goal, _ := cmd.Flags().GetString("goal")
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
				// Update goal if provided
				if goal != "" {
					rd.Colonies[i].LastGoal = goal
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
			LastGoal:     goal,
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

// updateRegistryFinalStats updates a colony's registry entry with final lifecycle
// stats and marks it inactive. Best-effort: returns silently if no registry or no
// matching entry (colony may not have been registered).
func updateRegistryFinalStats(repoPath string, stats registryFinalStats) {
	hub := resolveHubPath()
	registryPath := filepath.Join(hub, "registry", "registry.json")

	var rd registryData
	raw, err := os.ReadFile(registryPath)
	if err != nil {
		return // no registry file -- best effort
	}
	if err := json.Unmarshal(raw, &rd); err != nil {
		return
	}
	for i, c := range rd.Colonies {
		if c.RepoPath == repoPath {
			rd.Colonies[i].Active = false
			rd.Colonies[i].FinalStats = &stats
			_ = writeRegistry(registryPath, rd)
			return
		}
	}
	// No matching entry -- best effort, don't error
}

func init() {
	registryAddCmd.Flags().String("repo", "", "Repository path")
	registryAddCmd.Flags().String("path", "", "Repository path (alias for --repo)")
	registryAddCmd.Flags().String("domain", "", "Comma-separated domain tags")
	registryAddCmd.Flags().String("tags", "", "Comma-separated domain tags (alias for --domain)")
	registryAddCmd.Flags().String("goal", "", "Colony goal")
	registryAddCmd.Flags().Bool("active", true, "Set colony as active (default: true)")

	rootCmd.AddCommand(registryAddCmd)
	rootCmd.AddCommand(registryListCmd)
}

// resolveHubPathQuiet is resolveHubPath without the outputError side effect,
// for non-blocking callers (init/seal bookkeeping must never fail a lifecycle
// command).
func resolveHubPathQuiet() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return resolveHubPathForHome(home, resolveRuntimeChannel())
}

// upsertColonyRegistryEntry registers or updates this repo in the hub-level
// colony registry. Wired into init (active, with detected domains) and seal
// (inactive) — RECLAIM-02: the registry supplies the domain tags that scope
// hive wisdom retrieval, and until this wiring nothing populated it.
func upsertColonyRegistryEntry(repoPath, goal string, domains []string, active bool) (updated bool, err error) {
	repoPath = strings.TrimSpace(repoPath)
	if repoPath == "" {
		return false, fmt.Errorf("empty repo path")
	}
	hub := resolveHubPathQuiet()
	if hub == "" {
		return false, fmt.Errorf("hub path unavailable")
	}
	registryPath := filepath.Join(hub, "registry", "registry.json")

	var rd registryData
	if raw, readErr := os.ReadFile(registryPath); readErr == nil {
		_ = json.Unmarshal(raw, &rd)
	}

	for i, c := range rd.Colonies {
		if c.RepoPath == repoPath {
			if len(domains) > 0 {
				rd.Colonies[i].Domains = domains
			}
			if goal != "" {
				rd.Colonies[i].LastGoal = goal
			}
			rd.Colonies[i].Active = active
			return true, writeRegistry(registryPath, rd)
		}
	}

	if domains == nil {
		domains = []string{}
	}
	rd.Colonies = append(rd.Colonies, registryEntry{
		RepoPath:     repoPath,
		Domains:      domains,
		Active:       active,
		RegisteredAt: time.Now().UTC().Format(time.RFC3339),
		LastGoal:     goal,
	})
	return false, writeRegistry(registryPath, rd)
}
