package events

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// TestColonyLiveTopicsAreComplete parses colony_live.go's own source and
// counts the declared LiveTopic* constants, then asserts ColonyLiveTopics()
// returns exactly that many entries -- a typed-in count ("there are 16
// topics") would silently stop failing the moment a topic is added to the
// const block without also being added to the accessor.
func TestColonyLiveTopicsAreComplete(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "colony_live.go", nil, 0)
	if err != nil {
		t.Fatalf("parse colony_live.go: %v", err)
	}

	declared := 0
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if strings.HasPrefix(name.Name, "LiveTopic") {
					declared++
				}
			}
		}
	}

	topics := ColonyLiveTopics()
	if len(topics) != declared {
		t.Fatalf("ColonyLiveTopics() returned %d topics, but colony_live.go declares %d LiveTopic* constants -- every declared topic must be listed in ColonyLiveTopics()", len(topics), declared)
	}

	seen := map[string]bool{}
	for _, topic := range topics {
		if seen[topic] {
			t.Fatalf("ColonyLiveTopics() lists %q more than once", topic)
		}
		seen[topic] = true
		if !strings.HasPrefix(topic, "live.") {
			t.Fatalf("topic %q does not use the live. namespace prefix", topic)
		}
	}
}

// TestColonyLivePayloadRoundTrip proves ColonyLivePayload.RawMessage
// produces JSON that decodes back to an identical value, and that the
// schema version constant round-trips unchanged.
func TestColonyLivePayloadRoundTrip(t *testing.T) {
	payload := ColonyLivePayload{
		SchemaVersion: ColonyLiveSchemaVersion,
		EpisodeID:     "swarm-2026-09-11T120000Z",
		EpisodeKind:   "swarm",
		Sequence:      1,
		Wave:          1,
		WorkerID:      "Scout-12",
		Caste:         "scout",
		WorkerName:    "Scout-12",
		Workspace:     "/tmp/aether-test",
		Findings:      []string{"first finding"},
		Status:        "active",
	}

	raw, err := payload.RawMessage()
	if err != nil {
		t.Fatalf("RawMessage: %v", err)
	}

	var decoded ColonyLivePayload
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(decoded, payload) {
		t.Fatalf("round trip mismatch:\n got:  %+v\n want: %+v", decoded, payload)
	}
	if decoded.SchemaVersion != ColonyLiveSchemaVersion {
		t.Fatalf("schema version = %q, want %q", decoded.SchemaVersion, ColonyLiveSchemaVersion)
	}
}
