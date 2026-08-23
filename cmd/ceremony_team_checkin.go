package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// renderCeremonyTeamCheckinFromFile renders the pre-spawn team check-in card
// from a lifecycle manifest JSON file. Read-only: the wrapper pauses on this
// card and asks the owner to proceed, trim optional workers, or redirect —
// the runtime plan itself is not touched.
func renderCeremonyTeamCheckinFromFile(workflow, path string) (map[string]interface{}, string, error) {
	raw, err := readCeremonyJSONFile(path)
	if err != nil {
		return nil, "", err
	}
	manifest, _ := extractCeremonyManifest(raw)
	if len(manifest) == 0 {
		return nil, "", fmt.Errorf("manifest file %s does not contain a lifecycle manifest", path)
	}
	dispatches := ceremonyDispatchesFromManifest(manifest)
	result, visual := renderCeremonyTeamCheckin(normalizedCeremonyWorkflow(workflow), manifest, dispatches)
	result["manifest_file"] = path
	return result, visual, nil
}

func renderCeremonyTeamCheckin(workflow string, manifest map[string]interface{}, dispatches []ceremonyDispatch) (map[string]interface{}, string) {
	policy := mapValue(manifest["queen_execution_policy"])
	budget := mapValue(policy["spawn_budget"])
	requiredSet := stringSet(stringSliceValue(budget["required_castes"]))
	selectedReasons := ceremonyStringMapValue(budget["selected_reasons"])
	prunedReasons := ceremonyStringMapValue(budget["pruned_reasons"])

	// Unique castes in dispatch order; each shows once whatever its worker count.
	seen := map[string]int{}
	orderedCastes := []string{}
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" {
			continue
		}
		if _, ok := seen[caste]; !ok {
			orderedCastes = append(orderedCastes, caste)
		}
		seen[caste]++
	}

	required := []string{}
	optional := []string{}
	reasons := map[string]string{}
	whatItDoes := map[string]string{}
	for _, caste := range orderedCastes {
		reason := strings.TrimSpace(selectedReasons[caste])
		if reason == "" {
			// D-09: the roster's generic "what this caste does" description is
			// NOT a reason a worker was sent for THIS phase — showing it here
			// would read as a justification and is not one. By the time plan
			// 194-03 has landed, every dispatched caste should already carry a
			// per-phase reason (its refusal loop and its runtime-written
			// reasons both guarantee this) — a caste reaching the card with
			// none is a bug upstream, so say so plainly rather than papering
			// over it.
			reason = "no reason was recorded for sending this worker"
		}
		reasons[caste] = reason
		if produces := casteRosterProduces(manifest, caste); produces != "" {
			whatItDoes[caste] = produces
		}
		if requiredSet[caste] {
			required = append(required, caste)
		} else {
			optional = append(optional, caste)
		}
	}

	var b strings.Builder
	b.WriteString(renderOldStyleCeremonyHeader(commandEmoji(emptyFallback(workflow, "team-checkin")), "Team Check-In"))
	b.WriteString("\n")
	for _, caste := range orderedCastes {
		b.WriteString("  ")
		b.WriteString(casteIdentityWithModel(caste))
		if requiredSet[caste] {
			b.WriteString("  REQUIRED")
		} else {
			b.WriteString("  OPTIONAL")
		}
		if count := seen[caste]; count > 1 {
			b.WriteString(fmt.Sprintf("  ×%d", count))
		}
		if reason := reasons[caste]; reason != "" {
			b.WriteString("  — ")
			b.WriteString(reason)
		}
		if produces := whatItDoes[caste]; produces != "" {
			b.WriteString("  (what it does: ")
			b.WriteString(produces)
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
	if len(prunedReasons) > 0 {
		prunedCastes := make([]string, 0, len(prunedReasons))
		for caste := range prunedReasons {
			prunedCastes = append(prunedCastes, caste)
		}
		sort.Strings(prunedCastes)
		b.WriteString("\nNot sent:\n")
		for _, caste := range prunedCastes {
			b.WriteString("  ")
			b.WriteString(casteLabel(caste))
			if reason := strings.TrimSpace(prunedReasons[caste]); reason != "" {
				b.WriteString(" — ")
				b.WriteString(reason)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\nRequired workers stay — they are the safety floor. Optional workers can be trimmed.\n")

	result := map[string]interface{}{
		"workflow":     workflow,
		"required":     required,
		"optional":     optional,
		"reasons":      reasons,
		"what_it_does": whatItDoes,
		"pruned":       prunedReasons,
	}
	return result, b.String()
}

// casteRosterProduces returns the roster's "produces" prose for a caste, the
// fallback reason when the spawn budget carried no per-caste rationale.
func casteRosterProduces(manifest map[string]interface{}, caste string) string {
	roster, ok := manifest["caste_roster"].([]interface{})
	if !ok {
		return ""
	}
	for _, entry := range roster {
		row := mapValue(entry)
		if strings.TrimSpace(stringValue(row["caste"])) == caste {
			return strings.TrimSpace(stringValue(row["produces"]))
		}
	}
	return ""
}

var ceremonyTeamCheckinCmd = &cobra.Command{
	Use:   "team-checkin",
	Short: "Render the pre-spawn team check-in card from a lifecycle manifest JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, visual, err := renderCeremonyTeamCheckinFromFile(ceremonyFlags.Workflow, ceremonyFlags.ManifestFile)
		if err != nil {
			return err
		}
		outputWorkflow(result, visual)
		return nil
	},
}
