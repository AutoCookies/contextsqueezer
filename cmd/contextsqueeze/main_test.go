package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchOutputContainsKeyLines(t *testing.T) {
	tmp := t.TempDir()
	inputPath := filepath.Join(tmp, "in.txt")
	content := []byte("# HEADER\nImportant content with https://example.com link.\nThe quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog.\n")
	if err := os.WriteFile(inputPath, content, 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	code := runCLI([]string{"bench", inputPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("bench returned %d stderr=%s", code, errOut.String())
	}
	text := out.String()
	for _, must := range []string{"| aggr |", "| 0 |", "| 9 |", "Recommended aggressiveness:"} {
		if !strings.Contains(text, must) {
			t.Fatalf("missing %q in output:\n%s", must, text)
		}
	}
}
