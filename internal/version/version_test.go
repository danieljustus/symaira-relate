package version

import (
	"encoding/json"
	"testing"
)

func TestInfo_JSONUsesSnakeCase(t *testing.T) {
	b, err := json.Marshal(Get())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	for _, key := range []string{"tool", "version", "schema_version", "api_version"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing expected snake_case field %q in %s", key, b)
		}
	}
}
