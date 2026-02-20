package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestProfileCommandOutput(t *testing.T) {
	tmp := t.TempDir()
	infile := filepath.Join(tmp, "in.txt")
	if err := os.WriteFile(infile, []byte("# H\nline one. line two.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	var errb bytes.Buffer
	rc := run([]string{"profile", "--seconds", "1", "--max-memory-mb", "64", infile}, &out, &errb)
	if rc != 0 {
		t.Fatalf("profile failed: %s", errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "total time ms:") || !strings.Contains(s, "peak memory estimate bytes:") {
		t.Fatalf("unexpected profile output: %s", s)
	}
}

func TestBenchSuiteOutput(t *testing.T) {
	var out bytes.Buffer
	var errb bytes.Buffer
	rc := run([]string{"bench", "--suite", "default", "--runs", "1", "--warmup", "0", "--aggr", "6"}, &out, &errb)
	if rc != 0 {
		t.Fatalf("bench failed: %s", errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "| file | aggr | run |") || !strings.Contains(s, "| file | aggr | min ms") {
		t.Fatalf("unexpected bench output: %s", s)
	}
}

func TestBenchJSONSchemaVersion(t *testing.T) {
	var out bytes.Buffer
	var errb bytes.Buffer
	rc := run([]string{"bench", "--suite", "default", "--runs", "1", "--warmup", "0", "--aggr", "6", "--json"}, &out, &errb)
	if rc != 0 {
		t.Fatalf("bench json failed: %s", errb.String())
	}
	b := out.Bytes()
	idx := bytes.LastIndex(b, []byte(`{
  "schema_version"`))
	if idx < 0 {
		t.Fatalf("bench json payload not found: %s", out.String())
	}
	var m map[string]any
	if err := json.Unmarshal(b[idx:], &m); err != nil {
		t.Fatalf("invalid bench json: %v", err)
	}
	if m["schema_version"] != "1" {
		t.Fatalf("unexpected schema version: %v", m["schema_version"])
	}
}
