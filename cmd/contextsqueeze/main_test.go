package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchOutputParsable(t *testing.T) {
	tmp := t.TempDir()
	infile := filepath.Join(tmp, "sample.txt")
	content := "# SECTION\nVisit https://example.com/docs\n" +
		"Repeated noise sentence. Repeated noise sentence. Repeated noise sentence.\n" +
		"Unique token zircon77 appears once and should remain.\n"
	if err := os.WriteFile(infile, []byte(content), 0o644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	var out bytes.Buffer
	var errb bytes.Buffer
	rc := run([]string{"bench", infile}, &out, &errb)
	if rc != 0 {
		t.Fatalf("bench failed rc=%d stderr=%s", rc, errb.String())
	}
	text := out.String()
	if !strings.Contains(text, "| aggr | bytes out | reduction %") {
		t.Fatalf("missing table header: %s", text)
	}
	if !strings.Contains(text, "recommended aggressiveness:") {
		t.Fatalf("missing recommendation line: %s", text)
	}
}
