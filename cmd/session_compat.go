package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ensureLegacySessionMirror restores the top-level session.json mirror when a
// repo still carries a single legacy colony-scoped session under
// .aether/data/colonies/*/session.json. It is a no-op when the top-level
// mirror already exists, when there is no legacy session, or when multiple
// candidates exist and the active one is ambiguous.
func ensureLegacySessionMirror(s *storage.Store) (bool, error) {
	if s == nil {
		return false, nil
	}

	topLevel := filepath.Join(s.BasePath(), "session.json")
	if _, err := os.Stat(topLevel); err == nil {
		return false, nil
	}

	legacyRoot := filepath.Join(resolveAetherRootPath(), ".aether", "data", "colonies")
	entries, err := os.ReadDir(legacyRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	type legacyCandidate struct {
		path    string
		dirName string
		modTime int64
	}
	var candidates []legacyCandidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(legacyRoot, entry.Name(), "session.json")
		if info, err := os.Stat(candidate); err == nil {
			candidates = append(candidates, legacyCandidate{
				path:    candidate,
				dirName: entry.Name(),
				modTime: info.ModTime().UnixNano(),
			})
		}
	}
	if len(candidates) == 0 {
		return false, nil
	}

	chosen := candidates[0]
	if len(candidates) > 1 {
		var state colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &state); err == nil && state.Goal != nil {
			expected := sanitizeChamberGoal(*state.Goal)
			var matches []legacyCandidate
			for _, candidate := range candidates {
				if candidate.dirName == expected {
					matches = append(matches, candidate)
				}
			}
			if len(matches) == 1 {
				chosen = matches[0]
			} else {
				sort.Slice(candidates, func(i, j int) bool {
					if candidates[i].modTime == candidates[j].modTime {
						return candidates[i].dirName > candidates[j].dirName
					}
					return candidates[i].modTime > candidates[j].modTime
				})
				chosen = candidates[0]
			}
		} else {
			sort.Slice(candidates, func(i, j int) bool {
				if candidates[i].modTime == candidates[j].modTime {
					return candidates[i].dirName > candidates[j].dirName
				}
				return candidates[i].modTime > candidates[j].modTime
			})
			chosen = candidates[0]
		}
	}

	data, err := os.ReadFile(chosen.path)
	if err != nil {
		return false, err
	}

	var session colony.SessionFile
	if err := json.Unmarshal(data, &session); err != nil {
		return false, err
	}
	if err := s.SaveJSON("session.json", session); err != nil {
		return false, err
	}
	return true, nil
}
