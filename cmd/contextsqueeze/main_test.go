package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBenchOutputParsable(t *testing.T) {
	input := "../testdata/sample.txt"
	if _, err := os.Stat(input); err != nil {
		input = "../../testdata/sample.txt"
	}
	cmd := exec.Command("go", "run", ".", "bench", input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bench failed: %v\n%s", err, string(out))
	}
	s := string(out)
	if !strings.Contains(s, "| aggr |") || !strings.Contains(s, "recommended_aggr=") {
		t.Fatalf("unexpected bench output: %s", s)
	}
}
