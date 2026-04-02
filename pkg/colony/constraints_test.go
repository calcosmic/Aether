package colony

import (
	"encoding/json"
	"testing"
)

func TestConstraintsFileUnmarshalEmpty(t *testing.T) {
	var cf ConstraintsFile
	if err := json.Unmarshal([]byte(`{}`), &cf); err != nil {
		t.Fatalf("unmarshal empty object: %v", err)
	}
}
