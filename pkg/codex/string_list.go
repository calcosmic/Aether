package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	invopopjsonschema "github.com/invopop/jsonschema"
)

// stringList is a []string that also accepts a bare string when decoding JSON.
//
// Worker payloads are written by a language model, not by a serializer. Asking
// for a list and receiving a single sentence is routine model behaviour, and
// the `--json-schema` flag passed to the hosted CLIs is advisory rather than
// enforced — the 27 July 2026 M4L scout returned both a string for
// next_worker_instructions and a verification_status outside the declared enum
// in the same response. Decoding into a plain []string turned that into a hard
// unmarshal error, so one loose field in an optional relay sub-object
// discarded an entire 11 KB research result and failed three /ant-plan runs.
//
// Tolerance here is deliberately narrow: shapes that carry the same meaning are
// accepted, anything else still errors, so a genuinely broken payload is not
// laundered into silence.
type stringList []string

// JSONSchema implements invopop/jsonschema's customSchemaImpl hook so the
// generated completion-packet schema (cmd/contract_schema.go) describes
// what UnmarshalJSON actually accepts — null, a bare string, or an array —
// instead of the strict "array of strings" a naive reflection over the
// underlying []string would produce. Without this, the shipped schema would
// reject the exact bare-string shape the doc-shipped worked example (and
// real worker output) legitimately sends, defeating the point of generating
// the schema from the Go types in the first place.
func (stringList) JSONSchema() *invopopjsonschema.Schema {
	return &invopopjsonschema.Schema{
		OneOf: []*invopopjsonschema.Schema{
			{Type: "null"},
			{Type: "string"},
			{Type: "array"},
		},
	}
}

// UnmarshalJSON accepts null, a string, or an array, and normalizes all three
// to a list with blank entries dropped.
func (s *stringList) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*s = nil
		return nil
	}

	var list []string
	if err := json.Unmarshal(trimmed, &list); err == nil {
		*s = compactStrings(list)
		return nil
	}

	var single string
	if err := json.Unmarshal(trimmed, &single); err == nil {
		*s = compactStrings([]string{single})
		return nil
	}

	// A mixed array (["a", 2]) still carries usable content; render each
	// element as text rather than dropping the whole field.
	var raw []json.RawMessage
	if err := json.Unmarshal(trimmed, &raw); err == nil {
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			var str string
			if err := json.Unmarshal(item, &str); err == nil {
				out = append(out, str)
				continue
			}
			out = append(out, strings.TrimSpace(string(item)))
		}
		*s = compactStrings(out)
		return nil
	}

	return fmt.Errorf("expected a string or array of strings, got %s", truncateForError(trimmed))
}

func truncateForError(data []byte) string {
	const limit = 60
	text := strings.TrimSpace(string(data))
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
