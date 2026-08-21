package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestPolicySchemaRequiredFields replaces
// control-ts/tests/schemas/policy.schema.test.ts (deleted in Plan 06 of this
// phase). It deliberately reads the live colony/policies/*.yaml files rather
// than the fixture copies under the (now-deleted) control-ts test tree — a
// strict improvement, since a fixture can silently drift from the file the
// runtime actually loads. During the port, every live file was diff'd
// against its former fixture counterpart: all matched exactly except
// dispatch-contract.yaml, where the live file was (at the time) a strict
// superset (execution_models, deadline_policies, dependency_behaviors,
// fallback_behaviors, fallback_visibility, result_collection_policies added
// on top of the fields this test asserted) — no divergence was found in any
// field this test checked.
//
// dispatch-contract.yaml itself was deleted in the LATER, differently-scoped
// "191-dead-wood" phase (Plan 02, ROADMAP criterion 2): it was one of four
// CWD-relative silent-fallback loaders whose real file only ever resolved
// from a maintainer's own dev-checkout CWD, never from an installed Aether
// run. Its three fields this test checked (max_workers_per_phase,
// spawn_depth_limits, timeout_defaults) were never modeled by
// loadDispatchContractPolicy's dispatchContractPolicy struct in the first
// place (confirmed by a full-file read before deletion) -- deleting the file
// changed no runtime behavior, but did remove the schema surface this test
// validated. Its three checks were removed below rather than left pointing
// at a file that no longer exists.
//
// This test is intentionally narrow: it asserts presence and shallow type of
// the required-field surface below, not a full typed policy loader. A typed
// loader for these files (starting with model-routing.yaml) is Phase 161 /
// MODEL-01's job, not this test's — do not extend this file with typed
// structs or a validation subsystem.
func TestPolicySchemaRequiredFields(t *testing.T) {
	parsed := map[string]map[string]interface{}{}
	for _, c := range policySchemaChecks {
		if _, ok := parsed[c.file]; ok {
			continue
		}
		path := filepath.Join("..", "colony", "policies", c.file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("colony/policies/%s: read failed: %v", c.file, err)
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("colony/policies/%s: parse failed: %v", c.file, err)
		}
		parsed[c.file] = doc
	}

	for _, c := range policySchemaChecks {
		doc := parsed[c.file]
		label := "colony/policies/" + c.file
		val, ok := lookupPolicyDottedPath(doc, c.dottedPath)
		if !ok {
			t.Errorf("%s: %s missing or not a %s", label, c.dottedPath, c.kind)
			continue
		}

		switch c.kind {
		case "string":
			s, ok := val.(string)
			if !ok {
				t.Errorf("%s: %s missing or not a string", label, c.dottedPath)
				continue
			}
			if c.want != nil && s != c.want.(string) {
				t.Errorf("%s: %s = %q, want %q", label, c.dottedPath, s, c.want)
			}
		case "bool":
			b, ok := val.(bool)
			if !ok {
				t.Errorf("%s: %s missing or not a bool", label, c.dottedPath)
				continue
			}
			if c.want != nil && b != c.want.(bool) {
				t.Errorf("%s: %s = %v, want %v", label, c.dottedPath, b, c.want)
			}
		case "numeric":
			if !isPolicyNumeric(val) {
				t.Errorf("%s: %s missing or not numeric", label, c.dottedPath)
			}
		case "map":
			if !isPolicyNonEmptyMap(val) {
				t.Errorf("%s: %s missing or not a non-empty map", label, c.dottedPath)
			}
		case "listContains":
			items, ok := val.([]interface{})
			if !ok {
				t.Errorf("%s: %s missing or not a list", label, c.dottedPath)
				continue
			}
			for _, want := range c.want.([]string) {
				found := false
				for _, item := range items {
					if s, ok := item.(string); ok && s == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: %s missing required entry %q", label, c.dottedPath, want)
				}
			}
		default:
			t.Fatalf("unknown check kind %q for %s: %s", c.kind, label, c.dottedPath)
		}
	}
}

