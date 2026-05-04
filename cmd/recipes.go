package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type workflowRecipe struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	UseWhen      string   `json:"use_when"`
	WhyItMatters string   `json:"why_it_matters"`
	Commands     []string `json:"commands"`
}

var recipeCatalog = []workflowRecipe{
	{
		ID:           "start-existing-repo",
		Title:        "Start on an existing repository",
		UseWhen:      "You want Aether to understand a repo before it plans work.",
		WhyItMatters: "This mirrors the onboarding pattern in high-trust open source projects: inspect first, then plan.",
		Commands: []string{
			`aether lay-eggs`,
			`aether init "Improve this repository"`,
			`aether discuss`,
			`aether colonize`,
			`aether plan`,
		},
	},
	{
		ID:           "build-one-phase",
		Title:        "Build and verify one phase",
		UseWhen:      "You want controlled progress with a review step after implementation.",
		WhyItMatters: "A short build/continue loop keeps changes understandable and makes failures easier to recover.",
		Commands: []string{
			`aether status`,
			`aether build 1`,
			`aether continue`,
		},
	},
	{
		ID:           "autopilot-small-batch",
		Title:        "Autopilot a small batch",
		UseWhen:      "You trust the current plan but still want a bounded run.",
		WhyItMatters: "The limit gives speed without turning a long plan into an unobserved background task.",
		Commands: []string{
			`aether run --max-phases 2`,
			`aether status`,
		},
	},
	{
		ID:           "recover-session",
		Title:        "Recover after a break",
		UseWhen:      "You are returning after clearing context, closing the terminal, or switching tools.",
		WhyItMatters: "A repeatable recovery path prevents stale context from steering the next build.",
		Commands: []string{
			`aether resume`,
			`aether status`,
			`aether watch --once`,
		},
	},
	{
		ID:           "steer-active-work",
		Title:        "Steer active work",
		UseWhen:      "You need to point Aether toward a concern or away from a risky approach.",
		WhyItMatters: "Signals make intent explicit before workers start, which is cheaper than correcting the wrong build later.",
		Commands: []string{
			`aether focus "security-sensitive path"`,
			`aether redirect "do not rewrite unrelated files"`,
			`aether feedback "prefer smaller phases"`,
			`aether pheromones`,
		},
	},
	{
		ID:           "ship-local-update",
		Title:        "Ship a local Aether update",
		UseWhen:      "You changed Aether itself and need the local runtime and companion files refreshed.",
		WhyItMatters: "Aether has source files and installed hub files; this path keeps them aligned before other repos consume the update.",
		Commands: []string{
			`go test ./...`,
			`aether publish --channel stable --binary-dest "$HOME/.local/bin"`,
			`aether update --force`,
		},
	},
}

var recipesCmd = &cobra.Command{
	Use:     "recipes [id]",
	Aliases: []string{"examples", "quickstart"},
	Short:   "Show copyable Aether workflow recipes",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		recipes := recipeCatalog
		selected := false
		if len(args) > 0 {
			recipe, ok := findWorkflowRecipe(args[0])
			if !ok {
				outputError(1, fmt.Sprintf("unknown recipe %q", args[0]), map[string]interface{}{
					"available": workflowRecipeIDs(),
				})
				return nil
			}
			recipes = []workflowRecipe{recipe}
			selected = true
		}

		result := map[string]interface{}{
			"recipes": recipes,
			"count":   len(recipes),
		}
		if !selected {
			result["next"] = "aether recipes start-existing-repo"
		}
		outputWorkflow(result, renderRecipesVisual(recipes, selected))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(recipesCmd)
}

func findWorkflowRecipe(id string) (workflowRecipe, bool) {
	id = strings.TrimSpace(strings.ToLower(id))
	for _, recipe := range recipeCatalog {
		if recipe.ID == id {
			return recipe, true
		}
	}
	return workflowRecipe{}, false
}

func workflowRecipeIDs() []string {
	ids := make([]string, 0, len(recipeCatalog))
	for _, recipe := range recipeCatalog {
		ids = append(ids, recipe.ID)
	}
	return ids
}

func renderRecipesVisual(recipes []workflowRecipe, selected bool) string {
	var b strings.Builder
	b.WriteString(renderBanner("📚", "Aether Recipes"))
	b.WriteString(visualDivider)
	b.WriteString("Copyable paths for common Aether jobs.\n\n")

	for i, recipe := range recipes {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s  %s\n", recipe.ID, recipe.Title)
		fmt.Fprintf(&b, "Use when: %s\n", recipe.UseWhen)
		fmt.Fprintf(&b, "Why: %s\n", recipe.WhyItMatters)
		b.WriteString("Commands:\n")
		for _, command := range recipe.Commands {
			b.WriteString("  ")
			b.WriteString(command)
			b.WriteString("\n")
		}
	}

	if !selected {
		b.WriteString(renderNextUp(
			`Run `+"`aether recipes start-existing-repo`"+` to show one recipe.`,
			`Run `+"`AETHER_OUTPUT_MODE=json aether recipes`"+` for machine-readable output.`,
		))
	}
	return b.String()
}
