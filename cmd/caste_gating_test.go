package cmd

import (
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestKeywordGatedCastesDoNotSpawnWithoutKeywordMatch is the WP-8 gate, built
// from the real v1.0.47 acceptance failure: an Ambassador was dispatched to a
// phase that implements a local string helper, scored 30 because "api" is a
// substring of "capitalize", and could only report that it had no work.
func TestKeywordGatedCastesDoNotSpawnWithoutKeywordMatch(t *testing.T) {
	localPhase := colony.Phase{
		ID:          1,
		Name:        "Core slugify via TDD",
		Description: "Add a named slugify(s) export alongside capitalize(s) in src/strings.js, covering lowercase, hyphenation and trimming.",
	}
	if score := casteRelevanceScore(localPhase, "ambassador"); score != 0 {
		t.Errorf("ambassador scored %d on a purely local phase; it has no external surface to review", score)
	}
	if score := casteRelevanceScore(localPhase, "gatekeeper"); score != 0 {
		t.Errorf("gatekeeper scored %d on a purely local phase", score)
	}
	// The castes that always have work must be unaffected.
	if score := casteRelevanceScore(localPhase, "builder"); score == 0 {
		t.Error("builder must still score on an implementation phase")
	}

	externalPhase := colony.Phase{
		ID:          2,
		Name:        "OAuth token exchange",
		Description: "Integrate the third-party OAuth provider and store secrets safely.",
	}
	if score := casteRelevanceScore(externalPhase, "gatekeeper"); score == 0 {
		t.Error("gatekeeper must still spawn when the phase names auth/secrets work")
	}
	if score := casteRelevanceScore(externalPhase, "ambassador"); score == 0 {
		t.Error("ambassador must still spawn when the phase names an external integration")
	}
}

// The substring bug itself: keywords must match on word boundaries.
func TestKeywordMatchingRespectsWordBoundaries(t *testing.T) {
	cases := []struct {
		text    string
		keyword string
		want    bool
	}{
		{"add a capitalize helper", "api", false},
		{"call the api directly", "api", true},
		{"wire up the rest-api endpoint", "api", true},
		{"scaling the therapist directory", "api", false},
		{"integration with stripe", "integration", true},
		{"disintegration of the module", "integration", false},
		{"handle an external service call", "external service", true},
		// Inflections of the same concept must still match: anchoring both
		// ends silently dropped the Architect from "structured design" work.
		{"define the structured build manifest", "structure", true},
		{"the apis we call", "api", true},
		{"run the tests", "test", true},
	}
	for _, tc := range cases {
		if got := containsKeyword(tc.text, tc.keyword); got != tc.want {
			t.Errorf("containsKeyword(%q, %q) = %v, want %v", tc.text, tc.keyword, got, tc.want)
		}
	}
}
