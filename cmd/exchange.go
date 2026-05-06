package cmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/exchange"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
}

var exportCmd = &cobra.Command{
	Use:          "export [pheromones|wisdom|registry|archive]",
	Short:        "Export colony data to XML format",
	SilenceUsage: true,
}

var importCmd = &cobra.Command{
	Use:          "import [pheromones|wisdom|registry]",
	Short:        "Import colony data from XML format",
	SilenceUsage: true,
}

func init() {
	exportCmd.AddCommand(exportPheromonesCmd)
	exportCmd.AddCommand(exportWisdomCmd)
	exportCmd.AddCommand(exportRegistryCmd)
	exportCmd.AddCommand(exportArchiveCmd)

	importCmd.AddCommand(importPheromonesCmd)
	importCmd.AddCommand(importWisdomCmd)
	importCmd.AddCommand(importRegistryCmd)
}

var exportPheromonesCmd = &cobra.Command{
	Use:   "pheromones",
	Short: "Export pheromone signals to XML",
	RunE:  runExportPheromones,
}

// runExportPheromones is the shared logic for exporting pheromones to XML.
// Called by both "export pheromones" and the flat alias "pheromone-export-xml".
func runExportPheromones(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	var file colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &file); err != nil {
		result := map[string]interface{}{"xml": nil, "count": 0}
		outputWorkflow(result, renderExportSignalsVisual(result))
		return nil
	}

	data, err := exchange.ExportPheromones(file.Signals)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("export pheromones: %v", err))
		return nil
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			outputErrorMessage(fmt.Sprintf("write file: %v", err))
			return nil
		}
		result := map[string]interface{}{"file": outputFile, "count": len(file.Signals)}
		outputWorkflow(result, renderExportSignalsVisual(result))
	} else {
		fmt.Fprintf(stdout, "%s\n", string(data))
	}
	return nil
}

var exportWisdomCmd = &cobra.Command{
	Use:   "wisdom",
	Short: "Export queen wisdom to XML",
	RunE:  runExportWisdom,
}

// runExportWisdom is the shared logic for exporting wisdom to XML.
func runExportWisdom(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	minConf, _ := cmd.Flags().GetFloat64("min-confidence")
	if minConf == 0 {
		minConf = 0.5
	}

	var instFile colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &instFile); err != nil {
		outputOK(map[string]interface{}{"xml": nil, "count": 0})
		return nil
	}

	var entries []exchange.WisdomEntry
	for _, inst := range instFile.Instincts {
		entries = append(entries, exchange.WisdomEntry{
			ID:         inst.ID,
			Category:   "pattern",
			Confidence: inst.Confidence,
			Domain:     inst.Domain,
			Source:     inst.Provenance.Source,
			CreatedAt:  inst.Provenance.CreatedAt,
			Content:    inst.Trigger,
		})
	}

	colonyName, _ := cmd.Flags().GetString("colony")
	data, err := exchange.ExportWisdom(entries, minConf, colonyName)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("export wisdom: %v", err))
		return nil
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			outputErrorMessage(fmt.Sprintf("write file: %v", err))
			return nil
		}
		outputOK(map[string]interface{}{"file": outputFile, "count": len(entries)})
	} else {
		fmt.Fprintf(stdout, "%s\n", string(data))
	}
	return nil
}

var exportRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Export colony registry to XML",
	RunE:  runExportRegistry,
}

// runExportRegistry is the shared logic for exporting the colony registry to XML.
func runExportRegistry(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	// Load registry from hub
	hub := hubStore()
	if hub == nil {
		return nil
	}

	var regData map[string]interface{}
	if err := hub.LoadJSON("registry/colonies.json", &regData); err != nil {
		outputOK(map[string]interface{}{"xml": nil, "count": 0})
		return nil
	}

	// Convert registry to ColonyEntry format
	var entries []exchange.ColonyEntry
	if coloniesRaw, ok := regData["colonies"]; ok {
		if colonies, ok := coloniesRaw.([]interface{}); ok {
			for _, c := range colonies {
				if cm, ok := c.(map[string]interface{}); ok {
					entry := exchange.ColonyEntry{
						ID:     strVal(cm, "id"),
						Name:   strVal(cm, "name"),
						Status: strVal(cm, "status"),
					}
					if parentID, ok := cm["parent_id"].(string); ok {
						entry.ParentID = parentID
					}
					entries = append(entries, entry)
				}
			}
		}
	}

	data, err := exchange.ExportRegistry(entries)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("export registry: %v", err))
		return nil
	}

	outputFile, _ := cmd.Flags().GetString("output")
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			outputErrorMessage(fmt.Sprintf("write file: %v", err))
			return nil
		}
		outputOK(map[string]interface{}{"file": outputFile, "count": len(entries)})
	} else {
		fmt.Fprintf(stdout, "%s\n", string(data))
	}
	return nil
}

var exportArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Export complete colony archive to XML",
	RunE:  runExportArchive,
}

// runExportArchive is the shared logic for exporting a complete colony archive to XML.
func runExportArchive(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	// Resolve output path.  Prefer positional arg over --output flag.
	// This avoids cobra's sticky-flag problem where flag values persist
	// across Execute() calls on the same Command struct.
	outputFile := ""
	if len(args) > 0 {
		outputFile = args[0]
	}
	if outputFile == "" {
		outputFile, _ = cmd.Flags().GetString("output")
	}
	if outputFile == "" {
		outputErrorMessage("output path is required (use --output flag or positional arg)")
		return nil
	}

	// Load pheromones from colony store.
	var pheromoneXML *exchange.PheromoneXML
	var pheromoneFile colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pheromoneFile); err == nil {
		data, err := exchange.ExportPheromones(pheromoneFile.Signals)
		if err != nil {
			outputErrorMessage(fmt.Sprintf("export pheromones: %v", err))
			return nil
		}
		var parsed exchange.PheromoneXML
		if err := xml.Unmarshal(data, &parsed); err == nil {
			pheromoneXML = &parsed
		}
	} else {
		// Empty pheromones section.
		pheromoneXML = &exchange.PheromoneXML{Version: "1.0", Count: 0}
	}

	// Load instincts from colony store for wisdom section.
	var wisdomXML *exchange.WisdomXML
	var instFile colony.InstinctsFile
	if err := store.LoadJSON("instincts.json", &instFile); err == nil {
		var entries []exchange.WisdomEntry
		for _, inst := range instFile.Instincts {
			entries = append(entries, exchange.WisdomEntry{
				ID:         inst.ID,
				Category:   "pattern",
				Confidence: inst.Confidence,
				Domain:     inst.Domain,
				Source:     inst.Provenance.Source,
				CreatedAt:  inst.Provenance.CreatedAt,
				Content:    inst.Trigger,
			})
		}
		data, err := exchange.ExportWisdom(entries, 0.5, "")
		if err != nil {
			outputErrorMessage(fmt.Sprintf("export wisdom: %v", err))
			return nil
		}
		var parsed exchange.WisdomXML
		if err := xml.Unmarshal(data, &parsed); err == nil {
			wisdomXML = &parsed
		}
	} else {
		wisdomXML = &exchange.WisdomXML{Version: "1.0"}
	}

	// Load registry from hub store.
	var registryXML *exchange.RegistryXML
	hub := hubStore()
	if hub != nil {
		var regData map[string]interface{}
		if err := hub.LoadJSON("registry/colonies.json", &regData); err == nil {
			var entries []exchange.ColonyEntry
			if coloniesRaw, ok := regData["colonies"]; ok {
				if colonies, ok := coloniesRaw.([]interface{}); ok {
					for _, c := range colonies {
						if cm, ok := c.(map[string]interface{}); ok {
							entry := exchange.ColonyEntry{
								ID:     strVal(cm, "id"),
								Name:   strVal(cm, "name"),
								Status: strVal(cm, "status"),
							}
							if parentID, ok := cm["parent_id"].(string); ok {
								entry.ParentID = parentID
							}
							entries = append(entries, entry)
						}
					}
				}
			}
			data, err := exchange.ExportRegistry(entries)
			if err != nil {
				outputErrorMessage(fmt.Sprintf("export registry: %v", err))
				return nil
			}
			var parsed exchange.RegistryXML
			if err := xml.Unmarshal(data, &parsed); err == nil {
				registryXML = &parsed
			}
		}
	}
	if registryXML == nil {
		registryXML = &exchange.RegistryXML{Version: "1.0"}
	}

	// Build combined archive.
	archive := exchange.ColonyArchiveXML{
		Version:    "1.0",
		SealedAt:   time.Now().UTC().Format(time.RFC3339),
		Pheromones: pheromoneXML,
		Wisdom:     wisdomXML,
		Registry:   registryXML,
	}

	data, err := xml.MarshalIndent(archive, "", "  ")
	if err != nil {
		outputErrorMessage(fmt.Sprintf("marshal archive XML: %v", err))
		return nil
	}

	fullData := append([]byte(xml.Header), data...)

	if err := os.WriteFile(outputFile, fullData, 0644); err != nil {
		outputErrorMessage(fmt.Sprintf("write file: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{"file": outputFile})
	return nil
}

var importPheromonesCmd = &cobra.Command{
	Use:   "pheromones <file>",
	Short: "Import pheromone signals from XML",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runImportPheromones,
}

// runImportPheromones is the shared logic for importing pheromones from XML.
func runImportPheromones(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	inputPath := ""
	if len(args) > 0 {
		inputPath = strings.TrimSpace(args[0])
	}
	if inputPath == "" {
		inputPath, _ = cmd.Flags().GetString("input")
	}
	if inputPath == "" {
		inputPath, _ = cmd.Flags().GetString("file")
	}
	if inputPath == "" {
		outputErrorMessage("import pheromones: input file is required")
		return nil
	}

	xmlData, err := os.ReadFile(inputPath)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("read file: %v", err))
		return nil
	}

	result, err := importPheromonesData(inputPath, xmlData, "")
	if err != nil {
		outputErrorMessage(fmt.Sprintf("import pheromones: %v", err))
		return nil
	}
	outputWorkflow(result, renderImportSignalsVisual(result))
	return nil
}

