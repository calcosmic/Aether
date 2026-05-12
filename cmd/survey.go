package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// surveyFiles lists the expected survey JSON files.
var surveyFiles = []string{
	"blueprint",
	"chambers",
	"disciplines",
	"provisions",
	"pathogens",
}

var requiredSurveyMarkdownFiles = []string{
	"PROVISIONS.md",
	"TRAILS.md",
	"BLUEPRINT.md",
	"CHAMBERS.md",
	"DISCIPLINES.md",
	"SENTINEL-PROTOCOLS.md",
	"PATHOGENS.md",
}

type requiredSurveyArtifact struct {
	Name string
	Kind string
}

func requiredSurveyArtifacts() []requiredSurveyArtifact {
	artifacts := make([]requiredSurveyArtifact, 0, len(requiredSurveyMarkdownFiles)+len(surveyFiles))
	for _, name := range requiredSurveyMarkdownFiles {
		artifacts = append(artifacts, requiredSurveyArtifact{Name: name, Kind: "markdown"})
	}
	for _, name := range surveyFiles {
		artifacts = append(artifacts, requiredSurveyArtifact{Name: name + ".json", Kind: "json"})
	}
	return artifacts
}

var surveyLoadCmd = &cobra.Command{
	Use:   "survey-load",
	Short: "Load survey results from territory survey",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		surveyDir := filepath.Join(store.BasePath(), "survey")

		// Check if survey directory exists
		info, err := os.Stat(surveyDir)
		if err != nil || !info.IsDir() {
			outputOK(map[string]interface{}{
				"loaded": false,
				"files":  map[string]interface{}{},
				"data":   nil,
			})
			return nil
		}

		files := make(map[string]interface{})
		data := make(map[string]interface{})

		for _, name := range surveyFiles {
			filePath := filepath.Join(surveyDir, name+".json")
			content, err := os.ReadFile(filePath)
			if err != nil {
				files[name] = false
				continue
			}

			var parsed interface{}
			if err := json.Unmarshal(content, &parsed); err != nil {
				files[name] = false
				continue
			}

			files[name] = true
			data[name] = parsed
		}

		outputOK(map[string]interface{}{
			"loaded": true,
			"files":  files,
			"data":   data,
		})
		return nil
	},
}

var surveyVerifyCmd = &cobra.Command{
	Use:   "survey-verify",
	Short: "Verify survey data integrity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		surveyDir := filepath.Join(store.BasePath(), "survey")
		issues := []string{}
		allValid := true

		type fileCheck struct {
			Name      string `json:"name"`
			Kind      string `json:"kind"`
			Exists    bool   `json:"exists"`
			ValidJSON bool   `json:"valid_json"`
		}

		var checks []fileCheck

		for _, artifact := range requiredSurveyArtifacts() {
			filePath := filepath.Join(surveyDir, artifact.Name)
			checkName := artifact.Name
			if artifact.Kind == "json" {
				checkName = artifact.Name[:len(artifact.Name)-len(".json")]
			}
			check := fileCheck{Name: checkName, Kind: artifact.Kind}

			content, err := os.ReadFile(filePath)
			if err != nil {
				check.Exists = false
				check.ValidJSON = false
				allValid = false
				issues = append(issues, artifact.Name+": file not found")
			} else {
				check.Exists = true
				switch artifact.Kind {
				case "json":
					if !json.Valid(content) {
						check.ValidJSON = false
						allValid = false
						issues = append(issues, artifact.Name+": invalid JSON")
					} else {
						check.ValidJSON = true
					}
				case "markdown":
					if err := validateSurveyMarkdown(content); err != nil {
						check.ValidJSON = false
						allValid = false
						issues = append(issues, artifact.Name+": invalid markdown ("+err.Error()+")")
					} else {
						check.ValidJSON = true
					}
				}
			}

			checks = append(checks, check)
		}

		// Convert to []interface{} for outputOK compatibility
		checksIface := make([]interface{}, len(checks))
		for i, c := range checks {
			checksIface[i] = map[string]interface{}{
				"name":       c.Name,
				"kind":       c.Kind,
				"exists":     c.Exists,
				"valid_json": c.ValidJSON,
			}
		}

		outputOK(map[string]interface{}{
			"valid":  allValid,
			"files":  checksIface,
			"issues": issues,
		})
		return nil
	},
}

func validateSurveyMarkdown(content []byte) error {
	if len(bytes.TrimSpace(content)) == 0 {
		return fmt.Errorf("empty file")
	}
	if !utf8.Valid(content) || bytes.Contains(content, []byte{0}) {
		return fmt.Errorf("invalid UTF-8 or NUL byte")
	}
	if !strings.Contains(string(content), "#") {
		return fmt.Errorf("missing heading")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(surveyLoadCmd)
	rootCmd.AddCommand(surveyVerifyCmd)
}