// policyFieldCheck is one row of the required-field table driving
// TestPolicySchemaRequiredFields. Adding a required field later is one row,
// not a new code block.
type policyFieldCheck struct {
	file       string      // filename under colony/policies/, e.g. "safety-gates.yaml"
	dottedPath string      // dotted path into the decoded YAML document
	kind       string      // "string", "bool", "numeric", "map", "listContains"
	want       interface{} // exact expected value (string/bool) or []string of required list entries; nil means "type only"
}

var policySchemaChecks = []policyFieldCheck{
	{"skill-creation.yaml", "skill_creation.allowed", "bool", true},
	{"skill-creation.yaml", "skill_creation.max_skills_per_colony", "numeric", nil},
	{"skill-creation.yaml", "skill_creation.require_wisdom_threshold", "numeric", nil},

	{"safety-gates.yaml", "safety_gates.security_scan", "bool", true},
	{"safety-gates.yaml", "safety_gates.chaos_scan", "bool", nil},
	{"safety-gates.yaml", "safety_gates.auditor_score_threshold", "numeric", nil},

	// dispatch-contract.yaml (max_workers_per_phase, spawn_depth_limits,
	// timeout_defaults) was removed here in Phase 191 Plan 02: the file
	// itself was deleted (ROADMAP criterion 2 -- a CWD-relative
	// silent-fallback loader with zero readers for these three specific
	// fields; loadDispatchContractPolicy's dispatchContractPolicy struct
	// never modeled them, confirmed by full-file read before deletion).
	// There is nothing left under colony/policies/dispatch-contract.yaml
	// for this schema check to validate.

	{"pheromone-lifecycle.yaml", "pheromone_lifecycle.default_ttl_days", "numeric", nil},
	{"pheromone-lifecycle.yaml", "pheromone_lifecycle.max_active_signals", "numeric", nil},
	{"pheromone-lifecycle.yaml", "pheromone_lifecycle.strength_decay_per_day", "numeric", nil},
	{"pheromone-lifecycle.yaml", "pheromone_lifecycle.auto_expire_on_phase_end", "bool", nil},

	{"signal-rules.yaml", "signal_rules.auto_emit_on_phase_complete", "bool", nil},
	{"signal-rules.yaml", "signal_rules.max_feedback_per_phase", "numeric", nil},
	{"signal-rules.yaml", "signal_rules.hard_constraint_prefixes", "listContains", []string{"[error-pattern]", "[redirect]"}},
}

// lookupPolicyDottedPath walks a dotted path (e.g. "safety_gates.security_scan")
// through a decoded YAML document. yaml.v3 decodes mappings into
// map[string]interface{} when the target is interface{}, so every level of
// the walk is a plain string-keyed map.
func lookupPolicyDottedPath(doc map[string]interface{}, dottedPath string) (interface{}, bool) {
	parts := strings.Split(dottedPath, ".")
	var current interface{} = doc
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, ok := m[part]
		if !ok {
			return nil, false
		}
		current = val
	}
	return current, true
}

// isPolicyNumeric reports whether val is a numeric type as yaml.v3 decodes it
// into interface{} (int for integers, float64 for floats).
func isPolicyNumeric(val interface{}) bool {
	switch val.(type) {
	case int, int64, uint64, float32, float64:
		return true
	default:
		return false
	}
}

// isPolicyNonEmptyMap reports whether val is a non-empty map, regardless of
// key type. yaml.v3 decodes a mapping into interface{} as map[string]interface{}
// only when every key is a string; a mapping with non-string keys (for
// example dispatch_contract.spawn_depth_limits, whose YAML keys are the bare
// integers 0-3) decodes to map[interface{}]interface{} instead. Both must be
// accepted since this test only asserts the field is a populated map, not a
// specific key type.
func isPolicyNonEmptyMap(val interface{}) bool {
	switch m := val.(type) {
	case map[string]interface{}:
		return len(m) > 0
	case map[interface{}]interface{}:
		return len(m) > 0
	default:
		return false
	}
}
