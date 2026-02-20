package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONSchemaTextField(t *testing.T) {
	tmp := t.TempDir()
	infile := filepath.Join(tmp, "in.txt")
	if err := os.WriteFile(infile, []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var errb bytes.Buffer
	rc := run([]string{"--json", "--aggr", "0", infile}, &out, &errb)
	if rc != 0 {
		t.Fatalf("run failed: %s", errb.String())
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	required := []string{"bytes_in", "bytes_out", "tokens_in_approx", "tokens_out_approx", "reduction_pct", "aggressiveness", "profile", "budget_applied", "truncated", "source_type", "warnings"}
	for _, k := range required {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing key %s", k)
		}
	}
	if _, ok := m["text"]; !ok {
		t.Fatal("expected text field")
	}
}

func TestJSONSchemaBase64Field(t *testing.T) {
	tmp := t.TempDir()
	infile := filepath.Join(tmp, "in.txt")
	if err := os.WriteFile(infile, []byte{0xff, 0xfe, 0xfd, 0xfa}, 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var errb bytes.Buffer
	rc := run([]string{"--json", "--aggr", "0", infile}, &out, &errb)
	if rc != 0 {
		t.Fatalf("run failed: %s", errb.String())
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := m["text_b64"]; !ok {
		t.Fatal("expected text_b64 field")
	}
}
