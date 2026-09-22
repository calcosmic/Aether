package cmd

// 207-01-PLAN.md Task 1 (UED-07): the one declared trap list, read by BOTH
// scripts/build-messy-practice-project.sh (via jq) and this file, so the
// builder script and the journey's own Go assertions can never disagree
// about which traps exist. Mirrors cmd/eval_gates.go's manifest-path,
// schema-version and repo-root-override conventions exactly.
//
// Plain functions only -- this file registers no cobra command and adds no
// new subcommand to the binary (milestone rule: no new features, no new
// strict rules).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// journeyTrapsPath is the repo-relative path of the committed, versioned
// trap manifest.
const journeyTrapsPath = "cmd/testdata/journey/traps.json"

// journeyTrapsSchemaVersion is the schema_version constant the trap
// manifest carries.
const journeyTrapsSchemaVersion = "journey-traps/v1"

// journeyTrapsRepoRootOverride lets tests redirect trap-manifest reads to
// an isolated temporary directory instead of the real repository checkout,
// mirroring evalGateRepoRootOverride's precedent exactly.
var journeyTrapsRepoRootOverride string

// journeyTrapsRepoRoot resolves the directory journeyTrapsPath is relative
// to: journeyTrapsRepoRootOverride when a test has set it, otherwise the
// real Aether module root -- the same resolution evalGateRepoRoot uses.
func journeyTrapsRepoRoot() (string, error) {
	if journeyTrapsRepoRootOverride != "" {
		return journeyTrapsRepoRootOverride, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("locate Aether module root from %s", cwd)
	}
	return root, nil
}

// journeyTrapAssertion is the machine-readable descriptor naming the path(s)
// a trap lives at, a Kind identifying which generic checker in
// cmd/messy_practice_project_test.go applies, and the fact that must hold.
// Kind is deliberately its own vocabulary, separate from the trap's own id:
// the id names WHAT the trap is, Kind names HOW to check it, so the test file
// never needs to switch on (and therefore never needs to re-type) any of the
// nine declared trap ids.
type journeyTrapAssertion struct {
	Kind       string `json:"kind"`
	Path       string `json:"path,omitempty"`
	SecondPath string `json:"second_path,omitempty"`
	Must       string `json:"must"`
}

// journeyTrap is one declared, constructible trap in the messy practice
// project. JourneySteps is always a list, even for a trap that exercises
// only one journey step, so a trap serving two steps (nested-project serves
// both pause and survey) needs no separate schema shape.
type journeyTrap struct {
	ID           string               `json:"id"`
	Description  string               `json:"description"`
	JourneySteps []string             `json:"journey_steps"`
	Blocker      string               `json:"blocker"`
	Note         string               `json:"note"`
	Assertion    journeyTrapAssertion `json:"assertion"`
}

// journeyTrapFile is the on-disk container at journeyTrapsPath.
type journeyTrapFile struct {
	SchemaVersion string        `json:"schema_version"`
	Traps         []journeyTrap `json:"traps"`
}

// loadJourneyTraps reads and parses the committed trap manifest.
func loadJourneyTraps() (journeyTrapFile, error) {
	root, err := journeyTrapsRepoRoot()
	if err != nil {
		return journeyTrapFile{}, err
	}
	path := filepath.Join(root, journeyTrapsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return journeyTrapFile{}, fmt.Errorf("read journey traps manifest %s: %w", path, err)
	}
	var file journeyTrapFile
	if err := json.Unmarshal(data, &file); err != nil {
		return journeyTrapFile{}, fmt.Errorf("unmarshal journey traps manifest %s: %w", path, err)
	}
	return file, nil
}

// journeyTrapIDs returns the declared trap ids, in declared file order.
func journeyTrapIDs() ([]string, error) {
	file, err := loadJourneyTraps()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(file.Traps))
	for _, t := range file.Traps {
		ids = append(ids, t.ID)
	}
	return ids, nil
}

// journeyTrapByID returns the declared trap with the given id, and whether
// it was found. The builder script and the Go test both resolve traps this
// way, off the one committed manifest, rather than re-declaring the id list.
func journeyTrapByID(id string) (journeyTrap, bool) {
	file, err := loadJourneyTraps()
	if err != nil {
		return journeyTrap{}, false
	}
	for _, t := range file.Traps {
		if t.ID == id {
			return t, true
		}
	}
	return journeyTrap{}, false
}
