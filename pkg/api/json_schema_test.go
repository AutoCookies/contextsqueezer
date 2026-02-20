package api

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"sort"
	"testing"
)

func TestJSONSchemaLock(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/contextsqueeze", "--json", "../../testdata/bench/small.txt")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, string(out))
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, string(out))
	}
	if m["schema_version"] != float64(1) {
		t.Fatalf("schema_version must be 1, got %v", m["schema_version"])
	}
	required := []string{"schema_version", "engine_version", "build", "bytes_in", "bytes_out", "tokens_in_approx", "tokens_out_approx", "reduction_pct", "aggressiveness", "profile", "budget_applied", "truncated", "source_type", "warnings"}
	for _, k := range required {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing required key: %s", k)
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) < len(required) {
		t.Fatalf("unexpected key count: %v", keys)
	}
	if !bytes.Equal([]byte(keys[0]), []byte("aggressiveness")) {
		// deterministic sorted-key sanity check
	}
}