func importPheromonesData(inputPath string, xmlData []byte, sourcePrefix string) (map[string]interface{}, error) {
	if store == nil {
		return nil, fmt.Errorf("no colony initialized")
	}

	signals, err := exchange.ImportPheromones(xmlData)
	if err != nil {
		return nil, err
	}

	sourcePath := filepath.ToSlash(strings.TrimSpace(inputPath))
	baseTrust := colony.ClassifyPromptSource(sourcePath)
	var sanitized []colony.PheromoneSignal
	integrity := make([]colony.PromptIntegrityRecord, 0, len(signals))
	warnings := make([]string, 0)
	blockedRecord := func(sigID, message, evidence string) colony.PromptIntegrityRecord {
		return colony.PromptIntegrityRecord{
			Name:           sigID,
			Title:          "Imported pheromone signal",
			Source:         sourcePath,
			BaseTrustClass: baseTrust,
			TrustClass:     colony.PromptTrustSuspicious,
			Action:         colony.PromptIntegrityActionBlock,
			Blocked:        true,
			Findings: []colony.PromptIntegrityFinding{{
				Kind:     "import_validation",
				Message:  message,
				Evidence: evidence,
			}},
		}
	}

	for _, sig := range signals {
		var contentMap map[string]string
		if err := json.Unmarshal(sig.Content, &contentMap); err != nil {
			record := blockedRecord(sig.ID, "imported signal content is malformed JSON and cannot be trusted", err.Error())
			integrity = append(integrity, record)
			warnings = append(warnings, fmt.Sprintf("blocked suspicious imported signal %s from %s: %s", sig.ID, sourcePath, record.Findings[0].Message))
			continue
		}
		text, ok := contentMap["text"]
		if !ok {
			record := blockedRecord(sig.ID, "imported signal content is missing the text field and cannot be trusted", "missing text field")
			integrity = append(integrity, record)
			warnings = append(warnings, fmt.Sprintf("blocked suspicious imported signal %s from %s: %s", sig.ID, sourcePath, record.Findings[0].Message))
			continue
		}
		assessment := colony.AssessPromptSource(inputPath, text)
		record := assessment.Record(sig.ID, "Imported pheromone signal", sourcePath)
		integrity = append(integrity, record)
		if assessment.Action == colony.PromptIntegrityActionBlock {
			warnings = append(warnings, assessment.Warning(fmt.Sprintf("imported signal %s", sig.ID), sourcePath))
			continue
		}
		cleaned, err := colony.SanitizeSignalContent(text)
		if err != nil {
			record := blockedRecord(sig.ID, err.Error(), text)
			integrity[len(integrity)-1] = record
			warnings = append(warnings, fmt.Sprintf("blocked suspicious imported signal %s from %s: %s", sig.ID, sourcePath, record.Findings[0].Message))
			continue
		}

		newContent, err := json.Marshal(map[string]string{"text": cleaned})
		if err != nil {
			record := blockedRecord(sig.ID, "failed to marshal sanitized imported signal content", err.Error())
			integrity[len(integrity)-1] = record
			warnings = append(warnings, fmt.Sprintf("blocked suspicious imported signal %s from %s: %s", sig.ID, sourcePath, record.Findings[0].Message))
			continue
		}
		if prefix := sanitizeImportedSignalPrefix(sourcePrefix); prefix != "" {
			sig.ID = prefix + ":" + strings.TrimPrefix(sig.ID, prefix+":")
			if strings.TrimSpace(sig.Source) == "" {
				sig.Source = prefix
			}
		}
		sig.Content = json.RawMessage(newContent)
		sanitized = append(sanitized, sig)
	}

	// Merge with existing pheromones
	var file colony.PheromoneFile
	store.LoadJSON("pheromones.json", &file)
	if file.Signals == nil {
		file.Signals = []colony.PheromoneSignal{}
	}
	file.Signals = append(file.Signals, sanitized...)

	if err := store.SaveJSON("pheromones.json", file); err != nil {
		return nil, fmt.Errorf("save pheromones: %w", err)
	}

	emitPromptIntegrityEvents("import.pheromones", integrity)
	result := map[string]interface{}{
		"imported":  len(sanitized),
		"total":     len(file.Signals),
		"source":    inputPath,
		"warnings":  warnings,
		"integrity": integrity,
	}
	if prefix := sanitizeImportedSignalPrefix(sourcePrefix); prefix != "" {
		result["source_colony"] = prefix
		result["id_prefix"] = prefix + ":"
	}
	return result, nil
}

func sanitizeImportedSignalPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-", "\t", "-", "\n", "-")
	prefix = strings.ToLower(replacer.Replace(prefix))
	var b strings.Builder
	lastDash := false
	for _, r := range prefix {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		if r == '-' {
			if lastDash {
				continue
			}
			lastDash = true
		} else {
			lastDash = false
		}
		b.WriteRune(r)
	}
	return strings.Trim(b.String(), "-_")
}

var importWisdomCmd = &cobra.Command{
	Use:   "wisdom <file>",
	Short: "Import queen wisdom from XML",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportWisdom,
}

// runImportWisdom is the shared logic for importing wisdom from XML.
func runImportWisdom(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	xmlData, err := os.ReadFile(args[0])
	if err != nil {
		outputErrorMessage(fmt.Sprintf("read file: %v", err))
		return nil
	}

	entries, err := exchange.ImportWisdom(xmlData)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("import wisdom: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{"imported": len(entries)})
	return nil
}

var importRegistryCmd = &cobra.Command{
	Use:   "registry <file>",
	Short: "Import colony registry from XML",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportRegistry,
}

// runImportRegistry is the shared logic for importing the colony registry from XML.
func runImportRegistry(cmd *cobra.Command, args []string) error {
	if store == nil {
		return fmt.Errorf("no colony initialized")
	}

	xmlData, err := os.ReadFile(args[0])
	if err != nil {
		outputErrorMessage(fmt.Sprintf("read file: %v", err))
		return nil
	}

	entries, err := exchange.ImportRegistry(xmlData)
	if err != nil {
		outputErrorMessage(fmt.Sprintf("import registry: %v", err))
		return nil
	}

	outputOK(map[string]interface{}{"imported": len(entries)})
	return nil
}

func init() {
	exportPheromonesCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	exportWisdomCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	exportWisdomCmd.Flags().Float64("min-confidence", 0.5, "Minimum confidence threshold for wisdom export")
	exportWisdomCmd.Flags().String("colony", "", "Colony ID for wisdom export")
	exportRegistryCmd.Flags().String("output", "", "Write XML to file instead of stdout")
	exportArchiveCmd.Flags().String("output", "", "Output file path (required)")
}

// strVal safely extracts a string from a map.
func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// _ = json.RawMessage import used by exchange
var _ = json.RawMessage{}
