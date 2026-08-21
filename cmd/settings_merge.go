package cmd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// mergeClaudeSettings structurally merges Aether's shipped settings template
// into an existing .claude/settings.json instead of replacing the file.
//
// Ownership rule: Aether owns exactly the hook entries whose inner commands
// all start with "aether ". Those entries are refreshed from the template on
// every sync. Every other hook entry (GSD, user hooks), permission block, and
// top-level key passes through untouched. When the merge produces no semantic
// change, the existing bytes are returned unchanged so a no-op sync leaves the
// file byte-identical.
func mergeClaudeSettings(templateData, existingData []byte) ([]byte, error) {
	if len(strings.TrimSpace(string(existingData))) == 0 {
		return templateData, nil
	}

	var template map[string]interface{}
	if err := json.Unmarshal(templateData, &template); err != nil {
		return nil, fmt.Errorf("shipped settings template is not valid JSON: %w", err)
	}

	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return nil, fmt.Errorf("existing settings.json is not valid JSON, refusing to overwrite: %w", err)
	}

	// Snapshot for the no-op check before mutating.
	var original map[string]interface{}
	if err := json.Unmarshal(existingData, &original); err != nil {
		return nil, err
	}

	merged := mergeSettingsMaps(template, existing)

	if reflect.DeepEqual(merged, original) {
		return existingData, nil
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func mergeSettingsMaps(template, existing map[string]interface{}) map[string]interface{} {
	merged := existing

	existingHooks, _ := merged["hooks"].(map[string]interface{})
	templateHooks, _ := template["hooks"].(map[string]interface{})

	if existingHooks != nil || templateHooks != nil {
		if existingHooks == nil {
			existingHooks = map[string]interface{}{}
		}
		// Strip Aether-owned entries from every event so renamed or retired
		// Aether hooks do not linger, then re-add the template's entries.
		for event, raw := range existingHooks {
			entries, ok := raw.([]interface{})
			if !ok {
				continue
			}
			kept := make([]interface{}, 0, len(entries))
			stripped := false
			for _, entry := range entries {
				if isAetherOwnedHookEntry(entry) {
					stripped = true
					continue
				}
				kept = append(kept, entry)
			}
			if len(kept) == 0 && stripped {
				delete(existingHooks, event)
				continue
			}
			existingHooks[event] = kept
		}
		for event, raw := range templateHooks {
			entries, ok := raw.([]interface{})
			if !ok {
				continue
			}
			current, _ := existingHooks[event].([]interface{})
			existingHooks[event] = append(current, entries...)
		}
		if len(existingHooks) > 0 {
			merged["hooks"] = existingHooks
		} else {
			delete(merged, "hooks")
		}
	}

	// Top-level keys Aether ships beyond hooks are only added when absent;
	// existing user values are never overwritten.
	for key, value := range template {
		if key == "hooks" {
			continue
		}
		if _, ok := merged[key]; !ok {
			merged[key] = value
		}
	}

	return merged
}

// isAetherOwnedHookEntry reports whether a hook entry belongs to Aether: it
// must contain at least one inner hook and every inner hook command must start
// with "aether ". Entries mixing Aether and foreign commands are treated as
// foreign so user customizations are never deleted.
func isAetherOwnedHookEntry(entry interface{}) bool {
	m, ok := entry.(map[string]interface{})
	if !ok {
		return false
	}
	hooks, ok := m["hooks"].([]interface{})
	if !ok || len(hooks) == 0 {
		return false
	}
	for _, h := range hooks {
		hm, ok := h.(map[string]interface{})
		if !ok {
			return false
		}
		command, _ := hm["command"].(string)
		trimmed := strings.TrimSpace(command)
		if !strings.HasPrefix(trimmed, "aether ") && trimmed != "aether" {
			return false
		}
	}
	return true
}
